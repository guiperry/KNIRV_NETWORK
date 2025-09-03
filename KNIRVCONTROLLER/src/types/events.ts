/**
 * Event type definitions for KNIRV Controller
 * Replaces 'any' types with proper TypeScript interfaces
 */

// Base event data structures
export interface BaseEventData {
  timestamp: Date;
  source?: string;
  sessionId?: string;
  userId?: string;
}

// Input processing events
export interface InputProcessedEventData extends BaseEventData {
  inputType: 'text' | 'voice' | 'visual' | 'gesture' | 'multimodal';
  processingTime: number;
  response: CognitiveResponse;
}

export interface CognitiveResponse {
  type?: string;
  text?: string;
  confidence?: number;
  source?: string;
  metadata?: Record<string, unknown>;
  shouldSpeak?: boolean;
}

// Skill invocation events
export interface SkillInvokedEventData extends BaseEventData {
  skillId: string;
  parameters: Record<string, unknown>;
  result: SkillExecutionResult;
}

export interface SkillExecutionResult {
  success: boolean;
  result?: unknown;
  output?: unknown;
  error?: string;
  executionTime?: number;
  resourceUsage?: {
    memory: number;
    cpu: number;
  };
}

// Processing error events
export interface ProcessingErrorEventData extends BaseEventData {
  inputType: string;
  error: string;
  stack?: string;
}

// LoRA events
export interface LoRATrainingUpdateEventData extends BaseEventData {
  epoch: number;
  loss: number;
  accuracy?: number;
  learningRate: number;
  batchSize: number;
}

export interface LoRABatchCompleteEventData extends BaseEventData {
  batchId: string;
  totalBatches: number;
  completedBatches: number;
  averageLoss: number;
  trainingTime: number;
}

// Visual processing events
export interface VisualImageProcessedEventData extends BaseEventData {
  imageData: ArrayBuffer;
  detectedObjects?: DetectedObject[];
  sceneAnalysis?: SceneAnalysis;
  ocrResults?: OCRResult[];
}

export interface DetectedObject {
  label: string;
  confidence: number;
  boundingBox: BoundingBox;
  attributes?: Record<string, unknown>;
}

export interface BoundingBox {
  x: number;
  y: number;
  width: number;
  height: number;
}

export interface SceneAnalysis {
  sceneType: string;
  confidence: number;
  objects: DetectedObject[];
  lighting: 'bright' | 'dim' | 'natural' | 'artificial';
  setting: 'indoor' | 'outdoor' | 'unknown';
  mood: string;
  complexity: number;
}

export interface OCRResult {
  text: string;
  confidence: number;
  boundingBox: BoundingBox;
  language: string;
}

// Ecosystem communication events
export interface EcosystemComponentEventData extends BaseEventData {
  id: string;
  name: string;
  status: 'online' | 'offline' | 'error' | 'connecting';
  version?: string;
  capabilities?: string[];
  metrics?: Record<string, unknown>;
}

export interface EcosystemConnectionEventData extends BaseEventData {
  endpoint: ServiceEndpoint;
  success: boolean;
  error?: string;
}

export interface ServiceEndpoint {
  id: string;
  name: string;
  url: string;
  protocol: 'http' | 'websocket' | 'p2p';
  authentication: {
    type: 'none' | 'bearer' | 'api_key' | 'certificate';
    credentials?: unknown;
  };
  healthCheckPath?: string;
  capabilities: string[];
}

export interface EcosystemMessageEventData extends BaseEventData {
  id: string;
  from: string;
  to: string;
  type: 'command' | 'query' | 'response' | 'event' | 'heartbeat';
  payload: unknown;
  priority: 'low' | 'normal' | 'high' | 'critical';
  requiresResponse: boolean;
  correlationId?: string;
}

// HRM events
export interface HRMProcessedEventData extends BaseEventData {
  input: HRMCognitiveInput;
  output: HRMCognitiveOutput;
  processingTime: number;
}

export interface HRMCognitiveInput {
  text?: string;
  audio?: ArrayBuffer;
  visual?: ArrayBuffer;
  context?: Record<string, unknown>;
  confidenceLevel?: number;
}

