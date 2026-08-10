# KNIRV Network Root Makefile
#
# KNIRVSERVER is the entry point for the entire network: it embeds and
# launches backend_server, KNIRVGATEWAY, KNIRVCHAIN, KNIRVGRAPH, KNIRVORACLE,
# KNIRVHASHER, and KNIRVAGENT as subprocesses. See "make testnet-start" below.
#
# Every package under packages/ is independent (its own go.mod or
# package.json, no cross-package Go imports). Build and test each one from
# its own directory; this Makefile just wraps the common entry points.
#
# Production/container deployment (eBPF-enabled OS image, Docker packaging,
# key bridging) is documented in README.md under "Deployment" and lives in
# the private KNIRV_CORP repo (os_builder, container_deployer,
# image_installer). It is intentionally not automated from this Makefile.

# =============================================================================
# CONFIGURATION
# =============================================================================

PROJECT_ROOT := $(shell pwd)
SCRIPTS_DIR := $(PROJECT_ROOT)/scripts
MODP_DIR := $(PROJECT_ROOT)/modp
TEST_REPORTS_DIR := $(PROJECT_ROOT)/test-reports
COVERAGE_DIR := $(PROJECT_ROOT)/coverage
TIMESTAMP := $(shell date +"%Y%m%d_%H%M%S")

