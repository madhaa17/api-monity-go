FROM golang:1.25-alpine AS builder

WORKDIR /app

RUN apk add --no-cache ca-certificates

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux \
    go build -trimpath \
    -ldflags="-s -w" \
    -o /app/server ./cmd/server/main.go

FROM alpine:3.22

WORKDIR /app

RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /app/server ./server

# The app reads APP_PORT (see internal/config). Override it in docker-compose / Dokploy.
ENV APP_PORT=8080

EXPOSE 8080

USER 10001:10001

# /health also pings the database. busybox wget ships with alpine.
HEALTHCHECK --interval=30s --timeout=5s --start-period=15s --retries=3 \
    CMD wget -qO- "http://127.0.0.1:${APP_PORT}/health" >/dev/null || exit 1

CMD ["./server"]
