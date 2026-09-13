# 전체 파일 역할과 정리 기준

2026-09-14에 소스 본문, 호출·import 관계, Go 테스트 자동 발견, 임베드 입력, 도구 설정과 문서 링크를 대조했다.
이 목록은 최종 변경을 포함한 프로젝트 파일 **103개**(Git 추적 대상 **102개**, 로컬 SQLite DB **1개**)를 경로별로 기록한다.
Git 자체가 관리하는 `.git/` 내부 객체·인덱스·참조·로그는 버전 관리 메타데이터로 보존하며 아래 프로젝트 파일 수에는 포함하지 않는다.

## 판단 결과

- 프로젝트와 무관해 통째로 삭제할 소스·설정·문서 파일은 없었다. Go가 직접 import하지 않는 테스트·퍼즈 입력·설계 문서·도구 진입점도 용도를 확인했다.
- Makefile의 사용하지 않는 변수와 릴리스 대상에 없는 Windows 압축 설정을 제거했다. 전체 Docker 스택 명령, 스모크 임시파일 정리, AutoMigrate 검증 및 로그 수준 처리를 보완했다.
- `.dockerignore`로 Git 메타데이터·로컬 DB·환경파일·빌드 산출물을 컨테이너 빌드에서 제외한다. Go 소스·모듈 파일·SQL·OpenAPI는 빌드 입력으로 유지한다.
- `data/app.db`는 `internal/config/config.go`의 기본 경로이며 `Task` 모델과 일치하는 데이터가 있는 개발 DB다. 프로젝트 관련 로컬 데이터이므로 보존하고 Git에 올리지 않는다.
- `PROMPT.md`와 DB 인증 스케치는 설계·확장 참고 문서다. 실제 운영 규칙은 `AGENTS.md`, 현재 동작은 소스와 테스트가 기준이다. `database.WithTx`도 현재 호출부가 없는 문서화된 확장 헬퍼다.

## 파일별 근거

`유지·갱신`은 파일을 유지하면서 이번 점검에서 내용 또는 동작을 바로잡은 경우다. `추가`는 이번 점검 결과를 반영한 새 파일이다.

### 실행 소스와 임베드 입력 (38개)

