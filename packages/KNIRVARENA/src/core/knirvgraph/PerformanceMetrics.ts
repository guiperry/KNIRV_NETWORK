/**
 * Performance Metrics Tracking for KNIRVGRAPH
 * 
 * Tracks success rates and cluster-based competition metrics
 * Provides comprehensive analytics for the LoRA adapter creation platform
 */

import pino from 'pino';
import { EventEmitter } from 'events';
import { ErrorCluster } from './ErrorNodeClustering';
import { CompetitiveSolution } from './AgentAssignmentSystem';
import { LoRAAdapterSkill } from '../lora/LoRAAdapterEngine';
import { SkillDiscoveryResult } from './HRMCoreModel';

const logger = pino({ name: 'performance-metrics' });

export interface ClusterMetrics {
  clusterId: string;
  clusterName: string;
  totalErrors: number;
  totalSolutions: number;
  validatedSolutions: number;
  successRate: number;
  averageValidationScore: number;
  totalBountyAwarded: number;
  averageSolutionTime: number;
  participatingAgents: number;
  competitionIntensity: number;
  skillsGenerated: number;
  lastUpdated: Date;
}

export interface AgentMetrics {
  agentId: string;
  totalSolutionsSubmitted: number;
  validatedSolutions: number;
  successRate: number;
  averageValidationScore: number;
  totalBountyEarned: number;
  clustersParticipated: number;
  clustersOwned: number;
  averageSolutionTime: number;
  reputation: number;
  skillsContributed: number;
  lastActive: Date;
}

export interface SkillMetrics {
  skillId: string;
  skillName: string;
  category: string;
  trainingAccuracy: number;
  validationScore: number;
  usageCount: number;
  successfulInvocations: number;
  averageInferenceTime: number;
  memoryUsage: number;
  userRating: number;
  createdAt: Date;
  lastUsed: Date;
}

export interface SystemMetrics {
  totalClusters: number;
  totalAgents: number;
  totalSkills: number;
  totalSolutions: number;
  overallSuccessRate: number;
  averageClusterCompetition: number;
  totalBountyDistributed: number;
  systemThroughput: number;
  averageSkillCreationTime: number;
  networkHealth: number;
  lastUpdated: Date;
}

export interface CompetitionMetrics {
  clusterId: string;
  competitionRound: number;
  startTime: Date;
  endTime?: Date;
  participatingAgents: string[];
  solutionsSubmitted: number;
  leaderboard: LeaderboardEntry[];
  prizePool: number;
  winnerAgent?: string;
  winningScore: number;
  competitionIntensity: number;
}

export interface LeaderboardEntry {
  rank: number;
  agentId: string;
  score: number;
  solutionsSubmitted: number;
  validatedSolutions: number;
  bountyEarned: number;
  averageTime: number;
}

export interface MetricsConfig {
  updateInterval: number;
  retentionPeriod: number;
  aggregationWindow: number;
  enableRealTimeTracking: boolean;
  enableCompetitionMetrics: boolean;
  maxLeaderboardSize: number;
}

export class PerformanceMetrics extends EventEmitter {
  private clusterMetrics: Map<string, ClusterMetrics> = new Map();
  private agentMetrics: Map<string, AgentMetrics> = new Map();
  private skillMetrics: Map<string, SkillMetrics> = new Map();
  private competitionMetrics: Map<string, CompetitionMetrics[]> = new Map();
  private systemMetrics: SystemMetrics;
  
  private config: MetricsConfig;
  private isInitialized: boolean = false;
  private updateInterval?: NodeJS.Timeout;
  private metricsHistory: Map<string, unknown[]> = new Map();

  constructor(config: Partial<MetricsConfig> = {}) {
    super();
    
    this.config = {
      updateInterval: 30000, // 30 seconds
      retentionPeriod: 7 * 24 * 60 * 60 * 1000, // 7 days
      aggregationWindow: 60 * 60 * 1000, // 1 hour
      enableRealTimeTracking: true,
      enableCompetitionMetrics: true,
      maxLeaderboardSize: 100,
      ...config
    };

    this.systemMetrics = {
      totalClusters: 0,
      totalAgents: 0,
      totalSkills: 0,
      totalSolutions: 0,
      overallSuccessRate: 0,
      averageClusterCompetition: 0,
      totalBountyDistributed: 0,
      systemThroughput: 0,
      averageSkillCreationTime: 0,
      networkHealth: 1.0,
      lastUpdated: new Date()
    };
  }

