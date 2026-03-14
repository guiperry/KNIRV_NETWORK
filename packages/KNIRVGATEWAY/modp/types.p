// types.p - P Language Type Definitions for KNIRVORACLE ModP Module
// This file defines the data types used across all state machines in the oracle system.

// =====================================================================
// Address and Identity Types
// =====================================================================

// Address represents a 20-byte cryptographic address
type Address = (bytes: seq[int]);

// ChainID identifies a blockchain in the network
type ChainID = (id: string);

// =====================================================================
// Token and Economic Types
// =====================================================================

// BigInt represents large integer values for token amounts
type BigInt = (value: int, isNegative: bool);

// TokenInfo contains basic token information
type TokenInfo = (
    name: string,
    symbol: string,
    decimals: int,
    totalSupply: BigInt,
    maxSupply: BigInt,
    owner: Address
);

// TransferReceipt represents a token transfer receipt
type TransferReceipt = (
    txHash: string,
    from: Address,
    to: Address,
    amount: BigInt,
    timestamp: int,
    success: bool
);

// MintReceipt represents a token mint receipt
type MintReceipt = (
    txHash: string,
    to: Address,
    amount: BigInt,
    newTotalSupply: BigInt,
    timestamp: int,
    success: bool
);

// BurnReceipt represents a token burn receipt
type BurnReceipt = (
    txHash: string,
    from: Address,
    amount: BigInt,
    reason: string,
    newTotalSupply: BigInt,
    timestamp: int,
    success: bool
);

// FeeType enumeration
type FeeType = (feeName: string);

// FeeReceipt represents a fee payment receipt
type FeeReceipt = (
    txHash: string,
    from: Address,
    feeType: FeeType,
    amount: BigInt,
    burnAmount: BigInt,
    rewardAmount: BigInt,
    timestamp: int
);

// =====================================================================
// Consensus Types
// =====================================================================

// RoundStep represents the current step in the consensus round
type RoundStep = (stepName: string);

// Block represents a blockchain block
type Block = (
    height: int,
    chainID: string,
    time: int,
    proposer: Address,
    txCount: int,
    appHash: seq[int]
);

// BlockHeader represents block header information
type BlockHeader = (
    chainID: string,
    height: int,
    time: int,
    proposer: Address,
    validatorsHash: seq[int]
);

// ConsensusValidator represents a validator in consensus
type ConsensusValidator = (
    address: Address,
    pubKey: seq[int],
    votingPower: int,
    proposerPriority: int
);

// RoundState represents current consensus round state
type RoundState = (
    height: int,
    round: int,
    step: RoundStep,
    validatorCount: int
);

// =====================================================================
// Governance Types
// =====================================================================

// ProposalStatus enumeration
type ProposalStatus = (statusName: string);

// ProposalType enumeration
type ProposalType = (typeName: string);

// VoteOption enumeration
type VoteOption = (option: int);  // 0=Abstain, 1=Yes, 2=No, 3=NoWithVeto

// Proposal represents a governance proposal
type Proposal = (
    id: string,
    proposalType: ProposalType,
    title: string,
    description: string,
    proposer: Address,
    status: ProposalStatus,
    submitTime: int,
    votingStart: int,
    votingEnd: int,
    deposit: BigInt
);

// Vote represents a cast vote
type Vote = (
    proposalID: string,
    voter: Address,
    option: VoteOption,
    votingPower: BigInt,
    timestamp: int
);

// TallyResult represents the result of vote tallying
type TallyResult = (
    yesVotes: BigInt,
    noVotes: BigInt,
    abstainVotes: BigInt,
    vetoVotes: BigInt,
    totalVoted: BigInt
);

// NetworkParameters represents configurable network parameters
type NetworkParameters = (
    votingPeriod: int,
    proposalDeposit: BigInt,
    quorumThreshold: int,  // percentage as int (0-100)
    passThreshold: int,    // percentage as int (0-100)
    vetoThreshold: int,    // percentage as int (0-100)
    minValidatorStake: BigInt,
    maxValidators: int
);

// Validator represents a network validator
type Validator = (
    address: Address,
    operatorAddress: Address,
    pubKey: seq[int],
    stakedAmount: BigInt,
    commissionRate: int,  // percentage (0-100)
    active: bool,
    jailed: bool,
    joinedAt: int
);

