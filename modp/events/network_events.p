// network_events.p - Shared Event Definitions for KNIRV Network
// Defines all inter-machine communication events across components
// Based on: KNIRV Network Architecture

// =====================================================================
// SYSTEM LIFECYCLE EVENTS
// =====================================================================

// Network startup events
event eNetworkStart;
event eNetworkReady;
event eNetworkShutdown;

// Component startup events
event eComponentStart: (componentName: string);
event eComponentReady: (componentName: string);
event eComponentShutdown: (componentName: string);

// =====================================================================
// KNIRVCHAIN EVENTS - Skill & LLM Registry
// =====================================================================

// Skill Registry Events
event eRegisterSkill: (name: string, version: string, owner: Address, loraPointer: string, metadata: map[string, any]);

event eSkillRegistered: Skill;
event eSkillRegistrationFailed: string;

event eUpdateSkill: (skillID: UUID, updates: map[string, any]);
event eSkillUpdated: Skill;

event eDeprecateSkill: (skillID: UUID);
event eSkillDeprecated: UUID;

event eQuerySkill: (skillID: UUID);
event eSkillQueryResult: Skill;

// LLM Registry Events
event eRegisterLLM: (name: string, provider: Address, modelType: string, apiEndpoint: string, capabilities: seq[string], stakingAmount: BigInt);

event eLLMRegistered: LLMRegistration;
event eLLMRegistrationFailed: string;

event eSuspendLLM: (llmID: UUID, reason: string);
event eLLMSuspended: UUID;

event eReactivateLLM: (llmID: UUID);
event eLLMReactivated: UUID;

// MCP Capability Events
event eRegisterMCPCapability: (name: string, version: string, serverPointer: string, owner: Address, permissions: seq[string]);

event eMCPCapabilityRegistered: MCPCapability;
event eMCPCapabilityFailed: string;

event eInvokeMCPCapability: (capabilityID: UUID, caller: Address, params: map[string, any]);
event eMCPCapabilityResult: (capabilityID: UUID, result: map[string, any], success: bool);

// Node Transformation Events - ErrorNode → SkillNode Mining
event eSubmitError: (errorType: string, errorMessage: string, context: map[string, any], sourceModel: UUID);
event eErrorNodeCreated: ErrorNode;

event eMineSkillFromError: (errorNodeID: UUID, trainer: Address, loraAdapter: string);
event eSkillNodeMined: (skillNode: SkillNode, reward: BigInt);
event eSkillMiningFailed: (errorNodeID: UUID, reason: string);

// Node Transformation Events - ContextNode → CapabilityNode Minting
event eSubmitContext: (contextData: map[string, any], sourceInteraction: UUID);
event eContextNodeCreated: ContextNode;

event eMintCapabilityFromContext: (contextNodeID: UUID, creator: Address, mcpServer: string);
event eCapabilityNodeMinted: (capabilityNode: CapabilityNode);
event eCapabilityMintingFailed: (contextNodeID: UUID, reason: string);

// Node Transformation Events - IdeaNode → PropertyNode Making
event eSubmitIdea: (ideaContent: string, proposer: Address);
event eIdeaNodeCreated: IdeaNode;

event eVoteOnIdea: (ideaNodeID: UUID, voter: Address, support: bool);
event eIdeaVoted: (ideaNodeID: UUID, votes: int);

event eMakePropertyFromIdea: (ideaNodeID: UUID, owner: Address, inferenceNFT: string, royaltyRate: int);
event ePropertyNodeMade: PropertyNode;
event ePropertyMakingFailed: (ideaNodeID: UUID, reason: string);

// =====================================================================
// KNIRVGRAPH EVENTS - Knowledge Graph
// =====================================================================

// Error Recording Events
event eRecordError: (errorType: string, errorMessage: string, context: map[string, any]);
event eErrorRecorded: (record: ErrorRecord);
event eErrorDuplicate: (existingID: UUID, occurrences: int);

// Solution Events
event eSubmitSolution: (forError: UUID, solutionType: string, content: string, author: Address);
event eSolutionSubmitted: (solution: SolutionRecord);
event eSolutionSubmissionFailed: string;

event eValidateSolution: (solutionID: UUID, validator: Address, effective: bool);
event eSolutionValidated: (solutionID: UUID, newEffectiveness: int);

// Knowledge Graph Query Events
event eQueryKnowledgeGraph: (queryType: string, params: map[string, any]);
event eKnowledgeGraphResult: (nodes: seq[KnowledgeNode], patterns: seq[ErrorPattern]);

