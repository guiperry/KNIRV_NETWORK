#!/usr/bin/env node

/**
 * KNIRV Gateway Services Deployment Script
 * 
 * Handles the complete deployment process for gateway services:
 * 1. Detects CloudFlare credentials
 * 2. Updates DNS records if credentials available (public testnet)
 * 3. Falls back to localhost configuration (private testnet)
 * 4. Updates all configuration files
 * 5. Generates frontend configuration
 */

const fs = require('fs');
const path = require('path');
const { spawn } = require('child_process');

// Import our custom modules
const GatewayDNSManager = require('./update-gateway-dns');
const GatewayLinksLoader = require('./load-gateway-links');

class GatewayDeploymentManager {
  constructor() {
    this.envPath = path.join(__dirname, '..', '.env.testnet');
    this.hasCloudFlareCredentials = this.checkCloudFlareCredentials();
    this.deploymentMode = this.hasCloudFlareCredentials ? 'public_testnet' : 'private_testnet';
    
    console.log(`[Deployment] Mode: ${this.deploymentMode}`);
    console.log(`[Deployment] CloudFlare credentials: ${this.hasCloudFlareCredentials ? 'Available' : 'Not available'}`);
  }

  /**
   * Check if CloudFlare credentials are available
   */
  checkCloudFlareCredentials() {
    if (!fs.existsSync(this.envPath)) {
      console.log('[Deployment] No .env.testnet file found');
      return false;
    }

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

    const hasToken = envVars.CLOUDFLARE_API_TOKEN && envVars.CLOUDFLARE_API_TOKEN.length > 10;
    const hasZoneId = envVars.CLOUDFLARE_ZONE_ID && envVars.CLOUDFLARE_ZONE_ID.length > 10;

    return hasToken && hasZoneId;
  }

  /**
   * Update environment file with deployment mode
   */
  updateEnvironmentFile() {
    try {
      let envContent = '';
      
      if (fs.existsSync(this.envPath)) {
        envContent = fs.readFileSync(this.envPath, 'utf8');
      }

      // Add or update deployment mode
      if (envContent.includes('DEPLOYMENT_MODE=')) {
        envContent = envContent.replace(/DEPLOYMENT_MODE=.*/g, `DEPLOYMENT_MODE=${this.deploymentMode}`);
      } else {
        envContent += `\nDEPLOYMENT_MODE=${this.deploymentMode}\n`;
      }

      // Add timestamp
      const timestamp = new Date().toISOString();
      if (envContent.includes('LAST_DEPLOYMENT=')) {
        envContent = envContent.replace(/LAST_DEPLOYMENT=.*/g, `LAST_DEPLOYMENT=${timestamp}`);
      } else {
        envContent += `LAST_DEPLOYMENT=${timestamp}\n`;
      }

      fs.writeFileSync(this.envPath, envContent);
      console.log(`[Deployment] ✅ Environment file updated with mode: ${this.deploymentMode}`);
    } catch (error) {
      console.error('[Deployment] Failed to update environment file:', error.message);
      throw error;
    }
  }

  /**
   * Deploy DNS records (public testnet only)
   */
  async deployDNSRecords() {
    if (!this.hasCloudFlareCredentials) {
      console.log('[Deployment] Skipping DNS deployment - no CloudFlare credentials');
      return true;
    }

    try {
      console.log('[Deployment] Deploying DNS records...');
      const dnsManager = new GatewayDNSManager();
      const success = await dnsManager.updateAllServiceDNS();
      
      if (success) {
        console.log('[Deployment] ✅ DNS records deployed successfully');
      } else {
        console.log('[Deployment] ❌ DNS deployment failed');
      }
      
      return success;
    } catch (error) {
      console.error('[Deployment] DNS deployment error:', error.message);
      return false;
    }
  }

  /**
   * Update configuration files
   */
  updateConfigurationFiles() {
    try {
      console.log('[Deployment] Updating configuration files...');
      
      const linksLoader = new GatewayLinksLoader();
      
      // Update network website configuration
      const success = linksLoader.updateNetworkWebsiteConfig();
      
      if (success) {
        console.log('[Deployment] ✅ Configuration files updated');
      } else {
        console.log('[Deployment] ❌ Configuration update failed');
      }
      
      return success;
    } catch (error) {
      console.error('[Deployment] Configuration update error:', error.message);
      return false;
    }
  }