  /**
   * Initialize performance metrics tracking
   */
  async initialize(): Promise<void> {
    logger.info('Initializing Performance Metrics tracking...');

    try {
      // Start metrics update loop
      if (this.config.enableRealTimeTracking) {
        this.startMetricsUpdateLoop();
      }

      this.isInitialized = true;
      logger.info('Performance Metrics tracking initialized successfully');

    } catch (error) {
      logger.error({ error }, 'Failed to initialize Performance Metrics tracking');
      throw error;
    }
  }

  /**
   * Track cluster performance
   */
  trackClusterPerformance(
    cluster: ErrorCluster,
    solutions: CompetitiveSolution[],
    validatedSolutions: CompetitiveSolution[]
  ): void {
    const clusterId = cluster.clusterId;
    const validationScores = validatedSolutions.map(s => s.dveValidationScore || 0);
    const solutionTimes = solutions.map(s => s.submittedAt ? Date.now() - s.submittedAt.getTime() : 0);
    const totalBounty = validatedSolutions.reduce((sum, s) => sum + (s.bountyAwarded || 0), 0);
    const uniqueAgents = new Set(solutions.map(s => s.agentId)).size;

    const metrics: ClusterMetrics = {
      clusterId,
      clusterName: cluster.clusterName,
      totalErrors: cluster.errorNodes.length,
      totalSolutions: solutions.length,
      validatedSolutions: validatedSolutions.length,
      successRate: solutions.length > 0 ? validatedSolutions.length / solutions.length : 0,
      averageValidationScore: validationScores.length > 0 ? 
        validationScores.reduce((sum, score) => sum + score, 0) / validationScores.length : 0,
      totalBountyAwarded: totalBounty,
      averageSolutionTime: solutionTimes.length > 0 ?
        solutionTimes.reduce((sum, time) => sum + time, 0) / solutionTimes.length : 0,
      participatingAgents: uniqueAgents,
      competitionIntensity: this.calculateCompetitionIntensity(solutions),
      skillsGenerated: 0, // Will be updated when skills are created
      lastUpdated: new Date()
    };

    this.clusterMetrics.set(clusterId, metrics);

    logger.debug({
      clusterId,
      successRate: metrics.successRate,
      participatingAgents: metrics.participatingAgents
    }, 'Cluster performance tracked');

    this.emit('clusterMetricsUpdated', metrics);
  }

  /**
   * Track agent performance
   */
  trackAgentPerformance(agentId: string, solutions: CompetitiveSolution[]): void {
    const agentSolutions = solutions.filter(s => s.agentId === agentId);
    const validatedSolutions = agentSolutions.filter(s =>
      s.validationStatus === 'validated' &&
      s.dveValidationScore &&
      s.dveValidationScore > 0.7
    );
    const validationScores = validatedSolutions.map(s => s.dveValidationScore || 0);
    const solutionTimes = agentSolutions.map(s => s.submittedAt ? Date.now() - s.submittedAt.getTime() : 0);
    const totalBounty = validatedSolutions.reduce((sum, s) => sum + (s.bountyAwarded || 0), 0);
    const clustersParticipated = new Set(agentSolutions.map(s => s.errorNodeId)).size;

    const existingMetrics = this.agentMetrics.get(agentId);
    
    const metrics: AgentMetrics = {
      agentId,
      totalSolutionsSubmitted: agentSolutions.length,
      validatedSolutions: validatedSolutions.length,
      successRate: agentSolutions.length > 0 ? validatedSolutions.length / agentSolutions.length : 0,
      averageValidationScore: validationScores.length > 0 ?
        validationScores.reduce((sum, score) => sum + score, 0) / validationScores.length : 0,
      totalBountyEarned: totalBounty,
      clustersParticipated,
      clustersOwned: existingMetrics?.clustersOwned || 0,
      averageSolutionTime: solutionTimes.length > 0 ?
        solutionTimes.reduce((sum, time) => sum + time, 0) / solutionTimes.length : 0,
      reputation: this.calculateAgentReputation(agentSolutions, validatedSolutions),
      skillsContributed: existingMetrics?.skillsContributed || 0,
      lastActive: new Date()
    };

    this.agentMetrics.set(agentId, metrics);

    logger.debug({
      agentId,
      successRate: metrics.successRate,
      reputation: metrics.reputation
    }, 'Agent performance tracked');

    this.emit('agentMetricsUpdated', metrics);
  }

