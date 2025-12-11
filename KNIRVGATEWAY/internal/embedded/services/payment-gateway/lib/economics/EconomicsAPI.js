/**
 * Economics API Routes
 * Provides REST API for economics functionality
 */

const express = require('express');

class EconomicsAPI {
  constructor(tokenEconomics, integration) {
    this.tokenEconomics = tokenEconomics;
    this.integration = integration;
    this.router = express.Router();
    this.setupRoutes();
  }

  /**
   * Setup all API routes
   */
  setupRoutes() {
    // Economic operations
    this.router.post('/skill/invoke', this.handleSkillInvocation.bind(this));
    this.router.post('/llm/register', this.handleLLMRegistration.bind(this));
    this.router.post('/validation/reward', this.handleValidationReward.bind(this));
    this.router.post('/fees/calculate', this.handleCalculateNetworkFees.bind(this));

    // Metrics and data retrieval
    this.router.get('/metrics', this.handleGetMetrics.bind(this));
    this.router.get('/transaction/:id', this.handleGetTransaction.bind(this));
    this.router.get('/transactions', this.handleGetTransactions.bind(this));
    this.router.get('/burn/history', this.handleGetBurnHistory.bind(this));
    this.router.get('/burn/total', this.handleGetTotalBurned.bind(this));

    // Economic rules and configuration
    this.router.get('/rules', this.handleGetEconomicRules.bind(this));
    this.router.put('/rules', this.handleUpdateEconomicRules.bind(this));

    // Service-specific metrics
    this.router.get('/service/:service/metrics', this.handleGetServiceMetrics.bind(this));

    // Integration status
    this.router.get('/integration/status', this.handleIntegrationStatus.bind(this));

    // Health check
    this.router.get('/health', this.handleHealthCheck.bind(this));

    // System info
    this.router.get('/info', this.handleSystemInfo.bind(this));
  }

  /**
   * Handle skill invocation
   */
  async handleSkillInvocation(req, res) {
    try {
      const { userId, skillId, amount } = req.body;

      if (!userId || !skillId || !amount) {
        return this.sendError(res, 400, 'Missing required fields: userId, skillId, amount');
      }

      const tx = await this.tokenEconomics.processSkillInvocation(userId, skillId, amount);

      this.sendSuccess(res, {
        transactionId: tx.id,
        status: tx.status,
        amount: tx.amount.toString(),
        timestamp: tx.timestamp
      });
    } catch (error) {
      this.sendError(res, 400, error.message);
    }
  }

  /**
   * Handle LLM registration
   */
  async handleLLMRegistration(req, res) {
    try {
      const { userId, llmId, registrationFee } = req.body;

      if (!userId || !llmId || !registrationFee) {
        return this.sendError(res, 400, 'Missing required fields: userId, llmId, registrationFee');
      }

      const tx = await this.tokenEconomics.processLLMRegistration(userId, llmId, registrationFee);

      this.sendSuccess(res, {
        transactionId: tx.id,
        status: tx.status,
        fee: tx.amount.toString(),
        timestamp: tx.timestamp
      });
    } catch (error) {
      this.sendError(res, 400, error.message);
    }
  }

  /**
   * Handle validation reward
   */
  async handleValidationReward(req, res) {
    try {
      const { validatorId, targetId, validationResult } = req.body;

      if (!validatorId || !targetId || validationResult === undefined) {
        return this.sendError(res, 400, 'Missing required fields: validatorId, targetId, validationResult');
      }

      const tx = await this.tokenEconomics.processValidationReward(
        validatorId,
        targetId,
        validationResult
      );

      this.sendSuccess(res, {
        transactionId: tx.id,
        status: tx.status,
        reward: tx.amount.toString(),
        timestamp: tx.timestamp
      });
    } catch (error) {
      this.sendError(res, 400, error.message);
    }
  }

  /**
   * Handle network fees calculation
   */
  async handleCalculateNetworkFees(req, res) {
    try {
      const { gasUsed, priority } = req.body;

      if (!gasUsed) {
        return this.sendError(res, 400, 'Missing required field: gasUsed');
      }

      const totalFee = this.tokenEconomics.calculateNetworkFees(gasUsed, priority || 'medium');

      this.sendSuccess(res, {
        gasUsed: gasUsed,
        priority: priority || 'medium',
        totalFee: totalFee.toString(),
        gasPrice: (totalFee / BigInt(gasUsed)).toString()
      });
    } catch (error) {
      this.sendError(res, 400, error.message);
    }
  }

