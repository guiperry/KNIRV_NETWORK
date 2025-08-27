import { describe, it, expect, beforeAll, afterAll, beforeEach, afterEach } from '@jest/globals';
import { KNIRVRouterIntegration, ErrorContext, SkillNodeURI, KNIRVRouterRequest, KNIRVRouterResponse, LoRAAdapterData } from '../../src/sensory-shell/KNIRVRouterIntegration';
import { KNIRVChainIntegration } from '../../src/sensory-shell/KNIRVChainIntegration';

// Mock fetch for testing
global.fetch = jest.fn();
const mockFetch = global.fetch as jest.MockedFunction<typeof fetch>;

// Mock WebSocket for P2P testing
global.WebSocket = jest.fn().mockImplementation(() => ({
  onopen: null,
  onmessage: null,
  onerror: null,
  onclose: null,
  close: jest.fn(),
  send: jest.fn()
}));

describe('KNIRVROUTER Integration Tests', () => {
  let routerIntegration: KNIRVRouterIntegration;
  let chainIntegration: KNIRVChainIntegration;

  beforeAll(async () => {
    // Setup test environment
    console.log('Setting up KNIRVROUTER Integration test environment...');
  });

  afterAll(async () => {
    // Cleanup test environment
    console.log('Cleaning up KNIRVROUTER Integration test environment...');
  });

  beforeEach(() => {
    // Use fake timers to prevent real timeouts
    jest.useFakeTimers();

    // Reset mocks
    mockFetch.mockClear();

    // Initialize integrations
    routerIntegration = new KNIRVRouterIntegration({
      routerUrl: 'http://localhost:5000',
      graphUrl: 'http://localhost:5001',
      oracleUrl: 'http://localhost:5002',
      timeout: 10000,
      retryAttempts: 2,
      enableP2P: true,
      enableWASM: true
    });

    chainIntegration = new KNIRVChainIntegration({
      useKnirvRouter: true,
      knirvRouterUrl: 'http://localhost:5000',
      knirvGraphUrl: 'http://localhost:5001'
    });
  });

  afterEach(async () => {
    // Disconnect integrations and wait for cleanup
    await routerIntegration.disconnect();

    // Clear all timers and restore real timers
    jest.clearAllTimers();
    jest.useRealTimers();
  });

  describe('KNIRVRouterIntegration', () => {
    it('should initialize with correct configuration', () => {
      expect(routerIntegration).toBeDefined();
      expect(routerIntegration.isRouterConnected()).toBe(false); // Not connected until health check passes
    });

    it('should perform health check on initialization', async () => {
      // Mock successful health check
      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => ({ status: 'healthy' })
      } as Response);

      const healthCheckSpy = jest.spyOn(routerIntegration as any, 'performHealthCheck');
      
      // Trigger initialization
      await (routerIntegration as any).initializeConnection();
      
      expect(healthCheckSpy).toHaveBeenCalled();
      expect(mockFetch).toHaveBeenCalledWith(
        'http://localhost:5000/health',
        expect.objectContaining({
          method: 'GET',
          headers: { 'Content-Type': 'application/json' }
        })
      );
    });

    it('should resolve skill via ErrorContext', async () => {
      // Mock KNIRVGRAPH query response
      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => ({
          patterns: [{ similarity: 0.8 }],
          skillNodes: [{ nodeId: 'test-node-1', confidence: 0.9 }]
        })
      } as Response);

      // Mock KNIRVROUTER resolution response
      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => ({
          requestId: 'test-request-123',
          status: 'SUCCESS',
          skillNodeUri: {
            nodeId: 'test-node-1',
            skillId: 'test-skill',
            routerAddress: 'http://localhost:5000',
            networkPath: '/skills/test-skill',
            capabilities: ['text-processing'],
            confidence: 0.9
          },
          executionTime: 150,
          networkLatency: 50
        })
      } as Response);

      const errorContext: ErrorContext = {
        errorId: 'test-error-001',
        errorType: 'skill_invocation_request',
        errorMessage: 'Test skill invocation',
        stackTrace: 'test stack trace',
        userContext: { test: true },
        agentId: 'test-agent',
        timestamp: Date.now(),
        severity: 'medium'
      };

      const response = await routerIntegration.resolveSkillViaErrorContext(
        errorContext,
        ['text-processing'],
        {
          priority: 'normal',
          useP2P: false,
          useWASM: false,
          nrnToken: 'test-token'
        }
      );

      expect(response.status).toBe('SUCCESS');
      expect(response.requestId).toBe('test-request-123');
      expect(response.skillNodeUri).toBeDefined();
      expect(response.skillNodeUri?.nodeId).toBe('test-node-1');
      expect(response.executionTime).toBe(150);
    });

    it('should handle KNIRVGRAPH query failure gracefully', async () => {
      // Mock KNIRVGRAPH query failure
      mockFetch.mockResolvedValueOnce({
        ok: false,
        statusText: 'Service Unavailable'
      } as Response);

      // Mock KNIRVROUTER resolution response (should still work)
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
            capabilities: ['text-processing'],
            confidence: 0.7
          },
          executionTime: 200,
          networkLatency: 75
        })
      } as Response);

      const errorContext: ErrorContext = {
        errorId: 'test-error-002',
        errorType: 'skill_invocation_request',
        errorMessage: 'Test skill invocation with graph failure',
        stackTrace: 'test stack trace',
        userContext: { test: true },
        agentId: 'test-agent',
        timestamp: Date.now(),
        severity: 'medium'
      };

      const response = await routerIntegration.resolveSkillViaErrorContext(
        errorContext,
        ['text-processing']
      );

      expect(response.status).toBe('SUCCESS');
      expect(response.skillNodeUri?.nodeId).toBe('fallback-node');
    });

    it('should get LoRA adapters from KNIRVROUTER', async () => {
      const mockAdapters: LoRAAdapterData[] = [
        {
          adapterId: 'adapter-001',
          adapterName: 'Test Adapter 1',
          description: 'Test LoRA adapter',
          baseModelCompatibility: 'hrm-v1',
          version: 1,
          rank: 16,
          alpha: 0.5,
          weightsA: new Float32Array([1, 2, 3]),
          weightsB: new Float32Array([4, 5, 6]),
          metadata: { test: 'true' },
          createdAt: new Date(),
          lastUsed: new Date(),
          usageCount: 5,
          networkScore: 0.8,
          routerNodes: ['node-1', 'node-2']
        }
      ];

      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => ({ adapters: mockAdapters })
      } as Response);

      // Mock connection status
      (routerIntegration as any).isConnected = true;

      const adapters = await routerIntegration.getLoRAAdapters();

      expect(adapters).toHaveLength(1);
      expect(adapters[0].adapterId).toBe('adapter-001');
      expect(adapters[0].adapterName).toBe('Test Adapter 1');
    });

    it('should register LoRA adapter with KNIRVROUTER', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => ({ adapter_id: 'new-adapter-123' })
      } as Response);

      // Mock connection status
      (routerIntegration as any).isConnected = true;

      const adapterData = {
        adapterName: 'New Test Adapter',
        description: 'New test LoRA adapter',
        baseModelCompatibility: 'hrm-v1',
        version: 1,
        rank: 32,
        alpha: 0.3,
        weightsA: new Float32Array([1, 2, 3, 4]),
        weightsB: new Float32Array([5, 6, 7, 8]),
        metadata: { author: 'test' }
      };

      const adapterId = await routerIntegration.registerLoRAAdapter(adapterData);

      expect(adapterId).toBe('new-adapter-123');
      expect(mockFetch).toHaveBeenCalledWith(
        'http://localhost:5000/lora-adapters',
        expect.objectContaining({
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(adapterData)
        })
      );
    });

    it('should handle P2P connection events', (done) => {
      const mockWebSocket = new WebSocket('ws://test');
      
      routerIntegration.on('p2pConnected', (data) => {
        expect(data.endpoint).toContain('p2p');
        done();
      });

      // Simulate WebSocket open event
      if (mockWebSocket.onopen) {
        mockWebSocket.onopen({} as Event);
      }
    });

    it('should handle skill node discovery', (done) => {
      const testSkillNode: SkillNodeURI = {
        nodeId: 'discovered-node-1',
        skillId: 'discovered-skill',
        routerAddress: 'http://localhost:5000',
        networkPath: '/skills/discovered-skill',
        capabilities: ['analysis'],
        confidence: 0.85
      };

      routerIntegration.on('skillNodeDiscovered', (data) => {
        expect(data.skillNode.nodeId).toBe('discovered-node-1');
        expect(data.source).toBe('p2p');
        done();
      });

      // Simulate P2P skill node discovery
      (routerIntegration as any).handleSkillNodeDiscovery({
        skillNode: testSkillNode
      });
    });

    it('should track active requests and connections', () => {
      expect(routerIntegration.getActiveRequestsCount()).toBe(0);
      expect(routerIntegration.getP2PConnectionsCount()).toBe(0);
      expect(routerIntegration.getRoutingCacheSize()).toBe(0);
    });

    it('should clear routing cache', () => {
      routerIntegration.clearRoutingCache();
      expect(routerIntegration.getRoutingCacheSize()).toBe(0);
    });
  });

  describe('KNIRVChainIntegration with KNIRVROUTER', () => {
    it('should initialize with KNIRVROUTER integration', async () => {
      expect(chainIntegration).toBeDefined();
      expect((chainIntegration as any).knirvRouter).toBeDefined();
    });

    it('should invoke skill via KNIRVROUTER', async () => {
      // Mock KNIRVROUTER response
      const mockResponse: KNIRVRouterResponse = {
        requestId: 'chain-request-123',
        status: 'SUCCESS',
        skillNodeUri: {
          nodeId: 'chain-node-1',
          skillId: 'chain-skill',
          routerAddress: 'http://localhost:5000',
          networkPath: '/skills/chain-skill',
          capabilities: ['blockchain'],
          confidence: 0.9
        },
        executionTime: 100,
        networkLatency: 25
      };

      // Mock the router integration method
      jest.spyOn((chainIntegration as any).knirvRouter, 'resolveSkillViaErrorContext')
        .mockResolvedValue(mockResponse);

      const requestId = await chainIntegration.invokeSkillOnChain(
        'chain-skill',
        'knirv1test123456789',
        '100',
        {
          agentId: 'test-agent',
          capabilities: ['blockchain'],
          priority: 'normal'
        }
      );

      expect(requestId).toBe('chain-request-123');
    });

    it('should get LoRA adapters via KNIRVROUTER', async () => {
      const mockAdapters: LoRAAdapterData[] = [
        {
          adapterId: 'chain-adapter-001',
          adapterName: 'Chain Test Adapter',
          description: 'Test adapter for chain integration',
          baseModelCompatibility: 'hrm-v1',
          version: 1,
          rank: 16,
          alpha: 0.5,
          weightsA: new Float32Array([1, 2]),
          weightsB: new Float32Array([3, 4]),
          metadata: { chain: 'true' },
          createdAt: new Date(),
          lastUsed: new Date(),
          usageCount: 3,
          networkScore: 0.7,
          routerNodes: ['chain-node-1']
        }
      ];

      // Mock the router integration method
      jest.spyOn((chainIntegration as any).knirvRouter, 'getLoRAAdapters')
        .mockResolvedValue(mockAdapters);

      const adapters = await chainIntegration.getLoRAAdapterSkills();

      expect(adapters).toHaveLength(1);
      expect(adapters[0].adapterId).toBe('chain-adapter-001');
    });

    it('should register LoRA adapter via KNIRVROUTER', async () => {
      // Mock the router integration method
      jest.spyOn((chainIntegration as any).knirvRouter, 'registerLoRAAdapter')
        .mockResolvedValue('chain-new-adapter-456');

      const adapterData = {
        adapterName: 'Chain New Adapter',
        description: 'New adapter for chain integration',
        baseModelCompatibility: 'hrm-v1',
        version: 1,
        rank: 32,
        alpha: 0.4,
        weightsA: new Float32Array([1, 2, 3]),
        weightsB: new Float32Array([4, 5, 6]),
        metadata: { chain: 'true', test: 'true' }
      };

      const adapterId = await chainIntegration.registerLoRAAdapterSkill(adapterData);

      expect(adapterId).toBe('chain-new-adapter-456');
    });

    it('should handle KNIRVROUTER events', (done) => {
      let eventCount = 0;
      const expectedEvents = 2;

      chainIntegration.on('knirvRouterConnected', (data) => {
        expect(data).toBeDefined();
        eventCount++;
        if (eventCount === expectedEvents) done();
      });

      chainIntegration.on('skillResolvedViaKNIRVRouter', (data) => {
        expect(data.requestId).toBeDefined();
        eventCount++;
        if (eventCount === expectedEvents) done();
      });

      // Simulate events from KNIRVROUTER
      (chainIntegration as any).knirvRouter.emit('connected', { timestamp: Date.now() });
      (chainIntegration as any).knirvRouter.emit('skillResolved', { 
        requestId: 'test-123',
        response: { status: 'SUCCESS' }
      });
    });
  });

  describe('Error Handling', () => {
    it('should handle KNIRVROUTER connection failures', async () => {
      // Mock connection failure
      mockFetch.mockRejectedValue(new Error('Connection refused'));

      const errorContext: ErrorContext = {
        errorId: 'error-test-001',
        errorType: 'connection_test',
        errorMessage: 'Test connection failure',
        stackTrace: 'test',
        userContext: {},
        agentId: 'test-agent',
        timestamp: Date.now(),
        severity: 'high'
      };

      const response = await routerIntegration.resolveSkillViaErrorContext(errorContext);

      expect(response.status).toBe('FAILURE');
      expect(response.errorMessage).toContain('Connection refused');
    });

    it('should handle KNIRVROUTER service errors', async () => {
      // Mock service error
      mockFetch.mockResolvedValueOnce({
        ok: false,
        statusText: 'Internal Server Error'
      } as Response);

      const errorContext: ErrorContext = {
        errorId: 'error-test-002',
        errorType: 'service_test',
        errorMessage: 'Test service error',
        stackTrace: 'test',
        userContext: {},
        agentId: 'test-agent',
        timestamp: Date.now(),
        severity: 'high'
      };

      const response = await routerIntegration.resolveSkillViaErrorContext(errorContext);

      expect(response.status).toBe('FAILURE');
      expect(response.errorMessage).toContain('Internal Server Error');
    });

    it('should handle timeout errors', async () => {
      // Mock timeout
      mockFetch.mockImplementation(() => 
        new Promise((_, reject) => 
          setTimeout(() => reject(new Error('Timeout')), 100)
        )
      );

      const errorContext: ErrorContext = {
        errorId: 'error-test-003',
        errorType: 'timeout_test',
        errorMessage: 'Test timeout',
        stackTrace: 'test',
        userContext: {},
        agentId: 'test-agent',
        timestamp: Date.now(),
        severity: 'high'
      };

      const response = await routerIntegration.resolveSkillViaErrorContext(errorContext);

      expect(response.status).toBe('FAILURE');
      expect(response.errorMessage).toContain('Timeout');
    });
  });
});
