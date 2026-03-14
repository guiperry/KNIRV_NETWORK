use wasm_bindgen::prelude::*;
use shared_types::*;
use shared_types::abi::*;
use shared_types::error_codes;
use lora_engine::LoRAAdapterEngine;
use prost::Message;
use std::collections::HashMap;

mod external_inference;
use external_inference::{ExternalInferenceEngine, ExternalInferenceRequest};

mod heart_client;
use heart_client::{HEARTClient, HEARTConfig, HEARTErrorInquiry, HEARTHeuristicResponse};

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

// KNIRV-CORTEX: Deterministic cognitive shell with ProtoBuf ABI and HEART integration
#[wasm_bindgen]
pub struct KnirvCortex {
    // Configuration and policies
    config: Option<Config>,
    memory_policy: Option<MemoryPolicy>,

    // Context and tools
    context: Option<Context>,
    tools: Option<ToolList>,

    // Inner model runtime (stem.cortex)
    inner_runtime: InnerModelRuntime,

    // LoRA Adapter Engine
    lora_engine: LoRAAdapterEngine,

    // External Inference Engine
    external_inference: Option<ExternalInferenceEngine>,

    // HEART Client for error analysis
    heart_client: Option<HEARTClient>,

    // State management
    initialized: bool,

    // Error tracking
    error_history: Vec<HEARTErrorInquiry>,
    heuristic_cache: HashMap<String, HEARTHeuristicResponse>,
}

// Inner Model Runtime (stem.cortex) - statically linked into cortex.wasm
// Enhanced with HEART integration for intelligent error handling
pub struct InnerModelRuntime {
    model_weights: Option<Vec<u8>>,
    model_loaded: bool,
    context_window: Vec<String>,

    // HEART integration state
    error_count: u32,
    last_heart_query_time: f64,
    heart_query_count: u32,
}

impl InnerModelRuntime {
    fn new() -> Self {
        Self {
            model_weights: None,
            model_loaded: false,
            context_window: Vec::new(),
            error_count: 0,
            last_heart_query_time: 0.0,
            heart_query_count: 0,
        }
    }

    fn load_weights(&mut self, weights: Vec<u8>) -> Result<(), String> {
        if weights.len() < 1024 {
            return Err("Model weights too small".to_string());
        }

        console_log!("[stem.cortex] Loading {} bytes of model weights", weights.len());
        self.model_weights = Some(weights);
        self.model_loaded = true;
        Ok(())
    }

    fn run_inference(&mut self, input: &InferenceInput) -> Result<InferenceOutput, String> {
        if !self.model_loaded {
            let error_msg = "No model weights loaded";
            console_log!("[stem.cortex] ERROR: {}", error_msg);
            self.error_count += 1;
            return Err(error_msg.to_string());
        }

        // Update context
        self.context_window.push(input.context.clone());
        if self.context_window.len() > 10 {
            self.context_window.remove(0);
        }

        let start_time = js_sys::Date::now();

        // Simulate inference (in production, this would call actual model)
        let response = self.simulate_inference(input);

        let processing_time = (js_sys::Date::now() - start_time) as f32;

        Ok(InferenceOutput {
            response: response.clone(),
            confidence: 0.85,
            processing_time_ms: processing_time,
            debug_info: vec![
                format!("Context entries: {}", self.context_window.len()),
                format!("Model loaded: {}", self.model_loaded),
                format!("HEART queries: {}", self.heart_query_count),
            ],
        })
    }

    fn simulate_inference(&self, input: &InferenceInput) -> String {
        format!(
            "[stem.cortex] Processed '{}' with {} context entries",
            input.prompt.chars().take(50).collect::<String>(),
            self.context_window.len()
        )
    }

