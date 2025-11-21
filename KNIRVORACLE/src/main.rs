use crate::blockchain_adapter::*;
use crate::config::Config;
use crate::nrn_token::*;
use crate::smart_contracts::*;
use actix_web::rt::spawn;
use actix_web::{get, post, web, App, Error, HttpResponse, HttpServer, Responder};
use clap::Parser;
use dotenv::dotenv;
use futures::executor::block_on;
use num_bigint::BigInt;
use rand::Rng;
use serde::{Deserialize, Serialize};
use sha2::{Digest, Sha256};
use sled::Db;
use std::collections::HashMap;
use std::env;
use std::sync::Arc;
use std::time::{Duration, SystemTime, UNIX_EPOCH};
use tokio::sync::Mutex;
use tracing::{error, info, subscriber::set_global_default};
use tracing_subscriber::fmt;

mod blockchain_adapter;
mod config;
mod nrn_token;
mod smart_contracts;
mod testnet;
mod token_economics;

// New infrastructure modules
mod governance;
mod ipfs_client;
mod model_registry;
mod multi_model_engine;

// Cross-chain transfer modules
mod cross_chain;

// Consensus and networking
mod ibc_handler;
mod tendermint_consensus;

// LoRA and skill distribution
mod lora_skill_distributor;

// Cloud model integration
mod cloud_models;

/// KNIRVORACLE - A blockchain for AI model and skill management
#[derive(Parser, Debug)]
#[command(author, version, about, long_about = None)]
struct Args {
    /// Disable auto-mining (reduces resource usage)
    #[arg(long)]
    disable_mining: bool,

    /// Run in testnet mode
    #[arg(long)]
    testnet: bool,
}

// Custom error wrapper for anyhow::Error to implement ResponseError
#[derive(Debug)]
struct AnyhowError(anyhow::Error);

impl std::fmt::Display for AnyhowError {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        write!(f, "{}", self.0)
    }
}

impl actix_web::ResponseError for AnyhowError {}

impl From<anyhow::Error> for AnyhowError {
    fn from(err: anyhow::Error) -> Self {
        AnyhowError(err)
    }
}

// Structs

#[derive(Debug, Serialize, Deserialize, Clone)]
struct Block {
    index: u64,
    timestamp: u64,
    data: String,
    previous_hash: String,
    nonce: u64,
    hash: String,
}

#[derive(Debug, Serialize, Deserialize, Clone)]
struct BlockchainResponse {
    message: String,
    data: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    transaction_hash: Option<String>,
}

// In memory transaction pool
#[derive(Debug)]
#[allow(dead_code)]
struct SharedState {
    transaction_pool: Mutex<Vec<Transaction>>,
    blockchain: Mutex<Vec<Block>>,
    sled_db: Mutex<Db>,
    nrn: Mutex<NRN>,
    smart_contracts: Mutex<SmartContractEngine>,
    blockchain_adapter: Arc<BlockchainAdapter>,

    // New infrastructure components
    ipfs_client: Arc<ipfs_client::IpfsClient>,
    multi_model_engine: Arc<Mutex<multi_model_engine::MultiModelEngine>>,
    model_registry: Arc<model_registry::EnhancedMultiModelRegistry>,
    governance: Arc<governance::GovernanceSystem>,

    // Consensus and networking
    tendermint_consensus: Arc<Mutex<tendermint_consensus::TendermintConsensus>>,
    ibc_handler: Arc<ibc_handler::IBCHandler>,

    // LoRA and skill distribution
    lora_skill_distributor: Arc<lora_skill_distributor::LoRASkillDistributor>,

    // Cloud model testing (optional)
    cloud_testing_framework: Arc<Mutex<Option<cloud_models::CloudModelTestingFramework>>>,

    // Mining configuration
    mining_enabled: bool,
}

// Helper function to convert byte array to hex string
fn bytes_to_hex(bytes: &[u8]) -> String {
    bytes
        .iter()
        .map(|b| format!("{:02x}", b))
        .collect::<String>()
}

impl Block {
    fn calculate_hash(&self) -> String {
        let mut hasher = Sha256::new();
        hasher.update(
            format!(
                "{}{}{}{}{}",
                self.index, self.previous_hash, self.timestamp, self.data, self.nonce
            )
            .as_bytes(),
        );
        let result = hasher.finalize();
        bytes_to_hex(&result)
    }

