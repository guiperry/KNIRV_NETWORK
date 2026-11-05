//! Stable, serialization-only surface used by foreign-language bindings.
//!
//! This module deliberately does not expose SDK clients, Tokio types, or Rust
//! errors.  New operations are additive; changing an operation name or its
//! payload is a binding API major-version change.

use crate::{crypto, signing, ClientConfig, Error, KnirvClient, Network};
#[cfg(feature = "module-manifest")]
use crate::{wasm_module, WasmModule};
use base64::{engine::general_purpose::STANDARD as BASE64, Engine as _};
use serde::{Deserialize, Serialize};
use serde_json::{json, Value};
use crate::http_transport::HttpTransport;
use std::{collections::BTreeMap, sync::Arc, time::Duration};

mod ops;

pub const BINDING_API_VERSION: u32 = 1;

#[derive(Clone, Debug, Deserialize, Serialize)]
pub struct RequestEnvelope {
    pub version: u32,
    pub operation: String,
    #[serde(default)]
    pub payload: Value,
    /// Per-call client settings for network operations.
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub config: Option<BindingConfig>,
}

#[derive(Clone, Debug, Deserialize, Serialize)]
pub struct ResponseEnvelope {
    pub version: u32,
    pub operation: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub payload: Option<Value>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub error: Option<BindingError>,
}

#[derive(Clone, Copy, Debug, Deserialize, Serialize, PartialEq, Eq)]
#[serde(rename_all = "SCREAMING_SNAKE_CASE")]
pub enum BindingErrorCode {
    InvalidArgument,
    Authentication,
    Timeout,
    Transport,
    Api,
    Crypto,
    Unsupported,
    InternalPanic,
}

#[derive(Clone, Debug, Deserialize, Serialize)]
pub struct BindingError {
    pub code: BindingErrorCode,
    pub message: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub http_status: Option<u16>,
    pub retryable: bool,
}

#[derive(Clone, Debug, Default, Deserialize, Serialize)]
pub struct BindingConfig {
    #[serde(default)]
    pub network: Option<String>,
    #[serde(default)]
    pub api_key: Option<String>,
    #[serde(default)]
    pub timeout_ms: Option<u64>,
    #[serde(default)]
    pub transaction_url: Option<String>,
    #[serde(default)]
    pub gateway_url: Option<String>,
    #[serde(default)]
    pub controller_url: Option<String>,
    #[serde(default)]
    pub backend_url: Option<String>,
    #[serde(default)]
    pub headers: BTreeMap<String, String>,
}

impl TryFrom<BindingConfig> for ClientConfig {
    type Error = BindingError;

    fn try_from(value: BindingConfig) -> Result<Self, Self::Error> {
        let network = match value.network.as_deref().unwrap_or("public-production") {
            "public-production" | "production" => Network::PublicProduction,
            "public-testnet" | "testnet" => Network::PublicTestnet,
            "local-testnet" => Network::LocalTestnet,
            "local-production" => Network::LocalProduction,
            network => return Err(BindingError::invalid(format!("unknown network: {network}"))),
        };
        let mut config = ClientConfig {
            network,
            ..Default::default()
        };
        config.api_key = value.api_key;
        config.transaction_url = value.transaction_url;
        config.gateway_url = value.gateway_url;
        config.controller_url = value.controller_url;
        config.backend_url = value.backend_url;
        config.headers = value.headers;
        if let Some(timeout_ms) = value.timeout_ms {
            if timeout_ms == 0 {
                return Err(BindingError::invalid(
                    "timeout_ms must be greater than zero",
                ));
            }
            config.timeout = Duration::from_millis(timeout_ms);
        }
        Ok(config)
    }
}

impl BindingError {
    fn invalid(message: impl Into<String>) -> Self {
        Self {
            code: BindingErrorCode::InvalidArgument,
            message: message.into(),
            http_status: None,
            retryable: false,
        }
    }

