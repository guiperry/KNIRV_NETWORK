use bevy::prelude::*;
use serde::{Deserialize, Serialize};
use std::collections::HashMap;

#[derive(Resource, Default)]
pub struct PlayerData {
    pub current_player: Option<String>,
    pub players: HashMap<String, PlayerInfo>,
}

#[derive(Serialize, Deserialize, Clone)]
pub struct PlayerInfo {
    pub id: String,
    pub name: String,
    pub level: u32,
    pub experience: u64,
    pub nrn_balance: f64,
    pub skills: Vec<String>,
    pub achievements: Vec<String>,
}

#[derive(Resource, Default)]
pub struct NetworkManager {
    pub api_endpoint: String,
    pub connected: bool,
    pub last_ping: f64,
    pub player_session: Option<String>,
}

#[derive(Resource, Default)]
pub struct GameMetrics {
    pub frame_rate: f32,
    pub player_count: u32,
    pub active_challenges: u32,
    pub network_latency: f32,
}

#[derive(Resource, Default)]
pub struct MobileSettings {
    pub touch_sensitivity: f32,
    pub graphics_quality: GraphicsQuality,
    pub battery_optimization: bool,
    pub reduced_effects: bool,
}

#[derive(Default)]
pub enum GraphicsQuality {
    Low,
    #[default]
    Medium,
    High,
    Ultra,
}
