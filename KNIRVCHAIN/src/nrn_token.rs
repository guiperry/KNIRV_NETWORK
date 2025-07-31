use anyhow::{anyhow, Result};
//use generic_array::GenericArray;
use hex;
use k256::ecdsa::{signature::Signer, SigningKey};
use k256::elliptic_curve::sec1::EncodedPoint; // Import remains the same
use k256::FieldBytes;
use num_bigint::BigInt;
use rand::rngs::OsRng;
use serde::{Deserialize, Serialize};
use sha3::{Digest, Keccak256};
use std::collections::HashMap;
use std::time::{SystemTime, UNIX_EPOCH};
//use typenum::U32;
//use crate::transaction::Transaction; // Import the Transaction struct from main.rs

#[derive(Debug, Clone, PartialEq, Eq, Hash, Serialize, Deserialize)]
pub struct Address([u8; 20]);

impl std::fmt::Display for Address {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        write!(f, "0x{}", hex::encode(self.0))
    }
}

// Define Transaction struct here to avoid circular imports
#[derive(Debug, Serialize, Deserialize, Clone, PartialEq)]
pub struct Transaction {
    pub data: String,
    pub signature: String,
    pub transaction_hash: Option<String>,
}

// LLM Registry structures
#[derive(Debug, Serialize, Deserialize, Clone)]
pub struct LLMMetadata {
    pub name: String,
    pub version: String,
    pub model_hash: String,
    pub capabilities: Vec<String>,
    pub owner: Address,
    pub registration_fee: BigInt,
    pub usage_fee: BigInt,
    pub ipfs_hash: Option<String>,
    pub validation_status: ValidationStatus,
    pub registered_at: u64,
}

#[derive(Debug, Serialize, Deserialize, Clone)]
pub enum ValidationStatus {
    Pending,
    Validated,
    Rejected,
}

#[derive(Debug, Serialize, Deserialize, Clone)]
pub struct LLMRegistry {
    pub models: HashMap<String, LLMMetadata>,
    pub validators: Vec<Address>,
}

// Skill Registry structures
#[derive(Debug, Serialize, Deserialize, Clone)]
pub struct SkillMetadata {
    pub name: String,
    pub skill_type: String,
    pub capabilities: Vec<String>,
    pub requirements: HashMap<String, String>,
    pub owner: Address,
    pub usage_fee: BigInt,
    pub validation_status: ValidationStatus,
    pub performance_metrics: PerformanceMetrics,
    pub registered_at: u64,
}

#[derive(Debug, Serialize, Deserialize, Clone)]
pub struct PerformanceMetrics {
    pub success_rate: f64,
    pub average_latency: f64,
    pub total_invocations: u64,
    pub last_updated: u64,
}

#[derive(Debug, Serialize, Deserialize, Clone)]
pub struct SkillRegistry {
    pub skills: HashMap<String, SkillMetadata>,
    pub skill_invocations: HashMap<String, Vec<SkillInvocation>>,
}

#[derive(Debug, Serialize, Deserialize, Clone)]
pub struct SkillInvocation {
    pub skill_id: String,
    pub user: Address,
    pub amount_burned: BigInt,
    pub timestamp: u64,
    pub success: bool,
    pub result_hash: Option<String>,
}

// Helper function to convert hex string to Address
pub fn hex_to_address(s: &str) -> Result<Address> {
    if !s.starts_with("0x") || s.len() != 42 {
        return Err(anyhow!("Invalid address format"));
    }
    let bytes = hex::decode(&s[2..]).map_err(|e| anyhow!("Failed to decode hex address: {}", e))?;
    let mut addr = [0u8; 20];
    addr.copy_from_slice(&bytes);
    Ok(Address(addr))
}

// Helper function to get address from public key
fn get_address_from_public_key(public_key_bytes: &EncodedPoint<k256::Secp256k1>) -> Address {
    // Specify the generic type
    let mut hasher = Keccak256::new();
    hasher.update(public_key_bytes.as_bytes());
    let hash = hasher.finalize();
    let mut address_bytes = [0u8; 20];
    address_bytes.copy_from_slice(&hash[12..]);
    Address(address_bytes)
}

#[derive(Debug, Serialize, Deserialize, Clone)]
pub struct NRN {
    name: String,
    symbol: String,
    total_supply: BigInt,
    max_supply: BigInt,
    balances: HashMap<Address, BigInt>,
    owner: Address,
}

impl NRN {
    // NewNRN creates a new NRN token
    pub fn new(
        name: String,
        symbol: String,
        initial_supply: BigInt,
        max_supply: BigInt,
        owner_private_key: &str,
    ) -> Result<Self> {
        let private_key_bytes = hex::decode(owner_private_key)
            .map_err(|e| anyhow!("Failed to decode owner private key: {}", e))?;
        if private_key_bytes.len() != 32 {
            // Check the length!
            return Err(anyhow!("Invalid private key length. Must be 32 bytes."));
        }
        let signing_key = SigningKey::from_bytes(private_key_bytes.as_slice().into())
            .map_err(|e| anyhow!("Failed to parse owner private key: {}", e))?;

        let public_key = signing_key.verifying_key();
        let public_key_bytes = public_key.to_encoded_point(false);
        let owner_address = get_address_from_public_key(&public_key_bytes); // Pass the EncodedPoint

        let mut balances = HashMap::new();
        balances.insert(owner_address.clone(), initial_supply.clone());

        if initial_supply > max_supply {
            return Err(anyhow!("Initial supply cannot exceed max supply"));
        }

        Ok(NRN {
            name,
            symbol,
            total_supply: initial_supply,
            max_supply,
            balances,
            owner: owner_address,
        })
    }

