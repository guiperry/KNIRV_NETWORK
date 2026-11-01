//! KNIRV authority records: action digests, 80-byte proof-of-work headers,
//! session certificates, signed records and their verification.
//!
//! This module is the source of truth for the authority contract (see
//! `docs/AUTHORITY_SPEC.md`). The Go and TypeScript packages reach it
//! through the binding operations in [`crate::bindings`]; the shared vectors in
//! `testvectors/authority/` are generated and verified from here.
use crate::error::Error;
use crate::signing::{
    self, knirv_address, marshal_message_envelope, sign_message_envelope, MessageEnvelope,
    SignedMessageEnvelope,
};
use base64::{engine::general_purpose::STANDARD as BASE64, Engine};
use k256::ecdsa::SigningKey;
use serde::{Deserialize, Serialize};
use sha2::{Digest, Sha256};
use std::collections::BTreeMap;
use std::fmt;

pub const DIGEST_DOMAIN: &[u8] = b"KNIRV-AUTH-V2\x00";
pub const SESSION_CERT_DOMAIN: &[u8] = b"KNIRV-SESSION-CERT-V1\x00";
pub const AUTHORITY_DOMAIN: &str = "knirv.authority";
pub const PURPOSE_AUTHORITY_RECORD: &str = "authority-record";
pub const PURPOSE_SESSION_CERT: &str = "session-cert";
pub const AUTHORITY_CHAIN_ID: &str = "knirv-authority-1";
pub const HEADER_VERSION: u8 = 2;
pub const WITNESS_SOFTWARE: &str = "sw";
pub const WITNESS_ASIC: &str = "asic";

/// Failure classes. [`Failure::code`] names are part of the cross-language
/// contract (`testvectors/authority/tamper.json`).
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum Failure {
    UnknownOperator,
    ExpiredSession,
    SessionSignature,
    RecordSignature,
    Header,
    Witness,
    Target,
    Scope,
    Tier,
    Invalid,
}

impl Failure {
    pub fn code(self) -> &'static str {
        match self {
            Failure::UnknownOperator => "ErrUnknownOperator",
            Failure::ExpiredSession => "ErrExpiredSession",
            Failure::SessionSignature => "ErrSessionSignature",
            Failure::RecordSignature => "ErrRecordSignature",
            Failure::Header => "ErrHeader",
            Failure::Witness => "ErrWitness",
            Failure::Target => "ErrTarget",
            Failure::Scope => "ErrScope",
            Failure::Tier => "ErrTier",
            Failure::Invalid => "ErrInvalid",
        }
    }
    fn text(self) -> &'static str {
        match self {
            Failure::UnknownOperator => "unknown authority operator",
            Failure::ExpiredSession => "session certificate is not valid now",
            Failure::SessionSignature => "session certificate signature is invalid",
            Failure::RecordSignature => "authority record signature is invalid",
            Failure::Header => "authority header is invalid",
            Failure::Witness => "authority witness is invalid",
            Failure::Target => "authority target does not match tier",
            Failure::Scope => "authority scope is not covered",
            Failure::Tier => "authority tier is not allowed",
            Failure::Invalid => "invalid authority input",
        }
    }
}

#[derive(Clone, Debug, PartialEq, Eq)]
pub struct AuthorityError {
    pub kind: Failure,
    pub detail: String,
}

impl AuthorityError {
    fn new(kind: Failure, detail: impl Into<String>) -> Self {
        Self {
            kind,
            detail: detail.into(),
        }
    }
    fn invalid(detail: impl Into<String>) -> Self {
        Self::new(Failure::Invalid, detail)
    }
}

impl fmt::Display for AuthorityError {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        if self.detail.is_empty() {
            write!(f, "{}", self.kind.text())
        } else {
            write!(f, "{}: {}", self.kind.text(), self.detail)
        }
    }
}

impl std::error::Error for AuthorityError {}

impl From<Error> for AuthorityError {
    fn from(error: Error) -> Self {
        Self::invalid(error.to_string())
    }
}

pub type AuthResult<T> = std::result::Result<T, AuthorityError>;

/// Authority tier (`ASIC_Authority.md` §4.2).
#[derive(Clone, Copy, Debug, PartialEq, Eq, PartialOrd, Ord, Serialize, Deserialize)]
#[serde(try_from = "u8", into = "u8")]
pub enum Tier {
    /// Reads and user-scoped operations: no witness, no record.
    T0 = 0,
    /// Own-scope event, evidence and badge writes: software witness.
    T1 = 1,
    /// Revoke, issuance, admission, checkpoint: root only.
    T2 = 2,
    /// Rotation, verifier set, slashing, policy: root plus quorum.
    T3 = 3,
}

impl TryFrom<u8> for Tier {
    type Error = String;
    fn try_from(value: u8) -> std::result::Result<Self, String> {
        match value {
            0 => Ok(Tier::T0),
            1 => Ok(Tier::T1),
            2 => Ok(Tier::T2),
            3 => Ok(Tier::T3),
            other => Err(format!("invalid authority tier {other}")),
        }
    }
}

impl From<Tier> for u8 {
    fn from(tier: Tier) -> u8 {
        tier as u8
    }
}

impl fmt::Display for Tier {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        write!(f, "T{}", *self as u8)
    }
}

