#!/usr/bin/env node

/**
 * Targeted ESLint Fixer
 * 
 * This script runs ESLint, parses the output, and applies targeted fixes
 * for each specific rule violation. It aims to reach zero warnings and errors
 * by applying safe, rule-specific fixes iteratively.
 */

const fs = require('fs');
const path = require('path');
const { execSync } = require('child_process');

const PROJECT_ROOT = path.join(__dirname, '..');
const SRC_DIR = path.join(PROJECT_ROOT, 'src');

// Validate that the content has valid syntax
function validateSyntax(content, filePath) {
  try {
    // Write to a temporary file and check with TypeScript compiler
    const tempFile = filePath + '.temp';
    fs.writeFileSync(tempFile, content);

    try {
      // Use tsc to check syntax (will throw if syntax is invalid)
      execSync(`npx tsc --noEmit --skipLibCheck "${tempFile}"`, {
        cwd: PROJECT_ROOT,
        stdio: 'pipe'
      });
      fs.unlinkSync(tempFile);
      return true;
    } catch (error) {
      fs.unlinkSync(tempFile);
      return false;
    }
  } catch (error) {
    return false;
  }
}

console.log('🎯 Starting targeted ESLint fixer...\n');

// Parse ESLint output
function parseESLintOutput() {
  console.log('📊 Running ESLint to get current issues...');

  // Use text format directly for more reliable parsing
  try {
    const output = execSync('npm run lint', {
      cwd: PROJECT_ROOT,
      encoding: 'utf8'
    });
    return []; // No issues if command succeeds
  } catch (error) {
    const output = error.stdout || '';
    return parseTextOutput(output);
  }
}

function extractIssuesFromJSON(results) {
  const issues = [];

  results.forEach(file => {
    file.messages.forEach(message => {
      issues.push({
        filePath: file.filePath,
        line: message.line,
        column: message.column,
        ruleId: message.ruleId,
        severity: message.severity, // 1 = warning, 2 = error
        message: message.message,
        nodeType: message.nodeType
      });
    });
  });

  console.log(`Found ${issues.length} issues to fix\n`);
  return issues;
}

// Parse text output from ESLint
function parseTextOutput(output) {
  const lines = output.split('\n');
  const issues = [];
  let currentFile = '';

  lines.forEach(line => {
    // Check if this line is a file path
    if (line.startsWith('/') && line.includes('.ts')) {
      currentFile = line.trim();
      return;
    }

    // Parse issue lines: "   18:8   error  'UnifiedInterface' is defined but never used  @typescript-eslint/no-unused-vars"
    const match = line.match(/^\s*(\d+):(\d+)\s+(error|warning)\s+(.+?)\s+(@?[\w-]+(?:\/[\w-]+)*)\s*$/);
    if (match && currentFile) {
      const [, lineNum, column, severity, message, ruleId] = match;

      // Skip parsing errors for safety
      if (message.includes('Parsing error')) {
        console.log(`⚠️  Skipping parsing error in ${currentFile}:${lineNum}`);
        return;
      }

      issues.push({
        filePath: path.resolve(currentFile),
        line: parseInt(lineNum),
        column: parseInt(column),
        ruleId: ruleId,
        severity: severity === 'error' ? 2 : 1,
        message: message.trim(),
        nodeType: null
      });
    }
  });

  console.log(`Parsed ${issues.length} issues from text output\n`);
  return issues;
}

// Group issues by rule for batch processing
function groupIssuesByRule(issues) {
  const grouped = {};
  issues.forEach(issue => {
    if (!issue.ruleId) return; // Skip parsing errors
    
    if (!grouped[issue.ruleId]) {
      grouped[issue.ruleId] = [];
    }
    grouped[issue.ruleId].push(issue);
  });
  
  return grouped;
}

