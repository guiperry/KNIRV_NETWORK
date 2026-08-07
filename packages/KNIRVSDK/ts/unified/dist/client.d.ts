/**
 * Unified KNIRV Client - Provides access to all KNIRV Network services
 */
import { KNIRVWallet } from './wallet';
import { KNIRVNetworkInfo, NetworkHealth, XIONMetaAccount } from './types';
export interface KNIRVClientConfig {
    network?: {
        environment?: 'public-testnet' | 'public-production' | 'local-testnet' | 'local-production';
        customEndpoints?: {
            controller?: string;
            router?: string;
            graph?: string;
            chain?: string;
            oracle?: string;
            nexus?: string;
            gateway?: string;
        };
    };
    gateway?: {
        baseURL?: string;
        economicsURL?: string;
        apiKey?: string;
        timeout?: number;
        retries?: number;
    };
    auth?: {
        apiKey?: string;
        xionMetaAccount?: XIONMetaAccount;
        treasuryContract?: string;
    };
    wallet?: {
        provider?: string;
        chainId?: string;
        enableGasless?: boolean;
    };
    debug?: boolean;
    verbose?: boolean;
}
import { BadgeService, DVEService, TreasuryService, AgentService, NetworkService, FactualityService, HealthService, ConfigService } from './services';
/**
 * Unified KNIRV Client providing access to all KNIRV Network services
 */
export declare class KNIRVClient {
    readonly wallet: KNIRVWallet;
    readonly badges: BadgeService;
    readonly dve: DVEService;
    readonly treasury: TreasuryService;
    readonly agents: AgentService;
    readonly network: NetworkService;
    readonly factuality: FactualityService;
    readonly health: HealthService;
    readonly config: ConfigService;
    private readonly networkConfig;
    constructor(config?: KNIRVClientConfig);
    /**
     * Create a client optimized for gateway operations (temporarily disabled)
     */
    /**
     * Create a client optimized for transaction operations (temporarily disabled)
     */
    /**
     * Create a client optimized for transmission operations (temporarily disabled)
     */
    /**
     * Get network configuration for the specified environment
     */
    private getNetworkConfig;
    /**
     * Get the current network configuration
     */
    getNetworkInfo(): KNIRVNetworkInfo;
    /**
     * Switch to a different network environment
     */
    switchNetwork(environment: string): Promise<KNIRVNetworkInfo>;
    /**
     * Get the current wallet instance
     */
    getWallet(): KNIRVWallet;
    /**
     * Check if the client is properly configured and connected
     */
    isConnected(): Promise<boolean>;
    /**
     * Get version information for the SDK
     */
    getVersion(): string;
    /**
     * Health check for all services
     */
    healthCheck(): Promise<NetworkHealth>;
    /**
     * Clean up resources and close connections
     */
    close(): Promise<void>;
}
//# sourceMappingURL=client.d.ts.map