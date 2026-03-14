#!/usr/bin/env node

/**
 * KNIRVTESTNET Configuration Loader
 * 
 * Centralized configuration loading utility that reads from the config folder
 * and provides a unified interface for accessing configuration parameters
 */

const fs = require('fs');
const path = require('path');
const yaml = require('js-yaml');

class ConfigLoader {
  constructor(environment = 'testnet') {
    this.environment = environment;
    this.configRoot = path.resolve(__dirname, '..', 'config');
    this.cache = new Map();
  }

  /**
   * Load configuration from a YAML file
   */
  loadYamlConfig(filename) {
    if (this.cache.has(filename)) {
      return this.cache.get(filename);
    }

    const filePath = path.join(this.configRoot, filename);
    
    if (!fs.existsSync(filePath)) {
      console.warn(`⚠️  Config file not found: ${filename}`);
      return {};
    }

    try {
      const content = fs.readFileSync(filePath, 'utf8');
      const config = yaml.load(content);
      this.cache.set(filename, config);
      return config;
    } catch (error) {
      console.error(`❌ Error loading config file ${filename}:`, error.message);
      return {};
    }
  }

  /**
   * Get testnet configuration
   */
  getTestnetConfig() {
    return this.loadYamlConfig('testnet-config.yaml');
  }

  /**
   * Get ports configuration
   */
  getPortsConfig() {
    return this.loadYamlConfig('ports-config.yaml');
  }

  /**
   * Get test configuration
   */
  getTestConfig() {
    return this.loadYamlConfig('test-config.yaml');
  }

  /**
   * Get endpoints configuration
   */
  getEndpointsConfig() {
    return this.loadYamlConfig('endpoints.yaml');
  }

  /**
   * Get portal configuration
   */
  getPortalConfig() {
    const configFile = path.join(this.configRoot, 'portal-config.json');
    
    if (!fs.existsSync(configFile)) {
      return {};
    }

    try {
      const content = fs.readFileSync(configFile, 'utf8');
      return JSON.parse(content);
    } catch (error) {
      console.error('❌ Error loading portal config:', error.message);
      return {};
    }
  }

  /**
   * Get portal links configuration
   */
  getPortalLinks() {
    return this.loadYamlConfig('portal-links.yaml');
  }

  /**
   * Get all service endpoints for the current environment
   */
  getServiceEndpoints() {
    const testnetConfig = this.getTestnetConfig();
    const portsConfig = this.getPortsConfig();

    const endpoints = {};

    // Load endpoints from testnet config (URLs)
    if (testnetConfig.endpoints) {
      Object.entries(testnetConfig.endpoints).forEach(([service, url]) => {
        const apiKey = service.toUpperCase().replace('-', '') + '_API';
        endpoints[apiKey] = url;
      });
    }

    // If no URLs in testnet config, build from ports config
    if (Object.keys(endpoints).length === 0 && portsConfig.core_services) {
      Object.entries(portsConfig.core_services).forEach(([service, port]) => {
        const apiKey = service.toUpperCase().replace('-', '') + '_API';
        endpoints[apiKey] = `http://localhost:${port}`;
      });
    }

    return endpoints;
  }

  /**
   * Get all configuration parameters for the current environment
   */
  getAllConfig() {
    const testnetConfig = this.getTestnetConfig();
    const portsConfig = this.getPortsConfig();
    const testConfig = this.getTestConfig();
    
    return {
      environment: this.environment,
      testnet: testnetConfig,
      ports: portsConfig,
      test: testConfig,
      endpoints: this.getServiceEndpoints()
    };
  }

