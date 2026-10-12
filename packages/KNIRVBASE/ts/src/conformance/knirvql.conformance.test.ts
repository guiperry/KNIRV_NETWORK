// KNIRVQL: the TS parser/optimizer/executor must agree with Go on a shared corpus.
import { execFileSync, spawnSync } from 'child_process';
import * as fs from 'fs';
import * as os from 'os';
import * as path from 'path';
import { IndexType } from '../components/storage/index';
import { KNIRVQLParser, KNIRVQLExecutor, MemoryQueryHost } from '../components/query';

jest.setTimeout(180_000);

const GO_DIR = path.resolve(__dirname, '../../../go');
const BIN = process.env.NRVCONFORM_BIN;
const goAvailable = () => (BIN ? fs.existsSync(BIN) : spawnSync('go', ['version']).status === 0);
function goRun(args: string[], input?: string): string {
  const opts = { cwd: GO_DIR, input, maxBuffer: 64 << 20 };
  return (BIN
    ? execFileSync(BIN, args, opts)
    : execFileSync('go', ['run', './cmd/nrvconform', ...args], opts)
  ).toString().trim();
}

const PARSE_CORPUS = [
  'GET MEMORY',
  'get memory',
  'GET AUTH',
  'GET MEMORY FROM frames',
  'GET MEMORY WHERE name = "alice"',
  'GET MEMORY WHERE a = 5 AND b = 2.5 AND c = true AND d = false',
  'GET MEMORY WHERE a = 1 OR b = 2',
  'GET MEMORY WHERE x = 0x1F',
  'GET MEMORY WHERE x = 1e3',
  'GET MEMORY WHERE x = 85abc',
  'GET MEMORY WHERE intent_flags & 0x1',
  'GET MEMORY WHERE name contains bob',
  'GET MEMORY WHERE name STARTS_WITH al',
  'GET MEMORY WHERE WITH TAGS (error,build)',
  'GET MEMORY WHERE v SIMILAR TO [0.1,0.2,0.3]',
  'GET MEMORY WHERE a = 1 SIMILAR TO [1,2] LIMIT 5',
  'GET MEMORY SIMILAR TO [1,2,3]',
  'GET MEMORY WHERE a = 1 LIMIT 100',
  'GET MEMORY LIMIT 7',
  'GET MEMORY WHERE a = 1 LIMIT abc',
  'GET MEMORY.MODALITY(vector) WHERE dataset = "test" AND verified = true',
  'GET MEMORY.MODALITY(seed)',
  'GET MEMORY.BRACKET(golden_seed)',
  'GET MEMORY.BRACKET(pos_tag) WHERE pos_tag = VERB LIMIT 3',
  'GET MEMORY WHERE z3_status = VALID',
  'GET MEMORY WHERE avg_temp_c < 85',
  'GET MEMORY WHERE pos_tag = VERB AND domain = MATH',
  'SET k = "some value"',
  'SET k = v1 v2',
  'DELETE WHERE id = abc',
  'DELETE WHERE id = "abc"',
  'CREATE INDEX users:username ON users (username) UNIQUE',
  'CREATE INDEX users:vector ON users (a,b)',
  'create index users:u on users (a)',
  'CREATE COLLECTION things',
  'DROP INDEX users:username',
  'DROP COLLECTION things',
  // errors
  '',
  'FROB x',
  'GET',
  'GET FOO',
  'SET k v',
  'DELETE WHERE id',
  'CREATE INDEX a:b ON c (x)',
  'CREATE INDEX nocolon ON x (y)',
  'CREATE WIDGET x',
  'DROP INDEX bad',
];

