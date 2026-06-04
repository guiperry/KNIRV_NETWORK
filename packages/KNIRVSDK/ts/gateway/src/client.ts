/**
 * KNIRV Gateway SDK Client
 */

import { EconomicsService } from './economics';
import { GatewayService } from './gateway';
import { HealthService } from './health';
import { IntegrationService } from './integration';
import { PoAuDService } from './poaud';
import {
  ClientOptions,
  RequestConfig,
  APIResponse,
  RequestInterceptor,
  ResponseInterceptor,
  KNIRVAPIError,
  KNIRVValidationError,
  KNIRVConnectionError,
  KNIRVTimeoutError,
} from './types';

export class KNIRVGatewayClient {
  public readonly economics: EconomicsService;
  public readonly gateway: GatewayService;
  public readonly health: HealthService;
  public readonly integration: IntegrationService;
  public readonly poaud: PoAuDService;

  private config: Required<ClientOptions>;
  private requestInterceptors: RequestInterceptor[] = [];
  private responseInterceptors: ResponseInterceptor[] = [];

  constructor(options: Partial<ClientOptions> = {}) {
    // Set defaults and handle baseUrl/baseURL compatibility
    const baseURL = options.baseURL || options.baseUrl || 'https://gateway.knirv.com';

    // Validate that baseURL is not empty (including when explicitly set to empty string)
    if (options.baseURL === '' || options.baseUrl === '' || !baseURL || baseURL.trim() === '') {
      throw new Error('baseURL is required');
    }
    
    this.config = {
      baseURL,
      baseUrl: baseURL, // For compatibility
      economicsURL: options.economicsURL || `${baseURL}/economics`,
      apiKey: options.apiKey || '',
      nrnContract: options.nrnContract || '',
      timeout: options.timeout || 30000,
      retries: options.retries || 3,
      retryDelay: options.retryDelay || 1000,
      maxRetryDelay: options.maxRetryDelay || 10000,
      userAgent: options.userAgent || 'knirv-gateway-sdk-ts/1.0.0',
      defaultHeaders: options.defaultHeaders || {},
    };



    // Initialize services
    this.economics = new EconomicsService(this.config, this);
    this.gateway = new GatewayService(this.config, this);
    this.health = new HealthService(this.config, this);
    this.integration = new IntegrationService(this.config, this);
    this.poaud = new PoAuDService(this.config, this);
  }

  // Getter for baseURL (for test compatibility)
  get baseURL(): string {
    return this.config.baseURL;
  }

  // Getter for baseUrl (for test compatibility)
  get baseUrl(): string {
    return this.config.baseURL;
  }

  // Private getter for API key (for test compatibility)
  get _apiKey(): string {
    return this.config.apiKey;
  }

  /**
   * Add a request interceptor
   */
  addRequestInterceptor(interceptor: RequestInterceptor): void {
    this.requestInterceptors.push(interceptor);
  }

  /**
   * Add a response interceptor
   */
  addResponseInterceptor(interceptor: ResponseInterceptor): void {
    this.responseInterceptors.push(interceptor);
  }

  /**
   * Make an HTTP request with retry logic
   */
  async request<T = any>(config: RequestConfig): Promise<APIResponse<T>> {
    let requestConfig = { ...config };

    // Apply request interceptors
    for (const interceptor of this.requestInterceptors) {
      requestConfig = await interceptor(requestConfig);
    }

    // Retry logic
    let lastError: Error | null = null;
    for (let attempt = 0; attempt <= this.config.retries; attempt++) {
      try {
        return await this.makeRequest<T>(requestConfig);
      } catch (error) {
        lastError = error as Error;

        // Don't retry on the last attempt
        if (attempt === this.config.retries) {
          break;
        }

        // Only retry on 5xx errors
        if (error instanceof KNIRVAPIError && error.status && error.status >= 500) {
          // Wait before retrying
          await new Promise(resolve => setTimeout(resolve, this.config.retryDelay * (attempt + 1)));
          continue;
        }

        // Don't retry on other errors
        throw error;
      }
    }

    throw lastError;
  }

