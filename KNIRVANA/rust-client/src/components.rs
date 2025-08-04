use bevy::prelude::*;
use bevy_rapier3d::prelude::*;
use serde::{Deserialize, Serialize};

#[derive(Component)]
pub struct MainCamera;

#[derive(Component)]
pub struct PlayerAvatar;

#[derive(Component)]
pub struct Environment {
    pub terrain_type: String,
    pub weather: WeatherType,
    pub time_of_day: f32,
}

#[derive(Component)]
pub struct WeatherSystem {
    pub weather_type: WeatherType,
    pub intensity: f32,
    pub particle_count: u32,
}

#[derive(Serialize, Deserialize, Clone)]
pub enum WeatherType {
    Clear,
    Rain,
    Snow,
    Fog,
    Storm,
}

#[derive(Component)]
pub struct Collectible {
    pub item_type: String,
    pub value: f64,
    pub rarity: String,
}

#[derive(Component)]
pub struct QuestGiver {
    pub available_quests: Vec<String>,
    pub completed_quests: Vec<String>,
}

#[derive(Component)]
pub struct Inventory {
    pub items: Vec<InventoryItem>,
    pub capacity: u32,
}

#[derive(Serialize, Deserialize, Clone)]
pub struct InventoryItem {
    pub id: String,
    pub name: String,
    pub quantity: u32,
    pub item_type: String,
}

#[derive(Component)]
pub struct Health {
    pub current: f32,
    pub maximum: f32,
    pub regeneration_rate: f32,
}

#[derive(Component)]
pub struct Energy {
    pub current: f32,
    pub maximum: f32,
    pub consumption_rate: f32,
}

#[derive(Component)]
pub struct NetworkSync {
    pub last_sync: f64,
    pub sync_interval: f64,
    pub dirty: bool,
}

#[derive(Component)]
pub struct MobileOptimized {
    pub lod_level: u32,
    pub simplified_physics: bool,
    pub reduced_particles: bool,
}
