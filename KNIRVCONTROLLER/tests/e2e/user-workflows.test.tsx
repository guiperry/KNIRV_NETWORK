/**
 * End-to-End User Workflow Tests
 * Tests complete user journeys through the application
 */

import React from 'react';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import '@testing-library/jest-dom';
import Home from '../../src/pages/Home';
import Wallet from '../../src/pages/Wallet';
import { analyticsService } from '../../src/services/AnalyticsService';
import { taskSchedulingService } from '../../src/services/TaskSchedulingService';
import { udcManagementService } from '../../src/services/UDCManagementService';
import { settingsService } from '../../src/services/SettingsService';

// Mock all services
jest.mock('../../src/services/AnalyticsService');
jest.mock('../../src/services/TaskSchedulingService');
jest.mock('../../src/services/UDCManagementService');
jest.mock('../../src/services/SettingsService');

// Mock react-router-dom
jest.mock('react-router-dom', () => ({
  useNavigate: () => jest.fn(),
  useLocation: () => ({ pathname: '/' })
}));

// Mock desktop connection
jest.mock('../../src/services/DesktopConnection', () => ({
  desktopConnection: {
    getConnectionStatus: () => ({ connected: true, lastPing: Date.now() }),
    sendHRMRequest: jest.fn().mockResolvedValue({ success: true })
  }
}));

// Mock clipboard API
Object.assign(navigator, {
  clipboard: {
    writeText: jest.fn().mockResolvedValue(undefined)
  }
});

// Mock URL.createObjectURL
global.URL.createObjectURL = jest.fn(() => 'mock-url');
global.URL.revokeObjectURL = jest.fn();

