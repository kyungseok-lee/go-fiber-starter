# 개발 프롬프트: go-fiber-starter v2 (Production-Grade)

> AI 에이전트에게 이 파일 전체를 전달하면 프로젝트가 세팅되도록 작성된 실행 사양(spec)이다.
> v2 변경: graceful shutdown, 트랜잭션 전략, 버전 기반 마이그레이션, 관찰 가능성(livez/readyz/metrics),
> 보안 미들웨어, Fiber v3 API 가이드(v2 혼입 방지), 설정 fail-fast 검증, 에러 카탈로그, 테스트 전략 추가.
>
> **참고**: 프로젝트 생성 이후의 일상 운영 규칙은 루트 `AGENTS.md`가 단일 출처이다.
> 이 파일은 초기 설계 사양(이력 문서)으로 유지하되, 실행·테스트·계층 경계 지침은 현재 구현에 맞춘다.
> 본문과 AGENTS.md가 다르면 **AGENTS.md가 우선**이다. JWT 인증 스캐폴드와 OpenAPI는 이미 구현되었다.
> 현재 파일별 역할과 연결은 [docs/file-inventory.md](docs/file-inventory.md)를 참고한다.

---

너는 시니어 Go 백엔드 아키텍트이다. **REST API 보일러플레이트(go-fiber-starter)** 를 프로덕션 준비 수준으로 세팅하라.
클론 후 `make run` 한 번으로 즉시 실행되고, 신규 모듈을 정해진 패턴으로 복사해 확장할 수 있어야 한다
(create-next-app의 DX를 Go REST API에 적용한다).

## 1. 아키텍처 원칙

1. **12-Factor**: 설정은 환경변수에서만 주입. 로그는 stdout JSON 스트림.
2. **레이어드 아키텍처 + 의존성 방향 고정**: `handler → service → repository → model`.
   상위 계층만 하위를 참조. service에 `fiber.Ctx`·`*gorm.DB`를 노출하지 않는다.
   HTTP 미들웨어·헬퍼와 DB 인프라·조립 코드는 각 프레임워크 타입을 사용한다.
3. **인터페이스 소비자 정의**: service가 필요한 repository 인터페이스를 service 패키지에 선언 → mock 용이.
4. **Fail-fast**: 잘못된 설정/DB 연결 실패는 시작 시 즉시 종료(exit 1).
5. **Graceful shutdown**: SIGTERM/SIGINT 수신 시 새 요청 차단 → 진행 중 요청 완료 대기 → DB 커넥션 종료.
6. **관찰 가능성 기본 장착**: 구조화 로그(slog JSON) + request_id 상관관계, liveness/readiness 분리, Prometheus 메트릭.

## 2. 기술 스택

| 항목 | 선택 | 비고 |
|---|---|---|
| Go | 현재 `go.mod`의 `go` 지시문 이상 | `go version` 확인. 로컬·CI·Docker 버전 기준은 `go.mod` |
| HTTP | `github.com/gofiber/fiber/v3` | go.mod에 고정한 버전 |
| ORM | `gorm.io/gorm` | GORM v2 API, go.mod에 고정한 모듈 버전 |
| DB 드라이버 | `gorm.io/driver/postgres`, `gorm.io/driver/mysql`, `github.com/glebarez/sqlite`(순수 Go) | env로 선택 |
| 마이그레이션 | `github.com/golang-migrate/migrate/v4` + embed.FS | prod 경로 |
| 설정 | `github.com/caarlos0/env/v11` + `github.com/joho/godotenv` | |
| 검증 | `go-playground/validator/v10` (Fiber Bind와 연동) | |
| 로깅 | 표준 `log/slog` (JSON) | GORM logger 어댑팅 |
| 메트릭 | `github.com/prometheus/client_golang` | `/metrics` |
| 테스트 | 표준 `testing`, Fiber `app.Test()`, `internal/testutil.Do` | |
| 핫 리로드 | `air` | |

