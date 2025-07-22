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
//use typenum::U32;
//use crate::transaction::Transaction; // Import the Transaction struct from main.rs

#[derive(Debug, Clone, PartialEq, Eq, Hash, Serialize)]
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

#[derive(Debug, Serialize)]
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


