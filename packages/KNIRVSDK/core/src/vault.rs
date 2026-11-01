//! Vault crypto for the controller wallet: BIP-39 mnemonics, SLIP-10 secp256k1
//! derivation, the `knirv-vault-v2.` Argon2id + AES-256-GCM envelope, and
//! signing with derived keys. This is the Rust implementation of what
//! `KNIRV_CORP/packages/controller/src/react-app/wasm/vaultCrypto.worker.ts`
//! used to do in JavaScript; `testvectors/vault/vectors.json` (captured from that
//! worker) pins bit-for-bit compatibility.
//!
//! Secrets (seed, private keys, derived keys, passwords, mnemonics) live in
//! [`Zeroizing`] containers and never appear in a response. The code is
//! deterministic and has no imports: the host supplies every random value
//! (entropy, salt, nonce, session id) and timestamp as input.

use crate::signing::{self, DirectSignRequest, MessageEnvelope};
use aes_gcm::{
    aead::{Aead, KeyInit},
    Aes256Gcm, Nonce,
};
use argon2::{Algorithm, Argon2, Params, Version};
use base64::{engine::general_purpose::STANDARD as BASE64, Engine as _};
use hmac::{Hmac, Mac};
use k256::{
    ecdsa::SigningKey,
    elliptic_curve::ff::PrimeField,
    FieldBytes, Scalar,
};
use ripemd::Ripemd160;
use serde::{Deserialize, Serialize};
use serde_json::{json, Value};
use sha2::{Digest, Sha256, Sha512};
use std::collections::HashMap;
use zeroize::{Zeroize, Zeroizing};

pub const ENVELOPE_PREFIX: &str = "knirv-vault-v2.";
/// The account path; `i` is replaced by the account index.
pub const HD_PATH: &str = "m/44'/118'/0'/0/i";
const SALT_BYTES: usize = 16;
const NONCE_BYTES: usize = 12;
const KEY_BYTES: usize = 32;
const OPS_LIMIT: u32 = 3;
const MEM_LIMIT_KIB: u32 = 64 * 1024;

type Result<T> = std::result::Result<T, String>;

#[derive(Serialize, Deserialize)]
struct Kdf {
    name: String,
    output_length: u32,
    ops_limit: u32,
    mem_limit_kib: u32,
}

/// Field order is the serialized order and must not change: it is part of the
/// stored format.
#[derive(Serialize, Deserialize)]
struct Envelope {
    version: u32,
    migration_version: u32,
    algorithm: String,
    kdf: Kdf,
    salt: String,
    nonce: String,
    ciphertext: String,
    created_at: String,
}

type Secret = Zeroizing<Vec<u8>>;

/// Open sessions, each holding only a seed.
#[derive(Default)]
pub struct Vault {
    sessions: HashMap<String, Secret>,
}

fn b64(bytes: &[u8]) -> String {
    BASE64.encode(bytes)
}

fn un_b64(value: &str, what: &str) -> Result<Vec<u8>> {
    BASE64.decode(value).map_err(|_| format!("{what} is not valid base64"))
}

fn field<'a>(payload: &'a Value, name: &str) -> Result<&'a str> {
    payload
        .get(name)
        .and_then(Value::as_str)
        .ok_or_else(|| format!("{name} must be a string"))
}

fn bytes_field(payload: &Value, name: &str, len: usize) -> Result<Vec<u8>> {
    let bytes = un_b64(field(payload, name)?, name)?;
    if bytes.len() != len {
        return Err(format!("{name} must be {len} bytes"));
    }
    Ok(bytes)
}

fn normalise_mnemonic(raw: &str) -> Zeroizing<String> {
    Zeroizing::new(raw.split_whitespace().collect::<Vec<_>>().join(" "))
}

fn seed_from_mnemonic(mnemonic: &str) -> Result<Secret> {
    let parsed = bip39::Mnemonic::parse_in_normalized(bip39::Language::English, mnemonic)
        .map_err(|e| match e {
            bip39::Error::InvalidChecksum => "Invalid mnemonic checksum".to_owned(),
            bip39::Error::UnknownWord(_) => "Mnemonic contains invalid word".to_owned(),
            bip39::Error::BadWordCount(_) => "Invalid mnemonic word count".to_owned(),
            other => other.to_string(),
        })?;
    Ok(Zeroizing::new(parsed.to_seed_normalized("").to_vec()))
}

