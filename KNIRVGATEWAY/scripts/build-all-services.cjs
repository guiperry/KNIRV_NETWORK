#!/usr/bin/env node

/**
 * KNIRV Gateway Services Build Script
 * 
 * Builds all gateway services in the correct order and handles dependencies.
 * Integrates with the gateway links system for automatic configuration.
 */

const { spawn } = require('child_process');
const fs = require('fs');
const path = require('path');

class ServiceBuilder {
  constructor() {
    this.gatewayDir = path.join(__dirname, '..');
    this.services = [
      {
        name: 'payment-gateway',
        path: 'services/payment-gateway',
        buildRequired: false,
        description: 'Payment Gateway Service'
      },
      {
        name: 'tunnel-registry',
        path: 'services/tunnel-registry',
        buildRequired: false,
        description: 'Tunnel Registry Service'
      },
      {
        name: 'operator-registry',
        path: 'services/operator-registry',
        buildRequired: true,
        description: 'Operator Registry Service (Next.js)'
      },
      {
        name: 'webgui',
        path: 'services/webgui',
        buildRequired: true,
        description: 'Web GUI Service (Next.js)'
      }
    ];
    this.results = [];
  }

  /**
   * Run a command and return a promise
   */
  runCommand(command, args, options = {}) {
    return new Promise((resolve, reject) => {
      console.log(`[Build] Running: ${command} ${args.join(' ')}`);
      
      const process = spawn(command, args, {
        stdio: 'inherit',
        cwd: options.cwd || this.gatewayDir,
        ...options
      });

      process.on('close', (code) => {
        if (code === 0) {
          resolve(code);
        } else {
          reject(new Error(`Command failed with exit code ${code}`));
        }
      });

      process.on('error', (error) => {
        reject(error);
      });
    });
  }

  /**
   * Install dependencies for a service
   */
  async installService(service) {
    const servicePath = path.join(this.gatewayDir, service.path);
    
    if (!fs.existsSync(servicePath)) {
      throw new Error(`Service directory not found: ${servicePath}`);
    }

    if (!fs.existsSync(path.join(servicePath, 'package.json'))) {
      console.log(`[Build] ⚠️  No package.json found for ${service.name}, skipping install`);
      return true;
    }

    try {
      console.log(`[Build] 📦 Installing dependencies for ${service.description}...`);
      await this.runCommand('npm', ['install'], { cwd: servicePath });
      console.log(`[Build] ✅ Dependencies installed for ${service.name}`);
      return true;
    } catch (error) {
      console.error(`[Build] ❌ Failed to install dependencies for ${service.name}:`, error.message);
      return false;
    }
  }

  /**
   * Build a service if required
   */
  async buildService(service) {
    if (!service.buildRequired) {
      console.log(`[Build] ⏭️  ${service.description}: No build step required`);
      return true;
    }

    const servicePath = path.join(this.gatewayDir, service.path);

    try {
      console.log(`[Build] 🔨 Building ${service.description}...`);
      await this.runCommand('npm', ['run', 'build'], { cwd: servicePath });
      console.log(`[Build] ✅ Build completed for ${service.name}`);
      return true;
    } catch (error) {
      console.error(`[Build] ❌ Failed to build ${service.name}:`, error.message);
      return false;
    }
  }

  /**
   * Verify service health after build
   */
  async verifyService(service) {
    const servicePath = path.join(this.gatewayDir, service.path);
    
    // Check if build artifacts exist for services that require building
    if (service.buildRequired) {
      const buildDir = path.join(servicePath, '.next');
      if (!fs.existsSync(buildDir)) {
        console.log(`[Build] ⚠️  Build directory not found for ${service.name}`);
        return false;
      }
    }

    // Check if package.json exists
    const packageJsonPath = path.join(servicePath, 'package.json');
    if (!fs.existsSync(packageJsonPath)) {
      console.log(`[Build] ⚠️  package.json not found for ${service.name}`);
      return false;
    }

    console.log(`[Build] ✅ ${service.name} verification passed`);
    return true;
  }

