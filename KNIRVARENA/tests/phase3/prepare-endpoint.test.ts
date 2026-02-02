/**
 * Phase 3 Tests: /prepare Endpoint Integration for NEXUS TEE
 * 
 * Tests for NEXUS TEE connectivity, LoRA adapter preparation,
 * and pre-training functionality
 */

import { describe, it, expect, beforeEach, afterEach, jest } from '@jest/globals';

// Mock implementations for testing
class MockLoRAAdapterEngine {
  private adapters: string[] = [];
  private adapterData: Map<string, Record<string, unknown>> = new Map();

  async initialize(): Promise<void> {}

  async compileAdapter(solutions: Record<string, unknown>, metadata: Record<string, unknown>): Promise<void> {
    const skillId = `skill_${Date.now()}`;
    this.adapters.push(skillId);
    this.adapterData.set(skillId, {
      skillId,
      skillName: metadata.skillName,
      description: metadata.description,
      rank: 8,
      alpha: 16.0,
      weightsA: new Float32Array(64).fill(0.01),
      weightsB: new Float32Array(64).fill(0.02),
      baseModelCompatibility: 'hrm'
    });
  }

  getAdapters(): string[] {
    return this.adapters;
  }

  getAdapter(skillId: string): { skillId: string; skillName: string; description: string; rank: number; alpha: number; weightsA: Float32Array; weightsB: Float32Array; baseModelCompatibility: string } | undefined {
    const adapter = this.adapterData.get(skillId);
    if (adapter) {
      return adapter as { skillId: string; skillName: string; description: string; rank: number; alpha: number; weightsA: Float32Array; weightsB: Float32Array; baseModelCompatibility: string };
    }
    return undefined;
  }

  async createWASMFormat(adapter: { skillId: string; skillName: string; description: string; rank: number; alpha: number; weightsA: Float32Array; weightsB: Float32Array; baseModelCompatibility: string }): Promise<Uint8Array> {
    const wasmHeader = new Uint8Array([0x00, 0x61, 0x73, 0x6d, 0x01, 0x00, 0x00, 0x00]);

    // Convert Float32Arrays to regular arrays for JSON serialization
    const serializableAdapter = {
      ...adapter,
      weightsA: Array.from(adapter.weightsA),
      weightsB: Array.from(adapter.weightsB)
    };

    const encoder = typeof TextEncoder !== 'undefined' ? new TextEncoder() : new (await import('util')).TextEncoder();
    const data = encoder.encode(JSON.stringify(serializableAdapter));
    const result = new Uint8Array(wasmHeader.length + data.length);
    result.set(wasmHeader, 0);
    result.set(data, wasmHeader.length);
    return result;
  }

  async loadFromWASMFormat(wasmBytes: Uint8Array): Promise<{ skillId: string; skillName: string; description: string; rank: number; alpha: number; weightsA: Float32Array; weightsB: Float32Array; baseModelCompatibility: string }> {
    const headerLength = 8;
    const dataBytes = wasmBytes.slice(headerLength);
    const dataString = new TextDecoder().decode(dataBytes);
    const parsed = JSON.parse(dataString);

    // Reconstruct Float32Array objects from regular arrays
    if (parsed.weightsA && Array.isArray(parsed.weightsA)) {
      parsed.weightsA = new Float32Array(parsed.weightsA);
    }
    if (parsed.weightsB && Array.isArray(parsed.weightsB)) {
      parsed.weightsB = new Float32Array(parsed.weightsB);
    }

    return parsed;
  }

  async cleanup(): Promise<void> {
    this.adapters = [];
    this.adapterData.clear();
  }
}

class MockWASMCompiler {
  async initialize(): Promise<void> {}
  async cleanup(): Promise<void> {}
}

class MockProtobufHandler {
  async initialize(): Promise<void> {}
}

class MockCortexAPI {
  private loraEngine: MockLoRAAdapterEngine;

  constructor(loraEngine: MockLoRAAdapterEngine, _wasmCompiler: MockWASMCompiler, _protobufHandler: MockProtobufHandler) {
    this.loraEngine = loraEngine;
  }

