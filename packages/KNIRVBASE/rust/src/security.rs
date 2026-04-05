use aes_gcm::{
    aead::{Aead, KeyInit, OsRng},
    Aes256Gcm, Nonce,
};
use base64::{engine::general_purpose::STANDARD as BASE64, Engine as _};
use pbkdf2::pbkdf2_hmac_array;
use rand::RngCore;
use sha2::{Digest, Sha256};
use std::collections::HashMap;

pub struct MemoryEncryption {
    iterations: u32,
    key_length: usize,
}

impl MemoryEncryption {
    pub fn new() -> Self {
        Self {
            iterations: 100_000,
            key_length: 32,
        }
    }

    pub fn derive_key(&self, user_secret: &str, salt: &[u8]) -> Vec<u8> {
        pbkdf2_hmac_array::<Sha256, 32>(user_secret.as_bytes(), salt, self.iterations).to_vec()
    }

    pub fn encrypt_memory(&self, data: &[u8], key: &[u8]) -> Result<Vec<u8>, String> {
        let cipher = Aes256Gcm::new_from_slice(key)
            .map_err(|e| format!("failed to create cipher: {}", e))?;

        let mut nonce_bytes = [0u8; 12];
        OsRng.fill_bytes(&mut nonce_bytes);
        let nonce = Nonce::from_slice(&nonce_bytes);

        let ciphertext = cipher
            .encrypt(nonce, data)
            .map_err(|e| format!("encryption failed: {}", e))?;

        let mut result = nonce_bytes.to_vec();
        result.extend(ciphertext);

        Ok(result)
    }

    pub fn decrypt_memory(&self, encrypted: &[u8], key: &[u8]) -> Result<Vec<u8>, String> {
        if encrypted.len() < 12 {
            return Err("ciphertext too short".to_string());
        }

        let cipher = Aes256Gcm::new_from_slice(key)
            .map_err(|e| format!("failed to create cipher: {}", e))?;

        let nonce = Nonce::from_slice(&encrypted[..12]);
        let ciphertext = &encrypted[12..];

        cipher
            .decrypt(nonce, ciphertext)
            .map_err(|e| format!("decryption failed: {}", e))
    }

    pub fn generate_salt(&self) -> Vec<u8> {
        let mut salt = vec![0u8; 16];
        OsRng.fill_bytes(&mut salt);
        salt
    }

    pub fn encode_key(&self, key: &[u8]) -> String {
        BASE64.encode(key)
    }

    pub fn decode_key(&self, encoded: &str) -> Result<Vec<u8>, String> {
        BASE64
            .decode(encoded)
            .map_err(|e| format!("failed to decode key: {}", e))
    }
}

impl Default for MemoryEncryption {
    fn default() -> Self {
        Self::new()
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_encrypt_decrypt_roundtrip() {
        let encryption = MemoryEncryption::new();

        let key = encryption.derive_key("password", b"testsalt12345678");
        let plaintext = b"Hello, this is a secret message!";

        let encrypted = encryption.encrypt_memory(plaintext, &key).unwrap();
        assert_ne!(encrypted.as_slice(), plaintext);

        let decrypted = encryption.decrypt_memory(&encrypted, &key).unwrap();
        assert_eq!(decrypted.as_slice(), plaintext);
    }

    #[test]
    fn test_derive_key_deterministic() {
        let encryption = MemoryEncryption::new();

        let key1 = encryption.derive_key("password", b"testsalt12345678");
        let key2 = encryption.derive_key("password", b"testsalt12345678");

        assert_eq!(key1, key2);

        let key3 = encryption.derive_key("different", b"testsalt12345678");
        assert_ne!(key1, key3);
    }

    #[test]
    fn test_salt_generation() {
        let encryption = MemoryEncryption::new();

        let salt1 = encryption.generate_salt();
        let salt2 = encryption.generate_salt();

        assert_eq!(salt1.len(), 16);
        assert_ne!(salt1, salt2);
    }

    #[test]
    fn test_key_encoding() {
        let encryption = MemoryEncryption::new();
        let key = vec![1u8, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16];

        let encoded = encryption.encode_key(&key);
        let decoded = encryption.decode_key(&encoded).unwrap();

        assert_eq!(decoded, key);
    }
}
