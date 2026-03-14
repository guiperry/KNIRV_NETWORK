#[cfg(test)]
mod tests {
    use super::*;
    use wasm_bindgen_test::*;

    wasm_bindgen_test_configure!(run_in_browser);

    #[wasm_bindgen_test]
    fn test_lora_adapter_skill_creation() {
        let skill = LoRAAdapterSkill::new(
            "test-skill-001".to_string(),
            "test-skill".to_string(),
            "Test skill for unit testing".to_string(),
            "hrm".to_string(),
            1,
            8,
            16.0,
        );

        assert_eq!(skill.skill_id(), "test-skill-001");
        assert_eq!(skill.skill_name(), "test-skill");
        assert_eq!(skill.description(), "Test skill for unit testing");
        assert_eq!(skill.version, 1);
        assert_eq!(skill.rank, 8);
        assert_eq!(skill.alpha, 16.0);
    }

    #[wasm_bindgen_test]
    fn test_error_context_creation() {
        let error_context = ErrorContext::new(
            "test-agent-123".to_string(),
            "TypeError".to_string(),
            "Cannot read property 'foo' of undefined".to_string(),
            "Process user input data".to_string(),
        );

        assert_eq!(error_context.timestamp > 0, true);
    }

    #[wasm_bindgen_test]
    fn test_skill_invocation_request_creation() {
        let request = SkillInvocationRequest::new(
            "test-invocation-001".to_string(),
            "test-agent-456".to_string(),
            "test-nrn-token-abcdef123456789012345678901234567890".to_string(),
            "knirv://skill/javascript-type-checker-v1".to_string(),
        );

        assert_eq!(request.priority, "normal");
        assert_eq!(request.timestamp > 0, true);
    }

