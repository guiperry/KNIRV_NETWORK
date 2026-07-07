# KNIRVSERVER

KNIRVSERVER is the main execution-control service in the KNIRV Network. It serves the embedded frontend, routes API traffic, proxies long-lived connections, and exposes the current guardrail / DVE / knowledge-base surface of the platform.

The current codebase supports a guarded execution model rather than a generic app server:

- policies and guardrails for AI workloads
- DVE lifecycle and agent execution
- knowledge base and GraphRAG-backed retrieval surfaces
- onboarding and org configuration
- shell/session access for controlled execution
- evidence anchoring and other audit-oriented flows
- WebSocket and SSE streaming for live workflows

## Runtime Layout

`packages/KNIRVSERVER/main.go` is the actual entry point.

At runtime the process does four main things:

1. serves the embedded frontend export
2. proxies API traffic to the backend subprocess or gateway socket
3. handles local-only routes such as health, version, and update control
4. forwards WebSocket and streaming requests without buffering them away

That means the server acts as a router and host, not just a normal API process.

## What It Exposes

The canonical API inventory lives in [`server-api.yaml`](./server-api.yaml).

Current API groups include:

- health and system info
- auth and user management
- DVE and DVE creation
- agent launch, tasks, and status
- guardrails and policy management
- onboarding
- cognitive engine
- shell and session management
- payments
- evidence anchoring
- TEE support
- memory and knowledge base
- ontology, Nexus, analytics, and data-engine surfaces

The most useful mental model is:

- policy and guardrail control at the center
- DVE execution around it
- knowledge/agent tooling around that
- proxying and streaming at the edge

## Important Routes

The exact set is documented in the OpenAPI spec, but the major route families are:

- `/health` and `/version`
- `/api/v1/system/*`
- `/api/v1/auth/*`
- `/api/v1/onboarding/*`
- `/api/v1/dve/*`
- `/api/v1/agent/*`
- `/api/v1/shell/*`
- `/api/v1/knowledge-base/*`
- `/api/v1/cognitive/*`
- `/api/v1/payments/*`
- `/api/v1/workflow/*`
- `/api/v1/anchoring/*`
- `/ws/*`

The server also proxies:

- `/network`
- `/explorer`
- `/gateway`
- `/turn`
- `/dve`
- `/arena`
- `/tunnel`
- `/api/*`

## Frontend Surface

The embedded frontend is built and served from `frontend/out/`.

Notable UI areas in the current tree include:

- DVE dashboards and node views
- policy and guardrail editors
- knowledge-base and DVE management screens
- onboarding flows
- agent command surfaces
- system and network monitoring views
- compliance-oriented dashboards

The server serves static assets and also supports the interactive routes needed by the browser app.

## Operational Details

- The root key is embedded only on root-node builds.
- Oracle-related behavior depends on `packages/KNIRVSERVER/bin/root.key` and `ORACLE_KEY_PASSWORD`.
- The server exposes local update status and update apply routes.
- Streaming requests are detected and proxied without a normal timeout.
- WebSocket traffic is proxied to the backend subprocess.

The main file to inspect when changing behavior is `main.go`.

## Related Docs

Use these docs together when making changes:

- `server-api.yaml` for endpoint truth
- `Production_Status_Report.md` for current implementation notes
- `USER_WORKFLOWS_AND_PRODUCTION_PLAN.md` for end-to-end flows
- `missing_flows.md` for gaps and unverified routes
- `docs/` for deeper subsystem notes

## Testing

The server has several layers of validation:

- Go unit and integration tests under `packages/KNIRVSERVER`
- frontend tests under `packages/KNIRVSERVER/frontend`
- Playwright and Jest coverage for UI surfaces
- repo-level integration tests in `integration-tests`
- formal checks in `modp`

Common commands:

```bash
cd packages/KNIRVSERVER && go test -v ./...
cd packages/KNIRVSERVER/frontend && npm test
cd integration-tests && go test -v ./...
```

## Implementation Notes

A few things are important when reading the code:

- the repo contains both code and analysis docs; docs are not always a source of truth
- some features are implemented behind configuration or only in specific build modes
- some route families are proxied rather than implemented directly in the Go process
- if a feature is not in `server-api.yaml` or the current code, treat it as planned

## Where To Start

If you want to change the server safely, start in this order:

1. `main.go`
2. `server-api.yaml`
3. `Production_Status_Report.md`
4. the relevant frontend component under `frontend/src/`
5. the matching test in `frontend/tests/` or `integration-tests/`

That path keeps implementation, API surface, and docs aligned.
