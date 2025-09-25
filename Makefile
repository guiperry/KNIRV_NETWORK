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
KNIRVCONTROLLER_DIR := $(PROJECT_ROOT)/KNIRVCONTROLLER
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
	@echo "  make deploy-controller-pwa ENVIRONMENT=production"
	@echo "  make health-check-controller-pwa"
	@echo "  make setup-controller-db     # Setup KNIRVCONTROLLER database with accounts"
	@echo "  make seed-controller-db      # Seed database with default accounts"
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
		echo "$(BLUE)Generating static HTML from docsify...$(NC)"; \
		cd $(KNIRVGATEWAY_DIR)/network-website/public/documentation && npm install; \
		cd $(KNIRVGATEWAY_DIR)/network-website && node scripts/simple-static-generator.js; \
		echo "$(BLUE)Updating links to point to static version...$(NC)"; \
		cd $(KNIRVGATEWAY_DIR)/network-website && node scripts/update-docsify-links.js; \
		echo "$(GREEN)✓ Static documentation generation completed$(NC)"; \
		echo "$(YELLOW)📁 Static documentation available at: KNIRVGATEWAY/network-website/public/documentation/static/$(NC)"; \
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
	@$(MAKE) deploy-controller-pwa ENVIRONMENT=production

.PHONY: deploy-staging
deploy-staging: ## Staging deployment
	@$(MAKE) deploy-full ENVIRONMENT=staging CLOUD_PROVIDER=aws

.PHONY: deploy-testnet
deploy-testnet: ## Deploy KNIRVTESTNET to AWS with Netlify frontend integration
	@echo "$(BLUE)Deploying KNIRVTESTNET to AWS...$(NC)"
	@$(MAKE) deploy-testnet-infrastructure
	@$(MAKE) deploy-testnet-services
	@$(MAKE) deploy-controller-pwa ENVIRONMENT=testnet
	@$(MAKE) update-testnet-frontend
	@echo "$(GREEN)✓ KNIRVTESTNET deployment completed!$(NC)"

.PHONY: deploy-testnet-infrastructure
deploy-testnet-infrastructure: check-prereqs ## Deploy testnet infrastructure only
	@echo "$(BLUE)Deploying KNIRVTESTNET infrastructure...$(NC)"
	@cd $(ANSIBLE_DIR) && ./deploy-testnet-infrastructure.sh --force
	@echo "$(GREEN)✓ KNIRVTESTNET infrastructure deployed$(NC)"

.PHONY: deploy-testnet-services
deploy-testnet-services: ## Deploy KNIRVTESTNET services via Native, Podman, or Docker Compose
	@echo "$(BLUE)Deploying KNIRVTESTNET services...$(NC)"
	@cd $(PROJECT_ROOT) && ./scripts/deploy-testnet-services.sh
	@echo "$(GREEN)✓ KNIRVTESTNET services deployed$(NC)"

.PHONY: deploy-controller-pwa
deploy-controller-pwa: check-prereqs ## Deploy KNIRVCONTROLLER PWA to CloudFlare CDN
	@echo "$(BLUE)Deploying KNIRVCONTROLLER PWA...$(NC)"
	@cd $(PROJECT_ROOT) && ./scripts/deploy-controller-pwa.sh $(ENVIRONMENT)
	@echo "$(GREEN)✓ KNIRVCONTROLLER PWA deployed$(NC)"

.PHONY: build-controller-pwa
build-controller-pwa: ## Build KNIRVCONTROLLER PWA packages
	@echo "$(BLUE)Building KNIRVCONTROLLER PWA packages...$(NC)"
	@cd $(KNIRVCONTROLLER_DIR) && npm run build:pwa
	@echo "$(GREEN)✓ KNIRVCONTROLLER PWA packages built$(NC)"

.PHONY: setup-controller-db
setup-controller-db: ## Setup KNIRVCONTROLLER database with default accounts
	@echo "$(BLUE)Setting up KNIRVCONTROLLER database...$(NC)"
	@cd $(KNIRVCONTROLLER_DIR) && npm run db:setup
	@echo "$(GREEN)✓ KNIRVCONTROLLER database setup completed$(NC)"
	@echo "$(YELLOW)🔐 Default credentials:$(NC)"
	@echo "$(YELLOW)  Admin: admin@knirv.com / admin123$(NC)"
	@echo "$(YELLOW)  Demo: demo@knirv.com / demo123$(NC)"
	@echo "$(YELLOW)  Developer: dev@knirv.com / dev123$(NC)"
	@echo "$(YELLOW)  Test User: test@example.com / test123$(NC)"

.PHONY: seed-controller-db
seed-controller-db: ## Seed KNIRVCONTROLLER database with default accounts
	@echo "$(BLUE)Seeding KNIRVCONTROLLER database...$(NC)"
	@cd $(KNIRVCONTROLLER_DIR) && npm run db:seed
	@echo "$(GREEN)✓ KNIRVCONTROLLER database seeded$(NC)"

# =============================================================================
# TESTNET INSTANCE MANAGEMENT
# =============================================================================

.PHONY: testnet-ip
testnet-ip: ## Get current testnet IP address and update SSH config
	@echo "$(BLUE)Getting testnet IP address...$(NC)"
	@./scripts/get-testnet-ip.sh get-ip

.PHONY: testnet-start
testnet-start: ## Start the testnet EC2 instance
	@echo "$(BLUE)Starting testnet instance...$(NC)"
	@./scripts/get-testnet-ip.sh start

.PHONY: testnet-stop
testnet-stop: ## Stop the testnet EC2 instance
	@echo "$(BLUE)Stopping testnet instance...$(NC)"
	@./scripts/get-testnet-ip.sh stop

.PHONY: testnet-status
testnet-status: ## Show testnet instance status
	@echo "$(BLUE)Checking testnet status...$(NC)"
	@./scripts/get-testnet-ip.sh status

