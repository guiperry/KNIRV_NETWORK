// Real post-quantum cryptography using Kyber-768 (KEM) and Dilithium-3 (signatures)
// Kyber-768 public keys: 1184 bytes, ciphertexts: 1088 bytes, shared secret: 32 bytes
// Dilithium-3 public keys: 1952 bytes, signatures: 3293 bytes

use aes_gcm::{Aes256Gcm, Key, Nonce, KeyInit};
use aes_gcm::aead::Aead;
use base64::{Engine as _, engine::general_purpose};
use chrono::{DateTime, Utc};
use parking_lot::RwLock;
use pqcrypto_dilithium::dilithium3;
use pqcrypto_kyber::kyber768;
use pqcrypto_traits::kem::{Ciphertext as KemCiphertext, PublicKey as KemPublicKey, SecretKey as KemSecretKey, SharedSecret};
use pqcrypto_traits::sign::{DetachedSignature, PublicKey as SignPublicKey, SecretKey as SignSecretKey};
use serde::{Deserialize, Serialize};
use sha2::{Digest, Sha256};
use std::collections::HashMap;

/// PQCKeyPair represents a complete PQC key pair with both Kyber-768 and Dilithium-3 keys.
/// This mirrors the Go `PQCKeyPair` in `internal/crypto/pqc/keys.go`.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PQCKeyPair {
    pub id: String,
    pub name: String,
    pub purpose: String, // encryption, signature, kex
    pub algorithm: String,
    pub created_at: DateTime<Utc>,
    pub expires_at: Option<DateTime<Utc>>,
    pub status: String, // active, rotated, revoked, expired

    // Kyber-768 keys for encryption (stored as bytes)
    #[serde(rename = "kyber_public_key")]
    pub kyber_public_key_bytes: Vec<u8>,
    #[serde(rename = "kyber_private_key", skip_serializing_if = "Vec::is_empty", default)]
    pub kyber_private_key_bytes: Vec<u8>,

    // Dilithium-3 keys for signatures (stored as bytes)
    #[serde(rename = "dilithium_public_key", skip_serializing_if = "Vec::is_empty", default)]
    pub dilithium_public_key_bytes: Vec<u8>,
    #[serde(rename = "dilithium_private_key", skip_serializing_if = "Vec::is_empty", default)]
    pub dilithium_private_key_bytes: Vec<u8>,
}

impl PQCKeyPair {
    /// Generate a new PQC key pair with Kyber-768 and Dilithium-3 keys.
    /// Mirrors Go's `GeneratePQCKeyPair`.
    pub fn generate(name: String, purpose: String) -> Result<Self, Box<dyn std::error::Error + Send + Sync>> {
        // Generate unique ID
        let mut id_bytes = [0u8; 16];
        getrandom::getrandom(&mut id_bytes)?;
        let id = hex::encode(id_bytes);

        // Generate Kyber-768 key pair
        let (kyber_pk, kyber_sk) = kyber768::keypair();

        // Generate Dilithium-3 key pair
        let (dilithium_pk, dilithium_sk) = dilithium3::keypair();

        Ok(PQCKeyPair {
            id,
            name,
            purpose,
            algorithm: "Kyber-768+Dilithium-3".to_string(),
            created_at: Utc::now(),
            expires_at: None,
            status: "active".to_string(),
            kyber_public_key_bytes: kyber_pk.as_bytes().to_vec(),
            kyber_private_key_bytes: kyber_sk.as_bytes().to_vec(),
            dilithium_public_key_bytes: dilithium_pk.as_bytes().to_vec(),
            dilithium_private_key_bytes: dilithium_sk.as_bytes().to_vec(),
        })
    }

    /// Deserialize a key pair from JSON bytes. Mirrors Go's `LoadPQCKeyPair`.
    pub fn load(data: &[u8]) -> Result<Self, Box<dyn std::error::Error + Send + Sync>> {
        Ok(serde_json::from_slice(data)?)
    }

    /// Serialize the key pair to JSON without private key bytes.
    pub fn marshal(&self) -> Result<Vec<u8>, Box<dyn std::error::Error + Send + Sync>> {
        let mut public_kp = self.clone();
        public_kp.kyber_private_key_bytes = vec![];
        public_kp.dilithium_private_key_bytes = vec![];
        Ok(serde_json::to_vec(&public_kp)?)
    }

