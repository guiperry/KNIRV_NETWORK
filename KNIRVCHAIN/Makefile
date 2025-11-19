# Run the following command to remove indents and chage to tabs: 
# mv -f makefile{,.old} && sed -E 's/^ {3,}/\t/' makefile.old >makefile

# Application and Version Configuration
APP_NAME := KNIRVCHAIN
# Determine version: Use git tag, or fallback to dev-commit_hash
VERSION ?= $(shell git describe --tags --abbrev=0 2>/dev/null || echo "dev-$(shell git rev-parse --short HEAD)")
VERSION := $(shell echo $(VERSION) | xargs) # Trim whitespace

# Define the name of the output binary for local builds
BINARY_NAME := $(APP_NAME)
# Define the path to the main package (updated for refactored structure)
MAIN_PACKAGE_PATH := ./cmd/knirvchain

# Package and Service Configuration
PKG_DIRS := $(shell find internal -type d -name "internal" -prune -o -type d -print | grep -E "internal/[^/]+$$" | sort)
# Node.js services are now located in internal/embedded/nodejs/ subdirectories
NODEJS_SERVICES_BASE := internal/embedded/nodejs
NODEJS_SERVICES := $(NODEJS_SERVICES_BASE)/payment/agent-payment-gateway $(NODEJS_SERVICES_BASE)/tunnel/agent-tunnel-registry $(NODEJS_SERVICES_BASE)/operator/operator-registry $(NODEJS_SERVICES_BASE)/webgui/webGUI $(NODEJS_SERVICES_BASE)/docsify/docsify
EMBEDDED_SERVICES := network-monitor economics
TEST_UNIT_DIR := ./tests/unit
TEST_INTEGRATION_DIR := ./tests/integration
TEST_E2E_DIR := ./tests/e2e



# ==================================================================================== #
# AUTOMATED PRODUCTION BUILDS
# ==================================================================================== #

# Target platforms (should align with PLATFORMS in cross-compile.sh)
TARGETS := windows/amd64 darwin/amd64 darwin/arm64 linux/amd64 linux/arm64

# Directories
CROSS_COMPILE_OUTPUT_DIR := ./dist  # Output dir for raw binaries from cross-compile.sh
RELEASE_ASSETS_DIR := ./release_assets # Output dir for final archives, sigs, checksums

# GPG Key ID for signing release assets
# IMPORTANT: Set this environment variable or change the default value.
# Example: export GPG_KEY_ID="Your Name <your.email@example.com>" or your Key ID
GPG_KEY_ID ?= YOUR_GPG_KEY_ID_HERE

.PHONY: all clean build-binaries package-release upload-release upload-release-public release-public force-release-micro force-release-micro-public force-release-micro-nowebview force-release-micro-nowebview-public help install-nodejs-deps build-nodejs-services build-embedded clean-nodejs test-internal lint-internal

all: package-release