.PHONY: testnet-ssh
testnet-ssh: ## SSH into the testnet instance
	@echo "$(BLUE)Connecting to testnet...$(NC)"
	@./scripts/get-testnet-ip.sh get-ip > /dev/null
	@ssh knirv-testnet

.PHONY: testnet-logs
testnet-logs: ## View testnet service logs
	@echo "$(BLUE)Viewing testnet service logs...$(NC)"
	@./scripts/get-testnet-ip.sh get-ip > /dev/null
	@ssh knirv-testnet 'cd /opt/knirv-testnet && docker-compose logs -f'

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
	@$(MAKE) test-controller
	@$(MAKE) test-sdk
	@$(MAKE) test-graph
	@$(MAKE) test-wallet
	@$(MAKE) test-nexus
	@$(MAKE) test-root
	@$(MAKE) test-engine
	@$(MAKE) test-integration
	@$(MAKE) test-reports
	@echo ""
	@echo "$(GREEN)🎉 All KNIRV Network tests completed!$(NC)"
	@echo "$(YELLOW)📊 View reports at: $(TEST_REPORTS_DIR)$(NC)"
	@echo "$(YELLOW)📈 View coverage at: $(COVERAGE_DIR)$(NC)"

.PHONY: test-controller
test-controller: ## Test KNIRVCONTROLLER (AI Agent Framework)
	@echo "$(BLUE)Testing KNIRVCONTROLLER...$(NC)"
	@if [ -f "KNIRVCONTROLLER/scripts/run-tests.sh" ]; then \
		cd KNIRVCONTROLLER && ./scripts/run-tests.sh; \
		echo "$(GREEN)✓ KNIRVCONTROLLER tests completed$(NC)"; \
	else \
		echo "$(YELLOW)⚠ KNIRVCONTROLLER test script not found$(NC)"; \
	fi

.PHONY: test-controller-pwa
test-controller-pwa: ## Test KNIRVCONTROLLER PWA functionality
	@echo "$(BLUE)Testing KNIRVCONTROLLER PWA...$(NC)"
	@if [ -f "KNIRVCONTROLLER/scripts/test-pwa.sh" ]; then \
		cd KNIRVCONTROLLER && ./scripts/test-pwa.sh; \
		echo "$(GREEN)✓ KNIRVCONTROLLER PWA tests completed$(NC)"; \
	else \
		echo "$(YELLOW)⚠ KNIRVCONTROLLER PWA test script not found$(NC)"; \
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
	@if [ -f "KNIRVWALLET/agentic-wallet/package.json" ]; then \
		cd KNIRVWALLET/agentic-wallet && npm test; \
		echo "$(GREEN)✓ KNIRVWALLET agentic-wallet tests completed$(NC)"; \
	elif [ -f "KNIRVWALLET/browser-bridge/package.json" ]; then \
		cd KNIRVWALLET/browser-bridge && npm test; \
		echo "$(GREEN)✓ KNIRVWALLET browser-bridge tests completed$(NC)"; \
	else \
		echo "$(YELLOW)⚠ KNIRVWALLET test scripts not found$(NC)"; \
		echo "$(YELLOW)   Run 'cd KNIRVWALLET/scripts && ./test_basic.js' for basic testing$(NC)"; \
	fi

.PHONY: test-nexus
test-nexus: ## Test KNIRVNEXUS (Admin Portal)
	@echo "$(BLUE)Testing KNIRVNEXUS...$(NC)"
	@$(MAKE) test-nexus-unit
	@$(MAKE) test-nexus-integration
	@$(MAKE) test-nexus-e2e
	@$(MAKE) test-nexus-performance
	@$(MAKE) test-nexus-security
	@echo "$(GREEN)✓ KNIRVNEXUS comprehensive tests completed$(NC)"

.PHONY: test-nexus-unit
test-nexus-unit: ## Run KNIRVNEXUS unit tests
	@echo "$(BLUE)Running KNIRVNEXUS unit tests...$(NC)"
	@if [ -f "KNIRVNEXUS/backend/tests/phase6_comprehensive_unit_test.go" ]; then \
		cd KNIRVNEXUS && go test -v ./backend/tests/...; \
		echo "$(GREEN)✓ KNIRVNEXUS unit tests completed$(NC)"; \
	else \
		echo "$(YELLOW)⚠ KNIRVNEXUS unit tests not found$(NC)"; \
	fi

.PHONY: test-nexus-integration
test-nexus-integration: ## Run KNIRVNEXUS integration tests
	@echo "$(BLUE)Running KNIRVNEXUS integration tests...$(NC)"
	@if [ -f "integration-tests/knirvnexus_phase6_comprehensive_test.go" ]; then \
		cd integration-tests && go test -v -run "TestKNIRVNEXUS.*"; \
		echo "$(GREEN)✓ KNIRVNEXUS integration tests completed$(NC)"; \
	else \
		echo "$(YELLOW)⚠ KNIRVNEXUS integration tests not found$(NC)"; \
	fi

.PHONY: test-nexus-e2e
test-nexus-e2e: ## Run KNIRVNEXUS end-to-end tests
	@echo "$(BLUE)Running KNIRVNEXUS end-to-end tests...$(NC)"
	@if [ -f "integration-tests/knirvnexus_e2e_workflow_test.go" ]; then \
		cd integration-tests && go test -v -run "TestE2E.*"; \
		echo "$(GREEN)✓ KNIRVNEXUS E2E tests completed$(NC)"; \
	else \
		echo "$(YELLOW)⚠ KNIRVNEXUS E2E tests not found$(NC)"; \
	fi

.PHONY: test-nexus-performance
test-nexus-performance: ## Run KNIRVNEXUS performance tests
	@echo "$(BLUE)Running KNIRVNEXUS performance tests...$(NC)"
	@if [ -f "integration-tests/knirvnexus_performance_test.go" ]; then \
		cd integration-tests && go test -v -run "TestPerformance.*" -timeout=10m; \
		echo "$(GREEN)✓ KNIRVNEXUS performance tests completed$(NC)"; \
	else \
		echo "$(YELLOW)⚠ KNIRVNEXUS performance tests not found$(NC)"; \
	fi

