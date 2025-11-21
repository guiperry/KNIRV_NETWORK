use anyhow::{anyhow, Result};
use serde::{Deserialize, Serialize};
use std::collections::HashMap;
use std::sync::Arc;
use std::time::{SystemTime, UNIX_EPOCH};
use tokio::sync::Mutex;
use tracing::{info, warn};

use crate::multi_model_engine::{ModelPerformanceMetrics, ModelType};
use crate::nrn_token::Address;
use crate::cross_chain::router::CrossChainRouter;
use crate::token_economics::TokenEconomics;

#[derive(Debug)]
#[allow(dead_code)]
pub struct GovernanceSystem {
    validators: Arc<Mutex<Vec<Validator>>>,
    proposals: Arc<Mutex<HashMap<String, Proposal>>>,
    votes: Arc<Mutex<HashMap<String, Vec<Vote>>>>,
    governance_config: GovernanceConfig,
}

/// Enhanced governance system with full network authority
#[derive(Debug)]
pub struct NetworkGovernance {
    pub governance: Arc<GovernanceSystem>,
    pub cross_chain_router: Arc<CrossChainRouter>,
    pub economics: Arc<TokenEconomics>,

    // Network-wide governance scope
    pub network_parameters: NetworkParameters,
    pub chain_registrations: HashMap<crate::cross_chain::transfer::ChainId, ChainRegistration>,
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

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ChainRegistration {
    pub chain_id: crate::cross_chain::transfer::ChainId,
    pub registered_at: u64,
    pub governance_threshold: f64,
    pub active: bool,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Validator {
    pub address: Address,
    pub voting_power: u64,
    pub stake: num_bigint::BigInt,
    pub reputation_score: f64,
    pub active: bool,
    pub joined_at: u64,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Proposal {
    pub proposal_id: String,
    pub proposal_type: ProposalType,
    pub title: String,
    pub description: String,
    pub proposer: Address,
    pub created_at: u64,
    pub voting_deadline: u64,
    pub execution_deadline: u64,
    pub status: ProposalStatus,
    pub required_threshold: f64,
    pub current_support: f64,
    pub total_votes: u64,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum ProposalType {
    // Existing types
    ModelTransition(ModelTransitionData),
    NetworkParameter(NetworkParameterData),
    ValidatorUpdate(ValidatorUpdateData),
    Emergency(EmergencyData),

    // New cross-chain governance
    ChainRegistration(ChainRegistrationData),
    ChainDeregistration(crate::cross_chain::transfer::ChainId),
    BridgeParameterUpdate(BridgeParameterChange),
    CrossChainEmergencyHalt(Vec<crate::cross_chain::transfer::ChainId>),

    // Economic governance
    TokenomicsUpdate(TokenomicsChange),
    FeeStructureUpdate(FeeStructureChange),
    RewardDistributionUpdate(RewardDistributionChange),

    // Protocol upgrades
    ProtocolUpgrade(ProtocolUpgradeData),
    ChainFork(ChainForkData),
}

// New cross-chain governance data structures
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ChainRegistrationData {
    pub chain_id: crate::cross_chain::transfer::ChainId,
    pub chain_metadata: ChainMetadata,
    pub bridge_config: BridgeConfigData,
    pub governance_threshold: f64,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ChainMetadata {
    pub name: String,
    pub description: String,
    pub native_token: String,
    pub consensus_mechanism: String,
    pub block_time_seconds: u64,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct BridgeConfigData {
    pub channel_id: String,
    pub port_id: String,
    pub connection_id: String,
    pub client_id: String,
    pub trust_level: f64,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct BridgeParameterChange {
    pub chain_id: crate::cross_chain::transfer::ChainId,
    pub parameter: String,
    pub current_value: serde_json::Value,
    pub proposed_value: serde_json::Value,
}

// Economic governance data structures
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TokenomicsChange {
    pub parameter: String,
    pub current_value: serde_json::Value,
    pub proposed_value: serde_json::Value,
    pub economic_impact: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct FeeStructureChange {
    pub fee_type: String,
    pub current_basis_points: u16,
    pub proposed_basis_points: u16,
    pub impact_assessment: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RewardDistributionChange {
    pub reward_type: String,
    pub current_percentage: f64,
    pub proposed_percentage: f64,
    pub justification: String,
}

// Protocol upgrade data structures
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ProtocolUpgradeData {
    pub upgrade_name: String,
    pub version: String,
    pub affected_chains: Vec<crate::cross_chain::transfer::ChainId>,
    pub upgrade_plan: String,
    pub rollback_plan: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ChainForkData {
    pub chain_id: crate::cross_chain::transfer::ChainId,
    pub fork_height: u64,
    pub fork_hash: String,
    pub justification: String,
    pub migration_plan: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ModelTransitionData {
    pub current_model: Option<String>,
    pub proposed_model: String,
    pub model_type: ModelType,
    pub performance_comparison: ModelPerformanceComparison,
    pub migration_plan: MigrationPlan,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ModelPerformanceComparison {
    pub current_metrics: Option<ModelPerformanceMetrics>,
    pub proposed_metrics: ModelPerformanceMetrics,
    pub improvement_percentage: f64,
    pub risk_assessment: RiskAssessment,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RiskAssessment {
    pub technical_risk: RiskLevel,
    pub performance_risk: RiskLevel,
    pub security_risk: RiskLevel,
    pub overall_risk: RiskLevel,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum RiskLevel {
    Low,
    Medium,
    High,
    Critical,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MigrationPlan {
    pub estimated_downtime: u64,
    pub rollback_plan: bool,
    pub testing_completed: bool,
    pub validator_consensus_required: bool,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct NetworkParameterData {
    pub parameter_name: String,
    pub current_value: serde_json::Value,
    pub proposed_value: serde_json::Value,
    pub impact_assessment: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ValidatorUpdateData {
    pub action: ValidatorAction,
    pub validator_address: Address,
    pub new_voting_power: Option<u64>,
    pub justification: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum ValidatorAction {
    Add,
    Remove,
    UpdatePower,
    Suspend,
    Reactivate,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct EmergencyData {
    pub emergency_type: EmergencyType,
    pub immediate_action: String,
    pub justification: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum EmergencyType {
    SecurityBreach,
    ModelFailure,
    NetworkHalt,
    CriticalBug,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum ProposalStatus {
    Active,
    Passed,
    Rejected,
    Executed,
    Expired,
    Cancelled,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Vote {
    pub voter: Address,
    pub proposal_id: String,
    pub vote_type: VoteType,
    pub voting_power: u64,
    pub timestamp: u64,
    pub justification: Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum VoteType {
    Yes,
    No,
    Abstain,
    NoWithVeto,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GovernanceConfig {
    pub min_voting_period: u64, // seconds
    pub max_voting_period: u64, // seconds
    pub min_deposit: num_bigint::BigInt,
    pub quorum_threshold: f64,    // percentage
    pub pass_threshold: f64,      // percentage
    pub veto_threshold: f64,      // percentage
    pub emergency_threshold: f64, // percentage for emergency proposals
    pub validator_bond: num_bigint::BigInt,
}

impl Default for GovernanceConfig {
    fn default() -> Self {
        Self {
            min_voting_period: 24 * 3600,                    // 1 day
            max_voting_period: 7 * 24 * 3600,                // 7 days
            min_deposit: num_bigint::BigInt::from(1000),     // 1000 NRN
            quorum_threshold: 0.33,                          // 33%
            pass_threshold: 0.50,                            // 50%
            veto_threshold: 0.33,                            // 33%
            emergency_threshold: 0.67,                       // 67%
            validator_bond: num_bigint::BigInt::from(10000), // 10000 NRN
        }
    }
}

#[allow(dead_code)]
impl GovernanceSystem {
    pub fn new(config: Option<GovernanceConfig>) -> Self {
        Self {
            validators: Arc::new(Mutex::new(Vec::new())),
            proposals: Arc::new(Mutex::new(HashMap::new())),
            votes: Arc::new(Mutex::new(HashMap::new())),
            governance_config: config.unwrap_or_default(),
        }
    }

    /// Add a validator to the governance system
    pub async fn add_validator(&self, validator: Validator) -> Result<()> {
        let mut validators = self.validators.lock().await;

        // Check if validator already exists
        if validators.iter().any(|v| v.address == validator.address) {
            return Err(anyhow!("Validator already exists"));
        }

        validators.push(validator.clone());
        info!("Added validator: {}", validator.address);
        Ok(())
    }

    /// Submit a new proposal
    pub async fn submit_proposal(&self, proposal: Proposal) -> Result<String> {
        let proposal_id = proposal.proposal_id.clone();

        // Validate proposal
        self.validate_proposal(&proposal).await?;

        // Store proposal
        let mut proposals = self.proposals.lock().await;
        proposals.insert(proposal_id.clone(), proposal);

        // Initialize vote tracking
        let mut votes = self.votes.lock().await;
        votes.insert(proposal_id.clone(), Vec::new());

        info!("Submitted proposal: {}", proposal_id);
        Ok(proposal_id)
    }

    /// Cast a vote on a proposal
    pub async fn cast_vote(&self, vote: Vote) -> Result<()> {
        // Validate voter is a validator
        let validators = self.validators.lock().await;
        let validator = validators
            .iter()
            .find(|v| v.address == vote.voter && v.active)
            .ok_or_else(|| anyhow!("Invalid or inactive validator"))?;

        // Check if proposal exists and is active
        let proposals = self.proposals.lock().await;
        let proposal = proposals
            .get(&vote.proposal_id)
            .ok_or_else(|| anyhow!("Proposal not found"))?;

        if !matches!(proposal.status, ProposalStatus::Active) {
            return Err(anyhow!("Proposal is not active"));
        }

        // Check voting deadline
        let current_time = SystemTime::now().duration_since(UNIX_EPOCH)?.as_secs();
        if current_time > proposal.voting_deadline {
            return Err(anyhow!("Voting period has ended"));
        }

        // Store vote
        let mut votes = self.votes.lock().await;
        let proposal_votes = votes
            .get_mut(&vote.proposal_id)
            .ok_or_else(|| anyhow!("Vote tracking not found"))?;

        // Check if validator already voted
        if proposal_votes.iter().any(|v| v.voter == vote.voter) {
            return Err(anyhow!("Validator has already voted"));
        }

        let proposal_id = vote.proposal_id.clone();
        let mut vote_with_power = vote;
        vote_with_power.voting_power = validator.voting_power;
        proposal_votes.push(vote_with_power);

        info!(
            "Vote cast by {} on proposal {}",
            validator.address, proposal_id
        );
        Ok(())
    }

    /// Check if a model switch is approved by governance
    pub async fn is_model_switch_approved(&self, model_type: &ModelType) -> Result<bool> {
        let proposals = self.proposals.lock().await;

        for proposal in proposals.values() {
            if let ProposalType::ModelTransition(data) = &proposal.proposal_type {
                if data.model_type == *model_type
                    && matches!(
                        proposal.status,
                        ProposalStatus::Passed | ProposalStatus::Executed
                    )
                {
                    return Ok(true);
                }
            }
        }

        Ok(false)
    }

    /// Tally votes and update proposal status
    pub async fn tally_votes(&self, proposal_id: &str) -> Result<ProposalStatus> {
        let mut proposals = self.proposals.lock().await;
        let proposal = proposals
            .get_mut(proposal_id)
            .ok_or_else(|| anyhow!("Proposal not found"))?;

        let votes = self.votes.lock().await;
        let proposal_votes = votes
            .get(proposal_id)
            .ok_or_else(|| anyhow!("Votes not found"))?;

        let validators = self.validators.lock().await;
        let total_voting_power: u64 = validators
            .iter()
            .filter(|v| v.active)
            .map(|v| v.voting_power)
            .sum();

        let mut yes_power = 0u64;
        let mut no_power = 0u64;
        let mut abstain_power = 0u64;
        let mut veto_power = 0u64;

        for vote in proposal_votes {
            match vote.vote_type {
                VoteType::Yes => yes_power += vote.voting_power,
                VoteType::No => no_power += vote.voting_power,
                VoteType::Abstain => abstain_power += vote.voting_power,
                VoteType::NoWithVeto => veto_power += vote.voting_power,
            }
        }

        let total_voted_power = yes_power + no_power + abstain_power + veto_power;
        let participation_rate = total_voted_power as f64 / total_voting_power as f64;

        // Check quorum
        if participation_rate < self.governance_config.quorum_threshold {
            proposal.status = ProposalStatus::Rejected;
            return Ok(ProposalStatus::Rejected);
        }

        // Check veto
        let veto_rate = veto_power as f64 / total_voted_power as f64;
        if veto_rate >= self.governance_config.veto_threshold {
            proposal.status = ProposalStatus::Rejected;
            return Ok(ProposalStatus::Rejected);
        }

        // Check pass threshold
        let yes_rate = yes_power as f64 / total_voted_power as f64;
        let required_threshold = match &proposal.proposal_type {
            ProposalType::Emergency(_) | ProposalType::CrossChainEmergencyHalt(_) => self.governance_config.emergency_threshold,
            _ => self.governance_config.pass_threshold,
        };

        if yes_rate >= required_threshold {
            proposal.status = ProposalStatus::Passed;
            proposal.current_support = yes_rate;
            info!(
                "Proposal {} passed with {:.2}% support",
                proposal_id,
                yes_rate * 100.0
            );
            Ok(ProposalStatus::Passed)
        } else {
            proposal.status = ProposalStatus::Rejected;
            Ok(ProposalStatus::Rejected)
        }
    }

    /// Execute an approved proposal
    pub async fn execute_proposal(&self, proposal_id: &str) -> Result<()> {
        let mut proposals = self.proposals.lock().await;
        let proposal = proposals
            .get_mut(proposal_id)
            .ok_or_else(|| anyhow!("Proposal not found"))?;

        if !matches!(proposal.status, ProposalStatus::Passed) {
            return Err(anyhow!("Proposal not approved for execution"));
        }

        let current_time = SystemTime::now().duration_since(UNIX_EPOCH)?.as_secs();
        if current_time > proposal.execution_deadline {
            proposal.status = ProposalStatus::Expired;
            return Err(anyhow!("Proposal execution deadline has passed"));
        }

        // Execute based on proposal type
        match &proposal.proposal_type {
            ProposalType::ModelTransition(data) => {
                info!(
                    "Executing model transition: {} -> {}",
                    data.current_model.as_ref().unwrap_or(&"none".to_string()),
                    data.proposed_model
                );
                // Model transition execution is handled by the model registry
            }
            ProposalType::NetworkParameter(data) => {
                info!(
                    "Executing network parameter change: {} = {:?}",
                    data.parameter_name, data.proposed_value
                );
                // Network parameter changes would be handled by the respective modules
            }
            ProposalType::ValidatorUpdate(data) => {
                info!(
                    "Executing validator update: {:?} for {}",
                    data.action, data.validator_address
                );
                self.execute_validator_update(data).await?;
            }
            ProposalType::Emergency(data) => {
                warn!("Executing emergency proposal: {:?}", data.emergency_type);
                // Emergency actions would be handled by the respective modules
            }
            ProposalType::ChainRegistration(data) => {
                info!("Executing chain registration for: {:?}", data.chain_id);
                // Chain registration would be handled by cross-chain module
            }
            ProposalType::ChainDeregistration(chain_id) => {
                info!("Executing chain deregistration for: {:?}", chain_id);
                // Chain deregistration would be handled by cross-chain module
            }
            ProposalType::BridgeParameterUpdate(data) => {
                info!("Executing bridge parameter update for: {:?}", data.chain_id);
                // Bridge parameter updates would be handled by cross-chain module
            }
            ProposalType::CrossChainEmergencyHalt(chains) => {
                warn!("Executing cross-chain emergency halt for chains: {:?}", chains);
                // Emergency halt would be handled by cross-chain module
            }
            ProposalType::TokenomicsUpdate(data) => {
                info!("Executing tokenomics update: {}", data.parameter);
                // Tokenomics updates would be handled by economics module
            }
            ProposalType::FeeStructureUpdate(data) => {
                info!("Executing fee structure update: {}", data.fee_type);
                // Fee structure updates would be handled by economics module
            }
            ProposalType::RewardDistributionUpdate(data) => {
                info!("Executing reward distribution update: {}", data.reward_type);
                // Reward distribution updates would be handled by economics module
            }
            ProposalType::ProtocolUpgrade(data) => {
                info!("Executing protocol upgrade: {}", data.upgrade_name);
                // Protocol upgrades would be handled by respective modules
            }
            ProposalType::ChainFork(data) => {
                warn!("Executing chain fork for: {:?}", data.chain_id);
                // Chain forks would be handled by cross-chain module
            }
        }

        proposal.status = ProposalStatus::Executed;
        info!("Executed proposal: {}", proposal_id);
        Ok(())
    }

    /// Execute validator update
    async fn execute_validator_update(&self, data: &ValidatorUpdateData) -> Result<()> {
        let mut validators = self.validators.lock().await;

        match data.action {
            ValidatorAction::Add => {
                if validators
                    .iter()
                    .any(|v| v.address == data.validator_address)
                {
                    return Err(anyhow!("Validator already exists"));
                }
                // Add new validator logic would go here
            }
            ValidatorAction::Remove => {
                validators.retain(|v| v.address != data.validator_address);
            }
            ValidatorAction::UpdatePower => {
                if let Some(validator) = validators
                    .iter_mut()
                    .find(|v| v.address == data.validator_address)
                {
                    if let Some(new_power) = data.new_voting_power {
                        validator.voting_power = new_power;
                    }
                }
            }
            ValidatorAction::Suspend => {
                if let Some(validator) = validators
                    .iter_mut()
                    .find(|v| v.address == data.validator_address)
                {
                    validator.active = false;
                }
            }
            ValidatorAction::Reactivate => {
                if let Some(validator) = validators
                    .iter_mut()
                    .find(|v| v.address == data.validator_address)
                {
                    validator.active = true;
                }
            }
        }

        Ok(())
    }

    /// Validate a proposal before submission
    async fn validate_proposal(&self, proposal: &Proposal) -> Result<()> {
        let current_time = SystemTime::now().duration_since(UNIX_EPOCH)?.as_secs();

        // Check timing
        if proposal.voting_deadline <= current_time {
            return Err(anyhow!("Voting deadline must be in the future"));
        }

        if proposal.voting_deadline - current_time < self.governance_config.min_voting_period {
            return Err(anyhow!("Voting period too short"));
        }

        if proposal.voting_deadline - current_time > self.governance_config.max_voting_period {
            return Err(anyhow!("Voting period too long"));
        }

        // Validate proposal type specific requirements
        match &proposal.proposal_type {
            ProposalType::ModelTransition(data) => {
                if data.proposed_model.is_empty() {
                    return Err(anyhow!("Proposed model hash cannot be empty"));
                }
            }
            ProposalType::NetworkParameter(data) => {
                if data.parameter_name.is_empty() {
                    return Err(anyhow!("Parameter name cannot be empty"));
                }
            }
            ProposalType::ValidatorUpdate(_) => {
                // Validator update specific validation
            }
            ProposalType::Emergency(_) => {
                // Emergency proposals have relaxed validation
            }
            ProposalType::ChainRegistration(data) => {
                if data.chain_metadata.name.is_empty() {
                    return Err(anyhow!("Chain name cannot be empty"));
                }
            }
            ProposalType::ChainDeregistration(_) => {
                // Chain deregistration validation
            }
            ProposalType::BridgeParameterUpdate(data) => {
                if data.parameter.is_empty() {
                    return Err(anyhow!("Parameter name cannot be empty"));
                }
            }
            ProposalType::CrossChainEmergencyHalt(chains) => {
                if chains.is_empty() {
                    return Err(anyhow!("Must specify at least one chain for emergency halt"));
                }
            }
            ProposalType::TokenomicsUpdate(data) => {
                if data.parameter.is_empty() {
                    return Err(anyhow!("Parameter name cannot be empty"));
                }
            }
            ProposalType::FeeStructureUpdate(data) => {
                if data.fee_type.is_empty() {
                    return Err(anyhow!("Fee type cannot be empty"));
                }
            }
            ProposalType::RewardDistributionUpdate(data) => {
                if data.reward_type.is_empty() {
                    return Err(anyhow!("Reward type cannot be empty"));
                }
            }
            ProposalType::ProtocolUpgrade(data) => {
                if data.upgrade_name.is_empty() {
                    return Err(anyhow!("Upgrade name cannot be empty"));
                }
                if data.affected_chains.is_empty() {
                    return Err(anyhow!("Must specify affected chains"));
                }
            }
            ProposalType::ChainFork(data) => {
                if data.chain_id.to_string().is_empty() {
                    return Err(anyhow!("Chain ID cannot be empty"));
                }
            }
        }

        Ok(())
    }

    /// Get all active proposals
    pub async fn get_active_proposals(&self) -> Vec<Proposal> {
        let proposals = self.proposals.lock().await;
        proposals
            .values()
            .filter(|p| matches!(p.status, ProposalStatus::Active))
            .cloned()
            .collect()
    }

    /// Get proposal by ID
    pub async fn get_proposal(&self, proposal_id: &str) -> Option<Proposal> {
        let proposals = self.proposals.lock().await;
        proposals.get(proposal_id).cloned()
    }

    /// Get votes for a proposal
    pub async fn get_proposal_votes(&self, proposal_id: &str) -> Option<Vec<Vote>> {
        let votes = self.votes.lock().await;
        votes.get(proposal_id).cloned()
    }

    /// Get all validators
    pub async fn get_validators(&self) -> Vec<Validator> {
        let validators = self.validators.lock().await;
        validators.clone()
    }
}

impl NetworkGovernance {
    /// Create a new network governance system
    pub fn new(
        governance: Arc<GovernanceSystem>,
        cross_chain_router: Arc<CrossChainRouter>,
        economics: Arc<TokenEconomics>,
    ) -> Self {
        Self {
            governance,
            cross_chain_router,
            economics,
            network_parameters: NetworkParameters::default(),
            chain_registrations: HashMap::new(),
            emergency_council: vec![],
        }
    }

    /// Submit a network-wide proposal
    pub async fn submit_network_proposal(&self, proposal: Proposal) -> Result<String, anyhow::Error> {
        // Validate network-wide proposal requirements
        self.validate_network_proposal(&proposal).await?;

        // Submit to governance system
        self.governance.submit_proposal(proposal).await
    }

    /// Execute network parameter changes
    pub async fn execute_network_parameter_change(&self, parameter: &str, value: serde_json::Value) -> Result<(), anyhow::Error> {
        match parameter {
            "base_transaction_fee" => {
                if let Some(fee) = value.as_u64() {
                    // Update network parameters
                    // This would trigger cross-chain updates
                    info!("Updated base transaction fee to {}", fee);
                }
            }
            "max_transfer_amount" => {
                if let Some(amount) = value.as_u64() {
                    // Update cross-chain limits
                    info!("Updated max transfer amount to {}", amount);
                }
            }
            _ => {
                warn!("Unknown network parameter: {}", parameter);
            }
        }
        Ok(())
    }

    /// Register a new chain in the network
    pub async fn register_chain(&mut self, chain_id: crate::cross_chain::transfer::ChainId, governance_threshold: f64) -> Result<(), anyhow::Error> {
        let registration = ChainRegistration {
            chain_id: chain_id.clone(),
            registered_at: std::time::SystemTime::now().duration_since(std::time::UNIX_EPOCH)?.as_secs(),
            governance_threshold,
            active: true,
        };

        self.chain_registrations.insert(chain_id, registration);
        info!("Registered new chain in network governance");
        Ok(())
    }

    /// Emergency halt for specified chains
    pub async fn emergency_halt_chains(&self, chains: &[crate::cross_chain::transfer::ChainId]) -> Result<(), anyhow::Error> {
        warn!("Executing emergency halt for chains: {:?}", chains);
        // This would trigger emergency protocols across chains
        // Implementation would involve cross-chain governance messages
        Ok(())
    }

    /// Get current network parameters
    pub fn get_network_parameters(&self) -> &NetworkParameters {
        &self.network_parameters
    }

    /// Update network parameters
    pub fn update_network_parameters(&mut self, params: NetworkParameters) {
        self.network_parameters = params;
    }

    /// Validate network-wide proposal
    async fn validate_network_proposal(&self, proposal: &Proposal) -> Result<(), anyhow::Error> {
        match &proposal.proposal_type {
            ProposalType::ChainRegistration(data) => {
                // Check if chain is already registered
                if self.chain_registrations.contains_key(&data.chain_id) {
                    return Err(anyhow::anyhow!("Chain already registered"));
                }
            }
            ProposalType::ChainDeregistration(chain_id) => {
                // Check if chain exists and is registered
                if !self.chain_registrations.contains_key(chain_id) {
                    return Err(anyhow::anyhow!("Chain not registered"));
                }
            }
            _ => {
                // Other validations can be added here
            }
        }
        Ok(())
    }
}

impl Default for NetworkParameters {
    fn default() -> Self {
        Self {
            // Economic parameters
            base_transaction_fee: 1000,        // 0.001 NRN
            skill_invocation_fee: 100_000,      // 0.1 NRN
            llm_registration_fee: 1_000_000,    // 1 NRN
            validation_reward: 50_000,          // 0.05 NRN

            // Governance parameters
            proposal_deposit: 100_000_000,      // 100 NRN
            voting_period_blocks: 60480,        // ~7 days at 1 block/sec
            quorum_threshold: 0.33,
            pass_threshold: 0.50,
            veto_threshold: 0.33,

            // Cross-chain parameters
            max_transfer_amount: 1_000_000_000_000, // 1M NRN
            transfer_timeout_blocks: 100,
            bridge_fee_basis_points: 30,        // 0.3%

            // Staking parameters
            min_validator_stake: 100_000_000_000, // 100K NRN
            min_developer_stake: 10_000_000_000,  // 10K NRN
            unbonding_period_days: 14,
            slashing_penalty_percent: 5,
        }
    }
}
