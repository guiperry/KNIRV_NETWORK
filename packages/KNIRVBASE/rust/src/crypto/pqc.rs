// Real post-quantum cryptography using Kyber-768 (KEM) and ML-DSA-65 (FIPS 204, signatures)
// Kyber-768 public keys: 1184 bytes, ciphertexts: 1088 bytes, shared secret: 32 bytes
// ML-DSA-65 public keys: 1952 bytes, signatures: 3309 bytes, key seeds: 32 bytes
// ML-DSA-65 is provided by the KNIRVSDK Rust core (`knirv_sdk::mldsa`); this crate holds no signing code of its own.

use aes_gcm::{Aes256Gcm, Key, Nonce, KeyInit};
use aes_gcm::aead::Aead;
use base64::{Engine as _, engine::general_purpose};
use chrono::{DateTime, Utc};
use parking_lot::RwLock;
use knirv_sdk::mldsa;
use pqcrypto_kyber::kyber768;
use pqcrypto_traits::kem::{Ciphertext as KemCiphertext, PublicKey as KemPublicKey, SecretKey as KemSecretKey, SharedSecret};
use serde::{Deserialize, Serialize};
use sha2::{Digest, Sha256};
use std::collections::HashMap;

/// PQCKeyPair represents a complete PQC key pair with both Kyber-768 and ML-DSA-65 keys.
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

    // ML-DSA-65 keys for signatures (stored as bytes). The private key is the
    // 32-byte FIPS 204 key seed; the 1952-byte public key is derived from it.
    #[serde(rename = "mldsa_public_key", skip_serializing_if = "Vec::is_empty", default)]
    pub mldsa_public_key_bytes: Vec<u8>,
    #[serde(rename = "mldsa_private_key", skip_serializing_if = "Vec::is_empty", default)]
    pub mldsa_private_key_bytes: Vec<u8>,
}

impl PQCKeyPair {
    /// Generate a new PQC key pair with Kyber-768 and ML-DSA-65 keys.
    /// Mirrors Go's `GeneratePQCKeyPair`.
    pub fn generate(name: String, purpose: String) -> Result<Self, Box<dyn std::error::Error + Send + Sync>> {
        // Generate unique ID
        let mut id_bytes = [0u8; 16];
        getrandom::getrandom(&mut id_bytes)?;
        let id = hex::encode(id_bytes);

        // Generate Kyber-768 key pair
        let (kyber_pk, kyber_sk) = kyber768::keypair();

        // Generate ML-DSA-65 key pair: a random 32-byte seed, public key derived by the core.
        let mut mldsa_seed = [0u8; mldsa::SEED_LEN];
        getrandom::getrandom(&mut mldsa_seed)?;
        let mldsa_pk = mldsa::ml_dsa65_public_key(&mldsa_seed)
            .map_err(|e| format!("ML-DSA-65 key generation failed: {}", e.message()))?;

        Ok(PQCKeyPair {
            id,
            name,
            purpose,
            algorithm: "Kyber-768+ML-DSA-65".to_string(),
            created_at: Utc::now(),
            expires_at: None,
            status: "active".to_string(),
            kyber_public_key_bytes: kyber_pk.as_bytes().to_vec(),
            kyber_private_key_bytes: kyber_sk.as_bytes().to_vec(),
            mldsa_public_key_bytes: mldsa_pk,
            mldsa_private_key_bytes: mldsa_seed.to_vec(),
        })
    }

    /// Deserialize a key pair from JSON bytes. Mirrors Go's `LoadPQCKeyPair`.
    ///
    /// TODO(PI_Q31): remove the `dilithium_*` fallback after the next release
    /// (same release as the Go and backend_server fallbacks). Pre-rename files
    /// carry `dilithium_*` ML-DSA names; they are read only when the `mldsa_*`
    /// field is empty or identical. Conflicting values are refused.
    pub fn load(data: &[u8]) -> Result<Self, Box<dyn std::error::Error + Send + Sync>> {
        let mut kp: PQCKeyPair = serde_json::from_slice(data)?;
        let legacy: LegacyMldsaFields = serde_json::from_slice(data)?;
        kp.mldsa_public_key_bytes =
            resolve_legacy_mldsa_field(kp.mldsa_public_key_bytes, legacy.dilithium_public_key, "public_key")?;
        kp.mldsa_private_key_bytes =
            resolve_legacy_mldsa_field(kp.mldsa_private_key_bytes, legacy.dilithium_private_key, "private_key")?;
        Ok(kp)
    }

