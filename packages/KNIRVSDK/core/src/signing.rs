//! Canonical KNIRV Cosmos `SIGN_MODE_DIRECT` and relay-envelope wire encoding.
use crate::error::{Error, Result};
use base64::{engine::general_purpose::STANDARD as BASE64, Engine};
use bech32::{encode, Bech32, Hrp};
use k256::ecdsa::signature::hazmat::PrehashVerifier;
use k256::ecdsa::{signature::hazmat::PrehashSigner, Signature, SigningKey, VerifyingKey};
use ripemd::Ripemd160;
use serde::{Deserialize, Serialize};
use serde_json;
use sha2::{Digest, Sha256};

pub const ACTION_SCHEMA_VERSION: &str = "knirv.action.v1";
pub const MESSAGE_SCHEMA_VERSION: &str = "knirv.message.v1";
pub const RELAY_ENVELOPE_SCHEMA_VERSION: &str = "knirv.controller.relay-envelope.v1";
pub const WASM_PUBLICATION_SCHEMA_VERSION: &str = "knirv.wasm_publication.v1";
pub const WASM_MANIFEST_SCHEMA_VERSION: &str = "knirv.wasm_manifest.v1";
pub const ACTION_TYPE_URL: &str = "/knirv.signing.v1.Action";
pub const SECP256K1_TYPE_URL: &str = "/cosmos.crypto.secp256k1.PubKey";
pub const CONTROLLER_DOMAIN: &str = "knirv.controller";
pub const SCENES_DOMAIN: &str = "knirv.scenes";
pub const PURPOSE_SCENE_MANIFEST: &str = "scene-manifest";
pub const ROLE_SCENE_SIGNER: &str = "scene-signer";
pub const PURPOSE_RELAY_RESPONSE: &str = "relay-response";
pub const RELAY_TARGET_CLI_SUPERVISOR: &str = "cli_supervisor";

/// Registry / bond escrow domain and purposes (root tx-chain design rev 2).
///
/// A registry entry is a signed `MessageEnvelope` (secp256k1) over this domain,
/// reusing the existing signing/canonicalization path — no new crypto. The same
/// key signs holds and releases of the enrollment bond escrow on the Oracle NRN
/// ledger.
pub const REGISTRY_DOMAIN: &str = "knirv.registry";
pub const PURPOSE_ROOT_TX_CHAIN_RECORD: &str = "root-tx-chain-record";
pub const PURPOSE_BOND_HOLD: &str = "bond-hold";
pub const PURPOSE_BOND_RELEASE: &str = "bond-release";

/// Authority / enrollment domain and purposes.
///
/// `bootnode-enroll` is the signed-challenge purpose used for automatic node
/// enrollment (DEC-2 / 2d.2): a NEW boot-node key signs a challenge envelope, and the
/// root owner countersigns (approves) it, binding the new key to the network. This is
/// the approval-to-key binding that lets a node enroll automatically while the manual
/// runbook stays the fallback. Publisher authorization remains a root-owner signature
/// on the registry entry.
pub const AUTHORITY_DOMAIN: &str = "knirv.authority";
pub const PURPOSE_BOOTNODE_ENROLL: &str = "bootnode-enroll";

/// Maximum allowed lifetime of a published root tx-chain registry record. Records
/// whose `expires_at_unix - issued_at_unix` exceeds this are rejected by the core
/// verifier, bounding how long a compromised/stale root record can be served.
pub const REGISTRY_RECORD_MAX_TTL_SECONDS: u64 = 365 * 24 * 60 * 60; // 1 year

/// Prefix of a `knirv`-Bech32 address; used to validate `tx_chain_address`.
pub const KNIRV_ADDRESS_HRP: &str = "knirv";

#[derive(Clone, Debug, Serialize, Deserialize)]
pub struct Action {
    #[serde(default)]
    pub schema_version: String,
    pub action: String,
    pub sender: String,
    pub recipient: Option<String>,
    pub amount: Option<u64>,
    pub payload: Option<Vec<u8>>,
    pub timestamp_unix: u64,
}
#[derive(Clone, Debug, Default, Serialize, Deserialize)]
pub struct Fee {
    pub denom: Option<String>,
    pub amount: Option<String>,
    pub gas_limit: Option<u64>,
    pub payer: Option<String>,
    pub granter: Option<String>,
}
#[derive(Clone, Debug, Serialize, Deserialize)]
pub struct DirectSignRequest {
    pub action: Action,
    pub chain_id: String,
    pub account_number: u64,
    pub sequence: u64,
    #[serde(default)]
    pub fee: Fee,
}
#[derive(Clone, Debug, Serialize, Deserialize)]
pub struct SignedDirectTransaction {
    pub body_bytes: String,
    pub auth_info_bytes: String,
    pub signatures: Vec<String>,
    pub public_key: String,
    pub address: String,
    pub hash: String,
}
#[derive(Clone, Debug, Serialize, Deserialize)]
pub struct RelayEnvelope {
    #[serde(default)]
    pub schema_version: String,
    pub request_id: String,
    pub user_subject: String,
    pub device_id: String,
    pub dve_id: Option<String>,
    pub target_type: String,
    pub target_id: String,
    pub capability: String,
    pub sequence: u64,
    pub lease_epoch: Option<u64>,
    pub issued_at_unix: u64,
    pub expires_at_unix: u64,
    pub payload_digest: String,
}

/// Parameters used to create a canonical CLI-supervisor relay envelope.
#[derive(Clone, Debug, Serialize, Deserialize)]
pub struct CLISupervisorRelayParams {
    pub request_id: String,
    pub user_subject: String,
    pub device_id: String,
    pub target_id: String,
    pub capability: String,
    pub sequence: u64,
    pub lease_epoch: u64,
    pub payload: Vec<u8>,
    pub ttl_seconds: i64,
    /// Issue time. Hosts without a system clock (browser/edge WASM) must supply
    /// it; native callers may leave it unset to use the system clock.
    #[serde(default)]
    pub issued_at_unix: Option<u64>,
}
#[derive(Clone, Debug, Serialize, Deserialize)]
pub struct MessageEnvelope {
    pub schema_version: Option<String>,
    pub domain: String,
    pub purpose: String,
    pub chain_id: String,
    pub nonce: String,
    pub issued_at_unix: u64,
    pub expires_at_unix: u64,
    pub payload: Option<Vec<u8>>,
}
#[derive(Clone, Debug, Serialize, Deserialize)]
pub struct SignedMessageEnvelope {
    pub envelope: String,
    pub signature: String,
    pub public_key: String,
    pub address: String,
}
#[derive(Clone, Debug, Serialize, Deserialize)]
pub struct ParsedMessageEnvelope {
    pub schema_version: String,
    pub domain: String,
    pub purpose: String,
    pub chain_id: String,
    pub nonce: String,
    pub issued_at_unix: u64,
    pub expires_at_unix: u64,
    pub payload: Vec<u8>,
}

/// Canonical registry record published by the root owner for a chain identity.
///
/// The canonical payload is the field-order JSON of this struct (declaration order,
/// matching `Head::canonical`), so Go/TS bindings can reproduce identical bytes.
#[derive(Clone, Debug, Serialize, Deserialize, PartialEq, Eq)]
pub struct RegistryRecord {
    pub chain_id: String,
    pub tx_chain_address: String,
    pub owner_pub_key: String,
    pub nonce: u64,
    pub expires_at_unix: u64,
}

/// Bond hold request payload. Idempotent on `hold_id`: re-submitting the same
/// `hold_id` is a no-op because the ledger keys the escrow by `hold_id`.
#[derive(Clone, Debug, Serialize, Deserialize, PartialEq, Eq)]
pub struct BondHold {
    pub hold_id: String,
    pub operator: String,
    pub amount: String,
    pub root_registry_id: String,
    pub expires_at_unix: u64,
}

/// Bond release payload. `boot_node_address` is bound to the held escrow so a
/// mismatched boot address is rejected at completion time.
#[derive(Clone, Debug, Serialize, Deserialize, PartialEq, Eq)]
pub struct BondRelease {
    pub hold_id: String,
    pub boot_node_chain_id: String,
    pub boot_node_address: String,
}

/// Canonical JSON bytes for a record, in declaration order (deterministic and
/// reproducible by the Go/TS bindings). This is the payload stored in the
/// `MessageEnvelope` and what `verify_message_payload` checks against.
fn canonical_record<T: serde::Serialize>(record: &T) -> Result<Vec<u8>> {
    serde_json::to_vec(record).map_err(|e| Error::Validation(format!("canonicalize record: {e}")))
}

/// Canonical controller publication statement carried in a signed message.
#[derive(Clone, Debug, Default, Serialize, Deserialize, PartialEq, Eq)]
pub struct WasmPublicationPayload {
    #[serde(default)]
    pub schema_version: String,
    pub network_id: String,
    pub network_fingerprint: String,
    pub artifact_digest: String,
    pub byte_size: u64,
    pub module_kind: String,
    #[serde(default)]
    pub abi_version: u64,
    #[serde(default)]
    pub module_schema_version: u64,
    pub build_id: String,
    #[serde(default)]
    pub toolchain_digest: String,
    #[serde(default)]
    pub self_test_digest: String,
    pub publisher_address: String,
    #[serde(default)]
    pub dve_template_id: String,
}

#[derive(Clone, Debug, Default, Serialize, Deserialize, PartialEq, Eq)]
pub struct WasmManifestModule {
    pub module_kind: String,
    pub artifact_digest: String,
    pub byte_size: u64,
    pub abi_version: u64,
    pub module_schema_version: u64,
    #[serde(default)]
    pub capabilities_json: String,
    #[serde(default)]
    pub configuration_digest: String,
    #[serde(default)]
    pub download_path: String,
    #[serde(default)]
    pub publisher_address: String,
    #[serde(default)]
    pub publication_statement_digest: String,
}

#[derive(Clone, Debug, Default, Serialize, Deserialize, PartialEq, Eq)]
pub struct WasmManifestPayload {
    #[serde(default)]
    pub schema_version: String,
    pub manifest_id: String,
    pub network_id: String,
    pub chain_id: String,
    pub network_fingerprint: String,
    pub user_subject: String,
    pub device_id: String,
    #[serde(default)]
    pub dve_id: String,
    #[serde(default)]
    pub lease_epoch: u64,
    pub modules: Vec<WasmManifestModule>,
    #[serde(default)]
    pub relay_target_type: String,
    #[serde(default)]
    pub relay_target_id: String,
    pub assignment_id: String,
    #[serde(default)]
    pub assignment_version: u64,
    #[serde(default)]
    pub supersedes_assignment_id: String,
}

fn varint(mut value: u64) -> Vec<u8> {
    let mut out = Vec::new();
    loop {
        let mut next = (value & 0x7f) as u8;
        value >>= 7;
        if value != 0 {
            next |= 0x80;
        }
        out.push(next);
        if value == 0 {
            return out;
        }
    }
}
fn bytes_field(field: u64, value: &[u8]) -> Vec<u8> {
    if value.is_empty() {
        return vec![];
    }
    let mut out = varint((field << 3) | 2);
    out.extend(varint(value.len() as u64));
    out.extend(value);
    out
}
fn string_field(field: u64, value: Option<&str>) -> Vec<u8> {
    bytes_field(field, value.unwrap_or_default().as_bytes())
}
fn uint_field(field: u64, value: Option<u64>) -> Vec<u8> {
    match value.filter(|v| *v != 0) {
        Some(v) => {
            let mut out = varint(field << 3);
            out.extend(varint(v));
            out
        }
        None => vec![],
    }
}
fn int64_field(field: u64, value: Option<i64>) -> Vec<u8> {
    match value.filter(|v| *v != 0) {
        Some(v) => {
            let mut out = varint(field << 3);
            out.extend(varint(v as u64));
            out
        }
        None => vec![],
    }
}
fn concat(parts: impl IntoIterator<Item = Vec<u8>>) -> Vec<u8> {
    parts.into_iter().flatten().collect()
}
fn any(type_url: &str, value: &[u8]) -> Vec<u8> {
    concat([string_field(1, Some(type_url)), bytes_field(2, value)])
}

fn required(value: &str, field: &str) -> Result<()> {
    if value.trim().is_empty() {
        Err(Error::Validation(format!("{field} is required")))
    } else {
        Ok(())
    }
}

fn normalize_publication(mut payload: WasmPublicationPayload) -> Result<WasmPublicationPayload> {
    if payload.schema_version.is_empty() {
        payload.schema_version = WASM_PUBLICATION_SCHEMA_VERSION.into();
    }
    if payload.schema_version != WASM_PUBLICATION_SCHEMA_VERSION {
        return Err(Error::Validation(
            "unsupported wasm publication schema".into(),
        ));
    }
    for (value, field) in [
        (&payload.network_id, "network_id"),
        (&payload.network_fingerprint, "network_fingerprint"),
        (&payload.artifact_digest, "artifact_digest"),
        (&payload.module_kind, "module_kind"),
        (&payload.build_id, "build_id"),
        (&payload.publisher_address, "publisher_address"),
    ] {
        required(value, field)?;
    }
    if payload.byte_size == 0 {
        return Err(Error::Validation("byte_size is required".into()));
    }
    Ok(payload)
}

