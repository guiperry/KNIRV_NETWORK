/**
 * KNIRVTESTNET Treasury Management Service
 * 
 * Manages KNIRVORACLE treasury NRV balance and automatic transfers to testnet faucet.
 * Monitors treasury health, funding thresholds, and economic flow sustainability.
 * Implements emergency funding procedures and balance management.
 * 
 * @typedef {Object} TreasuryStatus
 * @property {number} balance_nrv - Current treasury NRV balance
 * @property {number} reserved_nrv - Reserved NRV (not available for faucet)
 * @property {number} available_nrv - Available NRV for faucet funding
 * @property {string} health_status - Treasury health status
 * @property {number} last_funding_timestamp - Last faucet funding timestamp
 * 
 * @typedef {Object} FundingConfig
 * @property {number} auto_transfer_threshold - Minimum treasury balance for auto transfer
 * @property {number} transfer_percentage - Percentage of new NRVs to transfer
 * @property {number} emergency_threshold - Emergency funding threshold
 * @property {number} max_daily_transfer - Maximum daily transfer limit
 */

const axios = require('axios');
const { FaucetMetrics } = require('../utils/metrics');

class TreasuryManagerService {
  /**
   * @param {Object} config - Treasury management configuration
   * @param {string} config.oracleEndpoint - KNIRVORACLE API endpoint
   * @param {number} config.checkInterval - Balance check interval in ms
   * @param {FaucetMetrics} metrics - Metrics collection instance
   */
  constructor(config, metrics) {
    this.config = {
      oracleEndpoint: config.oracleEndpoint || 'http://localhost:1317',
      checkInterval: config.checkInterval || 60000, // 1 minute
      timeout: config.timeout || 10000, // 10 seconds
      
      // Funding configuration
      auto_transfer_threshold: config.auto_transfer_threshold || 10000, // Min treasury balance
      transfer_percentage: config.transfer_percentage || 0.15, // 15% of new NRVs
      emergency_threshold: config.emergency_threshold || 1000, // Emergency funding threshold
      max_daily_transfer: config.max_daily_transfer || 50000, // Max daily transfer
      faucet_low_balance_threshold: config.faucet_low_balance_threshold || 5000,
      
      ...config
    };
    
    this.metrics = metrics;
    this.isMonitoring = false;
    this.monitoringInterval = null;
    this.lastTreasuryBalance = 0;
    this.dailyTransferAmount = 0;
    this.lastTransferReset = Date.now();
    this.transferHistory = [];
  }

  /**
   * Start monitoring treasury and automatic funding
   */
  async startMonitoring() {
    if (this.isMonitoring) {
      console.log('Treasury monitoring already active');
      return;
    }

    console.log('Starting Treasury management monitoring...');
    this.isMonitoring = true;

    // Initial treasury check
    await this.checkTreasuryHealth();

    // Set up periodic monitoring
    this.monitoringInterval = setInterval(async () => {
      try {
        await this.checkTreasuryHealth();
        await this.evaluateAutoFunding();
        await this.updateFundingMetrics();
        this.resetDailyLimitsIfNeeded();
      } catch (error) {
        console.error('Treasury monitoring error:', error.message);
        this.metrics.updateEconomicFlowHealth('treasury', 0);
      }
    }, this.config.checkInterval);

    console.log(`Treasury monitoring started (interval: ${this.config.checkInterval}ms)`);
  }

  /**
   * Stop monitoring
   */
  stopMonitoring() {
    if (this.monitoringInterval) {
      clearInterval(this.monitoringInterval);
      this.monitoringInterval = null;
    }
    this.isMonitoring = false;
    console.log('Treasury monitoring stopped');
  }

