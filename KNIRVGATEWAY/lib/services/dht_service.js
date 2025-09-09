/**
 * KNIRVGATEWAY DHT Service
 * 
 * Service wrapper for DHT Manager that integrates with KNIRVGATEWAY's
 * existing architecture and provides network monitoring capabilities.
 */

const { DHTManager } = require('../p2p/dht_manager');
const fs = require('fs').promises;
const path = require('path');

class DHTService {
  constructor() {
    this.dhtManager = null;
    this.isInitialized = false;
    this.config = null;
    this.networkEvents = [];
    this.maxEventHistory = 1000;
    
    // Network status tracking
    this.networkStatus = {
      lastUpdate: null,
      services: {
        knirvgraph: { status: 'unknown', lastSeen: null },
        knirvchain: { status: 'unknown', lastSeen: null },
        knirvnexus: { status: 'unknown', lastSeen: null },
        knirvrouter: { status: 'unknown', lastSeen: null },
        knirvoracle: { status: 'unknown', lastSeen: null }
      },
      networkPaused: false,
      pausedUntil: null
    };
  }

  /**
   * Initialize the DHT service with configuration
   */
  async initialize(config = {}) {
    try {
      this.config = {
        chainID: config.chainID || process.env.KNIRV_CHAIN_ID || 'testnet',
        serviceID: config.serviceID || 'knirvgateway',
        bootstrapPeers: config.bootstrapPeers || this.getDefaultBootstrapPeers(),
        enableLogging: config.enableLogging !== false,
        ...config
      };

      // Create DHT manager
      this.dhtManager = new DHTManager(this.config.chainID, this.config.serviceID);

      // Set up event listeners
      this.setupEventListeners();

      // Initialize DHT manager
      const success = await this.dhtManager.initialize(this.config.bootstrapPeers);
      if (!success) {
        throw new Error('Failed to initialize DHT Manager');
      }

      this.isInitialized = true;
      console.log('KNIRVGATEWAY DHT Service initialized successfully');
      return true;
    } catch (error) {
      console.error('Failed to initialize DHT Service:', error);
      return false;
    }
  }

  /**
   * Start the DHT service
   */
  async start() {
    if (!this.isInitialized) {
      throw new Error('DHT Service not initialized. Call initialize() first.');
    }

    try {
      const success = await this.dhtManager.start();
      if (success) {
        console.log('KNIRVGATEWAY DHT Service started');
        this.updateNetworkStatus();
      }
      return success;
    } catch (error) {
      console.error('Failed to start DHT Service:', error);
      return false;
    }
  }

  /**
   * Set up event listeners for network announcements
   */
  setupEventListeners() {
    // Network control events (pause/resume)
    this.dhtManager.addEventListener('networkControl', (event) => {
      this.handleNetworkControl(event);
    });

    // Graph announcements
    this.dhtManager.addEventListener('graphAnnouncement', (data) => {
      this.handleServiceAnnouncement('knirvgraph', data);
    });

    // Chain announcements
    this.dhtManager.addEventListener('chainAnnouncement', (data) => {
      this.handleServiceAnnouncement('knirvchain', data);
    });

    // Nexus announcements
    this.dhtManager.addEventListener('nexusAnnouncement', (data) => {
      this.handleServiceAnnouncement('knirvnexus', data);
    });

    // Router announcements
    this.dhtManager.addEventListener('routerAnnouncement', (data) => {
      this.handleServiceAnnouncement('knirvrouter', data);
    });
  }

  /**
   * Handle network control events
   */
  handleNetworkControl(event) {
    const timestamp = new Date().toISOString();
    
    if (event.type === 'pause') {
      this.networkStatus.networkPaused = true;
      this.networkStatus.pausedUntil = event.data.pausedUntil;
      console.log(`Network paused by KNIRVORACLE until: ${event.data.pausedUntil}`);
    } else if (event.type === 'resume') {
      this.networkStatus.networkPaused = false;
      this.networkStatus.pausedUntil = null;
      console.log('Network resumed by KNIRVORACLE');
    }

    // Add to event history
    this.addNetworkEvent({
      type: 'network_control',
      action: event.type,
      timestamp,
      data: event.data
    });

    this.updateNetworkStatus();
  }

