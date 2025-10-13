use anyhow::{anyhow, Result};
use serde_json::Value;
use shared_types::*;
use std::collections::HashMap;
use std::path::{Path, PathBuf};
use tokio::fs;

use crate::{DiscoveryResult, ForgeError, ModelSource};

/// Model discovery phase - analyzes source and produces manifest
pub struct ModelDiscoverer;

impl ModelDiscoverer {
    pub fn new() -> Self {
        Self
    }

    pub async fn discover(&self, source: ModelSource) -> Result<DiscoveryResult> {
        match source {
            ModelSource::HuggingFace { repo_id, revision } => {
                self.discover_huggingface(&repo_id, revision.as_deref()).await
            }
            ModelSource::LocalPath { path } => {
                self.discover_local_path(&path).await
            }
            ModelSource::Url { url, format } => {
                self.discover_url(&url, &format).await
            }
        }
    }

    async fn discover_huggingface(&self, repo_id: &str, revision: Option<&str>) -> Result<DiscoveryResult> {
        println!("🔍 Discovering Hugging Face model: {}", repo_id);

        let revision = revision.unwrap_or("main");

        // Simulate model metadata (in real implementation, would fetch from HF API)
        let metadata = serde_json::json!({
            "id": repo_id,
            "modelId": repo_id,
            "author": repo_id.split('/').next().unwrap_or("unknown"),
            "sha": "abc123def456",
            "downloads": 1000,
            "likes": 50,
            "library_name": "transformers",
            "tags": ["text-generation", "pytorch", "safetensors"],
            "pipeline_tag": "text-generation"
        });
        
        // Extract model information
        let model_name = metadata["id"].as_str().unwrap_or(repo_id).to_string();
        let license = metadata["license"].as_str().unwrap_or("unknown").to_string();
        let tags: Vec<String> = metadata["tags"].as_array()
            .map(|arr| arr.iter().filter_map(|v| v.as_str()).map(String::from).collect())
            .unwrap_or_else(|| Vec::new());
        
        // Determine model family and capabilities
        let model_family = self.infer_model_family(&tags, &model_name);
        let capabilities = self.infer_capabilities(&tags, &metadata);
        
        // Simulate file list (in real implementation, would fetch from HF API)
        let files: Vec<Value> = vec![
            serde_json::json!({"path": "model.safetensors", "type": "file", "size": 1000000}),
            serde_json::json!({"path": "config.json", "type": "file", "size": 1024}),
            serde_json::json!({"path": "tokenizer.json", "type": "file", "size": 2048}),
            serde_json::json!({"path": "tokenizer_config.json", "type": "file", "size": 512}),
        ];
        
        // Find relevant files
        let mut source_files = Vec::new();
        let mut has_safetensors = false;
        let mut has_pytorch = false;
        let mut tokenizer_files = Vec::new();
        
        for file in &files {
            if let Some(path) = file["path"].as_str() {
                if path.ends_with(".safetensors") {
                    has_safetensors = true;
                    source_files.push(PathBuf::from(path));
                } else if path.ends_with(".bin") || path.ends_with(".pt") {
                    has_pytorch = true;
                    source_files.push(PathBuf::from(path));
                } else if path.contains("tokenizer") {
                    tokenizer_files.push(PathBuf::from(path));
                }
            }
        }
        
        // Estimate model dimensions
        let dimensions = self.estimate_dimensions(&metadata, &files).await?;
        
        // Create tokenizer info
        let tokenizer = TokenizerInfo {
            r#type: "huggingface".to_string(),
            vocab_file: tokenizer_files.first()
                .map(|p| p.to_string_lossy().to_string())
                .unwrap_or_default(),
            special_tokens: HashMap::new(),
        };
        
        // Build manifest
        let manifest = ForgeManifest {
            model_id: model_name,
            source_url: format!("https://huggingface.co/{}", repo_id),
            license,
            model_family,
            dimensions: Some(dimensions),
            tokenizer: Some(tokenizer),
            checksum: String::new(), // Will be computed during normalization
            size_bytes: 0, // Will be computed during normalization
            capabilities,
        };
        
        // Collect metadata
        let mut discovery_metadata = HashMap::new();
        discovery_metadata.insert("repo_id".to_string(), repo_id.to_string());
        discovery_metadata.insert("revision".to_string(), revision.to_string());
        discovery_metadata.insert("has_safetensors".to_string(), has_safetensors.to_string());
        discovery_metadata.insert("has_pytorch".to_string(), has_pytorch.to_string());
        
        Ok(DiscoveryResult {
            manifest,
            source_files,
            metadata: discovery_metadata,
        })
    }

