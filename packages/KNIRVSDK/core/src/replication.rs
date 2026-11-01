//! Oracle replication contract (Phase 8.3): signed heads, their canonical hash,
//! the chain rules that separate "next", "gap" and "fork", and pull-request
//! authentication.
//!
//! The Go and TypeScript packages are bindings over this module. Head bytes are
//! canonical JSON in declaration order, so the hash of a head is identical in
//! every language; the shared fixture under `testvectors/replication/` is
//! checked by the Rust tests and by the bindings.

use crate::signing::{self, SignedMessageEnvelope};
use base64::{engine::general_purpose::STANDARD as BASE64, Engine as _};
use serde::{Deserialize, Serialize};
use sha2::{Digest, Sha256};

pub const VERSION: u64 = 1;
pub const DOMAIN: &str = "knirv.replication";
pub const PURPOSE_HEAD: &str = "replication-head";
pub const PURPOSE_PULL: &str = "replication-pull";
pub const MAX_CLOCK_SKEW_SECONDS: i64 = 120;

/// The payload of a replication-head envelope. Field order is the canonical
/// order: do not reorder without bumping VERSION.
#[derive(Clone, Debug, Default, Serialize, Deserialize, PartialEq, Eq)]
pub struct Head {
    #[serde(default)]
    pub version: u64,
    #[serde(default)]
    pub network_id: String,
    #[serde(default)]
    pub oracle_chain_id: String,
    #[serde(default)]
    pub owner_address: String,
    #[serde(default)]
    pub epoch: u64,
    #[serde(default)]
    pub seq: u64,
    #[serde(default)]
    pub mmr_size: u64,
    #[serde(default)]
    pub mmr_root: String,
    #[serde(default)]
    pub registry_hash: String,
    #[serde(default)]
    pub state_log_size: u64,
    #[serde(default)]
    pub state_root: String,
    #[serde(default)]
    pub aux_hash: String,
    #[serde(default)]
    pub consensus_height: u64,
    #[serde(default)]
    pub produced_at: i64,
    #[serde(default)]
    pub prev_head_hash: String,
}

/// A classified failure. `code` is stable and is what the bindings map to
/// typed errors; `message` is for operators.
#[derive(Clone, Debug)]
pub struct Failure {
    pub code: &'static str,
    pub message: String,
}

impl Failure {
    fn new(code: &'static str, message: impl Into<String>) -> Self {
        Self {
            code,
            message: message.into(),
        }
    }
}

pub type Outcome<T> = Result<T, Failure>;

impl Head {
    /// Canonical bytes: exactly what is signed and hashed.
    pub fn canonical(&self) -> Outcome<Vec<u8>> {
        serde_json::to_vec(self)
            .map_err(|e| Failure::new("invalid_head", format!("encode head: {e}")))
    }

    /// sha256(canonical), lowercase hex.
    pub fn hash(&self) -> Outcome<String> {
        let body = self.canonical()?;
        Ok(hex(&Sha256::digest(&body)))
    }

    /// Invariants that need neither a signature nor a predecessor.
    pub fn validate(&self) -> Outcome<()> {
        if self.version != VERSION {
            return Err(Failure::new(
                "version",
                format!("unsupported head version {}", self.version),
            ));
        }
        for (name, value) in [
            ("network_id", &self.network_id),
            ("oracle_chain_id", &self.oracle_chain_id),
            ("owner_address", &self.owner_address),
            ("mmr_root", &self.mmr_root),
        ] {
            if value.is_empty() {
                return Err(Failure::new("invalid_head", format!("{name} is empty")));
            }
        }
        if self.seq == 0 {
            return Err(Failure::new("invalid_head", "seq must be positive"));
        }
        Ok(())
    }
}

/// How a fetched head relates to the local one.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum Follow {
    Accepted,
    Old,
    Gap,
    Diverged,
}

impl Follow {
    pub fn as_str(self) -> &'static str {
        match self {
            Follow::Accepted => "accepted",
            Follow::Old => "old",
            Follow::Gap => "gap",
            Follow::Diverged => "diverged",
        }
    }
}

/// Compare applies the chain rules. A gap is recoverable by paging heads; a
/// divergence is absorbing and must never overwrite local history.
pub fn compare(local: &Head, next: &Head) -> Outcome<Follow> {
    let local_hash = local.hash()?;
    if next.seq <= local.seq {
        return Ok(Follow::Old);
    }
    if next.seq > local.seq + 1 {
        return Ok(Follow::Gap);
    }
    if next.prev_head_hash != local_hash {
        return Err(Failure::new(
            "diverged",
            format!(
                "seq {} names prev {}, local head is {}",
                next.seq, next.prev_head_hash, local_hash
            ),
        ));
    }
    if next.mmr_size < local.mmr_size {
        return Err(Failure::new("diverged", "mmr_size decreased"));
    }
    if next.state_log_size < local.state_log_size {
        return Err(Failure::new("diverged", "state_log_size decreased"));
    }
    if next.consensus_height < local.consensus_height {
        return Err(Failure::new("diverged", "consensus_height decreased"));
    }
    if next.produced_at < local.produced_at {
        return Err(Failure::new("diverged", "produced_at went backwards"));
    }
    if next.epoch < local.epoch {
        return Err(Failure::new("diverged", "epoch went backwards"));
    }
    Ok(Follow::Accepted)
}

