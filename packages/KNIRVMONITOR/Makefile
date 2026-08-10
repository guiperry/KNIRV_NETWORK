# KNIRVMONITOR — builds the network monitor aggregation service into a
# single binary named "monitor". This is the binary KNIRVSERVER embeds
# (see packages/KNIRVSERVER/pkg/knirvmonitor) and launches as a subprocess.

BINARY_NAME := monitor
MAIN_PACKAGE_PATH := ./cmd/server
DIST_DIR := ./dist

VERSION ?= $(shell git describe --tags --abbrev=0 2>/dev/null || echo "dev-$(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)")
LDFLAGS := -s -w -X main.version=$(VERSION)

.PHONY: all
all: build

.PHONY: deps
deps: ## Download and tidy module dependencies
	go mod download
	go mod tidy

.PHONY: build
build: ## Build the monitor binary into ./dist/monitor
	@mkdir -p $(DIST_DIR)
	go build -ldflags "$(LDFLAGS)" -o $(DIST_DIR)/$(BINARY_NAME) $(MAIN_PACKAGE_PATH)

.PHONY: build-embed
build-embed: ## Build the monitor binary for embedding into KNIRVSERVER (path override via OUT)
	@test -n "$(OUT)" || (echo "usage: make build-embed OUT=/path/to/output/monitor" && exit 1)
	@mkdir -p $(dir $(OUT))
	go build -ldflags "$(LDFLAGS)" -o $(OUT) $(MAIN_PACKAGE_PATH)

.PHONY: run
run: build ## Build and run the monitor binary locally
	$(DIST_DIR)/$(BINARY_NAME)

.PHONY: test
test: ## Run all tests
	go test -v ./...

.PHONY: vet
vet: ## Run go vet
	go vet ./...

.PHONY: fmt
fmt: ## Run gofmt across the module
	gofmt -l -s -w .

.PHONY: clean
clean: ## Remove build artifacts
	rm -rf $(DIST_DIR)

.PHONY: help
help:
	@grep -E '^[a-zA-Z_-]+:.*## ' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*## "}; {printf "\033[36m%-16s\033[0m %s\n", $$1, $$2}'
