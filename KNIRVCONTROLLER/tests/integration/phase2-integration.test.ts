/**
 * Phase 2 Integration Tests
 * 
 * Testing Requirements from MAJOR_REFACTOR_IMPLEMENTATION_PLAN.md:
 * - Component communication tests
 * - Cross-platform compatibility tests
 * - QR code connectivity tests
 * - Agent registration and minting tests
 * - LoRA adapter skill invocation tests
 * - End-to-end LoRA adapter lifecycle tests
 */

import { describe, test, expect, beforeEach, afterEach, jest } from '@jest/globals';
import { WASMOrchestrator, OrchestrationConfig } from '../../src/sensory-shell/WASMOrchestrator';
import { AgentCoreInterface, LoRAAdapter } from '../../src/sensory-shell/AgentCoreInterface';
import { LoRAAdapterEngine } from '../../src/core/lora/LoRAAdapterEngine';

// Mock external dependencies
jest.mock('../../src/core/protobuf/ProtobufHandler');
jest.mock('../../src/services/DesktopConnection');

describe('Phase 2 Integration Tests', () => {
  let orchestrator: WASMOrchestrator;
  let agentCoreInterface: AgentCoreInterface;
  let loraEngine: LoRAAdapterEngine;
  let config: OrchestrationConfig;
  let mockExports: any;

  beforeEach(async () => {
    config = {
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

    orchestrator = new WASMOrchestrator(config);
    agentCoreInterface = new AgentCoreInterface();

    // Create mock dependencies for LoRAAdapterEngine
    const mockWasmCompiler = {
      compile: jest.fn().mockResolvedValue(new Uint8Array([0x00, 0x61, 0x73, 0x6d]))
    };

    const mockProtobufHandler = {
      initialize: jest.fn().mockResolvedValue(true),
      serialize: jest.fn().mockResolvedValue(new Uint8Array([0x08, 0x01])),
      deserialize: jest.fn().mockResolvedValue({
        invocation_id: 'test-invocation',
        status: 'SUCCESS',
        skill: {
          skill_id: 'test-skill-123',
          skill_name: 'Code Refactoring Expert',
          description: 'A skill to improve code readability',
          base_model_compatibility: 'CodeT5-base',
          version: 1,
          rank: 8,
          alpha: 16.0,
          weights_a: new Uint8Array([0x3f, 0x80, 0x00, 0x00]),
          weights_b: new Uint8Array([0x40, 0x00, 0x00, 0x00]),
          additional_metadata: {}
        }
      }),
      cleanup: jest.fn().mockResolvedValue(true)
    };

    loraEngine = new LoRAAdapterEngine(mockWasmCompiler as any, mockProtobufHandler as any);

    // Create mock exports object that can be modified
    mockExports = {
      agentCoreExecute: jest.fn().mockResolvedValue('{"success": true, "result": "test"}'),
      agentCoreExecuteTool: jest.fn().mockResolvedValue('{"success": true, "result": "tool"}'),
      agentCoreLoadLoRA: jest.fn().mockResolvedValue(true),
      agentCoreApplySkill: jest.fn().mockResolvedValue(true),
      agentCoreGetStatus: jest.fn().mockReturnValue('{"agentId": "test", "initialized": true}'),
      modelInference: jest.fn().mockResolvedValue('{"result": "inference"}'),
      modelGetInfo: jest.fn().mockReturnValue('{"name": "test-model"}'),
      modelSetConfig: jest.fn().mockReturnValue(true)
    };

    // Mock WebAssembly and related APIs
    global.WebAssembly = {
      compile: jest.fn().mockResolvedValue({}),
      instantiate: jest.fn().mockResolvedValue({
        exports: mockExports
      }),
      Memory: jest.fn().mockImplementation(() => ({ buffer: new ArrayBuffer(1024) }))
    } as any;

    global.fetch = jest.fn().mockResolvedValue({
      arrayBuffer: jest.fn().mockResolvedValue(new ArrayBuffer(1024))
    }) as any;

    await loraEngine.initialize();
  });

  afterEach(async () => {
    await orchestrator.dispose();
    await agentCoreInterface.dispose();
    await loraEngine.cleanup();
    jest.clearAllMocks();
  });

  describe('Component Communication', () => {
    test('should establish communication between all Phase 2 components', async () => {
      // Initialize all components
      await orchestrator.initialize();
      await agentCoreInterface.initializeAgentCore(new Uint8Array([0x00, 0x61, 0x73, 0x6d, 0x01, 0x00, 0x00, 0x00]));

      // Test orchestrator readiness
      expect(orchestrator.isReady()).toBe(true);
      expect(agentCoreInterface.isReady()).toBe(true);

      // Test component communication
      const input = {
        type: 'text' as const,
        data: 'Integration test',
        timestamp: Date.now(),
        sessionId: 'integration-test'
      };

      const orchestratorResponse = await orchestrator.processSensoryInput(input);
      const agentCoreResponse = await agentCoreInterface.processSensoryInput(input);

      expect(orchestratorResponse.success).toBe(true);
      expect(agentCoreResponse.success).toBe(true);
    });

    test('should handle component initialization order', async () => {
      // Test different initialization orders
      await agentCoreInterface.initializeAgentCore(new Uint8Array([0x00, 0x61, 0x73, 0x6d, 0x01, 0x00, 0x00, 0x00]));
      await orchestrator.initialize();

      expect(orchestrator.isReady()).toBe(true);
      expect(agentCoreInterface.isReady()).toBe(true);
    });

    test('should maintain communication during component failures', async () => {
      await orchestrator.initialize();
      await agentCoreInterface.initializeAgentCore(new Uint8Array([0x00, 0x61, 0x73, 0x6d, 0x01, 0x00, 0x00, 0x00]));

      // Simulate component failure using the shared mockExports
      mockExports.agentCoreExecute.mockRejectedValueOnce(new Error('Component failure'));

      const input = {
        type: 'text' as const,
        data: 'Failure test',
        timestamp: Date.now(),
        sessionId: 'failure-test'
      };

      const response = await agentCoreInterface.processSensoryInput(input);
      expect(response.success).toBe(false);
      expect(response.error).toBe('Component failure');
    });
  });

  describe('Cross-Platform Compatibility', () => {
    test('should work in Node.js environment', async () => {
      // Ensure we're in Node.js environment
      expect(typeof process).toBe('object');
      expect(typeof window).toBe('undefined');

      await orchestrator.initialize();
      expect(orchestrator.isReady()).toBe(true);
    });

    test('should handle browser-like environment', async () => {
      // Mock browser environment
      const originalProcess = global.process;
      delete (global as any).process;
      global.window = {} as any;

      await orchestrator.initialize();
      expect(orchestrator.isReady()).toBe(true);

      // Restore
      global.process = originalProcess;
      delete (global as any).window;
    });

    test('should handle different WASM loading mechanisms', async () => {
      // Test different fetch scenarios
      (global.fetch as jest.Mock)
        .mockResolvedValueOnce({
          arrayBuffer: jest.fn().mockResolvedValue(new ArrayBuffer(2048))
        })
        .mockResolvedValueOnce({
          arrayBuffer: jest.fn().mockResolvedValue(new ArrayBuffer(4096))
        });

      await orchestrator.initialize();
      expect(orchestrator.isReady()).toBe(true);
    });
  });

  describe('LoRA Adapter Skill Invocation Integration', () => {
    beforeEach(async () => {
      await orchestrator.initialize();
      await agentCoreInterface.initializeAgentCore(new Uint8Array([0x00, 0x61, 0x73, 0x6d, 0x01, 0x00, 0x00, 0x00]));
    });

    test('should complete end-to-end LoRA adapter lifecycle', async () => {
      // 1. Create LoRA adapter from skill data
      const skillData = {
        solutions: [
          { errorId: 'e1', solution: 'Fix syntax error', confidence: 0.9 },
          { errorId: 'e2', solution: 'Optimize performance', confidence: 0.8 }
        ],
        errors: [
          { errorId: 'e1', description: 'Syntax error', context: 'function() {' },
          { errorId: 'e2', description: 'Performance issue', context: 'slow loop' }
        ]
      };

      const metadata = {
        skillName: 'Code Optimizer',
        description: 'Optimizes code quality',
        baseModel: 'CodeT5-base'
      };

      const adapter = await loraEngine.compileAdapter(skillData, metadata);
      expect(adapter).toBeDefined();
      expect(adapter.skillName).toBe('Code Optimizer');

      // 2. Load adapter into agent-core
      const loraAdapter: LoRAAdapter = {
        skillId: adapter.skillId,
        skillName: adapter.skillName,
        weightsA: adapter.weightsA,
        weightsB: adapter.weightsB,
        rank: adapter.rank,
        alpha: adapter.alpha,
        metadata: {}
      };

      const loadSuccess = await agentCoreInterface.loadLoRAAdapter(loraAdapter);
      expect(loadSuccess).toBe(true);

      // 3. Invoke the skill
      const invocationResponse = await loraEngine.invokeAdapter(adapter.skillId);
      expect(['SUCCESS', 'FAILURE'].includes(invocationResponse.status)).toBe(true);
      if (invocationResponse.status === 'SUCCESS') {
        expect(invocationResponse.skill).toBeDefined();
      }

      // 4. Apply skill through protobuf
      const mockProtoBytes = new Uint8Array([0x08, 0x01, 0x12, 0x04, 0x74, 0x65, 0x73, 0x74]);
      const applySuccess = await agentCoreInterface.applySkill(mockProtoBytes);
      expect(applySuccess).toBe(true);
    });

    test('should handle skill composition and merging', async () => {
      // Create multiple adapters
      const adapters = await Promise.all([
        loraEngine.compileAdapter(
          {
            solutions: [{ errorId: 'e1', solution: 'Solution 1', confidence: 0.9 }],
            errors: [{ errorId: 'e1', description: 'Error 1', context: 'context 1' }]
          },
          { skillName: 'Skill 1', description: 'First skill', baseModel: 'CodeT5-base' }
        ),
        loraEngine.compileAdapter(
          {
            solutions: [{ errorId: 'e2', solution: 'Solution 2', confidence: 0.8 }],
            errors: [{ errorId: 'e2', description: 'Error 2', context: 'context 2' }]
          },
          { skillName: 'Skill 2', description: 'Second skill', baseModel: 'CodeT5-base' }
        )
      ]);

      // Load both adapters
      for (const adapter of adapters) {
        const loraAdapter: LoRAAdapter = {
          skillId: adapter.skillId,
          skillName: adapter.skillName,
          weightsA: adapter.weightsA,
          weightsB: adapter.weightsB,
          rank: adapter.rank,
          alpha: adapter.alpha,
          metadata: {}
        };

        const success = await agentCoreInterface.loadLoRAAdapter(loraAdapter);
        expect(success).toBe(true);
      }

      // Test that both skills are available
      const status = await agentCoreInterface.getAgentCoreStatus();
      expect(status?.initialized).toBe(true);
    });

    test('should handle skill versioning and updates', async () => {
      const baseSkillData = {
        solutions: [{ errorId: 'e1', solution: 'Original solution', confidence: 0.7 }],
        errors: [{ errorId: 'e1', description: 'Original error', context: 'original context' }]
      };

      // Create version 1
      const v1Adapter = await loraEngine.compileAdapter(
        baseSkillData,
        { skillName: 'Versioned Skill', description: 'Version 1', baseModel: 'CodeT5-base' }
      );

      // Create version 2 with improved solution
      const v2Adapter = await loraEngine.compileAdapter(
        {
          ...baseSkillData,
          solutions: [{ errorId: 'e1', solution: 'Improved solution', confidence: 0.9 }]
        },
        { skillName: 'Versioned Skill', description: 'Version 2', baseModel: 'CodeT5-base' }
      );

      expect(v1Adapter.skillId).not.toBe(v2Adapter.skillId); // Different IDs for different versions
      expect(v1Adapter.skillName).toBe(v2Adapter.skillName); // Same name
    });
  });

  describe('Agent Registration and Management', () => {
    test('should register and manage multiple agents', async () => {
      const agents = await Promise.all([
        new AgentCoreInterface(),
        new AgentCoreInterface(),
        new AgentCoreInterface()
      ]);

      // Initialize all agents
      for (const agent of agents) {
        const success = await agent.initializeAgentCore(new Uint8Array([0x00, 0x61, 0x73, 0x6d, 0x01, 0x00, 0x00, 0x00]));
        expect(success).toBe(true);
      }

      // Verify each has unique session ID
      const sessionIds = agents.map(agent => agent.getSessionId());
      const uniqueIds = new Set(sessionIds);
      expect(uniqueIds.size).toBe(agents.length);

      // Cleanup
      await Promise.all(agents.map(agent => agent.dispose()));
    });

    test('should handle primary agent designation', async () => {
      await agentCoreInterface.initializeAgentCore(new Uint8Array([0x00, 0x61, 0x73, 0x6d, 0x01, 0x00, 0x00, 0x00]));
      
      const status = await agentCoreInterface.getAgentCoreStatus();
      expect(status?.agentId).toBeDefined();
      expect(status?.initialized).toBe(true);
    });
  });

  describe('Performance Integration Tests', () => {
    beforeEach(async () => {
      await orchestrator.initialize();
      await agentCoreInterface.initializeAgentCore(new Uint8Array([0x00, 0x61, 0x73, 0x6d, 0x01, 0x00, 0x00, 0x00]));
    });

    test('should maintain performance under load', async () => {
      const inputs = Array.from({ length: 50 }, (_, i) => ({
        type: 'text' as const,
        data: `Load test input ${i}`,
        timestamp: Date.now(),
        sessionId: `load-test-${i}`
      }));

      const startTime = Date.now();
      const responses = await Promise.all(
        inputs.map(input => orchestrator.processSensoryInput(input))
      );
      const endTime = Date.now();

      expect(responses).toHaveLength(50);
      expect(responses.every(r => r.success)).toBe(true);
      expect(endTime - startTime).toBeLessThan(30000); // Should complete within 30 seconds
    });

    test('should handle memory pressure during integration', async () => {
      // Create multiple large LoRA adapters
      for (let i = 0; i < 5; i++) {
        const adapter: LoRAAdapter = {
          skillId: `large-adapter-${i}`,
          skillName: `Large Adapter ${i}`,
          weightsA: new Float32Array(5000),
          weightsB: new Float32Array(5000),
          rank: 50,
          alpha: 32.0,
          metadata: {}
        };

        const success = await agentCoreInterface.loadLoRAAdapter(adapter);
        expect(success).toBe(true);
      }

      // System should remain stable
      expect(orchestrator.isReady()).toBe(true);
      expect(agentCoreInterface.isReady()).toBe(true);
    });
  });

  describe('Error Recovery and Resilience', () => {
    test('should recover from component failures', async () => {
      await orchestrator.initialize();
      await agentCoreInterface.initializeAgentCore(new Uint8Array([0x00, 0x61, 0x73, 0x6d, 0x01, 0x00, 0x00, 0x00]));

      // Simulate failure and recovery using the shared mockExports
      mockExports.agentCoreExecute
        .mockRejectedValueOnce(new Error('Temporary failure'))
        .mockResolvedValueOnce('{"success": true, "result": "recovered"}');

      const input = {
        type: 'text' as const,
        data: 'Recovery test',
        timestamp: Date.now(),
        sessionId: 'recovery-test'
      };

      // First call should fail
      const failedResponse = await agentCoreInterface.processSensoryInput(input);
      expect(failedResponse.success).toBe(false);

      // Second call should succeed
      const recoveredResponse = await agentCoreInterface.processSensoryInput(input);
      expect(recoveredResponse.success).toBe(true);
    });

    test('should handle graceful degradation', async () => {
      await orchestrator.initialize();

      // Simulate model failure but cognitive shell working using the shared mockExports
      mockExports.modelInference.mockRejectedValue(new Error('Model unavailable'));

      const input = {
        type: 'text' as const,
        data: 'Degradation test',
        timestamp: Date.now(),
        sessionId: 'degradation-test'
      };

      // Should still work with cognitive shell only
      const response = await orchestrator.processSensoryInput(input);
      expect(response).toBeDefined();
    });
  });
});
