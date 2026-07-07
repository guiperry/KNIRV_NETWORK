import { CapacitorConfig } from '@capacitor/cli';

const config: CapacitorConfig = {
  appId: 'com.knirv.controller',
  appName: 'KNIRV Controller',
  webDir: 'dist',
  server: { androidScheme: 'https' },
  plugins: {
    SplashScreen: { launchAutoHide: false, backgroundColor: '#08111f', showSpinner: false },
    StatusBar: { style: 'Dark', backgroundColor: '#08111f', overlaysWebView: false },
  },
};

export default config;
