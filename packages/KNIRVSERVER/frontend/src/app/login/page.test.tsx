import React from 'react';
import { render, screen, waitFor } from '@testing-library/react';
import '@testing-library/jest-dom';
import LoginPage from './page';

const push = jest.fn();
const authLogin = jest.fn();

jest.mock('next/navigation', () => ({
  useRouter: () => ({ push }),
}));

jest.mock('@/lib/auth-context', () => ({
  useAuth: () => ({ login: authLogin }),
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
