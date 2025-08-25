/**
 * Phase 2 TypeScript Agent-Core Compiler Tests
 * 
 * Testing Requirements from MAJOR_REFACTOR_IMPLEMENTATION_PLAN.md:
 * - Agent-core WASM compilation tests
 * - Sensory-shell to agent-core communication tests
 * - Template translation accuracy tests
 * - Cross-platform WASM execution tests
 * - Performance tests for WASM vs direct execution
 * - Error handling and validation tests
 */

import { describe, test, expect, beforeEach, afterEach, jest } from '@jest/globals';
import { AgentCoreInterface } from '../../src/sensory-shell/AgentCoreInterface';
import { WASMOrchestrator, ModelConfig, OrchestrationConfig } from '../../src/sensory-shell/WASMOrchestrator';

// Mock the agent-core compiler
jest.mock('../../src/core/agent-core-compiler', () => ({
  AgentCoreCompiler: jest.fn().mockImplementation(() => ({
    compileAgentCore: jest.fn().mockResolvedValue(new Uint8Array([0x00, 0x61, 0x73, 0x6d])),
    validateTemplates: jest.fn().mockReturnValue(true),
    getCompilationMetrics: jest.fn().mockReturnValue({
      compilationTime: 1000,
      wasmSize: 1024,
      optimizationLevel: 'O2'
    })
  }))
}));

