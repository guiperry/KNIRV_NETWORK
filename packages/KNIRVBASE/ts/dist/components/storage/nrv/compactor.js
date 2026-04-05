import * as fs from 'fs';
import { NRVReader } from './reader';
import { NRVWriter } from './writer';
const COMPACTION_THRESHOLD = 0.20;
export class Compactor {
    constructor(datasetPath, keyPair) {
        this.running = false;
        this.datasetPath = datasetPath;
        this.keyPair = keyPair;
        let resolve;
        let reject;
        this.stopCh = new Promise((res, rej) => {
            resolve = res;
            reject = rej;
        });
        this.stopCh.resolve = resolve;
        this.stopCh.reject = reject;
    }
    maybeCompact(registry) {
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
    start(callback) {
        if (this.running) {
            return;
        }
        this.running = true;
        this.compactCallback = callback;
        const tick = async () => {
            while (this.running) {
                await new Promise(resolve => setTimeout(resolve, 30000));
                if (!this.running)
                    break;
                // Check and compact
            }
        };
        tick().catch(err => {
            console.error('compactor error:', err);
        });
    }
    stop() {
        this.running = false;
    }
    async compact() {
        let reader = null;
        let writer = null;
        try {
            reader = await NRVReader.open(this.datasetPath);
            const tmpPath = this.datasetPath + '.tmp';
            // Delete tmp file if exists
            try {
                await fs.promises.unlink(tmpPath);
            }
            catch { }
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
            }
            catch { }
            this.compactCallback?.();
        }
        catch (err) {
            console.error('compaction failed:', err);
            throw err;
        }
        finally {
            if (reader) {
                reader.close();
            }
            if (writer) {
                await writer.close().catch(() => { });
            }
        }
    }
    getDatasetPath() {
        return this.datasetPath;
    }
}
//# sourceMappingURL=compactor.js.map