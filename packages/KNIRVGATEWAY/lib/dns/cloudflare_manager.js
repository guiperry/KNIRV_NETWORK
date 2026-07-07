/**
 * CloudFlare DNS Manager for KNIRVORACLE
 * 
 * Handles automated DNS failover and management for the private DHT deployment.
 * Manages DNS records for oracle.knirv.network and related services.
 * 
 * Features:
 * - Automated DNS record updates
 * - Failover detection and management
 * - Leader election for DNS updates
 * - Health monitoring integration
 * - Secure API key management
 */

import axios from 'axios';
import crypto from 'crypto';
import { EventEmitter } from 'events';

// CloudFlare API configuration
const CLOUDFLARE_CONFIG = {
  API_BASE_URL: 'https://api.cloudflare.com/client/v4',
  ZONE_NAME: 'knirv.network',
  GATEWAY_RECORD_NAME: 'oracle.knirv.network',
  TTL: 60, // 60 seconds for fast failover
  HEALTH_CHECK_INTERVAL: 30000, // 30 seconds
  FAILOVER_THRESHOLD: 3, // 3 failed checks before failover
  LEADER_ELECTION_TIMEOUT: 60000, // 1 minute
};

// Gateway services configuration
const GATEWAY_SERVICES = {
  payment_oracle: {
    domain: 'payments.knirv.network',
    port: 3001,
    subdomain: 'payments'
  },
  tunnel_registry: {
    domain: 'tunnel.knirv.network',
    port: 3004,
    subdomain: 'tunnel'
  },
  operator_registry: {
    domain: 'operators.knirv.network',
    port: 3007,
    subdomain: 'operators'
  },
  webgui: {
    domain: 'dashboard.knirv.network',
    port: 3007,
    subdomain: 'dashboard'
  }
};

class CloudFlareManager extends EventEmitter {
  constructor(options = {}) {
    super();
    
    this.apiToken = options.apiToken || process.env.CLOUDFLARE_API_TOKEN;
    this.zoneId = options.zoneId || process.env.CLOUDFLARE_ZONE_ID;
    this.oracleRecordId = options.oracleRecordId || process.env.CLOUDFLARE_GATEWAY_RECORD_ID;
    
    // Instance configuration
    this.instanceId = options.instanceId || this.generateInstanceId();
    this.instanceIP = options.instanceIP || process.env.INSTANCE_IP;
    this.isPrimary = options.isPrimary || false;
    this.isLeader = false;
    
    // Health monitoring
    this.healthCheckInterval = null;
    this.failedChecks = 0;
    this.lastHealthCheck = null;
    this.primaryGatewayURL = options.primaryGatewayURL || 'https://oracle.knirv.network';
    
    // Leader election
    this.leaderElectionInterval = null;
    this.lastLeaderHeartbeat = null;
    this.leaderElectionKey = `leader:oracle:${Date.now()}`;
    
    // DNS record cache
    this.currentRecord = null;
    this.recordCache = new Map();
    
    if (!this.apiToken) {
      throw new Error('CloudFlare API token is required');
    }
    
    if (!this.zoneId) {
      console.warn('CloudFlare Zone ID not provided - will attempt to discover');
    }
  }

  /**
   * Initialize the CloudFlare manager
   */
  async initialize() {
    try {
      console.log('[CloudFlare] Initializing DNS manager...');
      
      // Discover zone ID if not provided
      if (!this.zoneId) {
        this.zoneId = await this.discoverZoneId();
      }
      
      // Get current oracle record
      await this.getCurrentGatewayRecord();
      
      // Start health monitoring
      this.startHealthMonitoring();
      
      // Start leader election if not primary
      if (!this.isPrimary) {
        this.startLeaderElection();
      } else {
        this.isLeader = true;
        console.log('[CloudFlare] Instance configured as primary leader');
      }
      
      console.log('[CloudFlare] DNS manager initialized successfully');
      this.emit('initialized', {
        instanceId: this.instanceId,
        isLeader: this.isLeader,
        isPrimary: this.isPrimary
      });
      
      return true;
    } catch (error) {
      console.error('[CloudFlare] Failed to initialize:', error);
      this.emit('error', error);
      return false;
    }
  }

  /**
   * Generate unique instance ID
   */
  generateInstanceId() {
    return `oracle-${crypto.randomBytes(8).toString('hex')}`;
  }

