/**
 * WASM Orchestrator
 * 
 * Elegant orchestration system that manages:
 * 1. Cognitive-Shell WASM (agent-core compiled from templates)
 * 2. Model WASM (HRM, Phi-3, RecurrentGemma, or TinyLlama)
 * 
 * Both WASM modules intercommunicate with the sensory-shell for complete AI processing
 */

import { EventEmitter } from './EventEmitter';
import { AgentCoreInterface, SensoryInput, CognitiveResponse } from './AgentCoreInterface';

export interface ModelWASM {
  // Model WASM exported functions
  modelInference: (input: string, context: string) => Promise<string>;
  modelLoadWeights: (weights: Uint8Array) => Promise<boolean>;
  modelGetInfo: () => string;
  modelSetConfig: (config: string) => boolean;
}

export interface WASMModuleInfo {
  name: string;
  type: 'cognitive-shell' | 'model';
  version: string;
  size: number;
  capabilities: string[];
  loaded: boolean;
  initialized: boolean;
}

export interface ModelConfig {
  modelType: 'hrm_cognitive' | 'knirv_cortex' | 'phi-3-mini' | 'recurrentgemma-2b' | 'tinyllama';
  modelPath?: string;
  weightsPath?: string;
  maxTokens: number;
  temperature: number;
  topP: number;
  contextLength: number;
}

export interface OrchestrationConfig {
  defaultModel: ModelConfig;
  cognitiveShellPath?: string;
  enableModelFallback: boolean;
  enableCrossWASMCommunication: boolean;
  maxConcurrentInferences: number;
  timeoutMs: number;
}

/**
 * WASM Orchestrator - Manages dual WASM architecture
 */
export class WASMOrchestrator extends EventEmitter {
  private agentCoreInterface: AgentCoreInterface;
  private modelWASM: ModelWASM | null = null;
  private modelModule: WebAssembly.Module | null = null;
  private modelInstance: WebAssembly.Instance | null = null;
  
  private cognitiveShellLoaded = false;
  private modelLoaded = false;
  private isInitialized = false;
  
  private config: OrchestrationConfig;
  private sessionId: string;
  private inferenceQueue: Array<{
    input: SensoryInput;
    resolve: (response: CognitiveResponse) => void;
    reject: (error: Error) => void;
    timestamp: number;
  }> = [];

  constructor(config: OrchestrationConfig) {
    super();
    this.config = config;
    this.sessionId = this.generateSessionId();
    this.agentCoreInterface = new AgentCoreInterface();
    this.setupEventHandlers();
  }

  private setupEventHandlers(): void {
    // Agent-core events
    this.agentCoreInterface.on('agent_core_initialized', () => {
      this.cognitiveShellLoaded = true;
      this.emit('cognitive_shell_loaded');
      this.checkInitializationComplete();
    });

    this.agentCoreInterface.on('cognitive_processing_completed', (data) => {
      this.emit('cognitive_processing_completed', data);
    });

    // Model WASM events
    this.on('model_loaded', () => {
      this.modelLoaded = true;
      this.checkInitializationComplete();
    });
  }

  private checkInitializationComplete(): void {
    if (this.cognitiveShellLoaded && this.modelLoaded && !this.isInitialized) {
      this.isInitialized = true;
      this.emit('orchestrator_initialized');
      this.processQueuedInferences();
    }
  }

  /**
   * Initialize the WASM orchestrator with both modules
   */
  async initialize(): Promise<boolean> {
    try {
      this.emit('orchestrator_initialization_started');

      // Load default cognitive-shell WASM
      const cognitiveShellLoaded = await this.loadCognitiveShell();
      if (!cognitiveShellLoaded) {
        throw new Error('Failed to load cognitive-shell WASM');
      }

      // Load default model WASM
      const modelLoaded = await this.loadModel(this.config.defaultModel);
      if (!modelLoaded) {
        throw new Error('Failed to load model WASM');
      }

      return true;

    } catch (error) {
      this.emit('orchestrator_initialization_failed', { error: error.message });
      return false;
    }
  }

  /**
   * Load cognitive-shell WASM (compiled from templates)
   */
  async loadCognitiveShell(wasmBytes?: Uint8Array): Promise<boolean> {
    try {
      this.emit('cognitive_shell_loading_started');

      let cognitiveWASM: Uint8Array;

      if (wasmBytes) {
        // Use provided WASM bytes (uploaded agent-core)
        cognitiveWASM = wasmBytes;
      } else {
        // Load default cognitive-shell WASM
        cognitiveWASM = await this.loadDefaultCognitiveShell();
      }

      const success = await this.agentCoreInterface.initializeAgentCore(cognitiveWASM);
      
      if (success) {
        this.emit('cognitive_shell_loaded', {
          size: cognitiveWASM.length,
          type: wasmBytes ? 'uploaded' : 'default'
        });
      }

      return success;

    } catch (error) {
      this.emit('cognitive_shell_loading_failed', { error: error.message });
      return false;
    }
  }

