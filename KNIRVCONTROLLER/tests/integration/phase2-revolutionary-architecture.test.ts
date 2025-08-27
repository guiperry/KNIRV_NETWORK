/**
 * Phase 2: Revolutionary Architecture Integration Tests
 * Tests for ErrorContext → KNIRVGRAPH → KNIRVROUTER → LoRA adapter flow
 */

import { describe, it, expect, beforeEach, afterEach, jest } from '@jest/globals';
import { CognitiveEngine } from '../../src/sensory-shell/CognitiveEngine';
import { KNIRVRouterIntegration } from '../../src/sensory-shell/KNIRVRouterIntegration';
import { KNIRVChainIntegration } from '../../src/sensory-shell/KNIRVChainIntegration';

// Mock external network dependencies
const mockKNIRVGraphResponse = {
  skillNodeUri: 'knirv://skills/error-handling/v1.0.0',
  confidence: 0.85,
  similarErrors: ['error-001', 'error-002'],
  timestamp: Date.now()
};

const mockKNIRVRouterResponse = {
  status: 'SUCCESS',
  requestId: 'req-123',
  skillNodeUri: 'knirv://skills/error-handling/v1.0.0',
  loraAdapter: new Uint8Array([1, 2, 3, 4]), // Mock LoRA adapter data
  executionTime: 150,
  networkLatency: 50,
  nrnCost: 25
};

// Mock fetch for external network calls
global.fetch = jest.fn();

describe('Phase 2.1: ErrorContext Generation System', () => {
  let cognitiveEngine: CognitiveEngine;

  beforeEach(async () => {
    const config = {
      maxContextSize: 100,
      learningRate: 0.01,
      adaptationThreshold: 0.3,
      skillTimeout: 30000,
      voiceEnabled: false,
      visualEnabled: false,
      loraEnabled: true,
      enhancedLoraEnabled: false,
      hrmEnabled: false,
      adaptiveLearningEnabled: false,
      walletIntegrationEnabled: false,
      chainIntegrationEnabled: true,
      ecosystemCommunicationEnabled: false,
      errorContextEnabled: true // Enable ErrorContext generation
    };

    cognitiveEngine = new CognitiveEngine(config);
  });

  it('should generate comprehensive ErrorContext on failures', async () => {
    const testError = new Error('Test error for context generation');
    testError.stack = 'Error: Test error\n    at test.js:1:1';

    const taskContext = {
      task_description: 'Test task execution',
      input_data_hash: 'hash123',
      agent_state_hash: 'state456'
    };

    // This should generate an ErrorContext
    try {
      throw testError;
    } catch (error) {
      const errorContext = await cognitiveEngine.generateErrorContext(error as Error, taskContext);

      expect(errorContext).toBeDefined();
      expect(errorContext.agent_id).toBeDefined();
      expect(errorContext.error_type).toBe('Error');
      expect(errorContext.error_message).toBe('Test error for context generation');
      expect(errorContext.stack_trace).toContain('Error: Test error');
      expect(errorContext.task_description).toBe('Test task execution');
      expect(errorContext.timestamp).toBeDefined();
      expect(errorContext.agent_state_hash).toBe('state456');
    }
  });

  it('should include environment information in ErrorContext', async () => {
    const testError = new Error('Environment test error');
    const taskContext = { task_description: 'Environment test' };

    try {
      throw testError;
    } catch (error) {
      const errorContext = await cognitiveEngine.generateErrorContext(error as Error, taskContext);

      expect(errorContext.os).toBeDefined();
      expect(errorContext.architecture).toBeDefined();
      expect(errorContext.runtime_environment).toBeDefined();
      expect(errorContext.agent_version).toBeDefined();
      expect(errorContext.base_model_id).toBeDefined();
    }
  });

  it('should serialize ErrorContext to protobuf', async () => {
    const testError = new Error('Serialization test error');
    const taskContext = { task_description: 'Serialization test' };

    try {
      throw testError;
    } catch (error) {
      const errorContext = await cognitiveEngine.generateErrorContext(error as Error, taskContext);
      const serialized = await cognitiveEngine.serializeErrorContext(errorContext);

      expect(serialized).toBeInstanceOf(Uint8Array);
      expect(serialized.length).toBeGreaterThan(0);

      // Should be able to deserialize back
      const deserialized = await cognitiveEngine.deserializeErrorContext(serialized);
      expect(deserialized.error_message).toBe('Serialization test error');
    }
  });
});

