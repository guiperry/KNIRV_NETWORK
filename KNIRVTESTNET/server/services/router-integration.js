/**
 * KNIRVTESTNET ROUTER Integration Service
 * 
 * Monitors KNIRVROUTER connectivity proof generation and NRV creation.
 * Tracks the economic flow from ROUTER proofs to treasury funding.
 * Provides alerts and metrics for proof generation health.
 * 
 * @typedef {Object} RouterStatus
 * @property {boolean} proof_engine_active - Whether proof engine is running
 * @property {number} proofs_generated_today - Proofs generated in last 24h
 * @property {number} nrv_creation_rate - NRVs created per hour
 * @property {number} last_proof_timestamp - Last successful proof timestamp
 * @property {string} health_status - Overall router health status
 * 
 * @typedef {Object} ProofMetrics
 * @property {number} total_proofs - Total connectivity proofs generated
 * @property {number} successful_proofs - Successful proof validations
 * @property {number} failed_proofs - Failed proof attempts
 * @property {number} average_proof_time - Average proof generation time
 * @property {number} nrv_rewards_issued - Total NRV rewards issued
 */

const axios = require('axios');
const { FaucetMetrics } = require('../utils/metrics');

class RouterIntegrationService {
  /**
   * @param {Object} config - Router integration configuration
   * @param {string} config.routerEndpoint - KNIRVROUTER API endpoint
   * @param {string} config.oracleEndpoint - KNIRVORACLE API endpoint
   * @param {number} config.checkInterval - Health check interval in ms
   * @param {FaucetMetrics} metrics - Metrics collection instance
   */
  constructor(config, metrics) {
    this.config = {
      routerEndpoint: config.routerEndpoint || 'http://localhost:8086',
      oracleEndpoint: config.oracleEndpoint || 'http://localhost:1317',
      checkInterval: config.checkInterval || 30000, // 30 seconds
      timeout: config.timeout || 10000, // 10 seconds
      ...config
    };
    
    this.metrics = metrics;
    this.isMonitoring = false;
    this.monitoringInterval = null;
    this.lastProofCheck = 0;
    this.proofHistory = [];
  }

  /**
   * Start monitoring ROUTER connectivity proofs
   */
  async startMonitoring() {
    if (this.isMonitoring) {
      console.log('Router monitoring already active');
      return;
    }

    console.log('Starting ROUTER integration monitoring...');
    this.isMonitoring = true;

    // Initial health check
    await this.checkRouterHealth();

    // Set up periodic monitoring
    this.monitoringInterval = setInterval(async () => {
      try {
        await this.checkRouterHealth();
        await this.updateProofMetrics();
        await this.checkNRVCreationFlow();
      } catch (error) {
        console.error('Router monitoring error:', error.message);
        this.metrics.updateEconomicFlowHealth('router', 0);
      }
    }, this.config.checkInterval);

    console.log(`Router monitoring started (interval: ${this.config.checkInterval}ms)`);
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
    console.log('Router monitoring stopped');
  }

  /**
   * Check ROUTER health and proof engine status
   * @returns {Promise<RouterStatus>} Router status information
   */
  async checkRouterHealth() {
    try {
      const response = await axios.get(`${this.config.routerEndpoint}/api/connectivity/status`, {
        timeout: this.config.timeout
      });

      const status = response.data;
      const isHealthy = status.proof_engine_active && status.connectivity_score > 0.5;

      // Update metrics
      this.metrics.updateEconomicFlowHealth('router', isHealthy ? 1 : 0);
      
      if (status.proofs_generated_today) {
        this.metrics.recordRouterProofs(status.proofs_generated_today - this.lastProofCheck);
        this.lastProofCheck = status.proofs_generated_today;
      }

      return {
        proof_engine_active: status.proof_engine_active || false,
        proofs_generated_today: status.proofs_generated_today || 0,
        nrv_creation_rate: status.nrv_creation_rate || 0,
        last_proof_timestamp: status.last_proof_timestamp || 0,
        health_status: isHealthy ? 'healthy' : 'degraded',
        connectivity_score: status.connectivity_score || 0,
        peer_count: status.peer_count || 0
      };
    } catch (error) {
      console.error('Router health check failed:', error.message);
      this.metrics.updateEconomicFlowHealth('router', 0);
      
      return {
        proof_engine_active: false,
        proofs_generated_today: 0,
        nrv_creation_rate: 0,
        last_proof_timestamp: 0,
        health_status: 'unhealthy',
        error: error.message
      };
    }
  }

  /**
   * Update proof generation metrics
   */
  async updateProofMetrics() {
    try {
      const response = await axios.get(`${this.config.routerEndpoint}/api/connectivity/metrics`, {
        timeout: this.config.timeout
      });

      const metrics = response.data;
      
      // Update proof generation rate (proofs per hour)
      if (metrics.proof_generation_rate) {
        this.metrics.updateRouterProofRate(metrics.proof_generation_rate);
      }

      // Track proof history for trend analysis
      this.proofHistory.push({
        timestamp: Date.now(),
        proofs_count: metrics.total_proofs || 0,
        success_rate: metrics.success_rate || 0
      });

      // Keep only last 24 hours of history
      const oneDayAgo = Date.now() - (24 * 60 * 60 * 1000);
      this.proofHistory = this.proofHistory.filter(entry => entry.timestamp > oneDayAgo);

    } catch (error) {
      console.error('Failed to update proof metrics:', error.message);
    }
  }