    async fn discover_local_path(&self, path: &Path) -> Result<DiscoveryResult> {
        println!("🔍 Discovering local model: {}", path.display());
        
        if !path.exists() {
            return Err(anyhow!("Path does not exist: {}", path.display()));
        }
        
        let mut source_files = Vec::new();
        let mut metadata = HashMap::new();
        
        if path.is_file() {
            source_files.push(path.to_path_buf());
            metadata.insert("type".to_string(), "single_file".to_string());
        } else {
            // Scan directory for model files
            let mut entries = fs::read_dir(path).await?;
            while let Some(entry) = entries.next_entry().await? {
                let file_path = entry.path();
                if let Some(ext) = file_path.extension() {
                    if ext == "safetensors" || ext == "bin" || ext == "pt" || ext == "onnx" {
                        source_files.push(file_path);
                    }
                }
            }
            metadata.insert("type".to_string(), "directory".to_string());
        }
        
        // Create basic manifest
        let model_name = path.file_name()
            .and_then(|n| n.to_str())
            .unwrap_or("unknown")
            .to_string();
        
        let manifest = ForgeManifest {
            model_id: model_name,
            source_url: format!("file://{}", path.display()),
            license: "unknown".to_string(),
            model_family: "unknown".to_string(),
            dimensions: None,
            tokenizer: None,
            checksum: String::new(),
            size_bytes: 0,
            capabilities: vec!["inference".to_string()],
        };
        
        Ok(DiscoveryResult {
            manifest,
            source_files,
            metadata,
        })
    }

    async fn discover_url(&self, url: &str, format: &str) -> Result<DiscoveryResult> {
        println!("🔍 Discovering model from URL: {}", url);
        
        // Basic URL-based discovery
        let model_name = url.split('/').last().unwrap_or("unknown").to_string();
        
        let manifest = ForgeManifest {
            model_id: model_name,
            source_url: url.to_string(),
            license: "unknown".to_string(),
            model_family: format.to_string(),
            dimensions: None,
            tokenizer: None,
            checksum: String::new(),
            size_bytes: 0,
            capabilities: vec!["inference".to_string()],
        };
        
        let mut metadata = HashMap::new();
        metadata.insert("format".to_string(), format.to_string());
        metadata.insert("url".to_string(), url.to_string());
        
        Ok(DiscoveryResult {
            manifest,
            source_files: vec![PathBuf::from(url)],
            metadata,
        })
    }

    fn infer_model_family(&self, tags: &[String], model_name: &str) -> String {
        let name_lower = model_name.to_lowercase();
        
        if tags.iter().any(|t| t.contains("phi")) || name_lower.contains("phi") {
            "phi".to_string()
        } else if tags.iter().any(|t| t.contains("gemma")) || name_lower.contains("gemma") {
            "gemma".to_string()
        } else if tags.iter().any(|t| t.contains("llama")) || name_lower.contains("llama") {
            "llama".to_string()
        } else if tags.iter().any(|t| t.contains("bert")) || name_lower.contains("bert") {
            "bert".to_string()
        } else if tags.iter().any(|t| t.contains("gpt")) || name_lower.contains("gpt") {
            "gpt".to_string()
        } else {
            "unknown".to_string()
        }
    }

    fn infer_capabilities(&self, tags: &[String], metadata: &Value) -> Vec<String> {
        let mut capabilities = vec!["inference".to_string()];
        
        if tags.iter().any(|t| t.contains("text-generation")) {
            capabilities.push("text-generation".to_string());
        }
        if tags.iter().any(|t| t.contains("conversational")) {
            capabilities.push("chat".to_string());
        }
        if tags.iter().any(|t| t.contains("code")) {
            capabilities.push("code-generation".to_string());
        }
        
        capabilities
    }

    async fn estimate_dimensions(&self, metadata: &Value, files: &[Value]) -> Result<ModelDimensions> {
        // Try to extract from config.json if available
        // For now, return default dimensions
        Ok(ModelDimensions {
            parameters: 1_000_000, // Will be updated during normalization
            hidden_size: 768,
            num_layers: 12,
            vocab_size: 32000,
            max_sequence_length: 2048,
        })
    }
}