    /// Serialize the key pair to JSON including private keys (for encrypted storage only).
    pub fn marshal_with_private_keys(&self) -> Result<Vec<u8>, Box<dyn std::error::Error + Send + Sync>> {
        Ok(serde_json::to_vec(self)?)
    }

    /// Encrypt plaintext using Kyber-768 KEM + AES-256-GCM.
    /// Mirrors Go's `KyberEncrypt`: encapsulate → shared secret → AES-GCM encrypt.
    /// Output: `kyber_ciphertext (1088 bytes) || nonce (12 bytes) || aes_ciphertext`
    pub fn encrypt(&self, plaintext: &[u8]) -> Result<Vec<u8>, Box<dyn std::error::Error + Send + Sync>> {
        if self.kyber_public_key_bytes.is_empty() {
            return Err("no Kyber public key available".into());
        }
        let pk = kyber768::PublicKey::from_bytes(&self.kyber_public_key_bytes)
            .map_err(|_| "failed to parse Kyber public key")?;

        let (shared_secret, kyber_ct) = kyber768::encapsulate(&pk);

        // Derive 32-byte AES key from shared secret via SHA-256
        let aes_key_bytes = Sha256::digest(shared_secret.as_bytes());
        let key = Key::<Aes256Gcm>::from_slice(&aes_key_bytes);
        let cipher = Aes256Gcm::new(key);

        let mut nonce_bytes = [0u8; 12];
        getrandom::getrandom(&mut nonce_bytes)?;
        let nonce = Nonce::from_slice(&nonce_bytes);

        let aes_ct = cipher
            .encrypt(nonce, plaintext)
            .map_err(|_| "AES-GCM encryption failed")?;

        // Concatenate: kyber_ct || nonce || aes_ct
        let kyber_ct_bytes = kyber_ct.as_bytes();
        let mut result = Vec::with_capacity(kyber_ct_bytes.len() + 12 + aes_ct.len());
        result.extend_from_slice(kyber_ct_bytes);
        result.extend_from_slice(&nonce_bytes);
        result.extend(aes_ct);

        Ok(result)
    }

    /// Decrypt ciphertext using Kyber-768 KEM + AES-256-GCM.
    /// Mirrors Go's `KyberDecrypt`.
    pub fn decrypt(&self, ciphertext: &[u8]) -> Result<Vec<u8>, Box<dyn std::error::Error + Send + Sync>> {
        if self.kyber_private_key_bytes.is_empty() {
            return Err("no Kyber private key available".into());
        }

        let kyber_ct_len = kyber768::ciphertext_bytes();
        if ciphertext.len() < kyber_ct_len + 12 {
            return Err("ciphertext too short".into());
        }

        let kyber_ct = kyber768::Ciphertext::from_bytes(&ciphertext[..kyber_ct_len])
            .map_err(|_| "failed to parse Kyber ciphertext")?;
        let sk = kyber768::SecretKey::from_bytes(&self.kyber_private_key_bytes)
            .map_err(|_| "failed to parse Kyber secret key")?;

        let shared_secret = kyber768::decapsulate(&kyber_ct, &sk);

        let aes_key_bytes = Sha256::digest(shared_secret.as_bytes());
        let key = Key::<Aes256Gcm>::from_slice(&aes_key_bytes);
        let cipher = Aes256Gcm::new(key);

        let nonce = Nonce::from_slice(&ciphertext[kyber_ct_len..kyber_ct_len + 12]);
        let aes_ct = &ciphertext[kyber_ct_len + 12..];

        cipher
            .decrypt(nonce, aes_ct)
            .map_err(|_| "AES-GCM decryption failed".into())
    }

    /// Sign a message using Dilithium-3. Returns the detached signature bytes.
    pub fn sign(&self, message: &[u8]) -> Result<Vec<u8>, Box<dyn std::error::Error + Send + Sync>> {
        if self.dilithium_private_key_bytes.is_empty() {
            return Err("no Dilithium private key available".into());
        }
        let sk = dilithium3::SecretKey::from_bytes(&self.dilithium_private_key_bytes)
            .map_err(|_| "failed to parse Dilithium secret key")?;
        let sig = dilithium3::detached_sign(message, &sk);
        Ok(sig.as_bytes().to_vec())
    }

