import fs from "node:fs";
import path from "path";
import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import { nodePolyfills } from 'vite-plugin-node-polyfills';

function loadHttpsOptions() {
  if (process.env.KNIRV_HTTPS !== '1') { return undefined; }
  const keyPath = process.env.KNIRV_HTTPS_KEY;
  const certPath = process.env.KNIRV_HTTPS_CERT;
  if (!keyPath || !certPath) {
    throw new Error('KNIRV_HTTPS requires KNIRV_HTTPS_KEY and KNIRV_HTTPS_CERT');
  }
  return { key: fs.readFileSync(keyPath), cert: fs.readFileSync(certPath) };
}

const https = loadHttpsOptions();

export default defineConfig({
  plugins: [
    react(),
    nodePolyfills({
      include: ['buffer', 'process', 'util', 'stream', 'events'],
      globals: { Buffer: true, global: true, process: true },
    }),
  ],
  optimizeDeps: { esbuildOptions: { target: 'esnext' } },
  server: { allowedHosts: true, https, host: '0.0.0.0' },
  preview: { https, host: '0.0.0.0' },
  base: './',
  build: {
    target: 'esnext',
    chunkSizeWarningLimit: 5000,
  },
  resolve: {
    alias: {
      "@": path.resolve(__dirname, "./src"),
      "knirvwallet-module": path.resolve(__dirname, "./core/packages/knirvwallet-module/src"),
    },
  },
  define: { 'process.env': {} },
});
