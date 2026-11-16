//! Evidence-log and artifact-Merkle verification. The algorithms live in the
//! `proof-verifier` module's sources and are compiled into the core from there,
//! so the standalone WASM module and the core can never drift.

#[path = "../wasm-modules/proof-verifier/src/eventlog.rs"]
pub mod eventlog;
#[path = "../wasm-modules/proof-verifier/src/mmr.rs"]
pub mod mmr;

use serde::{Deserialize, Serialize};
use sha2::{Digest, Sha256};

#[cfg(feature = "ed25519")]
pub use ed25519::{
    ed25519_key_id, ed25519_public_from_seed, ed25519_public_key, ed25519_sign, ed25519_verify,
    Ed25519Failure,
};

/// SHA-256 of `data` as an evidence digest: `sha256:` and 64 lowercase hex digits.
/// Always compiled (no feature): the digests are the evidence format's own.
pub fn sha256_digest(data: &[u8]) -> String {
    format!("sha256:{}", hex::encode(Sha256::digest(data)))
}

/// The claim an evidence bundle's signature covers: the bundle's identity and roots.
/// Field order is the order of the signed message. `event_bundle_root` is left out
/// of the message when empty (the Go `omitempty` tag it replaces).
#[derive(Clone, Debug, Default, PartialEq, Eq, Serialize, Deserialize)]
pub struct SigningClaim {
    #[serde(default)]
    pub schema_version: String,
    #[serde(default)]
    pub session_id: String,
    #[serde(default)]
    pub dve_id: String,
    #[serde(default)]
    pub user_id: String,
    #[serde(default)]
    pub project_id: String,
    #[serde(default)]
    pub policy_hash: String,
    #[serde(default)]
    pub eventlog_root: String,
    #[serde(default)]
    pub artifact_merkle_root: String,
    #[serde(default)]
    pub workspace_base_hash: String,
    #[serde(default)]
    pub workspace_final_hash: String,
    #[serde(default)]
    pub event_bundle_root: String,
}

/// The bytes an evidence signature covers: the claim as a JSON object, keys in
/// field order. Strings are escaped exactly as Go's `encoding/json` escapes them
/// (HTML-safe: `<` `>` `&` and U+2028/U+2029 as `\u` escapes, `\b` and `\f` short
/// forms), so a signature made by either implementation verifies in the other.
pub fn signing_message(claim: &SigningClaim) -> Vec<u8> {
    let mut fields: Vec<(&str, &str)> = vec![
        ("schema_version", &claim.schema_version),
        ("session_id", &claim.session_id),
        ("dve_id", &claim.dve_id),
        ("user_id", &claim.user_id),
        ("project_id", &claim.project_id),
        ("policy_hash", &claim.policy_hash),
        ("eventlog_root", &claim.eventlog_root),
        ("artifact_merkle_root", &claim.artifact_merkle_root),
        ("workspace_base_hash", &claim.workspace_base_hash),
        ("workspace_final_hash", &claim.workspace_final_hash),
    ];
    if !claim.event_bundle_root.is_empty() {
        fields.push(("event_bundle_root", &claim.event_bundle_root));
    }
    let mut out = String::from("{");
    for (index, (key, value)) in fields.iter().enumerate() {
        if index > 0 {
            out.push(',');
        }
        push_go_json_string(&mut out, key);
        out.push(':');
        push_go_json_string(&mut out, value);
    }
    out.push('}');
    out.into_bytes()
}

/// Appends `value` as a JSON string, escaped as Go's `encoding/json` does with
/// HTML escaping on. Pinned by `testvectors/evidence/signing_claim_v1.json`.
fn push_go_json_string(out: &mut String, value: &str) {
    out.push('"');
    for c in value.chars() {
        match c {
            '"' => out.push_str("\\\""),
            '\\' => out.push_str("\\\\"),
            '\n' => out.push_str("\\n"),
            '\r' => out.push_str("\\r"),
            '\t' => out.push_str("\\t"),
            '\u{8}' => out.push_str("\\b"),
            '\u{c}' => out.push_str("\\f"),
            '<' | '>' | '&' | '\u{2028}' | '\u{2029}' => {
                out.push_str(&format!("\\u{:04x}", c as u32))
            }
            c if (c as u32) < 0x20 => out.push_str(&format!("\\u{:04x}", c as u32)),
            c => out.push(c),
        }
    }
    out.push('"');
}

