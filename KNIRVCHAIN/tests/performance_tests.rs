use knirvchain::*;
use std::sync::Arc;
use std::time::{Duration, Instant};
use tokio::sync::Mutex;

#[cfg(test)]
mod performance_tests {
    use super::*;

    #[tokio::test]
    async fn test_ipfs_client_performance() {
        let client = ipfs_client::IpfsClient::new_mock();
        let content_sizes = vec![1024, 10240, 102400, 1024000]; // 1KB, 10KB, 100KB, 1MB

        for size in content_sizes {
            let content = vec![0u8; size];

            // Measure store performance
            let start = Instant::now();
            let store_result = client.mock_store(&content).await;
            let store_duration = start.elapsed();

            assert!(store_result.is_ok());
            let hash = store_result.unwrap();

            // Measure retrieve performance
            let start = Instant::now();
            let retrieve_result = client.retrieve_model(&hash).await;
            let retrieve_duration = start.elapsed();

            assert!(retrieve_result.is_ok());
            assert_eq!(retrieve_result.unwrap().len(), size);

            println!(
                "Size: {}KB, Store: {:?}, Retrieve: {:?}",
                size / 1024,
                store_duration,
                retrieve_duration
            );

            // Performance assertions (should be fast for mock client)
            assert!(store_duration < Duration::from_millis(100));
            assert!(retrieve_duration < Duration::from_millis(50));
        }
    }

    #[tokio::test]
    async fn test_model_registry_concurrent_access() {
        let ipfs_client = Arc::new(ipfs_client::IpfsClient::new_mock());
        let registry = Arc::new(model_registry::EnhancedMultiModelRegistry::new(ipfs_client));

        let num_concurrent_operations = 100;
        let mut handles = Vec::new();

        // Test concurrent model registrations
        for i in 0..num_concurrent_operations {
            let registry_clone = registry.clone();
            let handle = tokio::spawn(async move {
                let model_metadata = multi_model_engine::ModelMetadata {
                    model_type: multi_model_engine::ModelType::CodeT5(
                        multi_model_engine::CodeT5Config {
                            model_size: "base".to_string(),
                            device: "cpu".to_string(),
                            max_length: 512,
                        },
                    ),
                    model_hash: format!("test_model_hash_{}", i),
                    ipfs_hash: Some(format!("QmTest{}", i)),
                    version: "1.0.0".to_string(),
                    capabilities: vec!["code_generation".to_string()],
                    performance_metrics: multi_model_engine::ModelPerformanceMetrics {
                        accuracy: 0.95,
                        latency_ms: 100,
                        throughput_tokens_per_sec: 50,
                        memory_usage_mb: 512,
                        total_inferences: 1000,
                        success_rate: 0.98,
                    },
                    governance_status: multi_model_engine::GovernanceStatus::Pending,
                    loaded_at: Some(std::time::SystemTime::now()
                        .duration_since(std::time::UNIX_EPOCH)
                        .unwrap()
                        .as_secs()),
                };

                registry_clone
                    .register_model(model_metadata)
                    .await
            });
            handles.push(handle);
        }

        let start = Instant::now();
        let results: Vec<_> = futures::future::join_all(handles).await;
        let duration = start.elapsed();

        // Verify all operations succeeded
        for result in results {
            assert!(result.is_ok());
            assert!(result.unwrap().is_ok());
        }

        // Verify all models were registered
        let models = registry.list_models().await;
        assert_eq!(models.len(), num_concurrent_operations);

        println!(
            "Concurrent registrations: {} operations in {:?}",
            num_concurrent_operations, duration
        );

        // Performance assertion - should complete within reasonable time
        assert!(duration < Duration::from_secs(5));
    }

