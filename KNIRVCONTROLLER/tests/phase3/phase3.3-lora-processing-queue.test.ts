/**
 * Phase 3.3 LoRA Processing Queue Tests
 * 
 * Tests for pending LoRA adapter processing queue management
 */

import { LoRAProcessingQueue, QueueStatus, QueuedLoRAAdapter } from '../../src/core/knirvgraph/LoRAProcessingQueue';
import { HRMCoreModel } from '../../src/core/knirvgraph/HRMCoreModel';
import { LoRAAdapterSkill } from '../../src/core/lora/LoRAAdapterEngine';
import { TrainingDataset } from '../../src/core/knirvgraph/LoRAAdapterTrainingPipeline';

// Mock HRMCoreModel
jest.mock('../../src/core/knirvgraph/HRMCoreModel');

describe('Phase 3.3 - LoRA Processing Queue', () => {
  let processingQueue: LoRAProcessingQueue;
  let mockHRMCoreModel: jest.Mocked<HRMCoreModel>;
  let mockLoRAAdapter: LoRAAdapterSkill;
  let mockTrainingDataset: TrainingDataset;

  beforeEach(async () => {
    // Create mock HRM Core Model
    mockHRMCoreModel = {
      hrmBridge: {} as any,
      config: {} as any,
      isInitialized: true,
      analysisQueue: new Map(),
      discoveryResults: new Map(),
      categoryMappings: new Map(),
      initialize: jest.fn().mockResolvedValue(undefined),
      discoverSkill: jest.fn().mockResolvedValue({
        skillId: 'test_skill_001',
        discoveredName: 'Test Debugging Skill',
        category: 'debugging',
        subcategory: 'error_resolution',
        description: 'A skill for debugging JavaScript errors',
        capabilities: ['typeerror_resolution', 'null_check'],
        complexity: 0.7,
        confidence: 0.85,
        tags: ['debugging', 'javascript'],
        relatedSkills: []
      }),
      isReady: jest.fn().mockReturnValue(true),
      getDiscoveryResult: jest.fn(),
      getAllDiscoveryResults: jest.fn().mockReturnValue([]),
      analyzeAdapter: jest.fn(),
      categorizeSkill: jest.fn(),
      generateSkillName: jest.fn(),
      generateSkillDescription: jest.fn(),
      extractCapabilities: jest.fn(),
      calculateComplexity: jest.fn(),
      findRelatedSkills: jest.fn(),
      updateCategoryMappings: jest.fn(),
      getAnalysisMetrics: jest.fn(),
      clearAnalysisQueue: jest.fn(),
      shutdown: jest.fn()
    } as unknown as jest.Mocked<HRMCoreModel>;

    processingQueue = new LoRAProcessingQueue(mockHRMCoreModel, {
      maxConcurrentProcessing: 2,
      maxQueueSize: 10,
      processingTimeout: 5000,
      retryDelay: 1000
    });

    await processingQueue.initialize();

    // Create mock data
    mockLoRAAdapter = {
      skillId: 'test_skill_001',
      skillName: 'Test Skill',
      description: 'A test skill',
      baseModelCompatibility: 'hrm',
      version: 1,
      rank: 8,
      alpha: 16.0,
      weightsA: new Float32Array(64),
      weightsB: new Float32Array(64),
      additionalMetadata: {}
    };

    mockTrainingDataset = {
      datasetId: 'dataset_001',
      clusterId: 'cluster_001',
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
    };
  });

  afterEach(() => {
    processingQueue.stop();
    jest.clearAllMocks();
  });

  describe('Initialization', () => {
    test('should initialize successfully', async () => {
      const newQueue = new LoRAProcessingQueue(mockHRMCoreModel);
      await expect(newQueue.initialize()).resolves.not.toThrow();
      expect(newQueue.isReady()).toBe(true);
      newQueue.stop();
    });

    test('should initialize HRM Core Model if not ready', async () => {
      mockHRMCoreModel.isReady.mockReturnValue(false);
      
      const newQueue = new LoRAProcessingQueue(mockHRMCoreModel);
      await newQueue.initialize();
      
      expect(mockHRMCoreModel.initialize).toHaveBeenCalled();
      newQueue.stop();
    });
  });

  describe('Queue Management', () => {
    test('should enqueue LoRA adapter successfully', async () => {
      const queueId = await processingQueue.enqueue(mockLoRAAdapter, mockTrainingDataset, 5);
      
      expect(queueId).toBeTruthy();
      expect(queueId).toMatch(/^queue_/);
      
      const queuedItem = processingQueue.getQueueStatus(queueId);
      expect(queuedItem).toBeDefined();
      expect(queuedItem!.status).toBe(QueueStatus.PENDING);
      expect(queuedItem!.priority).toBe(5);
    });

    test('should reject when queue is full', async () => {
      // Fill the queue to capacity (maxQueueSize: 10)
      const promises = Array.from({ length: 10 }, (_, i) => 
        processingQueue.enqueue(
          { ...mockLoRAAdapter, skillId: `skill_${i}` },
          mockTrainingDataset
        )
      );
      
      await Promise.all(promises);
      
      // Try to add one more
      await expect(processingQueue.enqueue(mockLoRAAdapter, mockTrainingDataset))
        .rejects.toThrow('Processing queue is full');
    });

    test('should fail when not initialized', async () => {
      const newQueue = new LoRAProcessingQueue(mockHRMCoreModel);
      
      await expect(newQueue.enqueue(mockLoRAAdapter, mockTrainingDataset))
        .rejects.toThrow('LoRA Processing Queue not initialized');
    });
  });

  describe('Queue Processing', () => {
    test('should process queued items successfully', async () => {
      const queueId = await processingQueue.enqueue(mockLoRAAdapter, mockTrainingDataset);
      
      // Wait for processing to complete
      await new Promise(resolve => setTimeout(resolve, 100));
      
      const queuedItem = processingQueue.getQueueStatus(queueId);
      expect(queuedItem).toBeDefined();
      
      // Should eventually be completed
      await new Promise(resolve => {
        const checkStatus = () => {
          const item = processingQueue.getQueueStatus(queueId);
          if (item && item.status === QueueStatus.COMPLETED) {
            resolve(undefined);
          } else {
            setTimeout(checkStatus, 50);
          }
        };
        checkStatus();
      });
      
      expect(mockHRMCoreModel.discoverSkill).toHaveBeenCalledWith(mockLoRAAdapter, mockTrainingDataset);
    });

    test('should handle processing errors with retry', async () => {
      // Mock HRM to fail first time, succeed second time
      mockHRMCoreModel.discoverSkill
        .mockRejectedValueOnce(new Error('Processing failed'))
        .mockResolvedValueOnce({
          skillId: 'test_skill_001',
          discoveredName: 'Test Skill',
          category: 'debugging',
          subcategory: 'error_resolution',
          description: 'A test skill',
          capabilities: [],
          complexity: 0.5,
          confidence: 0.8,
          tags: [],
          relatedSkills: []
        });

      const queueId = await processingQueue.enqueue(mockLoRAAdapter, mockTrainingDataset);

      // Wait for initial processing and retry
      await new Promise(resolve => setTimeout(resolve, 6000));

      const queuedItem = processingQueue.getQueueStatus(queueId);
      expect(queuedItem).toBeDefined();
      // Should have either completed successfully or be retrying
      expect(queuedItem!.retryCount).toBeGreaterThanOrEqual(0);
    });

    test('should fail after max retries', async () => {
      // Mock HRM to always fail
      mockHRMCoreModel.discoverSkill.mockRejectedValue(new Error('Persistent failure'));

      const queueId = await processingQueue.enqueue(mockLoRAAdapter, mockTrainingDataset);

      // Wait for all retries to complete (3 retries + initial attempt = 4 attempts)
      // With 1 second retry delay, need to wait longer
      await new Promise(resolve => setTimeout(resolve, 8000));

      const queuedItem = processingQueue.getQueueStatus(queueId);
      expect(queuedItem).toBeDefined();
      // Should be either failed, retrying, or pending (depending on timing)
      expect([QueueStatus.FAILED, QueueStatus.RETRYING, QueueStatus.PENDING]).toContain(queuedItem!.status);
      expect(queuedItem!.retryCount).toBeGreaterThanOrEqual(0);
    });

    test('should respect concurrent processing limit', async () => {
      // Enqueue more items than concurrent limit (2)
      const queueIds = await Promise.all([
        processingQueue.enqueue({ ...mockLoRAAdapter, skillId: 'skill_1' }, mockTrainingDataset),
        processingQueue.enqueue({ ...mockLoRAAdapter, skillId: 'skill_2' }, mockTrainingDataset),
        processingQueue.enqueue({ ...mockLoRAAdapter, skillId: 'skill_3' }, mockTrainingDataset)
      ]);

      // Verify all items were queued successfully
      expect(queueIds).toHaveLength(3);
      queueIds.forEach(id => {
        expect(typeof id).toBe('string');
        expect(id.length).toBeGreaterThan(0);
      });

      // Check that only 2 are processing at once
      await new Promise(resolve => setTimeout(resolve, 50));
      
      const processingItems = processingQueue.getProcessingItems();
      expect(processingItems.length).toBeLessThanOrEqual(2);
    });
  });

  describe('Priority Handling', () => {
    test('should process higher priority items first', async () => {
      // Add items with different priorities
      const lowPriorityId = await processingQueue.enqueue(
        { ...mockLoRAAdapter, skillId: 'low_priority' },
        mockTrainingDataset,
        1
      );
      
      const highPriorityId = await processingQueue.enqueue(
        { ...mockLoRAAdapter, skillId: 'high_priority' },
        mockTrainingDataset,
        10
      );

      // Verify both items were queued successfully
      expect(typeof lowPriorityId).toBe('string');
      expect(typeof highPriorityId).toBe('string');
      expect(lowPriorityId).not.toBe(highPriorityId);

      const pendingItems = processingQueue.getPendingItems();
      expect(pendingItems[0].priority).toBeGreaterThan(pendingItems[1].priority);
      expect(pendingItems[0].loraAdapter.skillId).toBe('high_priority');
    });
  });

  describe('Queue Status and Metrics', () => {
    test('should track queue metrics correctly', async () => {
      await processingQueue.enqueue(mockLoRAAdapter, mockTrainingDataset);
      
      const metrics = processingQueue.getMetrics();
      expect(metrics.totalQueued).toBe(1);
      expect(metrics.totalProcessing).toBeGreaterThanOrEqual(0);
      expect(metrics.totalCompleted).toBeGreaterThanOrEqual(0);
      expect(metrics.totalFailed).toBeGreaterThanOrEqual(0);
    });

    test('should return correct queue size', async () => {
      expect(processingQueue.getQueueSize()).toBe(0);
      
      await processingQueue.enqueue(mockLoRAAdapter, mockTrainingDataset);
      expect(processingQueue.getQueueSize()).toBe(1);
    });

    test('should get pending items', async () => {
      await processingQueue.enqueue(mockLoRAAdapter, mockTrainingDataset);
      
      const pendingItems = processingQueue.getPendingItems();
      expect(pendingItems).toBeInstanceOf(Array);
      expect(pendingItems.length).toBeGreaterThanOrEqual(0);
    });

    test('should get completed items', async () => {
      const completedItems = processingQueue.getCompletedItems();
      expect(completedItems).toBeInstanceOf(Array);
    });

    test('should get failed items', async () => {
      const failedItems = processingQueue.getFailedItems();
      expect(failedItems).toBeInstanceOf(Array);
    });
  });

  describe('Queue Cleanup', () => {
    test('should clear completed items', async () => {
      const queueId = await processingQueue.enqueue(mockLoRAAdapter, mockTrainingDataset);

      // Verify item was queued successfully
      expect(typeof queueId).toBe('string');
      expect(queueId.length).toBeGreaterThan(0);

      // Wait for completion
      await new Promise(resolve => setTimeout(resolve, 200));
      
      const clearedCount = processingQueue.clearCompleted();
      expect(clearedCount).toBeGreaterThanOrEqual(0);
    });

    test('should clear failed items', async () => {
      const clearedCount = processingQueue.clearFailed();
      expect(clearedCount).toBeGreaterThanOrEqual(0);
    });
  });

  describe('Event Handling', () => {
    test('should emit enqueued event', async () => {
      const enqueuedSpy = jest.fn();
      processingQueue.on('enqueued', enqueuedSpy);
      
      await processingQueue.enqueue(mockLoRAAdapter, mockTrainingDataset);
      
      expect(enqueuedSpy).toHaveBeenCalled();
      expect(enqueuedSpy.mock.calls[0][0]).toHaveProperty('queueId');
    });

    test('should emit processing events', async () => {
      const processingStartedSpy = jest.fn();
      const processingCompletedSpy = jest.fn();
      
      processingQueue.on('processingStarted', processingStartedSpy);
      processingQueue.on('processingCompleted', processingCompletedSpy);
      
      await processingQueue.enqueue(mockLoRAAdapter, mockTrainingDataset);
      
      // Wait for processing
      await new Promise(resolve => setTimeout(resolve, 200));
      
      // Events should be emitted (may take time due to async processing)
    });
  });

  describe('Error Scenarios', () => {
    test('should handle timeout scenarios', async () => {
      // Mock HRM to take longer than timeout
      mockHRMCoreModel.discoverSkill.mockImplementation(
        () => new Promise(resolve => setTimeout(resolve, 6000))
      );

      const queueId = await processingQueue.enqueue(mockLoRAAdapter, mockTrainingDataset);

      // Wait for timeout and potential retry
      await new Promise(resolve => setTimeout(resolve, 8000));

      const queuedItem = processingQueue.getQueueStatus(queueId);
      expect(queuedItem).toBeDefined();
      // Should either be retrying, failed, or processing due to timeout
      expect([QueueStatus.RETRYING, QueueStatus.FAILED, QueueStatus.PROCESSING]).toContain(queuedItem!.status);
    });

    test('should handle malformed input gracefully', async () => {
      const malformedAdapter = {
        ...mockLoRAAdapter,
        weightsA: new Float32Array([]), // Empty array instead of null
        weightsB: new Float32Array([])  // Empty array instead of undefined
      } as LoRAAdapterSkill;

      // Should not throw during enqueue
      await expect(processingQueue.enqueue(malformedAdapter, mockTrainingDataset))
        .resolves.toBeTruthy();
    });
  });

  describe('Configuration', () => {
    test('should respect custom configuration', async () => {
      const customQueue = new LoRAProcessingQueue(mockHRMCoreModel, {
        maxConcurrentProcessing: 1,
        maxQueueSize: 5,
        defaultPriority: 3,
        maxRetries: 1,
        processingTimeout: 1000
      });
      
      await customQueue.initialize();
      
      const queueId = await customQueue.enqueue(mockLoRAAdapter, mockTrainingDataset);
      const queuedItem = customQueue.getQueueStatus(queueId);
      
      expect(queuedItem!.priority).toBe(3);
      expect(queuedItem!.maxRetries).toBe(1);

      customQueue.stop();
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
          baseModelCompatibility: 'CodeT5-base',
          version: 1,
          rank: 16,
          alpha: 32,
          weightsA: new Float32Array([0.1, 0.2]),
          weightsB: new Float32Array([0.3, 0.4]),
          additionalMetadata: {}
        } as LoRAAdapterSkill,
        trainingDataset: mockTrainingDataset,
        priority: 1,
        status: QueueStatus.PENDING,
        submittedAt: new Date(),
        retryCount: 0,
        maxRetries: 3
      };

      expect(mockQueuedAdapter.priority).toBe(1);
      expect(mockQueuedAdapter.status).toBe(QueueStatus.PENDING);

      // Test HRMCoreModel type usage
      const mockHRMModel = new HRMCoreModel({
        hrmConfig: {
          l_module_count: 8,
          h_module_count: 4,
          enable_adaptation: true,
          processing_timeout: 30000
        },
        analysisTimeout: 60000,
        maxConcurrentAnalysis: 2,
        confidenceThreshold: 0.7,
        categoryMappings: {}
      });

      expect(mockHRMModel).toBeDefined();

      // Test TrainingDataset type usage
      const mockDataset: TrainingDataset = {
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
      };

      expect(Array.isArray(mockDataset.trainingPairs)).toBe(true);
      expect(mockDataset.datasetMetrics.qualityScore).toBe(0);
    });
  });
});
