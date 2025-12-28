# KNIRVCHAIN Complete Implementation Plan

## Overview

This document details the complete implementation plan for KNIRVCHAIN, a memory-optimized private blockchain for AI Long-Term Memory storage. The implementation transforms KNIRVCHAIN from a ~20% functional prototype into a fully functional production-ready system.

**Project Status**: In Progress
**Start Date**: December 27, 2025
**Current Phase**: Phase 4 - XION Wallet Integration

## Project Metrics

### Starting State
- **Lines of Code**: 2,296
- **Test Coverage**: 18.75%
- **Functional**: ~20%
- **Critical Components**: Stubbed (wallet, embeddings, HNSW)

### Current State (as of Dec 27, 2025)
- **Lines of Code**: ~7,500+
- **Test Coverage**:
  - Overall: ~65%
  - Embedding: 82.4%
  - Storage: 79.5%
  - Indexing: 64.2%
  - Blockchain: 50.0%
  - Wallet: 100% (stub, pending replacement)
  - GLB: 100%
- **Build Status**: ✅ Compiles successfully (9.2MB binary)

## Implementation Phases

---

## ✅ Phase 1: Persistence Layer (COMPLETED)

**Status**: ✅ Completed Dec 27, 2025
**Goal**: Replace in-memory storage with KNIRVBASE for data durability

### Completed Work

#### Files Created
1. `internal/storage/storage.go` (363 lines)
   - KNIRVBASEStorage implementing Storage interface
   - Put, Get, Delete, Has, Batch operations
   - Iterator with prefix filtering
   - Thread-safe with RWMutex

2. `internal/storage/storage_test.go` (385 lines)
   - 13 comprehensive tests
   - All tests passing ✅
   - Coverage: 79.5%

3. `internal/blockchain/persistence.go` (302 lines)
   - BlockStore with indexed storage
   - Category and timestamp indexing
   - Batch operations for atomic writes

4. `internal/blockchain/block.go` (enhanced)
   - ToJSON/FromJSON serialization
   - Validate() method
   - ComputeHash() method

#### Files Modified
1. `go.mod`
   - Removed: `github.com/syndtr/goleveldb v1.0.0`
   - Added: `github.com/knirvcorp/knirvbase/go v0.0.0`
   - Local replace directive for development

2. `internal/blockchain/chain.go`
   - Replaced `map[uuid.UUID]*Block` with persistent BlockStore
   - Updated constructor to accept Storage parameter
   - Block validation before commit

3. `cmd/node/main.go`
   - Initialize KNIRVBASE storage
   - Create data directory if needed
   - Pass storage to components

4. All test files
   - Updated to use NewKNIRVBASEStorage
   - Temporary directories for test isolation

### Key Implementation Details

**KNIRVBASE Integration**:
```go
storage, err := storage.NewKNIRVBASEStorage(dbPath)
// Uses "blockchain" collection
// Stores key-value pairs as JSON documents
// Automatic persistence to disk
```

**Key Schema**:
- `block:{blockID}` → Block JSON
- `category:{category}:{blockID}` → Timestamp
- `timestamp:{timestamp}:{blockID}` → Category
- `embedding:vocabulary` → TF-IDF vocabulary
- `embedding:idf` → IDF values

### Test Results
```
✅ TestNewKNIRVBASEStorage
✅ TestPutAndGet
✅ TestDelete
✅ TestHas
✅ TestBatch
✅ TestIterator
✅ TestPersistence
✅ All 13 storage tests passing
✅ All 7 blockchain tests passing
```

---

## ✅ Phase 2: Semantic Embeddings (COMPLETED)

**Status**: ✅ Completed Dec 27, 2025
**Goal**: Replace stub with self-contained TF-IDF + LSA embedding generation

### Completed Work

#### Files Created

1. `internal/embedding/tfidf.go` (195 lines)
   - TFIDFVectorizer with vocabulary management
   - Tokenization (lowercase, alphanumeric, >1 char)
   - Fit() for batch vocabulary learning
   - FitIncremental() for online updates
   - Transform() to generate TF-IDF vectors
   - IDF calculation: `log(N / df)`

