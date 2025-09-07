/**
 * WASM Agent Manager - Revolutionary Cognitive Shell Implementation
 * Manages uploaded agent.wasm files and LoRA adapter integration
 * Replaces default HRM with user-uploaded cognitive agents
 */

import { EventEmitter } from './EventEmitter';

export interface AgentWASMModule {
  memory: WebAssembly.Memory;
  apply_lora_adapter: (adapter_ptr: number, adapter_len: number) => number;
  process_input: (input_ptr: number, input_len: number) => number;
  get_output: (output_ptr: number) => number;
  initialize_agent: () => number;
  cleanup_agent: () => void;
  malloc: (size: number) => number;
  free: (ptr: number) => void;
}

export interface LoRAAdapter {
  skillId: string;
  skillName: string;
  weightsA: Float32Array;
  weightsB: Float32Array;
  rank: number;
  alpha: number;
}

export interface AgentConfig {
  maxMemoryMB: number;
  enableLoRAAdapters: boolean;
  maxConcurrentSkills: number;
  timeoutMs: number;
}

export interface AgentMetadata {
  name: string;
  version: string;
  description: string;
  capabilities: string[];
  author: string;
  uploadedAt: Date;
  size: number;
  hash: string;
}

export class WASMAgentManager extends EventEmitter {
  private wasmModule: WebAssembly.Module | null = null;
  private wasmInstance: WebAssembly.Instance | null = null;
  private agentModule: AgentWASMModule | null = null;
  private isInitialized: boolean = false;
  private config: AgentConfig;
  private metadata: AgentMetadata | null = null;
  private loadedAdapters: Map<string, LoRAAdapter> = new Map();
  private primaryAgent: string | null = null;

  constructor(config: Partial<AgentConfig> = {}) {
    super();
    this.config = {
      maxMemoryMB: 256,
      enableLoRAAdapters: true,
      maxConcurrentSkills: 10,
      timeoutMs: 30000,
      ...config
    };
  }

  /**
   * Upload and compile a new agent WASM file
   */
  async uploadAgent(wasmBytes: Uint8Array, metadata: Partial<AgentMetadata>): Promise<boolean> {
    try {
      this.emit('agent_upload_started', { size: wasmBytes.length });

      // Validate WASM module
      if (!this.validateWASMModule(wasmBytes)) {
        throw new Error('Invalid WASM module');
      }

      // Compile WASM module
      this.wasmModule = await WebAssembly.compile(wasmBytes);
      
      // Create metadata
      this.metadata = {
        name: metadata.name || 'Unnamed Agent',
        version: metadata.version || '1.0.0',
        description: metadata.description || 'Custom cognitive agent',
        capabilities: metadata.capabilities || [],
        author: metadata.author || 'Unknown',
        uploadedAt: new Date(),
        size: wasmBytes.length,
        hash: await this.calculateHash(wasmBytes)
      };

      // Initialize the agent
      await this.initializeAgent();

      this.emit('agent_uploaded', { metadata: this.metadata });
      return true;

    } catch (error) {
      this.emit('agent_upload_failed', { error: error instanceof Error ? error.message : String(error) });
      throw error;
    }
  }

  /**
   * Initialize the uploaded agent WASM module
   */
  private async initializeAgent(): Promise<void> {
    if (!this.wasmModule) {
      throw new Error('No WASM module loaded');
    }

    try {
      // Create WASM instance with memory
      const memory = new WebAssembly.Memory({ 
        initial: this.config.maxMemoryMB * 16, // 16 pages per MB (64KB per page)
        maximum: this.config.maxMemoryMB * 16
      });

      this.wasmInstance = await WebAssembly.instantiate(this.wasmModule, {
        env: {
          memory,
          abort: () => {
            throw new Error('WASM execution aborted');
          },
          console_log: (ptr: number, len: number) => {
            const message = this.readStringFromMemory(ptr, len);
            console.log(`[Agent]: ${message}`);
          }
        }
      });

      // Get exported functions
      this.agentModule = this.wasmInstance.exports as unknown as AgentWASMModule;

      // Initialize the agent
      const initResult = this.agentModule.initialize_agent();
      if (initResult !== 0) {
        throw new Error(`Agent initialization failed with code: ${initResult}`);
      }

      this.isInitialized = true;
      this.emit('agent_initialized', { metadata: this.metadata });

    } catch (error) {
      this.emit('agent_initialization_failed', { error: error instanceof Error ? error.message : String(error) });
      throw error;
    }
  }

