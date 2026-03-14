#!/usr/bin/env node

/**
 * KNIRVTESTNET Nexus Portal Health Check
 * 
 * Checks the health of the Nexus Portal application
 */

const fs = require('fs');
const path = require('path');

function checkNexusHealth() {
  console.log('🔍 Nexus Portal Health Check');
  console.log('=============================');
  
  const issues = [];
  const warnings = [];
  const nexusDir = path.join(__dirname, '..', 'nexus-portal');
  
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
  
  // Check for build output
  const distPath = path.join(nexusDir, 'dist');
  if (fs.existsSync(distPath)) {
    console.log('✅ Nexus Portal build output (dist) exists');
  } else {
    warnings.push('Nexus Portal not built yet - run npm run build in nexus-portal directory');
  }
  
  // Check for node_modules
  const nodeModulesPath = path.join(nexusDir, 'node_modules');
  if (fs.existsSync(nodeModulesPath)) {
    console.log('✅ Nexus Portal dependencies installed');
  } else {
    warnings.push('Nexus Portal dependencies not installed - run npm install in nexus-portal directory');
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

module.exports = { checkNexusHealth };
