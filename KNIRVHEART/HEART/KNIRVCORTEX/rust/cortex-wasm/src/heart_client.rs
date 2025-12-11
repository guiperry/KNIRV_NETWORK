// HEART Client Module for KNIRVCORTEX
// Provides interface for stem.cortex to query HEART transformer for heuristic analysis

use wasm_bindgen::prelude::*;
use wasm_bindgen::JsCast;
use wasm_bindgen_futures::JsFuture;
use web_sys::{Request, RequestInit, RequestMode, Response};
use serde::{Deserialize, Serialize};
use std::collections::HashMap;

// Import the `console.log` function
#[wasm_bindgen]
extern "C" {
    #[wasm_bindgen(js_namespace = console)]
    fn log(s: &str);
}

macro_rules! console_log {
    ($($t:tt)*) => (log(&format_args!($($t)*).to_string()))
}

/// HEART client for error inquiry and heuristic analysis
pub struct HEARTClient {
    endpoint: String,
    timeout_ms: u32,
    min_confidence: f32,
    stats: HEARTClientStats,
    use_local_fallback: bool,
}

#[derive(Clone)]
struct HEARTClientStats {
    total_inquiries: u64,
    successful_responses: u64,
    failed_responses: u64,
    total_response_time_ms: f64,
}

impl HEARTClientStats {
    fn new() -> Self {
        Self {
            total_inquiries: 0,
            successful_responses: 0,
            failed_responses: 0,
            total_response_time_ms: 0.0,
        }
    }

    fn avg_response_time(&self) -> f64 {
        if self.successful_responses > 0 {
            self.total_response_time_ms / self.successful_responses as f64
        } else {
            0.0
        }
    }
}

/// Configuration for HEART client
#[derive(Serialize, Deserialize, Clone)]
pub struct HEARTConfig {
    pub endpoint: String,
    pub timeout_ms: u32,
    pub min_confidence_threshold: f32,
    pub enable_pattern_recognition: bool,
    pub enable_similarity_search: bool,
    pub fallback_to_local: bool,
}

impl Default for HEARTConfig {
    fn default() -> Self {
        Self {
            endpoint: "http://localhost:8080/heart/analyze".to_string(),
            timeout_ms: 5000,
            min_confidence_threshold: 0.7,
            enable_pattern_recognition: true,
            enable_similarity_search: true,
            fallback_to_local: true,
        }
    }
}

/// Error inquiry sent to HEART
#[derive(Serialize, Deserialize, Clone)]
pub struct HEARTErrorInquiry {
    pub error_id: String,
    pub error_type: String,
    pub error_message: String,
    pub error_context: String,
    pub stack_trace: Option<String>,
    pub metadata: HashMap<String, String>,
    pub prompt: Option<String>,
    pub model_response: Option<String>,
    pub confidence_score: Option<f32>,
    pub timestamp: u64,
}

/// HEART's heuristic response
#[derive(Serialize, Deserialize, Clone, Debug)]
pub struct HEARTHeuristicResponse {
    pub inquiry_id: String,
    pub command_vector: Vec<f32>,
    pub alert_level: u32,
    pub heuristic_id: u32,
    pub target_node_id: u32,
    pub confidence_score: f32,
    pub action_parameters: Vec<f32>,
    pub analysis_summary: String,
    pub recommended_actions: Vec<String>,
    pub debug_insights: Vec<String>,
    pub identified_patterns: Vec<ErrorPattern>,
    pub similar_errors: Vec<SimilarError>,
    pub processing_time_ms: f32,
    pub timestamp: u64,
}

#[derive(Serialize, Deserialize, Clone, Debug)]
pub struct ErrorPattern {
    pub pattern_id: String,
    pub pattern_name: String,
    pub description: String,
    pub match_confidence: f32,
    pub indicators: Vec<String>,
}

#[derive(Serialize, Deserialize, Clone, Debug)]
pub struct SimilarError {
    pub error_id: String,
    pub error_type: String,
    pub similarity_score: f32,
    pub resolution: String,
    pub skill_id: Option<String>,
}