    fn from_sdk(error: Error) -> Self {
        match error {
            Error::Configuration(message) | Error::Validation(message) => Self::invalid(message),
            Error::Decode(error) => Self::invalid(error.to_string()),
            Error::Timeout => Self {
                code: BindingErrorCode::Timeout,
                message: "request timed out".into(),
                http_status: None,
                retryable: true,
            },
            Error::Transport(error) => Self {
                code: BindingErrorCode::Transport,
                message: error.to_string(),
                http_status: None,
                retryable: true,
            },
            Error::Api { status, message } => Self {
                code: if status == 401 || status == 403 {
                    BindingErrorCode::Authentication
                } else {
                    BindingErrorCode::Api
                },
                message,
                http_status: Some(status),
                retryable: status >= 500,
            },
            Error::Crypto(message) => Self {
                code: BindingErrorCode::Crypto,
                message,
                http_status: None,
                retryable: false,
            },
            Error::IdempotencyConflict => Self {
                code: BindingErrorCode::Api,
                message: "idempotency key conflict".into(),
                http_status: Some(409),
                retryable: false,
            },
            Error::UnsupportedNetwork(message) => Self {
                code: BindingErrorCode::Unsupported,
                message,
                http_status: None,
                retryable: false,
            },
        }
    }
}

/// Operations that touch no network. They run identically in every host, so a
/// WebAssembly host can serve them without supplying a transport. `None` means
/// the operation is not local (it is a network operation or unknown).
pub fn local_operation(operation: &str, payload: &Value) -> Option<Result<Value, BindingError>> {
    Some(match operation {
        "models.encode_byte_v1" => models_encode(&payload),
        "models.encode_choice_set_v1" => models_encode_choice_set(&payload),
        "models.plan_text" => models_plan_text(&payload),
        "models.input_width" => models_input_width(&payload),
        "models.build_advise_prompt" => models_build_advise_prompt(&payload),
        "models.abstain_advice" => models_abstain(&payload),
        "models.advice_from_reasoning" => models_advice(&payload),
        "models.platform_roles" => models_platform_roles(&payload),
        "models.verify_check" => models_verify_check(&payload),
        "models.verify_request_check" => models_verify_request_check(&payload),
        "noderole.resolve" => payload_as::<crate::noderole::Inputs>(&payload)
            .map(|inputs| crate::noderole::resolve(&inputs))
            .and_then(to_value),
        "noderole.with_override" => noderole_with_override(&payload),
        "frame.defaults" => frame_defaults(),
        "frame.validate" => frame_registry(&payload).map(|_| json!({ "ok": true })),
        "frame.stage" => frame_stage(&payload),
        "frame.hash" => required_string(&payload, "system")
            .map(|system| json!({ "hash": crate::frame::hash(system) })),
        "frame.build" | "frame.build_lite" | "frame.check_shape" | "frame.check" | "frame.manifest" => {
            frame_op(operation, &payload)
        }
        "crypto.sha256" => local_sha256(&payload),
        "crypto.encrypt_aes" => crypto_encrypt(&payload),
        "crypto.decrypt_aes" => crypto_decrypt(&payload),
        "crypto.pbkdf2" => crypto_pbkdf2(&payload),
        "signing.marshal_action" => {
            marshal_value::<signing::Action>(&payload, signing::marshal_action)
        }
        "signing.sign_direct_transaction" => sign_direct_transaction(&payload),
        "signing.verify_transaction" => verify_transaction(&payload),
        "signing.marshal_message_envelope" => marshal_value::<signing::MessageEnvelope>(
            &payload,
            signing::marshal_message_envelope,
        ),
        "signing.sign_message_envelope" => sign_message_envelope(&payload),
        "signing.verify_message" => verify_message(&payload),
        "address.encode" => encode_address(&payload),
        "address.decode" => decode_address(&payload),
        "relay.marshal" => marshal_value::<signing::RelayEnvelope>(
            &payload,
            signing::marshal_relay_envelope,
        ),
        "relay.parse" => parse_relay(&payload),
        "relay.new_cli_supervisor" => new_cli_supervisor_relay(&payload),
        "relay.sign_cli_supervisor_response" => sign_cli_supervisor_response(&payload),
        "wasm.publication.marshal" => marshal_value::<signing::WasmPublicationPayload>(
            &payload,
            signing::marshal_wasm_publication_payload,
        ),
        "wasm.publication.parse" => parse_wasm_publication(&payload),
        "wasm.assignment.marshal" => marshal_value::<signing::WasmManifestPayload>(
            &payload,
            signing::marshal_wasm_manifest_payload,
        ),
        "wasm.assignment.parse" => parse_wasm_manifest(&payload),
        "wasm.manifest" => wasm_manifest(),
        "wasm.bytes" => wasm_bytes(&payload),
        "wasm.verify" => wasm_verify(&payload),
        _ => return ops::dispatch(operation, payload),
    })
}

