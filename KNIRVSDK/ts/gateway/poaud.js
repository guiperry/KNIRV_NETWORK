/**
 * PoAu-D (Proof of Authority using Delegation) Service Client
 * TypeScript/JavaScript SDK for interacting with the PoAu-D consensus management API
 */
export class NetworkAuthorsService {
    constructor(config) {
        this.config = config;
    }
    /**
     * Add an address to the Network Authors set
     */
    async add(address) {
        const response = await fetch(`${this.config.baseURL}/poaud/network-authors/add`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
                ...this.config.headers,
            },
            body: JSON.stringify({ address }),
        });
        if (!response.ok) {
            throw new Error(`Failed to add network author: ${response.statusText}`);
        }
        return response.json();
    }
    /**
     * Remove an address from the Network Authors set
     */
    async remove(address) {
        const response = await fetch(`${this.config.baseURL}/poaud/network-authors/remove`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
                ...this.config.headers,
            },
            body: JSON.stringify({ address }),
        });
        if (!response.ok) {
            throw new Error(`Failed to remove network author: ${response.statusText}`);
        }
        return response.json();
    }
    /**
     * List all current Network Authors
     */
    async list() {
        const response = await fetch(`${this.config.baseURL}/poaud/network-authors`, {
            method: 'GET',
            headers: this.config.headers,
        });
        if (!response.ok) {
            throw new Error(`Failed to list network authors: ${response.statusText}`);
        }
        return response.json();
    }
}
export class PoAuDService {
    constructor(config) {
        this.config = config;
        this.networkAuthors = new NetworkAuthorsService(config);
    }
    /**
     * Enable the PoAu-D consensus mechanism
     */
    async enable() {
        const response = await fetch(`${this.config.baseURL}/poaud/enable`, {
            method: 'POST',
            headers: this.config.headers,
        });
        if (!response.ok) {
            throw new Error(`Failed to enable PoAu-D: ${response.statusText}`);
        }
        return response.json();
    }
    /**
     * Disable the PoAu-D consensus mechanism (fallback to PoW)
     */
    async disable() {
        const response = await fetch(`${this.config.baseURL}/poaud/disable`, {
            method: 'POST',
            headers: this.config.headers,
        });
        if (!response.ok) {
            throw new Error(`Failed to disable PoAu-D: ${response.statusText}`);
        }
        return response.json();
    }
    /**
     * Get the current PoAu-D status and statistics
     */
    async getStatus() {
        const response = await fetch(`${this.config.baseURL}/poaud/status`, {
            method: 'GET',
            headers: this.config.headers,
        });
        if (!response.ok) {
            throw new Error(`Failed to get PoAu-D status: ${response.statusText}`);
        }
        return response.json();
    }
}
/**
 * Standalone PoAu-D client for convenient access to PoAu-D operations
 */
export class PoAuDClient {
    constructor(config = {}) {
        const defaultConfig = {
            baseURL: process.env.KNIRVGATEWAY_BASE_URL || 'http://localhost:8000',
            headers: {
                'Content-Type': 'application/json',
            },
        };
        if (process.env.KNIRVGATEWAY_API_KEY) {
            defaultConfig.headers['Authorization'] = `Bearer ${process.env.KNIRVGATEWAY_API_KEY}`;
        }
        const finalConfig = { ...defaultConfig, ...config };
        this.service = new PoAuDService(finalConfig);
    }
    /**
     * Enable PoAu-D consensus mechanism
     */
    async enableConsensus() {
        return this.service.enable();
    }
    /**
     * Disable PoAu-D consensus mechanism
     */
    async disableConsensus() {
        return this.service.disable();
    }
    /**
     * Get the current PoAu-D status
     */
    async getConsensusStatus() {
        return this.service.getStatus();
    }
    /**
     * Add a Network Author Peer
     */
    async addNetworkAuthor(address) {
        return this.service.networkAuthors.add(address);
    }
    /**
     * Remove a Network Author Peer
     */
    async removeNetworkAuthor(address) {
        return this.service.networkAuthors.remove(address);
    }
    /**
     * List all Network Author Peers
     */
    async listNetworkAuthors() {
        return this.service.networkAuthors.list();
    }
    /**
     * Check if PoAu-D is currently enabled
     */
    async isPoAuDEnabled() {
        const status = await this.getConsensusStatus();
        return status.enabled;
    }
    /**
     * Get the number of current Network Authors
     */
    async getNetworkAuthorCount() {
        const authors = await this.listNetworkAuthors();
        return authors.count;
    }
    /**
     * Check if an address is a Network Author
     */
    async isNetworkAuthor(address) {
        const authors = await this.listNetworkAuthors();
        return authors.network_authors.includes(address);
    }
    /**
     * Get delegation statistics
     */
    async getDelegationStatistics() {
        const status = await this.getConsensusStatus();
        const stats = {
            enabled: status.enabled,
            network_authors_count: status.network_authors_count || 0,
            main_pool_size: status.main_pool_size || 0,
            pas_pool_size: status.pas_pool_size || 0,
            delegated_transactions: status.delegated_transactions || 0,
        };
        if (status.delegation_stats) {
            Object.assign(stats, status.delegation_stats);
        }
        return stats;
    }
}
/**
 * Validate that an address is properly formatted for use as a Network Author
 */
export function validateNetworkAuthor(address) {
    if (!address) {
        throw new Error('Network author address cannot be empty');
    }
    if (address.length < 10) {
        throw new Error(`Network author address too short: ${address}`);
    }
    // Add more validation as needed based on KNIRV address format
}
/**
 * Get default PoAu-D configuration
 */
export function getDefaultPoAuDConfig() {
    return {
        enabled: false,
        delegation_interval: '10s',
        max_subpool_stale_time: '5m',
        max_pap_subpool_queue: 100,
        status_advertise_interval: '30m',
    };
}
