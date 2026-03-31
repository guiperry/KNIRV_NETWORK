import * as tf from '@tensorflow/tfjs';
import { EventEmitter } from './EventEmitter';

export interface HRMLoRAMapping {
  hrmLayerName: string;
  loraModuleName: string;
  weightMapping: 'direct' | 'projection' | 'attention';
  adaptationStrength: number;
}

export interface WeightSyncConfig {
  syncFrequency: number; // milliseconds
  adaptationThreshold: number;
  maxWeightChange: number;
  enableBidirectional: boolean;
}

export interface HRMWeightInfo {
  layerName: string;
  shape: number[];
  dtype: string;
  parameterCount: number;
}

export interface HRMProcessingResult {
  l_module_activations?: number[];
  h_module_activations?: number[];
  [key: string]: unknown;
}

export class HRMLoRABridge extends EventEmitter {
  private hrmBridge: unknown = null;
  private enhancedLoraAdapter: unknown = null;
  private mappings: Map<string, HRMLoRAMapping> = new Map();
  private syncConfig: WeightSyncConfig;
  private isRunning: boolean = false;
  private syncInterval: NodeJS.Timeout | number | null = null;
  private hrmWeightCache: Map<string, tf.Tensor> = new Map();

  constructor(syncConfig?: Partial<WeightSyncConfig>) {
    super();
    
    this.syncConfig = {
      syncFrequency: 5000, // 5 seconds
      adaptationThreshold: 0.1,
      maxWeightChange: 0.5,
      enableBidirectional: true,
      ...syncConfig,
    };
  }

  public setHRMBridge(hrmBridge: unknown): void {
    this.hrmBridge = hrmBridge;
    console.log('HRM bridge connected to HRM-LoRA Bridge');
  }

  public setEnhancedLoRAAdapter(enhancedLoraAdapter: unknown): void {
    this.enhancedLoraAdapter = enhancedLoraAdapter;
    console.log('Enhanced LoRA adapter connected to HRM-LoRA Bridge');
  }

  public addMapping(mapping: HRMLoRAMapping): void {
    this.mappings.set(mapping.loraModuleName, mapping);
    console.log(`Added HRM-LoRA mapping: ${mapping.hrmLayerName} -> ${mapping.loraModuleName}`);
    this.emit('mappingAdded', mapping);
  }

  public removeMapping(loraModuleName: string): void {
    if (this.mappings.delete(loraModuleName)) {
      console.log(`Removed HRM-LoRA mapping for ${loraModuleName}`);
      this.emit('mappingRemoved', loraModuleName);
    }
  }

  public async start(): Promise<void> {
    if (!this.hrmBridge || !this.enhancedLoraAdapter) {
      throw new Error('Both HRM bridge and Enhanced LoRA adapter must be connected');
    }

    console.log('Starting HRM-LoRA Bridge...');

    try {
      // Initialize weight mappings
      await this.initializeWeightMappings();

      // Start synchronization loop
      this.startSyncLoop();

      this.isRunning = true;
      this.emit('bridgeStarted');
      console.log('HRM-LoRA Bridge started successfully');

    } catch (error) {
      console.error('Failed to start HRM-LoRA Bridge:', error);
      throw error;
    }
  }

  public async stop(): Promise<void> {
    console.log('Stopping HRM-LoRA Bridge...');

    this.isRunning = false;

    if (this.syncInterval) {
      clearInterval(this.syncInterval);
      this.syncInterval = null;
    }

    // Dispose cached tensors
    this.disposeCachedWeights();

    this.emit('bridgeStopped');
    console.log('HRM-LoRA Bridge stopped');
  }

  private async initializeWeightMappings(): Promise<void> {
    console.log('Initializing HRM-LoRA weight mappings...');

    // Get HRM model information
    const hrmModelInfo = (this.hrmBridge as { getModelInfo?: () => unknown }).getModelInfo?.();
    if (!hrmModelInfo) {
      console.warn('HRM model info not available, using default mappings');
      this.createDefaultMappings();
      return;
    }

    // Create intelligent mappings based on HRM model structure
    await this.createIntelligentMappings(hrmModelInfo);
  }