// ---- SLIP-10 secp256k1 ------------------------------------------------------

fn hmac512(key: &[u8], data: &[&[u8]]) -> Zeroizing<[u8; 64]> {
    let mut mac = <Hmac<Sha512> as Mac>::new_from_slice(key).expect("HMAC takes any key length");
    for part in data {
        mac.update(part);
    }
    let mut out = Zeroizing::new([0u8; 64]);
    out.copy_from_slice(&mac.finalize().into_bytes());
    out
}

fn parse_path(path: &str) -> Result<Vec<(u32, bool)>> {
    path.split('/')
        .filter(|part| !part.is_empty() && *part != "m")
        .map(|part| {
            let hardened = part.ends_with('\'');
            let digits = part.trim_end_matches('\'');
            let value: u32 = digits.parse().map_err(|_| format!("Invalid HD path segment: {part}"))?;
            if value >= 1 << 31 {
                return Err(format!("Invalid HD path segment: {part}"));
            }
            Ok((value, hardened))
        })
        .collect()
}

/// SLIP-10 `derivePath` for secp256k1.
fn derive_private_key(seed: &[u8], path: &[(u32, bool)]) -> Result<Zeroizing<[u8; 32]>> {
    let mut node = hmac512(b"Bitcoin seed", &[seed]);
    for &(index, hardened) in path {
        let mut key = Zeroizing::new([0u8; 32]);
        key.copy_from_slice(&node[..32]);
        let child_index = if hardened { index | 1 << 31 } else { index }.to_be_bytes();
        let next = if hardened {
            hmac512(&node[32..], &[&[0], &key[..], &child_index])
        } else {
            let public = SigningKey::from_bytes(FieldBytes::from_slice(&key[..]))
                .map_err(|e| e.to_string())?
                .verifying_key()
                .to_encoded_point(true);
            hmac512(&node[32..], &[public.as_bytes(), &child_index])
        };
        let il = Option::<Scalar>::from(Scalar::from_repr(*FieldBytes::from_slice(&next[..32])))
            .ok_or("SLIP-10 derivation produced an invalid key")?;
        let parent = Option::<Scalar>::from(Scalar::from_repr(*FieldBytes::from_slice(&key[..])))
            .ok_or("SLIP-10 derivation produced an invalid key")?;
        let child = il + parent;
        if bool::from(child.is_zero()) {
            return Err("SLIP-10 derivation produced an invalid key".into());
        }
        let mut child_bytes = Zeroizing::new([0u8; 32]);
        child_bytes.copy_from_slice(&child.to_bytes());
        node = {
            let mut merged = Zeroizing::new([0u8; 64]);
            merged[..32].copy_from_slice(&child_bytes[..]);
            merged[32..].copy_from_slice(&next[32..]);
            merged
        };
    }
    let mut out = Zeroizing::new([0u8; 32]);
    out.copy_from_slice(&node[..32]);
    Ok(out)
}

fn account_key(seed: &[u8], index: u32) -> Result<Zeroizing<[u8; 32]>> {
    derive_private_key(seed, &parse_path(&HD_PATH.replace('i', &index.to_string()))?)
}

/// `knirv` bech32 address of a compressed secp256k1 public key.
pub fn address_from_public_key(public_key: &[u8]) -> Result<String> {
    let hash = Ripemd160::digest(Sha256::digest(public_key));
    let hrp = bech32::Hrp::parse("knirv").map_err(|e| e.to_string())?;
    bech32::encode::<bech32::Bech32>(hrp, &hash).map_err(|e| e.to_string())
}

fn public_key(private_key: &[u8; 32]) -> Result<Vec<u8>> {
    let key = SigningKey::from_bytes(FieldBytes::from_slice(private_key)).map_err(|e| e.to_string())?;
    Ok(key.verifying_key().to_encoded_point(true).as_bytes().to_vec())
}

// ---- the envelope -----------------------------------------------------------