/// Marshals the protobuf-wire payload shared with the Go and TypeScript SDKs.
pub fn marshal_wasm_publication_payload(payload: &WasmPublicationPayload) -> Result<Vec<u8>> {
    let p = normalize_publication(payload.clone())?;
    Ok(concat([
        string_field(1, Some(&p.schema_version)),
        string_field(2, Some(&p.network_id)),
        string_field(3, Some(&p.network_fingerprint)),
        string_field(4, Some(&p.artifact_digest)),
        uint_field(5, Some(p.byte_size)),
        string_field(6, Some(&p.module_kind)),
        uint_field(7, Some(p.abi_version)),
        uint_field(8, Some(p.module_schema_version)),
        string_field(9, Some(&p.build_id)),
        string_field(10, Some(&p.toolchain_digest)),
        string_field(11, Some(&p.self_test_digest)),
        string_field(12, Some(&p.publisher_address)),
        string_field(13, Some(&p.dve_template_id)),
    ]))
}

pub fn parse_wasm_publication_payload(data: &[u8]) -> Result<WasmPublicationPayload> {
    let mut out = WasmPublicationPayload::default();
    for_field(data, |number, wire, value| {
        match (number, wire) {
            (1, 2) => out.schema_version = text(value)?,
            (2, 2) => out.network_id = text(value)?,
            (3, 2) => out.network_fingerprint = text(value)?,
            (4, 2) => out.artifact_digest = text(value)?,
            (5, 0) => out.byte_size = number_value(value)?,
            (6, 2) => out.module_kind = text(value)?,
            (7, 0) => out.abi_version = number_value(value)?,
            (8, 0) => out.module_schema_version = number_value(value)?,
            (9, 2) => out.build_id = text(value)?,
            (10, 2) => out.toolchain_digest = text(value)?,
            (11, 2) => out.self_test_digest = text(value)?,
            (12, 2) => out.publisher_address = text(value)?,
            (13, 2) => out.dve_template_id = text(value)?,
            _ => {}
        };
        Ok(())
    })?;
    normalize_publication(out)
}

fn normalize_manifest(mut payload: WasmManifestPayload) -> Result<WasmManifestPayload> {
    if payload.schema_version.is_empty() {
        payload.schema_version = WASM_MANIFEST_SCHEMA_VERSION.into();
    }
    if payload.schema_version != WASM_MANIFEST_SCHEMA_VERSION {
        return Err(Error::Validation("unsupported wasm manifest schema".into()));
    }
    for (value, field) in [
        (&payload.manifest_id, "manifest_id"),
        (&payload.network_id, "network_id"),
        (&payload.chain_id, "chain_id"),
        (&payload.network_fingerprint, "network_fingerprint"),
        (&payload.user_subject, "user_subject"),
        (&payload.device_id, "device_id"),
        (&payload.assignment_id, "assignment_id"),
    ] {
        required(value, field)?;
    }
    if payload.modules.is_empty() {
        return Err(Error::Validation(
            "at least one module entry is required".into(),
        ));
    }
    Ok(payload)
}
fn marshal_wasm_manifest_module(module: &WasmManifestModule) -> Vec<u8> {
    concat([
        string_field(1, Some(&module.module_kind)),
        string_field(2, Some(&module.artifact_digest)),
        uint_field(3, Some(module.byte_size)),
        uint_field(4, Some(module.abi_version)),
        uint_field(5, Some(module.module_schema_version)),
        string_field(6, Some(&module.capabilities_json)),
        string_field(7, Some(&module.configuration_digest)),
        string_field(8, Some(&module.download_path)),
        string_field(9, Some(&module.publisher_address)),
        string_field(10, Some(&module.publication_statement_digest)),
    ])
}
fn parse_wasm_manifest_module(data: &[u8]) -> Result<WasmManifestModule> {
    let mut out = WasmManifestModule::default();
    for_field(data, |number, wire, value| {
        match (number, wire) {
            (1, 2) => out.module_kind = text(value)?,
            (2, 2) => out.artifact_digest = text(value)?,
            (3, 0) => out.byte_size = number_value(value)?,
            (4, 0) => out.abi_version = number_value(value)?,
            (5, 0) => out.module_schema_version = number_value(value)?,
            (6, 2) => out.capabilities_json = text(value)?,
            (7, 2) => out.configuration_digest = text(value)?,
            (8, 2) => out.download_path = text(value)?,
            (9, 2) => out.publisher_address = text(value)?,
            (10, 2) => out.publication_statement_digest = text(value)?,
            _ => {}
        };
        Ok(())
    })?;
    Ok(out)
}
pub fn marshal_wasm_manifest_payload(payload: &WasmManifestPayload) -> Result<Vec<u8>> {
    let p = normalize_manifest(payload.clone())?;
    let mut fields = concat([
        string_field(1, Some(&p.schema_version)),
        string_field(2, Some(&p.manifest_id)),
        string_field(3, Some(&p.network_id)),
        string_field(4, Some(&p.chain_id)),
        string_field(5, Some(&p.network_fingerprint)),
        string_field(6, Some(&p.user_subject)),
        string_field(7, Some(&p.device_id)),
        string_field(8, Some(&p.dve_id)),
        uint_field(9, Some(p.lease_epoch)),
    ]);
    for module in &p.modules {
        fields.extend(bytes_field(10, &marshal_wasm_manifest_module(module)));
    }
    fields.extend(concat([
        string_field(11, Some(&p.relay_target_type)),
        string_field(12, Some(&p.relay_target_id)),
        string_field(13, Some(&p.assignment_id)),
        uint_field(14, Some(p.assignment_version)),
        string_field(15, Some(&p.supersedes_assignment_id)),
    ]));
    Ok(fields)
}
pub fn parse_wasm_manifest_payload(data: &[u8]) -> Result<WasmManifestPayload> {
    let mut out = WasmManifestPayload::default();
    for_field(data, |number, wire, value| {
        match (number, wire) {
            (1, 2) => out.schema_version = text(value)?,
            (2, 2) => out.manifest_id = text(value)?,
            (3, 2) => out.network_id = text(value)?,
            (4, 2) => out.chain_id = text(value)?,
            (5, 2) => out.network_fingerprint = text(value)?,
            (6, 2) => out.user_subject = text(value)?,
            (7, 2) => out.device_id = text(value)?,
            (8, 2) => out.dve_id = text(value)?,
            (9, 0) => out.lease_epoch = number_value(value)?,
            (10, 2) => out.modules.push(parse_wasm_manifest_module(value)?),
            (11, 2) => out.relay_target_type = text(value)?,
            (12, 2) => out.relay_target_id = text(value)?,
            (13, 2) => out.assignment_id = text(value)?,
            (14, 0) => out.assignment_version = number_value(value)?,
            (15, 2) => out.supersedes_assignment_id = text(value)?,
            _ => {}
        };
        Ok(())
    })?;
    normalize_manifest(out)
}

// Minimal protobuf wire reader used by the canonical payload parsers. Unknown
// fields are skipped so future additive fields remain forward compatible.
fn for_field(data: &[u8], mut visitor: impl FnMut(u32, u32, &[u8]) -> Result<()>) -> Result<()> {
    let mut pos = 0;
    while pos < data.len() {
        let (field, wire, consumed) = consume_tag(&data[pos..])?;
        pos += consumed;
        match wire {
            0 => {
                let start = pos;
                let (_, n) = consume_varint(&data[pos..])?;
                pos += n;
                visitor(field, wire, &data[start..pos])?;
            }
            2 => {
                let (start, end) = length_delimited(data, pos)?;
                visitor(field, wire, &data[start..end])?;
                pos = end;
            }
            _ => return Err(Error::Validation(format!("unsupported wire type {wire}"))),
        }
    }
    Ok(())
}
fn text(value: &[u8]) -> Result<String> {
    String::from_utf8(value.to_vec())
        .map_err(|_| Error::Validation("protobuf string is not UTF-8".into()))
}
fn number_value(value: &[u8]) -> Result<u64> {
    let (number, used) = consume_varint(value)?;
    if used != value.len() {
        return Err(Error::Validation("invalid protobuf varint".into()));
    }
    Ok(number)
}

pub fn marshal_action(action: &Action) -> Result<Vec<u8>> {
    let schema = if action.schema_version.is_empty() {
        ACTION_SCHEMA_VERSION
    } else {
        &action.schema_version
    };
    if schema != ACTION_SCHEMA_VERSION {
        return Err(Error::Validation(format!(
            "unsupported action schema: {schema}"
        )));
    }
    if action.action.trim().is_empty()
        || action.sender.trim().is_empty()
        || action.timestamp_unix == 0
    {
        return Err(Error::Validation(
            "action, sender, and a positive timestamp are required".into(),
        ));
    }
    Ok(concat([
        string_field(1, Some(schema)),
        string_field(2, Some(&action.action)),
        string_field(3, Some(&action.sender)),
        string_field(4, action.recipient.as_deref()),
        uint_field(5, action.amount),
        bytes_field(6, action.payload.as_deref().unwrap_or_default()),
        uint_field(7, Some(action.timestamp_unix)),
    ]))
}
fn marshal_fee(fee: &Fee) -> Vec<u8> {
    let coin = concat([
        string_field(1, fee.denom.as_deref()),
        string_field(2, fee.amount.as_deref()),
    ]);
    concat([
        bytes_field(1, &coin),
        uint_field(2, fee.gas_limit),
        string_field(3, fee.payer.as_deref()),
        string_field(4, fee.granter.as_deref()),
    ])
}
/// Builds the exact protobuf sign document used by Go and TypeScript SDKs.
pub fn build_direct_sign_doc(
    request: &DirectSignRequest,
    compressed_public_key: &[u8],
) -> Result<(Vec<u8>, Vec<u8>, Vec<u8>)> {
    if request.chain_id.trim().is_empty() {
        return Err(Error::Validation("chain_id is required".into()));
    }
    if compressed_public_key.len() != 33 {
        return Err(Error::Crypto(
            "compressed secp256k1 public key must be 33 bytes".into(),
        ));
    }
    let action = marshal_action(&request.action)?;
    let body = bytes_field(1, &any(ACTION_TYPE_URL, &action));
    let public_key = any(SECP256K1_TYPE_URL, &bytes_field(1, compressed_public_key));
    let mode_info = bytes_field(1, &uint_field(1, Some(1)));
    let signer_info = concat([
        bytes_field(1, &public_key),
        bytes_field(2, &mode_info),
        uint_field(3, Some(request.sequence)),
    ]);
    let auth_info = concat([
        bytes_field(1, &signer_info),
        bytes_field(2, &marshal_fee(&request.fee)),
    ]);
    let sign_doc = concat([
        bytes_field(1, &body),
        bytes_field(2, &auth_info),
        string_field(3, Some(&request.chain_id)),
        uint_field(4, Some(request.account_number)),
    ]);
    Ok((body, auth_info, sign_doc))
}
pub fn knirv_address(compressed_public_key: &[u8]) -> Result<String> {
    address(compressed_public_key, "knirv")
}
pub fn sign_direct_transaction(
    private_key: &[u8; 32],
    request: &DirectSignRequest,
) -> Result<SignedDirectTransaction> {
    let key =
        SigningKey::from_bytes(private_key.into()).map_err(|e| Error::Crypto(e.to_string()))?;
    let public = key.verifying_key().to_encoded_point(true);
    let public = public.as_bytes();
    let (body, auth, sign_doc) = build_direct_sign_doc(request, public)?;
    // KNIRV signs SHA-256(sign_doc) exactly once, matching Go's SignCompact
    // and the TypeScript SDK's Secp256k1.createSignature call.
    let digest = Sha256::digest(&sign_doc);
    let signature: Signature = key
        .sign_prehash(&digest)
        .map_err(|e| Error::Crypto(e.to_string()))?;
    let signature = signature.to_bytes();
    let raw = concat([
        bytes_field(1, &body),
        bytes_field(2, &auth),
        bytes_field(3, &signature),
    ]);
    Ok(SignedDirectTransaction {
        body_bytes: BASE64.encode(body),
        auth_info_bytes: BASE64.encode(auth),
        signatures: vec![BASE64.encode(signature)],
        public_key: BASE64.encode(public),
        address: knirv_address(public)?,
        hash: hex::encode_upper(Sha256::digest(raw)),
    })
}
pub fn marshal_relay_envelope(envelope: &RelayEnvelope) -> Result<Vec<u8>> {
    let schema = if envelope.schema_version.is_empty() {
        RELAY_ENVELOPE_SCHEMA_VERSION
    } else {
        &envelope.schema_version
    };
    if schema != RELAY_ENVELOPE_SCHEMA_VERSION {
        return Err(Error::Validation(format!(
            "unsupported relay envelope schema: {schema}"
        )));
    }
    if [
        envelope.request_id.as_str(),
        envelope.user_subject.as_str(),
        envelope.device_id.as_str(),
        envelope.target_id.as_str(),
        envelope.capability.as_str(),
        envelope.payload_digest.as_str(),
    ]
    .iter()
    .any(|v| v.trim().is_empty())
    {
        return Err(Error::Validation(
            "relay envelope required fields are missing".into(),
        ));
    }
    if !matches!(
        envelope.target_type.as_str(),
        "dve_expert_advisor" | "cli_supervisor"
    ) {
        return Err(Error::Validation(
            "target_type must be dve_expert_advisor or cli_supervisor".into(),
        ));
    }
    if envelope.sequence == 0
        || envelope.issued_at_unix == 0
        || envelope.expires_at_unix <= envelope.issued_at_unix
    {
        return Err(Error::Validation(
            "relay sequence and validity window are invalid".into(),
        ));
    }
    Ok(concat([
        string_field(1, Some(schema)),
        string_field(2, Some(&envelope.request_id)),
        string_field(3, Some(&envelope.user_subject)),
        string_field(4, Some(&envelope.device_id)),
        string_field(5, envelope.dve_id.as_deref()),
        string_field(6, Some(&envelope.target_type)),
        string_field(7, Some(&envelope.target_id)),
        string_field(8, Some(&envelope.capability)),
        uint_field(9, Some(envelope.sequence)),
        uint_field(10, envelope.lease_epoch),
        int64_field(11, Some(envelope.issued_at_unix as i64)),
        int64_field(12, Some(envelope.expires_at_unix as i64)),
        string_field(13, Some(&envelope.payload_digest)),
    ]))
}

