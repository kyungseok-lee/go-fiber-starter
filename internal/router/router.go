package router

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/adaptor"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"github.com/gofiber/fiber/v3/middleware/helmet"
	"github.com/gofiber/fiber/v3/middleware/limiter"
	fibermw "github.com/gofiber/fiber/v3/middleware/recover"
	"github.com/gofiber/fiber/v3/middleware/requestid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"gorm.io/gorm"

	"go-fiber-starter/api"
	"go-fiber-starter/internal/apperror"
	"go-fiber-starter/internal/config"
	"go-fiber-starter/internal/httpx"
	appmw "go-fiber-starter/internal/middleware"
	"go-fiber-starter/internal/modules/auth"
	"go-fiber-starter/internal/modules/health"
	"go-fiber-starter/internal/modules/task"
)

const bodyLimit = 1 << 20 // 1 MiB

// limiterSkipPaths는 전역 rate limit 예산을 소모하지 않는 운영 경로다.
// k8s 프로브와 스크래퍼가 임계값을 소진해 실제 API 트래픽을 429로 밀어내는 것을 방지한다.
var limiterSkipPaths = map[string]struct{}{
	"/livez":        {},
	"/readyz":       {},
	"/metrics":      {},
	"/openapi.yaml": {},
}

// skipRateLimit은 정확히 일치하는 경로만 전역 limiter에서 제외한다.
// c.Path()는 쿼리 문자열이 제거된 경로라 쿼리 우회 변형이 없다.
func skipRateLimit(c fiber.Ctx) bool {
	_, ok := limiterSkipPaths[c.Path()]
	return ok
}

// ErrRateLimited는 429 응답의 단일 출처이다. limiter LimitReached와 mapFiberError가
// 이를 공유해 어느 경로로 거부되든 동일한 엔벨로프(error.code=RATE_LIMITED)를 보장한다.
var ErrRateLimited = &apperror.AppError{Code: "RATE_LIMITED", Status: http.StatusTooManyRequests, Message: "too many requests"}

// New는 fiber 앱을 생성하고 미들웨어 체인/라우트/에러 처리를 조립한다.
func New(cfg *config.Config, db *gorm.DB, log *slog.Logger, reg *prometheus.Registry) *fiber.App {
	app := fiber.New(fiber.Config{
		AppName:         cfg.AppName,
		BodyLimit:       bodyLimit,
		ReadTimeout:     cfg.ReadTimeout,
		WriteTimeout:    cfg.WriteTimeout,
		IdleTimeout:     cfg.IdleTimeout,
		StructValidator: newStructValidator(),
		ErrorHandler:    errorHandler(log),
		// 역프록시 뒤에서 실제 클라이언트 IP를 신뢰할 수 있게 한다(allowlist 필수는 config.Validate가 보장).
		TrustProxy:       cfg.TrustProxy,
		TrustProxyConfig: fiber.TrustProxyConfig{Proxies: cfg.TrustProxyProxies},
		ProxyHeader:      cfg.TrustProxyHeader,
	})

	// 순서가 계약이다: requestid(상관관계) → recover → 보안헤더 → cors → rate limit
	// → 메트릭 수집 → 요청 로그 (로그는 최후순위라 latency 측정이 가장 정확)
	app.Use(requestid.New())
	app.Use(fibermw.New())
	app.Use(helmet.New())
	app.Use(cors.New(cors.Config{AllowOrigins: cfg.CORSAllowedOrigins}))
	app.Use(limiter.New(limiter.Config{
		Max:               cfg.RateLimitPerMinute,
		Expiration:        time.Minute,
		LimiterMiddleware: limiter.SlidingWindow{},
		LimitReached:      func(c fiber.Ctx) error { return ErrRateLimited },
		// 프로브/메트릭/스펙 경로는 예산 밖. 로그인 전용 guard에는 적용하지 않는다(별도 예산).
		Next: skipRateLimit,
	}))
	prom := appmw.NewPrometheus(reg)
	app.Use(prom.Handler())
	app.Use(appmw.RequestLogger(log))

	h := health.NewHandler(func() error { return pingDB(db) }, cfg.BuildCommit)
	app.Get("/livez", h.Livez)
	app.Get("/readyz", h.Readyz)
	app.Get("/metrics", adaptor.HTTPHandler(promhttp.HandlerFor(reg, promhttp.HandlerOpts{})))
	app.Get("/openapi.yaml", openapiSpecHandler)

	v1 := app.Group("/api/v1")

	var taskGuard []fiber.Handler
	if cfg.AuthEnabled {
		authSvc := auth.NewService(auth.NewDemoAuthenticator(cfg), cfg.AuthJWTSecret, cfg.AuthTokenTTL)
		// 로그인은 task 인증 가드 밖에 두고 전역 rate limit에 IP별 로그인 예산을 추가한다.
		loginGuard := limiter.New(limiter.Config{
			Max:               cfg.AuthRateLimitPerMinute,
			Expiration:        time.Minute,
			LimiterMiddleware: limiter.SlidingWindow{},
			LimitReached:      func(c fiber.Ctx) error { return ErrRateLimited },
		})
		auth.RegisterRoutes(v1, authSvc, loginGuard) // 로그인은 task 가드 밖
		taskGuard = []fiber.Handler{auth.RequireAuth(authSvc)}
		log.Info("auth enabled",
			"protected", "/api/v1/tasks",
			"login_rate_limit_per_minute", cfg.AuthRateLimitPerMinute)
	}
	task.RegisterRoutes(v1, taskService(db), taskGuard...)

	return app
}

