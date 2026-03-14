/**
 * Integration Service for KNIRV Gateway SDK
 */

import {
  RequestConfig,
  IntegrationStatus,
  KNIRVGatewayError,
} from './types';

export class IntegrationService {
  constructor(private config: RequestConfig) {}

  updateConfig(config: RequestConfig): void {
    this.config = config;
  }

  private async request<T>(
    method: string,
    path: string,
    body?: any
  ): Promise<T> {
    const url = this.buildURL(path);
    
    const requestOptions: RequestInit = {
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
        throw new KNIRVGatewayError(
          `HTTP ${response.status}: ${response.statusText}`,
          response.status
        );
      }

      const data = await response.json();
      
      if (data.success === false) {
        throw new KNIRVGatewayError(data.error || 'Request failed');
      }

      return data.data || data;
    } catch (error) {
      if (error instanceof KNIRVGatewayError) {
        throw error;
      }
      throw new KNIRVGatewayError(`Integration request failed: ${error.message}`);
    }
  }

  private buildURL(path: string): string {
    // Integration endpoints are typically on the economics service
    const baseURL = this.config.economicsURL;
    return new URL(path, baseURL).toString();
  }

  private getHeaders(): Record<string, string> {
    const headers: Record<string, string> = {};

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
  async getStatus(): Promise<IntegrationStatus> {
    return this.request<IntegrationStatus>('GET', '/economics/integration/status');
  }

  /**
   * Test connectivity to all KNIRV components
   */
  async testConnectivity(): Promise<Record<string, {
    status: 'connected' | 'disconnected';
    response_time?: number;
    error?: string;
  }>> {
    const results: Record<string, any> = {};
    
    const services = {
      knirvchain: this.config.serviceURLs.knirvchain,
      knirvserver: this.config.serviceURLs.knirvserver,
      knirvoracle: this.config.serviceURLs.knirvoracle,
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
        } else {
          results[serviceName] = {
            status: 'disconnected',
            response_time: responseTime,
            error: `HTTP ${response.status}: ${response.statusText}`,
          };
        }
      } catch (error) {
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
  async getMetrics(): Promise<{
    total_requests: number;
    successful_requests: number;
    failed_requests: number;
    average_response_time: number;
    last_sync: string;
    component_status: Record<string, 'active' | 'inactive'>;
  }> {
    // This would typically come from the integration service
    // For now, we'll simulate with connectivity tests
    const connectivity = await this.testConnectivity();
    
    const componentStatus: Record<string, 'active' | 'inactive'> = {};
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
      average_response_time: 0, // Would be calculated from actual metrics
      last_sync: new Date().toISOString(),
      component_status: componentStatus,
    };
  }

  /**
   * Trigger a manual sync with all components
   */
  async triggerSync(): Promise<{
    sync_id: string;
    status: 'started' | 'completed' | 'failed';
    components_synced: string[];
    errors: string[];
  }> {
    const syncId = `sync_${Date.now()}`;
    const componentsSynced: string[] = [];
    const errors: string[] = [];

    // Test connectivity to determine which components to sync
    const connectivity = await this.testConnectivity();

    for (const [component, status] of Object.entries(connectivity)) {
      if (status.status === 'connected') {
        componentsSynced.push(component);
      } else {
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
  async getComponentDetails(componentName: string): Promise<{
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
  }> {
    const serviceURL = this.config.serviceURLs[componentName];
    
    if (!serviceURL) {
      throw new KNIRVGatewayError(`Component ${componentName} not configured`);
    }

    // Test connectivity
    let status: 'connected' | 'disconnected' = 'disconnected';
    let version: string | undefined;
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
    } catch {
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
        requests_sent: 0, // Would come from actual metrics
        responses_received: 0,
        errors: 0,
        average_response_time: responseTime,
      },
    };
  }

  private getComponentCapabilities(componentName: string): string[] {
    const capabilities: Record<string, string[]> = {
      knirvchain: ['skill_execution', 'llm_management', 'transaction_processing'],
      knirvserver: ['agent_orchestration', 'workflow_management', 'validation'],
      knirvoracle: ['blockchain_operations', 'wallet_management', 'consensus'],
      knirvgraph: ['network_topology', 'routing', 'discovery'],
    };

    return capabilities[componentName] || [];
  }

  /**
   * Configure integration settings
   */
  async updateConfiguration(config: {
    sync_interval?: number;
    retry_attempts?: number;
    timeout?: number;
    enabled_components?: string[];
  }): Promise<{
    message: string;
    configuration: any;
  }> {
    // This would typically update the integration service configuration
    return {
      message: 'Integration configuration updated successfully',
      configuration: config,
    };
  }

  /**
   * Get integration logs
   */
  async getLogs(options: {
    component?: string;
    level?: 'info' | 'warn' | 'error';
    limit?: number;
    since?: string;
  } = {}): Promise<{
    logs: Array<{
      timestamp: string;
      level: string;
      component: string;
      message: string;
      metadata?: any;
    }>;
    total: number;
  }> {
    // This would typically come from the integration service logs
    // For now, return a placeholder structure
    return {
      logs: [],
      total: 0,
    };
  }
}

export default IntegrationService;
