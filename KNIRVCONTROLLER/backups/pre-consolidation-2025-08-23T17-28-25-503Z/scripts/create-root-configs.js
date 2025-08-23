#!/usr/bin/env node

/**
 * Creates unified configuration files for root consolidation
 * Merges configurations from frontend, backend, and manager
 */

import fs from 'fs/promises';
import path from 'path';
import { fileURLToPath } from 'url';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);
const rootDir = path.join(__dirname, '..');

class ConfigCreator {
  constructor() {
    this.rootDir = rootDir;
  }

  log(message, type = 'info') {
    const prefix = type === 'error' ? '❌' : type === 'success' ? '✅' : 'ℹ️';
    console.log(`${prefix} ${message}`);
  }

  async createViteConfig() {
    this.log('Creating unified vite.config.ts...');
    
    const viteConfig = `import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';
import { resolve } from 'path';
import topLevelAwait from 'vite-plugin-top-level-await';
import wasm from 'vite-plugin-wasm';

export default defineConfig({
  plugins: [
    react(),
    wasm(),
    topLevelAwait({
      promiseExportName: '__tla',
      promiseImportName: i => \`__tla_\${i}\`
    })
  ],
  
  // Root source directory
  root: '.',
  
  // Build configuration
  build: {
    outDir: 'dist',
    sourcemap: true,
    rollupOptions: {
      input: {
        main: resolve(__dirname, 'index.html')
      }
    }
  },
  
  // Development server
  server: {
    host: '0.0.0.0',
    port: 3000,
    open: false,
    proxy: {
      '/api': {
        target: 'http://localhost:3001',
        changeOrigin: true,
        secure: false
      },
      '/ws': {
        target: 'ws://localhost:3001',
        ws: true
      }
    }
  },
  
  // Preview server (production)
  preview: {
    host: '0.0.0.0',
    port: 3000,
    open: false
  },
  
  // Path resolution
  resolve: {
    alias: {
      '@': resolve(__dirname, 'src'),
      '@components': resolve(__dirname, 'src/components'),
      '@pages': resolve(__dirname, 'src/pages'),
      '@backend': resolve(__dirname, 'src/backend'),
      '@manager': resolve(__dirname, 'src/manager'),
      '@shared': resolve(__dirname, 'src/shared'),
      '@sensory-shell': resolve(__dirname, 'src/sensory-shell'),
      '@wasm': resolve(__dirname, 'src/wasm-pkg')
    }
  },
  
  // Optimization
  optimizeDeps: {
    include: [
      'react',
      'react-dom',
      'react-router-dom',
      'lucide-react',
      '@tensorflow/tfjs',
      'three'
    ],
    exclude: ['@napi-rs/wasm-runtime']
  },
  
  // WASM support
  worker: {
    format: 'es',
    plugins: [wasm(), topLevelAwait()]
  },
  
  // Environment variables
  define: {
    __DEV__: JSON.stringify(process.env.NODE_ENV === 'development'),
    __PROD__: JSON.stringify(process.env.NODE_ENV === 'production')
  }
});`;
    
    await fs.writeFile(path.join(this.rootDir, 'vite.config.ts'), viteConfig);
    this.log('vite.config.ts created', 'success');
  }

