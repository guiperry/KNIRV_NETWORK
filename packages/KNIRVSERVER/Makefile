# KNIRV-SERVER Unified Build System
# This Makefile orchestrates the complete build, test, and deployment pipeline

# Project Configuration
PROJECT_NAME := knirv-server
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# Directories
ROOT_DIR := $(shell pwd)
SERVER_WORKSPACE_DIR := $(shell if [ -n "$$KNIRVSERVER_WORKSPACE_DIR" ] && [ -d "$$KNIRVSERVER_WORKSPACE_DIR" ]; then printf '%s' "$$KNIRVSERVER_WORKSPACE_DIR"; elif [ -d "$(ROOT_DIR)/../../../KNIRV_CORP/packages/server/backend_server" ]; then printf '%s' "$(ROOT_DIR)/../../../KNIRV_CORP/packages/server/backend_server"; else printf '%s' "$(ROOT_DIR)"; fi)
SERVER_BACKEND_DIR := $(shell if [ -n "$$KNIRVSERVER_BACKEND_DIR" ] && [ -d "$$KNIRVSERVER_BACKEND_DIR" ]; then printf '%s' "$$KNIRVSERVER_BACKEND_DIR"; elif [ -d "$(SERVER_WORKSPACE_DIR)/backend" ]; then printf '%s' "$(SERVER_WORKSPACE_DIR)/backend"; else printf '%s' "$(SERVER_WORKSPACE_DIR)"; fi)
BACKEND_DIR := $(SERVER_BACKEND_DIR)
FRONTEND_DIR := $(ROOT_DIR)/frontend
DESKTOP_DIR := $(ROOT_DIR)/desktop
GATEWAY_DIR := $(ROOT_DIR)/pkg/knirvgateway
GATEWAY_SRC_DIR := $(ROOT_DIR)/../KNIRVGATEWAY
GATEWAY_PKG_BIN := $(GATEWAY_DIR)/bin/knirvgateway
DVE_VERIFIER_DIR := $(ROOT_DIR)/tools/dve-verifier-wasm
DVE_VERIFIER_ASSETS := $(GATEWAY_SRC_DIR)/internal/dveviewer/assets
GRAPH_DIR := $(ROOT_DIR)/pkg/knirvgraph
GRAPH_SRC_DIR := $(ROOT_DIR)/../KNIRVGRAPH
GRAPH_PKG_BIN := $(GRAPH_DIR)/bin/knirvgraph
CHAIN_DIR := $(ROOT_DIR)/pkg/knirvchain
CHAIN_SRC_DIR := $(ROOT_DIR)/../KNIRVCHAIN
CHAIN_PKG_BIN := $(CHAIN_DIR)/bin/knirvchain
MONITOR_DIR := $(ROOT_DIR)/pkg/knirvmonitor
MONITOR_SRC_DIR := $(ROOT_DIR)/../KNIRVMONITOR
MONITOR_PKG_BIN := $(MONITOR_DIR)/bin/monitor
ORACLE_DIR := $(ROOT_DIR)/../KNIRVORACLE
ORACLE_PKG_DIR := $(ROOT_DIR)/pkg/knirvoracle
ORACLE_PKG_BIN := $(ORACLE_PKG_DIR)/bin/knirvoracle
HASHER_DIR := $(ROOT_DIR)/pkg/knirvhasher
HASHER_SRC_DIR := $(ROOT_DIR)/../KNIRVHASHER
HASHER_PKG_BIN := $(HASHER_DIR)/bin/knirvhasher
CONNECTOR_SRC_DIR := $(ROOT_DIR)/../KNIRVHASHER/pipeline/0_DATA_CONNECTOR
CONNECTOR_PKG_BIN := $(HASHER_DIR)/internal/cli/embedded/bin/data-connector
MINER_SRC_DIR := $(ROOT_DIR)/../KNIRVHASHER/pipeline/1_DATA_MAPPER
MINER_PKG_BIN := $(HASHER_DIR)/internal/cli/embedded/bin/data-mapper
ENCODER_SRC_DIR := $(ROOT_DIR)/../KNIRVHASHER/pipeline/2_DATA_ENCODER
ENCODER_PKG_BIN := $(HASHER_DIR)/internal/cli/embedded/bin/data-encoder
TRAINER_SRC_DIR := $(ROOT_DIR)/../KNIRVHASHER/pipeline/3_DATA_TRAINER
TRAINER_PKG_BIN := $(HASHER_DIR)/internal/cli/embedded/bin/data-trainer
AGENT_DIR := $(ROOT_DIR)/pkg/knirvagent
AGENT_SRC_DIR := $(ROOT_DIR)/../KNIRVAGENT
AGENT_PKG_BIN := $(AGENT_DIR)/bin/knirvagent
ARENA_DIR := $(ROOT_DIR)/pkg/knirvarena
ARENA_SRC_DIR := $(ROOT_DIR)/../KNIRVARENA
ARENA_PKG_BUNDLE := $(ARENA_DIR)/bin/knirvarena.tar.gz
SCRIPTS_DIR := $(shell if [ -d "$(SERVER_WORKSPACE_DIR)/scripts" ]; then printf '%s' "$(SERVER_WORKSPACE_DIR)/scripts"; else printf '%s' "$(ROOT_DIR)/scripts"; fi)
BUILD_DIR := $(ROOT_DIR)/build
DIST_DIR := $(ROOT_DIR)/dist
DEPLOYMENT_DIR := $(ROOT_DIR)/deployment
APPDATA_DIR := $(HOME)/var/lib/knirvserver
CONTAINER_DEPLOYER_SOURCE_DIR := $(HOME)/.local/share/knirvserver/container_deployer/resources/golang-app-source

