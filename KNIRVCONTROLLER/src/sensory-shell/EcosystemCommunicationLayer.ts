import { EventEmitter } from './EventEmitter';

export interface EcosystemConfig {
  enableWalletIntegration: boolean;
  enableChainIntegration: boolean;
  enableNexusIntegration: boolean;
  enableGatewayIntegration: boolean;
  enableShellIntegration: boolean;
  communicationProtocol: 'websocket' | 'http' | 'p2p';
  heartbeatInterval: number;
  timeoutDuration: number;
  retryAttempts: number;
}

export interface ComponentStatus {
  id: string;
  name: string;
  status: 'online' | 'offline' | 'error' | 'connecting';
  lastHeartbeat: number;
  version: string;
  capabilities: string[];
  metrics: Record<string, unknown>;
}

export interface EcosystemMessage {
  id: string;
  from: string;
  to: string;
  type: 'command' | 'query' | 'response' | 'event' | 'heartbeat';
  payload: unknown;
  timestamp: number;
  priority: 'low' | 'normal' | 'high' | 'critical';
  requiresResponse: boolean;
  correlationId?: string;
}

export interface ServiceEndpoint {
  id: string;
  name: string;
  url: string;
  protocol: 'http' | 'websocket' | 'p2p';
  authentication: {
    type: 'none' | 'bearer' | 'api_key' | 'certificate';
    credentials?: unknown;
  };
  healthCheckPath?: string;
  capabilities: string[];
}

export class EcosystemCommunicationLayer extends EventEmitter {
  private config: EcosystemConfig;
  private components: Map<string, ComponentStatus> = new Map();
  private endpoints: Map<string, ServiceEndpoint> = new Map();
  private messageQueue: EcosystemMessage[] = [];
  private connections: Map<string, unknown> = new Map();
  private isRunning: boolean = false;
  private heartbeatInterval: NodeJS.Timeout | number | null = null;
  private messageHandlers: Map<string, (...args: unknown[]) => unknown> = new Map();

  constructor(config?: Partial<EcosystemConfig>) {
    super();
    
    this.config = {
      enableWalletIntegration: true,
      enableChainIntegration: true,
      enableNexusIntegration: true,
      enableGatewayIntegration: true,
      enableShellIntegration: true,
      communicationProtocol: 'websocket',
      heartbeatInterval: 30000, // 30 seconds
      timeoutDuration: 10000, // 10 seconds
      retryAttempts: 3,
      ...config,
    };

    this.setupDefaultEndpoints();
    this.setupMessageHandlers();
  }

  public async initialize(): Promise<void> {
    console.log('Initializing Ecosystem Communication Layer...');

    try {
      // Register KNIRV-CORTEX as the central coordinator
      this.registerComponent({
        id: 'knirv-cortex',
        name: 'KNIRV-CORTEX',
        status: 'online',
        lastHeartbeat: Date.now(),
        version: '1.0.0',
        capabilities: [
          'cognitive_processing',
          'hrm_integration',
          'neural_networks',
          'adaptive_learning',
          'multi_modal_ai',
        ],
        metrics: {
          uptime: 0,
          processedRequests: 0,
          averageResponseTime: 0,
        },
      });

      // Initialize connections to other KNIRV components
      await this.initializeConnections();

      // Start heartbeat monitoring
      this.startHeartbeatMonitoring();

      // Start message processing
      this.startMessageProcessing();

      this.isRunning = true;
      this.emit('ecosystemInitialized');
      console.log('Ecosystem Communication Layer initialized successfully');

    } catch (error) {
      console.error('Failed to initialize Ecosystem Communication Layer:', error);
      throw error;
    }
  }

  public async shutdown(): Promise<void> {
    console.log('Shutting down Ecosystem Communication Layer...');
    
    this.isRunning = false;

    // Stop heartbeat monitoring
    if (this.heartbeatInterval) {
      clearInterval(this.heartbeatInterval);
      this.heartbeatInterval = null;
    }

    interface ConnectionWithClose {
      close?: () => void;
    }

    // Close all connections
    this.connections.forEach((connection, componentId) => {
      try {
        const connectionTyped = connection as ConnectionWithClose;
        if (connectionTyped.close) {
          connectionTyped.close();
        }
        console.log(`Closed connection to ${componentId}`);
      } catch (error) {
        console.error(`Error closing connection to ${componentId}:`, error);
      }
    });

    this.connections.clear();
    this.components.clear();
    this.messageQueue.length = 0;

    this.emit('ecosystemShutdown');
    console.log('Ecosystem Communication Layer shutdown complete');
  }

