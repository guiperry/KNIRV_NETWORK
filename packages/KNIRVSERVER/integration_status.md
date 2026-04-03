# Integration Status

## Scope

This document tracks the current integration state of `KNIRVSERVER`, `KNIRVCHAIN`, `KNIRVGRAPH`, and `KNIRVGATEWAY` after the Unix-socket migration work and recent startup verification runs.

## Current Summary

- `KNIRVSERVER` backend wrapper is successfully starting in development mode and binding its API to a Unix socket.
- `KNIRVGATEWAY` is successfully starting on a Unix socket and no longer needs to expose its internal HTTP listener directly for wrapper-to-gateway communication.
- `KNIRVCHAIN` now receives socket-aware launch configuration from `KNIRVSERVER` and its main blockchain HTTP server now comes up on the requested Unix socket.
- `KNIRVGRAPH` starts, but its manager/config path still reports zero-valued ports and its health model does not align cleanly with actual startup behavior.
- Oracle initialization is still failing at runtime because `root.key` is present but `ORACLE_KEY_PASSWORD` is not available to decrypt it non-interactively.

## Confirmed Improvements

- `KNIRVSERVER` now defaults to development mode for launcher runs.
- `KNIRVSERVER` re-extracts embedded binaries/configs even when older extracted copies already exist.
- `KNIRVSERVER` backend proxying is Unix-socket aware.
- `KNIRVGATEWAY` source changes were synced into `KNIRVSERVER` and rebuilt.
- `KNIRVCHAIN` source changes were synced into `KNIRVSERVER` and rebuilt.
- `KNIRVCHAIN` now persists `socket_path` in its saved minimal config.
- `KNIRVCHAIN` manager launch now uses KNIRV-prefixed env vars and explicit role/socket flags.
- `KNIRVSERVER` now forces the embedded `KNIRVCHAIN` subprocess into headless mode with `-gui=false`, avoiding the most TCP-biased startup branch for internal service launches.
- `KNIRVCHAIN` DataEngine no longer starts standalone WebSocket/REST listeners when the chain itself is running in Unix-socket mode.
- `KNIRVCHAIN` DataEngine WebSocket server now honors its configured port instead of always binding `:8080`.
- `KNIRVCHAIN` now logs `Socket mode requested at startup` and `Embedded runtime socket path override active`, confirming the embedded launch flags/env are reaching the child process.
- `KNIRVCHAIN` now prepares and starts its blockchain server on `/home/gperry/.local/share/knirvserver/sockets/chain.sock`, and the backend manager health check passes against that socket-backed service.

## Inconsistencies

### 1. KNIRVCHAIN launch intent and blockchain server runtime now align, but not every sidecar follows that model yet

- `KNIRVSERVER` launches `KNIRVCHAIN` with:
  - `port: 8083`
  - `p2p_port: 4001`
  - `socket_path: /home/gperry/.local/share/knirvserver/sockets/chain.sock`
- Current runtime now logs:
  - `Socket mode requested at startup: /home/gperry/.local/share/knirvserver/sockets/chain.sock`
  - `Embedded runtime socket path override active: /home/gperry/.local/share/knirvserver/sockets/chain.sock`
  - `BlockchainServer for chain testnet prepared for socket /home/gperry/.local/share/knirvserver/sockets/chain.sock`
  - `Starting blockchain HTTP server on socket /home/gperry/.local/share/knirvserver/sockets/chain.sock...`
- The backend manager also now reports:
  - `KNIRVCHAIN health check passed`

Impact:
- The main KNIRVCHAIN API listener is now preserving socket-only intent end to end.
- Remaining inconsistencies are now concentrated in auxiliary services and install/startup assumptions rather than the blockchain server itself.

### 2. KNIRVCHAIN still contains TCP/public-port assumptions throughout its codebase

- Multiple subsystems still assume a TCP base URL or public port:
  - wallet server blockchain URL uses `http://localhost:<port>`
  - GUI backend URL update logic writes `http://<public-ip>:<port>`
  - reverse proxy config defaults still point to `:8080`
  - several logs and runtime branches assume “starting on port”

Impact:
- Even when socket mode is requested, downstream logic still tries to reconstruct a TCP-centric topology.

### 3. KNIRVCHAIN single-node GUI path appears to reconstruct or preserve stale TCP expectations

- The single-node GUI path is the most problematic branch.
- Embedded launcher runs now log:
  - `GUI is disabled for Single Node.`
  - `Starting Main Node (HTTP: 8083, P2P: 4001, GUI: false, DB: ...)`

Impact:
- The GUI/preinitialized path is still a risk area in the codebase, but it is no longer on the embedded KNIRVSERVER critical path.
- Current mitigation:
  - embedded launches from `KNIRVSERVER` are now forced headless so they use the simpler service path while deeper GUI-path cleanup continues.