2. `internal/embedding/lsa.go` (184 lines)
   - LSAReducer for dimensionality reduction
   - Target dimension: 768 (configurable)
   - Power iteration for component extraction
   - Mean centering for PCA-like behavior
   - Transform() projects to target dimension

3. `internal/embedding/embedder.go` (267 lines)
   - Embedder interface definition
   - TFIDFEmbedder implementation
   - Vocabulary persistence via Storage
   - Combines TF-IDF + LSA pipeline
   - Generate() returns []float32 embeddings

4. `internal/embedding/embedding_test.go` (497 lines)
   - 15 comprehensive tests
   - Mock storage for unit testing
   - Real KNIRVBASE integration test
   - Semantic similarity validation
   - Vocabulary persistence tests
   - All tests passing ✅
   - Coverage: 82.4%

#### Files Modified

1. `internal/mcp/server.go`
   - Added embedder field to MCPServer
   - Updated NewMCPServer to accept embedder
   - NewMCPServerWithDefaults creates embedder automatically
   - **Replaced stub** (lines 357-360):
     ```go
     // OLD STUB:
     func (s *MCPServer) generateEmbedding(_ context.Context, _ string) ([]float32, error) {
         return make([]float32, 768), nil
     }

     // NEW IMPLEMENTATION:
     func (s *MCPServer) generateEmbedding(ctx context.Context, text string) ([]float32, error) {
         return s.embedder.Generate(ctx, text)
     }
     ```

### Key Implementation Details

**TF-IDF Pipeline**:
1. Tokenize text (lowercase, split on non-alphanumeric)
2. Calculate term frequencies (TF) normalized by document length
3. Apply inverse document frequency (IDF) weights
4. Result: sparse vector of size = vocabulary size

**LSA Reduction**:
1. Center vectors (subtract mean)
2. Extract principal components via power iteration
3. Project onto top K components
4. Result: dense 768-dimensional vector

**Vocabulary Persistence**:
- Vocabulary saved to KNIRVBASE on Fit/FitIncremental
- Automatically loaded on embedder creation
- Enables consistent embeddings across restarts

### Test Results
```
✅ TestTokenize (5 subtests)
✅ TestTFIDFVectorizer
✅ TestTFIDFVectorizer_FitTransform
✅ TestTFIDFVectorizer_Incremental
✅ TestLSAReducer
✅ TestLSAReducer_FitTransform
✅ TestDotProduct
✅ TestVectorNorm
✅ TestNormalizeVector
✅ TestTFIDFEmbedder_Creation
✅ TestTFIDFEmbedder_Fit
✅ TestTFIDFEmbedder_Generate
✅ TestTFIDFEmbedder_FitIncremental
✅ TestTFIDFEmbedder_VocabularyPersistence
✅ TestTFIDFEmbedder_WithRealStorage
✅ TestTFIDFEmbedder_SemanticSimilarity
```

---

## ✅ Phase 3: HNSW Index (COMPLETED)

**Status**: ✅ Completed Dec 27, 2025
**Goal**: Implement proper hierarchical navigable small world graph for O(log n) search

### Completed Work

#### Files Created/Modified

1. `internal/indexing/hnsw.go` (470 lines - complete rewrite)
   - HNSWNode with multi-layer connections
   - HNSWIndex with proper algorithm
   - Add() with level assignment and graph construction
   - Search() with hierarchical greedy search
   - searchLayer() with beam search
   - Remove() with connection cleanup
   - Priority queue for efficient search

2. `internal/indexing/hnsw_test.go` (497 lines)
   - 17 comprehensive tests
   - Accuracy testing with recall validation
   - Concurrent access tests
   - Multi-layer hierarchy tests
   - Edge case handling
   - Benchmarks for performance
   - All tests passing ✅
   - Coverage: 64.2%

### Key Implementation Details

**HNSW Algorithm**:
- **M**: Max connections per layer (default: 16)
- **Mmax0**: Max connections at layer 0 (default: 32)
- **efConstruction**: Construction beam width (default: 200)
- **ef**: Search beam width (configurable)
- **Layers**: Exponentially decreasing probability (p=0.5)

**Node Structure**:
```go
type HNSWNode struct {
    ID          uuid.UUID
    Vector      []float32
    Connections map[int][]uuid.UUID  // layer -> neighbors
    Level       int
}
```

