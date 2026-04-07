#!/usr/bin/env node

/**
 * KNIRVTESTNET Configuration Validator
 * 
 * Validates all configuration files in the config folder
 * and checks for consistency and completeness
 */

const fs = require('fs');
const path = require('path');
const yaml = require('js-yaml');
const ConfigLoader = require('./config-loader');

function validateConfig() {
  console.log('🔍 KNIRVTESTNET Configuration Validator');
  console.log('=======================================');
  
  const configRoot = path.resolve(__dirname, '..', 'config');
  const issues = [];
  const warnings = [];
  
  // Check if config directory exists
  if (!fs.existsSync(configRoot)) {
    issues.push('Config directory not found');
    return false;
  }
  
  console.log('✅ Config directory exists');
  
  // Required config files
  const requiredFiles = [
    'testnet-config.yaml',
    'ports-config.yaml',
    'test-config.yaml',
    'endpoints.yaml',
    'portal-config.json',
    'portal-links.yaml'
  ];
  
  // Check required files exist
  requiredFiles.forEach(file => {
    const filePath = path.join(configRoot, file);
    if (!fs.existsSync(filePath)) {
      issues.push(`Missing required config file: ${file}`);
    } else {
      console.log(`✅ Found: ${file}`);
    }
  });
  
  // Validate YAML files
  const yamlFiles = [
    'testnet-config.yaml',
    'ports-config.yaml', 
    'test-config.yaml',
    'endpoints.yaml',
    'portal-links.yaml'
  ];
  
  yamlFiles.forEach(file => {
    const filePath = path.join(configRoot, file);
    if (fs.existsSync(filePath)) {
      try {
        const content = fs.readFileSync(filePath, 'utf8');
        yaml.load(content);
        console.log(`✅ Valid YAML: ${file}`);
      } catch (error) {
        issues.push(`Invalid YAML in ${file}: ${error.message}`);
      }
    }
  });
  
  // Validate JSON files
  const jsonFiles = ['portal-config.json'];
  
  jsonFiles.forEach(file => {
    const filePath = path.join(configRoot, file);
    if (fs.existsSync(filePath)) {
      try {
        const content = fs.readFileSync(filePath, 'utf8');
        JSON.parse(content);
        console.log(`✅ Valid JSON: ${file}`);
      } catch (error) {
        issues.push(`Invalid JSON in ${file}: ${error.message}`);
      }
    }
  });
  
  // Validate configuration structure using ConfigLoader
  try {
    const loader = new ConfigLoader('testnet');
    
    // Test loading all configs
    const testnetConfig = loader.getTestnetConfig();
    const portsConfig = loader.getPortsConfig();
    const testConfig = loader.getTestConfig();
    const endpointsConfig = loader.getEndpointsConfig();
    const portalConfig = loader.getPortalConfig();
    const portalLinks = loader.getPortalLinks();
    
    console.log('✅ All config files loaded successfully');
    
    // Validate testnet config structure
    if (!testnetConfig.endpoints) {
      issues.push('testnet-config.yaml missing endpoints section');
    } else {
      const requiredEndpoints = ['knirvchain', 'knirvgraph', 'knirvserver', 'knirvoracle', 'knirvrouter', 'knirvana'];
      requiredEndpoints.forEach(endpoint => {
        if (!testnetConfig.endpoints[endpoint]) {
          warnings.push(`Missing endpoint in testnet-config.yaml: ${endpoint}`);
        }
      });
    }
    
    if (!testnetConfig.testnet) {
      issues.push('testnet-config.yaml missing testnet section');
    }
    
    if (!testnetConfig.server) {
      warnings.push('testnet-config.yaml missing server section');
    }
    
    // Validate ports config structure
    if (!portsConfig.core_services) {
      warnings.push('ports-config.yaml missing core_services section');
    }
    
    if (!portsConfig.gateway) {
      warnings.push('ports-config.yaml missing gateway section');
    }
    
    // Validate test config structure
    if (!testConfig.testnet) {
      warnings.push('test-config.yaml missing testnet section');
    }
    
    // Check for port conflicts
    const allPorts = [];
    if (portsConfig.core_services) {
      Object.values(portsConfig.core_services).forEach(port => {
        if (allPorts.includes(port)) {
          issues.push(`Port conflict detected: ${port}`);
        } else {
          allPorts.push(port);
        }
      });
    }
    
    if (portsConfig.gateway) {
      Object.values(portsConfig.gateway).forEach(port => {
        if (allPorts.includes(port)) {
          issues.push(`Port conflict detected: ${port}`);
        } else {
          allPorts.push(port);
        }
      });
    }
    
    console.log(`✅ Port validation completed (${allPorts.length} ports checked)`);
    
  } catch (error) {
    issues.push(`Configuration loading error: ${error.message}`);
  }
  
  // Check environment file
  const envFile = path.resolve(__dirname, '..', '.env.testnet');
  if (fs.existsSync(envFile)) {
    console.log('✅ Environment file exists: .env.testnet');

    // Check that .env.testnet only contains environment variables (NO config parameters or endpoints)
    const envContent = fs.readFileSync(envFile, 'utf8');

    // Configuration parameters that should NOT be in .env files
    const configParams = [
      'TESTNET_MODE', 'DEBUG_MODE', 'ENABLE_CORS', 'AUTH_SIMPLIFIED',
      'CORS_ORIGIN', 'CORS_METHODS', 'RATE_LIMIT_ENABLED', 'LOG_LEVEL',
      'SSE_ENABLED', 'MOCK_RESPONSES', 'DEV_MODE', 'PORT', 'HOST'
    ];

    // Endpoint parameters that should NOT be in .env files
    const endpointParams = [
      'KNIRVCHAIN_API', 'KNIRVGRAPH_API', 'KNIRVNEXUS_API',
      'KNIRVORACLE_API', 'KNIRVROUTER_API', 'KNIRVANA_API'
    ];

    // Database and external service parameters that should NOT be in .env files
    const serviceParams = [
      'DATABASE_URL', 'REDIS_URL', 'IPFS_API_PORT', 'IPFS_GATEWAY_PORT',
      'XION_TESTNET_RPC', 'POSTGRES_HOST', 'POSTGRES_PORT', 'REDIS_HOST', 'REDIS_PORT'
    ];

    configParams.forEach(param => {
      if (envContent.includes(`${param}=`)) {
        issues.push(`Configuration parameter ${param} found in .env.testnet - should be in config/testnet-config.yaml`);
      }
    });

    endpointParams.forEach(param => {
      if (envContent.includes(`${param}=`)) {
        issues.push(`Endpoint ${param} found in .env.testnet - should be in config/testnet-config.yaml`);
      }
    });

    serviceParams.forEach(param => {
      if (envContent.includes(`${param}=`)) {
        issues.push(`Service parameter ${param} found in .env.testnet - should be in config/testnet-config.yaml`);
      }
    });

    // Check for allowed environment variables only (secrets and runtime environment only)
    const allowedEnvVars = ['NODE_ENV', 'DEPLOYMENT_ENV', 'JWT_SECRET'];
    const envLines = envContent.split('\n').filter(line => line.trim() && !line.trim().startsWith('#'));

    envLines.forEach(line => {
      const [key] = line.split('=');
      if (key && !allowedEnvVars.includes(key.trim())) {
        warnings.push(`Unexpected variable in .env.testnet: ${key} - verify this should be an environment variable`);
      }
    });

  } else {
    warnings.push('Environment file not found: .env.testnet');
  }
  
  // Summary
  console.log('\n📊 Configuration Validation Summary');
  console.log('===================================');
  
  if (issues.length === 0 && warnings.length === 0) {
    console.log('🎉 All configuration validation checks passed!');
    console.log('✅ Configuration is ready for deployment');
    return true;
  }
  
  if (warnings.length > 0) {
    console.log('\n⚠️  Warnings:');
    warnings.forEach(warning => console.log(`   - ${warning}`));
  }
  
  if (issues.length > 0) {
    console.log('\n❌ Issues found:');
    issues.forEach(issue => console.log(`   - ${issue}`));
    console.log('\n🔧 Please fix these issues before deployment');
    return false;
  }
  
  console.log('\n✅ Configuration validation completed with warnings only');
  return true;
}

// Run validation if called directly
if (require.main === module) {
  const valid = validateConfig();
  process.exit(valid ? 0 : 1);
}

module.exports = { validateConfig };
