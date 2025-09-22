import path from "path";
import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
// import { cloudflare } from "@cloudflare/vite-plugin";
// import { mochaPlugins } from "@getmocha/vite-plugins";

export default defineConfig({
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  plugins: [react()], // Temporarily disable other plugins
  server: {
    host: '0.0.0.0', // Allow access from any IP address
    port: 5173,
    strictPort: true,
    allowedHosts: true,
    cors: true, // Enable CORS for mobile access
  },
  build: {
    chunkSizeWarningLimit: 5000,
  },
  resolve: {
    alias: {
      "@": path.resolve(__dirname, "./src"),
    },
  },
});
