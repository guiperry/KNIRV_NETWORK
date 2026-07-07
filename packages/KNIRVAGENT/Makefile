.PHONY: all build build-wasm build-wasi-wasm build-browser-wasm install uninstall clean help test

# Build variables
BINARY_NAME=knirvagent
BUILD_DIR=build
CMD_DIR=cmd/$(BINARY_NAME)
MAIN_GO=$(CMD_DIR)/main.go

# Version
VERSION?=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
GIT_COMMIT=$(shell git rev-parse --short=8 HEAD 2>/dev/null || echo "dev")
BUILD_TIME=$(shell date +%FT%T%z)
GO_VERSION=$(shell $(GO) version | awk '{print $$3}')
LDFLAGS=-ldflags "-X main.version=$(VERSION) -X main.gitCommit=$(GIT_COMMIT) -X main.buildTime=$(BUILD_TIME) -X main.goVersion=$(GO_VERSION)"

# Go variables
GO?=go
GOFLAGS?=-v

# Installation
INSTALL_PREFIX?=$(HOME)/.local
INSTALL_BIN_DIR=$(INSTALL_PREFIX)/bin
INSTALL_MAN_DIR=$(INSTALL_PREFIX)/share/man/man1

# Workspace and Skills
KNIRVAGENT_HOME?=$(HOME)/.knirvagent
WORKSPACE_DIR?=$(KNIRVAGENT_HOME)/workspace
WORKSPACE_SKILLS_DIR=$(WORKSPACE_DIR)/skills
BUILTIN_SKILLS_DIR=$(CURDIR)/skills

# OS detection
UNAME_S:=$(shell uname -s)
UNAME_M:=$(shell uname -m)

# Platform-specific settings
ifeq ($(UNAME_S),Linux)
	PLATFORM=linux
	ifeq ($(UNAME_M),x86_64)
		ARCH=amd64
	else ifeq ($(UNAME_M),aarch64)
		ARCH=arm64
	else ifeq ($(UNAME_M),riscv64)
		ARCH=riscv64
	else
		ARCH=$(UNAME_M)
	endif
else ifeq ($(UNAME_S),Darwin)
	PLATFORM=darwin
	ifeq ($(UNAME_M),x86_64)
		ARCH=amd64
	else ifeq ($(UNAME_M),arm64)
		ARCH=arm64
	else
		ARCH=$(UNAME_M)
	endif
else
	PLATFORM=$(UNAME_S)
	ARCH=$(UNAME_M)
endif

BINARY_PATH=$(BUILD_DIR)/$(BINARY_NAME)-$(PLATFORM)-$(ARCH)
WASI_WASM_NAME=$(BINARY_NAME)-wasi.wasm
BROWSER_WASM_NAME=$(BINARY_NAME)-js.wasm
WASI_WASM_PATH=$(BUILD_DIR)/$(WASI_WASM_NAME)
BROWSER_WASM_PATH=$(BUILD_DIR)/$(BROWSER_WASM_NAME)
CLI_BIN_DIR?=$(CURDIR)/../../../KNIRV_CORP/packages/cli/bin
CLI_WASM_PATH=$(CLI_BIN_DIR)/$(BINARY_NAME).wasm
BRIDGE_BIN_DIR?=$(CURDIR)/../KNIRVBRIDGE/bin
BRIDGE_WASM_PATH=$(BRIDGE_BIN_DIR)/$(BINARY_NAME).wasm

# Default target
all: build

## generate: Run generate
generate:
	@echo "Run generate..."
	@rm -r ./$(CMD_DIR)/workspace 2>/dev/null || true
	@$(GO) generate ./...
	@echo "Run generate complete"

## build: Build the knirvagent binary for current platform
build: generate
	@echo "Building $(BINARY_NAME) for $(PLATFORM)/$(ARCH)..."
	@mkdir -p $(BUILD_DIR)
	@$(GO) build $(GOFLAGS) $(LDFLAGS) -o $(BINARY_PATH) ./$(CMD_DIR)
	@echo "Build complete: $(BINARY_PATH)"
	@ln -sf $(BINARY_NAME)-$(PLATFORM)-$(ARCH) $(BUILD_DIR)/$(BINARY_NAME)
	@$(MAKE) build-wasm

## build-wasm: Build both WASI and browser WebAssembly artifacts
build-wasm: build-wasi-wasm build-browser-wasm

## build-wasi-wasm: Build knirvagent as WASI WebAssembly and copy it into the KNIRV CLI package
build-wasi-wasm:
	@echo "Building $(BINARY_NAME) for wasip1/wasm..."
	@mkdir -p $(BUILD_DIR) $(CLI_BIN_DIR)
	@GOOS=wasip1 GOARCH=wasm $(GO) build $(GOFLAGS) $(LDFLAGS) -o $(WASI_WASM_PATH) ./$(CMD_DIR)
	@cp $(WASI_WASM_PATH) $(CLI_WASM_PATH)
	@echo "WASI WASM build complete: $(WASI_WASM_PATH)"
	@echo "Copied WASI WASM to: $(CLI_WASM_PATH)"

