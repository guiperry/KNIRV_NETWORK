/**
 * Agent-Core Interface
 * 
 * Communication bridge between sensory-shell (client) and agent-core (WASM)
 * Establishes clear communication channels and protocols
 */

import { EventEmitter } from './EventEmitter';

export interface AgentCoreWASM {
  // WASM exported functions
  agentCoreExecute: (input: string, context: string) => Promise<string>;
  agentCoreExecuteTool: (toolName: string, parameters: string, context: string) => Promise<string>;
  agentCoreLoadLoRA: (adapter: string) => Promise<boolean>;
  agentCoreGetStatus: () => string;
}

export interface SensoryInput {
  type: 'voice' | 'visual' | 'text' | 'gesture';
  data: any;
  timestamp: number;
  sessionId: string;
  userId?: string;
  metadata?: Record<string, any>;
}

export interface CognitiveResponse {
  success: boolean;
  result?: any;
  error?: string;
  processingTime: number;
  confidence: number;
  source: 'agent-core' | 'fallback';
  metadata?: Record<string, any>;
}

export interface AgentCoreStatus {
  agentId: string;
  agentName: string;
  version: string;
  initialized: boolean;
  cognitiveEngine: string;
  availableTools: string[];
  memorySize: number;
}

export interface LoRAAdapter {
  skillId: string;
  skillName: string;
  weightsA: Float32Array;
  weightsB: Float32Array;
  rank: number;
  alpha: number;
  metadata?: Record<string, any>;
}

/**
 * Agent-Core Interface Manager
 * Manages communication between sensory-shell and embedded agent-core WASM
 */
export class AgentCoreInterface extends EventEmitter {
  private wasmModule: WebAssembly.Module | null = null;
  private wasmInstance: WebAssembly.Instance | null = null;
  private agentCore: AgentCoreWASM | null = null;
  private isInitialized = false;
  private sessionId: string;
  private communicationQueue: Array<{ resolve: Function; reject: Function; timeout: NodeJS.Timeout }> = [];

  constructor() {
    super();
    this.sessionId = this.generateSessionId();
  }

  /**
   * Initialize agent-core WASM module
   */
  async initializeAgentCore(wasmBytes: Uint8Array): Promise<boolean> {
    try {
      this.emit('agent_core_initialization_started');

      // Compile WASM module
      this.wasmModule = await WebAssembly.compile(wasmBytes);

      // Create WASM instance with imports
      this.wasmInstance = await WebAssembly.instantiate(this.wasmModule, {
        env: {
          memory: new WebAssembly.Memory({ initial: 256, maximum: 512 }),
          
          // Console logging from WASM
          console_log: (ptr: number, len: number) => {
            const message = this.readStringFromWASM(ptr, len);
            console.log(`[Agent-Core]: ${message}`);
            this.emit('agent_core_log', { message });
          },
          
          console_error: (ptr: number, len: number) => {
            const message = this.readStringFromWASM(ptr, len);
            console.error(`[Agent-Core]: ${message}`);
            this.emit('agent_core_error', { message });
          },

          // Sensory-shell callbacks from WASM
          sensory_shell_callback: (type: number, dataPtr: number, dataLen: number) => {
            const data = this.readStringFromWASM(dataPtr, dataLen);
            this.handleAgentCoreCallback(type, data);
          }
        }
      });

      // Get exported functions
      this.agentCore = this.wasmInstance.exports as any;

      // Verify required functions exist
      if (!this.agentCore?.agentCoreExecute || 
          !this.agentCore?.agentCoreExecuteTool ||
          !this.agentCore?.agentCoreLoadLoRA ||
          !this.agentCore?.agentCoreGetStatus) {
        throw new Error('Required agent-core functions not found in WASM module');
      }

      this.isInitialized = true;
      this.emit('agent_core_initialized');
      
      return true;

    } catch (error) {
      this.emit('agent_core_initialization_failed', { error: error.message });
      console.error('Failed to initialize agent-core:', error);
      return false;
    }
  }

  /**
   * Process sensory input through agent-core
   */
  async processSensoryInput(input: SensoryInput): Promise<CognitiveResponse> {
    if (!this.isInitialized || !this.agentCore) {
      throw new Error('Agent-core not initialized');
    }

    const startTime = Date.now();

    try {
      this.emit('cognitive_processing_started', { input });

      // Prepare input for WASM
      const inputData = JSON.stringify({
        type: input.type,
        data: input.data,
        timestamp: input.timestamp,
        sessionId: input.sessionId,
        userId: input.userId,
        metadata: input.metadata
      });

      const context = JSON.stringify({
        sessionId: this.sessionId,
        inputType: input.type,
        timestamp: Date.now(),
        sensoryShellVersion: '1.0.0'
      });

      // Execute through agent-core
      const resultString = await this.agentCore.agentCoreExecute(inputData, context);
      const result = JSON.parse(resultString);

      const processingTime = Date.now() - startTime;

      const response: CognitiveResponse = {
        success: !result.error,
        result: result.result,
        error: result.error,
        processingTime,
        confidence: result.confidence || 0.8,
        source: 'agent-core',
        metadata: {
          ...result.metadata,
          sessionId: this.sessionId,
          inputType: input.type
        }
      };

      this.emit('cognitive_processing_completed', { input, response });
      return response;

    } catch (error) {
      const processingTime = Date.now() - startTime;
      
      const response: CognitiveResponse = {
        success: false,
        error: error.message,
        processingTime,
        confidence: 0,
        source: 'agent-core',
        metadata: {
          sessionId: this.sessionId,
          inputType: input.type,
          errorType: error.constructor.name
        }
      };

      this.emit('cognitive_processing_failed', { input, response, error });
      return response;
    }
  }

