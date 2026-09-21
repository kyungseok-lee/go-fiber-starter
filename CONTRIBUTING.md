# 기여 가이드

이 저장소는 AI 에이전트와 사람이 공동 작업하는 것을 전제로 한다. 기여 전 반드시
[AGENTS.md](AGENTS.md)(운영 규칙 단일 출처)를 읽을 것.

## 개발 환경

Go 1.27.1 이상을 사용한다. 버전 기준은 [go.mod](go.mod)의 `go` 지시문이며 CI와 릴리스도 이 값을 읽는다.
Docker 빌더는 같은 Go 버전을 사용한다.

```bash
go version
go mod download
make tools
```

`make tools`는 air 최신 버전과 golangci-lint v2.13.2를 설치한다.
`$(go env GOPATH)/bin`을 `PATH`에 추가하고, Go 업그레이드 후에는 개발 도구도 다시 설치한다.
golangci-lint 버전을 변경할 때는 Makefile과 `.github/workflows/ci.yml`의 `GOLANGCI_LINT_VERSION`을 함께 갱신한다.

## 의존성 갱신

공식 릴리스와 Go 모듈 프록시에서 최신 안정 버전을 확인한다. `go list -m -u all`은
현재 모듈 경로의 업데이트만 보여주므로, `/v2` 같은 새 메이저 경로는 upstream 릴리스도 확인한다.
프리릴리스는 제외하고, 직접 의존성과 테스트·간접 의존성을 함께 갱신한다.

```bash
go list -m -u all
go get -u -t ./...
```

`modernc.org/libc`는 `modernc.org/sqlite`의 `go.mod`에 지정된 버전과 정확히 맞춘다.
생성 코드의 호환성을 위한 [upstream 요구사항](https://pkg.go.dev/modernc.org/sqlite#hdr-Fragile_modernc_org_libc_dependency)이다.
2026-09-22 기준 `modernc.org/sqlite v1.59.0`은 `modernc.org/libc v1.75.7`을 요구하므로,
위 명령으로 libc가 독립적인 최신 버전까지 올라갔다면 아래처럼 SQLite 요구 버전으로 되돌린다.
다음 갱신 때는 새 SQLite 버전의 요구값을 먼저 확인한다.

```bash
go get modernc.org/libc@v1.75.7
go mod tidy
go mod verify
```

`go.mod`·`go.sum` 변경을 검토하고 아래 필수 검증 세트를 실행한다. DB 드라이버 변경 시에는
PostgreSQL/MySQL의 마이그레이션 적용·롤백·재적용과 `scripts/smoke.sh`도 확인한다(CI의 `migrations`·`smoke` 잡).
README의 의존성 표와 CHANGELOG의 Unreleased를 함께 갱신하고, `go mod tidy -diff`가 빈 출력인지 확인한다.

## 개발 워크플로

1. 이슈 또는 디스커션으로 의도 공유(사소한 문서 수정은 생략 가능)
2. `main`에서 토픽 브랜치 생성: `feat/xxx`, `fix/xxx`, `docs/xxx`
3. 변경 + 아래 검증 세트 통과
4. PR 생성 — CI(gofmt/lint/vet/test+coverage gate/build/migrations/smoke)가 모두 녹색이어야 머지
5. 완료한 임시 브랜치·워크트리·검증 생성물을 정리한다. 보관할 작업과 로컬 데이터는 유지한다.

### 커밋 전 필수 검증 세트

```bash
gofmt -w .
make lint
go vet ./...
go test ./... -race -count=1
CGO_ENABLED=0 go build ./...
```

CI는 `gofmt -l` 빈 출력과 커버리지 65% 하한을 강제한다.

## 규칙 요약 (상세는 AGENTS.md)

- 계층: `handler → service → repository → model`. 역방향 참조 금지.
- 새 도메인은 [internal/modules/task](internal/modules/task/)를 복제하고
  [.agents/rules/new-module-checklist.md](.agents/rules/new-module-checklist.md) 체크리스트를 따른다.
- 마이그레이션 이원화: sqlite=AutoMigrate(dev), postgres/mysql prod=`cmd/migrate` SQL 버전 관리 — 둘 다 작성.
- 업무 API의 JSON 응답/에러는 `httpx.Envelope` / `apperror.AppError` 카탈로그로 통일. 204는 빈 본문이며 프로브·메트릭·스펙은 별도 형식이다.
- API 변경 시 `api/openapi.yaml`의 요청·응답 계약을 동기화한다. CI 드리프트 테스트는 경로·메서드 일치를 검사하며, 응답 계약은 해당 handler 테스트로 검증한다.
- 파일을 추가·이동·제거하거나 역할을 바꾸면 [전체 파일 목록](docs/file-inventory.md)의 경로·연결·유지 이유도 갱신한다.

## 커밋 메시지

`feat|fix|refactor|docs|chore: 한국어 요약` 형식을 사용하며, 본문에 동기와 검증 방법을 남긴다.

## 보안 취약점

[SECURITY.md](SECURITY.md)의 비공개 보고 경로를 사용할 것. 공개 Issue 금지.
