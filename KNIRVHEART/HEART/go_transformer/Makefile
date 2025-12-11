
.PHONY: all build-gorgonite build-zookeeper copy-transformer clean


# Binary names
GORGONITE_BIN = gorgonite
ZOOKEEPER_BIN = zookeeper

# Directories
ROOT_DIR = .
ZOOKEEPER_DIR = zoo-keeper
BIN_DIR = $(ZOOKEEPER_DIR)/bin

# Build targets
all: build

# Build both binaries and copy gorgonite to zookeeper bin directory
build: build-gorgonite copy-gorgonite build-zookeeper 
	@echo "✅ All binaries built and deployed successfully"

# Build the gorgonite transformer training binary
build-gorgonite:
	@echo "🔨 Building gorgonite..."
	@cd $(ROOT_DIR) && go build -o $(GORGONITE_BIN) .
	@echo "✅ Gorgonite built successfully"

# Build the zookeeper GUI application
build-zookeeper:
	@echo "🔨 Building zookeeper..."
	@cd $(ZOOKEEPER_DIR) && go build -o $(ZOOKEEPER_BIN) ./cmd/zookeeper
	@echo "✅ Zookeeper built successfully"

# Copy gorgonite binary to zookeeper bin directory
copy-gorgonite:
	@echo "📁 Copying gorgonite to zookeeper bin directory..."
	@mkdir -p $(BIN_DIR)
	@cp $(GORGONITE_BIN) $(BIN_DIR)/
	@chmod +x $(BIN_DIR)/$(GORGONITE_BIN)
	@echo "✅ Gorgonite copied to $(BIN_DIR)/"

# Clean all built binaries
clean:
	@echo "🧹 Cleaning built binaries..."
	@rm -f $(ROOT_DIR)/$(GORGONITE_BIN)
	@rm -f $(ZOOKEEPER_DIR)/$(ZOOKEEPER_BIN)
	@rm -f $(BIN_DIR)/$(GORGONITE_BIN)
	@echo "✅ Clean completed"

# Run tests for both projects
test:
	@echo "🧪 Running tests..."
	@cd $(ROOT_DIR) && go test ./...
	@cd $(ZOOKEEPER_DIR) && go test ./...
	@echo "✅ Tests completed"

# Install binaries to system (optional)
install: build
	@echo "📦 Installing binaries..."
	@sudo cp $(BIN_DIR)/$(GORGONITE_BIN) /usr/local/bin/
	@sudo cp $(ZOOKEEPER_DIR)/$(ZOOKEEPER_BIN) /usr/local/bin/
	@echo "✅ Binaries installed to /usr/local/bin/"

# Development helpers
dev-setup:
	@echo "🔧 Setting up development environment..."
	@go mod download
	@cd $(ZOOKEEPER_DIR) && go mod download
	@echo "✅ Development environment ready"

# Quick rebuild (only rebuilds what's changed)
rebuild: clean build
	@echo "🔄 Quick rebuild completed"

# Show help
help:
	@echo "Gorgonite Transformer Training Suite"
	@echo ""
	@echo "Available targets:"
	@echo "  all          - Build everything (default)"
	@echo "  build        - Build both binaries and deploy gorgonite"
	@echo "  build-gorgonite - Build only gorgonite"
	@echo "  build-zookeeper - Build only zookeeper"
	@echo "  copy-gorgonite - Copy gorgonite to zookeeper bin directory"
	@echo "  clean        - Remove all built binaries"
	@echo "  test         - Run tests for both projects"
	@echo "  install      - Install binaries to system /usr/local/bin/"
	@echo "  dev-setup    - Download dependencies"
	@echo "  rebuild      - Clean and rebuild everything"
	@echo "  help         - Show this help message"
	@echo ""
	@echo "Usage examples:"
	@echo "  make                 # Build everything"
	@echo "  make build-gorgonite # Build only gorgonite"
	@echo "  make clean && make   # Clean rebuild"
	@echo "  make test            # Run all tests"

# Default target
.DEFAULT_GOAL := all