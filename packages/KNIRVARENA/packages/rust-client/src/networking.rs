use bevy::prelude::*;
use reqwest::Client;
use serde::{Deserialize, Serialize};
use tokio::runtime::Runtime;
use chrono::Utc;
use crate::game_engine::*;
use crate::resources::*;

pub struct NetworkingPlugin;

impl Plugin for NetworkingPlugin {
    fn build(&self, app: &mut App) {
        app
            .add_systems(Startup, initialize_networking)
            .add_systems(Update, (
                handle_network_events,
                sync_player_data,
                fetch_game_updates,
            ));
    }
}

#[derive(Serialize, Deserialize)]
pub struct NetworkMessage {
    pub message_type: String,
    pub payload: serde_json::Value,
    pub timestamp: u64,
}

#[derive(Serialize, Deserialize)]
pub struct PlayerSyncData {
    pub player_id: String,
    pub position: [f32; 3],
    pub rotation: [f32; 4],
    pub health: f32,
    pub energy: f32,
    pub nrn_balance: f64,
    pub level: u32,
    pub experience: u64,
}

#[derive(Serialize, Deserialize)]
pub struct ChallengeData {
    pub id: String,
    pub name: String,
    pub description: String,
    pub challenge_type: String,
    pub difficulty: u32,
    pub rewards: Vec<Reward>,
    pub is_active: bool,
}

fn initialize_networking(
    mut network_manager: ResMut<NetworkManager>,
    config: Res<GameConfig>,
) {
    network_manager.api_endpoint = config.api_endpoint.clone();
    network_manager.connected = false;

    info!("Networking initialized with endpoint: {}", network_manager.api_endpoint);
}

fn handle_network_events(
    mut network_manager: ResMut<NetworkManager>,
    time: Res<Time>,
) {
    let current_time = time.elapsed_seconds_f64();

    // Simple connection check
    if current_time - network_manager.last_ping > 30.0 {
        network_manager.last_ping = current_time;
        // Ping server to maintain connection
        info!("Pinging server...");
    }
}

fn sync_player_data(
    player_query: Query<(&Player, &Transform), Changed<Player>>,
    mut network_manager: ResMut<NetworkManager>,
) {
    for (player, transform) in player_query.iter() {
        let sync_data = PlayerSyncData {
            player_id: player.id.clone(),
            position: [
                transform.translation.x,
                transform.translation.y,
                transform.translation.z,
            ],
            rotation: [
                transform.rotation.x,
                transform.rotation.y,
                transform.rotation.z,
                transform.rotation.w,
            ],
            health: 100.0, // Would get from Health component
            energy: 100.0, // Would get from Energy component
            nrn_balance: player.nrn_balance,
            level: player.level,
            experience: player.experience,
        };

        // In a real implementation, this would send data to server
        info!("Syncing player data: {}", player.name);
    }
}

fn fetch_game_updates(
    mut commands: Commands,
    network_manager: Res<NetworkManager>,
    time: Res<Time>,
) {
    // Periodically fetch updates from server
    // This would include new challenges, world events, etc.

    // Placeholder for server communication
    if time.elapsed_seconds() as u32 % 60 == 0 {
        info!("Fetching game updates from server...");
    }
}

// Async networking functions (would be called from systems)
pub async fn authenticate_player(api_endpoint: &str, player_id: &str) -> Result<String, Box<dyn std::error::Error>> {
    let client = Client::new();
    let response = client
        .post(&format!("{}/auth/login", api_endpoint))
        .json(&serde_json::json!({
            "player_id": player_id,
            "timestamp": Utc::now().timestamp()
        }))
        .send()
        .await?;

    let session_token: String = response.json().await?;
    Ok(session_token)
}

pub async fn fetch_player_data(api_endpoint: &str, player_id: &str) -> Result<PlayerInfo, Box<dyn std::error::Error>> {
    let client = Client::new();
    let response = client
        .get(&format!("{}/players/{}", api_endpoint, player_id))
        .send()
        .await?;

    let player_data: PlayerInfo = response.json().await?;
    Ok(player_data)
}

pub async fn submit_challenge_result(
    api_endpoint: &str,
    challenge_id: &str,
    player_id: &str,
    success: bool,
    score: u32,
) -> Result<Vec<Reward>, Box<dyn std::error::Error>> {
    let client = Client::new();
    let response = client
        .post(&format!("{}/challenges/{}/complete", api_endpoint, challenge_id))
        .json(&serde_json::json!({
            "player_id": player_id,
            "success": success,
            "score": score,
            "timestamp": Utc::now().timestamp()
        }))
        .send()
        .await?;

    let rewards: Vec<Reward> = response.json().await?;
    Ok(rewards)
}
