package task

import "github.com/gofiber/fiber/v3"

// RegisterRoutes는 /api/v1 하위에 이 모듈의 라우트를 마운트한다.
// guard가 전달되면 모든 task 라우트(읽기 포함)에 인증이 요구된다(router.go의 AUTH_ENABLED 분기).
//
// 새 모듈 추가 패턴 (README 참고):
//  1. internal/modules/<name> 패키지를 task와 동일 구조로 생성
//  2. internal/router/wiring.go에서 NewService(NewRepository(db))로 조립
//  3. 서비스와 선택적 guard를 받는 RegisterRoutes 함수 제공
//  4. internal/router/router.go에서 RegisterRoutes로 라우트 마운트
func RegisterRoutes(v1 fiber.Router, svc *Service, guard ...fiber.Handler) {
	h := NewHandler(svc)
	// fiber v3 Group은 가변 인자가 any라 변환이 필요하다.
	mw := make([]any, len(guard))
	for i, g := range guard {
		mw[i] = g
	}
	tasks := v1.Group("/tasks", mw...)

	tasks.Post("/", h.Create)
	tasks.Get("/", h.List)
	tasks.Get("/:id", h.Get)
	tasks.Patch("/:id", h.Update)
	tasks.Delete("/:id", h.Delete)
}
