/**
 * KNIRVTESTNET Faucet Handler
 * 
 * Core business logic for NRV faucet operations including request processing,
 * rate limiting, validation, and integration with KNIRVORACLE for token distribution.
 * Handles the complete faucet request lifecycle with comprehensive error handling.
 * 
 * @typedef {Object} FaucetRequest
 * @property {string} address - Wallet address for NRV distribution
 * @property {number} amount - Amount of NRV tokens requested
 * @property {string} reason - Optional reason for the request
 * @property {string} ip - Client IP address
 * @property {number} timestamp - Request timestamp
 * 
 * @typedef {Object} FaucetResponse
 * @property {boolean} success - Whether request was successful
 * @property {string} request_id - Unique request identifier
 * @property {string} tx_hash - Transaction hash (if successful)
 * @property {number} amount - Amount distributed
 * @property {string} status - Request status
 * @property {string} estimated_confirmation - Estimated confirmation time
 */

const fs = require('fs').promises;
const path = require('path');
const crypto = require('crypto');
const axios = require('axios');
const { FaucetMetrics } = require('../utils/metrics');
const { RouterIntegrationService } = require('../services/router-integration');
const { TreasuryManagerService } = require('../services/treasury-manager');

class FaucetHandler {
  /**
   * @param {Object} config - Faucet configuration
   * @param {FaucetMetrics} metrics - Metrics collection instance
   */
  constructor(config) {
    this.config = {
      enabled: config.enabled !== false,
      daily_limit: config.daily_limit || 10000,
      per_ip_hourly_limit: config.per_ip_hourly_limit || 5,
      per_address_daily_limit: config.per_address_daily_limit || 1000,
      cooldown_minutes: config.cooldown_minutes || 15,
      default_amount: config.default_amount || 1000,
      max_custom_amount: config.max_custom_amount || 5000,
      min_amount: config.min_amount || 100,
      knirvoracle_endpoint: config.knirvoracle_endpoint || 'http://localhost:1317',
      database_path: config.database_path || './data/faucet-requests.json',
      ...config
    };

    this.metrics = new FaucetMetrics();
    this.routerService = new RouterIntegrationService(this.config, this.metrics);
    this.treasuryService = new TreasuryManagerService(this.config, this.metrics);
    
    // Rate limiting storage
    this.rateLimits = {
      ip: new Map(), // IP -> { count, resetTime }
      address: new Map(), // Address -> { count, resetTime }
      global: { count: 0, resetTime: Date.now() + 60 * 60 * 1000 } // Hourly reset
    };

    // Request tracking
    this.requestHistory = [];
    this.activeRequests = new Set();

    // Initialize services
    this.initializeServices();
  }

  /**
   * Initialize monitoring services
   */
  async initializeServices() {
    try {
      await this.loadRequestHistory();
      await this.routerService.startMonitoring();
      await this.treasuryService.startMonitoring();
      console.log('Faucet services initialized successfully');
    } catch (error) {
      console.error('Failed to initialize faucet services:', error.message);
    }
  }