  /**
   * Handle service announcements
   */
  handleServiceAnnouncement(serviceName, data) {
    const timestamp = new Date().toISOString();
    
    // Update service status
    if (this.networkStatus.services[serviceName]) {
      this.networkStatus.services[serviceName].status = 'active';
      this.networkStatus.services[serviceName].lastSeen = timestamp;
    }

    // Add to event history
    this.addNetworkEvent({
      type: 'service_announcement',
      service: serviceName,
      timestamp,
      data
    });

    if (this.config.enableLogging) {
      console.log(`Received announcement from ${serviceName}:`, data);
    }

    this.updateNetworkStatus();
  }

  /**
   * Add event to network event history
   */
  addNetworkEvent(event) {
    this.networkEvents.unshift(event);
    
    // Limit event history size
    if (this.networkEvents.length > this.maxEventHistory) {
      this.networkEvents = this.networkEvents.slice(0, this.maxEventHistory);
    }
  }

  /**
   * Update network status timestamp
   */
  updateNetworkStatus() {
    this.networkStatus.lastUpdate = new Date().toISOString();
  }

  /**
   * Get current network status
   */
  getNetworkStatus() {
    const dhtStatus = this.dhtManager ? this.dhtManager.getNetworkStatus() : null;
    
    return {
      ...this.networkStatus,
      dht: dhtStatus,
      isInitialized: this.isInitialized,
      config: {
        chainID: this.config?.chainID,
        serviceID: this.config?.serviceID,
        bootstrapPeers: this.config?.bootstrapPeers?.length || 0
      }
    };
  }

  /**
   * Get recent network events
   */
  getNetworkEvents(limit = 50) {
    return this.networkEvents.slice(0, limit);
  }

  /**
   * Get events for a specific service
   */
  getServiceEvents(serviceName, limit = 20) {
    return this.networkEvents
      .filter(event => event.service === serviceName)
      .slice(0, limit);
  }

  /**
   * Check if network is currently paused
   */
  isNetworkPaused() {
    return this.dhtManager ? this.dhtManager.isNetworkPaused() : false;
  }

  /**
   * Get default bootstrap peers from environment or config
   */
  getDefaultBootstrapPeers() {
    const envPeers = process.env.KNIRV_BOOTSTRAP_PEERS;
    if (envPeers) {
      return envPeers.split(',').map(peer => peer.trim());
    }

    // Default testnet bootstrap peers
    return [
      '/ip4/127.0.0.1/tcp/4001/p2p/12D3KooWBootstrapPeer1',
      '/ip4/127.0.0.1/tcp/4002/p2p/12D3KooWBootstrapPeer2'
    ];
  }

  /**
   * Export network status to file
   */
  async exportNetworkStatus(filePath) {
    try {
      const status = this.getNetworkStatus();
      const events = this.getNetworkEvents();
      
      const exportData = {
        timestamp: new Date().toISOString(),
        status,
        events,
        metadata: {
          version: '1.0.0',
          service: 'knirvgateway-dht'
        }
      };

      await fs.writeFile(filePath, JSON.stringify(exportData, null, 2));
      console.log(`Network status exported to: ${filePath}`);
      return true;
    } catch (error) {
      console.error('Failed to export network status:', error);
      return false;
    }
  }

  /**
   * Stop the DHT service
   */
  async stop() {
    if (this.dhtManager) {
      try {
        await this.dhtManager.stop();
        console.log('KNIRVGATEWAY DHT Service stopped');
      } catch (error) {
        console.error('Error stopping DHT Service:', error);
      }
    }
  }

  /**
   * Health check for the DHT service
   */
  async healthCheck() {
    if (!this.isInitialized || !this.dhtManager) {
      return {
        status: 'unhealthy',
        reason: 'DHT Service not initialized'
      };
    }

    const dhtStatus = this.dhtManager.getNetworkStatus();
    
    return {
      status: dhtStatus.isStarted ? 'healthy' : 'unhealthy',
      dht: dhtStatus,
      network: {
        paused: this.isNetworkPaused(),
        lastUpdate: this.networkStatus.lastUpdate,
        activeServices: Object.keys(this.networkStatus.services).filter(
          service => this.networkStatus.services[service].status === 'active'
        ).length
      }
    };
  }
}

module.exports = { DHTService };
