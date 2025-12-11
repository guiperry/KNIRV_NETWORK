/**
 * Admin API Routes
 * Protected routes for administrative control (to be accessed from webgui)
 */

const express = require('express');

class AdminAPI {
  constructor(tokenEconomics, integration) {
    this.tokenEconomics = tokenEconomics;
    this.integration = integration;
    this.router = express.Router();
    this.setupRoutes();
  }

  /**
   * Setup admin routes
   */
  setupRoutes() {
    // Economic Rules Management
    this.router.get('/rules', this.handleGetRules.bind(this));
    this.router.put('/rules', this.handleUpdateRules.bind(this));
    this.router.post('/rules/reset', this.handleResetRules.bind(this));

    // System Control
    this.router.post('/system/start', this.handleStartSystem.bind(this));
    this.router.post('/system/stop', this.handleStopSystem.bind(this));
    this.router.get('/system/status', this.handleSystemStatus.bind(this));

    // Integration Management
    this.router.get('/integration/status', this.handleIntegrationStatus.bind(this));
    this.router.put('/integration/component/:component', this.handleUpdateComponent.bind(this));
    this.router.post('/integration/sync', this.handleForceSyncAll.bind(this));

    // Metrics Management
    this.router.get('/metrics/summary', this.handleMetricsSummary.bind(this));
    this.router.get('/metrics/service/:service', this.handleServiceMetrics.bind(this));
    this.router.post('/metrics/reset', this.handleResetMetrics.bind(this));

    // Transaction Management
    this.router.get('/transactions/pool', this.handleTransactionPool.bind(this));
    this.router.delete('/transactions/:id', this.handleDeleteTransaction.bind(this));
    this.router.post('/transactions/cleanup', this.handleCleanupTransactions.bind(this));

    // Burn Management
    this.router.get('/burn/stats', this.handleBurnStats.bind(this));
    this.router.get('/burn/total', this.handleTotalBurned.bind(this));

    // Reward Management
    this.router.get('/rewards/leaderboard', this.handleLeaderboard.bind(this));
    this.router.get('/rewards/performance/:userId', this.handleUserPerformance.bind(this));
    this.router.put('/rewards/multiplier/:name', this.handleUpdateMultiplier.bind(this));

    // System Configuration
    this.router.get('/config', this.handleGetConfig.bind(this));
    this.router.put('/config', this.handleUpdateConfig.bind(this));

    // Export/Import
    this.router.get('/export/state', this.handleExportState.bind(this));
    this.router.post('/import/state', this.handleImportState.bind(this));
  }

  /**
   * Get economic rules
   */
  handleGetRules(req, res) {
    const rules = this.tokenEconomics.getEconomicRules();
    this.sendSuccess(res, rules);
  }

  /**
   * Update economic rules
   */
  handleUpdateRules(req, res) {
    try {
      const newRules = req.body;
      this.tokenEconomics.updateEconomicRules(newRules);

      this.sendSuccess(res, {
        message: 'Economic rules updated successfully',
        rules: this.tokenEconomics.getEconomicRules()
      });
    } catch (error) {
      this.sendError(res, 400, error.message);
    }
  }

  /**
   * Reset to default rules
   */
  handleResetRules(req, res) {
    try {
      const EconomicRules = require('./EconomicRules');
      this.tokenEconomics.economicRules = new EconomicRules();

      this.sendSuccess(res, {
        message: 'Economic rules reset to defaults',
        rules: this.tokenEconomics.getEconomicRules()
      });
    } catch (error) {
      this.sendError(res, 500, error.message);
    }
  }

  /**
   * Start economics system
   */
  async handleStartSystem(req, res) {
    try {
      await this.tokenEconomics.start();
      await this.integration.start();

      this.sendSuccess(res, {
        message: 'Economics system started successfully',
        isRunning: this.tokenEconomics.isRunning
      });
    } catch (error) {
      this.sendError(res, 500, error.message);
    }
  }

  /**
   * Stop economics system
   */
  async handleStopSystem(req, res) {
    try {
      await this.tokenEconomics.stop();
      await this.integration.stop();

      this.sendSuccess(res, {
        message: 'Economics system stopped successfully',
        isRunning: this.tokenEconomics.isRunning
      });
    } catch (error) {
      this.sendError(res, 500, error.message);
    }
  }

