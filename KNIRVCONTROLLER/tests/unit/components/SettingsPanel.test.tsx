/**
 * SettingsPanel Component Tests
 * Comprehensive test suite for settings panel functionality
 */

import * as React from 'react';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import '@testing-library/jest-dom';
import SettingsPanel from '../../../src/components/SettingsPanel';
import { settingsService } from '../../../src/services/SettingsService';

// Mock the settings service
jest.mock('../../../src/services/SettingsService', () => ({
  settingsService: {
    getSettings: jest.fn(),
    getProfiles: jest.fn(),
    getActiveProfile: jest.fn(),
    updateSettings: jest.fn(),
    resetSettings: jest.fn(),
    exportSettings: jest.fn(),
    importSettings: jest.fn(),
    createProfile: jest.fn(),
    loadProfile: jest.fn(),
    deleteProfile: jest.fn(),
    addChangeListener: jest.fn(),
    removeChangeListener: jest.fn()
  }
}));

// Mock URL.createObjectURL and related APIs
global.URL.createObjectURL = jest.fn(() => 'mock-url');
global.URL.revokeObjectURL = jest.fn();

// Mock document.createElement and appendChild/removeChild
const mockAnchor = {
  href: '',
  download: '',
  click: jest.fn()
};
document.createElement = jest.fn().mockImplementation((tagName) => {
  if (tagName === 'a') {
    return mockAnchor;
  }
  return {};
});
document.body.appendChild = jest.fn();
document.body.removeChild = jest.fn();

// FileReader is already mocked in jest-setup.js

