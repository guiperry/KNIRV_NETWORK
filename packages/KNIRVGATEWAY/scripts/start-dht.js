#!/usr/bin/env node

/**
 * KNIRVORACLE DHT Startup Script
 * 
 * Initializes and starts the DHT service for network monitoring.
 * Can be run standalone or integrated with the main KNIRVORACLE startup.
 */

const { DHTService } = require('../lib/services/dht_service');
const fs = require('fs').promises;
const path = require('path');

class DHTStarter {
  constructor() {
    this.dhtService = null;
    this.configPath = path.join(__dirname, '../config/dht-config.json');
    this.logPath = path.join(__dirname, '../logs/dht.log');
  }

  /**
   * Load DHT configuration from file or environment
   */
  async loadConfig() {
    try {
      // Try to load from config file first
      try {
        const configData = await fs.readFile(this.configPath, 'utf8');
        const config = JSON.parse(configData);
        console.log('Loaded DHT config from file:', this.configPath);
        return config;
      } catch (fileError) {
        console.log('No config file found, using environment variables and defaults');
      }

      // Fallback to environment variables and defaults
      const config = {
        chainID: process.env.KNIRV_CHAIN_ID || 'testnet',
        serviceID: 'knirvoracle',
        enableLogging: process.env.KNIRV_DHT_LOGGING !== 'false',
        bootstrapPeers: this.getBootstrapPeers(),
        autoStart: process.env.KNIRV_DHT_AUTO_START !== 'false'
      };

      return config;
    } catch (error) {
      console.error('Error loading DHT config:', error);
      return this.getDefaultConfig();
    }
  }

  /**
   * Get bootstrap peers from environment or defaults
   */
  getBootstrapPeers() {
    const envPeers = process.env.KNIRV_BOOTSTRAP_PEERS;
    if (envPeers) {
      return envPeers.split(',').map(peer => peer.trim()).filter(peer => peer.length > 0);
    }

    // Default bootstrap peers for testnet
    return [
      '/ip4/127.0.0.1/tcp/4001/p2p/12D3KooWBootstrapPeer1',
      '/ip4/127.0.0.1/tcp/4002/p2p/12D3KooWBootstrapPeer2'
    ];
  }

  /**
   * Get default configuration
   */
  getDefaultConfig() {
    return {
      chainID: 'testnet',
      serviceID: 'knirvoracle',
      enableLogging: true,
      bootstrapPeers: this.getBootstrapPeers(),
      autoStart: true
    };
  }

  /**
   * Initialize the DHT service
   */
  async initialize() {
    try {
      console.log('Initializing KNIRVORACLE DHT Service...');
      
      const config = await this.loadConfig();
      console.log('DHT Configuration:', {
        chainID: config.chainID,
        serviceID: config.serviceID,
        bootstrapPeers: config.bootstrapPeers.length,
        enableLogging: config.enableLogging
      });

      this.dhtService = new DHTService();
      const success = await this.dhtService.initialize(config);
      
      if (!success) {
        throw new Error('Failed to initialize DHT Service');
      }

      console.log('DHT Service initialized successfully');
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
    if (!this.dhtService) {
      throw new Error('DHT Service not initialized');
    }

    try {
      console.log('Starting KNIRVORACLE DHT Service...');
      const success = await this.dhtService.start();
      
      if (success) {
        console.log('DHT Service started successfully');
        this.setupEventHandlers();
        this.startHealthMonitoring();
        return true;
      } else {
        throw new Error('Failed to start DHT Service');
      }
    } catch (error) {
      console.error('Failed to start DHT Service:', error);
      return false;
    }
  }

  /**
   * Set up event handlers for monitoring
   */
  setupEventHandlers() {
    // Log network events
    setInterval(() => {
      const events = this.dhtService.getNetworkEvents(5);
      if (events.length > 0) {
        console.log('Recent network events:', events.length);
        events.forEach(event => {
          console.log(`  ${event.timestamp}: ${event.type} - ${event.service || event.action}`);
        });
      }
    }, 30000); // Every 30 seconds

    // Handle process termination
    process.on('SIGINT', () => {
      console.log('\nReceived SIGINT, shutting down DHT Service...');
      this.shutdown();
    });

    process.on('SIGTERM', () => {
      console.log('\nReceived SIGTERM, shutting down DHT Service...');
      this.shutdown();
    });
  }

  /**
   * Start health monitoring
   */
  startHealthMonitoring() {
    setInterval(async () => {
      try {
        const health = await this.dhtService.healthCheck();
        if (health.status !== 'healthy') {
          console.warn('DHT Service health check failed:', health);
        }
      } catch (error) {
        console.error('Health check error:', error);
      }
    }, 60000); // Every minute
  }

  /**
   * Get current status
   */
  async getStatus() {
    if (!this.dhtService) {
      return { status: 'not_initialized' };
    }

    try {
      const health = await this.dhtService.healthCheck();
      const networkStatus = this.dhtService.getNetworkStatus();
      const recentEvents = this.dhtService.getNetworkEvents(10);

      return {
        status: 'running',
        health,
        network: networkStatus,
        recentEvents: recentEvents.length,
        timestamp: new Date().toISOString()
      };
    } catch (error) {
      return {
        status: 'error',
        error: error.message,
        timestamp: new Date().toISOString()
      };
    }
  }

  /**
   * Export current status to file
   */
  async exportStatus(filePath) {
    try {
      const status = await this.getStatus();
      await fs.writeFile(filePath, JSON.stringify(status, null, 2));
      console.log(`Status exported to: ${filePath}`);
      return true;
    } catch (error) {
      console.error('Failed to export status:', error);
      return false;
    }
  }

  /**
   * Shutdown the DHT service
   */
  async shutdown() {
    if (this.dhtService) {
      try {
        await this.dhtService.stop();
        console.log('DHT Service shutdown complete');
      } catch (error) {
        console.error('Error during shutdown:', error);
      }
    }
    process.exit(0);
  }
}

// CLI handling
async function main() {
  const starter = new DHTStarter();
  const command = process.argv[2];

  switch (command) {
    case 'start':
      try {
        await starter.initialize();
        await starter.start();
        console.log('DHT Service is running. Press Ctrl+C to stop.');
        // Keep the process alive
        setInterval(() => {}, 1000);
      } catch (error) {
        console.error('Failed to start DHT Service:', error);
        process.exit(1);
      }
      break;

    case 'status':
      try {
        await starter.initialize();
        const status = await starter.getStatus();
        console.log(JSON.stringify(status, null, 2));
      } catch (error) {
        console.error('Failed to get status:', error);
        process.exit(1);
      }
      break;

    case 'export':
      const outputFile = process.argv[3] || './dht-status.json';
      try {
        await starter.initialize();
        await starter.exportStatus(outputFile);
      } catch (error) {
        console.error('Failed to export status:', error);
        process.exit(1);
      }
      break;

    default:
      console.log('KNIRVORACLE DHT Starter');
      console.log('Usage:');
      console.log('  node start-dht.js start    - Start the DHT service');
      console.log('  node start-dht.js status   - Get current status');
      console.log('  node start-dht.js export [file] - Export status to file');
      break;
  }
}

// Run if called directly
if (require.main === module) {
  main().catch(error => {
    console.error('Unhandled error:', error);
    process.exit(1);
  });
}

module.exports = { DHTStarter };
