# KNIRV Model Forge

Advanced model compilation and training pipeline with external AI integration for validation and optimization.

## 🆕 Recent Updates

### ✅ External AI Integration (Beta Phase)
- **Multi-Provider Validation**: Integrated Google Gemini, Anthropic Claude, OpenAI ChatGPT-5, and Deepseek for model validation
- **External Inference Forge**: New `external_inference.rs` module for compilation with external API integration
- **Model Training Enhancement**: External validation during training process
- **Compilation Pipeline**: Enhanced pipeline with external provider integration
- **Performance Metrics**: Comprehensive metrics tracking for external API usage

## Features

### Core Model Forge Pipeline
1. **Discovery**: Automatic model detection and analysis
2. **Normalization**: Model format standardization
3. **Runtime Binding**: Target runtime integration
4. **Compilation & Linking**: Model compilation with optimizations
5. **External Integration**: External AI provider integration (NEW)
6. **Validation**: Comprehensive model testing
7. **Packaging**: Final model packaging and distribution

### External AI Integration
- **Provider Configuration**: Support for multiple external AI providers
- **Model Validation**: External validation of model outputs
- **Training Enhancement**: External feedback during training
- **Compilation Integration**: External API integration code generation
- **Performance Monitoring**: Detailed metrics and analytics

## Usage

### Basic Model Forge
```rust
use knirv_model_forge::*;

#[tokio::main]
async fn main() -> Result<()> {
    let config = ForgeConfig::default();
    let mut forge = ModelForge::new(config);
    
    // Initialize external inference (optional)
    forge.init_external_inference()?;
    
    // Configure external provider
    let provider_config = ExternalInferenceConfig {
        provider: ExternalProvider::Gemini,
        api_key: "your-api-key".to_string(),
        enabled: true,
        ..Default::default()
    };
    forge.configure_external_provider(provider_config)?;
    
    // Forge model
    let source = ModelSource::HuggingFace {
        repo_id: "microsoft/codebert-base".to_string(),
        revision: None,
    };
    
    let output = forge.forge_model(source).await?;
    println!("Model forged successfully: {:?}", output.cortex_wasm);
    
    Ok(())
}
```

### External Validation
```rust
use knirv_model_forge::*;

async fn validate_model_output() -> Result<()> {
    let mut forge = ModelForge::new(ForgeConfig::default());
    forge.init_external_inference()?;
    
    let validation_request = ExternalValidationRequest {
        model_output: "Generated code output".to_string(),
        expected_output: "Expected code output".to_string(),
        context: Some("Code generation task".to_string()),
        validation_criteria: vec![
            "correctness".to_string(),
            "efficiency".to_string(),
            "readability".to_string(),
        ],
    };
    
    let response = forge.validate_with_external_inference(
        &validation_request.model_output,
        &validation_request.expected_output
    ).await?;
    
    println!("Validation score: {}", response.score);
    println!("Feedback: {}", response.feedback);
    
    Ok(())
}
```

### Model Compilation with External Integration
```rust
async fn compile_with_external_integration() -> Result<()> {
    let mut external_forge = ExternalInferenceForge::new();
    
    // Configure providers
    let gemini_config = ExternalInferenceConfig {
        provider: ExternalProvider::Gemini,
        api_key: "gemini-key".to_string(),
        enabled: true,
        ..Default::default()
    };
    external_forge.configure_provider(gemini_config)?;
    
    let claude_config = ExternalInferenceConfig {
        provider: ExternalProvider::Claude,
        api_key: "claude-key".to_string(),
        enabled: true,
        ..Default::default()
    };
    external_forge.configure_provider(claude_config)?;
    
    // Compile model with external integration
    let compilation_request = ModelCompilationRequest {
        source_code: "// TypeScript cognitive shell code".to_string(),
        target_format: "cortex.wasm".to_string(),
        optimization_level: 2,
        external_inference_integration: true,
        provider_configs: vec![gemini_config, claude_config],
    };
    
    let result = external_forge.compile_model_with_external_integration(
        compilation_request
    ).await?;
    
    if result.success {
        println!("Compilation successful!");
        for (provider, status) in &result.external_integration_status {
            println!("Provider {}: {}", provider, if *status { "✓" } else { "✗" });
        }
    }
    
    Ok(())
}
```

## Configuration

