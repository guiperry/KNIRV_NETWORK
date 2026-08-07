/**
 * KNIRV Network Service Classes
 * Provides access to all KNIRV Network services through a unified interface
 */
/**
 * Base service class with common HTTP functionality
 */
export class BaseService {
    constructor(baseURL, apiKey, timeout = 30000) {
        this.baseURL = baseURL;
        this.apiKey = apiKey;
        this.timeout = timeout;
    }
    /**
     * Update the base URL for this service
     */
    updateBaseURL(newBaseURL) {
        this.baseURL = newBaseURL;
    }
    async request(method, endpoint, data, headers) {
        const requestHeaders = {
            'Content-Type': 'application/json',
            ...headers,
        };
        if (this.apiKey) {
            requestHeaders['Authorization'] = `Bearer ${this.apiKey}`;
        }
        const controller = new AbortController();
        const timeoutId = setTimeout(() => controller.abort(), this.timeout);
        const canonicalGateways = ['https://gateway.knirv.network', 'https://testnet-gateway.knirv.network', 'http://localhost:8080'];
        const candidates = canonicalGateways.includes(this.baseURL)
            ? [this.baseURL, ...canonicalGateways.filter((candidate) => candidate !== this.baseURL)]
            : [this.baseURL];
        let lastError;
        try {
            for (const baseURL of candidates) {
                try {
                    const response = await fetch(`${baseURL}${endpoint}`, {
                        method,
                        headers: requestHeaders,
                        body: data ? JSON.stringify(data) : undefined,
                        signal: controller.signal,
                    });
                    if (response.status >= 500)
                        throw new Error(`${baseURL} returned HTTP ${response.status}`);
                    if (!response.ok)
                        throw new Error(`HTTP ${response.status}: ${response.statusText}`);
                    return await response.json();
                }
                catch (error) {
                    lastError = error;
                }
            }
            throw lastError instanceof Error ? lastError : new Error('No KNIRV gateway is available');
        }
        finally {
            clearTimeout(timeoutId);
        }
    }
}
/**
 * Badge System Service
 */
export class BadgeService extends BaseService {
    async getAgentBadges(agentId) {
        return this.request('GET', `/api/agents/${agentId}/badges`);
    }
    async getSkillBadges(agentId) {
        return this.request('GET', `/api/agents/${agentId}/badges/skills`);
    }
    async getCapabilityBadges(agentId) {
        return this.request('GET', `/api/agents/${agentId}/badges/capabilities`);
    }
    async getPropertyBadges(agentId) {
        return this.request('GET', `/api/agents/${agentId}/badges/properties`);
    }
    async validateBadge(badgeId) {
        return this.request('POST', `/api/badges/${badgeId}/validate`);
    }
    async issueBadge(badge) {
        return this.request('POST', '/api/badges', badge);
    }
}
/**
 * KNIRVSERVER DVE Service
 */
export class DVEService extends BaseService {
    async listEnvironments(userId) {
        const params = userId ? `?userId=${userId}` : '';
        return this.request('GET', `/api/dve/environments${params}`);
    }
    async createEnvironment(config) {
        return this.request('POST', '/api/dve/environments', config);
    }
    async getEnvironment(environmentId) {
        return this.request('GET', `/api/dve/environments/${environmentId}`);
    }
    async deleteEnvironment(environmentId) {
        await this.request('DELETE', `/api/dve/environments/${environmentId}`);
    }
    async startSession(environmentId) {
        return this.request('POST', `/api/dve/environments/${environmentId}/sessions`);
    }
    async getSession(sessionId) {
        return this.request('GET', `/api/dve/sessions/${sessionId}`);
    }
    async terminateSession(sessionId) {
        await this.request('DELETE', `/api/dve/sessions/${sessionId}`);
    }
}
/**
 * KNIRVORACLE Treasury Service
 */
