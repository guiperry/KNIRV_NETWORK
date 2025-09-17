// agent-tunnel-registry/stun/stunServer.js
const dgram = require('dgram');
const config = require('../config');

class StunServer {
    constructor(port) {
        this.port = port || config.stunPort;
        this.server = dgram.createSocket('udp4');
    }

    start() {
        this.server.on('error', (err) => {
            console.error(`[STUN] Server error: ${err.stack}`);
            this.server.close();
        });

        this.server.on('message', (msg, rinfo) => {
            console.log(`[STUN] Received ${msg.length} bytes from ${rinfo.address}:${rinfo.port}`);
            
            // Very basic STUN implementation - just echo back the client's IP and port
            // For a full STUN implementation, you'd need to parse the STUN message format
            // and respond according to RFC 5389
            
            // Create a simple response with the client's address information
            const response = Buffer.from(JSON.stringify({
                type: 'stun-response',
                remoteAddress: rinfo.address,
                remotePort: rinfo.port,
                timestamp: Date.now()
            }));
            
            this.server.send(response, 0, response.length, rinfo.port, rinfo.address, (err) => {
                if (err) {
                    console.error(`[STUN] Error sending response to ${rinfo.address}:${rinfo.port}: ${err}`);
                } else {
                    console.log(`[STUN] Sent response to ${rinfo.address}:${rinfo.port}`);
                }
            });
        });

        this.server.on('listening', () => {
            const address = this.server.address();
            console.log(`[STUN] Server listening on UDP ${address.address}:${address.port}`);
        });

        this.server.bind(this.port);
    }

    close() {
        if (this.server) {
            this.server.close(() => {
                console.log('[STUN] Server closed');
            });
        }
    }
}

module.exports = StunServer;