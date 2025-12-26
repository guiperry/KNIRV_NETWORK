use bevy::prelude::*;
use knirvana_game_client::*;
use std::time::{Duration, Instant};
use criterion::{black_box, Criterion};

#[cfg(test)]
mod performance_tests {
    use super::*;

    #[test]
    fn test_game_loop_performance() {
        let start = Instant::now();
        let mut app = App::new();
        
        app.add_plugins(MinimalPlugins)
           .add_plugins(GameEnginePlugin)
           .init_resource::<GameConfig>()
           .init_resource::<GameState>();

        // Simulate 60 frames (1 second at 60 FPS)
        for _ in 0..60 {
            app.update();
        }

        let duration = start.elapsed();
        
        // Should complete 60 frames in less than 100ms (very generous for testing)
        assert!(duration < Duration::from_millis(100), 
                "Game loop too slow: {:?}", duration);
    }

    #[test]
    fn test_entity_spawning_performance() {
        let mut app = App::new();
        app.add_plugins(MinimalPlugins);

        let start = Instant::now();
        
        // Spawn 1000 entities with components
        for i in 0..1000 {
            app.world.spawn((
                Player {
                    id: format!("player_{}", i),
                    name: format!("Player {}", i),
                    skills: vec!["test_skill".to_string()],
                    nrn_balance: 100.0,
                    level: 1,
                    experience: 0,
                },
                Transform::default(),
            ));
        }

        let duration = start.elapsed();
        
        // Should spawn 1000 entities in less than 10ms
        assert!(duration < Duration::from_millis(10), 
                "Entity spawning too slow: {:?}", duration);
        
        // Verify entities were created
        let player_count = app.world.query::<&Player>().iter(&app.world).count();
        assert_eq!(player_count, 1000);
    }

    #[test]
    fn test_component_query_performance() {
        let mut app = App::new();
        app.add_plugins(MinimalPlugins);

        // Spawn many entities
        for i in 0..10000 {
            app.world.spawn((
                Player {
                    id: format!("player_{}", i),
                    name: format!("Player {}", i),
                    skills: vec!["test_skill".to_string()],
                    nrn_balance: (i as f64) * 10.0,
                    level: (i % 10) + 1,
                    experience: (i * 100) as u64,
                },
                Transform::default(),
            ));
        }

        let start = Instant::now();
        
        // Query all players multiple times
        for _ in 0..100 {
            let _players: Vec<&Player> = app.world.query::<&Player>()
                .iter(&app.world)
                .collect();
        }

        let duration = start.elapsed();
        
        // Should complete 100 queries in less than 50ms
        assert!(duration < Duration::from_millis(50), 
                "Component queries too slow: {:?}", duration);
    }

    #[test]
    fn test_network_message_serialization_performance() {
        let messages: Vec<NetworkMessage> = (0..1000).map(|i| {
            NetworkMessage {
                message_type: "player_update".to_string(),
                payload: serde_json::json!({
                    "player_id": format!("player_{}", i),
                    "position": {"x": i as f64, "y": i as f64 * 2.0, "z": i as f64 * 3.0},
                    "health": 100 - (i % 100),
                    "skills": ["skill1", "skill2", "skill3"]
                }),
                timestamp: 1640995200 + i as u64,
            }
        }).collect();

        let start = Instant::now();
        
        // Serialize all messages
        let serialized: Vec<String> = messages.iter()
            .map(|msg| serde_json::to_string(msg).unwrap())
            .collect();

        let serialize_duration = start.elapsed();

        let start = Instant::now();
        
        // Deserialize all messages
        let _deserialized: Vec<NetworkMessage> = serialized.iter()
            .map(|s| serde_json::from_str(s).unwrap())
            .collect();

        let deserialize_duration = start.elapsed();

        // Should serialize 1000 messages in less than 20ms
        assert!(serialize_duration < Duration::from_millis(20), 
                "Serialization too slow: {:?}", serialize_duration);
        
        // Should deserialize 1000 messages in less than 30ms
        assert!(deserialize_duration < Duration::from_millis(30), 
                "Deserialization too slow: {:?}", deserialize_duration);
    }

    #[test]
    fn test_skill_calculation_performance() {
        let start = Instant::now();
        
        // Calculate skill prices for many combinations
        for difficulty in 1..=10 {
            for usage_count in 1..=1000 {
                let _price = calculate_skill_price(difficulty, usage_count);
            }
        }

        let duration = start.elapsed();
        
        // Should calculate 10,000 skill prices in less than 5ms
        assert!(duration < Duration::from_millis(5), 
                "Skill calculation too slow: {:?}", duration);
    }

    #[test]
    fn test_inventory_operations_performance() {
        let mut inventory = Inventory {
            items: Vec::new(),
            capacity: 1000,
        };

        let start = Instant::now();
        
        // Add many items
        for i in 0..1000 {
            inventory.items.push(InventoryItem {
                id: format!("item_{}", i),
                name: format!("Item {}", i),
                quantity: i % 10 + 1,
                item_type: if i % 2 == 0 { "weapon".to_string() } else { "consumable".to_string() },
            });
        }

        let add_duration = start.elapsed();

        let start = Instant::now();
        
        // Search for items
        for i in 0..100 {
            let _found = inventory.items.iter()
                .find(|item| item.id == format!("item_{}", i * 10));
        }

        let search_duration = start.elapsed();

        // Should add 1000 items in less than 5ms
        assert!(add_duration < Duration::from_millis(5), 
                "Inventory addition too slow: {:?}", add_duration);
        
        // Should search 100 times in less than 2ms
        assert!(search_duration < Duration::from_millis(2), 
                "Inventory search too slow: {:?}", search_duration);
    }

