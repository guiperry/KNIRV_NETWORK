//! Encrypted node key files (`root.key`, `boot.key`): discovery, decryption and
//! the field-number → named-field mapping.
//!
//! This is the single implementation of the key-file format. Go and TypeScript
//! reach it through the `secrets.*` binding operations; services keep
//! only thin shims. The envelope is
//! `EncryptedRootKeyFile{encrypted_content=1, salt=2}`, the key is scrypt
//! (N=32768, r=8, p=1, 32 bytes) over password+salt, and the cipher is
//! AES-256-GCM with the 12-byte nonce prepended to the ciphertext.
use crate::error::{Error, Result};
use aes_gcm::aead::{Aead, KeyInit};
use aes_gcm::{Aes256Gcm, Nonce};
use serde::{Deserialize, Serialize};
use std::collections::BTreeMap;
use std::path::{Path, PathBuf};

const SCRYPT_LOG_N: u8 = 15; // N = 32768
const SCRYPT_R: u32 = 8;
const SCRYPT_P: u32 = 1;
const KEY_LEN: usize = 32;
const NONCE_LEN: usize = 12;
const ENVELOPE_ENCRYPTED_CONTENT: u64 = 1;
const ENVELOPE_SALT: u64 = 2;

fn bad(message: impl Into<String>) -> Error {
    Error::Validation(message.into())
}

fn read_varint(data: &[u8], mut offset: usize) -> Result<(u64, usize)> {
    let mut value = 0u64;
    let mut shift = 0u32;
    while shift < 64 {
        let byte = *data.get(offset).ok_or_else(|| bad("truncated varint"))?;
        offset += 1;
        value |= u64::from(byte & 0x7f) << shift;
        if byte < 0x80 {
            return Ok((value, offset));
        }
        shift += 7;
    }
    Err(bad("varint overflow"))
}

fn length_delimited(data: &[u8], offset: usize, field: u64, what: &str) -> Result<(usize, usize)> {
    let (length, start) = read_varint(data, offset)?;
    let end = usize::try_from(length)
        .ok()
        .and_then(|l| start.checked_add(l))
        .filter(|end| *end <= data.len())
        .ok_or_else(|| bad(format!("truncated field {field}{what}")))?;
    Ok((start, end))
}

/// Extracts `(encrypted_content, salt)` from the outer envelope.
pub fn parse_envelope(data: &[u8]) -> Result<(Vec<u8>, Vec<u8>)> {
    let (mut content, mut salt) = (None, None);
    let mut i = 0;
    while i < data.len() {
        let (tag, next) = read_varint(data, i)?;
        let field = tag >> 3;
        if tag & 0x7 != 2 {
            return Err(bad(format!(
                "unexpected wire type {} at field {field} in envelope",
                tag & 0x7
            )));
        }
        let (start, end) = length_delimited(data, next, field, " in envelope")?;
        match field {
            ENVELOPE_ENCRYPTED_CONTENT => content = Some(data[start..end].to_vec()),
            ENVELOPE_SALT => salt = Some(data[start..end].to_vec()),
            _ => {}
        }
        i = end;
    }
    Ok((
        content.ok_or_else(|| bad("key envelope missing encrypted_content"))?,
        salt.ok_or_else(|| bad("key envelope missing salt"))?,
    ))
}

/// The key-derivation settings a key file calls for. The core owns the format
/// and these parameters; a host that can run scrypt much faster than a sandbox
/// can (a native `scrypt`) derives the key itself and hands it back through
/// [`decrypt_with_key`], so no format knowledge lives outside the core.
#[derive(Clone, Debug, PartialEq, Eq, Serialize, Deserialize)]
pub struct KdfParams {
    pub salt: Vec<u8>,
    /// scrypt cost as log2(N); N = 2^log_n.
    pub log_n: u8,
    pub r: u32,
    pub p: u32,
    pub key_len: usize,
}

