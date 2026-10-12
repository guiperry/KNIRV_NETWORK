// KNIRVQL executor — a port of Go's Query.Execute (internal/query/knirvql.go).
// It plans with QueryOptimizer and runs full / index / index-only scans over a
// QueryHost, so it works against any collection implementation.
//
// Matching deliberately reproduces Go's behaviour, including its quirks, so the
// two implementations return the same rows for the same query (verified by
// src/conformance/knirvql.conformance.test.ts):
//   * values are compared through Go's "%v" formatting (goString);
//   * only documents whose `payload` is an object can match any filter;
//   * `!=` on the numeric fields (avg_temp_c, drift_score) behaves as `=`.
import { EntryType } from '../types/types';
import { Index, IndexType } from '../storage/index';
import { Filter, KNIRVQLParser, Query, QueryExecutor, QueryResult, QueryType } from './knirvql';
import { QueryOptimizer, QueryPlan, ScanType } from './optimizer';

export type Doc = Record<string, any>;

export interface QueryHost {
  findAll(collection: string): Promise<Doc[]>;
  find(collection: string, id: string): Promise<Doc | null>;
  getIndexes(collection: string): Index[];
  queryIndex(collection: string, indexName: string, query: Record<string, any>): Promise<string[]>;
  insert(collection: string, doc: Doc): Promise<unknown>;
  delete(collection: string, id: string): Promise<unknown>;
  createIndex(
    collection: string,
    name: string,
    type: IndexType,
    fields: string[],
    unique: boolean,
    partialExpr: string,
    options: Record<string, any>
  ): Promise<void>;
  dropIndex(collection: string, name: string): Promise<void>;
}

// ---- Go "%v" emulation -----------------------------------------------------

function goFloat(n: number): string {
  if (Number.isNaN(n)) return 'NaN';
  if (!Number.isFinite(n)) return n > 0 ? '+Inf' : '-Inf';
  const abs = Math.abs(n);
  if (abs !== 0 && (abs < 1e-4 || abs >= 1e21)) {
    // %g style exponent: at least two digits, explicit sign.
    return n.toExponential().replace(/e([+-])(\d)$/, 'e$10$2');
  }
  return String(n);
}

export function goString(v: unknown): string {
  if (v === null || v === undefined) return '<nil>';
  if (typeof v === 'number') return goFloat(v);
  if (typeof v === 'bigint') return v.toString();
  if (typeof v === 'string') return v;
  if (typeof v === 'boolean') return v ? 'true' : 'false';
  if (Array.isArray(v)) return `[${v.map(goString).join(' ')}]`;
  if (typeof v === 'object') {
    const o = v as Record<string, unknown>;
    return `map[${Object.keys(o).sort().map((k) => `${k}:${goString(o[k])}`).join(' ')}]`;
  }
  return String(v);
}

/** Byte-wise string comparison (Go compares UTF-8 bytes, JS compares UTF-16 units). */
function cmpStrings(a: string, b: string): number {
  return Buffer.compare(Buffer.from(a, 'utf8'), Buffer.from(b, 'utf8'));
}

const compareValues = (a: unknown, b: unknown): number => cmpStrings(goString(a), goString(b));

const DECIMAL = /^[+-]?(\d+\.?\d*|\.\d+)([eE][+-]?\d+)?$/;

function compareNumericValues(a: number, b: unknown): number {
  let bVal = 0;
  if (typeof b === 'number') bVal = b;
  else if (typeof b === 'bigint') bVal = Number(b);
  else {
    const s = goString(b);
    if (DECIMAL.test(s)) bVal = parseFloat(s);
  }
  return a < bVal ? -1 : a > bVal ? 1 : 0;
}

function compareOperator(op: string): (c: number) => boolean {
  switch (op) {
    case '>': return (c) => c > 0;
    case '<': return (c) => c < 0;
    case '>=': return (c) => c >= 0;
    case '<=': return (c) => c <= 0;
    default: return (c) => c === 0;
  }
}

