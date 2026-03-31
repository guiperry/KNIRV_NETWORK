/**
 * Agent Assignment System for KNIRVGRAPH Error Clusters
 * 
 * Manages agent assignment to error clusters based on expertise and performance
 * Enables competitive solution development and cluster ownership tracking
 */

import pino from 'pino';
import { ErrorCluster} from './ErrorNodeClustering';

const logger = pino({ name: 'agent-assignment-system' });

export interface Agent {
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
  specializations: AgentSpecialization[];
}

export interface AgentSpecialization {
  domain: string;
  proficiencyLevel: number; // 0-1
  experiencePoints: number;
  lastUsed: Date;
}

export interface ClusterAssignment {
  assignmentId: string;
  clusterId: string;
  agentId: string;
  assignedAt: Date;
  status: 'active' | 'completed' | 'abandoned';
  solutionsSubmitted: number;
  solutionsValidated: number;
  bountyEarned: number;
  performanceRating: number;
}

export interface CompetitiveSolution {
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
  competitorRanking?: number;
}

export interface ClusterOwnership {
  clusterId: string;
  ownerAgentId: string;
  acquisitionDate: Date;
  totalSolutionsContributed: number;
  ownershipScore: number;
  skillInvocationFees: number;
  monthlyRevenue: number;
  ownershipStatus: 'active' | 'challenged' | 'transferred';
}

export class AgentAssignmentSystem {
  private agents: Map<string, Agent> = new Map();
  private assignments: Map<string, ClusterAssignment> = new Map();
  private solutions: Map<string, CompetitiveSolution> = new Map();
  private ownerships: Map<string, ClusterOwnership> = new Map();
  private isInitialized: boolean = false;

  constructor() {}

  /**
   * Initialize the agent assignment system
   */
  async initialize(): Promise<void> {
    logger.info('Initializing Agent Assignment System...');
    
    this.isInitialized = true;
    logger.info('Agent Assignment System initialized successfully');
  }

  /**
   * Register a new agent
   */
  async registerAgent(agent: Omit<Agent, 'assignedClusters' | 'ownedClusters' | 'lastActive'>): Promise<void> {
    if (!this.isInitialized) {
      throw new Error('Agent Assignment System not initialized');
    }

    const fullAgent: Agent = {
      ...agent,
      assignedClusters: [],
      ownedClusters: [],
      lastActive: new Date()
    };

    this.agents.set(agent.agentId, fullAgent);
    logger.info({ agentId: agent.agentId }, 'Agent registered successfully');
  }