**금지**: gin/echo 등 타 프레임워크, viper(과다), 전역 싱글톤 DB, `init()`에서의 부수효과.

## 3. 디렉터리 구조

```
go-fiber-starter/
├── cmd/
│   ├── api/main.go              # 진입점: 조립 + graceful shutdown
│   └── migrate/main.go          # 마이그레이션 CLI (up/down/version/force)
├── internal/
│   ├── config/config.go         # env 파싱 + 검증(fail-fast) + config_test
│   ├── database/
│   │   ├── database.go          # 연결/풀/드라이버 분기
│   │   ├── logger.go            # GORM logger → slog 어댑터
│   │   └── migrator.go          # AutoMigrate 정책(sqlite만 허용, prod 차단)
│   ├── middleware/
│   │   ├── requestlogger.go     # method/path/status/latency/request_id 구조화 로그
│   │   ├── prometheus.go        # 요청 카운터/히스토그램 수집
│   │   └── clock.go             # 시계 주입(테스트 대체용)
│   ├── apperror/errors.go        # AppError 타입 + 범용 에러 코드 카탈로그
│   ├── httpx/response.go         # 성공/실패 엔벨로프 헬퍼
│   ├── pagination/pagination.go  # 페이지네이션 파라미터/meta(MaxLimit 클램프)
│   ├── testutil/testutil.go      # 빈 응답을 처리하는 HTTP 테스트 헬퍼
│   ├── validator/validator.go   # go-playground ↔ Fiber StructValidator 연결
│   ├── router/
│   │   ├── router.go            # fiber.App 생성, 미들웨어 체인, ErrorHandler
│   │   ├── validator.go         # fiber.Config에 StructValidator 등록
│   │   └── wiring.go            # 모듈별 service/repository 조립
│   └── modules/
│       ├── auth/                # JWT 로그인·Authenticator·인증 가드
│       ├── task/                # 예제 도메인 = "새 모듈 추가 템플릿"
│       │   ├── model.go  dto.go  errors.go  repository.go  service.go  handler.go  routes.go
│       │   ├── service_test.go      # fake repo 단위 테스트
│       │   └── handler_test.go      # sqlite 파일 + app.Test 통합 테스트
│       └── health/handler.go    # livez/readyz
├── db/migrations/
│   ├── embed.go                 # SQL 임베드(embed.FS)
│   └── {postgres,mysql}/000001_create_tasks.{up,down}.sql
├── .env.example  .air.toml  .gitignore
├── api/                        # 임베드되는 OpenAPI 스펙
├── docs/                       # 아키텍처·확장 스케치·전체 파일 목록
├── scripts/smoke.sh             # PostgreSQL/MySQL 실제 DB 스모크
├── Dockerfile  docker-compose.yml
├── Makefile  README.md  AGENTS.md  CLAUDE.md  PROMPT.md
└── .github/workflows/          # CI 검증·DB 마이그레이션·스모크 및 태그 릴리스
```

## 4. Fiber v3 API 가이드 — ⚠️ v2 문법 혼입 금지

| 작업 | ❌ v2 | ✅ v3 |
|---|---|---|
| 핸들러 시그니처 | `func(c *fiber.Ctx) error` | `func(c fiber.Ctx) error` |
| 바디 파싱+검증 | `c.BodyParser(&dto)` | `c.Bind().Body(&dto)` (`fiber.Config.StructValidator` 등록 필요) |
| 쿼리 파싱 | `c.QueryInt("page")` | `c.Query("page", "1")` 후 strconv 또는 제네릭 getter |
| 서버 시작 | `app.Listen(":8080")` | `app.Listen(":8080", fiber.ListenConfig{ GracefulContext: ctx })` |
| 전역 에러 처리 | `fiber.Config{ErrorHandler}` 동일하나 시그니처 `func(fiber.Ctx, error) error` | |
| 미들웨어 import | `/v2/middleware/...` 또는 contrib | `github.com/gofiber/fiber/v3/middleware/{recover,requestid,cors,helmet,limiter}` |
| 핸들러 테스트 | `app.Test(req)` 동일 | |

