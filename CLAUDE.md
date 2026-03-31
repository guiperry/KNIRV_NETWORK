# CLAUDE.md

## Project Overview

KNIRV Network — Decentralized Trusted Execution Network (D-TEN). Transforms AI failures into collective knowledge through sovereign blockchain layers communicating via IBC. Oracle service consolidated into `packages/KNIRVSERVER` (root nodes only via encrypted `packages/KNIRVSERVER/bin/root.key`). KNIRVGATEWAY no longer manages oracle.

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

**KNIRVSERVER internals:** `backend/internal/services/cognitiveengine/cognitive_engine.go` · `backend/internal/services/onboarding/` · `backend/internal/services/pluginserver/` · `backend/internal/web/guardrail_handlers.go` · `backend/internal/oracle/` (root-node only) · `backend/config/embedded/`

**KNIRVCHAIN internals:** `internal/mining/` · `internal/validation/` · `internal/auth/` · `internal/cache/` · `internal/database/` · `internal/agent/` · `internal/pricing/` · `internal/classifier/` · `internal/resilience/` · `internal/security/` · `internal/tracing/`

**KNIRVGATEWAY internals:** `internal/server/` · `internal/config/` · `internal/turnserver/` · `internal/embedded/`

**Node Transformation Flows (KNIRVCHAIN):**
- `ErrorNode → SkillNode` mining → LoRA Adapter Pointer
- `ContextNode → CapabilityNode` minting → MCP Server Pointer
- `IdeaNode → PropertyNode` making → Inference NFT Pointer

**Oracle (KNIRVSERVER root nodes only):** Loads `packages/KNIRVSERVER/bin/root.key` (AES-encrypted). Password via `ORACLE_KEY_PASSWORD` env or stdin prompt. Routes mounted at `/oracle/` only when key present. Missing key = normal operation, no oracle.

## Common Commands

```bash
make tests                              # run all tests
make testnet-tests                      # quick testnet tests
make testnet-start                      # start testnet
make testnet-stop                       # stop testnet
make testnet-status                     # testnet health
make docs                               # generate docs
make deploy-full ENVIRONMENT=production CLOUD_PROVIDER=aws
```

## Go Tests (Per Package)

```bash
cd packages/KNIRVSERVER && go test -v ./backend/tests/...
cd packages/KNIRVCHAIN && go test -v ./tests/unit/...
cd packages/KNIRVGATEWAY && go test -v ./...
cd integration-tests && go test -v -run "TestKNIRVNEXUS.*"
cd devtools/KNIRVSYNC && go test -v ./...
cd devtools/network-monitor && go test -v ./...
```

## Testnet Scripts (`packages/KNIRVTESTNET/scripts/`)

```bash
./scripts/start-testnet.sh              # start all services
./scripts/stop-testnet.sh               # stop all services
./scripts/health-check.sh               # service health
./scripts/validate-config.sh            # config validation
./scripts/run-tests.sh                  # integration tests
./scripts/run-tests.sh --all            # full test suite
./scripts/build-all.sh                  # build all services
node scripts/load-endpoints.js testnet  # load testnet endpoints
```

**Config files:** `packages/KNIRVTESTNET/config/testnet-config.yaml` · `packages/KNIRVTESTNET/config/knirvserver-testnet-config.yaml` · `packages/KNIRVTESTNET/config/knirvrouter-testnet.env` · `packages/KNIRVTESTNET/.env`

**Service ports:** KNIRVORACLE `:1317` · KNIRVCHAIN `:8090` · KNIRVGRAPH `:8082` · KNIRVSERVER `:8084` · KNIRVROUTER `:8086` · KNIRVGATEWAY `:8888` · KNIRVTESTNET `:10000`

## Formal Verification (ModP / P Language)

P models in `modp/` verified with dotnet PChecker via `modp/KnirvNetwork.pproj`.

```bash
cd packages/KNIRVTESTNET && make test-modp   # run via testnet Makefile
bash modp/scripts/run-tests.sh               # direct P tests
```

**P model layout:** `modp/events/network_events.p` (events) · `modp/components/base/base_layer.p` · `modp/components/nexus/` · `modp/components/chain/skill_registry.p` · `modp/components/oracle/governance_machine.p` · `modp/components/oracle/token_machine.p` · `modp/components/router/p2p_network.p` · `modp/monitors/network_invariants.p` · `modp/tests/network_composition_tests.p`

Results: `modp/results/` · `modp/PCheckerOutput/BugFinding/` · `modp/PCheckerOutput/BugFinding3/`

ModP integration test: `integration-tests/modp_formal_verification_test.go`

## Integration Tests

Entry: `integration-tests/cmd/validate_network/main.go` · `integration-tests/validate_network_startup.go` · `integration-tests/utils/service_discovery.go`

Config: `integration-tests/config/service-discovery.yaml`

