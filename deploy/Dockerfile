# 构建阶段
FROM golang:1.23-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /app

ARG SERVICE_PATH=./cmd/services/management-service

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app/bin/service ${SERVICE_PATH}

# 运行阶段
FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=builder /app/bin/service /app/service
COPY --from=builder /app/config/config.yml ./config/

EXPOSE 8080

ENTRYPOINT ["/app/service"]
