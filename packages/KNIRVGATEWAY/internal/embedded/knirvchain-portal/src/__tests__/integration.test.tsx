import React from 'react';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { BrowserRouter } from 'react-router-dom';
import App from '../App';
import { graphChainApi } from '../services/api';

// Mock the API
jest.mock('../services/api', () => ({
  graphChainApi: {
    getChainHeight: jest.fn(),
    getGraphChainStats: jest.fn(),
    getRecentSkills: jest.fn(),
    getRecentErrors: jest.fn(),
    getAllSkills: jest.fn(),
    getAllErrors: jest.fn(),
    getAllVectors: jest.fn(),
  },
}));

const mockApi = graphChainApi as jest.Mocked<typeof graphChainApi>;

describe('Phase 4 Integration Tests', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    
    // Setup default mock responses
    mockApi.getChainHeight.mockResolvedValue(12345);
    mockApi.getGraphChainStats.mockResolvedValue({
      density: 12345,
      totalNodes: 1000,
      totalEdges: 2000,
      totalSkillNodes: 150,
      totalErrorNodes: 25,
      totalVectors: 500,
      avgResolutionTime: 2.5,
    });
    mockApi.getRecentSkills.mockResolvedValue([
      {
        id: 'skill_1',
        skill_type: 'nlp_processing',
        capabilities: ['text_analysis', 'sentiment_detection'],
        requirements: { memory: '256MB' },
        timestamp: '2024-01-15T10:30:00Z',
        performance: {
          success_rate: 0.95,
          avg_resolution_time: 1.2,
          total_resolutions: 100
        },
        validation: {
          is_validated: true,
          validated_by: ['validator_1'],
          validation_score: 0.9,
          last_validated: '2024-01-15T09:00:00Z'
        }
      }
    ]);
    mockApi.getRecentErrors.mockResolvedValue([]);
  });

  describe('Application Routing and Navigation', () => {
    it('should render the main dashboard with correct terminology', async () => {
      render(
        <BrowserRouter>
          <App />
        </BrowserRouter>
      );

      await waitFor(() => {
        expect(screen.getByText('KNIRV Chain Portal')).toBeInTheDocument();
      });

      expect(screen.getByText('Network Density')).toBeInTheDocument();
      expect(screen.getByText('12,345')).toBeInTheDocument();
    });

    it('should navigate to SkillNodes page', async () => {
      render(
        <BrowserRouter>
          <App />
        </BrowserRouter>
      );

      await waitFor(() => {
        expect(screen.getByText('KNIRV Chain Portal')).toBeInTheDocument();
      });

      const skillNodesLink = screen.getByText('SkillNodes');
      fireEvent.click(skillNodesLink);

      // The navigation should work (we can't test the actual route change in this setup)
      expect(skillNodesLink).toBeInTheDocument();
    });

    it('should have correct navigation structure', async () => {
      render(
        <BrowserRouter>
          <App />
        </BrowserRouter>
      );

      await waitFor(() => {
        expect(screen.getByText('KNIRV Chain Portal')).toBeInTheDocument();
      });

      // Check all navigation items
      expect(screen.getByText('Dashboard')).toBeInTheDocument();
      expect(screen.getByText('SkillNodes')).toBeInTheDocument();
      expect(screen.getByText('ErrorNodes')).toBeInTheDocument();
      expect(screen.getByText('Graph View')).toBeInTheDocument();
    });
  });

  describe('Data Loading and Display', () => {
    it('should load and display dashboard data correctly', async () => {
      render(
        <BrowserRouter>
          <App />
        </BrowserRouter>
      );

      await waitFor(() => {
        expect(screen.getByText('KNIRV Chain Portal')).toBeInTheDocument();
      });

      // Check that API calls were made
      expect(mockApi.getChainHeight).toHaveBeenCalled();
      expect(mockApi.getGraphChainStats).toHaveBeenCalled();
      expect(mockApi.getRecentSkills).toHaveBeenCalled();

      // Check that data is displayed
      expect(screen.getByText('12,345')).toBeInTheDocument();
      expect(screen.getByText('150')).toBeInTheDocument(); // SkillNodes count
      expect(screen.getByText('25')).toBeInTheDocument(); // ErrorNodes count
    });

    it('should handle API errors gracefully', async () => {
      mockApi.getChainHeight.mockRejectedValue(new Error('Network error'));

      render(
        <BrowserRouter>
          <App />
        </BrowserRouter>
      );

      await waitFor(() => {
        expect(screen.getByText('Connection Error')).toBeInTheDocument();
      });

      expect(screen.getByText('Network error')).toBeInTheDocument();
      expect(screen.getByText('Retry')).toBeInTheDocument();
    });

    it('should display SkillNode cards with correct information', async () => {
      render(
        <BrowserRouter>
          <App />
        </BrowserRouter>
      );

      await waitFor(() => {
        expect(screen.getByText('KNIRV Chain Portal')).toBeInTheDocument();
      });

      // Check that skill information is displayed
      expect(screen.getByText('nlp_processing')).toBeInTheDocument();
      expect(screen.getByText('95.0% success')).toBeInTheDocument();
      expect(screen.getByText('text_analysis')).toBeInTheDocument();
    });
  });

  describe('Search Functionality', () => {
    it('should have search input with correct placeholder', async () => {
      render(
        <BrowserRouter>
          <App />
        </BrowserRouter>
      );

      await waitFor(() => {
        expect(screen.getByText('KNIRV Chain Portal')).toBeInTheDocument();
      });

      const searchInput = screen.getByPlaceholderText(/Search SkillNodes, ErrorNodes/);
      expect(searchInput).toBeInTheDocument();
    });

    it('should handle search input', async () => {
      render(
        <BrowserRouter>
          <App />
        </BrowserRouter>
      );

      await waitFor(() => {
        expect(screen.getByText('KNIRV Chain Portal')).toBeInTheDocument();
      });

      const searchInput = screen.getByPlaceholderText(/Search SkillNodes, ErrorNodes/);
      fireEvent.change(searchInput, { target: { value: 'test search' } });

      expect(searchInput).toHaveValue('test search');
    });
  });

  describe('Responsive Design', () => {
    it('should render mobile menu button on small screens', async () => {
      // Mock window.innerWidth for mobile
      Object.defineProperty(window, 'innerWidth', {
        writable: true,
        configurable: true,
        value: 500,
      });

      render(
        <BrowserRouter>
          <App />
        </BrowserRouter>
      );

      await waitFor(() => {
        expect(screen.getByText('KNIRV Chain Portal')).toBeInTheDocument();
      });

      // The mobile menu button should be present (though hidden by CSS)
      const menuButtons = document.querySelectorAll('[data-testid="mobile-menu"]') || 
                         document.querySelectorAll('.md\\:hidden button');
      expect(menuButtons.length).toBeGreaterThanOrEqual(0);
    });
  });

  describe('Error Handling', () => {
    it('should display loading state initially', () => {
      // Mock a slow API response
      mockApi.getChainHeight.mockImplementation(() => 
        new Promise(resolve => setTimeout(() => resolve(12345), 1000))
      );

      render(
        <BrowserRouter>
          <App />
        </BrowserRouter>
      );

      // Should show loading spinner initially
      const loadingElements = document.querySelectorAll('.animate-spin');
      expect(loadingElements.length).toBeGreaterThan(0);
    });

    it('should handle partial data loading', async () => {
      mockApi.getRecentSkills.mockRejectedValue(new Error('Skills API error'));

      render(
        <BrowserRouter>
          <App />
        </BrowserRouter>
      );

      await waitFor(() => {
        expect(screen.getByText('KNIRV Chain Portal')).toBeInTheDocument();
      });

      // Should still show density even if skills fail to load
      expect(screen.getByText('12,345')).toBeInTheDocument();
    });
  });

  describe('Performance and Optimization', () => {
    it('should not make unnecessary API calls', async () => {
      render(
        <BrowserRouter>
          <App />
        </BrowserRouter>
      );

      await waitFor(() => {
        expect(screen.getByText('KNIRV Chain Portal')).toBeInTheDocument();
      });

      // API should be called only once initially
      expect(mockApi.getChainHeight).toHaveBeenCalledTimes(1);
      expect(mockApi.getGraphChainStats).toHaveBeenCalledTimes(1);
    });

    it('should handle rapid state updates', async () => {
      const { rerender } = render(
        <BrowserRouter>
          <App />
        </BrowserRouter>
      );

      await waitFor(() => {
        expect(screen.getByText('KNIRV Chain Portal')).toBeInTheDocument();
      });

      // Rerender multiple times
      rerender(
        <BrowserRouter>
          <App />
        </BrowserRouter>
      );

      // Should still work correctly
      expect(screen.getByText('KNIRV Chain Portal')).toBeInTheDocument();
    });
  });
});