    fn mine_block(&mut self, difficulty: u32) {
        let target = "0".repeat(difficulty as usize);
        info!(
            "[INFO] Mining block with difficulty: {}, current hash: {}, target: {}",
            difficulty, self.hash, target
        );
        while !self.hash.starts_with(&target) {
            self.nonce = rand::thread_rng().gen();
            self.hash = self.calculate_hash();
            info!(
                "[INFO] Trying nonce: {}, new hash: {}",
                self.nonce, self.hash
            );
        }
        info!("[INFO] Block Mined: {}", self.hash)
    }
}

// Load chain from Sled
async fn load_chain(db: &Mutex<Db>) -> Result<Vec<Block>, String> {
    let database = db.lock().await;
    let mut blocks: Vec<Block> = Vec::new();

    for item in database.iter() {
        let (_key_bytes, value_bytes) = item.map_err(|_| "Error reading database")?;
        let block: Block =
            serde_json::from_slice(&value_bytes).map_err(|_| "Error deserializing block")?;
        blocks.push(block);
    }

    blocks.sort_by_key(|block| block.index); // Sort blocks by index
    Ok(blocks)
}

//Save block to Sled
async fn save_block(db: &Mutex<Db>, block: &Block) -> Result<(), String> {
    let database = db.lock().await;
    let key = block.index.to_string();
    let value = serde_json::to_vec(block).map_err(|_| "Error serializing block")?;
    database
        .insert(key, value)
        .map_err(|_| "Error saving block to database")?;
    Ok(())
}

fn create_genesis_block() -> Block {
    Block {
        index: 0,
        timestamp: SystemTime::now()
            .duration_since(UNIX_EPOCH)
            .unwrap()
            .as_secs(),
        data: String::from("Genesis Block"),
        previous_hash: String::from("0"),
        nonce: 1984,
        hash: String::new(),
    }
}

async fn get_latest_block(blockchain: &Mutex<Vec<Block>>) -> Result<Block, String> {
    let chain = blockchain.lock().await;
    match chain.last().cloned() {
        Some(block) => Ok(block),
        None => Err("Blockchain is empty".to_string()),
    }
}

// Function to add a block to the chain, including database integration
async fn add_block(
    state: &SharedState,
    mut new_block: Block,
    difficulty: u32,
) -> Result<(), String> {
    //Get latest block from the chain
    let latest_block = match get_latest_block(&state.blockchain).await {
        Ok(block) => block,
        Err(e) => {
            error!("[ERROR] Cannot get latest block. Error: {}", e);
            return Err(format!("Could not get latest block, message: {}", e));
        }
    };
    //Set the new block previous hash to the latest block hash or default to 0
    let previous_hash = latest_block.hash;
    info!("[INFO] Adding block. Previous Hash: {}", previous_hash);
    new_block.previous_hash = previous_hash;

    new_block.mine_block(difficulty);

    //Acquire locks in correct order: database, and then blockchain
    let _db = state.sled_db.lock().await;
    let mut chain = state.blockchain.lock().await;
    //Save the block into the database
    save_block(&state.sled_db, &new_block).await?;

    chain.push(new_block.clone());

    info!("[INFO] Successfully added block {}", new_block.index);
    Ok(())
}

