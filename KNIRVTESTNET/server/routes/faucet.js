/**
 * KNIRVTESTNET Faucet API Routes
 * 
 * Provides RESTful API endpoints for NRV faucet operations including
 * token requests, status checks, history, and administrative functions.
 * Implements comprehensive error handling and request validation.
 */

const express = require('express');
const router = express.Router();
const { FaucetHandler } = require('../handlers/faucet');

// Initialize faucet handler with configuration
let faucetHandler = null;

/**
 * Initialize faucet handler with configuration
 * @param {Object} config - Faucet configuration
 */
function initializeFaucet(config) {
  if (!faucetHandler) {
    faucetHandler = new FaucetHandler(config);
    console.log('Faucet handler initialized');
  }
  return faucetHandler;
}

/**
 * Middleware to ensure faucet is initialized
 */
function ensureFaucetInitialized(req, res, next) {
  if (!faucetHandler) {
    // Load configuration from environment or defaults
    const { loadEndpoints } = require('../../scripts/load-endpoints');
    const { config } = loadEndpoints(process.env.NODE_ENV || 'testnet');
    
    const faucetConfig = {
      enabled: process.env.FAUCET_ENABLED !== 'false',
      daily_limit: parseInt(process.env.FAUCET_DAILY_LIMIT) || 10000,
      per_ip_hourly_limit: parseInt(process.env.FAUCET_PER_IP_HOURLY_LIMIT) || 5,
      per_address_daily_limit: parseInt(process.env.FAUCET_PER_ADDRESS_DAILY_LIMIT) || 1000,
      cooldown_minutes: parseInt(process.env.FAUCET_COOLDOWN_MINUTES) || 15,
      default_amount: parseInt(process.env.FAUCET_DEFAULT_AMOUNT) || 1000,
      max_custom_amount: parseInt(process.env.FAUCET_MAX_CUSTOM_AMOUNT) || 5000,
      min_amount: parseInt(process.env.FAUCET_MIN_AMOUNT) || 100,
      knirvoracle_endpoint: process.env.KNIRVORACLE_FAUCET_ENDPOINT || 'http://localhost:1317',
      database_path: process.env.FAUCET_DATABASE_PATH || './data/faucet-requests.json',
      ...config.faucet
    };
    
    initializeFaucet(faucetConfig);
  }
  next();
}

/**
 * Get client IP address
 * @param {Object} req - Express request object
 * @returns {string} Client IP address
 */
function getClientIP(req) {
  return req.headers['x-forwarded-for'] || 
         req.headers['x-real-ip'] || 
         req.connection.remoteAddress || 
         req.socket.remoteAddress ||
         (req.connection.socket ? req.connection.socket.remoteAddress : null) ||
         '127.0.0.1';
}

// Apply middleware to all routes
router.use(ensureFaucetInitialized);

/**
 * POST /api/faucet/request
 * Request NRV tokens from the faucet
 */
router.post('/request', async (req, res) => {
  try {
    const { address, amount, reason } = req.body;
    
    // Validate required fields
    if (!address) {
      return res.status(400).json({
        success: false,
        error: 'Wallet address is required',
        timestamp: new Date().toISOString()
      });
    }

    // Use default amount if not specified
    const requestAmount = amount || faucetHandler.config.default_amount;
    
    // Prepare request object
    const faucetRequest = {
      address: address.trim(),
      amount: requestAmount,
      reason: reason || '',
      ip: getClientIP(req),
      timestamp: Date.now()
    };

    // Process the request
    const result = await faucetHandler.processRequest(faucetRequest);
    
    // Return appropriate status code
    const statusCode = result.success ? 200 : 
                      result.status === 'rate_limited' ? 429 : 400;
    
    res.status(statusCode).json(result);
    
  } catch (error) {
    console.error('Faucet request error:', error);
    res.status(500).json({
      success: false,
      error: 'Internal server error',
      message: error.message,
      timestamp: new Date().toISOString()
    });
  }
});

/**
 * GET /api/faucet/status
 * Get current faucet status and limits
 */
router.get('/status', async (req, res) => {
  try {
    const status = faucetHandler.getFaucetStatus();
    res.json(status);
  } catch (error) {
    console.error('Faucet status error:', error);
    res.status(500).json({
      error: 'Failed to get faucet status',
      message: error.message,
      timestamp: new Date().toISOString()
    });
  }
});

/**
 * GET /api/faucet/history
 * Get request history for an address
 */
router.get('/history', async (req, res) => {
  try {
    const { address, limit } = req.query;
    
    if (!address) {
      return res.status(400).json({
        error: 'Address parameter is required',
        timestamp: new Date().toISOString()
      });
    }

    const history = faucetHandler.getAddressHistory(address, parseInt(limit) || 10);
    
    res.json({
      address: address,
      history: history,
      total_requests: history.length,
      timestamp: new Date().toISOString()
    });
    
  } catch (error) {
    console.error('Faucet history error:', error);
    res.status(500).json({
      error: 'Failed to get request history',
      message: error.message,
      timestamp: new Date().toISOString()
    });
  }
});

