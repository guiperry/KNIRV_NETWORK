/**
 * Consensus Mechanism for KNIRVGRAPH
 * 
 * Implements simultaneous consensus with all agent-cores during skill confirmation
 * Manages distributed validation and agreement across the network
 */

import pino from 'pino';
import { EventEmitter } from 'events';
import { LoRAAdapterSkill } from '../lora/LoRAAdapterEngine';
import { SkillValidationResult } from './SkillMintingProcess';

const logger = pino({ name: 'consensus-mechanism' });

export interface AgentCoreNode {
  nodeId: string;
  address: string;
  publicKey: string;
  reputation: number;
  lastSeen: Date;
  capabilities: string[];
  status: NodeStatus;
  votingPower: number;
}

export enum NodeStatus {
  ACTIVE = 'active',
  INACTIVE = 'inactive',
  SUSPENDED = 'suspended',
  OFFLINE = 'offline'
}

export interface ConsensusProposal {
  proposalId: string;
  skillId: string;
  loraAdapter: LoRAAdapterSkill;
  validationResult: SkillValidationResult;
  proposedBy: string;
  proposedAt: Date;
  submittedBy: string;
  submittedAt: Date;
  votingDeadline: Date;
  requiredVotes: number;
  status: ProposalStatus;
}

export enum ProposalStatus {
  PENDING = 'pending',
  VOTING = 'voting',
  APPROVED = 'approved',
  REJECTED = 'rejected',
  EXPIRED = 'expired'
}

export interface ConsensusVote {
  voteId: string;
  proposalId: string;
  nodeId: string;
  vote: VoteType;
  confidence: number;
  reasoning: string;
  signature: string;
  votedAt: Date;
}

export enum VoteType {
  APPROVE = 'approve',
  REJECT = 'reject',
  ABSTAIN = 'abstain'
}

export interface ConsensusResult {
  proposalId: string;
  skillId: string;
  submittedBy: string;
  submittedAt: Date;
  approved: boolean;
  outcome: 'approved' | 'rejected' | 'timeout';
  totalVotes: number;
  approveVotes: number;
  rejectVotes: number;
  abstainVotes: number;
  consensusReached: boolean;
  finalizedAt: Date;
  participatingNodes: string[];
}

export interface ConsensusConfig {
  votingTimeoutMs: number;
  minParticipationRate: number;
  approvalThreshold: number;
  maxConcurrentProposals: number;
  nodeDiscoveryInterval: number;
  heartbeatInterval: number;
  reputationThreshold: number;
  votingMonitoringInterval: number;
}

export class ConsensusMechanism extends EventEmitter {
  private agentNodes: Map<string, AgentCoreNode> = new Map();
  private activeProposals: Map<string, ConsensusProposal> = new Map();
  private votes: Map<string, ConsensusVote[]> = new Map();
  private consensusResults: Map<string, ConsensusResult> = new Map();
  
  private config: ConsensusConfig;
  private isInitialized: boolean = false;
  private discoveryInterval?: NodeJS.Timeout;
  private heartbeatInterval?: NodeJS.Timeout;
  private votingInterval?: NodeJS.Timeout;
  private logger = pino({ name: 'consensus-mechanism' });

  constructor(config: Partial<ConsensusConfig> = {}) {
    super();
    
    this.config = {
      votingTimeoutMs: 300000, // 5 minutes
      minParticipationRate: 0.67, // 67% of nodes must participate
      approvalThreshold: 0.75, // 75% approval required
      maxConcurrentProposals: 10,
      nodeDiscoveryInterval: 30000, // 30 seconds
      heartbeatInterval: 15000, // 15 seconds
      reputationThreshold: 0.5, // Minimum reputation to vote
      votingMonitoringInterval: 1000, // 1 second
      ...config
    };
  }

