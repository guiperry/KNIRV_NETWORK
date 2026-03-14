# KNIRV TESTNET Makefile
# Unified build and test automation for KNIRV Testnet

.PHONY: help build start stop health test clean validate backup all
.DEFAULT_GOAL := help

# Colors for output
BLUE := \033[0;34m
GREEN := \033[0;32m
YELLOW := \033[1;33m
RED := \033[0;31m
NC := \033[0m # No Color

# Configuration
SCRIPTS_DIR := scripts

help: ## Show this help message
	@echo "$(BLUE)KNIRV TESTNET - Make Commands$(NC)"
	@echo "================================"
	@echo ""
	@echo "$(GREEN)Build Commands:$(NC)"
	@echo "  build-all        Build all KNIRV components"
	@echo "  build-oracle       Build KNIRV-ORACLE"
	@echo "  build-chain      Build KNIRVCHAIN"
	@echo "  build-graph      Build KNIRVGRAPH"
	@echo "  build-nexus      Build KNIRV-NEXUS"
	@echo "  build-router     Build KNIRV-ROUTER"
	@echo "  build-gateway    Build KNIRV-GATEWAY"
	@echo ""
	@echo "$(GREEN)Service Management:$(NC)"
	@echo "  start            Start the testnet"
	@echo "  stop             Stop the testnet"
	@echo "  restart          Restart the testnet"
	@echo "  health           Check service health"
	@echo "  status           Show service status"
	@echo ""
	@echo "$(GREEN)Testing:$(NC)"
	@echo "  test             Run integration tests (default)"
	@echo "  test-all         Run all test categories"
	@echo "  test-e2e         Run end-to-end tests"
	@echo "  test-performance Run performance tests"
	@echo "  test-security    Run security tests"
	@echo "  test-cortex      Run CORTEX demos"
	@echo "  test-integration Run integration tests"
	@echo "  test-economics   Run economics tests"
	@echo ""
	@echo "$(GREEN)Maintenance:$(NC)"
	@echo "  validate         Validate configuration"
	@echo "  backup           Backup testnet data"
	@echo "  clean            Clean temporary files"
	@echo "  logs             Show recent logs"
	@echo ""
	@echo "$(GREEN)Combined Operations:$(NC)"
	@echo "  all              Build, start, and test everything"
	@echo "  testnet          Start testnet and run tests"

# Build Commands
build-all: ## Build all KNIRV components
	@echo "$(BLUE)Building all KNIRV components...$(NC)"
	@$(SCRIPTS_DIR)/build-all.sh

build-oracle: ## Build KNIRV-ORACLE
	@echo "$(BLUE)Building KNIRV-ORACLE...$(NC)"
	@$(SCRIPTS_DIR)/build-knirvoracle.sh

build-chain: ## Build KNIRVCHAIN
	@echo "$(BLUE)Building KNIRVCHAIN...$(NC)"
	@$(SCRIPTS_DIR)/build-knirvchain.sh

build-graph: ## Build KNIRVGRAPH
	@echo "$(BLUE)Building KNIRVGRAPH...$(NC)"
	@$(SCRIPTS_DIR)/build-knirvgraph.sh

build-nexus: ## Build KNIRV-NEXUS
	@echo "$(BLUE)Building KNIRV-NEXUS...$(NC)"
	@$(SCRIPTS_DIR)/build-knirvnexus.sh

build-router: ## Build KNIRV-ROUTER
	@echo "$(BLUE)Building KNIRV-ROUTER...$(NC)"
	@$(SCRIPTS_DIR)/build-knirvrouter.sh

build-gateway: ## Build KNIRV-GATEWAY
	@echo "$(BLUE)Building KNIRV-GATEWAY...$(NC)"
	@$(SCRIPTS_DIR)/build-knirvgateway.sh

# Service Management
start: ## Start the testnet
	@echo "$(GREEN)Starting KNIRV Testnet...$(NC)"
	@$(SCRIPTS_DIR)/start-testnet.sh

stop: ## Stop the testnet
	@echo "$(YELLOW)Stopping KNIRV Testnet...$(NC)"
	@$(SCRIPTS_DIR)/stop-testnet.sh

restart: stop start ## Restart the testnet

health: ## Check service health
	@echo "$(BLUE)Checking service health...$(NC)"
	@$(SCRIPTS_DIR)/health-check.sh

status: health ## Show service status (alias for health)

# Testing Commands
test: ## Run integration tests (default)
	@echo "$(GREEN)Running integration tests...$(NC)"
	@$(SCRIPTS_DIR)/run-tests.sh

test-all: ## Run all test categories
	@echo "$(GREEN)Running all test categories...$(NC)"
	@$(SCRIPTS_DIR)/run-tests.sh --all

test-e2e: ## Run end-to-end tests
	@echo "$(GREEN)Running end-to-end tests...$(NC)"
	@$(SCRIPTS_DIR)/run-tests.sh --category e2e

test-performance: ## Run performance tests
	@echo "$(GREEN)Running performance tests...$(NC)"
	@$(SCRIPTS_DIR)/run-tests.sh --category performance

test-security: ## Run security tests
	@echo "$(GREEN)Running security tests...$(NC)"
	@$(SCRIPTS_DIR)/run-tests.sh --category security

test-cortex: ## Run CORTEX demos
	@echo "$(GREEN)Running CORTEX demos...$(NC)"
	@$(SCRIPTS_DIR)/run-tests.sh --category cortex-demos

test-integration: ## Run integration tests
	@echo "$(GREEN)Running integration tests...$(NC)"
	@$(SCRIPTS_DIR)/test-integration.sh

test-economics: ## Run economics tests
	@echo "$(GREEN)Running economics tests...$(NC)"
	@$(SCRIPTS_DIR)/test-economics.sh

# Maintenance Commands
validate: ## Validate configuration
	@echo "$(BLUE)Validating configuration...$(NC)"
	@$(SCRIPTS_DIR)/validate-config.sh

backup: ## Backup testnet data
	@echo "$(BLUE)Backing up testnet data...$(NC)"
	@$(SCRIPTS_DIR)/backup-data.sh

clean: ## Clean temporary files
	@echo "$(YELLOW)Cleaning temporary files...$(NC)"
	@rm -rf logs/*.log || true
	@rm -rf data/*.tmp || true
	@rm -rf data/*.lock || true
	@echo "$(GREEN)Cleanup completed$(NC)"

logs: ## Show recent logs
	@echo "$(BLUE)Recent testnet logs:$(NC)"
	@tail -n 20 logs/*.log 2>/dev/null || echo "No log files found"

# Combined Operations
all: build-all start test-all ## Build, start, and test everything
	@echo "$(GREEN)Complete testnet setup and validation finished!$(NC)"

testnet: start test ## Start testnet and run tests
	@echo "$(GREEN)Testnet started and tested successfully!$(NC)"

# Special target for the user's request
testnet-tests: testnet ## Start testnet and run tests (alias for testnet)
