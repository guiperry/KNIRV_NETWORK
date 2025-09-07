/**
 * Cognitive Engine Service
 * Handles AI processing, skill execution, and cognitive operations
 */

export interface CognitiveProcessingRequest {
  input: string;
  context?: Record<string, unknown>;
  taskType: 'conversation' | 'analysis' | 'skill_execution' | 'learning';
  requiresSkillInvocation?: boolean;
  metadata?: Record<string, unknown>;
}

export interface CognitiveProcessingResult {
  output: string;
  confidence: number;
  skillsInvoked: string[];
  processingTime: number;
  contextUpdates?: Record<string, unknown>;
  adaptationTriggered?: boolean;
}

export interface SkillExecutionRequest {
  skillId: string;
  parameters: Record<string, unknown>;
  context?: Record<string, unknown>;
  timeout?: number;
}

export interface SkillExecutionResult {
  success: boolean;
  output: unknown;
  error?: string;
  executionTime: number;
  resourceUsage: {
    memory: number;
    cpu: number;
  };
}

export interface LearningEvent {
  id: string;
  type: 'adaptation' | 'skill_learning' | 'context_update';
  data: Record<string, unknown>;
  timestamp: Date;
  confidence: number;
}

export interface CognitiveMetrics {
  totalProcessingRequests: number;
  averageProcessingTime: number;
  skillInvocations: number;
  learningEvents: number;
  adaptationLevel: number;
  confidenceLevel: number;
  activeSkills: number;
  contextSize: number;
}

export class CognitiveEngineService {
  private isRunning: boolean = false;
  private context: Map<string, unknown> = new Map();
  private activeSkills: Set<string> = new Set();
  private learningEvents: LearningEvent[] = [];
  private metrics: CognitiveMetrics;
  private baseUrl: string;
  private wasmModule: WebAssembly.Module | null = null;
  private hrmBridge: unknown = null;

  constructor(baseUrl: string = 'http://localhost:3001') {
    this.baseUrl = baseUrl;
    this.metrics = this.initializeMetrics();
    this.initializeCognitiveEngine();
  }

  private initializeMetrics(): CognitiveMetrics {
    return {
      totalProcessingRequests: 0,
      averageProcessingTime: 0,
      skillInvocations: 0,
      learningEvents: 0,
      adaptationLevel: 0.75,
      confidenceLevel: 0.95,
      activeSkills: 0,
      contextSize: 0
    };
  }

  private async initializeCognitiveEngine(): Promise<void> {
    try {
      // Initialize WASM cognitive module
      await this.loadCognitiveWASM();
      
      // Initialize HRM bridge
      await this.initializeHRMBridge();
      
      console.log('Cognitive Engine initialized successfully');
    } catch (error) {
      console.error('Failed to initialize Cognitive Engine:', error);
    }
  }

  private async loadCognitiveWASM(): Promise<void> {
    try {
      const response = await fetch('/build/knirv-controller.wasm');
      const wasmBytes = await response.arrayBuffer();
      this.wasmModule = await WebAssembly.compile(wasmBytes);
      console.log('Cognitive WASM module loaded');
    } catch (error) {
      console.warn('Failed to load cognitive WASM module:', error);
    }
  }

