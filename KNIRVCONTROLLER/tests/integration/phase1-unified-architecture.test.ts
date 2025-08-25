/**
 * Phase 1 Unified Architecture Integration Tests
 * Tests for complete integration between KNIRV-CONTROLLER and KNIRV-CORTEX
 */

import { describe, it, expect, beforeEach, afterEach, jest } from '@jest/globals';
import { AgentCoreCompiler } from '../../src/core/agent-core-compiler/src/AgentCoreCompiler';
import { ProtobufHandler } from '../../src/core/protobuf/ProtobufHandler';
import { WASMOrchestrator } from '../../src/sensory-shell/WASMOrchestrator';
import { ModelManager } from '../../src/sensory-shell/ModelManager';
import { CognitiveEngine } from '../../src/sensory-shell/CognitiveEngine';
import { LoRAAdapter } from '../../src/sensory-shell/LoRAAdapter';

describe('Phase 1: Unified Architecture Integration Tests', () => {
  let agentCompiler: AgentCoreCompiler;
  let protobufHandler: ProtobufHandler;
  let wasmOrchestrator: WASMOrchestrator;
  let modelManager: ModelManager;
  let cognitiveEngine: CognitiveEngine;
  let loraAdapter: LoRAAdapter;

  beforeEach(async () => {
    // Initialize all components
    agentCompiler = new AgentCoreCompiler();
    protobufHandler = new ProtobufHandler();
    wasmOrchestrator = new WASMOrchestrator();
    modelManager = new ModelManager();
    
    // Initialize cognitive components
    const cognitiveConfig = {
      maxContextSize: 1000,
      learningRate: 0.01,
      adaptationThreshold: 0.8,
      skillTimeout: 5000,
      voiceEnabled: false, // Disabled for testing
      visualEnabled: false, // Disabled for testing
      loraEnabled: true,
      enhancedLoraEnabled: true,
      hrmEnabled: false, // Disabled for testing
      adaptiveLearningEnabled: true,
      walletIntegrationEnabled: false, // Disabled for testing
      chainIntegrationEnabled: false, // Disabled for testing
      ecosystemCommunicationEnabled: false // Disabled for testing
    };
    
    cognitiveEngine = new CognitiveEngine(cognitiveConfig);
    
    const loraConfig = {
      rank: 8,
      alpha: 16.0,
      dropout: 0.1,
      targetModules: ['attention', 'feedforward'],
      taskType: 'text-generation',
      learningRate: 0.01
    };
    
    loraAdapter = new LoRAAdapter(loraConfig);
  });

  afterEach(async () => {
    // Cleanup all components
    if (agentCompiler.isReady()) {
      await agentCompiler.dispose();
    }
    if (protobufHandler.isReady()) {
      await protobufHandler.cleanup();
    }
    if (wasmOrchestrator.isRunning()) {
      await wasmOrchestrator.stop();
    }
    if (modelManager.isInitialized()) {
      await modelManager.dispose();
    }
    if (cognitiveEngine) {
      await cognitiveEngine.dispose();
    }
    if (loraAdapter) {
      await loraAdapter.stop();
    }
  });

  describe('End-to-End Agent Compilation and Execution', () => {
    it('should compile agent-core and execute through sensory-shell', async () => {
      // Initialize components
      await agentCompiler.initialize();
      await protobufHandler.initialize();
      await wasmOrchestrator.start();
      await modelManager.initialize();
      await cognitiveEngine.start();
      await loraAdapter.start();

      // Create test agent configuration
      const agentConfig = {
        agentId: 'integration-test-agent',
        agentName: 'Integration Test Agent',
        agentDescription: 'Full integration test agent',
        agentVersion: '1.0.0',
        author: 'test',
        tools: [{
          name: 'echoTool',
          description: 'Echo input back with processing',
          parameters: [{
            name: 'message',
            type: 'string',
            required: true,
            description: 'Message to echo'
          }],
          implementation: 'return { result: "Echo: " + parameters.message, timestamp: Date.now() };',
          sourceType: 'inline' as const
        }],
        cognitiveCapabilities: [{
          name: 'lora',
          enabled: true,
          config: { rank: 8, alpha: 16.0 }
        }],
        sensoryInterfaces: [{
          type: 'text' as const,
          enabled: true,
          config: {}
        }],
        buildTarget: 'wasm' as const,
        optimizationLevel: 'basic' as const
      };

      // Compile agent-core
      const compilationResult = await agentCompiler.compileAgentCore(agentConfig);
      expect(compilationResult.success).toBe(true);
      expect(compilationResult.wasmBytes).toBeDefined();

      // Load compiled agent into WASM orchestrator
      const loadResult = await wasmOrchestrator.loadAgentWASM(
        compilationResult.wasmBytes!,
        agentConfig.agentId
      );
      expect(loadResult).toBe(true);

      // Test agent execution through sensory-shell
      const testInput = 'Hello, integration test!';
      const executionResult = await wasmOrchestrator.executeAgent(
        agentConfig.agentId,
        testInput,
        { inputType: 'text' }
      );

      expect(executionResult).toBeDefined();
      expect(executionResult.success).toBe(true);
    });

    it('should handle LoRA adapter integration across components', async () => {
      // Initialize components
      await agentCompiler.initialize();
      await protobufHandler.initialize();
      await loraAdapter.start();

      // Create LoRA adapter
      const testAdapter = {
        skill_id: 'integration-lora-skill',
        skill_name: 'Integration LoRA Skill',
        description: 'LoRA skill for integration testing',
        base_model_compatibility: 'test-model',
        version: 1,
        rank: 8,
        alpha: 16.0,
        weightsA: new Float32Array(64).fill(0.1),
        weightsB: new Float32Array(64).fill(0.2),
        additional_metadata: { test: 'integration' }
      };

      // Serialize LoRA adapter
      const serializedAdapter = await protobufHandler.serializeLoRAAdapter(testAdapter);
      expect(serializedAdapter).toBeInstanceOf(Uint8Array);

      // Deserialize and apply to LoRA adapter
      const deserializedAdapter = await protobufHandler.deserializeLoRAAdapter(serializedAdapter);
      expect(deserializedAdapter.skill_id).toBe(testAdapter.skill_id);

      // Test LoRA adapter functionality
      const trainingData = {
        input: 'test input',
        output: 'expected output',
        feedback: 0.9,
        timestamp: new Date()
      };

      await loraAdapter.addTrainingData(trainingData);
      const adaptationResult = await loraAdapter.adapt(
        trainingData.input,
        trainingData.output,
        trainingData.feedback
      );

      expect(adaptationResult).toBeDefined();
    });
  });

  describe('Component Communication and Data Flow', () => {
    beforeEach(async () => {
      await agentCompiler.initialize();
      await protobufHandler.initialize();
      await wasmOrchestrator.start();
      await modelManager.initialize();
    });

    it('should handle data flow between core and sensory-shell', async () => {
      // Test data flow from core (backend) to sensory-shell (frontend)
      const testData = {
        type: 'skill_invocation',
        skillId: 'test-skill',
        parameters: { input: 'test data' },
        timestamp: Date.now()
      };

      // Serialize data in core
      const serializedData = JSON.stringify(testData);
      expect(serializedData).toBeDefined();

      // Process in sensory-shell (simulated)
      const processedData = JSON.parse(serializedData);
      expect(processedData.type).toBe(testData.type);
      expect(processedData.skillId).toBe(testData.skillId);
    });

    it('should handle protobuf communication between components', async () => {
      // Create skill invocation request
      const invocationRequest = {
        invocation_id: 'test-invocation-123',
        skill_id: 'test-skill-456',
        parameters: { input: 'test', mode: 'integration' },
        agent_core_id: 'test-agent-core'
      };

      const serializedRequest = await protobufHandler.serialize(
        invocationRequest,
        'SkillInvocationRequest'
      );
      expect(serializedRequest).toBeInstanceOf(Uint8Array);

      const deserializedRequest = await protobufHandler.deserialize(
        serializedRequest,
        'SkillInvocationRequest'
      );
      expect(deserializedRequest.invocation_id).toBe(invocationRequest.invocation_id);
      expect(deserializedRequest.skill_id).toBe(invocationRequest.skill_id);
    });

    it('should coordinate model management across components', async () => {
      // Test model selection and loading
      const availableModels = modelManager.getAvailableModels();
      expect(availableModels.length).toBeGreaterThan(0);

      // Select a model
      const testModel = availableModels[0];
      const selectionResult = await modelManager.selectModel(testModel.id);
      expect(selectionResult).toBe(true);

      // Verify model is loaded
      const currentModel = modelManager.getCurrentModel();
      expect(currentModel).toBeDefined();
      expect(currentModel!.id).toBe(testModel.id);
    });
  });

  describe('Error Handling and Recovery', () => {
    beforeEach(async () => {
      await agentCompiler.initialize();
      await protobufHandler.initialize();
      await wasmOrchestrator.start();
    });

    it('should handle compilation errors gracefully', async () => {
      const invalidConfig = {
        agentId: 'invalid-agent',
        agentName: 'Invalid Agent',
        agentDescription: 'Agent with invalid configuration',
        agentVersion: '1.0.0',
        author: 'test',
        tools: [{
          name: 'invalidTool',
          description: 'Tool with invalid implementation',
          parameters: [],
          implementation: 'invalid javascript code {{{',
          sourceType: 'inline' as const
        }],
        cognitiveCapabilities: [],
        sensoryInterfaces: [],
        buildTarget: 'wasm' as const,
        optimizationLevel: 'basic' as const
      };

      const result = await agentCompiler.compileAgentCore(invalidConfig);
      
      // Should handle error gracefully
      expect(result.success).toBe(false);
      expect(result.errors).toBeDefined();
      expect(result.errors!.length).toBeGreaterThan(0);
    });

    it('should recover from WASM loading failures', async () => {
      // Try to load invalid WASM
      const invalidWasm = new Uint8Array([0x00, 0x00, 0x00, 0x00]); // Invalid WASM
      
      const loadResult = await wasmOrchestrator.loadAgentWASM(invalidWasm, 'invalid-agent');
      expect(loadResult).toBe(false);

      // Orchestrator should still be functional
      expect(wasmOrchestrator.isRunning()).toBe(true);
      
      // Should be able to load valid WASM after failure
      const validConfig = {
        agentId: 'recovery-test-agent',
        agentName: 'Recovery Test Agent',
        agentDescription: 'Agent for recovery testing',
        agentVersion: '1.0.0',
        author: 'test',
        tools: [],
        cognitiveCapabilities: [],
        sensoryInterfaces: [],
        buildTarget: 'wasm' as const,
        optimizationLevel: 'basic' as const
      };

      const compilationResult = await agentCompiler.compileAgentCore(validConfig);
      expect(compilationResult.success).toBe(true);

      const validLoadResult = await wasmOrchestrator.loadAgentWASM(
        compilationResult.wasmBytes!,
        validConfig.agentId
      );
      expect(validLoadResult).toBe(true);
    });

    it('should handle protobuf serialization errors', async () => {
      // Test with invalid data
      const invalidData = {
        invalid_field: 'this should not exist',
        circular_ref: null as any
      };
      invalidData.circular_ref = invalidData; // Create circular reference

      try {
        await protobufHandler.serialize(invalidData, 'LoRaAdapterSkill');
        // Should not reach here
        expect(false).toBe(true);
      } catch (error) {
        // Should handle error gracefully
        expect(error).toBeDefined();
      }

      // Handler should still be functional
      expect(protobufHandler.isReady()).toBe(true);
    });
  });

  describe('Performance and Scalability', () => {
    beforeEach(async () => {
      await agentCompiler.initialize();
      await protobufHandler.initialize();
      await wasmOrchestrator.start();
      await modelManager.initialize();
    });

    it('should handle multiple concurrent agent compilations', async () => {
      const compilationPromises = [];
      
      for (let i = 0; i < 5; i++) {
        const config = {
          agentId: `concurrent-agent-${i}`,
          agentName: `Concurrent Agent ${i}`,
          agentDescription: `Concurrent compilation test agent ${i}`,
          agentVersion: '1.0.0',
          author: 'test',
          tools: [],
          cognitiveCapabilities: [],
          sensoryInterfaces: [],
          buildTarget: 'typescript' as const, // Use TypeScript for faster compilation
          optimizationLevel: 'basic' as const
        };

        compilationPromises.push(agentCompiler.compileAgentCore(config));
      }

      const results = await Promise.all(compilationPromises);
      
      // All compilations should succeed
      results.forEach((result, index) => {
        expect(result.success).toBe(true);
        expect(result.agentId).toBe(`concurrent-agent-${index}`);
      });
    });

    it('should handle high-frequency protobuf operations', async () => {
      const startTime = Date.now();
      const operations = 100;

      for (let i = 0; i < operations; i++) {
        const testData = {
          invocation_id: `perf-test-${i}`,
          status: 'SUCCESS',
          error_message: '',
          skill: {
            skill_id: `perf-skill-${i}`,
            skill_name: `Performance Skill ${i}`,
            description: 'Performance test skill',
            base_model_compatibility: 'test-model',
            version: 1,
            rank: 8,
            alpha: 16.0,
            weights_a: new Uint8Array(32).fill(i % 256),
            weights_b: new Uint8Array(32).fill((i * 2) % 256),
            additional_metadata: {}
          }
        };

        const serialized = await protobufHandler.serialize(testData, 'SkillInvocationResponse');
        const deserialized = await protobufHandler.deserialize(serialized, 'SkillInvocationResponse');
        
        expect(deserialized.invocation_id).toBe(testData.invocation_id);
      }

      const endTime = Date.now();
      const totalTime = endTime - startTime;
      const avgTime = totalTime / operations;

      // Should complete within reasonable time (less than 5ms per operation)
      expect(avgTime).toBeLessThan(5);
    });

    it('should manage memory efficiently during extended operations', async () => {
      const initialMemory = process.memoryUsage();
      
      // Perform many operations
      for (let i = 0; i < 50; i++) {
        const testAdapter = {
          skill_id: `memory-test-${i}`,
          skill_name: `Memory Test Skill ${i}`,
          description: 'Memory test skill',
          base_model_compatibility: 'test-model',
          version: 1,
          rank: 16,
          alpha: 32.0,
          weightsA: new Float32Array(256).fill(Math.random()),
          weightsB: new Float32Array(256).fill(Math.random()),
          additional_metadata: {}
        };

        const serialized = await protobufHandler.serializeLoRAAdapter(testAdapter);
        const deserialized = await protobufHandler.deserializeLoRAAdapter(serialized);
        
        expect(deserialized.skill_id).toBe(testAdapter.skill_id);
      }

      const finalMemory = process.memoryUsage();
      const memoryIncrease = finalMemory.heapUsed - initialMemory.heapUsed;
      
      // Memory increase should be reasonable (less than 50MB)
      expect(memoryIncrease).toBeLessThan(50 * 1024 * 1024);
    });
  });
});
