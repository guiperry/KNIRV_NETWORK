/**
 * Unified KNIRV Client - Provides access to all KNIRV Network services
 */
import { KNIRVWallet } from './wallet';
export interface KNIRVClientConfig {
    gateway?: {
        environment?: 'development' | 'staging' | 'production';
        debug?: boolean;
        economicsURL?: string;
        gatewayURL?: string;
        apiKey?: string;
    };
    transaction?: {
        baseURL?: string;
        apiKey?: string;
        timeout?: number;
    };
    wallet?: {
        provider?: string;
        chainId?: string;
    };
}
/**
 * Unified KNIRV Client providing access to all KNIRV Network services
 */
export declare class KNIRVClient {
    readonly wallet: KNIRVWallet;
    constructor(config?: KNIRVClientConfig);
    /**
     * Create a client optimized for gateway operations
     */
    static createGatewayClient(config?: KNIRVClientConfig['gateway']): any;
    /**
     * Create a client optimized for transaction operations (temporarily disabled)
     */
    /**
     * Create a client optimized for transmission operations (temporarily disabled)
     */
    /**
     * Health check for all services
     */
    healthCheck(): Promise<{
        gateway: boolean;
        transaction: boolean;
        transmission: boolean;
    }>;
}
//# sourceMappingURL=client.d.ts.map