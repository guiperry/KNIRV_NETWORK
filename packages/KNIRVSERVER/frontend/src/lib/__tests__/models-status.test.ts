import { readFileSync } from 'fs';
import { join } from 'path';
import { STATUS_PATH, roles, type PlatformStatus, type RoleLine } from '../models-status';

// The SDK's shared vectors: the Go and TS twins in KNIRVSDK pass the same file.
const cases = JSON.parse(
  readFileSync(join(__dirname, '../../../../../KNIRVSDK/testvectors/models/platform_status.json'), 'utf8'),
) as Record<string, { status: PlatformStatus; roles: RoleLine[] }>;

const norm = (l: RoleLine) => ({ ...l, detail: l.detail || undefined });

describe('models-status (copy of the SDK module)', () => {
  it('produces the shared role lines for every case', () => {
    expect(Object.keys(cases)).toHaveLength(3);
    for (const [name, c] of Object.entries(cases)) {
      expect({ name, lines: roles(c.status).map(norm) }).toEqual({ name, lines: c.roles.map(norm) });
    }
  });
  it('uses the contract path', () => expect(STATUS_PATH).toBe('/api/models/status'));
});