    // Mint function to mint new tokens
    pub fn mint(&mut self, from_private_key: &str, to: Address, amount: &BigInt) -> Result<bool> {
        if amount <= &BigInt::from(0) {
            return Err(anyhow!("Mint amount must be greater than zero"));
        }
        let from_address = get_address_from_private_key(from_private_key)?;
        if from_address != self.owner {
            return Err(anyhow!("Only the owner can mint tokens"));
        }

        if (&self.total_supply + amount) > self.max_supply {
            return Err(anyhow!("Minting would exceed max supply"));
        }

        self.balances
            .entry(to)
            .and_modify(|balance| *balance += amount)
            .or_insert_with(|| amount.clone());

        self.total_supply += amount;
        Ok(true)
    }

    // GetBalance gets balance of a given address
    pub fn get_balance(&self, address: &Address) -> BigInt {
        self.balances
            .get(address)
            .cloned()
            .unwrap_or_else(|| BigInt::from(0))
    }

    pub fn get_total_supply(&self) -> BigInt {
        self.total_supply.clone()
    }

    pub fn get_owner(&self) -> Address {
        self.owner.clone()
    }

    pub fn transfer(
        &mut self,
        from_private_key: &str,
        to: Address,
        amount: &BigInt,
    ) -> Result<Transaction> {
        if amount <= &BigInt::from(0) {
            return Err(anyhow!("Transfer amount must be greater than zero"));
        }
        let from_address = get_address_from_private_key(from_private_key)?;

        if from_address == to {
            return Err(anyhow!("Cannot transfer to the same address"));
        }

        let from_balance = self
            .balances
            .get(&from_address)
            .cloned()
            .unwrap_or_else(|| BigInt::from(0));
        if from_balance < *amount {
            println!("Insufficient balance");
            return Err(anyhow!("Insufficient balance"));
        }

        let transaction_data = format!("Transfer {} from {} to {}", amount, from_address, to);

        let private_key_bytes = hex::decode(from_private_key)?;
        let signing_key = SigningKey::from_bytes(FieldBytes::from_slice(&private_key_bytes))
            .map_err(|e| anyhow!("Failed to parse private key: {}", e))?;
        let signature: k256::ecdsa::Signature = signing_key.sign(transaction_data.as_bytes());
        let signature_hex = hex::encode(signature.to_der());

        self.balances
            .entry(from_address)
            .and_modify(|b| *b -= amount);
        self.balances
            .entry(to)
            .and_modify(|b| *b += amount)
            .or_insert_with(|| amount.clone());

        let transaction = Transaction {
            data: transaction_data,
            signature: signature_hex,
            transaction_hash: None,
        };
        Ok(transaction)
    }

    // Burn tokens for skill invocation
    pub fn burn_for_skill(
        &mut self,
        from_private_key: &str,
        skill_id: &str,
        amount: &BigInt,
    ) -> Result<Transaction> {
        if amount <= &BigInt::from(0) {
            return Err(anyhow!("Burn amount must be greater than zero"));
        }

        let from_address = get_address_from_private_key(from_private_key)?;
        let from_balance = self
            .balances
            .get(&from_address)
            .cloned()
            .unwrap_or_else(|| BigInt::from(0));

        if from_balance < *amount {
            return Err(anyhow!("Insufficient balance for burn"));
        }

        let transaction_data = format!("Burn {} NRN for skill {} by {}", amount, skill_id, from_address);

        let private_key_bytes = hex::decode(from_private_key)?;
        let signing_key = SigningKey::from_bytes(FieldBytes::from_slice(&private_key_bytes))
            .map_err(|e| anyhow!("Failed to parse private key: {}", e))?;
        let signature: k256::ecdsa::Signature = signing_key.sign(transaction_data.as_bytes());
        let signature_hex = hex::encode(signature.to_der());

        // Burn tokens by reducing balance and total supply
        self.balances
            .entry(from_address)
            .and_modify(|b| *b -= amount);
        self.total_supply -= amount;

        let transaction = Transaction {
            data: transaction_data,
            signature: signature_hex,
            transaction_hash: None,
        };
        Ok(transaction)
    }

    // Mint reward tokens for network participation
    pub fn mint_reward(
        &mut self,
        to: Address,
        amount: &BigInt,
        reason: &str,
    ) -> Result<()> {
        if amount <= &BigInt::from(0) {
            return Err(anyhow!("Mint amount must be greater than zero"));
        }

        if (&self.total_supply + amount) > self.max_supply {
            return Err(anyhow!("Minting would exceed max supply"));
        }

        let to_clone = to.clone();
        self.balances
            .entry(to)
            .and_modify(|balance| *balance += amount)
            .or_insert_with(|| amount.clone());

        self.total_supply += amount;

        println!("Minted {} NRN to {} for: {}", amount, to_clone, reason);
        Ok(())
    }
}

