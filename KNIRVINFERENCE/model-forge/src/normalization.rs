use anyhow::{anyhow, Result};
use sha2::{Digest, Sha256};
use std::collections::HashMap;
use std::path::{Path, PathBuf};
use tokio::fs;
use tokio::io::AsyncReadExt;

use crate::{DiscoveryResult, ForgeConfig, ForgeError, NormalizationResult};

/// Model normalization phase - converts to safetensors and validates
pub struct ModelNormalizer {
    config: ForgeConfig,
}

impl ModelNormalizer {
    pub fn new(config: ForgeConfig) -> Self {
        Self { config }
    }

    pub async fn normalize(&self, discovery: DiscoveryResult) -> Result<NormalizationResult> {
        println!("🔧 Normalizing model files...");

        // Create temp directory
        fs::create_dir_all(&self.config.temp_dir).await?;

        let mut normalized_files = Vec::new();
        let mut safetensors_path = None;
        let mut tokenizer_path = None;
        let mut config_path = None;

        // Process each source file
        for source_file in &discovery.source_files {
            let normalized = self.normalize_file(source_file, &discovery.metadata).await?;
            
            if let Some(path) = normalized {
                if path.extension().and_then(|s| s.to_str()) == Some("safetensors") {
                    safetensors_path = Some(path.clone());
                } else if path.file_name().and_then(|s| s.to_str()).unwrap_or("").contains("tokenizer") {
                    tokenizer_path = Some(path.clone());
                } else if path.file_name().and_then(|s| s.to_str()) == Some("config.json") {
                    config_path = Some(path.clone());
                }
                normalized_files.push(path);
            }
        }

        // Validate normalized files
        self.validate_normalized_files(&normalized_files).await?;

        // Update manifest with computed information
        let mut manifest = discovery.manifest;
        manifest.size_bytes = self.compute_total_size(&normalized_files).await?;
        manifest.checksum = self.compute_checksum(&normalized_files).await?;

        // Update dimensions if we have a config file
        if let Some(config_path) = &config_path {
            if let Ok(dimensions) = self.extract_dimensions_from_config(config_path).await {
                manifest.dimensions = Some(dimensions);
            }
        }

        Ok(NormalizationResult {
            manifest,
            normalized_files,
            safetensors_path,
            tokenizer_path,
            config_path,
        })
    }

    async fn normalize_file(&self, source_file: &Path, metadata: &HashMap<String, String>) -> Result<Option<PathBuf>> {
        let file_name = source_file.file_name()
            .and_then(|n| n.to_str())
            .ok_or_else(|| anyhow!("Invalid file name"))?;

        println!("  📄 Processing: {}", file_name);

        // Determine if this is a remote or local file
        if source_file.to_string_lossy().starts_with("http") {
            // Download remote file
            return self.download_and_normalize(source_file, metadata).await;
        }

        // Handle local file
        if !source_file.exists() {
            println!("    ⚠️  File not found, skipping: {}", source_file.display());
            return Ok(None);
        }

        let extension = source_file.extension().and_then(|s| s.to_str()).unwrap_or("");
        
        match extension {
            "safetensors" => {
                // Already in safetensors format, just copy
                let dest = self.config.temp_dir.join(file_name);
                fs::copy(source_file, &dest).await?;
                println!("    ✅ Copied safetensors file");
                Ok(Some(dest))
            }
            "bin" | "pt" => {
                // Convert PyTorch to safetensors
                self.convert_pytorch_to_safetensors(source_file).await
            }
            "onnx" => {
                // Copy ONNX file as-is
                let dest = self.config.temp_dir.join(file_name);
                fs::copy(source_file, &dest).await?;
                println!("    ✅ Copied ONNX file");
                Ok(Some(dest))
            }
            "json" => {
                // Copy config/tokenizer JSON files
                let dest = self.config.temp_dir.join(file_name);
                fs::copy(source_file, &dest).await?;
                println!("    ✅ Copied JSON file");
                Ok(Some(dest))
            }
            _ => {
                println!("    ⚠️  Unknown file type, skipping: {}", extension);
                Ok(None)
            }
        }
    }