// =====================================================================
// Economics Types
// =====================================================================

// EconomicMetrics represents the current economic state
type EconomicMetrics = (
    totalSupply: BigInt,
    maxSupply: BigInt,
    circulatingSupply: BigInt,
    totalStaked: BigInt,
    totalBurned: BigInt,
    totalFeesCollected: BigInt,
    totalRewardsDistributed: BigInt,
    rewardPoolBalance: BigInt,
    stakingAPY: int,  // percentage * 100
    burnRate: int     // percentage * 100
);

// StakeInfo represents staking information for an address
type StakeInfo = (
    staker: Address,
    stakedAmount: BigInt,
    rewards: BigInt,
    stakedAt: int
);

// =====================================================================
// IBC Types
// =====================================================================

// ChannelState enumeration
type ChannelState = (stateName: string);

// ConnectionState enumeration
type ConnectionState = (stateName: string);

// ClientState enumeration
type ClientState = (stateName: string);

// ChannelOrdering enumeration
type ChannelOrdering = (orderingName: string);

// IBCChannel represents an IBC channel
type IBCChannel = (
    channelID: string,
    connectionID: string,
    portID: string,
    state: ChannelState,
    version: string,
    ordering: ChannelOrdering,
    chainID: ChainID,
    createdAt: int
);

// IBCConnection represents an IBC connection
type IBCConnection = (
    connectionID: string,
    clientID: string,
    state: ConnectionState,
    chainID: ChainID,
    endpoint: string,
    createdAt: int
);

// IBCClient represents an IBC light client
type IBCClient = (
    clientID: string,
    clientType: string,
    chainID: ChainID,
    state: ClientState,
    latestHeight: int,
    trustingPeriod: int,
    createdAt: int
);

// IBCPacket represents an IBC packet
type IBCPacket = (
    sequence: int,
    sourceChainID: ChainID,
    destChainID: ChainID,
    sourceChannel: string,
    destChannel: string,
    data: seq[int],
    timeoutHeight: int,
    timeoutTimestamp: int
);

// =====================================================================
// Cross-Chain Types
// =====================================================================

// TransferStatus enumeration
type TransferStatus = (statusName: string);

// CrossChainTransfer represents a cross-chain transfer request
type CrossChainTransfer = (
    transferID: string,
    sourceChainID: ChainID,
    destChainID: ChainID,
    sender: Address,
    recipient: Address,
    amount: BigInt,
    status: TransferStatus,
    initiatedAt: int,
    completedAt: int
);

// =====================================================================
// P2P Types
// =====================================================================

// PeerInfo represents information about a connected peer
type PeerInfo = (
    peerID: string,
    address: string,
    connectedAt: int,
    isActive: bool
);

// P2PStats represents P2P network statistics
type P2PStats = (
    peerCount: int,
    inboundPeers: int,
    outboundPeers: int,
    isRunning: bool
);

// =====================================================================
// "Second Me" Registration Types (from p_implementation.md)
// =====================================================================

// SecondMeRegistration represents a user's "Second Me" registration
type SecondMeRegistration = (
    secondMeID: string,
    user: Address,
    initialData: map[string, any],
    feePaid: BigInt,
    registeredAt: int
);

// =====================================================================
// Error Types
// =====================================================================

// ErrorCode represents error types that can occur
type ErrorCode = (code: int, message: string);

// Standard error codes
fun InsufficientBalance(): ErrorCode { return (code = 1, message = "Insufficient balance"); }
fun InvalidAmount(): ErrorCode { return (code = 2, message = "Invalid amount"); }
fun ExceedsMaxSupply(): ErrorCode { return (code = 3, message = "Exceeds max supply"); }
fun ProposalNotFound(): ErrorCode { return (code = 4, message = "Proposal not found"); }
fun VotingNotActive(): ErrorCode { return (code = 5, message = "Voting not active"); }
fun AlreadyVoted(): ErrorCode { return (code = 6, message = "Already voted"); }
fun InsufficientDeposit(): ErrorCode { return (code = 7, message = "Insufficient deposit"); }
fun ChannelNotFound(): ErrorCode { return (code = 8, message = "Channel not found"); }
fun TransferFailed(): ErrorCode { return (code = 9, message = "Transfer failed"); }
fun UnauthorizedMint(): ErrorCode { return (code = 10, message = "Unauthorized mint"); }
