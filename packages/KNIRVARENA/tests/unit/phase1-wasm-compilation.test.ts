/**
 * Phase 1 WASM Compilation Pipeline Tests
 * Tests for KNIRV-CORTEX Backend Isolation (1.2) from MAJOR_REFACTOR_IMPLEMENTATION_PLAN.md
 */

import { describe, it, expect, beforeEach, afterEach } from '@jest/globals';
import { AgentCoreCompiler, AgentCoreConfig } from '../../src/core/agent-core-compiler/src/AgentCoreCompiler';
import { WASMOrchestrator } from '../../src/sensory-shell/WASMOrchestrator';
import { ProtobufHandler } from '../../src/core/protobuf/ProtobufHandler';
import { promises as fs } from 'fs';
import { join } from 'path';

describe('Phase 1: WASM Compilation Pipeline Tests', () => {
  let agentCompiler: AgentCoreCompiler;
  let wasmOrchestrator: WASMOrchestrator;
  let protobufHandler: ProtobufHandler;
  let tempDir: string;

  beforeEach(async () => {
    agentCompiler = new AgentCoreCompiler();
    wasmOrchestrator = new WASMOrchestrator();
    protobufHandler = new ProtobufHandler();
    
    // Create temporary directory for test builds
    tempDir = join(__dirname, '../../temp/wasm-tests');
    await fs.mkdir(tempDir, { recursive: true });
  });

  afterEach(async () => {
    if (agentCompiler.isReady()) {
      await agentCompiler.dispose();
    }
    if (wasmOrchestrator.isRunning) {
      await wasmOrchestrator.shutdown();
    }
    if (protobufHandler.isReady()) {
      await protobufHandler.cleanup();
    }
    
    // Clean up temp directory
    try {
      await fs.rmdir(tempDir, { recursive: true });
    } catch {
      // Ignore cleanup errors
    }
  });

  describe('1.2 KNIRV-CORTEX Backend Isolation', () => {
    describe('WASM Compilation Pipeline', () => {
      beforeEach(async () => {
        await agentCompiler.initialize();
        await protobufHandler.initialize();
      });

      it('should compile TypeScript agent-core to WASM', async () => {
        const testConfig: AgentCoreConfig = {
          agentId: 'wasm-test-agent',
          agentName: 'WASM Test Agent',
          agentDescription: 'Agent for WASM compilation testing',
          agentVersion: '1.0.0',
          author: 'test',
          tools: [{
            name: 'testTool',
            description: 'Test tool for WASM',
            parameters: [{
              name: 'input',
              type: 'string',
              required: true,
              description: 'Test input'
            }],
            implementation: 'return { result: parameters.input.toUpperCase() };',
            sourceType: 'inline'
          }],
          cognitiveCapabilities: [{
            name: 'lora',
            enabled: true,
            config: { rank: 8, alpha: 16.0 }
          }],
          sensoryInterfaces: [{
            type: 'text',
            enabled: true,
            config: {}
          }],
          buildTarget: 'wasm',
          optimizationLevel: 'basic'
        };

        const result = await agentCompiler.compileAgentCore(testConfig);
        
        expect(result.success).toBe(true);
        expect(result.wasmBytes).toBeDefined();
        expect(result.wasmBytes!.length).toBeGreaterThan(0);
        expect(result.metadata.wasmSize).toBeGreaterThan(0);
        expect(result.metadata.compilationTime).toBeGreaterThan(0);
      });

      it('should generate valid WASM binary format', async () => {
        const testConfig: AgentCoreConfig = {
          agentId: 'wasm-format-test',
          agentName: 'WASM Format Test',
          agentDescription: 'Test WASM binary format',
          agentVersion: '1.0.0',
          author: 'test',
          tools: [],
          cognitiveCapabilities: [],
          sensoryInterfaces: [],
          buildTarget: 'wasm',
          optimizationLevel: 'basic'
        };

        const result = await agentCompiler.compileAgentCore(testConfig);
        
        expect(result.success).toBe(true);
        expect(result.wasmBytes).toBeDefined();
        
        // Check WASM magic number (0x00 0x61 0x73 0x6d)
        const wasmBytes = result.wasmBytes!;
        expect(wasmBytes[0]).toBe(0x00);
        expect(wasmBytes[1]).toBe(0x61);
        expect(wasmBytes[2]).toBe(0x73);
        expect(wasmBytes[3]).toBe(0x6d);
        
        // Check WASM version (0x01 0x00 0x00 0x00)
        expect(wasmBytes[4]).toBe(0x01);
        expect(wasmBytes[5]).toBe(0x00);
        expect(wasmBytes[6]).toBe(0x00);
        expect(wasmBytes[7]).toBe(0x00);
      });

      it('should preserve model training infrastructure in backend-only configuration', async () => {
        const testConfig: AgentCoreConfig = {
          agentId: 'training-test-agent',
          agentName: 'Training Test Agent',
          agentDescription: 'Agent with training infrastructure',
          agentVersion: '1.0.0',
          author: 'test',
          tools: [],
          cognitiveCapabilities: [{
            name: 'lora',
            enabled: true,
            config: { rank: 16, alpha: 32.0 }
          }, {
            name: 'adaptiveLearning',
            enabled: true,
            config: { learningRate: 0.01 }
          }],
          sensoryInterfaces: [],
          buildTarget: 'wasm',
          optimizationLevel: 'basic'
        };

        const result = await agentCompiler.compileAgentCore(testConfig);
        
        expect(result.success).toBe(true);
        expect(result.metadata.cognitiveCapabilities).toContain('lora');
        expect(result.metadata.cognitiveCapabilities).toContain('adaptiveLearning');
        expect(result.typeScriptCode).toContain('LoRAAdapter');
        expect(result.typeScriptCode).toContain('AdaptiveLearningPipeline');
      });

      it('should support deployment sequence functionality', async () => {
        const testConfig: AgentCoreConfig = {
          agentId: 'deployment-test-agent',
          agentName: 'Deployment Test Agent',
          agentDescription: 'Agent for deployment testing',
          agentVersion: '1.0.0',
          author: 'test',
          tools: [],
          cognitiveCapabilities: [],
          sensoryInterfaces: [],
          buildTarget: 'wasm',
          optimizationLevel: 'basic'
        };

        const result = await agentCompiler.compileAgentCore(testConfig);
        
        expect(result.success).toBe(true);
        expect(result.wasmBytes).toBeDefined();
        
        // Test that WASM can be loaded by orchestrator
        await wasmOrchestrator.start();
        const loadResult = await wasmOrchestrator.loadAgentWASM(result.wasmBytes!);
        expect(loadResult).toBe(true);
      });
    });

    describe('Protobuf Serialization/Deserialization for LoRA Adapters', () => {
      beforeEach(async () => {
        await protobufHandler.initialize();
      });

      it('should serialize LoRA adapter responses correctly', async () => {
        const testAdapter = {
          skill_id: 'test-lora-skill',
          skill_name: 'Test LoRA Skill',
          description: 'Test LoRA adapter for serialization',
          base_model_compatibility: 'test-model',
          version: 1,
          rank: 8,
          alpha: 16.0,
          weightsA: new Float32Array([0.1, 0.2, 0.3, 0.4]),
          weightsB: new Float32Array([0.5, 0.6, 0.7, 0.8]),
          additional_metadata: { test: 'metadata' }
        };

        const serialized = await protobufHandler.serializeLoRAAdapter(testAdapter);
        expect(serialized).toBeInstanceOf(Uint8Array);
        expect(serialized.length).toBeGreaterThan(0);
      });

      it('should deserialize LoRA adapter responses correctly', async () => {
        const testAdapter = {
          skill_id: 'test-deserialize-skill',
          skill_name: 'Test Deserialize Skill',
          description: 'Test LoRA adapter for deserialization',
          base_model_compatibility: 'test-model',
          version: 1,
          rank: 4,
          alpha: 8.0,
          weightsA: new Float32Array([1.0, 2.0, 3.0, 4.0]),
          weightsB: new Float32Array([5.0, 6.0, 7.0, 8.0]),
          additional_metadata: {}
        };

        const serialized = await protobufHandler.serializeLoRAAdapter(testAdapter);
        const deserialized = await protobufHandler.deserializeLoRAAdapter(serialized);

        expect((deserialized as any).skill_id).toBe(testAdapter.skill_id);
        expect((deserialized as any).skill_name).toBe(testAdapter.skill_name);
        expect((deserialized as any).rank).toBe(testAdapter.rank);
        expect((deserialized as any).alpha).toBe(testAdapter.alpha);
        expect((deserialized as any).weightsA).toEqual(testAdapter.weightsA);
        expect((deserialized as any).weightsB).toEqual(testAdapter.weightsB);
      });

      it('should handle skill invocation results correctly', async () => {
        const invocationResponse = await protobufHandler.createSkillInvocationResponse(
          'test-invocation-123',
          'SUCCESS',
          {
            skill_id: 'invocation-test-skill',
            skill_name: 'Invocation Test Skill',
            description: 'Test skill for invocation',
            base_model_compatibility: 'test-model',
            version: 1,
            rank: 8,
            alpha: 16.0,
            weights_a: new Uint8Array([1, 2, 3, 4]),
            weights_b: new Uint8Array([5, 6, 7, 8]),
            additional_metadata: {}
          }
        );

        expect(invocationResponse).toBeInstanceOf(Uint8Array);
        
        const deserialized = await protobufHandler.deserialize(invocationResponse, 'SkillInvocationResponse');
        expect((deserialized as any).invocation_id).toBe('test-invocation-123');
        expect((deserialized as any).status).toBe('SUCCESS');
        expect((deserialized as any).skill.skill_id).toBe('invocation-test-skill');
      });

      it('should convert Float32Array to bytes and back correctly', () => {
        const originalArray = new Float32Array([1.5, 2.7, 3.14159, -0.5, 100.0]);
        
        const bytes = protobufHandler.floatArrayToBytes(originalArray);
        expect(bytes).toBeInstanceOf(Uint8Array);
        expect(bytes.length).toBe(originalArray.length * 4); // 4 bytes per float
        
        const convertedArray = protobufHandler.bytesToFloatArray(bytes);
        expect(convertedArray).toBeInstanceOf(Float32Array);
        expect(convertedArray.length).toBe(originalArray.length);
        
        // Check values are approximately equal (floating point precision)
        for (let i = 0; i < originalArray.length; i++) {
          expect(convertedArray[i]).toBeCloseTo(originalArray[i], 5);
        }
      });
    });

    describe('Backend-Frontend Separation', () => {
      beforeEach(async () => {
        await agentCompiler.initialize();
        await wasmOrchestrator.start();
      });

      it('should isolate backend WASM compilation from frontend', async () => {
        const backendConfig: AgentCoreConfig = {
          agentId: 'backend-isolation-test',
          agentName: 'Backend Isolation Test',
          agentDescription: 'Test backend isolation',
          agentVersion: '1.0.0',
          author: 'test',
          tools: [],
          cognitiveCapabilities: [{
            name: 'lora',
            enabled: true,
            config: {}
          }],
          sensoryInterfaces: [],
          buildTarget: 'wasm',
          optimizationLevel: 'basic'
        };

        const result = await agentCompiler.compileAgentCore(backendConfig);
        
        expect(result.success).toBe(true);
        expect(result.typeScriptCode).toBeDefined();
        
        // Should not contain frontend-specific code
        expect(result.typeScriptCode).not.toContain('React');
        expect(result.typeScriptCode).not.toContain('DOM');
        expect(result.typeScriptCode).not.toContain('window');
        expect(result.typeScriptCode).not.toContain('document');
        
        // Should contain backend/WASM-specific code
        expect(result.typeScriptCode).toContain('AgentCore');
        expect(result.typeScriptCode).toContain('CognitiveEngine');
        expect(result.typeScriptCode).toContain('globalThis');
      });

      it('should establish clear communication channels between layers', async () => {
        const testConfig: AgentCoreConfig = {
          agentId: 'communication-test',
          agentName: 'Communication Test',
          agentDescription: 'Test communication channels',
          agentVersion: '1.0.0',
          author: 'test',
          tools: [],
          cognitiveCapabilities: [],
          sensoryInterfaces: [],
          buildTarget: 'wasm',
          optimizationLevel: 'basic'
        };

        const result = await agentCompiler.compileAgentCore(testConfig);
        
        expect(result.success).toBe(true);
        expect(result.typeScriptCode).toContain('agentCoreExecute');
        expect(result.typeScriptCode).toContain('agentCoreExecuteTool');
        expect(result.typeScriptCode).toContain('agentCoreLoadLoRA');
        expect(result.typeScriptCode).toContain('agentCoreGetStatus');
      });
    });
  });

  describe('Performance and Optimization Tests', () => {
    beforeEach(async () => {
      await agentCompiler.initialize();
    });

    it('should compile with different optimization levels', async () => {
      const optimizationLevels: Array<'none' | 'basic' | 'aggressive'> = ['none', 'basic', 'aggressive'];
      
      for (const level of optimizationLevels) {
        const testConfig: AgentCoreConfig = {
          agentId: `optimization-${level}-test`,
          agentName: `Optimization ${level} Test`,
          agentDescription: `Test ${level} optimization`,
          agentVersion: '1.0.0',
          author: 'test',
          tools: [],
          cognitiveCapabilities: [],
          sensoryInterfaces: [],
          buildTarget: 'wasm',
          optimizationLevel: level
        };

        const result = await agentCompiler.compileAgentCore(testConfig);
        
        expect(result.success).toBe(true);
        expect(result.metadata.optimizationLevel).toBe(level);
        expect(result.wasmBytes).toBeDefined();
      }
    });

    it('should handle large agent configurations efficiently', async () => {
      const largeConfig: AgentCoreConfig = {
        agentId: 'large-config-test',
        agentName: 'Large Configuration Test',
        agentDescription: 'Test with many tools and capabilities',
        agentVersion: '1.0.0',
        author: 'test',
        tools: Array.from({ length: 10 }, (_, i) => ({
          name: `tool${i}`,
          description: `Test tool ${i}`,
          parameters: [{
            name: 'input',
            type: 'string',
            required: true,
            description: 'Input parameter'
          }],
          implementation: `return { result: "Tool ${i} result: " + parameters.input };`,
          sourceType: 'inline' as const
        })),
        cognitiveCapabilities: [
          { name: 'lora', enabled: true, config: { rank: 16 } },
          { name: 'adaptiveLearning', enabled: true, config: {} },
          { name: 'voice', enabled: true, config: {} },
          { name: 'visual', enabled: true, config: {} }
        ],
        sensoryInterfaces: [
          { type: 'text', enabled: true, config: {} },
          { type: 'voice', enabled: true, config: {} },
          { type: 'visual', enabled: true, config: {} }
        ],
        buildTarget: 'wasm',
        optimizationLevel: 'basic'
      };

      const startTime = Date.now();
      const result = await agentCompiler.compileAgentCore(largeConfig);
      const compilationTime = Date.now() - startTime;

      expect(result.success).toBe(true);
      expect(result.metadata.cognitiveCapabilities.length).toBe(4);
      expect(result.metadata.sensoryInterfaces.length).toBe(3);
      
      // Should complete within reasonable time (less than 10 seconds)
      expect(compilationTime).toBeLessThan(10000);
    });
  });
});
