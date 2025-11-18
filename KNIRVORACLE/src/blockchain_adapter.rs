use crate::nrn_token::*;
use crate::smart_contracts::*;
use anyhow::Result;
use base64::Engine;
use serde::{Deserialize, Serialize};
use std::collections::HashMap;
use std::sync::Arc;
use tokio::sync::Mutex;
use uuid::Uuid;

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct BlockchainConfig {
    pub mode: BlockchainMode,
    pub xion_config: Option<XionConfig>,
    pub native_config: NativeConfig,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum BlockchainMode {
    Native,
    Xion,
    Hybrid, // Use both native and XION
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct XionConfig {
    pub rpc_endpoint: String,
    pub rest_endpoint: String,
    pub chain_id: String,
    pub nrn_contract: String,
    pub llm_registry_contract: String,
    pub skill_registry_contract: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct NativeConfig {
    pub chain_id: String,
    pub enable_cross_chain: bool,
}

#[derive(Debug, Clone)]
pub struct BlockchainAdapter {
    config: BlockchainConfig,
    smart_contracts: Arc<Mutex<SmartContractEngine>>,
    xion_client: Option<Arc<XionClient>>,
}

#[derive(Debug, Clone)]
pub struct XionClient {
    #[allow(dead_code)]
    rpc_endpoint: String,
    #[allow(dead_code)]
    rest_endpoint: String,
    #[allow(dead_code)]
    chain_id: String,
    #[allow(dead_code)]
    client: reqwest::Client,
}

#[derive(Debug, Serialize, Deserialize)]
pub struct TransactionResult {
    pub success: bool,
    pub tx_hash: String,
    pub block_height: Option<u64>,
    pub gas_used: Option<u64>,
    pub error: Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct LLMRegistrationRequest {
    pub name: String,
    pub version: String,
    pub capabilities: Vec<String>,
    pub model_data: String, // base64 encoded
    pub registration_fee: String,
    pub usage_fee: String,
    pub owner_address: String,
    pub ipfs_hash: Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SkillRegistrationRequest {
    pub name: String,
    pub skill_type: String,
    pub capabilities: Vec<String>,
    pub requirements: HashMap<String, String>,
    pub owner_address: String,
    pub usage_fee: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SkillInvocationRequest {
    pub skill_id: String,
    pub user_private_key: String,
    pub amount: String,
}

impl BlockchainAdapter {
    pub fn new(
        config: BlockchainConfig,
        smart_contracts: Arc<Mutex<SmartContractEngine>>,
    ) -> Result<Self> {
        let xion_client = if matches!(config.mode, BlockchainMode::Xion | BlockchainMode::Hybrid) {
            if let Some(xion_config) = &config.xion_config {
                Some(Arc::new(XionClient::new(
                    xion_config.rpc_endpoint.clone(),
                    xion_config.rest_endpoint.clone(),
                    xion_config.chain_id.clone(),
                )))
            } else {
                return Err(anyhow::anyhow!("XION config required for XION mode"));
            }
        } else {
            None
        };

        Ok(Self {
            config,
            smart_contracts,
            xion_client,
        })
    }

    pub async fn register_llm(&self, request: LLMRegistrationRequest) -> Result<TransactionResult> {
        match self.config.mode {
            BlockchainMode::Native => self.register_llm_native(request).await,
            BlockchainMode::Xion => self.register_llm_xion(request).await,
            BlockchainMode::Hybrid => {
                // Register on both chains
                let native_result = self.register_llm_native(request.clone()).await?;
                let xion_result = self.register_llm_xion(request).await?;

                // Return combined result
                Ok(TransactionResult {
                    success: native_result.success && xion_result.success,
                    tx_hash: format!(
                        "native:{},xion:{}",
                        native_result.tx_hash, xion_result.tx_hash
                    ),
                    block_height: native_result.block_height,
                    gas_used: native_result.gas_used,
                    error: None,
                })
            }
        }
    }

    pub async fn register_skill(
        &self,
        request: SkillRegistrationRequest,
    ) -> Result<TransactionResult> {
        match self.config.mode {
            BlockchainMode::Native => self.register_skill_native(request).await,
            BlockchainMode::Xion => self.register_skill_xion(request).await,
            BlockchainMode::Hybrid => {
                let native_result = self.register_skill_native(request.clone()).await?;
                let xion_result = self.register_skill_xion(request).await?;

                Ok(TransactionResult {
                    success: native_result.success && xion_result.success,
                    tx_hash: format!(
                        "native:{},xion:{}",
                        native_result.tx_hash, xion_result.tx_hash
                    ),
                    block_height: native_result.block_height,
                    gas_used: native_result.gas_used,
                    error: None,
                })
            }
        }
    }

    pub async fn invoke_skill(&self, request: SkillInvocationRequest) -> Result<TransactionResult> {
        match self.config.mode {
            BlockchainMode::Native => self.invoke_skill_native(request).await,
            BlockchainMode::Xion => self.invoke_skill_xion(request).await,
            BlockchainMode::Hybrid => {
                // For skill invocation, we primarily use native chain but can sync to XION
                let native_result = self.invoke_skill_native(request.clone()).await?;

                // Optionally sync to XION (non-blocking)
                if native_result.success {
                    tokio::spawn({
                        let adapter = self.clone();
                        let req = request;
                        async move {
                            let _ = adapter.invoke_skill_xion(req).await;
                        }
                    });
                }

                Ok(native_result)
            }
        }
    }

    async fn register_llm_native(
        &self,
        request: LLMRegistrationRequest,
    ) -> Result<TransactionResult> {
        let mut smart_contracts = self.smart_contracts.lock().await;

        let owner_address = hex_to_address(&request.owner_address)?;
        let model_data = base64::engine::general_purpose::STANDARD.decode(&request.model_data)?;

        let metadata = LLMMetadata {
            name: request.name,
            version: request.version,
            model_hash: String::new(), // Will be calculated by register_llm
            capabilities: request.capabilities,
            owner: owner_address,
            registration_fee: request.registration_fee.parse()?,
            usage_fee: request.usage_fee.parse()?,
            ipfs_hash: request.ipfs_hash,
            validation_status: ValidationStatus::Pending,
            registered_at: 0, // Will be set by register_llm
        };

        let _model_hash = smart_contracts
            .llm_registry
            .register_llm(metadata, &model_data)?;

        Ok(TransactionResult {
            success: true,
            tx_hash: format!("native_{}", Uuid::new_v4()),
            block_height: Some(1),  // Placeholder
            gas_used: Some(100000), // Placeholder
            error: None,
        })
    }

    async fn register_llm_xion(
        &self,
        _request: LLMRegistrationRequest,
    ) -> Result<TransactionResult> {
        if let Some(_xion_client) = &self.xion_client {
            // Implementation would call XION smart contracts
            // For now, return a placeholder
            Ok(TransactionResult {
                success: true,
                tx_hash: format!("xion_{}", Uuid::new_v4()),
                block_height: Some(1),
                gas_used: Some(150000),
                error: None,
            })
        } else {
            Err(anyhow::anyhow!("XION client not configured"))
        }
    }

    async fn register_skill_native(
        &self,
        request: SkillRegistrationRequest,
    ) -> Result<TransactionResult> {
        let mut smart_contracts = self.smart_contracts.lock().await;

        let owner_address = hex_to_address(&request.owner_address)?;

        let skill = SkillMetadata {
            name: request.name,
            skill_type: request.skill_type,
            capabilities: request.capabilities,
            requirements: request.requirements,
            owner: owner_address,
            usage_fee: request.usage_fee.parse()?,
            validation_status: ValidationStatus::Pending,
            performance_metrics: PerformanceMetrics {
                success_rate: 0.0,
                average_latency: 0.0,
                total_invocations: 0,
                last_updated: 0,
            },
            registered_at: std::time::SystemTime::now()
                .duration_since(std::time::UNIX_EPOCH)
                .unwrap()
                .as_secs(),
        };

        let _skill_id = smart_contracts.skill_registry.register_skill(skill)?;

        Ok(TransactionResult {
            success: true,
            tx_hash: format!("native_{}", Uuid::new_v4()),
            block_height: Some(1),
            gas_used: Some(80000),
            error: None,
        })
    }

    async fn register_skill_xion(
        &self,
        _request: SkillRegistrationRequest,
    ) -> Result<TransactionResult> {
        if let Some(_xion_client) = &self.xion_client {
            // Implementation would call XION smart contracts
            Ok(TransactionResult {
                success: true,
                tx_hash: format!("xion_{}", Uuid::new_v4()),
                block_height: Some(1),
                gas_used: Some(120000),
                error: None,
            })
        } else {
            Err(anyhow::anyhow!("XION client not configured"))
        }
    }

    async fn invoke_skill_native(
        &self,
        request: SkillInvocationRequest,
    ) -> Result<TransactionResult> {
        let mut smart_contracts = self.smart_contracts.lock().await;

        let amount = request.amount.parse()?;
        let _tx = smart_contracts.nrn_token.burn_for_skill(
            &request.user_private_key,
            &request.skill_id,
            &amount,
        )?;

        // Record skill invocation
        let user_address = get_address_from_private_key(&request.user_private_key)?;
        smart_contracts.skill_registry.invoke_skill(
            &request.skill_id,
            user_address,
            amount,
            true, // Assume success for now
            None,
        )?;

        Ok(TransactionResult {
            success: true,
            tx_hash: format!("native_{}", Uuid::new_v4()),
            block_height: Some(1),
            gas_used: Some(60000),
            error: None,
        })
    }

    async fn invoke_skill_xion(
        &self,
        _request: SkillInvocationRequest,
    ) -> Result<TransactionResult> {
        if let Some(_xion_client) = &self.xion_client {
            // Implementation would call XION smart contracts
            Ok(TransactionResult {
                success: true,
                tx_hash: format!("xion_{}", Uuid::new_v4()),
                block_height: Some(1),
                gas_used: Some(90000),
                error: None,
            })
        } else {
            Err(anyhow::anyhow!("XION client not configured"))
        }
    }
}

impl XionClient {
    pub fn new(rpc_endpoint: String, rest_endpoint: String, chain_id: String) -> Self {
        Self {
            rpc_endpoint,
            rest_endpoint,
            chain_id,
            client: reqwest::Client::new(),
        }
    }

    // Placeholder methods for XION integration
    #[allow(dead_code)]
    pub async fn query_contract(
        &self,
        _contract_addr: &str,
        _query_msg: serde_json::Value,
    ) -> Result<serde_json::Value> {
        // Implementation would query XION smart contracts
        Ok(serde_json::json!({}))
    }

    #[allow(dead_code)]
    pub async fn execute_contract(
        &self,
        _contract_addr: &str,
        _execute_msg: serde_json::Value,
    ) -> Result<String> {
        // Implementation would execute XION smart contracts
        Ok(format!("xion_tx_{}", Uuid::new_v4()))
    }
}
