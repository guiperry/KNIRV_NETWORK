/**
 * Settings Service
 * Handles configuration persistence, user preferences, and system settings
 */

export interface AppSettings {
  general: GeneralSettings;
  cognitive: CognitiveSettings;
  wallet: WalletSettings;
  analytics: AnalyticsSettings;
  security: SecuritySettings;
  ui: UISettings;
  advanced: AdvancedSettings;
}

export interface PartialAppSettings {
  general?: Partial<GeneralSettings>;
  cognitive?: Partial<CognitiveSettings>;
  wallet?: Partial<WalletSettings>;
  analytics?: Partial<AnalyticsSettings>;
  security?: Partial<SecuritySettings>;
  ui?: Partial<UISettings>;
  advanced?: Partial<AdvancedSettings>;
}

export interface GeneralSettings {
  theme: 'dark' | 'light' | 'auto';
  language: string;
  timezone: string;
  autoSave: boolean;
  autoBackup: boolean;
  backupInterval: number; // in minutes
  debugMode: boolean;
  telemetryEnabled: boolean;
}

export interface CognitiveSettings {
  defaultModel: string;
  maxTokens: number;
  temperature: number;
  topP: number;
  frequencyPenalty: number;
  presencePenalty: number;
  autoLearning: boolean;
  skillCaching: boolean;
  adaptationRate: number;
  contextWindow: number;
}

export interface WalletSettings {
  defaultNetwork: string;
  autoConnect: boolean;
  transactionTimeout: number;
  gasLimit: number;
  slippageTolerance: number;
  confirmationBlocks: number;
  showTestnets: boolean;
  currencyDisplay: 'USD' | 'EUR' | 'BTC' | 'ETH' | 'NRN';
}

export interface AnalyticsSettings {
  collectMetrics: boolean;
  shareAnonymousData: boolean;
  retentionPeriod: number; // in days
  metricsInterval: number; // in seconds
  alertThresholds: {
    cpuUsage: number;
    memoryUsage: number;
    errorRate: number;
    responseTime: number;
  };
}

export interface SecuritySettings {
  requireMFA: boolean;
  sessionTimeout: number; // in minutes
  maxLoginAttempts: number;
  passwordPolicy: {
    minLength: number;
    requireUppercase: boolean;
    requireLowercase: boolean;
    requireNumbers: boolean;
    requireSymbols: boolean;
  };
  encryptionLevel: 'basic' | 'standard' | 'high' | 'quantum';
  auditLogging: boolean;
}

export interface UISettings {
  compactMode: boolean;
  showTooltips: boolean;
  animationsEnabled: boolean;
  soundEnabled: boolean;
  notificationsEnabled: boolean;
  panelLayout: 'default' | 'compact' | 'expanded';
  fontSize: 'small' | 'medium' | 'large';
  colorScheme: string;
}

export interface AdvancedSettings {
  apiEndpoints: Record<string, string>;
  featureFlags: Record<string, boolean>;
  experimentalFeatures: string[];
  customCommands: Record<string, string>;
  integrations: Record<string, unknown>;
  performance: {
    maxConcurrentTasks: number;
    cacheSize: number;
    preloadData: boolean;
    lazyLoading: boolean;
  };
}

export interface SettingsProfile {
  id: string;
  name: string;
  description: string;
  settings: AppSettings;
  isDefault: boolean;
  createdAt: Date;
  updatedAt: Date;
}

export interface SettingsConfig {
  baseUrl?: string;
  enableNetworking?: boolean;
  enableSync?: boolean;
}

export class SettingsService {
  private currentSettings: AppSettings;
  private profiles: Map<string, SettingsProfile> = new Map();
  private activeProfileId: string = 'default';
  private baseUrl: string;
  private isInitialized: boolean = false;
  private initializationPromise: Promise<void> | null = null;
  private changeListeners: Array<(settings: AppSettings) => void> = [];
  private config: SettingsConfig;

  constructor(config: SettingsConfig = {}) {
    this.config = {
      baseUrl: 'http://localhost:3001',
      enableNetworking: process.env.NODE_ENV !== 'test',
      enableSync: process.env.NODE_ENV !== 'test',
      ...config
    };
    this.baseUrl = this.config.baseUrl!;
    this.currentSettings = this.getDefaultSettings();
    this.initializationPromise = this.initializeService();
  }

