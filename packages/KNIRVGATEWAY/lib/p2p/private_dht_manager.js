/**
 * Private DHT Manager for KNIRVORACLE
 * 
 * Implements a libp2p-based private DHT for the KNIRV Network.
 * Supports both persistent (Render) and serverless (Netlify/Vercel) modes.
 * 
 * Features:
 * - Private DHT with custom protocol
 * - Bootstrap node functionality
 * - Peer discovery and management
 * - Service announcement and discovery
 * - Health monitoring
 */

// Load polyfills first
import '../polyfills.js';

// Core Node.js modules
import { EventEmitter } from 'events';
import crypto from 'crypto';

// Import libp2p modules
import { createLibp2p } from 'libp2p';
import { tcp } from '@libp2p/tcp';
import { mplex } from '@libp2p/mplex';
import { noise } from '@chainsafe/libp2p-noise';
import { kadDHT } from '@libp2p/kad-dht';
import { bootstrap } from '@libp2p/bootstrap';

// Private DHT configuration
const PRIVATE_DHT_CONFIG = {
  PROTOCOL: '/knirv/private-dht/1.0.0',
  SERVICE_NAMESPACE: 'knirvoracle',
  DISCOVERY_INTERVAL: 30000, // 30 seconds
  HEALTH_CHECK_INTERVAL: 60000, // 1 minute
  MAX_PEERS: 50,
  CONNECTION_TIMEOUT: 30000,
  BOOTSTRAP_RETRY_DELAY: 5000,
  MAX_BOOTSTRAP_RETRIES: 5
};

class PrivateDHTManager extends EventEmitter {
  constructor(chainID = 'testnet', serviceID = 'knirvoracle', options = {}) {
    super();
    
    this.chainID = chainID;
    this.serviceID = `${serviceID}-${chainID}`;
    this.options = {
      clientMode: options.clientMode || false,
      enableBootstrap: options.enableBootstrap !== false,
      listenPort: options.listenPort || 0,
      ...options
    };
    
    this.node = null;
    this.isStarted = false;
    this.isBootstrapNode = false;
    this.connectedPeers = new Map();
    this.discoveredServices = new Map();
    this.healthCheckInterval = null;
    this.discoveryInterval = null;
    
    // Bootstrap configuration
    this.bootstrapPeers = options.bootstrapPeers || this.getDefaultBootstrapPeers();
    this.bootstrapRetries = 0;
    
    // Service announcement
    this.serviceAnnouncement = {
      serviceId: this.serviceID,
      chainId: this.chainID,
      timestamp: Date.now(),
      capabilities: options.capabilities || ['oracle', 'bootstrap'],
      version: '1.0.0'
    };
  }

  /**
   * Initialize and start the private DHT
   */
  async initialize() {
    try {
      console.log(`[PrivateDHT] Initializing DHT for ${this.serviceID}...`);
      
      // Create libp2p node configuration
      const nodeConfig = {
        addresses: {
          listen: [`/ip4/0.0.0.0/tcp/${this.options.listenPort}`]
        },
        transports: [tcp()],
        streamMuxers: [mplex()],
        connectionEncryption: [noise()],
        dht: kadDHT({
          protocol: PRIVATE_DHT_CONFIG.PROTOCOL,
          clientMode: this.options.clientMode,
          validators: {},
          selectors: {}
        }),
        connectionManager: {
          maxConnections: PRIVATE_DHT_CONFIG.MAX_PEERS,
          minConnections: 5
        }
      };

      // Add bootstrap configuration if enabled
      if (this.options.enableBootstrap && this.bootstrapPeers.length > 0) {
        nodeConfig.peerDiscovery = [
          bootstrap({
            list: this.bootstrapPeers,
            timeout: PRIVATE_DHT_CONFIG.CONNECTION_TIMEOUT,
            tagName: 'bootstrap',
            tagValue: 100,
            tagTTL: 120000
          })
        ];
      }

      // Create the libp2p node
      this.node = await createLibp2p(nodeConfig);
      
      // Set up event listeners
      this.setupEventListeners();
      
      // Start the node
      await this.node.start();
      this.isStarted = true;
      
      console.log(`[PrivateDHT] Node started with Peer ID: ${this.node.peerId.toString()}`);
      console.log(`[PrivateDHT] Listening on:`, this.node.getMultiaddrs().map(addr => addr.toString()));
      
      // Determine if this is a bootstrap node
      this.isBootstrapNode = !this.options.clientMode;
      
      // Start periodic tasks
      this.startPeriodicTasks();
      
      // Announce this service to the DHT
      await this.announceService();
      
      this.emit('initialized', {
        peerId: this.node.peerId.toString(),
        multiaddrs: this.node.getMultiaddrs().map(addr => addr.toString()),
        isBootstrapNode: this.isBootstrapNode
      });
      
      return true;
    } catch (error) {
      console.error('[PrivateDHT] Failed to initialize:', error);
      this.emit('error', error);
      return false;
    }
  }