  /**
   * Check treasury health and balance
   * @returns {Promise<TreasuryStatus>} Treasury status information
   */
  async checkTreasuryHealth() {
    try {
      const response = await axios.get(`${this.config.oracleEndpoint}/api/treasury/status`, {
        timeout: this.config.timeout
      });

      const status = response.data;
      const balance = status.balance_nrv || 0;
      const reserved = status.reserved_nrv || 0;
      const available = balance - reserved;

      // Update metrics
      this.metrics.updateTreasuryBalance(balance);
      
      // Determine health status
      let healthStatus = 'healthy';
      if (balance < this.config.emergency_threshold) {
        healthStatus = 'critical';
        this.metrics.updateEconomicFlowHealth('treasury', 0);
      } else if (balance < this.config.auto_transfer_threshold) {
        healthStatus = 'low';
        this.metrics.updateEconomicFlowHealth('treasury', 0.5);
      } else {
        this.metrics.updateEconomicFlowHealth('treasury', 1);
      }

      // Track balance changes for funding rate calculation
      if (this.lastTreasuryBalance > 0) {
        const balanceChange = balance - this.lastTreasuryBalance;
        if (balanceChange > 0) {
          // Calculate funding rate (NRV per hour)
          const timeDiff = this.config.checkInterval / (1000 * 60 * 60); // Convert to hours
          const fundingRate = balanceChange / timeDiff;
          this.metrics.updateTreasuryFundingRate(fundingRate);
        }
      }
      this.lastTreasuryBalance = balance;

      return {
        balance_nrv: balance,
        reserved_nrv: reserved,
        available_nrv: available,
        health_status: healthStatus,
        last_funding_timestamp: status.last_funding_timestamp || 0,
        inflow_rate_nrv_per_hour: status.inflow_rate || 0,
        outflow_rate_nrv_per_hour: status.outflow_rate || 0
      };
    } catch (error) {
      console.error('Treasury health check failed:', error.message);
      this.metrics.updateEconomicFlowHealth('treasury', 0);
      
      return {
        balance_nrv: 0,
        reserved_nrv: 0,
        available_nrv: 0,
        health_status: 'unhealthy',
        error: error.message
      };
    }
  }

  /**
   * Evaluate if automatic funding should occur
   */
  async evaluateAutoFunding() {
    try {
      // Check faucet balance
      const faucetResponse = await axios.get(`${this.config.oracleEndpoint}/api/faucet/balance`, {
        timeout: this.config.timeout
      });
      
      const faucetBalance = faucetResponse.data.balance_nrv || 0;
      this.metrics.updateBalance(faucetBalance);

      // Check treasury status
      const treasuryStatus = await this.checkTreasuryHealth();
      
      // Determine if funding is needed
      const needsFunding = faucetBalance < this.config.faucet_low_balance_threshold;
      const treasuryCanFund = treasuryStatus.available_nrv > this.config.auto_transfer_threshold;
      const withinDailyLimit = this.dailyTransferAmount < this.config.max_daily_transfer;

      if (needsFunding && treasuryCanFund && withinDailyLimit) {
        // Calculate transfer amount
        const targetFaucetBalance = this.config.faucet_low_balance_threshold * 2; // 2x threshold
        const transferAmount = Math.min(
          targetFaucetBalance - faucetBalance,
          treasuryStatus.available_nrv * this.config.transfer_percentage,
          this.config.max_daily_transfer - this.dailyTransferAmount
        );

        if (transferAmount > 0) {
          await this.executeFunding(transferAmount, 'automatic');
        }
      }

      // Check for emergency funding
      if (faucetBalance < (this.config.faucet_low_balance_threshold * 0.1) && treasuryCanFund) {
        const emergencyAmount = Math.min(
          this.config.faucet_low_balance_threshold,
          treasuryStatus.available_nrv * 0.5
        );
        
        if (emergencyAmount > 0) {
          await this.executeFunding(emergencyAmount, 'emergency');
        }
      }

    } catch (error) {
      console.error('Auto funding evaluation failed:', error.message);
    }
  }

  /**
   * Execute funding transfer from treasury to faucet
   * @param {number} amount - Amount of NRV to transfer
   * @param {'automatic'|'emergency'|'manual'} type - Transfer type
   * @returns {Promise<Object>} Transfer result
   */
  async executeFunding(amount, type = 'manual') {
    try {
      console.log(`Executing ${type} treasury funding: ${amount} NRV`);

      const response = await axios.post(`${this.config.oracleEndpoint}/api/treasury/transfer`, {
        destination: 'testnet_faucet',
        amount: amount,
        type: type,
        timestamp: Date.now()
      }, {
        timeout: this.config.timeout * 2 // Longer timeout for transfers
      });

      if (response.data.success) {
        // Update metrics
        this.metrics.recordTreasuryTransfer('to_faucet', amount);
        this.metrics.recordTreasuryTransfer('total', amount);
        
        // Update daily transfer tracking
        this.dailyTransferAmount += amount;
        
        // Record transfer history
        this.transferHistory.push({
          timestamp: Date.now(),
          amount: amount,
          type: type,
          tx_hash: response.data.tx_hash || null
        });

        // Keep only last 30 days of history
        const thirtyDaysAgo = Date.now() - (30 * 24 * 60 * 60 * 1000);
        this.transferHistory = this.transferHistory.filter(t => t.timestamp > thirtyDaysAgo);

        console.log(`Treasury funding successful: ${amount} NRV (${type})`);
        
        return {
          success: true,
          amount: amount,
          type: type,
          tx_hash: response.data.tx_hash,
          timestamp: Date.now()
        };
      } else {
        throw new Error(response.data.error || 'Transfer failed');
      }

    } catch (error) {
      console.error(`Treasury funding failed (${type}):`, error.message);
      
      return {
        success: false,
        amount: amount,
        type: type,
        error: error.message,
        timestamp: Date.now()
      };
    }
  }

