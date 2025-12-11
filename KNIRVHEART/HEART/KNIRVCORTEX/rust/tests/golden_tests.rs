use shared_types::*;
use prost::Message;

/// Golden tests for KNIRV-CORTEX ProtoBuf ABI conformance
/// These tests verify that the ABI works correctly with known inputs/outputs

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_inference_input_protobuf_encoding() {
        let input = InferenceInput {
            prompt: "Hello, world!".to_string(),
            context: "Test context".to_string(),
            config: Some(Config {
                version: "1.0".to_string(),
                max_tokens: 100,
                temperature: 0.7,
                deterministic: true,
                model_id: Some("test-model".to_string()),
                features: vec!["feature1".to_string(), "feature2".to_string()],
            }),
            memory_policy: Some(MemoryPolicy {
                short_term_window_size: 10,
                episodic_top_k: 5,
                semantic_top_k: 3,
                procedural_top_k: 2,
                relevance_threshold: 0.8,
            }),
        };

        // Test encoding
        let encoded = input.encode_to_vec();
        assert!(!encoded.is_empty(), "Encoded data should not be empty");

        // Test decoding
        let decoded = InferenceInput::decode(&encoded[..]).expect("Should decode successfully");
        assert_eq!(decoded.prompt, input.prompt);
        assert_eq!(decoded.context, input.context);
        assert!(decoded.config.is_some());
        assert!(decoded.memory_policy.is_some());
    }

    #[test]
    fn test_inference_output_protobuf_encoding() {
        let output = InferenceOutput {
            response: "Test response".to_string(),
            confidence: 0.85,
            processing_time_ms: 123.45,
            debug_info: vec![
                "Debug info 1".to_string(),
                "Debug info 2".to_string(),
            ],
        };

        // Test encoding
        let encoded = output.encode_to_vec();
        assert!(!encoded.is_empty(), "Encoded data should not be empty");

        // Test decoding
        let decoded = InferenceOutput::decode(&encoded[..]).expect("Should decode successfully");
        assert_eq!(decoded.response, output.response);
        assert_eq!(decoded.confidence, output.confidence);
        assert_eq!(decoded.processing_time_ms, output.processing_time_ms);
        assert_eq!(decoded.debug_info, output.debug_info);
    }

    #[test]
    fn test_envelope_ok_creation() {
        let test_data = b"test payload";
        let envelope = Envelope::ok(test_data.to_vec());

        assert_eq!(envelope.kind, EnvelopeKind::Ok as i32);
        assert_eq!(envelope.payload, test_data);
        assert!(envelope.timestamp > 0);
    }

    #[test]
    fn test_envelope_error_creation() {
        let error = CortexError::new(1001, "Test error message");
        let envelope = Envelope::error(error.clone());

        assert_eq!(envelope.kind, EnvelopeKind::Error as i32);
        assert!(!envelope.payload.is_empty());
        assert!(envelope.timestamp > 0);

        // Decode the error from payload
        let decoded_error = CortexError::decode(&envelope.payload[..])
            .expect("Should decode error successfully");
        assert_eq!(decoded_error.code, error.code);
        assert_eq!(decoded_error.message, error.message);
    }

    #[test]
    fn test_memory_policy_validation() {
        let policy = MemoryPolicy {
            short_term_window_size: 50,
            episodic_top_k: 10,
            semantic_top_k: 5,
            procedural_top_k: 3,
            relevance_threshold: 0.75,
        };

        // Test encoding/decoding
        let encoded = policy.encode_to_vec();
        let decoded = MemoryPolicy::decode(&encoded[..]).expect("Should decode successfully");

        assert_eq!(decoded.short_term_window_size, policy.short_term_window_size);
        assert_eq!(decoded.episodic_top_k, policy.episodic_top_k);
        assert_eq!(decoded.semantic_top_k, policy.semantic_top_k);
        assert_eq!(decoded.procedural_top_k, policy.procedural_top_k);
        assert_eq!(decoded.relevance_threshold, policy.relevance_threshold);
    }

    #[test]
    fn test_context_with_tools() {
        use std::collections::HashMap;

        let context = Context {
            short_term: "Recent conversation context".to_string(),
            episodic_items: vec![
                EpisodicItem {
                    id: "episode-1".to_string(),
                    content: "Previous interaction".to_string(),
                    timestamp: 1234567890,
                    relevance_score: 0.9,
                    tags: vec!["conversation".to_string(), "help".to_string()],
                },
            ],
            semantic_triples: vec![
                SemanticTriple {
                    subject: "user".to_string(),
                    predicate: "wants".to_string(),
                    object: "help".to_string(),
                    confidence: 0.8,
                    embedding: vec![0.1, 0.2, 0.3, 0.4, 0.5],
                },
            ],
            available_tools: vec![
                Tool {
                    id: "calc-1".to_string(),
                    name: "calculator".to_string(),
                    description: "Performs mathematical calculations".to_string(),
                    schema: Some(ToolSchema {
                        r#type: "function".to_string(),
                        parameters: HashMap::new(),
                        required: vec!["number1".to_string(), "number2".to_string()],
                    }),
                    relevance_score: 0.95,
                },
            ],
        };

        // Test encoding/decoding
        let encoded = context.encode_to_vec();
        let decoded = Context::decode(&encoded[..]).expect("Should decode successfully");

        assert_eq!(decoded.short_term, context.short_term);
        assert_eq!(decoded.episodic_items.len(), 1);
        assert_eq!(decoded.semantic_triples.len(), 1);
        assert_eq!(decoded.available_tools.len(), 1);
        assert_eq!(decoded.available_tools[0].name, "calculator");
        assert_eq!(decoded.episodic_items[0].id, "episode-1");
        assert_eq!(decoded.semantic_triples[0].embedding.len(), 5);
    }

    #[test]
    fn test_error_codes_constants() {
        use shared_types::error_codes::*;

        // Verify error codes are defined and unique
        let codes = vec![
            INVALID_INPUT,
            PROCESSING_FAILED,
            MEMORY_LIMIT_EXCEEDED,
            TIMEOUT,
            MODEL_NOT_LOADED,
            UNSUPPORTED_OPERATION,
            RUNTIME_ERROR,
        ];

        // Check all codes are different
        for i in 0..codes.len() {
            for j in i + 1..codes.len() {
                assert_ne!(codes[i], codes[j], "Error codes should be unique");
            }
        }

        // Check codes are in expected range
        for code in codes {
            assert!(code >= 1000, "Error codes should be >= 1000");
            assert!(code < 2000, "Error codes should be < 2000");
        }
    }

    #[test]
    fn test_abi_pointer_length_packing() {
        use shared_types::abi::*;

        // Test packing and unpacking with simulated WASM 32-bit addresses
        // In WASM, pointers are 32-bit offsets into linear memory
        let simulated_wasm_ptr = 0x1000u32 as *const u8;
        let len = 42usize;

        let packed = pack_ptr_len(simulated_wasm_ptr, len);
        let (unpacked_ptr, unpacked_len) = unpack_ptr_len(packed);

        // In WASM context, we only care about the lower 32 bits
        assert_eq!(unpacked_ptr as usize & 0xFFFFFFFF, simulated_wasm_ptr as usize);
        assert_eq!(unpacked_len, len);

        // Test with zero values
        let packed_zero = pack_ptr_len(std::ptr::null(), 0);
        let (zero_ptr, zero_len) = unpack_ptr_len(packed_zero);
        assert_eq!(zero_ptr, std::ptr::null());
        assert_eq!(zero_len, 0);

        // Test the packing format directly
        let test_ptr = 0x12345678u32 as *const u8;
        let test_len = 0x9ABCDEFusize;
        let packed_test = pack_ptr_len(test_ptr, test_len);

        // Verify the bit layout: high 32 bits = ptr, low 32 bits = len
        let expected = ((test_ptr as u64) << 32) | (test_len as u64 & 0xFFFFFFFF);
        assert_eq!(packed_test, expected);
    }

    #[test]
    fn test_envelope_with_trace_id() {
        let test_data = b"traced payload";
        let trace_id = "trace-123-456-789";

        let envelope = Envelope::ok(test_data.to_vec()).with_trace_id(trace_id.to_string());

        assert_eq!(envelope.kind, EnvelopeKind::Ok as i32);
        assert_eq!(envelope.payload, test_data);
        assert_eq!(envelope.trace_id, Some(trace_id.to_string()));
        assert!(envelope.timestamp > 0);
    }

    #[test]
    fn test_config_optional_fields() {
        // Test config with minimal fields
        let minimal_config = Config {
            version: "1.0".to_string(),
            max_tokens: 50,
            temperature: 0.5,
            deterministic: false,
            model_id: None,
            features: vec![],
        };

        let encoded = minimal_config.encode_to_vec();
        let decoded = Config::decode(&encoded[..]).expect("Should decode successfully");

        assert_eq!(decoded.version, minimal_config.version);
        assert_eq!(decoded.max_tokens, minimal_config.max_tokens);
        assert_eq!(decoded.temperature, minimal_config.temperature);
        assert_eq!(decoded.deterministic, minimal_config.deterministic);
        assert!(decoded.model_id.is_none());
        assert!(decoded.features.is_empty());
    }

    #[test]
    fn test_large_payload_handling() {
        // Test with a large prompt to ensure the ABI can handle substantial data
        let large_prompt = "A".repeat(10000); // 10KB prompt
        let large_context = "B".repeat(5000);  // 5KB context

        let input = InferenceInput {
            prompt: large_prompt.clone(),
            context: large_context.clone(),
            config: None,
            memory_policy: None,
        };

        let encoded = input.encode_to_vec();
        assert!(encoded.len() > 15000, "Encoded size should be substantial");

        let decoded = InferenceInput::decode(&encoded[..]).expect("Should decode large payload");
        assert_eq!(decoded.prompt, large_prompt);
        assert_eq!(decoded.context, large_context);
    }
}
