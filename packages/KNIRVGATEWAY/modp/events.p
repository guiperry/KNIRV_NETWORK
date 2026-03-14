// events.p - P Language Event Definitions for KNIRVORACLE ModP Module
// This file defines all events used for communication between state machines.

// =====================================================================
// Token Machine Events
// =====================================================================

// Events sent to TokenMachine
event eTransferRequest: (from: Address, to: Address, amount: BigInt);
event eMintRequest: (to: Address, amount: BigInt, authorized: bool);
event eBurnRequest: (from: Address, amount: BigInt, reason: string);
event eGetBalance: (address: Address);
event eGetTokenInfo: ();

// Events sent from TokenMachine
event eTransferResponse: (success: bool, receipt: TransferReceipt);
event eMintResponse: (success: bool, receipt: MintReceipt);
event eBurnResponse: (success: bool, receipt: BurnReceipt);
event eBalanceResponse: (address: Address, balance: BigInt);
event eTokenInfoResponse: (info: TokenInfo);
event eTransferFailed: (from: Address, to: Address, amount: BigInt, error: ErrorCode);
event eMintFailed: (to: Address, amount: BigInt, error: ErrorCode);
event eBurnFailed: (from: Address, amount: BigInt, error: ErrorCode);

// =====================================================================
// Consensus Machine Events
// =====================================================================

// Events sent to ConsensusMachine
event eInitChain: (chainID: string, validators: seq[ConsensusValidator]);
event eProposeBlock: (txs: seq[seq[int]]);
event eCommitBlock: (block: Block);
event eUpdateValidators: (validators: seq[ConsensusValidator]);
event eGetConsensusState: ();

// Events sent from ConsensusMachine
event eBlockProposed: (block: Block);
event eBlockCommitted: (block: Block, appHash: seq[int]);
event eValidatorsUpdated: (validators: seq[ConsensusValidator]);
event eConsensusStateResponse: (state: RoundState);
event eBlockProposalFailed: (error: ErrorCode);
event eBlockCommitFailed: (block: Block, error: ErrorCode);

// =====================================================================
// Governance Machine Events
// =====================================================================

// Events sent to GovernanceMachine
event eCreateProposal: (
    proposalType: ProposalType,
    title: string,
    description: string,
    proposer: Address,
    deposit: BigInt,
    content: map[string, any]
);
event eCastVote: (proposalID: string, voter: Address, option: VoteOption, votingPower: BigInt);
event eGetProposal: (proposalID: string);
event eListProposals: ();
event eUpdateProposalStatus: (proposalID: string);
event eCancelProposal: (proposalID: string, requester: Address);
event eGetNetworkParams: ();
event eUpdateNetworkParams: (params: NetworkParameters);

// Events sent from GovernanceMachine
event eProposalCreated: (proposal: Proposal);
event eVoteCast: (vote: Vote);
event eProposalResponse: (proposal: Proposal);
event eProposalsListResponse: (proposals: seq[Proposal]);
event eProposalPassed: (proposalID: string, tallyResult: TallyResult);
event eProposalRejected: (proposalID: string, tallyResult: TallyResult);
event eProposalCancelled: (proposalID: string);
event eNetworkParamsResponse: (params: NetworkParameters);
event eNetworkParamsUpdated: (params: NetworkParameters);
event eProposalCreationFailed: (error: ErrorCode);
event eVoteFailed: (proposalID: string, voter: Address, error: ErrorCode);

// =====================================================================
// Economics Machine Events
// =====================================================================

// Events sent to EconomicsMachine
event eProcessFee: (from: Address, feeType: FeeType, amount: BigInt);
event eDistributeRewards: (stakers: seq[StakeInfo], totalAmount: BigInt);
event eStake: (staker: Address, amount: BigInt, validator: Address);
event eUnstake: (staker: Address, amount: BigInt);
event eClaimRewards: (staker: Address);
event eGetEconomicMetrics: ();
event eGetStakeInfo: (staker: Address);

// Events sent from EconomicsMachine
event eFeeProcessed: (receipt: FeeReceipt);
event eRewardsDistributed: (totalAmount: BigInt, recipientCount: int);
event eStaked: (staker: Address, amount: BigInt, totalStaked: BigInt);
event eUnstaked: (staker: Address, amount: BigInt, remainingStaked: BigInt);
event eRewardsClaimed: (staker: Address, amount: BigInt);
event eEconomicMetricsResponse: (metrics: EconomicMetrics);
event eStakeInfoResponse: (stakeInfo: StakeInfo);
event eFeeProcessingFailed: (from: Address, error: ErrorCode);
event eStakingFailed: (staker: Address, error: ErrorCode);

// Treasury-specific events (for TreasuryInvariant monitor)
event eTreasuryDeposit: (amount: BigInt, source: string);
event eTreasuryWithdrawal: (amount: BigInt, recipient: Address, approvedByGovernance: bool);
event eTreasuryBalanceChanged: (oldBalance: BigInt, newBalance: BigInt);

// =====================================================================
// IBC Machine Events
// =====================================================================

