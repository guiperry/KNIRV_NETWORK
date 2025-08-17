use bevy::prelude::*;
use knirvana_game_client::*;
use tokio_test;
use mockall::predicate::*;
use proptest::prelude::*;

#[cfg(test)]
mod component_tests {
    use super::*;

    #[test]
    fn test_player_creation() {
        let player = Player {
            id: "test_player_123".to_string(),
            name: "Test Player".to_string(),
            skills: vec!["rust".to_string(), "debugging".to_string()],
            nrn_balance: 500.0,
            level: 5,
            experience: 1250,
        };

        assert_eq!(player.id, "test_player_123");
        assert_eq!(player.name, "Test Player");
        assert_eq!(player.nrn_balance, 500.0);
        assert_eq!(player.level, 5);
        assert_eq!(player.experience, 1250);
        assert!(player.skills.contains(&"rust".to_string()));
        assert!(player.skills.contains(&"debugging".to_string()));
    }

    #[test]
    fn test_player_skill_management() {
        let mut player = Player {
            id: "test".to_string(),
            name: "Test".to_string(),
            skills: vec!["initial_skill".to_string()],
            nrn_balance: 100.0,
            level: 1,
            experience: 0,
        };

        // Test adding skills
        player.skills.push("new_skill".to_string());
        assert!(player.skills.contains(&"new_skill".to_string()));
        assert_eq!(player.skills.len(), 2);

        // Test skill uniqueness (application logic)
        if !player.skills.contains(&"unique_skill".to_string()) {
            player.skills.push("unique_skill".to_string());
        }
        assert_eq!(player.skills.len(), 3);
    }

    #[test]
    fn test_reward_processing() {
        let nrn_reward = Reward {
            reward_type: "nrn".to_string(),
            amount: 75.0,
            item: None,
        };

        let item_reward = Reward {
            reward_type: "item".to_string(),
            amount: 1.0,
            item: Some("rare_artifact".to_string()),
        };

        assert_eq!(nrn_reward.reward_type, "nrn");
        assert_eq!(nrn_reward.amount, 75.0);
        assert!(nrn_reward.item.is_none());

        assert_eq!(item_reward.reward_type, "item");
        assert_eq!(item_reward.amount, 1.0);
        assert_eq!(item_reward.item.as_ref().unwrap(), "rare_artifact");
    }

    #[test]
    fn test_collectible_component() {
        let collectible = Collectible {
            item_type: "energy_crystal".to_string(),
            value: 25.5,
            rarity: "uncommon".to_string(),
        };

        assert_eq!(collectible.item_type, "energy_crystal");
        assert_eq!(collectible.value, 25.5);
        assert_eq!(collectible.rarity, "uncommon");
    }

    #[test]
    fn test_inventory_management() {
        let item1 = InventoryItem {
            id: "item_001".to_string(),
            name: "Health Potion".to_string(),
            quantity: 5,
            item_type: "consumable".to_string(),
        };

        let item2 = InventoryItem {
            id: "item_002".to_string(),
            name: "Magic Sword".to_string(),
            quantity: 1,
            item_type: "weapon".to_string(),
        };

        let inventory = Inventory {
            items: vec![item1, item2],
            capacity: 50,
        };

        assert_eq!(inventory.items.len(), 2);
        assert_eq!(inventory.capacity, 50);
        assert_eq!(inventory.items[0].name, "Health Potion");
        assert_eq!(inventory.items[1].quantity, 1);
    }

    #[test]
    fn test_weather_system() {
        let weather = WeatherSystem {
            weather_type: WeatherType::Rain,
            intensity: 0.7,
            particle_count: 1000,
        };

        assert!(matches!(weather.weather_type, WeatherType::Rain));
        assert_eq!(weather.intensity, 0.7);
        assert_eq!(weather.particle_count, 1000);
    }

    #[test]
    fn test_environment_component() {
        let environment = Environment {
            terrain_type: "forest".to_string(),
            weather: WeatherType::Fog,
            time_of_day: 14.5, // 2:30 PM
        };

        assert_eq!(environment.terrain_type, "forest");
        assert!(matches!(environment.weather, WeatherType::Fog));
        assert_eq!(environment.time_of_day, 14.5);
    }
}

#[cfg(test)]
mod system_tests {
    use super::*;

    #[test]
    fn test_game_state_initialization() {
        let game_state = GameState {
            is_running: true,
            current_scene: "main_world".to_string(),
            player_count: 1,
            game_mode: "single_player".to_string(),
            elapsed_time: 0.0,
        };

        assert!(game_state.is_running);
        assert_eq!(game_state.current_scene, "main_world");
        assert_eq!(game_state.player_count, 1);
        assert_eq!(game_state.game_mode, "single_player");
        assert_eq!(game_state.elapsed_time, 0.0);
    }

    #[test]
    fn test_game_config_defaults() {
        let config = GameConfig {
            graphics_quality: GraphicsQuality::Medium,
            audio_enabled: true,
            master_volume: 0.8,
            debug_mode: false,
            max_fps: 60,
        };

        assert!(matches!(config.graphics_quality, GraphicsQuality::Medium));
        assert!(config.audio_enabled);
        assert_eq!(config.master_volume, 0.8);
        assert!(!config.debug_mode);
        assert_eq!(config.max_fps, 60);
    }

