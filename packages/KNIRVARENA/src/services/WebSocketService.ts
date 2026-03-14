/**
 * WebSocket Service
 * Handles real-time data synchronization and event streaming
 */

export interface WebSocketEvent {
  type: string;
  data: unknown;
  timestamp: number;
}

export interface WebSocketConnectionStatus {
  connected: boolean;
  url?: string;
  lastPing?: number;
  reconnectAttempts?: number;
}

export class WebSocketService {
  private ws: WebSocket | null = null;
  private reconnectAttempts = 0;
  private maxReconnectAttempts = 5;
  private reconnectDelay = 1000;
  private url: string;
  private isConnecting = false;
  private connectionStatus: WebSocketConnectionStatus = { connected: false };
  private eventListeners: Map<string, ((event: WebSocketEvent) => void)[]> = new Map();
  private heartbeatInterval: NodeJS.Timeout | number | null = null;

  constructor(url: string = this.detectWebSocketURL()) {
    this.url = url;
  }

  /**
   * Auto-detect the appropriate WebSocket URL based on environment
   */
  private detectWebSocketURL(): string {
    if (typeof window !== 'undefined' && window.location) {
      if (window.location.hostname === 'localhost') {
        return 'ws://localhost:3001/ws'; // Local development
      }
      if (window.location.hostname.includes('testnet')) {
        return 'wss://api-testnet.knirv.com/ws'; // Testnet
      }
    }

    // Production fallback
    return 'wss://api.knirv.com/ws';
  }

  /**
   * Connect to WebSocket server
   */
  async connect(): Promise<boolean> {
    if (this.isConnecting || this.connectionStatus.connected) {
      return this.connectionStatus.connected;
    }

    this.isConnecting = true;

    return new Promise((resolve) => {
      try {
        this.ws = new WebSocket(this.url);
        this.ws.onopen = this.handleOpen.bind(this);
        this.ws.onmessage = this.handleMessage.bind(this);
        this.ws.onclose = this.handleClose.bind(this);
        this.ws.onerror = this.handleError.bind(this);

        // Timeout for connection attempt
        setTimeout(() => {
          if (!this.connectionStatus.connected) {
            this.isConnecting = false;
            resolve(false);
          }
        }, 10000);

        // Override the callback-based resolve with proper handling
        this.ws.onopen = () => {
          this.isConnecting = false;
          resolve(true);
        };

      } catch (error) {
        console.error('WebSocket connection error:', error);
        this.isConnecting = false;
        resolve(false);
      }
    });
  }

  /**
   * Disconnect from WebSocket server
   */
  disconnect(): void {
    this.isConnecting = false;

    if (this.heartbeatInterval) {
      clearInterval(this.heartbeatInterval);
      this.heartbeatInterval = null;
    }

    if (this.ws) {
      this.ws.close(1000, 'Client disconnect');
      this.ws = null;
    }

    this.connectionStatus = { connected: false };
  }

  /**
   * Send a message to the WebSocket server
   */
  send(type: string, data: unknown = {}): boolean {
    if (!this.ws || this.ws.readyState !== WebSocket.OPEN) {
      return false;
    }

    try {
      const message = JSON.stringify({
        type,
        data,
        timestamp: Date.now()
      });
      this.ws.send(message);
      return true;
    } catch (error) {
      console.error('Error sending WebSocket message:', error);
      return false;
    }
  }

  /**
   * Subscribe to a specific event type
   */
  on(eventType: string, callback: (event: WebSocketEvent) => void): void {
    if (!this.eventListeners.has(eventType)) {
      this.eventListeners.set(eventType, []);
    }
    this.eventListeners.get(eventType)!.push(callback);
  }

  /**
   * Unsubscribe from a specific event type
   */
  off(eventType: string, callback?: (event: WebSocketEvent) => void): void {
    if (!this.eventListeners.has(eventType)) {
      return;
    }

    const listeners = this.eventListeners.get(eventType)!;
    if (callback) {
      const index = listeners.indexOf(callback);
      if (index > -1) {
        listeners.splice(index, 1);
      }
    } else {
      listeners.length = 0;
    }
  }

  /**
   * Get current connection status
   */
  getConnectionStatus(): WebSocketConnectionStatus {
    return { ...this.connectionStatus };
  }

  /**
   * Update WebSocket URL (useful for environment switching)
   */
  setURL(url: string): void {
    this.url = url;
    if (this.connectionStatus.connected) {
      this.disconnect();
      this.connect();
    }
  }

  /**
   * Handle WebSocket open event
   */
  private handleOpen(): void {
    console.log('WebSocket connected to:', this.url);
    this.connectionStatus = {
      connected: true,
      url: this.url,
      lastPing: Date.now(),
      reconnectAttempts: 0
    };
    this.reconnectAttempts = 0;

    // Start heartbeat
    this.startHeartbeat();

    // Emit connection event
    this.emitEvent('connection:established', { connected: true });
  }

