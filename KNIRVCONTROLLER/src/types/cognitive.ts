// Cognitive engine types and interfaces
export interface CognitiveConfig {
  modelPath?: string;
  maxTokens?: number;
  temperature?: number;
  topP?: number;
  topK?: number;
  repetitionPenalty?: number;
  maxContextLength?: number;
  enableLoRA?: boolean;
  loraConfig?: {
    rank: number;
    alpha: number;
    dropout: number;
    targetModules: string[];
  };
  enableHRM?: boolean;
  hrmConfig?: {
    memorySize: number;
    learningRate: number;
    adaptationThreshold: number;
  };
  enableSEAL?: boolean;
  sealConfig?: {
    maxIterations: number;
    convergenceThreshold: number;
    feedbackWeight: number;
  };
}

export interface CognitiveState {
  status: 'idle' | 'processing' | 'learning' | 'adapting' | 'error';
  currentTask?: string;
  processingTime?: number;
  memoryUsage?: number;
  learningProgress?: number;
  adaptationLevel?: number;
  errorCount?: number;
  lastUpdate?: Date;
  metrics?: {
    accuracy: number;
    confidence: number;
    responseTime: number;
    memoryEfficiency: number;
  };
  activeSkills?: string[];
  pendingAdaptations?: string[];
  contextWindow?: {
    size: number;
    utilization: number;
  };
}

export interface CognitiveMetrics {
  processingTime: number;
  memoryUsage: number;
  accuracy: number;
  confidence: number;
  learningRate: number;
  adaptationCount: number;
  skillsActivated: number;
  contextUtilization: number;
  errorRate: number;
  timestamp: Date;
}

export interface CognitiveSkill {
  id: string;
  name: string;
  description: string;
  category: string;
  version: string;
  parameters: Record<string, unknown>;
  metadata: {
    accuracy: number;
    usageCount: number;
    lastUsed: Date;
    averageExecutionTime: number;
  };
  dependencies?: string[];
  loraAdaptations?: {
    rank: number;
    alpha: number;
    weights: Float32Array;
  };
}

export interface CognitiveAdaptation {
  id: string;
  type: 'lora' | 'hrm' | 'seal' | 'feedback';
  trigger: string;
  parameters: Record<string, unknown>;
  impact: {
    accuracy: number;
    performance: number;
    memoryUsage: number;
  };
  timestamp: Date;
  status: 'pending' | 'applied' | 'reverted' | 'failed';
}

export interface CognitiveContext {
  sessionId: string;
  userId?: string;
  conversationHistory: Array<{
    role: 'user' | 'assistant' | 'system';
    content: string;
    timestamp: Date;
    metadata?: Record<string, unknown>;
  }>;
  activeSkills: string[];
  memoryState: Record<string, unknown>;
  preferences: Record<string, unknown>;
  constraints: {
    maxTokens: number;
    timeoutMs: number;
    memoryLimitMB: number;
  };
}
