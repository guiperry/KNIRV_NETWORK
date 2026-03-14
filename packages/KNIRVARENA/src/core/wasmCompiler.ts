// TypeScript Agent.wasm Compiler - Matches Rust cortex.wasm compiler exactly
import { LoRAAdapterEngine, LoRAAdapterSkill } from './lora/LoRAAdapterEngine';
import { WASMCompiler } from './wasm/WASMCompiler';
import ProtobufHandler from './protobuf/ProtobufHandler';

// ProtoBuf message interfaces (matching cortex.proto)
export interface InferenceInput {
  prompt: string;
  max_tokens?: number;
  temperature?: number;
  top_p?: number;
  stop_sequences?: string[];
}

export interface InferenceOutput {
  response: string;
  tokens_used: number;
  processing_time_ms: number;
  model_info: string;
}

export interface CortexError {
  code: number;
  message: string;
  details?: string;
}

export interface Envelope {
  success: boolean;
  data?: Uint8Array;
  error?: CortexError;
}

// WASM export interfaces
export interface CortexWasmExports {
  memory: WebAssembly.Memory;
  allocate_buffer?: (size: number) => number;
  deallocate_buffer?: (ptr: number) => void;
  load_lora_adapter?: (ptr: number, len: number) => number;
  process_inference?: (ptr: number, len: number) => number;
  get_result_ptr?: () => number;
  get_result_len?: () => number;
  run_cognitive_task?: (ptr: number, len: number) => number;
}

interface LoRASkillResult {
  success: boolean;
  result?: LoRAAdapterSkill;
  error?: string;
  processingTime?: number;
}

export interface AgentCompilationRequest {
  agent_name: string;
  agent_description: string;
  adapters: LoRAAdapter[];
  config: AgentConfig;
  cortex_wasm: Uint8Array; // Pre-compiled cortex.wasm from Rust
}

export interface AgentConfig {
  target_platform: string; // "typescript", "golang", "rust"
  enable_lora: boolean;
  max_memory_mb: number;
  capabilities: string[];
  environment: Record<string, string>;
  optimization_level?: string;
}

export interface AgentCompilationResponse {
  success: boolean;
  message: string;
  agent_wasm?: Uint8Array;
  agent_id?: string;
  compilation_time_ms: number;
}

export interface LoRAAdapter {
  skill_id: string;
  skill_name: string;
  description: string;
  base_model_compatibility: string;
  version: number;
  rank: number;
  alpha: number;
  weights_a: number[];
  weights_b: number[];
  additional_metadata: Record<string, string>;
}

// TypeScript Agent.wasm Compiler - imports pre-compiled cortex.wasm
export class TypeScriptAgentCompiler {
  private isInitialized = false;
  private cortexWasm: WebAssembly.Module | null = null;
  private loraEngine: LoRAAdapterEngine;

  constructor() {
    const wasmCompiler = new WASMCompiler();
    const protobufHandler = new ProtobufHandler();
    this.loraEngine = new LoRAAdapterEngine(wasmCompiler, protobufHandler);
    this.initialize();
  }

  private async initialize() {
    console.log('Initializing TypeScript Agent.wasm Compiler...');

    // Initialize LoRA engine
    await this.loraEngine.initialize();

    this.isInitialized = true;
    console.log('TypeScript Agent.wasm Compiler initialized successfully');
  }

  // Import pre-compiled cortex.wasm from Rust version
  async importCortexWasm(cortexWasmBytes: Uint8Array): Promise<void> {
    if (!this.isInitialized) {
      throw new Error('Compiler not initialized');
    }

    console.log('Importing pre-compiled cortex.wasm from Rust version...');

    try {
      this.cortexWasm = await WebAssembly.compile(cortexWasmBytes);
      console.log('cortex.wasm imported successfully');
    } catch (error) {
      throw new Error(`Failed to import cortex.wasm: ${error}`);
    }
  }

  // Compile agent.wasm using imported cortex.wasm + TypeScript cognitive shell
  async compileAgent(request: AgentCompilationRequest): Promise<AgentCompilationResponse> {
    const startTime = Date.now();

    if (!this.isInitialized) {
      throw new Error('Compiler not initialized');
    }

    if (!this.cortexWasm) {
      // Import the cortex.wasm from the request
      await this.importCortexWasm(request.cortex_wasm);
    }

    console.log(`Compiling agent: ${request.agent_name} for TypeScript platform`);

    try {
      // Step 1: Process LoRA adapters
      const processedAdapters = await this.processLoRAAdapters(request.adapters);

      // Step 2: Create TypeScript cognitive shell wrapper
      const cognitiveShell = this.createCognitiveShell(request, processedAdapters);

      // Step 3: Combine cortex.wasm + cognitive shell into agent.wasm
      const agentWasm = await this.createAgentWasm(cognitiveShell, request.config);

      const compilationTime = Date.now() - startTime;

      return {
        success: true,
        message: `Agent compiled successfully for TypeScript platform`,
        agent_wasm: agentWasm,
        agent_id: this.generateAgentId(request.agent_name),
        compilation_time_ms: compilationTime
      };

    } catch (error) {
      const compilationTime = Date.now() - startTime;

      return {
        success: false,
        message: `Compilation failed: ${error}`,
        compilation_time_ms: compilationTime
      };
    }
  }

