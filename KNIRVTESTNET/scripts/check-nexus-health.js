#!/usr/bin/env node

/**
 * KNIRVTESTNET Nexus Portal Health Check
 *
 * Checks the health of the Nexus Portal application and optimizes builds
 * by comparing build logs to avoid unnecessary rebuilds
 */

const fs = require('fs');
const path = require('path');
const { execSync } = require('child_process');

/**
 * Compare build logs to determine if rebuild is needed
 * @param {string} sourcePath - Path to source KNIRVNEXUS directory
 * @param {string} targetPath - Path to target portal directory
 * @returns {object} - Comparison result with rebuild recommendation
 */
function compareBuildLogs(sourcePath, targetPath) {
  const sourceBuildLog = path.join(sourcePath, 'build.log');
  const targetBuildLog = path.join(targetPath, 'build.log');

  console.log('\n🔍 Build Log Comparison');
  console.log('======================');

  // Check if source build log exists
  if (!fs.existsSync(sourceBuildLog)) {
    console.log('⚠️  Source build.log not found - rebuild recommended');
    return { needsRebuild: true, reason: 'Source build log missing' };
  }

  // Check if target build log exists
  if (!fs.existsSync(targetBuildLog)) {
    console.log('⚠️  Target build.log not found - rebuild required');
    return { needsRebuild: true, reason: 'Target build log missing' };
  }

  try {
    const sourceLog = JSON.parse(fs.readFileSync(sourceBuildLog, 'utf8'));
    const targetLog = JSON.parse(fs.readFileSync(targetBuildLog, 'utf8'));

    console.log(`📅 Source build: ${sourceLog.buildTimestamp || 'unknown'}`);
    console.log(`📅 Target build: ${targetLog.buildTimestamp || 'unknown'}`);
    console.log(`🔗 Source git hash: ${sourceLog.gitHash || 'unknown'}`);
    console.log(`🔗 Target git hash: ${targetLog.gitHash || 'unknown'}`);

    // Compare git hashes first (most reliable)
    if (sourceLog.gitHash && targetLog.gitHash && sourceLog.gitHash !== 'unknown' && targetLog.gitHash !== 'unknown') {
      if (sourceLog.gitHash === targetLog.gitHash) {
        console.log('✅ Git hashes match - no rebuild needed');
        return { needsRebuild: false, reason: 'Git hashes match' };
      } else {
        console.log('🔄 Git hashes differ - rebuild recommended');
        return { needsRebuild: true, reason: 'Git hashes differ' };
      }
    }

    // Fallback to timestamp comparison
    if (sourceLog.buildTimestamp && targetLog.buildTimestamp) {
      const sourceTime = new Date(sourceLog.buildTimestamp);
      const targetTime = new Date(targetLog.buildTimestamp);

      // Allow target to be up to one build behind (as requested)
      const timeDiff = sourceTime.getTime() - targetTime.getTime();
      const oneHourMs = 60 * 60 * 1000; // 1 hour tolerance

      if (timeDiff <= oneHourMs) {
        console.log('✅ Build timestamps are close enough - no rebuild needed');
        return { needsRebuild: false, reason: 'Build timestamps within tolerance' };
      } else {
        console.log('🔄 Target build is significantly older - rebuild recommended');
        return { needsRebuild: true, reason: 'Target build too old' };
      }
    }

    // If we can't determine, err on the side of caution
    console.log('⚠️  Cannot determine build status - rebuild recommended');
    return { needsRebuild: true, reason: 'Cannot determine build status' };

  } catch (error) {
    console.log(`❌ Error reading build logs: ${error.message}`);
    return { needsRebuild: true, reason: `Build log error: ${error.message}` };
  }
}

/**
 * Check if build can be skipped based on build log comparison
 * @returns {boolean} - True if build can be skipped
 */