  async createTsConfig() {
    this.log('Creating unified tsconfig.json...');
    
    const tsConfig = {
      compilerOptions: {
        target: 'ES2022',
        lib: ['ES2022', 'DOM', 'DOM.Iterable'],
        allowJs: false,
        skipLibCheck: true,
        esModuleInterop: false,
        allowSyntheticDefaultImports: true,
        strict: true,
        forceConsistentCasingInFileNames: true,
        module: 'ESNext',
        moduleResolution: 'bundler',
        resolveJsonModule: true,
        isolatedModules: true,
        noEmit: true,
        jsx: 'react-jsx',
        declaration: true,
        declarationMap: true,
        sourceMap: true,
        outDir: 'dist',
        rootDir: 'src',
        baseUrl: '.',
        paths: {
          '@/*': ['src/*'],
          '@components/*': ['src/components/*'],
          '@pages/*': ['src/pages/*'],
          '@backend/*': ['src/backend/*'],
          '@manager/*': ['src/manager/*'],
          '@shared/*': ['src/shared/*'],
          '@sensory-shell/*': ['src/sensory-shell/*'],
          '@wasm/*': ['src/wasm-pkg/*']
        },
        types: ['vite/client', 'jest', '@testing-library/jest-dom', 'node']
      },
      include: [
        'src/**/*',
        'tests/**/*',
        'vite.config.ts',
        'jest.config.js'
      ],
      exclude: [
        'node_modules',
        'dist',
        'build',
        'coverage',
        'backups'
      ],
      references: [
        { path: './tsconfig.backend.json' }
      ]
    };
    
    await fs.writeFile(path.join(this.rootDir, 'tsconfig.json'), JSON.stringify(tsConfig, null, 2));
    this.log('tsconfig.json created', 'success');
  }

  async createBackendTsConfig() {
    this.log('Creating tsconfig.backend.json...');
    
    const backendTsConfig = {
      extends: './tsconfig.json',
      compilerOptions: {
        target: 'ES2022',
        module: 'ESNext',
        moduleResolution: 'node',
        noEmit: false,
        outDir: 'dist/backend',
        rootDir: 'src/backend',
        declaration: true,
        declarationMap: true,
        sourceMap: true,
        types: ['node']
      },
      include: [
        'src/backend/**/*'
      ],
      exclude: [
        'src/backend/**/*.test.ts',
        'src/backend/**/*.spec.ts'
      ]
    };
    
    await fs.writeFile(path.join(this.rootDir, 'tsconfig.backend.json'), JSON.stringify(backendTsConfig, null, 2));
    this.log('tsconfig.backend.json created', 'success');
  }

  async createIndexHtml() {
    this.log('Creating root index.html...');
    
    const indexHtml = `<!DOCTYPE html>
<html lang="en">
  <head>
    <meta charset="UTF-8" />
    <link rel="icon" type="image/svg+xml" href="/vite.svg" />
    <meta name="viewport" content="width=device-width, initial-scale=1.0" />
    <title>KNIRV Controller - Unified Interface</title>
    <meta name="description" content="KNIRV Controller - Unified cognitive shell and management interface" />
    
    <!-- Preload critical resources -->
    <link rel="preload" href="/src/main.tsx" as="script" type="module" />
    
    <!-- Theme and styling -->
    <style>
      body {
        margin: 0;
        padding: 0;
        font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', 'Roboto', 'Oxygen',
          'Ubuntu', 'Cantarell', 'Fira Sans', 'Droid Sans', 'Helvetica Neue',
          sans-serif;
        -webkit-font-smoothing: antialiased;
        -moz-osx-font-smoothing: grayscale;
        background-color: #111827;
        color: #ffffff;
      }
      
      #root {
        width: 100vw;
        height: 100vh;
        overflow: hidden;
      }
      
      /* Loading screen */
      .loading-screen {
        position: fixed;
        top: 0;
        left: 0;
        width: 100%;
        height: 100%;
        background: linear-gradient(135deg, #1f2937 0%, #111827 100%);
        display: flex;
        flex-direction: column;
        align-items: center;
        justify-content: center;
        z-index: 9999;
      }
      
      .loading-spinner {
        width: 64px;
        height: 64px;
        border: 4px solid #374151;
        border-top: 4px solid #3b82f6;
        border-radius: 50%;
        animation: spin 1s linear infinite;
        margin-bottom: 24px;
      }
      
      @keyframes spin {
        0% { transform: rotate(0deg); }
        100% { transform: rotate(360deg); }
      }
      
      .loading-text {
        color: #9ca3af;
        font-size: 18px;
        font-weight: 500;
      }
    </style>
  </head>
  <body>
    <div id="root">
      <!-- Loading screen shown while app initializes -->
      <div class="loading-screen" id="loading-screen">
        <div class="loading-spinner"></div>
        <div class="loading-text">Initializing KNIRV Controller...</div>
      </div>
    </div>
    
    <script type="module" src="/src/main.tsx"></script>
    
    <script>
      // Hide loading screen once app is ready
      window.addEventListener('load', () => {
        setTimeout(() => {
          const loadingScreen = document.getElementById('loading-screen');
          if (loadingScreen) {
            loadingScreen.style.opacity = '0';
            loadingScreen.style.transition = 'opacity 0.5s ease-out';
            setTimeout(() => {
              loadingScreen.remove();
            }, 500);
          }
        }, 1000);
      });
      
      // Error handling
      window.addEventListener('error', (event) => {
        console.error('Application error:', event.error);
      });
      
      window.addEventListener('unhandledrejection', (event) => {
        console.error('Unhandled promise rejection:', event.reason);
      });
    </script>
  </body>
</html>`;
    
    await fs.writeFile(path.join(this.rootDir, 'index.html'), indexHtml);
    this.log('index.html created', 'success');
  }

