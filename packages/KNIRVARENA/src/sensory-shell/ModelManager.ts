/**
 * Model Manager
 * 
 * Manages the different SLM (Small Language Model) options:
 * - Default: hrm_cognitive.wasm or knirv_cortex.wasm
 * - Alternative: Phi-3 Mini, RecurrentGemma, TinyLlama (from ALT_MODELS.md)
 */

import { EventEmitter } from './EventEmitter';

export interface ModelDefinition {
  id: string;
  name: string;
  description: string;
  type: 'hrm' | 'cortex' | 'phi3' | 'gemma' | 'llama';
  size: string;
  parameters: string;
  wasmPath: string;
  weightsPath?: string;
  configPath?: string;
  capabilities: string[];
  license: string;
  source: 'builtin' | 'huggingface' | 'custom';
  architecture: string;
  contextLength: number;
  recommended: boolean;
}

export interface ModelStatus {
  id: string;
  loaded: boolean;
  initialized: boolean;
  size: number;
  loadTime?: number;
  error?: string;
}

export class ModelManager extends EventEmitter {
  private availableModels: Map<string, ModelDefinition> = new Map();
  private modelStatuses: Map<string, ModelStatus> = new Map();
  private currentModel: string | null = null;

  constructor() {
    super();
    this.initializeBuiltinModels();
  }

  /**
   * Initialize the model manager
   */
  async initialize(): Promise<boolean> {
    try {
      this.emit('model_manager_initializing');

      // Check availability of builtin models
      const builtinModels = this.getModelsBySource('builtin');
      for (const model of builtinModels) {
        const available = await this.isModelAvailable(model.id);
        this.updateModelStatus(model.id, {
          loaded: available,
          initialized: available
        });
      }

      // Set default model if none is set
      if (!this.currentModel) {
        try {
          const defaultModel = this.getDefaultModel();
          this.setCurrentModel(defaultModel.id);
        } catch {
          // No models available, that's okay for now
        }
      }

      this.emit('model_manager_initialized');
      return true;
    } catch (error) {
      this.emit('model_manager_initialization_failed', { error: error instanceof Error ? error.message : String(error) });
      return false;
    }
  }

  private initializeBuiltinModels(): void {
    // Default KNIRV models
    this.registerModel({
      id: 'hrm-cognitive',
      name: 'HRM Cognitive',
      description: 'Hierarchical Reasoning Model - Default cognitive processing engine',
      type: 'hrm',
      size: '27MB',
      parameters: '27M',
      wasmPath: '/models/hrm_cognitive.wasm',
      weightsPath: '/models/hrm_cognitive.safetensors',
      capabilities: ['reasoning', 'planning', 'hierarchical-thinking', 'cognitive-processing'],
      license: 'MIT',
      source: 'builtin',
      architecture: 'Hierarchical Reasoning Model',
      contextLength: 4096,
      recommended: true
    });

    this.registerModel({
      id: 'knirv-cortex',
      name: 'KNIRV Cortex',
      description: 'KNIRV Cortex - Advanced cognitive processing with LoRA adaptation',
      type: 'cortex',
      size: '45MB',
      parameters: '40M',
      wasmPath: '/models/knirv_cortex_wasm.wasm',
      weightsPath: '/models/knirv_cortex.safetensors',
      capabilities: ['reasoning', 'adaptation', 'lora-integration', 'skill-learning'],
      license: 'MIT',
      source: 'builtin',
      architecture: 'Enhanced Transformer with LoRA',
      contextLength: 8192,
      recommended: false
    });

    // Alternative models from ALT_MODELS.md
    this.registerModel({
      id: 'phi3-mini',
      name: 'Phi-3 Mini',
      description: 'Microsoft Phi-3 Mini - Excellent performance for its size, outperforming larger models',
      type: 'phi3',
      size: '3.8GB',
      parameters: '3.8B',
      wasmPath: '/models/phi-3-mini.wasm',
      weightsPath: '/models/phi-3-mini.safetensors',
      capabilities: ['text-generation', 'reasoning', 'instruction-following', 'chat'],
      license: 'MIT',
      source: 'huggingface',
      architecture: 'Transformer',
      contextLength: 4096,
      recommended: true
    });

    this.registerModel({
      id: 'recurrentgemma-2b',
      name: 'RecurrentGemma 2B',
      description: 'Google RecurrentGemma - Novel recurrent architecture (GrGrU) for efficient long sequences',
      type: 'gemma',
      size: '2.7GB',
      parameters: '2.7B',
      wasmPath: '/models/recurrentgemma-2b.wasm',
      weightsPath: '/models/recurrentgemma-2b.safetensors',
      capabilities: ['text-generation', 'long-context', 'recurrent-processing', 'stateful-tasks'],
      license: 'Apache 2.0',
      source: 'huggingface',
      architecture: 'Recurrent (GrGrU)',
      contextLength: 8192,
      recommended: true
    });

    this.registerModel({
      id: 'tinyllama',
      name: 'TinyLlama',
      description: 'TinyLlama - Lightweight but capable model for constrained environments',
      type: 'llama',
      size: '1.1GB',
      parameters: '1.1B',
      wasmPath: '/models/tinyllama.wasm',
      weightsPath: '/models/tinyllama.safetensors',
      capabilities: ['text-generation', 'chat', 'lightweight-inference'],
      license: 'Apache 2.0',
      source: 'huggingface',
      architecture: 'Llama',
      contextLength: 2048,
      recommended: false
    });

    // Initialize all model statuses
    this.availableModels.forEach((model, id) => {
      this.modelStatuses.set(id, {
        id,
        loaded: false,
        initialized: false,
        size: 0
      });
    });
  }

