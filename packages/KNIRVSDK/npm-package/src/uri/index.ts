import { callLocal } from '../local.js';
export { initLocal } from '../local.js';

/** Typing aid: mirrors the registry in the Rust core (core/src/uri.rs); uri.test.ts fails if they differ. */
export const RESOURCE_TYPES = [
  'chain', 'nrn', 'ulora', 'mcpcontext', 'resource', 'tool', 'prompt',
  'memoryservice', 'memory', 'capability', 'dev', 'errornode', 'skillnode',
  'ideanode', 'propertynode', 'usdscene', 'usdasset',
] as const;

export const QUERY_KEYS = ['tenant', 'snapshot', 'sha256'] as const;
export type UriKind = 'canonical' | 'legacy_dve' | 'content_pointer' | 'action' | 'invalid';

export interface LegacyDve {
  form: string;
  id: string;
  hash?: string;
  name?: string;
}

export interface ContentPointer {
  kind: string;
  hash: string;
}

export interface UriAction {
  name: string;
  canonical: string;
  query: Record<string, string>;
}

export interface UriResult {
  kind: UriKind;
  canonical: string;
  id: string;
  resource_type: string;
  path: string;
  query: Record<string, string>;
  legacy_dve?: LegacyDve;
  content_pointer?: ContentPointer;
  action?: UriAction;
  reason?: string;
}


interface WireUri {
  kind: UriKind; canonical: string; id: string; resource_type: string; path: string;
  query?: Record<string, string> | null; legacy_dve?: LegacyDve | null; content_pointer?: ContentPointer | null;
  action?: UriAction | null; reason?: string | null;
}

/** Parses without ever throwing on bad input: failures are `kind: 'invalid'`. */
export function parse(raw: string): UriResult {
  const r = callLocal<WireUri>('uri.parse', { raw });
  const out: UriResult = { kind: r.kind, canonical: r.canonical, id: r.id, resource_type: r.resource_type, path: r.path, query: r.query ?? {} };
  if (r.legacy_dve) out.legacy_dve = { form: r.legacy_dve.form, id: r.legacy_dve.id, ...(r.legacy_dve.hash ? { hash: r.legacy_dve.hash } : {}), ...(r.legacy_dve.name ? { name: r.legacy_dve.name } : {}) };
  if (r.content_pointer) out.content_pointer = r.content_pointer;
  if (r.action) out.action = r.action;
  if (r.reason) out.reason = r.reason;
  return out;
}

export function normalizeId(raw: string): string {
  return callLocal<{ id: string }>('uri.normalize_id', { raw }).id;
}

export function deriveId(raw: string): string {
  return callLocal<{ id: string }>('uri.derive_id', { raw }).id;
}
