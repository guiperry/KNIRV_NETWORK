use criterion::{black_box, criterion_group, criterion_main, Criterion, BenchmarkId};
use bevy::prelude::*;
use knirvana_game_client::*;
use std::time::Duration;

fn bench_entity_spawning(c: &mut Criterion) {
    let mut group = c.benchmark_group("entity_spawning");
    
    for entity_count in [100, 500, 1000, 5000].iter() {
        group.bench_with_input(
            BenchmarkId::new("spawn_players", entity_count),
            entity_count,
            |b, &entity_count| {
                b.iter(|| {
                    let mut app = App::new();
                    app.add_plugins(MinimalPlugins);
                    
                    for i in 0..entity_count {
                        app.world.spawn((
                            Player {
                                id: format!("player_{}", i),
                                name: format!("Player {}", i),
                                skills: vec!["benchmark_skill".to_string()],
                                nrn_balance: 100.0,
                                level: 1,
                                experience: 0,
                            },
                            Transform::default(),
                        ));
                    }
                    
                    black_box(app);
                });
            },
        );
    }
    
    group.finish();
}

fn bench_component_queries(c: &mut Criterion) {
    let mut group = c.benchmark_group("component_queries");
    
    // Setup app with entities
    let mut app = App::new();
    app.add_plugins(MinimalPlugins);
    
    for i in 0..10000 {
        app.world.spawn((
            Player {
                id: format!("player_{}", i),
                name: format!("Player {}", i),
                skills: vec!["query_skill".to_string()],
                nrn_balance: (i as f64) * 10.0,
                level: (i % 10) + 1,
                experience: i * 100,
            },
            Transform::default(),
            Inventory {
                items: vec![],
                capacity: 50,
            },
        ));
    }
    
    group.bench_function("query_all_players", |b| {
        b.iter(|| {
            let players: Vec<&Player> = app.world.query::<&Player>()
                .iter(&app.world)
                .collect();
            black_box(players);
        });
    });
    
    group.bench_function("query_players_with_inventory", |b| {
        b.iter(|| {
            let players: Vec<(&Player, &Inventory)> = app.world.query::<(&Player, &Inventory)>()
                .iter(&app.world)
                .collect();
            black_box(players);
        });
    });
    
    group.bench_function("query_high_level_players", |b| {
        b.iter(|| {
            let players: Vec<&Player> = app.world.query::<&Player>()
                .iter(&app.world)
                .filter(|player| player.level > 5)
                .collect();
            black_box(players);
        });
    });
    
    group.finish();
}

fn bench_network_serialization(c: &mut Criterion) {
    let mut group = c.benchmark_group("network_serialization");
    
    let messages: Vec<NetworkMessage> = (0..1000).map(|i| {
        NetworkMessage {
            message_type: "benchmark_message".to_string(),
            payload: serde_json::json!({
                "player_id": format!("player_{}", i),
                "position": {"x": i as f64, "y": i as f64 * 2.0, "z": i as f64 * 3.0},
                "health": 100 - (i % 100),
                "skills": ["skill1", "skill2", "skill3"],
                "inventory": {
                    "items": [
                        {"id": "item1", "quantity": 5},
                        {"id": "item2", "quantity": 3}
                    ]
                }
            }),
            timestamp: 1640995200 + i as u64,
        }
    }).collect();
    
    group.bench_function("serialize_messages", |b| {
        b.iter(|| {
            let serialized: Vec<String> = messages.iter()
                .map(|msg| serde_json::to_string(msg).unwrap())
                .collect();
            black_box(serialized);
        });
    });
    
    let serialized_messages: Vec<String> = messages.iter()
        .map(|msg| serde_json::to_string(msg).unwrap())
        .collect();
    
    group.bench_function("deserialize_messages", |b| {
        b.iter(|| {
            let deserialized: Vec<NetworkMessage> = serialized_messages.iter()
                .map(|s| serde_json::from_str(s).unwrap())
                .collect();
            black_box(deserialized);
        });
    });
    
    group.finish();
}

fn bench_skill_calculations(c: &mut Criterion) {
    let mut group = c.benchmark_group("skill_calculations");
    
    group.bench_function("calculate_skill_prices", |b| {
        b.iter(|| {
            let mut total_price = 0.0;
            for difficulty in 1..=10 {
                for usage_count in 1..=100 {
                    total_price += calculate_skill_price(difficulty, usage_count);
                }
            }
            black_box(total_price);
        });
    });
    
    group.bench_function("calculate_experience_gain", |b| {
        b.iter(|| {
            let mut total_exp = 0;
            for difficulty in 1..=10 {
                for completion_time in 1..=60 {
                    total_exp += calculate_experience_gain(difficulty, completion_time);
                }
            }
            black_box(total_exp);
        });
    });
    
    group.bench_function("calculate_nrn_rewards", |b| {
        b.iter(|| {
            let mut total_reward = 0.0;
            for difficulty in 1..=10 {
                for player_level in 1..=20 {
                    total_reward += calculate_nrn_reward(difficulty, player_level);
                }
            }
            black_box(total_reward);
        });
    });
    
    group.finish();
}

