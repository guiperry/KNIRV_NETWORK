/**
 * Standardized API utilities for consistent communication with the backend
 */

import type { APIResponse, APIError } from '@/types/api';

// API configuration - detect if running in production or development
export const getApiBaseUrl = (): string => {
  // Use relative URLs to leverage the built-in proxy in the wrapper (port 8090)
  // or the relative path in production. This avoids CORS issues.
  return '';
};

export const API_BASE_URL = getApiBaseUrl();

const isAPIResponse = <T = unknown>(value: unknown): value is APIResponse<T> => {
  return typeof value === 'object' && value !== null && 'success' in value;
};

// Helper function to get auth headers
export const getAuthHeaders = (): HeadersInit => {
  if (typeof window === 'undefined') return {};
  
  const token = localStorage.getItem('knirv_nexus_token');
  return {
    'Content-Type': 'application/json',
    ...(token && { 'Authorization': `Bearer ${token}` })
  };
};

// Enhanced fetch function with error handling and type safety
export const apiRequest = async <T = any>(
  url: string, 
  options: RequestInit = {}
): Promise<APIResponse<T>> => {
  const requestOptions: RequestInit = {
    ...options,
    headers: {
      ...getAuthHeaders(),
      ...options.headers,
    },
  };

  try {
    const response = await fetch(url, requestOptions);
    
    if (!response.ok) {
      // Try to parse error response
      let errorData: APIError;
      try {
        errorData = await response.json();
      } catch {
        errorData = {
          code: `HTTP_${response.status}`,
          message: response.statusText || 'Unknown error occurred',
          timestamp: new Date().toISOString()
        };
      }
      
      throw new APIRequestError(
        errorData.message || `HTTP ${response.status}: ${response.statusText}`,
        response.status,
        errorData
      );
    }
    
    const data = await response.json();

    if (isAPIResponse<T>(data)) {
      return data;
    }

    return {
      success: true,
      data: data as T,
      timestamp: new Date().toISOString()
    };
  } catch (error) {
    if (error instanceof APIRequestError) {
      throw error;
    }
    
    // Network or other errors
    console.error('API request failed:', error);
    throw new APIRequestError(
      error instanceof Error ? error.message : 'Network error occurred',
      0
    );
  }
};

// Custom error class for API requests
export class APIRequestError extends Error {
  public readonly status: number;
  public readonly data?: APIError;

  constructor(message: string, status: number, data?: APIError) {
    super(message);
    this.name = 'APIRequestError';
    this.status = status;
    this.data = data;
  }

  public isNetworkError(): boolean {
    return this.status === 0;
  }

  public isClientError(): boolean {
    return this.status >= 400 && this.status < 500;
  }

  public isServerError(): boolean {
    return this.status >= 500;
  }

  public isAuthError(): boolean {
    return this.status === 401 || this.status === 403;
  }
}

// WebSocket utilities
export const getWebSocketUrl = (): string => {
  if (typeof window === 'undefined') {
    return 'ws://localhost:8090/ws';
  }

  const base = API_BASE_URL
    ? new URL(API_BASE_URL, window.location.origin)
    : new URL(window.location.origin);
  const protocol = base.protocol === 'https:' ? 'wss:' : 'ws:';

  return `${protocol}//${base.host}/ws`;
};

export interface WebSocketMessage {
  type: string;
  event?: string;
  payload?: any;
  timestamp?: string;
}

export class StandardWebSocket {
  private ws: WebSocket | null = null;
  private reconnectAttempts = 0;
  private maxReconnectAttempts = 5;
  private reconnectDelay = 1000; // Start with 1 second
  private maxReconnectDelay = 30000; // Max 30 seconds
  private isConnecting = false;
  private subscriptions = new Set<string>();
  
  public onOpen?: () => void;
  public onClose?: () => void;
  public onError?: (error: Event) => void;
  public onMessage?: (message: WebSocketMessage) => void;

  constructor() {
    if (typeof window === 'undefined') {
      return;
    }
    this.connect();
  }

  private connect(): void {
    if (this.isConnecting || (this.ws && this.ws.readyState === WebSocket.OPEN)) {
      return;
    }

    this.isConnecting = true;
    const wsUrl = getWebSocketUrl();
    
    try {
      this.ws = new WebSocket(wsUrl);
      
      this.ws.onopen = () => {
        console.log('WebSocket connected');
        this.isConnecting = false;
        this.reconnectAttempts = 0;
        this.reconnectDelay = 1000;
        
        // Resubscribe to topics
        if (this.subscriptions.size > 0) {
          Array.from(this.subscriptions).forEach(topic => {
            this.ws!.send(JSON.stringify({
              type: 'subscribe',
              topic: topic,
              timestamp: new Date().toISOString()
            }));
          });
        }
        
        this.onOpen?.();
      };

      this.ws.onclose = () => {
        console.log('WebSocket disconnected');
        this.isConnecting = false;
        this.ws = null;
        this.onClose?.();
        
        // Attempt to reconnect
        this.scheduleReconnect();
      };

      this.ws.onerror = (error) => {
        console.error('WebSocket error:', error);
        this.isConnecting = false;
        this.onError?.(error);
      };

      this.ws.onmessage = (event) => {
        try {
          const message: WebSocketMessage = JSON.parse(event.data);
          this.onMessage?.(message);
        } catch (error) {
          console.error('Failed to parse WebSocket message:', error);
        }
      };
    } catch (error) {
      console.error('Failed to create WebSocket connection:', error);
      this.isConnecting = false;
      this.scheduleReconnect();
    }
  }

