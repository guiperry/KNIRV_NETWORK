# Run the following command to remove indents and chage to tabs: 
# mv -f makefile{,.old} && sed -E 's/^ {3,}/\t/' makefile.old >makefile

# Application and Version Configuration
APP_NAME := KNIRV-NEXUS
# Determine version: Use git tag, or fallback to dev-commit_hash
VERSION ?= $(shell git describe --tags --abbrev=0 2>/dev/null || echo "dev-$(shell git rev-parse --short HEAD)")
VERSION := $(shell echo $(VERSION) | xargs) # Trim whitespace

# Define the name of the output binary for local builds
BINARY_NAME := $(APP_NAME)
# Define the path to the main package
MAIN_PACKAGE_PATH := . # Assuming your main package is in the root directory



# ==================================================================================== #
# AUTOMATED PRODUCTION BUILDS
# ==================================================================================== #

# Target platforms (should align with PLATFORMS in cross-compile.sh)
TARGETS := windows/amd64 darwin/amd64 darwin/arm64 linux/amd64 linux/arm64

# Directories
# Output dir for raw binaries from cross-compile.sh
CROSS_COMPILE_OUTPUT_DIR := ./dist
# Output dir for final archives, sigs, checksums
RELEASE_ASSETS_DIR := ./release_assets

# GPG Key ID for signing release assets
# IMPORTANT: Set this environment variable or change the default value.
# Example: export GPG_KEY_ID="Your Name <your.email@example.com>" or your Key ID
GPG_KEY_ID ?= YOUR_GPG_KEY_ID_HERE

.PHONY: all clean build-binaries package-release upload-release upload-release-public release-public force-release-micro force-release-micro-public help

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
	@rm -rf "$(CROSS_COMPILE_OUTPUT_DIR)" "$(RELEASE_ASSETS_DIR)"
	@echo "Cleanup complete."

# Target to run the cross-compilation script
build-binaries:
	@echo "Starting cross-compilation for $(APP_NAME) version $(VERSION)..."
	@VERSION=$(VERSION) BINARY_NAME=$(APP_NAME) ./scripts/cross-compile.sh
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
	@PUBLIC_REPO_DIR="/home/gperry/Documents/GitHub/cloud-equities/KNIRVCHAIN_PUBLIC"
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
	test -z "$(shell gofmt -l .)" 
	go vet ./...
	go run honnef.co/go/tools/cmd/staticcheck@latest -checks=all,-ST1000,-U1000 ./...
	go run golang.org/x/vuln/cmd/govulncheck@latest ./...

## test: run all tests
.PHONY: test
test:
	go test -v -race -buildvcs ./...

## test/cover: run all tests and display coverage
.PHONY: test/cover
test/cover:
	go test -v -race -buildvcs -coverprofile=/tmp/coverage.out ./...
	go tool cover -html=/tmp/coverage.out

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

## frontend/clean: clean frontend dependencies and build artifacts
.PHONY: frontend/clean
frontend/clean:
	@echo "Cleaning frontend dependencies and build artifacts..."
	@cd $(GUI_DIR) && rm -rf node_modules dist .vite
	@echo "Frontend cleaned."

## frontend/reinstall: clean and reinstall frontend dependencies
.PHONY: frontend/reinstall
frontend/reinstall: frontend/clean frontend/install
	@echo "Frontend dependencies reinstalled."

## frontend/dev: run frontend development server
.PHONY: frontend/dev
frontend/dev:
	@echo "Starting frontend development server..."
	@cd $(GUI_DIR) && npm run dev

## frontend/build: build frontend for production
.PHONY: frontend/build
frontend/build:
	@echo "Building frontend for production..."
	@echo "Checking if node_modules exists..."
	@if [ ! -d "$(GUI_DIR)/node_modules" ]; then \
		echo "node_modules not found, installing dependencies..."; \
		cd $(GUI_DIR) && npm install; \
	fi
	@echo "Verifying Vite installation..."
	@if [ ! -f "$(GUI_DIR)/node_modules/vite/dist/node/cli.js" ]; then \
		echo "Vite installation corrupted, reinstalling..."; \
		cd $(GUI_DIR) && rm -rf node_modules package-lock.json && npm install; \
	fi
	@cd $(GUI_DIR) && npm run build
	@echo "Frontend production build complete."