  /**
   * Discover CloudFlare zone ID for knirv.network
   */
  async discoverZoneId() {
    try {
      const response = await this.makeAPIRequest('GET', '/zones', {
        name: CLOUDFLARE_CONFIG.ZONE_NAME
      });
      
      if (response.result && response.result.length > 0) {
        const zoneId = response.result[0].id;
        console.log(`[CloudFlare] Discovered zone ID: ${zoneId}`);
        return zoneId;
      } else {
        throw new Error(`Zone ${CLOUDFLARE_CONFIG.ZONE_NAME} not found`);
      }
    } catch (error) {
      throw new Error(`Failed to discover zone ID: ${error.message}`);
    }
  }

  /**
   * Get current oracle DNS record
   */
  async getCurrentGatewayRecord() {
    try {
      const response = await this.makeAPIRequest('GET', `/zones/${this.zoneId}/dns_records`, {
        name: CLOUDFLARE_CONFIG.GATEWAY_RECORD_NAME,
        type: 'A'
      });
      
      if (response.result && response.result.length > 0) {
        this.currentRecord = response.result[0];
        this.oracleRecordId = this.currentRecord.id;
        console.log(`[CloudFlare] Current oracle record: ${this.currentRecord.content}`);
      } else {
        console.log('[CloudFlare] No existing oracle record found');
      }
      
      return this.currentRecord;
    } catch (error) {
      console.error('[CloudFlare] Failed to get current record:', error);
      throw error;
    }
  }

  /**
   * Update oracle DNS record
   */
  async updateGatewayRecord(newIP, reason = 'Manual update') {
    try {
      if (!this.isLeader) {
        console.log('[CloudFlare] Not leader - skipping DNS update');
        return false;
      }

      console.log(`[CloudFlare] Updating oracle record to ${newIP} (${reason})`);

      const recordData = {
        type: 'A',
        name: CLOUDFLARE_CONFIG.GATEWAY_RECORD_NAME,
        content: newIP,
        ttl: CLOUDFLARE_CONFIG.TTL,
        comment: `Updated by ${this.instanceId} - ${reason}`
      };

      let response;
      if (this.oracleRecordId) {
        // Update existing record
        response = await this.makeAPIRequest('PUT', `/zones/${this.zoneId}/dns_records/${this.oracleRecordId}`, recordData);
      } else {
        // Create new record
        response = await this.makeAPIRequest('POST', `/zones/${this.zoneId}/dns_records`, recordData);
        this.oracleRecordId = response.result.id;
      }

      this.currentRecord = response.result;

      console.log(`[CloudFlare] DNS record updated successfully: ${newIP}`);
      this.emit('dns:updated', {
        oldIP: this.currentRecord?.content,
        newIP: newIP,
        reason: reason,
        timestamp: Date.now()
      });

      return true;
    } catch (error) {
      console.error('[CloudFlare] Failed to update DNS record:', error);
      this.emit('dns:error', error);
      throw error;
    }
  }

  /**
   * Update all oracle service DNS records
   */
  async updateGatewayServiceRecords(newIP, reason = 'Gateway service update') {
    try {
      if (!this.isLeader) {
        console.log('[CloudFlare] Not leader - skipping oracle service DNS updates');
        return false;
      }

      console.log(`[CloudFlare] Updating oracle service records to ${newIP} (${reason})`);

      const results = [];
      for (const [serviceName, serviceConfig] of Object.entries(GATEWAY_SERVICES)) {
        try {
          const success = await this.updateServiceRecord(serviceName, serviceConfig, newIP, reason);
          results.push({ service: serviceName, success });
        } catch (error) {
          console.error(`[CloudFlare] Failed to update ${serviceName}:`, error.message);
          results.push({ service: serviceName, success: false, error: error.message });
        }
      }

      const successCount = results.filter(r => r.success).length;
      console.log(`[CloudFlare] Updated ${successCount}/${results.length} oracle service records`);

      return results;
    } catch (error) {
      console.error('[CloudFlare] Failed to update oracle service records:', error);
      throw error;
    }
  }

