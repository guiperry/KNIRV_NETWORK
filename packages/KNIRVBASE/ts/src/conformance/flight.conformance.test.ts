// E9: Arrow IPC "flight" encoding must be the same bytes-on-the-wire format in Go and TS.
import { execFileSync, spawnSync } from 'child_process';
import * as fs from 'fs';
import * as path from 'path';
import { Bracket, encodeBracket, packSyntacticByte } from '../components/storage/nrv/bracket';
import {
  FlightClient, FlightServer, bracketsToFlightData, calcBracketDriftScore, flightDataToBrackets, parseFlightTicket,
} from '../components/storage/nrv/flight';

jest.setTimeout(180_000);

const GO_DIR = path.resolve(__dirname, '../../../go');
const BIN = process.env.NRVCONFORM_BIN;
const goAvailable = () => (BIN ? fs.existsSync(BIN) : spawnSync('go', ['version']).status === 0);
const goRun = (...args: string[]): string =>
  (BIN
    ? execFileSync(BIN, args, { maxBuffer: 64 << 20 })
    : execFileSync('go', ['run', './cmd/nrvconform', ...args], { cwd: GO_DIR, maxBuffer: 64 << 20 })
  ).toString().trim();

// Must match fixtureBracket() + flightBracket() in go/cmd/nrvconform/main.go.
function flightBracket(i: number): Bracket {
  const b: Bracket = {
    id: `fb-${i}`,
    projections: Uint8Array.from({ length: 32 }, (_, j) => (i * 31 + j) & 0xff),
    subSecondUS: 1000 + i,
    syntactic: packSyntacticByte(i + 1, i % 4, (i + 1) % 4),
    depHead: -3 * (i + 1),
    intentFlags: i + 7,
    domainSig: 0xa000 + i,
    goldenSeed: (0xdeadbe00 + i) >>> 0,
    memory: Uint8Array.from({ length: 14 }, (_, j) => 100 + i + j),
    lshSalt: ((i << 16) | 0x1234) >>> 0,
    reserved: new Uint8Array(17),
    frameId: `ff-${i}`,
    frameUnix: 0,
  };
  if (i % 2 === 0) {
    b.frameUnix = 1700000000 + i;
    b.meta = {
      id: b.id, type: i % 4 === 2 ? 'P' : 'I', anchorId: i % 4 === 2 ? 'fb-0' : undefined,
      offset: 0, driftScore: 0.125 * i,
    };
  }
  return b;
}

const maybe = goAvailable() || process.env.NRVCONFORM_REQUIRED ? describe : describe.skip;

maybe('Arrow IPC flight data: Go <-> TS', () => {
  const N = 5;

  it('TS decodes what Go encodes', () => {
    const got = flightDataToBrackets(new Uint8Array(Buffer.from(goRun('flight', 'encode', String(N)), 'base64')));
    expect(got).toHaveLength(N);
    got.forEach((b, i) => {
      const want = flightBracket(i);
      expect(Buffer.from(encodeBracket(b))).toEqual(Buffer.from(encodeBracket(want)));
      expect(b.id).toBe(`fb-${i}`);
      expect(b.frameId).toBe(`ff-${i}`);
      // even: explicit FrameUnix; odd: falls back to SubSecondUS
      expect(b.frameUnix).toBe(i % 2 === 0 ? 1700000000 + i : 1000 + i);
      expect(b.meta?.type).toBe(i % 2 === 0 ? want.meta!.type : 'I');
      expect(b.meta?.driftScore).toBe(i % 2 === 0 ? 0.125 * i : 0);
    });
  });

  it('Go decodes what TS encodes', () => {
    const data = bracketsToFlightData(Array.from({ length: N }, (_, i) => flightBracket(i)));
    const got = JSON.parse(goRun('flight', 'decode', Buffer.from(data).toString('base64')));
    expect(got).toHaveLength(N);
    got.forEach((b: any, i: number) => {
      const want = flightBracket(i);
      expect(b.ID).toBe(`fb-${i}`);
      expect(b.FrameID).toBe(`ff-${i}`);
      expect(b.FrameUnix).toBe(i % 2 === 0 ? 1700000000 + i : 1000 + i);
      expect(b.Projections).toEqual(Array.from(want.projections));
      expect(b.GoldenSeed).toBe(want.goldenSeed);
      expect(b.DepHead).toBe(want.depHead);
      expect(b.Meta.type).toBe(i % 2 === 0 ? want.meta!.type : 'I');
      expect(b.Meta.drift_score).toBe(i % 2 === 0 ? 0.125 * i : 0);
    });
  });

  it('empty input encodes to nothing in both directions', () => {
    expect(bracketsToFlightData([])).toHaveLength(0);
    expect(flightDataToBrackets(new Uint8Array(0))).toEqual([]);
    expect(goRun('flight', 'encode', '0')).toBe('');
  });

  it.each([[0, 1], [1, 2], [0, 3], [2, 2]])('drift score matches Go for brackets %i,%i', (i, j) => {
    expect(calcBracketDriftScore(flightBracket(i), flightBracket(j))).toBe(Number(goRun('flight', 'drift', String(i), String(j))));
  });
});

describe('FlightServer / FlightClient (no Go needed)', () => {
  it('parses tickets strictly', () => {
    expect(parseFlightTicket('gold.frames')).toEqual({ filter: 'gold', collection: 'frames' });
    expect(() => parseFlightTicket('gold')).toThrow();
    expect(() => parseFlightTicket('bogus.frames')).toThrow();
  });

  it('streams in batches of 1024 as independent IPC streams, passing the gold flag through', async () => {
    const calls: Array<[string, boolean]> = [];
    const server = new FlightServer({
      async *streamBrackets(collection: string, goldOnly: boolean) {
        calls.push([collection, goldOnly]);
        for (let i = 0; i < 1500; i++) yield flightBracket(i);
      },
    });
    const sizes: number[] = [];
    for await (const chunk of server.streamBatches('gold.frames')) sizes.push(flightDataToBrackets(chunk).length);
    expect(sizes).toEqual([1024, 476]);
    expect(calls).toEqual([['frames', true]]);
  });

  it('client decodes what the connection returns', async () => {
    const data = bracketsToFlightData([flightBracket(0), flightBracket(1)]);
    const client = new FlightClient({ read: async () => data });
    expect((await client.streamBrackets(undefined, 'all.frames')).map((b) => b.id)).toEqual(['fb-0', 'fb-1']);
  });
});
