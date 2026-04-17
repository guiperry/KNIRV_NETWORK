# KNIRVSERVER Embedded Programs Implementation Plan

## Overview
This plan implements build, embedding, and runtime initialization for the three embedded programs in `backend/internal/embedded/`:
- graphrag-rs (Rust) - FFI CGo static library
- validation_chain (Rust) - FFI CGo static library
- transaction_chain (Node.js) - Folder embedding + go routine execution

---

## 1. Build System Integration (Makefile)

### 1.1 Directory Configuration
Add to Makefile variables section:
```makefile
# Embedded Programs
EMBEDDED_DIR := $(BACKEND_DIR)/internal/embedded
GRAPHRAG_DIR := $(EMBEDDED_DIR)/graphrag-rs
VALIDATION_CHAIN_DIR := $(EMBEDDED_DIR)/validation_chain
TRANSACTION_CHAIN_DIR := $(EMBEDDED_DIR)/transaction_chain
EMBEDDED_BUILD_DIR := $(ROOT_DIR)/build/embedded
```

### 1.2 Rust Build Targets
```makefile
# Rust Embedded Programs
graphrag-deps: ## Install graphrag-rs dependencies
	@echo "$(BLUE)Installing graphrag-rs dependencies...$(NC)"
	cd $(GRAPHRAG_DIR) && cargo fetch

graphrag-build: graphrag-deps ## Build graphrag-rs static library for CGo FFI
	@echo "$(BLUE)Building graphrag-rs static library...$(NC)"
	@mkdir -p $(EMBEDDED_BUILD_DIR)/graphrag
	cd $(GRAPHRAG_DIR) && cargo build --release --lib --features c-ffi
	cp $(GRAPHRAG_DIR)/target/release/libgraphrag.a $(EMBEDDED_BUILD_DIR)/graphrag/
	cp $(GRAPHRAG_DIR)/target/release/graphrag.h $(EMBEDDED_BUILD_DIR)/graphrag/
	@echo "$(GREEN)graphrag-rs built: $(EMBEDDED_BUILD_DIR)/graphrag/libgraphrag.a$(NC)"

validation-chain-deps: ## Install validation_chain dependencies
	@echo "$(BLUE)Installing validation_chain dependencies...$(NC)"
	cd $(VALIDATION_CHAIN_DIR) && cargo fetch

validation-chain-build: validation-chain-deps ## Build validation_chain static library for CGo FFI
	@echo "$(BLUE)Building validation_chain static library...$(NC)"
	@mkdir -p $(EMBEDDED_BUILD_DIR)/validation_chain
	cd $(VALIDATION_CHAIN_DIR) && cargo build --release --lib --features c-ffi
	cp $(VALIDATION_CHAIN_DIR)/target/release/libvalidationchain.a $(EMBEDDED_BUILD_DIR)/validation_chain/
	cp $(VALIDATION_CHAIN_DIR)/target/release/validationchain.h $(EMBEDDED_BUILD_DIR)/validation_chain/
	@echo "$(GREEN)validation_chain built: $(EMBEDDED_BUILD_DIR)/validation_chain/libvalidationchain.a$(NC)"
```

### 1.3 Node.js Build Target
```makefile
# Node.js Embedded Program
transaction-chain-deps: ## Install transaction_chain dependencies
	@echo "$(BLUE)Installing transaction_chain dependencies...$(NC)"
	cd $(TRANSACTION_CHAIN_DIR) && npm ci --production

transaction-chain-build: transaction-chain-deps ## Bundle transaction_chain for embedding
	@echo "$(BLUE)Building transaction_chain bundle...$(NC)"
	@mkdir -p $(EMBEDDED_BUILD_DIR)/transaction_chain
	cd $(TRANSACTION_CHAIN_DIR) && npm run build
	cp -R $(TRANSACTION_CHAIN_DIR)/dist/* $(EMBEDDED_BUILD_DIR)/transaction_chain/
	@echo "$(GREEN)transaction_chain bundled at $(EMBEDDED_BUILD_DIR)/transaction_chain/$(NC)"
```

### 1.4 Unified Embedded Build Target
```makefile
embedded-build: graphrag-build validation-chain-build transaction-chain-build ## Build all embedded programs
	@echo "$(GREEN)All embedded programs built successfully$(NC)"
```

### 1.5 Update Build Dependencies
Modify existing targets:
```makefile
# Add embedded-build as dependency
backend: deps-go gateway-build graph-build chain-build oracle-build hasher-build shell-build agent-build embedded-build

binary: proto ebpf-generate frontend desktop-build backend
```

---

## 2. FFI CGo Implementation (Rust Programs)