// Handler for sending transactions
#[post("/send_txn")]
async fn send_transaction(
    state: web::Data<Arc<SharedState>>,
    transaction: web::Json<Transaction>,
    difficulty: web::Data<u32>,
) -> Result<impl Responder, Error> {
    let txn = transaction.into_inner();
    info!("[INFO] Received transaction, current pool: {:?}", txn);
    let transaction_hash = bytes_to_hex(&Sha256::digest(
        serde_json::to_string(&txn).unwrap().as_bytes(),
    ));

    let difficulty_clone = **difficulty; //Double dereference to get u32 from Arc<u32>
    let state_clone = state.clone();

    // Only trigger automatic mining if mining is enabled
    if state.mining_enabled {
        spawn(async move {
            // 1. Acquire locks in consistent order (blockchain first, then transaction pool):
            let block_index: u64 = match get_latest_block(&state_clone.blockchain).await {
                Ok(block) => block.index + 1,
                Err(e) => {
                    error!(
                        "[ERROR] Cannot get latest block, could not mine block, Error: {}",
                        e
                    );
                    return;
                }
            };
            let mut pool = state_clone.transaction_pool.lock().await;
            let transactions: Vec<Transaction> = pool.drain(..).collect(); // Get all the transactions
            info!(
                "[INFO] Clearing transaction pool, new block has: {} transactions",
                transactions.len()
            );

            let new_block = Block {
                index: block_index,
                timestamp: SystemTime::now()
                    .duration_since(UNIX_EPOCH)
                    .unwrap()
                    .as_secs(),
                data: serde_json::to_string(&transactions).unwrap(),
                previous_hash: String::new(),
                nonce: 0,
                hash: String::new(),
            };
            match add_block(&state_clone, new_block, difficulty_clone).await {
                Ok(_) => info!("[INFO] Successfully added new block automatically with content"),
                Err(e) => error!(
                    "[ERROR] Error mining new block automatically with content: {}",
                    e
                ),
            }
        });
    } else {
        // Just add transaction to pool without mining
        let mut pool = state.transaction_pool.lock().await;
        pool.push(txn.clone());
        info!("[INFO] Transaction added to pool (mining disabled)");
    }

    let message = if state.mining_enabled {
        "Transaction submitted successfully (mining async)".to_string()
    } else {
        "Transaction submitted successfully (added to pool, mining disabled)".to_string()
    };

    Ok(HttpResponse::Created().json(BlockchainResponse {
        message,
        data: Some(serde_json::to_string(&txn).unwrap()),
        transaction_hash: Some(transaction_hash),
    }))
}

// Handler to create a new wallet (private key + address)
#[get("/wallets/new")]
async fn new_wallet() -> Result<impl Responder, Error> {
    let private_key = generate_private_key();
    let address = get_address_from_private_key(&private_key).map_err(AnyhowError::from)?;

    Ok(HttpResponse::Created().json(BlockchainResponse {
        message: "New Wallet Created!".to_string(),
        data: Some(format!(
            "Private key: {}, address: {}",
            private_key, address
        )),
        transaction_hash: None,
    }))
}

// Handler for minting tokens
#[post("/nrn/mint")]
async fn nrn_mint(
    state: web::Data<Arc<SharedState>>,
    mint_request: web::Json<HashMap<String, String>>,
) -> Result<impl Responder, Error> {
    let addr_map = mint_request.into_inner();
    let to_address = match addr_map.get("to") {
        Some(value) => hex_to_address(value).map_err(|e| {
            error!("[ERROR] Could not parse the provided 'to' address: {}.", e);
            Error::from(actix_web::error::ErrorBadRequest(format!(
                "Could not parse the provided 'to' address: {}.",
                e
            )))
        })?,
        None => {
            return Err(Error::from(actix_web::error::ErrorBadRequest(
                "'to' address is required.",
            )))
        }
    };

    let from_private_key = match addr_map.get("from") {
        Some(value) => value,
        None => {
            return Err(Error::from(actix_web::error::ErrorBadRequest(
                "'from' address is required.",
            )))
        }
    };

    let mint_amount: BigInt = match addr_map.get("amount") {
        Some(value) => value.parse().map_err(|e| {
            error!("[ERROR] Could not parse the provided mint 'amount': {}", e);
            Error::from(actix_web::error::ErrorBadRequest(format!(
                "Could not parse the provided mint 'amount': {}",
                e
            )))
        })?,
        None => {
            return Err(Error::from(actix_web::error::ErrorBadRequest(
                "'amount' is required.",
            )))
        }
    };

    let mut nrn = state.nrn.lock().await; // Obtain lock to the NRN structure.
    match nrn.mint(from_private_key, to_address, &mint_amount) {
        Ok(_) => {
            info!("[INFO] Tokens minted.");
            Ok(HttpResponse::Created().json(BlockchainResponse {
                message: "Tokens Minted".to_string(),
                data: None,
                transaction_hash: None,
            }))
        }
        Err(e) => {
            error!("[ERROR] Could not mint new tokens. Error: {}", e);
            Err(Error::from(actix_web::error::ErrorBadRequest(format!(
                "Could not mint new tokens: {}",
                e
            ))))
        }
    }
}

