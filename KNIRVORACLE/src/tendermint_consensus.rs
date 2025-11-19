use anyhow::{anyhow, Result};
use serde::{Deserialize, Serialize};
use sha2::Digest;
use std::collections::HashMap;
use std::sync::Arc;
use std::time::{SystemTime, UNIX_EPOCH};
use tokio::sync::Mutex;
use tracing::{info, warn};

use crate::nrn_token::{Address, Transaction};

#[derive(Debug)]
#[allow(dead_code)]
pub struct TendermintConsensus {
    validators: Arc<Mutex<ValidatorSet>>,
    proposer: Arc<Mutex<BlockProposer>>,
    voter: Arc<Mutex<BlockVoter>>,
    chain_state: Arc<Mutex<ChainState>>,
    consensus_config: ConsensusConfig,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ValidatorSet {
    validators: HashMap<Address, ValidatorInfo>,
    total_voting_power: u64,
    proposer_priority: HashMap<Address, i64>,
    current_proposer: Option<Address>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ValidatorInfo {
    pub address: Address,
    pub voting_power: u64,
    pub pub_key: String,
    pub active: bool,
    pub last_commit_height: u64,
    pub missed_blocks: u64,
    pub reputation_score: f64,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct BlockProposer {
    current_height: u64,
    current_round: u32,
    proposal_timeout: u64,
    last_proposal_time: u64,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct BlockVoter {
    votes: HashMap<u64, HashMap<Address, Vote>>, // height -> validator -> vote
    precommits: HashMap<u64, HashMap<Address, Precommit>>,
    voting_timeout: u64,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Vote {
    pub height: u64,
    pub round: u32,
    pub vote_type: VoteType,
    pub block_hash: Option<String>,
    pub validator: Address,
    pub timestamp: u64,
    pub signature: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum VoteType {
    Prevote,
    Precommit,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Precommit {
    pub height: u64,
    pub round: u32,
    pub block_hash: Option<String>,
    pub validator: Address,
    pub timestamp: u64,
    pub signature: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ChainState {
    pub height: u64,
    pub last_block_hash: String,
    pub last_commit_hash: String,
    pub validators_hash: String,
    pub app_hash: String,
    pub last_block_time: u64,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ConsensusConfig {
    pub block_time: u64,        // Target block time in seconds
    pub timeout_propose: u64,   // Proposal timeout in milliseconds
    pub timeout_prevote: u64,   // Prevote timeout in milliseconds
    pub timeout_precommit: u64, // Precommit timeout in milliseconds
    pub timeout_commit: u64,    // Commit timeout in milliseconds
    pub max_block_size: usize,  // Maximum block size in bytes
    pub max_gas: u64,           // Maximum gas per block
    pub evidence_max_age: u64,  // Maximum age for evidence in blocks
}

impl Default for ConsensusConfig {
    fn default() -> Self {
        Self {
            block_time: 3,               // 3 seconds
            timeout_propose: 3000,       // 3 seconds
            timeout_prevote: 1000,       // 1 second
            timeout_precommit: 1000,     // 1 second
            timeout_commit: 1000,        // 1 second
            max_block_size: 1024 * 1024, // 1MB
            max_gas: 1_000_000,          // 1M gas
            evidence_max_age: 100_000,   // 100k blocks
        }
    }
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TendermintBlock {
    pub header: BlockHeader,
    pub data: BlockData,
    pub evidence: Evidence,
    pub last_commit: Option<Commit>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct BlockHeader {
    pub version: String,
    pub chain_id: String,
    pub height: u64,
    pub time: u64,
    pub last_block_id: Option<BlockId>,
    pub last_commit_hash: String,
    pub data_hash: String,
    pub validators_hash: String,
    pub next_validators_hash: String,
    pub consensus_hash: String,
    pub app_hash: String,
    pub last_results_hash: String,
    pub evidence_hash: String,
    pub proposer_address: Address,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct BlockId {
    pub hash: String,
    pub parts: PartSetHeader,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PartSetHeader {
    pub total: u32,
    pub hash: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct BlockData {
    pub txs: Vec<Transaction>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Evidence {
    pub evidence: Vec<EvidenceItem>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct EvidenceItem {
    pub evidence_type: String,
    pub validator: Address,
    pub height: u64,
    pub time: u64,
    pub total_voting_power: u64,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Commit {
    pub height: u64,
    pub round: u32,
    pub block_id: BlockId,
    pub signatures: Vec<CommitSig>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CommitSig {
    pub block_id_flag: u8,
    pub validator_address: Address,
    pub timestamp: u64,
    pub signature: String,
}

#[allow(dead_code)]
impl TendermintConsensus {
    pub fn new(chain_id: String, config: Option<ConsensusConfig>) -> Self {
        let consensus_config = config.unwrap_or_default();

        Self {
            validators: Arc::new(Mutex::new(ValidatorSet::new())),
            proposer: Arc::new(Mutex::new(BlockProposer::new())),
            voter: Arc::new(Mutex::new(BlockVoter::new(consensus_config.clone()))),
            chain_state: Arc::new(Mutex::new(ChainState::new(chain_id))),
            consensus_config,
        }
    }

    /// Add a validator to the set
    pub async fn add_validator(&self, validator: ValidatorInfo) -> Result<()> {
        let mut validators = self.validators.lock().await;
        validators.add_validator(validator).await
    }

    /// Remove a validator from the set
    pub async fn remove_validator(&self, address: &Address) -> Result<()> {
        let mut validators = self.validators.lock().await;
        validators.remove_validator(address).await
    }

    /// Propose a new block
    pub async fn propose_block(&self, transactions: Vec<Transaction>) -> Result<TendermintBlock> {
        let mut proposer = self.proposer.lock().await;
        let validators = self.validators.lock().await;
        let chain_state = self.chain_state.lock().await;

        // Check if we are the proposer
        let current_proposer = validators
            .get_current_proposer()
            .ok_or_else(|| anyhow!("No current proposer"))?;

        // Create block proposal
        let block = proposer
            .create_block_proposal(
                &chain_state,
                &validators,
                transactions,
                &self.consensus_config,
            )
            .await?;

        info!(
            "Proposed block at height {} by {}",
            block.header.height, current_proposer
        );
        Ok(block)
    }

    /// Vote on a proposed block
    pub async fn vote_on_block(
        &self,
        block: &TendermintBlock,
        validator_address: &Address,
        vote_type: VoteType,
    ) -> Result<Vote> {
        let mut voter = self.voter.lock().await;
        let _validators = self.validators.lock().await;

        // Validate the block
        self.validate_block(block).await?;

        // Create vote
        let vote = voter.create_vote(
            block.header.height,
            0, // round
            vote_type,
            Some(self.calculate_block_hash(block)?),
            validator_address.clone(),
        )?;

        info!(
            "Vote cast by {} for block at height {}",
            validator_address, block.header.height
        );
        Ok(vote)
    }

    /// Commit a block after consensus
    pub async fn commit_block(&self, block: TendermintBlock) -> Result<()> {
        let mut chain_state = self.chain_state.lock().await;

        // Validate consensus
        self.validate_consensus(&block).await?;

        // Update chain state
        chain_state.height = block.header.height;
        chain_state.last_block_hash = self.calculate_block_hash(&block)?;
        chain_state.app_hash = block.header.app_hash.clone();
        chain_state.last_block_time = block.header.time;

        // Update validator set if needed
        self.update_validator_set_after_commit(&block).await?;

        info!("Committed block at height {}", block.header.height);
        Ok(())
    }

    /// Validate a proposed block
    async fn validate_block(&self, block: &TendermintBlock) -> Result<()> {
        let chain_state = self.chain_state.lock().await;

        // Check height
        if block.header.height != chain_state.height + 1 {
            return Err(anyhow!("Invalid block height"));
        }

        // Check previous block hash
        if let Some(last_block_id) = &block.header.last_block_id {
            if last_block_id.hash != chain_state.last_block_hash {
                return Err(anyhow!("Invalid previous block hash"));
            }
        }

        // Check block size
        let block_size = serde_json::to_vec(block)?.len();
        if block_size > self.consensus_config.max_block_size {
            return Err(anyhow!("Block size exceeds maximum"));
        }

        // Validate transactions
        for tx in &block.data.txs {
            self.validate_transaction(tx).await?;
        }

        Ok(())
    }

    /// Validate consensus for a block
    async fn validate_consensus(&self, block: &TendermintBlock) -> Result<()> {
        let validators = self.validators.lock().await;

        if let Some(commit) = &block.last_commit {
            let mut total_voting_power = 0u64;
            let mut committed_power = 0u64;

            for sig in &commit.signatures {
                if let Some(validator) = validators.get_validator(&sig.validator_address) {
                    total_voting_power += validator.voting_power;

                    // Verify signature (simplified)
                    if !sig.signature.is_empty() {
                        committed_power += validator.voting_power;
                    }
                }
            }

            // Check if we have more than 2/3 voting power
            if committed_power * 3 <= total_voting_power * 2 {
                return Err(anyhow!("Insufficient voting power for consensus"));
            }
        }

        Ok(())
    }

    /// Validate a transaction
    async fn validate_transaction(&self, _tx: &Transaction) -> Result<()> {
        // TODO: Implement transaction validation
        // This would include signature verification, balance checks, etc.
        Ok(())
    }

    /// Calculate block hash
    fn calculate_block_hash(&self, block: &TendermintBlock) -> Result<String> {
        let block_bytes = serde_json::to_vec(block)?;
        let hash = sha2::Sha256::digest(&block_bytes);
        Ok(hex::encode(hash))
    }

    /// Update validator set after committing a block
    async fn update_validator_set_after_commit(&self, _block: &TendermintBlock) -> Result<()> {
        let mut validators = self.validators.lock().await;

        // Update proposer priority and select next proposer
        validators.update_proposer_priority().await?;
        validators.select_next_proposer().await?;

        Ok(())
    }

    /// Get current chain state
    pub async fn get_chain_state(&self) -> ChainState {
        let chain_state = self.chain_state.lock().await;
        chain_state.clone()
    }

    /// Get validator set
    pub async fn get_validator_set(&self) -> ValidatorSet {
        let validators = self.validators.lock().await;
        validators.clone()
    }

    /// Check if consensus is healthy
    pub async fn health_check(&self) -> Result<bool> {
        let validators = self.validators.lock().await;
        let chain_state = self.chain_state.lock().await;

        // Check if we have enough validators
        if validators.validators.len() < 1 {
            return Ok(false);
        }

        // Check if chain is progressing
        let current_time = SystemTime::now().duration_since(UNIX_EPOCH)?.as_secs();
        if current_time - chain_state.last_block_time > self.consensus_config.block_time * 3 {
            warn!("Chain appears to be stalled");
            return Ok(false);
        }

        Ok(true)
    }
}

#[allow(dead_code)]
impl ValidatorSet {
    pub fn new() -> Self {
        Self {
            validators: HashMap::new(),
            total_voting_power: 0,
            proposer_priority: HashMap::new(),
            current_proposer: None,
        }
    }

    pub async fn add_validator(&mut self, validator: ValidatorInfo) -> Result<()> {
        if self.validators.contains_key(&validator.address) {
            return Err(anyhow!("Validator already exists"));
        }

        self.total_voting_power += validator.voting_power;
        self.proposer_priority.insert(validator.address.clone(), 0);
        self.validators.insert(validator.address.clone(), validator);

        // Select initial proposer if this is the first validator
        if self.current_proposer.is_none() {
            self.select_next_proposer().await?;
        }

        Ok(())
    }

    pub async fn remove_validator(&mut self, address: &Address) -> Result<()> {
        if let Some(validator) = self.validators.remove(address) {
            self.total_voting_power -= validator.voting_power;
            self.proposer_priority.remove(address);

            // Select new proposer if we removed the current one
            if self.current_proposer.as_ref() == Some(address) {
                self.select_next_proposer().await?;
            }
        }

        Ok(())
    }

    pub fn get_validator(&self, address: &Address) -> Option<&ValidatorInfo> {
        self.validators.get(address)
    }

    pub fn get_current_proposer(&self) -> Option<Address> {
        self.current_proposer.clone()
    }

    pub fn get_validator_count(&self) -> usize {
        self.validators.len()
    }

    pub async fn update_proposer_priority(&mut self) -> Result<()> {
        // Update proposer priorities (simplified algorithm)
        for (address, validator) in &self.validators {
            let priority = self.proposer_priority.get_mut(address).unwrap();
            *priority += validator.voting_power as i64;
        }

        // Decrease current proposer's priority
        if let Some(current) = &self.current_proposer {
            if let Some(priority) = self.proposer_priority.get_mut(current) {
                *priority -= self.total_voting_power as i64;
            }
        }

        Ok(())
    }

    pub async fn select_next_proposer(&mut self) -> Result<()> {
        // Select validator with highest priority
        let next_proposer = self
            .proposer_priority
            .iter()
            .filter(|(addr, _)| self.validators.get(addr).map_or(false, |v| v.active))
            .max_by_key(|(_, priority)| *priority)
            .map(|(addr, _)| addr.clone());

        self.current_proposer = next_proposer;
        Ok(())
    }
}

#[allow(dead_code)]
impl BlockProposer {
    pub fn new() -> Self {
        Self {
            current_height: 0,
            current_round: 0,
            proposal_timeout: 3000, // 3 seconds
            last_proposal_time: 0,
        }
    }

    pub async fn create_block_proposal(
        &mut self,
        chain_state: &ChainState,
        validators: &ValidatorSet,
        transactions: Vec<Transaction>,
        _config: &ConsensusConfig,
    ) -> Result<TendermintBlock> {
        let current_time = SystemTime::now().duration_since(UNIX_EPOCH)?.as_secs();

        let header = BlockHeader {
            version: "1.0".to_string(),
            chain_id: "knirvoracle-1".to_string(),
            height: chain_state.height + 1,
            time: current_time,
            last_block_id: if chain_state.height > 0 {
                Some(BlockId {
                    hash: chain_state.last_block_hash.clone(),
                    parts: PartSetHeader {
                        total: 1,
                        hash: chain_state.last_block_hash.clone(),
                    },
                })
            } else {
                None
            },
            last_commit_hash: chain_state.last_commit_hash.clone(),
            data_hash: self.calculate_data_hash(&transactions)?,
            validators_hash: self.calculate_validators_hash(validators)?,
            next_validators_hash: self.calculate_validators_hash(validators)?,
            consensus_hash: "consensus".to_string(),
            app_hash: chain_state.app_hash.clone(),
            last_results_hash: "results".to_string(),
            evidence_hash: "evidence".to_string(),
            proposer_address: validators
                .get_current_proposer()
                .ok_or_else(|| anyhow!("No current proposer"))?,
        };

        let block = TendermintBlock {
            header,
            data: BlockData { txs: transactions },
            evidence: Evidence {
                evidence: Vec::new(),
            },
            last_commit: None, // Will be filled by consensus
        };

        self.current_height = chain_state.height + 1;
        self.last_proposal_time = current_time;

        Ok(block)
    }

    fn calculate_data_hash(&self, transactions: &[Transaction]) -> Result<String> {
        let data_bytes = serde_json::to_vec(transactions)?;
        let hash = sha2::Sha256::digest(&data_bytes);
        Ok(hex::encode(hash))
    }

    fn calculate_validators_hash(&self, validators: &ValidatorSet) -> Result<String> {
        let validators_bytes = serde_json::to_vec(validators)?;
        let hash = sha2::Sha256::digest(&validators_bytes);
        Ok(hex::encode(hash))
    }
}

#[allow(dead_code)]
impl BlockVoter {
    pub fn new(config: ConsensusConfig) -> Self {
        Self {
            votes: HashMap::new(),
            precommits: HashMap::new(),
            voting_timeout: config.timeout_prevote,
        }
    }

    pub fn create_vote(
        &mut self,
        height: u64,
        round: u32,
        vote_type: VoteType,
        block_hash: Option<String>,
        validator: Address,
    ) -> Result<Vote> {
        let current_time = SystemTime::now().duration_since(UNIX_EPOCH)?.as_secs();

        let vote = Vote {
            height,
            round,
            vote_type,
            block_hash,
            validator: validator.clone(),
            timestamp: current_time,
            signature: "mock_signature".to_string(), // TODO: Implement actual signing
        };

        // Store vote
        self.votes
            .entry(height)
            .or_insert_with(HashMap::new)
            .insert(validator, vote.clone());

        Ok(vote)
    }
}

impl ChainState {
    pub fn new(_chain_id: String) -> Self {
        Self {
            height: 0,
            last_block_hash: "genesis".to_string(),
            last_commit_hash: "genesis".to_string(),
            validators_hash: "genesis".to_string(),
            app_hash: "genesis".to_string(),
            last_block_time: SystemTime::now()
                .duration_since(UNIX_EPOCH)
                .unwrap()
                .as_secs(),
        }
    }
}