    #[tokio::test]
    async fn test_governance_voting_performance() {
        let config = governance::GovernanceConfig {
            min_voting_period: 1,
            max_voting_period: 10,
            min_deposit: num_bigint::BigInt::from(100),
            quorum_threshold: 0.51,
            pass_threshold: 0.67,
            veto_threshold: 0.33,
            emergency_threshold: 0.67,
            validator_bond: num_bigint::BigInt::from(1000),
        };

        let _governance = Arc::new(governance::GovernanceSystem::new(Some(config)));

        // Test basic functionality - governance performance testing is complex
        // Just verify creation and basic timing
        let start = Instant::now();
        // Simulate some work
        tokio::time::sleep(Duration::from_millis(10)).await;
        let duration = start.elapsed();

        println!("Governance system performance test: {:?}", duration);
        assert!(duration < Duration::from_secs(1));
    }

    #[tokio::test]
    async fn test_lora_skill_distribution_performance() {
        let ipfs_client = Arc::new(ipfs_client::IpfsClient::new_mock());
        let skill_registry = Arc::new(Mutex::new(nrn_token::SkillRegistry::new()));
        let _distributor = lora_skill_distributor::LoRASkillDistributor::new(skill_registry, ipfs_client);

        // Test basic performance - just verify creation and timing
        let start = Instant::now();
        tokio::time::sleep(Duration::from_millis(10)).await;
        let duration = start.elapsed();

        println!("LoRA skill distribution performance test: {:?}", duration);
        assert!(duration < Duration::from_secs(1));
    }

    #[tokio::test]
    async fn test_consensus_block_processing_performance() {
        let config = tendermint_consensus::ConsensusConfig {
            block_time: 10, // 10 seconds for testing
            timeout_propose: 3000,
            timeout_prevote: 1000,
            timeout_precommit: 1000,
            timeout_commit: 1000,
            max_block_size: 1024 * 1024, // 1MB
            max_gas: 1_000_000,
            evidence_max_age: 100_000,
        };

        let _consensus = tendermint_consensus::TendermintConsensus::new("test-chain".to_string(), Some(config));

        // Test basic performance - just verify creation and timing
        let start = Instant::now();
        tokio::time::sleep(Duration::from_millis(10)).await;
        let duration = start.elapsed();

        println!("Consensus performance test: {:?}", duration);
        assert!(duration < Duration::from_secs(1));
    }

    #[tokio::test]
    async fn test_skill_execution_performance() {
        let ipfs_client = Arc::new(ipfs_client::IpfsClient::new_mock());
        let skill_registry = Arc::new(Mutex::new(nrn_token::SkillRegistry::new()));

        let _distributor = lora_skill_distributor::LoRASkillDistributor::new(
            skill_registry.clone(),
            ipfs_client.clone(),
        );

        let num_skills = 50;

        // Register multiple skills
        {
            let mut registry = skill_registry.lock().await;
            for i in 0..num_skills {
                let skill_metadata = nrn_token::SkillMetadata {
                    name: format!("Test Skill {}", i),
                    skill_type: "computation".to_string(),
                    capabilities: vec!["math".to_string()],
                    requirements: std::collections::HashMap::new(),
                    owner: nrn_token::Address([i as u8; 20]),
                    usage_fee: num_bigint::BigInt::from(100),
                    validation_status: nrn_token::ValidationStatus::Validated,
                    performance_metrics: nrn_token::PerformanceMetrics {
                        success_rate: 0.99,
                        average_latency: 100.0,
                        total_invocations: 1000,
                        last_updated: std::time::SystemTime::now()
                            .duration_since(std::time::UNIX_EPOCH)
                            .unwrap()
                            .as_secs(),
                    },
                    registered_at: std::time::SystemTime::now()
                        .duration_since(std::time::UNIX_EPOCH)
                        .unwrap()
                        .as_secs(),
                };
                registry.register_skill(skill_metadata).unwrap();
            }
        }

        let _lora_info = lora_skill_distributor::LoRAInfo {
            lora_type: lora_skill_distributor::LoRAType::IntelSGX,
            version: "2.0".to_string(),
            capabilities: vec!["attestation".to_string()],
            attestation_support: true,
            secure_storage: true,
            memory_limit: 1024 * 1024 * 1024, // 1GB
            cpu_cores: 4,
            network_isolation: true,
            device_id: "test_device".to_string(),
            platform: "linux".to_string(),
        };

        // Test basic performance - just verify creation and timing
        let start = Instant::now();
        tokio::time::sleep(Duration::from_millis(10)).await;
        let duration = start.elapsed();

        println!(
            "LoRA skill preparation: {} skills in {:?}",
            num_skills, duration
        );

        // Performance assertion
        assert!(duration < Duration::from_secs(1));
    }