  /**
   * Load a LoRA adapter into the agent
   */
  async loadLoRAAdapter(adapter: LoRAAdapter): Promise<boolean> {
    if (!this.isInitialized || !this.agentModule) {
      throw new Error('Agent not initialized');
    }

    if (!this.config.enableLoRAAdapters) {
      throw new Error('LoRA adapters are disabled');
    }

    try {
      this.emit('lora_loading_started', { skillId: adapter.skillId });

      // Serialize LoRA adapter for WASM
      const serializedAdapter = this.serializeLoRAAdapter(adapter);
      
      // Allocate memory in WASM
      const adapterPtr = this.agentModule.malloc(serializedAdapter.length);
      this.writeToMemory(adapterPtr, serializedAdapter);

      // Apply LoRA adapter
      const result = this.agentModule.apply_lora_adapter(adapterPtr, serializedAdapter.length);
      
      // Free memory
      this.agentModule.free(adapterPtr);

      if (result === 0) {
        this.loadedAdapters.set(adapter.skillId, adapter);
        this.emit('lora_loaded', { skillId: adapter.skillId, skillName: adapter.skillName });
        return true;
      } else {
        throw new Error(`LoRA adapter loading failed with code: ${result}`);
      }

    } catch (error) {
      this.emit('lora_loading_failed', { skillId: adapter.skillId, error: error instanceof Error ? error.message : String(error) });
      throw error;
    }
  }

  /**
   * Process input through the cognitive agent
   */
  async processInput(input: string, context: unknown = {}): Promise<string> {
    if (!this.isInitialized || !this.agentModule) {
      throw new Error('Agent not initialized');
    }

    try {
      this.emit('processing_started', { inputLength: input.length });

      // Prepare input data
      const inputData = JSON.stringify({ input, context, timestamp: Date.now() });
      const inputBytes = new TextEncoder().encode(inputData);

      // Allocate memory for input
      const inputPtr = this.agentModule.malloc(inputBytes.length);
      this.writeToMemory(inputPtr, inputBytes);

      // Process input
      const outputPtr = this.agentModule.process_input(inputPtr, inputBytes.length);
      
      // Free input memory
      this.agentModule.free(inputPtr);

      if (outputPtr === 0) {
        throw new Error('Processing failed - null output');
      }

      // Get output length and read result
      const outputLen = this.agentModule.get_output(outputPtr);
      const outputBytes = this.readFromMemory(outputPtr, outputLen);
      const result = new TextDecoder().decode(outputBytes);

      // Free output memory
      this.agentModule.free(outputPtr);

      this.emit('processing_completed', { outputLength: result.length });
      return result;

    } catch (error) {
      this.emit('processing_failed', { error: error instanceof Error ? error.message : String(error) });
      throw error;
    }
  }

  /**
   * Set this agent as the primary agent
   */
  setPrimaryAgent(): void {
    if (!this.metadata) {
      throw new Error('No agent loaded');
    }

    this.primaryAgent = this.metadata.hash;
    this.emit('primary_agent_set', { agentHash: this.primaryAgent });
  }

  /**
   * Export the current agent as agent.wasm
   */
  async exportAgent(): Promise<Uint8Array> {
    if (!this.wasmModule) {
      throw new Error('No agent loaded');
    }

    // Get the compiled WASM bytes
    const wasmBytes = await WebAssembly.compile(this.wasmModule as BufferSource);
    return new Uint8Array(await WebAssembly.Module.customSections(wasmBytes, 'name')[0] || new ArrayBuffer(0));
  }

