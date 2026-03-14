import { create } from 'zustand';
import { persist } from 'zustand/middleware';

type ThemeMode = 'dark' | 'light' | 'light-blue';

interface ThemeStore {
  themeMode: ThemeMode;
  toggleTheme: () => void;
  setTheme: (mode: ThemeMode) => void;
}

export const useThemeStore = create<ThemeStore>()(
  persist(
    (set) => ({
      themeMode: 'dark',
      toggleTheme: () => set((state) => {
        switch (state.themeMode) {
          case 'dark':
            return { themeMode: 'light' };
          case 'light':
            return { themeMode: 'light-blue' };
          case 'light-blue':
          default:
            return { themeMode: 'dark' };
        }
      }),
      setTheme: (mode) => set({ themeMode: mode }),
    }),
    {
      name: 'knirv-theme',
    }
  )
);