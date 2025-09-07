/**
 * KNIRVTESTNET Faucet Metrics Collection System
 * 
 * Provides comprehensive metrics collection for faucet operations,
 * ROUTER integration, treasury management, and economic flow monitoring.
 * Generates Prometheus-compatible metrics for monitoring integration.
 * 
 * @typedef {Object} RequestMetrics
 * @property {number} success - Successful requests count
 * @property {number} rejected - Rejected requests count  
 * @property {number} failed - Failed requests count
 * 
 * @typedef {Object} HistogramBucket
 * @property {Object<string, number>} buckets - Histogram buckets
 * @property {number} sum - Total sum of values
 * @property {number} count - Total count of values
 * 
 * @typedef {Object} RateLimitMetrics
 * @property {number} ip - IP-based rate limit hits
 * @property {number} address - Address-based rate limit hits
 * @property {number} global - Global rate limit hits
 * 
 * @typedef {Object} RouterMetrics
 * @property {number} proofs_generated - ROUTER connectivity proofs generated
 * @property {Object} nrv_minting_requests - NRV minting request metrics
 * @property {number} proof_generation_rate - Proof generation rate per hour
 * 
 * @typedef {Object} TreasuryMetrics
 * @property {number} balance_nrv - Treasury NRV balance
 * @property {Object} transfers - Treasury transfer metrics
 * @property {number} funding_rate - Treasury funding rate NRV per hour
 * 
 * @typedef {Object} EconomicFlowHealth
 * @property {number} router - Router health (0=unhealthy, 1=healthy)
 * @property {number} treasury - Treasury health (0=unhealthy, 1=healthy)
 * @property {number} faucet - Faucet health (0=unhealthy, 1=healthy)
 */

class FaucetMetrics {
  constructor() {
    this.metrics = {
      // Faucet operation metrics
      requests_total: { success: 0, rejected: 0, failed: 0 },
      balance_nrv: 0,
      request_duration_histogram: {
        buckets: { '0.5': 0, '1.0': 0, '2.0': 0, '5.0': 0, 'Inf': 0 },
        sum: 0,
        count: 0
      },
      rate_limit_hits: { ip: 0, address: 0, global: 0 },
      active_requests: 0,
      last_request_timestamp: 0,

      // ROUTER integration metrics
      router_proofs_generated: 0,
      router_nrv_minting_requests: { success: 0, failed: 0 },
      router_proof_generation_rate: 0,

      // Treasury management metrics
      treasury_balance_nrv: 0,
      treasury_transfers: { to_faucet: 0, total: 0 },
      treasury_funding_rate: 0,

      // Economic flow health metrics
      economic_flow_health: { router: 1, treasury: 1, faucet: 1 },
      funding_sustainability_days: 0
    };

    // Initialize metrics collection start time
    this.startTime = Date.now();
  }

  /**
   * Record a faucet request with status and duration
   * @param {'success'|'rejected'|'failed'} status - Request status
   * @param {number} duration - Request duration in seconds
   */
  recordRequest(status, duration) {
    this.metrics.requests_total[status]++;
    this.recordDuration(duration);
    this.metrics.last_request_timestamp = Date.now();
  }

  /**
   * Record request duration in histogram
   * @param {number} duration - Duration in seconds
   */
  recordDuration(duration) {
    this.metrics.request_duration_histogram.sum += duration;
    this.metrics.request_duration_histogram.count++;

    // Update histogram buckets
    const buckets = ['0.5', '1.0', '2.0', '5.0', 'Inf'];
    for (const bucket of buckets) {
      if (duration <= parseFloat(bucket) || bucket === 'Inf') {
        this.metrics.request_duration_histogram.buckets[bucket]++;
      }
    }
  }

  /**
   * Update faucet NRV balance
   * @param {number} balance - Current NRV balance
   */
  updateBalance(balance) {
    this.metrics.balance_nrv = balance;
  }

  /**
   * Record rate limit hit
   * @param {'ip'|'address'|'global'} type - Rate limit type
   */
  recordRateLimitHit(type) {
    this.metrics.rate_limit_hits[type]++;
  }

