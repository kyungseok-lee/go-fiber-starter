# ci.yml의 GOLANGCI_LINT_VERSION과 일치 필수
GOLANGCI_LINT_VERSION ?= v2.13.2
MIGRATE_NEW ?= change_description
name ?= # 소문자 별칭: make migrate-new name=add_users
ifneq ($(strip $(name)),)
MIGRATE_NEW := $(name)
endif

.DEFAULT_GOAL := help
.PHONY: help run dev smoke tools test test-cov lint fmt vet build tidy \
	docker-up docker-down migrate-up migrate-down migrate-new

help: ## 사용 가능한 타깃 나열
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-14s\033[0m %s\n", $$1, $$2}'

run: ## 개발 서버 실행 (sqlite 자동 마이그레이션 포함)
	go run ./cmd/api

dev: ## 핫 리로드 실행 (air 필요: make tools 또는 brew install air)
	air

test: ## 전체 테스트 (-race)
	go test ./... -race -count=1

test-cov: ## 커버리지 리포트 생성 (coverage.html)
	go test ./... -race -count=1 -coverpkg=./... -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html
	@go tool cover -func=coverage.out | tail -1

fmt: ## 코드 포맷
	gofmt -w .

vet: ## 정적 분석
	go vet ./...

lint: ## golangci-lint 실행 (미설치 시: make tools)
	golangci-lint run

build: ## 프로덕션 바이너리 빌드 → bin/
	CGO_ENABLED=0 go build -ldflags="-s -w" -o bin/api ./cmd/api
	CGO_ENABLED=0 go build -o bin/migrate ./cmd/migrate

tidy: ## 의존성 정리
	go mod tidy

tools: ## 개발 도구 설치(air, golangci-lint)
	go install github.com/air-verse/air@latest
	go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)

docker-up: ## postgres + 앱 컨테이너 기동
	docker compose --profile full up -d --build

docker-down: ## 컨테이너 중지/제거
	docker compose --profile full down -v

smoke: ## 실제 DB 대상 E2E 스모크 (compose postgres 기동 필요)
	./scripts/smoke.sh postgres 'postgres://starter:starter@localhost:5432/starter?sslmode=disable' 8090

migrate-up: ## SQL 마이그레이션 적용 (DB_DRIVER/DB_DSN 환경변수 사용)
	DB_AUTO_MIGRATE=false go run ./cmd/migrate up

migrate-down: ## 직전 버전으로 롤백 (1스텝)
	DB_AUTO_MIGRATE=false go run ./cmd/migrate down

migrate-new: ## 새 마이그레이션 파일 쌍 생성, 버전 자동 계산 (name=snake_case 설명)
	@if [ -z "$(MIGRATE_NEW)" ]; then echo "사용법: make migrate-new name=add_users"; exit 1; fi
	@mkdir -p db/migrations/postgres db/migrations/mysql
	@VER=$$(ls db/migrations/postgres 2>/dev/null | grep -oE '^[0-9]+' | sort -n | tail -1); \
	VER=$$(printf "%06d" $$((10#$${VER:-0} + 1))); \
	for d in postgres mysql; do \
		touch db/migrations/$$d/$${VER}_$(MIGRATE_NEW).up.sql db/migrations/$$d/$${VER}_$(MIGRATE_NEW).down.sql; \
	done; \
	echo "created $${VER}_$(MIGRATE_NEW).{up,down}.sql (postgres, mysql)"
