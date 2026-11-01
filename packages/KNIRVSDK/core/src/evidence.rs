//! Evidence-log and artifact-Merkle verification. The algorithms live in the
//! `proof-verifier` module's sources and are compiled into the core from there,
//! so the standalone WASM module and the core can never drift.

#[path = "../wasm-modules/proof-verifier/src/eventlog.rs"]
pub mod eventlog;
#[path = "../wasm-modules/proof-verifier/src/mmr.rs"]
pub mod mmr;

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
}
