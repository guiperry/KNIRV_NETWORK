#!/usr/bin/env node

// Script to generate tsconfig.json based on deployment environment
// This allows conditional inclusion of Cloudflare types

import fs from 'fs';
import path from 'path';
import { fileURLToPath } from 'url';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

// Detect environment
const isCloudflare = process.env.CF_PAGES === '1' || 
                     process.env.WRANGLER_SESSION === 'true' ||
                     process.argv.includes('--cloudflare');
const isRender = process.env.RENDER === 'true' || 
                 process.argv.includes('--render');

console.log(`Environment detection:`);
console.log(`- Cloudflare: ${isCloudflare}`);
console.log(`- Render: ${isRender}`);

// Base tsconfig
const baseConfig = {
  compilerOptions: {
    target: "esnext",
    module: "esnext",
    moduleResolution: "node",
    lib: ["esnext", "webworker"],
    jsx: "react-jsx",
    strict: true,
    esModuleInterop: true,
    allowSyntheticDefaultImports: true,
    skipLibCheck: true,
    forceConsistentCasingInFileNames: true,
    resolveJsonModule: true,
    isolatedModules: true,
    noEmit: true
  },
  include: ["./index.ts"],
  exclude: ["node_modules"]
};

// Add Cloudflare-specific configuration if needed
if (isCloudflare) {
  baseConfig.compilerOptions.types = ["@cloudflare/workers-types"];
  baseConfig.compilerOptions.typeRoots = ["./node_modules/@types", "./node_modules/@cloudflare"];
  baseConfig.compilerOptions.skipLibCheck = true;
  console.log('Using Cloudflare Workers types configuration');
} else {
  // For Render or other environments, completely skip Cloudflare types
  baseConfig.compilerOptions.skipLibCheck = true;
  delete baseConfig.compilerOptions.types;
  delete baseConfig.compilerOptions.typeRoots;
  console.log('Using simplified configuration for Render (skipLibCheck enabled)');
}

// Write the generated tsconfig
const tsconfigPath = path.join(__dirname, '..', 'tsconfig.json');
fs.writeFileSync(tsconfigPath, JSON.stringify(baseConfig, null, 2));
console.log(`Generated tsconfig.json at ${tsconfigPath}`);

// Also create a backup of the original if it exists
const originalPath = path.join(__dirname, '..', 'tsconfig.original.json');
if (!fs.existsSync(originalPath)) {
  const currentPath = path.join(__dirname, '..', 'tsconfig.json');
  if (fs.existsSync(currentPath)) {
    const currentConfig = fs.readFileSync(currentPath, 'utf8');
    fs.writeFileSync(originalPath, currentConfig);
    console.log(`Backed up original tsconfig to ${originalPath}`);
  }
}

// For Render builds, also create a modified index.ts without Cloudflare reference
if (!isCloudflare) {
  const indexPath = path.join(__dirname, '..', 'index.ts');
  if (fs.existsSync(indexPath)) {
    let indexContent = fs.readFileSync(indexPath, 'utf8');
    
    // Remove the Cloudflare types reference
    indexContent = indexContent.replace('/// <reference types="@cloudflare/workers-types" />\n', '');
    
    // Create a backup of the original index.ts
    const indexBackupPath = path.join(__dirname, '..', 'index.original.ts');
    if (!fs.existsSync(indexBackupPath)) {
      fs.writeFileSync(indexBackupPath, fs.readFileSync(indexPath, 'utf8'));
      console.log(`Backed up original index.ts to ${indexBackupPath}`);
    }
    
    // Write the modified index.ts
    fs.writeFileSync(indexPath, indexContent);
    console.log('Removed Cloudflare types reference from index.ts for Render build');
  }
} else {
  // For Cloudflare builds, restore the original index.ts if it exists
  const indexBackupPath = path.join(__dirname, '..', 'index.original.ts');
  const indexPath = path.join(__dirname, '..', 'index.ts');
  if (fs.existsSync(indexBackupPath) && fs.existsSync(indexPath)) {
    const originalContent = fs.readFileSync(indexBackupPath, 'utf8');
    fs.writeFileSync(indexPath, originalContent);
    console.log('Restored original index.ts with Cloudflare types reference');
  }
}