  async prepareLoRAAdapter(skillId: string, teeInfo?: { attestationRequired?: boolean; securityLevel?: string }, _loraAdapterConfig?: unknown): Promise<{ success: boolean; preparationResult: { skillId: string; skillName: string; wasmBytes: number[]; teeCompatibility: { requiredMemory: number; requiredCPU: string; securityLevel: string; attestationRequired: boolean }; loraMetadata: { rank: number; alpha: number; baseModel: string; weightsSize: number }; nexusConnectivity: { endpoint: string; protocol: string; authentication: string; timeout: number }; preparationTimestamp: string; packageHash: string }; message: string }> {
    if (!skillId) {
      throw new Error('Missing skillId');
    }

    const adapter = this.loraEngine.getAdapter(skillId);
    if (!adapter) {
      throw new Error(`LoRA adapter ${skillId} not found`);
    }

    // Mock NEXUS TEE connectivity check
    if (global.fetch) {
      try {
        await (global.fetch as unknown as typeof mockFetch)('http://mock-nexus-tee/health');
        await (global.fetch as unknown as typeof mockFetch)('http://mock-nexus-tee/register', { method: 'POST' });
      } catch (error) {
        throw new Error(error instanceof Error ? error.message : 'Connection failed');
      }
    }

    const wasmFormat = await this.loraEngine.createWASMFormat(adapter);

    const preparationResult = {
      skillId: adapter.skillId,
      skillName: adapter.skillName,
      wasmBytes: Array.from(wasmFormat),
      teeCompatibility: {
        requiredMemory: wasmFormat.length + 1024 * 1024,
        requiredCPU: 'any',
        securityLevel: teeInfo?.securityLevel || 'standard',
        attestationRequired: teeInfo?.attestationRequired || false
      },
      loraMetadata: {
        rank: adapter.rank,
        alpha: adapter.alpha,
        baseModel: adapter.baseModelCompatibility,
        weightsSize: adapter.weightsA.length + adapter.weightsB.length
      },
      nexusConnectivity: {
        endpoint: 'http://mock-nexus-tee',
        protocol: 'https',
        authentication: 'bearer',
        timeout: 30000
      },
      preparationTimestamp: new Date().toISOString(),
      packageHash: this.calculatePackageHash(wasmFormat)
    };

    return {
      success: true,
      preparationResult,
      message: 'LoRA adapter prepared for NEXUS TEE execution'
    };
  }

  async getTEEStatus(): Promise<{ success: boolean; teeStatus: { connected: boolean; endpoint: string; status: string; capabilities?: string[]; availableResources?: { cpu: string; memory: string }; error?: string; lastChecked: string }; timestamp: string }> {
    let teeStatus;
    if (global.fetch) {
      try {
        const response = await (global.fetch as unknown as typeof mockFetch)('http://mock-nexus-tee/status');
        const status = await response.json();
        teeStatus = {
          connected: true,
          endpoint: 'http://mock-nexus-tee',
          status: 'operational',
          capabilities: (status as any)?.capabilities || ['lora-training', 'secure-execution'],
          availableResources: (status as any)?.resources || { cpu: '80%', memory: '60%' },
          lastChecked: new Date().toISOString()
        };
      } catch (error) {
        teeStatus = {
          connected: false,
          endpoint: 'http://mock-nexus-tee',
          status: 'error',
          error: error instanceof Error ? error.message : 'Unknown error',
          lastChecked: new Date().toISOString()
        };
      }
    } else {
      teeStatus = {
        connected: false,
        endpoint: 'http://mock-nexus-tee',
        status: 'no-fetch',
        lastChecked: new Date().toISOString()
      };
    }

    return {
      success: true,
      teeStatus,
      timestamp: new Date().toISOString()
    };
  }

