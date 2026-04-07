/**
 * Gateway Service for KNIRV Gateway SDK
 */

import {
  RequestConfig,
  Route,
  GatewayStatus,
  GatewayServiceError,
} from './types';

export class GatewayService {
  constructor(private config: RequestConfig) {}

  updateConfig(config: RequestConfig): void {
    this.config = config;
  }

  private async request<T>(
    method: string,
    path: string,
    body?: any,
    queryParams?: Record<string, string>
  ): Promise<T> {
    const url = this.buildURL(path, queryParams);
    
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
        throw new GatewayServiceError(
          `HTTP ${response.status}: ${response.statusText}`,
          response.status
        );
      }

      const data = await response.json();
      
      if (data.success === false) {
        throw new GatewayServiceError(data.error || 'Request failed');
      }

      return data.data || data;
    } catch (error) {
      if (error instanceof GatewayServiceError) {
        throw error;
      }
      throw new GatewayServiceError(`Request failed: ${error.message}`);
    }
  }

  private buildURL(path: string, queryParams?: Record<string, string>): string {
    const baseURL = this.config.baseURL;
    const url = new URL(path, baseURL);
    
    if (queryParams) {
      Object.entries(queryParams).forEach(([key, value]) => {
        url.searchParams.set(key, value);
      });
    }

    return url.toString();
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
   * Get current gateway routes
   */
  async getRoutes(): Promise<Route[]> {
    return this.request<Route[]>('GET', '/gateway/routes');
  }

  /**
   * Get gateway status
   */
  async getStatus(): Promise<GatewayStatus> {
    return this.request<GatewayStatus>('GET', '/gateway/status');
  }

  /**
   * Test gateway connectivity to all services
   */
  async testConnectivity(): Promise<Record<string, boolean>> {
    const results: Record<string, boolean> = {};
    
    // Test economics service
    try {
      await this.request('GET', '/economics/health');
      results.economics = true;
    } catch {
      results.economics = false;
    }

    // Test other KNIRV services if URLs are configured
    const services = ['knirvchain', 'knirvserver', 'knirvoracle', 'knirvgraph'];
    
    for (const service of services) {
      const serviceURL = this.config.serviceURLs[service];
      if (serviceURL) {
        try {
          const response = await fetch(`${serviceURL}/health`);
          results[service] = response.ok;
        } catch {
          results[service] = false;
        }
      } else {
        results[service] = false;
      }
    }

    return results;
  }

  /**
   * Get gateway configuration
   */
  async getConfiguration(): Promise<{
    version: string;
    environment: string;
    services: Record<string, string>;
    features: string[];
  }> {
    return this.request('GET', '/gateway/config');
  }

  /**
   * Get gateway metrics
   */
  async getMetrics(): Promise<{
    requests_total: number;
    requests_per_second: number;
    average_response_time: number;
    error_rate: number;
    uptime: string;
  }> {
    return this.request('GET', '/gateway/metrics');
  }
}

export default GatewayService;