  /**
   * Track skill performance
   */
  trackSkillPerformance(
    skill: LoRAAdapterSkill,
    discoveryResult: SkillDiscoveryResult,
    trainingAccuracy: number
  ): void {
    const metrics: SkillMetrics = {
      skillId: skill.skillId,
      skillName: skill.skillName,
      category: discoveryResult.category,
      trainingAccuracy,
      validationScore: discoveryResult.confidence,
      usageCount: 0,
      successfulInvocations: 0,
      averageInferenceTime: 0,
      memoryUsage: (skill.weightsA.length + skill.weightsB.length) * 4, // bytes
      userRating: 0,
      createdAt: new Date(),
      lastUsed: new Date()
    };

    this.skillMetrics.set(skill.skillId, metrics);

    logger.debug({
      skillId: skill.skillId,
      category: discoveryResult.category,
      confidence: discoveryResult.confidence
    }, 'Skill performance tracked');

    this.emit('skillMetricsUpdated', metrics);
  }

  /**
   * Track competition metrics
   */
  trackCompetitionMetrics(
    clusterId: string,
    solutions: CompetitiveSolution[],
    competitionRound: number = 1
  ): void {
    if (!this.config.enableCompetitionMetrics) return;

    const participatingAgents = [...new Set(solutions.map(s => s.agentId))];
    const leaderboard = this.generateLeaderboard(solutions);
    const prizePool = solutions.reduce((sum, s) => sum + (s.bountyAwarded || 0), 0);

    const competition: CompetitionMetrics = {
      clusterId,
      competitionRound,
      startTime: new Date(Math.min(...solutions.map(s => s.submittedAt?.getTime() || Date.now()))),
      participatingAgents,
      solutionsSubmitted: solutions.length,
      leaderboard,
      prizePool,
      winnerAgent: leaderboard[0]?.agentId,
      winningScore: leaderboard[0]?.score || 0,
      competitionIntensity: this.calculateCompetitionIntensity(solutions)
    };

    const existingCompetitions = this.competitionMetrics.get(clusterId) || [];
    existingCompetitions.push(competition);
    this.competitionMetrics.set(clusterId, existingCompetitions);

    logger.info({
      clusterId,
      competitionRound,
      participatingAgents: participatingAgents.length,
      winnerAgent: competition.winnerAgent
    }, 'Competition metrics tracked');

    this.emit('competitionMetricsUpdated', competition);
  }

  /**
   * Calculate competition intensity
   */
  private calculateCompetitionIntensity(solutions: CompetitiveSolution[]): number {
    if (solutions.length === 0) return 0;

    const uniqueAgents = new Set(solutions.map(s => s.agentId)).size;
    const solutionsPerAgent = solutions.length / uniqueAgents;
    const timeSpread = this.calculateTimeSpread(solutions);
    
    // Intensity based on agent participation and solution frequency
    const participationScore = Math.min(1, uniqueAgents / 10); // Normalize to max 10 agents
    const frequencyScore = Math.min(1, solutionsPerAgent / 5); // Normalize to max 5 solutions per agent
    const timeScore = Math.max(0, 1 - timeSpread / (24 * 60 * 60 * 1000)); // Normalize to 24 hours
    
    return (participationScore + frequencyScore + timeScore) / 3;
  }

  /**
   * Calculate time spread of solutions
   */
  private calculateTimeSpread(solutions: CompetitiveSolution[]): number {
    if (solutions.length < 2) return 0;

    const times = solutions
      .map(s => s.submittedAt?.getTime() || 0)
      .filter(t => t > 0)
      .sort((a, b) => a - b);

    if (times.length < 2) return 0;

    return times[times.length - 1] - times[0];
  }

