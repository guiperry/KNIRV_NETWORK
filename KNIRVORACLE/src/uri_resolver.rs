use serde::{Deserialize, Serialize};
use std::collections::HashMap;
use std::sync::{Arc, Mutex};
use tracing::{info, warn};

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct NodeInfo {
    pub peer_id: String,
    pub chain_id: String,
    pub public_ip: Option<String>,
    pub p2p_port: u16,
    pub http_port: u16,
    pub websocket_port: Option<u16>,
    pub capabilities: Vec<String>,
    pub registered_at: u64,
    pub last_seen: u64,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TunnelInfo {
    pub node_id: String,
    pub tunnel_id: String,
    pub control_port: u16,
    pub public_relay_port: u16,
    pub stun_port: u16,
    pub server_public_host: String,
}

#[derive(Debug)]
pub struct UriResolver {
    nodes: Arc<Mutex<HashMap<String, NodeInfo>>>,
    tunnels: Arc<Mutex<HashMap<String, TunnelInfo>>>,
}

impl UriResolver {
    pub fn new() -> Self {
        Self {
            nodes: Arc::new(Mutex::new(HashMap::new())),
            tunnels: Arc::new(Mutex::new(HashMap::new())),
        }
    }

    pub fn register_node(&self, node_info: NodeInfo) -> Result<(), Box<dyn std::error::Error>> {
        let mut nodes = self.nodes.lock().map_err(|_| "Failed to lock nodes")?;
        let key = format!("{}_{}", node_info.peer_id, node_info.chain_id);
        nodes.insert(key, node_info.clone());
        info!("Registered node: {} on chain {}", node_info.peer_id, node_info.chain_id);
        Ok(())
    }

    pub fn resolve_uri(&self, uri: &str) -> Result<NodeInfo, Box<dyn std::error::Error>> {
        // Parse knirv://peer_id@chain_id format
        if !uri.starts_with("knirv://") {
            return Err("Invalid URI format".into());
        }

        let parts: Vec<&str> = uri[8..].split('@').collect();
        if parts.len() != 2 {
            return Err("Invalid URI format".into());
        }

        let peer_id = parts[0];
        let chain_id = parts[1];
        let key = format!("{}_{}", peer_id, chain_id);

        let nodes = self.nodes.lock().map_err(|_| "Failed to lock nodes")?;
        if let Some(node) = nodes.get(&key) {
            Ok(node.clone())
        } else {
            Err(format!("Node not found: {}", key).into())
        }
    }

    pub fn generate_uri(&self, peer_id: &str, chain_id: &str) -> String {
        format!("knirv://{}@{}", peer_id, chain_id)
    }

    pub fn list_nodes(&self) -> Result<Vec<NodeInfo>, Box<dyn std::error::Error>> {
        let nodes = self.nodes.lock().map_err(|_| "Failed to lock nodes")?;
        Ok(nodes.values().cloned().collect())
    }

    pub fn get_node_by_peer_id(&self, peer_id: &str) -> Result<NodeInfo, Box<dyn std::error::Error>> {
        let nodes = self.nodes.lock().map_err(|_| "Failed to lock nodes")?;
        for node in nodes.values() {
            if node.peer_id == peer_id {
                return Ok(node.clone());
            }
        }
        Err(format!("Node not found: {}", peer_id).into())
    }

    pub fn get_node_by_chain_id(&self, chain_id: &str) -> Result<NodeInfo, Box<dyn std::error::Error>> {
        let nodes = self.nodes.lock().map_err(|_| "Failed to lock nodes")?;
        for node in nodes.values() {
            if node.chain_id == chain_id {
                return Ok(node.clone());
            }
        }
        Err(format!("Node not found for chain: {}", chain_id).into())
    }

    pub fn register_tunnel(&self, tunnel_info: TunnelInfo) -> Result<(), Box<dyn std::error::Error>> {
        let mut tunnels = self.tunnels.lock().map_err(|_| "Failed to lock tunnels")?;
        tunnels.insert(tunnel_info.node_id.clone(), tunnel_info.clone());
        info!("Registered tunnel for node: {}", tunnel_info.node_id);
        Ok(())
    }

    pub fn get_tunnel(&self, node_id: &str) -> Result<TunnelInfo, Box<dyn std::error::Error>> {
        let tunnels = self.tunnels.lock().map_err(|_| "Failed to lock tunnels")?;
        if let Some(tunnel) = tunnels.get(node_id) {
            Ok(tunnel.clone())
        } else {
            Err(format!("Tunnel not found for node: {}", node_id).into())
        }
    }

    pub fn update_node_last_seen(&self, peer_id: &str, chain_id: &str) -> Result<(), Box<dyn std::error::Error>> {
        let mut nodes = self.nodes.lock().map_err(|_| "Failed to lock nodes")?;
        let key = format!("{}_{}", peer_id, chain_id);
        if let Some(node) = nodes.get_mut(&key) {
            node.last_seen = std::time::SystemTime::now()
                .duration_since(std::time::UNIX_EPOCH)
                .unwrap()
                .as_secs();
        }
        Ok(())
    }

    pub fn cleanup_stale_nodes(&self, max_age_seconds: u64) -> Result<usize, Box<dyn std::error::Error>> {
        let mut nodes = self.nodes.lock().map_err(|_| "Failed to lock nodes")?;
        let now = std::time::SystemTime::now()
            .duration_since(std::time::UNIX_EPOCH)
            .unwrap()
            .as_secs();

        let initial_count = nodes.len();
        nodes.retain(|_, node| now - node.last_seen < max_age_seconds);

        let removed_count = initial_count - nodes.len();
        if removed_count > 0 {
            info!("Cleaned up {} stale nodes", removed_count);
        }

        Ok(removed_count)
    }
}