fn kdf_key(password: &[u8], salt: &[u8]) -> Result<Zeroizing<[u8; KEY_BYTES]>> {
    let params = Params::new(MEM_LIMIT_KIB, OPS_LIMIT, 1, Some(KEY_BYTES)).map_err(|e| e.to_string())?;
    let mut key = Zeroizing::new([0u8; KEY_BYTES]);
    // Allocation failure here must surface as an error, never as weaker
    // parameters: the envelope pins them.
    Argon2::new(Algorithm::Argon2id, Version::V0x13, params)
        .hash_password_into(password, salt, &mut key[..])
        .map_err(|e| format!("key derivation failed: {e}"))?;
    Ok(key)
}

fn seal(payload: &[u8], password: &str, salt: &[u8], nonce: &[u8], created_at: &str) -> Result<String> {
    if password.is_empty() {
        return Err("Vault password is required".into());
    }
    let key = kdf_key(password.as_bytes(), salt)?;
    let cipher = Aes256Gcm::new_from_slice(&key[..]).map_err(|e| e.to_string())?;
    let ciphertext = cipher
        .encrypt(Nonce::from_slice(nonce), payload)
        .map_err(|_| "encryption failed".to_owned())?;
    let envelope = Envelope {
        version: 2,
        migration_version: 1,
        algorithm: "argon2id+aes-256-gcm".into(),
        kdf: Kdf {
            name: "argon2id".into(),
            output_length: KEY_BYTES as u32,
            ops_limit: OPS_LIMIT,
            mem_limit_kib: MEM_LIMIT_KIB,
        },
        salt: b64(salt),
        nonce: b64(nonce),
        ciphertext: b64(&ciphertext),
        created_at: created_at.to_owned(),
    };
    let json = serde_json::to_vec(&envelope).map_err(|e| e.to_string())?;
    Ok(format!("{ENVELOPE_PREFIX}{}", b64(&json)))
}

fn open(serialized: &str, password: &str) -> Result<Zeroizing<Vec<u8>>> {
    let body = serialized
        .strip_prefix(ENVELOPE_PREFIX)
        .ok_or("Invalid vault envelope")?;
    let envelope: Envelope = BASE64
        .decode(body)
        .ok()
        .and_then(|bytes| serde_json::from_slice(&bytes).ok())
        .ok_or("Invalid vault envelope")?;
    if envelope.version != 2
        || envelope.migration_version != 1
        || envelope.algorithm != "argon2id+aes-256-gcm"
        || envelope.kdf.name != "argon2id"
    {
        return Err("Unsupported vault envelope".into());
    }
    if envelope.kdf.output_length as usize != KEY_BYTES
        || envelope.kdf.ops_limit != OPS_LIMIT
        || envelope.kdf.mem_limit_kib != MEM_LIMIT_KIB
    {
        return Err("Unsupported vault KDF parameters".into());
    }
    let salt = BASE64.decode(&envelope.salt).map_err(|_| "Invalid vault envelope")?;
    let nonce = BASE64.decode(&envelope.nonce).map_err(|_| "Invalid vault envelope")?;
    if salt.len() != SALT_BYTES || nonce.len() != NONCE_BYTES {
        return Err("Invalid vault envelope parameters".into());
    }
    let ciphertext = BASE64.decode(&envelope.ciphertext).map_err(|_| "Invalid vault envelope")?;
    let key = kdf_key(password.as_bytes(), &salt)?;
    let cipher = Aes256Gcm::new_from_slice(&key[..]).map_err(|e| e.to_string())?;
    cipher
        .decrypt(Nonce::from_slice(&nonce), ciphertext.as_slice())
        .map(Zeroizing::new)
        .map_err(|_| "Incorrect password or unreadable vault".into())
}

fn extract_mnemonic(payload: &[u8]) -> Result<Zeroizing<String>> {
    fn visit(value: &Value) -> Option<String> {
        match value {
            Value::Object(map) => {
                if let Some(Value::String(m)) = map.get("mnemonic") {
                    return Some(m.clone());
                }
                map.values().find_map(visit)
            }
            _ => None,
        }
    }
    let mut parsed: Value = serde_json::from_slice(payload).map_err(|_| "Vault payload is malformed")?;
    let found = visit(&parsed).map(Zeroizing::new).map(|m| normalise_mnemonic(&m));
    scrub(&mut parsed);
    match found {
        Some(m) if !m.is_empty() => Ok(m),
        _ => Err("Legacy vault does not contain a recoverable mnemonic".into()),
    }
}