describe('Phase 2.2: KNIRVGRAPH Integration', () => {
  let cognitiveEngine: CognitiveEngine;

  beforeEach(async () => {
    const config = {
      maxContextSize: 100,
      learningRate: 0.01,
      adaptationThreshold: 0.3,
      skillTimeout: 30000,
      voiceEnabled: false,
      visualEnabled: false,
      loraEnabled: true,
      enhancedLoraEnabled: false,
      hrmEnabled: false,
      adaptiveLearningEnabled: false,
      walletIntegrationEnabled: false,
      chainIntegrationEnabled: true,
      ecosystemCommunicationEnabled: false,
      errorContextEnabled: true,
      knirvGraphUrl: 'http://localhost:6000'
    };

    cognitiveEngine = new CognitiveEngine(config);

    // Mock KNIRVGRAPH API response
    (global.fetch as jest.Mock).mockResolvedValue({
      ok: true,
      json: () => Promise.resolve(mockKNIRVGraphResponse)
    });
  });

  afterEach(() => {
    jest.clearAllMocks();
  });

  it('should query KNIRVGRAPH for similar error patterns', async () => {
    const errorContext = {
      agent_id: 'test-agent',
      agent_version: '1.0.0',
      base_model_id: 'hrm-v1',
      os: 'linux',
      architecture: 'x64',
      runtime_environment: 'node',
      error_type: 'TypeError',
      error_message: 'Cannot read property of undefined',
      stack_trace: 'TypeError: Cannot read property\n    at test.js:1:1',
      source_code_snippet: 'const value = obj.property;',
      task_description: 'Property access',
      input_data_hash: 'hash123',
      agent_state_hash: 'state456',
      timestamp: new Date(),
      additional_context: {}
    };

    const skillNodeUri = await cognitiveEngine.queryKNIRVGRAPH(errorContext);

    expect(skillNodeUri).toBe('knirv://skills/error-handling/v1.0.0');
    expect(global.fetch).toHaveBeenCalledWith(
      'http://localhost:6000/api/knirvgraph/query-similar-errors',
      expect.objectContaining({
        method: 'POST',
        headers: { 'Content-Type': 'application/json' }
      })
    );
  });

  it('should handle KNIRVGRAPH query failures gracefully', async () => {
    // Mock API failure
    (global.fetch as jest.Mock).mockResolvedValue({
      ok: false,
      status: 500,
      statusText: 'Internal Server Error'
    });

    const errorContext = {
      agent_id: 'test-agent',
      error_type: 'Error',
      error_message: 'Test error',
      // ... other required fields
    } as any;

    const skillNodeUri = await cognitiveEngine.queryKNIRVGRAPH(errorContext);
    expect(skillNodeUri).toBeNull();
  });

  it('should generate vector embeddings for error patterns', async () => {
    const errorContext = {
      error_type: 'TypeError',
      error_message: 'Cannot read property of undefined',
      source_code_snippet: 'const value = obj.property;',
      // ... other fields
    } as any;

    const embedding = await cognitiveEngine.generateErrorEmbedding(errorContext);

    expect(embedding).toBeDefined();
    expect(Array.isArray(embedding)).toBe(true);
    expect(embedding.length).toBeGreaterThan(0);
    expect(typeof embedding[0]).toBe('number');
  });
});

describe('Phase 2.3: KNIRVROUTER Network Integration', () => {
  let knirvRouter: KNIRVRouterIntegration;
  let chainIntegration: KNIRVChainIntegration;

  beforeEach(async () => {
    const routerConfig = {
      knirvRouterUrl: 'http://localhost:5000',
      knirvGraphUrl: 'http://localhost:6000',
      enableP2P: true,
      enableWASM: true,
      maxRetries: 3,
      timeoutMs: 30000
    };

    knirvRouter = new KNIRVRouterIntegration(routerConfig);

    const chainConfig = {
      rpcUrl: 'http://localhost:26657',
      chainId: 'knirv-testnet',
      networkName: 'KNIRV Testnet',
      contractAddresses: {
        nrnToken: 'knirv1...',
        llmRegistry: 'knirv1...',
        skillRegistry: 'knirv1...'
      },
      gasPrice: '0.025uknirv',
      gasLimit: '200000',
      knirvRouterUrl: 'http://localhost:5000',
      useKnirvRouter: true
    };

    chainIntegration = new KNIRVChainIntegration(chainConfig);

    // Mock KNIRVROUTER API response
    (global.fetch as jest.Mock).mockResolvedValue({
      ok: true,
      arrayBuffer: () => Promise.resolve(mockKNIRVRouterResponse.loraAdapter.buffer),
      json: () => Promise.resolve(mockKNIRVRouterResponse)
    });
  });

  afterEach(() => {
    jest.clearAllMocks();
  });

  it('should invoke skills via KNIRVROUTER network', async () => {
    const skillNodeUri = 'knirv://skills/error-handling/v1.0.0';
    const nrnToken = 'nrn-token-123';
    const agentId = 'test-agent';

    const loraAdapter = await knirvRouter.invokeSkillViaRouter(skillNodeUri, nrnToken, agentId);

    expect(loraAdapter).toBeDefined();
    expect(loraAdapter).toBeInstanceOf(Uint8Array);
    expect(global.fetch).toHaveBeenCalledWith(
      'http://localhost:5000/api/knirvrouter/invoke-skill',
      expect.objectContaining({
        method: 'POST',
        headers: { 'Content-Type': 'application/json' }
      })
    );
  });

  it('should resolve skills via ErrorContext', async () => {
    const errorContext = {
      agent_id: 'test-agent',
      error_type: 'TypeError',
      error_message: 'Cannot read property of undefined',
      // ... other required fields
    } as any;

    const requiredCapabilities = ['error-handling', 'property-access'];
    const options = {
      priority: 'high' as const,
      useP2P: true,
      useWASM: true,
      nrnToken: 'nrn-token-123'
    };

    const response = await knirvRouter.resolveSkillViaErrorContext(errorContext, requiredCapabilities, options);

    expect(response.status).toBe('SUCCESS');
    expect(response.skillNodeUri).toBe('knirv://skills/error-handling/v1.0.0');
    expect(response.requestId).toBeDefined();
  });

  it('should handle network failures with retry logic', async () => {
    // Mock network failure followed by success
    (global.fetch as jest.Mock)
      .mockRejectedValueOnce(new Error('Network error'))
      .mockRejectedValueOnce(new Error('Network error'))
      .mockResolvedValueOnce({
        ok: true,
        arrayBuffer: () => Promise.resolve(mockKNIRVRouterResponse.loraAdapter.buffer)
      });

    const skillNodeUri = 'knirv://skills/error-handling/v1.0.0';
    const nrnToken = 'nrn-token-123';
    const agentId = 'test-agent';

    const loraAdapter = await knirvRouter.invokeSkillViaRouter(skillNodeUri, nrnToken, agentId);

    expect(loraAdapter).toBeDefined();
    expect(global.fetch).toHaveBeenCalledTimes(3); // 2 failures + 1 success
  });

  it('should integrate with chain for skill invocation', async () => {
    const skillId = 'error-handling-skill';
    const userAddress = 'knirv1user123';
    const nrnAmount = '25';
    const parameters = {
      agentId: 'test-agent',
      capabilities: ['error-handling'],
      priority: 'high'
    };

    const transactionId = await chainIntegration.invokeSkillOnChain(skillId, userAddress, nrnAmount, parameters);

    expect(transactionId).toBeDefined();
    expect(typeof transactionId).toBe('string');
  });
});