  /**
   * Get flattened configuration for backward compatibility
   */
  getFlatConfig() {
    const testnetConfig = this.getTestnetConfig();
    const config = {
      NODE_ENV: this.environment,
      DEPLOYMENT_ENV: this.environment,
      TESTNET_MODE: this.environment === 'testnet',
      DEBUG_MODE: this.environment !== 'production',
      ENABLE_CORS: this.environment !== 'production'
    };

    // Map testnet configuration to flat structure
    if (testnetConfig.testnet) {
      config.TESTNET_MODE = testnetConfig.testnet.mode;
      config.DEBUG_MODE = testnetConfig.testnet.debug_mode;
      config.DEV_MODE = testnetConfig.testnet.dev_mode;
      config.DEBUG_ENABLED = testnetConfig.testnet.debug_enabled;
      config.MOCK_RESPONSES = testnetConfig.testnet.mock_responses;
    }

    if (testnetConfig.server) {
      config.ENABLE_CORS = testnetConfig.server.enable_cors;
      config.HOST = testnetConfig.server.host;
      config.GATEWAY_PORT = testnetConfig.server.default_port;
      config.PORT = testnetConfig.server.runtime_port;
    }

    if (testnetConfig.auth) {
      config.AUTH_SIMPLIFIED = testnetConfig.auth.simplified;
      config.AUTH_TESTNET_TOKENS = testnetConfig.auth.testnet_tokens;
      config.TESTNET_ADMIN_TOKEN = testnetConfig.auth.testnet_admin_token;
      config.TESTNET_VALIDATOR_TOKEN = testnetConfig.auth.testnet_validator_token;
      config.TESTNET_OBSERVER_TOKEN = testnetConfig.auth.testnet_observer_token;
    }

    if (testnetConfig.cors) {
      config.CORS_ORIGIN = testnetConfig.cors.origin;
      config.CORS_METHODS = testnetConfig.cors.methods;
      config.CORS_HEADERS = testnetConfig.cors.headers;
    }

    if (testnetConfig.rate_limit) {
      config.RATE_LIMIT_ENABLED = testnetConfig.rate_limit.enabled;
      config.RATE_LIMIT_REQUESTS = testnetConfig.rate_limit.requests;
      config.RATE_LIMIT_WINDOW = testnetConfig.rate_limit.window;
    }

    if (testnetConfig.logging) {
      config.LOG_LEVEL = testnetConfig.logging.level;
      config.LOG_REQUESTS = testnetConfig.logging.requests;
    }

    if (testnetConfig.health_check) {
      config.HEALTH_CHECK_INTERVAL = testnetConfig.health_check.interval;
      config.HEALTH_CHECK_TIMEOUT = testnetConfig.health_check.timeout;
    }

    if (testnetConfig.sse) {
      config.SSE_ENABLED = testnetConfig.sse.enabled;
      config.SSE_HEARTBEAT_INTERVAL = testnetConfig.sse.heartbeat_interval;
    }

    if (testnetConfig.security) {
      config.JWT_SECRET = testnetConfig.security.default_jwt_secret;
    }

    if (testnetConfig.database) {
      config.DATABASE_URL = testnetConfig.database.postgres_url;
      config.REDIS_URL = testnetConfig.database.redis_url;
      config.POSTGRES_HOST = testnetConfig.database.postgres_host;
      config.POSTGRES_PORT = testnetConfig.database.postgres_port;
      config.POSTGRES_DATABASE = testnetConfig.database.postgres_database;
      config.REDIS_HOST = testnetConfig.database.redis_host;
      config.REDIS_PORT = testnetConfig.database.redis_port;
    }

    if (testnetConfig.external_services) {
      if (testnetConfig.external_services.ipfs) {
        config.IPFS_API_PORT = testnetConfig.external_services.ipfs.api_port;
        config.IPFS_GATEWAY_PORT = testnetConfig.external_services.ipfs.gateway_port;
      }
      if (testnetConfig.external_services.xion) {
        config.XION_TESTNET_RPC = testnetConfig.external_services.xion.testnet_rpc;
      }
    }

    return config;
  }

  /**
   * Clear configuration cache
   */
  clearCache() {
    this.cache.clear();
  }
}

// Export for use in other modules
module.exports = ConfigLoader;

// CLI usage
if (require.main === module) {
  const environment = process.argv[2] || 'testnet';
  const loader = new ConfigLoader(environment);
  
  console.log('🔧 KNIRVTESTNET Configuration Loader');
  console.log('====================================');
  console.log(`Environment: ${environment}`);
  console.log('');
  
  const config = loader.getAllConfig();
  console.log('📋 Loaded Configuration:');
  console.log(JSON.stringify(config, null, 2));
}
