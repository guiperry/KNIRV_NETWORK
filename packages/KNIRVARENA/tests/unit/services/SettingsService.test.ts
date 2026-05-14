/**
 * SettingsService Unit Tests
 * Comprehensive test suite for settings management functionality
 */

import {
  settingsService,
  SettingsService,
  GeneralSettings,
  CognitiveSettings,
  WalletSettings,
  AnalyticsSettings,
  SecuritySettings,
  UISettings,
  AdvancedSettings
} from '../../../src/services/SettingsService';

// Mock fetch globally
global.fetch = jest.fn();

// Mock localStorage
const localStorageMock = {
  getItem: jest.fn(),
  setItem: jest.fn(),
  removeItem: jest.fn(),
  clear: jest.fn()
};
Object.defineProperty(window, 'localStorage', { value: localStorageMock });
Object.defineProperty(global, 'localStorage', { value: localStorageMock });

describe('SettingsService', () => {
  let service: SettingsService;
  const mockFetch = global.fetch as jest.MockedFunction<typeof fetch>;

  beforeEach(async () => {
    mockFetch.mockClear();
    localStorageMock.getItem.mockClear();
    localStorageMock.setItem.mockClear();
    localStorageMock.removeItem.mockClear();
    localStorageMock.clear.mockClear();

    service = new SettingsService({ baseUrl: 'http://localhost:3001' });
    await service.waitForInitialization();
  });

  afterEach(() => {
    mockFetch.mockRestore();
  });

  describe('Initialization', () => {
    it('should initialize with default settings', () => {
      expect(service).toBeDefined();
      expect(service.getSettings).toBeDefined();
      expect(service.updateSettings).toBeDefined();
    });

    it('should load default settings structure', () => {
      const settings = service.getSettings();
      
      expect(settings).toHaveProperty('general');
      expect(settings).toHaveProperty('cognitive');
      expect(settings).toHaveProperty('wallet');
      expect(settings).toHaveProperty('analytics');
      expect(settings).toHaveProperty('security');
      expect(settings).toHaveProperty('ui');
      expect(settings).toHaveProperty('advanced');
      
      // Check default values
      expect(settings.general.theme).toBe('dark');
      expect(settings.general.language).toBe('en');
      expect(settings.cognitive.defaultModel).toBe('gpt-4');
      expect(settings.wallet.defaultNetwork).toBe('knirv-mainnet');
    });
  });

  describe('Settings Management', () => {
    it('should update settings successfully', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => ({ success: true })
      } as Response);

      const updates = {
        general: {
          theme: 'light' as const,
          language: 'es'
        },
        cognitive: {
          temperature: 0.8
        }
      };

      await service.updateSettings(updates);

      const settings = service.getSettings();
      expect(settings.general.theme).toBe('light');
      expect(settings.general.language).toBe('es');
      expect(settings.cognitive.temperature).toBe(0.8);

      expect(localStorageMock.setItem).toHaveBeenCalledWith(
        'knirv-settings',
        expect.stringContaining('"theme":"light"')
      );
    });

    it('should validate settings before updating', async () => {
      const invalidSettings = {
        cognitive: {
          temperature: 5.0 // Invalid temperature > 2
        }
      };

      await expect(service.updateSettings(invalidSettings))
        .rejects.toThrow('Invalid temperature value');
    });

    it('should reset settings to defaults', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => ({ success: true })
      } as Response);

      // First update settings
      await service.updateSettings({
        general: { theme: 'light' as const }
      });

      // Then reset
      await service.resetSettings();
      
      const settings = service.getSettings();
      expect(settings.general.theme).toBe('dark'); // Back to default
    });

    it('should handle backend save failure gracefully', async () => {
      mockFetch.mockRejectedValueOnce(new Error('Network error'));

      const updates = {
        general: { theme: 'light' as const }
      };

      // Should not throw, but still update locally
      await expect(service.updateSettings(updates)).resolves.not.toThrow();
      
      const settings = service.getSettings();
      expect(settings.general.theme).toBe('light');
    });
  });

  describe('Profile Management', () => {
    it('should create settings profile successfully', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => ({ success: true })
      } as Response);

      const profile = await service.createProfile(
        'Development Profile',
        'Settings for development environment'
      );
      
      expect(profile).toHaveProperty('id');
      expect(profile).toHaveProperty('name', 'Development Profile');
      expect(profile).toHaveProperty('description', 'Settings for development environment');
      expect(profile).toHaveProperty('settings');
      expect(profile).toHaveProperty('isDefault', false);
      expect(profile).toHaveProperty('createdAt');
      expect(profile).toHaveProperty('updatedAt');
    });

    it('should load settings profile successfully', async () => {
      mockFetch.mockResolvedValue({
        ok: true,
        json: async () => ({ success: true })
      } as Response);

      const profile = await service.createProfile('Test Profile', 'Test description', {
        general: {
          theme: 'light' as const,
          language: 'en',
          timezone: 'UTC',
          autoSave: true,
          autoBackup: true,
          backupInterval: 30,
          debugMode: false,
          telemetryEnabled: true
        } as GeneralSettings,
        cognitive: {
          defaultModel: 'gpt-4',
          maxTokens: 2048,
          temperature: 0.9,
          topP: 1.0,
          frequencyPenalty: 0.0,
          presencePenalty: 0.0,
          autoLearning: true,
          skillCaching: true,
          adaptationRate: 0.1,
          contextWindow: 4096
        } as CognitiveSettings,
        wallet: {
          defaultNetwork: 'testnet',
          autoConnect: true,
          transactionTimeout: 300,
          gasLimit: 21000,
          slippageTolerance: 0.5,
          confirmationBlocks: 3,
          showTestnets: false,
          currencyDisplay: 'NRN'
        } as WalletSettings,
        analytics: {
          collectMetrics: false,
          shareAnonymousData: false,
          retentionPeriod: 30,
          metricsInterval: 30,
          alertThresholds: {
            cpuUsage: 80,
            memoryUsage: 85,
            errorRate: 5,
            responseTime: 1000
          }
        } as AnalyticsSettings,
        security: {
          requireMFA: true,
          sessionTimeout: 60,
          maxLoginAttempts: 5,
          passwordPolicy: {
            minLength: 8,
            requireUppercase: true,
            requireLowercase: true,
            requireNumbers: true,
            requireSymbols: false
          },
          encryptionLevel: 'standard',
          auditLogging: true
        } as SecuritySettings,
        ui: {
          compactMode: true,
          showTooltips: true,
          animationsEnabled: true,
          soundEnabled: true,
          notificationsEnabled: true,
          panelLayout: 'default',
          fontSize: 'medium',
          colorScheme: 'blue-purple'
        } as UISettings,
        advanced: {
          apiEndpoints: {},
          featureFlags: { betaFeatures: true },
          experimentalFeatures: [],
          customCommands: {},
          integrations: {},
          performance: {
            maxConcurrentTasks: 10,
            cacheSize: 100,
            preloadData: true,
            lazyLoading: true
          }
        } as AdvancedSettings
      });

      await service.loadProfile(profile.id);
      
      const settings = service.getSettings();
      expect(settings.general.theme).toBe('light');
      expect(settings.cognitive.temperature).toBe(0.9);
    });

    it('should delete settings profile successfully', async () => {
      mockFetch.mockResolvedValue({
        ok: true,
        json: async () => ({ success: true })
      } as Response);

      const profile = await service.createProfile('Temp Profile', 'Temporary profile');
      
      await service.deleteProfile(profile.id);
      
      const profiles = service.getProfiles();
      expect(profiles.find(p => p.id === profile.id)).toBeUndefined();
    });

    it('should not delete default profile', async () => {
      await expect(service.deleteProfile('default'))
        .rejects.toThrow('Cannot delete default profile');
    });

    it('should handle loading non-existent profile', async () => {
      await expect(service.loadProfile('non-existent-id'))
        .rejects.toThrow('Settings profile non-existent-id not found');
    });

    it('should get all profiles', () => {
      const profiles = service.getProfiles();
      expect(Array.isArray(profiles)).toBe(true);
    });

    it('should get active profile', async () => {
      mockFetch.mockResolvedValue({
        ok: true,
        json: async () => ({ success: true })
      } as Response);

      const profile = await service.createProfile('Active Profile', 'Test');
      await service.loadProfile(profile.id);
      
      const activeProfile = service.getActiveProfile();
      expect(activeProfile?.id).toBe(profile.id);
    });
  });

  describe('Import/Export', () => {
    it('should export settings successfully', () => {
      const exportedData = service.exportSettings(false);
      
      expect(typeof exportedData).toBe('string');
      expect(() => JSON.parse(exportedData)).not.toThrow();
      
      const parsed = JSON.parse(exportedData);
      expect(parsed).toHaveProperty('settings');
      expect(parsed).toHaveProperty('exportedAt');
      expect(parsed).toHaveProperty('version');
      expect(parsed.profiles).toBeUndefined(); // Not included
    });

    it('should export settings with profiles', () => {
      const exportedData = service.exportSettings(true);
      
      const parsed = JSON.parse(exportedData);
      expect(parsed).toHaveProperty('profiles');
    });

    it('should import settings successfully', async () => {
      mockFetch.mockResolvedValue({
        ok: true,
        json: async () => ({ success: true })
      } as Response);

      const importData = {
        settings: {
          general: { theme: 'light' as const, language: 'fr' },
          cognitive: { temperature: 0.5 },
          wallet: { defaultNetwork: 'testnet' },
          analytics: { collectMetrics: false },
          security: { requireMFA: true },
          ui: { compactMode: true },
          advanced: { featureFlags: {} }
        },
        version: '1.0',
        exportedAt: new Date().toISOString()
      };

      await service.importSettings(JSON.stringify(importData), true);
      
      const settings = service.getSettings();
      expect(settings.general.theme).toBe('light');
      expect(settings.general.language).toBe('fr');
      expect(settings.cognitive.temperature).toBe(0.5);
    });

    it('should merge settings when importing without overwrite', async () => {
      mockFetch.mockResolvedValue({
        ok: true,
        json: async () => ({ success: true })
      } as Response);

      // First set some settings
      await service.updateSettings({
        general: { theme: 'dark' as const, debugMode: true }
      });

      const importData = {
        settings: {
          general: { theme: 'light' as const }, // This should change
          cognitive: { temperature: 0.5 }       // This should be added
        },
        version: '1.0'
      };

      await service.importSettings(JSON.stringify(importData), false);
      
      const settings = service.getSettings();
      expect(settings.general.theme).toBe('light');     // Changed
      expect(settings.general.debugMode).toBe(true);    // Preserved
      expect(settings.cognitive.temperature).toBe(0.5); // Added
    });

    it('should handle invalid import data', async () => {
      await expect(service.importSettings('invalid json', false))
        .rejects.toThrow('Settings import failed');
      
      await expect(service.importSettings('{"invalid": "data"}', false))
        .rejects.toThrow('Settings import failed');
    });
  });

  describe('Individual Setting Management', () => {
    it('should get setting by path', async () => {
      await service.updateSettings({
        general: { theme: 'light' as const }
      });

      const theme = service.getSetting<string>('general.theme');
      expect(theme).toBe('light');
      
      const temperature = service.getSetting<number>('cognitive.temperature');
      expect(typeof temperature).toBe('number');
    });

    it('should return undefined for non-existent path', () => {
      const nonExistent = service.getSetting('non.existent.path');
      expect(nonExistent).toBeUndefined();
    });

    it('should set setting by path', async () => {
      mockFetch.mockResolvedValue({
        ok: true,
        json: async () => ({ success: true })
      } as Response);

      await service.setSetting('general.theme', 'light');
      
      const settings = service.getSettings();
      expect(settings.general.theme).toBe('light');
    });

    it('should set nested setting by path', async () => {
      mockFetch.mockResolvedValue({
        ok: true,
        json: async () => ({ success: true })
      } as Response);

      await service.setSetting('advanced.featureFlags.betaFeatures', true);
      
      const settings = service.getSettings();
      expect(settings.advanced.featureFlags.betaFeatures).toBe(true);
    });
  });

  describe('Change Listeners', () => {
    it('should add and notify change listeners', async () => {
      mockFetch.mockResolvedValue({
        ok: true,
        json: async () => ({ success: true })
      } as Response);

      const listener = jest.fn();
      service.addChangeListener(listener);
      
      await service.updateSettings({
        general: { theme: 'light' as const }
      });
      
      expect(listener).toHaveBeenCalledWith(
        expect.objectContaining({
          general: expect.objectContaining({ theme: 'light' })
        })
      );
    });

    it('should remove change listeners', async () => {
      mockFetch.mockResolvedValue({
        ok: true,
        json: async () => ({ success: true })
      } as Response);

      const listener = jest.fn();
      service.addChangeListener(listener);
      service.removeChangeListener(listener);
      
      await service.updateSettings({
        general: { theme: 'light' as const }
      });
      
      expect(listener).not.toHaveBeenCalled();
    });

    it('should handle listener errors gracefully', async () => {
      mockFetch.mockResolvedValue({
        ok: true,
        json: async () => ({ success: true })
      } as Response);

      const errorListener = jest.fn().mockImplementation(() => {
        throw new Error('Listener error');
      });
      
      service.addChangeListener(errorListener);
      
      // Should not throw
      await expect(service.updateSettings({
        general: { theme: 'light' as const }
      })).resolves.not.toThrow();
    });
  });

  describe('LocalStorage Integration', () => {
    it('should save to localStorage on update', async () => {
      mockFetch.mockResolvedValue({
        ok: true,
        json: async () => ({ success: true })
      } as Response);

      await service.updateSettings({
        general: { theme: 'light' as const }
      });
      
      expect(localStorageMock.setItem).toHaveBeenCalledWith(
        'knirv-settings',
        expect.stringContaining('"theme":"light"')
      );
    });

    it('should load from localStorage on initialization', async () => {
      const storedSettings = {
        settings: {
          general: { theme: 'light' as const }
        },
        activeProfile: 'test-profile',
        lastUpdated: new Date().toISOString()
      };

      localStorageMock.getItem.mockReturnValueOnce(JSON.stringify(storedSettings));

      const newService = new SettingsService();
      await newService.waitForInitialization();
      const settings = newService.getSettings();

      expect(settings.general.theme).toBe('light');
    });

    it('should handle localStorage errors gracefully', async () => {
      localStorageMock.setItem.mockImplementation(() => {
        throw new Error('Storage error');
      });

      mockFetch.mockResolvedValue({
        ok: true,
        json: async () => ({ success: true })
      } as Response);

      // Should not throw
      await expect(service.updateSettings({
        general: { theme: 'light' as const }
      })).resolves.not.toThrow();
    });
  });

  describe('Singleton Instance', () => {
    it('should provide a singleton instance', () => {
      expect(settingsService).toBeDefined();
      expect(settingsService).toBeInstanceOf(SettingsService);
    });
  });

  describe('Settings Validation', () => {
    it('should validate session timeout range', async () => {
      await expect(service.updateSettings({
        security: { sessionTimeout: 2 } // Too low
      })).rejects.toThrow('Invalid session timeout value');

      await expect(service.updateSettings({
        security: { sessionTimeout: 2000 } // Too high
      })).rejects.toThrow('Invalid session timeout value');
    });

    it('should validate temperature range', async () => {
      await expect(service.updateSettings({
        cognitive: { temperature: -1 } // Too low
      })).rejects.toThrow('Invalid temperature value');

      await expect(service.updateSettings({
        cognitive: { temperature: 3 } // Too high
      })).rejects.toThrow('Invalid temperature value');
    });

    it('should validate settings structure', async () => {
      await expect(service.updateSettings({
        general: { invalidProperty: 'invalid' } as unknown as Partial<GeneralSettings>
      })).rejects.toThrow('Invalid settings structure');
    });
  });
});
