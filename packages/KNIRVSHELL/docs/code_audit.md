# KNIRVSHELL Code Audit Report

**Date:** 2026-04-09  
**Reference Document:** `docs/CLI_Tool_Development_Plan.md`  
**Audited Package:** `packages/KNIRVSHELL/`

---

## Executive Summary

KNIRVSHELL is a comprehensive CLI tool for the KNIRV blockchain ecosystem with 87+ Go source files. The implementation covers the majority of planned features from the development plan, including wallet management, API client with circuit breaker, transaction signing, MCP server management, operational procedures, inference service with multiple LLM providers, watchtower monitoring, agent registry, and a terminal UI.

**Overall Status: PARTIAL IMPLEMENTATION** — solid foundations with significant security, completeness, and correctness gaps.

| Category | Score | Status |
|---|---|---|
| Architecture Conformance | 85% | Mostly aligned |
| Feature Completeness | 70% | Core features present, many TODOs |
| Security | 55% | Multiple HIGH+ issues |
| Code Quality | 65% | Inconsistent patterns |
| Production Readiness | ❌ | Not ready |

**Pre-Production Blockers:**
1. CLI password/private-key flags expose secrets in process list
2. `SignMessage` returns a mock signature — transactions cannot be submitted
3. Signature verification silently truncates the signature (off-by-one slice bug)
4. No input sanitization on file paths (directory traversal risk)
5. Module name mismatch (`go.mod` vs imports) — package may not compile cleanly

---

## Architecture Conformance

### Directory Structure — MOSTLY ALIGNED

```
✅ cmd/              Cobra CLI commands (19 files)
✅ core/             Core business logic (13 files; more than planned 8)
✅ pkg/inference/    Advanced inference engine (7 files)
✅ config/           Configuration management (2 files)
✅ ui/               Terminal UI with Bubbletea
✅ internal/
   ✅ runner/        Execution backends (local, docker, ssh)
   ✅ watchtower/    Monitoring and incident detection
   ✅ staging/       File staging utilities
   ✅ registry/      Agent registry
   ✅ incident/      Incident management
✅ test/             Unit, integration, e2e, benchmark suites
```

**Gaps vs. Plan:**
- `ui/` contains `screens/` and `components/` but not `views/` as specified in the plan
- `util/` directory (crypto, validation, deployment helpers) is absent; utilities are scattered across `cmd/utils.go` and `core/`
- `core/` has 5 additional client files beyond plan (`KNIRVRoot`, `KNIRVGateway`, `KNIRVGraph`, `KNIRVServer`, `KNIRVOracled`)

---

## Feature Completeness

| Feature | Status | Notes |
|---|---|---|
| Wallet: key generation | ✅ 100% | secp256k1 via go-ethereum/crypto |
| Wallet: AES-256-GCM encryption | ✅ 100% | Nonce + MAC implemented correctly |
| Wallet: Argon2id KDF | ✅ 100% | 64MB / 3 iterations / 4 parallelism |
| Wallet: new / import / list / export | 80% | Commands present; `SignMessage` is a mock |
| API client with retries + circuit breaker | 90% | Retry/backoff implemented; some endpoints stubbed |
| Transaction signing (hash) | 80% | JSON serialization instead of Protobuf (non-deterministic) |
| MCP Server Manager | 60% | Config CRUD works; deployment/discovery logic absent |
| Op Procedure Manager | 50% | Config CRUD works; step execution is TODO (`op_procedure_manager.go:206`) |
| Capability registration | 40% | Command structure exists; file reference + API calls incomplete |
| MCP server registration (extrapolation) | 30% | Command scaffolded, not implemented |
| Capability invocation | 20% | Command exists, implementation incomplete |
| AI-powered plugin generation | 20% | Command exists, no implementation |
| Terminal UI (Bubbletea) | 70% | Basic screens present; interactive state flows missing |
| Inference service (multi-provider) | 80% | Cerebras / Gemini / DeepSeek providers complete |
| Context manager | 80% | Chunking strategies present; edge-case coverage unclear |
| Conversation memory | 70% | Structure present; management logic incomplete |
| SSE client | 70% | Connection management present; reconnection / error recovery missing |
| WebSocket support | 10% | Multiple TODO stubs in all `knirv*_client.go` files |
| Watchtower monitoring | 80% | Signature-based detection, ring buffer, Obsidian integration |
| Agent runner: local | 90% | `exec.Command` with context, proper cleanup |
| Agent runner: docker | 70% | Started; some logic missing |
| Agent runner: ssh | 70% | Started; some logic missing |
| Agent registry | ✅ 100% | YAML persistence, UPSERT, full CRUD |
| File staging | ✅ 100% | SHA256 hash, temp storage |
| REPL / shell mode | ✅ 100% | readline history, tab completion, clear/exit |
| XION wallet integration | 40% | Structure defined; most methods are TODO stubs |
| NRN token manager | 50% | Balance tracking present; transfer/submission logic TODO |
| Incident management | 80% | Vault integration and ring buffer working |

