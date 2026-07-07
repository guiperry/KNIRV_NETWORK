# NRN Integration Refactor Plan

> **Goal:** Remove all ghost oracle endpoints from KNIRVGRAPH, simplify skill commitment to a single action, and equip ErrorNodes with NRN bounties.

**Architecture:** 
1. Strip everything from `nrn_integration.go` that talks to KNIRVORACLE (sync, confirm, distribute)
2. Replace the three ghost HTTP handlers on the KNIRVGRAPH RPC with one `commitSkill` handler
3. Add `NRNBounty` field to `ErrorNode` in `nrv/types.go`
4. Wire the bounty through `proof_of_solution.go` — rewards stay local (no HTTP calls)

**Files changed:**
- packages/KNIRVGRAPH/internal/economics/nrn_integration.go
- packages/KNIRVGRAPH/internal/economics/proof_of_solution.go
- packages/KNIRVGRAPH/internal/nrv/types.go
- packages/KNIRVGRAPH/internal/network/rpc.go

---

### Task 1: Add NRNBounty to ErrorNode

**Objective:** Add NRN bounty field to ErrorNode struct so error nodes can carry native NRN token rewards.

**Files:**
- Modify: packages/KNIRVGRAPH/internal/nrv/types.go:33-41

Add `NRNBounty string` field to ErrorNode struct (string for big.Int serialization).

### Task 2: Strip ghost functions from nrn_integration.go

**Objective:** Remove all functions that reach out to KNIRVORACLE ghost endpoints.

**Remove:**
- `testConnection()` (no longer pinging oracle)
- `periodicSync()` (no sync loop)
- `syncWithKNIRVRoot()` (no sync at all)
- `ProcessSkillConfirmation()` (replaced by CommitSkill in rpc.go)
- `DistributeRewards()` (rewards stay local)
- `makeRequest()` (only used by removed functions)
- `processEconomicEvents()` (placeholder loop)
- `processPendingEvents()` (empty placeholder)
- `handleEconomicEvent()` (only used by removed function)
- `updateNetworkStatistics()` (only used by removed sync)
- `SkillInvocationRequest` type (unused)
- `RewardDistributionRequest` type (unused)
- `NRVEconomicEvent` type (unused)
- `bytes` import (only used by makeRequest)
- `net/http` import (only used by makeRequest/testConnection)
- `net` import (only used by unix socket handling -- but keep if constructor still uses it)

**Keep:**
- `NRNIntegration` struct (simplified — remove `lastSync`, `rewardPool`, some fields)
- `EconomicMetrics` struct (used by GetEconomicMetrics)
- `NewNRNIntegration` constructor (simplified — no oracle URL needed, no httpClient)
- `Start()` (simplified — just log, no oracle connection)
- `GetEconomicMetrics()`, `IsEnabled()`, `SetEnabled()`
- `encoding/json` import (may still be needed)

### Task 3: Update proof_of_solution.go

**Objective:** Replace oracle-targeted reward distribution with local bounty assignment.

**Files:**
- Modify: packages/KNIRVGRAPH/internal/economics/proof_of_solution.go

**Changes:**
- `ProcessErrorNodeCreation()`: Instead of calling `DistributeRewards()`, set `NRNBounty` on the error node and do local accounting
- `ProcessSkillNodeCreation()`: Calculate reward locally and return (no HTTP call)
- `ProcessSuccessfulResolution()`: Calculate reward locally and log it
- Remove `proof_of_solution.go` dependency on oracle network calls

### Task 4: Simplify RPC route handlers

**Objective:** Replace three ghost HTTP handlers with one `commitSkill` handler, update route registration.

**Files:**
- Modify: packages/KNIRVGRAPH/internal/network/rpc.go

**Changes:**
- Remove `distributeRewards()` HTTP handler (line 656+)
- Rename `confirmSkill()` to `commitSkill()` — keeps the same POST route but does the logic locally (no oracle call)
- Remove route `/economics/skill/confirm` — replace with simpler handler or remove
- Remove route `/economics/rewards/distribute`
- Update the route registration in `NewRPCServerWithEconomics`

### Task 5: Build, test, verify

**Objective:** Everything compiles and tests pass.

**Files:**
- All modified packages

**Commands:**
```bash
cd /home/gperry/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVGRAPH
go build ./internal/economics/ && echo 'ECON OK'
go build ./internal/nrv/ && echo 'NRV OK'
go build ./internal/network/ && echo 'NETWORK OK'
go build ./internal/app/ && echo 'APP OK'
go build ./... && echo 'FULL OK'
go test ./... 2>&1 | tail -10
```