# Embedded Programs Configuration
EMBEDDED_DIR := $(BACKEND_DIR)/internal/embedded
VALIDATION_CHAIN_DIR := $(EMBEDDED_DIR)/validation_chain
TRANSACTION_CHAIN_DIR := $(EMBEDDED_DIR)/transaction_chain
EMBEDDED_BUILD_DIR := $(ROOT_DIR)/build/embedded

# Go Configuration
GO_VERSION := 1.21
GOOS := $(shell go env GOOS)
GOARCH := $(shell go env GOARCH)
HOST_ARCH := $(shell uname -m)
CGO_ENABLED := 1
export GOFLAGS := -buildvcs=false

# eBPF C header configuration
# bpf2go needs the arch-specific system header path on Debian/Ubuntu hosts
# so clang can resolve <asm/types.h>.
ifeq ($(HOST_ARCH),x86_64)
BPF2GO_CFLAGS := -I/usr/include/x86_64-linux-gnu
else ifeq ($(HOST_ARCH),aarch64)
BPF2GO_CFLAGS := -I/usr/include/aarch64-linux-gnu
else ifeq ($(HOST_ARCH),arm64)
BPF2GO_CFLAGS := -I/usr/include/aarch64-linux-gnu
else ifeq ($(HOST_ARCH),armv7l)
BPF2GO_CFLAGS := -I/usr/include/arm-linux-gnueabihf
else ifeq ($(HOST_ARCH),armv7)
BPF2GO_CFLAGS := -I/usr/include/arm-linux-gnueabihf
else ifeq ($(HOST_ARCH),riscv64)
BPF2GO_CFLAGS := -I/usr/include/riscv64-linux-gnu
endif
export BPF2GO_CFLAGS

# Build flags
LDFLAGS := -X main.Version=$(VERSION) \
           -X main.BuildTime=$(BUILD_TIME) \
           -X main.GitCommit=$(GIT_COMMIT) \
           -w -s

# Node.js Configuration
NODE_VERSION := 18
NPM_CMD := npm

# Docker Configuration
DOCKER_IMAGE := knirv-server
DOCKER_TAG := $(VERSION)

# AWS Configuration
AWS_REGION := us-east-1
ENVIRONMENT := production

