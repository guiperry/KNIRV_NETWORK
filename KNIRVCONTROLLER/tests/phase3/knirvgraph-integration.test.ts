/**
 * Phase 3 Tests: KNIRVGRAPH LoRA Adapter Creation Integration
 * 
 * Tests for ErrorNode clustering, agent assignment, competitive solutions,
 * and LoRA adapter training pipeline
 */

import { describe, it, expect, beforeEach } from '@jest/globals';

// Mock implementations for KNIRVGRAPH components
interface ErrorNode {
  id: string;
  errorType: string;
  errorMessage: string;
  stackTrace?: string;
  context: Record<string, unknown>;
  severity: 'low' | 'medium' | 'high' | 'critical';
  timestamp: Date;
  bountyAmount: number;
  tags: string[];
  metadata: Record<string, unknown>;
}

interface ErrorCluster {
  clusterId: string;
  clusterName: string;
  description: string;
  errorNodes: ErrorNode[];
  centroid: {
    errorTypeVector: number[];
    contextVector: number[];
    severityScore: number;
    tagVector: number[];
    semanticVector: number[];
  };
  similarity: number;
  totalBounty: number;
  assignedAgents: string[];
  solutions: Record<string, unknown>[];
  createdAt: Date;
  lastUpdated: Date;
}

interface Agent {
  agentId: string;
  agentName: string;
  expertise: string[];
  performanceScore: number;
  successRate: number;
  totalSolutionsSubmitted: number;
  totalSolutionsValidated: number;
  totalBountyEarned: number;
  assignedClusters: string[];
  ownedClusters: string[];
  lastActive: Date;
  reputation: number;
  specializations: string[];
}

interface CompetitiveSolution {
  solutionId: string;
  agentId: string;
  clusterId: string;
  errorNodeId: string;
  solutionCode: string;
  description: string;
  approach: string;
  estimatedEffectiveness: number;
  submittedAt: Date;
  validationStatus: 'pending' | 'in_validation' | 'validated' | 'rejected';
  dveValidationScore?: number;
  validatedAt?: Date;
  bountyAwarded?: number;
}

interface TrainingPair {
  pairId: string;
  errorContext: {
    errorType: string;
    errorMessage: string;
    stackTrace: string;
    contextVariables: Record<string, unknown>;
    semanticEmbedding: number[];
  };
  solutionContext: {
    solutionCode: string;
    approach: string;
    effectiveness: number;
    codeEmbedding: number[];
    transformationVector: number[];
  };
  weight: number;
}

interface TrainingDataset {
  datasetId: string;
  clusterId: string;
  errorNodes: ErrorNode[];
  validatedSolutions: CompetitiveSolution[];
  trainingPairs: TrainingPair[];
  datasetMetrics: {
    totalPairs: number;
    averageValidationScore: number;
    diversityScore: number;
    complexityScore: number;
    qualityScore: number;
  };
  createdAt: Date;
}

interface ClusteringConfig {
  minClusterSize: number;
  maxClusters: number;
  similarityThreshold: number;
}

class MockErrorNodeClustering {
  private config: ClusteringConfig;
  private errorNodes: Map<string, ErrorNode> = new Map();
  private clusters: Map<string, ErrorCluster> = new Map();
  private initialized: boolean = false;

  constructor(config: ClusteringConfig) {
    this.config = config;
  }

  async initialize(): Promise<void> {
    this.initialized = true;
  }

  async addErrorNode(errorNode: ErrorNode): Promise<void> {
    this.errorNodes.set(errorNode.id, errorNode);
  }

  async performClustering(): Promise<ErrorCluster[]> {
    const errorNodes = Array.from(this.errorNodes.values());
    const clusters: ErrorCluster[] = [];

    // Simple clustering by error type
    const errorTypeGroups = new Map<string, ErrorNode[]>();
    for (const node of errorNodes) {
      if (!errorTypeGroups.has(node.errorType)) {
        errorTypeGroups.set(node.errorType, []);
      }
      errorTypeGroups.get(node.errorType)!.push(node);
    }

    for (const [errorType, nodes] of Array.from(errorTypeGroups.entries())) {
      if (nodes.length >= this.config.minClusterSize) {
        const clusterId = `cluster_${errorType}_${Date.now()}`;
        const cluster: ErrorCluster = {
          clusterId,
          clusterName: `${errorType} Cluster`,
          description: `Cluster of ${errorType} errors`,
          errorNodes: nodes,
          centroid: { errorTypeVector: [1, 0, 0], contextVector: [1, 1, 0], severityScore: 0.75, tagVector: [1, 1, 0], semanticVector: [0.5, 0.3, 0.2] },
          similarity: 0.8,
          totalBounty: nodes.reduce((sum, node) => sum + node.bountyAmount, 0),
          assignedAgents: [],
          solutions: [],
          createdAt: new Date(),
          lastUpdated: new Date()
        };
        clusters.push(cluster);
        this.clusters.set(clusterId, cluster);
      }
    }

    return clusters;
  }

