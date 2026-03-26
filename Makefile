.PHONY: dev build run test lint clean build-server run-server

BIN_DIR := bin

# ── 开发 ──

dev:
	go run ./cmd/server

# ── 构建 ──

build:
	@mkdir -p $(BIN_DIR)
	$(MAKE) build-server

build-server:
	@mkdir -p $(BIN_DIR)
	go build -o $(BIN_DIR)/gantt-server ./cmd/server

run:
	go run ./cmd/server

run-server:
	go run ./cmd/server

# ── 测试 ──

test:
	go test ./...

test-cover:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# ── 代码质量 ──

lint:
	golangci-lint run

fmt:
	gofmt -w .

vet:
	go vet ./...

# ── 清理 ──

clean:
	rm -rf $(BIN_DIR)/ coverage.out coverage.html

# ── Docker ──

docker-up:
	docker compose -f deploy/docker-compose.yml up -d

docker-down:
	docker compose -f deploy/docker-compose.yml down

docker-build:
	docker build -t gantt-saas:latest -f deploy/Dockerfile .
