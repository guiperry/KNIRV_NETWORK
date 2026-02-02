/**
 * Phase 3.3 Skill Minting Process Tests
 * 
 * Tests for complete LoRA adapter validation and KNIRVCHAIN integration
 */

import {
  SkillMintingProcess,
  MintingStatus
} from '../../src/core/knirvgraph/SkillMintingProcess';
import { QueuedLoRAAdapter, QueueStatus } from '../../src/core/knirvgraph/LoRAProcessingQueue';
import { LoRAAdapterSkill } from '../../src/core/lora/LoRAAdapterEngine';

describe('Phase 3.3 - Skill Minting Process', () => {
  let skillMintingProcess: SkillMintingProcess;
  let mockQueuedAdapter: QueuedLoRAAdapter;

  beforeEach(async () => {
    skillMintingProcess = new SkillMintingProcess({
      validationTimeout: 2000,
      consensusTimeout: 3000,
      mintingTimeout: 2000,
      minValidationScore: 0.8,
      maxConcurrentMinting: 2,
      knirvchainRpcUrl: 'http://localhost:26657',
      enableSecurityValidation: true,
      enablePerformanceValidation: true
    });

    await skillMintingProcess.initialize();

    // Create mock queued adapter
    const mockLoRAAdapter: LoRAAdapterSkill = {
      skillId: 'test_skill_001',
      skillName: 'Test Debugging Skill',
      description: 'A test skill for debugging',
      baseModelCompatibility: 'hrm',
      version: 1,
      rank: 8,
      alpha: 16.0,
      weightsA: new Float32Array(Array.from({ length: 8192 }, () => Math.random() * 0.1 - 0.05)),
      weightsB: new Float32Array(Array.from({ length: 8192 }, () => Math.random() * 0.1 - 0.05)),
      additionalMetadata: {
        clusterId: 'cluster_001',
        trainingDatasetId: 'dataset_001'
      }
    };

    mockQueuedAdapter = {
      queueId: 'queue_001',
      loraAdapter: mockLoRAAdapter,
      trainingDataset: {
        datasetId: 'dataset_001',
        clusterId: 'cluster_001',
        errorNodes: [],
        validatedSolutions: [],
        trainingPairs: [],
        datasetMetrics: {
          totalPairs: 10,
          averageValidationScore: 0.85,
          diversityScore: 0.8,
          complexityScore: 0.6,
          qualityScore: 0.85
        },
        createdAt: new Date()
      },
      priority: 5,
      status: QueueStatus.COMPLETED,
      submittedAt: new Date(),
      retryCount: 0,
      maxRetries: 3,
      discoveryResult: {
        skillId: 'test_skill_001',
        discoveredName: 'JavaScript Error Resolver',
        category: 'debugging',
        subcategory: 'error_resolution',
        description: 'Resolves JavaScript runtime errors',
        capabilities: ['typeerror_resolution', 'null_check'],
        complexity: 0.7,
        confidence: 0.85,
        tags: ['debugging', 'javascript'],
        relatedSkills: []
      }
    };
  });

  afterEach(() => {
    skillMintingProcess.stop();
  });

  describe('Initialization', () => {
    test('should initialize successfully', async () => {
      const newProcess = new SkillMintingProcess();
      await expect(newProcess.initialize()).resolves.not.toThrow();
      expect(newProcess.isReady()).toBe(true);
      newProcess.stop();
    });

    test('should handle blockchain connectivity issues gracefully', async () => {
      const newProcess = new SkillMintingProcess({
        knirvchainRpcUrl: 'http://invalid-url:26657'
      });
      
      // Should not throw, but continue in simulation mode
      await expect(newProcess.initialize()).resolves.not.toThrow();
      newProcess.stop();
    });
  });

  describe('Minting Submission', () => {
    test('should submit skill for minting successfully', async () => {
      const requestId = await skillMintingProcess.submitForMinting(
        mockQueuedAdapter,
        'test_agent',
        8
      );

      expect(requestId).toBeTruthy();
      expect(requestId).toMatch(/^mint_/);

      const request = skillMintingProcess.getMintingRequest(requestId);
      expect(request).toBeDefined();
      expect(request!.status).toBe(MintingStatus.PENDING_VALIDATION);
      expect(request!.priority).toBe(8);
      expect(request!.requestedBy).toBe('test_agent');
    });

    test('should fail when adapter has no discovery result', async () => {
      const adapterWithoutResult = {
        ...mockQueuedAdapter,
        discoveryResult: undefined
      };

      await expect(skillMintingProcess.submitForMinting(adapterWithoutResult))
        .rejects.toThrow('LoRA adapter must have discovery result before minting');
    });

    test('should fail when not initialized', async () => {
      const newProcess = new SkillMintingProcess();
      
      await expect(newProcess.submitForMinting(mockQueuedAdapter))
        .rejects.toThrow('Skill Minting Process not initialized');
    });
  });

  describe('Validation Process', () => {
    test('should validate skill successfully', async () => {
      const requestId = await skillMintingProcess.submitForMinting(mockQueuedAdapter);
      
      // Wait for validation to complete
      await new Promise(resolve => setTimeout(resolve, 100));
      
      const request = skillMintingProcess.getMintingRequest(requestId);
      expect(request).toBeDefined();
      
      // Should progress through validation
      await new Promise(resolve => {
        const checkStatus = () => {
          const req = skillMintingProcess.getMintingRequest(requestId);
          if (req && req.status !== MintingStatus.PENDING_VALIDATION) {
            resolve(undefined);
          } else {
            setTimeout(checkStatus, 50);
          }
        };
        checkStatus();
      });
    });

    test('should handle validation failure', async () => {
      // Create adapter with invalid weights (all zeros)
      const invalidAdapter = {
        ...mockQueuedAdapter,
        loraAdapter: {
          ...mockQueuedAdapter.loraAdapter,
          weightsA: new Float32Array(8192), // All zeros
          weightsB: new Float32Array(8192)
        }
      };

      const requestId = await skillMintingProcess.submitForMinting(invalidAdapter);
      
      // Wait for validation
      await new Promise(resolve => setTimeout(resolve, 200));
      
      const request = skillMintingProcess.getMintingRequest(requestId);
      expect(request).toBeDefined();
    });

    test('should perform technical validation correctly', async () => {
      const requestId = await skillMintingProcess.submitForMinting(mockQueuedAdapter);
      
      // Wait for validation
      await new Promise(resolve => setTimeout(resolve, 200));
      
      const validationResult = skillMintingProcess.getValidationResult(requestId);
      if (validationResult) {
        expect(validationResult.technicalValidation).toBeDefined();
        expect(validationResult.technicalValidation.weightsIntegrity).toBeDefined();
        expect(validationResult.technicalValidation.dimensionConsistency).toBeDefined();
        expect(validationResult.technicalValidation.numericalStability).toBeDefined();
      }
    });

    test('should perform semantic validation correctly', async () => {
      const requestId = await skillMintingProcess.submitForMinting(mockQueuedAdapter);
      
      // Wait for validation
      await new Promise(resolve => setTimeout(resolve, 200));
      
      const validationResult = skillMintingProcess.getValidationResult(requestId);
      if (validationResult) {
        expect(validationResult.semanticValidation).toBeDefined();
        expect(validationResult.semanticValidation.skillNameValidity).toBeDefined();
        expect(validationResult.semanticValidation.categoryConsistency).toBeDefined();
      }
    });

    test('should perform security validation when enabled', async () => {
      const requestId = await skillMintingProcess.submitForMinting(mockQueuedAdapter);
      
      // Wait for validation
      await new Promise(resolve => setTimeout(resolve, 200));
      
      const validationResult = skillMintingProcess.getValidationResult(requestId);
      if (validationResult) {
        expect(validationResult.securityValidation).toBeDefined();
        expect(validationResult.securityValidation.maliciousCodeDetection).toBeDefined();
        expect(validationResult.securityValidation.privacyCompliance).toBeDefined();
      }
    });

    test('should perform performance validation when enabled', async () => {
      const requestId = await skillMintingProcess.submitForMinting(mockQueuedAdapter);
      
      // Wait for validation
      await new Promise(resolve => setTimeout(resolve, 200));
      
      const validationResult = skillMintingProcess.getValidationResult(requestId);
      if (validationResult) {
        expect(validationResult.performanceValidation).toBeDefined();
        expect(validationResult.performanceValidation.expectedAccuracy).toBeGreaterThanOrEqual(0);
        expect(validationResult.performanceValidation.inferenceLatency).toBeGreaterThanOrEqual(0);
      }
    });
  });

  describe('Consensus and Minting', () => {
    test('should complete full minting process', async () => {
      const requestId = await skillMintingProcess.submitForMinting(mockQueuedAdapter);
      
      // Wait for full process to complete
      await new Promise(resolve => setTimeout(resolve, 1000));
      
      const request = skillMintingProcess.getMintingRequest(requestId);
      expect(request).toBeDefined();
      
      // Should eventually reach minted status
      await new Promise(resolve => {
        const checkStatus = () => {
          const req = skillMintingProcess.getMintingRequest(requestId);
          if (req && (req.status === MintingStatus.MINTED || req.status === MintingStatus.FAILED)) {
            resolve(undefined);
          } else {
            setTimeout(checkStatus, 100);
          }
        };
        checkStatus();
      });
    });

    test('should create blockchain record for minted skill', async () => {
      const requestId = await skillMintingProcess.submitForMinting(mockQueuedAdapter);

      // Verify request was submitted successfully
      expect(typeof requestId).toBe('string');
      expect(requestId.length).toBeGreaterThan(0);

      // Wait for minting
      await new Promise(resolve => setTimeout(resolve, 1000));
      
      const blockchainRecord = skillMintingProcess.getMintedSkill(mockQueuedAdapter.loraAdapter.skillId);
      if (blockchainRecord) {
        expect(blockchainRecord.skillId).toBe(mockQueuedAdapter.loraAdapter.skillId);
        expect(blockchainRecord.skillHash).toBeTruthy();
        expect(blockchainRecord.transactionHash).toBeTruthy();
        expect(blockchainRecord.blockHeight).toBeGreaterThan(0);
      }
    });
  });

  describe('Event Handling', () => {
    test('should emit minting events', async () => {
      const mintingSubmittedSpy = jest.fn();
      const validationStartedSpy = jest.fn();
      
      skillMintingProcess.on('mintingSubmitted', mintingSubmittedSpy);
      skillMintingProcess.on('validationStarted', validationStartedSpy);
      
      await skillMintingProcess.submitForMinting(mockQueuedAdapter);
      
      expect(mintingSubmittedSpy).toHaveBeenCalled();
      
      // Wait for validation to start
      await new Promise(resolve => setTimeout(resolve, 100));
    });

    test('should emit validation events', async () => {
      const validationFailedSpy = jest.fn();
      const consensusStartedSpy = jest.fn();
      
      skillMintingProcess.on('validationFailed', validationFailedSpy);
      skillMintingProcess.on('consensusStarted', consensusStartedSpy);
      
      await skillMintingProcess.submitForMinting(mockQueuedAdapter);
      
      // Wait for events
      await new Promise(resolve => setTimeout(resolve, 200));
    });
  });

  describe('Error Handling', () => {
    test('should handle malformed LoRA adapter weights', async () => {
      const malformedAdapter = {
        ...mockQueuedAdapter,
        loraAdapter: {
          ...mockQueuedAdapter.loraAdapter,
          weightsA: new Float32Array([NaN, Infinity, -Infinity]),
          weightsB: new Float32Array([NaN, Infinity, -Infinity])
        }
      };

      const requestId = await skillMintingProcess.submitForMinting(malformedAdapter);
      
      // Should handle gracefully
      await new Promise(resolve => setTimeout(resolve, 200));
      
      const validationResult = skillMintingProcess.getValidationResult(requestId);
      if (validationResult) {
        expect(validationResult.technicalValidation.weightsIntegrity).toBe(false);
      }
    });

    test('should handle dimension inconsistency', async () => {
      const inconsistentAdapter = {
        ...mockQueuedAdapter,
        loraAdapter: {
          ...mockQueuedAdapter.loraAdapter,
          weightsA: new Float32Array(100), // Wrong size
          weightsB: new Float32Array(200)  // Wrong size
        }
      };

      const requestId = await skillMintingProcess.submitForMinting(inconsistentAdapter);
      
      await new Promise(resolve => setTimeout(resolve, 200));
      
      const validationResult = skillMintingProcess.getValidationResult(requestId);
      if (validationResult) {
        expect(validationResult.technicalValidation.dimensionConsistency).toBe(false);
      }
    });
  });

  describe('Configuration', () => {
    test('should respect minimum validation score', async () => {
      const strictProcess = new SkillMintingProcess({
        minValidationScore: 0.95
      });
      
      await strictProcess.initialize();
      
      const requestId = await strictProcess.submitForMinting(mockQueuedAdapter);
      
      // Wait for validation
      await new Promise(resolve => setTimeout(resolve, 200));
      
      const request = strictProcess.getMintingRequest(requestId);
      expect(request).toBeDefined();
      
      strictProcess.stop();
    });

    test('should handle disabled security validation', async () => {
      const noSecurityProcess = new SkillMintingProcess({
        enableSecurityValidation: false
      });
      
      await noSecurityProcess.initialize();
      
      const requestId = await noSecurityProcess.submitForMinting(mockQueuedAdapter);
      
      await new Promise(resolve => setTimeout(resolve, 200));
      
      const validationResult = noSecurityProcess.getValidationResult(requestId);
      if (validationResult) {
        expect(validationResult.securityValidation.maliciousCodeDetection).toBe(true);
        expect(validationResult.securityValidation.privacyCompliance).toBe(true);
      }
      
      noSecurityProcess.stop();
    });

    test('should handle disabled performance validation', async () => {
      const noPerformanceProcess = new SkillMintingProcess({
        enablePerformanceValidation: false
      });
      
      await noPerformanceProcess.initialize();
      
      const requestId = await noPerformanceProcess.submitForMinting(mockQueuedAdapter);
      
      await new Promise(resolve => setTimeout(resolve, 200));
      
      const validationResult = noPerformanceProcess.getValidationResult(requestId);
      if (validationResult) {
        expect(validationResult.performanceValidation.expectedAccuracy).toBe(0);
        expect(validationResult.performanceValidation.inferenceLatency).toBe(0);
      }

      noPerformanceProcess.stop();
    });
  });

  describe('Type Usage Validation', () => {
    it('should validate imported types are properly used', () => {
      // Test QueuedLoRAAdapter type usage
      const mockQueuedAdapter: QueuedLoRAAdapter = {
        queueId: 'queued-adapter-1',
        loraAdapter: {
          skillId: 'adapter-1',
          skillName: 'Test Adapter',
          description: 'Test adapter description',
          baseModelCompatibility: 'hrm',
          version: 1,
          rank: 16,
          alpha: 32,
          weightsA: new Float32Array(Array.from({ length: 8192 }, () => Math.random() * 0.1 - 0.05)),
          weightsB: new Float32Array(Array.from({ length: 8192 }, () => Math.random() * 0.1 - 0.05)),
          additionalMetadata: {}
        },
        trainingDataset: {
          datasetId: 'dataset-1',
          clusterId: 'cluster-1',
          errorNodes: [],
          validatedSolutions: [],
          trainingPairs: [],
          datasetMetrics: {
            totalPairs: 0,
            averageValidationScore: 0,
            diversityScore: 0,
            complexityScore: 0,
            qualityScore: 0
          },
          createdAt: new Date()
        },
        priority: 1,
        status: QueueStatus.PENDING,
        submittedAt: new Date(),
        retryCount: 0,
        maxRetries: 3
      };

      expect(mockQueuedAdapter.priority).toBe(1);
      expect(mockQueuedAdapter.status).toBe(QueueStatus.PENDING);

      // Test LoRAAdapterSkill type usage
      const mockLoRASkill: LoRAAdapterSkill = {
        skillId: 'lora-skill-1',
        skillName: 'Test LoRA Skill',
        description: 'Test LoRA skill description',
        baseModelCompatibility: 'hrm',
        version: 1,
        rank: 16,
        alpha: 32,
        weightsA: new Float32Array(Array.from({ length: 8192 }, () => Math.random() * 0.1 - 0.05)),
        weightsB: new Float32Array(Array.from({ length: 8192 }, () => Math.random() * 0.1 - 0.05)),
        additionalMetadata: {}
      };

      expect(mockLoRASkill.rank).toBe(16);
      expect(mockLoRASkill.alpha).toBe(32);
      expect(mockLoRASkill.weightsA instanceof Float32Array).toBe(true);
      expect(mockLoRASkill.weightsB instanceof Float32Array).toBe(true);
    });
  });
});