impl HEARTClient {
    /// Create a new HEART client with configuration
    pub fn new(config: HEARTConfig) -> Self {
        console_log!("Initializing HEART client with endpoint: {}", config.endpoint);

        Self {
            endpoint: config.endpoint,
            timeout_ms: config.timeout_ms,
            min_confidence: config.min_confidence_threshold,
            stats: HEARTClientStats::new(),
            use_local_fallback: config.fallback_to_local,
        }
    }

    /// Query HEART for heuristic analysis of an error
    pub async fn query_heart(&mut self, inquiry: HEARTErrorInquiry) -> Result<HEARTHeuristicResponse, String> {
        console_log!("Querying HEART for error: {} (type: {})", inquiry.error_id, inquiry.error_type);

        self.stats.total_inquiries += 1;
        let start_time = js_sys::Date::now();

        // Try to query HEART service
        match self.query_heart_service(&inquiry).await {
            Ok(response) => {
                let elapsed = js_sys::Date::now() - start_time;
                self.stats.successful_responses += 1;
                self.stats.total_response_time_ms += elapsed;

                console_log!("HEART response received: confidence={:.2}, alert_level={}",
                            response.confidence_score, response.alert_level);

                // Check confidence threshold
                if response.confidence_score >= self.min_confidence {
                    Ok(response)
                } else {
                    console_log!("HEART confidence too low ({:.2} < {:.2}), using fallback",
                                response.confidence_score, self.min_confidence);
                    if self.use_local_fallback {
                        Ok(self.local_heuristic_fallback(&inquiry))
                    } else {
                        Err(format!("HEART confidence too low: {:.2}", response.confidence_score))
                    }
                }
            }
            Err(e) => {
                self.stats.failed_responses += 1;
                console_log!("HEART query failed: {}. Using fallback.", e);

                if self.use_local_fallback {
                    Ok(self.local_heuristic_fallback(&inquiry))
                } else {
                    Err(format!("HEART query failed: {}", e))
                }
            }
        }
    }

    /// Query the HEART service via HTTP
    async fn query_heart_service(&self, inquiry: &HEARTErrorInquiry) -> Result<HEARTHeuristicResponse, String> {
        let window = web_sys::window().ok_or("No window object")?;

        // Serialize inquiry to JSON
        let inquiry_json = serde_json::to_string(inquiry)
            .map_err(|e| format!("Failed to serialize inquiry: {}", e))?;

        // Create request
        let mut opts = RequestInit::new();
        opts.method("POST");
        opts.mode(RequestMode::Cors);
        opts.body(Some(&JsValue::from_str(&inquiry_json)));

        let request = Request::new_with_str_and_init(&self.endpoint, &opts)
            .map_err(|e| format!("Failed to create request: {:?}", e))?;

        request.headers().set("Content-Type", "application/json")
            .map_err(|e| format!("Failed to set header: {:?}", e))?;

        // Fetch with timeout
        let resp_value = JsFuture::from(window.fetch_with_request(&request))
            .await
            .map_err(|e| format!("Fetch failed: {:?}", e))?;

        let resp: Response = resp_value.dyn_into()
            .map_err(|_| "Failed to cast response".to_string())?;

        if !resp.ok() {
            return Err(format!("HTTP error: {}", resp.status()));
        }

        // Parse response
        let json = JsFuture::from(resp.json().map_err(|e| format!("JSON parse error: {:?}", e))?)
            .await
            .map_err(|e| format!("JSON future error: {:?}", e))?;

        let response_text = js_sys::JSON::stringify(&json)
            .map_err(|_| "Failed to stringify response".to_string())?
            .as_string()
            .ok_or("Response not a string".to_string())?;

        serde_json::from_str(&response_text)
            .map_err(|e| format!("Failed to deserialize response: {}", e))
    }

