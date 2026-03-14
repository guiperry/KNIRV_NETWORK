/**
 * TypeScript types for KNIRV Gateway SDK
 */

// Client Configuration Types

export interface ClientOptions {
  baseURL?: string;
  economicsURL?: string;
  apiKey?: string;
  nrnContract?: string;
  timeout?: number;
  retries?: number;
  retryDelay?: number;
  environment?: 'development' | 'staging' | 'production';
  debug?: boolean;
  verbose?: boolean;
  serviceURLs?: {
    knirvchain?: string;
    knirvserver?: string;
    knirvoracle?: string;
    knirvgraph?: string;
  };
}

export interface RequestConfig extends Required<ClientOptions> {
  // All ClientOptions are required in RequestConfig
}

export function defaultClientOptions(): RequestConfig {
  return {
    baseURL: process.env.KNIRVGATEWAY_BASE_URL || 
             process.env.GATEWAY_SERVICE_URL || 
             'http://localhost:8000',
    economicsURL: process.env.ECONOMICS_SERVICE_URL || 
                  'http://localhost:8090',
    apiKey: process.env.KNIRVGATEWAY_API_KEY || '',
    nrnContract: process.env.NRN_CONTRACT || '',
    timeout: 30000,
    retries: 3,
    retryDelay: 1000,
    environment: (process.env.NODE_ENV as any) || 'development',
    debug: process.env.KNIRV_DEBUG === 'true',
    verbose: process.env.KNIRV_VERBOSE === 'true',
    serviceURLs: {
      knirvchain: process.env.KNIRVCHAIN_URL || 'http://localhost:8080',
      knirvserver: process.env.KNIRVNEXUS_URL || 'http://localhost:8081',
      knirvoracle: process.env.KNIRVORACLE_URL || 'http://localhost:8082',
      knirvgraph: process.env.KNIRVGRAPH_URL || 'http://localhost:8083',
    },
  };
}

// API Response Types

export interface APIResponse<T = any> {
  success: boolean;
  data?: T;
  error?: string;
}

// Economics Service Types

export interface SkillInvocationRequest {
  user_id: string;
  skill_id: string;
  amount: string;
}

export interface SkillInvocationResponse {
  transaction_id: string;
  status: string;
  amount: string;
  timestamp: string;
}

export interface LLMRegistrationRequest {
  user_id: string;
  llm_id: string;
  registration_fee: string;
}

export interface LLMRegistrationResponse {
  transaction_id: string;
  status: string;
  fee: string;
  timestamp: string;
}

export interface ValidationRewardRequest {
  validator_id: string;
  target_id: string;
  validation_result: boolean;
}

export interface ValidationRewardResponse {
  transaction_id: string;
  status: string;
  reward: string;
  timestamp: string;
}

export interface NetworkFeesRequest {
  gas_used: number;
  priority: 'low' | 'medium' | 'high';
}

export interface NetworkFeesResponse {
  gas_used: number;
  priority: string;
  total_fee: string;
  gas_price: string;
}

export interface EconomicMetrics {
  total_supply: string;
  circulating_supply: string;
  total_burned: string;
  total_staked: string;
  active_validators: number;
  transaction_volume: string;
  average_gas_price: string;
  network_utilization: number;
  token_velocity: number;
  last_updated: string;
  service_metrics: Record<string, ServiceEconomics>;
}

export interface ServiceEconomics {
  revenue: string;
  costs: string;
  profit: string;
  tokens_earned: string;
  tokens_spent: string;
  user_count: number;
  transaction_count: number;
  last_updated: string;
}

export interface Transaction {
  id: string;
  type: string;
  from: string;
  to: string;
  amount: string;
  purpose: string;
  metadata: Record<string, any>;
  status: string;
  timestamp: string;
  confirmed_at?: string;
  block_height?: number;
  gas_used?: number;
}

export interface BurnEvent {
  tx_id: string;
  user: string;
  amount: string;
  purpose: string;
  skill_id?: string;
  timestamp: string;
  validated: boolean;
}

export interface EconomicRules {
  skill_invocation_cost: string;
  llm_registration_fee: string;
  validation_reward: string;
  burn_rates: Record<string, string>;
  minting_rules: MintingRules;
  staking_requirements: StakingRequirements;
  governance_thresholds: GovernanceThresholds;
}

export interface MintingRules {
  max_supply: string;
  inflation_rate: number;
  validator_rewards: string;
  developer_rewards: string;
  community_rewards: string;
}

export interface StakingRequirements {
  min_validator_stake: string;
  min_developer_stake: string;
  slashing_penalty: number;
  unbonding_period: string; // Duration in string format
}

export interface GovernanceThresholds {
  proposal_deposit: string;
  voting_threshold: number;
  quorum_threshold: number;
  voting_period: string; // Duration in string format
}

// Gateway Service Types

export interface Route {
  path: string;
  methods: string[];
  target: string;
  auth_required: boolean;
  rate_limit: number;
}

export interface GatewayStatus {
  status: string;
  version: string;
  uptime: string; // Duration in string format
  services: Record<string, string>;
  last_updated: string;
}

