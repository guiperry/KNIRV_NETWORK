import { EventEmitter } from './EventEmitter';

export interface FabricConfig {
  contextSize: number;
  processingMode: 'adaptive' | 'static' | 'dynamic';
  memoryDepth: number;
  attentionHeads: number;
  learningRate: number;
  headIntegration?: boolean;
}

export interface FabricContext {
  inputHistory: unknown[];
  outputHistory: unknown[];
  attentionWeights: Map<string, number>;
  memoryState: unknown;
  processingMetrics: ProcessingMetrics;
  currentState: string;
  activePatterns: unknown[];
  memoryBuffer: Map<string, unknown>;
  timestamp: number;
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
  focusWeights: Map<string, number>;
  attentionHistory: unknown[];
  currentFocus: unknown;
}

export class FabricAlgorithm extends EventEmitter {
  private config: FabricConfig;
  private context: FabricContext = {
    inputHistory: [],
    outputHistory: [],
    attentionWeights: new Map(),
    memoryState: {},
    processingMetrics: {
      totalProcessed: 0,
      averageLatency: 0,
      accuracyScore: 0,
      adaptationCount: 0,
      lastProcessed: new Date()
    },
    currentState: 'idle',
    activePatterns: [],
    memoryBuffer: new Map(),
    timestamp: Date.now()
  };
  private attentionMechanism: AttentionMechanism = {
    weights: new Map(),
    focusAreas: [],
    contextRelevance: 0,
    focusWeights: new Map(),
    attentionHistory: [],
    currentFocus: null
  };
  private isRunning: boolean = false;
  private processingQueue: unknown[] = [];
  private headBridge: unknown = null; // Will be injected from CognitiveEngine

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
      currentState: 'idle',
      activePatterns: [],
      memoryBuffer: new Map(),
      timestamp: Date.now()
    };
  }

  private initializeAttentionMechanism(): void {
    this.attentionMechanism = {
      weights: new Map(),
      focusAreas: [],
      contextRelevance: 0.5,
      focusWeights: new Map(),
      attentionHistory: [],
      currentFocus: null
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

  public async process(input: unknown, options: unknown = {}): Promise<unknown> {
    const startTime = Date.now();

    try {
      // Use HEAD-enhanced processing if available
      if (this.config.headIntegration && this.headBridge && (this.headBridge as { isReady?: () => boolean }).isReady?.()) {
        return await this.headEnhancedProcess(input, options);
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

  private async headEnhancedProcess(input: unknown, options: unknown): Promise<unknown> {
    console.log('Processing with HEAD-enhanced Fabric Algorithm...');

    try {
      // Prepare input for HEAD cognitive processing
      const headInput = {
        sensory_data: this.convertToSensoryData(input),
        context: JSON.stringify({
          fabricContext: this.context,
          options: options,
          attentionWeights: Object.fromEntries(this.attentionMechanism.weights),
        }),
        task_type: this.determineTaskType(input, options),
      };

      // Get HEAD cognitive analysis
      const headOutput = await (this.headBridge as { processCognitiveInput?: (input: unknown) => Promise<unknown> }).processCognitiveInput?.(headInput);

      // Use HEAD insights to enhance traditional Fabric processing
      const enhancedOptions = {
        ...(typeof options === 'object' && options !== null ? options as Record<string, unknown> : {}),
        headGuidance: {
          reasoning: (headOutput as { reasoning_result?: unknown }).reasoning_result,
          confidence: (headOutput as { confidence?: unknown }).confidence,
          l_activations: (headOutput as { l_module_activations?: unknown }).l_module_activations,
          h_activations: (headOutput as { h_module_activations?: unknown }).h_module_activations,
        },
      };

      // Determine processing strategy based on HEAD analysis
      const processingStrategy = this.selectStrategyWithHEAD(input, headOutput);

      // Apply HEAD-guided attention mechanism
      const attentionResult = await this.applyHEADGuidedAttention(input, (options as { context?: unknown }).context, headOutput);

      // Execute enhanced processing
      const result = await this.executeHEADEnhancedStrategy(
        processingStrategy,
        attentionResult,
        enhancedOptions
      );

      // Generate NRV (Neural Reasoning Vector) with HEAD insights
      const nrv = this.generateNRVWithHEAD(result, headOutput);

      // Update context with HEAD insights
      this.updateContextWithHEAD(input, result, enhancedOptions, headOutput);

      interface HEADOutput {
        confidence?: number;
        reasoning_result?: unknown;
      }

      const headTyped = headOutput as HEADOutput;

      return {
        ...(result as Record<string, unknown>),
        nrv: nrv,
        headEnhanced: true,
        headConfidence: headTyped.confidence,
        headReasoning: headTyped.reasoning_result,
        processingStrategy: processingStrategy,
      };

    } catch (error) {
      console.error('Error in HEAD-enhanced processing:', error);
      // Fallback to adaptive processing
      return this.adaptiveProcess(input, options);
    }
  }

  private convertToSensoryData(input: unknown): number[] {
    // Convert input to numerical representation for HEAD processing
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
    const optionsAny = options as { inputType?: string };
    if (optionsAny.inputType) {
      return `fabric_${optionsAny.inputType}`;
    }

    if (typeof input === 'object' && input !== null && 'type' in input) {
      return `fabric_${(input as { type?: string }).type}`;
    }

    return 'fabric_general';
  }

  private selectStrategyWithHEAD(input: unknown, headOutput: unknown): string {
    // Use HEAD confidence and activations to select processing strategy
    const headAny = headOutput as { confidence?: number; h_module_activations?: number[] };
    const confidence = headAny.confidence || 0;
    const hActivations = headAny.h_module_activations || [];
    const avgHActivation = hActivations.length > 0 ? hActivations.reduce((a: number, b: number) => a + b, 0) / hActivations.length : 0;

    if (confidence > 0.8 && avgHActivation > 0.7) {
      return 'head_deep_analysis';
    } else if (confidence > 0.6 && avgHActivation > 0.5) {
      return 'head_standard_processing';
    } else {
      return 'head_fast_processing';
    }
  }

  private async applyHEADGuidedAttention(input: unknown, context: unknown, headOutput: unknown): Promise<unknown> {
    // Traditional attention mechanism
    const traditionalAttention = await this.applyAttention(input, context);

    // Enhance with HEAD module activations
    const headGuidedWeights = new Map();

    // Use L-module activations to guide sensory attention
    const headAny = headOutput as { l_module_activations?: number[]; h_module_activations?: number[]; confidence?: number };
    if (headAny.l_module_activations) {
      headAny.l_module_activations.forEach((activation: number, index: number) => {
        headGuidedWeights.set(`l_module_${index}`, activation);
      });
    }

    // Use H-module activations to guide planning attention
    if (headAny.h_module_activations) {
      headAny.h_module_activations.forEach((activation: number, index: number) => {
        headGuidedWeights.set(`h_module_${index}`, activation);
      });
    }

    return {
      ...traditionalAttention,
      headGuidedWeights: headGuidedWeights,
      headConfidence: headAny.confidence,
      combinedFocusAreas: [
        ...traditionalAttention.focusAreas,
        ...Array.from(headGuidedWeights.keys()).filter(key => headGuidedWeights.get(key) > 0.7),
      ],
    };
  }

  private generateNRVWithHEAD(result: unknown, headOutput: unknown): unknown {
    // Generate Neural Reasoning Vector combining Fabric and HEAD insights
    return {
      fabricVector: this.generateTraditionalNRV(result),
      headVector: {
        l_activations: (headOutput as { l_module_activations?: unknown }).l_module_activations,
        h_activations: (headOutput as { h_module_activations?: unknown }).h_module_activations,
        reasoning_confidence: (headOutput as { confidence?: number }).confidence,
        processing_time: (headOutput as { processing_time?: number }).processing_time,
      },
      combinedConfidence: (((result as { confidence?: number }).confidence || 0) + ((headOutput as { confidence?: number }).confidence || 0)) / 2,
      timestamp: new Date().toISOString(),
      version: '1.0.0-head',
    };
  }

  private generateTraditionalNRV(result: unknown): unknown {
    // Traditional NRV generation (existing logic)
    return {
      confidence: (result as { confidence?: number }).confidence || 0.5,
      complexity: (result as { complexity?: number }).complexity || 0.5,
      attentionWeights: Object.fromEntries(this.attentionMechanism.weights),
      contextRelevance: this.attentionMechanism.contextRelevance,
    };
  }

  private updateContextWithHEAD(input: unknown, result: unknown, options: unknown, headOutput: unknown): void {
    // Traditional context update
    this.updateContext(input, result, options);

    // Add HEAD-specific context
    const contextAny = this.context as { memoryState?: { headHistory?: unknown[] } };
    if (!contextAny.memoryState) contextAny.memoryState = {};
    contextAny.memoryState.headHistory = contextAny.memoryState.headHistory || [];
    contextAny.memoryState.headHistory.push({
      timestamp: new Date(),
      input: input,
      headOutput: headOutput,
      confidence: (headOutput as { confidence?: number }).confidence,
    });

    // Keep only recent HEAD history
    if (contextAny.memoryState.headHistory.length > 10) {
      contextAny.memoryState.headHistory = contextAny.memoryState.headHistory.slice(-10);
    }
  }

  private async executeHEADEnhancedStrategy(
    strategy: string,
    attentionResult: unknown,
    options: unknown
  ): Promise<unknown> {
    console.log(`Executing HEAD-enhanced strategy: ${strategy}`);

    const headGuidance = (options as { headGuidance?: unknown }).headGuidance;

    switch (strategy) {
      case 'head_deep_analysis':
        return await this.headDeepAnalysisProcessing(attentionResult, options, headGuidance);

      case 'head_standard_processing':
        return await this.headStandardProcessing(attentionResult, options, headGuidance);

      case 'head_fast_processing':
        return await this.headFastProcessing(attentionResult, options, headGuidance);

      default:
        return await this.standardProcessing(attentionResult, options);
    }
  }

  private async headDeepAnalysisProcessing(attentionResult: unknown, options: unknown, headGuidance: unknown): Promise<unknown> {
    // Deep analysis with HEAD cognitive insights
    const result = await this.deepAnalysisProcessing(attentionResult, options);

    return {
      ...(result as Record<string, unknown>),
      headEnhanced: true,
      headReasoning: (headGuidance as { reasoning?: unknown }).reasoning,
      headConfidence: (headGuidance as { confidence?: unknown }).confidence,
      analysisDepth: 'deep_with_head',
      cognitiveInsights: {
        l_module_patterns: (headGuidance as { l_activations?: unknown }).l_activations,
        h_module_planning: (headGuidance as { h_activations?: unknown }).h_activations,
      },
    };
  }

  private async headStandardProcessing(attentionResult: unknown, options: unknown, headGuidance: unknown): Promise<unknown> {
    // Standard processing with HEAD guidance
    const result = await this.standardProcessing(attentionResult, options);

    return {
      ...(result as Record<string, unknown>),
      headEnhanced: true,
      headReasoning: (headGuidance as { reasoning?: unknown }).reasoning,
      headConfidence: (headGuidance as { confidence?: unknown }).confidence,
      analysisDepth: 'standard_with_head',
    };
  }

  private async headFastProcessing(attentionResult: unknown, options: unknown, headGuidance: unknown): Promise<unknown> {
    // Fast processing with minimal HEAD overhead
    const result = await this.fastProcessing(attentionResult, options);

    return {
      ...(result as Record<string, unknown>),
      headEnhanced: true,
      headConfidence: (headGuidance as { confidence?: unknown }).confidence,
      analysisDepth: 'fast_with_head',
    };
  }

  private async adaptiveProcess(input: unknown, options: unknown): Promise<unknown> {
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
    const attentionResult = await this.applyAttention(input, (options as { context?: unknown }).context);

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

  private async directProcess(input: unknown, options: unknown): Promise<unknown> {
    // Direct processing without adaptive mechanisms
    const result = await this.executeBasicProcessing(input, options);
    this.updateContext(input, result, options);
    return result;
  }

  private analyzeComplexity(input: unknown): number {
    let complexity = 0;

    // Analyze input structure
    if (typeof input === 'object' && input !== null) {
      complexity += Object.keys(input).length * 0.1;

      // Check for nested structures
      for (const value of Object.values(input as Record<string, unknown>)) {
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
    if (Array.isArray(context)) {
      for (const [, value] of context) {
        const contextStr = JSON.stringify(value).toLowerCase();
        const similarity = this.calculateSimilarity(inputStr, contextStr);
        relevance += similarity * 0.1;
      }
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

  private async applyAttention(input: unknown, context: unknown): Promise<{ focusedInput: unknown; attentionWeights: Map<string, number>; focusAreas: string[] }> {
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
    if (typeof input === 'object' && input !== null) {
      for (const [key, value] of Object.entries(input as Record<string, unknown>)) {
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
    if (context && (context as { has?: (key: string) => boolean }).has?.(key)) {
      weight += 0.2;
    }

    return Math.min(weight, 1.0);
  }

  private applyAttentionToInput(input: unknown): unknown {
    if (typeof input !== 'object' || input === null) {
      return input;
    }

    const focusedInput: Record<string, unknown> = {};

    for (const [key, value] of Object.entries(input as Record<string, unknown>)) {
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
  ): Promise<unknown> {
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

  private async deepAnalysisProcessing(attentionResult: unknown, options: unknown): Promise<unknown> {
    // Simulate deep analysis with multiple passes
    const passes = 3;
    let result = (attentionResult as { focusedInput?: unknown }).focusedInput;

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

  private async standardProcessing(attentionResult: unknown, options: unknown): Promise<unknown> {
    const result = await this.processPass((attentionResult as { focusedInput?: unknown }).focusedInput, 'standard', options);

    return {
      type: 'standard_result',
      result,
      strategy: 'standard_processing',
      confidence: 0.75,
      processingTime: Date.now(),
    };
  }

  private async fastProcessing(attentionResult: unknown, _options: unknown): Promise<unknown> {
    // Quick processing with minimal analysis
    const result = {
      processed: true,
      input: (attentionResult as { focusedInput?: unknown }).focusedInput,
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

  private async executeBasicProcessing(input: unknown, options: unknown): Promise<unknown> {
    const result = await this.processPass(input, 'basic', options);

    return {
      type: 'basic_result',
      result,
      strategy: 'basic_processing',
      confidence: 0.7,
      processingTime: Date.now(),
    };
  }

  private async processPass(input: unknown, passType: string, options: unknown): Promise<unknown> {
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
    interface MemoryEntry {
      lastSeen?: Date;
      frequency?: number;
      associatedResults?: string[];
      associatedInputs?: string[];
    }

    // Update memory with key patterns and relationships
    const inputKey = this.generateMemoryKey(input);
    const resultKey = this.generateMemoryKey(result);

    const memoryState = this.context.memoryState as Record<string, MemoryEntry>;

    memoryState[inputKey] = {
      lastSeen: new Date(),
      frequency: (memoryState[inputKey]?.frequency || 0) + 1,
      associatedResults: [resultKey],
    };

    // Create associations
    if (memoryState[resultKey]) {
      memoryState[resultKey].associatedInputs = memoryState[resultKey].associatedInputs || [];
      memoryState[resultKey].associatedInputs!.push(inputKey);
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
            await this.process((item as { input?: unknown }).input, (item as { options?: unknown }).options);
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
    return { ...(this.attentionMechanism as AttentionMechanism) };
  }

  public getMetrics(): ProcessingMetrics {
    return { ...(this.context.processingMetrics as ProcessingMetrics) };
  }

  public clearContext(): void {
    this.initializeContext();
    this.emit('contextCleared');
  }

  public exportMemoryState(): unknown {
    return { ...(this.context.memoryState as Record<string, unknown>) };
  }

  public importMemoryState(memoryState: unknown): void {
    const contextAny = this.context as { memoryState?: unknown };
    contextAny.memoryState = { ...(typeof memoryState === 'object' && memoryState !== null ? memoryState as Record<string, unknown> : {}) };
    this.emit('memoryStateImported');
  }

  // HEAD Integration methods
  public setHEADBridge(headBridge: unknown): void {
    this.headBridge = headBridge;
    console.log('HEAD bridge injected into Fabric Algorithm');
  }

  public enableHEADIntegration(): void {
    this.config.headIntegration = true;
    console.log('HEAD integration enabled in Fabric Algorithm');
  }

  public disableHEADIntegration(): void {
    this.config.headIntegration = false;
    console.log('HEAD integration disabled in Fabric Algorithm');
  }

  public isHEADIntegrationEnabled(): boolean {
    return this.config.headIntegration === true;
  }

  public getHEADStatus(): unknown {
    return {
      enabled: this.config.headIntegration,
      bridgeAvailable: this.headBridge !== null,
      ready: this.headBridge ? (this.headBridge as { isReady?: () => boolean }).isReady?.() : false,
    };
  }

  public getHEADHistory(): unknown[] {
    return (this.context as { memoryState?: { headHistory?: unknown[] } }).memoryState?.headHistory || [];
  }

  public clearHEADHistory(): void {
    const contextAny = this.context as { memoryState?: { headHistory?: unknown[] } };
    if (contextAny.memoryState?.headHistory) {
      contextAny.memoryState.headHistory = [];
      this.emit('headHistoryCleared');
    }
  }

  public getEnhancedMetrics(): unknown {
    const baseMetrics = this.getMetrics();
    const headHistory = this.getHEADHistory();

    return {
      ...baseMetrics,
      headIntegration: this.config.headIntegration,
      headProcessedCount: headHistory.length,
      averageHEADConfidence: headHistory.length > 0
        ? (headHistory as { confidence?: number }[]).reduce((sum: number, item: { confidence?: number }) => sum + (item.confidence || 0), 0) / headHistory.length
        : 0,
      lastHEADProcessing: headHistory.length > 0
        ? (headHistory[headHistory.length - 1] as { timestamp?: unknown }).timestamp
        : null,
    };
  }
}
