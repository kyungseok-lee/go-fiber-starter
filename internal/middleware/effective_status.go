package middleware

import (
	"errors"

	"github.com/gofiber/fiber/v3"

	"go-fiber-starter/internal/apperror"
)

// EffectiveStatus는 c.Next() 언와인드 시점(err 반환 직후, ErrorHandler 실행 전)의
// 실제 응답 상태를 산정한다. 규칙은 router.go의 errorHandler와 동일해야 하며,
// 어긋나면 관측값과 실제 응답이 달라지므로 양쪽 수정은 함께 한다.
func EffectiveStatus(c fiber.Ctx, err error) int {
	if err == nil {
		return c.Response().StatusCode()
	}
	var fe *fiber.Error
	if errors.As(err, &fe) {
		return fe.Code
	}
	status := apperror.AsAppError(err).Status
	// errorHandler는 >=500을 고정 500으로 응답한다 — 관측값도 동일하게 클램프해야 3자 일치.
	if status >= fiber.StatusInternalServerError {
		return fiber.StatusInternalServerError
	}
	return status
}