    /// Verify a Dilithium-3 detached signature.
    pub fn verify(&self, message: &[u8], signature: &[u8]) -> bool {
        if self.dilithium_public_key_bytes.is_empty() {
            return false;
        }
        let pk = match dilithium3::PublicKey::from_bytes(&self.dilithium_public_key_bytes) {
            Ok(pk) => pk,
            Err(_) => return false,
        };
        let sig = match dilithium3::DetachedSignature::from_bytes(signature) {
            Ok(s) => s,
            Err(_) => return false,
        };
        dilithium3::verify_detached_signature(&sig, message, &pk).is_ok()
    }

    /// Check if the key pair has expired.
    pub fn is_expired(&self) -> bool {
        if let Some(expires_at) = self.expires_at {
            Utc::now() > expires_at
        } else {
            false
        }
    }

    /// Check if the key pair is active and not expired.
    pub fn is_active(&self) -> bool {
        self.status == "active" && !self.is_expired()
    }
}

/// EncryptionManager manages PQC encryption for sensitive data.
/// Mirrors Go's `EncryptionManager` in `internal/crypto/pqc/encryption.go`.
pub struct EncryptionManager {
    master_key: RwLock<Option<PQCKeyPair>>,
    key_cache: RwLock<HashMap<String, PQCKeyPair>>,
}

impl EncryptionManager {
    pub fn new() -> Self {
        EncryptionManager {
            master_key: RwLock::new(None),
            key_cache: RwLock::new(HashMap::new()),
        }
    }

    pub fn set_master_key(&self, key_pair: PQCKeyPair) {
        *self.master_key.write() = Some(key_pair);
    }

    pub fn get_master_key(&self) -> Option<PQCKeyPair> {
        self.master_key.read().clone()
    }

    pub fn encrypt_data(&self, plaintext: &[u8], key_id: &str) -> Result<String, Box<dyn std::error::Error + Send + Sync>> {
        let key_pair = self.resolve_key(key_id)?;

        if !key_pair.is_active() {
            return Err(format!("key {} is not active", key_id).into());
        }

        let ciphertext = key_pair.encrypt(plaintext)?;

        let payload = serde_json::json!({
            "key_id": key_id,
            "algorithm": "Kyber-768+AES-256-GCM",
            "ciphertext": general_purpose::STANDARD.encode(&ciphertext),
        });

        let payload_bytes = serde_json::to_vec(&payload)?;
        let signature = key_pair.sign(&payload_bytes)?;

        let encrypted = serde_json::json!({
            "payload": payload,
            "signature": general_purpose::STANDARD.encode(&signature),
        });

        let final_bytes = serde_json::to_vec(&encrypted)?;
        Ok(general_purpose::STANDARD.encode(final_bytes))
    }

    pub fn decrypt_data(&self, encrypted_data: &str) -> Result<Vec<u8>, Box<dyn std::error::Error + Send + Sync>> {
        let encrypted_bytes = general_purpose::STANDARD.decode(encrypted_data)?;
        let encrypted: serde_json::Value = serde_json::from_slice(&encrypted_bytes)?;

        let payload = &encrypted["payload"];
        let signature_b64 = encrypted["signature"].as_str().ok_or("missing signature")?;
        let signature = general_purpose::STANDARD.decode(signature_b64)?;

        let payload_bytes = serde_json::to_vec(payload)?;

        let key_id = payload["key_id"].as_str().ok_or("missing key_id")?;
        let ciphertext_b64 = payload["ciphertext"].as_str().ok_or("missing ciphertext")?;
        let ciphertext = general_purpose::STANDARD.decode(ciphertext_b64)?;

        let key_pair = self.resolve_key(key_id)?;

        if !key_pair.is_active() {
            return Err(format!("key {} is not active", key_id).into());
        }

        if !key_pair.verify(&payload_bytes, &signature) {
            return Err("signature verification failed".into());
        }

        key_pair.decrypt(&ciphertext)
    }

    pub fn cache_key(&self, key_id: String, key_pair: PQCKeyPair) {
        self.key_cache.write().insert(key_id, key_pair);
    }

