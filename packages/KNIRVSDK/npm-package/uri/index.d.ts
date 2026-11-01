export { initLocal } from '../local.js';
/** The routable resource-type registry. Defined once, in the Rust core. */
export declare const RESOURCE_TYPES: readonly ["chain", "nrn", "ulora", "mcpcontext", "resource", "tool", "prompt", "memoryservice", "memory", "capability", "dev", "errornode", "skillnode", "ideanode", "propertynode", "usdscene", "usdasset"];
export declare const QUERY_KEYS: readonly ["tenant", "snapshot", "sha256"];
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
/** Parses without ever throwing on bad input: failures are `kind: 'invalid'`. */
export declare function parse(raw: string): UriResult;
export declare function normalizeId(raw: string): string;
export declare function deriveId(raw: string): string;
