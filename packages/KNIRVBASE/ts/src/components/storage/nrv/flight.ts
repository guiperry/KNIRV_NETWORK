import { Field, FixedSizeBinary, Float64, Int64, RecordBatch, Schema, Struct, Table, Utf8, makeBuilder, makeData, tableFromIPC, tableToIPC } from 'apache-arrow';
import { Bracket, BracketMeta, DeltaType, DeltaTypeI, BracketSize, encodeBracket, decodeBracket } from './bracket';

export const FLIGHT_SCHEMA_FIELDS = [
  { name: 'bracket_id', type: 'Utf8' },
  { name: 'frame_id', type: 'Utf8' },
  { name: 'frame_timestamp_unix', type: 'Int64' },
  { name: 'payload_asic', type: 'FixedSizeBinary(80)' },
  { name: 'drift_score', type: 'Double' },
  { name: 'bracket_type', type: 'Utf8' },
];

export interface FlightTicket {
  filter: 'gold' | 'all';
  collection: string;
}

export function parseFlightTicket(ticket: string): FlightTicket {
  const parts = ticket.split('.');
  if (parts.length !== 2) {
    throw new Error('invalid ticket format: expected "<gold|all>.<collection>"');
  }
  const [filter, collection] = parts;
  if (filter !== 'gold' && filter !== 'all') {
    throw new Error('invalid ticket filter: expected "gold" or "all"');
  }
  return { filter: filter as 'gold' | 'all', collection };
}

export function formatFlightTicket(ticket: FlightTicket): string {
  return `${ticket.filter}.${ticket.collection}`;
}

// ---- Arrow IPC (same schema and encoding as Go's BracketsToFlightData) -----------------

const BATCH_SIZE = 1024;

/** Schema: Go's bracketArrowSchema (non-nullable columns, FixedSizeBinary(80) payload). */
function bracketSchema(): Schema {
  return new Schema([
    new Field('bracket_id', new Utf8(), false),
    new Field('frame_id', new Utf8(), false),
    new Field('frame_timestamp_unix', new Int64(), false),
    new Field('payload_asic', new FixedSizeBinary(BracketSize), false),
    new Field('drift_score', new Float64(), false),
    new Field('bracket_type', new Utf8(), false),
  ]);
}

function frameUnix(b: Bracket): number {
  return b.frameUnix !== 0 ? b.frameUnix : b.subSecondUS;
}

function typeStr(b: Bracket): string {
  return b.meta && b.meta.type ? b.meta.type : DeltaTypeI;
}

/** Encode brackets as one Arrow IPC stream (empty input -> empty buffer, like Go). */
export function bracketsToFlightData(brackets: Bracket[]): Uint8Array {
  if (brackets.length === 0) {
    return new Uint8Array(0);
  }
  const schema = bracketSchema();
  const col = (i: number) => makeBuilder({ type: schema.fields[i].type, nullValues: [] });
  const [id, frame, unix, asic, drift, type] = [0, 1, 2, 3, 4, 5].map(col);
  for (const b of brackets) {
    id.append(b.id);
    frame.append(b.frameId);
    unix.append(BigInt(frameUnix(b)));
    asic.append(encodeBracket(b));
    drift.append(b.meta ? b.meta.driftScore : 0);
    type.append(typeStr(b));
  }
  const children = [id, frame, unix, asic, drift, type].map((bld) => bld.finish().flush());
  const data = makeData({ type: new Struct(schema.fields), length: brackets.length, nullCount: 0, children });
  const table = new Table(schema, [new RecordBatch(schema, data)]);
  return tableToIPC(table, 'stream');
}

/** Decode every record batch of an Arrow IPC stream into brackets. */
export function flightDataToBrackets(data: Uint8Array): Bracket[] {
  if (data.length === 0) {
    return [];
  }
  const table = tableFromIPC(data);
  const out: Bracket[] = [];
  for (const batch of table.batches) {
    const ids = batch.getChild('bracket_id')!;
    const frames = batch.getChild('frame_id')!;
    const unixes = batch.getChild('frame_timestamp_unix')!;
    const asics = batch.getChild('payload_asic')!;
    const drifts = batch.getChild('drift_score')!;
    const types = batch.getChild('bracket_type')!;
    for (let i = 0; i < batch.numRows; i++) {
      const b = decodeBracket(asics.get(i) as Uint8Array);
      b.id = ids.get(i) as string;
      b.frameId = frames.get(i) as string;
      b.frameUnix = Number(unixes.get(i) as bigint);
      const meta: BracketMeta = {
        id: b.id,
        type: types.get(i) as DeltaType,
        offset: 0,
        driftScore: drifts.get(i) as number,
      };
      b.meta = meta;
      out.push(b);
    }
  }
  return out;
}

export interface BracketSource {
  /** Brackets of a collection; `goldOnly` keeps only verified (Z3 VALID) frames. */
  streamBrackets(collection: string, goldOnly: boolean): AsyncIterable<Bracket>;
}

export class FlightServer {
  private schema: Map<string, string>;

  constructor(private storage: BracketSource) {
    this.schema = new Map();
    for (const field of FLIGHT_SCHEMA_FIELDS) {
      this.schema.set(field.name, field.type);
    }
  }

  getSchema(): Map<string, string> {
    return this.schema;
  }

  /** Brackets for a "<gold|all>.<collection>" ticket. */
  streamBrackets(ticket: string): AsyncIterable<Bracket> {
    const parsed = parseFlightTicket(ticket);
    return this.storage.streamBrackets(parsed.collection, parsed.filter === 'gold');
  }

  /**
   * Arrow IPC batches of up to 1024 brackets, each a complete IPC stream like the
   * messages Go's StreamBrackets sends.
   */
  async *streamBatches(ticket: string): AsyncGenerator<Uint8Array> {
    let batch: Bracket[] = [];
    for await (const b of this.streamBrackets(ticket)) {
      batch.push(b);
      if (batch.length >= BATCH_SIZE) {
        yield bracketsToFlightData(batch);
        batch = [];
      }
    }
    if (batch.length > 0) yield bracketsToFlightData(batch);
  }
}

export class FlightClient {
  constructor(private connection: { read(ticket: string): Promise<Uint8Array> | Uint8Array }) {}

  async streamBrackets(_ctx: unknown, ticket: string): Promise<Bracket[]> {
    parseFlightTicket(ticket);
    return flightDataToBrackets(await this.connection.read(ticket));
  }
}

/**
 * Squared Euclidean distance between the 16-dimensional LSH vectors of two
 * brackets (Go: CalcBracketDriftScore), computed with float32 intermediates.
 */
export function calcBracketDriftScore(current: Bracket, anchor: Bracket): number {
  if (!current || !anchor) {
    return 0;
  }
  const dim = (p: Uint8Array, i: number) => Math.fround((p[i * 2] | (p[i * 2 + 1] << 8)) / 65535.0);
  let sum = 0;
  for (let i = 0; i < 16; i++) {
    const diff = Math.fround(dim(current.projections, i) - dim(anchor.projections, i));
    sum += diff * diff;
  }
  return sum;
}
