.PHONY: build clean test docker docker-compose

# Build the KNIRVBASE node binary
build:
	@echo "Building KNIRVBASE node..."
	@mkdir -p bin
	go build -o bin/knirvbase ./cmd/node
	@echo "Binary built: bin/knirvbase"

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -rf bin/
	go clean ./...

# Run tests
test:
	@echo "Running tests..."
	go test ./...

# Build Docker image
docker:
	@echo "Building Docker image..."
	docker build -t knirvbase/node:latest .

# Run with Docker Compose
docker-compose:
	@echo "Starting with Docker Compose..."
	docker-compose -f deployment/docker-compose.yml up -d

# Stop Docker Compose
docker-compose-stop:
	@echo "Stopping Docker Compose..."
	docker-compose -f deployment/docker-compose.yml down

# Run the node locally
run: build
	@echo "Starting KNIRVBASE node..."
	./bin/knirvbase

# Install dependencies
deps:
	@echo "Installing dependencies..."
	go mod download
	go mod tidy

# Format code
fmt:
	@echo "Formatting code..."
	go fmt ./...

# Lint code
lint:
	@echo "Linting code..."
	golangci-lint run

# Generate mocks (if needed)
mocks:
	@echo "Generating mocks..."
	mockgen -source=internal/blockchain/chain.go -destination=mocks/chain_mock.go -package=mocks

# Help
help:
	@echo "Available targets:"
	@echo "  build           - Build the KNIRVBASE node binary to bin/"
	@echo "  clean           - Clean build artifacts"
	@echo "  test            - Run tests"
	@echo "  docker          - Build Docker image"
	@echo "  docker-compose  - Start with Docker Compose"
	@echo "  docker-compose-stop - Stop Docker Compose"
	@echo "  run             - Build and run the node locally"
	@echo "  deps            - Install dependencies"
	@echo "  fmt             - Format code"
	@echo "  lint            - Lint code"
	@echo "  mocks           - Generate mocks"
	@echo "  help            - Show this help"