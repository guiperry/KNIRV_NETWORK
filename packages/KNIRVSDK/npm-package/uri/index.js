import { callLocal } from '../local.js';
export { initLocal } from '../local.js';
/** The routable resource-type registry. Defined once, in the Rust core. */
export const RESOURCE_TYPES = [
    'chain', 'nrn', 'ulora', 'mcpcontext', 'resource', 'tool', 'prompt',
    'memoryservice', 'memory', 'capability', 'dev', 'errornode', 'skillnode',
    'ideanode', 'propertynode', 'usdscene', 'usdasset',
];
export const QUERY_KEYS = ['tenant', 'snapshot', 'sha256'];
/** Parses without ever throwing on bad input: failures are `kind: 'invalid'`. */
export function parse(raw) {
    const r = callLocal('uri.parse', { raw });
    const out = { kind: r.kind, canonical: r.canonical, id: r.id, resource_type: r.resource_type, path: r.path, query: r.query ?? {} };
    if (r.legacy_dve)
        out.legacy_dve = { form: r.legacy_dve.form, id: r.legacy_dve.id, ...(r.legacy_dve.hash ? { hash: r.legacy_dve.hash } : {}), ...(r.legacy_dve.name ? { name: r.legacy_dve.name } : {}) };
    if (r.content_pointer)
        out.content_pointer = r.content_pointer;
    if (r.action)
        out.action = r.action;
    if (r.reason)
        out.reason = r.reason;
    return out;
}
export function normalizeId(raw) {
    return callLocal('uri.normalize_id', { raw }).id;
}
export function deriveId(raw) {
    return callLocal('uri.derive_id', { raw }).id;
}
