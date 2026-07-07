import { Capacitor } from '@capacitor/core';
import { Preferences } from '@capacitor/preferences';

export const nativeStorage = {
  async get(key: string): Promise<string | null> {
    if (Capacitor.isNativePlatform()) {
      const { value } = await Preferences.get({ key });
      return value ?? null;
    }
    return localStorage.getItem(key);
  },

  async set(key: string, value: string): Promise<void> {
    if (Capacitor.isNativePlatform()) {
      await Preferences.set({ key, value });
    } else {
      localStorage.setItem(key, value);
    }
  },

  async remove(key: string): Promise<void> {
    if (Capacitor.isNativePlatform()) {
      await Preferences.remove({ key });
    } else {
      localStorage.removeItem(key);
    }
  },
};
