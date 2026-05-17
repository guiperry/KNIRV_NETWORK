use crate::nrn_token::*;
use crate::transaction::Transaction;
use actix_web::rt::spawn;
use actix_web::{get, post, web, App, Error, HttpResponse, HttpServer, Responder};
use dotenv::dotenv;
use futures::executor::block_on;
use num_bigint::BigInt;
use rand::Rng;
use serde::{Deserialize, Serialize};
use sha2::{Digest, Sha256};
use sled::Db;
use std::collections::HashMap;
use std::env;
use std::path::PathBuf;
use std::sync::Arc;
use std::time::{Duration, SystemTime, UNIX_EPOCH};
use tokio::sync::Mutex;
use tracing::{error, info, subscriber::set_global_default};
use tracing_subscriber::fmt;

mod nrn_token;
// Structs
pub mod transaction {
    use serde::{Deserialize, Serialize};
    #[derive(Debug, Serialize, Deserialize, Clone)]
    pub struct Transaction {
        pub data: String,
        pub signature: String,
        pub transaction_hash: Option<String>,
    }
}
//#[derive(Debug, Serialize, Deserialize, Clone)]
//struct Transaction {
// data: String,
//  signature: String,
//}

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

#[derive(Debug, Serialize, Deserialize, Clone)]
struct ValidationRecord {
    id: String,
    record_type: String,
    payload: serde_json::Value,
    transaction_hash: String,
    block_height: u64,
    created_at: u64,
}

#[derive(Debug, Serialize, Deserialize, Clone)]
struct CommitValidationRequest {
    validation_id: String,
    #[serde(default)]
    node_id: Option<String>,
    #[serde(default)]
    result_type: Option<String>,
    #[serde(default)]
    payload: Option<serde_json::Value>,
}

#[derive(Debug, Serialize, Deserialize, Clone)]
struct PolicyCommitRequest {
    name: String,
    #[serde(default)]
    r#type: Option<String>,
    #[serde(default)]
    target_dve: Option<String>,
    #[serde(default)]
    rules: Option<serde_json::Value>,
    #[serde(default)]
    priority: Option<i64>,
    enabled: bool,
}

#[derive(Debug, Serialize, Deserialize, Clone)]
struct EvidenceAnchorRequest {
    #[serde(default)]
    evidence_id: Option<String>,
    #[serde(default)]
    evidence_type: Option<String>,
    #[serde(default)]
    node_id: Option<String>,
    #[serde(default)]
    validation_id: Option<String>,
    #[serde(default)]
    payload: Option<serde_json::Value>,
    #[serde(default)]
    signature: Option<String>,
    #[serde(default)]
    algorithm: Option<String>,
    #[serde(default)]
    public_key: Option<String>,
}