  /**
   * Handle WebSocket message event
   */
  private handleMessage(event: MessageEvent): void {
    try {
      const message: WebSocketEvent = JSON.parse(event.data);
      this.emitEvent(message.type, message.data);

      // Update ping time for heartbeat
      if (message.type === 'heartbeat:pong') {
        this.connectionStatus.lastPing = Date.now();
      }
    } catch (error) {
      console.error('Error parsing WebSocket message:', error);
    }
  }

  /**
   * Handle WebSocket close event
   */
  private handleClose(event: CloseEvent): void {
    console.log('WebSocket disconnected:', event.code, event.reason);
    this.connectionStatus = { connected: false };
    this.ws = null;

    if (this.heartbeatInterval) {
      clearInterval(this.heartbeatInterval);
      this.heartbeatInterval = null;
    }

    // Emit disconnection event
    this.emitEvent('connection:lost', { code: event.code, reason: event.reason });

    // Attempt to reconnect if not a clean close
    if (event.code !== 1000 && this.reconnectAttempts < this.maxReconnectAttempts) {
      this.attemptReconnect();
    }
  }

  /**
   * Handle WebSocket error event
   */
  private handleError(event: Event): void {
    console.error('WebSocket error:', event);
    this.emitEvent('connection:error', { error: event });
  }

  /**
   * Start heartbeat mechanism
   */
  private startHeartbeat(): void {
    this.heartbeatInterval = setInterval(() => {
      if (this.connectionStatus.connected) {
        this.send('heartbeat:ping', { timestamp: Date.now() });
      }
    }, 30000); // 30 seconds
  }

  /**
   * Attempt to reconnect to WebSocket server
   */
  private attemptReconnect(): void {
    this.reconnectAttempts++;
    const delay = this.reconnectDelay * Math.pow(2, this.reconnectAttempts - 1); // Exponential backoff

    console.log(`Attempting WebSocket reconnect ${this.reconnectAttempts}/${this.maxReconnectAttempts} in ${delay}ms`);

    setTimeout(() => {
      if (!this.connectionStatus.connected && this.reconnectAttempts < this.maxReconnectAttempts) {
        this.connect();
      }
    }, delay);
  }

  /**
   * Emit event to all registered listeners
   */
  private emitEvent(type: string, data: unknown): void {
    const listeners = this.eventListeners.get(type);
    if (listeners) {
      const event: WebSocketEvent = {
        type,
        data,
        timestamp: Date.now()
      };

      listeners.forEach(callback => {
        try {
          callback(event);
        } catch (error) {
          console.error('Error in WebSocket event listener:', error);
        }
      });
    }
  }

  /**
   * Subscribe to agent-related events
   */
  subscribeToAgent(agentId: string): void {
    this.on(`agent:${agentId}:status`, (event) => {
      console.log('Agent status update:', event.data);
    });

    this.on(`agent:${agentId}:performance`, (event) => {
      console.log('Agent performance update:', event.data);
    });
  }

  /**
   * Subscribe to wallet-related events
   */
  subscribeToWallet(accountId: string): void {
    this.on(`wallet:${accountId}:transaction`, (event) => {
      console.log('Wallet transaction update:', event.data);
    });

    this.on(`wallet:${accountId}:balance`, (event) => {
      console.log('Wallet balance update:', event.data);
    });
  }

  /**
   * Subscribe to analytics-related events
   */
  subscribeToAnalytics(dashboardId: string): void {
    this.on(`analytics:${dashboardId}:metrics`, (event) => {
      console.log('Analytics metrics update:', event.data);
    });
  }

  /**
   * Subscribe to task-related events
   */
  subscribeToTask(taskId: string): void {
    this.on(`task:${taskId}:status`, (event) => {
      console.log('Task status update:', event.data);
    });

    this.on(`task:${taskId}:progress`, (event) => {
      console.log('Task progress update:', event.data);
    });
  }

  /**
   * Unsubscribe from all events for a specific entity
   */
  unsubscribeFromEntity(entityType: string, entityId: string): void {
    const prefix = `${entityType}:${entityId}:`;
    const eventsToRemove: string[] = [];

    this.eventListeners.forEach((listeners, eventType) => {
      if (eventType.startsWith(prefix)) {
        eventsToRemove.push(eventType);
      }
    });

    eventsToRemove.forEach(eventType => {
      this.eventListeners.delete(eventType);
    });
  }
}

// Export singleton instance
export const webSocketService = new WebSocketService();

// Export convenience functions for common subscriptions
export const subscribeToWallet = (accountId: string) => webSocketService.subscribeToWallet(accountId);
export const subscribeToAgent = (agentId: string) => webSocketService.subscribeToAgent(agentId);
export const subscribeToAnalytics = (dashboardId?: string) => webSocketService.subscribeToAnalytics(dashboardId || 'default');
export const subscribeToTask = (taskId: string) => webSocketService.subscribeToTask(taskId);
