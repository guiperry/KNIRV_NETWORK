use anyhow::{anyhow, Result};
use serde::{Deserialize, Serialize};
use sha2::Digest;
use std::collections::HashMap;
use std::sync::Arc;
use std::time::{SystemTime, UNIX_EPOCH};
use tokio::sync::Mutex;
use tracing::info;

use crate::ipfs_client::IpfsClient;
use crate::multi_model_engine::{
    GovernanceStatus, ModelMetadata, ModelPerformanceMetrics, ModelType,
};

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ModelVersion {
    pub version: String,
    pub model_hash: String,
    pub ipfs_hash: Option<String>,
    pub created_at: u64,
    pub deprecated_at: Option<u64>,
    pub performance_metrics: ModelPerformanceMetrics,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ModelTransitionProposal {
    pub proposal_id: String,
    pub current_model: Option<String>,
    pub proposed_model: String,
    pub model_type: ModelType,
    pub validation_proof: ValidationProof,
    pub compatibility_assessment: CompatibilityAssessment,
    pub timestamp: u64,
    pub proposer: String,
    pub voting_deadline: u64,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ValidationProof {
    pub proof_id: String,
    pub nexus_dve_address: String,
    pub model_hash: String,
    pub validation_results: ValidationResults,
    pub cryptographic_proof: String,
    pub timestamp: u64,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ValidationResults {
    pub accuracy_score: f64,
    pub performance_benchmarks: HashMap<String, f64>,
    pub security_assessment: SecurityAssessment,
    pub compatibility_tests: Vec<CompatibilityTest>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SecurityAssessment {
    pub vulnerability_scan: bool,
    pub code_audit: bool,
    pub dependency_check: bool,
    pub risk_level: RiskLevel,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum RiskLevel {
    Low,
    Medium,
    High,
    Critical,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CompatibilityTest {
    pub test_name: String,
    pub passed: bool,
    pub score: f64,
    pub details: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CompatibilityAssessment {
    pub backward_compatibility: bool,
    pub api_compatibility: bool,
    pub performance_impact: f64,
    pub migration_complexity: MigrationComplexity,
    pub estimated_downtime: u64, // seconds
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum MigrationComplexity {
    Trivial,
    Simple,
    Moderate,
    Complex,
    Critical,
}

#[derive(Debug)]
#[allow(dead_code)]
pub struct EnhancedMultiModelRegistry {
    models: Arc<Mutex<HashMap<String, ModelMetadata>>>,
    active_model: Arc<Mutex<Option<String>>>,
    model_type: Arc<Mutex<ModelType>>,
    version_history: Arc<Mutex<Vec<ModelVersion>>>,
    ipfs_client: Arc<IpfsClient>,
    pending_proposals: Arc<Mutex<HashMap<String, ModelTransitionProposal>>>,
    approved_proposals: Arc<Mutex<HashMap<String, ModelTransitionProposal>>>,
}

#[allow(dead_code)]
impl EnhancedMultiModelRegistry {
    pub fn new(ipfs_client: Arc<IpfsClient>) -> Self {
        Self {
            models: Arc::new(Mutex::new(HashMap::new())),
            active_model: Arc::new(Mutex::new(None)),
            model_type: Arc::new(Mutex::new(ModelType::CodeT5(
                crate::multi_model_engine::CodeT5Config {
                    model_size: "base".to_string(),
                    device: "cpu".to_string(),
                    max_length: 512,
                },
            ))),
            version_history: Arc::new(Mutex::new(Vec::new())),
            ipfs_client,
            pending_proposals: Arc::new(Mutex::new(HashMap::new())),
            approved_proposals: Arc::new(Mutex::new(HashMap::new())),
        }
    }

    /// Register a new model
    pub async fn register_model(&self, metadata: ModelMetadata) -> Result<String> {
        let model_hash = metadata.model_hash.clone();

        // Store model metadata
        let mut models = self.models.lock().await;
        models.insert(model_hash.clone(), metadata.clone());

        // Add to version history
        let mut history = self.version_history.lock().await;
        history.push(ModelVersion {
            version: metadata.version.clone(),
            model_hash: model_hash.clone(),
            ipfs_hash: metadata.ipfs_hash.clone(),
            created_at: SystemTime::now().duration_since(UNIX_EPOCH)?.as_secs(),
            deprecated_at: None,
            performance_metrics: metadata.performance_metrics.clone(),
        });

        info!(
            "Registered model: {} (type: {:?})",
            model_hash, metadata.model_type
        );
        Ok(model_hash)
    }

    /// Propose a model transition
    pub async fn propose_model_transition(
        &self,
        new_model_type: ModelType,
        new_model_data: Option<&[u8]>,
        validation_proof: ValidationProof,
        proposer: String,
    ) -> Result<String> {
        // Store model in IPFS if provided
        let model_hash = if let Some(data) = new_model_data {
            self.ipfs_client.store_model(data).await?
        } else {
            // For cloud models, generate metadata hash
            self.generate_cloud_model_hash(&new_model_type)?
        };

        // Verify validation proof from KNIRVNEXUS
        self.verify_nexus_validation_proof(&validation_proof)
            .await?;

        // Assess compatibility
        let compatibility_assessment = self.assess_compatibility(&new_model_type).await?;

        // Create proposal
        let proposal_id = format!("proposal_{}", uuid::Uuid::new_v4());
        let current_model = self.active_model.lock().await.clone();

        let proposal = ModelTransitionProposal {
            proposal_id: proposal_id.clone(),
            current_model,
            proposed_model: model_hash.clone(),
            model_type: new_model_type,
            validation_proof,
            compatibility_assessment,
            timestamp: SystemTime::now().duration_since(UNIX_EPOCH)?.as_secs(),
            proposer,
            voting_deadline: SystemTime::now().duration_since(UNIX_EPOCH)?.as_secs()
                + 7 * 24 * 3600, // 7 days
        };

        // Store proposal
        let mut proposals = self.pending_proposals.lock().await;
        proposals.insert(proposal_id.clone(), proposal);

        info!("Created model transition proposal: {}", proposal_id);
        Ok(proposal_id)
    }

    /// Execute an approved model transition
    pub async fn execute_approved_transition(&self, proposal_id: &str) -> Result<()> {
        // Move proposal from pending to approved
        let proposal = {
            let mut pending = self.pending_proposals.lock().await;
            let proposal = pending
                .remove(proposal_id)
                .ok_or_else(|| anyhow!("Proposal not found: {}", proposal_id))?;

            let mut approved = self.approved_proposals.lock().await;
            approved.insert(proposal_id.to_string(), proposal.clone());
            proposal
        };

        // Update active model
        let mut active_model = self.active_model.lock().await;
        *active_model = Some(proposal.proposed_model.clone());

        // Update model type
        let mut model_type = self.model_type.lock().await;
        *model_type = proposal.model_type.clone();

        // Deprecate old model if exists
        if let Some(old_model_hash) = &proposal.current_model {
            self.deprecate_model(old_model_hash).await?;
        }

        info!(
            "Executed model transition: {} -> {}",
            proposal.current_model.unwrap_or("none".to_string()),
            proposal.proposed_model
        );
        Ok(())
    }

    /// Verify validation proof from KNIRVNEXUS
    async fn verify_nexus_validation_proof(&self, proof: &ValidationProof) -> Result<()> {
        // TODO: Implement actual verification with KNIRVNEXUS
        // For now, just validate the structure
        if proof.proof_id.is_empty() || proof.nexus_dve_address.is_empty() {
            return Err(anyhow!("Invalid validation proof structure"));
        }

        // Check if proof is recent (within 24 hours)
        let current_time = SystemTime::now().duration_since(UNIX_EPOCH)?.as_secs();
        if current_time - proof.timestamp > 24 * 3600 {
            return Err(anyhow!("Validation proof is too old"));
        }

        info!("Validation proof verified: {}", proof.proof_id);
        Ok(())
    }

    /// Assess compatibility of new model type
    async fn assess_compatibility(
        &self,
        new_model_type: &ModelType,
    ) -> Result<CompatibilityAssessment> {
        let current_type = self.model_type.lock().await.clone();

        let backward_compatibility = match (&current_type, new_model_type) {
            (ModelType::CodeT5(_), ModelType::CodeT5(_)) => true,
            (ModelType::Deepseek(_), ModelType::Deepseek(_)) => true,
            (ModelType::Gemini(_), ModelType::Gemini(_)) => true,
            _ => false, // Different model types require more careful migration
        };

        let api_compatibility = backward_compatibility; // Simplified for now

        let performance_impact = if backward_compatibility { 0.1 } else { 0.5 };

        let migration_complexity = if backward_compatibility {
            MigrationComplexity::Simple
        } else {
            MigrationComplexity::Moderate
        };

        let estimated_downtime = if backward_compatibility { 30 } else { 300 }; // seconds

        Ok(CompatibilityAssessment {
            backward_compatibility,
            api_compatibility,
            performance_impact,
            migration_complexity,
            estimated_downtime,
        })
    }

    /// Generate hash for cloud models
    fn generate_cloud_model_hash(&self, model_type: &ModelType) -> Result<String> {
        let model_string = serde_json::to_string(model_type)?;
        let hash = sha2::Sha256::digest(model_string.as_bytes());
        Ok(hex::encode(hash))
    }

    /// Deprecate an old model
    async fn deprecate_model(&self, model_hash: &str) -> Result<()> {
        let mut models = self.models.lock().await;
        if let Some(metadata) = models.get_mut(model_hash) {
            metadata.governance_status = GovernanceStatus::Deprecated;
        }

        let mut history = self.version_history.lock().await;
        for version in history.iter_mut() {
            if version.model_hash == model_hash && version.deprecated_at.is_none() {
                version.deprecated_at =
                    Some(SystemTime::now().duration_since(UNIX_EPOCH)?.as_secs());
                break;
            }
        }

        info!("Deprecated model: {}", model_hash);
        Ok(())
    }

    /// Get active model hash
    pub async fn get_active_model(&self) -> Option<String> {
        self.active_model.lock().await.clone()
    }

    /// Get active model type
    pub async fn get_active_model_type(&self) -> ModelType {
        self.model_type.lock().await.clone()
    }

    /// List all models
    pub async fn list_models(&self) -> Vec<ModelMetadata> {
        let models = self.models.lock().await;
        models.values().cloned().collect()
    }

    /// Get model by hash
    pub async fn get_model(&self, model_hash: &str) -> Option<ModelMetadata> {
        let models = self.models.lock().await;
        models.get(model_hash).cloned()
    }

    /// List pending proposals
    pub async fn list_pending_proposals(&self) -> Vec<ModelTransitionProposal> {
        let proposals = self.pending_proposals.lock().await;
        proposals.values().cloned().collect()
    }

    /// Get proposal by ID
    pub async fn get_proposal(&self, proposal_id: &str) -> Option<ModelTransitionProposal> {
        let proposals = self.pending_proposals.lock().await;
        proposals.get(proposal_id).cloned()
    }

    /// Get version history
    pub async fn get_version_history(&self) -> Vec<ModelVersion> {
        let history = self.version_history.lock().await;
        history.clone()
    }

    /// Update model performance metrics
    pub async fn update_performance_metrics(
        &self,
        model_hash: &str,
        metrics: ModelPerformanceMetrics,
    ) -> Result<()> {
        let mut models = self.models.lock().await;
        if let Some(metadata) = models.get_mut(model_hash) {
            metadata.performance_metrics = metrics;
            info!("Updated performance metrics for model: {}", model_hash);
            Ok(())
        } else {
            Err(anyhow!("Model not found: {}", model_hash))
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::ipfs_client::IpfsClient;

    #[tokio::test]
    async fn test_model_registration() {
        let ipfs_client = Arc::new(IpfsClient::new_mock());
        let registry = EnhancedMultiModelRegistry::new(ipfs_client);

        let metadata = ModelMetadata {
            model_type: ModelType::CodeT5(crate::multi_model_engine::CodeT5Config {
                model_size: "base".to_string(),
                device: "cpu".to_string(),
                max_length: 512,
            }),
            model_hash: "test_hash".to_string(),
            ipfs_hash: None,
            version: "1.0".to_string(),
            capabilities: vec!["code_generation".to_string()],
            performance_metrics: ModelPerformanceMetrics {
                accuracy: 0.95,
                latency_ms: 100,
                throughput_tokens_per_sec: 50,
                memory_usage_mb: 1024,
                total_inferences: 0,
                success_rate: 1.0,
            },
            governance_status: GovernanceStatus::Pending,
            loaded_at: None,
        };

        let result = registry.register_model(metadata).await;
        assert!(result.is_ok());
        assert_eq!(result.unwrap(), "test_hash");

        let models = registry.list_models().await;
        assert_eq!(models.len(), 1);
        assert_eq!(models[0].model_hash, "test_hash");
    }
}