  /**
   * Update active requests count
   * @param {number} count - Current active requests
   */
  updateActiveRequests(count) {
    this.metrics.active_requests = count;
  }

  /**
   * Record ROUTER proof generation
   * @param {number} count - Number of proofs generated
   */
  recordRouterProofs(count) {
    this.metrics.router_proofs_generated += count;
  }

  /**
   * Record ROUTER NRV minting request
   * @param {'success'|'failed'} status - Minting request status
   */
  recordRouterMinting(status) {
    this.metrics.router_nrv_minting_requests[status]++;
  }

  /**
   * Update ROUTER proof generation rate
   * @param {number} rate - Proofs per hour
   */
  updateRouterProofRate(rate) {
    this.metrics.router_proof_generation_rate = rate;
  }

  /**
   * Update treasury NRV balance
   * @param {number} balance - Treasury NRV balance
   */
  updateTreasuryBalance(balance) {
    this.metrics.treasury_balance_nrv = balance;
  }

  /**
   * Record treasury transfer
   * @param {'to_faucet'|'total'} type - Transfer type
   * @param {number} amount - Transfer amount
   */
  recordTreasuryTransfer(type, amount) {
    this.metrics.treasury_transfers[type] += amount;
  }

  /**
   * Update treasury funding rate
   * @param {number} rate - NRV per hour funding rate
   */
  updateTreasuryFundingRate(rate) {
    this.metrics.treasury_funding_rate = rate;
  }

  /**
   * Update economic flow health
   * @param {'router'|'treasury'|'faucet'} component - Component name
   * @param {number} health - Health status (0=unhealthy, 1=healthy)
   */
  updateEconomicFlowHealth(component, health) {
    this.metrics.economic_flow_health[component] = health;
  }

  /**
   * Update funding sustainability estimate
   * @param {number} days - Estimated days of sustainable operations
   */
  updateFundingSustainability(days) {
    this.metrics.funding_sustainability_days = days;
  }

  /**
   * Get current metrics summary
   * @returns {Object} Current metrics summary
   */
  getSummary() {
    const uptime = Math.floor((Date.now() - this.startTime) / 1000);
    const totalRequests = Object.values(this.metrics.requests_total).reduce((a, b) => a + b, 0);
    const successRate = totalRequests > 0 ? 
      (this.metrics.requests_total.success / totalRequests * 100).toFixed(2) : 0;

    return {
      uptime_seconds: uptime,
      total_requests: totalRequests,
      success_rate_percent: parseFloat(successRate),
      current_balance_nrv: this.metrics.balance_nrv,
      active_requests: this.metrics.active_requests,
      economic_flow_health: this.metrics.economic_flow_health,
      funding_sustainability_days: this.metrics.funding_sustainability_days
    };
  }

