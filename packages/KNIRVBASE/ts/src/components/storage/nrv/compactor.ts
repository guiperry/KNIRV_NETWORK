import * as fs from 'fs';
import { Frame, Registry } from './codec';
import { NRVReader } from './reader';
import { NRVWriter, Signer } from './writer';

const COMPACTION_THRESHOLD = 0.20;

export class Compactor {
  private datasetPath: string;
  private keyPair?: Signer;
  private running: boolean = false;
  private stopCh: Promise<void> & { resolve(): void; reject(err: any): void };
  private compactCallback?: () => void;

  constructor(datasetPath: string, keyPair?: Signer) {
    this.datasetPath = datasetPath;
    this.keyPair = keyPair;
    let resolve: () => void;
    let reject: (err: any) => void;
    this.stopCh = new Promise<void>((res, rej) => {
      resolve = res;
      reject = rej;
    }) as any;
    this.stopCh.resolve = resolve!;
    this.stopCh.reject = reject!;
  }

  maybeCompact(registry: Registry): void {
    if (this.running) {
      return;
    }

    if (registry.frameCount === 0) {
      return;
    }

    const ratio = registry.tombstoneCount / registry.frameCount;
    if (ratio >= COMPACTION_THRESHOLD) {
      this.running = true;
      this.compact().catch(err => {
        console.error('compaction error:', err);
      }).finally(() => {
        this.running = false;
      });
    }
  }

  start(callback?: () => void): void {
    if (this.running) {
      return;
    }
    this.running = true;
    this.compactCallback = callback;

    const tick = async () => {
      while (this.running) {
        await new Promise(resolve => setTimeout(resolve, 30000));
        if (!this.running) break;
        // Check and compact
      }
    };

    tick().catch(err => {
      console.error('compactor error:', err);
    });
  }

  stop(): void {
    this.running = false;
  }

  private async compact(): Promise<void> {
    let reader: NRVReader | null = null;
    let writer: NRVWriter | null = null;

    try {
      reader = await NRVReader.open(this.datasetPath);
      
      const tmpPath = this.datasetPath + '.tmp';
      
      // Delete tmp file if exists
      try {
        await fs.promises.unlink(tmpPath);
      } catch {}

      writer = await NRVWriter.create(tmpPath, this.keyPair);

      for (const entry of reader.getRegistry().frames) {
        if (entry.tombstone !== undefined) {
          continue;
        }

        const frame = reader.getFrame(entry.id);
        if (!frame) {
          continue;
        }

        await writer.appendFrame(frame, entry.verified, entry.ergoRank);
      }

      // Mark as compacted
      const registry = writer.getRegistry();
      registry.globalMetrics.compactedAt = new Date().toISOString();

      await writer.saveRegistry();
      await writer.close();
      writer = null;

      // Rename compacted file over original
      await fs.promises.rename(tmpPath, this.datasetPath);

      // Remove WAL
      const walPath = this.datasetPath + '.wal';
      try {
        await fs.promises.unlink(walPath);
      } catch {}

      this.compactCallback?.();
    } catch (err) {
      console.error('compaction failed:', err);
      throw err;
    } finally {
      if (reader) {
        reader.close();
      }
      if (writer) {
        await writer.close().catch(() => {});
      }
    }
  }

  getDatasetPath(): string {
    return this.datasetPath;
  }
}
