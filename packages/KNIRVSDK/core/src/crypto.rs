use crate::error::{Error, Result};
use aes::cipher::generic_array::GenericArray;
use aes::cipher::{BlockDecrypt, BlockEncrypt, KeyInit};
use aes::Aes256;
use base64::Engine;
use hex;
use pbkdf2::pbkdf2_hmac;
use rand::Rng;
use sha2::{Digest, Sha256};

/// Local cryptography service exposed by [`crate::KnirvClient`]. It performs
/// no network I/O; the methods delegate to the module's stable helper API.
#[derive(Clone, Debug, Default)]
pub struct CryptoService;

impl CryptoService {
    pub fn sha256_hex(&self, data: &[u8]) -> String {
        sha256_hex(data)
    }
    pub fn sha256_string(&self, data: &str) -> String {
        sha256_string(data)
    }
    pub fn encrypt_aes(&self, data: &str, password: &str) -> Result<String> {
        encrypt_aes(data, password)
    }
    pub fn decrypt_aes(&self, encrypted_data: &str, password: &str) -> Result<String> {
        decrypt_aes(encrypted_data, password)
    }
    pub fn generate_salt(&self, length: usize) -> String {
        generate_salt(length)
    }
    pub fn generate_iv(&self, length: usize) -> String {
        generate_iv(length)
    }
}

pub fn generate_salt(length: usize) -> String {
    let mut buf = vec![0u8; length];
    for byte in &mut buf {
        *byte = rand::thread_rng().gen();
    }
    hex::encode(buf)
}

pub fn generate_iv(length: usize) -> String {
    let mut buf = vec![0u8; length];
    for byte in &mut buf {
        *byte = rand::thread_rng().gen();
    }
    hex::encode(buf)
}

pub fn sha256_hex(data: &[u8]) -> String {
    hex::encode(Sha256::digest(data))
}

pub fn sha256_string(data: &str) -> String {
    sha256_hex(data.as_bytes())
}

pub fn pbkdf2_key(
    password: &str,
    salt: &[u8],
    iterations: u32,
    key_length: usize,
) -> Result<Vec<u8>> {
    let mut key = vec![0u8; key_length];
    pbkdf2_hmac::<Sha256>(password.as_bytes(), salt, iterations, &mut key);
    Ok(key)
}

pub fn encrypt_aes(data: &str, password: &str) -> Result<String> {
    if data.is_empty() || password.is_empty() {
        return Err(Error::Validation("data and password are required".into()));
    }
    let salt = rand::thread_rng().gen::<[u8; 16]>();
    let iv = rand::thread_rng().gen::<[u8; 16]>();
    let key = pbkdf2_key(password, &salt, 10000, 32)?;

    let cipher = Aes256::new(GenericArray::from_slice(&key));

    let mut plaintext = data.as_bytes().to_vec();
    pad_pkcs7(&mut plaintext, 16);

    let mut prev = iv.to_vec();
    let mut ciphertext = Vec::with_capacity(plaintext.len());

    for chunk in plaintext.chunks(16) {
        let block = xor(chunk, &prev);
        let mut block_array = GenericArray::clone_from_slice(&block);
        cipher.encrypt_block(&mut block_array);
        let encrypted = block_array.to_vec();
        prev = encrypted.clone();
        ciphertext.extend_from_slice(&encrypted);
    }

    let mut combined = salt.to_vec();
    combined.extend(iv);
    combined.extend(ciphertext);
    Ok(base64::engine::general_purpose::STANDARD.encode(combined))
}

pub fn decrypt_aes(encrypted_data: &str, password: &str) -> Result<String> {
    if encrypted_data.is_empty() || password.is_empty() {
        return Err(Error::Validation(
            "encrypted data and password are required".into(),
        ));
    }
    let combined = base64::engine::general_purpose::STANDARD
        .decode(encrypted_data)
        .map_err(|e| Error::Crypto(format!("base64 decode: {e}")))?;
    if combined.len() < 32 {
        return Err(Error::Validation("invalid encrypted data length".into()));
    }
    let salt = &combined[..16];
    let iv = &combined[16..32];
    let ciphertext = &combined[32..];
    let key = pbkdf2_key(password, salt, 10000, 32)?;

    let cipher = Aes256::new(GenericArray::from_slice(&key));

    let mut prev = iv.to_vec();
    let mut plaintext = Vec::with_capacity(ciphertext.len());

    for chunk in ciphertext.chunks(16) {
        let mut block_array = GenericArray::clone_from_slice(chunk);
        cipher.decrypt_block(&mut block_array);
        let mut decrypted = block_array.to_vec();
        decrypted = xor(&decrypted, &prev);
        prev = chunk.to_vec();
        plaintext.extend_from_slice(&decrypted);
    }

    unpad_pkcs7(&mut plaintext);
    String::from_utf8(plaintext).map_err(|e| Error::Crypto(format!("utf8: {e}")))
}

fn pad_pkcs7(buf: &mut Vec<u8>, block_size: usize) {
    let pad = block_size - (buf.len() % block_size);
    buf.extend(std::iter::repeat(pad as u8).take(pad));
}

fn unpad_pkcs7(buf: &mut Vec<u8>) {
    if let Some(&pad) = buf.last() {
        if pad > 0 && pad <= 16 && pad as usize <= buf.len() {
            buf.truncate(buf.len() - pad as usize);
        }
    }
}

fn xor(a: &[u8], b: &[u8]) -> Vec<u8> {
    a.iter().zip(b.iter()).map(|(x, y)| x ^ y).collect()
}