    #[test]
    fn test_mobile_settings_configuration() {
        let mobile_settings = MobileSettings {
            touch_sensitivity: 2.0,
            graphics_quality: GraphicsQuality::Low,
            battery_optimization: true,
            reduced_effects: true,
        };

        assert_eq!(mobile_settings.touch_sensitivity, 2.0);
        assert!(matches!(mobile_settings.graphics_quality, GraphicsQuality::Low));
        assert!(mobile_settings.battery_optimization);
        assert!(mobile_settings.reduced_effects);
    }
}

#[cfg(test)]
mod networking_tests {
    use super::*;
    use serde_json;

    #[test]
    fn test_network_message_creation() {
        let message = NetworkMessage {
            message_type: "player_join".to_string(),
            payload: serde_json::json!({
                "player_id": "player_123",
                "player_name": "TestPlayer"
            }),
            timestamp: 1640995200, // 2022-01-01 00:00:00 UTC
        };

        assert_eq!(message.message_type, "player_join");
        assert_eq!(message.timestamp, 1640995200);
        
        let payload_obj = message.payload.as_object().unwrap();
        assert_eq!(payload_obj["player_id"], "player_123");
        assert_eq!(payload_obj["player_name"], "TestPlayer");
    }

    #[test]
    fn test_network_message_serialization() {
        let original_message = NetworkMessage {
            message_type: "game_update".to_string(),
            payload: serde_json::json!({
                "position": {"x": 10.5, "y": 20.3, "z": 5.1},
                "health": 85
            }),
            timestamp: 1640995260,
        };

        // Serialize
        let serialized = serde_json::to_string(&original_message).unwrap();
        assert!(serialized.contains("game_update"));
        assert!(serialized.contains("1640995260"));

        // Deserialize
        let deserialized: NetworkMessage = serde_json::from_str(&serialized).unwrap();
        assert_eq!(deserialized.message_type, original_message.message_type);
        assert_eq!(deserialized.timestamp, original_message.timestamp);
        
        let original_pos = &original_message.payload["position"];
        let deserialized_pos = &deserialized.payload["position"];
        assert_eq!(original_pos["x"], deserialized_pos["x"]);
        assert_eq!(original_pos["y"], deserialized_pos["y"]);
        assert_eq!(original_pos["z"], deserialized_pos["z"]);
    }

    #[test]
    fn test_connection_config() {
        let config = ConnectionConfig {
            server_url: "wss://game.knirvana.com/ws".to_string(),
            timeout_ms: 10000,
            retry_attempts: 5,
            heartbeat_interval: 30,
        };

        assert_eq!(config.server_url, "wss://game.knirvana.com/ws");
        assert_eq!(config.timeout_ms, 10000);
        assert_eq!(config.retry_attempts, 5);
        assert_eq!(config.heartbeat_interval, 30);
    }
}

#[cfg(test)]
mod economics_tests {
    use super::*;

    #[test]
    fn test_nrn_transaction() {
        let transaction = NrnTransaction {
            from: "player_001".to_string(),
            to: "player_002".to_string(),
            amount: 150.0,
            transaction_type: "skill_purchase".to_string(),
            timestamp: 1640995320,
        };

        assert_eq!(transaction.from, "player_001");
        assert_eq!(transaction.to, "player_002");
        assert_eq!(transaction.amount, 150.0);
        assert_eq!(transaction.transaction_type, "skill_purchase");
        assert_eq!(transaction.timestamp, 1640995320);
    }

    #[test]
    fn test_skill_pricing() {
        let skill_price = calculate_skill_price(5, 10); // difficulty 5, usage count 10
        assert!(skill_price > 0.0);
        assert!(skill_price < 1000.0); // Reasonable upper bound
        
        // Higher difficulty should cost more
        let high_difficulty_price = calculate_skill_price(8, 10);
        assert!(high_difficulty_price > skill_price);
        
        // Higher usage should increase price
        let high_usage_price = calculate_skill_price(5, 50);
        assert!(high_usage_price > skill_price);
    }

    #[test]
    fn test_reward_calculation() {
        let base_reward = 100.0;
        let difficulty_multiplier = 1.5;
        let time_bonus = 0.2;
        
        let total_reward = base_reward * difficulty_multiplier * (1.0 + time_bonus);
        assert_eq!(total_reward, 180.0);
    }
}

// Property-based tests using proptest
proptest! {
    #[test]
    fn test_player_balance_never_negative(
        initial_balance in 0.0f64..10000.0,
        transaction_amount in 0.0f64..1000.0
    ) {
        let mut player = Player {
            id: "test".to_string(),
            name: "Test".to_string(),
            skills: vec![],
            nrn_balance: initial_balance,
            level: 1,
            experience: 0,
        };
        
        // Simulate spending (should not go negative)
        if player.nrn_balance >= transaction_amount {
            player.nrn_balance -= transaction_amount;
        }
        
        prop_assert!(player.nrn_balance >= 0.0);
    }

    #[test]
    fn test_skill_price_monotonic(
        difficulty1 in 1u32..10,
        difficulty2 in 1u32..10,
        usage_count in 1u32..100
    ) {
        let price1 = calculate_skill_price(difficulty1, usage_count);
        let price2 = calculate_skill_price(difficulty2, usage_count);
        
        if difficulty1 < difficulty2 {
            prop_assert!(price1 <= price2);
        }
    }
}

// Helper function for testing
fn calculate_skill_price(difficulty: u32, usage_count: u32) -> f64 {
    let base_price = 10.0;
    let difficulty_factor = difficulty as f64 * 5.0;
    let usage_factor = (usage_count as f64).sqrt() * 2.0;
    base_price + difficulty_factor + usage_factor
}
