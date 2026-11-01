//! The KNIRV wallet extension's message protocol: request kinds and the success,
//! failure and reject outcomes, each with a stable numeric code and message.
//! `data/wallet_protocol.json` is the one place it is written; the npm build
//! generates `@knirv/sdk/wallet` from the same file, and `wallet.protocol` returns
//! it to any other host.

use serde_json::Value;
use std::sync::OnceLock;

pub fn protocol() -> &'static Value {
    static PROTOCOL: OnceLock<Value> = OnceLock::new();
    PROTOCOL.get_or_init(|| {
        serde_json::from_str(include_str!("../data/wallet_protocol.json")).expect("wallet_protocol.json is valid")
    })
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::collections::BTreeSet;

    #[test]
    fn every_message_has_a_unique_code_in_its_family_range() {
        let p = protocol();
        let mut keys = BTreeSet::new();
        let mut codes = BTreeSet::new();
        for (family, base) in [("execute", 1000), ("success", 2000), ("failure", 3000), ("reject", 4000)] {
            for (key, entry) in p[family].as_object().unwrap() {
                let code = entry[0].as_u64().unwrap();
                assert!((base..base + 1000).contains(&code), "{key} is outside the {family} range");
                assert!(!entry[1].as_str().unwrap().is_empty(), "{key} has no message");
                assert!(keys.insert(key.clone()), "{key} appears in two families");
                assert!(codes.insert(code), "code {code} is used twice");
            }
        }
        assert_eq!(keys.len(), 8 + 7 + 15 + 5);
    }
}
