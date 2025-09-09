/**
 * TypeScript type definitions for KNIRVORACLE integration
 * Ensures type safety across all KNIRVORACLE API interactions
 */

// Base response interface for all KNIRVORACLE API calls
export interface KNIRVOracleBaseResponse {
  success: boolean;
  message: string;
}

// Configuration interfaces
export interface KNIRVOracleConfig {
  baseURL: string;
  apiKey?: string;
  timeout?: number;
}

// Agent-related interfaces
export interface AgentMintRequest {
  agent_id: string;
  name: string;
  description: string;
  owner: string;
  metadata: Record<string, any>;
  image_url?: string;
}

export interface AgentMintResponse extends KNIRVOracleBaseResponse {
  transaction_id: string;
  agent_nft_id: string;
}

export interface AgentMetadata {
  agent_type: string;
  model?: string;
  description?: string;
  use_search?: boolean;
  use_code_execution?: boolean;
  use_vertex_search?: boolean;
  created_at: number;
  version: string;
  [key: string]: any;
}

// Capability-related interfaces
export interface CapabilityRegistrationRequest {
  name: string;
  type: string;
  description: string;
  schema: CapabilitySchema;
  owner: string;
  gas_fee_nrn: number;
  location_hints?: string[];
}

export interface CapabilityRegistrationResponse extends KNIRVOracleBaseResponse {
  capability_id: string;
  tx_hash: string;
}

export interface CapabilitySchema {
  type: 'object' | 'string' | 'number' | 'boolean' | 'array';
  properties?: Record<string, CapabilitySchemaProperty>;
  required?: string[];
  items?: CapabilitySchemaProperty;
  description?: string;
}

export interface CapabilitySchemaProperty {
  type: 'object' | 'string' | 'number' | 'boolean' | 'array';
  description?: string;
  properties?: Record<string, CapabilitySchemaProperty>;
  items?: CapabilitySchemaProperty;
  required?: string[];
  enum?: any[];
  default?: any;
}

export interface AgentCapability {
  id: string;
  name: string;
  type: string;
  description: string;
  schema: CapabilitySchema;
  status: 'available' | 'registering' | 'registration_failed' | 'disabled';
  knirvoracle_id?: string;
  tx_hash?: string;
  registered_at?: number;
  error?: string;
}

export interface CapabilityInvocationRequest {
  capability_id: string;
  interaction_type: 'invoke' | 'query' | 'execute';
  input_data: Record<string, any>;
  timestamp: number;
}

export interface CapabilityInvocationResponse {
  success: boolean;
  result?: any;
  error?: string;
  execution_time?: number;
  gas_consumed?: number;
}

// Wallet-related interfaces
export interface FaucetRequest {
  address: string;
  amount: string;
  reason?: string;
}

export interface FaucetResponse extends KNIRVOracleBaseResponse {
  request_id: string;
  tx_hash: string;
  amount: string;
  status: 'pending' | 'completed' | 'failed';
}

export interface WalletBalanceResponse extends KNIRVOracleBaseResponse {
  address: string;
  balance: string;
  nrn_balance: string;
  usd_value: string;
}

export interface TransactionRequest {
  from: string;
  to: string;
  amount: string;
  token: string;
  memo?: string;
  gas_fee?: string;
}

export interface TransactionResponse extends KNIRVOracleBaseResponse {
  transaction_id: string;
  tx_hash: string;
  status: 'pending' | 'confirmed' | 'failed';
}

// Treasury and economics interfaces
export interface TreasuryStatus {
  total_balance: string;
  available_balance: string;
  reserved_balance: string;
  nrn_price_usd: string;
  last_updated: number;
  treasury_address: string;
}

export interface EconomicsOperation {
  operation_type: 'skill_invocation' | 'agent_minting' | 'capability_registration';
  cost_nrn: string;
  gas_fee: string;
  timestamp: number;
  transaction_id: string;
}

// Error handling interfaces
export interface KNIRVOracleError extends Error {
  code?: string;
  status?: number;
  response?: any;
  isKNIRVOracleError: true;
}

export interface KNIRVOracleErrorResponse {
  success: false;
  message: string;
  error_code?: string;
  details?: Record<string, any>;
}