## 5. 기능 요구사항

### 5.1 설정 (`internal/config`) — fail-fast
- `.env` 로드(선택) → 실제 환경변수 우선 → 구조체 바인딩(`caarlos0/env`)
- 핵심 설정: `APP_ENV`(기본 local), `APP_PORT`(8080), `DB_DRIVER`(sqlite), `DB_DSN`(data/app.db).
  기본값은 `internal/config/config.go`, 환경별 예시는 `.env.example`을 따른다.
- 선택(+기본값): `DB_MAX_OPEN_CONNS=25`, `DB_MAX_IDLE_CONNS=5`, `DB_CONN_MAX_LIFETIME=30m`,
  `CORS_ALLOWED_ORIGINS=*`, `RATE_LIMIT_PER_MINUTE=120`, `LOG_LEVEL=info`
- **검증 규칙**: 값 범위/port 숫자 여부/driver 지원 여부를 `Validate()`에서 검사. 위반 시 모든 오류를 모아 한 번에 출력하고 exit 1.
- prod + `DB_DRIVER=sqlite` 조합은 거부(dev 전용 명시)

### 5.2 데이터베이스 & 트랜잭션
- 드라이버 분기 → `*gorm.DB`. `Ping()` 확인, 커넥션 풀 적용, `SetMaxOpenConns` 등.
- GORM logger는 slog 어댑터를 사용한다. 일반 SQL은 debug, 오류·느린 SQL은 error/warn으로 기록하며
  SQL에 파라미터 값이 포함될 수 있다. 실제 출력은 설정한 로그 레벨에 따른다.
- **트랜잭션 원칙**: service가 업무 단위와 변경 규칙을 정하고, DB 트랜잭션은 repository 안에 둔다.
  현재 task `Update`는 service가 전달한 변경 함수를 `r.db.WithContext(ctx).Transaction(...)` 안에서 실행한다.
  PostgreSQL/MySQL은 `FOR UPDATE` 행 잠금을 사용하고 sqlite에서는 생략한다.
  다중 저장소 작업을 추가할 때도 service에는 업무 인터페이스만 노출한다. `database.WithTx`는 현재 운영 호출이 없는
  확장 헬퍼이며, 사용할 경우 repository 안에서 `r.db.WithContext(ctx)`를 전달한다.
- 요청 context는 handler→service→repository까지 전파한다. 현재 모든 DB 쓰기에 별도 5초 타임아웃을 추가하는 구현은 없다.

### 5.3 마이그레이션 전략 (이원화 — 중요)
- **dev/test(sqlite)**: `AutoMigrate` 허용 (기본 on). 빠른 개발 사이클용.
- **prod(postgres/mysql)**: **버전 기반 SQL 마이그레이션(golang-migrate)만 사용**. `db/migrations/<driver>/NNNNNN_name.{up,down}.sql`,
  `embed.FS`로 바이너리 임베드. `cmd/migrate` CLI: `up|down|version|force`.
- AutoMigrate는 prod(postgres/mysql)에서 하드 차단(`migrate up` 사용 안내).
  local/dev의 PostgreSQL/MySQL에서는 경고 후 허용되지만 실제 데이터는 버전 SQL로 관리한다.
- 초기 마이그레이션 000001: tasks 테이블 생성(down 포함).

### 5.4 라우팅 & 보안 미들웨어 (적용 순서 중요)
```
requestid → recover → helmet(보안헤더) → cors(설정 기반) → rate limiter(분당 N회)
→ prometheus 수집기 → requestlogger(slog: method path status latency_ms request_id)
```
- 라우트: `GET /livez`(프로세스 생존) / `GET /readyz`(DB ping만 검사, 스키마 검사 없음) /
  `GET /metrics`(Prometheus) / `GET /openapi.yaml` / `/api/v1/tasks` CRUD.
  `AUTH_ENABLED=true`이면 `POST /api/v1/auth/login`과 task Bearer 인증을 활성화한다.
  정의되지 않은 경로는 통일 에러 포맷 404를 반환한다.