# Colors for output
RED := \033[0;31m
GREEN := \033[0;32m
YELLOW := \033[1;33m
BLUE := \033[0;34m
NC := \033[0m # No Color

.PHONY: help all clean build test deploy install deps frontend backend binary relink docker os deployer key-encryptor deploy-dev-local ebpf-generate proto kali-docker-image gateway-build gateway-deps gateway-clean dve-verifier-build graph-build graph-deps graph-clean chain-build chain-deps chain-clean oracle-build oracle-deps oracle-clean hasher-build hasher-deps hasher-clean agent-build agent-deps agent-clean arena-build arena-deps arena-clean build-full

# Default target
all: proto frontend backend binary


help: ## Show this help message
	@echo "$(BLUE)KNIRV-SERVER Build System$(NC)"
	@echo ""
	@echo "$(YELLOW)Available targets:$(NC)"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  $(GREEN)%-20s$(NC) %s\n", $$1, $$2}' $(MAKEFILE_LIST)

# eBPF Generation
ebpf-generate: ## Generate eBPF Go code from C programs
	@echo "Generating eBPF programs..."
	cd $(BACKEND_DIR)/internal/ebpf && go generate
	@echo "eBPF generation completed"

# ProtoBuf Generation
proto: ## Generate protobuf Go code from .proto files
	@echo "Generating protobuf files..."
	cd $(BACKEND_DIR)/internal/proto && go generate
	@echo "Protobuf generation completed"

# Dependency Management
deps: deps-go deps-node deps-ansible ## Install all dependencies

deps-go: arena-deps ## Install Go dependencies (backend and standalone submodules)
	@echo "$(BLUE)Installing Go dependencies...$(NC)"
	cd $(BACKEND_DIR) && go mod download
	cd $(BACKEND_DIR) && go mod tidy
	cd $(GATEWAY_SRC_DIR) && go mod download
	cd $(GATEWAY_SRC_DIR) && go mod tidy
	cd $(GRAPH_SRC_DIR) && go mod download
	cd $(GRAPH_SRC_DIR) && go mod tidy
	cd $(CHAIN_SRC_DIR) && go mod download
	cd $(CHAIN_SRC_DIR) && go mod tidy
	cd $(ORACLE_DIR) && go mod download
	cd $(ORACLE_DIR) && go mod tidy
	cd $(CONNECTOR_SRC_DIR) && go mod download
	cd $(CONNECTOR_SRC_DIR) && go mod tidy
	cd $(MINER_SRC_DIR) && go mod download
	cd $(MINER_SRC_DIR) && go mod tidy
	cd $(ENCODER_SRC_DIR) && go mod download
	cd $(ENCODER_SRC_DIR) && go mod tidy
	cd $(TRAINER_SRC_DIR) && go mod download
	cd $(TRAINER_SRC_DIR) && go mod tidy
	cd $(HASHER_SRC_DIR) && go mod download
	cd $(HASHER_SRC_DIR) && go mod tidy
	cd $(AGENT_SRC_DIR) && go mod download
	cd $(AGENT_SRC_DIR) && go mod tidy

deps-node: ## Install Node.js dependencies
	@echo "$(BLUE)Installing Node.js dependencies...$(NC)"
	cd $(FRONTEND_DIR) && $(NPM_CMD) install

deps-ansible: ## Install Ansible dependencies
	@echo "$(BLUE)Installing Ansible dependencies...$(NC)"
	-cd $(DEPLOYMENT_DIR)/ansible && ansible-galaxy collection install -r requirements.yml 2>/dev/null || true

# Frontend Build
frontend: deps-node ## Build Next.js frontend
	@echo "$(BLUE)Building Next.js frontend...$(NC)"
	cd $(FRONTEND_DIR) && $(NPM_CMD) run build
	@echo "$(GREEN)Frontend build completed$(NC)"

frontend-dev: deps-node ## Start frontend development server
	@echo "$(BLUE)Starting frontend development server...$(NC)"
	cd $(FRONTEND_DIR) && $(NPM_CMD) run dev

# Backend Build (cmd/backend_server only)
backend: deps-go gateway-build graph-build chain-build monitor-build oracle-build hasher-build agent-build arena-build embedded-build ## Build complete Go backend service package
	@echo "$(BLUE)Building complete backend package binary...$(NC)"
	@rm -f $(BACKEND_DIR)/backend_server bin/backend_server
	cd $(BACKEND_DIR) && go build -ldflags "$(LDFLAGS)" -o backend_server ./cmd/backend_server
	@echo "$(BLUE)Copying backend binary to root...$(NC)"
	cp $(BACKEND_DIR)/backend_server bin/
	@echo "$(BLUE)Compressing backend binary with gzip...$(NC)"
	@rm -f bin/backend_server.tmp
	@gzip -cf bin/backend_server > bin/backend_server.tmp && mv bin/backend_server.tmp bin/backend_server
	@echo "$(GREEN)Backend build completed$(NC)"
	rm $(BACKEND_DIR)/backend_server

# Specialized Build Tools
os: deps-go ## Build os_builder binary
	@echo "$(BLUE)Building os_builder binary...$(NC)"
	@rm -f $(BACKEND_DIR)/cmd/os_builder/os_builder
	cd $(BACKEND_DIR)/cmd/os_builder && go build -ldflags "$(LDFLAGS)" -o os_builder main.go
	@echo "$(GREEN)os_builder build completed: backend_server/cmd/os_builder/os_builder$(NC)"
deployer: deps-go ## Build container_deployer binary
	@echo "$(BLUE)Building container_deployer binary...$(NC)"
	@rm -f $(BACKEND_DIR)/cmd/container_deployer/container_deployer
	cd $(BACKEND_DIR)/cmd/container_deployer && go build -ldflags "$(LDFLAGS)" -o container_deployer main.go
	@echo "$(GREEN)container_deployer build completed: backend_server/cmd/container_deployer/container_deployer$(NC)"

key-encryptor: ## Build root_key_encryptor binary
	@echo "$(BLUE)Building root_key_encryptor binary...$(NC)"
	@rm -f $(BACKEND_DIR)/root_key_encryptor bin/root_key_encryptor
	cd $(BACKEND_DIR) && CGO_ENABLED=1 go build -ldflags "$(LDFLAGS)" -o root_key_encryptor ./cmd/root_key_encryptor
	@echo "$(BLUE)Copying root_key_encryptor binary to root...$(NC)"
	@rm -f bin/root_key_encryptor
	cp $(BACKEND_DIR)/root_key_encryptor bin/
	@echo "$(GREEN)root_key_encryptor build completed$(NC)"
	rm $(BACKEND_DIR)/root_key_encryptor


# Unified Binary (Frontend + Desktop + Backend + embedded pkg modules)
# main.go embeds: bin/backend_server, config/*, desktop/dist/*, frontend/out/*
# Module-owned binaries like KNIRVGATEWAY, KNIRVGRAPH, KNIRVCHAIN, KNIRVORACLE, KNIRVHASHER, KNIRVMONITOR, and KNIRVAGENT
# are embedded and extracted by their pkg modules.
binary: proto ebpf-generate frontend backend arena-build ## Build unified binary embedding frontend, desktop, backend, KNIRVGATEWAY, KNIRVGRAPH, KNIRVCHAIN, KNIRVORACLE, KNIRVHASHER, KNIRVMONITOR, KNIRVAGENT, and KNIRVARENA
	@echo "$(BLUE)Creating unified binary (embeds frontend, desktop, backend, KNIRVGATEWAY, KNIRVGRAPH, KNIRVCHAIN, KNIRVORACLE, KNIRVHASHER, KNIRVMONITOR)...$(NC)"
	@mkdir -p $(DIST_DIR)
	@rm -f $(DIST_DIR)/knirv-server
	go build -ldflags "$(LDFLAGS)" -o $(DIST_DIR)/knirv-server .
	chmod 755 $(DIST_DIR)/knirv-server
	@echo "$(GREEN)Unified binary created: $(DIST_DIR)/knirv-server$(NC)"
	@echo "$(BLUE)Copying binary to container_deployer golang-app-source...$(NC)"
	@mkdir -p $(CONTAINER_DEPLOYER_SOURCE_DIR)
	@rm -f $(CONTAINER_DEPLOYER_SOURCE_DIR)/knirv-server
	cp $(DIST_DIR)/knirv-server $(CONTAINER_DEPLOYER_SOURCE_DIR)
	chmod +x $(CONTAINER_DEPLOYER_SOURCE_DIR)/knirv-server
	@echo "$(GREEN)Binary copied to $(CONTAINER_DEPLOYER_SOURCE_DIR)$(NC)"
	@echo "$(GREEN)Good Build!$(NC)"

# Quick Binary Relink (fast path, assumes all components already built)
# Use this after making changes to frontend, desktop, backend, or embedded binaries to quickly relink without rebuilding everything
relink: ## Quickly relink the unified binary (assumes all components already built)
	@echo "$(BLUE)Relinking unified binary (fast path)...$(NC)"
	@mkdir -p $(DIST_DIR)
	@rm -f $(DIST_DIR)/knirv-server
	go build -ldflags "$(LDFLAGS)" -o $(DIST_DIR)/knirv-server .
	chmod 755 $(DIST_DIR)/knirv-server
	@echo "$(BLUE)Copying binary to container_deployer golang-app-source...$(NC)"
	@mkdir -p $(CONTAINER_DEPLOYER_SOURCE_DIR)
	@rm -f $(CONTAINER_DEPLOYER_SOURCE_DIR)/knirv-server
	cp $(DIST_DIR)/knirv-server $(CONTAINER_DEPLOYER_SOURCE_DIR)
	chmod +x $(CONTAINER_DEPLOYER_SOURCE_DIR)/knirv-server
	@echo "$(GREEN)Done — binary relinked: $(DIST_DIR)/knirv-server$(NC)"

# Testing
test: test-go test-frontend ## Run standard tests (unit + frontend)

test-all: test-go test-frontend test-integration test-e2e test-architecture ## Run all non-privileged tests

test-complete: test-all test-privileged ## Run complete test suite including privileged tests

test-go: deps-go ## Run Go unit tests
	@echo "$(BLUE)Running Go unit tests...$(NC)"
	@KNIRVSERVER_WORKSPACE_DIR=$(SERVER_WORKSPACE_DIR) KNIRVSERVER_BACKEND_DIR=$(BACKEND_DIR) $(SCRIPTS_DIR)/run-tests.sh unit

test-frontend: deps-node ## Run frontend tests
	@echo "$(BLUE)Running frontend tests...$(NC)"
	cd $(FRONTEND_DIR) && $(NPM_CMD) test

test-integration: deps-go ## Run integration tests
	@echo "$(BLUE)Running integration tests...$(NC)"
	@KNIRVSERVER_WORKSPACE_DIR=$(SERVER_WORKSPACE_DIR) KNIRVSERVER_BACKEND_DIR=$(BACKEND_DIR) $(SCRIPTS_DIR)/run-tests.sh integration

test-e2e: deps-go ## Run end-to-end tests
	@echo "$(BLUE)Running end-to-end tests...$(NC)"
	@KNIRVSERVER_WORKSPACE_DIR=$(SERVER_WORKSPACE_DIR) KNIRVSERVER_BACKEND_DIR=$(BACKEND_DIR) $(SCRIPTS_DIR)/run-tests.sh e2e

test-architecture: deps-go ## Test the new three-tier architecture
	@echo "$(BLUE)Testing architecture...$(NC)"
	@KNIRVSERVER_WORKSPACE_DIR=$(SERVER_WORKSPACE_DIR) KNIRVSERVER_BACKEND_DIR=$(BACKEND_DIR) $(SCRIPTS_DIR)/run-tests.sh architecture

test-privileged: ## Run privileged tests in Docker container (cgroups, namespaces, etc.)
	@echo "$(BLUE)Running privileged tests in Docker container...$(NC)"
	@KNIRVSERVER_WORKSPACE_DIR=$(SERVER_WORKSPACE_DIR) KNIRVSERVER_BACKEND_DIR=$(BACKEND_DIR) $(SCRIPTS_DIR)/test-server-privileged.sh $(if $(SHOW_BUILD),--show-build)
	@echo "$(GREEN)Privileged tests completed$(NC)"

test-privileged-quick: ## Run privileged tests using existing container
	@echo "$(BLUE)Running quick privileged tests...$(NC)"
	@KNIRVSERVER_WORKSPACE_DIR=$(SERVER_WORKSPACE_DIR) KNIRVSERVER_BACKEND_DIR=$(BACKEND_DIR) $(SCRIPTS_DIR)/test-server-privileged.sh --no-deploy $(if $(SHOW_BUILD),--show-build)
	@echo "$(GREEN)Quick privileged tests completed$(NC)"

test-privileged-full: ## Run full privileged tests with cleanup
	@echo "$(BLUE)Running full privileged tests with cleanup...$(NC)"
	@KNIRVSERVER_WORKSPACE_DIR=$(SERVER_WORKSPACE_DIR) KNIRVSERVER_BACKEND_DIR=$(BACKEND_DIR) $(SCRIPTS_DIR)/test-server-privileged.sh --cleanup $(if $(SHOW_BUILD),--show-build)
	@echo "$(GREEN)Full privileged tests completed$(NC)"

# Linting and Formatting
lint: lint-go lint-frontend ## Run all linters

lint-go: ## Run Go linters
	@echo "$(BLUE)Running Go linters...$(NC)"
	cd $(BACKEND_DIR) && golangci-lint run

lint-frontend: ## Run frontend linters
	@echo "$(BLUE)Running frontend linters...$(NC)"
	cd $(FRONTEND_DIR) && $(NPM_CMD) run lint

format: format-go format-frontend ## Format all code

format-go: ## Format Go code
	@echo "$(BLUE)Formatting Go code...$(NC)"
	cd $(BACKEND_DIR) && go fmt ./...
	cd $(BACKEND_DIR) && goimports -w .

format-frontend: ## Format frontend code
	@echo "$(BLUE)Formatting frontend code...$(NC)"
	cd $(FRONTEND_DIR) && $(NPM_CMD) run format

# Development
dev: ## Start development environment
	@echo "$(BLUE)Starting development environment...$(NC)"
	$(MAKE) -j2 dev-backend dev-frontend

dev-backend: backend ## Start unified backend development server
	@echo "$(BLUE)Starting unified backend development server...$(NC)"
	cd $(BACKEND_DIR) && ./bin/backend_server --config ../config/development.yaml

dev-frontend: ## Start frontend development server
	cd $(FRONTEND_DIR) && $(NPM_CMD) run dev

# Docker
docker: ## Build Docker image
	@echo "$(BLUE)Building Docker image...$(NC)"
	docker build -t $(DOCKER_IMAGE):$(DOCKER_TAG) .
	docker tag $(DOCKER_IMAGE):$(DOCKER_TAG) $(DOCKER_IMAGE):latest

docker-run: docker ## Run Docker container
	@echo "$(BLUE)Running Docker container...$(NC)"
	docker run -p 8080:8080 $(DOCKER_IMAGE):$(DOCKER_TAG)

# Docker Build
kali-docker-image: ## Build Kali Docker image using os_builder
	@echo "$(BLUE)Building Kali Docker image using os_builder...$(NC)"
	go run $(BACKEND_DIR)/cmd/os_builder/main.go --action 1
	@echo "$(GREEN)Kali Docker image build completed and saved to artifacts.$(NC)"

# Packer Builds
kali: ## Build Kali Linux base image with Packer
	@echo "$(BLUE)Building Kali Linux base image with Packer...$(NC)"
	cd $(BACKEND_DIR)/cmd/os_builder/packer-base-kali && packer build .
	@echo "$(GREEN)Kali Linux base image build completed.$(NC)"

# Deployment
deploy-dev-local: ## Deploy to local development environment
	@echo "$(BLUE)Deploying to local development environment...$(NC)"
	cd $(BACKEND_DIR)/cmd/container_deployer && ./container_deployer --deploy-type local --env development --action 1

# Utilities
clean-all: ## Clean all build artifacts (binaries, dist, node_modules, Go cache)
	@echo "$(BLUE)Cleaning build artifacts...$(NC)"
	rm -rf $(BUILD_DIR)
	rm -rf $(DIST_DIR)
	rm -rf $(BACKEND_DIR)/bin
	rm -rf $(BACKEND_DIR)/coverage.out
	rm -rf $(BACKEND_DIR)/coverage.html
	rm -rf bin
	rm -rf .next
	rm -rf out
	rm -rf $(FRONTEND_DIR)/.next
	rm -rf $(FRONTEND_DIR)/out
	rm -rf $(FRONTEND_DIR)/node_modules
	rm -rf node_modules/.cache
	cd $(BACKEND_DIR) && go clean -modcache
	cd $(GATEWAY_DIR) && go clean
	-docker stop knirvserver-kali-local 2>/dev/null || true
	-docker rm knirvserver-kali-local 2>/dev/null || true
	-docker system prune -f 2>/dev/null || true

clean: ## Clean build outputs (preserves node_modules and Go module cache)
	@echo "$(BLUE)Cleaning build outputs...$(NC)"
	rm -rf $(BUILD_DIR) $(DIST_DIR)
	rm -rf $(BACKEND_DIR)/bin
	rm -f bin/backend_server
	rm -f $(GATEWAY_PKG_BIN) $(GRAPH_PKG_BIN) $(CHAIN_PKG_BIN) $(ORACLE_PKG_BIN) $(HASHER_PKG_BIN) $(AGENT_PKG_BIN) $(ARENA_PKG_BUNDLE)
	rm -rf $(FRONTEND_DIR)/.next $(FRONTEND_DIR)/out
	cd $(GATEWAY_DIR) && go clean
	cd $(HASHER_DIR) && go clean
	cd $(AGENT_DIR) && go clean
	rm -rf $(ARENA_DIR)/bin

install: binary ## Install binary to system
	@echo "$(BLUE)Installing binary...$(NC)"
	sudo cp $(DIST_DIR)/knirv-server /usr/local/bin/
	sudo chmod +x /usr/local/bin/knirv-server

uninstall: ## Uninstall binary from system
	@echo "$(BLUE)Uninstalling binary...$(NC)"
	sudo rm -f /usr/local/bin/knirv-server

# Health checks
health: ## Check system health
	@echo "$(BLUE)Checking system health...$(NC)"
	curl -f http://localhost:8080/api/v1/health || echo "$(RED)Health check failed$(NC)"

status: ## Check deployment status
	@echo "$(BLUE)Checking deployment status...$(NC)"
	cd $(DEPLOYMENT_DIR)/ansible && ansible all -i inventory/aws_ec2.yml -m ping

# Documentation
docs: ## Generate documentation
	@echo "$(BLUE)Generating documentation...$(NC)"
	cd $(BACKEND_DIR) && godoc -http=:6060 &
	@echo "$(GREEN)Documentation server started at http://localhost:6060$(NC)"

# Version information
version: ## Show version information
	@echo "$(BLUE)Version Information:$(NC)"
	@echo "  Project: $(PROJECT_NAME)"
	@echo "  Version: $(VERSION)"
	@echo "  Build Time: $(BUILD_TIME)"
	@echo "  Git Commit: $(GIT_COMMIT)"
	@echo "  Go Version: $(shell go version)"
	@echo "  Node Version: $(shell node --version 2>/dev/null || echo 'Not installed')"

# Environment setup
setup: ## Setup development environment
	@echo "$(BLUE)Setting up development environment...$(NC)"
	@echo "Checking prerequisites..."
	@command -v go >/dev/null 2>&1 || { echo "$(RED)Go is required but not installed$(NC)"; exit 1; }
	@command -v node >/dev/null 2>&1 || { echo "$(RED)Node.js is required but not installed$(NC)"; exit 1; }
	@command -v ansible >/dev/null 2>&1 || { echo "$(RED)Ansible is required but not installed$(NC)"; exit 1; }
	@echo "$(GREEN)Prerequisites check passed$(NC)"
	$(MAKE) deps

# Backup and restore
backup: ## Backup application data
	@echo "$(BLUE)Creating backup...$(NC)"
	mkdir -p backups
	tar -czf backups/server-backup-$(shell date +%Y%m%d-%H%M%S).tar.gz data/ config/

restore: ## Restore from backup (specify BACKUP_FILE)
	@echo "$(BLUE)Restoring from backup...$(NC)"
	@if [ -z "$(BACKUP_FILE)" ]; then echo "$(RED)Please specify BACKUP_FILE$(NC)"; exit 1; fi
	tar -xzf $(BACKUP_FILE)

# Performance testing
perf: binary ## Run performance tests
	@echo "$(BLUE)Running performance tests...$(NC)"
	cd $(BACKEND_DIR) && go test -bench=. -benchmem ./...

# Security scanning
security: ## Run security scans
	@echo "$(BLUE)Running security scans...$(NC)"
	cd $(BACKEND_DIR) && gosec ./...
	$(NPM_CMD) audit

# Quick targets for common workflows
quick-build: deps frontend backend ## Quick build without tests
quick-test: test-go ## Quick test (Go only)
quick-deploy: binary deploy-application ## Quick application deployment

# KNIRVGATEWAY (Embedded) targets
WEBGUI_SRC_DIR := $(GATEWAY_SRC_DIR)/internal/embedded/webgui

webgui-deps: ## Install KNIRVGATEWAY WebGUI Node.js dependencies
	@echo "$(BLUE)Installing KNIRVGATEWAY WebGUI dependencies...$(NC)"
	cd $(WEBGUI_SRC_DIR) && $(NPM_CMD) install --legacy-peer-deps

webgui-build: webgui-deps ## Build KNIRVGATEWAY WebGUI static export
	@echo "$(BLUE)Building KNIRVGATEWAY WebGUI static export...$(NC)"
	cd $(WEBGUI_SRC_DIR) && $(NPM_CMD) run build
	@echo "$(GREEN)KNIRVGATEWAY WebGUI build completed$(NC)"

gateway-deps: ## Install KNIRVGATEWAY dependencies (Go only)
	@echo "$(BLUE)Installing KNIRVGATEWAY Go dependencies...$(NC)"
	cd $(GATEWAY_SRC_DIR) && go mod download && go mod tidy

dve-verifier-build: ## Build the browser-side DVE wasm verifier into KNIRVGATEWAY
	@echo "$(BLUE)Building DVE browser verifier...$(NC)"
	cd $(DVE_VERIFIER_DIR) && wasm-pack build --target web --release --no-pack --no-typescript --out-name verifier --out-dir $(DVE_VERIFIER_ASSETS)
	@echo "$(GREEN)DVE browser verifier built$(NC)"

gateway-build: webgui-build gateway-deps dve-verifier-build ## Build KNIRVGATEWAY binary into pkg/knirvgateway/bin/ (builds WebGUI and DVE verifier first)
	@echo "$(BLUE)Building KNIRVGATEWAY binary...$(NC)"
	@mkdir -p $(GATEWAY_DIR)/bin
	@rm -f $(GATEWAY_PKG_BIN)
	cd $(GATEWAY_SRC_DIR) && go build -ldflags "-s -w" -o $(GATEWAY_PKG_BIN) ./cmd/gateway
	@echo "$(BLUE)Compressing KNIRVGATEWAY binary with gzip...$(NC)"
	@rm -f $(GATEWAY_PKG_BIN).tmp
	@gzip -cf $(GATEWAY_PKG_BIN) > $(GATEWAY_PKG_BIN).tmp && mv $(GATEWAY_PKG_BIN).tmp $(GATEWAY_PKG_BIN)
	@echo "$(GREEN)KNIRVGATEWAY binary built: $(GATEWAY_PKG_BIN)$(NC)"

gateway-clean: ## Clean KNIRVGATEWAY build artifacts
	@echo "$(BLUE)Cleaning KNIRVGATEWAY artifacts...$(NC)"
	rm -f $(GATEWAY_PKG_BIN)
	rm -rf $(WEBGUI_SRC_DIR)/out $(WEBGUI_SRC_DIR)/.next
	cd $(GATEWAY_SRC_DIR) && go clean

# KNIRVGRAPH (Embedded) targets
graph-deps: ## Install KNIRVGRAPH dependencies
	@echo "$(BLUE)Installing KNIRVGRAPH dependencies...$(NC)"
	cd $(GRAPH_SRC_DIR) && go mod download && go mod tidy

graph-build: graph-deps ## Build KNIRVGRAPH binary into pkg/knirvgraph/bin/ (required before binary target)
	@echo "$(BLUE)Building KNIRVGRAPH binary...$(NC)"
	@mkdir -p $(GRAPH_DIR)/bin
	@rm -f $(GRAPH_PKG_BIN)
	cd $(GRAPH_SRC_DIR)/cmd/node && go build -ldflags "-s -w" -o $(GRAPH_PKG_BIN)
	@echo "$(BLUE)Compressing KNIRVGRAPH binary with gzip...$(NC)"
	@rm -f $(GRAPH_PKG_BIN).tmp
	@gzip -cf $(GRAPH_PKG_BIN) > $(GRAPH_PKG_BIN).tmp && mv $(GRAPH_PKG_BIN).tmp $(GRAPH_PKG_BIN)
	@echo "$(GREEN)KNIRVGRAPH binary built: $(GRAPH_PKG_BIN)$(NC)"

graph-clean: ## Clean KNIRVGRAPH build artifacts
	@echo "$(BLUE)Cleaning KNIRVGRAPH artifacts...$(NC)"
	rm -f $(GRAPH_PKG_BIN)
	cd $(GRAPH_SRC_DIR) && go clean

# KNIRVCHAIN (Embedded) targets
chain-deps: ## Install KNIRVCHAIN dependencies
	@echo "$(BLUE)Installing KNIRVCHAIN dependencies...$(NC)"
	cd $(CHAIN_SRC_DIR) && go mod download && go mod tidy

chain-build: chain-deps ## Build KNIRVCHAIN binary into pkg/knirvchain/bin/ (required before binary target)
	@echo "$(BLUE)Building KNIRVCHAIN binary...$(NC)"
	@mkdir -p $(CHAIN_DIR)/bin
	@rm -f $(CHAIN_PKG_BIN)
	cd $(CHAIN_SRC_DIR)/cmd/knirvchain && go build -ldflags "-s -w" -o $(CHAIN_PKG_BIN)
	@echo "$(BLUE)Compressing KNIRVCHAIN binary with gzip...$(NC)"
	@rm -f $(CHAIN_PKG_BIN).tmp
	@gzip -cf $(CHAIN_PKG_BIN) > $(CHAIN_PKG_BIN).tmp && mv $(CHAIN_PKG_BIN).tmp $(CHAIN_PKG_BIN)
	@echo "$(GREEN)KNIRVCHAIN binary built: $(CHAIN_PKG_BIN)$(NC)"

chain-clean: ## Clean KNIRVCHAIN build artifacts
	@echo "$(BLUE)Cleaning KNIRVCHAIN artifacts...$(NC)"
	rm -f $(CHAIN_PKG_BIN)
	cd $(CHAIN_SRC_DIR) && go clean

# KNIRVMONITOR (Embedded) targets
monitor-deps: ## Install KNIRVMONITOR dependencies
	@echo "$(BLUE)Installing KNIRVMONITOR dependencies...$(NC)"
	cd $(MONITOR_SRC_DIR) && go mod download && go mod tidy

monitor-build: monitor-deps ## Build KNIRVMONITOR binary into pkg/knirvmonitor/bin/ (required before binary target)
	@echo "$(BLUE)Building KNIRVMONITOR binary...$(NC)"
	@mkdir -p $(MONITOR_DIR)/bin
	@rm -f $(MONITOR_PKG_BIN)
	cd $(MONITOR_SRC_DIR)/cmd/server && go build -ldflags "-s -w" -o $(MONITOR_PKG_BIN)
	@echo "$(BLUE)Compressing KNIRVMONITOR binary with gzip...$(NC)"
	@rm -f $(MONITOR_PKG_BIN).tmp
	@gzip -cf $(MONITOR_PKG_BIN) > $(MONITOR_PKG_BIN).tmp && mv $(MONITOR_PKG_BIN).tmp $(MONITOR_PKG_BIN)
	@echo "$(GREEN)KNIRVMONITOR binary built: $(MONITOR_PKG_BIN)$(NC)"

monitor-clean: ## Clean KNIRVMONITOR build artifacts
	@echo "$(BLUE)Cleaning KNIRVMONITOR artifacts...$(NC)"
	rm -f $(MONITOR_PKG_BIN)
	cd $(MONITOR_SRC_DIR) && go clean

# KNIRVORACLE (Embedded) targets
oracle-deps: ## Install KNIRVORACLE dependencies
	@echo "$(BLUE)Installing KNIRVORACLE dependencies...$(NC)"
	cd $(ORACLE_DIR) && go mod download && go mod tidy

oracle-build: oracle-deps ## Build KNIRVORACLE binary into pkg/knirvoracle/bin/ (required before binary target)
	@echo "$(BLUE)Building KNIRVORACLE binary...$(NC)"
	@mkdir -p $(ORACLE_PKG_DIR)/bin
	@rm -f $(ORACLE_PKG_BIN)
	cd $(ORACLE_DIR)/cmd/oracle && go build -ldflags "-s -w" -o $(ORACLE_PKG_BIN)
	@echo "$(BLUE)Compressing KNIRVORACLE binary with gzip...$(NC)"
	@rm -f $(ORACLE_PKG_BIN).tmp
	@gzip -cf $(ORACLE_PKG_BIN) > $(ORACLE_PKG_BIN).tmp && mv $(ORACLE_PKG_BIN).tmp $(ORACLE_PKG_BIN)
	@echo "$(GREEN)KNIRVORACLE binary built: $(ORACLE_PKG_BIN)$(NC)"

oracle-clean: ## Clean KNIRVORACLE build artifacts
	@echo "$(BLUE)Cleaning KNIRVORACLE artifacts...$(NC)"
	rm -f $(ORACLE_PKG_BIN)
	cd $(ORACLE_DIR) && go clean

# Full build including embedded KNIRVGATEWAY (gateway-build is now a dep of binary)
build-full: binary ## Build everything including KNIRVGATEWAY (alias for binary)

# KNIRVHASHER (Embedded) targets
hasher-deps: ## Install KNIRVHASHER dependencies
	@echo "$(BLUE)Installing KNIRVHASHER dependencies...$(NC)"
	cd $(HASHER_SRC_DIR) && go mod download && go mod tidy

hasher-build: hasher-deps ## Build KNIRVHASHER binaries into pkg/knirvhasher/bin/ (required before binary target)
	@echo "$(BLUE)Building KNIRVHASHER pipeline binaries...$(NC)"
	cd $(HASHER_SRC_DIR) && $(MAKE) build-pipeline-all
	@mkdir -p $(HASHER_DIR)/bin
	@rm -f $(HASHER_PKG_BIN)
	cd $(HASHER_SRC_DIR) && go build -ldflags "-s -w" -o $(HASHER_PKG_BIN) ./cmd/cli
	@echo "$(BLUE)Compressing KNIRVHASHER binary with gzip...$(NC)"
	@rm -f $(HASHER_PKG_BIN).tmp
	@gzip -cf $(HASHER_PKG_BIN) > $(HASHER_PKG_BIN).tmp && mv $(HASHER_PKG_BIN).tmp $(HASHER_PKG_BIN)
	@echo "$(GREEN)KNIRVHASHER binaries built: $(HASHER_PKG_BIN) and pipeline artifacts$(NC)"

hasher-clean: ## Clean KNIRVHASHER build artifacts
	@echo "$(BLUE)Cleaning KNIRVHASHER artifacts...$(NC)"
	rm -f $(HASHER_PKG_BIN)
	cd $(HASHER_SRC_DIR) && go clean

# Internal Embedded Programs - Transaction Chain (Node.js)
transaction-chain-deps: ## Install transaction_chain dependencies
	@echo "$(BLUE)Installing transaction_chain dependencies...$(NC)"
	cd $(TRANSACTION_CHAIN_DIR) && $(NPM_CMD) ci --production

transaction-chain-build: ## Bundle transaction_chain for Go embedding
	@echo "$(BLUE)Building transaction_chain bundle...$(NC)"
	@mkdir -p $(ROOT_DIR)/pkg/embedded/transactionchain/bundle
	cp $(TRANSACTION_CHAIN_DIR)/main.js $(ROOT_DIR)/pkg/embedded/transactionchain/bundle/
	cp $(TRANSACTION_CHAIN_DIR)/package.json $(ROOT_DIR)/pkg/embedded/transactionchain/bundle/
	@echo "$(GREEN)transaction_chain bundled at $(ROOT_DIR)/pkg/embedded/transactionchain/bundle/$(NC)"

# Internal Embedded Programs - Validation Chain (Rust)
validation-chain-deps: ## Install validation_chain dependencies
	@echo "$(BLUE)Installing validation_chain dependencies...$(NC)"
	cd $(VALIDATION_CHAIN_DIR) && cargo fetch

validation-chain-build: validation-chain-deps ## Build validation_chain standalone HTTP binary (the only supported embedding target — no CGo static lib)
	@echo "$(BLUE)Building validation_chain standalone binary...$(NC)"
	@mkdir -p $(EMBEDDED_BUILD_DIR)/validation_chain
	cd $(VALIDATION_CHAIN_DIR) && cargo build --release
	cp $(VALIDATION_CHAIN_DIR)/target/release/validationchain $(EMBEDDED_BUILD_DIR)/validation_chain/validationchain
	@echo "$(GREEN)validation_chain binary built: $(EMBEDDED_BUILD_DIR)/validation_chain/validationchain$(NC)"
	@mkdir -p $(ROOT_DIR)/pkg/embedded/validationchain/bin
	@gzip -cf $(VALIDATION_CHAIN_DIR)/target/release/validationchain > $(ROOT_DIR)/pkg/embedded/validationchain/bin/validationchain
	@echo "$(GREEN)validation_chain binary embedded: $(ROOT_DIR)/pkg/embedded/validationchain/bin/validationchain$(NC)"

# Unified Embedded Build
embedded-build: validation-chain-build transaction-chain-build ## Build all embedded programs
	@echo "$(GREEN)All embedded programs built successfully$(NC)"

# KNIRVAGENT (Embedded) targets
agent-deps: ## Install KNIRVAGENT dependencies
	@echo "$(BLUE)Installing KNIRVAGENT dependencies...$(NC)"
	cd $(AGENT_SRC_DIR) && go mod download && go mod tidy

agent-build: agent-deps ## Build KNIRVAGENT binary into pkg/knirvagent/bin/ (required before binary target)
	@echo "$(BLUE)Building KNIRVAGENT binary...$(NC)"
	@mkdir -p $(AGENT_DIR)/bin
	@rm -f $(AGENT_SRC_DIR)/build/knirvagent $(AGENT_PKG_BIN)
	cd $(AGENT_SRC_DIR) && make build
	cp $(AGENT_SRC_DIR)/build/knirvagent $(AGENT_PKG_BIN)
	@echo "$(BLUE)Compressing KNIRVAGENT binary with gzip...$(NC)"
	@rm -f $(AGENT_PKG_BIN).tmp
	@gzip -cf $(AGENT_PKG_BIN) > $(AGENT_PKG_BIN).tmp && mv $(AGENT_PKG_BIN).tmp $(AGENT_PKG_BIN)
	@echo "$(GREEN)KNIRVAGENT binary built: $(AGENT_PKG_BIN)$(NC)"

agent-clean: ## Clean KNIRVAGENT build artifacts
	@echo "$(BLUE)Cleaning KNIRVAGENT artifacts...$(NC)"
	rm -f $(AGENT_PKG_BIN)
	cd $(AGENT_SRC_DIR) && make clean

# KNIRVARENA (Embedded static bundle) targets
arena-deps: ## Install KNIRVARENA dependencies
	@echo "$(BLUE)Installing KNIRVARENA dependencies...$(NC)"
	cd $(ARENA_SRC_DIR) && $(NPM_CMD) install

arena-build: arena-deps ## Build KNIRVARENA dist and bundle into pkg/knirvarena/bin/ (required before binary target)
	@echo "$(BLUE)Building KNIRVARENA static bundle...$(NC)"
	cd $(ARENA_SRC_DIR) && $(NPM_CMD) run build
	@mkdir -p $(ARENA_DIR)/bin
	@rm -f $(ARENA_PKG_BUNDLE)
	cd $(ARENA_SRC_DIR)/dist && tar -czf $(ARENA_PKG_BUNDLE) .
	@echo "$(GREEN)KNIRVARENA bundle built: $(ARENA_PKG_BUNDLE)$(NC)"

arena-clean: ## Clean KNIRVARENA build artifacts
	@echo "$(BLUE)Cleaning KNIRVARENA artifacts...$(NC)"
	rm -f $(ARENA_PKG_BUNDLE)
	cd $(ARENA_SRC_DIR) && rm -rf dist
