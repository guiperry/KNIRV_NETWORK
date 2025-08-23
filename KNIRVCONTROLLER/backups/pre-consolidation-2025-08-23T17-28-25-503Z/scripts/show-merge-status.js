#!/usr/bin/env node

/**
 * KNIRV Controller Merge Status Display
 * 
 * This script shows the current status of the merge and provides
 * helpful information about the unified application structure.
 */

import fs from 'fs/promises';
import path from 'path';

const SCRIPT_DIR = process.cwd();

// Utility functions
const log = (message, type = 'info') => {
  const prefix = type === 'error' ? '❌' : type === 'success' ? '✅' : type === 'warning' ? '⚠️' : 'ℹ️';
  console.log(`${prefix} ${message}`);
};

const fileExists = async (filePath) => {
  try {
    await fs.access(filePath);
    return true;
  } catch {
    return false;
  }
};

const checkMergeStatus = async () => {
  console.log('🔍 KNIRV Controller Merge Status Check\n');
  
  // Check if frontend directory exists
  const frontendExists = await fileExists(path.join(SCRIPT_DIR, 'frontend'));
  const receiverMerged = await fileExists(path.join(SCRIPT_DIR, 'receiver', 'src', 'manager'));
  const backendUpdated = await fileExists(path.join(SCRIPT_DIR, 'backend', 'unifiedServer.ts'));
  
  if (frontendExists && receiverMerged && backendUpdated) {
    log('Merge completed successfully!', 'success');
    console.log('');
    
    // Show structure
    console.log('📁 Current Structure:');
    console.log('├── frontend/          (Unified application - Manager + Receiver)');
    console.log('├── receiver/          (Original receiver with manager merged)');
    console.log('├── manager/           (Original manager - preserved)');
    console.log('├── backend/           (Updated to serve frontend/)');
    console.log('└── backups/           (Timestamped backups)');
    console.log('');
    
    // Show available commands
    console.log('🚀 Available Commands:');
    console.log('npm run dev              - Start unified development server');
    console.log('npm run build:frontend   - Build frontend only');
    console.log('npm run build:unified    - Build both backend and frontend');
    console.log('npm start                - Start production server');
    console.log('npm run test:frontend    - Run frontend tests');
    console.log('npm run lint:frontend    - Lint frontend code');
    console.log('');
    
    // Show access points
    console.log('🌐 Access Points:');
    console.log('http://localhost:3000/           - Receiver Interface');
    console.log('http://localhost:3000/manager    - Manager Interface');
    console.log('http://localhost:3000/health     - Health Check');
    console.log('http://localhost:3000/api        - Backend API');
    console.log('');
    
    // Show navigation
    console.log('🔗 Navigation:');
    console.log('• Purple "Manager Interface →" button (top-right) - Go to Manager');
    console.log('• Teal "← Receiver Interface" button (top-right) - Go to Receiver');
    console.log('');
    
    // Show next steps
    console.log('📋 Next Steps:');
    console.log('1. Run "npm run dev" to start the unified application');
    console.log('2. Navigate to http://localhost:3000 to test the receiver interface');
    console.log('3. Click the navigation button to switch to manager interface');
    console.log('4. Test all functionality in both interfaces');
    console.log('5. Run "node scripts/validate-merge.js" to validate the merge');
    
  } else if (receiverMerged && !frontendExists) {
    log('Partial merge detected - receiver has manager merged but not exported to root', 'warning');
    console.log('');
    console.log('Run the merge script again to complete the root export:');
    console.log('  ./scripts/run-merge.sh');
    
  } else if (!receiverMerged && !frontendExists) {
    log('No merge detected', 'info');
    console.log('');
    console.log('To merge manager into receiver and export to root:');
    console.log('  ./scripts/run-merge.sh');
    console.log('');
    console.log('Or run the merge script directly:');
    console.log('  node scripts/backup-and-merge-manager.js');
    
  } else {
    log('Unexpected state detected', 'warning');
    console.log('');
    console.log('Please check the file structure and run validation:');
    console.log('  node scripts/validate-merge.js');
  }
};

const showBackupInfo = async () => {
  const backupDir = path.join(SCRIPT_DIR, 'backups');
  const backupExists = await fileExists(backupDir);
  
  if (backupExists) {
    try {
      const backupEntries = await fs.readdir(backupDir);
      const backupFolders = backupEntries.filter(entry => entry.startsWith('backup-'));
      
      if (backupFolders.length > 0) {
        console.log('');
        console.log('💾 Available Backups:');
        backupFolders.sort().reverse().slice(0, 5).forEach((backup, index) => {
          const timestamp = backup.replace('backup-', '').replace(/-/g, ':');
          console.log(`${index + 1}. ${backup} (${timestamp})`);
        });
        
        if (backupFolders.length > 5) {
          console.log(`   ... and ${backupFolders.length - 5} more`);
        }
        
        console.log('');
        console.log('To restore from backup:');
        console.log('  cp -r backups/backup-[timestamp]/receiver/* receiver/');
        console.log('  cp -r backups/backup-[timestamp]/manager/* manager/');
      }
    } catch (error) {
      log(`Error reading backups: ${error.message}`, 'error');
    }
  }
};

const showTroubleshooting = () => {
  console.log('');
  console.log('🔧 Troubleshooting:');
  console.log('');
  console.log('If you encounter issues:');
  console.log('1. Run validation: node scripts/validate-merge.js');
  console.log('2. Check logs for specific errors');
  console.log('3. Restore from backup if needed');
  console.log('4. Re-run merge script: ./scripts/run-merge.sh');
  console.log('');
  console.log('Common issues:');
  console.log('• Port 3000 in use: Kill other processes or change PORT env var');
  console.log('• Build errors: Run "npm install" in frontend/ directory');
  console.log('• Import errors: Check that all manager components were copied');
  console.log('• Navigation issues: Verify React Router configuration');
  console.log('');
  console.log('For detailed documentation: cat MERGE_DOCUMENTATION.md');
};

// Main function
const main = async () => {
  try {
    await checkMergeStatus();
    await showBackupInfo();
    showTroubleshooting();
    
  } catch (error) {
    log(`Error checking merge status: ${error.message}`, 'error');
    process.exit(1);
  }
};

// Run the script
if (import.meta.url === `file://${process.argv[1]}`) {
  main();
}

export { main };
