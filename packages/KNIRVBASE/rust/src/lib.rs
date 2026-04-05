use serde::{Deserialize, Serialize};
use std::sync::Arc;

pub mod clock;
pub mod types;
pub mod crypto;
pub mod storage;
pub mod network;
pub mod resolver;
pub mod collection;
pub mod database;
pub mod auth;
pub mod security;
pub mod wal;
pub mod hnsw;
pub mod embedding;
pub mod index_manager;
pub mod query_parser;

pub use clock::VectorClock;
pub use types::*;
pub use crypto::pqc::*;
pub use storage::{FileStorage, Storage};
pub use network::{NetworkManager, Network};
pub use resolver::*;
pub use database::DistributedDatabase;
pub use auth::*;
pub use security::*;
pub use wal::{WAL, WALEntry};
pub use hnsw::HNSWIndex;
pub use embedding::{TfidfVectorizer, LsaReducer, TfidfEmbedder, Embedder, Storage as EmbeddingStorage};
pub use index_manager::{IndexManager, Index, IndexType};
pub use query_parser::{Query, QueryType, KNIRVQLParser};

pub type Collection = crate::collection::DistributedCollection;

pub struct CollectionAdapter {
    pub c: Arc<tokio::sync::RwLock<crate::collection::DistributedCollection>>,
}

impl CollectionAdapter {
    pub async fn attach_to_network(&mut self, network_id: &str) -> Result<(), Box<dyn std::error::Error + Send + Sync>> {
        self.c.write().await.attach_to_network(network_id).await
    }

    pub async fn detach_from_network(&mut self) -> Result<(), Box<dyn std::error::Error + Send + Sync>> {
        self.c.write().await.detach_from_network().await
    }

    pub async fn insert(&self, doc: std::collections::HashMap<String, serde_json::Value>) -> Result<std::collections::HashMap<String, serde_json::Value>, Box<dyn std::error::Error + Send + Sync>> {
        self.c.read().await.insert("", doc).await
    }

    pub async fn update(&self, id: &str, update: std::collections::HashMap<String, serde_json::Value>) -> Result<i32, Box<dyn std::error::Error + Send + Sync>> {
        self.c.read().await.update(id, update).await
    }

    pub async fn delete(&self, id: &str) -> Result<i32, Box<dyn std::error::Error + Send + Sync>> {
        self.c.read().await.delete(id).await
    }

    pub async fn find(&self, id: &str) -> Result<Option<std::collections::HashMap<String, serde_json::Value>>, Box<dyn std::error::Error + Send + Sync>> {
        self.c.read().await.find(id).await
    }

    pub async fn find_all(&self) -> Result<Vec<std::collections::HashMap<String, serde_json::Value>>, Box<dyn std::error::Error + Send + Sync>> {
        self.c.read().await.find_all().await
    }

    pub async fn force_sync(&self) -> Result<(), Box<dyn std::error::Error + Send + Sync>> {
        self.c.read().await.force_sync().await
    }
}

#[derive(Debug, Clone)]
pub struct Options {
    pub data_dir: String,
    pub distributed_enabled: bool,
    pub distributed_network_id: String,
    pub distributed_bootstrap_peers: Vec<String>,
}

pub mod prelude {
    pub use super::clock::VectorClock;
    pub use super::types::*;
    pub use super::crypto::pqc::*;
    pub use super::storage::{FileStorage, Storage};
    pub use super::network::{NetworkManager, Network};
    pub use super::resolver::*;
    pub use super::collection::DistributedCollection;
    pub use super::database::{DistributedDatabase, DistributedDbOptions, DistributedOptions};
    pub use super::auth::{TokenManager, Claims, AuthMiddleware};
    pub use super::security::MemoryEncryption;
    pub use super::wal::{WAL, WALEntry};
    pub use super::hnsw::HNSWIndex;
    pub use super::embedding::{TfidfVectorizer, LsaReducer, TfidfEmbedder, Embedder, Storage as EmbeddingStorage};
    pub use super::index_manager::{IndexManager, Index, IndexType};
    pub use super::query_parser::{Query, QueryType, KNIRVQLParser};
    pub use super::{Collection, CollectionAdapter, Options};
    pub use super::{MemoryCategory, ModalityType, Frame, ThermoData, FrameEntry, Registry, NRV_MAGIC, NRV_VERSION};
}

pub struct DB {
    db: DistributedDatabase,
    storage: Arc<dyn Storage>,
}

impl DB {
    pub async fn new(opts: Options) -> Result<Self, Box<dyn std::error::Error + Send + Sync>> {
        if opts.data_dir.is_empty() {
            return Err("DataDir cannot be empty".into());
        }

        let storage: Arc<dyn Storage> = Arc::new(FileStorage::new(opts.data_dir.clone())?);
        let network = Arc::new(NetworkManager::new());

        let db_opts = crate::database::DistributedDbOptions {
            distributed: crate::database::DistributedOptions {
                enabled: opts.distributed_enabled,
                network_id: opts.distributed_network_id,
                bootstrap_peers: opts.distributed_bootstrap_peers,
            },
        };

        let db = DistributedDatabase::new(db_opts, storage.clone(), network).await?;

        Ok(DB { db, storage })
    }

