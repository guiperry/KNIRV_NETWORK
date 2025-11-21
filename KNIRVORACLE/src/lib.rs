// Core modules
pub mod blockchain_adapter;
pub mod config;
pub mod nrn_token;
pub mod smart_contracts;
pub mod testnet;
pub mod token_economics;
pub mod uri_resolver;

// New infrastructure modules
pub mod governance;
pub mod ipfs_client;
pub mod model_registry;
pub mod multi_model_engine;

// Cross-chain transfer modules
pub mod cross_chain;

// Registry event listener
pub mod registry_listener;

// Consensus and networking
pub mod ibc_handler;
pub mod tendermint_consensus;

// P2P and DHT
pub mod dht_manager;

// LoRA and skill distribution
pub mod lora_skill_distributor;

// Cloud model integration
pub mod cloud_models;

// WASM interface (only compiled when wasm feature is enabled)
#[cfg(feature = "wasm")]
pub mod wasm_interface;
