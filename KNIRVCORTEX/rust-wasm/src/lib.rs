use wasm_bindgen::prelude::*;
use shared_types::*;
use shared_types::abi::*;
// InnerModelRuntime is now defined inline for WASM compatibility
use prost::Message;
use std::collections::HashMap;

// Import the `console.log` function from the browser
#[wasm_bindgen]
extern "C" {
    #[wasm_bindgen(js_namespace = console)]
    fn log(s: &str);
}

// Define a macro to make console.log easier to use
macro_rules! console_log {
    ($($t:tt)*) => (log(&format_args!($($t)*).to_string()))
}

// KNIRV-CORTEX: Deterministic cognitive shell with ProtoBuf ABI
#[wasm_bindgen]
pub struct KnirvCortex {
    // Configuration and policies
    config: Option<Config>,
    memory_policy: Option<MemoryPolicy>,

    // Context and tools
    context: Option<Context>,
    tools: Option<ToolList>,

    // Inner model runtime (linked statically)
    inner_runtime: InnerModelRuntime,

    // State management
    initialized: bool,
}

// Inner Model Runtime - statically linked into cortex.wasm
pub struct InnerModelRuntime {
    model_weights: Option<Vec<u8>>,
    model_loaded: bool,
    context_window: Vec<String>,
}

impl InnerModelRuntime {
    fn new() -> Self {
        Self {
            model_weights: None,
            model_loaded: false,
            context_window: Vec::new(),
        }
    }

    fn load_weights(&mut self, weights: Vec<u8>) -> Result<(), String> {
        if weights.len() < 1024 {
            return Err("Model weights too small".to_string());
        }

        console_log!("Loading {} bytes of model weights", weights.len());
        self.model_weights = Some(weights);
        self.model_loaded = true;
        Ok(())
    }

    fn run_inference(&mut self, input: &InferenceInput) -> Result<InferenceOutput, String> {
        if !self.model_loaded {
            return Err("No model weights loaded".to_string());
        }

        // Update context
        self.context_window.push(input.context.clone());
        if self.context_window.len() > 10 {
            self.context_window.remove(0);
        }

        let start_time = js_sys::Date::now();

        // Simulate inference
        let response = format!(
            "Processed '{}' with {} context entries",
            input.prompt.chars().take(50).collect::<String>(),
            self.context_window.len()
        );

        let processing_time = (js_sys::Date::now() - start_time) as f32;

        Ok(InferenceOutput {
            response,
            confidence: 0.85,
            processing_time_ms: processing_time,
            debug_info: vec![
                format!("Context entries: {}", self.context_window.len()),
                format!("Model loaded: {}", self.model_loaded),
            ],
        })
    }
}

