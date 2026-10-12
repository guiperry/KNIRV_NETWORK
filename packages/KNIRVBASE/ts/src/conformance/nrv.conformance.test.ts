// Cross-implementation conformance: Go (source of truth) <-> TypeScript.
// The Go half is cmd/nrvconform. Set NRVCONFORM_BIN to a prebuilt binary, or
// leave it unset to `go run` it. The suite skips only when Go is unavailable
// and NRVCONFORM_REQUIRED is not set (CI sets it).
import { execFileSync, spawnSync } from 'child_process';
import * as fs from 'fs';
import * as os from 'os';
import * as path from 'path';
import { NRVReader } from '../components/storage/nrv/reader';
import { NRVWriter } from '../components/storage/nrv/writer';
import {
  Bracket,
  BracketMeta,
  decodeSyntactic,
  encodeBracket,
  encodeSyntactic,
  packSyntacticByte,
} from '../components/storage/nrv/bracket';
import { registryFromGoJSON, registryToGoJSON, registryToPaddedBuffer } from '../components/storage/nrv/registry_json';
import { createDefaultRegistry } from '../components/storage/nrv/codec';
import { pqcSigner } from '../components/storage/nrv/writer';
import { generatePQCKeyPair, loadPQCKeyPair, marshalWithPrivateKeys } from '../components/crypto/pqc/keys';

jest.setTimeout(180_000); // `go run` compiles on first use

const GO_DIR = path.resolve(__dirname, '../../../go');
const BIN = process.env.NRVCONFORM_BIN;

function goAvailable(): boolean {
  if (BIN) return fs.existsSync(BIN);
  return spawnSync('go', ['version']).status === 0;
}

function goRun(...args: string[]): string {
  if (BIN) return execFileSync(BIN, args, { maxBuffer: 64 << 20 }).toString().trim();
  return execFileSync('go', ['run', './cmd/nrvconform', ...args], { cwd: GO_DIR, maxBuffer: 64 << 20 }).toString().trim();
}

function fixtureBracket(i: number): Bracket {
  return {
    id: '',
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
    frameId: '',
    frameUnix: 0,
  };
}

describe('registry JSON schema (Go snake_case)', () => {
  it('round-trips and uses Go key names', () => {
    const r = createDefaultRegistry('ds-1', 'k');
    r.frames.push({
      id: 'f', offset: 5242892, length: 160, verified: true, ergoRank: 0.5,
      modalities: { vector: { offset: 0, length: 48 }, seed: { offset: 48, length: 32 }, thermo: { offset: 80, length: 16 } },
      timestampUnix: 1700000000, bracketCount: 2,
      bracketIndex: [{ id: 'a', type: 'I', offset: 0, driftScore: 0 }, { id: 'b', type: 'P', anchorId: 'a', offset: 80, driftScore: 0.5 }],
    });
    const json = JSON.parse(registryToGoJSON(r));
    expect(Object.keys(json)).toEqual([
      'version', 'dataset_id', 'dataset_version', 'chunk0_length', 'frame_count',
      'tombstone_count', 'global_metrics', 'frames', 'pqc_manifest',
    ]);
    expect(json.frames[0].brackets).toEqual({ count: 2, offset: 5242892, length: 160 });
    expect(json.frames[0].bracket_index[1].anchor_id).toBe('a');
    expect(json.frames[0].tombstone).toBeNull();
    const back = registryFromGoJSON(JSON.stringify(json));
    expect(back.frames[0].bracketIndex?.[1].anchorId).toBe('a');
    expect(back.frames[0].verified).toBe(true);
  });

  it('E4: pads by byte length and refuses to overflow', () => {
    const r = createDefaultRegistry('ünïcödé-🙂', 'k'); // multi-byte: UTF-16 length != byte length
    const buf = Buffer.from(registryToPaddedBuffer(r));
    expect(buf.length).toBe(5 * 1024 * 1024);
    const text = registryToGoJSON(r);
    expect(buf.subarray(0, Buffer.byteLength(text)).toString('utf8')).toBe(text);
    expect(buf[Buffer.byteLength(text)]).toBe(0);

    r.pqcManifest.frameSignatures['x'] = 'a'.repeat(5 * 1024 * 1024);
    expect(() => registryToPaddedBuffer(r)).toThrow('nrv: registry exceeds padding');
  });
});

describe('E3: syntactic packing', () => {
  it.each([-128, -3, -1, 0, 1, 127])('keeps depHead %i', (d) => {
    const v = encodeSyntactic({ syntactic: 0xa5, depHead: d });
    expect(decodeSyntactic(v)).toEqual({ syntactic: 0xa5, depHead: d });
  });
  it('matches Go bit layout for negative depHead', () => {
    // Go: uint32(int16(-1)) << 16 | 0xA5
    expect(encodeSyntactic({ syntactic: 0xa5, depHead: -1 })).toBe(0xffff00a5);
  });
});

