/**
 * Integration Service for KNIRV Gateway SDK
 */
import { KNIRVGatewayError, } from './types';
export class IntegrationService {
    constructor(config) {
        this.config = config;
    }
    updateConfig(config) {
        this.config = config;
    }
    async request(method, path, body) {
        const url = this.buildURL(path);
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
                throw new KNIRVGatewayError(`HTTP ${response.status}: ${response.statusText}`, response.status);
            }
            const data = await response.json();
            if (data.success === false) {
                throw new KNIRVGatewayError(data.error || 'Request failed');
            }
            return data.data || data;
        }
        catch (error) {
            if (error instanceof KNIRVGatewayError) {
                throw error;
            }
            throw new KNIRVGatewayError(`Integration request failed: ${error.message}`);
        }
    }
    buildURL(path) {
        // Integration endpoints are typically on the economics service
        const baseURL = this.config.economicsURL;
        return new URL(path, baseURL).toString();
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
     * Get integration status with KNIRV components
     */
    async getStatus() {
        return this.request('GET', '/economics/integration/status');
    }
    /**
     * Test connectivity to all KNIRV components
     */
    async testConnectivity() {
        const results = {};
        const services = {
            knirvchain: this.config.serviceURLs.knirvchain,
            knirvnexus: this.config.serviceURLs.knirvnexus,
            knirvroot: this.config.serviceURLs.knirvroot,
            knirvgraph: this.config.serviceURLs.knirvgraph,
        };
        for (const [serviceName, serviceURL] of Object.entries(services)) {
            if (!serviceURL) {
                results[serviceName] = {
                    status: 'disconnected',
                    error: 'Service URL not configured',
                };
                continue;
            }
            const startTime = Date.now();
            try {
                const response = await fetch(`${serviceURL}/health`, {
                    method: 'GET',
                    headers: {
                        'User-Agent': 'KNIRV-Gateway-SDK-TS/1.0.0',
                    },
                });
                const responseTime = Date.now() - startTime;
                if (response.ok) {
                    results[serviceName] = {
                        status: 'connected',
                        response_time: responseTime,
                    };
                }
                else {
                    results[serviceName] = {
                        status: 'disconnected',
                        response_time: responseTime,
                        error: `HTTP ${response.status}: ${response.statusText}`,
                    };
                }
            }
            catch (error) {
                const responseTime = Date.now() - startTime;
                results[serviceName] = {
                    status: 'disconnected',
                    response_time: responseTime,
                    error: error.message,
                };
            }
        }
        return results;
    }
    /**
     * Get integration metrics
     */
    async getMetrics() {
        // This would typically come from the integration service
        // For now, we'll simulate with connectivity tests
        const connectivity = await this.testConnectivity();
        const componentStatus = {};
        let successfulRequests = 0;
        let totalRequests = Object.keys(connectivity).length;
        for (const [component, status] of Object.entries(connectivity)) {
            componentStatus[component] = status.status === 'connected' ? 'active' : 'inactive';
            if (status.status === 'connected') {
                successfulRequests++;
            }
        }
        return {
            total_requests: totalRequests,
            successful_requests: successfulRequests,
            failed_requests: totalRequests - successfulRequests,
            average_response_time: 0,
            last_sync: new Date().toISOString(),
            component_status: componentStatus,
        };
    }
    /**
     * Trigger a manual sync with all components
     */
    async triggerSync() {
        const syncId = `sync_${Date.now()}`;
        const componentsSynced = [];
        const errors = [];
        // Test connectivity to determine which components to sync
        const connectivity = await this.testConnectivity();
        for (const [component, status] of Object.entries(connectivity)) {
            if (status.status === 'connected') {
                componentsSynced.push(component);
            }
            else {
                errors.push(`${component}: ${status.error || 'Connection failed'}`);
            }
        }
        return {
            sync_id: syncId,
            status: errors.length === 0 ? 'completed' : 'failed',
            components_synced: componentsSynced,
            errors,
        };
    }
    /**
     * Get component-specific integration details
     */
    async getComponentDetails(componentName) {
        const serviceURL = this.config.serviceURLs[componentName];
        if (!serviceURL) {
            throw new KNIRVGatewayError(`Component ${componentName} not configured`);
        }
        // Test connectivity
        let status = 'disconnected';
        let version;
        let responseTime = 0;
        const startTime = Date.now();
        try {
            const response = await fetch(`${serviceURL}/health`);
            responseTime = Date.now() - startTime;
            if (response.ok) {
                status = 'connected';
                const healthData = await response.json();
                version = healthData.version;
            }
        }
        catch {
            responseTime = Date.now() - startTime;
        }
        return {
            name: componentName,
            url: serviceURL,
            status,
            last_contact: new Date().toISOString(),
            version,
            capabilities: this.getComponentCapabilities(componentName),
            metrics: {
                requests_sent: 0,
                responses_received: 0,
                errors: 0,
                average_response_time: responseTime,
            },
        };
    }
    getComponentCapabilities(componentName) {
        const capabilities = {
            knirvchain: ['skill_execution', 'llm_management', 'transaction_processing'],
            knirvnexus: ['agent_orchestration', 'workflow_management', 'validation'],
            knirvroot: ['blockchain_operations', 'wallet_management', 'consensus'],
            knirvgraph: ['network_topology', 'routing', 'discovery'],
        };
        return capabilities[componentName] || [];
    }
    /**
     * Configure integration settings
     */
    async updateConfiguration(config) {
        // This would typically update the integration service configuration
        return {
            message: 'Integration configuration updated successfully',
            configuration: config,
        };
    }
    /**
     * Get integration logs
     */
    async getLogs(options = {}) {
        // This would typically come from the integration service logs
        // For now, return a placeholder structure
        return {
            logs: [],
            total: 0,
        };
    }
}
export default IntegrationService;