    #[wasm_bindgen_test]
    fn test_skill_invocation_response_creation() {
        let response = SkillInvocationResponse::new(
            "test-invocation-002".to_string(),
            "SUCCESS".to_string(),
            50,
            r#"{"skill_name":"test-skill","version":1}"#.to_string(),
        );

        assert_eq!(response.invocation_id(), "test-invocation-002");
        assert_eq!(response.status(), "SUCCESS");
        assert_eq!(response.skill_data(), r#"{"skill_name":"test-skill","version":1}"#);
        assert_eq!(response.execution_time, 50);
        assert_eq!(response.memory_used, 1024);
        assert_eq!(response.consensus_reached, true);
    }

    #[wasm_bindgen_test]
    fn test_embedded_knirvchain_creation() {
        let mut chain = EmbeddedKNIRVChain::new();
        assert_eq!(chain.is_initialized(), false);
        assert_eq!(chain.get_skill_count(), 0);
    }

    #[wasm_bindgen_test]
    fn test_embedded_knirvchain_initialization() {
        let mut chain = EmbeddedKNIRVChain::new();
        
        let result = chain.initialize();
        assert!(result.is_ok());
        assert_eq!(chain.is_initialized(), true);
        assert_eq!(chain.get_skill_count(), 2); // Default skills created
    }

    #[wasm_bindgen_test]
    fn test_embedded_knirvchain_skill_registration() {
        let mut chain = EmbeddedKNIRVChain::new();
        chain.initialize().unwrap();

        let skill = LoRAAdapterSkill::new(
            "test-custom-skill-001".to_string(),
            "custom-test-skill".to_string(),
            "Custom test skill for registration".to_string(),
            "hrm".to_string(),
            1,
            16,
            32.0,
        );

        let result = chain.register_skill(skill);
        assert!(result.is_ok());
        assert_eq!(chain.get_skill_count(), 3); // 2 default + 1 custom
    }

    #[wasm_bindgen_test]
    fn test_embedded_knirvchain_skill_invocation() {
        let mut chain = EmbeddedKNIRVChain::new();
        chain.initialize().unwrap();

        let request = SkillInvocationRequest::new(
            "test-wasm-invocation-001".to_string(),
            "test-wasm-agent-123".to_string(),
            "test-nrn-token-abcdef123456789012345678901234567890".to_string(),
            "knirv://skill/javascript-type-checker-v1".to_string(),
        );

        let result = chain.invoke_skill(request);
        assert!(result.is_ok());

        let response = result.unwrap();
        assert_eq!(response.invocation_id(), "test-wasm-invocation-001");
        assert_eq!(response.status(), "SUCCESS");
        assert!(response.execution_time > 0);
        assert!(!response.skill_data().is_empty());
    }

    #[wasm_bindgen_test]
    fn test_embedded_knirvchain_skill_invocation_invalid_token() {
        let mut chain = EmbeddedKNIRVChain::new();
        chain.initialize().unwrap();

        let request = SkillInvocationRequest::new(
            "test-wasm-invocation-002".to_string(),
            "test-wasm-agent-456".to_string(),
            "short-token".to_string(), // Invalid token (too short)
            "knirv://skill/syntax-error-fixer-v2".to_string(),
        );

        let result = chain.invoke_skill(request);
        assert!(result.is_ok());

        let response = result.unwrap();
        assert_eq!(response.invocation_id(), "test-wasm-invocation-002");
        assert_eq!(response.status(), "FAILURE");
        assert!(response.skill_data().is_empty());
    }

    #[wasm_bindgen_test]
    fn test_embedded_knirvchain_skill_invocation_not_found() {
        let mut chain = EmbeddedKNIRVChain::new();
        chain.initialize().unwrap();

        let request = SkillInvocationRequest::new(
            "test-wasm-invocation-003".to_string(),
            "test-wasm-agent-789".to_string(),
            "test-nrn-token-abcdef123456789012345678901234567890".to_string(),
            "knirv://skill/non-existent-skill-v99".to_string(),
        );

        let result = chain.invoke_skill(request);
        assert!(result.is_ok());

        let response = result.unwrap();
        assert_eq!(response.invocation_id(), "test-wasm-invocation-003");
        assert_eq!(response.status(), "NOT_FOUND");
        assert!(response.skill_data().is_empty());
    }

    #[wasm_bindgen_test]
    fn test_embedded_knirvchain_not_initialized() {
        let chain = EmbeddedKNIRVChain::new();

        let request = SkillInvocationRequest::new(
            "test-wasm-invocation-004".to_string(),
            "test-wasm-agent-000".to_string(),
            "test-nrn-token-abcdef123456789012345678901234567890".to_string(),
            "knirv://skill/javascript-type-checker-v1".to_string(),
        );

        let result = chain.invoke_skill(request);
        assert!(result.is_err());
    }

    #[wasm_bindgen_test]
    fn test_get_version() {
        let version = get_version();
        assert_eq!(version, "1.0.0");
    }

    #[wasm_bindgen_test]
    fn test_get_build_info() {
        let build_info = get_build_info();
        assert_eq!(build_info, "KNIRVCHAIN WASM - Revolutionary Embedded Skill Execution Engine");
    }

    #[wasm_bindgen_test]
    fn test_skill_uri_mapping() {
        let mut chain = EmbeddedKNIRVChain::new();
        chain.initialize().unwrap();

        // Test that default skills are properly mapped
        let request1 = SkillInvocationRequest::new(
            "test-mapping-001".to_string(),
            "test-agent-mapping".to_string(),
            "test-nrn-token-abcdef123456789012345678901234567890".to_string(),
            "knirv://skill/javascript-type-checker-v1".to_string(),
        );

        let result1 = chain.invoke_skill(request1);
        assert!(result1.is_ok());
        assert_eq!(result1.unwrap().status(), "SUCCESS");

        let request2 = SkillInvocationRequest::new(
            "test-mapping-002".to_string(),
            "test-agent-mapping".to_string(),
            "test-nrn-token-abcdef123456789012345678901234567890".to_string(),
            "knirv://skill/syntax-error-fixer-v2".to_string(),
        );

        let result2 = chain.invoke_skill(request2);
        assert!(result2.is_ok());
        assert_eq!(result2.unwrap().status(), "SUCCESS");
    }

    #[wasm_bindgen_test]
    fn test_concurrent_skill_invocations() {
        let mut chain = EmbeddedKNIRVChain::new();
        chain.initialize().unwrap();

        // Test multiple concurrent invocations
        let requests = vec![
            SkillInvocationRequest::new(
                "concurrent-001".to_string(),
                "agent-001".to_string(),
                "test-nrn-token-abcdef123456789012345678901234567890".to_string(),
                "knirv://skill/javascript-type-checker-v1".to_string(),
            ),
            SkillInvocationRequest::new(
                "concurrent-002".to_string(),
                "agent-002".to_string(),
                "test-nrn-token-abcdef123456789012345678901234567890".to_string(),
                "knirv://skill/syntax-error-fixer-v2".to_string(),
            ),
            SkillInvocationRequest::new(
                "concurrent-003".to_string(),
                "agent-003".to_string(),
                "test-nrn-token-abcdef123456789012345678901234567890".to_string(),
                "knirv://skill/javascript-type-checker-v1".to_string(),
            ),
        ];

        for request in requests {
            let result = chain.invoke_skill(request);
            assert!(result.is_ok());
            assert_eq!(result.unwrap().status(), "SUCCESS");
        }
    }
}
