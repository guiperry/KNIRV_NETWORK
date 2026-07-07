/**
 * KNIRVORACLE DHT Manager
 *
 * Implements a simplified DHT functionality for KNIRVORACLE to listen for network changes
 * and announcements from other KNIRV services. This is a lightweight client that
 * primarily listens rather than actively participating in the DHT.
 *
 * Uses WebSocket connections to communicate with other KNIRV services.
 */

const WebSocket = require('ws');
const crypto = require('crypto');
const EventEmitter = require('events');

// DHT configuration constants
const DHT_CONFIG = {
  DISCOVERY_NAMESPACE: 'knirvoracle',
  NETWORK_CONTROL_TOPIC: 'network-control',
  GRAPH_ANNOUNCEMENT_TOPIC: 'graph-announcements',
  CHAIN_ANNOUNCEMENT_TOPIC: 'chain-announcements',
  NEXUS_ANNOUNCEMENT_TOPIC: 'nexus-announcements',
  ROUTER_ANNOUNCEMENT_TOPIC: 'router-announcements',
  NETWORK_PAUSE_TIMEOUT: 30 * 60 * 1000, // 30 minutes in milliseconds
};

class DHTManager extends EventEmitter {
  constructor(chainID = 'testnet', serviceID = 'knirvoracle') {
    super();
    this.chainID = chainID;
    this.serviceID = `${serviceID}-${chainID}`;
    this.peerId = this.generatePeerId();
    this.connections = new Map();
    this.isStarted = false;
    this.networkPaused = false;
    this.pausedUntil = null;
    this.bootstrapPeers = [];
    this.reconnectAttempts = new Map();
    this.maxReconnectAttempts = 5;

    // Event listeners for network announcements
    this.listeners = {
      networkControl: [],
      graphAnnouncement: [],
      chainAnnouncement: [],
      nexusAnnouncement: [],
      routerAnnouncement: [],
    };

    // Bind methods
    this.handleNetworkControl = this.handleNetworkControl.bind(this);
    this.handleGraphAnnouncement = this.handleGraphAnnouncement.bind(this);
    this.handleChainAnnouncement = this.handleChainAnnouncement.bind(this);
    this.handleNexusAnnouncement = this.handleNexusAnnouncement.bind(this);
    this.handleRouterAnnouncement = this.handleRouterAnnouncement.bind(this);
  }

  /**
   * Generate a unique peer ID for this instance
   */
  generatePeerId() {
    return crypto.randomBytes(32).toString('hex');
  }

  /**
   * Initialize the DHT manager with bootstrap peers
   */
  async initialize(bootstrapPeers = []) {
    try {
      // Parse bootstrap peers as WebSocket URLs
      this.bootstrapPeers = bootstrapPeers.filter(peer => {
        try {
          // Convert libp2p multiaddr to WebSocket URL if needed
          if (peer.startsWith('/ip4/')) {
            const parts = peer.split('/');
            const ip = parts[2];
            const port = parts[4];
            return `ws://${ip}:${port}`;
          } else if (peer.startsWith('ws://') || peer.startsWith('wss://')) {
            return peer;
          }
          return null;
        } catch (error) {
          console.warn(`Invalid bootstrap peer: ${peer}`);
          return null;
        }
      }).filter(Boolean);

      console.log(`KNIRVORACLE DHT Manager initialized with PeerID: ${this.peerId}`);
      console.log(`Bootstrap peers: ${this.bootstrapPeers.length}`);
      return true;
    } catch (error) {
      console.error('Failed to initialize KNIRVORACLE DHT Manager:', error);
      return false;
    }
  }

  /**
   * Start the DHT manager and begin listening for network events
   */
  async start() {
    if (this.isStarted) {
      console.log('DHT Manager already started');
      return true;
    }

    try {
      this.isStarted = true;
      console.log('KNIRVORACLE DHT Manager started');
      console.log(`PeerID: ${this.peerId}`);

      // Connect to bootstrap peers
      await this.connectToBootstrapPeers();

      // Announce our presence as KNIRVORACLE
      await this.announceService();

      console.log('KNIRVORACLE DHT Manager fully operational');
      return true;
    } catch (error) {
      console.error('Failed to start KNIRVORACLE DHT Manager:', error);
      this.isStarted = false;
      return false;
    }
  }

