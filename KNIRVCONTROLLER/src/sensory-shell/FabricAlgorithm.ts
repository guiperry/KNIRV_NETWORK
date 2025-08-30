import { EventEmitter } from './EventEmitter';

export interface FabricConfig {
  contextSize: number;
  processingMode: 'adaptive' | 'static' | 'dynamic';
  memoryDepth: number;
  attentionHeads: number;
  learningRate: number;
  hrmIntegration?: boolean;
}

export interface FabricContext {
  inputHistory: unknown[];
  outputHistory: unknown[];
  attentionWeights: Map<string, number>;
  memoryState: unknown;
  processingMetrics: ProcessingMetrics;
}

export interface ProcessingMetrics {
  totalProcessed: number;
  averageLatency: number;
  accuracyScore: number;
  adaptationCount: number;
  lastProcessed: Date;
}

export interface AttentionMechanism {
  weights: Map<string, number>;
  focusAreas: string[];
  contextRelevance: number;
}

export class FabricAlgorithm extends EventEmitter {
  private config: FabricConfig;
  private context: FabricContext;
  private attentionMechanism: AttentionMechanism;
  private isRunning: boolean = false;
  private processingQueue: unknown[] = [];
  private hrmBridge: unknown = null; // Will be injected from CognitiveEngine

  constructor(config: FabricConfig) {
    super();
    this.config = config;
    this.initializeContext();
    this.initializeAttentionMechanism();
  }

  private initializeContext(): void {
    this.context = {
      inputHistory: [],
      outputHistory: [],
      attentionWeights: new Map(),
      memoryState: {},
      processingMetrics: {
        totalProcessed: 0,
        averageLatency: 0,
        accuracyScore: 0.5,
        adaptationCount: 0,
        lastProcessed: new Date(),
      },
    };
  }

  private initializeAttentionMechanism(): void {
    this.attentionMechanism = {
      weights: new Map(),
      focusAreas: [],
      contextRelevance: 0.5,
    };
  }

  public async start(): Promise<void> {
    console.log('Starting Fabric Algorithm...');
    this.isRunning = true;
    this.startProcessingLoop();
    this.emit('fabricStarted');
  }

  public async stop(): Promise<void> {
    console.log('Stopping Fabric Algorithm...');
    this.isRunning = false;
    this.emit('fabricStopped');
  }

  public async process(input: unknown, options: unknown = {}): Promise<any> {
    const startTime = Date.now();

    try {
      // Use HRM-enhanced processing if available
      if (this.config.hrmIntegration && this.hrmBridge && this.hrmBridge.isReady()) {
        return await this.hrmEnhancedProcess(input, options);
      }

      // Fallback to traditional processing
      if (this.config.processingMode === 'adaptive') {
        return await this.adaptiveProcess(input, options);
      } else {
        return await this.directProcess(input, options);
      }

    } catch (error) {
      console.error('Fabric processing error:', error);
      throw error;
    } finally {
      const latency = Date.now() - startTime;
      this.updateMetrics(latency);
    }
  }

