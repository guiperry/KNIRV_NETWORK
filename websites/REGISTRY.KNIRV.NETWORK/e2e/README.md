# Registry end-to-end test

Exercises bootnode validation against the real registry code, running in
workerd (the Cloudflare Workers runtime) via `wrangler dev`, with mock
bootnodes on loopback.

```bash
cd websites/REGISTRY.KNIRV.NETWORK
node e2e/run.mjs
```

Takes about 15 seconds. Exits `0` if every check passes, `1` otherwise. It needs
no Cloudflare login, deploys nothing, and uses ports `18788` (registry),
`14001-14004` (mock P2P) and `18081-18084` (mock `/health`).

## What it does

1. Copies `src/index.js`, `src/validation.js` and `wrangler.jsonc` into
   `e2e/.build/` (gitignored, deleted on exit) and applies the two patches
   below. `src/` itself is never modified.
2. Starts four mock bootnodes. Each has a TCP listener (the "P2P port") and an
   HTTP `/health` endpoint shaped like KNIRVGATEWAY's
   (`chainId`, `dht.peer_id`, `dht.self_addrs`).
3. Starts the patched registry with `wrangler dev --local` and registers each
   mock over HTTP.

## Scenarios

| Mock | Setup | Expected |
|---|---|---|
| `chain-good` | Everything correct | `201`, appears in `/bootnodes`, `/nodes` and `/status`; re-registering reuses the cached validation (no new health probes) |
| `chain-bad` | `/health` reports a different `chainId` | `422`, `health` fails, `connectivity` passes, not listed |
| `chain-closed` | P2P port not listening | `422`, `connectivity` fails; an immediate retry gets `429` with `Retry-After` |
| `chain-peer` | `/health` `peer_id` differs from the registered `nodeID` | `422`, `health` fails |
| no `fqdn` | Bootnode registers without a hostname | `422` |
| client | `type: "client"` | `201`, no validation (clients are unchanged) |
| `POST /validate` | The onboarding site's request shape | All four checks pass for the good node; `connectivity` fails for the closed port |

## What is real and what is patched

Real (shipped code, unchanged): the register gate and its 422/429 responses,
the TCP probe via `cloudflare:sockets`, the `/health` identity checks, the
latency check, the result cache and failure cooldown, the `/bootnodes` and
`/nodes` filter, the `/status` page, and `POST /validate`.

Patched in the `e2e/.build` copy only, because loopback cannot satisfy them:

- `isPublicIP` accepts any IPv4 literal (the real one refuses `127.0.0.1`).
- The HTTPS health probe goes to `http://127.0.0.1:<port>/health` instead of
  `https://<fqdn>/health`. The port comes from the first label of the fqdn
  (`p18081.knirv.com` probes port `18081`). The `.knirv.com` hostname allowlist
  still applies.

`run.mjs` throws if a patch target is not found, so a refactor of
`validation.js` fails loudly instead of silently testing unpatched code.

## What this does not prove

- **TLS.** The security check is only exercised over plain HTTP here. Real
  certificate verification needs a real HTTPS host.
- **Cloudflare's network.** Local workerd can open sockets that deployed Workers
  may not. Raw TCP from deployed Workers was checked separately with the
  `tcp-spike` Worker: it works to arbitrary public IPs and ports, but not to
  Cloudflare-owned IPs. A bootnode whose registered IP is behind Cloudflare
  will fail `connectivity`.
- **The gateway.** Whether KNIRVGATEWAY sends `fqdn` in its registration is
  covered by the Go tests in `KNIRVGATEWAY/internal/dht`, not here.

For comparison, the unit tests in `src/__tests__/validation.test.js` cover the
gate logic with fake probes; this test covers the same logic through a running
Worker with real sockets.

To test against real infrastructure, deploy the registry to a `workers.dev`
name and register a bootnode that has a public IP, an open P2P port and a
`.knirv.com` hostname serving `/health` over valid HTTPS.
