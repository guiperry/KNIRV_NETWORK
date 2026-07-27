import { ApiError, collectionFrom, requestJSON } from './api-client';

describe('requestJSON', () => {
  afterEach(() => {
    jest.restoreAllMocks();
    delete global.fetch;
  });

  it('returns JSON for a successful response', async () => {
    global.fetch = jest.fn().mockResolvedValue({
      ok: true,
      json: async () => ({ status: 'ok' }),
    });

    await expect(requestJSON('/api/example')).resolves.toEqual({ status: 'ok' });
    expect(global.fetch).toHaveBeenCalledWith('/api/example', expect.objectContaining({
      headers: { Accept: 'application/json' },
    }));
  });

  it('includes an HTTP failure in the error', async () => {
    global.fetch = jest.fn().mockResolvedValue({
      ok: false,
      status: 503,
      text: async () => 'service unavailable',
    });

    await expect(requestJSON('/api/example')).rejects.toEqual(expect.objectContaining({
      name: 'ApiError',
      status: 503,
      url: '/api/example',
    }));
  });
});

describe('collectionFrom', () => {
  it('normalizes direct and wrapped API collections', () => {
    expect(collectionFrom([{ id: 'a' }], ['blocks'])).toEqual([{ id: 'a' }]);
    expect(collectionFrom({ blocks: [{ id: 'b' }] }, ['blocks'])).toEqual([{ id: 'b' }]);
    expect(collectionFrom({ value: 'not-a-list' }, ['blocks'])).toEqual([]);
  });
});