  private async hrmEnhancedProcess(input: unknown, options: unknown): Promise<any> {
    console.log('Processing with HRM-enhanced Fabric Algorithm...');

    try {
      // Prepare input for HRM cognitive processing
      const hrmInput = {
        sensory_data: this.convertToSensoryData(input),
        context: JSON.stringify({
          fabricContext: this.context,
          options: options,
          attentionWeights: Object.fromEntries(this.attentionMechanism.weights),
        }),
        task_type: this.determineTaskType(input, options),
      };

      // Get HRM cognitive analysis
      const hrmOutput = await this.hrmBridge.processCognitiveInput(hrmInput);

      // Use HRM insights to enhance traditional Fabric processing
      const enhancedOptions = {
        ...options,
        hrmGuidance: {
          reasoning: hrmOutput.reasoning_result,
          confidence: hrmOutput.confidence,
          l_activations: hrmOutput.l_module_activations,
          h_activations: hrmOutput.h_module_activations,
        },
      };

      // Determine processing strategy based on HRM analysis
      const processingStrategy = this.selectStrategyWithHRM(input, hrmOutput);

      // Apply HRM-guided attention mechanism
      const attentionResult = await this.applyHRMGuidedAttention(input, options.context, hrmOutput);

      // Execute enhanced processing
      const result = await this.executeHRMEnhancedStrategy(
        processingStrategy,
        attentionResult,
        enhancedOptions
      );

      // Generate NRV (Neural Reasoning Vector) with HRM insights
      const nrv = this.generateNRVWithHRM(result, hrmOutput);

      // Update context with HRM insights
      this.updateContextWithHRM(input, result, enhancedOptions, hrmOutput);

      return {
        ...result,
        nrv: nrv,
        hrmEnhanced: true,
        hrmConfidence: hrmOutput.confidence,
        hrmReasoning: hrmOutput.reasoning_result,
        processingStrategy: processingStrategy,
      };

    } catch (error) {
      console.error('Error in HRM-enhanced processing:', error);
      // Fallback to adaptive processing
      return this.adaptiveProcess(input, options);
    }
  }

  private convertToSensoryData(input: unknown): number[] {
    // Convert input to numerical representation for HRM processing
    if (typeof input === 'string') {
      const encoder = new TextEncoder();
      const bytes = encoder.encode(input);
      return Array.from(bytes).map(b => b / 255.0).slice(0, 512);
    }

    if (Array.isArray(input)) {
      return input.slice(0, 512);
    }

    if (typeof input === 'object') {
      const str = JSON.stringify(input);
      const encoder = new TextEncoder();
      const bytes = encoder.encode(str);
      return Array.from(bytes).map(b => b / 255.0).slice(0, 512);
    }

    return new Array(512).fill(0);
  }

  private determineTaskType(input: unknown, options: unknown): string {
    if (options.inputType) {
      return `fabric_${options.inputType}`;
    }

    if (typeof input === 'object' && input.type) {
      return `fabric_${input.type}`;
    }

    return 'fabric_general';
  }

  private selectStrategyWithHRM(input: unknown, hrmOutput: unknown): string {
    // Use HRM confidence and activations to select processing strategy
    const confidence = hrmOutput.confidence;
    const avgHActivation = hrmOutput.h_module_activations.reduce((a: number, b: number) => a + b, 0) / hrmOutput.h_module_activations.length;

    if (confidence > 0.8 && avgHActivation > 0.7) {
      return 'hrm_deep_analysis';
    } else if (confidence > 0.6 && avgHActivation > 0.5) {
      return 'hrm_standard_processing';
    } else {
      return 'hrm_fast_processing';
    }
  }

  private async applyHRMGuidedAttention(input: unknown, context: unknown, hrmOutput: unknown): Promise<any> {
    // Traditional attention mechanism
    const traditionalAttention = await this.applyAttention(input, context);

    // Enhance with HRM module activations
    const hrmGuidedWeights = new Map();

    // Use L-module activations to guide sensory attention
    if (hrmOutput.l_module_activations) {
      hrmOutput.l_module_activations.forEach((activation: number, index: number) => {
        hrmGuidedWeights.set(`l_module_${index}`, activation);
      });
    }

    // Use H-module activations to guide planning attention
    if (hrmOutput.h_module_activations) {
      hrmOutput.h_module_activations.forEach((activation: number, index: number) => {
        hrmGuidedWeights.set(`h_module_${index}`, activation);
      });
    }

    return {
      ...traditionalAttention,
      hrmGuidedWeights: hrmGuidedWeights,
      hrmConfidence: hrmOutput.confidence,
      combinedFocusAreas: [
        ...traditionalAttention.focusAreas,
        ...Array.from(hrmGuidedWeights.keys()).filter(key => hrmGuidedWeights.get(key) > 0.7),
      ],
    };
  }