  /**
   * Generate leaderboard
   */
  private generateLeaderboard(solutions: CompetitiveSolution[]): LeaderboardEntry[] {
    const agentStats = new Map<string, {
      solutionsSubmitted: number;
      validatedSolutions: number;
      totalScore: number;
      bountyEarned: number;
      totalTime: number;
    }>();

    // Aggregate agent statistics
    for (const solution of solutions) {
      const agentId = solution.agentId;
      const stats = agentStats.get(agentId) || {
        solutionsSubmitted: 0,
        validatedSolutions: 0,
        totalScore: 0,
        bountyEarned: 0,
        totalTime: 0
      };

      stats.solutionsSubmitted++;
      if (solution.dveValidationScore && solution.dveValidationScore > 0.7) {
        stats.validatedSolutions++;
        stats.totalScore += solution.dveValidationScore;
      }
      stats.bountyEarned += solution.bountyAwarded || 0;
      stats.totalTime += solution.submittedAt ? Date.now() - solution.submittedAt.getTime() : 0;

      agentStats.set(agentId, stats);
    }

    // Create leaderboard entries
    const entries: LeaderboardEntry[] = [];
    for (const [agentId, stats] of agentStats.entries()) {
      const score = stats.validatedSolutions > 0 ? stats.totalScore / stats.validatedSolutions : 0;
      const averageTime = stats.solutionsSubmitted > 0 ? stats.totalTime / stats.solutionsSubmitted : 0;

      entries.push({
        rank: 0, // Will be set after sorting
        agentId,
        score,
        solutionsSubmitted: stats.solutionsSubmitted,
        validatedSolutions: stats.validatedSolutions,
        bountyEarned: stats.bountyEarned,
        averageTime
      });
    }

    // Sort by score (descending) and assign ranks
    entries.sort((a, b) => b.score - a.score);
    entries.forEach((entry, _index) => {
      entry.rank = _index + 1;
    });

    return entries.slice(0, this.config.maxLeaderboardSize);
  }

  /**
   * Calculate agent reputation
   */
  private calculateAgentReputation(
    allSolutions: CompetitiveSolution[],
    validatedSolutions: CompetitiveSolution[]
  ): number {
    if (allSolutions.length === 0) return 0;

    const successRate = validatedSolutions.length / allSolutions.length;
    const averageScore = validatedSolutions.length > 0 ?
      validatedSolutions.reduce((sum, s) => sum + (s.dveValidationScore || 0), 0) / validatedSolutions.length : 0;
    
    // Reputation based on success rate and quality
    return (successRate * 0.6) + (averageScore * 0.4);
  }

  /**
   * Update system metrics
   */
  private updateSystemMetrics(): void {
    const totalClusters = this.clusterMetrics.size;
    const totalAgents = this.agentMetrics.size;
    const totalSkills = this.skillMetrics.size;
    
    const allClusterMetrics = Array.from(this.clusterMetrics.values());
    const totalSolutions = allClusterMetrics.reduce((sum, m) => sum + m.totalSolutions, 0);
    const totalValidated = allClusterMetrics.reduce((sum, m) => sum + m.validatedSolutions, 0);
    const overallSuccessRate = totalSolutions > 0 ? totalValidated / totalSolutions : 0;
    
    const averageClusterCompetition = allClusterMetrics.length > 0 ?
      allClusterMetrics.reduce((sum, m) => sum + m.competitionIntensity, 0) / allClusterMetrics.length : 0;
    
    const totalBountyDistributed = allClusterMetrics.reduce((sum, m) => sum + m.totalBountyAwarded, 0);

    this.systemMetrics = {
      totalClusters,
      totalAgents,
      totalSkills,
      totalSolutions,
      overallSuccessRate,
      averageClusterCompetition,
      totalBountyDistributed,
      systemThroughput: this.calculateSystemThroughput(),
      averageSkillCreationTime: this.calculateAverageSkillCreationTime(),
      networkHealth: this.calculateNetworkHealth(),
      lastUpdated: new Date()
    };

    this.emit('systemMetricsUpdated', this.systemMetrics);
  }

  /**
   * Calculate system throughput
   */
  private calculateSystemThroughput(): number {
    // Solutions per hour
    const hourAgo = new Date(Date.now() - 60 * 60 * 1000);
    const recentSolutions = Array.from(this.clusterMetrics.values())
      .filter(m => m.lastUpdated > hourAgo)
      .reduce((sum, m) => sum + m.totalSolutions, 0);
    
    return recentSolutions;
  }

