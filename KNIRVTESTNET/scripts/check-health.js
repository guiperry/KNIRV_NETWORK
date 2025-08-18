#!/usr/bin/env node

/**
 * KNIRVTESTNET Health Check Script
 * 
 * Checks the health of all applications and services
 */

const fs = require('fs');
const path = require('path');

function checkHealth() {
  console.log('🏥 KNIRVTESTNET Health Check');
  console.log('============================');
  
  const issues = [];
  const warnings = [];
  
  // Check if required directories exist
  const requiredDirs = [
    'graphchain-explorer',
    'nexus-portal',
    'agent-developer-portal',
    'nanda_ans',
    'server',
    'scripts',
    'config',
    'assets'
  ];
  
  requiredDirs.forEach(dir => {
    const dirPath = path.join(__dirname, '..', dir);
    if (!fs.existsSync(dirPath)) {
      issues.push(`Missing required directory: ${dir}`);
    } else {
      console.log(`✅ Directory exists: ${dir}`);
    }
  });
  
  // Check if required files exist
  const requiredFiles = [
    'package.json',
    'server/app.js',
    'scripts/load-endpoints.js',
    'index.html'
  ];
  
  requiredFiles.forEach(file => {
    const filePath = path.join(__dirname, '..', file);
    if (!fs.existsSync(filePath)) {
      issues.push(`Missing required file: ${file}`);
    } else {
      console.log(`✅ File exists: ${file}`);
    }
  });
  
  // Check package.json dependencies
  try {
    const packageJson = require('../package.json');
    const requiredDeps = [
      'express',
      'cors',
      'helmet',
      'compression',
      'morgan',
      'js-yaml',
      'axios'
    ];
    
    requiredDeps.forEach(dep => {
      if (!packageJson.dependencies[dep]) {
        issues.push(`Missing required dependency: ${dep}`);
      } else {
        console.log(`✅ Dependency available: ${dep}`);
      }
    });
  } catch (error) {
    issues.push(`Failed to read package.json: ${error.message}`);
  }
  
  // Check environment configuration
  const { loadEndpoints } = require('./load-endpoints');
  try {
    const { endpoints, config } = loadEndpoints('testnet');
    console.log(`✅ Endpoints loaded: ${Object.keys(endpoints).length} services`);
    console.log(`✅ Configuration loaded for environment: ${config.DEPLOYMENT_ENV}`);
  } catch (error) {
    warnings.push(`Endpoint loading issue: ${error.message}`);
  }
  
  // Check application-specific health
  
  // GraphChain Explorer
  const graphchainConfigPath = path.join(__dirname, '..', 'graphchain-explorer', 'js', 'config.js');
  if (fs.existsSync(graphchainConfigPath)) {
    console.log('✅ GraphChain Explorer config exists');
  } else {
    warnings.push('GraphChain Explorer config not found - run load-endpoints script');
  }
  
  // Nexus Portal
  const nexusPackagePath = path.join(__dirname, '..', 'nexus-portal', 'package.json');
  if (fs.existsSync(nexusPackagePath)) {
    console.log('✅ Nexus Portal package.json exists');
  } else {
    warnings.push('Nexus Portal package.json not found');
  }
  
  // NANDA ANS
  const nandaPackagePath = path.join(__dirname, '..', 'nanda_ans', 'package.json');
  if (fs.existsSync(nandaPackagePath)) {
    console.log('✅ NANDA ANS package.json exists');
  } else {
    warnings.push('NANDA ANS package.json not found');
  }
  
  // Summary
  console.log('\n📊 Health Check Summary');
  console.log('=======================');
  
  if (issues.length === 0 && warnings.length === 0) {
    console.log('🎉 All health checks passed!');
    console.log('✅ KNIRVTESTNET is ready for deployment');
    return true;
  }
  
  if (warnings.length > 0) {
    console.log('\n⚠️  Warnings:');
    warnings.forEach(warning => console.log(`   - ${warning}`));
  }
  
  if (issues.length > 0) {
    console.log('\n❌ Issues found:');
    issues.forEach(issue => console.log(`   - ${issue}`));
    console.log('\n🔧 Please fix these issues before deployment');
    return false;
  }
  
  console.log('\n✅ Health check completed with warnings only');
  return true;
}

// Run health check if called directly
if (require.main === module) {
  const healthy = checkHealth();
  process.exit(healthy ? 0 : 1);
}

module.exports = { checkHealth };
