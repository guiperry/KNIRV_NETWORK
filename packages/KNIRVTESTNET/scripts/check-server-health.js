#!/usr/bin/env node

/**
 * KNIRVTESTNET Nexus Health Check (Updated for New Architecture)
 *
 * Checks the health of the KNIRV-NEXUS unified binary and its configuration
 * for the new embedded frontend/backend architecture
 */

const fs = require('fs');
const path = require('path');
const { execSync } = require('child_process');
const http = require('http');

/**
 * Check if KNIRV-NEXUS unified binary exists and is up to date
 * @returns {object} - Binary status and rebuild recommendation
 */
function checkUnifiedBinary() {
  const binaryPath = path.join(__dirname, '..', 'bin', 'knirvserver');
  const sourcePath = path.join(__dirname, '..', '..', 'packages', 'KNIRVSERVER');

  console.log('\n🔍 Unified Binary Check');
  console.log('=======================');

  // Check if binary exists
  if (!fs.existsSync(binaryPath)) {
    console.log('❌ KNIRV-NEXUS unified binary not found');
    return { needsRebuild: true, reason: 'Binary missing', binaryExists: false };
  }

  // Check if binary is executable
  try {
    fs.accessSync(binaryPath, fs.constants.X_OK);
    console.log('✅ Binary exists and is executable');
  } catch (error) {
    console.log('⚠️  Binary exists but is not executable');
    return { needsRebuild: true, reason: 'Binary not executable', binaryExists: true };
  }

  // Get binary stats
  const binaryStats = fs.statSync(binaryPath);
  const binarySize = (binaryStats.size / (1024 * 1024)).toFixed(1);
  const binaryAge = new Date() - binaryStats.mtime;
  const ageHours = Math.floor(binaryAge / (1000 * 60 * 60));

  console.log(`📦 Binary size: ${binarySize}MB`);
  console.log(`📅 Binary age: ${ageHours} hours`);

  // Check if source directory exists for comparison
  if (!fs.existsSync(sourcePath)) {
    console.log('⚠️  KNIRVSERVER source directory not found - cannot verify freshness');
    return { needsRebuild: false, reason: 'Source not available for comparison', binaryExists: true };
  }

  // Check if source has been modified more recently than binary
  try {
    const sourceMainGo = path.join(sourcePath, 'main.go');
    const sourcePackageJson = path.join(sourcePath, 'package.json');

    let sourceModified = binaryStats.mtime;

    if (fs.existsSync(sourceMainGo)) {
      const mainGoStats = fs.statSync(sourceMainGo);
      if (mainGoStats.mtime > sourceModified) {
        sourceModified = mainGoStats.mtime;
      }
    }

    if (fs.existsSync(sourcePackageJson)) {
      const packageStats = fs.statSync(sourcePackageJson);
      if (packageStats.mtime > sourceModified) {
        sourceModified = packageStats.mtime;
      }
    }

    if (sourceModified > binaryStats.mtime) {
      console.log('🔄 Source files newer than binary - rebuild recommended');
      return { needsRebuild: true, reason: 'Source files newer', binaryExists: true };
    } else {
      console.log('✅ Binary is up to date with source');
      return { needsRebuild: false, reason: 'Binary up to date', binaryExists: true };
    }

  } catch (error) {
    console.log(`⚠️  Error checking source freshness: ${error.message}`);
    return { needsRebuild: false, reason: 'Cannot verify freshness', binaryExists: true };
  }
}

/**
 * Test if KNIRV-NEXUS service is running and responding
 * @returns {Promise<object>} - Service status
 */
async function testNexusService() {
  const port = 8084;
  const host = 'localhost';

  console.log('\n🌐 Service Health Check');
  console.log('=======================');

  // Check if process is running
  const pidFile = path.join(__dirname, '..', 'data', 'knirvserver.pid');
  let processRunning = false;

  if (fs.existsSync(pidFile)) {
    try {
      const pid = fs.readFileSync(pidFile, 'utf8').trim();
      process.kill(pid, 0); // Test if process exists
      processRunning = true;
      console.log(`✅ Process running (PID: ${pid})`);
    } catch (error) {
      console.log('❌ Process not running (PID file exists but process dead)');
    }
  } else {
    console.log('❌ No PID file found');
  }

  if (!processRunning) {
    return { running: false, healthy: false, reason: 'Process not running' };
  }

  // Test HTTP endpoints
  const endpoints = [
    { path: '/health', name: 'Health Check' },
    { path: '/version', name: 'Version Info' },
    { path: '/', name: 'Frontend' }
  ];

  const results = {};

  for (const endpoint of endpoints) {
    try {
      const response = await makeHttpRequest(host, port, endpoint.path);
      if (response.statusCode === 200) {
        console.log(`✅ ${endpoint.name}: OK`);
        results[endpoint.path] = { status: 'ok', data: response.data };
      } else {
        console.log(`❌ ${endpoint.name}: HTTP ${response.statusCode}`);
        results[endpoint.path] = { status: 'error', code: response.statusCode };
      }
    } catch (error) {
      console.log(`❌ ${endpoint.name}: ${error.message}`);
      results[endpoint.path] = { status: 'error', error: error.message };
    }
  }

  const healthy = results['/health']?.status === 'ok';
  return { running: true, healthy, results };
}