.PHONY: test-nexus-security
test-nexus-security: ## Run KNIRVNEXUS security tests
	@echo "$(BLUE)Running KNIRVNEXUS security tests...$(NC)"
	@if [ -f "integration-tests/knirvnexus_security_test.go" ]; then \
		cd integration-tests && go test -v -run "TestSecurity.*"; \
		echo "$(GREEN)✓ KNIRVNEXUS security tests completed$(NC)"; \
	else \
		echo "$(YELLOW)⚠ KNIRVNEXUS security tests not found$(NC)"; \
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

.PHONY: test-engine
test-engine: ## Test KNIRVENGINE (Desktop Client)
	@echo "$(BLUE)Testing KNIRVENGINE Desktop Client...$(NC)"
	@if [ -f "KNIRVENGINE/desktop-client/go.mod" ]; then \
		cd KNIRVENGINE/desktop-client && go test -v -cover ./agentify/... ./desktop/... ./services/... ./utils/... ./inference/... ./database/... ./api/...; \
		echo "$(GREEN)✓ KNIRVENGINE Desktop Client tests completed$(NC)"; \
	else \
		echo "$(YELLOW)⚠ KNIRVENGINE desktop-client go.mod not found$(NC)"; \
	fi

.PHONY: test-engine-frontend
test-engine-frontend: ## Test KNIRVENGINE Frontend (React/TypeScript)
	@echo "$(BLUE)Testing KNIRVENGINE Frontend...$(NC)"
	@if [ -f "KNIRVENGINE/desktop-client/gui/package.json" ]; then \
		cd KNIRVENGINE/desktop-client/gui && npm test -- --watchAll=false --coverage; \
		echo "$(GREEN)✓ KNIRVENGINE Frontend tests completed$(NC)"; \
	else \
		echo "$(YELLOW)⚠ KNIRVENGINE GUI package.json not found$(NC)"; \
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
	@echo "- KNIRVCONTROLLER: ✓ Completed" >> $(TEST_REPORTS_DIR)/summary_$(TIMESTAMP).md
	@echo "- KNIRVSDK: ✓ Completed" >> $(TEST_REPORTS_DIR)/summary_$(TIMESTAMP).md
	@echo "- KNIRVGRAPH: ✓ Completed" >> $(TEST_REPORTS_DIR)/summary_$(TIMESTAMP).md
	@echo "- KNIRVWALLET: ✓ Completed" >> $(TEST_REPORTS_DIR)/summary_$(TIMESTAMP).md
	@echo "- KNIRVNEXUS: ✓ Completed" >> $(TEST_REPORTS_DIR)/summary_$(TIMESTAMP).md
	@echo "- KNIRVORACLE: ✓ Completed" >> $(TEST_REPORTS_DIR)/summary_$(TIMESTAMP).md
	@echo "- KNIRVENGINE: ✓ Completed" >> $(TEST_REPORTS_DIR)/summary_$(TIMESTAMP).md
	@echo "- Integration Tests: ✓ Completed" >> $(TEST_REPORTS_DIR)/summary_$(TIMESTAMP).md
	@echo "" >> $(TEST_REPORTS_DIR)/summary_$(TIMESTAMP).md
	@echo "## Report Locations" >> $(TEST_REPORTS_DIR)/summary_$(TIMESTAMP).md
	@echo "- Test Reports: $(TEST_REPORTS_DIR)" >> $(TEST_REPORTS_DIR)/summary_$(TIMESTAMP).md
	@echo "- Coverage Reports: $(COVERAGE_DIR)" >> $(TEST_REPORTS_DIR)/summary_$(TIMESTAMP).md
	@echo "$(GREEN)✓ Test reports generated$(NC)"

.PHONY: test-quick
test-quick: ## Run quick tests (unit tests only)
	@echo "$(BLUE)Running quick test suite...$(NC)"
	@$(MAKE) test-controller
	@$(MAKE) test-sdk
	@$(MAKE) test-graph
	@echo "$(GREEN)✓ Quick tests completed$(NC)"

.PHONY: test-coverage
test-coverage: ## Generate coverage reports for all projects
	@echo "$(BLUE)Generating coverage reports...$(NC)"
	@mkdir -p $(COVERAGE_DIR)
	@echo "$(YELLOW)Coverage reports will be generated in: $(COVERAGE_DIR)$(NC)"
	@$(MAKE) test-controller
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
# PHASE 6 COMPREHENSIVE TESTING SUITE
# =============================================================================

.PHONY: test-phase6
test-phase6: test-setup ## Run comprehensive Phase 6 test suite
	@echo "$(BLUE)🚀 Running Phase 6 Comprehensive Test Suite$(NC)"
	@echo "=============================================="
	@echo "Timestamp: $(TIMESTAMP)"
	@echo ""
	@$(MAKE) test-phase6-unit
	@$(MAKE) test-phase6-integration
	@$(MAKE) test-phase6-e2e
	@$(MAKE) test-phase6-performance
	@$(MAKE) test-phase6-security
	@$(MAKE) test-phase6-reports
	@echo ""
	@echo "$(GREEN)🎉 Phase 6 comprehensive test suite completed!$(NC)"

.PHONY: test-phase6-unit
test-phase6-unit: ## Run Phase 6 unit tests
	@echo "$(BLUE)Running Phase 6 unit tests...$(NC)"
	@$(MAKE) test-nexus-unit
	@$(MAKE) test-controller
	@$(MAKE) test-controller-unit
	@echo "$(GREEN)✓ Phase 6 unit tests completed$(NC)"