/// Operations that need the network (and therefore a transport).
pub fn is_network_operation(operation: &str) -> bool {
    matches!(
        operation,
        "actuarial.call" | "transaction.chain" | "gateway.health" | "transmission.resolve_uri" | "transmission.broadcast" | "wallet.account"
    )
}

fn parse_request(request_json: &[u8]) -> Result<RequestEnvelope, Vec<u8>> {
    serde_json::from_slice::<RequestEnvelope>(request_json).map_err(|error| {
        response_error("", BindingError::invalid(format!("invalid request JSON: {error}")))
    })
}

/// Runs one request that needs no network. Network operations answer `UNSUPPORTED`.
pub fn call_local_json(request_json: &[u8]) -> Vec<u8> {
    let request = match parse_request(request_json) {
        Ok(request) => request,
        Err(response) => return response,
    };
    let response = if request.version != BINDING_API_VERSION {
        ResponseEnvelope::error(&request.operation, BindingError::invalid(format!("unsupported binding API version: {}", request.version)))
    } else {
        match local_operation(&request.operation, &request.payload) {
            Some(Ok(payload)) => ResponseEnvelope::success(&request.operation, payload),
            Some(Err(error)) => ResponseEnvelope::error(&request.operation, error),
            None => ResponseEnvelope::error(&request.operation, BindingError {
                code: BindingErrorCode::Unsupported,
                message: format!("unsupported operation: {}", request.operation),
                http_status: None,
                retryable: false,
            }),
        }
    };
    serde_json::to_vec(&response).expect("binding response is serializable")
}

/// Runs one request with a host-supplied transport. The request may carry a
/// `config` (the same shape as [`BindingConfig`]); retries, failover and error
/// mapping stay in this crate, the host only moves the bytes.
pub async fn call_json_async_with_transport(request_json: &[u8], transport: Arc<dyn HttpTransport>) -> Vec<u8> {
    let request = match parse_request(request_json) {
        Ok(request) => request,
        Err(response) => return response,
    };
    let operation = request.operation.clone();
    if !is_network_operation(&operation) {
        return call_local_json(request_json);
    }
    let config = request.config.clone().unwrap_or_default();
    match BindingEngine::with_transport(config, transport) {
        Ok(engine) => serde_json::to_vec(&engine.call(request).await).expect("binding response is serializable"),
        Err(error) => response_error(&operation, error),
    }
}

/// Synchronous form of [`call_json_async_with_transport`], for hosts whose
/// transport answers immediately (WASI).
pub fn call_json_with_transport(request_json: &[u8], transport: Arc<dyn HttpTransport>) -> Vec<u8> {
    crate::http_transport::block_on(call_json_async_with_transport(request_json, transport))
}

/// Binding-safe engine. Calls return an envelope for all ordinary failures so
/// hosts can deserialize one stable response shape.
#[derive(Clone, Debug)]
pub struct BindingEngine {
    client: KnirvClient,
}

impl BindingEngine {
    pub fn new(config: BindingConfig) -> Result<Self, BindingError> {
        Self::with_transport_option(config, None)
    }

    pub fn with_transport(config: BindingConfig, transport: Arc<dyn HttpTransport>) -> Result<Self, BindingError> {
        Self::with_transport_option(config, Some(transport))
    }

    fn with_transport_option(config: BindingConfig, transport: Option<Arc<dyn HttpTransport>>) -> Result<Self, BindingError> {
        let mut config = ClientConfig::try_from(config)?;
        config.transport = transport;
        KnirvClient::new(config)
            .map(|client| Self { client })
            .map_err(BindingError::from_sdk)
    }

    pub async fn call_json(&self, request_json: &[u8]) -> Vec<u8> {
        let request = match serde_json::from_slice::<RequestEnvelope>(request_json) {
            Ok(request) => request,
            Err(error) => {
                return response_error(
                    "",
                    BindingError::invalid(format!("invalid request JSON: {error}")),
                )
            }
        };
        serde_json::to_vec(&self.call(request).await).expect("binding response is serializable")
    }

