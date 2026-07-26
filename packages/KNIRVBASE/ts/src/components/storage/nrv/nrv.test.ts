import { describe, it, expect, beforeEach, afterEach } from '@jest/globals';
import * as fs from 'fs/promises';
import * as path from 'path';
import os from 'os';
import { NRVStorage } from './';
import { NRVWriter } from './writer';
import { NRVReader } from './reader';
import { encodeFrame, Frame } from './codec';
import { generatePQCKeyPair, PQCKeyPair } from '../../crypto/pqc/keys';
import { ModalityType } from './spec';

describe('NRV Storage (TypeScript Implementation)', () => {
  let tempDir: string;
  let storage: NRVStorage;
  let keyPair: PQCKeyPair;

  beforeEach(async () => {
    tempDir = await fs.mkdtemp(path.join(os.tmpdir(), 'knirv-nrv-test-'));
    keyPair = generatePQCKeyPair('test-key', 'test-password');
    storage = new NRVStorage(tempDir, keyPair);
  });

  afterEach(async () => {
    await storage.close();
    await fs.rm(tempDir, { recursive: true, force: true });
  });

  it('should initialize storage correctly', async () => {
    expect(storage).toBeDefined();
    expect(storage.baseDir).toBe(tempDir);
    expect(storage.keyPair).toBe(keyPair);
    
    const exists = await fs.access(tempDir).then(() => true).catch(() => false);
    expect(exists).toBe(true);
  });

  it('should insert and find document round-trip', async () => {
    const testDoc = {
      id: 'test-doc-1',
      payload: {
        vector: [1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12],
        seed: new Uint8Array(Array.from({ length: 32 }, (_, i) => i)),
        thermo: {
          temp_celsius: 50.5,
          voltage_v: 12.0,
          freq_mhz: 500,
          fan_rpm: 3000
        },
        proof: new TextEncoder().encode('test proof string')
      },
      verified: true,
      ergo_rank: 0.85
    };

    await storage.insert('test-collection', testDoc);
    const found = await storage.find('test-collection', 'test-doc-1');

    expect(found).toBeDefined();
    expect(found!.id).toBe(testDoc.id);
    expect(found!.verified).toBe(testDoc.verified);
    expect(found!.ergo_rank).toBe(testDoc.ergo_rank);
    expect(found!.payload.vector).toEqual(testDoc.payload.vector);
  });

  it('should find all documents in collection', async () => {
    const docs = [
      { id: 'doc-1', payload: { vector: [1,2,3,4,5,6,7,8,9,10,11,12] }, verified: true, ergo_rank: 0.8 },
      { id: 'doc-2', payload: { vector: [2,3,4,5,6,7,8,9,10,11,12,13] }, verified: false, ergo_rank: 0.7 },
    ];

    for (const doc of docs) {
      await storage.insert('test-collection', doc);
    }

    const allDocs = await storage.findAll('test-collection');
    expect(allDocs.length).toBe(2);
    expect(allDocs.find(d => d.id === 'doc-1')!.verified).toBe(true);
    expect(allDocs.find(d => d.id === 'doc-2')!.verified).toBe(false);
  });

  it('should delete document correctly', async () => {
    const testDoc = {
      id: 'delete-test',
      payload: { vector: [1,2,3,4,5,6,7,8,9,10,11,12] },
      verified: true,
      ergo_rank: 0.9
    };

    await storage.insert('delete-collection', testDoc);
    const foundBefore = await storage.find('delete-collection', 'delete-test');
    expect(foundBefore).toBeDefined();

    await storage.delete('delete-collection', 'delete-test');
    const foundAfter = await storage.find('delete-collection', 'delete-test');
    expect(foundAfter).toBeNull();
  });

  it('should extract individual modalities', async () => {
    const testDoc = {
      id: 'modality-test',
      payload: {
        vector: [1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12],
        seed: new Uint8Array(32),
        thermo: { temp_celsius: 50.5, voltage_v: 12.0, freq_mhz: 500, fan_rpm: 3000 },
        proof: new Uint8Array([1,2,3,4,5,6,7,8,9,10,11,12,13,14,15,16])
      },
      verified: true,
      ergo_rank: 0.85
    };

    await storage.insert('modality-collection', testDoc);

    const vectorBytes = await storage.getModality('modality-collection', 'modality-test', ModalityType.Vector);
    expect(vectorBytes.length).toBe(48);

    const seedBytes = await storage.getModality('modality-collection', 'modality-test', ModalityType.Seed);
    expect(seedBytes.length).toBe(32);

    const thermoBytes = await storage.getModality('modality-collection', 'modality-test', ModalityType.Thermo);
    expect(thermoBytes.length).toBe(16);

    const proofBytes = await storage.getModality('modality-collection', 'modality-test', ModalityType.Proof);
    expect(proofBytes.length).toBe(16);
  });

  it('should stream frames correctly', async () => {
    const frameIds = ['stream-1', 'stream-2', 'stream-3'];
    
    for (const id of frameIds) {
      await storage.insert('stream-collection', {
        id,
        payload: { vector: [1,2,3,4,5,6,7,8,9,10,11,12] },
        verified: true,
        ergo_rank: 0.9
      });
    }

    let count = 0;
    for await (const frame of storage.streamFrames('stream-collection', ModalityType.Vector)) {
      expect(frame).toBeDefined();
      count++;
    }

    expect(count).toBe(3);
  });

  it('should work with standard production app data directory paths', async () => {
    const appDataPath = path.join(os.homedir(), '.local', 'share', 'KNIRV', 'test-storage');
    
    try {
      const productionStorage = new NRVStorage(appDataPath, keyPair);
      expect(productionStorage.baseDir).toBe(appDataPath);
      
      const testDoc = { id: 'prod-test', payload: { vector: [1,2,3,4,5,6,7,8,9,10,11,12] }, verified: true, ergo_rank: 0.8 };
      await productionStorage.insert('prod-collection', testDoc);
      
      const found = await productionStorage.find('prod-collection', 'prod-test');
      expect(found!.id).toBe('prod-test');
      
      await productionStorage.close();
    } finally {
      await fs.rm(appDataPath, { recursive: true, force: true });
    }
  });
});

