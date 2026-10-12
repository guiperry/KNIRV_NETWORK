import * as fs from 'fs';
import * as path from 'path';
import * as crypto from 'crypto';
import {
  Frame,
  Registry,
  FrameIndexEntry,
  GlobalMetrics,
  PQCManifest,
  encodeHeader,
  encodeFrame,
  createDefaultRegistry,
} from './codec';
import { NRV_HEADER_SIZE, NRV_REGISTRY_PADDING, createModalityMap } from './spec';
import { registryToPaddedBuffer, registryFromPaddedBuffer } from './registry_json';
import { WAL, WALEntry } from './wal';
import { PQCKeyPair, sign as pqcSign } from '../../crypto/pqc/keys';
import { BracketMeta, LinguisticMapping, ThermoAtmosphere } from './bracket';

export interface Signer {
  /** Key identifier recorded in the registry's pqc_manifest. */
  keyId?: string;
  /** Returns the base64 signature (ML-DSA-65) over `data`. */
  sign(data: Uint8Array): Promise<string>;
}

/** Signer backed by a PQC key pair, matching Go's base64(Dilithium/ML-DSA signature). */
export function pqcSigner(kp: PQCKeyPair): Signer {
  return {
    keyId: kp.id,
    sign: async (data) => Buffer.from(pqcSign(kp, data)).toString('base64'),
  };
}

export class NRVWriter {
  private path: string;
  private keyPair?: Signer;
  private wal: WAL;
  private file: fs.promises.FileHandle;
  private registry: Registry;
  private position: number;

  constructor(
    path: string,
    file: fs.promises.FileHandle,
    registry: Registry,
    wal: WAL,
    position: number,
    keyPair?: Signer
  ) {
    this.path = path;
    this.file = file;
    this.registry = registry;
    this.wal = wal;
    this.position = position;
    this.keyPair = keyPair;
  }

  static async create(path: string, keyPair?: Signer): Promise<NRVWriter> {
    // Create parent directories automatically — mirrors production behaviour where
    // the app data directory may not exist yet (e.g. first run after install).
    const parentDir = require('path').dirname(path);
    await fs.promises.mkdir(parentDir, { recursive: true });

    const walPath = path + '.wal';
    const wal = new WAL(walPath);

    let file: fs.promises.FileHandle;
    let registry: Registry;
    let position: number;

    try {
      const stats = await fs.promises.stat(path);
      if (stats.size === 0) {
        // Initialize new file
        file = await fs.promises.open(path, 'r+');
        const id = crypto.randomUUID();
        registry = createDefaultRegistry(id, keyPair?.keyId);
        
        // Write header
        const header = encodeHeader(NRV_HEADER_SIZE + NRV_REGISTRY_PADDING);
        await file.write(Buffer.from(header), 0, header.length, 0);

        // Write registry
        const regBuf = registryToPaddedBuffer(registry);
        await file.write(regBuf, 0, regBuf.length, NRV_HEADER_SIZE);
        
        position = NRV_HEADER_SIZE + NRV_REGISTRY_PADDING;

        // Recover from WAL if needed
        const recoverLen = await wal.recover();
        if (recoverLen > 0) {
          await file.truncate(recoverLen);
          position = recoverLen;
        }

        await file.sync();
      } else {
        // Open existing file
        file = await fs.promises.open(path, 'r+');
        
        // Recover from WAL
        const recoverLen = await wal.recover();
        if (recoverLen > 0) {
          await file.truncate(recoverLen);
          position = recoverLen;
        } else {
          position = stats.size;
        }

        // Load registry
        const registryBuf = Buffer.alloc(NRV_REGISTRY_PADDING);
        await file.read(registryBuf, 0, NRV_REGISTRY_PADDING, NRV_HEADER_SIZE);
        registry = registryFromPaddedBuffer(registryBuf);
      }
    } catch (err: any) {
      if (err.code === 'ENOENT') {
        // Create new file
        file = await fs.promises.open(path, 'w+');
        const id = crypto.randomUUID();
        registry = createDefaultRegistry(id, keyPair?.keyId);
        
        // Write header
        const header = encodeHeader(NRV_HEADER_SIZE + NRV_REGISTRY_PADDING);
        await file.write(Buffer.from(header), 0, header.length, 0);

        // Write registry
        const regBuf = registryToPaddedBuffer(registry);
        await file.write(regBuf, 0, regBuf.length, NRV_HEADER_SIZE);
        
        position = NRV_HEADER_SIZE + NRV_REGISTRY_PADDING;
        await file.sync();
      } else {
        throw err;
      }
    }

    return new NRVWriter(path, file, registry, wal, position, keyPair);
  }