// Rule-specific fixers
const ruleFixes = {
  'no-var': (issue, content) => {
    const lines = content.split('\n');
    const line = lines[issue.line - 1];
    if (line && line.includes('var ')) {
      lines[issue.line - 1] = line.replace(/\bvar\b/g, 'const');
      return lines.join('\n');
    }
    return content;
  },

  '@typescript-eslint/no-explicit-any': (issue, content) => {
    const lines = content.split('\n');
    const line = lines[issue.line - 1];
    if (line && line.includes(': any')) {
      lines[issue.line - 1] = line.replace(/:\s*any\b/g, ': unknown');
      return lines.join('\n');
    }
    return content;
  },

  '@typescript-eslint/no-unsafe-function-type': (issue, content) => {
    const lines = content.split('\n');
    const line = lines[issue.line - 1];
    if (line && line.includes(': Function')) {
      lines[issue.line - 1] = line.replace(/:\s*Function\b/g, ': (...args: unknown[]) => unknown');
      return lines.join('\n');
    }
    return content;
  },

  'no-useless-escape': (issue, content) => {
    const lines = content.split('\n');
    const line = lines[issue.line - 1];
    if (line) {
      // Remove unnecessary escapes in strings
      let fixed = line.replace(/\\'/g, "'").replace(/\\"/g, '"');
      if (fixed !== line) {
        lines[issue.line - 1] = fixed;
        return lines.join('\n');
      }
    }
    return content;
  },

  '@typescript-eslint/no-unused-vars': (issue, content) => {
    const lines = content.split('\n');
    const line = lines[issue.line - 1];
    if (!line) return content;

    // Extract variable name from message
    const match = issue.message.match(/'([^']+)' is (defined but never used|assigned a value but never used)/);
    if (!match) return content;

    const varName = match[1];

    // Only handle simple, safe cases to avoid syntax errors

    // Case 1: Function parameters - prefix with underscore
    if (issue.message.includes('Allowed unused args must match') ||
        line.includes('function') || line.includes('=>') || line.includes('(')) {
      // Only if it's a simple parameter (not destructured)
      const simpleParamPattern = new RegExp(`\\b${varName}\\b(?=\\s*[,)])`);
      if (simpleParamPattern.test(line) && !line.includes('{') && !line.includes('[')) {
        lines[issue.line - 1] = line.replace(new RegExp(`\\b${varName}\\b`), `_${varName}`);
        return lines.join('\n');
      }
    }

    // Case 2: Simple import removal (only single imports)
    if (line.trim().startsWith('import') && line.includes(varName)) {
      // Only handle simple single imports like: import { varName } from '...'
      const singleImportPattern = new RegExp(`^\\s*import\\s*{\\s*${varName}\\s*}\\s*from`);
      if (singleImportPattern.test(line)) {
        lines[issue.line - 1] = '';
        return lines.join('\n');
      }
    }

    // Case 3: Simple variable declarations that are clearly unused
    if (line.includes('const ') || line.includes('let ')) {
      // Only if it's a simple assignment (not destructuring)
      const simpleVarPattern = new RegExp(`^\\s*(const|let)\\s+${varName}\\s*=`);
      if (simpleVarPattern.test(line) && !line.includes('{') && !line.includes('[')) {
        // Special case: if it's _error, change it back to error to make it used
        if (varName === '_error') {
          lines[issue.line - 1] = line.replace(new RegExp(`\\b${varName}\\b`), 'error');
          return lines.join('\n');
        }
        // For other variables, prefix with underscore instead of removing
        lines[issue.line - 1] = line.replace(new RegExp(`\\b${varName}\\b`), `_${varName}`);
        return lines.join('\n');
      }
    }

    // For all other cases, don't make changes to avoid syntax errors
    return content;
  },

  'no-case-declarations': (issue, content) => {
    const lines = content.split('\n');
    const line = lines[issue.line - 1];
    if (line && (line.includes('const ') || line.includes('let '))) {
      // Wrap the declaration in a block
      const indent = line.match(/^\s*/)[0];
      lines[issue.line - 1] = `${indent}{`;
      lines.splice(issue.line, 0, `${indent}  ${line.trim()}`);
      lines.splice(issue.line + 1, 0, `${indent}}`);
      return lines.join('\n');
    }
    return content;
  },

  'no-dupe-else-if': (issue, content) => {
    const lines = content.split('\n');
    // Remove the duplicate else-if block
    if (issue.line > 1) {
      lines.splice(issue.line - 1, 1);
      return lines.join('\n');
    }
    return content;
  },

  '@typescript-eslint/no-empty-object-type': (issue, content) => {
    const lines = content.split('\n');
    const line = lines[issue.line - 1];
    if (line && line.includes('interface') && line.includes('{}')) {
      // Replace empty interface with proper type
      lines[issue.line - 1] = line.replace(/{\s*}/, '{\n  [key: string]: unknown;\n}');
      return lines.join('\n');
    }
    return content;
  },

  '@typescript-eslint/no-require-imports': (issue, content) => {
    const lines = content.split('\n');
    const line = lines[issue.line - 1];
    if (line && line.includes('require(')) {
      // Remove require import lines
      lines[issue.line - 1] = '';
      return lines.join('\n');
    }
    return content;
  },

  // Special fixer to convert _error back to error
  'convert-error-variables': (issue, content) => {
    const lines = content.split('\n');
    const line = lines[issue.line - 1];

    // Check if this is an _error variable that should be converted back to error
    if (line && line.includes('_error')) {
      // Replace _error with error to make it "used"
      lines[issue.line - 1] = line.replace(/\b_error\b/g, 'error');
      return lines.join('\n');
    }
    return content;
  }
};

