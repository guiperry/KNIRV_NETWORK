# KNIRV Network

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Version](https://img.shields.io/badge/Go-1.24%2B-blue.svg)](https://golang.org/)
[![Node.js Version](https://img.shields.io/badge/Node.js-18%2B-green.svg)](https://nodejs.org/)
[![TypeScript](https://img.shields.io/badge/TypeScript-Strict-blueviolet.svg)](https://www.typescriptlang.org/)

KNIRV Network is a multi-package system for AI execution control, validation, and coordination. The repo centers on a guarded execution server, a chain-side registry and validation layer, and a gateway that proxies and exposes the network’s runtime surfaces.

The safest factual description is:

- `KNIRVSERVER` handles policy, guardrails, DVE management, knowledge base operations, agent launch, shell sessions, evidence anchoring, and the embedded frontend
- `KNIRVGATEWAY` provides the public portal, DHT/TURN plumbing, auth, payment, operator, and URI services
- `KNIRVCHAIN` provides chain-side registry, P2P/discovery, mining, validation, wallet, and data-engine logic

Anything beyond that should be treated as roadmap unless it is documented in code or API spec.

## What Is Implemented

### KNIRVSERVER

The server at `packages/KNIRVSERVER` exposes:

- health and version endpoints
- local system info and update/apply routes
- auth, onboarding, DVE, agent, shell, payments, guardrail, cognitive, and knowledge-base APIs
- WebSocket and SSE pass-through for interactive sessions
- proxy routes for `/network`, `/explorer`, `/gateway`, `/turn`, `/dve`, `/arena`, and `/tunnel`
- an embedded frontend export served from the Go binary
- a compiled backend binary in `packages/KNIRVSERVER/bin/backend_server`, embedded so the portal/runtime layer does not expose the backend business logic directly

The canonical API surface is `packages/KNIRVSERVER/server-api.yaml`.

### KNIRVGATEWAY

The gateway at `packages/KNIRVGATEWAY` provides:

- web portal and embedded web GUI delivery
- session management
- auth handling
- operator registry routes
- DHT announce, discovery, cache, and peer routes
- TURN server plumbing
- payment service routes
- URI generation and registration routes
- proxying for chain, oracle, tunnel, and network-monitor surfaces

### KNIRVCHAIN

The chain package at `packages/KNIRVCHAIN` includes:

- node and agent registries
- P2P discovery and relay helpers
- mining and transformation flows
- proof and validation logic
- wallet and payment utilities
- data-engine streams and WebSocket APIs
- URI, plugin, and blockchain utilities

## Current Product Slice

The current product wedge is an execution-control system for AI workloads:

1. define policies and guardrails
2. attach them to DVEs or agent workflows
3. validate or execute against those controls
4. record the result in audit trails, knowledge stores, or chain-backed flows



## Repository Map

| Package | Purpose | Notes |
|---|---|---|
| `packages/KNIRVSERVER` | Guardrailed execution server | Embedded frontend, API proxying, knowledge base, onboarding, cognitive engine |
| `packages/KNIRVGATEWAY` | Public gateway and portal | DHT, TURN, auth, payments, URI routing, web GUI |
| `packages/KNIRVCHAIN` | Chain-side registry and network services | Mining, validation, wallet, P2P, data engine |
| `packages/KNIRVBASE/ts` | Shared TypeScript UI/auth tooling | Used by frontend components and shared client code |
| `packages/KNIRVARENA` | Interactive client / 3D surfaces | Browser client and visualization work |
| `packages/KNIRVHASHER` | Supporting AI / orchestration work | Python/Go support code |
| `devtools/KNIRVTESTNET` | Testnet orchestration | Scripts, configs, deployment helpers |
| `devtools/KNIRVSYNC` | Sync tooling | Go-based sync utilities |
| `devtools/network-monitor` | Monitoring utilities | Go monitoring stack |
| `integration-tests` | End-to-end validation | Real services, no mocks |
| `modp` | Formal models | P-language verification assets |

## Current Server Surface

`packages/KNIRVSERVER/server-api.yaml` documents the public API groups that currently exist in the server:

- health and system
- auth
- DVE and DVE creation
- agent management
- payments and NRN-related payment flows
- shell and session management
- onboarding
- cognitive engine
- guardrails and policy management
- evidence anchoring
- TEE support
- memory and knowledge base
- ontology and Nexus surfaces
- data engine and analytics
- WebSocket endpoints



## Core Runtime Model

The main server is a router and proxy layer around a few specific runtime concerns:

- serve the embedded frontend
- proxy API calls to a backend subprocess or gateway socket
- preserve long-lived streaming connections for SSE and WebSocket traffic
- expose local-only system info and update control
- bridge the user-facing UI into backend DVE and agent workflows

That is the actual shape of the runtime today.

## How To Work With The Repo

### Common Commands

```bash
make tests
make testnet-tests
make testnet-start
make testnet-stop
make testnet-status
make docs
make deploy-full ENVIRONMENT=production CLOUD_PROVIDER=aws
```

### Per-Package Go Tests

```bash
cd packages/KNIRVSERVER && go test -v ./...
cd packages/KNIRVCHAIN && go test -v ./tests/unit/...
cd packages/KNIRVGATEWAY && go test -v ./...
cd integration-tests && go test -v -run "TestKNIRVNEXUS.*"
```

### Testnet Scripts

```bash
./scripts/start-testnet.sh
./scripts/stop-testnet.sh
./scripts/health-check.sh
./scripts/validate-config.sh
./scripts/run-tests.sh --all
./scripts/build-all.sh
node scripts/load-endpoints.js testnet
```

Typical testnet ports:

- `KNIRVCHAIN`: `8090`
- `KNIRVGRAPH`: `8082`
- `KNIRVSERVER`: `8084`
- `KNIRVGATEWAY`: `8080`

## Formal Verification

ModP / P-language validation lives under `modp/`.

```bash
cd devtools/KNIRVTESTNET && make test-modp
bash modp/scripts/run-tests.sh
```

Relevant models:

- `modp/events/network_events.p`
- `modp/components/base/base_layer.p`
- `modp/components/nexus/`
- `modp/components/chain/skill_registry.p`
- `modp/components/oracle/governance_machine.p`
- `modp/monitors/network_invariants.p`
- `modp/tests/network_composition_tests.p`

## Documentation Index

Start here for the factual current-state docs:

- `packages/KNIRVSERVER/README.md`
- `packages/KNIRVSERVER/Production_Status_Report.md`
- `packages/KNIRVSERVER/USER_WORKFLOWS_AND_PRODUCTION_PLAN.md`
- `packages/KNIRVSERVER/server-api.yaml`
- `integration-tests/README.md`
- `packages/KNIRVGATEWAY/README.md`
- `modp/README.md`


## Roadmap Boundary

The repo still contains long-range language about D-TEN, sovereign layers, and network flywheels. We've kept those as roadmap language unless a specific file or endpoint to show the functionality is already in place.

The current codebase is strong enough to document as a guarded execution and coordination platform. It is not accurate to describe every aspirational layer as complete.

## Contributing

Use the project conventions in `AGENTS.md`:

- keep packages independent
- prefer `unknown` over `any` in TypeScript
- use real integration tests for service behavior
- update docs when behavior changes
- keep secrets out of the repo

## License

MIT
