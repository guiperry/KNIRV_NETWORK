import React from 'react';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import '@testing-library/jest-dom';
import { MemoryRouter } from 'react-router-dom';

// Mock the WebSocket utilities
jest.mock('../utils/websocket', () => ({
  initializeWebSocket: jest.fn(() => Promise.resolve()),
  cleanupWebSocket: jest.fn(),
}));

// Mock the AuthContext
jest.mock('../components/AuthContext', () => ({
  AuthProvider: ({ children }) => <div>{children}</div>,
  useAuth: () => ({
    user: { username: 'testuser', role: 'admin' },
    logout: jest.fn(),
    isAuthenticated: true,
  }),
}));

// Mock ProtectedRoute to just render children
jest.mock('../components/ProtectedRoute', () => {
  return function ProtectedRoute({ children }) {
    return <div>{children}</div>;
  };
});

// Mock LoginPage and UnauthorizedPage
jest.mock('../components/LoginPage', () => {
  return function LoginPage() {
    return <div data-testid="login-page">Login Page</div>;
  };
});

jest.mock('../components/UnauthorizedPage', () => {
  return function UnauthorizedPage() {
    return <div data-testid="unauthorized-page">Unauthorized Page</div>;
  };
});

// Mock the components
jest.mock('../components/Sidebar', () => ({
  Sidebar: jest.fn(({ activeView, setActiveView, isOpen, setIsOpen }) => (
    <div data-testid="sidebar" className={isOpen ? 'translate-x-0' : '-translate-x-full'}>
      <button onClick={() => setActiveView('dashboard')}>Dashboard</button>
      <button onClick={() => setActiveView('agents')}>Agents</button>
      <button onClick={() => setIsOpen(false)}>Close Sidebar</button>
    </div>
  ))
}));

jest.mock('../components/Dashboard', () => ({
  Dashboard: () => <div data-testid="dashboard">Dashboard Content</div>
}));

jest.mock('../components/AgentManager', () => ({
  AgentManager: () => <div data-testid="agent-manager">Agent Manager Content</div>
}));

jest.mock('../components/CapabilityStore', () => ({
  CapabilityStore: () => <div data-testid="capability-store">Capability Store Content</div>
}));

jest.mock('../components/TargetManager', () => ({
  TargetManager: () => <div data-testid="target-manager">Target Manager Content</div>
}));



jest.mock('../components/WorkflowOrchestrator', () => ({
  WorkflowOrchestrator: () => <div data-testid="workflow-orchestrator">Workflow Orchestrator Content</div>
}));

jest.mock('../components/Analytics', () => ({
  Analytics: () => <div data-testid="analytics">Analytics Content</div>
}));

jest.mock('../components/Settings', () => ({
  Settings: () => <div data-testid="settings">Settings Content</div>
}));

// Mock the asset path hook
jest.mock('../hooks/useAssetPath', () => ({
  useAppLogo: () => '/test-logo.png',
}));

// Import App after mocks
import App from '../App';

describe('App', () => {
  test('should render the App with Dashboard as default view', () => {
    render(<App />);

    // Sidebar should be rendered
    expect(screen.getByTestId('sidebar')).toBeInTheDocument();

    // Dashboard should be the default view
    expect(screen.getByTestId('dashboard')).toBeInTheDocument();
  });

  test('should toggle sidebar state', () => {
    render(<App />);

    // Sidebar should be closed by default
    const sidebar = screen.getByTestId('sidebar');
    expect(sidebar).toHaveClass('-translate-x-full');

    // Find the mobile menu button by its class
    const menuButton = document.querySelector('button.fixed.top-4.left-4');

    if (menuButton) {
      fireEvent.click(menuButton);

      // Sidebar should now be open
      expect(sidebar).toHaveClass('translate-x-0');
    }

    // Click the close button to close sidebar
    fireEvent.click(screen.getByText('Close Sidebar'));

    // Sidebar should be closed again
    expect(sidebar).toHaveClass('-translate-x-full');
  });

  test('should handle sidebar navigation', () => {
    render(<App />);

    // Initially Dashboard should be visible
    expect(screen.getByTestId('dashboard')).toBeInTheDocument();

    // Click Dashboard button in sidebar
    fireEvent.click(screen.getByText('Dashboard'));

    // Dashboard should still be visible
    expect(screen.getByTestId('dashboard')).toBeInTheDocument();
  });

  test('should initialize WebSocket on mount', () => {
    const { initializeWebSocket } = require('../utils/websocket');
    render(<App />);

    expect(initializeWebSocket).toHaveBeenCalled();
  });

  test('should cleanup WebSocket on unmount', () => {
    const { cleanupWebSocket } = require('../utils/websocket');
    const { unmount } = render(<App />);

    unmount();

    expect(cleanupWebSocket).toHaveBeenCalled();
  });
});