/// Creates a CLI-supervisor relay envelope with the payload bound by SHA-256.
/// A non-positive TTL uses the protocol default of 60 seconds.
/// Seconds since the Unix epoch from the system clock. `wasm32-unknown-unknown`
/// (browsers) has no clock and `SystemTime::now()` would trap there, so it is an
/// error; WASI hosts do provide one.
fn system_unix_seconds() -> Result<u64> {
    #[cfg(all(target_arch = "wasm32", target_os = "unknown"))]
    {
        Err(Error::Validation(
            "this platform has no system clock; pass issued_at_unix".into(),
        ))
    }
    #[cfg(not(all(target_arch = "wasm32", target_os = "unknown")))]
    {
        std::time::SystemTime::now()
            .duration_since(std::time::UNIX_EPOCH)
            .map(|d| d.as_secs())
            .map_err(|e| Error::Validation(format!("system clock is before Unix epoch: {e}")))
    }
}

pub fn new_cli_supervisor_relay(params: &CLISupervisorRelayParams) -> Result<RelayEnvelope> {
    let issued_at_unix = match params.issued_at_unix {
        Some(issued) => issued,
        None => system_unix_seconds()?,
    };
    let ttl_seconds = if params.ttl_seconds > 0 {
        params.ttl_seconds as u64
    } else {
        60
    };
    let expires_at_unix = issued_at_unix
        .checked_add(ttl_seconds)
        .ok_or_else(|| Error::Validation("relay expiration overflows Unix timestamp".into()))?;
    let envelope = RelayEnvelope {
        schema_version: RELAY_ENVELOPE_SCHEMA_VERSION.into(),
        request_id: params.request_id.clone(),
        user_subject: params.user_subject.clone(),
        device_id: params.device_id.clone(),
        dve_id: None,
        target_type: RELAY_TARGET_CLI_SUPERVISOR.into(),
        target_id: params.target_id.clone(),
        capability: params.capability.clone(),
        sequence: params.sequence,
        lease_epoch: Some(params.lease_epoch),
        issued_at_unix,
        expires_at_unix,
        payload_digest: format!("sha256:{}", hex::encode(Sha256::digest(&params.payload))),
    };
    marshal_relay_envelope(&envelope)?;
    Ok(envelope)
}

/// Signs the canonical relay-response message derived from a relay envelope.
pub fn sign_cli_supervisor_relay_response(
    private_key: &[u8; 32],
    envelope: &RelayEnvelope,
    chain_id: &str,
) -> Result<SignedMessageEnvelope> {
    sign_message_envelope(
        private_key,
        &MessageEnvelope {
            schema_version: Some(MESSAGE_SCHEMA_VERSION.into()),
            domain: CONTROLLER_DOMAIN.into(),
            purpose: PURPOSE_RELAY_RESPONSE.into(),
            chain_id: chain_id.into(),
            nonce: envelope.request_id.clone(),
            issued_at_unix: envelope.issued_at_unix,
            expires_at_unix: envelope.expires_at_unix,
            payload: Some(Vec::new()),
        },
    )
}

pub fn marshal_message_envelope(envelope: &MessageEnvelope) -> Result<Vec<u8>> {
    let schema = envelope
        .schema_version
        .as_deref()
        .unwrap_or(MESSAGE_SCHEMA_VERSION);
    if schema != MESSAGE_SCHEMA_VERSION {
        return Err(Error::Validation(format!(
            "unsupported message schema: {schema}"
        )));
    }
    if envelope.domain.trim().is_empty()
        || envelope.purpose.trim().is_empty()
        || envelope.chain_id.trim().is_empty()
        || envelope.nonce.trim().is_empty()
    {
        return Err(Error::Validation(
            "domain, purpose, chainId, and nonce are required".into(),
        ));
    }
    if envelope.issued_at_unix == 0 || envelope.expires_at_unix <= envelope.issued_at_unix {
        return Err(Error::Validation(
            "message envelope validity window is invalid".into(),
        ));
    }
    Ok(concat([
        string_field(1, Some(schema)),
        string_field(2, Some(&envelope.domain)),
        string_field(3, Some(&envelope.purpose)),
        string_field(4, Some(&envelope.chain_id)),
        string_field(5, Some(&envelope.nonce)),
        uint_field(6, Some(envelope.issued_at_unix)),
        uint_field(7, Some(envelope.expires_at_unix)),
        bytes_field(8, envelope.payload.as_deref().unwrap_or_default()),
    ]))
}
pub fn sign_message_envelope(
    private_key: &[u8; 32],
    envelope: &MessageEnvelope,
) -> Result<SignedMessageEnvelope> {
    let bytes = marshal_message_envelope(envelope)?;
    let key =
        SigningKey::from_bytes(private_key.into()).map_err(|e| Error::Crypto(e.to_string()))?;
    let public = key.verifying_key().to_encoded_point(true);
    let public = public.as_bytes();
    let digest = Sha256::digest(&bytes);
    let signature: Signature = key
        .sign_prehash(&digest)
        .map_err(|e| Error::Crypto(e.to_string()))?;
    let signature = signature.to_bytes();
    Ok(SignedMessageEnvelope {
        envelope: BASE64.encode(bytes),
        signature: BASE64.encode(signature),
        public_key: BASE64.encode(public),
        address: knirv_address(public)?,
    })
}
pub fn verify_message(
    signed: &SignedMessageEnvelope,
    expected_domain: &str,
    expected_purpose: &str,
    expected_chain_id: &str,
    expected_nonce: &str,
    now_unix: u64,
) -> Result<()> {
    let envelope = BASE64
        .decode(&signed.envelope)
        .map_err(|e| Error::Validation(format!("decode envelope: {e}")))?;
    let fields = parse_message_envelope(&envelope)?;
    if fields.domain != expected_domain
        || fields.purpose != expected_purpose
        || fields.chain_id != expected_chain_id
        || fields.nonce != expected_nonce
    {
        return Err(Error::Validation(
            "message signing domain does not match request".into(),
        ));
    }
    if now_unix < fields.issued_at_unix.saturating_sub(60) || now_unix > fields.expires_at_unix {
        return Err(Error::Validation(
            "signed message is outside its validity window".into(),
        ));
    }
    let public_key = BASE64
        .decode(&signed.public_key)
        .map_err(|e| Error::Validation(format!("decode public key: {e}")))?;
    if public_key.len() != 33 {
        return Err(Error::Validation(
            "compressed secp256k1 public key must be 33 bytes".into(),
        ));
    }
    let address = knirv_address(&public_key)?;
    if !signed.address.is_empty() && signed.address != address {
        return Err(Error::Validation(
            "message address does not match public key".into(),
        ));
    }
    let raw_signature = BASE64
        .decode(&signed.signature)
        .map_err(|e| Error::Validation(format!("decode signature: {e}")))?;
    if raw_signature.len() != 64 {
        return Err(Error::Validation(
            "signature must use Cosmos 64-byte r|s encoding".into(),
        ));
    }
    let digest = Sha256::digest(&envelope);
    verify_digest(&public_key, &raw_signature, &digest)?;
    Ok(())
}
pub fn verify_message_payload(
    signed: &SignedMessageEnvelope,
    expected_domain: &str,
    expected_purpose: &str,
    expected_chain_id: &str,
    expected_payload: &[u8],
    now_unix: u64,
) -> Result<()> {
    let envelope = BASE64
        .decode(&signed.envelope)
        .map_err(|e| Error::Validation(format!("decode envelope: {e}")))?;
    let fields = parse_message_envelope(&envelope)?;
    verify_message(
        signed,
        expected_domain,
        expected_purpose,
        expected_chain_id,
        &fields.nonce,
        now_unix,
    )?;
    if fields.payload.len() != expected_payload.len()
        || fields
            .payload
            .iter()
            .zip(expected_payload.iter())
            .any(|(a, b)| a != b)
    {
        return Err(Error::Validation(
            "signed message payload does not match request".into(),
        ));
    }
    Ok(())
}



/// Verify a signed envelope of the given domain/purpose/chain_id and decode its
/// canonical payload back into `T`. The signature, domain, purpose, chain_id, nonce
/// replay, and validity window are all checked by `verify_message_payload`; the
/// payload is then decoded and re-canonicalized to guarantee the verifier checks
/// exactly what was signed.
fn verify_signed_payload<T: serde::de::DeserializeOwned>(
    signed: &SignedMessageEnvelope,
    expected_domain: &str,
    expected_purpose: &str,
    expected_chain_id: &str,
    now_unix: u64,
) -> Result<T> {
    let envelope_bytes = BASE64
        .decode(&signed.envelope)
        .map_err(|e| Error::Validation(format!("decode envelope: {e}")))?;
    let fields = parse_message_envelope(&envelope_bytes)?;
    let payload = fields.payload.clone();
    verify_message_payload(
        signed,
        expected_domain,
        expected_purpose,
        expected_chain_id,
        &payload,
        now_unix,
    )?;
    serde_json::from_slice(&payload).map_err(|e| Error::Validation(format!("decode payload: {e}")))
}

// ---------------------------------------------------------------------------
// Registry record signing (CRITICAL fix: unauthenticated POST /register)
// ---------------------------------------------------------------------------

/// Sign a root-tx-chain registry record. The envelope is bound to the registry
/// `chain_id`, the `root-tx-chain-record` purpose, and the record's `nonce`
/// (replay protection). The `tx_chain_address` is a `knirv`-Bech32 address.
pub fn sign_registry_record(
    private_key: &[u8; 32],
    chain_id: &str,
    record: &RegistryRecord,
    issued_at_unix: u64,
) -> Result<SignedMessageEnvelope> {
    let payload = canonical_record(record)?;
    let envelope = MessageEnvelope {
        schema_version: Some(MESSAGE_SCHEMA_VERSION.into()),
        domain: REGISTRY_DOMAIN.into(),
        purpose: PURPOSE_ROOT_TX_CHAIN_RECORD.into(),
        chain_id: chain_id.into(),
        nonce: format!("root-tx-chain:{}", record.nonce),
        issued_at_unix,
        expires_at_unix: record.expires_at_unix,
        payload: Some(payload),
    };
    sign_message_envelope(private_key, &envelope)
}

