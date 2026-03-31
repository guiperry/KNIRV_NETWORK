# AGENTS.md

## Project Overview

KNIRV Network — Decentralized Trusted Execution Network (D-TEN). Transforms AI failures into collective knowledge through sovereign blockchain layers communicating via IBC. Oracle consolidated into `packages/KNIRVSERVER` (root nodes only via `packages/KNIRVSERVER/bin/root.key`). KNIRVGATEWAY no longer manages oracle.

## Component Map

| Package | Tech | Module file |
|---------|------|-------------|
| `packages/KNIRVCHAIN` | Go 1.21+ | `packages/KNIRVCHAIN/go.mod` |
| `packages/KNIRVSERVER` | Go 1.21+ | `packages/KNIRVSERVER/backend/go.mod` |
| `packages/KNIRVGATEWAY` | Go | `packages/KNIRVGATEWAY/go.mod` |
| `packages/KNIRVBASE/ts` | Node 18+ | `packages/KNIRVBASE/ts/package-lock.json` |
| `packages/KNIRVARENA` | TS/React/Three.js | `packages/KNIRVARENA/packages/ts_client_2/` |
| `packages/KNIRVHEART` | Python/Go | `packages/KNIRVHEART/HEART/` |
| `packages/KNIRVTESTNET` | Node.js | `packages/KNIRVTESTNET/Makefile` |
| `devtools/KNIRVSYNC` | Go | `devtools/KNIRVSYNC/go.mod` |
| `devtools/network-monitor` | Go | `devtools/network-monitor/go.mod` |
| `integration-tests` | Go | `integration-tests/go.mod` |
| `modp` | P language | `modp/KnirvNetwork.pproj` |

## Architecture

**Entry points:**
- KNIRVSERVER: `packages/KNIRVSERVER/main.go` → `packages/KNIRVSERVER/backend/cmd/backend_server/main.go`
- KNIRVGATEWAY: `packages/KNIRVGATEWAY/cmd/gateway/main.go` → `packages/KNIRVGATEWAY/internal/server/`
- KNIRVCHAIN: `packages/KNIRVCHAIN/cmd/knirvchain/` → `packages/KNIRVCHAIN/internal/`

**KNIRVSERVER internals:** `backend/internal/services/cognitiveengine/cognitive_engine.go` · `backend/internal/services/onboarding/` · `backend/internal/services/pluginserver/` · `backend/internal/web/guardrail_handlers.go` · `backend/internal/oracle/` (root-node only)

**KNIRVCHAIN internals:** `internal/mining/` · `internal/validation/` · `internal/auth/` · `internal/cache/` · `internal/database/` · `internal/agent/` · `internal/pricing/` · `internal/classifier/` · `internal/resilience/` · `internal/security/` · `internal/tracing/`

**Node Transformation Flows:**
- `ErrorNode → SkillNode` mining → LoRA Adapter Pointer
- `ContextNode → CapabilityNode` minting → MCP Server Pointer
- `IdeaNode → PropertyNode` making → Inference NFT Pointer

**Oracle:** Loads `packages/KNIRVSERVER/bin/root.key` (encrypted). Password via `ORACLE_KEY_PASSWORD` env. Routes at `/oracle/` only when key present.

## Commands

```bash
make tests
make testnet-tests
make testnet-start
make testnet-stop
make testnet-status
make docs
make deploy-full ENVIRONMENT=production CLOUD_PROVIDER=aws
```

## Go Tests (Per Package)

```bash
cd packages/KNIRVSERVER && go test -v ./backend/tests/...
cd packages/KNIRVCHAIN && go test -v ./tests/unit/...
cd packages/KNIRVGATEWAY && go test -v ./...
cd integration-tests && go test -v -run "TestKNIRVNEXUS.*"
cd devtools/KNIRVSYNC && go test -v ./...
```

## Testnet Scripts (`packages/KNIRVTESTNET/scripts/`)

```bash
./scripts/start-testnet.sh
./scripts/stop-testnet.sh
./scripts/health-check.sh
./scripts/validate-config.sh
./scripts/run-tests.sh --all
./scripts/build-all.sh
node scripts/load-endpoints.js testnet
```

