# 새 모듈 추가 체크리스트

트리거: `internal/modules/` 아래에 새 도메인을 추가할 때.
`task` 모듈이 유일한 공식 템플릿이다. 이 체크리스트를 순서대로 수행한다.

## 1. 패키지 생성 (task 복제)

- [ ] `internal/modules/<name>/` 에 model.go / dto.go / repository.go / service.go / handler.go / routes.go 생성
- [ ] Repository **인터페이스는 service.go(소비자)에 선언**, GORM 구현은 repository.go
- [ ] DTO ↔ 모델 매핑은 명시적 함수로 (toResponse 등). 모델 직접 노출 금지
- [ ] service에는 fiber.Ctx·*gorm.DB를 노출하지 않음(HTTP 미들웨어와 DB 조립 코드는 각 타입 사용 가능)

## 2. 검증/에러 연결

- [ ] 요청 DTO에 `validate:"..."` 태그 (StructValidator가 자동 실행)
- [ ] 도메인 에러는 `<name>/errors.go`에 apperror.AppError 센티널 정의(범용만 전역 카탈로그) 후 service에서 반환
- [ ] Bind 실패 매핑은 `validator.BindErrorToAppError`를 그대로 사용(모듈별 재정의 금지)

## 3. 마이그레이션 이원화 (놓치기 쉬움 — 둘 다!)

- [ ] sqlite(dev): `cmd/api/main.go`의 `AutoMigrateIfNeeded(...)` 목록에 모델 추가
- [ ] postgres/mysql(prod): `make migrate-new name=<desc>`로 SQL 파일 생성 후 양쪽 드라이버 작성(up/down)
- [ ] 양쪽 스키마를 테스트·API 호출·실제 DB 스모크로 확인(`readyz`는 DB ping만 검사하므로 테이블 누락을 검출하지 않음)

## 4. 조립

- [ ] `internal/router/wiring.go`: `<name>.NewService(<name>.NewRepository(db))`
- [ ] `internal/router/router.go`: `<name>.RegisterRoutes(v1, svc)`
- [ ] 인증 보호가 필요하면 guard 전달 방법은 task/routes.go와 router.go의 AUTH_ENABLED 분기 참고(로그인 자체는 인증 가드 대상이 아님)

## 5. 테스트 (커밋 전 필수)

- [ ] service 단위 테스트: fake repository 주입, 외부 I/O 없음
- [ ] handler 통합 테스트: 임시 디렉터리 sqlite + 실제 GORM + app.Test()
  - 성공/검증실패(422)/미존재(404) 시나리오 최소 3개
  - 응답 디코딩은 빈 본문(204 등) 건너뛰는 `testutil.Do` 헬퍼 사용
- [ ] `CONTRIBUTING.md`의 포맷·lint·vet·race 테스트·CGO 없는 빌드 검증 통과

## 6. 문서

- [ ] README의 API 예제 섹션에 curl 추가
- [ ] `api/openapi.yaml`에 라우트·요청·응답 계약 추가, `docs/file-inventory.md`에 새 파일 역할 등록