  /**
   * Initialize the consensus mechanism
   */
  async initialize(): Promise<void> {
    logger.info('Initializing Consensus Mechanism...');

    try {
      // Start node discovery
      this.startNodeDiscovery();

      // Start heartbeat monitoring
      this.startHeartbeatMonitoring();

      // Start voting monitoring
      this.startVotingMonitoring();

      this.isInitialized = true;
      logger.info('Consensus Mechanism initialized successfully');

    } catch (error) {
      logger.error({ error }, 'Failed to initialize Consensus Mechanism');
      throw error;
    }
  }

  /**
   * Submit proposal for consensus
   */
  async submitProposal(
    skillId: string,
    loraAdapter: LoRAAdapterSkill,
    validationResult: SkillValidationResult,
    proposedBy: string
  ): Promise<string> {
    if (!this.isInitialized) {
      throw new Error('Consensus Mechanism not initialized');
    }

    if (this.activeProposals.size >= this.config.maxConcurrentProposals) {
      throw new Error('Maximum concurrent proposals reached');
    }

    const proposalId = `proposal_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
    const votingDeadline = new Date(Date.now() + this.config.votingTimeoutMs);
    const activeNodes = this.getActiveNodes();
    const requiredVotes = Math.floor(activeNodes.length * this.config.minParticipationRate);

    const proposal: ConsensusProposal = {
      proposalId,
      skillId,
      loraAdapter,
      validationResult,
      proposedBy,
      proposedAt: new Date(),
      submittedBy: proposedBy,
      submittedAt: new Date(),
      votingDeadline,
      requiredVotes,
      status: ProposalStatus.PENDING
    };

    this.activeProposals.set(proposalId, proposal);
    this.votes.set(proposalId, []);

    logger.info({
      proposalId,
      skillId,
      requiredVotes,
      activeNodes: activeNodes.length,
      votingDeadline
    }, 'Consensus proposal submitted');

    // Broadcast proposal to all active nodes
    await this.broadcastProposal(proposal);

    // Start voting phase
    proposal.status = ProposalStatus.VOTING;
    this.emit('proposalSubmitted', proposal);

    return proposalId;
  }

  /**
   * Submit vote for proposal
   */
  async submitVote(
    proposalId: string,
    nodeId: string,
    vote: VoteType,
    confidence: number,
    reasoning: string = ''
  ): Promise<string> {
    const proposal = this.activeProposals.get(proposalId);
    if (!proposal) {
      throw new Error('Proposal not found');
    }

    if (proposal.status !== ProposalStatus.VOTING) {
      throw new Error('Proposal is not in voting phase');
    }

    if (new Date() > proposal.votingDeadline) {
      throw new Error('Voting deadline has passed');
    }

    const node = this.agentNodes.get(nodeId);
    if (!node) {
      throw new Error('Node not registered');
    }

    if (node.status !== NodeStatus.ACTIVE) {
      throw new Error('Node is not active');
    }

    if (node.reputation < this.config.reputationThreshold) {
      throw new Error('Node reputation below threshold');
    }

    // Check if node already voted
    const existingVotes = this.votes.get(proposalId) || [];
    if (existingVotes.some(v => v.nodeId === nodeId)) {
      throw new Error('Node has already voted on this proposal');
    }

    const voteId = `vote_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
    const signature = this.createVoteSignature(proposalId, nodeId, vote);

    const consensusVote: ConsensusVote = {
      voteId,
      proposalId,
      nodeId,
      vote,
      confidence,
      reasoning,
      signature,
      votedAt: new Date()
    };

    existingVotes.push(consensusVote);
    this.votes.set(proposalId, existingVotes);

    logger.info({
      voteId,
      proposalId,
      nodeId,
      vote,
      confidence
    }, 'Vote submitted');

    this.emit('voteSubmitted', consensusVote);

    // Check if consensus is reached
    await this.checkConsensus(proposalId);

    return voteId;
  }

  /**
   * Register agent-core node
   */
  async registerNode(
    nodeId: string,
    address: string,
    publicKey: string,
    capabilities: string[] = []
  ): Promise<void> {
    const node: AgentCoreNode = {
      nodeId,
      address,
      publicKey,
      reputation: 1.0, // Start with full reputation
      lastSeen: new Date(),
      capabilities,
      status: NodeStatus.ACTIVE,
      votingPower: 1.0
    };

    this.agentNodes.set(nodeId, node);

    logger.info({
      nodeId,
      address,
      capabilities: capabilities.length
    }, 'Agent-core node registered');

    this.emit('nodeRegistered', node);
  }