    async fn download_and_normalize(&self, url: &Path, metadata: &HashMap<String, String>) -> Result<Option<PathBuf>> {
        let url_str = url.to_string_lossy();
        println!("    📥 Downloading: {}", url_str);

        // For Hugging Face URLs, construct the proper download URL
        let download_url = if let Some(repo_id) = metadata.get("repo_id") {
            let file_path = url.file_name().and_then(|n| n.to_str()).unwrap_or("");
            let revision = metadata.get("revision").map(|s| s.as_str()).unwrap_or("main");
            format!("https://huggingface.co/{}/resolve/{}/{}", repo_id, revision, file_path)
        } else {
            url_str.to_string()
        };

        // Simulate file download (in real implementation, would download from URL)
        println!("Simulating download from: {}", download_url);

        let file_name = url.file_name()
            .and_then(|n| n.to_str())
            .ok_or_else(|| anyhow!("Invalid file name in URL"))?;
        
        let dest_path = self.config.temp_dir.join(file_name);

        // Create a dummy file for simulation
        let dummy_content = b"dummy model file content";
        fs::write(&dest_path, dummy_content).await?;

        println!("    ✅ Simulated download of {} bytes", dummy_content.len());

        // Return the downloaded file path (already normalized for simulation)
        Ok(Some(dest_path))
    }

    async fn convert_pytorch_to_safetensors(&self, pytorch_file: &Path) -> Result<Option<PathBuf>> {
        println!("    🔄 Converting PyTorch to safetensors...");
        
        // For now, we'll simulate the conversion
        // In a real implementation, this would use Python bindings or a Rust PyTorch library
        let file_name = pytorch_file.file_stem()
            .and_then(|n| n.to_str())
            .ok_or_else(|| anyhow!("Invalid file name"))?;
        
        let safetensors_name = format!("{}.safetensors", file_name);
        let dest_path = self.config.temp_dir.join(safetensors_name);
        
        // Simulate conversion by copying the file with new extension
        // TODO: Implement actual PyTorch to safetensors conversion
        fs::copy(pytorch_file, &dest_path).await?;
        
        println!("    ✅ Converted to safetensors (simulated)");
        Ok(Some(dest_path))
    }

    async fn validate_normalized_files(&self, files: &[PathBuf]) -> Result<()> {
        println!("  🔍 Validating normalized files...");
        
        for file in files {
            if !file.exists() {
                return Err(anyhow!("Normalized file missing: {}", file.display()));
            }
            
            let metadata = fs::metadata(file).await?;
            if metadata.len() == 0 {
                return Err(anyhow!("Empty file: {}", file.display()));
            }
            
            // Validate file format based on extension
            if let Some(ext) = file.extension().and_then(|s| s.to_str()) {
                match ext {
                    "safetensors" => {
                        self.validate_safetensors_file(file).await?;
                    }
                    "json" => {
                        self.validate_json_file(file).await?;
                    }
                    _ => {} // Skip validation for other formats
                }
            }
        }
        
        println!("  ✅ All files validated");
        Ok(())
    }

    async fn validate_safetensors_file(&self, file: &Path) -> Result<()> {
        // Basic safetensors validation
        let mut file_handle = fs::File::open(file).await?;
        let mut header_size_bytes = [0u8; 8];
        file_handle.read_exact(&mut header_size_bytes).await?;
        
        let header_size = u64::from_le_bytes(header_size_bytes);
        if header_size > 1024 * 1024 { // 1MB header size limit
            return Err(anyhow!("Safetensors header too large: {} bytes", header_size));
        }
        
        Ok(())
    }

    async fn validate_json_file(&self, file: &Path) -> Result<()> {
        let content = fs::read_to_string(file).await?;
        serde_json::from_str::<serde_json::Value>(&content)?;
        Ok(())
    }

    async fn compute_total_size(&self, files: &[PathBuf]) -> Result<u64> {
        let mut total_size = 0;
        for file in files {
            let metadata = fs::metadata(file).await?;
            total_size += metadata.len();
        }
        Ok(total_size)
    }

    async fn compute_checksum(&self, files: &[PathBuf]) -> Result<String> {
        let mut hasher = Sha256::new();
        
        for file in files {
            let content = fs::read(file).await?;
            hasher.update(&content);
        }
        
        Ok(format!("{:x}", hasher.finalize()))
    }

    async fn extract_dimensions_from_config(&self, config_path: &Path) -> Result<shared_types::ModelDimensions> {
        let content = fs::read_to_string(config_path).await?;
        let config: serde_json::Value = serde_json::from_str(&content)?;
        
        let hidden_size = config["hidden_size"].as_u64().unwrap_or(768) as u32;
        let num_layers = config["num_hidden_layers"].as_u64().unwrap_or(12) as u32;
        let vocab_size = config["vocab_size"].as_u64().unwrap_or(32000) as u32;
        let max_length = config["max_position_embeddings"].as_u64().unwrap_or(2048) as u32;
        
        // Estimate parameters (rough calculation)
        let parameters = (hidden_size as u64) * (num_layers as u64) * 12; // Rough estimate
        
        Ok(shared_types::ModelDimensions {
            parameters,
            hidden_size,
            num_layers,
            vocab_size,
            max_sequence_length: max_length,
        })
    }
}