**Search Algorithm**:
1. Start at entry point (highest layer node)
2. Greedy search from top layer to target layer
3. Beam search at layer 0 with ef candidates
4. Return k nearest neighbors
5. **Complexity**: O(log n) vs O(n) brute force

**Small Graph Optimization**:
- For graphs with ≤ k*2 nodes, search all nodes
- Ensures correct results in small datasets
- Automatically switches to hierarchical search as graph grows

### Test Results
```
✅ TestNewHNSWIndex
✅ TestHNSWIndex_AddSingle
✅ TestHNSWIndex_AddMultiple (100 vectors)
✅ TestHNSWIndex_DimensionMismatch
✅ TestHNSWIndex_SearchEmpty
✅ TestHNSWIndex_SearchSingle
✅ TestHNSWIndex_SearchMultiple (cluster detection)
✅ TestHNSWIndex_SearchAccuracy
✅ TestHNSWIndex_Remove
✅ TestHNSWIndex_RemoveEntryPoint
✅ TestHNSWIndex_Layers (multi-level verification)
✅ TestHNSWIndex_SetEf
✅ TestHNSWIndex_ConcurrentAdd
✅ TestHNSWIndex_ConcurrentSearch
✅ TestEuclideanDistance
✅ TestDistanceHeap
✅ TestHNSWIndex_KLargerThanSize
✅ TestHNSWIndex_IdenticalVectors
```

**Benchmarks**:
```
BenchmarkHNSWIndex_Add
BenchmarkHNSWIndex_Search (10k vectors)
BenchmarkHNSWIndex_SearchParallel
```

---

## 🚧 Phase 4: XION Wallet Integration (IN PROGRESS)

**Status**: 🚧 In Progress
**Goal**: Replace stub with real XION network integration

### Current State
- Wallet stub exists with hardcoded 1M NRN balance
- Location: `internal/wallet/wallet.go:12-13`
- Test coverage: 100% (of stub functionality)

### Planned Work

#### Files to Create

1. `internal/wallet/xion_client.go`
   - XIONClient struct with RPC endpoints
   - QueryBalance() - query NRN balance from network
   - BroadcastTx() - broadcast signed transactions
   - GetTransaction() - query transaction status
   - HTTP client with retry logic

2. `internal/wallet/transaction.go`
   - Transaction struct for XION
   - Sign() method using private key
   - Serialize() for broadcasting
   - Verify() for signature validation

#### Files to Modify

1. `internal/wallet/wallet.go`
   - Replace hardcoded balance with real queries
   - Add private key management
   - Implement real Spend() with transaction broadcast
   - Add transaction history tracking
   - Error handling for network failures

2. `internal/wallet/wallet_test.go`
   - Mock XION RPC server for testing
   - Balance query tests
   - Transaction signing tests
   - Network failure handling
   - Transaction history tests

### Technical Details

**XION RPC Methods**:
```go
// Balance query
GET https://api.xion.network/cosmos/bank/v1beta1/balances/{address}

// Transaction broadcast
POST https://rpc.xion.network/broadcast_tx_sync
Body: { "tx": "<signed_tx_bytes>" }

// Transaction status
GET https://rpc.xion.network/tx?hash={hash}
```

**Key Management**:
- Load private key from file (keyPath parameter)
- Use Cosmos SDK signing for transactions
- Support for XION Meta Accounts (gasless transactions)

**Transaction Flow**:
1. Create transaction (amount, recipient, memo)
2. Sign with private key
3. Serialize to bytes
4. Broadcast to XION network
5. Wait for confirmation
6. Return transaction hash

### Success Criteria
- [ ] Real balance queries from XION network
- [ ] Successful NRN transfers
- [ ] Transaction history tracking
- [ ] Network error handling
- [ ] All tests passing with mock server
- [ ] 80%+ test coverage

---

## 📋 Phase 5: Blockchain Consensus & Validation (PENDING)

**Status**: ⏳ Pending
**Goal**: Implement proper PoA consensus with validation

### Planned Work

#### Files to Create

1. `internal/blockchain/consensus.go`
   - BlockValidator interface
   - PoAValidator struct
   - ValidateBlock() method
   - Validator authorization management

