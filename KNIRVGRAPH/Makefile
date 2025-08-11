.PHONY: build clean test run docker-build docker-run

BINARY_NAME=graphchain-node
CLI_BINARY=graphchain-cli
BUILD_DIR=build
DOCKER_IMAGE=graphchain-app

build:
	go build -o $(BUILD_DIR)/$(BINARY_NAME) cmd/node/main.go
	go build -o $(BUILD_DIR)/$(CLI_BINARY) cmd/cli/main.go

clean:
	rm -rf $(BUILD_DIR)

test:
	go test -v ./...

# Comprehensive testing targets
test-unit:
	@echo "Running unit tests..."
	go test -timeout 30s -v ./internal/...
	go test -timeout 30s -v ./cmd/...

test-coverage:
	@echo "Running tests with coverage..."
	@mkdir -p coverage
	go test -timeout 30s -coverprofile=coverage/coverage.out -covermode=atomic ./...
	go tool cover -html=coverage/coverage.out -o coverage/coverage.html
	go tool cover -func=coverage/coverage.out
	@echo "Coverage report generated: coverage/coverage.html"

test-race:
	@echo "Running tests with race detection..."
	go test -timeout 30s -race -v ./...

test-bench:
	@echo "Running benchmark tests..."
	go test -timeout 30s -bench=. -benchtime=10s -benchmem ./...

test-storage:
	@echo "Testing storage implementations..."
	go test -v ./internal/storage/...

test-network:
	@echo "Testing network components..."
	go test -v ./internal/network/...

test-nrv:
	@echo "Testing NRV system..."
	go test -v ./internal/nrv/...

test-graphchain:
	@echo "Testing GraphChain..."
	go test -v ./internal/graphchain/...

test-types:
	@echo "Testing type definitions..."
	go test -v ./internal/types/...

test-clean:
	@echo "Cleaning test cache..."
	go clean -testcache

test-all: test-unit test-coverage test-race test-bench
	@echo "All tests completed successfully"

run:
	go run cmd/node/main.go

run-cli:
	go run cmd/cli/main.go

install:
	go install cmd/node/main.go
	go install cmd/cli/main.go

deps:
	go mod download
	go mod tidy

fmt:
	go fmt ./...

lint:
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run ./...; \
	else \
		echo "golangci-lint not found, installing..."; \
		go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest; \
		golangci-lint run ./...; \
	fi

vet:
	@echo "Running go vet..."
	go vet ./...

security:
	@echo "Running security checks..."
	@if command -v gosec >/dev/null 2>&1; then \
		gosec ./...; \
	else \
		echo "gosec not found, installing..."; \
		go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest; \
		gosec ./...; \
	fi

quality: fmt vet lint security
	@echo "Code quality checks completed"

generate:
	go generate ./...

init-testnet:
	./scripts/setup.sh

# Frontend commands
frontend-install:
	npm install

frontend-dev:
	npm run dev

frontend-build:
	npm run build

# Full stack development
dev: build frontend-dev

# Production build
prod: build frontend-build
# Graph-specific commands
create-genesis-graph:
	./build/$(CLI_BINARY) create-node --id genesis --weight 1.0

test-graph:
	./build/$(CLI_BINARY) create-node --id node1 --parents genesis --weight 0.8
	./build/$(CLI_BINARY) create-node --id node2 --parents genesis --weight 0.9
	./build/$(CLI_BINARY) create-edge --from node1 --to node2 --weight 0.5
	./build/$(CLI_BINARY) path node1 node2