/// Verify a signed root tx-chain registry record and return its decoded contents.
///
/// In addition to the envelope signature, domain/purpose, chain id, nonce replay
/// and validity window (enforced by `verify_message`/`verify_message_payload`),
/// this enforces the record invariants required to close the unauthenticated
/// `/register` root-address-overwrite issue:
///
///   * `record.chain_id == envelope.chain_id` (no internal inconsistency),
///   * `record.owner_pub_key == signing public key` (self-consistent ownership),
///   * the signer's `knirv` address equals `expected_root_address` (root anchor),
///   * `record.chain_id == expected_root_registry_id` (immutable root registry id),
///   * `tx_chain_address` carries the `knirv` Bech32 HRP (address syntax),
///   * the record lifetime does not exceed `REGISTRY_RECORD_MAX_TTL_SECONDS`.
///
/// `expected_root_address` and `expected_root_registry_id` MUST come from trusted
/// configuration (Worker env / genesis state) — never from the request or record.
/// `now_unix` is the verification time (seconds).
pub fn verify_registry_record(
    signed: &SignedMessageEnvelope,
    expected_root_address: &str,
    expected_root_registry_id: &str,
    now_unix: u64,
) -> Result<RegistryRecord> {
    let envelope = BASE64
        .decode(&signed.envelope)
        .map_err(|e| Error::Validation(format!("decode envelope: {e}")))?;
    let fields = parse_message_envelope(&envelope)?;
    // Verifies signature + chain id (== expected_root_registry_id) + validity window.
    verify_message(
        signed,
        REGISTRY_DOMAIN,
        PURPOSE_ROOT_TX_CHAIN_RECORD,
        expected_root_registry_id,
        &fields.nonce,
        now_unix,
    )?;
    // The decoded record must agree with the envelope's chain id and re-canonicalize
    // to exactly the signed payload (tamper detection on any field).
    let record: RegistryRecord =
        serde_json::from_slice(&fields.payload).map_err(|e| Error::Validation(format!("decode payload: {e}")))?;
    if record.chain_id != fields.chain_id {
        return Err(Error::Validation(
            "record.chain_id does not match envelope chain id".into(),
        ));
    }
    if record.chain_id != expected_root_registry_id {
        return Err(Error::Validation(
            "record chain id is not the immutable root registry id".into(),
        ));
    }
    // Self-consistent ownership: the claimed owner must be the key that signed.
    if record.owner_pub_key != signed.public_key {
        return Err(Error::Validation(
            "record.owner_pub_key does not match the signing key".into(),
        ));
    }
    // Root anchor: the signer must be the trusted root identity.
    let public_key = BASE64
        .decode(&signed.public_key)
        .map_err(|e| Error::Validation(format!("decode public key: {e}")))?;
    if public_key.len() != 33 {
        return Err(Error::Validation("compressed public key must be 33 bytes".into()));
    }
    let signer_address = knirv_address(&public_key)?;
    if signer_address != expected_root_address {
        return Err(Error::Validation("signer is not the trusted root".into()));
    }
    // Address syntax: a real `knirv`-Bech32 address — HRP, separator, case rules,
    // checksum, and a 20-byte payload — not merely a `knirv` prefix. `decode_address`
    // reuses the exact derivation/decoding path used to build the address.
    if decode_address(&record.tx_chain_address, KNIRV_ADDRESS_HRP).is_err() {
        return Err(Error::Validation(
            "tx_chain_address is not a valid knirv Bech32 address".into(),
        ));
    }
    // Bounded lifetime.
    let ttl = record
        .expires_at_unix
        .saturating_sub(fields.issued_at_unix);
    if ttl > REGISTRY_RECORD_MAX_TTL_SECONDS {
        return Err(Error::Validation(
            "registry record TTL exceeds maximum".into(),
        ));
    }
    // The payload must be the canonical record representation.
    let canonical = canonical_record(&record)?;
    if canonical.len() != fields.payload.len()
        || canonical.iter().zip(fields.payload.iter()).any(|(a, b)| a != b)
    {
        return Err(Error::Validation(
            "payload is not the canonical record representation".into(),
        ));
    }
    Ok(record)
}

// ---------------------------------------------------------------------------
// Bond escrow signing (HIGH-1/HIGH-3 fix: idempotent hold + signed release)
// ---------------------------------------------------------------------------

/// Sign a bond-hold request. The envelope is bound to `root_registry_id` and the
/// `bond-hold` purpose; the `hold_id` is the deterministic idempotency key.
pub fn sign_bond_hold(
    private_key: &[u8; 32],
    root_registry_id: &str,
    hold: &BondHold,
    issued_at_unix: u64,
) -> Result<SignedMessageEnvelope> {
    let payload = canonical_record(hold)?;
    let envelope = MessageEnvelope {
        schema_version: Some(MESSAGE_SCHEMA_VERSION.into()),
        domain: REGISTRY_DOMAIN.into(),
        purpose: PURPOSE_BOND_HOLD.into(),
        chain_id: root_registry_id.into(),
        nonce: format!("bond-hold:{}", hold.hold_id),
        issued_at_unix,
        expires_at_unix: hold.expires_at_unix,
        payload: Some(payload),
    };
    sign_message_envelope(private_key, &envelope)
}

/// Verify a signed bond-hold request.
pub fn verify_bond_hold(
    signed: &SignedMessageEnvelope,
    expected_root_registry_id: &str,
    now_unix: u64,
) -> Result<BondHold> {
    verify_signed_payload(
        signed,
        REGISTRY_DOMAIN,
        PURPOSE_BOND_HOLD,
        expected_root_registry_id,
        now_unix,
    )
}

/// Sign a bond-release request. The envelope is bound to the boot node's
/// `chain_id` and the `bond-release` purpose; `boot_node_address` is bound to the
/// held escrow so a mismatched address is rejected at completion.
pub fn sign_bond_release(
    private_key: &[u8; 32],
    boot_node_chain_id: &str,
    release: &BondRelease,
    issued_at_unix: u64,
) -> Result<SignedMessageEnvelope> {
    let payload = canonical_record(release)?;
    let envelope = MessageEnvelope {
        schema_version: Some(MESSAGE_SCHEMA_VERSION.into()),
        domain: REGISTRY_DOMAIN.into(),
        purpose: PURPOSE_BOND_RELEASE.into(),
        chain_id: boot_node_chain_id.into(),
        nonce: format!("bond-release:{}", release.hold_id),
        issued_at_unix,
        expires_at_unix: u64::MAX,
        payload: Some(payload),
    };
    sign_message_envelope(private_key, &envelope)
}

/// Verify a signed bond-release request.
pub fn verify_bond_release(
    signed: &SignedMessageEnvelope,
    expected_boot_node_chain_id: &str,
    now_unix: u64,
) -> Result<BondRelease> {
    verify_signed_payload(
        signed,
        REGISTRY_DOMAIN,
        PURPOSE_BOND_RELEASE,
        expected_boot_node_chain_id,
        now_unix,
    )
}

// ---------------------------------------------------------------------------
// Boot-node enrollment (DEC-2 / 2d.2): SDK-core-bound signed challenge.
//
// Binding rule (DEC-2): the enrollment binds the NEW boot-node key to a specific
// operator AND a nonce/expiry, and the root owner's approval is cryptographically
// bound to that exact challenge (approval-to-key binding). Replay is prevented by
// durable atomic nonce consumption supplied by the host (the core stays pure).
// ---------------------------------------------------------------------------

/// Canonical operator claim carried inside the boot-node enrollment challenge. The
/// `operator` is bound to the challenge signature so a different operator cannot
/// replay or substitute it; `claims` carries opaque operator-specific bytes.
#[derive(Clone, Debug, Serialize, Deserialize, PartialEq, Eq)]
pub struct BootnodeEnrollClaim {
    pub operator: String,
    pub claims: Vec<u8>,
}

/// An approved boot-node enrollment: a challenge signed by the NEW boot-node key,
/// countersigned (approved) by the root owner. The approval's payload is the exact
/// base64 `envelope` string of the challenge, so the root's approval is bound to that
/// specific challenge — this is the "approval-to-key binding".
#[derive(Clone, Debug, Serialize, Deserialize)]
pub struct BootnodeEnrollment {
    pub challenge: SignedMessageEnvelope,
    pub approval: SignedMessageEnvelope,
}

/// Durable atomic nonce-consumption store. The verification core is pure and never
/// persists; the host (Go/TS) supplies an implementation that durably records consumed
/// enrollment nonces so a verified enrollment cannot be replayed after expiry.
pub trait NonceStore {
    /// Mark `nonce` consumed. Returns `Ok(())` on first use, or `Err` if the nonce was
    /// already consumed (replay) or the store is unavailable. Implementations MUST make
    /// this atomic with respect to concurrent verifications.
    fn consume(&mut self, nonce: &str) -> Result<()>;
}

/// In-memory `NonceStore` for tests and single-process hosts. For multi-process
/// durability, implement `NonceStore` against the host's persistent store.
#[derive(Default)]
pub struct MemoryNonceStore {
    used: std::collections::HashSet<String>,
}

impl MemoryNonceStore {
    pub fn new() -> Self {
        Self::default()
    }
    pub fn with_used(used: impl IntoIterator<Item = String>) -> Self {
        Self {
            used: used.into_iter().collect(),
        }
    }
    /// Snapshot of consumed nonces (for hosts that persist the set durably).
    pub fn used(&self) -> impl Iterator<Item = &String> {
        self.used.iter()
    }
    /// Whether `nonce` has already been consumed. Exposed for tests that assert a
    /// failed verification did NOT consume a nonce.
    pub fn contains(&self, nonce: &str) -> bool {
        self.used.contains(nonce)
    }
}

impl NonceStore for MemoryNonceStore {
    fn consume(&mut self, nonce: &str) -> Result<()> {
        if !self.used.insert(nonce.to_string()) {
            return Err(Error::Validation(
                "enrollment nonce already consumed (possible replay)".into(),
            ));
        }
        Ok(())
    }
}

/// The NEW boot-node key signs a challenge (domain `knirv.authority`, purpose
/// `bootnode-enroll`) carrying a canonical `operator` claim and opaque `claims`. The
/// operator is bound into the signed payload, so an enrollment cannot be reused under a
/// different operator. Returns the signed challenge.
pub fn sign_bootnode_enroll_challenge(
    boot_node_key: &[u8; 32],
    chain_id: &str,
    nonce: &str,
    issued_at_unix: u64,
    expires_at_unix: u64,
    operator: &str,
    claims: &[u8],
) -> Result<SignedMessageEnvelope> {
    let claim = BootnodeEnrollClaim {
        operator: operator.to_string(),
        claims: claims.to_vec(),
    };
    let payload = serde_json::to_vec(&claim)?;
    let envelope = MessageEnvelope {
        schema_version: Some(MESSAGE_SCHEMA_VERSION.into()),
        domain: AUTHORITY_DOMAIN.into(),
        purpose: PURPOSE_BOOTNODE_ENROLL.into(),
        chain_id: chain_id.into(),
        nonce: nonce.into(),
        issued_at_unix,
        expires_at_unix,
        payload: Some(payload),
    };
    sign_message_envelope(boot_node_key, &envelope)
}

/// The root owner countersigns (approves) a boot-node challenge. The signed approval
/// envelope's payload is the exact base64 `envelope` string of the challenge, binding
/// the approval to that challenge (approval-to-key binding).
pub fn sign_bootnode_enroll_approval(
    root_key: &[u8; 32],
    chain_id: &str,
    nonce: &str,
    issued_at_unix: u64,
    expires_at_unix: u64,
    challenge_envelope_b64: &str,
) -> Result<SignedMessageEnvelope> {
    let envelope = MessageEnvelope {
        schema_version: Some(MESSAGE_SCHEMA_VERSION.into()),
        domain: AUTHORITY_DOMAIN.into(),
        purpose: PURPOSE_BOOTNODE_ENROLL.into(),
        chain_id: chain_id.into(),
        nonce: nonce.into(),
        issued_at_unix,
        expires_at_unix,
        payload: Some(challenge_envelope_b64.as_bytes().to_vec()),
    };
    sign_message_envelope(root_key, &envelope)
}

/// Verify an approved boot-node enrollment.
///
/// Checks:
///   * the challenge is a valid `knirv.authority`/`bootnode-enroll` envelope (signed by
///     the new boot-node key) within its validity window;
///   * the canonical operator claim inside the challenge equals `expected_operator`
///     (operator-to-key binding — DEC-2);
///   * the enrollment `nonce` is atomically consumed via `nonce_store` (replay
///     prevention); the host persists consumed nonces durably across calls;
///   * the approval is a valid `knirv.authority`/`bootnode-enroll` envelope signed by
///     `expected_root_public_key_b64` whose payload is EXACTLY the challenge's base64
///     envelope string (the approval-to-key binding), on the same chain;
///   * the approval has not expired.
///
/// `expected_root_public_key_b64` MUST come from trusted configuration — never from the
/// request. `now_unix` is the verification time (seconds).
pub fn verify_bootnode_enrollment(
    enrollment: &BootnodeEnrollment,
    expected_root_public_key_b64: &str,
    expected_operator: &str,
    nonce_store: &mut dyn NonceStore,
    now_unix: u64,
) -> Result<()> {
    let challenge_env = BASE64
        .decode(&enrollment.challenge.envelope)
        .map_err(|e| Error::Validation(format!("decode challenge envelope: {e}")))?;
    let challenge = parse_message_envelope(&challenge_env)?;
    verify_message(
        &enrollment.challenge,
        AUTHORITY_DOMAIN,
        PURPOSE_BOOTNODE_ENROLL,
        &challenge.chain_id,
        &challenge.nonce,
        now_unix,
    )?;
    // The operator claim is bound into the signed challenge payload; a substituted or
    // mismatched operator fails verification (operator-to-key binding, DEC-2).
    let claim = serde_json::from_slice::<BootnodeEnrollClaim>(&challenge.payload)
        .map_err(|e| Error::Validation(format!("parse bootnode enroll claim: {e}")))?;
    if claim.operator != expected_operator {
        return Err(Error::Validation(
            "enrollment operator does not match the approved operator".into(),
        ));
    }
    // The approval must be signed by the expected root key and carry the exact
    // challenge envelope bytes as its payload (approval-to-key binding).
    if enrollment.approval.public_key != expected_root_public_key_b64 {
        return Err(Error::Validation(
            "enrollment approval not signed by the trusted root key".into(),
        ));
    }
    verify_message_payload(
        &enrollment.approval,
        AUTHORITY_DOMAIN,
        PURPOSE_BOOTNODE_ENROLL,
        &challenge.chain_id,
        enrollment.challenge.envelope.as_bytes(),
        now_unix,
    )?;
    // Authenticate the full approval→challenge→root binding BEFORE touching the
    // nonce store. Only then consume atomically, so a bad/untrusted approval can
    // never burn the nonce and DoS the valid one (replay-safety, DEC-2).
    nonce_store.consume(&challenge.nonce)?;
    Ok(())
}

