#!/usr/bin/env node

/**
 * PWA Build Script for KNIRVCONTROLLER
 * Generates slim PWA packages for Android and iOS distribution
 */

import fs from 'fs';
import path from 'path';
import { execSync } from 'child_process';
import archiver from 'archiver';

const PWA_CONFIG = {
  android: {
    name: 'KNIRV Controller',
    short_name: 'KNIRV',
    display: 'standalone',
    orientation: 'portrait-primary',
    theme_color: '#1e40af',
    background_color: '#0f172a',
    start_url: '/',
    scope: '/',
    categories: ['productivity', 'developer', 'utilities', 'business'],
    prefer_related_applications: false,
    icons: [
      { src: '/icons/icon-72x72.png', sizes: '72x72', type: 'image/png', purpose: 'any maskable' },
      { src: '/icons/icon-96x96.png', sizes: '96x96', type: 'image/png', purpose: 'any maskable' },
      { src: '/icons/icon-128x128.png', sizes: '128x128', type: 'image/png', purpose: 'any maskable' },
      { src: '/icons/icon-144x144.png', sizes: '144x144', type: 'image/png', purpose: 'any maskable' },
      { src: '/icons/icon-152x152.png', sizes: '152x152', type: 'image/png', purpose: 'any maskable' },
      { src: '/icons/icon-192x192.png', sizes: '192x192', type: 'image/png', purpose: 'any maskable' },
      { src: '/icons/icon-384x384.png', sizes: '384x384', type: 'image/png', purpose: 'any maskable' },
      { src: '/icons/icon-512x512.png', sizes: '512x512', type: 'image/png', purpose: 'any maskable' }
    ]
  },
  ios: {
    name: 'KNIRV Controller',
    short_name: 'KNIRV',
    display: 'standalone',
    orientation: 'portrait-primary',
    theme_color: '#1e40af',
    background_color: '#0f172a',
    start_url: '/',
    scope: '/',
    categories: ['productivity', 'developer', 'utilities', 'business'],
    prefer_related_applications: false,
    icons: [
      { src: '/icons/icon-120x120.png', sizes: '120x120', type: 'image/png', purpose: 'any' },
      { src: '/icons/icon-152x152.png', sizes: '152x152', type: 'image/png', purpose: 'any' },
      { src: '/icons/icon-167x167.png', sizes: '167x167', type: 'image/png', purpose: 'any' },
      { src: '/icons/icon-180x180.png', sizes: '180x180', type: 'image/png', purpose: 'any' },
      { src: '/icons/icon-192x192.png', sizes: '192x192', type: 'image/png', purpose: 'any' },
      { src: '/icons/icon-384x384.png', sizes: '384x384', type: 'image/png', purpose: 'any' },
      { src: '/icons/icon-512x512.png', sizes: '512x512', type: 'image/png', purpose: 'any' }
    ]
  }
};

const ESSENTIAL_FILES = [
  'index.html',
  'manifest.json',
  'sw.js',
  'install.css',
  'install.js',
  'assets/**/*',
  'icons/**/*',
  'build/**/*',
  'vite.svg'
];

const SLIM_PACKAGE_SIZE_LIMIT = 50 * 1024 * 1024; // 50MB limit for slim packages

class PWABuilder {
  constructor() {
    this.distDir = path.resolve('dist');
    this.pwaDir = path.resolve('dist-pwa');
    this.packagesDir = path.resolve('packages');
  }

  async build() {
    console.log('🚀 Building PWA packages...');
    
    // Clean and create directories
    await this.setupDirectories();
    
    // Build the main application first
    await this.buildMainApp();
    
    // Generate platform-specific PWA packages
    await this.buildAndroidPWA();
    await this.buildIOSPWA();
    
    console.log('✅ PWA packages built successfully!');
    console.log(`📦 Packages available in: ${this.packagesDir}`);
  }

  async setupDirectories() {
    console.log('📁 Setting up directories...');
    
    // Clean existing PWA directories
    if (fs.existsSync(this.pwaDir)) {
      fs.rmSync(this.pwaDir, { recursive: true, force: true });
    }
    if (fs.existsSync(this.packagesDir)) {
      fs.rmSync(this.packagesDir, { recursive: true, force: true });
    }
    
    // Create directories
    fs.mkdirSync(this.pwaDir, { recursive: true });
    fs.mkdirSync(this.packagesDir, { recursive: true });
    fs.mkdirSync(path.join(this.pwaDir, 'android'), { recursive: true });
    fs.mkdirSync(path.join(this.pwaDir, 'ios'), { recursive: true });
  }

