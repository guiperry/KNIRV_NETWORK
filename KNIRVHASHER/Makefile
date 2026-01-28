.PHONY: help build build-probe build-probe-v2 build-protocol-discover build-monitor deploy deploy-probe deploy-probe-v2 deploy-protocol-discover deploy-monitor test clean hasher test-hasher

ANTMINER_IP ?= $(shell [ -f .env ] && source .env && echo $$DEVICE_IP || echo 192.168.12.151)
ANTMINER_USER ?= root
ANTMINER_PASSWORD ?= $(shell [ -f .env ] && source .env && echo $$DEVICE_PASSWORD || echo keperu100)

SDK_ROOT := $(CURDIR)/toolchain/openwrt-sdk-19.07.10-ar71xx-generic_gcc-7.5.0_musl.Linux-x86_64

# Binary name
BINARY_NAME := hasher

# Build directory
BUILD_DIR := ./bin

# Source files
SRC_DIR := ./cmd/hasher

# Test directories
TEST_DIRS := ./internal/cli/ui ./internal/cli/server ./internal/cli/chat ./tests

# Build flags
GOFLAGS :=


help:
	@echo "🛡️  Hasher PoC - Development Commands"
	@echo ""
	@echo "Available targets:"
	@echo "  make build                   - Build diagnostic tool (MIPS)"
	@echo "  make build-probe             - Build device probe tool (MIPS)"
	@echo "  make build-probe-v2          - Build enhanced probe tool v2 (MIPS)"
	@echo "  make build-protocol-discover - Build protocol discovery tool (MIPS)"
	@echo "  make build-monitor           - Build ASIC monitor tool (MIPS)"
	@echo "  make hasher                  - Build CLI tool (hasher)"
	@echo "  make deploy                  - Build and deploy diagnostics"
	@echo "  make deploy-probe            - Build and deploy probe tool"
	@echo "  make deploy-probe-v2         - Build and deploy enhanced probe v2"
	@echo "  make deploy-protocol-discover- Build and deploy protocol discovery"
	@echo "  make deploy-monitor          - Build and deploy ASIC monitor"
	@echo "  make test                    - Build, deploy, and run diagnostics"
	@echo "  make test-hasher             - Build, deploy, and run CLI tool"
	@echo "  make clean                   - Clean build artifacts"
	@echo ""
	@echo "Configuration:"
	@echo "  ANTMINER_IP=$(ANTMINER_IP)"
	@echo "  ANTMINER_USER=$(ANTMINER_USER)"

build:
	@echo "🔨 Building diagnostic tool for MIPS..."
	@mkdir -p bin
	GOOS=linux GOARCH=mips GOMIPS=softfloat go build -o bin/asic-test-mips cmd/asic-test/main.go
	@echo "✅ Build complete"
	@ls -lh bin/asic-test-mips

build-probe:
	@echo "🔨 Building device probe for MIPS..."
	@mkdir -p bin
	GOOS=linux GOARCH=mips GOMIPS=softfloat go build -o bin/device-probe-mips cmd/device-probe/main.go
	@echo "✅ Build complete"
	@ls -lh bin/device-probe-mips

build-probe-v2:
	@echo "🔨 Building enhanced device probe v2 for MIPS..."
	@mkdir -p bin
	GOOS=linux GOARCH=mips GOMIPS=softfloat go build -o bin/device-probe-v2-mips cmd/device-probe-v2/main.go
	@echo "✅ Build complete"
	@ls -lh bin/device-probe-v2-mips

build-protocol-discover:
	@echo "🔨 Building protocol discovery tool for MIPS..."
	@mkdir -p bin
	GOOS=linux GOARCH=mips GOMIPS=softfloat go build -o bin/protocol-discover-mips cmd/protocol-discover/main.go
	@echo "✅ Build complete"
	@ls -lh bin/protocol-discover-mips

build-monitor:
	@echo "🔨 Building ASIC monitor tool for MIPS (static)..."
	@mkdir -p bin
	CGO_ENABLED=1 \
	CC=$(SDK_ROOT)/staging_dir/toolchain-mips_24kc_gcc-7.5.0_musl/bin/mips-openwrt-linux-musl-gcc \
	CGO_CFLAGS="-I$(SDK_ROOT)/staging_dir/target-mips_24kc_musl/usr/include" \
	CGO_LDFLAGS="-L$(SDK_ROOT)/staging_dir/target-mips_24kc_musl/usr/lib -lusb-1.0 -static" \
	GOOS=linux GOARCH=mips GOMIPS=softfloat \
	go build -ldflags '-extldflags "-static"' -o bin/asic-monitor-mips cmd/asic-monitor/main.go
	@echo "✅ Build complete"
	@ls -lh bin/asic-monitor-mips


hasher:
	@echo "Creating build directory..."
	@mkdir -p $(BUILD_DIR)
	@echo "Building $(BINARY_NAME)..."
	@go build $(GOFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) $(SRC_DIR)
	@echo "Binary created successfully at $(BUILD_DIR)/$(BINARY_NAME)"

test-hasher:
	@echo "Running tests..."
	@for dir in $(TEST_DIRS); do \
		echo "Running tests in $$dir..."; \
		go test -v $$dir; \
	done

clean:
	@echo "Cleaning up..."
	@rm -rf $(BUILD_DIR)
	@echo "Cleanup completed"