    pub async fn create_network(&self, cfg: NetworkConfig) -> Result<String, Box<dyn std::error::Error + Send + Sync>> {
        self.db.create_network(cfg).await
    }

    pub async fn collection(&self, name: &str) -> Result<CollectionAdapter, Box<dyn std::error::Error + Send + Sync>> {
        if name.is_empty() {
            return Err("collection name cannot be empty".into());
        }

        let coll = self.db.collection(name).await;
        Ok(CollectionAdapter { c: coll })
    }

    pub fn raw(&self) -> &DistributedDatabase {
        &self.db
    }

    pub async fn shutdown(self) -> Result<(), Box<dyn std::error::Error + Send + Sync>> {
        self.db.shutdown().await
    }
}

#[derive(Debug, Clone)]
pub struct MemoryCategory(pub String);

impl MemoryCategory {
    pub const ERROR: &'static str = "ERROR";
    pub const CONTEXT: &'static str = "CONTEXT";
    pub const IDEA: &'static str = "IDEA";
    pub const SOLUTION: &'static str = "SOLUTION";
    pub const SKILL: &'static str = "SKILL";
    pub const GENERIC: &'static str = "GENERIC";
    pub const EVENT: &'static str = "EVENT";
    pub const PREFERENCE: &'static str = "PREFERENCE";
    pub const TRAIT: &'static str = "TRAIT";

    pub fn new(s: &str) -> Self {
        MemoryCategory(s.to_string())
    }

    pub fn as_str(&self) -> &str {
        &self.0
    }
}

impl Default for MemoryCategory {
    fn default() -> Self {
        MemoryCategory("GENERIC".to_string())
    }
}

impl std::fmt::Display for MemoryCategory {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        write!(f, "{}", self.0)
    }
}

impl std::str::FromStr for MemoryCategory {
    type Err = String;

    fn from_str(s: &str) -> Result<Self, Self::Err> {
        Ok(MemoryCategory(s.to_string()))
    }
}

#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
pub enum ModalityType {
    Vector,
    Audio,
    Video,
    Image,
    Text,
}

impl std::fmt::Display for ModalityType {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        match self {
            ModalityType::Vector => write!(f, "vector"),
            ModalityType::Audio => write!(f, "audio"),
            ModalityType::Video => write!(f, "video"),
            ModalityType::Image => write!(f, "image"),
            ModalityType::Text => write!(f, "text"),
        }
    }
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Frame {
    pub id: String,
    pub vector: [f32; 12],
    pub seed: [u8; 32],
    pub thermo: ThermoData,
    pub proof: Vec<u8>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ThermoData {
    pub temp_celsius: f32,
    pub voltage_v: f32,
    pub freq_mhz: f32,
    pub fan_rpm: f32,
}

impl Default for ThermoData {
    fn default() -> Self {
        Self {
            temp_celsius: 0.0,
            voltage_v: 0.0,
            freq_mhz: 0.0,
            fan_rpm: 0.0,
        }
    }
}

impl Default for Frame {
    fn default() -> Self {
        Self {
            id: String::new(),
            vector: [0.0; 12],
            seed: [0u8; 32],
            thermo: ThermoData::default(),
            proof: Vec::new(),
        }
    }
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct FrameEntry {
    pub id: String,
    pub offset: u64,
    pub length: u32,
    pub tombstone: Option<i64>,
    pub verified: bool,
    pub ergo_rank: f64,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ModalityIndex {
    pub offset: u64,
    pub length: u32,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GlobalMetrics {
    pub total_frames: u64,
    pub verified_frames: u64,
    pub average_ergo_rank: f64,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PQCManifest {
    pub public_key: Vec<u8>,
    pub signature: Vec<u8>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Registry {
    pub version: u32,
    pub dataset_id: String,
    pub chunk0_length: u64,
    pub frame_count: u64,
    pub tombstone_count: u64,
    pub global_metrics: GlobalMetrics,
    pub frames: Vec<FrameEntry>,
    pub pqc_manifest: Option<PQCManifest>,
}

impl Default for Registry {
    fn default() -> Self {
        Self {
            version: 1,
            dataset_id: String::new(),
            chunk0_length: 4096,
            frame_count: 0,
            tombstone_count: 0,
            global_metrics: GlobalMetrics {
                total_frames: 0,
                verified_frames: 0,
                average_ergo_rank: 0.0,
            },
            frames: Vec::new(),
            pqc_manifest: None,
        }
    }
}

pub const NRV_MAGIC: u32 = 0x4E525621;
pub const NRV_VERSION: u32 = 1;

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_memory_category_display() {
        let cat = MemoryCategory::new("ERROR");
        assert_eq!(format!("{}", cat), "ERROR");
    }

    #[test]
    fn test_frame_default() {
        let frame = Frame::default();
        assert_eq!(frame.id, "");
        assert_eq!(frame.vector, [0.0; 12]);
        assert_eq!(frame.thermo.temp_celsius, 0.0);
    }

    #[test]
    fn test_nrv_constants() {
        assert_eq!(NRV_MAGIC, 0x4E525621);
        assert_eq!(NRV_VERSION, 1);
    }
}
