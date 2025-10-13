use anyhow::{anyhow, Result};
use serde::{Deserialize, Serialize};
use shared_types::*;
use std::collections::HashMap;
use std::path::{Path, PathBuf};

pub mod discovery;
pub mod normalization;
pub mod runtime_binding;
pub mod validation;
pub mod external_inference;

// Re-export main types
pub use discovery::*;
pub use normalization::*;
pub use runtime_binding::*;
pub use validation::{ModelValidator, ModelCompiler, ModelPackager, ValidationTest};
pub use external_inference::*;

// Use explicit import for ValidationResult to avoid ambiguity
use validation::ValidationResult as ForgeValidationResult;

/// Main Model Forge pipeline coordinator
pub struct ModelForge {
    pub config: ForgeConfig,
    pub manifest: Option<ForgeManifest>,
    pub external_inference: Option<ExternalInferenceForge>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ForgeConfig {
    pub output_dir: PathBuf,
    pub temp_dir: PathBuf,
    pub max_model_size_gb: f64,
    pub preferred_runtime: String,
    pub quantization_enabled: bool,
    pub validation_enabled: bool,
}

impl Default for ForgeConfig {
    fn default() -> Self {
        Self {
            output_dir: PathBuf::from("./dist"),
            temp_dir: PathBuf::from("./temp"),
            max_model_size_gb: 10.0,
            preferred_runtime: "tract-onnx".to_string(),
            quantization_enabled: true,
            validation_enabled: true,
        }
    }
}

impl ModelForge {
    pub fn new(config: ForgeConfig) -> Self {
        Self {
            config,
            manifest: None,
            external_inference: None,
        }
    }

    /// Initialize external inference integration
    pub fn init_external_inference(&mut self) -> Result<()> {
        self.external_inference = Some(ExternalInferenceForge::new());
        Ok(())
    }

    /// Configure external inference provider
    pub fn configure_external_provider(&mut self, config: ExternalInferenceConfig) -> Result<()> {
        if let Some(ref mut external) = self.external_inference {
            external.configure_provider(config)?;
            Ok(())
        } else {
            Err(anyhow!("External inference not initialized. Call init_external_inference() first."))
        }
    }

    /// Check if external inference is available
    pub fn is_external_inference_available(&self) -> bool {
        self.external_inference.as_ref()
            .map(|ext| ext.is_available())
            .unwrap_or(false)
    }

    /// Execute the complete forge pipeline
    pub async fn forge_model(&mut self, source: ModelSource) -> Result<ForgeOutput> {
        println!("🔨 Starting Model Forge pipeline...");

        // Phase 1: Discovery
        println!("📋 Phase 1: Discovery");
        let discovery_result = self.discover_model(source).await?;
        self.manifest = Some(discovery_result.manifest.clone());

        // Phase 2: Normalization
        println!("🔧 Phase 2: Normalization");
        let normalization_result = self.normalize_model(discovery_result).await?;

        // Phase 3: Runtime Binding
        println!("⚙️  Phase 3: Runtime Binding");
        let binding_result = self.bind_runtime(normalization_result).await?;

        // Phase 4: Compilation & Linking
        println!("🏗️  Phase 4: Compilation & Linking");
        let compilation_result = self.compile_and_link(binding_result).await?;

        // Phase 4.5: External Inference Integration (if enabled)
        if self.is_external_inference_available() {
            println!("🌐 Phase 4.5: External Inference Integration");
            self.integrate_external_inference(&compilation_result).await?;
        }

        // Phase 5: Validation
        if self.config.validation_enabled {
            println!("✅ Phase 5: Validation");
            self.validate_output(&compilation_result).await?;
        }

        // Phase 6: Packaging
        println!("📦 Phase 6: Packaging");
        let package_result = self.package_output(compilation_result).await?;

        println!("🎉 Model Forge pipeline completed successfully!");
        Ok(package_result)
    }

    async fn discover_model(&self, source: ModelSource) -> Result<DiscoveryResult> {
        let discoverer = ModelDiscoverer::new();
        discoverer.discover(source).await
    }

    async fn normalize_model(&self, discovery: DiscoveryResult) -> Result<NormalizationResult> {
        let normalizer = ModelNormalizer::new(self.config.clone());
        normalizer.normalize(discovery).await
    }

    async fn bind_runtime(&self, normalization: NormalizationResult) -> Result<RuntimeBindingResult> {
        let binder = RuntimeBinder::new(self.config.preferred_runtime.clone());
        binder.bind(normalization).await
    }

    async fn compile_and_link(&self, binding: RuntimeBindingResult) -> Result<CompilationResult> {
        let compiler = ModelCompiler::new(self.config.clone());
        compiler.compile(binding).await
    }

