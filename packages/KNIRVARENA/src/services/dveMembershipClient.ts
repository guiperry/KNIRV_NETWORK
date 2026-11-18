/**
 * Client for GET /api/dves/{id}/error-nodes (DVE-U26, plan D7.6).
 *
 * The response is a reference list: each entry names an error node the DVE is
 * a member of (originator or witness) with its NRV suite progress. It never
 * carries node content such as code_context or project_path; the backend
 * whitelists those fields.
 */

export type DveMembershipRole = 'originator' | 'witness';

export interface DveMembershipSuite {
  /** "collecting" until the suite seals, then "sealed". */
  status: string;
  /** NRV test cases collected so far. */
  count: number;
  /** Test cases needed to seal (8). */
  required: number;
  /** "count/required", e.g. "5/8". */
  progress: string;
  suite_hash?: string;
}

export interface DveMembershipEntry {
  error_node_id: string;
  role: DveMembershipRole;
  seen_at: string;
  /** False when KNIRVGRAPH no longer lists the node. */
  found: boolean;
  status?: string;
  error_type?: string;
  description?: string;
  severity?: number;
  failure_count?: number;
  suite: DveMembershipSuite;
}

export interface DveMembershipResponse {
  dve_id: string;
  error_nodes: DveMembershipEntry[];
}

export class DveMembershipError extends Error {
  constructor(
    readonly code: 'not_found' | 'forbidden' | 'unavailable' | 'bad_response',
    message: string
  ) {
    super(message);
    this.name = 'DveMembershipError';
  }
}

export interface DveMembershipRequestOptions {
  /** Origin of the KNIRVSERVER API. Empty string means same origin. */
  baseUrl?: string;
  fetchImpl?: typeof fetch;
  /** Bearer token. Defaults to the stored session token when omitted. */
  token?: string | null;
}

const readToken = (): string | null => {
  try {
    if (typeof window !== 'undefined' && window.localStorage) {
      return window.localStorage.getItem('knirv_auth_token');
    }
  } catch {
    // Storage can be blocked (private window, cleared site data). Fall through.
  }
  return null;
};

export const dveMembershipPath = (dveId: string): string =>
  `/api/dves/${encodeURIComponent(dveId)}/error-nodes`;

/** Loads the DVE's membership list. Throws DveMembershipError on any failure. */
export async function fetchDveMembership(
  dveId: string,
  options: DveMembershipRequestOptions = {}
): Promise<DveMembershipResponse> {
  if (!dveId) {
    throw new DveMembershipError('not_found', 'a DVE id is required');
  }
  const doFetch = options.fetchImpl ?? fetch;
  const headers: Record<string, string> = { Accept: 'application/json' };
  const token = options.token !== undefined ? options.token : readToken();
  if (token) {
    headers.Authorization = `Bearer ${token}`;
  }
  const res = await doFetch(`${options.baseUrl ?? ''}${dveMembershipPath(dveId)}`, {
    method: 'GET',
    headers,
    credentials: 'include'
  });
  if (res.status === 404) {
    throw new DveMembershipError('not_found', 'DVE not found');
  }
  if (res.status === 401 || res.status === 403) {
    throw new DveMembershipError('forbidden', 'not allowed to view this DVE');
  }
  if (!res.ok) {
    throw new DveMembershipError('unavailable', `error nodes unavailable (${res.status})`);
  }
  let body: unknown;
  try {
    body = await res.json();
  } catch {
    throw new DveMembershipError('bad_response', 'membership response was not JSON');
  }
  const parsed = body as Partial<DveMembershipResponse> | null;
  if (!parsed || !Array.isArray(parsed.error_nodes)) {
    throw new DveMembershipError('bad_response', 'membership response is missing error_nodes');
  }
  return { dve_id: String(parsed.dve_id ?? dveId), error_nodes: parsed.error_nodes };
}
