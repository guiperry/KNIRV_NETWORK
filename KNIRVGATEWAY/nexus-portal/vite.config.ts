import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// Custom plugin to rename dashboard.html to index.html
const renameIndexPlugin = () => {
  return {
    name: 'rename-index',
    generateBundle(options, bundle) {
      if (bundle['dashboard.html']) {
        bundle['index.html'] = bundle['dashboard.html'];
        delete bundle['dashboard.html'];
      }
    }
  }
}

// https://vitejs.dev/config/
export default defineConfig({
  plugins: [react(), renameIndexPlugin()],
  base: '/nexus-portal/',
  build: {
    outDir: 'dist',
    assetsDir: 'assets',
    sourcemap: true,
    emptyOutDir: true,
    rollupOptions: {
      input: 'dashboard.html',
      output: {
        entryFileNames: 'assets/[name]-[hash].js',
        chunkFileNames: 'assets/[name]-[hash].js',
        assetFileNames: 'assets/[name]-[hash].[ext]'
      }
    }
  },
  server: {
    port: 3000,
    proxy: {
      '/gateway': {
        target: 'http://localhost:8888',
        changeOrigin: true,
        secure: false
      }
    }
  },
  preview: {
    port: 3000
  }
})