---

## New Features Beyond Plan

| Feature | Location | Notes |
|---|---|---|
| KNIRVSHELL REPL | `main.go:112-193` | Interactive shell with readline; not in plan but well-implemented |
| Watchtower system | `internal/watchtower/` | Signature-based anomaly detection, Obsidian vault integration |
| Agent registry with YAML | `internal/registry/registry.go` | Tags, timestamps, multi-runner support |
| Multiple service clients | `core/knirv*_client.go` | KNIRVRoot, KNIRVGateway, KNIRVGraph, KNIRVServer, KNIRVOracled |
| File staging system | `internal/staging/staging.go` | SHA256 content hashing |
| XION wallet adapter | `core/xion_wallet_manager.go` | Cosmos/XION chain integration (partially implemented) |
| Health monitor | `core/health_monitor.go` | Periodic service health checking |

---

## Security Findings

### CRITICAL

**1. Passwords and private keys passed via CLI flags**
- Files: `cmd/wallet.go:53`, `cmd/wallet.go:97`, `cmd/mcp/capability.go:140`
- `--password` and `--private-key` flags are visible in `ps aux`, shell history, and logs
- Fix: Accept only via environment variable or interactive prompt; reject if provided as flag

**2. Mock wallet signing in production path**
- File: `core/wallet_manager.go:99-100`
- `SignMessage` returns `fmt.Sprintf("mock_signature_%s_%x", ...)` — all submitted transactions will fail silently
- Fix: Implement proper ECDSA signing before any integration testing

**3. No input sanitization on file paths**
- Files: `core/file_manager.go`, `cmd/mcp/capability.go`
- User-supplied file paths are not validated against base directory; directory traversal is possible
- Fix: `filepath.Clean` + verify path is within allowed base directory

**4. JSON unmarshal of user-provided descriptor strings without schema validation**
- Files: `cmd/agent.go`, `cmd/mcp/capability.go`
- Arbitrary JSON from CLI args unmarshaled into structs without schema guard
- Fix: Validate against JSON schema before unmarshal

### HIGH

**5. Signature verification off-by-one truncation**
- File: `core/signer.go:138`
- Code: `crypto.VerifySignature(publicKeyBytes, hashBytes, signature[:len(signature)-1])`
- Removes last byte of every signature — valid signatures always fail verification
- Fix: Use the full `signature` slice

**6. Circuit breaker state transition race condition**
- File: `core/api_client.go:260-266`
- Lock is released between state check and update — TOCTOU vulnerability under concurrent load
- Fix: Hold lock across full state transition

**7. SSE response body leak on error**
- File: `core/sse_client.go:54-116`
- `resp.Body.Close()` is deferred but not immediately after successful response; error branches may skip it
- Fix: `defer resp.Body.Close()` immediately after `http.Get` succeeds

**8. Wallet file permissions not re-verified on load**
- File: `core/wallet_manager.go:225`
- File is created with 0600 (correct) but permissions are not checked at read time
- Fix: Stat the file before loading and reject if not 0600

### MEDIUM

**9. Non-deterministic transaction hash (JSON vs Protobuf)**
- File: `core/signer.go:61-67`
- Plan specifies Protobuf for canonical serialization; code uses a `map[string]interface{}` marshaled to JSON
- Go maps have non-deterministic iteration order — hash varies between runs
- Fix: Use a struct with fixed field order, or implement Protobuf serialization

**10. Hardcoded model names in inference service**
- File: `pkg/inference/inference_service.go:76-82`
- Model names, API keys, and base URLs hardcoded in `Start()` instead of loaded from config
- Fix: Move all values to `config/defaults.go` and load via Viper

**11. `--no-password` flag creates unencrypted wallets with no environment guard**
- File: `cmd/wallet.go:64-65`
- No check for development vs. production mode; any user can create unencrypted wallets
- Fix: Gate behind a `KNIRV_DEV_MODE=true` environment variable check

**12. No rate limiting on API calls**
- File: `core/api_client.go`
- Retry logic with exponential backoff exists but no global rate limiter — retry storms possible
- Fix: Add token bucket rate limiter

### LOW

**13. Errors silently discarded in Cobra `Run` handlers**
- Multiple files: `cmd/*.go`
- `Run:` used instead of `RunE:` in several commands; errors are lost
- Also: `log.Fatal` inside handlers kills the REPL process (see CALIBER_LEARNINGS.md)
- Fix: Use `RunE` throughout; return errors instead of calling `log.Fatal`

**14. Private key bytes not zeroed after use**
- Files: `core/wallet_manager.go`, `core/signer.go`
- `crypto.FromECDSA()` bytes remain in memory after use
- Fix: Explicitly zero the byte slice after the private key is no longer needed

**15. Nil pointer inconsistency in XION wallet**
- File: `core/xion_wallet_manager.go:152-155`
- `metaAccount` nil check present in some methods, absent in others
- Fix: Add consistent nil check or initialize in constructor

