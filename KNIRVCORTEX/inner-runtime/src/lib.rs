use anyhow::{anyhow, Result};
use serde::{Deserialize, Serialize};
use shared_types::*;
use std::collections::HashMap;

/// Inner Model Runtime - statically linked into cortex.wasm
/// This provides the actual ML inference capabilities
pub struct InnerModelRuntime {
    model_weights: Option<Vec<u8>>,
    model_config: Option<ModelConfig>,
    context_window: Vec<String>,
    memory_state: MemoryState,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ModelConfig {
    pub model_type: String,
    pub vocab_size: u32,
    pub hidden_size: u32,
    pub num_layers: u32,
    pub max_sequence_length: u32,
    pub temperature: f32,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MemoryState {
    pub short_term: Vec<String>,
    pub episodic: HashMap<String, String>,
    pub semantic: HashMap<String, Vec<f32>>,
    pub procedural: Vec<String>,
}

impl Default for InnerModelRuntime {
    fn default() -> Self {
        Self::new()
    }
}

impl InnerModelRuntime {
    /// Create a new inner runtime instance
    pub fn new() -> Self {
        Self {
            model_weights: None,
            model_config: None,
            context_window: Vec::new(),
            memory_state: MemoryState {
                short_term: Vec::new(),
                episodic: HashMap::new(),
                semantic: HashMap::new(),
                procedural: Vec::new(),
            },
        }
    }

    /// Load model weights from safetensors format
    pub fn load_weights(&mut self, weights: Vec<u8>) -> Result<()> {
        // In a real implementation, this would parse safetensors format
        // and load the weights into the ML framework (candle/tract)
        println!("Loading {} bytes of model weights", weights.len());
        
        // Simulate weight validation
        if weights.len() < 1024 {
            return Err(anyhow!("Model weights too small: {} bytes", weights.len()));
        }
        
        self.model_weights = Some(weights);
        
        // Set default config for simulation
        self.model_config = Some(ModelConfig {
            model_type: "llama".to_string(),
            vocab_size: 32000,
            hidden_size: 4096,
            num_layers: 32,
            max_sequence_length: 2048,
            temperature: 0.7,
        });
        
        println!("✅ Model weights loaded successfully");
        Ok(())
    }

    /// Set model configuration
    pub fn set_config(&mut self, config: ModelConfig) -> Result<()> {
        self.model_config = Some(config);
        Ok(())
    }

    /// Update context window
    pub fn update_context(&mut self, context: &str) -> Result<()> {
        self.context_window.push(context.to_string());
        
        // Keep only last 10 context entries for memory efficiency
        if self.context_window.len() > 10 {
            self.context_window.remove(0);
        }
        
        Ok(())
    }

    /// Update memory state
    pub fn update_memory(&mut self, memory_policy: &MemoryPolicy) -> Result<()> {
        // Update short-term memory based on window size
        if memory_policy.short_term_window_size > 0 {
            self.memory_state.short_term.extend(self.context_window.clone());

            // Keep only recent entries based on window size
            if self.memory_state.short_term.len() > memory_policy.short_term_window_size as usize {
                let excess = self.memory_state.short_term.len() - memory_policy.short_term_window_size as usize;
                self.memory_state.short_term.drain(0..excess);
            }
        }

        // Simulate episodic memory updates based on top-k
        if memory_policy.episodic_top_k > 0 {
            let episode_key = format!("episode_{}", self.memory_state.episodic.len());
            let episode_value = self.context_window.join(" ");
            self.memory_state.episodic.insert(episode_key, episode_value);

            // Keep only top-k episodic entries
            if self.memory_state.episodic.len() > memory_policy.episodic_top_k as usize {
                // Remove oldest entries (simple FIFO for simulation)
                let keys_to_remove: Vec<_> = self.memory_state.episodic.keys()
                    .take(self.memory_state.episodic.len() - memory_policy.episodic_top_k as usize)
                    .cloned()
                    .collect();
                for key in keys_to_remove {
                    self.memory_state.episodic.remove(&key);
                }
            }
        }

        Ok(())
    }

    /// Run inference on the loaded model
    pub fn run_inference(&mut self, input: &InferenceInput) -> Result<InferenceOutput> {
        // Validate model is loaded
        if self.model_weights.is_none() {
            return Err(anyhow!("No model weights loaded"));
        }

        // Update context and memory first
        self.update_context(&input.context)?;

        if let Some(memory_policy) = &input.memory_policy {
            self.update_memory(memory_policy)?;
        }

        // Get config after mutations
        let config = self.model_config.as_ref()
            .ok_or_else(|| anyhow!("No model config set"))?;

        // Simulate inference process
        let start_time = std::time::Instant::now();

        // In a real implementation, this would:
        // 1. Tokenize the input prompt
        // 2. Run forward pass through the neural network
        // 3. Sample from the output distribution
        // 4. Decode tokens back to text

        let response = self.simulate_inference(&input.prompt, config)?;

        let processing_time = start_time.elapsed().as_millis() as f32;

        Ok(InferenceOutput {
            response,
            confidence: 0.85, // Simulated confidence score
            processing_time_ms: processing_time,
            debug_info: vec![
                format!("Model: {}", config.model_type),
                format!("Context length: {}", self.context_window.len()),
                format!("Memory entries: {}", self.memory_state.short_term.len()),
            ],
        })
    }

    /// Simulate the inference process
    fn simulate_inference(&self, prompt: &str, _config: &ModelConfig) -> Result<String> {
        // This is a simulation - in reality would run actual ML inference
        let responses = vec![
            "I understand your request and will help you with that.",
            "Based on the context provided, here's my analysis:",
            "Let me think about this step by step.",
            "That's an interesting question. Here's what I think:",
            "I can help you solve this problem.",
        ];
        
        // Simple hash-based selection for deterministic responses
        let hash = prompt.chars().map(|c| c as u32).sum::<u32>();
        let index = (hash % responses.len() as u32) as usize;
        
        let base_response = responses[index];
        
        // Add some context-aware elements
        let context_info = if !self.context_window.is_empty() {
            format!(" (Context: {} previous interactions)", self.context_window.len())
        } else {
            String::new()
        };
        
        let memory_info = if !self.memory_state.short_term.is_empty() {
            format!(" [Memory: {} items]", self.memory_state.short_term.len())
        } else {
            String::new()
        };
        
        Ok(format!("{}{}{}", base_response, context_info, memory_info))
    }

    /// Get runtime statistics
    pub fn get_stats(&self) -> HashMap<String, String> {
        let mut stats = HashMap::new();
        
        stats.insert("weights_loaded".to_string(), 
                    self.model_weights.is_some().to_string());
        
        if let Some(config) = &self.model_config {
            stats.insert("model_type".to_string(), config.model_type.clone());
            stats.insert("vocab_size".to_string(), config.vocab_size.to_string());
            stats.insert("hidden_size".to_string(), config.hidden_size.to_string());
        }
        
        stats.insert("context_length".to_string(), self.context_window.len().to_string());
        stats.insert("memory_entries".to_string(), self.memory_state.short_term.len().to_string());
        
        stats
    }

    /// Reset the runtime state
    pub fn reset(&mut self) {
        self.context_window.clear();
        self.memory_state = MemoryState {
            short_term: Vec::new(),
            episodic: HashMap::new(),
            semantic: HashMap::new(),
            procedural: Vec::new(),
        };
    }
}

// C-style exports for static linking into WASM
#[no_mangle]
pub extern "C" fn inner_runtime_new() -> *mut InnerModelRuntime {
    Box::into_raw(Box::new(InnerModelRuntime::new()))
}

#[no_mangle]
pub extern "C" fn inner_runtime_free(runtime: *mut InnerModelRuntime) {
    if !runtime.is_null() {
        unsafe {
            let _ = Box::from_raw(runtime);
        }
    }
}

#[no_mangle]
pub extern "C" fn inner_runtime_load_weights(
    runtime: *mut InnerModelRuntime,
    weights_ptr: *const u8,
    weights_len: usize,
) -> i32 {
    if runtime.is_null() || weights_ptr.is_null() {
        return -1;
    }
    
    unsafe {
        let runtime = &mut *runtime;
        let weights = std::slice::from_raw_parts(weights_ptr, weights_len).to_vec();
        
        match runtime.load_weights(weights) {
            Ok(_) => 0,
            Err(_) => -1,
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_runtime_creation() {
        let runtime = InnerModelRuntime::new();
        assert!(runtime.model_weights.is_none());
        assert!(runtime.model_config.is_none());
    }

    #[test]
    fn test_weight_loading() {
        let mut runtime = InnerModelRuntime::new();
        let dummy_weights = vec![0u8; 2048]; // 2KB of dummy weights
        
        let result = runtime.load_weights(dummy_weights);
        assert!(result.is_ok());
        assert!(runtime.model_weights.is_some());
        assert!(runtime.model_config.is_some());
    }

    #[test]
    fn test_inference_without_weights() {
        let mut runtime = InnerModelRuntime::new();
        let input = InferenceInput {
            prompt: "Hello".to_string(),
            context: "test".to_string(),
            config: None,
            memory_policy: None,
        };
        
        let result = runtime.run_inference(&input);
        assert!(result.is_err());
    }

    #[test]
    fn test_inference_with_weights() {
        let mut runtime = InnerModelRuntime::new();
        let dummy_weights = vec![0u8; 2048];
        runtime.load_weights(dummy_weights).unwrap();
        
        let input = InferenceInput {
            prompt: "Hello world".to_string(),
            context: "test context".to_string(),
            config: None,
            memory_policy: Some(MemoryPolicy {
                short_term_enabled: true,
                short_term_limit: 100,
                episodic_enabled: true,
                episodic_limit: 50,
                semantic_enabled: false,
                semantic_limit: 0,
                procedural_enabled: false,
                procedural_limit: 0,
            }),
        };
        
        let result = runtime.run_inference(&input);
        assert!(result.is_ok());
        
        let output = result.unwrap();
        assert!(!output.response.is_empty());
        assert!(output.confidence > 0.0);
        assert!(output.processing_time_ms >= 0.0);
    }
}
