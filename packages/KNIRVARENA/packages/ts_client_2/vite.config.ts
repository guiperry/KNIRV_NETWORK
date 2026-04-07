import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';
import { resolve } from 'path';
import topLevelAwait from 'vite-plugin-top-level-await';
import wasm from 'vite-plugin-wasm';
import { visualizer } from 'rollup-plugin-visualizer';
import glsl from 'vite-plugin-glsl';

export default defineConfig({
  plugins: [
    react(),
    wasm(),
    glsl(),
    topLevelAwait({
      promiseExportName: '__tla',
      promiseImportName: i => `__tla_${i}`
    }),
    ...(process.env.ANALYZE === 'true' ? [visualizer()] : [])
  ],
  
  // Use relative base so builds work from file:// and subpaths
  base: './',
  
  // Root source directory
  root: '.',
  
  // Build configuration
  build: {
    outDir: 'dist',
    sourcemap: true,
    rollupOptions: {
      input: {
        main: resolve(__dirname, 'index.html')
      },
      output: {
        manualChunks: {
          // Vendor chunks for better caching
          vendor: ['react', 'react-dom', 'react-router-dom'],
          ui: ['lucide-react', '@react-three/fiber', '@react-three/drei', 'three'],
          blockchain: ['@cosmjs/stargate', '@gnolang/tm2-js-client', '@burnt-labs/abstraxion'],
          database: ['lokijs'],
          utils: ['uuid', 'bech32', 'qrcode', 'qr-scanner']
        }
      }
    },
    // Copy AssemblyScript WASM files
    copyPublicDir: true,
    // Bundle size optimizations
    minify: 'esbuild',
    cssCodeSplit: true,
    // Chunk size warning limit
    chunkSizeWarningLimit: 1000
  },
  
  // Development server
  server: {
    host: '0.0.0.0',
    port: 3000,
    open: false,
    headers: {
      'Content-Security-Policy': "default-src 'self'; script-src 'self' 'unsafe-inline' 'unsafe-eval'; style-src 'self' 'unsafe-inline'; img-src 'self' data: blob: https:; font-src 'self' data:; connect-src 'self' ws: wss: wasm: blob: http://localhost:3001 https://wallet.knirv.com https://generativelanguage.googleapis.com; frame-src 'none';"
    },
    proxy: {
      '/api': {
        target: 'http://localhost:3001',
        changeOrigin: true,
        secure: false
      },
      '/ws': {
        target: 'ws://localhost:3001',
        ws: true
      },
      '/wallet': {
        target: 'https://wallet.knirv.com',
        changeOrigin: true,
        secure: true,
        rewrite: (path) => path.replace(/^\/wallet/, '')
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
      '@hooks': resolve(__dirname, 'src/hooks'),
      '@services': resolve(__dirname, 'src/services'),
      '@types': resolve(__dirname, 'src/types'),
      '@core': resolve(__dirname, 'src/core'),
      '@manager': resolve(__dirname, 'src/manager'),
      '@shared': resolve(__dirname, 'src/shared'),
      '@sensory-shell': resolve(__dirname, 'src/sensory-shell'),
      '@wasm': resolve(__dirname, 'src/wasm-pkg'),
      '@game': resolve(__dirname, 'src/components/game'),
      'crypto': resolve(__dirname, 'src/shims/node-crypto.ts'),
      'jsonwebtoken': resolve(__dirname, 'src/shims/jsonwebtoken.ts')
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
      'three',
      '@react-three/fiber',
      '@react-three/drei',
      '@react-three/postprocessing',
      'zustand',
      'howler',
      'matter-js',
      'gsap',
      'framer-motion'
    ],
    exclude: ['@napi-rs/wasm-runtime']
  },
  
  // WASM support
  worker: {
    format: 'es',
    plugins: () => [wasm(), topLevelAwait()]
  },
  
  // Asset handling for 3D models and textures
  assetsInclude: ['**/*.gltf', '**/*.glb', '**/*.hdr', '**/*.mp3', '**/*.wav', '**/*.ogg'],
  
  // Environment variables
  define: {
    __DEV__: JSON.stringify(process.env.NODE_ENV === 'development'),
    __PROD__: JSON.stringify(process.env.NODE_ENV === 'production'),
    global: 'globalThis',
    // Provide a browser-safe process.env so Node-style packages don't crash.
    // All knirvbase-ts references use || fallbacks, so an empty object is fine.
    'process.env': JSON.stringify({ NODE_ENV: process.env.NODE_ENV || 'production' })
  }
});