/// Ed25519 verification of evidence bundles. Gated on the `ed25519` feature so the
/// vault and relay modules, which do not use it, keep their bytes.
#[cfg(feature = "ed25519")]
mod ed25519 {
    use base64::{engine::general_purpose::STANDARD as BASE64, Engine as _};
    use ed25519_dalek::{Signature, Signer, SigningKey, Verifier, VerifyingKey};
    use sha2::{Digest, Sha256};

    /// Why an Ed25519 public key or signature was refused. `code` is the stable
    /// wire name the bindings return; the acceptance rule is RFC 8032 verification
    /// without the strict small-order checks, the rule Go's `crypto/ed25519` applies.
    #[derive(Clone, Copy, Debug, PartialEq, Eq)]
    pub enum Ed25519Failure {
        PublicKeyEncoding,
        PublicKeyLength,
        PublicKeyMalformed,
        SignatureEncoding,
        SignatureLength,
        SignatureInvalid,
        SeedEncoding,
        SeedLength,
    }

    impl Ed25519Failure {
        pub fn code(self) -> &'static str {
            match self {
                Self::PublicKeyEncoding => "public_key_encoding",
                Self::PublicKeyLength => "public_key_length",
                Self::PublicKeyMalformed => "public_key_malformed",
                Self::SignatureEncoding => "signature_encoding",
                Self::SignatureLength => "signature_length",
                Self::SignatureInvalid => "signature_invalid",
                Self::SeedEncoding => "seed_encoding",
                Self::SeedLength => "seed_length",
            }
        }

