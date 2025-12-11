#!/usr/bin/env node

/**
 * KNIRV Gateway DNS Management Script
 * 
 * Updates CloudFlare DNS records for gateway services when credentials are available.
 * Falls back to localhost configuration for private testnet when credentials are missing.
 * 
 * Services managed:
 * - Payment Gateway -> payments.knirv.network
 * - Tunnel Registry -> tunnel.knirv.network  
 * - Operator Registry -> operators.knirv.network
 * - WebGUI -> dashboard.knirv.network
 */

const fs = require('fs');
const path = require('path');
const axios = require('axios');

// Load environment variables
require('dotenv').config({ path: path.join(__dirname, '..', '.env.testnet') });

// Gateway services configuration
const GATEWAY_SERVICES = {
  payment_gateway: {
    domain: 'payments.knirv.network',
    port: 3001,
    subdomain: 'payments'
  },
  tunnel_registry: {
    domain: 'tunnel.knirv.network', 
    port: 3004,
    subdomain: 'tunnel'
  },
  operator_registry: {
    domain: 'operators.knirv.network',
    port: 3007,
    subdomain: 'operators'
  },
  webgui: {
    domain: 'dashboard.knirv.network',
    port: 3007,
    subdomain: 'dashboard'
  }
};

// CloudFlare configuration
const CLOUDFLARE_CONFIG = {
  API_BASE_URL: 'https://api.cloudflare.com/client/v4',
  ZONE_NAME: 'knirv.network',
  TTL: 60 // 60 seconds for fast updates
};

class GatewayDNSManager {
  constructor() {
    this.apiToken = process.env.CLOUDFLARE_API_TOKEN;
    this.zoneId = process.env.CLOUDFLARE_ZONE_ID;
    this.publicIP = null;
    this.isPrivateTestnet = !this.apiToken || !this.zoneId;
    
    console.log(`[DNS Manager] Mode: ${this.isPrivateTestnet ? 'Private Testnet' : 'Public Testnet'}`);
  }

  /**
   * Get the current public IP address
   */
  async getPublicIP() {
    try {
      const response = await axios.get('https://api.ipify.org?format=json', { timeout: 5000 });
      this.publicIP = response.data.ip;
      console.log(`[DNS Manager] Public IP: ${this.publicIP}`);
      return this.publicIP;
    } catch (error) {
      console.error('[DNS Manager] Failed to get public IP:', error.message);
      throw error;
    }
  }

  /**
   * Make CloudFlare API request
   */
  async makeAPIRequest(method, endpoint, data = null) {
    if (this.isPrivateTestnet) {
      throw new Error('CloudFlare credentials not available - running in private testnet mode');
    }

    const config = {
      method,
      url: `${CLOUDFLARE_CONFIG.API_BASE_URL}${endpoint}`,
      headers: {
        'Authorization': `Bearer ${this.apiToken}`,
        'Content-Type': 'application/json'
      },
      timeout: 10000
    };

    if (data) {
      config.data = data;
    }

    try {
      const response = await axios(config);
      return response.data;
    } catch (error) {
      console.error(`[DNS Manager] API request failed: ${error.message}`);
      throw error;
    }
  }

  /**
   * Get existing DNS record for a subdomain
   */
  async getDNSRecord(subdomain) {
    const recordName = `${subdomain}.knirv.network`;
    
    try {
      const response = await this.makeAPIRequest('GET', `/zones/${this.zoneId}/dns_records`, {
        name: recordName,
        type: 'A'
      });
      
      return response.result && response.result.length > 0 ? response.result[0] : null;
    } catch (error) {
      console.error(`[DNS Manager] Failed to get DNS record for ${recordName}:`, error.message);
      return null;
    }
  }

  /**
   * Update or create DNS record for a service
   */
  async updateServiceDNS(serviceName, serviceConfig) {
    if (this.isPrivateTestnet) {
      console.log(`[DNS Manager] Private testnet mode - skipping DNS update for ${serviceName}`);
      return false;
    }

    try {
      const existingRecord = await this.getDNSRecord(serviceConfig.subdomain);
      const recordData = {
        type: 'A',
        name: serviceConfig.domain,
        content: this.publicIP,
        ttl: CLOUDFLARE_CONFIG.TTL,
        comment: `KNIRV Gateway ${serviceName} - Auto-updated`
      };

      let response;
      if (existingRecord) {
        // Update existing record
        console.log(`[DNS Manager] Updating existing DNS record for ${serviceConfig.domain}`);
        response = await this.makeAPIRequest('PUT', `/zones/${this.zoneId}/dns_records/${existingRecord.id}`, recordData);
      } else {
        // Create new record
        console.log(`[DNS Manager] Creating new DNS record for ${serviceConfig.domain}`);
        response = await this.makeAPIRequest('POST', `/zones/${this.zoneId}/dns_records`, recordData);
      }

      console.log(`[DNS Manager] ✅ DNS record updated: ${serviceConfig.domain} -> ${this.publicIP}:${serviceConfig.port}`);
      return true;
    } catch (error) {
      console.error(`[DNS Manager] ❌ Failed to update DNS for ${serviceName}:`, error.message);
      return false;
    }
  }