| 파일 | 역할과 사용 근거 | 처리 |
|---|---|---|
| [api/embed.go](../api/embed.go) | OpenAPI YAML을 API 바이너리에 포함하는 embed 진입점 — //go:embed openapi.yaml → internal/router/router.go의 openapiSpecHandler가 api.FS.ReadFile("openapi.yaml")로 소비 | 유지·갱신 |
| [api/openapi.yaml](../api/openapi.yaml) | 실제 REST 경로와 요청·응답 계약을 문서화하는 OpenAPI 스펙 — api/embed.go에 포함되어 GET /openapi.yaml로 제공되고 TestOpenAPI_NoRouteDrift가 실제 router/auth/task 라우트와 비교 | 유지·갱신 |
| [cmd/api/main.go](../cmd/api/main.go) | API 서버 실행·설정 로드·DB 준비·종료·메트릭 조립 진입점 — go run ./cmd/api / Makefile run 및 Dockerfile 빌드 대상; config.Load → database.New/AutoMigrateIfNeeded(Task) → router.New → app.Listen | 유지·갱신 |
| [cmd/migrate/main.go](../cmd/migrate/main.go) | PostgreSQL/MySQL SQL 마이그레이션 CLI 진입점 — Makefile migrate-* / Dockerfile migrate 빌드 대상; migrations.DriverFS → iofs.New → migrate.NewWithSourceInstance; up/down/version/force 실행 | 유지 |
| [db/migrations/embed.go](../db/migrations/embed.go) | 양 DB 드라이버 마이그레이션을 CLI에 내장하고 선택하는 진입점 — //go:embed postgres/*.sql mysql/*.sql; cmd/migrate/main.go run이 DriverFS(string(cfg.DBDriver)) 호출 | 유지 |
| [db/migrations/mysql/000001_create_tasks.down.sql](../db/migrations/mysql/000001_create_tasks.down.sql) | MySQL task 최초 스키마 롤백 — db/migrations/embed.go mysql/*.sql에 포함; cmd/migrate down의 m.Steps(-1)가 task 인덱스와 테이블을 제거 | 유지 |
| [db/migrations/mysql/000001_create_tasks.up.sql](../db/migrations/mysql/000001_create_tasks.up.sql) | MySQL task 테이블과 소프트 삭제·done 인덱스 생성 — db/migrations/embed.go mysql/*.sql에 포함; cmd/migrate up이 적용; internal/modules/task/model.go의 tasks 모델 컬럼과 대응 | 유지 |
| [db/migrations/postgres/000001_create_tasks.down.sql](../db/migrations/postgres/000001_create_tasks.down.sql) | PostgreSQL task 최초 스키마 롤백 — db/migrations/embed.go postgres/*.sql에 포함; cmd/migrate down의 m.Steps(-1)가 task 인덱스와 테이블을 제거 | 유지 |
| [db/migrations/postgres/000001_create_tasks.up.sql](../db/migrations/postgres/000001_create_tasks.up.sql) | PostgreSQL task 테이블과 소프트 삭제·done 인덱스 생성 — db/migrations/embed.go postgres/*.sql에 포함; cmd/migrate up이 적용; internal/modules/task/model.go의 tasks 모델 컬럼과 대응 | 유지 |
| [internal/apperror/errors.go](../internal/apperror/errors.go) | 공통 오류 코드·센티널·래핑 및 전송 오류 분류 — validator.BindErrorToAppError, auth.Service/RequireAuth, task.Repository/Service, router.errorHandler/mapFiberError, middleware.EffectiveStatus에서 참조 | 유지 |
| [internal/config/config.go](../internal/config/config.go) | 환경변수 기본값·검증·운영 정책의 단일 출처 — cmd/api.run 및 cmd/migrate.run이 Load 호출; Config는 database/router/auth 조립의 설정 입력 | 유지 |
| [internal/database/database.go](../internal/database/database.go) | 세 DB 연결·풀 설정·readiness ping 및 트랜잭션 확장 헬퍼 — cmd/api.run → New; router/wiring.go pingDB → Ping; WithTx는 README/PROMPT에 문서화된 확장 헬퍼이며 현재 런타임·테스트 직접 호출은 없음 | 유지·갱신 |
| [internal/database/logger.go](../internal/database/logger.go) | GORM logger.Interface를 slog로 연결하는 어댑터 — cmd/api.run의 database.NewLogger → database.New의 gorm.Config.Logger; GORM이 LogMode/Info/Warn/Error/Trace를 인터페이스로 호출 | 유지 |
| [internal/database/migrator.go](../internal/database/migrator.go) | 환경·드라이버별 AutoMigrate 허용 정책 — cmd/api.run → AutoMigrateIfNeeded(ctx,cfg,db,&task.Task{}); sqlite 개발 스키마를 만들고 prod 비-sqlite 자동 변경은 차단 | 유지 |
| [internal/httpx/response.go](../internal/httpx/response.go) | 성공·오류 API Envelope 생성과 HTTP 응답 헬퍼 — task.Handler의 OK/Created/OKWithMeta/NoContent; auth.Handler.Login의 OK; router.errorHandler의 ErrorBody | 유지 |
| [internal/middleware/clock.go](../internal/middleware/clock.go) | 요청 관측 미들웨어의 공통 시간 함수 — prometheus.go Handler와 requestlogger.go RequestLogger가 timeNow/timeSince를 직접 호출 | 유지 |
| [internal/middleware/effective_status.go](../internal/middleware/effective_status.go) | 에러 핸들러 실행 전 응답 상태를 계산하는 관측 헬퍼 — prometheus.go와 requestlogger.go가 EffectiveStatus(c, err)를 호출; effective_status_test.go는 로컬 오류 핸들러 모델과 비교 | 유지·갱신 |
| [internal/middleware/prometheus.go](../internal/middleware/prometheus.go) | 라우트 패턴별 요청 수·지연 메트릭 미들웨어 — router.New → appmw.NewPrometheus(reg) → app.Use(prom.Handler()); 동일 reg를 /metrics의 promhttp.HandlerFor가 노출 | 유지 |
| [internal/middleware/requestlogger.go](../internal/middleware/requestlogger.go) | 요청 ID·상태·지연·IP 구조화 요청 로그 — router.New → app.Use(appmw.RequestLogger(log)); requestid middleware가 앞서 등록 | 유지 |
| [internal/modules/auth/handler.go](../internal/modules/auth/handler.go) | 로그인 DTO 바인딩·검증·JWT 응답과 Identity Locals 키 — auth/routes.go → NewHandler(svc).Login; Login → Service.Issue(c.Context(),...); middleware.go가 IdentityKey 사용 | 유지 |
| [internal/modules/auth/middleware.go](../internal/modules/auth/middleware.go) | Bearer JWT 검증 및 인증 주체 전달 미들웨어 — router.New의 AuthEnabled 분기 → auth.RequireAuth(authSvc) → task.RegisterRoutes의 guard; Service.Verify와 IdentityKey를 사용 | 유지 |
| [internal/modules/auth/routes.go](../internal/modules/auth/routes.go) | 선택 인증 로그인 라우트 등록 — router.New의 AuthEnabled 분기 → auth.RegisterRoutes(v1,authSvc,loginGuard); POST /auth/login에 guard와 Handler.Login 배치 | 유지 |
| [internal/modules/auth/service.go](../internal/modules/auth/service.go) | 인증 저장소 추상화·데모 자격증명·HS256 JWT 발급 및 검증 — router.New가 NewDemoAuthenticator/NewService로 조립; Handler.Login → Issue → Authenticator.Verify; RequireAuth → Service.Verify | 유지 |
| [internal/modules/health/handler.go](../internal/modules/health/handler.go) | liveness와 DB readiness 운영 프로브 핸들러 — router.New → health.NewHandler(pingDB,cfg.BuildCommit); /livez → Livez, /readyz → Readyz | 유지 |
| [internal/modules/task/dto.go](../internal/modules/task/dto.go) | Task 생성·부분 수정·응답 DTO와 날짜 null/미전달 구분 — Handler.Create/Update가 요청을 바인딩하고 Service가 toResponse/toResponses 및 req.DueDate 상태를 사용; encoding/json이 NullableTime.UnmarshalJSON을 인터페이스로 호출 | 유지 |
| [internal/modules/task/errors.go](../internal/modules/task/errors.go) | Task 미존재 도메인 오류 — repository.FindByID/Delete 및 Service.Update/Delete에서 ErrTaskNotFound 사용; router.errorHandler가 AppError 404 TASK_NOT_FOUND로 전송 | 유지 |
| [internal/modules/task/handler.go](../internal/modules/task/handler.go) | Task CRUD HTTP 바인딩·검증·상태 응답 — task/routes.go가 Create/List/Get/Update/Delete 등록; 각 메서드가 c.Context()를 Service까지 전달 | 유지 |
| [internal/modules/task/model.go](../internal/modules/task/model.go) | Task GORM 영속 모델과 소프트 삭제 스키마 — cmd/api.run의 AutoMigrateIfNeeded 모델 입력; task/repository.go CRUD의 Task 사용; GORM이 TableName()을 인터페이스로 호출 | 유지 |
| [internal/modules/task/repository.go](../internal/modules/task/repository.go) | Task 저장소의 GORM CRUD·부분 갱신 트랜잭션 — router/wiring.go → task.NewRepository → task.NewService; Service의 Repository 인터페이스 메서드 구현; pg/mysql Update는 applyRowLock FOR UPDATE 사용 | 유지 |
| [internal/modules/task/routes.go](../internal/modules/task/routes.go) | Task CRUD 라우트와 선택 인증 가드 등록 — router.New → task.RegisterRoutes(v1,taskService(db),taskGuard...); /tasks 그룹의 5개 HTTP 작업을 Handler에 연결 | 유지·갱신 |
| [internal/modules/task/service.go](../internal/modules/task/service.go) | 소비자 Repository 인터페이스와 Task 업무 흐름 — router/wiring.go에서 NewService(NewRepository(db))로 생성; Handler의 CRUD 호출을 repository로 전달하고 요청 필드만 변경 | 유지 |
| [internal/pagination/pagination.go](../internal/pagination/pagination.go) | 페이지 파라미터 정규화·최대 크기·응답 메타 계산 — task.Handler.List가 DefaultPage/DefaultLimit 사용; Service.List → NewPageQuery/NewPageMeta; Repository.List → q.Limit/q.Offset() | 유지·갱신 |
| [internal/router/router.go](../internal/router/router.go) | Fiber 설정·미들웨어·운영/인증/Task 라우트·전역 오류 조립 — cmd/api.run → router.New; 내부 wiring/validator 및 모든 모듈 RegisterRoutes를 소비; embed FS를 /openapi.yaml로 제공 | 유지·갱신 |
| [internal/router/validator.go](../internal/router/validator.go) | Fiber 앱에 공통 StructValidator를 주입하는 조립 헬퍼 — router.New의 fiber.Config.StructValidator: newStructValidator() → validator.NewFiberStructValidator() | 유지 |
| [internal/router/wiring.go](../internal/router/wiring.go) | DB 기반 Task service와 readiness 의존성 명시 조립 — router.New에서 taskService(db), pingDB(db) 호출; task.NewService(task.NewRepository(db)) 및 database.Ping(db)에 연결 | 유지 |
| [internal/testutil/testutil.go](../internal/testutil/testutil.go) | 통합 테스트 공용 HTTP 요청·Envelope 디코딩 헬퍼 — auth/handler_test.go, task/handler_test.go, router/router_test.go의 testutil.Do 호출; 배포 진입점은 이 패키지를 import하지 않음 | 유지 |
| [internal/validator/bind.go](../internal/validator/bind.go) | 공통 바인딩 오류를 400 또는 필드별 422로 변환 — auth.Handler.Login와 task.Handler.Create/Update가 validator.BindErrorToAppError 호출; apperror.NewValidation/NewBadRequest 반환 | 유지 |
| [internal/validator/validator.go](../internal/validator/validator.go) | go-playground validator를 Fiber StructValidator 인터페이스로 연결 — router/validator.go 및 auth/task handler 통합 테스트가 NewFiberStructValidator 호출; Fiber Bind().Body가 Validate(out)를 인터페이스로 호출 | 유지·갱신 |

### 테스트와 회귀 입력 (27개)

| 파일 | 역할과 사용 근거 | 처리 |
|---|---|---|
| [cmd/api/main_test.go](../cmd/api/main_test.go) | 로그 수준 대소문자 처리 회귀 테스트 — cmd/api/main.go의 newLogger와 slog Logger.Enabled 동작을 검증 | 추가 |
| [cmd/migrate/main_test.go](../cmd/migrate/main_test.go) | 마이그레이션 DSN 변환 단위·퍼즈 회귀 테스트 — TestMigrateURL_PostgresURLForm/MysqlTCPForm/KeyValueDSN_Rejected → migrateURL; FuzzStripScheme → stripScheme | 유지 |
| [cmd/migrate/testdata/fuzz/FuzzStripScheme/e8ac48dfc452f36a](../cmd/migrate/testdata/fuzz/FuzzStripScheme/e8ac48dfc452f36a) | 마이그레이션 스킴 파서 퍼즈 회귀 입력 — go test fuzz v1 string("postgres://"); cmd/migrate/main_test.go의 활성 FuzzStripScheme target이 접두어 단독 입력 유지 계약 검증 | 유지 |
| [db/migrations/embed_test.go](../db/migrations/embed_test.go) | 임베드된 드라이버별 SQL 경로 회귀 테스트 — TestDriverFS_ListsSQLPairs/UnknownDriver_Errors → DriverFS; postgres/mysql *.up.sql과 대응 *.down.sql 검증 | 유지 |
| [internal/apperror/errors_test.go](../internal/apperror/errors_test.go) | 공통 에러 카탈로그·래핑 계약 단위 테스트 — TestSentinelCatalog_StatusesAndCodes → ErrInvalidRequest 등; TestAsAppError_* → AsAppError; TestIs_MatchesAcrossCauseCopies → WithCause/Is; TestNewValidation_DetailsPreserved → NewValidation | 유지 |
| [internal/config/config_test.go](../internal/config/config_test.go) | 환경설정 로딩·검증 규칙 회귀 테스트 — TestValidate_CollectsAllProblems/ProdSQLiteRejected/TrustProxyRules/AuthRules 등 → Config.Validate; TestLoad_EnvParsingError_FailsFast → Load | 유지 |
| [internal/database/database_test.go](../internal/database/database_test.go) | SQLite DB 생성 디렉터리 권한 회귀 테스트 — TestSQLite_CreatesPrivateDirectory → database.New; t.TempDir 하위 디렉터리 0700 확인, Windows 명시적 skip | 유지 |
| [internal/database/logger_test.go](../internal/database/logger_test.go) | GORM slog 어댑터 로그 레벨·쿼리 분기 테스트 — TestTrace_* → Logger.Trace; TestLogMode_ReturnsIndependentCopy → Logger.LogMode; TestInfoWarnError_Levels → Info/Warn/Error | 유지 |
| [internal/database/migrator_test.go](../internal/database/migrator_test.go) | 환경·드라이버별 AutoMigrate 정책 회귀 테스트 — TestAutoMigrateIfNeeded_ProdPostgres_HardBlocked/Disabled_Noop/SQLiteDev_CreatesTables → AutoMigrateIfNeeded | 유지·갱신 |
| [internal/httpx/helpers_test.go](../internal/httpx/helpers_test.go) | 응답 헬퍼 단위 테스트의 HTTP 요청·본문 유틸리티 — newGet/newPost/newDelete/bodyBytes는 같은 패키지 response_test.go의 TestOK_EnvelopeShape/TestCreated_Status201/TestNoContent_EmptyBody/TestErrorBody_MirrorsAppErrorContract에서 호출 | 유지 |
| [internal/httpx/response_test.go](../internal/httpx/response_test.go) | 공통 HTTP 응답 상태·엔벨로프 계약 테스트 — TestOK_EnvelopeShape → OK; TestCreated_Status201 → Created; TestNoContent_EmptyBody → NoContent; TestErrorBody_MirrorsAppErrorContract → ErrorBody | 유지 |
| [internal/middleware/effective_status_test.go](../internal/middleware/effective_status_test.go) | 응답 확정 이전 에러 상태 관찰 회귀 테스트 — TestEffectiveStatus_* → EffectiveStatus; jsonDecodeLogLine은 requestlogger_test.go 두 테스트에서 호출; localErrorHandler/statusVia는 해당 테스트 보조 함수 | 유지 |
| [internal/middleware/prometheus_test.go](../internal/middleware/prometheus_test.go) | Prometheus 에러 응답 상태 라벨 회귀 테스트 — TestPrometheus_HandlerError_CountedWithRealStatus → NewPrometheus/Handler; counterValue는 http_requests_total status=422 카운터 확인 | 유지·갱신 |
| [internal/middleware/requestlogger_test.go](../internal/middleware/requestlogger_test.go) | 요청 로그 request_id 유무 회귀 테스트 — TestRequestLogger_IncludesRequestID/NoRequestIDMiddleware_OmitsField → RequestLogger; requestid 미들웨어 포함 여부별 출력 확인 | 유지 |
| [internal/modules/auth/handler_test.go](../internal/modules/auth/handler_test.go) | 로그인·인증 가드 HTTP 통합 테스트 — TestLogin_* → RegisterRoutes/Handler.Login/Service.Verify; TestRequireAuth_* → RequireAuth; testutil.Do 사용 | 유지 |
| [internal/modules/auth/service_test.go](../internal/modules/auth/service_test.go) | 데모 인증·JWT 발급 검증 단위 테스트 — TestIssueAndVerify_Roundtrip → Service.Issue/Verify; TestIssue_WrongPassword_InvalidCredentials 및 TestVerify_ExpiredToken/TamperedToken_Unauthorized; newTestService는 handler_test에서도 사용 | 유지 |
| [internal/modules/health/handler_test.go](../internal/modules/health/handler_test.go) | 생존·준비 프로브 상태와 커밋 응답 테스트 — TestLivez_ReportsStatusAndCommit → Handler.Livez; TestReadyz_* → Handler.Readyz; TestNewHandler_EmptyCommit_FallsBackToDev → NewHandler | 유지 |
| [internal/modules/health/helpers_test.go](../internal/modules/health/helpers_test.go) | 프로브 HTTP 테스트 요청·본문 읽기 유틸리티 — request/readAll은 같은 패키지 handler_test.go의 TestLivez_ReportsStatusAndCommit/TestReadyz_*/TestNewHandler_EmptyCommit_FallsBackToDev에서 호출 | 유지 |
| [internal/modules/task/handler_bench_test.go](../internal/modules/task/handler_bench_test.go) | 태스크 목록·생성 경로 성능 회귀 벤치마크 — BenchmarkTasksList/BenchmarkTasksCreate → newTestApp → RegisterRoutes/NewService/NewRepository; Go Benchmark 자동발견, README에 -bench BenchmarkTasksList 명시 | 유지 |
| [internal/modules/task/handler_fuzz_test.go](../internal/modules/task/handler_fuzz_test.go) | 태스크 경로 ID 파서 퍼즈 테스트 — FuzzParseID → parseID; 정상 양수·0·음수·문자열·uint64 초과·전각 숫자 seed와 AppError/400 또는 양수 성공 계약 | 유지 |
| [internal/modules/task/handler_test.go](../internal/modules/task/handler_test.go) | 실제 SQLite·GORM 태스크 CRUD 통합 테스트와 벤치마크 fixture — TestHandler_Create/List/Get/Update/Delete/BadIDParam_* → RegisterRoutes/Handler/Service/Repository; TestHandler_Update_DueDateNullClears → NullableTime.UnmarshalJSON 및 PATCH 경로; newTestApp은 benchmark에서도 호출 | 유지 |
| [internal/modules/task/repository_test.go](../internal/modules/task/repository_test.go) | 드라이버별 행 잠금 SQL·없는 행 수정 테스트 — TestRowLock_SQLite_OmitsRowLock/Postgres_GeneratesRowLock → applyRowLock; TestUpdate_NotFound_ReturnsErrNotFound → Repository.Update; PostgreSQL는 DisableAutomaticPing 및 DryRun으로 외부 DB 불필요 | 유지 |
| [internal/modules/task/service_test.go](../internal/modules/task/service_test.go) | 메모리 fake repository를 이용한 태스크 서비스 단위 테스트 — TestService_Create/Get_NotFound/Update_*/Delete_NotFound → Service.Create/Get/Update/Delete; fakeRepo는 service.Repository 인터페이스 충족 | 유지 |
| [internal/pagination/pagination_test.go](../internal/pagination/pagination_test.go) | 페이지 기본값·상한·오프셋·총 페이지 수 단위 테스트 — TestNewPageQuery_Clamps → NewPageQuery/MaxLimit; TestPageQuery_Offset → PageQuery.Offset; TestNewPageMeta_Edges → NewPageMeta | 유지 |
| [internal/router/openapi_test.go](../internal/router/openapi_test.go) | OpenAPI 스펙과 실제 라우트 상호 드리프트 테스트 — TestOpenAPI_NoRouteDrift → router.New 및 api.FS/openapi.yaml; parseSpecPaths/collectCodeRoutes는 경로·메서드 양방향 비교, GET /openapi.yaml 응답 및 Content-Type 확인 | 유지 |
| [internal/router/router_test.go](../internal/router/router_test.go) | 전체 라우터 조립·인증/전역 제한·프로브·에러 매핑 회귀 테스트 — TestAuthLimiter_*/TestGlobalLimiter_SkipsProbeEndpoints → New의 limiter 조립; TestAuthDisabled_LoginRouteAbsent; TestLivez_ReportsBuildCommit; TestErrorHandler_InternalErrorMasked → ErrorHandler; TestMapFiberError_Branches → mapFiberError | 유지 |
| [internal/validator/bind_test.go](../internal/validator/bind_test.go) | 검증·파싱 오류의 HTTP 에러 변환 단위 테스트 — TestBindErrorToAppError_ValidationViolation_422WithFieldDetails/ParseFailure_400/WrappedValidationErrors_Still422 → BindErrorToAppError; 래핑 오류와 field detail 보존 확인 | 유지 |

