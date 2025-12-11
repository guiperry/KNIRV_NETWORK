/**
 * External Inference Integration for KNIRV-CORTEX Rust WASM
 * Provides external API inference routing during beta phase
 */

use wasm_bindgen::prelude::*;
use wasm_bindgen_futures::JsFuture;
use web_sys::{Request, RequestInit, RequestMode, Response};
use serde::{Deserialize, Serialize};
use std::collections::HashMap;

#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum InferenceProvider {
    #[serde(rename = "gemini")]
    Gemini,
    #[serde(rename = "claude")]
    Claude,
    #[serde(rename = "openai")]
    OpenAI,
    #[serde(rename = "deepseek")]
    Deepseek,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ExternalInferenceRequest {
    pub prompt: String,
    pub max_tokens: Option<u32>,
    pub temperature: Option<f32>,
    pub system_prompt: Option<String>,
    pub conversation_history: Option<Vec<ConversationMessage>>,
    pub task_type: Option<String>,
    pub context: Option<HashMap<String, serde_json::Value>>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ConversationMessage {
    pub role: String,
    pub content: String,
    pub timestamp: Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ExternalInferenceResponse {
    pub success: bool,
    pub content: String,
    pub provider: InferenceProvider,
    pub processing_time: u32,
    pub confidence: f32,
    pub usage: Option<TokenUsage>,
    pub error: Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TokenUsage {
    pub prompt_tokens: u32,
    pub completion_tokens: u32,
    pub total_tokens: u32,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ProviderConfig {
    pub provider: InferenceProvider,
    pub api_key: String,
    pub endpoint: Option<String>,
    pub model: Option<String>,
    pub enabled: bool,
}

#[wasm_bindgen]
pub struct ExternalInferenceEngine {
    active_provider: Option<InferenceProvider>,
    provider_configs: HashMap<String, ProviderConfig>,
    request_count: u32,
    total_processing_time: u32,
}

#[wasm_bindgen]
impl ExternalInferenceEngine {
    #[wasm_bindgen(constructor)]
    pub fn new() -> ExternalInferenceEngine {
        ExternalInferenceEngine {
            active_provider: None,
            provider_configs: HashMap::new(),
            request_count: 0,
            total_processing_time: 0,
        }
    }

    /// Configure external inference provider
    #[wasm_bindgen]
    pub fn configure_provider(&mut self, provider_json: &str) -> Result<(), JsValue> {
        let config: ProviderConfig = serde_json::from_str(provider_json)
            .map_err(|e| JsValue::from_str(&format!("Failed to parse provider config: {}", e)))?;
        
        let provider_key = match config.provider {
            InferenceProvider::Gemini => "gemini",
            InferenceProvider::Claude => "claude",
            InferenceProvider::OpenAI => "openai",
            InferenceProvider::Deepseek => "deepseek",
        };

        self.provider_configs.insert(provider_key.to_string(), config.clone());
        
        if config.enabled {
            self.active_provider = Some(config.provider);
        }

        Ok(())
    }

    /// Set active provider
    #[wasm_bindgen]
    pub fn set_active_provider(&mut self, provider: &str) -> Result<(), JsValue> {
        let inference_provider = match provider {
            "gemini" => InferenceProvider::Gemini,
            "claude" => InferenceProvider::Claude,
            "openai" => InferenceProvider::OpenAI,
            "deepseek" => InferenceProvider::Deepseek,
            _ => return Err(JsValue::from_str("Invalid provider")),
        };

        if self.provider_configs.contains_key(provider) {
            self.active_provider = Some(inference_provider);
            Ok(())
        } else {
            Err(JsValue::from_str("Provider not configured"))
        }
    }

    /// Check if external inference is available
    #[wasm_bindgen]
    pub fn is_available(&self) -> bool {
        self.active_provider.is_some()
    }

    /// Get active provider name
    #[wasm_bindgen]
    pub fn get_active_provider(&self) -> Option<String> {
        self.active_provider.as_ref().map(|p| match p {
            InferenceProvider::Gemini => "gemini".to_string(),
            InferenceProvider::Claude => "claude".to_string(),
            InferenceProvider::OpenAI => "openai".to_string(),
            InferenceProvider::Deepseek => "deepseek".to_string(),
        })
    }

    /// Perform external inference
    #[wasm_bindgen]
    pub async fn perform_inference(&mut self, request_json: &str) -> Result<String, JsValue> {
        let request: ExternalInferenceRequest = serde_json::from_str(request_json)
            .map_err(|e| JsValue::from_str(&format!("Failed to parse request: {}", e)))?;

        let provider = self.active_provider.as_ref()
            .ok_or_else(|| JsValue::from_str("No active provider configured"))?;

        let provider_key = match provider {
            InferenceProvider::Gemini => "gemini",
            InferenceProvider::Claude => "claude",
            InferenceProvider::OpenAI => "openai",
            InferenceProvider::Deepseek => "deepseek",
        };

        let config = self.provider_configs.get(provider_key)
            .ok_or_else(|| JsValue::from_str("Provider not configured"))?;

        let start_time = js_sys::Date::now() as u32;

        match self.call_external_api(config, &request).await {
            Ok(response) => {
                let processing_time = js_sys::Date::now() as u32 - start_time;
                self.request_count += 1;
                self.total_processing_time += processing_time;

                let inference_response = ExternalInferenceResponse {
                    success: true,
                    content: response,
                    provider: provider.clone(),
                    processing_time,
                    confidence: self.calculate_confidence(&request),
                    usage: None, // Would be populated by actual API response
                    error: None,
                };

                serde_json::to_string(&inference_response)
                    .map_err(|e| JsValue::from_str(&format!("Failed to serialize response: {}", e)))
            }
            Err(error) => {
                let inference_response = ExternalInferenceResponse {
                    success: false,
                    content: String::new(),
                    provider: provider.clone(),
                    processing_time: js_sys::Date::now() as u32 - start_time,
                    confidence: 0.0,
                    usage: None,
                    error: Some(error),
                };

                serde_json::to_string(&inference_response)
                    .map_err(|e| JsValue::from_str(&format!("Failed to serialize error response: {}", e)))
            }
        }
    }

    /// Get inference statistics
    #[wasm_bindgen]
    pub fn get_stats(&self) -> String {
        let stats = serde_json::json!({
            "request_count": self.request_count,
            "total_processing_time": self.total_processing_time,
            "average_processing_time": if self.request_count > 0 {
                self.total_processing_time / self.request_count
            } else {
                0
            },
            "active_provider": self.get_active_provider(),
            "configured_providers": self.provider_configs.keys().collect::<Vec<_>>()
        });

        stats.to_string()
    }
}

impl ExternalInferenceEngine {
    /// Call external API (simplified implementation for WASM)
    async fn call_external_api(
        &self,
        config: &ProviderConfig,
        request: &ExternalInferenceRequest,
    ) -> Result<String, String> {
        // In a real implementation, this would make actual HTTP requests to the external APIs
        // For now, we'll simulate the response based on the provider
        
        let simulated_response = match config.provider {
            InferenceProvider::Gemini => {
                format!("Gemini response to: {}", request.prompt)
            }
            InferenceProvider::Claude => {
                format!("Claude response to: {}", request.prompt)
            }
            InferenceProvider::OpenAI => {
                format!("OpenAI response to: {}", request.prompt)
            }
            InferenceProvider::Deepseek => {
                format!("Deepseek response to: {}", request.prompt)
            }
        };

        // Simulate processing delay
        let promise = js_sys::Promise::resolve(&JsValue::from_str(&simulated_response));
        let _result = JsFuture::from(promise).await.map_err(|e| format!("API call failed: {:?}", e))?;

        Ok(simulated_response)
    }

    /// Calculate confidence score for the request
    fn calculate_confidence(&self, request: &ExternalInferenceRequest) -> f32 {
        let mut confidence = 0.8; // Base confidence

        // Adjust based on prompt length
        if request.prompt.len() > 50 {
            confidence += 0.1;
        }

        // Adjust based on context availability
        if request.context.is_some() {
            confidence += 0.05;
        }

        // Adjust based on conversation history
        if let Some(history) = &request.conversation_history {
            if !history.is_empty() {
                confidence += 0.05;
            }
        }

        confidence.min(1.0)
    }
}

/// Initialize external inference engine for cortex
#[wasm_bindgen]
pub fn init_external_inference() -> ExternalInferenceEngine {
    ExternalInferenceEngine::new()
}
