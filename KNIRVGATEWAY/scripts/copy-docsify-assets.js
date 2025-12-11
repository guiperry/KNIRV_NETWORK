#!/usr/bin/env node

import fs from 'fs';
import path from 'path';
import { fileURLToPath } from 'url';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

/**
 * Copy Docsify assets from node_modules to public directory
 * This ensures Docsify works in CloudFlare Workers without CDN dependencies
 */

const sourceDir = path.join(__dirname, '..', 'node_modules');
const targetDir = path.join(__dirname, '..', 'network-website', 'public', 'assets', 'docsify');

// Ensure target directory exists
function ensureDir(dir) {
    if (!fs.existsSync(dir)) {
        fs.mkdirSync(dir, { recursive: true });
        console.log(`✅ Created directory: ${dir}`);
    }
}

// Copy file with error handling
function copyFile(src, dest) {
    try {
        if (fs.existsSync(src)) {
            fs.copyFileSync(src, dest);
            console.log(`✅ Copied: ${path.basename(dest)}`);
            return true;
        } else {
            console.log(`⚠️  Source not found: ${src}`);
            return false;
        }
    } catch (error) {
        console.error(`❌ Error copying ${src} to ${dest}:`, error.message);
        return false;
    }
}

// Copy directory recursively
function copyDir(src, dest) {
    try {
        if (!fs.existsSync(src)) {
            console.log(`⚠️  Source directory not found: ${src}`);
            return false;
        }
        
        ensureDir(dest);
        const files = fs.readdirSync(src);
        
        for (const file of files) {
            const srcPath = path.join(src, file);
            const destPath = path.join(dest, file);
            
            if (fs.statSync(srcPath).isDirectory()) {
                copyDir(srcPath, destPath);
            } else {
                copyFile(srcPath, destPath);
            }
        }
        return true;
    } catch (error) {
        console.error(`❌ Error copying directory ${src}:`, error.message);
        return false;
    }
}

console.log('🚀 Copying Docsify assets for CloudFlare Worker deployment...');

// Ensure target directory exists
ensureDir(targetDir);

// Assets to copy
const assets = [
    // Core Docsify files
    {
        src: path.join(sourceDir, 'docsify', 'lib', 'docsify.min.js'),
        dest: path.join(targetDir, 'docsify.min.js')
    },
    {
        src: path.join(sourceDir, 'docsify', 'lib', 'themes', 'dark.css'),
        dest: path.join(targetDir, 'dark.css')
    },
    
    // Docsify plugins
    {
        src: path.join(sourceDir, 'docsify', 'lib', 'plugins', 'search.min.js'),
        dest: path.join(targetDir, 'search.min.js')
    },
    
    // Mermaid
    {
        src: path.join(sourceDir, 'mermaid', 'dist', 'mermaid.min.js'),
        dest: path.join(targetDir, 'mermaid.min.js')
    },
    
    // Docsify Mermaid plugin
    {
        src: path.join(sourceDir, 'docsify-mermaid', 'dist', 'docsify-mermaid.js'),
        dest: path.join(targetDir, 'docsify-mermaid.js')
    }
];

// Copy PrismJS components
const prismDir = path.join(targetDir, 'prism');
ensureDir(prismDir);

const prismComponents = [
    'prism-bash.min.js',
    'prism-go.min.js', 
    'prism-yaml.min.js',
    'prism-rust.min.js',
    'prism-javascript.min.js',
    'prism-typescript.min.js'
];

// Copy main assets
let successCount = 0;
let totalCount = assets.length;

for (const asset of assets) {
    if (copyFile(asset.src, asset.dest)) {
        successCount++;
    }
}

// Copy PrismJS components
for (const component of prismComponents) {
    const src = path.join(sourceDir, 'prismjs', 'components', component);
    const dest = path.join(prismDir, component);
    if (copyFile(src, dest)) {
        successCount++;
    }
    totalCount++;
}

console.log(`\n📊 Copy Summary:`);
console.log(`✅ Successfully copied: ${successCount}/${totalCount} files`);

if (successCount === totalCount) {
    console.log('🎉 All Docsify assets copied successfully!');
    console.log('📁 Assets location:', targetDir);
    process.exit(0);
} else {
    console.log('⚠️  Some assets failed to copy. Docsify may not work properly.');
    process.exit(1);
}