  /**
   * Verify service availability
   */
  async verifyServices() {
    console.log('[Deployment] Verifying service availability...');
    
    const linksLoader = new GatewayLinksLoader();
    const gatewayServices = linksLoader.getGatewayServiceUrls();
    
    const results = [];
    
    for (const [serviceName, serviceConfig] of Object.entries(gatewayServices)) {
      try {
        const axios = require('axios');
        const response = await axios.get(serviceConfig.health_url, { 
          timeout: 5000,
          validateStatus: () => true // Accept any status code
        });
        
        const isHealthy = response.status >= 200 && response.status < 400;
        results.push({
          service: serviceName,
          url: serviceConfig.base_url,
          health_url: serviceConfig.health_url,
          status: response.status,
          healthy: isHealthy
        });
        
        if (isHealthy) {
          console.log(`[Deployment] ✅ ${serviceName}: ${serviceConfig.base_url} (${response.status})`);
        } else {
          console.log(`[Deployment] ⚠️  ${serviceName}: ${serviceConfig.base_url} (${response.status})`);
        }
      } catch (error) {
        results.push({
          service: serviceName,
          url: serviceConfig.base_url,
          healthy: false,
          error: error.message
        });
        console.log(`[Deployment] ❌ ${serviceName}: ${serviceConfig.base_url} (${error.message})`);
      }
    }
    
    const healthyCount = results.filter(r => r.healthy).length;
    console.log(`[Deployment] Service health: ${healthyCount}/${results.length} services healthy`);
    
    return results;
  }

  /**
   * Generate deployment summary
   */
  generateSummary(dnsSuccess, configSuccess, serviceResults) {
    const summary = {
      deployment_mode: this.deploymentMode,
      timestamp: new Date().toISOString(),
      cloudflare_credentials: this.hasCloudFlareCredentials,
      dns_deployment: dnsSuccess,
      config_update: configSuccess,
      services: serviceResults,
      healthy_services: serviceResults.filter(r => r.healthy).length,
      total_services: serviceResults.length
    };

    // Save summary to file
    const summaryPath = path.join(__dirname, '..', 'logs', 'deployment-summary.json');
    
    try {
      // Ensure logs directory exists
      const logsDir = path.dirname(summaryPath);
      if (!fs.existsSync(logsDir)) {
        fs.mkdirSync(logsDir, { recursive: true });
      }
      
      fs.writeFileSync(summaryPath, JSON.stringify(summary, null, 2));
      console.log(`[Deployment] ✅ Summary saved to ${summaryPath}`);
    } catch (error) {
      console.error('[Deployment] Failed to save summary:', error.message);
    }

    return summary;
  }

  /**
   * Run complete deployment process
   */
  async deploy() {
    console.log('=== KNIRV Gateway Services Deployment ===');
    console.log(`Starting deployment in ${this.deploymentMode} mode...`);
    
    try {
      // Step 1: Update environment file
      this.updateEnvironmentFile();
      
      // Step 2: Deploy DNS records (if credentials available)
      const dnsSuccess = await this.deployDNSRecords();
      
      // Step 3: Update configuration files
      const configSuccess = this.updateConfigurationFiles();
      
      // Step 4: Wait for services to be ready (if DNS was updated)
      if (dnsSuccess && this.hasCloudFlareCredentials) {
        console.log('[Deployment] Waiting for DNS propagation...');
        await new Promise(resolve => setTimeout(resolve, 10000));
      }
      
      // Step 5: Verify services
      const serviceResults = await this.verifyServices();
      
      // Step 6: Generate summary
      const summary = this.generateSummary(dnsSuccess, configSuccess, serviceResults);
      
      // Final status
      const overallSuccess = dnsSuccess && configSuccess && summary.healthy_services > 0;
      
      console.log('\n=== Deployment Summary ===');
      console.log(`Mode: ${summary.deployment_mode}`);
      console.log(`DNS Deployment: ${dnsSuccess ? '✅' : '❌'}`);
      console.log(`Config Update: ${configSuccess ? '✅' : '❌'}`);
      console.log(`Healthy Services: ${summary.healthy_services}/${summary.total_services}`);
      console.log(`Overall Status: ${overallSuccess ? '✅ SUCCESS' : '❌ PARTIAL/FAILED'}`);
      console.log('==========================\n');
      
      if (!overallSuccess) {
        process.exit(1);
      }
      
    } catch (error) {
      console.error('[Deployment] ❌ Deployment failed:', error.message);
      process.exit(1);
    }
  }
}

// CLI interface
if (require.main === module) {
  const manager = new GatewayDeploymentManager();
  manager.deploy();
}

module.exports = GatewayDeploymentManager;
