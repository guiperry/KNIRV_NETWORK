# KNIRV Network Registry Worker

This project deploys the KNIRV node-registration and failover-control HTTP API to
Cloudflare Workers. Node state is stored consistently in a Durable Object and
expires after one hour. Validator heartbeats, votes, and state transitions are
also stored here with shorter TTLs and signature verification.

## Deploy

```bash
npm install
npm run build
npm run deploy
```

Set the production custom domain in Cloudflare or add a `routes` entry to
`wrangler.jsonc`. Nodes must use the final HTTPS URL, for example:

```bash
CHAIN_BOOTNODE_REGISTRY=https://registry.knirv.com
```

## Configuration

| Variable | Purpose | Default |
|---|---|---|
| `REGISTRY_VERSION` | API version string returned by `/status` | `3.0.0` |
| `NODE_TTL_SECONDS` | TTL for `/register` node entries | `3600` |
| `HEARTBEAT_TTL_SECONDS` | TTL for validator heartbeats | `45` |
| `STUN_HOST` | External STUN server hostname | `""` |
| `STUN_PORT` | External STUN server UDP port | `3478` |
| `TURN_TCP_PORT` | External TURN server TCP port | `3479` |
| `BOOTNODE_VALIDATION` | `enforce` (bootnodes must pass validation to register or be listed), `monitor` (record results, do not block), `off` | `enforce` |
| `VALIDATION_TTL_SECONDS` | How long a passing validation is reused before the bootnode is re-probed | `3600` |
| `VALIDATION_FAIL_COOLDOWN_SECONDS` | Minimum wait before a failed bootnode is re-probed (returns `429`) | `60` |
| `VALIDATION_MAX_LATENCY_MS` | Median latency limit for the performance check | `1500` |
| `VALIDATION_FQDN_SUFFIXES` | Comma-separated hostname suffixes a node may claim; empty allows any public hostname | `.knirv.com` |
| `VALIDATION_SERVICE_TOKEN` | Optional secret. If set, `POST /validate` requires it as a Bearer token; if unset the endpoint is open (it is read-only and only probes public IPs and allowlisted hostnames) | unset |
| `ROOT_PUBLIC_KEY` | secp256k1 public key of the root node (hex, no `0x` prefix); trust anchor for `POST /validators` at genesis | `""` |

## API

### Node Registration

- `POST /register` — Register or refresh a node. Body: `{ chainID, ip?, port, type? ("bootnode"|"client"), peerID?, fqdn? }`. Clients register as before. A bootnode is validated first (see below): `422` with the failed checks if it fails, `429` with `Retry-After` during the failure cooldown. `nodeID` is accepted as an alias for `peerID`.
- `GET /lookup/:chainID` — Look up a node by chain ID.
- `GET /nodes` — List all registered nodes.
- `GET /bootnodes` — List only bootnode-registered nodes.
- `GET /status` — Registry health and node counts. Returns HTML for browsers.
- `GET /stun` — External STUN/TURN endpoint pointer.

### Bootnode Validation

A bootnode is registered and listed only after it passes four checks, run by the Worker:

| Check | What the registry does |
|---|---|
| `connectivity` | Opens a TCP connection to the registered `ip:port` |
| `health` | `GET https://<fqdn>/health` must return 200 with `chainId` equal to the registered `chainID`; `dht.peer_id` must match `peerID` when both are present; if `dht.self_addrs` is present it must include the registered `ip` |
| `security` | The HTTPS request to `fqdn` completes, so the certificate chain and hostname verified |
| `performance` | Median of the TCP connect and three health requests is within `VALIDATION_MAX_LATENCY_MS` |

`ip` must be publicly routable and `fqdn` must match `VALIDATION_FQDN_SUFFIXES`. A bootnode with no `fqdn` fails `health`, `security` and `performance`. KNIRVGATEWAY sends `fqdn` from its public hostname. In `enforce` mode `GET /nodes`, `GET /bootnodes` and the suggested bootnode returned to clients include only validated bootnodes, and `/status` shows a Validated column. Existing unvalidated bootnodes disappear from those lists until their next registration passes.

- `POST /validate` — Validation-service endpoint for ONBOARDING.KNIRV.COM. No auth required unless `VALIDATION_SERVICE_TOKEN` is set (then `Authorization: Bearer <token>`). Body: `{ registrationId, fqdn, publicIps, expectedPorts }`. Returns `{ checks: { connectivity, health, security, performance }, details }`. The onboarding site defaults to `https://registry.knirv.com/validate`, so nothing needs configuring. This endpoint cannot check chain identity, because the onboarding request carries no `chainID`.

Known limits: a passing node proves control of a reachable endpoint, not that its IP is bound to the hostname when `self_addrs` is absent. Validation attempts are rate-limited per `chainID` only.

### Validator Failover Control Plane

All write endpoints below require a `secp256k1` signature over the canonical
message described in each section. The Worker verifies signatures against the
current enrolled-validator set before accepting any write.

- `POST /validators` — Push a new validator set. Body: `{ chainID, version, validators: [{ validatorID, publicKey, active }], signature }`. Signed by the current authorized publisher (root's key at genesis, then the confirmed new root's key after a `CONFIRMED` state transition).
- `GET /validators/:chainID` — Return the current validator set for a chain.
- `POST /heartbeat` — Submit a validator heartbeat. Body: `{ chainID, validatorID, rootObservedHealthy, rootLastSeenMs, syncHeight, signature }`.
- `GET /heartbeats/:chainID` — Return all current heartbeats for a chain.
- `POST /vote` — Cast a failover vote. Body: `{ chainID, round, validatorID, candidateID, signature }`.
- `GET /votes/:chainID/:round` — Return the vote tally for a round.
- `POST /state` — Transition the chain state. Body: `{ chainID, phase, currentRootID, since, signature }`. Allowed phases: `NORMAL`, `VOTING`, `ACTING_ROOT`, `CONFIRMED`, `RECLAIMED`.
- `GET /state/:chainID` — Return the current state for a chain.

### Signature Message Formats

| Endpoint | Canonical message |
|---|---|
| `POST /heartbeat` | `chainID\|validatorID\|rootObservedHealthy\|rootLastSeenMs\|syncHeight` |
| `POST /vote` | `chainID\|round\|candidateID` |
| `POST /state` | `chainID\|phase\|currentRootID\|since` |
| `POST /validators` | `chainID\|version\|JSON.stringify(validators)` |

### State Machine

```
NORMAL ──quorum: root down──► VOTING ──winner──► ACTING_ROOT
                                        │              │
                                  root reclaims   24h elapses
                                        │              │
                                        ▼              ▼
                                   RECLAIMED      CONFIRMED
                                   (back to        (payment routing
                                    NORMAL)         swaps to new root)
```

On `CONFIRMED`, the authorized publisher key rotates from root to the new root.

## Testing

```bash
npm test
```

## Development

```bash
npm run dev
```
