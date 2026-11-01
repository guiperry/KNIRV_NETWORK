//! `authority.*` and `secrets.*` binding operations.
//!
//! Hex is used for hashes and keys, base64 for passwords and opaque bytes.
//! Verification failures are normal results (`{"valid":false,"code":…}`), not
//! binding errors, so callers can map `code` to their own error types.
use super::{bytes, payload_as, private_key, required_string, BindingError};
use crate::authority::{self, AuthorityError, Record, Registry, SessionCert, Tier};
use crate::replication;
#[cfg(feature = "secrets")]
use crate::secrets;
use crate::signing;
use crate::uri;
use base64::{engine::general_purpose::STANDARD as BASE64, Engine as _};
#[cfg(feature = "secrets")]
use serde::Deserialize;
use serde_json::{json, Value};

type Op = Result<Value, BindingError>;

/// Operations safe on any host: everything except the secrets operations that
/// read the filesystem themselves. The byte-based `secrets.*` operations are
/// included (when the `secrets` feature is on): the host reads the file and the
/// environment, the core does the decrypting and deciding.
pub(super) fn dispatch_portable(operation: &str, payload: &Value) -> Option<Op> {
    dispatch_inner(operation, payload).or_else(|| secrets_portable(operation, payload))
}

/// Everything, including the `secrets.*` operations that read files directly.
pub(super) fn dispatch(operation: &str, payload: &Value) -> Option<Op> {
    dispatch_portable(operation, payload).or_else(|| secrets_native(operation, payload))
}

#[cfg(feature = "secrets")]
fn secrets_portable(operation: &str, payload: &Value) -> Option<Op> {
    Some(match operation {
        "secrets.search_paths" => search_paths(payload),
        "secrets.kdf_params" => kdf_params(payload),
        "secrets.decrypt_bytes" => decrypt_bytes(payload),
        "secrets.fields_bytes" => fields_bytes(payload),
        "secrets.parse_fields" => parse_fields(payload),
        "secrets.open_bytes" => open_bytes(payload),
        _ => return None,
    })
}

#[cfg(not(feature = "secrets"))]
fn secrets_portable(_: &str, _: &Value) -> Option<Op> {
    None
}

#[cfg(all(feature = "secrets", not(target_family = "wasm")))]
fn secrets_native(operation: &str, payload: &Value) -> Option<Op> {
    Some(match operation {
        "secrets.find" => find(payload),
        "secrets.decrypt" => decrypt(payload),
        "secrets.fields" => fields(payload),
        "secrets.open" => open(payload),
        _ => return None,
    })
}

#[cfg(not(all(feature = "secrets", not(target_family = "wasm"))))]
fn secrets_native(_: &str, _: &Value) -> Option<Op> {
    None
}

fn dispatch_inner(operation: &str, payload: &Value) -> Option<Op> {
    Some(match operation {
        "authority.tier_target_bits" => tier_target_bits(payload),
        "authority.action_digest" => action_digest(payload),
        "authority.header" => header(payload),
        "authority.meets_target" => meets_target(payload),
        "authority.mine" => mine(payload),
        "authority.mine_header" => mine_header(payload),
        "authority.record_envelope" => record_envelope(payload),
        "authority.cert_envelope" => cert_envelope(payload),
        "authority.verify_cert" => verify_cert(payload),
        "authority.verify_record" => verify_record(payload),
        "signing.parse_action_body" => sign_parse_action_body(payload),
        "signing.parse_sequence" => sign_parse_sequence(payload),
        "signing.build_sign_doc" => sign_build_sign_doc(payload),
        "signing.marshal_tx_raw" => sign_marshal_tx_raw(payload),
        "signing.verify_message_payload" => sign_verify_message_payload(payload),
        "signing.parse_message_envelope" => sign_parse_message_envelope(payload),
        "signing.public_key" => sign_public_key(payload),
        "address.encode_raw" => address_encode_raw(payload),
        "replication.head_payload" => repl_head_payload(payload),
        "replication.head_hash" => repl_head_hash(payload),
        "replication.head_validate" => repl_head_validate(payload),
        "replication.compare" => repl_compare(payload),
        "replication.verify_head" => repl_verify_head(payload),
        "replication.pull_payload" => repl_pull_payload(payload),
        "replication.verify_pull" => repl_verify_pull(payload),
        "uri.parse" => uri_parse(payload),
        "uri.derive_id" => uri_id(payload, uri::derive_id),
        "uri.normalize_id" => uri_id(payload, uri::normalize_id),
        "uri.registry" => Ok(json!({
            "resource_types": uri::RESOURCE_TYPES,
            "query_keys": uri::QUERY_KEYS,
        })),
        _ => return None,
    })
}

