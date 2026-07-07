import { describe, it, expect, beforeEach, afterEach, jest } from '@jest/globals';
import { KNIRVRouterIntegration, ErrorContext, SkillNodeURI, LoRAAdapterData } from '../../src/sensory-shell/KNIRVRouterIntegration';

// Mock fetch
const mockFetch = jest.fn() as jest.MockedFunction<typeof fetch>;
global.fetch = mockFetch;

// Mock WebSocket
global.WebSocket = jest.fn().mockImplementation(() => ({
  onopen: null,
  onmessage: null,
  onerror: null,
  onclose: null,
  close: jest.fn(),
  send: jest.fn(),
  CONNECTING: 0,
  OPEN: 1,
  CLOSING: 2,
  CLOSED: 3
})) as unknown as typeof WebSocket;

// Extended interface for testing private methods
interface KNIRVRouterIntegrationWithPrivates {
  performHealthCheck(): Promise<{ success: boolean }>;
  connectionRetries: number;
  handleConnectionFailure(): void;
  isConnected: boolean;
  activeRequests: Map<string, unknown>;
  routingCache: Map<string, unknown>;
  handleSkillNodeDiscovery(data: { skillNode: unknown }): void;
}

describe('KNIRVRouterIntegration Unit Tests', () => {
  let integration: KNIRVRouterIntegration;

  beforeEach(() => {
    jest.useFakeTimers();
    mockFetch.mockClear();
    
    integration = new KNIRVRouterIntegration({
      routerUrl: 'http://localhost:5000',
      graphUrl: 'http://localhost:5001',
      oracleUrl: 'http://localhost:5002',
      timeout: 5000,
      retryAttempts: 2,
      enableP2P: true,
      enableWASM: true
    });
  });

  afterEach(async () => {
    await integration.disconnect();
    jest.clearAllTimers();
    jest.useRealTimers();
  });

  describe('Configuration', () => {
    it('should initialize with default configuration', () => {
      const defaultIntegration = new KNIRVRouterIntegration();
      expect(defaultIntegration).toBeDefined();
    });

    it('should merge custom configuration with defaults', () => {
      const customIntegration = new KNIRVRouterIntegration({
        routerUrl: 'http://custom:8000',
        timeout: 10000
      });
      expect(customIntegration).toBeDefined();
    });

    it('should handle empty configuration', () => {
      const emptyIntegration = new KNIRVRouterIntegration({});
      expect(emptyIntegration).toBeDefined();
    });
  });

  describe('Connection Management', () => {
    it('should report initial connection status', () => {
      expect(integration.isRouterConnected()).toBe(false);
    });

    it('should handle successful health check', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => ({ status: 'healthy', services: ['router', 'graph'] })
      } as Response);

      // Trigger health check manually
      const healthCheck = await (integration as unknown as KNIRVRouterIntegrationWithPrivates).performHealthCheck();
      expect(healthCheck.success).toBe(true);
    });

    it('should handle failed health check', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: false,
        statusText: 'Service Unavailable'
      } as Response);

      const healthCheck = await (integration as unknown as KNIRVRouterIntegrationWithPrivates).performHealthCheck();
      expect(healthCheck.success).toBe(false);
    });

    it('should handle network errors during health check', async () => {
      mockFetch.mockRejectedValueOnce(new Error('Network error'));

      const healthCheck = await (integration as unknown as KNIRVRouterIntegrationWithPrivates).performHealthCheck();
      expect(healthCheck.success).toBe(false);
    });

    it('should track connection retries', () => {
      // Reset connection retries to 0 for this test
      (integration as unknown as KNIRVRouterIntegrationWithPrivates).connectionRetries = 0;
      expect((integration as unknown as KNIRVRouterIntegrationWithPrivates).connectionRetries).toBe(0);
    });

    it('should handle connection failure with retries', () => {
      // Reset connection retries to 0 for this test
      (integration as unknown as KNIRVRouterIntegrationWithPrivates).connectionRetries = 0;
      const handleFailure = (integration as unknown as KNIRVRouterIntegrationWithPrivates).handleConnectionFailure.bind(integration);

      handleFailure();
      expect((integration as unknown as KNIRVRouterIntegrationWithPrivates).connectionRetries).toBe(1);

      handleFailure();
      expect((integration as unknown as KNIRVRouterIntegrationWithPrivates).connectionRetries).toBe(2);
    });

    it('should stop retrying after max attempts', () => {
      const handleFailure = (integration as unknown as KNIRVRouterIntegrationWithPrivates).handleConnectionFailure.bind(integration);
      const emitSpy = jest.spyOn(integration, 'emit');
      
      // Exceed max retries
      for (let i = 0; i <= 3; i++) {
        handleFailure();
      }
      
      expect(emitSpy).toHaveBeenCalledWith('connectionFailed', expect.any(Object));
    });

    it('should not retry when shutting down', async () => {
      await integration.disconnect();
      
      const handleFailure = (integration as unknown as KNIRVRouterIntegrationWithPrivates).handleConnectionFailure.bind(integration);
      const initialRetries = (integration as unknown as KNIRVRouterIntegrationWithPrivates).connectionRetries;

      handleFailure();

      expect((integration as unknown as KNIRVRouterIntegrationWithPrivates).connectionRetries).toBe(initialRetries);
    });
  });

  describe('Skill Resolution', () => {
    it('should resolve skill via ErrorContext', async () => {
      // Mock KNIRVGRAPH response
      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => ({
          patterns: [{ similarity: 0.8 }],
          skillNodes: [{ nodeId: 'test-node', confidence: 0.9 }]
        })
      } as Response);

      // Mock KNIRVROUTER response
      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => ({
          requestId: 'test-request-123',
          status: 'SUCCESS',
          skillNodeUri: {
            nodeId: 'test-node',
            skillId: 'test-skill',
            routerAddress: 'http://localhost:5000',
            networkPath: '/skills/test-skill',
            capabilities: ['test'],
            confidence: 0.9
          },
          executionTime: 100,
          networkLatency: 25
        })
      } as Response);

      const errorContext: ErrorContext = {
        errorId: 'test-error',
        errorType: 'skill_invocation_request',
        errorMessage: 'Test error',
        stackTrace: 'test stack',
        userContext: {},
        agentId: 'test-agent',
        timestamp: Date.now(),
        severity: 'medium'
      } as ErrorContext;

      const response = await integration.resolveSkillViaErrorContext(errorContext, ['test']);
      
      expect(response.status).toBe('SUCCESS');
      expect(response.requestId).toBe('test-request-123');
    });

    it('should handle KNIRVGRAPH failure gracefully', async () => {
      // Mock KNIRVGRAPH failure
      mockFetch.mockResolvedValueOnce({
        ok: false,
        statusText: 'Service Unavailable'
      } as Response);

      // Mock KNIRVROUTER success
      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => ({
          requestId: 'test-request-124',
          status: 'SUCCESS',
          skillNodeUri: {
            nodeId: 'fallback-node',
            skillId: 'test-skill',
            routerAddress: 'http://localhost:5000',
            networkPath: '/skills/test-skill',
            capabilities: ['test'],
            confidence: 0.7
          },
          executionTime: 150,
          networkLatency: 50
        })
      } as Response);

      const errorContext: ErrorContext = {
        errorId: 'test-error-2',
        errorType: 'skill_invocation_request',
        errorMessage: 'Test error with graph failure',
        stackTrace: 'test stack',
        userContext: {},
        agentId: 'test-agent',
        timestamp: Date.now(),
        severity: 'medium'
      } as ErrorContext;

      const response = await integration.resolveSkillViaErrorContext(errorContext, ['test']);
      
      expect(response.status).toBe('SUCCESS');
      expect(response.skillNodeUri?.nodeId).toBe('fallback-node');
    });

    it('should handle KNIRVROUTER failure', async () => {
      // Mock KNIRVGRAPH call (first fetch) - should succeed
      mockFetch.mockResolvedValueOnce({
        ok: true,
        status: 200,
        statusText: 'OK',
        headers: new Headers(),
        redirected: false,
        type: 'basic',
        url: '',
        clone: () => ({} as Response),
        body: null,
        bodyUsed: false,
        arrayBuffer: async () => new ArrayBuffer(0),
        blob: async () => new Blob(),
        formData: async () => new FormData(),
        text: async () => '',
        bytes: async () => new Uint8Array(),
        json: jest.fn().mockResolvedValue({ patterns: [], skillNodes: [] } as never)
      } as unknown as Response);

      // Mock KNIRVROUTER call (second fetch) - should fail
      mockFetch.mockResolvedValueOnce({
        ok: false,
        statusText: 'Internal Server Error'
      } as Response);

      const errorContext: ErrorContext = {
        errorId: 'test-error-3',
        errorType: 'skill_invocation_request',
        errorMessage: 'Test error',
        stackTrace: 'test stack',
        userContext: {},
        agentId: 'test-agent',
        timestamp: Date.now(),
        severity: 'medium'
      } as ErrorContext;

      const response = await integration.resolveSkillViaErrorContext(errorContext, ['test']);

      expect(response.status).toBe('FAILURE');
      expect(response.errorMessage).toContain('KNIRVROUTER resolution failed: Internal Server Error');
    });

    it('should handle network errors', async () => {
      // Mock KNIRVGRAPH call (first fetch) - should succeed
      mockFetch.mockResolvedValueOnce({
        ok: true,
        status: 200,
        statusText: 'OK',
        headers: new Headers(),
        redirected: false,
        type: 'basic',
        url: '',
        clone: () => ({} as Response),
        body: null,
        bodyUsed: false,
        arrayBuffer: async () => new ArrayBuffer(0),
        blob: async () => new Blob(),
        formData: async () => new FormData(),
        text: async () => '',
        bytes: async () => new Uint8Array(),
        json: jest.fn().mockResolvedValue({ patterns: [], skillNodes: [] } as never)
      } as unknown as Response);

      // Mock KNIRVROUTER call (second fetch) - should fail with network error
      mockFetch.mockRejectedValueOnce(new Error('Network timeout'));

      const errorContext: ErrorContext = {
        errorId: 'test-error-4',
        errorType: 'skill_invocation_request',
        errorMessage: 'Test error',
        stackTrace: 'test stack',
        userContext: {},
        agentId: 'test-agent',
        timestamp: Date.now(),
        severity: 'medium'
      } as ErrorContext;

      const response = await integration.resolveSkillViaErrorContext(errorContext, ['test']);

      expect(response.status).toBe('FAILURE');
      expect(response.errorMessage).toContain('KNIRVROUTER submission failed: Network timeout');
    });
  });

  describe('LoRA Adapter Management', () => {
    it('should get LoRA adapters', async () => {
      const mockAdapters: LoRAAdapterData[] = [
        {
          adapterId: 'adapter-001',
          adapterName: 'Test Adapter',
          description: 'Test LoRA adapter',
          baseModelCompatibility: 'hrm-v1',
          version: 1,
          rank: 16,
          alpha: 0.5,
          weightsA: new Float32Array([1, 2]),
          weightsB: new Float32Array([3, 4]),
          metadata: { test: 'true' },
          createdAt: new Date(),
          lastUsed: new Date(),
          usageCount: 5,
          networkScore: 0.8,
          routerNodes: ['node-1']
        }
      ];

      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => ({ adapters: mockAdapters })
      } as Response);

      // Mock connection status
      (integration as unknown as KNIRVRouterIntegrationWithPrivates).isConnected = true;

      const adapters = await integration.getLoRAAdapters();
      
      expect(adapters).toHaveLength(1);
      expect(adapters[0].adapterId).toBe('adapter-001');
    });

    it('should get LoRA adapters with filter', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => ({ adapters: [] })
      } as Response);

      (integration as unknown as KNIRVRouterIntegrationWithPrivates).isConnected = true;

      const filter = { domain: 'text-processing', minVersion: 1 };
      await integration.getLoRAAdapters(filter);

      expect(mockFetch).toHaveBeenCalledWith(
        expect.stringContaining('domain=text-processing'),
        expect.any(Object)
      );
    });

    it('should register LoRA adapter', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => ({ adapter_id: 'new-adapter-123' })
      } as Response);

      (integration as unknown as KNIRVRouterIntegrationWithPrivates).isConnected = true;

      const adapterData = {
        adapterId: 'test-adapter-id',
        adapterName: 'New Adapter',
        description: 'New test adapter',
        baseModelCompatibility: 'hrm-v1',
        version: 1,
        rank: 32,
        alpha: 0.3,
        weightsA: new Float32Array([1, 2, 3]),
        weightsB: new Float32Array([4, 5, 6]),
        metadata: { author: 'test' }
      };

      const adapterId = await integration.registerLoRAAdapter(adapterData);
      
      expect(adapterId).toBe('new-adapter-123');
    });

    it('should handle LoRA adapter registration failure', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: false,
        statusText: 'Bad Request'
      } as Response);

      (integration as unknown as KNIRVRouterIntegrationWithPrivates).isConnected = true;

      const adapterData = {
        adapterId: 'invalid-adapter-id',
        adapterName: 'Invalid Adapter',
        description: 'Invalid adapter',
        baseModelCompatibility: 'hrm-v1',
        version: 1,
        rank: 16,
        alpha: 0.5,
        weightsA: new Float32Array([1]),
        weightsB: new Float32Array([2]),
        metadata: {}
      };

      await expect(integration.registerLoRAAdapter(adapterData)).rejects.toThrow('Bad Request');
    });

    it('should handle disconnected state for LoRA operations', async () => {
      (integration as unknown as KNIRVRouterIntegrationWithPrivates).isConnected = false;

      await expect(integration.getLoRAAdapters()).rejects.toThrow('not connected');
      
      const adapterData = {
        adapterId: 'test-adapter-id',
        adapterName: 'Test',
        description: 'Test',
        baseModelCompatibility: 'hrm-v1',
        version: 1,
        rank: 16,
        alpha: 0.5,
        weightsA: new Float32Array([1]),
        weightsB: new Float32Array([2]),
        metadata: {}
      };

      await expect(integration.registerLoRAAdapter(adapterData)).rejects.toThrow('not connected');
    });
  });

  describe('Utility Methods', () => {
    it('should return active requests count', () => {
      expect(integration.getActiveRequestsCount()).toBe(0);
    });

    it('should return P2P connections count', () => {
      expect(integration.getP2PConnectionsCount()).toBe(0);
    });

    it('should return routing cache size', () => {
      expect(integration.getRoutingCacheSize()).toBe(0);
    });

    it('should clear routing cache', () => {
      integration.clearRoutingCache();
      expect(integration.getRoutingCacheSize()).toBe(0);
    });

    it('should track active requests', () => {
      const request = {
        requestId: 'test-request',
        errorContext: {} as ErrorContext,
        requiredCapabilities: [],
        nrnToken: 'test-token',
        agentId: 'test-agent',
        priority: 'normal' as const,
        timestamp: Date.now()
      };

      (integration as unknown as KNIRVRouterIntegrationWithPrivates).activeRequests.set('test-request', request);
      expect(integration.getActiveRequestsCount()).toBe(1);
    });

    it('should track routing cache entries', () => {
      const skillNode: SkillNodeURI = {
        nodeId: 'test-node',
        skillId: 'test-skill',
        routerAddress: 'http://localhost:5000',
        networkPath: '/skills/test-skill',
        capabilities: ['test'],
        confidence: 0.9
      };

      (integration as unknown as KNIRVRouterIntegrationWithPrivates).routingCache.set('test-key', [skillNode]);
      expect(integration.getRoutingCacheSize()).toBe(1);
    });
  });

  describe('Event Handling', () => {
    it('should emit connection events', (done) => {
      integration.on('connected', (data) => {
        expect(data).toBeDefined();
        done();
      });

      integration.emit('connected', { timestamp: Date.now() });
    });

    it('should emit disconnection events', (done) => {
      integration.on('disconnected', (data) => {
        expect(data).toBeDefined();
        done();
      });

      integration.emit('disconnected', { timestamp: Date.now() });
    });

    it('should emit skill resolution events', (done) => {
      integration.on('skillResolved', (data) => {
        expect(data.requestId).toBe('test-123');
        done();
      });

      integration.emit('skillResolved', { 
        requestId: 'test-123',
        response: { status: 'SUCCESS' }
      });
    });

    it('should handle skill node discovery events', (done) => {
      const skillNode: SkillNodeURI = {
        nodeId: 'discovered-node',
        skillId: 'discovered-skill',
        routerAddress: 'http://localhost:5000',
        networkPath: '/skills/discovered-skill',
        capabilities: ['discovery'],
        confidence: 0.8
      };

      integration.on('skillNodeDiscovered', (data) => {
        expect(data.skillNode.nodeId).toBe('discovered-node');
        done();
      });

      (integration as unknown as KNIRVRouterIntegrationWithPrivates).handleSkillNodeDiscovery({ skillNode });
    });
  });

  describe('Error Handling', () => {
    it('should handle malformed responses gracefully', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => { throw new Error('Invalid JSON'); }
      } as unknown as Response);

      const errorContext: ErrorContext = {
        errorId: 'test-error',
        errorType: 'skill_invocation_request',
        errorMessage: 'Test error',
        stackTrace: 'test stack',
        userContext: {},
        agentId: 'test-agent',
        timestamp: Date.now(),
        severity: 'medium'
      } as ErrorContext;

      const response = await integration.resolveSkillViaErrorContext(errorContext, ['test']);
      expect(response.status).toBe('FAILURE');
    });

    it('should handle timeout errors', async () => {
      // Use real timers for this test to avoid conflicts
      jest.useRealTimers();

      // Mock KNIRVGRAPH call (first fetch) - should succeed
      mockFetch.mockResolvedValueOnce({
        ok: true,
        status: 200,
        statusText: 'OK',
        headers: new Headers(),
        redirected: false,
        type: 'basic',
        url: '',
        clone: () => ({} as Response),
        body: null,
        bodyUsed: false,
        arrayBuffer: async () => new ArrayBuffer(0),
        blob: async () => new Blob(),
        formData: async () => new FormData(),
        text: async () => '',
        bytes: async () => new Uint8Array(),
        json: jest.fn().mockResolvedValue({ patterns: [], skillNodes: [] } as never)
      } as unknown as Response);

      // Mock KNIRVROUTER call (second fetch) - should timeout quickly
      mockFetch.mockImplementationOnce(() =>
        new Promise((_, reject) =>
          setTimeout(() => reject(new Error('Timeout')), 10)
        )
      );

      const errorContext: ErrorContext = {
        errorId: 'test-error',
        errorType: 'skill_invocation_request',
        errorMessage: 'Test error',
        stackTrace: 'test stack',
        userContext: {},
        agentId: 'test-agent',
        timestamp: Date.now(),
        severity: 'medium'
      } as ErrorContext;

      const response = await integration.resolveSkillViaErrorContext(errorContext, ['test']);
      expect(response.status).toBe('FAILURE');
      expect(response.errorMessage).toContain('KNIRVROUTER submission failed: Timeout');

      // Restore fake timers
      jest.useFakeTimers();
    }, 3000); // Reduce timeout to 3 seconds
  });
});
