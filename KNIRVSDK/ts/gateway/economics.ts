/**
 * Economics Service for KNIRV Gateway SDK
 */

import {
  RequestConfig,
  SkillInvocationRequest,
  SkillInvocationResponse,
  LLMRegistrationRequest,
  LLMRegistrationResponse,
  ValidationRewardRequest,
  ValidationRewardResponse,
  NetworkFeesRequest,
  NetworkFeesResponse,
  EconomicMetrics,
  ServiceEconomics,
  Transaction,
  BurnEvent,
  EconomicRules,
  PaginatedResponse,
  EconomicsServiceError,
} from './types';

export class EconomicsService {
  constructor(private config: RequestConfig) {}

  updateConfig(config: RequestConfig): void {
    this.config = config;
  }

  private async request<T>(
    method: string,
    path: string,
    body?: any,
    queryParams?: Record<string, string>
  ): Promise<T> {
    const url = this.buildURL(path, queryParams);
    
    const requestOptions: RequestInit = {
      method: method.toUpperCase(),
      headers: {
        'Content-Type': 'application/json',
        'User-Agent': 'KNIRV-Gateway-SDK-TS/1.0.0',
        ...this.getHeaders(),
      },
    };

    if (body) {
      requestOptions.body = JSON.stringify(body);
    }

    try {
      const response = await fetch(url, requestOptions);
      
      if (!response.ok) {
        throw new EconomicsServiceError(
          `HTTP ${response.status}: ${response.statusText}`,
          response.status
        );
      }

      const data = await response.json();
      
      if (data.success === false) {
        throw new EconomicsServiceError(data.error || 'Request failed');
      }

      return data.data || data;
    } catch (error) {
      if (error instanceof EconomicsServiceError) {
        throw error;
      }
      throw new EconomicsServiceError(`Request failed: ${error.message}`);
    }
  }

  private buildURL(path: string, queryParams?: Record<string, string>): string {
    const baseURL = this.config.economicsURL;
    const url = new URL(path, baseURL);
    
    if (queryParams) {
      Object.entries(queryParams).forEach(([key, value]) => {
        url.searchParams.set(key, value);
      });
    }

    return url.toString();
  }

  private getHeaders(): Record<string, string> {
    const headers: Record<string, string> = {};

    if (this.config.apiKey) {
      headers['X-API-Key'] = this.config.apiKey;
    }

    if (this.config.nrnContract) {
      headers['X-NRN-Contract'] = this.config.nrnContract;
    }

    if (this.config.environment) {
      headers['X-Environment'] = this.config.environment;
    }

    return headers;
  }

  // Skills Service Methods

  /**
   * Process a skill invocation and handle the economic transaction
   */
  async invokeSkill(request: SkillInvocationRequest): Promise<SkillInvocationResponse> {
    return this.request<SkillInvocationResponse>('POST', '/economics/skill/invoke', request);
  }

  // LLM Service Methods

  /**
   * Process an LLM registration and handle the registration fee
   */
  async registerLLM(request: LLMRegistrationRequest): Promise<LLMRegistrationResponse> {
    return this.request<LLMRegistrationResponse>('POST', '/economics/llm/register', request);
  }

  // Validation Service Methods

  /**
   * Process a validation reward
   */
  async processValidationReward(request: ValidationRewardRequest): Promise<ValidationRewardResponse> {
    return this.request<ValidationRewardResponse>('POST', '/economics/validation/reward', request);
  }

  // Fees Service Methods

  /**
   * Calculate network fees for a transaction
   */
  async calculateNetworkFees(request: NetworkFeesRequest): Promise<NetworkFeesResponse> {
    return this.request<NetworkFeesResponse>('POST', '/economics/fees/calculate', request);
  }

  // Metrics Service Methods

  /**
   * Get current economic metrics
   */
  async getMetrics(): Promise<EconomicMetrics> {
    return this.request<EconomicMetrics>('GET', '/economics/metrics');
  }

  /**
   * Get metrics for a specific service
   */
  async getServiceMetrics(serviceName: string): Promise<ServiceEconomics> {
    return this.request<ServiceEconomics>('GET', `/economics/service/${serviceName}/metrics`);
  }

  // Transactions Service Methods

