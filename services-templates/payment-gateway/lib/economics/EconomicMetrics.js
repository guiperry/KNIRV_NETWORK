/**
 * Economic Metrics
 * Tracks and calculates network economic metrics
 */

class EconomicMetrics {
  constructor() {
    this.totalSupply = BigInt(0);
    this.circulatingSupply = BigInt(0);
    this.totalBurned = BigInt(0);
    this.totalStaked = BigInt(0);
    this.activeValidators = 0;
    this.transactionVolume = BigInt(0);
    this.averageGasPrice = BigInt(1000);
    this.networkUtilization = 0.75; // 75%
    this.tokenVelocity = 2.5; // 2.5x
    this.lastUpdated = new Date();

    // Service-specific metrics
    this.serviceMetrics = {
      knirvchain: this.createServiceMetrics(),
      knirvnexus: this.createServiceMetrics(),
      knirvoracle: this.createServiceMetrics(),
      knirvgraph: this.createServiceMetrics()
    };
  }

  /**
   * Create empty service metrics object
   */
  createServiceMetrics() {
    return {
      revenue: BigInt(0),
      costs: BigInt(0),
      profit: BigInt(0),
      tokensEarned: BigInt(0),
      tokensSpent: BigInt(0),
      userCount: 0,
      transactionCount: 0,
      lastUpdated: new Date()
    };
  }

  /**
   * Update service metrics
   */
  updateServiceMetrics(serviceName, amount, operation) {
    if (!this.serviceMetrics[serviceName]) {
      this.serviceMetrics[serviceName] = this.createServiceMetrics();
    }

    const metrics = this.serviceMetrics[serviceName];
    const amountBigInt = BigInt(amount);

    switch (operation) {
      case 'earned':
        metrics.revenue += amountBigInt;
        metrics.tokensEarned += amountBigInt;
        break;
      case 'spent':
        metrics.costs += amountBigInt;
        metrics.tokensSpent += amountBigInt;
        break;
    }

    // Update profit
    metrics.profit = metrics.revenue - metrics.costs;
    metrics.transactionCount++;
    metrics.lastUpdated = new Date();
  }

  /**
   * Update total supply
   */
  updateTotalSupply(amount, operation = 'add') {
    const amountBigInt = BigInt(amount);
    if (operation === 'add') {
      this.totalSupply += amountBigInt;
      this.circulatingSupply += amountBigInt;
    } else {
      this.totalSupply -= amountBigInt;
      this.circulatingSupply -= amountBigInt;
    }
    this.lastUpdated = new Date();
  }

  /**
   * Update burned tokens
   */
  updateBurned(amount) {
    const amountBigInt = BigInt(amount);
    this.totalBurned += amountBigInt;
    this.circulatingSupply -= amountBigInt;
    this.lastUpdated = new Date();
  }

  /**
   * Update staked tokens
   */
  updateStaked(amount, operation = 'add') {
    const amountBigInt = BigInt(amount);
    if (operation === 'add') {
      this.totalStaked += amountBigInt;
    } else {
      this.totalStaked -= amountBigInt;
    }
    this.lastUpdated = new Date();
  }

  /**
   * Update transaction volume
   */
  updateTransactionVolume(amount) {
    const amountBigInt = BigInt(amount);
    this.transactionVolume += amountBigInt;
    this.lastUpdated = new Date();
  }

  /**
   * Calculate network utilization
   */
  calculateNetworkUtilization() {
    // This is a simplified calculation
    // In a real implementation, this would be based on actual network metrics
    return this.networkUtilization;
  }

  /**
   * Calculate token velocity
   */
  calculateTokenVelocity() {
    // Token velocity = Transaction Volume / Circulating Supply
    if (this.circulatingSupply === BigInt(0)) {
      return 0;
    }

    // Simplified calculation
    return this.tokenVelocity;
  }

  /**
   * Calculate average gas price
   */
  calculateAverageGasPrice() {
    // Calculate from recent transactions
    return this.averageGasPrice;
  }

  /**
   * Update all calculated metrics
   */
  updateCalculatedMetrics() {
    this.networkUtilization = this.calculateNetworkUtilization();
    this.tokenVelocity = this.calculateTokenVelocity();
    this.averageGasPrice = this.calculateAverageGasPrice();
    this.lastUpdated = new Date();
  }

  /**
   * Get service metrics for a specific service
   */
  getServiceMetrics(serviceName) {
    return this.serviceMetrics[serviceName] || null;
  }

  /**
   * Get all metrics
   */
  getAllMetrics() {
    return {
      totalSupply: this.totalSupply.toString(),
      circulatingSupply: this.circulatingSupply.toString(),
      totalBurned: this.totalBurned.toString(),
      totalStaked: this.totalStaked.toString(),
      activeValidators: this.activeValidators,
      transactionVolume: this.transactionVolume.toString(),
      averageGasPrice: this.averageGasPrice.toString(),
      networkUtilization: this.networkUtilization,
      tokenVelocity: this.tokenVelocity,
      lastUpdated: this.lastUpdated.toISOString(),
      serviceMetrics: this.getServiceMetricsJSON()
    };
  }

  /**
   * Get service metrics in JSON format
   */
  getServiceMetricsJSON() {
    const result = {};
    for (const [service, metrics] of Object.entries(this.serviceMetrics)) {
      result[service] = {
        revenue: metrics.revenue.toString(),
        costs: metrics.costs.toString(),
        profit: metrics.profit.toString(),
        tokensEarned: metrics.tokensEarned.toString(),
        tokensSpent: metrics.tokensSpent.toString(),
        userCount: metrics.userCount,
        transactionCount: metrics.transactionCount,
        lastUpdated: metrics.lastUpdated.toISOString()
      };
    }
    return result;
  }

  /**
   * Export metrics state
   */
  toJSON() {
    return this.getAllMetrics();
  }

  /**
   * Import metrics state
   */
  static fromJSON(json) {
    const metrics = new EconomicMetrics();

    metrics.totalSupply = BigInt(json.totalSupply);
    metrics.circulatingSupply = BigInt(json.circulatingSupply);
    metrics.totalBurned = BigInt(json.totalBurned);
    metrics.totalStaked = BigInt(json.totalStaked);
    metrics.activeValidators = json.activeValidators;
    metrics.transactionVolume = BigInt(json.transactionVolume);
    metrics.averageGasPrice = BigInt(json.averageGasPrice);
    metrics.networkUtilization = json.networkUtilization;
    metrics.tokenVelocity = json.tokenVelocity;
    metrics.lastUpdated = new Date(json.lastUpdated);

    // Import service metrics
    for (const [service, serviceMetrics] of Object.entries(json.serviceMetrics)) {
      metrics.serviceMetrics[service] = {
        revenue: BigInt(serviceMetrics.revenue),
        costs: BigInt(serviceMetrics.costs),
        profit: BigInt(serviceMetrics.profit),
        tokensEarned: BigInt(serviceMetrics.tokensEarned),
        tokensSpent: BigInt(serviceMetrics.tokensSpent),
        userCount: serviceMetrics.userCount,
        transactionCount: serviceMetrics.transactionCount,
        lastUpdated: new Date(serviceMetrics.lastUpdated)
      };
    }

    return metrics;
  }
}

module.exports = EconomicMetrics;