  private createDefaultMappings(): void {
    // Create default mappings for common layer types
    const defaultMappings: HRMLoRAMapping[] = [
      {
        hrmLayerName: 'l_module_0',
        loraModuleName: 'base_hidden_1',
        weightMapping: 'projection',
        adaptationStrength: 0.3,
      },
      {
        hrmLayerName: 'h_module_0',
        loraModuleName: 'base_hidden_2',
        weightMapping: 'attention',
        adaptationStrength: 0.4,
      },
      {
        hrmLayerName: 'h_module_1',
        loraModuleName: 'base_output',
        weightMapping: 'direct',
        adaptationStrength: 0.2,
      },
    ];

    for (const mapping of defaultMappings) {
      this.addMapping(mapping);
    }
  }

  private async createIntelligentMappings(hrmModelInfo: unknown): Promise<void> {
    // Analyze HRM model structure and create optimal mappings
    console.log('Creating intelligent HRM-LoRA mappings based on model structure...');

    // Map L-modules (sensory-motor) to early LoRA layers
    for (let i = 0; i < Math.min((hrmModelInfo as { l_modules?: number }).l_modules || 0, 2); i++) {
      this.addMapping({
        hrmLayerName: `l_module_${i}`,
        loraModuleName: i === 0 ? 'base_hidden_1' : 'base_hidden_2',
        weightMapping: 'projection',
        adaptationStrength: 0.3 + (i * 0.1),
      });
    }

    // Map H-modules (planning) to later LoRA layers
    for (let i = 0; i < Math.min((hrmModelInfo as { h_modules?: number }).h_modules || 0, 2); i++) {
      this.addMapping({
        hrmLayerName: `h_module_${i}`,
        loraModuleName: i === 0 ? 'base_hidden_2' : 'base_output',
        weightMapping: 'attention',
        adaptationStrength: 0.4 + (i * 0.1),
      });
    }
  }

  private startSyncLoop(): void {
    this.syncInterval = setInterval(async () => {
      if (this.isRunning) {
        await this.performWeightSync();
      }
    }, this.syncConfig.syncFrequency);
  }

  private async performWeightSync(): Promise<void> {
    try {
      // Get current HRM activations and weights
      const hrmModelInfo = (this.hrmBridge as { getModelInfo?: () => unknown }).getModelInfo?.();
      if (!hrmModelInfo) return;

      // Sync weights for each mapping
      this.mappings.forEach(async (mapping) => {
        await this.syncWeightsForMapping(mapping);
      });

      this.emit('weightsSynced', {
        timestamp: new Date(),
        mappingsCount: this.mappings.size,
      });

    } catch (error) {
      console.error('Error during weight synchronization:', error);
      this.emit('syncError', error);
    }
  }

  private async syncWeightsForMapping(mapping: HRMLoRAMapping): Promise<void> {
    try {
      // Get HRM weight influence based on recent activations
      const hrmInfluence = await this.getHRMInfluence(mapping.hrmLayerName);
      
      if (Math.abs(hrmInfluence) < this.syncConfig.adaptationThreshold) {
        return; // Skip if influence is too small
      }

      // Get current LoRA weights
      const loraWeights = (this.enhancedLoraAdapter as { exportWeights?: () => Record<string, unknown> }).exportWeights?.() || {};
      const moduleWeights = loraWeights[mapping.loraModuleName];
      
      if (!moduleWeights) return;

      // Apply HRM-guided adaptation
      const adaptedWeights = await this.applyHRMGuidedAdaptation(
        moduleWeights,
        hrmInfluence,
        mapping
      );

      // Update LoRA weights
      const updatedWeights = { ...loraWeights };
      updatedWeights[mapping.loraModuleName] = adaptedWeights;
      
      await (this.enhancedLoraAdapter as { importWeights?: (weights: Record<string, unknown>) => Promise<void> }).importWeights?.(updatedWeights);

      this.emit('mappingSynced', {
        mapping,
        hrmInfluence,
        timestamp: new Date(),
      });

    } catch (error) {
      console.error(`Error syncing weights for mapping ${mapping.loraModuleName}:`, error);
    }
  }

