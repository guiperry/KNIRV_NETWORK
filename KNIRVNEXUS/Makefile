# KNIRV-NEXUS Unified Build System
# This Makefile orchestrates the complete build, test, and deployment pipeline

# Project Configuration
PROJECT_NAME := knirv-nexus
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# Directories
ROOT_DIR := $(shell pwd)
BACKEND_DIR := $(ROOT_DIR)/backend
FRONTEND_DIR := $(ROOT_DIR)
BUILD_DIR := $(ROOT_DIR)/build
DIST_DIR := $(ROOT_DIR)/dist
DEPLOYMENT_DIR := $(ROOT_DIR)/deployment

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
DOCKER_IMAGE := knirv-nexus
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

.PHONY: help all clean build test deploy install deps frontend backend binary docker

# Default target
all: clean deps frontend backend binary

help: ## Show this help message
	@echo "$(BLUE)KNIRV-NEXUS Build System$(NC)"
	@echo ""
	@echo "$(YELLOW)Available targets:$(NC)"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  $(GREEN)%-20s$(NC) %s\n", $$1, $$2}' $(MAKEFILE_LIST)

# Dependency Management
deps: deps-go deps-node deps-ansible ## Install all dependencies

deps-go: ## Install Go dependencies
	@echo "$(BLUE)Installing Go dependencies...$(NC)"
	cd $(BACKEND_DIR) && go mod download
	cd $(BACKEND_DIR) && go mod tidy

deps-node: ## Install Node.js dependencies
	@echo "$(BLUE)Installing Node.js dependencies...$(NC)"
	$(NPM_CMD) install

deps-ansible: ## Install Ansible dependencies
	@echo "$(BLUE)Installing Ansible dependencies...$(NC)"
	cd $(DEPLOYMENT_DIR)/ansible && ansible-galaxy collection install -r requirements.yml

# Frontend Build
frontend: deps-node ## Build Next.js frontend
	@echo "$(BLUE)Building Next.js frontend...$(NC)"
	$(NPM_CMD) run build
	@echo "$(GREEN)Frontend build completed$(NC)"

frontend-dev: deps-node ## Start frontend development server
	@echo "$(BLUE)Starting frontend development server...$(NC)"
	$(NPM_CMD) run dev

# Backend Build (Unified single binary)
backend: deps-go ## Build unified Go backend service
	@echo "$(BLUE)Building unified backend binary...$(NC)"
	cd $(BACKEND_DIR) && go build -ldflags "$(LDFLAGS)" -o bin/nexus-server ./cmd/nexus-server
	@echo "$(BLUE)Copying backend binary to root...$(NC)"
	@mkdir -p bin
	cp $(BACKEND_DIR)/bin/nexus-server bin/
	@echo "$(GREEN)Backend build completed$(NC)"

# Unified Binary (Frontend + Unified Backend)
binary: frontend backend ## Build unified binary with embedded frontend and backend
	@echo "$(BLUE)Creating unified binary...$(NC)"
	@mkdir -p $(DIST_DIR)
	go build -ldflags "$(LDFLAGS)" -o $(DIST_DIR)/nexus-server .
	@echo "$(GREEN)Unified binary created: $(DIST_DIR)/nexus-server$(NC)"

# Testing
test: test-go test-frontend test-architecture ## Run all tests

test-go: deps-go ## Run Go unit tests
	@echo "$(BLUE)Running Go unit tests...$(NC)"
	cd $(BACKEND_DIR) && go test -v -race -coverprofile=coverage.out ./...
	cd $(BACKEND_DIR) && go tool cover -html=coverage.out -o coverage.html

test-frontend: deps-node ## Run frontend tests
	@echo "$(BLUE)Running frontend tests...$(NC)"
	$(NPM_CMD) test

test-integration: binary ## Run integration tests
	@echo "$(BLUE)Running integration tests...$(NC)"
	cd $(BACKEND_DIR) && go test -v -tags=integration ./tests/integration/...

test-e2e: binary ## Run end-to-end tests
	@echo "$(BLUE)Running end-to-end tests...$(NC)"
	cd $(BACKEND_DIR) && go test -v -tags=e2e ./tests/e2e/...

test-architecture: backend ## Test the new three-tier architecture
	@echo "$(BLUE)Testing architecture...$(NC)"
	cd $(BACKEND_DIR) && go test -v ./tests -run TestArchitecture
	@echo "$(GREEN)Architecture tests completed$(NC)"

# Linting and Formatting
lint: lint-go lint-frontend ## Run all linters

lint-go: ## Run Go linters
	@echo "$(BLUE)Running Go linters...$(NC)"
	cd $(BACKEND_DIR) && golangci-lint run