/// `{"mnemonic":"..."}` built so that no un-wiped copy of the phrase is left behind
/// (a `json!` value would clone it into an ordinary `String`).
fn mnemonic_payload(mnemonic: &str) -> Zeroizing<Vec<u8>> {
    let escaped = Zeroizing::new(serde_json::to_string(mnemonic).expect("a string serializes"));
    let mut out = Zeroizing::new(Vec::with_capacity(escaped.len() + 14));
    out.extend_from_slice(b"{\"mnemonic\":");
    out.extend_from_slice(escaped.as_bytes());
    out.push(b'}');
    out
}

/// Overwrites every string inside a parsed value.
pub fn scrub(value: &mut Value) {
    match value {
        Value::String(s) => s.zeroize(),
        Value::Array(items) => items.iter_mut().for_each(scrub),
        Value::Object(map) => map.values_mut().for_each(scrub),
        _ => {}
    }
}

// ---- sessions and operations ------------------------------------------------

impl Vault {
    pub fn new() -> Self {
        Self::default()
    }

    /// Wipes every session.
    pub fn zeroize_all(&mut self) {
        self.sessions.clear(); // each Secret zeroizes on drop
    }

    pub fn session_count(&self) -> usize {
        self.sessions.len()
    }

    fn open_session(&mut self, id: &str, seed: Secret) -> Result<()> {
        if id.is_empty() {
            return Err("session_id is required".into());
        }
        if self.sessions.contains_key(id) {
            return Err("session_id is already in use".into());
        }
        self.sessions.insert(id.to_owned(), seed);
        Ok(())
    }

    fn seed(&self, id: &str) -> Result<&Secret> {
        self.sessions.get(id).ok_or_else(|| "Session not found or expired".to_owned())
    }

    fn account(&self, id: &str, index: u32) -> Result<Value> {
        let key = account_key(self.seed(id)?, index)?;
        let public = public_key(&key)?;
        Ok(json!({ "address": address_from_public_key(&public)?, "public_key": b64(&public) }))
    }

    /// Opens a session from a mnemonic and seals it into a v2 envelope.
    fn establish(&mut self, payload: &Value, mnemonic: Zeroizing<String>, with_phrase: bool) -> Result<Value> {
        let password = Zeroizing::new(field(payload, "password")?.to_owned());
        let session_id = field(payload, "session_id")?.to_owned();
        let salt = bytes_field(payload, "salt", SALT_BYTES)?;
        let nonce = bytes_field(payload, "nonce", NONCE_BYTES)?;
        let created_at = field(payload, "created_at")?;
        let seed = seed_from_mnemonic(&mnemonic)?;
        let envelope = seal(&mnemonic_payload(&mnemonic), &password, &salt, &nonce, created_at)?;
        self.open_session(&session_id, seed)?;
        let account = match self.account(&session_id, 0) {
            Ok(account) => account,
            Err(error) => {
                self.sessions.remove(&session_id);
                return Err(error);
            }
        };
        let mut out = json!({ "session_id": session_id, "account": account, "envelope": envelope });
        if with_phrase {
            out["recovery_phrase"] = Value::String(mnemonic.to_string());
        }
        Ok(out)
    }

