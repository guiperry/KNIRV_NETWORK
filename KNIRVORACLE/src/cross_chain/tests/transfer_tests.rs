//! Unit tests for cross-chain transfer functionality

use super::super::transfer::{CrossChainTransfer, TransferStatus, ChainId, TransferError};
use super::super::bridge::BridgeManager;
use std::str::FromStr;
use num_bigint::BigInt;

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_transfer_creation() {
        let transfer = CrossChainTransfer::new(
            ChainId::KnirvOracle,
            ChainId::KnirvChain,
            "sender_address".to_string(),
            "recipient_address".to_string(),
            1000000, // 1 NRN (in wei)
            "NRN".to_string(),
            1000, // timeout height
            3600, // timeout timestamp (1 hour)
            Some("Test transfer".to_string()),
        );

        assert_eq!(transfer.source_chain, ChainId::KnirvOracle);
        assert_eq!(transfer.dest_chain, ChainId::KnirvChain);
        assert_eq!(transfer.sender, "sender_address");
        assert_eq!(transfer.recipient, "recipient_address");
        assert_eq!(transfer.amount, 1000000);
        assert_eq!(transfer.denom, "NRN");
        assert_eq!(transfer.status, TransferStatus::Pending);
        assert!(transfer.transfer_id.starts_with("transfer_"));
        assert!(transfer.created_at > 0);
        assert!(transfer.completed_at.is_none());
    }

    #[test]
    fn test_transfer_status_update() {
        let mut transfer = CrossChainTransfer::new(
            ChainId::KnirvOracle,
            ChainId::KnirvChain,
            "sender".to_string(),
            "recipient".to_string(),
            1000000,
            "NRN".to_string(),
            1000,
            3600,
            None,
        );

        transfer.update_status(TransferStatus::SourceLocked);
        assert_eq!(transfer.status, TransferStatus::SourceLocked);

        transfer.update_status(TransferStatus::Completed);
        assert_eq!(transfer.status, TransferStatus::Completed);
        assert!(transfer.completed_at.is_some());
    }

    #[test]
    fn test_transfer_timeout_check() {
        let transfer = CrossChainTransfer::new(
            ChainId::KnirvOracle,
            ChainId::KnirvChain,
            "sender".to_string(),
            "recipient".to_string(),
            1000000,
            "NRN".to_string(),
            1000,
            3600,
            None,
        );

        // Not timed out
        assert!(!transfer.is_timed_out(500, 1800));

        // Timed out by height
        assert!(transfer.is_timed_out(1500, 1800));

        // Timed out by timestamp
        assert!(transfer.is_timed_out(500, 7200));
    }

    #[test]
    fn test_chain_id_display() {
        assert_eq!(ChainId::KnirvOracle.to_string(), "knirv-oracle");
        assert_eq!(ChainId::KnirvChain.to_string(), "knirv-chain");
        assert_eq!(ChainId::Xion.to_string(), "xion");
        assert_eq!(ChainId::Cosmos("cosmoshub".to_string()).to_string(), "cosmos-cosmoshub");
        assert_eq!(ChainId::External("ethereum".to_string()).to_string(), "external-ethereum");
    }

    #[test]
    fn test_chain_id_from_str() {
        use std::str::FromStr;
        assert_eq!(ChainId::from_str("knirv-oracle").unwrap(), ChainId::KnirvOracle);
        assert_eq!(ChainId::from_str("knirv-chain").unwrap(), ChainId::KnirvChain);
        assert_eq!(ChainId::from_str("xion").unwrap(), ChainId::Xion);
        assert_eq!(ChainId::from_str("cosmos-cosmoshub").unwrap(), ChainId::Cosmos("cosmoshub".to_string()));
        assert_eq!(ChainId::from_str("external-ethereum").unwrap(), ChainId::External("ethereum".to_string()));
        assert!(ChainId::from_str("invalid").is_err());
    }

    #[test]
    fn test_bridge_manager() {
        let manager = BridgeManager::new();

        // Test default configurations
        let oracle_config = manager.get_config(&ChainId::KnirvOracle);
        assert!(oracle_config.is_some());

        let chain_config = manager.get_config(&ChainId::KnirvChain);
        assert!(chain_config.is_some());

        let xion_config = manager.get_config(&ChainId::Xion);
        assert!(xion_config.is_some());

        // Test route support
        assert!(manager.is_route_supported(&ChainId::KnirvOracle, &ChainId::KnirvChain));
        assert!(manager.is_route_supported(&ChainId::KnirvOracle, &ChainId::Xion));
        assert!(!manager.is_route_supported(&ChainId::KnirvOracle, &ChainId::Cosmos("unknown".to_string())));

        // Test supported chains
        let supported = manager.get_supported_chains();
        assert!(supported.contains(&ChainId::KnirvOracle));
        assert!(supported.contains(&ChainId::KnirvChain));
        assert!(supported.contains(&ChainId::Xion));
    }

    #[test]
    fn test_bridge_config_validation() {
        let manager = BridgeManager::new();
        let config = manager.get_config(&ChainId::KnirvChain).unwrap();

        // Valid amount
        assert!(config.validate_amount(1000000).is_ok());

        // Amount too low
        assert!(config.validate_amount(100).is_err());

        // Amount too high
        assert!(config.validate_amount(2000000000000).is_err());
    }

    #[test]
    fn test_bridge_fee_calculation() {
        let manager = BridgeManager::new();
        let config = manager.get_config(&ChainId::KnirvChain).unwrap();

        let fee = config.calculate_fee(1000000); // 1 NRN
        assert_eq!(fee, 3000); // 0.3% = 3,000 wei
    }

    #[test]
    fn test_transfer_error_display() {
        let error = TransferError::InvalidParameters("Test error".to_string());
        assert!(error.to_string().contains("Invalid transfer parameters"));

        let error = TransferError::InsufficientBalance { required: 1000, available: 500 };
        assert!(error.to_string().contains("Insufficient balance"));

        let error = TransferError::BridgeNotConfigured("source".to_string(), "dest".to_string());
        assert!(error.to_string().contains("Bridge not configured"));
    }
}