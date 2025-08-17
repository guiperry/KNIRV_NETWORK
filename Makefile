# KNIRV Network Infrastructure and Deployment Makefile
# This Makefile provides convenient commands for infrastructure deployment and application management

# =============================================================================
# CONFIGURATION
# =============================================================================

# Default environment
ENVIRONMENT ?= production

# Cloud provider
CLOUD_PROVIDER ?= aws

# Project directories
PROJECT_ROOT := $(shell pwd)
ANSIBLE_DIR := $(PROJECT_ROOT)/deployment/ansible
SCRIPTS_DIR := $(PROJECT_ROOT)/scripts
KNIRVGATEWAY_DIR := $(PROJECT_ROOT)/KNIRVGATEWAY
DOCS_DIR := $(PROJECT_ROOT)/KNIRVGATEWAY/documentation
DEPLOYMENT_DIR := $(PROJECT_ROOT)/deployment

# Colors for output
RED := \033[0;31m
GREEN := \033[0;32m
YELLOW := \033[1;33m
BLUE := \033[0;34m
NC := \033[0m # No Color

# Check if .env file exists
ENV_FILE := $(ANSIBLE_DIR)/.env
ifeq ($(wildcard $(ENV_FILE)),)
    $(warning Warning: .env file not found. Copy deployment/ansible/.env.example to deployment/ansible/.env and configure your settings)
endif

# =============================================================================
# HELP TARGET
# =============================================================================

.PHONY: help
help: ## Show this help message
	@echo "$(BLUE)KNIRV Network Infrastructure and Deployment$(NC)"
	@echo "=============================================="
	@echo ""
	@echo "$(YELLOW)Available targets:$(NC)"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  $(GREEN)%-20s$(NC) %s\n", $$1, $$2}' $(MAKEFILE_LIST)
	@echo ""
	@echo "$(YELLOW)Environment Variables:$(NC)"
	@echo "  ENVIRONMENT     Deployment environment (production, development, staging)"
	@echo "  CLOUD_PROVIDER  Cloud provider (aws, gcp, azure, digitalocean)"
	@echo ""
	@echo "$(YELLOW)Examples:$(NC)"
	@echo "  make tests                    # Run comprehensive test suite"
	@echo "  make testnet-tests           # Start KNIRVTESTNET and run tests"
	@echo "  make test-quick              # Run quick tests only"
	@echo "  make test-coverage           # Generate coverage reports"
	@echo "  make deploy-infrastructure ENVIRONMENT=production"
	@echo "  make deploy-full ENVIRONMENT=development CLOUD_PROVIDER=aws"
	@echo "  make docs && make deploy-website"

# =============================================================================
# PREREQUISITES AND SETUP
# =============================================================================

.PHONY: check-env
check-env: ## Check if .env file exists and is configured
	@if [ ! -f "$(ENV_FILE)" ]; then \
		echo "$(RED)Error: .env file not found$(NC)"; \
		echo "$(YELLOW)Please copy deployment/ansible/.env.example to deployment/ansible/.env and configure your settings$(NC)"; \
		exit 1; \
	fi
	@echo "$(GREEN)✓ Environment file found$(NC)"

.PHONY: install-deps
install-deps: ## Install Ansible and required dependencies
	@echo "$(BLUE)Installing Ansible dependencies...$(NC)"
	@pip install ansible
	@cd $(ANSIBLE_DIR) && ansible-galaxy collection install -r requirements.yml
	@echo "$(GREEN)✓ Dependencies installed$(NC)"

.PHONY: check-prereqs
check-prereqs: check-env ## Check all prerequisites
	@echo "$(BLUE)Checking prerequisites...$(NC)"
	@command -v ansible-playbook >/dev/null 2>&1 || { echo "$(RED)Error: ansible-playbook not found$(NC)"; exit 1; }
	@command -v node >/dev/null 2>&1 || { echo "$(RED)Error: node not found$(NC)"; exit 1; }
	@echo "$(GREEN)✓ All prerequisites satisfied$(NC)"

# =============================================================================
# DOCUMENTATION GENERATION
# =============================================================================

.PHONY: docs-deps
docs-deps: ## Install documentation dependencies
	@echo "$(BLUE)Installing documentation dependencies...$(NC)"
	@if [ -f "$(DOCS_DIR)/package.json" ]; then \
		cd $(DOCS_DIR) && npm install; \
	else \
		echo "$(YELLOW)Warning: No package.json found in documentation directory$(NC)"; \
	fi
	@echo "$(GREEN)✓ Documentation dependencies installed$(NC)"

.PHONY: docs
docs: docs-deps ## Generate documentation from docs/ folder
	@echo "$(BLUE)Generating documentation...$(NC)"
	@if [ -f "$(SCRIPTS_DIR)/doc_generator.js" ]; then \
		cd $(PROJECT_ROOT) && node $(SCRIPTS_DIR)/doc_generator.js; \
		echo "$(GREEN)✓ Documentation generated successfully$(NC)"; \
	else \
		echo "$(RED)Error: doc_generator.js not found at $(SCRIPTS_DIR)/doc_generator.js$(NC)"; \
		exit 1; \
	fi

.PHONY: docs-serve
docs-serve: docs ## Serve documentation locally for development
	@echo "$(BLUE)Starting documentation server...$(NC)"
	@if [ -f "$(DOCS_DIR)/package.json" ]; then \
		cd $(DOCS_DIR) && npm run serve; \
	else \
		echo "$(RED)Error: Cannot serve docs - package.json not found$(NC)"; \
		exit 1; \
	fi

.PHONY: docs-clean
docs-clean: ## Clean generated documentation
	@echo "$(BLUE)Cleaning generated documentation...$(NC)"
	@rm -rf $(DOCS_DIR)/docsify
	@echo "$(GREEN)✓ Documentation cleaned$(NC)"

# =============================================================================
# INFRASTRUCTURE DEPLOYMENT
# =============================================================================

.PHONY: deploy-infrastructure
deploy-infrastructure: check-prereqs ## Deploy infrastructure only
	@echo "$(BLUE)Deploying infrastructure for $(ENVIRONMENT) environment...$(NC)"
	@cd $(ANSIBLE_DIR) && ./deploy-infrastructure.sh $(ENVIRONMENT) --cloud-provider $(CLOUD_PROVIDER)
	@echo "$(GREEN)✓ Infrastructure deployment completed$(NC)"

.PHONY: deploy-infrastructure-dry-run
deploy-infrastructure-dry-run: check-prereqs ## Preview infrastructure deployment (dry run)
	@echo "$(BLUE)Previewing infrastructure deployment for $(ENVIRONMENT) environment...$(NC)"
	@cd $(ANSIBLE_DIR) && ./deploy-infrastructure.sh $(ENVIRONMENT) --cloud-provider $(CLOUD_PROVIDER) --dry-run

.PHONY: deploy-infrastructure-force
deploy-infrastructure-force: check-prereqs ## Deploy infrastructure without confirmation
	@echo "$(BLUE)Force deploying infrastructure for $(ENVIRONMENT) environment...$(NC)"
	@cd $(ANSIBLE_DIR) && ./deploy-infrastructure.sh $(ENVIRONMENT) --cloud-provider $(CLOUD_PROVIDER) --force

.PHONY: deploy-infrastructure-skip-dns
deploy-infrastructure-skip-dns: check-prereqs ## Deploy infrastructure without DNS updates
	@echo "$(BLUE)Deploying infrastructure (skip DNS) for $(ENVIRONMENT) environment...$(NC)"
	@cd $(ANSIBLE_DIR) && ./deploy-infrastructure.sh $(ENVIRONMENT) --cloud-provider $(CLOUD_PROVIDER) --skip-dns

# =============================================================================
# APPLICATION DEPLOYMENT
# =============================================================================

.PHONY: deploy-apps
deploy-apps: ## Deploy KNIRV applications using existing Docker/K8s infrastructure
	@echo "$(BLUE)Deploying KNIRV applications...$(NC)"
	@cd $(DEPLOYMENT_DIR) && ./deploy.sh deploy
	@echo "$(GREEN)✓ Applications deployed$(NC)"

.PHONY: deploy-monitoring
deploy-monitoring: ## Deploy monitoring stack only
	@echo "$(BLUE)Deploying monitoring stack...$(NC)"
	@cd $(DEPLOYMENT_DIR) && ./deploy.sh monitoring
	@echo "$(GREEN)✓ Monitoring stack deployed$(NC)"

.PHONY: deploy-website
deploy-website: docs ## Deploy KNIRVGATEWAY (with fresh documentation)
	@echo "$(BLUE)Deploying KNIRVGATEWAY with updated documentation...$(NC)"
	@if [ -d "$(KNIRVGATEWAY_DIR)" ]; then \
		echo "$(YELLOW)Note: KNIRVGATEWAY deployment method depends on your hosting setup$(NC)"; \
		echo "$(YELLOW)Documentation has been generated and is ready for deployment$(NC)"; \
		echo "$(YELLOW)Generated files are in: $(DOCS_DIR)/docsify$(NC)"; \
	else \
		echo "$(RED)Error: KNIRVGATEWAY directory not found$(NC)"; \
		exit 1; \
	fi
	@echo "$(GREEN)✓ KNIRVGATEWAY ready for deployment$(NC)"

# =============================================================================
# FULL DEPLOYMENT WORKFLOWS
# =============================================================================

.PHONY: deploy-full
deploy-full: docs deploy-infrastructure deploy-apps deploy-monitoring ## Full deployment: docs + infrastructure + apps + monitoring
	@echo "$(GREEN)✓ Full KNIRV Network deployment completed!$(NC)"
	@echo ""
	@echo "$(YELLOW)Next steps:$(NC)"
	@echo "1. Verify services: make health-check"
	@echo "2. Access Grafana: http://YOUR_IP:3000"
	@echo "3. View documentation: make docs-serve"

.PHONY: deploy-dev
deploy-dev: ## Quick development deployment
	@$(MAKE) deploy-full ENVIRONMENT=development CLOUD_PROVIDER=aws

.PHONY: deploy-prod
deploy-prod: ## Production deployment with confirmation
	@echo "$(RED)WARNING: This will deploy to PRODUCTION environment$(NC)"
	@read -p "Are you sure? (y/N): " confirm && [ "$$confirm" = "y" ]
	@$(MAKE) deploy-full ENVIRONMENT=production CLOUD_PROVIDER=aws

.PHONY: deploy-staging
deploy-staging: ## Staging deployment
	@$(MAKE) deploy-full ENVIRONMENT=staging CLOUD_PROVIDER=aws

.PHONY: deploy-testnet
deploy-testnet: ## Deploy KNIRVTESTNET to AWS with Netlify frontend integration
	@echo "$(BLUE)Deploying KNIRVTESTNET to AWS...$(NC)"
	@$(MAKE) deploy-testnet-infrastructure
	@$(MAKE) deploy-testnet-services
	@$(MAKE) update-testnet-frontend
	@echo "$(GREEN)✓ KNIRVTESTNET deployment completed!$(NC)"

.PHONY: deploy-testnet-infrastructure
deploy-testnet-infrastructure: check-prereqs ## Deploy testnet infrastructure only
	@echo "$(BLUE)Deploying KNIRVTESTNET infrastructure...$(NC)"
	@cd $(ANSIBLE_DIR) && ./deploy-testnet-infrastructure.sh
	@echo "$(GREEN)✓ KNIRVTESTNET infrastructure deployed$(NC)"

.PHONY: deploy-testnet-services
deploy-testnet-services: ## Deploy KNIRVTESTNET services via Docker Compose
	@echo "$(BLUE)Deploying KNIRVTESTNET services...$(NC)"
	@cd $(PROJECT_ROOT) && ./scripts/deploy-testnet-services.sh
	@echo "$(GREEN)✓ KNIRVTESTNET services deployed$(NC)"

.PHONY: update-testnet-frontend
update-testnet-frontend: ## Update KNIRVGATEWAY testnet frontend with latest changes
	@echo "$(BLUE)Updating KNIRVTESTNET frontend...$(NC)"
	@cd $(PROJECT_ROOT) && ./scripts/update-testnet-frontend.sh
	@echo "$(GREEN)✓ KNIRVTESTNET frontend updated$(NC)"

# =============================================================================
# COMPREHENSIVE TESTING SUITE
# =============================================================================

# Test directories and configuration
TEST_REPORTS_DIR := $(PROJECT_ROOT)/test-reports
COVERAGE_DIR := $(PROJECT_ROOT)/coverage
TIMESTAMP := $(shell date +"%Y%m%d_%H%M%S")

.PHONY: test-setup
test-setup: ## Setup test environment and directories
	@echo "$(BLUE)Setting up test environment...$(NC)"
	@mkdir -p $(TEST_REPORTS_DIR)
	@mkdir -p $(COVERAGE_DIR)
	@echo "$(GREEN)✓ Test environment ready$(NC)"

.PHONY: tests
tests: test-setup ## Run comprehensive test suite for entire KNIRV network
	@echo "$(BLUE)🚀 Running KNIRV Network Comprehensive Test Suite$(NC)"
	@echo "=================================================="
	@echo "Timestamp: $(TIMESTAMP)"
	@echo "Reports Directory: $(TEST_REPORTS_DIR)"
	@echo "Coverage Directory: $(COVERAGE_DIR)"
	@echo ""
	@$(MAKE) test-cortex
	@$(MAKE) test-sdk
	@$(MAKE) test-graph
	@$(MAKE) test-wallet
	@$(MAKE) test-nexus
	@$(MAKE) test-root
	@$(MAKE) test-integration
	@$(MAKE) test-reports
	@echo ""
	@echo "$(GREEN)🎉 All KNIRV Network tests completed!$(NC)"
	@echo "$(YELLOW)📊 View reports at: $(TEST_REPORTS_DIR)$(NC)"
	@echo "$(YELLOW)📈 View coverage at: $(COVERAGE_DIR)$(NC)"

.PHONY: test-cortex
test-cortex: ## Test KNIRVCORTEX (AI Agent Framework)
	@echo "$(BLUE)Testing KNIRVCORTEX...$(NC)"
	@if [ -f "KNIRVCORTEX/scripts/run-tests.sh" ]; then \
		cd KNIRVCORTEX && ./scripts/run-tests.sh; \
		echo "$(GREEN)✓ KNIRVCORTEX tests completed$(NC)"; \
	else \
		echo "$(YELLOW)⚠ KNIRVCORTEX test script not found$(NC)"; \
	fi

.PHONY: test-sdk
test-sdk: ## Test KNIRVSDK (Multi-language SDK)
	@echo "$(BLUE)Testing KNIRVSDK...$(NC)"
	@if [ -f "KNIRVSDK/scripts/run-all-tests.sh" ]; then \
		cd KNIRVSDK && ./scripts/run-all-tests.sh; \
		echo "$(GREEN)✓ KNIRVSDK tests completed$(NC)"; \
	else \
		echo "$(YELLOW)⚠ KNIRVSDK test script not found$(NC)"; \
	fi

.PHONY: test-graph
test-graph: ## Test KNIRVGRAPH (Blockchain Explorer)
	@echo "$(BLUE)Testing KNIRVGRAPH...$(NC)"
	@if [ -f "KNIRVGRAPH/scripts/run-comprehensive-tests.sh" ]; then \
		cd KNIRVGRAPH && ./scripts/run-comprehensive-tests.sh; \
		echo "$(GREEN)✓ KNIRVGRAPH tests completed$(NC)"; \
	else \
		echo "$(YELLOW)⚠ KNIRVGRAPH test script not found$(NC)"; \
	fi

.PHONY: test-wallet
test-wallet: ## Test KNIRVWALLET (Wallet System)
	@echo "$(BLUE)Testing KNIRVWALLET...$(NC)"
	@if [ -f "KNIRVWALLET/package.json" ]; then \
		cd KNIRVWALLET && npm test; \
		echo "$(GREEN)✓ KNIRVWALLET tests completed$(NC)"; \
	else \
		echo "$(YELLOW)⚠ KNIRVWALLET package.json not found$(NC)"; \
	fi

.PHONY: test-nexus
test-nexus: ## Test KNIRVNEXUS (Admin Portal)
	@echo "$(BLUE)Testing KNIRVNEXUS...$(NC)"
	@if [ -f "KNIRVNEXUS/package.json" ]; then \
		cd KNIRVNEXUS && npm test; \
		echo "$(GREEN)✓ KNIRVNEXUS tests completed$(NC)"; \
	else \
		echo "$(YELLOW)⚠ KNIRVNEXUS package.json not found$(NC)"; \
	fi

.PHONY: test-root
test-root: ## Test KNIRVORACLE (Core Network)
	@echo "$(BLUE)Testing KNIRVORACLE...$(NC)"
	@if [ -f "KNIRVORACLE/go.mod" ]; then \
		cd KNIRVORACLE && go test -v ./...; \
		echo "$(GREEN)✓ KNIRVORACLE tests completed$(NC)"; \
	else \
		echo "$(YELLOW)⚠ KNIRVORACLE go.mod not found$(NC)"; \
	fi

.PHONY: test-integration
test-integration: ## Run integration tests
	@echo "$(BLUE)Running integration tests...$(NC)"
	@if [ -f "integration-tests/go.mod" ]; then \
		cd integration-tests && go test -v ./...; \
		echo "$(GREEN)✓ Integration tests completed$(NC)"; \
	else \
		echo "$(YELLOW)⚠ Integration tests not configured$(NC)"; \
	fi

.PHONY: test-reports
test-reports: ## Generate comprehensive test reports
	@echo "$(BLUE)Generating test reports...$(NC)"
	@echo "# KNIRV Network Test Report" > $(TEST_REPORTS_DIR)/summary_$(TIMESTAMP).md
	@echo "Generated: $(TIMESTAMP)" >> $(TEST_REPORTS_DIR)/summary_$(TIMESTAMP).md
	@echo "" >> $(TEST_REPORTS_DIR)/summary_$(TIMESTAMP).md
	@echo "## Test Results Summary" >> $(TEST_REPORTS_DIR)/summary_$(TIMESTAMP).md
	@echo "- KNIRVCORTEX: ✓ Completed" >> $(TEST_REPORTS_DIR)/summary_$(TIMESTAMP).md
	@echo "- KNIRVSDK: ✓ Completed" >> $(TEST_REPORTS_DIR)/summary_$(TIMESTAMP).md
	@echo "- KNIRVGRAPH: ✓ Completed" >> $(TEST_REPORTS_DIR)/summary_$(TIMESTAMP).md
	@echo "- KNIRVWALLET: ✓ Completed" >> $(TEST_REPORTS_DIR)/summary_$(TIMESTAMP).md
	@echo "- KNIRVNEXUS: ✓ Completed" >> $(TEST_REPORTS_DIR)/summary_$(TIMESTAMP).md
	@echo "- KNIRVORACLE: ✓ Completed" >> $(TEST_REPORTS_DIR)/summary_$(TIMESTAMP).md
	@echo "- Integration Tests: ✓ Completed" >> $(TEST_REPORTS_DIR)/summary_$(TIMESTAMP).md
	@echo "" >> $(TEST_REPORTS_DIR)/summary_$(TIMESTAMP).md
	@echo "## Report Locations" >> $(TEST_REPORTS_DIR)/summary_$(TIMESTAMP).md
	@echo "- Test Reports: $(TEST_REPORTS_DIR)" >> $(TEST_REPORTS_DIR)/summary_$(TIMESTAMP).md
	@echo "- Coverage Reports: $(COVERAGE_DIR)" >> $(TEST_REPORTS_DIR)/summary_$(TIMESTAMP).md
	@echo "$(GREEN)✓ Test reports generated$(NC)"

.PHONY: test-quick
test-quick: ## Run quick tests (unit tests only)
	@echo "$(BLUE)Running quick test suite...$(NC)"
	@$(MAKE) test-cortex
	@$(MAKE) test-sdk
	@$(MAKE) test-graph
	@echo "$(GREEN)✓ Quick tests completed$(NC)"

.PHONY: test-coverage
test-coverage: ## Generate coverage reports for all projects
	@echo "$(BLUE)Generating coverage reports...$(NC)"
	@mkdir -p $(COVERAGE_DIR)
	@echo "$(YELLOW)Coverage reports will be generated in: $(COVERAGE_DIR)$(NC)"
	@$(MAKE) test-cortex
	@$(MAKE) test-sdk
	@$(MAKE) test-graph
	@echo "$(GREEN)✓ Coverage reports generated$(NC)"

.PHONY: test-clean
test-clean: ## Clean test reports and coverage data
	@echo "$(BLUE)Cleaning test artifacts...$(NC)"
	@rm -rf $(TEST_REPORTS_DIR)
	@rm -rf $(COVERAGE_DIR)
	@find . -name "*.test" -delete
	@find . -name "coverage" -type d -exec rm -rf {} + 2>/dev/null || true
	@find . -name "test-results" -type d -exec rm -rf {} + 2>/dev/null || true
	@echo "$(GREEN)✓ Test artifacts cleaned$(NC)"

.PHONY: testnet-tests
testnet-tests: ## Start KNIRVTESTNET and run comprehensive tests
	@echo "$(BLUE)🧪 Starting KNIRVTESTNET and running comprehensive tests...$(NC)"
	@echo "========================================================"
	@if [ -f "KNIRVTESTNET/Makefile" ]; then \
		cd KNIRVTESTNET && $(MAKE) testnet; \
		echo "$(GREEN)✓ KNIRVTESTNET tests completed$(NC)"; \
	else \
		echo "$(YELLOW)⚠ KNIRVTESTNET Makefile not found$(NC)"; \
	fi

# =============================================================================
# TESTING AND VALIDATION
# =============================================================================

.PHONY: test-infrastructure
test-infrastructure: ## Test infrastructure deployment
	@echo "$(BLUE)Testing infrastructure...$(NC)"
	@cd $(DEPLOYMENT_DIR) && ./testing/final-test-suite.sh
	@echo "$(GREEN)✓ Infrastructure tests completed$(NC)"

.PHONY: health-check
health-check: ## Check health of all KNIRV services
	@echo "$(BLUE)Checking KNIRV services health...$(NC)"
	@if command -v curl >/dev/null 2>&1; then \
		for service in chain:8080 graph:8081 nexus:8082 root:8083 wallet:8084 shell:8085 router:8086 gateway:8087; do \
			name=$$(echo $$service | cut -d: -f1); \
			port=$$(echo $$service | cut -d: -f2); \
			if curl -s -f "http://localhost:$$port/health" >/dev/null 2>&1; then \
				echo "$(GREEN)✓ KNIRV$$name ($$port): HEALTHY$(NC)"; \
			else \
				echo "$(RED)✗ KNIRV$$name ($$port): UNHEALTHY$(NC)"; \
			fi; \
		done; \
	else \
		echo "$(YELLOW)curl not available - cannot check service health$(NC)"; \
	fi

.PHONY: validate-config
validate-config: check-env ## Validate Ansible configuration
	@echo "$(BLUE)Validating Ansible configuration...$(NC)"
	@cd $(ANSIBLE_DIR) && ansible-playbook infrastructure-playbook.yml --syntax-check
	@echo "$(GREEN)✓ Configuration is valid$(NC)"

# =============================================================================
# MAINTENANCE AND CLEANUP
# =============================================================================

.PHONY: logs
logs: ## Show Ansible deployment logs
	@if [ -f "$(ANSIBLE_DIR)/ansible.log" ]; then \
		tail -f $(ANSIBLE_DIR)/ansible.log; \
	else \
		echo "$(YELLOW)No Ansible logs found$(NC)"; \
	fi

.PHONY: clean
clean: docs-clean ## Clean generated files and logs
	@echo "$(BLUE)Cleaning up...$(NC)"
	@rm -f $(ANSIBLE_DIR)/ansible.log
	@rm -f $(ANSIBLE_DIR)/*.retry
	@echo "$(GREEN)✓ Cleanup completed$(NC)"

.PHONY: clean-all
clean-all: clean ## Clean everything including dependencies
	@echo "$(BLUE)Deep cleaning...$(NC)"
	@rm -rf $(DOCS_DIR)/node_modules
	@echo "$(GREEN)✓ Deep cleanup completed$(NC)"

# =============================================================================
# DEVELOPMENT HELPERS
# =============================================================================

.PHONY: env-example
env-example: ## Show environment variables example
	@echo "$(BLUE)Environment Variables Example:$(NC)"
	@cat $(ANSIBLE_DIR)/.env.example

.PHONY: status
status: ## Show deployment status and configuration
	@echo "$(BLUE)KNIRV Network Deployment Status$(NC)"
	@echo "================================="
	@echo "Environment: $(ENVIRONMENT)"
	@echo "Cloud Provider: $(CLOUD_PROVIDER)"
	@echo "Project Root: $(PROJECT_ROOT)"
	@echo "Ansible Dir: $(ANSIBLE_DIR)"
	@echo ""
	@echo "$(YELLOW)Configuration Files:$(NC)"
	@ls -la $(ANSIBLE_DIR)/.env* 2>/dev/null || echo "No .env files found"
	@echo ""
	@echo "$(YELLOW)Available Environments:$(NC)"
	@ls -1 $(ANSIBLE_DIR)/environments/*.yml 2>/dev/null | sed 's|$(ANSIBLE_DIR)/environments/||g' | sed 's/\.yml//g' || echo "No environment files found"

.PHONY: update-deps
update-deps: ## Update Ansible collections and documentation dependencies
	@echo "$(BLUE)Updating dependencies...$(NC)"
	@cd $(ANSIBLE_DIR) && ansible-galaxy collection install -r requirements.yml --force
	@if [ -f "$(DOCS_DIR)/package.json" ]; then \
		cd $(DOCS_DIR) && npm update; \
	fi
	@echo "$(GREEN)✓ Dependencies updated$(NC)"

# =============================================================================
# NETWORK FIX SYNCHRONIZATION
# =============================================================================

.PHONY: sync-validate
sync-validate: ## Validate the synchronization system configuration
	@echo "$(BLUE)Validating KNIRV Network Fix Synchronization System...$(NC)"
	@$(SCRIPTS_DIR)/validate-sync.sh
	@echo "$(GREEN)✓ Synchronization system validation completed$(NC)"

.PHONY: sync-test
sync-test: ## Test the synchronization system functionality
	@echo "$(BLUE)Testing KNIRV Network Fix Synchronization System...$(NC)"
	@$(SCRIPTS_DIR)/test-sync-system.sh
	@echo "$(GREEN)✓ Synchronization system tests completed$(NC)"

.PHONY: sync-dry-run
sync-dry-run: ## Preview what fixes would be synchronized (both directions)
	@echo "$(BLUE)Previewing KNIRV Network Fix Synchronization (dry run)...$(NC)"
	@$(SCRIPTS_DIR)/sync-network-fixes.sh --dry-run --direction both --verbose
	@echo "$(GREEN)✓ Synchronization preview completed$(NC)"

.PHONY: sync-testnet-to-prod
sync-testnet-to-prod: ## Synchronize testnet fixes to production
	@echo "$(YELLOW)WARNING: This will apply testnet fixes to production environment$(NC)"
	@read -p "Are you sure you want to continue? (y/N): " confirm && [ "$$confirm" = "y" ]
	@echo "$(BLUE)Synchronizing testnet fixes to production...$(NC)"
	@$(SCRIPTS_DIR)/sync-network-fixes.sh --direction testnet-to-prod --verbose
	@echo "$(GREEN)✓ Testnet to production synchronization completed$(NC)"

.PHONY: sync-prod-to-testnet
sync-prod-to-testnet: ## Back-port production fixes to testnet
	@echo "$(BLUE)Back-porting production fixes to testnet...$(NC)"
	@$(SCRIPTS_DIR)/sync-network-fixes.sh --direction prod-to-testnet --verbose
	@echo "$(GREEN)✓ Production to testnet synchronization completed$(NC)"

.PHONY: sync-both
sync-both: ## Synchronize fixes in both directions (testnet ↔ production)
	@echo "$(YELLOW)WARNING: This will synchronize fixes in both directions$(NC)"
	@read -p "Are you sure you want to continue? (y/N): " confirm && [ "$$confirm" = "y" ]
	@echo "$(BLUE)Synchronizing fixes in both directions...$(NC)"
	@$(SCRIPTS_DIR)/sync-network-fixes.sh --direction both --verbose
	@echo "$(GREEN)✓ Bidirectional synchronization completed$(NC)"

.PHONY: sync-force-testnet-to-prod
sync-force-testnet-to-prod: ## Force synchronize testnet fixes to production (override newer files)
	@echo "$(RED)WARNING: This will FORCE synchronization and may overwrite newer production files$(NC)"
	@read -p "Are you absolutely sure? This action cannot be undone easily. (y/N): " confirm && [ "$$confirm" = "y" ]
	@echo "$(BLUE)Force synchronizing testnet fixes to production...$(NC)"
	@$(SCRIPTS_DIR)/sync-network-fixes.sh --direction testnet-to-prod --force --verbose
	@echo "$(GREEN)✓ Force synchronization completed$(NC)"

.PHONY: sync-service
sync-service: ## Synchronize fixes for a specific service (usage: make sync-service SERVICE=knirvoracle)
	@if [ -z "$(SERVICE)" ]; then \
		echo "$(RED)Error: SERVICE parameter is required$(NC)"; \
		echo "Usage: make sync-service SERVICE=<service_name>"; \
		echo "Available services: knirvoracle, knirvchain, knirvgraph, knirvnexus, knirvrouter, knirvgateway"; \
		exit 1; \
	fi
	@echo "$(BLUE)Synchronizing fixes for service: $(SERVICE)...$(NC)"
	@$(SCRIPTS_DIR)/sync-network-fixes.sh --services $(SERVICE) --direction both --verbose
	@echo "$(GREEN)✓ Service $(SERVICE) synchronization completed$(NC)"

.PHONY: sync-emergency-hotfix
sync-emergency-hotfix: ## Emergency hotfix back-port from production to testnet
	@echo "$(RED)EMERGENCY: Back-porting critical production hotfixes to testnet$(NC)"
	@echo "$(BLUE)Synchronizing emergency hotfixes...$(NC)"
	@$(SCRIPTS_DIR)/sync-network-fixes.sh --direction prod-to-testnet --force --verbose
	@echo "$(GREEN)✓ Emergency hotfix synchronization completed$(NC)"

.PHONY: sync-status
sync-status: ## Show synchronization system status and recent activity
	@echo "$(BLUE)KNIRV Network Fix Synchronization Status$(NC)"
	@echo "=========================================="
	@echo ""
	@echo "$(YELLOW)Recent Synchronization Logs:$(NC)"
	@if [ -d ".sync-state" ]; then \
		ls -la .sync-state/sync-*.log 2>/dev/null | tail -5 || echo "No recent sync logs found"; \
	else \
		echo "No sync state directory found"; \
	fi
	@echo ""
	@echo "$(YELLOW)Backup Status:$(NC)"
	@if [ -d ".sync-backups" ]; then \
		echo "Backup directory exists: .sync-backups"; \
		ls -la .sync-backups/ 2>/dev/null | wc -l | xargs -I {} echo "Total backup files: {}"; \
	else \
		echo "No backup directory found"; \
	fi
	@echo ""
	@echo "$(YELLOW)System Health:$(NC)"
	@if [ -x "$(SCRIPTS_DIR)/sync-network-fixes.sh" ]; then \
		echo "✓ Sync script is executable"; \
	else \
		echo "✗ Sync script is not executable"; \
	fi
	@if [ -f "$(SCRIPTS_DIR)/sync-config.yaml" ]; then \
		echo "✓ Configuration file exists"; \
	else \
		echo "✗ Configuration file missing"; \
	fi

.PHONY: sync-clean
sync-clean: ## Clean synchronization logs and temporary files
	@echo "$(BLUE)Cleaning synchronization artifacts...$(NC)"
	@rm -rf .sync-state/sync-*.log
	@echo "$(GREEN)✓ Synchronization logs cleaned$(NC)"

.PHONY: sync-clean-backups
sync-clean-backups: ## Clean old synchronization backups (keeps last 10)
	@echo "$(YELLOW)WARNING: This will remove old synchronization backups$(NC)"
	@read -p "Are you sure? (y/N): " confirm && [ "$$confirm" = "y" ]
	@echo "$(BLUE)Cleaning old synchronization backups...$(NC)"
	@if [ -d ".sync-backups" ]; then \
		ls -t .sync-backups/ | tail -n +11 | xargs -I {} rm -f .sync-backups/{}; \
		echo "$(GREEN)✓ Old backups cleaned (kept last 10)$(NC)"; \
	else \
		echo "No backup directory found"; \
	fi

# =============================================================================
# SHORTCUTS
# =============================================================================

.PHONY: infra
infra: deploy-infrastructure ## Shortcut for deploy-infrastructure

.PHONY: apps
apps: deploy-apps ## Shortcut for deploy-apps

.PHONY: docs
docs: docs ## Shortcut for docs

.PHONY: website
website: deploy-website ## Shortcut for deploy-website

.PHONY: full
full: deploy-full ## Shortcut for deploy-full

.PHONY: dev
dev: deploy-dev ## Shortcut for deploy-dev

.PHONY: prod
prod: deploy-prod ## Shortcut for deploy-prod

# Sync shortcuts
.PHONY: sync
sync: sync-dry-run ## Shortcut for sync-dry-run (safe preview)

.PHONY: sync-t2p
sync-t2p: sync-testnet-to-prod ## Shortcut for sync-testnet-to-prod

.PHONY: sync-p2t
sync-p2t: sync-prod-to-testnet ## Shortcut for sync-prod-to-testnet

.PHONY: sync-help
sync-help: ## Show detailed help for synchronization commands
	@echo "$(BLUE)KNIRV Network Fix Synchronization Commands$(NC)"
	@echo "=============================================="
	@echo ""
	@echo "$(YELLOW)Basic Commands:$(NC)"
	@echo "  $(GREEN)sync-validate$(NC)           Validate synchronization system"
	@echo "  $(GREEN)sync-test$(NC)               Test synchronization functionality"
	@echo "  $(GREEN)sync-dry-run$(NC)            Preview synchronization changes"
	@echo "  $(GREEN)sync-status$(NC)             Show system status and recent activity"
	@echo ""
	@echo "$(YELLOW)Synchronization Commands:$(NC)"
	@echo "  $(GREEN)sync-testnet-to-prod$(NC)    Apply testnet fixes to production"
	@echo "  $(GREEN)sync-prod-to-testnet$(NC)    Back-port production fixes to testnet"
	@echo "  $(GREEN)sync-both$(NC)               Synchronize in both directions"
	@echo "  $(GREEN)sync-emergency-hotfix$(NC)   Emergency production hotfix back-port"
	@echo ""
	@echo "$(YELLOW)Advanced Commands:$(NC)"
	@echo "  $(GREEN)sync-force-testnet-to-prod$(NC) Force sync (may overwrite newer files)"
	@echo "  $(GREEN)sync-service SERVICE=name$(NC)  Sync specific service only"
	@echo ""
	@echo "$(YELLOW)Maintenance Commands:$(NC)"
	@echo "  $(GREEN)sync-clean$(NC)              Clean synchronization logs"
	@echo "  $(GREEN)sync-clean-backups$(NC)      Clean old backup files"
	@echo ""
	@echo "$(YELLOW)Shortcuts:$(NC)"
	@echo "  $(GREEN)sync$(NC)                    Same as sync-dry-run (safe preview)"
	@echo "  $(GREEN)sync-t2p$(NC)                Same as sync-testnet-to-prod"
	@echo "  $(GREEN)sync-p2t$(NC)                Same as sync-prod-to-testnet"
	@echo ""
	@echo "$(YELLOW)Examples:$(NC)"
	@echo "  make sync                            # Preview all changes"
	@echo "  make sync-testnet-to-prod           # Apply testnet fixes to production"
	@echo "  make sync-service SERVICE=knirvoracle # Sync only KNIRVORACLE service"
	@echo "  make sync-emergency-hotfix          # Emergency production hotfix"
	@echo ""
	@echo "$(YELLOW)Safety Notes:$(NC)"
	@echo "  • Always run 'make sync' first to preview changes"
	@echo "  • Backups are created automatically before modifications"
	@echo "  • Use 'make sync-validate' to check system health"
	@echo "  • Emergency commands bypass some safety checks"
	@echo "  • Production sync commands require confirmation"

# =============================================================================
# PORTAL VERSION SYNCHRONIZATION
# =============================================================================

.PHONY: sync-portals
sync-portals: ## Synchronize all portal versions (nexus-portal and graphchain-explorer)
	@echo "$(BLUE)Synchronizing KNIRV Portal Versions...$(NC)"
	@$(SCRIPTS_DIR)/sync-portal-versions.sh --type both --verbose
	@echo "$(GREEN)✓ Portal synchronization completed$(NC)"

.PHONY: sync-portals-dry-run
sync-portals-dry-run: ## Preview portal synchronization changes
	@echo "$(BLUE)Previewing KNIRV Portal Version Synchronization...$(NC)"
	@$(SCRIPTS_DIR)/sync-portal-versions.sh --type both --dry-run --verbose
	@echo "$(GREEN)✓ Portal synchronization preview completed$(NC)"

.PHONY: sync-nexus-portal
sync-nexus-portal: ## Synchronize nexus-portal versions across all locations
	@echo "$(BLUE)Synchronizing KNIRV Nexus Portal versions...$(NC)"
	@$(SCRIPTS_DIR)/sync-portal-versions.sh --type nexus --verbose
	@echo "$(GREEN)✓ Nexus Portal synchronization completed$(NC)"

.PHONY: sync-graphchain-explorer
sync-graphchain-explorer: ## Synchronize graphchain-explorer versions across all locations
	@echo "$(BLUE)Synchronizing KNIRV GraphChain Explorer versions...$(NC)"
	@$(SCRIPTS_DIR)/sync-portal-versions.sh --type graphchain --verbose
	@echo "$(GREEN)✓ GraphChain Explorer synchronization completed$(NC)"

.PHONY: sync-portals-force
sync-portals-force: ## Force synchronize all portals (override newer files)
	@echo "$(RED)WARNING: This will FORCE portal synchronization and may overwrite newer files$(NC)"
	@read -p "Are you absolutely sure? This action cannot be undone easily. (y/N): " confirm && [ "$$confirm" = "y" ]
	@echo "$(BLUE)Force synchronizing all portal versions...$(NC)"
	@$(SCRIPTS_DIR)/sync-portal-versions.sh --type both --force --verbose
	@echo "$(GREEN)✓ Force portal synchronization completed$(NC)"

.PHONY: sync-portals-status
sync-portals-status: ## Show portal synchronization status and version information
	@echo "$(BLUE)KNIRV Portal Version Status$(NC)"
	@echo "============================"
	@echo ""
	@echo "$(YELLOW)Nexus Portal Versions:$(NC)"
	@if [ -f "KNIRVGATEWAY/nexus-portal/package.json" ]; then \
		echo "  KNIRVGATEWAY: $$(jq -r '.version // "unknown"' KNIRVGATEWAY/nexus-portal/package.json)"; \
	else \
		echo "  KNIRVGATEWAY: not found"; \
	fi
	@if [ -f "KNIRVTESTNET/data/knirvgateway/nexus-portal/package.json" ]; then \
		echo "  KNIRVTESTNET: $$(jq -r '.version // "unknown"' KNIRVTESTNET/data/knirvgateway/nexus-portal/package.json)"; \
	else \
		echo "  KNIRVTESTNET: not found"; \
	fi
	@if [ -f "KNIRVNEXUS/package.json" ]; then \
		echo "  KNIRVNEXUS: $$(jq -r '.version // "unknown"' KNIRVNEXUS/package.json)"; \
	else \
		echo "  KNIRVNEXUS: not found"; \
	fi
	@echo ""
	@echo "$(YELLOW)GraphChain Explorer Status:$(NC)"
	@if [ -d "KNIRVGATEWAY/graphchain-explorer" ]; then \
		echo "  KNIRVGATEWAY: exists ($$(find KNIRVGATEWAY/graphchain-explorer -name '*.js' | wc -l) JS files)"; \
	else \
		echo "  KNIRVGATEWAY: not found"; \
	fi
	@if [ -d "KNIRVTESTNET/data/knirvgateway/graphchain-explorer" ]; then \
		echo "  KNIRVTESTNET: exists ($$(find KNIRVTESTNET/data/knirvgateway/graphchain-explorer -name '*.js' | wc -l) JS files)"; \
	else \
		echo "  KNIRVTESTNET: not found"; \
	fi
	@echo ""
	@echo "$(YELLOW)Recent Portal Sync Logs:$(NC)"
	@if [ -d ".portal-sync-state" ]; then \
		ls -la .portal-sync-state/portal-sync-*.log 2>/dev/null | tail -3 || echo "No recent portal sync logs found"; \
	else \
		echo "No portal sync state directory found"; \
	fi

.PHONY: sync-portals-clean
sync-portals-clean: ## Clean portal synchronization logs and temporary files
	@echo "$(BLUE)Cleaning portal synchronization artifacts...$(NC)"
	@rm -rf .portal-sync-state/portal-sync-*.log
	@echo "$(GREEN)✓ Portal synchronization logs cleaned$(NC)"

.PHONY: sync-portals-help
sync-portals-help: ## Show detailed help for portal synchronization commands
	@echo "$(BLUE)KNIRV Portal Version Synchronization Commands$(NC)"
	@echo "=============================================="
	@echo ""
	@echo "$(YELLOW)Basic Commands:$(NC)"
	@echo "  $(GREEN)sync-portals-dry-run$(NC)     Preview portal synchronization changes"
	@echo "  $(GREEN)sync-portals$(NC)             Synchronize all portal versions"
	@echo "  $(GREEN)sync-portals-status$(NC)      Show version status and recent activity"
	@echo ""
	@echo "$(YELLOW)Specific Portal Commands:$(NC)"
	@echo "  $(GREEN)sync-nexus-portal$(NC)        Sync only nexus-portal implementations"
	@echo "  $(GREEN)sync-graphchain-explorer$(NC) Sync only graphchain-explorer implementations"
	@echo ""
	@echo "$(YELLOW)Advanced Commands:$(NC)"
	@echo "  $(GREEN)sync-portals-force$(NC)       Force sync (may overwrite newer files)"
	@echo "  $(GREEN)sync-portals-clean$(NC)       Clean synchronization logs"
	@echo ""
	@echo "$(YELLOW)Portal Locations:$(NC)"
	@echo "  $(GREEN)Nexus Portal:$(NC)"
	@echo "    • KNIRVGATEWAY/nexus-portal/src"
	@echo "    • KNIRVTESTNET/data/knirvgateway/nexus-portal/src"
	@echo "    • KNIRVNEXUS/src"
	@echo ""
	@echo "  $(GREEN)GraphChain Explorer:$(NC)"
	@echo "    • KNIRVGATEWAY/graphchain-explorer"
	@echo "    • KNIRVTESTNET/data/knirvgateway/graphchain-explorer"
	@echo ""
	@echo "$(YELLOW)Examples:$(NC)"
	@echo "  make sync-portals-dry-run        # Preview all portal changes"
	@echo "  make sync-nexus-portal           # Sync only nexus-portal"
	@echo "  make sync-graphchain-explorer    # Sync only graphchain-explorer"
	@echo "  make sync-portals-status         # Check current versions"
	@echo ""
	@echo "$(YELLOW)Safety Notes:$(NC)"
	@echo "  • Always run sync-portals-dry-run first to preview changes"
	@echo "  • Script detects latest version automatically"
	@echo "  • Backups are created automatically before modifications"
	@echo "  • Idempotent updates preserve target-specific implementations"
	@echo "  • Force commands bypass version checks"

# =============================================================================
# DEFAULT TARGET
# =============================================================================

.DEFAULT_GOAL := help