const DOCS = [
  { id: 'a', entryType: 'MEMORY', payload: { name: 'alice', n: 3, tags: ['x', 'y'], active: true, z3: { status: 'VALID' }, thermo: { temp_celsius: 60 }, brackets: { max_drift: 0.4, types: ['I', 'P'] },
      brackets_index: [{ id: 'a0', type: 'I', pos_tag: 2, tense: 2, domain_sig: 0x2000, intent_flags: 3, lsh_salt: 100, golden_seed: 77, subsecond_us: 5, drift_score: 0.1, lsh_vector: [0.5, 0.25] }, { id: 'a1', type: 'P', pos_tag: 1, tense: 1, domain_sig: 0x1000, intent_flags: 1, lsh_salt: 7, golden_seed: 78, subsecond_us: 6, drift_score: 0.9 }] } },
  { id: 'b', entryType: 'MEMORY', payload: { name: 'bob', n: 9, tags: ['y'], active: false, z3: { status: 'INVALID' }, thermo: { temp_celsius: 90 }, brackets: { max_drift: 0.05 },
      brackets_index: [{ id: 'b0', type: 'I', pos_tag: 1, tense: 0, domain_sig: 0x3000, intent_flags: 0, lsh_salt: 5000, golden_seed: 1, subsecond_us: 2, drift_score: 0 }] } },
  { id: 'c', entryType: 'MEMORY', payload: { name: 'carol', n: 12.5, score: 1e-5, tags: [] } },
  { id: 'd', entryType: 'MEMORY' }, // no payload: never matches a filter
  { id: 'e', entryType: 'MEMORY', payload: { name: 'alice', n: 3, note: null } },
];

const EXEC_CORPUS = [
  'GET MEMORY',
  'GET MEMORY WHERE name = alice',
  'GET MEMORY WHERE name = alice AND n = 3',
  'GET MEMORY WHERE name = alice AND n = 4',
  'GET MEMORY WHERE name != alice',
  'GET MEMORY WHERE n > 5',
  'GET MEMORY WHERE n >= 9',
  'GET MEMORY WHERE n < 10',
  'GET MEMORY WHERE n <= 3',
  'GET MEMORY WHERE name CONTAINS ar',
  'GET MEMORY WHERE name STARTS_WITH b',
  'GET MEMORY WHERE WITH TAGS (y)',
  'GET MEMORY WHERE WITH TAGS (x,y)',
  'GET MEMORY WHERE active = true',
  'GET MEMORY WHERE note = null',
  'GET MEMORY WHERE missing = 1',
  'GET MEMORY WHERE score > 0',
  'GET MEMORY WHERE z3_status = VALID',
  'GET MEMORY WHERE avg_temp_c < 85',
  'GET MEMORY WHERE avg_temp_c >= 90',
  'GET MEMORY WHERE avg_temp_c != 60',
  'GET MEMORY WHERE drift_score > 0.1',
  'GET MEMORY WHERE bracket_type = P',
  'GET MEMORY WHERE bracket_type = i',
  'GET MEMORY WHERE pos_tag = VERB',
  'GET MEMORY WHERE pos_tag = 1',
  'GET MEMORY WHERE tense = PRESENT',
  'GET MEMORY WHERE tense = N/A',
  'GET MEMORY WHERE domain = MATH',
  'GET MEMORY WHERE domain = CODE',
  'GET MEMORY WHERE pos_tag = VERB AND domain = MATH',
  'GET MEMORY WHERE intent_flags & 0x1',
  'GET MEMORY WHERE intent_flags & 3',
  'GET MEMORY WHERE intent_flags = 0',
  'GET MEMORY WHERE intent_flags > 0',
  'GET MEMORY WHERE lsh_salt = 100',
  'GET MEMORY WHERE lsh_salt != 100',
  'GET MEMORY WHERE lsh_salt > 4000',
  'GET MEMORY WHERE lsh_salt <= 7',
  'GET MEMORY WHERE name = alice LIMIT 1',
  'GET MEMORY.BRACKET(golden_seed)',
  'GET MEMORY.BRACKET(pos_tag) WHERE pos_tag = VERB',
  'GET MEMORY.BRACKET(drift_score)',
  'GET MEMORY.BRACKET(lsh_vector)',
  'GET MEMORY.BRACKET(projections)',
  'GET MEMORY.BRACKET(type)',
  'GET MEMORY.BRACKET(nope)',
  'GET MEMORY.MODALITY(vector)',
  'GET MEMORY.MODALITY(seed)',
  'GET MEMORY.MODALITY(thermo)',
  'GET MEMORY.MODALITY(proof)',
  'GET MEMORY.MODALITY(other)',
  'GET AUTH',
];

