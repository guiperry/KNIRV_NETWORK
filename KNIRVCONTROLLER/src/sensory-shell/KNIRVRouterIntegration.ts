import { EventEmitter } from 'events';
import { logger } from '../core/utils/logger';

export interface KNIRVRouterConfig {
  routerUrl: string;
  graphUrl: string;
  oracleUrl: string;
  timeout: number;
  retryAttempts: number;
  enableP2P: boolean;
  enableWASM: boolean;
}

export interface ErrorContext {
  errorId: string;
  errorType: string;
  errorMessage: string;
  stackTrace: string;
  userContext: unknown;
  agentId: string;
  timestamp: number;
  severity: 'low' | 'medium' | 'high' | 'critical';
}

export interface SkillNodeURI {
  nodeId: string;
  skillId: string;
  routerAddress: string;
  networkPath: string;
  capabilities: string[];
  confidence: number;
  p2pAddress?: string;
  wasmModule?: string;
}

export interface KNIRVRouterRequest {
  requestId: string;
  errorContext: ErrorContext;
  requiredCapabilities: string[];
  nrnToken: string;
  agentId: string;
  priority: 'low' | 'normal' | 'high';
  timestamp: number;
  p2pRouting?: boolean;
  wasmExecution?: boolean;
}

export interface KNIRVRouterResponse {
  requestId: string;
  status: 'SUCCESS' | 'FAILURE' | 'NOT_FOUND' | 'ROUTING';
  skillNodeUri?: SkillNodeURI;
  loraAdapter?: LoRAAdapterData;
  errorMessage?: string;
  executionTime: number;
  networkLatency: number;
  routingPath?: string[];
  wasmResult?: unknown;
}

export interface LoRAAdapterData {
  adapterId: string;
  adapterName: string;
  description: string;
  baseModelCompatibility: string;
  version: number;
  rank: number;
  alpha: number;
  weightsA: Float32Array;
  weightsB: Float32Array;
  metadata: Record<string, string>;
  createdAt: Date;
  lastUsed: Date;
  usageCount: number;
  networkScore: number;
  routerNodes: string[];
}

export interface LoRAAdapterFilter {
  [key: string]: string | string[] | number | boolean | undefined;
}

export interface P2PRoutingInfo {
  sourceNode: string;
  targetNode: string;
  routingPath: string[];
  latency: number;
  reliability: number;
}

export interface WASMExecutionContext {
  wasmModule: string;
  functionName: string;
  parameters: unknown;
  memoryLimit: number;
  timeoutMs: number;
}

/**
 * Revolutionary KNIRVROUTER Integration
 * Implements ErrorContext → KNIRVGRAPH → KNIRVROUTER → SkillNode architecture
 */
export class KNIRVRouterIntegration extends EventEmitter {
  private config: KNIRVRouterConfig;
  private isConnected: boolean = false;
  private connectionRetries: number = 0;
  private activeRequests: Map<string, KNIRVRouterRequest> = new Map();
  private routingCache: Map<string, SkillNodeURI[]> = new Map();
  private p2pConnections: Map<string, WebSocket> = new Map();
  private retryTimeouts: Set<NodeJS.Timeout | number> = new Set();
  private isShuttingDown: boolean = false;

  constructor(config: Partial<KNIRVRouterConfig> = {}) {
    super();
    
    this.config = {
      routerUrl: 'http://localhost:5000',
      graphUrl: 'http://localhost:5001',
      oracleUrl: 'http://localhost:5002',
      timeout: 30000,
      retryAttempts: 3,
      enableP2P: true,
      enableWASM: true,
      ...config,
    };

    this.initializeConnection();
  }

  /**
   * Initialize connection to KNIRVROUTER network
   */
  private async initializeConnection(): Promise<void> {
    try {
      logger.info('Initializing KNIRVROUTER connection...');
      
      // Test router connectivity
      const healthCheck = await this.performHealthCheck();
      if (healthCheck.success) {
        this.isConnected = true;
        this.connectionRetries = 0;
        logger.info('KNIRVROUTER connection established');
        
        // Initialize P2P if enabled
        if (this.config.enableP2P) {
          await this.initializeP2PConnections();
        }
        
        this.emit('connected', { timestamp: Date.now() });
      } else {
        throw new Error(`Health check failed: ${healthCheck.error}`);
      }
    } catch (error) {
      logger.error({ error }, 'Failed to initialize KNIRVROUTER connection');
      this.handleConnectionFailure();
    }
  }

