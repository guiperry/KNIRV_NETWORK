// Real-time service for NEXUS Portal using SSE with WebSocket fallback

export interface RealtimeMessage {
  type: string;
  channel: string;
  data: any;
  timestamp: number;
}

export interface RealtimeOptions {
  channel: string;
  token?: string;
  reconnectInterval?: number;
  maxReconnectAttempts?: number;
}

export type MessageHandler = (message: RealtimeMessage) => void;
export type ErrorHandler = (error: Error) => void;
export type ConnectionHandler = (connected: boolean) => void;

export class RealtimeService {
  private eventSource: EventSource | null = null;
  private websocket: WebSocket | null = null;
  private messageHandlers: Set<MessageHandler> = new Set();
  private errorHandlers: Set<ErrorHandler> = new Set();
  private connectionHandlers: Set<ConnectionHandler> = new Set();
  private reconnectAttempts = 0;
  private reconnectTimer: NodeJS.Timeout | null = null;
  private isConnected = false;
  private useWebSocket = false;

  constructor(private options: RealtimeOptions) {
    this.options = {
      reconnectInterval: 5000,
      maxReconnectAttempts: 10,
      ...options
    };
  }

  // Connect using SSE (preferred) with WebSocket fallback
  connect(): void {
    this.disconnect(); // Clean up any existing connections
    
    if (this.useWebSocket) {
      this.connectWebSocket();
    } else {
      this.connectSSE();
    }
  }

  // Connect using Server-Sent Events
  private connectSSE(): void {
    try {
      const url = `/gateway/events/nexus-${this.options.channel}`;
      const eventSource = new EventSource(url);

      eventSource.onopen = () => {
        this.isConnected = true;
        this.reconnectAttempts = 0;
        this.notifyConnectionHandlers(true);
        console.log(`SSE connected to channel: ${this.options.channel}`);
      };

      eventSource.onmessage = (event) => {
        try {
          const message: RealtimeMessage = JSON.parse(event.data);
          this.notifyMessageHandlers(message);
        } catch (error) {
          console.error('Failed to parse SSE message:', error);
        }
      };

      eventSource.onerror = (error) => {
        console.error('SSE connection error:', error);
        this.isConnected = false;
        this.notifyConnectionHandlers(false);
        
        // Try WebSocket fallback on SSE failure
        if (!this.useWebSocket) {
          console.log('SSE failed, trying WebSocket fallback...');
          this.useWebSocket = true;
          this.eventSource?.close();
          this.connectWebSocket();
          return;
        }
        
        this.handleReconnect();
      };

      this.eventSource = eventSource;
    } catch (error) {
      console.error('Failed to create SSE connection:', error);
      this.useWebSocket = true;
      this.connectWebSocket();
    }
  }

  // Connect using WebSocket (fallback)
  private connectWebSocket(): void {
    try {
      const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
      const host = window.location.host;
      const url = `${protocol}//${host}/gateway/ws/nexus-${this.options.channel}`;
      
      const websocket = new WebSocket(url);

      websocket.onopen = () => {
        this.isConnected = true;
        this.reconnectAttempts = 0;
        this.notifyConnectionHandlers(true);
        console.log(`WebSocket connected to channel: ${this.options.channel}`);
        
        // Send authentication if token is provided
        if (this.options.token) {
          websocket.send(JSON.stringify({
            type: 'auth',
            token: this.options.token
          }));
        }
      };

      websocket.onmessage = (event) => {
        try {
          const message: RealtimeMessage = JSON.parse(event.data);
          this.notifyMessageHandlers(message);
        } catch (error) {
          console.error('Failed to parse WebSocket message:', error);
        }
      };

      websocket.onclose = () => {
        this.isConnected = false;
        this.notifyConnectionHandlers(false);
        console.log('WebSocket connection closed');
        this.handleReconnect();
      };

      websocket.onerror = (error) => {
        console.error('WebSocket connection error:', error);
        this.notifyErrorHandlers(new Error('WebSocket connection failed'));
      };

      this.websocket = websocket;
    } catch (error) {
      console.error('Failed to create WebSocket connection:', error);
      this.notifyErrorHandlers(error as Error);
      this.handleReconnect();
    }
  }

