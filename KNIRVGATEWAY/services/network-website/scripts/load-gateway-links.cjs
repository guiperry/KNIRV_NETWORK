#!/usr/bin/env node

/**
 * Gateway Links Loader
 * 
 * Loads and processes gateway service links based on deployment environment.
 * Provides a unified interface for accessing gateway service URLs throughout the application.
 */

const fs = require('fs');
const path = require('path');
const yaml = require('js-yaml');

class GatewayLinksLoader {
  constructor() {
    this.configPath = path.join(__dirname, '..', 'config', 'portal-links.yaml');
    this.networkConfigPath = path.join(__dirname, '..', 'network-website', 'public', 'config', 'portal-links.yaml');
    this.envPath = path.join(__dirname, '..', '.env.testnet');
    this.deploymentMode = this.detectDeploymentMode();
    this.config = null;
  }

  /**
   * Detect deployment mode based on CloudFlare credentials
   */
  detectDeploymentMode() {
    // Load environment variables
    if (fs.existsSync(this.envPath)) {
      const envContent = fs.readFileSync(this.envPath, 'utf8');
      const envVars = {};
      
      envContent.split('\n').forEach(line => {
        const trimmed = line.trim();
        if (trimmed && !trimmed.startsWith('#')) {
          const [key, ...valueParts] = trimmed.split('=');
          const value = valueParts.join('=');
          if (key && value) {
            envVars[key] = value;
          }
        }
      });

      // Check for CloudFlare credentials
      const hasCloudFlareCredentials = envVars.CLOUDFLARE_API_TOKEN && envVars.CLOUDFLARE_ZONE_ID;
      
      // Check explicit deployment mode
      if (envVars.DEPLOYMENT_MODE) {
        return envVars.DEPLOYMENT_MODE;
      }
      
      return hasCloudFlareCredentials ? 'public_testnet' : 'private_testnet';
    }
    
    return 'private_testnet';
  }

  /**
   * Load configuration from YAML file
   */
  loadConfig(configPath = null) {
    const filePath = configPath || this.configPath;
    
    try {
      if (!fs.existsSync(filePath)) {
        throw new Error(`Configuration file not found: ${filePath}`);
      }
      
      const yamlContent = fs.readFileSync(filePath, 'utf8');
      this.config = yaml.load(yamlContent);
      
      console.log(`[Links Loader] Configuration loaded from ${filePath}`);
      console.log(`[Links Loader] Deployment mode: ${this.deploymentMode}`);
      
      return this.config;
    } catch (error) {
      console.error(`[Links Loader] Failed to load configuration: ${error.message}`);
      throw error;
    }
  }

  /**
   * Get gateway service URLs based on deployment mode
   */
  getGatewayServiceUrls() {
    if (!this.config) {
      this.loadConfig();
    }

    const gatewayServices = this.config.gateway_services;
    if (!gatewayServices || !gatewayServices[this.deploymentMode]) {
      console.warn(`[Links Loader] No gateway services configuration found for mode: ${this.deploymentMode}`);
      return {};
    }

    const services = gatewayServices[this.deploymentMode];
    const urls = {};

    for (const [serviceName, serviceConfig] of Object.entries(services)) {
      const protocol = serviceConfig.domain === 'localhost' ? 'http' : 'https';
      const port = serviceConfig.domain === 'localhost' ? `:${serviceConfig.port}` : '';
      
      urls[serviceName] = {
        base_url: `${protocol}://${serviceConfig.domain}${port}${serviceConfig.path}`,
        health_url: `${protocol}://${serviceConfig.domain}${port}${serviceConfig.health_endpoint}`,
        domain: serviceConfig.domain,
        port: serviceConfig.port,
        path: serviceConfig.path
      };
    }

    return urls;
  }

  /**
   * Get external service URLs
   */
  getExternalServiceUrls() {
    if (!this.config) {
      this.loadConfig();
    }

    return this.config.external_services || {};
  }

  /**
   * Get navigation links
   */
  getNavigationLinks() {
    if (!this.config) {
      this.loadConfig();
    }

    return this.config.navigation || {};
  }