/// Minimum witness difficulty for a tier. Lower `nbits` values are harder. T0
/// has no witness and returns 0.
pub fn tier_target_bits(tier: Tier) -> u32 {
    match tier {
        Tier::T1 => 0x207f_ffff,
        Tier::T2 => 0x1f00_ffff,
        Tier::T3 => 0x1e00_ffff,
        Tier::T0 => 0,
    }
}

fn append_lp(dst: &mut Vec<u8>, value: &[u8]) {
    dst.extend_from_slice(&(value.len() as u32).to_be_bytes());
    dst.extend_from_slice(value);
}

/// `SHA256("KNIRV-AUTH-V2\0" ‖ tier(1) ‖ lp(domain) ‖ lp(scope) ‖ lp(operator) ‖
/// epoch(8,BE) ‖ canonical_action_hash(32) ‖ prev_record_hash(32))`.
pub fn action_digest(
    tier: Tier,
    domain: &str,
    scope: &str,
    operator: &str,
    epoch: u64,
    canonical_action_hash: &[u8; 32],
    prev_record_hash: &[u8; 32],
) -> [u8; 32] {
    let mut buf = Vec::with_capacity(160);
    buf.extend_from_slice(DIGEST_DOMAIN);
    buf.push(tier as u8);
    append_lp(&mut buf, domain.as_bytes());
    append_lp(&mut buf, scope.as_bytes());
    append_lp(&mut buf, operator.as_bytes());
    buf.extend_from_slice(&epoch.to_be_bytes());
    buf.extend_from_slice(canonical_action_hash);
    buf.extend_from_slice(prev_record_hash);
    Sha256::digest(&buf).into()
}

/// Packs the header's 4-byte version/tier field.
pub fn version_tier(version: u8, tier: Tier) -> u32 {
    (version as u32) << 8 | tier as u32
}

/// The 80-byte proof-of-work header:
/// `version/tier(4) ‖ prevhash(32) ‖ merkle(32) ‖ ntime(4) ‖ nbits(4) ‖ nonce(4)`.
#[derive(Clone, Copy, PartialEq, Eq)]
pub struct Header(pub [u8; 80]);

impl fmt::Debug for Header {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        write!(f, "Header({})", hex::encode(self.0))
    }
}

impl Header {
    pub fn new(
        prev_hash: &[u8; 32],
        merkle: &[u8; 32],
        ntime: u32,
        nbits: u32,
        nonce: u32,
        version: u8,
        tier: Tier,
    ) -> Self {
        let mut h = [0u8; 80];
        h[0..4].copy_from_slice(&version_tier(version, tier).to_be_bytes());
        h[4..36].copy_from_slice(prev_hash);
        h[36..68].copy_from_slice(merkle);
        h[68..72].copy_from_slice(&ntime.to_be_bytes());
        h[72..76].copy_from_slice(&nbits.to_be_bytes());
        h[76..80].copy_from_slice(&nonce.to_be_bytes());
        Header(h)
    }
    fn word(&self, at: usize) -> u32 {
        u32::from_be_bytes(self.0[at..at + 4].try_into().expect("4-byte slice"))
    }
    pub fn version(&self) -> u8 {
        (self.word(0) >> 8) as u8
    }
    pub fn tier_byte(&self) -> u8 {
        (self.word(0) & 0xff) as u8
    }
    pub fn prev_hash(&self) -> [u8; 32] {
        self.0[4..36].try_into().expect("32-byte slice")
    }
    pub fn merkle(&self) -> [u8; 32] {
        self.0[36..68].try_into().expect("32-byte slice")
    }
    pub fn ntime(&self) -> u32 {
        self.word(68)
    }
    pub fn nbits(&self) -> u32 {
        self.word(72)
    }
    pub fn nonce(&self) -> u32 {
        self.word(76)
    }
    /// `sha256d(header)`, the value compared with the target.
    pub fn witness(&self) -> [u8; 32] {
        let first = Sha256::digest(self.0);
        Sha256::digest(first).into()
    }
}

/// Expands Bitcoin's compact `nbits` into a 256-bit big-endian target.
pub fn target_from_bits(bits: u32) -> AuthResult<[u8; 32]> {
    let exponent = bits >> 24;
    let mantissa = bits & 0x007f_ffff;
    if bits & 0x0080_0000 != 0 {
        return Err(AuthorityError::invalid("target is negative"));
    }
    if mantissa == 0 {
        return Err(AuthorityError::invalid("target is zero"));
    }
    let mut target = [0u8; 32];
    if exponent <= 3 {
        let value = mantissa >> (8 * (3 - exponent));
        if value == 0 {
            return Err(AuthorityError::invalid("target overflows 256 bits"));
        }
        target[28..32].copy_from_slice(&value.to_be_bytes());
        return Ok(target);
    }
    let shift = 8 * (exponent - 3);
    let bit_len = 32 - mantissa.leading_zeros();
    if bit_len as u64 + shift as u64 > 256 {
        return Err(AuthorityError::invalid("target overflows 256 bits"));
    }
    let offset = (exponent - 3) as usize; // bytes from the right
    for i in 0..3usize {
        let byte = ((mantissa >> (8 * i)) & 0xff) as u8;
        if byte == 0 {
            continue;
        }
        // bit_len + shift <= 256 guarantees offset + i <= 31.
        target[31 - offset - i] = byte;
    }
    Ok(target)
}