  /**
   * Register a new model definition
   */
  registerModel(model: ModelDefinition): void {
    this.availableModels.set(model.id, model);
    this.modelStatuses.set(model.id, {
      id: model.id,
      loaded: false,
      initialized: false,
      size: 0
    });
    
    this.emit('model_registered', { model });
  }

  /**
   * Get all available models
   */
  getAvailableModels(): ModelDefinition[] {
    return Array.from(this.availableModels.values());
  }

  /**
   * Get recommended models
   */
  getRecommendedModels(): ModelDefinition[] {
    return this.getAvailableModels().filter(model => model.recommended);
  }

  /**
   * Get models by type
   */
  getModelsByType(type: ModelDefinition['type']): ModelDefinition[] {
    return this.getAvailableModels().filter(model => model.type === type);
  }

  /**
   * Get models by source
   */
  getModelsBySource(source: ModelDefinition['source']): ModelDefinition[] {
    return this.getAvailableModels().filter(model => model.source === source);
  }

  /**
   * Get model definition by ID
   */
  getModel(id: string): ModelDefinition | null {
    return this.availableModels.get(id) || null;
  }

  /**
   * Get model status
   */
  getModelStatus(id: string): ModelStatus | null {
    return this.modelStatuses.get(id) || null;
  }

  /**
   * Get all model statuses
   */
  getAllModelStatuses(): ModelStatus[] {
    return Array.from(this.modelStatuses.values());
  }

  /**
   * Update model status
   */
  updateModelStatus(id: string, updates: Partial<ModelStatus>): void {
    const currentStatus = this.modelStatuses.get(id);
    if (currentStatus) {
      const newStatus = { ...currentStatus, ...updates };
      this.modelStatuses.set(id, newStatus);
      this.emit('model_status_updated', { id, status: newStatus });
    }
  }

  /**
   * Set current active model
   */
  setCurrentModel(id: string): boolean {
    const model = this.availableModels.get(id);
    if (!model) {
      return false;
    }

    const previousModel = this.currentModel;
    this.currentModel = id;
    
    this.emit('current_model_changed', { 
      previousModel, 
      currentModel: id, 
      model 
    });
    
    return true;
  }

  /**
   * Get current active model
   */
  getCurrentModel(): ModelDefinition | null {
    return this.currentModel ? this.availableModels.get(this.currentModel) || null : null;
  }

  /**
   * Get current model ID
   */
  getCurrentModelId(): string | null {
    return this.currentModel;
  }

  /**
   * Check if a model is available for loading
   */
  async isModelAvailable(id: string): Promise<boolean> {
    const model = this.availableModels.get(id);
    if (!model) {
      return false;
    }

    try {
      // Check if WASM file exists
      const wasmResponse = await fetch(model.wasmPath, { method: 'HEAD' });
      if (!wasmResponse.ok) {
        return false;
      }

      // Check if weights file exists (if specified)
      if (model.weightsPath) {
        const weightsResponse = await fetch(model.weightsPath, { method: 'HEAD' });
        if (!weightsResponse.ok) {
          return false;
        }
      }

      return true;
    } catch {
      return false;
    }
  }

  /**
   * Get model download/conversion instructions
   */
  getModelInstructions(id: string): string | null {
    const model = this.availableModels.get(id);
    if (!model) {
      return null;
    }

    switch (model.source) {
      case 'builtin':
        return 'This model should be included with the KNIRV installation.';
      
      case 'huggingface':
        return this.getHuggingFaceInstructions(model);
      
      case 'custom':
        return 'This is a custom model. Please ensure the WASM and weights files are available.';
      
      default:
        return 'No instructions available for this model.';
    }
  }

