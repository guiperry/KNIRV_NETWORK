# KNIRV-SERVER Unified Build System
# This Makefile orchestrates the complete build, test, and deployment pipeline

# Project Configuration
PROJECT_NAME := knirv-server
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# Directories
ROOT_DIR := $(shell pwd)
BACKEND_DIR := $(ROOT_DIR)/backend
FRONTEND_DIR := $(ROOT_DIR)/frontend
DESKTOP_DIR := $(ROOT_DIR)/desktop
GATEWAY_DIR := $(ROOT_DIR)/pkg/knirvgateway
BUILD_DIR := $(ROOT_DIR)/build
DIST_DIR := $(ROOT_DIR)/dist
DEPLOYMENT_DIR := $(ROOT_DIR)/deployment
APPDATA_DIR := $(HOME)/.local/share/knirvserver

# Go Configuration
GO_VERSION := 1.21
GOOS := $(shell go env GOOS)
GOARCH := $(shell go env GOARCH)
CGO_ENABLED := 0

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

.PHONY: help all clean build test deploy install deps frontend backend binary docker os deployer deploy-dev-local ebpf-generate kali-docker-image desktop desktop-deps desktop-build dash gateway-sync gateway-build gateway-deps gateway-clean build-full

# Default target
all: frontend desktop-build backend binary

# Desktop (Electron)
desktop-deps: ## Install desktop dependencies
	@echo "$(BLUE)Installing desktop dependencies...$(NC)"
	cd $(DESKTOP_DIR) && $(NPM_CMD) install

desktop-build: desktop-deps ## Build desktop TypeScript and copy assets to dist/
	@echo "$(BLUE)Building desktop...$(NC)"
	cd $(DESKTOP_DIR) && $(NPM_CMD) run build
	@echo "$(GREEN)Desktop build completed$(NC)"

desktop: desktop-build ## Build desktop and launch standalone (for development)
	@echo "$(BLUE)Starting desktop...$(NC)"
	cd $(DESKTOP_DIR) && $(NPM_CMD) run start

dash: desktop-build ## Rebuild desktop and relink into binary (fast path, skips frontend/backend/eBPF)
	@echo "$(BLUE)Rebuilding binary with updated desktop...$(NC)"
	@mkdir -p $(DIST_DIR)
	go build -ldflags "$(LDFLAGS)" -o $(DIST_DIR)/knirv-server .
	chmod 755 $(DIST_DIR)/knirv-server
	@echo "$(GREEN)Done — run: ./dist/knirv-server --desktop$(NC)"

help: ## Show this help message
	@echo "$(BLUE)KNIRV-SERVER Build System$(NC)"
	@echo ""
	@echo "$(YELLOW)Available targets:$(NC)"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  $(GREEN)%-20s$(NC) %s\n", $$1, $$2}' $(MAKEFILE_LIST)

# eBPF Generation
ebpf-generate: ## Generate eBPF Go code from C programs
	@echo "Generating eBPF programs..."
	cd backend/internal/ebpf && go generate
	@echo "eBPF generation completed"

# Dependency Management
deps: deps-go deps-node deps-ansible ## Install all dependencies

deps-go: ## Install Go dependencies (backend + gateway)
	@echo "$(BLUE)Installing Go dependencies...$(NC)"
	cd $(BACKEND_DIR) && go mod download
	cd $(BACKEND_DIR) && go mod tidy
	cd $(GATEWAY_DIR) && go mod download
	cd $(GATEWAY_DIR) && go mod tidy

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
backend: deps-go ## Build complete Go backend service package
	@echo "$(BLUE)Building complete backend package binary...$(NC)"
	cd $(BACKEND_DIR) && go build -ldflags "$(LDFLAGS)" -o bin/backend_server ./cmd/backend_server
	@echo "$(BLUE)Copying backend binary to root...$(NC)"
	@mkdir -p bin
	cp $(BACKEND_DIR)/bin/backend_server bin/
	@echo "$(GREEN)Backend build completed$(NC)"

# Specialized Build Tools
os: deps-go ## Build os_builder binary
	@echo "$(BLUE)Building os_builder binary...$(NC)"
	cd $(BACKEND_DIR)/cmd/os_builder && go build -ldflags "$(LDFLAGS)" -o os_builder main.go
	@echo "$(GREEN)os_builder build completed: backend/cmd/os_builder/os_builder$(NC)"

deployer: deps-go ## Build container_deployer binary
	@echo "$(BLUE)Building container_deployer binary...$(NC)"
	cd $(BACKEND_DIR)/cmd/container_deployer && go build -ldflags "$(LDFLAGS)" -o container_deployer main.go
	@echo "$(GREEN)container_deployer build completed: backend/cmd/container_deployer/container_deployer$(NC)"