### Forge Configuration
```rust
let config = ForgeConfig {
    output_dir: PathBuf::from("./dist"),
    temp_dir: PathBuf::from("./temp"),
    max_model_size_gb: 10.0,
    preferred_runtime: "tract-onnx".to_string(),
    quantization_enabled: true,
    validation_enabled: true,
};
```

### External Provider Configuration
```rust
let provider_config = ExternalInferenceConfig {
    provider: ExternalProvider::OpenAI,
    api_key: "sk-...".to_string(),
    endpoint: Some("https://api.openai.com/v1".to_string()),
    model: Some("gpt-4".to_string()),
    max_tokens: Some(2048),
    temperature: Some(0.7),
    timeout_seconds: Some(30),
    retry_attempts: Some(3),
    enabled: true,
};
```

## Model Sources

### Supported Sources
- **HuggingFace**: Direct model loading from HuggingFace Hub
- **Local Path**: Local model files and directories
- **URL**: Remote model downloads from URLs

```rust
// HuggingFace model
let source = ModelSource::HuggingFace {
    repo_id: "microsoft/DialoGPT-medium".to_string(),
    revision: Some("main".to_string()),
};

// Local model
let source = ModelSource::LocalPath {
    path: PathBuf::from("./models/my-model"),
};

// Remote URL
let source = ModelSource::Url {
    url: "https://example.com/model.onnx".to_string(),
    format: "onnx".to_string(),
};
```

## External Providers

### Supported Providers
- **Google Gemini**: Advanced reasoning and code generation
- **Anthropic Claude**: Thoughtful analysis and validation
- **OpenAI ChatGPT-5**: Comprehensive language understanding
- **Deepseek**: Specialized code and technical validation

### Provider Capabilities
```rust
// Provider-specific validation
match provider {
    ExternalProvider::Gemini => {
        // Comprehensive analysis with detailed reasoning
        confidence_boost = 0.05;
    }
    ExternalProvider::Claude => {
        // Thoughtful evaluation with nuanced feedback
        confidence_boost = 0.08;
    }
    ExternalProvider::OpenAI => {
        // Structured assessment with clear metrics
        confidence_boost = 0.03;
    }
    ExternalProvider::Deepseek => {
        // Technical analysis with optimization suggestions
        confidence_boost = 0.06;
    }
}
```

## Performance Metrics

### Compilation Metrics
- **Compilation Time**: Total time for model compilation
- **Model Size**: Final compiled model size in bytes
- **Optimization Level**: Applied optimization level (0-3)
- **Memory Usage**: Peak memory usage during compilation

### External Integration Metrics
- **Request Count**: Total external API requests
- **Success Rate**: Percentage of successful requests
- **Average Response Time**: Mean response time across providers
- **Provider Usage**: Request distribution across providers

### Validation Metrics
- **Validation Score**: Overall model validation score (0.0-1.0)
- **Confidence Level**: Validation confidence level
- **Feedback Quality**: Quality of external feedback
- **Suggestion Count**: Number of improvement suggestions

## Building

### Prerequisites
- Rust 1.70+
- Tokio runtime
- External API keys (for validation features)

### Build Commands
```bash
# Build the forge binary
cargo build --release

# Run tests
cargo test

# Build with external inference features
cargo build --release --features external-inference

# Run the forge
./target/release/forge --help
```

## CLI Usage

```bash
# Basic model forging
forge --source huggingface --repo microsoft/codebert-base --output ./dist

# With external validation
forge --source local --path ./my-model --external-validation --provider gemini

# Custom configuration
forge --config ./forge-config.toml --source url --url https://example.com/model.onnx
```

## Integration with KNIRVCONTROLLER

The Model Forge integrates seamlessly with KNIRVCONTROLLER's model creation workflow:

1. **Model Creation Page**: Users configure models through the UI
2. **External API Setup**: API keys are configured during onboarding
3. **Compilation Process**: Model Forge compiles with external integration
4. **Validation**: External providers validate model outputs
5. **Deployment**: Compiled models are deployed to KNIRV network

## Troubleshooting

### Common Issues
1. **Compilation Failures**: Check source model format and dependencies
2. **External API Errors**: Verify API keys and rate limits
3. **Memory Issues**: Adjust max_model_size_gb in configuration
4. **Network Timeouts**: Increase timeout_seconds for external providers

### Debug Mode
```rust
// Enable debug logging
env_logger::init();

// Get detailed metrics
let stats = forge.get_external_inference_stats();
println!("Forge stats: {:?}", stats);
```

## License

MIT License - see LICENSE file for details.
