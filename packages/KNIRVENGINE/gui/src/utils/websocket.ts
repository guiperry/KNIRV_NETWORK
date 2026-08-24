// WebSocket utility for real-time updates in the KNIRVENGINE
// Provides connection management, reconnection logic, and event handling

export interface WebSocketMessage<T> {
  type: string;
  data: T;
  timestamp: string;
}

export interface AgentStatusUpdate {
  agentId: string;
  status: string;
  currentTarget?: string;
  capability?: string;
  lastActivity: string;
}

export interface BuildProgressUpdate {
  buildId: string;
  agentId: string;
  status: 'started' | 'in_progress' | 'completed' | 'failed';
  progress: number;
  message: string;
  timestamp: string;
}

export interface SubAgentUpdate {
  parentAgentId: string;
  subAgentId: string;
  status: string;
  output?: string;
  timestamp: string;
}

export interface SystemMetricsUpdate {
  cpu: number;
  memory: number;
  activeAgents: number;
  totalInferences: number;
  timestamp: string;
}

export type WebSocketMessageTypes = {
  agent_status: AgentStatusUpdate;
  build_progress: BuildProgressUpdate;
  sub_agent_update: SubAgentUpdate;
  system_metrics: SystemMetricsUpdate;
  [key: string]: unknown;
};

// Union type for all possible message data types
export type WebSocketMessageData =
  | AgentStatusUpdate
  | BuildProgressUpdate
  | SubAgentUpdate
  | SystemMetricsUpdate
  | unknown;

export type WebSocketEventHandler<T = unknown> = (
  data: WebSocketMessage<T>
) => void;

// Generic event handler that can handle any message type
export type GenericWebSocketEventHandler = (
  data: WebSocketMessage<WebSocketMessageData>
) => void;

export class WebSocketManager {
  private ws: WebSocket | null = null;
  private reconnectAttempts = 0;
  private maxReconnectAttempts = 5;
  private reconnectDelay = 1000; // Start with 1 second
  private maxReconnectDelay = 30000; // Max 30 seconds
  private eventHandlers: Map<string, GenericWebSocketEventHandler[]> = new Map();
  private isConnecting = false;
  private shouldReconnect = true;
  private baseUrl: string;
  private isElectron: boolean;

  constructor(baseUrl?: string) {
    // Detect if running in Electron
    this.isElectron = window.location.protocol === 'file:';

    // Set appropriate base URL based on environment
    if (baseUrl) {
      this.baseUrl = baseUrl;
    } else {
      // Browser builds must use the GUI's origin. Vite proxies this route to
      // the launcher-selected API port, and production serves it through the
      // engine reverse proxy. This deliberately ignores VITE_API_BASE_URL in
      // browsers so an old/stale 8081 setting cannot override that port.
      const apiUrl = (this.isElectron
        ? (import.meta.env.VITE_API_BASE_URL || 'http://localhost:8081')
        : window.location.origin).replace(/\/$/, '');
      this.baseUrl = apiUrl.replace(/^http:/, 'ws:').replace(/^https:/, 'wss:');
    }

    console.log(`WebSocket Manager initialized - Electron: ${this.isElectron}, Base URL: ${this.baseUrl}`);
  }

  // Connect to WebSocket server
  connect(): Promise<void> {
    return new Promise((resolve, reject) => {
      // If already connected, resolve immediately
      if (this.ws && this.ws.readyState === WebSocket.OPEN) {
        resolve();
        return;
      }

      // If connection is in progress but not yet established, wait for it
      if (this.isConnecting) {
        console.warn('WebSocket connection already in progress, waiting for completion');
        
        // Create a timeout to prevent indefinite waiting
        const timeout = setTimeout(() => {
          reject(new Error('Connection attempt timed out'));
        }, 10000); // 10 second timeout
        
        // Check connection status periodically
        const checkInterval = setInterval(() => {
          if (this.ws && this.ws.readyState === WebSocket.OPEN) {
            clearTimeout(timeout);
            clearInterval(checkInterval);
            resolve();
          } else if (!this.isConnecting) {
            clearTimeout(timeout);
            clearInterval(checkInterval);
            reject(new Error('Connection attempt failed'));
          }
        }, 100);
        
        return;
      }

      this.isConnecting = true;

      try {
        // The standard API endpoint supports both browser and desktop GUI
        // connections. The secure desktop endpoint additionally requires an
        // ELECTRON_MODE server flag and otherwise rejects valid local clients.
        const wsEndpoint = `${this.baseUrl}/api/v1/ws`;

        console.log(`Connecting to WebSocket: ${wsEndpoint}`);
        
        // Close any existing connection before creating a new one
        if (this.ws) {
          this.ws.close(1000, 'Creating new connection');
          this.ws = null;
        }
        
        this.ws = new WebSocket(wsEndpoint);

        this.ws.onopen = () => {
          console.log(`WebSocket connected (${this.isElectron ? 'Desktop' : 'Cloud'} mode)`);
          this.isConnecting = false;
          this.reconnectAttempts = 0;
          this.reconnectDelay = 1000;

          // Send authentication message for desktop mode
          if (this.isElectron) {
            const authMessage = {
              type: 'auth',
              token: localStorage.getItem('token') || 'desktop-token',
              timestamp: new Date().toISOString()
            };
            this.ws?.send(JSON.stringify(authMessage));
          }

          resolve();
        };

        this.ws.onmessage = (event) => {
          try {
            const message: WebSocketMessage<WebSocketMessageData> = JSON.parse(event.data);
            this.handleMessage(message);
          } catch (error) {
            console.error('Failed to parse WebSocket message:', error);
          }
        };

        this.ws.onclose = (event) => {
          console.log('WebSocket disconnected:', event.code, event.reason);
          this.isConnecting = false;
          this.ws = null;

          // Reconnect for any disconnection if we should reconnect, not just unclean closes
          // This helps with intermittent connection issues
          if (this.shouldReconnect) {
            // Add a small delay even for clean closes to prevent rapid reconnection loops
            setTimeout(() => {
              if (this.shouldReconnect) {
                this.scheduleReconnect();
              }
            }, event.wasClean ? 1000 : 0);
          }
        };

        this.ws.onerror = (error) => {
          console.error('WebSocket error:', error);
          this.isConnecting = false;
          reject(error);
        };

      } catch (error) {
        this.isConnecting = false;
        reject(error);
      }
    });
  }