  /**
   * Set up event listeners for the libp2p node
   */
  setupEventListeners() {
    if (!this.node) return;

    // Connection events
    this.node.addEventListener('peer:connect', (event) => {
      const peerId = event.detail.toString();
      console.log(`[PrivateDHT] Peer connected: ${peerId}`);
      this.connectedPeers.set(peerId, {
        id: peerId,
        connectedAt: Date.now(),
        lastSeen: Date.now()
      });
      this.emit('peer:connect', { peerId });
    });

    this.node.addEventListener('peer:disconnect', (event) => {
      const peerId = event.detail.toString();
      console.log(`[PrivateDHT] Peer disconnected: ${peerId}`);
      this.connectedPeers.delete(peerId);
      this.emit('peer:disconnect', { peerId });
    });

    // DHT events
    this.node.addEventListener('peer:discovery', (event) => {
      const peerId = event.detail.id.toString();
      console.log(`[PrivateDHT] Peer discovered: ${peerId}`);
      this.emit('peer:discovery', { peerId });
    });
  }

  /**
   * Start periodic tasks (health checks, service discovery)
   */
  startPeriodicTasks() {
    // Health check interval
    this.healthCheckInterval = setInterval(() => {
      this.performHealthCheck();
    }, PRIVATE_DHT_CONFIG.HEALTH_CHECK_INTERVAL);

    // Service discovery interval
    this.discoveryInterval = setInterval(() => {
      this.discoverServices();
    }, PRIVATE_DHT_CONFIG.DISCOVERY_INTERVAL);
  }

  /**
   * Announce this service to the DHT
   */
  async announceService() {
    if (!this.node || !this.isStarted) return;

    try {
      const serviceKey = `service:${this.serviceID}`;
      const announcement = JSON.stringify(this.serviceAnnouncement);
      
      // Store service announcement in DHT
      await this.node.contentRouting.put(
        Buffer.from(serviceKey),
        Buffer.from(announcement)
      );
      
      console.log(`[PrivateDHT] Service announced: ${this.serviceID}`);
      this.emit('service:announced', this.serviceAnnouncement);
    } catch (error) {
      console.error('[PrivateDHT] Failed to announce service:', error);
    }
  }

  /**
   * Discover other services in the DHT
   */
  async discoverServices() {
    if (!this.node || !this.isStarted) return;

    try {
      // Look for other KNIRV services
      const serviceTypes = ['knirvoracle', 'knirvgraph', 'knirvchain', 'knirvserver', 'knirvrouter', 'knirvoracle'];
      
      for (const serviceType of serviceTypes) {
        const serviceKey = `service:${serviceType}-${this.chainID}`;
        
        try {
          const result = await this.node.contentRouting.get(Buffer.from(serviceKey));
          const serviceInfo = JSON.parse(result.toString());
          
          this.discoveredServices.set(serviceType, {
            ...serviceInfo,
            discoveredAt: Date.now()
          });
          
          this.emit('service:discovered', { serviceType, serviceInfo });
        } catch (error) {
          // Service not found or error retrieving - this is normal
        }
      }
    } catch (error) {
      console.error('[PrivateDHT] Service discovery error:', error);
    }
  }

