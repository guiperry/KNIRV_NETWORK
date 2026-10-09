# KNIRV Network

[![License: GPL-3.0](https://img.shields.io/badge/License-GPL--3.0-blue.svg)](https://www.gnu.org/licenses/gpl-3.0)
[![Go Version](https://img.shields.io/badge/Go-1.25%2B-blue.svg)](https://golang.org/)
[![Node.js Version](https://img.shields.io/badge/Node.js-18%2B-green.svg)](https://nodejs.org/)
[![TypeScript](https://img.shields.io/badge/TypeScript-Strict-blueviolet.svg)](https://www.typescriptlang.org/)

KNIRV Network is a multi-package platform for guarded AI agent execution. Policies and guardrails wrap agent actions before they run, executions get recorded in audit trails, and failures get mined into reusable `skill.md` knowledge instead of disappearing into a log file nobody rereads.

**`KNIRVSERVER` is the entry point for the whole network.** It's the one binary you build and run. Everything else in this repo, the chain, the gateway, the graph, the oracle, the hasher, the agent runtime, the adapter compiler, the local LLM, ships as a binary embedded inside KNIRVSERVER (directly, or inside the embedded `backend_server`) and gets extracted and launched as a subprocess when it starts.

This repo is one half of the KNIRV workspace. Its sibling, the private [`KNIRV_CORP`](../KNIRV_CORP/README.md) repo, holds the source for `backend_server` (the API KNIRVSERVER embeds) and the user-facing products: the KNIRV CLI, the desktop client, the KNIRVCONTROLLER mobile app, and KNIRVENGINE.

## Where to Start

```bash
cd packages/KNIRVSERVER
go build -o dist/knirv-server .
sudo ORACLE_KEY_PASSWORD=<your-oracle-key-password> ./dist/knirv-server -hasher
```

That is the same command that runs the public testnet at `testnet-gateway.knirv.com`. Full explanation, including how to run it locally without `sudo`, is in [Building and Running KNIRVSERVER](#building-and-running-knirvserver).

Don't want to run your own node? Install the CLI instead:

```bash
npm install -g @knirv/cli
knirv network status --all-services
```

## What Is Implemented

The safest factual description of the current codebase:

- `KNIRVSERVER` is the launcher and edge router: it serves the embedded Next.js frontend, proxies API/WebSocket/SSE traffic to `backend_server`, provisions TLS, runs the self-updater, hosts the agent control plane, the Transaction/Validation chains and KNIRVMONITOR, and starts every other service
- `backend_server` (source in `KNIRV_CORP`) handles policy, guardrails, DVE management, the knowledge base, agent launch, shell sessions, evidence/proof anchoring, the CLI-supervisor relay, badges, and the actuarial syndicate; it in turn launches KNIRVGATEWAY, KNIRVCHAIN, KNIRVGRAPH, KNIRVORACLE, KNIRVHASHER, and KNIRVARENA
- `KNIRVGATEWAY` provides the public portal, DHT/TURN, tunnel registry, auth, payment, operator, URI services, and the omnichannel (Matrix) messaging bridge
- `KNIRVCHAIN` provides chain-side registry, P2P/discovery, mining, validation, wallet, and data-engine logic
- `KNIRVGRAPH` provides the knowledge graph: Network Resolution Vectors, ErrorNodes/SkillNodes, Proof-of-Solution, and NRN economics
- `KNIRVORACLE` provides root-node governance, checkpoints, and payouts; it only serves `/oracle/*` when an encrypted `root.key` is present

Anything beyond that should be treated as roadmap unless it is documented in code or an API spec.

## Architecture: One Entry Point, Twelve Packages

KNIRVCHAIN started as a single Go monolith. In August 2025 the architecture split into independent packages, each with its own `go.mod`, `Cargo.toml`, or `package.json` and no cross-package Go imports between services. Services talk to each other over HTTP, gRPC, or Unix sockets only, never a direct import.

KNIRVSERVER is where that split converges at runtime. A real deployment builds and runs exactly one binary, `packages/KNIRVSERVER/dist/knirv-server`. Embedding happens at two levels: the KNIRVSERVER launcher embeds the frontend, its config, `backend_server`, `ulorad`, KNIRVAGENT and KNIRVLLAMA; `backend_server` in turn embeds the network services through the `packages/KNIRVSERVER/pkg/knirv*` wrapper modules. Each wrapper module carries its compiled binary in `pkg/<name>/bin/`, extracts it to `<app-data>/bin/` and supervises it as a child process:

```
KNIRVSERVER launcher  (packages/KNIRVSERVER/main.go -> internal/launcher -> dist/knirv-server)
   the only binary you build and run directly
   |
   |-- in-process: HTTP router/proxy (:8090), TLS provisioning, self-updater,
   |               agent control plane, Transaction/Validation chains, KNIRVMONITOR
   |-- runs: text-embedder, IPFS, Xion   (optional helpers; failures are non-fatal)
   |-- runs: ulorad          (KNIRVULORA adapter compiler; started before the backend)
   |-- runs: knirvagent      (KNIRVAGENT runtime, managed per DVE by the agent control plane)
   |-- runs: knirvllama      (local llama.cpp provider; only with -llama or auto_start_llama)
   `-- runs: backend_server  (API on :8082; source lives in KNIRV_CORP)
         |-- runs: KNIRVGATEWAY   (portal, DHT, TURN, tunnels, auth, payments, bridge; :8080)
         |-- runs: KNIRVCHAIN     (P2P, mining, validation, wallet; chain.sock)
         |-- runs: KNIRVGRAPH     (knowledge graph, NRV, ErrorNodes/SkillNodes; graph.sock)
         |-- runs: KNIRVORACLE    (root-node governance; /oracle/* only with a root.key)
         |-- runs: KNIRVHASHER    (ASIC inference pipeline; only with -hasher / -pipeline)
         `-- runs: KNIRVARENA     (ERGO arena backend)
```

The launcher's start order is: agent control plane → production credential check and TLS (production only) → optional helpers → `ulorad` → `backend_server` (which brings up the oracle and the rest) → Transaction Chain (after the oracle is healthy, since it funds wallets) → Validation Chain → KNIRVMONITOR → HTTP server → KNIRVLLAMA (asynchronously, since first-run model provisioning can take minutes).

Nothing above talks to a shared database. Coordination happens over Unix sockets under `/var/lib/knirvserver/sockets/` (or `$KNIRV_APP_DATA_DIR/sockets/`) and a handful of TCP ports defined in `packages/KNIRVSERVER/config/*.yaml`.

## Packages

| Package | Stack | Role |
|---|---|---|
| `KNIRVSERVER` | Go + Next.js | Entry point. Launcher, router/proxy, TLS, updater, agent control plane, Transaction/Validation chains, KNIRVMONITOR (`internal/monitor`, the admin Network Monitor backend), embedded frontend. Also hosts the `pkg/knirv*` wrapper modules that embed every other service binary. |
| `KNIRVGATEWAY` | Go | Public portal, DHT/TURN, tunnel registry, auth, payments, operator and URI routing, omnichannel messaging bridge (vendored Tuwunel Matrix homeserver via `make vendor-tuwunel`). |
| `KNIRVCHAIN` | Go | Node and agent registries, P2P discovery, mining, validation, wallet, data engine, NFT/skill economics, `SYNDICATE_*` transaction types. |
| `KNIRVGRAPH` | Go + TS | Knowledge graph: NRV system, ErrorNodes/SkillNodes, Proof-of-Solution, NRN economics, React graph explorer. |
| `KNIRVORACLE` | Go | Root-node governance, checkpoints, failover, and payouts. Routes only mount when an encrypted `root.key` is present. |
| `KNIRVHASHER` | Go | Repurposed ASIC mining hardware (ex-Bitcoin miners) doing neural-network inference instead of hashing, plus its data/training pipeline. |
| `KNIRVAGENT` | Go | Workspace-aware, tool-using agent runtime for CLI sessions, messaging channels, and managed per-DVE processes. Can use the local `knirvllama` provider. |
| `KNIRVULORA` | Go (+ Python engines) | `ulorad` adapter-compiler daemon and `ulora` CLI: safetensors handling, per-model-family connectors, adapter bundling. Exposed to other services over `ulora.sock`. |
| `KNIRVINFERENCER` | Go library | Shared LLM provider layer (Anthropic, Cerebras, DeepSeek, Gemini, llama) with fallback switching, conversation memory, and context strategy. Imported by `backend_server` as `github.com/guiperry/knirv-inference-go`; not a running service. |
| `KNIRVARENA` | TS / React / Three.js | ERGO (Error Resolution Gaming Operation): the 3D client where human architects submit training data against live error nodes. Its backend is embedded via `pkg/knirvarena`. |
| `KNIRVBASE` | Go / Rust / TS | Local-first database and sync layer: document collections, CRDTs, vector clocks, peer sync, NRV streaming. Go is the reference implementation. |
| `KNIRVSDK` | Rust core + C ABI / Go / Python / npm | Developer SDKs for KNIRV services, canonical signing formats, and verified WASM modules. (The KNIRV CLI is **not** here; it lives in `KNIRV_CORP/packages/cli`.) |

KNIRVLLAMA has no source package of its own here: `packages/KNIRVSERVER/pkg/knirvllama` wraps a llama.cpp server binary built by `make llama-build`.

### Websites

| Path | What it is |
|---|---|
| `websites/KNIRV.NETWORK` | Static marketing, docs, and product pages for knirv.network. |
| `websites/REGISTRY.KNIRV.NETWORK` | Cloudflare Worker: network node registry and failover control plane. Stores node registrations, validator heartbeats, votes, and state transitions in a Durable Object. Consumed by KNIRVGATEWAY (DHT bootstrap) and KNIRVCHAIN (self-registration). |
| `websites/SYNDICATE.KNIRV.NETWORK` | Static front end for the KNIRV Syndicate bounty network (the actuarial syndicate API lives in `backend_server`). |

## Building and Running KNIRVSERVER

### Prerequisites

- Go 1.25+ (KNIRVGATEWAY targets 1.26; other packages range from 1.23 to 1.25, so check each `go.mod` if you hit a toolchain mismatch)
- Node.js 18+, only if you're rebuilding a frontend or TypeScript package
- Optional: a sibling checkout of the private `KNIRV_CORP` repo at `../KNIRV_CORP`, needed only to rebuild `backend_server` from source. The Makefile finds it automatically (or via `KNIRVSERVER_WORKSPACE_DIR`). The compiled, gzipped binary already ships in this repo as `packages/KNIRVSERVER/bin/backend_server`, and the other service binaries live in `packages/KNIRVSERVER/pkg/knirv*/bin/`, so a plain build works without it.

### Build

```bash
cd packages/KNIRVSERVER
go build -o dist/knirv-server .
```

This links the already-built `backend_server` (which itself carries the gateway, chain, graph, oracle, hasher, and arena binaries), `ulorad`, `knirvagent`, `knirvllama`, the `config/*.yaml` files, and the built frontend export in `frontend/out/` into one Go binary. Nothing downstream needs recompiling, so this takes seconds. `make relink` does the same after a component change.

For a full from-source rebuild of every embedded package (requires the sibling `KNIRV_CORP` repo for `backend_server`'s source):

```bash
cd packages/KNIRVSERVER
make binary
```

`make binary` regenerates protobuf and eBPF code, builds the frontend, builds each service into its `pkg/*/bin/`, compiles and gzips `backend_server` into `bin/`, links `dist/knirv-server`, and copies the result into `KNIRV_CORP/packages/server/container_deployer` for image builds.

### Run

KNIRVSERVER defaults to **testnet mode**. There is no `--testnet` flag; testnet is simply what you get unless you pass `-prod`, `-dev`, or `-ent`.

```bash
sudo ORACLE_KEY_PASSWORD=<your-oracle-key-password> ./dist/knirv-server -hasher
```

By default the binary reads config from `/etc/knirv-server` and writes runtime state to `/var/lib/knirvserver`, which is why `sudo` is normally required. `ORACLE_KEY_PASSWORD` decrypts the node's `root.key`, which the loader looks for at `~/.config/knirv-server/.key/root.key` (or `~/.config/knirv-server/root.key`; under `sudo`, the invoking user's home is checked too). That unlocks the `/oracle/*` routes. With no key file, no password is needed and the server runs fine as a non-root node. A bootnode's `boot.key` is decrypted with `BOOT_KEY_PASSWORD` and also supplies the Cloudflare credentials used for production TLS. `-hasher` is optional and starts the KNIRVHASHER ASIC driver alongside everything else.

To run locally without `sudo`, point the app-data and config directories at somewhere you own:

```bash
mkdir -p .local/data .local/config
KNIRV_APP_DATA_DIR="$(pwd)/.local/data" \
KNIRV_CONFIG_DIR="$(pwd)/.local/config" \
./dist/knirv-server
```

Other flags: `-config <path>`, `-dev`, `-prod`, `-ent` (enterprise node at `enterprise-{tag}.knirv.network`), `-user-id-tag <tag>`, `-port <n>`, `-host <addr>`, `-pipeline` (start the KNIRVHASHER data pipeline), `-direct` (drive the ASIC over raw USB instead of CGMiner), and `-llama` (start the local llama.cpp provider).

### Verify it's running

```bash
curl http://localhost:8090/health
```

`8090` is the KNIRVSERVER wrapper's own port (`config/testnet.yaml`). The embedded backend API listens on `8082`, the embedded KNIRVGATEWAY on `8080`, P2P on `4001`, and metrics on `9090`. KNIRVCHAIN, KNIRVGRAPH, ULoRA, and KNIRVMONITOR communicate over Unix sockets rather than TCP. All of it runs under one process tree started by KNIRVSERVER.

### Or use the Makefile

```bash
make testnet-build   # builds packages/KNIRVSERVER/dist/knirv-server
make testnet-start   # starts it in the background and health-checks it
make testnet-status  # check status
make testnet-stop    # stop it
```

## Deployment

Everything in [Building and Running KNIRVSERVER](#building-and-running-knirvserver) above covers running the binary directly on a host you already have. Production and containerized rollout (including the box behind `testnet-gateway.knirv.com`) goes through tools that live in the private `KNIRV_CORP` repo: `packages/server/os_builder`, `packages/server/container_deployer`, and the installer built into the desktop client at `packages/client` (`knirv-client install`; it replaced the former standalone `image_installer`). This repo doesn't automate that pipeline; this section documents how it works.

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

By default it pushes the result to Docker Hub (`knirvcorp/knirvserver:debian-latest` / `:kali-latest`); pass `--push=false` to start it locally instead, or `--build-only` to just produce the image. The binary it bundles is the one `make binary` copied into the deployer's source directory. Other options: `--deploy-mode container|kata|native`, `--deploy-type local|cloud`, `--env development|testnet|production`.

### 3. Run it and bridge in the node key (`knirv-client install`)

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

`$DATA_DIR/.key/` is the filesystem bridge: drop `root.key` (root/oracle node), `enterprise.key` (enterprise operator node), or `boot.key` (bootnode) there on the host, and it shows up read-only inside the container at the fixed path KNIRVSERVER's key loader expects. Keys are produced (encrypted) by `KNIRV_CORP/packages/server/key_maker`. No key file present is a normal, supported state; the server just runs without oracle/root routes.

The KNIRV desktop client automates exactly this with `knirv-client install` (`--edition pro|enterprise`, `--mode local|cloud`, `--env`, `--data-dir`, `--ssh-*`). It discovers and stages a key file into `$DATA_DIR/.key/`, runs the equivalent `docker run` locally or over SSH, and polls `/health` until the container reports ready. The install path is gated: it only runs for an Enterprise or Investor account verified live against the Controller gateway.

## Public Testnet and the CLI

You don't need to run your own node to try the network. The public testnet is live at `testnet-gateway.knirv.com`, and **KNIRV-CLI is the flagship way to reach it**, or any node you run yourself:

```bash
npm install -g @knirv/cli

# or grab a standalone binary from releases.knirv.com/cli/<platform>/knirv

knirv network status --all-services
knirv economics balance --address 0x... --include-pending
knirv mcp nrv submit-error --auto-resolution --skill-suggestion
```

KNIRV-CLI is a single client for every layer of the network: service discovery with health checks and a circuit breaker, wallet support with gasless XION meta accounts, an NRN token manager, and an interactive shell. It is also the local DVE supervisor: `knirv supervisor` runs the embedded Zot supervisor over Claude/Codex/OpenCode/Hermes sub-agents, and `knirv git commit` / `knirv git push` turn supervised work into signed, encrypted proofs anchored to a KNIRVCHAIN receipt through KNIRVSERVER. Its source lives in `KNIRV_CORP/packages/cli`; see that package's README and [`docs/whitepapers/KNIRV-CLI_Whitepaper.md`](./docs/whitepapers/KNIRV-CLI_Whitepaper.md) for the full command surface.

## Repository Layout

| Path | What's there |
|---|---|
| `packages/` | The 12 packages listed above. |
| `Makefile` | Root targets: `testnet-*`, `health-check`, per-package `build-*`, `build-all`, `vendor-tuwunel`, `tests`, `build-modp`. Run `make help`. |
| `integration-tests/` | Go integration tests that hit real running services. No mocks. |
| `modp/` | Formal verification models written in the P language, checked with PChecker. |
| `docs/` | Whitepapers, architecture notes, and implementation guides. |
| `scripts/` | Sync, deployment, and local testnet management scripts. |
| `websites/` | KNIRV.NETWORK, REGISTRY.KNIRV.NETWORK, and SYNDICATE.KNIRV.NETWORK (see [Websites](#websites)). |
| `shared-proto/` | Shared Protobuf definitions (agent, chain, graph, hasher, lora, memory, signing, ...). |
| `test-reports/` | Generated test output. |

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
cd packages/KNIRVULORA && go test -v ./...
cd packages/KNIRVINFERENCER && go test -v ./...

# Or everything at once from the repo root
make tests

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
- No cross-package Go imports between services; inter-service communication is HTTP, gRPC, or Unix sockets only. The exceptions are the deliberate `replace` directives that let `backend_server` import the `packages/KNIRVSERVER/pkg/knirv*` wrapper modules and the KNIRVINFERENCER library
- TypeScript: prefer `unknown` over `any`
- Integration tests hit real services; no mock DB or mock network
- Oracle routes only mount when an encrypted `root.key` is present in `~/.config/knirv-server/.key/`
- Keep secrets out of the repo; use `ORACLE_KEY_PASSWORD`, `BOOT_KEY_PASSWORD`, and friends as environment variables

## Documentation Index

- [`packages/KNIRVSERVER/README.md`](./packages/KNIRVSERVER/README.md)
- [`../USER_WORKFLOWS_AND_PRODUCTION_PLAN.md`](../USER_WORKFLOWS_AND_PRODUCTION_PLAN.md) (workspace root)
- [`packages/KNIRVSERVER/server-api.yaml`](./packages/KNIRVSERVER/server-api.yaml)
- [`integration-tests/README.md`](./integration-tests/README.md)
- [`packages/KNIRVGATEWAY/README.md`](./packages/KNIRVGATEWAY/README.md)
- [`packages/KNIRVAGENT/README.md`](./packages/KNIRVAGENT/README.md)
- [`packages/KNIRVBASE/README.md`](./packages/KNIRVBASE/README.md)
- [`packages/KNIRVSDK/README.md`](./packages/KNIRVSDK/README.md)
- [`websites/REGISTRY.KNIRV.NETWORK/README.md`](./websites/REGISTRY.KNIRV.NETWORK/README.md)
- [`packages/KNIRVCHAIN/README.md`](./packages/KNIRVCHAIN/README.md)
- [`modp/README.md`](./modp/README.md)
- [`../KNIRV_CORP/README.md`](../KNIRV_CORP/README.md) (sibling repo)
- [`docs/whitepapers/KNIRV-CLI_Whitepaper.md`](./docs/whitepapers/KNIRV-CLI_Whitepaper.md)

## Roadmap Boundary

The repo still contains long-range language about sovereign layers and network flywheels in places. Treat that as roadmap language unless a specific file or endpoint backs it up. The current codebase is strong enough to document as a guarded execution and coordination platform; it is not accurate to describe every aspirational layer as complete.

## Contributing

See [Conventions](#conventions) above, and keep this README in sync when entry points, ports, or build commands change.

## License

GPL-3.0
