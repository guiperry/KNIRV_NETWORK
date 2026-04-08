
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