/// Verify a boot-node enrollment and return the consumed challenge nonce WITHOUT
/// mutating any store. Hosts (Go/TS) performing durable replay prevention call
/// this to authenticate, then persist the returned nonce through their own atomic
/// `consume-if-absent` store. This keeps the cryptographic verification pure and
/// lets the host own durable state.
pub fn verify_bootnode_enrollment_auth(
    enrollment: &BootnodeEnrollment,
    expected_root_public_key_b64: &str,
    expected_operator: &str,
    now_unix: u64,
) -> Result<String> {
    let mut sink = NullNonceStore;
    verify_bootnode_enrollment(
        enrollment,
        expected_root_public_key_b64,
        expected_operator,
        &mut sink,
        now_unix,
    )?;
    let challenge_env = BASE64
        .decode(&enrollment.challenge.envelope)
        .map_err(|e| Error::Validation(format!("decode challenge envelope: {e}")))?;
    let challenge = parse_message_envelope(&challenge_env)?;
    Ok(challenge.nonce)
}

/// A `NonceStore` that records nothing; used by `verify_bootnode_enrollment_auth`
/// to authenticate without consuming.
struct NullNonceStore;
impl NonceStore for NullNonceStore {
    fn consume(&mut self, _nonce: &str) -> Result<()> {
        Ok(())
    }
}

pub fn parse_message_envelope(data: &[u8]) -> Result<ParsedMessageEnvelope> {
    let mut out = ParsedMessageEnvelope {
        schema_version: String::new(),
        domain: String::new(),
        purpose: String::new(),
        chain_id: String::new(),
        nonce: String::new(),
        issued_at_unix: 0,
        expires_at_unix: 0,
        payload: Vec::new(),
    };
    let mut pos = 0usize;
    while pos < data.len() {
        let (field_number, wire_type, consumed) = consume_tag(&data[pos..])?;
        pos += consumed;
        match wire_type {
            0 => {
                let (value, consumed) = consume_varint(&data[pos..])?;
                pos += consumed;
                match field_number {
                    6 => out.issued_at_unix = value,
                    7 => out.expires_at_unix = value,
                    _ => {}
                }
            }
            2 => {
                let (start, end) = length_delimited(data, pos)?;
                let field_data = &data[start..end];
                pos = end;
                match field_number {
                    1 => out.schema_version = String::from_utf8_lossy(field_data).into_owned(),
                    2 => out.domain = String::from_utf8_lossy(field_data).into_owned(),
                    3 => out.purpose = String::from_utf8_lossy(field_data).into_owned(),
                    4 => out.chain_id = String::from_utf8_lossy(field_data).into_owned(),
                    5 => out.nonce = String::from_utf8_lossy(field_data).into_owned(),
                    8 => out.payload = field_data.to_vec(),
                    _ => {}
                }
            }
            _ => {
                pos += field_value_len(wire_type, &data[pos..])?;
            }
        }
    }
    if out.schema_version != MESSAGE_SCHEMA_VERSION
        || out.domain.is_empty()
        || out.purpose.is_empty()
        || out.chain_id.is_empty()
        || out.nonce.is_empty()
    {
        return Err(Error::Validation(
            "signed message envelope is incomplete".into(),
        ));
    }
    Ok(out)
}
pub fn marshal_tx_raw(body_bytes: &[u8], auth_info_bytes: &[u8], signatures: &[&[u8]]) -> Vec<u8> {
    let mut out = bytes_field(1, body_bytes);
    out = bytes_field_extend(out, 2, auth_info_bytes);
    for sig in signatures {
        out = bytes_field_extend(out, 3, sig);
    }
    out
}
pub fn verify_transaction(
    tx: &SignedDirectTransaction,
    chain_id: &str,
    account_number: u64,
) -> Result<()> {
    if tx.signatures.len() != 1 {
        return Err(Error::Validation(
            "exactly one signature is required".into(),
        ));
    }
    let public_key = BASE64
        .decode(&tx.public_key)
        .map_err(|e| Error::Validation(format!("decode public key: {e}")))?;
    if public_key.len() != 33 {
        return Err(Error::Validation(
            "compressed secp256k1 public key must be 33 bytes".into(),
        ));
    }
    let address = knirv_address(&public_key)?;
    if !tx.address.is_empty() && tx.address != address {
        return Err(Error::Validation(
            "signer address does not match public key".into(),
        ));
    }
    let body = BASE64
        .decode(&tx.body_bytes)
        .map_err(|e| Error::Validation(format!("decode body bytes: {e}")))?;
    let auth = BASE64
        .decode(&tx.auth_info_bytes)
        .map_err(|e| Error::Validation(format!("decode auth info bytes: {e}")))?;
    let sig = BASE64
        .decode(&tx.signatures[0])
        .map_err(|e| Error::Validation(format!("decode signature: {e}")))?;
    // Rebuild the SignDoc with the same encoders signing uses. (This used to
    // hand-roll field 3 with the wrong wire type, so no real transaction ever
    // verified.)
    let sign_doc = concat([
        bytes_field(1, &body),
        bytes_field(2, &auth),
        string_field(3, Some(chain_id)),
        uint_field(4, Some(account_number)),
    ]);
    let digest = Sha256::digest(&sign_doc);
    verify_digest(&public_key, &sig, &digest)?;
    Ok(())
}
pub fn parse_action_body(body: &[u8]) -> Result<Action> {
    let messages = consume_repeated_bytes_field(body, 1)?;
    if messages.len() != 1 {
        return Err(Error::Validation(
            "KNIRV transaction body must contain exactly one action".into(),
        ));
    }
    let type_urls = consume_repeated_bytes_field(&messages[0], 1)?;
    if type_urls.len() != 1 || String::from_utf8_lossy(&type_urls[0]) != ACTION_TYPE_URL {
        return Err(Error::Validation(
            "transaction does not contain a KNIRV action".into(),
        ));
    }
    let values = consume_repeated_bytes_field(&messages[0], 2)?;
    if values.len() != 1 {
        return Err(Error::Validation("KNIRV action payload is missing".into()));
    }
    parse_action(&values[0])
}
pub fn parse_sequence(auth_info: &[u8]) -> Result<u64> {
    let signers = consume_repeated_bytes_field(auth_info, 1)?;
    if signers.len() != 1 {
        return Err(Error::Validation(
            "AuthInfo must contain exactly one signer".into(),
        ));
    }
    let data = &signers[0];
    let mut pos = 0usize;
    while pos < data.len() {
        let (field_number, wire_type, consumed) = consume_tag(&data[pos..])?;
        pos += consumed;
        if field_number == 3 && wire_type == 0 {
            let (value, _consumed) = consume_varint(&data[pos..])?;
            return Ok(value);
        }
        pos += field_value_len(wire_type, &data[pos..])?;
    }
    Err(Error::Validation("SignerInfo sequence is missing".into()))
}
pub fn parse_relay_envelope(data: &[u8]) -> Result<RelayEnvelope> {
    let mut out = RelayEnvelope {
        schema_version: String::new(),
        request_id: String::new(),
        user_subject: String::new(),
        device_id: String::new(),
        dve_id: None,
        target_type: String::new(),
        target_id: String::new(),
        capability: String::new(),
        sequence: 0,
        lease_epoch: None,
        issued_at_unix: 0,
        expires_at_unix: 0,
        payload_digest: String::new(),
    };
    let mut pos = 0usize;
    while pos < data.len() {
        let (field_number, wire_type, consumed) = consume_tag(&data[pos..])?;
        pos += consumed;
        match wire_type {
            0 => {
                let (value, consumed) = consume_varint(&data[pos..])?;
                pos += consumed;
                match field_number {
                    9 => out.sequence = value,
                    10 => out.lease_epoch = Some(value),
                    11 => out.issued_at_unix = value,
                    12 => out.expires_at_unix = value,
                    _ => {}
                }
            }
            2 => {
                let (start, end) = length_delimited(data, pos)?;
                let field_data = &data[start..end];
                pos = end;
                match field_number {
                    1 => out.schema_version = String::from_utf8_lossy(field_data).into_owned(),
                    2 => out.request_id = String::from_utf8_lossy(field_data).into_owned(),
                    3 => out.user_subject = String::from_utf8_lossy(field_data).into_owned(),
                    4 => out.device_id = String::from_utf8_lossy(field_data).into_owned(),
                    5 => out.dve_id = Some(String::from_utf8_lossy(field_data).into_owned()),
                    6 => out.target_type = String::from_utf8_lossy(field_data).into_owned(),
                    7 => out.target_id = String::from_utf8_lossy(field_data).into_owned(),
                    8 => out.capability = String::from_utf8_lossy(field_data).into_owned(),
                    13 => out.payload_digest = String::from_utf8_lossy(field_data).into_owned(),
                    _ => {}
                }
            }
            _ => {
                pos += field_value_len(wire_type, &data[pos..])?;
            }
        }
    }
    if out.schema_version != RELAY_ENVELOPE_SCHEMA_VERSION {
        return Err(Error::Validation(
            "unsupported relay envelope schema".into(),
        ));
    }
    if out.request_id.trim().is_empty()
        || out.user_subject.trim().is_empty()
        || out.device_id.trim().is_empty()
        || out.target_id.trim().is_empty()
        || out.capability.trim().is_empty()
        || out.payload_digest.trim().is_empty()
    {
        return Err(Error::Validation(
            "relay envelope required fields are missing".into(),
        ));
    }
    if out.target_type != "dve_expert_advisor" && out.target_type != "cli_supervisor" {
        return Err(Error::Validation(
            "target_type must be dve_expert_advisor or cli_supervisor".into(),
        ));
    }
    if out.sequence == 0 || out.issued_at_unix == 0 || out.expires_at_unix <= out.issued_at_unix {
        return Err(Error::Validation(
            "relay sequence and validity window are invalid".into(),
        ));
    }
    Ok(out)
}
/// Validates `prefix` and returns the canonical 20-byte account ID.
pub fn decode_address(address: &str, prefix: &str) -> Result<Vec<u8>> {
    let (hrp, raw) = bech32::decode(address)
        .map_err(|e| Error::Validation(format!("decode bech32 address: {e}")))?;
    let expected_hrp =
        Hrp::parse(prefix).map_err(|e| Error::Validation(format!("invalid prefix: {e}")))?;
    if hrp != expected_hrp {
        return Err(Error::Validation(format!(
            "unexpected address prefix {:?}",
            hrp
        )));
    }
    if raw.len() != 20 {
        return Err(Error::Validation(format!(
            "invalid address payload length {}",
            raw.len()
        )));
    }
    Ok(raw)
}

/// Bech32-encodes a 20-byte account ID. `bech32::encode` takes 8-bit bytes and
/// does the 5-bit regrouping itself.
pub fn encode_address(raw: &[u8], prefix: &str) -> Result<String> {
    if raw.len() != 20 {
        return Err(Error::Validation(format!(
            "invalid address payload length {}",
            raw.len()
        )));
    }
    let hrp = Hrp::parse(prefix).map_err(|e| Error::Crypto(e.to_string()))?;
    encode::<Bech32>(hrp, raw).map_err(|e| Error::Crypto(e.to_string()))
}

/// The address of a compressed secp256k1 public key:
/// `bech32(prefix, ripemd160(sha256(pubkey)))`. The key must be a real curve
/// point, not just 33 bytes.
pub fn address(compressed_pubkey: &[u8], prefix: &str) -> Result<String> {
    if compressed_pubkey.len() != 33 {
        return Err(Error::Crypto(
            "compressed secp256k1 public key must be 33 bytes".into(),
        ));
    }
    VerifyingKey::from_sec1_bytes(compressed_pubkey)
        .map_err(|e| Error::Crypto(format!("invalid secp256k1 public key: {e}")))?;
    let sha = Sha256::digest(compressed_pubkey);
    encode_address(&Ripemd160::digest(sha), prefix)
}

// --- protowire-style helpers ---

/// Bounds `(start, end)` of a length-delimited value whose length varint begins
/// at `pos`. Every step is checked: a hostile length must produce an error, never
/// a reversed slice range (which would panic and, in WASM, trap the instance).
fn length_delimited(data: &[u8], pos: usize) -> Result<(usize, usize)> {
    let truncated = || Error::Validation("truncated length-delimited field".into());
    let rest = data.get(pos..).ok_or_else(truncated)?;
    let (length, consumed) = consume_varint(rest)?;
    let start = pos.checked_add(consumed).ok_or_else(truncated)?;
    let end = usize::try_from(length)
        .ok()
        .and_then(|length| start.checked_add(length))
        .filter(|end| *end <= data.len())
        .ok_or_else(truncated)?;
    Ok((start, end))
}

