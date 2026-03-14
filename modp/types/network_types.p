// network_types.p - Shared Type Definitions for KNIRV Network
// Defines common types used across all KNIRV components
// Based on: KNIRV Network Architecture and p_implementation.md

// =====================================================================
// CORE PRIMITIVE TYPES
// =====================================================================

// Address type for all network participants
type Address = (bytes: seq[int]);

// Big integer for token amounts and large numbers
type BigInt = (value: int, isNegative: bool);

// Hash type for cryptographic hashes
type Hash = (bytes: seq[int]);

// Timestamp in Unix milliseconds
type Timestamp = (milliseconds: int);

// Unique identifier
type UUID = (value: string);

// =====================================================================
// NETWORK IDENTITY TYPES
// =====================================================================

// Network node identifier
type NodeID = (
    id: string,
    publicKey: seq[int],
    nodeType: NodeType
);

// Types of nodes in the network
type NodeType = (typeName: string);
// Values: "validator", "full", "light", "archive", "router", "nexus"

// Peer information for P2P networking
type PeerInfo = (
    nodeID: NodeID,
    address: string,
    port: int,
    lastSeen: Timestamp,
    reputation: int
);

// =====================================================================
// BLOCKCHAIN COMMON TYPES
// =====================================================================

// Block header common to all chains
type BlockHeader = (
    height: int,
    chainID: string,
    previousHash: Hash,
    stateRoot: Hash,
    timestamp: Timestamp,
    proposer: Address
);

// Transaction common structure
type Transaction = (
    id: UUID,
    chainID: string,
    sender: Address,
    nonce: int,
    gasLimit: BigInt,
    payload: seq[int],
    signature: seq[int]
);

// Transaction receipt
type TransactionReceipt = (
    txHash: Hash,
    status: bool,
    gasUsed: BigInt,
    blockHeight: int,
    logs: seq[EventLog]
);

// Event log entry
type EventLog = (
    address: Address,
    topics: seq[Hash],
    payload: seq[int]
);

// =====================================================================
// KNIRVCHAIN SPECIFIC TYPES
// =====================================================================

// Skill definition
type Skill = (
    id: UUID,
    name: string,
    version: string,
    owner: Address,
    loraPointer: string,
    metadataPayload: map[string, any],
    createdAt: Timestamp,
    status: SkillStatus
);

type SkillStatus = (status: string);
// Values: "pending", "active", "deprecated", "revoked"

// LLM (Large Language Model) registration
type LLMRegistration = (
    id: UUID,
    name: string,
    provider: Address,
    modelType: string,
    apiEndpoint: string,
    capabilities: seq[string],
    stakingAmount: BigInt,
    status: LLMStatus
);

type LLMStatus = (status: string);
// Values: "registered", "active", "suspended", "deregistered"

// MCP (Model Context Protocol) capability
type MCPCapability = (
    id: UUID,
    name: string,
    version: string,
    serverPointer: string,
    owner: Address,
    permissions: seq[string],
    status: MCPStatus
);

type MCPStatus = (status: string);
// Values: "available", "restricted", "deprecated"

// MCP invocation result
type MCPCapabilityResultData = (
    capabilityID: UUID,
    result: map[string, any],
    success: bool
);

// Node transformation types
type ErrorNode = (
    id: UUID,
    errorType: string,
    errorMessage: string,
    contextPayload: map[string, any],
    sourceModel: UUID,
    timestamp: Timestamp,
    status: NodeStatus
);

type SkillNode = (
    id: UUID,
    derivedFrom: UUID,  // ErrorNode ID
    loraAdapter: string,
    trainer: Address,
    validationScore: int,
    reward: BigInt,
    timestamp: Timestamp
);

type ContextNode = (
    id: UUID,
    contextData: map[string, any],
    sourceInteraction: UUID,
    timestamp: Timestamp
);

type CapabilityNode = (
    id: UUID,
    derivedFrom: UUID,  // ContextNode ID
    mcpServer: string,
    creator: Address,
    usageCount: int,
    timestamp: Timestamp
);

type IdeaNode = (
    id: UUID,
    ideaContent: string,
    proposer: Address,
    votes: int,
    timestamp: Timestamp
);

type PropertyNode = (
    id: UUID,
    derivedFrom: UUID,  // IdeaNode ID
    inferenceNFT: string,
    owner: Address,
    royaltyRate: int,
    timestamp: Timestamp
);

type NodeStatus = (status: string);
// Values: "pending", "validated", "rejected", "transformed"

// Node transformation failure types
type SkillMiningFailedData = (
    errorNodeID: UUID,
    reason: string
);

type CapabilityMintingFailedData = (
    contextNodeID: UUID,
    reason: string
);

type PropertyMakingFailedData = (
    ideaNodeID: UUID,
    reason: string
);

// Successful transformation event data types
type SkillNodeMinedData = (
    skillNode: SkillNode,
    reward: BigInt
);

type IdeaVotedData = (
    ideaNodeID: UUID,
    votes: int
);

// =====================================================================
// KNIRVGRAPH SPECIFIC TYPES
// =====================================================================

