use serde::Serialize;
use std::sync::Arc;
use num_bigint::BigInt;
use crate::token_economics::TokenEconomics;
use crate::registry_listener::{RegistryEventListener, LLMRegistrationEvent, SkillInvocationEvent};
use tracing::info;

#[derive(Debug, Clone)]
pub struct EconomicsIntegration {
    token_economics: Arc<TokenEconomics>,
    enabled: bool,
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
    pub fn new(token_economics: Arc<TokenEconomics>) -> Self {
        Self {
            token_economics,
            enabled: true,
        }
    }

    pub fn from_env(token_economics: Arc<TokenEconomics>) -> Self {
        Self::new(token_economics)
    }

    pub fn process_skill_invocation(&self, user_id: &str, skill_id: &str, amount: &BigInt) -> Result<String, Box<dyn std::error::Error>> {
        if !self.enabled {
            return Ok("economics_disabled".to_string());
        }

        let tx = self.token_economics.process_skill_invocation(user_id, skill_id, amount)?;
        info!("Successfully processed skill invocation for user {} skill {}", user_id, skill_id);
        Ok(tx.id)
    }

    pub fn process_llm_registration(&self, user_id: &str, llm_id: &str, registration_fee: &BigInt) -> Result<String, Box<dyn std::error::Error>> {
        if !self.enabled {
            return Ok("economics_disabled".to_string());
        }

        let tx = self.token_economics.process_llm_registration(user_id, llm_id, registration_fee)?;
        info!("Successfully processed LLM registration for user {} LLM {}", user_id, llm_id);
        Ok(tx.id)
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

    pub fn get_economic_metrics(&self) -> Result<serde_json::Value, Box<dyn std::error::Error>> {
        if !self.enabled {
            return Ok(serde_json::json!({"status": "disabled"}));
        }

        let metrics = self.token_economics.get_economic_metrics()?;
        Ok(serde_json::to_value(metrics)?)
    }

    pub fn health_check(&self) -> bool {
        if !self.enabled {
            return true; // Consider disabled as healthy
        }

        // For local economics, health check is always true if enabled
        true
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

    // Start a background task for local economics (no sync needed)
    pub async fn start_background_sync(&self) {
        if !self.enabled {
            return;
        }

        // Local economics doesn't need background sync
        info!("Local economics integration started");
    }

    /// Handle LLM registration event from KNIRVCHAIN
    pub async fn handle_llm_registration_event(&self, event: LLMRegistrationEvent) -> Result<(), Box<dyn std::error::Error>> {
        if !self.enabled {
            return Ok(());
        }

        // Process the registration fee that was already collected via IBC
        info!("Processing LLM registration event: {} for user {} with fee {}",
              event.model_hash, event.owner, event.registration_fee);

        // Update economics metrics for LLM registration
        // In a real implementation, this would update metrics and potentially distribute rewards
        Ok(())
    }

    /// Handle skill invocation event from KNIRVCHAIN
    pub async fn handle_skill_invocation_event(&self, event: SkillInvocationEvent) -> Result<(), Box<dyn std::error::Error>> {
        if !self.enabled {
            return Ok(());
        }

        // Process the skill invocation fee that was already collected via IBC
        info!("Processing skill invocation event: {} for user {} with fee {} (success: {})",
              event.skill_id, event.user, event.amount_burned, event.success);

        // Update economics metrics for skill invocation
        // In a real implementation, this would update metrics and distribute creator rewards
        Ok(())
    }

    /// Integrate with RegistryEventListener to receive events from KNIRVCHAIN
    pub async fn setup_registry_event_listener(&self, registry_listener: Arc<RegistryEventListener>) -> Result<(), Box<dyn std::error::Error>> {
        // In a real implementation, this would register callbacks with the registry listener
        // For now, we just log that we're setting up the integration
        info!("Setting up economics integration with registry event listener");
        Ok(())
    }
}

// Helper functions for integration with existing KNIRVORACLE code

pub fn integrate_skill_execution(
    economics: &EconomicsIntegration,
    user_id: &str,
    skill_id: &str,
    cost: &BigInt,
    success: bool,
) -> Result<(), Box<dyn std::error::Error>> {
    // Process the economic transaction
    if success {
        economics.process_skill_invocation(user_id, skill_id, cost)?;
    }

    // Send event for tracking (placeholder)
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

    economics.send_skill_execution_event(event)?;
    Ok(())
}

pub fn integrate_llm_registration(
    economics: &EconomicsIntegration,
    user_id: &str,
    llm_id: &str,
    registration_fee: &BigInt,
) -> Result<String, Box<dyn std::error::Error>> {
    // Process the registration fee
    let tx_id = economics.process_llm_registration(user_id, llm_id, registration_fee)?;

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

    economics.send_llm_validation_event(event)?;
    Ok(tx_id)
}