  /**
   * Update all gateway service DNS records
   */
  async updateAllServiceDNS() {
    if (this.isPrivateTestnet) {
      console.log('[DNS Manager] Private testnet mode - using localhost configuration');
      await this.updateConfigFiles('private_testnet');
      return true;
    }

    try {
      await this.getPublicIP();
      
      const results = [];
      for (const [serviceName, serviceConfig] of Object.entries(GATEWAY_SERVICES)) {
        const success = await this.updateServiceDNS(serviceName, serviceConfig);
        results.push({ service: serviceName, success });
      }

      const successCount = results.filter(r => r.success).length;
      console.log(`[DNS Manager] Updated ${successCount}/${results.length} DNS records`);

      if (successCount > 0) {
        await this.updateConfigFiles('public_testnet');
      }

      return successCount === results.length;
    } catch (error) {
      console.error('[DNS Manager] Failed to update DNS records:', error.message);
      return false;
    }
  }

  /**
   * Update configuration files with current deployment mode
   */
  async updateConfigFiles(mode) {
    const configPath = path.join(__dirname, '..', 'config', 'portal-links.yaml');
    const networkConfigPath = path.join(__dirname, '..', 'network-website', 'public', 'config', 'portal-links.yaml');
    
    console.log(`[DNS Manager] Updating configuration files for ${mode} mode`);
    
    // Update deployment mode in environment
    const envPath = path.join(__dirname, '..', '.env.testnet');
    if (fs.existsSync(envPath)) {
      let envContent = fs.readFileSync(envPath, 'utf8');
      
      // Add or update deployment mode
      if (envContent.includes('DEPLOYMENT_MODE=')) {
        envContent = envContent.replace(/DEPLOYMENT_MODE=.*/g, `DEPLOYMENT_MODE=${mode}`);
      } else {
        envContent += `\nDEPLOYMENT_MODE=${mode}\n`;
      }
      
      fs.writeFileSync(envPath, envContent);
    }
    
    console.log(`[DNS Manager] ✅ Configuration updated for ${mode} mode`);
  }

  /**
   * Verify service health after DNS update
   */
  async verifyServiceHealth(serviceName, serviceConfig) {
    const baseUrl = this.isPrivateTestnet 
      ? `http://localhost:${serviceConfig.port}`
      : `https://${serviceConfig.domain}`;
    
    try {
      const healthUrl = `${baseUrl}/health`;
      const response = await axios.get(healthUrl, { timeout: 5000 });
      
      if (response.status === 200) {
        console.log(`[DNS Manager] ✅ ${serviceName} health check passed`);
        return true;
      }
    } catch (error) {
      console.log(`[DNS Manager] ⚠️  ${serviceName} health check failed: ${error.message}`);
    }
    
    return false;
  }

  /**
   * Run full DNS update and verification
   */
  async run() {
    console.log('[DNS Manager] Starting gateway DNS management...');
    
    try {
      const success = await this.updateAllServiceDNS();
      
      if (success) {
        console.log('[DNS Manager] ✅ DNS update completed successfully');
        
        // Wait a moment for DNS propagation
        if (!this.isPrivateTestnet) {
          console.log('[DNS Manager] Waiting for DNS propagation...');
          await new Promise(resolve => setTimeout(resolve, 5000));
        }
        
        // Verify service health
        console.log('[DNS Manager] Verifying service health...');
        for (const [serviceName, serviceConfig] of Object.entries(GATEWAY_SERVICES)) {
          await this.verifyServiceHealth(serviceName, serviceConfig);
        }
      } else {
        console.log('[DNS Manager] ❌ DNS update failed');
        process.exit(1);
      }
    } catch (error) {
      console.error('[DNS Manager] ❌ DNS management failed:', error.message);
      process.exit(1);
    }
  }
}

// Run if called directly
if (require.main === module) {
  const manager = new GatewayDNSManager();
  manager.run();
}

module.exports = GatewayDNSManager;
