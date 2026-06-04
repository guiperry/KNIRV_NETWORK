/**
 * Integration Service for KNIRV Gateway SDK
 */
import { RequestConfig, IntegrationStatus } from './types';
export declare class IntegrationService {
    private config;
    constructor(config: RequestConfig);
    updateConfig(config: RequestConfig): void;
    private request;
    private buildURL;
    private getHeaders;
    /**
     * Get integration status with KNIRV components
     */
    getStatus(): Promise<IntegrationStatus>;
    /**
     * Test connectivity to all KNIRV components
     */
    testConnectivity(): Promise<Record<string, {
        status: 'connected' | 'disconnected';
        response_time?: number;
        error?: string;
    }>>;
    /**
     * Get integration metrics
     */
    getMetrics(): Promise<{
        total_requests: number;
        successful_requests: number;
        failed_requests: number;
        average_response_time: number;
        last_sync: string;
        component_status: Record<string, 'active' | 'inactive'>;
    }>;
    /**
     * Trigger a manual sync with all components
     */
    triggerSync(): Promise<{
        sync_id: string;
        status: 'started' | 'completed' | 'failed';
        components_synced: string[];
        errors: string[];
    }>;
    /**
     * Get component-specific integration details
     */
    getComponentDetails(componentName: string): Promise<{
        name: string;
        url: string;
        status: 'connected' | 'disconnected';
        last_contact: string;
        version?: string;
        capabilities: string[];
        metrics: {
            requests_sent: number;
            responses_received: number;
            errors: number;
            average_response_time: number;
        };
    }>;
    private getComponentCapabilities;
    /**
     * Configure integration settings
     */
    updateConfiguration(config: {
        sync_interval?: number;
        retry_attempts?: number;
        timeout?: number;
        enabled_components?: string[];
    }): Promise<{
        message: string;
        configuration: any;
    }>;
    /**
     * Get integration logs
     */
    getLogs(options?: {
        component?: string;
        level?: 'info' | 'warn' | 'error';
        limit?: number;
        since?: string;
    }): Promise<{
        logs: Array<{
            timestamp: string;
            level: string;
            component: string;
            message: string;
            metadata?: any;
        }>;
        total: number;
    }>;
}
export default IntegrationService;
//# sourceMappingURL=integration.d.ts.map