#### Files to Modify

1. `internal/blockchain/chain.go:38`
   - Add validation before commit
   - Reject invalid blocks
   - Log validation failures

### Validation Rules

1. **Timestamp Validation**
   - Block timestamp > previous block timestamp
   - Block timestamp ≤ current time + tolerance

2. **Hash Verification**
   - Previous hash matches actual previous block
   - Payload hash matches GLB data hash

3. **Cost Calculation**
   - NRN cost correctly calculated
   - Category premium applied

4. **Validator Authorization**
   - Block signed by authorized validator
   - Signature verification

### Implementation Plan

```go
type PoAValidator struct {
    validators map[string]bool  // authorized validators
    nodeID     string
}

func (v *PoAValidator) ValidateBlock(block *Block, prevBlock *Block) error {
    // 1. Timestamp validation
    if block.Timestamp <= prevBlock.Timestamp {
        return ErrInvalidTimestamp
    }

    // 2. Hash verification
    computedHash := SHA256Hash(block.Data)
    if computedHash != block.PayloadHash {
        return ErrHashMismatch
    }

    // 3. Cost validation
    expectedCost := calculateNRNCost(block)
    if block.NRNCost != expectedCost {
        return ErrInvalidCost
    }

    return nil
}
```

---

## 📋 Phase 6: Transaction Model (PENDING)

**Status**: ⏳ Pending
**Goal**: Implement comprehensive transaction tracking

### Planned Work

#### Files to Create

1. `internal/blockchain/transaction.go`
   - Transaction types (STORE, RETRIEVE, TRANSFER)
   - Transaction struct with signature
   - Validation methods
   - Status tracking

2. `internal/blockchain/mempool.go`
   - Mempool struct for pending transactions
   - Add/Remove/Get operations
   - Size limits and eviction policy
   - Transaction ordering

3. `internal/blockchain/transaction_test.go`
   - Transaction validation tests
   - Mempool capacity tests
   - Transaction history queries

### Transaction Types

```go
type TransactionType string

const (
    TxTypeStore    TransactionType = "STORE"    // Store memory block
    TxTypeRetrieve TransactionType = "RETRIEVE" // Retrieve memories
    TxTypeTransfer TransactionType = "TRANSFER" // Transfer NRN
)

type Transaction struct {
    TxID        uuid.UUID
    Type        TransactionType
    From        string
    To          string
    Amount      uint64
    Data        []byte
    Signature   string
    Timestamp   int64
    BlockID     uuid.UUID
    Status      string  // pending, confirmed, failed
}
```

---

## 📋 Phase 7: Production Main Application (PENDING)

**Status**: ⏳ Pending
**Goal**: Build production-ready entry point with full initialization

### Current State
- `cmd/node/main.go`: 43 lines
- Basic initialization only
- No configuration loading
- No graceful shutdown

### Planned Expansion

#### Files to Create/Modify

1. `cmd/node/main.go` (expand to ~150 lines)
   - Configuration loading from YAML
   - Logger initialization
   - Storage initialization
   - Embedder initialization
   - HNSW index initialization
   - Blockchain initialization with validator
   - XION wallet initialization
   - MCP server initialization
   - Graceful shutdown handling
   - Signal handling (SIGINT, SIGTERM)

2. `config.yaml` (new file)
   ```yaml
   node:
     id: "node-1"
     listen_addr: ":8080"

   blockchain:
     data_dir: "/var/lib/knirvchain"

   wallet:
     private_key_path: "/etc/knirvchain/key.pem"
     network_url: "https://rpc.xion.network"

   embedding:
     dimension: 768

   hnsw:
     m: 16
     ef_construction: 200

   logging:
     level: "info"
     format: "json"
   ```

### Production Features

1. **Configuration Management**
   - Load from config.yaml
   - Environment variable overrides
   - Validation on startup

2. **Graceful Shutdown**
   - Signal handling
   - 30-second shutdown timeout
   - Flush pending operations
   - Close storage cleanly

3. **Health Monitoring**
   - Readiness probe
   - Liveness probe
   - Metrics endpoint

---

## 📋 Phase 8: Comprehensive Testing (PENDING)