    /// Create error inquiry for HEART when encountering issues
    fn create_heart_inquiry(&self, error_type: &str, error_msg: &str, context: &str) -> HEARTErrorInquiry {
        let error_id = format!("cortex-{}-{}", self.error_count, js_sys::Date::now() as u64);

        HEARTErrorInquiry {
            error_id,
            error_type: error_type.to_string(),
            error_message: error_msg.to_string(),
            error_context: context.to_string(),
            stack_trace: None,
            metadata: [
                ("model_loaded".to_string(), self.model_loaded.to_string()),
                ("context_size".to_string(), self.context_window.len().to_string()),
                ("error_count".to_string(), self.error_count.to_string()),
            ].iter().cloned().collect(),
            prompt: None,
            model_response: None,
            confidence_score: None,
            timestamp: js_sys::Date::now() as u64,
        }
    }

    fn should_query_heart(&self) -> bool {
        // Rate limiting: query HEART at most once per second
        let now = js_sys::Date::now();
        let time_since_last_query = now - self.last_heart_query_time;
        time_since_last_query > 1000.0
    }
}

// Global memory allocator for WASM
static mut MEMORY_BUFFER: Vec<u8> = Vec::new();

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
        console_log!("[cortex.wasm] Initializing KNIRV-CORTEX with HEART integration...");

        KnirvCortex {
            config: None,
            memory_policy: None,
            context: None,
            tools: None,
            inner_runtime: InnerModelRuntime::new(),
            lora_engine: LoRAAdapterEngine::new(),
            external_inference: None,
            heart_client: None,
            initialized: false,
            error_history: Vec::new(),
            heuristic_cache: HashMap::new(),
        }
    }

    /// Initialize HEART client for error analysis
    #[wasm_bindgen]
    pub fn init_heart_client(&mut self, config_json: &str) -> Result<(), JsValue> {
        console_log!("[cortex.wasm] Initializing HEART client...");

        let config: HEARTConfig = serde_json::from_str(config_json)
            .map_err(|e| JsValue::from_str(&format!("Invalid HEART config: {}", e)))?;

        self.heart_client = Some(HEARTClient::new(config));
        console_log!("[cortex.wasm] HEART client initialized successfully");

        Ok(())
    }

    /// Core ABI export: run_cognitive_task with HEART-enhanced error handling
    #[wasm_bindgen]
    pub async fn run_cognitive_task(&mut self, prompt_ptr: u32, prompt_len: u32) -> u64 {
        console_log!("[cortex.wasm] run_cognitive_task called (HEART-enhanced)");

        // Read prompt from memory
        let prompt_bytes = unsafe {
            std::slice::from_raw_parts(prompt_ptr as *const u8, prompt_len as usize)
        };

        let prompt = match std::str::from_utf8(prompt_bytes) {
            Ok(s) => s,
            Err(_) => {
                let error = CortexError::new(error_codes::INVALID_INPUT, "Invalid UTF-8 prompt");

                // Query HEART for error analysis
                if let Some(heart_response) = self.query_heart_for_error(
                    "UTF8Error",
                    "Invalid UTF-8 in prompt",
                    &format!("Prompt length: {}", prompt_len)
                ).await {
                    console_log!("[cortex.wasm] HEART analysis: {}", heart_response.analysis_summary);
                    self.apply_heart_heuristics(&heart_response);
                }

                let envelope = Envelope::error(error);
                return self.return_envelope(envelope);
            }
        };

        // Assemble full request using memory policy
        let input = self.assemble_inference_input(prompt);

        // Call stem.cortex (inner runtime)
        let result = self.inner_runtime.run_inference(&input);

        // Handle errors with HEART guidance
        let final_result = match result {
            Ok(output) => Ok(output),
            Err(e) => {
                console_log!("[cortex.wasm] stem.cortex error: {}", e);

                // Query HEART for heuristic guidance
                if let Some(heart_response) = self.query_heart_for_error(
                    "InferenceError",
                    &e,
                    &format!("Prompt: {}", prompt)
                ).await {
                    console_log!("[cortex.wasm] HEART heuristics received:");
                    console_log!("  Alert Level: {}", heart_response.alert_level);
                    console_log!("  Heuristic ID: {}", heart_response.heuristic_id);
                    console_log!("  Confidence: {:.2}", heart_response.confidence_score);
                    console_log!("  Analysis: {}", heart_response.analysis_summary);

                    // Apply HEART's recommended actions
                    let recovery_result = self.apply_heart_heuristics(&heart_response);

                    // Retry inference if HEART suggests it
                    if heart_response.recommended_actions.iter().any(|a| a.contains("retry")) {
                        console_log!("[cortex.wasm] HEART recommends retry, attempting...");
                        match self.inner_runtime.run_inference(&input) {
                            Ok(output) => Ok(output),
                            Err(retry_error) => {
                                Err(CortexError::new(
                                    error_codes::RUNTIME_ERROR,
                                    &format!("Inference failed after HEART-guided retry: {}", retry_error)
                                ))
                            }
                        }
                    } else {
                        Err(CortexError::new(
                            error_codes::RUNTIME_ERROR,
                            &format!("{} (HEART analysis: {})", e, heart_response.analysis_summary)
                        ))
                    }
                } else {
                    console_log!("[cortex.wasm] HEART unavailable, using local error handling");
                    Err(CortexError::new(error_codes::RUNTIME_ERROR, &e))
                }
            }
        };

        // Create envelope response
        let envelope = match final_result {
            Ok(output) => Envelope::ok(output.encode_to_vec()),
            Err(error) => Envelope::error(error),
        };

        self.return_envelope(envelope)
    }

    /// Query HEART for error analysis and heuristics
    async fn query_heart_for_error(&mut self, error_type: &str, error_msg: &str, context: &str) -> Option<HEARTHeuristicResponse> {
        // Check if HEART client is initialized
        let heart_client = match self.heart_client.as_mut() {
            Some(client) => client,
            None => {
                console_log!("[cortex.wasm] HEART client not initialized");
                return None;
            }
        };

        // Check rate limiting
        if !self.inner_runtime.should_query_heart() {
            console_log!("[cortex.wasm] HEART rate limit exceeded, skipping query");
            return None;
        }

        // Create inquiry
        let inquiry = self.inner_runtime.create_heart_inquiry(error_type, error_msg, context);
        let inquiry_id = inquiry.error_id.clone();

        // Check cache
        if let Some(cached) = self.heuristic_cache.get(&inquiry_id) {
            console_log!("[cortex.wasm] Using cached HEART response");
            return Some(cached.clone());
        }

        // Add to history
        self.error_history.push(inquiry.clone());
        if self.error_history.len() > 100 {
            self.error_history.remove(0);
        }

        // Query HEART
        console_log!("[cortex.wasm] Querying HEART for error analysis...");
        self.inner_runtime.last_heart_query_time = js_sys::Date::now();
        self.inner_runtime.heart_query_count += 1;

        match heart_client.query_heart(inquiry).await {
            Ok(response) => {
                // Cache the response
                self.heuristic_cache.insert(inquiry_id, response.clone());
                if self.heuristic_cache.len() > 50 {
                    // Simple cache eviction
                    if let Some(first_key) = self.heuristic_cache.keys().next().cloned() {
                        self.heuristic_cache.remove(&first_key);
                    }
                }

                Some(response)
            }
            Err(e) => {
                console_log!("[cortex.wasm] HEART query failed: {}", e);
                None
            }
        }
    }

    /// Apply HEART's heuristic recommendations
    fn apply_heart_heuristics(&mut self, response: &HEARTHeuristicResponse) -> Result<(), String> {
        console_log!("[cortex.wasm] Applying HEART heuristics...");

        // Process command vector
        if response.command_vector.len() >= 8 {
            let alert_level = response.command_vector[0] as u32;
            let heuristic_id = response.command_vector[1] as u32;
            let confidence = response.command_vector[3];

            console_log!("  Command vector: alert={}, heuristic={}, confidence={:.2}",
                        alert_level, heuristic_id, confidence);

            // Apply specific heuristics based on heuristic_id
            match heuristic_id {
                101 => {
                    // Type error handling
                    console_log!("  Applying type validation heuristic");
                    // In production: adjust model inputs, add type checking
                }
                201 => {
                    // Network error handling
                    console_log!("  Applying network retry heuristic");
                    // In production: implement exponential backoff, fallback endpoints
                }
                301 => {
                    // Model error handling
                    console_log!("  Applying model recovery heuristic");
                    // In production: reload weights, reset state
                }
                _ => {
                    console_log!("  Applying generic error mitigation");
                }
            }
        }

        // Log recommended actions
        for (i, action) in response.recommended_actions.iter().enumerate() {
            console_log!("  Action {}: {}", i + 1, action);
        }

        // Log identified patterns
        for pattern in &response.identified_patterns {
            console_log!("  Pattern detected: {} (confidence: {:.2})",
                        pattern.pattern_name, pattern.match_confidence);
        }

        // Log similar errors
        for similar in &response.similar_errors {
            console_log!("  Similar error: {} (similarity: {:.2})",
                        similar.error_id, similar.similarity_score);
            if let Some(skill_id) = &similar.skill_id {
                console_log!("    Resolved by skill: {}", skill_id);
                // In production: could auto-load and apply this LoRA adapter
            }
        }

        Ok(())
    }

    /// Get HEART client statistics
    #[wasm_bindgen]
    pub fn get_heart_stats(&self) -> String {
        if let Some(client) = &self.heart_client {
            client.get_stats()
        } else {
            "{}".to_string()
        }
    }

    /// Get error history
    #[wasm_bindgen]
    pub fn get_error_history(&self) -> String {
        serde_json::json!({
            "total_errors": self.error_history.len(),
            "cached_heuristics": self.heuristic_cache.len(),
            "recent_errors": self.error_history.iter().rev().take(10).map(|e| {
                serde_json::json!({
                    "error_id": e.error_id,
                    "error_type": e.error_type,
                    "timestamp": e.timestamp
                })
            }).collect::<Vec<_>>()
        }).to_string()
    }

    /// Check HEART service health
    #[wasm_bindgen]
    pub async fn check_heart_health(&self) -> Result<bool, JsValue> {
        if let Some(client) = &self.heart_client {
            client.health_check().await
                .map_err(|e| JsValue::from_str(&e))
        } else {
            Err(JsValue::from_str("HEART client not initialized"))
        }
    }

    // [Rest of the existing methods remain the same...]

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

    #[wasm_bindgen]
    pub fn initialize(&mut self, config_ptr: u32, config_len: u32) -> bool {
        console_log!("initialize called with ptr={}, len={}", config_ptr, config_len);

        let config_bytes = unsafe {
            std::slice::from_raw_parts(config_ptr as *const u8, config_len as usize)
        };

        match Config::decode(config_bytes) {
            Ok(config) => {
                self.config = Some(config);
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

        let ptr = allocate_buffer(len);
        unsafe {
            std::ptr::copy_nonoverlapping(envelope_bytes.as_ptr(), ptr, len);
        }

        pack_ptr_len(ptr as *const u8, len)
    }

    #[wasm_bindgen]
    pub fn get_model_info(&self) -> String {
        let weights_len = self.inner_runtime.model_weights.as_ref().map(|w| w.len()).unwrap_or(0);
        let info = format!(
            "{{\"total_parameters\": {}, \"context_entries\": {}, \"initialized\": {}, \"heart_enabled\": {}}}",
            weights_len / 4,
            self.inner_runtime.context_window.len(),
            self.initialized,
            self.heart_client.is_some()
        );
        info
    }

    #[wasm_bindgen]
    pub fn get_weights_info(&self) -> String {
        let weights_len = self.inner_runtime.model_weights.as_ref().map(|w| w.len()).unwrap_or(0);
        let info = format!(
            "{{\"total_parameters\": {}, \"memory_usage_mb\": {:.2}, \"loaded\": {}}}",
            weights_len / 4,
            (weights_len * 4) as f64 / (1024.0 * 1024.0),
            self.inner_runtime.model_loaded
        );
        info
    }
}

// Initialize the KNIRV-CORTEX WASM module
#[wasm_bindgen(start)]
pub fn main() {
    console_log!("KNIRV-CORTEX WASM module initialized with HEART integration");
}