  private scheduleReconnect(): void {
    if (this.reconnectAttempts >= this.maxReconnectAttempts) {
      console.error('Max reconnection attempts reached');
      return;
    }

    this.reconnectAttempts++;
    const delay = Math.min(this.reconnectDelay * Math.pow(2, this.reconnectAttempts - 1), this.maxReconnectDelay);
    
    console.log(`Attempting to reconnect in ${delay}ms (attempt ${this.reconnectAttempts})`);
    
    setTimeout(() => {
      this.connect();
    }, delay);
  }

  public send(message: WebSocketMessage): void {
    if (this.ws && this.ws.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify({
        ...message,
        timestamp: new Date().toISOString()
      }));
    } else {
      console.warn('WebSocket not connected, message not sent:', message);
    }
  }

  public subscribe(topics: string[]): void {
    topics.forEach(topic => this.subscriptions.add(topic));

    if (this.ws && this.ws.readyState === WebSocket.OPEN) {
      // Send individual subscription messages for each topic (backend expects this format)
      topics.forEach(topic => {
        this.ws!.send(JSON.stringify({
          type: 'subscribe',
          topic: topic,
          timestamp: new Date().toISOString()
        }));
      });
    }
  }

  public unsubscribe(topics: string[]): void {
    topics.forEach(topic => this.subscriptions.delete(topic));

    if (this.ws && this.ws.readyState === WebSocket.OPEN) {
      // Send individual unsubscription messages for each topic (backend expects this format)
      topics.forEach(topic => {
        this.ws!.send(JSON.stringify({
          type: 'unsubscribe',
          topic: topic,
          timestamp: new Date().toISOString()
        }));
      });
    }
  }

  public close(): void {
    this.reconnectAttempts = this.maxReconnectAttempts; // Prevent reconnection
    if (this.ws) {
      this.ws.close();
      this.ws = null;
    }
  }

  public isConnected(): boolean {
    return this.ws?.readyState === WebSocket.OPEN;
  }
}

// Utility functions for common API patterns
export const buildQueryString = (params: Record<string, any>): string => {
  const searchParams = new URLSearchParams();
  
  Object.entries(params).forEach(([key, value]) => {
    if (value !== undefined && value !== null && value !== '') {
      if (Array.isArray(value)) {
        value.forEach(item => searchParams.append(key, String(item)));
      } else {
        searchParams.append(key, String(value));
      }
    }
  });
  
  const queryString = searchParams.toString();
  return queryString ? `?${queryString}` : '';
};

export const formatTimestamp = (timestamp: string | Date): string => {
  const date = typeof timestamp === 'string' ? new Date(timestamp) : timestamp;
  return date.toISOString();
};

export const parseTimestamp = (timestamp: string): Date => {
  return new Date(timestamp);
};

// Payment API functions
export interface StripeCheckoutRequest {
  amount: number;
  currency: string;
  nrn_amount: number;
  success_url?: string;
  cancel_url?: string;
}

export interface StripeCheckoutResponse {
  session_id: string;
  url: string;
}

export interface PayPalOrderRequest {
  amount: number;
  currency: string;
  nrn_amount: number;
}

export interface PayPalOrderResponse {
  order_id: string;
  status: string;
}

export interface NRNPurchaseRequest {
  amount: number;
  payment_method: 'stripe' | 'paypal' | 'coinbase';
  currency?: string;
}

export interface NRNPurchaseResponse {
  purchase_id: string;
  amount: number;
  status: string;
  payment_url?: string;
}

export const createStripeCheckoutSession = async (request: StripeCheckoutRequest): Promise<StripeCheckoutResponse> => {
  const response = await apiRequest<StripeCheckoutResponse>('/api/v1/payments/stripe/create-session', {
    method: 'POST',
    body: JSON.stringify(request)
  });
  return response.data;
};

export const createPayPalOrder = async (request: PayPalOrderRequest): Promise<PayPalOrderResponse> => {
  const response = await apiRequest<PayPalOrderResponse>('/api/v1/payments/paypal/create-order', {
    method: 'POST',
    body: JSON.stringify(request)
  });
  return response.data;
};

export const capturePayPalOrder = async (orderId: string): Promise<any> => {
  const response = await apiRequest('/api/v1/payments/paypal/capture', {
    method: 'POST',
    body: JSON.stringify({ order_id: orderId })
  });
  return response.data;
};

export const getWalletBalance = async (address?: string): Promise<{ balance: number; address: string }> => {
  const query = address ? `?address=${address}` : '';
  const response = await apiRequest<{ balance: number; address: string }>(`/api/v1/payments/blockchain/balance${query}`);
  return response.data;
};

export const initiateNRNPurchase = async (request: NRNPurchaseRequest): Promise<NRNPurchaseResponse> => {
  const response = await apiRequest<NRNPurchaseResponse>('/api/v1/payments/nrn/purchase', {
    method: 'POST',
    body: JSON.stringify(request)
  });
  return response.data;
};