// ── NRVWriter / NRVReader direct tests ───────────────────────────────────────

describe('NRVWriter + NRVReader (direct)', () => {
  let tmpDir: string;

  beforeEach(async () => {
    tmpDir = await fs.mkdtemp(path.join(os.tmpdir(), 'knirv-nrv-rw-'));
  });

  afterEach(async () => {
    await fs.rm(tmpDir, { recursive: true, force: true });
  });

  function makeTestFrame(id: string): Frame {
    const seed = new Uint8Array(32);
    for (let i = 0; i < 32; i++) seed[i] = i + 1;
    return {
      id,
      vector: new Float32Array([1,2,3,4,5,6,7,8,9,10,11,12]),
      seed,
      thermo: { tempCelsius: 50.5, voltageV: 12.0, freqMHz: 500.0, fanRPM: 3000.0 },
      proof: new TextEncoder().encode('test proof data'),
    };
  }

  it('writer creates .nrv file in app data directory structure', async () => {
    const dataDir = path.join(tmpDir, 'knirvbase', 'datasets');
    const nrvPath = path.join(dataDir, 'training.nrv');

    const writer = await NRVWriter.create(nrvPath);
    expect(await fs.access(nrvPath).then(() => true).catch(() => false)).toBe(true);

    const stat = await fs.stat(nrvPath);
    expect(stat.size).toBeGreaterThan(5 * 1024 * 1024); // at least 5 MiB
    await writer.close();
  });

  it('writer append + reader roundtrip preserves all frame fields', async () => {
    const nrvPath = path.join(tmpDir, 'rt.nrv');
    const frame = makeTestFrame('rt-frame');

    const writer = await NRVWriter.create(nrvPath);
    await writer.appendFrame(frame, true, 0.9);
    await writer.close();

    const reader = await NRVReader.open(nrvPath);
    const got = reader.getFrame('rt-frame');
    expect(got).not.toBeNull();
    expect(got!.id).toBe('rt-frame');
    expect(Array.from(got!.vector)).toEqual(Array.from(frame.vector));
    expect(Array.from(got!.seed)).toEqual(Array.from(frame.seed));
    expect(got!.thermo.tempCelsius).toBeCloseTo(50.5, 3);
    expect(got!.thermo.voltageV).toBeCloseTo(12.0, 3);
    expect(Array.from(got!.proof)).toEqual(Array.from(frame.proof));
  });

  it('writer reopen persists frames across sessions', async () => {
    const nrvPath = path.join(tmpDir, 'persist.nrv');

    {
      const w = await NRVWriter.create(nrvPath);
      await w.appendFrame(makeTestFrame('frame-a'), true, 0.8);
      await w.close();
    }

    const w2 = await NRVWriter.create(nrvPath);
    const reg = w2.getRegistry();
    expect(reg.frameCount).toBe(1);
    await w2.appendFrame(makeTestFrame('frame-b'), false, 0.5);
    await w2.close();

    const reader = await NRVReader.open(nrvPath);
    expect(reader.getRegistry().frameCount).toBe(2);
    expect(reader.getFrame('frame-a')).not.toBeNull();
    expect(reader.getFrame('frame-b')).not.toBeNull();
  });

  it('reader get_modality returns correct bytes', async () => {
    const nrvPath = path.join(tmpDir, 'mod.nrv');
    const frame = makeTestFrame('m');

    const w = await NRVWriter.create(nrvPath);
    await w.appendFrame(frame, true, 0.9);
    await w.close();

    const reader = await NRVReader.open(nrvPath);

    const vectorBytes = reader.getModality('m', ModalityType.Vector);
    expect(vectorBytes).not.toBeNull();
    expect(vectorBytes!.length).toBe(48);
    const dv = new DataView(vectorBytes!.buffer, vectorBytes!.byteOffset);
    expect(dv.getFloat32(0, true)).toBeCloseTo(1.0, 5);

    const seedBytes = reader.getModality('m', ModalityType.Seed);
    expect(seedBytes!.length).toBe(32);
    expect(seedBytes![0]).toBe(1);

    const thermoBytes = reader.getModality('m', ModalityType.Thermo);
    expect(thermoBytes!.length).toBe(16);
    const tvDv = new DataView(thermoBytes!.buffer, thermoBytes!.byteOffset);
    expect(tvDv.getFloat32(0, true)).toBeCloseTo(50.5, 3);
  });
});

