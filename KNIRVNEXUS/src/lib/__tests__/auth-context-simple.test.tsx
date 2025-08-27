import React from 'react';
import { render, screen, waitFor } from '@testing-library/react';
import { AuthProvider, useAuth, ROLES } from '../auth-context';

// Mock the API module
jest.mock('../api', () => ({
  API_BASE_URL: 'http://localhost:8082',
  apiRequest: jest.fn(),
  getAuthHeaders: jest.fn(() => ({ 'Content-Type': 'application/json' })),
}));

// Simple test component to test auth context
const TestAuthComponent = () => {
  const { user, isLoading } = useAuth();

  return (
    <div>
      <div data-testid="loading">{isLoading ? 'loading' : 'not-loading'}</div>
      <div data-testid="user">{user ? user.user : 'no-user'}</div>
      <div data-testid="authenticated">{user?.authenticated ? 'authenticated' : 'not-authenticated'}</div>
    </div>
  );
};

describe('AuthContext', () => {
  beforeEach(() => {
    localStorage.clear();
    jest.clearAllMocks();
  });

  it('should provide auth context with initial state', async () => {
    render(
      <AuthProvider>
        <TestAuthComponent />
      </AuthProvider>
    );

    await waitFor(() => {
      expect(screen.getByTestId('loading')).toBeInTheDocument();
      expect(screen.getByTestId('user')).toBeInTheDocument();
      expect(screen.getByTestId('authenticated')).toBeInTheDocument();
    });
  });

  it('should test ROLES constant', () => {
    expect(ROLES.admin).toEqual({
      permissions: ['*:*'],
      nexus_access: ['dve:*', 'validation:*', 'system:*'],
      description: 'Full administrative access'
    });

    expect(ROLES.validator).toEqual({
      permissions: ['nexus:read', 'nexus:validate', 'nexus:update_assigned'],
      nexus_access: ['dve:read', 'validation:read', 'validation:execute', 'system:read'],
      description: 'Validator node operator with scoped access'
    });

    expect(ROLES.observer).toEqual({
      permissions: ['*:read'],
      nexus_access: ['dve:read', 'validation:read', 'system:read'],
      description: 'Read-only access to all services'
    });
  });

  it('should render without crashing', () => {
    render(
      <AuthProvider>
        <div>Test Content</div>
      </AuthProvider>
    );

    expect(screen.getByText('Test Content')).toBeInTheDocument();
  });

  it('should provide initial state', () => {
    render(
      <AuthProvider>
        <TestAuthComponent />
      </AuthProvider>
    );

    expect(screen.getByTestId('user')).toHaveTextContent('no-user');
    expect(screen.getByTestId('authenticated')).toHaveTextContent('not-authenticated');
  });

  it('should handle children prop', () => {
    render(
      <AuthProvider>
        <div data-testid="child">Child Component</div>
      </AuthProvider>
    );

    expect(screen.getByTestId('child')).toHaveTextContent('Child Component');
  });

  it('should provide auth functions', () => {
    const TestFunctionsComponent = () => {
      const { login, logout, hasPermission, hasNodeAccess } = useAuth();

      return (
        <div>
          <div data-testid="login">{typeof login}</div>
          <div data-testid="logout">{typeof logout}</div>
          <div data-testid="hasPermission">{typeof hasPermission}</div>
          <div data-testid="hasNodeAccess">{typeof hasNodeAccess}</div>
        </div>
      );
    };

    render(
      <AuthProvider>
        <TestFunctionsComponent />
      </AuthProvider>
    );

    expect(screen.getByTestId('login')).toHaveTextContent('function');
    expect(screen.getByTestId('logout')).toHaveTextContent('function');
    expect(screen.getByTestId('hasPermission')).toHaveTextContent('function');
    expect(screen.getByTestId('hasNodeAccess')).toHaveTextContent('function');
  });

  it('should handle multiple children', () => {
    render(
      <AuthProvider>
        <div data-testid="child1">Child 1</div>
        <div data-testid="child2">Child 2</div>
      </AuthProvider>
    );

    expect(screen.getByTestId('child1')).toHaveTextContent('Child 1');
    expect(screen.getByTestId('child2')).toHaveTextContent('Child 2');
  });

  it('should maintain context across re-renders', () => {
    const { rerender } = render(
      <AuthProvider>
        <TestAuthComponent />
      </AuthProvider>
    );

    const initialUser = screen.getByTestId('user').textContent;

    rerender(
      <AuthProvider>
        <TestAuthComponent />
      </AuthProvider>
    );

    expect(screen.getByTestId('user')).toHaveTextContent(initialUser || '');
  });

  it('should handle nested components', () => {
    const NestedComponent = () => {
      const { user } = useAuth();
      return <div data-testid="nested">{user ? 'has-user' : 'no-user'}</div>;
    };

    render(
      <AuthProvider>
        <div>
          <NestedComponent />
        </div>
      </AuthProvider>
    );

    expect(screen.getByTestId('nested')).toHaveTextContent('no-user');
  });

  it('should provide consistent context values', () => {
    const Component1 = () => {
      const { isLoading } = useAuth();
      return <div data-testid="comp1">{isLoading ? 'loading' : 'not-loading'}</div>;
    };

    const Component2 = () => {
      const { isLoading } = useAuth();
      return <div data-testid="comp2">{isLoading ? 'loading' : 'not-loading'}</div>;
    };

    render(
      <AuthProvider>
        <Component1 />
        <Component2 />
      </AuthProvider>
    );

    const comp1Text = screen.getByTestId('comp1').textContent;
    const comp2Text = screen.getByTestId('comp2').textContent;

    expect(comp1Text).toBe(comp2Text);
  });
});