/// Bytes occupied by a field value of `wire_type` at the start of `data`
/// (after its tag), so unknown fields can be skipped correctly.
fn field_value_len(wire_type: u32, data: &[u8]) -> Result<usize> {
    let need = |n: usize| {
        if data.len() >= n {
            Ok(n)
        } else {
            Err(Error::Validation("truncated fixed-width field".into()))
        }
    };
    match wire_type {
        0 => consume_varint(data).map(|(_, used)| used),
        1 => need(8),
        2 => length_delimited(data, 0).map(|(_, end)| end),
        5 => need(4),
        other => Err(Error::Validation(format!("unsupported wire type {other}"))),
    }
}

fn consume_tag(data: &[u8]) -> Result<(u32, u32, usize)> {
    if data.is_empty() {
        return Err(Error::Validation("truncated varint".into()));
    }
    let mut value: u64 = 0;
    let mut shift: u32 = 0;
    let mut i = 0usize;
    loop {
        if i >= data.len() {
            return Err(Error::Validation("truncated varint".into()));
        }
        let byte = data[i];
        value |= ((byte & 0x7f) as u64) << shift;
        i += 1;
        if (byte & 0x80) == 0 {
            break;
        }
        shift += 7;
        if shift >= 64 {
            return Err(Error::Validation("varint overflow".into()));
        }
    }
    let field_number = (value >> 3) as u32;
    let wire_type = (value & 0x7) as u32;
    Ok((field_number, wire_type, i))
}

fn consume_varint(data: &[u8]) -> Result<(u64, usize)> {
    if data.is_empty() {
        return Err(Error::Validation("truncated varint".into()));
    }
    let mut value: u64 = 0;
    let mut shift: u32 = 0;
    let mut i = 0usize;
    loop {
        if i >= data.len() {
            return Err(Error::Validation("truncated varint".into()));
        }
        let byte = data[i];
        value |= ((byte & 0x7f) as u64) << shift;
        i += 1;
        if (byte & 0x80) == 0 {
            return Ok((value, i));
        }
        shift += 7;
        if shift >= 64 {
            return Err(Error::Validation("varint overflow".into()));
        }
    }
}

fn consume_bytes_field(data: &[u8], field: u32) -> Result<Vec<Vec<u8>>> {
    let mut values = Vec::new();
    let mut pos = 0usize;
    while pos < data.len() {
        let (field_number, wire_type, consumed) = consume_tag(&data[pos..])?;
        pos += consumed;
        if field_number == field && wire_type == 2 {
            let (start, end) = length_delimited(data, pos)?;
            values.push(data[start..end].to_vec());
            pos = end;
        } else {
            pos += field_value_len(wire_type, &data[pos..])?;
        }
    }
    Ok(values)
}
fn consume_repeated_bytes_field(data: &[u8], field: u32) -> Result<Vec<Vec<u8>>> {
    consume_bytes_field(data, field)
}

fn parse_action(data: &[u8]) -> Result<Action> {
    let mut action = Action {
        schema_version: String::new(),
        action: String::new(),
        sender: String::new(),
        recipient: None,
        amount: None,
        payload: None,
        timestamp_unix: 0,
    };
    let mut pos = 0usize;
    while pos < data.len() {
        let (field_number, wire_type, consumed) = consume_tag(&data[pos..])?;
        pos += consumed;
        match wire_type {
            2 => {
                let (start, end) = length_delimited(data, pos)?;
                let field_data = &data[start..end];
                pos = end;
                match field_number {
                    1 => action.schema_version = String::from_utf8_lossy(field_data).into_owned(),
                    2 => action.action = String::from_utf8_lossy(field_data).into_owned(),
                    3 => action.sender = String::from_utf8_lossy(field_data).into_owned(),
                    4 => action.recipient = Some(String::from_utf8_lossy(field_data).into_owned()),
                    6 => action.payload = Some(field_data.to_vec()),
                    _ => {}
                }
            }
            0 => {
                let (value, consumed) = consume_varint(&data[pos..])?;
                pos += consumed;
                match field_number {
                    5 => action.amount = Some(value),
                    7 => action.timestamp_unix = value,
                    _ => {}
                }
            }
            _ => {
                pos += field_value_len(wire_type, &data[pos..])?;
            }
        }
    }
    let schema = if action.schema_version.is_empty() {
        ACTION_SCHEMA_VERSION
    } else {
        &action.schema_version
    };
    if schema != ACTION_SCHEMA_VERSION {
        return Err(Error::Validation(format!(
            "unsupported action schema: {schema}"
        )));
    }
    if action.action.trim().is_empty()
        || action.sender.trim().is_empty()
        || action.timestamp_unix == 0
    {
        return Err(Error::Validation(
            "action, sender, and a positive timestamp are required".into(),
        ));
    }
    Ok(action)
}

fn bytes_field_extend(mut out: Vec<u8>, field: u64, value: &[u8]) -> Vec<u8> {
    if value.is_empty() {
        return out;
    }
    out.extend(varint((field << 3) | 2));
    out.extend(varint(value.len() as u64));
    out.extend(value);
    out
}