describe('End-to-End User Workflows', () => {
  const mockAnalyticsService = analyticsService as jest.Mocked<typeof analyticsService>;
  const mockTaskSchedulingService = taskSchedulingService as jest.Mocked<typeof taskSchedulingService>;
  const mockUDCManagementService = udcManagementService as jest.Mocked<typeof udcManagementService>;
  const mockSettingsService = settingsService as jest.Mocked<typeof settingsService>;

  beforeEach(() => {
    jest.clearAllMocks();
    jest.useFakeTimers();

    // Setup default mock responses
    mockAnalyticsService.getDashboardStats.mockResolvedValue({
      activeAgents: 25,
      targetSystems: 12,
      inferencesToday: 1500,
      successRate: 96.5,
      changes: {},
      lastUpdated: new Date()
    });

    mockTaskSchedulingService.getAllTasks.mockReturnValue([]);
    mockTaskSchedulingService.createTask.mockResolvedValue({
      id: 'test-task-id',
      name: 'Test Task',
      description: 'Test Description',
      type: 'custom',
      status: 'pending',
      priority: 'medium',
      schedule: { type: 'once', startTime: new Date() },
      action: { type: 'api_call', target: 'http://example.com', parameters: {} },
      runCount: 0,
      successCount: 0,
      failureCount: 0,
      createdAt: new Date(),
      metadata: {}
    });

    mockUDCManagementService.getAllUDCs.mockReturnValue([]);
    mockUDCManagementService.createUDC.mockResolvedValue({
      id: 'test-udc-id',
      agentId: 'test-agent',
      type: 'basic',
      authorityLevel: 'read',
      status: 'active',
      issuedDate: new Date(),
      expiresDate: new Date(Date.now() + 30 * 24 * 60 * 60 * 1000),
      scope: 'Test scope',
      permissions: ['read'],
      signature: 'test-signature',
      issuer: 'KNIRV-CONTROLLER',
      subject: 'test-agent',
      metadata: {
        version: '1.0',
        description: 'Test UDC',
        tags: [],
        usage: { executionCount: 0, lastUsed: new Date(), usageHistory: [] },
        constraints: { maxExecutions: 1000, allowedHours: [] },
        security: {
          securityFlags: [],
          encryptionLevel: 'standard',
          requiresMFA: false
        }
      }
    });

    mockSettingsService.getSettings.mockReturnValue({
      general: { theme: 'dark', language: 'en', timezone: 'UTC', backupInterval: 60, autoSave: true, autoBackup: false, debugMode: false, telemetryEnabled: true },
      cognitive: {
        defaultModel: 'gpt-4',
        maxTokens: 4096,
        temperature: 0.7,
        topP: 0.9,
        autoLearning: true,
        skillCaching: true,
        frequencyPenalty: 0,
        presencePenalty: 0,
        adaptationRate: 0.1,
        contextWindow: 4096
      },
      wallet: { defaultNetwork: 'knirv-mainnet', autoConnect: true, transactionTimeout: 30000, gasLimit: 200000, slippageTolerance: 0.5, confirmationBlocks: 1, showTestnets: false, currencyDisplay: 'NRN' },
      analytics: { collectMetrics: true, shareAnonymousData: false, retentionPeriod: 30, metricsInterval: 60, alertThresholds: { memoryUsage: 80, cpuUsage: 80, errorRate: 5, responseTime: 1000 } },
      security: { requireMFA: false, sessionTimeout: 30, maxLoginAttempts: 3, passwordPolicy: { minLength: 8, requireUppercase: true, requireLowercase: true, requireNumbers: true, requireSymbols: false }, encryptionLevel: 'standard', auditLogging: true },
      ui: { compactMode: false, showTooltips: true, animationsEnabled: true, soundEnabled: false, notificationsEnabled: true, panelLayout: 'default', fontSize: 'medium', colorScheme: 'dark' },
      advanced: {
        featureFlags: { betaFeatures: false, experimentalUI: false },
        apiEndpoints: {},
        experimentalFeatures: [],
        customCommands: {},
        integrations: {},
        performance: { maxConcurrentTasks: 10, cacheSize: 100, preloadData: true, lazyLoading: false }
      }
    });
  });

  afterEach(() => {
    jest.useRealTimers();
  });

  describe('Analytics Dashboard Workflow', () => {
    it('should complete full analytics viewing workflow', async () => {
      render(<Home />);

      // Step 1: User clicks "View Analytics" button
      const viewAnalyticsButton = screen.getByText('View Analytics');
      fireEvent.click(viewAnalyticsButton);

      // Step 2: Analytics dashboard opens
      await waitFor(() => {
        expect(screen.getByText('Analytics Dashboard')).toBeInTheDocument();
      });

      // Step 3: User views different tabs
      fireEvent.click(screen.getByText('Performance'));
      await waitFor(() => {
        expect(mockAnalyticsService.getPerformanceMetrics).toHaveBeenCalled();
      });

      fireEvent.click(screen.getByText('Usage'));
      await waitFor(() => {
        expect(mockAnalyticsService.getUsageAnalytics).toHaveBeenCalled();
      });

      fireEvent.click(screen.getByText('Agents'));
      await waitFor(() => {
        expect(mockAnalyticsService.getAgentAnalytics).toHaveBeenCalled();
      });

      // Step 4: User refreshes data
      const refreshButton = screen.getByRole('button', { name: /refresh/i });
      fireEvent.click(refreshButton);

      await waitFor(() => {
        expect(mockAnalyticsService.getDashboardStats).toHaveBeenCalledTimes(2);
      });

      // Step 5: User exports data
      mockAnalyticsService.exportData.mockResolvedValue('{"data": "test"}');
      const exportButton = screen.getByRole('button', { name: /download/i });
      fireEvent.click(exportButton);

      await waitFor(() => {
        expect(mockAnalyticsService.exportData).toHaveBeenCalledWith('json');
      });

      // Step 6: User closes dashboard
      const closeButton = screen.getByText('×');
      fireEvent.click(closeButton);

      expect(screen.queryByText('Analytics Dashboard')).not.toBeInTheDocument();
    });
  });

  describe('Task Scheduling Workflow', () => {
    it('should complete full task creation and management workflow', async () => {
      render(<Home />);

      // Step 1: User clicks "Schedule Task" button
      const scheduleTaskButton = screen.getByText('Schedule Task');
      fireEvent.click(scheduleTaskButton);

      // Step 2: Task scheduler opens
      await waitFor(() => {
        expect(screen.getByText('Task Scheduler')).toBeInTheDocument();
      });

      // Step 3: User navigates to create task tab
      fireEvent.click(screen.getByText('Create Task'));

      // Step 4: User fills out task creation form
      fireEvent.change(screen.getByLabelText('Task Name'), {
        target: { value: 'E2E Test Task' }
      });
      fireEvent.change(screen.getByLabelText('Description'), {
        target: { value: 'End-to-end test task description' }
      });
      fireEvent.change(screen.getByLabelText('Target'), {
        target: { value: 'http://example.com/api' }
      });

      // Step 5: User submits form
      const createButton = screen.getByText('Create Task');
      fireEvent.click(createButton);

      await waitFor(() => {
        expect(mockTaskSchedulingService.createTask).toHaveBeenCalledWith(
          expect.objectContaining({
            name: 'E2E Test Task',
            description: 'End-to-end test task description'
          })
        );
      });

      // Step 6: User returns to tasks list
      await waitFor(() => {
        expect(screen.getByText('Tasks')).toBeInTheDocument();
      });

      // Step 7: User closes scheduler
      const closeButton = screen.getByText('×');
      fireEvent.click(closeButton);

      expect(screen.queryByText('Task Scheduler')).not.toBeInTheDocument();
    });
  });

  describe('UDC Management Workflow', () => {
    it('should complete full UDC creation and management workflow', async () => {
      render(<Home />);

      // Step 1: User clicks "Renew UDC" button (opens UDC Manager)
      const renewUDCButton = screen.getByText('Renew UDC');
      fireEvent.click(renewUDCButton);

      // Step 2: UDC Manager opens
      await waitFor(() => {
        expect(screen.getByText('UDC Manager')).toBeInTheDocument();
      });

      // Step 3: User navigates to create UDC tab
      fireEvent.click(screen.getByText('Create UDC'));

      // Step 4: User fills out UDC creation form
      fireEvent.change(screen.getByLabelText('Agent ID'), {
        target: { value: 'e2e-test-agent' }
      });
      fireEvent.change(screen.getByLabelText('Scope Description'), {
        target: { value: 'E2E test UDC scope' }
      });
      fireEvent.change(screen.getByLabelText('Authority Level'), {
        target: { value: 'write' }
      });

      // Step 5: User submits form
      const createButton = screen.getByText('Create UDC');
      fireEvent.click(createButton);

      await waitFor(() => {
        expect(mockUDCManagementService.createUDC).toHaveBeenCalledWith(
          expect.objectContaining({
            agentId: 'e2e-test-agent',
            scope: 'E2E test UDC scope',
            authorityLevel: 'write'
          })
        );
      });

      // Step 6: User checks expiring UDCs
      fireEvent.click(screen.getByText('Expiring Soon'));
      await waitFor(() => {
        expect(mockUDCManagementService.getExpiringUDCs).toHaveBeenCalled();
      });

      // Step 7: User closes UDC Manager
      const closeButton = screen.getByText('×');
      fireEvent.click(closeButton);

      expect(screen.queryByText('UDC Manager')).not.toBeInTheDocument();
    });
  });

  describe('Wallet Operations Workflow', () => {
    it('should complete full wallet interaction workflow', async () => {
      render(<Wallet />);

      // Step 1: User copies wallet address
      const copyButton = screen.getByRole('button', { name: 'Copy Address' });
      fireEvent.click(copyButton);

      await waitFor(() => {
        expect(navigator.clipboard.writeText).toHaveBeenCalled();
      });

      // Step 2: User views QR code
      const qrButton = screen.getByRole('button', { name: 'Show QR Code' });
      fireEvent.click(qrButton);

      await waitFor(() => {
        expect(screen.getByText('Wallet QR Code')).toBeInTheDocument();
      });

      // Close QR modal
      fireEvent.click(screen.getByText('×'));

      // Step 3: User opens Add Funds modal
      const addFundsButton = screen.getByText('Add Funds');
      fireEvent.click(addFundsButton);

      await waitFor(() => {
        expect(screen.getByText('Add Funds')).toBeInTheDocument();
      });

      // Fill out add funds form
      fireEvent.change(screen.getByPlaceholderText('0.00'), {
        target: { value: '100' }
      });
      fireEvent.change(screen.getByDisplayValue('Credit Card'), {
        target: { value: 'Bank Transfer' }
      });

      // Close add funds modal
      fireEvent.click(screen.getByText('×'));

      // Step 4: User opens Send NRN modal
      const sendNRNButton = screen.getByText('Send NRN');
      fireEvent.click(sendNRNButton);

      await waitFor(() => {
        expect(screen.getByText('Send NRN')).toBeInTheDocument();
      });

      // Fill out send form
      fireEvent.change(screen.getByPlaceholderText('0x...'), {
        target: { value: '0x1234567890abcdef' }
      });
      fireEvent.change(screen.getAllByPlaceholderText('0.00')[0], {
        target: { value: '50' }
      });

      // Close send modal
      fireEvent.click(screen.getByText('×'));
    });
  });

  describe('Settings Management Workflow', () => {
    it('should complete full settings configuration workflow', async () => {
      // Mock settings panel opening (would typically be triggered from a menu)
      const { default: SettingsPanel } = await import('../../src/components/SettingsPanel');
      const { rerender } = render(<SettingsPanel isOpen={false} onClose={jest.fn()} />);

      // Step 1: Open settings panel
      rerender(<SettingsPanel isOpen={true} onClose={jest.fn()} />);

      await waitFor(() => {
        expect(screen.getByText('Settings')).toBeInTheDocument();
      });

      // Step 2: User modifies general settings
      fireEvent.change(screen.getByLabelText('Theme'), {
        target: { value: 'light' }
      });

      // Step 3: User navigates to cognitive settings
      fireEvent.click(screen.getByText('Cognitive'));

      await waitFor(() => {
        expect(screen.getByText('Cognitive Engine Settings')).toBeInTheDocument();
      });

      // Modify cognitive settings
      fireEvent.change(screen.getByLabelText('Temperature'), {
        target: { value: '0.8' }
      });

      // Step 4: User saves settings
      const saveButton = screen.getByText('Save');
      fireEvent.click(saveButton);

      await waitFor(() => {
        expect(mockSettingsService.updateSettings).toHaveBeenCalled();
      });

      // Step 5: User creates a new profile
      fireEvent.click(screen.getByText('Profiles'));
      
      await waitFor(() => {
        expect(screen.getByText('Settings Profiles')).toBeInTheDocument();
      });

      // Step 6: User exports settings
      mockSettingsService.exportSettings.mockReturnValue('{"settings": "data"}');
      const exportButton = screen.getByText('Export');
      fireEvent.click(exportButton);

      expect(mockSettingsService.exportSettings).toHaveBeenCalledWith(true);
    });
  });

  describe('Multi-Modal Interaction Workflow', () => {
    it('should handle voice and visual processing workflow', async () => {
      render(<Home />);

      // Step 1: User activates voice processing
      const voiceButton = screen.getByRole('button', { name: /voice/i });
      fireEvent.click(voiceButton);

      // Voice processing should be activated
      expect(voiceButton).toHaveClass('bg-red-500/20');

      // Step 2: User activates visual processing
      const visualButton = screen.getByRole('button', { name: /visual/i });
      fireEvent.click(visualButton);

      // Visual processing should be activated
      expect(visualButton).toHaveClass('bg-blue-500/20');

      // Step 3: User opens QR scanner
      const qrButton = screen.getByRole('button', { name: /qr/i });
      fireEvent.click(qrButton);

      await waitFor(() => {
        expect(screen.getByText('QR Code Scanner')).toBeInTheDocument();
      });

      // Close QR scanner
      const closeQRButton = screen.getByText('×');
      fireEvent.click(closeQRButton);

      // Step 4: Deactivate voice and visual processing
      fireEvent.click(voiceButton);
      fireEvent.click(visualButton);

      expect(voiceButton).not.toHaveClass('bg-red-500/20');
      expect(visualButton).not.toHaveClass('bg-blue-500/20');
    });
  });

  describe('Error Handling Workflows', () => {
    it('should handle service errors gracefully during user workflows', async () => {
      // Mock service failures
      mockAnalyticsService.getDashboardStats.mockRejectedValue(new Error('Analytics service down'));
      mockTaskSchedulingService.createTask.mockRejectedValue(new Error('Task service down'));

      render(<Home />);

      // Step 1: Try to open analytics (should handle error)
      const viewAnalyticsButton = screen.getByText('View Analytics');
      fireEvent.click(viewAnalyticsButton);

      // Should still open dashboard despite error
      await waitFor(() => {
        expect(screen.getByText('Analytics Dashboard')).toBeInTheDocument();
      });

      // Close analytics
      fireEvent.click(screen.getByText('×'));

      // Step 2: Try to create task (should handle error)
      const scheduleTaskButton = screen.getByText('Schedule Task');
      fireEvent.click(scheduleTaskButton);

      await waitFor(() => {
        expect(screen.getByText('Task Scheduler')).toBeInTheDocument();
      });

      fireEvent.click(screen.getByText('Create Task'));

      // Fill form and submit (should handle error gracefully)
      fireEvent.change(screen.getByLabelText('Task Name'), {
        target: { value: 'Error Test Task' }
      });
      fireEvent.change(screen.getByLabelText('Target'), {
        target: { value: 'http://example.com' }
      });

      const createButton = screen.getByText('Create Task');
      fireEvent.click(createButton);

      // Should not crash the application
      expect(screen.getByText('Task Scheduler')).toBeInTheDocument();
    });
  });

  describe('Performance Workflows', () => {
    it('should handle rapid user interactions without performance degradation', async () => {
      render(<Home />);

      const startTime = Date.now();

      // Rapid interactions
      for (let i = 0; i < 10; i++) {
        const viewAnalyticsButton = screen.getByText('View Analytics');
        fireEvent.click(viewAnalyticsButton);

        await waitFor(() => {
          expect(screen.getByText('Analytics Dashboard')).toBeInTheDocument();
        });

        const closeButton = screen.getByText('×');
        fireEvent.click(closeButton);

        await waitFor(() => {
          expect(screen.queryByText('Analytics Dashboard')).not.toBeInTheDocument();
        });
      }

      const endTime = Date.now();
      const duration = endTime - startTime;

      // Should complete within reasonable time (5 seconds for 10 iterations)
      expect(duration).toBeLessThan(5000);
    });
  });
});
