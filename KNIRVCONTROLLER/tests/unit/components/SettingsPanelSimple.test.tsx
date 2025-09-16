import * as React from 'react';
import { render } from '@testing-library/react';
import '@testing-library/jest-dom';

// Mock lucide-react icons
jest.mock('lucide-react', () => ({
  Settings: () => <div>Settings Icon</div>,
  Save: () => <div>Save Icon</div>,
  RefreshCw: () => <div>Refresh Icon</div>,
  Download: () => <div>Download Icon</div>,
  Upload: () => <div>Upload Icon</div>,
  User: () => <div>User Icon</div>,
  Shield: () => <div>Shield Icon</div>,
  Palette: () => <div>Palette Icon</div>,
  Zap: () => <div>Zap Icon</div>,
  Brain: () => <div>Brain Icon</div>,
  Wallet: () => <div>Wallet Icon</div>,
  BarChart3: () => <div>BarChart3 Icon</div>,
}));

// Mock SettingsService
jest.mock('../../../src/services/SettingsService', () => ({
  settingsService: {
    getSettings: jest.fn().mockReturnValue({
      general: {
        theme: 'dark',
        language: 'en',
        autoSave: true,
        backupInterval: 30,
        enableNotifications: true,
        enableAnalytics: false,
        enableVoiceCommands: true,
        enableGestures: false
      },
      cognitive: {
        creativityLevel: 0.7,
        focusLevel: 0.8,
        memoryRetention: 0.9,
        learningRate: 0.6,
        adaptabilityLevel: 0.5,
        enableDeepThinking: true,
        enableContextualMemory: true,
        enablePredictiveText: false
      },
      wallet: {},
      analytics: {},
      security: {},
      ui: {},
      advanced: {}
    }),
    updateSettings: jest.fn().mockResolvedValue(undefined),
    resetSettings: jest.fn().mockResolvedValue(undefined),
    getProfiles: jest.fn().mockReturnValue([]),
    saveProfile: jest.fn().mockResolvedValue(undefined),
    loadProfile: jest.fn().mockResolvedValue(undefined),
    deleteProfile: jest.fn().mockResolvedValue(undefined),
    getActiveProfile: jest.fn().mockReturnValue(null),
    exportSettings: jest.fn().mockReturnValue('{}'),
    importSettings: jest.fn().mockResolvedValue(undefined)
  }
}));

import SettingsPanel from '../../../src/components/SettingsPanel';

describe('SettingsPanel Simple Test', () => {
  it('should render without crashing', () => {
    render(<SettingsPanel isOpen={true} onClose={jest.fn()} />);
    // Just check that it doesn't crash
    expect(document.body).toBeInTheDocument();
  });
});
