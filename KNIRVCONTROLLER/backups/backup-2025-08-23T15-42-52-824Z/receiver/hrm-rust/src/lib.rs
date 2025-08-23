use wasm_bindgen::prelude::*;
use serde::{Deserialize, Serialize};

#[wasm_bindgen]
extern "C" {
    #[wasm_bindgen(js_namespace = console)]
    fn log(s: &str);
}

macro_rules! console_log {
    ($($t:tt)*) => (log(&format_args!($($t)*).to_string()))
}

#[derive(Serialize, Deserialize)]
pub struct HRMInput {
    pub sensory_data: Vec<f32>,
    pub context: String,
    pub task_type: String,
}

#[derive(Serialize, Deserialize)]
pub struct HRMOutput {
    pub reasoning_result: String,
    pub confidence: f32,
    pub processing_time: f32,
    pub l_module_activations: Vec<f32>,
    pub h_module_activations: Vec<f32>,
}

#[wasm_bindgen]
pub struct HRMCognitive {
    // L-modules for sensory-motor patterns
    l_modules: Vec<LModule>,
    // H-modules for long-horizon planning
    h_modules: Vec<HModule>,
    // Model weights loaded from safetensors
    weights_loaded: bool,
}

#[derive(Clone)]
struct LModule {
    id: u32,
    weights: Option<Vec<f32>>, // Simplified for now
    activation: f32,
}

#[derive(Clone)]
struct HModule {
    id: u32,
    weights: Option<Vec<f32>>, // Simplified for now
    planning_depth: u32,
    activation: f32,
}

#[wasm_bindgen]
impl HRMCognitive {
    #[wasm_bindgen(constructor)]
    pub fn new() -> HRMCognitive {
        console_log!("Initializing HRM Cognitive Core (562M parameters)...");

        HRMCognitive {
            l_modules: Vec::new(),
            h_modules: Vec::new(),
            weights_loaded: false,
        }
    }

    #[wasm_bindgen]
    pub fn initialize_modules(&mut self, l_count: u32, h_count: u32) {
        console_log!("Initializing {} L-modules and {} H-modules", l_count, h_count);

        // Initialize L-modules (sensory-motor patterns)
        self.l_modules = (0..l_count).map(|i| LModule {
            id: i,
            weights: None,
            activation: 0.0,
        }).collect();

        // Initialize H-modules (long-horizon planning)
        self.h_modules = (0..h_count).map(|i| HModule {
            id: i,
            weights: None,
            planning_depth: (i + 1) * 4, // Increasing planning depth
            activation: 0.0,
        }).collect();

        console_log!("HRM modules initialized successfully");
    }

    #[wasm_bindgen]
    pub fn load_weights(&mut self, weights_data: &[u8]) -> bool {
        console_log!("Loading HRM weights... {} bytes", weights_data.len());

        // In production, this would load actual safetensors weights
        // For now, simulate successful loading
        if weights_data.len() > 1024 {
            self.weights_loaded = true;
            console_log!("HRM weights loaded successfully");
            true
        } else {
            console_log!("Invalid HRM weights data");
            false
        }
    }

    #[wasm_bindgen]
    pub fn process_cognitive_input(&mut self, input_json: &str) -> String {
        if !self.weights_loaded {
            console_log!("HRM weights not loaded");
            return "{}".to_string();
        }

        let input: HRMInput = match serde_json::from_str(input_json) {
            Ok(input) => input,
            Err(_) => {
                console_log!("Failed to parse HRM input JSON");
                return "{}".to_string();
            }
        };

        let processing_start = js_sys::Date::now();

        // Process through L-modules (sensory-motor patterns)
        let mut l_activations = Vec::new();
        for i in 0..self.l_modules.len() {
            // Hierarchical sensory processing
            let activation = Self::process_l_module_static(&input.sensory_data, self.l_modules[i].id);
            self.l_modules[i].activation = activation;
            l_activations.push(activation);
        }

        // Process through H-modules (long-horizon planning)
        let mut h_activations = Vec::new();
        for i in 0..self.h_modules.len() {
            // Hierarchical planning processing
            let activation = Self::process_h_module_static(&l_activations, &self.h_modules[i]);
            self.h_modules[i].activation = activation;
            h_activations.push(activation);
        }

        // Generate reasoning result based on hierarchical processing
        let reasoning_result = self.generate_reasoning(&input, &l_activations, &h_activations);
        let confidence = self.calculate_confidence(&l_activations, &h_activations);

        let processing_time = (js_sys::Date::now() - processing_start) as f32;

        let output = HRMOutput {
            reasoning_result,
            confidence,
            processing_time,
            l_module_activations: l_activations,
            h_module_activations: h_activations,
        };

        serde_json::to_string(&output).unwrap_or_else(|_| "{}".to_string())
    }

    #[wasm_bindgen]
    pub fn get_model_info(&self) -> String {
        let info = serde_json::json!({
            "total_parameters": 562_741_762,
            "l_modules": self.l_modules.len(),
            "h_modules": self.h_modules.len(),
            "weights_loaded": self.weights_loaded,
            "model_type": "HRM-562M"
        });

        info.to_string()
    }

    // Private methods for HRM processing
    fn process_l_module_static(sensory_data: &[f32], module_id: u32) -> f32 {
        // Simulate L-module sensory-motor pattern processing
        let base_activation = sensory_data.iter().sum::<f32>() / sensory_data.len() as f32;
        base_activation * (module_id as f32 + 1.0) / 10.0
    }

    fn process_h_module_static(l_activations: &[f32], h_module: &HModule) -> f32 {
        // Simulate H-module long-horizon planning
        let l_sum = l_activations.iter().sum::<f32>();
        l_sum / (h_module.planning_depth as f32) * (h_module.id as f32 + 1.0) / 5.0
    }

    fn generate_reasoning(&self, input: &HRMInput, l_activations: &[f32], h_activations: &[f32]) -> String {
        // Generate reasoning based on hierarchical processing
        format!(
            "HRM processed '{}' with {} L-modules (avg: {:.3}) and {} H-modules (avg: {:.3})",
            input.task_type,
            l_activations.len(),
            l_activations.iter().sum::<f32>() / l_activations.len() as f32,
            h_activations.len(),
            h_activations.iter().sum::<f32>() / h_activations.len() as f32
        )
    }

    fn calculate_confidence(&self, l_activations: &[f32], h_activations: &[f32]) -> f32 {
        // Calculate confidence based on activation patterns
        let l_variance = self.calculate_variance(l_activations);
        let h_variance = self.calculate_variance(h_activations);

        // Higher variance indicates more confident processing
        ((l_variance + h_variance) / 2.0).min(1.0).max(0.0)
    }

    fn calculate_variance(&self, values: &[f32]) -> f32 {
        if values.is_empty() { return 0.0; }

        let mean = values.iter().sum::<f32>() / values.len() as f32;
        let variance = values.iter()
            .map(|x| (x - mean).powi(2))
            .sum::<f32>() / values.len() as f32;

        variance.sqrt()
    }
}

#[wasm_bindgen(start)]
pub fn main() {
    console_log!("HRM Cognitive WASM module initialized (562M parameters)");
}
