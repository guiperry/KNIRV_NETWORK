//! Portable, deterministic subset of the KNIRV SDK core.
//!
//! The Rust core stays in `../core`. [`invoke`] runs its local operations
//! (signing, addresses, envelopes, relay and WASM payloads, URIs, authority
//! records) synchronously. [`invoke_async`] also runs the network operations:
//! the core builds requests, retries, fails over and maps errors, and the host
//! object supplies `http_request(json) -> Promise<string>` and
//! `sleep(ms) -> Promise<void>` (the wire format is `knirv_sdk::http_transport`).

use knirv_sdk::{decode_response, encode_request, HttpRequest, HttpResponse, HttpTransport, TransportError};
use knirv_sdk::http_transport::{BoxFuture, TransportErrorKind};
use sha2::{Digest, Sha256};
use std::{sync::Arc, time::Duration};
use wasm_bindgen::prelude::*;
use wasm_bindgen_futures::JsFuture;

/// The host's two functions. WebAssembly here is single-threaded, so the
/// `Send`/`Sync` the transport trait asks for is vacuous.
#[derive(Debug)]
struct JsHost {
    http_request: js_sys::Function,
    sleep: js_sys::Function,
}
unsafe impl Send for JsHost {}
unsafe impl Sync for JsHost {}

fn host_error(message: String) -> TransportError {
    TransportError {
        kind: TransportErrorKind::Other,
        message,
    }
}

impl HttpTransport for JsHost {
    fn send(&self, request: HttpRequest) -> BoxFuture<'_, Result<HttpResponse, TransportError>> {
        Box::pin(async move {
            let wire = String::from_utf8(encode_request(&request)).map_err(|e| host_error(e.to_string()))?;
            let promise = self
                .http_request
                .call1(&JsValue::NULL, &JsValue::from_str(&wire))
                .map_err(|e| host_error(format!("http_request threw: {e:?}")))?;
            let answer = JsFuture::from(js_sys::Promise::resolve(&promise))
                .await
                .map_err(|e| host_error(format!("http_request rejected: {e:?}")))?;
            let answer = answer
                .as_string()
                .ok_or_else(|| host_error("http_request must resolve to a JSON string".into()))?;
            decode_response(answer.as_bytes())
        })
    }

    fn sleep(&self, duration: Duration) -> BoxFuture<'_, ()> {
        Box::pin(async move {
            if let Ok(promise) = self.sleep.call1(&JsValue::NULL, &JsValue::from_f64(duration.as_millis() as f64)) {
                let _ = JsFuture::from(js_sys::Promise::resolve(&promise)).await;
            }
        })
    }
}

fn read_host(host: &JsValue) -> Result<JsHost, JsValue> {
    let get = |name: &str| -> Result<js_sys::Function, JsValue> {
        js_sys::Reflect::get(host, &JsValue::from_str(name))?
            .dyn_into::<js_sys::Function>()
            .map_err(|_| JsValue::from_str(&format!("host.{name} must be a function")))
    };
    Ok(JsHost {
        http_request: get("http_request")?,
        sleep: get("sleep")?,
    })
}

/// Like [`invoke`], and network operations run through `host`
/// (`{http_request(json): Promise<string>, sleep(ms): Promise<void>}`).
#[wasm_bindgen]
pub async fn invoke_async(request_json: String, host: JsValue) -> Result<String, JsValue> {
    let transport: Arc<dyn HttpTransport> = Arc::new(read_host(&host)?);
    let out = knirv_sdk::call_json_async_with_transport(request_json.as_bytes(), transport).await;
    Ok(String::from_utf8(out).unwrap_or_else(|_| String::from("{}")))
}

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

/// Runs one binding request (`{version, operation, payload}` as JSON) against
/// the Rust core's local operations and returns the response envelope JSON.
/// Unknown or network operations answer `UNSUPPORTED`.
#[wasm_bindgen]
pub fn invoke(request_json: &str) -> String {
    String::from_utf8(knirv_sdk::call_local_json(request_json.as_bytes()))
        .unwrap_or_else(|_| String::from("{}"))
}
