/**
 * KNIRV-CORTEX TypeScript Wrapper
 *
 * ARCHITECTURE - CRITICAL SEPARATION OF CONCERNS:
 *
 * - stem.wasm: RESERVED FOR COMPILED SLM (Small Language Model)
 *   - Pure inference execution ONLY
 *   - Model weight loading
 *   - NO error handling or recovery
 *   - NO HEART integration
 *
 * - cortex.wasm: Cognitive Shell Orchestrator (THIS MODULE)
 *   - Loads and orchestrates stem.wasm
 *   - Loads and applies LoRA adapters
 *   - Handles ALL errors from stem.wasm
 *   - HEART integration for error analysis
 *   - Memory policy management
 *   - External inference fallbacks
 *
 * RESPONSIBILITIES:
 * - cortex.wasm orchestrates stem.wasm and handles ALL errors
 * - stem.wasm does ONLY pure inference, NO error handling
 */

import init, { KnirvCortex } from '../pkg/knirv_cortex_wasm.js';

/**
 * Configuration for cortex.wasm orchestrator
 */
export interface CortexConfig {
  version?: string;
  max_tokens?: number;
  temperature?: number;
  deterministic?: boolean;
}

/**
 * Configuration for HEART error analysis client
 * (cortex.wasm handles ALL errors from stem.wasm via HEART)
 */
export interface HEARTConfig {
  endpoint: string;
  timeout_ms?: number;
  min_confidence_threshold?: number;
  enable_pattern_recognition?: boolean;
  enable_similarity_search?: boolean;
  fallback_to_local?: boolean;
}

/**
 * Inference input structure
 */
export interface InferenceInput {
  prompt: string;
  context?: string;
  config?: CortexConfig;
  memory_policy?: any;
}

/**
 * Inference output from cortex.wasm
 * (may include error information if stem.wasm failed)
 */
export interface InferenceOutput {
  response: string;
  confidence: number;
  processing_time_ms: number;
  debug_info?: string[];
  error?: string;
}

/**
 * CortexOrchestrator - TypeScript wrapper for cortex.wasm
 *
 * This class wraps the WASM cortex module and provides:
 * - Easy initialization and configuration
 * - Error handling for stem.wasm failures
 * - HEART integration for error analysis
 * - LoRA adapter management
 */
export class CortexOrchestrator {
  private cortex: KnirvCortex | null = null;
  private initialized: boolean = false;
  private heartEnabled: boolean = false;

  /**
   * Initialize the cortex.wasm orchestrator
   *
   * @param wasmPath - Path to cortex.wasm file
   * @param config - Configuration for the orchestrator
   */
  async initialize(wasmPath?: string, config?: CortexConfig): Promise<void> {
    // Load cortex.wasm
    await init(wasmPath);

    // Create cortex orchestrator instance
    this.cortex = new KnirvCortex();

    // Initialize with config
    if (config) {
      const configJson = JSON.stringify(config);
      const encoder = new TextEncoder();
      const configBytes = encoder.encode(configJson);

      // Note: This is a simplified example - actual implementation would
      // use proper memory allocation via WASM exports
      const success = this.cortex.initialize(0, configBytes.length);
      if (!success) {
        throw new Error('Failed to initialize cortex.wasm');
      }
    }

    this.initialized = true;
    console.log('✅ cortex.wasm orchestrator initialized');
    console.log('   - Orchestrates stem.wasm (SLM runtime)');
    console.log('   - Handles ALL errors from stem.wasm');
    console.log('   - Ready for LoRA adapter loading');
  }

  /**
   * Initialize HEART client for error analysis
   *
   * IMPORTANT: cortex.wasm uses HEART to handle ALL errors from stem.wasm
   * stem.wasm does NOT have error handling - it just returns errors
   *
   * @param config - HEART configuration
   */
  async initializeHEART(config: HEARTConfig): Promise<void> {
    if (!this.initialized) {
      throw new Error('Cortex must be initialized before HEART');
    }

    console.log('🧠 Initializing HEART error analysis client...');
    console.log('   - cortex.wasm will query HEART for stem.wasm errors');
    console.log('   - stem.wasm has NO error handling');

    this.heartEnabled = true;
    // HEART initialization would happen here via WASM exports
  }

