import { wasmModulesManifest as generatedManifest, wasmModulesReleaseVersion } from "./modules/manifest.js";

export interface WasmModuleMetadata {
  id: string;
  artifact_version: string;
  module_kind: string;
  abi_version: number;
  byte_length: number;
  sha256: string;
  capabilities: readonly string[];
}

export interface WasmModuleManifest {
  version: number;
  modules: readonly WasmModuleMetadata[];
}

export type WasmModuleSource = RequestInfo | URL | Response | ArrayBuffer | Uint8Array;

export const wasmModulesManifest: WasmModuleManifest = generatedManifest;

export function wasmModuleMetadata(id: string): WasmModuleMetadata {
  const module = wasmModulesManifest.modules.find(candidate => candidate.id === id);
  if (!module) throw new RangeError(`unknown KNIRV WASM module: ${id}`);
  return module;
}

/** Where versioned SDK artifacts are published. */
export const WASM_RELEASE_BASE_URL = "https://releases.knirv.com/sdk";

/** Modules shipped inside the npm package: a vault must work offline on first use. */
const BUNDLED_MODULES: ReadonlySet<string> = new Set(["crypto-core"]);

/**
 * Default location of a module: next to this package when it is bundled,
 * otherwise the published release for this SDK version (no `import.meta.url`
 * asset path, which bundlers and Workers handle inconsistently).
 */
export function wasmModuleURL(id: string): URL {
  wasmModuleMetadata(id);
  if (BUNDLED_MODULES.has(id)) return new URL(`./modules/${id}.wasm`, import.meta.url);
  return new URL(`${WASM_RELEASE_BASE_URL}/${wasmModulesReleaseVersion}/modules/${id}.wasm`);
}

/** Loads a raw module asset. Integrity verification is performed by SdkClient. */
export async function fetchWasmModule(id: string, source: WasmModuleSource = wasmModuleURL(id)): Promise<Uint8Array> {
  wasmModuleMetadata(id);
  if (source instanceof Uint8Array) return new Uint8Array(source);
  if (source instanceof ArrayBuffer) return new Uint8Array(source.slice(0));
  const response = source instanceof Response ? source : await fetch(source);
  if (!response.ok) throw new Error(`could not load KNIRV WASM module ${id}: HTTP ${response.status}`);
  return new Uint8Array(await response.arrayBuffer());
}