help:
	@echo "Available targets:"
	@echo "  all                Build binaries and package them for release (default)."
	@echo "  build-binaries     Compile binaries for all target platforms using cross-compile.sh."
	@echo "  package-release    Archive, sign, and checksum the compiled binaries."
	@echo "  upload-release     Upload packaged assets to a GitHub release in the private repo (requires 'gh' CLI)."
	@echo "  upload-release-public Upload packaged assets to a GitHub release in the public repo (requires 'gh' CLI)."
	@echo "  release-public     Upload packaged assets to both private and public repos (requires 'gh' CLI)."
	@echo "  force-release-micro Create a micro release even with uncommitted changes."
	@echo "  force-release-micro-public Create and publish a micro release to the public repo in one step."
	@echo "  force-release-micro-nowebview Create a micro release without WebView dependencies."
	@echo "  force-release-micro-nowebview-public Create and publish a micro release without WebView to the public repo."
	@echo "  clean              Remove build artifacts and release assets."
	@echo ""
	@echo "Development targets:"
	@echo "  docs               Generate documentation from docs/ folder into documentation/ folder."
	@echo "  build              Build the application for host OS (client role) and generate documentation."
	@echo "  build/root         Build the application for host OS (root role)."
	@echo "  build/bootnode     Build the application for host OS (bootnode role)."
	@echo "  build/dev          Build the application for host OS (developer role)."
	@echo "  build/all-roles    Build the application for all roles."
	@echo "  build/all          Build the application for all target platforms (client role)."
	@echo "  build/all/root     Build the application for all target platforms (root role)."
	@echo "  build/all/bootnode Build the application for all target platforms (bootnode role)."
	@echo "  build/all/dev      Build the application for all target platforms (developer role)."
	@echo "  build/all-platforms-all-roles Build the application for all platforms and all roles."
	@echo ""
	@echo "Node.js Services:"
	@echo "  install-nodejs-deps Install Node.js dependencies for all embedded services."
	@echo "  build-nodejs-services Build all embedded Node.js services."
	@echo "  clean-nodejs       Clean Node.js build artifacts and node_modules."
	@echo ""
	@echo "Package Management:"
	@echo "  test-internal PKG=<name> Test a specific package in internal/ directory."
	@echo "  lint-internal PKG=<name> Lint a specific package in internal/ directory."
	@echo "  build-embedded     Build embedded service binaries."
	@echo ""
	@echo "Configuration:"
	@echo "  VERSION          (default: $(VERSION)) - The version to build."
	@echo "  APP_NAME         (default: $(APP_NAME)) - The application name."
	@echo "  GPG_KEY_ID       (default: $(GPG_KEY_ID)) - GPG Key ID for signing."
	@echo 'Usage:'
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' |  sed -e 's/^/ /'

clean:
	@echo "Cleaning up build artifacts and release assets..."
	@rm -rf "$(CROSS_COMPILE_OUTPUT_DIR)" "$(RELEASE_ASSETS_DIR)"
	@$(MAKE) clean-nodejs
	@echo "Cleanup complete."

# ==================================================================================== #
# NODE.JS SERVICES MANAGEMENT
# ==================================================================================== #

## install-nodejs-deps: Install Node.js dependencies for all embedded services
.PHONY: install-nodejs-deps
install-nodejs-deps:
	@echo "Installing Node.js dependencies for embedded services..."
	@for service in $(NODEJS_SERVICES); do \
		if [ -d "$$service" ] && [ -f "$$service/package.json" ]; then \
			echo "Installing dependencies for $$service..."; \
			(cd "$$service" && npm install); \
		else \
			echo "Skipping $$service (no package.json found)"; \
		fi; \
	done
	@echo "Node.js dependencies installation complete."

## build-nodejs-services: Build all embedded Node.js services
.PHONY: build-nodejs-services
build-nodejs-services: install-nodejs-deps
	@echo "Building embedded Node.js services..."
	@for service in $(NODEJS_SERVICES); do \
		if [ -d "$$service" ] && [ -f "$$service/package.json" ]; then \
			echo "Building $$service..."; \
			(cd "$$service" && npm run build 2>/dev/null || echo "No build script for $$service"); \
		fi; \
	done
	@echo "Node.js services build complete."

## clean-nodejs: Clean Node.js build artifacts and node_modules
.PHONY: clean-nodejs
clean-nodejs:
	@echo "Cleaning Node.js artifacts..."
	@for service in $(NODEJS_SERVICES); do \
		if [ -d "$$service" ]; then \
			echo "Cleaning $$service..."; \
			rm -rf "$$service/node_modules" "$$service/dist" "$$service/.next" "$$service/build"; \
		fi; \
	done
	@echo "Node.js cleanup complete."

## build-embedded: Build embedded service binaries
.PHONY: build-embedded
build-embedded:
	@echo "Building embedded service binaries..."
	@for service in $(EMBEDDED_SERVICES); do \
		if [ -d "$$service" ]; then \
			echo "Building $$service binary..."; \
			if [ -f "$$service/Makefile" ]; then \
				(cd "$$service" && make build); \
			elif [ -f "$$service/go.mod" ]; then \
				(cd "$$service" && go build -o bin/$${service} ./cmd/...); \
			else \
				echo "No build method found for $$service"; \
			fi; \
		fi; \
	done
	@echo "Embedded services build complete."

