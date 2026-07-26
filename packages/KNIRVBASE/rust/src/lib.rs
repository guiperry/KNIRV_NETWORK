use serde::{Deserialize, Serialize};
use std::sync::Arc;

pub mod clock;
pub mod nrv;
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
pub mod logging;
pub mod monitoring;

pub use clock::VectorClock;
pub use nrv::{NRVWriter, NRVReader, encode_frame, decode_frame};
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
pub use logging::{Logger, Level, LogEntry};
pub use monitoring::{Metrics, Counter, Gauge, Histogram};

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
    pub use super::logging::{Logger, Level, LogEntry};
    pub use super::monitoring::{Metrics, Counter, Gauge, Histogram};
    pub use super::{Collection, CollectionAdapter, Options};
}

pub struct DB {
    db: DistributedDatabase,
    _storage: Arc<dyn Storage>,
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

        Ok(DB { db, _storage: storage })
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



#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
pub enum MediaModalityType {
    Vector,
    Audio,
    Video,
    Image,
    Text,
}

impl std::fmt::Display for MediaModalityType {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        match self {
            MediaModalityType::Vector => write!(f, "vector"),
            MediaModalityType::Audio => write!(f, "audio"),
            MediaModalityType::Video => write!(f, "video"),
            MediaModalityType::Image => write!(f, "image"),
            MediaModalityType::Text => write!(f, "text"),
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
    pub offset: i64,
    pub length: i32,
    pub tombstone: Option<i64>,
    pub verified: bool,
    pub ergo_rank: f64,
    pub modalities: std::collections::HashMap<String, ModalityIndex>,
}

#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
pub struct ModalityIndex {
    pub offset: i64,
    pub length: i32,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GlobalMetrics {
    pub feature_min: [f32; 12],
    pub feature_max: [f32; 12],
    pub feature_mean: [f32; 12],
    pub feature_std: [f32; 12],
    pub thermo_correlation_coefficient: f64,
    pub ergo_rank_sum: f64,
    pub verified_frame_count: i32,
    pub compacted_at: Option<String>,
}

impl Default for GlobalMetrics {
    fn default() -> Self {
        Self {
            feature_min: [0.0; 12],
            feature_max: [0.0; 12],
            feature_mean: [0.0; 12],
            feature_std: [0.0; 12],
            thermo_correlation_coefficient: 0.0,
            ergo_rank_sum: 0.0,
            verified_frame_count: 0,
            compacted_at: None,
        }
    }
}

#[derive(Debug, Clone, Serialize, Deserialize, Default)]
pub struct PQCManifest {
    pub key_id: String,
    pub algorithm: String,
    pub file_signature: String,
    pub frame_signatures: std::collections::HashMap<String, String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Registry {
    pub version: i32,
    pub dataset_id: String,
    pub dataset_version: crate::clock::VectorClock,
    pub chunk0_length: i32,
    pub frame_count: i32,
    pub tombstone_count: i32,
    pub global_metrics: GlobalMetrics,
    pub frames: Vec<FrameEntry>,
    pub pqc_manifest: PQCManifest,
}

impl Default for Registry {
    fn default() -> Self {
        Self {
            version: 1,
            dataset_id: String::new(),
            dataset_version: crate::clock::VectorClock::new(),
            chunk0_length: 4096,
            frame_count: 0,
            tombstone_count: 0,
            global_metrics: GlobalMetrics::default(),
            frames: Vec::new(),
            pqc_manifest: PQCManifest::default(),
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
        let cat = crate::types::MemoryCategory::from_str("ERROR").unwrap();
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
