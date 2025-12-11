#!/usr/bin/env node

const fs = require('fs');
const path = require('path');

// Function to extract links from markdown content
function extractLinks(content) {
    const linkRegex = /\[([^\]]+)\]\(([^)]+)\)/g;
    const links = [];
    let match;
    
    while ((match = linkRegex.exec(content)) !== null) {
        links.push({
            text: match[1],
            url: match[2]
        });
    }
    
    return links;
}

// Function to check if a file exists
function checkFileExists(basePath, linkUrl) {
    // Handle different link formats
    let filePath;
    
    if (linkUrl.startsWith('#/')) {
        // Docsify hash-based navigation
        const cleanUrl = linkUrl.substring(2); // Remove #/
        filePath = path.join(basePath, cleanUrl);
        
        // If it doesn't end with .md, try adding it
        if (!cleanUrl.endsWith('.md')) {
            filePath = filePath + '.md';
        }
    } else if (linkUrl.startsWith('http')) {
        // External link - assume it's valid
        return { exists: true, type: 'external' };
    } else {
        // Relative path
        filePath = path.join(basePath, linkUrl);
    }
    
    const exists = fs.existsSync(filePath);
    return { 
        exists, 
        type: 'internal', 
        resolvedPath: filePath,
        relativePath: path.relative(basePath, filePath)
    };
}

// Function to test navigation in a file
function testFileNavigation(filePath, docsPath) {
    const content = fs.readFileSync(filePath, 'utf8');
    const links = extractLinks(content);
    const results = [];
    
    for (const link of links) {
        const fileCheck = checkFileExists(docsPath, link.url);
        results.push({
            text: link.text,
            url: link.url,
            ...fileCheck
        });
    }
    
    return results;
}

function main() {
    const docsPath = path.join(__dirname, '..', 'documentation', 'docsify');
    
    if (!fs.existsSync(docsPath)) {
        console.error('Documentation directory not found:', docsPath);
        process.exit(1);
    }
    
    console.log('Testing navigation links in documentation...\n');
    
    // Test key files
    const testFiles = [
        'README.md',
        '_sidebar.md',
        'guides/README.md',
        'api/README.md',
        'legal/CODE_OF_CONDUCT.md'
    ];
    
    let totalLinks = 0;
    let brokenLinks = 0;
    
    for (const testFile of testFiles) {
        const filePath = path.join(docsPath, testFile);
        
        if (!fs.existsSync(filePath)) {
            console.log(`❌ File not found: ${testFile}`);
            continue;
        }
        
        console.log(`📄 Testing links in: ${testFile}`);
        
        const results = testFileNavigation(filePath, docsPath);
        
        for (const result of results) {
            totalLinks++;
            
            if (result.type === 'external') {
                console.log(`  🌐 External: ${result.text} → ${result.url}`);
            } else if (result.exists) {
                console.log(`  ✅ Valid: ${result.text} → ${result.url}`);
            } else {
                console.log(`  ❌ Broken: ${result.text} → ${result.url} (resolved to: ${result.relativePath})`);
                brokenLinks++;
            }
        }
        
        console.log('');
    }
    
    console.log(`Summary:`);
    console.log(`Total links tested: ${totalLinks}`);
    console.log(`Broken links: ${brokenLinks}`);
    console.log(`Success rate: ${((totalLinks - brokenLinks) / totalLinks * 100).toFixed(1)}%`);
    
    if (brokenLinks === 0) {
        console.log('🎉 All navigation links are working correctly!');
    } else {
        console.log('⚠️  Some navigation links need attention.');
    }
}

if (require.main === module) {
    main();
}
