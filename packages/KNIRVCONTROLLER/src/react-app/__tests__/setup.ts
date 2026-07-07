import { beforeEach } from 'vitest';
import '@testing-library/jest-dom';
import { resetVaultSession } from '@/react-app/hooks/vaultSession';

// Mock localStorage
const localStorageMock = (() => {
  let store: Record<string, string> = {};
  return {
    getItem: (key: string) => store[key] ?? null,
    setItem: (key: string, value: string) => { store[key] = value; },
    removeItem: (key: string) => { delete store[key]; },
    clear: () => { store = {}; },
  };
})();
Object.defineProperty(window, 'localStorage', { value: localStorageMock });

// Mock clipboard API
Object.defineProperty(navigator, 'clipboard', {
  value: {
    writeText: () => Promise.resolve(),
    readText: () => Promise.resolve(''),
  },
  configurable: true,
});

// jsdom does not implement scrollIntoView
Object.defineProperty(HTMLElement.prototype, 'scrollIntoView', {
  value: () => {},
  configurable: true,
});

beforeEach(() => {
  resetVaultSession();
  localStorage.clear();
});