  // Handle reconnection logic
  private handleReconnect(): void {
    if (this.reconnectAttempts >= (this.options.maxReconnectAttempts || 10)) {
      console.error('Max reconnection attempts reached');
      this.notifyErrorHandlers(new Error('Max reconnection attempts reached'));
      return;
    }

    this.reconnectAttempts++;
    const delay = this.options.reconnectInterval || 5000;
    
    console.log(`Reconnecting in ${delay}ms (attempt ${this.reconnectAttempts})`);
    
    this.reconnectTimer = setTimeout(() => {
      this.connect();
    }, delay);
  }

  // Disconnect from the service
  disconnect(): void {
    if (this.reconnectTimer) {
      clearTimeout(this.reconnectTimer);
      this.reconnectTimer = null;
    }

    if (this.eventSource) {
      this.eventSource.close();
      this.eventSource = null;
    }

    if (this.websocket) {
      this.websocket.close();
      this.websocket = null;
    }

    this.isConnected = false;
    this.notifyConnectionHandlers(false);
  }

  // Send a message (WebSocket only)
  send(message: any): void {
    if (this.websocket && this.websocket.readyState === WebSocket.OPEN) {
      this.websocket.send(JSON.stringify(message));
    } else {
      console.warn('Cannot send message: WebSocket not connected');
    }
  }

  // Event handler management
  onMessage(handler: MessageHandler): () => void {
    this.messageHandlers.add(handler);
    return () => this.messageHandlers.delete(handler);
  }

  onError(handler: ErrorHandler): () => void {
    this.errorHandlers.add(handler);
    return () => this.errorHandlers.delete(handler);
  }

  onConnection(handler: ConnectionHandler): () => void {
    this.connectionHandlers.add(handler);
    return () => this.connectionHandlers.delete(handler);
  }

  // Notify handlers
  private notifyMessageHandlers(message: RealtimeMessage): void {
    this.messageHandlers.forEach(handler => {
      try {
        handler(message);
      } catch (error) {
        console.error('Message handler error:', error);
      }
    });
  }

  private notifyErrorHandlers(error: Error): void {
    this.errorHandlers.forEach(handler => {
      try {
        handler(error);
      } catch (err) {
        console.error('Error handler error:', err);
      }
    });
  }

  private notifyConnectionHandlers(connected: boolean): void {
    this.connectionHandlers.forEach(handler => {
      try {
        handler(connected);
      } catch (error) {
        console.error('Connection handler error:', error);
      }
    });
  }

  // Getters
  get connected(): boolean {
    return this.isConnected;
  }

  get connectionType(): 'sse' | 'websocket' | 'none' {
    if (this.eventSource) return 'sse';
    if (this.websocket) return 'websocket';
    return 'none';
  }
}

// Factory function for creating realtime services
export function createRealtimeService(options: RealtimeOptions): RealtimeService {
  return new RealtimeService(options);
}

// Hook for React components
export function useRealtimeService(options: RealtimeOptions) {
  const [service] = React.useState(() => createRealtimeService(options));
  const [connected, setConnected] = React.useState(false);
  const [messages, setMessages] = React.useState<RealtimeMessage[]>([]);

  React.useEffect(() => {
    const unsubscribeConnection = service.onConnection(setConnected);
    const unsubscribeMessage = service.onMessage((message) => {
      setMessages(prev => [...prev.slice(-99), message]); // Keep last 100 messages
    });

    service.connect();

    return () => {
      unsubscribeConnection();
      unsubscribeMessage();
      service.disconnect();
    };
  }, [service]);

  return {
    service,
    connected,
    messages,
    connectionType: service.connectionType
  };
}
