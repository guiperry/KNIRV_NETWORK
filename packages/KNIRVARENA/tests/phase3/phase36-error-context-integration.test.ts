/**
 * Phase 3.6 Error Context Integration Tests
 * 
 * Tests for complete End-to-End Skill Invocation Lifecycle integration
 * between ErrorContextManager and CognitiveEngine
 */

import { describe, it, expect, beforeEach, afterEach, jest } from '@jest/globals';
import { ErrorContextManager, AgentConfiguration } from '../../src/core/cortex/ErrorContextManager';
import { ErrorContextHandler } from '../../src/core/protobuf/ErrorContextHandler';
import { CognitiveEngine, CognitiveConfig } from '../../src/sensory-shell/CognitiveEngine';

// Mock fetch for HTTP requests
global.fetch = jest.fn() as jest.MockedFunction<typeof fetch>;

describe('Phase 3.6 - Error Context Integration', () => {
  let errorContextManager: ErrorContextManager;
  let errorContextHandler: ErrorContextHandler;
  let cognitiveEngine: CognitiveEngine;
  let mockFetch: jest.MockedFunction<typeof fetch>;

  const testConfig: AgentConfiguration = {
    agentId: 'test-agent-phase36',
    agentVersion: '1.0.0',
    baseModelId: 'hrm-cognitive-v1',
    knirvgraphEndpoint: 'http://localhost:8081',
    knirvRouterEndpoint: 'http://localhost:8080',
    nrnWalletAddress: 'knirv1test123456789'
  };

  const cognitiveConfig: CognitiveConfig = {
    maxContextSize: 1000,
    learningRate: 0.01,
    adaptationThreshold: 0.7,
    skillTimeout: 30000,
    voiceEnabled: false,
    visualEnabled: false,
    loraEnabled: true,
    enhancedLoraEnabled: true,
    hrmEnabled: true,
    wasmAgentsEnabled: false,
    typeScriptCompilerEnabled: false,
    adaptiveLearningEnabled: true,
    walletIntegrationEnabled: false,
    chainIntegrationEnabled: false,
    ecosystemCommunicationEnabled: false,
    errorContextEnabled: true,
    errorContextConfig: testConfig
  };

  beforeEach(async () => {
    mockFetch = fetch as jest.MockedFunction<typeof fetch>;
    mockFetch.mockClear();

    errorContextHandler = new ErrorContextHandler();
    await errorContextHandler.initialize();

    errorContextManager = new ErrorContextManager(testConfig);
    await errorContextManager.initialize();

    cognitiveEngine = new CognitiveEngine(cognitiveConfig);
  });

  afterEach(async () => {
    if (cognitiveEngine) {
      await cognitiveEngine.dispose();
    }
    jest.clearAllMocks();
  });

  describe('ErrorContext Creation and Serialization', () => {
    it('should create valid ErrorContext from error', async () => {
      const testError = new TypeError('Cannot read property of undefined');
      const taskDescription = 'Processing user input data';
      
      const errorContext = errorContextHandler.createErrorContext(
        testError,
        testConfig.agentId,
        testConfig.agentVersion,
        testConfig.baseModelId,
        taskDescription,
        {
          inputData: 'test input',
          agentState: { confidence: 0.8 }
        }
      );

      expect(errorContext.agentId).toBe(testConfig.agentId);
      expect(errorContext.agentVersion).toBe(testConfig.agentVersion);
      expect(errorContext.baseModelId).toBe(testConfig.baseModelId);
      expect(errorContext.errorType).toBe('TypeError');
      expect(errorContext.errorMessage).toBe('Cannot read property of undefined');
      expect(errorContext.taskDescription).toBe(taskDescription);
      expect(errorContext.timestamp).toBeInstanceOf(Date);
      expect(errorContext.inputDataHash).toBeTruthy();
      expect(errorContext.agentStateHash).toBeTruthy();
    });

    it('should serialize and deserialize ErrorContext correctly', async () => {
      const testError = new Error('Test serialization error');
      const errorContext = errorContextHandler.createErrorContext(
        testError,
        testConfig.agentId,
        testConfig.agentVersion,
        testConfig.baseModelId,
        'Test serialization'
      );

      const serialized = await errorContextHandler.serializeErrorContext(errorContext);
      expect(serialized).toBeInstanceOf(Uint8Array);
      expect(serialized.length).toBeGreaterThan(0);

      const deserialized = await errorContextHandler.deserializeErrorContext(serialized);
      expect(deserialized.agentId).toBe(errorContext.agentId);
      expect(deserialized.errorMessage).toBe(errorContext.errorMessage);
      expect(deserialized.taskDescription).toBe(errorContext.taskDescription);
    });
  });

  describe('KNIRVGRAPH Integration', () => {
    it('should query KNIRVGRAPH for error clusters', async () => {
      // Mock successful KNIRVGRAPH response
      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => ({
          status: 'QUERY_SUCCESS',
          skillNodeResult: {
            skillUri: 'knirv://skill/javascript-type-checker-v1',
            skillNodeId: 'skill_node_001',
            clusterId: 'cluster_js_errors',
            confidence: 0.85
          }
        })
      } as Response);

      const testError = new TypeError('Cannot read property of undefined');
      const result = await errorContextManager.handleError(
        testError,
        'Processing JavaScript code'
      );

      expect(result.skillFound).toBe(true);
      expect(result.skillUri).toBe('knirv://skill/javascript-type-checker-v1');
      expect(result.confidence).toBe(0.85);
      expect(mockFetch).toHaveBeenCalledWith(
        'http://localhost:8081/api/error-clusters/query',
        expect.objectContaining({
          method: 'POST',
          headers: { 'Content-Type': 'application/json' }
        })
      );
    });

    it('should submit new error node when no match found', async () => {
      // Mock KNIRVGRAPH no match response
      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => ({
          status: 'QUERY_NO_MATCH'
        })
      } as Response);

      // Mock successful error node submission
      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => ({
          status: 'SUBMISSION_SUCCESS',
          errorNodeId: 'error_node_new_001',
          clusterId: 'cluster_new_errors'
        })
      } as Response);

      const testError = new Error('New unknown error type');
      const result = await errorContextManager.handleError(
        testError,
        'Processing unknown operation'
      );

      expect(result.skillFound).toBe(false);
      expect(result.errorNodeId).toBe('error_node_new_001');
      expect(result.clusterId).toBe('cluster_new_errors');
      expect(mockFetch).toHaveBeenCalledTimes(2);
    });
  });

  describe('KNIRVROUTER WASM Integration', () => {
    it('should invoke skill through WASM endpoint', async () => {
      // Mock successful WASM skill invocation
      mockFetch.mockResolvedValueOnce({
        ok: true,
        headers: {
          get: (name: string) => {
            const headers: Record<string, string> = {
              'X-KNIRV-Engine': 'wasm',
              'X-KNIRV-Status': 'SUCCESS',
              'X-KNIRV-Invocation-ID': 'inv_test_123'
            };
            return headers[name] || null;
          }
        },
        json: async () => ({
          invocation_id: 'inv_test_123',
          status: 'SUCCESS',
          execution_time: 150,
          memory_used: 1024,
          consensus_reached: true,
          skill_data: '{"skill_name":"javascript-type-checker","version":1}'
        })
      } as Response);

      const result = await errorContextManager.invokeSkill(
        'knirv://skill/javascript-type-checker-v1',
        'test_nrn_token_123',
        { error_type: 'TypeError' }
      );

      expect(result.success).toBe(true);
      expect(result.skillData).toBeTruthy();
      expect(result.invocationId).toBe('inv_test_123');
      expect(mockFetch).toHaveBeenCalledWith(
        'http://localhost:8080/wasm/invoke',
        expect.objectContaining({
          method: 'POST',
          headers: expect.objectContaining({
            'Content-Type': 'application/json',
            'X-KNIRV-Engine': 'wasm',
            'X-KNIRV-Version': '1.0.0'
          })
        })
      );
    });

    it('should handle WASM invocation failures', async () => {
      // Mock failed WASM skill invocation
      mockFetch.mockResolvedValueOnce({
        ok: false,
        status: 500,
        statusText: 'Internal Server Error'
      } as Response);

      const result = await errorContextManager.invokeSkill(
        'knirv://skill/invalid-skill',
        'test_nrn_token_123'
      );

      expect(result.success).toBe(false);
      expect(result.errorMessage).toContain('KNIRVROUTER WASM invocation failed');
    });
  });

  describe('CognitiveEngine Integration', () => {
    it('should handle errors through ErrorContextManager', async () => {
      // Mock KNIRVGRAPH skill discovery
      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => ({
          status: 'QUERY_SUCCESS',
          skillNodeResult: {
            skillUri: 'knirv://skill/error-handler-v1',
            skillNodeId: 'skill_node_002',
            clusterId: 'cluster_general_errors',
            confidence: 0.75
          }
        })
      } as Response);

      const testError = new Error('Test cognitive engine error');
      const errorContextManager = cognitiveEngine.getErrorContextManager();
      expect(errorContextManager).toBeTruthy();

      if (errorContextManager) {
        const result = await errorContextManager.handleError(
          testError,
          'Testing cognitive engine error handling'
        );

        expect(result.skillFound).toBe(true);
        expect(result.skillUri).toBe('knirv://skill/error-handler-v1');
      }
    });

    it('should complete end-to-end skill invocation lifecycle', async () => {
      // Mock KNIRVGRAPH skill discovery
      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => ({
          status: 'QUERY_SUCCESS',
          skillNodeResult: {
            skillUri: 'knirv://skill/complete-lifecycle-v1',
            skillNodeId: 'skill_node_003',
            clusterId: 'cluster_lifecycle_test',
            confidence: 0.90
          }
        })
      } as Response);

      // Mock WASM skill invocation
      mockFetch.mockResolvedValueOnce({
        ok: true,
        headers: {
          get: (name: string) => {
            const headers: Record<string, string> = {
              'X-KNIRV-Engine': 'wasm',
              'X-KNIRV-Status': 'SUCCESS'
            };
            return headers[name] || null;
          }
        },
        json: async () => ({
          invocation_id: 'inv_lifecycle_test',
          status: 'SUCCESS',
          execution_time: 200,
          skill_data: '{"lifecycle":"complete","success":true}'
        })
      } as Response);

      const testError = new Error('End-to-end lifecycle test error');
      const result = await cognitiveEngine.handleErrorAndInvokeSkill(
        testError,
        'Testing complete lifecycle',
        'test_nrn_token_lifecycle'
      );

      expect(result.discoveryResult.skillFound).toBe(true);
      expect(result.discoveryResult.skillUri).toBe('knirv://skill/complete-lifecycle-v1');
      expect(result.invocationResult?.success).toBe(true);
      expect(result.invocationResult?.invocationId).toBe('inv_lifecycle_test');
    });
  });

  describe('Error Handling Edge Cases', () => {
    it('should handle network failures gracefully', async () => {
      // Mock network failure
      mockFetch.mockRejectedValueOnce(new Error('Network error'));

      const testError = new Error('Test network failure');
      
      await expect(
        errorContextManager.handleError(testError, 'Network failure test')
      ).rejects.toThrow('Network error');
    });

    it('should handle invalid responses gracefully', async () => {
      // Mock invalid JSON response
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
        json: async () => { throw new Error('Invalid JSON'); }
      } as unknown as Response);

      const testError = new Error('Test invalid response');
      
      await expect(
        errorContextManager.handleError(testError, 'Invalid response test')
      ).rejects.toThrow('Invalid JSON');
    });

    it('should handle missing ErrorContextManager in CognitiveEngine', async () => {
      const configWithoutErrorContext: CognitiveConfig = {
        ...cognitiveConfig,
        errorContextEnabled: false
      };

      const engineWithoutErrorContext = new CognitiveEngine(configWithoutErrorContext);

      const testError = new Error('Test missing error context manager');
      
      await expect(
        engineWithoutErrorContext.handleErrorAndInvokeSkill(
          testError,
          'Missing error context test'
        )
      ).rejects.toThrow('Error Context Manager not initialized');

      await engineWithoutErrorContext.dispose();
    });
  });

  describe('Type Usage Validation', () => {
    it('should validate imported types are properly used', () => {
      // Test AgentConfiguration type usage
      const mockAgentConfig: AgentConfiguration = {
        agentId: 'test-agent',
        agentVersion: '1.0.0',
        baseModelId: 'CodeT5-base',
        knirvgraphEndpoint: 'http://localhost:8080',
        knirvRouterEndpoint: 'http://localhost:8081',
        nrnWalletAddress: '0x123456789'
      };

      expect(mockAgentConfig.agentId).toBe('test-agent');
      expect(mockAgentConfig.agentVersion).toBe('1.0.0');
      expect(mockAgentConfig.baseModelId).toBe('CodeT5-base');

      // Test ErrorContextHandler type usage
      const mockHandler = new ErrorContextHandler();
      expect(mockHandler).toBeDefined();
      expect(typeof mockHandler.createErrorContext).toBe('function');

      // Test CognitiveConfig type usage
      const mockCognitiveConfig: CognitiveConfig = {
        maxContextSize: 1000,
        learningRate: 0.01,
        adaptationThreshold: 0.8,
        skillTimeout: 30000,
        voiceEnabled: false,
        visualEnabled: false,
        loraEnabled: true,
        enhancedLoraEnabled: false,
        hrmEnabled: true,
        wasmAgentsEnabled: false,
        typeScriptCompilerEnabled: false,
        adaptiveLearningEnabled: true,
        walletIntegrationEnabled: true,
        chainIntegrationEnabled: true,
        ecosystemCommunicationEnabled: true,
        errorContextEnabled: true
      };

      expect(mockCognitiveConfig.maxContextSize).toBe(1000);
      expect(mockCognitiveConfig.learningRate).toBe(0.01);
    });
  });
});
