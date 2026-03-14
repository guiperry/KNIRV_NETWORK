#!/usr/bin/env node

/**
 * Netlify Dependencies Patcher
 * 
 * This script patches netlify function dependencies to ensure compatibility
 * and resolves common issues with Netlify Functions deployment.
 */

const fs = require('fs');
const path = require('path');

class NetlifyDepsPatcher {
  constructor() {
    this.functionsDir = path.join(__dirname, '..', 'netlify', 'functions');
    this.packageJsonPath = path.join(this.functionsDir, 'package.json');
    this.nodeModulesPath = path.join(this.functionsDir, 'node_modules');
  }

  /**
   * Log with timestamp
   */
  log(message, type = 'info') {
    const timestamp = new Date().toISOString();
    const prefix = {
      info: '🔧',
      success: '✅',
      warn: '⚠️',
      error: '❌'
    }[type] || 'ℹ️';
    
    console.log(`${prefix} [${timestamp}] ${message}`);
  }

  /**
   * Check if package.json exists
   */
  checkPackageJson() {
    if (!fs.existsSync(this.packageJsonPath)) {
      this.log('package.json not found in netlify/functions directory', 'error');
      return false;
    }
    
    this.log('package.json found', 'success');
    return true;
  }

  /**
   * Patch package.json for compatibility
   */
  patchPackageJson() {
    try {
      const packageJson = JSON.parse(fs.readFileSync(this.packageJsonPath, 'utf8'));
      
      // Ensure proper engine requirements
      if (!packageJson.engines) {
        packageJson.engines = {};
      }
      
      packageJson.engines.node = '>=18.0.0';
      
      // Add netlify-specific configurations
      if (!packageJson.netlify) {
        packageJson.netlify = {
          functions: {
            directory: ".",
            node_bundler: "esbuild"
          }
        };
      }

      // Ensure proper main entry
      if (!packageJson.main) {
        packageJson.main = 'index.js';
      }

      // Write back the patched package.json
      fs.writeFileSync(this.packageJsonPath, JSON.stringify(packageJson, null, 2));
      this.log('package.json patched successfully', 'success');
      return true;
    } catch (error) {
      this.log(`Failed to patch package.json: ${error.message}`, 'error');
      return false;
    }
  }

  /**
   * Create necessary directories
   */
  createDirectories() {
    const dirs = [
      this.functionsDir,
      path.join(this.functionsDir, 'lib'),
      path.join(this.functionsDir, 'utils')
    ];

    for (const dir of dirs) {
      if (!fs.existsSync(dir)) {
        fs.mkdirSync(dir, { recursive: true });
        this.log(`Created directory: ${path.relative(process.cwd(), dir)}`, 'success');
      }
    }
  }

  /**
   * Check for common dependency issues
   */
  checkDependencyIssues() {
    try {
      const packageJson = JSON.parse(fs.readFileSync(this.packageJsonPath, 'utf8'));
      const issues = [];

      // Check for problematic dependencies
      const problematicDeps = {
        'sharp': 'Image processing library that may have platform-specific builds',
        'bcryptjs': 'Crypto library that should work in serverless environments',
        'jsdom': 'DOM implementation that may be heavy for serverless'
      };

      if (packageJson.dependencies) {
        for (const [dep, issue] of Object.entries(problematicDeps)) {
          if (packageJson.dependencies[dep]) {
            this.log(`Dependency check - ${dep}: ${issue}`, 'warn');
          }
        }
      }

      return true;
    } catch (error) {
      this.log(`Failed to check dependencies: ${error.message}`, 'error');
      return false;
    }
  }

  /**
   * Create a simple health check function if none exists
   */
  createHealthCheckFunction() {
    const healthCheckPath = path.join(this.functionsDir, 'health.js');
    
    if (!fs.existsSync(healthCheckPath)) {
      const healthCheckContent = `// Netlify Function: Health Check
exports.handler = async (event, context) => {
  return {
    statusCode: 200,
    headers: {
      'Content-Type': 'application/json',
      'Access-Control-Allow-Origin': '*',
      'Access-Control-Allow-Headers': 'Content-Type',
      'Access-Control-Allow-Methods': 'GET, POST, OPTIONS'
    },
    body: JSON.stringify({
      status: 'healthy',
      timestamp: new Date().toISOString(),
      environment: process.env.NODE_ENV || 'development',
      netlify: {
        context: context.clientContext || 'unknown',
        functionName: context.functionName || 'health'
      }
    })
  };
};`;

      fs.writeFileSync(healthCheckPath, healthCheckContent);
      this.log('Created health check function', 'success');
    }
  }

  /**
   * Run all patches
   */
  async runPatches() {
    this.log('Starting Netlify dependencies patching...', 'info');

    // Create necessary directories
    this.createDirectories();

    // Check and patch package.json
    if (!this.checkPackageJson()) {
      return false;
    }

    if (!this.patchPackageJson()) {
      return false;
    }

    // Check for dependency issues
    this.checkDependencyIssues();

    // Create health check function
    this.createHealthCheckFunction();

    this.log('Netlify dependencies patching completed successfully!', 'success');
    return true;
  }
}

// CLI interface
if (require.main === module) {
  const patcher = new NetlifyDepsPatcher();
  
  patcher.runPatches()
    .then(success => {
      process.exit(success ? 0 : 1);
    })
    .catch(error => {
      console.error('❌ Patching failed:', error.message);
      process.exit(1);
    });
}

module.exports = NetlifyDepsPatcher;