  /**
   * Update funding sustainability metrics
   */
  async updateFundingMetrics() {
    try {
      const treasuryStatus = await this.checkTreasuryHealth();
      const faucetBalance = this.metrics.metrics.balance_nrv;
      
      // Calculate funding sustainability
      const dailyFaucetUsage = this.estimateDailyFaucetUsage();
      const totalAvailable = treasuryStatus.available_nrv + faucetBalance;
      const sustainabilityDays = dailyFaucetUsage > 0 ? totalAvailable / dailyFaucetUsage : 999;
      
      this.metrics.updateFundingSustainability(Math.floor(sustainabilityDays));

    } catch (error) {
      console.error('Failed to update funding metrics:', error.message);
    }
  }

  /**
   * Estimate daily faucet usage based on recent history
   * @returns {number} Estimated daily NRV usage
   */
  estimateDailyFaucetUsage() {
    // This would typically analyze faucet request history
    // For now, return a conservative estimate
    const averageRequestAmount = 1000; // NRV per request
    const estimatedDailyRequests = 50; // Requests per day
    return averageRequestAmount * estimatedDailyRequests;
  }

  /**
   * Reset daily transfer limits if needed
   */
  resetDailyLimitsIfNeeded() {
    const now = Date.now();
    const oneDayMs = 24 * 60 * 60 * 1000;
    
    if (now - this.lastTransferReset > oneDayMs) {
      this.dailyTransferAmount = 0;
      this.lastTransferReset = now;
      console.log('Daily transfer limits reset');
    }
  }

  /**
   * Get treasury management status
   * @returns {Promise<Object>} Complete treasury status
   */
  async getTreasuryStatus() {
    const treasuryHealth = await this.checkTreasuryHealth();
    const faucetBalance = this.metrics.metrics.balance_nrv;
    
    return {
      treasury_health: treasuryHealth,
      faucet_balance_nrv: faucetBalance,
      daily_transfer_used: this.dailyTransferAmount,
      daily_transfer_limit: this.config.max_daily_transfer,
      transfer_history: this.transferHistory.slice(-10), // Last 10 transfers
      monitoring_active: this.isMonitoring,
      funding_sustainability_days: this.metrics.metrics.funding_sustainability_days,
      auto_funding_config: {
        transfer_threshold: this.config.auto_transfer_threshold,
        transfer_percentage: this.config.transfer_percentage,
        emergency_threshold: this.config.emergency_threshold,
        faucet_low_threshold: this.config.faucet_low_balance_threshold
      },
      last_check: new Date().toISOString()
    };
  }

  /**
   * Manual treasury funding (admin function)
   * @param {number} amount - Amount to transfer
   * @returns {Promise<Object>} Transfer result
   */
  async manualFunding(amount) {
    console.log(`Manual treasury funding requested: ${amount} NRV`);
    
    // Validate amount
    if (amount <= 0 || amount > this.config.max_daily_transfer) {
      return {
        success: false,
        error: `Invalid amount. Must be between 1 and ${this.config.max_daily_transfer} NRV`,
        timestamp: Date.now()
      };
    }

    // Check daily limits
    if (this.dailyTransferAmount + amount > this.config.max_daily_transfer) {
      return {
        success: false,
        error: `Would exceed daily transfer limit (${this.config.max_daily_transfer} NRV)`,
        timestamp: Date.now()
      };
    }

    return await this.executeFunding(amount, 'manual');
  }
}

module.exports = { TreasuryManagerService };
