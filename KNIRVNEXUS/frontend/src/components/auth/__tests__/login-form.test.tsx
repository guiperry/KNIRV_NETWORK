import React from 'react';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import '@testing-library/jest-dom';
import { LoginForm } from '../login-form';

// Mock the auth context
jest.mock('@/lib/auth-context', () => ({
  useAuth: jest.fn(),
}));

// Mock UI components
jest.mock('@/components/ui/button', () => ({
  Button: ({ children, onClick, disabled, ...props }: any) => (
    <button onClick={onClick} disabled={disabled} {...props}>{children}</button>
  ),
}));

jest.mock('@/components/ui/input', () => ({
  Input: (props: any) => <input {...props} />,
}));

jest.mock('@/components/ui/label', () => ({
  Label: ({ children, ...props }: any) => <label {...props}>{children}</label>,
}));

jest.mock('@/components/ui/card', () => ({
  Card: ({ children, ...props }: any) => <div {...props}>{children}</div>,
  CardContent: ({ children, ...props }: any) => <div {...props}>{children}</div>,
  CardDescription: ({ children, ...props }: any) => <div {...props}>{children}</div>,
  CardHeader: ({ children, ...props }: any) => <div {...props}>{children}</div>,
  CardTitle: ({ children, ...props }: any) => <div {...props}>{children}</div>,
}));

jest.mock('@/components/ui/alert', () => ({
  Alert: ({ children, ...props }: any) => <div {...props}>{children}</div>,
  AlertDescription: ({ children, ...props }: any) => <div {...props}>{children}</div>,
}));

jest.mock('@/components/ui/badge', () => ({
  Badge: ({ children, ...props }: any) => <span {...props}>{children}</span>,
}));

// Mock lucide-react icons
jest.mock('lucide-react', () => ({
  Shield: (props: any) => <div data-testid="Shield-icon" {...props} />,
  Key: (props: any) => <div data-testid="Key-icon" {...props} />,
  User: (props: any) => <div data-testid="User-icon" {...props} />,
  Eye: (props: any) => <div data-testid="Eye-icon" {...props} />,
}));

const mockUseAuth = require('@/lib/auth-context').useAuth as jest.MockedFunction<typeof import('@/lib/auth-context').useAuth>;

describe('LoginForm', () => {
  const mockLogin = jest.fn();
  const mockLoginWithCredentials = jest.fn();

  beforeEach(() => {
    jest.clearAllMocks();

    mockUseAuth.mockReturnValue({
      user: null,
      login: mockLogin,
      loginWithCredentials: mockLoginWithCredentials,
      logout: jest.fn(),
      isLoading: false,
      error: null,
    });
  });

  it('should render login form with all fields', () => {
    render(<LoginForm />);

    expect(screen.getByText('KNIRV NEXUS')).toBeInTheDocument();
    expect(screen.getByText('Deterministic Validation Environment')).toBeInTheDocument();
    expect(screen.getByText('Username & Password')).toBeInTheDocument();
    expect(screen.getByText('Access Token')).toBeInTheDocument();
    expect(screen.getByLabelText(/username/i)).toBeInTheDocument();
    expect(screen.getByLabelText(/password/i)).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /login/i })).toBeInTheDocument();
  });

  it('should handle credentials login', async () => {
    const user = userEvent.setup();

    mockLoginWithCredentials.mockResolvedValue(true);

    render(<LoginForm />);

    await user.type(screen.getByLabelText(/username/i), 'testuser');
    await user.type(screen.getByLabelText(/password/i), 'password123');
    await user.click(screen.getByRole('button', { name: /login/i }));

    await waitFor(() => {
      expect(mockLoginWithCredentials).toHaveBeenCalledWith('testuser', 'password123');
    });
  });

  it('should disable login button when credentials are empty', () => {
    render(<LoginForm />);

    const loginButton = screen.getByRole('button', { name: /login/i });
    expect(loginButton).toBeDisabled();
  });

  it('should switch to token mode', async () => {
    const user = userEvent.setup();

    render(<LoginForm />);

    // Click on Access Token tab
    await user.click(screen.getByText('Access Token'));

    expect(screen.getByLabelText(/token/i)).toBeInTheDocument();
    expect(screen.queryByLabelText(/username/i)).not.toBeInTheDocument();
    expect(screen.queryByLabelText(/password/i)).not.toBeInTheDocument();
  });

  it('should handle token login', async () => {
    const user = userEvent.setup();

    mockLogin.mockResolvedValue(true);

    render(<LoginForm />);

    // Switch to token mode
    await user.click(screen.getByText('Access Token'));

    await user.type(screen.getByLabelText(/token/i), 'test-token-123');
    await user.click(screen.getByRole('button', { name: /login/i }));

    await waitFor(() => {
      expect(mockLogin).toHaveBeenCalledWith('test-token-123');
    });
  });

  it('should disable login button when token is empty', async () => {
    const user = userEvent.setup();

    render(<LoginForm />);

    // Switch to token mode
    await user.click(screen.getByText('Access Token'));

    const loginButton = screen.getByRole('button', { name: /login/i });
    expect(loginButton).toBeDisabled();
  });

  it('should handle login failure', async () => {
    const user = userEvent.setup();

    mockLoginWithCredentials.mockResolvedValue(false);

    render(<LoginForm />);

    await user.type(screen.getByLabelText(/username/i), 'testuser');
    await user.type(screen.getByLabelText(/password/i), 'wrongpassword');
    await user.click(screen.getByRole('button', { name: /login/i }));

    await waitFor(() => {
      expect(screen.getByText(/invalid username or password/i)).toBeInTheDocument();
    });
  });

  it('should handle token login failure', async () => {
    const user = userEvent.setup();

    mockLogin.mockResolvedValue(false);

    render(<LoginForm />);

    // Switch to token mode
    await user.click(screen.getByText('Access Token'));

    await user.type(screen.getByLabelText(/token/i), 'invalid-token');
    await user.click(screen.getByRole('button', { name: /login/i }));

    await waitFor(() => {
      expect(screen.getByText(/invalid token/i)).toBeInTheDocument();
    });
  });

  it('should show loading state during auth loading', () => {
    mockUseAuth.mockReturnValue({
      user: null,
      login: mockLogin,
      loginWithCredentials: mockLoginWithCredentials,
      logout: jest.fn(),
      isLoading: true,
      error: null,
    });

    render(<LoginForm />);

    // Should show loading spinner
    const spinner = document.querySelector('.animate-spin');
    expect(spinner).toBeInTheDocument();
  });
});
