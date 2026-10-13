import { readFileSync } from 'fs';
import { join } from 'path';
import { STATUS_PATH, fetchModelStatus, roles, type PlatformStatus, type RoleLine } from '../modelStatus';
import { setRuntimeConfig } from '../../config/runtimeConfig';

// The SDK's shared vectors: the Go and TS twins in KNIRVSDK pass the same file.
const cases = JSON.parse(
  readFileSync(join(__dirname, '../../../../KNIRVSDK/testvectors/models/platform_status.json'), 'utf8'),
) as Record<string, { status: PlatformStatus; roles: RoleLine[] }>;

const norm = (l: RoleLine) => ({ ...l, detail: l.detail || undefined });

describe('model status (copy of the SDK module)', () => {
  it('produces the shared role lines for every case', () => {
    expect(Object.keys(cases)).toHaveLength(3);
    for (const [name, c] of Object.entries(cases)) {
      expect({ name, lines: roles(c.status).map(norm) }).toEqual({ name, lines: c.roles.map(norm) });
    }
  });

  it('uses the contract path', () => {
    expect(STATUS_PATH).toBe('/api/models/status');
  });

  describe('fetchModelStatus', () => {
    beforeEach(() => setRuntimeConfig({ network: 'testnet', serverUrl: 'https://srv.test' }));
    afterEach(() => setRuntimeConfig(null));

    it('sends the bearer token and the limit', async () => {
      const fetchMock = jest.fn().mockResolvedValue({ ok: true, status: 200, json: () => Promise.resolve(cases.ready_all.status) });
      const s = await fetchModelStatus('tok', 4, fetchMock as unknown as typeof fetch);
      expect(s.daemon).toHaveLength(3);
      expect(fetchMock).toHaveBeenCalledWith('https://srv.test/api/models/status?limit=4', { headers: { Authorization: 'Bearer tok' } });
    });

    it('refuses without a token, names an old server, and surfaces other failures', async () => {
      await expect(fetchModelStatus(null)).rejects.toThrow(/Sign in/);
      const f = (status: number) => jest.fn().mockResolvedValue({ ok: status < 400, status, json: () => Promise.resolve({}) }) as unknown as typeof fetch;
      await expect(fetchModelStatus('t', 1, f(404))).rejects.toThrow(/does not report model status/);
      await expect(fetchModelStatus('t', 1, f(500))).rejects.toThrow(/HTTP 500/);
    });
  });
});