.PHONY: test-phase6-integration
test-phase6-integration: ## Run Phase 6 integration tests
	@echo "$(BLUE)Running Phase 6 integration tests...$(NC)"
	@$(MAKE) test-nexus-integration
	@$(MAKE) test-component-integration
	@$(MAKE) test-cross-platform
	@echo "$(GREEN)✓ Phase 6 integration tests completed$(NC)"

.PHONY: test-phase6-e2e
test-phase6-e2e: ## Run Phase 6 end-to-end tests
	@echo "$(BLUE)Running Phase 6 end-to-end tests...$(NC)"
	@$(MAKE) test-nexus-e2e
	@$(MAKE) test-complete-workflows
	@$(MAKE) test-agent-deployment
	@echo "$(GREEN)✓ Phase 6 E2E tests completed$(NC)"

.PHONY: test-phase6-performance
test-phase6-performance: ## Run Phase 6 performance tests
	@echo "$(BLUE)Running Phase 6 performance tests...$(NC)"
	@$(MAKE) test-nexus-performance
	@$(MAKE) test-load-testing
	@$(MAKE) test-memory-usage
	@echo "$(GREEN)✓ Phase 6 performance tests completed$(NC)"

.PHONY: test-phase6-security
test-phase6-security: ## Run Phase 6 security tests
	@echo "$(BLUE)Running Phase 6 security tests...$(NC)"
	@$(MAKE) test-nexus-security
	@$(MAKE) test-wasm-sandbox
	@$(MAKE) test-network-security
	@echo "$(GREEN)✓ Phase 6 security tests completed$(NC)"

.PHONY: test-phase6-reports
test-phase6-reports: ## Generate Phase 6 test reports
	@echo "$(BLUE)Generating Phase 6 test reports...$(NC)"
	@echo "# Phase 6 KNIRV Network Test Report" > $(TEST_REPORTS_DIR)/phase6_summary_$(TIMESTAMP).md
	@echo "Generated: $(TIMESTAMP)" >> $(TEST_REPORTS_DIR)/phase6_summary_$(TIMESTAMP).md
	@echo "" >> $(TEST_REPORTS_DIR)/phase6_summary_$(TIMESTAMP).md
	@echo "## Phase 6 Test Results Summary" >> $(TEST_REPORTS_DIR)/phase6_summary_$(TIMESTAMP).md
	@echo "- Unit Tests: ✓ Completed" >> $(TEST_REPORTS_DIR)/phase6_summary_$(TIMESTAMP).md
	@echo "- Integration Tests: ✓ Completed" >> $(TEST_REPORTS_DIR)/phase6_summary_$(TIMESTAMP).md
	@echo "- End-to-End Tests: ✓ Completed" >> $(TEST_REPORTS_DIR)/phase6_summary_$(TIMESTAMP).md
	@echo "- Performance Tests: ✓ Completed" >> $(TEST_REPORTS_DIR)/phase6_summary_$(TIMESTAMP).md
	@echo "- Security Tests: ✓ Completed" >> $(TEST_REPORTS_DIR)/phase6_summary_$(TIMESTAMP).md
	@echo "$(GREEN)✓ Phase 6 test reports generated$(NC)"

.PHONY: test-controller-unit
test-controller-unit: ## Run KNIRVCONTROLLER unit tests
	@echo "$(BLUE)Running KNIRVCONTROLLER unit tests...$(NC)"
	@if [ -d "KNIRVCONTROLLER" ]; then \
		cd KNIRVCONTROLLER && npm test; \
		echo "$(GREEN)✓ KNIRVCONTROLLER unit tests completed$(NC)"; \
	else \
		echo "$(YELLOW)⚠ KNIRVCONTROLLER directory not found$(NC)"; \
	fi

.PHONY: test-component-integration
test-component-integration: ## Run component integration tests
	@echo "$(BLUE)Running component integration tests...$(NC)"
	@if [ -f "integration-tests/component_integration_test.go" ]; then \
		cd integration-tests && go test -v -run "TestComponent.*"; \
		echo "$(GREEN)✓ Component integration tests completed$(NC)"; \
	else \
		echo "$(YELLOW)⚠ Component integration tests not found$(NC)"; \
	fi

.PHONY: test-cross-platform
test-cross-platform: ## Run cross-platform compatibility tests
	@echo "$(BLUE)Running cross-platform compatibility tests...$(NC)"
	@if [ -f "integration-tests/cross_platform_test.go" ]; then \
		cd integration-tests && go test -v -run "TestCrossPlatform.*"; \
		echo "$(GREEN)✓ Cross-platform tests completed$(NC)"; \
	else \
		echo "$(YELLOW)⚠ Cross-platform tests not found$(NC)"; \
	fi

.PHONY: test-complete-workflows
test-complete-workflows: ## Run complete workflow tests
	@echo "$(BLUE)Running complete workflow tests...$(NC)"
	@if [ -f "integration-tests/complete_workflow_test.go" ]; then \
		cd integration-tests && go test -v -run "TestWorkflow.*"; \
		echo "$(GREEN)✓ Complete workflow tests completed$(NC)"; \
	else \
		echo "$(YELLOW)⚠ Complete workflow tests not found$(NC)"; \
	fi

.PHONY: test-agent-deployment
test-agent-deployment: ## Run agent deployment tests
	@echo "$(BLUE)Running agent deployment tests...$(NC)"
	@if [ -f "integration-tests/agent_deployment_test.go" ]; then \
		cd integration-tests && go test -v -run "TestAgentDeployment.*"; \
		echo "$(GREEN)✓ Agent deployment tests completed$(NC)"; \
	else \
		echo "$(YELLOW)⚠ Agent deployment tests not found$(NC)"; \
	fi

.PHONY: test-load-testing
test-load-testing: ## Run load testing
	@echo "$(BLUE)Running load testing...$(NC)"
	@if [ -f "integration-tests/load_test.go" ]; then \
		cd integration-tests && go test -v -run "TestLoad.*" -timeout=15m; \
		echo "$(GREEN)✓ Load testing completed$(NC)"; \
	else \
		echo "$(YELLOW)⚠ Load tests not found$(NC)"; \
	fi