**Configs:** `packages/KNIRVTESTNET/config/testnet-config.yaml` · `packages/KNIRVTESTNET/config/knirvserver-testnet-config.yaml` · `packages/KNIRVTESTNET/.env`

**Ports:** KNIRVORACLE `:1317` · KNIRVCHAIN `:8090` · KNIRVGRAPH `:8082` · KNIRVSERVER `:8084` · KNIRVROUTER `:8086` · KNIRVGATEWAY `:8888`

## Formal Verification (ModP / P Language)

```bash
cd packages/KNIRVTESTNET && make test-modp
bash modp/scripts/run-tests.sh
```

**P models:** `modp/events/network_events.p` · `modp/components/base/base_layer.p` · `modp/components/nexus/` · `modp/components/chain/skill_registry.p` · `modp/components/oracle/governance_machine.p` · `modp/monitors/network_invariants.p` · `modp/tests/network_composition_tests.p`

ModP integration: `integration-tests/modp_formal_verification_test.go`

## Integration Tests

Entry: `integration-tests/cmd/validate_network/main.go` · `integration-tests/validate_network_startup.go`

Config: `integration-tests/config/service-discovery.yaml`

Scripts: `scripts/run-full-demo.sh` · `scripts/validate-testnet-complete.sh` · `integration-tests/run-modp-tests.sh`

## KNIRVARENA (3D RTS)

TS client: `packages/KNIRVARENA/packages/ts_client_2/src/`

Key: `src/components/KNIRVANAGameVisualization.tsx` · `src/components/game/GameScene.tsx` · `src/engine/TrainingManager.ts` · `src/networking/ArenaClient.ts` · `src/core/api/knirvbase.ts` · `src/core/storage/BrowserDB.ts`

## Devtools

- **KNIRVSYNC** (`devtools/KNIRVSYNC/`): `internal/orchestrator.go` · `internal/sync-manager.go` · `bin/sync`
- **Network Monitor** (`devtools/network-monitor/`): `docker-compose.monitoring.yml` · `config/prometheus.yml` · `config/alertmanager.yml`

## Conventions

- Each `packages/KNIRV*` is independent — run `go mod tidy` inside the package
- No cross-package Go imports — HTTP/gRPC only for inter-service communication
- TypeScript: `unknown` over `any`; import from `packages/KNIRVBASE/ts/dist/`
- Integration tests hit real services (no mock DB/network)
- New async protocols need corresponding P machine in `modp/components/`
- Oracle routes only when `packages/KNIRVSERVER/bin/root.key` present
- KNIRVGATEWAY does NOT initialize oracle

<!-- caliber:managed:pre-commit -->
## Before Committing

**IMPORTANT:** Before every git commit, you MUST ensure Caliber syncs agent configs with the latest code changes.

First, check if the pre-commit hook is already installed:
```bash
grep -q "caliber" .git/hooks/pre-commit 2>/dev/null && echo "hook-active" || echo "no-hook"
```

- If **hook-active**: the hook handles sync automatically — just commit normally. Tell the user: "Caliber will sync your agent configs automatically via the pre-commit hook."
- If **no-hook**: run Caliber manually before committing:
  1. Tell the user: "Caliber: Syncing agent configs with your latest changes..."
  2. Run: `caliber refresh && git add CLAUDE.md .claude/ .cursor/ .cursorrules .github/copilot-instructions.md .github/instructions/ AGENTS.md CALIBER_LEARNINGS.md .agents/ .opencode/ 2>/dev/null`
  3. After it completes, briefly tell the user what Caliber updated. Then proceed with the commit.

If `caliber` is not found, read `.agents/skills/setup-caliber/SKILL.md` and follow its instructions to install Caliber.
<!-- /caliber:managed:pre-commit -->

<!-- caliber:managed:learnings -->
## Session Learnings

Read `CALIBER_LEARNINGS.md` for patterns and anti-patterns learned from previous sessions.
These are auto-extracted from real tool usage — treat them as project-specific rules.
<!-- /caliber:managed:learnings -->