  private generateNRVWithHRM(result: unknown, hrmOutput: unknown): unknown {
    // Generate Neural Reasoning Vector combining Fabric and HRM insights
    return {
      fabricVector: this.generateTraditionalNRV(result),
      hrmVector: {
        l_activations: hrmOutput.l_module_activations,
        h_activations: hrmOutput.h_module_activations,
        reasoning_confidence: hrmOutput.confidence,
        processing_time: hrmOutput.processing_time,
      },
      combinedConfidence: (result.confidence + hrmOutput.confidence) / 2,
      timestamp: new Date().toISOString(),
      version: '1.0.0-hrm',
    };
  }

  private generateTraditionalNRV(result: unknown): unknown {
    // Traditional NRV generation (existing logic)
    return {
      confidence: result.confidence || 0.5,
      complexity: result.complexity || 0.5,
      attentionWeights: Object.fromEntries(this.attentionMechanism.weights),
      contextRelevance: this.attentionMechanism.contextRelevance,
    };
  }

  private updateContextWithHRM(input: unknown, result: unknown, options: unknown, hrmOutput: unknown): void {
    // Traditional context update
    this.updateContext(input, result, options);

    // Add HRM-specific context
    this.context.memoryState.hrmHistory = this.context.memoryState.hrmHistory || [];
    this.context.memoryState.hrmHistory.push({
      timestamp: new Date(),
      input: input,
      hrmOutput: hrmOutput,
      confidence: hrmOutput.confidence,
    });

    // Keep only recent HRM history
    if (this.context.memoryState.hrmHistory.length > 10) {
      this.context.memoryState.hrmHistory = this.context.memoryState.hrmHistory.slice(-10);
    }
  }

  private async executeHRMEnhancedStrategy(
    strategy: string,
    attentionResult: unknown,
    options: unknown
  ): Promise<any> {
    console.log(`Executing HRM-enhanced strategy: ${strategy}`);

    const hrmGuidance = options.hrmGuidance;

    switch (strategy) {
      case 'hrm_deep_analysis':
        return await this.hrmDeepAnalysisProcessing(attentionResult, options, hrmGuidance);

      case 'hrm_standard_processing':
        return await this.hrmStandardProcessing(attentionResult, options, hrmGuidance);

      case 'hrm_fast_processing':
        return await this.hrmFastProcessing(attentionResult, options, hrmGuidance);

      default:
        return await this.standardProcessing(attentionResult, options);
    }
  }

  private async hrmDeepAnalysisProcessing(attentionResult: unknown, options: unknown, hrmGuidance: unknown): Promise<any> {
    // Deep analysis with HRM cognitive insights
    const result = await this.deepAnalysisProcessing(attentionResult, options);

    return {
      ...result,
      hrmEnhanced: true,
      hrmReasoning: hrmGuidance.reasoning,
      hrmConfidence: hrmGuidance.confidence,
      analysisDepth: 'deep_with_hrm',
      cognitiveInsights: {
        l_module_patterns: hrmGuidance.l_activations,
        h_module_planning: hrmGuidance.h_activations,
      },
    };
  }

  private async hrmStandardProcessing(attentionResult: unknown, options: unknown, hrmGuidance: unknown): Promise<any> {
    // Standard processing with HRM guidance
    const result = await this.standardProcessing(attentionResult, options);

    return {
      ...result,
      hrmEnhanced: true,
      hrmReasoning: hrmGuidance.reasoning,
      hrmConfidence: hrmGuidance.confidence,
      analysisDepth: 'standard_with_hrm',
    };
  }

  private async hrmFastProcessing(attentionResult: unknown, options: unknown, hrmGuidance: unknown): Promise<any> {
    // Fast processing with minimal HRM overhead
    const result = await this.fastProcessing(attentionResult, options);

    return {
      ...result,
      hrmEnhanced: true,
      hrmConfidence: hrmGuidance.confidence,
      analysisDepth: 'fast_with_hrm',
    };
  }