  /**
   * Assign agents to error cluster based on expertise and performance
   */
  async assignAgentsToCluster(cluster: ErrorCluster, maxAgents: number = 5): Promise<ClusterAssignment[]> {
    logger.info({ clusterId: cluster.clusterId }, 'Assigning agents to cluster...');

    // Analyze cluster requirements
    const clusterRequirements = this.analyzeClusterRequirements(cluster);
    
    // Find suitable agents
    const suitableAgents = this.findSuitableAgents(clusterRequirements, maxAgents);
    
    // Create assignments
    const assignments: ClusterAssignment[] = [];
    
    for (const agent of suitableAgents) {
      const assignmentId = `assignment_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
      
      const assignment: ClusterAssignment = {
        assignmentId,
        clusterId: cluster.clusterId,
        agentId: agent.agentId,
        assignedAt: new Date(),
        status: 'active',
        solutionsSubmitted: 0,
        solutionsValidated: 0,
        bountyEarned: 0,
        performanceRating: 0
      };

      this.assignments.set(assignmentId, assignment);
      
      // Update agent's assigned clusters
      agent.assignedClusters.push(cluster.clusterId);
      agent.lastActive = new Date();
      
      assignments.push(assignment);
    }

    // Update cluster with assigned agents
    cluster.assignedAgents = assignments.map(a => a.agentId);
    cluster.lastUpdated = new Date();

    logger.info({ 
      clusterId: cluster.clusterId, 
      assignedAgents: assignments.length 
    }, 'Agents assigned to cluster successfully');

    return assignments;
  }

  /**
   * Analyze cluster requirements based on error patterns
   */
  private analyzeClusterRequirements(cluster: ErrorCluster): {
    requiredExpertise: string[];
    difficultyLevel: number;
    estimatedEffort: number;
    preferredSpecializations: string[];
  } {
    const errorTypes = [...new Set(cluster.errorNodes.map(node => node.errorType))];
    const tags = [...new Set(cluster.errorNodes.flatMap(node => node.tags))];
    const avgSeverity = cluster.errorNodes.reduce((sum, node) => {
      const severityScores = { low: 1, medium: 2, high: 3, critical: 4 };
      return sum + severityScores[node.severity];
    }, 0) / cluster.errorNodes.length;

    // Map error types to required expertise
    const expertiseMapping: Record<string, string[]> = {
      'TypeError': ['javascript', 'typescript', 'debugging'],
      'ReferenceError': ['javascript', 'typescript', 'variable-management'],
      'SyntaxError': ['javascript', 'typescript', 'code-parsing'],
      'NetworkError': ['networking', 'api', 'http'],
      'ValidationError': ['data-validation', 'input-handling', 'security']
    };

    const requiredExpertise = [...new Set(
      errorTypes.flatMap(type => expertiseMapping[type] || ['general-debugging'])
    )];

    return {
      requiredExpertise,
      difficultyLevel: avgSeverity / 4, // Normalize to 0-1
      estimatedEffort: cluster.errorNodes.length * avgSeverity,
      preferredSpecializations: tags
    };
  }

  /**
   * Find suitable agents for cluster requirements
   */
  private findSuitableAgents(requirements: {
    requiredExpertise: string[];
    difficultyLevel: number;
    estimatedEffort: number;
    preferredSpecializations: string[];
  }, maxAgents: number): Agent[] {
    const allAgents = Array.from(this.agents.values());
    
    // Score agents based on suitability
    const scoredAgents = allAgents.map(agent => {
      let score = 0;
      
      // Expertise match
      const expertiseMatch = requirements.requiredExpertise.filter(req => 
        agent.expertise.includes(req)
      ).length / requirements.requiredExpertise.length;
      score += expertiseMatch * 0.4;
      
      // Performance score
      score += agent.performanceScore * 0.3;
      
      // Success rate
      score += agent.successRate * 0.2;
      
      // Availability (fewer assigned clusters = higher availability)
      const availabilityScore = Math.max(0, 1 - (agent.assignedClusters.length / 10));
      score += availabilityScore * 0.1;

      return { agent, score };
    });

    // Sort by score and return top agents
    return scoredAgents
      .sort((a, b) => b.score - a.score)
      .slice(0, maxAgents)
      .map(item => item.agent);
  }

  /**
   * Submit competitive solution for error cluster
   */
  async submitSolution(solution: Omit<CompetitiveSolution, 'solutionId' | 'submittedAt' | 'validationStatus'>): Promise<string> {
    if (!this.isInitialized) {
      throw new Error('Agent Assignment System not initialized');
    }

    const solutionId = `solution_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
    
    const competitiveSolution: CompetitiveSolution = {
      ...solution,
      solutionId,
      submittedAt: new Date(),
      validationStatus: 'pending'
    };

    this.solutions.set(solutionId, competitiveSolution);

    // Update assignment statistics
    const assignment = Array.from(this.assignments.values()).find(
      a => a.agentId === solution.agentId && a.clusterId === solution.clusterId
    );
    
    if (assignment) {
      assignment.solutionsSubmitted++;
    }

    // Update agent statistics
    const agent = this.agents.get(solution.agentId);
    if (agent) {
      agent.totalSolutionsSubmitted++;
      agent.lastActive = new Date();
    }

    logger.info({ 
      solutionId, 
      agentId: solution.agentId, 
      clusterId: solution.clusterId 
    }, 'Competitive solution submitted');

    return solutionId;
  }

  /**
   * Validate solution through DVE system
   */
  async validateSolution(solutionId: string, dveValidationScore: number, bountyAmount: number): Promise<void> {
    const solution = this.solutions.get(solutionId);
    if (!solution) {
      throw new Error(`Solution ${solutionId} not found`);
    }

    solution.validationStatus = dveValidationScore >= 0.7 ? 'validated' : 'rejected';
    solution.dveValidationScore = dveValidationScore;
    solution.validatedAt = new Date();

    if (solution.validationStatus === 'validated') {
      solution.bountyAwarded = bountyAmount;

      // Update assignment statistics
      const assignment = Array.from(this.assignments.values()).find(
        a => a.agentId === solution.agentId && a.clusterId === solution.clusterId
      );
      
      if (assignment) {
        assignment.solutionsValidated++;
        assignment.bountyEarned += bountyAmount;
      }

      // Update agent statistics
      const agent = this.agents.get(solution.agentId);
      if (agent) {
        agent.totalSolutionsValidated++;
        agent.totalBountyEarned += bountyAmount;
        agent.successRate = agent.totalSolutionsValidated / agent.totalSolutionsSubmitted;
        agent.performanceScore = Math.min(1.0, agent.performanceScore + 0.01); // Incremental improvement
        agent.reputation += Math.floor(bountyAmount / 100); // Reputation based on bounty
      }

      // Check for cluster ownership
      await this.updateClusterOwnership(solution.clusterId, solution.agentId);
    }

    logger.info({ 
      solutionId, 
      validationStatus: solution.validationStatus, 
      bountyAwarded: solution.bountyAwarded 
    }, 'Solution validation completed');
  }

  /**
   * Update cluster ownership based on solution contributions
   */
  private async updateClusterOwnership(clusterId: string, _agentId: string): Promise<void> {
    // Count validated solutions per agent for this cluster
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
    for (const [agentId, count] of agentSolutionCounts) {
      if (count > maxSolutions) {
        maxSolutions = count;
        topAgent = agentId;
      }
    }

    // Update ownership if necessary
    const currentOwnership = this.ownerships.get(clusterId);
    if (!currentOwnership || currentOwnership.ownerAgentId !== topAgent) {
      const newOwnership: ClusterOwnership = {
        clusterId,
        ownerAgentId: topAgent,
        acquisitionDate: new Date(),
        totalSolutionsContributed: maxSolutions,
        ownershipScore: maxSolutions / clusterSolutions.length,
        skillInvocationFees: 0,
        monthlyRevenue: 0,
        ownershipStatus: 'active'
      };

      this.ownerships.set(clusterId, newOwnership);

      // Update agent's owned clusters
      const agent = this.agents.get(topAgent);
      if (agent && !agent.ownedClusters.includes(clusterId)) {
        agent.ownedClusters.push(clusterId);
      }

      logger.info({ 
        clusterId, 
        newOwner: topAgent, 
        solutionCount: maxSolutions 
      }, 'Cluster ownership updated');
    }
  }

  /**
   * Get agent assignments
   */
  getAgentAssignments(agentId: string): ClusterAssignment[] {
    return Array.from(this.assignments.values())
      .filter(assignment => assignment.agentId === agentId);
  }

  /**
   * Get cluster assignments
   */
  getClusterAssignments(clusterId: string): ClusterAssignment[] {
    return Array.from(this.assignments.values())
      .filter(assignment => assignment.clusterId === clusterId);
  }

  /**
   * Get competitive solutions for cluster
   */
  getClusterSolutions(clusterId: string): CompetitiveSolution[] {
    return Array.from(this.solutions.values())
      .filter(solution => solution.clusterId === clusterId);
  }

  /**
   * Get cluster ownership
   */
  getClusterOwnership(clusterId: string): ClusterOwnership | undefined {
    return this.ownerships.get(clusterId);
  }

  /**
   * Get agent performance metrics
   */
  getAgentMetrics(agentId: string): Agent | undefined {
    return this.agents.get(agentId);
  }

  /**
   * Get all agents
   */
  getAllAgents(): Agent[] {
    return Array.from(this.agents.values());
  }

  isReady(): boolean {
    return this.isInitialized;
  }
}
