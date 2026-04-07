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
  
  // Check and fix axios corruption first (with recursion protection)
  console.log('\n🔧 Checking axios installation...');

  // Check if we're already in an axios fix process to prevent recursion
  if (process.env.KNIRV_AXIOS_FIX_IN_PROGRESS === 'true') {
    console.log('⚠️  Axios fix already in progress, skipping to prevent recursion');
  } else {
    try {
      require('axios');
      console.log('✅ Axios is working correctly');
    } catch (error) {
      if (error.code === 'MODULE_NOT_FOUND' && error.message.includes('axios')) {
        console.log('❌ Axios corruption detected, running fix...');

        // Set environment variable to prevent recursion
        process.env.KNIRV_AXIOS_FIX_IN_PROGRESS = 'true';

        const { execSync } = require('child_process');
        try {
          execSync('./scripts/fix-axios-corruption.sh', {
            stdio: 'inherit',
            cwd: path.join(__dirname, '..'),
            env: { ...process.env, KNIRV_AXIOS_FIX_IN_PROGRESS: 'true' }
          });
          console.log('✅ Axios fix completed');

          // Clear the flag after successful fix
          delete process.env.KNIRV_AXIOS_FIX_IN_PROGRESS;
        } catch (fixError) {
          console.error('❌ Axios fix failed:', fixError.message);
          console.log('Attempting manual axios downgrade...');
          try {
            execSync('npm install axios@1.6.8 --save-exact', {
              stdio: 'inherit',
              cwd: path.join(__dirname, '..'),
              env: { ...process.env, KNIRV_AXIOS_FIX_IN_PROGRESS: 'true' }
            });
            console.log('✅ Axios downgraded successfully');

            // Clear the flag after successful fix
            delete process.env.KNIRV_AXIOS_FIX_IN_PROGRESS;
          } catch (downgradeError) {
            console.error('❌ Manual axios fix failed:', downgradeError.message);
            delete process.env.KNIRV_AXIOS_FIX_IN_PROGRESS;
            process.exit(1);
          }
        }
      } else {
        console.warn('⚠️  Axios check failed with unexpected error:', error.message);
      }
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
  
  // Initialization complete - do not start server here
  console.log('\n✅ Smart initialization completed successfully!');
  console.log('🔧 Environment variables set:');
  console.log(`   NODE_ENV: ${process.env.NODE_ENV || 'testnet'}`);
  console.log(`   PORT: ${process.env.PORT || '10000'}`);
  console.log('\n📝 Note: Server startup should be handled by the calling script');

  return {
    success: true,
    environment: process.env.NODE_ENV || 'testnet',
    port: process.env.PORT || '10000'
  };
}

// Smart initialization only (no server startup)
async function smartInit() {
  console.log('🚀 KNIRVTESTNET Smart Initialization');
  console.log('====================================');

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
    console.log(`📦 Initializing ${packageJson.name} v${packageJson.version}`);
  } catch (error) {
    console.error('❌ Failed to read package.json:', error.message);
    process.exit(1);
  }

  // Check and fix axios corruption first (with recursion protection)
  console.log('\n🔧 Checking axios installation...');

  // Check if we're already in an axios fix process to prevent recursion
  if (process.env.KNIRV_AXIOS_FIX_IN_PROGRESS === 'true') {
    console.log('⚠️  Axios fix already in progress, skipping to prevent recursion');
  } else {
    try {
      require('axios');
      console.log('✅ Axios is working correctly');
    } catch (error) {
      if (error.code === 'MODULE_NOT_FOUND' && error.message.includes('axios')) {
        console.log('❌ Axios corruption detected, running fix...');

        // Set environment variable to prevent recursion
        process.env.KNIRV_AXIOS_FIX_IN_PROGRESS = 'true';

        const { execSync } = require('child_process');
        try {
          execSync('./scripts/fix-axios-corruption.sh', {
            stdio: 'inherit',
            cwd: path.join(__dirname, '..'),
            env: { ...process.env, KNIRV_AXIOS_FIX_IN_PROGRESS: 'true' }
          });
          console.log('✅ Axios fix completed');

          // Clear the flag after successful fix
          delete process.env.KNIRV_AXIOS_FIX_IN_PROGRESS;
        } catch (fixError) {
          console.error('❌ Axios fix failed:', fixError.message);
          console.log('Attempting manual axios downgrade...');
          try {
            execSync('npm install axios@1.6.8 --save-exact', {
              stdio: 'inherit',
              cwd: path.join(__dirname, '..'),
              env: { ...process.env, KNIRV_AXIOS_FIX_IN_PROGRESS: 'true' }
            });
            console.log('✅ Axios downgraded successfully');

            // Clear the flag after successful fix
            delete process.env.KNIRV_AXIOS_FIX_IN_PROGRESS;
          } catch (downgradeError) {
            console.error('❌ Manual axios fix failed:', downgradeError.message);
            delete process.env.KNIRV_AXIOS_FIX_IN_PROGRESS;
            process.exit(1);
          }
        }
      } else {
        console.warn('⚠️  Axios check failed with unexpected error:', error.message);
      }
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

  // Set environment variables for the calling process
  process.env.NODE_ENV = process.env.NODE_ENV || 'testnet';
  process.env.PORT = process.env.PORT || '10000';

  console.log('\n✅ Smart initialization completed successfully!');
  console.log('🔧 Environment variables set:');
  console.log(`   NODE_ENV: ${process.env.NODE_ENV}`);
  console.log(`   PORT: ${process.env.PORT}`);

  return {
    success: true,
    environment: process.env.NODE_ENV,
    port: process.env.PORT
  };
}

// Run if called directly
if (require.main === module) {
  // Check if we should run initialization only or full startup
  const args = process.argv.slice(2);
  if (args.includes('--init-only')) {
    smartInit().catch(error => {
      console.error('❌ Smart initialization failed:', error.message);
      process.exit(1);
    });
  } else {
    smartStart().catch(error => {
      console.error('❌ Smart start failed:', error.message);
      process.exit(1);
    });
  }
}

module.exports = { smartStart, smartInit };