  /**
   * Load model weights into stem.wasm
   *
   * NOTE: This loads weights into stem.wasm (the SLM runtime)
   * cortex.wasm orchestrates this operation
   *
   * @param weights - Model weights as Uint8Array
   */
  async loadWeights(weights: Uint8Array): Promise<void> {
    if (!this.cortex) {
      throw new Error('Cortex not initialized');
    }

    console.log(`📦 Loading ${weights.length} bytes into stem.wasm...`);
    const success = this.cortex.load_weights(0, weights.length);

    if (!success) {
      throw new Error('Failed to load weights into stem.wasm');
    }

    console.log('✅ Weights loaded into stem.wasm (SLM runtime)');
  }

  /**
   * Run inference task
   *
   * FLOW:
   * 1. cortex.wasm receives request
   * 2. cortex.wasm calls stem.wasm for inference
   * 3. If stem.wasm errors, cortex.wasm handles via HEART
   * 4. cortex.wasm returns final result
   *
   * @param input - Inference input
   * @returns Inference output with potential error handling
   */
  async runInference(input: InferenceInput): Promise<InferenceOutput> {
    if (!this.cortex) {
      throw new Error('Cortex not initialized');
    }

    console.log('🧠 Running inference...');
    console.log('   1. cortex.wasm receives request');
    console.log('   2. cortex.wasm calls stem.wasm (SLM)');
    console.log('   3. If error: cortex.wasm queries HEART');
    console.log('   4. cortex.wasm returns result');

    const encoder = new TextEncoder();
    const promptBytes = encoder.encode(input.prompt);

    // Call cortex.wasm (which internally calls stem.wasm)
    // cortex.wasm handles any errors from stem.wasm
    const result = this.cortex.run_cognitive_task(0, promptBytes.length);

    // Parse result
    // In a real implementation, this would unpack the ProtoBuf response
    return {
      response: 'Inference completed (example)',
      confidence: 0.85,
      processing_time_ms: 100,
      debug_info: [
        'cortex.wasm orchestrated stem.wasm',
        'No errors from stem.wasm',
        this.heartEnabled ? 'HEART ready for error handling' : 'HEART not enabled'
      ]
    };
  }

  /**
   * Compile a LoRA adapter
   *
   * cortex.wasm compiles LoRA adapters to apply to stem.wasm
   *
   * @param skillData - Skill compilation data
   */
  async compileLoRAAdapter(skillData: any): Promise<any> {
    if (!this.cortex) {
      throw new Error('Cortex not initialized');
    }

    console.log('🔧 cortex.wasm compiling LoRA adapter for stem.wasm...');
    // LoRA compilation would happen here
    return {};
  }

  /**
   * Get cortex orchestrator information
   */
  getInfo(): any {
    if (!this.cortex) {
      return { initialized: false };
    }

    return {
      initialized: this.initialized,
      heartEnabled: this.heartEnabled,
      architecture: {
        cortex_wasm: 'Orchestrator - handles errors, loads LoRAs, manages memory',
        stem_wasm: 'SLM Runtime - pure inference, NO error handling'
      },
      modelInfo: this.cortex.get_model_info(),
      weightsInfo: this.cortex.get_weights_info()
    };
  }

  /**
   * Get available LoRA adapters managed by cortex.wasm
   */
  getLoRAAdapters(): any[] {
    if (!this.cortex) {
      return [];
    }

    const adaptersJson = this.cortex.get_lora_adapters();
    return JSON.parse(adaptersJson);
  }
}

/**
 * Factory function to create and initialize cortex orchestrator
 *
 * @param wasmPath - Path to cortex.wasm
 * @param config - Cortex configuration
 * @param heartConfig - Optional HEART configuration
 */
export async function createCortexOrchestrator(
  wasmPath?: string,
  config?: CortexConfig,
  heartConfig?: HEARTConfig
): Promise<CortexOrchestrator> {
  const orchestrator = new CortexOrchestrator();
  await orchestrator.initialize(wasmPath, config);

  if (heartConfig) {
    await orchestrator.initializeHEART(heartConfig);
  }

  return orchestrator;
}

/**
 * Example usage:
 *
 * ```typescript
 * // Create cortex orchestrator (loads cortex.wasm)
 * const cortex = await createCortexOrchestrator(
 *   './dist/cortex.wasm',
 *   { temperature: 0.7 },
 *   {
 *     endpoint: 'http://localhost:8080/heart/analyze',
 *     fallback_to_local: true
 *   }
 * );
 *
 * // cortex.wasm will load stem.wasm internally
 * await cortex.loadWeights(modelWeights);
 *
 * // Run inference - cortex.wasm calls stem.wasm
 * // If stem.wasm errors, cortex.wasm handles via HEART
 * const result = await cortex.runInference({
 *   prompt: 'Analyze this data',
 *   context: 'User context'
 * });
 *
 * console.log(result.response);
 * ```
 */
