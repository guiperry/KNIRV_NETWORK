import { EntryType } from '../types/types';
import { Index, IndexType } from '../storage/index';

const kw = (s: string): string => s.toUpperCase();

/** Strip any leading/trailing characters in `chars` (Go's strings.Trim). */
function trimChars(s: string, chars: string): string {
  let a = 0;
  let b = s.length;
  while (a < b && chars.includes(s[a])) a++;
  while (b > a && chars.includes(s[b - 1])) b--;
  return s.slice(a, b);
}

const DECIMAL = /^[+-]?(\d+\.?\d*|\.\d+)([eE][+-]?\d+)?$/;

/** Value typing identical to Go's parseFilterValue (numbers are always floats). */
export function parseFilterValue(raw: string): string | number | boolean {
  const v = trimChars(raw, '"');
  if (v === 'true') return true;
  if (v === 'false') return false;
  if (DECIMAL.test(v)) return parseFloat(v);
  if (/^0x[0-9a-f]+$/i.test(v)) return Number(BigInt(v));
  return v;
}

function parseVector(s: string): number[] {
  const out: number[] = [];
  for (const vp of trimChars(s, '[]').split(',')) {
    const t = vp.trim();
    if (DECIMAL.test(t)) out.push(parseFloat(t));
  }
  return out;
}

// KNIRVQLParser parses KNIRVQL queries
export class KNIRVQLParser {
  /**
   * Parse parses a KNIRVQL query
   */
  parse(query: string): Query | null {
    query = query.trim();
    const parts = query.split(/\s+/);
    if (query === '' || parts.length === 0) {
      throw new Error('empty query');
    }

    const cmd = parts[0].toUpperCase();
    switch (cmd) {
      case 'GET':
        return this.parseGet(parts.slice(1));
      case 'SET':
        return this.parseSet(parts.slice(1));
      case 'DELETE':
        return this.parseDelete(parts.slice(1));
      case 'CREATE':
        return this.parseCreate(parts.slice(1));
      case 'DROP':
        return this.parseDrop(parts.slice(1));
      default:
        throw new Error(`unknown command: ${cmd}`);
    }
  }

  private parseGet(parts: string[]): Query | null {
    if (parts.length < 1) {
      throw new Error('invalid GET query');
    }

    // Grammar shared with Go (internal/query/knirvql.go): keywords are
    // case-insensitive, WHERE filters are conjunctive (OR is accepted but
    // evaluated as AND), values are typed by parseFilterValue.
    const first = parts[0];
    let entryType = first.toUpperCase();
    let modalityType = '';
    let bracketField = '';
    if (entryType.startsWith('MEMORY.MODALITY(') && first.endsWith(')')) {
      modalityType = first.slice('MEMORY.MODALITY('.length, -1);
      entryType = 'MEMORY';
    }
    if (entryType.startsWith('MEMORY.BRACKET(') && first.endsWith(')')) {
      bracketField = first.slice('MEMORY.BRACKET('.length, -1);
      entryType = 'MEMORY';
    }
    if (entryType !== 'MEMORY' && entryType !== 'AUTH') {
      throw new Error(`invalid entry type: ${entryType}`);
    }

    let type = QueryType.Get;
    if (bracketField !== '') type = QueryType.GetBracketField;
    else if (modalityType !== '') type = QueryType.GetModality;

    let collection = '';
    const filters: Filter[] = [];
    let similarTo: number[] = [];
    let limit = 0;

    let i = 1;
    if (i < parts.length && kw(parts[i]) === 'FROM') {
      i++;
      if (i < parts.length) {
        collection = parts[i];
        i++;
      }
    }

    let inWhere = false;
    while (i < parts.length) {
      const tok = kw(parts[i]);
      if (tok === 'WHERE') {
        inWhere = true;
        i++;
      } else if (tok === 'SIMILAR' && i + 1 < parts.length && kw(parts[i + 1]) === 'TO') {
        i += 2;
        if (i < parts.length) {
          similarTo = parseVector(parts[i]);
          i++;
        }
        inWhere = false;
      } else if (tok === 'LIMIT') {
        i++;
        if (i < parts.length) {
          if (/^[+-]?\d+$/.test(parts[i])) limit = parseInt(parts[i], 10);
          i++;
        }
        inWhere = false;
      } else if (inWhere && tok === 'WITH' && i + 1 < parts.length && kw(parts[i + 1]) === 'TAGS') {
        i += 2;
        if (i < parts.length && parts[i].startsWith('(')) {
          for (const t of trimChars(parts[i], '()').split(',')) {
            filters.push({ key: 'tags', operator: 'CONTAINS', value: t.trim() });
          }
          i++;
        }
      } else if (inWhere && (tok === 'AND' || tok === 'OR')) {
        i++;
      } else if (inWhere && i + 2 < parts.length && kw(parts[i + 1]) === 'SIMILAR' && kw(parts[i + 2]) === 'TO') {
        i++; // "key SIMILAR TO [..]": the key is informational
      } else if (inWhere && i + 2 < parts.length) {
        let op = parts[i + 1];
        const up = op.toUpperCase();
        if (up === 'CONTAINS' || up === 'STARTS_WITH') op = up;
        filters.push({ key: parts[i], operator: op, value: parseFilterValue(parts[i + 2]) });
        i += 3;
      } else {
        i++;
      }
    }

    return {
      type,
      entryType: entryType as EntryType,
      collection,
      filters,
      similarTo,
      limit,
      modalityType,
      bracketField,
    };
  }

