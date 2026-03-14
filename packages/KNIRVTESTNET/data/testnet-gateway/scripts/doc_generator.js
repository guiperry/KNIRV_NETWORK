#!/usr/bin/env node

/**
 * Documentation Generator for KNIRV Network
 *
 * This script processes all documentation from the docs folder and transforms it into
 * an organized documentation system in a new documentation folder, with support for
 * both Docsify and Go embedding. Whitepapers are handled specially and placed in a
 * dedicated directory for direct access.
 *
 * Usage:
 *   node doc_generator.js
 *
 * The script will:
 * 1. Read all markdown files from the docs directory
 * 2. Process and organize them into a structured documentation system
 * 3. Handle whitepapers specially - copy them directly to a whitepapers directory
 * 4. Generate a sidebar and navigation structure
 * 5. Create Docsify-compatible output in the documentation folder
 * 6. Create a Go-embedding compatible structure
 */

const fs = require('fs');
const path = require('path');
const crypto = require('crypto');
const { execSync } = require('child_process');
const https = require('https');

// Load dotenv from the documentation folder
const dotenvPath = path.join(__dirname, '..', 'documentation', 'node_modules', 'dotenv');
const dotenv = require(dotenvPath);
dotenv.config({ path: path.join(__dirname, '..', 'documentation', '.env') });

// Configuration
const rootDir = path.dirname(__dirname); // Go up one level from scripts directory
const CONFIG = {
  sourceDir: path.join(rootDir, 'docs'),
  outputDir: path.join(rootDir, 'documentation'),
  docsifyDir: path.join(rootDir, 'documentation', 'docsify'),
  hashFile: path.join(rootDir, 'documentation', '.doc_hashes.json'),
  // Backup directory for consolidated .md files
  backupDir: path.join(rootDir, '.doc-consolidation-backups'),
  projectName: 'KNIRV Network',
  projectRepo: 'https://github.com/guiperry/KNIRV_NETWORK',
  categories: {
    'guides': 'User Guides',
    'deployment': 'Deployment',
    'development': 'Development',
    'api': 'API Reference',
    'security': 'Security',
    'architecture': 'Architecture',
    'contribute': 'How to Contribute',
    'legal': 'Legal Documents',
    // Subproduct categories
    'knirvchain': 'KNIRVCHAIN Documentation',
    'knirvgraph': 'KNIRVGRAPH Documentation',
    'knirvserver': 'KNIRVSERVER Documentation',
    'knirvoracle': 'KNIRVORACLE Documentation',
    'knirvrouter': 'KNIRVROUTER Documentation',
    'knirvsdk': 'KNIRVSDK Documentation',
    'knirvshell': 'KNIRVCORTEX Documentation',
    'knirvwallet': 'KNIRVWALLET Documentation'
  },
  // Special files that should be processed differently
  specialFiles: {
    'CODE_OF_CONDUCT.md': 'legal',
    'PRIVACY_POLICY.md': 'legal',
    'TERMS_AND_CONDITIONS.md': 'legal'
  },
  // Whitepapers directory - files here should be copied directly without refactoring
  whitepaperSourceDir: 'whitepapers',
  // Note: The 'docs' directory is excluded from .md file consolidation except for whitepapers
  // This prevents the script from processing its own output and source documentation
  // Subproduct directories to scan for additional documentation
  subproductDirs: [
    'KNIRVCHAIN',
    'KNIRVGRAPH',
    'KNIRVSERVER',
    'KNIRVORACLE',
    'KNIRVROUTER',
    'KNIRVSDK',
    'KNIRVCORTEX',
    'KNIRVWALLET',
    'KNIRVCLI',
    'KNIRVGATEWAY',
    'KNIRVTESTNET'
  ],
  // Subdirectories to exclude when scanning subproducts
  excludeSubDirs: [
    'node_modules',
    'target',
    'build',
    'dist',
    'coverage',
    'temp-extract',
    'release_assets',
    '.git',
    'vendor',
    'bin',
    '.venv',
    '.env',
    '__pycache__',
    '.pytest_cache',
    '.mypy_cache',
    '.tox',
    'venv',
    'env',
    '.aider.chat.history.md',
    '.zencoder',
    'documentation' // Exclude existing documentation directories to avoid duplication
  ],
  // File patterns to exclude
  excludeFilePatterns: [
    /^\.aider\.chat\.history\.md$/,
    /^LICENSE\.md$/,
    /template\.md$/,
    /site-packages/,
    /lib64/,
    /lib\/python/,
    /\.venv\//,
    /\.env\//
  ]
};

// Ensure directories exist
function ensureDirectoryExists(dir) {
  if (!fs.existsSync(dir)) {
    fs.mkdirSync(dir, { recursive: true });
    console.log(`Created directory: ${dir}`);
  }
}

// Clean output directory
function cleanDirectory(dir) {
  if (fs.existsSync(dir)) {
    // Instead of deleting the entire directory, we'll clear its contents
    // This approach is more idempotent as it doesn't change directory permissions or attributes
    const files = fs.readdirSync(dir);
    for (const file of files) {
      const filePath = path.join(dir, file);
      if (fs.lstatSync(filePath).isDirectory()) {
        cleanDirectory(filePath); // Recursively clean subdirectories
        fs.rmdirSync(filePath);   // Remove empty directory
      } else {
        fs.unlinkSync(filePath);  // Remove file
      }
    }
    console.log(`Cleaned contents of directory: ${dir}`);
  } else {
    ensureDirectoryExists(dir);
    console.log(`Created directory: ${dir}`);
  }
}

// Calculate hash of a file's content
function calculateFileHash(filePath) {
  const content = fs.readFileSync(filePath, 'utf8');
  return crypto.createHash('md5').update(content).digest('hex');
}

// Calculate hash of a string
function calculateStringHash(content) {
  return crypto.createHash('md5').update(content).digest('hex');
}

// Load existing hashes or create empty hash object
function loadHashes() {
  if (fs.existsSync(CONFIG.hashFile)) {
    try {
      return JSON.parse(fs.readFileSync(CONFIG.hashFile, 'utf8'));
    } catch (error) {
      console.log(`Error reading hash file: ${error.message}`);
      return {};
    }
  }
  return {};
}

// Save hashes to file
function saveHashes(hashes) {
  ensureDirectoryExists(path.dirname(CONFIG.hashFile));
  fs.writeFileSync(CONFIG.hashFile, JSON.stringify(hashes, null, 2));
}

// Check if file has changed by comparing hashes
function hasFileChanged(filePath, hashes) {
  const currentHash = calculateFileHash(filePath);
  const previousHash = hashes[filePath];
  return currentHash !== previousHash;
}

// Check if content has changed by comparing with stored hash
function hasContentChanged(content, key, hashes) {
  const currentHash = calculateStringHash(content);
  const previousHash = hashes[key];
  return currentHash !== previousHash;
}

// Update hash for a file
function updateFileHash(filePath, hashes) {
  hashes[filePath] = calculateFileHash(filePath);
  return hashes;
}

// Create comprehensive backup of files before consolidation
function createConsolidationBackup(dirInfo, timestamp) {
  console.log(`  💾 Creating backup for ${dirInfo.relativePath}...`);

  try {
    // Create timestamped backup directory
    const backupSubDir = path.join(CONFIG.backupDir, timestamp, dirInfo.relativePath);
    ensureDirectoryExists(backupSubDir);

    // Backup README.md if it exists
    const readmePath = path.join(dirInfo.directory, dirInfo.readmeFile);
    if (fs.existsSync(readmePath)) {
      const backupReadmePath = path.join(backupSubDir, dirInfo.readmeFile);
      fs.copyFileSync(readmePath, backupReadmePath);
      console.log(`    📄 Backed up ${dirInfo.readmeFile}`);
    }

    // Backup all other .md files
    for (const mdFile of dirInfo.otherMdFiles) {
      const sourcePath = path.join(dirInfo.directory, mdFile);
      const backupPath = path.join(backupSubDir, mdFile);
      if (fs.existsSync(sourcePath)) {
        fs.copyFileSync(sourcePath, backupPath);
        console.log(`    📄 Backed up ${mdFile}`);
      }
    }

    // Create backup metadata
    const metadata = {
      timestamp: timestamp,
      directory: dirInfo.relativePath,
      readmeFile: dirInfo.readmeFile,
      otherMdFiles: dirInfo.otherMdFiles,
      backupReason: 'Pre-consolidation backup',
      originalSizes: {}
    };

    // Record original file sizes
    if (fs.existsSync(readmePath)) {
      metadata.originalSizes[dirInfo.readmeFile] = fs.statSync(readmePath).size;
    }
    for (const mdFile of dirInfo.otherMdFiles) {
      const sourcePath = path.join(dirInfo.directory, mdFile);
      if (fs.existsSync(sourcePath)) {
        metadata.originalSizes[mdFile] = fs.statSync(sourcePath).size;
      }
    }

    fs.writeFileSync(path.join(backupSubDir, 'backup-metadata.json'), JSON.stringify(metadata, null, 2));

    console.log(`  ✅ Backup created successfully in ${backupSubDir}`);
    return backupSubDir;
  } catch (error) {
    console.log(`  ❌ Backup failed: ${error.message}`);
    throw new Error(`Backup failed for ${dirInfo.relativePath}: ${error.message}`);
  }
}

