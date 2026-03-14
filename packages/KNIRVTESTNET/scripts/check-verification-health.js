#!/usr/bin/env node

/**
 * KNIRVTESTNET Verification Health Check Script
 * 
 * Integrates formal verification status into health monitoring
 */

const fs = require('fs');
const path = require('path');
const { execSync } = require('child_process');

function checkVerificationHealth() {
  console.log('🔍 Verification Health Check');
  console.log('===========================');
  
  const issues = [];
  const warnings = [];
  
  // Check if modp directory exists
  const modpPath = path.join(__dirname, '..', '..', 'modp');
  if (!fs.existsSync(modpPath)) {
    issues.push('modp directory not found');
    return { healthy: false, issues, warnings };
  }
  
  console.log('✅ modp directory found');
  
  // Check P project file
  const projectFile = path.join(modpPath, 'KnirvNetwork.pproj');
  if (!fs.existsSync(projectFile)) {
    issues.push('KnirvNetwork.pproj not found');
  } else {
    console.log('✅ P project file exists');
  }
  
  // Check if P compiler is available
  try {
    const pCompilerPath = path.join(process.env.HOME || '', '.p-lang', 'Bld', 'Drops', 'Release', 'Binaries', 'net8.0', 'p.dll');
    if (fs.existsSync(pCompilerPath)) {
      console.log('✅ P compiler available');
    } else {
      warnings.push('P compiler not found - running in simulation mode');
    }
  } catch (error) {
    warnings.push('P compiler check failed');
  }
  
  // Check recent verification results
  const resultsDir = path.join(modpPath, 'results');
  if (fs.existsSync(resultsDir)) {
    const resultFiles = fs.readdirSync(resultsDir)
      .filter(file => file.startsWith('test-') && file.endsWith('.log'))
      .sort()
      .reverse();
    
    if (resultFiles.length > 0) {
      const latestResult = resultFiles[0];
      const resultPath = path.join(resultsDir, latestResult);
      
      try {
        const resultContent = fs.readFileSync(resultPath, 'utf8');
        const passedTests = (resultContent.match(/passed:/gi) || []).length;
        const failedTests = (resultContent.match(/failed:/gi) || []).length;
        
        console.log(`✅ Latest verification results: ${passedTests} passed, ${failedTests} failed`);
        
        if (failedTests > 0) {
          warnings.push(`${failedTests} verification tests failed in latest run`);
        }
      } catch (error) {
        warnings.push('Failed to parse latest verification results');
      }
    } else {
      warnings.push('No verification results found');
    }
  } else {
    warnings.push('Verification results directory not found');
  }
  
  // Check if verification service is running
  try {
    const pidFile = path.join(__dirname, '..', '..', 'modp', 'data', 'knirvverifier.pid');
    if (fs.existsSync(pidFile)) {
      const pid = fs.readFileSync(pidFile, 'utf8').trim();
      if (pid && !isNaN(pid)) {
        try {
          execSync(`kill -0 ${pid}`, { stdio: 'ignore' });
          console.log(`✅ Verification service running (PID: ${pid})`);
        } catch (error) {
          warnings.push('Verification service PID file exists but process not running');
        }
      }
    } else {
      console.log('ℹ️  Verification service not running');
    }
  } catch (error) {
    warnings.push('Failed to check verification service status');
  }
  
  return {
    healthy: issues.length === 0,
    issues,
    warnings
  };
}

// Run check if called directly
if (require.main === module) {
  const result = checkVerificationHealth();
  
  console.log('\n📊 Verification Health Summary');
  console.log('==============================');
  
  if (result.healthy && result.warnings.length === 0) {
    console.log('🎉 All verification checks passed!');
    process.exit(0);
  }
  
  if (result.warnings.length > 0) {
    console.log('\n⚠️  Warnings:');
    result.warnings.forEach(warning => console.log(`   - ${warning}`));
  }
  
  if (result.issues.length > 0) {
    console.log('\n❌ Issues:');
    result.issues.forEach(issue => console.log(`   - ${issue}`));
    process.exit(1);
  }
  
  process.exit(0);
}

module.exports = { checkVerificationHealth };