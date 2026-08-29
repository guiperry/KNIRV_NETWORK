import { defineConfig } from "vitest/config";

export default defineConfig({
  test: {
    environment: "node",
  },
  resolve: {
    alias: {
      "cloudflare:workers": "./src/__tests__/mock-cloudflare-workers.js",
    },
  },
});