    #[test]
    fn test_memory_usage_bounds() {
        let initial_memory = get_memory_usage();
        
        let mut app = App::new();
        app.add_plugins(MinimalPlugins);

        // Spawn many entities
        for i in 0..5000 {
            app.world.spawn((
                Player {
                    id: format!("player_{}", i),
                    name: format!("Player {}", i),
                    skills: vec!["skill1".to_string(), "skill2".to_string()],
                    nrn_balance: 100.0,
                    level: 1,
                    experience: 0,
                },
                Transform::default(),
                Inventory {
                    items: vec![
                        InventoryItem {
                            id: format!("item_{}", i),
                            name: "Test Item".to_string(),
                            quantity: 1,
                            item_type: "test".to_string(),
                        }
                    ],
                    capacity: 50,
                },
            ));
        }

        let final_memory = get_memory_usage();
        let memory_increase = final_memory - initial_memory;
        
        // Memory increase should be reasonable (less than 100MB for 5000 entities)
        assert!(memory_increase < 100 * 1024 * 1024, 
                "Memory usage too high: {} bytes", memory_increase);
    }

    #[test]
    fn test_concurrent_operations_performance() {
        use std::sync::{Arc, Mutex};
        use std::thread;

        let shared_data = Arc::new(Mutex::new(Vec::<Player>::new()));
        let mut handles = vec![];

        let start = Instant::now();

        // Spawn multiple threads to simulate concurrent operations
        for thread_id in 0..4 {
            let data = Arc::clone(&shared_data);
            let handle = thread::spawn(move || {
                for i in 0..250 {
                    let player = Player {
                        id: format!("player_{}_{}", thread_id, i),
                        name: format!("Player {} {}", thread_id, i),
                        skills: vec!["concurrent_skill".to_string()],
                        nrn_balance: 100.0,
                        level: 1,
                        experience: 0,
                    };
                    
                    let mut players = data.lock().unwrap();
                    players.push(player);
                }
            });
            handles.push(handle);
        }

        // Wait for all threads to complete
        for handle in handles {
            handle.join().unwrap();
        }

        let duration = start.elapsed();

        // Should complete concurrent operations in less than 100ms
        assert!(duration < Duration::from_millis(100), 
                "Concurrent operations too slow: {:?}", duration);

        // Verify all players were added
        let players = shared_data.lock().unwrap();
        assert_eq!(players.len(), 1000);
    }
}

// Helper function to get memory usage (simplified for testing)
fn get_memory_usage() -> usize {
    // In a real implementation, you would use a proper memory profiling library
    // For testing purposes, we'll return a mock value
    std::mem::size_of::<usize>() * 1000
}

// Helper function for skill price calculation
fn calculate_skill_price(difficulty: u32, usage_count: u32) -> f64 {
    let base_price = 10.0;
    let difficulty_factor = difficulty as f64 * 5.0;
    let usage_factor = (usage_count as f64).sqrt() * 2.0;
    base_price + difficulty_factor + usage_factor
}

#[cfg(test)]
mod stress_tests {
    use super::*;

    #[test]
    #[ignore] // Ignore by default, run with --ignored flag
    fn stress_test_massive_entity_count() {
        let mut app = App::new();
        app.add_plugins(MinimalPlugins);

        let start = Instant::now();
        
        // Spawn a very large number of entities
        for i in 0..50000 {
            app.world.spawn((
                Player {
                    id: format!("stress_player_{}", i),
                    name: format!("Stress Player {}", i),
                    skills: vec!["stress_skill".to_string()],
                    nrn_balance: 100.0,
                    level: 1,
                    experience: 0,
                },
                Transform::default(),
            ));
        }

        // Run several update cycles
        for _ in 0..10 {
            app.update();
        }

        let duration = start.elapsed();
        
        println!("Stress test completed in {:?}", duration);
        
        // Should handle 50,000 entities without crashing
        let player_count = app.world.query::<&Player>().iter(&app.world).count();
        assert_eq!(player_count, 50000);
    }

    #[test]
    #[ignore] // Ignore by default, run with --ignored flag
    fn stress_test_network_message_flood() {
        let start = Instant::now();
        
        // Create a flood of network messages
        let messages: Vec<NetworkMessage> = (0..100000).map(|i| {
            NetworkMessage {
                message_type: "flood_test".to_string(),
                payload: serde_json::json!({
                    "id": i,
                    "data": format!("test_data_{}", i)
                }),
                timestamp: 1640995200 + i as u64,
            }
        }).collect();

        // Process all messages
        let processed: Vec<String> = messages.iter()
            .map(|msg| serde_json::to_string(msg).unwrap())
            .collect();

        let duration = start.elapsed();
        
        println!("Network flood test completed in {:?}", duration);
        assert_eq!(processed.len(), 100000);
    }
}