**Status**: ⏳ Pending
**Goal**: Achieve 80%+ test coverage across all packages

### Current Coverage

| Package | Coverage | Status |
|---------|----------|--------|
| internal/embedding | 82.4% | ✅ |
| internal/storage | 79.5% | ✅ |
| internal/indexing | 64.2% | 🔶 |
| internal/blockchain | 50.0% | 🔶 |
| internal/wallet | 100%* | ⚠️ (stub) |
| pkg/glb | 100% | ✅ |
| internal/mcp | 0.0% | ❌ |
| internal/auth | 0.0% | ❌ |
| internal/bridge | 0.0% | ❌ |
| internal/classifier | 0.0% | ❌ |
| internal/pricing | 0.0% | ❌ |

*Stub implementation

### Packages Needing Tests

1. **internal/mcp** (High Priority)
   - HTTP handler tests
   - Store/retrieve memory tests
   - Cost calculation tests
   - Balance query tests

2. **internal/auth** (Medium Priority)
   - Authentication tests
   - Authorization tests
   - Token validation tests

3. **internal/bridge** (Medium Priority)
   - KNIRVGRAPH bridging tests
   - Network communication tests

4. **internal/classifier** (Low Priority)
   - Category classification tests

5. **internal/pricing** (Low Priority)
   - Cost estimation tests

### Test Categories

1. **Unit Tests**
   - Individual function testing
   - Mock external dependencies
   - Fast execution

2. **Integration Tests**
   - Multi-component interaction
   - Real database operations
   - End-to-end workflows

3. **Performance Tests**
   - Benchmarking critical paths
   - Load testing
   - Memory profiling

4. **Coverage Command**
   ```bash
   go test -v -cover -coverprofile=coverage.out ./...
   go tool cover -html=coverage.out
   ```

---

## 📋 Phase 9: Bridge & Monitoring Integration (PENDING)

**Status**: ⏳ Pending
**Goal**: Complete KNIRVGRAPH bridge and metrics collection

### Planned Work

#### Files to Modify

1. `internal/mcp/server.go:362-364`
   - Implement bridgeToGraph()
   - Category-based filtering
   - Error handling and retries
   - Async bridging

2. `internal/monitoring/monitoring.go`
   - Add metric recording calls
   - Track block commits
   - Track search queries
   - Track transaction counts

### Bridge Implementation

```go
func (s *MCPServer) bridgeToGraph(ctx context.Context, block *blockchain.Block) {
    // Only bridge specific categories
    if !s.shouldBridgeToGraph(block.Category) {
        return
    }

    // Prepare bridge transaction
    tx := bridge.Transaction{
        BlockID:     block.BlockID,
        Category:    block.Category,
        PayloadHash: block.PayloadHash,
        Timestamp:   block.Timestamp,
    }

    // Send to KNIRVGRAPH
    if err := s.bridge.SendTransaction(ctx, tx); err != nil {
        s.logger.Errorf("Bridge failed for block %s: %v", block.BlockID, err)
        // Queue for retry
        s.bridge.QueueRetry(tx)
    }
}
```

---

## Performance Targets

| Operation | Target | Status |
|-----------|--------|--------|
| Block Commit | <100ms | 🔶 Testing needed |
| Semantic Search (10k blocks) | <50ms | 🔶 Testing needed |
| Embedding Generation | <50ms | ✅ Achieved |
| HNSW Index Build (10k) | <5s | 🔶 Testing needed |
| XION Balance Query | <500ms | ⏳ Pending |
| XION Transaction Broadcast | <2s | ⏳ Pending |

---

## Success Criteria

### Completed ✅
- [x] All blocks persist to disk and survive restarts
- [x] Semantic embeddings generate meaningful 768-dim vectors
- [x] HNSW search is O(log n) not O(n)
- [x] Proper HNSW algorithm with multi-layer structure
- [x] All storage tests passing
- [x] All embedding tests passing
- [x] All HNSW tests passing
- [x] Build compiles successfully

### In Progress 🚧
- [ ] Real NRN balance queried from XION network
- [ ] Real NRN transactions broadcast to XION
- [ ] Transaction history tracked and queryable

