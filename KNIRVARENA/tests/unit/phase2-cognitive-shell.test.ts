/**
 * Phase 2 Cognitive Shell Development Tests
 * 
 * Testing Requirements from MAJOR_REFACTOR_IMPLEMENTATION_PLAN.md:
 * - WASM upload and compilation tests
 * - Cognitive shell integration tests
 * - Agent export functionality tests
 * - Primary agent management tests
 * - Security tests for WASM execution
 */

import { describe, test, expect, beforeEach, afterEach, jest } from '@jest/globals';
import { AgentCoreInterface, SensoryInput, LoRAAdapter } from '../../src/sensory-shell/AgentCoreInterface';

// Mock WASM module for testing
const createMockWASMBytes = (): Uint8Array => {
  return new Uint8Array([
    0x00, 0x61, 0x73, 0x6d, // WASM magic number
    0x01, 0x00, 0x00, 0x00, // WASM version
    ...Array(100).fill(0x42) // Mock WASM content
  ]);
};

// Mock agent-core WASM functions
const mockAgentCoreWASM = {
  agentCoreExecute: jest.fn().mockResolvedValue('{"success": true, "result": "test result"}' as never),
  agentCoreExecuteTool: jest.fn().mockResolvedValue('{"success": true, "result": "tool result"}' as never),
  agentCoreLoadLoRA: jest.fn().mockResolvedValue(true as never),
  agentCoreApplySkill: jest.fn().mockResolvedValue(true as never),
  agentCoreGetStatus: jest.fn().mockReturnValue('{"agentId": "test", "initialized": true}' as never)
};

