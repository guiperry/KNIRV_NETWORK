/**
 * KNIRV Network Service Classes
 * Provides access to all KNIRV Network services through a unified interface
 */

import {
  Badge,
  SkillBadge,
  CapabilityBadge,
  PropertyBadge,
  DVEEnvironment,
  DVESession,
  TreasuryOperation,
  FaucetRequest,
  Agent,
  AgentWorkflow,
  ConnectivityProof,
  NetworkRoute,
  FactualitySlice,
  FactualityVerification,
  ServiceHealth,
  NetworkHealth,
  NRNToken,
  SkillDefinition,
  SkillInvocation,
  KNIRVNetworkInfo
} from './types';

/**
 * Base service class with common HTTP functionality
 */
export class BaseService {
  constructor(
    protected baseURL: string,
    protected apiKey?: string,
    protected timeout: number = 30000
  ) {}

  /**
   * Update the base URL for this service
   */
  updateBaseURL(newBaseURL: string): void {
    this.baseURL = newBaseURL;
  }

  protected async request<T>(
    method: 'GET' | 'POST' | 'PUT' | 'DELETE',
    endpoint: string,
    data?: any,
    headers?: Record<string, string>
  ): Promise<T> {
    const url = `${this.baseURL}${endpoint}`;
    const requestHeaders: Record<string, string> = {
      'Content-Type': 'application/json',
      ...headers,
    };

    if (this.apiKey) {
      requestHeaders['Authorization'] = `Bearer ${this.apiKey}`;
    }

    const controller = new AbortController();
    const timeoutId = setTimeout(() => controller.abort(), this.timeout);

    try {
      const response = await fetch(url, {
        method,
        headers: requestHeaders,
        body: data ? JSON.stringify(data) : undefined,
        signal: controller.signal,
      });

      clearTimeout(timeoutId);

      if (!response.ok) {
        throw new Error(`HTTP ${response.status}: ${response.statusText}`);
      }

      return await response.json();
    } catch (error) {
      clearTimeout(timeoutId);
      throw error;
    }
  }
}

/**
 * Badge System Service
 */
export class BadgeService extends BaseService {
  async getAgentBadges(agentId: string): Promise<Badge[]> {
    return this.request<Badge[]>('GET', `/api/agents/${agentId}/badges`);
  }

  async getSkillBadges(agentId: string): Promise<SkillBadge[]> {
    return this.request<SkillBadge[]>('GET', `/api/agents/${agentId}/badges/skills`);
  }

  async getCapabilityBadges(agentId: string): Promise<CapabilityBadge[]> {
    return this.request<CapabilityBadge[]>('GET', `/api/agents/${agentId}/badges/capabilities`);
  }

  async getPropertyBadges(agentId: string): Promise<PropertyBadge[]> {
    return this.request<PropertyBadge[]>('GET', `/api/agents/${agentId}/badges/properties`);
  }

  async validateBadge(badgeId: string): Promise<{ valid: boolean; details: any }> {
    return this.request('POST', `/api/badges/${badgeId}/validate`);
  }

  async issueBadge(badge: Partial<Badge>): Promise<Badge> {
    return this.request<Badge>('POST', '/api/badges', badge);
  }
}

/**
 * KNIRVSERVER DVE Service
 */
export class DVEService extends BaseService {
  async listEnvironments(userId?: string): Promise<DVEEnvironment[]> {
    const params = userId ? `?userId=${userId}` : '';
    return this.request<DVEEnvironment[]>('GET', `/api/dve/environments${params}`);
  }

  async createEnvironment(config: Partial<DVEEnvironment>): Promise<DVEEnvironment> {
    return this.request<DVEEnvironment>('POST', '/api/dve/environments', config);
  }

  async getEnvironment(environmentId: string): Promise<DVEEnvironment> {
    return this.request<DVEEnvironment>('GET', `/api/dve/environments/${environmentId}`);
  }

  async deleteEnvironment(environmentId: string): Promise<void> {
    await this.request('DELETE', `/api/dve/environments/${environmentId}`);
  }

  async startSession(environmentId: string): Promise<DVESession> {
    return this.request<DVESession>('POST', `/api/dve/environments/${environmentId}/sessions`);
  }

  async getSession(sessionId: string): Promise<DVESession> {
    return this.request<DVESession>('GET', `/api/dve/sessions/${sessionId}`);
  }

  async terminateSession(sessionId: string): Promise<void> {
    await this.request('DELETE', `/api/dve/sessions/${sessionId}`);
  }
}

/**
 * KNIRVORACLE Treasury Service
 */
export class TreasuryService extends BaseService {
  async getNRNTokenInfo(): Promise<NRNToken> {
    return this.request<NRNToken>('GET', '/api/treasury/nrn');
  }

  async getTreasuryBalance(): Promise<{ balance: string; address: string }> {
    return this.request('GET', '/api/treasury/balance');
  }

  async getTreasuryOperations(limit?: number): Promise<TreasuryOperation[]> {
    const params = limit ? `?limit=${limit}` : '';
    return this.request<TreasuryOperation[]>('GET', `/api/treasury/operations${params}`);
  }

  async requestFaucet(userAddress: string, amount: string): Promise<FaucetRequest> {
    return this.request<FaucetRequest>('POST', '/api/treasury/faucet', {
      userAddress,
      amount,
    });
  }

