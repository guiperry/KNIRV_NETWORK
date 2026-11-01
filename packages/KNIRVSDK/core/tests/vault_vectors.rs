//! Holds the Rust vault to the golden vectors captured from the controller's
//! JavaScript vault worker (`testvectors/vault/generate.mjs`).
#![cfg(feature = "vault")]

use base64::{engine::general_purpose::STANDARD as BASE64, Engine as _};
use knirv_sdk::vault::{self, Vault};
use serde_json::{json, Value};
use std::path::Path;

fn vectors() -> Value {
    let path = Path::new(env!("CARGO_MANIFEST_DIR")).join("../testvectors/vault/vectors.json");
    serde_json::from_slice(&std::fs::read(path).unwrap()).unwrap()
}
fn unhex(s: &str) -> Vec<u8> {
    (0..s.len()).step_by(2).map(|i| u8::from_str_radix(&s[i..i + 2], 16).unwrap()).collect()
}
fn b64(s: &str) -> String {
    BASE64.encode(unhex(s))
}

#[test]
fn self_test_passes() {
    assert!(vault::self_test());
}

#[test]
fn derivation_addresses_and_signatures_match_the_js_worker() {
    let v = vectors();
    for account in v["accounts"].as_array().unwrap() {
        let mut vault = Vault::new();
        // import needs a password and randomness; the account data is what we compare.
        let imported = vault
            .handle(
                "vault.import",
                &json!({
                    "mnemonic": account["mnemonic"], "password": "pw", "session_id": "s",
                    "salt": b64(&"00".repeat(16)), "nonce": b64(&"00".repeat(12)), "created_at": "t"
                }),
            )
            .unwrap();
        assert_eq!(imported["account"]["address"], account["derived"][0]["address"], "{}", account["name"]);
        for d in account["derived"].as_array().unwrap() {
            let got = vault.handle("session.account", &json!({"session_id": "s", "index": d["index"]})).unwrap();
            assert_eq!(got["address"], d["address"], "{} #{}", account["name"], d["index"]);
            assert_eq!(got["public_key"], d["public_key"]);
            if d.get("sign_tx").is_some() {
                let tx = vault.handle("session.sign_tx", &json!({"session_id": "s", "index": d["index"], "tx": v["sign_inputs"]["tx"]})).unwrap();
                assert_eq!(tx["signatures"], d["sign_tx"]["signatures"]);
                assert_eq!(tx, d["sign_tx"]);
                let msg = vault.handle("session.sign_msg", &json!({"session_id": "s", "index": d["index"], "envelope": v["sign_inputs"]["message"]})).unwrap();
                assert_eq!(msg, d["sign_msg"]);
            }
            let addr = vault.handle("address.from_pubkey", &json!({"pubkey": d["public_key"]})).unwrap();
            assert_eq!(addr["address"], d["address"]);
        }
    }
}

#[test]
fn create_from_entropy_gives_the_js_mnemonic() {
    let v = vectors();
    for account in v["accounts"].as_array().unwrap().iter().filter(|a| a["entropy"].is_string()) {
        let mut vault = Vault::new();
        let out = vault
            .handle(
                "vault.create",
                &json!({
                    "entropy": b64(account["entropy"].as_str().unwrap()), "password": "pw", "session_id": "s",
                    "salt": b64(&"01".repeat(16)), "nonce": b64(&"02".repeat(12)), "created_at": "t"
                }),
            )
            .unwrap();
        assert_eq!(out["recovery_phrase"], account["mnemonic"]);
        assert_eq!(out["account"]["address"], account["derived"][0]["address"]);
    }
}

#[test]
fn sealing_reproduces_the_js_envelope_byte_for_byte_and_it_unlocks() {
    let v = vectors();
    for sealed in v["seal"].as_array().unwrap() {
        let account = v["accounts"].as_array().unwrap().iter().find(|a| a["name"] == sealed["account"]).unwrap();
        let mut vault = Vault::new();
        let out = vault
            .handle(
                "vault.import",
                &json!({
                    "mnemonic": account["mnemonic"], "password": sealed["password"], "session_id": "a",
                    "salt": b64(sealed["salt"].as_str().unwrap()), "nonce": b64(sealed["nonce"].as_str().unwrap()),
                    "created_at": sealed["created_at"]
                }),
            )
            .unwrap();
        assert_eq!(out["envelope"], sealed["serialized"], "{}", sealed["account"]);
        // and a JS-sealed vault opens in Rust
        let unlocked = vault
            .handle("vault.unlock", &json!({"serialized": sealed["serialized"], "password": sealed["password"], "session_id": "b"}))
            .unwrap();
        assert_eq!(unlocked["account"]["address"], account["derived"][0]["address"]);
        assert_eq!(unlocked["migrated"], false);
    }
}

