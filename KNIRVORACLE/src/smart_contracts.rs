use crate::nrn_token::*;
use anyhow::Result;
use serde::{Deserialize, Serialize};
use serde_json;
use std::collections::HashMap;

#[derive(Debug, Serialize, Deserialize, Clone)]
pub struct SmartContractEngine {
    pub nrn_token: NRN,
    // Registry functionality has been moved to KNIRVCHAIN
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
            // Registry functionality has been moved to KNIRVCHAIN
        })
    }

    pub fn execute_contract_call(&mut self, call: ContractCall) -> ContractResponse {
        match call.contract.as_str() {
            "nrn_token" => self.execute_nrn_call(call.method, call.params),
            // Registry contracts have been moved to KNIRVCHAIN
            "llm_registry" | "skill_registry" => ContractResponse {
                success: false,
                data: None,
                error: Some("Registry functionality has been moved to KNIRVCHAIN".to_string()),
            },
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
                                // Skill invocation recording has been moved to KNIRVCHAIN
                                // Events will be received via IBC from KNIRVCHAIN

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

// Registry contract methods have been removed
// All registry functionality is now handled by KNIRVCHAIN
}
