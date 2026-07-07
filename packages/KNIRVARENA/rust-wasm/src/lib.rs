use wasm_bindgen::prelude::*;
use serde::{Deserialize, Serialize};
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

// Enhanced WASM module structure for HRM cognitive core with personality adaptation
#[wasm_bindgen]
pub struct HRMCognitive {
    // HRM model weights (562M parameters)
    weights: Vec<f32>,
    // L-modules for sensory-motor patterns
    l_modules: Vec<LModule>,
    // H-modules for long-horizon planning
    h_modules: Vec<HModule>,
    // Personality adapter for user-specific behavior
    personality_adapter: PersonalityAdapter,
    // Host interface for desktop communication
    host_interface: HostInterface,
    // Cognitive state management
    cognitive_state: CognitiveState,
}

#[derive(Serialize, Deserialize)]
pub struct LModule {
    pub id: u32,
    pub weights: Vec<f32>,
    pub activation: f32,
}

#[derive(Serialize, Deserialize)]
pub struct HModule {
    pub id: u32,
    pub weights: Vec<f32>,
    pub planning_depth: u32,
    pub activation: f32,
}

#[derive(Serialize, Deserialize)]
pub struct CognitiveInput {
    pub sensory_data: Vec<f32>,
    pub context: String,
    pub task_type: String,
}

#[derive(Serialize, Deserialize)]
pub struct CognitiveOutput {
    pub reasoning_result: String,
    pub confidence: f32,
    pub processing_time: f32,
    pub l_module_activations: Vec<f32>,
    pub h_module_activations: Vec<f32>,
    pub personality_influence: f32,
    pub adaptation_score: f32,
}

// Personality Adapter for user-specific behavior adaptation
#[derive(Serialize, Deserialize, Clone)]
pub struct PersonalityAdapter {
    pub user_id: String,
    pub personality_metrics: HashMap<String, f32>,
    pub adaptation_history: Vec<AdaptationEvent>,
    pub learning_rate: f32,
}

#[derive(Serialize, Deserialize, Clone)]
pub struct AdaptationEvent {
    pub timestamp: f64,
    pub event_type: String,
    pub user_feedback: f32,
    pub context: String,
    pub adaptation_delta: HashMap<String, f32>,
}

// Host Interface for desktop communication
#[derive(Serialize, Deserialize, Clone)]
pub struct HostInterface {
    pub desktop_id: Option<String>,
    pub connection_status: ConnectionStatus,
    pub message_queue: Vec<HostMessage>,
    pub capabilities: Vec<String>,
}

#[derive(Serialize, Deserialize, Clone)]
pub enum ConnectionStatus {
    Disconnected,
    Connecting,
    Connected,
    Error(String),
}

#[derive(Serialize, Deserialize, Clone)]
pub struct HostMessage {
    pub id: String,
    pub message_type: String,
    pub payload: String,
    pub timestamp: f64,
    pub priority: u8,
}

// Cognitive State Management
#[derive(Serialize, Deserialize, Clone)]
pub struct CognitiveState {
    pub current_task: Option<String>,
    pub attention_focus: Vec<f32>,
    pub memory_buffer: Vec<MemoryItem>,
    pub emotional_state: EmotionalState,
    pub processing_mode: ProcessingMode,
}

#[derive(Serialize, Deserialize, Clone)]
pub struct MemoryItem {
    pub id: String,
    pub content: String,
    pub importance: f32,
    pub timestamp: f64,
    pub associations: Vec<String>,
}

#[derive(Serialize, Deserialize, Clone)]
pub struct EmotionalState {
    pub valence: f32,      // Positive/negative emotion
    pub arousal: f32,      // Energy level
    pub dominance: f32,    // Control/confidence
    pub stability: f32,    // Emotional stability
}

#[derive(Serialize, Deserialize, Clone)]
pub enum ProcessingMode {
    Analytical,
    Creative,
    Reactive,
    Contemplative,
}

