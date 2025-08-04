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
}
export interface KNIRVAccount {
    address: string;
    publicKey: string;
    balance?: string;
    sequence?: number;
    accountNumber?: number;
}
export interface KNIRVTransaction {
    hash: string;
    height: number;
    gasUsed: number;
    gasWanted: number;
    fee: string;
    memo?: string;
    timestamp: string;
}
export interface SkillDefinition {
    id: string;
    name: string;
    description: string;
    category: string;
    cost: string;
    provider: string;
    metadata?: Record<string, any>;
}
export interface SkillInvocation {
    id: string;
    skillId: string;
    userId: string;
    amount: string;
    status: 'pending' | 'completed' | 'failed';
    result?: any;
    timestamp: string;
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
export interface ServiceHealth {
    service: string;
    status: 'healthy' | 'unhealthy' | 'degraded';
    uptime: number;
    lastCheck: string;
    details?: Record<string, any>;
}
//# sourceMappingURL=types.d.ts.map