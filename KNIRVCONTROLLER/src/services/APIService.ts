/**
 * Centralized API Service
 * Handles all API communications with error handling and retries
 */

export interface APIError {
  code: string;
  message: string;
  details?: Record<string, unknown>;
}

export interface APIResponse<T = unknown> {
  success: boolean;
  data?: T;
  error?: APIError;
}

export interface RequestOptions {
  method?: 'GET' | 'POST' | 'PUT' | 'DELETE' | 'PATCH';
  headers?: Record<string, string>;
  body?: unknown;
  timeout?: number;
  retries?: number;
}

export class APIService {
  private baseURL: string;
  private defaultHeaders: Record<string, string>;
  private maxRetries: number;
  private timeout: number;

  constructor(baseURL: string = this.detectBaseURL()) {
    this.baseURL = baseURL;
    this.defaultHeaders = {
      'Content-Type': 'application/json',
      'Accept': 'application/json'
    };
    this.maxRetries = 3;
    this.timeout = 10000; // 10 seconds
  }

  /**
   * Auto-detect the appropriate base URL based on environment
   */
  private detectBaseURL(): string {
    // Check if running in development
    if (typeof window !== 'undefined' && window.location) {
      if (window.location.hostname === 'localhost') {
        return 'http://localhost:3001/api'; // Local development
      }
      if (window.location.hostname.includes('testnet')) {
        return 'https://api-testnet.knirv.com'; // Testnet
      }
    }

    // Production fallback
    return 'https://api.knirv.com';
  }

  /**
   * Make an authenticated request with automatic retries
   */
  async request<T = unknown>(
    endpoint: string,
    options: RequestOptions = {}
  ): Promise<APIResponse<T>> {
    const {
      method = 'GET',
      headers = {},
      body,
      timeout = this.timeout,
      retries = this.maxRetries
    } = options;

    const url = endpoint.startsWith('http') ? endpoint : `${this.baseURL}${endpoint}`;
    const requestHeaders = { ...this.defaultHeaders, ...headers };

    // Add authentication header if available
    const authToken = this.getAuthToken();
    if (authToken) {
      requestHeaders['Authorization'] = `Bearer ${authToken}`;
    }

    const requestBody = body ? JSON.stringify(body) : undefined;

    for (let attempt = 0; attempt <= retries; attempt++) {
      try {
        const controller = new AbortController();
        const timeoutId = setTimeout(() => controller.abort(), timeout);

        const response = await fetch(url, {
          method,
          headers: requestHeaders,
          body: requestBody,
          signal: controller.signal
        });

        clearTimeout(timeoutId);

        if (!response.ok) {
          const errorData = await response.json().catch(() => ({}));
          throw new Error(`HTTP ${response.status}: ${response.statusText}`, {
            cause: {
              status: response.status,
              statusText: response.statusText,
              data: errorData
            }
          });
        }

        const responseData = await response.json();
        return {
          success: true,
          data: responseData
        };

        } catch (error) {
          const isLastAttempt = attempt === retries;
          const shouldRetry = !isLastAttempt && this.shouldRetry(error);

          if (shouldRetry) {
            await this.delay(this.getRetryDelay(attempt));
            continue;
          }

          // Handle different types of errors
          const fallbackResponse: APIResponse<T> = {
            success: false,
            error: {
              code: 'UNKNOWN_ERROR',
              message: 'An unknown error occurred'
            }
          };
          return fallbackResponse;
        }
    }

    // This should never be reached, but TypeScript requires it
    return {
      success: false,
      error: {
        code: 'UNKNOWN_ERROR',
        message: 'An unknown error occurred'
      }
    };
  }

  /**
   * GET request
   */
  async get<T = unknown>(endpoint: string, options: Omit<RequestOptions, 'method' | 'body'> = {}): Promise<APIResponse<T>> {
    return this.request<T>(endpoint, { ...options, method: 'GET' });
  }

  /**
   * POST request
   */
  async post<T = unknown>(endpoint: string, body?: unknown, options: Omit<RequestOptions, 'method' | 'body'> = {}): Promise<APIResponse<T>> {
    return this.request<T>(endpoint, { ...options, method: 'POST', body });
  }

  /**
   * PUT request
   */
  async put<T = unknown>(endpoint: string, body?: unknown, options: Omit<RequestOptions, 'method' | 'body'> = {}): Promise<APIResponse<T>> {
    return this.request<T>(endpoint, { ...options, method: 'PUT', body });
  }

