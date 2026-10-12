import * as fs from 'fs';
import {
  Frame,
  FrameIndexEntry,
  Registry,
  decodeHeader,
  decodeFrame,
} from './codec';
import { NRV_HEADER_SIZE, NRV_REGISTRY_PADDING, ModalityType } from './spec';
import { Bracket, BracketSize, applyProjectionDelta, decodeBracket } from './bracket';
import { registryFromPaddedBuffer } from './registry_json';
import { PQCKeyPair, verify as pqcVerify } from '../../crypto/pqc/keys';

export class NRVReader {
  private path: string;
  private data: Uint8Array;
  private registry: Registry;

  constructor(path: string, data: Uint8Array, registry: Registry) {
    this.path = path;
    this.data = data;
    this.registry = registry;
  }

  static async open(path: string): Promise<NRVReader> {
    const buffer = await fs.promises.readFile(path);
    const data = new Uint8Array(buffer);

    const header = decodeHeader(data);
    
    const registry = registryFromPaddedBuffer(
      data.subarray(NRV_HEADER_SIZE, NRV_HEADER_SIZE + NRV_REGISTRY_PADDING)
    );

    return new NRVReader(path, data, registry);
  }

  getFrame(id: string): Frame | null {
    for (const entry of this.registry.frames) {
      if (entry.id === id) {
        if (entry.tombstone !== undefined) {
          return null;
        }
        return this.decodeFrame(entry);
      }
    }
    return null;
  }

  getModality(frameId: string, modality: ModalityType): Uint8Array | null {
    for (const entry of this.registry.frames) {
      if (entry.id === frameId) {
        if (entry.tombstone !== undefined) {
          return null;
        }
        const modIndex = entry.modalities[modality];
        if (!modIndex) {
          return null;
        }
        const start = entry.offset + modIndex.offset;
        const end = start + modIndex.length;
        return this.data.slice(start, end);
      }
    }
    return null;
  }

  *streamFrames(modalityFilter?: ModalityType): Generator<Frame> {
    for (const entry of this.registry.frames) {
      if (entry.tombstone !== undefined) {
        continue;
      }
      const frame = this.decodeFrame(entry);
      yield frame;
    }
  }

  /**
   * Decode a frame's brackets exactly as Go's NRVReader.decodeBrackets does:
   * positions come from bracket_index, and delta ("P") brackets are resolved
   * against their anchor's projections.
   */
  getBrackets(frameId: string): Bracket[] | null {
    const entry = this.registry.frames.find((e) => e.id === frameId);
    if (!entry || entry.tombstone !== undefined) {
      return null;
    }
    const index = entry.bracketIndex ?? [];
    const anchors = new Map<string, Bracket>();
    const out: Bracket[] = [];
    for (const meta of index) {
      const start = entry.offset + meta.offset;
      const b = decodeBracket(this.data.subarray(start, start + BracketSize));
      b.id = meta.id;
      b.frameId = frameId;
      b.frameUnix = entry.timestampUnix ?? 0;
      b.meta = { ...meta };
      if (meta.type === 'P' && meta.anchorId !== undefined) {
        const anchor = anchors.get(meta.anchorId);
        if (anchor) {
          b.projections = applyProjectionDelta(b.projections, anchor.projections);
        }
      }
      anchors.set(meta.id, b);
      out.push(b);
    }
    return out;
  }

  getRegistry(): Registry {
    return this.registry;
  }

  /**
   * Verify a frame's ML-DSA-65 signature over its raw bytes (Go: VerifyFrame).
   * Throws if the frame or its signature is missing, like Go does.
   */
  verifyFrame(id: string, keyPair: PQCKeyPair): boolean {
    const entry = this.registry.frames.find((e) => e.id === id);
    if (!entry) {
      throw new Error(`nrv: frame not found: ${id}`);
    }
    const sigStr = this.registry.pqcManifest.frameSignatures[id];
    if (!sigStr) {
      throw new Error('nrv: no signature for frame');
    }
    const sig = new Uint8Array(Buffer.from(sigStr, 'base64'));
    return pqcVerify(keyPair, this.data.subarray(entry.offset, entry.offset + entry.length), sig);
  }

  private decodeFrame(entry: FrameIndexEntry): Frame {
    const frameData = this.data.slice(entry.offset, entry.offset + entry.length);
    return decodeFrame(frameData, entry);
  }

  close(): void {
    // Nothing to close for memory-mapped data in Node.js
  }
}
