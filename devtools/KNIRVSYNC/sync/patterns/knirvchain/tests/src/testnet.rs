use actix_web::{web, HttpResponse, Result};
#[cfg(feature = "testnet")]
use serde::{Deserialize, Serialize};
use std::collections::HashMap;

#[cfg(feature = "testnet")]
use log::info;

#[cfg(feature = "testnet")]
#[derive(Debug, Serialize, Deserialize, Clone)]
pub struct TestnetConfig {
    pub chain_id: String,
    pub single_validator: bool,
    pub mock_llm: bool,
    pub simplified_storage: bool,
    pub in_memory: bool,
    pub pre_populate: bool,
}

#[cfg(feature = "testnet")]
impl Default for TestnetConfig {
    fn default() -> Self {
        Self {
            chain_id: "knirvchain-testnet-1".to_string(),
            single_validator: true,
            mock_llm: true,
            simplified_storage: true,
            in_memory: true,
            pre_populate: true,
        }
    }
}

#[cfg(feature = "testnet")]
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MockLLMResponse {
    pub success: bool,
    pub model_id: String,
    pub accuracy: f64,
    pub latency_ms: u64,
    pub throughput_tokens_per_sec: u64,
    pub validation_result: String,
}

#[cfg(feature = "testnet")]
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MockSkillResponse {
    pub success: bool,
    pub skill_id: String,
    pub validation_passed: bool,
    pub execution_time_ms: u64,
    pub test_results: MockTestResults,
}

#[cfg(feature = "testnet")]
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MockTestResults {
    pub passed: u32,
    pub failed: u32,
    pub total: u32,
}

#[cfg(feature = "testnet")]
pub struct MockLLMEngine {
    models: HashMap<String, MockLLMResponse>,
}

#[cfg(feature = "testnet")]
impl MockLLMEngine {
    pub fn new() -> Self {
        let mut models = HashMap::new();

        // Pre-populate with some mock models
        models.insert(
            "gpt-4-mock".to_string(),
            MockLLMResponse {
                success: true,
                model_id: "gpt-4-mock".to_string(),
                accuracy: 0.95,
                latency_ms: 50,
                throughput_tokens_per_sec: 100,
                validation_result: "Model validated successfully".to_string(),
            },
        );

        models.insert(
            "claude-3-mock".to_string(),
            MockLLMResponse {
                success: true,
                model_id: "claude-3-mock".to_string(),
                accuracy: 0.93,
                latency_ms: 45,
                throughput_tokens_per_sec: 120,
                validation_result: "Model validated successfully".to_string(),
            },
        );

        Self { models }
    }

    pub fn validate_model(&mut self, model_id: &str) -> MockLLMResponse {
        if let Some(existing) = self.models.get(model_id) {
            existing.clone()
        } else {
            // Create a new mock response for unknown models
            let response = MockLLMResponse {
                success: true,
                model_id: model_id.to_string(),
                accuracy: 0.85 + (rand::random::<f64>() * 0.1), // Random accuracy between 0.85-0.95
                latency_ms: 30 + (rand::random::<u64>() % 50),  // Random latency 30-80ms
                throughput_tokens_per_sec: 80 + (rand::random::<u64>() % 60), // Random throughput 80-140
                validation_result: "Mock validation completed".to_string(),
            };
            self.models.insert(model_id.to_string(), response.clone());
            response
        }
    }
}

#[cfg(feature = "testnet")]
pub struct MockSkillEngine {
    skills: HashMap<String, MockSkillResponse>,
}

#[cfg(feature = "testnet")]
impl MockSkillEngine {
    pub fn new() -> Self {
        Self {
            skills: HashMap::new(),
        }
    }

    pub fn validate_skill(&mut self, skill_id: &str, _skill_code: &str) -> MockSkillResponse {
        let response = MockSkillResponse {
            success: true,
            skill_id: skill_id.to_string(),
            validation_passed: true,
            execution_time_ms: 100 + (rand::random::<u64>() % 200), // Random execution time 100-300ms
            test_results: MockTestResults {
                passed: 8 + (rand::random::<u32>() % 3), // 8-10 passed tests
                failed: rand::random::<u32>() % 2,       // 0-1 failed tests
                total: 10,
            },
        };

        self.skills.insert(skill_id.to_string(), response.clone());
        response
    }
}

#[cfg(feature = "testnet")]
pub async fn mock_llm_validate(
    model_request: web::Json<HashMap<String, String>>,
) -> Result<HttpResponse> {
    let request = model_request.into_inner();
    let model_id = request
        .get("model_id")
        .unwrap_or(&"unknown".to_string())
        .clone();

    info!("Mock LLM validation for model: {}", model_id);

    let mut engine = MockLLMEngine::new();
    let response = engine.validate_model(&model_id);

    Ok(HttpResponse::Ok().json(response))
}

#[cfg(feature = "testnet")]
pub async fn mock_skill_validate(
    skill_request: web::Json<HashMap<String, String>>,
) -> Result<HttpResponse> {
    let request = skill_request.into_inner();
    let skill_id = request
        .get("skill_id")
        .unwrap_or(&"unknown".to_string())
        .clone();
    let skill_code = request.get("skill_code").unwrap_or(&"".to_string()).clone();

    info!("Mock skill validation for skill: {}", skill_id);

    let mut engine = MockSkillEngine::new();
    let response = engine.validate_skill(&skill_id, &skill_code);

    Ok(HttpResponse::Ok().json(response))
}

#[cfg(feature = "testnet")]
pub async fn testnet_status() -> Result<HttpResponse> {
    let status = serde_json::json!({
        "testnet": true,
        "chain_id": "knirvchain-testnet-1",
        "features": {
            "mock_llm": true,
            "simplified_storage": true,
            "in_memory_db": true,
            "single_validator": true
        },
        "endpoints": {
            "mock_llm_validate": "/testnet/llm/validate",
            "mock_skill_validate": "/testnet/skill/validate",
            "status": "/testnet/status"
        }
    });

    Ok(HttpResponse::Ok().json(status))
}

#[cfg(feature = "testnet")]
pub async fn health_check() -> Result<HttpResponse> {
    Ok(HttpResponse::Ok().json(serde_json::json!({
        "status": "healthy",
        "mode": "testnet",
        "timestamp": chrono::Utc::now().to_rfc3339()
    })))
}

// Non-testnet fallback functions
#[cfg(not(feature = "testnet"))]
pub async fn mock_llm_validate(_: web::Json<HashMap<String, String>>) -> Result<HttpResponse> {
    Ok(HttpResponse::NotImplemented().json(serde_json::json!({
        "error": "Testnet features not enabled"
    })))
}

#[cfg(not(feature = "testnet"))]
pub async fn mock_skill_validate(_: web::Json<HashMap<String, String>>) -> Result<HttpResponse> {
    Ok(HttpResponse::NotImplemented().json(serde_json::json!({
        "error": "Testnet features not enabled"
    })))
}

#[cfg(not(feature = "testnet"))]
pub async fn testnet_status() -> Result<HttpResponse> {
    Ok(HttpResponse::NotImplemented().json(serde_json::json!({
        "error": "Testnet features not enabled"
    })))
}

#[cfg(not(feature = "testnet"))]
pub async fn health_check() -> Result<HttpResponse> {
    Ok(HttpResponse::Ok().json(serde_json::json!({
        "status": "healthy",
        "mode": "testnet"
    })))
}
