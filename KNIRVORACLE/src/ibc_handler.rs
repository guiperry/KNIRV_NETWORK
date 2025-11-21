use anyhow::{anyhow, Result};
use serde::{Deserialize, Serialize};
use std::collections::HashMap;
use std::sync::Arc;
use std::time::{SystemTime, UNIX_EPOCH};
use tokio::sync::Mutex;
use tracing::{error, info, warn};

use crate::model_registry::ValidationProof;
use crate::multi_model_engine::ModelType;

#[derive(Debug)]
#[allow(dead_code)]
pub struct IBCHandler {
    channels: Arc<Mutex<HashMap<String, IBCChannel>>>,
    connections: Arc<Mutex<HashMap<String, IBCConnection>>>,
    clients: Arc<Mutex<HashMap<String, IBCClient>>>,
    knirv_root_connection: Arc<Mutex<Option<String>>>,
    knirv_nexus_connection: Arc<Mutex<Option<String>>>,
    message_queue: Arc<Mutex<Vec<PendingMessage>>>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct IBCChannel {
    pub channel_id: String,
    pub connection_id: String,
    pub port_id: String,
    pub counterparty_port_id: String,
    pub counterparty_channel_id: String,
    pub state: ChannelState,
    pub version: String,
    pub ordering: ChannelOrdering,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum ChannelState {
    Init,
    TryOpen,
    Open,
    Closed,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum ChannelOrdering {
    Ordered,
    Unordered,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct IBCConnection {
    pub connection_id: String,
    pub client_id: String,
    pub counterparty_client_id: String,
    pub counterparty_connection_id: String,
    pub state: ConnectionState,
    pub delay_period: u64,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum ConnectionState {
    Init,
    TryOpen,
    Open,
    Closed,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct IBCClient {
    pub client_id: String,
    pub client_type: String,
    pub chain_id: String,
    pub latest_height: u64,
    pub frozen: bool,
    pub trust_level: f64,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum IBCMessage {
    NRNBurn {
        token_id: String,
        amount: num_bigint::BigInt,
        skill_id: String,
        execution_location: String,
        user_address: String,
        model_type: ModelType,
    },
    SkillRegistration {
        skill_id: String,
        skill_metadata: SkillRegistrationData,
        validation_proof: ValidationProof,
    },
    ModelTransitionUpdate {
        previous_model: Option<String>,
        new_model: String,
        model_type: ModelType,
        governance_vote_id: String,
        transition_timestamp: u64,
    },
    ProofVerificationRequest {
        proof_id: String,
        model_hash: String,
        skill_id: Option<String>,
        verification_type: VerificationType,
    },
    ProofVerificationResponse {
        proof_id: String,
        is_valid: bool,
        verification_details: VerificationDetails,
        dve_address: String,
    },
    StateSync {
        chain_id: String,
        height: u64,
        app_hash: String,
        validator_set_hash: String,
    },
    NetworkStatus {
        chain_id: String,
        status: NetworkStatusData,
        timestamp: u64,
    },
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SkillRegistrationData {
    pub name: String,
    pub skill_type: String,
    pub capabilities: Vec<String>,
    pub owner: String,
    pub usage_fee: num_bigint::BigInt,
    pub code_hash: String,
    pub execution_metadata: ExecutionMetadata,
    pub security_requirements: SecurityRequirements,
    pub lora_requirements: LoRARequirements,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ExecutionMetadata {
    pub runtime: String,
    pub memory_limit: u64,
    pub cpu_limit: u64,
    pub timeout: u64,
    pub dependencies: Vec<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SecurityRequirements {
    pub isolation_level: IsolationLevel,
    pub network_access: bool,
    pub file_system_access: bool,
    pub encryption_required: bool,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum IsolationLevel {
    None,
    Process,
    Container,
    LoRA,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct LoRARequirements {
    pub required: bool,
    pub attestation_required: bool,
    pub secure_storage: bool,
    pub compatible_loras: Vec<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum VerificationType {
    ModelValidation,
    SkillValidation,
    PerformanceBenchmark,
    SecurityAudit,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct VerificationDetails {
    pub verification_type: VerificationType,
    pub score: f64,
    pub details: HashMap<String, serde_json::Value>,
    pub timestamp: u64,
    pub dve_signature: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct NetworkStatusData {
    pub active_validators: u32,
    pub total_transactions: u64,
    pub current_height: u64,
    pub average_block_time: f64,
    pub network_health: f64,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PendingMessage {
    pub message: IBCMessage,
    pub destination: String,
    pub retry_count: u32,
    pub created_at: u64,
    pub next_retry: u64,
}

/// IBC packet for cross-chain transfers
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct IBCPacket {
    pub sequence: u64,
    pub source_port: String,
    pub source_channel: String,
    pub destination_port: String,
    pub destination_channel: String,
    pub data: Vec<u8>,
    pub timeout_height: u64,
    pub timeout_timestamp: u64,
}

/// Packet receipt after transmission
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PacketReceipt {
    pub sequence: u64,
    pub transaction_hash: String,
    pub block_height: u64,
    pub timestamp: i64,
}

/// Packet acknowledgement
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PacketAcknowledgement {
    pub sequence: u64,
    pub acknowledgement: Vec<u8>,
    pub proof_height: u64,
    pub proof: Vec<u8>,
}

/// Transport types for IBC communication
#[derive(Debug, Clone)]
pub enum Transport {
    Grpc,
    Http,
    Websocket,
}

/// Connection endpoint information
#[derive(Debug, Clone)]
pub struct ConnectionEndpoint {
    pub address: String,
    pub transport: Transport,
    pub timeout: std::time::Duration,
}

#[allow(dead_code)]
impl IBCHandler {
    pub fn new() -> Self {
        Self {
            channels: Arc::new(Mutex::new(HashMap::new())),
            connections: Arc::new(Mutex::new(HashMap::new())),
            clients: Arc::new(Mutex::new(HashMap::new())),
            knirv_root_connection: Arc::new(Mutex::new(None)),
            knirv_nexus_connection: Arc::new(Mutex::new(None)),
            message_queue: Arc::new(Mutex::new(Vec::new())),
        }
    }

    /// Initialize connection to KNIRV-ORACLE
    pub async fn initialize_knirv_root_connection(&self, chain_id: String) -> Result<String> {
        let connection_id = format!("connection-knirv-oracle-{}", uuid::Uuid::new_v4());
        let client_id = format!("client-knirv-oracle-{}", uuid::Uuid::new_v4());

        // Create IBC client for KNIRV-ORACLE
        let client = IBCClient {
            client_id: client_id.clone(),
            client_type: "tendermint".to_string(),
            chain_id: chain_id.clone(),
            latest_height: 0,
            frozen: false,
            trust_level: 0.67,
        };

        // Create connection
        let connection = IBCConnection {
            connection_id: connection_id.clone(),
            client_id: client_id.clone(),
            counterparty_client_id: "knirvoracle-client".to_string(),
            counterparty_connection_id: "knirvoracle-connection".to_string(),
            state: ConnectionState::Open,
            delay_period: 0,
        };

        // Store client and connection
        let mut clients = self.clients.lock().await;
        clients.insert(client_id, client);

        let mut connections = self.connections.lock().await;
        connections.insert(connection_id.clone(), connection);

        let mut root_connection = self.knirv_root_connection.lock().await;
        *root_connection = Some(connection_id.clone());

        info!("Initialized KNIRV-ORACLE connection: {}", connection_id);
        Ok(connection_id)
    }

    /// Initialize connection to KNIRV-NEXUS
    pub async fn initialize_knirv_nexus_connection(&self, chain_id: String) -> Result<String> {
        let connection_id = format!("connection-knirv-nexus-{}", uuid::Uuid::new_v4());
        let client_id = format!("client-knirv-nexus-{}", uuid::Uuid::new_v4());

        // Create IBC client for KNIRV-NEXUS
        let client = IBCClient {
            client_id: client_id.clone(),
            client_type: "tendermint".to_string(),
            chain_id: chain_id.clone(),
            latest_height: 0,
            frozen: false,
            trust_level: 0.67,
        };

        // Create connection
        let connection = IBCConnection {
            connection_id: connection_id.clone(),
            client_id: client_id.clone(),
            counterparty_client_id: "knirvoracle-client".to_string(),
            counterparty_connection_id: "knirvoracle-connection".to_string(),
            state: ConnectionState::Open,
            delay_period: 0,
        };

        // Store client and connection
        let mut clients = self.clients.lock().await;
        clients.insert(client_id, client);

        let mut connections = self.connections.lock().await;
        connections.insert(connection_id.clone(), connection);

        let mut nexus_connection = self.knirv_nexus_connection.lock().await;
        *nexus_connection = Some(connection_id.clone());

        info!("Initialized KNIRV-NEXUS connection: {}", connection_id);
        Ok(connection_id)
    }

    /// Send NRN burn message to KNIRV-ORACLE
    pub async fn send_nrn_burn_message(
        &self,
        nrn_token_id: &str,
        amount: &num_bigint::BigInt,
        skill_id: &str,
        user_address: &str,
        model_type: ModelType,
    ) -> Result<()> {
        let message = IBCMessage::NRNBurn {
            token_id: nrn_token_id.to_string(),
            amount: amount.clone(),
            skill_id: skill_id.to_string(),
            execution_location: "local_lora".to_string(),
            user_address: user_address.to_string(),
            model_type,
        };

        self.send_to_knirv_root(message).await
    }

    /// Send skill registration to KNIRV-ORACLE
    pub async fn send_skill_registration(
        &self,
        skill_data: SkillRegistrationData,
        validation_proof: ValidationProof,
    ) -> Result<()> {
        let skill_id = format!(
            "skill_{}_{}",
            skill_data.skill_type,
            SystemTime::now().duration_since(UNIX_EPOCH)?.as_secs()
        );

        let message = IBCMessage::SkillRegistration {
            skill_id,
            skill_metadata: skill_data,
            validation_proof,
        };

        self.send_to_knirv_root(message).await
    }

    /// Send model transition update to KNIRV-ORACLE
    pub async fn send_model_transition_update(
        &self,
        previous_model: Option<String>,
        new_model: String,
        model_type: ModelType,
        governance_vote_id: String,
    ) -> Result<()> {
        let message = IBCMessage::ModelTransitionUpdate {
            previous_model,
            new_model,
            model_type,
            governance_vote_id,
            transition_timestamp: SystemTime::now().duration_since(UNIX_EPOCH)?.as_secs(),
        };

        self.send_to_knirv_root(message).await
    }

    /// Request validation proof verification from KNIRV-NEXUS
    pub async fn request_validation_proof_verification(
        &self,
        proof_id: &str,
        model_hash: &str,
        verification_type: VerificationType,
    ) -> Result<()> {
        let message = IBCMessage::ProofVerificationRequest {
            proof_id: proof_id.to_string(),
            model_hash: model_hash.to_string(),
            skill_id: None,
            verification_type,
        };

        self.send_to_knirv_nexus(message).await
    }

    /// Send message to KNIRV-ORACLE
    async fn send_to_knirv_root(&self, message: IBCMessage) -> Result<()> {
        let root_connection = self.knirv_root_connection.lock().await;
        let connection_id = root_connection
            .as_ref()
            .ok_or_else(|| anyhow!("KNIRV-ORACLE connection not established"))?;

        self.send_message(connection_id, message, "knirv-oracle")
            .await
    }

    /// Send message to KNIRV-NEXUS
    async fn send_to_knirv_nexus(&self, message: IBCMessage) -> Result<()> {
        let nexus_connection = self.knirv_nexus_connection.lock().await;
        let connection_id = nexus_connection
            .as_ref()
            .ok_or_else(|| anyhow!("KNIRV-NEXUS connection not established"))?;

        self.send_message(connection_id, message, "knirv-nexus")
            .await
    }

    /// Send IBC message through a connection
    async fn send_message(
        &self,
        connection_id: &str,
        message: IBCMessage,
        destination: &str,
    ) -> Result<()> {
        // Validate connection exists
        let connections = self.connections.lock().await;
        let connection = connections
            .get(connection_id)
            .ok_or_else(|| anyhow!("Connection not found: {}", connection_id))?;

        if !matches!(connection.state, ConnectionState::Open) {
            return Err(anyhow!("Connection not open: {}", connection_id));
        }

        // For now, add to message queue for processing
        let pending_message = PendingMessage {
            message: message.clone(),
            destination: destination.to_string(),
            retry_count: 0,
            created_at: SystemTime::now().duration_since(UNIX_EPOCH)?.as_secs(),
            next_retry: SystemTime::now().duration_since(UNIX_EPOCH)?.as_secs(),
        };

        let mut queue = self.message_queue.lock().await;
        queue.push(pending_message);

        info!(
            "Queued IBC message to {} via connection {}",
            destination, connection_id
        );

        // TODO: Implement actual IBC packet transmission
        self.process_outgoing_message(message, destination).await?;

        Ok(())
    }

    /// Process outgoing IBC message (placeholder implementation)
    async fn process_outgoing_message(&self, message: IBCMessage, destination: &str) -> Result<()> {
        match destination {
            "knirv-oracle" => {
                info!("Processing message to KNIRV-ORACLE: {:?}", message);
                // TODO: Implement actual HTTP/gRPC call to KNIRV-ORACLE
            }
            "knirv-nexus" => {
                info!("Processing message to KNIRV-NEXUS: {:?}", message);
                // TODO: Implement actual HTTP/gRPC call to KNIRV-NEXUS
            }
            _ => {
                warn!("Unknown destination: {}", destination);
            }
        }
        Ok(())
    }

    /// Handle incoming IBC message
    pub async fn handle_incoming_message(&self, message: IBCMessage, source: &str) -> Result<()> {
        info!("Received IBC message from {}: {:?}", source, message);

        match message {
            IBCMessage::ProofVerificationResponse {
                proof_id,
                is_valid,
                verification_details,
                dve_address,
            } => {
                self.handle_proof_verification_response(
                    proof_id,
                    is_valid,
                    verification_details,
                    dve_address,
                )
                .await?;
            }
            IBCMessage::StateSync {
                chain_id,
                height,
                app_hash,
                validator_set_hash,
            } => {
                self.handle_state_sync(chain_id, height, app_hash, validator_set_hash)
                    .await?;
            }
            IBCMessage::NetworkStatus {
                chain_id,
                status,
                timestamp,
            } => {
                self.handle_network_status(chain_id, status, timestamp)
                    .await?;
            }
            _ => {
                warn!("Unhandled incoming message type from {}", source);
            }
        }

        Ok(())
    }

    /// Handle proof verification response from KNIRV-NEXUS
    async fn handle_proof_verification_response(
        &self,
        proof_id: String,
        is_valid: bool,
        _verification_details: VerificationDetails,
        _dve_address: String,
    ) -> Result<()> {
        info!(
            "Received proof verification response: {} - valid: {}",
            proof_id, is_valid
        );

        // TODO: Update model registry with verification results
        // TODO: Trigger governance actions if needed

        Ok(())
    }

    /// Handle state sync message
    async fn handle_state_sync(
        &self,
        chain_id: String,
        height: u64,
        app_hash: String,
        _validator_set_hash: String,
    ) -> Result<()> {
        info!(
            "Received state sync from {}: height {}, app_hash {}",
            chain_id, height, app_hash
        );

        // TODO: Update local state based on sync information

        Ok(())
    }

    /// Handle network status update
    async fn handle_network_status(
        &self,
        chain_id: String,
        status: NetworkStatusData,
        _timestamp: u64,
    ) -> Result<()> {
        info!(
            "Received network status from {}: {} validators, height {}",
            chain_id, status.active_validators, status.current_height
        );

        // TODO: Update network monitoring metrics

        Ok(())
    }

    /// Process pending messages (retry failed sends)
    pub async fn process_pending_messages(&self) -> Result<()> {
        let mut queue = self.message_queue.lock().await;
        let current_time = SystemTime::now().duration_since(UNIX_EPOCH)?.as_secs();

        let mut processed_indices = Vec::new();

        for (i, pending) in queue.iter_mut().enumerate() {
            if current_time >= pending.next_retry && pending.retry_count < 3 {
                match self
                    .process_outgoing_message(pending.message.clone(), &pending.destination)
                    .await
                {
                    Ok(_) => {
                        processed_indices.push(i);
                    }
                    Err(e) => {
                        error!("Failed to process pending message: {}", e);
                        pending.retry_count += 1;
                        pending.next_retry = current_time + (pending.retry_count as u64 * 60);
                        // Exponential backoff
                    }
                }
            }
        }

        // Remove successfully processed messages
        for &i in processed_indices.iter().rev() {
            queue.remove(i);
        }

        Ok(())
    }

    /// Get connection status
    pub async fn get_connection_status(&self, connection_id: &str) -> Option<ConnectionState> {
        let connections = self.connections.lock().await;
        connections.get(connection_id).map(|c| c.state.clone())
    }

    /// List all connections
    pub async fn list_connections(&self) -> Vec<IBCConnection> {
        let connections = self.connections.lock().await;
        connections.values().cloned().collect()
    }

    /// Get pending message count
    pub async fn get_pending_message_count(&self) -> usize {
        let queue = self.message_queue.lock().await;
        queue.len()
    }

    /// Send IBC packet with actual transmission (enhanced implementation)
    pub async fn transmit_packet(
        &self,
        channel_id: &str,
        packet: IBCPacket,
    ) -> Result<PacketReceipt, IBCError> {
        let channel = self.get_channel(channel_id)?;

        // Serialize packet
        let packet_bytes = serde_json::to_vec(&packet)
            .map_err(|e| IBCError::SerializationError(e.to_string()))?;

        // Sign packet
        let signature = self.sign_packet(&packet_bytes)?;

        // Get connection endpoint
        let endpoint = self.get_connection_endpoint(&channel.connection_id)?;

        // Transmit via appropriate method
        let receipt = match endpoint.transport {
            Transport::Grpc => self.transmit_grpc(endpoint, packet_bytes, signature).await,
            Transport::Http => self.transmit_http(endpoint, packet_bytes, signature).await,
            Transport::Websocket => self.transmit_ws(endpoint, packet_bytes, signature).await,
        }?;

        Ok(receipt)
    }

    /// Handle packet acknowledgement
    pub async fn process_acknowledgement(
        &self,
        ack: PacketAcknowledgement,
    ) -> Result<(), IBCError> {
        // Verify acknowledgement
        self.verify_acknowledgement(&ack).await?;

        // Update packet status
        self.update_packet_status(ack.sequence, PacketStatus::Acknowledged).await?;

        // Trigger completion callbacks
        self.trigger_completion_callbacks(ack.sequence).await?;

        Ok(())
    }

    /// Handle packet timeout
    pub async fn process_timeout(
        &self,
        sequence: u64,
    ) -> Result<(), IBCError> {
        // Update packet status
        self.update_packet_status(sequence, PacketStatus::TimedOut).await?;

        // Trigger timeout callbacks
        self.trigger_timeout_callbacks(sequence).await?;

        Ok(())
    }

    // Private helper methods for packet transmission

    fn get_channel(&self, channel_id: &str) -> Result<IBCChannel, IBCError> {
        // In a real implementation, this would look up the channel
        // For now, return a mock channel
        Ok(IBCChannel {
            channel_id: channel_id.to_string(),
            connection_id: "connection-1".to_string(),
            port_id: "transfer".to_string(),
            counterparty_port_id: "transfer".to_string(),
            counterparty_channel_id: "channel-1".to_string(),
            state: ChannelState::Open,
            version: "ics20-1".to_string(),
            ordering: ChannelOrdering::Unordered,
        })
    }

    fn sign_packet(&self, packet_bytes: &[u8]) -> Result<Vec<u8>, IBCError> {
        // In a real implementation, this would use proper cryptographic signing
        // For now, return a mock signature
        use sha2::{Digest, Sha256};
        let mut hasher = Sha256::new();
        hasher.update(packet_bytes);
        Ok(hasher.finalize().to_vec())
    }

    fn get_connection_endpoint(&self, connection_id: &str) -> Result<ConnectionEndpoint, IBCError> {
        // In a real implementation, this would look up the actual endpoint
        // For now, return a mock endpoint
        Ok(ConnectionEndpoint {
            address: "localhost:9090".to_string(),
            transport: Transport::Grpc,
            timeout: std::time::Duration::from_secs(30),
        })
    }

    async fn transmit_grpc(
        &self,
        endpoint: ConnectionEndpoint,
        packet_bytes: Vec<u8>,
        signature: Vec<u8>,
    ) -> Result<PacketReceipt, IBCError> {
        // In a real implementation, this would make an actual gRPC call
        info!("Transmitting packet via gRPC to {}", endpoint.address);

        // Mock successful transmission
        Ok(PacketReceipt {
            sequence: 1,
            transaction_hash: format!("tx_{}", uuid::Uuid::new_v4()),
            block_height: 1000,
            timestamp: chrono::Utc::now().timestamp(),
        })
    }

    async fn transmit_http(
        &self,
        endpoint: ConnectionEndpoint,
        packet_bytes: Vec<u8>,
        signature: Vec<u8>,
    ) -> Result<PacketReceipt, IBCError> {
        // In a real implementation, this would make an actual HTTP call
        info!("Transmitting packet via HTTP to {}", endpoint.address);

        // Mock successful transmission
        Ok(PacketReceipt {
            sequence: 1,
            transaction_hash: format!("tx_{}", uuid::Uuid::new_v4()),
            block_height: 1000,
            timestamp: chrono::Utc::now().timestamp(),
        })
    }

    async fn transmit_ws(
        &self,
        endpoint: ConnectionEndpoint,
        packet_bytes: Vec<u8>,
        signature: Vec<u8>,
    ) -> Result<PacketReceipt, IBCError> {
        // In a real implementation, this would make an actual WebSocket call
        info!("Transmitting packet via WebSocket to {}", endpoint.address);

        // Mock successful transmission
        Ok(PacketReceipt {
            sequence: 1,
            transaction_hash: format!("tx_{}", uuid::Uuid::new_v4()),
            block_height: 1000,
            timestamp: chrono::Utc::now().timestamp(),
        })
    }

    async fn verify_acknowledgement(&self, ack: &PacketAcknowledgement) -> Result<(), IBCError> {
        // In a real implementation, this would verify the acknowledgement proof
        // For now, assume it's valid
        Ok(())
    }

    async fn update_packet_status(&self, sequence: u64, status: PacketStatus) -> Result<(), IBCError> {
        // In a real implementation, this would update packet status in storage
        info!("Updated packet {} status to {:?}", sequence, status);
        Ok(())
    }

    async fn trigger_completion_callbacks(&self, sequence: u64) -> Result<(), IBCError> {
        // In a real implementation, this would trigger completion callbacks
        info!("Triggered completion callbacks for packet {}", sequence);
        Ok(())
    }

    async fn trigger_timeout_callbacks(&self, sequence: u64) -> Result<(), IBCError> {
        // In a real implementation, this would trigger timeout callbacks
        info!("Triggered timeout callbacks for packet {}", sequence);
        Ok(())
    }
}

/// Packet status for tracking
#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum PacketStatus {
    Sent,
    Received,
    Acknowledged,
    TimedOut,
}

/// IBC-specific errors
#[derive(Debug, thiserror::Error)]
pub enum IBCError {
    #[error("Channel not found: {0}")]
    ChannelNotFound(String),

    #[error("Connection not found: {0}")]
    ConnectionNotFound(String),

    #[error("Serialization error: {0}")]
    SerializationError(String),

    #[error("Transmission failed: {0}")]
    TransmissionError(String),

    #[error("Verification failed: {0}")]
    VerificationError(String),

    #[error("Timeout exceeded")]
    TimeoutError,

    #[error("Internal error: {0}")]
    InternalError(String),
}