  private setupDefaultEndpoints(): void {
    // KNIRV-WALLET endpoints
    if (this.config.enableWalletIntegration) {
      this.registerEndpoint({
        id: 'knirv-wallet',
        name: 'KNIRV-WALLET',
        url: 'http://localhost:8083',
        protocol: 'http',
        authentication: { type: 'none' },
        healthCheckPath: '/api/v1/health',
        capabilities: ['asset_management', 'transactions', 'cross_platform'],
      });
    }

    // KNIRV-CHAIN endpoints
    if (this.config.enableChainIntegration) {
      this.registerEndpoint({
        id: 'knirv-chain',
        name: 'KNIRV-CHAIN',
        url: 'http://localhost:8080',
        protocol: 'http',
        authentication: { type: 'none' },
        healthCheckPath: '/status',
        capabilities: ['blockchain', 'smart_contracts', 'consensus'],
      });
    }

    // KNIRV-NEXUS endpoints
    if (this.config.enableNexusIntegration) {
      this.registerEndpoint({
        id: 'knirv-nexus',
        name: 'KNIRV-NEXUS',
        url: 'http://localhost:8081',
        protocol: 'http',
        authentication: { type: 'none' },
        healthCheckPath: '/api/health',
        capabilities: ['orchestration', 'skill_execution', 'dve'],
      });
    }

    // KNIRV-GATEWAY endpoints
    if (this.config.enableGatewayIntegration) {
      this.registerEndpoint({
        id: 'knirv-gateway',
        name: 'KNIRV-GATEWAY',
        url: 'http://localhost:8000',
        protocol: 'http',
        authentication: { type: 'none' },
        healthCheckPath: '/api/status',
        capabilities: ['web_interface', 'api_gateway', 'presentation'],
      });
    }

    // KNIRV-CLI endpoints
    if (this.config.enableShellIntegration) {
      this.registerEndpoint({
        id: 'knirv-shell',
        name: 'KNIRV-CLI',
        url: 'http://localhost:8082',
        protocol: 'http',
        authentication: { type: 'none' },
        healthCheckPath: '/health',
        capabilities: ['terminal_interface', 'command_execution', 'automation'],
      });
    }
  }

  private setupMessageHandlers(): void {
    // Skill execution requests
    this.messageHandlers.set('execute_skill', async (...args: unknown[]) => {
      const message = args[0] as EcosystemMessage;
      console.log('Handling skill execution request:', message.payload);
      
      // Route to KNIRV-NEXUS for execution
      const response = await this.sendMessage({
        from: 'knirv-cortex',
        to: 'knirv-nexus',
        type: 'command',
        payload: {
          action: 'execute_skill',
          skillId: (message.payload as { skillId?: string; parameters?: unknown }).skillId,
          parameters: (message.payload as { skillId?: string; parameters?: unknown }).parameters,
        },
        priority: 'high',
        requiresResponse: true,
      });

      return response;
    });

    // Wallet operations
    this.messageHandlers.set('wallet_operation', async (...args: unknown[]) => {
      const message = args[0] as EcosystemMessage;
      console.log('Handling wallet operation:', message.payload);
      
      // Route to KNIRV-WALLET
      const response = await this.sendMessage({
        from: 'knirv-cortex',
        to: 'knirv-wallet',
        type: 'command',
        payload: message.payload,
        priority: 'normal',
        requiresResponse: true,
      });

      return response;
    });

    // Blockchain operations
    this.messageHandlers.set('blockchain_operation', async (...args: unknown[]) => {
      const message = args[0] as EcosystemMessage;
      console.log('Handling blockchain operation:', message.payload);
      
      // Route to KNIRV-CHAIN
      const response = await this.sendMessage({
        from: 'knirv-cortex',
        to: 'knirv-chain',
        type: 'command',
        payload: message.payload,
        priority: 'normal',
        requiresResponse: true,
      });

      return response;
    });

    // Gateway operations
    this.messageHandlers.set('gateway_operation', async (...args: unknown[]) => {
      const message = args[0] as EcosystemMessage;
      console.log('Handling gateway operation:', message.payload);
      
      // Route to KNIRV-GATEWAY
      const response = await this.sendMessage({
        from: 'knirv-cortex',
        to: 'knirv-gateway',
        type: 'command',
        payload: message.payload,
        priority: 'low',
        requiresResponse: false,
      });

      return response;
    });
  }

  public registerComponent(component: ComponentStatus): void {
    this.components.set(component.id, component);
    console.log(`Registered component: ${component.name}`);
    this.emit('componentRegistered', component);
  }

  public registerEndpoint(endpoint: ServiceEndpoint): void {
    this.endpoints.set(endpoint.id, endpoint);
    console.log(`Registered endpoint: ${endpoint.name} at ${endpoint.url}`);
    this.emit('endpointRegistered', endpoint);
  }