// Validate consolidated content before writing
function validateConsolidatedContent(originalContent, consolidatedContent, dirInfo) {
  console.log(`  🔍 Validating consolidated content for ${dirInfo.relativePath}...`);

  const validationResults = {
    isValid: true,
    warnings: [],
    errors: [],
    metrics: {}
  };

  // Check if consolidated content is significantly longer than original
  const originalLength = originalContent.length;
  const consolidatedLength = consolidatedContent.length;

  validationResults.metrics.originalLength = originalLength;
  validationResults.metrics.consolidatedLength = consolidatedLength;
  validationResults.metrics.lengthIncrease = consolidatedLength - originalLength;
  validationResults.metrics.lengthRatio = originalLength > 0 ? consolidatedLength / originalLength : 0;

  // Validation checks
  if (consolidatedLength < originalLength * 0.8) {
    validationResults.errors.push(`Consolidated content is significantly shorter than original (${consolidatedLength} vs ${originalLength} chars)`);
    validationResults.isValid = false;
  }

  if (consolidatedLength < 1000) {
    validationResults.warnings.push(`Consolidated content seems too short (${consolidatedLength} chars)`);
  }

  // Check for truncation indicators
  const truncationIndicators = [
    /\*KNIR$/,  // Truncated at end
    /\.\.\.$/, // Ends with ellipsis
    /[^.!?]$/, // Doesn't end with proper punctuation (but allow markdown)
  ];

  for (const indicator of truncationIndicators) {
    if (indicator.test(consolidatedContent.trim())) {
      validationResults.errors.push(`Content appears to be truncated (matches pattern: ${indicator})`);
      validationResults.isValid = false;
    }
  }

  // Check for essential markdown structure
  const hasHeaders = /^#+ /.test(consolidatedContent);
  if (!hasHeaders) {
    validationResults.warnings.push('No markdown headers found in consolidated content');
  }

  // Check for minimum content requirements
  const wordCount = consolidatedContent.split(/\s+/).length;
  validationResults.metrics.wordCount = wordCount;

  if (wordCount < 100) {
    validationResults.errors.push(`Content too short: only ${wordCount} words`);
    validationResults.isValid = false;
  }

  console.log(`    📊 Validation metrics:`, validationResults.metrics);

  if (validationResults.warnings.length > 0) {
    console.log(`    ⚠️ Warnings:`, validationResults.warnings);
  }

  if (validationResults.errors.length > 0) {
    console.log(`    ❌ Errors:`, validationResults.errors);
  }

  return validationResults;
}

// Safe file writing with validation
function safeWriteConsolidatedFile(filePath, content, dirInfo, backupDir) {
  console.log(`  💾 Safely writing consolidated file: ${path.basename(filePath)}...`);

  try {
    // Create temporary file first
    const tempPath = filePath + '.tmp';
    fs.writeFileSync(tempPath, content);

    // Verify the temporary file was written correctly
    const writtenContent = fs.readFileSync(tempPath, 'utf8');
    if (writtenContent !== content) {
      throw new Error('File content verification failed after writing');
    }

    if (writtenContent.length !== content.length) {
      throw new Error(`File length mismatch: expected ${content.length}, got ${writtenContent.length}`);
    }

    // Check for truncation in written file
    if (writtenContent.endsWith('*KNIR') || writtenContent.length < content.length * 0.95) {
      throw new Error('Written file appears to be truncated');
    }

    // If validation passes, move temp file to final location
    fs.renameSync(tempPath, filePath);

    console.log(`  ✅ File written successfully: ${path.basename(filePath)} (${content.length} chars)`);
    return true;
  } catch (error) {
    console.log(`  ❌ Safe write failed: ${error.message}`);

    // Clean up temp file if it exists
    const tempPath = filePath + '.tmp';
    if (fs.existsSync(tempPath)) {
      fs.unlinkSync(tempPath);
    }

    throw error;
  }
}

// Clean old backup directories (keep last 10)
function cleanOldBackups() {
  try {
    if (!fs.existsSync(CONFIG.backupDir)) {
      return;
    }

    const backupDirs = fs.readdirSync(CONFIG.backupDir)
      .filter(item => {
        const itemPath = path.join(CONFIG.backupDir, item);
        return fs.lstatSync(itemPath).isDirectory();
      })
      .map(dir => ({
        name: dir,
        path: path.join(CONFIG.backupDir, dir),
        mtime: fs.statSync(path.join(CONFIG.backupDir, dir)).mtime
      }))
      .sort((a, b) => b.mtime - a.mtime); // Sort by modification time, newest first

    // Keep only the 10 most recent backups
    const backupsToDelete = backupDirs.slice(10);

    for (const backup of backupsToDelete) {
      console.log(`🗑️ Cleaning old backup: ${backup.name}`);
      // Recursively delete old backup directory
      fs.rmSync(backup.path, { recursive: true, force: true });
    }

    if (backupsToDelete.length > 0) {
      console.log(`✅ Cleaned ${backupsToDelete.length} old backup directories`);
    }
  } catch (error) {
    console.log(`⚠️ Failed to clean old backups: ${error.message}`);
  }
}

// Update hash for content
function updateContentHash(content, key, hashes) {
  hashes[key] = calculateStringHash(content);
  return hashes;
}

// Generate a consistent footer with legal links
function generateLegalFooter(categories) {
  let footer = `\n\n---\n\n<div class="footer-links">\n`;

  // Add legal documents if they exist
  if (categories['legal'] && categories['legal'].length > 0) {
    categories['legal'].forEach(doc => {
      footer += `<a href="#/legal/${doc.filename}" class="footer-link">${doc.title}</a> | `;
    });
    // Remove the last separator
    footer = footer.slice(0, -3);
  }

  footer += `\n\n© ${new Date().getFullYear()} ${CONFIG.projectName}\n</div>\n`;

  return footer;
}

// Generate the standard footer for all documentation files
function generateStandardFooter() {
  return `\n\n<div class="footer-links">
<a href="#/legal/CODE_OF_CONDUCT" class="footer-link">Code of Conduct</a> | <a href="#/legal/PRIVACY_POLICY" class="footer-link">Privacy Policy</a> | <a href="#/legal/TERMS_AND_CONDITIONS" class="footer-link">Terms and Conditions</a>

© 2025 KNIRV Network
</div>\n`;
}

// AI-powered documentation organization functions
async function callGeminiAPI(prompt) {
  return new Promise((resolve, reject) => {
    const apiKey = process.env.GEMINI_API_KEY;
    const model = process.env.GEMINI_MODEL_NAME || 'gemini-1.5-flash';

    if (!apiKey) {
      reject(new Error('GEMINI_API_KEY not found in environment variables'));
      return;
    }

    const requestData = JSON.stringify({
      contents: [{
        role: "user",
        parts: [{ text: prompt }]
      }],
      generationConfig: {
        temperature: 0.3,
        maxOutputTokens: 4096
      }
    });

    const options = {
      hostname: 'generativelanguage.googleapis.com',
      path: `/v1beta/models/${model}:generateContent?key=${apiKey}`,
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Content-Length': Buffer.byteLength(requestData)
      }
    };

    const req = https.request(options, (res) => {
      let data = '';
      res.on('data', (chunk) => data += chunk);
      res.on('end', () => {
        try {
          const response = JSON.parse(data);
          if (response.candidates && response.candidates[0] && response.candidates[0].content) {
            resolve(response.candidates[0].content.parts[0].text);
          } else {
            reject(new Error('Invalid response from Gemini API'));
          }
        } catch (error) {
          reject(new Error(`Failed to parse Gemini response: ${error.message}`));
        }
      });
    });

    req.on('error', (error) => {
      reject(new Error(`Gemini API request failed: ${error.message}`));
    });

    req.write(requestData);
    req.end();
  });
}

