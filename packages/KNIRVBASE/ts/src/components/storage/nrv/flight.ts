import { Bracket, BracketMeta, DeltaTypeI, DeltaTypeP, BracketSize, encodeBracket, decodeBracket } from './bracket';

export const FLIGHT_SCHEMA_FIELDS = [
  { name: 'bracket_id', type: 'Utf8' },
  { name: 'frame_id', type: 'Utf8' },
  { name: 'frame_timestamp_unix', type: 'Int64' },
  { name: 'payload_asic', type: 'Binary' },
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

const FLIGHT_MAGIC = 0x464C4948;
const FLIGHT_VERSION = 1;

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

export function bracketsToFlightData(brackets: Bracket[]): Uint8Array {
  if (brackets.length === 0) {
    return new Uint8Array(0);
  }

  const rowCount = brackets.length;
  const columnCount = FLIGHT_SCHEMA_FIELDS.length;

  const buffers: Uint8Array[] = [];

  buffers.push(new Uint8Array([
    (FLIGHT_MAGIC >> 0) & 0xFF,
    (FLIGHT_MAGIC >> 8) & 0xFF,
    (FLIGHT_MAGIC >> 16) & 0xFF,
    (FLIGHT_MAGIC >> 24) & 0xFF,
  ]));

  const versionBuf = new Uint8Array(4);
  new DataView(versionBuf.buffer).setUint32(0, FLIGHT_VERSION, true);
  buffers.push(versionBuf);

  const rowCountBuf = new Uint8Array(4);
  new DataView(rowCountBuf.buffer).setUint32(0, rowCount, true);
  buffers.push(rowCountBuf);

  const colCountBuf = new Uint8Array(4);
  new DataView(colCountBuf.buffer).setUint32(0, columnCount, true);
  buffers.push(colCountBuf);

  const columnOffsets: number[] = [];
  let dataStart = 4 + 4 + 4 + 4 + columnCount * 4;

  for (let col = 0; col < columnCount; col++) {
    columnOffsets.push(dataStart);
    const field = FLIGHT_SCHEMA_FIELDS[col];
    let colSize = 0;
    switch (field.type) {
      case 'Utf8':
        colSize = brackets.reduce((acc, b) => acc + getStringFieldLen(b, field.name) + 4, 0);
        break;
      case 'Int64':
      case 'Double':
        colSize = rowCount * 8;
        break;
      case 'Binary':
        colSize = rowCount * BracketSize;
        break;
    }
    dataStart += colSize;
  }

  for (let i = 0; i < columnCount; i++) {
    const offsetBuf = new Uint8Array(4);
    new DataView(offsetBuf.buffer).setUint32(0, columnOffsets[i], true);
    buffers.push(offsetBuf);
  }

  for (let col = 0; col < columnCount; col++) {
    const field = FLIGHT_SCHEMA_FIELDS[col];
    let colData: Uint8Array;

    switch (field.name) {
      case 'bracket_id': {
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
      case 'frame_id': {
        const totalLen = brackets.reduce((acc, b) => acc + b.frameId.length + 4, 0);
        const offsetBuf = new Uint8Array(rowCount * 4);
        const view = new DataView(offsetBuf.buffer);
        let offset = 0;
        for (let i = 0; i < rowCount; i++) {
          view.setUint32(i * 4, offset, true);
          offset += brackets[i].frameId.length + 4;
        }
        const dataBuf = new Uint8Array(offset);
        offset = 0;
        for (let i = 0; i < rowCount; i++) {
          const idBytes = new TextEncoder().encode(brackets[i].frameId);
          view.setUint32(offset, idBytes.length, true);
          dataBuf.set(idBytes, offset + 4);
          offset += idBytes.length + 4;
        }
        buffers.push(offsetBuf, dataBuf);
        break;
      }
      case 'frame_timestamp_unix': {
        const buf = new Uint8Array(rowCount * 8);
        const view = new DataView(buf.buffer);
        for (let i = 0; i < rowCount; i++) {
          view.setBigInt64(i * 8, BigInt(brackets[i].frameUnix || brackets[i].subSecondUS), true);
        }
        buffers.push(buf);
        break;
      }
      case 'payload_asic': {
        const buf = new Uint8Array(rowCount * BracketSize);
        for (let i = 0; i < rowCount; i++) {
          const encoded = encodeBracket(brackets[i]);
          buf.set(encoded, i * BracketSize);
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
          const type = brackets[i].meta?.type ?? DeltaTypeI;
          const typeBytes = new TextEncoder().encode(type);
          view.setUint32(offset, typeBytes.length, true);
          dataBuf.set(typeBytes, offset + 4);
          offset += typeBytes.length + 4;
        }
        buffers.push(offsetBuf, dataBuf);
        break;
      }
    }
  }

  return concatUint8Arrays(buffers);
}

function getStringFieldLen(b: Bracket, field: string): number {
  switch (field) {
    case 'bracket_id':
      return b.id.length;
    case 'frame_id':
      return b.frameId.length;
    case 'bracket_type':
      return b.meta?.type?.length ?? 1;
    default:
      return 0;
  }
}

function readStringField(data: Uint8Array, offset: Uint8Array, row: number): string {
  const view = new DataView(offset.buffer);
  const strOffset = view.getUint32(row * 4, true);
  const strLen = view.getUint32(strOffset, true);
  const strData = data.slice(strOffset + 4, strOffset + 4 + strLen);
  return new TextDecoder().decode(strData);
}

export function flightDataToBrackets(data: Uint8Array): Bracket[] {
  if (data.length === 0) {
    return [];
  }

  const view = new DataView(data.buffer, data.byteOffset);
  let pos = 0;

  const magic = view.getUint32(pos, true);
  if (magic !== FLIGHT_MAGIC) {
    throw new Error('invalid flight data magic');
  }

  pos += 4;
  const version = view.getUint32(pos, true);
  if (version !== FLIGHT_VERSION) {
    throw new Error(`unsupported flight version: ${version}`);
  }
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

  const brackets: Bracket[] = [];

  for (let row = 0; row < rowCount; row++) {
    const idColOffset = new Uint8Array(data.buffer, data.byteOffset + columnOffsets[0] + row * 4, 4);
    const frameIdColOffset = new Uint8Array(data.buffer, data.byteOffset + columnOffsets[1] + row * 4, 4);
    const timestampColOffset = data.byteOffset + columnOffsets[2] + row * 8;
    const asicColOffset = data.byteOffset + columnOffsets[3] + row * BracketSize;
    const driftColOffset = data.byteOffset + columnOffsets[4] + row * 8;
    const typeColOffset = new Uint8Array(data.buffer, data.byteOffset + columnOffsets[5] + row * 4, 4);

    const id = readStringField(data, idColOffset, 0);
    const frameId = readStringField(data, frameIdColOffset, 0);
    const frameUnix = view.getBigInt64(timestampColOffset, true);

    const asicData = new Uint8Array(data.buffer, asicColOffset, BracketSize);
    const decoded = decodeBracket(asicData);

    const driftScore = view.getFloat64(driftColOffset, true);

    const typeStr = readStringField(data, typeColOffset, 0);
    const type = typeStr as 'I' | 'P';

    const bracket: Bracket = {
      ...decoded,
      id,
      frameId,
      frameUnix: Number(frameUnix),
      meta: {
        id,
        type,
        offset: 0,
        driftScore,
      },
    };

    brackets.push(bracket);
  }

  return brackets;
}

export class FlightServer {
  private storage: any;
  private schema: Map<string, string>;

  constructor(storage: any) {
    this.storage = storage;
    this.schema = new Map();

    for (const field of FLIGHT_SCHEMA_FIELDS) {
      this.schema.set(field.name, field.type);
    }
  }

  getSchema(): Map<string, string> {
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

export function calcBracketDriftScore(current: Bracket, anchor: Bracket): number {
  if (!current || !anchor) {
    return 0;
  }

  let sum = 0;
  for (let i = 0; i < 16; i++) {
    const currentVal = (current.projections[i * 2] | (current.projections[i * 2 + 1] << 8)) / 65535.0;
    const anchorVal = (anchor.projections[i * 2] | (anchor.projections[i * 2 + 1] << 8)) / 65535.0;
    const diff = currentVal - anchorVal;
    sum += diff * diff;
  }
  return sum;
}
