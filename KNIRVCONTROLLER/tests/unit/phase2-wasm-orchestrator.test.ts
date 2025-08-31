/**
 * Phase 2 WASM Orchestrator and Model Integration Tests
 * 
 * Testing Requirements from MAJOR_REFACTOR_IMPLEMENTATION_PLAN.md:
 * - Dual-WASM orchestration tests
 * - Model switching performance tests
 * - Cross-WASM communication tests
 * - Memory management and optimization tests
 * - Model accuracy and inference quality tests
 */

import { describe, test, expect, beforeEach, afterEach, jest } from '@jest/globals';
import { WASMOrchestrator, ModelConfig, OrchestrationConfig } from '../../src/sensory-shell/WASMOrchestrator';


describe('Phase 2.4: WASM Orchestrator and Model Integration', () => {
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

    // WebAssembly and fetch are already mocked globally in jest.setup.js
    // No need to override them here as the global mocks should work
  });

  afterEach(async () => {
    await orchestrator.dispose();
    jest.clearAllMocks();
  });

  describe('Dual-WASM Orchestration', () => {
    test('should initialize both cognitive shell and model WASM', async () => {
      // Simulate the initialization events that would normally be emitted
      const success = await orchestrator.initialize();

      // Manually trigger the events that would be emitted during real initialization
      // Since we're mocking AgentCoreInterface, we need to simulate its events
      // Simulate the initialization completion that would happen in real scenario
      // This tests that the orchestrator can handle the ready state properly
      await new Promise(resolve => setTimeout(resolve, 10));

      expect(success).toBe(true);
      expect(orchestrator.isReady()).toBe(true);

      const moduleInfo = orchestrator.getModuleInfo();
      expect(moduleInfo).toHaveLength(2);
      expect(moduleInfo.find(m => m.type === 'cognitive-shell')).toBeDefined();
      expect(moduleInfo.find(m => m.type === 'model')).toBeDefined();
    });

    test('should handle cognitive shell loading failure', async () => {
      (WebAssembly.compile as jest.Mock).mockRejectedValueOnce(new Error('Cognitive shell compile failed') as never);
      
      const success = await orchestrator.initialize();
      expect(success).toBe(false);
    });

    test('should handle model WASM loading failure with fallback', async () => {
      const consoleSpy = jest.spyOn(console, 'warn').mockImplementation(() => {});
      (global.fetch as jest.Mock).mockRejectedValueOnce(new Error('Model fetch failed') as never);

      const success = await orchestrator.initialize();
      expect(success).toBe(true); // Should succeed with fallback
      expect(consoleSpy).toHaveBeenCalledWith(
        expect.stringContaining('Failed to load HRM cognitive WASM, using fallback:'),
        expect.any(Error)
      );

      consoleSpy.mockRestore();
    });

    test('should manage WASM module lifecycle', async () => {
      await orchestrator.initialize();
      expect(orchestrator.isReady()).toBe(true);

      await orchestrator.dispose();
      expect(orchestrator.isReady()).toBe(false);
    });
  });

  describe('Model Switching Performance', () => {
    beforeEach(async () => {
      await orchestrator.initialize();
    });

    test('should switch between different models efficiently', async () => {
      const phi3Config: ModelConfig = {
        modelType: 'phi-3-mini',
        maxTokens: 2048,
        temperature: 0.8,
        topP: 0.95,
        contextLength: 4096
      };

      const startTime = Date.now();
      const success = await orchestrator.switchModel(phi3Config);
      const endTime = Date.now();

      expect(success).toBe(true);
      expect(endTime - startTime).toBeLessThan(10000); // Should complete within 10 seconds
    });

    test('should maintain performance during model switching', async () => {
      const models: ModelConfig[] = [
        { modelType: 'hrm_cognitive', maxTokens: 1024, temperature: 0.7, topP: 0.9, contextLength: 2048 },
        { modelType: 'knirv_cortex', maxTokens: 1024, temperature: 0.7, topP: 0.9, contextLength: 2048 },
        { modelType: 'tinyllama', maxTokens: 512, temperature: 0.6, topP: 0.8, contextLength: 1024 }
      ];

      for (const model of models) {
        const startTime = Date.now();
        const success = await orchestrator.switchModel(model);
        const endTime = Date.now();

        expect(success).toBe(true);
        expect(endTime - startTime).toBeLessThan(5000);
      }
    });

    test('should handle invalid model types', async () => {
      const invalidConfig: ModelConfig = {
        modelType: 'invalid-model',
        maxTokens: 1024,
        temperature: 0.7,
        topP: 0.9,
        contextLength: 2048
      };

      const success = await orchestrator.switchModel(invalidConfig);
      expect(success).toBe(false);
    });
  });

  describe('Cross-WASM Communication', () => {
    beforeEach(async () => {
      await orchestrator.initialize();
    });

    test('should enable communication between cognitive shell and model', async () => {
      const input = {
        type: 'text' as const,
        data: 'Test cross-WASM communication',
        timestamp: Date.now(),
        sessionId: 'cross-wasm-test'
      };

      // The global mock in jest.setup.js already sets up agentCoreExecute
      // We can use the existing mock behavior for this test
      // No need to override it as the default mock should work

      const response = await orchestrator.processSensoryInput(input);

      expect(response).toBeDefined();
      expect(response.success).toBe(true);
      // The agentCoreExecute function should have been called via the global mock
      // We can verify this by checking if WebAssembly.instantiate was called
      expect(WebAssembly.instantiate).toHaveBeenCalled();
    });

    test('should handle cross-WASM communication failures', async () => {
      const input = {
        type: 'text' as const,
        data: 'Test communication failure',
        timestamp: Date.now(),
        sessionId: 'comm-failure-test'
      };

      // The global mock in jest.setup.js already sets up agentCoreExecute
      // We can use the existing mock behavior for this test
      // No need to override it as the default mock should work

      // The global mock should handle the rejection
      await expect(orchestrator.processSensoryInput(input)).rejects.toThrow('Model inference failed');
    });

    test('should route communication correctly', async () => {
      const input = {
        type: 'text' as const,
        data: 'Route test',
        timestamp: Date.now(),
        sessionId: 'route-test'
      };

      const response = await orchestrator.processSensoryInput(input);

      expect(response).toBeDefined();
      expect(response.source).toBe('agent-core');
    });
  });

  describe('Memory Management and Optimization', () => {
    beforeEach(async () => {
      await orchestrator.initialize();
    });

    test('should manage memory efficiently with multiple inferences', async () => {
      const inputs = Array.from({ length: 20 }, (_, i) => ({
        type: 'text' as const,
        data: `Memory test input ${i}`,
        timestamp: Date.now(),
        sessionId: `memory-test-${i}`
      }));

      const responses = await Promise.all(
        inputs.map(input => orchestrator.processSensoryInput(input))
      );

      expect(responses).toHaveLength(20);
      expect(responses.every(r => r.success)).toBe(true);

      // Memory should remain stable
      const moduleInfo = orchestrator.getModuleInfo();
      expect(moduleInfo.every(m => m.loaded && m.initialized)).toBe(true);
    });

    test('should optimize WASM memory allocation', async () => {
      const moduleInfo = orchestrator.getModuleInfo();
      
      // Check that memory is allocated appropriately
      expect(WebAssembly.Memory).toHaveBeenCalled();
      
      // Verify modules are loaded efficiently
      expect(moduleInfo.every(m => m.size > 0)).toBe(true);
    });

    test('should handle memory pressure gracefully', async () => {
      // Simulate high memory usage
      const largeInputs = Array.from({ length: 100 }, (_, i) => ({
        type: 'text' as const,
        data: 'x'.repeat(10000), // Large input
        timestamp: Date.now(),
        sessionId: `large-input-${i}`
      }));

      // Process in batches to avoid overwhelming
      for (let i = 0; i < largeInputs.length; i += 5) {
        const batch = largeInputs.slice(i, i + 5);
        const responses = await Promise.all(
          batch.map(input => orchestrator.processSensoryInput(input))
        );
        expect(responses.every(r => r.success)).toBe(true);
      }
    });
  });

  describe('Model Accuracy and Inference Quality', () => {
    beforeEach(async () => {
      await orchestrator.initialize();
    });

    test('should maintain inference quality across models', async () => {
      const testInput = {
        type: 'text' as const,
        data: 'What is the capital of France?',
        timestamp: Date.now(),
        sessionId: 'quality-test'
      };

      const models: ModelConfig[] = [
        { modelType: 'hrm_cognitive', maxTokens: 1024, temperature: 0.7, topP: 0.9, contextLength: 2048 },
        { modelType: 'knirv_cortex', maxTokens: 1024, temperature: 0.7, topP: 0.9, contextLength: 2048 }
      ];

      for (const model of models) {
        await orchestrator.switchModel(model);
        const response = await orchestrator.processSensoryInput(testInput);

        expect(response.success).toBe(true);
        expect(response.confidence).toBeGreaterThan(0.5);
        expect(response.processingTime).toBeGreaterThan(0);
      }
    });

    test('should provide consistent inference results', async () => {
      const input = {
        type: 'text' as const,
        data: 'Consistent test input',
        timestamp: Date.now(),
        sessionId: 'consistency-test'
      };

      const responses = await Promise.all([
        orchestrator.processSensoryInput(input),
        orchestrator.processSensoryInput(input),
        orchestrator.processSensoryInput(input)
      ]);

      expect(responses).toHaveLength(3);
      expect(responses.every(r => r.success)).toBe(true);
      
      // Results should be consistent (allowing for some variance)
      const processingTimes = responses.map(r => r.processingTime);
      const avgTime = processingTimes.reduce((a, b) => a + b, 0) / processingTimes.length;
      expect(processingTimes.every(t => Math.abs(t - avgTime) < avgTime * 0.5)).toBe(true);
    });

    test('should handle different input types correctly', async () => {
      const inputs = [
        { type: 'text' as const, data: 'Text input test', timestamp: Date.now(), sessionId: 'text-test' },
        { type: 'voice' as const, data: { audio: 'mock-audio-data' }, timestamp: Date.now(), sessionId: 'voice-test' },
        { type: 'visual' as const, data: { image: 'mock-image-data' }, timestamp: Date.now(), sessionId: 'visual-test' }
      ];

      for (const input of inputs) {
        const response = await orchestrator.processSensoryInput(input);
        expect(response.success).toBe(true);
        expect(response.metadata?.inputType).toBe(input.type);
      }
    });
  });

  describe('Performance Benchmarks', () => {
    beforeEach(async () => {
      await orchestrator.initialize();
    });

    test('should meet performance benchmarks for inference speed', async () => {
      const input = {
        type: 'text' as const,
        data: 'Performance benchmark test',
        timestamp: Date.now(),
        sessionId: 'benchmark-test'
      };

      const startTime = Date.now();
      const response = await orchestrator.processSensoryInput(input);
      const endTime = Date.now();

      expect(response.success).toBe(true);
      expect(endTime - startTime).toBeLessThan(1000); // Should complete within 1 second
      expect(response.processingTime).toBeLessThan(500); // Internal processing under 500ms
    });

    test('should handle concurrent inferences efficiently', async () => {
      const concurrentInputs = Array.from({ length: 10 }, (_, i) => ({
        type: 'text' as const,
        data: `Concurrent test ${i}`,
        timestamp: Date.now(),
        sessionId: `concurrent-${i}`
      }));

      const startTime = Date.now();
      const responses = await Promise.all(
        concurrentInputs.map(input => orchestrator.processSensoryInput(input))
      );
      const endTime = Date.now();

      expect(responses).toHaveLength(10);
      expect(responses.every(r => r.success)).toBe(true);
      expect(endTime - startTime).toBeLessThan(5000); // All should complete within 5 seconds
    });
  });
});
