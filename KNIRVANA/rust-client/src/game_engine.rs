use bevy::prelude::*;
use bevy_rapier3d::prelude::*;
use serde::{Deserialize, Serialize};
use std::collections::HashMap;

pub struct GameEnginePlugin;

impl Plugin for GameEnginePlugin {
    fn build(&self, app: &mut App) {
        app
            .add_event::<InteractionEvent>()
            .add_event::<ChallengeEvent>()
            .add_event::<RewardEvent>()
            .add_systems(Startup, initialize_game_engine)
            .add_systems(Update, (
                handle_interactions,
                update_challenges,
                process_rewards,
            ));
    }
}

#[derive(Resource, Default)]
pub struct GameConfig {
    pub enable_vr: bool,
    pub enable_ar: bool,
    pub enable_physics: bool,
    pub enable_networking: bool,
    pub api_endpoint: String,
    pub mobile_optimized: bool,
}

#[derive(Resource, Default)]
pub struct GameState {
    pub is_running: bool,
    pub current_scene: String,
    pub player_count: u32,
    pub active_challenges: Vec<String>,
}

#[derive(Component, Serialize, Deserialize, Clone)]
pub struct Player {
    pub id: String,
    pub name: String,
    pub skills: Vec<String>,
    pub nrn_balance: f64,
    pub level: u32,
    pub experience: u64,
}

#[derive(Component)]
pub struct PlayerController {
    pub move_speed: f32,
    pub jump_force: f32,
    pub is_grounded: bool,
}

#[derive(Component, Serialize, Deserialize)]
pub struct NPC {
    pub id: String,
    pub name: String,
    pub npc_type: String,
    pub dialogue: DialogueTree,
    pub skills: Vec<String>,
    pub quest_giver: bool,
}

#[derive(Component)]
pub struct NPCAi {
    pub behavior_type: String,
    pub interaction_range: f32,
    pub last_interaction: f64,
}

#[derive(Component, Serialize, Deserialize)]
pub struct Interactable {
    pub id: String,
    pub interaction_type: String,
    pub action: String,
    pub requirements: Vec<String>,
    pub rewards: Vec<Reward>,
}

#[derive(Component, Serialize, Deserialize)]
pub struct Challenge {
    pub id: String,
    pub name: String,
    pub description: String,
    pub challenge_type: String,
    pub difficulty: u32,
    pub requirements: Vec<String>,
    pub rewards: Vec<Reward>,
    pub time_limit: Option<u64>,
    pub is_active: bool,
}

#[derive(Serialize, Deserialize, Clone)]
pub struct Reward {
    pub reward_type: String,
    pub amount: f64,
    pub item: Option<String>,
}

#[derive(Serialize, Deserialize, Clone)]
pub struct DialogueTree {
    pub nodes: HashMap<String, DialogueNode>,
    pub current_node: String,
}

#[derive(Serialize, Deserialize, Clone)]
pub struct DialogueNode {
    pub id: String,
    pub text: String,
    pub speaker: String,
    pub options: Vec<DialogueOption>,
}

#[derive(Serialize, Deserialize, Clone)]
pub struct DialogueOption {
    pub text: String,
    pub next_node: String,
    pub requirements: Option<Vec<String>>,
    pub action: Option<String>,
}

#[derive(Event)]
pub struct InteractionEvent {
    pub player_entity: Entity,
    pub target_entity: Entity,
    pub interaction_type: String,
}

#[derive(Event)]
pub struct ChallengeEvent {
    pub challenge_id: String,
    pub event_type: String, // "start", "complete", "fail"
    pub player_entity: Entity,
}

#[derive(Event)]
pub struct RewardEvent {
    pub player_entity: Entity,
    pub rewards: Vec<Reward>,
}

fn initialize_game_engine(
    mut commands: Commands,
    mut config: ResMut<GameConfig>,
    mut game_state: ResMut<GameState>,
) {
    info!("Initializing KNIRVANA Game Engine (Rust)");

    // Configure for mobile optimization
    #[cfg(feature = "mobile")]
    {
        config.mobile_optimized = true;
        config.enable_physics = true; // Simplified physics for mobile
        info!("Mobile optimization enabled");
    }

    #[cfg(not(feature = "mobile"))]
    {
        config.mobile_optimized = false;
        config.enable_physics = true;
        config.enable_vr = true;
        info!("Desktop configuration loaded");
    }

    config.enable_networking = true;
    config.api_endpoint = "https://api.knirv.com".to_string();

    game_state.is_running = true;
    game_state.current_scene = "main_world".to_string();

    info!("Game engine initialized successfully");
}

fn handle_interactions(
    mut interaction_events: EventReader<InteractionEvent>,
    mut challenge_events: EventWriter<ChallengeEvent>,
    mut reward_events: EventWriter<RewardEvent>,
    players: Query<&Player>,
    interactables: Query<&Interactable>,
    npcs: Query<&NPC>,
) {
    for event in interaction_events.read() {
        if let Ok(player) = players.get(event.player_entity) {
            // Handle interactable interactions
            if let Ok(interactable) = interactables.get(event.target_entity) {
                match interactable.action.as_str() {
                    "skill_challenge" => {
                        challenge_events.send(ChallengeEvent {
                            challenge_id: interactable.id.clone(),
                            event_type: "start".to_string(),
                            player_entity: event.player_entity,
                        });
                    }
                    "nrn_reward" => {
                        reward_events.send(RewardEvent {
                            player_entity: event.player_entity,
                            rewards: interactable.rewards.clone(),
                        });
                    }
                    _ => {
                        info!("Unknown interaction action: {}", interactable.action);
                    }
                }
            }

            // Handle NPC interactions
            if let Ok(npc) = npcs.get(event.target_entity) {
                info!("Player {} interacting with NPC {}", player.name, npc.name);
                // Handle dialogue, quests, etc.
            }
        }
    }
}

fn update_challenges(
    mut challenge_events: EventReader<ChallengeEvent>,
    mut challenges: Query<&mut Challenge>,
    time: Res<Time>,
) {
    for event in challenge_events.read() {
        for mut challenge in challenges.iter_mut() {
            if challenge.id == event.challenge_id {
                match event.event_type.as_str() {
                    "start" => {
                        challenge.is_active = true;
                        info!("Challenge {} started", challenge.name);
                    }
                    "complete" => {
                        challenge.is_active = false;
                        info!("Challenge {} completed", challenge.name);
                    }
                    "fail" => {
                        challenge.is_active = false;
                        info!("Challenge {} failed", challenge.name);
                    }
                    _ => {}
                }
            }
        }
    }
}

fn process_rewards(
    mut reward_events: EventReader<RewardEvent>,
    mut players: Query<&mut Player>,
) {
    for event in reward_events.read() {
        if let Ok(mut player) = players.get_mut(event.player_entity) {
            for reward in &event.rewards {
                match reward.reward_type.as_str() {
                    "nrn" => {
                        player.nrn_balance += reward.amount;
                        info!("Player {} received {} NRN", player.name, reward.amount);
                    }
                    "experience" => {
                        player.experience += reward.amount as u64;
                        info!("Player {} gained {} experience", player.name, reward.amount);
                    }
                    "skill" => {
                        if let Some(skill) = &reward.item {
                            if !player.skills.contains(skill) {
                                player.skills.push(skill.clone());
                                info!("Player {} learned skill: {}", player.name, skill);
                            }
                        }
                    }
                    _ => {
                        info!("Unknown reward type: {}", reward.reward_type);
                    }
                }
            }
        }
    }
}
