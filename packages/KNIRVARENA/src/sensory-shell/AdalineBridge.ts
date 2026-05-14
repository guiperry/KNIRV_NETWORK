import { EventEmitter } from './EventEmitter';
import { getLLMProviderService, LLMProviderService } from '../services/llmProviderService';
import type { LLMProvider, ChatMessage } from '../types/chatBrain';

export interface AdalineCognitiveInput {
  sensory_data: number[];
  context: string;
  task_type: string;
  anchorDataset?: AnchorDatasetEntry[];
  noiseLevel?: number;
}

export interface AnchorDatasetEntry {
  template: string;
  context: Record<string, unknown>;
  examples: AnchorExample[];
  metadata?: Record<string, unknown>;
}

export interface AnchorExample {
  input: string;
  output: string;
  confidence: number;
}

export interface AdalineCognitiveOutput {
  reasoning_result: string;
  confidence: number;
  processing_time: number;
  l_module_activations: number[];
  h_module_activations: number[];
  provider_used?: string;
  anchor_applied?: boolean;
  noise_filtered?: boolean;
  validation_score?: number;
  metadata?: Record<string, unknown>;
}

export interface AdalineModelInfo {
  total_parameters: number;
  providers: string[];
  active_provider: string;
  capabilities: string[];
}

export interface AdalineConfig {
  enabled: boolean;
  defaultProvider: LLMProvider;
  fallbackProviders: LLMProvider[];
  enableAnchorDatasets: boolean;
  enableNoiseFiltering: boolean;
  enableDVERouting: boolean;
  processingTimeout: number;
  maxRetries: number;
  confidenceThreshold: number;
}

export interface DVEValidationResult {
  score: number;
  passed: boolean;
  simulationResults?: unknown[];
  warnings?: string[];
  timestamp: number;
}

export interface CDESandboxResult {
  success: boolean;
  output: unknown;
  constraintsSatisfied: boolean;
  violations?: string[];
  executionTime: number;
}

export enum SabotageType {
  NOISE_INJECTION = 'noise_injection',
  CONTEXT_POISONING = 'context_poisoning',
  PROMPT_INJECTION = 'prompt_injection',
  ADVERSARIAL_DRIFT = 'adversarial_drift',
  UNKNOWN = 'unknown'
}

const DEFAULT_ADALINE_CONFIG: AdalineConfig = {
  enabled: true,
  defaultProvider: 'adaline',
  fallbackProviders: ['gemini', 'openai'],
  enableAnchorDatasets: true,
  enableNoiseFiltering: true,
  enableDVERouting: true,
  processingTimeout: 30000,
  maxRetries: 3,
  confidenceThreshold: 0.7,
};

export class AdalineBridge extends EventEmitter {
  private config: AdalineConfig;
  private llmProviderService: LLMProviderService;
  private isInitialized: boolean = false;
  private modelInfo: AdalineModelInfo | null = null;
  private activeProvider: LLMProvider;
  private retryCount: Map<string, number> = new Map();

  constructor(config: Partial<AdalineConfig> = {}) {
    super();
    this.config = { ...DEFAULT_ADALINE_CONFIG, ...config };
    this.llmProviderService = getLLMProviderService();
    this.activeProvider = this.config.defaultProvider;
  }

  public async initialize(): Promise<void> {
    try {
      console.log('Initializing Adaline Bridge...');

      this.activeProvider = await this.selectBestProvider();

      this.modelInfo = {
        total_parameters: 0,
        providers: this.llmProviderService.getAvailableProviders(),
        active_provider: this.activeProvider,
        capabilities: [
          'text_processing',
          'voice_processing',
          'visual_processing',
          'anchor_dataset_injection',
          'noise_filtering',
          'dve_validation',
          'cde_sandboxing'
        ],
      };

      this.isInitialized = true;
      this.emit('initialized');
      console.log('Adaline Bridge initialized successfully');

    } catch (error) {
      console.error('Failed to initialize Adaline Bridge:', error);
      this.emit('error', error);
      throw error;
    }
  }

  private async selectBestProvider(): Promise<LLMProvider> {
    const availableProviders = this.llmProviderService.getAvailableProviders();

    if (availableProviders.includes(this.config.defaultProvider)) {
      return this.config.defaultProvider;
    }

    for (const fallback of this.config.fallbackProviders) {
      if (availableProviders.includes(fallback)) {
        console.log(`Using fallback provider: ${fallback}`);
        return fallback;
      }
    }

    throw new Error('No available LLM providers found');
  }