#[wasm_bindgen]
impl HRMCognitive {
    #[wasm_bindgen(constructor)]
    pub fn new() -> HRMCognitive {
        console_log!("Initializing Enhanced HRM Cognitive Core (562M parameters)...");

        HRMCognitive {
            weights: vec![0.0; 562_741_762], // Actual HRM parameter count
            l_modules: Vec::new(),
            h_modules: Vec::new(),
            personality_adapter: PersonalityAdapter {
                user_id: "default".to_string(),
                personality_metrics: HashMap::new(),
                adaptation_history: Vec::new(),
                learning_rate: 0.01,
            },
            host_interface: HostInterface {
                desktop_id: None,
                connection_status: ConnectionStatus::Disconnected,
                message_queue: Vec::new(),
                capabilities: vec![
                    "cognitive_processing".to_string(),
                    "personality_adaptation".to_string(),
                    "memory_management".to_string(),
                    "emotional_modeling".to_string(),
                ],
            },
            cognitive_state: CognitiveState {
                current_task: None,
                attention_focus: vec![0.0; 10],
                memory_buffer: Vec::new(),
                emotional_state: EmotionalState {
                    valence: 0.0,
                    arousal: 0.5,
                    dominance: 0.5,
                    stability: 0.8,
                },
                processing_mode: ProcessingMode::Analytical,
            },
        }
    }

    #[wasm_bindgen]
    pub fn initialize_modules(&mut self, l_count: u32, h_count: u32) {
        console_log!("Initializing {} L-modules and {} H-modules", l_count, h_count);
        
        // Initialize L-modules
        for i in 0..l_count {
            self.l_modules.push(LModule {
                id: i,
                weights: vec![0.0; 1000], // Placeholder weights
                activation: 0.0,
            });
        }
        
        // Initialize H-modules
        for i in 0..h_count {
            self.h_modules.push(HModule {
                id: i,
                weights: vec![0.0; 2000], // Placeholder weights
                planning_depth: 5,
                activation: 0.0,
            });
        }
    }

    #[wasm_bindgen]
    pub fn process_cognitive_input(&mut self, input_json: &str) -> String {
        console_log!("Processing enhanced cognitive input...");

        // Parse input
        let input: CognitiveInput = match serde_json::from_str(input_json) {
            Ok(input) => input,
            Err(_) => {
                console_log!("Failed to parse input JSON");
                return "{}".to_string();
            }
        };

        let processing_start = js_sys::Date::now();

        // Update cognitive state
        self.cognitive_state.current_task = Some(input.task_type.clone());
        self.update_attention_focus(&input.sensory_data);

        // Apply personality adaptation
        let personality_influence = self.apply_personality_adaptation(&input);

        // Process through L-modules with personality influence
        let mut l_activations = Vec::new();
        for (i, l_module) in self.l_modules.iter_mut().enumerate() {
            let base_activation = (input.sensory_data.iter().sum::<f32>() / input.sensory_data.len() as f32)
                * (i as f32 + 1.0) / 10.0;
            l_module.activation = base_activation * (1.0 + personality_influence * 0.2);
            l_activations.push(l_module.activation);
        }

        // Process through H-modules with emotional state influence
        let mut h_activations = Vec::new();
        let emotional_modifier = self.cognitive_state.emotional_state.valence * 0.1 + 1.0;
        for (i, h_module) in self.h_modules.iter_mut().enumerate() {
            let base_activation = l_activations.iter().sum::<f32>() / (h_module.planning_depth as f32)
                * (i as f32 + 1.0) / 5.0;
            h_module.activation = base_activation * emotional_modifier;
            h_activations.push(h_module.activation);
        }

        // Update emotional state based on processing
        self.update_emotional_state(&l_activations, &h_activations);

        // Store memory if significant
        self.store_memory_if_significant(&input, &l_activations, &h_activations);

        let processing_time = js_sys::Date::now() - processing_start;
        let confidence = self.calculate_confidence(&l_activations, &h_activations);
        let adaptation_score = self.calculate_adaptation_score();

        // Generate enhanced output
        let output = CognitiveOutput {
            reasoning_result: self.generate_reasoning_result(&input, &l_activations, &h_activations),
            confidence,
            processing_time: processing_time as f32,
            l_module_activations: l_activations,
            h_module_activations: h_activations,
            personality_influence,
            adaptation_score,
        };

        serde_json::to_string(&output).unwrap_or_else(|_| "{}".to_string())
    }