/// Reports whether `witness` (interpreted as Bitcoin's little-endian uint256)
/// is at or below the compact target.
pub fn meets_target(witness: &[u8; 32], bits: u32) -> bool {
    let Ok(target) = target_from_bits(bits) else {
        return false;
    };
    let mut value = *witness;
    value.reverse();
    value <= target
}

/// Searches for a header nonce satisfying `nbits`, starting at `start_nonce`.
pub fn mine(
    prev_hash: &[u8; 32],
    merkle: &[u8; 32],
    ntime: u32,
    nbits: u32,
    start_nonce: u32,
    version: u8,
    tier: Tier,
) -> AuthResult<(Header, u32)> {
    target_from_bits(nbits)?;
    let mut nonce = start_nonce;
    for _ in 0..=(1u64 << 32) {
        let header = Header::new(prev_hash, merkle, ntime, nbits, nonce, version, tier);
        if meets_target(&header.witness(), nbits) {
            return Ok((header, nonce));
        }
        nonce = nonce.wrapping_add(1);
    }
    Err(AuthorityError::invalid(format!(
        "exhausted nonce space for nbits {nbits:#x}"
    )))
}

fn verify_header_shape(
    h: &Header,
    tier: Tier,
    digest: &[u8; 32],
    prev: &[u8; 32],
    target_bits: u32,
) -> AuthResult<()> {
    let err = |detail: String| Err(AuthorityError::new(Failure::Header, detail));
    if h.version() != HEADER_VERSION {
        return err(format!("unsupported header version {}", h.version()));
    }
    if h.tier_byte() != tier as u8 {
        return err(format!(
            "header tier T{} does not match record tier {tier}",
            h.tier_byte()
        ));
    }
    if &h.prev_hash() != digest {
        return err("prevhash does not commit to the action digest".into());
    }
    if &h.merkle() != prev {
        return err("merkle root does not commit to the previous record hash".into());
    }
    if h.nbits() != target_bits {
        return err(format!(
            "nbits {:#x} does not match target {:#x}",
            h.nbits(),
            target_bits
        ));
    }
    Ok(())
}

/// Hex (de)serialisation for fixed-size and variable byte fields.
mod hexser {
    use serde::{de, Deserialize, Deserializer, Serializer};

    pub fn vec<S: Serializer>(value: &Vec<u8>, s: S) -> Result<S::Ok, S::Error> {
        s.serialize_str(&hex::encode(value))
    }
    pub fn vec_de<'de, D: Deserializer<'de>>(d: D) -> Result<Vec<u8>, D::Error> {
        let text = String::deserialize(d)?;
        hex::decode(text.trim_start_matches("0x")).map_err(de::Error::custom)
    }
    pub fn fixed<const N: usize, S: Serializer>(value: &[u8; N], s: S) -> Result<S::Ok, S::Error> {
        s.serialize_str(&hex::encode(value))
    }
    pub fn fixed_de<'de, const N: usize, D: Deserializer<'de>>(d: D) -> Result<[u8; N], D::Error> {
        let bytes = vec_de(d)?;
        bytes
            .try_into()
            .map_err(|v: Vec<u8>| de::Error::custom(format!("expected {N} bytes, got {}", v.len())))
    }
    pub fn header<S: Serializer>(value: &super::Header, s: S) -> Result<S::Ok, S::Error> {
        fixed(&value.0, s)
    }
    pub fn header_de<'de, D: Deserializer<'de>>(d: D) -> Result<super::Header, D::Error> {
        fixed_de::<80, D>(d).map(super::Header)
    }
}

fn key_pair(private_key: &[u8; 32]) -> AuthResult<(SigningKey, Vec<u8>)> {
    let key = SigningKey::from_bytes(private_key.into())
        .map_err(|e| AuthorityError::invalid(format!("invalid private key: {e}")))?;
    let public = key
        .verifying_key()
        .to_encoded_point(true)
        .as_bytes()
        .to_vec();
    Ok((key, public))
}

fn signed_message(
    envelope: &MessageEnvelope,
    signature: &[u8],
    public_key: &[u8],
) -> AuthResult<SignedMessageEnvelope> {
    Ok(SignedMessageEnvelope {
        envelope: BASE64.encode(marshal_message_envelope(envelope)?),
        signature: BASE64.encode(signature),
        public_key: BASE64.encode(public_key),
        address: knirv_address(public_key)?,
    })
}

/// Delegates an operator's long-term authority to an epoch-scoped session key.
#[derive(Clone, Debug, Default, PartialEq, Eq, Serialize, Deserialize)]
pub struct SessionCert {
    #[serde(serialize_with = "hexser::vec", deserialize_with = "hexser::vec_de")]
    pub session_pub: Vec<u8>,
    pub operator: String,
    pub epoch: u64,
    pub not_before: i64,
    pub not_after: i64,
    pub scope: String,
    pub entropy: String,
    #[serde(
        default,
        rename = "sig_by_longterm",
        serialize_with = "hexser::vec",
        deserialize_with = "hexser::vec_de"
    )]
    pub signature: Vec<u8>,
}

impl SessionCert {
    pub fn preimage(&self) -> Vec<u8> {
        let mut buf = Vec::new();
        buf.extend_from_slice(SESSION_CERT_DOMAIN);
        append_lp(&mut buf, &self.session_pub);
        append_lp(&mut buf, self.operator.as_bytes());
        buf.extend_from_slice(&self.epoch.to_be_bytes());
        buf.extend_from_slice(&(self.not_before as u64).to_be_bytes());
        buf.extend_from_slice(&(self.not_after as u64).to_be_bytes());
        append_lp(&mut buf, self.scope.as_bytes());
        append_lp(&mut buf, self.entropy.as_bytes());
        buf
    }

