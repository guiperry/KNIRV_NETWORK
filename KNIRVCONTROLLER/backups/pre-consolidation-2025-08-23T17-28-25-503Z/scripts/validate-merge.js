#!/usr/bin/env node

/**
 * KNIRV Controller Merge Validation Script
 * 
 * This script validates that the manager-receiver merge was successful
 * and all components are properly integrated.
 */

import fs from 'fs/promises';
import path from 'path';

const SCRIPT_DIR = process.cwd();
const RECEIVER_DIR = path.join(SCRIPT_DIR, 'receiver');
const FRONTEND_DIR = path.join(SCRIPT_DIR, 'frontend');
const BACKEND_DIR = path.join(SCRIPT_DIR, 'backend');

// Utility functions
const log = (message, type = 'info') => {
  const timestamp = new Date().toISOString();
  const prefix = type === 'error' ? '❌' : type === 'success' ? '✅' : type === 'warning' ? '⚠️' : 'ℹ️';
  console.log(`${prefix} [${timestamp}] ${message}`);
};

const fileExists = async (filePath) => {
  try {
    await fs.access(filePath);
    return true;
  } catch {
    return false;
  }
};

const readJsonFile = async (filePath) => {
  try {
    const content = await fs.readFile(filePath, 'utf8');
    return JSON.parse(content);
  } catch (error) {
    throw new Error(`Failed to read JSON file ${filePath}: ${error.message}`);
  }
};

// Validation functions
const validateDirectoryStructure = async () => {
  log('Validating directory structure...');

  const requiredPaths = [
    'frontend/src/App.tsx',
    'frontend/src/manager',
    'frontend/src/manager/react-app',
    'frontend/src/manager/react-app/components',
    'frontend/src/manager/react-app/pages',
    'frontend/src/manager/shared',
    'frontend/package.json',
    'receiver/src/App.tsx',
    'receiver/src/manager',
    'receiver/package.json'
  ];

  let allValid = true;

  for (const requiredPath of requiredPaths) {
    const fullPath = path.join(SCRIPT_DIR, requiredPath);
    const exists = await fileExists(fullPath);

    if (exists) {
      log(`✓ ${requiredPath}`, 'success');
    } else {
      log(`✗ Missing: ${requiredPath}`, 'error');
      allValid = false;
    }
  }

  return allValid;
};

const validatePackageJson = async () => {
  log('Validating package.json merge...');

  try {
    // Check frontend package.json
    const frontendPkg = await readJsonFile(path.join(FRONTEND_DIR, 'package.json'));

    // Check if manager dependencies are present
    const expectedDependencies = [
      'react-router',
      'hono',
      'lucide-react',
      'qr-scanner',
      'qrcode',
      'three'
    ];

    let allDepsPresent = true;

    for (const dep of expectedDependencies) {
      if (frontendPkg.dependencies && frontendPkg.dependencies[dep]) {
        log(`✓ Frontend dependency: ${dep}`, 'success');
      } else {
        log(`✗ Missing frontend dependency: ${dep}`, 'error');
        allDepsPresent = false;
      }
    }

    // Check frontend package name update
    if (frontendPkg.name === 'knirv-controller-frontend') {
      log('✓ Frontend package name updated correctly', 'success');
    } else {
      log(`✗ Frontend package name not updated. Expected: knirv-controller-frontend, Got: ${frontendPkg.name}`, 'error');
      allDepsPresent = false;
    }

    // Check root package.json scripts
    const rootPkg = await readJsonFile(path.join(SCRIPT_DIR, 'package.json'));
    const expectedScripts = [
      'dev:frontend',
      'build:frontend',
      'test:frontend',
      'lint:frontend',
      'install:frontend'
    ];

    for (const script of expectedScripts) {
      if (rootPkg.scripts && rootPkg.scripts[script]) {
        log(`✓ Root script: ${script}`, 'success');
      } else {
        log(`✗ Missing root script: ${script}`, 'error');
        allDepsPresent = false;
      }
    }

    return allDepsPresent;
  } catch (error) {
    log(`Error validating package.json: ${error.message}`, 'error');
    return false;
  }
};

const validateBackendConfig = async () => {
  log('Validating backend configuration...');

  try {
    const unifiedServerPath = path.join(BACKEND_DIR, 'unifiedServer.ts');
    const serverContent = await fs.readFile(unifiedServerPath, 'utf8');

    let allValid = true;

    // Check if receiverDistPath points to frontend
    if (serverContent.includes("path.join(rootDir, 'frontend', 'dist')")) {
      log('✓ Backend serves from frontend/dist', 'success');
    } else {
      log('✗ Backend not updated to serve from frontend/dist', 'error');
      allValid = false;
    }

    // Check log message updates
    if (serverContent.includes('Unified frontend available at:')) {
      log('✓ Backend log messages updated', 'success');
    } else {
      log('✗ Backend log messages not updated', 'error');
      allValid = false;
    }

    // Check error message updates
    if (serverContent.includes('npm run build:frontend')) {
      log('✓ Backend error messages updated', 'success');
    } else {
      log('✗ Backend error messages not updated', 'error');
      allValid = false;
    }

    return allValid;
  } catch (error) {
    log(`Error validating backend config: ${error.message}`, 'error');
    return false;
  }
};

