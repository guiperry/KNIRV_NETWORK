# KNIRV TESTNET Makefile
# Unified build and test automation for KNIRV Testnet

.PHONY: help build start stop health test clean validate backup all modp-compile modp-verify modp-network modp-component test-modp
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
	@echo "  build-nexus      Build KNIRV-SERVER"
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
	@echo "  test-modp        Run modp formal verification tests"
	@echo ""
	@echo "$(GREEN)Formal Verification:$(NC)"
	@echo "  modp-compile     Compile P model only"
	@echo "  modp-verify      Run full formal verification suite"
	@echo "  modp-network     Run network-wide P model tests"
	@echo "  modp-component   Run component P model tests"
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

build-nexus: ## Build KNIRV-SERVER
	@echo "$(BLUE)Building KNIRV-SERVER...$(NC)"
	@$(SCRIPTS_DIR)/build-knirvserver.sh

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

test-modp: ## Run modp formal verification tests as a testnet test category
	@echo "$(GREEN)Running modp formal verification tests...$(NC)"
	@$(SCRIPTS_DIR)/run-tests.sh --category formal-verification --no-start

# modp Formal Verification Targets
MODP_DIR ?= $(shell cd ../.. && pwd)/modp

modp-compile: ## Compile the P formal model only
	@echo "$(BLUE)Compiling P model...$(NC)"
	@test -d "$(MODP_DIR)" || (echo "$(RED)modp directory not found: $(MODP_DIR)$(NC)" && exit 1)
	@cd "$(MODP_DIR)" && bash scripts/run-tests.sh compile

modp-verify: ## Run full modp formal verification suite
	@echo "$(GREEN)Running full modp verification...$(NC)"
	@test -d "$(MODP_DIR)" || (echo "$(RED)modp directory not found: $(MODP_DIR)$(NC)" && exit 1)
	@cd "$(MODP_DIR)" && bash scripts/run-tests.sh all

modp-network: ## Run network-wide P model tests only
	@echo "$(GREEN)Running network-wide P model tests...$(NC)"
	@test -d "$(MODP_DIR)" || (echo "$(RED)modp directory not found: $(MODP_DIR)$(NC)" && exit 1)
	@cd "$(MODP_DIR)" && bash scripts/run-tests.sh network

modp-component: ## Run component P model tests only
	@echo "$(GREEN)Running component P model tests...$(NC)"
	@test -d "$(MODP_DIR)" || (echo "$(RED)modp directory not found: $(MODP_DIR)$(NC)" && exit 1)
	@cd "$(MODP_DIR)" && bash scripts/run-tests.sh component

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
all: build-all start test-all modp-verify ## Build, start, test everything, and run formal verification
	@echo "$(GREEN)Complete testnet setup and validation finished!$(NC)"

testnet: start test ## Start testnet and run tests
	@echo "$(GREEN)Testnet started and tested successfully!$(NC)"

# Special target for the user's request
testnet-tests: testnet ## Start testnet and run tests (alias for testnet)
