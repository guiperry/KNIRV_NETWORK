//! The five network operations through the host-import path, answered from the
//! recorded responses in `bindings-tests/network.json`. The Go and TypeScript
//! hosts run the same file.

use base64::{engine::general_purpose::STANDARD as BASE64, Engine as _};
use knirv_sdk::{call_json_with_transport, JsonTransport};
use serde_json::{json, Value};
use std::{path::Path, sync::Arc};

fn is_subset(expected: &Value, actual: &Value) -> bool {
    match (expected, actual) {
        (Value::Object(e), Value::Object(a)) => e
            .iter()
            .all(|(k, v)| a.get(k).is_some_and(|av| is_subset(v, av))),
        _ => expected == actual,
    }
}

#[test]
fn recorded_network_cases() {
    let path = Path::new(env!("CARGO_MANIFEST_DIR")).join("../bindings-tests/network.json");
    let fixtures: Value = serde_json::from_slice(&std::fs::read(path).unwrap()).unwrap();
    for case in fixtures["cases"].as_array().unwrap() {
        let name = case["name"].as_str().unwrap();
        let host = case["host"].clone();
        let transport = JsonTransport::new(move |raw| {
            let req: Value = serde_json::from_slice(raw).unwrap();
            let key = format!("{} {}", req["method"].as_str().unwrap(), req["url"].as_str().unwrap());
            let Some(answer) = host.get(&key) else {
                panic!("unexpected host request: {key}");
            };
            if answer.get("error").is_some() {
                return serde_json::to_vec(answer).unwrap();
            }
            serde_json::to_vec(&json!({
                "status": answer["status"],
                "headers": {},
                "body_base64": BASE64.encode(serde_json::to_vec(&answer["body"]).unwrap()),
            }))
            .unwrap()
        });
        let request = serde_json::to_vec(&case["request"]).unwrap();
        let response: Value =
            serde_json::from_slice(&call_json_with_transport(&request, Arc::new(transport))).unwrap();
        assert!(is_subset(&case["expect"], &response), "{name}: {response}");
    }
}