# Unified Binary (Frontend + Desktop + Backend + KNIRVGATEWAY)
# main.go embeds: bin/backend_server, bin/knirvgateway, config/*, desktop/dist/*, frontend/out/*
# At runtime the launcher extracts backend_server and knirvgateway to ~/.local/share/knirvserver/bin/
binary: ebpf-generate frontend desktop-build backend gateway-build ## Build unified binary embedding frontend, desktop, backend, and KNIRVGATEWAY
	@echo "$(BLUE)Creating unified binary (embeds backend_server + knirvgateway + frontend + desktop)...$(NC)"
	@mkdir -p $(DIST_DIR)
	go build -ldflags "$(LDFLAGS)" -o $(DIST_DIR)/knirv-server .
	chmod 755 $(DIST_DIR)/knirv-server
	@echo "$(GREEN)Unified binary created: $(DIST_DIR)/knirv-server$(NC)"
	@echo "$(BLUE)Copying binary to container_deployer golang-app-source in app data directory...$(NC)"
	@mkdir -p $(APPDATA_DIR)/container_deployer/resources/golang-app-source
	cp $(DIST_DIR)/knirv-server $(APPDATA_DIR)/container_deployer/resources/golang-app-source
	chmod +x $(APPDATA_DIR)/container_deployer/resources/golang-app-source/knirv-server
	@mkdir -p $(BACKEND_DIR)/cmd/container_deployer/golang-app-source
	cp $(DIST_DIR)/knirv-server $(BACKEND_DIR)/cmd/container_deployer/golang-app-source
	chmod +x $(BACKEND_DIR)/cmd/container_deployer/golang-app-source/knirv-server
	@echo "$(GREEN)Binary copied to container_deployer/golang-app-source in both embedded backend and app data directory$(NC)"
	@echo "$(GREEN)Good Build!$(NC)"

# Testing
test: test-go test-frontend ## Run standard tests (unit + frontend)

test-all: test-go test-frontend test-integration test-e2e test-architecture ## Run all non-privileged tests

test-complete: test-all test-privileged ## Run complete test suite including privileged tests

test-go: deps-go ## Run Go unit tests
	@echo "$(BLUE)Running Go unit tests...$(NC)"
	@./scripts/run-tests.sh unit

test-frontend: deps-node ## Run frontend tests
	@echo "$(BLUE)Running frontend tests...$(NC)"
	cd $(FRONTEND_DIR) && $(NPM_CMD) test

test-integration: deps-go ## Run integration tests
	@echo "$(BLUE)Running integration tests...$(NC)"
	@./scripts/run-tests.sh integration

test-e2e: deps-go ## Run end-to-end tests
	@echo "$(BLUE)Running end-to-end tests...$(NC)"
	@./scripts/run-tests.sh e2e

test-architecture: deps-go ## Test the new three-tier architecture
	@echo "$(BLUE)Testing architecture...$(NC)"
	@./scripts/run-tests.sh architecture

test-privileged: ## Run privileged tests in Docker container (cgroups, namespaces, etc.)
	@echo "$(BLUE)Running privileged tests in Docker container...$(NC)"
	@./scripts/test-server-privileged.sh $(if $(SHOW_BUILD),--show-build)
	@echo "$(GREEN)Privileged tests completed$(NC)"

test-privileged-quick: ## Run privileged tests using existing container
	@echo "$(BLUE)Running quick privileged tests...$(NC)"
	@./scripts/test-server-privileged.sh --no-deploy $(if $(SHOW_BUILD),--show-build)
	@echo "$(GREEN)Quick privileged tests completed$(NC)"

test-privileged-full: ## Run full privileged tests with cleanup
	@echo "$(BLUE)Running full privileged tests with cleanup...$(NC)"
	@./scripts/test-server-privileged.sh --cleanup $(if $(SHOW_BUILD),--show-build)
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
	cd $(BACKEND_DIR) && ./bin/backend_server --config config/development.yaml

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
	rm -f bin/backend_server bin/knirvgateway
	rm -rf $(FRONTEND_DIR)/.next $(FRONTEND_DIR)/out
	cd $(GATEWAY_DIR) && go clean

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
gateway-sync: ## Sync KNIRVGATEWAY source to embedded package
	@echo "$(BLUE)Syncing KNIRVGATEWAY to embedded package...$(NC)"
	@./scripts/sync-knirvgateway.sh

gateway-deps: ## Install KNIRVGATEWAY dependencies
	@echo "$(BLUE)Installing KNIRVGATEWAY dependencies...$(NC)"
	cd $(GATEWAY_DIR) && go mod download && go mod tidy

gateway-build: gateway-deps ## Build KNIRVGATEWAY binary into bin/ (required before binary target)
	@echo "$(BLUE)Building KNIRVGATEWAY binary...$(NC)"
	@mkdir -p bin
	cd $(GATEWAY_DIR) && go build -ldflags "-s -w" -o $(ROOT_DIR)/bin/knirvgateway ./cmd/gateway
	@echo "$(GREEN)KNIRVGATEWAY binary built: bin/knirvgateway$(NC)"

gateway-clean: ## Clean KNIRVGATEWAY build artifacts
	@echo "$(BLUE)Cleaning KNIRVGATEWAY artifacts...$(NC)"
	rm -f bin/knirvgateway
	cd $(GATEWAY_DIR) && go clean

# Full build including embedded KNIRVGATEWAY (gateway-build is now a dep of binary)
build-full: binary ## Build everything including KNIRVGATEWAY (alias for binary)

