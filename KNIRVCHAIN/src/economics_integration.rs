use serde::{Deserialize, Serialize};
use std::collections::HashMap;
use reqwest::Client;
use tokio::time::{Duration, interval};
use tracing::{info, error, warn};

#[derive(Debug, Clone)]
pub struct EconomicsIntegration {
    economics_url: String,
    client: Client,
    enabled: bool,
}

#[derive(Debug, Serialize, Deserialize)]
pub struct SkillInvocationRequest {
    pub user_id: String,
    pub skill_id: String,
    pub amount: String,
}

#[derive(Debug, Serialize, Deserialize)]
pub struct LLMRegistrationRequest {
    pub user_id: String,
    pub llm_id: String,
    pub registration_fee: String,
}

#[derive(Debug, Serialize, Deserialize)]
pub struct EconomicsResponse {
    pub success: bool,
    pub data: Option<serde_json::Value>,
    pub error: Option<String>,
}

#[derive(Debug, Serialize, Deserialize)]
pub struct SkillExecutionEvent {
    pub user_id: String,
    pub skill_id: String,
    pub cost: String,
    pub success: bool,
    pub timestamp: u64,
}

#[derive(Debug, Serialize, Deserialize)]
pub struct LLMValidationEvent {
    pub validator_id: String,
    pub llm_id: String,
    pub is_valid: bool,
    pub timestamp: u64,
}

impl EconomicsIntegration {
    pub fn new(economics_url: String) -> Self {
        Self {
            economics_url,
            client: Client::new(),
            enabled: true,
        }
    }

    pub fn from_env() -> Self {
        let economics_url = std::env::var("ECONOMICS_SERVICE_URL")
            .unwrap_or_else(|_| "http://localhost:8090".to_string());
        
        Self::new(economics_url)
    }

    pub async fn process_skill_invocation(&self, user_id: &str, skill_id: &str, amount: &str) -> Result<String, Box<dyn std::error::Error>> {
        if !self.enabled {
            return Ok("economics_disabled".to_string());
        }

        let request = SkillInvocationRequest {
            user_id: user_id.to_string(),
            skill_id: skill_id.to_string(),
            amount: amount.to_string(),
        };

        let url = format!("{}/economics/skill/invoke", self.economics_url);
        
        let response = self.client
            .post(&url)
            .json(&request)
            .timeout(Duration::from_secs(30))
            .send()
            .await?;

        if response.status().is_success() {
            let economics_response: EconomicsResponse = response.json().await?;
            if economics_response.success {
                info!("Successfully processed skill invocation for user {} skill {}", user_id, skill_id);
                Ok(economics_response.data
                    .and_then(|d| d.get("transaction_id"))
                    .and_then(|id| id.as_str())
                    .unwrap_or("unknown")
                    .to_string())
            } else {
                error!("Economics service returned error: {:?}", economics_response.error);
                Err(economics_response.error.unwrap_or("Unknown error".to_string()).into())
            }
        } else {
            error!("Economics service returned status: {}", response.status());
            Err(format!("HTTP error: {}", response.status()).into())
        }
    }

    pub async fn process_llm_registration(&self, user_id: &str, llm_id: &str, registration_fee: &str) -> Result<String, Box<dyn std::error::Error>> {
        if !self.enabled {
            return Ok("economics_disabled".to_string());
        }

        let request = LLMRegistrationRequest {
            user_id: user_id.to_string(),
            llm_id: llm_id.to_string(),
            registration_fee: registration_fee.to_string(),
        };

        let url = format!("{}/economics/llm/register", self.economics_url);
        
        let response = self.client
            .post(&url)
            .json(&request)
            .timeout(Duration::from_secs(30))
            .send()
            .await?;

        if response.status().is_success() {
            let economics_response: EconomicsResponse = response.json().await?;
            if economics_response.success {
                info!("Successfully processed LLM registration for user {} LLM {}", user_id, llm_id);
                Ok(economics_response.data
                    .and_then(|d| d.get("transaction_id"))
                    .and_then(|id| id.as_str())
                    .unwrap_or("unknown")
                    .to_string())
            } else {
                error!("Economics service returned error: {:?}", economics_response.error);
                Err(economics_response.error.unwrap_or("Unknown error".to_string()).into())
            }
        } else {
            error!("Economics service returned status: {}", response.status());
            Err(format!("HTTP error: {}", response.status()).into())
        }
    }

