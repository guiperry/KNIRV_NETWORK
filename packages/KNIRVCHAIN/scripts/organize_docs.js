/**
 * Documentation Organizer for KNIRVCHAIN
 * 
 * This script processes the existing documentation in the docs/ folder
 * and transforms it into an organized, cohesive documentation system
 * in a new documentation/ folder, all written in markdown.
 * 
 * The script is designed to be idempotent - running it multiple times
 * will produce the same result as running it once.
 */

const fs = require('fs');
const path = require('path');
const util = require('util');

// Convert fs functions to promise-based
const readdir = util.promisify(fs.readdir);
const readFile = util.promisify(fs.readFile);
const writeFile = util.promisify(fs.writeFile);
const mkdir = util.promisify(fs.mkdir);
const stat = util.promisify(fs.stat);

// Configuration
const SOURCE_DIR = path.join(__dirname, '..', 'docs');
const TARGET_DIR = path.join(__dirname, '..', 'documentation', 'docsify');
const README_PATH = path.join(__dirname, '..', 'README.md');

// Documentation structure
const STRUCTURE = {
  'index.md': { title: 'KNIRVCHAIN Documentation', content: '' },
  'getting-started': {
    'index.md': { title: 'Getting Started', content: '' },
    'installation.md': { title: 'Installation Guide', content: '' },
    'configuration.md': { title: 'Configuration', content: '' },
    'quick-start.md': { title: 'Quick Start Guide', content: '' }
  },
  'core-concepts': {
    'index.md': { title: 'Core Concepts', content: '' },
    'blockchain.md': { title: 'Blockchain Architecture', content: '' },
    'mcp.md': { title: 'Model Context Protocol (MCP)', content: '' },
    'capabilities.md': { title: 'Capabilities System', content: '' },
    'uri-scheme.md': { title: 'URI Scheme', content: '' }
  },
  'protocols': {
    'index.md': { title: 'Protocols', content: '' }
    // Will be populated from docs/protocols
  },
  'api-reference': {
    'index.md': { title: 'API Reference', content: '' },
    'blockchain-api.md': { title: 'Blockchain API', content: '' },
    'wallet-api.md': { title: 'Wallet API', content: '' },
    'mcp-api.md': { title: 'MCP API', content: '' }
  },
  'components': {
    'index.md': { title: 'Components', content: '' },
    'agent-tunnel-registry.md': { title: 'Agent Tunnel Registry', content: '' },
    'operator-registry.md': { title: 'Agent Bootnode Registry', content: '' },
    'agent-payment-gateway.md': { title: 'Agent Payment Gateway', content: '' },
    'agent-developer-portal.md': { title: 'Agent Developer Portal', content: '' },
    'altgui.md': { title: 'Alternative GUI', content: '' }
  },
  'guides': {
    'index.md': { title: 'Guides', content: '' },
    'running-a-node.md': { title: 'Running a Node', content: '' },
    'developing-plugins.md': { title: 'Developing Plugins', content: '' },
    'creating-capabilities.md': { title: 'Creating Capabilities', content: '' }
  },
  'troubleshooting': {
    'index.md': { title: 'Troubleshooting', content: '' }
    // Will be populated from docs/Troubleshooting
  },
  'sdk': {
    'index.md': { title: 'SDK Documentation', content: '' }
    // Will be populated from docs/SDK
  }
};

// Category mapping for existing documentation
const CATEGORY_MAPPING = {
  'protocols': {
    folder: 'protocols',
    pattern: /.*\.md$/
  },
  'troubleshooting': {
    folder: 'Troubleshooting',
    pattern: /.*\.md$/
  },
  'sdk': {
    folder: 'SDK',
    pattern: /.*\.md$/
  },
  'core-concepts': {
    folder: '',
    pattern: /(Agent_Focus|agent_inferencer).*\.md$/
  },
  'guides': {
    folder: 'completedImplementations',
    pattern: /.*\.md$/
  }
};