        pub fn message(self) -> &'static str {
            match self {
                Self::PublicKeyEncoding => "public key is not valid base64",
                Self::PublicKeyLength => "ed25519 public key must be 32 bytes",
                Self::PublicKeyMalformed => "ed25519 public key is not a valid curve point",
                Self::SignatureEncoding => "malformed signature encoding",
                Self::SignatureLength => "ed25519 signature must be 64 bytes",
                Self::SignatureInvalid => "signature verification failed",
                Self::SeedEncoding => "ed25519 seed is not valid base64",
                Self::SeedLength => "ed25519 seed must be 32 bytes",
            }
        }
    }

    fn parse_public_key(public_key_base64: &str) -> Result<VerifyingKey, Ed25519Failure> {
        let raw = BASE64
            .decode(public_key_base64)
            .map_err(|_| Ed25519Failure::PublicKeyEncoding)?;
        let bytes: [u8; 32] = raw
            .as_slice()
            .try_into()
            .map_err(|_| Ed25519Failure::PublicKeyLength)?;
        VerifyingKey::from_bytes(&bytes).map_err(|_| Ed25519Failure::PublicKeyMalformed)
    }

    /// Checks a standard-base64 Ed25519 public key and returns it in canonical form.
    pub fn ed25519_public_key(public_key_base64: &str) -> Result<String, Ed25519Failure> {
        let key = parse_public_key(public_key_base64)?;
        Ok(BASE64.encode(key.as_bytes()))
    }

    /// Verifies a standard-base64 Ed25519 signature over `message` under a
    /// standard-base64 public key.
    pub fn ed25519_verify(
        public_key_base64: &str,
        message: &[u8],
        signature_base64: &str,
    ) -> Result<(), Ed25519Failure> {
        let key = parse_public_key(public_key_base64)?;
        let raw = BASE64
            .decode(signature_base64)
            .map_err(|_| Ed25519Failure::SignatureEncoding)?;
        let bytes: [u8; 64] = raw
            .as_slice()
            .try_into()
            .map_err(|_| Ed25519Failure::SignatureLength)?;
        key.verify(message, &Signature::from_bytes(&bytes))
            .map_err(|_| Ed25519Failure::SignatureInvalid)
    }

    fn signing_key(seed_base64: &str) -> Result<SigningKey, Ed25519Failure> {
        let raw = BASE64
            .decode(seed_base64)
            .map_err(|_| Ed25519Failure::SeedEncoding)?;
        let seed: [u8; 32] = raw
            .as_slice()
            .try_into()
            .map_err(|_| Ed25519Failure::SeedLength)?;
        Ok(SigningKey::from_bytes(&seed))
    }

    /// Derives the standard-base64 Ed25519 public key from a 32-byte seed
    /// (RFC 8032). The host supplies the seed; the core draws no randomness.
    pub fn ed25519_public_from_seed(seed_base64: &str) -> Result<String, Ed25519Failure> {
        let key = signing_key(seed_base64)?;
        Ok(BASE64.encode(key.verifying_key().as_bytes()))
    }

    /// Signs `message` with the key derived from a 32-byte seed. Ed25519 is
    /// deterministic, so the same seed and message always give the same signature.
    pub fn ed25519_sign(seed_base64: &str, message: &[u8]) -> Result<String, Ed25519Failure> {
        let key = signing_key(seed_base64)?;
        Ok(BASE64.encode(key.sign(message).to_bytes()))
    }

    /// The key identifier for a public key: `knirv-pub-` and the first 8 bytes of
    /// SHA-256 of the key, in hex.
    pub fn ed25519_key_id(public_key_base64: &str) -> Result<String, Ed25519Failure> {
        let key = parse_public_key(public_key_base64)?;
        let digest = Sha256::digest(key.as_bytes());
        Ok(format!("knirv-pub-{}", hex::encode(&digest[..8])))
    }
    }

    #[cfg(test)]
    mod tests {
        use crate::bindings::local_operation;
        use serde_json::json;

        fn op(name: &str, payload: serde_json::Value) -> serde_json::Value {
            local_operation(name, &payload).expect("known operation").expect("ok")
        }

        #[test]
        fn a_built_chain_verifies_and_tampering_is_caught() {
            let built = op(
                "evidence.build_event_log",
                json!({"events": [
                    {"timestamp": "2026-10-31T00:00:00Z", "type": "start", "payload": {"b": 1, "a": 2}},
                    {"timestamp": "2026-10-31T00:00:01Z", "type": "end"}
                ]}),
            );
            let root = built["root"].as_str().unwrap().to_owned();
            assert!(root.starts_with("sha256:"));
            let ok = op("evidence.verify_event_log", json!({"events": built["events"], "expected_root": root}));
            assert_eq!(ok["valid"], true);
            // payload key order is part of the hash: it must round-trip as written
            assert_eq!(built["events"][0]["payload"].to_string(), r#"{"b":1,"a":2}"#);
            let mut tampered = built["events"].clone();
            tampered[1]["type"] = json!("other");
            let bad = op("evidence.verify_event_log", json!({"events": tampered, "expected_root": root}));
            assert_eq!(bad["valid"], false);
            assert_eq!(bad["failing_index"], 1);
        }

        #[test]
        fn a_missing_timestamp_is_rejected() {
            let err = local_operation("evidence.build_event_log", &json!({"events": [{"type": "x"}]})).unwrap().unwrap_err();
            assert!(err.message.contains("event 0 missing timestamp"));
        }

        #[test]
        fn merkle_root_verifies_and_normalises_prefixes() {
            let hashes = json!(["aa", "sha256:bb", " cc "]);
            let root = op("evidence.merkle_root", json!({"artifact_hashes": hashes}))["root"].as_str().unwrap().to_owned();
            assert_eq!(op("evidence.verify_artifact_merkle", json!({"artifact_hashes": hashes, "expected_root": root}))["valid"], true);
            assert_eq!(op("evidence.verify_artifact_merkle", json!({"artifact_hashes": ["aa"], "expected_root": root}))["valid"], false);
            // empty set hashes the empty string
            assert_eq!(op("evidence.merkle_root", json!({"artifact_hashes": []}))["root"], "sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855");
        }
        /// `testvectors/evidence/ed25519_v1.json`: Go `crypto/ed25519` is the reference
        /// (valid, bad signature, wrong key, tampered claim, malformed point, wrong
        /// lengths, bad encodings). Every verdict and code must match exactly.
        #[cfg(feature = "ed25519")]
        #[test]
        fn ed25519_matches_the_shared_vectors() {
            let path = format!("{}/../testvectors/evidence/ed25519_v1.json", env!("CARGO_MANIFEST_DIR"));
            let fixture: serde_json::Value = serde_json::from_slice(&std::fs::read(path).unwrap()).unwrap();
            for case in fixture["verify"].as_array().unwrap() {
                let name = case["name"].as_str().unwrap();
                let verdict = op(
                    "evidence.ed25519_verify",
                    json!({
                        "public_key_base64": case["public_key_base64"],
                        "message_base64": case["message_base64"],
                        "signature_base64": case["signature_base64"],
                    }),
                );
                assert_eq!(verdict["valid"], case["valid"], "{name}");
                assert_eq!(verdict.get("code").cloned().unwrap_or(json!(null)), case.get("code").cloned().unwrap_or(json!(null)), "{name}");
            }
            for case in fixture["public_key"].as_array().unwrap() {
                let name = case["name"].as_str().unwrap();
                let verdict = op("evidence.ed25519_public_key", json!({"public_key_base64": case["public_key_base64"]}));
                assert_eq!(verdict["valid"], case["valid"], "{name}");
                assert_eq!(verdict.get("code").cloned().unwrap_or(json!(null)), case.get("code").cloned().unwrap_or(json!(null)), "{name}");
                if let Some(canonical) = case.get("canonical_base64") {
                    assert_eq!(&verdict["public_key_base64"], canonical, "{name}");
                }
            }
        }

        #[cfg(feature = "ed25519")]
        #[test]
        fn ed25519_verify_requires_a_message() {
            let err = local_operation(
                "evidence.ed25519_verify",
                &json!({"public_key_base64": "AA==", "signature_base64": "AA=="}),
            )
            .unwrap()
            .unwrap_err();
            assert!(err.message.contains("message_base64"));
        }
}

/// `testvectors/evidence/signing_v1.json`: the Go implementation's outputs for the
/// signed claim bytes, the SHA-256 helper and Ed25519 seed/key/signature handling.
/// Every value must match exactly.
#[cfg(test)]
mod signing_vector_tests {
    use super::{signing_message, sha256_digest, SigningClaim};
    use base64::{engine::general_purpose::STANDARD as BASE64, Engine as _};
    use serde_json::Value;

    fn fixture() -> Value {
        let path = format!("{}/../testvectors/evidence/signing_v1.json", env!("CARGO_MANIFEST_DIR"));
        serde_json::from_slice(&std::fs::read(path).unwrap()).unwrap()
    }

    #[test]
    fn claim_message_matches_go_json_bytes() {
        for case in fixture()["claims"].as_array().unwrap() {
            let name = case["name"].as_str().unwrap();
            let claim: SigningClaim = serde_json::from_value(case["claim"].clone()).unwrap();
            let got = BASE64.encode(signing_message(&claim));
            assert_eq!(got, case["message_base64"].as_str().unwrap(), "{name}");
        }
    }

    #[test]
    fn sha256_matches_go_digest() {
        for case in fixture()["sha256"].as_array().unwrap() {
            let data = BASE64.decode(case["data_base64"].as_str().unwrap()).unwrap();
            assert_eq!(sha256_digest(&data), case["digest"].as_str().unwrap(), "{}", case["name"]);
        }
    }

    #[cfg(feature = "ed25519")]
    #[test]
    fn ed25519_seed_key_id_and_signature_match_go() {
        use super::{ed25519_key_id, ed25519_public_from_seed, ed25519_sign, ed25519_verify};
        for case in fixture()["signatures"].as_array().unwrap() {
            let name = case["name"].as_str().unwrap();
            let seed = case["seed_base64"].as_str().unwrap();
            let public = ed25519_public_from_seed(seed).unwrap();
            assert_eq!(public, case["public_key_base64"].as_str().unwrap(), "{name} public");
            assert_eq!(ed25519_key_id(&public).unwrap(), case["derived_key_id"].as_str().unwrap(), "{name} key id");
            let message = BASE64.decode(case["message_base64"].as_str().unwrap()).unwrap();
            let signature = ed25519_sign(seed, &message).unwrap();
            assert_eq!(signature, case["signature_base64"].as_str().unwrap(), "{name} signature");
            assert!(ed25519_verify(&public, &message, &signature).is_ok(), "{name} verify");
        }
    }

    #[cfg(feature = "ed25519")]
    #[test]
    fn ed25519_refuses_bad_seeds() {
        use super::{ed25519_public_from_seed, Ed25519Failure};
        assert_eq!(ed25519_public_from_seed("AAAA").unwrap_err(), Ed25519Failure::SeedLength);
        assert_eq!(ed25519_public_from_seed("!!").unwrap_err(), Ed25519Failure::SeedEncoding);
    }
}
