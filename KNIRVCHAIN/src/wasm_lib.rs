// KNIRVCHAIN WASM Library - Revolutionary Embedded Skill Execution
use wasm_bindgen::prelude::*;
use serde::{Deserialize, Serialize};
use std::collections::HashMap;

// Set up panic hook for better error messages in WASM
#[wasm_bindgen(start)]
pub fn main() {
    console_error_panic_hook::set_once();
}

// Use wee_alloc as the global allocator for smaller WASM size
#[global_allocator]
static ALLOC: wee_alloc::WeeAlloc = wee_alloc::WeeAlloc::INIT;

// Revolutionary LoRA Adapter Skill Structure
#[derive(Serialize, Deserialize, Clone, Debug)]
#[wasm_bindgen]
pub struct LoRAAdapterSkill {
    skill_id: String,
    skill_name: String,
    description: String,
    base_model_compatibility: String,
    pub version: u32,
    pub rank: u32,
    pub alpha: f64,
}

#[wasm_bindgen]
impl LoRAAdapterSkill {
    #[wasm_bindgen(constructor)]
    pub fn new(
        skill_id: String,
        skill_name: String,
        description: String,
        base_model_compatibility: String,
        version: u32,
        rank: u32,
        alpha: f64,
    ) -> LoRAAdapterSkill {
        LoRAAdapterSkill {
            skill_id,
            skill_name,
            description,
            base_model_compatibility,
            version,
            rank,
            alpha,
        }
    }

    #[wasm_bindgen(getter)]
    pub fn skill_id(&self) -> String {
        self.skill_id.clone()
    }

    #[wasm_bindgen(getter)]
    pub fn skill_name(&self) -> String {
        self.skill_name.clone()
    }

    #[wasm_bindgen(getter)]
    pub fn description(&self) -> String {
        self.description.clone()
    }
}

// Revolutionary Error Context for KNIRVGRAPH Discovery
#[derive(Serialize, Deserialize, Clone, Debug)]
#[wasm_bindgen]
pub struct ErrorContext {
    agent_id: String,
    agent_version: String,
    base_model_id: String,
    os: String,
    architecture: String,
    runtime_environment: String,
    error_type: String,
    error_message: String,
    stack_trace: String,
    source_code_snippet: String,
    task_description: String,
    input_data_hash: String,
    skill_invoked_id: String,
    agent_state_hash: String,
    pub timestamp: i64,
}

#[wasm_bindgen]
impl ErrorContext {
    #[wasm_bindgen(constructor)]
    pub fn new(
        agent_id: String,
        error_type: String,
        error_message: String,
        task_description: String,
    ) -> ErrorContext {
        ErrorContext {
            agent_id,
            agent_version: "1.0.0".to_string(),
            base_model_id: "hrm-v1".to_string(),
            os: "linux".to_string(),
            architecture: "x86_64".to_string(),
            runtime_environment: "knirv-nexus-dve".to_string(),
            error_type,
            error_message,
            stack_trace: String::new(),
            source_code_snippet: String::new(),
            task_description,
            input_data_hash: String::new(),
            skill_invoked_id: String::new(),
            agent_state_hash: String::new(),
            timestamp: js_sys::Date::now() as i64,
        }
    }
}

// Revolutionary Skill Invocation Request
#[derive(Serialize, Deserialize, Clone, Debug)]
#[wasm_bindgen]
pub struct SkillInvocationRequest {
    invocation_id: String,
    agent_id: String,
    nrn_token: String,
    skill_uri: String,
    priority: String,
    pub timestamp: i64,
}

#[wasm_bindgen]
impl SkillInvocationRequest {
    #[wasm_bindgen(constructor)]
    pub fn new(
        invocation_id: String,
        agent_id: String,
        nrn_token: String,
        skill_uri: String,
    ) -> SkillInvocationRequest {
        SkillInvocationRequest {
            invocation_id,
            agent_id,
            nrn_token,
            skill_uri,
            priority: "normal".to_string(),
            timestamp: js_sys::Date::now() as i64,
        }
    }
}

// Revolutionary Skill Invocation Response
#[derive(Serialize, Deserialize, Clone, Debug)]
#[wasm_bindgen]
pub struct SkillInvocationResponse {
    invocation_id: String,
    status: String,
    error_message: String,
    pub execution_time: i64,
    pub memory_used: i64,
    pub consensus_reached: bool,
    skill_data: String, // JSON serialized skill data
}

#[wasm_bindgen]
impl SkillInvocationResponse {
    #[wasm_bindgen(constructor)]
    pub fn new(
        invocation_id: String,
        status: String,
        execution_time: i64,
        skill_data: String,
    ) -> SkillInvocationResponse {
        SkillInvocationResponse {
            invocation_id,
            status,
            error_message: String::new(),
            execution_time,
            memory_used: 1024,
            consensus_reached: true,
            skill_data,
        }
    }

    #[wasm_bindgen(getter)]
    pub fn invocation_id(&self) -> String {
        self.invocation_id.clone()
    }

    #[wasm_bindgen(getter)]
    pub fn status(&self) -> String {
        self.status.clone()
    }

    #[wasm_bindgen(getter)]
    pub fn skill_data(&self) -> String {
        self.skill_data.clone()
    }
}

// Revolutionary Embedded KNIRVCHAIN WASM Engine
#[wasm_bindgen]
pub struct EmbeddedKNIRVChain {
    skills: HashMap<String, LoRAAdapterSkill>,
    skill_uri_mapping: HashMap<String, String>,
    initialized: bool,
}

#[wasm_bindgen]
impl EmbeddedKNIRVChain {
    #[wasm_bindgen(constructor)]
    pub fn new() -> EmbeddedKNIRVChain {
        EmbeddedKNIRVChain {
            skills: HashMap::new(),
            skill_uri_mapping: HashMap::new(),
            initialized: false,
        }
    }

