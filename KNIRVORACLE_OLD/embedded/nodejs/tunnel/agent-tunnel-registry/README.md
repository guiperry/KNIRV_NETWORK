# KNIRVORACLE Tunnel Registry Service

This service provides NAT traversal, node registration, and URI resolution for the KNIRVORACLE network.

## Features

- **Node Registration**: Allows bootnodes and devs to register their internal P2P details and unique identifiers
- **URI Generation/Resolution**: Generates and resolves `knirv://` URIs, directing clients to tunneled connections when necessary
- **NAT Tunneling (Relay)**: Provides relay services for nodes behind NAT by establishing persistent outbound connections
- **STUN Server**: Assists with NAT traversal by providing STUN services

## Ports

| Service | Port Name | Protocol | Purpose | Default Value |
|---------|-----------|----------|---------|--------------|
| **Tunnel Registry Service** | | | | |
| | HTTP API | TCP | Public REST API for node registration and URI resolution | 3003 |
| | Control Port | TCP | Internal nodes connect here for registration and tunneling | 4001 |
| | Public Relay Port | TCP | External clients connect here for tunneled connections | 4000 |
| | STUN Port | UDP | STUN server for NAT traversal assistance | 3478 |

## Installation

```bash
npm install
```

## Configuration

Configuration is done through environment variables:

- `HTTP_API_PORT`: Port for the HTTP API (default: 3003)
- `CONTROL_PORT`: Port for internal nodes to connect for registration and tunneling (default: 4001)
- `PUBLIC_RELAY_PORT`: Port for external clients to connect for tunneled connections (default: 4000)
- `STUN_PORT`: Port for the STUN server (default: 3478)
- `PUBLIC_HOST`: Publicly addressable hostname or IP of the server (default: ROOTCHAIN_URL)
- `RELAY_SERVER_PEER_ID`: PeerID of the KNIRVORACLE-ROOT node

## Usage

```bash
npm start
```

## API Endpoints

### Registration

- `POST /api/registry/register`: Register a node via API
- `GET /api/registry/nodes`: Get all registered nodes
- `GET /api/registry/node/dev/:devId`: Get a node by PeerID
- `GET /api/registry/node/chain/:chainId`: Get a node by ChainID

### URI

- `POST /api/uri/generate`: Generate a knirv:// URI
- `GET /api/uri/resolve`: Resolve a knirv:// URI

## Status Page

A status page is available at the root URL (`/`) showing the current status of the service, including active connections and system metrics.

## License

MIT