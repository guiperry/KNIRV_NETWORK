/**
 * Testnet Faucet Service for KNIRV Gateway SDK
 * 
 * Provides comprehensive integration with the KNIRVTESTNET NRV Faucet,
 * including token requests, status monitoring, history tracking, and health checks.
 */

import {
  RequestConfig,
  FaucetRequest,
  FaucetResponse,
  FaucetStatus,
  FaucetHistory,
  FaucetHealthResponse,
  EconomicsServiceError,
} from './types';

/**
 * Configuration options for the Faucet Service
 */
export interface FaucetServiceConfig {
  /** Base URL for the testnet faucet API */
  faucetURL: string;
  /** Maximum number of retry attempts */
  maxRetries?: number;
  /** Timeout for requests in milliseconds */
  timeout?: number;
  /** Enable debug logging */
  debug?: boolean;
}

/**
 * Options for faucet token requests
 */
export interface FaucetRequestOptions {
  /** Reason for the token request */
  reason?: string;
  /** Maximum retry attempts for this request */
  maxRetries?: number;
  /** Enable exponential backoff for retries */
  useExponentialBackoff?: boolean;
}

/**
 * Testnet Faucet Service
 * 
 * Handles all interactions with the KNIRVTESTNET NRV Faucet including:
 * - Token requests with automatic retry logic
 * - Faucet status monitoring
 * - Request history tracking
 * - Health checks and monitoring
 */
export class FaucetService {
  private config: FaucetServiceConfig;
  private defaultTimeout: number = 30000;
  private defaultMaxRetries: number = 3;

  constructor(config: FaucetServiceConfig) {
    this.config = {
      maxRetries: this.defaultMaxRetries,
      timeout: this.defaultTimeout,
      debug: false,
      ...config,
    };

    // Validate faucet URL
    if (!this.config.faucetURL) {
      throw new Error('Faucet URL is required');
    }

    // Ensure URL ends with proper path
    if (!this.config.faucetURL.endsWith('/api/faucet')) {
      this.config.faucetURL = this.config.faucetURL.replace(/\/$/, '') + '/api/faucet';
    }
  }

  /**
   * Update the faucet service configuration
   */
  updateConfig(config: Partial<FaucetServiceConfig>): void {
    this.config = { ...this.config, ...config };
  }

  /**
   * Request NRV tokens from the testnet faucet
   * 
   * @param address - Target address for token distribution
   * @param amount - Amount of NRV tokens to request
   * @param options - Additional request options
   * @returns Promise resolving to faucet response
   */
  async requestTokens(
    address: string,
    amount: number,
    options: FaucetRequestOptions = {}
  ): Promise<FaucetResponse> {
    this.validateAddress(address);
    this.validateAmount(amount);

    const request: FaucetRequest = {
      address,
      amount,
      reason: options.reason || 'SDK request',
    };

    const maxRetries = options.maxRetries ?? this.config.maxRetries!;
    const useBackoff = options.useExponentialBackoff ?? true;

    if (this.config.debug) {
      console.log(`[FaucetService] Requesting ${amount} NRV for ${address}`);
    }

    return this.requestWithRetry('/request', 'POST', request, maxRetries, useBackoff);
  }

  /**
   * Get the current status of the testnet faucet
   * 
   * @returns Promise resolving to faucet status
   */
  async getStatus(): Promise<FaucetStatus> {
    if (this.config.debug) {
      console.log('[FaucetService] Getting faucet status');
    }

    return this.request<FaucetStatus>('/status', 'GET');
  }

  /**
   * Get faucet request history for an address
   * 
   * @param address - Address to get history for
   * @param limit - Maximum number of entries to return (default: 10)
   * @returns Promise resolving to faucet history
   */
  async getHistory(address: string, limit: number = 10): Promise<FaucetHistory> {
    this.validateAddress(address);

    if (limit <= 0 || limit > 100) {
      throw new Error('Limit must be between 1 and 100');
    }

    if (this.config.debug) {
      console.log(`[FaucetService] Getting history for ${address} (limit: ${limit})`);
    }

    const queryParams = new URLSearchParams({
      address,
      limit: limit.toString(),
    });

    return this.request<FaucetHistory>(`/history?${queryParams}`, 'GET');
  }

  /**
   * Check the health of the testnet faucet service
   * 
   * @returns Promise resolving to health status
   */
  async checkHealth(): Promise<FaucetHealthResponse> {
    if (this.config.debug) {
      console.log('[FaucetService] Checking faucet health');
    }

    return this.request<FaucetHealthResponse>('/health', 'GET');
  }