fn invalid(message: impl Into<String>) -> BindingError {
    BindingError::invalid(message)
}

fn hex32(payload: &Value, field: &str) -> Result<[u8; 32], BindingError> {
    let bytes = hex::decode(required_string(payload, field)?)
        .map_err(|e| invalid(format!("invalid {field} hex: {e}")))?;
    bytes
        .try_into()
        .map_err(|v: Vec<u8>| invalid(format!("{field} must be 32 bytes, got {}", v.len())))
}

fn tier(payload: &Value) -> Result<Tier, BindingError> {
    let raw = payload
        .get("tier")
        .and_then(Value::as_u64)
        .ok_or_else(|| invalid("tier is required"))?;
    u8::try_from(raw)
        .map_err(|_| invalid("invalid tier"))
        .and_then(|t| Tier::try_from(t).map_err(invalid))
}

fn version(payload: &Value) -> Result<u8, BindingError> {
    match payload.get("version") {
        None | Some(Value::Null) => Ok(authority::HEADER_VERSION),
        Some(v) => v
            .as_u64()
            .and_then(|v| u8::try_from(v).ok())
            .ok_or_else(|| invalid("version must be a u8")),
    }
}

fn u32_field(payload: &Value, field: &str) -> Result<u32, BindingError> {
    payload
        .get(field)
        .and_then(Value::as_u64)
        .and_then(|v| u32::try_from(v).ok())
        .ok_or_else(|| invalid(format!("{field} must be a u32")))
}

fn now_unix(payload: &Value) -> Result<i64, BindingError> {
    payload
        .get("now_unix")
        .and_then(Value::as_i64)
        .ok_or_else(|| invalid("now_unix is required"))
}

/// Binding errors carry the failure class as a `[ErrX] ` prefix so callers can
/// rebuild typed errors without a second wire field.
fn coded(e: AuthorityError) -> BindingError {
    invalid(format!("[{}] {}", e.kind.code(), e))
}

/// A failed authority check is data; a malformed request is an error.
fn verdict(result: Result<(), AuthorityError>) -> Op {
    match result {
        Ok(()) => Ok(json!({ "valid": true })),
        Err(e) if e.kind == authority::Failure::Invalid => Err(coded(e)),
        Err(e) => Ok(json!({
            "valid": false,
            "code": e.kind.code(),
            "message": e.to_string(),
        })),
    }
}

fn envelope_json(e: &crate::signing::MessageEnvelope) -> Value {
    json!({
        "domain": e.domain,
        "purpose": e.purpose,
        "chain_id": e.chain_id,
        "nonce": e.nonce,
        "issued_at_unix": e.issued_at_unix,
        "expires_at_unix": e.expires_at_unix,
        "payload_base64": BASE64.encode(e.payload.as_deref().unwrap_or_default()),
    })
}

fn tier_target_bits(payload: &Value) -> Op {
    Ok(json!({ "bits": authority::tier_target_bits(tier(payload)?) }))
}

fn action_digest(payload: &Value) -> Op {
    let digest = authority::action_digest(
        tier(payload)?,
        required_string(payload, "domain")?,
        required_string(payload, "scope")?,
        required_string(payload, "operator")?,
        payload
            .get("epoch")
            .and_then(Value::as_u64)
            .ok_or_else(|| invalid("epoch is required"))?,
        &hex32(payload, "canonical_action_hash")?,
        &hex32(payload, "prev_record_hash")?,
    );
    Ok(json!({ "digest": hex::encode(digest) }))
}

