import React from 'react';
import { render, screen } from '@testing-library/react';
import '@testing-library/jest-dom';
import { RoleGuard } from '../role-guard';

// Mock the auth context
jest.mock('@/lib/auth-context', () => ({
  useAuth: jest.fn(),
}));

// Mock Next.js router
jest.mock('next/navigation', () => ({
  useRouter: jest.fn(() => ({
    push: jest.fn(),
    replace: jest.fn(),
  })),
}));

import { useAuth, AuthContextType } from '@/lib/auth-context';
import { useRouter } from 'next/navigation';

const mockUseAuth = useAuth as jest.MockedFunction<typeof useAuth>;
const mockUseRouter = useRouter as jest.MockedFunction<typeof useRouter>;

describe('RoleGuard', () => {
  const mockPush = jest.fn();
  const TestComponent = () => <div data-testid="protected-content">Protected Content</div>;

  beforeEach(() => {
    jest.clearAllMocks();
    
    mockUseRouter.mockReturnValue({
      push: mockPush,
      replace: jest.fn(),
      back: jest.fn(),
      forward: jest.fn(),
      refresh: jest.fn(),
      prefetch: jest.fn(),
    });
  });

  it('should render children when user has required role', () => {
            mockUseAuth.mockReturnValue({
              user: { user: 'admin@example.com', role: 'admin', authenticated: true, permissions: ['*:*'], nexus_access: ['dve:*', 'validation:*', 'system:*', 'fabric:*'] },
              login: jest.fn(),
              loginWithCredentials: jest.fn(),
              logout: jest.fn(),
              isLoading: false,          hasPermission: jest.fn(),
          hasNodeAccess: jest.fn(),
        });

    render(
      <RoleGuard allowedRoles={['admin']}>
        <TestComponent />
      </RoleGuard>
    );

    expect(screen.getByTestId('protected-content')).toBeInTheDocument();
  });

  it('should render children when user has one of multiple allowed roles', () => {
    mockUseAuth.mockReturnValue({
      user: { user: 'user@example.com', role: 'observer', authenticated: true, permissions: ['*:read'], nexus_access: ['dve:read', 'validation:read', 'system:read', 'fabric:read'] },
      login: jest.fn(),
      loginWithCredentials: jest.fn(),
      logout: jest.fn(),
      isLoading: false,

      hasPermission: jest.fn(),
      hasNodeAccess: jest.fn(),
    });

    render(
      <RoleGuard allowedRoles={['admin', 'observer']}>
        <TestComponent />
      </RoleGuard>
    );

    expect(screen.getByTestId('protected-content')).toBeInTheDocument();
  });

  it('should show access denied when user does not have required role', () => {
    mockUseAuth.mockReturnValue({
      user: { user: 'user@example.com', role: 'observer', authenticated: true, permissions: ['*:read'], nexus_access: ['dve:read', 'validation:read', 'system:read', 'fabric:read'] },
      login: jest.fn(),
      loginWithCredentials: jest.fn(),
      logout: jest.fn(),
      isLoading: false,

      hasPermission: jest.fn(),
      hasNodeAccess: jest.fn(),
    });

    render(
      <RoleGuard allowedRoles={['admin']}>
        <TestComponent />
      </RoleGuard>
    );

    expect(screen.getByText(/insufficient permissions/i)).toBeInTheDocument();
    expect(screen.getByText(/required role: admin/i)).toBeInTheDocument();
    expect(screen.queryByTestId('protected-content')).not.toBeInTheDocument();
  });

  it('should show authentication required when user is not authenticated', () => {
    mockUseAuth.mockReturnValue({
      user: null,
      login: jest.fn(),
      loginWithCredentials: jest.fn(),
      logout: jest.fn(),
      isLoading: false,

      hasPermission: jest.fn(),
      hasNodeAccess: jest.fn(),
    });

    render(
      <RoleGuard allowedRoles={['admin']}>
        <TestComponent />
      </RoleGuard>
    );

    expect(screen.getByText(/authentication required/i)).toBeInTheDocument();
    expect(screen.queryByTestId('protected-content')).not.toBeInTheDocument();
  });

  it('should show loading state when auth is loading', () => {
    mockUseAuth.mockReturnValue({
      user: null,
      login: jest.fn(),
      loginWithCredentials: jest.fn(),
      logout: jest.fn(),
      isLoading: true,

      hasPermission: jest.fn(),
      hasNodeAccess: jest.fn(),
    });

    render(
      <RoleGuard allowedRoles={['admin']}>
        <TestComponent />
      </RoleGuard>
    );

    expect(screen.getByTestId('loading-spinner')).toBeInTheDocument(); // Loading spinner
    expect(screen.queryByTestId('protected-content')).not.toBeInTheDocument();
  });

  it('should handle custom fallback component', () => {
    mockUseAuth.mockReturnValue({
      user: { user: 'user@example.com', role: 'observer', authenticated: true, permissions: ['*:read'], nexus_access: ['dve:read', 'validation:read', 'system:read', 'fabric:read'] },
      login: jest.fn(),
      loginWithCredentials: jest.fn(),
      logout: jest.fn(),
      isLoading: false,

      hasPermission: jest.fn(),
      hasNodeAccess: jest.fn(),
    });

    const CustomFallback = () => <div data-testid="custom-fallback">Custom Access Denied</div>;

    render(
      <RoleGuard allowedRoles={['admin']} fallback={<CustomFallback />}>
        <TestComponent />
      </RoleGuard>
    );

    expect(screen.getByTestId('custom-fallback')).toBeInTheDocument();
    expect(screen.queryByTestId('protected-content')).not.toBeInTheDocument();
  });

  it('should show authentication required for unauthenticated user', () => {
    mockUseAuth.mockReturnValue({
      user: { user: 'user@example.com', role: 'observer', authenticated: false, permissions: ['*:read'], nexus_access: ['dve:read', 'validation:read', 'system:read', 'fabric:read'] },
      login: jest.fn(),
      loginWithCredentials: jest.fn(),
      logout: jest.fn(),
      isLoading: false,

      hasPermission: jest.fn(),
      hasNodeAccess: jest.fn(),
    });

    render(
      <RoleGuard allowedRoles={['admin']}>
        <TestComponent />
      </RoleGuard>
    );

    expect(screen.getByText(/authentication required/i)).toBeInTheDocument();
  });

  it('should handle empty allowed roles array', () => {
    mockUseAuth.mockReturnValue({
      user: { user: 'user@example.com', role: 'observer', authenticated: true, permissions: ['*:read'], nexus_access: ['dve:read', 'validation:read', 'system:read', 'fabric:read'] },
      login: jest.fn(),
      loginWithCredentials: jest.fn(),
      logout: jest.fn(),
      isLoading: false,

      hasPermission: jest.fn(),
      hasNodeAccess: jest.fn(),
    });

    render(
      <RoleGuard allowedRoles={[]}>
        <TestComponent />
      </RoleGuard>
    );

    expect(screen.getByText(/insufficient permissions/i)).toBeInTheDocument();
    expect(screen.queryByTestId('protected-content')).not.toBeInTheDocument();
  });

  it('should be case sensitive for role matching', () => {
    mockUseAuth.mockReturnValue({
      user: { user: 'user@example.com', role: 'admin', authenticated: true, permissions: ['*:*'], nexus_access: ['dve:*', 'validation:*', 'system:*', 'fabric:*'] },
      login: jest.fn(),
      loginWithCredentials: jest.fn(),
      logout: jest.fn(),
      isLoading: false,

      hasPermission: jest.fn(),
      hasNodeAccess: jest.fn(),
    });

    render(
      <RoleGuard allowedRoles={['admin']}>
        <TestComponent />
      </RoleGuard>
    );

    expect(screen.queryByText(/insufficient permissions/i)).not.toBeInTheDocument();
    expect(screen.getByTestId('protected-content')).toBeInTheDocument();
  });
});