  async appendFrame(frame: Frame, verified: boolean, ergoRank: number): Promise<void> {
    // Begin WAL entry
    await this.wal.begin({
      frameId: frame.id,
      lastGoodLength: this.position,
      committed: false,
    });

    // Encode frame
    const { data: frameData, modalities } = encodeFrame(frame);

    // Sign if key pair available
    if (this.keyPair) {
      const signature = await this.keyPair.sign(frameData);
      this.registry.pqcManifest.frameSignatures[frame.id] = signature;
    }

    // Write frame
    await this.file.write(Buffer.from(frameData), 0, frameData.length, this.position);
    this.position += frameData.length;

    // Create frame entry
    const entry: FrameIndexEntry = {
      id: frame.id,
      offset: this.position - frameData.length,
      length: frameData.length,
      verified,
      ergoRank,
      modalities,
    };

    this.registry.frames.push(entry);
    this.registry.frameCount++;
    if (verified) {
      this.registry.globalMetrics.validFrameCount++;
    }
    this.registry.globalMetrics.invalidFrameCount += verified ? 0 : 1;

    // Save registry
    await this.saveRegistry();

    // Commit WAL
    await this.wal.commit(frame.id);
  }

  /**
   * Append a frame of pre-encoded 80-byte brackets (Go's AppendFrame). This is
   * the canonical write path: the file layout and registry entry are identical
   * to what the Go writer produces.
   */
  async appendBrackets(
    frameId: string,
    bracketBuf: Uint8Array,
    bracketIndex: BracketMeta[],
    thermo: ThermoAtmosphere,
    linguistic: LinguisticMapping
  ): Promise<void> {
    await this.wal.begin({ frameId, lastGoodLength: this.position, committed: false });

    const offset = this.position;
    if (this.keyPair) {
      this.registry.pqcManifest.frameSignatures[frameId] = await this.keyPair.sign(bracketBuf);
    }
    await this.file.write(bracketBuf, 0, bracketBuf.length, offset);
    this.position += bracketBuf.length;

    this.registry.frames.push({
      id: frameId,
      offset,
      length: bracketBuf.length,
      verified: true,
      ergoRank: 1.0,
      modalities: createModalityMap(0),
      timestampUnix: Math.floor(Date.now() / 1000),
      linguistic,
      thermo,
      bracketCount: bracketIndex.length,
      bracketIndex,
    });
    this.registry.frameCount++;
    this.registry.globalMetrics.validFrameCount++;
    this.registry.globalMetrics.totalBracketCount += bracketIndex.length;

    await this.saveRegistry();
    await this.wal.commit(frameId);
  }

  async saveRegistry(): Promise<void> {
    const regBuf = registryToPaddedBuffer(this.registry);
    await this.file.write(regBuf, 0, regBuf.length, NRV_HEADER_SIZE);

    // Update header with new total length
    const header = encodeHeader(this.position);
    await this.file.write(Buffer.from(header), 0, header.length, 0);
    
    await this.file.sync();
  }

  /** Mark a frame as tombstoned in the registry (soft-delete). */
  async setTombstone(id: string): Promise<void> {
    const now = Date.now();
    for (const entry of this.registry.frames) {
      if (entry.id === id) {
        (entry as any).tombstone = now;
        this.registry.tombstoneCount++;
        break;
      }
    }
    await this.saveRegistry();
  }

  getRegistry(): Registry {
    return this.registry;
  }

  async close(): Promise<void> {
    await this.file.close();
  }
}
