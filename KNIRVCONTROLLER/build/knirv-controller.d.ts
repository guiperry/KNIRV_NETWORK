/** Exported memory */
export declare const memory: WebAssembly.Memory;
// Exported runtime interface
export declare function __new(size: number, id: number): number;
export declare function __pin(ptr: number): number;
export declare function __unpin(ptr: number): void;
export declare function __collect(): void;
export declare const __rtti_base: number;
/**
 * assembly/index/createAgentCore
 * @param id `~lib/string/String`
 * @returns `bool`
 */
export declare function createAgentCore(id: string): boolean;
/**
 * assembly/index/initializeAgent
 * @returns `bool`
 */
export declare function initializeAgent(): boolean;
/**
 * assembly/index/executeAgent
 * @param input `~lib/string/String`
 * @param _context `~lib/string/String`
 * @returns `~lib/string/String`
 */
export declare function executeAgent(input: string, _context: string): string;
/**
 * assembly/index/executeAgentTool
 * @param toolName `~lib/string/String`
 * @param parameters `~lib/string/String`
 * @param _context `~lib/string/String`
 * @returns `~lib/string/String`
 */
export declare function executeAgentTool(toolName: string, parameters: string, _context: string): string;
/**
 * assembly/index/loadLoraAdapter
 * @param adapter `~lib/string/String`
 * @returns `bool`
 */
export declare function loadLoraAdapter(adapter: string): boolean;
/**
 * assembly/index/getAgentStatus
 * @returns `~lib/string/String`
 */
export declare function getAgentStatus(): string;
/**
 * assembly/index/createModel
 * @param type `~lib/string/String`
 * @returns `bool`
 */
export declare function createModel(type: string): boolean;
/**
 * assembly/index/loadModelWeights
 * @param _weightsPtr `usize`
 * @param weightsLen `i32`
 * @returns `bool`
 */
export declare function loadModelWeights(_weightsPtr: number, weightsLen: number): boolean;
/**
 * assembly/index/runModelInference
 * @param input `~lib/string/String`
 * @param _context `~lib/string/String`
 * @returns `~lib/string/String`
 */
export declare function runModelInference(input: string, _context: string): string;
/**
 * assembly/index/getModelInfo
 * @returns `~lib/string/String`
 */
export declare function getModelInfo(): string;
/**
 * assembly/index/getWasmVersion
 * @returns `~lib/string/String`
 */
export declare function getWasmVersion(): string;
/**
 * assembly/index/getSupportedFeatures
 * @returns `~lib/string/String`
 */
export declare function getSupportedFeatures(): string;
/**
 * assembly/index/allocateString
 * @param str `~lib/string/String`
 * @returns `usize`
 */
export declare function allocateString(str: string): number;
/**
 * assembly/index/deallocateString
 * @param _ptr `usize`
 */
export declare function deallocateString(_ptr: number): void;
/**
 * assembly/index/wasmInit
 */
export declare function wasmInit(): void;
