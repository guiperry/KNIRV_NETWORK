/**
 * Client module for the KNIRV Client SDK.
 *
 * This module provides the KnirvClient class for interacting with the KNIRVCHAIN network.
 */
/**
 * Base error class for KNIRV client errors.
 */
export declare class KnirvClientError extends Error {
    constructor(message: string);
}
/**
 * Error thrown when a resource is not found.
 */
export declare class ResourceNotFoundError extends KnirvClientError {
    constructor(message: string);
}
/**
 * Error thrown when connection to a peer fails.
 */
export declare class ConnectionFailedError extends KnirvClientError {
    constructor(message: string);
}
/**
 * Error thrown when fetching a resource fails.
 */
export declare class FetchFailedError extends KnirvClientError {
    constructor(message: string);
}
/**
 * Error thrown when operations are attempted on a closed client.
 */
export declare class ClientClosedError extends KnirvClientError {
    constructor(message?: string);
}
/**
 * Configuration for the KNIRV client.
 */
export interface KnirvClientConfig {
    /**
     * List of bootstrap peer multiaddresses.
     */
    bootstrapPeers: string[];
    /**
     * Port to use for the libp2p host (0 for random).
     */
    p2pPort?: number;
    /**
     * Whether to enable logging.
     */
    logEnabled?: boolean;
}
/**
 * Default configuration for the KNIRV client.
 */
export declare function defaultConfig(): KnirvClientConfig;
/**
 * Represents data fetched from a KNIRV resource.
 */
export declare class ResourceData {
    private readonly _data;
    /**
     * Create a new ResourceData.
     *
     * @param data - The raw byte data.
     */
    constructor(data: Uint8Array);
    /**
     * Get the raw byte data.
     *
     * @returns The raw byte data as a Uint8Array.
     */
    bytes(): Uint8Array;
    /**
     * Get the data as a string.
     *
     * @param encoding - The encoding to use (default: 'utf-8').
     * @returns The data as a string.
     */
    toString(encoding?: string): string;
}
/**
 * Client for interacting with the KNIRVCHAIN network.
 */
export declare class KnirvClient {
    private readonly config;
    private libp2p;
    private closed;
    private bootstrapped;
    /**
     * Create a new KnirvClient.
     *
     * @param config - The client configuration.
     */
    constructor(config: KnirvClientConfig);
    /**
     * Initialize the libp2p node.
     *
     * @returns A promise that resolves when the node is initialized.
     * @throws {KnirvClientError} If initialization fails.
     */
    private initLibp2p;
    /**
     * Register protocol handlers.
     */
    private registerProtocolHandlers;
    /**
     * Bootstrap the client by connecting to bootstrap peers and joining the DHT network.
     *
     * @returns A promise that resolves when bootstrapping is complete.
     * @throws {ClientClosedError} If the client is closed.
     * @throws {KnirvClientError} If bootstrapping fails.
     */
    bootstrap(): Promise<void>;
    /**
     * Close the client and release resources.
     *
     * @returns A promise that resolves when the client is closed.
     */
    stop(): Promise<void>;
    /**
     * Fetch a resource from the KNIRVCHAIN network.
     *
     * @param uriString - The URI string of the resource to fetch.
     * @returns A promise that resolves to a ResourceData object containing the fetched data.
     * @throws {KnirvURIError} If the URI is invalid.
     * @throws {ResourceNotFoundError} If the resource is not found.
     * @throws {ConnectionFailedError} If connection to a peer fails.
     * @throws {FetchFailedError} If fetching the resource fails.
     * @throws {ClientClosedError} If the client is closed.
     */
    fetchResource(uriString: string): Promise<ResourceData>;
    /**
     * Find providers for a resource in the DHT.
     *
     * @param id - The ID part of the resource.
     * @param resourceType - The resource type part of the resource.
     * @returns A promise that resolves to an array of peer IDs for providers of the resource.
     * @throws {ResourceNotFoundError} If no providers are found.
     */
    private findResourceProviders;
    /**
     * Fetch a resource from a specific provider.
     *
     * @param providerId - The peer ID of the provider.
     * @param uri - The parsed URI.
     * @returns A promise that resolves to the fetched data as a Uint8Array.
     * @throws {ConnectionFailedError} If connection to the peer fails.
     * @throws {FetchFailedError} If fetching the resource fails.
     */
    private fetchFromProvider;
    /**
     * Get the peer ID of this client.
     *
     * @returns The peer ID as a string.
     * @throws {KnirvClientError} If the client is not initialized.
     */
    getPeerId(): string;
    /**
     * Get the multiaddresses of this client.
     *
     * @returns An array of multiaddress strings.
     * @throws {KnirvClientError} If the client is not initialized.
     */
    getMultiaddrs(): string[];
}
//# sourceMappingURL=client.d.ts.map