  /**
   * Process a faucet request
   * @param {FaucetRequest} request - Faucet request data
   * @returns {Promise<FaucetResponse>} Request result
   */
  async processRequest(request) {
    const startTime = Date.now();
    const requestId = this.generateRequestId();
    
    try {
      // Update active requests
      this.activeRequests.add(requestId);
      this.metrics.updateActiveRequests(this.activeRequests.size);

      // Validate faucet is enabled
      if (!this.config.enabled) {
        throw new Error('Faucet is currently disabled');
      }

      // Validate request
      const validation = await this.validateRequest(request);
      if (!validation.valid) {
        this.recordRequest(request, 'rejected', startTime, requestId, validation.error);
        this.metrics.recordRequest('rejected', (Date.now() - startTime) / 1000);
        return {
          success: false,
          request_id: requestId,
          error: validation.error,
          status: 'rejected',
          timestamp: Date.now()
        };
      }

      // Check rate limits
      const rateLimitCheck = this.checkRateLimits(request.ip, request.address);
      if (!rateLimitCheck.allowed) {
        this.recordRequest(request, 'rejected', startTime, requestId, rateLimitCheck.reason);
        this.metrics.recordRequest('rejected', (Date.now() - startTime) / 1000);
        this.metrics.recordRateLimitHit(rateLimitCheck.type);
        return {
          success: false,
          request_id: requestId,
          error: rateLimitCheck.reason,
          status: 'rate_limited',
          retry_after: rateLimitCheck.retry_after,
          timestamp: Date.now()
        };
      }

      // Execute NRV distribution
      const distribution = await this.distributeNRV(request.address, request.amount);
      
      if (distribution.success) {
        // Update rate limits
        this.updateRateLimits(request.ip, request.address);
        
        // Record successful request
        this.recordRequest(request, 'success', startTime, requestId, null, distribution.tx_hash);
        this.metrics.recordRequest('success', (Date.now() - startTime) / 1000);
        
        return {
          success: true,
          request_id: requestId,
          tx_hash: distribution.tx_hash,
          amount: request.amount,
          status: 'completed',
          estimated_confirmation: '30s',
          timestamp: Date.now()
        };
      } else {
        throw new Error(distribution.error || 'Distribution failed');
      }

    } catch (error) {
      console.error(`Faucet request ${requestId} failed:`, error.message);
      this.recordRequest(request, 'failed', startTime, requestId, error.message);
      this.metrics.recordRequest('failed', (Date.now() - startTime) / 1000);
      
      return {
        success: false,
        request_id: requestId,
        error: error.message,
        status: 'failed',
        timestamp: Date.now()
      };
    } finally {
      // Clean up active requests
      this.activeRequests.delete(requestId);
      this.metrics.updateActiveRequests(this.activeRequests.size);
    }
  }

  /**
   * Validate faucet request
   * @param {FaucetRequest} request - Request to validate
   * @returns {Object} Validation result
   */
  async validateRequest(request) {
    // Validate address format
    if (!request.address || typeof request.address !== 'string') {
      return { valid: false, error: 'Invalid wallet address format' };
    }

    // Basic KNIRV address validation (starts with 'knirv1')
    if (!request.address.startsWith('knirv1') || request.address.length < 20) {
      return { valid: false, error: 'Invalid KNIRV wallet address format' };
    }

    // Validate amount
    if (!request.amount || typeof request.amount !== 'number') {
      return { valid: false, error: 'Invalid amount specified' };
    }

    if (request.amount < this.config.min_amount || request.amount > this.config.max_custom_amount) {
      return { 
        valid: false, 
        error: `Amount must be between ${this.config.min_amount} and ${this.config.max_custom_amount} NRV` 
      };
    }

    // Check faucet balance
    const faucetBalance = this.metrics.metrics.balance_nrv;
    if (faucetBalance < request.amount) {
      return { valid: false, error: 'Insufficient faucet balance. Please try again later.' };
    }

    return { valid: true };
  }

  /**
   * Check rate limits for IP and address
   * @param {string} ip - Client IP address
   * @param {string} address - Wallet address
   * @returns {Object} Rate limit check result
   */
  checkRateLimits(ip, address) {
    const now = Date.now();

    // Check IP rate limit (hourly)
    const ipLimit = this.rateLimits.ip.get(ip);
    if (ipLimit && ipLimit.resetTime > now) {
      if (ipLimit.count >= this.config.per_ip_hourly_limit) {
        return {
          allowed: false,
          type: 'ip',
          reason: `IP rate limit exceeded. Maximum ${this.config.per_ip_hourly_limit} requests per hour.`,
          retry_after: Math.ceil((ipLimit.resetTime - now) / 1000)
        };
      }
    }

    // Check address rate limit (daily)
    const addressLimit = this.rateLimits.address.get(address);
    if (addressLimit && addressLimit.resetTime > now) {
      if (addressLimit.count >= this.config.per_address_daily_limit) {
        return {
          allowed: false,
          type: 'address',
          reason: `Address rate limit exceeded. Maximum ${this.config.per_address_daily_limit} NRV per day.`,
          retry_after: Math.ceil((addressLimit.resetTime - now) / 1000)
        };
      }
    }

    // Check global rate limit (hourly)
    if (this.rateLimits.global.resetTime > now) {
      if (this.rateLimits.global.count >= this.config.daily_limit / 24) { // Hourly portion of daily limit
        return {
          allowed: false,
          type: 'global',
          reason: 'Global rate limit exceeded. Please try again later.',
          retry_after: Math.ceil((this.rateLimits.global.resetTime - now) / 1000)
        };
      }
    }

    return { allowed: true };
  }

