# Testnet Migration Plan: KNIRVTESTNET → KNIRVSERVER `--testnet`

## Goal

Replace `devtools/KNIRVTESTNET` (Node.js multi-process orchestrator) with a
`--testnet` flag on `KNIRVSERVER` that starts the entire network correctly in
a single Go process. All integration tests, modp targets, and root-Makefile
testnet commands must route through the new entry point.

---

## Background / Why This Is Safe

KNIRVSERVER already embeds every component via Manager subprocesses:

| Component | Manager field | Enabled by config |
|---|---|---|
| KNIRVORACLE | `oracleManager` | `root.key` present |
| KNIRVCHAIN | `chainManager` | `cfg.Chain.Enabled` |
| KNIRVGRAPH | `graphManager` | `cfg.Graph.Enabled` |
| KNIRVGATEWAY | `gatewayManager` | `cfg.Gateway.Enabled` |
| KNIRVHASHER | `hasherManager` | `cfg.Hasher.Enabled` |
| KNIRVARENA | `arenaManager` | always |

All external ports used by `devtools/KNIRVTESTNET` are the same ports these
managers bind:

| Service | Port | Source |
|---|---|---|
| KNIRVSERVER API | :8084 | `cfg.API.Port` |
| KNIRVCHAIN | :8090 | `cfg.Chain.APIPort` |
| KNIRVGRAPH | :8082 | `cfg.Graph.APIPort` |
| KNIRVORACLE | :1317 | oracle manager |
| KNIRVGATEWAY | :8888 | `cfg.Gateway.Port` |
| KNIRVROUTER (via gateway) | :8086 | gateway routes |

The Node.js portal on `:10000` is the only thing KNIRVSERVER doesn't yet
replace. That portal is a status dashboard only; integration tests don't
depend on it directly. A lightweight `/testnet/status` JSON endpoint on
KNIRVSERVER API is a sufficient replacement.

---

## Port / Endpoint Compatibility Contract

Integration tests and modp must resolve services at the same `localhost`
addresses after migration. No port numbers change. The gateway remains
the auth + proxy entry point at `:8888`.

---

## Step-by-Step Implementation

### Step 1 — Add `--testnet` CLI flag to `main.go`

File: `packages/KNIRVSERVER/backend/cmd/backend_server/main.go`, function `run()` (line ~3279).

```diff
-    var configFile = flag.String("config", "", "Path to configuration file")
-    flag.Parse()
+    var configFile = flag.String("config", "", "Path to configuration file")
+    var testnetMode = flag.Bool("testnet", false, "Run in testnet mode with all embedded services enabled")
+    flag.Parse()
```

Then, immediately after the config is loaded (after `config.Load()`, line ~3317),
inject testnet overrides when the flag is set:

```go
if *testnetMode || os.Getenv("KNIRVSERVER_TESTNET") == "true" {
    applyTestnetOverrides(config)
}
```

Add a helper (same file, new function):

```go
func applyTestnetOverrides(cfg *config.Config) {
    cfg.Testnet = true
    cfg.Environment = "testnet"
    cfg.Chain.ChainID = "testnet"

    // Enable all embedded managers
    cfg.Chain.Enabled = true
    cfg.Graph.Enabled = true
    cfg.Gateway.Enabled = true
    cfg.Hasher.Enabled = false   // heavy; keep opt-in via config

    // Relaxed security suitable for local testnet
    cfg.Security.AuthRequired = false
    cfg.TEE.SimulationMode = true

    // Ensure chain defaults if not set by config file
    if cfg.Chain.APIPort == 0 {
        cfg.Chain.APIPort = 8090
    }
    if cfg.Graph.APIPort == 0 {
        cfg.Graph.APIPort = 8082
    }
    if cfg.Gateway.Port == 0 {
        cfg.Gateway.Port = 8888
    }
    if cfg.API.Port == 0 {
        cfg.API.Port = 8084
    }

    log.Println("[testnet] Testnet overrides applied")
}
```

The flag must also accept `--testnet` before the `--dve-ns-helper` branch so
it doesn't collide (the ns-helper check happens before `run()` so no conflict
exists).