#[test]
fn errors_match() {
    let v = vectors();
    let mut vault = Vault::new();
    let rand = json!({"password": "pw", "session_id": "s", "salt": b64(&"00".repeat(16)), "nonce": b64(&"00".repeat(12)), "created_at": "t"});
    for case in v["errors"].as_array().unwrap() {
        match case["case"].as_str().unwrap() {
            "bad-checksum-mnemonic" | "unknown-words-mnemonic" => {
                let mut p = rand.clone();
                p["mnemonic"] = case["mnemonic"].clone();
                let err = vault.handle("vault.import", &p).unwrap_err();
                assert_eq!(err, case["js_error"].as_str().unwrap(), "{}", case["case"]);
            }
            "wrong-password" => {
                let err = vault
                    .handle("vault.unlock", &json!({"serialized": case["serialized"], "password": case["password"], "session_id": "w"}))
                    .unwrap_err();
                assert_eq!(err, "Incorrect password or unreadable vault");
            }
            _ => {}
        }
    }
    assert_eq!(vault.session_count(), 0, "failed operations must not leave sessions behind");
}

#[test]
fn whitespace_is_normalised_like_the_worker() {
    let v = vectors();
    let mut vault = Vault::new();
    let out = vault
        .handle(
            "vault.import",
            &json!({"mnemonic": v["normalisation"]["input"], "password": "pw", "session_id": "s",
                "salt": b64(&"00".repeat(16)), "nonce": b64(&"00".repeat(12)), "created_at": "t"}),
        )
        .unwrap();
    assert_eq!(out["account"]["address"], v["accounts"][0]["derived"][0]["address"]);
}

#[test]
fn tampering_and_bad_envelopes_are_rejected() {
    let v = vectors();
    let sealed = v["seal"][0]["serialized"].as_str().unwrap();
    let mut vault = Vault::new();
    let unlock = |vault: &mut Vault, s: &str| vault.handle("vault.unlock", &json!({"serialized": s, "password": v["seal"][0]["password"], "session_id": "t"}));
    assert_eq!(unlock(&mut vault, "garbage").unwrap_err(), "Invalid vault envelope");
    let body = BASE64.decode(sealed.strip_prefix(vault::ENVELOPE_PREFIX).unwrap()).unwrap();
    let mut env: Value = serde_json::from_slice(&body).unwrap();
    env["kdf"]["ops_limit"] = json!(1);
    let weak = format!("{}{}", vault::ENVELOPE_PREFIX, BASE64.encode(serde_json::to_vec(&env).unwrap()));
    assert_eq!(unlock(&mut vault, &weak).unwrap_err(), "Unsupported vault KDF parameters");
    let mut env: Value = serde_json::from_slice(&body).unwrap();
    let mut ct = BASE64.decode(env["ciphertext"].as_str().unwrap()).unwrap();
    ct[0] ^= 1;
    env["ciphertext"] = json!(BASE64.encode(ct));
    let flipped = format!("{}{}", vault::ENVELOPE_PREFIX, BASE64.encode(serde_json::to_vec(&env).unwrap()));
    assert_eq!(unlock(&mut vault, &flipped).unwrap_err(), "Incorrect password or unreadable vault");
}

#[test]
fn migrate_returns_a_verified_envelope_and_close_and_zeroize_drop_sessions() {
    let v = vectors();
    let mut vault = Vault::new();
    let out = vault
        .handle(
            "vault.migrate",
            &json!({"mnemonic": v["accounts"][0]["mnemonic"], "password": "pw", "session_id": "m",
                "salt": b64(&"05".repeat(16)), "nonce": b64(&"06".repeat(12)), "created_at": "t"}),
        )
        .unwrap();
    assert_eq!(out["migrated"], true);
    assert!(out["envelope"].as_str().unwrap().starts_with(vault::ENVELOPE_PREFIX));
    assert_eq!(vault.session_count(), 1);
    vault.handle("session.close", &json!({"session_id": "m"})).unwrap();
    assert_eq!(vault.session_count(), 0);
    assert!(vault.handle("session.account", &json!({"session_id": "m"})).is_err());
    vault.handle("vault.unlock", &json!({"serialized": out["envelope"], "password": "pw", "session_id": "x"})).unwrap();
    vault.handle("vault.unlock", &json!({"serialized": out["envelope"], "password": "pw", "session_id": "y"})).unwrap();
    assert_eq!(vault.session_count(), 2);
    vault.zeroize_all();
    assert_eq!(vault.session_count(), 0);
}
