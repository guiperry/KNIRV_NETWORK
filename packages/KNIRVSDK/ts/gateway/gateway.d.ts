/**
 * Gateway Service for KNIRV Gateway SDK
 */
import { RequestConfig, Route, GatewayStatus } from './types';
export declare class GatewayService {
    private config;
    constructor(config: RequestConfig);
    updateConfig(config: RequestConfig): void;
    private request;
    private buildURL;
    private getHeaders;
    /**
     * Get current gateway routes
     */
    getRoutes(): Promise<Route[]>;
    /**
     * Get gateway status
     */
    getStatus(): Promise<GatewayStatus>;
    /**
     * Test gateway connectivity to all services
     */
    testConnectivity(): Promise<Record<string, boolean>>;
    /**
     * Get gateway configuration
     */
    getConfiguration(): Promise<{
        version: string;
        environment: string;
        services: Record<string, string>;
        features: string[];
    }>;
    /**
     * Get gateway metrics
     */
    getMetrics(): Promise<{
        requests_total: number;
        requests_per_second: number;
        average_response_time: number;
        error_rate: number;
        uptime: string;
    }>;
}
export default GatewayService;
//# sourceMappingURL=gateway.d.ts.map