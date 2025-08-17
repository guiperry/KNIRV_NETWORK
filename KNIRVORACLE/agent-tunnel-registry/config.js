// agent-tunnel-registry/config.js
module.exports = {
    // HTTP API configuration
    httpApiPort: process.env.HTTP_API_PORT || 3003,
    
    // Tunnel service configuration
    controlListenerPort: process.env.CONTROL_PORT || 4001, // Internal nodes connect here
    publicRelayPort: process.env.PUBLIC_RELAY_PORT || 4000, // External clients connect here for tunneling
    
    // STUN server configuration
    stunPort: process.env.STUN_PORT || 3478, // Standard STUN port
    
    // Server identification
    serverPublicHost: process.env.PUBLIC_HOST || 'localhost', // Publicly addressable host of this server
    relayServerPeerId: process.env.RELAY_SERVER_PEER_ID || '', // PeerID of the KNIRVORACLE-ROOT node
    
    // Go internal API configuration
    goInternalApiPort: process.env.GO_INTERNAL_API_PORT || 8080, // Port for internal API communication with Go process
    
    // Connection pruning configuration
    connectionPruningInterval: 60 * 1000, // Check for inactive connections every minute
    connectionInactiveTimeout: 5 * 60 * 1000, // Consider connections inactive after 5 minutes of no pings
    
    // Add any other config like database connection strings if not in-memory
};