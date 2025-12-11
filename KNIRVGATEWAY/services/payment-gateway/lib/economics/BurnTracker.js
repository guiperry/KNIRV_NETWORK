/**
 * Burn Tracker
 * Tracks all token burn events and total burned amount
 */

class BurnTracker {
  constructor() {
    this.totalBurned = BigInt(0);
    this.burnHistory = [];
    this.burnRates = new Map();
    this.maxHistorySize = 10000;
  }

  /**
   * Add a burn event
   */
  addBurnEvent(event) {
    const burnEvent = {
      txId: event.txId,
      user: event.user,
      amount: BigInt(event.amount),
      purpose: event.purpose,
      skillId: event.skillId || null,
      timestamp: event.timestamp || new Date(),
      validated: event.validated || false
    };

    this.burnHistory.push(burnEvent);

    // Maintain history size
    if (this.burnHistory.length > this.maxHistorySize) {
      this.burnHistory = this.burnHistory.slice(-this.maxHistorySize);
    }

    return burnEvent;
  }

  /**
   * Process and validate burn event
   */
  processBurn(txId, amount) {
    const amountBigInt = BigInt(amount);
    this.totalBurned += amountBigInt;

    // Find and validate the burn event
    const event = this.burnHistory.find(e => e.txId === txId);
    if (event) {
      event.validated = true;
    }

    return this.totalBurned;
  }

  /**
   * Get total burned tokens
   */
  getTotalBurned() {
    return this.totalBurned;
  }

  /**
   * Get burn history with optional filters
   */
  getBurnHistory(options = {}) {
    let history = [...this.burnHistory];

    // Filter by user
    if (options.user) {
      history = history.filter(e => e.user === options.user);
    }

    // Filter by purpose
    if (options.purpose) {
      history = history.filter(e => e.purpose === options.purpose);
    }

    // Filter by validated status
    if (options.validated !== undefined) {
      history = history.filter(e => e.validated === options.validated);
    }

    // Filter by time range
    if (options.startTime) {
      const startTime = new Date(options.startTime);
      history = history.filter(e => e.timestamp >= startTime);
    }

    if (options.endTime) {
      const endTime = new Date(options.endTime);
      history = history.filter(e => e.timestamp <= endTime);
    }

    // Apply limit
    const limit = options.limit || 100;
    return history.slice(-limit);
  }

  /**
   * Get burn statistics by purpose
   */
  getBurnStatsByPurpose() {
    const stats = {};

    for (const event of this.burnHistory) {
      if (!stats[event.purpose]) {
        stats[event.purpose] = {
          count: 0,
          totalAmount: BigInt(0),
          validated: 0
        };
      }

      stats[event.purpose].count++;
      stats[event.purpose].totalAmount += event.amount;
      if (event.validated) {
        stats[event.purpose].validated++;
      }
    }

    // Convert BigInt to string for JSON serialization
    return Object.fromEntries(
      Object.entries(stats).map(([purpose, data]) => [
        purpose,
        {
          ...data,
          totalAmount: data.totalAmount.toString()
        }
      ])
    );
  }

  /**
   * Get burn rate for a specific purpose
   */
  getBurnRate(purpose) {
    return this.burnRates.get(purpose) || BigInt(0);
  }

  /**
   * Set burn rate for a purpose
   */
  setBurnRate(purpose, rate) {
    this.burnRates.set(purpose, BigInt(rate));
  }

  /**
   * Export burn tracker state
   */
  toJSON() {
    return {
      totalBurned: this.totalBurned.toString(),
      burnHistory: this.burnHistory.map(event => ({
        ...event,
        amount: event.amount.toString(),
        timestamp: event.timestamp.toISOString()
      })),
      burnRates: Object.fromEntries(
        Array.from(this.burnRates.entries()).map(([k, v]) => [k, v.toString()])
      )
    };
  }

  /**
   * Import burn tracker state
   */
  static fromJSON(json) {
    const tracker = new BurnTracker();
    tracker.totalBurned = BigInt(json.totalBurned);
    tracker.burnHistory = json.burnHistory.map(event => ({
      ...event,
      amount: BigInt(event.amount),
      timestamp: new Date(event.timestamp)
    }));
    tracker.burnRates = new Map(
      Object.entries(json.burnRates).map(([k, v]) => [k, BigInt(v)])
    );
    return tracker;
  }
}

module.exports = BurnTracker;