  async buildMainApp() {
    console.log('🔨 Building main application...');
    
    try {
      // Run the main build process
      execSync('npm run build', { stdio: 'inherit' });
      console.log('✅ Main application built successfully');
    } catch (error) {
      console.error('❌ Failed to build main application:', error.message);
      throw error;
    }
  }

  async buildAndroidPWA() {
    console.log('🤖 Building Android PWA...');
    
    const androidDir = path.join(this.pwaDir, 'android');
    
    // Copy essential files
    await this.copyEssentialFiles(androidDir);

    // Post-process HTML for file:// compatibility and relative paths
    await this.postProcessIndexHtml(androidDir);
    
    // Generate Android-specific manifest (file:// safe with relative paths)
    const androidManifest = {
      ...PWA_CONFIG.android,
      id: 'com.knirv.controller',
      lang: 'en',
      dir: 'ltr',
      start_url: './',
      scope: './',
      icons: (PWA_CONFIG.android.icons || []).map(icon => ({
        ...icon,
        src: icon.src ? icon.src.replace(/^\//, './') : './icons/icon-192x192.png'
      }))
    };
    
    fs.writeFileSync(
      path.join(androidDir, 'manifest.json'),
      JSON.stringify(androidManifest, null, 2)
    );

    // Ensure placeholder icons exist so links don't 404 when opened via file://
    await this.ensureIcons(androidDir);
    
    // Generate Android-specific service worker
    await this.generateServiceWorker(androidDir, 'android');
    
    // Create installation instructions
    await this.createInstallationInstructions(androidDir, 'android');
    
    // Package for distribution
    await this.packagePWA(androidDir, 'knirvcontroller-android-pwa.zip');
    
    console.log('✅ Android PWA built successfully');
  }

  async buildIOSPWA() {
    console.log('🍎 Building iOS PWA...');
    
    const iosDir = path.join(this.pwaDir, 'ios');
    
    // Copy essential files
    await this.copyEssentialFiles(iosDir);

    // Post-process HTML for file:// compatibility and relative paths
    await this.postProcessIndexHtml(iosDir);
    
    // Generate iOS-specific manifest (file:// safe with relative paths)
    const iosManifest = {
      ...PWA_CONFIG.ios,
      id: 'com.knirv.controller.ios',
      lang: 'en',
      dir: 'ltr',
      start_url: './',
      scope: './',
      icons: (PWA_CONFIG.ios.icons || []).map(icon => ({
        ...icon,
        src: icon.src ? icon.src.replace(/^\//, './') : './icons/icon-192x192.png'
      }))
    };
    
    fs.writeFileSync(
      path.join(iosDir, 'manifest.json'),
      JSON.stringify(iosManifest, null, 2)
    );

    // Ensure placeholder icons exist so links don't 404 when opened via file://
    await this.ensureIcons(iosDir);
    
    // Generate iOS-specific service worker
    await this.generateServiceWorker(iosDir, 'ios');
    
    // Create installation instructions
    await this.createInstallationInstructions(iosDir, 'ios');
    
    // Package for distribution
    await this.packagePWA(iosDir, 'knirvcontroller-ios-pwa.zip');
    
    console.log('✅ iOS PWA built successfully');
  }

  async copyEssentialFiles(targetDir) {
    console.log(`📋 Copying essential files to ${targetDir}...`);
    
    for (const filePattern of ESSENTIAL_FILES) {
      const sourcePath = path.join(this.distDir, filePattern);
      const targetPath = path.join(targetDir, filePattern);
      
      if (filePattern.includes('**')) {
        // Handle glob patterns
        const baseDir = filePattern.split('/**')[0];
        const sourceBaseDir = path.join(this.distDir, baseDir);
        const targetBaseDir = path.join(targetDir, baseDir);
        
        if (fs.existsSync(sourceBaseDir)) {
          fs.mkdirSync(targetBaseDir, { recursive: true });
          this.copyRecursive(sourceBaseDir, targetBaseDir);
        }
      } else {
        // Handle individual files
        if (fs.existsSync(sourcePath)) {
          fs.mkdirSync(path.dirname(targetPath), { recursive: true });
          fs.copyFileSync(sourcePath, targetPath);
        }
      }
    }
  }

  copyRecursive(src, dest) {
    const stats = fs.statSync(src);
    
    if (stats.isDirectory()) {
      fs.mkdirSync(dest, { recursive: true });
      const files = fs.readdirSync(src);
      
      for (const file of files) {
        this.copyRecursive(path.join(src, file), path.join(dest, file));
      }
    } else {
      fs.copyFileSync(src, dest);
    }
  }

  async generateServiceWorker(targetDir, platform) {
    console.log(`⚙️ Generating service worker for ${platform}...`);

    // Collect hashed asset references from built index.html
    const assets = await this.extractAssetsFromIndex(targetDir);

    const swContent = `
// KNIRV Controller PWA Service Worker - ${platform.toUpperCase()}
const CACHE_NAME = 'knirv-controller-${platform}-v1.0.0';
const OFFLINE_URL = './index.html';

const ESSENTIAL_RESOURCES = [
  './',
  './index.html',
  './manifest.json',
  ${assets.map(a => `'${a}'`).join(',\n  ')}
];

self.addEventListener('install', event => {
  event.waitUntil(
    caches.open(CACHE_NAME)
      .then(cache => cache.addAll(ESSENTIAL_RESOURCES))
      .then(() => self.skipWaiting())
  );
});

self.addEventListener('activate', event => {
  event.waitUntil(
    caches.keys()
      .then(cacheNames => Promise.all(cacheNames.filter(n => n !== CACHE_NAME).map(n => caches.delete(n))))
      .then(() => self.clients.claim())
  );
});

self.addEventListener('fetch', event => {
  if (event.request.mode === 'navigate') {
    event.respondWith(
      fetch(event.request).catch(() => caches.match('./index.html'))
    );
  } else {
    event.respondWith(
      caches.match(event.request)
        .then(response => response || fetch(event.request))
        .catch(() => {
          if (event.request.destination === 'document') {
            return caches.match('./index.html');
          }
        })
    );
  }
});
`;

    fs.writeFileSync(path.join(targetDir, 'sw.js'), swContent.trim());
  }

  // Parse built index.html to extract asset URLs (scripts, styles, modulepreload, icons)
  async extractAssetsFromIndex(targetDir) {
    const indexPath = path.join(targetDir, 'index.html');
    if (!fs.existsSync(indexPath)) return [];
    const html = fs.readFileSync(indexPath, 'utf-8');

    const refs = new Set();
    const add = (u) => {
      if (!u) return;
      // make relative and ignore external urls
      if (/^https?:/i.test(u) || /^data:/i.test(u)) return;
      const rel = u.startsWith('/') ? `.${u}` : u;
      refs.add(rel);
    };

    // scripts
    for (const m of html.matchAll(/<script[^>]+src="([^"]+)"/g)) add(m[1]);
    // stylesheets
    for (const m of html.matchAll(/<link[^>]+rel=["']stylesheet["'][^>]+href="([^"]+)"/g)) add(m[1]);
    // modulepreload
    for (const m of html.matchAll(/<link[^>]+rel=["']modulepreload["'][^>]+href="([^"]+)"/g)) add(m[1]);
    // icons referenced in head
    for (const m of html.matchAll(/<link[^>]+href="(\.\/icons\/[^"]+)"/g)) add(m[1]);

    return Array.from(refs);
  }

  async createInstallationInstructions(targetDir, platform) {
    console.log(`📖 Creating installation instructions for ${platform}...`);
    
    const instructions = platform === 'android' ? `
# KNIRV Controller - Android Installation

## Quick Install
1. Open this link on your Android device
2. Tap "Add to Home Screen" when prompted
3. The app will be installed like a native app

## Manual Installation
1. Open Chrome on your Android device
2. Navigate to the app URL
3. Tap the menu (⋮) and select "Add to Home Screen"
4. Confirm the installation

## Features
- Works offline
- Push notifications
- Native app experience
- Secure authentication
- Local data storage

## System Requirements
- Android 7.0 or later
- Chrome 70+ or compatible browser
- 50MB free storage space
` : `
# KNIRV Controller - iOS Installation

## Quick Install
1. Open this link on your iPhone/iPad using Safari
2. Tap the Share button (□↗)
3. Select "Add to Home Screen"
4. The app will be installed like a native app

## Manual Installation
1. Open Safari on your iOS device
2. Navigate to the app URL
3. Tap the Share button and select "Add to Home Screen"
4. Confirm the installation

## Features
- Works offline
- Native app experience
- Secure authentication
- Local data storage
- iOS integration

## System Requirements
- iOS 14.0 or later
- Safari 14+ (recommended)
- 50MB free storage space
`;

    fs.writeFileSync(path.join(targetDir, 'INSTALL.md'), instructions.trim());
  }

  async packagePWA(sourceDir, filename) {
    console.log(`📦 Packaging ${filename}...`);
    
    const outputPath = path.join(this.packagesDir, filename);
    const output = fs.createWriteStream(outputPath);
    const archive = archiver('zip', { zlib: { level: 9 } });
    
    return new Promise((resolve, reject) => {
      output.on('close', () => {
        const sizeInMB = (archive.pointer() / 1024 / 1024).toFixed(2);
        console.log(`✅ ${filename} created (${sizeInMB}MB)`);
        
        if (archive.pointer() > SLIM_PACKAGE_SIZE_LIMIT) {
          console.warn(`⚠️  Package size (${sizeInMB}MB) exceeds slim package limit (50MB)`);
        }
        
        resolve();
      });
      
      archive.on('error', reject);
      archive.pipe(output);
      archive.directory(sourceDir, false);
      archive.finalize();
    });
  }

  // Rewrite absolute paths to relative and guard SW registration for file://
  async postProcessIndexHtml(targetDir) {
    const indexPath = path.join(targetDir, 'index.html');
    if (!fs.existsSync(indexPath)) return;
    let html = fs.readFileSync(indexPath, 'utf-8');

    // Replace absolute-leading slashes with relative './'
    html = html
      .replace(/href="\/(manifest\.json)"/g, 'href="./$1"')
      .replace(/href="\/(icons\/[^"]+)"/g, 'href="./$1"')
      .replace(/src="\/(assets\/[^"]+)"/g, 'src="./$1"')
      .replace(/href="\/(assets\/[^"]+)"/g, 'href="./$1"')
      .replace(/src="\/(install\.js)"/g, 'src="./$1"')
      .replace(/href="\/(install\.css)"/g, 'href="./$1"')
      .replace(/href="\/vite\.svg"/g, 'href="./vite.svg"')
      .replace(/serviceWorker\.register\('\/sw\.js'\)/g, "serviceWorker.register('./sw.js')");

    // Guard SW registration to secure contexts only
    html = html.replace(
      /(if \('serviceWorker' in navigator\) \{[\s\S]*?navigator\.serviceWorker\.register\([^\)]*\)[\s\S]*?\}\);)/,
      (match) => {
        return match.replace(
          'window.addEventListener(\'load\', () => {',
          "window.addEventListener('load', () => {\n          const isSecureContext = (location.protocol === 'https:' || location.protocol === 'http:');\n          if (!isSecureContext) { console.log('Skipping SW on non-secure context'); return; }"
        );
      }
    );

    fs.writeFileSync(indexPath, html, 'utf-8');
  }

  // Ensure basic icon set exists by copying from dist/icons or writing placeholders
  async ensureIcons(targetDir) {
    const iconsDir = path.join(targetDir, 'icons');
    if (!fs.existsSync(iconsDir)) fs.mkdirSync(iconsDir, { recursive: true });

    const needed = [
      'icon-16x16.png','icon-32x32.png','icon-72x72.png','icon-96x96.png','icon-120x120.png',
      'icon-128x128.png','icon-144x144.png','icon-152x152.png','icon-167x167.png','icon-180x180.png',
      'icon-192x192.png','icon-384x384.png','icon-512x512.png'
    ];

    for (const name of needed) {
      const dest = path.join(iconsDir, name);
      if (fs.existsSync(dest)) continue;
      const srcInDist = path.join(this.distDir, 'icons', name);
      if (fs.existsSync(srcInDist)) {
        fs.copyFileSync(srcInDist, dest);
      } else {
        // Write a tiny 1x1 PNG placeholder to avoid 404s (transparent pixel)
        const transparentPngBase64 = 'iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR4nGNgYAAAAAMAASsJTYQAAAAASUVORK5CYII=';
        fs.writeFileSync(dest, Buffer.from(transparentPngBase64, 'base64'));
      }
    }

    // Ensure vite.svg placeholder
    const viteSvgDest = path.join(targetDir, 'vite.svg');
    if (!fs.existsSync(viteSvgDest)) {
      const viteSvgSrc = path.join(this.distDir, 'vite.svg');
      const svgContent = fs.existsSync(viteSvgSrc) ? fs.readFileSync(viteSvgSrc, 'utf-8') : '<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16"></svg>';
      fs.writeFileSync(viteSvgDest, svgContent, 'utf-8');
    }
  }
}

new PWABuilder().build().catch(err => {
  console.error('Build failed:', err);
  process.exit(1);
});

// Main execution
if (import.meta.url === `file://${process.argv[1]}`) {
  const builder = new PWABuilder();
  builder.build().catch(error => {
    console.error('❌ PWA build failed:', error);
    process.exit(1);
  });
}

export default PWABuilder;