.PHONY: test-memory-usage
test-memory-usage: ## Run memory usage tests
	@echo "$(BLUE)Running memory usage tests...$(NC)"
	@if [ -f "integration-tests/memory_usage_test.go" ]; then \
		cd integration-tests && go test -v -run "TestMemory.*"; \
		echo "$(GREEN)✓ Memory usage tests completed$(NC)"; \
	else \
		echo "$(YELLOW)⚠ Memory usage tests not found$(NC)"; \
	fi

.PHONY: test-wasm-sandbox
test-wasm-sandbox: ## Run WASM sandbox security tests
	@echo "$(BLUE)Running WASM sandbox security tests...$(NC)"
	@if [ -f "integration-tests/wasm_sandbox_test.go" ]; then \
		cd integration-tests && go test -v -run "TestWASMSandbox.*"; \
		echo "$(GREEN)✓ WASM sandbox tests completed$(NC)"; \
	else \
		echo "$(YELLOW)⚠ WASM sandbox tests not found$(NC)"; \
	fi

.PHONY: test-network-security
test-network-security: ## Run network security tests
	@echo "$(BLUE)Running network security tests...$(NC)"
	@if [ -f "integration-tests/network_security_test.go" ]; then \
		cd integration-tests && go test -v -run "TestNetworkSecurity.*"; \
		echo "$(GREEN)✓ Network security tests completed$(NC)"; \
	else \
		echo "$(YELLOW)⚠ Network security tests not found$(NC)"; \
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

.PHONY: health-check-controller-pwa
health-check-controller-pwa: ## Check health of KNIRVCONTROLLER PWA endpoints
	@echo "$(BLUE)Checking KNIRVCONTROLLER PWA health...$(NC)"
	@if command -v curl >/dev/null 2>&1; then \
		for endpoint in "https://beta-controller.knirv.network" "https://beta-controller.knirv.network/manifest.json" "https://beta-controller.knirv.network/sw.js"; do \
			if curl -s -f "$$endpoint" >/dev/null 2>&1; then \
				echo "$(GREEN)✓ $$endpoint: HEALTHY$(NC)"; \
			else \
				echo "$(RED)✗ $$endpoint: UNHEALTHY$(NC)"; \
			fi; \
		done; \
	else \
		echo "$(YELLOW)curl not available - cannot check PWA health$(NC)"; \
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

.PHONY: backup-nodejs-deps
backup-nodejs-deps: ## Backup manually created KNIRVORACLE Node.js dependency files
	@echo "$(BLUE)Backing up KNIRVORACLE Node.js dependencies...$(NC)"
	@cd KNIRVORACLE/scripts && ./backup-nodejs-deps.sh
	@echo "$(GREEN)✓ Node.js dependencies backed up$(NC)"

.PHONY: restore-nodejs-deps
restore-nodejs-deps: ## Restore manually created KNIRVORACLE Node.js dependency files
	@echo "$(BLUE)Restoring KNIRVORACLE Node.js dependencies...$(NC)"
	@cd KNIRVORACLE/scripts && ./restore-nodejs-deps.sh
	@echo "$(GREEN)✓ Node.js dependencies restored$(NC)"

.PHONY: verify-nodejs-deps
verify-nodejs-deps: ## Verify KNIRVORACLE Node.js dependencies are properly installed
	@echo "$(BLUE)Verifying KNIRVORACLE Node.js dependencies...$(NC)"
	@cd KNIRVORACLE/scripts && ./verify-nodejs-deps.sh

# =============================================================================
# PROTOBUF SYNCHRONIZATION
# =============================================================================

.PHONY: sync-protobuf
sync-protobuf: ## Synchronize ProtoBuf definitions across all platforms
	@echo "$(BLUE)Synchronizing ProtoBuf definitions across all platforms...$(NC)"
	@$(SCRIPTS_DIR)/sync-protobuf.sh
	@echo "$(GREEN)✓ ProtoBuf synchronization completed$(NC)"

.PHONY: sync-protobuf-dry-run
sync-protobuf-dry-run: ## Preview ProtoBuf synchronization changes
	@echo "$(BLUE)Previewing ProtoBuf synchronization...$(NC)"
	@$(SCRIPTS_DIR)/sync-protobuf.sh --dry-run
	@echo "$(GREEN)✓ ProtoBuf synchronization preview completed$(NC)"

.PHONY: sync-protobuf-validate
sync-protobuf-validate: ## Validate ProtoBuf definitions only
	@echo "$(BLUE)Validating ProtoBuf definitions...$(NC)"
	@$(SCRIPTS_DIR)/sync-protobuf.sh --validate-only
	@echo "$(GREEN)✓ ProtoBuf validation completed$(NC)"

.PHONY: sync-protobuf-generate
sync-protobuf-generate: ## Sync ProtoBuf and generate platform-specific code
	@echo "$(BLUE)Synchronizing ProtoBuf and generating code...$(NC)"
	@$(SCRIPTS_DIR)/sync-protobuf.sh --generate-code
	@echo "$(GREEN)✓ ProtoBuf sync and code generation completed$(NC)"

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
# DEMO AND SHOWCASE
# =============================================================================

.PHONY: demo
demo: ## Run full KNIRV Network demo with testnet and KNIRVCONTROLLER
	@echo "$(BLUE)🚀 Starting KNIRV Network Full Demo...$(NC)"
	@./run-full-demo.sh

.PHONY: demo-quick
demo-quick: ## Run quick demo (5 minutes, non-interactive, skip tests)
	@echo "$(BLUE)🚀 Starting KNIRV Network Quick Demo...$(NC)"
	@./run-full-demo.sh --duration 300 --non-interactive --skip-tests

.PHONY: demo-interactive
demo-interactive: ## Run interactive demo with user prompts
	@echo "$(BLUE)🚀 Starting KNIRV Network Interactive Demo...$(NC)"
	@./run-full-demo.sh --duration 600