function canSkipBuild() {
  const sourcePath = path.join(__dirname, '..', '..', 'KNIRVNEXUS');
  const altSourcePath = path.join(__dirname, '..', 'KNIRVNEXUS');
  const targetPath = path.join(__dirname, '..', 'data', 'knirvnexus', 'portal');

  // Determine source path
  let actualSourcePath;
  if (fs.existsSync(sourcePath)) {
    actualSourcePath = sourcePath;
  } else if (fs.existsSync(altSourcePath)) {
    actualSourcePath = altSourcePath;
  } else {
    console.log('❌ KNIRVNEXUS source directory not found');
    return false;
  }

  // Check if target directory exists and has a build
  if (!fs.existsSync(targetPath)) {
    console.log('📁 Target portal directory does not exist - build required');
    return false;
  }

  const distPath = path.join(targetPath, '.next');
  if (!fs.existsSync(distPath)) {
    console.log('📦 Target build output (.next) does not exist - build required');
    return false;
  }

  // Check for critical dependencies
  const nodeModulesPath = path.join(targetPath, 'node_modules');
  if (!fs.existsSync(nodeModulesPath)) {
    console.log('📦 Target node_modules does not exist - build required');
    return false;
  }

  // Verify critical dependencies exist
  const criticalDeps = ['socket.io', 'next', 'react'];
  for (const dep of criticalDeps) {
    const depPath = path.join(nodeModulesPath, dep);
    if (!fs.existsSync(depPath)) {
      console.log(`📦 Critical dependency '${dep}' missing - build required`);
      return false;
    }
  }

  // Check for server.js file
  const serverPath = path.join(targetPath, 'server.js');
  if (!fs.existsSync(serverPath)) {
    console.log('📄 server.js missing - build required');
    return false;
  }

  // Compare build logs
  const comparison = compareBuildLogs(actualSourcePath, targetPath);

  if (!comparison.needsRebuild) {
    console.log(`✅ Build can be skipped: ${comparison.reason}`);
    return true;
  } else {
    console.log(`🔄 Build required: ${comparison.reason}`);
    return false;
  }
}

function checkNexusHealth() {
  console.log('🔍 Nexus Portal Health Check');
  console.log('=============================');
  
  const issues = [];
  const warnings = [];
  const nexusDir = path.join(__dirname, '..', 'data', 'knirvnexus', 'portal');
  
  // Check if Nexus Portal directory exists
  if (!fs.existsSync(nexusDir)) {
    issues.push('Nexus Portal directory not found');
    return false;
  }
  
  console.log('✅ Nexus Portal directory exists');
  
  // Check package.json
  const packageJsonPath = path.join(nexusDir, 'package.json');
  if (!fs.existsSync(packageJsonPath)) {
    issues.push('Nexus Portal package.json not found');
  } else {
    console.log('✅ Nexus Portal package.json exists');
    
    try {
      const packageJson = require(packageJsonPath);
      console.log(`✅ Nexus Portal version: ${packageJson.version || 'unknown'}`);
      
      // Check for required scripts
      const requiredScripts = ['build', 'dev'];
      requiredScripts.forEach(script => {
        if (!packageJson.scripts || !packageJson.scripts[script]) {
          warnings.push(`Missing script: ${script}`);
        } else {
          console.log(`✅ Script available: ${script}`);
        }
      });
      
    } catch (error) {
      issues.push(`Failed to parse Nexus Portal package.json: ${error.message}`);
    }
  }
  
  // Check for required files
  const requiredFiles = [
    'index.html',
    'src',
    'public'
  ];
  
  requiredFiles.forEach(file => {
    const filePath = path.join(nexusDir, file);
    if (!fs.existsSync(filePath)) {
      warnings.push(`Nexus Portal file/directory not found: ${file}`);
    } else {
      console.log(`✅ Nexus Portal has: ${file}`);
    }
  });
  
  // Check for build output with build optimization
  const distPath = path.join(nexusDir, '.next');
  if (fs.existsSync(distPath)) {
    console.log('✅ Nexus Portal build output (.next) exists');

    // Check if build can be skipped
    if (canSkipBuild()) {
      console.log('🚀 Build optimization: Current build is up-to-date, rebuild can be skipped');
    } else {
      warnings.push('NEXUS Frontend build is outdated - run ./scripts/build-nexus-frontend.sh');
    }
  } else {
    warnings.push('NEXUS Frontend not built yet - run ./scripts/build-nexus-frontend.sh');
  }
  
  // Check for node_modules
  const nodeModulesPath = path.join(nexusDir, 'node_modules');
  if (fs.existsSync(nodeModulesPath)) {
    console.log('✅ Nexus Portal dependencies installed');
  } else {
    warnings.push('NEXUS Frontend dependencies not installed - run ./scripts/build-nexus-frontend.sh');
  }
  
  // Check configuration
  const configPath = path.join(nexusDir, 'public', 'config.js');
  if (fs.existsSync(configPath)) {
    console.log('✅ Nexus Portal config.js exists');
  } else {
    warnings.push('Nexus Portal config.js not found - run load-endpoints script');
  }
  
  // Summary
  console.log('\n📊 Nexus Portal Health Summary');
  console.log('===============================');
  
  if (issues.length === 0 && warnings.length === 0) {
    console.log('🎉 Nexus Portal health check passed!');
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
  
  console.log('\n✅ Nexus Portal health check completed with warnings only');
  return true;
}

// Run health check if called directly
if (require.main === module) {
  const healthy = checkNexusHealth();
  process.exit(healthy ? 0 : 1);
}

module.exports = { checkNexusHealth, canSkipBuild, compareBuildLogs };
