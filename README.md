# go-fiber-starter

[![CI](https://github.com/inininax/go-fiber-starter/actions/workflows/ci.yml/badge.svg)](https://github.com/inininax/go-fiber-starter/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

Go 최신 버전 + **Fiber v3** + **GORM** 기반 REST API 보일러플레이트.
클론 후 명령어 한 번으로 실행되고, 정해진 패턴을 복사해 모듈을 확장하는 스타터 템플릿이다.

- **즉시 실행**: DB 설치 없이 sqlite로 `make run`
- **프로덕션 경로**: postgres/mysql + 버전 기반 SQL 마이그레이션(`cmd/migrate`)
- **관찰 가능성**: slog JSON 로그(request_id 상관관계), `/livez` `/readyz` 분리, Prometheus `/metrics`
- **보안 기본 장착**: helmet, CORS, rate limit, body limit, 서버 타임아웃
- **graceful shutdown**: SIGTERM 시 진행 중 요청 완료 후 종료

## 요구 사항

| 도구 | 버전 | 비고 |
|---|---|---|
| Go | 1.27.1+ | 프로젝트 최소 요구. [go.mod](go.mod) 기준 |
| Docker(선택) | — | postgres/mysql 실행 시 |
| air(선택) | latest | 핫 리로드. `make tools` |
| golangci-lint(개발) | v2.13.2 | 로컬·CI 동일 버전. `make tools` |

Go 버전의 기준은 `go.mod`의 `go` 지시문이다. CI와 릴리스 워크플로는 이 값을 읽고,
Docker 빌더도 같은 버전(`golang:1.27.1-alpine`)을 사용한다. 로컬은 `go version`으로 확인한다.
Go 버전을 올릴 때는 `go.mod`, Dockerfile, 이 문서를 함께 갱신하고 `make tools`로 개발 도구를 다시 설치한다.

## Quickstart (3분)

```bash
cp .env.example .env   # 기본값 = sqlite 무설치
make run               # 또는: go run ./cmd/api
```

```bash
curl http://localhost:8080/livez   # {"status":"ok","commit":"..."}
```

핫 리로드 개발:

```bash
make tools && make dev
```

postgres로 실행:

```bash
docker compose up -d postgres
DB_DRIVER=postgres \
DB_DSN='postgres://starter:starter@localhost:5432/starter?sslmode=disable' \
go run ./cmd/api
```

> postgres/mysql은 prod 계열 드라이버다. `APP_ENV=prod`에서는 `DB_AUTO_MIGRATE=true`가 하드 차단되며
> SQL 마이그레이션(`make migrate-up`)을 사용해야 한다.

## 환경변수

`.env.example` 참고. 전체 목록과 기본값은 [internal/config/config.go](internal/config/config.go)가 단일 출처이다.

| 변수 | 기본값 | 설명 |
|---|---|---|
| `APP_ENV` | `local` | local/dev/prod. prod+sqlite 조합 금지 |
| `APP_PORT` | `8080` | |
| `LOG_LEVEL` | `info` | 일반 SQL은 debug, 오류·느린 SQL은 error/warn으로 기록(실제 출력은 로그 레벨에 따름) |
| `DB_DRIVER` | `sqlite` | postgres / mysql / sqlite |
| `DB_DSN` | `data/app.db` | 드라이버별 DSN 형식은 `.env.example` 주석 |
| `DB_AUTO_MIGRATE` | `true` | prod(pg/mysql)에서 true면 시작 거부 |
| `CORS_ALLOWED_ORIGINS` | `*` | 콤마 구분. prod에서 `*`는 시작 거부 |
| `RATE_LIMIT_PER_MINUTE` | `120` | IP당 분당 요청 수 |
| `TRUST_PROXY` | `false` | 역프록시 뒤에서만 true. true면 `TRUST_PROXY_PROXIES` 필수 |
| `TRUST_PROXY_PROXIES` | — | 신뢰할 프록시 IP/CIDR 목록(콤마 구분). TRUST_PROXY=false인데 설정하면 시작 거부 |
| `TRUST_PROXY_HEADER` | `X-Forwarded-For` | 클라이언트 IP를 읽을 헤더 |
| `AUTH_ENABLED` | `false` | true면 로그인 활성화 + `/api/v1/tasks` 전체 Bearer 요구 |
| `AUTH_RATE_LIMIT_PER_MINUTE` | `10` | 로그인 분당 시도 한도(IP별). 전역 값 이하 |
| `AUTH_JWT_SECRET` | — | AUTH_ENABLED=true 시 32바이트 이상 필수 |
| `AUTH_TOKEN_TTL` | `1h` | 액세스 토큰 만료 |

## 인증 (JWT 스캐폴드)

`AUTH_ENABLED=true`로 실행하면 데모 자격증명(env)으로 JWT를 발급하고 task API가 보호된다.
로그인 엔드포인트는 IP별 분당 시도 한도(`AUTH_RATE_LIMIT_PER_MINUTE`)로 보호되어 brute force 시도를 429로 거부한다.

> ⚠️ **다중 replica 주의**: rate limiter는 인메모리 상태라 인스턴스별로 예산이 분할된다.
> 수평 확장 배포에서는 Redis 기반 limiter 등 외부화가 필요하다(Roadmap 참고).
> 또한 `TRUST_PROXY=false` 상태의 프록시 뒤 배포는 전 사용자가 프록시 IP 하나로 묶여 집단 429가 발생한다 —
> 프록시 뒤에서는 반드시 `TRUST_PROXY=true` + 신뢰 프록시 목록을 설정할 것.

```bash
AUTH_ENABLED=true AUTH_JWT_SECRET=$(openssl rand -base64 48) go run ./cmd/api

# 로그인 → 토큰 발급 (200)
curl -s -X POST http://localhost:8080/api/v1/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"username":"admin","password":"admin123"}'
# {"success":true,"data":{"token":"eyJ...","token_type":"Bearer","expires_at":"..."}}

TOKEN=eyJ...

# 토큰 없이 접근 → 401 UNAUTHORIZED
curl -s http://localhost:8080/api/v1/tasks

# Bearer 토큰으로 접근
curl -s http://localhost:8080/api/v1/tasks -H "Authorization: Bearer $TOKEN"
```

> 이 스캐폴드의 자격증명 검사는 env 기반 demo 구현이다. 실제 사용자 스토어 연동 시
> `internal/modules/auth/service.go`의 `Authenticator` 인터페이스를 DB 기반(비밀번호 해시 비교)으로 교체하라.
> 핸들러에서 현재 신원: `c.Locals(auth.IdentityKey).(auth.Identity)`.

> **⚠️ 데모 자격증명은 local/dev 전용이다.** 기본값(`admin`/`admin123`)은 절대 운영에 사용하지 말 것.
> `APP_ENV=prod` + `AUTH_ENABLED=true` 상태에서 기본값이 남아 있으면 시작이 거부된다
> (`AUTH_DEMO_USERNAME`/`AUTH_DEMO_PASSWORD`를 명시적으로 설정하거나 DB 기반 Authenticator로 교체).
> 운영 환경은 DB 기반 Authenticator(비밀번호 해시 비교) 사용을 권장한다.

## API 예제

`/api/v1`의 업무 API는 JSON 성공·오류 응답에 공통 엔벨로프를 사용한다. 204 응답은 본문이 없다.
`/livez`·`/readyz`는 프로브 전용 JSON, `/metrics`는 Prometheus 텍스트, `/openapi.yaml`은 YAML을 반환한다.

```json
{"success":true,"data":[],"meta":{"page":1,"limit":20,"total_count":0,"total_pages":0}}
```

```json
{"success":false,"error":{"code":"TASK_NOT_FOUND","message":"task not found"}}
```

```bash
BASE=http://localhost:8080/api/v1

# 생성 (201)
curl -s -X POST $BASE/tasks -H 'Content-Type: application/json' \
  -d '{"title":"write tests"}'

# 목록 (페이지네이션 meta 포함, limit 최대 100)
curl -s "$BASE/tasks?page=1&limit=20"

# 단건 (없으면 404 TASK_NOT_FOUND)
curl -s $BASE/tasks/1

# 부분 수정 (present 필드만 반영)
curl -s -X PATCH $BASE/tasks/1 -H 'Content-Type: application/json' -d '{"done":true}'

# 삭제 (204, soft delete)
curl -s -X DELETE $BASE/tasks/1

# 검증 실패 → 422 + 필드별 상세
curl -s -X POST $BASE/tasks -H 'Content-Type: application/json' -d '{"title":""}'
```

기타: `GET /readyz`(DB ping), `GET /metrics`(Prometheus), `GET /openapi.yaml`(OpenAPI 3.0.3 스펙).

## 새 모듈 추가 가이드

`task` 모듈이 템플릿이다. 5단계:

1. **복제**: `internal/modules/task/`를 `<name>/`으로 복사한 뒤 model/dto/repo/service/handler/routes를 도메인에 맞게 수정
2. **인터페이스 위치 유지**: Repository 인터페이스는 service.go(소비자)에 선언 — mock이 자유로워진다
3. **마이그레이션 등록**:
   - sqlite(dev): `db/migrations` 불필요, 아래 4번의 AutoMigrate 목록에 모델 추가
   - postgres/mysql(prod): `db/migrations/{postgres,mysql}/000002_xxx.{up,down}.sql` 작성 (`make migrate-new name=xxx`)
4. **조립**: `cmd/api/main.go`의 `AutoMigrateIfNeeded(...)` 목록에 모델 추가(sqlite용)
5. **라우트 마운트**: `internal/router/wiring.go`에 `NewService(NewRepository(db))` 추가,
   `router.go`에서 `RegisterRoutes(v1, svc)` 한 줄 추가

규칙: `fiber.Ctx`는 handler에만, `*gorm.DB`는 repository에만 노출. 의존성 방향 `handler → service → repository`.

## 트랜잭션 패턴

service는 한 번에 수행할 업무와 변경 규칙을 정하고, repository 인터페이스로 저장을 요청한다.
현재 task 부분 수정은 [service.go](internal/modules/task/service.go)의 `Update`가 변경 함수를 넘기고,
[repository.go](internal/modules/task/repository.go)의 `Update(ctx, ...)`가
`r.db.WithContext(ctx).Transaction(...)` 안에서 조회·변경·저장을 묶는다.
PostgreSQL/MySQL은 조회에 `FOR UPDATE` 행 잠금을 사용하며 sqlite에는 적용하지 않는다.

다중 저장소 원자성이 필요하면 service에 업무 단위 인터페이스를 정의하고, 트랜잭션과
`*gorm.DB`는 repository 구현 안에 둔다. `database.WithTx`는 이를 위한 확장 헬퍼로 남아 있으며
현재 운영 코드에서 호출하지 않는다. 사용할 때는 repository 안에서 `r.db.WithContext(ctx)`를 전달한다.

## 마이그레이션 워크플로 (postgres/mysql)

```bash
export DB_DRIVER=postgres
export DB_DSN='postgres://starter:starter@localhost:5432/starter?sslmode=disable'

make migrate-up      # 미적용 전부 적용
make migrate-down    # 1스텝 롤백
go run ./cmd/migrate version     # 현재 버전/더티 상태
go run ./cmd/migrate force 1     # 더티 수동 복구
```

SQL 파일은 `embed.FS`로 바이너리에 포함된다. sqlite는 이 흐름 대상이 아니며 AutoMigrate를 쓴다.

## 폴더 구조

```
cmd/api          진입점(조립 + graceful shutdown)
cmd/migrate      SQL 마이그레이션 CLI(up/down/version/force)
api              임베드되는 OpenAPI YAML 스펙
internal/
  config         env 파싱 + fail-fast 검증
  database       연결/풀/slog 어댑터/AutoMigrate 정책
  apperror       에러 카탈로그(AppError)
  httpx          응답 엔벨로프 헬퍼
  pagination     페이지네이션 파라미터/meta
  testutil       handler 테스트 요청 헬퍼
  middleware     요청 로깅, Prometheus 수집
  router         fiber 앱 조립, 전역 ErrorHandler
  modules/auth   JWT 로그인과 인증 가드
  modules/task   예제 도메인 = 모듈 템플릿
  modules/health livez/readyz
  validator      validate 태그 ↔ Fiber 바인딩 연결
db/migrations    버전 기반 SQL(postgres, mysql)
```

각 파일의 역할, 실제 호출·설정 연결, 유지 이유와 로컬 생성물은
[전체 파일 목록](docs/file-inventory.md)을 참고한다. 요청 흐름과 계층 경계는
[아키텍처](docs/architecture.md)에 정리되어 있다.

## 테스트

```bash
make test        # 전체 (-race)
make test-cov    # 커버리지 HTML 리포트
make smoke       # 실제 DB 대상 E2E 스모크(docker compose postgres 필요)
go test ./internal/modules/task/ -bench BenchmarkTasksList -benchtime 5x   # 벤치마크
```

- service 단위 테스트: fake repository 주입(외부 I/O 없음)
- handler 통합 테스트: 실제 GORM + sqlite 파일(각 테스트 임시 디렉터리) + `app.Test()`
- E2E 스모크(`scripts/smoke.sh`): 실제 DB에서 마이그레이션→부팅→livez commit 노출→CRUD 전 주기 검증. CI가 pg/mysql 각각 실행
- 벤치마크: task 핸들러 목록/생성 경로 추적, 퍼즈 타깃(parseID/stripScheme) 포함
- CI는 `-coverpkg=./...` 계량식으로 전체 커버리지를 측정하고, 65%(환경변수 `COVERAGE_MIN`) 미만이면 실패한다

## CI

`.github/workflows/ci.yml`의 `verify` 잡은 gofmt 체크 → golangci-lint → vet → test(-race, cover)
→ coverage gate → CGO 없는 빌드를 실행한다. 별도 `migrations` 잡은 PostgreSQL/MySQL 마이그레이션의
적용·롤백·재적용을, `smoke` 잡은 두 DB에서 앱 부팅과 CRUD 전체 흐름을 검증한다.
Docker 이미지: 멀티스테이지 빌드, non-root, HEALTHCHECK(/livez).

## AI 에이전트 협업 (Codex / Claude / Cursor / opencode 등)

이 저장소는 AI 에이전트 공동 작업을 전제로 문서가 이원화되어 있다.
**운영 규칙의 단일 출처는 루트 `AGENTS.md` 하나뿐**이며, 나머지는 이를 가리키는 얇은 포인터다.

| 에이전트 | 연결 방식 | 파일 |
|---|---|---|
| OpenAI Codex / opencode / Cursor(신버전) | 네이티브 자동 인식 | `AGENTS.md` |
| Claude Code | 심링크 + `@import` | `CLAUDE.md` → `AGENTS.md` |
| Gemini CLI | 심링크 | `GEMINI.md` → `AGENTS.md` |
| Windsurf | 심링크 | `.windsurfrules` → `AGENTS.md` |
| GitHub Copilot | 심링크 | `.github/copilot-instructions.md` → `AGENTS.md` |
| Cursor(구버전 호환) | 요약본(mdc frontmatter) | `.cursor/rules/project.mdc` |

주제별 상세 규칙은 **`.agents/rules/*.md`**에 둔다. OpenCode는 `opencode.json`의 glob으로
자동 로드하고, Claude Code는 AGENTS.md의 `@import` 줄로 해석한다.
**규칙 변경 시 AGENTS.md(또는 .agents/rules/)만 수정한다.** 절차: `.agents/rules/README.md`.
새 에이전트 도입 시에도 별도 문서를 만들지 말고 AGENTS.md 심링크만 추가할 것.

## Roadmap

- ~~JWT 인증 모듈 스캐폴드(golang-jwt/v5, Bearer 추출 미들웨어)~~ → 구현 완료(위 "인증" 섹션). 남은 과제: DB 기반 Authenticator 교체([스케치 문서](docs/authenticator-sketch.md))
- ~~OpenAPI 3 스펙 문서화~~ → 구현 완료(`api/openapi.yaml`, `GET /openapi.yaml` 서빙, 드리프트 테스트 `internal/router/openapi_test.go`). 남은 과제: Swagger UI `/docs` 서빙
- OpenTelemetry tracing(현재 request_id 상관관계까지만 제공)
- cursor 기반 페이지네이션(대량 테이블용)
- compress / ETag 미들웨어
- 분산 rate limiter(Redis 등) — 다중 replica 배포 시 필요, 현재는 인메모리(위 "인증" 섹션 주의 참고)

## 기여

[CONTRIBUTING.md](CONTRIBUTING.md) 참고. 운영 규칙의 단일 출처는 [AGENTS.md](AGENTS.md)다.

## 변경 이력

[CHANGELOG.md](CHANGELOG.md).

## 라이선스

[LICENSE](LICENSE) — MIT
