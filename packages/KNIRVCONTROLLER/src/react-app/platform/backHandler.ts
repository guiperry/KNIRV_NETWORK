import { App } from '@capacitor/app';
import { Capacitor } from '@capacitor/core';

export function registerBackHandler(onBack: () => boolean) {
  if (!Capacitor.isNativePlatform()) return () => {};
  const listener = App.addListener('backButton', () => {
    if (!onBack()) App.exitApp();
  });
  return () => { listener.then(l => l.remove()); };
}