// Handler for transfering tokens.
#[post("/nrn/transfer")]
async fn nrn_transfer(
    state: web::Data<Arc<SharedState>>,
    transfer_request: web::Json<HashMap<String, String>>,
) -> Result<impl Responder, Error> {
    let addr_map = transfer_request.into_inner();
    let to_address = match addr_map.get("to") {
        Some(value) => hex_to_address(value).map_err(|e| {
            error!("[ERROR] Could not parse the provided 'to' address: {}.", e);
            Error::from(actix_web::error::ErrorBadRequest(format!(
                "Could not parse the provided 'to' address: {}.",
                e
            )))
        })?,
        None => {
            return Err(Error::from(actix_web::error::ErrorBadRequest(
                "'to' address is required.",
            )))
        }
    };
    let from_private_key = match addr_map.get("from") {
        Some(value) => value,
        None => {
            return Err(Error::from(actix_web::error::ErrorBadRequest(
                "'from' address is required.",
            )))
        }
    };

    let transfer_amount: BigInt = match addr_map.get("amount") {
        Some(value) => value.parse().map_err(|e| {
            error!(
                "[ERROR] Could not parse the provided transfer 'amount': {}",
                e
            );
            Error::from(actix_web::error::ErrorBadRequest(format!(
                "Could not parse the provided transfer 'amount': {}",
                e
            )))
        })?,
        None => {
            return Err(Error::from(actix_web::error::ErrorBadRequest(
                "'amount' is required.",
            )))
        }
    };

    let mut nrn = state.nrn.lock().await;
    match nrn.transfer(from_private_key, to_address, &transfer_amount) {
        Ok(tx) => {
            info!("[INFO] tokens transfered, adding transaction to pool.");
            drop(nrn); // Release the lock before accessing transaction pool

            // Add transaction to the pool instead of calling the handler directly
            let mut pool = state.transaction_pool.lock().await;
            pool.push(tx.clone());
            drop(pool);

            Ok(HttpResponse::Ok().json(BlockchainResponse {
                message: "Transfer completed and transaction added to pool".to_string(),
                data: Some(serde_json::to_string(&tx).unwrap()),
                transaction_hash: None,
            }))
        }
        Err(e) => {
            error!("[ERROR] Could not transfer tokens. Error: {}", e);
            Err(Error::from(actix_web::error::ErrorBadRequest(format!(
                "Could not transfer tokens: {}",
                e
            ))))
        }
    }
}

// Handler for retrieving user balance.
#[post("/nrn/balance")]
async fn nrn_balance(
    state: web::Data<Arc<SharedState>>,
    balance_request: web::Json<HashMap<String, String>>,
) -> Result<impl Responder, Error> {
    let addr_map = balance_request.into_inner();
    let address = match addr_map.get("address") {
        Some(address_value) => hex_to_address(address_value).map_err(|e| {
            error!("[ERROR] Could not parse address. Error: {}", e);
            Error::from(actix_web::error::ErrorBadRequest(format!(
                "Could not parse address. Error: {}",
                e
            )))
        })?,
        None => {
            return Err(Error::from(actix_web::error::ErrorBadRequest(
                "Address required",
            )))
        }
    };

    let nrn = state.nrn.lock().await;
    let balance = nrn.get_balance(&address);
    info!("[INFO] Retrieved balance for {}", address);
    Ok(HttpResponse::Ok().json(BlockchainResponse {
        message: format!("Balance for {}: {}", address, balance),
        data: Some(format!("{}", balance)),
        transaction_hash: None,
    }))
}

// Handler to retrieve the  NRN info.
#[get("/nrn/info")]
async fn nrn_info(state: web::Data<Arc<SharedState>>) -> Result<impl Responder, Error> {
    let nrn = state.nrn.lock().await;
    let total_supply = nrn.get_total_supply();
    let owner = nrn.get_owner();

    Ok(HttpResponse::Ok().json(BlockchainResponse {
        message: "NRN Token information".to_string(),
        data: Some(format!("Total Supply: {}, Owner: {}", total_supply, owner)),
        transaction_hash: None,
    }))
}

// Handler to retrieve the whole blockchain.
#[get("/blocks")]
async fn get_blocks(state: web::Data<Arc<SharedState>>) -> Result<impl Responder, Error> {
    let blockchain = state.blockchain.lock().await;
    Ok(HttpResponse::Ok().json(blockchain.clone()))
}

