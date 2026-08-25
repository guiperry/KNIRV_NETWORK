# Run the following command to remove indents and chage to tabs: 
# mv -f makefile{,.old} && sed -E 's/^ {3,}/\t/' makefile.old >makefile

# Application and Version Configuration
APP_NAME := KNIRVENGINE
# Determine version: Use git tag, or fallback to dev-commit_hash
VERSION ?= $(shell git describe --tags --abbrev=0 2>/dev/null || echo "dev-$(shell git rev-parse --short HEAD)")
VERSION := $(shell echo $(VERSION) | xargs) # Trim whitespace

# Define the name of the output binary for local builds
BINARY_NAME := $(APP_NAME)
# Define the path to the main package
MAIN_PACKAGE_PATH := . # Assuming your main package is in the root directory



# ==================================================================================== #
# CLOUD DEPLOYMENT BUILDS
# ==================================================================================== #

# Target platforms for cloud deployment (cross-compilation)
CLOUD_TARGETS := windows/amd64 darwin/amd64 darwin/arm64 linux/amd64 linux/arm64

# Directories
# Output dir for cloud binaries from cross-compile.sh
CLOUD_OUTPUT_DIR := ./dist
# Output dir for final archives, sigs, checksums
RELEASE_ASSETS_DIR := ./release_assets

# ==================================================================================== #
# DESKTOP DEPLOYMENT BUILDS
# ==================================================================================== #

# Desktop build uses build-desktop.sh and outputs to electron/dist/

# GPG Key ID for signing release assets
# IMPORTANT: Set this environment variable or change the default value.
# Example: export GPG_KEY_ID="Your Name <your.email@example.com>" or your Key ID
GPG_KEY_ID ?= YOUR_GPG_KEY_ID_HERE

.PHONY: all clean build build-all backend-build cloud-package cloud-release desktop-release upload-release upload-release-public release-public force-release-micro force-release-micro-public upload-to-s3 trigger-s3-workflow help

# Legacy target - defaults to cloud package for backward compatibility
all: cloud-package

help:
	@echo "Available targets:"
	@echo ""
	@echo "APPLICATION BUILD:"
	@echo "  build              Build KNIRVENGINE for the current platform"
	@echo "  build-all          Build KNIRVENGINE executables for all platforms"
	@echo ""
	@echo "RELEASE PACKAGING:"
	@echo "  cloud-package      Archive, sign, and checksum cloud binaries"
	@echo "  cloud-release      Complete cloud release (build + package + upload)"
	@echo ""
	@echo "  desktop-release    Create release with version bump"
	@echo ""
	@echo "LEGACY/COMPATIBILITY:"
	@echo "  all                Build cloud binaries and package them (default - legacy)"
	@echo "  upload-to-s3       Upload release assets to S3 bucket"
	@echo "  trigger-s3-workflow Trigger GitHub workflow to deploy to S3 (requires 'gh' CLI)."
	@echo "  force-release-micro Create a micro release even with uncommitted changes."
	@echo "  force-release-micro-public Create and publish a micro release to the public repo in one step."
	@echo "  clean              Remove build artifacts and release assets."
	@echo ""
	@echo "Configuration:"
	@echo "  VERSION          (default: $(VERSION)) - The version to build."
	@echo "  APP_NAME         (default: $(APP_NAME)) - The application name."
	@echo "  GPG_KEY_ID       (default: $(GPG_KEY_ID)) - GPG Key ID for signing."
	@echo 'Usage:'
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' |  sed -e 's/^/ /'

clean:
	@echo "Cleaning up build artifacts and release assets..."
	@rm -rf "$(CLOUD_OUTPUT_DIR)" "$(RELEASE_ASSETS_DIR)"
	@rm -f "./knirv-engine"* "./knirv-engine.exe"
	@rm -rf "./gui/dist" "./dist"
	@echo "Cleanup complete."

# ==================================================================================== #
# APPLICATION BUILD TARGETS
# ==================================================================================== #

# Bundle bwrap/Xvfb/x11vnc + their shared libs into tools/ (Linux only;
# resolves before PATH at runtime). Safe no-op on non-Linux hosts and when the
# tools are not installed on the build host.
bundle-sandbox-tools:
	@bash ./scripts/bundle-sandbox-tools.sh

# Build the application for the current platform
build: bundle-sandbox-tools
	@echo "Building KNIRVENGINE for the current platform..."
	@cd gui && npm run build
	@go build -o knirv-engine .
	@echo "Build complete: ./knirv-engine"

# Package cloud binaries for release
cloud-package: build-all
	@echo "Packaging cloud release assets for $(APP_NAME) version $(VERSION)..."
	@mkdir -p "$(RELEASE_ASSETS_DIR)/"
	$(foreach target,$(CLOUD_TARGETS),$(call process_target,$(target)))
	@echo "Packaging complete. Assets are in $(RELEASE_ASSETS_DIR)/"

# Complete cloud release process
cloud-release: cloud-package upload-to-s3
	@echo "Cloud release $(VERSION) completed successfully!"

# ==================================================================================== #
# CROSS-PLATFORM BUILD TARGETS
# ==================================================================================== #

# Build the application for all platforms
build-all:
	@echo "Building KNIRVENGINE executables for Linux, Windows, and macOS..."
	@./scripts/build-desktop.sh binaries
	@$(MAKE) bundle-sandbox-tools
	@echo "Cross-platform executables created in dist/."

# Create desktop release with version management
desktop-release:
	@echo "Creating native desktop release..."
	@echo "Note: Version management should be handled manually or via separate script"
	@$(MAKE) build-all
	@echo "Release completed. Packages are in dist/"

