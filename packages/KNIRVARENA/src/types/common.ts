/**
 * Common TypeScript interfaces to replace 'any' types throughout the codebase
 * These interfaces provide proper type safety for commonly used objects
 */

// Base interfaces for tool system
export interface ToolParameter {
  name: string;
  type: 'string' | 'number' | 'boolean' | 'object' | 'array';
  required: boolean;
  description: string;
  defaultValue?: unknown;
}

export interface ToolContext {
  agentId: string;
  sessionId?: string;
  userId?: string;
  environment: 'wasm' | 'browser' | 'node';
  memory: Map<string, unknown>;
  logger: {
    log: (message: string) => void;
    error: (message: string) => void;
    warn: (message: string) => void;
  };
}

export interface ToolResult {
  success: boolean;
  result?: unknown;
  error?: string;
  executionTime: number;
  metadata?: Record<string, unknown>;
}

// Memory and storage interfaces
export interface MemoryEntry {
  key: string;
  value: unknown;
  timestamp: Date;
  type: 'temporary' | 'persistent' | 'session';
  metadata?: Record<string, unknown>;
}

export interface StorageAdapter {
  get(key: string): Promise<unknown>;
  set(key: string, value: unknown): Promise<void>;
  delete(key: string): Promise<boolean>;
  clear(): Promise<void>;
  keys(): Promise<string[]>;
}

// Event system interfaces
export interface EventPayload {
  type: string;
  data: unknown;
  timestamp: Date;
  source: string;
  metadata?: Record<string, unknown>;
}

export interface EventHandler {
  (payload: EventPayload): void | Promise<void>;
}

export interface EventEmitterInterface {
  on(event: string, handler: EventHandler): void;
  off(event: string, handler: EventHandler): void;
  emit(event: string, payload: EventPayload): void;
}

// API and network interfaces
export interface ApiResponse<T = unknown> {
  success: boolean;
  data?: T;
  error?: string;
  statusCode: number;
  headers?: Record<string, string>;
  metadata?: Record<string, unknown>;
}

export interface ApiRequest {
  method: 'GET' | 'POST' | 'PUT' | 'DELETE' | 'PATCH';
  url: string;
  headers?: Record<string, string>;
  body?: unknown;
  timeout?: number;
  retries?: number;
}

export interface NetworkConfig {
  baseUrl: string;
  timeout: number;
  retries: number;
  headers: Record<string, string>;
  interceptors?: {
    request?: (config: ApiRequest) => ApiRequest;
    response?: (response: ApiResponse) => ApiResponse;
  };
}

// Configuration interfaces
export interface ConfigurationValue {
  value: unknown;
  type: 'string' | 'number' | 'boolean' | 'object' | 'array';
  required: boolean;
  description?: string;
  defaultValue?: unknown;
  validation?: {
    min?: number;
    max?: number;
    pattern?: string;
    enum?: unknown[];
  };
}

export interface ConfigurationSchema {
  [key: string]: ConfigurationValue;
}

export interface RuntimeConfiguration {
  environment: 'development' | 'staging' | 'production';
  debug: boolean;
  features: Record<string, boolean>;
  services: Record<string, ServiceConfig>;
  security: SecurityConfig;
}

export interface ServiceConfig {
  enabled: boolean;
  endpoint?: string;
  timeout?: number;
  retries?: number;
  config?: Record<string, unknown>;
}

export interface SecurityConfig {
  encryption: {
    algorithm: string;
    keySize: number;
  };
  authentication: {
    provider: string;
    config: Record<string, unknown>;
  };
  authorization: {
    roles: string[];
    permissions: Record<string, string[]>;
  };
}

// Data processing interfaces
export interface ProcessingResult<T = unknown> {
  success: boolean;
  data?: T;
  error?: string;
  processingTime: number;
  metadata?: {
    inputSize?: number;
    outputSize?: number;
    transformations?: string[];
    warnings?: string[];
  };
}

export interface DataTransformer<TInput = unknown, TOutput = unknown> {
  transform(input: TInput): Promise<ProcessingResult<TOutput>>;
  validate(input: TInput): boolean;
  getSchema(): Record<string, unknown>;
}

