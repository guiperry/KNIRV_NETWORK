# KNIRVORACLE Implementation Plan

## Overview

KNIRVORACLE is the Rust-based governance and economic orchestrator for the KNIRV network. This document plans the full implementation of cross-chain transfers, governance consolidation, and the deprecation of LLM/Skill registries (now handled by KNIRVCHAIN).

## Core Focus Areas

1. **Cross-Chain Transfers** - Full IBC implementation for asset transfers
2. **Network Governance** - Centralized governance authority for the D-TEN
3. **Economics Engine** - NRN token economics, staking, and rewards
4. **Registry Deprecation** - Move LLM/Skill registries to KNIRVCHAIN

---

## 1. Cross-Chain Transfer Implementation

### Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                        KNIRVORACLE                               │
│  ┌─────────────────┐  ┌──────────────────┐  ┌───────────────┐  │
│  │ Cross-Chain     │  │ Transfer         │  │ Bridge        │  │
│  │ Router          │  │ Processor        │  │ Validator     │  │
│  └────────┬────────┘  └────────┬─────────┘  └───────┬───────┘  │
│           │                    │                     │          │
│  ┌────────┴────────────────────┴─────────────────────┴───────┐  │
│  │                    IBC Handler (Enhanced)                  │  │
│  └────────┬─────────────────┬─────────────────────┬──────────┘  │
└───────────┼─────────────────┼─────────────────────┼─────────────┘
            │                 │                     │
    ┌───────▼───────┐ ┌───────▼───────┐    ┌───────▼───────┐
    │  KNIRVCHAIN   │ │  KNIRVNEXUS   │    │  XION/Cosmos  │
    │  (Go)         │ │  (DVE)        │    │  (External)   │
    └───────────────┘ └───────────────┘    └───────────────┘
