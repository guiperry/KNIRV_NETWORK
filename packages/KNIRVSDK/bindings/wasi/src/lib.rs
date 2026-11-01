//! The KNIRV SDK core as a WASI module.
//!
//! Hosts embed a WebAssembly runtime (the Go package uses wazero) instead of
//! linking native code, so they build without a C toolchain or CGO. The ABI is
//! three functions over linear memory:
//!
//! * `knirv_alloc(len) -> ptr`   reserve `len` bytes the host can write into
//! * `knirv_free(ptr, len)`      release a buffer from `knirv_alloc` or a response
//! * `knirv_invoke(ptr, len) -> u64`  run one request, `{version, operation,
//!   payload}` as JSON; returns `(response_ptr << 32) | response_len`
//!
//! The response is the binding response envelope as JSON and must be released
//! with `knirv_free`.
//!
//! Network operations go through one host import, `knirv_host.http_request`
//! (see `knirv_sdk::http_transport` for the JSON wire contract):
//!
//! * `http_request(ptr, len) -> u64`  `ptr..ptr+len` is the request JSON; the
//!   host allocates the response JSON with `knirv_alloc` and returns
//!   `(ptr << 32) | len`. The module frees it.
//!
//! The core owns retries, failover and error mapping; the host only moves bytes
//! and enforces the request's `timeout_ms`. Backoff sleeps use WASI clocks.
//!
//! Buffers are boxed slices, so their capacity always equals their length and
//! `knirv_free` can rebuild them exactly.

#[link(wasm_import_module = "knirv_host")]
extern "C" {
    fn http_request(ptr: *const u8, len: usize) -> u64;
}

/// Calls the host import and takes ownership of the response it allocated.
fn host_http(request: &[u8]) -> Vec<u8> {
    // SAFETY: the host reads `request` during the call and returns a buffer it
    // allocated with `knirv_alloc`, which `knirv_free` releases below.
    unsafe {
        let packed = http_request(request.as_ptr(), request.len());
        let (ptr, len) = ((packed >> 32) as *mut u8, packed as u32 as usize);
        if ptr.is_null() {
            return Vec::new();
        }
        let answer = std::slice::from_raw_parts(ptr, len).to_vec();
        knirv_free(ptr, len);
        answer
    }
}

/// Reserves `len` writable bytes. Never returns null.
#[no_mangle]
pub extern "C" fn knirv_alloc(len: usize) -> *mut u8 {
    Box::into_raw(vec![0u8; len.max(1)].into_boxed_slice()) as *mut u8
}

/// Releases a buffer of `len` bytes obtained from `knirv_alloc` or `knirv_invoke`.
///
/// # Safety
/// `ptr`/`len` must be exactly a value this module returned, released once.
#[no_mangle]
pub unsafe extern "C" fn knirv_free(ptr: *mut u8, len: usize) {
    if !ptr.is_null() {
        drop(Box::from_raw(std::ptr::slice_from_raw_parts_mut(ptr, len.max(1))));
    }
}

/// Runs one binding request and returns `(ptr << 32) | len` of the response.
///
/// # Safety
/// `ptr..ptr+len` must be readable memory (typically from `knirv_alloc`).
#[no_mangle]
pub unsafe extern "C" fn knirv_invoke(ptr: *const u8, len: usize) -> u64 {
    let request = std::slice::from_raw_parts(ptr, len);
    let mut response =
        knirv_sdk::call_json_with_transport(request, std::sync::Arc::new(knirv_sdk::JsonTransport::new(host_http)));
    // `len` must equal the allocation size for `knirv_free`; a zero-length
    // response cannot occur, but keep the invariant explicit.
    if response.is_empty() {
        response.push(b'{');
    }
    let length = response.len() as u64;
    let pointer = Box::into_raw(response.into_boxed_slice()) as *mut u8 as u64;
    (pointer << 32) | length
}