const isObj = (v: unknown): v is Record<string, any> =>
  typeof v === 'object' && v !== null && !Array.isArray(v);

const floatFrom = (v: unknown): number | undefined => (typeof v === 'number' ? v : undefined);

function uintFrom(v: unknown, bits: 8 | 16 | 32): number {
  if (typeof v !== 'number' || !Number.isFinite(v)) return 0;
  const t = Math.trunc(v);
  return Number(BigInt.asUintN(bits, BigInt(t)));
}

function uintFromFilterValue(v: unknown): number {
  if (typeof v === 'number') return Math.trunc(v);
  if (typeof v === 'bigint') return Number(v);
  const s = goString(v).trim();
  if (s.toLowerCase().startsWith('0x') && /^0x[0-9a-f]+$/i.test(s)) return Number(BigInt(s));
  return /^\d+$/.test(s) ? Number(s) : 0;
}

const upperToken = (v: unknown): string => goString(v).trim().toUpperCase();

function parsePOSToken(v: unknown): number {
  const t = upperToken(v);
  return t === 'NOUN' ? 0x01 : t === 'VERB' ? 0x02 : uintFromFilterValue(v) & 0xff;
}

function parseTenseToken(v: unknown): number {
  switch (upperToken(v)) {
    case 'N/A': case 'NA': return 0;
    case 'PAST': return 1;
    case 'PRESENT': return 2;
    case 'FUTURE': return 3;
    default: return uintFromFilterValue(v) & 0xff;
  }
}

function parseDomainToken(v: unknown): number {
  switch (upperToken(v)) {
    case 'PROSE': return 0x1000;
    case 'MATH': return 0x2000;
    case 'CODE': return 0x3000;
    default: return uintFromFilterValue(v) & 0xffff;
  }
}

function bracketsIndex(payload: Doc): Doc[] {
  const v = payload['brackets_index'];
  return Array.isArray(v) ? v.filter(isObj) : [];
}

// ---- filter matching ---------------------------------------------------------

export function matchesFilter(payload: Doc, filter: Filter): boolean {
  const { key, operator: op, value } = filter;
  switch (key) {
    case 'z3_status':
      return isObj(payload.z3) ? goString(payload.z3.status) === goString(value) : false;
    case 'avg_temp_c': {
      const t = isObj(payload.thermo) ? floatFrom(payload.thermo.temp_celsius) : undefined;
      return t !== undefined ? compareOperator(op)(compareNumericValues(t, value)) : false;
    }
    case 'drift_score': {
      const d = isObj(payload.brackets) ? floatFrom(payload.brackets.max_drift) : undefined;
      return d !== undefined ? compareOperator(op)(compareNumericValues(d, value)) : false;
    }
    case 'bracket_type': {
      const want = goString(value).toUpperCase();
      if (bracketsIndex(payload).some((b) => goString(b.type).toUpperCase() === want)) return true;
      if (isObj(payload.brackets) && Array.isArray(payload.brackets.types)) {
        return payload.brackets.types.some((t: unknown) => goString(t).toUpperCase() === want);
      }
      return false;
    }
    case 'pos_tag': {
      const want = parsePOSToken(value);
      return bracketsIndex(payload).some((b) => uintFrom(b.pos_tag, 8) === want);
    }
    case 'tense': {
      const want = parseTenseToken(value);
      return bracketsIndex(payload).some((b) => uintFrom(b.tense, 8) === want);
    }
    case 'domain': {
      const want = parseDomainToken(value);
      return bracketsIndex(payload).some((b) => uintFrom(b.domain_sig, 16) === want);
    }
    case 'intent_flags': {
      const mask = uintFromFilterValue(value) & 0xff;
      return bracketsIndex(payload).some((b) => {
        const flags = uintFrom(b.intent_flags, 8);
        if (op === '&') return (flags & mask) === mask;
        if (op === '=') return flags === mask;
        return false;
      });
    }
    case 'lsh_salt': {
      const want = uintFrom(value, 32);
      return bracketsIndex(payload).some((b) => {
        const got = uintFrom(b.lsh_salt, 32);
        switch (op) {
          case '=': return got === want;
          case '!=': return got !== want;
          case '>': return compareNumericValues(got, want) > 0;
          case '<': return compareNumericValues(got, want) < 0;
          case '>=': return compareNumericValues(got, want) >= 0;
          case '<=': return compareNumericValues(got, want) <= 0;
          default: return false;
        }
      });
    }
    default: {
      if (!(key in payload)) return false;
      const val = payload[key];
      switch (op) {
        case '=': return goString(val) === goString(value);
        case '!=': return goString(val) !== goString(value);
        case '>': return compareValues(val, value) > 0;
        case '<': return compareValues(val, value) < 0;
        case '>=': return compareValues(val, value) >= 0;
        case '<=': return compareValues(val, value) <= 0;
        case 'CONTAINS': return goString(val).includes(goString(value));
        case 'STARTS_WITH': return goString(val).startsWith(goString(value));
        default: return false;
      }
    }
  }
}

