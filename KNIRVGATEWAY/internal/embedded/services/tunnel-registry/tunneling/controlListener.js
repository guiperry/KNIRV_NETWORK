// agent-tunnel-registry/tunneling/controlListener.js
const net = require('net');
const config = require('../config');
const registryManager = require('../registry/registryManager');
const tunnelManager = require('./tunnelManager');

// Access the global connection registry
const connectionRegistry = global.connectionRegistry || {};

class ControlListener {
    constructor() {
        this.port = config.controlListenerPort;
        this.server = net.createServer(this._handleConnection.bind(this));
    }

    start() {
        this.server.listen(this.port, () => {
            console.log(`Control Listener for internal nodes started on TCP port ${this.port}`);
        });
        this.server.on('error', (err) => {
            console.error('[ControlListener] Server error:', err);
        });
    }

    _handleConnection(socket) {
        // Assign a unique ID to the socket for tracking
        socket.id = `${socket.remoteAddress}:${socket.remotePort}-${Math.random().toString(36).substr(2, 5)}`;
        console.log(`[ControlListener] Internal node connected: ${socket.id}`);

        let identifiedPeerId = null;

        socket.on('data', (data) => {
            const dataStr = data.toString();
            
            // Check if this is an HTTP request (starts with HTTP methods)
            if (dataStr.startsWith('GET ') || dataStr.startsWith('POST ') || dataStr.startsWith('HEAD ') ||
                dataStr.startsWith('PUT ') || dataStr.startsWith('DELETE ') || dataStr.startsWith('OPTIONS ')) {
                // This is an HTTP request - send appropriate response
                console.log(`[ControlListener] Received HTTP request from ${socket.id}, sending service info`);
                const httpResponse = `HTTP/1.1 200 OK\r
Content-Type: text/plain\r
Connection: close\r
\r
KNIRV Tunnel Registry Control Service
This port (${this.port}) is for internal node control connections.
Use the HTTP API on port ${config.httpApiPort} for web requests.
Protocol: JSON messages (IDENTIFY, PING, etc.)
`;
                socket.write(httpResponse);
                socket.end();
                return;
            }
            
            try {
                const message = JSON.parse(dataStr);
                // Expect: { action: 'IDENTIFY', devId: 'Qm...', chainId: '...', internalIp: '192...', internalP2pPort: 5050, type: 'bootnode' }
                if (message.action === 'IDENTIFY' && message.devId && message.internalIp && message.internalP2pPort) {
                    identifiedPeerId = message.devId;
                    tunnelManager.addControlSocket(identifiedPeerId, socket);
                    registryManager.registerNodeViaControlSocket(
                        identifiedPeerId,
                        message.chainId,
                        message.internalIp,
                        message.internalP2pPort,
                        message.type || 'dev',
                        socket.id,
                        config.serverPublicHost,
                        config.publicRelayPort
                    );
                    socket.write(JSON.stringify({ status: 'SUCCESS', message: 'Identified and control channel active.' }) + '\n');
                    console.log(`[ControlListener] Socket ${socket.id} identified as Peer ${identifiedPeerId}`);
                } else if (message.action === 'PING') {
                    // Update the lastSeen timestamp in the connection registry to prevent pruning
                    if (identifiedPeerId && connectionRegistry[identifiedPeerId]) {
                        connectionRegistry[identifiedPeerId].lastSeen = Date.now();
                        console.log(`[ControlListener] Received PING from ${identifiedPeerId}, updated lastSeen timestamp`);
                    } else if (identifiedPeerId) {
                        // If the connection exists in tunnelManager but not in connectionRegistry, re-register it
                        console.log(`[ControlListener] Received PING from ${identifiedPeerId} but not in registry, re-registering`);
                        // Create a basic entry in the connection registry
                        connectionRegistry[identifiedPeerId] = {
                            id: identifiedPeerId,
                            type: 'dev', // Default type
                            sourceIP: socket.remoteAddress,
                            sourcePort: socket.remotePort,
                            lastSeen: Date.now(),
                            socket: socket
                        };
                    }
                    // Send PONG response back to the client
                    socket.write(JSON.stringify({
                        action: 'PONG',
                        timestamp: Date.now(),
                        status: 'active'
                    }) + '\n');
                } else {
                    console.warn(`[ControlListener] Unknown message from ${socket.id}:`, message);
                }
            } catch (e) {
                console.error(`[ControlListener] Error processing data from ${socket.id}:`, e.message, dataStr);
            }
        });

        socket.on('error', (err) => {
            console.error(`[ControlListener] Socket error for ${socket.id} (Peer ${identifiedPeerId || 'unknown'}):`, err.message);
            // Cleanup handled by 'close'
        });

        socket.on('close', () => {
            console.log(`[ControlListener] Socket closed for ${socket.id} (Peer ${identifiedPeerId || 'unknown'})`);
            if (identifiedPeerId) {
                tunnelManager.removeControlSocket(identifiedPeerId, socket.id);
                registryManager.deregisterNodeByControlSocket(socket.id);
            }
        });

        socket.setKeepAlive(true, 30000); // Send keep-alive packets every 30s
    }
}

module.exports = new ControlListener();