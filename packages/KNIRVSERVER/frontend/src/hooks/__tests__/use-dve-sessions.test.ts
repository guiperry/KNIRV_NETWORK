import { renderHook, waitFor } from '@testing-library/react';
import { useDVESessions } from '../use-dve-sessions';
import { apiRequest } from '@/lib/api';

jest.mock('@/lib/api', () => ({ apiRequest: jest.fn() }));

const mockApiRequest = apiRequest as jest.MockedFunction<typeof apiRequest>;

describe('useDVESessions', () => {
  beforeEach(() => jest.clearAllMocks());

  it('does not request sessions without a creation ID', () => {
    const { result } = renderHook(() => useDVESessions(null));
    expect(result.current.sessions).toEqual([]);
    expect(mockApiRequest).not.toHaveBeenCalled();
  });

  it('loads sessions for the selected creation', async () => {
    const sessions = [{ id: 'session-1', status: 'active' }];
    mockApiRequest.mockResolvedValue({ success: true, data: sessions } as never);

    const { result } = renderHook(() => useDVESessions('creation-1'));

    await waitFor(() => expect(result.current.loading).toBe(false));
    expect(result.current.sessions).toEqual(sessions);
    expect(mockApiRequest).toHaveBeenCalledWith(
      '/api/dve-creation/nodes/creation-1/sessions',
      { method: 'GET' },
    );
  });

  it('reports request failures', async () => {
    mockApiRequest.mockRejectedValue(new Error('sessions unavailable'));
    const { result } = renderHook(() => useDVESessions('creation-1'));

    await waitFor(() => expect(result.current.loading).toBe(false));
    expect(result.current.error).toBe('sessions unavailable');
    expect(result.current.sessions).toEqual([]);
  });
});