  /**
   * Connect to bootstrap peers
   */
  async connectToBootstrapPeers() {
    for (const peerUrl of this.bootstrapPeers) {
      try {
        await this.connectToPeer(peerUrl);
      } catch (error) {
        console.warn(`Failed to connect to bootstrap peer ${peerUrl}:`, error.message);
      }
    }
  }

  /**
   * Connect to a specific peer
   */
  async connectToPeer(peerUrl) {
    if (this.connections.has(peerUrl)) {
      return; // Already connected
    }

    return new Promise((resolve, reject) => {
      const ws = new WebSocket(peerUrl);

      ws.on('open', () => {
        console.log(`Connected to peer: ${peerUrl}`);
        this.connections.set(peerUrl, ws);
        this.reconnectAttempts.delete(peerUrl);

        // Send identification message
        this.sendMessage(ws, {
          type: 'identify',
          peerId: this.peerId,
          serviceId: this.serviceID,
          chainId: this.chainID,
          timestamp: Date.now()
        });

        resolve();
      });

      ws.on('message', (data) => {
        try {
          const message = JSON.parse(data.toString());
          this.handleMessage(message, peerUrl);
        } catch (error) {
          console.error(`Error parsing message from ${peerUrl}:`, error);
        }
      });

      ws.on('close', () => {
        console.log(`Disconnected from peer: ${peerUrl}`);
        this.connections.delete(peerUrl);
        this.scheduleReconnect(peerUrl);
      });

      ws.on('error', (error) => {
        console.error(`WebSocket error for ${peerUrl}:`, error.message);
        reject(error);
      });

      // Timeout for connection
      setTimeout(() => {
        if (ws.readyState === WebSocket.CONNECTING) {
          ws.terminate();
          reject(new Error('Connection timeout'));
        }
      }, 10000);
    });
  }

  /**
   * Schedule reconnection to a peer
   */
  scheduleReconnect(peerUrl) {
    const attempts = this.reconnectAttempts.get(peerUrl) || 0;
    if (attempts >= this.maxReconnectAttempts) {
      console.log(`Max reconnection attempts reached for ${peerUrl}`);
      return;
    }

    const delay = Math.min(1000 * Math.pow(2, attempts), 30000); // Exponential backoff, max 30s
    this.reconnectAttempts.set(peerUrl, attempts + 1);

    setTimeout(async () => {
      if (this.isStarted && !this.connections.has(peerUrl)) {
        try {
          await this.connectToPeer(peerUrl);
        } catch (error) {
          console.warn(`Reconnection failed for ${peerUrl}:`, error.message);
        }
      }
    }, delay);
  }

  /**
   * Send a message to a WebSocket connection
   */
  sendMessage(ws, message) {
    if (ws.readyState === WebSocket.OPEN) {
      ws.send(JSON.stringify(message));
    }
  }

  /**
   * Broadcast a message to all connected peers
   */
  broadcast(message) {
    for (const [peerUrl, ws] of this.connections) {
      this.sendMessage(ws, message);
    }
  }

  /**
   * Handle incoming messages from peers
   */
  handleMessage(message, peerUrl) {
    try {
      switch (message.type) {
        case 'network_control':
          this.handleNetworkControl(message.data);
          break;
        case 'graph_announcement':
          this.handleGraphAnnouncement(message.data);
          break;
        case 'chain_announcement':
          this.handleChainAnnouncement(message.data);
          break;
        case 'nexus_announcement':
          this.handleNexusAnnouncement(message.data);
          break;
        case 'router_announcement':
          this.handleRouterAnnouncement(message.data);
          break;
        case 'identify':
          console.log(`Peer identified: ${message.peerId} (${message.serviceId})`);
          break;
        default:
          console.log(`Received unknown message type: ${message.type} from ${peerUrl}`);
      }
    } catch (error) {
      console.error('Error handling message:', error);
    }
  }