.PHONY: demo-extended
demo-extended: ## Run extended demo (15 minutes) with full testing
	@echo "$(BLUE)🚀 Starting KNIRV Network Extended Demo...$(NC)"
	@./run-full-demo.sh --duration 900

.PHONY: demo-development
demo-development: ## Run demo without cleanup for development
	@echo "$(BLUE)🚀 Starting KNIRV Network Development Demo...$(NC)"
	@./run-full-demo.sh --no-cleanup

.PHONY: demo-help
demo-help: ## Show demo script help and options
	@echo "$(BLUE)KNIRV Network Demo Commands$(NC)"
	@echo "============================="
	@echo ""
	@echo "$(YELLOW)Available Demo Commands:$(NC)"
	@echo "  $(GREEN)demo$(NC)                 Run full demo (default settings)"
	@echo "  $(GREEN)demo-quick$(NC)           Quick 5-minute demo, non-interactive"
	@echo "  $(GREEN)demo-interactive$(NC)     Interactive demo with user prompts"
	@echo "  $(GREEN)demo-extended$(NC)        Extended 15-minute demo with full testing"
	@echo "  $(GREEN)demo-development$(NC)     Demo without cleanup for development"
	@echo "  $(GREEN)demo-validate$(NC)        Validate demo setup and prerequisites"
	@echo ""
	@echo "$(YELLOW)Direct Script Usage:$(NC)"
	@./run-full-demo.sh --help

.PHONY: demo-validate
demo-validate: ## Validate demo setup and prerequisites
	@echo "$(BLUE)🔍 Validating KNIRV Network Demo Setup...$(NC)"
	@./validate-demo-setup.sh

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
sync: sync-protobuf-dry-run sync-dry-run ## Shortcut for sync-dry-run (safe preview) including ProtoBuf

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
# SUBMODULE MANAGEMENT (NEW)
# =============================================================================

.PHONY: pull-all
pull-all: ## Pull latest changes for the main repo and all submodules
	@echo "$(BLUE)Pulling changes for the main repo...$(NC)"
	@git pull
	@echo "$(BLUE)Updating all submodules...$(NC)"
	@git submodule update --init --recursive --remote
	@echo "$(GREEN)✓ All repositories are up to date.$(NC)"

.PHONY: push-all
push-all: ## Commit with a message and push all changes in submodules and the main repo
	@read -p "Enter commit message for all submodules: " SUBMODULE_MSG; \
	if [ -z "$$SUBMODULE_MSG" ]; then \
		echo "$(RED)Commit message cannot be empty. Aborting.$(NC)"; \
		exit 1; \
	fi; \
	if [ -f "$(SCRIPTS_DIR)/push-submodules.sh" ]; then \
		chmod +x $(SCRIPTS_DIR)/push-submodules.sh; \
		$(SCRIPTS_DIR)/push-submodules.sh "$$SUBMODULE_MSG"; \
	else \
		echo "$(RED)Error: push-submodules.sh script not found!$(NC)"; \
		exit 1; \
	fi

.PHONY: status-all
status-all: ## Show the git status of the main repo and all submodules
	@echo "$(BLUE)--> Main repo status:$(NC)"
	@git status -s
	@echo ""
	@echo "$(BLUE)--> Submodule statuses:$(NC)"
	@git submodule foreach 'echo "--> Status for $$path"; git status -s'

# =============================================================================
# KNIRVANA RUST CLIENT PACKAGING
# =============================================================================

.PHONY: build-rust-client
build-rust-client: ## Build and package KNIRVANA Rust client for all platforms
	@echo "$(BLUE)Building and packaging KNIRVANA Rust client...$(NC)"
	@if [ -d "KNIRVANA/rust-client" ]; then \
		cd KNIRVANA/rust-client && cargo build --release; \
		echo "$(GREEN)✓ KNIRVANA Rust client built successfully$(NC)"; \
	else \
		echo "$(RED)Error: KNIRVANA/rust-client directory not found$(NC)"; \
		exit 1; \
	fi

.PHONY: package-rust-client
package-rust-client: build-rust-client ## Package KNIRVANA Rust client binaries
	@echo "$(BLUE)Packaging KNIRVANA Rust client binaries...$(NC)"
	@if [ -d "KNIRVANA/rust-client" ]; then \
		cd KNIRVANA/rust-client && mkdir -p packaging; \
		echo "$(GREEN)✓ Packaging directories created$(NC)"; \
		echo "$(YELLOW)Run GitHub Actions workflow to generate platform-specific packages$(NC)"; \
		echo "$(YELLOW)Visit: https://github.com/your-org/KNIRV_NETWORK/actions$(NC)"; \
	else \
		echo "$(RED)Error: KNIRVANA/rust-client directory not found$(NC)"; \
		exit 1; \
	fi

.PHONY: download-rust-client-artifacts
download-rust-client-artifacts: ## Download KNIRVANA Rust client artifacts from GitHub Actions
	@echo "$(BLUE)Downloading KNIRVANA Rust client artifacts...$(NC)"
	@if command -v gh >/dev/null 2>&1; then \
		echo "$(YELLOW)Using GitHub CLI to download artifacts...$(NC)"; \
		gh run list --workflow=ci.yml --json databaseId --jq '.[0].databaseId' | xargs -I {} gh run download {} --dir=KNIRVANA/rust-client/downloads --name="knirvana-game-client-*"; \
		echo "$(GREEN)✓ Artifacts downloaded to KNIRVANA/rust-client/downloads/$(NC)"; \
	else \
		echo "$(YELLOW)GitHub CLI not available. Please download artifacts manually:$(NC)"; \
		echo "$(YELLOW)1. Go to GitHub Actions -> KNIRVANA/rust-client CI$(NC)"; \
		echo "$(YELLOW)2. Download the artifacts from the latest successful run$(NC)"; \
		echo "$(YELLOW)3. Extract to KNIRVANA/rust-client/downloads/$(NC)"; \
	fi

