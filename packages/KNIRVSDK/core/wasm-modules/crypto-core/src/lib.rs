//! KNIRVSDK crypto WASM module (module kind 1): the controller's vault crypto.
//!
//! A thin raw-ABI wrapper over `knirv_sdk::vault`. Zero imports and no JS glue,
//! as the controller's artifact validator requires. Every random value and
//! timestamp arrives in the request; nothing here reads a clock or an RNG.
//!
//! ABI (same family as `proof-verifier`): `allocate` / `deallocate` for request
//! buffers, `invoke(op, ptr, len) -> handle`, then `result_pointer` /
//! `result_length` / `free_result`. `op` 1 takes `{"op": "<name>", ...fields}` as
//! JSON and returns `{"ok": <value>}` or `{"error": "<message>"}`. Operation
//! names and fields are those of `knirv_sdk::vault::Vault::handle`; binary fields
//! are base64.
//!
//! Secrets stay in linear memory. The request buffer is wiped as soon as it has
//! been parsed, `deallocate` zeroes what it frees, `free_result` zeroes results,
//! and `zeroize()` drops every session. Code in the same worker can still read
//! this memory; this removes the stray JS copies and makes wiping testable.

use knirv_sdk::vault::{self, Vault};
use serde_json::{json, Value};
use std::{cell::RefCell, collections::HashMap};
use zeroize::Zeroize;

// The vault never draws randomness (the host supplies it), and the module must
// have no imports, so any attempt to is an error rather than a host call.
fn no_entropy(_: &mut [u8]) -> Result<(), getrandom::Error> {
    Err(getrandom::Error::UNSUPPORTED)
}
getrandom::register_custom_getrandom!(no_entropy);

const ABI_VERSION: i32 = 1;
/// Module kind 1 is the SDK crypto core in controller ABI v1.
const MODULE_KIND_CRYPTO: i32 = 1;
const OP_DISPATCH: i32 = 1;

thread_local! {
    static VAULT: RefCell<Vault> = RefCell::new(Vault::new());
    static RESULTS: RefCell<HashMap<i32, Vec<u8>>> = RefCell::new(HashMap::new());
    static NEXT_HANDLE: RefCell<i32> = const { RefCell::new(1) };
}

fn store(bytes: Vec<u8>) -> i32 {
    let handle = NEXT_HANDLE.with(|next| {
        let mut next = next.borrow_mut();
        let handle = *next;
        *next = next.wrapping_add(1).max(1);
        handle
    });
    RESULTS.with(|r| r.borrow_mut().insert(handle, bytes));
    handle
}

#[no_mangle]
pub extern "C" fn abi_version() -> i32 {
    ABI_VERSION
}

#[no_mangle]
pub extern "C" fn module_kind() -> i32 {
    MODULE_KIND_CRYPTO
}

#[no_mangle]
pub extern "C" fn crypto_protocol_version() -> i32 {
    2
}

#[no_mangle]
pub extern "C" fn self_test() -> i32 {
    (abi_version() == 1 && module_kind() == MODULE_KIND_CRYPTO && vault::self_test()) as i32
}

#[no_mangle]
pub extern "C" fn knirv_self_test() -> i32 {
    self_test()
}

#[no_mangle]
pub extern "C" fn allocate(length: i32) -> i32 {
    if length <= 0 {
        return 0;
    }
    let mut buffer = vec![0u8; length as usize];
    let ptr = buffer.as_mut_ptr() as i32;
    std::mem::forget(buffer); // reclaimed by deallocate
    ptr
}

#[no_mangle]
pub extern "C" fn deallocate(ptr: i32, length: i32) {
    if ptr == 0 || length <= 0 {
        return;
    }
    // SAFETY: exactly what `allocate` returned and forgot.
    let mut buffer = unsafe { Vec::from_raw_parts(ptr as *mut u8, length as usize, length as usize) };
    buffer.zeroize();
}

#[no_mangle]
pub extern "C" fn result_pointer(handle: i32) -> i32 {
    RESULTS.with(|r| {
        r.borrow()
            .get(&handle)
            .filter(|b| !b.is_empty())
            .map(|b| b.as_ptr() as i32)
            .unwrap_or(0)
    })
}

#[no_mangle]
pub extern "C" fn result_length(handle: i32) -> i32 {
    RESULTS.with(|r| r.borrow().get(&handle).map(|b| b.len() as i32).unwrap_or(0))
}

#[no_mangle]
pub extern "C" fn free_result(handle: i32) {
    if let Some(mut bytes) = RESULTS.with(|r| r.borrow_mut().remove(&handle)) {
        bytes.zeroize();
    }
}

/// Drops every session and every pending result.
#[no_mangle]
pub extern "C" fn zeroize() {
    VAULT.with(|v| v.borrow_mut().zeroize_all());
    RESULTS.with(|r| {
        for bytes in r.borrow_mut().values_mut() {
            bytes.zeroize();
        }
        r.borrow_mut().clear();
    });
}

/// Number of open sessions (for tests of `session.close` / `zeroize`).
#[no_mangle]
pub extern "C" fn session_count() -> i32 {
    VAULT.with(|v| v.borrow().session_count() as i32)
}

fn error(message: &str) -> Vec<u8> {
    json!({ "error": message }).to_string().into_bytes()
}

/// # Safety
/// `ptr`/`len` must be a live buffer from `allocate` that the host filled.
#[no_mangle]
pub unsafe extern "C" fn invoke(op: i32, ptr: i32, len: i32) -> i32 {
    if op != OP_DISPATCH {
        return store(error("unknown operation code"));
    }
    if ptr == 0 || len <= 0 {
        return store(error("empty request"));
    }
    let input = std::slice::from_raw_parts_mut(ptr as *mut u8, len as usize);
    let parsed = serde_json::from_slice::<Value>(input);
    input.zeroize(); // the secrets in the request are now in `parsed` only
    let mut request = match parsed {
        Ok(Value::Object(map)) => Value::Object(map),
        _ => return store(error("request must be a JSON object")),
    };
    let name = request
        .get("op")
        .and_then(Value::as_str)
        .map(str::to_owned)
        .unwrap_or_default();
    let result = VAULT.with(|v| v.borrow_mut().handle(&name, &request));
    vault::scrub(&mut request);
    store(match result {
        Ok(value) => json!({ "ok": value }).to_string().into_bytes(),
        Err(message) => error(&message),
    })
}