lint-frontend: ## Run frontend linters
	@echo "$(BLUE)Running frontend linters...$(NC)"
	$(NPM_CMD) run lint

format: format-go format-frontend ## Format all code

format-go: ## Format Go code
	@echo "$(BLUE)Formatting Go code...$(NC)"
	cd $(BACKEND_DIR) && go fmt ./...
	cd $(BACKEND_DIR) && goimports -w .

format-frontend: ## Format frontend code
	@echo "$(BLUE)Formatting frontend code...$(NC)"
	$(NPM_CMD) run format

# Development
dev: ## Start development environment
	@echo "$(BLUE)Starting development environment...$(NC)"
	$(MAKE) -j2 dev-backend dev-frontend

dev-backend: backend ## Start unified backend development server
	@echo "$(BLUE)Starting unified backend development server...$(NC)"
	cd $(BACKEND_DIR) && ./bin/nexus-server --config config/development.yaml

dev-frontend: ## Start frontend development server
	$(MAKE) frontend-dev

# Docker
docker: ## Build Docker image
	@echo "$(BLUE)Building Docker image...$(NC)"
	docker build -t $(DOCKER_IMAGE):$(DOCKER_TAG) .
	docker tag $(DOCKER_IMAGE):$(DOCKER_TAG) $(DOCKER_IMAGE):latest

docker-run: docker ## Run Docker container
	@echo "$(BLUE)Running Docker container...$(NC)"
	docker run -p 8080:8080 $(DOCKER_IMAGE):$(DOCKER_TAG)

# Deployment
deploy: deploy-infrastructure deploy-dns deploy-application ## Full deployment

deploy-infrastructure: ## Deploy AWS infrastructure
	@echo "$(BLUE)Deploying AWS infrastructure...$(NC)"
	cd $(DEPLOYMENT_DIR) && ./scripts/deploy.sh --environment $(ENVIRONMENT) --skip-dns --skip-application

deploy-dns: ## Configure DNS
	@echo "$(BLUE)Configuring DNS...$(NC)"
	cd $(DEPLOYMENT_DIR) && ./scripts/deploy.sh --environment $(ENVIRONMENT) --skip-infrastructure --skip-application

deploy-application: binary ## Deploy application
	@echo "$(BLUE)Deploying application...$(NC)"
	cd $(DEPLOYMENT_DIR) && ./scripts/deploy.sh --environment $(ENVIRONMENT) --skip-infrastructure --skip-dns

deploy-full: ## Full deployment with all components
	@echo "$(BLUE)Running full deployment...$(NC)"
	cd $(DEPLOYMENT_DIR) && ./scripts/full-deploy.sh --environment $(ENVIRONMENT)

deploy-monitoring: ## Deploy monitoring stack
	@echo "$(BLUE)Deploying monitoring...$(NC)"
	cd $(DEPLOYMENT_DIR)/ansible && ansible-playbook playbooks/monitoring.yml -e env=$(ENVIRONMENT)

deploy-security: ## Apply security hardening
	@echo "$(BLUE)Applying security hardening...$(NC)"
	cd $(DEPLOYMENT_DIR)/ansible && ansible-playbook playbooks/security.yml -e env=$(ENVIRONMENT)

# Database
db-migrate: ## Run database migrations
	@echo "$(BLUE)Running database migrations...$(NC)"
	cd $(BACKEND_DIR) && go run ./cmd/migrate

db-seed: ## Seed database with test data
	@echo "$(BLUE)Seeding database...$(NC)"
	cd $(BACKEND_DIR) && go run ./cmd/seed

# Utilities
clean: ## Clean build artifacts
	@echo "$(BLUE)Cleaning build artifacts...$(NC)"
	rm -rf $(BUILD_DIR)
	rm -rf $(DIST_DIR)
	rm -rf $(BACKEND_DIR)/bin
	rm -rf $(BACKEND_DIR)/coverage.out
	rm -rf $(BACKEND_DIR)/coverage.html
	rm -rf bin
	rm -rf .next
	rm -rf out
	rm -rf node_modules/.cache

clean-all: clean ## Clean everything including dependencies
	@echo "$(BLUE)Cleaning all artifacts and dependencies...$(NC)"
	rm -rf node_modules
	cd $(BACKEND_DIR) && go clean -modcache

install: binary ## Install binary to system
	@echo "$(BLUE)Installing binary...$(NC)"
	sudo cp $(DIST_DIR)/nexus-server /usr/local/bin/
	sudo chmod +x /usr/local/bin/nexus-server

uninstall: ## Uninstall binary from system
	@echo "$(BLUE)Uninstalling binary...$(NC)"
	sudo rm -f /usr/local/bin/nexus-server

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
	tar -czf backups/nexus-backup-$(shell date +%Y%m%d-%H%M%S).tar.gz data/ config/

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