---

### Step 2 — Create `packages/KNIRVSERVER/config/testnet.yaml`

A minimal YAML that sets testnet-appropriate values for all embedded managers,
overlaid on top of defaults. This is what `--testnet` points at when no
explicit `--config` is supplied.

```yaml
# packages/KNIRVSERVER/config/testnet.yaml
environment: testnet
testnet: true

api:
  port: 8084

database:
  path: "./data/knirvserver/testnet.db"
  use_knirvbase: false

security:
  auth_required: false
  jwt_secret: "knirv_testnet_jwt_secret_2025_secure_token_for_development_only"

tee:
  simulation_mode: true

chain:
  enabled: true
  api_port: 8090
  chain_id: testnet
  role: validator
  data_path: "./data/knirvchain"
  start_timeout: 60
  stop_timeout: 10

graph:
  enabled: true
  api_port: 8082
  data_path: "./data/knirvgraph"
  sync_interval: "30s"
  start_timeout: 60
  stop_timeout: 10

gateway:
  enabled: true
  port: 8888
  auth_secret: "knirv_testnet_jwt_secret_2025_secure_token_for_development_only"
  start_timeout: 60
  stop_timeout: 10

hasher:
  enabled: false

log:
  level: info
```

`run()` must auto-load this file when `--testnet` is set and no explicit
`--config` was provided:

```go
if *testnetMode && *configFile == "" {
    // Resolve testnet.yaml relative to the binary's working directory
    candidates := []string{
        "config/testnet.yaml",
        "packages/KNIRVSERVER/config/testnet.yaml",
        filepath.Join(filepath.Dir(os.Args[0]), "config/testnet.yaml"),
    }
    for _, p := range candidates {
        if _, err := os.Stat(p); err == nil {
            viper.SetConfigFile(p)
            log.Printf("[testnet] Loading testnet config: %s", p)
            break
        }
    }
}
```

This block goes in `run()` before the existing config-file discovery block.

---

### Step 3 — Add `/testnet/status` route to KNIRVSERVER

File: `packages/KNIRVSERVER/backend/cmd/backend_server/main.go`,
function `setupRoutes()` (line ~1932), add after the `/health` route:

```go
if s.config.Testnet {
    s.router.HandleFunc("/testnet/status", s.handleTestnetStatus).Methods("GET")
    s.router.HandleFunc("/auth/testnet-tokens", s.handleTestnetTokens).Methods("GET")
    log.Println("Testnet status + token routes registered")
}
```

Add `handleTestnetStatus` returning a JSON summary of all embedded manager
states (chain running, graph running, gateway running, oracle running).

Add `handleTestnetTokens` returning the same static tokens the old gateway
testnet endpoint served:

```json
{
  "admin":     "testnet-admin-123",
  "validator": "testnet-validator-456",
  "observer":  "testnet-observer-789"
}
```

---

### Step 4 — Update root `Makefile` testnet targets

The root Makefile currently has `testnet-start` / `testnet-stop` / `testnet-status`
pointing at EC2 infrastructure and a `testnet-tests` target delegating to
`devtools/KNIRVTESTNET`. Replace all of them:

```makefile
KNIRVSERVER_TESTNET_PID ?= /tmp/knirvserver-testnet.pid
KNIRVSERVER_BIN        ?= packages/KNIRVSERVER/backend/cmd/backend_server/main.go

.PHONY: testnet-build
testnet-build: ## Build KNIRVSERVER for testnet
	@cd packages/KNIRVSERVER && go build -o bin/knirvserver ./backend/cmd/backend_server/

.PHONY: testnet-start
testnet-start: testnet-build ## Start KNIRVSERVER in testnet mode (all services)
	@echo "Starting KNIRVSERVER --testnet ..."
	@cd packages/KNIRVSERVER && \
	    ./bin/knirvserver --testnet & echo $$! > $(KNIRVSERVER_TESTNET_PID)
	@echo "PID stored in $(KNIRVSERVER_TESTNET_PID)"
	@sleep 5
	@curl -sf http://localhost:8084/health && echo "KNIRVSERVER healthy" || echo "health check pending"

.PHONY: testnet-stop
testnet-stop: ## Stop KNIRVSERVER testnet
	@if [ -f $(KNIRVSERVER_TESTNET_PID) ]; then \
	    kill $$(cat $(KNIRVSERVER_TESTNET_PID)) 2>/dev/null || true; \
	    rm -f $(KNIRVSERVER_TESTNET_PID); \
	    echo "KNIRVSERVER testnet stopped"; \
	fi

.PHONY: testnet-status
testnet-status: ## Show KNIRVSERVER testnet status
	@curl -sf http://localhost:8084/testnet/status | python3 -m json.tool || \
	    echo "Testnet not running (try: make testnet-start)"

.PHONY: testnet-tests
testnet-tests: testnet-start ## Start testnet then run integration tests
	@sleep 10   # allow embedded managers time to start
	@cd integration-tests && go test -v -timeout 300s ./... ; \
	    RESULT=$$? ; \
	    $(MAKE) testnet-stop ; \
	    exit $$RESULT
```

Also update the `testnet-tests` entry in the help echo block.

---

### Step 5 — Update `devtools/KNIRVTESTNET/Makefile` testnet targets

The KNIRVTESTNET Makefile's `start` / `stop` / `testnet` / `testnet-tests`
targets must delegate to the root Makefile equivalents so existing muscle
memory still works:

```makefile
start: ## Start the testnet (now delegates to KNIRVSERVER --testnet)
	@echo "[KNIRVTESTNET] Delegating to KNIRVSERVER --testnet ..."
	@$(MAKE) -C ../.. testnet-start

stop:
	@$(MAKE) -C ../.. testnet-stop

testnet: start test

testnet-tests: testnet

health:
	@$(MAKE) -C ../.. testnet-status
```

Keep the `modp-*` targets unchanged — they resolve `MODP_DIR` from `../../modp`
which stays correct.

---

### Step 6 — Update integration tests

#### 6a. `integration-tests/knirvtestnet_integration_test.go`

This test currently looks for `../KNIRVGATEWAY/knirvtestnet/test-integration.sh`
which doesn't exist in the repo. Replace it with a proper Go health check
against KNIRVSERVER:

```go
// knirvtestnet_integration_test.go — rewritten
package integration_tests

import (
    "net/http"
    "testing"
    "time"
)

func TestKNIRVTestnetIntegration(t *testing.T) {
    client := &http.Client{Timeout: 10 * time.Second}

    endpoints := map[string]string{
        "knirvserver":  "http://localhost:8084/health",
        "knirvchain":   "http://localhost:8090/health",
        "knirvgraph":   "http://localhost:8082/height",
        "knirvgateway": "http://localhost:8888/gateway/health",
    }

    for name, url := range endpoints {
        resp, err := client.Get(url)
        if err != nil {
            t.Errorf("%s health check failed: %v", name, err)
            continue
        }
        resp.Body.Close()
        if resp.StatusCode >= 300 {
            t.Errorf("%s returned status %d", name, resp.StatusCode)
        }
    }
}
```

#### 6b. `integration-tests/validate_network_startup.go`

Update `NewNetworkValidator()` and `tokenURL` (line 137) to resolve against
KNIRVSERVER when the old gateway auth endpoint is unavailable:

```go
// Primary token source (gateway proxy still works at :8888)
tokenURL := "http://localhost:8888/auth/testnet-tokens"
// Fallback — direct KNIRVSERVER testnet tokens
fallbackTokenURL := "http://localhost:8084/auth/testnet-tokens"
```

No port number changes needed for service health URLs in the map; they all
stay the same.

#### 6c. `integration-tests/test_constants.go`

Add a constant for the new testnet entry point so tests can reference it:

```go
const (
    TestnetServerURL  = "http://localhost:8084"
    TestnetStatusURL  = "http://localhost:8084/testnet/status"
    TestnetTokensURL  = "http://localhost:8084/auth/testnet-tokens"
    // Legacy gateway URL kept for compatibility:
    GatewayURL        = "http://localhost:8888"
)
```

---

### Step 7 — Update modp integration

