//! Portable, deterministic subset of the KNIRV SDK core.
//!
//! Native HTTP clients stay in `../core`; this crate intentionally contains no
//! Tokio, filesystem, TLS, or OS entropy dependency so it can run in browser
//! and edge WebAssembly hosts.

use sha2::{Digest, Sha256};
use wasm_bindgen::prelude::*;

#[wasm_bindgen]
pub fn core_version() -> String {
    env!("CARGO_PKG_VERSION").to_owned()
}

/// Hash arbitrary caller-owned bytes with the SDK's canonical SHA-256 helper.
#[wasm_bindgen]
pub fn sha256_hex(data: &[u8]) -> String {
    hex::encode(Sha256::digest(data))
}

/// Fast integrity primitive for module assets fetched by a browser or edge host.
#[wasm_bindgen]
pub fn sha256_matches(data: &[u8], expected_hex: &str) -> bool {
    sha256_hex(data).eq_ignore_ascii_case(expected_hex)
}