  private async adaptiveProcess(input: unknown, options: unknown): Promise<any> {
    // Analyze input complexity and context
    const complexity = this.analyzeComplexity(input);


    // Adjust processing strategy based on complexity
    let processingStrategy: string;
    if (complexity > 0.8) {
      processingStrategy = 'deep_analysis';
    } else if (complexity > 0.5) {
      processingStrategy = 'standard_processing';
    } else {
      processingStrategy = 'fast_processing';
    }

    // Apply attention mechanism
    const attentionResult = await this.applyAttention(input, options.context);

    // Process with selected strategy
    const result = await this.executeProcessingStrategy(
      processingStrategy,
      attentionResult,
      options
    );

    // Update context and memory
    this.updateContext(input, result, options);

    return result;
  }

  private async directProcess(input: unknown, options: unknown): Promise<any> {
    // Direct processing without adaptive mechanisms
    const result = await this.executeBasicProcessing(input, options);
    this.updateContext(input, result, options);
    return result;
  }

  private analyzeComplexity(input: unknown): number {
    let complexity = 0;

    // Analyze input structure
    if (typeof input === 'object') {
      complexity += Object.keys(input).length * 0.1;

      // Check for nested structures
      for (const value of Object.values(input)) {
        if (typeof value === 'object') {
          complexity += 0.2;
        }
      }
    }

    // Analyze input size
    const inputSize = JSON.stringify(input).length;
    complexity += Math.min(inputSize / 1000, 0.5);

    // Analyze input type
    if (Array.isArray(input)) {
      complexity += input.length * 0.05;
    }

    return Math.min(complexity, 1.0);
  }

  private calculateContextRelevance(input: unknown, context: unknown): number {
    if (!context) return 0.5;

    let relevance = 0;
    const inputStr = JSON.stringify(input).toLowerCase();

    // Check against recent inputs
    for (const historyItem of this.context.inputHistory.slice(-5)) {
      const historyStr = JSON.stringify(historyItem).toLowerCase();
      const similarity = this.calculateSimilarity(inputStr, historyStr);
      relevance += similarity * 0.2;
    }

    // Check against context data
    for (const [, value] of context) {
      const contextStr = JSON.stringify(value).toLowerCase();
      const similarity = this.calculateSimilarity(inputStr, contextStr);
      relevance += similarity * 0.1;
    }

    return Math.min(relevance, 1.0);
  }

  private calculateSimilarity(str1: string, str2: string): number {
    // Simple similarity calculation (could be improved with more sophisticated algorithms)
    const words1 = str1.split(/\s+/);
    const words2 = str2.split(/\s+/);

    const commonWords = words1.filter(word => words2.includes(word));
    const totalWords = new Set([...words1, ...words2]).size;

    return totalWords > 0 ? commonWords.length / totalWords : 0;
  }

  private async applyAttention(input: unknown, context: unknown): Promise<any> {
    // Update attention weights based on input and context
    this.updateAttentionWeights(input, context);

    // Apply attention to input
    const attentionResult = {
      focusedInput: this.applyAttentionToInput(input),
      attentionWeights: new Map(this.attentionMechanism.weights),
      focusAreas: [...this.attentionMechanism.focusAreas],
    };

    this.emit('attentionApplied', attentionResult);
    return attentionResult;
  }

  private updateAttentionWeights(input: unknown, context: unknown): void {
    // Clear old weights
    this.attentionMechanism.weights.clear();
    this.attentionMechanism.focusAreas = [];

    // Analyze input for attention targets
    if (typeof input === 'object') {
      for (const [key, value] of Object.entries(input)) {
        const weight = this.calculateAttentionWeight(key, value, context);
        this.attentionMechanism.weights.set(key, weight);

        if (weight > 0.7) {
          this.attentionMechanism.focusAreas.push(key);
        }
      }
    }

    // Update context relevance
    this.attentionMechanism.contextRelevance = this.calculateContextRelevance(input, context);
  }

  private calculateAttentionWeight(key: string, value: unknown, context: unknown): number {
    let weight = 0.5; // Base weight

    // Increase weight for certain key patterns
    const importantPatterns = ['error', 'skill', 'command', 'request', 'problem'];
    if (importantPatterns.some(pattern => key.toLowerCase().includes(pattern))) {
      weight += 0.3;
    }

    // Increase weight based on value complexity
    if (typeof value === 'object') {
      weight += 0.2;
    }

    // Increase weight if related to recent context
    if (context && context.has(key)) {
      weight += 0.2;
    }

    return Math.min(weight, 1.0);
  }