# Target to build network-monitor binary first (optional, fallback to embedded)
build-network-monitor:
	@echo "Attempting to build network-monitor binary..."
	@mkdir -p bin
	@if [ -d "network-monitor" ] && [ -f "network-monitor/go.mod" ]; then \
		echo "Building local network-monitor..."; \
		(cd network-monitor && go build -o ../bin/knirv-network-monitor ./cmd/monitor) && \
		echo "Network-monitor binary built successfully."; \
	elif cd ../network-monitor 2>/dev/null && go build -o ../KNIRVCHAIN/bin/knirv-network-monitor ./cmd/monitor 2>/dev/null; then \
		echo "Network-monitor binary built successfully from parent directory."; \
	else \
		echo "Network-monitor binary build failed, will use embedded monitor instead."; \
	fi

# Target to run the cross-compilation script
build-binaries: build-embedded build-nodejs-services build-network-monitor
	@echo "Starting cross-compilation for $(APP_NAME) version $(VERSION)..."
	@echo "NO_WEBVIEW flag is set to: $(NO_WEBVIEW)"
	@VERSION=$(VERSION) BINARY_NAME=$(APP_NAME) NO_WEBVIEW=$(NO_WEBVIEW) ./scripts/cross-compile.sh
	@echo "Cross-compilation finished."