    pub async fn call(&self, request: RequestEnvelope) -> ResponseEnvelope {
        if request.version != BINDING_API_VERSION {
            return ResponseEnvelope::error(
                &request.operation,
                BindingError::invalid(format!(
                    "unsupported binding API version: {}",
                    request.version
                )),
            );
        }
        let operation = request.operation.clone();
        let result = match local_operation(&operation, &request.payload) {
            Some(result) => result,
            None => match operation.as_str() {
                "actuarial.call" => required_string(&request.payload, "op")
                .and_then_async(|op| self.client.actuarial.call(op, &request.payload))
                .await,
            "transaction.chain" => self
                    .client
                    .transaction
                    .chain()
                    .await
                    .map_err(BindingError::from_sdk)
                    .and_then(to_value),
                "gateway.health" => self
                    .client
                    .gateway
                    .health()
                    .await
                    .map_err(BindingError::from_sdk)
                    .and_then(to_value),
                "transmission.resolve_uri" => {
                    required_string(&request.payload, "uri")
                        .and_then_async(|uri| self.client.transmission.resolve_uri(uri))
                        .await
                }
                "transmission.broadcast" => self
                    .client
                    .transmission
                    .broadcast(request.payload)
                    .await
                    .map_err(BindingError::from_sdk),
                "wallet.account" => self
                    .client
                    .wallet
                    .account()
                    .await
                    .map_err(BindingError::from_sdk)
                    .and_then(to_value),
                _ => Err(BindingError {
                    code: BindingErrorCode::Unsupported,
                    message: format!("unsupported operation: {operation}"),
                    http_status: None,
                    retryable: false,
                }),
            },
        };
        match result {
            Ok(payload) => ResponseEnvelope::success(&operation, payload),
            Err(error) => ResponseEnvelope::error(&operation, error),
        }
    }
}

trait BindingAsync<T> {
    async fn and_then_async<F, Fut>(self, f: F) -> Result<Value, BindingError>
    where
        F: FnOnce(T) -> Fut,
        Fut: std::future::Future<Output = crate::Result<Value>>;
}
impl<T> BindingAsync<T> for Result<T, BindingError> {
    async fn and_then_async<F, Fut>(self, f: F) -> Result<Value, BindingError>
    where
        F: FnOnce(T) -> Fut,
        Fut: std::future::Future<Output = crate::Result<Value>>,
    {
        match self {
            Ok(value) => f(value).await.map_err(BindingError::from_sdk),
            Err(error) => Err(error),
        }
    }
}

impl ResponseEnvelope {
    fn success(operation: &str, payload: Value) -> Self {
        Self {
            version: BINDING_API_VERSION,
            operation: operation.into(),
            payload: Some(payload),
            error: None,
        }
    }
    fn error(operation: &str, error: BindingError) -> Self {
        Self {
            version: BINDING_API_VERSION,
            operation: operation.into(),
            payload: None,
            error: Some(error),
        }
    }
}