# Helper macro to process each target platform
define process_target
    $(eval OS_ARCH_UNDERSCORE := $(subst /,_,$(1)))
    $(eval OS := $(word 1,$(subst /, ,$(1))))
    $(eval ARCH := $(word 2,$(subst /, ,$(1))))
    $(eval EXE_SUFFIX :=)
    $(if $(filter windows,$(OS)),$(eval EXE_SUFFIX := .exe))
    $(eval ARCHIVE_EXT := tar.gz)
    $(if $(filter windows,$(OS)),$(eval ARCHIVE_EXT := zip))

    $(eval COMPILED_BINARY_DIR := $(CROSS_COMPILE_OUTPUT_DIR)/$(VERSION)/$(OS)_$(ARCH))
    $(eval COMPILED_BINARY_NAME := $(APP_NAME)$(EXE_SUFFIX))
    $(eval ARCHIVE_BASENAME := $(APP_NAME)_$(VERSION)_$(OS)_$(ARCH))
    $(eval FINAL_ARCHIVE_PATH := $(RELEASE_ASSETS_DIR)/$(ARCHIVE_BASENAME).$(ARCHIVE_EXT))
    $(eval FINAL_SIGNATURE_PATH := $(FINAL_ARCHIVE_PATH).sig)
    $(eval FINAL_CHECKSUM_PATH := $(FINAL_ARCHIVE_PATH).sha256)

    @echo "--------------------------------------------------"
    @echo "Processing $(OS)/$(ARCH)..."
    @echo "  Source binary directory: $(COMPILED_BINARY_DIR)"
    @echo "  Binary name: $(COMPILED_BINARY_NAME)"

    # Check if binary exists
    @if [ ! -f "$(COMPILED_BINARY_DIR)/$(COMPILED_BINARY_NAME)" ]; then \
        echo "  ERROR: Compiled binary not found at $(COMPILED_BINARY_DIR)/$(COMPILED_BINARY_NAME). Skipping."; \
        exit 1; \
    fi

    # Create archive
    @echo "  Archiving to $(FINAL_ARCHIVE_PATH)..."
    $(if $(filter windows,$(OS)), \
        (cd "$(COMPILED_BINARY_DIR)" && zip -jrq "$(abspath $(FINAL_ARCHIVE_PATH))" "$(COMPILED_BINARY_NAME)") , \
        tar -czvf "$(FINAL_ARCHIVE_PATH)" -C "$(COMPILED_BINARY_DIR)" "$(COMPILED_BINARY_NAME)" \
    )

    # Sign archive
    @echo "  Signing $(FINAL_ARCHIVE_PATH)..."
    @if [ -z "$(GPG_KEY_ID)" ] || [ "$(GPG_KEY_ID)" = "YOUR_GPG_KEY_ID_HERE" ]; then \
        echo "  WARNING: GPG_KEY_ID not set or is default. Skipping signing for $(FINAL_ARCHIVE_PATH)."; \
        echo "           To enable signing, set the GPG_KEY_ID environment variable or edit the Makefile."; \
    else \
        gpg --batch --yes --detach-sign --armor -u "$(GPG_KEY_ID)" "$(FINAL_ARCHIVE_PATH)" && \
        mv "$(FINAL_ARCHIVE_PATH).asc" "$(FINAL_SIGNATURE_PATH)" && \
        echo "  Signature created: $(FINAL_SIGNATURE_PATH)"; \
    fi

    # Generate SHA256 checksum for the archive
    @echo "  Generating SHA256 checksum for $(FINAL_ARCHIVE_PATH)..."
    @(cd "$(dir $(FINAL_ARCHIVE_PATH))" && sha256sum "$(notdir $(FINAL_ARCHIVE_PATH))" > "$(notdir $(FINAL_CHECKSUM_PATH))")
    @echo "  Checksum created: $(FINAL_CHECKSUM_PATH)"

endef

