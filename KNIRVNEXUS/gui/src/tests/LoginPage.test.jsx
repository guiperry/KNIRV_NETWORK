import React from 'react';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import '@testing-library/jest-dom';
import { BrowserRouter } from 'react-router-dom';
import LoginPage from '../components/LoginPage';
import * as AuthContext from '../components/AuthContext';

// Mock the useAuth hook
const mockUseAuth = jest.fn();
jest.spyOn(AuthContext, 'useAuth').mockImplementation(mockUseAuth);

describe('LoginPage', () => {
  const mockLogin = jest.fn();

  const renderLoginPage = () => {
    mockUseAuth.mockReturnValue({
      isAuthenticated: false,
      user: null,
      login: mockLogin,
      logout: jest.fn(),
      loading: false,
      register: jest.fn(),
      updateUser: jest.fn(),
      hasPermission: jest.fn(),
      refreshToken: jest.fn()
    });

    return render(
      <BrowserRouter>
        <LoginPage />
      </BrowserRouter>
    );
  };

  beforeEach(() => {
    jest.clearAllMocks();
  });

  test('should render login form', () => {
    renderLoginPage();

    expect(screen.getByText('KNIRV-NEXUS')).toBeInTheDocument();
    expect(screen.getByRole('heading', { name: /sign in/i })).toBeInTheDocument();
    expect(screen.getByLabelText(/username/i)).toBeInTheDocument();
    expect(screen.getByLabelText(/password/i)).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /sign in/i })).toBeInTheDocument();
  });

  test('should handle form submission with valid credentials', async () => {
    mockLogin.mockResolvedValue({ success: true });
    renderLoginPage();
    
    const usernameInput = screen.getByLabelText(/username/i);
    const passwordInput = screen.getByLabelText(/password/i);
    const submitButton = screen.getByRole('button', { name: /sign in/i });
    
    fireEvent.change(usernameInput, { target: { value: 'testuser' } });
    fireEvent.change(passwordInput, { target: { value: 'password123' } });
    fireEvent.click(submitButton);
    
    await waitFor(() => {
      expect(mockLogin).toHaveBeenCalledWith({ username: 'testuser', password: 'password123' });
    });
  });

  test('should show error message on failed login', async () => {
    mockLogin.mockResolvedValue({ success: false, error: 'Authentication failed' });
    renderLoginPage();
    
    const usernameInput = screen.getByLabelText(/username/i);
    const passwordInput = screen.getByLabelText(/password/i);
    const submitButton = screen.getByRole('button', { name: /sign in/i });
    
    fireEvent.change(usernameInput, { target: { value: 'wronguser' } });
    fireEvent.change(passwordInput, { target: { value: 'wrongpass' } });
    fireEvent.click(submitButton);
    
    await waitFor(() => {
      expect(screen.getByText(/authentication failed/i)).toBeInTheDocument();
    });
  });

  test('should disable submit button when form is submitting', async () => {
    mockLogin.mockImplementation(() => new Promise(resolve => setTimeout(() => resolve({ success: true }), 100)));
    renderLoginPage();

    const usernameInput = screen.getByLabelText(/username/i);
    const passwordInput = screen.getByLabelText(/password/i);
    const submitButton = screen.getByRole('button', { name: /sign in/i });

    // Fill in the form
    fireEvent.change(usernameInput, { target: { value: 'testuser' } });
    fireEvent.change(passwordInput, { target: { value: 'password123' } });

    fireEvent.click(submitButton);

    // Check that the button is disabled immediately after clicking
    expect(submitButton).toBeDisabled();

    // Wait for the loading to complete
    await waitFor(() => {
      expect(submitButton).not.toBeDisabled();
    });
  });
});