  /**
   * Load model WASM (HRM, Phi-3, RecurrentGemma, or TinyLlama)
   */
  async loadModel(modelConfig: ModelConfig): Promise<boolean> {
    try {
      this.emit('model_loading_started', { modelType: modelConfig.modelType });

      let modelWASM: Uint8Array;

      switch (modelConfig.modelType) {
        case 'hrm_cognitive':
          modelWASM = await this.loadHRMCognitive();
          break;
        case 'knirv_cortex':
          modelWASM = await this.loadKNIRVCortex();
          break;
        case 'phi-3-mini':
          modelWASM = await this.loadPhi3Mini(modelConfig);
          break;
        case 'recurrentgemma-2b':
          modelWASM = await this.loadRecurrentGemma(modelConfig);
          break;
        case 'tinyllama':
          modelWASM = await this.loadTinyLlama(modelConfig);
          break;
        default:
          throw new Error(`Unknown model type: ${modelConfig.modelType}`);
      }

      // Initialize model WASM
      const success = await this.initializeModelWASM(modelWASM, modelConfig);
      
      if (success) {
        this.emit('model_loaded', {
          modelType: modelConfig.modelType,
          size: modelWASM.length
        });
      }

      return success;

    } catch (error) {
      this.emit('model_loading_failed', { 
        modelType: modelConfig.modelType, 
        error: error.message 
      });
      return false;
    }
  }

  private async loadDefaultCognitiveShell(): Promise<Uint8Array> {
    try {
      // Try to load compiled cognitive-shell WASM first
      const response = await fetch('/wasm/cognitive-shell.wasm');
      if (response.ok) {
        return new Uint8Array(await response.arrayBuffer());
      }
    } catch (error) {
      console.warn('Failed to load compiled cognitive-shell WASM, using fallback:', error);
    }

    // Fallback: Generate a minimal WASM module that provides the required interface
    return this.generateMinimalCognitiveShellWASM();
  }

  private generateMinimalCognitiveShellWASM(): Uint8Array {
    // Generate a minimal WASM module with the required exports
    // This is a temporary solution until real WASM compilation is implemented
    const wasmModule = new Uint8Array([
      0x00, 0x61, 0x73, 0x6d, // WASM magic number
      0x01, 0x00, 0x00, 0x00, // WASM version

      // Type section
      0x01, 0x07, 0x01, 0x60, 0x02, 0x7f, 0x7f, 0x7f,

      // Function section
      0x03, 0x06, 0x05, 0x00, 0x00, 0x00, 0x00, 0x00,

      // Memory section
      0x05, 0x03, 0x01, 0x00, 0x01,

      // Export section
      0x07, 0x5a, 0x05,
      0x06, 0x6d, 0x65, 0x6d, 0x6f, 0x72, 0x79, 0x02, 0x00,
      0x11, 0x61, 0x67, 0x65, 0x6e, 0x74, 0x43, 0x6f, 0x72, 0x65, 0x45, 0x78, 0x65, 0x63, 0x75, 0x74, 0x65, 0x00, 0x00,
      0x15, 0x61, 0x67, 0x65, 0x6e, 0x74, 0x43, 0x6f, 0x72, 0x65, 0x45, 0x78, 0x65, 0x63, 0x75, 0x74, 0x65, 0x54, 0x6f, 0x6f, 0x6c, 0x00, 0x01,
      0x13, 0x61, 0x67, 0x65, 0x6e, 0x74, 0x43, 0x6f, 0x72, 0x65, 0x4c, 0x6f, 0x61, 0x64, 0x4c, 0x6f, 0x52, 0x41, 0x00, 0x02,
      0x15, 0x61, 0x67, 0x65, 0x6e, 0x74, 0x43, 0x6f, 0x72, 0x65, 0x41, 0x70, 0x70, 0x6c, 0x79, 0x53, 0x6b, 0x69, 0x6c, 0x6c, 0x00, 0x03,
      0x13, 0x61, 0x67, 0x65, 0x6e, 0x74, 0x43, 0x6f, 0x72, 0x65, 0x47, 0x65, 0x74, 0x53, 0x74, 0x61, 0x74, 0x75, 0x73, 0x00, 0x04,

      // Code section with minimal function implementations
      0x0a, 0x2d, 0x05,
      0x07, 0x00, 0x41, 0x00, 0x41, 0x00, 0x0b, // agentCoreExecute: return 0
      0x07, 0x00, 0x41, 0x00, 0x41, 0x00, 0x0b, // agentCoreExecuteTool: return 0
      0x04, 0x00, 0x41, 0x01, 0x0b,             // agentCoreLoadLoRA: return 1 (true)
      0x04, 0x00, 0x41, 0x01, 0x0b,             // agentCoreApplySkill: return 1 (true)
      0x04, 0x00, 0x41, 0x00, 0x0b              // agentCoreGetStatus: return 0
    ]);

    return wasmModule;
  }