  /**
   * Update node heartbeat
   */
  updateNodeHeartbeat(nodeId: string): void {
    const node = this.agentNodes.get(nodeId);
    if (node) {
      node.lastSeen = new Date();
      node.status = NodeStatus.ACTIVE;
    }
  }

  /**
   * Get active nodes
   */
  private getActiveNodes(): AgentCoreNode[] {
    return Array.from(this.agentNodes.values())
      .filter(node => 
        node.status === NodeStatus.ACTIVE && 
        node.reputation >= this.config.reputationThreshold
      );
  }

  /**
   * Broadcast proposal to all active nodes
   */
  private async broadcastProposal(proposal: ConsensusProposal): Promise<void> {
    const activeNodes = this.getActiveNodes();
    
    logger.info({
      proposalId: proposal.proposalId,
      targetNodes: activeNodes.length
    }, 'Broadcasting proposal to active nodes');

    // In a real implementation, this would send HTTP requests to each node
    // For now, we simulate the broadcast
    for (const node of activeNodes) {
      try {
        await this.sendProposalToNode(node, proposal);
      } catch (error) {
        logger.warn({
          nodeId: node.nodeId,
          error: error instanceof Error ? error.message : String(error)
        }, 'Failed to send proposal to node');
      }
    }
  }

  /**
   * Send proposal to specific node
   */
  private async sendProposalToNode(node: AgentCoreNode, proposal: ConsensusProposal): Promise<void> {
    // Simulate network call
    await new Promise(resolve => setTimeout(resolve, 100));
    
    logger.debug({
      nodeId: node.nodeId,
      proposalId: proposal.proposalId
    }, 'Proposal sent to node');
  }

  /**
   * Check if consensus is reached
   */
  private async checkConsensus(proposalId: string): Promise<void> {
    const proposal = this.activeProposals.get(proposalId);
    if (!proposal) return;

    const votes = this.votes.get(proposalId) || [];
    const activeNodes = this.getActiveNodes();
    
    // Check if voting deadline passed
    if (new Date() > proposal.votingDeadline) {
      await this.finalizeProposal(proposalId, 'timeout');
      return;
    }

    // Check if minimum required votes reached
    if (votes.length < proposal.requiredVotes) {
      return; // Wait for more votes
    }

    // Calculate vote results
    const approveVotes = votes.filter(v => v.vote === VoteType.APPROVE).length;
    const rejectVotes = votes.filter(v => v.vote === VoteType.REJECT).length;
    const abstainVotes = votes.filter(v => v.vote === VoteType.ABSTAIN).length;

    const totalVotes = approveVotes + rejectVotes + abstainVotes;
    const approvalRate = approveVotes / (approveVotes + rejectVotes); // Exclude abstains

    // Check if consensus reached
    if (approvalRate >= this.config.approvalThreshold) {
      await this.finalizeProposal(proposalId, 'approved');
    } else {
      // Check if we have votes from all active nodes or if it's impossible to reach threshold
      const votingNodes = new Set(votes.map(v => v.nodeId));
      const allNodesVoted = activeNodes.every(node => votingNodes.has(node.nodeId));

      if (allNodesVoted) {
        // All nodes have voted and threshold not met, reject
        await this.finalizeProposal(proposalId, 'rejected');
      } else {
        // Check if it's mathematically impossible to reach threshold
        const remainingNodes = activeNodes.filter(node => !votingNodes.has(node.nodeId));
        const maxPossibleApproves = approveVotes + remainingNodes.length;
        const maxPossibleTotal = totalVotes + remainingNodes.length;
        const maxPossibleRate = maxPossibleApproves / maxPossibleTotal;

        if (maxPossibleRate < this.config.approvalThreshold) {
          await this.finalizeProposal(proposalId, 'rejected');
        }
      }
    }
  }