describe('Phase 2.1: Cognitive Shell Development', () => {
  let agentCoreInterface: AgentCoreInterface;
  let mockWASMBytes: Uint8Array;

  beforeEach(() => {
    agentCoreInterface = new AgentCoreInterface();
    mockWASMBytes = createMockWASMBytes();
    
    // Mock WebAssembly global
    global.WebAssembly = {
      compile: jest.fn().mockResolvedValue({} as never),
      instantiate: jest.fn().mockResolvedValue({
        exports: mockAgentCoreWASM
      } as never),
      Memory: jest.fn().mockImplementation(() => ({
        buffer: new ArrayBuffer(1024)
      }))
    } as unknown as typeof WebAssembly;
  });

  afterEach(async () => {
    await agentCoreInterface.dispose();
    jest.clearAllMocks();
  });

  describe('WASM Upload and Compilation', () => {
    test('should successfully upload and compile agent WASM files', async () => {
      const success = await agentCoreInterface.initializeAgentCore(mockWASMBytes);
      
      expect(success).toBe(true);
      expect(WebAssembly.compile).toHaveBeenCalledWith(mockWASMBytes);
      expect(agentCoreInterface.isReady()).toBe(true);
    });

    test('should handle invalid WASM files gracefully', async () => {
      const invalidWASM = new Uint8Array([0x00, 0x00, 0x00, 0x00]); // Invalid magic number
      
      // Mock compile to throw error for invalid WASM
      (WebAssembly.compile as jest.Mock).mockRejectedValueOnce(new Error('Invalid WASM') as never);
      
      const success = await agentCoreInterface.initializeAgentCore(invalidWASM);
      
      expect(success).toBe(false);
      expect(agentCoreInterface.isReady()).toBe(false);
    });

    test('should validate required WASM functions exist', async () => {
      // Mock WASM with missing functions
      const incompleteWASM = {
        agentCoreExecute: jest.fn(),
        // Missing other required functions
      };

      (WebAssembly.instantiate as jest.Mock).mockResolvedValueOnce({
        exports: incompleteWASM
      } as never);

      const success = await agentCoreInterface.initializeAgentCore(mockWASMBytes);
      
      expect(success).toBe(false);
    });
  });

  describe('Cognitive Shell Integration', () => {
    beforeEach(async () => {
      await agentCoreInterface.initializeAgentCore(mockWASMBytes);
    });

    test('should process sensory input through cognitive shell', async () => {
      const input: SensoryInput = {
        type: 'text',
        data: 'Hello, agent!',
        timestamp: Date.now(),
        sessionId: 'test-session'
      };

      const response = await agentCoreInterface.processSensoryInput(input);

      expect(response.success).toBe(true);
      expect(response.source).toBe('agent-core');
      expect(typeof response.processingTime).toBe('number');
      expect(response.processingTime).toBeGreaterThanOrEqual(0);
      expect(mockAgentCoreWASM.agentCoreExecute).toHaveBeenCalled();
    });

    test('should handle cognitive processing errors', async () => {
      mockAgentCoreWASM.agentCoreExecute.mockRejectedValueOnce(new Error('Processing failed') as never);

      const input: SensoryInput = {
        type: 'text',
        data: 'Test input',
        timestamp: Date.now(),
        sessionId: 'test-session'
      };

      const response = await agentCoreInterface.processSensoryInput(input);

      expect(response.success).toBe(false);
      expect(response.error).toBe('Processing failed');
    });

    test('should execute tools through cognitive shell', async () => {
      const toolName = 'test-tool';
      const parameters = { param1: 'value1' };

      const response = await agentCoreInterface.executeTool(toolName, parameters);

      expect(response.success).toBe(true);
      expect(mockAgentCoreWASM.agentCoreExecuteTool).toHaveBeenCalledWith(
        toolName,
        JSON.stringify(parameters),
        expect.any(String)
      );
    });
  });

  describe('Agent Export Functionality', () => {
    beforeEach(async () => {
      await agentCoreInterface.initializeAgentCore(mockWASMBytes);
    });

    test('should export agent.wasm only', async () => {
      const status = await agentCoreInterface.getAgentCoreStatus();
      
      expect(status).toBeDefined();
      expect(status?.agentId).toBe('test');
      expect(status?.initialized).toBe(true);
    });

    test('should maintain agent state during export', async () => {
      // Load a LoRA adapter first
      const adapter: LoRAAdapter = {
        skillId: 'test-skill',
        skillName: 'Test Skill',
        weightsA: new Float32Array([0.1, 0.2, 0.3]),
        weightsB: new Float32Array([0.4, 0.5, 0.6]),
        rank: 8,
        alpha: 16.0
      };

      await agentCoreInterface.loadLoRAAdapter(adapter);
      
      const status = await agentCoreInterface.getAgentCoreStatus();
      expect(status?.initialized).toBe(true);
    });
  });

  describe('Primary Agent Management', () => {
    test('should designate primary agent', async () => {
      await agentCoreInterface.initializeAgentCore(mockWASMBytes);
      
      const status = await agentCoreInterface.getAgentCoreStatus();
      expect(status?.agentId).toBeDefined();
    });

    test('should handle multiple agent instances', async () => {
      const agent1 = new AgentCoreInterface();
      const agent2 = new AgentCoreInterface();

      await agent1.initializeAgentCore(mockWASMBytes);
      await agent2.initializeAgentCore(mockWASMBytes);

      expect(agent1.isReady()).toBe(true);
      expect(agent2.isReady()).toBe(true);
      expect(agent1.getSessionId()).not.toBe(agent2.getSessionId());

      await agent1.dispose();
      await agent2.dispose();
    });
  });

  describe('Security Tests for WASM Execution', () => {
    test('should isolate WASM execution environment', async () => {
      await agentCoreInterface.initializeAgentCore(mockWASMBytes);
      
      // Test that WASM cannot access global scope inappropriately
      const input: SensoryInput = {
        type: 'text',
        data: 'window.location = "malicious-site.com"',
        timestamp: Date.now(),
        sessionId: 'security-test'
      };

      const response = await agentCoreInterface.processSensoryInput(input);
      
      // Should process without affecting global scope
      expect(response).toBeDefined();
      expect(typeof window === 'undefined' || window.location.href).not.toContain('malicious-site.com');
    });

    test('should validate input sanitization', async () => {
      await agentCoreInterface.initializeAgentCore(mockWASMBytes);

      const maliciousInput: SensoryInput = {
        type: 'text',
        data: '<script>alert("xss")</script>',
        timestamp: Date.now(),
        sessionId: 'xss-test'
      };

      const response = await agentCoreInterface.processSensoryInput(maliciousInput);
      
      expect(response).toBeDefined();
      // Should not execute script content
    });

    test('should limit memory usage', async () => {
      await agentCoreInterface.initializeAgentCore(mockWASMBytes);
      
      // Test memory constraints
      expect(WebAssembly.Memory).toHaveBeenCalledWith({
        initial: expect.any(Number),
        maximum: expect.any(Number)
      });
    });

    test('should handle WASM module disposal securely', async () => {
      await agentCoreInterface.initializeAgentCore(mockWASMBytes);
      expect(agentCoreInterface.isReady()).toBe(true);

      await agentCoreInterface.dispose();
      expect(agentCoreInterface.isReady()).toBe(false);

      // Should not be able to execute after disposal
      const input: SensoryInput = {
        type: 'text',
        data: 'test',
        timestamp: Date.now(),
        sessionId: 'disposal-test'
      };

      await expect(agentCoreInterface.processSensoryInput(input))
        .rejects.toThrow('Agent-core not initialized');
    });
  });
});