// Knowledge graph node
type KnowledgeNode = (
    id: UUID,
    nodeType: KnowledgeNodeType,
    contentPayload: map[string, any],
    embedding: seq[float],
    connections: seq[UUID],
    createdAt: Timestamp,
    updatedAt: Timestamp
);

type KnowledgeNodeType = (typeName: string);
// Values: "error", "solution", "pattern", "insight", "reference"

// Error record in knowledge graph
type ErrorRecord = (
    id: UUID,
    errorHash: Hash,
    errorType: string,
    stackTrace: string,
    contextPayload: map[string, any],
    occurrences: int,
    firstSeen: Timestamp,
    lastSeen: Timestamp,
    solutions: seq[UUID]
);

// Solution record in knowledge graph
type SolutionRecord = (
    id: UUID,
    forError: UUID,
    solutionType: string,
    content: string,
    author: Address,
    effectiveness: int,  // 0-100 score
    validations: int,
    timestamp: Timestamp
);

// Pattern detected in errors
type ErrorPattern = (
    id: UUID,
    patternName: string,
    description: string,
    errorTypes: seq[string],
    frequency: int,
    suggestedSolutions: seq[UUID]
);

// KNIRVGRAPH event payload types
type ErrorDuplicateData = (
    existingID: UUID,
    occurrences: int
);

type SolutionValidatedData = (
    solutionID: UUID,
    newEffectiveness: int
);

type PatternUpdatedData = (
    patternID: UUID,
    occurrenceCount: int
);

type NodeLinkData = (
    sourceID: UUID,
    targetID: UUID
);

type KnowledgeGraphResultData = (
    nodes: seq[KnowledgeNode],
    patterns: seq[ErrorPattern]
);

// =====================================================================
// KNIRVROUTER SPECIFIC TYPES
// =====================================================================

// Router event payload types
type PeerConnectionFailedData = (
    peer: PeerInfo,
    reason: string
);

type MessageRoutedData = (
    messageID: UUID,
    hopCount: int
);

type MessageRoutingFailedData = (
    messageID: UUID,
    reason: string
);

type ConnectivityProofVerifiedData = (
    proof: ConnectivityProof,
    valid: bool
);

type ConnectivityChallengedData = (
    prover: NodeID,
    challenger: NodeID
);

type ConnectivityChallengeResultData = (
    prover: NodeID,
    passed: bool
);

type BroadcastDeliveredData = (
    messageID: UUID,
    deliveryCount: int
);

// Routing table entry
type RouteEntry = (
    destination: NodeID,
    nextHop: NodeID,
    metric: int,
    lastUpdated: Timestamp,
    status: RouteStatus
);

type RouteStatus = (status: string);
// Values: "active", "stale", "unreachable"

// Proof of Connectivity
type ConnectivityProof = (
    prover: NodeID,
    verifier: NodeID,
    proofData: seq[int],
    timestamp: Timestamp,
    validUntil: Timestamp,
    signature: seq[int]
);

// Network topology snapshot
type TopologySnapshot = (
    snapshotID: UUID,
    timestamp: Timestamp,
    nodeCount: int,
    edgeCount: int,
    avgConnectivity: int,
    partitions: seq[seq[NodeID]]
);

// Message routing
type RoutedMessage = (
    id: UUID,
    source: NodeID,
    destination: NodeID,
    payload: seq[int],
    ttl: int,
    hops: seq[NodeID],
    timestamp: Timestamp
);

// =====================================================================
// KNIRVSERVER SPECIFIC TYPES
// =====================================================================

// Validation task
type ValidationTask = (
    id: UUID,
    taskType: ValidationTaskType,
    payload: map[string, any],
    submitter: Address,
    priority: int,
    deadline: Timestamp,
    status: ValidationStatus
);

type ValidationTaskType = (typeName: string);
// Values: "code_execution", "model_inference", "proof_verification", "data_validation"

type ValidationStatus = (status: string);
// Values: "queued", "running", "completed", "failed", "timeout"

// Validation result
type ValidationResult = (
    taskID: UUID,
    validator: NodeID,
    passed: bool,
    score: int,
    detailsPayload: map[string, any],
    executionTime: int,
    resourcesUsed: ResourceUsage,
    timestamp: Timestamp
);

// Resource usage tracking
type ResourceUsage = (
    cpuMillis: int,
    memoryBytes: int,
    storageBytes: int,
    networkBytes: int
);

// Execution sandbox configuration
type SandboxConfig = (
    maxCPU: int,
    maxMemory: int,
    maxStorage: int,
    maxNetwork: int,
    timeout: int,
    allowedSyscalls: seq[string],
    isolationLevel: string
);

// =====================================================================
// KNIRVBASE SPECIFIC TYPES
// =====================================================================

// Base layer data record
type BaseRecord = (
    id: UUID,
    dataType: string,
    payload: seq[int],
    merkleProof: seq[Hash],
    timestamp: Timestamp
);