fn response_error(operation: &str, error: BindingError) -> Vec<u8> {
    serde_json::to_vec(&ResponseEnvelope::error(operation, error))
        .expect("binding response is serializable")
}
fn to_value<T: Serialize>(value: T) -> Result<Value, BindingError> {
    serde_json::to_value(value).map_err(|error| BindingError::invalid(error.to_string()))
}
fn required_string<'a>(payload: &'a Value, field: &str) -> Result<&'a str, BindingError> {
    payload
        .get(field)
        .and_then(Value::as_str)
        .filter(|value| !value.is_empty())
        .ok_or_else(|| BindingError::invalid(format!("{field} must be a non-empty string")))
}
fn local_sha256(payload: &Value) -> Result<Value, BindingError> {
    Ok(json!({ "digest": crypto::sha256_string(required_string(payload, "data")?) }))
}
fn models_encode(payload: &Value) -> Result<Value, BindingError> {
    let width = payload
        .get("width")
        .and_then(Value::as_u64)
        .ok_or_else(|| BindingError::invalid("width must be a non-negative integer"))?;
    let text = payload
        .get("text")
        .and_then(Value::as_str)
        .ok_or_else(|| BindingError::invalid("text must be a string"))?;
    Ok(json!({ "tokens": crate::models::encode_byte_v1(text, width as usize) }))
}
fn models_encode_choice_set(payload: &Value) -> Result<Value, BindingError> {
    let width = payload
        .get("width")
        .and_then(Value::as_u64)
        .ok_or_else(|| BindingError::invalid("width must be a non-negative integer"))?;
    let text = |name: &str| {
        payload
            .get(name)
            .and_then(Value::as_str)
            .ok_or_else(|| BindingError::invalid(format!("{name} must be a string")))
    };
    let choices = payload
        .get("choices")
        .and_then(Value::as_array)
        .ok_or_else(|| BindingError::invalid("choices must be an array of strings"))?
        .iter()
        .map(|value| {
            value
                .as_str()
                .map(str::to_owned)
                .ok_or_else(|| BindingError::invalid("choices must be an array of strings"))
        })
        .collect::<Result<Vec<_>, _>>()?;
    Ok(json!({ "tokens": crate::models::encode_choice_set_v1(text("question")?, text("state")?, &choices, width as usize) }))
}
fn models_plan_text(payload: &Value) -> Result<Value, BindingError> {
    let tokens: Vec<u32> = payload
        .get("plan_tokens")
        .map(payload_as)
        .transpose()?
        .unwrap_or_default();
    Ok(json!({ "bytes_base64": BASE64.encode(crate::models::plan_bytes(&tokens)) }))
}
fn models_input_width(payload: &Value) -> Result<Value, BindingError> {
    crate::models::input_width(&payload.get("config").cloned().unwrap_or(Value::Null))
        .map(|width| json!({ "width": width }))
        .map_err(BindingError::invalid)
}
fn models_build_advise_prompt(payload: &Value) -> Result<Value, BindingError> {
    payload_as::<crate::models::AdviseContext>(payload)
        .map(|context| json!({ "prompt": crate::models::build_advise_prompt(&context) }))
}
fn models_abstain(payload: &Value) -> Result<Value, BindingError> {
    let text = |name: &str| payload.get(name).and_then(Value::as_str).unwrap_or_default();
    to_value(crate::models::abstain_advice(
        text("reason"),
        text("model_version"),
        text("dataset_manifest_hash"),
        text("now"),
    ))
}
fn models_advice(payload: &Value) -> Result<Value, BindingError> {
    let reasoning = payload
        .get("reasoning")
        .ok_or_else(|| BindingError::invalid("reasoning is required"))?;
    let context = payload
        .get("context")
        .ok_or_else(|| BindingError::invalid("context is required"))?;
    let now = payload.get("now").and_then(Value::as_str).unwrap_or_default();
    to_value(crate::models::advice_from_reasoning(
        &payload_as::<crate::models::Reasoning>(reasoning)?,
        &payload_as::<crate::models::AdviseContext>(context)?,
        now,
    ))
}
fn models_verify_check(payload: &Value) -> Result<Value, BindingError> {
    let response = payload.get("response").cloned().unwrap_or(Value::Null);
    // A response that cannot even be read is a broken answer, reported like any other.
    Ok(match serde_json::from_value::<crate::models::VerifyResponseView>(response) {
        Ok(view) => json!({ "error": crate::models::check_verify(&view) }),
        Err(e) => json!({ "error": format!("malformed verify response: {e}") }),
    })
}
fn models_verify_request_check(payload: &Value) -> Result<Value, BindingError> {
    payload_as::<crate::models::VerifyRequestView>(&payload.get("request").cloned().unwrap_or(Value::Null))
        .map(|req| json!({ "error": crate::models::check_verify_request(&req) }))
}
fn models_platform_roles(payload: &Value) -> Result<Value, BindingError> {
    payload_as::<crate::models::PlatformStatus>(&payload.get("status").cloned().unwrap_or(Value::Null))
        .map(|status| json!({ "roles": crate::models::roles(&status) }))
}
fn noderole_with_override(payload: &Value) -> Result<Value, BindingError> {
    let resolution = payload
        .get("resolution")
        .ok_or_else(|| BindingError::invalid("resolution is required"))?;
    let overrides = payload
        .get("override")
        .ok_or_else(|| BindingError::invalid("override is required"))?;
    payload_as::<crate::noderole::Resolution>(resolution)
        .and_then(|resolution| {
            payload_as::<crate::noderole::Override>(overrides)
                .map(|overrides| crate::noderole::with_override(resolution, &overrides))
        })
        .and_then(to_value)
}
fn frame_defaults() -> Result<Value, BindingError> {
    let (stages, tools) = crate::frame::defaults();
    Ok(json!({ "stages": stages, "tools": tools }))
}
fn frame_registry(payload: &Value) -> Result<crate::frame::Registry, BindingError> {
    frame_registry_with(payload, true)
}
fn frame_registry_with(payload: &Value, strict: bool) -> Result<crate::frame::Registry, BindingError> {
    let registry = payload
        .get("registry")
        .ok_or_else(|| BindingError::invalid("registry is required"))?;
    let stages = registry
        .get("stages")
        .map(payload_as)
        .transpose()?
        .unwrap_or_default();
    let tools = registry
        .get("tools")
        .map(payload_as)
        .transpose()?
        .unwrap_or_default();
    if strict {
        crate::frame::Registry::new(stages, tools).map_err(BindingError::invalid)
    } else {
        Ok(crate::frame::Registry::lenient(stages, tools))
    }
}
fn frame_stage(payload: &Value) -> Result<Value, BindingError> {
    let registry = frame_registry_with(payload, false)?;
    Ok(json!({ "stage": registry.stage(payload.get("id").and_then(Value::as_str).unwrap_or_default()) }))
}
fn frame_op(operation: &str, payload: &Value) -> Result<Value, BindingError> {
    use crate::frame::{FrameError, LiveFacts};
    let registry = frame_registry_with(payload, operation != "frame.build_lite")?;
    let stage = payload.get("stage").and_then(Value::as_str).unwrap_or_default();
    let system = payload.get("system").and_then(Value::as_str).unwrap_or_default();
    let frame_hash = payload.get("frame_hash").and_then(Value::as_str).unwrap_or_default();
    let facts = || -> Result<LiveFacts, BindingError> {
        payload.get("facts").map(payload_as).transpose().map(Option::unwrap_or_default)
    };
    let failed = |error: FrameError| {
        let kind = error.kind();
        let message = match error {
            FrameError::UnknownStage => format!("frame: unknown stage: {stage:?}"),
            FrameError::Unframed => "frame: call carries no frame".to_owned(),
            FrameError::HashMismatch => "frame: system message does not match the claimed frame hash".to_owned(),
            FrameError::StaleFrame => "frame: system message is not the current frame for this stage".to_owned(),
        };
        json!({ "ok": false, "kind": kind, "message": message })
    };
    let outcome = match operation {
        "frame.build" => registry.build(stage, &facts()?).map(|frame| json!({ "ok": true, "frame": frame })),
        "frame.build_lite" => registry.build_lite(stage).map(|lite| json!({ "ok": true, "lite": lite })),
        "frame.check_shape" => registry.check_shape(stage, system, frame_hash).map(|_| json!({ "ok": true })),
        "frame.check" => registry.check(stage, system, frame_hash, &facts()?).map(|_| json!({ "ok": true })),
        "frame.manifest" => Ok(json!({ "ok": true, "manifest": registry.manifest(&facts()?) })),
        _ => unreachable!("only frame operations are routed here"),
    };
    Ok(outcome.unwrap_or_else(failed))
}
fn payload_as<T: for<'de> Deserialize<'de>>(payload: &Value) -> Result<T, BindingError> {
    serde_json::from_value(payload.clone())
        .map_err(|error| BindingError::invalid(error.to_string()))
}
fn encoded(bytes: Vec<u8>) -> Value {
    json!({ "bytes_base64": BASE64.encode(bytes) })
}
fn bytes(payload: &Value, field: &str) -> Result<Vec<u8>, BindingError> {
    BASE64
        .decode(required_string(payload, field)?)
        .map_err(|error| BindingError::invalid(format!("invalid {field} base64: {error}")))
}
fn marshal_value<T: for<'de> Deserialize<'de>>(
    payload: &Value,
    marshal: impl FnOnce(&T) -> crate::Result<Vec<u8>>,
) -> Result<Value, BindingError> {
    let value = payload_as(payload)?;
    marshal(&value).map(encoded).map_err(BindingError::from_sdk)
}
fn crypto_encrypt(payload: &Value) -> Result<Value, BindingError> {
    crypto::encrypt_aes(
        required_string(payload, "data")?,
        required_string(payload, "password")?,
    )
    .map(|ciphertext| json!({"ciphertext": ciphertext}))
    .map_err(BindingError::from_sdk)
}
fn crypto_decrypt(payload: &Value) -> Result<Value, BindingError> {
    crypto::decrypt_aes(
        required_string(payload, "ciphertext")?,
        required_string(payload, "password")?,
    )
    .map(|data| json!({"data": data}))
    .map_err(BindingError::from_sdk)
}
fn crypto_pbkdf2(payload: &Value) -> Result<Value, BindingError> {
    let iterations = payload
        .get("iterations")
        .and_then(Value::as_u64)
        .ok_or_else(|| BindingError::invalid("iterations must be an integer"))?;
    let key_length = payload
        .get("key_length")
        .and_then(Value::as_u64)
        .ok_or_else(|| BindingError::invalid("key_length must be an integer"))?;
    crypto::pbkdf2_key(
        required_string(payload, "password")?,
        &bytes(payload, "salt_base64")?,
        u32::try_from(iterations).map_err(|_| BindingError::invalid("iterations is too large"))?,
        usize::try_from(key_length)
            .map_err(|_| BindingError::invalid("key_length is too large"))?,
    )
    .map(encoded)
    .map_err(BindingError::from_sdk)
}
fn private_key(payload: &Value) -> Result<[u8; 32], BindingError> {
    bytes(payload, "private_key_base64")?
        .try_into()
        .map_err(|_| BindingError::invalid("private_key_base64 must decode to 32 bytes"))
}
fn sign_direct_transaction(payload: &Value) -> Result<Value, BindingError> {
    let request = payload
        .get("request")
        .ok_or_else(|| BindingError::invalid("request is required"))?;
    signing::sign_direct_transaction(&private_key(payload)?, &payload_as(request)?)
        .map_err(BindingError::from_sdk)
        .and_then(to_value)
}
fn verify_transaction(payload: &Value) -> Result<Value, BindingError> {
    let tx = payload
        .get("transaction")
        .ok_or_else(|| BindingError::invalid("transaction is required"))?;
    let account_number = payload
        .get("account_number")
        .and_then(Value::as_u64)
        .ok_or_else(|| BindingError::invalid("account_number is required"))?;
    signing::verify_transaction(
        &payload_as(tx)?,
        required_string(payload, "chain_id")?,
        account_number,
    )
    .map(|_| json!({"valid":true}))
    .map_err(BindingError::from_sdk)
}
fn sign_message_envelope(payload: &Value) -> Result<Value, BindingError> {
    let envelope = payload
        .get("envelope")
        .ok_or_else(|| BindingError::invalid("envelope is required"))?;
    signing::sign_message_envelope(&private_key(payload)?, &payload_as(envelope)?)
        .map_err(BindingError::from_sdk)
        .and_then(to_value)
}
fn new_cli_supervisor_relay(payload: &Value) -> Result<Value, BindingError> {
    signing::new_cli_supervisor_relay(&payload_as(payload)?)
        .map_err(BindingError::from_sdk)
        .and_then(to_value)
}
fn sign_cli_supervisor_response(payload: &Value) -> Result<Value, BindingError> {
    let envelope = payload
        .get("envelope")
        .ok_or_else(|| BindingError::invalid("envelope is required"))?;
    signing::sign_cli_supervisor_relay_response(
        &private_key(payload)?,
        &payload_as(envelope)?,
        required_string(payload, "chain_id")?,
    )
    .map_err(BindingError::from_sdk)
    .and_then(to_value)
}
fn verify_message(payload: &Value) -> Result<Value, BindingError> {
    let signed = payload
        .get("signed")
        .ok_or_else(|| BindingError::invalid("signed is required"))?;
    let now = payload
        .get("now_unix")
        .and_then(Value::as_u64)
        .ok_or_else(|| BindingError::invalid("now_unix is required"))?;
    signing::verify_message(
        &payload_as(signed)?,
        required_string(payload, "domain")?,
        required_string(payload, "purpose")?,
        required_string(payload, "chain_id")?,
        required_string(payload, "nonce")?,
        now,
    )
    .map(|_| json!({"valid":true}))
    .map_err(BindingError::from_sdk)
}
fn encode_address(payload: &Value) -> Result<Value, BindingError> {
    signing::address(
        &bytes(payload, "public_key_base64")?,
        payload
            .get("prefix")
            .and_then(Value::as_str)
            .unwrap_or("knirv"),
    )
    .map(|address| json!({"address":address}))
    .map_err(BindingError::from_sdk)
}
fn decode_address(payload: &Value) -> Result<Value, BindingError> {
    signing::decode_address(
        required_string(payload, "address")?,
        payload
            .get("prefix")
            .and_then(Value::as_str)
            .unwrap_or("knirv"),
    )
    .map(encoded)
    .map_err(BindingError::from_sdk)
}
fn parse_relay(payload: &Value) -> Result<Value, BindingError> {
    signing::parse_relay_envelope(&bytes(payload, "bytes_base64")?)
        .map_err(BindingError::from_sdk)
        .and_then(to_value)
}
fn parse_wasm_publication(payload: &Value) -> Result<Value, BindingError> {
    signing::parse_wasm_publication_payload(&bytes(payload, "bytes_base64")?)
        .map_err(BindingError::from_sdk)
        .and_then(to_value)
}
fn parse_wasm_manifest(payload: &Value) -> Result<Value, BindingError> {
    signing::parse_wasm_manifest_payload(&bytes(payload, "bytes_base64")?)
        .map_err(BindingError::from_sdk)
        .and_then(to_value)
}
#[cfg(feature = "module-manifest")]
fn selected_module(payload: &Value) -> Result<WasmModule, BindingError> {
    wasm_module(required_string(payload, "name")?)
        .ok_or_else(|| BindingError::invalid("unknown KNIRV WASM module"))
}
#[cfg(feature = "module-manifest")]
fn wasm_manifest() -> Result<Value, BindingError> {
    Ok(json!({ "modules": WasmModule::ALL.into_iter().map(|module| module.metadata()).collect::<Vec<_>>() }))
}
#[cfg(all(feature = "module-manifest", feature = "embedded-modules"))]
fn wasm_bytes(payload: &Value) -> Result<Value, BindingError> {
    let module = selected_module(payload)?;
    Ok(json!({ "module": module.metadata(), "bytes_base64": BASE64.encode(module.bytes()) }))
}
#[cfg(feature = "module-manifest")]
fn wasm_verify(payload: &Value) -> Result<Value, BindingError> {
    let module = selected_module(payload)?;
    let bytes = match payload.get("bytes_base64") {
        Some(value) => BASE64
            .decode(
                value
                    .as_str()
                    .ok_or_else(|| BindingError::invalid("bytes_base64 must be a string"))?,
            )
            .map_err(|error| BindingError::invalid(format!("invalid base64: {error}")))?,
        None => embedded_bytes(module)?,
    };
    Ok(json!({ "valid": module.verify(&bytes).is_ok(), "module": module.metadata() }))
}
#[cfg(all(feature = "module-manifest", feature = "embedded-modules"))]
fn embedded_bytes(module: WasmModule) -> Result<Vec<u8>, BindingError> {
    Ok(module.bytes().to_vec())
}
#[cfg(all(feature = "module-manifest", not(feature = "embedded-modules")))]
fn embedded_bytes(_: WasmModule) -> Result<Vec<u8>, BindingError> {
    Err(wasm_modules_unavailable())
}
#[cfg(all(feature = "module-manifest", not(feature = "embedded-modules")))]
fn wasm_bytes(_: &Value) -> Result<Value, BindingError> {
    Err(wasm_modules_unavailable())
}
#[cfg(not(feature = "module-manifest"))]
fn wasm_manifest() -> Result<Value, BindingError> {
    Err(wasm_modules_unavailable())
}
#[cfg(not(feature = "module-manifest"))]
fn wasm_bytes(_: &Value) -> Result<Value, BindingError> {
    Err(wasm_modules_unavailable())
}
#[cfg(not(feature = "module-manifest"))]
fn wasm_verify(_: &Value) -> Result<Value, BindingError> {
    Err(wasm_modules_unavailable())
}
#[cfg(not(all(feature = "module-manifest", feature = "embedded-modules")))]
fn wasm_modules_unavailable() -> BindingError {
    BindingError {
        code: BindingErrorCode::Unsupported,
        message: "this build does not embed the WASM modules; fetch them from the release URL".into(),
        http_status: None,
        retryable: false,
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    #[tokio::test]
    async fn local_operations_use_versioned_envelopes() {
        let engine = BindingEngine::new(BindingConfig::default()).unwrap();
        let response = engine
            .call(RequestEnvelope {
                version: 1,
                operation: "crypto.sha256".into(),
                payload: json!({"data":"knirv"}),
                config: None,
            })
            .await;
        assert_eq!(
            response.payload.unwrap()["digest"],
            crypto::sha256_string("knirv")
        );
    }
    #[tokio::test]
    async fn invalid_requests_are_structured() {
        let engine = BindingEngine::new(BindingConfig::default()).unwrap();
        let response = engine
            .call(RequestEnvelope {
                version: 999,
                operation: "crypto.sha256".into(),
                payload: json!({}),
                config: None,
            })
            .await;
        assert_eq!(
            response.error.unwrap().code,
            BindingErrorCode::InvalidArgument
        );
    }
}
