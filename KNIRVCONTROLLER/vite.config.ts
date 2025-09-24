import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';
import { resolve } from 'path';
import topLevelAwait from 'vite-plugin-top-level-await';
import wasm from 'vite-plugin-wasm';
import { visualizer } from 'rollup-plugin-visualizer';

export default defineConfig({
  plugins: [
    react(),
    wasm(),
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
          database: ['rxdb', 'lokijs'],
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
      '@hooks': resolve(__dirname, 'src/hooks'),
      '@services': resolve(__dirname, 'src/services'),
      '@types': resolve(__dirname, 'src/types'),
      '@core': resolve(__dirname, 'src/core'),
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
    plugins: () => [wasm(), topLevelAwait()]
  },
  
  // Environment variables
  define: {
    __DEV__: JSON.stringify(process.env.NODE_ENV === 'development'),
    __PROD__: JSON.stringify(process.env.NODE_ENV === 'production')
  }
});
