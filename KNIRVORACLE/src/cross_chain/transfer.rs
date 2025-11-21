//! Cross-chain transfer data structures and types

use serde::{Deserialize, Serialize};
use std::collections::HashMap;

/// Cross-chain transfer request
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CrossChainTransfer {
    pub transfer_id: String,
    pub source_chain: ChainId,
    pub dest_chain: ChainId,
    pub sender: String,
    pub recipient: String,
    pub amount: u64,
    pub denom: String, // NRN, USDC, etc.
    pub timeout_height: u64,
    pub timeout_timestamp: u64,
    pub memo: Option<String>,
    pub status: TransferStatus,
    pub created_at: i64,
    pub completed_at: Option<i64>,
}

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq, Eq, Hash)]
pub enum ChainId {
    KnirvChain,
    KnirvOracle,
    KnirvNexus,
    KnirvRouter,
    KnirvGraph,
    Xion,
    Cosmos(String), // Generic Cosmos chain
    External(String), // Non-Cosmos chains
}

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
pub enum TransferStatus {
    Pending,
    SourceLocked, // Tokens locked on source chain
    InTransit,    // IBC packet sent
    DestReceived, // Received on destination
    Completed,    // Fully finalized
    Failed(String), // Error message
    Refunded,    // Tokens returned to sender
    TimedOut,
}

/// Transfer proof for validation
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TransferProof {
    pub transfer_id: String,
    pub merkle_proof: Vec<Vec<u8>>,
    pub block_height: u64,
    pub block_hash: String,
    pub validator_signatures: Vec<ValidatorSignature>,
}

/// Validator signature for proof validation
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ValidatorSignature {
    pub validator_address: String,
    pub signature: Vec<u8>,
    pub timestamp: i64,
}

/// Transfer error types
#[derive(Debug, thiserror::Error)]
pub enum TransferError {
    #[error("Invalid transfer parameters: {0}")]
    InvalidParameters(String),

    #[error("Insufficient balance: required {required}, available {available}")]
    InsufficientBalance { required: u64, available: u64 },

    #[error("Chain not supported: {0}")]
    UnsupportedChain(String),

    #[error("Bridge not configured for route {0} -> {1}")]
    BridgeNotConfigured(String, String),

    #[error("Transfer timeout exceeded")]
    TimeoutExceeded,

    #[error("IBC transmission failed: {0}")]
    IbcTransmissionError(String),

    #[error("Proof validation failed: {0}")]
    ProofValidationError(String),

    #[error("Transfer already exists: {0}")]
    TransferAlreadyExists(String),

    #[error("Transfer not found: {0}")]
    TransferNotFound(String),

    #[error("Internal error: {0}")]
    InternalError(String),
}

/// Transfer receipt returned after successful operations
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TransferReceipt {
    pub transfer_id: String,
    pub status: TransferStatus,
    pub timestamp: i64,
    pub fee_amount: u64,
    pub fee_denom: String,
    pub transaction_hash: Option<String>,
}

/// Transfer history entry
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TransferHistory {
    pub transfers: Vec<CrossChainTransfer>,
    pub total_count: usize,
    pub page: usize,
    pub page_size: usize,
}

impl CrossChainTransfer {
    /// Create a new cross-chain transfer
    pub fn new(
        source_chain: ChainId,
        dest_chain: ChainId,
        sender: String,
        recipient: String,
        amount: u64,
        denom: String,
        timeout_height: u64,
        timeout_timestamp: u64,
        memo: Option<String>,
    ) -> Self {
        let transfer_id = format!("transfer_{}_{}", uuid::Uuid::new_v4(), chrono::Utc::now().timestamp());
        Self {
            transfer_id,
            source_chain,
            dest_chain,
            sender,
            recipient,
            amount,
            denom,
            timeout_height,
            timeout_timestamp,
            memo,
            status: TransferStatus::Pending,
            created_at: chrono::Utc::now().timestamp(),
            completed_at: None,
        }
    }

    /// Check if transfer has timed out
    pub fn is_timed_out(&self, current_height: u64, current_timestamp: u64) -> bool {
        current_height >= self.timeout_height || current_timestamp >= self.timeout_timestamp
    }

    /// Update transfer status
    pub fn update_status(&mut self, new_status: TransferStatus) {
        self.status = new_status.clone();
        if matches!(new_status, TransferStatus::Completed | TransferStatus::Failed(_) | TransferStatus::Refunded) {
            self.completed_at = Some(chrono::Utc::now().timestamp());
        }
    }
}

impl Default for TransferStatus {
    fn default() -> Self {
        TransferStatus::Pending
    }
}

impl std::fmt::Display for ChainId {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        match self {
            ChainId::KnirvChain => write!(f, "knirv-chain"),
            ChainId::KnirvOracle => write!(f, "knirv-oracle"),
            ChainId::KnirvNexus => write!(f, "knirv-nexus"),
            ChainId::KnirvRouter => write!(f, "knirv-router"),
            ChainId::KnirvGraph => write!(f, "knirv-graph"),
            ChainId::Xion => write!(f, "xion"),
            ChainId::Cosmos(chain) => write!(f, "cosmos-{}", chain),
            ChainId::External(chain) => write!(f, "external-{}", chain),
        }
    }
}

impl std::str::FromStr for ChainId {
    type Err = String;

    fn from_str(s: &str) -> Result<Self, Self::Err> {
        match s {
            "knirv-chain" => Ok(ChainId::KnirvChain),
            "knirv-oracle" => Ok(ChainId::KnirvOracle),
            "knirv-nexus" => Ok(ChainId::KnirvNexus),
            "knirv-router" => Ok(ChainId::KnirvRouter),
            "knirv-graph" => Ok(ChainId::KnirvGraph),
            "xion" => Ok(ChainId::Xion),
            chain if chain.starts_with("cosmos-") => {
                Ok(ChainId::Cosmos(chain[7..].to_string()))
            }
            chain if chain.starts_with("external-") => {
                Ok(ChainId::External(chain[9..].to_string()))
            }
            _ => Err(format!("Unknown chain ID: {}", s)),
        }
    }
}