fn meets_target(payload: &Value) -> Op {
    Ok(json!({
        "meets_target": authority::meets_target(&hex32(payload, "witness")?, u32_field(payload, "nbits")?)
    }))
}

fn header(payload: &Value) -> Op {
    let nbits = u32_field(payload, "nbits")?;
    let header = authority::Header::new(
        &hex32(payload, "prev_hash")?,
        &hex32(payload, "merkle")?,
        u32_field(payload, "ntime")?,
        nbits,
        u32_field(payload, "nonce")?,
        version(payload)?,
        tier(payload)?,
    );
    let witness = header.witness();
    Ok(json!({
        "header": hex::encode(header.0),
        "witness": hex::encode(witness),
        "meets_target": authority::meets_target(&witness, nbits),
    }))
}

fn mine_header(payload: &Value) -> Op {
    let nbits = u32_field(payload, "nbits")?;
    let (header, nonce) = authority::mine(
        &hex32(payload, "prev_hash")?,
        &hex32(payload, "merkle")?,
        u32_field(payload, "ntime")?,
        nbits,
        u32_field(payload, "start_nonce")?,
        version(payload)?,
        tier(payload)?,
    )
    .map_err(coded)?;
    Ok(json!({
        "header": hex::encode(header.0),
        "nonce": nonce,
        "witness": hex::encode(header.witness()),
    }))
}

fn record_of(payload: &Value) -> Result<Record, BindingError> {
    payload_as(
        payload
            .get("record")
            .ok_or_else(|| invalid("record is required"))?,
    )
}

fn mine(payload: &Value) -> Op {
    let mut record = record_of(payload)?;
    record.mine(u32_field(payload, "ntime")?).map_err(coded)?;
    serde_json::to_value(&record).map_err(|e| invalid(e.to_string()))
}

fn record_envelope(payload: &Value) -> Op {
    let record = record_of(payload)?;
    Ok(json!({
        "digest": hex::encode(record.digest()),
        "envelope": envelope_json(&record.envelope()),
    }))
}

fn cert_of(payload: &Value) -> Result<SessionCert, BindingError> {
    payload_as(
        payload
            .get("cert")
            .ok_or_else(|| invalid("cert is required"))?,
    )
}

fn cert_envelope(payload: &Value) -> Op {
    let cert = cert_of(payload)?;
    cert.validate().map_err(coded)?;
    Ok(json!({ "envelope": envelope_json(&cert.envelope()) }))
}

fn verify_cert(payload: &Value) -> Op {
    let cert = cert_of(payload)?;
    let long_term = hex::decode(required_string(payload, "long_term_pub")?)
        .map_err(|e| invalid(format!("invalid long_term_pub hex: {e}")))?;
    verdict(cert.verify(&long_term, now_unix(payload)?))
}

fn verify_record(payload: &Value) -> Op {
    let record = record_of(payload)?;
    let registry: Registry = payload_as(
        payload
            .get("registry")
            .ok_or_else(|| invalid("registry is required"))?,
    )?;
    verdict(authority::verify(&record, &registry, now_unix(payload)?))
}

#[cfg(feature = "secrets")]
#[derive(Deserialize)]
struct Kind {
    kind: String,
}

#[cfg(feature = "secrets")]
fn kind_of(payload: &Value) -> Result<String, BindingError> {
    let Kind { kind } = payload_as(payload)?;
    match kind.as_str() {
        "root" | "boot" => Ok(kind),
        other => Err(invalid(format!(
            "unknown key kind {other:?} (want root or boot)"
        ))),
    }
}

/// `secrets.search_paths`. With a `context` (the host's environment and
/// directories) the order is a pure function of it; without one the process's
/// own environment is used, which only makes sense natively.
#[cfg(feature = "secrets")]
fn search_paths(payload: &Value) -> Op {
    let context = match payload.get("context") {
        Some(value) if !value.is_null() => payload_as::<secrets::PathContext>(value)?,
        _ => secrets::PathContext::from_process(),
    };
    let paths = if kind_of(payload)? == "root" {
        secrets::root_search_paths_in(&context)
    } else {
        secrets::boot_search_paths_in(&context)
    };
    Ok(json!({ "paths": paths }))
}