describe('SettingsPanel', () => {
  const mockSettingsService = settingsService as jest.Mocked<typeof settingsService>;
  
  const mockSettings = {
    general: {
      theme: 'dark' as const,
      language: 'en',
      timezone: 'UTC',
      backupInterval: 60,
      autoSave: true,
      autoBackup: false,
      debugMode: false,
      telemetryEnabled: true
    },
    cognitive: {
      defaultModel: 'gpt-4',
      maxTokens: 4096,
      temperature: 0.7,
      topP: 0.9,
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
      currencyDisplay: 'NRN' as const
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
      encryptionLevel: 'standard' as const,
      auditLogging: true,
      autoLock: true
    },
    ui: {
      compactMode: false,
      showTooltips: true,
      animationsEnabled: true,
      soundEnabled: true,
      notificationsEnabled: true,
      panelLayout: 'default' as const,
      fontSize: 'medium' as const,
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

  const mockProfiles = [
    {
      id: 'default',
      name: 'Default Profile',
      description: 'Default settings profile',
      settings: mockSettings,
      isDefault: true,
      createdAt: new Date('2024-01-01T00:00:00Z'),
      updatedAt: new Date('2024-01-01T00:00:00Z')
    },
    {
      id: 'dev-profile',
      name: 'Development Profile',
      description: 'Settings for development environment',
      settings: {
        ...mockSettings,
        general: { ...mockSettings.general, debugMode: true },
        cognitive: { ...mockSettings.cognitive }, // Ensure cognitive settings are fully copied
        advanced: {
          ...mockSettings.advanced,
          featureFlags: { ...mockSettings.advanced.featureFlags, betaFeatures: true, experimentalUI: true }
        }
      },
      isDefault: false,
      createdAt: new Date('2024-01-10T00:00:00Z'),
      updatedAt: new Date('2024-01-10T00:00:00Z')
    }
  ];

  beforeEach(() => {
    jest.clearAllMocks();
    
    mockSettingsService.getSettings.mockReturnValue(mockSettings);
    mockSettingsService.getProfiles.mockReturnValue(mockProfiles);
    mockSettingsService.getActiveProfile.mockReturnValue(mockProfiles[0]);
    mockSettingsService.updateSettings.mockResolvedValue(undefined);
    mockSettingsService.resetSettings.mockResolvedValue(undefined);
    mockSettingsService.exportSettings.mockReturnValue(JSON.stringify({
      settings: mockSettings,
      profiles: mockProfiles,
      exportedAt: new Date().toISOString(),
      version: '1.0'
    }));
    mockSettingsService.importSettings.mockResolvedValue(undefined);
    mockSettingsService.createProfile.mockResolvedValue(mockProfiles[1]);
    mockSettingsService.loadProfile.mockResolvedValue(undefined);
    mockSettingsService.deleteProfile.mockResolvedValue(undefined);
  });

  describe('Rendering', () => {
    it('should not render when isOpen is false', () => {
      render(<SettingsPanel isOpen={false} onClose={jest.fn()} />);
      
      expect(screen.queryByText('Settings')).not.toBeInTheDocument();
    });

    it('should render when isOpen is true', async () => {
      render(<SettingsPanel isOpen={true} onClose={jest.fn()} />);
      
      await waitFor(() => {
        expect(screen.getByText('Settings')).toBeInTheDocument();
        expect(screen.getByText('Configure KNIRV Controller preferences')).toBeInTheDocument();
      });
    });

    it('should render all sidebar tabs', async () => {
      render(<SettingsPanel isOpen={true} onClose={jest.fn()} />);
      
      await waitFor(() => {
        expect(screen.getByText('General')).toBeInTheDocument();
        expect(screen.getByText('Cognitive')).toBeInTheDocument();
        expect(screen.getByText('Wallet')).toBeInTheDocument();
        expect(screen.getByText('Analytics')).toBeInTheDocument();
        expect(screen.getByText('Security')).toBeInTheDocument();
        expect(screen.getByText('Interface')).toBeInTheDocument();
        expect(screen.getByText('Advanced')).toBeInTheDocument();
        expect(screen.getByText('Profiles')).toBeInTheDocument();
      });
    });

    it('should render sidebar action buttons', async () => {
      render(<SettingsPanel isOpen={true} onClose={jest.fn()} />);
      
      await waitFor(() => {
        expect(screen.getByText('Export')).toBeInTheDocument();
        expect(screen.getByText('Import')).toBeInTheDocument();
        expect(screen.getByText('Reset')).toBeInTheDocument();
      });
    });
  });

  describe('Data Loading', () => {
    it('should load settings data on mount', async () => {
      render(<SettingsPanel isOpen={true} onClose={jest.fn()} />);
      
      await waitFor(() => {
        expect(mockSettingsService.getSettings).toHaveBeenCalled();
        expect(mockSettingsService.getProfiles).toHaveBeenCalled();
        expect(mockSettingsService.getActiveProfile).toHaveBeenCalled();
      });
    });

    it('should handle loading errors gracefully', async () => {
      mockSettingsService.getSettings.mockImplementation(() => {
        throw new Error('Loading failed');
      });
      
      render(<SettingsPanel isOpen={true} onClose={jest.fn()} />);
      
      // Should not crash
      expect(screen.getByText('Settings')).toBeInTheDocument();
    });
  });

  describe('General Settings Tab', () => {
    beforeEach(async () => {
      render(<SettingsPanel isOpen={true} onClose={jest.fn()} />);
      
      await waitFor(() => {
        expect(screen.getByText('General Settings')).toBeInTheDocument();
      });
    });

    it('should display general settings form', () => {
      expect(screen.getByLabelText('Theme')).toBeInTheDocument();
      expect(screen.getByLabelText('Language')).toBeInTheDocument();
      expect(screen.getByLabelText('Timezone')).toBeInTheDocument();
      expect(screen.getByLabelText('Backup Interval (minutes)')).toBeInTheDocument();
    });

    it('should display toggle switches', () => {
      expect(screen.getByText('Auto Save')).toBeInTheDocument();
      expect(screen.getByText('Auto Backup')).toBeInTheDocument();
      expect(screen.getByText('Debug Mode')).toBeInTheDocument();
      expect(screen.getByText('Telemetry')).toBeInTheDocument();
    });

    it('should update settings when form fields change', async () => {
      const themeSelect = screen.getByLabelText('Theme');
      fireEvent.change(themeSelect, { target: { value: 'light' } });
      
      // Should show save button
      await waitFor(() => {
        expect(screen.getByText('Save')).toBeInTheDocument();
      });
    });

    it('should toggle boolean settings', async () => {
      const autoSaveToggle = screen.getByText('Auto Save').closest('label')?.querySelector('input');
      
      if (autoSaveToggle) {
        fireEvent.click(autoSaveToggle);
        
        await waitFor(() => {
          expect(screen.getByText('Save')).toBeInTheDocument();
        });
      }
    });
  });

  describe('Cognitive Settings Tab', () => {
    beforeEach(async () => {
      render(<SettingsPanel isOpen={true} onClose={jest.fn()} />);
      
      await waitFor(() => {
        fireEvent.click(screen.getByText('Cognitive'));
      });
    });

    it('should display cognitive settings form', async () => {
      await waitFor(() => {
        expect(screen.getByText('Cognitive Engine Settings')).toBeInTheDocument();
        expect(screen.getByLabelText('Default Model')).toBeInTheDocument();
        expect(screen.getByLabelText('Max Tokens')).toBeInTheDocument();
        expect(screen.getByLabelText('Temperature')).toBeInTheDocument();
        expect(screen.getByLabelText('Top P')).toBeInTheDocument();
      });
    });

    it('should display range sliders with values', async () => {
      await waitFor(() => {
        expect(screen.getByText('0.7')).toBeInTheDocument(); // Temperature value
        expect(screen.getByText('0.9')).toBeInTheDocument(); // Top P value
      });
    });

    it('should display cognitive toggle switches', async () => {
      await waitFor(() => {
        expect(screen.getByText('Auto Learning')).toBeInTheDocument();
        expect(screen.getByText('Skill Caching')).toBeInTheDocument();
      });
    });
  });

  describe('Profiles Tab', () => {
    beforeEach(async () => {
      render(<SettingsPanel isOpen={true} onClose={jest.fn()} />);
      
      await waitFor(() => {
        fireEvent.click(screen.getByText('Profiles'));
      });
    });

    it('should display profiles list', async () => {
      await waitFor(() => {
        expect(screen.getByText('Settings Profiles')).toBeInTheDocument();
        expect(screen.getByText('Default Profile')).toBeInTheDocument();
        expect(screen.getByText('Development Profile')).toBeInTheDocument();
      });
    });

    it('should show active profile indicator', async () => {
      await waitFor(() => {
        expect(screen.getByText('Active')).toBeInTheDocument();
      });
    });

    it('should load profile when Load button is clicked', async () => {
      await waitFor(() => {
        const loadButtons = screen.getAllByText('Load');
        fireEvent.click(loadButtons[1]); // Load development profile
      });
      
      expect(mockSettingsService.loadProfile).toHaveBeenCalledWith('dev-profile');
    });

    it('should delete profile when Delete button is clicked', async () => {
      await waitFor(() => {
        const deleteButtons = screen.getAllByText('Delete');
        fireEvent.click(deleteButtons[0]); // Delete development profile
      });
      
      expect(mockSettingsService.deleteProfile).toHaveBeenCalledWith('dev-profile');
    });

    it('should not show delete button for default profile', async () => {
      await waitFor(() => {
        const profileCards = screen.getAllByText(/Profile/);
        const defaultCard = profileCards[0].closest('div');
        
        expect(defaultCard?.querySelector('button:contains("Delete")')).toBeFalsy();
      });
    });
  });

  describe('Save Functionality', () => {
    it('should save settings when Save button is clicked', async () => {
      render(<SettingsPanel isOpen={true} onClose={jest.fn()} />);
      
      await waitFor(() => {
        const themeSelect = screen.getByLabelText('Theme');
        fireEvent.change(themeSelect, { target: { value: 'light' } });
      });
      
      const saveButton = screen.getByText('Save');
      fireEvent.click(saveButton);
      
      await waitFor(() => {
        expect(mockSettingsService.updateSettings).toHaveBeenCalled();
      });
    });

    it('should hide save button after successful save', async () => {
      render(<SettingsPanel isOpen={true} onClose={jest.fn()} />);
      
      await waitFor(() => {
        const themeSelect = screen.getByLabelText('Theme');
        fireEvent.change(themeSelect, { target: { value: 'light' } });
      });
      
      const saveButton = screen.getByText('Save');
      fireEvent.click(saveButton);
      
      await waitFor(() => {
        expect(screen.queryByText('Save')).not.toBeInTheDocument();
      });
    });

    it('should show save button only when there are changes', async () => {
      render(<SettingsPanel isOpen={true} onClose={jest.fn()} />);
      
      await waitFor(() => {
        expect(screen.queryByText('Save')).not.toBeInTheDocument();
      });
      
      const themeSelect = screen.getByLabelText('Theme');
      fireEvent.change(themeSelect, { target: { value: 'light' } });
      
      await waitFor(() => {
        expect(screen.getByText('Save')).toBeInTheDocument();
      });
    });
  });

  describe('Export/Import Functionality', () => {
    it('should export settings when Export button is clicked', async () => {
      render(<SettingsPanel isOpen={true} onClose={jest.fn()} />);
      
      await waitFor(() => {
        const exportButton = screen.getByText('Export');
        fireEvent.click(exportButton);
      });
      
      expect(mockSettingsService.exportSettings).toHaveBeenCalledWith(true);
      expect(mockAnchor.download).toBe('knirv-settings.json');
      expect(mockAnchor.click).toHaveBeenCalled();
    });

    it('should handle file import', async () => {
      const mockFile = new File(['{"settings": {}}'], 'settings.json', { type: 'application/json' });
      const mockFileReader = {
        readAsText: jest.fn(),
        onload: jest.fn(),
        result: '{"settings": {}}'
      };
      
      (global.FileReader as unknown as jest.Mock).mockImplementation(() => mockFileReader);
      
      render(<SettingsPanel isOpen={true} onClose={jest.fn()} />);
      
      await waitFor(() => {
        const importInput = screen.getByText('Import').closest('label')?.querySelector('input');
        
        if (importInput) {
          Object.defineProperty(importInput, 'files', {
            value: [mockFile],
            writable: false
          });
          
          fireEvent.change(importInput);
          
          // Simulate FileReader onload
          if (mockFileReader.onload) {
            const progressEvent = {
              target: mockFileReader,
              lengthComputable: false,
              loaded: 0,
              total: 0
            } as unknown as ProgressEvent<FileReader>;
            mockFileReader.onload(progressEvent);
          }
        }
      });
      
      expect(mockSettingsService.importSettings).toHaveBeenCalledWith('{"settings": {}}', false);
    });
  });

  describe('Reset Functionality', () => {
    it('should reset settings when Reset button is clicked', async () => {
      render(<SettingsPanel isOpen={true} onClose={jest.fn()} />);
      
      await waitFor(() => {
        const resetButton = screen.getByText('Reset');
        fireEvent.click(resetButton);
      });
      
      expect(mockSettingsService.resetSettings).toHaveBeenCalled();
    });
  });

  describe('Refresh Functionality', () => {
    it('should refresh data when refresh button is clicked', async () => {
      render(<SettingsPanel isOpen={true} onClose={jest.fn()} />);
      
      await waitFor(() => {
        expect(mockSettingsService.getSettings).toHaveBeenCalledTimes(1);
      });
      
      const refreshButton = screen.getByRole('button', { name: /refresh/i });
      fireEvent.click(refreshButton);
      
      await waitFor(() => {
        expect(mockSettingsService.getSettings).toHaveBeenCalledTimes(2);
      });
    });
  });

  describe('Close Functionality', () => {
    it('should call onClose when close button is clicked', async () => {
      const onCloseMock = jest.fn();
      render(<SettingsPanel isOpen={true} onClose={onCloseMock} />);
      
      await waitFor(() => {
        const closeButton = screen.getByText('×');
        fireEvent.click(closeButton);
      });
      
      expect(onCloseMock).toHaveBeenCalled();
    });
  });

  describe('Tab Navigation', () => {
    it('should highlight active tab', async () => {
      render(<SettingsPanel isOpen={true} onClose={jest.fn()} />);
      
      await waitFor(() => {
        const generalTab = screen.getByText('General').closest('button');
        const cognitiveTab = screen.getByText('Cognitive').closest('button');
        
        // General should be active by default
        expect(generalTab).toHaveClass('text-blue-400');
        
        fireEvent.click(screen.getByText('Cognitive'));
        
        expect(cognitiveTab).toHaveClass('text-blue-400');
      });
    });

    it('should switch content when tabs are clicked', async () => {
      render(<SettingsPanel isOpen={true} onClose={jest.fn()} />);
      
      await waitFor(() => {
        expect(screen.getByText('General Settings')).toBeInTheDocument();
        
        fireEvent.click(screen.getByText('Cognitive'));
        
        expect(screen.getByText('Cognitive Engine Settings')).toBeInTheDocument();
      });
    });
  });

  describe('Error Handling', () => {
    it('should handle save errors gracefully', async () => {
      mockSettingsService.updateSettings.mockRejectedValue(new Error('Save failed'));
      
      render(<SettingsPanel isOpen={true} onClose={jest.fn()} />);
      
      await waitFor(() => {
        const themeSelect = screen.getByLabelText('Theme');
        fireEvent.change(themeSelect, { target: { value: 'light' } });
      });
      
      const saveButton = screen.getByText('Save');
      fireEvent.click(saveButton);
      
      // Should not crash
      expect(screen.getByText('Settings')).toBeInTheDocument();
    });

    it('should handle reset errors gracefully', async () => {
      mockSettingsService.resetSettings.mockRejectedValue(new Error('Reset failed'));
      
      render(<SettingsPanel isOpen={true} onClose={jest.fn()} />);
      
      await waitFor(() => {
        const resetButton = screen.getByText('Reset');
        fireEvent.click(resetButton);
      });
      
      // Should not crash
      expect(screen.getByText('Settings')).toBeInTheDocument();
    });
  });

  describe('Accessibility', () => {
    it('should have proper ARIA labels', async () => {
      render(<SettingsPanel isOpen={true} onClose={jest.fn()} />);
      
      await waitFor(() => {
        expect(screen.getByRole('button', { name: /refresh/i })).toBeInTheDocument();
      });
    });

    it('should support keyboard navigation', async () => {
      render(<SettingsPanel isOpen={true} onClose={jest.fn()} />);
      
      await waitFor(() => {
        const generalTab = screen.getByText('General').closest('button');
        const cognitiveTab = screen.getByText('Cognitive').closest('button');
        
        expect(generalTab).toBeInTheDocument();
        expect(cognitiveTab).toBeInTheDocument();
        
        // Tabs should be focusable
        generalTab?.focus();
        expect(document.activeElement).toBe(generalTab);
      });
    });
  });

  describe('Form Validation', () => {
    it('should validate numeric inputs', async () => {
      render(<SettingsPanel isOpen={true} onClose={jest.fn()} />);
      
      await waitFor(() => {
        const backupIntervalInput = screen.getByLabelText('Backup Interval (minutes)');
        fireEvent.change(backupIntervalInput, { target: { value: '0' } });
      });
      
      // Should enforce minimum value
      const backupIntervalInput = screen.getByLabelText('Backup Interval (minutes)');
      expect(backupIntervalInput).toHaveAttribute('min', '5');
    });

    it('should validate range inputs', async () => {
      render(<SettingsPanel isOpen={true} onClose={jest.fn()} />);
      
      await waitFor(() => {
        fireEvent.click(screen.getByText('Cognitive'));
      });
      
      await waitFor(() => {
        const temperatureSlider = screen.getByLabelText('Temperature');
        expect(temperatureSlider).toHaveAttribute('min', '0');
        expect(temperatureSlider).toHaveAttribute('max', '2');
      });
    });
  });
});
