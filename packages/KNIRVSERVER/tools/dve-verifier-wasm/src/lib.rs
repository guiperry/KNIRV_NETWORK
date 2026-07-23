pub mod eventlog;
pub mod mmr;

use js_sys::Array;
use wasm_bindgen::prelude::*;

fn decode_hash(value: &str) -> Option<mmr::Hash> {
    let value = value.strip_prefix("sha256:").unwrap_or(value);
    if value.len() != 64 {
        return None;
    }
    let mut hash = [0u8; 32];
    for (index, byte) in hash.iter_mut().enumerate() {
        *byte = u8::from_str_radix(&value[index * 2..index * 2 + 2], 16).ok()?;
    }
    Some(hash)
}

fn encode_hash(hash: mmr::Hash) -> String {
    hash.iter().map(|byte| format!("{byte:02x}")).collect()
}

#[wasm_bindgen(js_name = empty_root)]
pub fn wasm_empty_root() -> String {
    encode_hash(mmr::empty_root())
}

#[wasm_bindgen(js_name = leaf_hash)]
pub fn wasm_leaf_hash(data: &[u8]) -> String {
    encode_hash(mmr::leaf_hash(data))
}

#[wasm_bindgen(js_name = parent_hash)]
pub fn wasm_parent_hash(left: &str, right: &str) -> Option<String> {
    Some(encode_hash(mmr::parent_hash(
        decode_hash(left)?,
        decode_hash(right)?,
    )))
}

#[wasm_bindgen]
pub fn verify_mmr_proof(
    expected_root: &str,
    leaf: &str,
    leaf_index: u64,
    tree_size: u64,
    sibling_hashes: Array,
    sibling_sides: Array,
) -> bool {
    if sibling_hashes.length() != sibling_sides.length() {
        return false;
    }
    let (Some(root), Some(leaf)) = (decode_hash(expected_root), decode_hash(leaf)) else {
        return false;
    };
    let mut path = Vec::with_capacity(sibling_hashes.length() as usize);
    for index in 0..sibling_hashes.length() {
        let Some(hash_text) = sibling_hashes.get(index).as_string() else {
            return false;
        };
        let Some(hash) = decode_hash(&hash_text) else {
            return false;
        };
        let side = match sibling_sides.get(index).as_string().as_deref() {
            Some("left") => mmr::Side::Left,
            Some("right") => mmr::Side::Right,
            _ => return false,
        };
        path.push(mmr::Sibling { hash, side });
    }
    mmr::verify_proof(
        root,
        leaf,
        &mmr::Proof {
            leaf_index,
            tree_size,
            path,
        },
        tree_size,
    )
}

#[wasm_bindgen]
pub fn verify_event_log(events_json: &str, expected_root: &str) -> String {
    eventlog::verify_json(events_json, expected_root)
}

#[wasm_bindgen]
pub fn verify_artifact_merkle(artifact_hashes_json: &str, expected_root: &str) -> bool {
    let Ok(hashes) = serde_json::from_str::<Vec<String>>(artifact_hashes_json) else {
        return false;
    };
    eventlog::artifact_merkle_root(&hashes) == expected_root
}