export function matchesFilters(doc: Doc, filters: Filter[]): boolean {
  if (!isObj(doc.payload)) return false;
  return filters.every((f) => matchesFilter(doc.payload, f));
}

// ---- result shaping ------------------------------------------------------------

function extractBracketField(br: Doc, field: string): unknown {
  switch (field) {
    case 'golden_seed': return uintFrom(br.golden_seed, 32);
    case 'subsecond_us': return uintFrom(br.subsecond_us, 32);
    case 'pos_tag': return uintFrom(br.pos_tag, 8);
    case 'tense': return uintFrom(br.tense, 8);
    case 'domain_sig': case 'domain': return uintFrom(br.domain_sig, 16);
    case 'intent_flags': return uintFrom(br.intent_flags, 8);
    case 'drift_score': return floatFrom(br.drift_score);
    case 'lsh_salt': return uintFrom(br.lsh_salt, 32);
    case 'projections': case 'lsh_vector': return br.lsh_vector;
    default: return br[field];
  }
}

function projectBracketField(docs: Doc[], bracketField: string): Doc[] {
  const field = bracketField.trim().toLowerCase();
  const out: Doc[] = [];
  for (const doc of docs) {
    const payload = isObj(doc.payload) ? doc.payload : {};
    for (const br of bracketsIndex(payload)) {
      const v = extractBracketField(br, field);
      if (v === undefined || v === null) continue;
      out.push({ frame_id: doc.id, bracket_id: br.id, [field]: v });
    }
  }
  return out;
}

function projectModality(docs: Doc[], modalityType: string): Doc[] {
  const mod = modalityType.trim().toLowerCase();
  const out: Doc[] = [];
  for (const doc of docs) {
    const payload = isObj(doc.payload) ? doc.payload : {};
    const idx = bracketsIndex(payload);
    if (idx.length === 0) continue;
    const br = idx[0];
    const row: Doc = { frame_id: doc.id, modality: modalityType };
    switch (mod) {
      case 'vector': if (Array.isArray(br.lsh_vector)) row.value = br.lsh_vector; break;
      case 'seed': row.value = uintFrom(br.golden_seed, 32); break;
      case 'thermo': if (isObj(payload.thermo)) row.value = payload.thermo; break;
      case 'proof': row.value = null; break;
      default: break; // Go only serves a typed []float64 here, never decoded JSON
    }
    out.push(row);
  }
  return out;
}

// ---- executor ------------------------------------------------------------------

export class KNIRVQLExecutor implements QueryExecutor {
  constructor(private host: QueryHost, private parser: KNIRVQLParser = new KNIRVQLParser()) {}

  /** Parse and run a KNIRVQL string; throws on parse or execution errors. */
  async run(text: string, collection?: string): Promise<unknown> {
    const q = this.parser.parse(text);
    if (!q) throw new Error('empty query');
    return this.executeQuery(q, collection);
  }