#[cfg(all(feature = "secrets", not(target_family = "wasm")))]
fn find(payload: &Value) -> Op {
    let path = if kind_of(payload)? == "root" {
        secrets::find_root()
    } else {
        secrets::find_boot()
    };
    Ok(json!({ "path": path }))
}

#[cfg(feature = "secrets")]
fn password(payload: &Value) -> Result<Vec<u8>, BindingError> {
    BASE64
        .decode(required_string(payload, "password_base64")?)
        .map_err(|e| invalid(format!("invalid password_base64: {e}")))
}

#[cfg(all(feature = "secrets", not(target_family = "wasm")))]
fn decrypt(payload: &Value) -> Op {
    let plaintext = secrets::decrypt_file(required_string(payload, "path")?, &password(payload)?)
        .map_err(BindingError::from_sdk)?;
    Ok(json!({ "plaintext_base64": BASE64.encode(plaintext) }))
}

#[cfg(feature = "secrets")]
fn parse_fields(payload: &Value) -> Op {
    let data = BASE64
        .decode(required_string(payload, "data_base64")?)
        .map_err(|e| invalid(format!("invalid data_base64: {e}")))?;
    let fields = secrets::parse_fields(&data).map_err(BindingError::from_sdk)?;
    serde_json::to_value(fields).map_err(|e| invalid(e.to_string()))
}

#[cfg(all(feature = "secrets", not(target_family = "wasm")))]
fn fields(payload: &Value) -> Op {
    let fields = secrets::decrypt_fields(required_string(payload, "path")?, &password(payload)?)
        .map_err(BindingError::from_sdk)?;
    serde_json::to_value(fields).map_err(|e| invalid(e.to_string()))
}

#[cfg(all(feature = "secrets", not(target_family = "wasm")))]
fn open(payload: &Value) -> Op {
    let path = required_string(payload, "path")?;
    let password = password(payload)?;
    let value = if kind_of(payload)? == "root" {
        serde_json::to_value(secrets::open_root(path, &password).map_err(BindingError::from_sdk)?)
    } else {
        serde_json::to_value(secrets::open_boot(path, &password).map_err(BindingError::from_sdk)?)
    };
    value.map_err(|e| invalid(e.to_string()))
}

fn uri_id(payload: &Value, f: fn(&str) -> String) -> Op {
    let raw = payload
        .get("raw")
        .and_then(Value::as_str)
        .unwrap_or_default();
    Ok(json!({ "id": f(raw) }))
}

/// `uri.parse` never fails on bad input: an unparseable URI is a normal result
/// with `kind: "invalid"` and a reason.
fn uri_parse(payload: &Value) -> Op {
    let raw = payload
        .get("raw")
        .and_then(Value::as_str)
        .unwrap_or_default();
    let r = uri::parse(raw);
    let kind = match r.kind {
        uri::Kind::Canonical => "canonical",
        uri::Kind::LegacyDve => "legacy_dve",
        uri::Kind::ContentPointer => "content_pointer",
        uri::Kind::Action => "action",
        uri::Kind::Invalid => "invalid",
    };
    Ok(json!({
        "kind": kind,
        "canonical": r.canonical,
        "id": r.id,
        "resource_type": r.resource_type,
        "path": r.path,
        "query": r.query,
        "legacy_dve": r.legacy_dve.map(|l| json!({
            "form": l.form, "id": l.id, "hash": l.hash, "name": l.name,
        })),
        "content_pointer": r.content_pointer.map(|c| json!({ "kind": c.kind, "hash": c.hash })),
        "action": r.action.map(|a| json!({
            "name": a.name, "query": a.query, "canonical": a.canonical,
        })),
        "reason": r.reason,
    }))
}

// ---- signing helpers (byte fields are base64; see bindings.rs for the
// existing marshal/sign/verify operations) ----

fn b64(value: impl AsRef<[u8]>) -> String {
    BASE64.encode(value)
}

fn core_err(e: crate::Error) -> BindingError {
    BindingError::from_sdk(e)
}

