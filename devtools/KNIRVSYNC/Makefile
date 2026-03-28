.PHONY: all build sync-tool doc-sync scan sync-readme sync-api sync-unified sync-all install uninstall clean help test

# Build configuration
SYNC_BINARY_NAME=sync-tool
DOCSYNC_BINARY_NAME=doc-sync
BIN_DIR=bin
SYNC_CMD_DIR=cmd/sync
DOCSYNC_CMD_DIR=cmd/doc-sync
CONFIG_DIR=config
REPORTS_DIR=reports
PREFIX?=/usr/local

all: build

# Build the sync-tool binary
sync-tool:
	@echo "Building $(SYNC_BINARY_NAME)..."
	@mkdir -p $(BIN_DIR)
	@go build -o $(BIN_DIR)/$(SYNC_BINARY_NAME) ./$(SYNC_CMD_DIR)/main.go
	@echo "Build complete: $(BIN_DIR)/$(SYNC_BINARY_NAME)"

# Build the doc-sync binary
doc-sync:
	@echo "Building $(DOCSYNC_BINARY_NAME)..."
	@mkdir -p $(BIN_DIR)
	@go build -o $(BIN_DIR)/$(DOCSYNC_BINARY_NAME) ./$(DOCSYNC_CMD_DIR)/main.go
	@echo "Build complete: $(BIN_DIR)/$(DOCSYNC_BINARY_NAME)"

# Build both binaries
build: sync-tool doc-sync

# Scan for documentation gaps
scan: doc-sync
	@echo "Scanning for documentation gaps..."
	@mkdir -p $(REPORTS_DIR)
	@./$(BIN_DIR)/$(DOCSYNC_BINARY_NAME) -scan -config $(CONFIG_DIR)/doc-sync-config.json
	@echo "Gap report: $(REPORTS_DIR)/doc-gaps-report.md"

# Sync README files
sync-readme: doc-sync
	@echo "Syncing README files..."
	@./$(BIN_DIR)/$(DOCSYNC_BINARY_NAME) -readme -config $(CONFIG_DIR)/doc-sync-config.json

# Sync API specifications
sync-api: doc-sync
	@echo "Syncing API specifications..."
	@./$(BIN_DIR)/$(DOCSYNC_BINARY_NAME) -api -config $(CONFIG_DIR)/doc-sync-config.json

# Generate unified OpenAPI spec
sync-unified: doc-sync
	@echo "Generating unified OpenAPI specification..."
	@./$(BIN_DIR)/$(DOCSYNC_BINARY_NAME) -unified -config $(CONFIG_DIR)/doc-sync-config.json

# Sync all documentation
sync-all: doc-sync
	@echo "Performing complete documentation synchronization..."
	@mkdir -p $(REPORTS_DIR)
	@./$(BIN_DIR)/$(DOCSYNC_BINARY_NAME) -config $(CONFIG_DIR)/doc-sync-config.json
	@echo "Sync report: $(REPORTS_DIR)/doc-sync-report.json"

# Run via shell script (alternative method)
script-sync:
	@./scripts/doc-sync.sh all

# Install CLI tools globally
install: build
	@echo "Installing CLI tools to $(PREFIX)/bin..."
	@mkdir -p $(PREFIX)/bin
	@cp $(BIN_DIR)/$(SYNC_BINARY_NAME) $(PREFIX)/bin/
	@cp $(BIN_DIR)/$(DOCSYNC_BINARY_NAME) $(PREFIX)/bin/
	@echo "Installation complete:"
	@echo "  $(SYNC_BINARY_NAME) installed to $(PREFIX)/bin/$(SYNC_BINARY_NAME)"
	@echo "  $(DOCSYNC_BINARY_NAME) installed to $(PREFIX)/bin/$(DOCSYNC_BINARY_NAME)"

# Uninstall CLI tools
uninstall:
	@echo "Removing CLI tools from $(PREFIX)/bin..."
	@rm -f $(PREFIX)/bin/$(SYNC_BINARY_NAME)
	@rm -f $(PREFIX)/bin/$(DOCSYNC_BINARY_NAME)
	@echo "Uninstallation complete"

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf $(BIN_DIR)
	@rm -rf $(REPORTS_DIR)
	@echo "Clean complete"

# Run tests
test:
	@echo "Running tests..."
	@go test -v ./internal/...

# Display help
help:
	@echo "KNIRVSYNC - Makefile"
	@echo ""
	@echo "Available targets:"
	@echo "  build           Build both sync-tool and doc-sync binaries"
	@echo "  sync-tool       Build the main synchronization tool"
	@echo "  doc-sync        Build the documentation synchronization tool"
	@echo "  scan            Scan for documentation gaps"
	@echo "  sync-readme     Sync README files to central documentation"
	@echo "  sync-api        Sync API specifications"
	@echo "  sync-unified    Generate unified OpenAPI specification"
	@echo "  sync-all        Perform complete documentation sync (default)"
	@echo "  script-sync     Run via shell script"
	@echo "  install         Install CLI tools globally to system"
	@echo "  uninstall       Remove CLI tools from system"
	@echo "  test            Run tests"
	@echo "  clean           Remove build artifacts"
	@echo "  help            Display this help message"
	@echo ""
	@echo "Examples:"
	@echo "  make build"
	@echo "  make scan"
	@echo "  make sync-all"
	@echo "  make install PREFIX=/usr/local"
	@echo ""
	@echo "Environment Variables:"
	@echo "  PREFIX          Installation prefix (default: /usr/local)"