  private async initializeHRMBridge(): Promise<void> {
    try {
      // Initialize connection to HRM (Hierarchical Reasoning Model)
      const response = await fetch(`${this.baseUrl}/api/cognitive/hrm/init`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          modelPath: '/models/hrm-core.wasm',
          config: {
            maxMemoryMB: 512,
            enableGPU: false,
            batchSize: 1,
            sequenceLength: 2048
          }
        })
      });

      if (response.ok) {
        this.hrmBridge = await response.json();
        console.log('HRM Bridge initialized');
      }
    } catch (error) {
      console.warn('HRM Bridge initialization failed:', error);
    }
  }

  /**
   * Start the cognitive engine
   */
  async start(): Promise<void> {
    if (this.isRunning) {
      return;
    }

    try {
      const response = await fetch(`${this.baseUrl}/api/cognitive/start`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        }
      });

      if (!response.ok) {
        throw new Error(`Failed to start cognitive engine: ${response.statusText}`);
      }

      this.isRunning = true;
      console.log('Cognitive Engine started');
    } catch (error) {
      console.error('Failed to start cognitive engine:', error);
      throw error;
    }
  }

  /**
   * Stop the cognitive engine
   */
  async stop(): Promise<void> {
    if (!this.isRunning) {
      return;
    }

    try {
      await fetch(`${this.baseUrl}/api/cognitive/stop`, {
        method: 'POST'
      });

      this.isRunning = false;
      console.log('Cognitive Engine stopped');
    } catch (error) {
      console.error('Failed to stop cognitive engine:', error);
      throw error;
    }
  }

  /**
   * Process cognitive input
   */
  async processInput(request: CognitiveProcessingRequest): Promise<CognitiveProcessingResult> {
    if (!this.isRunning) {
      throw new Error('Cognitive engine is not running');
    }

    const startTime = Date.now();
    this.metrics.totalProcessingRequests++;

    try {
      // Prepare processing request
      const processingRequest = {
        input: request.input,
        context: Object.fromEntries(this.context),
        taskType: request.taskType,
        requiresSkillInvocation: request.requiresSkillInvocation,
        metadata: request.metadata,
        activeSkills: Array.from(this.activeSkills)
      };

      // Send to cognitive processing endpoint
      const response = await fetch(`${this.baseUrl}/api/cognitive/process`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(processingRequest)
      });

      if (!response.ok) {
        throw new Error(`Cognitive processing failed: ${response.statusText}`);
      }

      const result = await response.json();
      const processingTime = Date.now() - startTime;

      // Update metrics
      this.updateProcessingMetrics(processingTime);

      // Update context if provided
      if (result.contextUpdates) {
        Object.entries(result.contextUpdates).forEach(([key, value]) => {
          this.context.set(key, value);
        });
        this.metrics.contextSize = this.context.size;
      }

      // Handle skill invocations
      if (result.skillsInvoked && result.skillsInvoked.length > 0) {
        this.metrics.skillInvocations += result.skillsInvoked.length;
      }

      // Handle adaptation
      if (result.adaptationTriggered) {
        await this.handleAdaptation(result.adaptationData);
      }

      return {
        output: result.output || 'Processing completed',
        confidence: result.confidence || 0.95,
        skillsInvoked: result.skillsInvoked || [],
        processingTime,
        contextUpdates: result.contextUpdates,
        adaptationTriggered: result.adaptationTriggered
      };
    } catch (error) {
      const processingTime = Date.now() - startTime;
      this.updateProcessingMetrics(processingTime);
      
      console.error('Cognitive processing failed:', error);
      
      // Return fallback response
      return {
        output: `Error processing input: ${error instanceof Error ? error.message : 'Unknown error'}`,
        confidence: 0.1,
        skillsInvoked: [],
        processingTime
      };
    }
  }

  /**
   * Execute a specific skill
   */
  async executeSkill(request: SkillExecutionRequest): Promise<SkillExecutionResult> {
    if (!this.isRunning) {
      throw new Error('Cognitive engine is not running');
    }

    const startTime = Date.now();

    try {
      const response = await fetch(`${this.baseUrl}/api/cognitive/skills/${request.skillId}/execute`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          parameters: request.parameters,
          context: request.context,
          timeout: request.timeout || 30000
        })
      });

      if (!response.ok) {
        throw new Error(`Skill execution failed: ${response.statusText}`);
      }

      const result = await response.json();
      const executionTime = Date.now() - startTime;

      this.metrics.skillInvocations++;

      return {
        success: true,
        output: result.output,
        executionTime,
        resourceUsage: result.resourceUsage || { memory: 0, cpu: 0 }
      };
    } catch (error) {
      const executionTime = Date.now() - startTime;
      
      return {
        success: false,
        output: null,
        error: error instanceof Error ? error.message : 'Unknown error',
        executionTime,
        resourceUsage: { memory: 0, cpu: 0 }
      };
    }
  }

  /**
   * Activate a skill
   */
  async activateSkill(skillId: string): Promise<void> {
    try {
      const response = await fetch(`${this.baseUrl}/api/cognitive/skills/${skillId}/activate`, {
        method: 'POST'
      });

      if (response.ok) {
        this.activeSkills.add(skillId);
        this.metrics.activeSkills = this.activeSkills.size;
        console.log(`Skill ${skillId} activated`);
      }
    } catch (error) {
      console.error(`Failed to activate skill ${skillId}:`, error);
    }
  }

  /**
   * Deactivate a skill
   */
  async deactivateSkill(skillId: string): Promise<void> {
    try {
      const response = await fetch(`${this.baseUrl}/api/cognitive/skills/${skillId}/deactivate`, {
        method: 'POST'
      });

      if (response.ok) {
        this.activeSkills.delete(skillId);
        this.metrics.activeSkills = this.activeSkills.size;
        console.log(`Skill ${skillId} deactivated`);
      }
    } catch (error) {
      console.error(`Failed to deactivate skill ${skillId}:`, error);
    }
  }

  /**
   * Start learning mode
   */
  async startLearningMode(): Promise<void> {
    try {
      const response = await fetch(`${this.baseUrl}/api/cognitive/learning/start`, {
        method: 'POST'
      });

      if (response.ok) {
        console.log('Learning mode started');
      }
    } catch (error) {
      console.error('Failed to start learning mode:', error);
    }
  }

  /**
   * Save current adaptation
   */
  async saveCurrentAdaptation(): Promise<void> {
    try {
      const adaptationData = {
        context: Object.fromEntries(this.context),
        activeSkills: Array.from(this.activeSkills),
        metrics: this.metrics,
        timestamp: new Date()
      };

      const response = await fetch(`${this.baseUrl}/api/cognitive/adaptation/save`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(adaptationData)
      });

      if (response.ok) {
        console.log('Adaptation saved successfully');
      }
    } catch (error) {
      console.error('Failed to save adaptation:', error);
    }
  }

  /**
   * Get current metrics
   */
  getMetrics(): CognitiveMetrics {
    return { ...this.metrics };
  }

  /**
   * Get current context
   */
  getContext(): Map<string, unknown> {
    return new Map(this.context);
  }

  /**
   * Get active skills
   */
  getActiveSkills(): string[] {
    return Array.from(this.activeSkills);
  }

  /**
   * Check if engine is running
   */
  isEngineRunning(): boolean {
    return this.isRunning;
  }

  private updateProcessingMetrics(processingTime: number): void {
    const totalTime = this.metrics.averageProcessingTime * (this.metrics.totalProcessingRequests - 1) + processingTime;
    this.metrics.averageProcessingTime = totalTime / this.metrics.totalProcessingRequests;
  }

  private async handleAdaptation(adaptationData: Record<string, unknown>): Promise<void> {
    const learningEvent: LearningEvent = {
      id: `adaptation_${Date.now()}`,
      type: 'adaptation',
      data: adaptationData,
      timestamp: new Date(),
      confidence: (adaptationData.confidence as number) || 0.8
    };

    this.learningEvents.push(learningEvent);
    this.metrics.learningEvents++;
    this.metrics.adaptationLevel = Math.min(1.0, this.metrics.adaptationLevel + 0.01);

    console.log('Adaptation triggered:', learningEvent);
  }
}

// Export singleton instance
export const cognitiveEngineService = new CognitiveEngineService();