## frontend/test: run frontend tests
.PHONY: frontend/test
frontend/test:
	@echo "Running frontend tests..."
	@cd $(GUI_DIR) && npm run test
	@echo "Frontend tests completed."

## tidy: tidy modfiles and format .go files
.PHONY: tidy
tidy:
	go mod tidy -v
	go fmt ./...

## cache: clear mod cache and test cache for .go files
.PHONY: cache
cache:
	go clean -cache -modcache -testcache

## build: build the application
.PHONY: build
build:
	@echo "Building $(APP_NAME) for host OS..."
	@mkdir -p ${OUTPUT_DIR} # Ensure the output directory exists
	go build -o=${OUTPUT_DIR}/${BINARY_NAME} ${MAIN_PACKAGE_PATH}

## build/all: build the application for all target platforms
.PHONY: build/all
build/all:
	@echo "Building for all target platforms using cross-compile.sh script..."
	@echo "Make sees VERSION as: [$(VERSION)]" # This VERSION is the release version
	@echo "Make sees BINARY_NAME as: [$(BINARY_NAME)]" # This BINARY_NAME is for cross-compile.sh
	@VERSION=$(VERSION) BINARY_NAME=$(BINARY_NAME) bash ./scripts/cross-compile.sh
	@echo "All builds completed. See ./dist/$(VERSION)/ for output."

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
run: ./build_and_run.sh
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
	--build.cmd "make build" --build.bin "/tmp/bin/${BINARY_NAME}" --build.delay "100" \
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

## dev-start: start KNIRV-NEXUS in development mode (single command)
.PHONY: dev-start
dev-start: build
	@echo "Starting KNIRV-NEXUS in development mode..."
	@echo "Main application will be available at: http://localhost:8080"
	@echo "Plugin server will be available at: http://localhost:8082"
	@echo ""
	@echo "Building plugin server if needed..."
	@cd plugin-server && go build -o plugin-server . 2>/dev/null || true
	@echo "Starting plugin server in background..."
	@cd plugin-server && ./plugin-server --port 8082 --agents ./agents --cors > /tmp/plugin-server.log 2>&1 &
	@sleep 2
	@echo "Plugin server started (logs: /tmp/plugin-server.log)"
	@echo "Starting main application..."
	@echo "Press Ctrl+C to stop both services"
	@$(OUTPUT_DIR)/$(BINARY_NAME)

## dev-stop: stop development services
.PHONY: dev-stop
dev-stop:
	@echo "Stopping development services..."
	@pkill -f "plugin-server" || echo "Plugin server not running"
	@pkill -f "$(BINARY_NAME)" || echo "Main application not running"
	@echo "Development services stopped"

## dev-setup-local: setup for local development without system installation
.PHONY: dev-setup-local
dev-setup-local:
	@echo "Setting up local development environment..."

	# Check if Go is available
	@which go >/dev/null || (echo "Go not found. Install with: sudo apt install golang-1.21" && exit 1)

	# Check if Node.js is available
	@which node >/dev/null || (echo "Node.js not found. Install with: sudo apt install nodejs npm" && exit 1)

	# Install frontend dependencies
	@echo "Installing frontend dependencies..."
	@cd $(GUI_DIR) && npm install

	# Build frontend
	@echo "Building frontend..."
	@cd $(GUI_DIR) && npm run build

	# Build backend
	@echo "Building backend..."
	@make build

	# Build plugin server
	@echo "Building plugin server..."
	@cd plugin-server && go build -o plugin-server .

	# Create agents directory
	@mkdir -p plugin-server/agents

	@echo "✓ Local development setup complete!"
	@echo ""
	@echo "To start KNIRV-NEXUS:"
	@echo "  make dev-start"
	@echo ""
	@echo "To stop:"
	@echo "  make dev-stop"