  /**
   * Perform health check on KNIRVROUTER services
   */
  private async performHealthCheck(): Promise<{ success: boolean; error?: string }> {
    try {
      const controller = this.createTimeoutController(5000);
      const response = await fetch(`${this.config.routerUrl}/health`, {
        method: 'GET',
        headers: { 'Content-Type': 'application/json' },
        signal: controller.signal
      });

      if (response.ok) {
        const health = await response.json();
        return { success: health.status === 'healthy' };
      } else {
        return { success: false, error: `HTTP ${response.status}` };
      }
    } catch (error) {
      return { success: false, error: error instanceof Error ? error.message : 'Unknown error' };
    }
  }

  /**
   * Initialize P2P connections for direct routing
   */
  private async initializeP2PConnections(): Promise<void> {
    try {
      logger.info('Initializing P2P connections...');
      
      const p2pEndpoint = `${this.config.routerUrl.replace('http', 'ws')}/p2p`;
      const ws = new WebSocket(p2pEndpoint);
      
      ws.onopen = () => {
        logger.info('P2P WebSocket connection established');
        this.p2pConnections.set('main', ws);
        this.emit('p2pConnected', { endpoint: p2pEndpoint });
      };
      
      ws.onmessage = (event) => {
        this.handleP2PMessage(JSON.parse(event.data));
      };
      
      ws.onerror = (error) => {
        logger.error({ error }, 'P2P WebSocket error');
        this.emit('p2pError', { error });
      };
      
    } catch (error) {
      logger.warn({ error }, 'Failed to initialize P2P connections');
    }
  }

  /**
   * Handle P2P messages for direct routing
   */
  private handleP2PMessage(message: unknown): void {
    try {
      const messageAny = message as { type?: string; data?: unknown };
      switch (messageAny.type) {
        case 'skill_node_discovered':
          this.handleSkillNodeDiscovery(messageAny.data);
          break;
        case 'routing_update':
          this.handleRoutingUpdate(messageAny.data);
          break;
        case 'wasm_execution_result':
          this.handleWASMExecutionResult(messageAny.data);
          break;
        default:
          logger.debug({ message }, 'Unknown P2P message type');
      }
    } catch (error) {
      logger.error({ error, message }, 'Error handling P2P message');
    }
  }