### 2.1 Rust FFI Exports
Both Rust crates will export:
- `#[no_mangle]` C-compatible functions
- C header generation via `cbindgen`
- `c-ffi` feature flag
- Static library target in Cargo.toml

### 2.2 Go CGo Wrapper Layout
```
pkg/
└── embedded/
    ├── graphrag/
    │   ├── graphrag.go         # CGo import + Go wrapper functions
    │   └── doc.go
    ├── validationchain/
    │   ├── validationchain.go  # CGo import + Go wrapper functions
    │   └── doc.go
    └── common.go               # Shared error handling, init logic
```

### 2.3 CGo Example Structure
```go
package graphrag

/*
#cgo LDFLAGS: ${SRCDIR}/../../../build/embedded/graphrag/libgraphrag.a -ldl -pthread
#include "graphrag.h"
*/
import "C"

import (
	"unsafe"
)

// Init initializes graphrag engine
func Init(config []byte) error {
	cConfig := C.CString(string(config))
	defer C.free(unsafe.Pointer(cConfig))

	result := C.graphrag_init(cConfig, C.size_t(len(config)))
	if result != 0 {
		return fmt.Errorf("graphrag init failed with code: %d", result)
	}
	return nil
}
```

---

## 3. Transaction Chain (Node.js) Embedding

### 3.1 Embedding (Optimized)
Only embed required files (no node_modules):
```go
package transactionchain

import (
	"embed"
)

//go:embed main.js
//go:embed package.json
var embeddedFiles embed.FS
```

### 3.2 Runtime Extraction
Implement extraction following existing embedded pattern:
```go
func ExtractEmbeddedApp(destDir string) (string, error) {
	// Extract all files from embeddedFS to destDir
	// Preserve file permissions
	// Write atomically
	// Check existing version hash to skip re-extraction
}
```

### 3.3 Runtime Execution Flow
```go
func Start(ctx context.Context, port int) error {
	// 1. Extract embedded files
	destDir, err := ExtractEmbeddedFiles()
	if err != nil { return err }

	// 2. Verify node/npm are available
	if err := checkNodeNpmVersions(); err != nil { return err }

	// 3. Run npm install
	installCmd := exec.CommandContext(ctx, "npm", "install", "--production")
	installCmd.Dir = destDir
	if err := installCmd.Run(); err != nil { return err }

	// 4. Start transaction chain
	cmd := exec.CommandContext(ctx, "node", "main.js", "--port", strconv.Itoa(port))
	cmd.Dir = destDir
	// Setup stdout/stderr pipes
	// Start in goroutine
	// Monitor process exit with automatic restart
}
```

---

## 4. Runtime Initialization Flow

### 4.1 Service Manager Interface
```go
type EmbeddedService interface {
	Name() string
	Init(config []byte) error
	Start(ctx context.Context) error
	Stop() error
	Health() error
}
```

### 4.2 Initialization Order
1.  GraphRag (FFI library) - initialized first
2.  Validation Chain (FFI library) - initialized second
3.  Transaction Chain (Node.js process) - started last in goroutine

### 4.3 Lifecycle Management
- All services tied to root context
- Graceful shutdown on SIGTERM/SIGINT
- Health check endpoints exposed via admin API
- Automatic restart on crash (with backoff)

---

## 5. Cross Platform Considerations

| OS      | Static Library Extension | Node.js Runtime Handling |
|---------|--------------------------|--------------------------|
| Linux   | `.a`                     | System node or embedded  |
| macOS   | `.a`                     | System node or embedded  |
| Windows | `.lib` / `.dll`          | System node.exe          |

---

## 6. Implementation Milestones

| Phase | Description | Status |
|-------|-------------|--------|
| 1     | Add Makefile build targets | Pending |
| 2     | Implement Rust FFI exports for graphrag-rs | Pending |
| 3     | Implement Go CGo wrapper for graphrag-rs | Pending |
| 4     | Implement Rust FFI exports for validation_chain | Pending |
| 5     | Implement Go CGo wrapper for validation_chain | Pending |
| 6     | Implement transaction_chain folder embedding | Pending |
| 7     | Implement transaction_chain runtime extraction | Pending |
| 8     | Implement transaction_chain go routine execution | Pending |
| 9     | Implement embedded service manager | Pending |
| 10    | Integrate into backend server startup | Pending |
| 11    | Add health checks and admin endpoints | Pending |
| 12    | Test cross-platform build | Pending |

---

## 7. Dependencies and Prerequisites

- `cargo` (Rust toolchain) required for Rust builds
- `cbindgen` for C header generation
- `node >= 18` for transaction_chain build
- `CGO_ENABLED=1` required during Go build

---

**Last Updated**: 2026-04-17
**Status**: Draft Approved
