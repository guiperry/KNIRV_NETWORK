/**
 * Economics Service for KNIRV Gateway SDK
 */
import { RequestConfig, SkillInvocationRequest, SkillInvocationResponse, LLMRegistrationRequest, LLMRegistrationResponse, ValidationRewardRequest, ValidationRewardResponse, NetworkFeesRequest, NetworkFeesResponse, EconomicMetrics, ServiceEconomics, Transaction, BurnEvent, EconomicRules, PaginatedResponse } from './types';
export declare class EconomicsService {
    private config;
    constructor(config: RequestConfig);
    updateConfig(config: RequestConfig): void;
    private request;
    private buildURL;
    private getHeaders;
    /**
     * Process a skill invocation and handle the economic transaction
     */
    invokeSkill(request: SkillInvocationRequest): Promise<SkillInvocationResponse>;
    /**
     * Process an LLM registration and handle the registration fee
     */
    registerLLM(request: LLMRegistrationRequest): Promise<LLMRegistrationResponse>;
    /**
     * Process a validation reward
     */
    processValidationReward(request: ValidationRewardRequest): Promise<ValidationRewardResponse>;
    /**
     * Calculate network fees for a transaction
     */
    calculateNetworkFees(request: NetworkFeesRequest): Promise<NetworkFeesResponse>;
    /**
     * Get current economic metrics
     */
    getMetrics(): Promise<EconomicMetrics>;
    /**
     * Get metrics for a specific service
     */
    getServiceMetrics(serviceName: string): Promise<ServiceEconomics>;
    /**
     * Get a specific transaction by ID
     */
    getTransaction(transactionId: string): Promise<Transaction>;
    /**
     * List transactions with optional filters
     */
    listTransactions(options?: {
        limit?: number;
        status?: string;
        offset?: number;
    }): Promise<PaginatedResponse<Transaction>>;
    /**
     * Get burn event history
     */
    getBurnHistory(limit?: number): Promise<PaginatedResponse<BurnEvent>>;
    /**
     * Get total amount of burned tokens
     */
    getTotalBurned(): Promise<{
        total_burned: string;
        timestamp: string;
    }>;
    /**
     * Get current economic rules
     */
    getEconomicRules(): Promise<EconomicRules>;
    /**
     * Update economic rules (requires admin privileges)
     */
    updateEconomicRules(rules: EconomicRules): Promise<EconomicRules>;
    /**
     * Check if a user has sufficient balance for a skill invocation
     */
    checkSkillInvocationBalance(userId: string, skillId: string, amount: string): Promise<boolean>;
    /**
     * Get economics summary for a user
     */
    getUserEconomicsSummary(userId: string): Promise<{
        total_spent: string;
        total_earned: string;
        transaction_count: number;
        last_activity: string;
    }>;
    /**
     * Get network statistics
     */
    getNetworkStatistics(): Promise<{
        total_transactions: number;
        total_volume: string;
        average_transaction_size: string;
        active_users: number;
    }>;
}
//# sourceMappingURL=economics.d.ts.map