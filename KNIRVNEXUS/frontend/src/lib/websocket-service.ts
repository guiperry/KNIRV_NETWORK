"use client";

import { StandardWebSocket, WebSocketMessage } from './api';

type EventHandler = (data: any) => void;

class WebSocketService {
  private static instance: WebSocketService | null = null;
  private ws: StandardWebSocket | null = null;
  private eventHandlers: Map<string, Set<EventHandler>> = new Map();
  private isConnected = false;
  private subscribers = new Set<string>();

  private constructor() {
    if (typeof window !== 'undefined') {
      this.connect();
    }
  }

  public static getInstance(): WebSocketService {
    if (!WebSocketService.instance) {
      WebSocketService.instance = new WebSocketService();
    }
    return WebSocketService.instance;
  }

  private connect(): void {
    if (this.ws) {
      this.ws.close();
    }

    this.ws = new StandardWebSocket();

    this.ws.onOpen = () => {
      console.log('Centralized WebSocket connected');
      this.isConnected = true;
      
      // Resubscribe to all topics
      if (this.subscribers.size > 0) {
        this.ws!.subscribe(Array.from(this.subscribers));
      }
      
      // Notify all connection handlers
      this.emit('connection', { connected: true });
    };

    this.ws.onClose = () => {
      console.log('Centralized WebSocket disconnected');
      this.isConnected = false;
      this.emit('connection', { connected: false });
    };

    this.ws.onError = (error) => {
      console.error('Centralized WebSocket error:', error);
      this.emit('error', error);
    };

    this.ws.onMessage = (message: WebSocketMessage) => {
      if (message.event) {
        this.emit(message.event, message.payload);
      }
    };
  }

  public subscribe(events: string[]): void {
    events.forEach(event => this.subscribers.add(event));
    
    if (this.ws && this.isConnected) {
      this.ws.subscribe(events);
    }
  }

  public unsubscribe(events: string[]): void {
    events.forEach(event => this.subscribers.delete(event));
    
    if (this.ws && this.isConnected) {
      this.ws.unsubscribe(events);
    }
  }

  public on(event: string, handler: EventHandler): void {
    if (!this.eventHandlers.has(event)) {
      this.eventHandlers.set(event, new Set());
    }
    this.eventHandlers.get(event)!.add(handler);
  }

  public off(event: string, handler: EventHandler): void {
    const handlers = this.eventHandlers.get(event);
    if (handlers) {
      handlers.delete(handler);
      if (handlers.size === 0) {
        this.eventHandlers.delete(event);
      }
    }
  }

  private emit(event: string, data: any): void {
    const handlers = this.eventHandlers.get(event);
    if (handlers) {
      handlers.forEach(handler => {
        try {
          handler(data);
        } catch (error) {
          console.error(`Error in WebSocket event handler for ${event}:`, error);
        }
      });
    }
  }

  public send(message: WebSocketMessage): void {
    if (this.ws && this.isConnected) {
      this.ws.send(message);
    } else {
      console.warn('WebSocket not connected, message not sent:', message);
    }
  }

  public getConnectionStatus(): boolean {
    return this.isConnected;
  }

  public disconnect(): void {
    if (this.ws) {
      this.ws.close();
      this.ws = null;
    }
    this.isConnected = false;
    this.eventHandlers.clear();
    this.subscribers.clear();
  }
}

// Export singleton instance
export const webSocketService = WebSocketService.getInstance();