    /// Serialize the key pair to JSON without private key bytes.
    pub fn marshal(&self) -> Result<Vec<u8>, Box<dyn std::error::Error + Send + Sync>> {
        let mut public_kp = self.clone();
        public_kp.kyber_private_key_bytes = vec![];
        public_kp.mldsa_private_key_bytes = vec![];
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

    /// Sign a message using ML-DSA-65 (pure, empty context). Returns the 3309-byte
    /// detached signature. Deterministic: the same key and message give the same bytes.
    pub fn sign(&self, message: &[u8]) -> Result<Vec<u8>, Box<dyn std::error::Error + Send + Sync>> {
        if self.mldsa_private_key_bytes.is_empty() {
            return Err("no ML-DSA-65 private key available".into());
        }
        let sig = mldsa::ml_dsa65_sign(&self.mldsa_private_key_bytes, message, &[])
            .map_err(|e| format!("ML-DSA-65 sign failed: {}", e.message()))?;
        Ok(sig)
    }

    /// Verify an ML-DSA-65 detached signature (pure, empty context).
    pub fn verify(&self, message: &[u8], signature: &[u8]) -> bool {
        if self.mldsa_public_key_bytes.is_empty() {
            return false;
        }
        mldsa::ml_dsa65_verify(&self.mldsa_public_key_bytes, message, &[], signature)
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

/// Read-only view of the pre-rename ML-DSA JSON names. Never written.
/// TODO(PI_Q31): remove together with the fallback in `PQCKeyPair::load`.
#[derive(Deserialize, Default)]
struct LegacyMldsaFields {
    #[serde(default)]
    dilithium_public_key: Vec<u8>,
    #[serde(default)]
    dilithium_private_key: Vec<u8>,
}

/// Use the legacy `dilithium_*` value only when the `mldsa_*` value is empty;
/// refuse if both are present and differ.
fn resolve_legacy_mldsa_field(
    current: Vec<u8>,
    legacy: Vec<u8>,
    suffix: &str,
) -> Result<Vec<u8>, Box<dyn std::error::Error + Send + Sync>> {
    if legacy.is_empty() {
        return Ok(current);
    }
    if current.is_empty() {
        return Ok(legacy);
    }
    if current != legacy {
        return Err(format!(
            "key file has conflicting mldsa_{} and dilithium_{}: refusing to load",
            suffix, suffix
        )
        .into());
    }
    Ok(current)
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
        assert_eq!(kp.algorithm, "Kyber-768+ML-DSA-65");
        assert_eq!(kp.status, "active");
        assert!(!kp.kyber_public_key_bytes.is_empty());
        assert!(!kp.kyber_private_key_bytes.is_empty());
        assert!(!kp.mldsa_public_key_bytes.is_empty());
        assert!(!kp.mldsa_private_key_bytes.is_empty());
        // Kyber-768 public key is 1184 bytes
        assert_eq!(kp.kyber_public_key_bytes.len(), kyber768::public_key_bytes());
        // ML-DSA-65 public key is 1952 bytes (FIPS 204 Table 2)
        assert_eq!(kp.mldsa_public_key_bytes.len(), mldsa::PUBLIC_KEY_LEN);
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
        let message = b"Sign this with ML-DSA-65";

        let signature = kp.sign(message).unwrap();
        // ML-DSA-65 detached signature is 3309 bytes (FIPS 204 Table 2)
        assert_eq!(signature.len(), mldsa::SIGNATURE_LEN);

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
        assert!(loaded.mldsa_private_key_bytes.is_empty());
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

    #[test]
    fn test_verify_fails_closed_without_mldsa_public_key() {
        // A valid signature from an intact key pair must not verify once the
        // ML-DSA public key is missing: verify() has to fail closed, not accept.
        let kp = PQCKeyPair::generate("test".to_string(), "signature".to_string()).unwrap();
        let message = b"fail closed";
        let signature = kp.sign(message).unwrap();
        assert!(kp.verify(message, &signature));

        let mut stripped = kp.clone();
        stripped.mldsa_public_key_bytes = vec![];
        assert!(!stripped.verify(message, &signature));
        assert!(!stripped.verify(b"anything at all", &[0u8; mldsa::SIGNATURE_LEN]));
        assert!(!stripped.verify(message, &[]));
    }

    #[test]
    fn test_sign_fails_without_mldsa_private_key() {
        let kp = PQCKeyPair::generate("test".to_string(), "signature".to_string()).unwrap();
        let mut stripped = kp.clone();
        stripped.mldsa_private_key_bytes = vec![];
        assert!(stripped.sign(b"message").is_err());
        // The stripped key pair must not be able to produce a signature the full key verifies.
        assert!(kp.sign(b"message").is_ok());
    }

    #[test]
    fn test_legacy_dilithium_key_json_loads_into_mldsa_fields() {
        // Key JSON written before the Dilithium -> ML-DSA-65 rename carries
        // `dilithium_*` ML-DSA fields and no `mldsa_*` fields. It must load the
        // values into the ML-DSA fields (PI_Q31 read fallback).
        let kp = PQCKeyPair::generate("legacy".to_string(), "signature".to_string()).unwrap();
        let mut value = serde_json::to_value(&kp).unwrap();
        let obj = value.as_object_mut().unwrap();
        obj.remove("mldsa_public_key");
        obj.remove("mldsa_private_key");
        obj.insert("dilithium_public_key".to_string(), serde_json::json!(kp.mldsa_public_key_bytes));
        obj.insert("dilithium_private_key".to_string(), serde_json::json!(kp.mldsa_private_key_bytes));
        let legacy_bytes = serde_json::to_vec(&value).unwrap();

        let loaded = PQCKeyPair::load(&legacy_bytes).unwrap();
        assert_eq!(loaded.id, kp.id);
        assert_eq!(loaded.mldsa_public_key_bytes, kp.mldsa_public_key_bytes);
        assert_eq!(loaded.mldsa_private_key_bytes, kp.mldsa_private_key_bytes);
        // Kyber fields survive the load.
        assert_eq!(loaded.kyber_public_key_bytes, kp.kyber_public_key_bytes);
        // The loaded ML-DSA key signs and verifies like the original.
        let sig = loaded.sign(b"m").unwrap();
        assert!(kp.verify(b"m", &sig));
    }

    #[test]
    fn test_legacy_dilithium_conflicting_values_are_refused() {
        // Both names present with different values: refuse, do not guess.
        let kp = PQCKeyPair::generate("conflict".to_string(), "signature".to_string()).unwrap();
        let other = PQCKeyPair::generate("other".to_string(), "signature".to_string()).unwrap();
        let mut value = serde_json::to_value(&kp).unwrap();
        let obj = value.as_object_mut().unwrap();
        obj.insert("dilithium_private_key".to_string(), serde_json::json!(other.mldsa_private_key_bytes));
        let bytes = serde_json::to_vec(&value).unwrap();
        assert!(PQCKeyPair::load(&bytes).is_err());
    }

    #[test]
    fn test_legacy_and_mldsa_identical_loads() {
        // Both present and identical: loads.
        let kp = PQCKeyPair::generate("equal".to_string(), "signature".to_string()).unwrap();
        let mut value = serde_json::to_value(&kp).unwrap();
        let obj = value.as_object_mut().unwrap();
        obj.insert("dilithium_public_key".to_string(), serde_json::json!(kp.mldsa_public_key_bytes));
        obj.insert("dilithium_private_key".to_string(), serde_json::json!(kp.mldsa_private_key_bytes));
        let bytes = serde_json::to_vec(&value).unwrap();
        let loaded = PQCKeyPair::load(&bytes).unwrap();
        assert_eq!(loaded.mldsa_private_key_bytes, kp.mldsa_private_key_bytes);
    }

    #[test]
    fn test_writer_emits_mldsa_names_only() {
        let kp = PQCKeyPair::generate("writer".to_string(), "signature".to_string()).unwrap();
        let s = String::from_utf8(kp.marshal_with_private_keys().unwrap()).unwrap();
        assert!(s.contains("\"mldsa_public_key\""));
        assert!(s.contains("\"mldsa_private_key\""));
        assert!(!s.contains("dilithium_"));
    }
}
