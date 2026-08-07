/**
 * Common types used across KNIRV SDK modules
 */
export interface KNIRVNetworkInfo {
    chainId: string;
    networkName: string;
    rpcUrl: string;
    currency: {
        name: string;
        symbol: string;
        decimals: number;
    };
    environment: 'public-testnet' | 'public-production' | 'local-testnet' | 'local-production';
    services: {
        controller: string;
        router: string;
        graph: string;
        chain: string;
        oracle: string;
        nexus: string;
        gateway: string;
    };
}
export interface KNIRVAccount {
    address: string;
    publicKey: string;
    balance?: string;
    sequence?: number;
    accountNumber?: number;
    nrnBalance?: string;
    xionMetaAccount?: XIONMetaAccount;
}
export interface KNIRVTransaction {
    hash: string;
    height: number;
    gasUsed: number;
    gasWanted: number;
    fee: string;
    memo?: string;
    timestamp: string;
    nrnAmount?: string;
    gasless?: boolean;
}
export interface XIONMetaAccount {
    address: string;
    email?: string;
    socialProvider?: string;
    walletProvider?: string;
    passkeyEnabled: boolean;
    treasuryContract?: string;
}
export interface TreasuryContract {
    address: string;
    balance: string;
    gaslessTransactionsEnabled: boolean;
    sponsorshipLimit: string;
    dailyLimit: string;
}
export interface NRNToken {
    symbol: 'NRN';
    decimals: 18;
    totalSupply: string;
    circulatingSupply: string;
    mintingRate: string;
    treasuryBalance: string;
}
export interface Badge {
    id: string;
    type: 'skill' | 'capability' | 'property';
    name: string;
    description: string;
    criteria: string;
    issuer: string;
    issuedAt: string;
    expiresAt?: string;
    metadata: Record<string, any>;
    verified: boolean;
}
export interface SkillBadge extends Badge {
    type: 'skill';
    skillId: string;
    proficiencyLevel: 'beginner' | 'intermediate' | 'advanced' | 'expert';
    validationProofs: string[];
}
export interface CapabilityBadge extends Badge {
    type: 'capability';
    capabilityType: string;
    permissions: string[];
    scope: string[];
}
export interface PropertyBadge extends Badge {
    type: 'property';
    propertyType: string;
    value: string;
    attestations: string[];
}
export interface SkillDefinition {
    id: string;
    name: string;
    description: string;
    category: string;
    cost: string;
    provider: string;
    metadata?: Record<string, any>;
    badges?: SkillBadge[];
    capabilities?: string[];
}
export interface SkillInvocation {
    id: string;
    skillId: string;
    userId: string;
    agentId?: string;
    amount: string;
    status: 'pending' | 'completed' | 'failed';
    result?: any;
    timestamp: string;
    nrnCost?: string;
    gasless?: boolean;
}
export interface KNIRVResource {
    uri: string;
    type: string;
    size: number;
    hash: string;
    metadata?: Record<string, any>;
}
export interface EconomicMetrics {
    totalSupply: string;
    circulatingSupply: string;
    marketCap?: string;
    volume24h?: string;
    priceUSD?: string;
}
export interface DVEEnvironment {
    id: string;
    name: string;
    type: 'development' | 'testing' | 'staging';
    status: 'active' | 'inactive' | 'suspended';
    resources: {
        cpu: string;
        memory: string;
        storage: string;
    };
    endpoints: {
        api: string;
        websocket: string;
        ssh?: string;
    };
    createdAt: string;
    expiresAt?: string;
}
export interface DVESession {
    id: string;
    environmentId: string;
    userId: string;
    status: 'active' | 'idle' | 'terminated';
    startedAt: string;
    lastActivity: string;
    resources: {
        cpuUsage: number;
        memoryUsage: number;
        storageUsage: number;
    };
}
export interface TreasuryOperation {
    id: string;
    type: 'mint' | 'burn' | 'transfer' | 'faucet';
    amount: string;
    fromAddress?: string;
    toAddress?: string;
    reason: string;
    timestamp: string;
    txHash?: string;
    status: 'pending' | 'completed' | 'failed';
}
export interface FaucetRequest {
    id: string;
    userAddress: string;
    amount: string;
    status: 'pending' | 'approved' | 'rejected' | 'completed';
    requestedAt: string;
    processedAt?: string;
    txHash?: string;
}
export interface Agent {
    id: string;
    name: string;
    type: string;
    status: 'active' | 'inactive' | 'error';
    capabilities: string[];
    badges: Badge[];
    owner: string;
    createdAt: string;
    lastActivity: string;
    configuration: Record<string, any>;
}
export interface AgentWorkflow {
    id: string;
    agentId: string;
    name: string;
    description: string;
    steps: WorkflowStep[];
    status: 'draft' | 'active' | 'paused' | 'completed';
    createdAt: string;
    updatedAt: string;
}
export interface WorkflowStep {
    id: string;
    type: 'skill' | 'condition' | 'action';
    name: string;
    configuration: Record<string, any>;
    dependencies: string[];
    timeout?: number;
}
export interface ConnectivityProof {
    id: string;
    agentId: string;
    proofType: 'ping' | 'bandwidth' | 'latency' | 'availability';
    proofData: Record<string, any>;
    timestamp: string;
    verified: boolean;
    nrnReward?: string;
}
export interface NetworkRoute {
    id: string;
    source: string;
    destination: string;
    latency: number;
    bandwidth: number;
    reliability: number;
    cost: string;
    lastUpdated: string;
}
export interface FactualitySlice {
    id: string;
    type: 'factuality-verification';
    status: 'active' | 'degraded' | 'inactive';
    configuration: {
        verificationThreshold: number;
        confidenceLevels: string[];
        supportedDomains: string[];
        maxConcurrentVerifications: number;
        cacheDurationHours: number;
    };
    networkIntegration: {
        controllerEndpoint: string;
        routerEndpoint: string;
        graphEndpoint: string;
        chainEndpoint: string;
        oracleEndpoint: string;
        nexusEndpoint: string;
    };
    initializedAt: string;
    lastHealthCheck: string;
}
export interface FactualityVerification {
    id: string;
    content: string;
    domain: string;
    confidence: 'low' | 'medium' | 'high' | 'verified';
    score: number;
    sources: string[];
    timestamp: string;
    cached: boolean;
}
export interface ServiceHealth {
    service: string;
    status: 'healthy' | 'unhealthy' | 'degraded';
    uptime: number;
    lastCheck: string;
    responseTime?: string;
    details?: Record<string, any>;
}
export interface NetworkHealth {
    overall: 'healthy' | 'degraded' | 'unhealthy';
    services: Record<string, ServiceHealth>;
    timestamp: string;
    summary: {
        total: number;
        healthy: number;
        unhealthy: number;
    };
}
//# sourceMappingURL=types.d.ts.map