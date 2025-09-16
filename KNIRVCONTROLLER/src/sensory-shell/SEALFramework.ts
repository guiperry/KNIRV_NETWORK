import { EventEmitter } from './EventEmitter';

export interface SEALConfig {
  maxAgents: number;
  learningRate: number;
  adaptationThreshold: number;
  skillTimeout: number;
  hrmIntegration?: boolean;
}

export interface SEALAgent {
  id: string;
  type: string;
  capabilities: string[];
  state: unknown;
  performance: AgentPerformance;
  created: Date;
  lastActive: Date;
}

export interface AgentPerformance {
  successRate: number;
  averageLatency: number;
  totalInvocations: number;
  errorCount: number;
}

export interface SkillInvocation {
  skillId: string;
  parameters: unknown;
  agent?: SEALAgent;
  startTime: Date;
  endTime?: Date;
  result?: unknown;
  error?: string;
}

export class SEALFramework extends EventEmitter {
  private config: SEALConfig;
  private agents: Map<string, SEALAgent> = new Map();
  private activeInvocations: Map<string, SkillInvocation> = new Map();
  private learningMode: boolean = false;
  private isRunning: boolean = false;
  private hrmBridge: unknown = null; // Will be injected from CognitiveEngine

  constructor(config: SEALConfig) {
    super();
    this.config = config;
  }

  public async start(): Promise<void> {
    console.log('Starting SEAL Framework...');

    // Initialize default agents
    await this.createDefaultAgents();

    this.isRunning = true;
    this.emit('sealStarted');
  }

  public async stop(): Promise<void> {
    console.log('Stopping SEAL Framework...');

    // Stop all active invocations
    this.activeInvocations.forEach(async (invocation, id) => {
      await this.cancelInvocation(id);
    });

    this.agents.clear();
    this.isRunning = false;
    this.emit('sealStopped');
  }

  private async createDefaultAgents(): Promise<void> {
    const defaultAgents = [
      {
        type: 'text_processor',
        capabilities: ['text_analysis', 'summarization', 'translation'],
      },
      {
        type: 'code_assistant',
        capabilities: ['code_generation', 'debugging', 'refactoring'],
      },
      {
        type: 'problem_solver',
        capabilities: ['logical_reasoning', 'pattern_recognition', 'optimization'],
      },
      {
        type: 'visual_analyzer',
        capabilities: ['image_analysis', 'object_detection', 'scene_understanding'],
      },
      {
        type: 'voice_handler',
        capabilities: ['speech_processing', 'command_interpretation', 'voice_synthesis'],
      },
    ];

    for (const agentConfig of defaultAgents) {
      await this.createAgent(agentConfig.type, agentConfig.capabilities);
    }
  }

  public async createAgent(type: string, capabilities: string[]): Promise<SEALAgent> {
    const agentId = `agent_${type}_${Date.now()}`;

    const agent: SEALAgent = {
      id: agentId,
      type,
      capabilities,
      state: {},
      performance: {
        successRate: 0.0,
        averageLatency: 0.0,
        totalInvocations: 0,
        errorCount: 0,
      },
      created: new Date(),
      lastActive: new Date(),
    };

    this.agents.set(agentId, agent);
    this.emit('agentCreated', agent);

    console.log(`Created SEAL agent: ${agentId} (${type})`);
    return agent;
  }

  public async generateResponse(input: unknown, context: unknown): Promise<unknown> {
    const startTime = Date.now();

    try {
      // Use HRM for enhanced reasoning if available
      if (this.config.hrmIntegration && this.hrmBridge && (this.hrmBridge as { isReady?: () => boolean }).isReady?.()) {
        return await this.generateHRMEnhancedResponse(input, context);
      }

      // Fallback to traditional agent selection
      const agent = await this.selectAgent(input, context);

      if (!agent) {
        throw new Error('No suitable agent found for input');
      }

      // Generate response using selected agent
      const response = await this.executeWithAgent(agent, input, context);

      // Update agent performance
      this.updateAgentPerformance(agent, Date.now() - startTime, true);

      return response;

    } catch (error) {
      console.error('Error generating response:', error);
      throw error;
    }
  }

