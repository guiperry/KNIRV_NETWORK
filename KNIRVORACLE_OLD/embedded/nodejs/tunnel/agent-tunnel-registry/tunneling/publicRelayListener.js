// agent-tunnel-registry/tunneling/publicRelayListener.js
const net = require('net');
const config = require('../config');
const tunnelManager = require('./tunnelManager');

class PublicRelayListener {
    constructor() {
        this.port = config.publicRelayPort;
        this.server = net.createServer(this._handleConnection.bind(this));
    }

    start() {
        this.server.listen(this.port, () => {
            console.log(`Public Relay Listener started on TCP port ${this.port}`);
        });
        this.server.on('error', (err) => {
            console.error('[PublicRelayListener] Server error:', err);
        });
    }

    _handleConnection(externalClientSocket) {
        console.log(`[PublicRelayListener] Incoming external connection from ${externalClientSocket.remoteAddress}:${externalClientSocket.remotePort}`);

        // The client needs to tell us which internal PeerID it wants to connect to.
        let targetPeerId = null;
        let initialDataBuffer = '';

        const onData = (data) => {
            initialDataBuffer += data.toString();
            const newlineIndex = initialDataBuffer.indexOf('\n');

            if (newlineIndex !== -1) {
                const firstLine = initialDataBuffer.substring(0, newlineIndex).trim();
                initialDataBuffer = initialDataBuffer.substring(newlineIndex + 1); // Keep rest for piping

                try {
                    // Example: Client sends JSON { "targetPeerId": "Qm..." }
                    const message = JSON.parse(firstLine);
                    if (message.targetPeerId) {
                        targetPeerId = message.targetPeerId;
                    } else {
                        throw new Error("Missing targetPeerId in initial message");
                    }
                } catch (e) {
                     // Fallback: assume firstLine is the PeerID directly
                    if (firstLine.startsWith("Qm") || firstLine.startsWith("12D3Koo")) { // Basic PeerID check
                        targetPeerId = firstLine;
                    } else {
                        console.warn(`[PublicRelayListener] Invalid initial message from external client: ${firstLine}. Error: ${e.message}`);
                        externalClientSocket.end('ERROR: Invalid initial message. Expecting target PeerID.\n');
                        return;
                    }
                }
                
                console.log(`[PublicRelayListener] External client wants to connect to PeerID: ${targetPeerId}`);
                externalClientSocket.removeListener('data', onData); // Stop listening for identification

                if (tunnelManager.relay(externalClientSocket, targetPeerId)) {
                    // If relay started, send remaining buffered data (if any)
                    const internalControlSocket = tunnelManager.getControlSocket(targetPeerId);
                    if (internalControlSocket && initialDataBuffer.length > 0) {
                        console.log(`[PublicRelayListener] Forwarding ${initialDataBuffer.length} buffered bytes to ${targetPeerId}`);
                        internalControlSocket.write(initialDataBuffer);
                    }
                } else {
                    externalClientSocket.end(`ERROR: Could not establish relay to ${targetPeerId}.\n`);
                }
            }
        };

        externalClientSocket.on('data', onData);

        externalClientSocket.on('error', (err) => {
            console.error('[PublicRelayListener] External client socket error:', err.message);
        });
        externalClientSocket.on('close', () => {
            console.log('[PublicRelayListener] External client connection closed.');
        });
    }
}

module.exports = new PublicRelayListener();