  /**
   * Finalize proposal
   */
  private async finalizeProposal(proposalId: string, outcome: 'approved' | 'rejected' | 'timeout'): Promise<void> {
    const proposal = this.activeProposals.get(proposalId);
    if (!proposal) return;

    const votes = this.votes.get(proposalId) || [];
    const approveVotes = votes.filter(v => v.vote === VoteType.APPROVE).length;
    const rejectVotes = votes.filter(v => v.vote === VoteType.REJECT).length;
    const abstainVotes = votes.filter(v => v.vote === VoteType.ABSTAIN).length;

    let approved = false;
    let consensusReached = true;

    switch (outcome) {
      case 'approved':
        proposal.status = ProposalStatus.APPROVED;
        approved = true;
        break;
      case 'rejected':
        proposal.status = ProposalStatus.REJECTED;
        approved = false;
        break;
      case 'timeout':
        proposal.status = ProposalStatus.EXPIRED;
        approved = false;
        consensusReached = false;
        break;
    }

    const result: ConsensusResult = {
      proposalId,
      skillId: proposal.skillId,
      submittedBy: proposal.submittedBy,
      submittedAt: proposal.submittedAt,
      approved,
      outcome,
      totalVotes: votes.length,
      approveVotes,
      rejectVotes,
      abstainVotes,
      consensusReached,
      finalizedAt: new Date(),
      participatingNodes: votes.map(v => v.nodeId)
    };

    this.consensusResults.set(proposalId, result);
    this.activeProposals.delete(proposalId);

    logger.info({
      proposalId,
      outcome,
      approved,
      totalVotes: votes.length,
      approveVotes,
      rejectVotes
    }, 'Consensus proposal finalized');

    this.emit('consensusReached', { proposal, result });

    // Update node reputations based on voting behavior
    await this.updateNodeReputations(proposalId, votes, approved);
  }

  /**
   * Update node reputations based on voting behavior
   */
  private async updateNodeReputations(
    proposalId: string,
    votes: ConsensusVote[],
    finalOutcome: boolean
  ): Promise<void> {
    this.logger?.info({ proposalId, finalOutcome, totalVotes: votes.length }, 'Updating node reputations');

    for (const vote of votes) {
      const node = this.agentNodes.get(vote.nodeId);
      if (!node) continue;

      // Reward nodes that voted with the majority
      const votedCorrectly = (vote.vote === VoteType.APPROVE && finalOutcome) ||
                            (vote.vote === VoteType.REJECT && !finalOutcome);

      const oldReputation = node.reputation;
      if (votedCorrectly) {
        node.reputation = Math.min(2.0, node.reputation + 0.1);
      } else if (vote.vote !== VoteType.ABSTAIN) {
        node.reputation = Math.max(0.0, node.reputation - 0.1);
      }

      this.logger?.info({
        nodeId: vote.nodeId,
        vote: vote.vote,
        votedCorrectly,
        oldReputation,
        newReputation: node.reputation
      }, 'Node reputation updated');
    }
  }

  /**
   * Create vote signature
   */
  private createVoteSignature(proposalId: string, nodeId: string, vote: VoteType): string {
    // Simple signature simulation (in production, use proper cryptographic signing)
    const data = `${proposalId}:${nodeId}:${vote}:${Date.now()}`;
    let hash = 0;
    for (let i = 0; i < data.length; i++) {
      const char = data.charCodeAt(i);
      hash = ((hash << 5) - hash) + char;
      hash = hash & hash;
    }
    return Math.abs(hash).toString(16);
  }

  /**
   * Start node discovery
   */
  private startNodeDiscovery(): void {
    this.discoveryInterval = setInterval(async () => {
      await this.discoverNodes();
    }, this.config.nodeDiscoveryInterval) as unknown as NodeJS.Timeout;
  }

  /**
   * Discover new nodes
   */
  private async discoverNodes(): Promise<void> {
    // In a real implementation, this would query the network for new nodes
    // For now, we simulate node discovery
    logger.debug('Discovering agent-core nodes...');
  }