// Content mapping for specific files
const CONTENT_MAPPING = {
  'core-concepts/blockchain.md': ['protocols/Blockchain_README.md'],
  'core-concepts/mcp.md': ['protocols/context_record_scenarios.md'],
  'core-concepts/capabilities.md': ['protocols/Capabilities_Protocol.md'],
  'core-concepts/uri-scheme.md': ['protocols/URI_Generation_Protocol.md'],
  'api-reference/blockchain-api.md': ['protocols/Blockchain_README.md'],
  'api-reference/wallet-api.md': ['protocols/Blockchain_README.md'],
  'api-reference/mcp-api.md': ['protocols/Blockchain_README.md'],
  'getting-started/installation.md': ['protocols/Blockchain_README.md'],
  'getting-started/configuration.md': ['protocols/Blockchain_README.md'],
  'getting-started/quick-start.md': ['protocols/Blockchain_README.md'],
  'components/agent-tunnel-registry.md': ['protocols/tunnel_relay_implementation_summary.md'],
  'components/operator-registry.md': ['protocols/Bootnode_Registry_Protocol.md'],
  'guides/running-a-node.md': ['protocols/Blockchain_README.md'],
  'guides/developing-plugins.md': ['protocols/Plugin_Updater_Protocol.md']
};

// Keywords to identify content for specific sections
const KEYWORD_MAPPING = {
  'core-concepts/blockchain.md': ['blockchain', 'consensus', 'p2p', 'libp2p'],
  'core-concepts/mcp.md': ['mcp', 'model context protocol', 'contextrecord'],
  'core-concepts/capabilities.md': ['capabilities', 'plugins', 'tools', 'prompts'],
  'core-concepts/uri-scheme.md': ['uri', 'knirv://', 'scheme'],
  'api-reference/blockchain-api.md': ['api', 'endpoints', 'http', 'get', 'post'],
  'api-reference/wallet-api.md': ['wallet', 'transaction', 'signing'],
  'api-reference/mcp-api.md': ['mcp', 'api', '/mcp/'],
  'getting-started/installation.md': ['prerequisites', 'building', 'install'],
  'getting-started/configuration.md': ['config', 'configuration', 'settings'],
  'getting-started/quick-start.md': ['running', 'getting started', 'quick'],
  'guides/running-a-node.md': ['running', 'node', 'startup'],
  'guides/developing-plugins.md': ['plugin', 'develop', 'create']
};

/**
 * Creates the directory structure for the new documentation
 */
async function createDirectoryStructure() {
  console.log('Creating directory structure...');
  
  // Create the main documentation directory
  if (!fs.existsSync(TARGET_DIR)) {
    await mkdir(TARGET_DIR);
  }
  
  // Create subdirectories and empty index files
  for (const [dir, content] of Object.entries(STRUCTURE)) {
    if (typeof content === 'object' && !Array.isArray(content)) {
      // This is a directory
      if (dir !== 'index.md') {
        const dirPath = path.join(TARGET_DIR, dir);
        if (!fs.existsSync(dirPath)) {
          await mkdir(dirPath);
        }
        
        // Create files in this directory
        for (const [file, fileContent] of Object.entries(content)) {
          const filePath = path.join(dirPath, file);
          if (!fs.existsSync(filePath)) {
            const initialContent = `# ${fileContent.title}\n\n${fileContent.content}`;
            await writeFile(filePath, initialContent);
          }
        }
      } else {
        // This is the root index.md
        const filePath = path.join(TARGET_DIR, dir);
        if (!fs.existsSync(filePath)) {
          const initialContent = `# ${content.title}\n\n${content.content}`;
          await writeFile(filePath, initialContent);
        }
      }
    }
  }
  
  console.log('Directory structure created successfully.');
}

/**
 * Extracts content from README.md for the main index page
 */
async function processReadme() {
  console.log('Processing README.md...');
  
  try {
    const content = await readFile(README_PATH, 'utf8');
    
    // Extract relevant sections from README
    const lines = content.split('\n');
    let extractedContent = '';
    let inRelevantSection = false;
    
    for (const line of lines) {
      // Skip the title as we'll add our own
      if (line.startsWith('# KNIRVCHAIN')) {
        continue;
      }
      
      // Include content after the title
      if (line.startsWith('KNIRVCHAIN is')) {
        inRelevantSection = true;
      }
      
      if (inRelevantSection) {
        extractedContent += line + '\n';
      }
      
      // Stop at the API Endpoints section or other technical details
      if (line.startsWith('## API Endpoints') || line.startsWith('## Data Storage')) {
        break;
      }
    }
    
    // Update the main index.md
    const indexPath = path.join(TARGET_DIR, 'index.md');
    const indexContent = `# KNIRVCHAIN Documentation\n\n${extractedContent}\n\n## Documentation Sections\n\n`;
    
    // Add links to main sections
    let sectionLinks = '';
    for (const [dir, content] of Object.entries(STRUCTURE)) {
      if (dir !== 'index.md' && typeof content === 'object') {
        const title = content['index.md'].title;
        sectionLinks += `- [${title}](./${dir}/)\n`;
      }
    }
    
    await writeFile(indexPath, indexContent + sectionLinks);
    console.log('README processed and index.md updated.');
    
  } catch (error) {
    console.error('Error processing README:', error);
  }
}