const validateUnifiedApp = async () => {
  log('Validating unified App.tsx...');

  try {
    const appPath = path.join(FRONTEND_DIR, 'src', 'App.tsx');
    const appContent = await fs.readFile(appPath, 'utf8');
    
    const requiredImports = [
      'react-router-dom',
      './manager/react-app/components/UnifiedInterface',
      './manager/react-app/pages/Skills',
      './manager/react-app/pages/UDC',
      './manager/react-app/pages/Wallet'
    ];
    
    const requiredComponents = [
      'NavigationButton',
      'ReceiverInterface',
      'ManagerInterface',
      'Router',
      'Routes',
      'Route'
    ];
    
    let allValid = true;
    
    // Check imports
    for (const importPath of requiredImports) {
      if (appContent.includes(importPath)) {
        log(`✓ Import: ${importPath}`, 'success');
      } else {
        log(`✗ Missing import: ${importPath}`, 'error');
        allValid = false;
      }
    }
    
    // Check components
    for (const component of requiredComponents) {
      if (appContent.includes(component)) {
        log(`✓ Component: ${component}`, 'success');
      } else {
        log(`✗ Missing component: ${component}`, 'error');
        allValid = false;
      }
    }
    
    // Check routing structure
    if (appContent.includes('path="/manager/*"') && appContent.includes('path="/*"')) {
      log('✓ Routing structure correct', 'success');
    } else {
      log('✗ Routing structure incorrect', 'error');
      allValid = false;
    }
    
    return allValid;
  } catch (error) {
    log(`Error validating App.tsx: ${error.message}`, 'error');
    return false;
  }
};

const validateManagerComponents = async () => {
  log('Validating manager components...');

  const managerComponentPaths = [
    'frontend/src/manager/react-app/components/UnifiedInterface.tsx',
    'frontend/src/manager/react-app/pages/Skills.tsx',
    'frontend/src/manager/react-app/pages/UDC.tsx',
    'frontend/src/manager/react-app/pages/Wallet.tsx',
    'frontend/src/manager/shared/ComponentBridge.ts',
    'receiver/src/manager/react-app/components/UnifiedInterface.tsx',
    'receiver/src/manager/react-app/pages/Skills.tsx',
    'receiver/src/manager/react-app/pages/UDC.tsx',
    'receiver/src/manager/react-app/pages/Wallet.tsx',
    'receiver/src/manager/shared/ComponentBridge.ts'
  ];

  let allValid = true;

  for (const componentPath of managerComponentPaths) {
    const fullPath = path.join(SCRIPT_DIR, componentPath);
    const exists = await fileExists(fullPath);

    if (exists) {
      log(`✓ Manager component: ${componentPath}`, 'success');
    } else {
      log(`✗ Missing manager component: ${componentPath}`, 'error');
      allValid = false;
    }
  }

  return allValid;
};

const validateBackups = async () => {
  log('Validating backups...');
  
  const backupDir = path.join(SCRIPT_DIR, 'backups');
  const backupExists = await fileExists(backupDir);
  
  if (!backupExists) {
    log('⚠️ No backups directory found', 'warning');
    return false;
  }
  
  try {
    const backupEntries = await fs.readdir(backupDir);
    const backupFolders = backupEntries.filter(entry => entry.startsWith('backup-'));
    
    if (backupFolders.length > 0) {
      log(`✓ Found ${backupFolders.length} backup(s)`, 'success');
      
      // Check the most recent backup
      const latestBackup = backupFolders.sort().pop();
      const latestBackupPath = path.join(backupDir, latestBackup);
      
      const managerBackup = await fileExists(path.join(latestBackupPath, 'manager'));
      const receiverBackup = await fileExists(path.join(latestBackupPath, 'receiver'));
      
      if (managerBackup && receiverBackup) {
        log(`✓ Latest backup complete: ${latestBackup}`, 'success');
        return true;
      } else {
        log(`✗ Latest backup incomplete: ${latestBackup}`, 'error');
        return false;
      }
    } else {
      log('✗ No backup folders found', 'error');
      return false;
    }
  } catch (error) {
    log(`Error validating backups: ${error.message}`, 'error');
    return false;
  }
};

// Main validation function
const main = async () => {
  try {
    log('Starting KNIRV Controller merge validation...');
    
    const validations = [
      { name: 'Directory Structure', fn: validateDirectoryStructure },
      { name: 'Package.json Merge', fn: validatePackageJson },
      { name: 'Backend Configuration', fn: validateBackendConfig },
      { name: 'Unified App.tsx', fn: validateUnifiedApp },
      { name: 'Manager Components', fn: validateManagerComponents },
      { name: 'Backups', fn: validateBackups }
    ];
    
    let allPassed = true;
    const results = [];
    
    for (const validation of validations) {
      log(`\n--- ${validation.name} ---`);
      const result = await validation.fn();
      results.push({ name: validation.name, passed: result });
      
      if (!result) {
        allPassed = false;
      }
    }
    
    // Summary
    log('\n=== VALIDATION SUMMARY ===');
    for (const result of results) {
      log(`${result.name}: ${result.passed ? 'PASSED' : 'FAILED'}`, result.passed ? 'success' : 'error');
    }
    
    if (allPassed) {
      log('\n🎉 All validations passed! The merge was successful.', 'success');
      log('You can now run "npm run dev:receiver" to start the unified application.');
    } else {
      log('\n💥 Some validations failed. Please review the errors above.', 'error');
      log('You may need to re-run the merge script or manually fix the issues.');
    }
    
    return allPassed;
    
  } catch (error) {
    log(`❌ Validation failed: ${error.message}`, 'error');
    return false;
  }
};

// Run the script
if (import.meta.url === `file://${process.argv[1]}`) {
  main().then(success => {
    process.exit(success ? 0 : 1);
  });
}

export { main };