// Handler for LLM registration
#[post("/llm/register")]
async fn register_llm(
    state: web::Data<Arc<SharedState>>,
    llm_request: web::Json<HashMap<String, serde_json::Value>>,
) -> Result<impl Responder, Error> {
    let mut smart_contracts = state.smart_contracts.lock().await;

    let contract_call = ContractCall {
        contract: "llm_registry".to_string(),
        method: "register".to_string(),
        params: serde_json::Value::Object(llm_request.into_inner().into_iter().collect()),
    };

    let response = smart_contracts.execute_contract_call(contract_call);

    if response.success {
        Ok(HttpResponse::Ok().json(response))
    } else {
        Ok(HttpResponse::BadRequest().json(response))
    }
}

// Handler for skill registration
#[post("/skill/register")]
async fn register_skill(
    state: web::Data<Arc<SharedState>>,
    skill_request: web::Json<HashMap<String, serde_json::Value>>,
) -> Result<impl Responder, Error> {
    let mut smart_contracts = state.smart_contracts.lock().await;

    let contract_call = ContractCall {
        contract: "skill_registry".to_string(),
        method: "register".to_string(),
        params: serde_json::Value::Object(skill_request.into_inner().into_iter().collect()),
    };

    let response = smart_contracts.execute_contract_call(contract_call);

    if response.success {
        Ok(HttpResponse::Ok().json(response))
    } else {
        Ok(HttpResponse::BadRequest().json(response))
    }
}

// Handler for skill invocation (burns NRN)
#[post("/skill/invoke")]
async fn invoke_skill(
    state: web::Data<Arc<SharedState>>,
    invoke_request: web::Json<HashMap<String, String>>,
) -> Result<impl Responder, Error> {
    let mut smart_contracts = state.smart_contracts.lock().await;

    let params = serde_json::Value::Object(
        invoke_request
            .into_inner()
            .into_iter()
            .map(|(k, v)| (k, serde_json::Value::String(v)))
            .collect(),
    );

    let contract_call = ContractCall {
        contract: "nrn_token".to_string(),
        method: "burn_for_skill".to_string(),
        params,
    };

    let response = smart_contracts.execute_contract_call(contract_call);

    if response.success {
        Ok(HttpResponse::Ok().json(response))
    } else {
        Ok(HttpResponse::BadRequest().json(response))
    }
}

// Enhanced LLM registration using blockchain adapter
#[post("/v2/llm/register")]
async fn register_llm_v2(
    _state: web::Data<Arc<SharedState>>,
    _llm_request: web::Json<LLMRegistrationRequest>,
) -> Result<impl Responder, Error> {
    // LLM registration has been moved to KNIRVCHAIN
    Ok(HttpResponse::BadRequest().json(serde_json::json!({
        "success": false,
        "error": "LLM registration has been moved to KNIRVCHAIN"
    })))
}

// Enhanced skill registration using blockchain adapter
#[post("/v2/skill/register")]
async fn register_skill_v2(
    _state: web::Data<Arc<SharedState>>,
    _skill_request: web::Json<SkillRegistrationRequest>,
) -> Result<impl Responder, Error> {
    // Skill registration has been moved to KNIRVCHAIN
    Ok(HttpResponse::BadRequest().json(serde_json::json!({
        "success": false,
        "error": "Skill registration has been moved to KNIRVCHAIN"
    })))
}

// Enhanced skill invocation using blockchain adapter
#[post("/v2/skill/invoke")]
async fn invoke_skill_v2(
    _state: web::Data<Arc<SharedState>>,
    _invoke_request: web::Json<SkillInvocationRequest>,
) -> Result<impl Responder, Error> {
    // Skill invocation fees are now collected via IBC from KNIRVCHAIN
    Ok(HttpResponse::BadRequest().json(serde_json::json!({
        "success": false,
        "error": "Skill invocation fees are now collected via IBC from KNIRVCHAIN"
    })))
}

// New API handlers for enhanced functionality

// List all registered models
async fn list_models(state: web::Data<Arc<SharedState>>) -> Result<impl Responder, Error> {
    let models = state.model_registry.list_models().await;
    Ok(HttpResponse::Ok().json(models))
}