// openapiSpecHandler는 embed된 api/openapi.yaml을 그대로 응답한다.
// 정적 파일 서빙 설정 없이 단일 파일 응답이면 충분하다.
// 참고: fiber v3의 c.Type(ext, charset...)은 두 번째 인자가 charset이라 MIME 지정 용도가 아니어서
// Content-Type을 직접 설정한다(GetMIME에 yaml이 없어 시스템 의존 결과를 피함).
func openapiSpecHandler(c fiber.Ctx) error {
	yaml, err := api.FS.ReadFile("openapi.yaml")
	if err != nil {
		return fmt.Errorf("read embedded openapi.yaml: %w", err)
	}
	c.Set(fiber.HeaderContentType, "application/yaml; charset=utf-8")
	return c.Send(yaml)
}

// errorHandler는 모든 에러를 통일 엔벨로프로 변환한다.
// 미들웨어 바깥에서 실행되므로 요청 로그 미기록 에러도 여기서 남긴다. 500은 내부 상세를 노출하지 않는다.
func errorHandler(log *slog.Logger) fiber.ErrorHandler {
	return func(c fiber.Ctx, err error) error {
		var fe *fiber.Error
		if errors.As(err, &fe) {
			appErr := mapFiberError(fe)
			return c.Status(fe.Code).JSON(httpx.ErrorBody(appErr))
		}

		appErr := apperror.AsAppError(err)
		fields := []any{
			"error", err.Error(),
			"method", c.Method(),
			"path", c.Path(),
			"request_id", requestid.FromContext(c),
		}
		if appErr.Status >= http.StatusInternalServerError {
			log.Error("request failed", fields...)
			// 클라이언트에는 고정 메시지만 노출한다.
			safe := *apperror.ErrInternal
			return c.Status(safe.Status).JSON(httpx.ErrorBody(&safe))
		}
		log.Warn("request rejected", fields...)
		return c.Status(appErr.Status).JSON(httpx.ErrorBody(appErr))
	}
}

func mapFiberError(fe *fiber.Error) *apperror.AppError {
	switch fe.Code {
	case fiber.StatusNotFound:
		return apperror.ErrNotFound.WithCause(fe)
	case fiber.StatusTooManyRequests:
		return ErrRateLimited.WithCause(fe)
	case fiber.StatusRequestEntityTooLarge:
		return &apperror.AppError{Code: "PAYLOAD_TOO_LARGE", Status: fe.Code, Message: "request body too large"}
	default:
		return &apperror.AppError{Code: fmt.Sprintf("HTTP_%d", fe.Code), Status: fe.Code, Message: fe.Message}
	}
}
