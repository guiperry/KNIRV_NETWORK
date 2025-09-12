#!/usr/bin/env node

const fs = require('fs');
const path = require('path');
const { spawn } = require('child_process');
const puppeteer = require('puppeteer');

class DocsifyStaticGenerator {
  constructor() {
    this.docsifyDir = path.join(__dirname, '../public/documentation/docsify');
    this.staticDir = path.join(__dirname, '../public/documentation/static');
    this.baseUrl = 'http://localhost:3000';
    this.server = null;
    this.browser = null;
    this.visitedUrls = new Set();
    this.urlQueue = [];
  }

  async init() {
    console.log('🚀 Starting Docsify Static Generator...');

    // Ensure static directory exists
    if (!fs.existsSync(this.staticDir)) {
      fs.mkdirSync(this.staticDir, { recursive: true });
    }

    // Copy assets to docsify directory so they can be served
    await this.copyAssetsToDocsify();

    // Start docsify server
    await this.startDocsifyServer();

    // Wait for server to be ready
    await this.waitForServer();

    // Start browser
    this.browser = await puppeteer.launch({
      headless: true,
      args: ['--no-sandbox', '--disable-setuid-sandbox']
    });

    console.log('✅ Docsify server and browser ready');
  }

  async copyAssetsToDocsify() {
    console.log('📁 Copying assets to docsify directory...');

    const assetsSource = path.join(__dirname, '../public/assets');
    const assetsTarget = path.join(this.docsifyDir, 'assets');

    // Create assets directory in docsify
    if (!fs.existsSync(assetsTarget)) {
      fs.mkdirSync(assetsTarget, { recursive: true });
    }

    // Copy docsify assets
    const docsifyAssetsSource = path.join(assetsSource, 'docsify');
    const docsifyAssetsTarget = path.join(assetsTarget, 'docsify');

    if (fs.existsSync(docsifyAssetsSource)) {
      this.copyDirectory(docsifyAssetsSource, docsifyAssetsTarget);
      console.log('✅ Docsify assets copied');
    }
  }

  copyDirectory(src, dest) {
    if (!fs.existsSync(dest)) {
      fs.mkdirSync(dest, { recursive: true });
    }

    const items = fs.readdirSync(src);
    for (const item of items) {
      const srcPath = path.join(src, item);
      const destPath = path.join(dest, item);

      if (fs.statSync(srcPath).isDirectory()) {
        this.copyDirectory(srcPath, destPath);
      } else {
        fs.copyFileSync(srcPath, destPath);
      }
    }
  }

  async startDocsifyServer() {
    return new Promise((resolve, reject) => {
      console.log('📡 Starting docsify server...');
      
      this.server = spawn('npx', ['docsify', 'serve', '.', '--port', '3000'], {
        cwd: this.docsifyDir,
        stdio: ['pipe', 'pipe', 'pipe']
      });

      this.server.stdout.on('data', (data) => {
        const output = data.toString();
        console.log(`[Docsify] ${output.trim()}`);
        if (output.includes('Serving') || output.includes('localhost:3000')) {
          resolve();
        }
      });

      this.server.stderr.on('data', (data) => {
        console.error(`[Docsify Error] ${data.toString().trim()}`);
      });

      this.server.on('error', reject);
      
      // Resolve after 3 seconds if no explicit ready message
      setTimeout(resolve, 3000);
    });
  }

  async waitForServer() {
    console.log('⏳ Waiting for server to be ready...');
    let attempts = 0;
    const maxAttempts = 30;

    while (attempts < maxAttempts) {
      try {
        const response = await fetch(this.baseUrl);
        if (response.ok) {
          console.log('✅ Server is ready');
          return;
        }
      } catch (error) {
        // Server not ready yet
      }
      
      await new Promise(resolve => setTimeout(resolve, 1000));
      attempts++;
    }
    
    throw new Error('Server failed to start within 30 seconds');
  }

  async crawlAndGenerate() {
    console.log('🕷️  Starting to crawl docsify site...');
    
    // Start with the main page
    this.urlQueue.push('/');
    
    while (this.urlQueue.length > 0) {
      const urlPath = this.urlQueue.shift();
      
      if (this.visitedUrls.has(urlPath)) {
        continue;
      }
      
      await this.processPage(urlPath);
    }
    
    console.log(`✅ Crawled ${this.visitedUrls.size} pages`);
  }