  /**
   * Update individual service DNS record
   */
  async updateServiceRecord(serviceName, serviceConfig, newIP, reason) {
    try {
      // Get existing record
      const existingRecord = await this.getServiceRecord(serviceConfig.subdomain);

      const recordData = {
        type: 'A',
        name: serviceConfig.domain,
        content: newIP,
        ttl: CLOUDFLARE_CONFIG.TTL,
        comment: `${serviceName} - Updated by ${this.instanceId} - ${reason}`
      };

      let response;
      if (existingRecord) {
        // Update existing record
        response = await this.makeAPIRequest('PUT', `/zones/${this.zoneId}/dns_records/${existingRecord.id}`, recordData);
      } else {
        // Create new record
        response = await this.makeAPIRequest('POST', `/zones/${this.zoneId}/dns_records`, recordData);
      }

      console.log(`[CloudFlare] ✅ ${serviceName} DNS record updated: ${serviceConfig.domain} -> ${newIP}:${serviceConfig.port}`);

      this.emit('service:dns:updated', {
        service: serviceName,
        domain: serviceConfig.domain,
        oldIP: existingRecord?.content,
        newIP: newIP,
        port: serviceConfig.port,
        reason: reason,
        timestamp: Date.now()
      });

      return true;
    } catch (error) {
      console.error(`[CloudFlare] Failed to update ${serviceName} DNS record:`, error);
      throw error;
    }
  }

  /**
   * Get existing DNS record for a service subdomain
   */
  async getServiceRecord(subdomain) {
    const recordName = `${subdomain}.knirv.network`;

    try {
      const response = await this.makeAPIRequest('GET', `/zones/${this.zoneId}/dns_records`, {
        name: recordName,
        type: 'A'
      });

      return response.result && response.result.length > 0 ? response.result[0] : null;
    } catch (error) {
      console.error(`[CloudFlare] Failed to get service record for ${recordName}:`, error);
      return null;
    }
  }

  /**
   * Start health monitoring of primary oracle
   */
  startHealthMonitoring() {
    if (this.healthCheckInterval) {
      clearInterval(this.healthCheckInterval);
    }
    
    this.healthCheckInterval = setInterval(async () => {
      await this.performHealthCheck();
    }, CLOUDFLARE_CONFIG.HEALTH_CHECK_INTERVAL);
    
    console.log('[CloudFlare] Health monitoring started');
  }

  /**
   * Perform health check on primary oracle
   */
  async performHealthCheck() {
    try {
      const response = await axios.get(`${this.primaryGatewayURL}/health`, {
        timeout: 10000,
        headers: {
          'User-Agent': `KNIRVORACLE-HealthCheck/${this.instanceId}`
        }
      });
      
      if (response.status === 200) {
        // Health check passed
        this.failedChecks = 0;
        this.lastHealthCheck = Date.now();
        
        this.emit('health:check', {
          status: 'healthy',
          response: response.data,
          timestamp: this.lastHealthCheck
        });
      } else {
        throw new Error(`Unexpected status: ${response.status}`);
      }
    } catch (error) {
      this.failedChecks++;
      console.error(`[CloudFlare] Health check failed (${this.failedChecks}/${CLOUDFLARE_CONFIG.FAILOVER_THRESHOLD}):`, error.message);
      
      this.emit('health:check', {
        status: 'unhealthy',
        error: error.message,
        failedChecks: this.failedChecks,
        timestamp: Date.now()
      });
      
      // Trigger failover if threshold reached
      if (this.failedChecks >= CLOUDFLARE_CONFIG.FAILOVER_THRESHOLD) {
        await this.triggerFailover();
      }
    }
  }

  /**
   * Trigger DNS failover
   */
  async triggerFailover() {
    try {
      console.log('[CloudFlare] Triggering DNS failover...');
      
      if (!this.instanceIP) {
        throw new Error('Instance IP not configured for failover');
      }
      
      // Attempt to become leader
      const becameLeader = await this.attemptLeaderElection();
      if (!becameLeader) {
        console.log('[CloudFlare] Another instance is handling failover');
        return false;
      }
      
      // Update DNS to point to this instance
      await this.updateGatewayRecord(this.instanceIP, 'Automatic failover');
      
      // Reset failed checks
      this.failedChecks = 0;
      
      console.log('[CloudFlare] Failover completed successfully');
      this.emit('failover:completed', {
        newIP: this.instanceIP,
        timestamp: Date.now()
      });
      
      return true;
    } catch (error) {
      console.error('[CloudFlare] Failover failed:', error);
      this.emit('failover:failed', error);
      return false;
    }
  }