  /**
   * Wait for service initialization to complete
   */
  async waitForInitialization(): Promise<void> {
    if (this.initializationPromise) {
      await this.initializationPromise;
    }
  }

  private async initializeService(): Promise<void> {
    try {
      // Load settings from localStorage first
      this.loadFromLocalStorage();

      // Then try to sync with backend only if networking is enabled
      if (this.config.enableSync && this.config.enableNetworking) {
        await this.syncWithBackend();
      }

      this.isInitialized = true;
      console.log('Settings Service initialized');
    } catch (error) {
      console.error('Failed to initialize Settings Service:', error);
      // Continue with default settings if initialization fails
      this.isInitialized = true;
    } finally {
      this.initializationPromise = null;
    }
  }

  /**
   * Get current settings
   */
  getSettings(): AppSettings {
    return { ...this.currentSettings };
  }

  /**
   * Update settings
   */
  async updateSettings(updates: PartialAppSettings): Promise<void> {
    const newSettings = this.mergeSettings(this.currentSettings, updates);
    
    try {
      // Validate settings
      this.validateSettings(newSettings);
      
      // Update current settings
      this.currentSettings = newSettings;
      
      // Save to localStorage
      this.saveToLocalStorage();
      
      // Save to backend
      await this.saveToBackend();
      
      // Notify listeners
      this.notifyListeners();
      
      console.log('Settings updated successfully');
    } catch (error) {
      console.error('Failed to update settings:', error);
      throw error;
    }
  }

  /**
   * Reset settings to defaults
   */
  async resetSettings(): Promise<void> {
    this.currentSettings = this.getDefaultSettings();
    this.saveToLocalStorage();
    await this.saveToBackend();
    this.notifyListeners();
    console.log('Settings reset to defaults');
  }

  /**
   * Create settings profile
   */
  async createProfile(name: string, description: string, settings?: AppSettings): Promise<SettingsProfile> {
    const profile: SettingsProfile = {
      id: this.generateProfileId(),
      name,
      description,
      settings: settings || { ...this.currentSettings },
      isDefault: false,
      createdAt: new Date(),
      updatedAt: new Date()
    };

    this.profiles.set(profile.id, profile);
    await this.saveProfilesToBackend();
    
    console.log(`Settings profile created: ${name}`);
    return profile;
  }

  /**
   * Load settings profile
   */
  async loadProfile(profileId: string): Promise<void> {
    const profile = this.profiles.get(profileId);
    if (!profile) {
      throw new Error(`Settings profile ${profileId} not found`);
    }

    this.currentSettings = { ...profile.settings };
    this.activeProfileId = profileId;
    
    this.saveToLocalStorage();
    this.notifyListeners();
    
    console.log(`Settings profile loaded: ${profile.name}`);
  }

  /**
   * Delete settings profile
   */
  async deleteProfile(profileId: string): Promise<void> {
    if (profileId === 'default') {
      throw new Error('Cannot delete default profile');
    }

    const profile = this.profiles.get(profileId);
    if (!profile) {
      throw new Error(`Settings profile ${profileId} not found`);
    }

    this.profiles.delete(profileId);
    await this.saveProfilesToBackend();
    
    // Switch to default if current profile was deleted
    if (this.activeProfileId === profileId) {
      this.activeProfileId = 'default';
      this.currentSettings = this.getDefaultSettings();
      this.saveToLocalStorage();
      this.notifyListeners();
    }
    
    console.log(`Settings profile deleted: ${profile.name}`);
  }

  /**
   * Get all profiles
   */
  getProfiles(): SettingsProfile[] {
    return Array.from(this.profiles.values());
  }

  /**
   * Get active profile
   */
  getActiveProfile(): SettingsProfile | undefined {
    return this.profiles.get(this.activeProfileId);
  }

  /**
   * Export settings
   */
  exportSettings(includeProfiles: boolean = false): string {
    const exportData = {
      settings: this.currentSettings,
      activeProfile: this.activeProfileId,
      profiles: includeProfiles ? Object.fromEntries(this.profiles) : undefined,
      exportedAt: new Date().toISOString(),
      version: '1.0'
    };

    return JSON.stringify(exportData, null, 2);
  }