/**
 * Make HTTP request helper
 * @param {string} host - Host to connect to
 * @param {number} port - Port to connect to
 * @param {string} path - Path to request
 * @returns {Promise<object>} - Response object
 */
function makeHttpRequest(host, port, path) {
  return new Promise((resolve, reject) => {
    const options = {
      hostname: host,
      port: port,
      path: path,
      method: 'GET',
      timeout: 5000
    };

    const req = http.request(options, (res) => {
      let data = '';
      res.on('data', (chunk) => {
        data += chunk;
      });
      res.on('end', () => {
        resolve({
          statusCode: res.statusCode,
          headers: res.headers,
          data: data
        });
      });
    });

    req.on('error', (error) => {
      reject(error);
    });

    req.on('timeout', () => {
      req.destroy();
      reject(new Error('Request timeout'));
    });

    req.end();
  });
}

async function checkNexusHealth() {
  console.log('🔍 KNIRV-NEXUS Health Check (New Architecture)');
  console.log('===============================================');

  const issues = [];
  const warnings = [];

  // Check unified binary
  const binaryCheck = checkUnifiedBinary();
  if (!binaryCheck.binaryExists) {
    issues.push('KNIRV-NEXUS unified binary not found - run: npm run build:nexus');
  } else if (binaryCheck.needsRebuild) {
    warnings.push(`Binary rebuild recommended: ${binaryCheck.reason}`);
  } else {
    console.log('✅ Unified binary is up to date');
  }

  // Check configuration files
  const configFiles = [
    { path: 'config/nexus-testnet.yaml', name: 'Testnet Configuration' },
    { path: 'data/knirvserver/config.yaml', name: 'Legacy Configuration' }
  ];

  configFiles.forEach(config => {
    const configPath = path.join(__dirname, '..', config.path);
    if (fs.existsSync(configPath)) {
      console.log(`✅ ${config.name} exists`);
    } else {
      warnings.push(`${config.name} not found at ${config.path}`);
    }
  });

  // Check data directory
  const dataDir = path.join(__dirname, '..', 'data', 'knirvserver');
  if (fs.existsSync(dataDir)) {
    console.log('✅ Data directory exists');
  } else {
    warnings.push('Data directory not found - will be created on startup');
  }

  // Check logs directory
  const logsDir = path.join(__dirname, '..', 'logs');
  if (fs.existsSync(logsDir)) {
    console.log('✅ Logs directory exists');
  } else {
    warnings.push('Logs directory not found - will be created on startup');
  }

  // Test service if it's supposed to be running
  let serviceStatus = null;
  try {
    serviceStatus = await testNexusService();
    if (serviceStatus.running && serviceStatus.healthy) {
      console.log('✅ Service is running and healthy');
    } else if (serviceStatus.running && !serviceStatus.healthy) {
      warnings.push('Service is running but not responding properly');
    } else {
      console.log('ℹ️  Service is not currently running (this is normal if not started)');
    }
  } catch (error) {
    warnings.push(`Service health check failed: ${error.message}`);
  }

  // Check KNIRVSERVER source availability
  const sourcePath = path.join(__dirname, '..', '..', 'packages', 'KNIRVSERVER');
  if (fs.existsSync(sourcePath)) {
    console.log('✅ KNIRVSERVER source directory available');

    // Check if source has package.json
    const sourcePackageJson = path.join(sourcePath, 'package.json');
    if (fs.existsSync(sourcePackageJson)) {
      console.log('✅ KNIRVSERVER source has package.json');
    } else {
      warnings.push('KNIRVSERVER source missing package.json');
    }

    // Check if source has main.go
    const sourceMainGo = path.join(sourcePath, 'main.go');
    if (fs.existsSync(sourceMainGo)) {
      console.log('✅ KNIRVSERVER source has main.go');
    } else {
      warnings.push('KNIRVSERVER source missing main.go');
    }
  } else {
    issues.push('KNIRVSERVER source directory not found - cannot build');
  }

  // Summary
  console.log('\n📊 KNIRV-NEXUS Health Summary');
  console.log('==============================');

  if (issues.length === 0 && warnings.length === 0) {
    console.log('🎉 KNIRV-NEXUS health check passed!');
    return true;
  }

  if (warnings.length > 0) {
    console.log('\n⚠️  Warnings:');
    warnings.forEach(warning => console.log(`   - ${warning}`));
  }

  if (issues.length > 0) {
    console.log('\n❌ Issues found:');
    issues.forEach(issue => console.log(`   - ${issue}`));
    return false;
  }

  console.log('\n✅ KNIRV-NEXUS health check completed with warnings only');

  // Provide helpful commands
  console.log('\n🔧 Helpful Commands:');
  console.log('   Build binary:     npm run build:nexus');
  console.log('   Start service:    ./scripts/start-knirvserver.sh');
  console.log('   Check service:    curl http://localhost:8084/health');
  console.log('   View logs:        tail -f logs/knirvserver.log');

  return true;
}

// Run health check if called directly
if (require.main === module) {
  checkNexusHealth().then(healthy => {
    process.exit(healthy ? 0 : 1);
  }).catch(error => {
    console.error('❌ Health check failed:', error.message);
    process.exit(1);
  });
}

module.exports = {
  checkNexusHealth,
  checkUnifiedBinary,
  testNexusService,
  makeHttpRequest
};