### 4. KNIRVCHAIN installation state is still logically incomplete every run

- Startup still reports:
  - `No wallet file found and MinersAddress not configured.`
  - `This node requires reinstallation. Forcing installer.`
- `-skip-install` now prevents a full installer reset, but the node is still considered uninstalled each run.

Impact:
- Startup remains noisy and brittle.
- Any future config-save or install-side mutation can reintroduce drift.

### 5. KNIRVCHAIN’s command entrypoint still contains placeholder/stubbed functions

- `cmd/knirvchain/main.go` still includes placeholder implementations for things such as:
  - `Install`
  - `LoadPaymentProcessorConfig`
  - `NewGoReverseProxy`
  - `NewLevelDB`
  - `NewBlockchain`

Impact:
- This makes the command package internally inconsistent and hard to reason about.
- Some runtime paths may be using placeholder behavior rather than real subsystem integrations.

### 6. KNIRVCHAIN DataEngine still assumes a public TCP listener

- Previously, startup logged:
  - `WebSocket server error: listen tcp :8080: bind: address already in use`
  - `Starting REST API server on :7080...`
- Root cause:
  - the WebSocket server ignored its configured port and always bound `:8080`
  - standalone DataEngine HTTP listeners were still starting even when `KNIRVCHAIN` was launched as an internal socket-backed service
- Current status:
  - the blockchain server no longer falls back to TCP and now binds to its Unix socket correctly
  - however, `DataEngine started successfully` still appears in socket mode without the expected “disabled” log line, so this area still needs one more pass if we want to fully guarantee no standalone sidecar listeners

Impact:
- The main socket migration goal is met for KNIRVCHAIN’s API server.
- DataEngine remains the main residual risk for stray TCP listeners.

### 7. KNIRVCHAIN GUI backend URL update is not socket-aware enough

- Startup still tries to write:
  - `NEXT_PUBLIC_BACKEND_URL=http://<public-ip>:8080`
- It also warns that the target file is missing:
  - `...backend.config: no such file or directory`

Impact:
- The browser-facing config update path is both path-fragile and TCP-oriented.
- In a socket-proxied architecture, the browser should generally receive a wrapper/gateway URL, not a raw internal service URL.

### 8. KNIRVCHAIN chain identity and runtime config sources disagree

- Manager launch intends `chain_id: "testnet"`.
- Child process now loads and retains `ChainID: 'testnet'` after runtime overrides.

Impact:
- Chain identity is now being controlled consistently by the embedded launcher.
- This reduces DHT/service-discovery drift.

### 9. KNIRVGRAPH manager is still starting with zero-valued ports

- The manager logs:
  - `port:0`
  - `p2p_port:0`
  - `api_port:0`

Impact:
- This is still a config propagation bug.
- Even if KNIRVGRAPH self-heals at runtime, the manager’s model of the service is wrong.

### 10. KNIRVGRAPH health model does not match actual startup behavior

- Runtime logs show graph startup activity and a DHT node, but `KNIRVSERVER` still later warns:
  - `KNIRVGRAPH did not become healthy within timeout`

Impact:
- Startup health checks and actual operational readiness are not aligned.

### 11. Oracle startup is expected but not satisfiable in the current launcher flow

- `root.key` is present.
- Startup logs show:
  - `Failed to load secrets from root.key`
  - `Failed to initialise oracle — continuing without it`

Impact:
- This is not a disabled oracle. It is a failed oracle initialization.
- Root-node behavior is therefore incomplete and inconsistent with the presence of `root.key`.

### 12. KNIRVGATEWAY is socket-bound internally but still carries a stale `port:8080` identity

- Gateway startup logs include both:
  - `socketPath: .../gateway.sock`
  - `port:8080`

Impact:
- That port is no longer the internal listener of record.
- It is now just an external/default/public identity hint, which is confusing unless explicitly modeled as such.

### 13. KNIRVGATEWAY extracts the `network-website`, but KNIRVSERVER does not currently expose that site surface via wrapper routing

- Gateway server code serves the network website at `/`.
- In the integrated runtime, the gateway itself listens on a Unix socket.
- The wrapper mainly proxies backend API/WS traffic and serves its own frontend.

Impact:
- The network website may exist internally but not be reachable from the public wrapper unless explicit proxy routes are added.

### 14. Desktop/runtime environment still adds unrelated graphics noise

- Electron/graphics warnings still appear:
  - `bad option: --disable-gpu`
  - `bad option: --disable-software-rasterizer`
  - `libva`, `vkCreateInstance`, `libEGL` warnings