  /**
   * Update rate limits after successful request
   * @param {string} ip - Client IP address
   * @param {string} address - Wallet address
   */
  updateRateLimits(ip, address) {
    const now = Date.now();
    const oneHour = 60 * 60 * 1000;
    const oneDay = 24 * 60 * 60 * 1000;

    // Update IP rate limit
    const ipLimit = this.rateLimits.ip.get(ip);
    if (!ipLimit || ipLimit.resetTime <= now) {
      this.rateLimits.ip.set(ip, { count: 1, resetTime: now + oneHour });
    } else {
      ipLimit.count++;
    }

    // Update address rate limit
    const addressLimit = this.rateLimits.address.get(address);
    if (!addressLimit || addressLimit.resetTime <= now) {
      this.rateLimits.address.set(address, { count: 1, resetTime: now + oneDay });
    } else {
      addressLimit.count++;
    }

    // Update global rate limit
    if (this.rateLimits.global.resetTime <= now) {
      this.rateLimits.global = { count: 1, resetTime: now + oneHour };
    } else {
      this.rateLimits.global.count++;
    }
  }

  /**
   * Distribute NRV tokens via KNIRVORACLE
   * @param {string} address - Recipient address
   * @param {number} amount - Amount to distribute
   * @returns {Promise<Object>} Distribution result
   */
  async distributeNRV(address, amount) {
    try {
      const response = await axios.post(`${this.config.knirvoracle_endpoint}/api/mint/nrv`, {
        recipient: address,
        amount: amount,
        source: 'testnet_faucet',
        timestamp: Date.now()
      }, {
        timeout: 30000,
        headers: {
          'Content-Type': 'application/json',
          'User-Agent': 'KNIRV-Testnet-Faucet/1.0'
        }
      });

      if (response.data.success) {
        // Update faucet balance
        const newBalance = this.metrics.metrics.balance_nrv - amount;
        this.metrics.updateBalance(Math.max(0, newBalance));
        
        return {
          success: true,
          tx_hash: response.data.tx_hash || response.data.transaction_id,
          amount: amount
        };
      } else {
        return {
          success: false,
          error: response.data.error || 'KNIRVORACLE distribution failed'
        };
      }

    } catch (error) {
      console.error('NRV distribution error:', error.message);
      return {
        success: false,
        error: `Distribution service error: ${error.message}`
      };
    }
  }

  /**
   * Record request in history
   * @param {FaucetRequest} request - Original request
   * @param {string} status - Request status
   * @param {number} startTime - Request start time
   * @param {string} requestId - Request ID
   * @param {string} error - Error message (if any)
   * @param {string} txHash - Transaction hash (if successful)
   */
  recordRequest(request, status, startTime, requestId, error = null, txHash = null) {
    const record = {
      request_id: requestId,
      timestamp: Date.now(),
      ip: request.ip,
      address: request.address,
      amount: request.amount,
      reason: request.reason || '',
      status: status,
      processing_time_ms: Date.now() - startTime,
      error: error,
      tx_hash: txHash
    };

    this.requestHistory.push(record);
    
    // Keep only last 1000 requests in memory
    if (this.requestHistory.length > 1000) {
      this.requestHistory = this.requestHistory.slice(-1000);
    }

    // Save to persistent storage
    this.saveRequestHistory().catch(err => {
      console.error('Failed to save request history:', err.message);
    });
  }

  /**
   * Generate unique request ID
   * @returns {string} Unique request identifier
   */
  generateRequestId() {
    return `faucet_req_${Date.now()}_${crypto.randomBytes(4).toString('hex')}`;
  }

  /**
   * Load request history from storage
   */
  async loadRequestHistory() {
    try {
      const data = await fs.readFile(this.config.database_path, 'utf8');
      this.requestHistory = JSON.parse(data);
      console.log(`Loaded ${this.requestHistory.length} faucet request records`);
    } catch (error) {
      if (error.code !== 'ENOENT') {
        console.error('Failed to load request history:', error.message);
      }
      this.requestHistory = [];
    }
  }