// Switch active model
async fn switch_model(
    state: web::Data<Arc<SharedState>>,
    request: web::Json<HashMap<String, String>>,
) -> Result<impl Responder, Error> {
    let model_hash = request
        .get("model_hash")
        .ok_or_else(|| Error::from(actix_web::error::ErrorBadRequest("model_hash required")))?;

    let mut engine = state.multi_model_engine.lock().await;
    match engine.switch_model(model_hash).await {
        Ok(_) => Ok(HttpResponse::Ok().json(serde_json::json!({
            "success": true,
            "message": "Model switched successfully"
        }))),
        Err(e) => Ok(HttpResponse::BadRequest().json(serde_json::json!({
            "success": false,
            "error": e.to_string()
        }))),
    }
}

// Get model performance metrics
async fn get_model_performance(
    state: web::Data<Arc<SharedState>>,
) -> Result<impl Responder, Error> {
    let engine = state.multi_model_engine.lock().await;
    let model_info = engine.get_current_model_info();
    let model_hash = engine.get_current_model_hash();

    Ok(HttpResponse::Ok().json(serde_json::json!({
        "current_model": model_hash,
        "model_info": model_info,
        "health_status": engine.health_check().await.unwrap_or(false)
    })))
}

// List governance proposals
async fn list_proposals(state: web::Data<Arc<SharedState>>) -> Result<impl Responder, Error> {
    let proposals = state.governance.get_active_proposals().await;
    Ok(HttpResponse::Ok().json(proposals))
}

// Cast a governance vote
async fn cast_vote(
    _state: web::Data<Arc<SharedState>>,
    _vote_request: web::Json<HashMap<String, serde_json::Value>>,
) -> Result<impl Responder, Error> {
    // TODO: Implement vote casting with proper validation
    Ok(HttpResponse::Ok().json(serde_json::json!({
        "success": true,
        "message": "Vote cast successfully"
    })))
}

// Get consensus status
async fn consensus_status(state: web::Data<Arc<SharedState>>) -> Result<impl Responder, Error> {
    let consensus = state.tendermint_consensus.lock().await;
    let chain_state = consensus.get_chain_state().await;
    let validator_set = consensus.get_validator_set().await;
    let health = consensus.health_check().await.unwrap_or(false);

    Ok(HttpResponse::Ok().json(serde_json::json!({
        "chain_state": chain_state,
        "validator_count": validator_set.get_validator_count(),
        "health": health
    })))
}

// Get IBC connections
async fn ibc_connections(state: web::Data<Arc<SharedState>>) -> Result<impl Responder, Error> {
    let connections = state.ibc_handler.list_connections().await;
    let pending_messages = state.ibc_handler.get_pending_message_count().await;

    Ok(HttpResponse::Ok().json(serde_json::json!({
        "connections": connections,
        "pending_messages": pending_messages
    })))
}

// Prepare skill for LoRA execution
async fn prepare_lora_skill(
    state: web::Data<Arc<SharedState>>,
    request: web::Json<HashMap<String, serde_json::Value>>,
) -> Result<impl Responder, Error> {
    let skill_id = request
        .get("skill_id")
        .and_then(|v| v.as_str())
        .ok_or_else(|| Error::from(actix_web::error::ErrorBadRequest("skill_id required")))?;

    // TODO: Parse LoRA info from request
    let lora_info = lora_skill_distributor::LoRAInfo {
        lora_type: lora_skill_distributor::LoRAType::Software,
        version: "1.0".to_string(),
        capabilities: vec!["wasm".to_string()],
        attestation_support: false,
        secure_storage: false,
        memory_limit: 256 * 1024 * 1024,
        cpu_cores: 2,
        network_isolation: true,
        device_id: "test-device".to_string(),
        platform: "linux".to_string(),
    };

    match state
        .lora_skill_distributor
        .prepare_skill_for_lora_execution(skill_id, &lora_info)
        .await
    {
        Ok(package) => Ok(HttpResponse::Ok().json(serde_json::json!({
            "success": true,
            "package_hash": package.package_hash,
            "message": "Skill prepared for LoRA execution"
        }))),
        Err(e) => Ok(HttpResponse::BadRequest().json(serde_json::json!({
            "success": false,
            "error": e.to_string()
        }))),
    }
}

