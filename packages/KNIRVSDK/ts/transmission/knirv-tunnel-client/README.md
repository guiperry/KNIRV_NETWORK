# KNIRVCHAIN Tunnel Client

This is a Node.js client for KNIRVCHAIN nodes behind NAT to establish a control connection with the central relay server.

## Features

- Establishes a persistent control connection to the tunnel server
- Automatically reconnects if the connection is lost
- Sends periodic pings to keep the connection alive
- Identifies the node to the tunnel server with peer ID, chain ID, and other details

## Installation

```bash
npm install
```

## Usage

```javascript
const KNIRVTunnelClient = require('./tunnelClient');

const client = new KNIRVTunnelClient({
    tunnelServerHost: process.env.TUNNEL_SERVER_HOST || ROOTCHAIN_URL,
    tunnelServerControlPort: 4001,
    peerId: 'QmYourPeerID',
    chainId: 'your-chain-id',
    internalP2pPort: 5050,
    type: 'peer', // 'peer', 'bootnode', etc.
});

// Connect to the tunnel server
client.connect();

// Start sending pings to keep the connection alive
client.startPingInterval(30000); // 30 seconds

// When shutting down
client.stopPingInterval();
client.disconnect();
```

## Configuration

The client can be configured with the following options:

- `tunnelServerHost`: The hostname or IP address of the tunnel server (default: ROOTCHAIN_URL)
- `tunnelServerControlPort`: The port for the control connection (default: 4001)
- `peerId`: The peer ID of the node (required)
- `chainId`: The chain ID of the node (optional)
- `internalIp`: The internal IP address of the node (default: auto-detected)
- `internalP2pPort`: The internal P2P port of the node (default: 5050)
- `type`: The type of node ('peer', 'bootnode', etc.) (default: 'peer')
- `maxReconnectAttempts`: The maximum number of reconnect attempts (default: 10)
- `reconnectDelay`: The delay between reconnect attempts in milliseconds (default: 5000)

## License

MIT