No changes required to `integration-tests/modp_formal_verification_test.go`.
It resolves `modp/` from the repo root via `runtime.Caller` and drives
`modp/scripts/run-tests.sh` directly. This is already correct.

The root Makefile testnet targets (Step 4) must also expose modp targets so
the KNIRVTESTNET Makefile delegation in Step 5 still finds them. Confirm
these exist in the root Makefile (they currently do under their own `.PHONY`
section) — no changes needed.

---

### Step 8 — Archive `devtools/KNIRVTESTNET`

Once all integration tests pass against KNIRVSERVER `--testnet`:

1. Move `devtools/KNIRVTESTNET` → `devtools/KNIRVTESTNET_ARCHIVED`
2. Add a `README.md` at the archived path explaining the migration
3. Update `CLAUDE.md` Component Map to remove KNIRVTESTNET and update
   Common Commands section

Do **not** delete the directory outright — the assets, Node.js portal
code, and per-service build scripts in `devtools/KNIRVTESTNET/scripts/` are
useful references. The formal verification infra in `devtools/KNIRVTESTNET/Makefile`
(`modp-compile`, `modp-verify`, etc.) should be lifted into the root Makefile
before archiving.

---

## File Change Summary

| File | Change |
|---|---|
| `packages/KNIRVSERVER/backend/cmd/backend_server/main.go` | Add `--testnet` flag, `applyTestnetOverrides()`, testnet config auto-load, `/testnet/status` + `/auth/testnet-tokens` routes |
| `packages/KNIRVSERVER/config/testnet.yaml` | **New file** — testnet YAML overlay |
| `Makefile` | Replace `testnet-start/stop/status/tests` with KNIRVSERVER-based versions; add `testnet-build`; lift `modp-*` targets from KNIRVTESTNET |
| `devtools/KNIRVTESTNET/Makefile` | Replace `start/stop/health/testnet/testnet-tests` with root Makefile delegation |
| `integration-tests/knirvtestnet_integration_test.go` | Rewrite to Go HTTP health checks (remove shell exec to missing script) |
| `integration-tests/validate_network_startup.go` | Add KNIRVSERVER fallback token URL |
| `integration-tests/test_constants.go` | Add `TestnetServerURL`, `TestnetStatusURL`, `TestnetTokensURL` |
| `CLAUDE.md` | Update Component Map and Common Commands once archived |

---

## Validation Criteria

The implementation is complete when:

1. `make testnet-start` builds and starts KNIRVSERVER with `--testnet`; all
   embedded managers come up; `curl localhost:8084/testnet/status` returns
   JSON with all services listed as running.

2. `make testnet-tests` starts KNIRVSERVER `--testnet`, waits for readiness,
   runs `cd integration-tests && go test ./...`, stops KNIRVSERVER, and exits
   with the test result code.

3. `cd integration-tests && go test -v -run TestKNIRVModPFormalVerification`
   passes unchanged.

4. `cd devtools/KNIRVTESTNET && make start` delegates to the root Makefile
   and starts KNIRVSERVER correctly.

5. Auth tokens are available at both `:8888/auth/testnet-tokens` (via
   embedded gateway) and `:8084/auth/testnet-tokens` (direct KNIRVSERVER
   testnet route).

---

## Risks and Mitigations

| Risk | Mitigation |
|---|---|
| Embedded manager startup order differs from old shell-ordered startup | The `Start()` method already starts managers in dependency order (chain → graph → gateway → oracle). Verify with `testnet-start` smoke test. |
| KNIRVROUTER was started as a separate binary in old testnet | KNIRVROUTER functionality is proxied through KNIRVGATEWAY. If a standalone router binary is still needed, add a `RouterConfig` + manager in KNIRVSERVER following the same pattern. |
| Old testnet Node.js portal (`:10000`) depended on by CI scripts | No integration test contacts `:10000` directly. The portal was a developer convenience. Serve equivalent status from `/testnet/status` at `:8084`. |
| `devtools/KNIRVTESTNET/scripts/build-*.sh` break after archive | Build scripts are not called during `make testnet-start` in the new flow. The single `go build` in `testnet-build` replaces all of them. |
