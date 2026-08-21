//! Opaque-handle C ABI for the JSON envelope binding surface.
//!
//! All non-null output buffers are allocated by this library and must be
//! released by `knirv_bytes_free`. Inputs are borrowed for the call only.

use knirv_sdk::{wasm_module, BindingConfig, BindingEngine, RequestEnvelope, WasmModule, BINDING_API_VERSION};
use prost::Message;
use serde_json::Value;
use std::{
    collections::HashSet,
    panic::{catch_unwind, AssertUnwindSafe},
    ptr, slice,
    sync::{Mutex, OnceLock},
};

#[repr(C)]
pub struct knirv_engine_t {
    engine: BindingEngine,
    runtime: tokio::runtime::Runtime,
}
#[repr(C)]
#[derive(Clone, Copy)]
pub struct knirv_bytes_t {
    pub ptr: *const u8,
    pub len: usize,
}

/// Protobuf request envelope shared with the Go SDK. `payload` is the JSON
/// operation payload from the stable binding envelope, not a Rust memory view.
#[derive(Clone, PartialEq, Message)]
struct SdkRequest {
    #[prost(string, tag = "1")]
    action: String,
    #[prost(bytes = "vec", tag = "2")]
    payload: Vec<u8>,
}

#[derive(Clone, PartialEq, Message)]
struct SdkResponse {
    #[prost(bool, tag = "1")]
    success: bool,
    #[prost(string, tag = "2")]
    error_message: String,
    #[prost(bytes = "vec", tag = "3")]
    data: Vec<u8>,
}

pub const KNIRV_STATUS_OK: i32 = 0;
pub const KNIRV_STATUS_INVALID_ARGUMENT: i32 = 1;
pub const KNIRV_STATUS_AUTHENTICATION: i32 = 2;
pub const KNIRV_STATUS_TIMEOUT: i32 = 3;
pub const KNIRV_STATUS_TRANSPORT: i32 = 4;
pub const KNIRV_STATUS_API: i32 = 5;
pub const KNIRV_STATUS_CRYPTO: i32 = 6;
pub const KNIRV_STATUS_INTERNAL_PANIC: i32 = 7;

fn allocations() -> &'static Mutex<HashSet<usize>> {
    static VALUE: OnceLock<Mutex<HashSet<usize>>> = OnceLock::new();
    VALUE.get_or_init(|| Mutex::new(HashSet::new()))
}
fn engines() -> &'static Mutex<HashSet<usize>> {
    static VALUE: OnceLock<Mutex<HashSet<usize>>> = OnceLock::new();
    VALUE.get_or_init(|| Mutex::new(HashSet::new()))
}
fn clear(out: *mut knirv_bytes_t) {
    if !out.is_null() {
        unsafe {
            *out = knirv_bytes_t {
                ptr: ptr::null(),
                len: 0,
            };
        }
    }
}
fn owned_bytes(bytes: Vec<u8>) -> knirv_bytes_t {
    if bytes.is_empty() {
        return knirv_bytes_t {
            ptr: ptr::null(),
            len: 0,
        };
    }
    let bytes = bytes;
    let result = knirv_bytes_t {
        ptr: bytes.as_ptr(),
        len: bytes.len(),
    };
    allocations()
        .lock()
        .expect("allocation mutex poisoned")
        .insert(result.ptr as usize);
    std::mem::forget(bytes);
    result
}
fn input<'a>(bytes: knirv_bytes_t) -> Result<&'a [u8], i32> {
    if bytes.len == 0 {
        return Ok(&[]);
    }
    if bytes.ptr.is_null() {
        return Err(KNIRV_STATUS_INVALID_ARGUMENT);
    }
    Ok(unsafe { slice::from_raw_parts(bytes.ptr, bytes.len) })
}
fn boundary(out: *mut knirv_bytes_t, call: impl FnOnce() -> Result<Vec<u8>, i32>) -> i32 {
    clear(out);
    if out.is_null() {
        return KNIRV_STATUS_INVALID_ARGUMENT;
    }
    match catch_unwind(AssertUnwindSafe(call)) {
        Ok(Ok(bytes)) => {
            unsafe {
                *out = owned_bytes(bytes);
            }
            KNIRV_STATUS_OK
        }
        Ok(Err(status)) => status,
        Err(_) => KNIRV_STATUS_INTERNAL_PANIC,
    }
}

