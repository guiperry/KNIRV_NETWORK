# asic-driver Makefile

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod

# Binary names
SERVER_BINARY=pixie-server
CLIENT_BINARY=pixie-client
BIN_DIR=bin

# Proto parameters
PROTO_DIR=internal/proto
PROTO_OUT=.

# eBPF parameters
EBPF_SRC=internal/ebpf/pixie.bpf.c
EBPF_OUT=internal/ebpf/pixie.bpf.o
CLANG=clang
LLVM_STRIP=llvm-strip

# Build flags
LDFLAGS=-ldflags "-s -w"
TAGS=-tags netgo

.PHONY: all build clean test proto ebpf deps help

all: deps proto ebpf build

## help: Show this help message
help:
	@echo 'Usage:'
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' | sed -e 's/^/ /'

## build: Build server and client binaries
build: build-server build-client

## build-server: Build server binary
build-server:
	@echo "Building server..."
	@mkdir -p $(BIN_DIR)
	$(GOBUILD) $(LDFLAGS) $(TAGS) -o $(BIN_DIR)/$(SERVER_BINARY) ./cmd/pixie-server

## build-client: Build client binary
build-client:
	@echo "Building client..."
	@mkdir -p $(BIN_DIR)
	$(GOBUILD) $(LDFLAGS) $(TAGS) -o $(BIN_DIR)/$(CLIENT_BINARY) ./cmd/pixie-client

## proto: Generate Go code from Protocol Buffers
proto:
	@echo "Generating protobuf code..."
	@mkdir -p $(PROTO_OUT)
	protoc --go_out=$(PROTO_OUT) --go_opt=paths=source_relative \
		--go-grpc_out=$(PROTO_OUT) --go-grpc_opt=paths=source_relative \
		$(PROTO_DIR)/pixie/v1/*.proto

## ebpf: Compile eBPF programs
ebpf: $(EBPF_OUT)

$(EBPF_OUT): $(EBPF_SRC)
	@echo "Compiling eBPF program..."
	$(CLANG) -g -O2 -target bpf -D__TARGET_ARCH_x86 \
		-I/usr/include/bpf \
		-c $(EBPF_SRC) -o $(EBPF_OUT)
	$(LLVM_STRIP) -g $(EBPF_OUT)

## generate: Generate all code (proto, ebpf, go generate)
generate: proto ebpf
	@echo "Running go generate..."
	$(GOCMD) generate ./...

## deps: Install dependencies
deps:
	@echo "Installing dependencies..."
	$(GOMOD) download
	$(GOMOD) tidy

## test: Run tests
test:
	@echo "Running tests..."
	$(GOTEST) -v -race -coverprofile=coverage.txt -covermode=atomic ./...

## clean: Clean build artifacts
clean:
	@echo "Cleaning..."
	$(GOCLEAN)
	rm -rf $(BIN_DIR)
	rm -f $(EBPF_OUT)
	rm -f coverage.txt

## run-server: Run the server
run-server: build-server
	@echo "Starting server..."
	sudo $(BIN_DIR)/$(SERVER_BINARY)

## run-client: Run the client
run-client: build-client
	@echo "Starting client..."
	$(BIN_DIR)/$(CLIENT_BINARY)

## docker-build: Build Docker image
docker-build:
	docker build -t pixie:latest .

## install: Install binaries to system
install: build
	@echo "Installing binaries..."
	sudo cp $(BIN_DIR)/$(SERVER_BINARY) /usr/local/bin/
	cp $(BIN_DIR)/$(CLIENT_BINARY) /usr/local/bin/

## lint: Run linters
lint:
	@echo "Running linters..."
	golangci-lint run ./...

.DEFAULT_GOAL := help