  private async loadHRMCognitive(): Promise<Uint8Array> {
    try {
      // Try to load HRM cognitive model WASM
      const response = await fetch('/models/hrm_cognitive.wasm');
      if (response.ok) {
        return new Uint8Array(await response.arrayBuffer());
      }
    } catch (error) {
      console.warn('Failed to load HRM cognitive WASM, using fallback:', error);
    }
    return this.generateMinimalModelWASM('hrm_cognitive');
  }

  private async loadKNIRVCortex(): Promise<Uint8Array> {
    try {
      // Try to load KNIRV Cortex model WASM
      const response = await fetch('/models/knirv_cortex_wasm.wasm');
      if (response.ok) {
        return new Uint8Array(await response.arrayBuffer());
      }
    } catch (error) {
      console.warn('Failed to load KNIRV Cortex WASM, using fallback:', error);
    }
    return this.generateMinimalModelWASM('knirv_cortex');
  }

  private async loadPhi3Mini(config: ModelConfig): Promise<Uint8Array> {
    try {
      // Try to load Phi-3 Mini model WASM
      const modelPath = config.modelPath || '/models/phi-3-mini.wasm';
      const response = await fetch(modelPath);
      if (response.ok) {
        return new Uint8Array(await response.arrayBuffer());
      }
    } catch (error) {
      console.warn('Failed to load Phi-3 Mini WASM, using fallback:', error);
    }
    return this.generateMinimalModelWASM('phi-3-mini');
  }

  private async loadRecurrentGemma(config: ModelConfig): Promise<Uint8Array> {
    try {
      // Try to load RecurrentGemma model WASM
      const modelPath = config.modelPath || '/models/recurrentgemma-2b.wasm';
      const response = await fetch(modelPath);
      if (response.ok) {
        return new Uint8Array(await response.arrayBuffer());
      }
    } catch (error) {
      console.warn('Failed to load RecurrentGemma WASM, using fallback:', error);
    }
    return this.generateMinimalModelWASM('recurrentgemma-2b');
  }

  private async loadTinyLlama(config: ModelConfig): Promise<Uint8Array> {
    try {
      // Try to load TinyLlama model WASM
      const modelPath = config.modelPath || '/models/tinyllama.wasm';
      const response = await fetch(modelPath);
      if (response.ok) {
        return new Uint8Array(await response.arrayBuffer());
      }
    } catch (error) {
      console.warn('Failed to load TinyLlama WASM, using fallback:', error);
    }
    return this.generateMinimalModelWASM('tinyllama');
  }

  private generateMinimalModelWASM(modelType: string): Uint8Array {
    // Generate a minimal WASM module for model inference
    // This provides the basic interface until real models are available
    const wasmModule = new Uint8Array([
      0x00, 0x61, 0x73, 0x6d, // WASM magic number
      0x01, 0x00, 0x00, 0x00, // WASM version

      // Type section
      0x01, 0x05, 0x01, 0x60, 0x01, 0x7f, 0x7f,

      // Function section
      0x03, 0x04, 0x03, 0x00, 0x00, 0x00,

      // Memory section
      0x05, 0x03, 0x01, 0x00, 0x01,

      // Export section
      0x07, 0x2a, 0x03,
      0x06, 0x6d, 0x65, 0x6d, 0x6f, 0x72, 0x79, 0x02, 0x00,
      0x0c, 0x6d, 0x6f, 0x64, 0x65, 0x6c, 0x49, 0x6e, 0x66, 0x65, 0x72, 0x65, 0x6e, 0x63, 0x65, 0x00, 0x00,
      0x0d, 0x6d, 0x6f, 0x64, 0x65, 0x6c, 0x47, 0x65, 0x74, 0x49, 0x6e, 0x66, 0x6f, 0x00, 0x01,

      // Code section with minimal implementations
      0x0a, 0x0e, 0x03,
      0x04, 0x00, 0x41, 0x00, 0x0b, // modelInference: return 0
      0x04, 0x00, 0x41, 0x00, 0x0b, // modelGetInfo: return 0
    ]);

    return wasmModule;
  }