fn sign_parse_action_body(payload: &Value) -> Op {
    let a = signing::parse_action_body(&bytes(payload, "body_base64")?).map_err(core_err)?;
    Ok(json!({
        "schema_version": a.schema_version,
        "action": a.action,
        "sender": a.sender,
        "recipient": a.recipient.unwrap_or_default(),
        "amount": a.amount.unwrap_or_default(),
        "payload_base64": b64(a.payload.unwrap_or_default()),
        "timestamp_unix": a.timestamp_unix,
    }))
}

fn sign_parse_sequence(payload: &Value) -> Op {
    let sequence =
        signing::parse_sequence(&bytes(payload, "auth_info_base64")?).map_err(core_err)?;
    Ok(json!({ "sequence": sequence }))
}

fn sign_build_sign_doc(payload: &Value) -> Op {
    let request: signing::DirectSignRequest = payload_as(
        payload
            .get("request")
            .ok_or_else(|| invalid("request is required"))?,
    )?;
    let (body, auth_info, sign_doc) =
        signing::build_direct_sign_doc(&request, &bytes(payload, "public_key_base64")?)
            .map_err(core_err)?;
    Ok(json!({
        "body_base64": b64(body),
        "auth_info_base64": b64(auth_info),
        "sign_doc_base64": b64(sign_doc),
    }))
}

fn sign_marshal_tx_raw(payload: &Value) -> Op {
    let signatures: Vec<Vec<u8>> = payload
        .get("signatures_base64")
        .and_then(Value::as_array)
        .ok_or_else(|| invalid("signatures_base64 must be an array"))?
        .iter()
        .map(|v| {
            v.as_str()
                .ok_or_else(|| invalid("signatures_base64 entries must be strings"))
                .and_then(|s| {
                    BASE64
                        .decode(s)
                        .map_err(|e| invalid(format!("invalid signature base64: {e}")))
                })
        })
        .collect::<Result<_, _>>()?;
    let refs: Vec<&[u8]> = signatures.iter().map(Vec::as_slice).collect();
    let raw = signing::marshal_tx_raw(
        &bytes(payload, "body_base64")?,
        &bytes(payload, "auth_info_base64")?,
        &refs,
    );
    Ok(json!({ "bytes_base64": b64(raw) }))
}

fn sign_verify_message_payload(payload: &Value) -> Op {
    let signed = payload
        .get("signed")
        .ok_or_else(|| invalid("signed is required"))?;
    let now = payload
        .get("now_unix")
        .and_then(Value::as_u64)
        .ok_or_else(|| invalid("now_unix is required"))?;
    signing::verify_message_payload(
        &payload_as(signed)?,
        required_string(payload, "domain")?,
        required_string(payload, "purpose")?,
        required_string(payload, "chain_id")?,
        &bytes_or_empty(payload, "payload_base64")?,
        now,
    )
    .map(|_| json!({ "valid": true }))
    .map_err(core_err)
}

/// Like `bytes`, but an absent or empty field is an empty byte string.
fn bytes_or_empty(payload: &Value, field: &str) -> Result<Vec<u8>, BindingError> {
    match payload.get(field).and_then(Value::as_str) {
        None | Some("") => Ok(Vec::new()),
        Some(text) => BASE64
            .decode(text)
            .map_err(|e| invalid(format!("invalid {field} base64: {e}"))),
    }
}

fn sign_parse_message_envelope(payload: &Value) -> Op {
    let p = signing::parse_message_envelope(&bytes(payload, "bytes_base64")?).map_err(core_err)?;
    Ok(json!({
        "schema_version": p.schema_version,
        "domain": p.domain,
        "purpose": p.purpose,
        "chain_id": p.chain_id,
        "nonce": p.nonce,
        "issued_at_unix": p.issued_at_unix,
        "expires_at_unix": p.expires_at_unix,
        "payload_base64": b64(p.payload),
    }))
}