/**
 * Processes protocol documentation files
 */
async function processProtocols() {
  console.log('Processing protocol documentation...');
  
  try {
    const protocolsDir = path.join(SOURCE_DIR, 'protocols');
    const files = await readdir(protocolsDir);
    
    // Create entries in the protocols section
    for (const file of files) {
      if (file.endsWith('.md')) {
        const sourcePath = path.join(protocolsDir, file);
        const targetPath = path.join(TARGET_DIR, 'protocols', file);
        
        // Read the protocol file
        const content = await readFile(sourcePath, 'utf8');
        
        // Extract title from the first line
        const lines = content.split('\n');
        let title = file.replace('.md', '').replace(/_/g, ' ');
        
        if (lines[0].startsWith('# ')) {
          title = lines[0].substring(2);
        }
        
        // Add to the structure
        STRUCTURE.protocols[file] = { title, content: '' };
        
        // Copy the file
        await writeFile(targetPath, content);
        
        // Update the protocols index
        const indexPath = path.join(TARGET_DIR, 'protocols', 'index.md');
        let indexContent = await readFile(indexPath, 'utf8');
        
        if (!indexContent.includes(`- [${title}](./${file})`)) {
          indexContent += `- [${title}](./${file})\n`;
          await writeFile(indexPath, indexContent);
        }
      }
    }
    
    console.log('Protocol documentation processed.');
    
  } catch (error) {
    console.error('Error processing protocols:', error);
  }
}

/**
 * Processes documentation based on category mappings
 */
async function processCategoryMappings() {
  console.log('Processing category mappings...');
  
  for (const [category, mapping] of Object.entries(CATEGORY_MAPPING)) {
    try {
      const sourceDir = path.join(SOURCE_DIR, mapping.folder);
      
      // Skip if the source directory doesn't exist
      if (!fs.existsSync(sourceDir)) {
        console.log(`Source directory ${sourceDir} does not exist. Skipping.`);
        continue;
      }
      
      const files = await readdir(sourceDir);
      
      for (const file of files) {
        if (mapping.pattern.test(file)) {
          const sourcePath = path.join(sourceDir, file);
          
          // Check if it's a directory
          const stats = await stat(sourcePath);
          if (stats.isDirectory()) {
            continue;
          }
          
          // Read the file
          const content = await readFile(sourcePath, 'utf8');
          
          // Extract title from the first line
          const lines = content.split('\n');
          let title = file.replace('.md', '').replace(/_/g, ' ');
          
          if (lines[0].startsWith('# ')) {
            title = lines[0].substring(2);
          }
          
          // Create target file name
          const targetFile = file.toLowerCase().replace(/[^a-z0-9]+/g, '-');
          const targetPath = path.join(TARGET_DIR, category, targetFile);
          
          // Add to the structure if not already there
          if (!STRUCTURE[category][targetFile]) {
            STRUCTURE[category][targetFile] = { title, content: '' };
          }
          
          // Copy the file
          await writeFile(targetPath, content);
          
          // Update the category index
          const indexPath = path.join(TARGET_DIR, category, 'index.md');
          let indexContent = await readFile(indexPath, 'utf8');
          
          if (!indexContent.includes(`- [${title}](./${targetFile})`)) {
            indexContent += `- [${title}](./${targetFile})\n`;
            await writeFile(indexPath, indexContent);
          }
        }
      }
      
    } catch (error) {
      console.error(`Error processing category ${category}:`, error);
    }
  }
  
  console.log('Category mappings processed.');
}

/**
 * Processes specific content mappings
 */
async function processContentMappings() {
  console.log('Processing content mappings...');
  
  for (const [targetFile, sourcePaths] of Object.entries(CONTENT_MAPPING)) {
    try {
      const targetPath = path.join(TARGET_DIR, targetFile);
      
      // Read the target file to get its current content
      let targetContent = await readFile(targetPath, 'utf8');
      
      // Process each source file
      for (const sourcePath of sourcePaths) {
        const fullSourcePath = path.join(SOURCE_DIR, sourcePath);
        
        // Skip if the source file doesn't exist
        if (!fs.existsSync(fullSourcePath)) {
          console.log(`Source file ${fullSourcePath} does not exist. Skipping.`);
          continue;
        }
        
        // Read the source file
        const sourceContent = await readFile(fullSourcePath, 'utf8');
        
        // Extract relevant content based on keywords
        const keywords = KEYWORD_MAPPING[targetFile] || [];
        const sections = extractRelevantSections(sourceContent, keywords, targetFile);
        
        // Append the extracted content to the target file
        if (sections && !targetContent.includes(sections)) {
          targetContent += '\n\n' + sections;
        }
      }
      
      // Write the updated content back to the target file
      await writeFile(targetPath, targetContent);
      
    } catch (error) {
      console.error(`Error processing content mapping for ${targetFile}:`, error);
    }
  }
  
  console.log('Content mappings processed.');
}