  private getHuggingFaceInstructions(model: ModelDefinition): string {
    const repoId = this.getHuggingFaceRepoId(model);
    
    return `To use this model, download and convert it using:

1. Install dependencies:
   pip install transformers safetensors huggingface_hub

2. Download and convert the model:
   python convert_checkpoint_to_safetensors.py \\
     --repo-id="${repoId}" \\
     --output="${model.weightsPath?.split('/').pop()}"

3. Compile to WASM (requires additional toolchain setup):
   # Follow WASM compilation instructions for ${model.architecture}

4. Place files in the models directory:
   - ${model.wasmPath}
   - ${model.weightsPath}`;
  }

  private getHuggingFaceRepoId(model: ModelDefinition): string {
    switch (model.id) {
      case 'phi3-mini':
        return 'microsoft/phi-3-mini-4k-instruct';
      case 'recurrentgemma-2b':
        return 'google/recurrentgemma-2b';
      case 'tinyllama':
        return 'TinyLlama/TinyLlama-1.1B-Chat-v1.0';
      default:
        return 'unknown/model';
    }
  }

  /**
   * Get model comparison data
   */
  getModelComparison(): Array<{
    id: string;
    name: string;
    parameters: string;
    size: string;
    architecture: string;
    contextLength: number;
    capabilities: string[];
    recommended: boolean;
    available: boolean;
  }> {
    return this.getAvailableModels().map(model => {
      const status = this.getModelStatus(model.id);
      return {
        id: model.id,
        name: model.name,
        parameters: model.parameters,
        size: model.size,
        architecture: model.architecture,
        contextLength: model.contextLength,
        capabilities: model.capabilities,
        recommended: model.recommended,
        available: status?.loaded || false
      };
    });
  }

  /**
   * Get default model recommendation
   */
  getDefaultModel(): ModelDefinition {
    // Prefer HRM Cognitive as default
    const hrmModel = this.availableModels.get('hrm-cognitive');
    if (hrmModel) {
      return hrmModel;
    }

    // Fallback to first recommended model
    const recommended = this.getRecommendedModels();
    if (recommended.length > 0) {
      return recommended[0];
    }

    // Fallback to first available model
    const available = this.getAvailableModels();
    if (available.length > 0) {
      return available[0];
    }

    throw new Error('No models available');
  }

  /**
   * Search models by capability
   */
  searchByCapability(capability: string): ModelDefinition[] {
    return this.getAvailableModels().filter(model => 
      model.capabilities.some(cap => 
        cap.toLowerCase().includes(capability.toLowerCase())
      )
    );
  }

  /**
   * Get models suitable for resource constraints
   */
  getModelsForConstraints(maxParameters: string, maxSize: string): ModelDefinition[] {
    // Simple parameter comparison (would need more sophisticated parsing in production)
    const maxParamNum = parseFloat(maxParameters);
    const maxSizeNum = parseFloat(maxSize);

    return this.getAvailableModels().filter(model => {
      const modelParamNum = parseFloat(model.parameters);
      const modelSizeNum = parseFloat(model.size);
      
      return modelParamNum <= maxParamNum && modelSizeNum <= maxSizeNum;
    });
  }

  /**
   * Export model configuration
   */
  exportConfiguration(): unknown {
    return {
      availableModels: Array.from(this.availableModels.values()),
      currentModel: this.currentModel,
      modelStatuses: Array.from(this.modelStatuses.values()),
      timestamp: new Date().toISOString()
    };
  }

  /**
   * Import model configuration
   */
  importConfiguration(config: unknown): void {
    if ((config as { availableModels?: unknown[]; currentModel?: string }).availableModels) {
      for (const model of (config as { availableModels: unknown[] }).availableModels) {
        if (this.isValidModelDefinition(model)) {
          this.registerModel(model);
        } else {
          console.warn('Invalid model definition in configuration:', model);
        }
      }
    }

    if ((config as { availableModels?: unknown[]; currentModel?: string }).currentModel) {
      this.setCurrentModel((config as { currentModel: string }).currentModel);
    }

    this.emit('configuration_imported', { config });
  }

  /**
   * Type guard for ModelDefinition
   */
  private isValidModelDefinition(model: unknown): model is ModelDefinition {
    return (
      typeof model === 'object' &&
      model !== null &&
      'id' in model &&
      'name' in model &&
      'description' in model &&
      'type' in model &&
      'size' in model &&
      'parameters' in model &&
      'wasmPath' in model &&
      'capabilities' in model &&
      'license' in model &&
      'source' in model &&
      'architecture' in model &&
      'contextLength' in model &&
      'recommended' in model &&
      typeof (model as Record<string, unknown>).id === 'string' &&
      typeof (model as Record<string, unknown>).name === 'string' &&
      Array.isArray((model as Record<string, unknown>).capabilities)
    );
  }

  /**
   * Check if the model manager is initialized
   */
  isInitialized(): boolean {
    return this.availableModels.size > 0;
  }
}

export default ModelManager;