  public async sendMessage(messageData: Omit<EcosystemMessage, 'id' | 'timestamp'>): Promise<unknown> {
    const message: EcosystemMessage = {
      id: this.generateMessageId(),
      timestamp: Date.now(),
      ...messageData,
    };

    console.log(`Sending message from ${message.from} to ${message.to}:`, message.type);

    try {
      // Add to queue for processing
      this.messageQueue.push(message);

      // If requires response, wait for it
      if (message.requiresResponse) {
        return await this.waitForResponse(message);
      }

      this.emit('messageSent', message);
      return { success: true, messageId: message.id };

    } catch (error) {
      console.error('Failed to send message:', error);
      throw error;
    }
  }

  private async waitForResponse(message: EcosystemMessage): Promise<unknown> {
    return new Promise((resolve, reject) => {
      const timeout = setTimeout(() => {
        reject(new Error(`Message timeout: ${message.id}`));
      }, this.config.timeoutDuration);

      // Listen for response
      const responseHandler = (response: unknown) => {
        if ((response as { correlationId?: string }).correlationId === message.id) {
          clearTimeout(timeout);
          this.off('messageResponse', responseHandler);
          resolve(response);
        }
      };

      this.on('messageResponse', responseHandler);

      // Simulate response for demo (in real implementation, this would come from actual services)
      setTimeout(() => {
        const mockResponse = {
          correlationId: message.id,
          success: true,
          data: { processed: true, timestamp: Date.now() },
        };
        this.emit('messageResponse', mockResponse);
      }, 1000);
    });
  }

  private async initializeConnections(): Promise<void> {
    console.log('Initializing connections to KNIRV ecosystem components...');

    this.endpoints.forEach(async (endpoint) => {
      try {
        await this.connectToEndpoint(endpoint);
      } catch (error) {
        console.error(`Failed to connect to ${endpoint.name}:`, error);
        // Continue with other connections
      }
    });
  }

  private async connectToEndpoint(endpoint: ServiceEndpoint): Promise<void> {
    console.log(`Connecting to ${endpoint.name} at ${endpoint.url}...`);

    try {
      // Perform health check
      const isHealthy = await this.performHealthCheck(endpoint);
      
      if (isHealthy) {
        // Create connection based on protocol
        let connection;
        
        switch (endpoint.protocol) {
          case 'websocket':
            connection = await this.createWebSocketConnection(endpoint);
            break;
          case 'http':
            connection = await this.createHttpConnection(endpoint);
            break;
          case 'p2p':
            connection = await this.createP2PConnection(endpoint);
            break;
          default:
            throw new Error(`Unsupported protocol: ${endpoint.protocol}`);
        }

        this.connections.set(endpoint.id, connection);
        
        // Register component as online
        this.registerComponent({
          id: endpoint.id,
          name: endpoint.name,
          status: 'online',
          lastHeartbeat: Date.now(),
          version: '1.0.0',
          capabilities: endpoint.capabilities,
          metrics: {},
        });

        console.log(`Successfully connected to ${endpoint.name}`);
        this.emit('connectionEstablished', endpoint);

      } else {
        throw new Error('Health check failed');
      }

    } catch (error) {
      console.error(`Connection to ${endpoint.name} failed:`, error);
      
      // Register component as offline
      this.registerComponent({
        id: endpoint.id,
        name: endpoint.name,
        status: 'offline',
        lastHeartbeat: 0,
        version: 'unknown',
        capabilities: endpoint.capabilities,
        metrics: {},
      });

      this.emit('connectionFailed', { endpoint, error });
    }
  }

  private async performHealthCheck(endpoint: ServiceEndpoint): Promise<boolean> {
    if (!endpoint.healthCheckPath) {
      return true; // Assume healthy if no health check path
    }

    try {
      const response = await fetch(`${endpoint.url}${endpoint.healthCheckPath}`, {
        method: 'GET',
      });

      return response.ok;

    } catch (error) {
      console.error(`Health check failed for ${endpoint.name}:`, error);
      return false;
    }
  }

