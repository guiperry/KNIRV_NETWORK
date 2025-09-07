/**
 * AnalyticsDashboard Component Tests
 * Comprehensive test suite for analytics dashboard functionality
 */

import * as React from 'react';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import '@testing-library/jest-dom';
import AnalyticsDashboard from '../../../src/components/AnalyticsDashboard';
import { analyticsService } from '../../../src/services/AnalyticsService';

// Mock the analytics service
jest.mock('../../../src/services/AnalyticsService', () => ({
  analyticsService: {
    getDashboardStats: jest.fn(),
    getPerformanceMetrics: jest.fn(),
    getUsageAnalytics: jest.fn(),
    getAgentAnalytics: jest.fn(),
    exportData: jest.fn()
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

describe('AnalyticsDashboard', () => {
  const mockAnalyticsService = analyticsService as jest.Mocked<typeof analyticsService>;
  
  const mockDashboardStats = {
    activeAgents: 25,
    targetSystems: 12,
    inferencesToday: 1500,
    successRate: 96.5,
    changes: {
      active_agents: '+15%',
      target_systems: '+2',
      inferences_today: '+250',
      success_rate: '+1.2%'
    },
    lastUpdated: new Date('2024-01-15T10:30:00Z')
  };

  const mockPerformanceMetrics = {
    cpuUsage: 45.2,
    memoryUsage: 67.8,
    networkLatency: 25,
    responseTime: 150,
    throughput: 850,
    errorRate: 2.1
  };

  const mockUsageAnalytics = {
    totalSessions: 1250,
    averageSessionDuration: 28,
    mostUsedFeatures: [
      { feature: 'Agent Management', usage: 85 },
      { feature: 'Cognitive Shell', usage: 72 },
      { feature: 'Wallet Operations', usage: 58 }
    ],
    userEngagement: 87,
    peakUsageHours: [9, 10, 14, 15, 16]
  };

  const mockAgentAnalytics = {
    totalAgents: 45,
    activeAgents: 32,
    deploymentSuccess: 94.5,
    averageExecutionTime: 250,
    skillInvocations: 1850,
    errorCount: 8
  };

  beforeEach(() => {
    jest.clearAllMocks();
    jest.useFakeTimers();
    
    mockAnalyticsService.getDashboardStats.mockResolvedValue(mockDashboardStats);
    mockAnalyticsService.getPerformanceMetrics.mockResolvedValue(mockPerformanceMetrics);
    mockAnalyticsService.getUsageAnalytics.mockResolvedValue(mockUsageAnalytics);
    mockAnalyticsService.getAgentAnalytics.mockResolvedValue(mockAgentAnalytics);
  });

  afterEach(() => {
    jest.useRealTimers();
  });

  describe('Rendering', () => {
    it('should not render when isOpen is false', () => {
      render(<AnalyticsDashboard isOpen={false} onClose={jest.fn()} />);
      
      expect(screen.queryByText('Analytics Dashboard')).not.toBeInTheDocument();
    });

    it('should render when isOpen is true', async () => {
      render(<AnalyticsDashboard isOpen={true} onClose={jest.fn()} />);
      
      expect(screen.getByText('Analytics Dashboard')).toBeInTheDocument();
      expect(screen.getByText('Real-time system metrics and insights')).toBeInTheDocument();
    });

    it('should render all tabs', async () => {
      render(<AnalyticsDashboard isOpen={true} onClose={jest.fn()} />);
      
      expect(screen.getByText('Overview')).toBeInTheDocument();
      expect(screen.getByText('Performance')).toBeInTheDocument();
      expect(screen.getByText('Usage')).toBeInTheDocument();
      expect(screen.getByText('Agents')).toBeInTheDocument();
    });
  });

  describe('Data Loading', () => {
    it('should load analytics data on mount', async () => {
      render(<AnalyticsDashboard isOpen={true} onClose={jest.fn()} />);
      
      await waitFor(() => {
        expect(mockAnalyticsService.getDashboardStats).toHaveBeenCalled();
        expect(mockAnalyticsService.getPerformanceMetrics).toHaveBeenCalled();
        expect(mockAnalyticsService.getUsageAnalytics).toHaveBeenCalled();
        expect(mockAnalyticsService.getAgentAnalytics).toHaveBeenCalled();
      });
    });

    it('should display dashboard stats in overview tab', async () => {
      render(<AnalyticsDashboard isOpen={true} onClose={jest.fn()} />);
      
      await waitFor(() => {
        expect(screen.getByText('25')).toBeInTheDocument(); // Active Agents
        expect(screen.getByText('12')).toBeInTheDocument(); // Target Systems
        expect(screen.getByText('1500')).toBeInTheDocument(); // Inferences Today
        expect(screen.getByText('96.5%')).toBeInTheDocument(); // Success Rate
      });
    });

    it('should handle loading errors gracefully', async () => {
      mockAnalyticsService.getDashboardStats.mockRejectedValue(new Error('Network error'));
      
      render(<AnalyticsDashboard isOpen={true} onClose={jest.fn()} />);
      
      // Should not crash and should still render the component
      expect(screen.getByText('Analytics Dashboard')).toBeInTheDocument();
    });
  });

  describe('Tab Navigation', () => {
    it('should switch to performance tab', async () => {
      render(<AnalyticsDashboard isOpen={true} onClose={jest.fn()} />);
      
      fireEvent.click(screen.getByText('Performance'));
      
      await waitFor(() => {
        expect(screen.getByText('45.2%')).toBeInTheDocument(); // CPU Usage
        expect(screen.getByText('67.8%')).toBeInTheDocument(); // Memory Usage
        expect(screen.getByText('25ms')).toBeInTheDocument(); // Network Latency
      });
    });

    it('should switch to usage tab', async () => {
      render(<AnalyticsDashboard isOpen={true} onClose={jest.fn()} />);
      
      fireEvent.click(screen.getByText('Usage'));
      
      await waitFor(() => {
        expect(screen.getByText('1250')).toBeInTheDocument(); // Total Sessions
        expect(screen.getByText('28m')).toBeInTheDocument(); // Avg Session Duration
        expect(screen.getByText('87%')).toBeInTheDocument(); // User Engagement
      });
    });

    it('should switch to agents tab', async () => {
      render(<AnalyticsDashboard isOpen={true} onClose={jest.fn()} />);
      
      fireEvent.click(screen.getByText('Agents'));
      
      await waitFor(() => {
        expect(screen.getByText('45')).toBeInTheDocument(); // Total Agents
        expect(screen.getByText('32')).toBeInTheDocument(); // Active Agents
        expect(screen.getByText('94.5%')).toBeInTheDocument(); // Deployment Success
      });
    });

    it('should highlight active tab', async () => {
      render(<AnalyticsDashboard isOpen={true} onClose={jest.fn()} />);
      
      const overviewTab = screen.getByText('Overview').closest('button');
      const performanceTab = screen.getByText('Performance').closest('button');
      
      // Overview should be active by default
      expect(overviewTab).toHaveClass('text-blue-400');
      
      fireEvent.click(screen.getByText('Performance'));
      
      await waitFor(() => {
        expect(performanceTab).toHaveClass('text-blue-400');
      });
    });
  });

  describe('Refresh Functionality', () => {
    it('should refresh data when refresh button is clicked', async () => {
      render(<AnalyticsDashboard isOpen={true} onClose={jest.fn()} />);
      
      // Wait for initial load
      await waitFor(() => {
        expect(mockAnalyticsService.getDashboardStats).toHaveBeenCalledTimes(1);
      });
      
      const refreshButton = screen.getByRole('button', { name: /refresh/i });
      fireEvent.click(refreshButton);
      
      await waitFor(() => {
        expect(mockAnalyticsService.getDashboardStats).toHaveBeenCalledTimes(2);
      });
    });

    it('should show loading state during refresh', async () => {
      render(<AnalyticsDashboard isOpen={true} onClose={jest.fn()} />);
      
      const refreshButton = screen.getByRole('button', { name: /refresh/i });
      fireEvent.click(refreshButton);
      
      // Should show loading state (disabled button)
      expect(refreshButton).toBeDisabled();
    });

    it('should auto-refresh every 30 seconds', async () => {
      render(<AnalyticsDashboard isOpen={true} onClose={jest.fn()} />);
      
      // Wait for initial load
      await waitFor(() => {
        expect(mockAnalyticsService.getDashboardStats).toHaveBeenCalledTimes(1);
      });
      
      // Fast-forward 30 seconds
      jest.advanceTimersByTime(30000);
      
      await waitFor(() => {
        expect(mockAnalyticsService.getDashboardStats).toHaveBeenCalledTimes(2);
      });
    });
  });

  describe('Export Functionality', () => {
    it('should export data as JSON', async () => {
      mockAnalyticsService.exportData.mockResolvedValue('{"data": "test"}');
      
      render(<AnalyticsDashboard isOpen={true} onClose={jest.fn()} />);
      
      const exportButton = screen.getByRole('button', { name: /download/i });
      fireEvent.click(exportButton);
      
      await waitFor(() => {
        expect(mockAnalyticsService.exportData).toHaveBeenCalledWith('json');
        expect(mockAnchor.download).toBe('knirv-analytics.json');
        expect(mockAnchor.click).toHaveBeenCalled();
      });
    });

    it('should handle export errors gracefully', async () => {
      mockAnalyticsService.exportData.mockRejectedValue(new Error('Export failed'));
      
      render(<AnalyticsDashboard isOpen={true} onClose={jest.fn()} />);
      
      const exportButton = screen.getByRole('button', { name: /download/i });
      fireEvent.click(exportButton);
      
      // Should not crash
      expect(screen.getByText('Analytics Dashboard')).toBeInTheDocument();
    });
  });

  describe('Close Functionality', () => {
    it('should call onClose when close button is clicked', () => {
      const onCloseMock = jest.fn();
      render(<AnalyticsDashboard isOpen={true} onClose={onCloseMock} />);
      
      const closeButton = screen.getByText('×');
      fireEvent.click(closeButton);
      
      expect(onCloseMock).toHaveBeenCalled();
    });

    it('should clear auto-refresh interval when closed', async () => {
      const { rerender } = render(<AnalyticsDashboard isOpen={true} onClose={jest.fn()} />);
      
      // Wait for initial load
      await waitFor(() => {
        expect(mockAnalyticsService.getDashboardStats).toHaveBeenCalledTimes(1);
      });
      
      // Close the component
      rerender(<AnalyticsDashboard isOpen={false} onClose={jest.fn()} />);
      
      // Fast-forward time - should not trigger more calls
      jest.advanceTimersByTime(30000);
      
      expect(mockAnalyticsService.getDashboardStats).toHaveBeenCalledTimes(1);
    });
  });

  describe('MetricCard Component', () => {
    it('should display metric values correctly', async () => {
      render(<AnalyticsDashboard isOpen={true} onClose={jest.fn()} />);
      
      await waitFor(() => {
        // Check for metric cards in overview
        expect(screen.getByText('Active Agents')).toBeInTheDocument();
        expect(screen.getByText('Target Systems')).toBeInTheDocument();
        expect(screen.getByText('Inferences Today')).toBeInTheDocument();
        expect(screen.getByText('Success Rate')).toBeInTheDocument();
      });
    });

    it('should show change indicators when available', async () => {
      render(<AnalyticsDashboard isOpen={true} onClose={jest.fn()} />);
      
      await waitFor(() => {
        expect(screen.getByText('+15%')).toBeInTheDocument();
        expect(screen.getByText('+2')).toBeInTheDocument();
        expect(screen.getByText('+250')).toBeInTheDocument();
        expect(screen.getByText('+1.2%')).toBeInTheDocument();
      });
    });

    it('should apply correct color classes based on metric type', async () => {
      render(<AnalyticsDashboard isOpen={true} onClose={jest.fn()} />);
      
      fireEvent.click(screen.getByText('Performance'));
      
      await waitFor(() => {
        // Error rate should have red color when > 5%
        const errorRateCard = screen.getByText('Error Rate').closest('div');
        expect(errorRateCard).toHaveClass('bg-green-500/20'); // 2.1% is good
      });
    });
  });

  describe('Usage Analytics Display', () => {
    it('should display most used features', async () => {
      render(<AnalyticsDashboard isOpen={true} onClose={jest.fn()} />);
      
      fireEvent.click(screen.getByText('Usage'));
      
      await waitFor(() => {
        expect(screen.getByText('Most Used Features')).toBeInTheDocument();
        expect(screen.getByText('Agent Management')).toBeInTheDocument();
        expect(screen.getByText('Cognitive Shell')).toBeInTheDocument();
        expect(screen.getByText('Wallet Operations')).toBeInTheDocument();
        expect(screen.getByText('85')).toBeInTheDocument();
        expect(screen.getByText('72')).toBeInTheDocument();
        expect(screen.getByText('58')).toBeInTheDocument();
      });
    });
  });

  describe('Accessibility', () => {
    it('should have proper ARIA labels', () => {
      render(<AnalyticsDashboard isOpen={true} onClose={jest.fn()} />);
      
      expect(screen.getByRole('button', { name: /refresh/i })).toBeInTheDocument();
      expect(screen.getByRole('button', { name: /download/i })).toBeInTheDocument();
    });

    it('should support keyboard navigation', () => {
      render(<AnalyticsDashboard isOpen={true} onClose={jest.fn()} />);
      
      const overviewTab = screen.getByText('Overview').closest('button');
      const performanceTab = screen.getByText('Performance').closest('button');
      
      expect(overviewTab).toBeInTheDocument();
      expect(performanceTab).toBeInTheDocument();
      
      // Tabs should be focusable
      overviewTab?.focus();
      expect(document.activeElement).toBe(overviewTab);
    });
  });

  describe('Responsive Design', () => {
    it('should render properly on different screen sizes', () => {
      render(<AnalyticsDashboard isOpen={true} onClose={jest.fn()} />);
      
      // Check for responsive grid classes
      const container = screen.getByText('Analytics Dashboard').closest('div');
      expect(container).toHaveClass('max-w-6xl');
    });
  });
});
