//! Cross-chain transfer module for KNIRVORACLE
//!
//! This module implements IBC-based cross-chain asset transfers
//! between KNIRV chains and external networks.

pub mod router;
pub mod transfer;
pub mod bridge;
pub mod proof;

#[cfg(test)]
mod tests;

// Re-export main types
pub use router::CrossChainRouter;
pub use transfer::{CrossChainTransfer, TransferStatus, ChainId, TransferProof};
pub use bridge::BridgeConfig;