fn verify_digest(public_key: &[u8], signature: &[u8], digest: &[u8]) -> Result<()> {
    let verifying_key = VerifyingKey::from_encoded_point(
        &k256::EncodedPoint::from_bytes(public_key).map_err(|e| Error::Crypto(e.to_string()))?,
    )
    .map_err(|e| Error::Crypto(e.to_string()))?;
    let sig = k256::ecdsa::Signature::from_bytes(signature.into())
        .map_err(|e| Error::Crypto(e.to_string()))?;
    verifying_key
        .verify_prehash(digest, &sig)
        .map_err(|_| Error::Validation("signature verification failed".into()))?;
    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;
    use bech32::{encode as b32_encode, Bech32, Hrp};
    use serde_json::Value;

    fn test_key() -> [u8; 32] {
        // Deterministic key for tests only.
        [
            0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77, 0x88, 0x99, 0xaa, 0xbb, 0xcc, 0xdd, 0xee,
            0xff, 0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b, 0x0c,
            0x0d, 0x0e, 0x0f, 0x10,
        ]
    }

    use k256::ecdsa::SigningKey;

    /// Derive the signer's base64 compressed public key and `knirv` address from a
    /// 32-byte secret, mirroring what the verifier recovers from the envelope.
    fn signer_identity(key: &[u8; 32]) -> (String, String) {
        let sk = SigningKey::from_bytes(key.as_slice().into()).unwrap();
        let pk = sk.verifying_key().to_encoded_point(true);
        let pk_b64 = BASE64.encode(pk.as_bytes());
        let addr = knirv_address(pk.as_bytes()).unwrap();
        (pk_b64, addr)
    }

    const ROOT_ID: &str = "chain-abc";

    #[test]
    fn registry_record_sign_verify_roundtrip() {
        let key = test_key();
        let (pk_b64, addr) = signer_identity(&key);
        let record = RegistryRecord {
            chain_id: ROOT_ID.into(),
            tx_chain_address: addr.clone(),
            owner_pub_key: pk_b64,
            nonce: 7,
            expires_at_unix: 1_700_000_000 + 600,
        };
        let signed = sign_registry_record(&key, ROOT_ID, &record, 1_700_000_000).expect("sign");
        let verified = verify_registry_record(&signed, &addr, ROOT_ID, 1_700_000_000).expect("verify");
        assert_eq!(verified, record);
    }

    #[test]
    fn registry_record_wrong_chain_id_rejected() {
        let key = test_key();
        let (pk_b64, addr) = signer_identity(&key);
        let record = RegistryRecord {
            chain_id: ROOT_ID.into(),
            tx_chain_address: addr.clone(),
            owner_pub_key: pk_b64,
            nonce: 1,
            expires_at_unix: 1_700_000_000 + 600,
        };
        let signed = sign_registry_record(&key, ROOT_ID, &record, 1_700_000_000).unwrap();
        // Requester expects a different (immutable) root registry id: must be denied.
        assert!(verify_registry_record(&signed, &addr, "chain-other", 1_700_000_000).is_err());
    }

    #[test]
    fn registry_record_tampered_payload_rejected() {
        let key = test_key();
        let (pk_b64, addr) = signer_identity(&key);
        let record = RegistryRecord {
            chain_id: ROOT_ID.into(),
            tx_chain_address: addr.clone(),
            owner_pub_key: pk_b64,
            nonce: 1,
            expires_at_unix: 1_700_000_000 + 600,
        };
        let mut signed = sign_registry_record(&key, ROOT_ID, &record, 1_700_000_000).unwrap();
        // Flip a byte in the encoded envelope -> signature mismatch.
        let mut bytes = BASE64.decode(&signed.envelope).unwrap();
        let idx = bytes.len() - 1;
        bytes[idx] ^= 0xff;
        signed.envelope = BASE64.encode(bytes);
        assert!(verify_registry_record(&signed, &addr, ROOT_ID, 1_700_000_000).is_err());
    }

    #[test]
    fn registry_record_owner_pub_key_mismatch_rejected() {
        let key = test_key();
        let (_pk_b64, addr) = signer_identity(&key);
        // Claim an owner that is not the signing key inside the signed record.
        let record = RegistryRecord {
            chain_id: ROOT_ID.into(),
            tx_chain_address: addr.clone(),
            owner_pub_key: "BASE64NOTTHEOWNER".into(),
            nonce: 1,
            expires_at_unix: 1_700_000_000 + 600,
        };
        let signed = sign_registry_record(&key, ROOT_ID, &record, 1_700_000_000).unwrap();
        assert!(verify_registry_record(&signed, &addr, ROOT_ID, 1_700_000_000).is_err());
    }

    #[test]
    fn registry_record_wrong_root_address_rejected() {
        let key = test_key();
        let (pk_b64, addr) = signer_identity(&key);
        let record = RegistryRecord {
            chain_id: ROOT_ID.into(),
            tx_chain_address: addr.clone(),
            owner_pub_key: pk_b64,
            nonce: 1,
            expires_at_unix: 1_700_000_000 + 600,
        };
        let signed = sign_registry_record(&key, ROOT_ID, &record, 1_700_000_000).unwrap();
        // Verifier trusts a different root identity than the signer.
        assert!(verify_registry_record(&signed, "knirv1someotherroot", ROOT_ID, 1_700_000_000).is_err());
    }

    #[test]
    fn registry_record_address_must_be_valid_knirv_bech32() {
        let key = test_key();
        let (pk_b64, addr) = signer_identity(&key);

        // A real derived `knirv` address is accepted.
        let good = RegistryRecord {
            chain_id: ROOT_ID.into(),
            tx_chain_address: addr.clone(),
            owner_pub_key: pk_b64.clone(),
            nonce: 1,
            expires_at_unix: 1_700_000_000 + 600,
        };
        let signed = sign_registry_record(&key, ROOT_ID, &good, 1_700_000_000).unwrap();
        assert!(verify_registry_record(&signed, &addr, ROOT_ID, 1_700_000_000).is_ok());

        // Not bech32 at all.
        let not_bech32 = RegistryRecord {
            chain_id: ROOT_ID.into(),
            tx_chain_address: "0xabc123".into(),
            owner_pub_key: pk_b64.clone(),
            nonce: 1,
            expires_at_unix: 1_700_000_000 + 600,
        };
        let signed = sign_registry_record(&key, ROOT_ID, &not_bech32, 1_700_000_000).unwrap();
        assert!(verify_registry_record(&signed, &addr, ROOT_ID, 1_700_000_000).is_err());

        // Wrong HRP (a valid bech32, but not `knirv`).
        let wrong_hrp = b32_encode::<Bech32>(Hrp::parse("cosmos").unwrap(), &[0u8; 20]).unwrap();
        let wrong_hrp_rec = RegistryRecord {
            chain_id: ROOT_ID.into(),
            tx_chain_address: wrong_hrp,
            owner_pub_key: pk_b64.clone(),
            nonce: 1,
            expires_at_unix: 1_700_000_000 + 600,
        };
        let signed = sign_registry_record(&key, ROOT_ID, &wrong_hrp_rec, 1_700_000_000).unwrap();
        assert!(verify_registry_record(&signed, &addr, ROOT_ID, 1_700_000_000).is_err());

        // Wrong payload length (valid bech32, but 19 bytes, not 20).
        let wrong_len = b32_encode::<Bech32>(Hrp::parse("knirv").unwrap(), &[0u8; 19]).unwrap();
        let wrong_len_rec = RegistryRecord {
            chain_id: ROOT_ID.into(),
            tx_chain_address: wrong_len,
            owner_pub_key: pk_b64.clone(),
            nonce: 1,
            expires_at_unix: 1_700_000_000 + 600,
        };
        let signed = sign_registry_record(&key, ROOT_ID, &wrong_len_rec, 1_700_000_000).unwrap();
        assert!(verify_registry_record(&signed, &addr, ROOT_ID, 1_700_000_000).is_err());

        // Bad checksum: flip the last character of a valid address.
        let mut corrupted = addr.clone();
        let last = corrupted.pop().unwrap();
        let replacement = if last == '0' { '1' } else { '0' };
        corrupted.push(replacement);
        let bad_checksum = RegistryRecord {
            chain_id: ROOT_ID.into(),
            tx_chain_address: corrupted,
            owner_pub_key: pk_b64.clone(),
            nonce: 1,
            expires_at_unix: 1_700_000_000 + 600,
        };
        let signed = sign_registry_record(&key, ROOT_ID, &bad_checksum, 1_700_000_000).unwrap();
        assert!(verify_registry_record(&signed, &addr, ROOT_ID, 1_700_000_000).is_err());

        // Mixed case is not valid bech32.
        let mut mixed = addr.clone();
        if let Some(ch) = mixed.get_mut(6..7) {
            ch.make_ascii_uppercase();
        }
        let mixed_rec = RegistryRecord {
            chain_id: ROOT_ID.into(),
            tx_chain_address: mixed,
            owner_pub_key: pk_b64.clone(),
            nonce: 1,
            expires_at_unix: 1_700_000_000 + 600,
        };
        let signed = sign_registry_record(&key, ROOT_ID, &mixed_rec, 1_700_000_000).unwrap();
        assert!(verify_registry_record(&signed, &addr, ROOT_ID, 1_700_000_000).is_err());
    }

    #[test]
    fn registry_record_overlong_ttl_rejected() {
        let key = test_key();
        let (pk_b64, addr) = signer_identity(&key);
        let record = RegistryRecord {
            chain_id: ROOT_ID.into(),
            tx_chain_address: addr.clone(),
            owner_pub_key: pk_b64,
            nonce: 1,
            expires_at_unix: 1_700_000_000 + REGISTRY_RECORD_MAX_TTL_SECONDS + 1,
        };
        let signed = sign_registry_record(&key, ROOT_ID, &record, 1_700_000_000).unwrap();
        assert!(verify_registry_record(&signed, &addr, ROOT_ID, 1_700_000_000).is_err());
    }

    #[test]
    fn registry_record_max_ttl_accepted() {
        let key = test_key();
        let (pk_b64, addr) = signer_identity(&key);
        let record = RegistryRecord {
            chain_id: ROOT_ID.into(),
            tx_chain_address: addr.clone(),
            owner_pub_key: pk_b64,
            nonce: 1,
            expires_at_unix: 1_700_000_000 + REGISTRY_RECORD_MAX_TTL_SECONDS,
        };
        let signed = sign_registry_record(&key, ROOT_ID, &record, 1_700_000_000).unwrap();
        assert!(verify_registry_record(&signed, &addr, ROOT_ID, 1_700_000_000).is_ok());
    }

    #[test]
    fn bond_hold_sign_verify_roundtrip_and_bound_to_root() {
        let key = test_key();
        let hold = BondHold {
            hold_id: "hold-1".into(),
            operator: "knirv1operator".into(),
            amount: "1000".into(),
            root_registry_id: "root-reg-xyz".into(),
            expires_at_unix: 1_700_000_000 + 600,
        };
        let signed = sign_bond_hold(&key, "root-reg-xyz", &hold, 1_700_000_000).unwrap();
        let verified =
            verify_bond_hold(&signed, "root-reg-xyz", 1_700_000_000).expect("verify");
        assert_eq!(verified, hold);
        // Bound to a different root registry id -> denied.
        assert!(verify_bond_hold(&signed, "root-reg-other", 1_700_000_000).is_err());
    }

    #[test]
    fn bond_release_sign_verify_roundtrip_and_bound_to_boot_node() {
        let key = test_key();
        let release = BondRelease {
            hold_id: "hold-1".into(),
            boot_node_chain_id: "boot-chain-9".into(),
            boot_node_address: "knirv1boot".into(),
        };
        let signed = sign_bond_release(&key, "boot-chain-9", &release, 1_700_000_000).unwrap();
        let verified = verify_bond_release(&signed, "boot-chain-9", 1_700_000_000).unwrap();
        assert_eq!(verified, release);
        assert!(
            verify_bond_release(&signed, "boot-chain-other", 1_700_000_000).is_err()
        );
    }

    #[test]
    fn scene_manifest_matches_shared_cross_language_golden_vector() {
        let vectors: Value = serde_json::from_str(include_str!(
            "../../testvectors/models/scene_manifest_v1.json"
        ))
        .unwrap();
        let raw_signed = &vectors["signed"];
        let signed = SignedMessageEnvelope {
            envelope: raw_signed["envelope_base64"].as_str().unwrap().to_string(),
            signature: raw_signed["signature_base64"].as_str().unwrap().to_string(),
            public_key: raw_signed["public_key_base64"]
                .as_str()
                .unwrap()
                .to_string(),
            address: raw_signed["address"].as_str().unwrap().to_string(),
        };
        let envelope = &vectors["envelope"];
        assert_eq!(envelope["domain"].as_str().unwrap(), SCENES_DOMAIN);
        assert_eq!(
            envelope["purpose"].as_str().unwrap(),
            PURPOSE_SCENE_MANIFEST
        );
        verify_message(
            &signed,
            SCENES_DOMAIN,
            PURPOSE_SCENE_MANIFEST,
            envelope["chain_id"].as_str().unwrap(),
            envelope["nonce"].as_str().unwrap(),
            vectors["verify_at_unix"].as_u64().unwrap(),
        )
        .unwrap();
        let bytes = BASE64.decode(&signed.envelope).unwrap();
        let fields = parse_message_envelope(&bytes).unwrap();
        assert_eq!(
            fields.payload,
            BASE64
                .decode(envelope["payload_base64"].as_str().unwrap())
                .unwrap()
        );
        assert!(verify_message(
            &signed,
            SCENES_DOMAIN,
            PURPOSE_SCENE_MANIFEST,
            envelope["chain_id"].as_str().unwrap(),
            envelope["nonce"].as_str().unwrap(),
            envelope["expires_at_unix"].as_u64().unwrap() + 1,
        )
        .is_err());
    }

    #[test]
    fn wasm_payloads_match_shared_cross_language_golden_vectors() {
        let vectors: Value =
            serde_json::from_str(include_str!("../../testvectors/wasm_payloads.json")).unwrap();
        let publication: WasmPublicationPayload =
            serde_json::from_value(vectors["wasm_publication_v1"]["input"].clone()).unwrap();
        let publication_bytes = marshal_wasm_publication_payload(&publication).unwrap();
        assert_eq!(
            hex::encode(&publication_bytes),
            vectors["wasm_publication_v1"]["canonical_hex"]
                .as_str()
                .unwrap()
        );
        assert_eq!(
            parse_wasm_publication_payload(&publication_bytes).unwrap(),
            WasmPublicationPayload {
                schema_version: WASM_PUBLICATION_SCHEMA_VERSION.into(),
                ..publication
            }
        );

        let manifest: WasmManifestPayload =
            serde_json::from_value(vectors["wasm_manifest_v1"]["input"].clone()).unwrap();
        let manifest_bytes = marshal_wasm_manifest_payload(&manifest).unwrap();
        assert_eq!(
            hex::encode(&manifest_bytes),
            vectors["wasm_manifest_v1"]["canonical_hex"]
                .as_str()
                .unwrap()
        );
        assert_eq!(
            parse_wasm_manifest_payload(&manifest_bytes).unwrap(),
            WasmManifestPayload {
                schema_version: WASM_MANIFEST_SCHEMA_VERSION.into(),
                ..manifest
            }
        );
    }
    #[test]
    fn canonical_action_wire_format() {
        let got = marshal_action(&Action {
            schema_version: String::new(),
            action: "x".into(),
            sender: "y".into(),
            recipient: None,
            amount: None,
            payload: None,
            timestamp_unix: 1,
        })
        .unwrap();
        assert_eq!(
            hex::encode(got),
            "0a0f6b6e6972762e616374696f6e2e76311201781a01793801"
        );
    }
    #[test]
    fn signs_and_addresses() {
        let signed = sign_direct_transaction(
            &[7; 32],
            &DirectSignRequest {
                action: Action {
                    schema_version: String::new(),
                    action: "send".into(),
                    sender: "knirv1x".into(),
                    recipient: None,
                    amount: Some(1),
                    payload: None,
                    timestamp_unix: 1,
                },
                chain_id: "knirv-1".into(),
                account_number: 1,
                sequence: 1,
                fee: Fee::default(),
            },
        )
        .unwrap();
        assert!(signed.address.starts_with("knirv1"));
        assert_eq!(signed.hash.len(), 64);
    }
    #[test]
    fn signed_transaction_verifies_and_rejects_tampering() {
        let request = DirectSignRequest {
            action: Action {
                schema_version: String::new(),
                action: "transfer".into(),
                sender: "knirv1sender".into(),
                recipient: Some("knirv1recipient".into()),
                amount: Some(42),
                payload: Some(br#"{"memo":"hello"}"#.to_vec()),
                timestamp_unix: 1_750_000_000,
            },
            chain_id: "knirv-testnet-1".into(),
            account_number: 7,
            sequence: 9,
            fee: Fee {
                denom: Some("unrn".into()),
                amount: Some("3".into()),
                gas_limit: Some(200_000),
                payer: None,
                granter: None,
            },
        };
        let mut key = [0u8; 32];
        key[31] = 1;
        let signed = sign_direct_transaction(&key, &request).unwrap();
        assert_eq!(
            signed.address,
            "knirv1w508d6qejxtdg4y5r3zarvary0c5xw7kaqx36e"
        );
        verify_transaction(&signed, "knirv-testnet-1", 7).unwrap();
        assert!(verify_transaction(&signed, "other-chain", 7).is_err());
        assert!(verify_transaction(&signed, "knirv-testnet-1", 8).is_err());
        let mut tampered = signed.clone();
        let mut body = BASE64.decode(&tampered.body_bytes).unwrap();
        *body.last_mut().unwrap() ^= 1;
        tampered.body_bytes = BASE64.encode(body);
        assert!(verify_transaction(&tampered, "knirv-testnet-1", 7).is_err());
    }
    #[test]
    fn message_envelope_round_trip() {
        let env = MessageEnvelope {
            schema_version: Some(MESSAGE_SCHEMA_VERSION.into()),
            domain: "test".into(),
            purpose: "auth".into(),
            chain_id: "knirv-1".into(),
            nonce: "abc123".into(),
            issued_at_unix: 1000,
            expires_at_unix: 2000,
            payload: Some(vec![1, 2, 3]),
        };
        let bytes = marshal_message_envelope(&env).unwrap();
        let parsed = parse_message_envelope(&bytes).unwrap();
        assert_eq!(parsed.domain, "test");
        assert_eq!(parsed.purpose, "auth");
        assert_eq!(parsed.chain_id, "knirv-1");
        assert_eq!(parsed.nonce, "abc123");
        assert_eq!(parsed.issued_at_unix, 1000);
        assert_eq!(parsed.expires_at_unix, 2000);
        assert_eq!(parsed.payload, vec![1, 2, 3]);
    }
    #[test]
    fn relay_envelope_round_trip() {
        let env = RelayEnvelope {
            schema_version: RELAY_ENVELOPE_SCHEMA_VERSION.into(),
            request_id: "req-1".into(),
            user_subject: "user-1".into(),
            device_id: "dev-1".into(),
            dve_id: Some("dve-1".into()),
            target_type: "dve_expert_advisor".into(),
            target_id: "target-1".into(),
            capability: "cap-1".into(),
            sequence: 1,
            lease_epoch: Some(0),
            issued_at_unix: 1000,
            expires_at_unix: 2000,
            payload_digest: "sha256:abc".into(),
        };
        let bytes = marshal_relay_envelope(&env).unwrap();
        let parsed = parse_relay_envelope(&bytes).unwrap();
        assert_eq!(parsed.request_id, "req-1");
        assert_eq!(parsed.target_type, "dve_expert_advisor");
        assert_eq!(parsed.sequence, 1);
    }
    #[test]
    fn cli_supervisor_relay_binds_payload_and_signs_response() {
        let relay = new_cli_supervisor_relay(&CLISupervisorRelayParams {
            request_id: "req-1".into(),
            user_subject: "user-1".into(),
            device_id: "device-1".into(),
            target_id: "supervisor-1".into(),
            capability: "execute".into(),
            sequence: 1,
            lease_epoch: 2,
            payload: b"payload".to_vec(),
            ttl_seconds: 0,
            issued_at_unix: None,
        })
        .unwrap();
        // An explicit issue time is honoured (this is how WASM hosts supply it).
        let fixed = new_cli_supervisor_relay(&CLISupervisorRelayParams {
            request_id: "req-1".into(),
            user_subject: "user-1".into(),
            device_id: "device-1".into(),
            target_id: "supervisor-1".into(),
            capability: "execute".into(),
            sequence: 1,
            lease_epoch: 2,
            payload: b"payload".to_vec(),
            ttl_seconds: 120,
            issued_at_unix: Some(1_750_000_000),
        })
        .unwrap();
        assert_eq!(fixed.issued_at_unix, 1_750_000_000);
        assert_eq!(fixed.expires_at_unix, 1_750_000_120);
        assert_eq!(relay.target_type, RELAY_TARGET_CLI_SUPERVISOR);
        assert_eq!(
            relay.payload_digest,
            "sha256:239f59ed55e737c77147cf55ad0c1b030b6d7ee748a7426952f9b852d5a935e5"
        );
        assert_eq!(relay.expires_at_unix - relay.issued_at_unix, 60);
        let signed =
            sign_cli_supervisor_relay_response(&[7; 32], &relay, "knirv-testnet-1").unwrap();
        verify_message(
            &signed,
            CONTROLLER_DOMAIN,
            PURPOSE_RELAY_RESPONSE,
            "knirv-testnet-1",
            "req-1",
            relay.issued_at_unix,
        )
        .unwrap();
    }

    // --- Boot-node enrollment (DEC-2 / 2d.2) ---

    fn enroll_keys() -> ([u8; 32], [u8; 32]) {
        let boot = test_key();
        let mut root = [0u8; 32];
        root[0] = 0x02; // distinct, valid secp256k1 scalar
        (boot, root)
    }

    #[test]
    fn bootnode_enrollment_roundtrip() {
        let (boot_key, root_key) = enroll_keys();
        let (root_pk, _) = signer_identity(&root_key);
        let now = 1_700_000_000u64;
        let challenge = sign_bootnode_enroll_challenge(
            &boot_key,
            ROOT_ID,
            "enroll:1",
            now,
            now + 600,
            "operator:alice",
            b"boot-node-claims",
        )
        .expect("challenge");
        let approval = sign_bootnode_enroll_approval(
            &root_key,
            ROOT_ID,
            "enroll:1",
            now,
            now + 600,
            &challenge.envelope,
        )
        .expect("approval");
        let enrollment = BootnodeEnrollment { challenge, approval };
        let mut store = MemoryNonceStore::new();
        verify_bootnode_enrollment(&enrollment, &root_pk, "operator:alice", &mut store, now)
            .expect("verify");
    }

    #[test]
    fn bootnode_enrollment_wrong_root_rejected() {
        let (boot_key, root_key) = enroll_keys();
        let mut other_root = [0u8; 32];
        other_root[0] = 0x03; // a different, valid root key
        let (other_root_pk, _) = signer_identity(&other_root);
        let now = 1_700_000_000u64;
        let challenge = sign_bootnode_enroll_challenge(
            &boot_key,
            ROOT_ID,
            "enroll:1",
            now,
            now + 600,
            "operator:alice",
            b"x",
        )
        .unwrap();
        let approval =
            sign_bootnode_enroll_approval(&root_key, ROOT_ID, "enroll:1", now, now + 600, &challenge.envelope)
                .unwrap();
        let enrollment = BootnodeEnrollment { challenge, approval };
        let mut store = MemoryNonceStore::new();
        // Approval signed by an untrusted root key must be rejected.
        assert!(verify_bootnode_enrollment(&enrollment, &other_root_pk, "operator:alice", &mut store, now).is_err());
    }

    #[test]
    fn bootnode_enrollment_tampered_challenge_rejected() {
        let (boot_key, root_key) = enroll_keys();
        let (root_pk, _) = signer_identity(&root_key);
        let now = 1_700_000_000u64;
        let challenge = sign_bootnode_enroll_challenge(
            &boot_key,
            ROOT_ID,
            "enroll:1",
            now,
            now + 600,
            "operator:alice",
            b"x",
        )
        .unwrap();
        // Root approves a DIFFERENT challenge string -> the approval-to-key binding must fail.
        let approval = sign_bootnode_enroll_approval(
            &root_key,
            ROOT_ID,
            "enroll:1",
            now,
            now + 600,
            "tampered-challenge-envelope",
        )
        .unwrap();
        let enrollment = BootnodeEnrollment { challenge, approval };
        let mut store = MemoryNonceStore::new();
        assert!(verify_bootnode_enrollment(&enrollment, &root_pk, "operator:alice", &mut store, now).is_err());
    }

    #[test]
    fn bootnode_enrollment_wrong_operator_rejected() {
        let (boot_key, root_key) = enroll_keys();
        let (root_pk, _) = signer_identity(&root_key);
        let now = 1_700_000_000u64;
        // Challenge is signed for operator:alice.
        let challenge = sign_bootnode_enroll_challenge(
            &boot_key,
            ROOT_ID,
            "enroll:1",
            now,
            now + 600,
            "operator:alice",
            b"x",
        )
        .unwrap();
        let approval = sign_bootnode_enroll_approval(
            &root_key,
            ROOT_ID,
            "enroll:1",
            now,
            now + 600,
            &challenge.envelope,
        )
        .unwrap();
        let enrollment = BootnodeEnrollment { challenge, approval };
        let mut store = MemoryNonceStore::new();
        // Verifier expects operator:bob -> operator binding must fail.
        assert!(verify_bootnode_enrollment(&enrollment, &root_pk, "operator:bob", &mut store, now).is_err());
    }

    #[test]
    fn bootnode_enrollment_replay_rejected() {
        let (boot_key, root_key) = enroll_keys();
        let (root_pk, _) = signer_identity(&root_key);
        let now = 1_700_000_000u64;
        let challenge = sign_bootnode_enroll_challenge(
            &boot_key,
            ROOT_ID,
            "enroll:1",
            now,
            now + 600,
            "operator:alice",
            b"x",
        )
        .unwrap();
        let approval = sign_bootnode_enroll_approval(
            &root_key,
            ROOT_ID,
            "enroll:1",
            now,
            now + 600,
            &challenge.envelope,
        )
        .unwrap();
        let enrollment = BootnodeEnrollment { challenge, approval };
        let mut store = MemoryNonceStore::new();
        // First verification consumes the nonce and succeeds.
        verify_bootnode_enrollment(&enrollment, &root_pk, "operator:alice", &mut store, now)
            .expect("first verify");
        // Replaying the same enrollment (same nonce) must be rejected.
        assert!(verify_bootnode_enrollment(&enrollment, &root_pk, "operator:alice", &mut store, now).is_err());
    }

    #[test]
    fn bootnode_enrollment_invalid_approval_does_not_consume_nonce() {
        let (boot_key, root_key) = enroll_keys();
        let mut other_root = [0u8; 32];
        other_root[0] = 0x03; // a different, valid (but untrusted) root key
        let (other_root_pk, _) = signer_identity(&other_root);
        let (root_pk, _) = signer_identity(&root_key);
        let now = 1_700_000_000u64;
        let challenge = sign_bootnode_enroll_challenge(
            &boot_key,
            ROOT_ID,
            "enroll:1",
            now,
            now + 600,
            "operator:alice",
            b"x",
        )
        .unwrap();
        let approval =
            sign_bootnode_enroll_approval(&root_key, ROOT_ID, "enroll:1", now, now + 600, &challenge.envelope)
                .unwrap();
        let challenge_env = challenge.envelope.clone();
        let enrollment = BootnodeEnrollment { challenge, approval };
        let mut store = MemoryNonceStore::new();
        // A bad approval (signed by an untrusted root) is rejected BEFORE the nonce is
        // touched, so a later valid approval with the same nonce can still succeed.
        assert!(verify_bootnode_enrollment(&enrollment, &other_root_pk, "operator:alice", &mut store, now).is_err());
        assert!(!store.contains("enroll:1"), "invalid approval must not consume the nonce");

        // The same nonce is therefore still consumable by a valid enrollment.
        let valid = BootnodeEnrollment {
            challenge: sign_bootnode_enroll_challenge(
                &boot_key,
                ROOT_ID,
                "enroll:1",
                now,
                now + 600,
                "operator:alice",
                b"x",
            )
            .unwrap(),
            approval: sign_bootnode_enroll_approval(&root_key, ROOT_ID, "enroll:1", now, now + 600, &challenge_env)
                .unwrap(),
        };
        verify_bootnode_enrollment(&valid, &root_pk, "operator:alice", &mut store, now)
            .expect("valid enrollment must still verify after a prior invalid one");
    }
}

#[cfg(test)]
mod address_tests {
    use super::*;

    fn pubkey(n: u8) -> Vec<u8> {
        let mut key = [0u8; 32];
        key[31] = n;
        let sk = SigningKey::from_bytes((&key).into()).unwrap();
        sk.verifying_key()
            .to_encoded_point(true)
            .as_bytes()
            .to_vec()
    }

    #[test]
    fn address_matches_the_canonical_vector_and_roundtrips() {
        // Same address the Go and TypeScript SDKs derive for private key 1.
        let canonical = "knirv1w508d6qejxtdg4y5r3zarvary0c5xw7kaqx36e";
        assert_eq!(address(&pubkey(1), "knirv").unwrap(), canonical);
        assert_eq!(knirv_address(&pubkey(1)).unwrap(), canonical);
        let raw = decode_address(canonical, "knirv").unwrap();
        assert_eq!(raw.len(), 20);
        assert_eq!(encode_address(&raw, "knirv").unwrap(), canonical);
    }

    #[test]
    fn address_rejects_bad_input() {
        assert!(address(&[2u8; 32], "knirv").is_err()); // wrong length
        assert!(address(&[0u8; 33], "knirv").is_err()); // 33 bytes, not a curve point
        assert!(decode_address("knirv1w508d6qejxtdg4y5r3zarvary0c5xw7kaqx36e", "cosmos").is_err());
        assert!(decode_address("not-bech32", "knirv").is_err());
        assert!(encode_address(&[1u8; 19], "knirv").is_err());
    }
}

#[cfg(test)]
mod hostile_input_tests {
    use super::*;

    /// Every parser that accepts attacker-supplied bytes. None may panic: in
    /// WASM a panic traps the whole instance (`panic = "abort"`).
    fn parse_all(data: &[u8]) {
        let _ = parse_message_envelope(data);
        let _ = parse_relay_envelope(data);
        let _ = parse_action_body(data);
        let _ = parse_sequence(data);
        let _ = parse_wasm_publication_payload(data);
        let _ = parse_wasm_manifest_payload(data);
    }

    #[test]
    fn hostile_lengths_and_wire_types_error_instead_of_panicking() {
        let max = [0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x01];
        let mut cases: Vec<Vec<u8>> = vec![
            vec![],
            vec![0x08],          // tag with no value
            vec![0x0a],          // length-delimited with no length
            vec![0x0a, 0x05, 1], // length past the end
            vec![0x12, 0x00],    // empty length-delimited
            vec![0x1a, 0x80],    // unterminated varint
            vec![0x09, 1, 2, 3], // fixed64 with 3 bytes
            vec![0x0d, 1],       // fixed32 with 1 byte
            vec![0x0b],          // start-group
            vec![
                0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x01,
            ], // tag overflow
        ];
        for tag in [0x0a, 0x12, 0x1a, 0x22] {
            let mut huge = vec![tag];
            huge.extend_from_slice(&max); // length = 2^64-1
            cases.push(huge);
            let mut near = vec![tag, 0xff, 0xff, 0xff, 0xff, 0x0f]; // length = 2^32-1
            near.push(1);
            cases.push(near);
        }
        for case in &cases {
            parse_all(case);
        }
        assert!(parse_message_envelope(&cases[3]).is_err());
        assert!(parse_action_body(&cases[3]).is_err());
    }

    #[test]
    fn deterministic_fuzz_never_panics() {
        let mut state = 0x9e37_79b9_7f4a_7c15u64;
        let mut next = move || {
            state ^= state << 13;
            state ^= state >> 7;
            state ^= state << 17;
            state
        };
        for _ in 0..20_000 {
            let len = (next() % 80) as usize;
            let data: Vec<u8> = (0..len).map(|_| next() as u8).collect();
            parse_all(&data);
        }
    }

    #[test]
    fn unknown_fields_of_every_wire_type_are_skipped() {
        let envelope = MessageEnvelope {
            schema_version: None,
            domain: "d".into(),
            purpose: "p".into(),
            chain_id: "c".into(),
            nonce: "n".into(),
            issued_at_unix: 1_750_000_000,
            expires_at_unix: 1_750_000_600,
            payload: Some(b"hi".to_vec()),
        };
        let mut wire = marshal_message_envelope(&envelope).unwrap();
        wire.extend([0x78, 0x2a]); // field 15 varint 42
        wire.extend([0x7d, 1, 2, 3, 4]); // field 15 fixed32
        wire.extend([0x79, 1, 2, 3, 4, 5, 6, 7, 8]); // field 15 fixed64
        wire.extend([0x7a, 0x02, 9, 9]); // field 15 length-delimited
        let parsed = parse_message_envelope(&wire).unwrap();
        assert_eq!(parsed.domain, "d");
        assert_eq!(parsed.nonce, "n");
        assert_eq!(parsed.payload, b"hi");
    }
}
