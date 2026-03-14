/**
 * Health Service for KNIRV Gateway SDK
 */
import { KNIRVGatewayError, } from './types';
export class HealthService {
    constructor(config) {
        this.config = config;
    }
    updateConfig(config) {
        this.config = config;
    }
    async request(method, path, baseURL) {
        const url = baseURL ? `${baseURL}${path}` : this.buildURL(path);
        const requestOptions = {
            method: method.toUpperCase(),
            headers: {
                'User-Agent': 'KNIRV-Gateway-SDK-TS/1.0.0',
                ...this.getHeaders(),
            },
        };
        try {
            const response = await fetch(url, requestOptions);
            if (!response.ok) {
                throw new KNIRVGatewayError(`HTTP ${response.status}: ${response.statusText}`, response.status);
            }
            const data = await response.json();
            return data.data || data;
        }
        catch (error) {
            if (error instanceof KNIRVGatewayError) {
                throw error;
            }
            throw new KNIRVGatewayError(`Health check failed: ${error.message}`);
        }
    }
    buildURL(path) {
        // For health checks, prefer economics URL for /economics paths
        let baseURL = this.config.baseURL;
        if (path.startsWith('/economics')) {
            baseURL = this.config.economicsURL;
        }
        return new URL(path, baseURL).toString();
    }
    getHeaders() {
        const headers = {};
        if (this.config.environment) {
            headers['X-Environment'] = this.config.environment;
        }
        return headers;
    }
    /**
     * Check health of the economics service
     */
    async checkEconomicsHealth() {
        return this.request('GET', '/economics/health');
    }
    /**
     * Check health of the API gateway
     */
    async checkGatewayHealth() {
        return this.request('GET', '/health');
    }
    /**
     * Check health of all KNIRV services
     */
    async checkAllServices() {
        const results = {};
        // Check economics service
        try {
            results.economics = await this.checkEconomicsHealth();
        }
        catch {
            results.economics = null;
        }
        // Check API gateway
        try {
            results.gateway = await this.checkGatewayHealth();
        }
        catch {
            results.gateway = null;
        }
        // Check other KNIRV services
        const services = {
            knirvchain: this.config.serviceURLs.knirvchain,
            knirvnexus: this.config.serviceURLs.knirvnexus,
            knirvoracle: this.config.serviceURLs.knirvoracle,
            knirvgraph: this.config.serviceURLs.knirvgraph,
        };
        for (const [serviceName, serviceURL] of Object.entries(services)) {
            if (serviceURL) {
                try {
                    results[serviceName] = await this.request('GET', '/health', serviceURL);
                }
                catch {
                    results[serviceName] = null;
                }
            }
            else {
                results[serviceName] = null;
            }
        }
        return results;
    }
    /**
     * Get overall system health status
     */
    async getSystemHealth() {
        const serviceChecks = await this.checkAllServices();
        const services = {};
        let healthyCount = 0;
        let totalCount = 0;
        for (const [serviceName, healthCheck] of Object.entries(serviceChecks)) {
            totalCount++;
            if (healthCheck && healthCheck.status === 'healthy') {
                services[serviceName] = 'healthy';
                healthyCount++;
            }
            else {
                services[serviceName] = 'unhealthy';
            }
        }
        let overallStatus;
        if (healthyCount === totalCount) {
            overallStatus = 'healthy';
        }
        else if (healthyCount > totalCount / 2) {
            overallStatus = 'degraded';
        }
        else {
            overallStatus = 'unhealthy';
        }
        return {
            status: overallStatus,
            services,
            timestamp: new Date().toISOString(),
        };
    }
    /**
     * Wait for a service to become healthy
     */
    async waitForService(serviceName, options = {}) {
        const { timeout = 60000, interval = 2000 } = options;
        const startTime = Date.now();
        while (Date.now() - startTime < timeout) {
            try {
                let isHealthy = false;
                switch (serviceName) {
                    case 'economics':
                        const economicsHealth = await this.checkEconomicsHealth();
                        isHealthy = economicsHealth.status === 'healthy';
                        break;
                    case 'gateway':
                        const gatewayHealth = await this.checkGatewayHealth();
                        isHealthy = gatewayHealth.status === 'healthy';
                        break;
                    default:
                        const serviceURL = this.config.serviceURLs[serviceName];
                        if (serviceURL) {
                            const serviceHealth = await this.request('GET', '/health', serviceURL);
                            isHealthy = serviceHealth.status === 'healthy';
                        }
                }
                if (isHealthy) {
                    return true;
                }
            }
            catch {
                // Service is not healthy, continue waiting
            }
            await new Promise(resolve => setTimeout(resolve, interval));
        }
        return false;
    }
    /**
     * Get detailed health information for debugging
     */
    async getDetailedHealth() {
        const timestamp = new Date().toISOString();
        const services = {};
        const serviceChecks = [
            { name: 'economics', check: () => this.checkEconomicsHealth() },
            { name: 'gateway', check: () => this.checkGatewayHealth() },
        ];
        // Add KNIRV service checks
        for (const [serviceName, serviceURL] of Object.entries(this.config.serviceURLs)) {
            if (serviceURL) {
                serviceChecks.push({
                    name: serviceName,
                    check: () => this.request('GET', '/health', serviceURL),
                });
            }
        }
        for (const { name, check } of serviceChecks) {
            const startTime = Date.now();
            try {
                const result = await check();
                const responseTime = Date.now() - startTime;
                services[name] = {
                    status: 'healthy',
                    response_time: responseTime,
                    details: result,
                };
            }
            catch (error) {
                const responseTime = Date.now() - startTime;
                services[name] = {
                    status: 'unhealthy',
                    response_time: responseTime,
                    error: error.message,
                };
            }
        }
        return {
            timestamp,
            environment: this.config.environment,
            services,
        };
    }
}
export default HealthService;
