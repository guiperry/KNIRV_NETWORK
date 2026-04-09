import { Bracket, BracketMeta } from './bracket';

export const FLIGHT_SCHEMA_FIELDS = [
  { name: 'frame_id', type: 'Utf8' },
  { name: 'lsh_salt', type: 'Uint32' },
  { name: 'subsecond_us', type: 'Uint32' },
  { name: 'asic_loops', type: 'Uint32' },
  { name: 'golden_seed', type: 'Uint32' },
  { name: 'drift_score', type: 'Double' },
  { name: 'bracket_type', type: 'Utf8' },
  { name: 'projections', type: 'Binary' },
  { name: 'frame_timestamp', type: 'Int64' },
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

export function bracketsToFlightData(brackets: Bracket[]): Uint8Array {
  const rowCount = brackets.length;
  const columnCount = FLIGHT_SCHEMA_FIELDS.length;
  
  const buffers: Uint8Array[] = [];
  
  for (let col = 0; col < columnCount; col++) {
    const field = FLIGHT_SCHEMA_FIELDS[col];
    let colData: Uint8Array;
    
    switch (field.name) {
      case 'frame_id': {
        const totalLen = brackets.reduce((acc, b) => acc + b.id.length + 4, 0);
        const offsetBuf = new Uint8Array(rowCount * 4);
        const view = new DataView(offsetBuf.buffer);
        let offset = 0;
        for (let i = 0; i < rowCount; i++) {
          view.setUint32(i * 4, offset, true);
          offset += brackets[i].id.length + 4;
        }
        const dataBuf = new Uint8Array(offset);
        offset = 0;
        for (let i = 0; i < rowCount; i++) {
          const idBytes = new TextEncoder().encode(brackets[i].id);
          view.setUint32(offset, idBytes.length, true);
          dataBuf.set(idBytes, offset + 4);
          offset += idBytes.length + 4;
        }
        buffers.push(offsetBuf, dataBuf);
        break;
      }
      case 'lsh_salt': {
        const buf = new Uint8Array(rowCount * 4);
        const view = new DataView(buf.buffer);
        for (let i = 0; i < rowCount; i++) {
          view.setUint32(i * 4, brackets[i].lshSalt, true);
        }
        buffers.push(buf);
        break;
      }
      case 'subsecond_us': {
        const buf = new Uint8Array(rowCount * 4);
        const view = new DataView(buf.buffer);
        for (let i = 0; i < rowCount; i++) {
          view.setUint32(i * 4, brackets[i].subSecondUS, true);
        }
        buffers.push(buf);
        break;
      }
      case 'asic_loops': {
        const buf = new Uint8Array(rowCount * 4);
        const view = new DataView(buf.buffer);
        for (let i = 0; i < rowCount; i++) {
          view.setUint32(i * 4, brackets[i].asicLoops, true);
        }
        buffers.push(buf);
        break;
      }
      case 'golden_seed': {
        const buf = new Uint8Array(rowCount * 4);
        const view = new DataView(buf.buffer);
        for (let i = 0; i < rowCount; i++) {
          view.setUint32(i * 4, brackets[i].goldenSeed, true);
        }
        buffers.push(buf);
        break;
      }
      case 'drift_score': {
        const buf = new Uint8Array(rowCount * 8);
        const view = new DataView(buf.buffer);
        for (let i = 0; i < rowCount; i++) {
          view.setFloat64(i * 8, brackets[i].meta?.driftScore ?? 0, true);
        }
        buffers.push(buf);
        break;
      }
      case 'bracket_type': {
        const totalLen = brackets.reduce((acc, b) => acc + (b.meta?.type?.length ?? 1) + 4, 0);
        const offsetBuf = new Uint8Array(rowCount * 4);
        const view = new DataView(offsetBuf.buffer);
        let offset = 0;
        for (let i = 0; i < rowCount; i++) {
          view.setUint32(i * 4, offset, true);
          offset += (brackets[i].meta?.type?.length ?? 1) + 4;
        }
        const dataBuf = new Uint8Array(offset);
        offset = 0;
        for (let i = 0; i < rowCount; i++) {
          const type = brackets[i].meta?.type ?? 'I';
          const typeBytes = new TextEncoder().encode(type);
          view.setUint32(offset, typeBytes.length, true);
          dataBuf.set(typeBytes, offset + 4);
          offset += typeBytes.length + 4;
        }
        buffers.push(offsetBuf, dataBuf);
        break;
      }
      case 'projections': {
        const totalLen = rowCount * 64;
        const offsetBuf = new Uint8Array(rowCount * 4);
        const view = new DataView(offsetBuf.buffer);
        let offset = 0;
        for (let i = 0; i < rowCount; i++) {
          view.setUint32(i * 4, offset, true);
          offset += 64;
        }
        const dataBuf = new Uint8Array(64 * rowCount);
        for (let i = 0; i < rowCount; i++) {
          dataBuf.set(brackets[i].projections.slice(0, 64), i * 64);
        }
        buffers.push(offsetBuf, dataBuf);
        break;
      }
      case 'frame_timestamp': {
        const buf = new Uint8Array(rowCount * 8);
        const view = new DataView(buf.buffer);
        for (let i = 0; i < rowCount; i++) {
          view.setBigInt64(i * 8, BigInt(brackets[i].subSecondUS), true);
        }
        buffers.push(buf);
        break;
      }
    }
  }
  
  return concatUint8Arrays(buffers);
}

export function flightDataToBrackets(data: Uint8Array): Bracket[] {
  const brackets: Bracket[] = [];
  
  const view = new DataView(data.buffer, data.byteOffset);
  let pos = 0;
  
  const magic = view.getUint32(0, true);
  if (magic !== 0x464C4948) {
    throw new Error('invalid flight data magic');
  }
  
  pos += 4;
  const version = view.getUint32(pos, true);
  pos += 4;
  const rowCount = view.getUint32(pos, true);
  pos += 4;
  const columnCount = view.getUint32(pos, true);
  pos += 4;
  
  const columnOffsets: number[] = [];
  for (let i = 0; i < columnCount; i++) {
    columnOffsets.push(view.getUint32(pos, true));
    pos += 4;
  }
  
  for (let row = 0; row < rowCount; row++) {
    const bracket: Bracket = {
      id: '',
      lshSalt: 0,
      projections: new Uint8Array(64),
      subSecondUS: 0,
      asicLoops: 0,
      goldenSeed: 0,
      meta: undefined,
    };
    
    bracket.lshSalt = view.getUint32(pos + columnOffsets[1] + row * 4, true);
    bracket.subSecondUS = view.getUint32(pos + columnOffsets[2] + row * 4, true);
    bracket.asicLoops = view.getUint32(pos + columnOffsets[3] + row * 4, true);
    bracket.goldenSeed = view.getUint32(pos + columnOffsets[4] + row * 4, true);
    
    if (bracket.meta) {
      bracket.meta.driftScore = view.getFloat64(pos + columnOffsets[5] + row * 8, true);
    }
    
    brackets.push(bracket);
  }
  
  return brackets;
}

function concatUint8Arrays(arrays: Uint8Array[]): Uint8Array {
  const totalLen = arrays.reduce((acc, a) => acc + a.length, 0);
  const result = new Uint8Array(totalLen);
  let offset = 0;
  for (const arr of arrays) {
    result.set(arr, offset);
    offset += arr.length;
  }
  return result;
}

export class FlightServer {
  private storage: any;
  private schema: Map<string, any>;

  constructor(storage: any) {
    this.storage = storage;
    this.schema = new Map();
    
    for (const field of FLIGHT_SCHEMA_FIELDS) {
      this.schema.set(field.name, field.type);
    }
  }

  async getSchema(collection: string): Promise<any> {
    return this.schema;
  }

  async streamBrackets(ticket: string): Promise<AsyncIterableIterator<Bracket>> {
    const parsed = parseFlightTicket(ticket);
    const collection = parsed.collection;
    const goldOnly = parsed.filter === 'gold';
    
    const reader = await this.storage.getReader(collection);
    if (!reader) {
      throw new Error(`collection not found: ${collection}`);
    }

    const registry = reader.getRegistry();
    const brackets: Bracket[] = [];
    
    for (const entry of registry.frames) {
      if ((entry as any).tombstone !== undefined) {
        continue;
      }
      const frame = reader.getFrame(entry.id);
      if (frame && (frame as any).brackets) {
        for (const bracket of (frame as any).brackets) {
          if (!goldOnly || (bracket.meta && bracket.meta.type === 'P')) {
            brackets.push(bracket);
          }
        }
      }
    }
    
    class BracketIterator implements AsyncIterableIterator<Bracket> {
      private index = 0;

      async next(): Promise<IteratorResult<Bracket>> {
        if (this.index >= brackets.length) {
          return { done: true, value: undefined };
        }
        return { done: false, value: brackets[this.index++] };
      }

      [Symbol.asyncIterator](): AsyncIterableIterator<Bracket> {
        return this;
      }
    }

    return new BracketIterator();
  }
}

export class FlightClient {
  private connection: any;

  constructor(conn: any) {
    this.connection = conn;
  }

  async streamBrackets(ctx: any, ticket: string): Promise<Bracket[]> {
    const parsed = parseFlightTicket(ticket);
    const data = await this.connection.read(ticket);
    return flightDataToBrackets(data);
  }
}