  /**
   * Get agent information
   */
  getAgentInfo(): AgentMetadata | null {
    return this.metadata;
  }

  /**
   * Get loaded LoRA adapters
   */
  getLoadedAdapters(): LoRAAdapter[] {
    return Array.from(this.loadedAdapters.values());
  }

  /**
   * Remove a LoRA adapter
   */
  removeLoRAAdapter(skillId: string): boolean {
    return this.loadedAdapters.delete(skillId);
  }

  /**
   * Check if agent is ready
   */
  isReady(): boolean {
    return this.isInitialized && this.agentModule !== null;
  }

  /**
   * Cleanup resources
   */
  cleanup(): void {
    if (this.agentModule) {
      this.agentModule.cleanup_agent();
    }
    
    this.wasmModule = null;
    this.wasmInstance = null;
    this.agentModule = null;
    this.isInitialized = false;
    this.loadedAdapters.clear();
    this.metadata = null;
    this.primaryAgent = null;

    this.emit('agent_cleanup_completed');
  }

  // Private helper methods
  private validateWASMModule(wasmBytes: Uint8Array): boolean {
    // Check WASM magic number
    const magicNumber = new Uint32Array(wasmBytes.buffer.slice(0, 4))[0];
    return magicNumber === 0x6d736100; // '\0asm'
  }

  private async calculateHash(data: Uint8Array): Promise<string> {
    const hashBuffer = await crypto.subtle.digest('SHA-256', data);
    const hashArray = Array.from(new Uint8Array(hashBuffer));
    return hashArray.map(b => b.toString(16).padStart(2, '0')).join('');
  }

  private serializeLoRAAdapter(adapter: LoRAAdapter): Uint8Array {
    // Simple serialization format for LoRA adapter
    const metadata = JSON.stringify({
      skillId: adapter.skillId,
      skillName: adapter.skillName,
      rank: adapter.rank,
      alpha: adapter.alpha
    });

    const metadataBytes = new TextEncoder().encode(metadata);
    const weightsABytes = new Uint8Array(adapter.weightsA.buffer);
    const weightsBBytes = new Uint8Array(adapter.weightsB.buffer);

    // Create combined buffer
    const totalSize = 4 + metadataBytes.length + 4 + weightsABytes.length + 4 + weightsBBytes.length;
    const buffer = new ArrayBuffer(totalSize);
    const view = new DataView(buffer);
    let offset = 0;

    // Write metadata length and data
    view.setUint32(offset, metadataBytes.length, true);
    offset += 4;
    new Uint8Array(buffer, offset, metadataBytes.length).set(metadataBytes);
    offset += metadataBytes.length;

    // Write weights A length and data
    view.setUint32(offset, weightsABytes.length, true);
    offset += 4;
    new Uint8Array(buffer, offset, weightsABytes.length).set(weightsABytes);
    offset += weightsABytes.length;

    // Write weights B length and data
    view.setUint32(offset, weightsBBytes.length, true);
    offset += 4;
    new Uint8Array(buffer, offset, weightsBBytes.length).set(weightsBBytes);

    return new Uint8Array(buffer);
  }

  private writeToMemory(ptr: number, data: Uint8Array): void {
    if (!this.agentModule) return;
    const memory = new Uint8Array(this.agentModule.memory.buffer);
    memory.set(data, ptr);
  }

  private readFromMemory(ptr: number, length: number): Uint8Array {
    if (!this.agentModule) return new Uint8Array();
    const memory = new Uint8Array(this.agentModule.memory.buffer);
    return memory.slice(ptr, ptr + length);
  }

  private readStringFromMemory(ptr: number, length: number): string {
    const bytes = this.readFromMemory(ptr, length);
    return new TextDecoder().decode(bytes);
  }
}

export default WASMAgentManager;
