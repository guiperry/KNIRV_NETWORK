//! Stable, serialization-only surface used by foreign-language bindings.
//!
//! This module deliberately does not expose SDK clients, Tokio types, or Rust
//! errors.  New operations are additive; changing an operation name or its
//! payload is a binding API major-version change.

use crate::{crypto, signing, wasm_module, ClientConfig, Error, KnirvClient, Network, WasmModule};
use base64::{engine::general_purpose::STANDARD as BASE64, Engine as _};
use serde::{Deserialize, Serialize};
use serde_json::{json, Value};
use std::{collections::BTreeMap, time::Duration};

pub const BINDING_API_VERSION: u32 = 1;

#[derive(Clone, Debug, Deserialize, Serialize)]
pub struct RequestEnvelope {
    pub version: u32,
    pub operation: String,
    #[serde(default)]
    pub payload: Value,
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

/// Binding-safe engine. Calls return an envelope for all ordinary failures so
/// hosts can deserialize one stable response shape.
#[derive(Clone, Debug)]
pub struct BindingEngine {
    client: KnirvClient,
}

impl BindingEngine {
    pub fn new(config: BindingConfig) -> Result<Self, BindingError> {
        let config = ClientConfig::try_from(config)?;
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
        let result = match operation.as_str() {
            "crypto.sha256" => local_sha256(&request.payload),
            "crypto.encrypt_aes" => crypto_encrypt(&request.payload),
            "crypto.decrypt_aes" => crypto_decrypt(&request.payload),
            "crypto.pbkdf2" => crypto_pbkdf2(&request.payload),
            "signing.marshal_action" => {
                marshal_value::<signing::Action>(&request.payload, signing::marshal_action)
            }
            "signing.sign_direct_transaction" => sign_direct_transaction(&request.payload),
            "signing.verify_transaction" => verify_transaction(&request.payload),
            "signing.marshal_message_envelope" => marshal_value::<signing::MessageEnvelope>(
                &request.payload,
                signing::marshal_message_envelope,
            ),
            "signing.sign_message_envelope" => sign_message_envelope(&request.payload),
            "signing.verify_message" => verify_message(&request.payload),
            "address.encode" => encode_address(&request.payload),
            "address.decode" => decode_address(&request.payload),
            "relay.marshal" => marshal_value::<signing::RelayEnvelope>(
                &request.payload,
                signing::marshal_relay_envelope,
            ),
            "relay.parse" => parse_relay(&request.payload),
            "wasm.publication.marshal" => marshal_value::<signing::WasmPublicationPayload>(
                &request.payload,
                signing::marshal_wasm_publication_payload,
            ),
            "wasm.publication.parse" => parse_wasm_publication(&request.payload),
            "wasm.assignment.marshal" => marshal_value::<signing::WasmManifestPayload>(
                &request.payload,
                signing::marshal_wasm_manifest_payload,
            ),
            "wasm.assignment.parse" => parse_wasm_manifest(&request.payload),
            "wasm.manifest" => Ok(wasm_manifest()),
            "wasm.bytes" => wasm_bytes(&request.payload),
            "wasm.verify" => wasm_verify(&request.payload),
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
fn selected_module(payload: &Value) -> Result<WasmModule, BindingError> {
    wasm_module(required_string(payload, "name")?)
        .ok_or_else(|| BindingError::invalid("unknown KNIRV WASM module"))
}
fn wasm_manifest() -> Value {
    json!({ "modules": WasmModule::ALL.into_iter().map(|module| module.metadata()).collect::<Vec<_>>() })
}
fn wasm_bytes(payload: &Value) -> Result<Value, BindingError> {
    let module = selected_module(payload)?;
    Ok(json!({ "module": module.metadata(), "bytes_base64": BASE64.encode(module.bytes()) }))
}
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
        None => module.bytes().to_vec(),
    };
    Ok(json!({ "valid": module.verify(&bytes).is_ok(), "module": module.metadata() }))
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
            })
            .await;
        assert_eq!(
            response.error.unwrap().code,
            BindingErrorCode::InvalidArgument
        );
    }
}
