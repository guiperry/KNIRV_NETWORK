#!/usr/bin/env node

const fs = require('fs');
const path = require('path');

// New Markdown footer
const newFooter = `
---

**Links:** [Code of Conduct](#/legal/CODE_OF_CONDUCT) | [Privacy Policy](#/legal/PRIVACY_POLICY) | [Terms and Conditions](#/legal/TERMS_AND_CONDITIONS)

© 2025 KNIRV Network`;

// Function to fix footer in a file
function fixFooter(filePath) {
    try {
        let content = fs.readFileSync(filePath, 'utf8');
        
        // Check if file has HTML footer
        if (content.includes('<div class="footer-links">')) {
            console.log(`Fixing footer in ${filePath}`);
            
            // Remove HTML footer section
            content = content.replace(/<div class="footer-links">[\s\S]*?<\/div>/g, '');
            
            // Remove any trailing whitespace
            content = content.trim();
            
            // Add new Markdown footer
            content += newFooter;
            
            // Write back to file
            fs.writeFileSync(filePath, content);
            
            return true;
        }
        return false;
    } catch (error) {
        console.error(`Error processing ${filePath}:`, error.message);
        return false;
    }
}

// Find all README files in documentation
function findReadmeFiles(dir) {
    const files = [];
    
    function scanDir(currentDir) {
        const items = fs.readdirSync(currentDir);
        
        for (const item of items) {
            const fullPath = path.join(currentDir, item);
            const stat = fs.statSync(fullPath);
            
            if (stat.isDirectory()) {
                scanDir(fullPath);
            } else if (item === 'README.md') {
                files.push(fullPath);
            }
        }
    }
    
    scanDir(dir);
    return files;
}

// Main execution
const docsDir = 'KNIRVGATEWAY/documentation/docsify';
const readmeFiles = findReadmeFiles(docsDir);

console.log(`Found ${readmeFiles.length} README files`);

let fixedCount = 0;
for (const file of readmeFiles) {
    if (fixFooter(file)) {
        fixedCount++;
    }
}

console.log(`Fixed ${fixedCount} files`);
console.log('Footer fixes complete!');