fn sign_public_key(payload: &Value) -> Op {
    let key = private_key(payload)?;
    let signing_key = k256::ecdsa::SigningKey::from_bytes((&key).into())
        .map_err(|e| BindingError::from_sdk(crate::Error::Crypto(e.to_string())))?;
    let public = signing_key.verifying_key().to_encoded_point(true);
    let public = public.as_bytes();
    Ok(json!({
        "public_key_base64": b64(public),
        "address": signing::knirv_address(public).map_err(core_err)?,
    }))
}

fn address_encode_raw(payload: &Value) -> Op {
    let prefix = payload
        .get("prefix")
        .and_then(Value::as_str)
        .unwrap_or("knirv");
    signing::encode_address(&bytes(payload, "raw_base64")?, prefix)
        .map(|address| json!({ "address": address }))
        .map_err(core_err)
}

// ---- byte-based secrets operations: the host reads the file ----

#[cfg(feature = "secrets")]
fn file_bytes(payload: &Value) -> Result<Vec<u8>, BindingError> {
    bytes(payload, "data_base64")
}

/// Errors carry the file's path when the host supplies one, as the path-based
/// operations always did.
#[cfg(feature = "secrets")]
fn labelled<T>(payload: &Value, result: crate::Result<T>) -> Result<T, BindingError> {
    result.map_err(|error| {
        match payload
            .get("path")
            .and_then(Value::as_str)
            .filter(|p| !p.is_empty())
        {
            Some(path) => BindingError::from_sdk(match error {
                crate::Error::Crypto(m) => crate::Error::Crypto(format!("{path}: {m}")),
                crate::Error::Validation(m) => {
                    crate::Error::Validation(format!("parse {path}: {m}"))
                }
                other => other,
            }),
            None => BindingError::from_sdk(error),
        }
    })
}

/// The password or, when the host already ran scrypt, the key it derived from
/// `secrets.kdf_params`.
#[cfg(feature = "secrets")]
fn credential(payload: &Value) -> Result<(Vec<u8>, bool), BindingError> {
    match payload
        .get("derived_key_base64")
        .and_then(Value::as_str)
        .filter(|k| !k.is_empty())
    {
        Some(key) => Ok((
            BASE64
                .decode(key)
                .map_err(|e| invalid(format!("invalid derived_key_base64: {e}")))?,
            true,
        )),
        None => Ok((password(payload)?, false)),
    }
}

#[cfg(feature = "secrets")]
fn secret_of(bytes: &[u8], derived: bool) -> secrets::Secret<'_> {
    if derived {
        secrets::Secret::DerivedKey(bytes)
    } else {
        secrets::Secret::Password(bytes)
    }
}

/// `secrets.kdf_params`: the salt and scrypt cost a key file calls for.
#[cfg(feature = "secrets")]
fn kdf_params(payload: &Value) -> Op {
    let p = labelled(payload, secrets::kdf_params(&file_bytes(payload)?))?;
    Ok(json!({
        "salt_base64": BASE64.encode(p.salt),
        "log_n": p.log_n, "r": p.r, "p": p.p, "key_len": p.key_len,
    }))
}

#[cfg(feature = "secrets")]
fn decrypt_bytes(payload: &Value) -> Op {
    let (credential, derived) = credential(payload)?;
    let plaintext = labelled(
        payload,
        secrets::decrypt_with(&file_bytes(payload)?, secret_of(&credential, derived)),
    )?;
    Ok(json!({ "plaintext_base64": BASE64.encode(plaintext) }))
}

#[cfg(feature = "secrets")]
fn fields_bytes(payload: &Value) -> Op {
    let (credential, derived) = credential(payload)?;
    let fields = labelled(
        payload,
        secrets::decrypt_fields_with(&file_bytes(payload)?, secret_of(&credential, derived)),
    )?;
    serde_json::to_value(fields).map_err(|e| invalid(e.to_string()))
}

#[cfg(feature = "secrets")]
fn open_bytes(payload: &Value) -> Op {
    let path = payload
        .get("path")
        .and_then(Value::as_str)
        .unwrap_or_default();
    let data = file_bytes(payload)?;
    let (credential, derived) = credential(payload)?;
    let secret = secret_of(&credential, derived);
    let value = if kind_of(payload)? == "root" {
        serde_json::to_value(labelled(
            payload,
            secrets::open_root_with(path, &data, secret),
        )?)
    } else {
        serde_json::to_value(labelled(
            payload,
            secrets::open_boot_with(path, &data, secret),
        )?)
    };
    value.map_err(|e| invalid(e.to_string()))
}