  private parseSet(parts: string[]): Query | null {
    if (parts.length < 3 || parts[1] !== '=') {
      throw new Error('invalid SET query');
    }

    const key = parts[0];
    const value = trimChars(parts.slice(2).join(' '), '"');

    return {
      type: QueryType.Set,
      key,
      value,
      entryType: EntryType.Auth,
    };
  }

  private parseDelete(parts: string[]): Query | null {
    if (parts.length < 4 || parts[0].toUpperCase() !== 'WHERE' || parts[1] !== 'id' || parts[2] !== '=') {
      throw new Error('invalid DELETE query');
    }

    const id = trimChars(parts[3], '"');

    return {
      type: QueryType.Delete,
      id,
    };
  }

  private parseCreate(parts: string[]): Query | null {
    if (parts.length < 2) {
      throw new Error('invalid CREATE command');
    }

    const subCmd = parts[0].toUpperCase();
    switch (subCmd) {
      case 'INDEX':
        return this.parseCreateIndex(parts.slice(1));
      case 'COLLECTION':
        return this.parseCreateCollection(parts.slice(1));
      default:
        throw new Error(`unknown CREATE command: ${subCmd}`);
    }
  }

  private parseCreateIndex(parts: string[]): Query | null {
    if (parts.length < 3) {
      throw new Error('invalid CREATE INDEX command');
    }

    // Parse collection:index format
    const indexRef = parts[0];
    const indexParts = indexRef.split(':');
    if (indexParts.length !== 2) {
      throw new Error('invalid index reference, expected collection:index');
    }
    const collection = indexParts[0];
    const indexName = indexParts[1];

    if (parts[1].toUpperCase() !== 'ON') {
      throw new Error('expected ON after index name');
    }

    if (parts[2] !== collection) {
      throw new Error('collection mismatch in index definition');
    }

    const fields: string[] = [];
    let unique = false;

    let i = 3;
    if (i < parts.length && parts[i].startsWith('(')) {
      // Handle (field1,field2) format
      const fieldStr = parts[i].replace(/[\(\)]/g, '');
      if (fieldStr) {
        const fieldParts = fieldStr.split(',');
        for (const f of fieldParts) {
          const field = f.trim();
          if (field) {
            fields.push(field);
          }
        }
      }
      i++;
    }

    if (i < parts.length && parts[i].toUpperCase() === 'UNIQUE') {
      unique = true;
    }

    return {
      type: QueryType.CreateIndex,
      indexName,
      collection,
      fields,
      unique,
    };
  }

  private parseCreateCollection(parts: string[]): Query | null {
    if (parts.length < 1) {
      throw new Error('invalid CREATE COLLECTION command');
    }

    const collectionName = parts[0];

    return {
      type: QueryType.CreateCollection,
      collection: collectionName,
    };
  }

  private parseDrop(parts: string[]): Query | null {
    if (parts.length < 2) {
      throw new Error('invalid DROP command');
    }

    const subCmd = parts[0].toUpperCase();
    switch (subCmd) {
      case 'INDEX':
        return this.parseDropIndex(parts.slice(1));
      case 'COLLECTION':
        return this.parseDropCollection(parts.slice(1));
      default:
        throw new Error(`unknown DROP command: ${subCmd}`);
    }
  }

  private parseDropIndex(parts: string[]): Query | null {
    if (parts.length < 1) {
      throw new Error('invalid DROP INDEX command');
    }

    // Parse collection:index format
    const indexRef = parts[0];
    const indexParts = indexRef.split(':');
    if (indexParts.length !== 2) {
      throw new Error('invalid index reference, expected collection:index');
    }

    return {
      type: QueryType.DropIndex,
      collection: indexParts[0],
      indexName: indexParts[1],
    };
  }

  private parseDropCollection(parts: string[]): Query | null {
    if (parts.length < 1) {
      throw new Error('invalid DROP COLLECTION command');
    }

    const collectionName = parts[0];

    return {
      type: QueryType.DropCollection,
      collection: collectionName,
    };
  }
}

// Query represents a parsed KNIRVQL query
export interface Query {
  type: QueryType;
  entryType?: EntryType;
  collection?: string;
  id?: string;
  key?: string;
  value?: string;
  filters?: Filter[];
  similarTo?: number[];
  limit?: number;
  sort?: { field: string; ascending: boolean };

  // Index management
  indexName?: string;
  fields?: string[];
  unique?: boolean;

  // MEMORY.MODALITY(x) / MEMORY.BRACKET(x) projections
  modalityType?: string;
  bracketField?: string;
}

// QueryType enum
export enum QueryType {
  // Numeric values mirror Go's QueryType iota.
  Get = 0,
  Set = 1,
  Delete = 2,
  CreateIndex = 3,
  CreateCollection = 4,
  DropIndex = 5,
  DropCollection = 6,
  GetModality = 7,
  GetBracketField = 8,
}

// Filter for WHERE clauses
export interface Filter {
  key: string;
  operator: string;
  value: any;
}

// Query execution interfaces
export interface QueryResult {
  success: boolean;
  data?: any;
  error?: string;
}

export interface QueryExecutor {
  execute(query: Query): Promise<QueryResult>;
}