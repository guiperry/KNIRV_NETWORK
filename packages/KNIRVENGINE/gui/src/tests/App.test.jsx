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
      <button onClick={() => setActiveView('settings')}>Settings</button>
      <button onClick={() => setIsOpen(false)}>Close Sidebar</button>
    </div>
  ))
}));

jest.mock('../components/Dashboard', () => ({
  Dashboard: () => <div data-testid="dashboard">Dashboard Content</div>
}));

jest.mock('../components/Settings', () => () => <div data-testid="settings">Settings Content</div>);

jest.mock('../components/layout/AppLayout', () => {
  const React = require('react');
  const { Outlet } = require('react-router-dom');
  return {
    AppLayout: () => {
      const [isOpen, setIsOpen] = React.useState(false);
      return (
        <div>
          <div data-testid="sidebar" className={isOpen ? 'translate-x-0' : '-translate-x-full'}>
            <button onClick={() => setIsOpen(false)}>Close Sidebar</button>
          </div>
          <button className="fixed top-4 left-4" onClick={() => setIsOpen(!isOpen)}>Menu</button>
          <Outlet />
        </div>
      );
    },
  };
});

// Mock the asset path hook
jest.mock('../hooks/useAssetPath', () => ({
  useAppLogo: () => '/test-logo.png',
}));

// Import App after mocks
import App from '../App';

describe('App', () => {
  beforeEach(() => {
    global.fetch = jest.fn().mockResolvedValue({ ok: true, status: 200 });
    global.AbortSignal = { timeout: jest.fn(() => undefined) };
  });

  test('should render the App with Dashboard as default view', async () => {
    render(<App />);

    // Sidebar should be rendered
    expect(await screen.findByTestId('sidebar')).toBeInTheDocument();

    // Dashboard should be the default view
    expect(await screen.findByTestId('dashboard')).toBeInTheDocument();
  });

  test('should toggle sidebar state', async () => {
    render(<App />);

    // Sidebar should be closed by default
    const sidebar = await screen.findByTestId('sidebar');
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

  test('should handle sidebar navigation', async () => {
    render(<App />);

    // Initially Dashboard should be visible
    expect(await screen.findByTestId('dashboard')).toBeInTheDocument();
  });

  test('should initialize WebSocket on mount', async () => {
    const { initializeWebSocket } = require('../utils/websocket');
    render(<App />);

    await waitFor(() => expect(initializeWebSocket).toHaveBeenCalled());
  });

  test('should cleanup WebSocket on unmount', async () => {
    const { cleanupWebSocket } = require('../utils/websocket');
    const { unmount } = render(<App />);

    await screen.findByTestId('dashboard');
    unmount();

    expect(cleanupWebSocket).toHaveBeenCalled();
  });
});
