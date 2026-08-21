import { wasmModulesManifest as generatedManifest } from "./modules/manifest.js";
export const wasmModulesManifest = generatedManifest;
export function wasmModuleMetadata(id) {
    const module = wasmModulesManifest.modules.find(candidate => candidate.id === id);
    if (!module)
        throw new RangeError(`unknown KNIRV WASM module: ${id}`);
    return module;
}
export function wasmModuleURL(id) {
    wasmModuleMetadata(id);
    return new URL(`./modules/${id}.wasm`, import.meta.url);
}
/** Loads a raw module asset. Integrity verification is performed by SdkClient. */
export async function fetchWasmModule(id, source = wasmModuleURL(id)) {
    wasmModuleMetadata(id);
    if (source instanceof Uint8Array)
        return new Uint8Array(source);
    if (source instanceof ArrayBuffer)
        return new Uint8Array(source.slice(0));
    const response = source instanceof Response ? source : await fetch(source);
    if (!response.ok)
        throw new Error(`could not load KNIRV WASM module ${id}: HTTP ${response.status}`);
    return new Uint8Array(await response.arrayBuffer());
}
