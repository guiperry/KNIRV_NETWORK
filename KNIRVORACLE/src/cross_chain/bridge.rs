//! Bridge configurations for cross-chain transfers

use serde::{Deserialize, Serialize};
use std::collections::HashMap;
use crate::cross_chain::transfer::ChainId;

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
    pub transfer_fee_basis_points: u16, // 100 = 1%
    pub enabled: bool,
}

impl BridgeConfig {
    /// Create a new bridge configuration
    pub fn new(
        chain_id: ChainId,
        channel_id: String,
        port_id: String,
        connection_id: String,
        client_id: String,
    ) -> Self {
        Self {
            chain_id,
            channel_id,
            port_id,
            connection_id,
            client_id,
            trust_level: 0.66, // 2/3 trust level
            max_transfer_amount: 1_000_000_000_000, // 1M NRN
            min_transfer_amount: 1_000, // 0.001 NRN
            transfer_fee_basis_points: 30, // 0.3%
            enabled: true,
        }
    }

    /// Calculate transfer fee for given amount
    pub fn calculate_fee(&self, amount: u64) -> u64 {
        ((amount as u128 * self.transfer_fee_basis_points as u128) / 10000) as u64
    }

    /// Validate transfer amount
    pub fn validate_amount(&self, amount: u64) -> Result<(), String> {
        if amount < self.min_transfer_amount {
            return Err(format!(
                "Transfer amount {} below minimum {}",
                amount, self.min_transfer_amount
            ));
        }
        if amount > self.max_transfer_amount {
            return Err(format!(
                "Transfer amount {} above maximum {}",
                amount, self.max_transfer_amount
            ));
        }
        Ok(())
    }

    /// Check if bridge is operational
    pub fn is_operational(&self) -> bool {
        self.enabled && self.trust_level >= 0.5
    }
}

/// Bridge configuration manager
#[derive(Debug)]
pub struct BridgeManager {
    configs: HashMap<ChainId, BridgeConfig>,
}

impl BridgeManager {
    /// Create a new bridge manager with default configurations
    pub fn new() -> Self {
        let mut configs = HashMap::new();

        // KNIRV Oracle bridge (self-bridge for consistency)
        configs.insert(
            ChainId::KnirvOracle,
            BridgeConfig::new(
                ChainId::KnirvOracle,
                "channel-oracle-1".to_string(),
                "transfer".to_string(),
                "connection-oracle-1".to_string(),
                "client-oracle-1".to_string(),
            ),
        );

        // KNIRV Chain bridge
        configs.insert(
            ChainId::KnirvChain,
            BridgeConfig::new(
                ChainId::KnirvChain,
                "channel-1".to_string(),
                "transfer".to_string(),
                "connection-1".to_string(),
                "client-1".to_string(),
            ),
        );

        // KNIRV Nexus bridge
        configs.insert(
            ChainId::KnirvNexus,
            BridgeConfig::new(
                ChainId::KnirvNexus,
                "channel-2".to_string(),
                "transfer".to_string(),
                "connection-2".to_string(),
                "client-2".to_string(),
            ),
        );

        // XION bridge for USDC transfers
        let mut xion_config = BridgeConfig::new(
            ChainId::Xion,
            "channel-xion-1".to_string(),
            "transfer".to_string(),
            "connection-xion-1".to_string(),
            "client-xion-1".to_string(),
        );
        xion_config.transfer_fee_basis_points = 50; // 0.5% for external bridge
        configs.insert(ChainId::Xion, xion_config);

        Self { configs }
    }

    /// Get bridge configuration for a chain
    pub fn get_config(&self, chain_id: &ChainId) -> Option<&BridgeConfig> {
        self.configs.get(chain_id)
    }

    /// Get all bridge configurations
    pub fn get_all_configs(&self) -> &HashMap<ChainId, BridgeConfig> {
        &self.configs
    }

    /// Add or update bridge configuration
    pub fn set_config(&mut self, config: BridgeConfig) {
        self.configs.insert(config.chain_id.clone(), config);
    }

    /// Remove bridge configuration
    pub fn remove_config(&mut self, chain_id: &ChainId) -> Option<BridgeConfig> {
        self.configs.remove(chain_id)
    }

    /// Check if route is supported
    pub fn is_route_supported(&self, source: &ChainId, dest: &ChainId) -> bool {
        self.get_config(source).is_some() && self.get_config(dest).is_some()
    }

    /// Get supported chains
    pub fn get_supported_chains(&self) -> Vec<ChainId> {
        self.configs.keys().cloned().collect()
    }

    /// Get operational bridges
    pub fn get_operational_bridges(&self) -> Vec<&BridgeConfig> {
        self.configs.values().filter(|config| config.is_operational()).collect()
    }
}

impl Default for BridgeManager {
    fn default() -> Self {
        Self::new()
    }
}

/// Bridge status information
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct BridgeStatus {
    pub chain_id: ChainId,
    pub operational: bool,
    pub trust_level: f64,
    pub total_transfers: u64,
    pub pending_transfers: u64,
    pub failed_transfers: u64,
    pub last_block_height: u64,
    pub last_update: i64,
}

impl BridgeStatus {
    /// Create bridge status from configuration
    pub fn from_config(config: &BridgeConfig) -> Self {
        Self {
            chain_id: config.chain_id.clone(),
            operational: config.is_operational(),
            trust_level: config.trust_level,
            total_transfers: 0,
            pending_transfers: 0,
            failed_transfers: 0,
            last_block_height: 0,
            last_update: chrono::Utc::now().timestamp(),
        }
    }
}

/// Bridge health check result
#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum BridgeHealth {
    Healthy,
    Degraded(String),
    Unhealthy(String),
}

impl BridgeManager {
    /// Perform health check on all bridges
    pub async fn health_check(&self) -> HashMap<ChainId, BridgeHealth> {
        let mut results = HashMap::new();

        for (chain_id, config) in &self.configs {
            let health = if config.is_operational() {
                // In a real implementation, this would check actual connectivity
                // For now, we assume healthy if enabled
                BridgeHealth::Healthy
            } else {
                BridgeHealth::Unhealthy("Bridge disabled".to_string())
            };
            results.insert(chain_id.clone(), health);
        }

        results
    }
}