  /**
   * Make a single HTTP request attempt
   */
  private async makeRequest<T = any>(requestConfig: RequestConfig): Promise<APIResponse<T>> {

    // Build URL
    const url = requestConfig.url?.startsWith('http') 
      ? requestConfig.url 
      : `${this.config.baseURL}${requestConfig.url || ''}`;

    // Build headers
    const headers: Record<string, string> = {
      'Content-Type': 'application/json',
      'User-Agent': this.config.userAgent,
      ...this.config.defaultHeaders,
      ...requestConfig.headers,
    };

    if (this.config.apiKey) {
      headers['Authorization'] = `Bearer ${this.config.apiKey}`;
    }

    // Build request options
    const requestOptions: RequestInit = {
      method: requestConfig.method || 'GET',
      headers,
      signal: requestConfig.signal,
    };

    if (requestConfig.data && requestConfig.method !== 'GET') {
      requestOptions.body = JSON.stringify(requestConfig.data);
    }

    // Add query parameters
    const urlWithParams = new URL(url);
    if (requestConfig.params) {
      Object.entries(requestConfig.params).forEach(([key, value]) => {
        if (value !== undefined && value !== null) {
          if (Array.isArray(value)) {
            value.forEach(v => urlWithParams.searchParams.append(key, String(v)));
          } else {
            urlWithParams.searchParams.append(key, String(value));
          }
        }
      });
    }

    try {
      // Set timeout
      const controller = new AbortController();
      const timeoutId = setTimeout(() => controller.abort(), requestConfig.timeout || this.config.timeout);

      if (requestConfig.signal) {
        requestConfig.signal.addEventListener('abort', () => controller.abort());
      }

      requestOptions.signal = controller.signal;

      const response = await fetch(urlWithParams.toString(), requestOptions);
      clearTimeout(timeoutId);

      let data: T;
      const contentType = response.headers.get('content-type');

      // Try JSON first if content-type suggests it or if json method exists
      if ((contentType && contentType.includes('application/json')) ||
          (typeof response.json === 'function' && !contentType)) {
        try {
          data = await response.json();
        } catch (error) {
          // Fallback to text if JSON parsing fails
          if (typeof response.text === 'function') {
            data = await response.text() as any;
          } else {
            data = '' as any;
          }
        }
      } else {
        // Handle cases where response.text might not exist (like in mocks)
        if (typeof response.text === 'function') {
          data = await response.text() as any;
        } else {
          data = '' as any;
        }
      }

      const apiResponse: APIResponse<T> = {
        data,
        status: response.status,
        statusText: response.statusText,
        headers: Object.fromEntries(response.headers.entries()),
      };

      if (!response.ok) {
        // Convert 400 errors to KNIRVValidationError
        if (response.status === 400) {
          throw new KNIRVValidationError(
            `Request failed with status ${response.status}: ${response.statusText}`
          );
        }

        throw new KNIRVAPIError(
          `Request failed with status ${response.status}: ${response.statusText}`,
          response.status,
          apiResponse,
          requestConfig
        );
      }

      // Apply response interceptors
      let finalResponse = apiResponse;
      for (const interceptor of this.responseInterceptors) {
        finalResponse = await interceptor(finalResponse);
      }

      return finalResponse;
    } catch (error) {
      if (error instanceof DOMException && error.name === 'AbortError') {
        throw new KNIRVTimeoutError('Request timeout');
      }

      if (error instanceof TypeError && error.message.includes('fetch')) {
        throw new KNIRVConnectionError('Network connection error');
      }

      // Convert generic errors to KNIRVAPIError
      if (error instanceof Error && !(error instanceof KNIRVAPIError)) {
        throw new KNIRVAPIError(error.message);
      }

      throw error;
    }
  }

  /**
   * GET request helper
   */
  async get<T = any>(url: string, config: Omit<RequestConfig, 'method' | 'url'> = {}): Promise<T> {
    const response = await this.request<T>({ ...config, method: 'GET', url });

    // If response interceptors have added properties, merge them with the data
    const responseKeys = Object.keys(response);
    const standardKeys = ['data', 'status', 'statusText', 'headers'];
    const interceptorKeys = responseKeys.filter(key => !standardKeys.includes(key));

    if (interceptorKeys.length > 0) {
      // Merge interceptor properties with the data
      const interceptorProps = interceptorKeys.reduce((acc, key) => {
        acc[key] = (response as any)[key];
        return acc;
      }, {} as any);

      if (typeof response.data === 'object' && response.data !== null) {
        return { ...response.data, ...interceptorProps };
      } else {
        return { ...interceptorProps, data: response.data } as T;
      }
    }

    return response.data;
  }

  /**
   * POST request helper
   */
  async post<T = any>(url: string, data?: any, config: Omit<RequestConfig, 'method' | 'url' | 'data'> = {}): Promise<T> {
    const response = await this.request<T>({ ...config, method: 'POST', url, data });
    return response.data;
  }

  /**
   * PUT request helper
   */
  async put<T = any>(url: string, data?: any, config: Omit<RequestConfig, 'method' | 'url' | 'data'> = {}): Promise<T> {
    const response = await this.request<T>({ ...config, method: 'PUT', url, data });
    return response.data;
  }

  /**
   * DELETE request helper
   */
  async delete<T = any>(url: string, config: Omit<RequestConfig, 'method' | 'url'> = {}): Promise<T> {
    const response = await this.request<T>({ ...config, method: 'DELETE', url });
    return response.data;
  }
}
