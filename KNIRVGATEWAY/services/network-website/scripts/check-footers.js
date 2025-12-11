#!/usr/bin/env node

const fs = require('fs');
const path = require('path');

const FOOTER_PATTERN = /© 2025 KNIRV Network/;
const FOOTER_TEMPLATE = `
<div class="footer-links">
<a href="#/legal/CODE_OF_CONDUCT" class="footer-link">Code of Conduct</a> | <a href="#/legal/PRIVACY_POLICY" class="footer-link">Privacy Policy</a> | <a href="#/legal/TERMS_AND_CONDITIONS" class="footer-link">Terms and Conditions</a>

© 2025 KNIRV Network
</div>
`;

function findMarkdownFiles(dir) {
    const files = [];
    
    function traverse(currentDir) {
        const items = fs.readdirSync(currentDir);
        
        for (const item of items) {
            const fullPath = path.join(currentDir, item);
            const stat = fs.statSync(fullPath);
            
            if (stat.isDirectory()) {
                traverse(fullPath);
            } else if (item.endsWith('.md')) {
                files.push(fullPath);
            }
        }
    }
    
    traverse(dir);
    return files;
}

function checkFooter(filePath) {
    try {
        const content = fs.readFileSync(filePath, 'utf8');
        return FOOTER_PATTERN.test(content);
    } catch (error) {
        console.error(`Error reading ${filePath}:`, error.message);
        return false;
    }
}

function addFooter(filePath) {
    try {
        const content = fs.readFileSync(filePath, 'utf8');
        
        // Don't add footer if it already exists
        if (FOOTER_PATTERN.test(content)) {
            return false;
        }
        
        // Add footer at the end
        const newContent = content.trimEnd() + '\n' + FOOTER_TEMPLATE;
        fs.writeFileSync(filePath, newContent);
        return true;
    } catch (error) {
        console.error(`Error adding footer to ${filePath}:`, error.message);
        return false;
    }
}

function main() {
    const docsDir = path.join(__dirname, '..', 'documentation', 'docsify');
    
    if (!fs.existsSync(docsDir)) {
        console.error('Documentation directory not found:', docsDir);
        process.exit(1);
    }
    
    console.log('Checking for missing footers in documentation...\n');
    
    const markdownFiles = findMarkdownFiles(docsDir);
    const filesWithoutFooters = [];
    
    for (const file of markdownFiles) {
        const hasFooter = checkFooter(file);
        const relativePath = path.relative(docsDir, file);
        
        if (!hasFooter) {
            filesWithoutFooters.push(file);
            console.log(`❌ Missing footer: ${relativePath}`);
        } else {
            console.log(`✅ Has footer: ${relativePath}`);
        }
    }
    
    console.log(`\nSummary:`);
    console.log(`Total files: ${markdownFiles.length}`);
    console.log(`Files with footers: ${markdownFiles.length - filesWithoutFooters.length}`);
    console.log(`Files missing footers: ${filesWithoutFooters.length}`);
    
    if (filesWithoutFooters.length > 0) {
        console.log('\nFiles missing footers:');
        filesWithoutFooters.forEach(file => {
            console.log(`  - ${path.relative(docsDir, file)}`);
        });
        
        // Ask if user wants to add footers
        if (process.argv.includes('--fix')) {
            console.log('\nAdding footers to files...');
            let addedCount = 0;
            
            for (const file of filesWithoutFooters) {
                if (addFooter(file)) {
                    addedCount++;
                    console.log(`✅ Added footer to: ${path.relative(docsDir, file)}`);
                }
            }
            
            console.log(`\nAdded footers to ${addedCount} files.`);
        } else {
            console.log('\nTo add footers automatically, run: node scripts/check-footers.js --fix');
        }
    }
}

if (require.main === module) {
    main();
}

module.exports = { findMarkdownFiles, checkFooter, addFooter, FOOTER_TEMPLATE };