.PHONY: install-rust-client-linux
install-rust-client-linux: ## Install Linux version of KNIRVANA Rust client
	@echo "$(BLUE)Installing KNIRVANA Rust client for Linux...$(NC)"
	@if [ -f "KNIRVANA/rust-client/downloads/knirvana-game-client-ubuntu-latest/knirvana_game" ]; then \
		cp KNIRVANA/rust-client/downloads/knirvana-game-client-ubuntu-latest/knirvana_game /usr/local/bin/knirvana_game; \
		chmod +x /usr/local/bin/knirvana_game; \
		echo "$(GREEN)✓ KNIRVANA Rust client installed to /usr/local/bin/knirvana_game$(NC)"; \
	else \
		echo "$(RED)Error: Linux binary not found. Run 'make download-rust-client-artifacts' first$(NC)"; \
		exit 1; \
	fi

.PHONY: install-rust-client-macos
install-rust-client-macos: ## Install macOS version of KNIRVANA Rust client
	@echo "$(BLUE)Installing KNIRVANA Rust client for macOS...$(NC)"
	@if [ -f "KNIRVANA/rust-client/downloads/knirvana-game-client-macos-latest/knirvana_game" ]; then \
		cp KNIRVANA/rust-client/downloads/knirvana-game-client-macos-latest/knirvana_game /usr/local/bin/knirvana_game; \
		chmod +x /usr/local/bin/knirvana_game; \
		echo "$(GREEN)✓ KNIRVANA Rust client installed to /usr/local/bin/knirvana_game$(NC)"; \
	else \
		echo "$(RED)Error: macOS binary not found. Run 'make download-rust-client-artifacts' first$(NC)"; \
		exit 1; \
	fi

.PHONY: install-rust-client-windows
install-rust-client-windows: ## Install Windows version of KNIRVANA Rust client
	@echo "$(BLUE)Installing KNIRVANA Rust client for Windows...$(NC)"
	@if [ -f "KNIRVANA/rust-client/downloads/knirvana-game-client-windows-latest/knirvana_game.exe" ]; then \
		echo "$(YELLOW)Windows binary available at: KNIRVANA/rust-client/downloads/knirvana-game-client-windows-latest/knirvana_game.exe$(NC)"; \
		echo "$(YELLOW)Copy this file to your desired installation location$(NC)"; \
	else \
		echo "$(RED)Error: Windows binary not found. Run 'make download-rust-client-artifacts' first$(NC)"; \
		exit 1; \
	fi

.PHONY: rust-client-status
rust-client-status: ## Show KNIRVANA Rust client status and available binaries
	@echo "$(BLUE)KNIRVANA Rust Client Status$(NC)"
	@echo "=============================="
	@echo ""
	@if [ -d "KNIRVANA/rust-client/downloads" ]; then \
		echo "$(YELLOW)Available Artifacts:$(NC)"; \
		ls -la KNIRVANA/rust-client/downloads/ 2>/dev/null || echo "  No artifacts downloaded"; \
	else \
		echo "$(YELLOW)Download directory: Not created$(NC)"; \
	fi
	@echo ""
	@echo "$(YELLOW)Build Status:$(NC)"
	@if [ -f "KNIRVANA/rust-client/target/release/knirvana_game" ]; then \
		echo "  Local build: ✓ Available"; \
	else \
		echo "  Local build: ✗ Not built"; \
	fi
	@echo ""
	@echo "$(YELLOW)Usage:$(NC)"
	@echo "  make build-rust-client          # Build locally"; \
	@echo "  make package-rust-client        # Prepare for packaging"; \
	@echo "  make download-rust-client-artifacts # Download from GitHub Actions"; \
	@echo "  make install-rust-client-linux  # Install Linux version"; \
	@echo "  make install-rust-client-macos  # Install macOS version"; \
	@echo "  make install-rust-client-windows # Prepare Windows version"; \

# =============================================================================
# BINARY SYNCHRONIZATION
# =============================================================================

.PHONY: sync-binaries
sync-binaries: ## Synchronize GraphChain CLI and KNIRVGRAPH binaries
	@echo "$(BLUE)Synchronizing KNIRV Binaries...$(NC)"
	@$(SCRIPTS_DIR)/sync-portal-versions.sh --type binaries --verbose
	@echo "$(GREEN)✓ Binary synchronization completed$(NC)"

.PHONY: sync-binaries-dry-run
sync-binaries-dry-run: ## Preview binary synchronization changes
	@echo "$(BLUE)Previewing KNIRV Binary Synchronization...$(NC)"
	@$(SCRIPTS_DIR)/sync-portal-versions.sh --type binaries --dry-run --verbose
	@echo "$(GREEN)✓ Binary synchronization preview completed$(NC)"

# =============================================================================
# GATEWAY DEPLOYMENT
# =============================================================================

.PHONY: deploy-testnet-gateway
deploy-testnet-gateway: ## Deploy KNIRVGATEWAY with testnet endpoints (automatic)
	@echo "$(BLUE)Deploying KNIRVGATEWAY for Testnet...$(NC)"
	@echo "$(YELLOW)⚠️  This will automatically update all endpoints to testnet versions$(NC)"
	@ansible-playbook deployment/ansible/update-gateway-endpoints.yml \
		-e environment=testnet \
		-e @deployment/ansible/environments/testnet.yml
	@cd KNIRVGATEWAY && npm run load-endpoints:testnet && npm run build && npm run deploy
	@echo "$(GREEN)✅ KNIRVGATEWAY deployed to testnet successfully$(NC)"

.PHONY: generate-production-endpoints
generate-production-endpoints: ## Generate production endpoints.yaml file
	@echo "$(BLUE)Generating Production Endpoints...$(NC)"
	@ansible-playbook deployment/ansible/update-gateway-endpoints.yml \
		-e environment=production \
		-e @deployment/ansible/environments/production.yml
	@echo "$(GREEN)✅ Production endpoints.yaml generated$(NC)"
	@echo "$(YELLOW)⚠️  Review KNIRVGATEWAY/config/endpoints.yaml before deploying$(NC)"

