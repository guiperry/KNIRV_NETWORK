import { NRV_MAGIC, NRV_VERSION, NRV_HEADER_SIZE, ModalityType, align8, } from './spec';
export function encodeHeader(totalLength) {
    const buf = new Uint8Array(NRV_HEADER_SIZE);
    const view = new DataView(buf.buffer);
    view.setUint32(0, NRV_MAGIC, true);
    view.setUint32(4, NRV_VERSION, true);
    view.setUint32(8, totalLength, true);
    return buf;
}
export function decodeHeader(data) {
    const view = new DataView(data.buffer, data.byteOffset);
    const magic = view.getUint32(0, true);
    if (magic !== NRV_MAGIC) {
        throw new Error(`invalid magic bytes: got 0x${magic.toString(16)}, want 0x${NRV_MAGIC.toString(16)}`);
    }
    return {
        magic,
        version: view.getUint32(4, true),
        totalLength: view.getUint32(8, true),
    };
}
export function encodeFrame(frame) {
    const proofAligned = align8(frame.proof.length);
    const total = 48 + 32 + 16 + proofAligned;
    const buf = new Uint8Array(total);
    // Encode vector (12 float32 values = 48 bytes)
    const vectorView = new DataView(buf.buffer, buf.byteOffset);
    for (let i = 0; i < 12; i++) {
        vectorView.setFloat32(i * 4, frame.vector[i], true);
    }
    // Encode seed (32 bytes)
    buf.set(frame.seed.slice(0, 32), 48);
    // Encode thermo data (4 float32 values = 16 bytes)
    const thermoOffset = 80;
    vectorView.setFloat32(thermoOffset, frame.thermo.tempCelsius, true);
    vectorView.setFloat32(thermoOffset + 4, frame.thermo.voltageV, true);
    vectorView.setFloat32(thermoOffset + 8, frame.thermo.freqMHz, true);
    vectorView.setFloat32(thermoOffset + 12, frame.thermo.fanRPM, true);
    // Encode proof
    buf.set(frame.proof.slice(0, frame.proof.length), 96);
    const modalities = {
        [ModalityType.Vector]: { offset: 0, length: 48 },
        [ModalityType.Seed]: { offset: 48, length: 32 },
        [ModalityType.Thermo]: { offset: 80, length: 16 },
        [ModalityType.Proof]: { offset: 96, length: frame.proof.length },
    };
    return { data: buf, modalities };
}
export function decodeFrame(data, entry) {
    if (data.length < 96) {
        throw new Error('frame too small');
    }
    const view = new DataView(data.buffer, data.byteOffset);
    // Decode vector
    const vector = new Float32Array(12);
    for (let i = 0; i < 12; i++) {
        vector[i] = view.getFloat32(i * 4, true);
    }
    // Decode seed
    const seed = new Uint8Array(32);
    for (let i = 0; i < 32; i++) {
        seed[i] = data[48 + i];
    }
    // Decode thermo
    const thermo = {
        tempCelsius: view.getFloat32(80, true),
        voltageV: view.getFloat32(84, true),
        freqMHz: view.getFloat32(88, true),
        fanRPM: view.getFloat32(92, true),
    };
    // Decode proof if present
    let proof = new Uint8Array(0);
    if (entry.modalities[ModalityType.Proof]) {
        const mod = entry.modalities[ModalityType.Proof];
        proof = data.slice(mod.offset, mod.offset + mod.length);
    }
    return {
        id: entry.id,
        vector,
        seed,
        thermo,
        proof,
    };
}
export function createDefaultRegistry(datasetId, keyId) {
    return {
        version: 1,
        datasetId,
        datasetVersion: {},
        chunk0Length: 5 * 1024 * 1024,
        frameCount: 0,
        tombstoneCount: 0,
        globalMetrics: {
            featureMin: new Float32Array(12),
            featureMax: new Float32Array(12),
            featureMean: new Float32Array(12),
            featureStd: new Float32Array(12),
            thermoCorrelationCoefficient: 0,
            ergoRankSum: 0,
            verifiedFrameCount: 0,
        },
        frames: [],
        pqcManifest: {
            keyId: keyId || '',
            algorithm: 'Dilithium-3',
            fileSignature: '',
            frameSignatures: {},
        },
    };
}
//# sourceMappingURL=codec.js.map