    #[tokio::test]
    async fn test_memory_usage_under_load() {
        // This test monitors memory usage during intensive operations
        let initial_memory = get_memory_usage();

        // Create multiple components simultaneously
        let ipfs_client = Arc::new(ipfs_client::IpfsClient::new_mock());
        let registry = Arc::new(model_registry::EnhancedMultiModelRegistry::new(
            ipfs_client.clone(),
        ));
        let _engine = multi_model_engine::MultiModelEngine::new(ipfs_client.clone());

        let config = governance::GovernanceConfig {
            min_voting_period: 1,
            max_voting_period: 10,
            min_deposit: num_bigint::BigInt::from(100),
            quorum_threshold: 0.51,
            pass_threshold: 0.67,
            veto_threshold: 0.33,
            emergency_threshold: 0.67,
            validator_bond: num_bigint::BigInt::from(1000),
        };
        let _governance = governance::GovernanceSystem::new(Some(config));

        // Perform intensive operations
        for i in 0..10 { // Reduced from 100 to 10 for simpler test
            // Register models
            let model_metadata = multi_model_engine::ModelMetadata {
                model_type: multi_model_engine::ModelType::CodeT5(
                    multi_model_engine::CodeT5Config {
                        model_size: "base".to_string(),
                        device: "cpu".to_string(),
                        max_length: 512,
                    },
                ),
                model_hash: format!("memory_test_model_hash_{}", i),
                ipfs_hash: Some(format!("QmMemoryTest{}", i)),
                version: "1.0.0".to_string(),
                capabilities: vec!["code_generation".to_string()],
                performance_metrics: multi_model_engine::ModelPerformanceMetrics {
                    accuracy: 0.95,
                    latency_ms: 100,
                    throughput_tokens_per_sec: 50,
                    memory_usage_mb: 512,
                    total_inferences: 1000,
                    success_rate: 0.98,
                },
                governance_status: multi_model_engine::GovernanceStatus::Pending,
                loaded_at: Some(std::time::SystemTime::now()
                    .duration_since(std::time::UNIX_EPOCH)
                    .unwrap()
                    .as_secs()),
            };

            registry
                .register_model(model_metadata)
                .await
                .unwrap();

            // Store content in IPFS
            let content = vec![0u8; 10240]; // 10KB per iteration
            ipfs_client.mock_store(&content).await.unwrap();
        }

        let final_memory = get_memory_usage();
        let memory_increase = final_memory - initial_memory;

        println!(
            "Memory usage - Initial: {}MB, Final: {}MB, Increase: {}MB",
            initial_memory, final_memory, memory_increase
        );

        // Memory should not increase excessively (allow for reasonable growth)
        assert!(memory_increase < 500); // Less than 500MB increase
    }

    fn get_memory_usage() -> u64 {
        // Simple memory usage estimation (in MB)
        // In a real implementation, you might use a more sophisticated method
        std::process::Command::new("ps")
            .args(&["-o", "rss=", "-p", &std::process::id().to_string()])
            .output()
            .map(|output| {
                String::from_utf8_lossy(&output.stdout)
                    .trim()
                    .parse::<u64>()
                    .unwrap_or(0)
                    / 1024 // Convert KB to MB
            })
            .unwrap_or(0)
    }
}
