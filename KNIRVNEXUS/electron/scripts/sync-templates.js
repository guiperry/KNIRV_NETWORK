#!/usr/bin/env node

/**
 * Template Sync Script for Agentic Engine Electron Build
 * 
 * This script copies agent templates from the source directory to the Electron
 * build directory so they can be bundled with the packaged application.
 * 
 * Usage: node scripts/sync-templates.js
 */

const fs = require('fs');
const path = require('path');

// Paths
const sourceTemplatesDir = path.join(__dirname, '../../agent/templates');
const targetTemplatesDir = path.join(__dirname, '../agent/templates');

/**
 * Recursively copy directory contents
 */
function copyDirectory(src, dest) {
  // Create destination directory if it doesn't exist
  if (!fs.existsSync(dest)) {
    fs.mkdirSync(dest, { recursive: true });
  }

  // Read source directory
  const entries = fs.readdirSync(src, { withFileTypes: true });

  for (const entry of entries) {
    const srcPath = path.join(src, entry.name);
    const destPath = path.join(dest, entry.name);

    if (entry.isDirectory()) {
      // Recursively copy subdirectories
      copyDirectory(srcPath, destPath);
    } else {
      // Copy files
      fs.copyFileSync(srcPath, destPath);
      console.log(`  ✓ Copied: ${entry.name}`);
    }
  }
}

/**
 * Clean target directory
 */
function cleanDirectory(dir) {
  if (fs.existsSync(dir)) {
    fs.rmSync(dir, { recursive: true, force: true });
    console.log(`  🗑️  Cleaned: ${dir}`);
  }
}

/**
 * Main sync function
 */
function syncTemplates() {
  console.log('🔄 Syncing Agent Templates for Electron Build\n');

  // Check if source directory exists
  if (!fs.existsSync(sourceTemplatesDir)) {
    console.error(`❌ Source templates directory not found: ${sourceTemplatesDir}`);
    process.exit(1);
  }

  console.log(`📂 Source: ${sourceTemplatesDir}`);
  console.log(`📂 Target: ${targetTemplatesDir}\n`);

  try {
    // Clean target directory
    console.log('🧹 Cleaning target directory...');
    cleanDirectory(targetTemplatesDir);

    // Copy templates
    console.log('📋 Copying templates...');
    copyDirectory(sourceTemplatesDir, targetTemplatesDir);

    // Verify copy
    const sourceFiles = fs.readdirSync(sourceTemplatesDir);
    const targetFiles = fs.readdirSync(targetTemplatesDir);

    console.log(`\n📊 Sync Summary:`);
    console.log(`   Source files: ${sourceFiles.length}`);
    console.log(`   Target files: ${targetFiles.length}`);

    if (sourceFiles.length === targetFiles.length) {
      console.log('✅ Template sync completed successfully!');
    } else {
      console.warn('⚠️  File count mismatch - some files may not have been copied');
    }

    // List copied files
    console.log('\n📄 Copied template files:');
    targetFiles.forEach(file => {
      console.log(`   • ${file}`);
    });

  } catch (error) {
    console.error(`❌ Error syncing templates: ${error.message}`);
    process.exit(1);
  }
}

// Run the sync if this script is executed directly
if (require.main === module) {
  syncTemplates();
}

module.exports = { syncTemplates };
