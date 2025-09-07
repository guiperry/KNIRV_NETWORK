// Configuration Loader for KNIRV TESTNET Gateway Functions
// This module loads testnet configuration with environment variables and defaults

const fs = require('fs');
const path = require('path');

class TestnetConfigLoader {
    constructor() {
        this.config = {};
        this.loaded = false;
    }

    loadConfig() {
        if (this.loaded) {
            return this.config;
        }

        console.log('🔧 Loading KNIRV TESTNET configuration...');

        // Load from environment variables
        this.mergeWithEnvironment();
        
        // Set testnet defaults
        this.setTestnetDefaults();
        
        this.loaded = true;
        console.log('✅ KNIRV TESTNET configuration loaded');
        return this.config;
    }

    mergeWithEnvironment() {
        // Load KNIRV TESTNET environment variables
        const testnetEnvVars = [
            'NODE_ENV',
            'TESTNET_MODE',
            'KNIRVORACLE_URL',
            'KNIRVCHAIN_URL',
            'KNIRVGRAPH_URL',
            'KNIRVNEXUS_URL',
            'KNIRVNEXUS_API_URL',
            'KNIRVROUTER_URL',
            'KNIRV_NETWORK',
            'KNIRV_CHAIN_ID',
            'JWT_SECRET',
            'DEBUG_MODE',
            'LOG_LEVEL',
            'MOCK_TEE',
            'DEPLOYMENT_ENV'
        ];

        for (const envVar of testnetEnvVars) {
            if (process.env[envVar] !== undefined) {
                this.config[envVar] = process.env[envVar];
            }
        }
    }

    setTestnetDefaults() {
        const testnetDefaults = {
            NODE_ENV: 'testnet',
            TESTNET_MODE: 'true',
            KNIRVORACLE_URL: 'http://localhost:1317',
            KNIRVCHAIN_URL: 'http://localhost:8090',
            KNIRVGRAPH_URL: 'http://localhost:8082',
            KNIRVNEXUS_URL: 'http://localhost:8084',
            KNIRVNEXUS_API_URL: 'http://localhost:8084/api',
            KNIRVROUTER_URL: 'http://localhost:8086',
            KNIRV_NETWORK: 'testnet',
            KNIRV_CHAIN_ID: 'knirv-testnet-1',
            JWT_SECRET: 'testnet-jwt-secret-not-for-production',
            DEBUG_MODE: 'true',
            LOG_LEVEL: 'debug',
            MOCK_TEE: 'true',
            DEPLOYMENT_ENV: 'testnet'
        };

        for (const [key, value] of Object.entries(testnetDefaults)) {
            if (this.config[key] === undefined) {
                this.config[key] = value;
            }
        }
    }

    get(key, defaultValue = undefined) {
        if (!this.loaded) {
            this.loadConfig();
        }
        return this.config[key] !== undefined ? this.config[key] : defaultValue;
    }

    getBoolean(key, defaultValue = false) {
        const value = this.get(key);
        if (value === undefined) return defaultValue;
        return value === 'true' || value === '1' || value === 'yes' || value === 'on';
    }

    getNumber(key, defaultValue = 0) {
        const value = this.get(key);
        if (value === undefined) return defaultValue;
        const num = parseInt(value, 10);
        return isNaN(num) ? defaultValue : num;
    }

    getArray(key, defaultValue = [], separator = ',') {
        const value = this.get(key);
        if (value === undefined) return defaultValue;
        return value.split(separator).map(item => item.trim()).filter(item => item.length > 0);
    }

    isDevelopment() {
        return this.get('NODE_ENV') === 'development';
    }

    isProduction() {
        return this.get('NODE_ENV') === 'production';
    }

    isTestnet() {
        return this.get('NODE_ENV') === 'testnet' || this.getBoolean('TESTNET_MODE');
    }

    isDebugEnabled() {
        return this.getBoolean('DEBUG_MODE');
    }
}

// Create singleton instance for testnet
const testnetConfig = new TestnetConfigLoader();

// Export both the instance and a loadConfig function for compatibility
module.exports = {
    loadConfig: () => testnetConfig.loadConfig(),
    get: (key, defaultValue) => testnetConfig.get(key, defaultValue),
    getBoolean: (key, defaultValue) => testnetConfig.getBoolean(key, defaultValue),
    getNumber: (key, defaultValue) => testnetConfig.getNumber(key, defaultValue),
    getArray: (key, defaultValue, separator) => testnetConfig.getArray(key, defaultValue, separator),
    isDevelopment: () => testnetConfig.isDevelopment(),
    isProduction: () => testnetConfig.isProduction(),
    isTestnet: () => testnetConfig.isTestnet(),
    isDebugEnabled: () => testnetConfig.isDebugEnabled(),
    config: testnetConfig
};