    async fn validate_output(&self, compilation: &CompilationResult) -> Result<ForgeValidationResult> {
        let validator = ModelValidator::new();
        validator.validate(compilation).await
    }

    async fn package_output(&self, compilation: CompilationResult) -> Result<ForgeOutput> {
        let packager = ModelPackager::new(self.config.clone());
        packager.package(compilation).await
    }

    /// Integrate external inference capabilities into the compiled model
    async fn integrate_external_inference(&mut self, compilation: &CompilationResult) -> Result<()> {
        if let Some(ref mut external) = self.external_inference {
            println!("  🔗 Integrating external inference providers...");

            // Create a mock compilation request for external integration
            let integration_request = ModelCompilationRequest {
                source_code: "// Mock source code for integration".to_string(),
                target_format: "cortex.wasm".to_string(),
                optimization_level: 2,
                external_inference_integration: true,
                provider_configs: vec![], // Would be populated with actual configs
            };

            let integration_result = external.compile_model_with_external_integration(integration_request).await?;

            if integration_result.success {
                println!("  ✅ External inference integration completed");
                for (provider, status) in &integration_result.external_integration_status {
                    let status_icon = if *status { "✅" } else { "❌" };
                    println!("    {} {}: {}", status_icon, provider, if *status { "Ready" } else { "Failed" });
                }
            } else {
                return Err(anyhow!("External inference integration failed: {:?}", integration_result.error));
            }
        }

        Ok(())
    }

    /// Validate model output using external inference
    pub async fn validate_with_external_inference(
        &mut self,
        model_output: &str,
        expected_output: &str
    ) -> Result<ExternalValidationResponse> {
        if let Some(ref mut external) = self.external_inference {
            let validation_request = ExternalValidationRequest {
                model_output: model_output.to_string(),
                expected_output: expected_output.to_string(),
                context: None,
                validation_criteria: vec![
                    "accuracy".to_string(),
                    "coherence".to_string(),
                    "relevance".to_string(),
                ],
            };

            external.validate_model_output(validation_request).await
        } else {
            Err(anyhow!("External inference not initialized"))
        }
    }

    /// Get external inference statistics
    pub fn get_external_inference_stats(&self) -> Option<std::collections::HashMap<String, serde_json::Value>> {
        self.external_inference.as_ref().map(|ext| ext.get_stats())
    }
}

/// Input source for model discovery
#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum ModelSource {
    HuggingFace {
        repo_id: String,
        revision: Option<String>,
    },
    LocalPath {
        path: PathBuf,
    },
    Url {
        url: String,
        format: String,
    },
}

/// Final output of the forge pipeline
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ForgeOutput {
    pub cortex_wasm: PathBuf,
    pub manifest: ForgeManifest,
    pub checksums: HashMap<String, String>,
    pub size_bytes: u64,
    pub validation_report: Option<ForgeValidationResult>,
}

/// Result types for each phase
#[derive(Debug, Clone)]
pub struct DiscoveryResult {
    pub manifest: ForgeManifest,
    pub source_files: Vec<PathBuf>,
    pub metadata: HashMap<String, String>,
}

#[derive(Debug, Clone)]
pub struct NormalizationResult {
    pub manifest: ForgeManifest,
    pub normalized_files: Vec<PathBuf>,
    pub safetensors_path: Option<PathBuf>,
    pub tokenizer_path: Option<PathBuf>,
    pub config_path: Option<PathBuf>,
}

#[derive(Debug, Clone)]
pub struct RuntimeBindingResult {
    pub manifest: ForgeManifest,
    pub runtime_config: RuntimeConfig,
    pub model_files: Vec<PathBuf>,
    pub binding_code: String,
}

#[derive(Debug, Clone)]
pub struct CompilationResult {
    pub manifest: ForgeManifest,
    pub wasm_binary: Vec<u8>,
    pub size_bytes: u64,
    pub optimization_report: String,
}

/// Error types
#[derive(Debug, thiserror::Error)]
pub enum ForgeError {
    #[error("Discovery failed: {0}")]
    Discovery(String),
    
    #[error("Normalization failed: {0}")]
    Normalization(String),
    
    #[error("Runtime binding failed: {0}")]
    RuntimeBinding(String),
    
    #[error("Compilation failed: {0}")]
    Compilation(String),
    
    #[error("Validation failed: {0}")]
    Validation(String),
    
    #[error("Packaging failed: {0}")]
    Packaging(String),
    
    #[error("IO error: {0}")]
    Io(#[from] std::io::Error),

    #[error("Serialization error: {0}")]
    Serialization(#[from] serde_json::Error),
}
