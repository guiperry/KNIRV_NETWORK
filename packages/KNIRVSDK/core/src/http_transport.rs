//! The network seam. The core owns request construction, retries, failover,
//! auth headers and error mapping (`client.rs`); an [`HttpTransport`] only moves
//! bytes. Native Rust callers get [`ReqwestTransport`]. WebAssembly hosts supply
//! an `http_request` import and the core drives it through [`JsonTransport`].
//!
//! Wire contract with a host (JSON, bytes as base64):
//!
//! ```text
//! request:  {"method","url","headers":{name:value},"body_base64","timeout_ms"}
//! response: {"status","headers":{name:value},"body_base64"}
//!        or {"error":{"kind":"timeout|dns|tls|refused|aborted|other","message"}}
//! ```
//!
//! `timeout_ms` is the deadline the host must enforce; the host aborts the
//! request in flight when it passes.

use crate::error::Error;
use base64::{engine::general_purpose::STANDARD as BASE64, Engine as _};
use serde::{Deserialize, Serialize};
use std::{collections::BTreeMap, fmt, future::Future, pin::Pin, sync::Arc, time::Duration};

#[cfg(not(target_arch = "wasm32"))]
pub type BoxFuture<'a, T> = Pin<Box<dyn Future<Output = T> + Send + 'a>>;
#[cfg(target_arch = "wasm32")]
pub type BoxFuture<'a, T> = Pin<Box<dyn Future<Output = T> + 'a>>;

#[derive(Clone, Debug, Default, PartialEq, Eq)]
pub struct HttpRequest {
    pub method: String,
    pub url: String,
    pub headers: BTreeMap<String, String>,
    pub body: Vec<u8>,
    pub timeout_ms: u64,
}

#[derive(Clone, Debug, Default, PartialEq, Eq)]
pub struct HttpResponse {
    pub status: u16,
    /// Header names are lower-cased.
    pub headers: BTreeMap<String, String>,
    pub body: Vec<u8>,
}

#[derive(Clone, Copy, Debug, Deserialize, Serialize, PartialEq, Eq)]
#[serde(rename_all = "snake_case")]
pub enum TransportErrorKind {
    Timeout,
    Dns,
    Tls,
    Refused,
    Aborted,
    #[serde(other)]
    Other,
}

impl fmt::Display for TransportErrorKind {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        f.write_str(match self {
            Self::Timeout => "timeout",
            Self::Dns => "dns",
            Self::Tls => "tls",
            Self::Refused => "refused",
            Self::Aborted => "aborted",
            Self::Other => "other",
        })
    }
}

#[derive(Clone, Debug, Deserialize, Serialize, PartialEq, Eq)]
pub struct TransportError {
    pub kind: TransportErrorKind,
    #[serde(default)]
    pub message: String,
}

impl From<TransportError> for Error {
    fn from(e: TransportError) -> Self {
        match e.kind {
            TransportErrorKind::Timeout => Error::Timeout,
            kind => Error::Transport(format!("{kind}: {}", e.message)),
        }
    }
}

/// Moves one HTTP exchange. Implementations do no retrying and no URL or header
/// policy; that all lives in `HttpClient`.
pub trait HttpTransport: fmt::Debug + Send + Sync {
    fn send(&self, request: HttpRequest) -> BoxFuture<'_, Result<HttpResponse, TransportError>>;
    /// Backoff between retries. The transport owns it because only the host
    /// knows how to wait (tokio, the browser, or a blocking WASI sleep).
    fn sleep(&self, duration: Duration) -> BoxFuture<'_, ()>;
}

/// The default transport for native callers.
#[cfg(feature = "http-reqwest")]
#[derive(Clone, Debug)]
pub struct ReqwestTransport {
    client: reqwest::Client,
}

#[cfg(feature = "http-reqwest")]
impl ReqwestTransport {
    pub fn new() -> Result<Self, Error> {
        Ok(Self {
            client: reqwest::Client::builder().build()?,
        })
    }
}

#[cfg(all(feature = "http-reqwest", not(target_arch = "wasm32")))]
fn is_connect(error: &reqwest::Error) -> bool {
    error.is_connect()
}
#[cfg(all(feature = "http-reqwest", target_arch = "wasm32"))]
fn is_connect(_: &reqwest::Error) -> bool {
    false
}

#[cfg(feature = "http-reqwest")]
fn classify(error: &reqwest::Error) -> TransportErrorKind {
    if error.is_timeout() {
        TransportErrorKind::Timeout
    } else if cfg!(not(target_arch = "wasm32")) && is_connect(error) {
        TransportErrorKind::Refused
    } else {
        TransportErrorKind::Other
    }
}

