#!/usr/bin/env node

/**
 * Simple Documentation Generator for KNIRV Network
 * 
 * This script generates documentation structure from README files across all subprojects
 * without the complex consolidation process.
 */

const fs = require('fs');
const path = require('path');

// Configuration
const rootDir = path.dirname(__dirname);
const CONFIG = {
  outputDir: path.join(rootDir, 'KNIRVGATEWAY', 'documentation'),
  docsifyDir: path.join(rootDir, 'KNIRVGATEWAY', 'documentation', 'docsify'),
  projectName: 'KNIRV Network',
  subproductDirs: [
    'KNIRVANA',
    'KNIRVCHAIN',
    'KNIRVCONTROLLER',
    'KNIRVCORTEX',
    'KNIRVGATEWAY',
    'KNIRVGRAPH',
    'KNIRVNEXUS',
    'KNIRVORACLE',
    'KNIRVROUTER',
    'KNIRVSDK',
    'KNIRVTESTNET',
    'KNIRVWALLET'
  ]
};

// Ensure directory exists
function ensureDirectoryExists(dir) {
  if (!fs.existsSync(dir)) {
    fs.mkdirSync(dir, { recursive: true });
  }
}

// Find all README files
function findAllReadmeFiles() {
  console.log('🔍 Finding all README files...');
  const readmeFiles = [];
  
  // Add root README
  const rootReadme = path.join(rootDir, 'README.md');
  if (fs.existsSync(rootReadme)) {
    readmeFiles.push({
      path: rootReadme,
      relativePath: 'README.md',
      directory: rootDir,
      size: fs.statSync(rootReadme).size
    });
  }
  
  // Add subproject READMEs
  CONFIG.subproductDirs.forEach(subproductDir => {
    const subproductPath = path.join(rootDir, subproductDir);
    const readmePath = path.join(subproductPath, 'README.md');
    
    if (fs.existsSync(readmePath)) {
      readmeFiles.push({
        path: readmePath,
        relativePath: path.join(subproductDir, 'README.md'),
        directory: subproductPath,
        size: fs.statSync(readmePath).size
      });
    }
  });
  
  console.log(`📊 Found ${readmeFiles.length} README files`);
  return readmeFiles;
}

// Generate content for categories (disabled - categories removed per user request)
function generateCategoryContent(allReadmeFiles) {
  console.log('📝 Skipping category content generation (categories disabled)...');

  const categories = {
    // Categories removed per user request
    // 'deployment': 'Deployment Guides',
    // 'troubleshooting': 'Troubleshooting',
    // 'architecture': 'Architecture',
    // 'api': 'API Documentation',
    // 'guides': 'User Guides'
  };

  // Categorize README files based on content and path
  const categorizedContent = {
    'deployment': [],
    'troubleshooting': [],
    'architecture': [],
    'api': [],
    'guides': []
  };

  // Analyze each README file and categorize it
  allReadmeFiles.forEach(file => {
    if (file.relativePath.startsWith('docs/')) return; // Skip docs directory
    
    try {
      const content = fs.readFileSync(file.path, 'utf8');
      const lowerContent = content.toLowerCase();
      const fileName = path.basename(path.dirname(file.relativePath)) || 'KNIRV_NETWORK';
      
      // Categorize based on content keywords and file paths
      if (lowerContent.includes('deploy') || lowerContent.includes('installation') || 
          lowerContent.includes('setup') || lowerContent.includes('getting started') ||
          file.relativePath.includes('deployment') || fileName === 'KNIRVTESTNET') {
        categorizedContent.deployment.push({file, content, title: fileName});
      }
      
      if (lowerContent.includes('troubleshoot') || lowerContent.includes('error') || 
          lowerContent.includes('debug') || lowerContent.includes('issue') ||
          lowerContent.includes('problem') || lowerContent.includes('fix')) {
        categorizedContent.troubleshooting.push({file, content, title: fileName});
      }
      
      if (lowerContent.includes('architecture') || lowerContent.includes('design') || 
          lowerContent.includes('component') || lowerContent.includes('system') ||
          lowerContent.includes('overview') || fileName.includes('CORTEX') || fileName.includes('NEXUS')) {
        categorizedContent.architecture.push({file, content, title: fileName});
      }
      
      if (lowerContent.includes('api') || lowerContent.includes('endpoint') || 
          lowerContent.includes('rest') || lowerContent.includes('graphql') ||
          lowerContent.includes('interface') || fileName.includes('SDK') || fileName.includes('GATEWAY')) {
        categorizedContent.api.push({file, content, title: fileName});
      }
      
      if (lowerContent.includes('user') || lowerContent.includes('guide') || 
          lowerContent.includes('tutorial') || lowerContent.includes('how to') ||
          fileName.includes('WALLET') || fileName.includes('CONTROLLER')) {
        categorizedContent.guides.push({file, content, title: fileName});
      }
    } catch (error) {
      console.log(`  ⚠️ Could not read ${file.relativePath}: ${error.message}`);
    }
  });

  return { categories, categorizedContent };
}