    pub async fn send_skill_execution_event(&self, event: SkillExecutionEvent) -> Result<(), Box<dyn std::error::Error>> {
        if !self.enabled {
            return Ok(());
        }

        // In a real implementation, this would send the event to the economics service
        // For now, we'll just log it
        info!("Would send skill execution event: {:?}", event);
        Ok(())
    }

    pub async fn send_llm_validation_event(&self, event: LLMValidationEvent) -> Result<(), Box<dyn std::error::Error>> {
        if !self.enabled {
            return Ok(());
        }

        // In a real implementation, this would send the event to the economics service
        // For now, we'll just log it
        info!("Would send LLM validation event: {:?}", event);
        Ok(())
    }

    pub async fn get_economic_metrics(&self) -> Result<serde_json::Value, Box<dyn std::error::Error>> {
        if !self.enabled {
            return Ok(serde_json::json!({"status": "disabled"}));
        }

        let url = format!("{}/economics/metrics", self.economics_url);
        
        let response = self.client
            .get(&url)
            .timeout(Duration::from_secs(15))
            .send()
            .await?;

        if response.status().is_success() {
            let economics_response: EconomicsResponse = response.json().await?;
            if economics_response.success {
                Ok(economics_response.data.unwrap_or(serde_json::json!({})))
            } else {
                Err(economics_response.error.unwrap_or("Unknown error".to_string()).into())
            }
        } else {
            Err(format!("HTTP error: {}", response.status()).into())
        }
    }

    pub async fn health_check(&self) -> bool {
        if !self.enabled {
            return true; // Consider disabled as healthy
        }

        let url = format!("{}/economics/health", self.economics_url);
        
        match self.client
            .get(&url)
            .timeout(Duration::from_secs(5))
            .send()
            .await
        {
            Ok(response) => response.status().is_success(),
            Err(_) => false,
        }
    }

    pub fn disable(&mut self) {
        self.enabled = false;
        warn!("Economics integration disabled");
    }

    pub fn enable(&mut self) {
        self.enabled = true;
        info!("Economics integration enabled");
    }

    pub fn is_enabled(&self) -> bool {
        self.enabled
    }

    // Start a background task to periodically sync with economics service
    pub async fn start_background_sync(&self) {
        if !self.enabled {
            return;
        }

        let economics_integration = self.clone();
        tokio::spawn(async move {
            let mut interval = interval(Duration::from_secs(60)); // Sync every minute
            
            loop {
                interval.tick().await;
                
                // Perform health check
                if !economics_integration.health_check().await {
                    warn!("Economics service health check failed");
                    continue;
                }

                // Get and log metrics
                match economics_integration.get_economic_metrics().await {
                    Ok(metrics) => {
                        info!("Economics metrics sync successful");
                        // In a real implementation, you might store these metrics locally
                    }
                    Err(e) => {
                        error!("Failed to sync economics metrics: {}", e);
                    }
                }
            }
        });
    }
}

// Helper functions for integration with existing KNIRVCHAIN code

pub async fn integrate_skill_execution(
    economics: &EconomicsIntegration,
    user_id: &str,
    skill_id: &str,
    cost: &str,
    success: bool,
) -> Result<(), Box<dyn std::error::Error>> {
    // Process the economic transaction
    if success {
        economics.process_skill_invocation(user_id, skill_id, cost).await?;
    }

    // Send event for tracking
    let event = SkillExecutionEvent {
        user_id: user_id.to_string(),
        skill_id: skill_id.to_string(),
        cost: cost.to_string(),
        success,
        timestamp: std::time::SystemTime::now()
            .duration_since(std::time::UNIX_EPOCH)
            .unwrap()
            .as_secs(),
    };

    economics.send_skill_execution_event(event).await?;
    Ok(())
}

pub async fn integrate_llm_registration(
    economics: &EconomicsIntegration,
    user_id: &str,
    llm_id: &str,
    registration_fee: &str,
) -> Result<String, Box<dyn std::error::Error>> {
    // Process the registration fee
    let tx_id = economics.process_llm_registration(user_id, llm_id, registration_fee).await?;

    // Send validation event (assuming registration is successful)
    let event = LLMValidationEvent {
        validator_id: "system".to_string(), // System validation
        llm_id: llm_id.to_string(),
        is_valid: true,
        timestamp: std::time::SystemTime::now()
            .duration_since(std::time::UNIX_EPOCH)
            .unwrap()
            .as_secs(),
    };

    economics.send_llm_validation_event(event).await?;
    Ok(tx_id)
}
