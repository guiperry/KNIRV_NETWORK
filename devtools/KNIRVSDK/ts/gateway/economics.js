/**
 * Economics Service for KNIRV Gateway SDK
 */
import { EconomicsServiceError, } from './types';
export class EconomicsService {
    constructor(config) {
        this.config = config;
    }
    updateConfig(config) {
        this.config = config;
    }
    async request(method, path, body, queryParams) {
        const url = this.buildURL(path, queryParams);
        const requestOptions = {
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
                throw new EconomicsServiceError(`HTTP ${response.status}: ${response.statusText}`, response.status);
            }
            const data = await response.json();
            if (data.success === false) {
                throw new EconomicsServiceError(data.error || 'Request failed');
            }
            return data.data || data;
        }
        catch (error) {
            if (error instanceof EconomicsServiceError) {
                throw error;
            }
            throw new EconomicsServiceError(`Request failed: ${error.message}`);
        }
    }
    buildURL(path, queryParams) {
        const baseURL = this.config.economicsURL;
        const url = new URL(path, baseURL);
        if (queryParams) {
            Object.entries(queryParams).forEach(([key, value]) => {
                url.searchParams.set(key, value);
            });
        }
        return url.toString();
    }
    getHeaders() {
        const headers = {};
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
    async invokeSkill(request) {
        return this.request('POST', '/economics/skill/invoke', request);
    }
    // LLM Service Methods
    /**
     * Process an LLM registration and handle the registration fee
     */
    async registerLLM(request) {
        return this.request('POST', '/economics/llm/register', request);
    }
    // Validation Service Methods
    /**
     * Process a validation reward
     */
    async processValidationReward(request) {
        return this.request('POST', '/economics/validation/reward', request);
    }
    // Fees Service Methods
    /**
     * Calculate network fees for a transaction
     */
    async calculateNetworkFees(request) {
        return this.request('POST', '/economics/fees/calculate', request);
    }
    // Metrics Service Methods
    /**
     * Get current economic metrics
     */
    async getMetrics() {
        return this.request('GET', '/economics/metrics');
    }
    /**
     * Get metrics for a specific service
     */
    async getServiceMetrics(serviceName) {
        return this.request('GET', `/economics/service/${serviceName}/metrics`);
    }
    // Transactions Service Methods
    /**
     * Get a specific transaction by ID
     */
    async getTransaction(transactionId) {
        return this.request('GET', `/economics/transaction/${transactionId}`);
    }
    /**
     * List transactions with optional filters
     */
    async listTransactions(options = {}) {
        const queryParams = {};
        if (options.limit)
            queryParams.limit = options.limit.toString();
        if (options.status)
            queryParams.status = options.status;
        if (options.offset)
            queryParams.offset = options.offset.toString();
        const response = await this.request('GET', '/economics/transactions', undefined, queryParams);
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
    async getBurnHistory(limit = 100) {
        const response = await this.request('GET', '/economics/burn/history', undefined, { limit: limit.toString() });
        return {
            items: response.burn_events,
            count: response.count,
            limit: response.limit,
        };
    }
    /**
     * Get total amount of burned tokens
     */
    async getTotalBurned() {
        return this.request('GET', '/economics/burn/total');
    }
    // Rules Service Methods
    /**
     * Get current economic rules
     */
    async getEconomicRules() {
        return this.request('GET', '/economics/rules');
    }
    /**
     * Update economic rules (requires admin privileges)
     */
    async updateEconomicRules(rules) {
        const response = await this.request('PUT', '/economics/rules', rules);
        return response.rules;
    }
    // Convenience Methods
    /**
     * Check if a user has sufficient balance for a skill invocation
     */
    async checkSkillInvocationBalance(userId, skillId, amount) {
        try {
            // This would typically check user balance against required amount
            // For now, we'll simulate by trying to get the current rules
            const rules = await this.getEconomicRules();
            const requiredAmount = BigInt(rules.skill_invocation_cost);
            const providedAmount = BigInt(amount);
            return providedAmount >= requiredAmount;
        }
        catch (error) {
            throw new EconomicsServiceError(`Balance check failed: ${error.message}`);
        }
    }
    /**
     * Get economics summary for a user
     */
    async getUserEconomicsSummary(userId) {
        // This would typically aggregate user-specific data
        // For now, we'll return a placeholder structure
        const transactions = await this.listTransactions({ limit: 1000 });
        const userTransactions = transactions.items.filter(tx => tx.from === userId || tx.to === userId);
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
    async getNetworkStatistics() {
        const metrics = await this.getMetrics();
        const transactions = await this.listTransactions({ limit: 1 });
        return {
            total_transactions: transactions.count,
            total_volume: metrics.transaction_volume,
            average_transaction_size: '0',
            active_users: metrics.active_validators, // Placeholder
        };
    }
}