/// Reads the salt from the file's envelope and pairs it with the format's
/// fixed scrypt cost.
pub fn kdf_params(file: &[u8]) -> Result<KdfParams> {
    let (_, salt) = parse_envelope(file)?;
    Ok(KdfParams {
        salt,
        log_n: SCRYPT_LOG_N,
        r: SCRYPT_R,
        p: SCRYPT_P,
        key_len: KEY_LEN,
    })
}

/// Derives the AES key from a password, in-process.
pub fn derive_key(password: &[u8], params: &KdfParams) -> Result<[u8; KEY_LEN]> {
    let cost = scrypt::Params::new(params.log_n, params.r, params.p, params.key_len)
        .map_err(|e| Error::Crypto(format!("derive key: {e}")))?;
    let mut key = [0u8; KEY_LEN];
    scrypt::scrypt(password, &params.salt, &cost, &mut key)
        .map_err(|e| Error::Crypto(format!("derive key: {e}")))?;
    Ok(key)
}

/// How the caller proves knowledge of the password.
#[derive(Clone, Copy)]
pub enum Secret<'a> {
    /// The password; the core runs scrypt.
    Password(&'a [u8]),
    /// The scrypt output the host derived from [`kdf_params`].
    DerivedKey(&'a [u8]),
}

/// Decrypts the bytes of an encrypted key file and returns the inner protobuf.
pub fn decrypt(file: &[u8], password: &[u8]) -> Result<Vec<u8>> {
    decrypt_with(file, Secret::Password(password))
}

/// Decrypts with a key the host already derived.
pub fn decrypt_with_key(file: &[u8], key: &[u8]) -> Result<Vec<u8>> {
    decrypt_with(file, Secret::DerivedKey(key))
}

pub fn decrypt_with(file: &[u8], secret: Secret<'_>) -> Result<Vec<u8>> {
    let (encrypted, _) = parse_envelope(file)?;
    let key = match secret {
        Secret::Password(password) => derive_key(password, &kdf_params(file)?)?.to_vec(),
        Secret::DerivedKey(key) => key.to_vec(),
    };
    if key.len() != KEY_LEN {
        return Err(bad(format!("derived key must be {KEY_LEN} bytes")));
    }
    let cipher = Aes256Gcm::new_from_slice(&key).map_err(|e| Error::Crypto(e.to_string()))?;
    if encrypted.len() < NONCE_LEN {
        return Err(bad("encrypted content too short"));
    }
    let (nonce, body) = encrypted.split_at(NONCE_LEN);
    cipher
        .decrypt(Nonce::from_slice(nonce), body)
        .map_err(|_| Error::Crypto("wrong password or corrupted file".into()))
}

/// Reads and decrypts a key file from disk.
pub fn decrypt_file(path: impl AsRef<Path>, password: &[u8]) -> Result<Vec<u8>> {
    let path = path.as_ref();
    let data = std::fs::read(path).map_err(|e| bad(format!("read {}: {e}", path.display())))?;
    decrypt(&data, password).map_err(|e| match e {
        Error::Crypto(m) => Error::Crypto(format!("{}: {m}", path.display())),
        Error::Validation(m) => bad(format!("parse {} envelope: {m}", path.display())),
        other => other,
    })
}

/// A decrypted key file by proto field number.
#[derive(Clone, Debug, Default, PartialEq, Eq, Serialize, Deserialize)]
pub struct Fields {
    pub strings: BTreeMap<u32, String>,
    pub varints: BTreeMap<u32, u64>,
}

impl Fields {
    /// First value of field `n`, or `""`.
    pub fn string(&self, n: u32) -> String {
        self.strings.get(&n).cloned().unwrap_or_default()
    }
    /// True when field `n` is a non-zero varint.
    pub fn bool(&self, n: u32) -> bool {
        self.varints.get(&n).copied().unwrap_or(0) != 0
    }
}

/// Walks decrypted proto3 bytes into string and varint maps.
pub fn parse_fields(data: &[u8]) -> Result<Fields> {
    let mut fields = Fields::default();
    let mut i = 0;
    while i < data.len() {
        let (tag, next) = read_varint(data, i)?;
        i = next;
        let number = (tag >> 3) as u32;
        match tag & 0x7 {
            0 => {
                let (value, next) = read_varint(data, i)?;
                fields.varints.insert(number, value);
                i = next;
            }
            1 => i += 8,
            2 => {
                let (start, end) = length_delimited(data, i, u64::from(number), "")?;
                fields
                    .strings
                    .entry(number)
                    .or_insert_with(|| String::from_utf8_lossy(&data[start..end]).into_owned());
                i = end;
            }
            5 => i += 4,
            wire => {
                return Err(bad(format!(
                    "unsupported wire type {wire} at field {number}"
                )))
            }
        }
        if i > data.len() {
            return Err(bad("truncated protobuf content"));
        }
    }
    Ok(fields)
}

/// Reads, decrypts and walks a key file.
pub fn decrypt_fields(path: impl AsRef<Path>, password: &[u8]) -> Result<Fields> {
    let path = path.as_ref();
    let plaintext = decrypt_file(path, password)?;
    parse_fields(&plaintext).map_err(|e| bad(format!("parse {} content: {e}", path.display())))
}

/// Cloudflare fields; their proto numbers differ between root and boot keys.
#[derive(Clone, Debug, Default, PartialEq, Eq, Serialize, Deserialize)]
pub struct Cloudflare {
    pub api_token: String,
    pub zone_id: String,
    pub account_id: String,
    pub mainnet_tunnel_token: String,
    pub testnet_tunnel_token: String,
    pub oracle_tunnel_token: String,
}

/// The decrypted contents of `root.key` with named fields (root_key.proto).
#[derive(Clone, Debug, Default, PartialEq, Eq, Serialize, Deserialize)]
pub struct RootSecrets {
    pub path: String,
    pub stripe_secret_key: String,
    pub stripe_webhook_secret: String,
    pub coinbase_api_key: String,
    pub coinbase_webhook_secret: String,
    pub root_private_key_hex: String,
    pub jwt_secret: String,
    pub knirv_jwt_secret: String,
    pub gemini_api_key: String,
    pub deepseek_api_key: String,
    pub cerebras_api_key: String,
    pub tls_cert: String,
    pub tls_key: String,
    pub cloudflare: Cloudflare,
    pub smtp_password: String,
    pub default_github_token: String,
    pub device_ip: String,
    pub device_password: String,
    pub device_username: String,
    pub payment_processor_enabled: bool,
    pub stripe_professional_price_id: String,
    pub stripe_enterprise_price_id: String,
    pub stripe_investor_price_id: String,
    pub payment_success_url: String,
    pub payment_cancel_url: String,
    pub payment_onboarding_callback_url: String,
    pub stripe_thin_webhook_secret: String,
    pub wasm_publisher_private_key_hex: String,
}

/// The decrypted contents of `boot.key` with named fields (bootnode_key.proto).
#[derive(Clone, Debug, Default, PartialEq, Eq, Serialize, Deserialize)]
pub struct BootSecrets {
    pub path: String,
    pub master_wallet_key_hex: String,
    pub registration_id: String,
    pub jwt_secret: String,
    pub gemini_api_key: String,
    pub deepseek_api_key: String,
    pub cerebras_api_key: String,
    pub cloudflare: Cloudflare,
    pub device_ip: String,
    pub device_password: String,
    pub device_username: String,
    pub stripe_secret_key: String,
    pub default_github_token: String,
    pub smtp_password: String,
}

impl RootSecrets {
    pub fn from_fields(path: &str, f: &Fields) -> Self {
        Self {
            path: path.into(),
            stripe_secret_key: f.string(1),
            stripe_webhook_secret: f.string(2),
            coinbase_api_key: f.string(3),
            coinbase_webhook_secret: f.string(4),
            root_private_key_hex: f.string(5),
            jwt_secret: f.string(10),
            knirv_jwt_secret: f.string(11),
            gemini_api_key: f.string(12),
            deepseek_api_key: f.string(13),
            cerebras_api_key: f.string(14),
            tls_cert: f.string(16),
            tls_key: f.string(17),
            cloudflare: Cloudflare {
                api_token: f.string(18),
                zone_id: f.string(19),
                account_id: f.string(26),
                mainnet_tunnel_token: f.string(27),
                oracle_tunnel_token: f.string(28),
                testnet_tunnel_token: f.string(38),
            },
            smtp_password: f.string(20),
            default_github_token: f.string(21),
            device_ip: f.string(22),
            device_password: f.string(23),
            device_username: f.string(24),
            payment_processor_enabled: f.bool(29),
            stripe_professional_price_id: f.string(30),
            stripe_enterprise_price_id: f.string(31),
            stripe_investor_price_id: f.string(32),
            payment_success_url: f.string(33),
            payment_cancel_url: f.string(34),
            payment_onboarding_callback_url: f.string(35),
            stripe_thin_webhook_secret: f.string(36),
            wasm_publisher_private_key_hex: f.string(37),
        }
    }
}

impl BootSecrets {
    pub fn from_fields(path: &str, f: &Fields) -> Self {
        Self {
            path: path.into(),
            master_wallet_key_hex: f.string(1),
            registration_id: f.string(2),
            jwt_secret: f.string(10),
            gemini_api_key: f.string(11),
            deepseek_api_key: f.string(12),
            cerebras_api_key: f.string(13),
            cloudflare: Cloudflare {
                api_token: f.string(18),
                zone_id: f.string(19),
                account_id: f.string(20),
                mainnet_tunnel_token: f.string(21),
                oracle_tunnel_token: f.string(29),
                testnet_tunnel_token: f.string(28),
            },
            device_ip: f.string(22),
            device_password: f.string(23),
            device_username: f.string(24),
            stripe_secret_key: f.string(25),
            default_github_token: f.string(26),
            smtp_password: f.string(27),
        }
    }
}

/// Decrypts a root key file into named fields.
pub fn open_root(path: &str, password: &[u8]) -> Result<RootSecrets> {
    Ok(RootSecrets::from_fields(
        path,
        &decrypt_fields(path, password)?,
    ))
}

/// Decrypts a boot key file into named fields.
pub fn open_boot(path: &str, password: &[u8]) -> Result<BootSecrets> {
    Ok(BootSecrets::from_fields(
        path,
        &decrypt_fields(path, password)?,
    ))
}

/// Decrypts key-file bytes the host already read, and walks them into fields.
pub fn decrypt_fields_bytes(file: &[u8], password: &[u8]) -> Result<Fields> {
    decrypt_fields_with(file, Secret::Password(password))
}

pub fn decrypt_fields_with(file: &[u8], secret: Secret<'_>) -> Result<Fields> {
    parse_fields(&decrypt_with(file, secret)?)
}

/// Named `root.key` contents from file bytes (`path` is carried through only).
pub fn open_root_bytes(path: &str, file: &[u8], password: &[u8]) -> Result<RootSecrets> {
    open_root_with(path, file, Secret::Password(password))
}

pub fn open_root_with(path: &str, file: &[u8], secret: Secret<'_>) -> Result<RootSecrets> {
    Ok(RootSecrets::from_fields(
        path,
        &decrypt_fields_with(file, secret)?,
    ))
}

/// Named `boot.key` contents from file bytes (`path` is carried through only).
pub fn open_boot_bytes(path: &str, file: &[u8], password: &[u8]) -> Result<BootSecrets> {
    open_boot_with(path, file, Secret::Password(password))
}

pub fn open_boot_with(path: &str, file: &[u8], secret: Secret<'_>) -> Result<BootSecrets> {
    Ok(BootSecrets::from_fields(
        path,
        &decrypt_fields_with(file, secret)?,
    ))
}

/// The process facts that key-file discovery depends on. Search order is a pure
/// function of this value so it can be tested and rendered deterministically.
#[derive(Clone, Debug, Default, Deserialize)]
#[serde(default)]
pub struct PathContext {
    pub oracle_key_path: String,
    pub knirv_root_key_path: String,
    pub knirv_boot_key_path: String,
    pub knirv_config_dir: String,
    pub sudo_user: String,
    /// `os.UserConfigDir()`; empty when unknown.
    pub user_config_dir: String,
    /// `os.UserHomeDir()`; empty when unknown.
    pub home_dir: String,
    pub exe_dir: String,
    pub cwd: String,
    /// Directory holding per-user home directories (`/home`).
    pub home_root: String,
    /// Names of the directories under `home_root`. Supplied by the caller (the
    /// host does the directory listing), so the search order is a pure function
    /// of this value. Sorted before use.
    pub home_users: Vec<String>,
    /// Directory holding system config (`/etc`).
    pub etc_dir: String,
}

fn env(name: &str) -> String {
    std::env::var(name).unwrap_or_default()
}

impl PathContext {
    /// Reads the current process environment.
    pub fn from_process() -> Self {
        let home = env("HOME");
        let user_config_dir = if cfg!(target_os = "macos") {
            if home.is_empty() {
                String::new()
            } else {
                format!("{home}/Library/Application Support")
            }
        } else if cfg!(windows) {
            env("APPDATA")
        } else {
            let xdg = env("XDG_CONFIG_HOME");
            if !xdg.is_empty() && Path::new(&xdg).is_absolute() {
                xdg
            } else if !home.is_empty() {
                format!("{home}/.config")
            } else {
                String::new()
            }
        };
        let dir_of = |p: Option<PathBuf>| {
            p.and_then(|p| p.parent().map(|d| d.to_string_lossy().into_owned()))
                .unwrap_or_default()
        };
        Self {
            oracle_key_path: env("ORACLE_KEY_PATH"),
            knirv_root_key_path: env("KNIRV_ROOT_KEY_PATH"),
            knirv_boot_key_path: env("KNIRV_BOOT_KEY_PATH"),
            knirv_config_dir: env("KNIRV_CONFIG_DIR"),
            sudo_user: env("SUDO_USER"),
            user_config_dir,
            home_dir: home,
            exe_dir: dir_of(std::env::current_exe().ok()),
            cwd: std::env::current_dir()
                .map(|p| p.to_string_lossy().into_owned())
                .unwrap_or_default(),
            home_root: "/home".into(),
            home_users: scan_home_users("/home"),
            etc_dir: "/etc".into(),
        }
    }
}

/// Lists the sub-directories of `home_root` (not following symlinks).
pub fn scan_home_users(home_root: &str) -> Vec<String> {
    let mut names: Vec<String> = std::fs::read_dir(home_root)
        .map(|entries| {
            entries
                .flatten()
                .filter(|e| e.file_type().map(|t| t.is_dir()).unwrap_or(false))
                .map(|e| e.file_name().to_string_lossy().into_owned())
                .collect()
        })
        .unwrap_or_default();
    names.sort();
    names
}

fn join(base: &str, parts: &[&str]) -> String {
    if base.is_empty() {
        return String::new();
    }
    let mut path = PathBuf::from(base);
    for part in parts {
        path.push(part);
    }
    path.to_string_lossy().into_owned()
}

fn dedupe(paths: Vec<String>) -> Vec<String> {
    let mut seen = std::collections::HashSet::new();
    paths
        .into_iter()
        .map(|p| p.trim().to_string())
        .filter(|p| !p.is_empty() && seen.insert(p.clone()))
        .collect()
}

/// Legacy locations shared by both key kinds, in search order.
fn legacy(c: &PathContext, name: &str) -> Vec<String> {
    let knirv_config = if c.knirv_config_dir.trim().is_empty() {
        String::new()
    } else {
        join(c.knirv_config_dir.trim(), &[name])
    };
    vec![
        knirv_config,
        join(&c.etc_dir, &["knirv-server", name]),
        join(&c.user_config_dir, &["knirv-server", name]),
        join(&c.home_dir, &[".knirv", name]),
        join(&c.home_dir, &[".local", "share", "knirv-server", name]),
        join(&c.exe_dir, &[name]),
        join(&c.exe_dir, &["bin", name]),
        join(&c.cwd, &[name]),
        join(&c.cwd, &["bin", name]),
    ]
}

/// The `boot.key` search order.
pub fn boot_search_paths_in(c: &PathContext) -> Vec<String> {
    let mut paths = vec![c.knirv_boot_key_path.clone()];
    paths.extend(legacy(c, "boot.key"));
    dedupe(paths)
}

/// The `root.key` search order, including hidden `.key` directories, the
/// `SUDO_USER` fallback and the `/home/*` scan.
pub fn root_search_paths_in(c: &PathContext) -> Vec<String> {
    let mut paths = vec![c.oracle_key_path.clone(), c.knirv_root_key_path.clone()];
    if !c.knirv_config_dir.trim().is_empty() {
        paths.push(join(c.knirv_config_dir.trim(), &[".key", "root.key"]));
    }
    paths.push(join(
        &c.user_config_dir,
        &["knirv-server", ".key", "root.key"],
    ));
    paths.push(join(&c.user_config_dir, &["knirv-server", "root.key"]));
    paths.push(join(&c.etc_dir, &["knirv-server", ".key", "root.key"]));
    let sudo = c.sudo_user.trim();
    if !sudo.is_empty() {
        for tail in [
            &[".config", "knirv-server", ".key", "root.key"][..],
            &[".config", "knirv-server", "root.key"][..],
        ] {
            paths.push(join(&join(&c.home_root, &[sudo]), tail));
        }
    }
    let mut users = c.home_users.clone();
    users.sort();
    for name in users {
        let home = join(&c.home_root, &[&name]);
        paths.push(join(
            &home,
            &[".config", "knirv-server", ".key", "root.key"],
        ));
        paths.push(join(&home, &[".config", "knirv-server", "root.key"]));
    }
    paths.push(c.knirv_boot_key_path.clone());
    paths.extend(legacy(c, "root.key"));
    dedupe(paths)
}

pub fn boot_search_paths() -> Vec<String> {
    boot_search_paths_in(&PathContext::from_process())
}
pub fn root_search_paths() -> Vec<String> {
    root_search_paths_in(&PathContext::from_process())
}

fn first_existing(paths: Vec<String>) -> Option<String> {
    paths
        .into_iter()
        .find(|p| std::fs::metadata(p).map(|m| !m.is_dir()).unwrap_or(false))
}

pub fn find_boot() -> Option<String> {
    first_existing(boot_search_paths())
}
pub fn find_root() -> Option<String> {
    first_existing(root_search_paths())
}

#[cfg(test)]
mod tests {
    use super::*;
    use base64::{engine::general_purpose::STANDARD as BASE64, Engine};
    use serde_json::Value;

    fn vector() -> Value {
        let path = format!(
            "{}/../testvectors/secrets/keyfile_v1.json",
            env!("CARGO_MANIFEST_DIR")
        );
        serde_json::from_slice(&std::fs::read(path).unwrap()).unwrap()
    }
    fn file(v: &Value, key: &str) -> Vec<u8> {
        BASE64.decode(v[key].as_str().unwrap()).unwrap()
    }

    #[test]
    fn root_vector_decrypts_to_named_fields() {
        let v = vector();
        let password = v["password"].as_str().unwrap().as_bytes();
        let fields = parse_fields(&decrypt(&file(&v, "root_key"), password).unwrap()).unwrap();
        let root = RootSecrets::from_fields("root.key", &fields);
        assert_eq!(root.stripe_secret_key, "sk_test_stripe");
        assert_eq!(root.stripe_webhook_secret, "whsec_test");
        assert_eq!(
            root.root_private_key_hex,
            "0000000000000000000000000000000000000000000000000000000000000001"
        );
        assert_eq!(root.jwt_secret, "jwt-root");
        assert_eq!(root.cloudflare.api_token, "cf-token");
        assert_eq!(root.cloudflare.zone_id, "cf-zone");
        assert_eq!(root.cloudflare.account_id, "cf-account");
        assert_eq!(root.cloudflare.mainnet_tunnel_token, "cf-mainnet");
        assert_eq!(root.cloudflare.oracle_tunnel_token, "cf-oracle");
        assert_eq!(root.cloudflare.testnet_tunnel_token, "cf-testnet");
        assert!(root.payment_processor_enabled);
        assert_eq!(root.stripe_professional_price_id, "price_pro");
        assert_eq!(root.stripe_thin_webhook_secret, "whsec_thin");
        assert_eq!(root.wasm_publisher_private_key_hex, "wasm-key");
    }

    #[test]
    fn boot_vector_decrypts_to_named_fields() {
        let v = vector();
        let password = v["password"].as_str().unwrap().as_bytes();
        let fields = parse_fields(&decrypt(&file(&v, "boot_key"), password).unwrap()).unwrap();
        let boot = BootSecrets::from_fields("boot.key", &fields);
        assert_eq!(
            boot.master_wallet_key_hex,
            "0000000000000000000000000000000000000000000000000000000000000002"
        );
        assert_eq!(boot.registration_id, "reg-123");
        assert_eq!(boot.jwt_secret, "jwt-boot");
        assert_eq!(boot.gemini_api_key, "gemini");
        assert_eq!(boot.cloudflare.account_id, "cf-account-b");
        assert_eq!(boot.cloudflare.mainnet_tunnel_token, "cf-mainnet-b");
        assert_eq!(boot.cloudflare.testnet_tunnel_token, "cf-testnet-b");
        assert_eq!(boot.cloudflare.oracle_tunnel_token, "cf-oracle-b");
    }

    #[test]
    fn wrong_password_and_corruption_fail_closed() {
        let v = vector();
        let data = file(&v, "root_key");
        assert!(decrypt(&data, b"wrong-password").is_err());
        let mut corrupt = data.clone();
        corrupt[20] ^= 1; // inside encrypted_content
        assert!(decrypt(&corrupt, v["password"].as_str().unwrap().as_bytes()).is_err());
        assert!(decrypt(&[], b"x").is_err());
        assert!(parse_envelope(&[0x08, 0x01]).is_err()); // varint wire type
    }

    #[test]
    fn fields_first_string_wins_and_bool() {
        // field 1 string "a", field 1 string "b", field 29 varint 1
        let data = [0x0a, 1, b'a', 0x0a, 1, b'b', 0xe8, 0x01, 0x01];
        let f = parse_fields(&data).unwrap();
        assert_eq!(f.string(1), "a");
        assert!(f.bool(29));
        assert!(!f.bool(30));
        assert!(parse_fields(&[0x0a, 5, b'a']).is_err()); // truncated
    }

    fn ctx() -> PathContext {
        PathContext {
            oracle_key_path: "/o/root.key".into(),
            knirv_root_key_path: "/r/root.key".into(),
            knirv_boot_key_path: "/b/boot.key".into(),
            knirv_config_dir: "/cfg".into(),
            sudo_user: "sudoer".into(),
            user_config_dir: "/u/.config".into(),
            home_dir: "/u".into(),
            exe_dir: "/exe".into(),
            cwd: "/cwd".into(),
            home_root: "/nonexistent-home-root".into(),
            home_users: vec![],
            etc_dir: "/etc".into(),
        }
    }

    #[test]
    fn boot_search_order() {
        assert_eq!(
            boot_search_paths_in(&ctx()),
            vec![
                "/b/boot.key",
                "/cfg/boot.key",
                "/etc/knirv-server/boot.key",
                "/u/.config/knirv-server/boot.key",
                "/u/.knirv/boot.key",
                "/u/.local/share/knirv-server/boot.key",
                "/exe/boot.key",
                "/exe/bin/boot.key",
                "/cwd/boot.key",
                "/cwd/bin/boot.key",
            ]
        );
    }

    #[test]
    fn root_search_order_and_dedupe() {
        let paths = root_search_paths_in(&ctx());
        assert_eq!(paths[0], "/o/root.key");
        assert_eq!(paths[1], "/r/root.key");
        assert_eq!(paths[2], "/cfg/.key/root.key");
        assert_eq!(paths[3], "/u/.config/knirv-server/.key/root.key");
        assert_eq!(paths[4], "/u/.config/knirv-server/root.key");
        assert_eq!(paths[5], "/etc/knirv-server/.key/root.key");
        assert_eq!(
            paths[6],
            "/nonexistent-home-root/sudoer/.config/knirv-server/.key/root.key"
        );
        // The legacy block repeats /u/.config/knirv-server/root.key: deduped.
        assert_eq!(
            paths
                .iter()
                .filter(|p| *p == "/u/.config/knirv-server/root.key")
                .count(),
            1
        );
        assert!(paths.contains(&"/b/boot.key".to_string()));
        assert!(paths.contains(&"/cwd/bin/root.key".to_string()));
    }

    #[test]
    fn home_scan_lists_only_directories_sorted() {
        let dir = std::env::temp_dir().join(format!("knirv-home-{}", std::process::id()));
        let _ = std::fs::remove_dir_all(&dir);
        std::fs::create_dir_all(dir.join("zed")).unwrap();
        std::fs::create_dir_all(dir.join("amy")).unwrap();
        std::fs::write(dir.join("afile"), b"x").unwrap();
        assert_eq!(scan_home_users(&dir.to_string_lossy()), vec!["amy", "zed"]);
        std::fs::remove_dir_all(&dir).unwrap();
    }

    #[test]
    fn home_users_are_searched_in_sorted_order() {
        let mut c = ctx();
        c.sudo_user.clear();
        c.home_users = vec!["zed".into(), "amy".into()];
        let paths = root_search_paths_in(&c);
        let amy = paths.iter().position(|p| p.contains("/amy/")).unwrap();
        let zed = paths.iter().position(|p| p.contains("/zed/")).unwrap();
        assert!(amy < zed);
    }

    #[test]
    fn host_derived_key_decrypts_the_same_as_the_password() {
        let v = vector();
        let password = v["password"].as_str().unwrap().as_bytes();
        let data = file(&v, "root_key");
        let params = kdf_params(&data).unwrap();
        assert_eq!(
            (params.log_n, params.r, params.p, params.key_len),
            (15, 8, 1, 32)
        );
        assert_eq!(params.salt.len(), 16);
        let key = derive_key(password, &params).unwrap();
        assert_eq!(
            decrypt_with_key(&data, &key).unwrap(),
            decrypt(&data, password).unwrap()
        );
        // A wrong or malformed key fails closed.
        assert!(decrypt_with_key(&data, &[0u8; 32]).is_err());
        assert!(decrypt_with_key(&data, &[0u8; 5]).is_err());
    }

    #[test]
    fn bytes_entry_points_match_the_file_ones() {
        let v = vector();
        let password = v["password"].as_str().unwrap().as_bytes();
        let root = open_root_bytes("root.key", &file(&v, "root_key"), password).unwrap();
        assert_eq!(root.jwt_secret, "jwt-root");
        let boot = open_boot_bytes("boot.key", &file(&v, "boot_key"), password).unwrap();
        assert_eq!(boot.registration_id, "reg-123");
        assert!(open_root_bytes("root.key", &file(&v, "root_key"), b"wrong").is_err());
    }
}
