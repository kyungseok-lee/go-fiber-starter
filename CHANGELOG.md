# Changelog

모든 주목할 만한 변경은 이 파일에 기록된다.
형식은 [Keep a Changelog](https://keepachangelog.com/ko/1.1.0/)를 따르고, 버전은 [Semantic Versioning](https://semver.org/lang/ko/)을 준수한다.

## [Unreleased]

### Added

- `docs/file-inventory.md`에 전체 파일의 역할·연결·유지 이유와 로컬 생성물 정리 기준 기록
- `docs/architecture.md` — 요청 생명주기·계층 경계·스키마 정책 한 장 정리
- OpenAPI 스펙 보강: securitySchemes(Bearer), tasks 401 응답, 대표 요청/응답 examples
- GitHub 표준 파일: 이슈 템플릿, PR 템플릿(검증 체크리스트), CODEOWNERS
- 순수 패키지 외 잔여 테스트: migrator prod 차단 규칙, GORM slog 어댑터, health readyz 503,
  ErrorHandler 500 마스킹 계약, mapFiberError 분기, validator 직접 테이블, config Load+미커버 규칙,
  due_date 명시적 null 해제
- 릴리스 자동화: `.goreleaser.yaml` + 태그 push 시 테스트→바이너리 릴리스 워크플로우
- E2E 스모크(`scripts/smoke.sh`, CI `smoke` 잡): pg/mysql 실제 DB에서 부팅→CRUD 전 주기 검증
- 벤치마크(task 목록/생성)와 퍼즈 타깃(parseID, stripScheme)
- `apperror`/`httpx`/`pagination` 순수 패키지 직접 단위 테스트

### Fixed

- Air 작업 디렉터리를 문자열로 설정해 `make dev` 시작 시 TOML 타입 오류 수정, 실행 경로는 현재 `entrypoint` 설정 사용
- `make docker-up`이 `full` 프로필의 앱 컨테이너까지 기동하도록 수정
- `make docker-down`에도 `full` 프로필을 적용해 앱 컨테이너까지 정리
- 대소문자를 섞은 `LOG_LEVEL`이 검증 후 로거 설정에서 다르게 해석되던 문제 수정
- 스모크의 성공·실패 경로에서 임시 바이너리·로그·프로세스를 정리하고, 마이그레이션 실패 시 잔여 파일 방지
- `.dockerignore`로 로컬 데이터·비밀 설정·Git 메타데이터·빌드 생성물이 Docker 빌드 컨텍스트에 포함되지 않도록 수정
- OpenAPI task 단건 조회·삭제의 400, 단건 조회·수정·삭제의 429 응답 누락 보완
- `.github/copilot-instructions.md` 깨진 심링크(`../AGENTS.md`로 수정) — Copilot이 규칙을 읽지 못하던 결함
- PATCH `due_date` 명시적 null 해제가 동작하지 않던 버그(**time.Time null/미전달 구분 불가 → NullableTime 도입)
- CI와 docker-compose의 DB 버전 불일치(pg 17-alpine, mysql 9로 정렬)
- `.cursor/rules/project.mdc`의 삭제된 common 패키지 참조, README Go 버전 표기(1.27+) 등 문서 드리프트

### Changed

- 현재 소스 기준으로 환경변수 전체 목록·인증 교체 절차·모듈 체크리스트·릴리스와 로컬 생성물 안내 정합성 보완
- 실제 구현을 기준으로 응답 형식 예외·트랜잭션과 context 경계·readyz 검사 범위·429 관측 범위·SQL 로그 설명 정정
- 초기 설계 문서의 테스트·커버리지·미들웨어 순서·구현 완료 기능 안내와 보안 지원 정책·PR 검증 체크리스트 최신화
- 미사용 Makefile 변수와 지원하지 않는 Windows용 릴리스 설정을 제거하고 AutoMigrate 정책 테스트의 실제 검증 보강
- Go 최소 버전과 Docker 빌더를 1.27.1로 갱신하고, golangci-lint를 v2.13.2로 업데이트
- SQLite 런타임과 PostgreSQL/MySQL 드라이버, Fiber·Prometheus 관련 모듈 및 `golang.org/x` 간접 의존성 갱신
- OpenAPI 드리프트 테스트의 YAML 파서를 YAML 조직이 유지보수하는 `go.yaml.in/yaml/v3 v3.0.5`로 전환
- README·기여 가이드에 Go 버전 기준, 개발 도구 설치, CI lint·마이그레이션·스모크 검증 안내 최신화
- GitHub Actions 의존성 갱신: checkout@v7, setup-go@v7, golangci-lint-action@v9 (dependabot)
- CI에 concurrency 취소 설정(같은 브랜치 연속 push 시 구 run 자동 취소)
- README에 다중 replica rate limiter 한계 및 TRUST_PROXY 배포 주의사항 문서화

## [0.3.0] - 2026-08-23

### Changed

- 패키지 구조 개편: `internal/common` 해체 → `apperror`/`httpx`/`pagination`/`testutil`
  분할(GitHub 인기 Go 프로젝트 패턴 준거), task 도메인 에러 모듈 귀환
- `.github/dependabot.yml` — go modules + github-actions 주간 갱신(minor/patch 그룹핑)
- OpenAPI 3.0.3 스펙(`api/openapi.yaml`) + `GET /openapi.yaml` 서빙 + 라우트 드리프트 검증 테스트
- livez 응답에 빌드 commit 노출(`Config.BuildCommit`, `dev` 폴백)
- CI 커버리지 게이트(`-coverpkg=./...` 계량, COVERAGE_MIN=65 fail-under)
- 전역 rate limiter의 probe/metrics/spec 경로 skip(exact 매칭)

## [0.2.0] - 2026-08-23

### Added

- JWT 인증 스캐폴드: `POST /api/v1/auth/login`, HS256 발급/검증, RequireAuth 가드
- 관찰 가능성: 에러 실제 상태 메트릭화(EffectiveStatus), request_id 상관관계, Prometheus 수집
- 보안: prod 데모 크리덴셜 차단 + constant-time 비교, prod CORS `*` 거부, TRUST_PROXY,
  로그인 전용 rate limiter, task UPDATE 행잠금(sqlite 예외), SQLite 디렉터리 0o700
- CI: postgres/mysql 마이그레이션 검증 잡, golangci-lint v2 연동
- AI 규칙 구조: AGENTS.md 단일 출처 + 심링크 포인터 + `.agents/rules/`(hanppyeom-ttang 패턴)

### Fixed

- 에러 응답이 메트릭·액세스 로그에 200으로 기록되던 결함
- 액세스 로그 request_id 누락(fiber v3 requestid API 전환)
- 마이그레이션 임베드 경로 버그(CI migrations 잡 실패)

## [0.1.0] - 2026-08-23

### Added

- 초기 스캐폴드: Fiber v3 + GORM 레이어드 아키텍처(handler→service→repository)
- task 예제 CRUD 모듈(모듈 템플릿) + health(livez/readyz)
- 통일 응답 엔벨로프/에러 카탈로그, fail-fast 설정 검증, graceful shutdown
- 마이그레이션 이원화: sqlite AutoMigrate(dev) / postgres·mysql 버전 SQL(`cmd/migrate`)
- 멀티 에이전트 협업 문서 체계(AGENTS.md 단일 출처)
