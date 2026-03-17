# KNIRVCHAIN Gap Report
**Reference:** `~/KNIRV/LAB/KNIRVCHAIN_ROOT/` (monolithic, working reference)
**Target:** `packages/KNIRVCHAIN/` (modular, current implementation)
**Date:** 2026-03-17

---

## Executive Summary

`packages/KNIRVCHAIN` is architecturally ahead of `KNIRVCHAIN_ROOT` in most areas — it has role-based node management, LLM inference, ChromemDB, NFT support, PoAuD, agent mode, and a far richer API surface. However, several functional items from the ROOT version are either missing, commented out, broken, or replaced without a working substitute. This report enumerates every gap in priority order so implementation can proceed without over-engineering.

---

## 1. GUI — Fyne Replaced by Web GUI (Incomplete Replacement)

### ROOT behaviour
`gui.go` (39 KB, ~1,337 lines) contains a full Fyne desktop GUI with:
- `RunGUI()` — entry point called from `main()` when `--gui` flag is set
- `newBoundWriter()` / `boundWriter` — streams log output into the GUI in real time
- Dashboard tab: chain height, peer count, mining status, latest block hash
- Blockchain tab: scrollable block list with details
- Transactions tab: pending pool display
- Network tab: peer list, per-peer capability browser
- Context Records tab: MCP context record viewer
- Payment Processor tab: Stripe webhook status, token disbursement history
- Creator Settings tab: payment processor configuration
- Mining tab: start/stop mining controls

### Current state
- Fyne dependency (`fyne.io/fyne/v2 v2.7.1`) is still in `go.mod` but **no `gui.go` exists** in the main package
- `--gui` flag is flagged `DEPRECATED: GUI functionality has been removed`
- `guiNodeConfig` pre-initialization still runs in `main()` (lines 1625–1668) but **never calls `RunGUI()`** — it only pre-initializes DB/discovery/blockchain for the GUI node, then starts `startNodeWithComponents()` headlessly
- A web-based alternative GUI (Next.js) is planned per `altgui_Integration_Strategy.md` but the `internal/embedded/nodejs/webgui/` path **does not exist** — referenced by `main.go:2069` but absent on disk
- `newBoundWriter()` / `boundWriter` are completely absent — log output goes only to stderr + file

### Gap severity: **HIGH**
The `--gui` flag does nothing useful. Users who rely on a visual interface have no working replacement.

### Required actions
- **Option A (Web GUI):** Create `internal/embedded/nodejs/webgui/webGUI/backend.config` and the Next.js web app the code already expects to configure (line 2069–2110), or adjust the path to point at the existing KNIRVSYNC/KNIRVGATEWAY web app
- **Option B (Remove dead code):** Strip all `guiNodeConfig` pre-initialization and `--gui` flag machinery from `main()` if GUI is permanently headless — avoids confusion and dead code paths

---

## 2. `--reflect` Flag — Commented Out

### ROOT behaviour
```go
var reflectFlags []string
flag.Func("reflect", "Add a reflection URL (can be specified multiple times)", func(url string) error {
    reflectFlags = append(reflectFlags, url)
    return nil
})
// ...
if len(reflectFlags) > 0 {
    cfg.ReflectionURLs = reflectFlags
}
```
Operators can pass `--reflect http://peer:5000` one or more times to manually seed reflection URLs.

### Current state
Lines 952–955 and 1226–1227 in `main.go` have this block **commented out** with the note "Viper loads ReflectionURLs from config file". Viper does load them from config, but there is **no CLI override path**. Operators cannot inject reflection URLs at startup without editing the config file.

### Gap severity: **MEDIUM**
Testnet and multi-node startup scripts depend on this pattern.

### Required action
Uncomment and restore the `--reflect` flag with Viper as fallback, not replacement.

---

## 3. `--creator` Flag → Payment Processor Activation

### ROOT behaviour
```go
creator := flag.Bool("creator", false, "Run as a creator node with payment processor enabled")
// ...
if *creator {
    cfg.PaymentProcessor.Enabled = true
    LoadPaymentProcessorConfig(&cfg.PaymentProcessor)
}
```
The `--creator` flag activates the Stripe payment processor inline and populates `cfg.PaymentProcessor` from environment variables.

