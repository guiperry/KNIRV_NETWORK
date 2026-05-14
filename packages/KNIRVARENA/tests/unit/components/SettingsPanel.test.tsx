import React from 'react';
import { render, screen, fireEvent } from '@testing-library/react';
import '@testing-library/jest-dom';
import SettingsPanel from '../../../src/components/SettingsPanel';

// Clean up after each test
afterEach(() => {
  // Clean up any remaining DOM elements
  document.body.innerHTML = '';
});

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

// Mock the settings service
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
    getProfiles: jest.fn().mockReturnValue([
      { id: 'default', name: 'Default', isDefault: true },
      { id: 'custom', name: 'Custom Profile', isDefault: false }
    ]),
    getActiveProfile: jest.fn().mockReturnValue({ id: 'default', name: 'Default', isDefault: true }),
    updateSettings: jest.fn().mockResolvedValue(undefined),
    resetSettings: jest.fn().mockResolvedValue(undefined),
    exportSettings: jest.fn().mockReturnValue('{}'),
    importSettings: jest.fn().mockResolvedValue(undefined),
    createProfile: jest.fn().mockResolvedValue(undefined),
    loadProfile: jest.fn().mockResolvedValue(undefined),
    deleteProfile: jest.fn().mockResolvedValue(undefined),
    addChangeListener: jest.fn(),
    removeChangeListener: jest.fn()
  }
}));

describe('SettingsPanel', () => {
  it('should render when isOpen is true', () => {
    render(<SettingsPanel isOpen={true} onClose={jest.fn()} />);
    
    expect(screen.getByText('Settings')).toBeInTheDocument();
  });

  it('should not render when isOpen is false', () => {
    render(<SettingsPanel isOpen={false} onClose={jest.fn()} />);
    
    expect(screen.queryByText('Settings')).not.toBeInTheDocument();
  });

  it('should render all sidebar tabs', () => {
    render(<SettingsPanel isOpen={true} onClose={jest.fn()} />);

    expect(screen.getByText('General')).toBeInTheDocument();
    expect(screen.getByText('Cognitive')).toBeInTheDocument();
    expect(screen.getByText('Wallet')).toBeInTheDocument();
    expect(screen.getByText('Analytics')).toBeInTheDocument();
    expect(screen.getByText('Security')).toBeInTheDocument();
    expect(screen.getByText('Interface')).toBeInTheDocument();
    expect(screen.getByText('Advanced')).toBeInTheDocument();
    expect(screen.getByText('Profiles')).toBeInTheDocument();
  });

  it('should call onClose when close button is clicked', () => {
    const mockOnClose = jest.fn();
    render(<SettingsPanel isOpen={true} onClose={mockOnClose} />);

    const closeButton = screen.getByText('×');
    fireEvent.click(closeButton);

    expect(mockOnClose).toHaveBeenCalledTimes(1);
  });

  it('should switch tabs when clicked', () => {
    render(<SettingsPanel isOpen={true} onClose={jest.fn()} />);

    // Click on Cognitive tab
    fireEvent.click(screen.getByText('Cognitive'));

    // Should show cognitive settings content
    expect(screen.getByText('Cognitive Engine Settings')).toBeInTheDocument();
    expect(screen.getByText('Default Model')).toBeInTheDocument();
  });

  it('should show export and import options', () => {
    render(<SettingsPanel isOpen={true} onClose={jest.fn()} />);

    expect(screen.getByText('Export')).toBeInTheDocument();
    expect(screen.getByText('Import')).toBeInTheDocument();
    expect(screen.getByText('Reset')).toBeInTheDocument();
  });
});

describe('SettingsPanel - General Tab', () => {
  beforeEach(() => {
    render(<SettingsPanel isOpen={true} onClose={jest.fn()} />);
  });

  it('should display theme setting', () => {
    expect(screen.getByText('Theme')).toBeInTheDocument();
  });

  it('should display language setting', () => {
    expect(screen.getByText('Language')).toBeInTheDocument();
  });
});

describe('SettingsPanel - Cognitive Tab', () => {
  beforeEach(() => {
    render(<SettingsPanel isOpen={true} onClose={jest.fn()} />);
    fireEvent.click(screen.getByText('Cognitive'));
  });

  it('should display temperature setting', () => {
    expect(screen.getByText('Temperature')).toBeInTheDocument();
  });

  it('should display top p setting', () => {
    expect(screen.getByText('Top P')).toBeInTheDocument();
  });
});