// Pattern Detection Events
event ePatternDetected: (pattern: ErrorPattern);
event ePatternUpdated: (patternID: UUID, occurrenceCount: int);

// Graph Linking Events
event eLinkErrorToSolution: (errorID: UUID, solutionID: UUID);
event eErrorSolutionLinked: (errorID: UUID, solutionID: UUID);

event eLinkNodes: (sourceID: UUID, targetID: UUID, linkType: string);
event eNodesLinked: (sourceID: UUID, targetID: UUID);

// =====================================================================
// KNIRVROUTER EVENTS - P2P Networking
// =====================================================================

// Peer Discovery Events
event eDiscoverPeers;
event ePeersDiscovered: (peers: seq[PeerInfo]);

event eConnectToPeer: (peer: PeerInfo);
event ePeerConnected: (peer: PeerInfo);
event ePeerConnectionFailed: (peer: PeerInfo, reason: string);

event eDisconnectPeer: (nodeID: NodeID);
event ePeerDisconnected: (nodeID: NodeID);

// Routing Events
event eUpdateRoute: (routeEntry: RouteEntry);
event eRouteUpdated: (routeEntry: RouteEntry);

event eRouteMessage: (message: RoutedMessage);
event eMessageRouted: (messageID: UUID, hopCount: int);
event eMessageRoutingFailed: (messageID: UUID, reason: string);

event eQueryRoute: (destination: NodeID);
event eRouteQueryResult: (path: seq[NodeID]);

// Proof of Connectivity Events
event eGenerateConnectivityProof: (verifier: NodeID);
event eConnectivityProofGenerated: (proof: ConnectivityProof);

event eVerifyConnectivityProof: (proof: ConnectivityProof);
event eConnectivityProofVerified: (proof: ConnectivityProof, valid: bool);

event eConnectivityChallenged: (prover: NodeID, challenger: NodeID);
event eConnectivityChallengeResult: (prover: NodeID, passed: bool);

// Topology Events
event eRequestTopologySnapshot;
event eTopologySnapshotReady: (snapshot: TopologySnapshot);

event eNetworkPartitionDetected: (partitions: seq[seq[NodeID]]);
event eNetworkHealed: (timestamp: Timestamp);

// Broadcast Events
event eBroadcastMessage: (messageType: string, msgPayload: seq[int], ttl: int);
event eBroadcastDelivered: (messageID: UUID, deliveryCount: int);

// =====================================================================
// KNIRVNEXUS EVENTS - Distributed Validation
// =====================================================================

// Task Submission Events
event eSubmitValidationTask: (taskType: ValidationTaskType, taskPayload: map[string, any], submitter: Address, priority: int, deadline: Timestamp);
event eValidationTaskQueued: (task: ValidationTask);
event eValidationTaskRejected: (reason: string);

// Task Execution Events
event eStartValidation: (taskID: UUID, validator: NodeID);
event eValidationStarted: (taskID: UUID);

event eValidationProgress: (taskID: UUID, progressPercent: int, details: map[string, any]);

event eValidationComplete: (result: ValidationResult);
event eValidationFailed: (taskID: UUID, reason: string);
event eValidationTimeout: (taskID: UUID);

// Sandbox Events
event eCreateSandbox: (config: SandboxConfig);
event eSandboxCreated: (sandboxID: UUID);
event eSandboxCreationFailed: string;

event eExecuteInSandbox: (sandboxID: UUID, code: seq[int], inputs: map[string, any]);
event eSandboxExecutionResult: (sandboxID: UUID, output: map[string, any], resourcesUsed: ResourceUsage);
event eSandboxExecutionFailed: (sandboxID: UUID, reason: string);

event eDestroySandbox: (sandboxID: UUID);
event eSandboxDestroyed: (sandboxID: UUID);

// Resource Monitoring Events
event eResourceLimitExceeded: (sandboxID: UUID, resourceType: string, limit: int, actual: int);

event eValidatorOverloaded: (validatorID: NodeID, queueSize: int);
event eValidatorAvailable: (validatorID: NodeID);

// =====================================================================
// KNIRVBASE EVENTS - Base Layer
// =====================================================================

// Data Storage Events
event eStoreData: (dataType: string, payload: seq[int]);
event eDataStored: (record: BaseRecord);
event eDataStorageFailed: string;

event eRetrieveData: (recordID: UUID);
event eDataRetrieved: (record: BaseRecord);
event eDataNotFound: (recordID: UUID);