  /**
   * Get all links in a unified format
   */
  getAllLinks() {
    const gatewayServices = this.getGatewayServiceUrls();
    const externalServices = this.getExternalServiceUrls();
    const navigation = this.getNavigationLinks();

    return {
      deployment_mode: this.deploymentMode,
      gateway_services: gatewayServices,
      external_services: externalServices,
      navigation: navigation,
      timestamp: new Date().toISOString()
    };
  }

  /**
   * Generate JavaScript configuration for frontend
   */
  generateJSConfig() {
    const allLinks = this.getAllLinks();
    
    return `
// Auto-generated KNIRV Gateway Links Configuration
// Generated at: ${allLinks.timestamp}
// Deployment Mode: ${allLinks.deployment_mode}

window.KNIRV_GATEWAY_CONFIG = ${JSON.stringify(allLinks, null, 2)};

// Helper functions for easy access
window.getGatewayServiceUrl = function(serviceName) {
  const service = window.KNIRV_GATEWAY_CONFIG.gateway_services[serviceName];
  return service ? service.base_url : null;
};

window.getGatewayServiceHealthUrl = function(serviceName) {
  const service = window.KNIRV_GATEWAY_CONFIG.gateway_services[serviceName];
  return service ? service.health_url : null;
};

window.isPrivateTestnet = function() {
  return window.KNIRV_GATEWAY_CONFIG.deployment_mode === 'private_testnet';
};

window.isPublicTestnet = function() {
  return window.KNIRV_GATEWAY_CONFIG.deployment_mode === 'public_testnet';
};
`;
  }

  /**
   * Save JavaScript configuration to file
   */
  saveJSConfig(outputPath = null) {
    const defaultPath = path.join(__dirname, '..', 'network-website', 'public', 'js', 'gateway-config.js');
    const filePath = outputPath || defaultPath;
    
    try {
      const jsConfig = this.generateJSConfig();
      
      // Ensure directory exists
      const dir = path.dirname(filePath);
      if (!fs.existsSync(dir)) {
        fs.mkdirSync(dir, { recursive: true });
      }
      
      fs.writeFileSync(filePath, jsConfig);
      console.log(`[Links Loader] ✅ JavaScript configuration saved to ${filePath}`);
      
      return filePath;
    } catch (error) {
      console.error(`[Links Loader] Failed to save JavaScript configuration: ${error.message}`);
      throw error;
    }
  }

  /**
   * Update network-website configuration
   */
  updateNetworkWebsiteConfig() {
    try {
      // Load network-website specific configuration
      this.loadConfig(this.networkConfigPath);
      
      // Generate and save JavaScript configuration
      this.saveJSConfig();
      
      console.log('[Links Loader] ✅ Network website configuration updated');
      return true;
    } catch (error) {
      console.error(`[Links Loader] Failed to update network website configuration: ${error.message}`);
      return false;
    }
  }

  /**
   * Print current configuration summary
   */
  printSummary() {
    const allLinks = this.getAllLinks();
    
    console.log('\n=== KNIRV Gateway Links Configuration ===');
    console.log(`Deployment Mode: ${allLinks.deployment_mode}`);
    console.log(`Timestamp: ${allLinks.timestamp}`);
    
    console.log('\nGateway Services:');
    for (const [serviceName, serviceConfig] of Object.entries(allLinks.gateway_services)) {
      console.log(`  ${serviceName}: ${serviceConfig.base_url}`);
    }
    
    console.log('\nExternal Services:');
    for (const [serviceName, url] of Object.entries(allLinks.external_services)) {
      console.log(`  ${serviceName}: ${url}`);
    }
    
    console.log('==========================================\n');
  }
}

// CLI interface
if (require.main === module) {
  const loader = new GatewayLinksLoader();
  
  const command = process.argv[2];
  
  switch (command) {
    case 'summary':
      loader.printSummary();
      break;
      
    case 'update-website':
      loader.updateNetworkWebsiteConfig();
      break;
      
    case 'generate-js':
      loader.saveJSConfig();
      break;
      
    default:
      console.log('KNIRV Gateway Links Loader');
      console.log('Usage: node load-gateway-links.js [command]');
      console.log('Commands:');
      console.log('  summary        - Print configuration summary');
      console.log('  update-website - Update network-website configuration');
      console.log('  generate-js    - Generate JavaScript configuration');
      loader.printSummary();
  }
}

module.exports = GatewayLinksLoader;
