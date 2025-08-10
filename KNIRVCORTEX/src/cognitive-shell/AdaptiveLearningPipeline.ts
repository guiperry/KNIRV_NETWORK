import { EventEmitter } from './EventEmitter';

export interface UserInteraction {
  id: string;
  timestamp: Date;
  inputType: 'text' | 'voice' | 'visual' | 'gesture';
  input: any;
  output: any;
  userFeedback?: number; // -1 to 1 scale
  implicitFeedback?: number; // Derived from behavior
  context: any;
  sessionId: string;
}

export interface LearningPattern {
  patternId: string;
  inputPattern: any;
  outputPattern: any;
  confidence: number;
  frequency: number;
  lastSeen: Date;
  adaptationStrength: number;
}

export interface AdaptationMetrics {
  totalInteractions: number;
  positiveAdaptations: number;
  negativeAdaptations: number;
  averageConfidence: number;
  learningRate: number;
  adaptationEffectiveness: number;
  lastAdaptation: Date;
}

export interface LearningConfig {
  minInteractionsForPattern: number;
  adaptationThreshold: number;
  maxPatternsStored: number;
  learningRateDecay: number;
  feedbackWeight: number;
  hrmInfluenceWeight: number;
  realTimeAdaptation: boolean;
}

export class AdaptiveLearningPipeline extends EventEmitter {
  private config: LearningConfig;
  private interactions: UserInteraction[] = [];
  private patterns: Map<string, LearningPattern> = new Map();
  private metrics: AdaptationMetrics;
  private isRunning: boolean = false;
  private hrmBridge: any = null;
  private enhancedLoraAdapter: any = null;
  private hrmLoraBridge: any = null;
  private currentSessionId: string = '';

  constructor(config?: Partial<LearningConfig>) {
    super();
    
    this.config = {
      minInteractionsForPattern: 3,
      adaptationThreshold: 0.6,
      maxPatternsStored: 1000,
      learningRateDecay: 0.95,
      feedbackWeight: 0.7,
      hrmInfluenceWeight: 0.3,
      realTimeAdaptation: true,
      ...config,
    };

    this.initializeMetrics();
    this.generateSessionId();
  }

  private initializeMetrics(): void {
    this.metrics = {
      totalInteractions: 0,
      positiveAdaptations: 0,
      negativeAdaptations: 0,
      averageConfidence: 0.5,
      learningRate: 0.01,
      adaptationEffectiveness: 0.5,
      lastAdaptation: new Date(),
    };
  }

