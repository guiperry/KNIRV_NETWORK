use bevy::prelude::*;
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

// ============================================================================
// KNIRVANA-SPECIFIC GAME COMPONENTS
// ============================================================================

/// ErrorNode component representing problems to be solved in the KNIRV-GRAPH
#[derive(Component, Serialize, Deserialize, Clone)]
pub struct ErrorNode {
    pub id: String,
    pub node_type: ErrorNodeType,
    pub difficulty: f32,           // 0.0 to 1.0
    pub bounty: f64,              // NRN reward for solving
    pub progress: f32,            // 0.0 to 1.0 completion
    pub is_being_solved: bool,
    pub solver_agent_id: Option<String>,
    pub description: String,
    pub created_at: f64,
    pub estimated_time: f32,      // Estimated time to solve in seconds
}

/// Types of ErrorNodes that can appear in the game
#[derive(Serialize, Deserialize, Clone, Debug)]
pub enum ErrorNodeType {
    LogicError,
    DataInconsistency,
    NetworkFailure,
    SecurityVulnerability,
    PerformanceBottleneck,
    IntegrationIssue,
}

/// SkillNode component representing learned capabilities in the KNIRV-GRAPH
#[derive(Component, Serialize, Deserialize, Clone)]
pub struct SkillNode {
    pub id: String,
    pub name: String,
    pub category: SkillCategory,
    pub created_by: String,       // Player ID who created this skill
    pub usage_count: u32,
    pub value: f64,               // NRN value of this skill
    pub created_at: f64,
    pub effectiveness: f32,       // 0.0 to 1.0 how effective this skill is
}

/// Categories of skills that can be learned
#[derive(Serialize, Deserialize, Clone, Debug)]
pub enum SkillCategory {
    Debugging,
    Optimization,
    Security,
    Integration,
    Analysis,
    Automation,
}

/// IdeaNode component representing ideas that become properties through collaboration
#[derive(Component, Serialize, Deserialize, Clone)]
pub struct IdeaNode {
    pub id: String,
    pub name: String,
    pub idea_type: IdeaType,
    pub description: String,
    pub feasibility_score: f32,      // 0.0 to 1.0 feasibility rating
    pub existence_check: bool,       // Whether this idea already exists
    pub collaborators: Vec<String>,  // Agent IDs collaborating on this idea
    pub stakes: std::collections::HashMap<String, f32>, // Agent stakes in resulting property
    pub collaboration_value: f64,    // Total NRN value from collaboration
    pub created_at: f64,
    pub status: IdeaStatus,
}

/// Types of ideas that can be developed
#[derive(Serialize, Deserialize, Clone, Debug)]
pub enum IdeaType {
    Asset,
    Characteristic,
    Attribute,
    Innovation,
    Improvement,
    Feature,
}

/// Status of idea development
#[derive(Serialize, Deserialize, Clone, Debug)]
pub enum IdeaStatus {
    Pending,
    Collaborative,
    PropertyCreated,
    Abandoned,
}

/// PropertyNode component representing properties created from ideas
#[derive(Component, Serialize, Deserialize, Clone)]
pub struct PropertyNode {
    pub id: String,
    pub name: String,
    pub property_type: PropertyType,
    pub source_idea: String,         // IdeaNode ID that created this property
    pub value_type: String,          // "string", "number", "boolean", "object"
    pub immutable: bool,
    pub category: String,
    pub owners: std::collections::HashMap<String, f32>, // Agent ownership stakes
    pub market_value: f64,           // Current NRN market value
    pub usage_count: u32,
    pub created_at: f64,
}

/// Types of properties that can be owned
#[derive(Serialize, Deserialize, Clone, Debug)]
pub enum PropertyType {
    Asset,
    Characteristic,
    Attribute,
    License,
    Patent,
    Trademark,
}

/// AIAgent component representing player-controlled agent units
#[derive(Component, Serialize, Deserialize, Clone)]
pub struct AIAgent {
    pub id: String,
    pub owner_id: String,         // Player who owns this agent
    pub agent_type: AgentType,
    pub specialization: SkillCategory,
    pub efficiency: f32,          // 0.0 to 1.0 how fast this agent works
    pub energy: f32,              // Current energy level
    pub max_energy: f32,          // Maximum energy capacity
    pub experience: u32,          // Experience points gained
    pub skills: Vec<String>,      // List of skill IDs this agent knows
    pub current_task: Option<String>, // ErrorNode ID currently working on
    pub status: AgentStatus,
    pub thought_process: Vec<String>, // Real-time thoughts for display
    pub last_action_time: f64,
}

/// Types of AI agents with different capabilities
#[derive(Serialize, Deserialize, Clone, Debug)]
pub enum AgentType {
    Resolver,    // Specialized in solving ErrorNodes
    Observer,    // Gathers information and provides insights
    Helper,      // Assists other agents
    Specialist,  // Expert in specific skill categories
}

/// Current status of an AI agent
#[derive(Serialize, Deserialize, Clone, Debug, PartialEq)]
pub enum AgentStatus {
    Idle,
    Moving,
    Working,
    Upgrading,
    Resting,     // Recovering energy
    Thinking,    // Processing a problem
}

/// Component for visual effects and animations
#[derive(Component)]
pub struct TronEffect {
    pub glow_intensity: f32,
    pub pulse_speed: f32,
    pub color: Color,
    pub is_pulsing: bool,
    pub animation_time: f32,
}

/// Component for connection lines between nodes in the KNIRV-GRAPH
#[derive(Component)]
pub struct GraphConnection {
    pub from_node: Entity,
    pub to_node: Entity,
    pub connection_type: ConnectionType,
    pub data_flow_speed: f32,
    pub is_active: bool,
}

/// Types of connections in the KNIRV-GRAPH
#[derive(Serialize, Deserialize, Clone, Debug)]
pub enum ConnectionType {
    DataFlow,
    SkillDependency,
    ErrorPropagation,
    AgentPath,
}

/// Component for selectable game objects
#[derive(Component)]
pub struct Selectable {
    pub is_selected: bool,
    pub is_hovered: bool,
    pub selection_radius: f32,
}

/// Component for objects that can be interacted with
#[derive(Component)]
pub struct Interactable {
    pub interaction_type: InteractionType,
    pub interaction_range: f32,
    pub is_available: bool,
    pub cooldown_remaining: f32,
}

/// Types of interactions available in the game
#[derive(Serialize, Deserialize, Clone, Debug)]
pub enum InteractionType {
    DeployAgent,
    ViewProgress,
    UpgradeAgent,
    InvokeSkill,
    InspectNode,
}

/// Component for progress tracking and visualization
#[derive(Component)]
pub struct ProgressIndicator {
    pub current_progress: f32,    // 0.0 to 1.0
    pub target_progress: f32,
    pub animation_speed: f32,
    pub show_percentage: bool,
}

/// Component for floating text and UI elements in 3D space
#[derive(Component)]
pub struct FloatingUI {
    pub text: String,
    pub font_size: f32,
    pub color: Color,
    pub lifetime: f32,
    pub fade_speed: f32,
    pub billboard: bool,          // Always face camera
}