## build-browser-wasm: Build knirvagent as browser WebAssembly and copy it into KNIRVBRIDGE
build-browser-wasm:
	@echo "Building $(BINARY_NAME) for js/wasm..."
	@mkdir -p $(BUILD_DIR) $(BRIDGE_BIN_DIR)
	@GOOS=js GOARCH=wasm $(GO) build $(GOFLAGS) $(LDFLAGS) -o $(BROWSER_WASM_PATH) ./$(CMD_DIR)
	@cp $(BROWSER_WASM_PATH) $(BRIDGE_WASM_PATH)
	@echo "Browser WASM build complete: $(BROWSER_WASM_PATH)"
	@echo "Copied browser WASM to: $(BRIDGE_WASM_PATH)"

## build-all: Build knirvagent for all platforms
build-all: generate
	@echo "Building for multiple platforms..."
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 $(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 ./$(CMD_DIR)
	GOOS=linux GOARCH=arm64 $(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 ./$(CMD_DIR)
	GOOS=linux GOARCH=riscv64 $(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-riscv64 ./$(CMD_DIR)
	GOOS=darwin GOARCH=arm64 $(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 ./$(CMD_DIR)
	GOOS=windows GOARCH=amd64 $(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe ./$(CMD_DIR)
	@echo "All builds complete"

## install: Install knirvagent to system and copy builtin skills
install: build
	@echo "Installing $(BINARY_NAME)..."
	@mkdir -p $(INSTALL_BIN_DIR)
	@cp $(BUILD_DIR)/$(BINARY_NAME) $(INSTALL_BIN_DIR)/$(BINARY_NAME)
	@chmod +x $(INSTALL_BIN_DIR)/$(BINARY_NAME)
	@echo "Installed binary to $(INSTALL_BIN_DIR)/$(BINARY_NAME)"
	@echo "Installation complete!"

## uninstall: Remove knirvagent from system
uninstall:
	@echo "Uninstalling $(BINARY_NAME)..."
	@rm -f $(INSTALL_BIN_DIR)/$(BINARY_NAME)
	@echo "Removed binary from $(INSTALL_BIN_DIR)/$(BINARY_NAME)"
	@echo "Note: Only the executable file has been deleted."
	@echo "If you need to delete all configurations (config.json, workspace, etc.), run 'make uninstall-all'"

## uninstall-all: Remove knirvagent and all data
uninstall-all:
	@echo "Removing workspace and skills..."
	@rm -rf $(KNIRVAGENT_HOME)
	@echo "Removed workspace: $(KNIRVAGENT_HOME)"
	@echo "Complete uninstallation done!"

## clean: Remove build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf $(BUILD_DIR)
	@echo "Clean complete"

## fmt: Format Go code
vet:
	@$(GO) vet ./...

## fmt: Format Go code
test:
	@$(GO) test ./...

## fmt: Format Go code
fmt:
	@$(GO) fmt ./...

## deps: Update dependencies
deps:
	@$(GO) get -u ./...
	@$(GO) mod tidy

## run: Build and run knirvagent
run: build
	@$(BUILD_DIR)/$(BINARY_NAME) $(ARGS)

## help: Show this help message
help:
	@echo "knirvagent Makefile"
	@echo ""
	@echo "Usage:"
	@echo "  make [target]"
	@echo ""
	@echo "Targets:"
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/## /  /'
	@echo ""
	@echo "Examples:"
	@echo "  make build              # Build for current platform"
	@echo "  make build-wasm         # Build WASI/browser WebAssembly and copy artifacts"
	@echo "  make build-wasi-wasm    # Build WASI WebAssembly and copy it to KNIRV CLI"
	@echo "  make build-browser-wasm # Build browser WebAssembly and copy it to KNIRVBRIDGE"
	@echo "  make install            # Install to ~/.local/bin"
	@echo "  make uninstall          # Remove from /usr/local/bin"
	@echo "  make install-skills     # Install skills to workspace"
	@echo ""
	@echo "Environment Variables:"
	@echo "  INSTALL_PREFIX          # Installation prefix (default: ~/.local)"
	@echo "  WORKSPACE_DIR           # Workspace directory (default: ~/.knirvagent/workspace)"
	@echo "  VERSION                 # Version string (default: git describe)"
	@echo ""
	@echo "Current Configuration:"
	@echo "  Platform: $(PLATFORM)/$(ARCH)"
	@echo "  Binary: $(BINARY_PATH)"
	@echo "  Install Prefix: $(INSTALL_PREFIX)"
	@echo "  Workspace: $(WORKSPACE_DIR)"
	@echo "  CLI WASI WASM Path: $(CLI_WASM_PATH)"
	@echo "  Bridge Browser WASM Path: $(BRIDGE_WASM_PATH)"
