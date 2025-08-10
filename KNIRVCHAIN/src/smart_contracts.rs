use crate::nrn_token::*;
use anyhow::Result;
use serde::{Deserialize, Serialize};
use serde_json;
use std::collections::HashMap;

#[derive(Debug, Serialize, Deserialize, Clone)]
pub struct SmartContractEngine {
    pub nrn_token: NRN,
    pub llm_registry: LLMRegistry,
    pub skill_registry: SkillRegistry,
}

#[derive(Debug, Serialize, Deserialize)]
pub struct ContractCall {
    pub contract: String,
    pub method: String,
    pub params: serde_json::Value,
}

#[derive(Debug, Serialize, Deserialize)]
pub struct ContractResponse {
    pub success: bool,
    pub data: Option<serde_json::Value>,
    pub error: Option<String>,
}

impl SmartContractEngine {
    pub fn new(owner_private_key: &str) -> Result<Self> {
        let initial_supply = num_bigint::BigInt::from(1_000_000_000_000_000u64); // 1 trillion
        let max_supply = num_bigint::BigInt::from(10_000_000_000_000_000u64); // 10 trillion

        let nrn_token = NRN::new(
            "KNIRV Network Token".to_string(),
            "NRN".to_string(),
            initial_supply,
            max_supply,
            owner_private_key,
        )?;

        Ok(Self {
            nrn_token,
            llm_registry: LLMRegistry::new(),
            skill_registry: SkillRegistry::new(),
        })
    }

    pub fn execute_contract_call(&mut self, call: ContractCall) -> ContractResponse {
        match call.contract.as_str() {
            "nrn_token" => self.execute_nrn_call(call.method, call.params),
            "llm_registry" => self.execute_llm_call(call.method, call.params),
            "skill_registry" => self.execute_skill_call(call.method, call.params),
            _ => ContractResponse {
                success: false,
                data: None,
                error: Some("Unknown contract".to_string()),
            },
        }
    }

    fn execute_nrn_call(&mut self, method: String, params: serde_json::Value) -> ContractResponse {
        match method.as_str() {
            "transfer" => {
                let from = params["from"].as_str().unwrap_or("");
                let to_str = params["to"].as_str().unwrap_or("");
                let amount_str = params["amount"].as_str().unwrap_or("0");

                match (
                    hex_to_address(to_str),
                    amount_str.parse::<num_bigint::BigInt>(),
                ) {
                    (Ok(to_address), Ok(amount)) => {
                        match self.nrn_token.transfer(from, to_address, &amount) {
                            Ok(tx) => ContractResponse {
                                success: true,
                                data: Some(serde_json::to_value(tx).unwrap()),
                                error: None,
                            },
                            Err(e) => ContractResponse {
                                success: false,
                                data: None,
                                error: Some(e.to_string()),
                            },
                        }
                    }
                    _ => ContractResponse {
                        success: false,
                        data: None,
                        error: Some("Invalid parameters".to_string()),
                    },
                }
            }
            "burn_for_skill" => {
                let from = params["from"].as_str().unwrap_or("");
                let skill_id = params["skill_id"].as_str().unwrap_or("");
                let amount_str = params["amount"].as_str().unwrap_or("0");

                match amount_str.parse::<num_bigint::BigInt>() {
                    Ok(amount) => {
                        match self.nrn_token.burn_for_skill(from, skill_id, &amount) {
                            Ok(tx) => {
                                // Record skill invocation
                                if let Ok(user_address) = get_address_from_private_key(from) {
                                    let _ = self.skill_registry.invoke_skill(
                                        skill_id,
                                        user_address,
                                        amount.clone(),
                                        true, // Assume success for now
                                        None,
                                    );
                                }

                                ContractResponse {
                                    success: true,
                                    data: Some(serde_json::to_value(tx).unwrap()),
                                    error: None,
                                }
                            }
                            Err(e) => ContractResponse {
                                success: false,
                                data: None,
                                error: Some(e.to_string()),
                            },
                        }
                    }
                    Err(_) => ContractResponse {
                        success: false,
                        data: None,
                        error: Some("Invalid amount".to_string()),
                    },
                }
            }
            "mint_reward" => {
                let to_str = params["to"].as_str().unwrap_or("");
                let amount_str = params["amount"].as_str().unwrap_or("0");
                let reason = params["reason"].as_str().unwrap_or("Network participation");

                match (
                    hex_to_address(to_str),
                    amount_str.parse::<num_bigint::BigInt>(),
                ) {
                    (Ok(to_address), Ok(amount)) => {
                        match self.nrn_token.mint_reward(to_address, &amount, reason) {
                            Ok(_) => ContractResponse {
                                success: true,
                                data: Some(serde_json::json!({"minted": amount.to_string()})),
                                error: None,
                            },
                            Err(e) => ContractResponse {
                                success: false,
                                data: None,
                                error: Some(e.to_string()),
                            },
                        }
                    }
                    _ => ContractResponse {
                        success: false,
                        data: None,
                        error: Some("Invalid parameters".to_string()),
                    },
                }
            }
            "balance" => {
                let address_str = params["address"].as_str().unwrap_or("");
                match hex_to_address(address_str) {
                    Ok(address) => {
                        let balance = self.nrn_token.get_balance(&address);
                        ContractResponse {
                            success: true,
                            data: Some(serde_json::json!({"balance": balance.to_string()})),
                            error: None,
                        }
                    }
                    Err(_) => ContractResponse {
                        success: false,
                        data: None,
                        error: Some("Invalid address".to_string()),
                    },
                }
            }
            _ => ContractResponse {
                success: false,
                data: None,
                error: Some("Unknown method".to_string()),
            },
        }
    }

