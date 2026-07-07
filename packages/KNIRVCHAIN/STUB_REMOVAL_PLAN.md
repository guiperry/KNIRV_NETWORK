# KNIRVCHAIN Stub & Placeholder Removal Plan

This document outlines the plan to remove all stubs, placeholders, and unused code from KNIRVCHAIN now that non-sovereign functionality has moved to KNIRVORACLE.

---

## Phase 1: Remove Entire Modules (Moved to KNIRVORACLE)

### 1.1 P2P Consensus Module
**Files to delete or strip:**
- `internal/p2p/p2p_consensus.go` - Entire file (~1100 lines)
- Remove ConsensusManager interface usage from:
  - `cmd/knirvchain/main.go:355-430`
  - `internal/p2p/interfaces.go:12-30`

**Action:** Delete P2PConsensusManager type and all its methods. Each KNIRVCHAIN is now sovereign.

---

### 1.2 Economics Integration
**Files to delete:**
- `internal/integrations/economics/` - Entire directory
- `config/components/economics.go`

**Action:** Delete entire `internal/integrations/economics/` directory. Economics moved to KNIRVORACLE.

---

### 1.3 XION Integration  
**Files to delete:**
- `internal/integrations/xion/` - Entire directory

**Action:** Delete entire `internal/integrations/xion/` directory. Cross-chain payments handled by KNIRVORACLE.

---

### 1.4 LoRA Manager
**Files to modify:**
- `internal/mining/lora_manager.go` - Delete file or stub out completely
- Remove LoRAAdapterPointer type usage from:
  - `internal/types/pointer_types.go`
  - `internal/mining/skill_mining.go`
  - `internal/validation/proof_verifier.go`

**Action:** LoRA adapters are now stored with skill nodes in KNIRVORACLE. Remove from KNIRVCHAIN.

---

## Phase 2: Remove Placeholder Types

### 2.1 Network Placeholder Types
**File:** `cmd/knirvchain/main.go:50-76`

```go
// DELETE these placeholder types:
type WAN struct {}
type HostID struct {}
type Host struct {}
type DHT struct {}
```

**Replace with:** Either remove entirely or use actual libp2p types.

---

### 2.2 ConsensusManager Wrapper (main.go)
**File:** `cmd/knirvchain/main.go:355-430`

```go
// DELETE entire ConsensusManager struct and all its methods:
// - StartConsensus()
// - StopConsensus()  
// - ProposeValue()
// - VoteOnProposal()
// - AddParticipant()
// - RemoveParticipant()
// - OnConsensusReached()
// - OnProposalReceived()
```

---

## Phase 3: Fully Implement MCPProcessor Methods

### 3.1 ProcessMCPRegisterCapability
**File:** `internal/blockchain/blockchain_struct.go:48-50`

**Current:**
```go
func (mcp *MCPProcessor) ProcessMCPRegisterCapability(tx *Transaction) error {
    return fmt.Errorf("process mcp register capability not implemented")
}
```

**Implement:**
- Parse transaction data as RegisterCapabilityData
- Validate capability schema
- Store in ChromemDB capability collection
- Create ContextRecord for registration
- Return nil on success

---

### 3.2 ProcessMCPInvokeCapability
**File:** `internal/blockchain/blockchain_struct.go:52-54`

**Current:** Returns "not implemented" error

**Implement:**
- Parse transaction as InvokeCapabilityData
- Locate capability in ChromemDB
- Execute capability (invoke MCP server)
- Create ContextRecord with result
- Return nil on success

---

### 3.3 ApplyMCPTransactionEffects
**File:** `internal/blockchain/blockchain_struct.go:56-62`

**Current:** Returns "not implemented" error

**Implement:**
- Update blockchain state (balances, capability ownership)
- Handle gas fee distribution
- Commit changes to LevelDB

---

## Phase 4: Fully Implement Agent Manager Methods

### 4.1 LinkResourceToCapability
**File:** `internal/blockchain/blockchain_struct.go:204-206`

**Current:** Returns "not implemented" error

**Implement:**
- Verify resource exists
- Verify capability exists
- Create resource-capability link in ChromemDB
- Update agent metadata

---

### 4.2 CreateResourceCapabilityGroup
**File:** `internal/blockchain/blockchain_struct.go:208-210`