  async createMainTsx() {
    this.log('Creating src/main.tsx...');
    
    await fs.mkdir(path.join(this.rootDir, 'src'), { recursive: true });
    
    const mainTsx = `import React from 'react';
import ReactDOM from 'react-dom/client';
import App from './App';
import './index.css';

// Initialize WASM modules
async function initializeWasm() {
  try {
    // Import WASM modules if available
    const wasmModule = await import('./wasm-pkg/knirv_cortex_wasm');
    await wasmModule.default();
    console.log('WASM modules initialized successfully');
  } catch (error) {
    console.warn('WASM modules not available or failed to initialize:', error);
  }
}

// Initialize application
async function initializeApp() {
  try {
    // Initialize WASM first
    await initializeWasm();
    
    // Create React root and render app
    const root = ReactDOM.createRoot(
      document.getElementById('root') as HTMLElement
    );
    
    root.render(
      <React.StrictMode>
        <App />
      </React.StrictMode>
    );
    
    console.log('KNIRV Controller initialized successfully');
  } catch (error) {
    console.error('Failed to initialize KNIRV Controller:', error);
    
    // Show error message to user
    const root = document.getElementById('root');
    if (root) {
      root.innerHTML = \`
        <div style="
          display: flex;
          flex-direction: column;
          align-items: center;
          justify-content: center;
          height: 100vh;
          background: #111827;
          color: #ef4444;
          font-family: monospace;
          text-align: center;
          padding: 20px;
        ">
          <h1>KNIRV Controller Initialization Failed</h1>
          <p>Error: \${error.message}</p>
          <p>Please check the console for more details.</p>
          <button onclick="location.reload()" style="
            margin-top: 20px;
            padding: 10px 20px;
            background: #3b82f6;
            color: white;
            border: none;
            border-radius: 5px;
            cursor: pointer;
          ">
            Retry
          </button>
        </div>
      \`;
    }
  }
}

// Start the application
initializeApp();`;
    
    await fs.writeFile(path.join(this.rootDir, 'src', 'main.tsx'), mainTsx);
    this.log('src/main.tsx created', 'success');
  }

