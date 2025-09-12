use bevy::prelude::*;
use serde::{Deserialize, Serialize};
use std::collections::HashMap;
use crate::components::{ErrorNode, SkillNode, AIAgent, ErrorNodeType, SkillCategory, AgentType};

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

// ============================================================================
// KNIRVANA-SPECIFIC GAME STATE RESOURCES
// ============================================================================

/// Main game state resource for KNIRVANA
#[derive(Resource)]
pub struct KnirvanaGameState {
    pub game_phase: GamePhase,
    pub game_time: f64,
    pub selected_error_node: Option<Entity>,
    pub selected_agent: Option<Entity>,
    pub selected_skill_node: Option<Entity>,
    pub camera_target: Option<Entity>,
    pub is_paused: bool,
    pub last_update_time: f64,
}

impl Default for KnirvanaGameState {
    fn default() -> Self {
        Self {
            game_phase: GamePhase::Playing, // Start directly in playing mode
            game_time: 0.0,
            selected_error_node: None,
            selected_agent: None,
            selected_skill_node: None,
            camera_target: None,
            is_paused: false,
            last_update_time: 0.0,
        }
    }
}

/// Game phases for KNIRVANA
#[derive(Serialize, Deserialize, Clone, Debug, PartialEq)]
pub enum GamePhase {
    Menu,
    Loading,
    Playing,
    Paused,
    GameOver,
    Victory,
}

/// Resource for tracking player's NRN balance and game statistics
#[derive(Resource)]
pub struct PlayerResources {
    pub nrn_balance: f64,
    pub skills_learned: u32,
    pub errors_resolved: u32,
    pub agents_deployed: u32,
    pub total_bounty_earned: f64,
    pub session_start_time: f64,
    pub best_solve_time: f32,
    pub current_streak: u32,
}

impl Default for PlayerResources {
    fn default() -> Self {
        Self {
            nrn_balance: 500.0,  // Starting balance
            skills_learned: 0,
            errors_resolved: 0,
            agents_deployed: 0,
            total_bounty_earned: 0.0,
            session_start_time: 0.0,
            best_solve_time: f32::MAX,
            current_streak: 0,
        }
    }
}

/// Resource for managing the KNIRV-GRAPH state
#[derive(Resource, Default)]
pub struct KnirvGraphState {
    pub error_nodes: HashMap<String, Entity>,
    pub skill_nodes: HashMap<String, Entity>,
    pub idea_nodes: HashMap<String, Entity>,
    pub property_nodes: HashMap<String, Entity>,
    pub agents: HashMap<String, Entity>,
    pub connections: Vec<(Entity, Entity)>,
    pub graph_center: Vec3,
    pub graph_radius: f32,
    pub node_spawn_timer: f32,
    pub max_error_nodes: u32,
    pub max_skill_nodes: u32,
    pub max_idea_nodes: u32,
    pub max_property_nodes: u32,
}

/// Resource for competitive multiplayer state
#[derive(Resource, Default)]
pub struct CompetitiveState {
    pub other_players: HashMap<String, PlayerInfo>,
    pub leaderboard: Vec<LeaderboardEntry>,
    pub current_competitions: Vec<CompetitionInfo>,
    pub last_leaderboard_update: f64,
}

#[derive(Serialize, Deserialize, Clone)]
pub struct LeaderboardEntry {
    pub player_id: String,
    pub player_name: String,
    pub score: u32,
    pub errors_resolved: u32,
    pub nrn_earned: f64,
}

#[derive(Serialize, Deserialize, Clone)]
pub struct CompetitionInfo {
    pub error_node_id: String,
    pub participants: Vec<String>,
    pub progress: HashMap<String, f32>,
    pub started_at: f64,
    pub estimated_completion: f64,
}

/// Resource for managing agent pretraining and deployment
#[derive(Resource, Default)]
pub struct AgentManager {
    pub available_agents: Vec<AgentTemplate>,
    pub deployed_agents: HashMap<String, Entity>,
    pub pretraining_queue: Vec<PretrainingTask>,
    pub agent_costs: HashMap<AgentType, f64>,
    pub deployment_cooldown: f32,
}

#[derive(Serialize, Deserialize, Clone)]
pub struct AgentTemplate {
    pub id: String,
    pub name: String,
    pub agent_type: AgentType,
    pub specialization: SkillCategory,
    pub base_efficiency: f32,
    pub energy_capacity: f32,
    pub training_cost: f64,
    pub deployment_cost: f64,
}

#[derive(Serialize, Deserialize, Clone)]
pub struct PretrainingTask {
    pub agent_id: String,
    pub skill_category: SkillCategory,
    pub progress: f32,
    pub estimated_completion: f64,
    pub nrn_cost: f64,
}

/// Resource for visual effects and animations
#[derive(Resource, Default)]
pub struct VisualEffectsState {
    pub tron_glow_intensity: f32,
    pub connection_animation_speed: f32,
    pub particle_density: f32,
    pub screen_effects_enabled: bool,
    pub bloom_intensity: f32,
    pub ambient_pulse_speed: f32,
}

/// Resource for UI state management
#[derive(Resource)]
pub struct UIState {
    pub show_agent_panel: bool,
    pub show_node_info: bool,
    pub show_leaderboard: bool,
    pub show_settings: bool,
    pub hud_opacity: f32,
    pub selected_ui_tab: UITab,
    pub notification_queue: Vec<Notification>,
}

impl Default for UIState {
    fn default() -> Self {
        Self {
            show_agent_panel: true,  // Show agent panel by default
            show_node_info: true,    // Show node info by default
            show_leaderboard: false,
            show_settings: false,
            hud_opacity: 1.0,
            selected_ui_tab: UITab::Agents,
            notification_queue: vec![
                Notification {
                    message: "Welcome to KNIRVANA! Click on error nodes to deploy agents.".to_string(),
                    notification_type: NotificationType::Info,
                    timestamp: 0.0,
                    duration: 10.0,
                }
            ],
        }
    }
}

#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, Default)]
pub enum UITab {
    #[default]
    Agents,
    Skills,
    Leaderboard,
    Settings,
}

#[derive(Serialize, Deserialize, Clone)]
pub struct Notification {
    pub message: String,
    pub notification_type: NotificationType,
    pub timestamp: f64,
    pub duration: f32,
}

#[derive(Serialize, Deserialize, Clone, Debug)]
pub enum NotificationType {
    Success,
    Warning,
    Error,
    Info,
    Achievement,
}
