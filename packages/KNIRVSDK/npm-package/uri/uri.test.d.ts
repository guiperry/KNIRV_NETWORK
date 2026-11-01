/**
 * Runs the shared vectors from testvectors/uri/uri_v1.json. The JS runner in
 * scripts/test-uri.mjs loads the file and calls this so the TS package needs no
 * Node type definitions just for tests.
 */
export declare function runUriVectors(vectors: {
    resource_types: string[];
    derive: Array<{
        raw: string;
        id: string;
    }>;
    cases: Array<Record<string, any>>;
}): void;
