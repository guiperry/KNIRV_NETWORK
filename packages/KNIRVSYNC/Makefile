.PHONY: all build scan sync-readme sync-api sync-unified sync-all clean help test

# Build configuration
BINARY_NAME=doc-sync
BIN_DIR=bin
CMD_DIR=cmd/doc-sync
CONFIG_DIR=config
REPORTS_DIR=reports

all: build

# Build the doc-sync binary
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BIN_DIR)
	@go build -o $(BIN_DIR)/$(BINARY_NAME) ./$(CMD_DIR)/main.go
	@echo "Build complete: $(BIN_DIR)/$(BINARY_NAME)"

# Scan for documentation gaps
scan: build
	@echo "Scanning for documentation gaps..."
	@mkdir -p $(REPORTS_DIR)
	@./$(BIN_DIR)/$(BINARY_NAME) -scan -config $(CONFIG_DIR)/doc-sync-config.json
	@echo "Gap report: $(REPORTS_DIR)/doc-gaps-report.md"

# Sync README files
sync-readme: build
	@echo "Syncing README files..."
	@./$(BIN_DIR)/$(BINARY_NAME) -readme -config $(CONFIG_DIR)/doc-sync-config.json

# Sync API specifications
sync-api: build
	@echo "Syncing API specifications..."
	@./$(BIN_DIR)/$(BINARY_NAME) -api -config $(CONFIG_DIR)/doc-sync-config.json

# Generate unified OpenAPI spec
sync-unified: build
	@echo "Generating unified OpenAPI specification..."
	@./$(BIN_DIR)/$(BINARY_NAME) -unified -config $(CONFIG_DIR)/doc-sync-config.json

# Sync all documentation
sync-all: build
	@echo "Performing complete documentation synchronization..."
	@mkdir -p $(REPORTS_DIR)
	@./$(BIN_DIR)/$(BINARY_NAME) -config $(CONFIG_DIR)/doc-sync-config.json
	@echo "Sync report: $(REPORTS_DIR)/doc-sync-report.json"

# Run via shell script (alternative method)
script-sync:
	@./scripts/doc-sync.sh all

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
	@echo "KNIRVSYNC Documentation Sync - Makefile"
	@echo ""
	@echo "Available targets:"
	@echo "  build           Build the doc-sync binary"
	@echo "  scan            Scan for documentation gaps"
	@echo "  sync-readme     Sync README files to central documentation"
	@echo "  sync-api        Sync API specifications"
	@echo "  sync-unified    Generate unified OpenAPI specification"
	@echo "  sync-all        Perform complete documentation sync (default)"
	@echo "  script-sync     Run via shell script"
	@echo "  test            Run tests"
	@echo "  clean           Remove build artifacts"
	@echo "  help            Display this help message"
	@echo ""
	@echo "Examples:"
	@echo "  make build"
	@echo "  make scan"
	@echo "  make sync-all"
