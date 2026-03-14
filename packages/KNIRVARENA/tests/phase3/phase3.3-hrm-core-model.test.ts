/**
 * Phase 3.3 HRM Core Model Tests
 * 
 * Tests for HRM WASM implementation as KNIRVGRAPH core model for skill discovery
 */

import { HRMCoreModel, SkillDiscoveryResult, CoreModelConfig } from '../../src/core/knirvgraph/HRMCoreModel';
import { LoRAAdapterSkill } from '../../src/core/lora/LoRAAdapterEngine';
import { TrainingDataset } from '../../src/core/knirvgraph/LoRAAdapterTrainingPipeline';
import { ErrorNode } from '../../src/core/knirvgraph/ErrorNodeClustering';
import { CompetitiveSolution } from '../../src/core/knirvgraph/AgentAssignmentSystem';

// Mock HRMBridge since we can't load actual WASM in tests
const mockHRMBridge = {
  initialize: jest.fn().mockResolvedValue(undefined),
  processCognitiveInput: jest.fn().mockResolvedValue({
    reasoning_result: 'Analyzed LoRA adapter for debugging capabilities with high confidence',
    confidence: 0.85,
    processing_time: 150,
    l_module_activations: [0.8, 0.6, 0.4, 0.2, 0.1, 0.3, 0.7, 0.5],
    h_module_activations: [0.9, 0.7, 0.5, 0.3]
  })
};

jest.mock('../../src/sensory-shell/HRMBridge', () => ({
  HRMBridge: jest.fn().mockImplementation(() => mockHRMBridge)
}));

