// Package validator는 go-playground/validator를 Fiber v3의 StructValidator로 연결한다.
// fiber.Config{StructValidator: validator.NewFiberStructValidator()}처럼 등록하면
// Bind().Body() 계열에서 validate 태그가 자동 실행된다.
package validator

import (
	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v3"
)

type fiberStructValidator struct {
	v *validator.Validate
}

// NewFiberStructValidator는 Fiber용 StructValidator 구현체를 반환한다.
func NewFiberStructValidator() fiber.StructValidator {
	return &fiberStructValidator{v: validator.New(validator.WithRequiredStructEnabled())}
}

func (fv *fiberStructValidator) Validate(out any) error {
	return fv.v.Struct(out)
}