const maybe = goAvailable() || process.env.NRVCONFORM_REQUIRED ? describe : describe.skip;

maybe('Go <-> TS .nrv files', () => {
  let dir: string;
  beforeAll(() => { dir = fs.mkdtempSync(path.join(os.tmpdir(), 'nrvconform-')); });
  afterAll(() => fs.rmSync(dir, { recursive: true, force: true }));

  it('TS reads a Go-written file', async () => {
    const file = path.join(dir, 'go.nrv');
    goRun('write', file);
    const reader = await NRVReader.open(file);
    const reg = reader.getRegistry();
    expect(reg.frameCount).toBe(3);
    expect(reg.globalMetrics.totalBracketCount).toBe(6);
    expect(reg.frames.map((f) => f.id)).toEqual(['frame-0', 'frame-1', 'frame-2']);
    expect(reg.frames[1].linguistic).toEqual({ token: 'tok', unit: 'u' });
    expect(reg.frames[1].thermo?.avgTempC).toBe(61);

    for (let f = 0; f < 3; f++) {
      const bs = reader.getBrackets(`frame-${f}`)!;
      expect(bs).toHaveLength(2);
      bs.forEach((b, k) => {
        const want = fixtureBracket(f * 2 + k);
        expect(Buffer.from(encodeBracket(b))).toEqual(Buffer.from(encodeBracket(want)));
        expect(b.depHead).toBe(want.depHead);
        expect(b.meta?.id).toBe(`b-${f}-${k}`);
      });
    }
  });

  it('Go reads a TS-written file', async () => {
    const file = path.join(dir, 'ts.nrv');
    const w = await NRVWriter.create(file);
    for (let f = 0; f < 3; f++) {
      const idx: BracketMeta[] = [0, 1].map((k) => ({
        id: `b-${f}-${k}`, type: 'I', offset: k * 80, driftScore: 0.25 * k,
      }));
      const buf = Buffer.concat([0, 1].map((k) => Buffer.from(encodeBracket(fixtureBracket(f * 2 + k)))));
      await w.appendBrackets(`frame-${f}`, buf, idx, { avgTempC: 60 + f, peakVoltV: 1.1, clockMHz: 500 }, { token: 'tok', unit: 'u' });
    }
    await w.close();

    const dump = JSON.parse(goRun('dump', file));
    expect(dump.registry.frame_count).toBe(3);
    expect(dump.registry.global_metrics.total_bracket_count).toBe(6);
    for (let f = 0; f < 3; f++) {
      const got = dump.brackets[`frame-${f}`];
      expect(got).toHaveLength(2);
      got.forEach((b: any, k: number) => {
        const want = fixtureBracket(f * 2 + k);
        expect(b.DepHead).toBe(want.depHead);
        expect(b.GoldenSeed).toBe(want.goldenSeed);
        expect(b.DomainSig).toBe(want.domainSig);
        expect(b.Projections).toEqual(Array.from(want.projections));
        expect(b.ID).toBe(`b-${f}-${k}`);
      });
    }
  });

  it('signed frames: Go signs -> TS verifies; TS signs -> Go verifies', async () => {
    const goKey = path.join(dir, 'sig-go-key.json');
    fs.writeFileSync(goKey, goRun('pqc', 'keygen'));
    const goFile = path.join(dir, 'go-signed.nrv');
    goRun('write-signed', goFile, goKey);
    const reader = await NRVReader.open(goFile);
    const kp = loadPQCKeyPair(fs.readFileSync(goKey, 'utf8'));
    expect(reader.getRegistry().pqcManifest.keyId).toBe(kp.id);
    for (let f = 0; f < 3; f++) {
      expect(reader.verifyFrame(`frame-${f}`, kp)).toBe(true);
    }
    expect(reader.verifyFrame('frame-0', generatePQCKeyPair('other', 'signature'))).toBe(false);

    const tsKp = generatePQCKeyPair('ts', 'signature');
    const tsKey = path.join(dir, 'sig-ts-key.json');
    fs.writeFileSync(tsKey, marshalWithPrivateKeys(tsKp));
    const tsFile = path.join(dir, 'ts-signed.nrv');
    const w = await NRVWriter.create(tsFile, pqcSigner(tsKp));
    const meta: BracketMeta[] = [{ id: 'b0', type: 'I', offset: 0, driftScore: 0 }];
    await w.appendBrackets('frame-x', Buffer.from(encodeBracket(fixtureBracket(0))), meta,
      { avgTempC: 1, peakVoltV: 1, clockMHz: 1 }, { token: '', unit: '' });
    await w.close();
    expect(goRun('verify-frame', tsFile, tsKey, 'frame-x')).toBe('true');
    expect(goRun('verify-frame', tsFile, goKey, 'frame-x')).toBe('false');
  });

  // Must match nrv3Bracket() in go/cmd/nrvconform/main.go.
  function nrv3Bracket(n: number): Bracket {
    const projections = new Uint8Array(32);
    for (let j = 0; j < 32; j++) {
      projections[j] = ((10 + j + (n % 3)) & 0xff);
      if (n % 90 === 89) projections[j] = (255 - 7 * j) & 0xff;
    }
    return {
      id: '', projections, subSecondUS: n, syntactic: 0, depHead: (n % 100) - 50, intentFlags: 0, domainSig: 0,
      goldenSeed: Math.imul(n, 2654435761 | 0) >>> 0, memory: Uint8Array.from({ length: 14 }, (_, i) => (i === 0 ? n & 0xff : 0)),
      lshSalt: (n << 8) >>> 0, reserved: new Uint8Array(17), frameId: '', frameUnix: 0,
    };
  }

  it('E17: TS reads Go-converted NRV3 files and recovers the original brackets', async () => {
    const nrv3File = path.join(dir, 'fixture.nrv3');
    goRun('write-nrv3', nrv3File, '150', '40');

    const outFile = path.join(dir, 'from-nrv3.nrv');
    expect(goRun('convert-nrv3', nrv3File, outFile).split('\n')).toEqual([outFile]);

    const reader = await NRVReader.open(outFile);
    const reg = reader.getRegistry();
    expect(reg.frames.map((f) => f.id)).toEqual(['frame-0', 'frame-1']);
    expect(reg.frameCount).toBe(2);

    // The index uses real KNIRVBASE deltas: P-brackets anchor to an I-bracket of the same frame.
    const types = reg.frames[0].bracketIndex!.map((m) => m.type);
    expect(types[0]).toBe('I');
    expect(types).toContain('P');
    const ids = new Set(reg.frames[0].bracketIndex!.map((m) => m.id));
    for (const m of reg.frames[0].bracketIndex!) {
      if (m.type === 'P') expect(ids.has(m.anchorId!)).toBe(true);
    }

    // Reading resolves the deltas back to exactly what the hasher wrote.
    let n = 0;
    for (const [frame, count] of [['frame-0', 150], ['frame-1', 40]] as const) {
      const bs = reader.getBrackets(frame)!;
      expect(bs).toHaveLength(count);
      for (const b of bs) {
        expect(Buffer.from(encodeBracket(b))).toEqual(Buffer.from(encodeBracket(nrv3Bracket(n))));
        n++;
      }
    }
    expect(n).toBe(190);
  });

  it('E17: a frame too big for one registry is sharded and every shard reads back', async () => {
    const nrv3File = path.join(dir, 'big.nrv3');
    goRun('write-nrv3', nrv3File, '16100');
    const outFile = path.join(dir, 'big.nrv');
    const files = goRun('convert-nrv3', nrv3File, outFile).split('\n');
    expect(files).toEqual([outFile, path.join(dir, 'big.1.nrv')]);

    let n = 0;
    for (const f of files) {
      const reader = await NRVReader.open(f);
      for (const entry of reader.getRegistry().frames) {
        for (const b of reader.getBrackets(entry.id)!) {
          expect(Buffer.from(encodeBracket(b))).toEqual(Buffer.from(encodeBracket(nrv3Bracket(n))));
          n++;
        }
      }
    }
    expect(n).toBe(16100);
  });

  it('E18: convert-nrv3 with a key signs every frame; TS verifies, a wrong key fails', async () => {
    const nrv3File = path.join(dir, 'fixture2.nrv3');
    goRun('write-nrv3', nrv3File, '30', '30');

    const keyFile = path.join(dir, 'convert-sig-key.json');
    fs.writeFileSync(keyFile, goRun('pqc', 'keygen'));
    const kp = loadPQCKeyPair(fs.readFileSync(keyFile, 'utf8'));

    const outFile = path.join(dir, 'from-nrv3-signed.nrv');
    goRun('convert-nrv3', nrv3File, outFile, keyFile);

    const reader = await NRVReader.open(outFile);
    expect(reader.getRegistry().pqcManifest.keyId).toBe(kp.id);
    expect(reader.verifyFrame('frame-0', kp)).toBe(true);
    expect(reader.verifyFrame('frame-1', kp)).toBe(true);
    expect(reader.verifyFrame('frame-0', generatePQCKeyPair('other', 'signature'))).toBe(false);
  });
});