```

### Data Structures

```rust
/// Cross-chain transfer request
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CrossChainTransfer {
    pub transfer_id: String,
    pub source_chain: ChainId,
    pub dest_chain: ChainId,
    pub sender: String,
    pub recipient: String,
    pub amount: u64,
    pub denom: String,                    // NRN, USDC, etc.
    pub timeout_height: u64,
    pub timeout_timestamp: u64,
    pub memo: Option<String>,
    pub status: TransferStatus,
    pub created_at: i64,
    pub completed_at: Option<i64>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum ChainId {
    KnirvChain,
    KnirvOracle,
    KnirvNexus,
    KnirvRouter,
    KnirvGraph,
    Xion,
    Cosmos(String),     // Generic Cosmos chain
    External(String),   // Non-Cosmos chains
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum TransferStatus {
    Pending,
    SourceLocked,       // Tokens locked on source chain
    InTransit,          // IBC packet sent
    DestReceived,       // Received on destination
    Completed,          // Fully finalized
    Failed(String),     // Error message
    Refunded,           // Tokens returned to sender
    TimedOut,
}

/// Transfer proof for validation
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TransferProof {
    pub transfer_id: String,
    pub merkle_proof: Vec<u8>,
    pub block_height: u64,
    pub block_hash: String,
    pub validator_signatures: Vec<ValidatorSignature>,
}

/// Bridge configuration per chain
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct BridgeConfig {
    pub chain_id: ChainId,
    pub channel_id: String,
    pub port_id: String,
    pub connection_id: String,
    pub client_id: String,
    pub trust_level: f64,
    pub max_transfer_amount: u64,
    pub min_transfer_amount: u64,
    pub transfer_fee_basis_points: u16,  // 100 = 1%
    pub enabled: bool,
}
```

### Transfer Flow

```rust
/// Cross-chain transfer processor
pub struct CrossChainRouter {
    ibc_handler: Arc<IBCHandler>,
    transfer_queue: Mutex<VecDeque<CrossChainTransfer>>,
    pending_transfers: Mutex<HashMap<String, CrossChainTransfer>>,
    bridge_configs: HashMap<ChainId, BridgeConfig>,
    economics: Arc<TokenEconomics>,
}

impl CrossChainRouter {
    /// Initiate a cross-chain transfer
    pub async fn initiate_transfer(
        &self,
        sender: &str,
        recipient: &str,
        amount: u64,
        denom: &str,
        dest_chain: ChainId,
    ) -> Result<CrossChainTransfer, TransferError> {
        // 1. Validate transfer parameters
        // 2. Check sender balance
        // 3. Calculate and deduct fees
        // 4. Lock tokens on source chain
        // 5. Create IBC packet
        // 6. Submit to IBC handler
        // 7. Return transfer receipt
    }

    /// Process incoming transfer from another chain
    pub async fn receive_transfer(
        &self,
        packet: IBCPacket,
        proof: TransferProof,
    ) -> Result<(), TransferError> {
        // 1. Verify proof authenticity
        // 2. Validate packet data
        // 3. Check destination address
        // 4. Mint/release tokens
        // 5. Send acknowledgement
    }

    /// Handle transfer timeout
    pub async fn handle_timeout(
        &self,
        transfer_id: &str,
    ) -> Result<(), TransferError> {
        // 1. Verify timeout conditions
        // 2. Refund locked tokens
        // 3. Update transfer status
        // 4. Emit timeout event
    }
}
```

### IBC Enhancement

Complete the existing IBC handler with actual packet transmission:

```rust
// File: src/ibc_handler.rs - Enhancement

impl IBCHandler {
    /// Send IBC packet with actual transmission
    pub async fn transmit_packet(
        &self,
        channel_id: &str,
        packet: IBCPacket,
    ) -> Result<PacketReceipt, IBCError> {
        let channel = self.get_channel(channel_id)?;

        // Serialize packet
        let packet_bytes = serde_json::to_vec(&packet)?;

        // Sign packet
        let signature = self.sign_packet(&packet_bytes)?;

        // Get connection endpoint
        let endpoint = self.get_connection_endpoint(&channel.connection_id)?;

        // Transmit via appropriate method
        match endpoint.transport {
            Transport::Grpc => self.transmit_grpc(endpoint, packet_bytes, signature).await,
            Transport::Http => self.transmit_http(endpoint, packet_bytes, signature).await,
            Transport::Websocket => self.transmit_ws(endpoint, packet_bytes, signature).await,
        }
    }

    /// Handle packet acknowledgement
    pub async fn process_acknowledgement(
        &self,
        ack: PacketAcknowledgement,
    ) -> Result<(), IBCError> {
        // Verify acknowledgement
        // Update packet status
        // Trigger completion callbacks
    }
}
```

### Supported Transfer Types

| Type | Source | Destination | Notes |
|------|--------|-------------|-------|
| NRN Transfer | Any KNIRV chain | Any KNIRV chain | Native token |
| USDC Bridge | XION | KNIRVORACLE | Via XION Meta Accounts |
| Skill Fee | KNIRVCHAIN | KNIRVORACLE | Automatic on skill invocation |
| Validation Reward | KNIRVORACLE | KNIRVNEXUS | DVE operator rewards |
| Staking Deposit | Any | KNIRVORACLE | Validator/developer stakes |
| Governance Fee | Any | KNIRVORACLE | Proposal submission fees |

---

## 2. Network Governance Implementation

### Governance Authority Transfer

KNIRVORACLE becomes the sole governance authority for network-wide decisions:

```rust
/// Enhanced governance system with full network authority
pub struct NetworkGovernance {
    pub governance: Arc<GovernanceSystem>,
    pub cross_chain_router: Arc<CrossChainRouter>,
    pub economics: Arc<TokenEconomics>,

    // Network-wide governance scope
    pub network_parameters: NetworkParameters,
    pub chain_registrations: HashMap<ChainId, ChainRegistration>,
    pub emergency_council: Vec<String>,  // Emergency multisig addresses
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct NetworkParameters {
    // Economic parameters
    pub base_transaction_fee: u64,
    pub skill_invocation_fee: u64,
    pub llm_registration_fee: u64,
    pub validation_reward: u64,

    // Governance parameters
    pub proposal_deposit: u64,
    pub voting_period_blocks: u64,
    pub quorum_threshold: f64,
    pub pass_threshold: f64,
    pub veto_threshold: f64,

    // Cross-chain parameters
    pub max_transfer_amount: u64,
    pub transfer_timeout_blocks: u64,
    pub bridge_fee_basis_points: u16,

    // Staking parameters
    pub min_validator_stake: u64,
    pub min_developer_stake: u64,
    pub unbonding_period_days: u32,
    pub slashing_penalty_percent: u8,
}

/// Extended proposal types for network governance
#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum NetworkProposalType {
    // Existing types
    ModelTransition(ModelTransitionData),
    NetworkParameter(ParameterChange),
    ValidatorUpdate(ValidatorAction),
    Emergency(EmergencyAction),

    // New cross-chain governance
    ChainRegistration(ChainRegistrationData),
    ChainDeregistration(ChainId),
    BridgeParameterUpdate(BridgeParameterChange),
    CrossChainEmergencyHalt(Vec<ChainId>),

    // Economic governance
    TokenomicsUpdate(TokenomicsChange),
    FeeStructureUpdate(FeeStructureChange),
    RewardDistributionUpdate(RewardDistributionChange),

    // Protocol upgrades
    ProtocolUpgrade(ProtocolUpgradeData),
    ChainFork(ChainForkData),
}
```

### Governance Flow

1. **Proposal Submission** (100 NRN deposit)
   - Any staked address can submit
   - Proposals routed to KNIRVORACLE
   - Cross-chain proposals require higher deposits

2. **Voting Period** (7 days default, 24 hours for emergency)
   - Validators vote with stake-weighted power
   - Delegators can override validator votes
   - Real-time tally via IBC to all chains

3. **Execution**
   - Passed proposals auto-execute via governance module
   - Cross-chain execution via IBC governance messages
   - Failed proposals refund deposit minus fee

### API Endpoints (Governance)

```
POST /v3/governance/propose              - Submit proposal
POST /v3/governance/vote                 - Cast vote
GET  /v3/governance/proposals            - List proposals
GET  /v3/governance/proposal/{id}        - Get proposal details
GET  /v3/governance/tally/{id}           - Get vote tally
POST /v3/governance/execute/{id}         - Execute passed proposal
GET  /v3/governance/parameters           - Get network parameters
GET  /v3/governance/validators           - List validators
POST /v3/governance/delegate             - Delegate stake
POST /v3/governance/undelegate           - Undelegate stake
```

---

## 3. Economics Engine Implementation

The economics engine remains in KNIRVORACLE and is enhanced for cross-chain operations:

### Fee Structure

```rust
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct FeeStructure {
    // Base fees (in NRN wei, 1 NRN = 1_000_000 wei)
    pub base_transaction_fee: u64,        // 1,000 wei
    pub cross_chain_transfer_fee: u64,    // 10,000 wei + basis points

    // Skill/Capability fees
    pub skill_invocation_fee: u64,        // 100,000 wei (0.1 NRN)
    pub capability_mint_fee: u64,         // 500,000 wei (0.5 NRN)
    pub property_mint_fee: u64,           // 1,000,000 wei (1 NRN)

    // Registration fees (moved to KNIRVCHAIN but collected here)
    pub llm_registration_fee: u64,        // 1,000,000 wei (1 NRN)
    pub skill_registration_fee: u64,      // 500,000 wei (0.5 NRN)

    // Governance fees
    pub proposal_deposit: u64,            // 100,000,000 wei (100 NRN)
    pub emergency_proposal_deposit: u64,  // 1,000,000,000 wei (1000 NRN)

    // Staking parameters
    pub min_validator_stake: u64,         // 100,000,000,000 wei (100K NRN)
    pub min_developer_stake: u64,         // 10,000,000,000 wei (10K NRN)
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RewardDistribution {
    pub validator_block_reward: u64,      // 10 NRN per block
    pub developer_block_reward: u64,      // 5 NRN per block
    pub community_pool_reward: u64,       // 2 NRN per block

    pub validation_reward: u64,           // 0.05 NRN per validation
    pub skill_creator_share: f64,         // 70% of invocation fee
    pub network_share: f64,               // 30% of invocation fee

    // Performance multipliers
    pub high_performance_multiplier: f64,  // 1.5x
    pub consistent_user_multiplier: f64,   // 1.2x
    pub early_adopter_multiplier: f64,     // 1.3x
    pub community_leader_multiplier: f64,  // 2.0x
}
```

### Cross-Chain Fee Collection

```rust
impl TokenEconomics {
    /// Process fee from any KNIRV chain
    pub async fn collect_cross_chain_fee(
        &self,
        source_chain: ChainId,
        fee_type: FeeType,
        amount: u64,
        payer: &str,
    ) -> Result<FeeReceipt, EconomicsError> {
        // 1. Verify fee payment via IBC
        // 2. Record in economics ledger
        // 3. Distribute to appropriate pools
        // 4. Return receipt
    }

    /// Distribute rewards across chains
    pub async fn distribute_cross_chain_rewards(
        &self,
        recipients: Vec<(ChainId, String, u64)>,
    ) -> Result<Vec<TransferReceipt>, EconomicsError> {
        // Batch cross-chain transfers for efficiency
    }
}
```

### API Endpoints (Economics)

```
GET  /v3/economics/metrics               - Get network economic metrics
GET  /v3/economics/supply                - Get token supply info
GET  /v3/economics/fees                  - Get current fee structure
GET  /v3/economics/rewards               - Get reward distribution
POST /v3/economics/stake                 - Stake tokens
POST /v3/economics/unstake               - Unstake tokens
GET  /v3/economics/staking/{address}     - Get staking info
GET  /v3/economics/burns                 - Get burn history
```

---

## 4. Registry Deprecation Plan

### Files to Modify

| File | Changes |
|------|---------|
| `nrn_token.rs` | Remove LLMRegistry and SkillRegistry structs |
| `blockchain_adapter.rs` | Remove registry methods, keep XION bridge |
| `smart_contracts.rs` | Remove registry contract calls |
| `economics_integration.rs` | Update to receive events from KNIRVCHAIN |
| `main.rs` | Remove registry initialization |

### Deprecation Steps

#### Phase 1: Mark Deprecated
```rust
// nrn_token.rs
#[deprecated(since = "2.0.0", note = "Use KNIRVCHAIN for LLM registration")]
pub struct LLMRegistry { ... }

#[deprecated(since = "2.0.0", note = "Use KNIRVCHAIN for skill registration")]
pub struct SkillRegistry { ... }
```

#### Phase 2: Add IBC Listeners
```rust
// New file: src/registry_listener.rs
pub struct RegistryEventListener {
    ibc_handler: Arc<IBCHandler>,
    economics: Arc<TokenEconomics>,
}

impl RegistryEventListener {
    /// Listen for LLM registration events from KNIRVCHAIN
    pub async fn on_llm_registered(&self, event: LLMRegistrationEvent) {
        // Collect registration fee
        // Update economics metrics
        // No local registry update
    }

    /// Listen for skill invocation events from KNIRVCHAIN
    pub async fn on_skill_invoked(&self, event: SkillInvocationEvent) {
        // Process invocation fee
        // Distribute rewards
        // Update metrics
    }
}
```

#### Phase 3: Remove Code
```rust
// Remove from nrn_token.rs:
// - LLMRegistry struct (lines 54-393)
// - SkillRegistry struct (lines 395-472)
// - Associated methods

// Remove from blockchain_adapter.rs:
// - register_llm_native()
// - register_skill_native()
// - Registry-related XION calls

// Remove from smart_contracts.rs:
// - LLM registry contract calls
// - Skill registry contract calls
```

---

## File Structure

```
KNIRVORACLE/src/
├── main.rs                    # Entry point (updated)
├── lib.rs                     # Module exports (updated)
├── governance.rs              # Enhanced governance system
├── token_economics.rs         # Economics engine (keep)
├── cross_chain/
│   ├── mod.rs
│   ├── router.rs              # Cross-chain router (new)
│   ├── transfer.rs            # Transfer types and processing (new)
│   ├── bridge.rs              # Bridge configurations (new)
│   └── proof.rs               # Transfer proof validation (new)
├── ibc_handler.rs             # Enhanced with actual transmission
├── registry_listener.rs       # Event listener for KNIRVCHAIN (new)
├── nrn_token.rs               # Token only (registries removed)
├── blockchain_adapter.rs      # XION bridge only (registries removed)
├── tendermint_consensus.rs    # Consensus (keep)
├── lora_skill_distributor.rs  # LoRA distribution (keep)
├── model_registry.rs          # Model governance (keep)
└── deprecated/
    ├── llm_registry.rs        # Archived for reference
    └── skill_registry.rs      # Archived for reference
```

---

## Implementation Phases

### Phase 1: Cross-Chain Infrastructure (Week 1-2)
- Implement CrossChainRouter
- Complete IBC packet transmission
- Add transfer proof validation
- Create bridge configurations

### Phase 2: Governance Consolidation (Week 3)
- Extend proposal types
- Add cross-chain governance execution
- Implement network parameter management
- Add emergency halt functionality

### Phase 3: Registry Deprecation (Week 4)
- Mark registries deprecated
- Implement registry event listeners
- Update economics to use IBC events
- Remove registry code

### Phase 4: Economics Enhancement (Week 5)
- Add cross-chain fee collection
- Implement cross-chain reward distribution
- Update metrics for multi-chain
- Performance optimization

### Phase 5: Testing & Integration (Week 6)
- End-to-end transfer testing
- Governance proposal testing
- Economics flow testing
- Cross-chain stress testing

---

## API Summary

### Cross-Chain Transfers
```
POST /v3/transfer/initiate               - Start cross-chain transfer
GET  /v3/transfer/{id}                   - Get transfer status
GET  /v3/transfer/history/{address}      - Get transfer history
GET  /v3/bridge/config                   - Get bridge configurations
GET  /v3/bridge/status                   - Get bridge health status
```

### Governance
```
POST /v3/governance/propose              - Submit proposal
POST /v3/governance/vote                 - Cast vote
GET  /v3/governance/proposals            - List proposals
POST /v3/governance/execute/{id}         - Execute proposal
GET  /v3/governance/parameters           - Get network params
```

### Economics
```
GET  /v3/economics/metrics               - Network metrics
GET  /v3/economics/fees                  - Fee structure
POST /v3/economics/stake                 - Stake tokens
GET  /v3/economics/staking/{address}     - Staking info
```

### Health
```
GET  /health                             - Basic health check
GET  /v3/status                          - Full status
GET  /v3/ibc/connections                 - IBC connection status
```
