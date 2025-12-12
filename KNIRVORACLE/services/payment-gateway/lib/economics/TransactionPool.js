/**
 * Transaction Pool
 * Manages pending and confirmed economic transactions
 */

class TransactionPool {
  constructor(config = {}) {
    this.pendingTxs = new Map();
    this.confirmedTxs = new Map();
    this.maxPoolSize = config.maxPoolSize || 10000;
    this.cleanupInterval = config.cleanupInterval || 60 * 60 * 1000; // 1 hour
    this.cleanupTimer = null;
  }

  /**
   * Add a transaction to the pending pool
   */
  addTransaction(tx) {
    this.pendingTxs.set(tx.id, tx);

    // Clean up if pool is too large
    if (this.pendingTxs.size > this.maxPoolSize) {
      this.cleanupOldTransactions();
    }
  }

  /**
   * Move transaction from pending to confirmed
   */
  confirmTransaction(txId) {
    const tx = this.pendingTxs.get(txId);
    if (tx) {
      tx.status = 'confirmed';
      tx.confirmedAt = new Date();
      this.pendingTxs.delete(txId);
      this.confirmedTxs.set(txId, tx);
    }
    return tx;
  }

  /**
   * Mark transaction as failed
   */
  failTransaction(txId, error) {
    const tx = this.pendingTxs.get(txId);
    if (tx) {
      tx.status = 'failed';
      tx.error = error;
      this.pendingTxs.delete(txId);
      this.confirmedTxs.set(txId, tx);
    }
    return tx;
  }

  /**
   * Get transaction by ID
   */
  getTransaction(txId) {
    return this.confirmedTxs.get(txId) || this.pendingTxs.get(txId);
  }

  /**
   * Get all pending transactions
   */
  getPendingTransactions() {
    return Array.from(this.pendingTxs.values());
  }

  /**
   * Get all confirmed transactions
   */
  getConfirmedTransactions(limit = 100) {
    const txs = Array.from(this.confirmedTxs.values());
    return txs.slice(-limit);
  }

  /**
   * Get transactions by status
   */
  getTransactionsByStatus(status, limit = 100) {
    const pool = status === 'pending' ? this.pendingTxs : this.confirmedTxs;
    const txs = Array.from(pool.values()).filter(tx => tx.status === status);
    return txs.slice(-limit);
  }

  /**
   * Clean up old transactions
   */
  cleanupOldTransactions() {
    const cutoff = Date.now() - this.cleanupInterval;

    // Clean pending transactions
    for (const [id, tx] of this.pendingTxs.entries()) {
      if (tx.timestamp.getTime() < cutoff) {
        this.pendingTxs.delete(id);
      }
    }

    // Keep only recent confirmed transactions (last 10000)
    if (this.confirmedTxs.size > 10000) {
      const entries = Array.from(this.confirmedTxs.entries());
      entries.sort((a, b) => b[1].timestamp - a[1].timestamp);
      this.confirmedTxs = new Map(entries.slice(0, 10000));
    }
  }

  /**
   * Start automatic cleanup timer
   */
  startCleanup() {
    if (this.cleanupTimer) {
      clearInterval(this.cleanupTimer);
    }
    this.cleanupTimer = setInterval(() => {
      this.cleanupOldTransactions();
    }, this.cleanupInterval);
  }

  /**
   * Stop automatic cleanup
   */
  stopCleanup() {
    if (this.cleanupTimer) {
      clearInterval(this.cleanupTimer);
      this.cleanupTimer = null;
    }
  }

  /**
   * Get pool statistics
   */
  getStats() {
    return {
      pendingCount: this.pendingTxs.size,
      confirmedCount: this.confirmedTxs.size,
      totalCount: this.pendingTxs.size + this.confirmedTxs.size,
      maxPoolSize: this.maxPoolSize
    };
  }
}

module.exports = TransactionPool;