  /**
   * Save request history to storage
   */
  async saveRequestHistory() {
    try {
      // Ensure directory exists
      const dir = path.dirname(this.config.database_path);
      await fs.mkdir(dir, { recursive: true });
      
      // Save request history
      await fs.writeFile(this.config.database_path, JSON.stringify(this.requestHistory, null, 2));
    } catch (error) {
      console.error('Failed to save request history:', error.message);
    }
  }

  /**
   * Get faucet status
   * @returns {Object} Current faucet status
   */
  getFaucetStatus() {
    const now = Date.now();
    const todayStart = new Date().setHours(0, 0, 0, 0);
    const todayRequests = this.requestHistory.filter(r => r.timestamp >= todayStart);
    const todayDistributed = todayRequests
      .filter(r => r.status === 'success')
      .reduce((sum, r) => sum + r.amount, 0);

    return {
      faucet_enabled: this.config.enabled,
      daily_limit: this.config.daily_limit,
      remaining_today: Math.max(0, this.config.daily_limit - todayDistributed),
      distributed_today: todayDistributed,
      current_balance: this.metrics.metrics.balance_nrv,
      rate_limits: {
        per_ip_hourly: this.config.per_ip_hourly_limit,
        per_address_daily: this.config.per_address_daily_limit,
        cooldown_minutes: this.config.cooldown_minutes
      },
      current_queue_size: this.activeRequests.size,
      total_requests_today: todayRequests.length,
      success_rate_today: todayRequests.length > 0 ? 
        (todayRequests.filter(r => r.status === 'success').length / todayRequests.length * 100).toFixed(2) : 0,
      last_request: this.requestHistory.length > 0 ? 
        this.requestHistory[this.requestHistory.length - 1].timestamp : 0
    };
  }

  /**
   * Get request history for an address
   * @param {string} address - Wallet address
   * @param {number} limit - Maximum number of records
   * @returns {Array} Request history
   */
  getAddressHistory(address, limit = 10) {
    return this.requestHistory
      .filter(r => r.address === address)
      .slice(-limit)
      .reverse();
  }

  /**
   * Get comprehensive faucet health
   * @returns {Promise<Object>} Health status
   */
  async getHealth() {
    const routerStatus = await this.routerService.getIntegrationStatus();
    const treasuryStatus = await this.treasuryService.getTreasuryStatus();
    const faucetStatus = this.getFaucetStatus();
    const metrics = this.metrics.getSummary();

    return {
      service: 'testnet-faucet',
      status: this.config.enabled ? 'healthy' : 'disabled',
      timestamp: new Date().toISOString(),
      version: '1.0.0',
      faucet: faucetStatus,
      router_integration: routerStatus,
      treasury_management: treasuryStatus,
      metrics: metrics,
      economic_flow_health: this.metrics.metrics.economic_flow_health
    };
  }

  /**
   * Get Prometheus metrics
   * @returns {string} Prometheus metrics
   */
  getMetrics() {
    return this.metrics.getPrometheusMetrics();
  }

  /**
   * Manual treasury funding (admin function)
   * @param {number} amount - Amount to fund
   * @returns {Promise<Object>} Funding result
   */
  async manualTreasuryFunding(amount) {
    return await this.treasuryService.manualFunding(amount);
  }

  /**
   * Get economic flow metrics
   * @returns {Object} Economic flow status
   */
  getEconomicFlowMetrics() {
    return {
      router_health: this.metrics.metrics.economic_flow_health.router,
      treasury_health: this.metrics.metrics.economic_flow_health.treasury,
      faucet_health: this.metrics.metrics.economic_flow_health.faucet,
      funding_sustainability_days: this.metrics.metrics.funding_sustainability_days,
      router_proof_rate: this.metrics.metrics.router_proof_generation_rate,
      treasury_funding_rate: this.metrics.metrics.treasury_funding_rate,
      current_balance: this.metrics.metrics.balance_nrv,
      treasury_balance: this.metrics.metrics.treasury_balance_nrv
    };
  }

  /**
   * Cleanup and shutdown
   */
  async shutdown() {
    console.log('Shutting down faucet services...');

    try {
      await this.saveRequestHistory();
      this.routerService.stopMonitoring();
      this.treasuryService.stopMonitoring();
      console.log('Faucet services shutdown complete');
    } catch (error) {
      console.error('Error during faucet shutdown:', error.message);
    }
  }
}

module.exports = { FaucetHandler };
