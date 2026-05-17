# KNIRVGATEWAY Build System
# Builds the gateway binary with embedded WebGUI static export.
# The graphchain-explorer and knirvchain-portal standalone builds have been
# migrated into the WebGUI as native React pages and are no longer built.

PROJECT_ROOT := $(shell pwd)
EMBEDDED_DIR := $(PROJECT_ROOT)/internal/embedded
WEBGUI_DIR   := $(EMBEDDED_DIR)/webgui
BIN_DIR      := $(PROJECT_ROOT)/bin

.PHONY: all help webgui webgui-deps gateway clean

all: help

help: ## Show this help message
	@echo "KNIRVGATEWAY Build Targets:"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-20s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

# WebGUI (Next.js)
webgui-deps: ## Install WebGUI Node.js dependencies
	@echo "Installing WebGUI dependencies..."
	cd $(WEBGUI_DIR) && npm install --legacy-peer-deps

webgui-build: webgui-deps ## Build WebGUI Next.js static export into webgui/out/
	@echo "Building WebGUI static export..."
	cd $(WEBGUI_DIR) && npm run build
	@echo "WebGUI build complete: $(WEBGUI_DIR)/out/"

webgui: webgui-build ## Build WebGUI (alias for webgui-build)

# Gateway Go binary
gateway-deps: ## Install Go dependencies
	cd $(PROJECT_ROOT) && go mod download && go mod tidy

gateway-build: webgui-build gateway-deps ## Build gateway binary (requires WebGUI build first)
	@echo "Building KNIRVGATEWAY binary..."
	@mkdir -p $(BIN_DIR)
	cd $(PROJECT_ROOT) && go build -ldflags "-s -w" -o $(BIN_DIR)/knirvgateway ./cmd/gateway
	@echo "Binary built: $(BIN_DIR)/knirvgateway"

gateway: gateway-build ## Build gateway (alias for gateway-build)

clean: ## Clean all build artifacts
	rm -rf $(BIN_DIR)
	rm -rf $(WEBGUI_DIR)/.next $(WEBGUI_DIR)/out
	cd $(PROJECT_ROOT) && go clean