// Rows are compared as sets: Go returns documents in collection order, which
// FileStorage does not define. Go runs against the collection object the caller
// passes (always "memory" here), so TS is given the same collection explicitly. null and [] both mean "no rows".
function normalise(res: any): any {
  if (res === null || res === undefined) return [];
  if (Array.isArray(res)) {
    return res
      .map((r) => (r && typeof r === 'object' && !Array.isArray(r) && 'payload' in r ? { id: r.id } : r))
      .map((r) => JSON.stringify(r, (_k, v) => (v && typeof v === 'object' && !Array.isArray(v) ? Object.fromEntries(Object.entries(v).sort()) : v)))
      .sort();
  }
  return res;
}

const maybe = goAvailable() || process.env.NRVCONFORM_REQUIRED ? describe : describe.skip;

maybe('KNIRVQL Go <-> TS', () => {
  it.each(PARSE_CORPUS)('parse: %j', (text) => {
    const go = JSON.parse(goRun(['parse', ...text.split(/\s+/).filter(Boolean)]));
    let ts: any;
    try {
      const q = new KNIRVQLParser().parse(text);
      ts = { ...q };
    } catch (e: any) {
      ts = { error: e.message };
    }
    if (go.error !== undefined) {
      expect(ts.error).toBeDefined();
      expect(ts.error).toBe(go.error);
      return;
    }
    expect(ts.error).toBeUndefined();
    expect({
      type: ts.type, entryType: ts.entryType, collection: ts.collection ?? '', id: ts.id ?? '', key: ts.key ?? '',
      value: ts.value ?? '', filters: ts.filters ?? [], similarTo: ts.similarTo ?? [], limit: ts.limit ?? 0,
      indexName: ts.indexName ?? '', fields: ts.fields ?? [], unique: ts.unique ?? false,
      modalityType: ts.modalityType ?? '', bracketField: ts.bracketField ?? '',
    }).toEqual({
      type: go.type, entryType: go.entryType || ts.entryType, collection: go.collection, id: go.id, key: go.key,
      value: go.value, filters: go.filters, similarTo: go.similarTo, limit: go.limit, indexName: go.indexName,
      fields: go.fields, unique: go.unique, modalityType: go.modalityType, bracketField: go.bracketField,
    });
  });

  for (const withIndex of [false, true]) {
    describe(withIndex ? 'execution with a btree index on name' : 'execution (full scans)', () => {
      let goOut: any[];
      let tsOut: any[];
      beforeAll(async () => {
        const indexes = withIndex ? [{ name: 'name', fields: ['name'], unique: false }] : [];
        goOut = JSON.parse(goRun(['query'], JSON.stringify({ docs: DOCS, indexes, queries: EXEC_CORPUS })));
        const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'knirvql-'));
        const host = new MemoryQueryHost(dir);
        for (const ix of indexes) await host.createIndex('memory', ix.name, IndexType.BTree, ix.fields, ix.unique, '', {});
        for (const d of DOCS) await host.insert('memory', JSON.parse(JSON.stringify(d)));
        const ex = new KNIRVQLExecutor(host);
        tsOut = [];
        for (const q of EXEC_CORPUS) {
          try { tsOut.push({ result: await ex.run(q, 'memory') }); } catch (e: any) { tsOut.push({ error: e.message }); }
        }
        fs.rmSync(dir, { recursive: true, force: true });
      });

      it.each(EXEC_CORPUS.map((q, i) => [q, i] as const))('%s', (q, i) => {
        const g = goOut[i];
        const t = tsOut[i];
        if (g.error !== undefined) {
          expect(t.error).toBeDefined();
          return;
        }
        expect(t.error).toBeUndefined();
        if (/LIMIT/.test(q)) {
          expect(normalise(t.result)).toHaveLength(normalise(g.result).length);
        } else {
          expect(normalise(t.result)).toEqual(normalise(g.result));
        }
      });
    });
  }
});