# Target to package, sign, and checksum the compiled binaries
package-release: build-binaries
	@echo "Packaging release assets for $(APP_NAME) version $(VERSION)..."
	@mkdir -p "$(RELEASE_ASSETS_DIR)/$(VERSION)"
	$(foreach target,$(TARGETS),$(call process_target,$(target)))
	@echo "Packaging complete. Assets are in $(RELEASE_ASSETS_DIR)/$(VERSION)"

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
    $(eval FINAL_ARCHIVE_PATH := $(RELEASE_ASSETS_DIR)/$(VERSION)/$(ARCHIVE_BASENAME).$(ARCHIVE_EXT))
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
upload-release: package-release
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
	@echo "Uploading assets from $(RELEASE_ASSETS_DIR)/$(VERSION)/ to release $(VERSION)..."
	@gh release upload $(VERSION) $(RELEASE_ASSETS_DIR)/$(VERSION)/* --clobber
	@echo "Upload complete for version $(VERSION) in private repo."

# Target to upload assets to GitHub Release in the public repo (requires GitHub CLI 'gh')
.PHONY: upload-release-public
upload-release-public: package-release
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
	@PUBLIC_REPO_DIR="/home/gperry/Documents/GitHub/cloud-equities/KNIRVORACLE_PUBLIC"
	@mkdir -p "$$PUBLIC_REPO_DIR"
	
	@echo "Checking if public repo is a git repository..."
	@if [ ! -d "$$PUBLIC_REPO_DIR/.git" ]; then \
		echo "Initializing git repository in public directory..."; \
		(cd "$$PUBLIC_REPO_DIR" && git init && \
		 git config user.name "KNIRVCHAIN Release Bot" && \
		 git config user.email "release-bot@example.com"); \
	fi
	
	@echo "Checking for existing release $(VERSION) in public repo..."
	@(cd "$$PUBLIC_REPO_DIR" && \
	  gh release view $(VERSION) >/dev/null 2>&1 || \
	  (echo "Release $(VERSION) not found in public repo, creating it..." && \
	   gh release create $(VERSION) --generate-notes -t "Version $(VERSION)" -p))
	
	@echo "Copying release assets to public repo..."
	@mkdir -p "$$PUBLIC_REPO_DIR/releases/$(VERSION)"
	@cp -r "$(RELEASE_ASSETS_DIR)/$(VERSION)/"* "$$PUBLIC_REPO_DIR/releases/$(VERSION)/"
	
	@echo "Uploading assets from public repo to GitHub release $(VERSION)..."
	@(cd "$$PUBLIC_REPO_DIR" && \
	  gh release upload $(VERSION) "releases/$(VERSION)/"* --clobber)
	
	@echo "Upload complete for version $(VERSION) in public repo."

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
	test -z "$(shell gofmt -l ./internal ./cmd ./tests)"
	go vet ./internal/... ./cmd/... $(TEST_UNIT_DIR)/... $(TEST_INTEGRATION_DIR)/...
	go run honnef.co/go/tools/cmd/staticcheck@latest -checks=all,-ST1000,-U1000 ./internal/... ./cmd/...
	go run golang.org/x/vuln/cmd/govulncheck@latest ./...

## test: run all tests (unit tests + integration tests)
.PHONY: test
test: test-unit test-integration

## test-unit: run unit tests only
.PHONY: test-unit
test-unit:
	@echo "Cleaning up any existing test databases..."
	@./scripts/cleanup-test-artifacts.sh 2>/dev/null || true
	@mkdir -p test-reports logs
	@echo "Running unit tests from $(TEST_UNIT_DIR)..."
	go test -v -race -buildvcs -coverprofile=test-reports/coverage.out $(TEST_UNIT_DIR)/...
	@echo "Running package tests..."
	go test -v -race -buildvcs -coverprofile=test-reports/internal-coverage.out ./internal/...
	@echo "Cleaning up test databases after tests..."
	@./scripts/cleanup-test-artifacts.sh 2>/dev/null || true

## test-integration: run integration tests across all KNIRV components
.PHONY: test-integration
test-integration:
	@echo "Running comprehensive integration tests..."
	@mkdir -p test-reports logs
	@if [ -d "$(TEST_INTEGRATION_DIR)" ]; then \
		echo "Running integration tests from $(TEST_INTEGRATION_DIR)..."; \
		go test -v -race -buildvcs $(TEST_INTEGRATION_DIR)/...; \
	fi
	@if [ -f "./scripts/run-integration-tests.sh" ]; then \
		echo "Running integration test script..."; \
		./scripts/run-integration-tests.sh; \
	fi

## test-internal: run tests for a specific package
.PHONY: test-internal
test-internal:
	@if [ -z "$(PKG)" ]; then \
		echo "Usage: make test-internal PKG=<package_name>"; \
		echo "Available packages:"; \
		ls -1 internal/; \
		exit 1; \
	fi
	@if [ ! -d "internal/$(PKG)" ]; then \
		echo "Package internal/$(PKG) does not exist"; \
		exit 1; \
	fi
	@echo "Running tests for package $(PKG)..."
	go test -v -race -buildvcs ./internal/$(PKG)/...

## lint-internal: lint a specific package
.PHONY: lint-internal
lint-internal:
	@if [ -z "$(PKG)" ]; then \
		echo "Usage: make lint-internal PKG=<package_name>"; \
		echo "Available packages:"; \
		ls -1 internal/; \
		exit 1; \
	fi
	@if [ ! -d "internal/$(PKG)" ]; then \
		echo "Package internal/$(PKG) does not exist"; \
		exit 1; \
	fi
	@echo "Linting package $(PKG)..."
	go vet ./internal/$(PKG)/...
	go run honnef.co/go/tools/cmd/staticcheck@latest ./internal/$(PKG)/...

## test/cover: run all tests and display coverage
.PHONY: test/cover
test/cover:
	@mkdir -p test-reports logs
	@./scripts/cleanup-test-artifacts.sh 2>/dev/null || true
	go test -v -race -buildvcs -coverprofile=test-reports/coverage.out $(TEST_UNIT_DIR)/... ./internal/...
	go tool cover -html=test-reports/coverage.out
	@./scripts/cleanup-test-artifacts.sh 2>/dev/null || true

## upgradeable: list direct dependencies that have upgrades available
.PHONY: upgradeable
upgradeable:
	@go list -u -f '{{if (and (not (or .Main .Indirect)) .Update)}}{{.Path}}: {{.Version}} -> {{.Update.Version}}{{end}}' -m all

# ==================================================================================== #
# DEVELOPMENT
# ==================================================================================== #

OUTPUT_DIR := /tmp/bin

## docs: generate documentation from docs/ folder into documentation/docsify folder
.PHONY: docs
docs:
	@echo "Installing documentation dependencies..."
	@if [ -d "documentation" ] && [ -f "documentation/package.json" ]; then \
		cd documentation && npm install; \
	else \
		echo "No documentation package.json found, skipping npm install"; \
	fi
	@echo "Generating documentation..."
	@if [ -f "scripts/organize_docs.js" ]; then \
		node scripts/organize_docs.js; \
	else \
		echo "Documentation generation script not found, skipping"; \
	fi
	@echo "Documentation generation complete"

## tidy: tidy modfiles and format .go files
.PHONY: tidy
tidy:
	go mod tidy -v
	go fmt ./...

## cache: clear mod cache and test cache for .go files
.PHONY: cache
cache:
	go clean -cache -modcache -testcache

## cleanup-test: clean up test artifacts and databases
.PHONY: cleanup-test
cleanup-test:
	@echo "Cleaning up test artifacts..."
	@./scripts/cleanup-test-artifacts.sh

## build: build the application (default client role) and generate documentation
.PHONY: build
build: docs build-embedded build-nodejs-services build-network-monitor
	@echo "Building $(APP_NAME) for host OS (client role)..."
	@mkdir -p ${OUTPUT_DIR} # Ensure the output directory exists
	go build -tags="client testnet no_webview" -o=${OUTPUT_DIR}/${BINARY_NAME} ${MAIN_PACKAGE_PATH}

## build/root: build the application with root role
.PHONY: build/root
build/root: build-embedded build-nodejs-services build-network-monitor
	@echo "Building $(APP_NAME) for host OS (root role)..."
	@mkdir -p ${OUTPUT_DIR} # Ensure the output directory exists
	go build -tags="root testnet no_webview" -o=${OUTPUT_DIR}/${BINARY_NAME}_root ${MAIN_PACKAGE_PATH}

## build/bootnode: build the application with bootnode role
.PHONY: build/bootnode
build/bootnode: build-embedded build-nodejs-services build-network-monitor
	@echo "Building $(APP_NAME) for host OS (bootnode role)..."
	@mkdir -p ${OUTPUT_DIR} # Ensure the output directory exists
	go build -tags=bootnode -o=${OUTPUT_DIR}/${BINARY_NAME}_bootnode ${MAIN_PACKAGE_PATH}

## build/dev: build the application with developer role
.PHONY: build/dev
build/dev: build-embedded build-nodejs-services build-network-monitor
	@echo "Building $(APP_NAME) for host OS (developer role)..."
	@mkdir -p ${OUTPUT_DIR} # Ensure the output directory exists
	go build -tags=dev -o=${OUTPUT_DIR}/${BINARY_NAME}_dev ${MAIN_PACKAGE_PATH}

## build/all-roles: build the application for all roles
.PHONY: build/all-roles
build/all-roles: docs build build/root build/bootnode build/dev
	@echo "All role-specific builds completed. See ${OUTPUT_DIR}/ for output."

## build/all: build the application for all target platforms (default client role)
.PHONY: build/all
build/all: docs build-embedded build-nodejs-services
	@echo "Building for all target platforms using cross-compile.sh script (client role)..."
	@echo "Make sees VERSION as: [$(VERSION)]" # This VERSION is the release version
	@echo "Make sees BINARY_NAME as: [$(BINARY_NAME)]" # This BINARY_NAME is for cross-compile.sh
	@VERSION=$(VERSION) BINARY_NAME=$(BINARY_NAME) NODE_ROLE=client bash ./scripts/cross-compile.sh
	@echo "All builds completed. See ./dist/$(VERSION)/ for output."

## build/all/root: build the application for all target platforms with root role
.PHONY: build/all/root
build/all/root: build-embedded build-nodejs-services
	@echo "Building for all target platforms using cross-compile.sh script (root role)..."
	@VERSION=$(VERSION) BINARY_NAME=$(BINARY_NAME) NODE_ROLE=root bash ./scripts/cross-compile.sh
	@echo "All builds completed. See ./dist/$(VERSION)/ for output."

## build/all/bootnode: build the application for all target platforms with bootnode role
.PHONY: build/all/bootnode
build/all/bootnode: build-embedded build-nodejs-services
	@echo "Building for all target platforms using cross-compile.sh script (bootnode role)..."
	@VERSION=$(VERSION) BINARY_NAME=$(BINARY_NAME) NODE_ROLE=bootnode bash ./scripts/cross-compile.sh
	@echo "All builds completed. See ./dist/$(VERSION)/ for output."

## build/all/dev: build the application for all target platforms with developer role
.PHONY: build/all/dev
build/all/dev: build-embedded build-nodejs-services
	@echo "Building for all target platforms using cross-compile.sh script (developer role)..."
	@VERSION=$(VERSION) BINARY_NAME=$(BINARY_NAME) NODE_ROLE=dev bash ./scripts/cross-compile.sh
	@echo "All builds completed. See ./dist/$(VERSION)/ for output."

## build/all-platforms-all-roles: build the application for all target platforms and all roles
.PHONY: build/all-platforms-all-roles
build/all-platforms-all-roles: docs build/all build/all/root build/all/bootnode build/all/dev
	@echo "All platform and role-specific builds completed. See ./dist/$(VERSION)/ for output."

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

## run: run the  application
.PHONY: run
run: build
	@echo "Running ${OUTPUT_DIR}/${BINARY_NAME}..."
	${OUTPUT_DIR}/${BINARY_NAME}

## run/live: run the application with reloading on file changes
.PHONY: run/live
run/live:
	go run github.com/cosmtrek/air@v1.43.0 \
	--build.cmd "make build" --build.bin "/tmp/bin/${BINARY_NAME}" --build.delay "100" \
	--build.exclude_dir "" \
	--build.include_ext "go, tpl, tmpl, html, css, scss, js, ts, sql, jpeg, jpg, gif, png, bmp, svg, webp, ico" \
	--misc.clean_on_exit "true"

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
	@echo "Assets are in $(RELEASE_ASSETS_DIR)/$$(_NEW_VERSION)/"
	@echo "To upload to GitHub, run one of the following:"
	@echo "  - $(MAKE) upload-release         (for private repo only)"
	@echo "  - $(MAKE) upload-release-public  (for public repo only)"
	@echo "  - $(MAKE) release-public         (for both repos)"
	@echo "Ensure the DEFAULT_GITHUB_TOKEN environment variable is set for 'gh release upload' if needed."
endef

# Target to release to both private and public repos
.PHONY: release-public
release-public:
	@echo "--------------------------------------------------"
	@echo "Uploading release $(VERSION) to both private and public repositories..."
	$(MAKE) upload-release VERSION=$(VERSION)
	$(MAKE) upload-release-public VERSION=$(VERSION)
	@echo "--------------------------------------------------"
	@echo "Release $(VERSION) has been uploaded to both private and public repositories."
	@echo "Private repo: $(shell git remote get-url origin)"
	@echo "Public repo: /home/gperry/Documents/GitHub/cloud-equities/KNIRVORACLE_PUBLIC"

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
	@echo "Assets are in $(RELEASE_ASSETS_DIR)/$$(_NEW_VERSION)/"
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
	@echo "Assets are in $(RELEASE_ASSETS_DIR)/$$(_NEW_VERSION)/"
	
	# Upload to public repo
	@echo "Uploading to public repository..."
	VERSION=$$(_NEW_VERSION) $(MAKE) upload-release-public
	
	@echo "--------------------------------------------------"
	@echo "Release $$(_NEW_VERSION) has been built and uploaded to the public repository."
	@echo "Public repo: /home/gperry/Documents/GitHub/cloud-equities/KNIRVORACLE_PUBLIC"

## force-release-micro-nowebview: Create a micro release without WebView dependencies
.PHONY: force-release-micro-nowebview
force-release-micro-nowebview:
	@echo "WARNING: Forcing release with potentially uncommitted changes and without WebView dependencies!"
	$(MAKE) confirm
	@echo "Preparing micro release without WebView..."
	
	@# Calculate the new version
	@NEW_VERSION=$$($(MAKE) -s version-micro 2>/dev/null || echo "0.1.0"); \
	NEW_TAG_NAME="v$$NEW_VERSION-nowebview"; \
	TAG_MSG_CONTENT="$$(date +'%H:%M:%S %d.%m.%Y') $$(git config user.email) $$(git rev-parse --abbrev-ref HEAD) [NoWebView]"; \
	\
	echo "Proposed tag: $$NEW_TAG_NAME"; \
	echo "Proposed version: $$NEW_VERSION"; \
	echo "Tag message content: $$TAG_MSG_CONTENT"; \
	\
	git tag -a "$$NEW_TAG_NAME" -m "Release $$NEW_VERSION - $$TAG_MSG_CONTENT"; \
	echo "Created tag $$NEW_TAG_NAME locally."; \
	git push origin "$$NEW_TAG_NAME" || echo "Warning: Could not push tag to remote repository"; \
	echo "Building and packaging release for version $$NEW_VERSION without WebView..."; \
	\
	# Set VERSION and NO_WEBVIEW for the build \
	NO_WEBVIEW=1 VERSION=$$NEW_VERSION $(MAKE) all; \
	\
	echo "--------------------------------------------------"; \
	echo "Release $$NEW_VERSION (NoWebView) built and packaged."; \
	echo "Assets are in $(RELEASE_ASSETS_DIR)/$$NEW_VERSION/"; \
	echo "To upload to GitHub, run one of the following:"; \
	echo "  - VERSION=$$NEW_VERSION $(MAKE) upload-release         (for private repo only)"; \
	echo "  - VERSION=$$NEW_VERSION $(MAKE) upload-release-public  (for public repo only)"

## force-release-micro-nowebview-public: Create a micro release without WebView and publish to public repo in one step
.PHONY: force-release-micro-nowebview-public
force-release-micro-nowebview-public:
	@echo "WARNING: Forcing release without WebView dependencies and publishing to public repo!"
	$(MAKE) confirm
	@echo "Preparing micro release without WebView..."
	
	@# Calculate the new version
	@NEW_VERSION=$$($(MAKE) -s version-micro 2>/dev/null || echo "0.1.0"); \
	NEW_TAG_NAME="v$$NEW_VERSION-nowebview"; \
	TAG_MSG_CONTENT="$$(date +'%H:%M:%S %d.%m.%Y') $$(git config user.email) $$(git rev-parse --abbrev-ref HEAD) [NoWebView]"; \
	\
	echo "Proposed tag: $$NEW_TAG_NAME"; \
	echo "Proposed version: $$NEW_VERSION"; \
	echo "Tag message content: $$TAG_MSG_CONTENT"; \
	\
	git tag -f -a "$$NEW_TAG_NAME" -m "Release $$NEW_VERSION - $$TAG_MSG_CONTENT"; \
	echo "Created tag $$NEW_TAG_NAME locally."; \
	git push origin "$$NEW_TAG_NAME" || echo "Warning: Could not push tag to remote repository"; \
	echo "Building and packaging release for version $$NEW_VERSION without WebView..."; \
	\
	# Set VERSION and NO_WEBVIEW for the build \
	NO_WEBVIEW=1 VERSION=$$NEW_VERSION $(MAKE) all; \
	\
	echo "--------------------------------------------------"; \
	echo "Release $$NEW_VERSION (NoWebView) built and packaged."; \
	echo "Assets are in $(RELEASE_ASSETS_DIR)/$$NEW_VERSION/"; \
	\
	# Upload to public repo \
	echo "Uploading to public repository..."; \
	NO_WEBVIEW=1 VERSION=$$NEW_VERSION $(MAKE) upload-release-public; \
	\
	echo "--------------------------------------------------"; \
	echo "Release $$NEW_VERSION (NoWebView) has been built and uploaded to the public repository."; \
	echo "Public repo: /home/gperry/Documents/GitHub/cloud-equities/KNIRVORACLE_PUBLIC"