  /**
   * Revolutionary ErrorContext → KNIRVGRAPH → KNIRVROUTER skill resolution
   */
  public async resolveSkillViaErrorContext(
    errorContext: ErrorContext,
    requiredCapabilities: string[] = [],
    options: {
      priority?: 'low' | 'normal' | 'high';
      useP2P?: boolean;
      useWASM?: boolean;
      nrnToken?: string;
    } = {}
  ): Promise<KNIRVRouterResponse> {
    const requestId = `req_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
    
    try {
      logger.info({ requestId, errorContext }, 'Resolving skill via ErrorContext');
      
      // Step 1: Query KNIRVGRAPH for similar error patterns
      const graphResults = await this.queryKNIRVGraphForPatterns(errorContext);
      
      // Step 2: Submit to KNIRVROUTER for skill resolution
      const routerRequest: KNIRVRouterRequest = {
        requestId,
        errorContext,
        requiredCapabilities,
        nrnToken: options.nrnToken || 'default-token',
        agentId: errorContext.agentId,
        priority: options.priority || 'normal',
        timestamp: Date.now(),
        p2pRouting: options.useP2P && this.config.enableP2P,
        wasmExecution: options.useWASM && this.config.enableWASM
      };
      
      this.activeRequests.set(requestId, routerRequest);
      
      // Step 3: Route through KNIRVROUTER
      const response = await this.submitToKNIRVRouter(routerRequest, graphResults);
      
      // Step 4: Handle P2P routing if enabled
      if (options.useP2P && response.skillNodeUri?.p2pAddress) {
        await this.establishP2PRoute(response.skillNodeUri);
      }
      
      // Step 5: Execute WASM if enabled
      if (options.useWASM && response.skillNodeUri?.wasmModule) {
        response.wasmResult = await this.executeWASMSkill(response.skillNodeUri);
      }
      
      this.activeRequests.delete(requestId);
      
      this.emit('skillResolved', {
        requestId,
        response,
        timestamp: Date.now()
      });
      
      return response;

    } catch (error) {
      this.activeRequests.delete(requestId);
      logger.error({ error, requestId }, 'Failed to resolve skill via ErrorContext');

      return {
        requestId,
        status: 'FAILURE',
        errorMessage: error instanceof Error ? error.message : 'Unknown error',
        executionTime: 0,
        networkLatency: 0
      };
    }
  }

  /**
   * Query KNIRVGRAPH for similar error patterns and skill nodes
   */
  private async queryKNIRVGraphForPatterns(errorContext: ErrorContext): Promise<unknown> {
    try {
      const graphQuery = {
        errorContext,
        similarityThreshold: 0.7,
        maxResults: 10,
        includeSkillNodes: true,
        includeRoutingInfo: true
      };

      const controller = this.createTimeoutController(this.config.timeout);
      const response = await fetch(`${this.config.graphUrl}/query-error-patterns`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(graphQuery),
        signal: controller.signal
      });

      if (!response.ok) {
        throw new Error(`KNIRVGRAPH query failed: ${response.statusText}`);
      }

      const results = await response.json();
      logger.debug({ results }, 'KNIRVGRAPH query results');

      return results;
    } catch (error) {
      logger.warn({ error }, 'KNIRVGRAPH query failed, proceeding without graph data');
      return { patterns: [], skillNodes: [] };
    }
  }

  /**
   * Submit request to KNIRVROUTER for skill resolution
   */
  private async submitToKNIRVRouter(
    request: KNIRVRouterRequest,
    graphResults: unknown
  ): Promise<KNIRVRouterResponse> {
    const startTime = Date.now();

    try {
      const routerPayload = {
        ...request,
        graphResults,
        routingPreferences: {
          preferP2P: request.p2pRouting,
          preferWASM: request.wasmExecution,
          maxHops: 5,
          timeoutMs: this.config.timeout
        }
      };

      const controller = this.createTimeoutController(this.config.timeout);
      const response = await fetch(`${this.config.routerUrl}/resolve-skill`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${request.nrnToken}`,
          'X-Agent-ID': request.agentId
        },
        body: JSON.stringify(routerPayload),
        signal: controller.signal
      });

      if (!response.ok) {
        throw new Error(`KNIRVROUTER resolution failed: ${response.statusText}`);
      }

      const result = await response.json();
      const executionTime = Date.now() - startTime;

      return {
        ...result,
        executionTime,
        networkLatency: result.networkLatency || executionTime
      };
    } catch (error) {

      throw new Error(`KNIRVROUTER submission failed: ${error instanceof Error ? error.message : 'Unknown error'}`);
    }
  }

  /**
   * Establish P2P route to skill node
   */
  private async establishP2PRoute(skillNode: SkillNodeURI): Promise<void> {
    if (!skillNode.p2pAddress) {
      logger.warn('No P2P address available for skill node');
      return;
    }

    try {
      logger.info({ skillNode }, 'Establishing P2P route to skill node');

      const p2pWs = new WebSocket(skillNode.p2pAddress);

      p2pWs.onopen = () => {
        logger.info('P2P route established to skill node');
        this.p2pConnections.set(skillNode.nodeId, p2pWs);
        this.emit('p2pRouteEstablished', { skillNode });
      };

      p2pWs.onmessage = (event) => {
        const message = JSON.parse(event.data);
        this.emit('p2pSkillMessage', { skillNode, message });
      };

      p2pWs.onerror = (error) => {
        logger.error({ error, skillNode }, 'P2P route error');
        this.emit('p2pRouteError', { skillNode, error });
      };

    } catch (error) {
      logger.error({ error, skillNode }, 'Failed to establish P2P route');
    }
  }

  /**
   * Execute WASM skill on skill node
   */
  private async executeWASMSkill(skillNode: SkillNodeURI): Promise<unknown> {
    if (!skillNode.wasmModule) {
      logger.warn('No WASM module available for skill node');
      return null;
    }

    try {
      logger.info({ skillNode }, 'Executing WASM skill');

      const wasmContext: WASMExecutionContext = {
        wasmModule: skillNode.wasmModule,
        functionName: 'execute_skill',
        parameters: {},
        memoryLimit: 64 * 1024 * 1024, // 64MB
        timeoutMs: 30000
      };

      const controller = this.createTimeoutController(wasmContext.timeoutMs);
      const response = await fetch(`${this.config.routerUrl}/execute-wasm`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(wasmContext),
        signal: controller.signal
      });

      if (!response.ok) {
        throw new Error(`WASM execution failed: ${response.statusText}`);
      }

      const result = await response.json();
      logger.info({ result }, 'WASM skill execution completed');

      this.emit('wasmSkillExecuted', { skillNode, result });
      return result;

    } catch (error) {
      logger.error({ error, skillNode }, 'WASM skill execution failed');
      return { error: error instanceof Error ? error.message : 'Unknown error' };
    }
  }

  /**
   * Handle skill node discovery from P2P network
   */
  private handleSkillNodeDiscovery(data: unknown): void {
    try {
      const skillNode: SkillNodeURI = (data as { skillNode: SkillNodeURI }).skillNode;

      // Cache the discovered skill node
      const cacheKey = `${skillNode.skillId}_${skillNode.capabilities.join('_')}`;
      if (!this.routingCache.has(cacheKey)) {
        this.routingCache.set(cacheKey, []);
      }
      this.routingCache.get(cacheKey)!.push(skillNode);

      logger.info({ skillNode }, 'Skill node discovered via P2P');
      this.emit('skillNodeDiscovered', { skillNode, source: 'p2p' });

    } catch (error) {
      logger.error({ error, data }, 'Error handling skill node discovery');
    }
  }

  /**
   * Handle routing updates from P2P network
   */
  private handleRoutingUpdate(data: unknown): void {
    try {
      const routingInfo: P2PRoutingInfo = (data as { routingInfo: P2PRoutingInfo }).routingInfo;

      logger.debug({ routingInfo }, 'Routing update received');
      this.emit('routingUpdate', { routingInfo });

    } catch (error) {
      logger.error({ error, data }, 'Error handling routing update');
    }
  }

  /**
   * Handle WASM execution results from P2P network
   */
  private handleWASMExecutionResult(data: unknown): void {
    try {
      const { requestId, result, error } = data as { requestId?: string; result?: unknown; error?: string };

      logger.info({ requestId, result, error }, 'WASM execution result received');
      this.emit('wasmExecutionResult', { requestId, result, error });

    } catch (error) {
      logger.error({ error, data }, 'Error handling WASM execution result');
    }
  }

  /**
   * Handle connection failures with retry logic
   */
  private handleConnectionFailure(): void {
    if (this.isShuttingDown) {
      return; // Don't retry if we're shutting down
    }

    this.isConnected = false;
    this.connectionRetries++;

    if (this.connectionRetries <= this.config.retryAttempts) {
      const retryDelay = Math.min(1000 * Math.pow(2, this.connectionRetries), 30000);
      logger.warn(`KNIRVROUTER connection failed, retrying in ${retryDelay}ms (attempt ${this.connectionRetries}/${this.config.retryAttempts})`);

      const timeout = setTimeout(() => {
        this.retryTimeouts.delete(timeout);
        if (!this.isShuttingDown) {
          this.initializeConnection();
        }
      }, retryDelay);

      this.retryTimeouts.add(timeout);
    } else {
      logger.error('KNIRVROUTER connection failed permanently after maximum retries');
      this.emit('connectionFailed', {
        retries: this.connectionRetries,
        maxRetries: this.config.retryAttempts
      });
    }
  }

  /**
   * Get LoRA adapters from KNIRVROUTER network
   */
  public async getLoRAAdapters(filter?: LoRAAdapterFilter): Promise<LoRAAdapterData[]> {
    if (!this.isConnected) {
      throw new Error('KNIRVROUTER not connected');
    }

    try {
      let url = `${this.config.routerUrl}/lora-adapters`;

      if (filter) {
        const params = new URLSearchParams();
        Object.keys(filter).forEach(key => {
          const value = filter[key];
          if (value !== undefined) {
            if (Array.isArray(value)) {
              value.forEach((item: string) => params.append(key, item));
            } else {
              params.append(key, value.toString());
            }
          }
        });
        if (params.toString()) {
          url += `?${params.toString()}`;
        }
      }

      const controller = this.createTimeoutController(this.config.timeout);
      const response = await fetch(url, {
        headers: { 'Content-Type': 'application/json' },
        signal: controller.signal
      });

      if (!response.ok) {
        throw new Error(`Failed to get LoRA adapters: ${response.statusText}`);
      }

      const result = await response.json();
      return result.adapters || [];

    } catch (error) {
      logger.error({ error }, 'Failed to get LoRA adapters from KNIRVROUTER');
      throw error;
    }
  }

  /**
   * Register LoRA adapter with KNIRVROUTER network
   */
  public async registerLoRAAdapter(adapter: Omit<LoRAAdapterData, 'createdAt' | 'lastUsed' | 'usageCount' | 'networkScore' | 'routerNodes'>): Promise<string> {
    if (!this.isConnected) {
      throw new Error('KNIRVROUTER not connected');
    }

    try {
      const controller = this.createTimeoutController(this.config.timeout);
      const response = await fetch(`${this.config.routerUrl}/lora-adapters`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(adapter),
        signal: controller.signal
      });

      if (!response.ok) {
        throw new Error(`LoRA adapter registration failed: ${response.statusText}`);
      }

      const result = await response.json();

      this.emit('loraAdapterRegistered', {
        adapterId: result.adapter_id,
        adapterName: adapter.adapterName,
        timestamp: Date.now()
      });

      return result.adapter_id;

    } catch (error) {
      logger.error({ error }, 'Failed to register LoRA adapter with KNIRVROUTER');
      throw error;
    }
  }

  /**
   * Get connection status
   */
  public isRouterConnected(): boolean {
    return this.isConnected;
  }

  /**
   * Get active requests count
   */
  public getActiveRequestsCount(): number {
    return this.activeRequests.size;
  }

  /**
   * Get P2P connections count
   */
  public getP2PConnectionsCount(): number {
    return this.p2pConnections.size;
  }

  /**
   * Get routing cache size
   */
  public getRoutingCacheSize(): number {
    return this.routingCache.size;
  }

  /**
   * Clear routing cache
   */
  public clearRoutingCache(): void {
    this.routingCache.clear();
    logger.info('Routing cache cleared');
  }

  /**
   * Disconnect from KNIRVROUTER
   */
  public async disconnect(): Promise<void> {
    try {
      this.isShuttingDown = true;
      this.isConnected = false;

      // Clear all retry timeouts
      this.retryTimeouts.forEach((timeout) => {
        clearTimeout(timeout);
      });
      this.retryTimeouts.clear();

      // Close P2P connections
      this.p2pConnections.forEach((ws, nodeId) => {
        ws.close();
        logger.debug(`Closed P2P connection to ${nodeId}`);
      });
      this.p2pConnections.clear();

      // Clear active requests
      this.activeRequests.clear();

      // Clear routing cache
      this.routingCache.clear();

      logger.info('KNIRVROUTER disconnected');
      this.emit('disconnected', { timestamp: Date.now() });

    } catch (error) {
      logger.error({ error }, 'Error during KNIRVROUTER disconnection');
    }
  }

  /**
   * Create an AbortController with timeout for fetch requests
   * This is a polyfill for AbortSignal.timeout which is not available in all Node.js versions
   */
  private createTimeoutController(timeoutMs: number): AbortController {
    const controller = new AbortController();
    const timeout = setTimeout(() => {
      controller.abort();
    }, timeoutMs);

    this.retryTimeouts.add(timeout);

    // Clean up timeout when request completes
    controller.signal.addEventListener('abort', () => {
      this.retryTimeouts.delete(timeout);
    });

    return controller;
  }
}