  /**
   * Start heartbeat monitoring
   */
  private startHeartbeatMonitoring(): void {
    this.heartbeatInterval = setInterval(() => {
      this.checkNodeHeartbeats();
    }, this.config.heartbeatInterval) as unknown as NodeJS.Timeout;
  }

  /**
   * Check node heartbeats
   */
  private checkNodeHeartbeats(): void {
    const now = new Date();
    const timeoutMs = this.config.heartbeatInterval * 3; // 3 missed heartbeats = offline

    for (const [nodeId, node] of this.agentNodes.entries()) {
      const timeSinceLastSeen = now.getTime() - node.lastSeen.getTime();
      
      if (timeSinceLastSeen > timeoutMs && node.status === NodeStatus.ACTIVE) {
        node.status = NodeStatus.OFFLINE;
        logger.warn({ nodeId }, 'Node marked as offline due to missed heartbeats');
        this.emit('nodeOffline', node);
      }
    }
  }

  /**
   * Start voting monitoring
   */
  private startVotingMonitoring(): void {
    this.votingInterval = setInterval(async () => {
      await this.checkVotingDeadlines();
    }, this.config.votingMonitoringInterval) as unknown as NodeJS.Timeout;
  }

  /**
   * Check voting deadlines
   */
  private async checkVotingDeadlines(): Promise<void> {
    const now = new Date();

    for (const [proposalId, proposal] of this.activeProposals.entries()) {
      if (proposal.status === ProposalStatus.VOTING && now > proposal.votingDeadline) {
        // Mark as expired first
        proposal.status = ProposalStatus.EXPIRED;
        await this.finalizeProposal(proposalId, 'timeout');
      }
    }
  }

  /**
   * Get proposal status
   */
  getProposal(proposalId: string): ConsensusProposal | undefined {
    // First check active proposals
    const activeProposal = this.activeProposals.get(proposalId);
    if (activeProposal) {
      return activeProposal;
    }

    // If not found in active, check if it was finalized
    const result = this.consensusResults.get(proposalId);
    if (result) {
      // Reconstruct proposal from result for backward compatibility
      let status: ProposalStatus;
      if (result.outcome === 'timeout') {
        status = ProposalStatus.EXPIRED;
      } else if (result.approved) {
        status = ProposalStatus.APPROVED;
      } else {
        status = ProposalStatus.REJECTED;
      }

      return {
        proposalId: proposalId,
        skillId: result.skillId,
        loraAdapter: {} as LoRAAdapterSkill, // Not stored in result
        validationResult: {} as SkillValidationResult, // Not stored in result
        proposedBy: result.submittedBy,
        proposedAt: result.submittedAt,
        submittedBy: result.submittedBy,
        submittedAt: result.submittedAt,
        votingDeadline: result.finalizedAt, // Use finalized time as deadline
        requiredVotes: 0, // Not stored in result
        status: status
      };
    }

    return undefined;
  }

  /**
   * Get consensus result
   */
  getConsensusResult(proposalId: string): ConsensusResult | undefined {
    return this.consensusResults.get(proposalId);
  }

  /**
   * Get active proposals
   */
  getActiveProposals(): ConsensusProposal[] {
    return Array.from(this.activeProposals.values());
  }

  /**
   * Get registered nodes
   */
  getRegisteredNodes(): AgentCoreNode[] {
    return Array.from(this.agentNodes.values());
  }

  /**
   * Stop consensus mechanism
   */
  stop(): void {
    if (this.discoveryInterval) {
      clearInterval(this.discoveryInterval);
      this.discoveryInterval = undefined;
    }
    
    if (this.heartbeatInterval) {
      clearInterval(this.heartbeatInterval);
      this.heartbeatInterval = undefined;
    }
    
    if (this.votingInterval) {
      clearInterval(this.votingInterval);
      this.votingInterval = undefined;
    }
    
    logger.info('Consensus Mechanism stopped');
  }

  /**
   * Check if ready
   */
  isReady(): boolean {
    return this.isInitialized;
  }
}
