/**
 * KNIRV Gateway SDK Types and Interfaces
 */

// Base configuration interface
export interface ClientOptions {
  baseURL?: string;
  baseUrl?: string; // Alias for compatibility
  economicsURL?: string;
  apiKey?: string;
  nrnContract?: string;
  timeout?: number;
  retries?: number;
  retryDelay?: number;
  maxRetryDelay?: number;
  userAgent?: string;
  defaultHeaders?: Record<string, string>;
}

// Request configuration
export interface RequestConfig {
  method?: 'GET' | 'POST' | 'PUT' | 'DELETE' | 'PATCH';
  url?: string;
  headers?: Record<string, string>;
  params?: Record<string, any>;
  data?: any;
  timeout?: number;
  signal?: AbortSignal;
}

// Response interface
export interface APIResponse<T = any> {
  data: T;
  status: number;
  statusText: string;
  headers: Record<string, string>;
}

// Exception classes
export class KNIRVAPIError extends Error {
  public status?: number;
  public response?: any;
  public request?: any;

  constructor(message: string, status?: number, response?: any, request?: any) {
    super(message);
    this.name = 'KNIRVAPIError';
    this.status = status;
    this.response = response;
    this.request = request;
  }
}

export class KNIRVValidationError extends KNIRVAPIError {
  public field?: string;

  constructor(message: string, fieldOrStatus?: string | number, response?: any) {
    super(message);
    this.name = 'KNIRVValidationError';

    // Handle different constructor signatures for compatibility
    if (typeof fieldOrStatus === 'string') {
      this.field = fieldOrStatus;
    } else if (typeof fieldOrStatus === 'number') {
      this.status = fieldOrStatus;
      this.response = response;
    }
  }
}

export class KNIRVConnectionError extends KNIRVAPIError {
  constructor(message: string = 'Connection error') {
    super(message);
    this.name = 'KNIRVConnectionError';
  }
}

export class KNIRVTimeoutError extends KNIRVAPIError {
  constructor(message: string = 'Request timeout') {
    super(message);
    this.name = 'KNIRVTimeoutError';
  }
}

// Skill-related types
export interface Skill {
  id: string;
  name: string;
  description?: string;
  capabilities: string[];
  category?: string;
  cost?: number;
  metadata?: Record<string, any>;
  created_at?: string;
  updated_at?: string;
}

export interface SkillCreateRequest {
  name?: string; // Made optional for test compatibility
  description?: string;
  capabilities?: string[]; // Made optional for test compatibility
  category?: string;
  cost?: number;
  metadata?: Record<string, any>;
}

export interface SkillUpdateRequest {
  name?: string;
  description?: string;
  capabilities?: string[];
  category?: string;
  cost?: number;
  metadata?: Record<string, any>;
}

export interface SkillSearchRequest {
  query?: string;
  category?: string;
  capabilities?: string[];
  limit?: number;
  offset?: number;
}

// LLM-related types
export interface LLMModel {
  id: string;
  name: string;
  provider: string;
  capabilities: string[];
  cost_per_token?: number;
  max_tokens?: number;
}

export interface LLMUsage {
  total_tokens: number;
  total_cost: number;
  period: string;
  breakdown?: Record<string, any>;
}

export interface LLMCostEstimate {
  estimated_tokens: number;
  estimated_cost: number;
  model: string;
  breakdown?: Record<string, any>;
}

export interface LLMCostEstimateRequest {
  model: string;
  prompt?: string;
  text?: string; // Alias for prompt
  max_tokens?: number;
  messages?: Array<{ role: string; content: string }>;
}

// Validation-related types
export interface ValidationRequest {
  type?: string;
  skill_id?: string; // For skill validation
  data: any;
  rules?: string[];
}

export interface ValidationResponse {
  valid: boolean;
  errors?: Array<{
    field: string;
    message: string;
    code?: string;
  }>;
}

export interface ValidationRule {
  id: string;
  name: string;
  description?: string;
  category?: string;
  rule_type: string;
  parameters?: Record<string, any>;
}

// Pagination types
export interface PaginationOptions {
  page?: number;
  per_page?: number;
  limit?: number;
  offset?: number;
}

export interface PaginatedResponse<T> {
  data: T[];
  pagination: {
    page: number;
    per_page: number;
    total: number;
    total_pages: number;
  };
}

// Generic list response
export interface ListResponse<T> {
  items?: T[];
  data?: T[];
  total?: number;
  page?: number;
  per_page?: number;
}

// Request interceptor types
export type RequestInterceptor = (config: RequestConfig) => RequestConfig | Promise<RequestConfig>;
export type ResponseInterceptor = (response: APIResponse) => APIResponse | Promise<APIResponse>;

// Health check types
export interface HealthStatus {
  status: 'healthy' | 'unhealthy' | 'degraded';
  timestamp: string;
  services?: Record<string, any>;
}

// Gateway types
export interface GatewayRoute {
  id: string;
  path: string;
  method: string;
  target: string;
  enabled: boolean;
}

// Integration types
export interface Integration {
  id: string;
  name: string;
  type: string;
  config: Record<string, any>;
  enabled: boolean;
}

// PoAuD types
export interface Proof {
  id: string;
  skill_id: string;
  user_id: string;
  proof_data: any;
  status: 'pending' | 'verified' | 'rejected';
  created_at: string;
}

export interface Challenge {
  id: string;
  title: string;
  description: string;
  difficulty: 'easy' | 'medium' | 'hard';
  reward: number;
  status: 'active' | 'completed' | 'expired';
}

export interface Reputation {
  user_id: string;
  score: number;
  rank: number;
  badges: string[];
}
