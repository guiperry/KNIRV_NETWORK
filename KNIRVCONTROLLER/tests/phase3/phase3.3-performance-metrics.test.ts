/**
 * Phase 3.3 Performance Metrics Tests
 * 
 * Tests for success rates and cluster-based competition metrics tracking
 */

import {
  PerformanceMetrics,
  ClusterMetrics,
  AgentMetrics,
  SkillMetrics
} from '../../src/core/knirvgraph/PerformanceMetrics';
import { ErrorCluster, ErrorNode } from '../../src/core/knirvgraph/ErrorNodeClustering';
import { CompetitiveSolution } from '../../src/core/knirvgraph/AgentAssignmentSystem';
import { LoRAAdapterSkill } from '../../src/core/lora/LoRAAdapterEngine';
import { SkillDiscoveryResult } from '../../src/core/knirvgraph/HRMCoreModel';

describe('Phase 3.3 - Performance Metrics', () => {
  let performanceMetrics: PerformanceMetrics;
  let mockErrorCluster: ErrorCluster;
  let mockSolutions: CompetitiveSolution[];
  let mockValidatedSolutions: CompetitiveSolution[];
  let mockLoRAAdapter: LoRAAdapterSkill;
  let mockDiscoveryResult: SkillDiscoveryResult;

  beforeEach(async () => {
    performanceMetrics = new PerformanceMetrics({
      updateInterval: 1000,
      retentionPeriod: 24 * 60 * 60 * 1000, // 24 hours
      enableRealTimeTracking: true,
      enableCompetitionMetrics: true,
      maxLeaderboardSize: 10
    });

    await performanceMetrics.initialize();

    // Create mock data
    const mockErrorNodes: ErrorNode[] = [
      {
        id: 'error_001',
        errorType: 'TypeError',
        errorMessage: 'Cannot read property of undefined',
        stackTrace: 'at function1 (file.js:10:5)',
        severity: 'high',
        context: { variable: 'user', line: 10 },
        timestamp: new Date(),
        bountyAmount: 100,
        tags: ['javascript', 'undefined'],
        metadata: {}
      },
      {
        id: 'error_002',
        errorType: 'ReferenceError',
        errorMessage: 'Variable is not defined',
        stackTrace: 'at function2 (file.js:15:3)',
        severity: 'medium',
        context: { variable: 'config', line: 15 },
        timestamp: new Date(),
        bountyAmount: 75,
        tags: ['javascript', 'reference'],
        metadata: {}
      }
    ];

    mockErrorCluster = {
      clusterId: 'cluster_001',
      clusterName: 'JavaScript Error Cluster',
      description: 'Cluster for JavaScript runtime errors',
      errorNodes: mockErrorNodes,
      centroid: {
        errorTypeVector: [1, 0, 0, 0, 0, 0],
        contextVector: [2, 1, 1, 0, 1],
        severityScore: 0.75,
        tagVector: [1, 0, 0, 0, 0, 0, 0],
        semanticVector: [0.5, 0.3, 0.8, 0.2, 0.1, 0.4, 0.6, 0.2]
      },
      similarity: 0.85,
      totalBounty: 175,
      assignedAgents: ['agent_001', 'agent_002'],
      solutions: [],
      createdAt: new Date(),
      lastUpdated: new Date()
    };

    mockSolutions = [
      {
        solutionId: 'sol_001',
        agentId: 'agent_001',
        clusterId: 'cluster_001',
        errorNodeId: 'error_001',
        solutionCode: 'if (user && user.property) { return user.property; }',
        description: 'Null check solution',
        approach: 'null_check',
        estimatedEffectiveness: 0.9,
        submittedAt: new Date(Date.now() - 60000),
        validationStatus: 'validated' as const,
        dveValidationScore: 0.9,
        validatedAt: new Date(),
        bountyAwarded: 90
      },
      {
        solutionId: 'sol_002',
        agentId: 'agent_002',
        clusterId: 'cluster_001',
        errorNodeId: 'error_002',
        solutionCode: 'const config = getConfig() || defaultConfig;',
        description: 'Default fallback solution',
        approach: 'default_fallback',
        estimatedEffectiveness: 0.85,
        submittedAt: new Date(Date.now() - 45000),
        validationStatus: 'validated' as const,
        dveValidationScore: 0.85,
        validatedAt: new Date(),
        bountyAwarded: 64
      },
      {
        solutionId: 'sol_003',
        agentId: 'agent_001',
        clusterId: 'cluster_001',
        errorNodeId: 'error_001',
        solutionCode: 'try { return user.property; } catch(e) { return null; }',
        description: 'Try-catch solution',
        approach: 'try_catch',
        estimatedEffectiveness: 0.75,
        submittedAt: new Date(Date.now() - 30000),
        validationStatus: 'pending' as const,
        dveValidationScore: 0.75,
        validatedAt: undefined,
        bountyAwarded: 0
      }
    ];

    mockValidatedSolutions = mockSolutions.filter(s =>
      s.validationStatus === 'validated' &&
      s.dveValidationScore &&
      s.dveValidationScore > 0.7
    );

    mockLoRAAdapter = {
      skillId: 'skill_001',
      skillName: 'JavaScript Error Resolver',
      description: 'Resolves JavaScript runtime errors',
      baseModelCompatibility: 'hrm',
      version: 1,
      rank: 8,
      alpha: 16.0,
      weightsA: new Float32Array(64),
      weightsB: new Float32Array(64),
      additionalMetadata: {}
    };

    mockDiscoveryResult = {
      skillId: 'skill_001',
      discoveredName: 'JavaScript Error Resolver',
      category: 'debugging',
      subcategory: 'error_resolution',
      description: 'Resolves JavaScript runtime errors',
      capabilities: ['typeerror_resolution', 'null_check'],
      complexity: 0.7,
      confidence: 0.85,
      tags: ['debugging', 'javascript'],
      relatedSkills: []
    };
  });

  afterEach(() => {
    performanceMetrics.stop();
    performanceMetrics.reset();
  });

  describe('Initialization', () => {
    test('should initialize successfully', async () => {
      const newMetrics = new PerformanceMetrics();
      await expect(newMetrics.initialize()).resolves.not.toThrow();
      expect(newMetrics.isReady()).toBe(true);
      newMetrics.stop();
    });

    test('should initialize with custom configuration', async () => {
      const customMetrics = new PerformanceMetrics({
        updateInterval: 5000,
        enableRealTimeTracking: false,
        enableCompetitionMetrics: false
      });

      await expect(customMetrics.initialize()).resolves.not.toThrow();
      customMetrics.stop();
    });
  });

  describe('Cluster Performance Tracking', () => {
    test('should track cluster performance correctly', () => {
      performanceMetrics.trackClusterPerformance(
        mockErrorCluster,
        mockSolutions,
        mockValidatedSolutions
      );

      const clusterMetrics = performanceMetrics.getClusterMetrics('cluster_001') as ClusterMetrics;
      
      expect(clusterMetrics).toBeDefined();
      expect(clusterMetrics.clusterId).toBe('cluster_001');
      expect(clusterMetrics.totalErrors).toBe(2);
      expect(clusterMetrics.totalSolutions).toBe(3);
      expect(clusterMetrics.validatedSolutions).toBe(2);
      expect(clusterMetrics.successRate).toBeCloseTo(2/3, 2);
      expect(clusterMetrics.averageValidationScore).toBeCloseTo(0.875, 2);
      expect(clusterMetrics.totalBountyAwarded).toBe(154);
      expect(clusterMetrics.participatingAgents).toBe(2);
    });

    test('should handle empty solutions', () => {
      performanceMetrics.trackClusterPerformance(
        mockErrorCluster,
        [],
        []
      );

      const clusterMetrics = performanceMetrics.getClusterMetrics('cluster_001') as ClusterMetrics;
      
      expect(clusterMetrics).toBeDefined();
      expect(clusterMetrics.successRate).toBe(0);
      expect(clusterMetrics.averageValidationScore).toBe(0);
      expect(clusterMetrics.participatingAgents).toBe(0);
    });

    test('should get all cluster metrics', () => {
      performanceMetrics.trackClusterPerformance(
        mockErrorCluster,
        mockSolutions,
        mockValidatedSolutions
      );

      const allMetrics = performanceMetrics.getClusterMetrics() as ClusterMetrics[];
      
      expect(allMetrics).toBeInstanceOf(Array);
      expect(allMetrics.length).toBe(1);
      expect(allMetrics[0].clusterId).toBe('cluster_001');
    });
  });

  describe('Agent Performance Tracking', () => {
    test('should track agent performance correctly', () => {
      performanceMetrics.trackAgentPerformance('agent_001', mockSolutions);

      const agentMetrics = performanceMetrics.getAgentMetrics('agent_001') as AgentMetrics;
      
      expect(agentMetrics).toBeDefined();
      expect(agentMetrics.agentId).toBe('agent_001');
      expect(agentMetrics.totalSolutionsSubmitted).toBe(2);
      expect(agentMetrics.validatedSolutions).toBe(1);
      expect(agentMetrics.successRate).toBe(0.5);
      expect(agentMetrics.totalBountyEarned).toBe(90);
      expect(agentMetrics.clustersParticipated).toBe(1);
    });

    test('should calculate agent reputation correctly', () => {
      performanceMetrics.trackAgentPerformance('agent_002', mockSolutions);

      const agentMetrics = performanceMetrics.getAgentMetrics('agent_002') as AgentMetrics;
      
      expect(agentMetrics).toBeDefined();
      expect(agentMetrics.reputation).toBeGreaterThan(0);
      expect(agentMetrics.reputation).toBeLessThanOrEqual(1);
    });

    test('should handle agent with no solutions', () => {
      performanceMetrics.trackAgentPerformance('agent_003', []);

      const agentMetrics = performanceMetrics.getAgentMetrics('agent_003') as AgentMetrics;
      
      expect(agentMetrics).toBeDefined();
      expect(agentMetrics.totalSolutionsSubmitted).toBe(0);
      expect(agentMetrics.successRate).toBe(0);
      expect(agentMetrics.reputation).toBe(0);
    });

    test('should get all agent metrics', () => {
      performanceMetrics.trackAgentPerformance('agent_001', mockSolutions);
      performanceMetrics.trackAgentPerformance('agent_002', mockSolutions);

      const allMetrics = performanceMetrics.getAgentMetrics() as AgentMetrics[];
      
      expect(allMetrics).toBeInstanceOf(Array);
      expect(allMetrics.length).toBe(2);
    });
  });

  describe('Skill Performance Tracking', () => {
    test('should track skill performance correctly', () => {
      performanceMetrics.trackSkillPerformance(
        mockLoRAAdapter,
        mockDiscoveryResult,
        0.85
      );

      const skillMetrics = performanceMetrics.getSkillMetrics('skill_001') as SkillMetrics;
      
      expect(skillMetrics).toBeDefined();
      expect(skillMetrics.skillId).toBe('skill_001');
      expect(skillMetrics.skillName).toBe('JavaScript Error Resolver');
      expect(skillMetrics.category).toBe('debugging');
      expect(skillMetrics.trainingAccuracy).toBe(0.85);
      expect(skillMetrics.validationScore).toBe(0.85);
      expect(skillMetrics.memoryUsage).toBe(512); // 64 + 64 floats * 4 bytes
    });

    test('should get all skill metrics', () => {
      performanceMetrics.trackSkillPerformance(
        mockLoRAAdapter,
        mockDiscoveryResult,
        0.85
      );

      const allMetrics = performanceMetrics.getSkillMetrics() as SkillMetrics[];
      
      expect(allMetrics).toBeInstanceOf(Array);
      expect(allMetrics.length).toBe(1);
      expect(allMetrics[0].skillId).toBe('skill_001');
    });
  });

  describe('Competition Metrics Tracking', () => {
    test('should track competition metrics when enabled', () => {
      performanceMetrics.trackCompetitionMetrics(
        'cluster_001',
        mockSolutions,
        1
      );

      const competitionMetrics = performanceMetrics.getCompetitionMetrics('cluster_001');
      
      expect(competitionMetrics).toBeInstanceOf(Array);
      expect(competitionMetrics.length).toBe(1);
      
      const competition = competitionMetrics[0];
      expect(competition.clusterId).toBe('cluster_001');
      expect(competition.competitionRound).toBe(1);
      expect(competition.participatingAgents).toEqual(['agent_001', 'agent_002']);
      expect(competition.solutionsSubmitted).toBe(3);
      expect(competition.leaderboard.length).toBeGreaterThan(0);
    });

    test('should generate leaderboard correctly', () => {
      performanceMetrics.trackCompetitionMetrics(
        'cluster_001',
        mockSolutions,
        1
      );

      const competitionMetrics = performanceMetrics.getCompetitionMetrics('cluster_001');
      const leaderboard = competitionMetrics[0].leaderboard;
      
      expect(leaderboard).toBeInstanceOf(Array);
      expect(leaderboard.length).toBe(2);
      
      // Should be sorted by score (descending)
      expect(leaderboard[0].score).toBeGreaterThanOrEqual(leaderboard[1].score);
      
      // Check leaderboard entry structure
      expect(leaderboard[0]).toHaveProperty('rank');
      expect(leaderboard[0]).toHaveProperty('agentId');
      expect(leaderboard[0]).toHaveProperty('score');
      expect(leaderboard[0]).toHaveProperty('solutionsSubmitted');
      expect(leaderboard[0]).toHaveProperty('validatedSolutions');
    });

    test('should calculate competition intensity correctly', () => {
      performanceMetrics.trackCompetitionMetrics(
        'cluster_001',
        mockSolutions,
        1
      );

      const competitionMetrics = performanceMetrics.getCompetitionMetrics('cluster_001');
      const competition = competitionMetrics[0];
      
      expect(competition.competitionIntensity).toBeGreaterThanOrEqual(0);
      expect(competition.competitionIntensity).toBeLessThanOrEqual(1);
    });

    test('should not track competition metrics when disabled', () => {
      const noCompetitionMetrics = new PerformanceMetrics({
        enableCompetitionMetrics: false
      });

      noCompetitionMetrics.trackCompetitionMetrics(
        'cluster_001',
        mockSolutions,
        1
      );

      const competitionMetrics = noCompetitionMetrics.getCompetitionMetrics('cluster_001');
      expect(competitionMetrics).toEqual([]);

      noCompetitionMetrics.stop();
    });
  });

  describe('System Metrics', () => {
    test('should calculate system metrics correctly', () => {
      // Add some data to calculate system metrics
      performanceMetrics.trackClusterPerformance(
        mockErrorCluster,
        mockSolutions,
        mockValidatedSolutions
      );
      
      performanceMetrics.trackAgentPerformance('agent_001', mockSolutions);
      performanceMetrics.trackSkillPerformance(
        mockLoRAAdapter,
        mockDiscoveryResult,
        0.85
      );

      const systemMetrics = performanceMetrics.getSystemMetrics();
      
      expect(systemMetrics).toBeDefined();
      expect(systemMetrics.totalClusters).toBe(1);
      expect(systemMetrics.totalAgents).toBe(1);
      expect(systemMetrics.totalSkills).toBe(1);
      expect(systemMetrics.overallSuccessRate).toBeCloseTo(2/3, 2);
      expect(systemMetrics.networkHealth).toBeGreaterThanOrEqual(0);
      expect(systemMetrics.networkHealth).toBeLessThanOrEqual(1);
    });

    test('should handle empty system state', () => {
      const systemMetrics = performanceMetrics.getSystemMetrics();
      
      expect(systemMetrics.totalClusters).toBe(0);
      expect(systemMetrics.totalAgents).toBe(0);
      expect(systemMetrics.totalSkills).toBe(0);
      expect(systemMetrics.overallSuccessRate).toBe(0);
    });
  });

  describe('Event Handling', () => {
    test('should emit cluster metrics updated event', () => {
      const clusterMetricsUpdatedSpy = jest.fn();
      performanceMetrics.on('clusterMetricsUpdated', clusterMetricsUpdatedSpy);

      performanceMetrics.trackClusterPerformance(
        mockErrorCluster,
        mockSolutions,
        mockValidatedSolutions
      );

      expect(clusterMetricsUpdatedSpy).toHaveBeenCalled();
      expect(clusterMetricsUpdatedSpy.mock.calls[0][0]).toHaveProperty('clusterId', 'cluster_001');
    });

    test('should emit agent metrics updated event', () => {
      const agentMetricsUpdatedSpy = jest.fn();
      performanceMetrics.on('agentMetricsUpdated', agentMetricsUpdatedSpy);

      performanceMetrics.trackAgentPerformance('agent_001', mockSolutions);

      expect(agentMetricsUpdatedSpy).toHaveBeenCalled();
      expect(agentMetricsUpdatedSpy.mock.calls[0][0]).toHaveProperty('agentId', 'agent_001');
    });

    test('should emit skill metrics updated event', () => {
      const skillMetricsUpdatedSpy = jest.fn();
      performanceMetrics.on('skillMetricsUpdated', skillMetricsUpdatedSpy);

      performanceMetrics.trackSkillPerformance(
        mockLoRAAdapter,
        mockDiscoveryResult,
        0.85
      );

      expect(skillMetricsUpdatedSpy).toHaveBeenCalled();
      expect(skillMetricsUpdatedSpy.mock.calls[0][0]).toHaveProperty('skillId', 'skill_001');
    });

    test('should emit competition metrics updated event', () => {
      const competitionMetricsUpdatedSpy = jest.fn();
      performanceMetrics.on('competitionMetricsUpdated', competitionMetricsUpdatedSpy);

      performanceMetrics.trackCompetitionMetrics(
        'cluster_001',
        mockSolutions,
        1
      );

      expect(competitionMetricsUpdatedSpy).toHaveBeenCalled();
      expect(competitionMetricsUpdatedSpy.mock.calls[0][0]).toHaveProperty('clusterId', 'cluster_001');
    });
  });

  describe('Data Retrieval', () => {
    test('should return undefined for non-existent cluster metrics', () => {
      const metrics = performanceMetrics.getClusterMetrics('non_existent');
      expect(metrics).toBeUndefined();
    });

    test('should return undefined for non-existent agent metrics', () => {
      const metrics = performanceMetrics.getAgentMetrics('non_existent');
      expect(metrics).toBeUndefined();
    });

    test('should return undefined for non-existent skill metrics', () => {
      const metrics = performanceMetrics.getSkillMetrics('non_existent');
      expect(metrics).toBeUndefined();
    });

    test('should return empty array for non-existent competition metrics', () => {
      const metrics = performanceMetrics.getCompetitionMetrics('non_existent');
      expect(metrics).toEqual([]);
    });
  });

  describe('Performance and Scalability', () => {
    test('should handle large datasets efficiently', () => {
      const largeSolutions = Array.from({ length: 1000 }, (_, i) => ({
        ...mockSolutions[0],
        solutionId: `sol_${i}`,
        agentId: `agent_${i % 10}`,
        dveValidationScore: Math.random()
      }));

      const startTime = Date.now();
      
      performanceMetrics.trackClusterPerformance(
        mockErrorCluster,
        largeSolutions,
        largeSolutions.filter(s => s.dveValidationScore > 0.7)
      );

      const endTime = Date.now();
      
      // Should complete within reasonable time (less than 1 second)
      expect(endTime - startTime).toBeLessThan(1000);
    });

    test('should handle concurrent metric updates', () => {
      const promises = Array.from({ length: 10 }, (_, i) => {
        return new Promise<void>(resolve => {
          setTimeout(() => {
            performanceMetrics.trackAgentPerformance(`agent_${i}`, mockSolutions);
            resolve();
          }, Math.random() * 100);
        });
      });

      return Promise.all(promises).then(() => {
        const allMetrics = performanceMetrics.getAgentMetrics() as AgentMetrics[];
        expect(allMetrics.length).toBe(10);
      });
    });
  });

  describe('Type Usage Validation', () => {
    it('should validate imported types are properly used', () => {
      // Test ErrorCluster type usage
      const mockErrorCluster: ErrorCluster = {
        clusterId: 'cluster-1',
        clusterName: 'Test Cluster',
        description: 'Test cluster description',
        errorNodes: [],
        centroid: {
          errorTypeVector: [1, 0, 0, 0, 0, 0],
          contextVector: [1, 0, 0, 0, 0],
          severityScore: 0.5,
          tagVector: [1, 0, 0, 0, 0, 0, 0],
          semanticVector: [0.5, 0.3, 0.2, 0.1, 0.4, 0.6, 0.2, 0.1]
        },
        similarity: 0.8,
        totalBounty: 100,
        assignedAgents: [],
        solutions: [],
        createdAt: new Date(),
        lastUpdated: new Date()
      };

      expect(mockErrorCluster.clusterId).toBe('cluster-1');
      expect(mockErrorCluster.similarity).toBe(0.8);

      // Test ErrorNode type usage
      const mockErrorNode: ErrorNode = {
        id: 'error-node-1',
        errorType: 'TypeError',
        errorMessage: 'Test error message',
        context: { variable: 'test' },
        severity: 'medium' as const,
        timestamp: new Date(),
        bountyAmount: 50,
        tags: ['test'],
        metadata: {}
      };

      expect(mockErrorNode.errorType).toBe('TypeError');
      expect(mockErrorNode.bountyAmount).toBe(50);

      // Test CompetitiveSolution type usage
      const mockSolution: CompetitiveSolution = {
        solutionId: 'solution-1',
        agentId: 'agent-1',
        clusterId: 'cluster-1',
        errorNodeId: 'error-1',
        solutionCode: 'test solution code',
        description: 'Test solution description',
        approach: 'test_approach',
        estimatedEffectiveness: 0.95,
        submittedAt: new Date(),
        validationStatus: 'pending' as const
      };

      expect(mockSolution.solutionId).toBe('solution-1');
      expect(mockSolution.validationStatus).toBe('pending');

      // Test LoRAAdapterSkill type usage
      const mockLoRASkill: LoRAAdapterSkill = {
        skillId: 'skill-1',
        skillName: 'Test LoRA Skill',
        description: 'Test skill description',
        baseModelCompatibility: 'hrm',
        version: 1,
        rank: 16,
        alpha: 32.0,
        weightsA: new Float32Array(64),
        weightsB: new Float32Array(64),
        additionalMetadata: {}
      };

      expect(mockLoRASkill.skillId).toBe('skill-1');
      expect(mockLoRASkill.rank).toBe(16);
    });
  });
});
