use crate::blockchain_adapter::*;
use anyhow::Result;
use serde::{Deserialize, Serialize};
use std::fs;

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Config {
    pub blockchain: BlockchainSettings,
    pub native: NativeSettings,
    pub xion: Option<XionSettings>,
    pub smart_contracts: SmartContractSettings,
    pub network: NetworkSettings,
    pub api: ApiSettings,
    pub logging: LoggingSettings,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct BlockchainSettings {
    pub mode: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct NativeSettings {
    pub chain_id: String,
    pub enable_cross_chain: bool,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct XionSettings {
    pub rpc_endpoint: String,
    pub rest_endpoint: String,
    pub chain_id: String,
    pub nrn_contract: String,
    pub llm_registry_contract: String,
    pub skill_registry_contract: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SmartContractSettings {
    pub initial_supply: String,
    pub max_supply: String,
    pub default_registration_fee: String,
    pub default_usage_fee: String,
    pub default_skill_fee: String,
    pub performance_tracking: bool,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct NetworkSettings {
    pub reward_multiplier: String,
    pub max_daily_rewards: String,
    pub connectivity_threshold: f64,
    pub reward_frequency_seconds: u64,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ApiSettings {
    pub enable_v1_endpoints: bool,
    pub enable_v2_endpoints: bool,
    pub default_version: String,
    pub max_requests_per_minute: u32,
    pub burst_limit: u32,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct LoggingSettings {
    pub level: String,
    pub enable_transaction_logging: bool,
    pub enable_performance_metrics: bool,
}

impl Config {
    pub fn load_from_file(path: &str) -> Result<Self> {
        let content = fs::read_to_string(path)?;
        let config: Config = toml::from_str(&content)?;
        Ok(config)
    }

    pub fn load_default() -> Self {
        Self {
            blockchain: BlockchainSettings {
                mode: "native".to_string(),
            },
            native: NativeSettings {
                chain_id: "knirv-1".to_string(),
                enable_cross_chain: false,
            },
            xion: None,
            smart_contracts: SmartContractSettings {
                initial_supply: "1000000000000000".to_string(),
                max_supply: "10000000000000000".to_string(),
                default_registration_fee: "1000".to_string(),
                default_usage_fee: "10".to_string(),
                default_skill_fee: "100".to_string(),
                performance_tracking: true,
            },
            network: NetworkSettings {
                reward_multiplier: "1.5".to_string(),
                max_daily_rewards: "10000".to_string(),
                connectivity_threshold: 0.8,
                reward_frequency_seconds: 3600,
            },
            api: ApiSettings {
                enable_v1_endpoints: true,
                enable_v2_endpoints: true,
                default_version: "v2".to_string(),
                max_requests_per_minute: 100,
                burst_limit: 20,
            },
            logging: LoggingSettings {
                level: "info".to_string(),
                enable_transaction_logging: true,
                enable_performance_metrics: true,
            },
        }
    }

    pub fn to_blockchain_config(&self) -> Result<BlockchainConfig> {
        let mode = match self.blockchain.mode.as_str() {
            "native" => BlockchainMode::Native,
            "xion" => BlockchainMode::Xion,
            "hybrid" => BlockchainMode::Hybrid,
            _ => {
                return Err(anyhow::anyhow!(
                    "Invalid blockchain mode: {}",
                    self.blockchain.mode
                ))
            }
        };

        let xion_config = if matches!(mode, BlockchainMode::Xion | BlockchainMode::Hybrid) {
            if let Some(xion_settings) = &self.xion {
                Some(XionConfig {
                    rpc_endpoint: xion_settings.rpc_endpoint.clone(),
                    rest_endpoint: xion_settings.rest_endpoint.clone(),
                    chain_id: xion_settings.chain_id.clone(),
                    nrn_contract: xion_settings.nrn_contract.clone(),
                    llm_registry_contract: xion_settings.llm_registry_contract.clone(),
                    skill_registry_contract: xion_settings.skill_registry_contract.clone(),
                })
            } else {
                return Err(anyhow::anyhow!(
                    "XION configuration required for XION/Hybrid mode"
                ));
            }
        } else {
            None
        };

        Ok(BlockchainConfig {
            mode,
            xion_config,
            native_config: NativeConfig {
                chain_id: self.native.chain_id.clone(),
                enable_cross_chain: self.native.enable_cross_chain,
            },
        })
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_default_config() {
        let config = Config::load_default();
        assert_eq!(config.blockchain.mode, "native");
        assert_eq!(config.native.chain_id, "knirv-1");
        assert!(!config.native.enable_cross_chain);
    }

    #[test]
    fn test_blockchain_config_conversion() {
        let config = Config::load_default();
        let blockchain_config = config.to_blockchain_config().unwrap();

        assert!(matches!(blockchain_config.mode, BlockchainMode::Native));
        assert_eq!(blockchain_config.native_config.chain_id, "knirv-1");
        assert!(blockchain_config.xion_config.is_none());
    }
}
