use anyhow::{anyhow, Result};
use cid::Cid;
use futures::StreamExt;
use libp2p::{
    gossipsub::{self, IdentTopic, MessageAuthenticity, ValidationMode},
    identity::Keypair,
    kad::{record::Key, Behaviour as KademliaBehaviour, Event as KademliaEvent},
    swarm::{SwarmEvent, NetworkBehaviour, Config as SwarmConfig},
    Multiaddr, PeerId, Swarm, Transport,
};
use multihash::{Code, MultihashDigest};
use serde::{Deserialize, Serialize};
use std::{
    collections::HashMap,
    sync::{Arc, Mutex},
    time::{Duration, SystemTime, UNIX_EPOCH},
};
use tokio::{
    sync::mpsc,
    time::interval,
};
use tracing::{error, info, warn};

// Network control constants
const NETWORK_CONTROL_TOPIC: &str = "network-control";
const SKILL_CONFIRMATION_TOPIC: &str = "skill-confirmations";
const NETWORK_PAUSE_TIMEOUT: Duration = Duration::from_secs(30 * 60); // 30 minutes

// DHT manager for KNIRVORACLE skill confirmation
pub struct DHTManager {
    swarm: Swarm<ChainBehaviour>,
    service_id: String,
    chain_id: String,
    network_paused: Arc<Mutex<bool>>,
    paused_until: Arc<Mutex<Option<SystemTime>>>,
    command_sender: mpsc::UnboundedSender<DHTCommand>,
    command_receiver: mpsc::UnboundedReceiver<DHTCommand>,
}