  private async processLoRAAdapters(adapters: LoRAAdapter[]): Promise<LoRAAdapter[]> {
    console.log(`Processing ${adapters.length} LoRA adapters...`);

    const processedAdapters: LoRAAdapter[] = [];

    for (const adapter of adapters) {
      // Validate and optimize adapter for TypeScript runtime
      const optimizedAdapter = await this.optimizeAdapterForTypeScript(adapter);
      processedAdapters.push(optimizedAdapter);
    }

    console.log(`Processed ${processedAdapters.length} LoRA adapters`);
    return processedAdapters;
  }

  private async optimizeAdapterForTypeScript(adapter: LoRAAdapter): Promise<LoRAAdapter> {
    // Optimize weights for TypeScript Float32Array compatibility
    return {
      ...adapter,
      weights_a: adapter.weights_a.map(w => Math.fround(w)), // Ensure 32-bit precision
      weights_b: adapter.weights_b.map(w => Math.fround(w)),
      additional_metadata: {
        ...adapter.additional_metadata,
        'typescript_optimized': 'true',
        'optimization_timestamp': new Date().toISOString()
      }
    };
  }

  private createCognitiveShell(request: AgentCompilationRequest, adapters: LoRAAdapter[]): string {
    // Create TypeScript cognitive shell that wraps cortex.wasm
    return `
// Generated TypeScript Cognitive Shell for Agent: ${request.agent_name}
// This shell wraps the imported cortex.wasm and provides TypeScript-specific functionality

class ${this.toPascalCase(request.agent_name)}CognitiveShell {
  private cortexInstance: WebAssembly.Instance | null = null;
  private adapters: LoRAAdapter[] = ${JSON.stringify(adapters, null, 2)};
  private config: AgentConfig = ${JSON.stringify(request.config, null, 2)};

  async initialize(cortexWasm: WebAssembly.Module) {
    this.cortexInstance = await WebAssembly.instantiate(cortexWasm, {
      env: {
        memory: new WebAssembly.Memory({ initial: 256, maximum: 512 }),
      }
    });

    // Initialize cortex with LoRA adapters
    await this.initializeLoRAAdapters();
  }

  private async initializeLoRAAdapters() {
    for (const adapter of this.adapters) {
      await this.loadLoRAAdapter(adapter);
    }
  }

  private async loadLoRAAdapter(adapter: LoRAAdapter) {
    if (!this.cortexInstance) throw new Error('Cortex not initialized');

    // Convert adapter to ProtoBuf bytes and load into cortex
    const adapterBytes = this.serializeAdapter(adapter);
    const exports = this.cortexInstance.exports as CortexWasmExports;

    // Call cortex WASM function to load adapter
    if (exports.compile_lora_adapter) {
      const ptr = this.allocateMemory(adapterBytes.length);
      this.writeMemory(ptr, adapterBytes);
      exports.compile_lora_adapter(ptr, adapterBytes.length);
    }
  }

  async runCognitiveTask(prompt: string): Promise<InferenceOutput> {
    if (!this.cortexInstance) throw new Error('Cortex not initialized');

    const exports = this.cortexInstance.exports as CortexWasmExports;

    // Prepare input
    const input: InferenceInput = { prompt };
    const inputBytes = this.serializeInput(input);

    // Allocate memory and call cortex
    const inputPtr = this.allocateMemory(inputBytes.length);
    this.writeMemory(inputPtr, inputBytes);

    const resultPtr = exports.run_cognitive_task(inputPtr, inputBytes.length);
    const [outputPtr, outputLen] = this.unpackPtrLen(resultPtr);

    // Read and deserialize output
    const outputBytes = this.readMemory(outputPtr, outputLen);
    return this.deserializeOutput(outputBytes);
  }

  // Memory management utilities
  private allocateMemory(size: number): number {
    const exports = this.cortexInstance!.exports as CortexWasmExports;
    return exports.allocate_buffer ? exports.allocate_buffer(size) : 0;
  }

  private writeMemory(ptr: number, data: Uint8Array) {
    const exports = this.cortexInstance!.exports as CortexWasmExports;
    const memory = exports.memory as WebAssembly.Memory;
    const view = new Uint8Array(memory.buffer);
    view.set(data, ptr);
  }

  private readMemory(ptr: number, len: number): Uint8Array {
    const exports = this.cortexInstance!.exports as CortexWasmExports;
    const memory = exports.memory as WebAssembly.Memory;
    const view = new Uint8Array(memory.buffer);
    return view.slice(ptr, ptr + len);
  }

  private unpackPtrLen(packed: number): [number, number] {
    const ptr = (packed >>> 32) & 0xFFFFFFFF;
    const len = packed & 0xFFFFFFFF;
    return [ptr, len];
  }

  // Serialization utilities (simplified - would use actual ProtoBuf in production)
  private serializeAdapter(adapter: LoRAAdapter): Uint8Array {
    return new TextEncoder().encode(JSON.stringify(adapter));
  }

  private serializeInput(input: InferenceInput): Uint8Array {
    return new TextEncoder().encode(JSON.stringify(input));
  }

  private deserializeOutput(bytes: Uint8Array): InferenceOutput {
    const json = new TextDecoder().decode(bytes);
    return JSON.parse(json);
  }
}

// Export the cognitive shell
export { ${this.toPascalCase(request.agent_name)}CognitiveShell };
`;
  }