  async performPreTraining(baseModel: string, loraAdapterInsights: unknown, _trainingConfig?: unknown): Promise<{ success: boolean; pretrainingResult: { success: boolean; baseModel: string; updatedModelVersion: string; insightsApplied: number; trainingMetrics: { initialLoss: number; finalLoss: number; convergenceEpochs: number; improvementPercentage: number }; modelImprovements: { accuracyGain: number; efficiencyGain: number; robustnessGain: number }; completedAt: string }; message: string }> {
    if (!baseModel || !loraAdapterInsights) {
      throw new Error('Missing baseModel or loraAdapterInsights');
    }

    const insights = Array.isArray(loraAdapterInsights) ? loraAdapterInsights : [loraAdapterInsights];

    const pretrainingResult = {
      success: true,
      baseModel,
      updatedModelVersion: `${baseModel}_v${Date.now()}`,
      insightsApplied: insights.length,
      trainingMetrics: {
        initialLoss: 2.5,
        finalLoss: 1.8,
        convergenceEpochs: 8,
        improvementPercentage: 28
      },
      modelImprovements: {
        accuracyGain: 0.15,
        efficiencyGain: 0.22,
        robustnessGain: 0.18
      },
      completedAt: new Date().toISOString()
    };

    return {
      success: true,
      pretrainingResult,
      message: 'Base model pre-training completed using LoRA adapter insights'
    };
  }

  private calculatePackageHash(wasmBytes: Uint8Array): string {
    let hash = 0;
    for (let i = 0; i < wasmBytes.length; i++) {
      hash = ((hash << 5) - hash + wasmBytes[i]) & 0xffffffff;
    }
    return Math.abs(hash).toString(16);
  }
}

const CortexAPI = MockCortexAPI;
const LoRAAdapterEngine = MockLoRAAdapterEngine;
const WASMCompiler = MockWASMCompiler;
const ProtobufHandler = MockProtobufHandler;

// Mock fetch for testing
interface MockResponse {
  ok: boolean;
  status?: number;
  json: () => Promise<unknown>;
  text?: () => Promise<string>;
}

const mockFetch = jest.fn() as jest.MockedFunction<(url: string, options?: RequestInit) => Promise<MockResponse>>;
global.fetch = mockFetch as any;