# ==================================================================================== #
# DESKTOP APPLICATION
# ==================================================================================== #

## desktop/setup: setup Electron dependencies for desktop application
.PHONY: desktop/setup
desktop/setup:
	@echo "Setting up Electron dependencies..."
	@cd electron && npm install
	@echo "Electron setup complete."

## desktop/dev: start desktop application in development mode
.PHONY: desktop/dev
desktop/dev:
	@echo "Starting desktop application in development mode..."
	@./scripts/build-desktop.sh dev

## desktop/build: build desktop application for current platform
.PHONY: desktop/build
desktop/build:
	@echo "Building desktop application for current platform..."
	@./scripts/build-desktop.sh build

## desktop/build/linux: build desktop application for Linux
.PHONY: desktop/build/linux
desktop/build/linux:
	@echo "Building desktop application for Linux..."
	@./scripts/build-desktop.sh build linux

## desktop/build/windows: build desktop application for Windows
.PHONY: desktop/bui>ld/windows
desktop/build/windows:
	@echo "Building desktop application for Windows..."
	@./scripts/build-desktop.sh build windows

## desktop/build/macos: build desktop application for macOS
.PHONY: desktop/build/macos
desktop/build/macos:
	@echo "Building desktop application for macOS..."
	@./scripts/build-desktop.sh build macos

## desktop/build/all: build desktop application for all platforms
.PHONY: desktop/build/all
desktop/build/all:
	@echo "Building desktop application for all platforms..."
	@./scripts/build-desktop.sh build all

## desktop/clean: clean desktop build artifacts
.PHONY: desktop/clean
desktop/clean:
	@echo "Cleaning desktop build artifacts..."
	@./scripts/build-desktop.sh clean

## desktop/release/patch: create a patch release (1.0.0 -> 1.0.1)
.PHONY: desktop/release/patch
desktop/release/patch:
	@echo "Creating patch release..."
	@./scripts/release.sh patch

## desktop/release/minor: create a minor release (1.0.0 -> 1.1.0)
.PHONY: desktop/release/minor
desktop/release/minor:
	@echo "Creating minor release..."
	@./scripts/release.sh minor

## desktop/release/major: create a major release (1.0.0 -> 2.0.0)
.PHONY: desktop/release/major
desktop/release/major:
	@echo "Creating major release..."
	@./scripts/release.sh major

## desktop/release/version: create a release with specific version
.PHONY: desktop/release/version
desktop/release/version:
	@echo "Creating release with specific version..."
	@echo "Usage: make desktop/release/version VERSION=1.2.3"
	@if [ -z "$(VERSION)" ]; then echo "ERROR: VERSION not specified"; exit 1; fi
	@./scripts/release.sh version $(VERSION)

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
	@echo "Public repo: /home/gperry/Documents/GitHub/cloud-equities/KNIRVCHAIN_PUBLIC"

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
	@echo "Public repo: /home/gperry/Documents/GitHub/cloud-equities/KNIRVCHAIN_PUBLIC"

# ==================================================================================== #
# KALI LINUX FOUNDATION TARGETS
# ==================================================================================== #

# Kali Linux specific directories
KALI_INSTALL_DIR := /opt/knirv-nexus
KALI_SERVICE_DIR := /etc/systemd/system
KALI_LOG_DIR := /var/log/knirv-nexus
KALI_CONFIG_DIR := /etc/knirv-nexus

## kali/check-deps: Check if running on Kali Linux and verify dependencies
.PHONY: kali/check-deps
kali/check-deps:
	@echo "Checking Kali Linux environment..."
	@if [ ! -f /etc/os-release ] || ! grep -q "ID=kali" /etc/os-release; then \
		echo "WARNING: This doesn't appear to be Kali Linux"; \
		echo "KNIRV-NEXUS is optimized for Kali Linux 2024.3 with kernel 6.6.x LTS"; \
	else \
		echo "✓ Kali Linux detected"; \
	fi
	@echo "Checking kernel version..."
	@uname -r | grep -q "^6\.6\." && echo "✓ Kernel 6.6.x LTS detected" || echo "⚠ Kernel 6.6.x LTS recommended"
	@echo "Checking TEE support..."
	@lsmod | grep -q intel_sgx && echo "✓ Intel SGX module loaded" || echo "⚠ Intel SGX module not loaded"
	@lsmod | grep -q ccp && echo "✓ AMD CCP module loaded" || echo "⚠ AMD CCP module not loaded"

