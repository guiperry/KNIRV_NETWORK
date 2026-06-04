// knirv-tunnel-client/example.js
const KNIRVTunnelClient = require('./tunnelClient');

// Example usage
const client = new KNIRVTunnelClient({
    tunnelServerHost: process.env.TUNNEL_SERVER_HOST || ROOTCHAIN_URL,
    tunnelServerControlPort: parseInt(process.env.TUNNEL_SERVER_CONTROL_PORT || '4001'),
    peerId: process.env.PEER_ID || 'QmExamplePeerID', // This should be the actual peer ID
    chainId: process.env.CHAIN_ID || 'knirv-default',
    internalP2pPort: parseInt(process.env.INTERNAL_P2P_PORT || '5050'),
    type: process.env.NODE_TYPE || 'peer', // 'peer', 'bootnode', etc.
    maxReconnectAttempts: parseInt(process.env.MAX_RECONNECT_ATTEMPTS || '10'),
    reconnectDelay: parseInt(process.env.RECONNECT_DELAY || '5000')
});

// Connect to the tunnel server
client.connect();

// Start sending pings to keep the connection alive
client.startPingInterval(30000); // 30 seconds

// Handle process termination
process.on('SIGINT', () => {
    console.log('Received SIGINT. Shutting down...');
    client.stopPingInterval();
    client.disconnect();
    process.exit(0);
});

process.on('SIGTERM', () => {
    console.log('Received SIGTERM. Shutting down...');
    client.stopPingInterval();
    client.disconnect();
    process.exit(0);
});