/// Creates an engine from UTF-8 JSON matching `BindingConfig`.
#[no_mangle]
pub extern "C" fn knirv_engine_new(
    config_json: knirv_bytes_t,
    out: *mut *mut knirv_engine_t,
) -> i32 {
    if !out.is_null() {
        unsafe {
            *out = ptr::null_mut();
        }
    }
    if out.is_null() {
        return KNIRV_STATUS_INVALID_ARGUMENT;
    }
    match catch_unwind(AssertUnwindSafe(|| {
        let config = input(config_json)?;
        let config = serde_json::from_slice::<BindingConfig>(config)
            .map_err(|_| KNIRV_STATUS_INVALID_ARGUMENT)?;
        let engine = BindingEngine::new(config).map_err(|_| KNIRV_STATUS_INVALID_ARGUMENT)?;
        let runtime = tokio::runtime::Builder::new_multi_thread()
            .enable_all()
            .build()
            .map_err(|_| KNIRV_STATUS_INTERNAL_PANIC)?;
        let value = Box::into_raw(Box::new(knirv_engine_t { engine, runtime }));
        engines()
            .lock()
            .expect("engine mutex poisoned")
            .insert(value as usize);
        unsafe {
            *out = value;
        }
        Ok(())
    })) {
        Ok(Ok(())) => KNIRV_STATUS_OK,
        Ok(Err(status)) => status,
        Err(_) => KNIRV_STATUS_INTERNAL_PANIC,
    }
}

/// Executes one versioned envelope synchronously and returns an owned JSON buffer.
#[no_mangle]
pub extern "C" fn knirv_engine_call(
    engine: *mut knirv_engine_t,
    request_json: knirv_bytes_t,
    response_json: *mut knirv_bytes_t,
) -> i32 {
    boundary(response_json, || {
        if engine.is_null() {
            return Err(KNIRV_STATUS_INVALID_ARGUMENT);
        }
        if !engines()
            .lock()
            .expect("engine mutex poisoned")
            .contains(&(engine as usize))
        {
            return Err(KNIRV_STATUS_INVALID_ARGUMENT);
        }
        let request = input(request_json)?;
        let engine = unsafe { &mut *engine };
        Ok(engine.runtime.block_on(engine.engine.call_json(request)))
    })
}

/// Executes a protobuf request using the binding-safe Rust envelope. The
/// returned bytes encode `SdkResponse` and are released with
/// `knirv_sdk_free_buffer` (or `knirv_bytes_free`).
#[no_mangle]
pub extern "C" fn knirv_sdk_invoke_proto(
    request_proto: knirv_bytes_t,
    response_proto: *mut knirv_bytes_t,
) -> i32 {
    boundary(response_proto, || {
        let request = SdkRequest::decode(input(request_proto)?).map_err(|_| KNIRV_STATUS_INVALID_ARGUMENT)?;
        if request.action.is_empty() {
            return Err(KNIRV_STATUS_INVALID_ARGUMENT);
        }
        let payload = if request.payload.is_empty() {
            Value::Object(Default::default())
        } else {
            serde_json::from_slice(&request.payload).map_err(|_| KNIRV_STATUS_INVALID_ARGUMENT)?
        };
        let engine = BindingEngine::new(BindingConfig::default()).map_err(|_| KNIRV_STATUS_INTERNAL_PANIC)?;
        let runtime = tokio::runtime::Builder::new_current_thread().enable_all().build().map_err(|_| KNIRV_STATUS_INTERNAL_PANIC)?;
        let response = runtime.block_on(engine.call(RequestEnvelope {
            version: BINDING_API_VERSION,
            operation: request.action,
            payload,
        }));
        let proto = if let Some(error) = response.error {
            SdkResponse { success: false, error_message: error.message, data: Vec::new() }
        } else {
            SdkResponse {
                success: true,
                error_message: String::new(),
                data: serde_json::to_vec(&response.payload.unwrap_or(Value::Null)).map_err(|_| KNIRV_STATUS_INTERNAL_PANIC)?,
            }
        };
        Ok(proto.encode_to_vec())
    })
}