  /**
   * Generate Prometheus-compatible metrics
   * @returns {string} Prometheus metrics format
   */
  getPrometheusMetrics() {
    const {
      requests_total, balance_nrv, request_duration_histogram, rate_limit_hits,
      router_proofs_generated, router_nrv_minting_requests, router_proof_generation_rate,
      treasury_balance_nrv, treasury_transfers, treasury_funding_rate,
      economic_flow_health, funding_sustainability_days
    } = this.metrics;

    return `
# HELP faucet_requests_total Total number of faucet requests
# TYPE faucet_requests_total counter
faucet_requests_total{status="success"} ${requests_total.success}
faucet_requests_total{status="rejected"} ${requests_total.rejected}
faucet_requests_total{status="failed"} ${requests_total.failed}

# HELP faucet_balance_nrv Current faucet balance in NRV
# TYPE faucet_balance_nrv gauge
faucet_balance_nrv ${balance_nrv}

# HELP faucet_request_duration_seconds Request processing time
# TYPE faucet_request_duration_seconds histogram
${Object.entries(request_duration_histogram.buckets).map(([le, count]) =>
  `faucet_request_duration_seconds_bucket{le="${le}"} ${count}`
).join('\n')}
faucet_request_duration_seconds_sum ${request_duration_histogram.sum}
faucet_request_duration_seconds_count ${request_duration_histogram.count}

# HELP faucet_rate_limit_hits_total Rate limit violations
# TYPE faucet_rate_limit_hits_total counter
faucet_rate_limit_hits_total{type="ip"} ${rate_limit_hits.ip}
faucet_rate_limit_hits_total{type="address"} ${rate_limit_hits.address}
faucet_rate_limit_hits_total{type="global"} ${rate_limit_hits.global}

# HELP faucet_active_requests Currently processing requests
# TYPE faucet_active_requests gauge
faucet_active_requests ${this.metrics.active_requests}

# HELP knirv_router_proofs_generated_total ROUTER connectivity proofs generated
# TYPE knirv_router_proofs_generated_total counter
knirv_router_proofs_generated_total ${router_proofs_generated}

# HELP knirv_router_nrv_minting_requests_total ROUTER NRV minting requests
# TYPE knirv_router_nrv_minting_requests_total counter
knirv_router_nrv_minting_requests_total{status="success"} ${router_nrv_minting_requests.success}
knirv_router_nrv_minting_requests_total{status="failed"} ${router_nrv_minting_requests.failed}

# HELP knirv_router_proof_generation_rate ROUTER proof generation rate per hour
# TYPE knirv_router_proof_generation_rate gauge
knirv_router_proof_generation_rate ${router_proof_generation_rate}

# HELP knirv_oracle_treasury_balance_nrv ORACLE treasury wallet balance
# TYPE knirv_oracle_treasury_balance_nrv gauge
knirv_oracle_treasury_balance_nrv ${treasury_balance_nrv}

# HELP knirv_oracle_treasury_transfers_total Treasury transfers
# TYPE knirv_oracle_treasury_transfers_total counter
knirv_oracle_treasury_transfers_total{destination="testnet_faucet"} ${treasury_transfers.to_faucet}
knirv_oracle_treasury_transfers_total{destination="all"} ${treasury_transfers.total}

# HELP knirv_oracle_treasury_funding_rate Treasury funding rate NRV per hour
# TYPE knirv_oracle_treasury_funding_rate gauge
knirv_oracle_treasury_funding_rate ${treasury_funding_rate}

# HELP knirv_economic_flow_health Economic flow component health (0=unhealthy, 1=healthy)
# TYPE knirv_economic_flow_health gauge
knirv_economic_flow_health{component="router"} ${economic_flow_health.router}
knirv_economic_flow_health{component="treasury"} ${economic_flow_health.treasury}
knirv_economic_flow_health{component="faucet"} ${economic_flow_health.faucet}

# HELP knirv_testnet_funding_sustainability_days Days of sustainable testnet operations
# TYPE knirv_testnet_funding_sustainability_days gauge
knirv_testnet_funding_sustainability_days ${funding_sustainability_days}
    `.trim();
  }

  /**
   * Reset all metrics (useful for testing)
   */
  reset() {
    this.metrics = {
      requests_total: { success: 0, rejected: 0, failed: 0 },
      balance_nrv: 0,
      request_duration_histogram: {
        buckets: { '0.5': 0, '1.0': 0, '2.0': 0, '5.0': 0, 'Inf': 0 },
        sum: 0,
        count: 0
      },
      rate_limit_hits: { ip: 0, address: 0, global: 0 },
      active_requests: 0,
      last_request_timestamp: 0,
      router_proofs_generated: 0,
      router_nrv_minting_requests: { success: 0, failed: 0 },
      router_proof_generation_rate: 0,
      treasury_balance_nrv: 0,
      treasury_transfers: { to_faucet: 0, total: 0 },
      treasury_funding_rate: 0,
      economic_flow_health: { router: 1, treasury: 1, faucet: 1 },
      funding_sustainability_days: 0
    };
    this.startTime = Date.now();
  }
}

module.exports = { FaucetMetrics };