  // Disconnect from WebSocket server
  disconnect(): void {
    this.shouldReconnect = false;
    if (this.ws) {
      this.ws.close(1000, 'Client disconnect');
      this.ws = null;
    }
  }

  // Subscribe to specific event types
  subscribe(eventType: string, handler: GenericWebSocketEventHandler): void {
    if (!this.eventHandlers.has(eventType)) {
      this.eventHandlers.set(eventType, []);
    }
    this.eventHandlers.get(eventType)!.push(handler);
  }

  // Unsubscribe from event types
  unsubscribe(eventType: string, handler: GenericWebSocketEventHandler): void {
    const handlers = this.eventHandlers.get(eventType);
    if (handlers) {
      const index = handlers.indexOf(handler);
      if (index > -1) {
        handlers.splice(index, 1);
      }
    }
  }

  // Send message to server
  send<T extends keyof WebSocketMessageTypes>(
    message: WebSocketMessage<WebSocketMessageTypes[T]>
  ): void {
    if (this.ws && this.ws.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify(message));
    } else {
      console.warn('WebSocket not connected, message not sent:', message);
    }
  }

  // Get connection status
  isConnected(): boolean {
    return this.ws !== null && this.ws.readyState === WebSocket.OPEN;
  }

  // Handle incoming messages
  private handleMessage(message: WebSocketMessage<WebSocketMessageData>): void {
    const handlers = this.eventHandlers.get(message.type);
    if (handlers) {
      handlers.forEach(handler => {
        try {
          handler(message);
        } catch (error) {
          console.error(`Error in WebSocket handler for ${message.type}:`, error);
        }
      });
    }
  }

  // Schedule reconnection with exponential backoff
  private scheduleReconnect(): void {
    if (this.reconnectAttempts >= this.maxReconnectAttempts) {
      console.error('Max reconnection attempts reached');
      return;
    }

    this.reconnectAttempts++;
    const delay = Math.min(this.reconnectDelay * Math.pow(2, this.reconnectAttempts - 1), this.maxReconnectDelay);
    
    console.log(`Scheduling reconnection attempt ${this.reconnectAttempts} in ${delay}ms`);
    
    setTimeout(() => {
      if (this.shouldReconnect) {
        this.connect().catch(error => {
          console.error('Reconnection failed:', error);
        });
      }
    }, delay);
  }
}

// Global WebSocket manager instance - auto-detects environment
export const wsManager = new WebSocketManager();

// Convenience functions for specific event types
export const subscribeToAgentStatus = (handler: (update: AgentStatusUpdate) => void) => {
  const wrappedHandler: GenericWebSocketEventHandler = (message) => {
    handler(message.data as AgentStatusUpdate);
  };
  wsManager.subscribe('agent_status', wrappedHandler);
  return () => wsManager.unsubscribe('agent_status', wrappedHandler);
};

export const subscribeToBuildProgress = (handler: (update: BuildProgressUpdate) => void) => {
  wsManager.subscribe('build_progress', (message) => {
    handler(message.data as BuildProgressUpdate);
  });
};

export const subscribeToSubAgentUpdates = (handler: (update: SubAgentUpdate) => void) => {
  wsManager.subscribe('sub_agent_update', (message) => {
    handler(message.data as SubAgentUpdate);
  });
};

export const subscribeToSystemMetrics = (handler: (update: SystemMetricsUpdate) => void) => {
  wsManager.subscribe('system_metrics', (message) => {
    handler(message.data as SystemMetricsUpdate);
  });
};

// Initialize WebSocket connection with retry logic
export const initializeWebSocket = async (): Promise<void> => {
  const maxRetries = 5;
  const baseRetryDelay = 2000; // 2 seconds base delay

  for (let attempt = 1; attempt <= maxRetries; attempt++) {
    try {
      // Clear any existing connection state before attempting to connect
      if (wsManager.isConnected()) {
        wsManager.disconnect();
        // Small delay to ensure disconnect completes
        await new Promise(resolve => setTimeout(resolve, 500));
      }
      
      await wsManager.connect();
      console.log('WebSocket initialized successfully');
      return;
    } catch (error) {
      console.warn(`WebSocket connection attempt ${attempt} failed:`, error);

      if (attempt < maxRetries) {
        // Use exponential backoff with jitter for retries
        const jitter = Math.random() * 0.3 + 0.85; // Random factor between 0.85 and 1.15
        const delay = Math.floor(baseRetryDelay * Math.pow(1.5, attempt - 1) * jitter);
        
        console.log(`Retrying WebSocket connection in ${delay}ms...`);
        await new Promise(resolve => setTimeout(resolve, delay));
      } else {
        console.error('Failed to initialize WebSocket after all retries');
        // Don't throw the error, just log it - this allows the application to continue
        // even if WebSocket connection fails
      }
    }
  }
};

// Cleanup function
export const cleanupWebSocket = (): void => {
  wsManager.disconnect();
};
