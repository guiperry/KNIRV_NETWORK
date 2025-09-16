/**
 * Standardized API utilities for consistent communication with the backend
 */

import type { APIResponse, APIError } from '@/types/api';

// API configuration - detect if running in production or development
export const getApiBaseUrl = (): string => {
  // In production (static export), use relative URLs that will be proxied
  if (typeof window !== 'undefined' && window.location.hostname !== 'localhost') {
    return '';
  }
  // In development, use the backend API port (8082 as configured)
  return 'http://localhost:8082';
};

export const API_BASE_URL = getApiBaseUrl();

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
    
    const data: APIResponse<T> = await response.json();
    
    // Validate response structure
    if (typeof data !== 'object' || data === null) {
      throw new APIRequestError('Invalid response format', 500);
    }
    
    return data;
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
  return `${API_BASE_URL.replace('http', 'ws')}/ws`;
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