// ── Cross-language binary format spec ─────────────────────────────────────────
//
// These tests pin the exact byte layout that Go, Rust, and TypeScript all share.
// If any implementation diverges, one of these three suites will fail first.

describe('NRV binary format specification (cross-language)', () => {
  it('encodeFrame produces spec-compliant byte layout', () => {
    const seed = new Uint8Array(32);
    for (let i = 0; i < 32; i++) seed[i] = i + 1;

    const frame: Frame = {
      id: 'spec-frame',
      vector: new Float32Array([1,2,3,4,5,6,7,8,9,10,11,12]),
      seed,
      thermo: { tempCelsius: 50.5, voltageV: 12.0, freqMHz: 500.0, fanRPM: 3000.0 },
      proof: new TextEncoder().encode('ok'),
    };

    const { data, modalities } = encodeFrame(frame);

    // Total = 48 + 32 + 16 + align8(2) = 104
    expect(data.length).toBe(104);

    // Modality map
    expect(modalities[ModalityType.Vector]).toEqual({ offset: 0, length: 48 });
    expect(modalities[ModalityType.Seed]).toEqual({ offset: 48, length: 32 });
    expect(modalities[ModalityType.Thermo]).toEqual({ offset: 80, length: 16 });
    expect(modalities[ModalityType.Proof]).toEqual({ offset: 96, length: 2 });

    const dv = new DataView(data.buffer);

    // vector[0] = 1.0f32 LE
    expect(dv.getFloat32(0, true)).toBeCloseTo(1.0, 5);
    // vector[11] = 12.0f32 LE at offset 44
    expect(dv.getFloat32(44, true)).toBeCloseTo(12.0, 5);

    // seed bytes 1..32
    for (let i = 0; i < 32; i++) {
      expect(data[48 + i]).toBe(i + 1);
    }

    // thermo at offsets 80..96
    expect(dv.getFloat32(80, true)).toBeCloseTo(50.5, 3);
    expect(dv.getFloat32(84, true)).toBeCloseTo(12.0, 5);
    expect(dv.getFloat32(88, true)).toBeCloseTo(500.0, 5);
    expect(dv.getFloat32(92, true)).toBeCloseTo(3000.0, 5);

    // proof "ok"
    expect(data[96]).toBe(0x6f); // 'o'
    expect(data[97]).toBe(0x6b); // 'k'
    // 6 bytes of alignment padding must be zero
    for (let i = 98; i < 104; i++) {
      expect(data[i]).toBe(0);
    }
  });

  it('header magic bytes match spec: 0x21 0x56 0x52 0x4E', async () => {
    const tmpDir = await fs.mkdtemp(path.join(os.tmpdir(), 'knirv-hdr-'));
    try {
      const nrvPath = path.join(tmpDir, 'hdr.nrv');
      const w = await NRVWriter.create(nrvPath);
      await w.close();

      const buf = await fs.readFile(nrvPath);
      expect(buf[0]).toBe(0x21);
      expect(buf[1]).toBe(0x56);
      expect(buf[2]).toBe(0x52);
      expect(buf[3]).toBe(0x4e);

      const dv = new DataView(buf.buffer);
      expect(dv.getUint32(4, true)).toBe(1); // version
    } finally {
      await fs.rm(tmpDir, { recursive: true, force: true });
    }
  });
});