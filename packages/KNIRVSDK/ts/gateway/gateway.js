/**
 * Gateway Service for KNIRV Gateway SDK
 */
import { GatewayServiceError, } from './types';
export class GatewayService {
    constructor(config) {
        this.config = config;
    }
    updateConfig(config) {
        this.config = config;
    }
    async request(method, path, body, queryParams) {
        const url = this.buildURL(path, queryParams);
        const requestOptions = {
            method: method.toUpperCase(),
            headers: {
                'Content-Type': 'application/json',
                'User-Agent': 'KNIRV-Gateway-SDK-TS/1.0.0',
                ...this.getHeaders(),
            },
        };
        if (body) {
            requestOptions.body = JSON.stringify(body);
        }
        try {
            const response = await fetch(url, requestOptions);
            if (!response.ok) {
                throw new GatewayServiceError(`HTTP ${response.status}: ${response.statusText}`, response.status);
            }
            const data = await response.json();
            if (data.success === false) {
                throw new GatewayServiceError(data.error || 'Request failed');
            }
            return data.data || data;
        }
        catch (error) {
            if (error instanceof GatewayServiceError) {
                throw error;
            }
            throw new GatewayServiceError(`Request failed: ${error.message}`);
        }
    }
    buildURL(path, queryParams) {
        const baseURL = this.config.baseURL;
        const url = new URL(path, baseURL);
        if (queryParams) {
            Object.entries(queryParams).forEach(([key, value]) => {
                url.searchParams.set(key, value);
            });
        }
        return url.toString();
    }
    getHeaders() {
        const headers = {};
        if (this.config.apiKey) {
            headers['X-API-Key'] = this.config.apiKey;
        }
        if (this.config.environment) {
            headers['X-Environment'] = this.config.environment;
        }
        return headers;
    }
    /**
     * Get current gateway routes
     */
    async getRoutes() {
        return this.request('GET', '/gateway/routes');
    }
    /**
     * Get gateway status
     */
    async getStatus() {
        return this.request('GET', '/gateway/status');
    }
    /**
     * Test gateway connectivity to all services
     */
    async testConnectivity() {
        const results = {};
        // Test economics service
        try {
            await this.request('GET', '/economics/health');
            results.economics = true;
        }
        catch {
            results.economics = false;
        }
        // Test other KNIRV services if URLs are configured
        const services = ['knirvchain', 'knirvnexus', 'knirvoracle', 'knirvgraph'];
        for (const service of services) {
            const serviceURL = this.config.serviceURLs[service];
            if (serviceURL) {
                try {
                    const response = await fetch(`${serviceURL}/health`);
                    results[service] = response.ok;
                }
                catch {
                    results[service] = false;
                }
            }
            else {
                results[service] = false;
            }
        }
        return results;
    }
    /**
     * Get gateway configuration
     */
    async getConfiguration() {
        return this.request('GET', '/gateway/config');
    }
    /**
     * Get gateway metrics
     */
    async getMetrics() {
        return this.request('GET', '/gateway/metrics');
    }
}
export default GatewayService;