  /**
   * Get economic metrics
   */
  handleGetMetrics(req, res) {
    const metrics = this.tokenEconomics.getEconomicMetrics();
    this.sendSuccess(res, metrics);
  }

  /**
   * Get transaction by ID
   */
  handleGetTransaction(req, res) {
    const { id } = req.params;
    const tx = this.tokenEconomics.transactionPool.getTransaction(id);

    if (!tx) {
      return this.sendError(res, 404, 'Transaction not found');
    }

    // Convert BigInt to string for JSON
    const txData = {
      ...tx,
      amount: tx.amount.toString()
    };

    this.sendSuccess(res, txData);
  }

  /**
   * Get transactions list
   */
  handleGetTransactions(req, res) {
    const { limit, status } = req.query;
    const parsedLimit = parseInt(limit) || 100;

    let transactions;
    if (status) {
      transactions = this.tokenEconomics.transactionPool.getTransactionsByStatus(
        status,
        parsedLimit
      );
    } else {
      const pending = this.tokenEconomics.transactionPool.getPendingTransactions();
      const confirmed = this.tokenEconomics.transactionPool.getConfirmedTransactions(parsedLimit);
      transactions = [...pending, ...confirmed].slice(-parsedLimit);
    }

    // Convert BigInt to string
    const txData = transactions.map(tx => ({
      ...tx,
      amount: tx.amount.toString()
    }));

    this.sendSuccess(res, {
      transactions: txData,
      count: txData.length,
      limit: parsedLimit,
      statusFilter: status || 'all'
    });
  }

  /**
   * Get burn history
   */
  handleGetBurnHistory(req, res) {
    const { limit, user, purpose } = req.query;
    const options = {
      limit: parseInt(limit) || 100,
      user,
      purpose
    };

    const burnHistory = this.tokenEconomics.burnTracker.getBurnHistory(options);

    // Convert BigInt to string
    const historyData = burnHistory.map(event => ({
      ...event,
      amount: event.amount.toString()
    }));

    this.sendSuccess(res, {
      burnEvents: historyData,
      count: historyData.length,
      limit: options.limit
    });
  }

  /**
   * Get total burned
   */
  handleGetTotalBurned(req, res) {
    const totalBurned = this.tokenEconomics.burnTracker.getTotalBurned();

    this.sendSuccess(res, {
      totalBurned: totalBurned.toString(),
      timestamp: new Date().toISOString()
    });
  }

  /**
   * Get economic rules
   */
  handleGetEconomicRules(req, res) {
    const rules = this.tokenEconomics.getEconomicRules();
    this.sendSuccess(res, rules);
  }

  /**
   * Update economic rules (admin only - should be protected)
   */
  handleUpdateEconomicRules(req, res) {
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
   * Get service metrics
   */
  handleGetServiceMetrics(req, res) {
    const { service } = req.params;
    const metrics = this.tokenEconomics.metrics.getServiceMetrics(service);

    if (!metrics) {
      return this.sendError(res, 404, `Service '${service}' not found`);
    }

    // Convert BigInt to string
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
   * Get integration status
   */
  handleIntegrationStatus(req, res) {
    const status = this.integration.getIntegrationStatus();
    this.sendSuccess(res, status);
  }

  /**
   * Health check
   */
  handleHealthCheck(req, res) {
    this.sendSuccess(res, {
      status: 'healthy',
      timestamp: new Date().toISOString(),
      version: '1.0.0',
      service: 'KNIRV Economics Engine'
    });
  }

  /**
   * System info
   */
  handleSystemInfo(req, res) {
    const state = this.tokenEconomics.toJSON();

    this.sendSuccess(res, {
      service: 'KNIRV Economics Service',
      version: '1.0.0',
      isRunning: this.tokenEconomics.isRunning,
      nrnContract: state.nrnContract,
      xionRPC: state.xionRPC,
      timestamp: state.timestamp
    });
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

module.exports = EconomicsAPI;