describe('Phase 2: Revolutionary Architecture Integration', () => {
  let cognitiveEngine: CognitiveEngine;

  beforeEach(async () => {
    const config = {
      maxContextSize: 100,
      learningRate: 0.01,
      adaptationThreshold: 0.3,
      skillTimeout: 30000,
      voiceEnabled: false,
      visualEnabled: false,
      loraEnabled: true,
      enhancedLoraEnabled: false,
      hrmEnabled: false,
      adaptiveLearningEnabled: false,
      walletIntegrationEnabled: false,
      chainIntegrationEnabled: true,
      ecosystemCommunicationEnabled: false,
      errorContextEnabled: true,
      knirvGraphUrl: 'http://localhost:6000',
      knirvRouterUrl: 'http://localhost:5000'
    };

    cognitiveEngine = new CognitiveEngine(config);

    // Mock successful responses for the full flow
    (global.fetch as jest.Mock)
      .mockResolvedValueOnce({
        ok: true,
        json: () => Promise.resolve(mockKNIRVGraphResponse)
      })
      .mockResolvedValueOnce({
        ok: true,
        arrayBuffer: () => Promise.resolve(mockKNIRVRouterResponse.loraAdapter.buffer)
      });
  });

  afterEach(() => {
    jest.clearAllMocks();
  });

  it('should execute complete ErrorContext → KNIRVGRAPH → KNIRVROUTER flow', async () => {
    const testError = new Error('Integration test error');
    const taskContext = {
      task_description: 'Integration test task',
      input_data_hash: 'hash123',
      agent_state_hash: 'state456'
    };

    // Simulate error handling with skill acquisition
    try {
      throw testError;
    } catch (error) {
      // Generate ErrorContext
      const errorContext = await cognitiveEngine.generateErrorContext(error as Error, taskContext);
      expect(errorContext).toBeDefined();

      // Query KNIRVGRAPH
      const skillNodeUri = await cognitiveEngine.queryKNIRVGRAPH(errorContext);
      expect(skillNodeUri).toBe('knirv://skills/error-handling/v1.0.0');

      // Invoke skill via KNIRVROUTER
      const loraAdapter = await cognitiveEngine.invokeSkillByUri(skillNodeUri!, 'nrn-token-123');
      expect(loraAdapter).toBeDefined();
      expect(loraAdapter.success).toBe(true);

      // Verify the complete flow was executed
      expect(global.fetch).toHaveBeenCalledTimes(2); // KNIRVGRAPH + KNIRVROUTER
    }
  });

  it('should handle skill acquisition failure gracefully', async () => {
    // Mock KNIRVGRAPH returning no skill
    (global.fetch as jest.Mock).mockResolvedValueOnce({
      ok: true,
      json: () => Promise.resolve({ skillNodeUri: null })
    });

    const testError = new Error('No skill available error');
    const taskContext = { task_description: 'No skill test' };

    try {
      throw testError;
    } catch (error) {
      const errorContext = await cognitiveEngine.generateErrorContext(error as Error, taskContext);
      const skillNodeUri = await cognitiveEngine.queryKNIRVGRAPH(errorContext);

      expect(skillNodeUri).toBeNull();

      // Should submit new ErrorNode to KNIRVGRAPH
      const submitted = await cognitiveEngine.submitErrorNode(errorContext);
      expect(submitted).toBe(true);
    }
  });
});