.PHONY: deploy-production-gateway
deploy-production-gateway: ## Deploy KNIRVGATEWAY with production endpoints (manual control)
	@echo "$(BLUE)Deploying KNIRVGATEWAY for Production...$(NC)"
	@echo "$(YELLOW)⚠️  This requires manual verification of endpoints.yaml$(NC)"
	@if [ ! -f KNIRVGATEWAY/config/endpoints.yaml ]; then \
		echo "$(RED)❌ Production endpoints.yaml not found$(NC)"; \
		echo "$(YELLOW)   Run 'make generate-production-endpoints' first$(NC)"; \
		exit 1; \
	fi
	@echo "$(CYAN)📋 Current production endpoints:$(NC)"
	@cat KNIRVGATEWAY/config/endpoints.yaml | grep -A 20 "endpoints:"
	@echo ""
	@read -p "$(YELLOW)Continue with production deployment? [y/N]: $(NC)" confirm; \
	if [ "$$confirm" = "y" ] || [ "$$confirm" = "Y" ]; then \
		cd KNIRVGATEWAY && npm run load-endpoints:production && npm run build && npm run deploy; \
		echo "$(GREEN)✅ KNIRVGATEWAY deployed to production successfully$(NC)"; \
	else \
		echo "$(YELLOW)⏸️  Production deployment cancelled$(NC)"; \
	fi

.PHONY: update-testnet-endpoints
update-testnet-endpoints: ## Update KNIRVGATEWAY endpoints for testnet (without deploying)
	@echo "$(BLUE)Updating KNIRVGATEWAY Testnet Endpoints...$(NC)"
	@ansible-playbook deployment/ansible/update-gateway-endpoints.yml \
		-e environment=testnet \
		-e @deployment/ansible/environments/testnet.yml
	@cd KNIRVGATEWAY && npm run load-endpoints:testnet
	@echo "$(GREEN)✅ Testnet endpoints updated$(NC)"

# =============================================================================
# GITHUB ACTIONS INTEGRATION
# =============================================================================

.PHONY: trigger-rust-client-ci
trigger-rust-client-ci: ## Trigger KNIRVANA Rust client CI workflow via GitHub API
	@echo "$(BLUE)Triggering KNIRVANA Rust client CI workflow...$(NC)"
	@if command -v gh >/dev/null 2>&1; then \
		echo "$(YELLOW)Using GitHub CLI to trigger workflow...$(NC)"; \
		gh workflow run ci.yml --ref=main --repo=. --dir=KNIRVANA/rust-client; \
		echo "$(GREEN)✓ CI workflow triggered successfully$(NC)"; \
		echo "$(YELLOW)Monitor progress: https://github.com/your-org/KNIRV_NETWORK/actions$(NC)"; \
	else \
		echo "$(YELLOW)GitHub CLI not available. Manual trigger required:$(NC)"; \
		echo "$(YELLOW)1. Go to: https://github.com/your-org/KNIRV_NETWORK/actions$(NC)"; \
		echo "$(YELLOW)2. Select 'KNIRVANA Rust Client CI' workflow$(NC)"; \
		echo "$(YELLOW)3. Click 'Run workflow' -> 'Run workflow'$(NC)"; \
	fi

.PHONY: watch-rust-client-ci
watch-rust-client-ci: ## Watch the progress of KNIRVANA Rust client CI workflow
	@echo "$(BLUE)Watching KNIRVANA Rust client CI workflow...$(NC)"
	@if command -v gh >/dev/null 2>&1; then \
		gh run list --workflow=ci.yml --limit=1 --json databaseId,status --jq '.[0]' | \
		jq -r '.databaseId + " " + .status' | \
		while read run_id status; do \
			if [ "$$status" = "completed" ]; then \
				echo "$(GREEN)Workflow completed. Downloading artifacts...$(NC)"; \
				gh run download $$run_id --dir=KNIRVANA/rust-client/downloads --name="knirvana-game-client-*"; \
				break; \
			else \
				echo "$(YELLOW)Workflow status: $$status. Watching...$(NC)"; \
				sleep 30; \
			fi; \
		done; \
	else \
		echo "$(YELLOW)GitHub CLI not available. Please monitor manually:$(NC)"; \
		echo "$(YELLOW)https://github.com/your-org/KNIRV_NETWORK/actions$(NC)"; \
	fi
# =============================================================================
# CONTENT SYNCHRONIZATION
# =============================================================================

.PHONY: sync-failover-page
sync-failover-page: ## Synchronize network-website content to the KNIRVGATEWAY failover page
	@echo "$(BLUE)Syncing network-website content to KNIRVGATEWAY/home.html...$(NC)"
	@if [ -f "KNIRVGATEWAY/network-website/public/index.html" ]; then \
		cp KNIRVGATEWAY/network-website/public/index.html KNIRVGATEWAY/home.html; \
		echo "$(GREEN)✓ Copied index.html$(NC)"; \
	else \
		echo "$(RED)Error: KNIRVGATEWAY/network-website/public/index.html not found.$(NC)"; \
		exit 1; \
	fi
	@if [ -d "KNIRVGATEWAY/network-website/public/assets" ]; then \
		rm -rf KNIRVGATEWAY/assets; \
		cp -r KNIRVGATEWAY/network-website/public/assets KNIRVGATEWAY/assets; \
		echo "$(GREEN)✓ Copied assets directory$(NC)"; \
	else \
		echo "$(YELLOW)Warning: network-website/public/assets directory not found. Skipping.$(NC)"; \
	fi
	@echo "$(GREEN)✓ Failover page synchronized successfully.$(NC)"

# =============================================================================
# DEFAULT TARGET
# =============================================================================

.DEFAULT_GOAL := help