export interface ValidationRule {
  field: string;
  type: 'required' | 'type' | 'range' | 'pattern' | 'custom';
  value?: unknown;
  message: string;
  validator?: (value: unknown) => boolean;
}

export interface ValidationResult {
  valid: boolean;
  errors: Array<{
    field: string;
    message: string;
    value?: unknown;
  }>;
}

// Skill and adaptation interfaces
export interface SkillMetadata {
  id: string;
  name: string;
  description: string;
  version: string;
  author: string;
  tags: string[];
  capabilities: string[];
  requirements: Record<string, unknown>;
  compatibility: {
    models: string[];
    environments: string[];
  };
}

export interface AdaptationContext {
  agentId: string;
  taskType: string;
  environment: Record<string, unknown>;
  constraints: Record<string, unknown>;
  objectives: string[];
  feedback?: FeedbackData;
}

export interface FeedbackData {
  type: 'positive' | 'negative' | 'neutral';
  score: number;
  details: Record<string, unknown>;
  timestamp: Date;
  source: string;
}

export interface LearningMetrics {
  accuracy: number;
  precision: number;
  recall: number;
  f1Score: number;
  learningRate: number;
  epochs: number;
  convergence: boolean;
  metadata?: Record<string, unknown>;
}

// Error handling interfaces
export interface ErrorContext {
  errorType: string;
  message: string;
  stack?: string;
  timestamp: Date;
  source: string;
  severity: 'low' | 'medium' | 'high' | 'critical';
  metadata?: Record<string, unknown>;
  recovery?: {
    attempted: boolean;
    successful: boolean;
    strategy: string;
  };
}

export interface ErrorHandler {
  canHandle(error: ErrorContext): boolean;
  handle(error: ErrorContext): Promise<ProcessingResult>;
  getPriority(): number;
}

// Generic utility types
export type DeepPartial<T> = {
  [P in keyof T]?: T[P] extends object ? DeepPartial<T[P]> : T[P];
};

export type RequiredFields<T, K extends keyof T> = T & Required<Pick<T, K>>;

export type OptionalFields<T, K extends keyof T> = Omit<T, K> & Partial<Pick<T, K>>;

export interface Serializable {
  serialize(): string;
  deserialize(data: string): void;
}

export interface Disposable {
  dispose(): void | Promise<void>;
}

export interface Initializable {
  initialize(): Promise<void>;
  isInitialized(): boolean;
}

// Type guards
export function isApiResponse(obj: unknown): obj is ApiResponse {
  return typeof obj === 'object' && obj !== null && 'success' in obj;
}

export function isToolResult(obj: unknown): obj is ToolResult {
  return typeof obj === 'object' && obj !== null && 'success' in obj && 'executionTime' in obj;
}

export function isErrorContext(obj: unknown): obj is ErrorContext {
  return typeof obj === 'object' && obj !== null && 'errorType' in obj && 'severity' in obj;
}

// Unified Agent interface - consolidates different Agent definitions across the codebase
export interface Agent {
  agentId: string;
  name: string;
  version: string;
  baseModelId?: string;
  type: 'wasm' | 'lora' | 'hybrid';
  status: 'Available' | 'Deployed' | 'Error' | 'Compiling';
  nrnCost: number;
  capabilities: string[];
  metadata: AgentMetadata;
  wasmModule?: string;
  loraAdapter?: string;
  createdAt: string;
  lastActivity?: string;
}

export interface AgentMetadata {
  name: string;
  version: string;
  baseModelId?: string;
  description: string;
  author: string;
  capabilities: string[];
  requirements: {
    memory: number;
    cpu: number;
    storage: number;
  };
  permissions: string[];
}

// Legacy Agent interface for backward compatibility (App.tsx style)
export interface LegacyAgent {
  id: string;
  name: string;
  type: 'wasm' | 'lora' | 'hybrid';
  status: 'Available' | 'Deployed' | 'Error' | 'Compiling';
  nrnCost: number;
  capabilities: string[];
  metadata: {
    name: string;
    version: string;
    description: string;
    author: string;
    capabilities: string[];
    requirements: {
      memory: number;
      cpu: number;
      storage: number;
    };
    permissions: string[];
  };
  wasmModule?: WebAssembly.Module;
  lastActivity?: Date;
}
