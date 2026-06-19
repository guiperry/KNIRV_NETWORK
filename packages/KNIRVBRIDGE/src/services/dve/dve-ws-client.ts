import { DVE_CONSTANTS } from '@common/constants/dve.constant';

export type WSMessageHandler = (message: any) => void;

export type WSMessageType =
  | 'task_assigned'
  | 'policy_sync'
  | 'badge_refresh'
  | 'heartbeat_ack';

export interface WSEnvelope {
  type: string;
  timestamp: number;
  payload: any;
}

export interface TaskResultPayload {
  taskID: string;
  status: 'success' | 'failure' | 'error';
  score?: number;
  results?: any;
  errorMessage?: string;
  executionTimeMs?: number;
}

export class DVEWebSocketClient {
  private ws: WebSocket | null = null;
  private serverURL: string;
  private authToken: string;
  private reconnectAttempts: number = 0;
  private maxReconnectAttempts: number = 10;
  private handlers: Map<string, WSMessageHandler[]> = new Map();
  private heartbeatInterval: ReturnType<typeof setInterval> | null = null;
  private shouldReconnect: boolean = false;
  private isConnected: boolean = false;

  constructor(serverURL: string, authToken: string) {
    this.serverURL = serverURL.replace(/\/+$/, '');
    this.authToken = authToken;
  }

  /**
   * Open a WebSocket connection to the DVE server.
   */
  connect(): void {
    if (this.ws && (this.ws.readyState === WebSocket.OPEN || this.ws.readyState === WebSocket.CONNECTING)) {
      console.warn('DVE WebSocket already connected or connecting');
      return;
    }

    this.shouldReconnect = true;
    this.reconnectAttempts = 0;

    const wsURL = this.buildWSURL();
    console.info(`DVE WebSocket connecting to ${wsURL}`);

    try {
      this.ws = new WebSocket(wsURL);

      this.ws.onopen = () => {
        console.info('DVE WebSocket connected');
        this.isConnected = true;
        this.reconnectAttempts = 0;
        this.startHeartbeatInterval();
        this.emit('_connected', {});
      };

      this.ws.onmessage = (event: MessageEvent) => {
        try {
          const envelope: WSEnvelope = JSON.parse(event.data);
          this.emit(envelope.type, envelope.payload);
        } catch (err) {
          console.error('DVE WebSocket failed to parse message:', err);
        }
      };

      this.ws.onerror = (event: Event) => {
        console.error('DVE WebSocket error:', event);
        this.emit('_error', { error: event });
      };

      this.ws.onclose = (event: CloseEvent) => {
        console.info(`DVE WebSocket closed (code=${event.code})`);
        this.isConnected = false;
        this.stopHeartbeatInterval();
        this.emit('_disconnected', { code: event.code, reason: event.reason });

        if (this.shouldReconnect) {
          this.reconnect();
        }
      };
    } catch (err) {
      console.error('DVE WebSocket failed to create connection:', err);
      if (this.shouldReconnect) {
        this.reconnect();
      }
    }
  }

  /**
   * Close the WebSocket connection and stop reconnecting.
   */
  disconnect(): void {
    this.shouldReconnect = false;
    this.stopHeartbeatInterval();

    if (this.ws) {
      this.ws.onclose = null; // prevent reconnect
      this.ws.close(1000, 'Client disconnecting');
      this.ws = null;
    }

    this.isConnected = false;
  }

  /**
   * Send a typed message over the WebSocket.
   */
  send(type: string, payload: any): void {
    if (!this.ws || this.ws.readyState !== WebSocket.OPEN) {
      console.warn('DVE WebSocket not open, cannot send message');
      return;
    }

    const envelope: WSEnvelope = {
      type,
      timestamp: Date.now(),
      payload,
    };

    this.ws.send(JSON.stringify(envelope));
  }

  /**
   * Register a handler for a specific message type.
   * Returns an unsubscribe function.
   */
  on(type: string, handler: WSMessageHandler): () => void {
    const handlers = this.handlers.get(type) || [];
    handlers.push(handler);
    this.handlers.set(type, handlers);

    return () => this.off(type, handler);
  }

  /**
   * Remove a previously registered handler.
   */
  off(type: string, handler: WSMessageHandler): void {
    const handlers = this.handlers.get(type);
    if (!handlers) {
      return;
    }

    const filtered = handlers.filter((h) => h !== handler);
    if (filtered.length === 0) {
      this.handlers.delete(type);
    } else {
      this.handlers.set(type, filtered);
    }
  }

  /**
   * Send a heartbeat ping to the server.
   */
  sendHeartbeat(): void {
    this.send('heartbeat', {
      timestamp: Date.now(),
    });
  }

  /**
   * Send a task result back to the server.
   */
  sendTaskResult(result: TaskResultPayload): void {
    this.send('task_result', result);
  }

  /**
   * Attempt to reconnect with exponential backoff.
   */
  private reconnect(): void {
    if (!this.shouldReconnect) {
      return;
    }

    if (this.reconnectAttempts >= this.maxReconnectAttempts) {
      console.error('DVE WebSocket max reconnect attempts reached');
      this.emit('_reconnect_failed', { attempts: this.reconnectAttempts });
      return;
    }

    const delay = Math.min(
      DVE_CONSTANTS.WS_RECONNECT_BASE_DELAY_MS * Math.pow(2, this.reconnectAttempts) +
        Math.random() * 1_000,
      DVE_CONSTANTS.WS_RECONNECT_MAX_DELAY_MS,
    );

    this.reconnectAttempts++;
    console.info(
      `DVE WebSocket reconnecting in ${delay}ms (attempt ${this.reconnectAttempts})`,
    );

    setTimeout(() => {
      if (this.shouldReconnect) {
        this.connect();
      }
    }, delay);
  }

  /**
   * Start the periodic heartbeat interval.
   */
  private startHeartbeatInterval(): void {
    this.stopHeartbeatInterval();
    this.heartbeatInterval = setInterval(() => {
      this.sendHeartbeat();
    }, DVE_CONSTANTS.HEARTBEAT_INTERVAL_MS);
  }

  /**
   * Stop the periodic heartbeat interval.
   */
  private stopHeartbeatInterval(): void {
    if (this.heartbeatInterval) {
      clearInterval(this.heartbeatInterval);
      this.heartbeatInterval = null;
    }
  }

  /**
   * Emit a message to all registered handlers of a given type.
   */
  private emit(type: string, payload: any): void {
    const handlers = this.handlers.get(type);
    if (!handlers) {
      return;
    }
    for (const handler of handlers) {
      try {
        handler(payload);
      } catch (err) {
        console.error(`DVE WebSocket handler error for type "${type}":`, err);
      }
    }
  }

  /**
   * Build the full WebSocket URL with auth token as query parameter.
   */
  private buildWSURL(): string {
    const base = this.serverURL.replace(/^http/, 'ws');
    const wsPath = DVE_CONSTANTS.WS_PATH;
    const url = `${base}${wsPath}`;
    if (this.authToken) {
      return `${url}?token=${encodeURIComponent(this.authToken)}`;
    }
    return url;
  }
}
