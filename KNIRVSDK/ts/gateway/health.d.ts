/**
 * Health Service for KNIRV Gateway SDK
 */
import { RequestConfig, HealthCheckResponse } from './types';
export declare class HealthService {
    private config;
    constructor(config: RequestConfig);
    updateConfig(config: RequestConfig): void;
    private request;
    private buildURL;
    private getHeaders;
    /**
     * Check health of the economics service
     */
    checkEconomicsHealth(): Promise<HealthCheckResponse>;
    /**
     * Check health of the API gateway
     */
    checkGatewayHealth(): Promise<HealthCheckResponse>;
    /**
     * Check health of all KNIRV services
     */
    checkAllServices(): Promise<Record<string, HealthCheckResponse | null>>;
    /**
     * Get overall system health status
     */
    getSystemHealth(): Promise<{
        status: 'healthy' | 'degraded' | 'unhealthy';
        services: Record<string, 'healthy' | 'unhealthy'>;
        timestamp: string;
    }>;
    /**
     * Wait for a service to become healthy
     */
    waitForService(serviceName: 'economics' | 'gateway' | 'knirvchain' | 'knirvnexus' | 'knirvoracle' | 'knirvgraph', options?: {
        timeout?: number;
        interval?: number;
    }): Promise<boolean>;
    /**
     * Get detailed health information for debugging
     */
    getDetailedHealth(): Promise<{
        timestamp: string;
        environment: string;
        services: Record<string, {
            status: 'healthy' | 'unhealthy';
            response_time?: number;
            error?: string;
            details?: any;
        }>;
    }>;
}
export default HealthService;
//# sourceMappingURL=health.d.ts.map