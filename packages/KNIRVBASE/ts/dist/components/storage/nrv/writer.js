import * as fs from 'fs';
import * as crypto from 'crypto';
import { encodeHeader, encodeFrame, createDefaultRegistry, } from './codec';
import { NRV_HEADER_SIZE, NRV_REGISTRY_PADDING } from './spec';
import { WAL } from './wal';
export class NRVWriter {
    constructor(path, file, registry, wal, position, keyPair) {
        this.path = path;
        this.file = file;
        this.registry = registry;
        this.wal = wal;
        this.position = position;
        this.keyPair = keyPair;
    }
    static async create(path, keyPair) {
        const walPath = path + '.wal';
        const wal = new WAL(walPath);
        let file;
        let registry;
        let position;
        try {
            const stats = await fs.promises.stat(path);
            if (stats.size === 0) {
                // Initialize new file
                file = await fs.promises.open(path, 'r+');
                const id = crypto.randomUUID();
                registry = createDefaultRegistry(id, keyPair ? 'key' : undefined);
                // Write header
                const header = encodeHeader(NRV_HEADER_SIZE + NRV_REGISTRY_PADDING);
                await file.write(Buffer.from(header), 0, header.length, 0);
                // Write registry
                const registryJson = JSON.stringify(registry, null, 2);
                const padding = Buffer.alloc(NRV_REGISTRY_PADDING - registryJson.length);
                await file.write(Buffer.from(registryJson), 0, registryJson.length, NRV_HEADER_SIZE);
                await file.write(padding, 0, padding.length, NRV_HEADER_SIZE + registryJson.length);
                position = NRV_HEADER_SIZE + NRV_REGISTRY_PADDING;
                // Recover from WAL if needed
                const recoverLen = await wal.recover();
                if (recoverLen > 0) {
                    await file.truncate(recoverLen);
                    position = recoverLen;
                }
                await file.sync();
            }
            else {
                // Open existing file
                file = await fs.promises.open(path, 'r+');
                // Recover from WAL
                const recoverLen = await wal.recover();
                if (recoverLen > 0) {
                    await file.truncate(recoverLen);
                    position = recoverLen;
                }
                else {
                    position = stats.size;
                }
                // Load registry
                const registryBuf = Buffer.alloc(NRV_REGISTRY_PADDING);
                await file.read(registryBuf, 0, NRV_REGISTRY_PADDING, NRV_HEADER_SIZE);
                const end = findEndOfJson(registryBuf);
                if (end === 0) {
                    throw new Error('empty registry');
                }
                const registryJson = registryBuf.slice(0, end).toString('utf8');
                registry = JSON.parse(registryJson);
                // Restore typed arrays
                registry.globalMetrics.featureMin = new Float32Array(registry.globalMetrics.featureMin);
                registry.globalMetrics.featureMax = new Float32Array(registry.globalMetrics.featureMax);
                registry.globalMetrics.featureMean = new Float32Array(registry.globalMetrics.featureMean);
                registry.globalMetrics.featureStd = new Float32Array(registry.globalMetrics.featureStd);
            }
        }
        catch (err) {
            if (err.code === 'ENOENT') {
                // Create new file
                file = await fs.promises.open(path, 'w+');
                const id = crypto.randomUUID();
                registry = createDefaultRegistry(id, keyPair ? 'key' : undefined);
                // Write header
                const header = encodeHeader(NRV_HEADER_SIZE + NRV_REGISTRY_PADDING);
                await file.write(Buffer.from(header), 0, header.length, 0);
                // Write registry
                const registryJson = JSON.stringify(registry, null, 2);
                const padding = Buffer.alloc(NRV_REGISTRY_PADDING - registryJson.length);
                await file.write(Buffer.from(registryJson), 0, registryJson.length, NRV_HEADER_SIZE);
                await file.write(padding, 0, padding.length, NRV_HEADER_SIZE + registryJson.length);
                position = NRV_HEADER_SIZE + NRV_REGISTRY_PADDING;
                await file.sync();
            }
            else {
                throw err;
            }
        }
        return new NRVWriter(path, file, registry, wal, position, keyPair);
    }
    async appendFrame(frame, verified, ergoRank) {
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
        const entry = {
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
            this.registry.globalMetrics.verifiedFrameCount++;
        }
        this.registry.globalMetrics.ergoRankSum += ergoRank;
        // Save registry
        await this.saveRegistry();
        // Commit WAL
        await this.wal.commit(frame.id);
    }
    async saveRegistry() {
        const registryJson = JSON.stringify(this.registry, null, 2);
        const padding = Buffer.alloc(NRV_REGISTRY_PADDING - registryJson.length);
        await this.file.write(Buffer.from(registryJson), 0, registryJson.length, NRV_HEADER_SIZE);
        await this.file.write(padding, 0, padding.length, NRV_HEADER_SIZE + registryJson.length);
        // Update header with new total length
        const header = encodeHeader(this.position);
        await this.file.write(Buffer.from(header), 0, header.length, 0);
        await this.file.sync();
    }
    getRegistry() {
        return this.registry;
    }
    async close() {
        await this.file.close();
    }
}
function findEndOfJson(buf) {
    let end = buf.length;
    for (let i = buf.length - 1; i >= 0; i--) {
        const byte = buf[i];
        if (byte !== 0 && byte !== 0x20 && byte !== 0x0a && byte !== 0x0d && byte !== 0x09) {
            end = i + 1;
            break;
        }
    }
    if (end > 0) {
        const lastChar = buf[end - 1];
        if (lastChar !== 0x7d) {
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
//# sourceMappingURL=writer.js.map