  private async generateHRMEnhancedResponse(input: unknown, context: unknown): Promise<unknown> {
    console.log('Generating HRM-enhanced SEAL response...');

    try {
      // First, get HRM reasoning about the input
      const hrmInput = {
        sensory_data: this.convertInputToSensoryData(input),
        context: JSON.stringify(context),
        task_type: this.determineTaskType(input, context),
      };

      const hrmOutput = await (this.hrmBridge as { processCognitiveInput?: (input: unknown) => Promise<unknown> }).processCognitiveInput?.(hrmInput);

      // Use HRM reasoning to select and guide agent execution
      const agent = await this.selectAgentWithHRMGuidance(input, context, hrmOutput);

      if (!agent) {
        // Fallback to HRM-only response
        return this.formatHRMResponse(hrmOutput, context);
      }

      // Execute agent with HRM guidance
      const response = await this.executeAgentWithHRMGuidance(agent, input, context, hrmOutput);

      // Update agent performance based on HRM confidence
      this.updateAgentPerformance(agent, (hrmOutput as { processing_time?: number }).processing_time || 0, ((hrmOutput as { confidence?: number }).confidence || 0) > 0.7);

      return response;

    } catch (error) {
      console.error('Error in HRM-enhanced response generation:', error);
      // Fallback to traditional processing
      return this.generateResponse(input, context);
    }
  }