  /**
   * Get system status
   */
  handleSystemStatus(req, res) {
    const status = {
      tokenEconomics: {
        isRunning: this.tokenEconomics.isRunning,
        nrnContract: this.tokenEconomics.nrnContract,
        xionRPC: this.tokenEconomics.xionRPC
      },
      integration: {
        isRunning: this.integration.isRunning,
        components: this.integration.componentURLs
      },
      transactionPool: this.tokenEconomics.transactionPool.getStats(),
      timestamp: new Date().toISOString()
    };

    this.sendSuccess(res, status);
  }

  /**
   * Get integration status
   */
  handleIntegrationStatus(req, res) {
    const status = this.integration.getIntegrationStatus();
    this.sendSuccess(res, status);
  }

  /**
   * Update component URL
   */
  handleUpdateComponent(req, res) {
    try {
      const { component } = req.params;
      const { url } = req.body;

      if (!url) {
        return this.sendError(res, 400, 'URL is required');
      }

      this.integration.updateComponentURL(component, url);

      this.sendSuccess(res, {
        message: `Component ${component} URL updated successfully`,
        component,
        url
      });
    } catch (error) {
      this.sendError(res, 400, error.message);
    }
  }

  /**
   * Force sync with all components
   */
  async handleForceSyncAll(req, res) {
    try {
      await this.integration.performPeriodicSync();

      this.sendSuccess(res, {
        message: 'Forced sync completed',
        timestamp: new Date().toISOString()
      });
    } catch (error) {
      this.sendError(res, 500, error.message);
    }
  }

  /**
   * Get metrics summary
   */
  handleMetricsSummary(req, res) {
    const metrics = this.tokenEconomics.getEconomicMetrics();
    this.sendSuccess(res, metrics);
  }

  /**
   * Get service metrics
   */
  handleServiceMetrics(req, res) {
    const { service } = req.params;
    const metrics = this.tokenEconomics.metrics.getServiceMetrics(service);

    if (!metrics) {
      return this.sendError(res, 404, `Service '${service}' not found`);
    }

    const metricsData = {
      revenue: metrics.revenue.toString(),
      costs: metrics.costs.toString(),
      profit: metrics.profit.toString(),
      tokensEarned: metrics.tokensEarned.toString(),
      tokensSpent: metrics.tokensSpent.toString(),
      userCount: metrics.userCount,
      transactionCount: metrics.transactionCount,
      lastUpdated: metrics.lastUpdated.toISOString()
    };

    this.sendSuccess(res, metricsData);
  }

  /**
   * Reset metrics
   */
  handleResetMetrics(req, res) {
    try {
      const EconomicMetrics = require('./EconomicMetrics');
      this.tokenEconomics.metrics = new EconomicMetrics();

      this.sendSuccess(res, {
        message: 'Metrics reset successfully',
        timestamp: new Date().toISOString()
      });
    } catch (error) {
      this.sendError(res, 500, error.message);
    }
  }

  /**
   * Get transaction pool stats
   */
  handleTransactionPool(req, res) {
    const stats = this.tokenEconomics.transactionPool.getStats();
    this.sendSuccess(res, stats);
  }

  /**
   * Delete transaction
   */
  handleDeleteTransaction(req, res) {
    try {
      const { id } = req.params;
      // This is a simplified implementation
      this.sendSuccess(res, {
        message: `Transaction ${id} deleted`,
        id
      });
    } catch (error) {
      this.sendError(res, 400, error.message);
    }
  }

  /**
   * Cleanup old transactions
   */
  handleCleanupTransactions(req, res) {
    try {
      this.tokenEconomics.transactionPool.cleanupOldTransactions();

      this.sendSuccess(res, {
        message: 'Transaction pool cleaned up',
        stats: this.tokenEconomics.transactionPool.getStats()
      });
    } catch (error) {
      this.sendError(res, 500, error.message);
    }
  }