/// Authenticate a signed head. The envelope payload is authoritative; the copy
/// carried alongside it is never trusted.
pub fn verify_head(
    signed: &SignedMessageEnvelope,
    trust_anchor: &str,
    now_unix: u64,
) -> Outcome<Head> {
    let envelope = decode_envelope(signed)?;
    let parsed = signing::parse_message_envelope(&envelope)
        .map_err(|e| Failure::new("invalid_head", format!("parse envelope: {e}")))?;
    let head: Head = serde_json::from_slice(&parsed.payload)
        .map_err(|e| Failure::new("invalid_head", format!("payload is not a head: {e}")))?;
    head.validate()?;
    if parsed.domain != DOMAIN
        || parsed.purpose != PURPOSE_HEAD
        || parsed.chain_id != head.oracle_chain_id
    {
        return Err(Failure::new(
            "bad_signature",
            "envelope is not a replication head for this chain",
        ));
    }
    signing::verify_message_payload(
        signed,
        DOMAIN,
        PURPOSE_HEAD,
        &head.oracle_chain_id,
        &parsed.payload,
        now_unix,
    )
    .map_err(|e| Failure::new("bad_signature", format!("head signature: {e}")))?;
    if signed.address != head.owner_address {
        return Err(Failure::new(
            "bad_signature",
            format!(
                "envelope address {} != owner {}",
                signed.address, head.owner_address
            ),
        ));
    }
    if !trust_anchor.is_empty() && head.owner_address != trust_anchor {
        return Err(Failure::new(
            "trust_anchor",
            format!(
                "head owner {} is not the trust anchor {}",
                head.owner_address, trust_anchor
            ),
        ));
    }
    Ok(head)
}

#[derive(Clone, Debug, Default, Serialize, Deserialize, PartialEq, Eq)]
pub struct PullRequest {
    #[serde(default)]
    pub chain_id: String,
    #[serde(default)]
    pub head_seq_seen: u64,
    #[serde(default)]
    pub nonce: String,
    #[serde(default)]
    pub ts: i64,
}

/// Authenticate a pull request against the registered validator set. Nonce
/// replay state lives on the source; everything else is decided here.
pub fn verify_pull(
    signed: &SignedMessageEnvelope,
    allowed: &[String],
    now_unix: u64,
    skew_seconds: i64,
) -> Outcome<PullRequest> {
    let envelope = decode_envelope(signed)?;
    let parsed = signing::parse_message_envelope(&envelope)
        .map_err(|e| Failure::new("pull_auth", format!("parse envelope: {e}")))?;
    if parsed.domain != DOMAIN || parsed.purpose != PURPOSE_PULL {
        return Err(Failure::new("pull_auth", "not a replication pull envelope"));
    }
    let request: PullRequest = serde_json::from_slice(&parsed.payload)
        .map_err(|e| Failure::new("pull_auth", format!("payload is not a pull request: {e}")))?;
    if request.chain_id != parsed.chain_id || request.nonce.is_empty() {
        return Err(Failure::new("pull_auth", "chain/nonce mismatch"));
    }
    let skew = if skew_seconds <= 0 {
        MAX_CLOCK_SKEW_SECONDS
    } else {
        skew_seconds
    };
    if (now_unix as i64 - request.ts).abs() > skew {
        return Err(Failure::new(
            "pull_auth",
            "timestamp is outside the skew window",
        ));
    }
    if !allowed.iter().any(|a| a == &signed.address) {
        return Err(Failure::new(
            "pull_auth",
            format!("{} is not a registered validator", signed.address),
        ));
    }
    signing::verify_message_payload(
        signed,
        DOMAIN,
        PURPOSE_PULL,
        &request.chain_id,
        &parsed.payload,
        now_unix,
    )
    .map_err(|e| Failure::new("pull_auth", format!("pull signature: {e}")))?;
    Ok(request)
}

fn decode_envelope(signed: &SignedMessageEnvelope) -> Outcome<Vec<u8>> {
    BASE64
        .decode(&signed.envelope)
        .map_err(|e| Failure::new("bad_signature", format!("decode envelope: {e}")))
}

fn hex(bytes: &[u8]) -> String {
    let mut out = String::with_capacity(bytes.len() * 2);
    for b in bytes {
        out.push_str(&format!("{b:02x}"));
    }
    out
}

#[cfg(test)]
mod tests {
    use super::*;

    fn fixture() -> Vec<Head> {
        let path = concat!(
            env!("CARGO_MANIFEST_DIR"),
            "/../testvectors/replication/heads_v1.json"
        );
        let raw = std::fs::read_to_string(path).expect("read replication fixture");
        let value: serde_json::Value = serde_json::from_str(&raw).expect("decode fixture");
        let heads: Vec<Head> = serde_json::from_value(value["heads"].clone()).expect("heads");
        let hashes: Vec<String> = serde_json::from_value(value["hashes"].clone()).expect("hashes");
        for (head, want) in heads.iter().zip(hashes.iter()) {
            assert_eq!(&head.hash().expect("hash"), want, "fixture hash drifted");
        }
        heads
    }

    #[test]
    fn shared_fixture_hashes_match() {
        let heads = fixture();
        assert_eq!(heads.len(), 3);
        for pair in heads.windows(2) {
            assert_eq!(compare(&pair[0], &pair[1]).unwrap(), Follow::Accepted);
        }
    }

    #[test]
    fn compare_distinguishes_gap_and_fork() {
        let heads = fixture();
        assert_eq!(compare(&heads[0], &heads[2]).unwrap(), Follow::Gap);
        let mut fork = heads[1].clone();
        fork.prev_head_hash = "not-the-local-head".into();
        assert_eq!(compare(&heads[0], &fork).unwrap_err().code, "diverged");
    }

    #[test]
    fn head_validation_is_fail_closed() {
        let mut head = fixture().remove(0);
        head.version = 99;
        assert_eq!(head.validate().unwrap_err().code, "version");
        head.version = VERSION;
        head.seq = 0;
        assert_eq!(head.validate().unwrap_err().code, "invalid_head");
    }
}
