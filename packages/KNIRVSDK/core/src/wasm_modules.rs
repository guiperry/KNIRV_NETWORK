//! Embedded KNIRV WebAssembly modules.
//!
//! The bytes are compiled once and included in the SDK binary/library with
//! [`include_bytes!`]. Consumers can execute the borrowed bytes directly or
//! materialize an exact copy when a host requires a `.wasm` file.

use serde::Serialize;
use sha2::{Digest, Sha256};
#[cfg(all(not(target_arch = "wasm32"), feature = "filesystem"))]
use std::fs;
use std::{fmt, io, path::Path};

#[derive(Clone, Debug, Serialize, PartialEq, Eq)]
pub struct WasmModuleMetadata {
    pub id: &'static str,
    pub artifact_version: &'static str,
    pub module_kind: &'static str,
    pub abi_version: u32,
    pub byte_length: usize,
    pub sha256: &'static str,
    pub capabilities: &'static [&'static str],
}

/// A WebAssembly module packaged with the KNIRV SDK.
#[derive(Clone, Copy, Debug, Eq, Hash, PartialEq)]
pub enum WasmModule {
    CognitiveShell,
    ControllerRelay,
    CryptoCore,
    DveVerifier,
}

impl WasmModule {
    pub const ALL: [Self; 4] = [
        Self::CognitiveShell,
        Self::ControllerRelay,
        Self::CryptoCore,
        Self::DveVerifier,
    ];

    /// Stable module identifier for configuration, CLI, and host integrations.
    pub const fn id(self) -> &'static str {
        match self {
            Self::CognitiveShell => "cognitive-shell",
            Self::ControllerRelay => "controller-relay",
            Self::CryptoCore => "crypto-core",
            Self::DveVerifier => "dve-verifier",
        }
    }

    pub const fn description(self) -> &'static str {
        match self {
            Self::CognitiveShell => "KNIRV Cortex cognitive processing module",
            Self::ControllerRelay => "KNIRV controller relay envelope module",
            Self::CryptoCore => "KNIRV SDK crypto policy module",
            Self::DveVerifier => "KNIRV DVE event-log verifier module",
        }
    }

    /// Embedded, immutable module bytes. This is zero-copy.
    pub const fn bytes(self) -> &'static [u8] {
        match self {
            Self::CognitiveShell => include_bytes!(concat!(
                env!("CARGO_MANIFEST_DIR"),
                "/wasm-modules/assets/cognitive-shell.wasm"
            )),
            Self::ControllerRelay => include_bytes!(concat!(
                env!("CARGO_MANIFEST_DIR"),
                "/wasm-modules/assets/controller-relay.wasm"
            )),
            Self::CryptoCore => include_bytes!(concat!(
                env!("CARGO_MANIFEST_DIR"),
                "/wasm-modules/assets/crypto-core.wasm"
            )),
            Self::DveVerifier => include_bytes!(concat!(
                env!("CARGO_MANIFEST_DIR"),
                "/wasm-modules/assets/dve-verifier.wasm"
            )),
        }
    }

    /// SHA-256 of the exact embedded artifact, used to pin host-side loading.
    pub const fn sha256(self) -> &'static str {
        match self {
            Self::CognitiveShell => {
                "ecea193f876691b91544f0820d153882640057c572bad5ad73d0dfb31d73a05e"
            }
            Self::ControllerRelay => {
                "7164752c846f88039690351ca56949c8c8c6ff47eb5ac34574081a62210f9ba3"
            }
            Self::CryptoCore => "9da085719be6a515b5188bf1295d52153a685c2853ad7e46d42d118f1b716f86",
            Self::DveVerifier => "6ace987570191c350d390ba64074d7d44678245982f56188c9261cf1de828eee",
        }
    }

    /// Versioned metadata for safe consumption outside this crate.
    pub fn metadata(self) -> WasmModuleMetadata {
        let (module_kind, capabilities) = match self {
            Self::CognitiveShell => ("cognitive", &["cognitive.process"] as &[_]),
            Self::ControllerRelay => ("relay", &["relay.envelope"] as &[_]),
            Self::CryptoCore => ("crypto", &["crypto.policy"] as &[_]),
            Self::DveVerifier => ("verifier", &["dve.verify"] as &[_]),
        };
        WasmModuleMetadata {
            id: self.id(),
            artifact_version: "1.0.0",
            module_kind,
            abi_version: 1,
            byte_length: self.bytes().len(),
            sha256: self.sha256(),
            capabilities,
        }
    }

    /// Checks immutable bytes against the SDK's pinned manifest.
    pub fn verify(self, bytes: &[u8]) -> Result<(), String> {
        if !bytes.starts_with(b"\0asm") {
            return Err("not a WebAssembly binary".into());
        }
        if bytes.len() != self.bytes().len() {
            return Err(format!("unexpected byte length: {}", bytes.len()));
        }
        if hex::encode(Sha256::digest(bytes)) != self.sha256() {
            return Err("SHA-256 digest does not match manifest".into());
        }
        let exports = wasm_exports(bytes)?;
        for expected in self.required_exports() {
            if !exports.iter().any(|actual| actual == expected) {
                return Err(format!(
                    "missing required {} ABI export: {expected}",
                    self.metadata().module_kind
                ));
            }
        }
        Ok(())
    }

    /// The exports that define this module's ABI. They are verified in addition
    /// to the digest, making an accidental manifest/module mismatch visible.
    pub const fn required_exports(self) -> &'static [&'static str] {
        match self {
            Self::CognitiveShell => &["hrmcognitive_process_cognitive_input"],
            Self::ControllerRelay => &["abi_version", "module_kind", "invoke"],
            Self::CryptoCore => &["abi_version", "module_kind", "crypto_protocol_version"],
            Self::DveVerifier => &["abi_version", "module_kind", "invoke"],
        }
    }

    pub fn parse(value: &str) -> Option<Self> {
        Self::ALL.into_iter().find(|module| module.id() == value)
    }

    /// Writes this module to `destination`. The destination is fully owned by
    /// the caller; the SDK never creates or mutates other files.
    #[cfg(all(not(target_arch = "wasm32"), feature = "filesystem"))]
    pub fn materialize(self, destination: impl AsRef<Path>) -> io::Result<()> {
        fs::write(destination, self.bytes())
    }

    #[cfg(not(all(not(target_arch = "wasm32"), feature = "filesystem")))]
    pub fn materialize(self, _destination: impl AsRef<Path>) -> io::Result<()> {
        Err(io::Error::new(
            io::ErrorKind::Unsupported,
            "WASM module materialization requires the native filesystem feature",
        ))
    }
}

