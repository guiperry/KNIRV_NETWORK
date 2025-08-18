#!/usr/bin/env node

/**
 * Agentic-Engine App Data Migration Script
 * 
 * This script migrates data from the old Electron-generated app data directory
 * to the standardized directory that matches the Go backend.
 * 
 * Old directory: ~/.config/knirv-engine-desktop/
 * New directory: ~/.config/Agentic-Engine/
 * 
 * Usage: node scripts/migrate-appdata.js [--dry-run] [--force]
 */

const fs = require('fs');
const path = require('path');
const os = require('os');

// Parse command line arguments
const args = process.argv.slice(2);
const isDryRun = args.includes('--dry-run');
const isForce = args.includes('--force');

// Get app data directory paths
function getAppDataPaths() {
  const appName = 'Agentic-Engine';
  
  let configDir;
  switch (process.platform) {
    case 'win32':
      const appData = process.env.APPDATA;
      if (!appData) {
        throw new Error('APPDATA environment variable not set');
      }
      configDir = appData;
      break;
    case 'darwin':
      configDir = path.join(os.homedir(), 'Library', 'Application Support');
      break;
    case 'linux':
    default:
      configDir = process.env.XDG_CONFIG_HOME || path.join(os.homedir(), '.config');
      break;
  }
  
  const newAppDataPath = path.join(configDir, appName);
  const oldAppDataPath = path.join(os.homedir(), '.config', 'knirv-engine-desktop');
  
  return { newAppDataPath, oldAppDataPath };
}

// Copy files recursively
function copyRecursive(src, dest, dryRun = false) {
  const stats = fs.statSync(src);
  
  if (stats.isDirectory()) {
    if (!dryRun && !fs.existsSync(dest)) {
      fs.mkdirSync(dest, { recursive: true });
      console.log(`📁 Created directory: ${dest}`);
    } else if (dryRun) {
      console.log(`📁 [DRY RUN] Would create directory: ${dest}`);
    }
    
    const items = fs.readdirSync(src);
    items.forEach(item => {
      const srcPath = path.join(src, item);
      const destPath = path.join(dest, item);
      copyRecursive(srcPath, destPath, dryRun);
    });
  } else {
    // Only copy if destination doesn't exist (don't overwrite)
    if (!fs.existsSync(dest)) {
      if (!dryRun) {
        fs.copyFileSync(src, dest);
        console.log(`📄 Copied file: ${path.relative(process.cwd(), dest)}`);
      } else {
        console.log(`📄 [DRY RUN] Would copy file: ${path.relative(process.cwd(), dest)}`);
      }
    } else {
      console.log(`⚠️  Skipped existing file: ${path.relative(process.cwd(), dest)}`);
    }
  }
}

// Get directory size
function getDirectorySize(dirPath) {
  let totalSize = 0;
  
  function calculateSize(currentPath) {
    const stats = fs.statSync(currentPath);
    if (stats.isDirectory()) {
      const items = fs.readdirSync(currentPath);
      items.forEach(item => {
        calculateSize(path.join(currentPath, item));
      });
    } else {
      totalSize += stats.size;
    }
  }
  
  try {
    calculateSize(dirPath);
  } catch (error) {
    console.warn(`Warning: Could not calculate size for ${dirPath}: ${error.message}`);
  }
  
  return totalSize;
}

// Format bytes to human readable
function formatBytes(bytes) {
  if (bytes === 0) return '0 Bytes';
  const k = 1024;
  const sizes = ['Bytes', 'KB', 'MB', 'GB'];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
}

// Main migration function
function migrate() {
  console.log('🚀 Agentic-Engine App Data Migration Tool\n');
  
  const { newAppDataPath, oldAppDataPath } = getAppDataPaths();
  
  console.log(`📂 Old directory: ${oldAppDataPath}`);
  console.log(`📂 New directory: ${newAppDataPath}\n`);
  
  // Check if old directory exists
  if (!fs.existsSync(oldAppDataPath)) {
    console.log('✅ No old app data directory found. Migration not needed.');
    return;
  }
  
  // Check if new directory already exists and has content
  if (fs.existsSync(newAppDataPath) && fs.readdirSync(newAppDataPath).length > 0 && !isForce) {
    console.log('⚠️  New directory already exists and contains data.');
    console.log('   Use --force to proceed anyway (will not overwrite existing files).');
    return;
  }
  
  // Calculate sizes
  const oldSize = getDirectorySize(oldAppDataPath);
  console.log(`📊 Old directory size: ${formatBytes(oldSize)}`);
  
  if (isDryRun) {
    console.log('\n🔍 DRY RUN MODE - No files will be modified\n');
  } else {
    console.log('\n🔄 Starting migration...\n');
  }
  
  try {
    // Copy contents from old to new directory
    copyRecursive(oldAppDataPath, newAppDataPath, isDryRun);
    
    if (!isDryRun) {
      console.log('\n✅ Migration completed successfully!');
      
      // Rename old directory to indicate it's been migrated
      const backupPath = oldAppDataPath + '.migrated';
      if (!fs.existsSync(backupPath)) {
        fs.renameSync(oldAppDataPath, backupPath);
        console.log(`📦 Old directory backed up to: ${backupPath}`);
      }
      
      console.log('\n🎉 All done! Your app data has been migrated to the standardized location.');
      console.log('   The application will now use the same directory as the Go backend.');
    } else {
      console.log('\n✅ Dry run completed. Use without --dry-run to perform actual migration.');
    }
    
  } catch (error) {
    console.error('\n❌ Error during migration:', error.message);
    process.exit(1);
  }
}

// Show help
function showHelp() {
  console.log(`
Agentic-Engine App Data Migration Tool

Usage: node scripts/migrate-appdata.js [options]

Options:
  --dry-run    Show what would be migrated without making changes
  --force      Proceed even if new directory already exists
  --help       Show this help message

Examples:
  node scripts/migrate-appdata.js --dry-run    # Preview migration
  node scripts/migrate-appdata.js             # Perform migration
  node scripts/migrate-appdata.js --force     # Force migration
`);
}

// Main execution
if (args.includes('--help') || args.includes('-h')) {
  showHelp();
} else {
  migrate();
}