// Old InnerModelRuntime implementation removed - now using inner-runtime crate
/*
impl InnerModelRuntime {
    fn new() -> Self {
        Self {
            weights: Vec::new(),
            model_loaded: false,
            runtime_config: None,
            l_modules: Vec::new(),
            h_modules: Vec::new(),
        }
    }

    fn initialize_modules(&mut self, l_count: u32, h_count: u32) {
        console_log!("Initializing {} L-modules and {} H-modules", l_count, h_count);

        // Initialize L-modules
        self.l_modules = (0..l_count).map(|i| LModule {
            id: i,
            weights: vec![0.0; 1000], // Placeholder weights
            activation: 0.0,
        }).collect();

        // Initialize H-modules
        self.h_modules = (0..h_count).map(|i| HModule {
            id: i,
            weights: vec![0.0; 2000], // Placeholder weights
            planning_depth: (i + 1) * 4,
            activation: 0.0,
        }).collect();
    }

    fn load_weights(&mut self, weights_data: &[u8]) -> bool {
        console_log!("Loading model weights... {} bytes", weights_data.len());

        if weights_data.len() >= 1024 {
            // In production, this would parse safetensors format
            self.weights.resize(weights_data.len() / 4, 0.0);

            // Convert bytes to f32 weights
            for (i, chunk) in weights_data.chunks(4).enumerate() {
                if i < self.weights.len() && chunk.len() == 4 {
                    let bytes = [chunk[0], chunk[1], chunk[2], chunk[3]];
                    self.weights[i] = f32::from_le_bytes(bytes);
                }
            }

            self.model_loaded = true;
            console_log!("Model weights loaded successfully: {} parameters", self.weights.len());
            true
        } else {
            console_log!("Invalid weights data - too small");
            false
        }
    }

    fn infer(&mut self, input: &InferenceInput) -> Result<InferenceOutput, CortexError> {
        if !self.model_loaded {
            return Err(CortexError::new(
                error_codes::MODEL_NOT_LOADED,
                "Model weights not loaded"
            ));
        }

        let start_time = js_sys::Date::now();

        // Simulate processing through L and H modules
        let mut l_activations = Vec::new();
        for l_module in &mut self.l_modules {
            // Simple activation based on prompt length and module id
            let activation = (input.prompt.len() as f32 / 100.0) * (l_module.id as f32 + 1.0) / 10.0;
            l_module.activation = activation;
            l_activations.push(activation);
        }

        let mut h_activations = Vec::new();
        for h_module in &mut self.h_modules {
            let activation = l_activations.iter().sum::<f32>() / (h_module.planning_depth as f32);
            h_module.activation = activation;
            h_activations.push(activation);
        }

        // Generate response
        let response = format!(
            "Processed '{}' through {} L-modules and {} H-modules",
            input.prompt.chars().take(50).collect::<String>(),
            self.l_modules.len(),
            self.h_modules.len()
        );

        let confidence = self.calculate_confidence(&l_activations, &h_activations);
        let processing_time = (js_sys::Date::now() - start_time) as f32;

        Ok(InferenceOutput {
            response,
            confidence,
            processing_time_ms: processing_time,
            debug_info: vec![
                format!("L-modules: {}", l_activations.len()),
                format!("H-modules: {}", h_activations.len()),
                format!("Weights: {}", self.weights.len()),
            ],
        })
    }

    fn calculate_confidence(&self, l_activations: &[f32], h_activations: &[f32]) -> f32 {
        if l_activations.is_empty() || h_activations.is_empty() {
            return 0.0;
        }

        let l_avg = l_activations.iter().sum::<f32>() / l_activations.len() as f32;
        let h_avg = h_activations.iter().sum::<f32>() / h_activations.len() as f32;

        ((l_avg + h_avg) / 2.0).min(1.0).max(0.0)
    }
}
*/

// Global memory allocator for WASM
static mut MEMORY_BUFFER: Vec<u8> = Vec::new();

// Memory management for ABI boundary
fn allocate_buffer(size: usize) -> *mut u8 {
    unsafe {
        MEMORY_BUFFER.resize(size, 0);
        MEMORY_BUFFER.as_mut_ptr()
    }
}

fn deallocate_buffer() {
    unsafe {
        MEMORY_BUFFER.clear();
    }
}

#[wasm_bindgen]
impl KnirvCortex {
    #[wasm_bindgen(constructor)]
    pub fn new() -> KnirvCortex {
        console_log!("Initializing KNIRV-CORTEX with ProtoBuf ABI...");

        KnirvCortex {
            config: None,
            memory_policy: None,
            context: None,
            tools: None,
            inner_runtime: InnerModelRuntime::new(),
            initialized: false,
        }
    }

    /// Core ABI export: run_cognitive_task(prompt_ptr, prompt_len) -> u64
    /// Input: UTF-8 prompt; cortex assembles full request using configured memory policy
    /// Returns packed pointer/len to serialized InferenceOutput (ProtoBuf bytes)
    #[wasm_bindgen]
    pub fn run_cognitive_task(&mut self, prompt_ptr: u32, prompt_len: u32) -> u64 {
        console_log!("run_cognitive_task called with ptr={}, len={}", prompt_ptr, prompt_len);

        // Read prompt from memory
        let prompt_bytes = unsafe {
            std::slice::from_raw_parts(prompt_ptr as *const u8, prompt_len as usize)
        };

        let prompt = match std::str::from_utf8(prompt_bytes) {
            Ok(s) => s,
            Err(_) => {
                let error = CortexError::new(error_codes::INVALID_INPUT, "Invalid UTF-8 prompt");
                let envelope = Envelope::error(error);
                return self.return_envelope(envelope);
            }
        };

        // Assemble full request using memory policy
        let input = self.assemble_inference_input(prompt);

        // Call inner runtime (no cross-module import - direct function call)
        let result = self.inner_runtime.run_inference(&input)
            .map_err(|e| CortexError::new(error_codes::RUNTIME_ERROR, &e));

        // Create envelope response
        let envelope = match result {
            Ok(output) => Envelope::ok(output.encode_to_vec()),
            Err(error) => Envelope::error(error),
        };

        self.return_envelope(envelope)
    }

