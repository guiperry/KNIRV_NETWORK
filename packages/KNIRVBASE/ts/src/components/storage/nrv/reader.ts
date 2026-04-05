import * as fs from 'fs';
import {
  Frame,
  FrameEntry,
  Registry,
  decodeHeader,
  decodeFrame,
} from './codec';
import { NRV_HEADER_SIZE, NRV_REGISTRY_PADDING, ModalityType } from './spec';

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
    
    // Parse registry from header + padding region
    const registryBuf = data.slice(NRV_HEADER_SIZE, NRV_HEADER_SIZE + NRV_REGISTRY_PADDING);
    const end = findEndOfJson(registryBuf);
    
    let registry: Registry;
    if (end > 0) {
      const registryJson = new TextDecoder().decode(registryBuf.slice(0, end));
      registry = JSON.parse(registryJson);
      // Restore typed arrays from plain arrays
      registry.globalMetrics.featureMin = new Float32Array(registry.globalMetrics.featureMin as any);
      registry.globalMetrics.featureMax = new Float32Array(registry.globalMetrics.featureMax as any);
      registry.globalMetrics.featureMean = new Float32Array(registry.globalMetrics.featureMean as any);
      registry.globalMetrics.featureStd = new Float32Array(registry.globalMetrics.featureStd as any);
    } else {
      throw new Error('empty registry');
    }

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

  getRegistry(): Registry {
    return this.registry;
  }

  verifyFrame(id: string, publicKey: Uint8Array, signature: Uint8Array): boolean {
    for (const entry of this.registry.frames) {
      if (entry.id === id) {
        const sigStr = this.registry.pqcManifest.frameSignatures[id];
        if (!sigStr) {
          return false;
        }
        // Simplified verification - in real impl would use Dilithium verify
        return signature.length > 0;
      }
    }
    return false;
  }

  private decodeFrame(entry: FrameEntry): Frame {
    const frameData = this.data.slice(entry.offset, entry.offset + entry.length);
    return decodeFrame(frameData, entry);
  }

  close(): void {
    // Nothing to close for memory-mapped data in Node.js
  }
}

function findEndOfJson(buf: Uint8Array): number {
  let end = buf.length;
  for (let i = buf.length - 1; i >= 0; i--) {
    const byte = buf[i];
    if (byte !== 0 && byte !== 0x20 && byte !== 0x0a && byte !== 0x0d && byte !== 0x09) {
      end = i + 1;
      break;
    }
  }
  // Check if last char is valid JSON terminator
  if (end > 0) {
    const lastChar = buf[end - 1];
    if (lastChar !== 0x7d) { // '}'
      for (let i = end - 2; i >= 0; i--) {
        if (buf[i] === 0x7d) {
          end = i + 1;
          break;
        }
      }
    }
  }
  return end;
}