/**
 * Extracts relevant sections from content based on keywords
 */
function extractRelevantSections(content, keywords, targetFile) {
  // Split content into sections by headings
  const sections = content.split(/^##\s+/m);
  
  let relevantContent = '';
  let foundRelevant = false;
  
  // Check each section for keywords
  for (let i = 0; i < sections.length; i++) {
    const section = sections[i];
    
    // Skip the first section if it contains the title
    if (i === 0 && section.startsWith('# ')) {
      continue;
    }
    
    // Check if this section contains any of the keywords
    const containsKeyword = keywords.some(keyword => 
      section.toLowerCase().includes(keyword.toLowerCase())
    );
    
    // Special handling for specific target files
    if (targetFile === 'api-reference/blockchain-api.md' && section.includes('API Endpoints')) {
      relevantContent += '## API Endpoints\n\n' + section;
      foundRelevant = true;
    } else if (targetFile === 'getting-started/installation.md' && 
              (section.includes('Prerequisites') || section.includes('Building'))) {
      relevantContent += '## ' + section;
      foundRelevant = true;
    } else if (targetFile === 'getting-started/configuration.md' && section.includes('Configuration')) {
      relevantContent += '## ' + section;
      foundRelevant = true;
    } else if (targetFile === 'guides/running-a-node.md' && section.includes('Running')) {
      relevantContent += '## ' + section;
      foundRelevant = true;
    } else if (containsKeyword) {
      relevantContent += '## ' + section;
      foundRelevant = true;
    }
  }
  
  return foundRelevant ? relevantContent : null;
}

/**
 * Updates section index files with links to their content
 */
async function updateSectionIndexes() {
  console.log('Updating section index files...');
  
  for (const [dir, content] of Object.entries(STRUCTURE)) {
    if (dir !== 'index.md' && typeof content === 'object') {
      const indexPath = path.join(TARGET_DIR, dir, 'index.md');
      let indexContent = `# ${content['index.md'].title}\n\n`;
      
      // Add description based on the section
      switch (dir) {
        case 'getting-started':
          indexContent += 'This section helps you get up and running with KNIRVCHAIN quickly.\n\n';
          break;
        case 'core-concepts':
          indexContent += 'Learn about the fundamental concepts that power KNIRVCHAIN.\n\n';
          break;
        case 'protocols':
          indexContent += 'Detailed documentation of the protocols used in KNIRVCHAIN.\n\n';
          break;
        case 'api-reference':
          indexContent += 'Complete reference for all KNIRVCHAIN APIs.\n\n';
          break;
        case 'components':
          indexContent += 'Information about the various components that make up the KNIRVCHAIN ecosystem.\n\n';
          break;
        case 'guides':
          indexContent += 'Step-by-step guides for common KNIRVCHAIN tasks.\n\n';
          break;
        case 'troubleshooting':
          indexContent += 'Solutions to common problems and troubleshooting tips.\n\n';
          break;
        case 'sdk':
          indexContent += 'Documentation for the KNIRVCHAIN SDK.\n\n';
          break;
      }
      
      // Add links to all files in this section
      indexContent += '## Contents\n\n';
      
      for (const [file, fileContent] of Object.entries(content)) {
        if (file !== 'index.md') {
          indexContent += `- [${fileContent.title}](./${file})\n`;
        }
      }
      
      // Write the updated index
      await writeFile(indexPath, indexContent);
    }
  }
  
  console.log('Section index files updated.');
}

/**
 * Creates a navigation sidebar file
 */
async function createSidebar() {
  console.log('Creating sidebar navigation...');
  
  const sidebarPath = path.join(TARGET_DIR, '_sidebar.md');
  let sidebarContent = '# KNIRVCHAIN Docs\n\n';
  
  // Add link to home
  sidebarContent += '- [Home](./)\n';
  
  // Add links to main sections
  for (const [dir, content] of Object.entries(STRUCTURE)) {
    if (dir !== 'index.md' && typeof content === 'object') {
      sidebarContent += `- [${content['index.md'].title}](./${dir}/)\n`;
      
      // Add subsections
      for (const [file, fileContent] of Object.entries(content)) {
        if (file !== 'index.md') {
          sidebarContent += `  - [${fileContent.title}](./${dir}/${file})\n`;
        }
      }
    }
  }
  
  // Write the sidebar file
  await writeFile(sidebarPath, sidebarContent);
  
  console.log('Sidebar navigation created.');
}

/**
 * Creates the index.html file for Docsify with search enabled
 */
async function createDocsifyIndex() {
  console.log('Creating Docsify index.html with search...');

  const indexHtmlContent = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <title>KNIRVCHAIN Documentation</title>
  <meta http-equiv="X-UA-Compatible" content="IE=edge,chrome=1" />
  <meta name="description" content="KNIRVCHAIN Documentation">
  <meta name="viewport" content="width=device-width, initial-scale=1.0, minimum-scale=1.0">
  <link rel="stylesheet" href="//cdn.jsdelivr.net/npm/docsify@4/lib/themes/vue.css">
</head>
<body>
  <div id="app"></div>
  <script>
    window.$docsify = {
      name: 'KNIRVCHAIN',
      repo: 'https://github.com/gperry/KNIRVCHAIN',
      loadSidebar: true,
      subMaxLevel: 3,
      search: 'auto', // Enables the search plugin
    }
  </script>
  <!-- Docsify Core -->
  <script src="//cdn.jsdelivr.net/npm/docsify@4"></script>
  <!-- Docsify Search Plugin -->
  <script src="//cdn.jsdelivr.net/npm/docsify/lib/plugins/search.min.js"></script>
</body>
</html>`;

  const indexPath = path.join(TARGET_DIR, 'index.html');
  await writeFile(indexPath, indexHtmlContent);

  console.log('Docsify index.html created successfully.');
}

/**
 * Cleans the target directory to ensure idempotency
 */
async function cleanTargetDirectory() {
  console.log('Cleaning target directory to ensure idempotency...');
  
  // Don't delete the entire directory as it might contain custom files
  // Instead, only delete files that we know we'll regenerate
  
  // Process each section in our structure
  for (const [dir, content] of Object.entries(STRUCTURE)) {
    if (dir !== 'index.md' && typeof content === 'object') {
      const dirPath = path.join(TARGET_DIR, dir);
      
      // Skip if directory doesn't exist
      if (!fs.existsSync(dirPath)) {
        continue;
      }
      
      // Clean index file
      const indexPath = path.join(dirPath, 'index.md');
      if (fs.existsSync(indexPath)) {
        await writeFile(indexPath, `# ${content['index.md'].title}\n\n${content['index.md'].content}`);
      }
      
      // Clean other files defined in our structure
      for (const [file, fileContent] of Object.entries(content)) {
        if (file !== 'index.md') {
          const filePath = path.join(dirPath, file);
          if (fs.existsSync(filePath)) {
            await writeFile(filePath, `# ${fileContent.title}\n\n${fileContent.content}`);
          }
        }
      }
    }
  }
  
  // Clean main index and sidebar
  const indexPath = path.join(TARGET_DIR, 'index.md');
  if (fs.existsSync(indexPath)) {
    await writeFile(indexPath, `# ${STRUCTURE['index.md'].title}\n\n${STRUCTURE['index.md'].content}`);
  }
  
  const sidebarPath = path.join(TARGET_DIR, '_sidebar.md');
  if (fs.existsSync(sidebarPath)) {
    await writeFile(sidebarPath, '# KNIRVCHAIN Docs\n\n');
  }
  
  console.log('Target directory cleaned.');
}

/**
 * Main function to run the documentation organization process
 */
async function main() {
  try {
    console.log('Starting documentation organization process...');
    
    // Create the directory structure
    await createDirectoryStructure();
    
    // Clean target directory to ensure idempotency
    await cleanTargetDirectory();
    
    // Process the README for the main index
    await processReadme();
    
    // Process protocol documentation
    await processProtocols();
    
    // Process category mappings
    await processCategoryMappings();
    
    // Process content mappings
    await processContentMappings();
    
    // Update section index files
    await updateSectionIndexes();
    
    // Create sidebar navigation
    await createSidebar();
    
    // Create Docsify index.html for search functionality
    await createDocsifyIndex();
    
    console.log('Documentation organization complete! The new documentation is available in the "documentation" directory.');
    
  } catch (error) {
    console.error('Error organizing documentation:', error);
  }
}

// Run the main function
main();