    /// Set context for cognitive processing
    #[wasm_bindgen]
    pub fn set_context(&mut self, ptr: u32, len: u32) -> bool {
        console_log!("set_context called with ptr={}, len={}", ptr, len);

        let context_bytes = unsafe {
            std::slice::from_raw_parts(ptr as *const u8, len as usize)
        };

        match Context::decode(context_bytes) {
            Ok(context) => {
                self.context = Some(context);
                console_log!("Context set successfully");
                true
            }
            Err(_) => {
                console_log!("Failed to decode context");
                false
            }
        }
    }

    /// Register available tools
    #[wasm_bindgen]
    pub fn set_tools(&mut self, ptr: u32, len: u32) -> bool {
        console_log!("set_tools called with ptr={}, len={}", ptr, len);

        let tools_bytes = unsafe {
            std::slice::from_raw_parts(ptr as *const u8, len as usize)
        };

        match ToolList::decode(tools_bytes) {
            Ok(tools) => {
                self.tools = Some(tools);
                console_log!("Tools set successfully");
                true
            }
            Err(_) => {
                console_log!("Failed to decode tools");
                false
            }
        }
    }

    /// Initialize the cortex with configuration
    #[wasm_bindgen]
    pub fn initialize(&mut self, config_ptr: u32, config_len: u32) -> bool {
        console_log!("initialize called with ptr={}, len={}", config_ptr, config_len);

        let config_bytes = unsafe {
            std::slice::from_raw_parts(config_ptr as *const u8, config_len as usize)
        };

        match Config::decode(config_bytes) {
            Ok(config) => {
                self.config = Some(config);
                // Inner runtime is already initialized
                self.initialized = true;
                console_log!("Cortex initialized successfully");
                true
            }
            Err(_) => {
                console_log!("Failed to decode config");
                false
            }
        }
    }

    /// Load model weights into the inner runtime
    #[wasm_bindgen]
    pub fn load_weights(&mut self, weights_ptr: u32, weights_len: u32) -> bool {
        console_log!("load_weights called with ptr={}, len={}", weights_ptr, weights_len);

        let weights_bytes = unsafe {
            std::slice::from_raw_parts(weights_ptr as *const u8, weights_len as usize)
        };

        match self.inner_runtime.load_weights(weights_bytes.to_vec()) {
            Ok(_) => {
                console_log!("Model weights loaded successfully");
                true
            }
            Err(e) => {
                console_log!("Failed to load weights: {}", e);
                false
            }
        }
    }

    // Helper methods for the new ProtoBuf-based implementation

    fn assemble_inference_input(&self, prompt: &str) -> InferenceInput {
        let context = if let Some(ctx) = &self.context {
            format!("Short-term: {}\nEpisodic: {} items\nSemantic: {} triples\nTools: {} available",
                ctx.short_term,
                ctx.episodic_items.len(),
                ctx.semantic_triples.len(),
                ctx.available_tools.len()
            )
        } else {
            "No context available".to_string()
        };

        InferenceInput {
            prompt: prompt.to_string(),
            context,
            config: self.config.clone(),
            memory_policy: self.memory_policy.clone(),
        }
    }

    fn return_envelope(&self, envelope: Envelope) -> u64 {
        let envelope_bytes = envelope.encode_to_vec();
        let len = envelope_bytes.len();

        // Allocate buffer and copy data
        let ptr = allocate_buffer(len);
        unsafe {
            std::ptr::copy_nonoverlapping(envelope_bytes.as_ptr(), ptr, len);
        }

        pack_ptr_len(ptr as *const u8, len)
    }

    /// Get information about the loaded model
    #[wasm_bindgen]
    pub fn get_model_info(&self) -> String {
        let weights_len = self.inner_runtime.model_weights.as_ref().map(|w| w.len()).unwrap_or(0);
        let info = format!(
            "{{\"total_parameters\": {}, \"context_entries\": {}, \"initialized\": {}}}",
            weights_len / 4, // Assuming f32 weights
            self.inner_runtime.context_window.len(),
            self.initialized
        );
        info
    }


    /// Get information about loaded weights
    #[wasm_bindgen]
    pub fn get_weights_info(&self) -> String {
        let weights_len = self.inner_runtime.model_weights.as_ref().map(|w| w.len()).unwrap_or(0);
        let info = format!(
            "{{\"total_parameters\": {}, \"memory_usage_mb\": {:.2}, \"loaded\": {}}}",
            weights_len / 4, // Assuming f32 weights
            (weights_len * 4) as f64 / (1024.0 * 1024.0),
            self.inner_runtime.model_loaded
        );
        info
    }

}


// Initialize the KNIRV-CORTEX WASM module
#[wasm_bindgen(start)]
pub fn main() {
    console_log!("KNIRV-CORTEX WASM module initialized with ProtoBuf ABI");
}