    fn execute_llm_call(&mut self, method: String, params: serde_json::Value) -> ContractResponse {
        match method.as_str() {
            "register" => {
                // Parse LLM metadata from params
                let name = params["name"].as_str().unwrap_or("").to_string();
                let version = params["version"].as_str().unwrap_or("").to_string();
                let capabilities: Vec<String> = params["capabilities"]
                    .as_array()
                    .unwrap_or(&vec![])
                    .iter()
                    .filter_map(|v| v.as_str().map(|s| s.to_string()))
                    .collect();
                let owner_str = params["owner"].as_str().unwrap_or("");
                let registration_fee_str = params["registration_fee"].as_str().unwrap_or("0");
                let usage_fee_str = params["usage_fee"].as_str().unwrap_or("0");
                let model_data = params["model_data"].as_str().unwrap_or("").as_bytes();

                match (
                    hex_to_address(owner_str),
                    registration_fee_str.parse::<num_bigint::BigInt>(),
                    usage_fee_str.parse::<num_bigint::BigInt>(),
                ) {
                    (Ok(owner), Ok(reg_fee), Ok(usage_fee)) => {
                        let metadata = LLMMetadata {
                            name,
                            version,
                            model_hash: String::new(), // Will be set by register_llm
                            capabilities,
                            owner,
                            registration_fee: reg_fee,
                            usage_fee,
                            ipfs_hash: params["ipfs_hash"].as_str().map(|s| s.to_string()),
                            validation_status: ValidationStatus::Pending,
                            registered_at: 0, // Will be set by register_llm
                        };

                        match self.llm_registry.register_llm(metadata, model_data) {
                            Ok(model_hash) => ContractResponse {
                                success: true,
                                data: Some(serde_json::json!({"model_hash": model_hash})),
                                error: None,
                            },
                            Err(e) => ContractResponse {
                                success: false,
                                data: None,
                                error: Some(e.to_string()),
                            },
                        }
                    }
                    _ => ContractResponse {
                        success: false,
                        data: None,
                        error: Some("Invalid parameters".to_string()),
                    },
                }
            }
            "validate" => {
                let model_hash = params["model_hash"].as_str().unwrap_or("");
                let is_valid = params["is_valid"].as_bool().unwrap_or(false);

                match self.llm_registry.validate_llm(model_hash, is_valid) {
                    Ok(_) => ContractResponse {
                        success: true,
                        data: Some(serde_json::json!({"validated": is_valid})),
                        error: None,
                    },
                    Err(e) => ContractResponse {
                        success: false,
                        data: None,
                        error: Some(e.to_string()),
                    },
                }
            }
            "get" => {
                let model_hash = params["model_hash"].as_str().unwrap_or("");
                match self.llm_registry.get_llm(model_hash) {
                    Some(metadata) => ContractResponse {
                        success: true,
                        data: Some(serde_json::to_value(metadata).unwrap()),
                        error: None,
                    },
                    None => ContractResponse {
                        success: false,
                        data: None,
                        error: Some("Model not found".to_string()),
                    },
                }
            }
            _ => ContractResponse {
                success: false,
                data: None,
                error: Some("Unknown method".to_string()),
            },
        }
    }

    fn execute_skill_call(
        &mut self,
        method: String,
        params: serde_json::Value,
    ) -> ContractResponse {
        match method.as_str() {
            "register" => {
                let name = params["name"].as_str().unwrap_or("").to_string();
                let skill_type = params["skill_type"].as_str().unwrap_or("").to_string();
                let capabilities: Vec<String> = params["capabilities"]
                    .as_array()
                    .unwrap_or(&vec![])
                    .iter()
                    .filter_map(|v| v.as_str().map(|s| s.to_string()))
                    .collect();
                let owner_str = params["owner"].as_str().unwrap_or("");
                let usage_fee_str = params["usage_fee"].as_str().unwrap_or("0");

                match (
                    hex_to_address(owner_str),
                    usage_fee_str.parse::<num_bigint::BigInt>(),
                ) {
                    (Ok(owner), Ok(usage_fee)) => {
                        let skill = SkillMetadata {
                            name,
                            skill_type,
                            capabilities,
                            requirements: HashMap::new(), // Could be extended
                            owner,
                            usage_fee,
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

                        match self.skill_registry.register_skill(skill) {
                            Ok(skill_id) => ContractResponse {
                                success: true,
                                data: Some(serde_json::json!({"skill_id": skill_id})),
                                error: None,
                            },
                            Err(e) => ContractResponse {
                                success: false,
                                data: None,
                                error: Some(e.to_string()),
                            },
                        }
                    }
                    _ => ContractResponse {
                        success: false,
                        data: None,
                        error: Some("Invalid parameters".to_string()),
                    },
                }
            }
            "get" => {
                let skill_id = params["skill_id"].as_str().unwrap_or("");
                match self.skill_registry.get_skill(skill_id) {
                    Some(skill) => ContractResponse {
                        success: true,
                        data: Some(serde_json::to_value(skill).unwrap()),
                        error: None,
                    },
                    None => ContractResponse {
                        success: false,
                        data: None,
                        error: Some("Skill not found".to_string()),
                    },
                }
            }
            _ => ContractResponse {
                success: false,
                data: None,
                error: Some("Unknown method".to_string()),
            },
        }
    }
}
