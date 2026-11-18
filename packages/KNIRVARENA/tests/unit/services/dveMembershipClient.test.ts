import {
  DveMembershipError,
  dveMembershipPath,
  fetchDveMembership
} from '../../../src/services/dveMembershipClient';

const jsonResponse = (status: number, body: unknown): Response =>
  ({
    ok: status >= 200 && status < 300,
    status,
    json: async () => body
  }) as unknown as Response;

describe('dveMembershipClient', () => {
  afterEach(() => {
    window.localStorage.clear();
  });

  it('builds the per-DVE path and encodes the id', () => {
    expect(dveMembershipPath('dve-a')).toBe('/api/dves/dve-a/error-nodes');
    expect(dveMembershipPath('a/b')).toBe('/api/dves/a%2Fb/error-nodes');
  });

  it('GETs the membership with the session token and returns the entries', async () => {
    const fetchImpl = jest.fn().mockResolvedValue(
      jsonResponse(200, {
        dve_id: 'dve-a',
        error_nodes: [
          {
            error_node_id: 'node-a',
            role: 'originator',
            seen_at: '2026-11-16T00:00:00Z',
            found: true,
            status: 'open',
            suite: { status: 'collecting', count: 5, required: 8, progress: '5/8' }
          }
        ]
      })
    );
    const res = await fetchDveMembership('dve-a', { baseUrl: 'https://gw.example', fetchImpl, token: 'tok-123' });
    expect(fetchImpl).toHaveBeenCalledTimes(1);
    const [url, init] = fetchImpl.mock.calls[0];
    expect(url).toBe('https://gw.example/api/dves/dve-a/error-nodes');
    expect(init.method).toBe('GET');
    expect(init.credentials).toBe('include');
    expect(init.headers.Authorization).toBe('Bearer tok-123');
    expect(res.error_nodes[0].suite.progress).toBe('5/8');
  });

  it('omits the Authorization header when no token is stored', async () => {
    const fetchImpl = jest.fn().mockResolvedValue(jsonResponse(200, { dve_id: 'dve-a', error_nodes: [] }));
    await fetchDveMembership('dve-a', { fetchImpl });
    expect(fetchImpl.mock.calls[0][1].headers.Authorization).toBeUndefined();
  });

  it.each([
    [404, 'not_found'],
    [403, 'forbidden'],
    [401, 'forbidden'],
    [503, 'unavailable']
  ])('maps HTTP %i to %s', async (status, code) => {
    const fetchImpl = jest.fn().mockResolvedValue(jsonResponse(status, {}));
    await expect(fetchDveMembership('dve-a', { fetchImpl })).rejects.toMatchObject({ code });
  });

  it('rejects a body without error_nodes', async () => {
    const fetchImpl = jest.fn().mockResolvedValue(jsonResponse(200, { dve_id: 'dve-a' }));
    await expect(fetchDveMembership('dve-a', { fetchImpl })).rejects.toBeInstanceOf(DveMembershipError);
  });

  it('refuses an empty DVE id before any request', async () => {
    const fetchImpl = jest.fn();
    await expect(fetchDveMembership('', { fetchImpl })).rejects.toMatchObject({ code: 'not_found' });
    expect(fetchImpl).not.toHaveBeenCalled();
  });
});
