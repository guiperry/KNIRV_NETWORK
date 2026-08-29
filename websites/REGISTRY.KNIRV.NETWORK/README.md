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
CHAIN_BOOTNODE_REGISTRY=https://registry.knirv.network
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
| `ROOT_PUBLIC_KEY` | secp256k1 public key of the root node (hex, no `0x` prefix); trust anchor for `POST /validators` at genesis | `""` |

## API

### Node Registration

- `POST /register` — Register or refresh a node. Body: `{ chainID, ip?, port, type? ("bootnode"|"client"), peerID? }`
- `GET /lookup/:chainID` — Look up a node by chain ID.
- `GET /nodes` — List all registered nodes.
- `GET /bootnodes` — List only bootnode-registered nodes.
- `GET /status` — Registry health and node counts. Returns HTML for browsers.
- `GET /stun` — External STUN/TURN endpoint pointer.

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