    /// The deterministic envelope the long-term key signs.
    pub fn envelope(&self) -> MessageEnvelope {
        let preimage = self.preimage();
        let sum = Sha256::digest(&preimage);
        MessageEnvelope {
            schema_version: None,
            domain: AUTHORITY_DOMAIN.into(),
            purpose: PURPOSE_SESSION_CERT.into(),
            chain_id: AUTHORITY_CHAIN_ID.into(),
            nonce: hex::encode(&sum[..8]),
            issued_at_unix: self.not_before.max(0) as u64,
            expires_at_unix: self.not_after.max(0) as u64,
            payload: Some(preimage),
        }
    }

    /// Checks the certificate's own fields without verifying its signature.
    pub fn validate(&self) -> AuthResult<()> {
        if self.session_pub.len() != 33 {
            return Err(AuthorityError::invalid(
                "session pubkey must be compressed secp256k1",
            ));
        }
        if self.operator.is_empty() {
            return Err(AuthorityError::invalid(
                "session certificate operator is required",
            ));
        }
        if self.scope.is_empty() {
            return Err(AuthorityError::invalid(
                "session certificate scope is required",
            ));
        }
        if self.not_before <= 0 || self.not_after <= self.not_before {
            return Err(AuthorityError::invalid(
                "session certificate validity window is invalid",
            ));
        }
        if self.entropy != "csprng" && self.entropy != "csprng+asic" {
            return Err(AuthorityError::invalid(format!(
                "unknown session entropy mode {:?}",
                self.entropy
            )));
        }
        knirv_address(&self.session_pub)?;
        Ok(())
    }

    /// Certifies the session key with the operator's long-term private key.
    pub fn sign_with_key(mut self, long_term_key: &[u8; 32]) -> AuthResult<Self> {
        self.validate()?;
        let (_, public) = key_pair(long_term_key)?;
        let address = knirv_address(&public)?;
        if address != self.operator {
            return Err(AuthorityError::invalid(format!(
                "long-term signer {address} does not match certificate operator {}",
                self.operator
            )));
        }
        let signed = sign_message_envelope(long_term_key, &self.envelope())?;
        self.signature = BASE64
            .decode(signed.signature)
            .map_err(|e| AuthorityError::invalid(e.to_string()))?;
        Ok(self)
    }

    /// Checks the certificate window and the long-term signature.
    pub fn verify(&self, long_term_public_key: &[u8], now_unix: i64) -> AuthResult<()> {
        self.validate()?;
        if now_unix < self.not_before {
            return Err(AuthorityError::new(
                Failure::ExpiredSession,
                "certificate is not yet valid",
            ));
        }
        if now_unix > self.not_after {
            return Err(AuthorityError::new(
                Failure::ExpiredSession,
                format!("certificate expired at {}", self.not_after),
            ));
        }
        if self.signature.len() != 64 {
            return Err(AuthorityError::invalid(
                "session certificate signature must be 64 bytes",
            ));
        }
        let envelope = self.envelope();
        let signed = signed_message(&envelope, &self.signature, long_term_public_key)
            .map_err(|e| AuthorityError::new(Failure::SessionSignature, e.to_string()))?;
        signing::verify_message(
            &signed,
            AUTHORITY_DOMAIN,
            PURPOSE_SESSION_CERT,
            AUTHORITY_CHAIN_ID,
            &envelope.nonce,
            now_unix.max(0) as u64,
        )
        .map_err(|e| AuthorityError::new(Failure::SessionSignature, e.to_string()))
    }
}

/// One link in an operator's authority chain.
#[derive(Clone, Debug, PartialEq, Eq, Serialize, Deserialize)]
pub struct Record {
    pub tier: Tier,
    pub domain: String,
    pub scope: String,
    pub operator: String,
    pub epoch: u64,
    #[serde(
        serialize_with = "hexser::fixed",
        deserialize_with = "hexser::fixed_de"
    )]
    pub canonical_action_hash: [u8; 32],
    #[serde(
        serialize_with = "hexser::fixed",
        deserialize_with = "hexser::fixed_de"
    )]
    pub prev_record_hash: [u8; 32],
    #[serde(
        serialize_with = "hexser::header",
        deserialize_with = "hexser::header_de",
        default = "zero_header"
    )]
    pub header: Header,
    #[serde(default)]
    pub nonce: u32,
    #[serde(
        default,
        serialize_with = "hexser::fixed",
        deserialize_with = "hexser::fixed_de"
    )]
    pub hash: [u8; 32],
    #[serde(default)]
    pub target_bits: u32,
    #[serde(default)]
    pub witness_kind: String,
    #[serde(default, skip_serializing_if = "String::is_empty")]
    pub device_claim: String,
    pub session_cert: SessionCert,
    #[serde(
        default,
        serialize_with = "hexser::vec",
        deserialize_with = "hexser::vec_de"
    )]
    pub signature: Vec<u8>,
}

fn zero_header() -> Header {
    Header([0u8; 80])
}