fn bench_inventory_operations(c: &mut Criterion) {
    let mut group = c.benchmark_group("inventory_operations");
    
    // Create a large inventory for testing
    let mut large_inventory = Inventory {
        items: Vec::new(),
        capacity: 1000,
    };
    
    for i in 0..1000 {
        large_inventory.items.push(InventoryItem {
            id: format!("item_{}", i),
            name: format!("Item {}", i),
            quantity: (i % 10) + 1,
            item_type: if i % 3 == 0 { "weapon".to_string() } 
                      else if i % 3 == 1 { "armor".to_string() } 
                      else { "consumable".to_string() },
        });
    }
    
    group.bench_function("search_by_id", |b| {
        b.iter(|| {
            for i in (0..1000).step_by(10) {
                let _found = large_inventory.items.iter()
                    .find(|item| item.id == format!("item_{}", i));
            }
        });
    });
    
    group.bench_function("filter_by_type", |b| {
        b.iter(|| {
            let weapons: Vec<&InventoryItem> = large_inventory.items.iter()
                .filter(|item| item.item_type == "weapon")
                .collect();
            black_box(weapons);
        });
    });
    
    group.bench_function("sort_by_name", |b| {
        b.iter(|| {
            let mut sorted_items = large_inventory.items.clone();
            sorted_items.sort_by(|a, b| a.name.cmp(&b.name));
            black_box(sorted_items);
        });
    });
    
    group.finish();
}

fn bench_game_loop_simulation(c: &mut Criterion) {
    let mut group = c.benchmark_group("game_loop");
    
    group.bench_function("single_frame_update", |b| {
        let mut app = App::new();
        app.add_plugins(MinimalPlugins)
           .add_plugins(GameEnginePlugin)
           .init_resource::<GameConfig>()
           .init_resource::<GameState>();
        
        // Add some entities to make it realistic
        for i in 0..100 {
            app.world.spawn((
                Player {
                    id: format!("player_{}", i),
                    name: format!("Player {}", i),
                    skills: vec!["game_loop_skill".to_string()],
                    nrn_balance: 100.0,
                    level: 1,
                    experience: 0,
                },
                Transform::default(),
            ));
        }
        
        b.iter(|| {
            app.update();
        });
    });
    
    group.bench_function("60_frame_simulation", |b| {
        b.iter(|| {
            let mut app = App::new();
            app.add_plugins(MinimalPlugins)
               .add_plugins(GameEnginePlugin)
               .init_resource::<GameConfig>()
               .init_resource::<GameState>();
            
            // Simulate 60 frames (1 second at 60 FPS)
            for _ in 0..60 {
                app.update();
            }
            
            black_box(app);
        });
    });
    
    group.finish();
}

fn bench_memory_allocations(c: &mut Criterion) {
    let mut group = c.benchmark_group("memory_allocations");
    
    group.bench_function("create_players", |b| {
        b.iter(|| {
            let players: Vec<Player> = (0..1000).map(|i| {
                Player {
                    id: format!("player_{}", i),
                    name: format!("Player {}", i),
                    skills: vec![
                        "skill1".to_string(),
                        "skill2".to_string(),
                        "skill3".to_string(),
                    ],
                    nrn_balance: 100.0,
                    level: 1,
                    experience: 0,
                }
            }).collect();
            black_box(players);
        });
    });
    
    group.bench_function("create_network_messages", |b| {
        b.iter(|| {
            let messages: Vec<NetworkMessage> = (0..1000).map(|i| {
                NetworkMessage {
                    message_type: "test_message".to_string(),
                    payload: serde_json::json!({
                        "id": i,
                        "data": format!("test_data_{}", i)
                    }),
                    timestamp: 1640995200 + i as u64,
                }
            }).collect();
            black_box(messages);
        });
    });
    
    group.finish();
}

// Helper functions for benchmarking
fn calculate_skill_price(difficulty: u32, usage_count: u32) -> f64 {
    let base_price = 10.0;
    let difficulty_factor = difficulty as f64 * 5.0;
    let usage_factor = (usage_count as f64).sqrt() * 2.0;
    base_price + difficulty_factor + usage_factor
}

fn calculate_experience_gain(difficulty: u32, completion_time_seconds: u32) -> u32 {
    let base_exp = 100;
    let difficulty_bonus = difficulty * 50;
    let time_penalty = if completion_time_seconds > 300 { 
        (completion_time_seconds - 300) / 10 
    } else { 
        0 
    };
    base_exp + difficulty_bonus - time_penalty
}

fn calculate_nrn_reward(difficulty: u32, player_level: u32) -> f64 {
    let base_reward = 50.0;
    let difficulty_multiplier = 1.0 + (difficulty as f64 * 0.2);
    let level_adjustment = 1.0 + (player_level as f64 * 0.05);
    base_reward * difficulty_multiplier * level_adjustment
}

criterion_group!(
    benches,
    bench_entity_spawning,
    bench_component_queries,
    bench_network_serialization,
    bench_skill_calculations,
    bench_inventory_operations,
    bench_game_loop_simulation,
    bench_memory_allocations
);

criterion_main!(benches);