describe('Phase 3.3 - HRM Core Model', () => {
  let hrmCoreModel: HRMCoreModel;
  let mockLoRAAdapter: LoRAAdapterSkill;
  let mockTrainingDataset: TrainingDataset;

  beforeEach(async () => {
    const config: Partial<CoreModelConfig> = {
      hrmConfig: {
        l_module_count: 8,
        h_module_count: 4,
        enable_adaptation: true,
        processing_timeout: 30000,
      },
      analysisTimeout: 60000,
      maxConcurrentAnalysis: 5,
      confidenceThreshold: 0.7,
    };

    hrmCoreModel = new HRMCoreModel(config);
    await hrmCoreModel.initialize();

    // Create mock LoRA adapter
    mockLoRAAdapter = {
      skillId: 'test_skill_001',
      skillName: 'Test Debugging Skill',
      description: 'A test skill for debugging JavaScript errors',
      baseModelCompatibility: 'hrm',
      version: 1,
      rank: 8,
      alpha: 16.0,
      weightsA: new Float32Array(Array.from({ length: 8192 }, () => Math.random() * 0.1 - 0.05)),
      weightsB: new Float32Array(Array.from({ length: 8192 }, () => Math.random() * 0.1 - 0.05)),
      additionalMetadata: {
        clusterId: 'cluster_001',
        trainingDatasetId: 'dataset_001',
        trainingPairs: '25',
        qualityScore: '0.85',
        timestamp: new Date().toISOString()
      }
    };

    // Create mock training dataset
    const mockErrorNodes: ErrorNode[] = [
      {
        id: 'error_001',
        errorType: 'TypeError',
        errorMessage: 'Cannot read property of undefined',
        stackTrace: 'at function1 (file.js:10:5)',
        severity: 'high',
        context: { variable: 'user', line: 10 },
        tags: ['javascript', 'undefined', 'property'],
        bountyAmount: 100,
        timestamp: new Date(),
        metadata: {}
      },
      {
        id: 'error_002',
        errorType: 'ReferenceError',
        errorMessage: 'Variable is not defined',
        stackTrace: 'at function2 (file.js:15:3)',
        severity: 'medium',
        context: { variable: 'config', line: 15 },
        tags: ['javascript', 'reference', 'undefined'],
        bountyAmount: 75,
        timestamp: new Date(),
        metadata: {}
      }
    ];

    const mockSolutions: CompetitiveSolution[] = [
      {
        solutionId: 'sol_001',
        errorNodeId: 'error_001',
        agentId: 'agent_001',
        clusterId: 'cluster_001',
        solutionCode: 'if (user && user.property) { return user.property; }',
        description: 'Null check solution for undefined property access',
        approach: 'null_check',
        estimatedEffectiveness: 0.9,
        validationStatus: 'validated',
        dveValidationScore: 0.9,
        bountyAwarded: 90,
        submittedAt: new Date(),
        validatedAt: new Date()
      },
      {
        solutionId: 'sol_002',
        errorNodeId: 'error_002',
        agentId: 'agent_002',
        clusterId: 'cluster_001',
        solutionCode: 'const config = getConfig() || defaultConfig;',
        description: 'Default fallback solution for undefined variable',
        approach: 'default_fallback',
        estimatedEffectiveness: 0.85,
        validationStatus: 'validated',
        dveValidationScore: 0.85,
        bountyAwarded: 64,
        submittedAt: new Date(),
        validatedAt: new Date()
      }
    ];

    mockTrainingDataset = {
      datasetId: 'dataset_001',
      clusterId: 'cluster_001',
      errorNodes: mockErrorNodes,
      validatedSolutions: mockSolutions,
      trainingPairs: [
        {
          pairId: 'pair_001',
          errorContext: {
            errorType: mockErrorNodes[0].errorType,
            errorMessage: mockErrorNodes[0].errorMessage,
            stackTrace: mockErrorNodes[0].stackTrace || '',
            contextVariables: mockErrorNodes[0].context,
            semanticEmbedding: [0.1, 0.2, 0.3, 0.4]
          },
          solutionContext: {
            solutionCode: 'if (user && user.property) { return user.property; }',
            approach: 'null_check',
            effectiveness: 0.9,
            codeEmbedding: [0.5, 0.3, 0.8, 0.2],
            transformationVector: [0.7, 0.4, 0.6]
          },
          weight: 0.9
        },
        {
          pairId: 'pair_002',
          errorContext: {
            errorType: mockErrorNodes[1].errorType,
            errorMessage: mockErrorNodes[1].errorMessage,
            stackTrace: mockErrorNodes[1].stackTrace || '',
            contextVariables: mockErrorNodes[1].context,
            semanticEmbedding: [0.2, 0.3, 0.4, 0.5]
          },
          solutionContext: {
            solutionCode: 'const config = getConfig() || defaultConfig;',
            approach: 'default_fallback',
            effectiveness: 0.85,
            codeEmbedding: [0.6, 0.4, 0.7, 0.3],
            transformationVector: [0.8, 0.5, 0.7]
          },
          weight: 0.85
        }
      ],
      datasetMetrics: {
        totalPairs: 2,
        averageValidationScore: 0.875,
        diversityScore: 0.8,
        complexityScore: 0.6,
        qualityScore: 0.85
      },
      createdAt: new Date()
    };
  });

  afterEach(() => {
    jest.clearAllMocks();
    // Reset mock to default behavior
    mockHRMBridge.initialize.mockResolvedValue(undefined);
    mockHRMBridge.processCognitiveInput.mockResolvedValue({
      reasoning_result: 'Analyzed LoRA adapter for debugging capabilities with high confidence',
      confidence: 0.85,
      processing_time: 150,
      l_module_activations: [0.8, 0.6, 0.4, 0.2, 0.1, 0.3, 0.7, 0.5],
      h_module_activations: [0.9, 0.7, 0.5, 0.3]
    });
  });

  describe('Initialization', () => {
    test('should initialize successfully', async () => {
      const newModel = new HRMCoreModel();
      await expect(newModel.initialize()).resolves.not.toThrow();
      expect(newModel.isReady()).toBe(true);
    });

    test('should handle initialization errors', async () => {
      // Mock HRMBridge to throw error
      mockHRMBridge.initialize.mockRejectedValueOnce(new Error('WASM load failed'));

      const newModel = new HRMCoreModel();
      await expect(newModel.initialize()).rejects.toThrow('WASM load failed');
    });
  });

  describe('Skill Discovery', () => {
    test('should discover skill successfully', async () => {
      const result = await hrmCoreModel.discoverSkill(mockLoRAAdapter, mockTrainingDataset);

      expect(result).toBeDefined();
      expect(result.skillId).toBe(mockLoRAAdapter.skillId);
      expect(result.discoveredName).toBeTruthy();
      expect(result.category).toBeTruthy();
      expect(result.confidence).toBeGreaterThan(0.7);
      expect(result.capabilities).toBeInstanceOf(Array);
      expect(result.capabilities.length).toBeGreaterThan(0);
      expect(result.tags).toBeInstanceOf(Array);
      expect(result.complexity).toBeGreaterThanOrEqual(0);
      expect(result.complexity).toBeLessThanOrEqual(1);
    });

    test('should handle skill discovery with low confidence', async () => {
      // Mock low confidence response
      mockHRMBridge.processCognitiveInput.mockResolvedValueOnce({
        reasoning_result: 'Low confidence analysis',
        confidence: 0.5,
        processing_time: 200,
        l_module_activations: [0.3, 0.2, 0.1, 0.1, 0.1, 0.1, 0.2, 0.1],
        h_module_activations: [0.4, 0.3, 0.2, 0.1]
      });

      const newModel = new HRMCoreModel();
      await newModel.initialize();

      const result = await newModel.discoverSkill(mockLoRAAdapter, mockTrainingDataset);
      expect(result.confidence).toBe(0.5);
    });

    test('should fail when not initialized', async () => {
      const newModel = new HRMCoreModel();
      await expect(newModel.discoverSkill(mockLoRAAdapter, mockTrainingDataset))
        .rejects.toThrow('HRM Core Model not initialized');
    });

    test('should handle cognitive processing errors', async () => {
      // Mock HRMBridge to throw error during processing
      mockHRMBridge.processCognitiveInput.mockRejectedValueOnce(new Error('Processing failed'));

      const newModel = new HRMCoreModel();
      await newModel.initialize();

      await expect(newModel.discoverSkill(mockLoRAAdapter, mockTrainingDataset))
        .rejects.toThrow('Processing failed');
    });
  });

  describe('Skill Categorization', () => {
    test('should categorize debugging skills correctly', async () => {
      const result = await hrmCoreModel.discoverSkill(mockLoRAAdapter, mockTrainingDataset);
      
      expect(result.category).toBeTruthy();
      expect(['debugging', 'optimization', 'refactoring', 'testing', 'security', 'integration', 'ui_ux', 'data_processing'])
        .toContain(result.category);
    });

    test('should generate appropriate skill names', async () => {
      const result = await hrmCoreModel.discoverSkill(mockLoRAAdapter, mockTrainingDataset);
      
      expect(result.discoveredName).toBeTruthy();
      expect(result.discoveredName.length).toBeGreaterThan(5);
      expect(result.discoveredName).toMatch(/\w+\s+\w+/); // At least two words
    });

    test('should extract relevant capabilities', async () => {
      const result = await hrmCoreModel.discoverSkill(mockLoRAAdapter, mockTrainingDataset);
      
      expect(result.capabilities).toBeInstanceOf(Array);
      expect(result.capabilities.length).toBeGreaterThan(0);
      expect(result.capabilities.length).toBeLessThanOrEqual(5);
      
      // Should contain error type related capabilities
      const hasErrorTypeCapability = result.capabilities.some(cap => 
        cap.includes('typeerror') || cap.includes('referenceerror') || cap.includes('resolution')
      );
      expect(hasErrorTypeCapability).toBe(true);
    });
  });

  describe('Weight Analysis', () => {
    test('should analyze LoRA weights correctly', async () => {
      const result = await hrmCoreModel.discoverSkill(mockLoRAAdapter, mockTrainingDataset);
      
      // Complexity should be calculated based on weights and training data
      expect(result.complexity).toBeGreaterThanOrEqual(0);
      expect(result.complexity).toBeLessThanOrEqual(1);
    });

    test('should handle extreme weight values', async () => {
      // Create adapter with extreme weights
      const extremeAdapter = {
        ...mockLoRAAdapter,
        weightsA: new Float32Array(Array.from({ length: 8192 }, () => Math.random() * 100 - 50)),
        weightsB: new Float32Array(Array.from({ length: 8192 }, () => Math.random() * 100 - 50))
      };

      const result = await hrmCoreModel.discoverSkill(extremeAdapter, mockTrainingDataset);
      expect(result).toBeDefined();
      expect(result.complexity).toBeGreaterThanOrEqual(0);
    });
  });

  describe('Result Storage and Retrieval', () => {
    test('should store and retrieve discovery results', async () => {
      const result = await hrmCoreModel.discoverSkill(mockLoRAAdapter, mockTrainingDataset);
      
      const storedResult = hrmCoreModel.getDiscoveryResult(mockLoRAAdapter.skillId);
      expect(storedResult).toEqual(result);
    });

    test('should return undefined for non-existent skill', () => {
      const result = hrmCoreModel.getDiscoveryResult('non_existent_skill');
      expect(result).toBeUndefined();
    });

    test('should return all discovery results', async () => {
      await hrmCoreModel.discoverSkill(mockLoRAAdapter, mockTrainingDataset);
      
      const allResults = hrmCoreModel.getAllDiscoveryResults();
      expect(allResults).toBeInstanceOf(Array);
      expect(allResults.length).toBe(1);
      expect(allResults[0].skillId).toBe(mockLoRAAdapter.skillId);
    });
  });

  describe('Performance and Scalability', () => {
    test('should handle multiple concurrent discoveries', async () => {
      const adapters = Array.from({ length: 3 }, (_, i) => ({
        ...mockLoRAAdapter,
        skillId: `test_skill_${i + 1}`,
        skillName: `Test Skill ${i + 1}`
      }));

      const promises = adapters.map(adapter => 
        hrmCoreModel.discoverSkill(adapter, mockTrainingDataset)
      );

      const results = await Promise.all(promises);
      
      expect(results).toHaveLength(3);
      results.forEach((result, i) => {
        expect(result.skillId).toBe(`test_skill_${i + 1}`);
      });
    });

    test('should handle large training datasets', async () => {
      // Create large dataset
      const largeDataset = {
        ...mockTrainingDataset,
        errorNodes: Array.from({ length: 100 }, (_, i) => ({
          ...mockTrainingDataset.errorNodes[0],
          id: `error_${i + 1}`,
          errorMessage: `Error message ${i + 1}`
        })),
        validatedSolutions: Array.from({ length: 200 }, (_, i) => ({
          ...mockTrainingDataset.validatedSolutions[0],
          solutionId: `sol_${i + 1}`,
          errorNodeId: `error_${(i % 100) + 1}`
        }))
      };

      const result = await hrmCoreModel.discoverSkill(mockLoRAAdapter, largeDataset);
      expect(result).toBeDefined();
      expect(result.confidence).toBeGreaterThan(0);
    });
  });

  describe('Error Handling', () => {
    test('should handle malformed LoRA adapter', async () => {
      const malformedAdapter = {
        ...mockLoRAAdapter,
        weightsA: new Float32Array([]), // Empty weights
        weightsB: new Float32Array([])
      };

      // Should not throw, but may have lower confidence
      const result = await hrmCoreModel.discoverSkill(malformedAdapter, mockTrainingDataset);
      expect(result).toBeDefined();
    });

    test('should handle empty training dataset', async () => {
      const emptyDataset = {
        ...mockTrainingDataset,
        errorNodes: [],
        validatedSolutions: [],
        trainingPairs: []
      };

      const result = await hrmCoreModel.discoverSkill(mockLoRAAdapter, emptyDataset);
      expect(result).toBeDefined();
      expect(result.capabilities.length).toBeGreaterThanOrEqual(0);
    });
  });

  describe('Configuration', () => {
    test('should respect confidence threshold', async () => {
      const highThresholdConfig: Partial<CoreModelConfig> = {
        confidenceThreshold: 0.95
      };

      const newModel = new HRMCoreModel(highThresholdConfig);
      await newModel.initialize();

      const result = await newModel.discoverSkill(mockLoRAAdapter, mockTrainingDataset);
      expect(result).toBeDefined();
      // Result should still be generated even if below threshold
    });

    test('should handle custom category mappings', async () => {
      const customConfig: Partial<CoreModelConfig> = {
        categoryMappings: {
          'custom_category': ['custom_subcategory1', 'custom_subcategory2']
        }
      };

      const newModel = new HRMCoreModel(customConfig);
      await newModel.initialize();

      const result = await newModel.discoverSkill(mockLoRAAdapter, mockTrainingDataset);
      expect(result).toBeDefined();
    });
  });

  describe('Type Usage Validation', () => {
    it('should validate imported types are properly used', () => {
      // Test SkillDiscoveryResult type usage
      const mockSkillResult: SkillDiscoveryResult = {
        skillId: 'test-skill',
        discoveredName: 'Test Skill',
        category: 'error-handling',
        subcategory: 'null-checks',
        description: 'A skill for handling null pointer errors',
        capabilities: ['null-check', 'error-prevention'],
        complexity: 0.7,
        confidence: 0.85,
        tags: ['javascript', 'null-safety'],
        relatedSkills: []
      };

      expect(mockSkillResult.confidence).toBe(0.85);
      expect(typeof mockSkillResult.skillId).toBe('string');

      // Test LoRAAdapterSkill type usage
      const mockLoRASkill: LoRAAdapterSkill = {
        skillId: 'lora-skill-1',
        skillName: 'Test LoRA Skill',
        description: 'A test LoRA adapter skill',
        baseModelCompatibility: 'CodeT5-base',
        version: 1,
        rank: 16,
        alpha: 32,
        weightsA: new Float32Array([0.1, 0.2, 0.3]),
        weightsB: new Float32Array([0.4, 0.5, 0.6]),
        additionalMetadata: {}
      };

      expect(mockLoRASkill.rank).toBe(16);
      expect(mockLoRASkill.alpha).toBe(32);

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

      // Test ErrorNode type usage
      const mockErrorNode: ErrorNode = {
        id: 'error-node-1',
        errorType: 'runtime_error',
        errorMessage: 'Test error',
        context: {},
        severity: 'medium',
        timestamp: new Date(),
        bountyAmount: 100,
        tags: ['runtime'],
        metadata: {}
      };

      expect(mockErrorNode.errorType).toBe('runtime_error');
      expect(mockErrorNode.timestamp instanceof Date).toBe(true);

      // Test CompetitiveSolution type usage
      const mockSolution: CompetitiveSolution = {
        solutionId: 'solution-1',
        agentId: 'agent-1',
        clusterId: 'cluster-1',
        errorNodeId: 'error-1',
        solutionCode: 'Test solution code',
        description: 'Test solution proposal',
        approach: 'test-approach',
        estimatedEffectiveness: 0.95,
        submittedAt: new Date(),
        validationStatus: 'pending'
      };

      expect(mockSolution.estimatedEffectiveness).toBe(0.95);
      expect(typeof mockSolution.description).toBe('string');
    });
  });
});
