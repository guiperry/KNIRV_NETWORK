/**
 * Component Bridge - Unified communication system between KNIRV-CONTROLLER components
 * Enables seamless integration between manager, receiver, and CLI components
 */

export interface ComponentMessage {
  id: string;
  type: string;
  source: string;
  target: string;
  payload: unknown;
  timestamp: number;
}

export interface ComponentConfig {
  name: string;
  port: number;
  endpoints: Record<string, string>;
  features: Record<string, boolean>;
}

export interface CognitiveState {
  hrmActive: boolean;
  loraAdapters: string[];
  currentSkill?: string;
  learningMode: boolean;
  confidence: number;
}

export interface WalletState {
  connected: boolean;
  balance: number;
  address?: string;
  transactions: unknown[];
}

export interface SystemState {
  components: Record<string, 'running' | 'stopped' | 'error'>;
  cognitive: CognitiveState;
  wallet: WalletState;
  network: {
    connected: boolean;
    peers: number;
    blockHeight: number;
  };
}

export class ComponentBridge {
  private ws: WebSocket | null = null;
  private messageHandlers: Map<string, (message: ComponentMessage) => void> = new Map();
  private state: SystemState;
  private config: ComponentConfig;
  private reconnectAttempts = 0;
  private maxReconnectAttempts = 5;

  constructor(config: ComponentConfig) {
    this.config = config;
    this.state = this.initializeState();
    this.connect();
  }

  private initializeState(): SystemState {
    return {
      components: {},
      cognitive: {
        hrmActive: false,
        loraAdapters: [],
        learningMode: false,
        confidence: 0
      },
      wallet: {
        connected: false,
        balance: 0,
        transactions: []
      },
      network: {
        connected: false,
        peers: 0,
        blockHeight: 0
      }
    };
  }

  private connect() {
    try {
      this.ws = new WebSocket('ws://localhost:3000');
      
      this.ws.onopen = () => {
        console.log(`[${this.config.name}] Connected to orchestrator`);
        this.reconnectAttempts = 0;
        this.sendMessage('system', 'orchestrator', 'component_ready', {
          component: this.config.name,
          config: this.config
        });
      };

      this.ws.onmessage = (event) => {
        try {
          const message: ComponentMessage = JSON.parse(event.data);
          this.handleMessage(message);
        } catch (error) {
          console.error(`[${this.config.name}] Failed to parse message:`, error);
        }
      };

      this.ws.onclose = () => {
        console.log(`[${this.config.name}] Disconnected from orchestrator`);
        this.attemptReconnect();
      };

      this.ws.onerror = (error) => {
        console.error(`[${this.config.name}] WebSocket error:`, error);
      };
    } catch (error) {
      console.error(`[${this.config.name}] Failed to connect:`, error);
      this.attemptReconnect();
    }
  }

  private attemptReconnect() {
    if (this.reconnectAttempts < this.maxReconnectAttempts) {
      this.reconnectAttempts++;
      const delay = Math.pow(2, this.reconnectAttempts) * 1000;
      console.log(`[${this.config.name}] Reconnecting in ${delay}ms (attempt ${this.reconnectAttempts})`);
      setTimeout(() => this.connect(), delay);
    } else {
      console.error(`[${this.config.name}] Max reconnection attempts reached`);
    }
  }

  private handleMessage(message: ComponentMessage) {
    // Update state based on message
    this.updateState(message);

    // Call registered handlers
    const handler = this.messageHandlers.get(message.type);
    if (handler) {
      handler(message);
    }

    // Broadcast to all handlers if no specific handler
    this.messageHandlers.forEach((handler, type) => {
      if (type === '*') {
        handler(message);
      }
    });
  }

  private updateState(message: ComponentMessage) {
    switch (message.type) {
      case 'component_status':
        this.state.components[message.source] = (message.payload as { status: 'running' | 'stopped' | 'error' }).status;
        break;
      case 'cognitive_update':
        this.state.cognitive = { ...this.state.cognitive, ...(message.payload as Partial<CognitiveState>) };
        break;
      case 'wallet_update':
        this.state.wallet = { ...this.state.wallet, ...(message.payload as Partial<WalletState>) };
        break;
      case 'network_update':
        this.state.network = { ...this.state.network, ...(message.payload as Partial<{ connected: boolean; peers: number; blockHeight: number }>) };
        break;
    }
  }

  public sendMessage(type: string, target: string, action: string, payload: unknown = {}) {
    if (!this.ws || this.ws.readyState !== WebSocket.OPEN) {
      console.warn(`[${this.config.name}] Cannot send message - not connected`);
      return;
    }

    const message: ComponentMessage = {
      id: this.generateId(),
      type,
      source: this.config.name,
      target,
      payload: { action, ...(typeof payload === 'object' && payload !== null ? payload as Record<string, unknown> : {}) },
      timestamp: Date.now()
    };

    this.ws.send(JSON.stringify(message));
  }

  public onMessage(type: string, handler: (message: ComponentMessage) => void) {
    this.messageHandlers.set(type, handler);
  }

  public getState(): SystemState {
    return { ...this.state };
  }

  public updateCognitiveState(updates: Partial<CognitiveState>) {
    this.state.cognitive = { ...this.state.cognitive, ...updates };
    this.sendMessage('cognitive_update', 'broadcast', 'state_change', updates);
  }

  public updateWalletState(updates: Partial<WalletState>) {
    this.state.wallet = { ...this.state.wallet, ...updates };
    this.sendMessage('wallet_update', 'broadcast', 'state_change', updates);
  }

  public invokeSkill(skillName: string, parameters: unknown = {}) {
    this.sendMessage('skill_invocation', 'receiver', 'invoke', {
      skill: skillName,
      parameters
    });
  }

  public requestQRScan() {
    this.sendMessage('qr_request', 'manager', 'scan', {});
  }

  public executeCliCommand(command: string) {
    this.sendMessage('cli_command', 'cli', 'execute', { command });
  }

  public subscribeToEvents(events: string[]) {
    this.sendMessage('subscription', 'orchestrator', 'subscribe', { events });
  }

  private generateId(): string {
    return `${this.config.name}-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`;
  }

  public disconnect() {
    if (this.ws) {
      this.ws.close();
      this.ws = null;
    }
  }
}

// Utility functions for component integration
export class ComponentIntegration {
  static async waitForComponent(bridge: ComponentBridge, componentName: string, timeout = 10000): Promise<boolean> {
    return new Promise((resolve) => {
      const startTime = Date.now();
      
      const checkComponent = () => {
        const state = bridge.getState();
        if (state.components[componentName] === 'running') {
          resolve(true);
          return;
        }
        
        if (Date.now() - startTime > timeout) {
          resolve(false);
          return;
        }
        
        setTimeout(checkComponent, 500);
      };
      
      checkComponent();
    });
  }

  static createCrossComponentCall(bridge: ComponentBridge, target: string, method: string, params: unknown = {}) {
    return new Promise((resolve, reject) => {
      const callId = `call-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`;
      
      const timeout = setTimeout(() => {
        reject(new Error(`Cross-component call timeout: ${target}.${method}`));
      }, 5000);

      bridge.onMessage('call_response', (message) => {
        const payload = message.payload as {
          callId: string;
          success: boolean;
          result?: unknown;
          error?: string;
        };
        if (payload.callId === callId) {
          clearTimeout(timeout);
          if (payload.success) {
            resolve(payload.result);
          } else {
            reject(new Error(payload.error));
          }
        }
      });

      bridge.sendMessage('cross_component_call', target, 'call', {
        callId,
        method,
        params
      });
    });
  }
}

export default ComponentBridge;