    /// Initialize the Revolutionary Embedded KNIRVCHAIN
    #[wasm_bindgen]
    pub fn initialize(&mut self) -> Result<(), JsValue> {
        // Log initialization
        web_sys::console::log_1(&"🚀 Initializing Revolutionary Embedded KNIRVCHAIN WASM...".into());

        // Create default skills for testing
        self.create_default_skills()?;

        self.initialized = true;
        web_sys::console::log_1(&"✅ Revolutionary Embedded KNIRVCHAIN WASM initialized successfully".into());
        Ok(())
    }

    /// Register a new LoRA Adapter Skill
    #[wasm_bindgen]
    pub fn register_skill(&mut self, skill: LoRAAdapterSkill) -> Result<(), JsValue> {
        if !self.initialized {
            return Err(JsValue::from_str("KNIRVCHAIN not initialized"));
        }

        let skill_id = skill.skill_id.clone();
        let skill_uri = format!("knirv://skill/{}-v{}", skill.skill_name, skill.version);
        
        self.skill_uri_mapping.insert(skill_uri, skill_id.clone());
        self.skills.insert(skill_id, skill);

        Ok(())
    }

    /// Revolutionary Skill Invocation via WASM
    #[wasm_bindgen]
    pub fn invoke_skill(&self, request: SkillInvocationRequest) -> Result<SkillInvocationResponse, JsValue> {
        if !self.initialized {
            return Err(JsValue::from_str("KNIRVCHAIN not initialized"));
        }

        let start_time = js_sys::Date::now() as i64;

        // Log invocation
        web_sys::console::log_1(&format!(
            "🎯 Revolutionary skill invocation: {} (agent: {})",
            request.invocation_id, request.agent_id
        ).into());

        // Validate NRN token
        if request.nrn_token.len() < 32 {
            return Ok(SkillInvocationResponse {
                invocation_id: request.invocation_id,
                status: "FAILURE".to_string(),
                error_message: "Invalid NRN token format".to_string(),
                execution_time: js_sys::Date::now() as i64 - start_time,
                memory_used: 0,
                consensus_reached: false,
                skill_data: String::new(),
            });
        }

        // Resolve skill URI
        let skill_id = match self.skill_uri_mapping.get(&request.skill_uri) {
            Some(id) => id,
            None => {
                return Ok(SkillInvocationResponse {
                    invocation_id: request.invocation_id,
                    status: "NOT_FOUND".to_string(),
                    error_message: format!("Skill URI {} not found", request.skill_uri),
                    execution_time: js_sys::Date::now() as i64 - start_time,
                    memory_used: 0,
                    consensus_reached: false,
                    skill_data: String::new(),
                });
            }
        };

        // Get skill
        let skill = match self.skills.get(skill_id) {
            Some(skill) => skill,
            None => {
                return Ok(SkillInvocationResponse {
                    invocation_id: request.invocation_id,
                    status: "NOT_FOUND".to_string(),
                    error_message: format!("Skill {} not found in registry", skill_id),
                    execution_time: js_sys::Date::now() as i64 - start_time,
                    memory_used: 0,
                    consensus_reached: false,
                    skill_data: String::new(),
                });
            }
        };

        // Serialize skill data
        let skill_data = match serde_json::to_string(skill) {
            Ok(data) => data,
            Err(e) => {
                return Ok(SkillInvocationResponse {
                    invocation_id: request.invocation_id,
                    status: "FAILURE".to_string(),
                    error_message: format!("Failed to serialize skill: {}", e),
                    execution_time: js_sys::Date::now() as i64 - start_time,
                    memory_used: 0,
                    consensus_reached: false,
                    skill_data: String::new(),
                });
            }
        };

        // Simulate skill execution
        let execution_time = js_sys::Date::now() as i64 - start_time;

        // Log success
        web_sys::console::log_1(&format!(
            "✅ Revolutionary skill invocation completed: {} ({}ms)",
            request.invocation_id, execution_time
        ).into());

        Ok(SkillInvocationResponse {
            invocation_id: request.invocation_id,
            status: "SUCCESS".to_string(),
            error_message: String::new(),
            execution_time,
            memory_used: 1024,
            consensus_reached: true,
            skill_data,
        })
    }

    /// Get skill count
    #[wasm_bindgen]
    pub fn get_skill_count(&self) -> usize {
        self.skills.len()
    }

    /// Check if initialized
    #[wasm_bindgen]
    pub fn is_initialized(&self) -> bool {
        self.initialized
    }

    /// Create default skills for testing
    fn create_default_skills(&mut self) -> Result<(), JsValue> {
        let default_skills = vec![
            LoRAAdapterSkill::new(
                "skill-js-type-checker-v1".to_string(),
                "javascript-type-checker".to_string(),
                "Revolutionary JavaScript/TypeScript type checking skill".to_string(),
                "hrm".to_string(),
                1,
                8,
                16.0,
            ),
            LoRAAdapterSkill::new(
                "skill-syntax-fixer-v2".to_string(),
                "syntax-error-fixer".to_string(),
                "Revolutionary syntax error detection and fixing skill".to_string(),
                "hrm".to_string(),
                2,
                12,
                24.0,
            ),
        ];

        for skill in default_skills {
            self.register_skill(skill)?;
        }

        Ok(())
    }
}

// Export functions for direct WASM usage
#[wasm_bindgen]
pub fn get_version() -> String {
    "1.0.0".to_string()
}

#[wasm_bindgen]
pub fn get_build_info() -> String {
    "KNIRVCHAIN WASM - Revolutionary Embedded Skill Execution Engine".to_string()
}
