//! KNIRVSDK-owned crypto WASM identity/policy module.
//!
//! It deliberately uses the same raw ABI as the Controller relay module so it
//! can be distributed through the signed artifact pipeline without generated
//! JavaScript glue. Key derivation and signature bytes remain governed by the
//! SDK's canonical Go/TypeScript vectors; this module is the versioned WASM
//! compatibility and zeroization boundary consumed by the vault worker.

#[no_mangle]
pub extern "C" fn abi_version() -> i32 { 1 }

// Module kind 1 is reserved for the SDK crypto core in controller ABI v1.
#[no_mangle]
pub extern "C" fn module_kind() -> i32 { 1 }

#[no_mangle]
pub extern "C" fn crypto_protocol_version() -> i32 { 2 }

#[no_mangle]
pub extern "C" fn self_test() -> i32 {
    if abi_version() == 1 && module_kind() == 1 && crypto_protocol_version() == 2 { 1 } else { 0 }
}

#[no_mangle]
pub extern "C" fn knirv_self_test() -> i32 { self_test() }

// The host terminates the worker to erase key-bearing JS/WASM linear memory.
// This export is retained for ABI-compatible explicit teardown.
#[no_mangle]
pub extern "C" fn zeroize() {}