pub fn generate_private_key() -> String {
    let mut rng = OsRng;
    let signing_key = SigningKey::random(&mut rng);
    hex::encode(signing_key.to_bytes())
}

pub fn get_address_from_private_key(private_key: &str) -> Result<Address> {
    let private_key_bytes =
        hex::decode(private_key).map_err(|e| anyhow!("Failed to decode private key: {}", e))?;
    let signing_key = SigningKey::from_bytes(private_key_bytes.as_slice().into())
        .map_err(|e| anyhow!("Failed to parse private key: {}", e))?;
    let public_key = signing_key.verifying_key();
    let public_key_bytes = public_key.to_encoded_point(false);
    Ok(get_address_from_public_key(&public_key_bytes)) // Pass the EncodedPoint
}

impl LLMRegistry {
    pub fn new() -> Self {
        Self {
            models: HashMap::new(),
            validators: Vec::new(),
        }
    }

    pub fn register_llm(
        &mut self,
        metadata: LLMMetadata,
        model_data: &[u8],
    ) -> Result<String> {
        // Calculate model hash
        let mut hasher = sha3::Keccak256::new();
        hasher.update(model_data);
        let model_hash = hex::encode(hasher.finalize());

        let mut llm_metadata = metadata;
        llm_metadata.model_hash = model_hash.clone();
        llm_metadata.validation_status = ValidationStatus::Pending;
        llm_metadata.registered_at = SystemTime::now()
            .duration_since(UNIX_EPOCH)
            .unwrap()
            .as_secs();

        self.models.insert(model_hash.clone(), llm_metadata);
        Ok(model_hash)
    }

    pub fn validate_llm(&mut self, model_hash: &str, is_valid: bool) -> Result<()> {
        if let Some(metadata) = self.models.get_mut(model_hash) {
            metadata.validation_status = if is_valid {
                ValidationStatus::Validated
            } else {
                ValidationStatus::Rejected
            };
            Ok(())
        } else {
            Err(anyhow!("Model not found"))
        }
    }

    pub fn get_llm(&self, model_hash: &str) -> Option<&LLMMetadata> {
        self.models.get(model_hash)
    }

    #[allow(dead_code)]
    pub fn list_validated_llms(&self) -> Vec<&LLMMetadata> {
        self.models
            .values()
            .filter(|metadata| matches!(metadata.validation_status, ValidationStatus::Validated))
            .collect()
    }
}

impl SkillRegistry {
    pub fn new() -> Self {
        Self {
            skills: HashMap::new(),
            skill_invocations: HashMap::new(),
        }
    }

    pub fn register_skill(&mut self, skill: SkillMetadata) -> Result<String> {
        let skill_id = format!("skill_{}_{}", skill.skill_type, skill.registered_at);
        self.skills.insert(skill_id.clone(), skill);
        self.skill_invocations.insert(skill_id.clone(), Vec::new());
        Ok(skill_id)
    }

    pub fn invoke_skill(
        &mut self,
        skill_id: &str,
        user: Address,
        amount_burned: BigInt,
        success: bool,
        result_hash: Option<String>,
    ) -> Result<()> {
        let invocation = SkillInvocation {
            skill_id: skill_id.to_string(),
            user,
            amount_burned,
            timestamp: SystemTime::now()
                .duration_since(UNIX_EPOCH)
                .unwrap()
                .as_secs(),
            success,
            result_hash,
        };

        self.skill_invocations
            .entry(skill_id.to_string())
            .or_insert_with(Vec::new)
            .push(invocation);

        // Update performance metrics
        if let Some(skill) = self.skills.get_mut(skill_id) {
            skill.performance_metrics.total_invocations += 1;
            if success {
                let total = skill.performance_metrics.total_invocations as f64;
                let current_success = skill.performance_metrics.success_rate * (total - 1.0);
                skill.performance_metrics.success_rate = (current_success + 1.0) / total;
            } else {
                let total = skill.performance_metrics.total_invocations as f64;
                let current_success = skill.performance_metrics.success_rate * (total - 1.0);
                skill.performance_metrics.success_rate = current_success / total;
            }
            skill.performance_metrics.last_updated = SystemTime::now()
                .duration_since(UNIX_EPOCH)
                .unwrap()
                .as_secs();
        }

        Ok(())
    }

    pub fn get_skill(&self, skill_id: &str) -> Option<&SkillMetadata> {
        self.skills.get(skill_id)
    }

    #[allow(dead_code)]
    pub fn list_skills_by_type(&self, skill_type: &str) -> Vec<&SkillMetadata> {
        self.skills
            .values()
            .filter(|skill| skill.skill_type == skill_type)
            .collect()
    }

    #[allow(dead_code)]
    pub fn get_skill_invocations(&self, skill_id: &str) -> Option<&Vec<SkillInvocation>> {
        self.skill_invocations.get(skill_id)
    }
}


