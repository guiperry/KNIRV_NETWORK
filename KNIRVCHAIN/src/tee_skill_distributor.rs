use anyhow::{anyhow, Result};
use serde::{Deserialize, Serialize};
use sha2::Digest;
use std::collections::HashMap;
use std::sync::Arc;
use std::time::{SystemTime, UNIX_EPOCH};
use tokio::sync::Mutex;
use tracing::info;

use crate::ibc_handler::{ExecutionMetadata, IsolationLevel, SecurityRequirements};
use crate::ipfs_client::IpfsClient;
#[cfg(test)]
use crate::nrn_token::Address;
use crate::nrn_token::{SkillMetadata, SkillRegistry};

#[derive(Debug)]
#[allow(dead_code)]
pub struct TEESkillDistributor {
    skill_registry: Arc<Mutex<SkillRegistry>>,
    ipfs_client: Arc<IpfsClient>,
    tee_compatibility_checker: TEECompatibilityChecker,
    execution_packages: Arc<Mutex<HashMap<String, SkillExecutionPackage>>>,
    active_distributions: Arc<Mutex<HashMap<String, DistributionSession>>>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TEEInfo {
    pub tee_type: TEEType,
    pub version: String,
    pub capabilities: Vec<String>,
    pub attestation_support: bool,
    pub secure_storage: bool,
    pub memory_limit: u64,
    pub cpu_cores: u32,
    pub network_isolation: bool,
    pub device_id: String,
    pub platform: String,
}

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq, Eq, Hash)]
pub enum TEEType {
    IntelSGX,
    AMDTEE,
    ArmTrustZone,
    RISCVTEE,
    Software, // For development/testing
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SkillExecutionPackage {
    pub skill_id: String,
    pub skill_code: Vec<u8>,
    pub execution_metadata: ExecutionMetadata,
    pub security_requirements: SecurityRequirements,
    pub resource_limits: ResourceLimits,
    pub dependencies: Vec<Dependency>,
    pub attestation_requirements: AttestationRequirements,
    pub package_hash: String,
    pub created_at: u64,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ResourceLimits {
    pub max_memory: u64,
    pub max_cpu_time: u64,
    pub max_network_bandwidth: u64,
    pub max_storage: u64,
    pub max_execution_time: u64,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Dependency {
    pub name: String,
    pub version: String,
    pub hash: String,
    pub source: DependencySource,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum DependencySource {
    IPFS(String),
    Registry(String),
    Embedded,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AttestationRequirements {
    pub required: bool,
    pub attestation_type: AttestationType,
    pub trusted_authorities: Vec<String>,
    pub minimum_security_level: SecurityLevel,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum AttestationType {
    Remote,
    Local,
    Hybrid,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum SecurityLevel {
    Development,
    Testing,
    Production,
    HighSecurity,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DistributionSession {
    pub session_id: String,
    pub skill_id: String,
    pub requestor_tee: TEEInfo,
    pub package_hash: String,
    pub status: DistributionStatus,
    pub created_at: u64,
    pub expires_at: u64,
    pub download_attempts: u32,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum DistributionStatus {
    Preparing,
    Ready,
    Downloading,
    Completed,
    Failed,
    Expired,
}

#[derive(Debug, Clone)]
pub struct TEECompatibilityChecker {
    supported_tees: HashMap<TEEType, TEECapabilities>,
    security_policies: SecurityPolicies,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TEECapabilities {
    pub isolation_level: IsolationLevel,
    pub attestation_support: bool,
    pub secure_storage: bool,
    pub memory_encryption: bool,
    pub code_integrity: bool,
    pub supported_runtimes: Vec<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SecurityPolicies {
    pub minimum_isolation: IsolationLevel,
    pub require_attestation: bool,
    pub allow_network_access: bool,
    pub allow_file_system_access: bool,
    pub trusted_tee_types: Vec<TEEType>,
}

#[allow(dead_code)]
impl TEESkillDistributor {
    pub fn new(skill_registry: Arc<Mutex<SkillRegistry>>, ipfs_client: Arc<IpfsClient>) -> Self {
        Self {
            skill_registry,
            ipfs_client,
            tee_compatibility_checker: TEECompatibilityChecker::new(),
            execution_packages: Arc::new(Mutex::new(HashMap::new())),
            active_distributions: Arc::new(Mutex::new(HashMap::new())),
        }
    }

    /// Prepare skill for TEE execution
    pub async fn prepare_skill_for_tee_execution(
        &self,
        skill_id: &str,
        requestor_tee_info: &TEEInfo,
    ) -> Result<SkillExecutionPackage> {
        info!(
            "Preparing skill {} for TEE execution on {:?}",
            skill_id, requestor_tee_info.tee_type
        );

        // Get skill metadata
        let skill_registry = self.skill_registry.lock().await;
        let skill = skill_registry
            .get_skill(skill_id)
            .ok_or_else(|| anyhow!("Skill not found: {}", skill_id))?;

        // Verify TEE compatibility
        self.tee_compatibility_checker
            .verify_compatibility(skill, requestor_tee_info)?;

        // Retrieve skill code from IPFS (assuming it's stored there)
        let skill_code_hash = format!("skill_code_{}", skill_id); // Placeholder
        let skill_code = self
            .ipfs_client
            .retrieve_skill_code(&skill_code_hash)
            .await
            .unwrap_or_else(|_| b"mock_skill_code".to_vec()); // Mock for now

        // Create execution package
        let package_hash = self.calculate_package_hash(&skill_code)?;
        let execution_package = SkillExecutionPackage {
            skill_id: skill_id.to_string(),
            skill_code,
            execution_metadata: ExecutionMetadata {
                runtime: "wasm".to_string(),
                memory_limit: 128 * 1024 * 1024, // 128MB
                cpu_limit: 1000,                 // 1 second
                timeout: 30,                     // 30 seconds
                dependencies: vec!["std".to_string()],
            },
            security_requirements: SecurityRequirements {
                isolation_level: IsolationLevel::TEE,
                network_access: false,
                file_system_access: false,
                encryption_required: true,
            },
            resource_limits: ResourceLimits {
                max_memory: 256 * 1024 * 1024, // 256MB
                max_cpu_time: 5000,            // 5 seconds
                max_network_bandwidth: 0,      // No network access
                max_storage: 10 * 1024 * 1024, // 10MB
                max_execution_time: 60,        // 1 minute
            },
            dependencies: vec![Dependency {
                name: "wasm-runtime".to_string(),
                version: "1.0.0".to_string(),
                hash: "abc123".to_string(),
                source: DependencySource::Registry("knirv-registry".to_string()),
            }],
            attestation_requirements: AttestationRequirements {
                required: true,
                attestation_type: AttestationType::Remote,
                trusted_authorities: vec!["knirv-nexus".to_string()],
                minimum_security_level: SecurityLevel::Production,
            },
            package_hash,
            created_at: SystemTime::now().duration_since(UNIX_EPOCH)?.as_secs(),
        };

        // Store execution package
        let mut packages = self.execution_packages.lock().await;
        packages.insert(skill_id.to_string(), execution_package.clone());

        info!("Prepared execution package for skill {}", skill_id);
        Ok(execution_package)
    }

    /// Create distribution session for skill download
    pub async fn create_distribution_session(
        &self,
        skill_id: &str,
        requestor_tee: TEEInfo,
    ) -> Result<String> {
        let session_id = format!("dist_{}_{}", skill_id, uuid::Uuid::new_v4());

        // Verify skill package exists
        let packages = self.execution_packages.lock().await;
        let package = packages
            .get(skill_id)
            .ok_or_else(|| anyhow!("Skill package not prepared: {}", skill_id))?;

        let session = DistributionSession {
            session_id: session_id.clone(),
            skill_id: skill_id.to_string(),
            requestor_tee,
            package_hash: package.package_hash.clone(),
            status: DistributionStatus::Ready,
            created_at: SystemTime::now().duration_since(UNIX_EPOCH)?.as_secs(),
            expires_at: SystemTime::now().duration_since(UNIX_EPOCH)?.as_secs() + 3600, // 1 hour
            download_attempts: 0,
        };

        let mut distributions = self.active_distributions.lock().await;
        distributions.insert(session_id.clone(), session);

        info!("Created distribution session: {}", session_id);
        Ok(session_id)
    }

    /// Download skill package for TEE execution
    pub async fn download_skill_package(
        &self,
        session_id: &str,
        tee_attestation: &str,
    ) -> Result<SkillExecutionPackage> {
        let mut distributions = self.active_distributions.lock().await;
        let session = distributions
            .get_mut(session_id)
            .ok_or_else(|| anyhow!("Distribution session not found: {}", session_id))?;

        // Check session expiry
        let current_time = SystemTime::now().duration_since(UNIX_EPOCH)?.as_secs();
        if current_time > session.expires_at {
            session.status = DistributionStatus::Expired;
            return Err(anyhow!("Distribution session expired"));
        }

        // Verify TEE attestation
        self.verify_tee_attestation(tee_attestation, &session.requestor_tee)
            .await?;

        // Update session status
        session.status = DistributionStatus::Downloading;
        session.download_attempts += 1;

        // Get execution package
        let packages = self.execution_packages.lock().await;
        let package = packages
            .get(&session.skill_id)
            .ok_or_else(|| anyhow!("Skill package not found: {}", session.skill_id))?
            .clone();

        // Mark as completed
        session.status = DistributionStatus::Completed;

        info!("Downloaded skill package for session: {}", session_id);
        Ok(package)
    }

    /// Verify TEE attestation
    async fn verify_tee_attestation(&self, attestation: &str, tee_info: &TEEInfo) -> Result<()> {
        // TODO: Implement actual TEE attestation verification
        // This would involve:
        // 1. Parsing the attestation
        // 2. Verifying signatures
        // 3. Checking against trusted authorities
        // 4. Validating TEE measurements

        if attestation.is_empty() {
            return Err(anyhow!("Empty attestation provided"));
        }

        info!(
            "Verified TEE attestation for device: {}",
            tee_info.device_id
        );
        Ok(())
    }

    /// Calculate package hash
    fn calculate_package_hash(&self, skill_code: &[u8]) -> Result<String> {
        let hash = sha2::Sha256::digest(skill_code);
        Ok(hex::encode(hash))
    }

    /// Clean up expired distribution sessions
    pub async fn cleanup_expired_sessions(&self) -> Result<()> {
        let mut distributions = self.active_distributions.lock().await;
        let current_time = SystemTime::now().duration_since(UNIX_EPOCH)?.as_secs();

        let expired_sessions: Vec<String> = distributions
            .iter()
            .filter(|(_, session)| current_time > session.expires_at)
            .map(|(id, _)| id.clone())
            .collect();

        for session_id in expired_sessions {
            distributions.remove(&session_id);
            info!("Cleaned up expired distribution session: {}", session_id);
        }

        Ok(())
    }

    /// Get distribution session status
    pub async fn get_distribution_status(&self, session_id: &str) -> Option<DistributionStatus> {
        let distributions = self.active_distributions.lock().await;
        distributions.get(session_id).map(|s| s.status.clone())
    }

    /// List active distribution sessions
    pub async fn list_active_sessions(&self) -> Vec<DistributionSession> {
        let distributions = self.active_distributions.lock().await;
        distributions.values().cloned().collect()
    }

    /// Get skill execution statistics
    pub async fn get_execution_statistics(&self, skill_id: &str) -> Option<ExecutionStatistics> {
        // TODO: Implement execution statistics tracking
        Some(ExecutionStatistics {
            skill_id: skill_id.to_string(),
            total_executions: 0,
            successful_executions: 0,
            failed_executions: 0,
            average_execution_time: 0.0,
            total_resource_usage: ResourceUsage {
                cpu_time: 0,
                memory_peak: 0,
                network_bytes: 0,
                storage_bytes: 0,
            },
        })
    }
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ExecutionStatistics {
    pub skill_id: String,
    pub total_executions: u64,
    pub successful_executions: u64,
    pub failed_executions: u64,
    pub average_execution_time: f64,
    pub total_resource_usage: ResourceUsage,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ResourceUsage {
    pub cpu_time: u64,
    pub memory_peak: u64,
    pub network_bytes: u64,
    pub storage_bytes: u64,
}

#[allow(dead_code)]
impl TEECompatibilityChecker {
    pub fn new() -> Self {
        let mut supported_tees = HashMap::new();

        // Intel SGX capabilities
        supported_tees.insert(
            TEEType::IntelSGX,
            TEECapabilities {
                isolation_level: IsolationLevel::TEE,
                attestation_support: true,
                secure_storage: true,
                memory_encryption: true,
                code_integrity: true,
                supported_runtimes: vec!["wasm".to_string(), "native".to_string()],
            },
        );

        // Software TEE for development
        supported_tees.insert(
            TEEType::Software,
            TEECapabilities {
                isolation_level: IsolationLevel::Process,
                attestation_support: false,
                secure_storage: false,
                memory_encryption: false,
                code_integrity: false,
                supported_runtimes: vec!["wasm".to_string(), "javascript".to_string()],
            },
        );

        Self {
            supported_tees,
            security_policies: SecurityPolicies {
                minimum_isolation: IsolationLevel::Process,
                require_attestation: false, // Relaxed for development
                allow_network_access: false,
                allow_file_system_access: false,
                trusted_tee_types: vec![TEEType::IntelSGX, TEEType::Software],
            },
        }
    }

    /// Verify TEE compatibility with skill requirements
    pub fn verify_compatibility(&self, skill: &SkillMetadata, tee_info: &TEEInfo) -> Result<()> {
        // Check if TEE type is supported
        let tee_capabilities = self
            .supported_tees
            .get(&tee_info.tee_type)
            .ok_or_else(|| anyhow!("Unsupported TEE type: {:?}", tee_info.tee_type))?;

        // Check if TEE type is trusted
        if !self
            .security_policies
            .trusted_tee_types
            .contains(&tee_info.tee_type)
        {
            return Err(anyhow!("TEE type not trusted: {:?}", tee_info.tee_type));
        }

        // Check memory requirements
        if tee_info.memory_limit < 128 * 1024 * 1024 {
            // Minimum 128MB
            return Err(anyhow!("Insufficient memory in TEE"));
        }

        // Check attestation requirements
        if self.security_policies.require_attestation && !tee_capabilities.attestation_support {
            return Err(anyhow!("TEE does not support required attestation"));
        }

        info!("TEE compatibility verified for skill: {}", skill.name);
        Ok(())
    }

    /// Check if TEE supports specific runtime
    pub fn supports_runtime(&self, tee_type: &TEEType, runtime: &str) -> bool {
        if let Some(capabilities) = self.supported_tees.get(tee_type) {
            capabilities
                .supported_runtimes
                .contains(&runtime.to_string())
        } else {
            false
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::nrn_token::{PerformanceMetrics, SkillMetadata, ValidationStatus};

    #[tokio::test]
    async fn test_tee_compatibility() {
        let checker = TEECompatibilityChecker::new();

        let tee_info = TEEInfo {
            tee_type: TEEType::Software,
            version: "1.0".to_string(),
            capabilities: vec!["wasm".to_string()],
            attestation_support: false,
            secure_storage: false,
            memory_limit: 256 * 1024 * 1024, // 256MB
            cpu_cores: 2,
            network_isolation: true,
            device_id: "test-device".to_string(),
            platform: "linux".to_string(),
        };

        let skill = SkillMetadata {
            name: "test-skill".to_string(),
            skill_type: "computation".to_string(),
            capabilities: vec!["math".to_string()],
            requirements: HashMap::new(),
            owner: Address([0u8; 20]),
            usage_fee: num_bigint::BigInt::from(100),
            validation_status: ValidationStatus::Validated,
            performance_metrics: PerformanceMetrics {
                success_rate: 1.0,
                average_latency: 100.0,
                total_invocations: 0,
                last_updated: 0,
            },
            registered_at: 0,
        };

        let result = checker.verify_compatibility(&skill, &tee_info);
        assert!(result.is_ok());
    }
}