  public async processCognitiveInput(input: AdalineCognitiveInput): Promise<AdalineCognitiveOutput> {
    if (!this.isInitialized) {
      throw new Error('Adaline Bridge not initialized');
    }

    const startTime = performance.now();

    try {
      let processedInput = input;

      if (this.config.enableNoiseFiltering && (input.noiseLevel ?? 0) > 0.3) {
        processedInput = this.filterNoise(input);
      }

      const prompt = this.buildPrompt(processedInput);
      const history = this.buildHistoryFromContext(processedInput.context);

      const response = await this.llmProviderService.chat(
        prompt,
        this.activeProvider,
        history
      );

      const output = this.parseResponse(response.text);

      const processingTime = performance.now() - startTime;

      const adalineOutput: AdalineCognitiveOutput = {
        reasoning_result: output.reasoning,
        confidence: output.confidence,
        processing_time: processingTime,
        l_module_activations: this.generateModuleActivations('L', output.confidence),
        h_module_activations: this.generateModuleActivations('H', output.confidence),
        provider_used: this.activeProvider,
        anchor_applied: processedInput.anchorDataset !== undefined && processedInput.anchorDataset.length > 0,
        noise_filtered: processedInput.noiseLevel !== input.noiseLevel,
        validation_score: output.validationScore,
        metadata: {
          rawResponse: response.text,
          inputTokens: this.countTokens(prompt),
          outputTokens: this.countTokens(response.text),
          ...response.metadata,
        },
      };

      this.emit('inputProcessed', {
        input,
        output: adalineOutput,
        processingTime,
      });

      return adalineOutput;

    } catch (error) {
      console.error('Error processing cognitive input:', error);

      const currentRetries = this.retryCount.get('processCognitiveInput') || 0;

      if (currentRetries < this.config.maxRetries) {
        this.retryCount.set('processCognitiveInput', currentRetries + 1);
        await this.attemptProviderFailover();
        return this.processCognitiveInput(input);
      }

      this.emit('error', error);
      throw error;
    } finally {
      this.retryCount.delete('processCognitiveInput');
    }
  }

  public async processTextInput(text: string, context?: unknown): Promise<AdalineCognitiveOutput> {
    const input: AdalineCognitiveInput = {
      sensory_data: this.textToSensoryData(text),
      context: JSON.stringify(context || {}),
      task_type: 'text_processing',
    };

    return this.processCognitiveInput(input);
  }

  public async processVoiceInput(audioData: number[], context?: unknown): Promise<AdalineCognitiveOutput> {
    const input: AdalineCognitiveInput = {
      sensory_data: audioData,
      context: JSON.stringify(context || {}),
      task_type: 'voice_processing',
    };

    return this.processCognitiveInput(input);
  }

  public async processVisualInput(visualData: number[], context?: unknown): Promise<AdalineCognitiveOutput> {
    const input: AdalineCognitiveInput = {
      sensory_data: visualData,
      context: JSON.stringify(context || {}),
      task_type: 'visual_processing',
    };

    return this.processCognitiveInput(input);
  }

  public async processComplexReasoning(
    task: string,
    context: Record<string, unknown>,
    options?: {
      anchorDataset?: AnchorDatasetEntry[];
      validateWithDVE?: boolean;
      useCDE?: boolean;
      maxSeverity?: 'low' | 'medium' | 'high' | 'critical';
    }
  ): Promise<AdalineCognitiveOutput> {
    const input: AdalineCognitiveInput = {
      sensory_data: this.textToSensoryData(task),
      context: JSON.stringify(context),
      task_type: 'complex_reasoning',
      anchorDataset: options?.anchorDataset,
    };

    let output = await this.processCognitiveInput(input);

    if (options?.validateWithDVE && output.confidence >= this.config.confidenceThreshold) {
      const dveResult = await this.validateWithDVE(output);
      output.validation_score = dveResult.score;

      if (!dveResult.passed && dveResult.warnings?.length) {
        this.emit('dveWarning', {
          warnings: dveResult.warnings,
          validationScore: dveResult.score,
        });
      }
    }

    if (options?.useCDE) {
      const cdeResult = await this.validateWithCDE(output.reasoning_result, {
        maxSeverity: options.maxSeverity || 'high',
      });

      if (!cdeResult.success) {
        output.confidence = output.confidence * 0.5;
        this.emit('cdeViolation', {
          violations: cdeResult.violations,
        });
      }
    }

    return output;
  }