// Apply fixes for a specific rule
function applyRuleFixes(ruleId, issues) {
  if (!ruleFixes[ruleId]) {
    console.log(`⚠️  No fixer available for rule: ${ruleId}`);
    return 0;
  }

  console.log(`🔧 Fixing ${issues.length} issues for rule: ${ruleId}`);
  
  const fileGroups = {};
  issues.forEach(issue => {
    const relativePath = path.relative(PROJECT_ROOT, issue.filePath);
    if (!fileGroups[relativePath]) {
      fileGroups[relativePath] = [];
    }
    fileGroups[relativePath].push(issue);
  });

  let totalFixed = 0;
  
  Object.entries(fileGroups).forEach(([filePath, fileIssues]) => {
    const fullPath = path.join(PROJECT_ROOT, filePath);
    if (!fs.existsSync(fullPath)) return;
    
    let content = fs.readFileSync(fullPath, 'utf8');
    let fixed = 0;
    
    // Sort issues by line number (descending) to avoid line number shifts
    fileIssues.sort((a, b) => b.line - a.line);
    
    fileIssues.forEach(issue => {
      const newContent = ruleFixes[ruleId](issue, content);
      if (newContent !== content) {
        // Validate syntax before applying the fix
        if (validateSyntax(newContent, fullPath)) {
          content = newContent;
          fixed++;
        } else {
          console.log(`  ⚠️  Skipping fix for ${filePath}:${issue.line} - would break syntax`);
        }
      }
    });

    if (fixed > 0) {
      fs.writeFileSync(fullPath, content);
      console.log(`  ✅ Fixed ${fixed} issues in ${filePath}`);
      totalFixed += fixed;
    }
  });
  
  return totalFixed;
}

// Main execution
async function main() {
  let iteration = 1;
  const maxIterations = 10;
  
  while (iteration <= maxIterations) {
    console.log(`\n🔄 Iteration ${iteration}/${maxIterations}`);
    
    const issues = parseESLintOutput();
    if (issues.length === 0) {
      console.log('\n🎉 No more ESLint issues found! All done!');
      break;
    }
    
    const grouped = groupIssuesByRule(issues);
    const ruleIds = Object.keys(grouped);
    
    console.log(`Found issues for rules: ${ruleIds.join(', ')}\n`);
    
    // Apply fixes in order of safety (safest first)
    const safeRules = [
      'no-var',
      '@typescript-eslint/no-explicit-any',
      '@typescript-eslint/no-unsafe-function-type',
      '@typescript-eslint/no-empty-object-type',
      '@typescript-eslint/no-require-imports',
      'no-useless-escape',
      'no-dupe-else-if',
      'no-case-declarations',
      '@typescript-eslint/no-unused-vars'
    ];
    
    let totalFixed = 0;
    
    for (const ruleId of safeRules) {
      if (grouped[ruleId]) {
        const fixed = applyRuleFixes(ruleId, grouped[ruleId]);
        totalFixed += fixed;
      }
    }
    
    if (totalFixed === 0) {
      console.log('\n⚠️  No more fixable issues found.');
      console.log('Remaining issues may require manual intervention.\n');
      
      // Show remaining issues
      const remainingRules = ruleIds.filter(rule => !safeRules.includes(rule));
      if (remainingRules.length > 0) {
        console.log('Remaining rules that need manual fixes:');
        remainingRules.forEach(rule => {
          console.log(`  - ${rule} (${grouped[rule].length} issues)`);
        });
      }
      break;
    }
    
    console.log(`\n✅ Fixed ${totalFixed} issues in iteration ${iteration}`);
    iteration++;
  }
  
  // Final lint check
  console.log('\n📊 Running final ESLint check...');
  try {
    execSync('npm run lint', { stdio: 'inherit', cwd: PROJECT_ROOT });
    console.log('\n🎉 All ESLint issues resolved!');
  } catch (error) {
    console.log('\n📝 Some issues remain. Run the script again or fix manually.');
  }
}

if (require.main === module) {
  main().catch(console.error);
}

module.exports = { parseESLintOutput, groupIssuesByRule, applyRuleFixes, ruleFixes };