// Data Availability Events
event eRequestDAProof: (recordID: UUID);
event eDAProofGenerated: (proof: DAProof);
event eDAProofFailed: (recordID: UUID, reason: string);

event eVerifyDAProof: (proof: DAProof);
event eDAProofVerified: (recordID: UUID, valid: bool);

// =====================================================================
// KNIRVORACLE EVENTS - Token & Governance
// =====================================================================

// Token Events
event eTransferRequest: (from: Address, recipient: Address, amount: BigInt);
event eTransferResponse: TransferResponseData;
event eTransferFailed: string;

event eMintRequest: (recipient: Address, amount: BigInt, authorized: bool);
event eMintResponse: (success: bool, receipt: MintReceipt);
event eMintFailed: string;

event eBurnRequest: (from: Address, amount: BigInt);
event eBurnResponse: (success: bool, amount: BigInt);
event eBurnFailed: string;

event eQueryBalance: (address: Address);
event eBalanceResponse: TokenBalance;

// Governance Events
event eCreateProposal: (proposalType: ProposalType, title: string, description: string, proposer: Address, deposit: BigInt, content: map[string, any]);
event eProposalCreated: Proposal;
event eProposalCreationFailed: string;

event eCastVote: (proposalID: string, voter: Address, option: VoteOption, votingPower: BigInt);
event eVoteCast: VoteCastData;
event eVoteFailed: VoteFailedData;

event eProposalStatusChanged: (proposalID: string, oldStatus: ProposalStatus, newStatus: ProposalStatus);

event eExecuteProposal: (proposalID: string);
event eProposalExecuted: string;
event eProposalExecutionFailed: ProposalExecutionFailedData;

// Second Me Registration Events
event eRegisterRequest: (user: Address, metadata: map[string, any]);
event eVerifyFeePayment: FeeVerificationData;
event eFeePayment: (user: Address, amount: BigInt);
event eFeePaymentVerified: (user: Address, amount: BigInt, verified: bool);
event eRegistrationConfirm: RegistrationConfirmData;
event eRegistrationFail: RegistrationFailData;

// Economics Events
event eProcessFee: (from: Address, feeType: FeeType, amount: BigInt);
event eFeeProcessed: FeeProcessedData;
event eFeeProcessingFailed: string;

event eStake: StakeData;
event eStaked: StakeInfo;
event eStakeFailed: string;

event eUnstake: UnstakeData;
event eUnstaked: UnstakeData;
event eUnstakeFailed: string;

event eDistributeRewards: (epoch: int);
event eRewardsDistributed: seq[RewardDistribution];

// Treasury Events
event eTreasuryDeposit: TreasuryDepositData;
event eTreasuryWithdrawal: TreasuryWithdrawalData;
event eTreasuryBalanceQuery;
event eTreasuryBalanceResponse: BigInt;

// IBC Events
event eIBCChannelOpen: (channel: IBCChannel);
event eIBCChannelOpened: (channelID: string);
event eIBCChannelOpenFailed: (reason: string);

event eIBCChannelClose: (channelID: string);
event eIBCChannelClosed: (channelID: string);

event eIBCSendPacket: (packet: IBCPacket);
event eIBCPacketSent: (sequence: int);
event eIBCPacketFailed: (reason: string);

event eIBCReceivePacket: (packet: IBCPacket);
event eIBCPacketReceived: (sequence: int);
event eIBCPacketAcknowledged: (sequence: int);
event eIBCPacketTimeout: (sequence: int);

// =====================================================================
// CROSS-COMPONENT EVENTS
// =====================================================================

// Cross-chain messaging
event eSendCrossChainMessage: (message: CrossChainMessage);
event eCrossChainMessageSent: (messageID: UUID);
event eCrossChainMessageReceived: (message: CrossChainMessage);
event eCrossChainMessageFailed: (messageID: UUID, reason: string);

// Network coordination events
event eNetworkConfigUpdate: (config: NetworkConfig);
event eNetworkConfigUpdated: NetworkConfig;

event eEmergencyHalt: string;
event eEmergencyResume;

// =====================================================================
// MONITORING & ASSERTION EVENTS
// =====================================================================

// Monitor events for invariant checking
event eMonitorViolation: (monitorName: string, violationType: string, details: map[string, any]);

event eMonitorAssertion: (monitorName: string, assertionMessage: string, passed: bool);

event eNetworkHealthCheck;
event eNetworkHealthReport: (healthy: bool, componentStatus: map[string, bool], issues: seq[string]);