## kali/install-deps: Install Kali Linux dependencies for KNIRV-NEXUS
.PHONY: kali/install-deps
kali/install-deps:
	@echo "Installing Kali Linux dependencies..."
	sudo apt update

	# Remove conflicting packages first
	@echo "Resolving container runtime conflicts..."
	sudo apt remove -y containerd.io containerd runc docker-ce docker-ce-cli 2>/dev/null || true
	sudo apt autoremove -y

	# Install core dependencies
	@echo "Installing core dependencies..."
	sudo apt install -y \
		golang-1.21 \
		nodejs \
		npm \
		build-essential \
		git \
		curl \
		wget \
		htop \
		iotop \
		strace \
		gdb \
		valgrind \
		tpm2-tools

	# Verify Node.js and npm versions
	@echo "Verifying Node.js and npm installation..."
	@node --version || echo "Node.js installation may have issues"
	@npm --version || echo "npm installation may have issues"

	# Update npm to latest version
	@echo "Updating npm to latest version..."
	sudo npm install -g npm@latest || echo "npm update failed"

	# Install container runtime (prefer podman for better security)
	@echo "Installing container runtime..."
	sudo apt install -y podman || echo "Podman installation failed, trying docker.io..."

	# Try docker.io if podman failed
	@if ! command -v podman >/dev/null 2>&1; then \
		echo "Installing docker.io as fallback..."; \
		sudo apt install -y docker.io || echo "Docker installation failed"; \
	fi

	# Verify Go installation and set up PATH
	@echo "Setting up Go environment..."
	@if [ -d "/usr/lib/go-1.21" ]; then \
		echo 'export PATH=/usr/lib/go-1.21/bin:$$PATH' | sudo tee -a /etc/environment; \
		echo "Go 1.21 path added to /etc/environment"; \
	fi

	@echo "✓ Dependencies installed"

## kali/install-docker: Install Docker CE from official repository (alternative to docker.io)
.PHONY: kali/install-docker
kali/install-docker:
	@echo "Installing Docker CE from official repository..."

	# Remove old versions
	sudo apt remove -y docker docker-engine docker.io containerd runc 2>/dev/null || true

	# Install prerequisites
	sudo apt update
	sudo apt install -y \
		ca-certificates \
		curl \
		gnupg \
		lsb-release

	# Add Docker's official GPG key
	sudo mkdir -p /etc/apt/keyrings
	curl -fsSL https://download.docker.com/linux/debian/gpg | sudo gpg --dearmor -o /etc/apt/keyrings/docker.gpg

	# Set up the repository (using Debian repos for Kali)
	echo "deb [arch=$$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/debian bullseye stable" | \
		sudo tee /etc/apt/sources.list.d/docker.list > /dev/null

	# Install Docker Engine
	sudo apt update
	sudo apt install -y docker-ce docker-ce-cli containerd.io docker-compose-plugin

	# Add current user to docker group
	sudo usermod -aG docker $$USER || true

	@echo "✓ Docker CE installed"
	@echo "Note: You may need to log out and back in for group changes to take effect"

## kali/fix-container-deps: Fix container runtime dependency issues
.PHONY: kali/fix-container-deps
kali/fix-container-deps:
	@echo "Fixing container runtime dependency conflicts..."

	# Clean up conflicting packages
	sudo apt remove -y containerd.io containerd runc docker-ce docker-ce-cli docker.io 2>/dev/null || true
	sudo apt autoremove -y
	sudo apt autoclean

	# Clear package cache
	sudo apt clean

	# Update package lists
	sudo apt update

	# Install podman as the preferred container runtime
	sudo apt install -y podman

	@echo "✓ Container runtime conflicts resolved"