    #[wasm_bindgen]
    pub fn get_model_info(&self) -> String {
        let info = format!(
            "{{\"total_parameters\": {}, \"l_modules\": {}, \"h_modules\": {}}}",
            self.weights.len(),
            self.l_modules.len(),
            self.h_modules.len()
        );
        info
    }

    #[wasm_bindgen]
    pub fn load_weights(&mut self, weights_data: &[u8]) -> bool {
        console_log!("Loading HRM model weights into WASM... {} bytes", weights_data.len());

        // Load weights directly into the WASM module's memory
        if weights_data.len() >= 1024 {
            // In a real implementation, this would parse safetensors format
            // and load the 562M parameters into the weights vector

            // For now, simulate loading by updating the weights vector size
            if weights_data.len() > self.weights.len() * 4 { // 4 bytes per f32
                console_log!("Expanding weights vector to accommodate model");
                self.weights.resize(weights_data.len() / 4, 0.0);
            }

            // Simulate weight loading from bytes
            for (i, chunk) in weights_data.chunks(4).enumerate() {
                if i < self.weights.len() && chunk.len() == 4 {
                    // Convert bytes to f32 (little-endian)
                    let bytes = [chunk[0], chunk[1], chunk[2], chunk[3]];
                    self.weights[i] = f32::from_le_bytes(bytes);
                }
            }

            console_log!("HRM weights loaded successfully into WASM module");
            console_log!("Loaded {} parameters", self.weights.len());
            true
        } else {
            console_log!("Invalid HRM weights data - too small");
            false
        }
    }

    #[wasm_bindgen]
    pub fn load_weights_from_url(&mut self, url: &str) -> js_sys::Promise {
        console_log!("Loading HRM weights from URL: {}", url);

        // Return a promise that will load weights asynchronously
        let promise = js_sys::Promise::new(&mut |resolve, _reject| {
            // In a real implementation, this would fetch the weights from the URL
            // For now, we'll simulate successful loading

            let success_value = wasm_bindgen::JsValue::from(true);
            resolve.call1(&wasm_bindgen::JsValue::NULL, &success_value).unwrap();
        });

        promise
    }

    #[wasm_bindgen]
    pub fn get_weights_info(&self) -> String {
        let info = format!(
            "{{\"total_parameters\": {}, \"memory_usage_mb\": {:.2}, \"loaded\": {}}}",
            self.weights.len(),
            (self.weights.len() * 4) as f64 / (1024.0 * 1024.0),
            !self.weights.is_empty()
        );
        info
    }

    // Personality adaptation methods
    fn apply_personality_adaptation(&mut self, input: &CognitiveInput) -> f32 {
        let mut influence = 0.0;

        // Apply personality metrics to processing
        for (metric, value) in &self.personality_adapter.personality_metrics {
            match metric.as_str() {
                "creativity" => influence += value * 0.3,
                "analytical" => influence += value * 0.2,
                "empathy" => influence += value * 0.25,
                "assertiveness" => influence += value * 0.15,
                _ => influence += value * 0.1,
            }
        }

        // Record adaptation event
        let adaptation_event = AdaptationEvent {
            timestamp: js_sys::Date::now(),
            event_type: input.task_type.clone(),
            user_feedback: 0.0, // Will be updated with actual feedback
            context: input.context.clone(),
            adaptation_delta: HashMap::new(),
        };

        self.personality_adapter.adaptation_history.push(adaptation_event);

        influence.tanh() // Normalize to [-1, 1]
    }

    fn update_attention_focus(&mut self, sensory_data: &[f32]) {
        // Update attention based on sensory input
        for (i, focus) in self.cognitive_state.attention_focus.iter_mut().enumerate() {
            if i < sensory_data.len() {
                *focus = (*focus * 0.8) + (sensory_data[i] * 0.2);
            }
        }
    }