- 전역 limiter가 먼저 거부한 429는 ErrorHandler만 기록한다. 수집 미들웨어 뒤의 로그인 limiter가 반환한 429는
  메트릭·액세스 로그에도 반영된다. 자세한 흐름은 `docs/architecture.md`를 따른다.
- fiber.Config: `BodyLimit: 1 MiB`, ReadTimeout/WriteTimeout/IdleTimeout은 fiber.Config로 설정(기본 15s/15s/60s).

### 5.5 응답/에러 포맷 + 에러 카탈로그

`/api/v1` 업무 API의 JSON 성공·오류 응답은 아래 엔벨로프를 사용한다. 204는 빈 본문이며,
health 프로브는 전용 JSON, 메트릭은 텍스트, OpenAPI는 YAML 형식이다.

```json
{"success":true,"data":[],"meta":{"page":1,"limit":20,"total_count":0,"total_pages":0}}
```

```json
{"success":false,"error":{"code":"TASK_NOT_FOUND","message":"task not found"}}
```
- `apperror.AppError{Code, Status, Message, Details}` + 생성자(`NewBadRequest`/`NewValidation`).
  도메인 전용 센티널(예: TASK_NOT_FOUND)은 해당 모듈 `errors.go`에 선언한다.
- **에러 코드 카탈로그** 상수화: `INVALID_REQUEST(400 파싱/422 검증)`, `NOT_FOUND(404)`, `CONFLICT(409)`,
  `INTERNAL_ERROR(500)`, `TASK_NOT_FOUND(404)` …
- service/repository는 필요 시 `fmt.Errorf("...: %w", err)`로 원인을 보존한다. task 미존재는 `ErrTaskNotFound`를 사용하며,
  repository `Update`가 반환한 범용 `apperror.ErrNotFound`는 service에서 도메인 에러로 변환한다. 전역 `ErrorHandler`에서
  AppError → 엔벨로프 변환, 알 수 없는 에러는 500 INTERNAL_ERROR(내부 메시지 노출 금지, 로그에는 상세 기록).

### 5.6 예제 도메인 Task (모듈 템플릿)
- 모델: `ID uint PK`, `Title string(필수 1~200)`, `Done bool`, `DueDate *time.Time`, 타임스탬프, soft delete.
- DTO ↔ model 매핑 함수 명시적 작성. 요청 검증 태그(`validate:"required,min=1,max=200"`).
- API:
  - `POST /api/v1/tasks` → 201 + data
  - `GET /api/v1/tasks?page=&limit=` → data[] + meta(page,limit,total_count,total_pages) / limit max 100 클램프
  - `GET /api/v1/tasks/:id` → 200 or 404 TASK_NOT_FOUND
  - `PATCH /api/v1/tasks/:id` 부분 수정(present-field만 반영)
  - `DELETE /api/v1/tasks/:id` → 204 (soft delete)
- `due_date` 부분 수정은 `NullableTime`으로 미전달·null 해제·값 변경을 구분한다.
- **repository 인터페이스는 service.go에 선언**, 공통 GORM 구현체는 repository.go에서 제공.

### 5.7 테스트 전략
- **단위**: service 테스트는 fake repository 주입(외부 I/O 없음, ms 단위).
- **통합**: handler 테스트는 테스트별 임시 디렉터리의 sqlite 파일 + 실제 GORM + `app.Test()`를 사용한다.
  `:memory:`는 연결 풀마다 DB가 달라질 수 있어 사용하지 않는다. 응답은 204 빈 본문을 처리하는 `testutil.Do`로 읽는다.