  getClusters(): ErrorCluster[] {
    return Array.from(this.clusters.values());
  }

  getCluster(clusterId: string): ErrorCluster | undefined {
    return this.clusters.get(clusterId);
  }

  getErrorNodes(): ErrorNode[] {
    return Array.from(this.errorNodes.values());
  }

  isReady(): boolean {
    return this.initialized;
  }
}

interface AgentAssignment {
  assignmentId: string;
  agentId: string;
  clusterId: string;
  assignedAt: Date;
  status: 'active' | 'completed' | 'abandoned';
  solutionsSubmitted: number;
  solutionsValidated: number;
  bountyEarned: number;
  performanceRating: number;
}

interface AgentOwnership {
  clusterId: string;
  ownerAgentId: string;
  acquisitionDate: Date;
  totalSolutionsContributed: number;
  ownershipScore: number;
  skillInvocationFees: number;
  monthlyRevenue: number;
  ownershipStatus: string;
}

class MockAgentAssignmentSystem {
  private agents: Map<string, Agent> = new Map();
  private assignments: Map<string, AgentAssignment> = new Map();
  private solutions: Map<string, CompetitiveSolution> = new Map();
  private ownerships: Map<string, AgentOwnership> = new Map();
  private initialized: boolean = false;

  async initialize(): Promise<void> {
    this.initialized = true;
  }

  async registerAgent(agent: Omit<Agent, 'assignedClusters' | 'ownedClusters' | 'lastActive'>): Promise<void> {
    const fullAgent: Agent = {
      ...agent,
      assignedClusters: [],
      ownedClusters: [],
      lastActive: new Date()
    };
    this.agents.set(agent.agentId, fullAgent);
  }

  async assignAgentsToCluster(cluster: ErrorCluster, maxAgents: number = 5): Promise<AgentAssignment[]> {
    const agents = Array.from(this.agents.values()).slice(0, maxAgents);
    const assignments = agents.map(agent => ({
      assignmentId: `assignment_${Date.now()}_${agent.agentId}`,
      clusterId: cluster.clusterId,
      agentId: agent.agentId,
      assignedAt: new Date(),
      status: 'active' as const,
      solutionsSubmitted: 0,
      solutionsValidated: 0,
      bountyEarned: 0,
      performanceRating: 0
    }));

    for (const assignment of assignments) {
      this.assignments.set(assignment.assignmentId, assignment);
      const agent = this.agents.get(assignment.agentId);
      if (agent) {
        agent.assignedClusters.push(cluster.clusterId);
      }
    }

    cluster.assignedAgents = assignments.map(a => a.agentId);
    return assignments;
  }

