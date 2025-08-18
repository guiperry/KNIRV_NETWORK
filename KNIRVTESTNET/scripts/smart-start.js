#!/usr/bin/env node

/**
 * KNIRVTESTNET Smart Start Script
 * 
 * Intelligently starts the KNIRVTESTNET server with proper initialization
 */

const { spawn } = require('child_process');
const fs = require('fs');
const path = require('path');

async function smartStart() {
  console.log('🚀 KNIRVTESTNET Smart Start');
  console.log('===========================');
  
  // Check if we're in the right directory
  const packageJsonPath = path.join(__dirname, '..', 'package.json');
  if (!fs.existsSync(packageJsonPath)) {
    console.error('❌ package.json not found. Make sure you\'re in the KNIRVTESTNET directory.');
    process.exit(1);
  }
  
  // Load package.json to verify this is KNIRVTESTNET
  try {
    const packageJson = require(packageJsonPath);
    if (packageJson.name !== 'knirvtestnet') {
      console.warn('⚠️  Package name is not "knirvtestnet". Continuing anyway...');
    }
    console.log(`📦 Starting ${packageJson.name} v${packageJson.version}`);
  } catch (error) {
    console.error('❌ Failed to read package.json:', error.message);
    process.exit(1);
  }
  
  // Check and fix axios corruption first
  console.log('\n🔧 Checking axios installation...');
  try {
    require('axios');
    console.log('✅ Axios is working correctly');
  } catch (error) {
    if (error.code === 'MODULE_NOT_FOUND' && error.message.includes('axios')) {
      console.log('❌ Axios corruption detected, running fix...');
      const { execSync } = require('child_process');
      try {
        execSync('./scripts/fix-axios-corruption.sh', {
          stdio: 'inherit',
          cwd: path.join(__dirname, '..')
        });
        console.log('✅ Axios fix completed');
      } catch (fixError) {
        console.error('❌ Axios fix failed:', fixError.message);
        console.log('Attempting manual axios downgrade...');
        try {
          execSync('npm install axios@1.6.8 --save-exact', {
            stdio: 'inherit',
            cwd: path.join(__dirname, '..')
          });
          console.log('✅ Axios downgraded successfully');
        } catch (downgradeError) {
          console.error('❌ Manual axios fix failed:', downgradeError.message);
          process.exit(1);
        }
      }
    } else {
      console.warn('⚠️  Axios check failed with unexpected error:', error.message);
    }
  }

  // Run health check
  console.log('\n🏥 Running health check...');
  try {
    const { checkHealth } = require('./check-health');
    const healthy = checkHealth();
    if (!healthy) {
      console.error('\n❌ Health check failed. Please fix issues before starting.');
      process.exit(1);
    }
  } catch (error) {
    console.warn('⚠️  Health check failed to run:', error.message);
    console.log('Continuing with startup...');
  }
  
  // Load endpoints
  console.log('\n🔧 Loading endpoints...');
  try {
    const { loadEndpoints } = require('./load-endpoints');
    const { endpoints, config } = loadEndpoints('testnet');
    console.log(`✅ Loaded ${Object.keys(endpoints).length} endpoints`);
    console.log(`✅ Environment: ${config.DEPLOYMENT_ENV}`);
  } catch (error) {
    console.error('❌ Failed to load endpoints:', error.message);
    process.exit(1);
  }
  
  // Check if dependencies are installed
  const nodeModulesPath = path.join(__dirname, '..', 'node_modules');
  if (!fs.existsSync(nodeModulesPath)) {
    console.log('\n📦 Installing dependencies...');
    const npmInstall = spawn('npm', ['install'], {
      cwd: path.join(__dirname, '..'),
      stdio: 'inherit'
    });
    
    await new Promise((resolve, reject) => {
      npmInstall.on('close', (code) => {
        if (code === 0) {
          console.log('✅ Dependencies installed successfully');
          resolve();
        } else {
          console.error('❌ Failed to install dependencies');
          reject(new Error(`npm install failed with code ${code}`));
        }
      });
    });
  }
  
  // Start the server
  console.log('\n🌐 Starting KNIRVTESTNET server...');
  const serverPath = path.join(__dirname, '..', 'server', 'app.js');
  
  if (!fs.existsSync(serverPath)) {
    console.error('❌ Server file not found:', serverPath);
    process.exit(1);
  }
  
  // Set environment variables
  process.env.NODE_ENV = process.env.NODE_ENV || 'testnet';
  process.env.PORT = process.env.PORT || '10000';
  
  console.log(`🔧 Environment: ${process.env.NODE_ENV}`);
  console.log(`🔧 Port: ${process.env.PORT}`);
  
  // Start the server process
  const server = spawn('node', [serverPath], {
    stdio: 'inherit',
    env: process.env
  });
  
  server.on('close', (code) => {
    if (code === 0) {
      console.log('\n✅ Server stopped gracefully');
    } else {
      console.error(`\n❌ Server stopped with code ${code}`);
    }
    process.exit(code);
  });
  
  server.on('error', (error) => {
    console.error('\n❌ Failed to start server:', error.message);
    process.exit(1);
  });
  
  // Handle graceful shutdown
  process.on('SIGINT', () => {
    console.log('\n🛑 Received SIGINT, shutting down gracefully...');
    server.kill('SIGINT');
  });
  
  process.on('SIGTERM', () => {
    console.log('\n🛑 Received SIGTERM, shutting down gracefully...');
    server.kill('SIGTERM');
  });
  
  console.log('\n🎉 KNIRVTESTNET server started successfully!');
  console.log(`🌐 Server should be available at http://localhost:${process.env.PORT}`);
  console.log('📊 Health monitor: http://localhost:' + process.env.PORT + '/health-monitor');
  console.log('🔍 API health: http://localhost:' + process.env.PORT + '/health');
}

// Run if called directly
if (require.main === module) {
  smartStart().catch(error => {
    console.error('❌ Smart start failed:', error.message);
    process.exit(1);
  });
}

module.exports = { smartStart };