  private filterNoise(input: AdalineCognitiveInput): AdalineCognitiveInput {
    const entropyThreshold = 0.3;
    const currentNoiseLevel = input.noiseLevel ?? 0;

    if (currentNoiseLevel <= entropyThreshold) {
      return input;
    }

    const contextJson = input.context;
    let filteredContext = contextJson;

    const entropyScore = this.calculateEntropy(contextJson);

    if (entropyScore > entropyThreshold) {
      filteredContext = this.removeAdversarialPatterns(contextJson);
    }

    return {
      ...input,
      context: filteredContext,
      noiseLevel: entropyScore,
    };
  }

  private calculateEntropy(text: string): number {
    if (!text || text.length === 0) return 0;

    const charFrequencies = new Map<string, number>();
    for (const char of text) {
      charFrequencies.set(char, (charFrequencies.get(char) || 0) + 1);
    }

    let entropy = 0;
    const length = text.length;

    for (const freq of charFrequencies.values()) {
      const probability = freq / length;
      if (probability > 0) {
        entropy -= probability * Math.log2(probability);
      }
    }

    const maxEntropy = Math.log2(256);
    return entropy / maxEntropy;
  }

  private removeAdversarialPatterns(text: string): string {
    const adversarialPatterns = [
      /\x00-\x1F/g,
      /[\u200B-\u200F\uFEFF]/g,
      /[\u180E\u0600-\u0605]/g,
    ];

    let cleaned = text;

    for (const pattern of adversarialPatterns) {
      cleaned = cleaned.replace(pattern, '');
    }

    cleaned = this.filterRandomCharacters(cleaned);

    return cleaned;
  }