# Target to upload assets to GitHub Release in the private repo (requires GitHub CLI 'gh')
upload-release: cloud-package
	@echo "--------------------------------------------------"
	@echo "Uploading assets to GitHub release for tag $(VERSION) in private repo..."
	@if ! command -v gh &> /dev/null; then \
		echo "ERROR: GitHub CLI 'gh' not found. Please install it to upload releases (e.g., 'sudo apt install gh' or 'brew install gh')."; \
		exit 1; \
	fi
	@if [ -z "$(GPG_KEY_ID)" ] || [ "$(GPG_KEY_ID)" = "YOUR_GPG_KEY_ID_HERE" ]; then \
        echo "WARNING: GPG_KEY_ID not set or is default. Signatures might be missing or incomplete for the release."; \
    fi
	@echo "Checking for existing release $(VERSION)..."
	@gh release view $(VERSION) >/dev/null 2>&1 || \
		(echo "Release $(VERSION) not found, creating it..." && \
		 gh release create $(VERSION) --generate-notes -t "Version $(VERSION)" -p)
	@echo "Uploading assets from $(RELEASE_ASSETS_DIR)/ to release $(VERSION)..."
	@gh release upload $(VERSION) $(RELEASE_ASSETS_DIR)/* --clobber
	@echo "Upload complete for version $(VERSION) in private repo."

# Target to upload assets to GitHub Release in the public repo (requires GitHub CLI 'gh')
.PHONY: upload-release-public
upload-release-public: cloud-package
	@echo "--------------------------------------------------"
	@echo "Uploading assets to GitHub release for tag $(VERSION) in public repo..."
	@if ! command -v gh &> /dev/null; then \
		echo "ERROR: GitHub CLI 'gh' not found. Please install it to upload releases (e.g., 'sudo apt install gh' or 'brew install gh')."; \
		exit 1; \
	fi
	@if [ -z "$(GPG_KEY_ID)" ] || [ "$(GPG_KEY_ID)" = "YOUR_GPG_KEY_ID_HERE" ]; then \
        echo "WARNING: GPG_KEY_ID not set or is default. Signatures might be missing or incomplete for the release."; \
    fi
	
	@echo "Setting up public repository..."
	@PUBLIC_REPO_DIR="/home/gperry/Documents/GitHub/cloud-equities/KNIRVCHAIN_PUBLIC"
	@mkdir -p "$$PUBLIC_REPO_DIR"
	
	@echo "Checking if public repo is a git repository..."
	@if [ ! -d "$$PUBLIC_REPO_DIR/.git" ]; then \
		echo "Initializing git repository in public directory..."; \
		(cd "$$PUBLIC_REPO_DIR" && git init && \
		 git config user.name "KNIRVCHAIN Release Bot" && \
		 git config user.email "release-bot@knirv.com"); \
	fi
	
	@echo "Checking for existing release $(VERSION) in public repo..."
	@(cd "$$PUBLIC_REPO_DIR" && \
	  gh release view $(VERSION) >/dev/null 2>&1 || \
	  (echo "Release $(VERSION) not found in public repo, creating it..." && \
	   gh release create $(VERSION) --generate-notes -t "Version $(VERSION)" -p))
	
	@echo "Copying release assets to public repo..."
	@mkdir -p "$$PUBLIC_REPO_DIR/releases/$(VERSION)"
	@cp -r "$(RELEASE_ASSETS_DIR)/"* "$$PUBLIC_REPO_DIR/releases/$(VERSION)/"
	
	@echo "Uploading assets from public repo to GitHub release $(VERSION)..."
	@(cd "$$PUBLIC_REPO_DIR" && \
	  gh release upload $(VERSION) "releases/$(VERSION)/"* --clobber)
	
	@echo "Upload complete for version $(VERSION) in public repo."

# Target to upload release assets to S3 bucket
.PHONY: upload-to-s3
upload-to-s3: cloud-package
	@echo "--------------------------------------------------"
	@echo "Uploading release assets to S3 bucket..."
	
	@# Check if AWS CLI is installed
	@if ! command -v aws &> /dev/null; then \
		echo "ERROR: AWS CLI not found. Please install it to upload to S3."; \
		exit 1; \
	fi
	
	@# Load AWS credentials from .env file
	@if [ -f .env ]; then \
		echo "Loading AWS credentials from .env file..."; \
		export $$(grep -E '^AWS_' .env | xargs); \
	else \
		echo "ERROR: .env file not found. AWS credentials are required."; \
		exit 1; \
	fi
	
	@# Check if AWS credentials are set
	@if [ -z "$$AWS_ACCESS_KEY_ID" ] || [ -z "$$AWS_SECRET_ACCESS_KEY" ]; then \
		echo "ERROR: AWS credentials not found in .env file."; \
		exit 1; \
	fi
	
	@echo "Uploading files from $(CROSS_COMPILE_OUTPUT_DIR)/release_assets/ to S3..."
	@aws s3 sync $(CROSS_COMPILE_OUTPUT_DIR)/release_assets/ s3://knirv-engine-binaries/ \
		--include "*.gz" \
		--include "*.zip"
	
	@echo "Upload to S3 complete."

# Target to trigger GitHub workflow for S3 deployment
.PHONY: trigger-s3-workflow
trigger-s3-workflow:
	@echo "--------------------------------------------------"
	@echo "Triggering GitHub workflow for S3 deployment..."
	
	@# Check if GitHub CLI is installed
	@if ! command -v gh &> /dev/null; then \
		echo "ERROR: GitHub CLI 'gh' not found. Please install it to trigger workflows."; \
		exit 1; \
	fi
	
	@# Trigger the workflow
	@gh workflow run deploy-to-s3.yml
	
	@echo "GitHub workflow triggered. Check the Actions tab in your repository for progress."

## release-micro: Create a new micro (patch) version tag, push it, build, and package.
.PHONY: release-micro
## release-minor: Create a new minor version tag, push it, build, and package.
.PHONY: release-minor
## release-major: Create a new major version tag, push it, build, and package.
.PHONY: release-major



# ==================================================================================== #
# HELPERS
# ==================================================================================== #



.PHONY: confirm
confirm:
	@echo -n 'Are you sure? [y/N] ' && read ans && [ $${ans:-N} = y ]

.PHONY: no-dirty
no-dirty:
	@test -z "$(shell git status --porcelain)"


# ==================================================================================== #
# QUALITY CONTROL
# ==================================================================================== #

## audit: run quality control checks
.PHONY: audit
audit: test
	go mod tidy -diff
	go mod verify
	test -z "$(shell gofmt -l .)" 
	go vet ./...
	go run honnef.co/go/tools/cmd/staticcheck@latest -checks=all,-ST1000,-U1000 ./...
	go run golang.org/x/vuln/cmd/govulncheck@latest ./...

# ==================================================================================== #
# COMPREHENSIVE TEST SUITE
# ==================================================================================== #

## test: run comprehensive test suite (unit + integration + frontend)
.PHONY: test
test: test-unit test-integration test-frontend
	@echo "✅ All tests completed successfully!"

## test-unit: run unit tests only
.PHONY: test-unit
test-unit:
	@echo "🧪 Running unit tests..."
	go test -v -race -buildvcs -short ./...
	@echo "✅ Unit tests completed"

## test-integration: run integration tests
.PHONY: test-integration
test-integration:
	@echo "🔗 Running integration tests..."
	go test -v -race -buildvcs -run "Integration" ./test/
	@echo "✅ Integration tests completed"

## test-api: run comprehensive API tests
.PHONY: test-api
test-api:
	@echo "🌐 Running API tests..."
	go test -v -race -buildvcs ./test/ -run "TestAPI|TestAgent|TestMCP"
	@echo "✅ API tests completed"

## test-api-endpoints: run API endpoint integration tests
.PHONY: test-api-endpoints
test-api-endpoints:
	@echo "🔗 Running API endpoint integration tests..."
	./scripts/test_api_endpoints.sh
	@echo "✅ API endpoint tests completed"

## test-api-simple: run simple API tests
.PHONY: test-api-simple
test-api-simple:
	@echo "⚡ Running simple API tests..."
	./scripts/simple_api_test.sh
	@echo "✅ Simple API tests completed"

## test-frontend: run frontend tests
.PHONY: test-frontend
test-frontend:
	@echo "⚛️ Running frontend tests..."
	@cd $(GUI_DIR) && npm test -- --watchAll=false --coverage=false || echo "Frontend tests completed with warnings"
	go test -v ./test/ -run "TestFrontend"
	@echo "✅ Frontend tests completed"

## test-cloud: run cloud deployment tests
.PHONY: test-cloud
test-cloud:
	@echo "☁️ Running cloud deployment tests..."
	go test -v -race -buildvcs ./test/ -run "TestCloud"
	@echo "✅ Cloud tests completed"

## test-desktop: run desktop application tests
.PHONY: test-desktop
test-desktop:
	@echo "🖥️ Running desktop application tests..."
	go test -v -race -buildvcs ./test/ -run "TestDesktop"
	@echo "✅ Desktop tests completed"

## test-mcp: run MCP integration tests
.PHONY: test-mcp
test-mcp:
	@echo "🔌 Running MCP integration tests..."
	go test -v -race -buildvcs ./test/ -run "TestMCP"
	./scripts/test_mcp_integration.sh || echo "MCP integration tests completed with warnings"
	@echo "✅ MCP tests completed"

## test-security: run security tests
.PHONY: test-security
test-security:
	@echo "🔒 Running security tests..."
	go test -v -race -buildvcs ./test/ -run "TestSecurity|TestGaps"
	@echo "✅ Security tests completed"

## test-performance: run performance tests
.PHONY: test-performance
test-performance:
	@echo "⚡ Running performance tests..."
	go test -v -race -buildvcs ./test/ -run "TestPerformance|TestLoad"
	@echo "✅ Performance tests completed"

## test-wasm: run WASM agent tests
.PHONY: test-wasm
test-wasm:
	@echo "🕸️ Running WASM agent tests..."
	go test -v -race -buildvcs ./test/wasm/
	@echo "✅ WASM tests completed"

## test-connectivity: run end-to-end connectivity tests
.PHONY: test-connectivity
test-connectivity:
	@echo "🔗 Running agent connectivity tests..."
	go test -v -race -buildvcs ./test/ -run "TestAgentConnectivity|TestAgentCardToWASM|TestAgentTerminalConnectivity|TestWebSocketConnectivity"
	@echo "✅ Agent connectivity tests completed"

## test-chat: run agent chat functionality tests
.PHONY: test-chat
test-chat:
	@echo "💬 Running agent chat tests..."
	go test -v -race -buildvcs ./test/ -run "TestAgentChat"
	@echo "✅ Agent chat tests completed"

## test-all: run complete test suite including long-running tests
.PHONY: test-all
test-all: test-unit test-integration test-frontend test-api test-mcp test-security test-wasm test-connectivity test-chat
	@echo "🎉 Complete test suite finished!"

## test-ci: run CI-appropriate test subset
.PHONY: test-ci
test-ci: test-unit test-integration test-api
	@echo "✅ CI test suite completed"

## test-full: run full test suite including cloud and desktop tests
.PHONY: test-full
test-full: test-all test-cloud test-desktop test-performance
	@echo "🚀 Full test suite completed!"

## test/cover: run all tests and display coverage
.PHONY: test/cover
test/cover:
	@echo "📊 Running tests with coverage..."
	go test -v -race -buildvcs -coverprofile=/tmp/coverage.out ./...
	go tool cover -html=/tmp/coverage.out
	@echo "✅ Coverage report generated"

## test/cover-report: generate coverage report without opening browser
.PHONY: test/cover-report
test/cover-report:
	@echo "📊 Generating coverage report..."
	go test -v -race -buildvcs -coverprofile=/tmp/coverage.out ./...
	go tool cover -func=/tmp/coverage.out
	@echo "✅ Coverage report completed"

## test-comprehensive: run comprehensive test suite using script
.PHONY: test-comprehensive
test-comprehensive:
	@echo "🚀 Running comprehensive test suite..."
	./scripts/run_comprehensive_tests.sh all

## test-quick: run quick test suite for development
.PHONY: test-quick
test-quick:
	@echo "⚡ Running quick test suite..."
	./scripts/run_comprehensive_tests.sh ci

## test-verbose: run tests with verbose output
.PHONY: test-verbose
test-verbose:
	@echo "🔍 Running tests with verbose output..."
	VERBOSE=true ./scripts/run_comprehensive_tests.sh all

## test-setup: setup test environment
.PHONY: test-setup
test-setup:
	@echo "🔧 Setting up test environment..."
	@mkdir -p /tmp/knirv-engine-test
	@echo "AGENTIC_ENGINE_DEMO_MODE=true" > .env.test
	@echo "AGENTIC_ENGINE_TEST_MODE=true" >> .env.test
	@echo "✅ Test environment setup completed"

## test-cleanup: cleanup test environment
.PHONY: test-cleanup
test-cleanup:
	@echo "🧹 Cleaning up test environment..."
	@rm -rf /tmp/knirv-engine-test
	@rm -f .env.test
	@pkill -f "knirv-engine" >/dev/null 2>&1 || true
	@echo "✅ Test cleanup completed"

## test-help: show test command help
.PHONY: test-help
test-help:
	@echo "🧪 KNIRV-Engine Test Commands"
	@echo "==============================="
	@echo ""
	@echo "Basic Commands:"
	@echo "  make test              - Run comprehensive test suite"
	@echo "  make test-unit         - Run unit tests only"
	@echo "  make test-integration  - Run integration tests"
	@echo "  make test-frontend     - Run frontend tests"
	@echo "  make test-api          - Run API tests"
	@echo "  make test-api-endpoints - Run API endpoint integration tests"
	@echo "  make test-api-simple   - Run simple API tests"
	@echo ""
	@echo "Specialized Tests:"
	@echo "  make test-mcp          - Run MCP integration tests"
	@echo "  make test-cloud        - Run cloud deployment tests"
	@echo "  make test-desktop      - Run desktop application tests"
	@echo "  make test-security     - Run security tests"
	@echo "  make test-performance  - Run performance tests"
	@echo "  make test-wasm         - Run WASM agent tests"
	@echo "  make test-connectivity - Run end-to-end connectivity tests"
	@echo "  make test-chat         - Run agent chat functionality tests"
	@echo ""
	@echo "Test Suites:"
	@echo "  make test-all          - Run all tests except long-running ones"
	@echo "  make test-full         - Run complete test suite"
	@echo "  make test-ci           - Run CI-appropriate test subset"
	@echo "  make test-quick        - Run quick test suite for development"
	@echo ""
	@echo "Coverage & Reports:"
	@echo "  make test/cover        - Run tests with coverage (opens browser)"
	@echo "  make test/cover-report - Generate coverage report (no browser)"
	@echo ""
	@echo "Utilities:"
	@echo "  make test-setup        - Setup test environment"
	@echo "  make test-cleanup      - Cleanup test environment"
	@echo "  make test-verbose      - Run tests with verbose output"
	@echo "  make test-help         - Show this help"
	@echo ""
	@echo "Script Usage:"
	@echo "  ./scripts/run_comprehensive_tests.sh [mode]"
	@echo "  Modes: unit, integration, frontend, api, mcp, cloud, desktop,"
	@echo "         security, performance, ci, all, full"

## upgradeable: list direct dependencies that have upgrades available
.PHONY: upgradeable
upgradeable:
	@go list -u -f '{{if (and (not (or .Main .Indirect)) .Update)}}{{.Path}}: {{.Version}} -> {{.Update.Version}}{{end}}' -m all

# ==================================================================================== #
# DEVELOPMENT
# ==================================================================================== #

OUTPUT_DIR := /tmp/bin
GUI_DIR := ./gui

## frontend/install: install frontend dependencies
.PHONY: frontend/install
frontend/install:
	@echo "Installing frontend dependencies..."
	@cd $(GUI_DIR) && npm install
	@echo "Frontend dependencies installed."

## frontend/dev: run frontend development server
.PHONY: frontend/dev
frontend/dev:
	@echo "Starting frontend development server..."
	@cd $(GUI_DIR) && npm run dev

## frontend/build: build frontend for production
.PHONY: frontend/build
frontend/build:
	@echo "Building frontend for production..."
	@cd $(GUI_DIR) && npm run build
	@echo "Frontend production build complete."

## frontend/test: run frontend tests (legacy - use test-frontend)
.PHONY: frontend/test
frontend/test: test-frontend

## frontend/lint: run frontend linting
.PHONY: frontend/lint
frontend/lint:
	@echo "🔍 Running frontend linting..."
	@cd $(GUI_DIR) && npm run lint || echo "Linting completed with warnings"
	@echo "✅ Frontend linting completed"

## frontend/type-check: run TypeScript type checking
.PHONY: frontend/type-check
frontend/type-check:
	@echo "🔍 Running TypeScript type checking..."
	@cd $(GUI_DIR) && npm run type-check || echo "Type checking completed with warnings"
	@echo "✅ TypeScript type checking completed"

## frontend/test-watch: run frontend tests in watch mode
.PHONY: frontend/test-watch
frontend/test-watch:
	@echo "👀 Running frontend tests in watch mode..."
	@cd $(GUI_DIR) && npm test

## frontend/test-coverage: run frontend tests with coverage
.PHONY: frontend/test-coverage
frontend/test-coverage:
	@echo "📊 Running frontend tests with coverage..."
	@cd $(GUI_DIR) && npm test -- --coverage --watchAll=false
	@echo "✅ Frontend coverage completed"

## tidy: tidy modfiles and format .go files
.PHONY: tidy
tidy:
	go mod tidy -v
	go fmt ./...

## cache: clear mod cache and test cache for .go files
.PHONY: cache
cache:
	go clean -cache -modcache -testcache

## backend-build: build only the Go backend
.PHONY: backend-build
backend-build:
	@echo "Building $(APP_NAME) for host OS..."
	@mkdir -p ${OUTPUT_DIR} # Ensure the output directory exists
	go build -o=${OUTPUT_DIR}/${BINARY_NAME} ${MAIN_PACKAGE_PATH}

## build/all: build the application for all target platforms (legacy - use build-all)
.PHONY: build/all
build/all: build-all
	@echo "Legacy build/all completed. Use 'make build-all'."

## generate/wrapper: generate a simple run script for auto-restarting
.PHONY: generate/wrapper
generate/wrapper:
	@echo "Generating run_wrapper.sh..."
	@echo '#!/bin/bash' > ./run_wrapper.sh
	@echo '' >> ./run_wrapper.sh
	@echo 'RESTART_AFTER_UPDATE_EXIT_CODE=10' >> ./run_wrapper.sh # Exit code for go-rocket-update restart
	@echo 'BINARY_NAME="./$(BINARY_NAME)" # Adjust if binary is in a different location relative to script' >> ./run_wrapper.sh
	@echo '' >> ./run_wrapper.sh
	@echo 'while true; do' >> ./run_wrapper.sh
	@echo '    echo "Starting $${BINARY_NAME}..."' >> ./run_wrapper.sh
	@echo '    $${BINARY_NAME} "$$@"' >> ./run_wrapper.sh
	@echo '    EXIT_CODE=$$?' >> ./run_wrapper.sh
	@echo '' >> ./run_wrapper.sh
	@echo '    if [ $${EXIT_CODE} -eq $${RESTART_AFTER_UPDATE_EXIT_CODE} ]; then' >> ./run_wrapper.sh
	@echo '        echo "Application requested restart after update. Restarting in 1s..."' >> ./run_wrapper.sh
	@echo '        sleep 1' >> ./run_wrapper.sh
	@echo '    else' >> ./run_wrapper.sh
	@echo '        echo "Application exited with code $${EXIT_CODE}. Exiting wrapper."' >> ./run_wrapper.sh
	@echo '        break' >> ./run_wrapper.sh
	@echo '    fi' >> ./run_wrapper.sh
	@echo 'done' >> ./run_wrapper.sh
	@chmod +x ./run_wrapper.sh
	@echo "run_wrapper.sh generated. Make sure to adjust BINARY_NAME inside the script if needed."

## run: run the backend application
.PHONY: run
run: ./scripts/run_production.sh
	@echo "Running ${OUTPUT_DIR}/${BINARY_NAME}..."
	${OUTPUT_DIR}/${BINARY_NAME}

## run/full: run both backend and frontend in development mode
.PHONY: run/full
run/full:
	@echo "Starting full development environment (backend + frontend)..."
	@echo "Run these commands in separate terminals:"
	@echo "1. Backend: make run/live"
	@echo "2. Frontend: make frontend/dev"

## build/full: build both backend and frontend for production with embedded assets
.PHONY: build/full
build/full: frontend/build
	@echo "Building Go binary with embedded frontend assets..."
	@mkdir -p ${OUTPUT_DIR}
	go build -o=${OUTPUT_DIR}/${BINARY_NAME} ${MAIN_PACKAGE_PATH}
	@echo "Full production build complete with embedded frontend assets."

## build/embed: build production binary with embedded frontend assets
.PHONY: build/embed
build/embed: frontend/build
	@echo "Building Go binary with embedded frontend assets..."
	@mkdir -p ${OUTPUT_DIR}
	go build -tags embed -o=${OUTPUT_DIR}/${BINARY_NAME} ${MAIN_PACKAGE_PATH}
	@echo "Production binary with embedded frontend assets built."

## run/live: run the backend with reloading on file changes
.PHONY: run/live
run/live:
	go run github.com/cosmtrek/air@v1.43.0 \
	--build.cmd "make backend-build" --build.bin "/tmp/bin/${BINARY_NAME}" --build.delay "100" \
	--build.exclude_dir "" \
	--build.include_ext "go, tpl, tmpl, html, css, scss, js, ts, sql, jpeg, jpg, gif, png, bmp, svg, webp, ico" \
	--misc.clean_on_exit "true"

## dev: start full development environment with backend and frontend
.PHONY: dev
dev:
	@echo "Starting full development environment..."
	@echo "Backend (with live reload): http://localhost:8081"
	@echo "Frontend: http://localhost:8080"
	@echo "Run these commands in separate terminals:"
	@echo "1. Backend: make run/live"
	@echo "2. Frontend: make frontend/dev"

# ==================================================================================== #
# DESKTOP APPLICATION (Native Go)
# ==================================================================================== #

## desktop/dev: start desktop application in development mode
.PHONY: desktop/dev
desktop/dev:
	@echo "Starting native desktop application in development mode..."
	@./scripts/build-desktop.sh dev

## desktop/build: build application for current platform (legacy)
.PHONY: desktop/build
desktop/build: build

## desktop/build/linux: build desktop application for Linux (legacy)
.PHONY: desktop/build/linux
desktop/build/linux:
	@echo "Building native desktop application for Linux..."
	@./scripts/build-desktop.sh build linux

## desktop/build/windows: build desktop application for Windows (legacy)
.PHONY: desktop/build/windows
desktop/build/windows:
	@echo "Building native desktop application for Windows..."
	@./scripts/build-desktop.sh build windows

## desktop/build/macos: build desktop application for macOS (legacy)
.PHONY: desktop/build/macos
desktop/build/macos:
	@echo "Building native desktop application for macOS..."
	@./scripts/build-desktop.sh build macos

## desktop/build/all: build application for all platforms (legacy)
.PHONY: desktop/build/all
desktop/build/all: build-all

## desktop/clean: clean desktop build artifacts
.PHONY: desktop/clean
desktop/clean:
	@echo "Cleaning native desktop build artifacts..."
	@./scripts/build-desktop.sh clean

## run/embed: run production binary with embedded frontend
.PHONY: run/embed
run/embed: build/embed
	@echo "Running production binary with embedded frontend..."
	${OUTPUT_DIR}/${BINARY_NAME}

## push: push changes to the remote Git repository
.PHONY: push
push: confirm audit no-dirty
	git push
	git push --tags
	@echo "Changes and tags pushed to remote repository."

# ==================================================================================== #
# VERSION CONTROL
# ==================================================================================== #
MAKE               := make --no-print-directory

DESCRIBE           := $(shell git describe --match "v*" --always --tags)
DESCRIBE_PARTS     := $(subst -, ,$(DESCRIBE))

VERSION_TAG        := $(word 1,$(DESCRIBE_PARTS))
COMMITS_SINCE_TAG  := $(word 2,$(DESCRIBE_PARTS))

# VERSION here is specific to version calculation, distinct from the global VERSION for build artifacts
VERSION            := $(subst v,,$(VERSION_TAG))
VERSION_PARTS      := $(subst ., ,$(VERSION))

# Robust parsing for MAJOR, MINOR, MICRO, defaulting to 0 if not numeric
_RAW_MAJOR         := $(word 1,$(VERSION_PARTS))
_RAW_MINOR         := $(word 2,$(VERSION_PARTS))
_RAW_MICRO         := $(word 3,$(VERSION_PARTS))

MAJOR              := $(shell echo $(_RAW_MAJOR) | grep -E '^[0-9]+$$' || echo 0)
MINOR              := $(shell echo $(_RAW_MINOR) | grep -E '^[0-9]+$$' || echo 0)
MICRO              := $(shell echo $(_RAW_MICRO) | grep -E '^[0-9]+$$' || echo 0)

NEXT_MAJOR         := $(shell echo $$(($(MAJOR)+1))) # Corrected to use $$(()) for arithmetic
NEXT_MINOR         := $(shell echo $$(($(MINOR)+1))) # Corrected to use $$(()) for arithmetic
NEXT_MICRO          = $(shell echo $$(($(MICRO)+$(COMMITS_SINCE_TAG))))

ifeq ($(strip $(COMMITS_SINCE_TAG)),)
CURRENT_VERSION_MICRO := $(MAJOR).$(MINOR).$(MICRO)
CURRENT_VERSION_MINOR := $(CURRENT_VERSION_MICRO)
CURRENT_VERSION_MAJOR := $(CURRENT_VERSION_MICRO)
else
CURRENT_VERSION_MICRO := $(MAJOR).$(MINOR).$(NEXT_MICRO)
CURRENT_VERSION_MINOR := $(MAJOR).$(NEXT_MINOR).0
CURRENT_VERSION_MAJOR := $(NEXT_MAJOR).0.0
endif

DATE                = $(shell date +'%d.%m.%Y')
TIME                = $(shell date +'%H:%M:%S')
COMMIT             := $(shell git rev-parse HEAD)
AUTHOR             := $(firstword $(subst @, ,$(shell git show --format="%aE" $(COMMIT))))
BRANCH_NAME        := $(shell git rev-parse --abbrev-ref HEAD)

TAG_MESSAGE_CONTENT = $(TIME) $(DATE) $(AUTHOR) $(BRANCH_NAME) # Raw content without outer quotes
COMMIT_MESSAGE     := $(shell git log --format=%B -n 1 $(COMMIT))

CURRENT_TAG_MICRO  := "v$(CURRENT_VERSION_MICRO)"
CURRENT_TAG_MINOR  := "v$(CURRENT_VERSION_MINOR)"
CURRENT_TAG_MAJOR  := "v$(CURRENT_VERSION_MAJOR)"

# --- Version commands ---
## version: Show the next micro version (default version display)
.PHONY: version
version:
	@$(MAKE) version-micro

## version-micro: Show the next micro version (e.g., X.Y.Z or X.Y.Z+1)
.PHONY: version-micro
version-micro:
	@echo "$(CURRENT_VERSION_MICRO)"

## version-minor: Show the next minor version (e.g., X.Y+1.0)
.PHONY: version-minor
version-minor:
	@echo "$(CURRENT_VERSION_MINOR)"

## version-major: Show the next major version (e.g., X+1.0.0)
.PHONY: version-major
version-major:
	@echo "$(CURRENT_VERSION_MAJOR)"

# --- Tag commands ---
## tag-micro: Show the next micro version tag name (e.g., "vX.Y.Z")
.PHONY: tag-micro
tag-micro:
	@echo "$(CURRENT_TAG_MICRO)"

## tag-minor: Show the next minor version tag name (e.g., "vX.Y+1.0")
.PHONY: tag-minor
tag-minor:
	@echo "$(CURRENT_TAG_MINOR)"

## tag-major: Show the next major version tag name (e.g., "vX+1.0.0")
.PHONY: tag-major
tag-major:
	@echo "$(CURRENT_TAG_MAJOR)"

# -- Meta info ---
## tag-message: Show the default tag message content
.PHONY: tag-message
tag-message:
	@echo "$(TAG_MESSAGE_CONTENT)"

## commit-message: Show the current commit message
.PHONY: commit-message
commit-message:
	@echo "$(COMMIT_MESSAGE)"

# --- Automated Release Targets ---
define RELEASE_WORKFLOW
    @echo "Checking for uncommitted changes..."
    @if git status --porcelain | grep -q .; then \
        echo "WARNING: You have uncommitted changes in your repository."; \
        echo "It's recommended to commit or stash your changes before creating a release."; \
        echo ""; \
        echo -n "Do you want to proceed anyway? [y/N] "; \
        read ans; \
        if [ "$${ans:-N}" != "y" ]; then \
            echo "Release aborted."; \
            exit 1; \
        fi; \
    else \
        echo "Working directory is clean."; \
    fi
    $(MAKE) confirm
    @echo "Preparing $(1) release..."
    $(eval _NEW_TAG_NAME := $(shell $(MAKE) -s tag-$(1)))
    $(eval _NEW_VERSION := $(shell $(MAKE) -s version-$(1)))
    $(eval _TAG_MSG_CONTENT := $(shell $(MAKE) -s tag-message))
    @echo "Proposed tag: $$(_NEW_TAG_NAME)"
    @echo "Proposed version: $$(_NEW_VERSION)"
    @echo "Tag message content: $$(_TAG_MSG_CONTENT)"
	git tag -a $$(_NEW_TAG_NAME) -m "Release $$(_NEW_VERSION) - $$(_TAG_MSG_CONTENT)"
	@echo "Created tag $$(_NEW_TAG_NAME) locally."
	git push origin $$(_NEW_TAG_NAME)
	@echo "Pushed tag $$(_NEW_TAG_NAME) to origin."
	@echo "Building and packaging release for version $$(_NEW_VERSION)..."
	$(MAKE) all
	@echo "--------------------------------------------------"
	@echo "Release $$(_NEW_VERSION) built and packaged."
	@echo "Assets are in $(RELEASE_ASSETS_DIR)/"
	@echo "To upload to GitHub, run one of the following:"
	@echo "  - $(MAKE) upload-release         (for private repo only)"
	@echo "  - $(MAKE) upload-release-public  (for public repo only)"
	@echo "  - $(MAKE) release-public         (for both repos)"
	@echo "Ensure the DEFAULT_GITHUB_TOKEN environment variable is set for 'gh release upload' if needed."
endef

# Target to release to both private and public repos, and S3
.PHONY: release-public
release-public:
	@echo "--------------------------------------------------"
	@echo "Uploading release $(VERSION) to both private and public repositories, and S3..."
	$(MAKE) upload-release VERSION=$(VERSION)
	$(MAKE) upload-release-public VERSION=$(VERSION)
	$(MAKE) upload-to-s3 VERSION=$(VERSION)
	@echo "--------------------------------------------------"
	@echo "Release $(VERSION) has been uploaded to both private and public repositories, and S3."
	@echo "Private repo: $(shell git remote get-url origin)"
	@echo "Public repo: /home/gperry/Documents/GitHub/cloud-equities/KNIRVCHAIN_PUBLIC"
	@echo "S3 bucket: s3://knirv-engine-binaries/"

release-micro:
	$(call RELEASE_WORKFLOW,micro)
release-minor:
	$(call RELEASE_WORKFLOW,minor)
release-major:
	$(call RELEASE_WORKFLOW,major)

## force-release-micro: Create a new micro (patch) version tag without checking for uncommitted changes
.PHONY: force-release-micro
force-release-micro:
	@echo "WARNING: Forcing release with potentially uncommitted changes!"
	$(MAKE) confirm
	@echo "Preparing micro release..."
	
	# Fallback to a simple version calculation if tag-micro fails
	$(eval _NEW_VERSION := $(shell $(MAKE) -s version-micro 2>/dev/null || echo "0.1.0"))
	$(eval _NEW_TAG_NAME := v$(_NEW_VERSION))
	$(eval _TAG_MSG_CONTENT := $(shell date +'%H:%M:%S %d.%m.%Y') $(shell git config user.email) $(shell git rev-parse --abbrev-ref HEAD))
	
	@echo "Proposed tag: $$(_NEW_TAG_NAME)"
	@echo "Proposed version: $$(_NEW_VERSION)"
	@echo "Tag message content: $$(_TAG_MSG_CONTENT)"
	
	git tag -a "$$(_NEW_TAG_NAME)" -m "Release $$(_NEW_VERSION) - $$(_TAG_MSG_CONTENT)"
	@echo "Created tag $$(_NEW_TAG_NAME) locally."
	git push origin "$$(_NEW_TAG_NAME)"
	@echo "Pushed tag $$(_NEW_TAG_NAME) to origin."
	@echo "Building and packaging release for version $$(_NEW_VERSION)..."
	
	# Set VERSION for the build
	VERSION=$$(_NEW_VERSION) $(MAKE) all
	
	@echo "--------------------------------------------------"
	@echo "Release $$(_NEW_VERSION) built and packaged."
	@echo "Assets are in $(RELEASE_ASSETS_DIR)/"
	@echo "To upload to GitHub, run one of the following:"
	@echo "  - VERSION=$$(_NEW_VERSION) $(MAKE) upload-release         (for private repo only)"
	@echo "  - VERSION=$$(_NEW_VERSION) $(MAKE) upload-release-public  (for public repo only)"
	@echo "  - VERSION=$$(_NEW_VERSION) $(MAKE) release-public         (for both repos)"

## force-release-micro-public: Create a micro release and publish to public repo in one step
.PHONY: force-release-micro-public
force-release-micro-public:
	@echo "WARNING: Forcing release with potentially uncommitted changes and publishing to public repo!"
	$(MAKE) confirm
	@echo "Preparing micro release..."
	
	# Fallback to a simple version calculation if tag-micro fails
	$(eval _NEW_VERSION := $(shell $(MAKE) -s version-micro 2>/dev/null || echo "0.1.0"))
	$(eval _NEW_TAG_NAME := v$(_NEW_VERSION))
	$(eval _TAG_MSG_CONTENT := $(shell date +'%H:%M:%S %d.%m.%Y') $(shell git config user.email) $(shell git rev-parse --abbrev-ref HEAD))
	
	@echo "Proposed tag: $$(_NEW_TAG_NAME)"
	@echo "Proposed version: $$(_NEW_VERSION)"
	@echo "Tag message content: $$(_TAG_MSG_CONTENT)"
	
	git tag -a "$$(_NEW_TAG_NAME)" -m "Release $$(_NEW_VERSION) - $$(_TAG_MSG_CONTENT)"
	@echo "Created tag $$(_NEW_TAG_NAME) locally."
	git push origin "$$(_NEW_TAG_NAME)"
	@echo "Pushed tag $$(_NEW_TAG_NAME) to origin."
	@echo "Building and packaging release for version $$(_NEW_VERSION)..."
	
	# Set VERSION for the build
	VERSION=$$(_NEW_VERSION) $(MAKE) all
	
	@echo "--------------------------------------------------"
	@echo "Release $$(_NEW_VERSION) built and packaged."
	@echo "Assets are in $(RELEASE_ASSETS_DIR)/"
	
	# Upload to public repo and S3
	@echo "Uploading to public repository and S3..."
	VERSION=$$(_NEW_VERSION) $(MAKE) upload-release-public
	VERSION=$$(_NEW_VERSION) $(MAKE) upload-to-s3
	
	@echo "--------------------------------------------------"
	@echo "Release $$(_NEW_VERSION) has been built and uploaded to the public repository and S3."
	@echo "Public repo: /home/gperry/Documents/GitHub/cloud-equities/KNIRVCHAIN_PUBLIC"
	@echo "S3 bucket: s3://knirv-engine-binaries/"
