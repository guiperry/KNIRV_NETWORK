import type { CapacitorConfig } from '@capacitor/cli';

const config: CapacitorConfig = {
  appId: 'com.knirv.controller',
  appName: 'KNIRV Controller',
  webDir: 'dist',
  server: {
    androidScheme: 'https'
  }
};

export default config;