// Data availability proof
type DAProof = (
    recordID: UUID,
    commitmentRoot: Hash,
    proofPath: seq[Hash],
    validator: Address,
    signature: seq[int]
);

// =====================================================================
// KNIRVORACLE SPECIFIC TYPES (Token & Governance)
// =====================================================================

// NRN Token types
type TokenBalance = (
    owner: Address,
    available: BigInt,
    staked: BigInt,
    locked: BigInt
);

type TransferReceipt = (
    from: Address,
    recipient: Address,
    amount: BigInt,
    fee: BigInt,
    timestamp: Timestamp,
    txHash: Hash
);

type TransferResponseData = (
    success: bool,
    receipt: TransferReceipt
);

type MintReceipt = (
    recipient: Address,
    amount: BigInt,
    reason: string,
    authorized: bool,
    timestamp: Timestamp
);

// Governance types
type Proposal = (
    id: string,
    proposalType: ProposalType,
    title: string,
    description: string,
    proposer: Address,
    deposit: BigInt,
    status: ProposalStatus,
    votingStart: Timestamp,
    votingEnd: Timestamp,
    yesVotes: BigInt,
    noVotes: BigInt,
    abstainVotes: BigInt,
    vetoVotes: BigInt,
    contentPayload: map[string, any]
);

type ProposalType = (typeName: string);
// Values: "NetworkParameter", "SoftwareUpgrade", "TreasurySpend", "EmergencyAction", "SecondMeRegistration"

type ProposalStatus = (status: int);
// Values: 0=Pending, 1=Active, 2=Passed, 3=Rejected, 4=Executed, 5=Vetoed

type VoteOption = (option: int);
// Values: 0=Abstain, 1=Yes, 2=No, 3=Veto

type Vote = (
    proposalID: string,
    voter: Address,
    option: VoteOption,
    votingPower: BigInt,
    timestamp: Timestamp
);

// Event payload types for P type system compatibility
type VoteFailedData = (
    proposalID: string,
    reason: string
);

type VoteCastData = (
    proposalID: string,
    voter: Address
);

type ProposalExecutionFailedData = (
    proposalID: string,
    reason: string
);

type FeeVerificationData = (
    user: Address,
    amount: BigInt
);

type RegistrationFailData = (
    user: Address,
    reason: string
);

type RegistrationConfirmData = (
    user: Address,
    registrationID: string
);

type ValidationRejectedReason = (reason: string);

// Economics event types
type TreasuryDepositData = (
    amount: BigInt,
    source: string
);

type TreasuryWithdrawalData = (
    amount: BigInt,
    destination: Address,
    proposalID: string
);

type FeeProcessedData = (
    user: Address,
    feePaid: BigInt,
    treasuryAmount: BigInt
);

type StakeData = (
    staker: Address,
    amount: BigInt,
    validator: Address
);

type UnstakeData = (
    staker: Address,
    amount: BigInt
);

// Second Me registration
type SecondMeRegistration = (
    user: Address,
    secondMeID: string,
    registrationPayload: map[string, any],
    feesPaid: BigInt,
    status: RegistrationStatus,
    timestamp: Timestamp
);

type RegistrationStatus = (status: string);
// Values: "pending", "verified", "confirmed", "rejected"

// Economics types
type FeeType = (feeName: string);

type StakeInfo = (
    staker: Address,
    validator: Address,
    amount: BigInt,
    startTime: Timestamp,
    lockPeriod: int,
    rewards: BigInt
);

type RewardDistribution = (
    recipient: Address,
    amount: BigInt,
    rewardType: string,
    epoch: int,
    timestamp: Timestamp
);

// IBC types
type IBCChannel = (
    channelID: string,
    portID: string,
    counterpartyChannelID: string,
    counterpartyPortID: string,
    connectionID: string,
    channelState: ChannelState,
    ordering: string
);

type ChannelState = (value: int);
// Values: 0=Uninitialized, 1=Init, 2=TryOpen, 3=Open, 4=Closed

// Wrapper type for IBC packet sequence numbers (enables set operations)
type SeqNumber = (value: int);

type IBCPacket = (
    sequence: int,
    sourcePort: string,
    sourceChannel: string,
    destPort: string,
    destChannel: string,
    payload: seq[int],
    timeoutHeight: int,
    timeoutTimestamp: Timestamp
);

// =====================================================================
// CROSS-COMPONENT TYPES
// =====================================================================

// Cross-chain message
type CrossChainMessage = (
    id: UUID,
    sourceChain: string,
    destChain: string,
    messageType: string,
    payload: seq[int],
    sender: Address,
    timestamp: Timestamp,
    status: MessageStatus
);

type MessageStatus = (status: string);
// Values: "pending", "relayed", "confirmed", "failed"

// Network-wide event for monitoring
type NetworkEvent = (
    eventID: UUID,
    component: string,
    eventType: string,
    payload: map[string, any],
    timestamp: Timestamp,
    severity: string
);

// Configuration for network parameters
type NetworkConfig = (
    chainID: string,
    blockTime: int,
    maxBlockSize: int,
    minStake: BigInt,
    maxValidators: int,
    epochLength: int
);