export class TreasuryService extends BaseService {
    async getNRNTokenInfo() {
        return this.request('GET', '/api/treasury/nrn');
    }
    async getTreasuryBalance() {
        return this.request('GET', '/api/treasury/balance');
    }
    async getTreasuryOperations(limit) {
        const params = limit ? `?limit=${limit}` : '';
        return this.request('GET', `/api/treasury/operations${params}`);
    }
    async requestFaucet(userAddress, amount) {
        return this.request('POST', '/api/treasury/faucet', {
            userAddress,
            amount,
        });
    }
    async getFaucetRequest(requestId) {
        return this.request('GET', `/api/treasury/faucet/${requestId}`);
    }
    async mintNRN(amount, reason) {
        return this.request('POST', '/api/treasury/mint', {
            amount,
            reason,
        });
    }
}
/**
 * KNIRVCONTROLLER Agent Service
 */
export class AgentService extends BaseService {
    async listAgents(userId) {
        const params = userId ? `?userId=${userId}` : '';
        return this.request('GET', `/api/agents${params}`);
    }
    async getAgent(agentId) {
        return this.request('GET', `/api/agents/${agentId}`);
    }
    async createAgent(agent) {
        return this.request('POST', '/api/agents', agent);
    }
    async updateAgent(agentId, updates) {
        return this.request('PUT', `/api/agents/${agentId}`, updates);
    }
    async deleteAgent(agentId) {
        await this.request('DELETE', `/api/agents/${agentId}`);
    }
    async getAgentWorkflows(agentId) {
        return this.request('GET', `/api/agents/${agentId}/workflows`);
    }
    async createWorkflow(agentId, workflow) {
        return this.request('POST', `/api/agents/${agentId}/workflows`, workflow);
    }
    async invokeSkill(request) {
        return this.request('POST', '/api/skills/invoke', request);
    }
}
/**
 * KNIRVROUTER Network Service
 */
export class NetworkService extends BaseService {
    async submitConnectivityProof(proof) {
        return this.request('POST', '/api/network/proofs', proof);
    }
    async getConnectivityProofs(agentId) {
        const params = agentId ? `?agentId=${agentId}` : '';
        return this.request('GET', `/api/network/proofs${params}`);
    }
    async getNetworkRoutes() {
        return this.request('GET', '/api/network/routes');
    }
    async findOptimalRoute(source, destination) {
        return this.request('GET', `/api/network/routes/optimal?source=${source}&destination=${destination}`);
    }
    async getNetworkStats() {
        return this.request('GET', '/api/network/stats');
    }
}
/**
 * Factuality Slice Service
 */
export class FactualityService extends BaseService {
    async getSliceInfo() {
        return this.request('GET', '/api/factuality/slice');
    }
    async verifyContent(content, domain) {
        return this.request('POST', '/api/factuality/verify', {
            content,
            domain,
        });
    }
    async getVerificationHistory(limit) {
        const params = limit ? `?limit=${limit}` : '';
        return this.request('GET', `/api/factuality/history${params}`);
    }
    async getVerification(verificationId) {
        return this.request('GET', `/api/factuality/verifications/${verificationId}`);
    }
}
/**
 * Health Monitoring Service
 */
export class HealthService extends BaseService {
    async getNetworkHealth() {
        return this.request('GET', '/api/health/network');
    }
    async getServiceHealth(serviceName) {
        return this.request('GET', `/api/health/services/${serviceName}`);
    }
    async getAllServicesHealth() {
        return this.request('GET', '/api/health/services');
    }
    async pingService(serviceName) {
        return this.request('POST', `/api/health/ping/${serviceName}`);
    }
}
/**
 * Network Configuration Service
 */
export class ConfigService extends BaseService {
    async getNetworkInfo() {
        return this.request('GET', '/api/config/network');
    }
    async switchNetwork(environment) {
        return this.request('POST', '/api/config/network/switch', {
            environment,
        });
    }
    async listNetworks() {
        return this.request('GET', '/api/config/networks');
    }
}
