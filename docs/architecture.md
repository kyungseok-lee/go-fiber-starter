# 아키텍처

이 문서는 요청 생명주기, 계층 경계, 스키마 정책의 "왜"를 한 장으로 설명한다.
상세 규칙은 [AGENTS.md](../AGENTS.md)와 `.agents/rules/`를 참고.

## 요청 생명주기

```
클라이언트
  │
  ▼
[requestid]      상관관계 ID 발급(클라이언트 X-Request-ID는 sanitize 후 수용)
[recover]        패닉을 에러로 변환해 ErrorHandler로 전달
[helmet]         보안 헤더
[cors]           설정 기반 오리진 허용(prod에서 '*' 거부)
[limiter 전역]   IP당 분당 예산 — /livez /readyz /metrics /openapi.yaml은 skip
[prometheus]     method×route패턴×status 카운터/히스토그램(EffectiveStatus 기준)
[requestlogger]  실제 상태 기반 구조화 액세스 로그(request_id 포함)
  │
  ▼
router (/api/v1 그룹)
  ├─ auth: POST /auth/login (로그인 전용 엄격 limiter)
  └─ task: CRUD (AUTH_ENABLED=true면 RequireAuth 가드)
       │ fiber.Ctx는 여기서 끝난다
       ▼
   handler   Bind().Body() → StructValidator(validate 태그) → DTO
       │        BindErrorToAppError: 파싱=400 / 검증 위반=422+details
       ▼
   service   업무 단위·변경 규칙. Repository 인터페이스 선언(소비자 정의).
       │        도메인 에러 정규화, 원인은 %w로 보존
       ▼
 repository  도메인의 *gorm.DB 접근·트랜잭션 구현. WithContext(ctx) 필수.
             조회·삭제 미존재는 task.ErrTaskNotFound 반환
```

### 관찰 정합성 (핵심 설계)

Fiber v3는 미들웨어 언와인드 이후에 ErrorHandler를 실행하므로, 언와인드 시점의
`c.Response().StatusCode()`는 핸들러가 에러를 반환해도 아직 200일 수 있다.
그래서 두 관측 미들웨어 모두 `EffectiveStatus(c, err)`로 **실제 응답 상태**를 산정한다:

- `*fiber.Error` → fe.Code
- `apperror.AppError` → Status(단, ≥500은 errorHandler가 고정 500으로 응답하므로 동일하게 클램프)
- 그 외 → 500

규칙이 router.errorHandler와 어긋나면 지표·로그·응답 3자가 불일치한다 — 양쪽은 함께 수정한다.

### 429 흐름

두 limiter의 `LimitReached`는 `ErrRateLimited`(AppError)를 반환한다. ErrorHandler는
429 엔벨로프(`RATE_LIMITED`)로 응답하고 warn 로그를 남긴다.
전역 limiter는 수집 미들웨어보다 먼저 실행되므로 여기서 거부된 요청은 Prometheus 카운터와
RequestLogger 액세스 로그에 포함되지 않는다. 로그인 limiter까지 도달해 거부된 요청은 두 수집 미들웨어가
`EffectiveStatus`로 429를 기록한다.

## 계층 경계 (위반 금지)

| 계층 | 가질 수 있는 것 | 절대 금지 |
|---|---|---|
| handler | fiber.Ctx, DTO 조립 | 비즈니스 로직, gorm |
| service | Repository 인터페이스 선언, 업무 단위·변경 규칙 | fiber.Ctx, *gorm.DB |
| repository | *gorm.DB, SQL, 트랜잭션 구현 | HTTP 개념 |
| model | GORM 태그 | JSON 직렬화 책임(DTO가 담당) |

task 조회·삭제의 미존재는 repository가 `ErrTaskNotFound`로 반환한다. `Update` 내부의
`gorm.ErrRecordNotFound`는 범용 `apperror.ErrNotFound`로 바꾼 뒤 service가 `ErrTaskNotFound`로 정규화한다.
기타 DB 오류는 원인을 보존해 상위로 전달하고, ErrorHandler가 알려진 업무 오류와 내부 오류를 HTTP 응답으로 변환한다.

task 수정은 service가 변경 함수를 넘기고 repository가 `WithContext(ctx).Transaction(...)`으로
조회·변경·저장을 묶는다. PostgreSQL/MySQL의 `FOR UPDATE`는 동시 수정을 직렬화하며 sqlite에는 적용하지 않는다.
`database.WithTx`는 현재 운영 호출이 없는 확장 헬퍼다. 다중 저장소 작업에 사용할 때도
repository 구현에 두고 `WithContext(ctx)`를 적용한 DB를 전달한다.

## 응답 형식

`/api/v1` 업무 API의 JSON 성공·오류 응답은 `httpx.Envelope`를 사용한다. 204 삭제 응답은 빈 본문이다.
운영 경로는 각각 health 전용 JSON, Prometheus 텍스트, OpenAPI YAML을 사용한다.

## 스키마 정책 (이원화)

| 드라이버 | 방식 | 이유 |
|---|---|---|
| sqlite(dev/test) | GORM AutoMigrate | 사이클 속도. `cmd/api/main.go` 모델 목록에 등록 |
| postgres/mysql(prod) | 버전 SQL(`db/migrations/<driver>/`) + `cmd/migrate` | 컬럼 삭제/락 제어. AutoMigrate는 prod에서 하드 차단 |

양쪽을 모두 갱신하지 않으면 환경에 따라 스키마가 누락될 수 있다. `readyz`는 DB ping만 수행하므로
스키마 유무를 보장하지 않는다. 마이그레이션·스키마 테스트와 실제 API·DB 스모크에서 필요한 테이블을 확인한다.

목록의 `limit` 상한 100은 가져오는 행 수와 응답 크기를 제한한다. 페이지 메타를 위한 전체 `Count`는
별도 쿼리이므로 이 상한으로 비용이 줄어들지 않는다. 대량 테이블의 집계·offset 비용은 별도 설계 대상이다.

## 인증

HS256 JWT(만료 필수, 알고리즘 강제). 자격증명 검증은 `Authenticator` 인터페이스 뒤에
숨겨져 있어 env 데모 구현 ↔ DB 구현 교체가 자유롭다. 교체 스케치는
[docs/authenticator-sketch.md](authenticator-sketch.md).

## 의도적 보류 (과잉설계 방지)

compress/ETag, OTel, cursor pagination, 분산 rate limiter — 트리거 조건은 README Roadmap 참고.
