// Core modules
pub mod blockchain_adapter;
pub mod config;
pub mod nrn_token;
pub mod smart_contracts;
pub mod testnet;

// New infrastructure modules
pub mod governance;
pub mod ipfs_client;
pub mod model_registry;
pub mod multi_model_engine;

// Consensus and networking
pub mod ibc_handler;
pub mod tendermint_consensus;

// TEE and skill distribution
pub mod tee_skill_distributor;

// Cloud model integration
pub mod cloud_models;

// WASM interface (only compiled when wasm feature is enabled)
#[cfg(feature = "wasm")]
pub mod wasm_interface;