  private async getHRMInfluence(hrmLayerName: string): Promise<number> {
    // This would ideally get the actual HRM layer activations
    // For now, we'll simulate based on recent HRM processing
    
    try {
      // Get recent HRM processing results
      const recentProcessing = await this.getRecentHRMProcessing();
      
      if (!recentProcessing) return 0;

      // Extract influence based on layer type
      if (hrmLayerName.startsWith('l_module_')) {
        const index = parseInt(hrmLayerName.split('_')[2]);
        const activations = recentProcessing.l_module_activations || [];
        return activations[index] || 0;
      }

      if (hrmLayerName.startsWith('h_module_')) {
        const index = parseInt(hrmLayerName.split('_')[2]);
        const activations = recentProcessing.h_module_activations || [];
        return activations[index] || 0;
      }

      return 0;

    } catch (error) {
      console.error('Error getting HRM influence:', error);
      return 0;
    }
  }

  private async getRecentHRMProcessing(): Promise<HRMProcessingResult | null> {
    // This would get the most recent HRM processing results
    // For now, we'll simulate a simple request
    
    try {
      const dummyInput = {
        sensory_data: new Array(512).fill(0.5),
        context: JSON.stringify({ sync: true }),
        task_type: 'weight_sync',
      };

      return await (this.hrmBridge as { processCognitiveInput?: (input: unknown) => Promise<unknown> }).processCognitiveInput?.(dummyInput) as HRMProcessingResult | null;

    } catch (error) {
      console.error('Error getting recent HRM processing:', error);
      return null;
    }
  }

  private async applyHRMGuidedAdaptation(
    moduleWeights: unknown,
    hrmInfluence: number,
    mapping: HRMLoRAMapping
  ): Promise<unknown> {
    // Apply HRM influence to LoRA weights based on mapping type
    const adaptationStrength = mapping.adaptationStrength * hrmInfluence;
    
    // Clamp adaptation to prevent instability
    const clampedAdaptation = Math.max(
      -this.syncConfig.maxWeightChange,
      Math.min(this.syncConfig.maxWeightChange, adaptationStrength)
    );

    switch (mapping.weightMapping) {
      case 'direct':
        return this.applyDirectAdaptation(moduleWeights, clampedAdaptation);
      
      case 'projection':
        return this.applyProjectionAdaptation(moduleWeights, clampedAdaptation);
      
      case 'attention':
        return this.applyAttentionAdaptation(moduleWeights, clampedAdaptation);
      
      default:
        return moduleWeights;
    }
  }

  private applyDirectAdaptation(weights: unknown, adaptation: number): unknown {
    // Direct scaling of weights
    const weightsAny = weights as Record<string, unknown>;
    return {
      ...(typeof weights === 'object' && weights !== null ? weights as Record<string, unknown> : {}),
      A: (weightsAny.A as number[][]).map((row: number[]) =>
        row.map((val: number) => val * (1 + adaptation))
      ),
      B: (weightsAny.B as number[][]).map((row: number[]) =>
        row.map((val: number) => val * (1 + adaptation))
      ),
    };
  }

  private applyProjectionAdaptation(weights: unknown, adaptation: number): unknown {
    // Apply adaptation through projection matrix
    const weightsAny = weights as Record<string, unknown>;
    return {
      ...(typeof weights === 'object' && weights !== null ? weights as Record<string, unknown> : {}),
      A: (weightsAny.A as number[][]).map((row: number[], i: number) =>
        row.map((val: number, j: number) =>
          val + (adaptation * Math.sin(i + j) * 0.1)
        )
      ),
      scaling: (weightsAny.scaling as number) * (1 + adaptation * 0.1),
    };
  }

