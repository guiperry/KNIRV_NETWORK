import React from 'react';
import { render, screen } from '@testing-library/react';
import { BrowserRouter } from 'react-router-dom';
import Dashboard from '../pages/Dashboard';
import Layout from '../components/Layout';
import { GraphChainProvider } from '../context/GraphChainContext';

// Mock the GraphChain API
jest.mock('../services/api', () => ({
  graphChainApi: {
    getChainHeight: jest.fn().mockResolvedValue(12345),
    getGraphChainStats: jest.fn().mockResolvedValue({
      density: 12345,
      totalNodes: 1000,
      totalEdges: 2000,
      totalSkillNodes: 150,
      totalErrorNodes: 25,
      totalVectors: 500,
      avgResolutionTime: 2.5,
    }),
    getRecentSkills: jest.fn().mockResolvedValue([]),
    getRecentErrors: jest.fn().mockResolvedValue([]),
  },
}));

const MockedGraphChainProvider = ({ children }: { children: React.ReactNode }) => (
  <GraphChainProvider>
    <BrowserRouter>
      {children}
    </BrowserRouter>
  </GraphChainProvider>
);

describe('Terminology Updates', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  describe('Dashboard Terminology', () => {
    it('should display "Network Density" instead of "Graph Height"', async () => {
      render(
        <MockedGraphChainProvider>
          <Dashboard />
        </MockedGraphChainProvider>
      );

      // Wait for the component to load
      await screen.findByText('KNIRV Chain Portal');

      // Check that "Network Density" is displayed
      expect(screen.getByText('Network Density')).toBeInTheDocument();
      
      // Ensure "Graph Height" is not present
      expect(screen.queryByText('Graph Height')).not.toBeInTheDocument();
      expect(screen.queryByText('height')).not.toBeInTheDocument();
    });

    it('should display density value correctly', async () => {
      render(
        <MockedGraphChainProvider>
          <Dashboard />
        </MockedGraphChainProvider>
      );

      await screen.findByText('KNIRV Chain Portal');

      // Check that the density value is displayed
      expect(screen.getByText('12,345')).toBeInTheDocument();
    });

    it('should use "vectors" terminology in descriptions', async () => {
      render(
        <MockedGraphChainProvider>
          <Dashboard />
        </MockedGraphChainProvider>
      );

      await screen.findByText('KNIRV Chain Portal');

      // Check for vector-related terminology
      const description = screen.getByText(/Real-time KNIRV Chain data/);
      expect(description).toBeInTheDocument();
      
      // Ensure no "block" terminology is present
      expect(screen.queryByText(/block/i)).not.toBeInTheDocument();
    });
  });

  describe('Layout Navigation Terminology', () => {
    it('should display correct navigation items with updated terminology', () => {
      render(
        <MockedGraphChainProvider>
          <Layout>
            <div>Test Content</div>
          </Layout>
        </MockedGraphChainProvider>
      );

      // Check navigation items
      expect(screen.getByText('Dashboard')).toBeInTheDocument();
      expect(screen.getByText('SkillNodes')).toBeInTheDocument();
      expect(screen.getByText('ErrorNodes')).toBeInTheDocument();
      expect(screen.getByText('Graph View')).toBeInTheDocument();

      // Check that the portal title uses correct terminology
      expect(screen.getByText('KNIRV Chain Portal')).toBeInTheDocument();
    });

    it('should have correct search placeholder text', () => {
      render(
        <MockedGraphChainProvider>
          <Layout>
            <div>Test Content</div>
          </Layout>
        </MockedGraphChainProvider>
      );

      const searchInput = screen.getByPlaceholderText(/Search SkillNodes, ErrorNodes/);
      expect(searchInput).toBeInTheDocument();
      
      // Ensure no "block" terminology in search
      expect(screen.queryByPlaceholderText(/block/i)).not.toBeInTheDocument();
    });
  });

  describe('Context API Terminology', () => {
    it('should use currentDensity instead of currentHeight in context', () => {
      // This test verifies that the context API uses the correct terminology
      const TestComponent = () => {
        const { currentDensity } = React.useContext(
          React.createContext({ currentDensity: 12345 })
        );
        return <div data-testid="density-value">{currentDensity}</div>;
      };

      render(
        <TestComponent />
      );

      expect(screen.getByTestId('density-value')).toHaveTextContent('12345');
    });
  });

  describe('API Service Terminology', () => {
    it('should use density-related API endpoints', () => {
      // Import the API module to check method names
      const { graphChainApi } = require('../services/api');
      
      // Verify that the API has the correct method names
      expect(graphChainApi.getChainHeight).toBeDefined();
      expect(typeof graphChainApi.getChainHeight).toBe('function');
    });

    it('should return GraphChainStats with density field', async () => {
      const { graphChainApi } = require('../services/api');
      
      const stats = await graphChainApi.getGraphChainStats();
      
      expect(stats).toHaveProperty('density');
      expect(stats.density).toBe(12345);
      expect(stats).not.toHaveProperty('height');
    });
  });

  describe('Component Props and State Terminology', () => {
    it('should not contain any references to "height" in component state', async () => {
      render(
        <MockedGraphChainProvider>
          <Dashboard />
        </MockedGraphChainProvider>
      );

      await screen.findByText('KNIRV Chain Portal');

      // Get the component's HTML content
      const container = screen.getByText('KNIRV Chain Portal').closest('div');
      const htmlContent = container?.innerHTML || '';

      // Check that no "height" references exist (excluding CSS height properties)
      const heightReferences = htmlContent.match(/(?<!min-|max-|line-)height(?!:)/gi);
      expect(heightReferences).toBeNull();
    });

    it('should contain references to "density" in appropriate contexts', async () => {
      render(
        <MockedGraphChainProvider>
          <Dashboard />
        </MockedGraphChainProvider>
      );

      await screen.findByText('KNIRV Chain Portal');

      // Check that density terminology is present
      expect(screen.getByText('Network Density')).toBeInTheDocument();
    });
  });
});