  private async initializeModelWASM(wasmBytes: Uint8Array, config: ModelConfig): Promise<boolean> {
    try {
      // Compile model WASM module
      this.modelModule = await WebAssembly.compile(wasmBytes);

      // Create WASM instance with imports
      this.modelInstance = await WebAssembly.instantiate(this.modelModule, {
        env: {
          memory: new WebAssembly.Memory({ 
            initial: Math.ceil(config.contextLength / 1024), 
            maximum: Math.ceil(config.contextLength / 512) 
          }),
          
          // Console logging from model WASM
          console_log: (ptr: number, len: number) => {
            const message = this.readStringFromWASM(ptr, len);
            console.log(`[Model-${config.modelType}]: ${message}`);
            this.emit('model_log', { modelType: config.modelType, message });
          },

          // Cross-WASM communication callback
          cognitive_shell_callback: (type: number, dataPtr: number, dataLen: number) => {
            if (this.config.enableCrossWASMCommunication) {
              const data = this.readStringFromWASM(dataPtr, dataLen);
              this.handleCrossWASMCommunication('model-to-cognitive', type, data);
            }
          }
        }
      });

      // Get exported functions
      this.modelWASM = this.modelInstance.exports as any;

      // Verify required functions exist
      if (!this.modelWASM?.modelInference || 
          !this.modelWASM?.modelGetInfo) {
        throw new Error('Required model functions not found in WASM module');
      }

      // Configure the model
      const modelConfigString = JSON.stringify({
        maxTokens: config.maxTokens,
        temperature: config.temperature,
        topP: config.topP,
        contextLength: config.contextLength
      });

      this.modelWASM.modelSetConfig(modelConfigString);

      // Load weights if provided
      if (config.weightsPath) {
        const weightsResponse = await fetch(config.weightsPath);
        const weightsBytes = new Uint8Array(await weightsResponse.arrayBuffer());
        await this.modelWASM.modelLoadWeights(weightsBytes);
      }

      return true;

    } catch (error) {
      console.error('Failed to initialize model WASM:', error);
      return false;
    }
  }

  /**
   * Process sensory input through the orchestrated WASM modules
   */
  async processSensoryInput(input: SensoryInput): Promise<CognitiveResponse> {
    if (!this.isInitialized) {
      // Queue the inference if not ready
      return new Promise((resolve, reject) => {
        this.inferenceQueue.push({
          input,
          resolve,
          reject,
          timestamp: Date.now()
        });
      });
    }

    try {
      this.emit('orchestrated_processing_started', { input });

      // Step 1: Process through cognitive-shell for reasoning and planning
      const cognitiveResponse = await this.agentCoreInterface.processSensoryInput(input);

      // Step 2: If cognitive-shell requests model inference, route to model WASM
      if (cognitiveResponse.metadata?.requiresModelInference && this.modelWASM) {
        const modelInput = JSON.stringify({
          prompt: cognitiveResponse.result,
          context: input.data,
          cognitiveContext: cognitiveResponse.metadata
        });

        const modelContext = JSON.stringify({
          sessionId: this.sessionId,
          inputType: input.type,
          cognitiveShellResponse: cognitiveResponse
        });

        const modelResult = await this.modelWASM.modelInference(modelInput, modelContext);
        const parsedModelResult = JSON.parse(modelResult);

        // Step 3: Send model result back to cognitive-shell for post-processing
        const finalInput: SensoryInput = {
          ...input,
          data: {
            originalInput: input.data,
            modelResult: parsedModelResult,
            cognitiveContext: cognitiveResponse
          },
          type: 'model-result'
        };

        const finalResponse = await this.agentCoreInterface.processSensoryInput(finalInput);
        
        this.emit('orchestrated_processing_completed', { 
          input, 
          cognitiveResponse, 
          modelResult: parsedModelResult, 
          finalResponse 
        });

        return finalResponse;
      }

      // Return cognitive-shell response if no model inference needed
      this.emit('orchestrated_processing_completed', { input, cognitiveResponse });
      return cognitiveResponse;

    } catch (error) {
      this.emit('orchestrated_processing_failed', { input, error: error.message });
      throw error;
    }
  }

