use bevy::prelude::*;
use knirvana_game_client::*;

#[test]
fn test_game_engine_initialization() {
    let mut app = App::new();
    
    app.add_plugins(MinimalPlugins)
       .add_plugins(GameEnginePlugin)
       .init_resource::<GameConfig>()
       .init_resource::<GameState>();
    
    app.update();
    
    // Verify resources are initialized
    let game_config = app.world.resource::<GameConfig>();
    let game_state = app.world.resource::<GameState>();
    
    assert!(game_state.is_running);
    assert_eq!(game_state.current_scene, "main_world");
}

#[test]
fn test_player_component_creation() {
    let player = Player {
        id: "test_player".to_string(),
        name: "Test Player".to_string(),
        skills: vec!["test_skill".to_string()],
        nrn_balance: 100.0,
        level: 1,
        experience: 0,
    };
    
    assert_eq!(player.id, "test_player");
    assert_eq!(player.nrn_balance, 100.0);
    assert_eq!(player.level, 1);
    assert!(player.skills.contains(&"test_skill".to_string()));
}

#[test]
fn test_reward_processing() {
    let reward = Reward {
        reward_type: "nrn".to_string(),
        amount: 50.0,
        item: None,
    };
    
    assert_eq!(reward.reward_type, "nrn");
    assert_eq!(reward.amount, 50.0);
    assert!(reward.item.is_none());
}

#[test]
fn test_mobile_settings() {
    let mobile_settings = MobileSettings {
        touch_sensitivity: 1.5,
        graphics_quality: GraphicsQuality::Low,
        battery_optimization: true,
        reduced_effects: true,
    };
    
    assert_eq!(mobile_settings.touch_sensitivity, 1.5);
    assert!(mobile_settings.battery_optimization);
    assert!(mobile_settings.reduced_effects);
}

#[test]
fn test_network_message_serialization() {
    let message = NetworkMessage {
        message_type: "player_update".to_string(),
        payload: serde_json::json!({"test": "data"}),
        timestamp: 1234567890,
    };
    
    let serialized = serde_json::to_string(&message).unwrap();
    let deserialized: NetworkMessage = serde_json::from_str(&serialized).unwrap();
    
    assert_eq!(deserialized.message_type, "player_update");
    assert_eq!(deserialized.timestamp, 1234567890);
}
