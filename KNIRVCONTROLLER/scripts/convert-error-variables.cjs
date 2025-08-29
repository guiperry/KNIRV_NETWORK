#!/usr/bin/env node

/**
 * Convert _error variables back to error
 * 
 * This script finds all instances of '_error' variables and converts them back to 'error'
 * so they will be considered "used" by ESLint.
 */

const fs = require('fs');
const path = require('path');
const { execSync } = require('child_process');

const PROJECT_ROOT = path.join(__dirname, '..');
const SRC_DIR = path.join(PROJECT_ROOT, 'src');

console.log('🔄 Converting _error variables back to error...\n');

// Function to recursively find all TypeScript files
function findTSFiles(dir) {
  const files = [];
  const items = fs.readdirSync(dir);
  
  for (const item of items) {
    const fullPath = path.join(dir, item);
    const stat = fs.statSync(fullPath);
    
    if (stat.isDirectory() && !item.startsWith('.') && item !== 'node_modules') {
      files.push(...findTSFiles(fullPath));
    } else if (stat.isFile() && (item.endsWith('.ts') || item.endsWith('.tsx'))) {
      files.push(fullPath);
    }
  }
  
  return files;
}

// Function to convert _error to error in a file
function convertErrorVariables(filePath) {
  const content = fs.readFileSync(filePath, 'utf8');
  const originalContent = content;
  
  // Replace all instances of _error with error
  const newContent = content.replace(/\b_error\b/g, 'error');
  
  if (newContent !== originalContent) {
    fs.writeFileSync(filePath, newContent);
    const relativePath = path.relative(PROJECT_ROOT, filePath);
    console.log(`✅ Converted _error to error in ${relativePath}`);
    return true;
  }
  
  return false;
}

// Main function
function main() {
  const tsFiles = findTSFiles(SRC_DIR);
  let totalConverted = 0;
  
  console.log(`Found ${tsFiles.length} TypeScript files to process...\n`);
  
  for (const file of tsFiles) {
    if (convertErrorVariables(file)) {
      totalConverted++;
    }
  }
  
  console.log(`\n🎉 Conversion complete! Converted _error to error in ${totalConverted} files.`);
  
  // Run ESLint to see the improvement
  console.log('\n📊 Running ESLint to check results...');
  try {
    execSync('npm run lint', { stdio: 'inherit', cwd: PROJECT_ROOT });
    console.log('\n✨ All ESLint issues resolved!');
  } catch (error) {
    console.log('\n📝 Some issues remain, but _error variables have been converted.');
  }
}

if (require.main === module) {
  main();
}

module.exports = { convertErrorVariables, findTSFiles };