/// Returns a copied JSON manifest for every embedded module. Release with
/// `knirv_bytes_free`.
#[no_mangle]
pub extern "C" fn knirv_module_manifest(out: *mut knirv_bytes_t) -> i32 {
    boundary(out, || {
        serde_json::to_vec(
            &WasmModule::ALL
                .into_iter()
                .map(WasmModule::metadata)
                .collect::<Vec<_>>(),
        )
        .map_err(|_| KNIRV_STATUS_INTERNAL_PANIC)
    })
}

/// Returns a copied byte buffer for a module selected by its UTF-8 identifier.
/// The output belongs to the SDK and must be released with `knirv_bytes_free`.
#[no_mangle]
pub extern "C" fn knirv_module_bytes(name: knirv_bytes_t, out: *mut knirv_bytes_t) -> i32 {
    boundary(out, || {
        let name = std::str::from_utf8(input(name)?).map_err(|_| KNIRV_STATUS_INVALID_ARGUMENT)?;
        wasm_module(name)
            .map(|module| module.bytes().to_vec())
            .ok_or(KNIRV_STATUS_INVALID_ARGUMENT)
    })
}

#[no_mangle]
pub extern "C" fn knirv_engine_free(engine: *mut knirv_engine_t) {
    if engine.is_null() {
        return;
    }
    let owned = engines()
        .lock()
        .expect("engine mutex poisoned")
        .remove(&(engine as usize));
    if owned {
        let _ = catch_unwind(AssertUnwindSafe(|| unsafe {
            drop(Box::from_raw(engine));
        }));
    }
}

/// Frees a buffer returned by `knirv_engine_call`; repeated frees are ignored.
#[no_mangle]
pub extern "C" fn knirv_bytes_free(bytes: knirv_bytes_t) {
    if bytes.ptr.is_null() || bytes.len == 0 {
        return;
    }
    let owned = allocations()
        .lock()
        .expect("allocation mutex poisoned")
        .remove(&(bytes.ptr as usize));
    if owned {
        let _ = catch_unwind(AssertUnwindSafe(|| unsafe {
            drop(Vec::from_raw_parts(
                bytes.ptr as *mut u8,
                bytes.len,
                bytes.len,
            ));
        }));
    }
}

/// Alias used by the protobuf ABI. It has the same ownership semantics as
/// `knirv_bytes_free` and accepts only SDK-returned buffers.
#[no_mangle]
pub extern "C" fn knirv_sdk_free_buffer(bytes: knirv_bytes_t) {
    knirv_bytes_free(bytes)
}

#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn crypto_round_trip_and_double_free_are_safe() {
        let config = br#"{}"#;
        let mut engine = ptr::null_mut();
        assert_eq!(
            knirv_engine_new(
                knirv_bytes_t {
                    ptr: config.as_ptr(),
                    len: config.len()
                },
                &mut engine
            ),
            KNIRV_STATUS_OK
        );
        let request = br#"{"version":1,"operation":"crypto.sha256","payload":{"data":"knirv"}}"#;
        let mut response = knirv_bytes_t {
            ptr: ptr::null(),
            len: 0,
        };
        assert_eq!(
            knirv_engine_call(
                engine,
                knirv_bytes_t {
                    ptr: request.as_ptr(),
                    len: request.len()
                },
                &mut response
            ),
            KNIRV_STATUS_OK
        );
        assert!(unsafe { slice::from_raw_parts(response.ptr, response.len) }
            .starts_with(b"{\"version\":1"));
        knirv_bytes_free(response);
        knirv_bytes_free(response);
        knirv_engine_free(engine);
    }

    #[test]
    fn protobuf_call_returns_a_go_owned_payload_after_copying() {
        let request = SdkRequest { action: "crypto.sha256".into(), payload: br#"{"data":"knirv"}"#.to_vec() }.encode_to_vec();
        let mut response = knirv_bytes_t { ptr: ptr::null(), len: 0 };
        assert_eq!(knirv_sdk_invoke_proto(knirv_bytes_t { ptr: request.as_ptr(), len: request.len() }, &mut response), KNIRV_STATUS_OK);
        let decoded = SdkResponse::decode(unsafe { slice::from_raw_parts(response.ptr, response.len) }).unwrap();
        assert!(decoded.success);
        assert_eq!(decoded.data, br#"{"digest":"844ab93035e622284f4d0db575d2a4a29a89c06acc652925e7f0b2b0392d35cf"}"#);
        knirv_sdk_free_buffer(response);
    }
}
