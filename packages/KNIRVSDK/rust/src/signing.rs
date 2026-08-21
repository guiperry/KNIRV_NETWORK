//! Canonical KNIRV Cosmos `SIGN_MODE_DIRECT` and relay-envelope wire encoding.
use crate::error::{Error, Result};
use base64::{engine::general_purpose::STANDARD as BASE64, Engine};
use bech32::{encode, Bech32, Hrp};
use k256::ecdsa::signature::Verifier;
use k256::ecdsa::{signature::hazmat::PrehashSigner, Signature, SigningKey, VerifyingKey};
use ripemd::Ripemd160;
use serde::{Deserialize, Serialize};
use sha2::{Digest, Sha256};

pub const ACTION_SCHEMA_VERSION: &str = "knirv.action.v1";
pub const MESSAGE_SCHEMA_VERSION: &str = "knirv.message.v1";
pub const RELAY_ENVELOPE_SCHEMA_VERSION: &str = "knirv.controller.relay-envelope.v1";
pub const ACTION_TYPE_URL: &str = "/knirv.signing.v1.Action";
pub const SECP256K1_TYPE_URL: &str = "/cosmos.crypto.secp256k1.PubKey";

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
    if compressed_public_key.len() != 33 {
        return Err(Error::Crypto(
            "compressed secp256k1 public key must be 33 bytes".into(),
        ));
    }
    let sha = Sha256::digest(compressed_public_key);
    let digest = Ripemd160::digest(sha);
    encode::<Bech32>(
        Hrp::parse("knirv").map_err(|e| Error::Crypto(e.to_string()))?,
        &digest,
    )
    .map_err(|e| Error::Crypto(e.to_string()))
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
    if signed.address.is_empty() || signed.address != address {
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
                let (length, consumed) = consume_varint(&data[pos..])?;
                let end = pos + consumed + length as usize;
                if end > data.len() {
                    return Err(Error::Validation(
                        "truncated length-delimited field in message envelope".into(),
                    ));
                }
                let field_data = &data[pos + consumed..end];
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
                return Err(Error::Validation(format!(
                    "unsupported wire type {wire_type} in message envelope"
                )));
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
    let mut sign_doc = bytes_field(1, &body);
    sign_doc = bytes_field_extend(sign_doc, 2, &auth);
    let chain_bytes = chain_id.as_bytes();
    let mut chain_field = Vec::new();
    chain_field.extend(varint(3 << 3));
    chain_field.extend(varint(chain_bytes.len() as u64));
    chain_field.extend(chain_bytes);
    sign_doc.extend(chain_field);
    let mut acct_field = Vec::new();
    acct_field.extend(varint(4 << 3));
    acct_field.extend(varint(account_number));
    sign_doc.extend(acct_field);
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
        let (field_len, consumed) = consume_varint(&data[pos..])?;
        pos += consumed + field_len as usize;
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
                let (length, consumed) = consume_varint(&data[pos..])?;
                let end = pos + consumed + length as usize;
                if end > data.len() {
                    return Err(Error::Validation(
                        "truncated length-delimited field in relay envelope".into(),
                    ));
                }
                let field_data = &data[pos + consumed..end];
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
                return Err(Error::Validation(format!(
                    "unsupported wire type {wire_type} in relay envelope"
                )));
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
pub fn decode_address(address: &str, prefix: &str) -> Result<Vec<u8>> {
    let (hrp, data) = bech32::decode(address)
        .map_err(|e| Error::Validation(format!("decode bech32 address: {e}")))?;
    let expected_hrp =
        Hrp::parse(prefix).map_err(|e| Error::Validation(format!("invalid prefix: {e}")))?;
    if hrp != expected_hrp {
        return Err(Error::Validation(format!(
            "unexpected address prefix {:?}",
            hrp
        )));
    }
    let raw = convert_bits(&data, 5, 8, false)
        .map_err(|e| Error::Validation(format!("decode address payload: {e}")))?;
    if raw.len() != 20 {
        return Err(Error::Validation(format!(
            "invalid address payload length {}",
            raw.len()
        )));
    }
    Ok(raw)
}
pub fn address(compressed_pubkey: &[u8], prefix: &str) -> Result<String> {
    if compressed_pubkey.len() != 33 {
        return Err(Error::Crypto(
            "compressed secp256k1 public key must be 33 bytes".into(),
        ));
    }
    let sha = Sha256::digest(compressed_pubkey);
    let digest = Ripemd160::digest(sha);
    let five_bit = convert_bits(&digest, 8, 5, true)
        .map_err(|_| Error::Crypto("convert_bits failed".into()))?;
    let hrp = Hrp::parse(prefix).map_err(|e| Error::Crypto(e.to_string()))?;
    encode::<Bech32>(hrp, &five_bit).map_err(|e| Error::Crypto(e.to_string()))
}

// --- protowire-style helpers ---

fn convert_bits(data: &[u8], from_bits: u32, to_bits: u32, pad: bool) -> Result<Vec<u8>> {
    let mut acc: u32 = 0;
    let mut bits: u32 = 0;
    let mut out = Vec::new();
    let maxv = (1u32 << to_bits) - 1;
    for &byte in data {
        acc = (acc << from_bits) | (byte as u32);
        bits += from_bits;
        while bits >= to_bits {
            bits -= to_bits;
            out.push((acc >> bits) as u8);
        }
    }
    if pad {
        if bits > 0 {
            out.push((acc << (to_bits - bits)) as u8);
        }
    } else if bits >= from_bits || ((acc << (to_bits - bits)) & maxv) != 0 {
        return Err(Error::Validation("invalid bits conversion".into()));
    }
    Ok(out)
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
        if field_number != field || wire_type != 2 {
            let (len, consumed) = consume_varint(&data[pos..])?;
            pos += consumed + len as usize;
            continue;
        }
        let (length, consumed) = consume_varint(&data[pos..])?;
        let end = pos + consumed + length as usize;
        if end > data.len() {
            return Err(Error::Validation("truncated bytes field".into()));
        }
        values.push(data[pos + consumed..end].to_vec());
        pos = end;
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
                let (length, consumed) = consume_varint(&data[pos..])?;
                let end = pos + consumed + length as usize;
                if end > data.len() {
                    return Err(Error::Validation("truncated action field".into()));
                }
                let field_data = &data[pos + consumed..end];
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
                let (len, consumed) = consume_varint(&data[pos..])?;
                pos += consumed + len as usize;
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
        .verify(digest, &sig)
        .map_err(|_| Error::Validation("signature verification failed".into()))?;
    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;
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
}