  /**
   * Check NRV creation flow from proofs to treasury
   */
  async checkNRVCreationFlow() {
    try {
      // Check ROUTER NRV minting requests
      const routerResponse = await axios.get(`${this.config.routerEndpoint}/api/nrv/minting-status`, {
        timeout: this.config.timeout
      });

      const mintingStatus = routerResponse.data;
      
      // Update minting metrics
      if (mintingStatus.successful_mints) {
        this.metrics.recordRouterMinting('success');
      }
      if (mintingStatus.failed_mints) {
        this.metrics.recordRouterMinting('failed');
      }

      // Verify treasury receives NRVs from ROUTER
      const treasuryResponse = await axios.get(`${this.config.oracleEndpoint}/api/treasury/nrv-inflow`, {
        timeout: this.config.timeout
      });

      const treasuryInflow = treasuryResponse.data;
      
      // Check if treasury is receiving NRVs from ROUTER proofs
      const recentInflow = treasuryInflow.recent_inflow_nrv || 0;
      const expectedInflow = mintingStatus.successful_mints * (mintingStatus.nrv_per_proof || 10);
      
      // Alert if treasury inflow doesn't match ROUTER output
      if (recentInflow < expectedInflow * 0.8) { // 20% tolerance
        console.warn('Treasury NRV inflow lower than expected from ROUTER proofs');
        this.metrics.updateEconomicFlowHealth('treasury', 0.5); // Degraded
      }

    } catch (error) {
      console.error('NRV creation flow check failed:', error.message);
    }
  }

  /**
   * Get proof generation trends
   * @returns {Object} Proof generation trend analysis
   */
  getProofTrends() {
    if (this.proofHistory.length < 2) {
      return { trend: 'insufficient_data', change_rate: 0 };
    }

    const recent = this.proofHistory.slice(-6); // Last 6 data points
    const older = this.proofHistory.slice(-12, -6); // Previous 6 data points

    if (older.length === 0) {
      return { trend: 'insufficient_data', change_rate: 0 };
    }

    const recentAvg = recent.reduce((sum, entry) => sum + entry.proofs_count, 0) / recent.length;
    const olderAvg = older.reduce((sum, entry) => sum + entry.proofs_count, 0) / older.length;

    const changeRate = olderAvg > 0 ? ((recentAvg - olderAvg) / olderAvg) * 100 : 0;

    let trend = 'stable';
    if (changeRate > 10) trend = 'increasing';
    else if (changeRate < -10) trend = 'decreasing';

    return {
      trend,
      change_rate: changeRate,
      recent_average: recentAvg,
      previous_average: olderAvg,
      data_points: this.proofHistory.length
    };
  }

  /**
   * Estimate NRV production capacity
   * @returns {Object} Production capacity estimates
   */
  estimateNRVProduction() {
    const trends = this.getProofTrends();
    const currentRate = this.metrics.metrics.router_proof_generation_rate;
    
    // Estimate daily/weekly/monthly NRV production
    const nrvPerProof = 10; // Default NRV reward per proof
    const dailyProofs = currentRate * 24;
    const weeklyProofs = dailyProofs * 7;
    const monthlyProofs = dailyProofs * 30;

    return {
      current_hourly_rate: currentRate,
      estimated_daily_nrv: dailyProofs * nrvPerProof,
      estimated_weekly_nrv: weeklyProofs * nrvPerProof,
      estimated_monthly_nrv: monthlyProofs * nrvPerProof,
      trend_direction: trends.trend,
      confidence: this.proofHistory.length >= 24 ? 'high' : 'low'
    };
  }

  /**
   * Get comprehensive ROUTER integration status
   * @returns {Promise<Object>} Complete integration status
   */
  async getIntegrationStatus() {
    const routerHealth = await this.checkRouterHealth();
    const trends = this.getProofTrends();
    const production = this.estimateNRVProduction();

    return {
      router_health: routerHealth,
      proof_trends: trends,
      nrv_production: production,
      monitoring_active: this.isMonitoring,
      last_check: new Date().toISOString(),
      economic_flow_health: this.metrics.metrics.economic_flow_health.router
    };
  }

  /**
   * Force a manual proof generation check (for testing)
   * @returns {Promise<Object>} Manual check results
   */
  async manualProofCheck() {
    console.log('Performing manual ROUTER proof check...');
    
    try {
      const health = await this.checkRouterHealth();
      await this.updateProofMetrics();
      await this.checkNRVCreationFlow();
      
      return {
        success: true,
        router_health: health,
        timestamp: new Date().toISOString()
      };
    } catch (error) {
      return {
        success: false,
        error: error.message,
        timestamp: new Date().toISOString()
      };
    }
  }
}

module.exports = { RouterIntegrationService };
