/**
 * Reward Calculator
 * Calculates rewards with performance-based multipliers
 */

class RewardCalculator {
  constructor() {
    // Base rewards for different activities
    this.baseRewards = {
      validation: BigInt(50000),
      skill_creation: BigInt(100000),
      bug_reporting: BigInt(25000),
      community_help: BigInt(10000)
    };

    // Performance multipliers
    this.multipliers = {
      high_performance: 1.5,
      consistent_user: 1.2,
      early_adopter: 1.3,
      community_leader: 2.0
    };

    // Performance data for users
    this.performanceData = new Map();
  }

  /**
   * Calculate reward with performance multipliers
   */
  calculateReward(userId, rewardType, baseAmount) {
    const baseAmountBigInt = typeof baseAmount === 'bigint' ? baseAmount : BigInt(baseAmount);
    let multiplier = 1.0;

    // Get user performance metrics
    const metrics = this.performanceData.get(userId);

    if (metrics) {
      // Apply high performance multiplier
      if (metrics.successRate > 0.9) {
        multiplier *= this.multipliers.high_performance;
      }

      // Apply consistent user multiplier
      if (metrics.uptime > 0.95) {
        multiplier *= this.multipliers.consistent_user;
      }

      // Apply community leader multiplier if user satisfaction is high
      if (metrics.userSatisfaction > 0.95) {
        multiplier *= this.multipliers.community_leader;
      }
    }

    // Calculate final reward
    const multipliedAmount = Number(baseAmountBigInt) * multiplier;
    return BigInt(Math.floor(multipliedAmount));
  }

  /**
   * Update performance metrics for a user
   */
  updatePerformanceMetrics(userId, metrics) {
    const performanceMetrics = {
      successRate: metrics.successRate || 0,
      responseTime: metrics.responseTime || 0,
      userSatisfaction: metrics.userSatisfaction || 0,
      uptime: metrics.uptime || 0,
      lastUpdated: new Date()
    };

    this.performanceData.set(userId, performanceMetrics);
    return performanceMetrics;
  }

  /**
   * Get performance metrics for a user
   */
  getPerformanceMetrics(userId) {
    return this.performanceData.get(userId) || null;
  }

  /**
   * Get base reward for a reward type
   */
  getBaseReward(rewardType) {
    return this.baseRewards[rewardType] || BigInt(0);
  }

  /**
   * Set base reward for a reward type
   */
  setBaseReward(rewardType, amount) {
    this.baseRewards[rewardType] = BigInt(amount);
  }

  /**
   * Get all multipliers
   */
  getMultipliers() {
    return { ...this.multipliers };
  }

  /**
   * Set multiplier
   */
  setMultiplier(name, value) {
    this.multipliers[name] = value;
  }

  /**
   * Calculate estimated reward for user
   */
  estimateReward(userId, rewardType) {
    const baseReward = this.baseRewards[rewardType];
    if (!baseReward) {
      return BigInt(0);
    }

    return this.calculateReward(userId, rewardType, baseReward);
  }

  /**
   * Get leaderboard based on performance metrics
   */
  getLeaderboard(limit = 10, metric = 'successRate') {
    const users = Array.from(this.performanceData.entries())
      .map(([userId, metrics]) => ({
        userId,
        ...metrics
      }))
      .sort((a, b) => b[metric] - a[metric])
      .slice(0, limit);

    return users;
  }

  /**
   * Export reward calculator state
   */
  toJSON() {
    return {
      baseRewards: Object.fromEntries(
        Object.entries(this.baseRewards).map(([k, v]) => [k, v.toString()])
      ),
      multipliers: this.multipliers,
      performanceData: Object.fromEntries(
        Array.from(this.performanceData.entries()).map(([userId, metrics]) => [
          userId,
          {
            ...metrics,
            lastUpdated: metrics.lastUpdated.toISOString()
          }
        ])
      )
    };
  }

  /**
   * Import reward calculator state
   */
  static fromJSON(json) {
    const calculator = new RewardCalculator();

    calculator.baseRewards = Object.fromEntries(
      Object.entries(json.baseRewards).map(([k, v]) => [k, BigInt(v)])
    );

    calculator.multipliers = json.multipliers;

    calculator.performanceData = new Map(
      Object.entries(json.performanceData).map(([userId, metrics]) => [
        userId,
        {
          ...metrics,
          lastUpdated: new Date(metrics.lastUpdated)
        }
      ])
    );

    return calculator;
  }
}

module.exports = RewardCalculator;
