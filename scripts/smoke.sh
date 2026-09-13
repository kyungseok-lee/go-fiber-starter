#!/usr/bin/env bash
# smoke.sh는 실제 DB(postgres|mysql) 위에서 앱 부팅 → 헬스체크 → task CRUD 전 주기를 검증한다.
# CI(smoke 잡)와 로컬(docker compose up postgres 후 `make smoke`) 양쪽에서 사용한다.
#
# 사용법: ./scripts/smoke.sh <driver> <dsn> [port]
set -euo pipefail

DRIVER="${1:?usage: smoke.sh <postgres|mysql> <dsn> [port]}"
DSN="${2:?usage: smoke.sh <driver> <dsn> [port]}"
PORT="${3:-8090}"
BASE="http://127.0.0.1:${PORT}"

command -v go > /dev/null || { echo "go required"; exit 1; }
curl --version > /dev/null || { echo "curl required"; exit 1; }

# 포트 선점은 readyz 폴링 실패로만 알 수 있어 원인 파악이 늦는다 — 시작 전에 즉시 거부.
if curl -s --max-time 1 "http://127.0.0.1:${PORT}" > /dev/null 2>&1; then
  echo "port ${PORT} already in use — choose another port"; exit 1
fi

echo "[smoke] driver=${DRIVER} port=${PORT}"

SMOKE_DIR=$(mktemp -d "${TMPDIR:-/tmp}/go-fiber-smoke.XXXXXX")
BIN="${SMOKE_DIR}/api"
LOG="${SMOKE_DIR}/api.log"
APP_PID=""
cleanup() {
  local status=$?
  if [ -n "${APP_PID}" ]; then
    kill "${APP_PID}" 2>/dev/null || true
    wait "${APP_PID}" 2>/dev/null || true
  fi
  if [ "${status}" -ne 0 ] && [ -f "${LOG}" ]; then
    tail -n 40 "${LOG}" >&2
  fi
  rm -rf -- "${SMOKE_DIR}"
  exit "${status}"
}
trap cleanup EXIT
trap 'exit 130' INT
trap 'exit 143' TERM

# 1. 빌드 + 마이그레이션 (prod 계열 경로를 그대로 밟는다)
export DB_DRIVER="${DRIVER}" DB_DSN="${DSN}" DB_AUTO_MIGRATE=false APP_PORT="${PORT}"
go build -o "${BIN}" ./cmd/api
go run ./cmd/migrate up

# 2. 앱 부팅 + readyz 폴링(최대 15초)
"${BIN}" > "${LOG}" 2>&1 &
APP_PID=$!

for _ in $(seq 1 30); do
  if curl -fsS "${BASE}/readyz" > /dev/null 2>&1; then break; fi
  sleep 0.5
done
curl -fsS "${BASE}/readyz" > /dev/null || { echo "readyz failed" >&2; exit 1; }

fail() { echo "FAIL: $1" >&2; exit 1; }

# 3. livez에 빌드 commit 노출 확인
curl -fsS "${BASE}/livez" | grep -q '"commit"' || fail "livez missing commit field"

# 4. CRUD 전 주기
CREATE_BODY='{"title":"smoke-test"}'
CREATED=$(curl -fsS -X POST "${BASE}/api/v1/tasks" -H 'Content-Type: application/json' -d "${CREATE_BODY}") \
  || fail "create failed"
ID=$(echo "${CREATED}" | sed -E 's/.*"id":([0-9]+).*/\1/')
[ -n "${ID}" ] && [ "${ID}" != "0" ] || fail "created id not parsed: ${CREATED}"

curl -fsS "${BASE}/api/v1/tasks?page=1&limit=10" | grep -q '"total_count"' || fail "list meta missing"
curl -fsS "${BASE}/api/v1/tasks/${ID}" | grep -q "smoke-test" || fail "get by id mismatch"

UPDATED=$(curl -fsS -X PATCH "${BASE}/api/v1/tasks/${ID}" -H 'Content-Type: application/json' -d '{"done":true}')
echo "${UPDATED}" | grep -q '"done":true' || fail "patch did not apply"

CODE=$(curl -s -o /dev/null -w '%{http_code}' -X DELETE "${BASE}/api/v1/tasks/${ID}")
[ "${CODE}" = "204" ] || fail "delete want 204, got ${CODE}"
CODE=$(curl -s -o /dev/null -w '%{http_code}' "${BASE}/api/v1/tasks/${ID}")
[ "${CODE}" = "404" ] || fail "deleted task still reachable: got ${CODE}"

echo "[smoke] ALL CHECKS PASSED"
