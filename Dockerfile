# --- 빌드 스테이지 ---
FROM golang:1.27.1-alpine AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ARG COMMIT=dev
RUN CGO_ENABLED=0 GOOS=linux go build \
	-ldflags="-s -w -X main.commit=${COMMIT}" \
	-o /out/api ./cmd/api && \
	CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /out/migrate ./cmd/migrate

# --- 실행 스테이지 ---
# alpine은 LTS가 없고 릴리스마다 ~2년 지원. 마이너 태그 고정으로 보안 패치는 자동 수급.
# 마이너 bump는 https://endoflife.date/alpine-linux 기준 분기 점검(현재 3.23 → EOL 2027-11).
# digest 핀은 갱신 자동화 부재 시 역효과라 채택하지 않음.
FROM alpine:3.23
RUN addgroup -S app && adduser -S app -G app \
	&& apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=builder /out/api /app/api
COPY --from=builder /out/migrate /app/migrate
USER app
EXPOSE 8080

HEALTHCHECK --interval=15s --timeout=3s --start-period=5s --retries=3 \
	CMD wget -qO- http://127.0.0.1:8080/livez || exit 1

ENTRYPOINT ["/app/api"]