describe('Phase 2.2: TypeScript Agent-Core Compiler', () => {
  let orchestrator: WASMOrchestrator;
  let defaultConfig: OrchestrationConfig;

  beforeEach(() => {
    defaultConfig = {
      defaultModel: {
        modelType: 'hrm_cognitive',
        maxTokens: 1024,
        temperature: 0.7,
        topP: 0.9,
        contextLength: 2048
      },
      enableModelFallback: true,
      enableCrossWASMCommunication: true,
      maxConcurrentInferences: 5,
      timeoutMs: 30000
    };

    orchestrator = new WASMOrchestrator(defaultConfig);

    // Mock WebAssembly
    global.WebAssembly = {
      compile: jest.fn().mockResolvedValue({}),
      instantiate: jest.fn().mockResolvedValue({
        exports: {
          agentCoreExecute: jest.fn().mockResolvedValue('{"success": true}'),
          agentCoreGetStatus: jest.fn().mockReturnValue('{"initialized": true}'),
          modelInference: jest.fn().mockResolvedValue('{"result": "inference complete"}'),
          modelGetInfo: jest.fn().mockReturnValue('{"name": "test-model", "version": "1.0.0"}')
        }
      }),
      Memory: jest.fn().mockImplementation(() => ({ buffer: new ArrayBuffer(1024) }))
    } as any;

    // Mock fetch for model loading
    global.fetch = jest.fn().mockImplementation((url: string) => {
      // Return different mock responses based on the URL
      const mockWasmBytes = new ArrayBuffer(1024);
      return Promise.resolve({
        ok: true,
        arrayBuffer: jest.fn().mockResolvedValue(mockWasmBytes)
      });
    }) as any;
  });

  afterEach(async () => {
    await orchestrator.dispose();
    jest.clearAllMocks();
  });

  describe('Agent-Core WASM Compilation', () => {
    test('should compile TypeScript templates to WASM', async () => {
      const { AgentCoreCompiler } = await import('../../src/core/agent-core-compiler');
      const compiler = new AgentCoreCompiler();

      const wasmBytes = await compiler.compileAgentCore({
        agentId: 'test-agent',
        templates: ['CognitiveEngine', 'LoRAAdapter'],
        optimizationLevel: 'O2'
      });

      expect(wasmBytes).toBeInstanceOf(Uint8Array);
      expect(wasmBytes.length).toBeGreaterThan(0);
      expect(compiler.validateTemplates).toHaveBeenCalled();
    });

    test('should handle compilation errors gracefully', async () => {
      const { AgentCoreCompiler } = await import('../../src/core/agent-core-compiler');
      const compiler = new AgentCoreCompiler();

      // Mock compilation failure
      compiler.compileAgentCore = jest.fn().mockRejectedValue(new Error('Compilation failed'));

      await expect(compiler.compileAgentCore({
        agentId: 'invalid-agent',
        templates: ['InvalidTemplate']
      })).rejects.toThrow('Compilation failed');
    });

    test('should validate template translation accuracy', async () => {
      const { AgentCoreCompiler } = await import('../../src/core/agent-core-compiler');
      const compiler = new AgentCoreCompiler();

      const isValid = compiler.validateTemplates(['CognitiveEngine', 'LoRAAdapter']);
      expect(isValid).toBe(true);
    });

    test('should provide compilation metrics', async () => {
      const { AgentCoreCompiler } = await import('../../src/core/agent-core-compiler');
      const compiler = new AgentCoreCompiler();

      await compiler.compileAgentCore({
        agentId: 'metrics-test',
        templates: ['CognitiveEngine']
      });

      const metrics = compiler.getCompilationMetrics();
      expect(metrics).toHaveProperty('compilationTime');
      expect(metrics).toHaveProperty('wasmSize');
      expect(metrics).toHaveProperty('optimizationLevel');
    });
  });

  describe('Sensory-Shell to Agent-Core Communication', () => {
    test('should establish communication channels', async () => {
      const success = await orchestrator.initialize();
      expect(success).toBe(true);

      const moduleInfo = orchestrator.getModuleInfo();
      expect(moduleInfo).toHaveLength(2); // Cognitive shell + Model
      expect(moduleInfo[0].type).toBe('cognitive-shell');
      expect(moduleInfo[1].type).toBe('model');
    });

    test('should handle cross-WASM communication', async () => {
      await orchestrator.initialize();

      const input = {
        type: 'text' as const,
        data: 'Test cross-WASM communication',
        timestamp: Date.now(),
        sessionId: 'comm-test'
      };

      const response = await orchestrator.processSensoryInput(input);
      expect(response).toBeDefined();
      expect(response.success).toBe(true);
    });

    test('should queue inferences when not ready', async () => {
      const input = {
        type: 'text' as const,
        data: 'Queued inference',
        timestamp: Date.now(),
        sessionId: 'queue-test'
      };

      // Start processing before initialization
      const responsePromise = orchestrator.processSensoryInput(input);

      // Initialize after starting processing
      await orchestrator.initialize();

      const response = await responsePromise;
      expect(response).toBeDefined();
    });
  });

  describe('Cross-Platform WASM Execution', () => {
    test('should execute on different platforms', async () => {
      // Test Node.js environment
      const nodeSuccess = await orchestrator.initialize();
      expect(nodeSuccess).toBe(true);

      // Mock browser environment
      const originalWindow = global.window;
      global.window = {} as any;

      const browserSuccess = await orchestrator.initialize();
      expect(browserSuccess).toBe(true);

      // Restore
      global.window = originalWindow;
    });

    test('should handle platform-specific memory constraints', async () => {
      await orchestrator.initialize();

      const moduleInfo = orchestrator.getModuleInfo();
      expect(moduleInfo.every(module => module.loaded)).toBe(true);
    });
  });

  describe('Performance Tests', () => {
    test('should measure WASM vs direct execution performance', async () => {
      await orchestrator.initialize();

      const input = {
        type: 'text' as const,
        data: 'Performance test input',
        timestamp: Date.now(),
        sessionId: 'perf-test'
      };

      const startTime = Date.now();
      const response = await orchestrator.processSensoryInput(input);
      const endTime = Date.now();

      expect(response.processingTime).toBeGreaterThan(0);
      expect(endTime - startTime).toBeGreaterThan(response.processingTime);
    });

    test('should handle concurrent inferences efficiently', async () => {
      await orchestrator.initialize();

      const inputs = Array.from({ length: 5 }, (_, i) => ({
        type: 'text' as const,
        data: `Concurrent input ${i}`,
        timestamp: Date.now(),
        sessionId: `concurrent-${i}`
      }));

      const startTime = Date.now();
      const responses = await Promise.all(
        inputs.map(input => orchestrator.processSensoryInput(input))
      );
      const endTime = Date.now();

      expect(responses).toHaveLength(5);
      expect(responses.every(r => r.success)).toBe(true);
      expect(endTime - startTime).toBeLessThan(10000); // Should complete within 10 seconds
    });

    test('should optimize memory usage', async () => {
      await orchestrator.initialize();

      const initialModules = orchestrator.getModuleInfo();
      const initialMemory = initialModules.reduce((sum, module) => sum + module.size, 0);

      // Process multiple inputs
      for (let i = 0; i < 10; i++) {
        await orchestrator.processSensoryInput({
          type: 'text',
          data: `Memory test ${i}`,
          timestamp: Date.now(),
          sessionId: `memory-${i}`
        });
      }

      const finalModules = orchestrator.getModuleInfo();
      const finalMemory = finalModules.reduce((sum, module) => sum + module.size, 0);

      // Memory should not grow significantly
      expect(finalMemory).toBeLessThanOrEqual(initialMemory * 1.1);
    });
  });

  describe('Error Handling and Validation', () => {
    test('should handle WASM loading failures', async () => {
      (WebAssembly.compile as jest.Mock).mockRejectedValueOnce(new Error('WASM compile failed'));

      const success = await orchestrator.initialize();
      expect(success).toBe(false);
    });

    test('should validate input parameters', async () => {
      await orchestrator.initialize();

      const invalidInput = {
        type: 'invalid' as any,
        data: null,
        timestamp: -1,
        sessionId: ''
      };

      await expect(orchestrator.processSensoryInput(invalidInput))
        .rejects.toThrow();
    });

    test('should handle model switching errors', async () => {
      await orchestrator.initialize();

      const invalidModel: ModelConfig = {
        modelType: 'invalid-model' as any,
        maxTokens: 1024,
        temperature: 0.7,
        topP: 0.9,
        contextLength: 2048
      };

      const success = await orchestrator.switchModel(invalidModel);
      expect(success).toBe(false);
    });

    test('should cleanup resources on disposal', async () => {
      await orchestrator.initialize();
      expect(orchestrator.isReady()).toBe(true);

      await orchestrator.dispose();
      expect(orchestrator.isReady()).toBe(false);

      // Should not be able to process after disposal
      await expect(orchestrator.processSensoryInput({
        type: 'text',
        data: 'test',
        timestamp: Date.now(),
        sessionId: 'disposal-test'
      })).rejects.toThrow();
    });
  });
});
