// E8: FrameTicker (I/P deltas, drift, memory XOR) and WAL format parity with Go.
import { execFileSync, spawnSync } from 'child_process';
import * as fs from 'fs';
import * as os from 'os';
import * as path from 'path';
import { Bracket } from '../components/storage/nrv/bracket';
import { FrameTicker } from '../components/storage/nrv/ticker';
import { NRVWriter } from '../components/storage/nrv/writer';
import { WAL } from '../components/storage/nrv/wal';
import { packSyntacticByte } from '../components/storage/nrv/bracket';

jest.setTimeout(180_000);

const GO_DIR = path.resolve(__dirname, '../../../go');
const BIN = process.env.NRVCONFORM_BIN;
const goAvailable = () => (BIN ? fs.existsSync(BIN) : spawnSync('go', ['version']).status === 0);
const goRun = (...args: string[]): string =>
  (BIN
    ? execFileSync(BIN, args, { maxBuffer: 256 << 20 })
    : execFileSync('go', ['run', './cmd/nrvconform', ...args], { cwd: GO_DIR, maxBuffer: 256 << 20 })
  ).toString().trim();

const f32 = Math.fround;

// Must match tickerBracket() in go/cmd/nrvconform/main.go.
function tickerBracket(i: number): Bracket {
  const projections = new Uint8Array(32);
  for (let j = 0; j < 32; j++) {
    projections[j] = (10 + (i % 3) + j) & 0xff;
    if (i % 40 === 39) projections[j] = (255 - j * 7) & 0xff;
  }
  return {
    id: `t-${String(i).padStart(3, '0')}`,
    projections,
    subSecondUS: 5000 + i,
    syntactic: packSyntacticByte(i % 15, i % 4, i % 3),
    depHead: (i % 20) - 10,
    intentFlags: i & 0xff,
    domainSig: 0x1000 * ((i % 3) + 1),
    goldenSeed: Math.imul(0x9e3779b1, i + 1) >>> 0,
    memory: new Uint8Array(14),
    lshSalt: ((i << 16) | 0xbeef) >>> 0,
    reserved: new Uint8Array(17),
    frameId: '',
    frameUnix: 0,
  };
}

// Frame ids/times are random; everything else must be identical.
function comparable(dump: any) {
  const frames = dump.registry.frames.map((f: any) => ({
    // Go prints float32 shortest-form (50.37); TS holds the exact float32 -> compare as float32.
    thermo: Object.fromEntries(Object.entries(f.thermo).map(([k, v]) => [k, Math.fround(v as number)])), linguistic: f.linguistic, z3: f.z3,
    count: f.brackets.count, length: f.brackets.length,
    index: f.bracket_index.map((m: any) => ({ ...m, drift_score: Number(m.drift_score.toFixed(12)) })),
  }));
  const brackets = dump.registry.frames.flatMap((f: any) =>
    dump.brackets[f.id].map((b: any) => { const { FrameID, FrameUnix, ...rest } = b; return rest; }));
  return { frames, brackets, metrics: dump.registry.global_metrics };
}

const maybe = goAvailable() || process.env.NRVCONFORM_REQUIRED ? describe : describe.skip;

maybe('FrameTicker + WAL', () => {
  let dir: string;
  beforeAll(() => { dir = fs.mkdtempSync(path.join(os.tmpdir(), 'ticker-')); });
  afterAll(() => fs.rmSync(dir, { recursive: true, force: true }));

  it('TS ticker produces the same frames as the Go ticker', async () => {
    const goFile = path.join(dir, 'go-ticker.nrv');
    goRun('ticker-write', goFile);

    const tsFile = path.join(dir, 'ts-ticker.nrv');
    const w = await NRVWriter.create(tsFile);
    const ft = new FrameTicker(w, 3_600_000);
    ft.setLinguistic('tok', 'unit');
    for (let i = 0; i < 120; i++) {
      ft.appendBracket(tickerBracket(i), {
        // Go evaluates these in float32 arithmetic: round after every operation.
        avgTempC: f32(50 + f32(i * f32(0.37))), peakVoltV: f32(1.0 + f32(i * f32(0.013))), clockMHz: f32(400 + i),
      });
      if (i % 7 === 6) await ft.flush();
    }
    await ft.stop();
    await w.close();

    const goDump = JSON.parse(goRun('dump', goFile));
    const tsDump = JSON.parse(goRun('dump', tsFile)); // Go reads the TS-written file
    const a = comparable(goDump);
    const b = comparable(tsDump);
    expect(b.frames.length).toBe(a.frames.length);
    expect(a.frames.length).toBe(18); // 17 flushes of 7 + the final 1
    expect(a.frames.flatMap((f: any) => f.index.map((m: any) => m.type))).toContain('P');
    expect(a.frames.flatMap((f: any) => f.index.map((m: any) => m.type)).filter((t: string) => t === 'I').length).toBeGreaterThan(2);
    expect(b.brackets).toEqual(a.brackets);
    expect(b.frames).toEqual(a.frames);
    expect(b.metrics.total_bracket_count).toBe(120);
  });

  it('WAL: Go reads TS entries and TS reads Go-shaped entries', async () => {
    const walFile = path.join(dir, 'x.wal');
    const wal = new WAL(walFile);
    await wal.begin({ frameId: 'f1', lastGoodLength: 5242892, committed: false });
    await wal.begin({ frameId: 'f2', lastGoodLength: 5243000, committed: false });
    expect(fs.readFileSync(walFile, 'utf8')).toContain('"frame_id":"f1"');
    expect(goRun('wal-recover', walFile)).toBe('5242892');
    await wal.commit('f1');
    expect(goRun('wal-recover', walFile)).toBe('5243000');
    await wal.commit('f2');
    expect(goRun('wal-recover', walFile)).toBe('-1');
    expect(await wal.recover()).toBe(-1);

    // Go-format file with a torn last line: both skip it and still see f3.
    fs.writeFileSync(walFile,
      '{"frame_id":"f3","last_good_length":777,"committed":false}\n{"frame_id":"f4","last_good_len');
    expect(goRun('wal-recover', walFile)).toBe('777');
    expect(await new WAL(walFile).recover()).toBe(777);

    // Legacy camelCase lines from older TS builds are still understood.
    fs.writeFileSync(walFile, '{"frameId":"old","lastGoodLength":42,"committed":false}\n');
    expect(await new WAL(walFile).recover()).toBe(42);
  });
});
