# Build stage
FROM golang:1.24.4-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git ca-certificates

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /server ./cmd/server

# Runtime stage
FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=builder /server .

# 8080 = local default; production uses APP_PORT=8386 via docker-compose
EXPOSE 8080
ENV APP_PORT=8080

ENTRYPOINT ["./server"]