// Network control message types
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct NetworkControlMessage {
    pub message_type: String,
    pub payload: serde_json::Value,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct NetworkPausePayload {
    pub initiator_peer_id: String,
    pub reason: String,
    pub timestamp: u64,
}

// Skill confirmation message types
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SkillConfirmationMessage {
    pub message_type: String, // "skill_confirmed", "skill_invokable", "skill_error"
    pub action: String,       // "confirmed", "available", "failed"
    pub service_id: String,
    pub chain_id: String,
    pub data: serde_json::Value,
    pub timestamp: u64,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SkillConfirmationData {
    pub skill_id: String,
    pub skill_name: String,
    pub skill_hash: String,
    pub confirmation_status: String,
    pub validator_peer_id: String,
    pub metadata: HashMap<String, String>,
}

// Commands for DHT operations
#[derive(Debug)]
pub enum DHTCommand {
    ConfirmSkill {
        skill_id: String,
        skill_name: String,
        skill_hash: String,
        metadata: HashMap<String, String>,
    },
    AnnounceSkillInvokable {
        skill_id: String,
        skill_name: String,
        endpoint: String,
        metadata: HashMap<String, String>,
    },
    ReportSkillError {
        skill_id: String,
        error_message: String,
        metadata: HashMap<String, String>,
    },
    DiscoverServices {
        service_id: String,
    },
    GetServiceRecord {
        service_id: String,
    },
}

// Combined behaviour for Kademlia and GossipSub
#[derive(NetworkBehaviour)]
#[behaviour(to_swarm = "ChainBehaviourEvent")]
pub struct ChainBehaviour {
    pub kademlia: KademliaBehaviour<libp2p::kad::store::MemoryStore>,
    pub gossipsub: gossipsub::Behaviour,
}

#[derive(Debug)]
pub enum ChainBehaviourEvent {
    Kademlia(KademliaEvent),
    Gossipsub(gossipsub::Event),
}

impl From<KademliaEvent> for ChainBehaviourEvent {
    fn from(event: KademliaEvent) -> Self {
        ChainBehaviourEvent::Kademlia(event)
    }
}

impl From<gossipsub::Event> for ChainBehaviourEvent {
    fn from(event: gossipsub::Event) -> Self {
        ChainBehaviourEvent::Gossipsub(event)
    }
}

impl DHTManager {
    /// Create a new DHT manager for KNIRVORACLE
    pub async fn new(
        service_id: String,
        chain_id: String,
        bootstrap_peers: Vec<Multiaddr>,
    ) -> Result<Self> {
        // Generate a new keypair
        let local_key = Keypair::generate_ed25519();
        let local_peer_id = PeerId::from(local_key.public());

        info!("KNIRVORACLE DHT Manager starting with PeerID: {}", local_peer_id);

        // Create Kademlia behaviour
        let store = libp2p::kad::store::MemoryStore::new(local_peer_id);
        let mut kademlia = KademliaBehaviour::new(local_peer_id, store);

        // Add bootstrap peers to Kademlia
        for addr in &bootstrap_peers {
            if let Some(peer_id) = addr.iter().find_map(|p| {
                if let libp2p::multiaddr::Protocol::P2p(hash) = p {
                    PeerId::from_multihash(hash.into()).ok()
                } else {
                    None
                }
            }) {
                kademlia.add_address(&peer_id, addr.clone());
            }
        }

        // Create GossipSub behaviour
        let gossipsub_config = gossipsub::ConfigBuilder::default()
            .heartbeat_interval(Duration::from_secs(10))
            .validation_mode(ValidationMode::Strict)
            .build()
            .map_err(|e| anyhow!("Failed to create gossipsub config: {}", e))?;

        let mut gossipsub = gossipsub::Behaviour::new(
            MessageAuthenticity::Signed(local_key.clone()),
            gossipsub_config,
        )
        .map_err(|e| anyhow!("Failed to create gossipsub behaviour: {}", e))?;

        // Subscribe to topics
        let network_control_topic = IdentTopic::new(NETWORK_CONTROL_TOPIC);
        let skill_confirmation_topic = IdentTopic::new(SKILL_CONFIRMATION_TOPIC);

        gossipsub.subscribe(&network_control_topic)?;
        gossipsub.subscribe(&skill_confirmation_topic)?;

        // Create combined behaviour
        let behaviour = ChainBehaviour {
            kademlia,
            gossipsub,
        };

        // Create swarm with default transport
        let swarm = Swarm::new(
            libp2p::tcp::tokio::Transport::default()
                .upgrade(libp2p::core::upgrade::Version::V1)
                .authenticate(libp2p::noise::Config::new(&local_key)?)
                .multiplex(libp2p::yamux::Config::default())
                .boxed(),
            behaviour,
            local_peer_id,
            SwarmConfig::with_tokio_executor(),
        );

        // Create command channel
        let (command_sender, command_receiver) = mpsc::unbounded_channel();

        Ok(DHTManager {
            swarm,
            service_id,
            chain_id,
            network_paused: Arc::new(Mutex::new(false)),
            paused_until: Arc::new(Mutex::new(None)),
            command_sender,
            command_receiver,
        })
    }

    /// Start the DHT manager
    pub async fn start(&mut self, listen_addr: Multiaddr) -> Result<()> {
        // Start listening
        self.swarm.listen_on(listen_addr)?;

        info!("KNIRVORACLE DHT Manager started successfully");

        // Bootstrap the DHT
        if let Err(e) = self.swarm.behaviour_mut().kademlia.bootstrap() {
            warn!("Failed to bootstrap DHT: {}", e);
        }

        // Start the main event loop
        self.run().await
    }

    /// Main event loop
    async fn run(&mut self) -> Result<()> {
        let mut heartbeat_interval = interval(Duration::from_secs(30));

        loop {
            tokio::select! {
                // Handle swarm events
                event = self.swarm.next() => {
                    if let Some(event) = event {
                        self.handle_swarm_event(event).await;
                    }
                }

                // Handle commands
                Some(command) = self.command_receiver.recv() => {
                    self.handle_command(command).await;
                }

                // Periodic heartbeat and maintenance
                _ = heartbeat_interval.tick() => {
                    self.periodic_maintenance().await;
                }
            }
        }
    }

    /// Handle swarm events
    async fn handle_swarm_event(&mut self, event: SwarmEvent<ChainBehaviourEvent, libp2p::swarm::derive_prelude::Either<std::io::Error, void::Void>>) {
        match event {
            SwarmEvent::Behaviour(ChainBehaviourEvent::Gossipsub(gossipsub::Event::Message {
                propagation_source: _,
                message_id: _,
                message,
            })) => {
                self.handle_gossipsub_message(message).await;
            }
            SwarmEvent::Behaviour(ChainBehaviourEvent::Kademlia(kad_event)) => {
                self.handle_kademlia_event(kad_event).await;
            }
            SwarmEvent::NewListenAddr { address, .. } => {
                info!("KNIRVORACLE DHT listening on: {}", address);
            }
            SwarmEvent::ConnectionEstablished { peer_id, .. } => {
                info!("Connected to peer: {}", peer_id);
            }
            SwarmEvent::ConnectionClosed { peer_id, .. } => {
                info!("Disconnected from peer: {}", peer_id);
            }
            _ => {}
        }
    }

    /// Handle GossipSub messages
    async fn handle_gossipsub_message(&mut self, message: gossipsub::Message) {
        let topic = message.topic.as_str();

        match topic {
            NETWORK_CONTROL_TOPIC => {
                self.handle_network_control_message(&message.data).await;
            }
            SKILL_CONFIRMATION_TOPIC => {
                self.handle_skill_confirmation_message(&message.data).await;
            }
            _ => {
                warn!("Received message on unknown topic: {}", topic);
            }
        }
    }

    /// Handle network control messages
    async fn handle_network_control_message(&mut self, data: &[u8]) {
        match serde_json::from_slice::<NetworkControlMessage>(data) {
            Ok(msg) => {
                info!("Received network control message: {}", msg.message_type);

                match msg.message_type.as_str() {
                    "NetworkPause" => {
                        if let Ok(payload) = serde_json::from_value::<NetworkPausePayload>(msg.payload) {
                            self.handle_network_pause(payload).await;
                        }
                    }
                    "NetworkResume" => {
                        self.handle_network_resume().await;
                    }
                    _ => {
                        warn!("Unknown network control message type: {}", msg.message_type);
                    }
                }
            }
            Err(e) => {
                error!("Failed to decode network control message: {}", e);
            }
        }
    }

    /// Handle network pause
    async fn handle_network_pause(&mut self, payload: NetworkPausePayload) {
        let mut paused = self.network_paused.lock().unwrap();
        *paused = true;

        let mut paused_until = self.paused_until.lock().unwrap();
        *paused_until = Some(SystemTime::now() + NETWORK_PAUSE_TIMEOUT);

        info!(
            "Network PAUSED by {} - Reason: {} - Skill confirmations suspended",
            payload.initiator_peer_id, payload.reason
        );
    }

    /// Handle network resume
    async fn handle_network_resume(&mut self) {
        let mut paused = self.network_paused.lock().unwrap();
        *paused = false;

        let mut paused_until = self.paused_until.lock().unwrap();
        *paused_until = None;

        info!("Network RESUMED - Skill confirmations active");
    }

    /// Check if network is paused
    pub fn is_network_paused(&self) -> bool {
        let paused = self.network_paused.lock().unwrap();
        if *paused {
            let paused_until = self.paused_until.lock().unwrap();
            if let Some(until) = *paused_until {
                if SystemTime::now() > until {
                    // Pause has expired
                    drop(paused);
                    drop(paused_until);
                    let mut paused = self.network_paused.lock().unwrap();
                    *paused = false;
                    let mut paused_until = self.paused_until.lock().unwrap();
                    *paused_until = None;
                    info!("Network pause expired");
                    return false;
                }
            }
            true
        } else {
            false
        }
    }

    /// Handle skill confirmation messages
    async fn handle_skill_confirmation_message(&mut self, data: &[u8]) {
        match serde_json::from_slice::<SkillConfirmationMessage>(data) {
            Ok(msg) => {
                info!(
                    "Received skill confirmation: {} {} from {}",
                    msg.action, msg.message_type, msg.service_id
                );

                // Process the skill confirmation based on type
                match msg.message_type.as_str() {
                    "skill_confirmed" => {
                        self.process_skill_confirmed(msg).await;
                    }
                    "skill_invokable" => {
                        self.process_skill_invokable(msg).await;
                    }
                    "skill_error" => {
                        self.process_skill_error(msg).await;
                    }
                    _ => {
                        warn!("Unknown skill confirmation type: {}", msg.message_type);
                    }
                }
            }
            Err(e) => {
                error!("Failed to decode skill confirmation message: {}", e);
            }
        }
    }

    /// Process skill confirmed message
    async fn process_skill_confirmed(&mut self, msg: SkillConfirmationMessage) {
        info!("Processing skill confirmation from {}", msg.service_id);
        // TODO: Integrate with KNIRVORACLE skill validation logic
    }

    /// Process skill invokable message
    async fn process_skill_invokable(&mut self, msg: SkillConfirmationMessage) {
        info!("Processing skill invokable announcement from {}", msg.service_id);
        // TODO: Integrate with KNIRVORACLE skill registry
    }

    /// Process skill error message
    async fn process_skill_error(&mut self, msg: SkillConfirmationMessage) {
        warn!("Processing skill error from {}: {:?}", msg.service_id, msg.data);
        // TODO: Integrate with KNIRVORACLE error handling
    }

    /// Handle Kademlia events
    async fn handle_kademlia_event(&mut self, event: KademliaEvent) {
        match event {
            KademliaEvent::OutboundQueryProgressed { result, .. } => {
                match result {
                    libp2p::kad::QueryResult::Bootstrap(Ok(_)) => {
                        info!("DHT bootstrap completed successfully");
                    }
                    libp2p::kad::QueryResult::Bootstrap(Err(e)) => {
                        warn!("DHT bootstrap failed: {}", e);
                    }
                    libp2p::kad::QueryResult::GetRecord(Ok(libp2p::kad::GetRecordOk::FoundRecord(_record))) => {
                        info!("Successfully retrieved DHT record");
                        // Record processing would be implemented here in a complete implementation
                    }
                    libp2p::kad::QueryResult::GetRecord(Ok(libp2p::kad::GetRecordOk::FinishedWithNoAdditionalRecord { .. })) => {
                        info!("DHT record lookup completed with no additional records");
                    }
                    libp2p::kad::QueryResult::GetRecord(Err(e)) => {
                        warn!("DHT record lookup failed: {}", e);
                    }
                    _ => {}
                }
            }
            _ => {}
        }
    }

    /// Handle commands
    async fn handle_command(&mut self, command: DHTCommand) {
        if self.is_network_paused() {
            warn!("Network is paused, ignoring command: {:?}", command);
            return;
        }

        match command {
            DHTCommand::ConfirmSkill {
                skill_id,
                skill_name,
                skill_hash,
                metadata,
            } => {
                self.confirm_skill(skill_id, skill_name, skill_hash, metadata)
                    .await;
            }
            DHTCommand::AnnounceSkillInvokable {
                skill_id,
                skill_name,
                endpoint,
                metadata,
            } => {
                self.announce_skill_invokable(skill_id, skill_name, endpoint, metadata)
                    .await;
            }
            DHTCommand::ReportSkillError {
                skill_id,
                error_message,
                metadata,
            } => {
                self.report_skill_error(skill_id, error_message, metadata)
                    .await;
            }
            DHTCommand::DiscoverServices { service_id } => {
                let _ = self.discover_services(&service_id).await;
            }
            DHTCommand::GetServiceRecord { service_id } => {
                let _ = self.get_service_record(&service_id).await;
            }
        }
    }

    /// Confirm a skill has been validated and is ready for invocation
    async fn confirm_skill(
        &mut self,
        skill_id: String,
        skill_name: String,
        skill_hash: String,
        metadata: HashMap<String, String>,
    ) {
        let peer_id = *self.swarm.local_peer_id();

        let confirmation_data = SkillConfirmationData {
            skill_id: skill_id.clone(),
            skill_name,
            skill_hash,
            confirmation_status: "confirmed".to_string(),
            validator_peer_id: peer_id.to_string(),
            metadata,
        };

        let message = SkillConfirmationMessage {
            message_type: "skill_confirmed".to_string(),
            action: "confirmed".to_string(),
            service_id: self.service_id.clone(),
            chain_id: self.chain_id.clone(),
            data: serde_json::to_value(confirmation_data).unwrap_or_default(),
            timestamp: SystemTime::now()
                .duration_since(UNIX_EPOCH)
                .unwrap()
                .as_secs(),
        };

        if let Err(e) = self.publish_skill_confirmation(message).await {
            error!("Failed to confirm skill {}: {}", skill_id, e);
        } else {
            info!("Skill {} confirmed and announced", skill_id);
        }
    }

    /// Announce that a skill is invokable
    async fn announce_skill_invokable(
        &mut self,
        skill_id: String,
        skill_name: String,
        endpoint: String,
        metadata: HashMap<String, String>,
    ) {
        let mut invokable_data = metadata;
        invokable_data.insert("skill_id".to_string(), skill_id.clone());
        invokable_data.insert("skill_name".to_string(), skill_name);
        invokable_data.insert("endpoint".to_string(), endpoint);

        let message = SkillConfirmationMessage {
            message_type: "skill_invokable".to_string(),
            action: "available".to_string(),
            service_id: self.service_id.clone(),
            chain_id: self.chain_id.clone(),
            data: serde_json::to_value(invokable_data).unwrap_or_default(),
            timestamp: SystemTime::now()
                .duration_since(UNIX_EPOCH)
                .unwrap()
                .as_secs(),
        };

        if let Err(e) = self.publish_skill_confirmation(message).await {
            error!("Failed to announce skill invokable {}: {}", skill_id, e);
        } else {
            info!("Skill {} announced as invokable", skill_id);
        }
    }

    /// Report a skill error
    async fn report_skill_error(
        &mut self,
        skill_id: String,
        error_message: String,
        metadata: HashMap<String, String>,
    ) {
        let mut error_data = metadata;
        error_data.insert("skill_id".to_string(), skill_id.clone());
        error_data.insert("error_message".to_string(), error_message);

        let message = SkillConfirmationMessage {
            message_type: "skill_error".to_string(),
            action: "failed".to_string(),
            service_id: self.service_id.clone(),
            chain_id: self.chain_id.clone(),
            data: serde_json::to_value(error_data).unwrap_or_default(),
            timestamp: SystemTime::now()
                .duration_since(UNIX_EPOCH)
                .unwrap()
                .as_secs(),
        };

        if let Err(e) = self.publish_skill_confirmation(message).await {
            error!("Failed to report skill error {}: {}", skill_id, e);
        } else {
            warn!("Skill error {} reported", skill_id);
        }
    }

    /// Publish skill confirmation message
    async fn publish_skill_confirmation(&mut self, message: SkillConfirmationMessage) -> Result<()> {
        let topic = IdentTopic::new(SKILL_CONFIRMATION_TOPIC);
        let data = serde_json::to_vec(&message)?;

        self.swarm
            .behaviour_mut()
            .gossipsub
            .publish(topic, data)
            .map_err(|e| anyhow!("Failed to publish skill confirmation: {}", e))?;

        Ok(())
    }

    /// Periodic maintenance tasks
    async fn periodic_maintenance(&mut self) {
        // Check if network pause has expired
        if self.is_network_paused() {
            // The check itself handles expiration
        }

        // Announce service to DHT
        self.announce_service_to_dht().await;
    }

    /// Announce this service to the DHT
    async fn announce_service_to_dht(&mut self) {
        // Create a CID-based key for better content addressing
        let cid = match Self::create_cid_from_service_id(&self.service_id) {
            Ok(cid) => cid,
            Err(e) => {
                warn!("Failed to create CID for service ID {}: {}", self.service_id, e);
                // Fallback to simple key format
                let service_key = format!("knirvoracle-{}", self.chain_id);
                let key = Key::new(&service_key);
                return self.announce_with_key(key).await;
            }
        };
        
        let key = Key::from(cid.to_bytes());

        // Create a record for this service
        let _record_value = serde_json::json!({
            "service_id": self.service_id,
            "chain_id": self.chain_id,
            "peer_id": self.swarm.local_peer_id().to_string(),
            "timestamp": SystemTime::now().duration_since(UNIX_EPOCH).unwrap().as_secs(),
            "capabilities": ["skill_confirmation", "skill_validation"]
        });

        self.announce_with_key(key).await
    }

    /// Discover services in the DHT using CID-based lookup
    pub async fn discover_services(&mut self, service_id: &str) -> Result<()> {
        let cid = Self::create_cid_from_service_id(service_id)?;
        let key = Key::from(cid.to_bytes());
        
        info!("Starting DHT lookup for service with CID: {} (service_id: {})", cid, service_id);
        
        // Start a DHT lookup for the service - results will be handled in Kademlia events
        self.swarm.behaviour_mut().kademlia.get_record(key);
        
        Ok(())
    }

    /// Get a specific service record from the DHT using CID
    pub async fn get_service_record(&mut self, service_id: &str) -> Result<()> {
        let cid = Self::create_cid_from_service_id(service_id)?;
        let key = Key::from(cid.to_bytes());
        
        info!("Getting service record with CID: {} (service_id: {})", cid, service_id);
        
        // Start a DHT get operation - results will be handled in Kademlia events
        self.swarm.behaviour_mut().kademlia.get_record(key);
        
        Ok(())
    }

    /// Helper method to announce service with a given key
    async fn announce_with_key(&mut self, key: Key) {
        // Create a record for this service
        let record_value = serde_json::json!({
            "service_id": self.service_id,
            "chain_id": self.chain_id,
            "peer_id": self.swarm.local_peer_id().to_string(),
            "timestamp": SystemTime::now().duration_since(UNIX_EPOCH).unwrap().as_secs(),
            "capabilities": ["skill_confirmation", "skill_validation"]
        });

        if let Ok(value) = serde_json::to_vec(&record_value) {
            let record = libp2p::kad::Record::new(key, value);
            if let Err(e) = self.swarm.behaviour_mut().kademlia.put_record(record, libp2p::kad::Quorum::One) {
                warn!("Failed to announce service to DHT: {}", e);
            }
        }
    }

    /// Get command sender for external use
    pub fn command_sender(&self) -> mpsc::UnboundedSender<DHTCommand> {
        self.command_sender.clone()
    }

    /// Create a CID from service ID for DHT operations
    pub fn create_cid_from_service_id(service_id: &str) -> Result<Cid> {
        let hash = Code::Sha2_256.digest(service_id.as_bytes());
        Ok(Cid::new_v1(0x55, hash)) // 0x55 is raw codec
    }
}
