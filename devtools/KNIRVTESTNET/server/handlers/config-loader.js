/**
 * Configuration Loader Handler
 * Migrated from KNIRVGATEWAY/netlify/functions/config-loader.js
 */

const getConfig = (req, res) => {
  try {
    const { loadEndpoints } = require('../../scripts/load-endpoints');
    const { endpoints, config } = loadEndpoints(process.env.NODE_ENV || 'testnet');
    
    res.json({
      success: true,
      environment: process.env.NODE_ENV || 'testnet',
      endpoints,
      config: {
        ...config,
        // Remove sensitive information
        JWT_SECRET: undefined,
        DATABASE_URL: undefined,
        REDIS_URL: undefined
      },
      timestamp: new Date().toISOString()
    });
  } catch (error) {
    res.status(500).json({
      success: false,
      error: 'Failed to load configuration',
      message: error.message,
      timestamp: new Date().toISOString()
    });
  }
};

const updateConfig = (req, res) => {
  // For testnet, we can allow some configuration updates
  const { environment, endpoints: newEndpoints } = req.body;
  
  if (process.env.NODE_ENV !== 'testnet') {
    return res.status(403).json({
      success: false,
      error: 'Configuration updates only allowed in testnet environment'
    });
  }
  
  try {
    // In a real implementation, this would update the configuration
    // For now, just return success
    res.json({
      success: true,
      message: 'Configuration update received',
      environment: environment || process.env.NODE_ENV,
      timestamp: new Date().toISOString()
    });
  } catch (error) {
    res.status(500).json({
      success: false,
      error: 'Failed to update configuration',
      message: error.message,
      timestamp: new Date().toISOString()
    });
  }
};

module.exports = {
  getConfig,
  updateConfig
};