## kali/fix-frontend: Fix frontend dependency issues
.PHONY: kali/fix-frontend
kali/fix-frontend:
	@echo "Fixing frontend dependency issues..."

	# Check Node.js version
	@echo "Current Node.js version:"
	@node --version || echo "Node.js not found"

	# Check npm version
	@echo "Current npm version:"
	@npm --version || echo "npm not found"

	# Clean and reinstall frontend dependencies
	@echo "Cleaning frontend dependencies..."
	@cd $(GUI_DIR) && rm -rf node_modules package-lock.json .vite dist 2>/dev/null || true

	# Clear npm cache
	@echo "Clearing npm cache..."
	@npm cache clean --force

	# Reinstall dependencies
	@echo "Reinstalling frontend dependencies..."
	@cd $(GUI_DIR) && npm install

	# Verify Vite installation
	@echo "Verifying Vite installation..."
	@if [ -f "$(GUI_DIR)/node_modules/vite/dist/node/cli.js" ]; then \
		echo "✓ Vite installation verified"; \
	else \
		echo "✗ Vite installation failed"; \
		exit 1; \
	fi

	@echo "✓ Frontend dependencies fixed"

## kali/setup-tee: Setup TEE (Trusted Execution Environment) support
.PHONY: kali/setup-tee
kali/setup-tee:
	@echo "Setting up TEE support..."
	# Enable Intel SGX
	sudo modprobe intel_sgx || echo "Intel SGX module not available"
	echo 'intel_sgx' | sudo tee -a /etc/modules || true
	# Configure AMD SEV-SNP
	sudo modprobe ccp || echo "AMD CCP module not available"
	sudo modprobe sev-guest || echo "AMD SEV-guest module not available"
	# Setup TPM for attestation
	sudo systemctl enable tpm2-abrmd || echo "TPM2 ABRMD not available"
	sudo systemctl start tpm2-abrmd || echo "TPM2 ABRMD failed to start"
	@echo "✓ TEE setup completed"