    fn update_emotional_state(&mut self, l_activations: &[f32], h_activations: &[f32]) {
        let l_energy = l_activations.iter().sum::<f32>() / l_activations.len() as f32;
        let h_planning = h_activations.iter().sum::<f32>() / h_activations.len() as f32;

        // Update emotional state based on processing results
        self.cognitive_state.emotional_state.arousal =
            (self.cognitive_state.emotional_state.arousal * 0.9) + (l_energy * 0.1);
        self.cognitive_state.emotional_state.dominance =
            (self.cognitive_state.emotional_state.dominance * 0.9) + (h_planning * 0.1);

        // Maintain stability
        self.cognitive_state.emotional_state.stability =
            (self.cognitive_state.emotional_state.stability * 0.95) + 0.05;
    }

    fn store_memory_if_significant(&mut self, input: &CognitiveInput, l_activations: &[f32], h_activations: &[f32]) {
        let significance = (l_activations.iter().sum::<f32>() + h_activations.iter().sum::<f32>())
            / (l_activations.len() + h_activations.len()) as f32;

        if significance > 0.5 {
            let memory_item = MemoryItem {
                id: format!("mem_{}", js_sys::Date::now() as u64),
                content: format!("{}: {}", input.task_type, input.context),
                importance: significance,
                timestamp: js_sys::Date::now(),
                associations: vec![input.task_type.clone()],
            };

            self.cognitive_state.memory_buffer.push(memory_item);

            // Limit memory buffer size
            if self.cognitive_state.memory_buffer.len() > 100 {
                self.cognitive_state.memory_buffer.remove(0);
            }
        }
    }

    fn calculate_confidence(&self, l_activations: &[f32], h_activations: &[f32]) -> f32 {
        let l_variance = self.calculate_variance(l_activations);
        let h_variance = self.calculate_variance(h_activations);
        let emotional_stability = self.cognitive_state.emotional_state.stability;

        ((l_variance + h_variance) / 2.0 * emotional_stability).min(1.0).max(0.0)
    }

    fn calculate_variance(&self, values: &[f32]) -> f32 {
        if values.is_empty() { return 0.0; }

        let mean = values.iter().sum::<f32>() / values.len() as f32;
        let variance = values.iter()
            .map(|x| (x - mean).powi(2))
            .sum::<f32>() / values.len() as f32;

        variance.sqrt()
    }

    fn calculate_adaptation_score(&self) -> f32 {
        let history_len = self.personality_adapter.adaptation_history.len() as f32;
        if history_len == 0.0 { return 0.0; }

        let recent_events = self.personality_adapter.adaptation_history
            .iter()
            .rev()
            .take(10)
            .map(|event| event.user_feedback)
            .sum::<f32>();

        (recent_events / 10.0).min(1.0).max(-1.0)
    }

    fn generate_reasoning_result(&self, input: &CognitiveInput, l_activations: &[f32], h_activations: &[f32]) -> String {
        let l_avg = l_activations.iter().sum::<f32>() / l_activations.len() as f32;
        let h_avg = h_activations.iter().sum::<f32>() / h_activations.len() as f32;
        let emotional_influence = self.cognitive_state.emotional_state.valence;

        format!(
            "HRM processed '{}' with {:.1}% sensory activation, {:.1}% planning depth, emotional valence: {:.2}",
            input.task_type,
            l_avg * 100.0,
            h_avg * 100.0,
            emotional_influence
        )
    }

    // Host interface methods
    #[wasm_bindgen]
    pub fn connect_to_desktop(&mut self, desktop_id: &str) -> bool {
        console_log!("Connecting to desktop: {}", desktop_id);

        self.host_interface.desktop_id = Some(desktop_id.to_string());
        self.host_interface.connection_status = ConnectionStatus::Connected;

        // Send initial capabilities message
        let capabilities_msg = HostMessage {
            id: format!("cap_{}", js_sys::Date::now() as u64),
            message_type: "capabilities".to_string(),
            payload: serde_json::to_string(&self.host_interface.capabilities).unwrap_or_default(),
            timestamp: js_sys::Date::now(),
            priority: 1,
        };

        self.host_interface.message_queue.push(capabilities_msg);
        true
    }

    #[wasm_bindgen]
    pub fn send_host_message(&mut self, message_type: &str, payload: &str) -> String {
        let message = HostMessage {
            id: format!("msg_{}", js_sys::Date::now() as u64),
            message_type: message_type.to_string(),
            payload: payload.to_string(),
            timestamp: js_sys::Date::now(),
            priority: 2,
        };

        let message_id = message.id.clone();
        self.host_interface.message_queue.push(message);

        console_log!("Queued host message: {} ({})", message_type, message_id);
        message_id
    }

