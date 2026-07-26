import * as fs from 'fs';
import * as path from 'path';
import { Storage } from '../storage';
import { IndexManager, Index, IndexType } from '../index';
import { EncryptionManager, PQCKeyPair } from '../../crypto/pqc';
import { EntryType } from '../../types/types';
import { NRVReader } from './reader';
import { NRVWriter } from './writer';
import { Compactor } from './compactor';
import { WAL } from './wal';
import * as nrv from './spec';
import { Bracket, ThermoAtmosphere, BracketMeta, encodeBracket } from './bracket';

export class NRVStorage implements Storage {
  readonly baseDir: string;
  readonly keyPair: PQCKeyPair | null;
  
  private indexManager: IndexManager;
  private encryptionMgr: EncryptionManager;
  private wal: WAL;
  private walPath: string;
  private writers: Map<string, NRVWriter> = new Map();
  private readers: Map<string, NRVReader> = new Map();

  constructor(baseDir: string, keyPair?: PQCKeyPair) {
    this.baseDir = baseDir;
    this.keyPair = keyPair || null;
    
    fs.mkdirSync(baseDir, { recursive: true, mode: 0o700 });
    
    this.indexManager = new IndexManager(baseDir);
    this.indexManager.loadIndexes();
    
    this.encryptionMgr = new EncryptionManager();
    if (keyPair) {
      this.encryptionMgr.setMasterKey(keyPair);
    }
    
    this.walPath = path.join(baseDir, 'wal.log');
    this.wal = new WAL(this.walPath);
  }

  private getCollectionPath(collection: string): string {
    return path.join(this.baseDir, `${collection}.nrv`);
  }

  private async getWriter(collection: string): Promise<NRVWriter> {
    if (!this.writers.has(collection)) {
      const filePath = this.getCollectionPath(collection);
      // Convert PQCKeyPair to Signer if needed
      const signer = this.keyPair ? {
        sign: async (data: Uint8Array) => {
          // Simplified signing - in real implementation, use actual PQC signing
          return Buffer.from(data).toString('base64');
        }
      } : undefined;
      const writer = await NRVWriter.create(filePath, signer);
      this.writers.set(collection, writer);
    }
    return this.writers.get(collection)!;
  }

  private async getReader(collection: string): Promise<NRVReader | null> {
    if (this.readers.has(collection)) {
      return this.readers.get(collection)!;
    }
    
    const filePath = this.getCollectionPath(collection);
    if (!fs.existsSync(filePath)) {
      return null;
    }
    
    try {
      const reader = await NRVReader.open(filePath);
      this.readers.set(collection, reader);
      return reader;
    } catch {
      return null;
    }
  }

  async insert(collection: string, doc: Record<string, any>): Promise<void> {
    await this.wal.begin({ frameId: doc.id, lastGoodLength: 0, committed: false });

    const writer = await this.getWriter(collection);
    const frame = this.documentToFrame(doc);
    await writer.appendFrame(frame, doc.verified || false, doc.ergo_rank || 0.0);

    await this.wal.commit(doc.id);

    // Invalidate cached reader so next find() sees the updated registry.
    this.readers.delete(collection);

    await this.indexManager.insert(collection, doc);

    // Check if we need to truncate WAL
    try {
      const stats = await fs.promises.stat(this.walPath);
      if (stats.size > 1000 * 1024) {
        await this.wal.truncate();
      }
    } catch {
      // Ignore if file doesn't exist
    }
  }

  async update(collection: string, id: string, update: Record<string, any>): Promise<number> {
    const existing = await this.find(collection, id);
    if (!existing) {
      throw new Error(`Document not found: ${id}`);
    }
    
    Object.assign(existing, update);
    await this.insert(collection, existing);
    return 1;
  }

  async delete(collection: string, id: string): Promise<number> {
    const writer = await this.getWriter(collection);
    // Mark the registry entry as tombstoned (matching Go's approach).
    await writer.setTombstone(id);
    // Invalidate cached reader so next find() sees the tombstone.
    this.readers.delete(collection);
    await this.indexManager.delete(collection, id);
    return 1;
  }