async function callCerebrasAPI(prompt) {
  return new Promise((resolve, reject) => {
    const apiKey = process.env.CEREBRAS_API_KEY;
    const baseUrl = process.env.CEREBRAS_BASE_URL || 'https://api.cerebras.ai/v1/chat/completions';

    if (!apiKey) {
      reject(new Error('CEREBRAS_API_KEY not found in environment variables'));
      return;
    }

    const requestData = JSON.stringify({
      model: "llama3.1-8b",
      messages: [{
        role: "user",
        content: prompt
      }],
      max_tokens: 4096,
      temperature: 0.3
    });

    const url = new URL(baseUrl);
    const options = {
      hostname: url.hostname,
      path: url.pathname,
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${apiKey}`,
        'Content-Length': Buffer.byteLength(requestData)
      }
    };

    const req = https.request(options, (res) => {
      let data = '';
      res.on('data', (chunk) => data += chunk);
      res.on('end', () => {
        try {
          const response = JSON.parse(data);
          if (response.choices && response.choices[0] && response.choices[0].message) {
            resolve(response.choices[0].message.content);
          } else {
            reject(new Error('Invalid response from Cerebras API'));
          }
        } catch (error) {
          reject(new Error(`Failed to parse Cerebras response: ${error.message}`));
        }
      });
    });

    req.on('error', (error) => {
      reject(new Error(`Cerebras API request failed: ${error.message}`));
    });

    req.write(requestData);
    req.end();
  });
}

// Get all markdown files from a directory, excluding whitepapers
function getMarkdownFiles(dir, excludeWhitepapers = true) {
  const files = fs.readdirSync(dir);
  let markdownFiles = [];

  files.forEach(file => {
    const fullPath = path.join(dir, file);

    // Skip the docs/dev directory
    if (file === 'dev' && dir.endsWith('docs')) {
      console.log(`Skipping dev directory: ${fullPath}`);
      return;
    }

    // Skip whitepapers directory if excludeWhitepapers is true
    if (excludeWhitepapers && file === CONFIG.whitepaperSourceDir && dir.endsWith('docs')) {
      console.log(`Skipping whitepapers directory for regular processing: ${fullPath}`);
      return;
    }

    if (fs.statSync(fullPath).isDirectory()) {
      // Recursively get markdown files from subdirectories
      const subDirFiles = getMarkdownFiles(fullPath, excludeWhitepapers);
      markdownFiles = markdownFiles.concat(subDirFiles);
    } else if (file.endsWith('.md')) {
      markdownFiles.push(fullPath);
    }
  });

  return markdownFiles;
}

// Get whitepaper files specifically
function getWhitepaperFiles() {
  const whitepaperSourcePath = path.join(CONFIG.sourceDir, CONFIG.whitepaperSourceDir);

  if (!fs.existsSync(whitepaperSourcePath)) {
    console.log('No whitepapers directory found');
    return [];
  }

  const files = fs.readdirSync(whitepaperSourcePath);
  const whitepaperFiles = [];

  files.forEach(file => {
    const fullPath = path.join(whitepaperSourcePath, file);
    if (fs.statSync(fullPath).isFile() && file.endsWith('.md')) {
      whitepaperFiles.push(fullPath);
    }
  });

  return whitepaperFiles;
}

// Get markdown files from subproduct directories
function getSubproductMarkdownFiles() {
  const subproductFiles = [];

  CONFIG.subproductDirs.forEach(subproductDir => {
    const subproductPath = path.join(rootDir, subproductDir);

    if (!fs.existsSync(subproductPath)) {
      console.log(`Subproduct directory not found: ${subproductDir}`);
      return;
    }

    console.log(`Scanning subproduct: ${subproductDir}`);
    const files = getMarkdownFilesRecursive(subproductPath, subproductDir);
    subproductFiles.push(...files);
  });

  return subproductFiles;
}

// Recursively get markdown files from a directory, excluding certain subdirectories
function getMarkdownFilesRecursive(dir, subproductName, relativePath = '') {
  const files = [];

  try {
    const dirContents = fs.readdirSync(dir);

    dirContents.forEach(item => {
      const fullPath = path.join(dir, item);
      const currentRelativePath = path.join(relativePath, item);

      // Skip excluded directories
      if (CONFIG.excludeSubDirs.includes(item)) {
        return;
      }

      // Skip files that match exclusion patterns
      if (item.endsWith('.md')) {
        const shouldExclude = CONFIG.excludeFilePatterns.some(pattern => {
          if (pattern instanceof RegExp) {
            return pattern.test(currentRelativePath) || pattern.test(fullPath);
          }
          return currentRelativePath.includes(pattern) || fullPath.includes(pattern);
        });

        if (shouldExclude) {
          return;
        }
      }

      if (fs.statSync(fullPath).isDirectory()) {
        // Recursively scan subdirectories
        const subFiles = getMarkdownFilesRecursive(fullPath, subproductName, currentRelativePath);
        files.push(...subFiles);
      } else if (item.endsWith('.md')) {
        files.push({
          fullPath,
          subproductName,
          relativePath: currentRelativePath,
          filename: item
        });
      }
    });
  } catch (error) {
    console.warn(`Error reading directory ${dir}: ${error.message}`);
  }

  return files;
}

// Parse BibTeX file to extract citations
function parseBibTeX(bibPath) {
  if (!fs.existsSync(bibPath)) {
    return {};
  }
  
  const content = fs.readFileSync(bibPath, 'utf8');
  const citations = {};
  
  // Simple regex-based BibTeX parser
  // This is a basic implementation - a full parser would be more robust
  const entryRegex = /@(\w+)\s*{\s*([^,]+),\s*([\s\S]*?)\s*}\s*(?=@|\s*$)/g;
  const fieldRegex = /\s*(\w+)\s*=\s*{([^}]*)}/g;
  
  let match;
  while ((match = entryRegex.exec(content)) !== null) {
    const type = match[1];
    const key = match[2];
    const fieldsText = match[3];
    
    const fields = {};
    let fieldMatch;
    while ((fieldMatch = fieldRegex.exec(fieldsText)) !== null) {
      fields[fieldMatch[1].toLowerCase()] = fieldMatch[2];
    }
    
    citations[key] = {
      type,
      ...fields
    };
  }
  
  return citations;
}

// Format a citation in APA-like style
function formatCitation(citation) {
  if (!citation) {
    return '[Citation not found]';
  }
  
  try {
    const authors = citation.author ? citation.author.split(' and ').map(author => {
      const parts = author.split(',');
      if (parts.length > 1) {
        return `${parts[1].trim()} ${parts[0].trim()}`;
      }
      return author.trim();
    }).join(', ') : '';
    
    const year = citation.year || '';
    const title = citation.title || '';
    const journal = citation.journal || '';
    const volume = citation.volume || '';
    const number = citation.number ? `(${citation.number})` : '';
    const pages = citation.pages ? `pp. ${citation.pages}` : '';
    const publisher = citation.publisher || '';
    const url = citation.url || '';
    const link = citation.link || '';
    
    let formattedCitation = '';
    
    if (citation.type.toLowerCase() === 'article') {
      formattedCitation = `${authors} (${year}). ${title}. <em>${journal}</em>, ${volume}${number}, ${pages}.`;
    } else if (citation.type.toLowerCase() === 'book') {
      formattedCitation = `${authors} (${year}). <em>${title}</em>. ${publisher}.`;
    } else if (citation.type.toLowerCase() === 'inproceedings' || citation.type.toLowerCase() === 'conference') {
      formattedCitation = `${authors} (${year}). ${title}. In <em>${citation.booktitle || ''}</em>, ${pages}.`;
    } else {
      formattedCitation = `${authors} (${year}). ${title}.`;
    }
    
    // Add URL if available
    if (url) {
      formattedCitation += ` Retrieved from <a href="${url}" target="_blank">${url}</a>`;
    }
    
    // Add link to the paper/report if available
    if (link) {
      formattedCitation += ` <a href="${link}" target="_blank" class="citation-link">[View Paper]</a>`;
    }
    
    return formattedCitation;
  } catch (error) {
    console.error(`Error formatting citation: ${error.message}`);
    return '[Citation format error]';
  }
}

// Process citations in markdown content
function processCitations(content, citations) {
  if (!citations || Object.keys(citations).length === 0) {
    return content;
  }
  
  // Replace citation markers with formatted citations
  return content.replace(/\[cite:(\w+)\]/g, (match, citationKey) => {
    const citation = citations[citationKey];
    if (citation) {
      return formatCitation(citation);
    } else {
      console.warn(`Citation not found: ${citationKey}`);
      return `[Citation not found: ${citationKey}]`;
    }
  });
}

// Parse markdown file to extract metadata and content
function parseMarkdownFile(filePath, citations, subproductInfo = null) {
  const content = fs.readFileSync(filePath, 'utf8');

  // Process citations in the content if citations are provided
  const processedContent = citations ? processCitations(content, citations) : content;

  const lines = processedContent.split('\n');

  // Extract title from first heading
  let title = '';
  const titleMatch = processedContent.match(/^# (.*)/m);
  if (titleMatch) {
    title = titleMatch[1];
  } else {
    // Use filename as fallback
    title = path.basename(filePath, '.md')
      .replace(/_/g, ' ')
      .replace(/\b\w/g, l => l.toUpperCase());
  }

  // Determine category based on content or filename
  let category = 'guides'; // Default category

  // If this is a subproduct file, categorize it under the subproduct
  if (subproductInfo) {
    category = subproductInfo.subproductName.toLowerCase();

    // Add subproduct prefix to title if not already present
    if (!title.toLowerCase().includes(subproductInfo.subproductName.toLowerCase())) {
      title = `${subproductInfo.subproductName}: ${title}`;
    }
  } else {
    // Check for special files first
    const filename = path.basename(filePath);
    if (CONFIG.specialFiles[filename]) {
      category = CONFIG.specialFiles[filename];
    }
    // Check for other specific files
    else if (filename === 'KEY_MANAGEMENT.md') {
      category = 'security';
    } else if (filename === 'CONTRIBUTE.md') {
      category = 'contribute';
    } else if (filePath.toLowerCase().includes('contribute') || processedContent.toLowerCase().includes('how to contribute')) {
      category = 'contribute';
    } else if (filePath.toLowerCase().includes('deploy') || processedContent.toLowerCase().includes('deployment')) {
      category = 'deployment';
    } else if (filePath.toLowerCase().includes('voice') || processedContent.toLowerCase().includes('tts')) {
      category = 'voice';
    } else if (filePath.toLowerCase().includes('key') || processedContent.toLowerCase().includes('security')) {
      category = 'security';
    } else if (filePath.toLowerCase().includes('embed') || processedContent.toLowerCase().includes('development')) {
      category = 'development';
    } else if (filePath.toLowerCase().includes('api') || processedContent.toLowerCase().includes('api reference')) {
      category = 'api';
    }
  }

  // Extract description from first paragraph after title
  let description = '';
  const paragraphs = processedContent.split('\n\n');
  for (let i = 0; i < paragraphs.length; i++) {
    if (!paragraphs[i].startsWith('#') && paragraphs[i].trim() !== '') {
      description = paragraphs[i].replace(/\n/g, ' ').trim();
      if (description.length > 160) {
        description = description.substring(0, 157) + '...';
      }
      break;
    }
  }

  return {
    title,
    category,
    description,
    content: processedContent,
    filename: path.basename(filePath),
    originalPath: filePath,
    subproductInfo: subproductInfo || null
  };
}

// Helper function to determine category from subproduct name
function determineCategory(subproductName) {
  const categoryMap = {
    'KNIRVCHAIN': 'knirvchain',
    'KNIRVGRAPH': 'knirvgraph',
    'KNIRVSERVER': 'knirvserver',
    'KNIRVORACLE': 'knirvoracle',
    'KNIRVROUTER': 'knirvrouter',
    'KNIRVSDK': 'knirvsdk',
    'KNIRVCORTEX': 'knirvshell',
    'KNIRVWALLET': 'knirvwallet',
    'KNIRVCLI': 'knirvshell',
    'KNIRVGATEWAY': 'guides',
    'KNIRVTESTNET': 'deployment'
  };
  return categoryMap[subproductName] || 'guides';
}

// AI-powered documentation organization and processing (README.md files in sub-project directories only)
async function organizeDocumentationWithAI(hashes) {
  console.log('Starting AI-powered documentation organization for sub-project README.md files...');

  // Get README.md files from sub-project directories
  const readmeFiles = getSubprojectReadmeFiles();

  if (readmeFiles.length === 0) {
    console.log('No README.md files found in sub-project directories to organize');
    return [];
  }

  const organizedDocs = [];

  for (const fileInfo of readmeFiles) {
    const filePath = fileInfo.fullPath;

    // Check if source file has changed - if not, skip AI processing
    if (!hasFileChanged(filePath, hashes)) {
      console.log(`Skipping ${fileInfo.subproductName}/README.md - no changes detected`);

      // Try to load existing processed file
      const outputCategory = determineCategory(fileInfo.subproductName);
      const outputPath = path.join(CONFIG.docsifyDir, outputCategory, 'README.md');

      if (fs.existsSync(outputPath)) {
        const existingContent = fs.readFileSync(outputPath, 'utf8');
        const titleMatch = existingContent.match(/^# (.*)/m);
        const title = titleMatch ? titleMatch[1] : `${fileInfo.subproductName} User Guide`;

        organizedDocs.push({
          title: title,
          category: outputCategory,
          description: `User guide for ${fileInfo.subproductName}`,
          content: existingContent,
          filename: 'README.md',
          originalPath: filePath,
          qualityScore: 8 // Default for existing files
        });

        console.log(`Loaded existing processed file for ${fileInfo.subproductName}`);
        continue;
      }
    }

    // Update hash for changed file
    updateFileHash(filePath, hashes);

    // Check if source file has changed - if not, skip AI processing
    if (!hasFileChanged(filePath, hashes)) {
      console.log(`Skipping ${fileInfo.subproductName}/README.md - no changes detected`);

      // Try to load existing processed file
      const outputCategory = determineCategory(fileInfo.subproductName);
      const outputPath = path.join(CONFIG.docsifyDir, outputCategory, 'README.md');

      if (fs.existsSync(outputPath)) {
        const existingContent = fs.readFileSync(outputPath, 'utf8');
        const titleMatch = existingContent.match(/^# (.*)/m);
        const title = titleMatch ? titleMatch[1] : `${fileInfo.subproductName} User Guide`;

        organizedDocs.push({
          title: title,
          category: outputCategory,
          description: `User guide for ${fileInfo.subproductName}`,
          content: existingContent,
          filename: 'README.md',
          originalPath: filePath,
          qualityScore: 8 // Default for existing files
        });

        console.log(`Loaded existing processed file for ${fileInfo.subproductName}`);
        continue;
      }
    }

    // Update hash for changed file
    updateFileHash(filePath, hashes);
    try {
      console.log(`Processing: ${path.basename(filePath)}`);

      // Read file content
      const content = fs.readFileSync(filePath, 'utf8');

      // First pass: Gemini for initial organization and filtering
      const geminiPrompt = `
You are a technical documentation organizer for the KNIRV Network project. Analyze this README.md file from the ${fileInfo.subproductName} sub-project and transform it into user-friendly documentation:

1. CATEGORY: Choose the most appropriate category from: guides, deployment, development, api, security, architecture, contribute, legal, knirvchain, knirvgraph, knirvserver, knirvoracle, knirvrouter, knirvsdk, knirvshell, knirvwallet
2. TITLE: A clear, user-friendly title (e.g., "${fileInfo.subproductName} User Guide" or "${fileInfo.subproductName} Troubleshooting Guide")
3. DESCRIPTION: A brief 1-2 sentence description (max 160 characters)
4. PRIVACY_LEVEL: Either "PUBLIC" (safe for external users) or "PRIVATE" (contains admin-only or sensitive information)
5. ORGANIZED_CONTENT: Transform the content into user-friendly guidelines, troubleshooting tips, and usage instructions. Focus on:
   - Installation and setup instructions
   - Common usage patterns
   - Troubleshooting common issues
   - Configuration guidelines
   - Remove or simplify technical implementation details
   - Make it accessible to end users and developers

Sub-project: ${fileInfo.subproductName}
Document content:
${content}

Respond in this exact format:
CATEGORY: [category]
TITLE: [title]
DESCRIPTION: [description]
PRIVACY_LEVEL: [PUBLIC/PRIVATE]
ORGANIZED_CONTENT:
[organized content here]
`;

      const geminiResponse = await callGeminiAPI(geminiPrompt);

      // Parse Gemini response
      const categoryMatch = geminiResponse.match(/CATEGORY:\s*(.+)/);
      const titleMatch = geminiResponse.match(/TITLE:\s*(.+)/);
      const descriptionMatch = geminiResponse.match(/DESCRIPTION:\s*(.+)/);
      const privacyMatch = geminiResponse.match(/PRIVACY_LEVEL:\s*(PUBLIC|PRIVATE)/);
      const contentMatch = geminiResponse.match(/ORGANIZED_CONTENT:\s*([\s\S]+)/);

      if (!categoryMatch || !titleMatch || !privacyMatch || !contentMatch) {
        console.warn(`Failed to parse Gemini response for ${path.basename(filePath)}, skipping AI processing`);
        continue;
      }

      const category = categoryMatch[1].trim();
      const title = titleMatch[1].trim();
      const description = descriptionMatch ? descriptionMatch[1].trim() : '';
      const privacyLevel = privacyMatch[1].trim();
      let organizedContent = contentMatch[1].trim();

      // Skip private documents
      if (privacyLevel === 'PRIVATE') {
        console.log(`Skipping private document: ${path.basename(filePath)}`);
        continue;
      }

      // Second pass: Cerebras for quality check and final refinement
      const cerebrasPrompt = `
Review this user-focused documentation for the KNIRV Network ${fileInfo.subproductName} component. Evaluate and improve:

1. USER-FRIENDLINESS: Is this accessible to end users and developers?
2. PRACTICAL VALUE: Does it provide actionable guidance and troubleshooting?
3. COMPLETENESS: Are installation, usage, and troubleshooting covered?
4. CLARITY: Are technical concepts explained clearly?

Focus on making this a practical guide that helps users successfully use ${fileInfo.subproductName}.

Original category: ${category}
Original title: ${title}
Content to review:
${organizedContent}

Respond in this exact format:
QUALITY_SCORE: [1-10]
CATEGORY_CORRECT: [YES/NO]
SUGGESTED_CATEGORY: [category if different]
TITLE_CORRECT: [YES/NO]
SUGGESTED_TITLE: [title if different]
IMPROVEMENTS_NEEDED: [YES/NO]
FINAL_CONTENT:
[improved content here]
`;

      const cerebrasResponse = await callCerebrasAPI(cerebrasPrompt);

      // Parse Cerebras response
      const qualityMatch = cerebrasResponse.match(/QUALITY_SCORE:\s*(\d+)/);
      const categoryCorrectMatch = cerebrasResponse.match(/CATEGORY_CORRECT:\s*(YES|NO)/);
      const suggestedCategoryMatch = cerebrasResponse.match(/SUGGESTED_CATEGORY:\s*(.+)/);
      const titleCorrectMatch = cerebrasResponse.match(/TITLE_CORRECT:\s*(YES|NO)/);
      const suggestedTitleMatch = cerebrasResponse.match(/SUGGESTED_TITLE:\s*(.+)/);
      const finalContentMatch = cerebrasResponse.match(/FINAL_CONTENT:\s*([\s\S]+)/);

      // Apply Cerebras suggestions if available
      let finalCategory = category;
      let finalTitle = title;
      let finalContent = organizedContent;

      if (categoryCorrectMatch && categoryCorrectMatch[1] === 'NO' && suggestedCategoryMatch) {
        finalCategory = suggestedCategoryMatch[1].trim();
        console.log(`Updated category for ${path.basename(filePath)}: ${category} -> ${finalCategory}`);
      }

      if (titleCorrectMatch && titleCorrectMatch[1] === 'NO' && suggestedTitleMatch) {
        finalTitle = suggestedTitleMatch[1].trim();
        console.log(`Updated title for ${path.basename(filePath)}: ${title} -> ${finalTitle}`);
      }

      if (finalContentMatch) {
        finalContent = finalContentMatch[1].trim();
      }

      // Add standard footer to the content
      finalContent += generateStandardFooter();

      organizedDocs.push({
        title: finalTitle,
        category: finalCategory,
        description: description,
        content: finalContent,
        filename: path.basename(filePath),
        originalPath: filePath,
        qualityScore: qualityMatch ? parseInt(qualityMatch[1]) : 5
      });

      console.log(`Successfully processed: ${path.basename(filePath)} (Category: ${finalCategory}, Quality: ${qualityMatch ? qualityMatch[1] : 'N/A'})`);

    } catch (error) {
      console.error(`Error processing ${path.basename(filePath)}: ${error.message}`);
      // Continue with next file
    }
  }

  console.log(`AI organization complete. Processed ${organizedDocs.length} documents.`);
  return organizedDocs;
}

// Get README.md files from sub-project directories
function getSubprojectReadmeFiles() {
  const readmeFiles = [];

  CONFIG.subproductDirs.forEach(subproductDir => {
    const subproductPath = path.join(rootDir, subproductDir);
    const readmePath = path.join(subproductPath, 'README.md');

    if (fs.existsSync(readmePath)) {
      readmeFiles.push({
        fullPath: readmePath,
        subproductName: subproductDir,
        filename: 'README.md'
      });
      console.log(`Found README.md in ${subproductDir}`);
    } else {
      console.log(`No README.md found in ${subproductDir}`);
    }
  });

  return readmeFiles;
}

// Process deterministic documentation files (whitepapers only)
function processDeterministicMarkdownFiles() {
  console.log('Processing deterministic documentation files (whitepapers only)...');

  // Only process whitepapers deterministically - no other docs from docs/ folder
  console.log('Skipping docs/ folder files - only processing whitepapers deterministically');
  return [];
}

// Process all markdown files (AI organization for sub-project README.md files only)
async function processMarkdownFiles(hashes) {
  // Process deterministic files first (empty array - only whitepapers processed separately)
  const deterministicDocs = processDeterministicMarkdownFiles();

  // Process sub-project README.md files with AI
  const aiOrganizedDocs = await organizeDocumentationWithAI(hashes);

  // Combine both sets of documents
  return [...deterministicDocs, ...aiOrganizedDocs];
}

// Generate sidebar content for Docsify with organized documentation
function generateSidebar(docs) {
  let sidebar = `# ${CONFIG.projectName}\n\n`;

  // Group documents by category
  const categorizedDocs = {};
  docs.forEach(doc => {
    if (!categorizedDocs[doc.category]) {
      categorizedDocs[doc.category] = [];
    }
    categorizedDocs[doc.category].push(doc);
  });

  // Add organized documentation sections
  Object.keys(CONFIG.categories).forEach(categoryKey => {
    if (categorizedDocs[categoryKey] && categorizedDocs[categoryKey].length > 0) {
      const categoryTitle = CONFIG.categories[categoryKey];
      sidebar += `## ${categoryTitle}\n\n`;

      categorizedDocs[categoryKey].forEach(doc => {
        const filename = doc.filename.replace('.md', '');
        sidebar += `* [${doc.title}](${categoryKey}/${filename}.md)\n`;
      });
      sidebar += '\n';
    }
  });

  // Add whitepapers section
  sidebar += `## 📄 Whitepapers\n\n`;
  sidebar += `* [📚 View All Whitepapers](whitepapers/)\n\n`;

  // Add footer
  sidebar += `<div class="sidebar-footer">\n\n---\n\n`;
  sidebar += `\n© ${new Date().getFullYear()} ${CONFIG.projectName}\n</div>\n`;

  return sidebar;
}

// Generate index content for Docsify with organized documentation
function generateIndex(docs) {
  let index = `# ${CONFIG.projectName} Documentation\n\n`;

  // Add description
  index += `Welcome to the ${CONFIG.projectName} documentation. This comprehensive guide provides information about the KNIRV Decentralized Trusted Execution Network (D-TEN) and its components.\n\n`;

  // Group documents by category for overview
  const categorizedDocs = {};
  docs.forEach(doc => {
    if (!categorizedDocs[doc.category]) {
      categorizedDocs[doc.category] = [];
    }
    categorizedDocs[doc.category].push(doc);
  });

  // Add documentation sections overview
  if (Object.keys(categorizedDocs).length > 0) {
    index += `## 📚 Documentation Sections\n\n`;

    Object.keys(CONFIG.categories).forEach(categoryKey => {
      if (categorizedDocs[categoryKey] && categorizedDocs[categoryKey].length > 0) {
        const categoryTitle = CONFIG.categories[categoryKey];
        index += `### ${categoryTitle}\n\n`;

        categorizedDocs[categoryKey].forEach(doc => {
          const filename = doc.filename.replace('.md', '');
          index += `* [${doc.title}](${categoryKey}/${filename}.md)`;
          if (doc.description) {
            index += ` - ${doc.description}`;
          }
          index += '\n';
        });
        index += '\n';
      }
    });
  }

  // Add whitepapers section
  index += `## 📄 Technical Whitepapers\n\n`;
  index += `The KNIRV Network consists of multiple interconnected components, each with detailed technical specifications:\n\n`;
  index += `* [📚 **View All Whitepapers**](whitepapers/) - Complete collection of technical whitepapers\n\n`;
  index += `The whitepapers provide in-depth technical details about each component of the KNIRV D-TEN, including architecture, consensus mechanisms, and implementation specifications.\n\n`;

  // Add developer community message
  index += `## 🚀 Developer Community\n\n`;
  index += `We're building an open source developer community around the KNIRV Network. If you're interested in contributing, please check out our contribution guidelines.\n\n`;

  // Add standard footer
  index += generateStandardFooter();

  return index;
}

// Generate Docsify configuration
function generateDocsifyConfig() {
  return `
<!DOCTYPE html>
<html>
<head>
  <meta http-equiv="X-UA-Compatible" content="IE=edge,chrome=1">
  <meta name="viewport" content="width=device-width,initial-scale=1">
  <meta charset="UTF-8">
  <title>${CONFIG.projectName} Documentation</title>
  
  <!-- Favicon -->
  <link rel="apple-touch-icon" sizes="180x180" href="/static/assets/favicon/apple-touch-icon.png">
  <link rel="icon" type="image/png" sizes="32x32" href="/static/assets/favicon/favicon-32x32.png">
  <link rel="icon" type="image/png" sizes="16x16" href="/static/assets/favicon/favicon-16x16.png">
  <link rel="manifest" href="/static/assets/favicon/site.webmanifest">
  <link rel="shortcut icon" href="/static/assets/favicon/favicon.ico">
  <meta name="theme-color" content="#6a0dad">
  <meta name="msapplication-TileColor" content="#6a0dad">
  <meta name="msapplication-config" content="/static/assets/favicon/browserconfig.xml">
  
  <link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/docsify@4/themes/dark.css">
  <style>
    :root {
      --theme-color: #4a9eff;
      --sidebar-nav-link-color--active: #6fb5ff;
      --sidebar-nav-link-border-color--active: #6fb5ff;
    }
    body {
      background-color: #0a0a0a;
      color: #e0e0e0;
    }
      .sidebar:before {
      content: '';
      display: block;
      width: 2rem;
      height: 2rem;
      margin: 1rem auto 1rem 0.5rem;
    
    }
    .sidebar {
      background-color: #1a1a2e;
      border-right: 1px solid #333;
      padding-left: 5px;
    }
    .sidebar-toggle {
      background-color: rgba(26, 26, 46, 0.8);
    }
    .search input {
      background-color: #16213e;
      color: #e0e0e0;
      border: 1px solid #333;
    }
    .markdown-section code {
      background-color: #16213e;
    }
    .markdown-section pre {
      background-color: #16213e;
    }
    .markdown-section pre > code {
      background-color: #16213e;
    }
    .markdown-section blockquote {
      border-left: 4px solid #4a9eff;
      background-color: #16213e;
    }
    .markdown-section a {
      color: #4a9eff;
    }
    .markdown-section a:hover {
      color: #6fb5ff;
      text-decoration: underline;
    }
    .citation-link {
      display: inline-block;
      margin-left: 8px;
      padding: 2px 8px;
      background-color: #4a9eff;
      color: #fff;
      border-radius: 4px;
      font-size: 0.8em;
      text-decoration: none;
      transition: background-color 0.3s ease;
    }
    .citation-link:hover {
      background-color: #6fb5ff;
      color: #fff;
      text-decoration: none;
    }
    .sidebar-footer {
      margin-top: 2rem;
      padding-top: 1rem;
      border-top: 1px solid #333;
      font-size: 0.85rem;
      opacity: 0.8;
    }
    .footer-links {
      margin-top: 2rem;
      padding-top: 1rem;
      border-top: 1px solid #333;
      font-size: 0.85rem;
      opacity: 0.8;
      text-align: center;
    }
    .footer-link {
      color: #4a9eff;
      margin: 0 0.5rem;
      text-decoration: none;
    }
    .footer-link:hover {
      text-decoration: underline;
    }
    /* Mermaid diagram styling */
    .mermaid {
      text-align: center;
      margin: 1rem 0;
    }
    .mermaid svg {
      max-width: 100%;
      height: auto;
    }
  </style>
</head>
<body>
  <div id="app"></div>
  <script>
    window.$docsify = {
      name: '${CONFIG.projectName}',
      repo: '${CONFIG.projectRepo}',
      loadSidebar: true,
      subMaxLevel: 3,
      auto2top: true,
      relativePath: false,
      themeColor: '#4a9eff',
      search: {
        maxAge: 86400000,
        paths: 'auto',
        placeholder: 'Search',
        noData: 'No results found',
        depth: 6
      },
      plugins: [
        function(hook, vm) {
          hook.ready(function() {
            mermaid.initialize({ startOnLoad: true, theme: 'dark' });
          });
        }
      ]
    }
  </script>
  <script src="https://cdn.jsdelivr.net/npm/docsify@4"></script>
  <script src="https://cdn.jsdelivr.net/npm/docsify@4/lib/plugins/search.min.js"></script>
  <script src="https://cdn.jsdelivr.net/npm/mermaid@10/dist/mermaid.min.js"></script>
  <script src="https://unpkg.com/docsify-mermaid@2/dist/docsify-mermaid.js"></script>
  <script src="https://cdn.jsdelivr.net/npm/prismjs@1/components/prism-bash.min.js"></script>
  <script src="https://cdn.jsdelivr.net/npm/prismjs@1/components/prism-go.min.js"></script>
  <script src="https://cdn.jsdelivr.net/npm/prismjs@1/components/prism-yaml.min.js"></script>
  <script src="https://cdn.jsdelivr.net/npm/prismjs@1/components/prism-rust.min.js"></script>
  <script src="https://cdn.jsdelivr.net/npm/prismjs@1/components/prism-javascript.min.js"></script>
  <script src="https://cdn.jsdelivr.net/npm/prismjs@1/components/prism-typescript.min.js"></script>
</body>
</html>
  `;
}



// Write file if content has changed
function writeFileIfChanged(filePath, content, hashes) {
  const fileKey = `content:${filePath}`;
  
  if (!fs.existsSync(filePath) || hasContentChanged(content, fileKey, hashes)) {
    ensureDirectoryExists(path.dirname(filePath));
    fs.writeFileSync(filePath, content);
    console.log(`Updated: ${path.relative(rootDir, filePath)}`);
    updateContentHash(content, fileKey, hashes);
    return true;
  }
  
  return false;
}

// Create Docsify structure with organized documentation
function createDocsifyStructure(docs, hashes) {
  ensureDirectoryExists(CONFIG.docsifyDir);
  let filesChanged = 0;

  // Create .nojekyll file (required for GitHub Pages and proper Docsify functionality)
  const nojekyllPath = path.join(CONFIG.docsifyDir, '.nojekyll');
  if (writeFileIfChanged(nojekyllPath, '', hashes)) {
    filesChanged++;
  }

  // Create index.html
  const indexHtmlPath = path.join(CONFIG.docsifyDir, 'index.html');
  const indexHtmlContent = generateDocsifyConfig();
  if (writeFileIfChanged(indexHtmlPath, indexHtmlContent, hashes)) {
    filesChanged++;
  }

  // Create README.md (homepage)
  const readmePath = path.join(CONFIG.docsifyDir, 'README.md');
  const readmeContent = generateIndex(docs);
  if (writeFileIfChanged(readmePath, readmeContent, hashes)) {
    filesChanged++;
  }

  // Create _sidebar.md
  const sidebarPath = path.join(CONFIG.docsifyDir, '_sidebar.md');
  const sidebarContent = generateSidebar(docs);
  if (writeFileIfChanged(sidebarPath, sidebarContent, hashes)) {
    filesChanged++;
  }

  // Create organized documentation files by category
  const categorizedDocs = {};
  docs.forEach(doc => {
    if (!categorizedDocs[doc.category]) {
      categorizedDocs[doc.category] = [];
    }
    categorizedDocs[doc.category].push(doc);
  });

  // Write documentation files to their respective category directories
  Object.keys(categorizedDocs).forEach(category => {
    const categoryDir = path.join(CONFIG.docsifyDir, category);
    ensureDirectoryExists(categoryDir);

    categorizedDocs[category].forEach(doc => {
      const outputPath = path.join(categoryDir, doc.filename);
      if (writeFileIfChanged(outputPath, doc.content, hashes)) {
        filesChanged++;
      }
    });
  });

  console.log(`Docsify documentation structure: ${filesChanged} files updated`);
}



// Process whitepapers - copy them only to docsify directory with footer
function processWhitepapers(hashes) {
  const whitepaperFiles = getWhitepaperFiles();

  if (whitepaperFiles.length === 0) {
    console.log('No whitepapers found to process');
    return;
  }

  // Ensure docsify whitepapers directory exists
  const docsifyWhitepaperDir = path.join(CONFIG.docsifyDir, 'whitepapers');
  ensureDirectoryExists(docsifyWhitepaperDir);

  let filesChanged = 0;

  whitepaperFiles.forEach(filePath => {
    let content = fs.readFileSync(filePath, 'utf8');
    const filename = path.basename(filePath);

    // Add standard footer to whitepaper content
    content += generateStandardFooter();

    // Copy to docsify whitepapers directory only
    const docsifyOutputPath = path.join(docsifyWhitepaperDir, filename);
    if (writeFileIfChanged(docsifyOutputPath, content, hashes)) {
      filesChanged++;
    }

    console.log(`Processed whitepaper: ${filename}`);
  });

  // Generate an index file for docsify whitepapers
  let docsifyIndexContent = '# KNIRV Network Whitepapers\n\n';
  docsifyIndexContent += 'This section contains the technical whitepapers for the KNIRV Network components.\n\n';
  docsifyIndexContent += '## Available Whitepapers\n\n';

  whitepaperFiles.forEach(filePath => {
    const filename = path.basename(filePath);
    const title = filename.replace('.md', '').replace(/_/g, ' ');
    const linkName = filename.replace('.md', ''); // Remove .md extension for Docsify links
    docsifyIndexContent += `* [${title}](whitepapers/${linkName})\n`;
  });

  docsifyIndexContent += '\n---\n\n';
  docsifyIndexContent += `© ${new Date().getFullYear()} ${CONFIG.projectName}\n`;

  const docsifyIndexPath = path.join(docsifyWhitepaperDir, 'README.md');
  if (writeFileIfChanged(docsifyIndexPath, docsifyIndexContent, hashes)) {
    filesChanged++;
  }

  console.log(`Whitepapers: ${filesChanged} files updated`);
}

// Process references.bib if it exists
function processReferences(hashes) {
  const bibPath = path.join(CONFIG.sourceDir, 'references.bib');
  if (fs.existsSync(bibPath)) {
    const bibContent = fs.readFileSync(bibPath, 'utf8');

    // Copy to output directory if changed
    const docsifyBibPath = path.join(CONFIG.docsifyDir, 'references.bib');

    let filesChanged = 0;

    if (writeFileIfChanged(docsifyBibPath, bibContent, hashes)) {
      filesChanged++;
    }

    // Generate a references page with all citations
    const citations = parseBibTeX(bibPath);
    if (Object.keys(citations).length > 0) {
      let referencesContent = '# References\n\n';
      referencesContent += 'This page lists all references used in the documentation.\n\n';

      // Sort citations by key
      const sortedKeys = Object.keys(citations).sort();

      sortedKeys.forEach(key => {
        const citation = citations[key];
        referencesContent += `## ${key}\n\n`;
        referencesContent += `${formatCitation(citation)}\n\n`;
        referencesContent += `**Citation Key**: \`[cite:${key}]\`\n\n`;
        referencesContent += '---\n\n';
      });

      // Write references page to output directory
      const docsifyReferencesPath = path.join(CONFIG.docsifyDir, 'references.md');

      if (writeFileIfChanged(docsifyReferencesPath, referencesContent, hashes)) {
        filesChanged++;
      }

      console.log(`Generated references page with ${Object.keys(citations).length} citations`);
    }

    if (filesChanged > 0) {
      console.log(`References: ${filesChanged} files updated`);
    } else {
      console.log('References: No changes detected');
    }
  }
}

// Comprehensive .md file consolidation across all subproject directories
async function consolidateAllProjectDocumentation() {
  console.log('🔄 Starting comprehensive .md file consolidation across all subproject directories...');

  const projectRoot = rootDir;
  const consolidationResults = [];

  // Ensure backup directory exists and clean old backups
  ensureDirectoryExists(CONFIG.backupDir);
  cleanOldBackups();

  // Function to find all directories with multiple .md files (excluding docs directory)
  function findDirectoriesWithMultipleMdFiles(dir, depth = 0) {
    if (depth > 3) return []; // Limit recursion depth

    const results = [];
    const relativePath = path.relative(projectRoot, dir);

    // Skip the docs directory entirely (except for whitepapers which are handled separately)
    if (relativePath === 'docs' || relativePath.startsWith('docs/')) {
      console.log(`Skipping docs directory: ${relativePath}`);
      return results;
    }

    try {
      const items = fs.readdirSync(dir);
      const mdFiles = items.filter(item => {
        const itemPath = path.join(dir, item);
        return fs.lstatSync(itemPath).isFile() && item.endsWith('.md');
      });

      // If this directory has multiple .md files, it's a candidate for consolidation
      if (mdFiles.length > 1) {
        const hasReadme = mdFiles.some(file => file.toLowerCase() === 'readme.md');
        const otherMdFiles = mdFiles.filter(file => file.toLowerCase() !== 'readme.md');

        if (hasReadme && otherMdFiles.length > 0) {
          results.push({
            directory: dir,
            relativePath: relativePath,
            readmeFile: mdFiles.find(file => file.toLowerCase() === 'readme.md'),
            otherMdFiles: otherMdFiles,
            allMdFiles: mdFiles
          });
        }
      }

      // Recursively check subdirectories
      for (const item of items) {
        const itemPath = path.join(dir, item);
        const stat = fs.lstatSync(itemPath);

        if (stat.isDirectory() && !CONFIG.excludeSubDirs.includes(item) && !item.startsWith('.')) {
          // Additional check to skip docs directory
          const subRelativePath = path.relative(projectRoot, itemPath);
          if (subRelativePath !== 'docs' && !subRelativePath.startsWith('docs/')) {
            results.push(...findDirectoriesWithMultipleMdFiles(itemPath, depth + 1));
          }
        }
      }
    } catch (error) {
      console.log(`Skipping directory ${dir}: ${error.message}`);
    }

    return results;
  }

  // Find all directories that need consolidation
  const candidateDirectories = findDirectoriesWithMultipleMdFiles(projectRoot);

  console.log(`Found ${candidateDirectories.length} directories with multiple .md files:`);
  candidateDirectories.forEach(dir => {
    console.log(`  📁 ${dir.relativePath}: README.md + ${dir.otherMdFiles.length} other .md files`);
  });

  // Process each directory for consolidation
  for (const dirInfo of candidateDirectories) {
    await consolidateDirectoryDocumentation(dirInfo);
    consolidationResults.push(dirInfo);
  }

  return consolidationResults;
}

// Consolidate .md files in a specific directory using AI with comprehensive safety checks
async function consolidateDirectoryDocumentation(dirInfo) {
  console.log(`\n🔄 Consolidating documentation in: ${dirInfo.relativePath}`);

  const readmePath = path.join(dirInfo.directory, dirInfo.readmeFile);
  const otherMdPaths = dirInfo.otherMdFiles.map(file => path.join(dirInfo.directory, file));

  // Create timestamped backup before any changes
  const timestamp = new Date().toISOString().replace(/[:.]/g, '-');
  let backupDir;

  try {
    backupDir = createConsolidationBackup(dirInfo, timestamp);
  } catch (error) {
    console.log(`  ❌ Backup failed, skipping consolidation for safety: ${error.message}`);
    return;
  }

  // Read all .md file contents
  const readmeContent = fs.existsSync(readmePath) ? fs.readFileSync(readmePath, 'utf8') : '';
  const otherContents = otherMdPaths.map(filePath => ({
    filename: path.basename(filePath),
    content: fs.readFileSync(filePath, 'utf8'),
    path: filePath
  }));

  // Check if consolidation has already been done
  const hasConsolidationMarkers = otherContents.some(file =>
    readmeContent.includes(file.filename.replace('.md', '')) ||
    readmeContent.includes(file.content.substring(0, 100))
  );

  if (hasConsolidationMarkers && readmeContent.length > 5000) {
    console.log(`  ✅ ${dirInfo.relativePath} already appears to be consolidated`);

    // Validate existing content before removing files
    const validation = validateConsolidatedContent(readmeContent, readmeContent, dirInfo);
    if (!validation.isValid) {
      console.log(`  ⚠️ Existing consolidated content failed validation, keeping individual files`);
      return;
    }

    // Remove the individual .md files since they're consolidated and validated
    for (const fileInfo of otherContents) {
      if (fs.existsSync(fileInfo.path)) {
        console.log(`    🗑️ Removing consolidated file: ${fileInfo.filename}`);
        fs.unlinkSync(fileInfo.path);
      }
    }
    return;
  }

  // Use AI to intelligently consolidate the documentation
  console.log(`  🤖 Generating consolidated documentation...`);
  const consolidatedContent = await generateConsolidatedDocumentation(
    dirInfo.relativePath,
    readmeContent,
    otherContents
  );

  if (!consolidatedContent) {
    console.log(`  ❌ Failed to generate consolidated content for ${dirInfo.relativePath}`);
    return;
  }

  // Validate the consolidated content
  const validation = validateConsolidatedContent(readmeContent, consolidatedContent, dirInfo);
  if (!validation.isValid) {
    console.log(`  ❌ Consolidated content failed validation for ${dirInfo.relativePath}`);
    console.log(`  📋 Validation errors:`, validation.errors);
    return;
  }

  if (validation.warnings.length > 0) {
    console.log(`  ⚠️ Validation warnings for ${dirInfo.relativePath}:`, validation.warnings);
  }

  // Safely write the consolidated content
  try {
    safeWriteConsolidatedFile(readmePath, consolidatedContent, dirInfo, backupDir);
    console.log(`  ✅ Successfully consolidated ${otherContents.length} files into ${dirInfo.readmeFile}`);

    // Only remove individual files after successful validation and writing
    for (const fileInfo of otherContents) {
      if (fs.existsSync(fileInfo.path)) {
        console.log(`    🗑️ Removing consolidated file: ${fileInfo.filename}`);
        fs.unlinkSync(fileInfo.path);
      }
    }

    console.log(`  📊 Final metrics: ${validation.metrics.consolidatedLength} chars, ${validation.metrics.wordCount} words`);
  } catch (error) {
    console.log(`  ❌ Failed to write consolidated content: ${error.message}`);
    console.log(`  💾 Backup preserved at: ${backupDir}`);
  }
}

// Generate consolidated documentation using AI
async function generateConsolidatedDocumentation(directoryPath, readmeContent, otherContents) {
  console.log(`  🤖 Using AI to consolidate documentation for ${directoryPath}...`);

  try {
    // Prepare the prompt for AI consolidation
    const prompt = `You are a technical documentation expert. Please consolidate the following documentation files into a single, comprehensive README.md file.

DIRECTORY: ${directoryPath}

CURRENT README.md CONTENT:
${readmeContent}

ADDITIONAL .MD FILES TO CONSOLIDATE:
${otherContents.map(file => `
=== ${file.filename} ===
${file.content}
`).join('\n')}

CONSOLIDATION REQUIREMENTS:
1. Create a comprehensive, well-organized README.md that includes ALL information from the additional files
2. Organize content logically with clear sections and subsections
3. Remove any duplicate information
4. Maintain all technical details, commands, examples, and specifications
5. Use proper markdown formatting with headers, code blocks, tables, and lists
6. Include a table of contents for easy navigation
7. Preserve all important links, references, and technical specifications
8. Organize content into logical categories like: Overview, Features, Usage, Configuration, Troubleshooting, etc.
9. Make the documentation comprehensive enough that the individual files are no longer needed
10. Ensure the consolidated document is professional and easy to navigate

Please provide ONLY the consolidated README.md content, no explanations or comments.`;

    // Use the first available AI service
    let consolidatedContent = null;

    if (process.env.GEMINI_API_KEY) {
      consolidatedContent = await callGeminiAPI(prompt);
    } else if (process.env.CEREBRAS_API_KEY) {
      consolidatedContent = await callCerebrasAPI(prompt);
    } else {
      console.log('  ⚠️ No AI API keys available, performing basic consolidation');
      return performBasicConsolidation(readmeContent, otherContents);
    }

    return consolidatedContent;
  } catch (error) {
    console.log(`  ❌ AI consolidation failed: ${error.message}`);
    return performBasicConsolidation(readmeContent, otherContents);
  }
}

// Fallback basic consolidation without AI
function performBasicConsolidation(readmeContent, otherContents) {
  console.log('  📝 Performing basic consolidation without AI...');

  let consolidated = readmeContent;

  // Add a consolidation header if not present
  if (!consolidated.includes('# Consolidated Documentation')) {
    consolidated = `# Consolidated Documentation\n\n${consolidated}`;
  }

  // Add each file's content as a section
  for (const fileInfo of otherContents) {
    const sectionTitle = fileInfo.filename.replace('.md', '').replace(/[_-]/g, ' ').replace(/\b\w/g, l => l.toUpperCase());
    consolidated += `\n\n## ${sectionTitle}\n\n${fileInfo.content}`;
  }

  return consolidated;
}

// Enhanced README detection with comprehensive subdirectory scanning (excluding docs directory)
function findAllReadmeFiles() {
  console.log('🔍 Enhanced README detection with comprehensive subdirectory scanning (excluding docs directory)...');

  const readmeFiles = [];
  const projectRoot = rootDir;

  // Recursive function to scan directories
  function scanDirectory(dir, depth = 0) {
    if (depth > 4) return; // Limit recursion depth

    const relativePath = path.relative(projectRoot, dir);

    // Skip the docs directory entirely (except for whitepapers which are handled separately)
    if (relativePath === 'docs' || relativePath.startsWith('docs/')) {
      console.log(`Skipping docs directory: ${relativePath}`);
      return;
    }

    try {
      const items = fs.readdirSync(dir);

      for (const item of items) {
        const itemPath = path.join(dir, item);
        const stat = fs.lstatSync(itemPath);

        if (stat.isDirectory()) {
          // Skip excluded directories
          if (CONFIG.excludeSubDirs.includes(item) || item.startsWith('.')) {
            continue;
          }

          // Additional check to skip docs directory
          const subRelativePath = path.relative(projectRoot, itemPath);
          if (subRelativePath !== 'docs' && !subRelativePath.startsWith('docs/')) {
            // Recursively scan subdirectories
            scanDirectory(itemPath, depth + 1);
          }
        } else if (stat.isFile() && item.toLowerCase() === 'readme.md') {
          // Found a README.md file (not in docs directory)
          const fileRelativePath = path.relative(projectRoot, itemPath);
          readmeFiles.push({
            path: itemPath,
            relativePath: fileRelativePath,
            directory: path.dirname(fileRelativePath),
            lastModified: stat.mtime,
            size: stat.size
          });
        }
      }
    } catch (error) {
      // Skip directories we can't read
      console.log(`Skipping directory ${dir}: ${error.message}`);
    }
  }

  // Start scanning from project root
  scanDirectory(projectRoot);

  console.log(`📊 Found ${readmeFiles.length} README.md files across the project (excluding docs directory)`);

  // Log found files for debugging
  readmeFiles.forEach(file => {
    console.log(`  📄 ${file.relativePath} (${file.size} bytes, modified: ${file.lastModified.toISOString().split('T')[0]})`);
  });

  return readmeFiles;
}

// Main function
async function main() {
  console.log('🚀 Starting enhanced documentation generation with comprehensive consolidation...');

  // Load existing hashes
  let hashes = loadHashes();

  // Ensure output directory exists
  ensureDirectoryExists(CONFIG.outputDir);

  // STEP 1: Consolidate all .md files across all subproject directories
  console.log('\n📚 STEP 1: Comprehensive .md file consolidation across all directories');
  const consolidationResults = await consolidateAllProjectDocumentation();
  console.log(`✅ Consolidated documentation in ${consolidationResults.length} directories`);

  // STEP 2: Enhanced README detection after consolidation
  console.log('\n🔍 STEP 2: Enhanced README detection with updated file structure');
  const allReadmeFiles = findAllReadmeFiles();
  console.log(`📊 Detected ${allReadmeFiles.length} README.md files for processing`);

  // STEP 3: Check if whitepaper files have changed
  console.log('\n📄 STEP 3: Whitepaper change detection');
  const whitepaperFiles = getWhitepaperFiles();
  for (const filePath of whitepaperFiles) {
    if (hasFileChanged(filePath, hashes)) {
      updateFileHash(filePath, hashes);
      console.log(`Whitepaper file changed: ${path.basename(filePath)}`);
    }
  }

  // STEP 4: Process markdown files with AI organization
  console.log('\n🤖 STEP 4: AI-powered documentation organization');
  const docs = await processMarkdownFiles(hashes);

  // STEP 5: Create comprehensive Docsify structure with organized documentation
  console.log('\n🏗️ STEP 5: Creating comprehensive documentation hierarchy');
  await createEnhancedDocsifyStructure(docs, hashes, allReadmeFiles);

  // STEP 6: Process whitepapers separately
  console.log('\n📄 STEP 6: Whitepaper processing');
  processWhitepapers(hashes);

  // STEP 7: Save updated hashes
  saveHashes(hashes);

  // STEP 8: Final validation check for critical files
  console.log('\n🔍 STEP 8: Final validation of critical documentation files');
  await performFinalValidation(allReadmeFiles);

  console.log('\n🎉 Enhanced documentation generation complete!');
  console.log(`📚 Documentation: ${CONFIG.docsifyDir}`);
  console.log(`📄 Whitepapers: ${path.join(CONFIG.docsifyDir, 'whitepapers')}`);
  console.log(`🤖 Organized ${docs.length} documentation files using AI`);
  console.log(`📁 Consolidated documentation in ${consolidationResults.length} directories`);
  console.log(`🔍 Processed ${allReadmeFiles.length} README.md files`);
  console.log(`💾 Backups stored in: ${CONFIG.backupDir}`);
  console.log('📊 Generated comprehensive documentation hierarchy for public navigation');
}

// Perform final validation of critical documentation files
async function performFinalValidation(allReadmeFiles) {
  console.log('🔍 Performing final validation of critical documentation files...');

  const criticalFiles = [
    'KNIRVTESTNET/README.md',
    'KNIRVWALLET/README.md',
    'scripts/README.md',
    'README.md'
  ];

  let validationIssues = 0;

  for (const criticalFile of criticalFiles) {
    const filePath = path.join(rootDir, criticalFile);

    if (!fs.existsSync(filePath)) {
      console.log(`  ❌ Critical file missing: ${criticalFile}`);
      validationIssues++;
      continue;
    }

    const content = fs.readFileSync(filePath, 'utf8');
    const fileSize = content.length;
    const wordCount = content.split(/\s+/).length;

    // Check for truncation indicators
    const isTruncated = content.endsWith('*KNIR') ||
                       content.endsWith('- Ensure') ||
                       (content.match(/[^.!?*]\s*$/) && !content.endsWith('*'));

    if (isTruncated) {
      console.log(`  ❌ File appears truncated: ${criticalFile}`);
      validationIssues++;
    } else if (fileSize < 1000) {
      console.log(`  ⚠️ File seems too short: ${criticalFile} (${fileSize} chars)`);
      validationIssues++;
    } else {
      console.log(`  ✅ ${criticalFile} validated (${fileSize} chars, ${wordCount} words)`);
    }
  }

  if (validationIssues > 0) {
    console.log(`\n⚠️ Found ${validationIssues} validation issues with critical files`);
    console.log(`💾 Check backup directory for recovery: ${CONFIG.backupDir}`);
  } else {
    console.log('\n✅ All critical files passed validation');
  }
}

// Enhanced Docsify structure creation with comprehensive organization
async function createEnhancedDocsifyStructure(docs, hashes, allReadmeFiles) {
  console.log('🏗️ Creating enhanced Docsify structure with comprehensive organization...');

  // Create the standard Docsify structure
  createDocsifyStructure(docs, hashes);

  // Generate comprehensive documentation categories using AI
  await generateComprehensiveDocumentationHierarchy(allReadmeFiles, hashes);

  console.log('✅ Enhanced Docsify structure created successfully');
}

// Generate comprehensive documentation hierarchy using AI inference
async function generateComprehensiveDocumentationHierarchy(allReadmeFiles, hashes) {
  console.log('🤖 Generating comprehensive documentation hierarchy using AI inference...');

  try {
    // Prepare content for AI analysis (excluding docs directory files)
    const documentationContent = allReadmeFiles
      .filter(file => !file.relativePath.startsWith('docs/') && file.relativePath !== 'docs')
      .map(file => ({
        path: file.relativePath,
        directory: file.directory,
        content: fs.readFileSync(file.path, 'utf8').substring(0, 2000), // First 2000 chars for analysis
        size: file.size,
        lastModified: file.lastModified
      }));

    // Create AI prompt for comprehensive organization
    const organizationPrompt = `You are a technical documentation architect. Analyze the following README.md files from a complex software project and organize them into a comprehensive, navigable documentation hierarchy.

PROJECT: KNIRV Network (Decentralized Trusted Execution Network)

README FILES TO ORGANIZE:
${documentationContent.map(doc => `
=== ${doc.path} ===
Directory: ${doc.directory}
Size: ${doc.size} bytes
Content Preview:
${doc.content}
`).join('\n')}

ORGANIZATION REQUIREMENTS:
1. Create a comprehensive documentation hierarchy with these main categories:
   - Deployment Guides (step-by-step deployment instructions)
   - Troubleshooting Guides (common issues and solutions)
   - Architecture Documents (system design and component relationships)
   - Current Status Reports (real-time system status and metrics)
   - API Documentation (comprehensive API reference)
   - User Guides (end-user documentation and tutorials)
   - Developer Guides (development setup and contribution guides)

2. For each category, organize the README files into logical subcategories
3. Create a navigation structure suitable for public documentation
4. Generate appropriate titles and descriptions for each section
5. Identify which files belong in which categories based on their content
6. Create cross-references between related documentation
7. Generate a master table of contents

Please provide a JSON structure with the following format:
{
  "categories": {
    "deployment": {
      "title": "Deployment Guides",
      "description": "...",
      "subcategories": {
        "infrastructure": {
          "title": "Infrastructure Deployment",
          "files": ["path/to/readme.md", ...],
          "description": "..."
        }
      }
    }
  },
  "navigation": [...],
  "crossReferences": {...}
}`;

    let organizationResult = null;

    // Try AI organization
    if (process.env.GEMINI_API_KEY) {
      organizationResult = await callGeminiAPI(organizationPrompt);
    } else if (process.env.CEREBRAS_API_KEY) {
      organizationResult = await callCerebrasAPI(organizationPrompt);
    }

    if (organizationResult) {
      // Parse and apply the AI-generated organization
      await applyDocumentationOrganization(organizationResult, allReadmeFiles);
    } else {
      // Fallback to basic organization
      await createBasicDocumentationHierarchy(allReadmeFiles);
    }

  } catch (error) {
    console.log(`❌ AI organization failed: ${error.message}`);
    await createBasicDocumentationHierarchy(allReadmeFiles);
  }
}

// Apply AI-generated documentation organization
async function applyDocumentationOrganization(organizationJson, allReadmeFiles) {
  console.log('📊 Applying AI-generated documentation organization...');

  try {
    const organization = JSON.parse(organizationJson);

    // Create organized directory structure in docsify
    const docsifyDir = CONFIG.docsifyDir;

    for (const [categoryKey, categoryInfo] of Object.entries(organization.categories || {})) {
      const categoryDir = path.join(docsifyDir, categoryKey);
      ensureDirectoryExists(categoryDir);

      // Create category index file
      const categoryIndexContent = `# ${categoryInfo.title}\n\n${categoryInfo.description}\n\n`;
      fs.writeFileSync(path.join(categoryDir, 'README.md'), categoryIndexContent);

      // Process subcategories
      for (const [subKey, subInfo] of Object.entries(categoryInfo.subcategories || {})) {
        const subDir = path.join(categoryDir, subKey);
        ensureDirectoryExists(subDir);

        // Copy relevant files to subcategory
        for (const filePath of subInfo.files || []) {
          const sourceFile = allReadmeFiles.find(f => f.relativePath === filePath);
          if (sourceFile) {
            const targetPath = path.join(subDir, path.basename(sourceFile.path));
            fs.copyFileSync(sourceFile.path, targetPath);
          }
        }
      }
    }

    console.log('✅ AI-generated documentation organization applied successfully');
  } catch (error) {
    console.log(`❌ Failed to apply AI organization: ${error.message}`);
    await createBasicDocumentationHierarchy(allReadmeFiles);
  }
}

// Fallback basic documentation hierarchy
async function createBasicDocumentationHierarchy(allReadmeFiles) {
  console.log('📝 Creating basic documentation hierarchy...');

  const docsifyDir = CONFIG.docsifyDir;
  const categories = {
    'deployment': 'Deployment Guides',
    'troubleshooting': 'Troubleshooting',
    'architecture': 'Architecture',
    'api': 'API Documentation',
    'guides': 'User Guides'
  };

  for (const [key, title] of Object.entries(categories)) {
    const categoryDir = path.join(docsifyDir, key);
    ensureDirectoryExists(categoryDir);

    const indexContent = `# ${title}\n\nThis section contains ${title.toLowerCase()} for the KNIRV Network.\n\n`;
    fs.writeFileSync(path.join(categoryDir, 'README.md'), indexContent);
  }

  console.log('✅ Basic documentation hierarchy created');
}

// Run the main function
main();