  private applyAttentionToInput(input: unknown): unknown {
    if (typeof input !== 'object') {
      return input;
    }

    const focusedInput: unknown = {};

    for (const [key, value] of Object.entries(input)) {
      const weight = this.attentionMechanism.weights.get(key) || 0.5;

      if (weight > 0.3) {
        focusedInput[key] = {
          value,
          attentionWeight: weight,
          isFocused: this.attentionMechanism.focusAreas.includes(key),
        };
      }
    }

    return focusedInput;
  }

  private async executeProcessingStrategy(
    strategy: string,
    attentionResult: unknown,
    options: unknown
  ): Promise<any> {
    console.log(`Executing processing strategy: ${strategy}`);

    switch (strategy) {
      case 'deep_analysis':
        return await this.deepAnalysisProcessing(attentionResult, options);

      case 'standard_processing':
        return await this.standardProcessing(attentionResult, options);

      case 'fast_processing':
        return await this.fastProcessing(attentionResult, options);

      default:
        return await this.standardProcessing(attentionResult, options);
    }
  }

  private async deepAnalysisProcessing(attentionResult: unknown, options: unknown): Promise<any> {
    // Simulate deep analysis with multiple passes
    const passes = 3;
    let result = attentionResult.focusedInput;

    for (let i = 0; i < passes; i++) {
      result = await this.processPass(result, `deep_pass_${i}`, options);

      // Add delay to simulate complex processing
      await new Promise(resolve => setTimeout(resolve, 200));
    }

    return {
      type: 'deep_analysis_result',
      result,
      strategy: 'deep_analysis',
      passes,
      confidence: 0.9,
      processingTime: Date.now(),
    };
  }

  private async standardProcessing(attentionResult: unknown, options: unknown): Promise<any> {
    const result = await this.processPass(attentionResult.focusedInput, 'standard', options);

    return {
      type: 'standard_result',
      result,
      strategy: 'standard_processing',
      confidence: 0.75,
      processingTime: Date.now(),
    };
  }

  private async fastProcessing(attentionResult: unknown, _options: unknown): Promise<any> {
    // Quick processing with minimal analysis
    const result = {
      processed: true,
      input: attentionResult.focusedInput,
      quickAnalysis: 'Fast processing applied',
    };

    return {
      type: 'fast_result',
      result,
      strategy: 'fast_processing',
      confidence: 0.6,
      processingTime: Date.now(),
    };
  }

  private async executeBasicProcessing(input: unknown, options: unknown): Promise<any> {
    const result = await this.processPass(input, 'basic', options);

    return {
      type: 'basic_result',
      result,
      strategy: 'basic_processing',
      confidence: 0.7,
      processingTime: Date.now(),
    };
  }

  private async processPass(input: unknown, passType: string, options: unknown): Promise<any> {
    // Simulate processing pass
    await new Promise(resolve => setTimeout(resolve, 100));

    return {
      passType,
      processedInput: input,
      metadata: {
        timestamp: new Date(),
        options,
      },
    };
  }

  private updateContext(input: unknown, result: unknown, options: unknown): void {
    // Add to input history
    this.context.inputHistory.push({
      input,
      timestamp: new Date(),
      options,
    });

    // Add to output history
    this.context.outputHistory.push({
      result,
      timestamp: new Date(),
    });

    // Maintain history size limits
    if (this.context.inputHistory.length > this.config.contextSize) {
      this.context.inputHistory.shift();
    }

    if (this.context.outputHistory.length > this.config.contextSize) {
      this.context.outputHistory.shift();
    }

    // Update memory state
    this.updateMemoryState(input, result);

    this.emit('contextUpdated', {
      inputHistorySize: this.context.inputHistory.length,
      outputHistorySize: this.context.outputHistory.length,
    });
  }