  private filterRandomCharacters(text: string): string {
    if (text.length < 10) return text;

    const charCounts = new Map<string, number>();
    for (const char of text) {
      if (/[a-zA-Z0-9\s.,!?;:'"()-]/.test(char)) {
        charCounts.set(char, (charCounts.get(char) || 0) + 1);
      }
    }

    const threshold = text.length * 0.01;
    const filteredChars: string[] = [];

    for (const char of text) {
      const count = charCounts.get(char) || 0;
      if (count >= threshold || /[a-zA-Z0-9\s.,!?;:'"()-]/.test(char)) {
        filteredChars.push(char);
      }
    }

    return filteredChars.join('');
  }

  public detectSabotageType(input: string): { type: SabotageType; confidence: number } {
    const noiseLevel = this.calculateEntropy(input);

    if (noiseLevel > 0.7) {
      return { type: SabotageType.NOISE_INJECTION, confidence: 0.9 };
    }

    const promptInjectionPatterns = [
      /(?:ignore\s+(?:previous|above)|forget\s+(?:all|previous)|disregard)\s+(?:instructions?|rules?|constraints?)/i,
      /(?:you\s+are\s+now|switch\s+to)\s+[:;]/i,
      /<\|(?:system|user)\|>/i,
      /\[INST\]/i,
    ];

    for (const pattern of promptInjectionPatterns) {
      if (pattern.test(input)) {
        return { type: SabotageType.PROMPT_INJECTION, confidence: 0.95 };
      }
    }

    const contextPoisoningPatterns = [
      /as\s+a\s+(?:hypothetical|different|pretend)/i,
      /for\s+(?:research|testing|educational)\s+purposes?/i,
      /(?:pretend|imagine)\s+you\s+(?:don't|have\s+no)/i,
    ];

    for (const pattern of contextPoisoningPatterns) {
      if (pattern.test(input)) {
        return { type: SabotageType.CONTEXT_POISONING, confidence: 0.85 };
      }
    }

    return { type: SabotageType.UNKNOWN, confidence: noiseLevel > 0.5 ? noiseLevel : 0.1 };
  }

  public async validateWithDVE(
    output: AdalineCognitiveOutput,
    _simulationConfig?: unknown
  ): Promise<DVEValidationResult> {
    const startTime = performance.now();

    try {
      const simulationResults = await this.runDVESimulation(output.reasoning_result);

      const score = this.calculateDVEValidationScore(output, simulationResults);

      const importMeta = eval('import.meta');
      const dveThreshold = importMeta?.env?.VITE_DVE_VALIDATION_THRESHOLD || '0.7';
      const passed = score >= parseFloat(dveThreshold);

      const result: DVEValidationResult = {
        score,
        passed,
        simulationResults,
        warnings: score < 0.9 ? ['Validation score below optimal threshold'] : undefined,
        timestamp: startTime,
      };

      this.emit('dveValidationComplete', result);

      return result;

    } catch (error) {
      console.error('DVE validation error:', error);

      return {
        score: 0,
        passed: false,
        warnings: ['DVE validation failed due to error'],
        timestamp: performance.now() - startTime,
      };
    }
  }

  private async runDVESimulation(reasoning: string): Promise<unknown[]> {
    const simulations: unknown[] = [];

    simulations.push({
      type: 'consistency_check',
      passed: this.checkConsistency(reasoning),
    });

    simulations.push({
      type: 'constraint_satisfaction',
      passed: this.checkConstraintSatisfaction(reasoning),
    });

    simulations.push({
      type: 'adversarial_robustness',
      score: this.calculateAdversarialRobustness(reasoning),
    });

    return simulations;
  }

  private checkConsistency(reasoning: string): boolean {
    const contradictions = [
      /(?:however|but)\s+(?:at\s+the\s+same\s+time|simultaneously)/i,
      /(?:on\s+the\s+other\s+hand|conversely)/i,
      /(?:despite|although|while)\s+(?:this|that|it)/i,
    ];

    const contradictionCount = contradictions.reduce(
      (count, pattern) => count + (reasoning.match(pattern)?.length || 0),
      0
    );

    return contradictionCount <= 2;
  }

  private checkConstraintSatisfaction(reasoning: string): boolean {
    const safeKeywords = [
      'safe', 'secure', 'verified', 'validated', 'checked',
      'confirmed', 'approved', 'permitted', 'allowed', 'legal'
    ];

    const unsafeKeywords = [
      'hack', 'exploit', 'bypass', 'crack', 'illegal',
      'unauthorized', 'forbidden', 'dangerous', 'harmful', 'attack'
    ];

    const hasSafeKeywords = safeKeywords.some(k => reasoning.toLowerCase().includes(k));
    const hasUnsafeKeywords = unsafeKeywords.some(k => reasoning.toLowerCase().includes(k));

    return hasSafeKeywords || !hasUnsafeKeywords;
  }

  private calculateAdversarialRobustness(reasoning: string): number {
    const promptInjectionIndicators = [
      /ignore\s+(?:previous|all)/i,
      /forget\s+instructions?/i,
      /disregard\s+rules?/i,
      /<[^>]+>/,
    ];

    let robustnessScore = 1.0;

    for (const indicator of promptInjectionIndicators) {
      if (indicator.test(reasoning)) {
        robustnessScore -= 0.3;
      }
    }

    return Math.max(0, Math.min(1, robustnessScore));
  }

  private calculateDVEValidationScore(
    output: AdalineCognitiveOutput,
    simulations: unknown[]
  ): number {
    const consistencyWeight = 0.3;
    const constraintWeight = 0.3;
    const robustnessWeight = 0.2;
    const confidenceWeight = 0.2;

    let consistencyScore = 1.0;
    let constraintScore = 1.0;
    let robustnessScore = 1.0;

    for (const sim of simulations) {
      const simAny = sim as { type?: string; passed?: boolean; score?: number };

      if (simAny.type === 'consistency_check') {
        consistencyScore = simAny.passed ? 1.0 : 0.3;
      }
      if (simAny.type === 'constraint_satisfaction') {
        constraintScore = simAny.passed ? 1.0 : 0.3;
      }
      if (simAny.type === 'adversarial_robustness') {
        robustnessScore = simAny.score ?? 0.5;
      }
    }

    return (
      consistencyScore * consistencyWeight +
      constraintScore * constraintWeight +
      robustnessScore * robustnessWeight +
      output.confidence * confidenceWeight
    );
  }

  public async validateWithCDE(
    solution: string,
    options?: { maxSeverity?: 'low' | 'medium' | 'high' | 'critical' }
  ): Promise<CDESandboxResult> {
    const startTime = performance.now();
    const maxSeverity = options?.maxSeverity || 'high';

    const severityLevels = { low: 1, medium: 2, high: 3, critical: 4 };
    const maxSeverityLevel = severityLevels[maxSeverity];

    const constraints = this.defineGameConstraints();
    const violations: string[] = [];

    for (const constraint of constraints) {
      if (constraint.severity > maxSeverityLevel) {
        continue;
      }

      if (constraint.check(solution)) {
        violations.push(constraint.description);
      }
    }

    const success = violations.length === 0;
    const constraintsSatisfied = violations.length <= 1;

    const result: CDESandboxResult = {
      success,
      output: solution,
      constraintsSatisfied,
      violations: violations.length > 0 ? violations : undefined,
      executionTime: performance.now() - startTime,
    };

    this.emit('cdeValidationComplete', result);

    return result;
  }

  private defineGameConstraints(): Array<{
    description: string;
    severity: number;
    check: (solution: string) => boolean;
  }> {
    return [
      {
        description: 'Solution contains potentially unsafe operations',
        severity: 3,
        check: (solution: string) => /hack|exploit|bypass|crack/i.test(solution),
      },
      {
        description: 'Solution references sensitive data inappropriately',
        severity: 3,
        check: (solution: string) => /password|secret|api[_-]?key|token/i.test(solution),
      },
      {
        description: 'Solution contains time-wasting loops or operations',
        severity: 2,
        check: (solution: string) => /\b(while\s*\(\s*true\s*\)|for\s*\(\s*;\s*;\s*\))/i.test(solution),
      },
      {
        description: 'Solution exceeds reasonable complexity',
        severity: 2,
        check: (solution: string) => {
          const lines = solution.split('\n').length;
          const nestedDepth = Math.max(
            ...solution.match(/\{[^}]*\{[^}]*\}[^}]*\}/g)?.map(m => (m.match(/\{/g) || []).length) || [0]
          );
          return lines > 500 || nestedDepth > 5;
        },
      },
      {
        description: 'Solution contains potentially harmful instructions',
        severity: 4,
        check: (solution: string) => /rm\s+-[rf]\s+\/|del\s+\/f\s+\/s\s+/i.test(solution),
      },
    ];
  }

  private buildPrompt(input: AdalineCognitiveInput): string {
    const parts: string[] = [];

    if (input.anchorDataset && input.anchorDataset.length > 0) {
      parts.push('## Contextual Examples:\n');
      for (const anchor of input.anchorDataset) {
        parts.push(`Template: ${anchor.template}`);
        if (anchor.examples.length > 0) {
          parts.push('Examples:');
          for (const example of anchor.examples) {
            parts.push(`  - Input: ${example.input}`);
            parts.push(`    Output: ${example.output} (confidence: ${example.confidence})`);
          }
        }
        parts.push('');
      }
    }

    parts.push('## Task Context:');
    parts.push(`Task Type: ${input.task_type}`);
    parts.push(`Context: ${input.context}`);

    let sensoryDescription = 'Sensory Data';
    switch (input.task_type) {
      case 'text_processing':
        sensoryDescription = `Text Input (${input.sensory_data.length} tokens)`;
        break;
      case 'voice_processing':
        sensoryDescription = `Voice Input (${input.sensory_data.length} samples)`;
        break;
      case 'visual_processing':
        sensoryDescription = `Visual Input (${input.sensory_data.length} features)`;
        break;
      default:
        sensoryDescription = `Input Data (${input.sensory_data.length} features)`;
    }

    parts.push(`## ${sensoryDescription}`);
    parts.push('## Reasoning Request:');
    parts.push('Please provide a structured reasoning response with the following format:');
    parts.push('1. Analysis of the context and sensory input');
    parts.push('2. Reasoning steps to derive the solution');
    parts.push('3. Final recommendation with confidence score (0-1)');

    return parts.join('\n');
  }

  private buildHistoryFromContext(context: string): ChatMessage[] {
    try {
      const parsed = JSON.parse(context);
      const history: ChatMessage[] = [];

      if (parsed.previousInteractions && Array.isArray(parsed.previousInteractions)) {
        for (const interaction of parsed.previousInteractions.slice(-5)) {
          history.push({
            id: `hist-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`,
            text: interaction.input || '',
            type: 'user',
            timestamp: interaction.timestamp || Date.now(),
          });

          if (interaction.output) {
            history.push({
              id: `hist-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`,
              text: interaction.output,
              type: 'bot',
              timestamp: (interaction.timestamp || Date.now()) + 1,
            });
          }
        }
      }

      return history;

    } catch {
      return [];
    }
  }

  private parseResponse(
    text: string
  ): { reasoning: string; confidence: number; validationScore?: number } {
    const confidenceMatch = text.match(/(?:confidence|confidence[:\s]+)(0?\.\d{1,2}|1\.0)/i);
    const confidence = confidenceMatch
      ? parseFloat(confidenceMatch[1])
      : 0.7;

    const validationMatch = text.match(/(?:validation[:\s]+)(0?\.\d{1,2}|1\.0)/i);
    const validationScore = validationMatch
      ? parseFloat(validationMatch[1])
      : undefined;

    const reasoningMatch = text.match(/##\s*Reasoning\s*(?:Request)?:?([^#]+)/is);
    let reasoning = reasoningMatch ? reasoningMatch[1].trim() : text;

    const cleanReasoning = reasoning
      .replace(/\n{3,}/g, '\n\n')
      .replace(/^\s+|\s+$/g, '')
      .trim();

    return {
      reasoning: cleanReasoning || text,
      confidence: Math.max(0, Math.min(1, confidence)),
      validationScore,
    };
  }

  private generateModuleActivations(moduleType: 'L' | 'H', confidence: number): number[] {
    const moduleCount = moduleType === 'L' ? 8 : 4;
    const activations: number[] = [];

    for (let i = 0; i < moduleCount; i++) {
      const baseActivation = confidence * 0.8 + Math.random() * 0.2;
      activations.push(Math.min(1, Math.max(0, baseActivation)));
    }

    return activations;
  }

  private countTokens(text: string): number {
    return Math.ceil(text.length / 4);
  }

  private textToSensoryData(text: string): number[] {
    const encoder = new TextEncoder();
    const bytes = encoder.encode(text);
    const normalized = Array.from(bytes).map((b) => b / 255.0);

    const maxLength = 512;
    if (normalized.length > maxLength) {
      return normalized.slice(0, maxLength);
    } else {
      return [...normalized, ...new Array(maxLength - normalized.length).fill(0)];
    }
  }

  private async attemptProviderFailover(): Promise<void> {
    const availableProviders = this.llmProviderService.getAvailableProviders();
    const currentIndex = availableProviders.indexOf(this.activeProvider);

    for (let i = currentIndex + 1; i < availableProviders.length; i++) {
      if (this.llmProviderService.isProviderAvailable(availableProviders[i])) {
        console.log(`Failing over to provider: ${availableProviders[i]}`);
        this.activeProvider = availableProviders[i];
        this.emit('providerFailover', { newProvider: this.activeProvider });
        return;
      }
    }

    for (const fallback of this.config.fallbackProviders) {
      if (availableProviders.includes(fallback)) {
        console.log(`Using fallback provider: ${fallback}`);
        this.activeProvider = fallback;
        this.emit('providerFailover', { newProvider: this.activeProvider });
        return;
      }
    }

    throw new Error('All LLM providers have failed');
  }

  public getModelInfo(): AdalineModelInfo | null {
    return this.modelInfo;
  }

  public isReady(): boolean {
    return this.isInitialized;
  }

  public async destroy(): Promise<void> {
    this.isInitialized = false;
    this.modelInfo = null;
    this.retryCount.clear();
    this.emit('destroyed');
    console.log('Adaline Bridge destroyed');
  }

  public getConfig(): AdalineConfig {
    return { ...this.config };
  }

  public updateConfig(newConfig: Partial<AdalineConfig>): void {
    this.config = { ...this.config, ...newConfig };
    this.emit('configUpdated', this.config);
  }

  public getActiveProvider(): string {
    return this.activeProvider;
  }

  public async process(data: unknown): Promise<unknown> {
    if (typeof data === 'object' && data !== null && 'sensory_data' in data) {
      return this.processCognitiveInput(data as AdalineCognitiveInput);
    }

    const adalineInput: AdalineCognitiveInput = {
      sensory_data: Array.isArray(data) ? data as number[] : this.textToSensoryData(String(data)),
      context: typeof data === 'string' ? '{}' : JSON.stringify(data),
      task_type: 'general',
    };

    return this.processCognitiveInput(adalineInput);
  }

  public isConnected(): boolean {
    return this.isInitialized && this.llmProviderService.isProviderAvailable(this.activeProvider);
  }
}

let adalineBridgeInstance: AdalineBridge | null = null;

export const getAdalineBridge = (config?: Partial<AdalineConfig>): AdalineBridge => {
  if (!adalineBridgeInstance) {
    adalineBridgeInstance = new AdalineBridge(config);
  }
  return adalineBridgeInstance;
};

export default getAdalineBridge;