  async execute(query: Query): Promise<QueryResult> {
    try {
      return { success: true, data: await this.executeQuery(query) };
    } catch (e: any) {
      return { success: false, error: e?.message ?? String(e) };
    }
  }

  async executeQuery(q: Query, collection?: string): Promise<unknown> {
    const target = collection ?? q.collection ?? '';
    switch (q.type) {
      case QueryType.Get:
      case QueryType.GetBracketField:
      case QueryType.GetModality:
        return this.executeGet(q, collection);
      case QueryType.Set:
        await this.host.insert(target, {
          id: q.key,
          entryType: EntryType.Auth,
          payload: { key: q.key, value: q.value },
        });
        return null;
      case QueryType.Delete:
        await this.host.delete(target, q.id!);
        return null;
      case QueryType.CreateIndex: {
        let type = IndexType.BTree;
        if (q.indexName === 'vector') type = IndexType.HNSW;
        else if (q.indexName === 'tag') type = IndexType.Tag;
        await this.host.createIndex(q.collection!, q.indexName!, type, q.fields ?? [], !!q.unique, '', {});
        return null;
      }
      case QueryType.DropIndex:
        await this.host.dropIndex(q.collection!, q.indexName!);
        return null;
      case QueryType.CreateCollection:
      case QueryType.DropCollection:
        return null;
      default:
        throw new Error('unsupported query type');
    }
  }

  private async executeGet(q: Query, explicit?: string): Promise<unknown> {
    const name = explicit || q.collection || (q.entryType === EntryType.Auth ? 'auth' : 'memory');
    const optimizer = new QueryOptimizer(name, this.host.getIndexes(name));
    const plan = optimizer.optimize(q);
    return this.executePlan(q, plan, name);
  }

  private async executePlan(q: Query, plan: QueryPlan, name: string): Promise<unknown> {
    switch (plan.scanType) {
      case ScanType.IndexScan:
        return this.indexScan(q, plan, name);
      case ScanType.IndexOnlyScan:
        // An id list cannot re-check extra index filters.
        return plan.indexFilters.length > 1 ? this.indexScan(q, plan, name) : this.indexOnlyScan(plan, name);
      default:
        return this.fullScan(q, plan, name);
    }
  }

  private async fullScan(q: Query, plan: QueryPlan, name: string): Promise<unknown> {
    const docs = await this.host.findAll(name);
    let results = docs.filter((d) => matchesFilters(d, plan.postFilters));
    if (plan.limit > 0 && results.length > plan.limit) results = results.slice(0, plan.limit);
    return this.shape(q, results);
  }

  private async indexScan(q: Query, plan: QueryPlan, name: string): Promise<unknown> {
    const ids = await this.host.queryIndex(name, plan.indexName, { value: plan.indexFilters[0].value });
    let results: Doc[] = [];
    for (const id of ids) {
      const doc = await this.host.find(name, id).catch(() => null);
      // Re-check every filter: the index narrows candidates, it does not prove a match.
      if (doc && matchesFilters(doc, plan.filters)) results.push(doc);
    }
    if (plan.limit > 0 && results.length > plan.limit) results = results.slice(0, plan.limit);
    return this.shape(q, results);
  }

  private async indexOnlyScan(plan: QueryPlan, name: string): Promise<string[]> {
    let ids = await this.host.queryIndex(name, plan.indexName, { value: plan.indexFilters[0].value });
    if (plan.limit > 0 && ids.length > plan.limit) ids = ids.slice(0, plan.limit);
    return ids;
  }

  private shape(q: Query, docs: Doc[]): unknown {
    if (q.type === QueryType.GetBracketField) return projectBracketField(docs, q.bracketField ?? '');
    if (q.type === QueryType.GetModality) return projectModality(docs, q.modalityType ?? '');
    return docs;
  }
}