  /**
   * Import settings
   */
  async importSettings(settingsJson: string, overwrite: boolean = false): Promise<void> {
    try {
      const importData = JSON.parse(settingsJson);

      if (!importData.settings) {
        throw new Error('Invalid settings format');
      }

      if (overwrite) {
        // For overwrite, validate the complete settings structure
        this.validateSettings(importData.settings);
        this.currentSettings = importData.settings;

        if (importData.profiles) {
          this.profiles.clear();
          for (const [id, profile] of Object.entries(importData.profiles)) {
            this.profiles.set(id, profile as SettingsProfile);
          }
        }

        if (importData.activeProfile) {
          this.activeProfileId = importData.activeProfile;
        }
      } else {
        // For merge, create the merged settings first, then validate
        const mergedSettings = this.mergeSettings(this.currentSettings, importData.settings);
        this.validateSettings(mergedSettings);
        this.currentSettings = mergedSettings;
      }

      this.saveToLocalStorage();
      await this.saveToBackend();
      this.notifyListeners();

      console.log('Settings imported successfully');
    } catch (error) {
      console.error('Failed to import settings:', error);
      throw new Error(`Settings import failed: ${error instanceof Error ? error.message : 'Unknown error'}`);
    }
  }

  /**
   * Add change listener
   */
  addChangeListener(listener: (settings: AppSettings) => void): void {
    this.changeListeners.push(listener);
  }

  /**
   * Remove change listener
   */
  removeChangeListener(listener: (settings: AppSettings) => void): void {
    const index = this.changeListeners.indexOf(listener);
    if (index > -1) {
      this.changeListeners.splice(index, 1);
    }
  }

  /**
   * Get setting by path
   */
  getSetting<T>(path: string): T | undefined {
    const keys = path.split('.');
    let current: unknown = this.currentSettings;
    
    for (const key of keys) {
      if (current && typeof current === 'object' && key in current) {
        current = (current as Record<string, unknown>)[key];
      } else {
        return undefined;
      }
    }
    
    return current as T;
  }

  /**
   * Set setting by path
   */
  async setSetting(path: string, value: unknown): Promise<void> {
    const keys = path.split('.');
    const updates: Record<string, unknown> = {};
    let current = updates;
    
    for (let i = 0; i < keys.length - 1; i++) {
      current[keys[i]] = {};
      current = current[keys[i]] as Record<string, unknown>;
    }
    
    current[keys[keys.length - 1]] = value;
    
    await this.updateSettings(updates);
  }

  private getDefaultSettings(): AppSettings {
    return {
      general: {
        theme: 'dark',
        language: 'en',
        timezone: Intl.DateTimeFormat().resolvedOptions().timeZone,
        autoSave: true,
        autoBackup: true,
        backupInterval: 30,
        debugMode: false,
        telemetryEnabled: true
      },
      cognitive: {
        defaultModel: 'gpt-4',
        maxTokens: 2048,
        temperature: 0.7,
        topP: 1.0,
        frequencyPenalty: 0.0,
        presencePenalty: 0.0,
        autoLearning: true,
        skillCaching: true,
        adaptationRate: 0.1,
        contextWindow: 4096
      },
      wallet: {
        defaultNetwork: 'knirv-mainnet',
        autoConnect: true,
        transactionTimeout: 300,
        gasLimit: 21000,
        slippageTolerance: 0.5,
        confirmationBlocks: 3,
        showTestnets: false,
        currencyDisplay: 'NRN'
      },
      analytics: {
        collectMetrics: true,
        shareAnonymousData: false,
        retentionPeriod: 30,
        metricsInterval: 30,
        alertThresholds: {
          cpuUsage: 80,
          memoryUsage: 85,
          errorRate: 5,
          responseTime: 1000
        }
      },
      security: {
        requireMFA: false,
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
      },
      ui: {
        compactMode: false,
        showTooltips: true,
        animationsEnabled: true,
        soundEnabled: true,
        notificationsEnabled: true,
        panelLayout: 'default',
        fontSize: 'medium',
        colorScheme: 'blue-purple'
      },
      advanced: {
        apiEndpoints: {
          cognitive: 'http://localhost:3001/api/cognitive',
          wallet: 'http://localhost:3001/api/wallet',
          analytics: 'http://localhost:3001/api/analytics'
        },
        featureFlags: {
          experimentalUI: false,
          betaFeatures: false,
          advancedMetrics: false
        },
        experimentalFeatures: [],
        customCommands: {},
        integrations: {},
        performance: {
          maxConcurrentTasks: 10,
          cacheSize: 100,
          preloadData: true,
          lazyLoading: true
        }
      }
    };
  }