// Events sent to IBCMachine
event eRegisterChannel: (
    channelID: string,
    connectionID: string,
    portID: string,
    chainID: ChainID,
    ordering: ChannelOrdering
);
event eRegisterConnection: (
    connectionID: string,
    clientID: string,
    endpoint: string,
    chainID: ChainID
);
event eRegisterClient: (
    clientID: string,
    clientType: string,
    chainID: ChainID,
    trustingPeriod: int
);
event eOpenChannel: (channelID: string);
event eOpenConnection: (connectionID: string);
event eSendPacket: (packet: IBCPacket);
event eReceivePacket: (packet: IBCPacket);
event eAcknowledgePacket: (sequence: int, channelID: string, success: bool);
event eGetChannel: (channelID: string);
event eListChannels: ();
event eListConnections: ();

// Events sent from IBCMachine
event eChannelRegistered: (channel: IBCChannel);
event eConnectionRegistered: (connection: IBCConnection);
event eClientRegistered: (client: IBCClient);
event eChannelOpened: (channelID: string);
event eConnectionOpened: (connectionID: string);
event ePacketSent: (packet: IBCPacket);
event ePacketReceived: (packet: IBCPacket);
event ePacketAcknowledged: (sequence: int, channelID: string, success: bool);
event eChannelResponse: (channel: IBCChannel);
event eChannelsListResponse: (channels: seq[IBCChannel]);
event eConnectionsListResponse: (connections: seq[IBCConnection]);
event eIBCOperationFailed: (operation: string, error: ErrorCode);

// =====================================================================
// P2P Interface Events
// =====================================================================

// Events sent to P2PInterface
event eStartP2P: ();
event eStopP2P: ();
event ePauseP2P: ();
event eResumeP2P: ();
event eBroadcastMessage: (messageType: string, data: seq[int]);
event eSendToPeer: (peerID: string, messageType: string, data: seq[int]);
event eConnectToPeer: (peerAddress: string);
event eDisconnectPeer: (peerID: string);
event eGetP2PStatus: ();
event eListPeers: ();

// Events sent from P2PInterface
event eP2PStarted: ();
event eP2PStopped: ();
event eP2PPaused: ();
event eP2PResumed: ();
event eMessageBroadcasted: (messageType: string);
event eMessageSent: (peerID: string, messageType: string);
event ePeerConnected: (peerInfo: PeerInfo);
event ePeerDisconnected: (peerID: string);
event eP2PStatusResponse: (stats: P2PStats);
event ePeersListResponse: (peers: seq[PeerInfo]);
event eMessageReceived: (fromPeer: string, messageType: string, data: seq[int]);
event eP2PError: (operation: string, error: ErrorCode);

// =====================================================================
// Cross-Chain Transfer Events
// =====================================================================

// Events sent to CrossChainRouter
event eInitiateTransfer: (transfer: CrossChainTransfer);
event eCompleteTransfer: (transferID: string, success: bool);
event eGetTransfer: (transferID: string);
event eListPendingTransfers: ();

// Events sent from CrossChainRouter
event eTransferInitiated: (transfer: CrossChainTransfer);
event eTransferCompleted: (transfer: CrossChainTransfer);
event eTransferFailed_CC: (transferID: string, error: ErrorCode);
event eTransferResponse: (transfer: CrossChainTransfer);
event ePendingTransfersResponse: (transfers: seq[CrossChainTransfer]);

// =====================================================================
// "Second Me" Registration Workflow Events
// (From Section 5.2 of p_implementation.md)
// =====================================================================

// Events from AppLayer/External Client
event eRegisterRequest: (user: Address, initialData: map[string, any]);
event eFeePayment: (user: Address, amount: BigInt);

// Events from GovernanceMachine to EconomicsMachine
event eVerifyFeePayment: (user: Address, expectedAmount: BigInt);

// Response events
event eRegistrationConfirm: (user: Address, secondMeID: string);
event eRegistrationFail: (user: Address, reason: string);
event eFeePaymentVerified: (user: Address, amount: BigInt, verified: bool);

// =====================================================================
// Monitor Notification Events
// (Used by specification machines to report violations)
// =====================================================================

// General monitor events
event eMonitorViolation: (monitorName: string, violation: string, details: map[string, any]);
event eMonitorAssertion: (monitorName: string, assertion: string, passed: bool);

// Specific invariant events
event eTreasuryInvariantViolation: (expectedBalance: BigInt, actualBalance: BigInt, operation: string);
event eTokenSupplyInvariantViolation: (totalSupply: BigInt, maxSupply: BigInt, unauthorized: bool);
event eGovernanceInvariantViolation: (proposalID: string, violation: string);

// =====================================================================
// Lifecycle Events
// =====================================================================

// Oracle lifecycle events
event eOracleStart: ();
event eOracleStop: ();
event eOracleReady: ();
event eOracleStopped: ();
event eOracleError: (component: string, error: ErrorCode);

// Component lifecycle events
event eComponentStarted: (componentName: string);
event eComponentStopped: (componentName: string);
event eComponentReady: (componentName: string);
event eComponentError: (componentName: string, error: ErrorCode);
