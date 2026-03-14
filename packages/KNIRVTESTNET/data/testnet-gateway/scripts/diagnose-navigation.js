#!/usr/bin/env node

const fs = require('fs');
const path = require('path');

function checkDocsifyConfig() {
    console.log('🔍 Checking Docsify Configuration...\n');
    
    const indexPath = path.join(__dirname, '..', 'documentation', 'docsify', 'index.html');
    
    if (!fs.existsSync(indexPath)) {
        console.log('❌ index.html not found');
        return false;
    }
    
    const content = fs.readFileSync(indexPath, 'utf8');
    
    // Extract docsify config
    const configMatch = content.match(/window\.\$docsify\s*=\s*({[\s\S]*?});/);
    if (!configMatch) {
        console.log('❌ Docsify config not found in index.html');
        return false;
    }
    
    console.log('✅ Docsify config found:');
    console.log(configMatch[1]);
    console.log('');
    
    // Check for common issues
    const config = configMatch[1];
    
    if (config.includes('basePath')) {
        console.log('⚠️  basePath is set - this might cause routing issues');
    }
    
    if (config.includes('relativePath: true')) {
        console.log('⚠️  relativePath is true - this might cause issues with absolute paths');
    }
    
    if (!config.includes('loadSidebar: true')) {
        console.log('❌ loadSidebar is not enabled');
    } else {
        console.log('✅ loadSidebar is enabled');
    }
    
    return true;
}

function checkSidebarFile() {
    console.log('🔍 Checking Sidebar File...\n');
    
    const sidebarPath = path.join(__dirname, '..', 'documentation', 'docsify', '_sidebar.md');
    
    if (!fs.existsSync(sidebarPath)) {
        console.log('❌ _sidebar.md not found');
        return false;
    }
    
    const content = fs.readFileSync(sidebarPath, 'utf8');
    console.log('✅ _sidebar.md exists');
    console.log(`📏 Size: ${content.length} characters`);
    
    // Check for common markdown issues
    const lines = content.split('\n');
    let linkCount = 0;
    let issues = [];
    
    for (let i = 0; i < lines.length; i++) {
        const line = lines[i];
        const linkMatch = line.match(/\[([^\]]+)\]\(([^)]+)\)/);
        
        if (linkMatch) {
            linkCount++;
            const linkText = linkMatch[1];
            const linkUrl = linkMatch[2];
            
            // Check if target file exists
            const targetPath = path.join(__dirname, '..', 'documentation', 'docsify', linkUrl);
            if (!fs.existsSync(targetPath)) {
                issues.push(`Line ${i + 1}: Link "${linkText}" points to non-existent file: ${linkUrl}`);
            }
        }
    }
    
    console.log(`📊 Found ${linkCount} links in sidebar`);
    
    if (issues.length > 0) {
        console.log('❌ Issues found:');
        issues.forEach(issue => console.log(`  ${issue}`));
    } else {
        console.log('✅ All sidebar links point to existing files');
    }
    
    console.log('');
    return issues.length === 0;
}

function checkFileStructure() {
    console.log('🔍 Checking File Structure...\n');
    
    const docsPath = path.join(__dirname, '..', 'documentation', 'docsify');
    const requiredFiles = [
        'README.md',
        '_sidebar.md',
        'index.html',
        'guides/README.md',
        'api/README.md',
        'legal/CODE_OF_CONDUCT.md'
    ];
    
    let allExist = true;
    
    for (const file of requiredFiles) {
        const filePath = path.join(docsPath, file);
        if (fs.existsSync(filePath)) {
            console.log(`✅ ${file}`);
        } else {
            console.log(`❌ ${file} - MISSING`);
            allExist = false;
        }
    }
    
    console.log('');
    return allExist;
}

function checkServerResponse() {
    console.log('🔍 Checking Server Response...\n');
    
    const http = require('http');
    
    return new Promise((resolve) => {
        const req = http.get('http://localhost:3000', (res) => {
            console.log(`✅ Server responding with status: ${res.statusCode}`);
            
            let data = '';
            res.on('data', chunk => data += chunk);
            res.on('end', () => {
                if (data.includes('window.$docsify')) {
                    console.log('✅ Docsify script found in response');
                } else {
                    console.log('❌ Docsify script not found in response');
                }
                
                if (data.includes('loadSidebar')) {
                    console.log('✅ loadSidebar config found');
                } else {
                    console.log('❌ loadSidebar config not found');
                }
                
                console.log('');
                resolve(true);
            });
        });
        
        req.on('error', (err) => {
            console.log(`❌ Server not responding: ${err.message}`);
            console.log('');
            resolve(false);
        });
        
        req.setTimeout(5000, () => {
            console.log('❌ Server response timeout');
            console.log('');
            resolve(false);
        });
    });
}

async function main() {
    console.log('🚀 KNIRV Documentation Navigation Diagnostics\n');
    console.log('=' .repeat(50) + '\n');
    
    const configOk = checkDocsifyConfig();
    const sidebarOk = checkSidebarFile();
    const filesOk = checkFileStructure();
    const serverOk = await checkServerResponse();
    
    console.log('📋 SUMMARY\n');
    console.log(`Docsify Config: ${configOk ? '✅' : '❌'}`);
    console.log(`Sidebar File: ${sidebarOk ? '✅' : '❌'}`);
    console.log(`File Structure: ${filesOk ? '✅' : '❌'}`);
    console.log(`Server Response: ${serverOk ? '✅' : '❌'}`);
    
    if (configOk && sidebarOk && filesOk && serverOk) {
        console.log('\n🎉 All checks passed! Navigation should be working.');
        console.log('\nIf navigation still doesn\'t work, the issue might be:');
        console.log('1. Browser cache - try hard refresh (Ctrl+F5)');
        console.log('2. JavaScript errors - check browser console');
        console.log('3. Network issues - check browser network tab');
    } else {
        console.log('\n⚠️  Issues detected that may cause navigation problems.');
    }
}

if (require.main === module) {
    main().catch(console.error);
}