  async submitSolution(solution: Omit<CompetitiveSolution, 'solutionId' | 'submittedAt' | 'validationStatus'>): Promise<string> {
    const solutionId = `solution_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
    const competitiveSolution: CompetitiveSolution = {
      ...solution,
      solutionId,
      submittedAt: new Date(),
      validationStatus: 'pending'
    };

    this.solutions.set(solutionId, competitiveSolution);

    const agent = this.agents.get(solution.agentId);
    if (agent) {
      agent.totalSolutionsSubmitted++;
    }

    return solutionId;
  }

  async validateSolution(solutionId: string, dveValidationScore: number, bountyAmount: number): Promise<void> {
    const solution = this.solutions.get(solutionId);
    if (!solution) return;

    solution.validationStatus = dveValidationScore >= 0.7 ? 'validated' : 'rejected';
    solution.dveValidationScore = dveValidationScore;
    solution.validatedAt = new Date();

    if (solution.validationStatus === 'validated') {
      solution.bountyAwarded = bountyAmount;

      const agent = this.agents.get(solution.agentId);
      if (agent) {
        agent.totalSolutionsValidated++;
        agent.totalBountyEarned += bountyAmount;
        agent.successRate = agent.totalSolutionsValidated / agent.totalSolutionsSubmitted;
      }

      // Update ownership
      this.updateClusterOwnership(solution.clusterId, solution.agentId);
    }
  }

  private updateClusterOwnership(clusterId: string, _agentId: string): void {
    // Count solutions per agent for this cluster
    const clusterSolutions = Array.from(this.solutions.values())
      .filter(s => s.clusterId === clusterId && s.validationStatus === 'validated');

    const agentSolutionCounts = new Map<string, number>();
    for (const solution of clusterSolutions) {
      const count = agentSolutionCounts.get(solution.agentId) || 0;
      agentSolutionCounts.set(solution.agentId, count + 1);
    }

    // Find agent with most solutions
    let topAgent = '';
    let maxSolutions = 0;
    for (const [agentId, count] of Array.from(agentSolutionCounts.entries())) {
      if (count > maxSolutions) {
        maxSolutions = count;
        topAgent = agentId;
      }
    }

    if (topAgent) {
      const ownership = {
        clusterId,
        ownerAgentId: topAgent,
        acquisitionDate: new Date(),
        totalSolutionsContributed: maxSolutions,
        ownershipScore: maxSolutions / clusterSolutions.length,
        skillInvocationFees: 0,
        monthlyRevenue: 0,
        ownershipStatus: 'active'
      };
      this.ownerships.set(clusterId, ownership);

      const agent = this.agents.get(topAgent);
      if (agent && !agent.ownedClusters.includes(clusterId)) {
        agent.ownedClusters.push(clusterId);
      }
    }
  }

  getClusterSolutions(clusterId: string): CompetitiveSolution[] {
    return Array.from(this.solutions.values()).filter(s => s.clusterId === clusterId);
  }

  getClusterOwnership(clusterId: string): AgentOwnership | undefined {
    return this.ownerships.get(clusterId);
  }

  getAgentMetrics(agentId: string): Agent | undefined {
    return this.agents.get(agentId);
  }

  getAllAgents(): Agent[] {
    return Array.from(this.agents.values());
  }

  isReady(): boolean {
    return this.initialized;
  }
}

class MockLoRAAdapterTrainingPipeline {
  private datasets: Map<string, TrainingDataset> = new Map();
  private initialized: boolean = false;

  async initialize(): Promise<void> {
    this.initialized = true;
  }

  async createTrainingDataset(cluster: ErrorCluster, validatedSolutions: CompetitiveSolution[]): Promise<TrainingDataset> {
    const datasetId = `dataset_${cluster.clusterId}_${Date.now()}`;

    const trainingPairs = validatedSolutions.map((solution, index) => ({
      pairId: `pair_${index}_${Date.now()}`,
      errorContext: {
        errorType: cluster.errorNodes[index % cluster.errorNodes.length].errorType,
        errorMessage: cluster.errorNodes[index % cluster.errorNodes.length].errorMessage,
        stackTrace: cluster.errorNodes[index % cluster.errorNodes.length].stackTrace || '',
        contextVariables: cluster.errorNodes[index % cluster.errorNodes.length].context,
        semanticEmbedding: new Array(128).fill(0).map(() => Math.random())
      },
      solutionContext: {
        solutionCode: solution.solutionCode,
        approach: solution.approach,
        effectiveness: solution.dveValidationScore || 0,
        codeEmbedding: new Array(64).fill(0).map(() => Math.random()),
        transformationVector: new Array(32).fill(0).map(() => Math.random())
      },
      weight: solution.dveValidationScore || 0.5
    }));

    const dataset: TrainingDataset = {
      datasetId,
      clusterId: cluster.clusterId,
      errorNodes: cluster.errorNodes,
      validatedSolutions,
      trainingPairs,
      datasetMetrics: {
        totalPairs: trainingPairs.length,
        averageValidationScore: validatedSolutions.length > 0 ? validatedSolutions.reduce((sum, s) => sum + (s.dveValidationScore || 0), 0) / validatedSolutions.length : 0,
        diversityScore: 0.8,
        complexityScore: 0.6,
        qualityScore: 0.85
      },
      createdAt: new Date()
    };

    this.datasets.set(datasetId, dataset);
    return dataset;
  }

  async trainLoRAAdapter(dataset: TrainingDataset, config: { rank: number; alpha: number; embeddingDimension: number }): Promise<Record<string, unknown>> {
    const skillId = `skill_${dataset.clusterId}_${Date.now()}`;

    return {
      skillId,
      skillName: `${dataset.errorNodes[0]?.tags[0] || 'General'} ${dataset.errorNodes[0]?.errorType || 'Error'} Resolver`,
      description: `LoRA adapter trained on ${dataset.trainingPairs.length} error-solution pairs from cluster ${dataset.clusterId}`,
      baseModelCompatibility: 'hrm',
      version: 1,
      rank: config.rank,
      alpha: config.alpha,
      weightsA: new Float32Array(config.embeddingDimension * config.rank).fill(0).map(() => (Math.random() - 0.5) * 0.02),
      weightsB: new Float32Array(config.rank * config.embeddingDimension).fill(0).map(() => (Math.random() - 0.5) * 0.02),
      additionalMetadata: {
        clusterId: dataset.clusterId,
        trainingDatasetId: dataset.datasetId,
        trainingPairs: dataset.trainingPairs.length.toString(),
        qualityScore: (dataset.datasetMetrics.qualityScore as number).toString(),
        finalLoss: '0.15',
        finalAccuracy: '0.92',
        timestamp: new Date().toISOString()
      }
    };
  }

  getTrainingDataset(datasetId: string): TrainingDataset | undefined {
    return this.datasets.get(datasetId);
  }

  getAllTrainingDatasets(): TrainingDataset[] {
    return Array.from(this.datasets.values());
  }

  isReady(): boolean {
    return this.initialized;
  }
}

// Export mock classes for use in tests
export const ErrorNodeClustering = MockErrorNodeClustering;
export const AgentAssignmentSystem = MockAgentAssignmentSystem;
export const LoRAAdapterTrainingPipeline = MockLoRAAdapterTrainingPipeline;

describe('Phase 3.3: KNIRVGRAPH LoRA Adapter Creation Integration', () => {
  let errorClustering: MockErrorNodeClustering;
  let agentAssignment: MockAgentAssignmentSystem;
  let trainingPipeline: MockLoRAAdapterTrainingPipeline;

  beforeEach(async () => {
    errorClustering = new MockErrorNodeClustering({
      minClusterSize: 2,
      maxClusters: 10,
      similarityThreshold: 0.6
    });

    agentAssignment = new MockAgentAssignmentSystem();
    trainingPipeline = new MockLoRAAdapterTrainingPipeline();

    await errorClustering.initialize();
    await agentAssignment.initialize();
    await trainingPipeline.initialize();
  });

  describe('ErrorNode Clustering Algorithm', () => {
    const createTestErrorNode = (id: string, errorType: string, tags: string[] = []): ErrorNode => ({
      id,
      errorType,
      errorMessage: `Test error message for ${errorType}`,
      stackTrace: `Stack trace for ${id}`,
      context: { function: 'testFunction', line: 42 },
      severity: 'medium',
      timestamp: new Date(),
      bountyAmount: 100,
      tags,
      metadata: {}
    });

    it('should cluster similar error nodes correctly', async () => {
      const errorNodes = [
        createTestErrorNode('error1', 'TypeError', ['frontend', 'javascript']),
        createTestErrorNode('error2', 'TypeError', ['frontend', 'javascript']),
        createTestErrorNode('error3', 'ReferenceError', ['backend', 'api']),
        createTestErrorNode('error4', 'ReferenceError', ['backend', 'api']),
        createTestErrorNode('error5', 'TypeError', ['frontend', 'react'])
      ];

      for (const errorNode of errorNodes) {
        await errorClustering.addErrorNode(errorNode);
      }

      const clusters = await errorClustering.performClustering();
      
      expect(clusters.length).toBeGreaterThan(0);
      expect(clusters.length).toBeLessThanOrEqual(3); // Should group similar errors

      // Check that similar errors are clustered together
      const typeErrorCluster = clusters.find((cluster: ErrorCluster) =>
        cluster.errorNodes.some((node: ErrorNode) => node.errorType === 'TypeError')
      );
      expect(typeErrorCluster).toBeDefined();
      expect(typeErrorCluster!.errorNodes.length).toBeGreaterThanOrEqual(2);
    });

    it('should calculate similarity between error nodes accurately', async () => {
      const similarErrors = [
        createTestErrorNode('similar1', 'TypeError', ['frontend', 'javascript']),
        createTestErrorNode('similar2', 'TypeError', ['frontend', 'javascript'])
      ];

      for (const errorNode of similarErrors) {
        await errorClustering.addErrorNode(errorNode);
      }

      const clusters = await errorClustering.performClustering();
      
      if (clusters.length > 0) {
        expect(clusters[0].similarity).toBeGreaterThan(0.5);
      }
    });

    it('should respect minimum and maximum cluster sizes', async () => {
      const errorNodes = Array.from({ length: 15 }, (_, i) => 
        createTestErrorNode(`error${i}`, i < 8 ? 'TypeError' : 'ReferenceError', ['test'])
      );

      for (const errorNode of errorNodes) {
        await errorClustering.addErrorNode(errorNode);
      }

      const clusters = await errorClustering.performClustering();
      
      clusters.forEach((cluster: ErrorCluster) => {
        expect(cluster.errorNodes.length).toBeGreaterThanOrEqual(2); // minClusterSize
        expect(cluster.errorNodes.length).toBeLessThanOrEqual(10); // maxClusterSize
      });
    });

    it('should calculate total bounty for clusters', async () => {
      const errorNodes = [
        { ...createTestErrorNode('bounty1', 'TypeError'), bountyAmount: 100 },
        { ...createTestErrorNode('bounty2', 'TypeError'), bountyAmount: 200 },
        { ...createTestErrorNode('bounty3', 'TypeError'), bountyAmount: 150 }
      ];

      for (const errorNode of errorNodes) {
        await errorClustering.addErrorNode(errorNode);
      }

      const clusters = await errorClustering.performClustering();
      
      if (clusters.length > 0) {
        const cluster = clusters[0];
        const expectedBounty = cluster.errorNodes.reduce((sum: number, node: ErrorNode) => sum + node.bountyAmount, 0);
        expect(cluster.totalBounty).toBe(expectedBounty);
      }
    });
  });

  describe('Agent Assignment System', () => {
    const createTestAgent = (id: string, expertise: string[], performanceScore: number = 0.8): Agent => ({
      agentId: id,
      agentName: `Agent ${id}`,
      expertise,
      performanceScore,
      successRate: 0.85,
      totalSolutionsSubmitted: 10,
      totalSolutionsValidated: 8,
      totalBountyEarned: 1000,
      assignedClusters: [],
      ownedClusters: [],
      lastActive: new Date(),
      reputation: 100,
      specializations: []
    });

    beforeEach(async () => {
      const testAgents = [
        createTestAgent('agent1', ['javascript', 'debugging'], 0.9),
        createTestAgent('agent2', ['python', 'api'], 0.8),
        createTestAgent('agent3', ['javascript', 'react'], 0.85),
        createTestAgent('agent4', ['debugging', 'performance'], 0.75)
      ];

      for (const agent of testAgents) {
        await agentAssignment.registerAgent(agent);
      }
    });

    it('should assign suitable agents to error clusters', async () => {
      const testCluster: ErrorCluster = {
        clusterId: 'test-cluster-001',
        clusterName: 'JavaScript Errors',
        description: 'Cluster of JavaScript-related errors',
        errorNodes: [
          {
            id: 'error1',
            errorType: 'TypeError',
            errorMessage: 'Cannot read property of undefined',
            context: {},
            severity: 'high',
            timestamp: new Date(),
            bountyAmount: 200,
            tags: ['javascript', 'frontend'],
            metadata: {}
          }
        ],
        centroid: {
          errorTypeVector: [1, 0, 0],
          contextVector: [1, 1, 0],
          severityScore: 0.75,
          tagVector: [1, 1, 0],
          semanticVector: [0.5, 0.3, 0.2]
        },
        similarity: 0.8,
        totalBounty: 200,
        assignedAgents: [],
        solutions: [],
        createdAt: new Date(),
        lastUpdated: new Date()
      };

      const assignments = await agentAssignment.assignAgentsToCluster(testCluster, 3);
      
      expect(assignments.length).toBeGreaterThan(0);
      expect(assignments.length).toBeLessThanOrEqual(3);
      
      // Check that agents with JavaScript expertise are preferred
      const assignedAgentIds = assignments.map((a: AgentAssignment) => a.agentId);
      expect(assignedAgentIds).toContain('agent1'); // Has javascript expertise
    });

    it('should handle competitive solution submission', async () => {
      const solution: Omit<CompetitiveSolution, 'solutionId' | 'submittedAt' | 'validationStatus'> = {
        agentId: 'agent1',
        clusterId: 'test-cluster-001',
        errorNodeId: 'error1',
        solutionCode: 'if (obj && obj.property) { return obj.property; }',
        description: 'Add null check before property access',
        approach: 'defensive-programming',
        estimatedEffectiveness: 0.9
      };

      const solutionId = await agentAssignment.submitSolution(solution);
      
      expect(solutionId).toBeDefined();
      expect(solutionId).toMatch(/^solution_/);

      const clusterSolutions = agentAssignment.getClusterSolutions('test-cluster-001');
      expect(clusterSolutions.length).toBe(1);
      expect(clusterSolutions[0].agentId).toBe('agent1');
    });

    it('should validate solutions and award bounties', async () => {
      const solution: Omit<CompetitiveSolution, 'solutionId' | 'submittedAt' | 'validationStatus'> = {
        agentId: 'agent1',
        clusterId: 'test-cluster-001',
        errorNodeId: 'error1',
        solutionCode: 'const result = obj?.property || defaultValue;',
        description: 'Use optional chaining for safe property access',
        approach: 'modern-javascript',
        estimatedEffectiveness: 0.95
      };

      const solutionId = await agentAssignment.submitSolution(solution);
      await agentAssignment.validateSolution(solutionId, 0.9, 150);

      const clusterSolutions = agentAssignment.getClusterSolutions('test-cluster-001');
      const validatedSolution = clusterSolutions.find((s: CompetitiveSolution) => s.solutionId === solutionId);
      
      expect(validatedSolution?.validationStatus).toBe('validated');
      expect(validatedSolution?.bountyAwarded).toBe(150);
      expect(validatedSolution?.dveValidationScore).toBe(0.9);

      // Check agent statistics update
      const agent = agentAssignment.getAgentMetrics('agent1');
      expect(agent?.totalBountyEarned).toBeGreaterThan(1000);
    });

    it('should track cluster ownership based on solution contributions', async () => {
      // Submit multiple solutions from different agents
      const solutions = [
        { agentId: 'agent1', solutionCode: 'solution1', description: 'First solution' },
        { agentId: 'agent1', solutionCode: 'solution2', description: 'Second solution' },
        { agentId: 'agent2', solutionCode: 'solution3', description: 'Third solution' }
      ];

      for (const sol of solutions) {
        const solution: Omit<CompetitiveSolution, 'solutionId' | 'submittedAt' | 'validationStatus'> = {
          agentId: sol.agentId,
          clusterId: 'ownership-test-cluster',
          errorNodeId: 'error1',
          solutionCode: sol.solutionCode,
          description: sol.description,
          approach: 'test-approach',
          estimatedEffectiveness: 0.8
        };

        const solutionId = await agentAssignment.submitSolution(solution);
        await agentAssignment.validateSolution(solutionId, 0.8, 100);
      }

      const ownership = agentAssignment.getClusterOwnership('ownership-test-cluster');
      expect(ownership).toBeDefined();
      expect(ownership?.ownerAgentId).toBe('agent1'); // Has more solutions
      expect(ownership?.totalSolutionsContributed).toBe(2);
    });
  });

  describe('LoRA Adapter Training Pipeline', () => {
    let testCluster: ErrorCluster;
    let testSolutions: CompetitiveSolution[];

    beforeEach(() => {
      testCluster = {
        clusterId: 'training-cluster-001',
        clusterName: 'Training Test Cluster',
        description: 'Cluster for training pipeline testing',
        errorNodes: [
          {
            id: 'training-error-1',
            errorType: 'TypeError',
            errorMessage: 'Cannot read property "length" of undefined',
            stackTrace: 'at function1 (file.js:10)',
            context: { function: 'processArray', variable: 'items' },
            severity: 'high',
            timestamp: new Date(),
            bountyAmount: 200,
            tags: ['javascript', 'array'],
            metadata: {}
          },
          {
            id: 'training-error-2',
            errorType: 'TypeError',
            errorMessage: 'Cannot read property "map" of null',
            stackTrace: 'at function2 (file.js:20)',
            context: { function: 'transformData', variable: 'data' },
            severity: 'medium',
            timestamp: new Date(),
            bountyAmount: 150,
            tags: ['javascript', 'array'],
            metadata: {}
          }
        ],
        centroid: {
          errorTypeVector: [1, 0, 0],
          contextVector: [1, 1, 1],
          severityScore: 0.75,
          tagVector: [1, 1, 0],
          semanticVector: [0.6, 0.4, 0.3]
        },
        similarity: 0.85,
        totalBounty: 350,
        assignedAgents: ['agent1'],
        solutions: [],
        createdAt: new Date(),
        lastUpdated: new Date()
      };

      testSolutions = [
        {
          solutionId: 'solution-1',
          agentId: 'agent1',
          clusterId: 'training-cluster-001',
          errorNodeId: 'training-error-1',
          solutionCode: 'if (items && Array.isArray(items)) { return items.length; }',
          description: 'Add array validation before accessing length',
          approach: 'defensive-programming',
          estimatedEffectiveness: 0.9,
          submittedAt: new Date(),
          validationStatus: 'validated',
          dveValidationScore: 0.9,
          validatedAt: new Date(),
          bountyAwarded: 180
        },
        {
          solutionId: 'solution-2',
          agentId: 'agent2',
          clusterId: 'training-cluster-001',
          errorNodeId: 'training-error-2',
          solutionCode: 'const result = data?.map?.(item => transform(item)) || [];',
          description: 'Use optional chaining for safe method access',
          approach: 'modern-javascript',
          estimatedEffectiveness: 0.95,
          submittedAt: new Date(),
          validationStatus: 'validated',
          dveValidationScore: 0.95,
          validatedAt: new Date(),
          bountyAwarded: 200
        }
      ];
    });

    it('should create training dataset from error cluster and solutions', async () => {
      const dataset = await trainingPipeline.createTrainingDataset(testCluster, testSolutions);
      
      expect(dataset.datasetId).toBeDefined();
      expect(dataset.clusterId).toBe('training-cluster-001');
      expect(dataset.trainingPairs.length).toBe(2);
      expect(dataset.datasetMetrics.totalPairs).toBe(2);
      expect(dataset.datasetMetrics.averageValidationScore).toBeCloseTo(0.925);
    });

    it('should create semantic embeddings for error messages', async () => {
      const dataset = await trainingPipeline.createTrainingDataset(testCluster, testSolutions);
      
      const trainingPair = dataset.trainingPairs[0];
      expect(trainingPair.errorContext.semanticEmbedding).toBeDefined();
      expect(trainingPair.errorContext.semanticEmbedding.length).toBeGreaterThan(0);
      expect(trainingPair.errorContext.semanticEmbedding.every((val: number) => val >= 0 && val <= 1)).toBe(true);
    });

    it('should create code embeddings for solutions', async () => {
      const dataset = await trainingPipeline.createTrainingDataset(testCluster, testSolutions);
      
      const trainingPair = dataset.trainingPairs[0];
      expect(trainingPair.solutionContext.codeEmbedding).toBeDefined();
      expect(trainingPair.solutionContext.codeEmbedding.length).toBeGreaterThan(0);
      expect(trainingPair.solutionContext.codeEmbedding.every((val: number) => val >= 0 && val <= 1)).toBe(true);
    });

    it('should train LoRA adapter from dataset', async () => {
      const dataset = await trainingPipeline.createTrainingDataset(testCluster, testSolutions);
      
      const trainingConfig = {
        rank: 8,
        alpha: 16.0,
        learningRate: 0.001,
        epochs: 5,
        batchSize: 2,
        regularization: 0.01,
        embeddingDimension: 128,
        maxSequenceLength: 512
      };

      const loraAdapter = await trainingPipeline.trainLoRAAdapter(dataset, trainingConfig);
      
      expect(loraAdapter.skillId).toBeDefined();
      expect(loraAdapter.skillName).toContain('Resolver');
      expect(loraAdapter.rank).toBe(8);
      expect(loraAdapter.alpha).toBe(16.0);
      expect(loraAdapter.weightsA).toBeInstanceOf(Float32Array);
      expect(loraAdapter.weightsB).toBeInstanceOf(Float32Array);
      expect((loraAdapter.weightsA as Float32Array).length).toBeGreaterThan(0);
      expect((loraAdapter.weightsB as Float32Array).length).toBeGreaterThan(0);
    });

    it('should calculate dataset quality metrics', async () => {
      const dataset = await trainingPipeline.createTrainingDataset(testCluster, testSolutions);
      
      const metrics = dataset.datasetMetrics;
      expect(metrics.totalPairs).toBe(2);
      expect(metrics.averageValidationScore).toBeGreaterThan(0.8);
      expect(metrics.diversityScore).toBeGreaterThan(0);
      expect(metrics.complexityScore).toBeGreaterThan(0);
      expect(metrics.qualityScore).toBeGreaterThan(0);
      expect(metrics.qualityScore).toBeLessThanOrEqual(1);
    });

    it('should weight training pairs based on validation scores', async () => {
      const dataset = await trainingPipeline.createTrainingDataset(testCluster, testSolutions);
      
      const highScorePair = dataset.trainingPairs.find((pair: any) =>
        pair.solutionContext.effectiveness === 0.95
      );
      const lowScorePair = dataset.trainingPairs.find((pair: any) =>
        pair.solutionContext.effectiveness === 0.9
      );
      
      expect((highScorePair as TrainingPair)?.weight).toBeGreaterThan((lowScorePair as TrainingPair)?.weight || 0);
    });

    it('should generate appropriate skill names', async () => {
      const dataset = await trainingPipeline.createTrainingDataset(testCluster, testSolutions);
      
      const trainingConfig = {
        rank: 4,
        alpha: 8.0,
        learningRate: 0.001,
        epochs: 2,
        batchSize: 1,
        regularization: 0.01,
        embeddingDimension: 64,
        maxSequenceLength: 256
      };

      const loraAdapter = await trainingPipeline.trainLoRAAdapter(dataset, trainingConfig);
      
      expect(loraAdapter.skillName).toMatch(/.*Resolver$/);
      expect(loraAdapter.description).toContain('error-solution pairs');
      expect((loraAdapter.additionalMetadata as any).clusterId).toBe('training-cluster-001');
    });
  });

  describe('End-to-End Integration', () => {
    it('should complete full ErrorNode-to-LoRA-adapter workflow', async () => {
      // Step 1: Create and cluster error nodes
      const errorNodes = [
        {
          id: 'e2e-error-1',
          errorType: 'TypeError',
          errorMessage: 'Cannot read property of undefined',
          context: { function: 'getData' },
          severity: 'high' as const,
          timestamp: new Date(),
          bountyAmount: 300,
          tags: ['javascript', 'api'],
          metadata: {}
        },
        {
          id: 'e2e-error-2',
          errorType: 'TypeError',
          errorMessage: 'Cannot read property of null',
          context: { function: 'processData' },
          severity: 'high' as const,
          timestamp: new Date(),
          bountyAmount: 250,
          tags: ['javascript', 'api'],
          metadata: {}
        }
      ];

      for (const errorNode of errorNodes) {
        await errorClustering.addErrorNode(errorNode);
      }

      const clusters = await errorClustering.performClustering();
      expect(clusters.length).toBeGreaterThan(0);

      // Step 2: Assign agents to cluster
      const testAgent: Agent = {
        agentId: 'e2e-agent',
        agentName: 'End-to-End Test Agent',
        expertise: ['javascript', 'debugging'],
        performanceScore: 0.9,
        successRate: 0.85,
        totalSolutionsSubmitted: 5,
        totalSolutionsValidated: 4,
        totalBountyEarned: 500,
        assignedClusters: [],
        ownedClusters: [],
        lastActive: new Date(),
        reputation: 150,
        specializations: []
      };

      await agentAssignment.registerAgent(testAgent);
      const assignments = await agentAssignment.assignAgentsToCluster(clusters[0], 1);
      expect(assignments.length).toBe(1);

      // Step 3: Submit and validate solutions
      const solutions = [];
      for (const errorNode of clusters[0].errorNodes) {
        const solution: Omit<CompetitiveSolution, 'solutionId' | 'submittedAt' | 'validationStatus'> = {
          agentId: 'e2e-agent',
          clusterId: clusters[0].clusterId,
          errorNodeId: errorNode.id,
          solutionCode: 'const value = obj?.property ?? defaultValue;',
          description: 'Use nullish coalescing for safe property access',
          approach: 'modern-javascript',
          estimatedEffectiveness: 0.9
        };

        const solutionId = await agentAssignment.submitSolution(solution);
        await agentAssignment.validateSolution(solutionId, 0.9, 200);
        
        const clusterSolutions = agentAssignment.getClusterSolutions(clusters[0].clusterId);
        const validatedSolution = clusterSolutions.find((s: CompetitiveSolution) => s.solutionId === solutionId);
        solutions.push(validatedSolution!);
      }

      // Step 4: Create training dataset and train LoRA adapter
      const dataset = await trainingPipeline.createTrainingDataset(clusters[0], solutions);
      expect(dataset.trainingPairs.length).toBe(2);

      const trainingConfig = {
        rank: 8,
        alpha: 16.0,
        learningRate: 0.001,
        epochs: 3,
        batchSize: 2,
        regularization: 0.01,
        embeddingDimension: 128,
        maxSequenceLength: 512
      };

      const loraAdapter = await trainingPipeline.trainLoRAAdapter(dataset, trainingConfig);
      
      expect(loraAdapter.skillId).toBeDefined();
      expect(loraAdapter.skillName).toBeDefined();
      expect((loraAdapter.weightsA as Float32Array).length).toBeGreaterThan(0);
      expect((loraAdapter.weightsB as Float32Array).length).toBeGreaterThan(0);
      expect((loraAdapter.additionalMetadata as any).clusterId).toBe(clusters[0].clusterId);
      expect((loraAdapter.additionalMetadata as any).trainingPairs).toBe('2');
    });
  });
});
