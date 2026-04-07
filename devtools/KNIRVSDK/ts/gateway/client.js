/**
 * KNIRV Gateway SDK Client for TypeScript/JavaScript
 * Provides access to KNIRVGATEWAY services including Economics and API Gateway
 */
import { EconomicsService } from './economics';
import { GatewayService } from './gateway';
import { HealthService } from './health';
import { IntegrationService } from './integration';
import { PoAuDService } from './poaud';
import { defaultClientOptions } from './types';
export class KNIRVGatewayClient {
    constructor(options = {}) {
        this.config = { ...defaultClientOptions(), ...options };
        // Initialize services
        this.economics = new EconomicsService(this.config);
        this.gateway = new GatewayService(this.config);
        this.health = new HealthService(this.config);
        this.integration = new IntegrationService(this.config);
        this.poaud = new PoAuDService(this.config);
    }
    /**
     * Create a client specifically for the Economics Service
     */
    static createEconomicsClient(options = {}) {
        const economicsOptions = {
            baseURL: process.env.ECONOMICS_SERVICE_URL || 'http://localhost:8090',
            ...options
        };
        return new KNIRVGatewayClient(economicsOptions);
    }
    /**
     * Create a client specifically for the API Gateway
     */
    static createGatewayClient(options = {}) {
        const gatewayOptions = {
            baseURL: process.env.GATEWAY_SERVICE_URL || 'http://localhost:8000',
            ...options
        };
        return new KNIRVGatewayClient(gatewayOptions);
    }
    /**
     * Update client configuration
     */
    updateConfig(options) {
        this.config = { ...this.config, ...options };
        // Update service configurations
        this.economics.updateConfig(this.config);
        this.gateway.updateConfig(this.config);
        this.health.updateConfig(this.config);
        this.integration.updateConfig(this.config);
    }
    /**
     * Get current client configuration
     */
    getConfig() {
        return { ...this.config };
    }
    /**
     * Make a raw HTTP request
     */
    async request(method, path, options = {}) {
        const url = this.buildURL(path, options.queryParams);
        const requestOptions = {
            method: method.toUpperCase(),
            headers: {
                'Content-Type': 'application/json',
                'User-Agent': 'KNIRV-Gateway-SDK-TS/1.0.0',
                ...this.getDefaultHeaders(),
                ...options.headers,
            },
        };
        if (options.body) {
            requestOptions.body = JSON.stringify(options.body);
        }
        const response = await fetch(url, requestOptions);
        if (!response.ok) {
            throw new Error(`HTTP ${response.status}: ${response.statusText}`);
        }
        const data = await response.json();
        if (data.success === false) {
            throw new Error(data.error || 'Request failed');
        }
        return data.data || data;
    }
    /**
     * Make a GET request
     */
    async get(path, queryParams) {
        return this.request('GET', path, { queryParams });
    }
    /**
     * Make a POST request
     */
    async post(path, body) {
        return this.request('POST', path, { body });
    }
    /**
     * Make a PUT request
     */
    async put(path, body) {
        return this.request('PUT', path, { body });
    }
    /**
     * Make a DELETE request
     */
    async delete(path) {
        return this.request('DELETE', path);
    }
    buildURL(path, queryParams) {
        // Determine which base URL to use
        let baseURL = this.config.baseURL;
        if (path.startsWith('/economics') && this.config.economicsURL) {
            baseURL = this.config.economicsURL;
        }
        const url = new URL(path, baseURL);
        if (queryParams) {
            Object.entries(queryParams).forEach(([key, value]) => {
                url.searchParams.set(key, value);
            });
        }
        return url.toString();
    }
    getDefaultHeaders() {
        const headers = {};
        if (this.config.apiKey) {
            headers['X-API-Key'] = this.config.apiKey;
        }
        if (this.config.nrnContract) {
            headers['X-NRN-Contract'] = this.config.nrnContract;
        }
        if (this.config.environment) {
            headers['X-Environment'] = this.config.environment;
        }
        if (this.config.debug) {
            headers['X-Debug'] = 'true';
        }
        if (this.config.verbose) {
            headers['X-Verbose'] = 'true';
        }
        return headers;
    }
}
// Export the main client class and types
export * from './types';
export * from './economics';
export * from './gateway';
export * from './health';
export * from './integration';
// Default export
export default KNIRVGatewayClient;
