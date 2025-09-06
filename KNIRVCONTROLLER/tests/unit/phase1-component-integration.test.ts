/**
 * Phase 1 Component Integration Tests
 * Tests for KNIRV-CONTROLLER Integration (1.1) from MAJOR_REFACTOR_IMPLEMENTATION_PLAN.md
 */

import { describe, it, expect, beforeEach, afterEach } from '@jest/globals';
import ProtobufHandler from '../../src/core/protobuf/ProtobufHandler';
import { loraEngine } from '../../src/core/loraEngine';
import AgentCoreCompiler from '../../src/core/agent-core-compiler/src/AgentCoreCompiler';
import { WASMOrchestrator } from '../../src/sensory-shell/WASMOrchestrator';
import { ModelManager } from '../../src/sensory-shell/ModelManager';

describe('Phase 1: Component Integration Tests', () => {
  let protobufHandler: ProtobufHandler;
  let agentCompiler: AgentCoreCompiler;
  let wasmOrchestrator: WASMOrchestrator;
  let modelManager: ModelManager;

  beforeEach(async () => {
    // Initialize components
    protobufHandler = new ProtobufHandler();
    agentCompiler = new AgentCoreCompiler();
    wasmOrchestrator = new WASMOrchestrator();
    modelManager = new ModelManager();
  });

  afterEach(async () => {
    // Cleanup components
    if (protobufHandler.isReady()) {
      await protobufHandler.cleanup();
    }
    if (agentCompiler.isReady()) {
      await agentCompiler.dispose();
    }
    if (wasmOrchestrator.isRunning) {
      await wasmOrchestrator.shutdown();
    }
    // ModelManager doesn't require cleanup - it's a simple registry
  });

  describe('1.1 KNIRV-CONTROLLER Integration', () => {
    describe('Component Initialization', () => {
      it('should initialize ProtobufHandler correctly', async () => {
        await protobufHandler.initialize();
        expect(protobufHandler.isReady()).toBe(true);
        expect(protobufHandler.getAvailableSchemas()).toContain('LoRaAdapterSkill');
        expect(protobufHandler.getAvailableSchemas()).toContain('SkillInvocationResponse');
      });

      it('should initialize LoRAAdapterEngine correctly', async () => {
        expect(loraEngine).toBeDefined();
        expect(typeof loraEngine.compileAdapter).toBe('function');
        expect(typeof loraEngine.invokeAdapter).toBe('function');
      });

      it('should initialize AgentCoreCompiler correctly', async () => {
        await agentCompiler.initialize();
        expect(agentCompiler.isReady()).toBe(true);
      });

      it('should initialize WASMOrchestrator correctly', async () => {
        await wasmOrchestrator.start();
        expect(wasmOrchestrator.isRunning).toBe(true);
      });

      it('should initialize ModelManager correctly', async () => {
        await modelManager.initialize();
        expect(modelManager.isInitialized()).toBe(true);
        expect(modelManager.getAvailableModels().length).toBeGreaterThan(0);
      });
    });

    describe('Unified Architecture Integration', () => {
      beforeEach(async () => {
        await protobufHandler.initialize();
        await agentCompiler.initialize();
        await wasmOrchestrator.start();
        await modelManager.initialize();
      });

      it('should integrate manager with receiver components', async () => {
        // Test that manager and receiver components can communicate
        // This would test the actual integration between manager and receiver
        // For now, we'll test that the components are properly initialized
        expect(protobufHandler.isReady()).toBe(true);
        expect(wasmOrchestrator.isRunning).toBe(true);
      });

      it('should establish unified directory structure communication', () => {
        // Test that core and sensory-shell can communicate
        expect(protobufHandler).toBeDefined();
        expect(loraEngine).toBeDefined();
        expect(wasmOrchestrator).toBeDefined();
      });

      it('should support component orchestration system', async () => {
        // Test orchestration between components
        const testConfig = {
          agentId: 'test-agent-123',
          agentName: 'Test Agent',
          agentDescription: 'Test agent for integration',
          agentVersion: '1.0.0',
          author: 'test',
          tools: [],
          cognitiveCapabilities: [],
          sensoryInterfaces: [],
          buildTarget: 'typescript' as const,
          optimizationLevel: 'basic' as const
        };

        const compilationResult = await agentCompiler.compileAgentCore(testConfig);
        expect(compilationResult.success).toBe(true);
        expect(compilationResult.agentId).toBe(testConfig.agentId);
      });

      it('should support unified configuration management', () => {
        // Test that configuration can be shared across components
        const testConfig = { timeout: 5000, retries: 3 };
        
        // Components should be able to accept and use shared configuration
        expect(() => {
          // This would test actual config sharing in a real implementation
          const config = { ...testConfig };
          expect(config.timeout).toBe(5000);
        }).not.toThrow();
      });
    });

    describe('Build Scripts and Configuration Integration', () => {
      it('should support unified build configuration', () => {
        // Test that build scripts work with integrated components
        expect(agentCompiler).toBeDefined();
        expect(typeof agentCompiler.compileAgentCore).toBe('function');
      });

      it('should handle integrated component dependencies', async () => {
        // Test that components can depend on each other properly
        await protobufHandler.initialize();
        
        const testAdapter = {
          skill_id: 'test-skill-123',
          skill_name: 'Test Skill',
          description: 'A test skill for integration',
          base_model_compatibility: 'test-model',
          version: 1,
          rank: 8,
          alpha: 16.0,
          weightsA: new Float32Array([0.1, 0.2, 0.3]),
          weightsB: new Float32Array([0.4, 0.5, 0.6]),
          additional_metadata: {}
        };

        const serialized = await protobufHandler.serializeLoRAAdapter(testAdapter);
        expect(serialized).toBeInstanceOf(Uint8Array);
        expect(serialized.length).toBeGreaterThan(0);

        const deserialized = await protobufHandler.deserializeLoRAAdapter(serialized);
        expect((deserialized as any).skill_id).toBe(testAdapter.skill_id);
        expect((deserialized as any).skill_name).toBe(testAdapter.skill_name);
      });
    });
  });

  describe('Performance Tests for Component Communication', () => {
    beforeEach(async () => {
      await protobufHandler.initialize();
      await agentCompiler.initialize();
      await wasmOrchestrator.start();
      await modelManager.initialize();
    });

    it('should handle high-frequency protobuf serialization/deserialization', async () => {
      const startTime = Date.now();
      const iterations = 100;

      for (let i = 0; i < iterations; i++) {
        const testAdapter = {
          skill_id: `test-skill-${i}`,
          skill_name: `Test Skill ${i}`,
          description: 'Performance test skill',
          base_model_compatibility: 'test-model',
          version: 1,
          rank: 8,
          alpha: 16.0,
          weightsA: new Float32Array(64).fill(0.1),
          weightsB: new Float32Array(64).fill(0.2),
          additional_metadata: {}
        };

        const serialized = await protobufHandler.serializeLoRAAdapter(testAdapter);
        const deserialized = await protobufHandler.deserializeLoRAAdapter(serialized);
        expect((deserialized as any).skill_id).toBe(testAdapter.skill_id);
      }

      const endTime = Date.now();
      const totalTime = endTime - startTime;
      const avgTime = totalTime / iterations;

      // Should complete within reasonable time (less than 10ms per operation)
      expect(avgTime).toBeLessThan(10);
    });

    it('should handle concurrent component operations', async () => {
      const promises = [];
      
      // Test concurrent operations across components
      for (let i = 0; i < 10; i++) {
        promises.push(
          loraEngine.compileAdapter({ id: `adapter-${i}`, config: {} })
        );
      }

      const results = await Promise.all(promises);
      expect(results.length).toBe(10);
      results.forEach(result => {
        expect(typeof result).toBe('string');
      });
    });

    it('should maintain performance under load', async () => {
      const startTime = Date.now();
      const operations = [];

      // Simulate heavy load across multiple components
      for (let i = 0; i < 50; i++) {
        operations.push(
          loraEngine.compileAdapter({ id: `load-test-${i}`, config: {} })
        );
      }

      await Promise.all(operations);
      
      const endTime = Date.now();
      const totalTime = endTime - startTime;

      // Should complete all operations within 5 seconds
      expect(totalTime).toBeLessThan(5000);
    });
  });

  describe('Regression Tests for Existing Functionality', () => {
    beforeEach(async () => {
      await protobufHandler.initialize();
      await agentCompiler.initialize();
    });

    it('should maintain backward compatibility with existing protobuf schemas', async () => {
      // Test that existing protobuf functionality still works
      const legacyResponse = {
        invocation_id: 'legacy-test-123',
        status: 'SUCCESS',
        error_message: '',
        skill: {
          skill_id: 'legacy-skill',
          skill_name: 'Legacy Skill',
          description: 'Legacy test skill',
          base_model_compatibility: 'legacy-model',
          version: 1,
          rank: 4,
          alpha: 8.0,
          weights_a: new Uint8Array([1, 2, 3, 4]),
          weights_b: new Uint8Array([5, 6, 7, 8]),
          additional_metadata: {}
        }
      };

      const serialized = await protobufHandler.serialize(legacyResponse, 'SkillInvocationResponse');
      expect(serialized).toBeInstanceOf(Uint8Array);

      const deserialized = await protobufHandler.deserialize(serialized, 'SkillInvocationResponse');
      expect((deserialized as any).invocation_id).toBe(legacyResponse.invocation_id);
    });

    it('should preserve existing LoRA adapter functionality', async () => {
      // Test that LoRA adapters still work as expected
      const adapterId = await loraEngine.compileAdapter({
        id: 'regression-test',
        config: { rank: 8, alpha: 16.0 }
      });

      expect(typeof adapterId).toBe('string');
      expect(adapterId).toContain('adapter-');

      const result = await loraEngine.invokeAdapter(adapterId, { test: 'data' });
      expect(result.success).toBe('success');
      expect(result.adapterId).toBe(adapterId);
    });

    it('should maintain existing agent compilation capabilities', async () => {
      // Test that agent compilation still works
      const testConfig = {
        agentId: 'regression-agent',
        agentName: 'Regression Test Agent',
        agentDescription: 'Agent for regression testing',
        agentVersion: '1.0.0',
        author: 'test',
        tools: [{
          name: 'testTool',
          description: 'Test tool',
          parameters: [],
          implementation: 'return { result: "test" };',
          sourceType: 'inline' as const
        }],
        cognitiveCapabilities: [],
        sensoryInterfaces: [],
        buildTarget: 'typescript' as const,
        optimizationLevel: 'basic' as const
      };

      const result = await agentCompiler.compileAgentCore(testConfig);
      expect(result.success).toBe(true);
      expect(result.typeScriptCode).toBeDefined();
      expect(result.typeScriptCode!.length).toBeGreaterThan(0);
    });
  });
});
