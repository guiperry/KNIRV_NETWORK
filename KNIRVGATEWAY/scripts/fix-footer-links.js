#!/usr/bin/env node

const fs = require('fs');
const path = require('path');

// Function to find all markdown files
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

// Function to fix footer links in a file
function fixFooterLinks(filePath) {
    try {
        let content = fs.readFileSync(filePath, 'utf8');
        let changed = false;
        
        // Fix the footer links to use proper docsify navigation
        const oldFooter = /<div class="footer-links">\s*<a href="#\/legal\/CODE_OF_CONDUCT\.md" class="footer-link">Contributor Covenant Code of Conduct<\/a> \| <a href="#\/legal\/PRIVACY_POLICY\.md" class="footer-link">PRIVACY_POLICY\.md<\/a> \| <a href="#\/legal\/TERMS_AND_CONDITIONS\.md" class="footer-link">TERMS AND CONDITIONS<\/a>\s*© 2025 KNIRV Network\s*<\/div>/g;
        
        const newFooter = `<div class="footer-links">
<a href="#/legal/CODE_OF_CONDUCT" class="footer-link">Code of Conduct</a> | <a href="#/legal/PRIVACY_POLICY" class="footer-link">Privacy Policy</a> | <a href="#/legal/TERMS_AND_CONDITIONS" class="footer-link">Terms and Conditions</a>

© 2025 KNIRV Network
</div>`;
        
        if (oldFooter.test(content)) {
            content = content.replace(oldFooter, newFooter);
            changed = true;
        }
        
        // Also fix any standalone footer link patterns
        content = content.replace(/#\/legal\/CODE_OF_CONDUCT\.md/g, '#/legal/CODE_OF_CONDUCT');
        content = content.replace(/#\/legal\/PRIVACY_POLICY\.md/g, '#/legal/PRIVACY_POLICY');
        content = content.replace(/#\/legal\/TERMS_AND_CONDITIONS\.md/g, '#/legal/TERMS_AND_CONDITIONS');
        
        if (changed || content !== fs.readFileSync(filePath, 'utf8')) {
            fs.writeFileSync(filePath, content);
            return true;
        }
        
        return false;
    } catch (error) {
        console.error(`Error processing ${filePath}:`, error.message);
        return false;
    }
}

function main() {
    const docsDir = path.join(__dirname, '..', 'documentation', 'docsify');
    
    if (!fs.existsSync(docsDir)) {
        console.error('Documentation directory not found:', docsDir);
        process.exit(1);
    }
    
    console.log('Fixing footer links in documentation...\n');
    
    const markdownFiles = findMarkdownFiles(docsDir);
    let fixedCount = 0;
    
    for (const file of markdownFiles) {
        const relativePath = path.relative(docsDir, file);
        
        if (fixFooterLinks(file)) {
            console.log(`✅ Fixed footer links: ${relativePath}`);
            fixedCount++;
        } else {
            console.log(`⏭️  No changes needed: ${relativePath}`);
        }
    }
    
    console.log(`\nFixed footer links in ${fixedCount} files.`);
}

if (require.main === module) {
    main();
}