### Current state
- No `--creator` flag exists in `packages/KNIRVCHAIN`
- Payment processor activation is gated behind `cfg.IsRoot && cfg.PaymentProcessor.Enabled` (lines 1837–1845)
- `LoadPaymentProcessorConfig()` exists in `internal/wallet/payment_processor.go` and is functionally identical
- **However,** there is no CLI mechanism to enable the payment processor without pre-editing the config file or using the `--root` role

### Gap severity: **MEDIUM**
Root-node operators cannot activate the payment processor via CLI flag.

### Required action
Add `--creator` (or `--enable-payment-processor`) flag that sets `cfg.PaymentProcessor.Enabled = true` and calls `LoadPaymentProcessorConfig()`, matching ROOT behaviour.

---

## 4. `generateAndSaveWallet()` Helper — Missing from `main.go`

### ROOT behaviour
```go
func generateAndSaveWallet(ws *WalletServer, walletPath string, cfg *config.Config, loadedConfigPath string) {
    // Generates a new wallet, saves to walletPath, updates cfg.MinersAddress, saves config
}
```
Called from `main()` when `cfg.MinersAddress == ""` and no `wallet.dat` exists.

### Current state
The current `main()` (lines 1100–1117) handles this inline via `wm.LoadWallet()` / `wm.CreateWallet()` through the `WalletManager` abstraction. This is functionally equivalent but the error paths differ — in the current code if `wm.CreateWallet()` fails, the node continues with a warning rather than fatal-exiting. The ROOT fatals.

### Gap severity: **LOW** (functionally covered, error handling differs)

### Required action
Verify the `WalletManager.CreateWallet()` path in `internal/wallet/wallet_manager.go` covers all cases the ROOT's `generateAndSaveWallet()` did. Specifically confirm it:
- Writes the wallet file to `walletPath`
- Updates `cfg.MinersAddress`
- Saves the config via `config.SaveConfig()`

---

## 5. Wallet Type: `Wallet` → `WalletImpl` Interface Break

### ROOT behaviour
All wallet-related types are `*Wallet` throughout the codebase. `WalletServer`, `PaymentProcessor`, `BlockchainServer` all accept `*Wallet` directly.