**Current:** Returns "not implemented" error

**Implement:**
- Validate all capability IDs exist
- Create group metadata
- Store group in ChromemDB
- Link to agent

---

### 4.3 GetResourceCapabilities / GetResourceCapabilityGroups
**File:** `internal/blockchain/blockchain_struct.go:212-228`

**Current:** Returns empty arrays

**Implement:**
- Query ChromemDB for agent's resources/groups
- Return actual data

---

## Phase 5: Fully Implement ChromemManager Methods

### 5.1 OnNewBlockConfirmed
**File:** `internal/blockchain/blockchain_struct.go:340-343`

**Current:** Returns "not implemented" error

**Implement:**
- Sync newly confirmed capabilities to ChromemDB
- Process ContextRecords from block
- Update sync status

---

### 5.2 OnBlockOrphaned
**File:** `internal/blockchain/blockchain_struct.go:344-347`

**Current:** Returns "not implemented" error

**Implement:**
- Rollback capability changes from orphaned block
- Restore previous state

---

### 5.3 Get (ChromemManager)
**File:** `internal/blockchain/blockchain_struct.go:348-354`

**Current:** Returns "not implemented" error

**Implement:**
- Query ChromemDB collections
- Return results with proper error handling

---

## Phase 6: Fully Implement NFTManager Methods

### 6.1 GetNFTsByOwner
**File:** `internal/blockchain/blockchain_struct.go:400-402`

**Current:** Returns "not implemented" error

**Implement:** Query ChromemDB NFT collection by owner address

---

### 6.2 CreateNFT
**File:** `internal/blockchain/blockchain_struct.go:404-406`

**Current:** Returns "not implemented" error

**Implement:**
- Generate unique NFT ID
- Store in ChromemDB
- Create initial ContextRecord

---

### 6.3 AttachCapability
**File:** `internal/blockchain/blockchain_struct.go:408-410`

**Current:** Returns "not implemented" error

**Implement:**
- Verify NFT and capability exist
- Create attachment record
- Update NFT capabilities list

---

## Phase 7: Remove Agent Manager Stubs

### 7.1 LevelDB Methods
**File:** `internal/agent/agent_manager.go:127-145`

**Current:**
```go
func (db *LevelDB) Get(key []byte) ([]byte, error) {
    return nil, fmt.Errorf("leveldb get not implemented")
}
```

**Action:** Either:
- Remove these methods (unused)
- Implement using actual LevelDB from `internal/database/`

---

### 7.2 LoadMasterWallet
**File:** `internal/agent/agent_manager.go:71-74`

**Current:** Returns "not implemented" error

**Action:** Either remove or implement using `internal/wallet/`

---

### 7.3 DiscoveryManager Stubs
**File:** `internal/agent/agent_manager.go:178-186`

**Current:** Returns "not implemented" errors

**Action:** Either remove or implement using actual DHT

---

## Phase 8: Clean Up Documentation

### 8.1 Delete Stale Documentation
- `pending_implementation.md`
- `knirvchain_gap_report.md`

### 8.2 Remove "placeholder" Comments
Search and remove:
- `// placeholder` comments
- `"placeholder"` string literals
- Debug comments like `// ---- TEMPORARY DEBUGGING ----`

---

## Implementation Order

1. **Phase 1** (Delete unused modules) - Highest impact, lowest risk
2. **Phase 2** (Remove placeholder types) - Easy deletions
3. **Phase 3-6** (Implement MCP/Agent/NFT/Chromem methods) - Core functionality, careful testing needed
4. **Phase 7** (Clean up agent stubs) - Lower priority
5. **Phase 8** (Documentation cleanup) - Final step

---

## Estimated Effort

| Phase | Files Affected | Complexity |
|-------|---------------|------------|
| 1 | 10+ | Low (deletions) |
| 2 | 3 | Low (deletions) |
| 3-6 | 4 | High (new implementation) |
| 7 | 1 | Medium |
| 8 | 2 | Low |

---

## Testing Strategy

After each phase:
1. Run `go build ./...` to verify compilation
2. Run existing unit tests: `go test ./tests/unit/...`
3. Verify HTTP endpoints still respond correctly
4. Check blockchain operations (add block, add transaction)