  private convertInputToSensoryData(input: unknown): number[] {
    // Convert various input types to numerical data for HRM
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

  private determineTaskType(input: unknown, context: unknown): string {
    const contextAny = context as { inputType?: string };
    if (contextAny.inputType) {
      return contextAny.inputType + '_processing';
    }

    if (typeof input === 'string') {
      if (input.includes('code') || input.includes('function')) {
        return 'code_processing';
      }
      return 'text_processing';
    }

    return 'general_processing';
  }

  private async selectAgentWithHRMGuidance(input: unknown, context: unknown, hrmOutput: unknown): Promise<SEALAgent | null> {
    // Use HRM activations to guide agent selection
    const requiredCapabilities = this.analyzeRequiredCapabilities(input, context);

    let bestAgent: SEALAgent | null = null;
    let bestScore = 0;

    this.agents.forEach((agent) => {
      let score = this.calculateAgentScore(agent, requiredCapabilities);

      // Boost score based on HRM module activations
      const hrmAny = hrmOutput as { h_module_activations?: number[] };
      if (hrmAny.h_module_activations && hrmAny.h_module_activations.length > 0) {
        const avgActivation = hrmAny.h_module_activations.reduce((a: number, b: number) => a + b, 0) / hrmAny.h_module_activations.length;
        score *= (1 + avgActivation); // Boost by HRM confidence
      }

      if (score > bestScore) {
        bestScore = score;
        bestAgent = agent;
      }
    });

    return bestAgent;
  }

  private formatHRMResponse(hrmOutput: unknown, context: unknown): unknown {
    interface HRMOutput {
      reasoning_result?: unknown;
      confidence?: number;
      processing_time?: number;
      l_module_activations?: unknown;
      h_module_activations?: unknown;
    }

    const hrmTyped = hrmOutput as HRMOutput;
    const contextAny = context as Record<string, unknown>;
    return {
      type: 'hrm_response',
      content: hrmTyped.reasoning_result,
      confidence: hrmTyped.confidence,
      processingTime: hrmTyped.processing_time,
      source: 'hrm_direct',
      shouldSpeak: contextAny.inputType === 'voice' && (hrmTyped.confidence || 0) > 0.7,
      text: hrmTyped.reasoning_result,
      metadata: {
        l_module_activations: hrmTyped.l_module_activations,
        h_module_activations: hrmTyped.h_module_activations,
      },
    };
  }

  private async executeAgentWithHRMGuidance(agent: SEALAgent, input: unknown, context: unknown, hrmOutput: unknown): Promise<unknown> {
    agent.lastActive = new Date();
    agent.performance.totalInvocations++;

    // Enhance agent processing with HRM insights
    const hrmAny = hrmOutput as { reasoning_result?: unknown; confidence?: number; l_module_activations?: unknown; h_module_activations?: unknown };
    const enhancedContext = {
      ...(typeof context === 'object' && context !== null ? context as Record<string, unknown> : {}),
      hrmReasoning: hrmAny.reasoning_result,
      hrmConfidence: hrmAny.confidence,
      hrmActivations: {
        l_modules: hrmAny.l_module_activations,
        h_modules: hrmAny.h_module_activations,
      },
    };

    const response = await this.simulateAgentProcessing(agent, input, enhancedContext);

    // Merge HRM insights with agent response
    return {
      ...(response as Record<string, unknown>),
      hrmEnhanced: true,
      hrmConfidence: hrmAny.confidence,
      combinedConfidence: (((response as { confidence?: number }).confidence || 0) + (hrmAny.confidence || 0)) / 2,
      hrmReasoning: hrmAny.reasoning_result,
    };
  }

  private async selectAgent(input: unknown, context: unknown): Promise<SEALAgent | null> {
    const requiredCapabilities = this.analyzeRequiredCapabilities(input, context);

    let bestAgent: SEALAgent | null = null;
    let bestScore = 0;

    this.agents.forEach((agent) => {
      const score = this.calculateAgentScore(agent, requiredCapabilities);

      if (score > bestScore) {
        bestScore = score;
        bestAgent = agent;
      }
    });

    return bestAgent;
  }

  private analyzeRequiredCapabilities(input: unknown, context: unknown): string[] {
    const capabilities: string[] = [];

    // Analyze input type and content to determine required capabilities
    if (typeof input === 'string') {
      if (input.includes('code') || input.includes('function')) {
        capabilities.push('code_generation', 'debugging');
      } else {
        capabilities.push('text_analysis', 'summarization');
      }
    }

    // Add context-based capabilities
    const contextAny = context as { inputType?: string };
    if (contextAny.inputType === 'voice') {
      capabilities.push('speech_processing');
    }

    if (contextAny.inputType === 'visual') {
      capabilities.push('image_analysis', 'object_detection');
    }

    return capabilities;
  }

  private calculateAgentScore(agent: SEALAgent, requiredCapabilities: string[]): number {
    let score = 0;

    // Capability match score
    const matchingCapabilities = agent.capabilities.filter(cap =>
      requiredCapabilities.includes(cap)
    );
    score += matchingCapabilities.length * 10;

    // Performance score
    score += agent.performance.successRate * 5;
    score -= agent.performance.errorCount * 2;

    // Recency score (prefer recently active agents)
    const hoursSinceActive = (Date.now() - agent.lastActive.getTime()) / (1000 * 60 * 60);
    score += Math.max(0, 5 - hoursSinceActive);

    return score;
  }

  private async executeWithAgent(agent: SEALAgent, input: unknown, context: unknown): Promise<unknown> {
    agent.lastActive = new Date();
    agent.performance.totalInvocations++;

    // Simulate agent processing
    // In a real implementation, this would call actual AI models or services
    const response = await this.simulateAgentProcessing(agent, input, context);

    return response;
  }

  private async simulateAgentProcessing(agent: SEALAgent, input: unknown, context: unknown): Promise<unknown> {
    // Simulate processing delay
    await new Promise(resolve => setTimeout(resolve, Math.random() * 1000 + 500));

    // Generate mock response based on agent type
    switch (agent.type) {
      case 'text_processor':
        return {
          type: 'text_response',
          content: `Processed text input: ${JSON.stringify(input)}`,
          confidence: 0.85,
          shouldSpeak: (context as { inputType?: string }).inputType === 'voice',
          text: `I've analyzed your input and found relevant information.`,
        };

      case 'code_assistant':
        return {
          type: 'code_response',
          content: `Generated code solution for: ${JSON.stringify(input)}`,
          code: '// Generated code would be here\nfunction solution() {\n  return "implemented";\n}',
          confidence: 0.90,
          text: `I've generated a code solution for your request.`,
        };

      case 'problem_solver':
        return {
          type: 'solution_response',
          content: `Analyzed problem and found solution: ${JSON.stringify(input)}`,
          steps: ['Identify the problem', 'Analyze constraints', 'Generate solution'],
          confidence: 0.80,
          text: `I've broken down the problem into manageable steps.`,
        };

      case 'visual_analyzer':
        return {
          type: 'visual_response',
          content: `Analyzed visual input: ${JSON.stringify(input)}`,
          objects: ['detected objects would be here'],
          confidence: 0.75,
          text: `I've identified several objects in the visual input.`,
        };

      case 'voice_handler':
        return {
          type: 'voice_response',
          content: `Processed voice input: ${JSON.stringify(input)}`,
          command: 'parsed command would be here',
          confidence: 0.88,
          shouldSpeak: true,
          text: `I understand your voice command.`,
        };

      default:
        return {
          type: 'generic_response',
          content: `Processed by ${agent.type}: ${JSON.stringify(input)}`,
          confidence: 0.70,
          text: `I've processed your request using the ${agent.type} agent.`,
        };
    }
  }

  public async invokeSkill(skillId: string, parameters: unknown): Promise<unknown> {
    const invocationId = `invocation_${Date.now()}`;

    const invocation: SkillInvocation = {
      skillId,
      parameters,
      startTime: new Date(),
    };

    this.activeInvocations.set(invocationId, invocation);

    try {
      // Find agent capable of handling this skill
      const agent = await this.findSkillAgent(skillId);

      if (agent) {
        invocation.agent = agent;
      }

      // Execute skill (this would integrate with KNIRVCHAIN)
      const result = await this.executeSkill(skillId, parameters, agent || undefined);

      invocation.result = result;
      invocation.endTime = new Date();

      if (agent) {
        this.updateAgentPerformance(agent,
          invocation.endTime.getTime() - invocation.startTime.getTime(),
          true
        );
      }

      this.activeInvocations.delete(invocationId);
      return result;

    } catch (error) {
      invocation.error = error instanceof Error ? error.message : String(error);
      invocation.endTime = new Date();

      if (invocation.agent) {
        this.updateAgentPerformance(invocation.agent,
          invocation.endTime.getTime() - invocation.startTime.getTime(),
          false
        );
      }

      this.activeInvocations.delete(invocationId);
      throw error;
    }
  }

  private async findSkillAgent(skillId: string): Promise<SEALAgent | null> {
    // Find agent with capabilities matching the skill
    let foundAgent: SEALAgent | null = null;
    this.agents.forEach((agent) => {
      if (!foundAgent && agent.capabilities.some(cap => skillId.includes(cap))) {
        foundAgent = agent;
      }
    });
    if (foundAgent) return foundAgent;

    return null;
  }

  private async executeSkill(skillId: string, parameters: unknown, agent?: SEALAgent): Promise<unknown> {
    console.log(`Executing skill: ${skillId}`, parameters);

    // Simulate skill execution
    await new Promise(resolve => setTimeout(resolve, Math.random() * 2000 + 1000));

    return {
      skillId,
      result: `Skill ${skillId} executed successfully`,
      parameters,
      executedBy: agent?.id || 'unknown',
      timestamp: new Date(),
    };
  }

  public async generateAdaptation(learningHistory: unknown[]): Promise<unknown> {
    if (!this.learningMode) {
      return null;
    }

    console.log('Generating adaptation from learning history...');

    // Analyze learning history to generate adaptation
    interface AdaptationChange {
      type: string;
      patterns: unknown[];
      adjustments: unknown;
    }

    const adaptation: {
      type: string;
      changes: AdaptationChange[];
      confidence: number;
      timestamp: Date;
    } = {
      type: 'performance_improvement',
      changes: [],
      confidence: 0.75,
      timestamp: new Date(),
    };

    // Analyze patterns in learning history
    const errorPatterns = this.analyzeErrorPatterns(learningHistory);
    const successPatterns = this.analyzeSuccessPatterns(learningHistory);

    // Generate adaptation changes
    if (errorPatterns.length > 0) {
      adaptation.changes.push({
        type: 'error_reduction',
        patterns: errorPatterns,
        adjustments: this.generateErrorAdjustments(errorPatterns),
      });
    }

    if (successPatterns.length > 0) {
      adaptation.changes.push({
        type: 'success_amplification',
        patterns: successPatterns,
        adjustments: this.generateSuccessAdjustments(successPatterns),
      });
    }

    this.emit('adaptationGenerated', adaptation);
    return adaptation;
  }

  private analyzeErrorPatterns(history: unknown[]): unknown[] {
    return history
      .filter(event => ((event as { feedback?: number }).feedback || 0) < 0)
      .map(event => ({
        inputType: (event as { eventType?: string }).eventType,
        input: (event as { input?: unknown }).input,
        output: (event as { output?: unknown }).output,
        feedback: (event as { feedback?: number }).feedback,
      }));
  }

  private analyzeSuccessPatterns(history: unknown[]): unknown[] {
    return history
      .filter(event => ((event as { feedback?: number }).feedback || 0) > 0.5)
      .map(event => ({
        inputType: (event as { eventType?: string }).eventType,
        input: (event as { input?: unknown }).input,
        output: (event as { output?: unknown }).output,
        feedback: (event as { feedback?: number }).feedback,
      }));
  }

  private generateErrorAdjustments(patterns: unknown[]): unknown[] {
    return patterns.map(pattern => ({
      target: (pattern as { inputType?: string }).inputType,
      adjustment: 'reduce_confidence',
      magnitude: Math.abs((pattern as { feedback?: number }).feedback || 0) * 0.1,
    }));
  }

  private generateSuccessAdjustments(patterns: unknown[]): unknown[] {
    return patterns.map(pattern => ({
      target: (pattern as { inputType?: string }).inputType,
      adjustment: 'increase_confidence',
      magnitude: ((pattern as { feedback?: number }).feedback || 0) * 0.1,
    }));
  }

  public async enableLearningMode(): Promise<void> {
    this.learningMode = true;
    console.log('SEAL learning mode enabled');
    this.emit('learningModeEnabled');
  }

  public async disableLearningMode(): Promise<void> {
    this.learningMode = false;
    console.log('SEAL learning mode disabled');
    this.emit('learningModeDisabled');
  }

  private updateAgentPerformance(agent: SEALAgent, latency: number, success: boolean): void {
    const perf = agent.performance;

    // Update success rate
    const totalAttempts = perf.totalInvocations;
    const previousSuccesses = perf.successRate * (totalAttempts - 1);
    perf.successRate = (previousSuccesses + (success ? 1 : 0)) / totalAttempts;

    // Update average latency
    perf.averageLatency = ((perf.averageLatency * (totalAttempts - 1)) + latency) / totalAttempts;

    // Update error count
    if (!success) {
      perf.errorCount++;
    }

    this.emit('agentPerformanceUpdated', {
      agentId: agent.id,
      performance: perf,
    });
  }

  private async cancelInvocation(invocationId: string): Promise<void> {
    const invocation = this.activeInvocations.get(invocationId);
    if (invocation) {
      invocation.error = 'Cancelled';
      invocation.endTime = new Date();
      this.activeInvocations.delete(invocationId);
    }
  }

  public getAgents(): SEALAgent[] {
    return Array.from(this.agents.values());
  }

  public getActiveInvocations(): SkillInvocation[] {
    return Array.from(this.activeInvocations.values());
  }

  public getMetrics(): unknown {
    const agents = Array.from(this.agents.values());

    return {
      totalAgents: agents.length,
      activeInvocations: this.activeInvocations.size,
      learningMode: this.learningMode,
      hrmIntegration: this.config.hrmIntegration,
      hrmReady: this.hrmBridge ? (this.hrmBridge as { isReady?: () => boolean }).isReady?.() : false,
      averageSuccessRate: agents.length > 0 ? agents.reduce((sum, agent) => sum + agent.performance.successRate, 0) / agents.length : 0,
      totalInvocations: agents.reduce((sum, agent) => sum + agent.performance.totalInvocations, 0),
    };
  }

  // HRM Integration methods
  public setHRMBridge(hrmBridge: unknown): void {
    this.hrmBridge = hrmBridge;
    console.log('HRM bridge injected into SEAL Framework');
  }

  public enableHRMIntegration(): void {
    this.config.hrmIntegration = true;
    console.log('HRM integration enabled in SEAL Framework');
  }

  public disableHRMIntegration(): void {
    this.config.hrmIntegration = false;
    console.log('HRM integration disabled in SEAL Framework');
  }

  public isHRMIntegrationEnabled(): boolean {
    return this.config.hrmIntegration === true;
  }

  public getHRMStatus(): unknown {
    return {
      enabled: this.config.hrmIntegration,
      bridgeAvailable: this.hrmBridge !== null,
      ready: this.hrmBridge ? (this.hrmBridge as { isReady?: () => boolean }).isReady?.() : false,
    };
  }

  public dispose(): void {
    try {
      // Stop the framework
      this.stop();

      // Clear all agents
      this.agents.clear();

      // Clear active invocations
      this.activeInvocations.clear();

      // Clear HRM bridge
      this.hrmBridge = null;

      // Remove all event listeners
      this.removeAllListeners();

      console.log('SEALFramework disposed successfully');
    } catch (error) {
      console.error('Error during SEALFramework disposal:', error);
    }
  }
}