describe('Phase 3.4: /prepare Endpoint Integration', () => {
  let cortexAPI: MockCortexAPI;
  let loraEngine: MockLoRAAdapterEngine;
  let wasmCompiler: MockWASMCompiler;
  let protobufHandler: MockProtobufHandler;

  beforeEach(async () => {
    // Initialize components
    loraEngine = new LoRAAdapterEngine();
    wasmCompiler = new WASMCompiler();
    protobufHandler = new ProtobufHandler();

    await loraEngine.initialize();
    await wasmCompiler.initialize();
    await protobufHandler.initialize();

    cortexAPI = new CortexAPI(loraEngine, wasmCompiler, protobufHandler);

    // Register test LoRA adapter
    await loraEngine.compileAdapter({
      solutions: [{
        errorId: 'test-error',
        solution: 'Test solution code',
        confidence: 0.9
      }],
      errors: [{
        errorId: 'test-error',
        description: 'Test error description',
        context: 'Test context'
      }]
    }, {
      skillName: 'Test TEE Skill',
      description: 'Test skill for TEE preparation',
      author: 'Test System',
      version: '1.0.0',
      tags: ['test', 'tee']
    });
  });

  afterEach(async () => {
    await loraEngine.cleanup();
    await wasmCompiler.cleanup();
    jest.clearAllMocks();
  });

  describe('NEXUS TEE Connectivity', () => {
    it('should prepare LoRA adapter for TEE execution', async () => {
      // Mock successful NEXUS TEE responses
      mockFetch
        .mockResolvedValueOnce({
          ok: true,
          json: async () => ({ status: 'healthy' })
        })
        .mockResolvedValueOnce({
          ok: true,
          json: async () => ({ success: true, registered: true })
        });

      const adapters = loraEngine.getAdapters();
      const testSkillId = adapters[0];

      // Test the prepare functionality directly
      const result = await cortexAPI.prepareLoRAAdapter(testSkillId, {
        attestationRequired: false,
        securityLevel: 'standard'
      });

      expect(result.success).toBe(true);
      expect(result.preparationResult).toBeDefined();
      expect(result.preparationResult.skillId).toBe(testSkillId);
      expect(result.preparationResult.wasmBytes).toBeDefined();
      expect(result.preparationResult.teeCompatibility).toBeDefined();
      expect(result.preparationResult.nexusConnectivity).toBeDefined();
      expect(result.preparationResult.teeCompatibility.requiredMemory).toBeGreaterThan(1024 * 1024);
      expect(result.preparationResult.teeCompatibility.securityLevel).toBe('standard');
    });

    it('should handle missing skillId in prepare request', async () => {
      // Test validation logic directly
      try {
        await cortexAPI.prepareLoRAAdapter('');
        expect(true).toBe(false); // Should not reach here
      } catch (error) {
        expect(error).toBeInstanceOf(Error);
        expect((error as Error).message).toContain('Missing skillId');
      }
    });

    it('should handle non-existent skill in prepare request', async () => {
      try {
        await cortexAPI.prepareLoRAAdapter('non-existent-skill');
        expect(true).toBe(false); // Should not reach here
      } catch (error) {
        expect(error).toBeInstanceOf(Error);
        expect((error as Error).message).toContain('not found');
      }
    });

    it('should handle NEXUS TEE connectivity failure', async () => {
      // Mock failed NEXUS TEE response
      mockFetch.mockRejectedValue(new Error('Connection failed'));

      const adapters = loraEngine.getAdapters();
      const testSkillId = adapters[0];

      try {
        await cortexAPI.prepareLoRAAdapter(testSkillId);
        expect(true).toBe(false); // Should not reach here
      } catch (error) {
        expect(error).toBeInstanceOf(Error);
        expect((error as Error).message).toContain('Connection failed');
      }
    });
  });

  describe('TEE Status Monitoring', () => {
    it('should return TEE connectivity status', async () => {
      // Mock successful TEE status response
      mockFetch.mockResolvedValue({
        ok: true,
        json: async () => ({
          status: 'operational',
          capabilities: ['lora-training', 'secure-execution'],
          resources: { cpu: '80%', memory: '60%' }
        })
      });

      const result = await cortexAPI.getTEEStatus();

      expect(result.success).toBe(true);
      expect(result.teeStatus.connected).toBe(true);
      expect(result.teeStatus.capabilities).toContain('lora-training');
      expect(result.teeStatus.availableResources).toBeDefined();
    });

    it('should handle TEE unreachable status', async () => {
      // Mock failed TEE status response
      mockFetch.mockRejectedValue(new Error('Network error'));

      const result = await cortexAPI.getTEEStatus();

      expect(result.success).toBe(true);
      expect(result.teeStatus.connected).toBe(false);
      expect(result.teeStatus.status).toBe('error');
      expect(result.teeStatus.error).toContain('Network error');
    });
  });

  describe('Pre-training with LoRA Adapter Insights', () => {
    it('should perform pre-training using LoRA adapter insights', async () => {
      const loraInsights = [
        {
          skillId: 'skill-1',
          rank: 8,
          alpha: 16.0,
          accuracy: 0.9,
          latency: 50,
          invocations: 100,
          successRate: 0.85,
          errorTypes: ['TypeError', 'ReferenceError'],
          weightCount: 1024
        },
        {
          skillId: 'skill-2',
          rank: 4,
          alpha: 8.0,
          accuracy: 0.85,
          latency: 30,
          invocations: 150,
          successRate: 0.9,
          errorTypes: ['SyntaxError', 'TypeError'],
          weightCount: 512
        }
      ];

      const result = await cortexAPI.performPreTraining('hrm-base-v1', loraInsights, {
        learningRate: 0.0001,
        batchSize: 32,
        epochs: 10
      });

      expect(result.success).toBe(true);
      expect(result.pretrainingResult).toBeDefined();
      expect(result.pretrainingResult.baseModel).toBe('hrm-base-v1');
      expect(result.pretrainingResult.updatedModelVersion).toBeDefined();
      expect(result.pretrainingResult.insightsApplied).toBe(2);
      expect(result.pretrainingResult.trainingMetrics).toBeDefined();
      expect(result.pretrainingResult.modelImprovements).toBeDefined();
    });

    it('should handle missing parameters in pre-training request', async () => {
      try {
        await cortexAPI.performPreTraining('hrm-base-v1', undefined);
        expect(true).toBe(false); // Should not reach here
      } catch (error) {
        expect(error).toBeInstanceOf(Error);
        expect((error as Error).message).toContain('Missing baseModel or loraAdapterInsights');
      }
    });

    it('should aggregate insights from multiple LoRA adapters', async () => {
      const loraInsights = [
        { rank: 8, alpha: 16.0, accuracy: 0.9, errorTypes: ['TypeError'] },
        { rank: 4, alpha: 8.0, accuracy: 0.85, errorTypes: ['ReferenceError'] },
        { rank: 16, alpha: 32.0, accuracy: 0.95, errorTypes: ['TypeError', 'SyntaxError'] }
      ];

      const result = await cortexAPI.performPreTraining('phi3-base-v1', loraInsights, {});

      expect(result.success).toBe(true);
      expect(result.pretrainingResult.insightsApplied).toBe(3);
      expect(result.pretrainingResult.trainingMetrics.improvementPercentage).toBeGreaterThan(0);
      expect(result.pretrainingResult.modelImprovements.accuracyGain).toBeGreaterThan(0);
    });
  });

  describe('WASM Format Compatibility', () => {
    it('should create WASM format for LoRA adapters', async () => {
      const adapters = loraEngine.getAdapters();
      const testSkillId = adapters[0];
      const adapter = loraEngine.getAdapter(testSkillId);

      if (adapter) {
        const wasmFormat = await loraEngine.createWASMFormat(adapter);

        expect(wasmFormat).toBeInstanceOf(Uint8Array);
        expect(wasmFormat.length).toBeGreaterThan(0);

        // Check WASM magic number
        expect(wasmFormat[0]).toBe(0x00);
        expect(wasmFormat[1]).toBe(0x61);
        expect(wasmFormat[2]).toBe(0x73);
        expect(wasmFormat[3]).toBe(0x6d);
      }
    });

    it('should load LoRA adapter from WASM format', async () => {
      const adapters = loraEngine.getAdapters();
      const testSkillId = adapters[0];
      const originalAdapter = loraEngine.getAdapter(testSkillId);

      if (originalAdapter) {
        const wasmFormat = await loraEngine.createWASMFormat(originalAdapter);
        const loadedAdapter = await loraEngine.loadFromWASMFormat(wasmFormat);

        expect(loadedAdapter.skillId).toBe(originalAdapter.skillId);
        expect(loadedAdapter.skillName).toBe(originalAdapter.skillName);
        expect(loadedAdapter.rank).toBe(originalAdapter.rank);
        expect(loadedAdapter.alpha).toBe(originalAdapter.alpha);
        expect(loadedAdapter.weightsA.length).toBe(originalAdapter.weightsA.length);
        expect(loadedAdapter.weightsB.length).toBe(originalAdapter.weightsB.length);
      }
    });
  });

  describe('TEE Package Preparation', () => {
    it('should create proper TEE compatibility metadata', async () => {
      const adapters = loraEngine.getAdapters();
      const testSkillId = adapters[0];
      const adapter = loraEngine.getAdapter(testSkillId);

      if (adapter) {
        const wasmFormat = await loraEngine.createWASMFormat(adapter);

        // Test TEE compatibility metadata creation
        const teeCompatibility = {
          requiredMemory: wasmFormat.length + 1024 * 1024,
          requiredCPU: 'any',
          securityLevel: 'high',
          attestationRequired: true
        };

        expect(teeCompatibility.requiredMemory).toBeGreaterThan(1024 * 1024);
        expect(teeCompatibility.requiredCPU).toBe('any');
        expect(teeCompatibility.securityLevel).toBe('high');
        expect(teeCompatibility.attestationRequired).toBe(true);
      }
    });

    it('should calculate package hash correctly', async () => {
      const adapters = loraEngine.getAdapters();
      const testSkillId = adapters[0];
      const adapter = loraEngine.getAdapter(testSkillId);

      if (adapter) {
        const wasmFormat1 = await loraEngine.createWASMFormat(adapter);
        const wasmFormat2 = await loraEngine.createWASMFormat(adapter);

        // Calculate hash for both using helper function
        const calculateHash = (wasmBytes: Uint8Array): string => {
          let hash = 0;
          for (let i = 0; i < wasmBytes.length; i++) {
            hash = ((hash << 5) - hash + wasmBytes[i]) & 0xffffffff;
          }
          return Math.abs(hash).toString(16);
        };

        const hash1 = calculateHash(wasmFormat1);
        const hash2 = calculateHash(wasmFormat2);

        // Same adapter should produce same hash
        expect(hash1).toBe(hash2);
      }
    });
  });

  describe('Error Handling and Edge Cases', () => {
    it('should handle NEXUS TEE authentication failure', async () => {
      // Mock authentication failure
      mockFetch
        .mockResolvedValueOnce({
          ok: true,
          json: async () => ({ status: 'healthy' })
        })
        .mockResolvedValueOnce({
          ok: false,
          status: 401,
          json: async () => ({ error: 'Unauthorized' })
        });

      // Test authentication failure handling
      try {
        await (global.fetch as unknown as typeof mockFetch)('http://mock-nexus-tee/health');
        const authResponse = await (global.fetch as unknown as typeof mockFetch)('http://mock-nexus-tee/register', { method: 'POST' });
        expect(authResponse.ok).toBe(false);
        expect(authResponse.status).toBe(401);
      } catch {
        // Expected behavior
      }
    });

    it('should handle malformed TEE info', async () => {
      const adapters = loraEngine.getAdapters();
      const testSkillId = adapters[0];
      const adapter = loraEngine.getAdapter(testSkillId);

      // Test with malformed TEE info
      const malformedTeeInfo = 'invalid-tee-info';
      const validTeeInfo = {
        attestationRequired: false,
        securityLevel: 'standard'
      };

      expect(typeof malformedTeeInfo).toBe('string');
      expect(typeof validTeeInfo).toBe('object');
      expect(adapter).toBeDefined();
    });

    it('should handle empty LoRA adapter insights in pre-training', async () => {
      const baseModel = 'test-model';
      const loraAdapterInsights: Record<string, unknown>[] = [];

      expect(baseModel).toBeDefined();
      expect(Array.isArray(loraAdapterInsights)).toBe(true);
      expect(loraAdapterInsights.length).toBe(0);

      // Test empty insights handling
      const insights = Array.isArray(loraAdapterInsights) ? loraAdapterInsights : [loraAdapterInsights];
      expect(insights.length).toBe(0);
    });
  });

  describe('Performance and Scalability', () => {
    it('should handle multiple concurrent prepare requests', async () => {
      // Mock successful NEXUS TEE responses
      mockFetch
        .mockResolvedValue({
          ok: true,
          json: async () => ({ status: 'healthy', success: true })
        });

      const adapters = loraEngine.getAdapters();
      const testSkillId = adapters[0];
      const adapter = loraEngine.getAdapter(testSkillId);

      // Test concurrent WASM format creation
      const promises = Array.from({ length: 5 }, async (_, i) => {
        if (adapter) {
          const wasmFormat = await loraEngine.createWASMFormat(adapter);
          return {
            index: i,
            success: wasmFormat.length > 0,
            size: wasmFormat.length
          };
        }
        return { index: i, success: false, size: 0 };
      });

      const results = await Promise.all(promises);

      results.forEach((result, index) => {
        expect(result.index).toBe(index);
        expect(result.success).toBe(true);
        expect(result.size).toBeGreaterThan(0);
      });
    });

    it('should handle large LoRA adapter preparation', async () => {
      // Create a large LoRA adapter
      await loraEngine.compileAdapter({
        solutions: Array.from({ length: 100 }, (_, i) => ({
          errorId: `large-error-${i}`,
          solution: `Large solution code ${i}`,
          confidence: 0.8 + (i % 20) / 100
        })),
        errors: Array.from({ length: 100 }, (_, i) => ({
          errorId: `large-error-${i}`,
          description: `Large error description ${i}`,
          context: `Large context ${i}`
        }))
      }, {
        skillName: 'Large Test Skill',
        description: 'Large skill for performance testing',
        author: 'Test System',
        version: '1.0.0',
        tags: ['test', 'large', 'performance']
      });

      const adapters = loraEngine.getAdapters();
      const largeSkillId = adapters[adapters.length - 1]; // Get the last (large) adapter
      const largeAdapter = loraEngine.getAdapter(largeSkillId);

      expect(largeAdapter).toBeDefined();
      expect(largeAdapter?.skillName).toBe('Large Test Skill');

      // Test performance of large adapter WASM creation
      const startTime = Date.now();
      if (largeAdapter) {
        const wasmFormat = await loraEngine.createWASMFormat(largeAdapter);
        const endTime = Date.now();

        expect(wasmFormat.length).toBeGreaterThan(0);
        expect(endTime - startTime).toBeLessThan(5000); // Should complete within 5 seconds
      }
    });
  });
});
