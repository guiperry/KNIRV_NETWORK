import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';
import fs from 'fs';
import path from 'path';

// Function to read port configuration
function readPortConfig() {
  const configuredApiUrl = process.env.VITE_API_BASE_URL;
  if (configuredApiUrl) {
    return configuredApiUrl;
  }

  const configPath = path.join(__dirname, 'ports.config');
  let apiPort = '8081'; // default

  try {
    const configContent = fs.readFileSync(configPath, 'utf-8');
    const lines = configContent.split('\n');

    for (const line of lines) {
      const trimmedLine = line.trim();
      if (trimmedLine.startsWith('API_PORT=')) {
        apiPort = trimmedLine.split('=')[1].trim();
        break;
      }
    }
  } catch {
    console.warn('Could not read ports.config, using default API port 8081');
  }

  return apiPort;
}

// https://vitejs.dev/config/
export default defineConfig(() => {
  const apiPort = readPortConfig();

  return {
    plugins: [react()],
    server: {
      proxy: {
        '/api': {
          target: apiPort.startsWith('http') ? apiPort : `http://localhost:${apiPort}`,
          changeOrigin: true,
          secure: false,
          ws: true,
        }
      }
    },
    optimizeDeps: {
      exclude: ['lucide-react'],
    },
    build: {
      outDir: 'dist',
      emptyOutDir: true,
      assetsDir: 'assets',
      // noVNC 1.7 ships ESM with top-level await, which the legacy es2020
      // target rejects. This is an Electron/Chromium desktop client, so a
      // modern target is safe.
      target: 'esnext',
    },
    // Use relative paths for Electron compatibility
    base: './',
  };
});