  /**
   * Request tokens with automatic retry logic and exponential backoff
   */
  private async requestWithRetry<T>(
    path: string,
    method: string,
    body?: any,
    maxRetries: number = this.defaultMaxRetries,
    useExponentialBackoff: boolean = true
  ): Promise<T> {
    let lastError: Error;

    for (let attempt = 1; attempt <= maxRetries; attempt++) {
      try {
        if (this.config.debug && attempt > 1) {
          console.log(`[FaucetService] Retry attempt ${attempt}/${maxRetries}`);
        }

        const response = await this.request<T>(path, method, body);
        
        // Check if it's a faucet response with success field
        if (typeof response === 'object' && response !== null && 'success' in response) {
          const faucetResponse = response as any;
          if (!faucetResponse.success) {
            throw new EconomicsServiceError(
              faucetResponse.error || 'Faucet request failed',
              faucetResponse.code
            );
          }
        }

        return response;
      } catch (error) {
        lastError = error as Error;

        if (this.config.debug) {
          console.log(`[FaucetService] Attempt ${attempt} failed:`, error);
        }

        // Don't retry on the last attempt
        if (attempt === maxRetries) {
          break;
        }

        // Handle rate limiting with specific retry delay
        if (error instanceof EconomicsServiceError && error.message.includes('RATE_LIMITED')) {
          const retryAfter = this.extractRetryAfter(error.message);
          if (retryAfter > 0) {
            if (this.config.debug) {
              console.log(`[FaucetService] Rate limited, waiting ${retryAfter}s`);
            }
            await this.sleep(retryAfter * 1000);
            continue;
          }
        }

        // Exponential backoff for other errors
        if (useExponentialBackoff) {
          const delay = Math.min(1000 * Math.pow(2, attempt - 1), 10000);
          if (this.config.debug) {
            console.log(`[FaucetService] Waiting ${delay}ms before retry`);
          }
          await this.sleep(delay);
        } else {
          await this.sleep(1000);
        }
      }
    }

    throw new EconomicsServiceError(
      `Faucet request failed after ${maxRetries} attempts: ${lastError.message}`
    );
  }

  /**
   * Make HTTP request to faucet API
   */
  private async request<T>(
    path: string,
    method: string,
    body?: any
  ): Promise<T> {
    const url = `${this.config.faucetURL}${path}`;
    
    const requestOptions: RequestInit = {
      method: method.toUpperCase(),
      headers: {
        'Content-Type': 'application/json',
        'User-Agent': 'KNIRV-Gateway-SDK-TS/1.0.0',
      },
      signal: AbortSignal.timeout(this.config.timeout!),
    };

    if (body) {
      requestOptions.body = JSON.stringify(body);
    }

    try {
      const response = await fetch(url, requestOptions);
      
      if (!response.ok) {
        throw new EconomicsServiceError(
          `HTTP ${response.status}: ${response.statusText}`,
          response.status.toString()
        );
      }

      const data = await response.json();
      return data;
    } catch (error) {
      if (error instanceof EconomicsServiceError) {
        throw error;
      }
      throw new EconomicsServiceError(`Request failed: ${error.message}`);
    }
  }

  /**
   * Validate KNIRV address format
   */
  private validateAddress(address: string): void {
    if (!address) {
      throw new Error('Address is required');
    }

    if (!address.startsWith('knirv1')) {
      throw new Error('Address must start with "knirv1"');
    }

    if (address.length < 20) {
      throw new Error('Address too short (minimum 20 characters)');
    }
  }

  /**
   * Validate token amount
   */
  private validateAmount(amount: number): void {
    if (!Number.isInteger(amount) || amount <= 0) {
      throw new Error('Amount must be a positive integer');
    }

    if (amount > 10000) {
      throw new Error('Amount too large (maximum 10000 NRV per request)');
    }
  }

  /**
   * Extract retry delay from rate limit error message
   */
  private extractRetryAfter(message: string): number {
    const match = message.match(/retry_after[":]\s*(\d+)/i);
    return match ? parseInt(match[1], 10) : 0;
  }

  /**
   * Sleep for specified milliseconds
   */
  private sleep(ms: number): Promise<void> {
    return new Promise(resolve => setTimeout(resolve, ms));
  }
}

/**
 * Create a new FaucetService instance with default configuration
 * 
 * @param faucetURL - Base URL for the testnet faucet
 * @param options - Additional configuration options
 * @returns Configured FaucetService instance
 */
export function createFaucetService(
  faucetURL: string,
  options: Partial<FaucetServiceConfig> = {}
): FaucetService {
  return new FaucetService({
    faucetURL,
    ...options,
  });
}