// ===== replication.* =====
//
// Verification outcomes are normal results (`{valid:false, code, message}`) so
// the bindings can map `code` to their own typed errors.

fn repl_head(payload: &Value, key: &str) -> Result<replication::Head, BindingError> {
    let raw = payload.get(key).cloned().unwrap_or(Value::Null);
    serde_json::from_value(raw).map_err(|e| BindingError::invalid(format!("{key}: {e}")))
}

fn repl_failure(f: replication::Failure) -> Value {
    json!({"valid": false, "code": f.code, "message": f.message})
}

fn repl_head_payload(payload: &Value) -> Op {
    let head = repl_head(payload, "head")?;
    let body = head
        .canonical()
        .map_err(|f| BindingError::invalid(f.message))?;
    Ok(json!({"payload_base64": BASE64.encode(body)}))
}

fn repl_head_hash(payload: &Value) -> Op {
    let head = repl_head(payload, "head")?;
    match head.hash() {
        Ok(hash) => Ok(json!({"valid": true, "hash": hash})),
        Err(f) => Ok(repl_failure(f)),
    }
}

fn repl_head_validate(payload: &Value) -> Op {
    let head = repl_head(payload, "head")?;
    match head.validate() {
        Ok(()) => Ok(json!({"valid": true})),
        Err(f) => Ok(repl_failure(f)),
    }
}

fn repl_compare(payload: &Value) -> Op {
    let local = repl_head(payload, "local")?;
    let next = repl_head(payload, "next")?;
    match replication::compare(&local, &next) {
        Ok(result) => Ok(json!({"valid": true, "result": result.as_str()})),
        Err(f) => Ok(json!({
            "valid": false, "result": "diverged", "code": f.code, "message": f.message
        })),
    }
}

fn repl_signed(payload: &Value) -> Result<crate::signing::SignedMessageEnvelope, BindingError> {
    let raw = payload.get("signed").cloned().unwrap_or(Value::Null);
    serde_json::from_value(raw).map_err(|e| BindingError::invalid(format!("signed: {e}")))
}

fn repl_verify_head(payload: &Value) -> Op {
    let signed = repl_signed(payload)?;
    let anchor = payload
        .get("trust_anchor")
        .and_then(Value::as_str)
        .unwrap_or_default();
    let now = payload.get("now_unix").and_then(Value::as_u64).unwrap_or(0);
    match replication::verify_head(&signed, anchor, now) {
        Ok(head) => Ok(json!({"valid": true, "head": head})),
        Err(f) => Ok(repl_failure(f)),
    }
}

fn repl_pull_payload(payload: &Value) -> Op {
    let raw = payload.get("request").cloned().unwrap_or(Value::Null);
    let request: replication::PullRequest =
        serde_json::from_value(raw).map_err(|e| BindingError::invalid(format!("request: {e}")))?;
    let body = serde_json::to_vec(&request)
        .map_err(|e| BindingError::invalid(format!("encode request: {e}")))?;
    Ok(json!({"payload_base64": BASE64.encode(body)}))
}

fn repl_verify_pull(payload: &Value) -> Op {
    let signed = repl_signed(payload)?;
    let allowed: Vec<String> = payload
        .get("allowed")
        .and_then(Value::as_array)
        .map(|a| {
            a.iter()
                .filter_map(|v| v.as_str().map(str::to_owned))
                .collect()
        })
        .unwrap_or_default();
    let now = payload.get("now_unix").and_then(Value::as_u64).unwrap_or(0);
    let skew = payload
        .get("skew_seconds")
        .and_then(Value::as_i64)
        .unwrap_or(0);
    match replication::verify_pull(&signed, &allowed, now, skew) {
        Ok(request) => Ok(json!({"valid": true, "request": request})),
        Err(f) => Ok(repl_failure(f)),
    }
}
