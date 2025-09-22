#!/usr/bin/env node

/**
 * CloudFlare CDN Deployment Script for KNIRVCONTROLLER
 * Deploys PWA to beta-controller.knirv.network with automatic DNS updates
 */

import fs from 'fs';
import path from 'path';
import { execSync } from 'child_process';
import fetch from 'node-fetch';

const CLOUDFLARE_CONFIG = {
  zoneId: process.env.CLOUDFLARE_ZONE_ID,
  apiToken: process.env.CLOUDFLARE_API_TOKEN,
  domain: 'knirv.network',
  subdomain: 'beta-controller',
  fullDomain: 'beta-controller.knirv.network'
};

const DEPLOYMENT_CONFIG = {
  buildDir: 'dist',
  pwaDir: 'dist-pwa',
  packagesDir: 'packages',
  deploymentBranch: 'main',
  environment: process.env.NODE_ENV || 'production'
};

class CloudFlareDeployer {
  constructor() {
    this.validateEnvironment();
  }

  validateEnvironment() {
    if (!CLOUDFLARE_CONFIG.zoneId) {
      throw new Error('CLOUDFLARE_ZONE_ID environment variable is required');
    }
    if (!CLOUDFLARE_CONFIG.apiToken) {
      throw new Error('CLOUDFLARE_API_TOKEN environment variable is required');
    }
  }

  async deploy() {
    console.log('🚀 Starting CloudFlare deployment...');
    console.log(`📍 Target: ${CLOUDFLARE_CONFIG.fullDomain}`);
    
    try {
      // Build the application
      await this.buildApplication();
      
      // Build PWA packages
      await this.buildPWAPackages();
      
      // Deploy to CloudFlare Pages
      await this.deployToCloudFlarePages();
      
      // Update DNS records
      await this.updateDNSRecords();
      
      // Verify deployment
      await this.verifyDeployment();
      
      console.log('✅ Deployment completed successfully!');
      console.log(`🌐 Application available at: https://${CLOUDFLARE_CONFIG.fullDomain}`);
      
    } catch (error) {
      console.error('❌ Deployment failed:', error);
      throw error;
    }
  }

  async buildApplication() {
    console.log('🔨 Building application...');
    
    try {
      // Clean previous builds
      if (fs.existsSync(DEPLOYMENT_CONFIG.buildDir)) {
        fs.rmSync(DEPLOYMENT_CONFIG.buildDir, { recursive: true, force: true });
      }
      
      // Build the main application
      execSync('npm run build', { stdio: 'inherit' });
      
      console.log('✅ Application built successfully');
    } catch (error) {
      console.error('❌ Application build failed:', error);
      throw error;
    }
  }

  async buildPWAPackages() {
    console.log('📱 Building PWA packages...');
    
    try {
      // Build PWA packages
      execSync('npm run build:pwa', { stdio: 'inherit' });
      
      console.log('✅ PWA packages built successfully');
    } catch (error) {
      console.error('❌ PWA build failed:', error);
      throw error;
    }
  }

  async deployToCloudFlarePages() {
    console.log('☁️ Deploying to CloudFlare Pages...');
    
    try {
      // Check if wrangler is available
      try {
        execSync('npx wrangler --version', { stdio: 'pipe' });
      } catch {
        console.log('📦 Installing Wrangler...');
        execSync('npm install -g wrangler', { stdio: 'inherit' });
      }
      
      // Deploy using Wrangler
      const deployCommand = `npx wrangler pages deploy ${DEPLOYMENT_CONFIG.buildDir} --project-name=knirvcontroller --compatibility-date=2024-01-01`;
      
      console.log('🚀 Deploying with Wrangler...');
      execSync(deployCommand, { stdio: 'inherit' });
      
      console.log('✅ CloudFlare Pages deployment completed');
    } catch (error) {
      console.error('❌ CloudFlare Pages deployment failed:', error);
      throw error;
    }
  }

  async updateDNSRecords() {
    console.log('🌐 Updating DNS records...');
    
    try {
      // Get current public IP
      const publicIP = await this.getPublicIP();
      console.log(`📍 Public IP: ${publicIP}`);
      
      // Update A record for the subdomain
      await this.updateDNSRecord({
        type: 'A',
        name: CLOUDFLARE_CONFIG.subdomain,
        content: publicIP,
        ttl: 300,
        proxied: true
      });
      
      // Update CNAME record for PWA downloads
      await this.updateDNSRecord({
        type: 'CNAME',
        name: `pwa.${CLOUDFLARE_CONFIG.subdomain}`,
        content: CLOUDFLARE_CONFIG.fullDomain,
        ttl: 300,
        proxied: false
      });
      
      console.log('✅ DNS records updated successfully');
    } catch (error) {
      console.error('❌ DNS update failed:', error);
      throw error;
    }
  }

  async getPublicIP() {
    try {
      const response = await fetch('https://api.ipify.org?format=json');
      const data = await response.json();
      return data.ip;
    } catch (error) {
      console.warn('Failed to get public IP, using fallback');
      return '127.0.0.1'; // Fallback for local development
    }
  }

  async updateDNSRecord(record) {
    const url = `https://api.cloudflare.com/client/v4/zones/${CLOUDFLARE_CONFIG.zoneId}/dns_records`;
    
    try {
      // First, check if record exists
      const existingRecords = await this.getDNSRecords(record.name);
      
      if (existingRecords.length > 0) {
        // Update existing record
        const recordId = existingRecords[0].id;
        await this.updateExistingDNSRecord(recordId, record);
      } else {
        // Create new record
        await this.createDNSRecord(record);
      }
    } catch (error) {
      console.error(`Failed to update DNS record for ${record.name}:`, error);
      throw error;
    }
  }

