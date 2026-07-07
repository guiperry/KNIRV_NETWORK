
use wasm_bindgen::prelude::*;
use web_sys::console;
use serde::{Deserialize, Serialize};

// Import console.log for debugging
#[wasm_bindgen]
extern "C" {
    #[wasm_bindgen(js_namespace = console)]
    fn log(s: &str);
}

macro_rules! console_log {
    ($($t:tt)*) => (log(&format_args!($($t)*).to_string()))
}

// Initialize panic hook
#[wasm_bindgen(start)]
pub fn main() {
    console_error_panic_hook::set_once();
    console_log!("Agent-Core WASM initialized: {}", "test-agent-init");
}

// Agent configuration
#[derive(Serialize, Deserialize)]
pub struct AgentConfig {
    pub agent_id: String,
    pub max_context_size: usize,
    pub learning_rate: f32,
    pub adaptation_threshold: f32,
    pub skill_timeout: u32,
}

// Agent-Core implementation
#[wasm_bindgen]
pub struct AgentCore {
    config: AgentConfig,
    initialized: bool,
    memory: Vec<String>,
}

#[wasm_bindgen]
impl AgentCore {
    #[wasm_bindgen(constructor)]
    pub fn new() -> AgentCore {
        let config = AgentConfig {
            agent_id: "test-agent-init".to_string(),
            max_context_size: 10000,
            learning_rate: 0.01,
            adaptation_threshold: 0.7,
            skill_timeout: 30000,
        };

        AgentCore {
            config,
            initialized: false,
            memory: Vec::new(),
        }
    }

    #[wasm_bindgen]
    pub fn agent_core_execute(&mut self, input: &str, context: &str) -> String {
        if !self.initialized {
            return r#"{"error": "Agent not initialized"}"#.to_string();
        }

        console_log!("Executing agent-core with input: {}", input);

        // Parse input and context
        let result = self.process_cognitive_input(input, context);

        // Store in memory for learning
        self.memory.push(input.to_string());
        if self.memory.len() > self.config.max_context_size {
            self.memory.remove(0);
        }

        result
    }

    #[wasm_bindgen]
    pub fn agent_core_execute_tool(&self, tool_name: &str, parameters: &str, context: &str) -> String {
        if !self.initialized {
            return r#"{"error": "Agent not initialized"}"#.to_string();
        }

        console_log!("Executing tool: {} with parameters: {}", tool_name, parameters);

        // Tool execution logic
        format!(
            r#"{{"success": true, "result": "Tool {} executed successfully", "parameters": {}, "agentId": "{}"}}"#,
            tool_name, parameters, self.config.agent_id
        )
    }

    #[wasm_bindgen]
    pub fn agent_core_load_lora(&mut self, adapter: &str) -> bool {
        console_log!("Loading LoRA adapter: {}", adapter);
        // LoRA adapter loading logic would go here
        true
    }

    #[wasm_bindgen]
    pub fn agent_core_apply_skill(&mut self, proto_bytes: &[u8]) -> bool {
        console_log!("Applying skill from protobuf ({} bytes)", proto_bytes.len());
        // Skill application logic would go here
        true
    }

    #[wasm_bindgen]
    pub fn agent_core_get_status(&self) -> String {
        format!(
            r#"{{"agentId": "{}", "agentName": "Test Agent Init", "version": "1.0.0", "initialized": {}, "cognitiveEngine": "rust-wasm", "availableTools": [], "memorySize": {}}}"#,
            self.config.agent_id, self.initialized, self.memory.len()
        )
    }
}

impl AgentCore {
    fn process_cognitive_input(&self, input: &str, context: &str) -> String {
        // Cognitive processing logic
        // This would contain the actual AI processing logic

        let confidence = if input.len() > 10 { 0.8 } else { 0.6 };

        format!(
            r#"{{"success": true, "result": {{"response": "Processed: {}", "confidence": {}, "source": "rust-agent-core"}}, "processingTime": 50, "metadata": {{"agentId": "{}", "contextSize": {}}}}}"#,
            input, confidence, self.config.agent_id, context.len()
        )
    }
}
