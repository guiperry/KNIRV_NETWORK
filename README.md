# KNIRV Network

[![License: GPL-3.0](https://img.shields.io/badge/License-GPL--3.0-blue.svg)](https://www.gnu.org/licenses/gpl-3.0)
[![Go Version](https://img.shields.io/badge/Go-1.24%2B-blue.svg)](https://golang.org/)
[![Node.js Version](https://img.shields.io/badge/Node.js-18%2B-green.svg)](https://nodejs.org/)
[![TypeScript](https://img.shields.io/badge/TypeScript-Strict-blueviolet.svg)](https://www.typescriptlang.org/)

KNIRV Network is a multi-package platform for guarded AI agent execution. Policies and guardrails wrap agent actions before they run, executions get recorded in audit trails, and failures get mined into reusable `skill.md` knowledge instead of disappearing into a log file nobody rereads.

**`KNIRVSERVER` is the entry point for the whole network.** It's the one binary you build and run. Everything else in this repo, the chain, the gateway, the graph, the oracle, the hasher, the agent runtime, ships as a binary embedded inside KNIRVSERVER and gets extracted and launched as a subprocess when it starts.

## Where to Start

```bash
cd packages/KNIRVSERVER
go build -o dist/knirv-server .
sudo ORACLE_KEY_PASSWORD=<your-oracle-key-password> ./dist/knirv-server -hasher
```

That is the same command that runs the public testnet at `testnet-gateway.knirv.network`. Full explanation, including how to run it locally without `sudo`, is in [Building and Running KNIRVSERVER](#building-and-running-knirvserver).

Don't want to run your own node? Install the CLI instead:

```bash
npm install -g @knirv/cli
knirv network status --all-services
```

## What Is Implemented

The safest factual description of the current codebase:

- `KNIRVSERVER` handles policy, guardrails, DVE management, knowledge base operations, agent launch, shell sessions, evidence anchoring, the embedded frontend, and the oracle (root nodes only)
- `KNIRVGATEWAY` provides the public portal, DHT/TURN plumbing, auth, payment, operator, and URI services, embedded inside KNIRVSERVER at runtime
- `KNIRVCHAIN` provides chain-side registry, P2P/discovery, mining, validation, wallet, and data-engine logic
- `KNIRVGRAPH` provides the knowledge graph: Network Resolution Vectors, ErrorNodes/SkillNodes, Proof-of-Solution, and NRN economics

Anything beyond that should be treated as roadmap unless it is documented in code or an API spec.

## Architecture: One Entry Point, Twelve Packages

KNIRVCHAIN started as a single Go monolith. In August 2025, about a year ago, the architecture split into independent packages, each with its own `go.mod` or `package.json` and no cross-package Go imports. Services talk to each other over HTTP, gRPC, or Unix sockets only, never a direct import.

KNIRVSERVER is where that split converges at runtime. A real deployment builds and runs exactly one binary, `packages/KNIRVSERVER/dist/knirv-server`, which embeds the compiled binaries for the backend API and every other backend package, extracts them to disk on first run, and launches each one as a child process:

```
KNIRVSERVER  (packages/KNIRVSERVER/main.go -> dist/knirv-server)
   the only binary you build and run directly
   |
   |-- extracts and runs: backend_server   (embedded API; source lives in the separate KNIRV_CORP repo)
   |-- extracts and runs: KNIRVGATEWAY     (portal, DHT, TURN, auth, payments)
   |-- extracts and runs: KNIRVCHAIN       (P2P, mining, validation, wallet)
   |-- extracts and runs: KNIRVGRAPH       (knowledge graph, NRV, ErrorNodes/SkillNodes)
   |-- extracts and runs: KNIRVORACLE      (root-node governance; only if bin/root.key is present)
   |-- extracts and runs: KNIRVHASHER      (ASIC-based neural inference pipeline)
   `-- extracts and runs: KNIRVAGENT       (autonomous agent runtime)
```

Nothing above talks to a shared database. Coordination happens over Unix sockets under `/var/lib/knirvserver/sockets/` (or your local override directory) and a handful of TCP ports defined in `packages/KNIRVSERVER/config/*.yaml`.

## Packages

| Package | Stack | Role |
|---|---|---|
| `KNIRVSERVER` | Go | Entry point. Router, proxy, guardrails, DVE and agent execution, cognitive engine, knowledge base, embedded frontend. Extracts and runs everything below it. |
| `KNIRVCHAIN` | Go | Node and agent registries, P2P discovery, mining, validation, wallet, data engine. |
| `KNIRVGATEWAY` | Go | Public portal, DHT/TURN, auth, payments, operator and URI routing. Embedded inside KNIRVSERVER at runtime. |
| `KNIRVGRAPH` | Go + TS | Knowledge graph: NRV system, ErrorNodes/SkillNodes, Proof-of-Solution, NRN economics, React graph explorer. |
| `KNIRVORACLE` | Go | Root-node governance and checkpoints. Routes only mount when an encrypted `bin/root.key` is present. |
| `KNIRVHASHER` | Go | Repurposed ASIC mining hardware (ex-Bitcoin miners) doing neural-network inference instead of hashing. |
| `KNIRVAGENT` | Go | Lightweight autonomous agent runtime, built to run on constrained hardware. |
| `KNIRVARENA` | TS / React / Three.js | 3D interactive client where human architects submit training data against live error nodes. |
| `KNIRVCONTROLLER` | React/TS + Vite | The end-user app: vault, DVE identities, voice/text chat with the cognitive engine. Ships as a PWA and as native Android/iOS wrappers via Capacitor. |
| `KNIRVBRIDGE` | TS | Browser wallet extension for NRN tokens and dApp interaction. |
| `KNIRVBASE` | Go + TS | Shared SDK/library other packages build on: auth, UI primitives, shared client code. |
| `KNIRVSDK` | Go / TS / Py | Developer SDKs, plus the source for KNIRV-CLI (`@knirv/cli`). |
| `websites/REGISTRY.KNIRV.NETWORK` | Cloudflare Workers | Network node registry and failover control plane. Stores node registrations, validator heartbeats, votes, and state transitions in a Durable Object. Consumed by KNIRVGATEWAY (DHT bootstrap) and KNIRVCHAIN (self-registration). |

## Building and Running KNIRVSERVER

### Prerequisites

- Go 1.24+ (individual packages target between 1.24 and 1.25; check each `go.mod` if you hit a toolchain mismatch)
- Node.js 18+, only if you're rebuilding a frontend or TypeScript package
- Optional: a sibling checkout of the private `KNIRV_CORP` repo, needed only if you want to rebuild `backend_server` and the other embedded packages from source. Their compiled binaries already ship inside this repo (`packages/KNIRVSERVER/bin/backend_server`, `packages/KNIRVSERVER/pkg/knirv*/bin/*`), so a plain build works without it.

### Build

```bash
cd packages/KNIRVSERVER
go build -o dist/knirv-server .
```

This links the already-vendored `backend_server`, `KNIRVGATEWAY`, `KNIRVCHAIN`, `KNIRVGRAPH`, `KNIRVORACLE`, `KNIRVHASHER`, and `KNIRVAGENT` binaries, plus the built frontend export in `frontend/out/`, into one Go binary. Nothing downstream needs recompiling, so this takes seconds.

For a full from-source rebuild of every embedded package (requires the sibling `KNIRV_CORP` repo for `backend_server`'s source):

```bash
cd packages/KNIRVSERVER
make binary
```

### Run

KNIRVSERVER defaults to **testnet mode**. There is no `--testnet` flag; testnet is simply what you get unless you pass `-prod`, `-dev`, or `-ent`.

```bash
sudo ORACLE_KEY_PASSWORD=<your-oracle-key-password> ./dist/knirv-server -hasher
```

By default the binary reads config from `/etc/knirv-server` and writes runtime state to `/var/lib/knirvserver`, which is why `sudo` is normally required. `ORACLE_KEY_PASSWORD` decrypts `packages/KNIRVSERVER/bin/root.key` to unlock the `/oracle/*` routes; no key file, no password needed, and the server runs fine as a non-root node. `-hasher` starts the KNIRVHASHER training pipeline alongside everything else and is optional.

To run locally without `sudo`, point the app-data and config directories at somewhere you own:

```bash
mkdir -p .local/data .local/config
KNIRV_APP_DATA_DIR="$(pwd)/.local/data" \
KNIRV_CONFIG_DIR="$(pwd)/.local/config" \
./dist/knirv-server
```

Other useful flags: `-config <path>`, `-dev`, `-prod`, `-ent`, `-port <n>`, `-host <addr>`, `-user-id-tag <tag>`.

### Verify it's running

```bash
curl http://localhost:8090/health
```

`8090` is the KNIRVSERVER wrapper's own port (`config/testnet.yaml`). The embedded backend API listens on `8082`, the embedded KNIRVGATEWAY on `8080`, and KNIRVCHAIN/KNIRVGRAPH mostly communicate over Unix sockets rather than TCP. All of it runs under one process tree started by KNIRVSERVER.

### Or use the Makefile

```bash
make testnet-build   # builds packages/KNIRVSERVER/dist/knirv-server
make testnet-start   # starts it in the background and health-checks it
make testnet-status  # check status
make testnet-stop    # stop it
```

## Deployment

Everything in [Building and Running KNIRVSERVER](#building-and-running-knirvserver) above covers running the binary directly on a host you already have. Production and containerized rollout (including the box behind `testnet-gateway.knirv.network`) goes through three purpose-built tools that live in the private `KNIRV_CORP` repo, at `packages/server/{os_builder,container_deployer,image_installer}`. This repo doesn't automate that pipeline; this section documents how it works.

### 1. Build an eBPF-capable OS image (`os_builder`)

`os_builder` produces the base OS artifact KNIRVSERVER's eBPF features (XDP filtering, LSM sandboxing) need: a Debian image built around a custom `linux-lts` kernel with `CONFIG_BPF`, `CONFIG_BPF_SYSCALL`, `CONFIG_BPF_JIT`, `CONFIG_DEBUG_INFO_BTF`, `CONFIG_KPROBES`, `CONFIG_UPROBES`, and related options enabled (see its `Debian_config.md`).

```bash
# from KNIRV_CORP/packages/server/os_builder
go run . -image debian -action 0   # base OVA image
go run . -image debian -action 2   # Kata guest kernel + rootfs (Terraform)
go run . -image debian -action 3   # AWS AMI
```

A Kali-based image is also supported (`-image kali`) for the enterprise/hardened edition. Building the Docker image itself was migrated out of `os_builder` and into `container_deployer` below; `os_builder` now only needs to run first if you want a from-scratch custom kernel rather than the host's existing one.

### 2. Bundle the `knirv-server` binary into a container (`container_deployer`)

`container_deployer` is self-contained: it embeds its own Ansible playbooks, Containerfile/Packer templates, and the `knirv-server` binary, so it doesn't need anything checked out separately.

```bash
# from KNIRV_CORP/packages/server/container_deployer
./container_deployer --image debian --action 1   # build knirvserver-debian-base:latest
./container_deployer --image kali --action 1     # build knirvserver-kali-base:latest

# skip the rebuild and load a base image os_builder already produced
docker load -i ~/.local/share/knirvserver/os_builder/artifacts/knirvserver-kali-base.tar
./container_deployer --skip-image-build --image kali --action 1
```

By default it pushes the result to Docker Hub (`knirvcorp/knirvserver:debian-latest` / `:kali-latest`); pass `--push=false` to start it locally instead, or `--build-only` to just produce the image.

### 3. Run it and bridge in the node key (`image_installer`)

The container needs two things from the host at `docker run` time: the eBPF capabilities, and the node's identity key. Neither gets baked into the image; both are bridged in through bind mounts.

```bash
DATA_DIR=/opt/knirv/data   # or wherever you want node state to live on the host

docker run -d \
  --name knirvserver \
  --cap-add NET_ADMIN --cap-add BPF --cap-add SYS_PTRACE --cap-add PERFMON --cap-add NET_RAW \
  --security-opt seccomp=unconfined \
  -p 8080:8080 -p 8090:8090 -p 8082:8082 -p 8089:8089 \
  -p 4001:4001/tcp -p 4001:4001/udp -p 9090:9090 \
  -v "$DATA_DIR:/var/lib/knirvserver" \
  -v "$DATA_DIR/.key:/root/.config/knirv-server/.key:ro" \
  -e ORACLE_KEY_PASSWORD="$ORACLE_KEY_PASSWORD" \
  knirvcorp/knirvserver:debian-latest
```

`$DATA_DIR/.key/` is the filesystem bridge: drop `root.key` (root/oracle node), `enterprise.key` (enterprise operator node), or `boot.key` (bootnode) there on the host, and it shows up read-only inside the container at the fixed path KNIRVSERVER's key loader expects. No key file present is a normal, supported state; the server just runs without oracle/root routes.

`image_installer` (the `knirvserver` installer binary, built via `make installer` in `KNIRV_CORP`) automates exactly this: it prompts for edition and environment, discovers a key file dropped next to the binary or in the invocation directory, stages it into `$DATA_DIR/.key/`, then runs the equivalent `docker run` and polls `/health` until the container reports ready. Use it directly rather than hand-rolling the command above unless you need something the installer's flags don't cover.

## Public Testnet and the CLI

You don't need to run your own node to try the network. The public testnet is live at `testnet-gateway.knirv.network`, and **KNIRV-CLI is the flagship way to reach it**, or any node you run yourself:

```bash
npm install -g @knirv/cli

# or grab a standalone binary from releases.knirv.com/cli/<platform>/knirv

knirv network status --all-services
knirv economics balance --address 0x... --include-pending
knirv mcp nrv submit-error --auto-resolution --skill-suggestion
```

KNIRV-CLI is a single client for every layer of the network: service discovery with health checks and a circuit breaker, wallet support with gasless XION meta accounts, an NRN token manager, an interactive REPL, and AI-assisted MCP plugin generation. See `packages/KNIRVSDK` and [`docs/whitepapers/KNIRV-CLI_Whitepaper.md`](./docs/whitepapers/KNIRV-CLI_Whitepaper.md) for the full command surface.

## Repository Layout

| Path | What's there |
|---|---|
| `packages/` | The 12 packages listed above. |
| `integration-tests/` | Go integration tests that hit real running services. No mocks. |
| `modp/` | Formal verification models written in the P language, checked with PChecker. |
| `docs/` | Whitepapers, architecture notes, and implementation guides. |
| `scripts/` | Sync and local testnet management scripts. |
| `websites/` | Public-facing sites, including the KNIRV.NETWORK marketing and docs site. |
| `shared-proto/` | Shared Protobuf definitions used across packages. |

Production/container deployment lives in the private `KNIRV_CORP` repo, not in this one; see [Deployment](#deployment) above.

## Testing

```bash
# KNIRVSERVER: wrapper-level tests that live in this repo
cd packages/KNIRVSERVER && go test -v ./integration-tests/...

# Other packages
cd packages/KNIRVCHAIN && go test -v ./tests/unit/...
cd packages/KNIRVGATEWAY && go test -v ./...
cd packages/KNIRVGRAPH && go test -v ./...
cd packages/KNIRVORACLE && go test -v ./...

# Cross-service integration tests (real services, no mocks)
cd integration-tests && go test -v ./...
```

`backend_server`'s own business-logic tests (cognitive engine, onboarding, guardrails) live in the separate `KNIRV_CORP` repo, since that's where its source lives. This repo only embeds the compiled binary.

## Formal Verification

Async protocols get a corresponding P-language model, verified with PChecker.

```bash
bash modp/scripts/run-tests.sh
```

Relevant models:

- `modp/events/network_events.p`
- `modp/components/base/base_layer.p`
- `modp/components/chain/skill_registry.p`
- `modp/components/oracle/governance_machine.p`
- `modp/monitors/network_invariants.p`

## Conventions

- Each `packages/KNIRV*` is independent: `go mod tidy` and builds run inside the package, not at the root
- No cross-package Go imports; inter-service communication is HTTP, gRPC, or Unix sockets only
- TypeScript: prefer `unknown` over `any`
- Integration tests hit real services; no mock DB or mock network
- Oracle routes only mount when `packages/KNIRVSERVER/bin/root.key` is present
- Keep secrets out of the repo; use `ORACLE_KEY_PASSWORD` and friends as environment variables

## Documentation Index

- [`packages/KNIRVSERVER/README.md`](./packages/KNIRVSERVER/README.md)
- [`packages/KNIRVSERVER/USER_WORKFLOWS_AND_PRODUCTION_PLAN.md`](./packages/KNIRVSERVER/USER_WORKFLOWS_AND_PRODUCTION_PLAN.md)
- [`packages/KNIRVSERVER/server-api.yaml`](./packages/KNIRVSERVER/server-api.yaml)
- [`integration-tests/README.md`](./integration-tests/README.md)
- [`packages/KNIRVGATEWAY/README.md`](./packages/KNIRVGATEWAY/README.md)
- [`packages/KNIRVCHAIN/README.md`](./packages/KNIRVCHAIN/README.md)
- [`modp/README.md`](./modp/README.md)
- [`docs/whitepapers/KNIRV-CLI_Whitepaper.md`](./docs/whitepapers/KNIRV-CLI_Whitepaper.md)

## Roadmap Boundary

The repo still contains long-range language about sovereign layers and network flywheels in places. Treat that as roadmap language unless a specific file or endpoint backs it up. The current codebase is strong enough to document as a guarded execution and coordination platform; it is not accurate to describe every aspirational layer as complete.

## Contributing

See [Conventions](#conventions) above, and keep this README in sync when entry points, ports, or build commands change.

## License

GPL-3.0
