import { renderHook, act, waitFor } from '@testing-library/react';
import { useUpdateCheck } from '../use-update-check';

global.fetch = jest.fn();
const mockFetch = fetch as jest.MockedFunction<typeof fetch>;

const makeUpdateResponse = (available: boolean, latestTag = 'v2.0.0') => ({
  ok: true,
  json: () =>
    Promise.resolve({
      available,
      latest_tag: latestTag,
      current_version: 'v1.0.0',
      checked_at: '2026-05-16T12:00:00Z',
    }),
} as unknown as Response);

describe('useUpdateCheck', () => {
  beforeEach(() => {
    jest.useFakeTimers();
    jest.clearAllMocks();
  });

  afterEach(() => {
    jest.useRealTimers();
  });

  it('returns null status before first fetch resolves', () => {
    mockFetch.mockImplementation(() => new Promise(() => {}));
    const { result } = renderHook(() => useUpdateCheck());
    expect(result.current.status).toBeNull();
    expect(result.current.error).toBeNull();
    expect(result.current.applying).toBe(false);
  });

  it('sets status.available=true when update is available', async () => {
    mockFetch.mockResolvedValueOnce(makeUpdateResponse(true));
    const { result } = renderHook(() => useUpdateCheck());
    await waitFor(() => expect(result.current.status).not.toBeNull());
    expect(result.current.status?.available).toBe(true);
    expect(result.current.status?.latest_tag).toBe('v2.0.0');
    expect(result.current.status?.current_version).toBe('v1.0.0');
  });

  it('sets status.available=false when up to date', async () => {
    mockFetch.mockResolvedValueOnce(makeUpdateResponse(false, 'v1.0.0'));
    const { result } = renderHook(() => useUpdateCheck());
    await waitFor(() => expect(result.current.status).not.toBeNull());
    expect(result.current.status?.available).toBe(false);
  });

  it('does not update status on non-ok response', async () => {
    mockFetch.mockResolvedValueOnce({ ok: false } as unknown as Response);
    const { result } = renderHook(() => useUpdateCheck());
    await waitFor(() => {
      // Give the hook a tick to run the fetch
      return new Promise((r) => setTimeout(r, 10));
    });
    expect(result.current.status).toBeNull();
  });

  it('sets error on network failure', async () => {
    mockFetch.mockRejectedValueOnce(new Error('Network error'));
    const { result } = renderHook(() => useUpdateCheck());
    await waitFor(() => expect(result.current.error).not.toBeNull());
    expect(result.current.error).toBe('Network error');
  });

  it('polls again after 5 minutes', async () => {
    mockFetch
      .mockResolvedValueOnce(makeUpdateResponse(false))
      .mockResolvedValueOnce(makeUpdateResponse(true));

    const { result } = renderHook(() => useUpdateCheck());
    await waitFor(() => expect(result.current.status).not.toBeNull());
    expect(result.current.status?.available).toBe(false);

    act(() => {
      jest.advanceTimersByTime(5 * 60 * 1000);
    });
    await waitFor(() => expect(result.current.status?.available).toBe(true));
    expect(mockFetch).toHaveBeenCalledTimes(2);
  });

  it('applyUpdate calls POST /api/v1/system/update/apply', async () => {
    mockFetch
      .mockResolvedValueOnce(makeUpdateResponse(true))   // initial check
      .mockResolvedValueOnce({ ok: true } as unknown as Response); // apply

    const { result } = renderHook(() => useUpdateCheck());
    await waitFor(() => expect(result.current.status).not.toBeNull());

    await act(async () => {
      await result.current.applyUpdate();
    });

    expect(mockFetch).toHaveBeenCalledWith(
      '/api/v1/system/update/apply',
      { method: 'POST' },
    );
    expect(result.current.applying).toBe(true);
  });

  it('cleanup removes the polling interval on unmount', async () => {
    mockFetch.mockResolvedValue(makeUpdateResponse(false));
    const { unmount } = renderHook(() => useUpdateCheck());
    await waitFor(() => expect(mockFetch).toHaveBeenCalledTimes(1));
    unmount();
    act(() => jest.advanceTimersByTime(10 * 60 * 1000));
    // No additional calls after unmount.
    expect(mockFetch).toHaveBeenCalledTimes(1);
  });
});