  /**
   * Handle cross-WASM communication
   */
  private handleCrossWASMCommunication(direction: string, type: number, data: string): void {
    try {
      const parsedData = JSON.parse(data);
      
      this.emit('cross_wasm_communication', {
        direction,
        type,
        data: parsedData,
        timestamp: Date.now()
      });

      // Route communication between WASM modules
      if (direction === 'model-to-cognitive' && this.agentCoreInterface.isReady()) {
        // Forward model communication to cognitive-shell
        // Implementation would depend on specific communication protocol
      } else if (direction === 'cognitive-to-model' && this.modelWASM) {
        // Forward cognitive-shell communication to model
        // Implementation would depend on specific communication protocol
      }

    } catch (error) {
      console.error('Failed to handle cross-WASM communication:', error);
    }
  }

  private async processQueuedInferences(): Promise<void> {
    const queue = [...this.inferenceQueue];
    this.inferenceQueue = [];

    for (const { input, resolve, reject } of queue) {
      try {
        const response = await this.processSensoryInput(input);
        resolve(response);
      } catch (error) {
        reject(error);
      }
    }
  }

  /**
   * Get information about loaded WASM modules
   */
  getModuleInfo(): WASMModuleInfo[] {
    const modules: WASMModuleInfo[] = [];

    if (this.cognitiveShellLoaded) {
      modules.push({
        name: 'Cognitive Shell',
        type: 'cognitive-shell',
        version: '1.0.0',
        size: 0, // Would be actual size
        capabilities: ['reasoning', 'planning', 'skill-execution', 'lora-adaptation'],
        loaded: true,
        initialized: this.agentCoreInterface.isReady()
      });
    }

    if (this.modelLoaded && this.modelWASM) {
      const modelInfo = JSON.parse(this.modelWASM.modelGetInfo());
      modules.push({
        name: modelInfo.name || this.config.defaultModel.modelType,
        type: 'model',
        version: modelInfo.version || '1.0.0',
        size: modelInfo.size || 0,
        capabilities: modelInfo.capabilities || ['text-generation', 'inference'],
        loaded: true,
        initialized: true
      });
    }

    return modules;
  }

  /**
   * Switch to a different model
   */
  async switchModel(newModelConfig: ModelConfig): Promise<boolean> {
    try {
      this.emit('model_switching_started', { 
        from: this.config.defaultModel.modelType, 
        to: newModelConfig.modelType 
      });

      // Dispose current model
      if (this.modelInstance) {
        this.modelWASM = null;
        this.modelInstance = null;
        this.modelModule = null;
        this.modelLoaded = false;
      }

      // Load new model
      const success = await this.loadModel(newModelConfig);
      
      if (success) {
        this.config.defaultModel = newModelConfig;
        this.emit('model_switched', { modelType: newModelConfig.modelType });
      }

      return success;

    } catch (error) {
      this.emit('model_switching_failed', { error: error.message });
      return false;
    }
  }

  private readStringFromWASM(ptr: number, len: number): string {
    if (!this.modelInstance) return '';
    
    const memory = this.modelInstance.exports.memory as WebAssembly.Memory;
    const bytes = new Uint8Array(memory.buffer, ptr, len);
    return new TextDecoder().decode(bytes);
  }

  private generateSessionId(): string {
    return `orchestrator-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`;
  }

  isReady(): boolean {
    return this.isInitialized;
  }

  /**
   * Start the WASM orchestrator
   */
  async start(): Promise<boolean> {
    try {
      this.emit('orchestrator_starting');

      // Initialize the orchestrator if not already done
      if (!this.isInitialized) {
        const initialized = await this.initialize();
        if (!initialized) {
          throw new Error('Failed to initialize WASM orchestrator');
        }
      }

      this.emit('orchestrator_started');
      return true;
    } catch (error) {
      this.emit('orchestrator_start_failed', { error: error.message });
      return false;
    }
  }

  /**
   * Stop the WASM orchestrator
   */
  async stop(): Promise<boolean> {
    try {
      this.emit('orchestrator_stopping');

      // Clear any pending inferences
      this.inferenceQueue.forEach(({ reject }) => {
        reject(new Error('Orchestrator stopped'));
      });
      this.inferenceQueue = [];

      // Stop agent-core interface
      if (this.agentCoreInterface) {
        await this.agentCoreInterface.dispose();
      }

      this.emit('orchestrator_stopped');
      return true;
    } catch (error) {
      this.emit('orchestrator_stop_failed', { error: error.message });
      return false;
    }
  }

  async dispose(): Promise<void> {
    // Stop the orchestrator first
    await this.stop();

    // Reset state
    this.modelWASM = null;
    this.modelInstance = null;
    this.modelModule = null;
    this.cognitiveShellLoaded = false;
    this.modelLoaded = false;
    this.isInitialized = false;

    this.emit('orchestrator_disposed');
  }
}

export default WASMOrchestrator;