    pub fn remove_key(&self, key_id: &str) {
        self.key_cache.write().remove(key_id);
    }

    pub fn generate_data_encryption_key(&self, name: String) -> Result<PQCKeyPair, Box<dyn std::error::Error + Send + Sync>> {
        let key_pair = PQCKeyPair::generate(name, "encryption".to_string())?;
        self.cache_key(key_pair.id.clone(), key_pair.clone());
        Ok(key_pair)
    }

    fn resolve_key(&self, key_id: &str) -> Result<PQCKeyPair, Box<dyn std::error::Error + Send + Sync>> {
        let cache = self.key_cache.read();
        if let Some(kp) = cache.get(key_id) {
            return Ok(kp.clone());
        }
        drop(cache);
        if let Some(master) = self.master_key.read().as_ref() {
            if master.id == key_id {
                return Ok(master.clone());
            }
        }
        Err(format!("key {} not found in cache", key_id).into())
    }
}

impl Default for EncryptionManager {
    fn default() -> Self {
        Self::new()
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_generate_key_pair() {
        let kp = PQCKeyPair::generate("test".to_string(), "encryption".to_string()).unwrap();
        assert_eq!(kp.algorithm, "Kyber-768+Dilithium-3");
        assert_eq!(kp.status, "active");
        assert!(!kp.kyber_public_key_bytes.is_empty());
        assert!(!kp.kyber_private_key_bytes.is_empty());
        assert!(!kp.dilithium_public_key_bytes.is_empty());
        assert!(!kp.dilithium_private_key_bytes.is_empty());
        // Kyber-768 public key is 1184 bytes
        assert_eq!(kp.kyber_public_key_bytes.len(), kyber768::public_key_bytes());
        // Dilithium-3 public key is 1952 bytes
        assert_eq!(kp.dilithium_public_key_bytes.len(), dilithium3::public_key_bytes());
    }

    #[test]
    fn test_encrypt_decrypt_roundtrip() {
        let kp = PQCKeyPair::generate("test".to_string(), "encryption".to_string()).unwrap();
        let plaintext = b"Hello from Kyber-768+AES-256-GCM";

        let ciphertext = kp.encrypt(plaintext).unwrap();
        assert_ne!(ciphertext.as_slice(), plaintext);

        let decrypted = kp.decrypt(&ciphertext).unwrap();
        assert_eq!(decrypted.as_slice(), plaintext);
    }

    #[test]
    fn test_sign_verify_roundtrip() {
        let kp = PQCKeyPair::generate("test".to_string(), "signature".to_string()).unwrap();
        let message = b"Sign this with Dilithium-3";

        let signature = kp.sign(message).unwrap();
        // Dilithium-3 detached signature is 3293 bytes
        assert_eq!(signature.len(), dilithium3::signature_bytes());

        assert!(kp.verify(message, &signature));
        assert!(!kp.verify(b"different message", &signature));
    }

    #[test]
    fn test_is_active() {
        let kp = PQCKeyPair::generate("test".to_string(), "encryption".to_string()).unwrap();
        assert!(kp.is_active());
        assert!(!kp.is_expired());
    }

    #[test]
    fn test_marshal_strips_private_keys() {
        let kp = PQCKeyPair::generate("test".to_string(), "encryption".to_string()).unwrap();
        let public_bytes = kp.marshal().unwrap();
        let loaded: PQCKeyPair = serde_json::from_slice(&public_bytes).unwrap();
        assert!(!loaded.kyber_public_key_bytes.is_empty());
        assert!(loaded.kyber_private_key_bytes.is_empty());
        assert!(loaded.dilithium_private_key_bytes.is_empty());
    }

    #[test]
    fn test_encryption_manager() {
        let manager = EncryptionManager::new();
        let kp = PQCKeyPair::generate("master".to_string(), "encryption".to_string()).unwrap();
        let key_id = kp.id.clone();
        manager.set_master_key(kp);

        let plaintext = b"Sensitive data encrypted via EncryptionManager";
        let encrypted = manager.encrypt_data(plaintext, &key_id).unwrap();
        let decrypted = manager.decrypt_data(&encrypted).unwrap();
        assert_eq!(decrypted.as_slice(), plaintext);
    }
}
