# KNIRVGATEWAY

KNIRVGATEWAY is the public gateway and portal layer for the KNIRV Network. It is responsible for routing browser and API traffic to the right internal service, serving the embedded web GUI, and exposing the network utilities that sit in front of the chain and control-plane components.

The current implementation is a gateway, not the source of truth for business logic:

- it proxies requests to oracle, chain, tunnel, and network-monitor services
- it exposes DHT, TURN, auth, operator, payment, and URI routes
- it serves the embedded web portal and static UI
- it provides session and session-controller handling

## What It Currently Does

The gateway server in `internal/server/server.go` wires together these services:

- session manager
- auth handler
- operator registry
- tunnel registry and TURN server
- payment service
- URI generation and resolution
- embedded web GUI
- DHT manager
- proxy layer for internal services

The most important idea is that the gateway coordinates access and routing. It does not own the deeper domain logic.

## Route Families

The main route groups are:

- `/health`
- `/session/controller`
- `/dht/*`
- `/p2p/*`
- `/knirvbase/*`
- `/auth/*`
- `/operator/*`
- `/tunnel/*`
- `/payment/*`
- `/uri/*`
- `/webgui/*`
- `/api/*`

The gateway also serves the embedded portal and proxies selected paths into the network control plane.

## Embedded Web GUI

The gateway contains an embedded web GUI under `internal/embedded/webgui/`.

That UI includes pages and API helpers for:

- dashboards
- chain and graph explorers
- oracle views
- network monitor views
- wallet and marketplace surfaces
- capability and skill pages
- validation and settlement views
- admin and settings pages

The portal is shipped as part of the repo, not as a separate application.

## Omnichannel Messaging Bridge

`internal/bridge/` runs a self-hosted, non-federating Matrix homeserver (vendored [Tuwunel](https://github.com/matrix-construct/tuwunel) binary at `internal/bridge/bin/tuwunel`) fronted by a Matrix appservice and a WhatsApp bridge (`whatsmeow`). It stores its own state in the same SQLite database whatsmeow already requires — no separate bridge database. Inbound customer messages are relayed, signed, to a connected CLI Supervisor session; when more than one team member is active, routing between them is decided by KNIRVSERVER's `supervisorrouting` package, not a hardcoded default.

The bridge starts last in `cmd/gateway/main.go`'s init sequence, after every other gateway service. It is configured through the `Bridge*` fields in `internal/config/`.

To rebuild the vendored Tuwunel binary from source:

```bash
make vendor-tuwunel
```

See `internal/bridge/bin/README.md` for the vendoring details.

## Configuration And Runtime

The gateway is configured through `internal/config/` and related env/config files.

Important runtime concerns:

- DHT can be enabled or disabled
- TURN can be enabled or disabled
- oracle access is proxied through the gateway’s configured socket or URL
- payment and operator services are initialized in-process
- WebSocket and proxy traffic are handled through the gateway router

## Related Files

Start here when changing gateway behavior:

- `cmd/gateway/main.go`
- `internal/server/server.go`
- `internal/server/middleware.go`
- `internal/proxy/handlers.go`
- `internal/dht/`
- `internal/turnserver/`
- `internal/bridge/`
- `internal/payment/`
- `internal/uri/`
- `internal/webgui/`
- `internal/embedded/webgui/`

## Testing

Common test entry points:

```bash
cd packages/KNIRVGATEWAY && go test -v ./...
```

The embedded portal also has its own tests and scripts under `internal/embedded/webgui/`.

## Documentation Boundary

If a route, page, or service is not present in `internal/server/server.go` or the embedded web GUI tree, treat it as planned or historical, not current.
