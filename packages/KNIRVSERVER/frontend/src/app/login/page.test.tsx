import React from 'react';
import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import '@testing-library/jest-dom';
import LoginPage from './page';

const push = jest.fn();
const authLogin = jest.fn();

jest.mock('next/navigation', () => ({
  useRouter: () => ({ push }),
}));

jest.mock('@/lib/auth-context', () => ({
  clearStoredAuth: () => localStorage.clear(),
  getStoredAuthToken: () => localStorage.getItem('knirv_nexus_token') || localStorage.getItem('knirv_auth_token'),
  persistStoredAuth: (token: string, role: string, username: string) => {
    localStorage.setItem('knirv_nexus_token', token);
    localStorage.setItem('knirv_auth_token', token);
    localStorage.setItem('knirv_nexus_role', role);
    localStorage.setItem('knirv_nexus_user', username);
  },
  useAuth: () => ({ login: authLogin, user: null, isLoading: false }),
}));

describe('CLI device authorization', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    localStorage.clear();
    window.history.replaceState({}, '', '/login?cli_token=device-code');
  });

  it('approves the CLI without redirecting an already logged-in browser', async () => {
    localStorage.setItem('knirv_nexus_token', 'saved-jwt');
    const fetchMock = jest.fn()
      .mockResolvedValueOnce({ ok: true })
      .mockResolvedValueOnce({ ok: true });
    global.fetch = fetchMock as typeof fetch;

    render(<LoginPage />);

    expect(await screen.findByText('KNIRV CLI authorized. You may close this window.')).toBeInTheDocument();
    expect(fetchMock).toHaveBeenNthCalledWith(1, '/api/auth/me', {
      headers: { Authorization: 'Bearer saved-jwt' },
    });
    expect(fetchMock).toHaveBeenNthCalledWith(2, '/api/auth/device/approve', expect.objectContaining({
      method: 'POST',
      headers: expect.objectContaining({ Authorization: 'Bearer saved-jwt' }),
      body: JSON.stringify({ token: 'device-code' }),
    }));
    await waitFor(() => expect(push).not.toHaveBeenCalled());
  });
});

it('does not navigate with an unverified credential token', async () => {
  window.history.replaceState({}, '', '/login');
  authLogin.mockResolvedValueOnce(false);
  global.fetch = jest.fn().mockResolvedValueOnce({
    ok: true,
    json: async () => ({ token: 'unverified-token', role: 'observer' }),
  }) as typeof fetch;

  render(<LoginPage />);
  fireEvent.change(screen.getByPlaceholderText('Enter username'), { target: { value: 'operator' } });
  fireEvent.change(screen.getByPlaceholderText('Enter password'), { target: { value: 'password' } });
  fireEvent.click(screen.getByRole('button', { name: 'AUTHENTICATE' }));

  expect(await screen.findByText('Authentication could not be verified. Please try again.')).toBeInTheDocument();
  expect(push).not.toHaveBeenCalled();
  expect(localStorage.getItem('knirv_nexus_token')).toBeNull();
  expect(localStorage.getItem('knirv_auth_token')).toBeNull();
});
