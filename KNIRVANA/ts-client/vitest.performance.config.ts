/// <reference types="vitest" />
import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import { resolve } from 'path'

export default defineConfig({
  plugins: [react()],
  test: {
    globals: true,
    environment: 'jsdom',
    setupFiles: ['./src/test/setup.ts', './src/test/performance-setup.ts'],
    include: ['src/test/performance/**/*.{test,spec}.{js,mjs,cjs,ts,mts,cts,jsx,tsx}'],
    exclude: [
      'node_modules',
      'dist',
      '.idea',
      '.git',
      '.cache'
    ],
    testTimeout: 60000, // Longer timeout for performance tests
    hookTimeout: 30000,
    teardownTimeout: 15000,
    isolate: true,
    pool: 'threads',
    poolOptions: {
      threads: {
        singleThread: true, // Performance tests need consistent environment
        minThreads: 1,
        maxThreads: 1
      }
    },
    sequence: {
      concurrent: false // Run performance tests sequentially
    },
    benchmark: {
      include: ['src/test/performance/**/*.bench.{js,mjs,cjs,ts,mts,cts,jsx,tsx}'],
      exclude: ['node_modules', 'dist'],
      reporters: ['default', 'json']
    }
  },
  resolve: {
    alias: {
      '@': resolve(__dirname, './src'),
      '@/components': resolve(__dirname, './src/components'),
      '@/lib': resolve(__dirname, './src/lib'),
      '@/types': resolve(__dirname, './src/types'),
      '@/hooks': resolve(__dirname, './src/hooks'),
      '@/pages': resolve(__dirname, './src/pages'),
      '@/test': resolve(__dirname, './src/test')
    }
  },
  define: {
    'import.meta.vitest': undefined
  }
})
