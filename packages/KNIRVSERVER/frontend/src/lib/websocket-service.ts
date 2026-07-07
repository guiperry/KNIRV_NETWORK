"use client";

import { StandardWebSocket, WebSocketMessage } from './api';

type EventHandler = (data: any) => void;

export const WS_EVENTS = {
  CONNECTION: 'connection',
  NEURAL_TASK: 'neural_task',
  NEURAL_TASK_RESULT: 'neural_task_result',
  NEURAL_TASK_ERROR: 'neural_task_error',
  COGNITIVE_ENGINE_ACTIVITY: 'cognitive-engine-activity',
  NODE_STATUS: 'node:status',
  NODE_METRICS: 'node:metrics',
  NODE_TASK: 'node:task',
  TASK_PROGRESS: 'task:progress',
  TASK_COMPLETE: 'task:complete',
  TASK_FAILED: 'task:failed',
  POLICY_VIOLATION: 'policy:violation',
  POLICY_UPDATE: 'policy:update',
  CHAIN_SESSION: 'chain:session',
  SECRET_RETRIEVED: 'secret:retrieved',
  EVIDENCE_PACK: 'evidence:pack',
  GUARDRAIL_TRIGGER: 'guardrail:trigger',
  SYSTEM_HEALTH: 'system:health',
  P2P_DISCOVERY: 'p2p:discovery',
  P2P_PEERS: 'p2p:peers',
  P2P_TOPOLOGY: 'p2p:topology',
  WORKFLOW_UPDATE: 'workflow:update',
  ONTOLOGY_INGEST: 'ontology:ingest',
  EMBEDDING_UPDATE: 'embedding:update',
  ALIGNMENT_DRIFT: 'alignment:drift',
} as const;

export type WS_EVENT_TYPE = typeof WS_EVENTS[keyof typeof WS_EVENTS];

class WebSocketService {
  private static instance: WebSocketService | null = null;
  private ws: StandardWebSocket | null = null;
  private eventHandlers: Map<string, Set<EventHandler>> = new Map();
  private isConnected = false;
  private subscribers = new Set<string>();
  private reconnectAttempts = 0;
  private maxReconnectAttempts = 5;
  private reconnectDelay = 3000;

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
      this.reconnectAttempts = 0;
      
      if (this.subscribers.size > 0) {
        this.ws!.subscribe(Array.from(this.subscribers));
      }
      
      this.emit(WS_EVENTS.CONNECTION, { connected: true });
    };

    this.ws.onClose = () => {
      console.log('Centralized WebSocket disconnected');
      this.isConnected = false;
      this.emit(WS_EVENTS.CONNECTION, { connected: false });
      
      this.attemptReconnect();
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

  private attemptReconnect(): void {
    if (this.reconnectAttempts < this.maxReconnectAttempts) {
      this.reconnectAttempts++;
      console.log(`Attempting to reconnect (${this.reconnectAttempts}/${this.maxReconnectAttempts})...`);
      setTimeout(() => this.connect(), this.reconnectDelay);
    }
  }

  public subscribe(events: string[]): void {
    events.forEach(event => this.subscribers.add(event));
    
    if (this.ws && this.isConnected) {
      this.ws.subscribe(events);
    }
  }

  public subscribeToAll(): void {
    this.subscribe([
      WS_EVENTS.NEURAL_TASK_RESULT,
      WS_EVENTS.NEURAL_TASK_ERROR,
      WS_EVENTS.COGNITIVE_ENGINE_ACTIVITY,
      WS_EVENTS.NODE_STATUS,
      WS_EVENTS.NODE_METRICS,
      WS_EVENTS.TASK_PROGRESS,
      WS_EVENTS.TASK_COMPLETE,
      WS_EVENTS.TASK_FAILED,
      WS_EVENTS.POLICY_VIOLATION,
      WS_EVENTS.POLICY_UPDATE,
      WS_EVENTS.SYSTEM_HEALTH,
      WS_EVENTS.P2P_DISCOVERY,
      WS_EVENTS.P2P_PEERS,
      WS_EVENTS.P2P_TOPOLOGY,
      WS_EVENTS.WORKFLOW_UPDATE,
      WS_EVENTS.GUARDRAIL_TRIGGER,
    ]);
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

export const webSocketService = WebSocketService.getInstance();