export interface IntegrationStatus {
  knirvchain_url: string;
  knirvnexus_url: string;
  knirvoracle_url: string;
  knirvgraph_url: string;
  last_sync: string;
  status: string;
}

// Utility Types

export interface PaginatedResponse<T> {
  items: T[];
  count: number;
  limit: number;
  offset?: number;
  total?: number;
}

export interface HealthCheckResponse {
  status: 'healthy' | 'unhealthy' | 'degraded';
  timestamp: string;
  version?: string;
  checks?: Record<string, {
    status: 'pass' | 'fail' | 'warn';
    message?: string;
    timestamp: string;
  }>;
}

// Error Types

export class KNIRVGatewayError extends Error {
  constructor(
    message: string,
    public statusCode?: number,
    public response?: any
  ) {
    super(message);
    this.name = 'KNIRVGatewayError';
  }
}

export class EconomicsServiceError extends KNIRVGatewayError {
  constructor(message: string, statusCode?: number, response?: any) {
    super(message, statusCode, response);
    this.name = 'EconomicsServiceError';
  }
}

export class GatewayServiceError extends KNIRVGatewayError {
  constructor(message: string, statusCode?: number, response?: any) {
    super(message, statusCode, response);
    this.name = 'GatewayServiceError';
  }
}

// Event Types for Real-time Updates

export interface EconomicEvent {
  type: 'skill_invocation' | 'llm_registration' | 'validation_reward' | 'burn_event';
  data: any;
  timestamp: string;
}

export interface TransactionEvent {
  transaction_id: string;
  status: 'pending' | 'confirmed' | 'failed';
  timestamp: string;
}

// Webhook Types

export interface WebhookPayload {
  event: string;
  data: any;
  timestamp: string;
  signature?: string;
}

// Configuration Validation

export function validateClientOptions(options: Partial<ClientOptions>): string[] {
  const errors: string[] = [];

  if (options.baseURL && !isValidURL(options.baseURL)) {
    errors.push('Invalid baseURL format');
  }

  if (options.economicsURL && !isValidURL(options.economicsURL)) {
    errors.push('Invalid economicsURL format');
  }

  if (options.timeout && (options.timeout < 0 || options.timeout > 300000)) {
    errors.push('Timeout must be between 0 and 300000ms');
  }

  if (options.retries && (options.retries < 0 || options.retries > 10)) {
    errors.push('Retries must be between 0 and 10');
  }

  return errors;
}

function isValidURL(url: string): boolean {
  try {
    new URL(url);
    return true;
  } catch {
    return false;
  }
}

// Testnet Faucet Types

/**
 * Request for NRV tokens from the testnet faucet
 */
export interface FaucetRequest {
  /** Target address for token distribution */
  address: string;
  /** Amount of NRV tokens to request */
  amount: number;
  /** Optional reason for the request */
  reason?: string;
}

/**
 * Response from testnet faucet request
 */
export interface FaucetResponse {
  /** Whether the request was successful */
  success: boolean;
  /** Unique request identifier */
  request_id?: string;
  /** Target address */
  address?: string;
  /** Amount of tokens requested */
  amount?: number;
  /** Transaction hash if successful */
  tx_hash?: string;
  /** Request timestamp */
  timestamp?: string;
  /** Estimated confirmation time */
  estimated_confirmation?: string;
  /** Error message if failed */
  error?: string;
  /** Error code for programmatic handling */
  code?: string;
  /** Seconds to wait before retry (for rate limiting) */
  retry_after?: number;
  /** Type of rate limit hit */
  limit_type?: string;
}

/**
 * Current status of the testnet faucet
 */
export interface FaucetStatus {
  /** Whether the faucet is currently enabled */
  faucet_enabled: boolean;
  /** Current faucet balance in NRV */
  current_balance: number;
  /** Daily distribution limit */
  daily_limit: number;
  /** Remaining tokens available today */
  remaining_today: number;
  /** Current request queue size */
  current_queue_size: number;
  /** Success rate for today (0-1) */
  success_rate_today: number;
  /** Rate limiting configuration */
  rate_limits: Record<string, any>;
  /** Supported request amounts */
  supported_amounts: Record<string, any>;
  /** Last funding timestamp */
  last_funding?: string;
  /** Estimated next funding time */
  next_funding_estimate?: string;
}

/**
 * Single entry in faucet request history
 */
export interface FaucetHistoryEntry {
  /** Unique request identifier */
  request_id: string;
  /** Amount of tokens requested */
  amount: number;
  /** Request status */
  status: string;
  /** Request timestamp */
  timestamp: string;
  /** Transaction hash if successful */
  tx_hash?: string;
  /** Error message if failed */
  error?: string;
  /** Reason for the request */
  reason?: string;
}

/**
 * Faucet request history for an address
 */
export interface FaucetHistory {
  /** Target address */
  address: string;
  /** Total number of requests */
  total_requests: number;
  /** Total amount of tokens received */
  total_amount: number;
  /** Array of historical requests */
  history: FaucetHistoryEntry[];
}

/**
 * Faucet health check response
 */
export interface FaucetHealthResponse {
  /** Overall health status */
  status: string;
  /** Detailed health information */
  [key: string]: any;
}