    /// Runs one operation. `payload` fields are documented in the ADR at the
    /// top of `crypto-core`; binary fields are base64.
    pub fn handle(&mut self, op: &str, payload: &Value) -> Result<Value> {
        match op {
            "vault.create" => {
                let entropy = un_b64(field(payload, "entropy")?, "entropy")?;
                let mnemonic = bip39::Mnemonic::from_entropy_in(bip39::Language::English, &entropy)
                    .map_err(|e| e.to_string())?;
                let phrase = Zeroizing::new(mnemonic.to_string());
                self.establish(payload, phrase, true)
            }
            "vault.import" => {
                let mnemonic = normalise_mnemonic(field(payload, "mnemonic")?);
                self.establish(payload, mnemonic, false)
            }
            "vault.unlock" => {
                let serialized = field(payload, "serialized")?;
                let password = Zeroizing::new(field(payload, "password")?.to_owned());
                let session_id = field(payload, "session_id")?.to_owned();
                let plaintext = open(serialized, &password)?;
                let mnemonic = extract_mnemonic(&plaintext)?;
                self.open_session(&session_id, seed_from_mnemonic(&mnemonic)?)?;
                match self.account(&session_id, 0) {
                    Ok(account) => Ok(json!({ "session_id": session_id, "account": account, "migrated": false })),
                    Err(error) => {
                        self.sessions.remove(&session_id);
                        Err(error)
                    }
                }
            }
            // The host decrypted a legacy vault (its format is retired and stays
            // in JavaScript) and hands over the mnemonic. The result is only
            // returned once the new envelope has been opened again successfully.
            "vault.migrate" => {
                let mnemonic = normalise_mnemonic(field(payload, "mnemonic")?);
                let password = Zeroizing::new(field(payload, "password")?.to_owned());
                let mut out = self.establish(payload, mnemonic.clone(), false)?;
                let session_id = field(payload, "session_id")?.to_owned();
                let envelope = out["envelope"].as_str().unwrap_or_default().to_owned();
                let verified = open(&envelope, &password)
                    .and_then(|plain| extract_mnemonic(&plain))
                    .map(|m| *m == *mnemonic);
                if !matches!(verified, Ok(true)) {
                    self.sessions.remove(&session_id);
                    return Err("Legacy vault migration verification failed".into());
                }
                out["migrated"] = Value::Bool(true);
                Ok(out)
            }
            "session.account" => {
                let index = payload.get("index").and_then(Value::as_u64).unwrap_or(0) as u32;
                self.account(field(payload, "session_id")?, index)
            }
            "session.sign_tx" => {
                let index = payload.get("index").and_then(Value::as_u64).unwrap_or(0) as u32;
                let request: DirectSignRequest = serde_json::from_value(payload.get("tx").cloned().unwrap_or_default())
                    .map_err(|e| format!("invalid transaction: {e}"))?;
                let key = account_key(self.seed(field(payload, "session_id")?)?, index)?;
                let signed = signing::sign_direct_transaction(&key, &request).map_err(|e| e.to_string())?;
                serde_json::to_value(signed).map_err(|e| e.to_string())
            }
            "session.sign_msg" => {
                let index = payload.get("index").and_then(Value::as_u64).unwrap_or(0) as u32;
                let envelope: MessageEnvelope =
                    serde_json::from_value(payload.get("envelope").cloned().unwrap_or_default())
                        .map_err(|e| format!("invalid envelope: {e}"))?;
                let key = account_key(self.seed(field(payload, "session_id")?)?, index)?;
                let signed = signing::sign_message_envelope(&key, &envelope).map_err(|e| e.to_string())?;
                serde_json::to_value(signed).map_err(|e| e.to_string())
            }
            "session.close" => {
                self.sessions.remove(field(payload, "session_id")?);
                Ok(json!({ "closed": true }))
            }
            "address.from_pubkey" => {
                let public = un_b64(field(payload, "pubkey")?, "pubkey")?;
                Ok(json!({ "address": address_from_public_key(&public)? }))
            }
            other => Err(format!("Unknown vault request: {other}")),
        }
    }
}

/// Known-answer check for `self_test`: cheap (no Argon2), exercises BIP-39,
/// SLIP-10, the address encoding and AES-GCM.
pub fn self_test() -> bool {
    let mnemonic = "abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about";
    let Ok(seed) = seed_from_mnemonic(mnemonic) else { return false };
    let Ok(key) = account_key(&seed, 0) else { return false };
    let Ok(public) = public_key(&key) else { return false };
    let Ok(address) = address_from_public_key(&public) else { return false };
    let Ok(cipher) = Aes256Gcm::new_from_slice(&[7u8; 32]) else { return false };
    let nonce = Nonce::from_slice(&[1u8; NONCE_BYTES]);
    let round = cipher
        .encrypt(nonce, b"knirv".as_ref())
        .and_then(|ct| cipher.decrypt(nonce, ct.as_ref()));
    let mut wiped = [1u8; 4];
    wiped.zeroize();
    address == SELF_TEST_ADDRESS && round.as_deref() == Ok(b"knirv".as_ref()) && wiped == [0; 4]
}

/// Address of account 0 for the "abandon … about" mnemonic (from the golden vectors).
const SELF_TEST_ADDRESS: &str = "knirv19rl4cm2hmr8afy4kldpxz3fka4jguq0amauk25";
