use anyhow::Result;
use clap::{Parser, Subcommand};
use knirv_model_forge::*;
use std::path::PathBuf;

#[derive(Parser)]
#[command(name = "forge")]
#[command(about = "KNIRV Model Forge - ingestion/normalization/compilation pipeline")]
struct Cli {
    #[command(subcommand)]
    command: Commands,
}

#[derive(Subcommand)]
enum Commands {
    /// Forge a model from Hugging Face
    HuggingFace {
        /// Repository ID (e.g., microsoft/phi-3-mini-4k-instruct)
        repo_id: String,
        /// Git revision (default: main)
        #[arg(short, long)]
        revision: Option<String>,
        /// Output directory
        #[arg(short, long, default_value = "./dist")]
        output: PathBuf,
        /// Runtime type
        #[arg(short, long, default_value = "tract-onnx")]
        runtime: String,
    },
    /// Forge a model from local path
    Local {
        /// Path to model files
        path: PathBuf,
        /// Output directory
        #[arg(short, long, default_value = "./dist")]
        output: PathBuf,
        /// Runtime type
        #[arg(short, long, default_value = "tract-onnx")]
        runtime: String,
    },
    /// Forge a model from URL
    Url {
        /// URL to model file
        url: String,
        /// Model format
        #[arg(short, long, default_value = "onnx")]
        format: String,
        /// Output directory
        #[arg(short, long, default_value = "./dist")]
        output: PathBuf,
        /// Runtime type
        #[arg(short, long, default_value = "tract-onnx")]
        runtime: String,
    },
    /// List supported models
    List,
    /// Validate a forged model
    Validate {
        /// Path to cortex.wasm file
        wasm_file: PathBuf,
    },
}

#[tokio::main]
async fn main() -> Result<()> {
    let cli = Cli::parse();

    match cli.command {
        Commands::HuggingFace { repo_id, revision, output, runtime } => {
            forge_huggingface_model(repo_id, revision, output, runtime).await
        }
        Commands::Local { path, output, runtime } => {
            forge_local_model(path, output, runtime).await
        }
        Commands::Url { url, format, output, runtime } => {
            forge_url_model(url, format, output, runtime).await
        }
        Commands::List => {
            list_supported_models().await
        }
        Commands::Validate { wasm_file } => {
            validate_model(wasm_file).await
        }
    }
}

async fn forge_huggingface_model(
    repo_id: String,
    revision: Option<String>,
    output: PathBuf,
    runtime: String,
) -> Result<()> {
    println!("🚀 Forging Hugging Face model: {}", repo_id);

    let config = ForgeConfig {
        output_dir: output,
        preferred_runtime: runtime,
        ..Default::default()
    };

    let mut forge = ModelForge::new(config);
    let source = ModelSource::HuggingFace { repo_id, revision };

    let result = forge.forge_model(source).await?;

    println!("✅ Model forged successfully!");
    println!("📁 Output: {}", result.cortex_wasm.display());
    println!("📊 Size: {:.2} MB", result.size_bytes as f64 / (1024.0 * 1024.0));
    println!("🔍 Checksum: {}", result.checksums.get("cortex.wasm").unwrap_or(&"unknown".to_string()));

    Ok(())
}

async fn forge_local_model(
    path: PathBuf,
    output: PathBuf,
    runtime: String,
) -> Result<()> {
    println!("🚀 Forging local model: {}", path.display());

    let config = ForgeConfig {
        output_dir: output,
        preferred_runtime: runtime,
        ..Default::default()
    };

    let mut forge = ModelForge::new(config);
    let source = ModelSource::LocalPath { path };

    let result = forge.forge_model(source).await?;

    println!("✅ Model forged successfully!");
    println!("📁 Output: {}", result.cortex_wasm.display());
    println!("📊 Size: {:.2} MB", result.size_bytes as f64 / (1024.0 * 1024.0));

    Ok(())
}

async fn forge_url_model(
    url: String,
    format: String,
    output: PathBuf,
    runtime: String,
) -> Result<()> {
    println!("🚀 Forging model from URL: {}", url);

    let config = ForgeConfig {
        output_dir: output,
        preferred_runtime: runtime,
        ..Default::default()
    };

    let mut forge = ModelForge::new(config);
    let source = ModelSource::Url { url, format };

    let result = forge.forge_model(source).await?;

    println!("✅ Model forged successfully!");
    println!("📁 Output: {}", result.cortex_wasm.display());
    println!("📊 Size: {:.2} MB", result.size_bytes as f64 / (1024.0 * 1024.0));

    Ok(())
}

async fn list_supported_models() -> Result<()> {
    println!("📋 Supported Models:");
    println!();
    
    println!("🤗 Hugging Face Models:");
    println!("  • microsoft/phi-3-mini-4k-instruct (Phi-3 Mini)");
    println!("  • google/recurrentgemma-2b (RecurrentGemma-2B)");
    println!("  • TinyLlama/TinyLlama-1.1B-Chat-v1.0 (TinyLlama-1.1B)");
    println!("  • Salesforce/codet5-small (CodeT5 Small)");
    println!();
    
    println!("🔧 Supported Runtimes:");
    println!("  • tract-onnx (ONNX models via Tract)");
    println!("  • candle (Safetensors models via Candle)");
    println!();
    
    println!("📁 Supported Formats:");
    println!("  • .safetensors (preferred)");
    println!("  • .bin/.pt (PyTorch, converted to safetensors)");
    println!("  • .onnx (ONNX format)");
    println!();

    Ok(())
}

async fn validate_model(wasm_file: PathBuf) -> Result<()> {
    println!("🔍 Validating model: {}", wasm_file.display());

    if !wasm_file.exists() {
        anyhow::bail!("WASM file not found: {}", wasm_file.display());
    }

    let wasm_binary = tokio::fs::read(&wasm_file).await?;
    
    // Create a mock compilation result for validation
    let compilation = CompilationResult {
        manifest: shared_types::ForgeManifest {
            model_id: "validation-test".to_string(),
            source_url: "file://validation".to_string(),
            license: "unknown".to_string(),
            model_family: "unknown".to_string(),
            dimensions: None,
            tokenizer: None,
            checksum: "".to_string(),
            size_bytes: wasm_binary.len() as u64,
            capabilities: vec!["inference".to_string()],
        },
        wasm_binary,
        size_bytes: wasm_file.metadata()?.len(),
        optimization_report: "Validation run".to_string(),
    };

    let validator = ModelValidator::new();
    let result = validator.validate(&compilation).await?;

    println!("📊 Validation Results:");
    println!("  Overall: {}", if result.passed { "✅ PASSED" } else { "❌ FAILED" });
    println!("  Score: {:.1}%", result.performance_score);
    println!();

    for test in &result.test_results {
        let status = if test.passed { "✅" } else { "❌" };
        println!("  {} {} ({:.1}ms)", status, test.name, test.duration_ms);
        println!("    {}", test.details);
    }

    if !result.errors.is_empty() {
        println!();
        println!("❌ Errors:");
        for error in &result.errors {
            println!("  • {}", error);
        }
    }

    if !result.warnings.is_empty() {
        println!();
        println!("⚠️  Warnings:");
        for warning in &result.warnings {
            println!("  • {}", warning);
        }
    }

    Ok(())
}
