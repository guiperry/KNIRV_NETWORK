// agent-tunnel-registry/tunneling/tunnelManager.js
class TunnelManager {
    constructor() {
        this.activeControlSockets = new Map(); // devId -> socket object (the control socket)
    }

    addControlSocket(devId, socket) {
        // Handle if a dev reconnects, close old socket?
        const existingSocket = this.activeControlSockets.get(devId);
        if (existingSocket && !existingSocket.destroyed) {
            console.warn(`[TunnelMgr] Peer ${devId} re-established control. Closing old control socket.`);
            existingSocket.destroy();
        }
        this.activeControlSockets.set(devId, socket);
        
        // Update the connection registry with the new socket
        if (connectionRegistry && typeof connectionRegistry === 'object') {
            if (!connectionRegistry[devId]) {
                connectionRegistry[devId] = {
                    id: devId,
                    type: 'dev', // Default type, will be updated by IDENTIFY message
                    sourceIP: socket.remoteAddress,
                    sourcePort: socket.remotePort,
                    lastSeen: Date.now(),
                    socket: socket
                };
            } else {
                // Update existing entry
                connectionRegistry[devId].lastSeen = Date.now();
                connectionRegistry[devId].socket = socket;
                connectionRegistry[devId].sourceIP = socket.remoteAddress;
                connectionRegistry[devId].sourcePort = socket.remotePort;
            }
        }
        
        console.log(`[TunnelMgr] Control socket for ${devId} is now active.`);
    }

    removeControlSocket(devId, socketIdToMatch) {
        const currentSocket = this.activeControlSockets.get(devId);
        // Only remove if the socketId matches, in case of rapid reconnects
        if (currentSocket && currentSocket.id === socketIdToMatch) {
            this.activeControlSockets.delete(devId);
            
            // Don't remove from connection registry here - let the pruning function handle that
            // Just mark the socket as destroyed in the registry
            if (connectionRegistry && connectionRegistry[devId]) {
                // Don't delete the entry, just mark the socket as null/destroyed
                // This allows the entry to remain for the status page until pruned
                connectionRegistry[devId].socket = null;
                console.log(`[TunnelMgr] Marked socket as destroyed in connection registry for ${devId}`);
            }
            
            console.log(`[TunnelMgr] Control socket for ${devId} removed.`);
        } else if (currentSocket) {
            console.log(`[TunnelMgr] Stale removeControlSocket call for ${devId}, current socket ID ${currentSocket.id} != ${socketIdToMatch}`);
        }
    }

    getControlSocket(devId) {
        const socket = this.activeControlSockets.get(devId);
        if (socket && !socket.destroyed) {
            return socket;
        }
        return null;
    }

    // This is the core relay logic
    relay(externalClientSocket, targetInternalPeerId) {
        const internalControlSocket = this.getControlSocket(targetInternalPeerId);

        if (!internalControlSocket) {
            console.warn(`[TunnelMgr] No active control socket for target ${targetInternalPeerId}. Cannot relay.`);
            externalClientSocket.end(); // Or send an error message
            return false;
        }

        console.log(`[TunnelMgr] Relaying data between external client and ${targetInternalPeerId}`);

        // Bidirectional pipe
        externalClientSocket.pipe(internalControlSocket, { end: false }); // Don't end control socket if external client disconnects
        internalControlSocket.pipe(externalClientSocket, { end: false }); // Don't end external client if control socket has issues

        const cleanup = () => {
            console.log(`[TunnelMgr] Cleaning up relay for external client to ${targetInternalPeerId}`);
            if (!externalClientSocket.destroyed) {
                externalClientSocket.unpipe(internalControlSocket);
                externalClientSocket.destroy();
            }
            if (!internalControlSocket.destroyed) { // Control socket might be reused
                internalControlSocket.unpipe(externalClientSocket);
            }
        };

        externalClientSocket.on('error', (err) => {
            console.error('[TunnelMgr] External client socket error during relay:', err.message);
            cleanup();
        });
        externalClientSocket.on('close', () => {
            console.log('[TunnelMgr] External client socket closed during relay.');
            cleanup();
        });
        // Errors on internalControlSocket are handled by its own 'error' and 'close' listeners

        return true;
    }
}

module.exports = new TunnelManager();