run:
	@echo "Running $(BINARY_NAME)..."
	@go run $(SRC_DIR)


deploy: build
	@./scripts/deploy.sh

deploy-probe: build-probe
	@echo "🚀 Deploying device probe to Antminer..."
	@sshpass -p '$(ANTMINER_PASSWORD)' scp -o KexAlgorithms=+diffie-hellman-group14-sha1 \
		-o HostKeyAlgorithms=+ssh-rsa -o StrictHostKeyChecking=no \
		bin/device-probe-mips $(ANTMINER_USER)@$(ANTMINER_IP):/tmp/device-probe
	@sshpass -p '$(ANTMINER_PASSWORD)' ssh -o KexAlgorithms=+diffie-hellman-group14-sha1 \
		-o HostKeyAlgorithms=+ssh-rsa -o StrictHostKeyChecking=no \
		$(ANTMINER_USER)@$(ANTMINER_IP) 'chmod +x /tmp/device-probe'
	@echo "✅ Deployed! Run with:"
	@echo "   sshpass -p '$(ANTMINER_PASSWORD)' ssh -o KexAlgorithms=+diffie-hellman-group14-sha1 \\"
	@echo "     -o HostKeyAlgorithms=+ssh-rsa $(ANTMINER_USER)@$(ANTMINER_IP) '/tmp/device-probe'"

deploy-probe-v2: build-probe-v2
	@echo "🚀 Deploying enhanced device probe v2 to Antminer..."
	@sshpass -p '$(ANTMINER_PASSWORD)' scp -o KexAlgorithms=+diffie-hellman-group14-sha1 \
		-o HostKeyAlgorithms=+ssh-rsa -o StrictHostKeyChecking=no \
		bin/device-probe-v2-mips $(ANTMINER_USER)@$(ANTMINER_IP):/tmp/device-probe-v2
	@sshpass -p '$(ANTMINER_PASSWORD)' ssh -o KexAlgorithms=+diffie-hellman-group14-sha1 \
		-o HostKeyAlgorithms=+ssh-rsa -o StrictHostKeyChecking=no \
		$(ANTMINER_USER)@$(ANTMINER_IP) 'chmod +x /tmp/device-probe-v2'
	@echo "✅ Deployed! Run with:"
	@echo "   sshpass -p '$(ANTMINER_PASSWORD)' ssh -o KexAlgorithms=+diffie-hellman-group14-sha1 \\"
	@echo "     -o HostKeyAlgorithms=+ssh-rsa $(ANTMINER_USER)@$(ANTMINER_IP) '/tmp/device-probe-v2'"

deploy-protocol-discover: build-protocol-discover
	@echo "🚀 Deploying protocol discovery tool to Antminer..."
	@sshpass -p '$(ANTMINER_PASSWORD)' scp -o KexAlgorithms=+diffie-hellman-group14-sha1 \
		-o HostKeyAlgorithms=+ssh-rsa -o StrictHostKeyChecking=no \
		bin/protocol-discover-mips $(ANTMINER_USER)@$(ANTMINER_IP):/tmp/protocol-discover
	@sshpass -p '$(ANTMINER_PASSWORD)' ssh -o KexAlgorithms=+diffie-hellman-group14-sha1 \
		-o HostKeyAlgorithms=+ssh-rsa -o StrictHostKeyChecking=no \
		$(ANTMINER_USER)@$(ANTMINER_IP) 'chmod +x /tmp/protocol-discover'
	@echo "✅ Deployed! Run with:"
	@echo "   sshpass -p '$(ANTMINER_PASSWORD)' ssh -o KexAlgorithms=+diffie-hellman-group14-sha1 \\"
	@echo "     -o HostKeyAlgorithms=+ssh-rsa $(ANTMINER_USER)@$(ANTMINER_IP) '/tmp/protocol-discover'"

deploy-monitor: build-monitor
	@echo "🚀 Deploying ASIC monitor tool to Antminer..."
	@sshpass -p '$(ANTMINER_PASSWORD)' scp -o KexAlgorithms=+diffie-hellman-group14-sha1 \
		-o HostKeyAlgorithms=+ssh-rsa -o StrictHostKeyChecking=no \
		bin/asic-monitor-mips $(ANTMINER_USER)@$(ANTMINER_IP):/tmp/asic-monitor
	@sshpass -p '$(ANTMINER_PASSWORD)' ssh -o KexAlgorithms=+diffie-hellman-group14-sha1 \
		-o HostKeyAlgorithms=+ssh-rsa -o StrictHostKeyChecking=no \
		$(ANTMINER_USER)@$(ANTMINER_IP) 'chmod +x /tmp/asic-monitor'
	@echo "✅ Deployed! Run with:"
	@echo "   sshpass -p '$(ANTMINER_PASSWORD)' ssh -o KexAlgorithms=+diffie-hellman-group14-sha1 \\"
	@echo "     -o HostKeyAlgorithms=+ssh-rsa $(ANTMINER_USER)@$(ANTMINER_IP) '/tmp/asic-monitor --dump-status --dump-interval 2'"

test: build
	@echo "Running full diagnostic test..."
	@./scripts/deploy.sh

clean:
	@echo "🧹 Cleaning build artifacts..."
	rm -rf bin/
	@echo "✅ Clean complete"