## kali/install: Install KNIRV-NEXUS on Kali Linux with proper permissions
.PHONY: kali/install
kali/install: build kali/check-deps
	@echo "Installing KNIRV-NEXUS on Kali Linux..."
	# Create directories
	sudo mkdir -p $(KALI_INSTALL_DIR)/bin
	sudo mkdir -p $(KALI_INSTALL_DIR)/gui
	sudo mkdir -p $(KALI_INSTALL_DIR)/plugin-server/agents
	sudo mkdir -p $(KALI_LOG_DIR)
	sudo mkdir -p $(KALI_CONFIG_DIR)

	# Install binaries
	sudo cp $(OUTPUT_DIR)/$(BINARY_NAME) $(KALI_INSTALL_DIR)/bin/knirv-nexus
	sudo cp plugin-server/plugin-server $(KALI_INSTALL_DIR)/bin/knirv-plugin-server || echo "Plugin server binary not found"

	# Install GUI
	sudo cp -r gui/dist/* $(KALI_INSTALL_DIR)/gui/ || echo "GUI dist not found, run 'make frontend/build' first"

	# Create user if it doesn't exist
	sudo useradd -r -s /bin/false -d $(KALI_INSTALL_DIR) knirv-nexus 2>/dev/null || true

	# Set permissions
	sudo chown -R root:root $(KALI_INSTALL_DIR)
	sudo chmod +x $(KALI_INSTALL_DIR)/bin/*
	sudo chown -R knirv-nexus:knirv-nexus $(KALI_LOG_DIR) 2>/dev/null || true
	sudo chown -R knirv-nexus:knirv-nexus $(KALI_CONFIG_DIR) 2>/dev/null || true

	@echo "✓ KNIRV-NEXUS installed to $(KALI_INSTALL_DIR)"

## kali/install-services: Install systemd services for KNIRV-NEXUS
.PHONY: kali/install-services
kali/install-services: kali/install
	@echo "Installing systemd services..."
	# Create service files if they don't exist
	sudo mkdir -p systemd
	@echo "Creating knirv-nexus.service..."
	@echo '[Unit]' | sudo tee $(KALI_SERVICE_DIR)/knirv-nexus.service
	@echo 'Description=KNIRV-NEXUS TEE Agent Platform' | sudo tee -a $(KALI_SERVICE_DIR)/knirv-nexus.service
	@echo 'After=network.target' | sudo tee -a $(KALI_SERVICE_DIR)/knirv-nexus.service
	@echo '' | sudo tee -a $(KALI_SERVICE_DIR)/knirv-nexus.service
	@echo '[Service]' | sudo tee -a $(KALI_SERVICE_DIR)/knirv-nexus.service
	@echo 'Type=simple' | sudo tee -a $(KALI_SERVICE_DIR)/knirv-nexus.service
	@echo 'User=knirv-nexus' | sudo tee -a $(KALI_SERVICE_DIR)/knirv-nexus.service
	@echo 'WorkingDirectory=$(KALI_INSTALL_DIR)' | sudo tee -a $(KALI_SERVICE_DIR)/knirv-nexus.service
	@echo 'ExecStart=$(KALI_INSTALL_DIR)/bin/knirv-nexus' | sudo tee -a $(KALI_SERVICE_DIR)/knirv-nexus.service
	@echo 'Restart=always' | sudo tee -a $(KALI_SERVICE_DIR)/knirv-nexus.service
	@echo 'RestartSec=10' | sudo tee -a $(KALI_SERVICE_DIR)/knirv-nexus.service
	@echo '' | sudo tee -a $(KALI_SERVICE_DIR)/knirv-nexus.service
	@echo '[Install]' | sudo tee -a $(KALI_SERVICE_DIR)/knirv-nexus.service
	@echo 'WantedBy=multi-user.target' | sudo tee -a $(KALI_SERVICE_DIR)/knirv-nexus.service

	# Reload systemd and enable services
	sudo systemctl daemon-reload
	sudo systemctl enable knirv-nexus.service

	@echo "✓ Services installed and enabled"

## kali/start: Start KNIRV-NEXUS services
.PHONY: kali/start
kali/start:
	@echo "Starting KNIRV-NEXUS services..."
	sudo systemctl start knirv-nexus.service
	@echo "✓ Services started"

## kali/stop: Stop KNIRV-NEXUS services
.PHONY: kali/stop
kali/stop:
	@echo "Stopping KNIRV-NEXUS services..."
	sudo systemctl stop knirv-nexus.service
	@echo "✓ Services stopped"

## kali/status: Show KNIRV-NEXUS service status
.PHONY: kali/status
kali/status:
	@echo "KNIRV-NEXUS Service Status:"
	@echo "=========================="
	@systemctl is-active knirv-nexus.service 2>/dev/null && echo "Main Service: ACTIVE" || echo "Main Service: INACTIVE"
	@echo ""
	@echo "Recent logs:"
	@sudo journalctl -u knirv-nexus.service --no-pager -n 10 || echo "No logs available"

## kali/logs: Show KNIRV-NEXUS logs
.PHONY: kali/logs
kali/logs:
	@echo "KNIRV-NEXUS Logs:"
	@echo "================"
	sudo journalctl -u knirv-nexus.service -f

## kali/uninstall: Uninstall KNIRV-NEXUS from Kali Linux
.PHONY: kali/uninstall
kali/uninstall:
	@echo "Uninstalling KNIRV-NEXUS..."
	# Stop and disable services
	sudo systemctl stop knirv-nexus.service 2>/dev/null || true
	sudo systemctl disable knirv-nexus.service 2>/dev/null || true

	# Remove service files
	sudo rm -f $(KALI_SERVICE_DIR)/knirv-nexus.service

	# Remove installation directory
	sudo rm -rf $(KALI_INSTALL_DIR)

	# Remove user (optional, commented for safety)
	# sudo userdel knirv-nexus 2>/dev/null || true

	sudo systemctl daemon-reload
	@echo "✓ KNIRV-NEXUS uninstalled"

## kali/security-setup: Setup security hardening for Kali Linux
.PHONY: kali/security-setup
kali/security-setup:
	@echo "Setting up security hardening..."
	# Configure UFW firewall
	sudo ufw --force reset
	sudo ufw default deny incoming
	sudo ufw default allow outgoing
	sudo ufw allow 8080/tcp comment 'KNIRV-NEXUS Main'
	sudo ufw allow 8082/tcp comment 'KNIRV-NEXUS Plugin Server'
	sudo ufw --force enable

	# Setup audit rules
	echo '-w $(KALI_INSTALL_DIR) -p rwxa -k knirv-access' | sudo tee -a /etc/audit/rules.d/knirv-nexus.rules
	echo '-w $(KALI_LOG_DIR) -p rwxa -k knirv-logs' | sudo tee -a /etc/audit/rules.d/knirv-nexus.rules
	sudo systemctl restart auditd || echo "Auditd not available"

	@echo "✓ Security hardening completed"

## kali/full-setup: Complete Kali Linux setup for KNIRV-NEXUS
.PHONY: kali/full-setup
kali/full-setup: kali/check-deps kali/install-deps kali/setup-tee frontend/build kali/install kali/install-services kali/security-setup
	@echo "=========================================="
	@echo "KNIRV-NEXUS Kali Linux Setup Complete!"
	@echo "=========================================="
	@echo "Installation directory: $(KALI_INSTALL_DIR)"
	@echo "Log directory: $(KALI_LOG_DIR)"
	@echo "Config directory: $(KALI_CONFIG_DIR)"
	@echo ""
	@echo "To start KNIRV-NEXUS:"
	@echo "  make kali/start"
	@echo ""
	@echo "To check status:"
	@echo "  make kali/status"
	@echo ""
	@echo "To view logs:"
	@echo "  make kali/logs"
	@echo ""
	@echo "Access the web interface at: http://localhost:8080"

## kali/help: Show Kali Linux specific help and troubleshooting
.PHONY: kali/help
kali/help:
	@echo "KNIRV-NEXUS Kali Linux Commands:"
	@echo "================================"
	@echo ""
	@echo "Setup Commands:"
	@echo "  kali/check-deps      - Check Kali Linux environment and dependencies"
	@echo "  kali/install-deps    - Install required dependencies (uses podman by default)"
	@echo "  kali/install-docker  - Install Docker CE from official repository"
	@echo "  kali/fix-container-deps - Fix container runtime conflicts"
	@echo "  kali/setup-tee       - Setup TEE (Trusted Execution Environment) support"
	@echo "  kali/security-setup  - Configure security hardening"
	@echo "  kali/full-setup      - Complete automated setup"
	@echo ""
	@echo "Installation Commands:"
	@echo "  kali/install         - Install KNIRV-NEXUS binaries and files"
	@echo "  kali/install-services - Install and enable systemd services"
	@echo ""
	@echo "Service Management:"
	@echo "  kali/start           - Start KNIRV-NEXUS services"
	@echo "  kali/stop            - Stop KNIRV-NEXUS services"
	@echo "  kali/status          - Show service status and recent logs"
	@echo "  kali/logs            - Follow live logs"
	@echo ""
	@echo "Maintenance:"
	@echo "  kali/uninstall       - Uninstall KNIRV-NEXUS"
	@echo "  kali/help            - Show this help message"
	@echo ""
	@echo "Troubleshooting:"
	@echo "  If you encounter container runtime conflicts:"
	@echo "    make kali/fix-container-deps"
	@echo ""
	@echo "  If you encounter frontend build issues:"
	@echo "    make kali/fix-frontend"
	@echo ""
	@echo "  If you need Docker instead of Podman:"
	@echo "    make kali/install-docker"
	@echo ""
	@echo "  For TEE support issues, ensure your hardware supports:"
	@echo "    - Intel SGX, AMD SEV-SNP, or Intel TDX"
	@echo "    - BIOS/UEFI settings are properly configured"