  /**
   * Get a specific transaction by ID
   */
  async getTransaction(transactionId: string): Promise<Transaction> {
    return this.request<Transaction>('GET', `/economics/transaction/${transactionId}`);
  }

  /**
   * List transactions with optional filters
   */
  async listTransactions(options: {
    limit?: number;
    status?: string;
    offset?: number;
  } = {}): Promise<PaginatedResponse<Transaction>> {
    const queryParams: Record<string, string> = {};
    
    if (options.limit) queryParams.limit = options.limit.toString();
    if (options.status) queryParams.status = options.status;
    if (options.offset) queryParams.offset = options.offset.toString();

    const response = await this.request<{
      transactions: Transaction[];
      count: number;
      limit: number;
    }>('GET', '/economics/transactions', undefined, queryParams);

    return {
      items: response.transactions,
      count: response.count,
      limit: response.limit,
      offset: options.offset,
    };
  }

  // Burn Service Methods

  /**
   * Get burn event history
   */
  async getBurnHistory(limit: number = 100): Promise<PaginatedResponse<BurnEvent>> {
    const response = await this.request<{
      burn_events: BurnEvent[];
      count: number;
      limit: number;
    }>('GET', '/economics/burn/history', undefined, { limit: limit.toString() });

    return {
      items: response.burn_events,
      count: response.count,
      limit: response.limit,
    };
  }

  /**
   * Get total amount of burned tokens
   */
  async getTotalBurned(): Promise<{ total_burned: string; timestamp: string }> {
    return this.request<{ total_burned: string; timestamp: string }>('GET', '/economics/burn/total');
  }

  // Rules Service Methods

  /**
   * Get current economic rules
   */
  async getEconomicRules(): Promise<EconomicRules> {
    return this.request<EconomicRules>('GET', '/economics/rules');
  }

  /**
   * Update economic rules (requires admin privileges)
   */
  async updateEconomicRules(rules: EconomicRules): Promise<EconomicRules> {
    const response = await this.request<{
      message: string;
      rules: EconomicRules;
    }>('PUT', '/economics/rules', rules);

    return response.rules;
  }

  // Convenience Methods

  /**
   * Check if a user has sufficient balance for a skill invocation
   */
  async checkSkillInvocationBalance(userId: string, skillId: string, amount: string): Promise<boolean> {
    try {
      // This would typically check user balance against required amount
      // For now, we'll simulate by trying to get the current rules
      const rules = await this.getEconomicRules();
      const requiredAmount = BigInt(rules.skill_invocation_cost);
      const providedAmount = BigInt(amount);
      
      return providedAmount >= requiredAmount;
    } catch (error) {
      throw new EconomicsServiceError(`Balance check failed: ${error.message}`);
    }
  }

  /**
   * Get economics summary for a user
   */
  async getUserEconomicsSummary(userId: string): Promise<{
    total_spent: string;
    total_earned: string;
    transaction_count: number;
    last_activity: string;
  }> {
    // This would typically aggregate user-specific data
    // For now, we'll return a placeholder structure
    const transactions = await this.listTransactions({ limit: 1000 });
    
    const userTransactions = transactions.items.filter(
      tx => tx.from === userId || tx.to === userId
    );

    let totalSpent = BigInt(0);
    let totalEarned = BigInt(0);
    let lastActivity = '';

    userTransactions.forEach(tx => {
      const amount = BigInt(tx.amount);
      if (tx.from === userId) {
        totalSpent += amount;
      }
      if (tx.to === userId) {
        totalEarned += amount;
      }
      if (tx.timestamp > lastActivity) {
        lastActivity = tx.timestamp;
      }
    });

    return {
      total_spent: totalSpent.toString(),
      total_earned: totalEarned.toString(),
      transaction_count: userTransactions.length,
      last_activity: lastActivity,
    };
  }

  /**
   * Get network statistics
   */
  async getNetworkStatistics(): Promise<{
    total_transactions: number;
    total_volume: string;
    average_transaction_size: string;
    active_users: number;
  }> {
    const metrics = await this.getMetrics();
    const transactions = await this.listTransactions({ limit: 1 });

    return {
      total_transactions: transactions.count,
      total_volume: metrics.transaction_volume,
      average_transaction_size: '0', // Would be calculated from actual data
      active_users: metrics.active_validators, // Placeholder
    };
  }
}