// In memory transaction pool
#[derive(Debug)]
struct SharedState {
    transaction_pool: Mutex<Vec<Transaction>>,
    blockchain: Mutex<Vec<Block>>,
    sled_db: Mutex<Db>,
    nrn: Mutex<NRN>,
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

fn now_unix_secs() -> u64 {
    SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .unwrap()
        .as_secs()
}

fn record_key(id: &str) -> String {
    format!("record:{}", id)
}

fn dve_uri_key(dve_id: &str) -> String {
    format!("dve_uri:{}", dve_id)
}

fn dve_wallet_key(wallet: &str) -> String {
    format!("dve_wallet:{}", wallet)
}

async fn save_record(db: &Mutex<Db>, record: &ValidationRecord) -> Result<(), String> {
    let database = db.lock().await;
    let key = record_key(&record.id);
    let value = serde_json::to_vec(record).map_err(|_| "Error serializing validation record")?;
    database
        .insert(key, value)
        .map_err(|_| "Error saving validation record to database")?;
    Ok(())
}

async fn get_record_by_id(db: &Mutex<Db>, id: &str) -> Result<Option<ValidationRecord>, String> {
    let database = db.lock().await;
    let key = record_key(id);
    match database.get(key).map_err(|_| "Error loading validation record")? {
        Some(bytes) => {
            let record: ValidationRecord =
                serde_json::from_slice(&bytes).map_err(|_| "Error deserializing validation record")?;
            Ok(Some(record))
        }
        None => Ok(None),
    }
}

async fn list_records(db: &Mutex<Db>, filter_type: Option<&str>) -> Result<Vec<ValidationRecord>, String> {
    let database = db.lock().await;
    let mut records = Vec::new();

    for item in database.scan_prefix("record:") {
        let (_key, value) = item.map_err(|_| "Error reading validation records")?;
        let record: ValidationRecord =
            serde_json::from_slice(&value).map_err(|_| "Error deserializing validation record")?;
        if filter_type.map(|value| value == record.record_type).unwrap_or(true) {
            records.push(record);
        }
    }

    records.sort_by_key(|record| record.created_at);
    Ok(records)
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

    //Save the block into the database
    save_block(&state.sled_db, &new_block).await?;

    let mut chain = state.blockchain.lock().await;
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
    let response = enqueue_transaction(state.clone(), txn).await?;
    let difficulty_clone = **difficulty; //Dereference at source
    let state_clone = state.clone();

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

    Ok(response)
}

async fn enqueue_transaction(
    state: web::Data<Arc<SharedState>>,
    txn: Transaction,
) -> Result<HttpResponse, Error> {
    info!("[INFO] Received transaction, current pool: {:?}", txn);
    let transaction_hash = bytes_to_hex(&Sha256::digest(
        serde_json::to_string(&txn).unwrap().as_bytes(),
    ));

    let mut pool = state.transaction_pool.lock().await;
    pool.push(txn.clone());

    Ok(HttpResponse::Created().json(BlockchainResponse {
        message: "Transaction submitted successfully (mining async)".to_string(),
        data: Some(serde_json::to_string(&txn).unwrap()),
        transaction_hash: Some(transaction_hash),
    }))
}

async fn commit_record(
    state: web::Data<Arc<SharedState>>,
    record_type: &str,
    record_id: String,
    payload: serde_json::Value,
    difficulty: u32,
) -> Result<HttpResponse, Error> {
    let created_at = now_unix_secs();
    let transaction_payload = serde_json::json!({
        "record_type": record_type,
        "record_id": record_id,
        "payload": payload,
        "created_at": created_at,
    });

    let transaction = Transaction {
        data: transaction_payload.to_string(),
        signature: record_type.to_string(),
        transaction_hash: None,
    };

    let transaction_hash = bytes_to_hex(&Sha256::digest(
        serde_json::to_string(&transaction).unwrap().as_bytes(),
    ));

    {
        let mut pool = state.transaction_pool.lock().await;
        pool.push(transaction);
    }

    let block_height = match get_latest_block(&state.blockchain).await {
        Ok(block) => block.index + 1,
        Err(_) => 0,
    };

    let new_block = Block {
        index: block_height,
        timestamp: created_at,
        data: transaction_payload.to_string(),
        previous_hash: String::new(),
        nonce: 0,
        hash: String::new(),
    };

    add_block(state.as_ref().as_ref(), new_block, difficulty)
        .await
        .map_err(actix_web::error::ErrorInternalServerError)?;

    {
        let mut pool = state.transaction_pool.lock().await;
        pool.clear();
    }

    let record = ValidationRecord {
        id: record_id,
        record_type: record_type.to_string(),
        payload,
        transaction_hash: transaction_hash.clone(),
        block_height,
        created_at,
    };

    save_record(&state.sled_db, &record)
        .await
        .map_err(actix_web::error::ErrorInternalServerError)?;

    Ok(HttpResponse::Created().json(serde_json::json!({
        "record": record,
        "transaction_hash": transaction_hash,
    })))
}

#[get("/health")]
async fn health(state: web::Data<Arc<SharedState>>) -> Result<impl Responder, Error> {
    let height = match get_latest_block(&state.blockchain).await {
        Ok(block) => block.index,
        Err(_) => 0,
    };
    let pending = state.transaction_pool.lock().await.len();

    Ok(HttpResponse::Ok().json(serde_json::json!({
        "status": "ok",
        "healthy": true,
        "height": height,
        "pending_transactions": pending,
    })))
}

#[get("/chain/height")]
async fn chain_height(state: web::Data<Arc<SharedState>>) -> Result<impl Responder, Error> {
    let height = match get_latest_block(&state.blockchain).await {
        Ok(block) => block.index,
        Err(_) => 0,
    };

    Ok(HttpResponse::Ok().json(serde_json::json!({ "height": height })))
}

#[post("/validation/commit")]
async fn validation_commit(
    state: web::Data<Arc<SharedState>>,
    request: web::Json<CommitValidationRequest>,
    difficulty: web::Data<u32>,
) -> Result<impl Responder, Error> {
    let req = request.into_inner();
    let record_id = req.validation_id.clone();
    commit_record(
        state,
        "validation",
        record_id,
        serde_json::to_value(req).map_err(actix_web::error::ErrorBadRequest)?,
        **difficulty,
    )
    .await
}

#[post("/policy/commit")]
async fn policy_commit(
    state: web::Data<Arc<SharedState>>,
    request: web::Json<PolicyCommitRequest>,
    difficulty: web::Data<u32>,
) -> Result<impl Responder, Error> {
    let req = request.into_inner();
    let record_id = req.name.clone();
    commit_record(
        state,
        "policy",
        record_id,
        serde_json::to_value(req).map_err(actix_web::error::ErrorBadRequest)?,
        **difficulty,
    )
    .await
}

#[post("/evidence/anchor")]
async fn evidence_anchor(
    state: web::Data<Arc<SharedState>>,
    request: web::Json<EvidenceAnchorRequest>,
    difficulty: web::Data<u32>,
) -> Result<impl Responder, Error> {
    let req = request.into_inner();
    let record_id = req
        .evidence_id
        .clone()
        .unwrap_or_else(|| format!("evidence-{}", now_unix_secs()));
    commit_record(
        state,
        "evidence",
        record_id,
        serde_json::to_value(req).map_err(actix_web::error::ErrorBadRequest)?,
        **difficulty,
    )
    .await
}

#[get("/records/{id}")]
async fn get_record(
    state: web::Data<Arc<SharedState>>,
    path: web::Path<String>,
) -> Result<impl Responder, Error> {
    match get_record_by_id(&state.sled_db, &path.into_inner())
        .await
        .map_err(actix_web::error::ErrorInternalServerError)?
    {
        Some(record) => Ok(HttpResponse::Ok().json(record)),
        None => Ok(HttpResponse::NotFound().json(serde_json::json!({
            "message": "Record not found"
        }))),
    }
}

#[get("/records")]
async fn get_records(
    state: web::Data<Arc<SharedState>>,
    query: web::Query<HashMap<String, String>>,
) -> Result<impl Responder, Error> {
    let filter_type = query.get("type").map(|value| value.as_str());
    let records = list_records(&state.sled_db, filter_type)
        .await
        .map_err(actix_web::error::ErrorInternalServerError)?;
    Ok(HttpResponse::Ok().json(records))
}

// --- DVE URI handlers ---

#[derive(Debug, Serialize, Deserialize, Clone)]
struct DVEURIRecord {
    dve_id: String,
    full_uri: String,
    wallet_address: String,
    created_at: i64,
    tx_hash: String,
}

#[post("/dve_uri/submit")]
async fn dve_uri_submit(
    state: web::Data<Arc<SharedState>>,
    request: web::Json<DVEURIRecord>,
) -> Result<impl Responder, Error> {
    let req = request.into_inner();
    let tx_hash = bytes_to_hex(&Sha256::digest(
        format!("{}:{}:{}:{}", req.dve_id, req.full_uri, req.wallet_address, req.created_at).as_bytes(),
    ));

    let record = DVEURIRecord {
        tx_hash: tx_hash.clone(),
        ..req
    };

    let value = serde_json::to_vec(&record).map_err(|_| {
        actix_web::error::ErrorInternalServerError("Failed to serialize DVE URI record")
    })?;

    {
        let db = state.sled_db.lock().await;
        db.insert(dve_uri_key(&record.dve_id), value.as_slice())
            .map_err(|e| actix_web::error::ErrorInternalServerError(format!("DB insert error: {}", e)))?;
        // Also index by wallet
        let wallet_key = dve_wallet_key(&record.wallet_address);
        let mut wallet_ids: Vec<String> = db
            .get(&wallet_key)
            .map_err(|e| actix_web::error::ErrorInternalServerError(format!("DB read error: {}", e)))?
            .map(|v| serde_json::from_slice::<Vec<String>>(&v).unwrap_or_default())
            .unwrap_or_default();
        wallet_ids.push(record.dve_id.clone());
        let wallet_value = serde_json::to_vec(&wallet_ids)
            .map_err(|_| actix_web::error::ErrorInternalServerError("Failed to serialize wallet index"))?;
        db.insert(wallet_key, wallet_value.as_slice())
            .map_err(|e| actix_web::error::ErrorInternalServerError(format!("DB wallet index error: {}", e)))?;
    }

    Ok(HttpResponse::Created().json(serde_json::json!({
        "record": record,
        "transaction_hash": tx_hash,
    })))
}

#[get("/dve_uri/{dve_id}")]
async fn dve_uri_get(
    state: web::Data<Arc<SharedState>>,
    path: web::Path<String>,
) -> Result<impl Responder, Error> {
    let dve_id = path.into_inner();
    let db = state.sled_db.lock().await;
    match db
        .get(dve_uri_key(&dve_id))
        .map_err(|e| actix_web::error::ErrorInternalServerError(format!("DB error: {}", e)))?
    {
        Some(bytes) => {
            let record: DVEURIRecord = serde_json::from_slice(&bytes)
                .map_err(|_| actix_web::error::ErrorInternalServerError("Failed to deserialize DVE URI record"))?;
            Ok(HttpResponse::Ok().json(record))
        }
        None => Ok(HttpResponse::NotFound().json(serde_json::json!({
            "message": "DVE URI not found"
        }))),
    }
}

#[get("/dve_uri/by_wallet/{wallet}")]
async fn dve_uri_by_wallet(
    state: web::Data<Arc<SharedState>>,
    path: web::Path<String>,
) -> Result<impl Responder, Error> {
    let wallet = path.into_inner();
    let db = state.sled_db.lock().await;
    let dve_ids: Vec<String> = db
        .get(dve_wallet_key(&wallet))
        .map_err(|e| actix_web::error::ErrorInternalServerError(format!("DB error: {}", e)))?
        .map(|v| serde_json::from_slice::<Vec<String>>(&v).unwrap_or_default())
        .unwrap_or_default();

    let mut records = Vec::new();
    for id in &dve_ids {
        if let Some(bytes) = db
            .get(dve_uri_key(id))
            .map_err(|e| actix_web::error::ErrorInternalServerError(format!("DB error: {}", e)))?
        {
            if let Ok(record) = serde_json::from_slice::<DVEURIRecord>(&bytes) {
                records.push(record);
            }
        }
    }

    Ok(HttpResponse::Ok().json(records))
}

// Handler to create a new wallet (private key + address)
#[get("/wallets/new")]
async fn new_wallet() -> Result<impl Responder, Error> {
    let private_key = generate_private_key();
    let address = get_address_from_private_key(&private_key).map_err(|e| {
        actix_web::error::ErrorInternalServerError(format!(
            "Failed to derive address from private key: {}",
            e
        ))
    })?;

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
            info!("[INFO] tokens transfered, posting transfer tx.");
            Ok(HttpResponse::Created().json(BlockchainResponse {
                message: "Tokens transferred".to_string(),
                data: Some(format!("{:?}", tx)),
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

// Handler to retrieve the NRN info.
#[get("/nrn/info")]
async fn nrn_info(state: web::Data<Arc<SharedState>>) -> Result<impl Responder, Error> {
    let nrn = state.nrn.lock().await;

    Ok(HttpResponse::Ok().json(BlockchainResponse {
        message: "NRN Token information".to_string(),
        data: Some(format!("{:?}", *nrn)),
        transaction_hash: None,
    }))
}

// Handler to retrieve the whole blockchain.
#[get("/blocks")]
async fn get_blocks(state: web::Data<Arc<SharedState>>) -> Result<impl Responder, Error> {
    let blockchain = state.blockchain.lock().await;
    Ok(HttpResponse::Ok().json(blockchain.clone()))
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
    let rpc_endpoint =
        env::var("KNIRVCHAIN_RPC_ENDPOINT").unwrap_or_else(|_| String::from("127.0.0.1:8000"));
    let difficulty: u32 = env::var("BLOCK_DIFFICULTY").map_or(0, |v| v.parse().unwrap_or(0));
    let chain_id: u64 = env::var("KNIRVCHAIN_ID").map_or(1, |v| v.parse().unwrap_or(1)); //Default to chain id 1 if not set
    let block_time: u64 = env::var("BLOCK_TIME").map_or(5, |v| v.parse().unwrap_or(5)); //Block time of 5 seconds by default.
    let data_path = env::var("DATA_PATH").unwrap_or_else(|_| String::from("."));
    let owner_private_key = generate_private_key();

    setup_logging();
    info!("[INFO] Starting the application");

    //Initialize the database
    let mut db_path = PathBuf::from(data_path);
    db_path.push("validation_chain.db");
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
    let nrn = NRN::new("Neuron".to_string(), "NRN".to_string(), initial_supply, max_supply, &owner_private_key)
      .expect("Error when creating token, please check the configurations are valid, specifically the private keys, or the data passed for initial and max supply");

    let shared_state = Arc::new(SharedState {
        transaction_pool: Mutex::new(Vec::new()),
        blockchain: Mutex::new(chain),
        sled_db: shared_db,
        nrn: Mutex::new(nrn),
    });

    println!("[INFO] Starting server at http://{}", rpc_endpoint);
    println!("[INFO] KNIRVCHAIN Chain ID: {}", chain_id);
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

    HttpServer::new(move || {
        App::new()
            .app_data(web::Data::new(shared_state.clone()))
            .app_data(web::Data::new(difficulty)) //Pass web data to main App structure.
            .app_data(web::Data::new(chain_id))
            .service(health)
            .service(chain_height)
            .service(validation_commit)
            .service(policy_commit)
            .service(evidence_anchor)
            .service(get_record)
            .service(get_records)
            .service(send_transaction)
            .service(get_blocks)
            .service(nrn_mint)
            .service(nrn_transfer)
            .service(nrn_balance)
            .service(nrn_info)
            .service(dve_uri_submit)
            .service(dve_uri_get)
            .service(dve_uri_by_wallet)
    })
    .bind(rpc_endpoint)?
    .run()
    .await
}