  /**
   * Calculate average skill creation time
   */
  private calculateAverageSkillCreationTime(): number {
    const skillCreationTimes = Array.from(this.skillMetrics.values())
      .map(s => s.createdAt.getTime())
      .sort((a, b) => b - a)
      .slice(0, 10); // Last 10 skills

    if (skillCreationTimes.length < 2) return 0;

    const intervals = [];
    for (let i = 1; i < skillCreationTimes.length; i++) {
      intervals.push(skillCreationTimes[i - 1] - skillCreationTimes[i]);
    }

    return intervals.reduce((sum, interval) => sum + interval, 0) / intervals.length;
  }

  /**
   * Calculate network health
   */
  private calculateNetworkHealth(): number {
    const activeAgents = Array.from(this.agentMetrics.values())
      .filter(a => Date.now() - a.lastActive.getTime() < 60 * 60 * 1000).length; // Active in last hour
    
    const totalAgents = this.agentMetrics.size;
    const activityRate = totalAgents > 0 ? activeAgents / totalAgents : 0;
    
    const averageSuccessRate = Array.from(this.clusterMetrics.values())
      .reduce((sum, m) => sum + m.successRate, 0) / Math.max(1, this.clusterMetrics.size);
    
    return (activityRate * 0.4) + (averageSuccessRate * 0.6);
  }

  /**
   * Start metrics update loop
   */
  private startMetricsUpdateLoop(): void {
    this.updateInterval = setInterval(() => {
      this.updateSystemMetrics();
      this.cleanupOldMetrics();
    }, this.config.updateInterval) as unknown as NodeJS.Timeout;
  }

  /**
   * Cleanup old metrics
   */
  private cleanupOldMetrics(): void {
    const cutoffTime = new Date(Date.now() - this.config.retentionPeriod);
    
    // Clean up old competition metrics
    for (const [clusterId, competitions] of this.competitionMetrics.entries()) {
      const recentCompetitions = competitions.filter(c => c.startTime > cutoffTime);
      if (recentCompetitions.length !== competitions.length) {
        this.competitionMetrics.set(clusterId, recentCompetitions);
      }
    }
  }

  /**
   * Get cluster metrics
   */
  getClusterMetrics(clusterId?: string): ClusterMetrics | ClusterMetrics[] | undefined {
    if (clusterId) {
      return this.clusterMetrics.get(clusterId);
    }
    return Array.from(this.clusterMetrics.values());
  }

  /**
   * Get agent metrics
   */
  getAgentMetrics(agentId?: string): AgentMetrics | AgentMetrics[] | undefined {
    if (agentId) {
      return this.agentMetrics.get(agentId);
    }
    return Array.from(this.agentMetrics.values());
  }

  /**
   * Get skill metrics
   */
  getSkillMetrics(skillId?: string): SkillMetrics | SkillMetrics[] | undefined {
    if (skillId) {
      return this.skillMetrics.get(skillId);
    }
    return Array.from(this.skillMetrics.values());
  }

  /**
   * Get system metrics
   */
  getSystemMetrics(): SystemMetrics {
    this.updateSystemMetrics();
    return { ...this.systemMetrics };
  }

  /**
   * Get competition metrics
   */
  getCompetitionMetrics(clusterId?: string): CompetitionMetrics[] {
    if (clusterId) {
      return this.competitionMetrics.get(clusterId) || [];
    }
    return Array.from(this.competitionMetrics.values()).flat();
  }

  /**
   * Stop metrics tracking
   */
  stop(): void {
    if (this.updateInterval) {
      clearInterval(this.updateInterval);
      this.updateInterval = undefined;
    }
    logger.info('Performance Metrics tracking stopped');
  }

  /**
   * Reset all metrics data (useful for testing)
   */
  reset(): void {
    this.clusterMetrics.clear();
    this.agentMetrics.clear();
    this.skillMetrics.clear();
    this.competitionMetrics.clear();
    this.systemMetrics = {
      totalClusters: 0,
      totalAgents: 0,
      totalSkills: 0,
      totalSolutions: 0,
      overallSuccessRate: 0,
      averageClusterCompetition: 0,
      totalBountyDistributed: 0,
      systemThroughput: 0,
      averageSkillCreationTime: 0,
      networkHealth: 0,
      lastUpdated: new Date()
    };
    logger.info('Performance Metrics data reset');
  }

  /**
   * Check if ready
   */
  isReady(): boolean {
    return this.isInitialized;
  }
}