  async processPage(urlPath) {
    try {
      const fullUrl = `${this.baseUrl}${urlPath}`;
      console.log(`📄 Processing: ${urlPath}`);

      const page = await this.browser.newPage();

      // Set a longer timeout and enable console logging
      page.setDefaultTimeout(30000);

      // Listen for console messages to debug
      page.on('console', msg => {
        if (msg.type() === 'error') {
          console.log(`🔴 Browser error: ${msg.text()}`);
        }
      });

      // Navigate to page
      console.log(`🌐 Navigating to: ${fullUrl}`);
      await page.goto(fullUrl, { waitUntil: 'domcontentloaded' });

      // Wait a bit for initial load
      await new Promise(resolve => setTimeout(resolve, 3000));

      // Check what we have so far
      const initialCheck = await page.evaluate(() => {
        const app = document.querySelector('#app');
        const scripts = document.querySelectorAll('script').length;
        return {
          hasApp: !!app,
          appContent: app ? app.innerHTML.substring(0, 100) : 'No app',
          scriptCount: scripts,
          title: document.title
        };
      });

      console.log(`🔍 Initial check:`, initialCheck);

      // Try to wait for docsify to load, but don't fail if it doesn't
      try {
        await page.waitForFunction(() => {
          const app = document.querySelector('#app');
          return app && (
            app.innerHTML.includes('KNIRV') ||
            app.innerHTML.includes('sidebar') ||
            app.innerHTML.length > 200
          );
        }, { timeout: 15000 });
        console.log(`✅ Docsify content detected`);
      } catch (error) {
        console.log(`⚠️  Docsify may not have loaded fully, capturing anyway...`);
      }

      // Final wait
      await new Promise(resolve => setTimeout(resolve, 2000));
      
      // Get the fully rendered HTML
      const html = await page.evaluate(() => {
        return document.documentElement.outerHTML;
      });

      // Debug: log some info about what we captured
      const appContent = await page.evaluate(() => {
        const app = document.querySelector('#app');
        return app ? app.innerHTML.substring(0, 200) + '...' : 'No #app found';
      });
      console.log(`📋 App content preview: ${appContent}`);
      
      // Extract internal links for further crawling
      const links = await page.evaluate(() => {
        const anchors = Array.from(document.querySelectorAll('a[href]'));
        return anchors
          .map(a => a.getAttribute('href'))
          .filter(href => href && !href.startsWith('http') && !href.startsWith('#') && !href.startsWith('mailto:'))
          .map(href => {
            // Clean up the href
            let cleanHref = href.replace(/\.md$/, '').replace(/\/$/, '');
            if (!cleanHref.startsWith('/')) {
              cleanHref = `/${cleanHref}`;
            }
            return cleanHref;
          })
          .filter(href => href !== '/'); // Don't re-add the root
      });
      
      // Add new links to queue
      console.log(`🔗 Found ${links.length} links on ${urlPath}:`, links.slice(0, 5));
      links.forEach(link => {
        if (!this.visitedUrls.has(link) && !this.urlQueue.includes(link)) {
          this.urlQueue.push(link);
          console.log(`➕ Added to queue: ${link}`);
        }
      });
      
      // Process and save the HTML
      const processedHtml = this.processHtml(html, urlPath);
      await this.saveStaticPage(urlPath, processedHtml);
      
      this.visitedUrls.add(urlPath);
      await page.close();
      
    } catch (error) {
      console.error(`❌ Error processing ${urlPath}:`, error.message);
    }
  }

  processHtml(html, urlPath) {
    // Convert docsify links to static links
    let processed = html;

    // Replace docsify router links with static HTML links (but not CSS/JS/assets)
    processed = processed.replace(/href="([^"]*?)"/g, (match, href) => {
      if (href.startsWith('http') || href.startsWith('#') || href.startsWith('mailto:')) {
        return match;
      }

      // Don't modify CSS, JS, or other asset files
      if (href.includes('.css') || href.includes('.js') || href.includes('.png') ||
          href.includes('.jpg') || href.includes('.ico') || href.includes('.svg') ||
          href.includes('.woff') || href.includes('.ttf') || href.includes('.webmanifest') ||
          href.includes('.xml')) {
        return match;
      }

      // Convert to static HTML path for navigation links only
      let staticPath = href;
      if (!staticPath.endsWith('.html') && !staticPath.endsWith('/')) {
        staticPath += '.html';
      }

      return `href="${staticPath}"`;
    });
    
    // Remove docsify scripts and add static navigation
    processed = processed.replace(
      /<script[^>]*docsify[^>]*><\/script>/gi, 
      ''
    );
    
    // Add static CSS and remove dynamic docsify behavior
    processed = processed.replace(
      '</head>',
      `<style>
        /* Static navigation styles */
        .sidebar { position: fixed !important; }
        .content { margin-left: 300px !important; }
        @media (max-width: 768px) {
          .sidebar { transform: translateX(-300px); }
          .content { margin-left: 0 !important; }
        }
      </style>
      </head>`
    );
    
    return processed;
  }

  async saveStaticPage(urlPath, html) {
    let filePath;
    
    if (urlPath === '/') {
      filePath = path.join(this.staticDir, 'index.html');
    } else {
      // Create directory structure
      const cleanPath = urlPath.replace(/^\//, '').replace(/\/$/, '');
      filePath = path.join(this.staticDir, `${cleanPath}.html`);
      
      // Ensure directory exists
      const dir = path.dirname(filePath);
      if (!fs.existsSync(dir)) {
        fs.mkdirSync(dir, { recursive: true });
      }
    }
    
    fs.writeFileSync(filePath, html, 'utf8');
    console.log(`💾 Saved: ${filePath}`);
  }

  async cleanup() {
    console.log('🧹 Cleaning up...');
    
    if (this.browser) {
      await this.browser.close();
    }
    
    if (this.server) {
      this.server.kill();
    }
    
    console.log('✅ Cleanup complete');
  }

  async run() {
    try {
      await this.init();
      await this.crawlAndGenerate();
      console.log('🎉 Static site generation complete!');
      console.log(`📁 Static files saved to: ${this.staticDir}`);
    } catch (error) {
      console.error('❌ Error:', error);
      process.exit(1);
    } finally {
      await this.cleanup();
    }
  }
}

// Run if called directly
if (require.main === module) {
  const generator = new DocsifyStaticGenerator();
  generator.run();
}

module.exports = DocsifyStaticGenerator;
