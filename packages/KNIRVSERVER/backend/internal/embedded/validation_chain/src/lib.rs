//pub mod nrn_vault; // Make the module public if needed by other crates

pub mod nrn_token; //If you want to make the token library accessible outside.

// ---------------------------------------------------------------------------
// C FFI exports for CGo integration
// ---------------------------------------------------------------------------

use std::ffi::{CStr, CString};
use std::os::raw::{c_char, c_int};
use std::sync::Mutex;

use nrn_token::Transaction;

// Opaque engine state — holds the sled DB handle and any runtime config.
struct ValidationEngine {
    config: serde_json::Value,
}

impl ValidationEngine {
    fn new(config: serde_json::Value) -> anyhow::Result<Self> {
        Ok(Self { config })
    }

    /// Validate a raw transaction payload.
    ///
    /// Returns `(result_code, json_response)`:
    /// - `result_code == 0`: valid
    /// - `result_code == 1`: invalid signature / bad format
    /// - `result_code == -1`: internal error
    fn validate(&self, tx_json: &str) -> (i32, String) {
        let tx: Transaction = match serde_json::from_str(tx_json) {
            Ok(t) => t,
            Err(e) => {
                let msg = serde_json::json!({
                    "valid": false,
                    "error": format!("parse error: {}", e)
                });
                return (1, msg.to_string());
            }
        };

        // Basic structural validation: data and signature must be non-empty.
        if tx.data.trim().is_empty() || tx.signature.trim().is_empty() {
            let msg = serde_json::json!({
                "valid": false,
                "error": "transaction data or signature is empty"
            });
            return (1, msg.to_string());
        }

        let resp = serde_json::json!({
            "valid": true,
            "transaction_hash": tx.transaction_hash
        });
        (0, resp.to_string())
    }
}

static ENGINE: Mutex<Option<ValidationEngine>> = Mutex::new(None);

unsafe fn cstr_to_str<'a>(ptr: *const c_char) -> Option<&'a str> {
    if ptr.is_null() {
        return None;
    }
    CStr::from_ptr(ptr).to_str().ok()
}

fn to_raw_cstring(s: String) -> *mut c_char {
    match CString::new(s) {
        Ok(cs) => cs.into_raw(),
        Err(_) => std::ptr::null_mut(),
    }
}

/// Initialise the validation chain engine.
///
/// `config` is an optional NUL-terminated JSON string (may be `""` for defaults).
/// Returns 0 on success, -1 on error.
#[no_mangle]
pub extern "C" fn validation_chain_init(config: *const c_char, _config_len: usize) -> c_int {
    let cfg: serde_json::Value = if config.is_null() {
        serde_json::Value::Null
    } else {
        match unsafe { cstr_to_str(config) } {
            Some(s) if !s.trim().is_empty() => {
                serde_json::from_str(s).unwrap_or(serde_json::Value::Null)
            }
            _ => serde_json::Value::Null,
        }
    };

    let engine = match ValidationEngine::new(cfg) {
        Ok(e) => e,
        Err(_) => return -1,
    };

    match ENGINE.lock() {
        Ok(mut guard) => {
            *guard = Some(engine);
            0
        }
        Err(_) => -1,
    }
}

/// Shut down the validation chain engine.
///
/// Returns 0 on success, -1 if not initialised.
#[no_mangle]
pub extern "C" fn validation_chain_shutdown() -> c_int {
    match ENGINE.lock() {
        Ok(mut guard) => {
            *guard = None;
            0
        }
        Err(_) => -1,
    }
}

/// Return 0 if the engine is healthy, -1 otherwise.
#[no_mangle]
pub extern "C" fn validation_chain_health_check() -> c_int {
    match ENGINE.lock() {
        Ok(guard) => {
            if guard.is_some() { 0 } else { -1 }
        }
        Err(_) => -1,
    }
}

/// Validate a transaction.
///
/// `tx_data` must be a NUL-terminated UTF-8 JSON string.
/// `result_code` is set to 0 (valid), 1 (invalid), or -1 (error).
/// `result_len` is set to the byte length of the returned string (excluding NUL).
///
/// Returns a heap-allocated NUL-terminated JSON response string.
/// The caller **must** free it with `validation_chain_free_string`.
/// Returns NULL on internal error.
#[no_mangle]
pub extern "C" fn validation_chain_validate(
    tx_data: *const c_char,
    _tx_len: usize,
    result_code: *mut c_int,
    result_len: *mut usize,
) -> *mut c_char {
    let tx_str = match unsafe { cstr_to_str(tx_data) } {
        Some(s) => s.to_owned(),
        None => {
            if !result_code.is_null() {
                unsafe { *result_code = -1 };
            }
            return std::ptr::null_mut();
        }
    };

    let (code, payload) = match ENGINE.lock() {
        Ok(guard) => match guard.as_ref() {
            Some(engine) => engine.validate(&tx_str),
            None => (
                -1,
                serde_json::json!({"valid": false, "error": "engine not initialised"}).to_string(),
            ),
        },
        Err(_) => (
            -1,
            serde_json::json!({"valid": false, "error": "lock poisoned"}).to_string(),
        ),
    };

    if !result_code.is_null() {
        unsafe { *result_code = code };
    }
    if !result_len.is_null() {
        unsafe { *result_len = payload.len() };
    }

    to_raw_cstring(payload)
}

/// Free a string previously returned by `validation_chain_validate`.
///
/// Passing NULL is a no-op.
#[no_mangle]
pub extern "C" fn validation_chain_free_string(ptr: *mut c_char) {
    if !ptr.is_null() {
        unsafe { drop(CString::from_raw(ptr)) };
    }
}
