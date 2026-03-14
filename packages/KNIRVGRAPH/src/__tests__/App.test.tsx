import React from 'react';
import { render, screen } from '@testing-library/react';
import '@testing-library/jest-dom';
import App from '../App';

// Mock React Router
interface MockRouterProps {
  children: React.ReactNode;
}

interface MockRouteProps {
  element: React.ReactNode;
}

jest.mock('react-router-dom', () => ({
  ...jest.requireActual('react-router-dom'),
  BrowserRouter: ({ children }: MockRouterProps) => <div data-testid="router">{children}</div>,
  Routes: ({ children }: MockRouterProps) => <div data-testid="routes">{children}</div>,
  Route: ({ element }: MockRouteProps) => <div data-testid="route">{element}</div>,
}));

// Mock the GraphChain context
jest.mock('../context/GraphChainContext', () => ({
  GraphChainProvider: ({ children }: MockRouterProps) => (
    <div data-testid="graphchain-provider">{children}</div>
  ),
  useGraphChain: () => ({
    currentHeight: 12345,
    isLoading: false,
    error: null,
    refreshData: jest.fn(),
  }),
}));

// Mock all page components
jest.mock('../pages/Dashboard', () => {
  return function MockDashboard() {
    return <div data-testid="dashboard-page">Dashboard</div>;
  };
});

jest.mock('../pages/SkillNodes', () => {
  return function MockSkillNodes() {
    return <div data-testid="skillnodes-page">SkillNodes</div>;
  };
});

jest.mock('../pages/SkillNodeDetails', () => {
  return function MockSkillNodeDetails() {
    return <div data-testid="skillnode-details-page">SkillNode Details</div>;
  };
});

jest.mock('../pages/ErrorNodes', () => {
  return function MockErrorNodes() {
    return <div data-testid="errornodes-page">ErrorNodes</div>;
  };
});

jest.mock('../pages/ErrorNodeDetails', () => {
  return function MockErrorNodeDetails() {
    return <div data-testid="errornode-details-page">ErrorNode Details</div>;
  };
});

jest.mock('../pages/GraphVisualization', () => {
  return function MockGraphVisualization() {
    return <div data-testid="graph-visualization-page">Graph Visualization</div>;
  };
});

jest.mock('../pages/Search', () => {
  return function MockSearch() {
    return <div data-testid="search-page">Search</div>;
  };
});

// Mock Layout component
jest.mock('../components/Layout', () => {
  return function MockLayout({ children }: MockRouterProps) {
    return (
      <div data-testid="layout">
        <header data-testid="header">KNIRV Graph</header>
        <main data-testid="main">{children}</main>
      </div>
    );
  };
});

describe('App Component', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  describe('Rendering', () => {
    it('should render without crashing', () => {
      render(<App />);
      expect(screen.getByTestId('layout')).toBeInTheDocument();
    });

    it('should render all main components', () => {
      render(<App />);

      expect(screen.getByTestId('router')).toBeInTheDocument();
      expect(screen.getByTestId('graphchain-provider')).toBeInTheDocument();
      expect(screen.getByTestId('layout')).toBeInTheDocument();
      expect(screen.getByTestId('routes')).toBeInTheDocument();
    });

    it('should display the application title', () => {
      render(<App />);
      expect(screen.getByText(/KNIRV Graph/i)).toBeInTheDocument();
    });

    it('should have proper layout structure', () => {
      const { container } = render(<App />);
      const appElement = container.firstChild as HTMLElement;

      expect(appElement).toHaveClass('min-h-screen');
    });
  });

  describe('Context Integration', () => {
    it('should provide GraphChain context to child components', () => {
      render(<App />);
      expect(screen.getByTestId('graphchain-provider')).toBeInTheDocument();
    });

    it('should render layout with proper structure', () => {
      render(<App />);
      expect(screen.getByTestId('header')).toBeInTheDocument();
      expect(screen.getByTestId('main')).toBeInTheDocument();
    });

    it('should set up routing correctly', () => {
      render(<App />);
      expect(screen.getByTestId('router')).toBeInTheDocument();
      expect(screen.getByTestId('routes')).toBeInTheDocument();
    });
  });

  describe('Component Structure', () => {
    it('should render the main app container with correct classes', () => {
      const { container } = render(<App />);
      const appDiv = container.firstChild as HTMLElement;

      expect(appDiv).toHaveClass('min-h-screen', 'bg-gray-900', 'text-white');
    });

    it('should contain the dashboard page by default', () => {
      render(<App />);
      expect(screen.getByTestId('dashboard-page')).toBeInTheDocument();
    });

    it('should wrap everything in the GraphChain provider', () => {
      render(<App />);
      const provider = screen.getByTestId('graphchain-provider');
      const layout = screen.getByTestId('layout');

      expect(provider).toContainElement(layout);
    });
  });

  describe('Error Handling', () => {
    it('should render without throwing errors', () => {
      expect(() => render(<App />)).not.toThrow();
    });

    it('should handle component unmounting gracefully', () => {
      const { unmount } = render(<App />);
      expect(() => unmount()).not.toThrow();
    });
  });

  describe('Accessibility', () => {
    it('should have proper semantic structure', () => {
      render(<App />);

      // Check for main content area
      const main = screen.getByTestId('main');
      expect(main).toBeInTheDocument();
    });

    it('should have proper heading structure', () => {
      render(<App />);

      // Check for header
      const header = screen.getByTestId('header');
      expect(header).toBeInTheDocument();
      expect(header).toHaveTextContent('KNIRV Graph');
    });
  });
});
