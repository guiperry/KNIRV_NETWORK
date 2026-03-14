/**
 * TypeScript types for KNIRV Gateway SDK
 */
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
        knirvnexus?: string;
        knirvoracle?: string;
        knirvgraph?: string;
    };
}
export interface RequestConfig extends Required<ClientOptions> {
}
export declare function defaultClientOptions(): RequestConfig;
export interface APIResponse<T = any> {
    success: boolean;
    data?: T;
    error?: string;
}
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
    unbonding_period: string;
}
export interface GovernanceThresholds {
    proposal_deposit: string;
    voting_threshold: number;
    quorum_threshold: number;
    voting_period: string;
}
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
    uptime: string;
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
export declare class KNIRVGatewayError extends Error {
    statusCode?: number;
    response?: any;
    constructor(message: string, statusCode?: number, response?: any);
}
export declare class EconomicsServiceError extends KNIRVGatewayError {
    constructor(message: string, statusCode?: number, response?: any);
}
export declare class GatewayServiceError extends KNIRVGatewayError {
    constructor(message: string, statusCode?: number, response?: any);
}
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
export interface WebhookPayload {
    event: string;
    data: any;
    timestamp: string;
    signature?: string;
}
export declare function validateClientOptions(options: Partial<ClientOptions>): string[];
//# sourceMappingURL=types.d.ts.map