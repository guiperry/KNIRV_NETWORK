/**
 * KNIRV Network Service Classes
 * Provides access to all KNIRV Network services through a unified interface
 */
import { Badge, SkillBadge, CapabilityBadge, PropertyBadge, DVEEnvironment, DVESession, TreasuryOperation, FaucetRequest, Agent, AgentWorkflow, ConnectivityProof, NetworkRoute, FactualitySlice, FactualityVerification, ServiceHealth, NetworkHealth, NRNToken, SkillInvocation, KNIRVNetworkInfo } from './types';
/**
 * Base service class with common HTTP functionality
 */
export declare class BaseService {
    protected baseURL: string;
    protected apiKey?: string;
    protected timeout: number;
    constructor(baseURL: string, apiKey?: string, timeout?: number);
    /**
     * Update the base URL for this service
     */
    updateBaseURL(newBaseURL: string): void;
    protected request<T>(method: 'GET' | 'POST' | 'PUT' | 'DELETE', endpoint: string, data?: any, headers?: Record<string, string>): Promise<T>;
}
/**
 * Badge System Service
 */
export declare class BadgeService extends BaseService {
    getAgentBadges(agentId: string): Promise<Badge[]>;
    getSkillBadges(agentId: string): Promise<SkillBadge[]>;
    getCapabilityBadges(agentId: string): Promise<CapabilityBadge[]>;
    getPropertyBadges(agentId: string): Promise<PropertyBadge[]>;
    validateBadge(badgeId: string): Promise<{
        valid: boolean;
        details: any;
    }>;
    issueBadge(badge: Partial<Badge>): Promise<Badge>;
}
/**
 * KNIRVSERVER DVE Service
 */
export declare class DVEService extends BaseService {
    listEnvironments(userId?: string): Promise<DVEEnvironment[]>;
    createEnvironment(config: Partial<DVEEnvironment>): Promise<DVEEnvironment>;
    getEnvironment(environmentId: string): Promise<DVEEnvironment>;
    deleteEnvironment(environmentId: string): Promise<void>;
    startSession(environmentId: string): Promise<DVESession>;
    getSession(sessionId: string): Promise<DVESession>;
    terminateSession(sessionId: string): Promise<void>;
}
/**
 * KNIRVORACLE Treasury Service
 */
export declare class TreasuryService extends BaseService {
    getNRNTokenInfo(): Promise<NRNToken>;
    getTreasuryBalance(): Promise<{
        balance: string;
        address: string;
    }>;
    getTreasuryOperations(limit?: number): Promise<TreasuryOperation[]>;
    requestFaucet(userAddress: string, amount: string): Promise<FaucetRequest>;
    getFaucetRequest(requestId: string): Promise<FaucetRequest>;
    mintNRN(amount: string, reason: string): Promise<TreasuryOperation>;
}
/**
 * KNIRVCONTROLLER Agent Service
 */
export declare class AgentService extends BaseService {
    listAgents(userId?: string): Promise<Agent[]>;
    getAgent(agentId: string): Promise<Agent>;
    createAgent(agent: Partial<Agent>): Promise<Agent>;
    updateAgent(agentId: string, updates: Partial<Agent>): Promise<Agent>;
    deleteAgent(agentId: string): Promise<void>;
    getAgentWorkflows(agentId: string): Promise<AgentWorkflow[]>;
    createWorkflow(agentId: string, workflow: Partial<AgentWorkflow>): Promise<AgentWorkflow>;
    invokeSkill(request: {
        agentId: string;
        skillId: string;
        userId: string;
        amount: string;
        parameters?: Record<string, any>;
    }): Promise<SkillInvocation>;
}
/**
 * KNIRVROUTER Network Service
 */
export declare class NetworkService extends BaseService {
    submitConnectivityProof(proof: Partial<ConnectivityProof>): Promise<ConnectivityProof>;
    getConnectivityProofs(agentId?: string): Promise<ConnectivityProof[]>;
    getNetworkRoutes(): Promise<NetworkRoute[]>;
    findOptimalRoute(source: string, destination: string): Promise<NetworkRoute>;
    getNetworkStats(): Promise<{
        totalAgents: number;
        activeRoutes: number;
        averageLatency: number;
        networkReliability: number;
    }>;
}
/**
 * Factuality Slice Service
 */
export declare class FactualityService extends BaseService {
    getSliceInfo(): Promise<FactualitySlice>;
    verifyContent(content: string, domain: string): Promise<FactualityVerification>;
    getVerificationHistory(limit?: number): Promise<FactualityVerification[]>;
    getVerification(verificationId: string): Promise<FactualityVerification>;
}
/**
 * Health Monitoring Service
 */
export declare class HealthService extends BaseService {
    getNetworkHealth(): Promise<NetworkHealth>;
    getServiceHealth(serviceName: string): Promise<ServiceHealth>;
    getAllServicesHealth(): Promise<Record<string, ServiceHealth>>;
    pingService(serviceName: string): Promise<{
        success: boolean;
        responseTime: number;
    }>;
}
/**
 * Network Configuration Service
 */
export declare class ConfigService extends BaseService {
    getNetworkInfo(): Promise<KNIRVNetworkInfo>;
    switchNetwork(environment: string): Promise<KNIRVNetworkInfo>;
    listNetworks(): Promise<KNIRVNetworkInfo[]>;
}
//# sourceMappingURL=services.d.ts.map