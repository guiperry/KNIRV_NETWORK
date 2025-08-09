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
KNIRVWEBSITE_DIR := $(PROJECT_ROOT)/KNIRVWEBSITE
DOCS_DIR := $(PROJECT_ROOT)/KNIRVWEBSITE/documentation
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
deploy-website: docs ## Deploy KNIRVWEBSITE (with fresh documentation)
	@echo "$(BLUE)Deploying KNIRVWEBSITE with updated documentation...$(NC)"
	@if [ -d "$(KNIRVWEBSITE_DIR)" ]; then \
		echo "$(YELLOW)Note: KNIRVWEBSITE deployment method depends on your hosting setup$(NC)"; \
		echo "$(YELLOW)Documentation has been generated and is ready for deployment$(NC)"; \
		echo "$(YELLOW)Generated files are in: $(DOCS_DIR)/docsify$(NC)"; \
	else \
		echo "$(RED)Error: KNIRVWEBSITE directory not found$(NC)"; \
		exit 1; \
	fi
	@echo "$(GREEN)✓ KNIRVWEBSITE ready for deployment$(NC)"

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

# =============================================================================
# DEFAULT TARGET
# =============================================================================

.DEFAULT_GOAL := help
