/**
 * PoAu-D (Proof of Authority using Delegation) Service Client
 * TypeScript/JavaScript SDK for interacting with the PoAu-D consensus management API
 */
import { RequestConfig } from './types';
export interface PoAuDStatus {
    enabled: boolean;
    network_authors_count?: number;
    main_pool_size?: number;
    pas_pool_size?: number;
    delegated_transactions?: number;
    delegation_stats?: Record<string, any>;
}
export interface NetworkAuthor {
    address: string;
}
export interface NetworkAuthorsResponse {
    network_authors: string[];
    count: number;
}
export interface PoAuDResponse {
    success?: boolean;
    enabled?: boolean;
    message?: string;
    address?: string;
}
export interface PoAuDConfig {
    enabled: boolean;
    delegation_interval: string;
    max_subpool_stale_time: string;
    max_pap_subpool_queue: number;
    status_advertise_interval: string;
}
export declare class NetworkAuthorsService {
    private config;
    constructor(config: RequestConfig);
    /**
     * Add an address to the Network Authors set
     */
    add(address: string): Promise<PoAuDResponse>;
    /**
     * Remove an address from the Network Authors set
     */
    remove(address: string): Promise<PoAuDResponse>;
    /**
     * List all current Network Authors
     */
    list(): Promise<NetworkAuthorsResponse>;
}
export declare class PoAuDService {
    private config;
    readonly networkAuthors: NetworkAuthorsService;
    constructor(config: RequestConfig);
    /**
     * Enable the PoAu-D consensus mechanism
     */
    enable(): Promise<PoAuDResponse>;
    /**
     * Disable the PoAu-D consensus mechanism (fallback to PoW)
     */
    disable(): Promise<PoAuDResponse>;
    /**
     * Get the current PoAu-D status and statistics
     */
    getStatus(): Promise<PoAuDStatus>;
}
/**
 * Standalone PoAu-D client for convenient access to PoAu-D operations
 */
export declare class PoAuDClient {
    private service;
    constructor(config?: Partial<RequestConfig>);
    /**
     * Enable PoAu-D consensus mechanism
     */
    enableConsensus(): Promise<PoAuDResponse>;
    /**
     * Disable PoAu-D consensus mechanism
     */
    disableConsensus(): Promise<PoAuDResponse>;
    /**
     * Get the current PoAu-D status
     */
    getConsensusStatus(): Promise<PoAuDStatus>;
    /**
     * Add a Network Author Peer
     */
    addNetworkAuthor(address: string): Promise<PoAuDResponse>;
    /**
     * Remove a Network Author Peer
     */
    removeNetworkAuthor(address: string): Promise<PoAuDResponse>;
    /**
     * List all Network Author Peers
     */
    listNetworkAuthors(): Promise<NetworkAuthorsResponse>;
    /**
     * Check if PoAu-D is currently enabled
     */
    isPoAuDEnabled(): Promise<boolean>;
    /**
     * Get the number of current Network Authors
     */
    getNetworkAuthorCount(): Promise<number>;
    /**
     * Check if an address is a Network Author
     */
    isNetworkAuthor(address: string): Promise<boolean>;
    /**
     * Get delegation statistics
     */
    getDelegationStatistics(): Promise<Record<string, any>>;
}
/**
 * Validate that an address is properly formatted for use as a Network Author
 */
export declare function validateNetworkAuthor(address: string): void;
/**
 * Get default PoAu-D configuration
 */
export declare function getDefaultPoAuDConfig(): PoAuDConfig;
export type { PoAuDStatus, NetworkAuthor, NetworkAuthorsResponse, PoAuDResponse, PoAuDConfig, };
//# sourceMappingURL=poaud.d.ts.map