  private updateMemoryState(input: unknown, result: unknown): void {
    // Update memory with key patterns and relationships
    const inputKey = this.generateMemoryKey(input);
    const resultKey = this.generateMemoryKey(result);

    this.context.memoryState[inputKey] = {
      lastSeen: new Date(),
      frequency: (this.context.memoryState[inputKey]?.frequency || 0) + 1,
      associatedResults: [resultKey],
    };

    // Create associations
    if (this.context.memoryState[resultKey]) {
      this.context.memoryState[resultKey].associatedInputs =
        this.context.memoryState[resultKey].associatedInputs || [];
      this.context.memoryState[resultKey].associatedInputs.push(inputKey);
    }
  }

  private generateMemoryKey(data: unknown): string {
    // Generate a key for memory storage
    if (typeof data === 'string') {
      return data.substring(0, 50);
    }

    return JSON.stringify(data).substring(0, 50);
  }

  private updateMetrics(latency: number): void {
    const metrics = this.context.processingMetrics;

    metrics.totalProcessed++;
    metrics.averageLatency = ((metrics.averageLatency * (metrics.totalProcessed - 1)) + latency) / metrics.totalProcessed;
    metrics.lastProcessed = new Date();

    this.emit('metricsUpdated', metrics);
  }

  private startProcessingLoop(): void {
    // Background processing loop for queued items
    const processLoop = async () => {
      while (this.isRunning) {
        if (this.processingQueue.length > 0) {
          const item = this.processingQueue.shift();
          try {
            await this.process(item.input, item.options);
          } catch (error) {
            console.error('Background processing error:', error);
          }
        }

        await new Promise(resolve => setTimeout(resolve, 100));
      }
    };

    processLoop();
  }

  public queueForProcessing(input: unknown, options: unknown = {}): void {
    this.processingQueue.push({ input, options });
  }

  public getContext(): FabricContext {
    return { ...this.context };
  }

  public getAttentionState(): AttentionMechanism {
    return { ...this.attentionMechanism };
  }

  public getMetrics(): ProcessingMetrics {
    return { ...this.context.processingMetrics };
  }

  public clearContext(): void {
    this.initializeContext();
    this.emit('contextCleared');
  }

  public exportMemoryState(): unknown {
    return { ...this.context.memoryState };
  }

  public importMemoryState(memoryState: unknown): void {
    this.context.memoryState = { ...memoryState };
    this.emit('memoryStateImported');
  }

  // HRM Integration methods
  public setHRMBridge(hrmBridge: unknown): void {
    this.hrmBridge = hrmBridge;
    console.log('HRM bridge injected into Fabric Algorithm');
  }

  public enableHRMIntegration(): void {
    this.config.hrmIntegration = true;
    console.log('HRM integration enabled in Fabric Algorithm');
  }

  public disableHRMIntegration(): void {
    this.config.hrmIntegration = false;
    console.log('HRM integration disabled in Fabric Algorithm');
  }

  public isHRMIntegrationEnabled(): boolean {
    return this.config.hrmIntegration === true;
  }

  public getHRMStatus(): unknown {
    return {
      enabled: this.config.hrmIntegration,
      bridgeAvailable: this.hrmBridge !== null,
      ready: this.hrmBridge ? this.hrmBridge.isReady() : false,
    };
  }

  public getHRMHistory(): unknown[] {
    return this.context.memoryState.hrmHistory || [];
  }

  public clearHRMHistory(): void {
    if (this.context.memoryState.hrmHistory) {
      this.context.memoryState.hrmHistory = [];
      this.emit('hrmHistoryCleared');
    }
  }

  public getEnhancedMetrics(): unknown {
    const baseMetrics = this.getMetrics();
    const hrmHistory = this.getHRMHistory();

    return {
      ...baseMetrics,
      hrmIntegration: this.config.hrmIntegration,
      hrmProcessedCount: hrmHistory.length,
      averageHRMConfidence: hrmHistory.length > 0
        ? hrmHistory.reduce((sum, item) => sum + item.confidence, 0) / hrmHistory.length
        : 0,
      lastHRMProcessing: hrmHistory.length > 0
        ? hrmHistory[hrmHistory.length - 1].timestamp
        : null,
    };
  }
}
