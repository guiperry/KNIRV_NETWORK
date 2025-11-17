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

const mockUseAuth = require('@/lib/auth-context').useAuth as jest.MockedFunction<typeof import('@/lib/auth-context').useAuth>;
const mockUseRouter = require('next/navigation').useRouter as jest.MockedFunction<typeof import('next/navigation').useRouter>;

describe('RoleGuard', () => {
  const mockPush = jest.fn();
  const TestComponent = () => <div data-testid="protected-content">Protected Content</div>;

  beforeEach(() => {
    jest.clearAllMocks();
    
    mockUseRouter.mockReturnValue({
      push: mockPush,
      replace: jest.fn(),
    });
  });

  it('should render children when user has required role', () => {
    mockUseAuth.mockReturnValue({
      user: { id: '1', email: 'admin@example.com', role: 'admin', authenticated: true },
      login: jest.fn(),
      logout: jest.fn(),
      isLoading: false,
      error: null,
      hasPermission: jest.fn(),
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
      user: { id: '1', email: 'user@example.com', role: 'user', authenticated: true },
      login: jest.fn(),
      logout: jest.fn(),
      isLoading: false,
      error: null,
      hasPermission: jest.fn(),
      hasNodeAccess: jest.fn(),
    });

    render(
      <RoleGuard allowedRoles={['admin', 'user']}>
        <TestComponent />
      </RoleGuard>
    );

    expect(screen.getByTestId('protected-content')).toBeInTheDocument();
  });

  it('should show access denied when user does not have required role', () => {
    mockUseAuth.mockReturnValue({
      user: { id: '1', email: 'user@example.com', role: 'user', authenticated: true },
      login: jest.fn(),
      logout: jest.fn(),
      isLoading: false,
      error: null,
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
      logout: jest.fn(),
      isLoading: false,
      error: null,
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
      logout: jest.fn(),
      isLoading: true,
      error: null,
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
      user: { id: '1', email: 'user@example.com', role: 'user', authenticated: true },
      login: jest.fn(),
      logout: jest.fn(),
      isLoading: false,
      error: null,
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
      user: { id: '1', email: 'user@example.com', role: 'user', authenticated: false },
      login: jest.fn(),
      logout: jest.fn(),
      isLoading: false,
      error: null,
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
      user: { id: '1', email: 'user@example.com', role: 'user', authenticated: true },
      login: jest.fn(),
      logout: jest.fn(),
      isLoading: false,
      error: null,
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
      user: { id: '1', email: 'user@example.com', role: 'Admin', authenticated: true },
      login: jest.fn(),
      logout: jest.fn(),
      isLoading: false,
      error: null,
      hasPermission: jest.fn(),
      hasNodeAccess: jest.fn(),
    });

    render(
      <RoleGuard allowedRoles={['admin']}>
        <TestComponent />
      </RoleGuard>
    );

    expect(screen.getByText(/insufficient permissions/i)).toBeInTheDocument();
    expect(screen.queryByTestId('protected-content')).not.toBeInTheDocument();
  });
});