// Service status interfaces
export interface HealthCheckResponse {
  status: 'healthy' | 'degraded' | 'unhealthy';
  version: string;
  uptime: number;
  services: ServiceStatus[];
}

export interface ServiceStatus {
  name: string;
  status: 'online' | 'offline' | 'degraded';
  last_check: number;
  response_time?: number;
}

// Integration status interfaces
export interface IntegrationStatus {
  knirvoracle_connected: boolean;
  last_health_check: number;
  api_key_valid: boolean;
  services_available: string[];
  error?: string;
}

// Utility types for type safety
export type KNIRVOracleOperation = 
  | 'agent_mint'
  | 'capability_register'
  | 'capability_invoke'
  | 'faucet_request'
  | 'balance_query'
  | 'transaction_send'
  | 'treasury_status'
  | 'health_check';

export type KNIRVOracleEndpoint = 
  | '/agent/mint'
  | '/wallet/mcp/create_register_capability'
  | '/wallet/mcp/create_invoke_capability'
  | '/api/mint/nrv'
  | '/balance/{address}'
  | '/transactions'
  | '/api/treasury/status'
  | '/health';

// Request/Response mapping for type safety
export interface KNIRVOracleRequestMap {
  '/agent/mint': AgentMintRequest;
  '/wallet/mcp/create_register_capability': CapabilityRegistrationRequest;
  '/wallet/mcp/create_invoke_capability': CapabilityInvocationRequest;
  '/api/mint/nrv': FaucetRequest;
  '/transactions': TransactionRequest;
}

export interface KNIRVOracleResponseMap {
  '/agent/mint': AgentMintResponse;
  '/wallet/mcp/create_register_capability': CapabilityRegistrationResponse;
  '/wallet/mcp/create_invoke_capability': CapabilityInvocationResponse;
  '/api/mint/nrv': FaucetResponse;
  '/balance/{address}': WalletBalanceResponse;
  '/transactions': TransactionResponse;
  '/api/treasury/status': TreasuryStatus;
  '/health': HealthCheckResponse;
}

// Event interfaces for real-time updates
export interface KNIRVOracleEvent {
  type: 'agent_minted' | 'capability_registered' | 'transaction_confirmed' | 'balance_updated';
  data: any;
  timestamp: number;
  source: 'knirvoracle';
}

export interface AgentMintedEvent extends KNIRVOracleEvent {
  type: 'agent_minted';
  data: {
    agent_id: string;
    nft_id: string;
    transaction_id: string;
    owner: string;
  };
}

export interface CapabilityRegisteredEvent extends KNIRVOracleEvent {
  type: 'capability_registered';
  data: {
    capability_id: string;
    name: string;
    tx_hash: string;
    owner: string;
  };
}

// Validation helpers
export const isKNIRVOracleError = (error: any): error is KNIRVOracleError => {
  return error && error.isKNIRVOracleError === true;
};

export const isValidAddress = (address: string): boolean => {
  return /^0x[a-fA-F0-9]{40}$/.test(address);
};

export const isValidAmount = (amount: string): boolean => {
  return /^\d+(\.\d+)?$/.test(amount) && parseFloat(amount) > 0;
};

// Constants for type safety
export const KNIRVORACLE_ENDPOINTS = {
  AGENT_MINT: '/agent/mint',
  CAPABILITY_REGISTER: '/wallet/mcp/create_register_capability',
  CAPABILITY_INVOKE: '/wallet/mcp/create_invoke_capability',
  FAUCET: '/api/mint/nrv',
  BALANCE: '/balance',
  TRANSACTIONS: '/transactions',
  TREASURY_STATUS: '/api/treasury/status',
  HEALTH: '/health',
} as const;

export const KNIRVORACLE_TIMEOUTS = {
  HEALTH_CHECK: 5000,
  BALANCE_QUERY: 10000,
  TRANSACTION: 30000,
  AGENT_MINT: 60000,
  CAPABILITY_REGISTER: 30000,
} as const;

export const KNIRVORACLE_RETRY_CONFIG = {
  MAX_RETRIES: 3,
  RETRY_DELAY: 1000,
  BACKOFF_MULTIPLIER: 2,
} as const;