  /**
   * DELETE request
   */
  async delete<T = unknown>(endpoint: string, options: Omit<RequestOptions, 'method' | 'body'> = {}): Promise<APIResponse<T>> {
    return this.request<T>(endpoint, { ...options, method: 'DELETE' });
  }

  /**
   * PATCH request
   */
  async patch<T = unknown>(endpoint: string, body?: unknown, options: Omit<RequestOptions, 'method' | 'body'> = {}): Promise<APIResponse<T>> {
    return this.request<T>(endpoint, { ...options, method: 'PATCH', body });
  }

  /**
   * Get authentication token from storage
   * This can be extended to get tokens from various sources
   */
  private getAuthToken(): string | null {
    if (typeof window !== 'undefined' && window.localStorage) {
      return localStorage.getItem('knirv_auth_token');
    }
    return null;
  }

  /**
   * Determine if a request should be retried based on the error
   */
  private shouldRetry(error: unknown): boolean {
    if (error instanceof Error && error.name === 'AbortError') {
      return false; // Don't retry on timeout
    }

    if (error instanceof Error && error.cause) {
      const cause = error.cause as { status?: number; statusText?: string; data?: unknown };
      const status = cause.status;

      // Retry on 5xx errors and network errors (undefined status indicates network error)
      return status === undefined || status >= 500;
    }

    return false; // Don't retry by default
  }

  /**
   * Calculate retry delay with exponential backoff
   */
  private getRetryDelay(attempt: number): number {
    return Math.min(1000 * Math.pow(2, attempt), 10000); // Max 10 seconds
  }

  /**
   * Handle request errors and return standardized error response
   */
  private handleRequestError(error: unknown): APIResponse {
    if (error instanceof Error && error.cause) {
      const cause = error.cause as { status?: number; statusText?: string; data?: unknown };
      const status = cause?.status;

      switch (status || 500) {
        case 400:
          return {
            success: false,
            error: {
              code: 'BAD_REQUEST',
              message: 'Invalid request parameters',
              details: (cause.data ?? {}) as Record<string, unknown>
            }
          };
        case 401:
          return {
            success: false,
            error: {
              code: 'UNAUTHORIZED',
              message: 'Authentication required'
            }
          };
        case 403:
          return {
            success: false,
            error: {
              code: 'FORBIDDEN',
              message: 'Access denied'
            }
          };
        case 404:
          return {
            success: false,
            error: {
              code: 'NOT_FOUND',
              message: 'Resource not found'
            }
          };
        case 422:
          return {
            success: false,
            error: {
              code: 'VALIDATION_ERROR',
              message: 'Validation failed',
              details: (cause.data ?? {}) as Record<string, unknown>
            }
          };
        case 500:
          return {
            success: false,
            error: {
              code: 'INTERNAL_SERVER_ERROR',
              message: 'Server error occurred'
            }
          };
        default:
          return {
            success: false,
            error: {
              code: 'HTTP_ERROR',
              message: error.message,
              details: { status: status ?? 0, statusText: cause.statusText }
            }
          };
      }
    }

    // Handle network errors or other issues
    if (error instanceof Error && error.name === 'AbortError') {
      return {
        success: false,
        error: {
          code: 'TIMEOUT',
          message: 'Request timed out'
        }
      };
    }

    return {
      success: false,
      error: {
        code: 'NETWORK_ERROR',
        message: 'Network request failed',
        details: { originalError: error instanceof Error ? error.message : String(error) }
      }
    };
  }

  /**
   * Delay utility for retries
   */
  private delay(ms: number): Promise<void> {
    return new Promise(resolve => setTimeout(resolve, ms));
  }

  /**
   * Set authentication token
   */
  setAuthToken(token: string): void {
    if (typeof window !== 'undefined' && window.localStorage) {
      localStorage.setItem('knirv_auth_token', token);
    }
  }

  /**
   * Clear authentication token
   */
  clearAuthToken(): void {
    if (typeof window !== 'undefined' && window.localStorage) {
      localStorage.removeItem('knirv_auth_token');
    }
  }

  /**
   * Update base URL (useful for environment switching)
   */
  setBaseURL(url: string): void {
    this.baseURL = url;
  }

  /**
   * Get current base URL
   */
  getBaseURL(): string {
    return this.baseURL;
  }
}

// Export singleton instance
export const apiService = new APIService();

// Export error handling utilities
export const isAPIError = (response: APIResponse): response is APIResponse & { error: APIError } => {
  return !response.success && response.error !== undefined;
};

export const getErrorMessage = (response: APIResponse): string => {
  if (response.error) {
    return response.error.message;
  }
  return 'An unknown error occurred';
};
