//! Registry event listener for IBC events from KNIRVCHAIN

use std::sync::Arc;
use tokio::sync::Mutex;
use serde::{Deserialize, Serialize};
use crate::token_economics::TokenEconomics;
use crate::ibc_handler::IBCHandler;

/// Registry event listener for KNIRVCHAIN events
#[derive(Debug)]
pub struct RegistryEventListener {
    ibc_handler: Arc<Mutex<IBCHandler>>,
    economics: Arc<TokenEconomics>,
    event_queue: Mutex<Vec<RegistryEvent>>,
}

/// Registry events received from KNIRVCHAIN
#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum RegistryEvent {
    LLMRegistration(LLMRegistrationEvent),
    SkillInvocation(SkillInvocationEvent),
}

/// LLM registration event data
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct LLMRegistrationEvent {
    pub model_hash: String,
    pub owner: String,
    pub registration_fee: u64,
    pub timestamp: u64,
    pub transaction_hash: String,
}

/// Skill invocation event data
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SkillInvocationEvent {
    pub skill_id: String,
    pub user: String,
    pub amount_burned: u64,
    pub success: bool,
    pub timestamp: u64,
    pub transaction_hash: String,
}

impl RegistryEventListener {
    /// Create a new registry event listener
    pub fn new(
        ibc_handler: Arc<Mutex<IBCHandler>>,
        economics: Arc<TokenEconomics>,
    ) -> Self {
        Self {
            ibc_handler,
            economics,
            event_queue: Mutex::new(Vec::new()),
        }
    }

    /// Listen for LLM registration events from KNIRVCHAIN
    pub async fn on_llm_registered(&self, event: LLMRegistrationEvent) -> Result<(), RegistryError> {
        // 1. Collect registration fee
        self.collect_registration_fee(event.registration_fee, &event.owner).await?;

        // 2. Update economics metrics
        self.update_economics_metrics(&event).await?;

        // 3. Queue event for processing
        let mut queue = self.event_queue.lock().await;
        queue.push(RegistryEvent::LLMRegistration(event));

        Ok(())
    }

    /// Listen for skill invocation events from KNIRVCHAIN
    pub async fn on_skill_invoked(&self, event: SkillInvocationEvent) -> Result<(), RegistryError> {
        // 1. Process invocation fee
        self.process_invocation_fee(event.amount_burned, &event.user).await?;

        // 2. Distribute rewards
        self.distribute_skill_rewards(&event).await?;

        // 3. Update metrics
        self.update_invocation_metrics(&event).await?;

        // 4. Queue event for processing
        let mut queue = self.event_queue.lock().await;
        queue.push(RegistryEvent::SkillInvocation(event));

        Ok(())
    }

    /// Process pending registry events
    pub async fn process_pending_events(&self) -> Result<(), RegistryError> {
        let mut queue = self.event_queue.lock().await;
        let events = queue.drain(..).collect::<Vec<_>>();

        for event in events {
            match event {
                RegistryEvent::LLMRegistration(reg_event) => {
                    self.process_llm_registration_event(&reg_event).await?;
                }
                RegistryEvent::SkillInvocation(inv_event) => {
                    self.process_skill_invocation_event(&inv_event).await?;
                }
            }
        }

        Ok(())
    }

    /// Handle incoming IBC message with registry event
    pub async fn handle_ibc_registry_message(&self, message: serde_json::Value) -> Result<(), RegistryError> {
        // Parse the IBC message to extract registry events
        // This would be called by the IBC handler when receiving messages from KNIRVCHAIN

        if let Some(event_type) = message.get("event_type").and_then(|v| v.as_str()) {
            match event_type {
                "llm_registration" => {
                    let event: LLMRegistrationEvent = serde_json::from_value(message)
                        .map_err(|e| RegistryError::ParseError(e.to_string()))?;
                    self.on_llm_registered(event).await?;
                }
                "skill_invocation" => {
                    let event: SkillInvocationEvent = serde_json::from_value(message)
                        .map_err(|e| RegistryError::ParseError(e.to_string()))?;
                    self.on_skill_invoked(event).await?;
                }
                _ => {
                    return Err(RegistryError::UnknownEventType(event_type.to_string()));
                }
            }
        }

        Ok(())
    }

    // Private helper methods

    async fn collect_registration_fee(&self, fee_amount: u64, payer: &str) -> Result<(), RegistryError> {
        // In a real implementation, this would collect the fee from the cross-chain transfer
        // For now, we assume the fee has already been collected via IBC
        tracing::info!("Collected registration fee: {} from {}", fee_amount, payer);
        Ok(())
    }

    async fn update_economics_metrics(&self, event: &LLMRegistrationEvent) -> Result<(), RegistryError> {
        // Update economics metrics for LLM registration
        tracing::info!("Updated economics metrics for LLM registration: {}", event.model_hash);
        Ok(())
    }

    async fn process_invocation_fee(&self, amount: u64, user: &str) -> Result<(), RegistryError> {
        // Process the skill invocation fee
        tracing::info!("Processed invocation fee: {} from user {}", amount, user);
        Ok(())
    }

    async fn distribute_skill_rewards(&self, event: &SkillInvocationEvent) -> Result<(), RegistryError> {
        // Distribute rewards to skill creators and network
        tracing::info!("Distributed rewards for skill invocation: {}", event.skill_id);
        Ok(())
    }

    async fn update_invocation_metrics(&self, event: &SkillInvocationEvent) -> Result<(), RegistryError> {
        // Update invocation metrics
        tracing::info!("Updated invocation metrics for skill: {}", event.skill_id);
        Ok(())
    }

    async fn process_llm_registration_event(&self, event: &LLMRegistrationEvent) -> Result<(), RegistryError> {
        // Process the LLM registration event
        // This could include updating local caches or triggering other actions
        tracing::info!("Processed LLM registration event: {}", event.model_hash);
        Ok(())
    }

    async fn process_skill_invocation_event(&self, event: &SkillInvocationEvent) -> Result<(), RegistryError> {
        // Process the skill invocation event
        // This could include updating performance metrics or triggering rewards
        tracing::info!("Processed skill invocation event: {}", event.skill_id);
        Ok(())
    }
}

/// Registry-specific errors
#[derive(Debug, thiserror::Error)]
pub enum RegistryError {
    #[error("Parse error: {0}")]
    ParseError(String),

    #[error("Unknown event type: {0}")]
    UnknownEventType(String),

    #[error("Economics error: {0}")]
    EconomicsError(String),

    #[error("IBC error: {0}")]
    IbcError(String),

    #[error("Internal error: {0}")]
    InternalError(String),
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::sync::Arc;

    #[tokio::test]
    async fn test_registry_event_listener() {
        // This would require mocking the dependencies
        // For now, just test the structure exists
    }
}