  /**
   * Build all services
   */
  async buildAll() {
    console.log('=== KNIRV Gateway Services Build ===');
    console.log(`Building ${this.services.length} services...`);

    for (const service of this.services) {
      const result = {
        service: service.name,
        description: service.description,
        installSuccess: false,
        buildSuccess: false,
        verifySuccess: false,
        error: null
      };

      try {
        // Step 1: Install dependencies
        result.installSuccess = await this.installService(service);
        
        if (!result.installSuccess) {
          result.error = 'Failed to install dependencies';
          this.results.push(result);
          continue;
        }

        // Step 2: Build if required
        result.buildSuccess = await this.buildService(service);
        
        if (!result.buildSuccess) {
          result.error = 'Failed to build service';
          this.results.push(result);
          continue;
        }

        // Step 3: Verify
        result.verifySuccess = await this.verifyService(service);
        
        if (!result.verifySuccess) {
          result.error = 'Failed verification';
        }

      } catch (error) {
        result.error = error.message;
        console.error(`[Build] ❌ Error building ${service.name}:`, error.message);
      }

      this.results.push(result);
    }

    return this.generateSummary();
  }

  /**
   * Generate build summary
   */
  generateSummary() {
    const successful = this.results.filter(r => r.installSuccess && r.buildSuccess && r.verifySuccess);
    const failed = this.results.filter(r => !r.installSuccess || !r.buildSuccess || !r.verifySuccess);

    console.log('\n=== Build Summary ===');
    console.log(`✅ Successful: ${successful.length}/${this.results.length}`);
    console.log(`❌ Failed: ${failed.length}/${this.results.length}`);

    if (successful.length > 0) {
      console.log('\n✅ Successfully built services:');
      successful.forEach(result => {
        console.log(`  - ${result.service}: ${result.description}`);
      });
    }

    if (failed.length > 0) {
      console.log('\n❌ Failed services:');
      failed.forEach(result => {
        console.log(`  - ${result.service}: ${result.error}`);
      });
    }

    console.log('=====================\n');

    const summary = {
      total: this.results.length,
      successful: successful.length,
      failed: failed.length,
      results: this.results,
      timestamp: new Date().toISOString()
    };

    // Save summary to file
    const summaryPath = path.join(this.gatewayDir, 'logs', 'build-summary.json');
    try {
      // Ensure logs directory exists
      const logsDir = path.dirname(summaryPath);
      if (!fs.existsSync(logsDir)) {
        fs.mkdirSync(logsDir, { recursive: true });
      }
      
      fs.writeFileSync(summaryPath, JSON.stringify(summary, null, 2));
      console.log(`[Build] 📄 Build summary saved to ${summaryPath}`);
    } catch (error) {
      console.error('[Build] Failed to save build summary:', error.message);
    }

    return summary;
  }

  /**
   * Clean all services
   */
  async cleanAll() {
    console.log('=== Cleaning All Services ===');
    
    for (const service of this.services) {
      const servicePath = path.join(this.gatewayDir, service.path);
      
      try {
        console.log(`[Clean] 🧹 Cleaning ${service.description}...`);
        
        // Remove node_modules
        const nodeModulesPath = path.join(servicePath, 'node_modules');
        if (fs.existsSync(nodeModulesPath)) {
          await this.runCommand('rm', ['-rf', 'node_modules'], { cwd: servicePath });
        }
        
        // Remove build artifacts
        if (service.buildRequired) {
          const buildPaths = ['.next', 'out', 'dist'];
          for (const buildPath of buildPaths) {
            const fullPath = path.join(servicePath, buildPath);
            if (fs.existsSync(fullPath)) {
              await this.runCommand('rm', ['-rf', buildPath], { cwd: servicePath });
            }
          }
        }
        
        console.log(`[Clean] ✅ Cleaned ${service.name}`);
      } catch (error) {
        console.error(`[Clean] ❌ Failed to clean ${service.name}:`, error.message);
      }
    }
    
    console.log('=== Cleaning Complete ===\n');
  }
}

// CLI interface
if (require.main === module) {
  const builder = new ServiceBuilder();
  
  const command = process.argv[2];
  
  switch (command) {
    case 'clean':
      builder.cleanAll()
        .then(() => process.exit(0))
        .catch(error => {
          console.error('Clean failed:', error.message);
          process.exit(1);
        });
      break;
      
    case 'build':
    default:
      builder.buildAll()
        .then(summary => {
          if (summary.failed > 0) {
            console.error(`Build completed with ${summary.failed} failures`);
            process.exit(1);
          } else {
            console.log('All services built successfully!');
            process.exit(0);
          }
        })
        .catch(error => {
          console.error('Build failed:', error.message);
          process.exit(1);
        });
  }
}

module.exports = ServiceBuilder;
