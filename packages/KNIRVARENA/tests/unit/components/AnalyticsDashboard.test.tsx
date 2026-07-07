/**
 * Simple AnalyticsDashboard Test
 * Test to isolate the AnalyticsDashboard rendering issue
 */

import * as React from 'react';
import { render, screen } from '@testing-library/react';
import '@testing-library/jest-dom';

// Mock the analytics service first
jest.mock('../../../src/services/AnalyticsService', () => ({
  analyticsService: {
    getDashboardStats: jest.fn().mockResolvedValue({
      activeAgents: 25,
      totalSkills: 12,
      totalTransactions: 1500,
      networkHealth: 'Good',
      lastUpdated: new Date()
    }),
    getPerformanceMetrics: jest.fn().mockResolvedValue({
      throughput: 45.2,
      errorRate: 2.1,
      uptime: 99.8,
      lastMeasured: new Date()
    }),
    getUsageAnalytics: jest.fn().mockResolvedValue({
      totalSessions: 850,
      averageSessionDuration: 12.5,
      popularFeatures: [
        { feature: 'Agent Management', usage: 85 },
        { feature: 'Skill Execution', usage: 72 }
      ],
      lastCalculated: new Date()
    }),
    getAgentAnalytics: jest.fn().mockResolvedValue({
      successRate: 94.5,
      averageExecutionTime: 250,
      resourceUtilization: 67.8,
      lastAnalyzed: new Date()
    }),
    exportData: jest.fn().mockResolvedValue('{"data": "test"}')
  }
}));

// Import the component after mocking
import AnalyticsDashboard from '../../../src/components/AnalyticsDashboard';

describe('AnalyticsDashboard Simple Test', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  it('should import the component without errors', () => {
    expect(AnalyticsDashboard).toBeDefined();
    expect(typeof AnalyticsDashboard).toBe('function');
  });

  it('should render when isOpen is false (should not show content)', () => {
    try {
      render(<AnalyticsDashboard isOpen={false} onClose={jest.fn()} />);
      
      // When isOpen is false, the component should not render its content
      expect(screen.queryByText('Analytics Dashboard')).not.toBeInTheDocument();
    } catch (error) {
      console.error('Render error with isOpen=false:', error);
      throw error;
    }
  });

  it('should render when isOpen is true', () => {
    try {
      render(<AnalyticsDashboard isOpen={true} onClose={jest.fn()} />);
      
      // When isOpen is true, the component should render its content
      expect(screen.getByText('Analytics Dashboard')).toBeInTheDocument();
    } catch (error) {
      console.error('Render error with isOpen=true:', error);
      throw error;
    }
  });
});