Impact:
- These are probably not core integration failures, but they clutter startup debugging and can hide real service issues.

## Feasibility Report: Gateway Proxy Strategy

### Intent

The intended architecture is:

- all internal services bind to Unix sockets
- `KNIRVGATEWAY` becomes the public fan-out layer
- selected gateway ports expose public HTTP/TCP/UDP entry points that proxy or relay to internal socket-backed services

### Feasibility

This strategy is feasible, with caveats.

### What already works in favor of the strategy

- `KNIRVSERVER` can already proxy to an internal Unix-socket backend.
- `KNIRVGATEWAY` itself already supports binding its own HTTP surface to a Unix socket.
- Gateway already has explicit concepts for public TURN/tunnel ports and internal service coordination.
- Browser-facing traffic should not talk to raw sockets directly, so a gateway/wrapper public proxy layer is architecturally appropriate.

### Strong benefits

- Internal services are no longer directly exposed on arbitrary localhost/public ports.
- Port ownership becomes centralized in one outward-facing process.
- Security posture improves because Unix sockets are local-only and file-permission controlled.
- The public topology becomes simpler:
  - wrapper/public frontend
  - gateway/public service ports
  - internal socket fabric behind them

### Main engineering requirements

1. Every internal service must stop assuming it owns a public HTTP port.
2. Every browser/client-facing URL generator must stop emitting raw internal ports.
3. Gateway must gain explicit upstream mappings for each public route/port to the correct internal socket target.
4. Health checks must become socket-aware and must verify the actual route that the gateway/wrapper depends on.
5. Service discovery and config persistence must preserve socket paths just as reliably as ports.

### Primary blockers visible today

1. `KNIRVCHAIN` still internally reconstructs TCP/public-port behavior.
2. `KNIRVGRAPH` manager config is still inconsistent.
3. `KNIRVGATEWAY` does not yet appear to proxy internal service root/static surfaces outward through the wrapper.
4. Browser config generation in submodules still assumes direct HTTP ports.
5. Some sidecar services such as DataEngine WebSocket/REST still bind directly to TCP.

### Recommended proxy model

- `KNIRVSERVER` remains the desktop/wrapper frontend entrypoint on its own public port.
- `KNIRVSERVER` proxies backend API and WebSocket traffic to backend Unix sockets.
- `KNIRVGATEWAY` remains a separate internal service bound to its own Unix socket.
- `KNIRVSERVER` or `KNIRVGATEWAY` should expose explicit public routes for:
  - chain explorer / network website
  - gateway dashboard
  - TURN API
  - tunnel registry/control APIs
- TURN/STUN/relay ports remain true network ports because those protocols are not browser HTTP proxy traffic.

### Recommended implementation direction

1. Finish removing `KNIRVCHAIN` TCP assumptions.
2. Fix `KNIRVGRAPH` config/health alignment.
3. Add explicit wrapper-to-gateway reverse proxy routes for selected gateway surfaces.
4. Treat public ports as gateway-owned assignments, not submodule-owned defaults.
5. Make generated frontend URLs resolve to wrapper/gateway public routes, never internal service ports.

### Concrete proxy shape that looks feasible

- Keep `KNIRVSERVER` on a single user-facing desktop/web port such as `8090`.
- Keep `KNIRVGATEWAY` internal on a Unix socket.
- Add explicit wrapper routes that proxy from `KNIRVSERVER` to `KNIRVGATEWAY`'s socket for:
  - `/network`
  - `/gateway`
  - `/turn`
  - `/tunnel`
- Let `KNIRVGATEWAY` own only true public protocol ports that cannot be cleanly multiplexed through the wrapper:
  - TURN UDP/TCP
  - STUN
  - relay/control listeners
- Keep `KNIRVCHAIN` and `KNIRVGRAPH` private on sockets and never expose their raw internal URLs to browsers.

### Overall assessment

The proxy strategy is technically sound and fits the architecture you want.

The main risk is not feasibility; it is consistency.

Until each internal service becomes fully socket-native and stops regenerating public-port assumptions, the gateway cannot be the single trustworthy outward-facing proxy layer.

## Recommended Next Steps

1. Finish the `KNIRVCHAIN` socket audit in the single-node GUI path and DataEngine sidecars.
2. Patch `KNIRVCHAIN` browser/backend URL generation so socket mode resolves to wrapper/gateway routes instead of raw internal ports.
3. Fix `KNIRVGRAPH` manager defaults and health target alignment.
4. Add explicit `KNIRVSERVER` routes that proxy selected gateway website/dashboard surfaces out from the gateway socket.
5. Decide which public ports should belong to `KNIRVGATEWAY` versus the wrapper, and document those assignments centrally.