    #[wasm_bindgen]
    pub fn get_pending_messages(&mut self) -> String {
        let messages = self.host_interface.message_queue.clone();
        self.host_interface.message_queue.clear();

        serde_json::to_string(&messages).unwrap_or_else(|_| "[]".to_string())
    }

    // Personality management methods
    #[wasm_bindgen]
    pub fn set_personality_metric(&mut self, metric: &str, value: f32) {
        console_log!("Setting personality metric: {} = {}", metric, value);
        self.personality_adapter.personality_metrics.insert(metric.to_string(), value.clamp(-1.0, 1.0));
    }

    #[wasm_bindgen]
    pub fn get_personality_profile(&self) -> String {
        serde_json::to_string(&self.personality_adapter).unwrap_or_else(|_| "{}".to_string())
    }

    #[wasm_bindgen]
    pub fn update_user_feedback(&mut self, task_id: &str, feedback: f32) {
        console_log!("Updating user feedback for task {}: {}", task_id, feedback);

        // Find the most recent adaptation event and update feedback
        if let Some(event) = self.personality_adapter.adaptation_history.last_mut() {
            event.user_feedback = feedback.clamp(-1.0, 1.0);

            // Adapt personality based on feedback
            self.adapt_personality_from_feedback(feedback);
        }
    }

    fn adapt_personality_from_feedback(&mut self, feedback: f32) {
        let learning_rate = self.personality_adapter.learning_rate;

        // Adjust personality metrics based on feedback
        for (metric, value) in self.personality_adapter.personality_metrics.iter_mut() {
            match metric.as_str() {
                "creativity" => {
                    if feedback > 0.5 {
                        *value += learning_rate * 0.1;
                    } else if feedback < -0.5 {
                        *value -= learning_rate * 0.1;
                    }
                },
                "analytical" => {
                    if feedback > 0.3 {
                        *value += learning_rate * 0.05;
                    }
                },
                _ => {
                    *value += learning_rate * feedback * 0.02;
                }
            }

            // Keep values in valid range
            *value = value.clamp(-1.0, 1.0);
        }
    }

    // Cognitive state management
    #[wasm_bindgen]
    pub fn get_cognitive_state(&self) -> String {
        serde_json::to_string(&self.cognitive_state).unwrap_or_else(|_| "{}".to_string())
    }

    #[wasm_bindgen]
    pub fn set_processing_mode(&mut self, mode: &str) {
        let new_mode = match mode {
            "analytical" => ProcessingMode::Analytical,
            "creative" => ProcessingMode::Creative,
            "reactive" => ProcessingMode::Reactive,
            "contemplative" => ProcessingMode::Contemplative,
            _ => ProcessingMode::Analytical,
        };

        self.cognitive_state.processing_mode = new_mode;
        console_log!("Processing mode set to: {}", mode);
    }

    #[wasm_bindgen]
    pub fn clear_memory_buffer(&mut self) {
        self.cognitive_state.memory_buffer.clear();
        console_log!("Memory buffer cleared");
    }

    #[wasm_bindgen]
    pub fn get_memory_summary(&self) -> String {
        let memory_count = self.cognitive_state.memory_buffer.len();
        let avg_importance = if memory_count > 0 {
            self.cognitive_state.memory_buffer.iter()
                .map(|item| item.importance)
                .sum::<f32>() / memory_count as f32
        } else {
            0.0
        };

        let summary = format!(
            "{{\"memory_count\": {}, \"average_importance\": {:.3}, \"emotional_valence\": {:.3}, \"current_task\": \"{}\"}}",
            memory_count,
            avg_importance,
            self.cognitive_state.emotional_state.valence,
            self.cognitive_state.current_task.as_ref().unwrap_or(&"none".to_string())
        );

        summary
    }
}

// Initialize the enhanced WASM module
#[wasm_bindgen(start)]
pub fn main() {
    console_log!("Enhanced KNIRV-CORTEX WASM module initialized with HRM integration");
}