  /**
   * Handle network control messages (pause/resume)
   */
  handleNetworkControl(data) {
    console.log('Received network control message:', data);
    
    if (data.type === 'pause') {
      this.networkPaused = true;
      this.pausedUntil = new Date(Date.now() + DHT_CONFIG.NETWORK_PAUSE_TIMEOUT);
      console.log(`Network paused until: ${this.pausedUntil}`);
      
      // Notify listeners
      this.listeners.networkControl.forEach(listener => {
        try {
          listener({ type: 'pause', data });
        } catch (error) {
          console.error('Error in network control listener:', error);
        }
      });
    } else if (data.type === 'resume') {
      this.networkPaused = false;
      this.pausedUntil = null;
      console.log('Network resumed');
      
      // Notify listeners
      this.listeners.networkControl.forEach(listener => {
        try {
          listener({ type: 'resume', data });
        } catch (error) {
          console.error('Error in network control listener:', error);
        }
      });
    }
  }

  /**
   * Handle graph announcements from KNIRVGRAPH
   */
  handleGraphAnnouncement(data) {
    console.log('Received graph announcement:', data);
    
    // Notify listeners
    this.listeners.graphAnnouncement.forEach(listener => {
      try {
        listener(data);
      } catch (error) {
        console.error('Error in graph announcement listener:', error);
      }
    });
  }

  /**
   * Handle chain announcements from KNIRVCHAIN
   */
  handleChainAnnouncement(data) {
    console.log('Received chain announcement:', data);
    
    // Notify listeners
    this.listeners.chainAnnouncement.forEach(listener => {
      try {
        listener(data);
      } catch (error) {
        console.error('Error in chain announcement listener:', error);
      }
    });
  }

  /**
   * Handle nexus announcements from KNIRVSERVER
   */
  handleNexusAnnouncement(data) {
    console.log('Received nexus announcement:', data);
    
    // Notify listeners
    this.listeners.nexusAnnouncement.forEach(listener => {
      try {
        listener(data);
      } catch (error) {
        console.error('Error in nexus announcement listener:', error);
      }
    });
  }

  /**
   * Handle router announcements from KNIRVROUTER
   */
  handleRouterAnnouncement(data) {
    console.log('Received router announcement:', data);
    
    // Notify listeners
    this.listeners.routerAnnouncement.forEach(listener => {
      try {
        listener(data);
      } catch (error) {
        console.error('Error in router announcement listener:', error);
      }
    });
  }

  /**
   * Announce our service presence on the network
   */
  async announceService() {
    try {
      const announcement = {
        type: 'service_announcement',
        data: {
          peerId: this.peerId,
          serviceId: this.serviceID,
          chainId: this.chainID,
          timestamp: Date.now(),
          status: 'active'
        }
      };

      this.broadcast(announcement);
      console.log(`Announced KNIRVORACLE service: ${this.serviceID}`);
    } catch (error) {
      console.error('Failed to announce service:', error);
    }
  }

  /**
   * Add event listener for specific announcement types
   */
  addEventListener(type, listener) {
    if (this.listeners[type]) {
      this.listeners[type].push(listener);
    } else {
      console.warn(`Unknown event type: ${type}`);
    }
  }

  /**
   * Remove event listener
   */
  removeEventListener(type, listener) {
    if (this.listeners[type]) {
      const index = this.listeners[type].indexOf(listener);
      if (index > -1) {
        this.listeners[type].splice(index, 1);
      }
    }
  }

  /**
   * Check if network is currently paused
   */
  isNetworkPaused() {
    if (this.networkPaused && this.pausedUntil) {
      // Check if pause has expired
      if (Date.now() > this.pausedUntil.getTime()) {
        this.networkPaused = false;
        this.pausedUntil = null;
        console.log('Network pause expired, resuming operations');
      }
    }
    return this.networkPaused;
  }

  /**
   * Stop the DHT manager
   */
  async stop() {
    if (this.isStarted) {
      try {
        // Close all WebSocket connections
        for (const [peerUrl, ws] of this.connections) {
          ws.close();
        }
        this.connections.clear();
        this.reconnectAttempts.clear();

        this.isStarted = false;
        console.log('KNIRVORACLE DHT Manager stopped');
      } catch (error) {
        console.error('Error stopping DHT Manager:', error);
      }
    }
  }

  /**
   * Get current network status
   */
  getNetworkStatus() {
    return {
      isStarted: this.isStarted,
      networkPaused: this.networkPaused,
      pausedUntil: this.pausedUntil,
      peerId: this.peerId,
      connectedPeers: this.connections.size,
      bootstrapPeers: this.bootstrapPeers.length
    };
  }
}

module.exports = { DHTManager, DHT_CONFIG };
