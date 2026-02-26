import React from 'react';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import '@testing-library/jest-dom';
import { UserProfile } from '../user-profile';
import type { AuthUser } from '@/lib/auth-context';

// Mock the auth context
jest.mock('@/lib/auth-context', () => ({
  useAuth: jest.fn(),
  ROLES: {
    admin: { description: 'Full system access' },
    validator: { description: 'Validation operations' },
    observer: { description: 'Read-only access' },
  },
}));

// Mock UI components
jest.mock('@/components/ui/button', () => ({
  Button: ({ children, onClick, ...props }: any) => (
    <button onClick={onClick} {...props}>{children}</button>
  ),
}));

jest.mock('@/components/ui/badge', () => ({
  Badge: ({ children, ...props }: any) => <span {...props}>{children}</span>,
}));

jest.mock('@/components/ui/card', () => ({
  Card: ({ children, ...props }: any) => <div {...props}>{children}</div>,
  CardContent: ({ children, ...props }: any) => <div {...props}>{children}</div>,
  CardDescription: ({ children, ...props }: any) => <div {...props}>{children}</div>,
  CardHeader: ({ children, ...props }: any) => <div {...props}>{children}</div>,
  CardTitle: ({ children, ...props }: any) => <div {...props}>{children}</div>,
}));

jest.mock('@/components/ui/dropdown-menu', () => ({
  DropdownMenu: ({ children }: any) => <div>{children}</div>,
  DropdownMenuContent: ({ children, ...props }: any) => <div {...props}>{children}</div>,
  DropdownMenuItem: ({ children, onClick, ...props }: any) => (
    <div onClick={onClick} {...props}>{children}</div>
  ),
  DropdownMenuLabel: ({ children, ...props }: any) => <div {...props}>{children}</div>,
  DropdownMenuSeparator: (props: any) => <hr {...props} />,
  DropdownMenuTrigger: ({ children, asChild, ...props }: any) =>
    asChild ? React.cloneElement(children, props) : <div {...props}>{children}</div>,
}));

jest.mock('@/components/ui/avatar', () => ({
  Avatar: ({ children, ...props }: any) => <div {...props}>{children}</div>,
  AvatarFallback: ({ children, ...props }: any) => <div {...props}>{children}</div>,
}));

// Mock lucide-react icons
jest.mock('lucide-react', () => ({
  User: (props: any) => <div data-testid="User-icon" {...props} />,
  Shield: (props: any) => <div data-testid="Shield-icon" {...props} />,
  Eye: (props: any) => <div data-testid="Eye-icon" {...props} />,
  LogOut: (props: any) => <div data-testid="LogOut-icon" {...props} />,
  Settings: (props: any) => <div data-testid="Settings-icon" {...props} />,
  Key: (props: any) => <div data-testid="Key-icon" {...props} />,
}));

import { useAuth } from '@/lib/auth-context';

const mockUseAuth = useAuth as jest.MockedFunction<typeof useAuth>;