### 문서·설정·개발 도구와 로컬 데이터 (38개)

| 파일 | 역할과 사용 근거 | 처리 |
|---|---|---|
| [.agents/rules/README.md](../.agents/rules/README.md) | 상세 에이전트 규칙 등록 절차 — AGENTS.md의 규칙 확장 안내; opencode.json instructions glob | 유지 |
| [.agents/rules/new-module-checklist.md](../.agents/rules/new-module-checklist.md) | 새 업무 모듈 생성 체크리스트 — AGENTS.md 상세 규칙 인덱스와 CONTRIBUTING.md 참조 | 유지·갱신 |
| [.air.toml](../.air.toml) | 개발 서버 핫 리로드 설정 — Makefile dev의 air 실행; cmd/api 빌드와 cmd/internal/db/api 감시 | 유지 |
| [.cursor/rules/project.mdc](../.cursor/rules/project.mdc) | Cursor 규칙 요약 — alwaysApply frontmatter; AGENTS.md가 지정한 도구별 진입점 | 유지·갱신 |
| [.dockerignore](../.dockerignore) | 컨테이너 빌드 입력에서 로컬 파일 제외 — Dockerfile의 COPY . . 이전 빌드 컨텍스트 필터; Git·DB·환경파일·빌드 산출물 제외 | 추가 |
| [.env.example](../.env.example) | 환경변수 예시 및 기본값 — README Quickstart 복사 대상; internal/config/config.go env 태그와 대조 | 유지 |
| [.github/CODEOWNERS](../.github/CODEOWNERS) | 경로별 코드 소유자 지정 — GitHub가 읽는 CODEOWNERS 진입점 | 유지 |
| [.github/ISSUE_TEMPLATE/bug_report.md](../.github/ISSUE_TEMPLATE/bug_report.md) | 버그 신고 입력 양식 — GitHub Issue 생성 시 읽는 이름과 about frontmatter | 유지 |
| [.github/ISSUE_TEMPLATE/feature_request.md](../.github/ISSUE_TEMPLATE/feature_request.md) | 기능 제안 입력 양식 — GitHub Issue 생성 시 읽는 이름과 about frontmatter | 유지 |
| [.github/PULL_REQUEST_TEMPLATE.md](../.github/PULL_REQUEST_TEMPLATE.md) | 변경 요약 및 검증 체크리스트 — GitHub PR 생성 기본 본문; CONTRIBUTING.md 검증 절차 | 유지·갱신 |
| [.github/copilot-instructions.md](../.github/copilot-instructions.md) | Copilot 운영 규칙 연결 — 실제 ../AGENTS.md 심링크 대상 존재 확인 | 유지 |
| [.github/dependabot.yml](../.github/dependabot.yml) | 의존성 갱신 일정 — GitHub Dependabot gomod/github-actions 주간 갱신 설정 | 유지 |
| [.github/workflows/ci.yml](../.github/workflows/ci.yml) | 품질 및 실제 DB 검증 자동화 — main push/pull_request; verify/migrations/smoke 잡 | 유지 |
| [.github/workflows/release.yml](../.github/workflows/release.yml) | 태그 기반 바이너리 릴리스 — v* 태그 push; .goreleaser.yaml 로드 | 유지 |
| [.gitignore](../.gitignore) | 로컬 데이터와 산출물의 Git 추적 제외 — git check-ignore로 data/app.db 제외 확인; 빌드/환경/에디터 패턴 | 유지 |
| [.golangci.yml](../.golangci.yml) | Go 정적 검사 설정 — Makefile lint와 CI golangci-lint 실행 시 기본 설정 로드 | 유지 |
| [.goreleaser.yaml](../.goreleaser.yaml) | 릴리스 대상과 압축파일 정의 — release.yml GoReleaser; api/migrate의 Linux/macOS 빌드 | 유지·갱신 |
| [.windsurfrules](../.windsurfrules) | Windsurf 운영 규칙 연결 — AGENTS.md를 가리키는 유효한 심링크 | 유지 |
| [AGENTS.md](../AGENTS.md) | 저장소 작업 규칙의 단일 출처 — README/CONTRIBUTING과 도구별 심링크; 상세 규칙 인덱스 | 유지·갱신 |
| [CHANGELOG.md](../CHANGELOG.md) | 배포 및 미배포 변경 이력 — README 변경 이력 링크와 CONTRIBUTING 변경 기록 규칙 | 유지·갱신 |
| [CLAUDE.md](../CLAUDE.md) | Claude Code 운영 규칙 연결 — AGENTS.md를 가리키는 유효한 심링크 | 유지 |
| [CONTRIBUTING.md](../CONTRIBUTING.md) | 개발 및 기여 절차 — README 기여 안내; 도구 설치와 검증 명령 | 유지·갱신 |
| [Dockerfile](../Dockerfile) | API 및 마이그레이션 컨테이너 빌드 — docker-compose.yml app/migrate build 설정; cmd/api와 cmd/migrate | 유지 |
| [GEMINI.md](../GEMINI.md) | Gemini CLI 운영 규칙 연결 — AGENTS.md를 가리키는 유효한 심링크 | 유지 |
| [LICENSE](../LICENSE) | MIT 사용·배포 조건 — README License 배지/링크; 소프트웨어 사용 허가 | 유지 |
| [Makefile](../Makefile) | 개발·검증·배포 명령 진입점 — README/CONTRIBUTING의 make 명령; Go/Docker/Air 도구 호출 | 유지·갱신 |
| [PROMPT.md](../PROMPT.md) | 프로젝트 설계 사양과 초기 의도 기록 — AGENTS.md와 README 개발 사양 참조; 현재 운영 규칙은 AGENTS.md 우선 | 유지·갱신 |
| [README.md](../README.md) | 프로젝트 사용 및 구조 안내 — 저장소 기본 문서; Quickstart/API/테스트/개발 규칙 링크 | 유지·갱신 |
| [SECURITY.md](../SECURITY.md) | 보안 보고와 지원 정책 — CONTRIBUTING.md의 취약점 보고 링크; GitHub 보안 정책 진입점 | 유지·갱신 |
| `data/app.db` | 로컬 개발용 SQLite 데이터 — config.DBDSN 기본경로와 tasks 모델 컬럼 일치; 미추적 앱 데이터이므로 유지 | 로컬 보존·Git 제외 |
| [docker-compose.yml](../docker-compose.yml) | 로컬 DB와 전체 앱 스택 조립 — Makefile docker-up/down와 README PostgreSQL/MySQL 실행 절차 | 유지 |
| [docs/architecture.md](../docs/architecture.md) | 요청·계층·DB 정책 설명 — CHANGELOG.md와 전체 파일 목록의 설계 참조 | 유지·갱신 |
| [docs/authenticator-sketch.md](../docs/authenticator-sketch.md) | 추후 DB 인증 확장 설계 — README Roadmap와 SECURITY.md, docs/architecture.md 참조; 구현 전 참고 문서 | 유지 |
| [docs/file-inventory.md](../docs/file-inventory.md) | 전체 파일 역할과 처리 근거 목록 — README와 CONTRIBUTING에서 안내하는 전수 점검 결과; 이 문서 자체도 목록에 포함 | 추가 |
| [go.mod](../go.mod) | Go 버전과 모듈 의존성 선언 — Go build/test 도구, CI/release go-version-file, Docker 의존성 다운로드 | 유지 |
| [go.sum](../go.sum) | 다운로드 모듈 체크섬 — Go 모듈 다운로드 및 go mod verify 무결성 검사 | 유지 |
| [opencode.json](../opencode.json) | OpenCode 상세 규칙 로딩 설정 — .agents/rules/**/*.md instructions glob; AGENTS.md에 로딩 방식 명시 | 유지 |
| [scripts/smoke.sh](../scripts/smoke.sh) | 실제 DB 기반 앱 CRUD 스모크 — Makefile smoke 및 CI postgres/mysql smoke 단계 | 유지·갱신 |

## 이후 파일 정리

파일을 추가·이동·삭제할 때는 이 목록의 경로와 용도를 함께 갱신한다. 빌드 산출물(`bin/`, `tmp/`, `dist/`, 커버리지·프로파일)과 OS 메타데이터는 재생성 가능한 정리 대상이다.
`.env`와 `data/`처럼 로컬 설정·데이터가 있는 경로는 사용 근거와 내용을 확인한 뒤 별도로 판단한다.
테스트 helper, `testdata/fuzz/` 입력, `go:embed` 파일, 에이전트 심링크, 플랫폼이 자동으로 읽는 설정을 import 검색 결과만으로 삭제하지 않는다.
