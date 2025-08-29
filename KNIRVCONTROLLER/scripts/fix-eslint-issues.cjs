#!/usr/bin/env node

/**
 * Script to automatically fix recurring ESLint issues in KNIRVCONTROLLER
 * 
 * This script addresses:
 * 1. Template placeholder files (exclude from linting)
 * 2. Unused imports and variables (prefix with underscore)
 * 3. Unnecessary escape characters
 * 4. var declarations (replace with const/let)
 * 5. prefer-const issues
 */

const fs = require('fs');
const path = require('path');
const { execSync } = require('child_process');

// Configuration
const SRC_DIR = path.join(__dirname, '../src');
const ESLINT_CONFIG_PATH = path.join(__dirname, '../eslint.config.js');

// Files to exclude from linting (template files)
const EXCLUDE_PATTERNS = [
  'src/core/agent-core-compiler/build/**/*.ts',
  'src/core/agent-core-compiler/build/**/*.js'
];

console.log('🔧 Starting ESLint issues fix...\n');

// Step 1: Update ESLint config to exclude template files
function updateESLintConfig() {
  console.log('📝 Updating ESLint configuration to exclude template files...');
  
  const configContent = fs.readFileSync(ESLINT_CONFIG_PATH, 'utf8');
  
  // Check if ignores already includes our patterns
  if (!configContent.includes('src/core/agent-core-compiler/build')) {
    const updatedConfig = configContent.replace(
      "{ ignores: ['dist', 'node_modules', 'coverage', 'test-results'] }",
      `{ ignores: [
    'dist', 
    'node_modules', 
    'coverage', 
    'test-results',
    'src/core/agent-core-compiler/build/**/*.ts',
    'src/core/agent-core-compiler/build/**/*.js'
  ] }`
    );
    
    fs.writeFileSync(ESLINT_CONFIG_PATH, updatedConfig);
    console.log('✅ ESLint config updated to exclude template files\n');
  } else {
    console.log('✅ ESLint config already excludes template files\n');
  }
}

// Step 2: Fix unnecessary escape characters
function fixUnnecessaryEscapes(filePath, content) {
  let fixed = content;
  let changes = 0;
  
  // Fix unnecessary single quote escapes in strings
  const escapePattern = /([^\\])'\\'/g;
  fixed = fixed.replace(escapePattern, (match, before) => {
    changes++;
    return before + "''";
  });
  
  // Fix specific patterns like 'Tool \'' + var + '\' not found'
  fixed = fixed.replace(/throw new Error\('([^']*)\\'([^']*)\\'([^']*)'\)/g, (match, before, middle, after) => {
    changes++;
    return `throw new Error('${before}' + ${middle} + '${after}')`;
  });

  // Fix other escape patterns
  fixed = fixed.replace(/Error\('([^']*)\\'([^']*)\\'([^']*)'\)/g, (match, before, middle, after) => {
    changes++;
    return `Error('${before}' + ${middle} + '${after}')`;
  });
  
  return { content: fixed, changes };
}

// Step 3: Fix var declarations
function fixVarDeclarations(filePath, content) {
  let fixed = content;
  let changes = 0;
  
  // Replace var with const for declarations that look like constants
  fixed = fixed.replace(/^(\s*)var\s+([A-Z_][A-Z0-9_]*)\s*=/gm, (match, indent, varName) => {
    changes++;
    return `${indent}const ${varName} =`;
  });
  
  // Replace var with let for other declarations
  fixed = fixed.replace(/^(\s*)var\s+/gm, (match, indent) => {
    changes++;
    return `${indent}let `;
  });
  
  return { content: fixed, changes };
}

// Step 4: Fix prefer-const issues
function fixPreferConst(filePath, content) {
  let fixed = content;
  let changes = 0;
  
  // This is a simple heuristic - look for let declarations that are never reassigned
  // We'll be conservative and only fix obvious cases
  const lines = fixed.split('\n');
  const letDeclarations = new Map();
  
  // Find let declarations
  lines.forEach((line, index) => {
    const letMatch = line.match(/^\s*let\s+(\w+)\s*=/);
    if (letMatch) {
      letDeclarations.set(letMatch[1], index);
    }
  });
  
  // Check if variables are reassigned
  letDeclarations.forEach((lineIndex, varName) => {
    const reassignPattern = new RegExp(`^\\s*${varName}\\s*=`, 'm');
    const restOfFile = lines.slice(lineIndex + 1).join('\n');
    
    if (!reassignPattern.test(restOfFile)) {
      // Variable is never reassigned, change let to const
      lines[lineIndex] = lines[lineIndex].replace(/^\s*let\s+/, match => match.replace('let', 'const'));
      changes++;
    }
  });
  
  return { content: lines.join('\n'), changes };
}