  async find(collection: string, id: string): Promise<Record<string, any> | null> {
    const reader = await this.getReader(collection);
    if (!reader) {
      return null;
    }

    try {
      // Check registry entry first: ensures tombstones and metadata (verified, ergoRank) are current.
      const registry = reader.getRegistry();
      const entry = registry.frames.find((f: any) => f.id === id);
      if (!entry || entry.tombstone !== undefined) {
        return null;
      }

      const frame = reader.getFrame(id);
      if (!frame) {
        return null;
      }

      return this.frameToDocument(frame, entry);
    } catch {
      return null;
    }
  }

  async findAll(collection: string): Promise<Record<string, any>[]> {
    const reader = await this.getReader(collection);
    if (!reader) {
      return [];
    }

    const docs: Record<string, any>[] = [];

    const registry = reader.getRegistry();
    for (const entry of registry.frames) {
      if ((entry as any).tombstone !== undefined) {
        continue;
      }
      const frame = reader.getFrame(entry.id);
      if (!frame) {
        continue;
      }
      docs.push(this.frameToDocument(frame, entry));
    }
    
    return docs;
  }

  async put(key: string, value: Uint8Array): Promise<void> {
    const kvPath = path.join(this.baseDir, 'kv', key);
    fs.mkdirSync(path.dirname(kvPath), { recursive: true });
    fs.writeFileSync(kvPath, Buffer.from(value));
  }

  async get(key: string): Promise<Uint8Array | null> {
    const kvPath = path.join(this.baseDir, 'kv', key);
    try {
      return new Uint8Array(fs.readFileSync(kvPath));
    } catch (err: any) {
      if (err.code === 'ENOENT') return null;
      throw err;
    }
  }

  async deleteKey(key: string): Promise<void> {
    const kvPath = path.join(this.baseDir, 'kv', key);
    try {
      fs.unlinkSync(kvPath);
    } catch {}
  }

  async has(key: string): Promise<boolean> {
    const kvPath = path.join(this.baseDir, 'kv', key);
    return fs.existsSync(kvPath);
  }

  async storeObject(key: string, obj: any): Promise<void> {
    const json = JSON.stringify(obj);
    await this.put(key, new TextEncoder().encode(json));
  }

  async getObject<T = any>(key: string): Promise<T | null> {
    const data = await this.get(key);
    if (!data) return null;
    const json = new TextDecoder().decode(data);
    return JSON.parse(json) as T;
  }

  async projectToMarkdown(key: string, targetPath: string): Promise<void> {
    const data = await this.get(key);
    if (!data) {
      throw new Error(`Key not found: ${key}`);
    }
    
    const content = new TextDecoder().decode(data);
    const markdown = `# Storage Object: ${key}\n\n\`\`\`json\n${JSON.stringify(JSON.parse(content), null, 2)}\n\`\`\``;
    
    fs.mkdirSync(path.dirname(targetPath), { recursive: true });
    fs.writeFileSync(targetPath, markdown);
  }

  async createIndex(collection: string, name: string, indexType: IndexType, fields: string[], unique: boolean, partialExpr: string, options: Record<string, any>): Promise<void> {
    await this.indexManager.createIndex(collection, name, indexType, fields, unique, partialExpr, options);
  }

  async dropIndex(collection: string, name: string): Promise<void> {
    await this.indexManager.dropIndex(collection, name);
  }

  getIndex(collection: string, name: string): Index | null {
    return this.indexManager.getIndex(collection, name);
  }

  getIndexesForCollection(collection: string): Index[] {
    return this.indexManager.getIndexesForCollection(collection);
  }

  async queryIndex(collection: string, indexName: string, query: Record<string, any>): Promise<string[]> {
    return this.indexManager.queryIndex(collection, indexName, query);
  }

  async close(): Promise<void> {
    const writersArray = Array.from(this.writers.values());
    for (const writer of writersArray) {
      await writer.close();
    }
    
    await this.wal.truncate();
    
    this.writers.clear();
    this.readers.clear();
  }