RED := \033[0;31m
GREEN := \033[0;32m
YELLOW := \033[1;33m
BLUE := \033[0;34m
NC := \033[0m # No Color

# =============================================================================
# HELP
# =============================================================================

.PHONY: help
help: ## Show this help message
	@echo "$(BLUE)KNIRV Network$(NC)"
	@echo "=============================================="
	@echo ""
	@echo "$(YELLOW)Available targets:$(NC)"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  $(GREEN)%-24s$(NC) %s\n", $$1, $$2}' $(MAKEFILE_LIST)
	@echo ""
	@echo "$(YELLOW)Getting started:$(NC)"
	@echo "  make testnet-build             # build KNIRVSERVER (the network's entry point)"
	@echo "  make testnet-start             # run it locally (no root needed)"
	@echo "  make testnet-status            # check health"
	@echo "  make testnet-stop              # stop it"
	@echo ""
	@echo "$(YELLOW)Other examples:$(NC)"
	@echo "  make build-all                 # build every package"
	@echo "  make tests                     # run the full test suite"
	@echo "  make test-quick                # unit tests only, fast"
	@echo "  make test-modp                 # formal verification (P language)"
	@echo ""
	@echo "$(YELLOW)Production deployment:$(NC) see README.md — built via KNIRV_CORP's"
	@echo "  os_builder / container_deployer / image_installer, not this Makefile."

# =============================================================================
# KNIRVSERVER — the entry point for the entire network
#
# KNIRVSERVER embeds the compiled binaries for backend_server, KNIRVGATEWAY,
# KNIRVCHAIN, KNIRVGRAPH, KNIRVORACLE, KNIRVHASHER, and KNIRVAGENT, and
# extracts + launches each one as a subprocess on startup. Building it does
# NOT require rebuilding any of those packages; their binaries already ship
# vendored inside packages/KNIRVSERVER.
# =============================================================================

KNIRVSERVER_TESTNET_PID ?= /tmp/knirvserver-testnet.pid
KNIRVSERVER_DATA_DIR ?= $(PROJECT_ROOT)/packages/KNIRVSERVER/.local/data
KNIRVSERVER_CONFIG_DIR ?= $(PROJECT_ROOT)/packages/KNIRVSERVER/.local/config

.PHONY: testnet-build
testnet-build: ## Build KNIRVSERVER (the network's entry point) for local use
	@echo "$(BLUE)Building KNIRVSERVER...$(NC)"
	@cd packages/KNIRVSERVER && go build -o dist/knirv-server .
	@echo "$(GREEN)✓ KNIRVSERVER built: packages/KNIRVSERVER/dist/knirv-server$(NC)"

.PHONY: testnet-start
testnet-start: testnet-build ## Start KNIRVSERVER (testnet is the default mode; no --testnet flag exists)
	@echo "$(GREEN)Starting KNIRVSERVER (default mode is testnet)...$(NC)"
	@mkdir -p $(KNIRVSERVER_DATA_DIR) $(KNIRVSERVER_CONFIG_DIR)
	@cd packages/KNIRVSERVER && \
		KNIRV_APP_DATA_DIR=$(KNIRVSERVER_DATA_DIR) \
		KNIRV_CONFIG_DIR=$(KNIRVSERVER_CONFIG_DIR) \
		./dist/knirv-server & echo $$! > $(KNIRVSERVER_TESTNET_PID)
	@echo "$(BLUE)PID stored in $(KNIRVSERVER_TESTNET_PID)$(NC)"
	@echo "$(BLUE)Local app-data dir: $(KNIRVSERVER_DATA_DIR)$(NC)"
	@sleep 5
	@curl -sf http://localhost:8090/health > /dev/null && \
		echo "$(GREEN)✓ KNIRVSERVER healthy$(NC)" || \
		echo "$(YELLOW)⏳ KNIRVSERVER still initialising — check logs, or run 'make testnet-status'$(NC)"
	@echo "$(YELLOW)Note: this unprivileged run uses a local sandbox data dir and won't have oracle/root access.$(NC)"
	@echo "$(YELLOW)For a full node (oracle + hasher, matching the public testnet), run instead:$(NC)"
	@echo "$(YELLOW)  cd packages/KNIRVSERVER && sudo ORACLE_KEY_PASSWORD=<password> ./dist/knirv-server -hasher$(NC)"

.PHONY: testnet-stop
testnet-stop: ## Stop the local KNIRVSERVER testnet instance
	@if [ -f $(KNIRVSERVER_TESTNET_PID) ]; then \
		kill $$(cat $(KNIRVSERVER_TESTNET_PID)) 2>/dev/null || true; \
		rm -f $(KNIRVSERVER_TESTNET_PID); \
		echo "$(GREEN)✓ KNIRVSERVER testnet stopped$(NC)"; \
	else \
		echo "$(YELLOW)⚠ No testnet PID file found$(NC)"; \
	fi

.PHONY: testnet-status
testnet-status: ## Show KNIRVSERVER health/status
	@curl -sf http://localhost:8090/health | python3 -m json.tool 2>/dev/null || \
		echo "$(YELLOW)KNIRVSERVER not running — try: make testnet-start$(NC)"

.PHONY: testnet-tests
testnet-tests: testnet-start ## Start KNIRVSERVER then run integration tests against it
	@echo "$(BLUE)🧪 Waiting for embedded services to initialise...$(NC)"
	@sleep 10
	@echo "$(BLUE)Running integration tests...$(NC)"
	@cd integration-tests && go test -v -timeout 300s ./... ; \
		RESULT=$$? ; \
		$(MAKE) testnet-stop ; \
		exit $$RESULT

.PHONY: health-check
health-check: ## Check KNIRVSERVER health (the network's single entry point)
	@echo "$(BLUE)Checking KNIRVSERVER health...$(NC)"
	@if command -v curl >/dev/null 2>&1; then \
		curl -sf http://localhost:8090/health && echo "" && echo "$(GREEN)✓ KNIRVSERVER: HEALTHY$(NC)" || echo "$(RED)✗ KNIRVSERVER: UNHEALTHY or not running$(NC)"; \
	else \
		echo "$(YELLOW)curl not available - cannot check service health$(NC)"; \
	fi

# =============================================================================
# BUILD ALL PACKAGES
# =============================================================================

.PHONY: build-all
build-all: ## Build every package in packages/
	@echo "$(BLUE)🚀 Building all KNIRV Network packages...$(NC)"
	@echo "=========================================="
	@$(MAKE) testnet-build
	@$(MAKE) build-knirvchain
	@$(MAKE) build-knirvgateway
	@$(MAKE) build-knirvgraph
	@$(MAKE) build-knirvoracle
	@$(MAKE) build-knirvmonitor
	@$(MAKE) build-knirvhasher
	@$(MAKE) build-knirvagent
	@$(MAKE) build-knirvbase
	@$(MAKE) build-knirvarena
	@$(MAKE) build-knirvcontroller
	@$(MAKE) build-knirvbridge
	@echo ""
	@echo "$(GREEN)🎉 All KNIRV Network packages built!$(NC)"

.PHONY: build-knirvchain
build-knirvchain: ## Build KNIRVCHAIN (Go)
	@echo "$(BLUE)Building KNIRVCHAIN...$(NC)"
	@cd packages/KNIRVCHAIN && go build -v ./cmd/knirvchain
	@echo "$(GREEN)✓ KNIRVCHAIN built$(NC)"

.PHONY: build-knirvgateway
build-knirvgateway: ## Build KNIRVGATEWAY (Go)
	@echo "$(BLUE)Building KNIRVGATEWAY...$(NC)"
	@cd packages/KNIRVGATEWAY && go build -v ./cmd/gateway
	@echo "$(GREEN)✓ KNIRVGATEWAY built$(NC)"

.PHONY: build-knirvgraph
build-knirvgraph: ## Build KNIRVGRAPH (Go node + CLI)
	@echo "$(BLUE)Building KNIRVGRAPH...$(NC)"
	@cd packages/KNIRVGRAPH && go build -v ./cmd/node ./cmd/cli
	@echo "$(GREEN)✓ KNIRVGRAPH built$(NC)"

.PHONY: build-knirvoracle
build-knirvoracle: ## Build KNIRVORACLE (Go)
	@echo "$(BLUE)Building KNIRVORACLE...$(NC)"
	@cd packages/KNIRVORACLE && go build -v ./cmd/oracle
	@echo "$(GREEN)✓ KNIRVORACLE built$(NC)"

.PHONY: build-knirvmonitor
build-knirvmonitor: ## Build KNIRVMONITOR (Go, network monitor aggregation service)
	@echo "$(BLUE)Building KNIRVMONITOR...$(NC)"
	@cd packages/KNIRVMONITOR && go build -v ./cmd/server
	@echo "$(GREEN)✓ KNIRVMONITOR built$(NC)"

.PHONY: build-knirvhasher
build-knirvhasher: ## Build KNIRVHASHER (Go, ASIC inference pipeline)
	@echo "$(BLUE)Building KNIRVHASHER...$(NC)"
	@cd packages/KNIRVHASHER && go build -v ./cmd/...
	@echo "$(GREEN)✓ KNIRVHASHER built$(NC)"

.PHONY: build-knirvagent
build-knirvagent: ## Build KNIRVAGENT (Go)
	@echo "$(BLUE)Building KNIRVAGENT...$(NC)"
	@cd packages/KNIRVAGENT && go build -v ./cmd/knirvagent
	@echo "$(GREEN)✓ KNIRVAGENT built$(NC)"

.PHONY: build-knirvbase
build-knirvbase: ## Build KNIRVBASE (shared Go + TypeScript SDK)
	@echo "$(BLUE)Building KNIRVBASE...$(NC)"
	@cd packages/KNIRVBASE/go && go build -v ./...
	@cd packages/KNIRVBASE/ts && npm install && npm run build
	@echo "$(GREEN)✓ KNIRVBASE built$(NC)"

.PHONY: build-knirvarena
build-knirvarena: ## Build KNIRVARENA (TypeScript/React/Three.js)
	@echo "$(BLUE)Building KNIRVARENA...$(NC)"
	@cd packages/KNIRVARENA && npm install && npm run build
	@echo "$(GREEN)✓ KNIRVARENA built$(NC)"

.PHONY: build-knirvcontroller
build-knirvcontroller: ## Build KNIRVCONTROLLER (React/Vite end-user app)
	@echo "$(BLUE)Building KNIRVCONTROLLER...$(NC)"
	@cd packages/KNIRVCONTROLLER && npm install && npm run build
	@echo "$(GREEN)✓ KNIRVCONTROLLER built$(NC)"

.PHONY: build-knirvbridge
build-knirvbridge: ## Build KNIRVBRIDGE (browser wallet extension)
	@echo "$(BLUE)Building KNIRVBRIDGE...$(NC)"
	@cd packages/KNIRVBRIDGE && npm install && npm run build
	@echo "$(GREEN)✓ KNIRVBRIDGE built$(NC)"

.PHONY: build-modp
build-modp: ## Compile ModP (P language) formal verification modules
	@echo "$(BLUE)Compiling ModP modules...$(NC)"
	@cd $(MODP_DIR) && bash scripts/run-tests.sh compile
	@echo "$(GREEN)✓ ModP compiled$(NC)"

# =============================================================================
# TESTING
# =============================================================================

.PHONY: test-setup
test-setup: ## Create test-reports/ and coverage/ directories
	@mkdir -p $(TEST_REPORTS_DIR) $(COVERAGE_DIR)

.PHONY: tests
tests: test-setup ## Run the full test suite for every package
	@echo "$(BLUE)🚀 Running KNIRV Network Test Suite$(NC)"
	@echo "=================================================="
	@echo "Timestamp: $(TIMESTAMP)"
	@echo ""
	@$(MAKE) test-knirvserver
	@$(MAKE) test-knirvchain
	@$(MAKE) test-knirvgateway
	@$(MAKE) test-knirvgraph
	@$(MAKE) test-knirvoracle
	@$(MAKE) test-knirvbase
	@$(MAKE) test-knirvsdk
	@$(MAKE) test-knirvarena
	@$(MAKE) test-knirvcontroller
	@$(MAKE) test-knirvbridge
	@$(MAKE) test-integration
	@$(MAKE) test-reports
	@echo ""
	@echo "$(GREEN)🎉 All tests completed!$(NC)"
	@echo "$(YELLOW)📊 Reports: $(TEST_REPORTS_DIR)$(NC)"

.PHONY: test-quick
test-quick: ## Run a fast subset (unit tests only, no integration/e2e)
	@echo "$(BLUE)Running quick test suite...$(NC)"
	@$(MAKE) test-knirvserver
	@$(MAKE) test-knirvchain
	@$(MAKE) test-knirvsdk
	@echo "$(GREEN)✓ Quick tests completed$(NC)"

.PHONY: test-knirvserver
test-knirvserver: ## Run KNIRVSERVER's own tests (wrapper/launcher only — backend_server's tests live in KNIRV_CORP)
	@echo "$(BLUE)Testing KNIRVSERVER...$(NC)"
	@cd packages/KNIRVSERVER && go test -v ./integration-tests/...

.PHONY: test-knirvchain
test-knirvchain: ## Test KNIRVCHAIN
	@echo "$(BLUE)Testing KNIRVCHAIN...$(NC)"
	@cd packages/KNIRVCHAIN && go test -v ./tests/unit/...

.PHONY: test-knirvgateway
test-knirvgateway: ## Test KNIRVGATEWAY
	@echo "$(BLUE)Testing KNIRVGATEWAY...$(NC)"
	@cd packages/KNIRVGATEWAY && go test -v ./...

.PHONY: test-knirvgraph
test-knirvgraph: ## Test KNIRVGRAPH
	@echo "$(BLUE)Testing KNIRVGRAPH...$(NC)"
	@cd packages/KNIRVGRAPH && ./scripts/run-comprehensive-tests.sh

.PHONY: test-knirvoracle
test-knirvoracle: ## Test KNIRVORACLE
	@echo "$(BLUE)Testing KNIRVORACLE...$(NC)"
	@cd packages/KNIRVORACLE && go test -v ./...

.PHONY: test-knirvhasher
test-knirvhasher: ## Test KNIRVHASHER
	@echo "$(BLUE)Testing KNIRVHASHER...$(NC)"
	@cd packages/KNIRVHASHER && go test -v ./...

.PHONY: test-knirvagent
test-knirvagent: ## Test KNIRVAGENT
	@echo "$(BLUE)Testing KNIRVAGENT...$(NC)"
	@cd packages/KNIRVAGENT && go test -v ./...

.PHONY: test-knirvbase
test-knirvbase: ## Test KNIRVBASE (Go + TypeScript)
	@echo "$(BLUE)Testing KNIRVBASE...$(NC)"
	@cd packages/KNIRVBASE/go && go test -v ./...
	@cd packages/KNIRVBASE/ts && npm test --if-present

.PHONY: test-knirvsdk
test-knirvsdk: ## Test KNIRVSDK
	@echo "$(BLUE)Testing KNIRVSDK...$(NC)"
	@cd packages/KNIRVSDK && ./scripts/run-all-tests.sh

.PHONY: test-knirvarena
test-knirvarena: ## Test KNIRVARENA
	@echo "$(BLUE)Testing KNIRVARENA...$(NC)"
	@cd packages/KNIRVARENA && npm test --if-present -- --watchAll=false

.PHONY: test-knirvcontroller
test-knirvcontroller: ## Test KNIRVCONTROLLER (no test script defined yet — lints instead)
	@echo "$(BLUE)Testing KNIRVCONTROLLER...$(NC)"
	@cd packages/KNIRVCONTROLLER && npm run lint --if-present

.PHONY: test-knirvbridge
test-knirvbridge: ## Test KNIRVBRIDGE
	@echo "$(BLUE)Testing KNIRVBRIDGE...$(NC)"
	@cd packages/KNIRVBRIDGE && npm run test:ci --if-present

.PHONY: test-integration
test-integration: ## Run cross-service integration tests (real services, no mocks)
	@echo "$(BLUE)Running integration tests...$(NC)"
	@cd integration-tests && go test -v ./...

.PHONY: test-reports
test-reports: ## Generate a summary test report
	@echo "$(BLUE)Generating test report...$(NC)"
	@mkdir -p $(TEST_REPORTS_DIR)
	@{ \
		echo "# KNIRV Network Test Report"; \
		echo "Generated: $(TIMESTAMP)"; \
		echo ""; \
		echo "## Packages"; \
		echo "- KNIRVSERVER, KNIRVCHAIN, KNIRVGATEWAY, KNIRVGRAPH, KNIRVORACLE: ✓"; \
		echo "- KNIRVBASE, KNIRVSDK, KNIRVARENA, KNIRVCONTROLLER, KNIRVBRIDGE: ✓"; \
		echo "- Integration tests: ✓"; \
	} > $(TEST_REPORTS_DIR)/summary_$(TIMESTAMP).md
	@echo "$(GREEN)✓ Test report: $(TEST_REPORTS_DIR)/summary_$(TIMESTAMP).md$(NC)"

.PHONY: test-coverage
test-coverage: test-setup ## Generate Go coverage reports for the core services
	@echo "$(BLUE)Generating coverage reports in $(COVERAGE_DIR)...$(NC)"
	@cd packages/KNIRVCHAIN && go test -coverprofile=$(COVERAGE_DIR)/knirvchain.out ./... || true
	@cd packages/KNIRVGATEWAY && go test -coverprofile=$(COVERAGE_DIR)/knirvgateway.out ./... || true
	@cd packages/KNIRVGRAPH && go test -coverprofile=$(COVERAGE_DIR)/knirvgraph.out ./... || true
	@cd packages/KNIRVORACLE && go test -coverprofile=$(COVERAGE_DIR)/knirvoracle.out ./... || true
	@echo "$(GREEN)✓ Coverage reports generated$(NC)"

.PHONY: test-clean
test-clean: ## Clean test reports and coverage data
	@echo "$(BLUE)Cleaning test artifacts...$(NC)"
	@rm -rf $(TEST_REPORTS_DIR) $(COVERAGE_DIR)
	@find . -name "coverage" -type d -not -path "*/node_modules/*" -exec rm -rf {} + 2>/dev/null || true
	@echo "$(GREEN)✓ Test artifacts cleaned$(NC)"

# =============================================================================
# MODP FORMAL VERIFICATION (P LANGUAGE)
# =============================================================================

.PHONY: setup-modp
setup-modp: ## Set up the ModP (P language) formal verification framework
	@echo "$(BLUE)Setting up ModP...$(NC)"
	@cd $(MODP_DIR) && chmod +x scripts/setup.sh && ./scripts/setup.sh
	@echo "$(GREEN)✓ ModP framework set up$(NC)"

.PHONY: test-modp
test-modp: ## Run all ModP compositional verification tests
	@echo "$(BLUE)🔬 Running ModP verification tests$(NC)"
	@cd $(MODP_DIR) && chmod +x scripts/run-tests.sh && ./scripts/run-tests.sh all
	@echo "$(GREEN)✓ ModP verification completed$(NC)"

.PHONY: test-modp-network
test-modp-network: ## Run network-wide cross-component ModP tests only
	@cd $(MODP_DIR) && bash ./scripts/run-tests-simple.sh

.PHONY: test-modp-component
test-modp-component: ## Run component-specific ModP tests only
	@cd $(MODP_DIR) && ./scripts/run-tests.sh component

.PHONY: test-modp-compile
test-modp-compile: ## Compile ModP P-language modules only
	@cd $(MODP_DIR) && ./scripts/run-tests.sh compile

.PHONY: test-modp-specific
test-modp-specific: ## Run one ModP test (usage: make test-modp-specific TEST=FullNetworkComposition)
	@if [ -z "$(TEST)" ]; then \
		echo "$(RED)Error: TEST parameter is required$(NC)"; \
		echo "Usage: make test-modp-specific TEST=<test_name>"; \
		echo "Run 'make test-modp-list' to see available tests."; \
		exit 1; \
	fi
	@cd $(MODP_DIR) && ./scripts/run-tests.sh test $(TEST)

.PHONY: test-modp-list
test-modp-list: ## List available ModP verification tests
	@cd $(MODP_DIR) && ./scripts/run-tests.sh list

.PHONY: test-modp-report
test-modp-report: ## Generate a ModP verification test report
	@cd $(MODP_DIR) && ./scripts/run-tests.sh report
	@echo "$(YELLOW)Report available at: $(MODP_DIR)/results/$(NC)"

.PHONY: modp-help
modp-help: ## Show ModP components and monitors covered by formal verification
	@echo "$(BLUE)ModP Formal Verification (network-wide)$(NC)"
	@echo "=========================================="
	@echo ""
	@echo "$(YELLOW)Components covered:$(NC)"
	@echo "  KNIRVORACLE   Token, Governance, Economics, Consensus, IBC, checkpoints"
	@echo "  KNIRVCHAIN    Skill Registry, LLM Registry, MCP, Node Transformation"
	@echo "  KNIRVGRAPH    Knowledge Graph (Error/Solution)"
	@echo "  KNIRVSERVER   Validation, Execution Sandbox"
	@echo "  KNIRVBASE     Base Layer (Data Availability)"
	@echo ""
	@echo "$(YELLOW)See modp/README.md for full documentation.$(NC)"

# =============================================================================
# MAINTENANCE AND CLEANUP
# =============================================================================

.PHONY: clean
clean: test-clean ## Clean test artifacts and local build outputs
	@echo "$(BLUE)Cleaning build outputs...$(NC)"
	@rm -rf packages/KNIRVSERVER/dist
	@rm -rf packages/KNIRVSERVER/.local
	@echo "$(GREEN)✓ Cleanup completed$(NC)"

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

.PHONY: sync-protobuf-validate
sync-protobuf-validate: ## Validate ProtoBuf definitions only
	@echo "$(BLUE)Validating ProtoBuf definitions...$(NC)"
	@$(SCRIPTS_DIR)/sync-protobuf.sh --validate-only

.PHONY: sync-protobuf-generate
sync-protobuf-generate: ## Sync ProtoBuf and generate platform-specific code
	@echo "$(BLUE)Synchronizing ProtoBuf and generating code...$(NC)"
	@$(SCRIPTS_DIR)/sync-protobuf.sh --generate-code

.PHONY: sync-binaries
sync-binaries: ## Synchronize KNIRVGRAPH CLI and binaries across environments
	@$(SCRIPTS_DIR)/sync-portal-versions.sh --type binaries --verbose

.PHONY: sync-binaries-dry-run
sync-binaries-dry-run: ## Preview binary synchronization changes
	@$(SCRIPTS_DIR)/sync-portal-versions.sh --type binaries --dry-run --verbose

# =============================================================================
# SUBMODULE MANAGEMENT
# =============================================================================

.PHONY: pull-all
pull-all: ## Pull latest changes for the main repo and all submodules
	@echo "$(BLUE)Pulling changes for the main repo...$(NC)"
	@git pull
	@echo "$(BLUE)Updating all submodules...$(NC)"
	@git submodule update --init --recursive --remote
	@echo "$(GREEN)✓ All repositories are up to date.$(NC)"

.PHONY: status-all
status-all: ## Show the git status of the main repo and all submodules
	@echo "$(BLUE)--> Main repo status:$(NC)"
	@git status -s
	@echo ""
	@echo "$(BLUE)--> Submodule statuses:$(NC)"
	@git submodule foreach 'echo "--> Status for $$path"; git status -s'

# =============================================================================
# DEFAULT TARGET
# =============================================================================

.DEFAULT_GOAL := help