impl Record {
    /// Recomputes the action digest from the record's immutable inputs.
    pub fn digest(&self) -> [u8; 32] {
        action_digest(
            self.tier,
            &self.domain,
            &self.scope,
            &self.operator,
            self.epoch,
            &self.canonical_action_hash,
            &self.prev_record_hash,
        )
    }

    /// The deterministic envelope signed by the session key.
    pub fn envelope(&self) -> MessageEnvelope {
        let digest = self.digest();
        let sum = Sha256::digest(digest);
        MessageEnvelope {
            schema_version: None,
            domain: AUTHORITY_DOMAIN.into(),
            purpose: PURPOSE_AUTHORITY_RECORD.into(),
            chain_id: AUTHORITY_CHAIN_ID.into(),
            nonce: hex::encode(&sum[..8]),
            issued_at_unix: self.session_cert.not_before.max(0) as u64,
            expires_at_unix: self.session_cert.not_after.max(0) as u64,
            payload: Some(digest.to_vec()),
        }
    }

    fn validate_inputs(&self) -> AuthResult<()> {
        if self.tier == Tier::T0 {
            return Err(AuthorityError::new(
                Failure::Tier,
                format!("{} records do not carry a witness", self.tier),
            ));
        }
        if self.domain.is_empty() || self.scope.is_empty() || self.operator.is_empty() {
            return Err(AuthorityError::invalid(
                "domain, scope and operator are required",
            ));
        }
        if self.session_cert.operator != self.operator {
            return Err(AuthorityError::invalid(
                "session certificate operator does not match record operator",
            ));
        }
        if self.session_cert.epoch != self.epoch {
            return Err(AuthorityError::invalid(
                "session certificate epoch does not match record epoch",
            ));
        }
        self.session_cert.validate()
    }

    /// Derives the header and mines a witness at the tier's target, without
    /// signing. Callers holding only a remote signer call this, sign
    /// [`Record::envelope`] themselves and set `signature`.
    pub fn mine(&mut self, ntime: u32) -> AuthResult<()> {
        self.validate_inputs()?;
        if self.target_bits == 0 {
            self.target_bits = tier_target_bits(self.tier);
        }
        if self.witness_kind.is_empty() {
            self.witness_kind = WITNESS_SOFTWARE.into();
        }
        if self.witness_kind != WITNESS_SOFTWARE && self.witness_kind != WITNESS_ASIC {
            return Err(AuthorityError::invalid(format!(
                "unknown witness kind {:?}",
                self.witness_kind
            )));
        }
        let digest = self.digest();
        let (header, nonce) = mine(
            &digest,
            &self.prev_record_hash,
            ntime,
            self.target_bits,
            self.nonce,
            HEADER_VERSION,
            self.tier,
        )?;
        self.header = header;
        self.nonce = nonce;
        self.hash = header.witness();
        Ok(())
    }

    /// Signs the already-mined record with an in-process session key. Rejects
    /// a key that does not match the certificate.
    pub fn sign_with_key(&mut self, session_key: &[u8; 32]) -> AuthResult<()> {
        self.validate_inputs()?;
        let (_, public) = key_pair(session_key)?;
        if public != self.session_cert.session_pub {
            return Err(AuthorityError::invalid(
                "session signer does not match session certificate",
            ));
        }
        let signed = sign_message_envelope(session_key, &self.envelope())?;
        self.signature = BASE64
            .decode(signed.signature)
            .map_err(|e| AuthorityError::invalid(e.to_string()))?;
        Ok(())
    }

    /// Mines and signs with an in-process session key.
    pub fn seal_with_key(&mut self, session_key: &[u8; 32], ntime: u32) -> AuthResult<()> {
        self.mine(ntime)?;
        self.sign_with_key(session_key)
    }
}

#[derive(Clone, Copy, Debug, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "lowercase")]
pub enum OperatorKind {
    Root,
    Boot,
}

impl fmt::Display for OperatorKind {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        f.write_str(match self {
            OperatorKind::Root => "root",
            OperatorKind::Boot => "boot",
        })
    }
}

/// A registered long-term authority identity.
#[derive(Clone, Debug, PartialEq, Eq, Serialize, Deserialize)]
pub struct Operator {
    pub address: String,
    pub kind: OperatorKind,
    #[serde(serialize_with = "hexser::vec", deserialize_with = "hexser::vec_de")]
    pub long_term_pub: Vec<u8>,
    #[serde(default)]
    pub scopes: Vec<String>,
    /// hex(session_pub) → unix time of retirement. A retired session may only
    /// sign records issued at or before that time.
    #[serde(default, skip_serializing_if = "BTreeMap::is_empty")]
    pub retired_sessions: BTreeMap<String, i64>,
}

/// The verifier's view of registered operators.
#[derive(Clone, Debug, Default, PartialEq, Eq, Serialize, Deserialize)]
pub struct Registry {
    #[serde(default)]
    pub operators: BTreeMap<String, Operator>,
}

fn scope_covers(granted: &str, requested: &str) -> bool {
    granted == "*" || granted == requested
}

