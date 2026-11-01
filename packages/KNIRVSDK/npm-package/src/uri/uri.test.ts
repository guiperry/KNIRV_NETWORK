import { RESOURCE_TYPES, deriveId, parse, type UriResult } from './index.js';

function fail(message: string): never {
  throw new Error(message);
}

function equal(actual: unknown, expected: unknown, label: string): void {
  if (actual !== expected) fail(`${label}: got ${String(actual)}, want ${String(expected)}`);
}

/**
 * Runs the shared vectors from testvectors/uri/uri_v1.json. The JS runner in
 * scripts/test-uri.mjs loads the file and calls this so the TS package needs no
 * Node type definitions just for tests.
 */
export function runUriVectors(vectors: {
  resource_types: string[];
  derive: Array<{ raw: string; id: string }>;
  cases: Array<Record<string, any>>;
}): void {
  for (const name of vectors.resource_types) {
    if (!(RESOURCE_TYPES as readonly string[]).includes(name)) fail(`missing resource type ${name}`);
  }
  equal(RESOURCE_TYPES.length, vectors.resource_types.length, 'resource type count');

  for (const tc of vectors.derive) {
    equal(deriveId(tc.raw), tc.id, `derive(${tc.raw})`);
  }

  for (const tc of vectors.cases) {
    const got: UriResult = parse(tc.input);
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
      for (const [key, value] of Object.entries(tc.query)) equal(got.query[key], value, `${tc.name} query ${key}`);
      const again = parse(got.canonical);
      equal(again.canonical, tc.canonical, `${tc.name} reparse`);
    }
    if (tc.kind === 'legacy_dve') {
      equal(got.legacy_dve?.form, tc.legacy_dve.form, `${tc.name} form`);
      equal(got.legacy_dve?.id, tc.legacy_dve.id, `${tc.name} id`);
    }
    if (tc.kind === 'content_pointer') {
      equal(got.content_pointer?.kind, tc.content_pointer.kind, `${tc.name} pointer kind`);
      equal(got.content_pointer?.hash, tc.content_pointer.hash, `${tc.name} pointer hash`);
    }
    if (tc.kind === 'action') equal(got.action?.name, tc.action.name, `${tc.name} action`);
  }
}
