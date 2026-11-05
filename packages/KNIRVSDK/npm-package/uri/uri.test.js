import { callLocal } from '../local.js';
import { QUERY_KEYS, RESOURCE_TYPES, deriveId, parse } from './index.js';
function fail(message) {
    throw new Error(message);
}
function equal(actual, expected, label) {
    if (actual !== expected)
        fail(`${label}: got ${String(actual)}, want ${String(expected)}`);
}
/**
 * Runs the shared vectors from testvectors/uri/uri_v1.json. The JS runner in
 * scripts/test-uri.mjs loads the file and calls this so the TS package needs no
 * Node type definitions just for tests.
 */
export function runUriVectors(vectors) {
    for (const name of vectors.resource_types) {
        if (!RESOURCE_TYPES.includes(name))
            fail(`missing resource type ${name}`);
    }
    equal(RESOURCE_TYPES.length, vectors.resource_types.length, 'resource type count');
    // The TS constants are a typing aid; the core is the registry. They must agree.
    const core = callLocal('uri.registry');
    equal(JSON.stringify([...core.resource_types].sort()), JSON.stringify([...RESOURCE_TYPES].sort()), 'resource types vs core');
    equal(JSON.stringify([...core.query_keys].sort()), JSON.stringify([...QUERY_KEYS].sort()), 'query keys vs core');
    equal(JSON.stringify([...core.query_keys].sort()), JSON.stringify([...vectors.query_keys].sort()), 'query keys vs vectors');
    for (const tc of vectors.derive) {
        equal(deriveId(tc.raw), tc.id, `derive(${tc.raw})`);
    }
    for (const tc of vectors.cases) {
        const got = parse(tc.input);
        equal(got.kind, tc.kind, `${tc.name} kind`);
        if (tc.kind === 'invalid') {
            equal(got.reason, tc.reason, `${tc.name} reason`);
            continue;
        }
        equal(got.canonical, tc.canonical, `${tc.name} canonical`);
        if (tc.kind === 'canonical') {
            equal(got.id, tc.id, `${tc.name} id`);
            equal(got.resource_type, tc.resource_type, `${tc.name} type`);
            equal(got.path, tc.path, `${tc.name} path`);
            equal(Object.keys(got.query).length, Object.keys(tc.query).length, `${tc.name} query size`);
            for (const [key, value] of Object.entries(tc.query))
                equal(got.query[key], value, `${tc.name} query ${key}`);
            const again = parse(got.canonical);
            equal(again.canonical, tc.canonical, `${tc.name} reparse`);
        }
        if (tc.kind === 'legacy_dve') {
            equal(got.legacy_dve?.form, tc.legacy_dve.form, `${tc.name} form`);
            equal(got.legacy_dve?.id, tc.legacy_dve.id, `${tc.name} id`);
            equal(got.legacy_dve?.hash, tc.legacy_dve.hash, `${tc.name} hash`);
            equal(got.legacy_dve?.name, tc.legacy_dve.name, `${tc.name} name`);
        }
        if (tc.kind === 'content_pointer') {
            equal(got.content_pointer?.kind, tc.content_pointer.kind, `${tc.name} pointer kind`);
            equal(got.content_pointer?.hash, tc.content_pointer.hash, `${tc.name} pointer hash`);
        }
        if (tc.kind === 'action') {
            equal(got.action?.name, tc.action.name, `${tc.name} action`);
            equal(Object.keys(got.action?.query ?? {}).length, Object.keys(tc.action.query).length, `${tc.name} action query size`);
            for (const [key, value] of Object.entries(tc.action.query))
                equal(got.action?.query[key], value, `${tc.name} action query ${key}`);
        }
    }
}
