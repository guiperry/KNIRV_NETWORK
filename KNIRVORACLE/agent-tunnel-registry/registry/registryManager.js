// agent-tunnel-registry/registry/registryManager.js
const axios = require('axios');
const config = require('../config');
const { v4: uuidv4 } = require('uuid');

class RegistryManager {
    constructor() {
        // In-memory for now. For production, use a persistent store (Redis, DB).
        this.nodes = new Map(); // devId -> nodeInfo
        this.chainIdToPeerId = new Map(); // chainId -> devId (if chainId is unique)
        this.tunneledResourceIds = new Map(); // uniqueId -> devId (for URI resolution)
    }

    // Called when a node connects its control socket
    registerNodeViaControlSocket(devId, chainId, internalIp, internalP2pPort, type, controlSocketId, serverPublicHost, publicRelayPort) {
        const nodeInfo = {
            devId,
            chainId: chainId || devId, // Fallback if chainId not distinct
            internalIp,
            internalP2pPort,
            type,
            lastSeen: Date.now(),
            controlSocketId, // ID of the control socket for tunneling
            isTunneled: true,
            // The public address clients should use to reach this node via THIS tunnel server
            publicRelayUrl: `tcp://${serverPublicHost}:${publicRelayPort}/p2p_tunnel/${devId}` // Example
        };
        this.nodes.set(devId, nodeInfo);
        if (nodeInfo.chainId) {
            this.chainIdToPeerId.set(nodeInfo.chainId, devId);
        }
        console.log(`[Registry] Node registered/updated via control: ${type} ${devId} on chain ${nodeInfo.chainId}`);
        return nodeInfo;
    }

    // Called via HTTP API for nodes NOT behind NAT or for initial info
    registerNodeViaApi(devId, chainId, publicIp, publicP2pPort, type) {
        const nodeInfo = {
            devId,
            chainId: chainId || devId,
            publicIp, // Directly reachable public IP
            publicP2pPort,
            type,
            lastSeen: Date.now(),
            isTunneled: false,
            publicRelayUrl: null // Not tunneled through this server
        };
        this.nodes.set(devId, nodeInfo);
        
        // For publicly registered nodes, announce on the global DHT via the Go process
        const multiaddress = `/ip4/${publicIp}/tcp/${publicP2pPort}/p2p/${devId}`;
        axios.post(`http://localhost:${config.goInternalApiPort}/internal/dht/announceResource`, { 
            id: devId, 
            type: type || 'dev', 
            multiaddress: multiaddress 
        }).catch(err => console.error(`[Registry] Failed to announce node ${devId} on DHT: ${err.message}`));
        
        if (nodeInfo.chainId) {
            this.chainIdToPeerId.set(nodeInfo.chainId, devId);
        }
        console.log(`[Registry] Node registered/updated via API: ${type} ${devId} on chain ${nodeInfo.chainId}`);
        return nodeInfo;
    }

    // Map a unique tunneled resource ID to a dev ID (for URI resolution)
    mapTunneledResource(uniqueId, devId) {
        this.tunneledResourceIds.set(uniqueId, devId);
        return uniqueId;
    }

    // Get the dev ID associated with a tunneled resource ID
    getPeerIdForTunneledResource(uniqueId) {
        return this.tunneledResourceIds.get(uniqueId);
    }

    getNodeByPeerId(devId) {
        return this.nodes.get(devId);
    }

    getNodeByChainId(chainId) {
        const devId = this.chainIdToPeerId.get(chainId);
        return devId ? this.nodes.get(devId) : undefined;
    }

    getAllNodes() {
        return Array.from(this.nodes.values());
    }

    deregisterNodeByControlSocket(controlSocketId) {
        for (const [devId, nodeInfo] of this.nodes.entries()) {
            if (nodeInfo.controlSocketId === controlSocketId) {
                this.nodes.delete(devId);
                if (nodeInfo.chainId) {
                    this.chainIdToPeerId.delete(nodeInfo.chainId);
                }
                console.log(`[Registry] Node ${devId} deregistered (control socket closed).`);
                break;
            }
        }
    }

    // TODO: Periodic pruning of inactive nodes not seen recently
}

module.exports = new RegistryManager();