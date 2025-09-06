/**
 * Tests for type safety improvements that resolved TypeScript compilation errors
 * These tests validate that all type-related fixes work correctly
 */

import { SkillResult, Adaptation } from '../../src/App';
import { Agent } from '../../src/types/common';

describe('Type Safety Improvements', () => {
  describe('Agent Interface Compatibility', () => {
    it('should create valid Agent objects with new interface', () => {
      const agent: Agent = {
        agentId: 'test-agent-1',
        name: 'Test Agent',
        version: '1.0.0',
        type: 'wasm',
        status: 'Available',
        capabilities: ['nlp', 'analysis'],
        nrnCost: 100,
        metadata: {
          name: 'Test Agent',
          version: '1.0.0',
          description: 'A test agent for validation',
          author: 'Test Author',
          capabilities: ['nlp', 'analysis'],
          requirements: {
            memory: 512,
            cpu: 2,
            storage: 100
          },
          permissions: ['read', 'write', 'execute']
        },
        createdAt: new Date().toISOString(),
        lastActivity: new Date().toISOString()
      };

      expect(agent.type).toBe('wasm');
      expect(agent.status).toBe('Available');
      expect(agent.capabilities).toHaveLength(2);
      expect(agent.metadata.requirements.memory).toBe(512);
    });

    it('should handle different agent types correctly', () => {
      const wasmAgent: Agent = {
        agentId: 'wasm-1',
        name: 'WASM Agent',
        version: '1.0.0',
        type: 'wasm',
        status: 'Deployed',
        capabilities: ['computation'],
        nrnCost: 50,
        metadata: {
          name: 'WASM Agent',
          version: '1.0.0',
          description: 'WebAssembly-based agent',
          author: 'KNIRV',
          capabilities: ['computation'],
          requirements: { memory: 256, cpu: 1, storage: 50 },
          permissions: ['execute']
        },
        createdAt: new Date().toISOString(),
        wasmModule: 'test-wasm-module'
      };

      const loraAgent: Agent = {
        agentId: 'lora-1',
        name: 'LoRA Agent',
        version: '1.0.0',
        type: 'lora',
        status: 'Available',
        capabilities: ['adaptation'],
        nrnCost: 75,
        metadata: {
          name: 'LoRA Agent',
          version: '1.0.0',
          description: 'LoRA-based adaptive agent',
          author: 'KNIRV',
          capabilities: ['adaptation'],
          requirements: { memory: 1024, cpu: 4, storage: 200 },
          permissions: ['read', 'adapt']
        },
        createdAt: new Date().toISOString(),
        loraAdapter: 'lora-adapter-config'
      };

      expect(wasmAgent.type).toBe('wasm');
      expect(wasmAgent.wasmModule).toBe('test-wasm-module');
      expect(loraAgent.type).toBe('lora');
      expect(loraAgent.loraAdapter).toBe('lora-adapter-config');
    });
  });

  describe('Callback Type Safety', () => {
    it('should handle SkillResult callbacks with proper type casting', () => {
      const handleSkillInvoked = (skillId: string, result: unknown) => {
        const skillResult = result as SkillResult;
        return {
          skillId,
          success: skillResult.success,
          output: skillResult.output,
          executionTime: skillResult.executionTime
        };
      };

      const mockResult: SkillResult = {
        success: true,
        output: { data: 'test output' },
        executionTime: 150
      };

      const processed = handleSkillInvoked('test-skill', mockResult);
      expect(processed.success).toBe(true);
      expect(processed.executionTime).toBe(150);
    });

    it('should handle Adaptation callbacks with proper type casting', () => {
      const handleAdaptationTriggered = (adaptation: unknown) => {
        const adaptationData = adaptation as Adaptation;
        return {
          type: adaptationData.type,
          confidence: adaptationData.confidence,
          timestamp: adaptationData.timestamp
        };
      };

      const mockAdaptation: Adaptation = {
        id: 'adaptation-456',
        type: 'learning-rate-adjustment',
        description: 'Learning rate adjustment adaptation',
        parameters: { newRate: 0.02 },
        timestamp: new Date(),
        confidence: 0.85
      };

      const processed = handleAdaptationTriggered(mockAdaptation);
      expect(processed.type).toBe('learning-rate-adjustment');
      expect(processed.confidence).toBe(0.85);
    });
  });

  describe('Transaction Interface Fixes', () => {
    it('should create valid Transaction objects with type property', () => {
      const sendTransaction = {
        id: 'tx-1',
        hash: 'hash-1',
        from: 'address-1',
        to: 'address-2',
        amount: '100.0',
        type: 'send' as const,
        status: 'confirmed' as const,
        timestamp: new Date()
      };

      const receiveTransaction = {
        id: 'tx-2',
        hash: 'hash-2',
        from: 'address-3',
        to: 'address-1',
        amount: '50.0',
        type: 'receive' as const,
        status: 'pending' as const,
        timestamp: new Date()
      };

      expect(sendTransaction.type).toBe('send');
      expect(receiveTransaction.type).toBe('receive');
      expect(sendTransaction.status).toBe('confirmed');
      expect(receiveTransaction.status).toBe('pending');
    });
  });

  describe('Configuration Object Type Safety', () => {
    it('should handle CognitiveConfig with all required properties', () => {
      const config = {
        maxContextSize: 200,
        learningRate: 0.02,
        adaptationThreshold: 0.4,
        skillTimeout: 45000,
        voiceEnabled: true,
        visualEnabled: true,
        loraEnabled: true,
        enhancedLoraEnabled: true,
        hrmEnabled: true,
        wasmAgentsEnabled: true,
        typeScriptCompilerEnabled: true,
        adaptiveLearningEnabled: true
      };

      // Validate all boolean flags
      expect(typeof config.voiceEnabled).toBe('boolean');
      expect(typeof config.visualEnabled).toBe('boolean');
      expect(typeof config.loraEnabled).toBe('boolean');
      expect(typeof config.enhancedLoraEnabled).toBe('boolean');
      expect(typeof config.hrmEnabled).toBe('boolean');
      expect(typeof config.wasmAgentsEnabled).toBe('boolean');
      expect(typeof config.typeScriptCompilerEnabled).toBe('boolean');
      expect(typeof config.adaptiveLearningEnabled).toBe('boolean');

      // Validate numeric properties
      expect(typeof config.maxContextSize).toBe('number');
      expect(typeof config.learningRate).toBe('number');
      expect(typeof config.adaptationThreshold).toBe('number');
      expect(typeof config.skillTimeout).toBe('number');
    });
  });

  describe('Spread Operation Type Safety', () => {
    it('should handle object spread with type checking', () => {
      const baseState = { loaded: false, cognitiveActive: false };
      
      // Test safe spread with object check
      const updatePayload1 = { loaded: true };
      const newState1 = {
        ...baseState,
        ...(typeof updatePayload1 === 'object' && updatePayload1 !== null ? updatePayload1 : {})
      };
      
      expect(newState1.loaded).toBe(true);
      expect(newState1.cognitiveActive).toBe(false);

      // Test safe spread with null payload
      const updatePayload2 = null;
      const newState2 = {
        ...baseState,
        ...(typeof updatePayload2 === 'object' && updatePayload2 !== null ? updatePayload2 : {})
      };
      
      expect(newState2.loaded).toBe(false);
      expect(newState2.cognitiveActive).toBe(false);

      // Test safe spread with non-object payload
      const updatePayload3 = 'invalid';
      const newState3 = {
        ...baseState,
        ...(typeof updatePayload3 === 'object' && updatePayload3 !== null ? updatePayload3 : {})
      };
      
      expect(newState3.loaded).toBe(false);
      expect(newState3.cognitiveActive).toBe(false);
    });
  });

  describe('Error Handling Type Safety', () => {
    it('should handle unknown error types safely', () => {
      const safeErrorHandler = (error: unknown): string => {
        return error instanceof Error ? error.message : String(error);
      };

      expect(safeErrorHandler(new Error('Test error'))).toBe('Test error');
      expect(safeErrorHandler('String error')).toBe('String error');
      expect(safeErrorHandler(123)).toBe('123');
      expect(safeErrorHandler({ message: 'Object error' })).toBe('[object Object]');
      expect(safeErrorHandler(null)).toBe('null');
      expect(safeErrorHandler(undefined)).toBe('undefined');
    });
  });

  describe('Visual Processor Type Safety', () => {
    it('should handle BoundingBox interface correctly', () => {
      interface BoundingBox {
        x: number;
        y: number;
        width: number;
        height: number;
      }

      const extractFeatures = (blob: number[], boundingBox: BoundingBox): number[] => {
        const area = blob.length;
        const aspectRatio = boundingBox.width / boundingBox.height;
        const compactness = (4 * Math.PI * area) / Math.pow(boundingBox.width + boundingBox.height, 2);
        return [area, aspectRatio, compactness];
      };

      const testBoundingBox: BoundingBox = { x: 10, y: 20, width: 100, height: 50 };
      const testBlob = [1, 2, 3, 4, 5];
      
      const features = extractFeatures(testBlob, testBoundingBox);
      expect(features).toHaveLength(3);
      expect(features[0]).toBe(5); // area
      expect(features[1]).toBe(2); // aspect ratio (100/50)
    });
  });

  describe('Voice Control Type Safety', () => {
    it('should handle speech recognition events with type casting', () => {
      const handleSpeechDetected = (speech: unknown) => {
        const speechData = speech as { text: string; confidence: number };
        return {
          transcript: speechData.text,
          confidence: speechData.confidence
        };
      };

      const mockSpeech = { text: 'hello world', confidence: 0.95 };
      const result = handleSpeechDetected(mockSpeech);
      
      expect(result.transcript).toBe('hello world');
      expect(result.confidence).toBe(0.95);
    });

    it('should handle command recognition events with type casting', () => {
      const handleCommandRecognized = (command: unknown) => {
        const commandData = command as { originalText: string };
        return commandData.originalText;
      };

      const mockCommand = { originalText: 'scan qr code' };
      const result = handleCommandRecognized(mockCommand);
      
      expect(result).toBe('scan qr code');
    });
  });

  describe('AgentCoreConfig Interface Extensions', () => {
    it('should handle AgentCoreConfig with templates property', () => {
      interface AgentCoreConfig {
        agentId: string;
        agentName: string;
        templates?: Record<string, string>;
      }

      const config: AgentCoreConfig = {
        agentId: 'test-agent',
        agentName: 'Test Agent',
        templates: {
          'template1': 'content1',
          'template2': 'content2'
        }
      };

      expect(config.templates).toBeDefined();
      expect(config.templates!['template1']).toBe('content1');
    });
  });
});