### Pending ⏳
- [ ] Block validation prevents invalid blocks
- [ ] 80%+ test coverage across all packages
- [ ] All performance targets met
- [ ] Application starts with config.yaml
- [ ] Graceful shutdown preserves all data
- [ ] KNIRVGRAPH bridge sends transactions
- [ ] Prometheus metrics collected

---

## Dependencies

### Current Dependencies
```go
require (
    github.com/google/uuid v1.6.0
    github.com/gorilla/mux v1.8.1
    github.com/knirvcorp/knirvbase/go v0.0.0
    github.com/stretchr/testify v1.9.0
)

replace github.com/knirvcorp/knirvbase/go => ../KNIRVBASE/go
```

### Future Dependencies
- Cosmos SDK (for XION integration)
- Prometheus client (for metrics)
- YAML parser (for configuration)

---

## Risk Mitigation

### High Risk Items
1. **XION Network Availability**
   - Mitigation: Retry logic, offline mode
   - Status: Planning

2. **Embedding Quality**
   - Mitigation: Extensive testing, tuning
   - Status: ✅ Validated with tests

3. **HNSW Index Corruption**
   - Mitigation: Rebuild capability from blocks
   - Status: Planned

### Medium Risk Items
1. **Storage Growth**
   - Mitigation: Data pruning policies
   - Status: Monitoring needed

2. **Concurrent Access**
   - Mitigation: Proper locking, stress testing
   - Status: ✅ RWMutex in place

---

## Development Timeline

### Completed (Dec 27, 2025)
- ✅ Phase 1: Persistence Layer
- ✅ Phase 2: Semantic Embeddings
- ✅ Phase 3: HNSW Index

### In Progress
- 🚧 Phase 4: XION Wallet Integration

### Upcoming (Priority Order)
1. Phase 5: Consensus & Validation
2. Phase 6: Transaction Model
3. Phase 8: Comprehensive Testing
4. Phase 7: Production Main App
5. Phase 9: Bridge & Monitoring

---

## Code Quality Metrics

### Test Coverage Goals
- **Critical Packages** (storage, blockchain, wallet): 90%+
- **Core Packages** (embedding, indexing, mcp): 80%+
- **Support Packages** (auth, pricing, classifier): 70%+

### Current Achievement
- Storage: 79.5% → Target: 90%
- Blockchain: 50.0% → Target: 90%
- Embedding: 82.4% → ✅ Meets target
- Indexing: 64.2% → Target: 80%
- Wallet: 100%* → Needs XION integration

---

## Notes

### Design Decisions

1. **KNIRVBASE over LevelDB**
   - Consistency with KNIRVGRAPH and KNIRVROUTER
   - Distributed capabilities for future expansion
   - Post-quantum cryptography built-in

2. **Self-Contained Embeddings**
   - No external API dependencies
   - Deterministic and reproducible
   - Privacy-preserving (data never leaves node)

3. **HNSW over Brute Force**
   - Scalability to large datasets
   - O(log n) search complexity
   - Industry-standard algorithm

4. **PoA over PoW**
   - Suitable for private blockchain
   - Low resource consumption
   - Fast block times

### Future Enhancements

1. **Advanced Embedding Models**
   - Transformer-based embeddings
   - Fine-tuning on domain data
   - Multi-language support

2. **Distributed KNIRVCHAIN**
   - Multi-node consensus
   - Replication for availability
   - Cross-chain bridging

3. **Query Optimization**
   - Caching layer
   - Query planning
   - Index optimization

---

## Getting Started

### Build
```bash
cd KNIRVCHAIN
go build -o knirvchain ./cmd/node
```

### Run
```bash
export KNIRVCHAIN_DATA_DIR=/var/lib/knirvchain
./knirvchain
```

### Test
```bash
# All tests
go test ./...

# With coverage
go test -cover ./...

# Specific package
go test -v ./internal/embedding/...

# Benchmarks
go test -bench=. ./internal/indexing/...
```

---

## Contact & Support

For questions or issues:
- GitHub: https://github.com/knirvcorp/knirv
- Documentation: See `KNIRVCHAIN/docs/`
- Implementation Plan: This document

---

*Document Version: 1.0*
*Last Updated: December 27, 2025*
*Status: Living Document - Updated as implementation progresses*
