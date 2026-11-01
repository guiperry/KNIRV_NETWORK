/**
 * Platform model status (GET /api/models/status): which model does what, right now.
 *
 * The wire types and `roles()` come from `@knirv/sdk/models`; `roles()` runs in the
 * SDK's Rust core, so it needs the core initialised (main.tsx awaits `initLocal()`
 * before the app starts). `modelStatus.test.ts` checks it against the SDK's shared
 * fixture (testvectors/models/platform_status.json).
 */

import { roles, STATUS_PATH, type PlatformStatus } from '@knirv/sdk/models';
import { getKnirvServerUrl } from '../config/runtimeConfig';

export { roles, STATUS_PATH };
export type {
  PlatformStatus, RoleLine, RoleState, ModelStatus, LlamaAdapter, LlamaStatus, RoutingEvent, RouteStats,
} from '@knirv/sdk/models';

/** Reads the status document with the signed-in user's token. Throws on any failure. */
export async function fetchModelStatus(token: string | null, limit = 10, fetchImpl: typeof fetch = fetch): Promise<PlatformStatus> {
  if (!token) throw new Error('Sign in to KNIRV to see model status');
  const res = await fetchImpl(`${getKnirvServerUrl()}${STATUS_PATH}?limit=${limit}`, { headers: { Authorization: `Bearer ${token}` } });
  if (res.status === 404) throw new Error('This KNIRV server does not report model status yet');
  if (!res.ok) throw new Error(`Model status unavailable (HTTP ${res.status})`);
  return (await res.json()) as PlatformStatus;
}