export interface HRMCognitiveOutput {
  reasoning_result: string;
  confidence: number;
  processing_time: number;
  l_module_activations?: number[];
  h_module_activations?: number[];
}

// Wallet integration events
export interface WalletEventData extends BaseEventData {
  account?: WalletAccount;
  transaction?: TransactionRequest;
  balance?: string;
  error?: string;
}

export interface WalletAccount {
  id: string;
  address: string;
  name: string;
  balance: string;
  nrnBalance: string;
  isActive: boolean;
  keyringType: 'hd' | 'private' | 'ledger' | 'web3auth';
}

export interface TransactionRequest {
  from: string;
  to: string;
  amount: string;
  token?: string;
  memo?: string;
  gasLimit?: string;
  chainId?: string;
  skillId?: string;
  nrnAmount?: string;
}

// KNIRV Chain events
export interface ChainEventData extends BaseEventData {
  blockHeight?: number;
  transactionHash?: string;
  skillInvocation?: SkillInvocation;
  consensus?: NetworkConsensus;
}

export interface SkillInvocation {
  skillId: string;
  user: string;
  amountBurned: string;
  timestamp: number;
  success: boolean;
  resultHash?: string;
  transactionHash: string;
}

export interface NetworkConsensus {
  blockHeight: number;
  blockHash: string;
  validators: string[];
  consensusReached: boolean;
  timestamp: number;
}

// Agent and SEAL Framework events
export interface AgentEventData extends BaseEventData {
  agentId: string;
  agentType: string;
  status: 'created' | 'active' | 'idle' | 'disposed';
  capabilities?: string[];
  metadata?: Record<string, unknown>;
}

export interface AdaptationEventData extends BaseEventData {
  adaptationType: string;
  weights?: number[];
  metrics?: Record<string, number>;
  success: boolean;
}

// Generic cognitive event
export interface CognitiveEventData extends BaseEventData {
  type: string;
  data: unknown;
}

// Memory and performance events
export interface MemoryEventData extends BaseEventData {
  usedJSHeapSize: number;
  totalJSHeapSize: number;
  jsHeapSizeLimit: number;
  usagePercentage: number;
}

export interface PerformanceEventData extends BaseEventData {
  metric: string;
  value: number;
  unit: string;
  threshold?: number;
}

// Metrics interfaces
export interface CognitiveMetrics {
  confidenceLevel: number;
  adaptationLevel: number;
  activeSkills: number;
  learningEvents: number;
  contextSize: number;
  isRunning: boolean;
}

export interface VisualProcessorMetrics {
  isSupported: boolean;
  resolution: string;
  frameRate: number;
  objectDetection: boolean;
  gestureRecognition: boolean;
  ocrEnabled: boolean;
  faceRecognition: boolean;
}

export interface LoRAConfig {
  taskType: string;
  rank: number;
  alpha: number;
  dropout: number;
  targetModules: string[];
}

export interface LoRAMetrics {
  epoch: number;
  loss: number;
  accuracy: number;
  learningRate: number;
}

export interface TrainingData {
  input: {
    text: string;
    features: number[];
  };
  output: {
    text: string;
    confidence: number;
  };
  feedback: number;
  timestamp: Date;
}

export interface AdaptedOutput {
  confidence?: number;
  text?: string;
  metadata?: Record<string, unknown>;
}

// Visual processor interfaces
export interface VisualProcessor {
  getMetrics(): VisualProcessorMetrics;
  updateConfig(config: Partial<{ frameRate: number }>): void;
  emit(event: string, data: unknown): void;
}

export interface LoRAAdapter {
  getConfig(): LoRAConfig;
  getMetrics(): LoRAMetrics;
  getTrainingDataSize(): number;
  enableTraining(): void;
  addTrainingData(data: TrainingData): Promise<void>;
  trainOnBatch(data: TrainingData[]): Promise<void>;
  adapt(input: unknown, expected: unknown, feedback: number): Promise<AdaptedOutput>;
  exportWeights(): Map<string, unknown>;
  importWeights(weights: Map<string, unknown>): void;
}