**16. 20+ TODO comments in critical paths**
- Procedure execution engine, XION integration, WebSocket support all have inline TODO markers
- These should be tracked as issues, not left as dead code

---

## Code Quality Issues

### Inconsistent error handling
- Some flag reads use `_, _ = cmd.Flags().GetString(...)` — silent failures
- Mix of `fmt.Errorf("%w", err)` and raw string concatenation
- Recommendation: enforce `%w` wrapping consistently

### Concurrency hazards
1. `core/sse_client.go:150` — iterates over `connections` map while deleting from it in `DisconnectAll`
2. `core/api_client.go:260-266` — lock released between circuit breaker check and state update
3. `pkg/inference/inference_service.go:150` — incorrect else block logic noted in source comment

### Large functions
- Wallet command handlers (`cmd/wallet.go`) are 50+ lines each
- Recommendation: extract sub-steps into named helpers for testability

### Logging
- Positive: logrus structured logging is present throughout
- Issue: some INFO-level logs belong at DEBUG (e.g., per-request API logs)
- Issue: API response bodies logged in full — may leak credentials

---

## Missing Tests / Test Coverage Gaps

| Area | Gap |
|---|---|
| Wallet crypto | No roundtrip test: encrypt → decrypt → compare |
| Wallet crypto | No test: decrypt with wrong password must return error |
| Transaction signing | No known test vectors for canonical hash |
| Signature verification | No test demonstrating the truncation bug |
| API client | Circuit breaker transitions (closed → open → half-open) untested |
| API client | Retry logic across different HTTP status codes |
| Procedure execution | Step execution engine missing — no tests possible yet |
| Inference service | Provider fallback routing logic |
| Context manager | Chunking with overlapping windows, edge-case input sizes |
| Runner (docker/ssh) | Integration tests with live containers / SSH targets |
| Security | No path traversal tests; no fuzz tests for JSON unmarshaling |

---

## Recommendations (Prioritized)

### Critical — Do Before Any Integration Testing

1. **Remove `--password` / `--private-key` CLI flags** — accept secrets only via env var or interactive prompt  
   Files: `cmd/wallet.go`, `cmd/mcp/*.go`, `cmd/utils.go` | ~3 hours

2. **Implement `SignMessage` with real ECDSA**  
   File: `core/wallet_manager.go:99` | ~1 hour

3. **Fix signature verification truncation**  
   File: `core/signer.go:138` — remove `[:len(signature)-1]` | ~30 minutes

4. **Add path sanitization to all file operations**  
   Files: `core/file_manager.go`, `cmd/mcp/capability.go` | ~4 hours

5. **Add JSON schema validation before unmarshal**  
   Files: `cmd/agent.go`, `cmd/mcp/capability.go` | ~4 hours

### High — Before Feature Completion

6. **Fix canonical transaction hashing (Protobuf or deterministic struct)**  
   File: `core/signer.go` | ~6 hours

7. **Implement procedure step execution engine**  
   File: `core/op_procedure_manager.go:206` | ~10 hours

8. **Complete XION wallet integration** (replace all TODOs)  
   File: `core/xion_wallet_manager.go` | ~16 hours

9. **Complete MCP capability registration flow**  
   Files: `cmd/mcp/capability.go`, `core/file_manager.go` | ~10 hours

10. **Fix SSE connection leak**  
    File: `core/sse_client.go` | ~2 hours

### Medium — For Production Readiness

11. Convert all Cobra `Run:` handlers to `RunE:` — eliminate `log.Fatal`
12. Add rate limiter to API client
13. Move hardcoded inference values to config
14. Add comprehensive test suite (~30 hours)
15. Complete Docker and SSH agent runner implementations

### Low — Code Quality

16. Fix map-iteration race in `core/sse_client.go:DisconnectAll`
17. Fix circuit breaker TOCTOU in `core/api_client.go`
18. Zero private key bytes after use in wallet/signer
19. Refactor large wallet command handlers
20. Implement WebSocket subscriptions in service clients

---

## Implementation Status Summary

| Area | Planned | Implemented | Gap |
|---|---|---|---|
| CLI framework | Cobra | ✅ | — |
| REPL / shell mode | No | ✅ | New feature, well done |
| Wallet management | Full | 80% | Real signing, XION |
| Crypto primitives | Full | ✅ | — |
| API client | Full | 90% | Minor endpoint gaps |
| Transaction signing | Protobuf | 80% | Hash format, verification bug |
| Terminal UI | Full | 70% | Interactive workflows |
| Inference service | Full | 80% | Config hardcoding |
| MCP server manager | Full | 60% | Deployment, discovery |
| Operational procedures | Full | 50% | Step execution engine |
| Capability registration | Full | 40% | File refs, API calls |
| Agent runners | local+docker+ssh | 80% | docker/ssh incomplete |
| Watchtower | Not planned | 80% | Good addition |
| XION integration | Not planned | 40% | Mostly stubbed |

**Estimated effort for critical fixes:** 15–20 hours  
**Estimated effort for full production readiness:** 60–80 hours