  async getModality(collection: string, id: string, modality: nrv.ModalityType): Promise<Uint8Array> {
    const reader = await this.getReader(collection);
    if (!reader) {
      throw new Error(`Collection not found: ${collection}`);
    }
    
    const modalityData = reader.getModality(id, modality);
    if (modalityData === null) {
      throw new Error(`Modality not found for id: ${id}`);
    }
    
    return modalityData;
  }

  async *streamFrames(collection: string, modalityFilter: nrv.ModalityType): AsyncIterableIterator<any> {
    const reader = await this.getReader(collection);
    if (!reader) {
      return;
    }

    const registry = reader.getRegistry();
    for (const entry of registry.frames) {
      if ((entry as any).tombstone !== undefined) {
        continue;
      }
      const frame = reader.getFrame(entry.id);
      if (frame) {
        yield this.frameToDocument(frame, entry);
      }
    }
  }

  private documentToFrame(doc: Record<string, any>): any {
    return {
      id: doc.id,
      vector: new Float32Array(doc.payload?.vector || []),
      seed: new Uint8Array(doc.payload?.seed ? [doc.payload.seed] : [0]),
      thermo: {
        tempCelsius: doc.payload?.thermo?.temp_celsius || 0,
        voltageV: doc.payload?.thermo?.voltage_v || 0,
        freqMHz: doc.payload?.thermo?.freq_mhz || 0,
        fanRPM: doc.payload?.thermo?.fan_rpm || 0
      },
      proof: new Uint8Array(doc.payload?.proof || []),
      verified: doc.verified || false,
      ergoRank: doc.ergo_rank || 0.0
    };
  }

  private frameToDocument(frame: any, entry?: any): Record<string, any> {
    return {
      id: frame.id,
      payload: {
        vector: Array.from(frame.vector),
        seed: frame.seed.length > 0 ? frame.seed[0] : 0,
        thermo: {
          temp_celsius: frame.thermo.tempCelsius,
          voltage_v: frame.thermo.voltageV,
          freq_mhz: frame.thermo.freqMHz,
          fan_rpm: frame.thermo.fanRPM
        },
        proof: frame.proof
      },
      // Prefer registry entry metadata (authoritative) over frame fields.
      verified: entry !== undefined ? entry.verified : (frame.verified || false),
      ergo_rank: entry !== undefined ? entry.ergoRank : (frame.ergoRank || 0.0),
    };
  }

  async compact(collection: string): Promise<void> {
    const filePath = this.getCollectionPath(collection);
    const signer = this.keyPair ? {
      sign: async (data: Uint8Array) => Buffer.from(data).toString('base64')
    } : undefined;
    const compactor = new Compactor(filePath, signer);
    
    const reader = await this.getReader(collection);
    if (reader) {
      const registry = (reader as any).getRegistry?.();
      if (registry) {
        compactor.maybeCompact(registry);
      }
    }
  }

  async appendBracket(collection: string, bracket: Bracket, thermo: ThermoAtmosphere): Promise<void> {
    const writer = await this.getWriter(collection);
    const registry = writer.getRegistry();
    
    const bracketData = encodeBracket(bracket);
    const timestamp = Date.now();
    
    const meta = bracket.meta;
    if (meta) {
      registry.globalMetrics.totalBracketCount++;
    }
    
    registry.globalMetrics.avgTempCMean = (registry.globalMetrics.avgTempCMean * (registry.globalMetrics.validFrameCount) + thermo.avgTempC) / (registry.globalMetrics.validFrameCount + 1);
    if (thermo.avgTempC > registry.globalMetrics.avgTempCMax) {
      registry.globalMetrics.avgTempCMax = thermo.avgTempC;
    }
    registry.globalMetrics.peakVoltVMean = (registry.globalMetrics.peakVoltVMean * (registry.globalMetrics.validFrameCount) + thermo.peakVoltV) / (registry.globalMetrics.validFrameCount + 1);
    registry.globalMetrics.clockMHzMean = (registry.globalMetrics.clockMHzMean * (registry.globalMetrics.validFrameCount) + thermo.clockMHz) / (registry.globalMetrics.validFrameCount + 1);

    await writer.saveRegistry();
  }
}