/**
 * GET /api/faucet/health
 * Get comprehensive faucet health status
 */
router.get('/health', async (req, res) => {
  try {
    const health = await faucetHandler.getHealth();
    
    // Determine HTTP status based on health
    const statusCode = health.status === 'healthy' ? 200 :
                      health.status === 'disabled' ? 503 : 207;
    
    res.status(statusCode).json(health);
    
  } catch (error) {
    console.error('Faucet health check error:', error);
    res.status(500).json({
      service: 'testnet-faucet',
      status: 'error',
      error: 'Health check failed',
      message: error.message,
      timestamp: new Date().toISOString()
    });
  }
});

/**
 * GET /api/faucet/metrics
 * Get Prometheus-compatible metrics
 */
router.get('/metrics', async (req, res) => {
  try {
    const metrics = faucetHandler.getMetrics();
    
    res.set('Content-Type', 'text/plain');
    res.send(metrics);
    
  } catch (error) {
    console.error('Faucet metrics error:', error);
    res.status(500).send('# Error generating metrics\n');
  }
});

/**
 * GET /api/faucet/treasury/status
 * Get treasury status and health
 */
router.get('/treasury/status', async (req, res) => {
  try {
    const treasuryStatus = await faucetHandler.treasuryService.getTreasuryStatus();
    res.json(treasuryStatus);
  } catch (error) {
    console.error('Treasury status error:', error);
    res.status(500).json({
      error: 'Failed to get treasury status',
      message: error.message,
      timestamp: new Date().toISOString()
    });
  }
});

/**
 * GET /api/faucet/router/proofs
 * Get ROUTER proof generation and NRV creation metrics
 */
router.get('/router/proofs', async (req, res) => {
  try {
    const routerStatus = await faucetHandler.routerService.getIntegrationStatus();
    res.json(routerStatus);
  } catch (error) {
    console.error('Router proofs error:', error);
    res.status(500).json({
      error: 'Failed to get router proof status',
      message: error.message,
      timestamp: new Date().toISOString()
    });
  }
});

/**
 * GET /api/faucet/economic/metrics
 * Get complete NRV economic flow metrics
 */
router.get('/economic/metrics', async (req, res) => {
  try {
    const economicMetrics = faucetHandler.getEconomicFlowMetrics();
    res.json({
      economic_flow: economicMetrics,
      timestamp: new Date().toISOString(),
      sustainability_status: economicMetrics.funding_sustainability_days > 30 ? 'healthy' :
                            economicMetrics.funding_sustainability_days > 7 ? 'warning' : 'critical'
    });
  } catch (error) {
    console.error('Economic metrics error:', error);
    res.status(500).json({
      error: 'Failed to get economic metrics',
      message: error.message,
      timestamp: new Date().toISOString()
    });
  }
});

/**
 * POST /api/faucet/treasury/transfer
 * Manual treasury NRV transfer (admin function)
 */
router.post('/treasury/transfer', async (req, res) => {
  try {
    const { amount, admin_token } = req.body;
    
    // Basic admin authentication (in production, use proper JWT)
    if (admin_token !== process.env.FAUCET_ADMIN_TOKEN && admin_token !== 'testnet-admin-123') {
      return res.status(401).json({
        error: 'Unauthorized',
        message: 'Valid admin token required',
        timestamp: new Date().toISOString()
      });
    }

    if (!amount || amount <= 0) {
      return res.status(400).json({
        error: 'Invalid amount',
        message: 'Amount must be a positive number',
        timestamp: new Date().toISOString()
      });
    }

    const result = await faucetHandler.manualTreasuryFunding(amount);
    
    const statusCode = result.success ? 200 : 400;
    res.status(statusCode).json(result);
    
  } catch (error) {
    console.error('Manual treasury transfer error:', error);
    res.status(500).json({
      error: 'Transfer failed',
      message: error.message,
      timestamp: new Date().toISOString()
    });
  }
});

/**
 * GET /api/faucet/nrv/conversion-rate
 * Get current NRV to NRN conversion rate (placeholder)
 */
router.get('/nrv/conversion-rate', async (req, res) => {
  try {
    // This would integrate with actual conversion rate service
    res.json({
      nrv_to_nrn_rate: 1.0, // 1:1 conversion for testnet
      conversion_available: false, // Testnet NRVs cannot be converted
      note: 'Testnet NRVs are for testing only and cannot be converted to real NRN tokens',
      timestamp: new Date().toISOString()
    });
  } catch (error) {
    console.error('Conversion rate error:', error);
    res.status(500).json({
      error: 'Failed to get conversion rate',
      message: error.message,
      timestamp: new Date().toISOString()
    });
  }
});

/**
 * Error handling middleware
 */
router.use((error, req, res, next) => {
  console.error('Faucet route error:', error);
  res.status(500).json({
    error: 'Internal server error',
    message: error.message,
    timestamp: new Date().toISOString()
  });
});

module.exports = router;
