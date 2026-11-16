//! Evidence-log and artifact-Merkle verification. The algorithms live in the
//! `proof-verifier` module's sources and are compiled into the core from there,
//! so the standalone WASM module and the core can never drift.

#[path = "../wasm-modules/proof-verifier/src/eventlog.rs"]
pub mod eventlog;
#[path = "../wasm-modules/proof-verifier/src/mmr.rs"]
pub mod mmr;

#[cfg(feature = "ed25519")]
pub use ed25519::{ed25519_public_key, ed25519_verify, Ed25519Failure};

/// Ed25519 verification of evidence bundles. Gated on the `ed25519` feature so the
/// vault and relay modules, which do not use it, keep their bytes.
#[cfg(feature = "ed25519")]
mod ed25519 {
    use base64::{engine::general_purpose::STANDARD as BASE64, Engine as _};
    use ed25519_dalek::{Signature, Verifier, VerifyingKey};

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
