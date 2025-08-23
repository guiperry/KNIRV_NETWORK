#!/usr/bin/env node

/**
 * KNIRV Controller Template Literals and Imports Fixer
 * 
 * This script fixes:
 * 1. Escaped template literals (\`) that should be normal template literals (`)
 * 2. Incorrect import paths using @/ alias that should be relative paths
 * 3. Any other syntax issues in the merged application
 */

import fs from 'fs/promises';
import path from 'path';
import { glob } from 'glob';

const SCRIPT_DIR = process.cwd();
const FRONTEND_DIR = path.join(SCRIPT_DIR, 'frontend');
const RECEIVER_DIR = path.join(SCRIPT_DIR, 'receiver');
const SRC_DIR = path.join(SCRIPT_DIR, 'src');

// Utility functions
const log = (message, type = 'info') => {
  const timestamp = new Date().toISOString();
  const prefix = type === 'error' ? '❌' : type === 'success' ? '✅' : type === 'warning' ? '⚠️' : 'ℹ️';
  console.log(`${prefix} [${timestamp}] ${message}`);
};

// Fix escaped template literals
const fixTemplateLiterals = async (filePath) => {
  try {
    let content = await fs.readFile(filePath, 'utf8');
    let changed = false;
    
    // Fix escaped template literals: {\` -> {`
    const escapedTemplateRegex = /\{\\`([^`]*(?:\\.[^`]*)*)`\}/g;
    if (escapedTemplateRegex.test(content)) {
      content = content.replace(escapedTemplateRegex, '{`$1`}');
      changed = true;
      log(`Fixed escaped template literals in ${filePath}`);
    }
    
    // Fix escaped template literal interpolations: \${ -> ${
    const escapedInterpolationRegex = /\\(\$\{[^}]*\})/g;
    if (escapedInterpolationRegex.test(content)) {
      content = content.replace(escapedInterpolationRegex, '$1');
      changed = true;
      log(`Fixed escaped interpolations in ${filePath}`);
    }
    
    // Fix triple-escaped template literals: \\\` -> `
    const tripleEscapedRegex = /\\\\\`([^`]*(?:\\.[^`]*)*)\\\\\`/g;
    if (tripleEscapedRegex.test(content)) {
      content = content.replace(tripleEscapedRegex, '`$1`');
      changed = true;
      log(`Fixed triple-escaped template literals in ${filePath}`);
    }
    
    if (changed) {
      await fs.writeFile(filePath, content);
      return true;
    }
    
    return false;
  } catch (error) {
    log(`Error fixing template literals in ${filePath}: ${error.message}`, 'error');
    return false;
  }
};

// Fix import paths
const fixImportPaths = async (filePath) => {
  try {
    let content = await fs.readFile(filePath, 'utf8');
    let changed = false;
    
    // Fix @/react-app/components/ imports to relative paths
    const importRegex = /import\s+([^'"\n]+)\s+from\s+['"]@\/react-app\/components\/([^'"]+)['"]/g;
    const matches = [...content.matchAll(importRegex)];
    
    if (matches.length > 0) {
      for (const match of matches) {
        const [fullMatch, importName, componentPath] = match;
        const relativePath = `../components/${componentPath}`;
        const newImport = `import ${importName} from '${relativePath}'`;
        content = content.replace(fullMatch, newImport);
        changed = true;
        log(`Fixed import path in ${filePath}: ${componentPath}`);
      }
    }
    
    // Fix @/react-app/pages/ imports to relative paths
    const pagesImportRegex = /import\s+([^'"\n]+)\s+from\s+['"]@\/react-app\/pages\/([^'"]+)['"]/g;
    const pagesMatches = [...content.matchAll(pagesImportRegex)];
    
    if (pagesMatches.length > 0) {
      for (const match of pagesMatches) {
        const [fullMatch, importName, pagePath] = match;
        const relativePath = `../pages/${pagePath}`;
        const newImport = `import ${importName} from '${relativePath}'`;
        content = content.replace(fullMatch, newImport);
        changed = true;
        log(`Fixed page import path in ${filePath}: ${pagePath}`);
      }
    }
    
    // Fix @/shared/ imports to relative paths
    const sharedImportRegex = /import\s+([^'"\n]+)\s+from\s+['"]@\/shared\/([^'"]+)['"]/g;
    const sharedMatches = [...content.matchAll(sharedImportRegex)];

    if (sharedMatches.length > 0) {
      for (const match of sharedMatches) {
        const [fullMatch, importName, sharedPath] = match;
        const relativePath = `../../shared/${sharedPath}`;
        const newImport = `import ${importName} from '${relativePath}'`;
        content = content.replace(fullMatch, newImport);
        changed = true;
        log(`Fixed shared import path in ${filePath}: ${sharedPath}`);
      }
    }

    // Fix incorrect relative paths for consolidated structure
    // For files in src/components/ importing from sensory-shell
    if (filePath.includes('/src/components/') && content.includes("from './sensory-shell/")) {
      content = content.replace(/from '\.\/sensory-shell\//g, "from '../sensory-shell/");
      changed = true;
      log(`Fixed sensory-shell import path in ${filePath}`);
    }

    // For files in src/receiver-backup/components/ importing from sensory-shell
    if (filePath.includes('/src/receiver-backup/components/') && content.includes("from './sensory-shell/")) {
      content = content.replace(/from '\.\/sensory-shell\//g, "from '../../sensory-shell/");
      changed = true;
      log(`Fixed receiver-backup sensory-shell import path in ${filePath}`);
    }

    // For files in src/receiver-backup/ importing from sensory-shell
    if (filePath.includes('/src/receiver-backup/') && !filePath.includes('/components/') && content.includes("from './sensory-shell/")) {
      content = content.replace(/from '\.\/sensory-shell\//g, "from '../sensory-shell/");
      changed = true;
      log(`Fixed receiver-backup root sensory-shell import path in ${filePath}`);
    }

    // For files in src/manager/react-app/pages/ importing from components
    if (filePath.includes('/src/manager/react-app/pages/') && content.includes("from './components/")) {
      content = content.replace(/from '\.\/components\//g, "from '../components/");
      changed = true;
      log(`Fixed manager pages component import path in ${filePath}`);
    }

    // For files in src/manager/react-app/pages/ importing from other relative paths
    if (filePath.includes('/src/manager/react-app/pages/')) {
      // Fix any remaining ./components/ imports
      if (content.includes("from './components/")) {
        content = content.replace(/from '\.\/components\//g, "from '../components/");
        changed = true;
        log(`Fixed remaining component imports in ${filePath}`);
      }
    }
    
    if (changed) {
      await fs.writeFile(filePath, content);
      return true;
    }
    
    return false;
  } catch (error) {
    log(`Error fixing import paths in ${filePath}: ${error.message}`, 'error');
    return false;
  }
};

// Process all TypeScript and JavaScript files
const processFiles = async (directory) => {
  log(`Processing files in ${directory}...`);
  
  const pattern = path.join(directory, '**/*.{ts,tsx,js,jsx}');
  const files = await glob(pattern, { 
    ignore: ['**/node_modules/**', '**/dist/**', '**/.git/**'] 
  });
  
  let totalFixed = 0;
  
  for (const file of files) {
    const templateLiteralsFixed = await fixTemplateLiterals(file);
    const importPathsFixed = await fixImportPaths(file);
    
    if (templateLiteralsFixed || importPathsFixed) {
      totalFixed++;
    }
  }
  
  log(`Processed ${files.length} files, fixed ${totalFixed} files in ${directory}`, 'success');
  return totalFixed;
};

// Sync changes between frontend and receiver
const syncChanges = async () => {
  log('Syncing changes between frontend and receiver...');
  
  try {
    // Copy fixed App.tsx from frontend to receiver
    const frontendApp = path.join(FRONTEND_DIR, 'src', 'App.tsx');
    const receiverApp = path.join(RECEIVER_DIR, 'src', 'App.tsx');
    
    await fs.copyFile(frontendApp, receiverApp);
    log('Synced App.tsx from frontend to receiver', 'success');
    
    // Copy fixed manager components from frontend to receiver
    const frontendManager = path.join(FRONTEND_DIR, 'src', 'manager');
    const receiverManager = path.join(RECEIVER_DIR, 'src', 'manager');
    
    // Remove existing manager in receiver
    try {
      await fs.rm(receiverManager, { recursive: true, force: true });
    } catch (error) {
      // Ignore if doesn't exist
    }
    
    // Copy manager directory
    await copyDirectory(frontendManager, receiverManager);
    log('Synced manager components from frontend to receiver', 'success');
    
  } catch (error) {
    log(`Error syncing changes: ${error.message}`, 'error');
  }
};

const copyDirectory = async (src, dest) => {
  await fs.mkdir(dest, { recursive: true });
  const entries = await fs.readdir(src, { withFileTypes: true });
  
  for (const entry of entries) {
    const srcPath = path.join(src, entry.name);
    const destPath = path.join(dest, entry.name);
    
    if (entry.isDirectory()) {
      await copyDirectory(srcPath, destPath);
    } else {
      await fs.copyFile(srcPath, destPath);
    }
  }
};

// Main execution function
const main = async () => {
  try {
    log('Starting template literals and imports fix process...');

    let totalFixed = 0;

    // Process consolidated src directory (priority)
    try {
      await fs.access(SRC_DIR);
      const srcFixed = await processFiles(SRC_DIR);
      totalFixed += srcFixed;
      log(`📁 Processed consolidated src directory: ${srcFixed} files fixed`);
    } catch (error) {
      log('📁 No consolidated src directory found, processing legacy structure...');
    }

    // Process frontend directory (legacy)
    try {
      await fs.access(FRONTEND_DIR);
      const frontendFixed = await processFiles(FRONTEND_DIR);
      totalFixed += frontendFixed;
      log(`📁 Processed frontend directory: ${frontendFixed} files fixed`);
    } catch (error) {
      log('📁 No frontend directory found');
    }

    // Process receiver directory (legacy)
    try {
      await fs.access(RECEIVER_DIR);
      const receiverFixed = await processFiles(RECEIVER_DIR);
      totalFixed += receiverFixed;
      log(`� Processed receiver directory: ${receiverFixed} files fixed`);
    } catch (error) {
      log('📁 No receiver directory found');
    }

    // Only sync changes if we're in legacy structure
    if (totalFixed > 0) {
      try {
        await fs.access(FRONTEND_DIR);
        await fs.access(RECEIVER_DIR);
        await syncChanges();
        log('🔄 Changes synced between frontend and receiver');
      } catch (error) {
        log('🔄 Skipping sync - consolidated structure detected');
      }
    }

    log(`✅ Fix completed successfully!`, 'success');
    log(`📊 Summary: Fixed ${totalFixed} files total`);
    log('');
    log('🚀 Next steps:');
    log('  1. Run "npm run build" to test the build');
    log('  2. Run "npm start" to start the unified application');
    log('  3. Check the interface for proper functionality');

  } catch (error) {
    log(`❌ Fix failed: ${error.message}`, 'error');
    process.exit(1);
  }
};

// Run the script
if (import.meta.url === `file://${process.argv[1]}`) {
  main();
}

export { main };