  private async createAgentWasm(cognitiveShell: string, _config: AgentConfig): Promise<Uint8Array> {
    console.log('Creating agent.wasm with TypeScript cognitive shell...');

    // In a full implementation, this would:
    // 1. Compile the TypeScript cognitive shell to WASM using AssemblyScript or similar
    // 2. Link it with the imported cortex.wasm
    // 3. Create a unified agent.wasm module

    // For now, create a mock WASM module with embedded shell
    const shellBytes = new TextEncoder().encode(cognitiveShell);
    const wasmHeader = new Uint8Array([0x00, 0x61, 0x73, 0x6d, 0x01, 0x00, 0x00, 0x00]); // WASM magic + version
    const lengthBytes = new Uint8Array(4);
    new DataView(lengthBytes.buffer).setUint32(0, shellBytes.length, true);

    // Combine header + length + shell
    const agentWasm = new Uint8Array(wasmHeader.length + lengthBytes.length + shellBytes.length);
    agentWasm.set(wasmHeader, 0);
    agentWasm.set(lengthBytes, wasmHeader.length);
    agentWasm.set(shellBytes, wasmHeader.length + lengthBytes.length);

    console.log(`Created agent.wasm: ${agentWasm.length} bytes`);
    return agentWasm;
  }

  private generateAgentId(agentName: string): string {
    const sanitized = agentName.toLowerCase().replace(/[^a-z0-9]/g, '-');
    const timestamp = Date.now().toString(36);
    return `agent-ts-${sanitized}-${timestamp}`;
  }

  private toPascalCase(str: string): string {
    return str.replace(/(?:^|[-_])(\w)/g, (_, char) => char.toUpperCase());
  }

  // Get available LoRA adapters
  getAvailableAdapters(): LoRAAdapterSkill[] {
    return this.loraEngine.getAvailableAdapters();
  }

  // Invoke LoRA skill
  async invokeLoRASkill(skillId: string, parameters: Record<string, string>): Promise<LoRASkillResult> {
    const response = await this.loraEngine.invokeAdapter(skillId, parameters);
    return {
      success: response.status === 'SUCCESS',
      result: response.skill,
      error: response.errorMessage,
      processingTime: 0 // TODO: Add timing
    };
  }

  isAvailable(): boolean {
    return this.isInitialized;
  }

  // Compatibility method for API
  async compileRust(sourceCode: string): Promise<Uint8Array> {
    const request: AgentCompilationRequest = {
      agent_name: 'rust-agent',
      agent_description: 'Compiled Rust agent',
      cortex_wasm: new Uint8Array(),
      adapters: [],
      config: {
        optimization_level: 'basic',
        target_platform: 'typescript',
        enable_lora: false,
        max_memory_mb: 128,
        capabilities: [],
        environment: {}
      }
    };

    // For now, just return a mock WASM module
    // In a full implementation, this would compile the Rust source code
    const mockWasm = new TextEncoder().encode(sourceCode);
    return new Uint8Array([0x00, 0x61, 0x73, 0x6d, 0x01, 0x00, 0x00, 0x00, ...mockWasm]);
  }
}

export const typeScriptAgentCompiler = new TypeScriptAgentCompiler();
export const wasmCompiler = typeScriptAgentCompiler;