### Current state
`internal/wallet/wallet.go` defines `*WalletImpl`, with a `Wallet` interface in `interfaces.go`. However:
- `internal/wallet/wallet_server.go` line ~68 calls `NewWalletServer(port, backendURL)` — no wallet parameter
- `internal/wallet/payment_processor.go:31` takes `*WalletImpl` (concrete type) instead of the `Wallet` interface
- Tests in `tests/unit/` have compiler errors (`cannot use *WalletImpl as Wallet`) — visible in the pre-existing diagnostics
- `internal/installation/install.go` `loadWalletFromFile` (missing from current's wallet.go public surface)

### Gap severity: **HIGH** (test failures, potential runtime type confusion)

### Required action
1. Standardize `payment_processor.go` to accept the `Wallet` interface, not `*WalletImpl`
2. Fix test files that have `*WalletImpl` → `Wallet` assignment errors (pre-existing diagnostics)
3. Ensure `loadWalletFromFile` is accessible from the installation package (currently it's private in `wallet_manager.go`)

---

## 6. `self_consensus_manager.go` — Unexported Method Names Changed

### ROOT behaviour
```go
func (cm *ConsensusManager) getUpdateRequired() bool  // unexported
func (cm *ConsensusManager) getMiningLockState() bool // unexported
```

### Current state
```go
func (cm *ConsensusManager) GetUpdateRequired() bool   // exported
func (cm *ConsensusManager) GetMiningLockState() bool  // exported
func (cm *ConsensusManager) GetStatus() string         // NEW
func (cm *ConsensusManager) GetPeerCount() int         // NEW
func (cm *ConsensusManager) IsSyncing() bool           // NEW
```
The current version is a **superset** — exported names + new methods. No functionality is lost. This is an improvement.

### Gap severity: **NONE** (enhanced, not degraded)

---

## 7. `mcp_processor.go` — API Changed, Old Methods Absent

### ROOT behaviour — key public methods
```go
func (mcp *MCPProcessor) ProcessMCPRegisterCapability(transaction *Transaction) error
func (mcp *MCPProcessor) ProcessMCPInvokeCapability(capabilityID, capabilityType string, data []byte, ownerAddress string, fee uint64, txHash string) error
func (mcp *MCPProcessor) ApplyMCPTransactionEffects(tx *Transaction, accounts map[string]*big.Int) error
```

### Current state (`internal/mcp/mcp_processor.go`)
```go
func (mcp *MCPProcessorImpl) ProcessMCPTransaction(transaction *Transaction) error  // unified dispatcher
func (mcp *MCPProcessorImpl) RegisterCapability(transaction *Transaction) error
func (mcp *MCPProcessorImpl) InvokeCapability(transaction *Transaction) error
func (mcp *MCPProcessorImpl) GetCapability(id string) (interface{}, error)
// validateCapabilityDeletion() — NEW
// processCapabilityDeletion() — NEW
```
`ApplyMCPTransactionEffects()` **does not exist** in the current `MCPProcessorImpl`. The blockchain's transaction application loop calls this in ROOT's `blockchain_struct.go` — verify the current `blockchain_struct.go` does not still reference it.

Also: `NewMCPProcessorWithAgentManager()` is new and takes an `AgentManager` and `Wallet` — ROOT's `NewMCPProcessor()` only takes a `*LevelDB`. The constructor signature mismatch must be reconciled at call sites in `main.go`.

### Gap severity: **MEDIUM**

### Required action
1. Verify `blockchain_struct.go` does not call `ApplyMCPTransactionEffects()` — if it does, add it to `MCPProcessorImpl` or alias via `ProcessMCPTransaction`
2. Audit all `NewMCPProcessor()` call sites in `main.go` and ensure they pass the right arguments

---

## 8. `BlockchainServer` — `Start()` Method Renamed

### ROOT behaviour
```go
func (bcs *BlockchainServer) Start() error  // single start method
```

### Current state
```go
func (bcs *BlockchainServer) Prepare() (uint64, error)     // port binding + setup
func (bcs *BlockchainServer) StartListenAndServe() error   // actual HTTP serve
```
The two-phase startup is correct for port-conflict fallback. However, callers in `startNode()` / `startNodeWithComponents()` must use `Prepare()` + `StartListenAndServe()` — confirm `main.go` does this consistently and does not have any dangling `bcs.Start()` calls.

### Gap severity: **LOW** (likely already handled)

### Required action
Grep for `bcs.Start()` in current codebase and confirm zero occurrences.

---

## 9. `--reflect` Flag's Counterpart: `ReflectionURLs` Not Applied from CLI

See item 2 above. Related: in ROOT, `reflectFlags` were also passed into network mode to set the reflection node's upstream. In the current codebase the reflection node's `ReflectionURLs` is **hardcoded** to `localhost` (lines 1568, 1770). This prevents non-localhost reflection topologies from being configured via CLI.

### Gap severity: **MEDIUM**

---

## 10. `install.go` — `configureWindowsService` Signature Differs

### ROOT
```go
func ConfigureSystemService(configPath string, cfg *config.Config, serviceType string) error
func configureWindowsService(configPath string, cfg *config.Config, serviceType string) error
```

### Current
```go
func ConfigureSystemService(configPath string, cfg *config.Config, role config.Role) error
func configureWindowsService(configPath string, cfg *config.Config, role config.Role) error
```
The `serviceType string` has been replaced by `role config.Role`. This is an improvement, but any deployment scripts or test code that passed an explicit `serviceType` string will break.

### Gap severity: **LOW** (internal API change, no external consumers known)

---

## 11. URI Generation — `calculateChainID()` Present in ROOT, Status Unknown

### ROOT `uri_generation.go`
```go
func calculateChainID() string         // derives ChainID from genesis hash
func generateChainURI(metadata ChainMetadata) string
```

### Current `internal/uri/uri_generation.go`
```go
func GenerateResourceURI(...)          // present
func ParseResourceURI(...)             // present
func GenerateMCPCapabilityURI(...)     // present
// calculateChainID() — verify presence
```

### Required action
Confirm `calculateChainID()` exists in `internal/uri/uri_generation.go` or that its logic has been absorbed into config initialization.

---

## 12. `discovery_manager.go` — `DiscoverPublicAddress()` Signature Changed

### ROOT
```go
func DiscoverPublicAddress() (string, string, error)  // returns ip, port-string, err
```

### Current
```go
func DiscoverPublicAddress(registryHTTPBaseURL string, logger *logrus.Logger) (string, int, error)  // int port
```
The addition of `registryHTTPBaseURL` and `*logrus.Logger` parameters means any call site using the ROOT signature will fail to compile. Also port is now `int` not `string`. Verify all call sites in `main.go` pass the correct arguments.

### Gap severity: **MEDIUM** (compile error if any ROOT-style caller remains)

---

## 13. `bounded_writer` / Log Binding — No Replacement

### ROOT
`newBoundWriter(binding.String, maxLines int)` bridges the Go `log` package output into the Fyne GUI's real-time log display.

### Current
No log binding exists. Log output goes to `stderr` + `logs/KNIRVCHAIN.log`. If a web GUI is added (item 1), a WebSocket-based log streamer should be implemented to replace this functionality.

### Gap severity: **LOW** (only relevant if GUI is restored)

---

## 14. `payment_processor_integration.go` — `LoadPaymentProcessorConfig()` Call Site

### ROOT `payment_processor_integration.go`
```go
// Thin wrapper that calls LoadPaymentProcessorConfig from the payment_processor.go
// and connects it with the creator-mode flag activation
```

### Current
`LoadPaymentProcessorConfig()` exists in `internal/wallet/payment_processor.go` (functionally identical). However, the ROOT file exposed this via a dedicated integration file to keep `main.go` clean. The current inlines it.

### Gap severity: **NONE** (cosmetic/organizational difference only)

---

## 15. Test Suite — Pre-existing Compile Errors

The following test files have pre-existing compiler errors unrelated to the updater fix:

| File | Error |
|------|-------|
| `tests/unit/role_integration_test.go` | `cannot use *wallet.WalletImpl as Wallet` (×6) |
| `tests/unit/test_helpers.go` | `cannot use *WalletImpl as Wallet` |

These must be fixed before the test suite can run. Root cause: `Wallet` interface requires methods not present on `WalletImpl`, or `WalletImpl` is not registered as implementing `Wallet`.

### Gap severity: **HIGH** (blocks all unit testing)

### Required action
In `internal/wallet/interfaces.go`, ensure the `Wallet` interface only declares methods that `*WalletImpl` actually implements. Alternatively, change the test helper to use `*WalletImpl` directly where the interface isn't needed.

---

## 16. `agent/ChromemManager` — Stub Methods (Added Today)

`chromemdb_stubs.go` was added as part of the updater fix to resolve the pre-existing `ChromemManager` undefined-type build error. All 18 stub methods (`StoreAgent`, `GetAgent`, `StoreBadge`, etc.) return `nil` or not-found errors.

### Gap severity: **HIGH** (agent functionality non-operational)

### Required action
Implement real ChromemDB-backed storage in each stub. The `transactionCollection` and `contextRecordCollection` fields need to be populated via a constructor. A `NewAgentChromemManager(db *chromem.DB) *ChromemManager` constructor should be added.

---

## 17. `agent/interfaces.go` — `chromemCollection` Interface Fields Not Wired

`ChromemManager.transactionCollection` and `ChromemManager.contextRecordCollection` are declared as `*chromem.Collection` but there is no constructor that sets them. The five methods in `agent_manager.go` that call these collections (`StoreNFT`, `GetNFT`, `GetNFTsByOwner`, `StoreNFTCapabilityAttachment`, `GetNFTCapabilityAttachments`) will panic on nil pointer at runtime.

### Gap severity: **HIGH** (runtime nil-pointer panic)

### Required action
Add `NewAgentChromemManager(db *chromem.DB, collectionName string) (*ChromemManager, error)` that initializes both collections via `db.GetOrCreateCollection()`.

---

## 18. Summary Table

| # | Feature | ROOT | Current | Severity | Action |
|---|---------|------|---------|----------|--------|
| 1 | Fyne GUI (`RunGUI`) | ✅ Working | ❌ Removed, no replacement | HIGH | Implement web GUI or strip dead code |
| 2 | `--reflect` flag | ✅ Working | ❌ Commented out | MEDIUM | Uncomment and restore |
| 3 | `--creator` flag | ✅ Working | ❌ Missing | MEDIUM | Add flag → `PaymentProcessor.Enabled` |
| 4 | `generateAndSaveWallet()` | ✅ Working | ⚠️ Inline equivalent | LOW | Verify WalletManager covers all cases |
| 5 | `Wallet` vs `WalletImpl` | `*Wallet` | `*WalletImpl` + interface | HIGH | Fix test compile errors + interface alignment |
| 6 | `ConsensusManager` methods | Unexported | Exported + new | NONE | No action needed |
| 7 | `ApplyMCPTransactionEffects` | ✅ Present | ❌ Missing | MEDIUM | Verify blockchain_struct.go call sites |
| 8 | `BlockchainServer.Start()` | `Start()` | `Prepare()+StartListenAndServe()` | LOW | Verify no dangling `Start()` calls |
| 9 | Network-mode reflection URLs | CLI-settable | Hardcoded localhost | MEDIUM | Restore CLI override path |
| 10 | `ConfigureSystemService` sig | `serviceType string` | `role config.Role` | LOW | Internal only, no action |
| 11 | `calculateChainID()` | ✅ Present | Unknown | LOW | Verify in `internal/uri/` |
| 12 | `DiscoverPublicAddress` sig | 2 returns | 3 params + int port | MEDIUM | Verify all call sites compile |
| 13 | `boundWriter` / log binding | ✅ Present | ❌ Absent | LOW | Only needed if GUI restored |
| 14 | `LoadPaymentProcessorConfig` | Separate file | Inline | NONE | No action needed |
| 15 | Unit test compile errors | N/A | ❌ Fail to compile | HIGH | Fix `WalletImpl`/`Wallet` interface |
| 16 | `ChromemManager` stub methods | N/A | ⚠️ Stubs only | HIGH | Implement real ChromemDB ops |
| 17 | `ChromemManager` constructor | N/A | ❌ Missing | HIGH | Add `NewAgentChromemManager()` |

---

## 19. New Features in `packages/KNIRVCHAIN` NOT in ROOT (no action needed, informational)

The following exist in `packages/KNIRVCHAIN` and have no ROOT equivalent — they represent forward progress:

| Feature | Location | Notes |
|---------|----------|-------|
| LLM Inference Pipeline | `internal/inference/` | Cerebras, DeepSeek, Gemini providers |
| Agent Mode | `internal/agent/`, `--agent` flag | Autonomous inference + delegation |
| ChromemDB vector store | `internal/database/chromemDB_manager.go` | Semantic search for capabilities |
| NFT endpoints | `blockchain_server.go` handleNFTList/Upload/etc | NFT capability attachments |
| PoAuD (Proof of Authorship) | `blockchain_server.go` EnablePoAuD/DisablePoAuD | New consensus mechanism |
| Resource Capability Groups | `blockchain_server.go` handleAddResourceCapability etc | Grouped MCP resource management |
| Role-based architecture | `config/` | Root, Bootnode, Peer, Client, Developer, Agent |
| Xion integration | `internal/integrations/xion/` | Cross-chain payment gateway |
| Data Engine | `internal/dataengine/` | Kafka-backed event streaming |
| Failover Manager | `blockchain_server.go` NewBlockchainServerWithFailover | High-availability node operation |
| REST API for DataEngine | `internal/dataengine/rest_api.go` | Separate API surface for data ops |
| PoAuD validation | `internal/crypto/validate_poaud.go` | Standalone validation binary |
| Skill/Capability/Property mining | `internal/mining/` | Node transformation flows |
| Graph store | `internal/graph/` | Knowledge graph for AI errors/solutions |

---

## 20. Recommended Implementation Order

1. **Fix `WalletImpl`/`Wallet` interface alignment** (unblocks all tests)
2. **Add `NewAgentChromemManager()` constructor** (unblocks agent functionality, prevents nil panics)
3. **Implement `ChromemManager` stub methods** with real ChromemDB calls
4. **Restore `--reflect` flag** (critical for multi-node testnet)
5. **Add `--creator` flag** (payment processor activation)
6. **Verify `ApplyMCPTransactionEffects` call sites** in `blockchain_struct.go`
7. **Verify `DiscoverPublicAddress` call sites** compile with new signature
8. **Resolve GUI path** (decide: implement web GUI or remove dead pre-initialization code)
9. **Run full test suite** after items 1–3

---

*This report reflects the state of `packages/KNIRVCHAIN` as of commit `c6037d44` + the updater implementation added on 2026-03-16.*