  private generateSessionId(): void {
    this.currentSessionId = `session_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
  }

  public setHRMBridge(hrmBridge: any): void {
    this.hrmBridge = hrmBridge;
    console.log('HRM bridge connected to Adaptive Learning Pipeline');
  }

  public setEnhancedLoRAAdapter(enhancedLoraAdapter: any): void {
    this.enhancedLoraAdapter = enhancedLoraAdapter;
    console.log('Enhanced LoRA adapter connected to Adaptive Learning Pipeline');
  }

  public setHRMLoRABridge(hrmLoraBridge: any): void {
    this.hrmLoraBridge = hrmLoraBridge;
    console.log('HRM-LoRA bridge connected to Adaptive Learning Pipeline');
  }

  public async start(): Promise<void> {
    console.log('Starting Adaptive Learning Pipeline...');
    
    this.isRunning = true;
    this.generateSessionId();
    
    // Start background learning process
    this.startLearningLoop();
    
    this.emit('pipelineStarted');
    console.log('Adaptive Learning Pipeline started successfully');
  }

  public async stop(): Promise<void> {
    console.log('Stopping Adaptive Learning Pipeline...');
    
    this.isRunning = false;
    
    // Save learned patterns before stopping
    await this.saveLearnedPatterns();
    
    this.emit('pipelineStopped');
    console.log('Adaptive Learning Pipeline stopped');
  }

  public async recordInteraction(interaction: Omit<UserInteraction, 'id' | 'timestamp' | 'sessionId'>): Promise<void> {
    const fullInteraction: UserInteraction = {
      id: `interaction_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`,
      timestamp: new Date(),
      sessionId: this.currentSessionId,
      ...interaction,
    };

    this.interactions.push(fullInteraction);
    this.metrics.totalInteractions++;

    // Derive implicit feedback if not provided
    if (fullInteraction.implicitFeedback === undefined) {
      fullInteraction.implicitFeedback = await this.deriveImplicitFeedback(fullInteraction);
    }

    this.emit('interactionRecorded', fullInteraction);

    // Trigger real-time adaptation if enabled
    if (this.config.realTimeAdaptation) {
      await this.processInteractionForLearning(fullInteraction);
    }

    // Maintain interaction history size
    if (this.interactions.length > this.config.maxPatternsStored * 2) {
      this.interactions = this.interactions.slice(-this.config.maxPatternsStored);
    }
  }

  private async deriveImplicitFeedback(interaction: UserInteraction): Promise<number> {
    // Derive implicit feedback from various signals
    let implicitScore = 0;

    // Use HRM confidence as a signal
    if (this.hrmBridge && this.hrmBridge.isReady()) {
      try {
        const hrmInput = {
          sensory_data: this.convertToSensoryData(interaction.input),
          context: JSON.stringify(interaction.context),
          task_type: 'feedback_analysis',
        };

        const hrmOutput = await this.hrmBridge.processCognitiveInput(hrmInput);
        implicitScore += hrmOutput.confidence * 0.4;

      } catch (error) {
        console.error('Error getting HRM feedback:', error);
      }
    }

    // Analyze output quality
    if (interaction.output) {
      const outputQuality = this.analyzeOutputQuality(interaction.output);
      implicitScore += outputQuality * 0.3;
    }

    // Consider interaction timing and context
    const contextScore = this.analyzeContextQuality(interaction.context);
    implicitScore += contextScore * 0.3;

    // Normalize to -1 to 1 range
    return Math.max(-1, Math.min(1, implicitScore));
  }

  private convertToSensoryData(input: any): number[] {
    // Convert input to numerical representation
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
      return this.convertToSensoryData(str);
    }

    return new Array(512).fill(0);
  }

  private analyzeOutputQuality(output: any): number {
    // Simple output quality analysis
    let quality = 0.5; // Base quality

    if (typeof output === 'object') {
      if (output.confidence) {
        quality += output.confidence * 0.3;
      }

      if (output.text && output.text.length > 10) {
        quality += 0.2; // Bonus for substantial text
      }

      if (output.hrmEnhanced) {
        quality += 0.1; // Bonus for HRM enhancement
      }
    }

    return Math.max(0, Math.min(1, quality));
  }

  private analyzeContextQuality(context: any): number {
    // Analyze context richness and relevance
    let quality = 0.5;

    if (context && typeof context === 'object') {
      const keys = Object.keys(context);
      quality += Math.min(keys.length * 0.05, 0.3); // More context is better

      if (context.confidenceLevel && context.confidenceLevel > 0.7) {
        quality += 0.2;
      }
    }

    return Math.max(0, Math.min(1, quality));
  }

  private async processInteractionForLearning(interaction: UserInteraction): Promise<void> {
    try {
      // Extract patterns from the interaction
      const patterns = await this.extractPatterns(interaction);

      // Update or create learning patterns
      for (const pattern of patterns) {
        await this.updateLearningPattern(pattern, interaction);
      }

      // Trigger adaptation if thresholds are met
      await this.triggerAdaptationIfNeeded(interaction);

    } catch (error) {
      console.error('Error processing interaction for learning:', error);
    }
  }

  private async extractPatterns(interaction: UserInteraction): Promise<LearningPattern[]> {
    const patterns: LearningPattern[] = [];

    // Create input-output pattern
    const ioPattern: LearningPattern = {
      patternId: `io_${interaction.inputType}_${this.hashInput(interaction.input)}`,
      inputPattern: this.normalizeInput(interaction.input),
      outputPattern: this.normalizeOutput(interaction.output),
      confidence: this.calculatePatternConfidence(interaction),
      frequency: 1,
      lastSeen: interaction.timestamp,
      adaptationStrength: this.calculateAdaptationStrength(interaction),
    };

    patterns.push(ioPattern);

    // Create context-based patterns if context is rich enough
    if (interaction.context && Object.keys(interaction.context).length > 2) {
      const contextPattern: LearningPattern = {
        patternId: `ctx_${this.hashInput(interaction.context)}`,
        inputPattern: this.normalizeInput(interaction.context),
        outputPattern: this.normalizeOutput(interaction.output),
        confidence: this.calculatePatternConfidence(interaction) * 0.8,
        frequency: 1,
        lastSeen: interaction.timestamp,
        adaptationStrength: this.calculateAdaptationStrength(interaction) * 0.8,
      };

      patterns.push(contextPattern);
    }

    return patterns;
  }

  private hashInput(input: any): string {
    // Simple hash function for input
    const str = JSON.stringify(input);
    let hash = 0;
    for (let i = 0; i < str.length; i++) {
      const char = str.charCodeAt(i);
      hash = ((hash << 5) - hash) + char;
      hash = hash & hash; // Convert to 32-bit integer
    }
    return Math.abs(hash).toString(36);
  }

  private normalizeInput(input: any): any {
    // Normalize input for pattern matching
    if (typeof input === 'string') {
      return input.toLowerCase().trim();
    }

    if (Array.isArray(input)) {
      return input.map(item => this.normalizeInput(item));
    }

    if (typeof input === 'object' && input !== null) {
      const normalized: any = {};
      for (const [key, value] of Object.entries(input)) {
        normalized[key.toLowerCase()] = this.normalizeInput(value);
      }
      return normalized;
    }

    return input;
  }

  private normalizeOutput(output: any): any {
    // Normalize output for pattern storage
    if (typeof output === 'object' && output !== null) {
      return {
        type: output.type || 'unknown',
        confidence: output.confidence || 0.5,
        text: output.text || '',
        hrmEnhanced: output.hrmEnhanced || false,
      };
    }

    return { text: String(output), confidence: 0.5 };
  }

  private calculatePatternConfidence(interaction: UserInteraction): number {
    let confidence = 0.5; // Base confidence

    // Factor in user feedback
    if (interaction.userFeedback !== undefined) {
      confidence += interaction.userFeedback * this.config.feedbackWeight * 0.3;
    }

    // Factor in implicit feedback
    if (interaction.implicitFeedback !== undefined) {
      confidence += interaction.implicitFeedback * 0.2;
    }

    // Factor in output quality
    if (interaction.output && interaction.output.confidence) {
      confidence += interaction.output.confidence * 0.3;
    }

    return Math.max(0, Math.min(1, confidence));
  }

  private calculateAdaptationStrength(interaction: UserInteraction): number {
    let strength = 0.1; // Base adaptation strength

    // Increase strength for explicit positive feedback
    if (interaction.userFeedback && interaction.userFeedback > 0.5) {
      strength += interaction.userFeedback * 0.3;
    }

    // Increase strength for high-confidence outputs
    if (interaction.output && interaction.output.confidence > 0.8) {
      strength += 0.2;
    }

    // Factor in HRM confidence if available
    if (interaction.output && interaction.output.hrmConfidence) {
      strength += interaction.output.hrmConfidence * this.config.hrmInfluenceWeight;
    }

    return Math.max(0.05, Math.min(0.8, strength));
  }

  private async updateLearningPattern(pattern: LearningPattern, interaction: UserInteraction): Promise<void> {
    const existingPattern = this.patterns.get(pattern.patternId);

    if (existingPattern) {
      // Update existing pattern
      existingPattern.frequency++;
      existingPattern.lastSeen = pattern.lastSeen;
      existingPattern.confidence = (existingPattern.confidence + pattern.confidence) / 2;
      existingPattern.adaptationStrength = Math.max(existingPattern.adaptationStrength, pattern.adaptationStrength);

      this.emit('patternUpdated', existingPattern);
    } else {
      // Add new pattern
      this.patterns.set(pattern.patternId, pattern);
      this.emit('patternCreated', pattern);
    }

    // Maintain pattern storage limits
    if (this.patterns.size > this.config.maxPatternsStored) {
      this.pruneOldPatterns();
    }
  }

  private pruneOldPatterns(): void {
    // Remove oldest and least frequent patterns
    const sortedPatterns = Array.from(this.patterns.entries())
      .sort(([, a], [, b]) => {
        const scoreA = a.frequency * a.confidence;
        const scoreB = b.frequency * b.confidence;
        return scoreA - scoreB;
      });

    const toRemove = sortedPatterns.slice(0, Math.floor(this.config.maxPatternsStored * 0.1));
    
    for (const [patternId] of toRemove) {
      this.patterns.delete(patternId);
    }

    this.emit('patternsPruned', toRemove.length);
  }

  private async triggerAdaptationIfNeeded(interaction: UserInteraction): Promise<void> {
    // Check if we should trigger adaptation
    const shouldAdapt = this.shouldTriggerAdaptation(interaction);

    if (!shouldAdapt) return;

    try {
      // Prepare training data from recent patterns
      const trainingData = this.prepareTrainingData();

      if (trainingData.length === 0) return;

      // Apply adaptation through Enhanced LoRA
      if (this.enhancedLoraAdapter && this.enhancedLoraAdapter.isAdapterReady()) {
        await this.enhancedLoraAdapter.trainOnBatch(trainingData);
        this.metrics.positiveAdaptations++;
      }

      // Trigger HRM-LoRA synchronization
      if (this.hrmLoraBridge && this.hrmLoraBridge.getStatus().isRunning) {
        await this.hrmLoraBridge.forceSyncNow();
      }

      this.metrics.lastAdaptation = new Date();
      this.updateAdaptationMetrics();

      this.emit('adaptationTriggered', {
        interaction,
        trainingDataSize: trainingData.length,
        timestamp: new Date(),
      });

    } catch (error) {
      console.error('Error triggering adaptation:', error);
      this.metrics.negativeAdaptations++;
      this.emit('adaptationError', error);
    }
  }

  private shouldTriggerAdaptation(interaction: UserInteraction): boolean {
    // Check various conditions for triggering adaptation
    
    // Must have minimum interactions
    if (this.interactions.length < this.config.minInteractionsForPattern) {
      return false;
    }

    // Check if we have strong enough patterns
    const strongPatterns = Array.from(this.patterns.values())
      .filter(p => p.confidence > this.config.adaptationThreshold && p.frequency >= 2);

    if (strongPatterns.length === 0) {
      return false;
    }

    // Check feedback quality
    const totalFeedback = interaction.userFeedback || 0 + interaction.implicitFeedback || 0;
    if (Math.abs(totalFeedback) < 0.3) {
      return false; // Feedback not strong enough
    }

    return true;
  }

  private prepareTrainingData(): any[] {
    const trainingData: any[] = [];

    // Get recent high-confidence patterns
    const recentPatterns = Array.from(this.patterns.values())
      .filter(p => p.confidence > this.config.adaptationThreshold)
      .sort((a, b) => b.lastSeen.getTime() - a.lastSeen.getTime())
      .slice(0, 20); // Limit to most recent 20 patterns

    for (const pattern of recentPatterns) {
      trainingData.push({
        input: pattern.inputPattern,
        output: pattern.outputPattern,
        feedback: pattern.confidence,
        timestamp: pattern.lastSeen,
      });
    }

    return trainingData;
  }

  private updateAdaptationMetrics(): void {
    const totalAdaptations = this.metrics.positiveAdaptations + this.metrics.negativeAdaptations;
    
    if (totalAdaptations > 0) {
      this.metrics.adaptationEffectiveness = this.metrics.positiveAdaptations / totalAdaptations;
    }

    // Calculate average confidence from patterns
    const patterns = Array.from(this.patterns.values());
    if (patterns.length > 0) {
      this.metrics.averageConfidence = patterns.reduce((sum, p) => sum + p.confidence, 0) / patterns.length;
    }

    // Apply learning rate decay
    this.metrics.learningRate *= this.config.learningRateDecay;

    this.emit('metricsUpdated', this.metrics);
  }

  private startLearningLoop(): void {
    // Background learning process
    const learningLoop = async () => {
      while (this.isRunning) {
        try {
          // Periodic pattern analysis and optimization
          await this.analyzeAndOptimizePatterns();
          
          // Wait before next iteration
          await new Promise(resolve => setTimeout(resolve, 10000)); // 10 seconds

        } catch (error) {
          console.error('Error in learning loop:', error);
        }
      }
    };

    learningLoop();
  }

  private async analyzeAndOptimizePatterns(): Promise<void> {
    // Analyze pattern effectiveness and optimize
    const patterns = Array.from(this.patterns.values());
    
    for (const pattern of patterns) {
      // Decay old patterns
      const ageInDays = (Date.now() - pattern.lastSeen.getTime()) / (1000 * 60 * 60 * 24);
      if (ageInDays > 7) {
        pattern.confidence *= 0.95; // Decay confidence for old patterns
      }
    }

    // Remove very low confidence patterns
    for (const [patternId, pattern] of this.patterns) {
      if (pattern.confidence < 0.1) {
        this.patterns.delete(patternId);
      }
    }

    this.emit('patternsOptimized', {
      totalPatterns: this.patterns.size,
      timestamp: new Date(),
    });
  }

  private async saveLearnedPatterns(): Promise<void> {
    // Save patterns to local storage or external storage
    try {
      const patternsData = {
        patterns: Object.fromEntries(this.patterns),
        metrics: this.metrics,
        config: this.config,
        timestamp: new Date(),
      };

      localStorage.setItem('knirv_learned_patterns', JSON.stringify(patternsData));
      console.log('Learned patterns saved successfully');

    } catch (error) {
      console.error('Error saving learned patterns:', error);
    }
  }

  public async loadLearnedPatterns(): Promise<void> {
    // Load patterns from storage
    try {
      const savedData = localStorage.getItem('knirv_learned_patterns');
      if (!savedData) return;

      const patternsData = JSON.parse(savedData);
      
      // Restore patterns
      this.patterns.clear();
      for (const [patternId, pattern] of Object.entries(patternsData.patterns)) {
        this.patterns.set(patternId, pattern as LearningPattern);
      }

      // Restore metrics
      if (patternsData.metrics) {
        this.metrics = { ...this.metrics, ...patternsData.metrics };
      }

      console.log(`Loaded ${this.patterns.size} learned patterns`);
      this.emit('patternsLoaded', this.patterns.size);

    } catch (error) {
      console.error('Error loading learned patterns:', error);
    }
  }

  public getMetrics(): AdaptationMetrics {
    return { ...this.metrics };
  }

  public getPatterns(): LearningPattern[] {
    return Array.from(this.patterns.values());
  }

  public getConfig(): LearningConfig {
    return { ...this.config };
  }

  public updateConfig(newConfig: Partial<LearningConfig>): void {
    this.config = { ...this.config, ...newConfig };
    this.emit('configUpdated', this.config);
  }

  public clearPatterns(): void {
    this.patterns.clear();
    this.initializeMetrics();
    this.emit('patternsCleared');
  }

  public getStatus(): any {
    return {
      isRunning: this.isRunning,
      currentSessionId: this.currentSessionId,
      totalInteractions: this.interactions.length,
      totalPatterns: this.patterns.size,
      metrics: this.metrics,
      config: this.config,
    };
  }
}
