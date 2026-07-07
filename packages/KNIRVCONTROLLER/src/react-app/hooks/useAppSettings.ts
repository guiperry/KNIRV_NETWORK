import { useEffect, useState } from 'react';
import { readBooleanStorage, writeStorageItem } from '@/react-app/platform/browserStorage';

export const APP_SETTINGS_KEYS = {
  autoOpenVault: 'knirv.settings.autoOpenVault',
  badgeNotifications: 'knirv.settings.badgeNotifications',
  voiceHints: 'knirv.settings.voiceHints',
} as const;

function usePersistedBooleanSetting(storageKey: string, defaultValue: boolean) {
  const [value, setValue] = useState(() => readBooleanStorage(storageKey, defaultValue));

  useEffect(() => {
    writeStorageItem(storageKey, String(value));
  }, [storageKey, value]);

  return [value, setValue] as const;
}

export function useAppSettings() {
  const [autoOpenVault, setAutoOpenVault] = usePersistedBooleanSetting(APP_SETTINGS_KEYS.autoOpenVault, true);
  const [badgeNotifications, setBadgeNotifications] = usePersistedBooleanSetting(APP_SETTINGS_KEYS.badgeNotifications, true);
  const [voiceHints, setVoiceHints] = usePersistedBooleanSetting(APP_SETTINGS_KEYS.voiceHints, false);

  return {
    autoOpenVault,
    setAutoOpenVault,
    badgeNotifications,
    setBadgeNotifications,
    voiceHints,
    setVoiceHints,
  };
}
