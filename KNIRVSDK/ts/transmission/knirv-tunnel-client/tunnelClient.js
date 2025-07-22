// knirv-tunnel-client/tunnelClient.js
const net = require('net');
const os = require('os');

class KNIRVTunnelClient {
    constructor(options) {
        this.tunnelServerHost = options.tunnelServerHost || process.env.TUNNEL_SERVER_HOST || ROOTCHAIN_URL;
        this.tunnelServerControlPort = options.tunnelServerControlPort || 4001;
        this.peerId = options.peerId; // Required
        this.chainId = options.chainId;
        this.internalIp = options.internalIp || this._getLocalIp();
        this.internalP2pPort = options.internalP2pPort || 5050;
        this.type = options.type || 'peer';
        
        this.controlSocket = null;
        this.isConnected = false;
        this.reconnectAttempts = 0;
        this.maxReconnectAttempts = options.maxReconnectAttempts || 10;
        this.reconnectDelay = options.reconnectDelay || 5000; // 5 seconds
    }
    
    _getLocalIp() {
        // Simple function to get a local IP address
        const interfaces = os.networkInterfaces();
        for (const name of Object.keys(interfaces)) {
            for (const iface of interfaces[name]) {
                if (iface.family === 'IPv4' && !iface.internal) {
                    return iface.address;
                }
            }
        }
        return '127.0.0.1'; // Fallback
    }
    
    connect() {
        if (this.isConnected) {
            console.log('Already connected to tunnel server.');
            return;
        }
        
        console.log(`Connecting to tunnel server at ${this.tunnelServerHost}:${this.tunnelServerControlPort}...`);
        
        this.controlSocket = new net.Socket();
        
        this.controlSocket.on('connect', () => {
            console.log('Connected to tunnel server. Sending identification...');
            this.isConnected = true;
            this.reconnectAttempts = 0;
            
            // Send identification message
            const identifyMsg = {
                action: 'IDENTIFY',
                peerId: this.peerId,
                chainId: this.chainId,
                internalIp: this.internalIp,
                internalP2pPort: this.internalP2pPort,
                type: this.type
            };
            
            this.controlSocket.write(JSON.stringify(identifyMsg) + '\n');
        });
        
        this.controlSocket.on('data', (data) => {
            try {
                const messages = data.toString().split('\n').filter(msg => msg.trim());
                for (const msg of messages) {
                    const parsed = JSON.parse(msg);
                    console.log('Received from tunnel server:', parsed);
                    
                    if (parsed.action === 'PONG') {
                        console.log('Received PONG from tunnel server.');
                    }
                }
            } catch (e) {
                console.error('Error parsing message from tunnel server:', e.message);
            }
        });
        
        this.controlSocket.on('error', (err) => {
            console.error('Control socket error:', err.message);
            this.isConnected = false;
        });
        
        this.controlSocket.on('close', () => {
            console.log('Control socket closed.');
            this.isConnected = false;
            
            // Attempt to reconnect
            if (this.reconnectAttempts < this.maxReconnectAttempts) {
                this.reconnectAttempts++;
                console.log(`Attempting to reconnect (${this.reconnectAttempts}/${this.maxReconnectAttempts}) in ${this.reconnectDelay}ms...`);
                setTimeout(() => this.connect(), this.reconnectDelay);
            } else {
                console.error('Max reconnect attempts reached. Giving up.');
            }
        });
        
        // Connect to the tunnel server
        this.controlSocket.connect(this.tunnelServerControlPort, this.tunnelServerHost);
    }
    
    disconnect() {
        if (this.controlSocket && !this.controlSocket.destroyed) {
            console.log('Disconnecting from tunnel server...');
            this.controlSocket.destroy();
        }
    }
    
    // Send a ping to keep the connection alive
    ping() {
        if (this.isConnected && this.controlSocket && !this.controlSocket.destroyed) {
            this.controlSocket.write(JSON.stringify({ action: 'PING' }) + '\n');
        }
    }
    
    // Start a ping interval
    startPingInterval(interval = 30000) {
        this._pingInterval = setInterval(() => this.ping(), interval);
    }
    
    // Stop the ping interval
    stopPingInterval() {
        if (this._pingInterval) {
            clearInterval(this._pingInterval);
            this._pingInterval = null;
        }
    }
}

module.exports = KNIRVTunnelClient;