  private async createHttpConnection(endpoint: ServiceEndpoint): Promise<unknown> {
    // HTTP connections are stateless, so we just return a connection object
    return {
      type: 'http',
      endpoint,
      send: async (data: unknown) => {
        const response = await fetch(`${endpoint.url}/api/message`, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(data),
        });
        return response.json();
      },
      close: () => {
        // Nothing to close for HTTP
      },
    };
  }

  private async createWebSocketConnection(endpoint: ServiceEndpoint): Promise<unknown> {
    // Simulate WebSocket connection
    return {
      type: 'websocket',
      endpoint,
      send: async (data: unknown) => {
        console.log(`WebSocket send to ${endpoint.name}:`, data);
        return { success: true };
      },
      close: () => {
        console.log(`WebSocket connection to ${endpoint.name} closed`);
      },
    };
  }

  private async createP2PConnection(endpoint: ServiceEndpoint): Promise<unknown> {
    // Simulate P2P connection
    return {
      type: 'p2p',
      endpoint,
      send: async (data: unknown) => {
        console.log(`P2P send to ${endpoint.name}:`, data);
        return { success: true };
      },
      close: () => {
        console.log(`P2P connection to ${endpoint.name} closed`);
      },
    };
  }

  private startHeartbeatMonitoring(): void {
    this.heartbeatInterval = setInterval(async () => {
      await this.performHeartbeatCheck();
    }, this.config.heartbeatInterval);
  }

  private async performHeartbeatCheck(): Promise<void> {
    const now = Date.now();
    
    this.components.forEach(async (component, componentId) => {
      if (componentId === 'knirv-cortex') {
        // Update our own heartbeat
        component.lastHeartbeat = now;
        return;
      }

      // Check if component is still responsive
      const timeSinceLastHeartbeat = now - component.lastHeartbeat;
      
      if (timeSinceLastHeartbeat > this.config.heartbeatInterval * 2) {
        // Component is unresponsive
        if (component.status !== 'offline') {
          component.status = 'offline';
          this.emit('componentOffline', component);
          console.warn(`Component ${component.name} is offline`);
        }
      } else {
        // Try to ping the component
        try {
          const endpoint = this.endpoints.get(componentId);
          if (endpoint) {
            const isHealthy = await this.performHealthCheck(endpoint);
            if (isHealthy) {
              component.status = 'online';
              component.lastHeartbeat = now;
            }
          }
        } catch (error) {
          console.error(`Heartbeat check failed for ${component.name}:`, error);
        }
      }
    });

    this.emit('heartbeatComplete', {
      timestamp: now,
      componentsOnline: Array.from(this.components.values()).filter(c => c.status === 'online').length,
      componentsTotal: this.components.size,
    });
  }

  private startMessageProcessing(): void {
    const processMessages = async () => {
      while (this.isRunning) {
        if (this.messageQueue.length > 0) {
          const message = this.messageQueue.shift()!;
          await this.processMessage(message);
        }
        
        await new Promise(resolve => setTimeout(resolve, 100)); // 100ms delay
      }
    };

    processMessages();
  }

  private async processMessage(message: EcosystemMessage): Promise<void> {
    try {
      console.log(`Processing message: ${message.type} from ${message.from} to ${message.to}`);

      // Find the target connection
      const connection = this.connections.get(message.to);
      
      if (connection) {
        interface ConnectionWithSend {
          send: (message: EcosystemMessage) => Promise<unknown>;
        }

        // Send message through the connection
        const response = await (connection as ConnectionWithSend).send(message);

        if (message.requiresResponse) {
          this.emit('messageResponse', {
            correlationId: message.id,
            ...(response as Record<string, unknown>),
          });
        }
      } else {
        console.warn(`No connection found for target: ${message.to}`);
        
        if (message.requiresResponse) {
          this.emit('messageResponse', {
            correlationId: message.id,
            success: false,
            error: 'Target not connected',
          });
        }
      }

      this.emit('messageProcessed', message);

    } catch (error) {
      console.error('Error processing message:', error);
      
      if (message.requiresResponse) {
        this.emit('messageResponse', {
          correlationId: message.id,
          success: false,
          error: error instanceof Error ? error.message : String(error),
        });
      }
    }
  }

  private generateMessageId(): string {
    return `msg_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
  }

  public getComponents(): ComponentStatus[] {
    return Array.from(this.components.values());
  }

  public getEndpoints(): ServiceEndpoint[] {
    return Array.from(this.endpoints.values());
  }

  public getComponent(componentId: string): ComponentStatus | null {
    return this.components.get(componentId) || null;
  }

  public isComponentOnline(componentId: string): boolean {
    const component = this.components.get(componentId);
    return component ? component.status === 'online' : false;
  }

  public getEcosystemStatus(): unknown {
    const components = Array.from(this.components.values());
    
    return {
      isRunning: this.isRunning,
      totalComponents: components.length,
      onlineComponents: components.filter(c => c.status === 'online').length,
      offlineComponents: components.filter(c => c.status === 'offline').length,
      errorComponents: components.filter(c => c.status === 'error').length,
      messageQueueLength: this.messageQueue.length,
      activeConnections: this.connections.size,
      lastHeartbeat: Date.now(),
      config: this.config,
    };
  }

  public updateConfig(newConfig: Partial<EcosystemConfig>): void {
    this.config = { ...this.config, ...newConfig };
    this.emit('configUpdated', this.config);
  }
}
