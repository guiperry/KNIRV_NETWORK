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
	golangci-lint run

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