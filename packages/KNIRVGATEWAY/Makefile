# KNIRVGATEWAY Build System
# Builds the gateway binary with embedded WebGUI static export.

PROJECT_ROOT := $(shell pwd)
EMBEDDED_DIR := $(PROJECT_ROOT)/internal/embedded
WEBGUI_DIR   := $(EMBEDDED_DIR)/webgui
PORTAL_DIR   := $(EMBEDDED_DIR)/knirvchain-portal
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

webgui-build: webgui-deps portal-build ## Build WebGUI Next.js static export into webgui/out/ (also builds KNIRVChain Portal)
	@echo "Building WebGUI static export..."
	cd $(WEBGUI_DIR) && npm run build
	@echo "WebGUI build complete: $(WEBGUI_DIR)/out/"

webgui: webgui-build ## Build WebGUI (alias for webgui-build)

# KNIRVChain Portal (Vite React)
portal-deps: ## Install KNIRVChain Portal Node.js dependencies
	@echo "Installing KNIRVChain Portal dependencies..."
	cd $(PORTAL_DIR) && npm install

portal-build: portal-deps ## Build KNIRVChain Portal Vite static export into knirvchain-portal/dist/
	@echo "Building KNIRVChain Portal static export..."
	cd $(PORTAL_DIR) && npm run build
	@echo "Portal build complete: $(PORTAL_DIR)/dist/"

portal: portal-build ## Build Portal (alias for portal-build)

# Gateway Go binary
gateway-deps: ## Install Go dependencies
	cd $(PROJECT_ROOT) && go mod download && go mod tidy

gateway-build: webgui-build portal-build gateway-deps ## Build gateway binary (requires WebGUI + Portal build first)
	@echo "Building KNIRVGATEWAY binary..."
	@mkdir -p $(BIN_DIR)
	cd $(PROJECT_ROOT) && go build -ldflags "-s -w" -o $(BIN_DIR)/knirvgateway ./cmd/gateway
	@echo "Binary built: $(BIN_DIR)/knirvgateway"

gateway: gateway-build ## Build gateway (alias for gateway-build)

clean: ## Clean all build artifacts
	rm -rf $(BIN_DIR)
	rm -rf $(WEBGUI_DIR)/.next $(WEBGUI_DIR)/out
	rm -rf $(PORTAL_DIR)/dist $(PORTAL_DIR)/node_modules
	cd $(PROJECT_ROOT) && go clean