fn wasm_exports(bytes: &[u8]) -> Result<Vec<String>, String> {
    if bytes.len() < 8 || &bytes[..4] != b"\0asm" {
        return Err("not a WebAssembly binary".into());
    }
    let mut pos = 8;
    while pos < bytes.len() {
        let id = bytes[pos];
        pos += 1;
        let (length, used) = read_leb(&bytes[pos..])?;
        pos += used;
        let end = pos
            .checked_add(length as usize)
            .ok_or("invalid WebAssembly section length")?;
        if end > bytes.len() {
            return Err("truncated WebAssembly section".into());
        }
        if id == 7 {
            let section = &bytes[pos..end];
            let (count, mut offset) = read_leb(section)?;
            let mut names = Vec::with_capacity(count as usize);
            for _ in 0..count {
                let (name_length, used) = read_leb(&section[offset..])?;
                offset += used;
                let name_end = offset
                    .checked_add(name_length as usize)
                    .ok_or("invalid export name length")?;
                if name_end >= section.len() {
                    return Err("truncated WebAssembly export".into());
                }
                let name = std::str::from_utf8(&section[offset..name_end])
                    .map_err(|_| "non-UTF-8 WebAssembly export")?
                    .to_owned();
                offset = name_end;
                offset += 1; // export kind
                let (_, used) = read_leb(&section[offset..])?;
                offset += used; // index
                names.push(name);
            }
            return Ok(names);
        }
        pos = end;
    }
    Err("WebAssembly module has no export section".into())
}
fn read_leb(input: &[u8]) -> Result<(u32, usize), String> {
    let mut value = 0u32;
    for (index, byte) in input.iter().copied().enumerate().take(5) {
        value |= ((byte & 0x7f) as u32) << (7 * index);
        if byte & 0x80 == 0 {
            return Ok((value, index + 1));
        }
    }
    Err("invalid unsigned LEB128 value".into())
}

impl fmt::Display for WasmModule {
    fn fmt(&self, formatter: &mut fmt::Formatter<'_>) -> fmt::Result {
        formatter.write_str(self.id())
    }
}

/// Returns an embedded module by stable identifier.
pub fn wasm_module(id: &str) -> Option<WasmModule> {
    WasmModule::parse(id)
}

/// Materializes an embedded module by stable identifier.
pub fn materialize_wasm_module(id: &str, destination: impl AsRef<Path>) -> io::Result<()> {
    let module = WasmModule::parse(id).ok_or_else(|| {
        io::Error::new(
            io::ErrorKind::InvalidInput,
            format!("unknown KNIRV WASM module: {id}"),
        )
    })?;
    module.materialize(destination)
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn every_embedded_module_has_the_pinned_digest() {
        for module in WasmModule::ALL {
            assert!(module.verify(module.bytes()).is_ok());
        }
    }

    #[test]
    fn modules_are_found_by_their_stable_identifiers() {
        assert_eq!(wasm_module("crypto-core"), Some(WasmModule::CryptoCore));
        assert_eq!(wasm_module("unknown"), None);
    }
}
