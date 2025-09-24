#!/usr/bin/env node

import fs from 'fs';
import path from 'path';
import { fileURLToPath } from 'url';

class StaticGenerationTester {
  constructor() {
    const __filename = fileURLToPath(import.meta.url);
    const __dirname = path.dirname(__filename);
    this.staticDir = path.join(__dirname, '../public/documentation/static');
  }

  async test() {
    console.log('🧪 Testing static generation...');
    
    // Check if static directory exists
    if (!fs.existsSync(this.staticDir)) {
      console.error('❌ Static directory does not exist');
      return false;
    }

    // Check for index.html
    const indexPath = path.join(this.staticDir, 'index.html');
    if (!fs.existsSync(indexPath)) {
      console.error('❌ index.html not found in static directory');
      return false;
    }

    // Read and validate index.html
    const indexContent = fs.readFileSync(indexPath, 'utf8');
    
    // Check for essential elements
    const checks = [
      { name: 'HTML structure', test: () => indexContent.includes('<html') },
      { name: 'KNIRV branding', test: () => indexContent.includes('KNIRV') },
      { name: 'Sidebar content', test: () => indexContent.includes('sidebar') },
      { name: 'Navigation links', test: () => indexContent.includes('knirvchain') },
      { name: 'Static CSS', test: () => indexContent.includes('position: fixed') }
    ];

    let allPassed = true;
    for (const check of checks) {
      const passed = check.test();
      console.log(`${passed ? '✅' : '❌'} ${check.name}`);
      if (!passed) allPassed = false;
    }

    // Count generated files
    const files = this.countFiles(this.staticDir);
    console.log(`📊 Generated ${files} static files`);

    if (allPassed && files > 0) {
      console.log('🎉 Static generation test PASSED!');
      return true;
    } else {
      console.log('❌ Static generation test FAILED!');
      return false;
    }
  }

  countFiles(dir) {
    let count = 0;
    const items = fs.readdirSync(dir);
    
    for (const item of items) {
      const fullPath = path.join(dir, item);
      const stat = fs.statSync(fullPath);
      
      if (stat.isDirectory()) {
        count += this.countFiles(fullPath);
      } else if (item.endsWith('.html')) {
        count++;
      }
    }
    
    return count;
  }

  async run() {
    try {
      const success = await this.test();
      process.exit(success ? 0 : 1);
    } catch (error) {
      console.error('❌ Test error:', error);
      process.exit(1);
    }
  }
}

// Run if called directly
if (import.meta.url === `file://${process.argv[1]}`) {
  const tester = new StaticGenerationTester();
  tester.run();
}

export default StaticGenerationTester;
