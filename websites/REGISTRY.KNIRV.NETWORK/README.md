# KNIRV Network Registry Worker

This project deploys the KNIRV node-registration HTTP API to Cloudflare Workers.
Node state is stored consistently in a Durable Object and expires after one hour.

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

## STUN/TURN

Workers cannot host UDP STUN/TURN listeners. Run TURN/STUN separately and set
`STUN_HOST`, `STUN_PORT`, and `TURN_TCP_PORT` in the Worker configuration.
`GET /stun` exposes those external endpoints.

## API

- `POST /register`
- `GET /lookup/:chainID`
- `GET /nodes`
- `GET /bootnodes`
- `GET /status`
- `GET /stun`