- 커버리지: CI는 `-coverpkg=./...`로 전체 패키지를 계량하고 65% 미만이면 실패한다. `make test-cov`로 같은 기준의 리포트를 생성한다.
- 표준 `testing`으로 테이블 테스트와 실패 검사를 작성한다. testify의 `assert/require`는 사용하지 않는다.
- PostgreSQL/MySQL 마이그레이션·CRUD는 별도 CI 잡에서 실제 DB로 검증한다. OpenAPI 드리프트 테스트는 경로·메서드 일치를 검사한다.

### 5.8 운영/DX 도구
- **Makefile**: `help run dev tools smoke test test-cov lint vet fmt build tidy docker-up docker-down migrate-up migrate-down migrate-new name=` — `make`만 치면 help.
- **air**: `.air.toml` (`root = "."`, tmp/api 빌드, cmd/internal/db/api 아래의 지정 확장자 감시).
- **Dockerfile**: 멀티스테이지(golang 빌드 → alpine), non-root, HEALTHCHECK(/livez), CGO_ENABLED=0.
- **docker-compose.yml**: `postgres`(기본, healthcheck 포함), `mysql`(profile mysql),
  `migrate`와 `app`(profile full). PostgreSQL 준비 → migrate 성공 → app 시작 순서다.
- **CI (.github/workflows/ci.yml)**: checkout → setup-go(go.mod, cache) → `gofmt -l` → golangci-lint → `go vet`
  → race 테스트(전체 패키지 커버리지) → 65% 게이트 → CGO 없는 빌드. PostgreSQL/MySQL 마이그레이션·스모크는 별도 잡이다.
- **README.md**(한국어): 3분 Quickstart(sqlite 무설치 강조), 환경변수 표, API curl 예제 전체,
  **"새 모듈 추가 가이드"(복사→수정 5단계)**, 마이그레이션 워크플로, 실제 트랜잭션 흐름, 폴더 구조를 설명한다.
  JWT 스캐폴드·OpenAPI는 구현 완료이며, DB 인증·Swagger UI·tracing은 확장 과제로 구분한다.

### 5.9 코드 컨벤션
- 패키지 소문자 단수, 파일 snake_case. 주석은 "왜"만. magic number 금지(상수화).
- 에러는 `%w` 래핑, 센티널 에러 비교는 `errors.Is`.
- 컨텍스트: handler에서 `c.Context()` → service → repository 전파. 컨텍스트 버림 금지.

## 6. 완료 조건 (Definition of Done)
1. `make run` → `curl :8080/livez`=200, `readyz`=200(DB ping), `/metrics` 응답.
2. README curl 예제(Task CRUD 전체)는 생성 응답의 ID를 사용한다. 인증을 켰으면 Bearer 헤더를 추가하며,
   201/200/404/422/204 시나리오를 확인한다.
3. 존재하지 않는 경로/검증 실패/미지 ID가 **통일된 에러 엔벨로프**로 반환.
4. `go build ./... && go vet ./... && go test ./... -race` 전부 깨끗.
5. `make lint` 통과(설치 안내 포함).
6. SIGTERM 전송 시 "shut down gracefully", "database connections closed" 로그와 정상 종료를 확인한다.
7. `docker compose up -d --wait postgres` 후 README의 DSN·`DB_AUTO_MIGRATE=false` 설정과
   `make migrate-up`을 적용하고 동일 동작을 확인한다.
8. 신규 모듈 추가 가이드대로 따라했을 때 task 모듈과 동일한 구조가 나옴.

## 7. 진행 순서
1. 스켈레톤(go.mod, 디렉터리) → 2. config → 3. apperror/httpx/pagination(순수 패키지) →
4. database(+logger) → 5. middleware → 6. task 모듈(model→repo→service→handler→routes) →
7. health 모듈 → 8. router(ErrorHandler 포함) → 9. main(graceful shutdown) →
10. 마이그레이션(cmd/migrate + SQL) → 11. 테스트 → 12. 도구/CI/문서 → 13. DoD 검증.
각 단계 끝나면 `go build ./...`로 컴파일 유지할 것.