Scripts: `integration-tests/run-modp-tests.sh` · `scripts/run-full-demo.sh` · `scripts/validate-testnet-complete.sh`

## KNIRVARENA (3D RTS Game)

TS client: `packages/KNIRVARENA/packages/ts_client_2/src/` — React + Three.js

Key files: `src/components/KNIRVANAGameVisualization.tsx` · `src/components/game/GameScene.tsx` · `src/components/game/RewardAnchor3D.tsx` · `src/engine/TrainingManager.ts` · `src/engine/Sabotage.ts` · `src/networking/ArenaClient.ts` · `src/core/api/knirvbase.ts`

Storage: `src/core/storage/BrowserDB.ts` · `src/core/storage/BrowserStorage.ts`

Agent compiler builds: `src/core/agent-core-compiler/build/` · tests: `tests/unit/engine/`

Run: `packages/KNIRVARENA/scripts/run-all-tests.sh` · `packages/KNIRVARENA/Makefile`

## KNIRVHEART (Neural Intelligence)

Specs: `packages/KNIRVHEART/HEART/HEART_SDD_Full.md` · `packages/KNIRVHEART/VLM/VLM_SDD.md`

Python: `packages/KNIRVHEART/HEART/SequenceBuffer.py` · transformer: `packages/KNIRVHEART/HEART/go_transformer/`

CEREAS SDK: `packages/KNIRVHEART/HEART/CERERAS_SDK/`

## Devtools

- **KNIRVSYNC** (`devtools/KNIRVSYNC/`): doc/env sync — `internal/orchestrator.go` · `internal/sync-manager.go` · `internal/doc-sync.go` · `bin/sync`
- **Network Monitor** (`devtools/network-monitor/`): Prometheus + Grafana — `docker-compose.monitoring.yml` · `config/prometheus.yml` · `config/grafana/` · `config/alertmanager.yml`

## KNIRVBASE SDK

- Go SDK: `packages/KNIRVBASE/go/`
- TS SDK: `packages/KNIRVBASE/ts/` — dist at `packages/KNIRVBASE/ts/dist/lib/index.js` (includes new `dist/components/auth/`, `dist/components/security/`, `dist/components/monitoring/`)

## Websites

- `websites/KNIRVHUB/network-website/` — KNIRV network hub (health-monitor.html, index.html)
- `websites/KNIRVRAMP/` — Next.js + Supabase (`websites/KNIRVRAMP/supabase/migrations/`, `websites/KNIRVRAMP/src/components/`)

## KNIRVSERVER Frontend

Next.js: `packages/KNIRVSERVER/frontend/` · built to `packages/KNIRVSERVER/frontend/out/`

Key components: `src/components/dashboard/badge-lab-panel.tsx` · `src/components/dashboard/policy-editor.tsx` · `src/components/onboarding/onboarding-guide.tsx`

Desktop app: `packages/KNIRVSERVER/desktop/` (Electron — `renderer.js`, `index.html`)

## Conventions

- Each `packages/KNIRV*` is independent: `go mod tidy` and builds run inside the package, not root
- No cross-package Go imports — inter-service communication via HTTP/gRPC only
- TypeScript: `unknown` over `any`; imports from `packages/KNIRVBASE/ts/dist/`
- Integration tests hit real services — no mock DB/network (see `integration-tests/`)
- New async protocols require corresponding P machine in `modp/components/`
- Oracle routes only when `packages/KNIRVSERVER/bin/root.key` present
- KNIRVGATEWAY does NOT initialize oracle — removed in recent refactor

## Production / Docs

`packages/KNIRVSERVER/config/production.yaml` · `packages/KNIRVSERVER/docs/SYSTEMD_SERVICE.md` · `packages/KNIRVSERVER/docs/Production_Deployment_Architecture.md` · `packages/KNIRVSERVER/docs/TESTING_PRIVILEGED.md` · `packages/KNIRVSERVER/docs/eBPF_Integration_Guide.md`

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

If `caliber` is not found, tell the user: "This project uses Caliber for agent config sync. Run /setup-caliber to get set up."
<!-- /caliber:managed:pre-commit -->

<!-- caliber:managed:learnings -->
## Session Learnings

Read `CALIBER_LEARNINGS.md` for patterns and anti-patterns learned from previous sessions.
These are auto-extracted from real tool usage — treat them as project-specific rules.
<!-- /caliber:managed:learnings -->

<!-- caliber:managed:sync -->
## Context Sync

This project uses [Caliber](https://github.com/caliber-ai-org/ai-setup) to keep AI agent configs in sync across Claude Code, Cursor, Copilot, and Codex.
Configs update automatically before each commit via `caliber refresh`.
If the pre-commit hook is not set up, run `/setup-caliber` to configure everything automatically.
<!-- /caliber:managed:sync -->