    /// Local heuristic fallback when HEART is unavailable
    fn local_heuristic_fallback(&self, inquiry: &HEARTErrorInquiry) -> HEARTHeuristicResponse {
        console_log!("Using local heuristic fallback for: {}", inquiry.error_type);

        // Simple rule-based heuristics
        let (alert_level, heuristic_id, analysis, actions) = match inquiry.error_type.as_str() {
            "TypeError" | "ReferenceError" => (
                3,
                101,
                "Type-related error detected. Check variable types and initialization.".to_string(),
                vec![
                    "Review variable declarations".to_string(),
                    "Check for undefined references".to_string(),
                    "Validate input types".to_string(),
                ],
            ),
            "NetworkError" | "FetchError" => (
                4,
                201,
                "Network connectivity issue. Check endpoints and network status.".to_string(),
                vec![
                    "Verify endpoint availability".to_string(),
                    "Check network connectivity".to_string(),
                    "Review CORS configuration".to_string(),
                ],
            ),
            "InferenceError" | "ModelError" => (
                5,
                301,
                "Model inference failure. Check model weights and input format.".to_string(),
                vec![
                    "Verify model is loaded".to_string(),
                    "Validate input shape and format".to_string(),
                    "Check model compatibility".to_string(),
                ],
            ),
            _ => (
                2,
                999,
                "Generic error detected. Review error context and stack trace.".to_string(),
                vec![
                    "Review error message".to_string(),
                    "Check stack trace".to_string(),
                    "Validate input data".to_string(),
                ],
            ),
        };

        // Generate command vector (8 floats)
        let command_vector = vec![
            alert_level as f32,
            heuristic_id as f32,
            0.0, // target_node_id
            0.6, // confidence (lower for fallback)
            1.0, // action param 1: enable logging
            0.0, // action param 2
            0.0, // action param 3
            0.0, // action param 4
        ];

        HEARTHeuristicResponse {
            inquiry_id: inquiry.error_id.clone(),
            command_vector,
            alert_level,
            heuristic_id,
            target_node_id: 0,
            confidence_score: 0.6,
            action_parameters: vec![1.0, 0.0, 0.0, 0.0],
            analysis_summary: analysis,
            recommended_actions: actions,
            debug_insights: vec![
                "Using local heuristic fallback".to_string(),
                format!("Error type: {}", inquiry.error_type),
            ],
            identified_patterns: vec![],
            similar_errors: vec![],
            processing_time_ms: 1.0,
            timestamp: js_sys::Date::now() as u64,
        }
    }

    /// Check HEART service health
    pub async fn health_check(&self) -> Result<bool, String> {
        // Simple ping to endpoint
        let health_endpoint = format!("{}/health", self.endpoint.trim_end_matches("/analyze"));

        let window = web_sys::window().ok_or("No window")?;
        let request = Request::new_with_str(&health_endpoint)
            .map_err(|e| format!("Request error: {:?}", e))?;

        match JsFuture::from(window.fetch_with_request(&request)).await {
            Ok(resp_value) => {
                let resp: Response = resp_value.dyn_into()
                    .map_err(|_| "Cast error".to_string())?;
                Ok(resp.ok())
            }
            Err(_) => Ok(false),
        }
    }

    /// Get client statistics
    pub fn get_stats(&self) -> String {
        serde_json::json!({
            "total_inquiries": self.stats.total_inquiries,
            "successful_responses": self.stats.successful_responses,
            "failed_responses": self.stats.failed_responses,
            "avg_response_time_ms": self.stats.avg_response_time(),
            "success_rate": if self.stats.total_inquiries > 0 {
                self.stats.successful_responses as f64 / self.stats.total_inquiries as f64
            } else {
                0.0
            }
        }).to_string()
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_heart_config_default() {
        let config = HEARTConfig::default();
        assert_eq!(config.timeout_ms, 5000);
        assert_eq!(config.min_confidence_threshold, 0.7);
        assert!(config.fallback_to_local);
    }

    #[test]
    fn test_local_fallback() {
        let config = HEARTConfig::default();
        let client = HEARTClient::new(config);

        let inquiry = HEARTErrorInquiry {
            error_id: "test-123".to_string(),
            error_type: "TypeError".to_string(),
            error_message: "Cannot read property of undefined".to_string(),
            error_context: "Test context".to_string(),
            stack_trace: None,
            metadata: HashMap::new(),
            prompt: None,
            model_response: None,
            confidence_score: None,
            timestamp: 0,
        };

        let response = client.local_heuristic_fallback(&inquiry);
        assert_eq!(response.alert_level, 3);
        assert_eq!(response.heuristic_id, 101);
        assert!(response.confidence_score > 0.0);
    }
}