  async createIndexCss() {
    this.log('Creating src/index.css...');
    
    const indexCss = `@tailwind base;
@tailwind components;
@tailwind utilities;

/* Global styles for KNIRV Controller */
:root {
  --knirv-primary: #3b82f6;
  --knirv-secondary: #8b5cf6;
  --knirv-accent: #14b8a6;
  --knirv-background: #111827;
  --knirv-surface: #1f2937;
  --knirv-text: #ffffff;
  --knirv-text-secondary: #9ca3af;
}

* {
  box-sizing: border-box;
}

html, body {
  margin: 0;
  padding: 0;
  height: 100%;
  overflow: hidden;
}

body {
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', 'Roboto', 'Oxygen',
    'Ubuntu', 'Cantarell', 'Fira Sans', 'Droid Sans', 'Helvetica Neue',
    sans-serif;
  -webkit-font-smoothing: antialiased;
  -moz-osx-font-smoothing: grayscale;
  background-color: var(--knirv-background);
  color: var(--knirv-text);
}

#root {
  width: 100vw;
  height: 100vh;
  overflow: hidden;
}

/* Custom scrollbar */
::-webkit-scrollbar {
  width: 8px;
  height: 8px;
}

::-webkit-scrollbar-track {
  background: var(--knirv-surface);
}

::-webkit-scrollbar-thumb {
  background: var(--knirv-text-secondary);
  border-radius: 4px;
}

::-webkit-scrollbar-thumb:hover {
  background: var(--knirv-text);
}

/* Utility classes */
.knirv-glass {
  background: rgba(31, 41, 55, 0.8);
  backdrop-filter: blur(10px);
  border: 1px solid rgba(255, 255, 255, 0.1);
}

.knirv-gradient {
  background: linear-gradient(135deg, var(--knirv-primary) 0%, var(--knirv-secondary) 100%);
}

.knirv-shadow {
  box-shadow: 0 10px 25px rgba(0, 0, 0, 0.3);
}

/* Animation utilities */
.knirv-fade-in {
  animation: fadeIn 0.5s ease-out;
}

@keyframes fadeIn {
  from {
    opacity: 0;
    transform: translateY(20px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

.knirv-pulse {
  animation: pulse 2s infinite;
}

@keyframes pulse {
  0%, 100% {
    opacity: 1;
  }
  50% {
    opacity: 0.5;
  }
}

/* Component-specific styles */
.navigation-button {
  @apply fixed top-4 right-4 z-50 px-4 py-2 rounded-lg shadow-lg transition-all duration-200 font-medium;
}

.sliding-panel {
  @apply fixed top-0 h-full bg-gray-900 border-gray-700 shadow-2xl transform transition-transform duration-300 ease-in-out z-40;
}

.cognitive-shell {
  @apply min-h-screen bg-gray-900 text-white relative overflow-hidden;
}

/* Responsive design */
@media (max-width: 768px) {
  .navigation-button {
    @apply top-2 right-2 px-3 py-1 text-sm;
  }
  
  .sliding-panel {
    @apply w-full;
  }
}

/* Print styles */
@media print {
  body {
    background: white !important;
    color: black !important;
  }
  
  .navigation-button,
  .sliding-panel {
    display: none !important;
  }
}`;
    
    await fs.writeFile(path.join(this.rootDir, 'src', 'index.css'), indexCss);
    this.log('src/index.css created', 'success');
  }

  async createConfigs() {
    try {
      this.log('Creating unified configuration files...');
      
      await this.createViteConfig();
      await this.createTsConfig();
      await this.createBackendTsConfig();
      await this.createIndexHtml();
      await this.createMainTsx();
      await this.createIndexCss();
      
      this.log('✅ All configuration files created successfully!', 'success');
      this.log('');
      this.log('📁 Created files:');
      this.log('  - vite.config.ts (unified Vite configuration)');
      this.log('  - tsconfig.json (main TypeScript config)');
      this.log('  - tsconfig.backend.json (backend-specific config)');
      this.log('  - index.html (root HTML entry point)');
      this.log('  - src/main.tsx (application entry point)');
      this.log('  - src/index.css (global styles)');
      
    } catch (error) {
      this.log(`❌ Failed to create configurations: ${error.message}`, 'error');
      throw error;
    }
  }
}

// Run the config creation
if (import.meta.url === `file://${process.argv[1]}`) {
  const creator = new ConfigCreator();
  creator.createConfigs().catch(error => {
    console.error('Config creation failed:', error);
    process.exit(1);
  });
}

export { ConfigCreator };
