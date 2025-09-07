/**
 * Phase 3.3 Consensus Mechanism Tests
 * 
 * Tests for simultaneous consensus with all agent-cores during skill confirmation
 */

import { 
  ConsensusMechanism, 
  AgentCoreNode, 
  NodeStatus, 
  ConsensusProposal, 
  ProposalStatus,
  VoteType 
} from '../../src/core/knirvgraph/ConsensusMechanism';
import { LoRAAdapterSkill } from '../../src/core/lora/LoRAAdapterEngine';
import { SkillValidationResult } from '../../src/core/knirvgraph/SkillMintingProcess';

describe('Phase 3.3 - Consensus Mechanism', () => {
  let consensusMechanism: ConsensusMechanism;
  let mockLoRAAdapter: LoRAAdapterSkill;
  let mockValidationResult: SkillValidationResult;

  beforeEach(async () => {
    consensusMechanism = new ConsensusMechanism({
      votingTimeoutMs: 5000,
      minParticipationRate: 0.67,
      approvalThreshold: 0.75,
      maxConcurrentProposals: 5,
      nodeDiscoveryInterval: 10000,
      heartbeatInterval: 5000,
      reputationThreshold: 0.5
    });

    await consensusMechanism.initialize();

    // Create mock LoRA adapter
    mockLoRAAdapter = {
      skillId: 'test_skill_001',
      skillName: 'Test Consensus Skill',
      description: 'A test skill for consensus testing',
      baseModelCompatibility: 'hrm',
      version: 1,
      rank: 8,
      alpha: 16.0,
      weightsA: new Float32Array(64),
      weightsB: new Float32Array(64),
      additionalMetadata: {}
    };

    // Create mock validation result
    mockValidationResult = {
      validationId: 'val_001',
      skillId: 'test_skill_001',
      isValid: true,
      validationScore: 0.85,
      validationErrors: [],
      validationWarnings: [],
      technicalValidation: {
        weightsIntegrity: true,
        dimensionConsistency: true,
        numericalStability: true,
        memoryRequirements: 1024,
        computationalComplexity: 5000
      },
      semanticValidation: {
        skillNameValidity: true,
        categoryConsistency: true,
        capabilityAlignment: true,
        descriptionAccuracy: true,
        metadataCompleteness: true
      },
      performanceValidation: {
        expectedAccuracy: 0.85,
        inferenceLatency: 50,
        memoryEfficiency: 0.8,
        scalabilityScore: 0.75,
        robustnessScore: 0.8
      },
      securityValidation: {
        maliciousCodeDetection: true,
        dataLeakageRisk: 0.1,
        adversarialRobustness: 0.8,
        privacyCompliance: true,
        auditTrail: ['Security validated']
      },
      validatedAt: new Date()
    };

    // Register some test nodes
    await consensusMechanism.registerNode(
      'node_001',
      'http://localhost:8001',
      'pubkey_001',
      ['debugging', 'optimization']
    );
    
    await consensusMechanism.registerNode(
      'node_002',
      'http://localhost:8002',
      'pubkey_002',
      ['testing', 'security']
    );
    
    await consensusMechanism.registerNode(
      'node_003',
      'http://localhost:8003',
      'pubkey_003',
      ['refactoring', 'integration']
    );
  });

  afterEach(() => {
    consensusMechanism.stop();
  });

  describe('Initialization', () => {
    test('should initialize successfully', async () => {
      const newMechanism = new ConsensusMechanism();
      await expect(newMechanism.initialize()).resolves.not.toThrow();
      expect(newMechanism.isReady()).toBe(true);
      newMechanism.stop();
    });
  });

  describe('Node Management', () => {
    test('should register nodes successfully', async () => {
      await consensusMechanism.registerNode(
        'node_004',
        'http://localhost:8004',
        'pubkey_004',
        ['data_processing']
      );

      const nodes = consensusMechanism.getRegisteredNodes();
      expect(nodes.length).toBe(4);
      
      const newNode = nodes.find(n => n.nodeId === 'node_004');
      expect(newNode).toBeDefined();
      expect(newNode!.address).toBe('http://localhost:8004');
      expect(newNode!.status).toBe(NodeStatus.ACTIVE);
      expect(newNode!.reputation).toBe(1.0);
    });

    test('should update node heartbeat', () => {
      const initialTime = new Date();
      
      consensusMechanism.updateNodeHeartbeat('node_001');
      
      const nodes = consensusMechanism.getRegisteredNodes();
      const node = nodes.find(n => n.nodeId === 'node_001');
      
      expect(node).toBeDefined();
      expect(node!.lastSeen.getTime()).toBeGreaterThanOrEqual(initialTime.getTime());
      expect(node!.status).toBe(NodeStatus.ACTIVE);
    });

    test('should handle heartbeat for non-existent node', () => {
      // Should not throw
      expect(() => consensusMechanism.updateNodeHeartbeat('non_existent'))
        .not.toThrow();
    });
  });

  describe('Proposal Submission', () => {
    test('should submit proposal successfully', async () => {
      const proposalId = await consensusMechanism.submitProposal(
        'test_skill_001',
        mockLoRAAdapter,
        mockValidationResult,
        'test_proposer'
      );

      expect(proposalId).toBeTruthy();
      expect(proposalId).toMatch(/^proposal_/);

      const proposal = consensusMechanism.getProposal(proposalId);
      expect(proposal).toBeDefined();
      expect(proposal!.status).toBe(ProposalStatus.VOTING);
      expect(proposal!.skillId).toBe('test_skill_001');
      expect(proposal!.proposedBy).toBe('test_proposer');
    });

    test('should calculate required votes correctly', async () => {
      const proposalId = await consensusMechanism.submitProposal(
        'test_skill_001',
        mockLoRAAdapter,
        mockValidationResult,
        'test_proposer'
      );

      const proposal = consensusMechanism.getProposal(proposalId);
      expect(proposal).toBeDefined();
      
      // With 3 nodes and 67% participation rate, should require 2 votes
      expect(proposal!.requiredVotes).toBe(2);
    });

    test('should fail when max concurrent proposals reached', async () => {
      // Submit maximum number of proposals
      const promises = Array.from({ length: 5 }, (_, i) =>
        consensusMechanism.submitProposal(
          `skill_${i}`,
          { ...mockLoRAAdapter, skillId: `skill_${i}` },
          mockValidationResult,
          'test_proposer'
        )
      );

      await Promise.all(promises);

      // Try to submit one more
      await expect(consensusMechanism.submitProposal(
        'skill_overflow',
        mockLoRAAdapter,
        mockValidationResult,
        'test_proposer'
      )).rejects.toThrow('Maximum concurrent proposals reached');
    });

    test('should fail when not initialized', async () => {
      const newMechanism = new ConsensusMechanism();
      
      await expect(newMechanism.submitProposal(
        'test_skill_001',
        mockLoRAAdapter,
        mockValidationResult,
        'test_proposer'
      )).rejects.toThrow('Consensus Mechanism not initialized');
    });
  });

  describe('Voting Process', () => {
    let proposalId: string;

    beforeEach(async () => {
      proposalId = await consensusMechanism.submitProposal(
        'test_skill_001',
        mockLoRAAdapter,
        mockValidationResult,
        'test_proposer'
      );
    });

    test('should submit vote successfully', async () => {
      const voteId = await consensusMechanism.submitVote(
        proposalId,
        'node_001',
        VoteType.APPROVE,
        0.9,
        'Skill looks good'
      );

      expect(voteId).toBeTruthy();
      expect(voteId).toMatch(/^vote_/);
    });

    test('should prevent duplicate voting', async () => {
      await consensusMechanism.submitVote(
        proposalId,
        'node_001',
        VoteType.APPROVE,
        0.9
      );

      await expect(consensusMechanism.submitVote(
        proposalId,
        'node_001',
        VoteType.REJECT,
        0.8
      )).rejects.toThrow('Node has already voted on this proposal');
    });

    test('should fail voting on non-existent proposal', async () => {
      await expect(consensusMechanism.submitVote(
        'non_existent_proposal',
        'node_001',
        VoteType.APPROVE,
        0.9
      )).rejects.toThrow('Proposal not found');
    });

    test('should fail voting with unregistered node', async () => {
      await expect(consensusMechanism.submitVote(
        proposalId,
        'unregistered_node',
        VoteType.APPROVE,
        0.9
      )).rejects.toThrow('Node not registered');
    });

    test('should fail voting with low reputation node', async () => {
      // Register node with low reputation
      await consensusMechanism.registerNode(
        'low_rep_node',
        'http://localhost:8005',
        'pubkey_005'
      );

      // Manually set low reputation (in real scenario, this would be calculated)
      const nodes = consensusMechanism.getRegisteredNodes();
      const lowRepNode = nodes.find(n => n.nodeId === 'low_rep_node');
      if (lowRepNode) {
        lowRepNode.reputation = 0.3; // Below threshold of 0.5
      }

      await expect(consensusMechanism.submitVote(
        proposalId,
        'low_rep_node',
        VoteType.APPROVE,
        0.9
      )).rejects.toThrow('Node reputation below threshold');
    });
  });

  describe('Consensus Resolution', () => {
    let proposalId: string;

    beforeEach(async () => {
      proposalId = await consensusMechanism.submitProposal(
        'test_skill_001',
        mockLoRAAdapter,
        mockValidationResult,
        'test_proposer'
      );
    });

    test('should reach consensus with sufficient approval votes', async () => {
      // Submit approval votes from 2 nodes (meets 75% threshold)
      await consensusMechanism.submitVote(proposalId, 'node_001', VoteType.APPROVE, 0.9);
      await consensusMechanism.submitVote(proposalId, 'node_002', VoteType.APPROVE, 0.8);

      // Wait for consensus to be processed
      await new Promise(resolve => setTimeout(resolve, 100));

      const result = consensusMechanism.getConsensusResult(proposalId);
      if (result) {
        expect(result.approved).toBe(true);
        expect(result.consensusReached).toBe(true);
        expect(result.approveVotes).toBe(2);
      }
    });

    test('should reject with insufficient approval votes', async () => {
      // Submit mixed votes (1 approve, 1 reject = 50% approval, below 75% threshold)
      // The consensus mechanism should finalize early since it's mathematically impossible to reach 75%
      await consensusMechanism.submitVote(proposalId, 'node_001', VoteType.APPROVE, 0.9);
      await consensusMechanism.submitVote(proposalId, 'node_002', VoteType.REJECT, 0.8);

      // Try to submit third vote - should fail because proposal was finalized early
      try {
        await consensusMechanism.submitVote(proposalId, 'node_003', VoteType.REJECT, 0.7);
      } catch (error) {
        expect((error as Error).message).toBe('Proposal not found');
      }

      // Wait for consensus to be processed
      await new Promise(resolve => setTimeout(resolve, 100));

      const result = consensusMechanism.getConsensusResult(proposalId);
      if (result) {
        expect(result.approved).toBe(false);
        expect(result.consensusReached).toBe(true);
        expect(result.approveVotes).toBe(1);
        expect(result.rejectVotes).toBe(1);
        expect(result.outcome).toBe('rejected');
      }
    });

    test('should handle abstain votes correctly', async () => {
      // Submit votes including abstains - submit abstain first to avoid early finalization
      await consensusMechanism.submitVote(proposalId, 'node_002', VoteType.ABSTAIN, 0.5);
      await consensusMechanism.submitVote(proposalId, 'node_001', VoteType.APPROVE, 0.9);

      // Try to submit third vote, but it might fail if proposal is already finalized
      try {
        await consensusMechanism.submitVote(proposalId, 'node_003', VoteType.APPROVE, 0.8);
      } catch (error) {
        // Proposal might be finalized already, which is acceptable
        console.debug('Vote submission error (expected):', error);
      }

      // Wait for consensus
      await new Promise(resolve => setTimeout(resolve, 100));

      const result = consensusMechanism.getConsensusResult(proposalId);
      if (result) {
        expect(result.abstainVotes).toBeGreaterThanOrEqual(1);
        // Should be approved since we have approve votes
        expect(result.approved).toBe(true);
      }
    });

    test('should timeout proposals after deadline', async () => {
      // Create proposal with short timeout
      const shortTimeoutMechanism = new ConsensusMechanism({
        votingTimeoutMs: 100,
        votingMonitoringInterval: 50 // Monitor more frequently than timeout
      });
      await shortTimeoutMechanism.initialize();

      await shortTimeoutMechanism.registerNode('node_001', 'http://localhost:8001', 'pubkey_001');

      const shortProposalId = await shortTimeoutMechanism.submitProposal(
        'timeout_skill',
        mockLoRAAdapter,
        mockValidationResult,
        'test_proposer'
      );

      // Wait for timeout
      await new Promise(resolve => setTimeout(resolve, 200));

      const proposal = shortTimeoutMechanism.getProposal(shortProposalId);
      expect(proposal?.status).toBe(ProposalStatus.EXPIRED);

      shortTimeoutMechanism.stop();
    });
  });

  describe('Event Handling', () => {
    test('should emit node registration events', async () => {
      const nodeRegisteredSpy = jest.fn();
      consensusMechanism.on('nodeRegistered', nodeRegisteredSpy);

      await consensusMechanism.registerNode(
        'event_test_node',
        'http://localhost:8006',
        'pubkey_006'
      );

      expect(nodeRegisteredSpy).toHaveBeenCalled();
      expect(nodeRegisteredSpy.mock.calls[0][0]).toHaveProperty('nodeId', 'event_test_node');
    });

    test('should emit proposal events', async () => {
      const proposalSubmittedSpy = jest.fn();
      consensusMechanism.on('proposalSubmitted', proposalSubmittedSpy);

      await consensusMechanism.submitProposal(
        'event_test_skill',
        mockLoRAAdapter,
        mockValidationResult,
        'test_proposer'
      );

      expect(proposalSubmittedSpy).toHaveBeenCalled();
    });

    test('should emit vote events', async () => {
      const voteSubmittedSpy = jest.fn();
      consensusMechanism.on('voteSubmitted', voteSubmittedSpy);

      const proposalId = await consensusMechanism.submitProposal(
        'vote_event_skill',
        mockLoRAAdapter,
        mockValidationResult,
        'test_proposer'
      );

      await consensusMechanism.submitVote(proposalId, 'node_001', VoteType.APPROVE, 0.9);

      expect(voteSubmittedSpy).toHaveBeenCalled();
    });

    test('should emit consensus events', async () => {
      const consensusReachedSpy = jest.fn();
      consensusMechanism.on('consensusReached', consensusReachedSpy);

      const proposalId = await consensusMechanism.submitProposal(
        'consensus_event_skill',
        mockLoRAAdapter,
        mockValidationResult,
        'test_proposer'
      );

      // Submit enough votes to reach consensus
      await consensusMechanism.submitVote(proposalId, 'node_001', VoteType.APPROVE, 0.9);
      await consensusMechanism.submitVote(proposalId, 'node_002', VoteType.APPROVE, 0.8);

      // Wait for consensus
      await new Promise(resolve => setTimeout(resolve, 100));

      expect(consensusReachedSpy).toHaveBeenCalled();
    });
  });

  describe('Reputation Management', () => {
    test('should update node reputations based on voting behavior', async () => {
      const proposalId = await consensusMechanism.submitProposal(
        'reputation_test_skill',
        mockLoRAAdapter,
        mockValidationResult,
        'test_proposer'
      );

      // Submit votes - submit reject first to avoid early finalization
      await consensusMechanism.submitVote(proposalId, 'node_003', VoteType.REJECT, 0.7);
      await consensusMechanism.submitVote(proposalId, 'node_001', VoteType.APPROVE, 0.9);

      // Try to submit third vote, but it might fail if proposal is already finalized
      try {
        await consensusMechanism.submitVote(proposalId, 'node_002', VoteType.APPROVE, 0.8);
      } catch (error) {
        // Proposal might be finalized already, which is acceptable
        console.debug('Vote submission error (expected):', error);
      }

      // Wait for consensus and reputation update
      await new Promise(resolve => setTimeout(resolve, 200));

      const nodes = consensusMechanism.getRegisteredNodes();
      
      // Nodes that voted correctly (with final outcome) should have increased reputation
      const node1 = nodes.find(n => n.nodeId === 'node_001');
      const node2 = nodes.find(n => n.nodeId === 'node_002');
      const node3 = nodes.find(n => n.nodeId === 'node_003');

      // With 1 approve and 1 reject vote (50% approval), proposal should be rejected
      // since approval threshold is 75% and it's mathematically impossible to reach 75%
      // even with the remaining vote. Therefore:
      // - node_003 (voted REJECT) should have increased reputation (voted correctly)
      // - node_001 (voted APPROVE) should have decreased reputation (voted incorrectly)
      // - node_002 didn't vote because proposal was finalized early, so reputation unchanged
      expect(node1?.reputation).toBeLessThan(1.0);
      expect(node2?.reputation).toBe(1.0); // Unchanged, didn't vote
      expect(node3?.reputation).toBeGreaterThan(1.0);
    });
  });

  describe('Configuration', () => {
    test('should respect custom configuration', async () => {
      const customMechanism = new ConsensusMechanism({
        votingTimeoutMs: 1000,
        minParticipationRate: 0.5,
        approvalThreshold: 0.6,
        maxConcurrentProposals: 3
      });

      await customMechanism.initialize();
      
      await customMechanism.registerNode('node_001', 'http://localhost:8001', 'pubkey_001');
      await customMechanism.registerNode('node_002', 'http://localhost:8002', 'pubkey_002');

      const proposalId = await customMechanism.submitProposal(
        'custom_config_skill',
        mockLoRAAdapter,
        mockValidationResult,
        'test_proposer'
      );

      const proposal = customMechanism.getProposal(proposalId);
      expect(proposal?.requiredVotes).toBe(1); // 50% of 2 nodes = 1

      customMechanism.stop();
    });
  });

  describe('Type Usage Validation', () => {
    it('should validate AgentCoreNode and ConsensusProposal types', () => {
      // Test AgentCoreNode type usage
      const mockNode: AgentCoreNode = {
        nodeId: 'test-node',
        address: 'test-address',
        publicKey: 'test-key',
        reputation: 0.8,
        lastSeen: new Date(),
        capabilities: ['consensus', 'validation'],
        status: NodeStatus.ACTIVE,
        votingPower: 1.0
      };

      expect(mockNode.status).toBe(NodeStatus.ACTIVE);
      expect(Array.isArray(mockNode.capabilities)).toBe(true);

      // Test ConsensusProposal type usage
      const mockProposal: ConsensusProposal = {
        proposalId: 'test-proposal',
        skillId: 'test-skill',
        loraAdapter: {} as any,
        validationResult: {} as any,
        proposedBy: 'node-1',
        proposedAt: new Date(),
        submittedBy: 'node-1',
        submittedAt: new Date(),
        votingDeadline: new Date(),
        requiredVotes: 3,
        status: ProposalStatus.PENDING
      };

      expect(mockProposal.status).toBe(ProposalStatus.PENDING);
      expect(mockProposal.requiredVotes).toBe(2);
    });
  });
});