// Get IPFS status
async fn ipfs_status(state: web::Data<Arc<SharedState>>) -> Result<impl Responder, Error> {
    let (cache_items, cache_size) = state.ipfs_client.get_cache_stats().await;

    // Try to get node info
    let node_info = state.ipfs_client.get_node_info().await.ok();

    Ok(HttpResponse::Ok().json(serde_json::json!({
        "cache_items": cache_items,
        "cache_size_bytes": cache_size,
        "node_info": node_info
    })))
}

fn setup_logging() {
    let subscriber = fmt::Subscriber::builder()
        .with_max_level(tracing::Level::INFO)
        .finish();
    set_global_default(subscriber).expect("Failed to set default tracing subscriber");
}

#[actix_web::main]
async fn main() -> std::io::Result<()> {
    dotenv().ok();

    // Parse command line arguments
    let args = Args::parse();

    let rpc_endpoint =
        env::var("KNIRVCHAIN_RPC_ENDPOINT").unwrap_or_else(|_| String::from("127.0.0.1:8000"));
    let difficulty: u32 = env::var("BLOCK_DIFFICULTY").map_or(0, |v| v.parse().unwrap_or(0));
    let chain_id: u64 = env::var("KNIRVCHAIN_ID").map_or(1, |v| v.parse().unwrap_or(1)); //Default to chain id 1 if not set
    let block_time: u64 = env::var("BLOCK_TIME").map_or(5, |v| v.parse().unwrap_or(5)); //Block time of 5 seconds by default.
    let owner_private_key = generate_private_key();

    setup_logging();
    info!("[INFO] Starting the application");

    //Initialize the database
    let db_path = "./sledchain.db";
    let db = sled::open(db_path).expect("Failed to open database");
    let shared_db = Mutex::new(db);

    //Initialize the blockchain state, loading from the database or creating a genesis block
    let mut chain = block_on(load_chain(&shared_db)).expect("Error loading chain from database");
    if chain.is_empty() {
        println!("[INFO] No existing blockchain found. Creating Genesis block.");
        let genesis_block = create_genesis_block();
        chain.push(genesis_block.clone());
        block_on(save_block(&shared_db, &genesis_block))
            .expect("Could not save genesis block into the database");
    }

    let initial_supply = BigInt::from(1000);
    let max_supply = BigInt::from(10000);
    let nrn = NRN::new("MyNewToken".to_string(), "MNT".to_string(), initial_supply, max_supply, &owner_private_key)
      .expect("Error when creating token, please check the configurations are valid, specifically the private keys, or the data passed for initial and max supply");

    // Initialize smart contract engine
    let smart_contracts = SmartContractEngine::new(&owner_private_key)
        .expect("Failed to initialize smart contract engine");
    let smart_contracts_arc = Arc::new(Mutex::new(smart_contracts));

    // Load configuration
    let config = Config::load_from_file("config/blockchain.toml").unwrap_or_else(|_| {
        println!("Warning: Could not load config file, using defaults");
        Config::load_default()
    });

    let blockchain_config = config
        .to_blockchain_config()
        .expect("Failed to convert config to blockchain config");

    let blockchain_adapter = Arc::new(
        BlockchainAdapter::new(blockchain_config, smart_contracts_arc.clone())
            .expect("Failed to initialize blockchain adapter"),
    );

    // Initialize new infrastructure components
    let ipfs_client =
        Arc::new(ipfs_client::IpfsClient::new(None).expect("Failed to initialize IPFS client"));

    let multi_model_engine = Arc::new(Mutex::new(multi_model_engine::MultiModelEngine::new(
        ipfs_client.clone(),
    )));

    let model_registry = Arc::new(model_registry::EnhancedMultiModelRegistry::new(
        ipfs_client.clone(),
    ));

    let governance = Arc::new(governance::GovernanceSystem::new(None));

    // Initialize consensus and networking
    let tendermint_consensus = Arc::new(Mutex::new(
        tendermint_consensus::TendermintConsensus::new(format!("knirvoracle-{}", chain_id), None),
    ));

    let ibc_handler = Arc::new(ibc_handler::IBCHandler::new());

    // Initialize LoRA skill distributor
    let lora_skill_distributor = Arc::new(lora_skill_distributor::LoRASkillDistributor::new(
        ipfs_client.clone(),
    ));

    // Initialize cloud testing framework using environment variables
    let cloud_testing_framework = Arc::new(Mutex::new(Some(
        cloud_models::CloudModelTestingFramework::from_env(),
    )));

    let shared_state = Arc::new(SharedState {
        transaction_pool: Mutex::new(Vec::new()),
        blockchain: Mutex::new(chain),
        sled_db: shared_db,
        nrn: Mutex::new(nrn),
        smart_contracts: Mutex::new(
            SmartContractEngine::new(&owner_private_key).expect("Failed to create smart contracts"),
        ),
        blockchain_adapter,

        // New infrastructure components
        ipfs_client,
        multi_model_engine,
        model_registry,
        governance,

        // Consensus and networking
        tendermint_consensus,
        ibc_handler,

        // LoRA and skill distribution
        lora_skill_distributor,

        // Cloud model testing
        cloud_testing_framework,

        // Mining configuration
        mining_enabled: !args.disable_mining,
    });

    println!("[INFO] Starting server at http://{}", rpc_endpoint);
    println!("[INFO] KNIRVORACLE Chain ID: {}", chain_id);

    // Start auto-mining loop only if not disabled
    if !args.disable_mining {
        println!("[INFO] Auto-mining enabled");
        let state_clone = shared_state.clone();
        let difficulty_clone = difficulty;
        spawn(async move {
            loop {
                let mut pool = state_clone.transaction_pool.lock().await;
                if !pool.is_empty() {
                    let transactions: Vec<Transaction> = pool.drain(..).collect();
                    info!("[INFO] Transactions found in pool, creating block automatically");
                    let block_index: u64 = match block_on(get_latest_block(&state_clone.blockchain)) {
                        Ok(block) => block.index + 1,
                        Err(err) => {
                            error!("[ERROR] Failed to retrieve latest block {}", err);
                            0
                        }
                    };

                    let new_block = Block {
                        index: block_index,
                        timestamp: SystemTime::now()
                            .duration_since(UNIX_EPOCH)
                            .unwrap()
                            .as_secs(),
                        data: serde_json::to_string(&transactions).unwrap(),
                        previous_hash: String::new(),
                        nonce: 0,
                        hash: String::new(),
                    };
                    match add_block(&state_clone, new_block, difficulty_clone).await {
                        //Difficulty directly, with no deref
                        Ok(_) => {
                            info!("[INFO] Successfully added new block automatically with content")
                        }
                        Err(e) => error!(
                            "[ERROR] Error mining new block automatically with content: {}",
                            e
                        ),
                    };
                } else {
                    info!("[INFO] No transactions in the pool, skipping block creation.");
                }

                actix_web::rt::time::sleep(Duration::from_secs(block_time)).await;
            }
        });
    } else {
        println!("[INFO] Auto-mining disabled - blocks will only be created manually via API");
    }

    HttpServer::new(move || {
        App::new()
            .app_data(web::Data::new(shared_state.clone()))
            .app_data(web::Data::new(difficulty)) //Pass web data to main App structure.
            .app_data(web::Data::new(chain_id))
            .service(send_transaction)
            .service(new_wallet)
            .service(get_blocks)
            .service(nrn_mint)
            .service(nrn_transfer)
            .service(nrn_balance)
            .service(nrn_info)
            .service(register_llm)
            .service(register_skill)
            .service(invoke_skill)
            .service(register_llm_v2)
            .service(register_skill_v2)
            .service(invoke_skill_v2)
            .route(
                "/testnet/llm/validate",
                web::post().to(testnet::mock_llm_validate),
            )
            .route(
                "/testnet/skill/validate",
                web::post().to(testnet::mock_skill_validate),
            )
            .route("/testnet/status", web::get().to(testnet::testnet_status))
            .route("/health", web::get().to(testnet::health_check))
            // New API endpoints for enhanced functionality
            .route("/v3/models/list", web::get().to(list_models))
            .route("/v3/models/switch", web::post().to(switch_model))
            .route(
                "/v3/models/performance",
                web::get().to(get_model_performance),
            )
            .route("/v3/governance/proposals", web::get().to(list_proposals))
            .route("/v3/governance/vote", web::post().to(cast_vote))
            .route("/v3/consensus/status", web::get().to(consensus_status))
            .route("/v3/ibc/connections", web::get().to(ibc_connections))
            .route("/v3/lora/prepare", web::post().to(prepare_lora_skill))
            .route("/v3/ipfs/status", web::get().to(ipfs_status))
    })
    .bind(rpc_endpoint)?
    .run()
    .await
}