  async getDNSRecords(name) {
    const url = `https://api.cloudflare.com/client/v4/zones/${CLOUDFLARE_CONFIG.zoneId}/dns_records?name=${name}.${CLOUDFLARE_CONFIG.domain}`;
    
    const response = await fetch(url, {
      headers: {
        'Authorization': `Bearer ${CLOUDFLARE_CONFIG.apiToken}`,
        'Content-Type': 'application/json'
      }
    });
    
    if (!response.ok) {
      throw new Error(`CloudFlare API error: ${response.statusText}`);
    }
    
    const data = await response.json();
    return data.result || [];
  }

  async createDNSRecord(record) {
    const url = `https://api.cloudflare.com/client/v4/zones/${CLOUDFLARE_CONFIG.zoneId}/dns_records`;
    
    const response = await fetch(url, {
      method: 'POST',
      headers: {
        'Authorization': `Bearer ${CLOUDFLARE_CONFIG.apiToken}`,
        'Content-Type': 'application/json'
      },
      body: JSON.stringify({
        type: record.type,
        name: `${record.name}.${CLOUDFLARE_CONFIG.domain}`,
        content: record.content,
        ttl: record.ttl,
        proxied: record.proxied
      })
    });
    
    if (!response.ok) {
      const error = await response.text();
      throw new Error(`Failed to create DNS record: ${error}`);
    }
    
    console.log(`✅ Created DNS record: ${record.name}.${CLOUDFLARE_CONFIG.domain} -> ${record.content}`);
  }

  async updateExistingDNSRecord(recordId, record) {
    const url = `https://api.cloudflare.com/client/v4/zones/${CLOUDFLARE_CONFIG.zoneId}/dns_records/${recordId}`;
    
    const response = await fetch(url, {
      method: 'PUT',
      headers: {
        'Authorization': `Bearer ${CLOUDFLARE_CONFIG.apiToken}`,
        'Content-Type': 'application/json'
      },
      body: JSON.stringify({
        type: record.type,
        name: `${record.name}.${CLOUDFLARE_CONFIG.domain}`,
        content: record.content,
        ttl: record.ttl,
        proxied: record.proxied
      })
    });
    
    if (!response.ok) {
      const error = await response.text();
      throw new Error(`Failed to update DNS record: ${error}`);
    }
    
    console.log(`✅ Updated DNS record: ${record.name}.${CLOUDFLARE_CONFIG.domain} -> ${record.content}`);
  }

  async verifyDeployment() {
    console.log('🔍 Verifying deployment...');
    
    try {
      // Wait a moment for DNS propagation
      await new Promise(resolve => setTimeout(resolve, 5000));
      
      // Test the main application
      const response = await fetch(`https://${CLOUDFLARE_CONFIG.fullDomain}/health`, {
        timeout: 10000
      });
      
      if (response.ok) {
        console.log('✅ Application is responding correctly');
      } else {
        console.warn(`⚠️ Application responded with status: ${response.status}`);
      }
      
      // Test PWA manifest
      const manifestResponse = await fetch(`https://${CLOUDFLARE_CONFIG.fullDomain}/manifest.json`, {
        timeout: 10000
      });
      
      if (manifestResponse.ok) {
        console.log('✅ PWA manifest is accessible');
      } else {
        console.warn(`⚠️ PWA manifest responded with status: ${manifestResponse.status}`);
      }
      
    } catch (error) {
      console.warn('⚠️ Deployment verification failed:', error.message);
      console.log('🔄 The application may still be deploying. Please check manually.');
    }
  }

  async generateDeploymentReport() {
    const report = {
      timestamp: new Date().toISOString(),
      environment: DEPLOYMENT_CONFIG.environment,
      domain: CLOUDFLARE_CONFIG.fullDomain,
      buildDir: DEPLOYMENT_CONFIG.buildDir,
      pwaPackages: [],
      dnsRecords: []
    };

    // Check for PWA packages
    if (fs.existsSync(DEPLOYMENT_CONFIG.packagesDir)) {
      const packages = fs.readdirSync(DEPLOYMENT_CONFIG.packagesDir);
      report.pwaPackages = packages.map(pkg => ({
        name: pkg,
        size: fs.statSync(path.join(DEPLOYMENT_CONFIG.packagesDir, pkg)).size,
        url: `https://${CLOUDFLARE_CONFIG.fullDomain}/packages/${pkg}`
      }));
    }

    // Save report
    const reportPath = path.join(DEPLOYMENT_CONFIG.buildDir, 'deployment-report.json');
    fs.writeFileSync(reportPath, JSON.stringify(report, null, 2));
    
    console.log('📊 Deployment report generated:', reportPath);
    return report;
  }
}

// CLI interface
if (import.meta.url === `file://${process.argv[1]}`) {
  const deployer = new CloudFlareDeployer();
  
  const command = process.argv[2];
  
  switch (command) {
    case 'deploy':
      deployer.deploy()
        .then(() => deployer.generateDeploymentReport())
        .catch(error => {
          console.error('Deployment failed:', error);
          process.exit(1);
        });
      break;
      
    case 'dns-only':
      deployer.updateDNSRecords()
        .catch(error => {
          console.error('DNS update failed:', error);
          process.exit(1);
        });
      break;
      
    case 'verify':
      deployer.verifyDeployment()
        .catch(error => {
          console.error('Verification failed:', error);
          process.exit(1);
        });
      break;
      
    default:
      console.log('Usage: node deploy-cloudflare.js [deploy|dns-only|verify]');
      process.exit(1);
  }
}

export default CloudFlareDeployer;
