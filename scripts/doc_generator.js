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
const dotenvPath = path.join(__dirname, '..', 'KNIRVGATEWAY', 'documentation', 'node_modules', 'dotenv');
const dotenv = require(dotenvPath);
dotenv.config({ path: path.join(__dirname, '..', 'KNIRVGATEWAY', 'documentation', '.env') });

// Configuration
const rootDir = path.dirname(__dirname); // Go up one level from scripts directory
const CONFIG = {
  sourceDir: path.join(rootDir, 'docs'),
  outputDir: path.join(rootDir, 'KNIRVGATEWAY', 'documentation'),
  docsifyDir: path.join(rootDir, 'KNIRVGATEWAY', 'documentation', 'docsify'),
  hashFile: path.join(rootDir, 'KNIRVGATEWAY', 'documentation', '.doc_hashes.json'),
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
    'knirvnexus': 'KNIRVNEXUS Documentation',
    'knirvroot': 'KNIRVROOT Documentation',
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
  // Subproduct directories to scan for additional documentation
  subproductDirs: [
    'KNIRVCHAIN',
    'KNIRVGRAPH',
    'KNIRVNEXUS',
    'KNIRVROOT',
    'KNIRVROUTER',
    'KNIRVSDK',
    'KNIRVCORTEX',
    'KNIRVWALLET',
    'KNIRVSHELL',
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
<a href="#/legal/CODE_OF_CONDUCT.md" class="footer-link">Contributor Covenant Code of Conduct</a> | <a href="#/legal/PRIVACY_POLICY.md" class="footer-link">PRIVACY_POLICY.md</a> | <a href="#/legal/TERMS_AND_CONDITIONS.md" class="footer-link">TERMS AND CONDITIONS</a>

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
    'KNIRVNEXUS': 'knirvnexus',
    'KNIRVROOT': 'knirvroot',
    'KNIRVROUTER': 'knirvrouter',
    'KNIRVSDK': 'knirvsdk',
    'KNIRVCORTEX': 'knirvshell',
    'KNIRVWALLET': 'knirvwallet',
    'KNIRVSHELL': 'knirvshell',
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

1. CATEGORY: Choose the most appropriate category from: guides, deployment, development, api, security, architecture, contribute, legal, knirvchain, knirvgraph, knirvnexus, knirvroot, knirvrouter, knirvsdk, knirvshell, knirvwallet
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

// Main function
async function main() {
  console.log('Starting documentation generation...');

  // Load existing hashes
  let hashes = loadHashes();

  // Ensure output directory exists
  ensureDirectoryExists(CONFIG.outputDir);

  // Check if whitepaper files have changed
  const whitepaperFiles = getWhitepaperFiles();
  for (const filePath of whitepaperFiles) {
    if (hasFileChanged(filePath, hashes)) {
      updateFileHash(filePath, hashes);
      console.log(`Whitepaper file changed: ${path.basename(filePath)}`);
    }
  }

  // Process markdown files with AI organization
  const docs = await processMarkdownFiles(hashes);

  // Create Docsify structure with organized documentation
  createDocsifyStructure(docs, hashes);

  // Process whitepapers separately
  processWhitepapers(hashes);

  // Save updated hashes
  saveHashes(hashes);

  console.log('Documentation generation complete!');
  console.log(`Documentation: ${CONFIG.docsifyDir}`);
  console.log(`Whitepapers: ${path.join(CONFIG.docsifyDir, 'whitepapers')}`);
  console.log(`Organized ${docs.length} documentation files using AI`);
}

// Run the main function
main();