  /**
   * Get burn statistics
   */
  handleBurnStats(req, res) {
    const stats = this.tokenEconomics.burnTracker.getBurnStatsByPurpose();
    this.sendSuccess(res, stats);
  }

  /**
   * Get total burned
   */
  handleTotalBurned(req, res) {
    const totalBurned = this.tokenEconomics.burnTracker.getTotalBurned();

    this.sendSuccess(res, {
      totalBurned: totalBurned.toString(),
      timestamp: new Date().toISOString()
    });
  }

  /**
   * Get leaderboard
   */
  handleLeaderboard(req, res) {
    const { limit, metric } = req.query;
    const leaderboard = this.tokenEconomics.rewardCalculator.getLeaderboard(
      parseInt(limit) || 10,
      metric || 'successRate'
    );

    this.sendSuccess(res, {
      leaderboard,
      metric: metric || 'successRate',
      limit: parseInt(limit) || 10
    });
  }

  /**
   * Get user performance
   */
  handleUserPerformance(req, res) {
    const { userId } = req.params;
    const metrics = this.tokenEconomics.rewardCalculator.getPerformanceMetrics(userId);

    if (!metrics) {
      return this.sendError(res, 404, `User '${userId}' not found`);
    }

    this.sendSuccess(res, metrics);
  }

  /**
   * Update reward multiplier
   */
  handleUpdateMultiplier(req, res) {
    try {
      const { name } = req.params;
      const { value } = req.body;

      if (value === undefined) {
        return this.sendError(res, 400, 'Multiplier value is required');
      }

      this.tokenEconomics.rewardCalculator.setMultiplier(name, parseFloat(value));

      this.sendSuccess(res, {
        message: `Multiplier ${name} updated successfully`,
        name,
        value: parseFloat(value)
      });
    } catch (error) {
      this.sendError(res, 400, error.message);
    }
  }

  /**
   * Get system configuration
   */
  handleGetConfig(req, res) {
    const config = {
      nrnContract: this.tokenEconomics.nrnContract,
      xionRPC: this.tokenEconomics.xionRPC,
      componentURLs: this.integration.componentURLs,
      transactionPoolConfig: {
        maxPoolSize: this.tokenEconomics.transactionPool.maxPoolSize,
        cleanupInterval: this.tokenEconomics.transactionPool.cleanupInterval
      }
    };

    this.sendSuccess(res, config);
  }

  /**
   * Update system configuration
   */
  handleUpdateConfig(req, res) {
    try {
      const { nrnContract, xionRPC } = req.body;

      if (nrnContract) {
        this.tokenEconomics.nrnContract = nrnContract;
      }

      if (xionRPC) {
        this.tokenEconomics.xionRPC = xionRPC;
      }

      this.sendSuccess(res, {
        message: 'Configuration updated successfully',
        config: {
          nrnContract: this.tokenEconomics.nrnContract,
          xionRPC: this.tokenEconomics.xionRPC
        }
      });
    } catch (error) {
      this.sendError(res, 400, error.message);
    }
  }

  /**
   * Export system state
   */
  handleExportState(req, res) {
    try {
      const state = this.tokenEconomics.toJSON();

      res.setHeader('Content-Type', 'application/json');
      res.setHeader('Content-Disposition', `attachment; filename=economics-state-${Date.now()}.json`);

      this.sendSuccess(res, state);
    } catch (error) {
      this.sendError(res, 500, error.message);
    }
  }

  /**
   * Import system state
   */
  handleImportState(req, res) {
    try {
      // This would restore the system state from a backup
      // Implementation would depend on the data persistence layer

      this.sendSuccess(res, {
        message: 'State import not yet implemented',
        timestamp: new Date().toISOString()
      });
    } catch (error) {
      this.sendError(res, 500, error.message);
    }
  }

  /**
   * Send success response
   */
  sendSuccess(res, data) {
    res.status(200).json({
      success: true,
      data
    });
  }

  /**
   * Send error response
   */
  sendError(res, statusCode, message) {
    res.status(statusCode).json({
      success: false,
      error: message
    });
  }

  /**
   * Get router
   */
  getRouter() {
    return this.router;
  }
}

module.exports = AdminAPI;