  private applyAttentionAdaptation(weights: unknown, adaptation: number): unknown {
    // Apply attention-based adaptation
    const attentionFactor = Math.tanh(adaptation);
    
    const weightsAny = weights as Record<string, unknown>;
    return {
      ...(typeof weights === 'object' && weights !== null ? weights as Record<string, unknown> : {}),
      B: (weightsAny.B as number[][]).map((row: number[], i: number) =>
        row.map((val: number, j: number) => {
          const attention = Math.exp(-(i - j) * (i - j) / (2 * 4)); // Gaussian attention
          return val + (attentionFactor * attention * 0.05);
        })
      ),
    };
  }

  private disposeCachedWeights(): void {
    this.hrmWeightCache.forEach((tensor) => {
      tensor.dispose();
    });
    this.hrmWeightCache.clear();
  }

  public getMappings(): Map<string, HRMLoRAMapping> {
    return new Map(this.mappings);
  }

  public getSyncConfig(): WeightSyncConfig {
    return { ...this.syncConfig };
  }

  public updateSyncConfig(newConfig: Partial<WeightSyncConfig>): void {
    this.syncConfig = { ...this.syncConfig, ...newConfig };
    
    // Restart sync loop if frequency changed
    if (newConfig.syncFrequency && this.syncInterval) {
      clearInterval(this.syncInterval);
      this.startSyncLoop();
    }
    
    this.emit('syncConfigUpdated', this.syncConfig);
  }

  public async forceSyncNow(): Promise<void> {
    if (!this.isRunning) {
      throw new Error('Bridge not running');
    }

    console.log('Forcing immediate weight synchronization...');
    await this.performWeightSync();
    console.log('Forced synchronization complete');
  }

  public getStatus(): unknown {
    return {
      isRunning: this.isRunning,
      mappingsCount: this.mappings.size,
      hrmBridgeReady: this.hrmBridge ? (this.hrmBridge as { isReady?: () => boolean }).isReady?.() || false : false,
      loraAdapterReady: this.enhancedLoraAdapter ? (this.enhancedLoraAdapter as { isAdapterReady?: () => boolean }).isAdapterReady?.() || false : false,
      syncConfig: this.syncConfig,
      cachedWeights: this.hrmWeightCache.size,
    };
  }

  // Methods required by AdaptiveLearningPipeline interface
  public async processWithLoRA(data: unknown, loraConfig: Record<string, unknown>): Promise<unknown> {
    if (!this.hrmBridge || !this.enhancedLoraAdapter) {
      throw new Error('HRM Bridge or Enhanced LoRA Adapter not initialized');
    }

    try {
      // Process with HRM first
      const hrmResult = await (this.hrmBridge as { process?: (data: unknown) => Promise<unknown> }).process?.(data);

      // Apply LoRA adaptation
      const adaptedResult = await (this.enhancedLoraAdapter as { adapt?: (result: unknown, config: unknown) => Promise<unknown> }).adapt?.(hrmResult, loraConfig);

      return adaptedResult;
    } catch (error) {
      console.error('Error in processWithLoRA:', error);
      throw error;
    }
  }

  public updateLoRAWeights(weights: Record<string, unknown>): void {
    if (!this.enhancedLoraAdapter) {
      console.warn('Enhanced LoRA Adapter not initialized');
      return;
    }

    try {
      // Update the cached weights
      for (const [key, value] of Object.entries(weights)) {
        // Convert value to tensor if it's not already one
        if (value && typeof value === 'object' && 'data' in value) {
          // Assume it's tensor-like data
          const tensor = tf.tensor(value as number[] | number[][] | number[][][] | number[][][][]);
          this.hrmWeightCache.set(key, tensor);
        } else if (Array.isArray(value)) {
          // Convert array to tensor
          const tensor = tf.tensor(value);
          this.hrmWeightCache.set(key, tensor);
        }
        // Skip non-tensor values
      }

      // Trigger weight synchronization
      this.performWeightSync().catch(error => {
        console.error('Error updating LoRA weights:', error);
      });
    } catch (error) {
      console.error('Error updating LoRA weights:', error);
    }
  }


}