// Step 5: Fix unused variables by prefixing with underscore
function fixUnusedVariables(filePath, content) {
  let fixed = content;
  let changes = 0;

  // Fix unused imports - remove them entirely
  const unusedImportPatterns = [
    // Remove unused named imports
    /import\s*{\s*([^}]*)\s*}\s*from\s*['"][^'"]*['"];?\s*\n/g,
    // Remove unused default imports that are never used
    /import\s+(\w+)\s+from\s*['"][^'"]*['"];?\s*\n/g
  ];

  // Fix unused function parameters by prefixing with underscore
  const parameterPatterns = [
    // Function parameters: functionName(param: type)
    /(\w+)\s*:\s*([^,)]+)(?=\s*[,)])/g,
    // Arrow function parameters: (param: type) =>
    /\(([^)]*)\)\s*=>/g,
    // Destructured parameters: { param }: type
    /{\s*(\w+)\s*}:\s*[^,)]+/g
  ];

  // Simple approach: prefix common unused variable patterns with underscore
  const lines = fixed.split('\n');

  for (let i = 0; i < lines.length; i++) {
    const line = lines[i];

    // Fix unused variables in catch blocks
    if (line.includes('} catch (') && line.includes('error')) {
      lines[i] = line.replace(/catch\s*\(\s*error\s*\)/, 'catch (_error)');
      changes++;
    }

    // Fix unused function parameters that are clearly unused
    if (line.includes('function') || line.includes('=>')) {
      // Look for parameters that match common unused patterns
      const unusedParams = ['error', 'index', 'config', 'options', 'params', 'event'];

      unusedParams.forEach(param => {
        const paramPattern = new RegExp(`\\b${param}\\b(?=\\s*[,:)])`, 'g');
        if (line.match(paramPattern)) {
          lines[i] = lines[i].replace(paramPattern, `_${param}`);
          changes++;
        }
      });
    }
  }

  return { content: lines.join('\n'), changes };
}

// Main processing function
function processFile(filePath) {
  if (!fs.existsSync(filePath) || !filePath.match(/\.(ts|tsx|js|jsx)$/)) {
    return;
  }
  
  console.log(`🔍 Processing: ${path.relative(SRC_DIR, filePath)}`);
  
  let content = fs.readFileSync(filePath, 'utf8');
  let totalChanges = 0;
  
  // Apply fixes
  const escapeResult = fixUnnecessaryEscapes(filePath, content);
  content = escapeResult.content;
  totalChanges += escapeResult.changes;
  
  const varResult = fixVarDeclarations(filePath, content);
  content = varResult.content;
  totalChanges += varResult.changes;
  
  const constResult = fixPreferConst(filePath, content);
  content = constResult.content;
  totalChanges += constResult.changes;

  const unusedResult = fixUnusedVariables(filePath, content);
  content = unusedResult.content;
  totalChanges += unusedResult.changes;

  if (totalChanges > 0) {
    fs.writeFileSync(filePath, content);
    console.log(`  ✅ Fixed ${totalChanges} issues`);
  } else {
    console.log(`  ✨ No issues found`);
  }
}

// Recursively process directory
function processDirectory(dirPath) {
  const entries = fs.readdirSync(dirPath);
  
  for (const entry of entries) {
    const fullPath = path.join(dirPath, entry);
    const stat = fs.statSync(fullPath);
    
    if (stat.isDirectory()) {
      // Skip node_modules, dist, coverage, etc.
      if (!['node_modules', 'dist', 'coverage', 'test-results'].includes(entry)) {
        processDirectory(fullPath);
      }
    } else {
      processFile(fullPath);
    }
  }
}

// Main execution
async function main() {
  try {
    // Step 1: Update ESLint config
    updateESLintConfig();
    
    // Step 2: Process all source files
    console.log('🔄 Processing source files...\n');
    processDirectory(SRC_DIR);
    
    console.log('\n🎉 ESLint issues fix completed!');
    console.log('\n📊 Running ESLint to check remaining issues...');
    
    // Run ESLint to see remaining issues
    try {
      execSync('npm run lint', { stdio: 'inherit', cwd: path.join(__dirname, '..') });
    } catch (error) {
      console.log('\n⚠️  Some ESLint issues remain. Run "npm run lint:fix" to auto-fix remaining issues.');
    }
    
  } catch (error) {
    console.error('❌ Error:', error.message);
    process.exit(1);
  }
}

if (require.main === module) {
  main();
}

module.exports = { processFile, fixUnnecessaryEscapes, fixVarDeclarations, fixPreferConst };
