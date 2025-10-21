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
 * @param context `~lib/string/String`
 * @returns `~lib/string/String`
 */
export declare function executeAgent(input: string, context: string): string;
/**
 * assembly/index/executeAgentTool
 * @param toolName `~lib/string/String`
 * @param parameters `~lib/string/String`
 * @param context `~lib/string/String`
 * @returns `~lib/string/String`
 */
export declare function executeAgentTool(toolName: string, parameters: string, context: string): string;
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
 * @param weightsPtr `f64`
 * @param weightsLen `f64`
 * @returns `bool`
 */
export declare function loadModelWeights(weightsPtr: number, weightsLen: number): boolean;
/**
 * assembly/index/runModelInference
 * @param input `~lib/string/String`
 * @param context `~lib/string/String`
 * @returns `~lib/string/String`
 */
export declare function runModelInference(input: string, context: string): string;
/**
 * assembly/index/getModelInfo
 * @returns `~lib/string/String`
 */
export declare function getModelInfo(): string;
/**
 * assembly/index/configureExternalInference
 * @param provider `~lib/string/String`
 * @param apiKey `~lib/string/String`
 * @param endpoint `~lib/string/String`
 * @param model `~lib/string/String`
 * @returns `bool`
 */
export declare function configureExternalInference(provider: string, apiKey: string, endpoint: string, model: string): boolean;
/**
 * assembly/index/setActiveInferenceProvider
 * @param provider `~lib/string/String`
 * @returns `bool`
 */
export declare function setActiveInferenceProvider(provider: string): boolean;
/**
 * assembly/index/getConfiguredProviders
 * @returns `~lib/string/String`
 */
export declare function getConfiguredProviders(): string;
/**
 * assembly/index/performExternalInference
 * @param prompt `~lib/string/String`
 * @param systemPrompt `~lib/string/String`
 * @param maxTokens `f64`
 * @param temperature `f64`
 * @returns `~lib/string/String`
 */
export declare function performExternalInference(prompt: string, systemPrompt?: string, maxTokens?: number, temperature?: number): string;
/**
 * assembly/index/getExternalInferenceStatus
 * @returns `~lib/string/String`
 */
export declare function getExternalInferenceStatus(): string;
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
 * assembly/index/performChatCompletion
 * @param messagesJson `~lib/string/String`
 * @param configJson `~lib/string/String`
 * @returns `~lib/string/String`
 */
export declare function performChatCompletion(messagesJson: string, configJson?: string): string;
/**
 * assembly/index/initializeExternalInferenceFromEnv
 * @param envConfigJson `~lib/string/String`
 * @returns `bool`
 */
export declare function initializeExternalInferenceFromEnv(envConfigJson: string): boolean;
/**
 * assembly/index/allocateString
 * @param str `~lib/string/String`
 * @returns `f64`
 */
export declare function allocateString(str: string): number;
/**
 * assembly/index/deallocateString
 * @param _ptr `f64`
 */
export declare function deallocateString(_ptr: number): void;
/**
 * assembly/index/wasmInit
 */
export declare function wasmInit(): void;
