import { defineConfig } from 'vitest/config';
import path from 'path';

export default defineConfig({
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
      'knirvwallet-module': path.resolve(__dirname, './core/packages/knirvwallet-module/src'),
    },
  },
  test: {
    include: ['src/react-app/**/*.test.tsx', 'src/react-app/**/*.spec.tsx'],
    environment: 'jsdom',
    globals: true,
    setupFiles: ['./src/react-app/__tests__/setup.ts'],
  },
});