  async getFaucetRequest(requestId: string): Promise<FaucetRequest> {
    return this.request<FaucetRequest>('GET', `/api/treasury/faucet/${requestId}`);
  }

  async mintNRN(amount: string, reason: string): Promise<TreasuryOperation> {
    return this.request<TreasuryOperation>('POST', '/api/treasury/mint', {
      amount,
      reason,
    });
  }
}

/**
 * KNIRVCONTROLLER Agent Service
 */
export class AgentService extends BaseService {
  async listAgents(userId?: string): Promise<Agent[]> {
    const params = userId ? `?userId=${userId}` : '';
    return this.request<Agent[]>('GET', `/api/agents${params}`);
  }

  async getAgent(agentId: string): Promise<Agent> {
    return this.request<Agent>('GET', `/api/agents/${agentId}`);
  }

  async createAgent(agent: Partial<Agent>): Promise<Agent> {
    return this.request<Agent>('POST', '/api/agents', agent);
  }

  async updateAgent(agentId: string, updates: Partial<Agent>): Promise<Agent> {
    return this.request<Agent>('PUT', `/api/agents/${agentId}`, updates);
  }

  async deleteAgent(agentId: string): Promise<void> {
    await this.request('DELETE', `/api/agents/${agentId}`);
  }

  async getAgentWorkflows(agentId: string): Promise<AgentWorkflow[]> {
    return this.request<AgentWorkflow[]>('GET', `/api/agents/${agentId}/workflows`);
  }

  async createWorkflow(agentId: string, workflow: Partial<AgentWorkflow>): Promise<AgentWorkflow> {
    return this.request<AgentWorkflow>('POST', `/api/agents/${agentId}/workflows`, workflow);
  }

  async invokeSkill(request: {
    agentId: string;
    skillId: string;
    userId: string;
    amount: string;
    parameters?: Record<string, any>;
  }): Promise<SkillInvocation> {
    return this.request<SkillInvocation>('POST', '/api/skills/invoke', request);
  }
}

/**
 * KNIRVROUTER Network Service
 */
export class NetworkService extends BaseService {
  async submitConnectivityProof(proof: Partial<ConnectivityProof>): Promise<ConnectivityProof> {
    return this.request<ConnectivityProof>('POST', '/api/network/proofs', proof);
  }

  async getConnectivityProofs(agentId?: string): Promise<ConnectivityProof[]> {
    const params = agentId ? `?agentId=${agentId}` : '';
    return this.request<ConnectivityProof[]>('GET', `/api/network/proofs${params}`);
  }

  async getNetworkRoutes(): Promise<NetworkRoute[]> {
    return this.request<NetworkRoute[]>('GET', '/api/network/routes');
  }

  async findOptimalRoute(source: string, destination: string): Promise<NetworkRoute> {
    return this.request<NetworkRoute>('GET', `/api/network/routes/optimal?source=${source}&destination=${destination}`);
  }

  async getNetworkStats(): Promise<{
    totalAgents: number;
    activeRoutes: number;
    averageLatency: number;
    networkReliability: number;
  }> {
    return this.request('GET', '/api/network/stats');
  }
}

/**
 * Factuality Slice Service
 */
export class FactualityService extends BaseService {
  async getSliceInfo(): Promise<FactualitySlice> {
    return this.request<FactualitySlice>('GET', '/api/factuality/slice');
  }

  async verifyContent(content: string, domain: string): Promise<FactualityVerification> {
    return this.request<FactualityVerification>('POST', '/api/factuality/verify', {
      content,
      domain,
    });
  }

  async getVerificationHistory(limit?: number): Promise<FactualityVerification[]> {
    const params = limit ? `?limit=${limit}` : '';
    return this.request<FactualityVerification[]>('GET', `/api/factuality/history${params}`);
  }

  async getVerification(verificationId: string): Promise<FactualityVerification> {
    return this.request<FactualityVerification>('GET', `/api/factuality/verifications/${verificationId}`);
  }
}

/**
 * Health Monitoring Service
 */
export class HealthService extends BaseService {
  async getNetworkHealth(): Promise<NetworkHealth> {
    return this.request<NetworkHealth>('GET', '/api/health/network');
  }

  async getServiceHealth(serviceName: string): Promise<ServiceHealth> {
    return this.request<ServiceHealth>('GET', `/api/health/services/${serviceName}`);
  }

  async getAllServicesHealth(): Promise<Record<string, ServiceHealth>> {
    return this.request('GET', '/api/health/services');
  }

  async pingService(serviceName: string): Promise<{ success: boolean; responseTime: number }> {
    return this.request('POST', `/api/health/ping/${serviceName}`);
  }
}

/**
 * Network Configuration Service
 */
export class ConfigService extends BaseService {
  async getNetworkInfo(): Promise<KNIRVNetworkInfo> {
    return this.request<KNIRVNetworkInfo>('GET', '/api/config/network');
  }

  async switchNetwork(environment: string): Promise<KNIRVNetworkInfo> {
    return this.request<KNIRVNetworkInfo>('POST', '/api/config/network/switch', {
      environment,
    });
  }

  async listNetworks(): Promise<KNIRVNetworkInfo[]> {
    return this.request<KNIRVNetworkInfo[]>('GET', '/api/config/networks');
  }
}