describe('UserProfile', () => {
  const mockUser: AuthUser = {
    user: 'test-user',
    role: 'admin',
    authenticated: true,
    node_id: 'node-123',
    nexus_access: ['read', 'write', 'execute'],
    permissions: ['create', 'update', 'delete', 'manage'],
  };

  const mockLogout = jest.fn();

  beforeEach(() => {
    jest.clearAllMocks();

    mockUseAuth.mockReturnValue({
      user: mockUser,
      login: jest.fn(),
      loginWithCredentials: jest.fn(),
      logout: mockLogout,
      isLoading: false,
      hasPermission: jest.fn(),
      hasNodeAccess: jest.fn(),
    });
  });

  it('should render user profile dropdown trigger', () => {
    render(<UserProfile />);

    // Should render the dropdown trigger button with user initials
    const triggerButton = screen.getByRole('button');
    expect(triggerButton).toBeInTheDocument();

    // Should show user initials in avatar
    expect(screen.getByText('TU')).toBeInTheDocument(); // test-user -> TU
  });

  it('should show user information when dropdown is opened', async () => {
    const user = userEvent.setup();
    render(<UserProfile />);

    // Click the dropdown trigger
    const triggerButton = screen.getByRole('button');
    await user.click(triggerButton);

    // Should show user information
    expect(screen.getByText('test-user')).toBeInTheDocument();
    expect(screen.getByText('ADMIN')).toBeInTheDocument();
    expect(screen.getByText('Node: node-123')).toBeInTheDocument();
    expect(screen.getByText('Full system access')).toBeInTheDocument();
  });

  it('should show permissions in dropdown', async () => {
    const user = userEvent.setup();
    render(<UserProfile />);

    // Click the dropdown trigger
    const triggerButton = screen.getByRole('button');
    await user.click(triggerButton);

    // Should show NEXUS access permissions
    expect(screen.getByText('read')).toBeInTheDocument();
    expect(screen.getByText('write')).toBeInTheDocument();
    expect(screen.getByText('execute')).toBeInTheDocument();

    // Should show general permissions (first 3)
    expect(screen.getByText('create')).toBeInTheDocument();
    expect(screen.getByText('update')).toBeInTheDocument();
    expect(screen.getByText('delete')).toBeInTheDocument();
    expect(screen.getByText('+1 more')).toBeInTheDocument();
  });

  it('should handle logout when clicked', async () => {
    const user = userEvent.setup();
    render(<UserProfile />);

    // Click the dropdown trigger
    const triggerButton = screen.getByRole('button');
    await user.click(triggerButton);

    // Click logout
    const logoutButton = screen.getByText('Log out');
    await user.click(logoutButton);

    expect(mockLogout).toHaveBeenCalled();
  });

  it('should show different role badges for different roles', async () => {
    const user = userEvent.setup();

    // Test validator role
    mockUseAuth.mockReturnValue({
      user: { ...mockUser, role: 'validator' },
      login: jest.fn(),
      loginWithCredentials: jest.fn(),
      logout: mockLogout,
      isLoading: false,
      hasPermission: jest.fn(),
      hasNodeAccess: jest.fn(),
    });

    render(<UserProfile />);

    const triggerButton = screen.getByRole('button');
    await user.click(triggerButton);

    expect(screen.getByText('VALIDATOR')).toBeInTheDocument();
    expect(screen.getByText('Validation operations')).toBeInTheDocument();
  });

  it('should show observer role correctly', async () => {
    const user = userEvent.setup();

    mockUseAuth.mockReturnValue({
      user: { ...mockUser, role: 'observer' },
      login: jest.fn(),
      loginWithCredentials: jest.fn(),
      logout: mockLogout,
      isLoading: false,
      hasPermission: jest.fn(),
      hasNodeAccess: jest.fn(),
    });

    render(<UserProfile />);

    const triggerButton = screen.getByRole('button');
    await user.click(triggerButton);

    expect(screen.getByText('OBSERVER')).toBeInTheDocument();
    expect(screen.getByText('Read-only access')).toBeInTheDocument();
  });

  it('should not render when user is not authenticated', () => {
    mockUseAuth.mockReturnValue({
      user: { ...mockUser, authenticated: false },
      login: jest.fn(),
      loginWithCredentials: jest.fn(),
      logout: mockLogout,
      isLoading: false,
      hasPermission: jest.fn(),
      hasNodeAccess: jest.fn(),
    });

    const { container } = render(<UserProfile />);
    expect(container.firstChild).toBeNull();
  });

  it('should not render when user is null', () => {
    mockUseAuth.mockReturnValue({
      user: null,
      login: jest.fn(),
      loginWithCredentials: jest.fn(),
      logout: mockLogout,
      isLoading: false,
      hasPermission: jest.fn(),
      hasNodeAccess: jest.fn(),
    });

    const { container } = render(<UserProfile />);
    expect(container.firstChild).toBeNull();
  });

  it('should handle user without node_id', async () => {
    const user = userEvent.setup();

    mockUseAuth.mockReturnValue({
      user: { ...mockUser, node_id: undefined },
      login: jest.fn(),
      loginWithCredentials: jest.fn(),
      logout: mockLogout,
      isLoading: false,
      hasPermission: jest.fn(),
      hasNodeAccess: jest.fn(),
    });

    render(<UserProfile />);

    const triggerButton = screen.getByRole('button');
    await user.click(triggerButton);

    expect(screen.getByText('test-user')).toBeInTheDocument();
    expect(screen.queryByText(/Node:/)).not.toBeInTheDocument();
  });

  it('should handle user without general permissions', async () => {
    const user = userEvent.setup();

    mockUseAuth.mockReturnValue({
      user: { ...mockUser, permissions: [] },
      login: jest.fn(),
      loginWithCredentials: jest.fn(),
      logout: mockLogout,
      isLoading: false,
      hasPermission: jest.fn(),
      hasNodeAccess: jest.fn(),
    });

    render(<UserProfile />);

    const triggerButton = screen.getByRole('button');
    await user.click(triggerButton);

    expect(screen.getByText('NEXUS Access:')).toBeInTheDocument();
    expect(screen.queryByText('General:')).not.toBeInTheDocument();
  });
});