#[cfg(feature = "http-reqwest")]
impl HttpTransport for ReqwestTransport {
    fn send(&self, request: HttpRequest) -> BoxFuture<'_, Result<HttpResponse, TransportError>> {
        Box::pin(async move {
            let fail = |e: reqwest::Error| TransportError {
                kind: classify(&e),
                message: e.to_string(),
            };
            let method = reqwest::Method::from_bytes(request.method.as_bytes()).map_err(|e| {
                TransportError {
                    kind: TransportErrorKind::Other,
                    message: e.to_string(),
                }
            })?;
            let mut builder = self.client.request(method, &request.url);
            #[cfg(not(target_arch = "wasm32"))]
            if request.timeout_ms > 0 {
                builder = builder.timeout(Duration::from_millis(request.timeout_ms));
            }
            for (name, value) in &request.headers {
                builder = builder.header(name, value);
            }
            if !request.body.is_empty() {
                builder = builder.body(request.body);
            }
            let response = builder.send().await.map_err(fail)?;
            let status = response.status().as_u16();
            let headers = response
                .headers()
                .iter()
                .filter_map(|(k, v)| Some((k.as_str().to_ascii_lowercase(), v.to_str().ok()?.to_owned())))
                .collect();
            let body = response.bytes().await.map_err(fail)?.to_vec();
            Ok(HttpResponse {
                status,
                headers,
                body,
            })
        })
    }

    fn sleep(&self, duration: Duration) -> BoxFuture<'_, ()> {
        Box::pin(tokio::time::sleep(duration))
    }
}

#[derive(Serialize)]
struct WireRequest<'a> {
    method: &'a str,
    url: &'a str,
    headers: &'a BTreeMap<String, String>,
    body_base64: String,
    timeout_ms: u64,
}

#[derive(Deserialize)]
struct WireResponse {
    #[serde(default)]
    status: u16,
    #[serde(default)]
    headers: BTreeMap<String, String>,
    #[serde(default)]
    body_base64: String,
    error: Option<TransportError>,
}

fn other(message: impl Into<String>) -> TransportError {
    TransportError {
        kind: TransportErrorKind::Other,
        message: message.into(),
    }
}

/// Encodes a request for the host.
pub fn encode_request(request: &HttpRequest) -> Vec<u8> {
    serde_json::to_vec(&WireRequest {
        method: &request.method,
        url: &request.url,
        headers: &request.headers,
        body_base64: BASE64.encode(&request.body),
        timeout_ms: request.timeout_ms,
    })
    .expect("an HTTP request is serializable")
}

/// Decodes the host's answer.
pub fn decode_response(bytes: &[u8]) -> Result<HttpResponse, TransportError> {
    let wire: WireResponse =
        serde_json::from_slice(bytes).map_err(|e| other(format!("malformed host response: {e}")))?;
    if let Some(error) = wire.error {
        return Err(error);
    }
    if wire.status == 0 {
        return Err(other("host response has no status"));
    }
    let body = BASE64
        .decode(wire.body_base64)
        .map_err(|e| other(format!("host response body is not base64: {e}")))?;
    Ok(HttpResponse {
        status: wire.status,
        headers: wire
            .headers
            .into_iter()
            .map(|(k, v)| (k.to_ascii_lowercase(), v))
            .collect(),
        body,
    })
}

/// A transport over a synchronous host call that takes request JSON and returns
/// response JSON. On WASI the call is the `knirv_host.http_request` import; the
/// future is ready on its first poll, so a trivial executor can drive it.
#[derive(Clone)]
pub struct JsonTransport {
    call: Arc<dyn Fn(&[u8]) -> Vec<u8> + Send + Sync>,
}

impl JsonTransport {
    pub fn new(call: impl Fn(&[u8]) -> Vec<u8> + Send + Sync + 'static) -> Self {
        Self {
            call: Arc::new(call),
        }
    }
}

impl fmt::Debug for JsonTransport {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        f.write_str("JsonTransport")
    }
}

impl HttpTransport for JsonTransport {
    fn send(&self, request: HttpRequest) -> BoxFuture<'_, Result<HttpResponse, TransportError>> {
        let answer = (self.call)(&encode_request(&request));
        Box::pin(std::future::ready(decode_response(&answer)))
    }

    fn sleep(&self, duration: Duration) -> BoxFuture<'_, ()> {
        std::thread::sleep(duration);
        Box::pin(std::future::ready(()))
    }
}

/// Runs a future to completion on the current thread. Only for futures that
/// never return `Pending` for long (the [`JsonTransport`] case); a pending
/// future is polled again after yielding.
pub fn block_on<F: Future>(future: F) -> F::Output {
    use std::task::{Context, Poll, RawWaker, RawWakerVTable, Waker};
    fn clone(_: *const ()) -> RawWaker {
        RawWaker::new(std::ptr::null(), &VTABLE)
    }
    fn noop(_: *const ()) {}
    static VTABLE: RawWakerVTable = RawWakerVTable::new(clone, noop, noop, noop);
    // SAFETY: the vtable functions ignore the data pointer.
    let waker = unsafe { Waker::from_raw(RawWaker::new(std::ptr::null(), &VTABLE)) };
    let mut context = Context::from_waker(&waker);
    let mut future = Box::pin(future);
    loop {
        if let Poll::Ready(value) = future.as_mut().poll(&mut context) {
            return value;
        }
        std::thread::yield_now();
    }
}
