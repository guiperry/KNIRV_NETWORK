use serde::{Deserialize, Serialize};
use serde_json::Value;
use sha2::{Digest, Sha256};

#[derive(Debug, Deserialize)]
struct Event {
    index: usize,
    timestamp: String,
    #[serde(rename = "type")]
    event_type: String,
    #[serde(default)]
    payload: Option<Value>,
    #[serde(default)]
    prev_hash: String,
    #[serde(default)]
    hash: String,
}

#[derive(Serialize)]
struct CanonicalEvent<'a> {
    index: usize,
    timestamp: &'a str,
    #[serde(rename = "type")]
    event_type: &'a str,
    #[serde(skip_serializing_if = "Option::is_none")]
    payload: &'a Option<Value>,
    #[serde(skip_serializing_if = "str::is_empty")]
    prev_hash: &'a str,
}

#[derive(Serialize)]
struct VerificationResult {
    valid: bool,
    tip: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    failing_index: Option<usize>,
    #[serde(skip_serializing_if = "Option::is_none")]
    error: Option<String>,
}

fn sha256_prefixed(data: &[u8]) -> String {
    let digest = Sha256::digest(data);
    let hex: String = digest.iter().map(|byte| format!("{byte:02x}")).collect();
    format!("sha256:{hex}")
}

fn event_hash(event: &Event, previous: &str) -> Result<String, serde_json::Error> {
    serde_json::to_vec(&CanonicalEvent {
        index: event.index,
        timestamp: &event.timestamp,
        event_type: &event.event_type,
        payload: &event.payload,
        prev_hash: previous,
    })
    .map(|canonical| sha256_prefixed(&canonical))
}

pub fn verify_json(events_json: &str, expected_root: &str) -> String {
    let events = match serde_json::from_str::<Vec<Event>>(events_json) {
        Ok(events) => events,
        Err(error) => {
            return result(
                false,
                "",
                None,
                Some(format!("invalid events JSON: {error}")),
            )
        }
    };
    let mut previous = String::new();
    for (index, event) in events.iter().enumerate() {
        if event.index != index {
            return result(
                false,
                &previous,
                Some(index),
                Some("event index mismatch".into()),
            );
        }
        if event.prev_hash != previous {
            return result(
                false,
                &previous,
                Some(index),
                Some("broken prev_hash link".into()),
            );
        }
        let computed = match event_hash(event, &previous) {
            Ok(hash) => hash,
            Err(error) => return result(false, &previous, Some(index), Some(error.to_string())),
        };
        if event.hash != computed {
            return result(
                false,
                &previous,
                Some(index),
                Some("event hash mismatch".into()),
            );
        }
        previous = computed;
    }
    if previous != expected_root {
        return result(
            false,
            &previous,
            None,
            Some("event-log root mismatch".into()),
        );
    }
    result(true, &previous, None, None)
}

fn result(valid: bool, tip: &str, failing_index: Option<usize>, error: Option<String>) -> String {
    serde_json::to_string(&VerificationResult {
        valid,
        tip: tip.into(),
        failing_index,
        error,
    })
    .unwrap_or_else(|_| "{\"valid\":false,\"tip\":\"\",\"error\":\"serialization failed\"}".into())
}

pub fn artifact_merkle_root(hashes: &[String]) -> String {
    if hashes.is_empty() {
        return sha256_prefixed(&[]);
    }
    let mut level: Vec<String> = hashes
        .iter()
        .map(|hash| {
            let trimmed = hash.trim();
            if trimmed.starts_with("sha256:") {
                trimmed.into()
            } else {
                format!("sha256:{trimmed}")
            }
        })
        .collect();
    while level.len() > 1 {
        let mut next = Vec::with_capacity((level.len() + 1) / 2);
        for pair in level.chunks(2) {
            let right = pair.get(1).unwrap_or(&pair[0]);
            next.push(sha256_prefixed(format!("{}|{}", pair[0], right).as_bytes()));
        }
        level = next;
    }
    level.remove(0)
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn verifies_go_canonical_event_chain() {
        let vector: Value = serde_json::from_str(include_str!(
            "../../../internal/dveevidence/testdata/eventlog_golden.json"
        ))
        .unwrap();
        let events = serde_json::to_string(&vector["events"]).unwrap();
        let report: Value =
            serde_json::from_str(&verify_json(&events, vector["root"].as_str().unwrap())).unwrap();
        assert_eq!(report["valid"], true);
    }

    #[test]
    fn artifact_merkle_matches_go_rules() {
        assert_eq!(
            artifact_merkle_root(&[]),
            "sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
        );
        assert_eq!(artifact_merkle_root(&["aaa".into()]), "sha256:aaa");
    }
}