  private mergeSettings(current: AppSettings, updates: PartialAppSettings): AppSettings {
    const merged = { ...current };
    
    for (const [key, value] of Object.entries(updates)) {
      if (value && typeof value === 'object' && !Array.isArray(value)) {
        (merged as Record<string, unknown>)[key] = {
          ...(merged as Record<string, unknown>)[key],
          ...value
        };
      } else {
        (merged as Record<string, unknown>)[key] = value;
      }
    }
    
    return merged;
  }

  private validateSettings(settings: AppSettings): void {
    // Basic validation - would be more comprehensive in production
    if (!settings.general || !settings.cognitive || !settings.wallet) {
      throw new Error('Invalid settings structure');
    }

    // Validate general settings structure
    if (settings.general) {
      const validGeneralKeys = ['theme', 'language', 'timezone', 'autoSave', 'autoBackup', 'backupInterval', 'debugMode', 'telemetryEnabled'];
      for (const key of Object.keys(settings.general)) {
        if (!validGeneralKeys.includes(key)) {
          throw new Error('Invalid settings structure');
        }
      }
    }

    if (settings.cognitive.temperature < 0 || settings.cognitive.temperature > 2) {
      throw new Error('Invalid temperature value');
    }

    if (settings.security.sessionTimeout < 5 || settings.security.sessionTimeout > 1440) {
      throw new Error('Invalid session timeout value');
    }
  }

  private saveToLocalStorage(): void {
    try {
      localStorage.setItem('knirv-settings', JSON.stringify({
        settings: this.currentSettings,
        activeProfile: this.activeProfileId,
        lastUpdated: new Date().toISOString()
      }));
    } catch (error) {
      console.error('Failed to save settings to localStorage:', error);
    }
  }

  private loadFromLocalStorage(): void {
    try {
      const stored = localStorage.getItem('knirv-settings');
      if (stored) {
        const data = JSON.parse(stored);
        if (data.settings) {
          this.currentSettings = this.mergeSettings(this.getDefaultSettings(), data.settings);
        }
        if (data.activeProfile) {
          this.activeProfileId = data.activeProfile;
        }
      }
    } catch (error) {
      console.error('Failed to load settings from localStorage:', error);
    }
  }

  private async saveToBackend(): Promise<void> {
    if (!this.config.enableNetworking || !this.config.enableSync) {
      return;
    }

    try {
      await fetch(`${this.baseUrl}/api/settings/save`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          settings: this.currentSettings,
          activeProfile: this.activeProfileId
        })
      });
    } catch (error) {
      console.error('Failed to save settings to backend:', error);
    }
  }

  private async saveProfilesToBackend(): Promise<void> {
    if (!this.config.enableNetworking || !this.config.enableSync) {
      return;
    }

    try {
      await fetch(`${this.baseUrl}/api/settings/profiles`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(Object.fromEntries(this.profiles))
      });
    } catch (error) {
      console.error('Failed to save profiles to backend:', error);
    }
  }

  private async syncWithBackend(): Promise<void> {
    try {
      const response = await fetch(`${this.baseUrl}/api/settings/load`);
      if (response.ok) {
        const data = await response.json();
        if (data.settings) {
          this.currentSettings = this.mergeSettings(this.getDefaultSettings(), data.settings);
        }
        if (data.activeProfile) {
          this.activeProfileId = data.activeProfile;
        }
        if (data.profiles) {
          this.profiles.clear();
          for (const [id, profile] of Object.entries(data.profiles)) {
            this.profiles.set(id, profile as SettingsProfile);
          }
        }
      }
    } catch (error) {
      console.error('Failed to sync with backend:', error);
    }
  }

  private notifyListeners(): void {
    for (const listener of this.changeListeners) {
      try {
        listener(this.currentSettings);
      } catch (error) {
        console.error('Error in settings change listener:', error);
      }
    }
  }

  private generateProfileId(): string {
    return `profile_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
  }
}

// Export singleton instance
export const settingsService = new SettingsService();