// Generate documentation structure (categories disabled)
function generateDocumentationStructure(allReadmeFiles) {
  console.log('🏗️ Skipping documentation structure generation (categories disabled)...');

  ensureDirectoryExists(CONFIG.docsifyDir);

  // Categories are disabled per user request
  console.log('  ✅ Category generation skipped');
}

// Generate sidebar without categories
function generateSidebar(allReadmeFiles) {
  console.log('📋 Generating sidebar...');

  let sidebar = `# ${CONFIG.projectName}\n\n`;

  // Add navigation links
  sidebar += `## Navigation\n\n`;
  sidebar += `* [🏠 Documentation Home](#/)\n`;
  sidebar += `* [🌐 Main Site](https://knirv.com)\n\n`;

  // Add component documentation sections
  CONFIG.subproductDirs.forEach(subproductDir => {
    const readmePath = path.join(rootDir, subproductDir, 'README.md');
    if (fs.existsSync(readmePath)) {
      const title = `${subproductDir} Documentation`;
      sidebar += `## ${title}\n\n`;
      sidebar += `* [${subproductDir} User Guide](${subproductDir.toLowerCase()}/README.md)\n\n`;
    }
  });

  // Add whitepapers section
  sidebar += `## 📄 Whitepapers\n\n`;
  sidebar += `* [📚 View All Whitepapers](whitepapers/README.md)\n\n`;

  // Add footer
  sidebar += `<div class="sidebar-footer">\n\n---\n\n`;
  sidebar += `\n© ${new Date().getFullYear()} ${CONFIG.projectName}\n</div>\n`;

  fs.writeFileSync(path.join(CONFIG.docsifyDir, '_sidebar.md'), sidebar);
  console.log('✅ Sidebar generated');
}

// Copy component documentation
function copyComponentDocumentation(allReadmeFiles) {
  console.log('📄 Copying component documentation...');
  
  CONFIG.subproductDirs.forEach(subproductDir => {
    const readmePath = path.join(rootDir, subproductDir, 'README.md');
    if (fs.existsSync(readmePath)) {
      const targetDir = path.join(CONFIG.docsifyDir, subproductDir.toLowerCase());
      ensureDirectoryExists(targetDir);
      
      const content = fs.readFileSync(readmePath, 'utf8');
      fs.writeFileSync(path.join(targetDir, 'README.md'), content);
      console.log(`  ✅ Copied ${subproductDir} documentation`);
    }
  });
}

// Main function
async function main() {
  console.log('🚀 Starting simple documentation generation...');
  
  try {
    const allReadmeFiles = findAllReadmeFiles();
    generateDocumentationStructure(allReadmeFiles);
    copyComponentDocumentation(allReadmeFiles);
    generateSidebar(allReadmeFiles);
    
    console.log('✅ Documentation generation completed successfully!');
    console.log(`📁 Documentation available at: ${CONFIG.docsifyDir}`);
  } catch (error) {
    console.error('❌ Error during documentation generation:', error);
    process.exit(1);
  }
}

// Run the script
main();