/// Verifies a record in the contract order: session certificate, record
/// signature, witness, then tier and scope authority.
pub fn verify(record: &Record, registry: &Registry, now_unix: i64) -> AuthResult<()> {
    if record.tier == Tier::T0 {
        return Err(AuthorityError::new(
            Failure::Tier,
            format!("{} records do not carry a witness", record.tier),
        ));
    }
    let op = registry
        .operators
        .get(&record.operator)
        .ok_or_else(|| AuthorityError::new(Failure::UnknownOperator, record.operator.clone()))?;

    let cert = &record.session_cert;
    if cert.operator != record.operator {
        return Err(AuthorityError::new(
            Failure::SessionSignature,
            format!(
                "certificate operator {} does not match record operator {}",
                cert.operator, record.operator
            ),
        ));
    }
    if cert.epoch != record.epoch {
        return Err(AuthorityError::new(
            Failure::SessionSignature,
            format!(
                "certificate epoch {} does not match record epoch {}",
                cert.epoch, record.epoch
            ),
        ));
    }
    cert.verify(&op.long_term_pub, now_unix)?;

    let digest = record.digest();
    let envelope = record.envelope();
    if record.signature.len() != 64 {
        return Err(AuthorityError::new(
            Failure::RecordSignature,
            "signature must be 64 bytes",
        ));
    }
    let signed = signed_message(&envelope, &record.signature, &cert.session_pub)
        .map_err(|e| AuthorityError::new(Failure::RecordSignature, e.to_string()))?;
    signing::verify_message(
        &signed,
        AUTHORITY_DOMAIN,
        PURPOSE_AUTHORITY_RECORD,
        AUTHORITY_CHAIN_ID,
        &envelope.nonce,
        now_unix.max(0) as u64,
    )
    .map_err(|e| AuthorityError::new(Failure::RecordSignature, e.to_string()))?;

    if record.target_bits != tier_target_bits(record.tier) {
        return Err(AuthorityError::new(
            Failure::Target,
            format!(
                "record uses {:#x}, tier {} requires {:#x}",
                record.target_bits,
                record.tier,
                tier_target_bits(record.tier)
            ),
        ));
    }
    verify_header_shape(
        &record.header,
        record.tier,
        &digest,
        &record.prev_record_hash,
        record.target_bits,
    )?;
    if record.header.nonce() != record.nonce {
        return Err(AuthorityError::new(
            Failure::Header,
            "header nonce does not match record nonce",
        ));
    }
    if record.hash != record.header.witness() {
        return Err(AuthorityError::new(
            Failure::Header,
            "record hash does not match header witness",
        ));
    }
    if !meets_target(&record.hash, record.target_bits) {
        return Err(AuthorityError::new(
            Failure::Witness,
            format!("hash does not meet target {:#x}", record.target_bits),
        ));
    }
    if record.witness_kind != WITNESS_SOFTWARE && record.witness_kind != WITNESS_ASIC {
        return Err(AuthorityError::new(
            Failure::Witness,
            format!("unknown witness kind {:?}", record.witness_kind),
        ));
    }

    match record.tier {
        Tier::T1 => {
            if record.scope != op.address {
                return Err(AuthorityError::new(
                    Failure::Scope,
                    "T1 must be scoped to the operator's own address",
                ));
            }
        }
        Tier::T2 | Tier::T3 => {
            if op.kind != OperatorKind::Root {
                return Err(AuthorityError::new(
                    Failure::Tier,
                    format!(
                        "{} operators may not sign {} without root co-attestation",
                        op.kind, record.tier
                    ),
                ));
            }
        }
        Tier::T0 => {}
    }

    if !scope_covers(&cert.scope, &record.scope) {
        return Err(AuthorityError::new(
            Failure::Scope,
            format!(
                "session certificate scope {:?} does not cover {:?}",
                cert.scope, record.scope
            ),
        ));
    }
    if !op.scopes.iter().any(|g| scope_covers(g, &record.scope)) {
        return Err(AuthorityError::new(
            Failure::Scope,
            format!("registered authority does not cover {:?}", record.scope),
        ));
    }
    if let Some(retired_at) = op.retired_sessions.get(&hex::encode(&cert.session_pub)) {
        if cert.not_before > *retired_at {
            return Err(AuthorityError::new(
                Failure::SessionSignature,
                format!("session key was retired at {retired_at}"),
            ));
        }
    }
    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;
    use serde_json::{json, Value};

    const ACTION: &str = "evidence.commit";
    const EPOCH: u64 = 7;
    const NTIME: u32 = 1_750_000_000;
    const NOT_BEFORE: i64 = 1_750_000_000;
    const NOT_AFTER: i64 = 1_750_086_400;
    const NOW: i64 = 1_750_000_100;
    const TARGET: u32 = 0x207f_ffff;

    fn test_key(n: u8) -> [u8; 32] {
        let mut key = [0u8; 32];
        key[31] = n;
        key
    }
    fn pub_of(key: &[u8; 32]) -> Vec<u8> {
        key_pair(key).unwrap().1
    }
    fn sha(text: &str) -> [u8; 32] {
        Sha256::digest(text.as_bytes()).into()
    }
    fn vector(name: &str) -> Value {
        let path = format!(
            "{}/../testvectors/authority/{name}",
            env!("CARGO_MANIFEST_DIR")
        );
        serde_json::from_slice(&std::fs::read(path).unwrap()).unwrap()
    }

    struct Fixture {
        long_key: [u8; 32],
        session_key: [u8; 32],
        record: Record,
        registry: Registry,
    }

    fn fixture() -> Fixture {
        let long_key = test_key(1);
        let session_key = test_key(2);
        let operator = knirv_address(&pub_of(&long_key)).unwrap();
        let cert = SessionCert {
            session_pub: pub_of(&session_key),
            operator: operator.clone(),
            epoch: EPOCH,
            not_before: NOT_BEFORE,
            not_after: NOT_AFTER,
            scope: operator.clone(),
            entropy: "csprng".into(),
            signature: vec![],
        }
        .sign_with_key(&long_key)
        .unwrap();
        let mut record = Record {
            tier: Tier::T1,
            domain: ACTION.into(),
            scope: operator.clone(),
            operator: operator.clone(),
            epoch: EPOCH,
            canonical_action_hash: sha("badge:42"),
            prev_record_hash: sha("genesis"),
            header: Header([0; 80]),
            nonce: 0,
            hash: [0; 32],
            target_bits: TARGET,
            witness_kind: WITNESS_SOFTWARE.into(),
            device_claim: String::new(),
            session_cert: cert,
            signature: vec![],
        };
        record.seal_with_key(&session_key, NTIME).unwrap();
        let mut operators = BTreeMap::new();
        operators.insert(
            operator.clone(),
            Operator {
                address: operator.clone(),
                kind: OperatorKind::Root,
                long_term_pub: pub_of(&long_key),
                scopes: vec![operator],
                retired_sessions: BTreeMap::new(),
            },
        );
        Fixture {
            long_key,
            session_key,
            record,
            registry: Registry { operators },
        }
    }

    fn h32(v: &[u8; 32]) -> String {
        hex::encode(v)
    }

    /// Regenerates the shared vectors (`UPDATE_AUTHORITY_VECTORS=1`). Without
    /// the flag it asserts the committed vectors equal what Rust produces, so
    /// a drift in either direction fails the build.
    #[test]
    fn vectors_match_rust_output() {
        let f = fixture();
        let r = &f.record;
        let digest = json!({
            "version": 1, "tier": r.tier as u8, "domain": r.domain, "scope": r.scope,
            "operator": r.operator, "epoch": r.epoch,
            "canonical_action_hash": h32(&r.canonical_action_hash),
            "prev_record_hash": h32(&r.prev_record_hash), "digest": h32(&r.digest()),
        });
        let witness = json!({
            "version": 1, "prev_hash": h32(&r.digest()), "merkle": h32(&r.prev_record_hash),
            "ntime": r.header.ntime(), "nbits": r.header.nbits(), "nonce": r.header.nonce(),
            "header": hex::encode(r.header.0), "hash": h32(&r.hash), "valid": true,
        });
        let c = &r.session_cert;
        let cert = json!({
            "version": 1, "session_pub": hex::encode(&c.session_pub),
            "long_term_pub": hex::encode(pub_of(&f.long_key)), "operator": c.operator,
            "epoch": c.epoch, "not_before": c.not_before, "not_after": c.not_after,
            "scope": c.scope, "entropy": c.entropy, "sig_by_longterm": hex::encode(&c.signature),
        });
        let files = [
            ("digest.json", digest),
            ("witness.json", witness),
            ("session_cert.json", cert),
        ];
        for (name, value) in files {
            if std::env::var("UPDATE_AUTHORITY_VECTORS").as_deref() == Ok("1") {
                let path = format!(
                    "{}/../testvectors/authority/{name}",
                    env!("CARGO_MANIFEST_DIR")
                );
                let mut text = serde_json::to_string_pretty(&value).unwrap();
                text.push('\n');
                std::fs::write(path, text).unwrap();
            } else {
                assert_eq!(vector(name), value, "{name} differs from Rust output");
            }
        }
    }

    #[test]
    fn digest_vector() {
        let v = vector("digest.json");
        let get = |k: &str| v[k].as_str().unwrap();
        let to32 = |s: &str| -> [u8; 32] { hex::decode(s).unwrap().try_into().unwrap() };
        let got = action_digest(
            Tier::try_from(v["tier"].as_u64().unwrap() as u8).unwrap(),
            get("domain"),
            get("scope"),
            get("operator"),
            v["epoch"].as_u64().unwrap(),
            &to32(get("canonical_action_hash")),
            &to32(get("prev_record_hash")),
        );
        assert_eq!(h32(&got), get("digest"));
    }

    #[test]
    fn witness_vector() {
        let v = vector("witness.json");
        let to32 = |k: &str| -> [u8; 32] {
            hex::decode(v[k].as_str().unwrap())
                .unwrap()
                .try_into()
                .unwrap()
        };
        let header = Header::new(
            &to32("prev_hash"),
            &to32("merkle"),
            v["ntime"].as_u64().unwrap() as u32,
            v["nbits"].as_u64().unwrap() as u32,
            v["nonce"].as_u64().unwrap() as u32,
            HEADER_VERSION,
            Tier::T1,
        );
        assert_eq!(hex::encode(header.0), v["header"].as_str().unwrap());
        assert_eq!(h32(&header.witness()), v["hash"].as_str().unwrap());
        assert_eq!(
            meets_target(&header.witness(), header.nbits()),
            v["valid"].as_bool().unwrap()
        );
    }

    #[test]
    fn target_edge_cases() {
        assert!(target_from_bits(0x0080_0001).is_err()); // negative
        assert!(target_from_bits(0x2000_0000).is_err()); // zero mantissa
        assert!(target_from_bits(0x0100_0001).is_err()); // shifts to zero
        assert!(target_from_bits(0x2200_ffff).is_err()); // > 256 bits
        assert!(target_from_bits(0x2100_ffff).is_ok()); // exactly 256 bits
        let t = target_from_bits(0x1d00_ffff).unwrap();
        assert_eq!(&t[..8], &[0, 0, 0, 0, 0xff, 0xff, 0, 0]); // Bitcoin genesis target
        let easy = target_from_bits(TARGET).unwrap();
        assert_eq!(&easy[..4], &[0x7f, 0xff, 0xff, 0]);
    }

    #[test]
    fn session_cert_vector() {
        let v = vector("session_cert.json");
        let cert = SessionCert {
            session_pub: hex::decode(v["session_pub"].as_str().unwrap()).unwrap(),
            operator: v["operator"].as_str().unwrap().into(),
            epoch: v["epoch"].as_u64().unwrap(),
            not_before: v["not_before"].as_i64().unwrap(),
            not_after: v["not_after"].as_i64().unwrap(),
            scope: v["scope"].as_str().unwrap().into(),
            entropy: v["entropy"].as_str().unwrap().into(),
            signature: hex::decode(v["sig_by_longterm"].as_str().unwrap()).unwrap(),
        };
        let long_pub = hex::decode(v["long_term_pub"].as_str().unwrap()).unwrap();
        cert.verify(&long_pub, NOW).unwrap();
        let expired = cert.verify(&long_pub, cert.not_after + 1).unwrap_err();
        assert_eq!(expired.kind, Failure::ExpiredSession);
    }

    #[test]
    fn base_record_verifies_and_roundtrips_json() {
        let f = fixture();
        verify(&f.record, &f.registry, NOW).unwrap();
        let wire = serde_json::to_string(&f.record).unwrap();
        let back: Record = serde_json::from_str(&wire).unwrap();
        assert_eq!(back, f.record);
    }

    #[test]
    fn tamper_cases() {
        let cases = vector("tamper.json");
        for case in cases["cases"].as_array().unwrap() {
            let field = case["field"].as_str().unwrap();
            let want = case["error"].as_str().unwrap();
            let f = fixture();
            let mut r = f.record.clone();
            match field {
                "domain" => r.domain = "badge.select".into(),
                "scope" => r.scope = "knirv1other".into(),
                "operator" => r.operator = "knirv1other".into(),
                "epoch" => r.epoch += 1,
                "canonical_action_hash" => r.canonical_action_hash[0] ^= 1,
                "prev_record_hash" => r.prev_record_hash[0] ^= 1,
                "header_prevhash" => r.header.0[4] ^= 1,
                "header_merkle" => r.header.0[36] ^= 1,
                "nonce" => r.nonce += 1,
                "hash" => r.hash[0] ^= 1,
                "target_bits" => r.target_bits = 0x1f00_ffff,
                "witness_kind" => r.witness_kind = "gpu".into(),
                "session_signature" => r.session_cert.signature[0] ^= 1,
                "record_signature" => r.signature[0] ^= 1,
                other => panic!("unknown tamper field {other}"),
            }
            let err = verify(&r, &f.registry, NOW).expect_err(field);
            assert_eq!(err.kind.code(), want, "{field}: {err}");
        }
    }

    #[test]
    fn expired_scope_tier_and_retirement() {
        let f = fixture();
        assert_eq!(
            verify(&f.record, &f.registry, NOT_AFTER + 1)
                .unwrap_err()
                .kind,
            Failure::ExpiredSession
        );

        let mut wrong = f.record.clone();
        wrong.scope = "knirv1other".into();
        wrong.session_cert.scope = "knirv1other".into();
        wrong.session_cert = wrong.session_cert.sign_with_key(&f.long_key).unwrap();
        wrong.seal_with_key(&f.session_key, NTIME).unwrap();
        assert_eq!(
            verify(&wrong, &f.registry, NOW).unwrap_err().kind,
            Failure::Scope
        );

        let mut harder = f.record.clone();
        harder.target_bits = tier_target_bits(Tier::T2);
        assert_eq!(
            verify(&harder, &f.registry, NOW).unwrap_err().kind,
            Failure::Target
        );

        let mut registry = f.registry.clone();
        registry
            .operators
            .values_mut()
            .next()
            .unwrap()
            .retired_sessions
            .insert(
                hex::encode(&f.record.session_cert.session_pub),
                NOT_BEFORE - 1,
            );
        assert_eq!(
            verify(&f.record, &registry, NOW).unwrap_err().kind,
            Failure::SessionSignature
        );
    }

    #[test]
    fn t0_and_boot_operator_rules() {
        let f = fixture();
        let mut t0 = f.record.clone();
        t0.tier = Tier::T0;
        assert_eq!(
            verify(&t0, &f.registry, NOW).unwrap_err().kind,
            Failure::Tier
        );
        assert_eq!(t0.mine(NTIME).unwrap_err().kind, Failure::Tier);

        // A boot operator may not sign T2.
        let mut registry = f.registry.clone();
        registry.operators.values_mut().next().unwrap().kind = OperatorKind::Boot;
        let mut t2 = f.record.clone();
        t2.tier = Tier::T2;
        t2.target_bits = 0;
        t2.seal_with_key(&f.session_key, NTIME).unwrap();
        assert_eq!(verify(&t2, &registry, NOW).unwrap_err().kind, Failure::Tier);
    }

    #[test]
    fn sign_rejects_wrong_session_key() {
        let f = fixture();
        let mut r = f.record.clone();
        assert!(r.sign_with_key(&test_key(9)).is_err());
    }
}
