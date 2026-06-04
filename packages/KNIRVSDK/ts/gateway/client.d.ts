/**
 * KNIRV Gateway SDK Client for TypeScript/JavaScript
 * Provides access to KNIRVGATEWAY services including Economics and API Gateway
 */
import { EconomicsService } from './economics';
import { GatewayService } from './gateway';
import { HealthService } from './health';
import { IntegrationService } from './integration';
import { PoAuDService } from './poaud';
import { RequestConfig, ClientOptions } from './types';
export declare class KNIRVGatewayClient {
    private config;
    readonly economics: EconomicsService;
    readonly gateway: GatewayService;
    readonly health: HealthService;
    readonly integration: IntegrationService;
    readonly poaud: PoAuDService;
    constructor(options?: Partial<ClientOptions>);
    /**
     * Create a client specifically for the Economics Service
     */
    static createEconomicsClient(options?: Partial<ClientOptions>): KNIRVGatewayClient;
    /**
     * Create a client specifically for the API Gateway
     */
    static createGatewayClient(options?: Partial<ClientOptions>): KNIRVGatewayClient;
    /**
     * Update client configuration
     */
    updateConfig(options: Partial<ClientOptions>): void;
    /**
     * Get current client configuration
     */
    getConfig(): RequestConfig;
    /**
     * Make a raw HTTP request
     */
    request<T = any>(method: string, path: string, options?: {
        body?: any;
        headers?: Record<string, string>;
        queryParams?: Record<string, string>;
    }): Promise<T>;
    /**
     * Make a GET request
     */
    get<T = any>(path: string, queryParams?: Record<string, string>): Promise<T>;
    /**
     * Make a POST request
     */
    post<T = any>(path: string, body?: any): Promise<T>;
    /**
     * Make a PUT request
     */
    put<T = any>(path: string, body?: any): Promise<T>;
    /**
     * Make a DELETE request
     */
    delete<T = any>(path: string): Promise<T>;
    private buildURL;
    private getDefaultHeaders;
}
export * from './types';
export * from './economics';
export * from './gateway';
export * from './health';
export * from './integration';
export default KNIRVGatewayClient;
//# sourceMappingURL=client.d.ts.map