  /**
   * Perform health check on connected peers
   */
  async performHealthCheck() {
    if (!this.node || !this.isStarted) return;

    const now = Date.now();
    const staleThreshold = 5 * 60 * 1000; // 5 minutes

    // Check for stale connections
    for (const [peerId, peerInfo] of this.connectedPeers) {
      if (now - peerInfo.lastSeen > staleThreshold) {
        console.log(`[PrivateDHT] Removing stale peer: ${peerId}`);
        this.connectedPeers.delete(peerId);
      }
    }

    // Emit health status
    this.emit('health:check', {
      connectedPeers: this.connectedPeers.size,
      discoveredServices: this.discoveredServices.size,
      isHealthy: this.isStarted && this.connectedPeers.size > 0
    });
  }

  /**
   * Get list of connected peers for /provision endpoint
   */
  getProvisionPeers() {
    if (!this.node || !this.isStarted) {
      return [];
    }

    const peers = new Set();
    
    // Add self to the list
    this.node.getMultiaddrs().forEach(addr => {
      peers.add(`${addr.toString()}/p2p/${this.node.peerId.toString()}`);
    });

    // Add connected peers
    for (const [peerId, peerInfo] of this.connectedPeers) {
      // Get peer multiaddrs from peerstore
      const peer = this.node.peerStore.get(peerId);
      if (peer && peer.addresses.length > 0) {
        peer.addresses.forEach(addr => {
          peers.add(`${addr.multiaddr.toString()}/p2p/${peerId}`);
        });
      }
    }

    return Array.from(peers);
  }

  /**
   * Get default bootstrap peers
   */
  getDefaultBootstrapPeers() {
    // These would be replaced with actual bootstrap peer addresses
    return [
      '/ip4/127.0.0.1/tcp/4001/p2p/12D3KooWBootstrapPeer1',
      '/ip4/127.0.0.1/tcp/4002/p2p/12D3KooWBootstrapPeer2'
    ];
  }

  /**
   * Get network status
   */
  getNetworkStatus() {
    return {
      isStarted: this.isStarted,
      isBootstrapNode: this.isBootstrapNode,
      peerId: this.node ? this.node.peerId.toString() : null,
      multiaddrs: this.node ? this.node.getMultiaddrs().map(addr => addr.toString()) : [],
      connectedPeers: this.connectedPeers.size,
      discoveredServices: Array.from(this.discoveredServices.keys()),
      serviceAnnouncement: this.serviceAnnouncement
    };
  }

  /**
   * Stop the DHT manager
   */
  async stop() {
    if (!this.isStarted) return;

    try {
      console.log('[PrivateDHT] Stopping DHT manager...');
      
      // Clear intervals
      if (this.healthCheckInterval) {
        clearInterval(this.healthCheckInterval);
        this.healthCheckInterval = null;
      }
      
      if (this.discoveryInterval) {
        clearInterval(this.discoveryInterval);
        this.discoveryInterval = null;
      }
      
      // Stop the libp2p node
      if (this.node) {
        await this.node.stop();
        this.node = null;
      }
      
      this.isStarted = false;
      this.connectedPeers.clear();
      this.discoveredServices.clear();
      
      console.log('[PrivateDHT] DHT manager stopped');
      this.emit('stopped');
    } catch (error) {
      console.error('[PrivateDHT] Error stopping DHT manager:', error);
      this.emit('error', error);
    }
  }
}

export { PrivateDHTManager, PRIVATE_DHT_CONFIG };