  /**
   * Execute a specific tool through agent-core
   */
  async executeTool(toolName: string, parameters: any, context: any = {}): Promise<CognitiveResponse> {
    if (!this.isInitialized || !this.agentCore) {
      throw new Error('Agent-core not initialized');
    }

    const startTime = Date.now();

    try {
      this.emit('tool_execution_started', { toolName, parameters });

      const parametersString = JSON.stringify(parameters);
      const contextString = JSON.stringify({
        ...context,
        sessionId: this.sessionId,
        timestamp: Date.now()
      });

      const resultString = await this.agentCore.agentCoreExecuteTool(toolName, parametersString, contextString);
      const result = JSON.parse(resultString);

      const processingTime = Date.now() - startTime;

      const response: CognitiveResponse = {
        success: result.success,
        result: result.result,
        error: result.error,
        processingTime,
        confidence: 0.9, // Tools typically have high confidence
        source: 'agent-core',
        metadata: {
          toolName,
          sessionId: this.sessionId,
          ...result.metadata
        }
      };

      this.emit('tool_execution_completed', { toolName, parameters, response });
      return response;

    } catch (error) {
      const processingTime = Date.now() - startTime;
      
      const response: CognitiveResponse = {
        success: false,
        error: error.message,
        processingTime,
        confidence: 0,
        source: 'agent-core',
        metadata: {
          toolName,
          sessionId: this.sessionId,
          errorType: error.constructor.name
        }
      };

      this.emit('tool_execution_failed', { toolName, parameters, response, error });
      return response;
    }
  }

  /**
   * Load LoRA adapter into agent-core
   */
  async loadLoRAAdapter(adapter: LoRAAdapter): Promise<boolean> {
    if (!this.isInitialized || !this.agentCore) {
      throw new Error('Agent-core not initialized');
    }

    try {
      this.emit('lora_loading_started', { skillId: adapter.skillId });

      // Serialize LoRA adapter for WASM
      const adapterString = JSON.stringify({
        skillId: adapter.skillId,
        skillName: adapter.skillName,
        weightsA: Array.from(adapter.weightsA),
        weightsB: Array.from(adapter.weightsB),
        rank: adapter.rank,
        alpha: adapter.alpha,
        metadata: adapter.metadata
      });

      const success = await this.agentCore.agentCoreLoadLoRA(adapterString);

      if (success) {
        this.emit('lora_loaded', { skillId: adapter.skillId, skillName: adapter.skillName });
      } else {
        this.emit('lora_loading_failed', { skillId: adapter.skillId, error: 'Load operation failed' });
      }

      return success;

    } catch (error) {
      this.emit('lora_loading_failed', { skillId: adapter.skillId, error: error.message });
      return false;
    }
  }

  /**
   * Get agent-core status
   */
  async getAgentCoreStatus(): Promise<AgentCoreStatus | null> {
    if (!this.isInitialized || !this.agentCore) {
      return null;
    }

    try {
      const statusString = this.agentCore.agentCoreGetStatus();
      return JSON.parse(statusString) as AgentCoreStatus;
    } catch (error) {
      console.error('Failed to get agent-core status:', error);
      return null;
    }
  }

  /**
   * Handle callbacks from agent-core to sensory-shell
   */
  private handleAgentCoreCallback(type: number, data: string): void {
    try {
      const parsedData = JSON.parse(data);

      switch (type) {
        case 1: // Request sensory input
          this.emit('sensory_input_requested', parsedData);
          break;
        case 2: // Request tool execution
          this.emit('tool_execution_requested', parsedData);
          break;
        case 3: // Memory update
          this.emit('memory_update', parsedData);
          break;
        case 4: // Status update
          this.emit('status_update', parsedData);
          break;
        default:
          console.warn('Unknown callback type from agent-core:', type);
      }
    } catch (error) {
      console.error('Failed to handle agent-core callback:', error);
    }
  }

  /**
   * Read string from WASM memory
   */
  private readStringFromWASM(ptr: number, len: number): string {
    if (!this.wasmInstance) return '';
    
    const memory = this.wasmInstance.exports.memory as WebAssembly.Memory;
    const bytes = new Uint8Array(memory.buffer, ptr, len);
    return new TextDecoder().decode(bytes);
  }

  /**
   * Write string to WASM memory
   */
  private writeStringToWASM(str: string): { ptr: number; len: number } {
    if (!this.wasmInstance) return { ptr: 0, len: 0 };
    
    const memory = this.wasmInstance.exports.memory as WebAssembly.Memory;
    const malloc = this.wasmInstance.exports.malloc as Function;
    
    const bytes = new TextEncoder().encode(str);
    const ptr = malloc(bytes.length);
    const memoryView = new Uint8Array(memory.buffer);
    memoryView.set(bytes, ptr);
    
    return { ptr, len: bytes.length };
  }

  /**
   * Generate unique session ID
   */
  private generateSessionId(): string {
    return `session-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`;
  }

  /**
   * Check if agent-core is ready
   */
  isReady(): boolean {
    return this.isInitialized && this.agentCore !== null;
  }

  /**
   * Get current session ID
   */
  getSessionId(): string {
    return this.sessionId;
  }

  /**
   * Cleanup resources
   */
  async dispose(): Promise<void> {
    // Clear any pending operations
    this.communicationQueue.forEach(({ reject, timeout }) => {
      clearTimeout(timeout);
      reject(new Error('Interface disposed'));
    });
    this.communicationQueue = [];

    // Reset state
    this.wasmModule = null;
    this.wasmInstance = null;
    this.agentCore = null;
    this.isInitialized = false;

    this.emit('agent_core_disposed');
  }
}

export default AgentCoreInterface;
