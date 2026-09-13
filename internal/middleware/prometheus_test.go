package middleware

import (
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/prometheus/client_golang/prometheus"

	"go-fiber-starter/internal/apperror"
	"go-fiber-starter/internal/httpx"
)

// TestPrometheus_HandlerError_CountedWithRealStatus는 핸들러가 에러를 반환해도
// 실제 응답 상태(422)로 카운트됨을 검증한다. ErrorHandler 실행 전 관측하므로
// 실제 응답이 어떻게 매핑되든 라벨은 AppError 상태를 따른다.
func TestPrometheus_HandlerError_CountedWithRealStatus(t *testing.T) {
	reg := prometheus.NewRegistry()
	p := NewPrometheus(reg)

	app := fiber.New(fiber.Config{
		ErrorHandler: func(c fiber.Ctx, err error) error {
			appErr := apperror.AsAppError(err)
			return c.Status(appErr.Status).JSON(httpx.ErrorBody(appErr))
		},
	})
	app.Use(p.Handler())
	app.Get("/forced", func(c fiber.Ctx) error {
		return apperror.NewValidation(nil)
	})

	req := httptest.NewRequest("GET", "/forced", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 422 {
		t.Fatalf("want response 422, got %d", resp.StatusCode)
	}

	if got := counterValue(t, reg, "422"); got != 1 {
		t.Fatalf("want http_requests_total{status=\"422\"}=1, got %v", got)
	}
}

// counterValue는 수집된 http_requests_total에서 status 라벨이 일치하는 계열 값을 찾는다.
func counterValue(t *testing.T, reg *prometheus.Registry, status string) float64 {
	t.Helper()
	fams, err := reg.Gather()
	if err != nil {
		t.Fatal(err)
	}
	for _, mf := range fams {
		if mf.GetName() != "http_requests_total" {
			continue
		}
		for _, m := range mf.GetMetric() {
			for _, l := range m.GetLabel() {
				if l.GetName() == "status" && l.GetValue() == status {
					return m.GetCounter().GetValue()
				}
			}
		}
	}
	return 0
}