  /**
   * Start leader election process
   */
  startLeaderElection() {
    if (this.leaderElectionInterval) {
      clearInterval(this.leaderElectionInterval);
    }
    
    this.leaderElectionInterval = setInterval(async () => {
      await this.checkLeaderStatus();
    }, CLOUDFLARE_CONFIG.LEADER_ELECTION_TIMEOUT / 2);
    
    console.log('[CloudFlare] Leader election started');
  }

  /**
   * Check leader status and attempt election if needed
   */
  async checkLeaderStatus() {
    try {
      // Simple leader election using DNS record comments
      const record = await this.getCurrentGatewayRecord();
      
      if (record && record.comment) {
        const commentMatch = record.comment.match(/Updated by ([^-]+)/);
        if (commentMatch) {
          const leaderId = commentMatch[1];
          
          if (leaderId === this.instanceId) {
            this.isLeader = true;
            this.lastLeaderHeartbeat = Date.now();
          } else {
            // Check if leader is still active (based on record age)
            const recordAge = Date.now() - new Date(record.modified_on).getTime();
            if (recordAge > CLOUDFLARE_CONFIG.LEADER_ELECTION_TIMEOUT * 2) {
              console.log('[CloudFlare] Leader appears inactive, attempting election');
              await this.attemptLeaderElection();
            }
          }
        }
      }
    } catch (error) {
      console.error('[CloudFlare] Leader election check failed:', error);
    }
  }

  /**
   * Attempt to become leader
   */
  async attemptLeaderElection() {
    try {
      // Simple leader election: try to update DNS record with our instance ID
      if (this.currentRecord) {
        const updatedRecord = {
          ...this.currentRecord,
          comment: `Updated by ${this.instanceId} - Leader election at ${new Date().toISOString()}`
        };
        
        await this.makeAPIRequest('PUT', `/zones/${this.zoneId}/dns_records/${this.oracleRecordId}`, updatedRecord);
        
        this.isLeader = true;
        this.lastLeaderHeartbeat = Date.now();
        
        console.log('[CloudFlare] Successfully became leader');
        this.emit('leader:elected', { instanceId: this.instanceId });
        
        return true;
      }
      
      return false;
    } catch (error) {
      console.error('[CloudFlare] Leader election failed:', error);
      return false;
    }
  }

  /**
   * Make CloudFlare API request
   */
  async makeAPIRequest(method, endpoint, data = null) {
    const url = `${CLOUDFLARE_CONFIG.API_BASE_URL}${endpoint}`;
    
    const config = {
      method,
      url,
      headers: {
        'Authorization': `Bearer ${this.apiToken}`,
        'Content-Type': 'application/json'
      },
      timeout: 10000
    };
    
    if (data) {
      if (method === 'GET') {
        config.params = data;
      } else {
        config.data = data;
      }
    }
    
    try {
      const response = await axios(config);
      
      if (!response.data.success) {
        throw new Error(`CloudFlare API error: ${JSON.stringify(response.data.errors)}`);
      }
      
      return response.data;
    } catch (error) {
      if (error.response) {
        throw new Error(`CloudFlare API error: ${error.response.status} - ${JSON.stringify(error.response.data)}`);
      } else {
        throw new Error(`CloudFlare API request failed: ${error.message}`);
      }
    }
  }

  /**
   * Get current status
   */
  getStatus() {
    return {
      instanceId: this.instanceId,
      instanceIP: this.instanceIP,
      isPrimary: this.isPrimary,
      isLeader: this.isLeader,
      failedChecks: this.failedChecks,
      lastHealthCheck: this.lastHealthCheck,
      currentRecord: this.currentRecord,
      zoneId: this.zoneId,
      oracleRecordId: this.oracleRecordId
    };
  }

  /**
   * Stop the CloudFlare manager
   */
  async stop() {
    console.log('[CloudFlare] Stopping DNS manager...');
    
    if (this.healthCheckInterval) {
      clearInterval(this.healthCheckInterval);
      this.healthCheckInterval = null;
    }
    
    if (this.leaderElectionInterval) {
      clearInterval(this.leaderElectionInterval);
      this.leaderElectionInterval = null;
    }
    
    this.isLeader = false;
    
    console.log('[CloudFlare] DNS manager stopped');
    this.emit('stopped');
  }
}

export { CloudFlareManager, CLOUDFLARE_CONFIG };
