//! Raw WASM ABI v1 (KNIRV_CORP/packages/controller/stateless_pwa_controller.md
//! section 6.2) for the dve_verifier module. No wasm-bindgen/js-sys: every
//! export is a plain `extern "C"` function operating on linear-memory
//! pointer/length pairs, matching the pattern already proven live for
//! controller_relay (KNIRV_CORP/packages/controller/wasm-modules/controller_relay).
//! This is deliberate — downloaded WASM must never pull in dynamically
//! generated JS glue (plan section 6.2), and the host never needs anything
//! wasm-bindgen provides here: every operation's input/output is JSON or raw
//! bytes.
//!
//! eventlog.rs and mmr.rs contain the actual verification algorithms and are
//! unchanged by this ABI rewrite — they have never depended on wasm-bindgen.

pub mod eventlog;
pub mod mmr;

use std::cell::RefCell;
use std::collections::HashMap;

use serde::Deserialize;

// module_kind values shared with controller_wasm_artifacts.module_kind
// ('crypto', 'dve_verifier', 'relay') and with controller_relay's
// moduleKindRelay = 3 convention — 2 = dve_verifier.
const MODULE_KIND_DVE_VERIFIER: i32 = 2;
const ABI_VERSION: i32 = 1;

// invoke() operation codes.
const OP_EMPTY_ROOT: i32 = 1;
const OP_LEAF_HASH: i32 = 2;
const OP_PARENT_HASH: i32 = 3;
const OP_VERIFY_MMR_PROOF: i32 = 4;
const OP_VERIFY_EVENT_LOG: i32 = 5;
const OP_VERIFY_ARTIFACT_MERKLE: i32 = 6;

thread_local! {
    // Keyed by an opaque handle (not a memory pointer); result_pointer
    // translates a handle to the real address for reading.
    static RESULTS: RefCell<HashMap<i32, Vec<u8>>> = RefCell::new(HashMap::new());
    static NEXT_RESULT_HANDLE: RefCell<i32> = const { RefCell::new(1) };
}

fn store_result(bytes: Vec<u8>) -> i32 {
    NEXT_RESULT_HANDLE.with(|next| {
        let handle = *next.borrow();
        *next.borrow_mut() = handle + 1;
        RESULTS.with(|results| results.borrow_mut().insert(handle, bytes));
        handle
    })
}

fn error_json(message: &str) -> Vec<u8> {
    serde_json::json!({ "error": message })
        .to_string()
        .into_bytes()
}

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

#[no_mangle]
pub extern "C" fn abi_version() -> i32 {
    ABI_VERSION
}

#[no_mangle]
pub extern "C" fn module_kind() -> i32 {
    MODULE_KIND_DVE_VERIFIER
}

#[no_mangle]
pub extern "C" fn allocate(length: i32) -> i32 {
    if length <= 0 {
        return 0;
    }
    // Zero-initialized rather than Vec::with_capacity + set_len: the host
    // overwrites this region before it's read either way, but leaving it
    // formally uninitialized is UB per Rust's abstract machine even though
    // the uninitialized bytes are themselves never observed.
    let mut buf: Vec<u8> = vec![0u8; length as usize];
    let ptr = buf.as_mut_ptr() as i32;
    std::mem::forget(buf); // reclaimed explicitly by deallocate()
    ptr
}

#[no_mangle]
pub extern "C" fn deallocate(ptr: i32, length: i32) {
    if ptr == 0 || length <= 0 {
        return;
    }
    // SAFETY: ptr/length are exactly what allocate() returned/was called
    // with and forgot; reconstructing the Vec here and letting it drop is
    // the intended matching deallocation.
    unsafe {
        drop(Vec::from_raw_parts(
            ptr as *mut u8,
            length as usize,
            length as usize,
        ));
    }
}

#[no_mangle]
pub extern "C" fn result_pointer(handle: i32) -> i32 {
    RESULTS.with(|results| {
        results
            .borrow()
            .get(&handle)
            .filter(|bytes| !bytes.is_empty())
            .map(|bytes| bytes.as_ptr() as i32)
            .unwrap_or(0)
    })
}

#[no_mangle]
pub extern "C" fn result_length(handle: i32) -> i32 {
    RESULTS.with(|results| {
        results
            .borrow()
            .get(&handle)
            .map(|bytes| bytes.len() as i32)
            .unwrap_or(0)
    })
}

#[no_mangle]
pub extern "C" fn free_result(handle: i32) {
    RESULTS.with(|results| {
        results.borrow_mut().remove(&handle);
    });
}

#[no_mangle]
pub extern "C" fn zeroize() {
    RESULTS.with(|results| results.borrow_mut().clear());
}

/// # Safety
/// `ptr`/`len` must be a currently-allocated (not yet deallocated) region
/// returned by `allocate`, with the host having written exactly `len` bytes
/// into it.
unsafe fn input_bytes(ptr: i32, len: i32) -> &'static [u8] {
    if ptr == 0 || len <= 0 {
        return &[];
    }
    std::slice::from_raw_parts(ptr as *const u8, len as usize)
}

#[no_mangle]
pub extern "C" fn self_test(_input_ptr: i32, _input_len: i32) -> i32 {
    let root = mmr::empty_root();
    let leaf = mmr::leaf_hash(b"self-test-leaf");
    // In a 2-leaf MMR, leaf index 0 is the left child of the sole parent, so
    // its proof sibling side is Right (see mmr::proof_sides) and the parent
    // is parent_hash(leaf, sibling) — leaf first, not sibling first.
    let parent = mmr::parent_hash(leaf, root);
    // Round-trips through the exact hex encode/decode path invoke() uses, to
    // prove the WASM-facing encoding layer is internally consistent — the
    // underlying algorithms are separately covered by mmr.rs's and
    // eventlog.rs's own #[cfg(test)] suites against the shared Go golden
    // vectors (mmr_golden.json / eventlog_golden.json).
    let ok = decode_hash(&encode_hash(root)) == Some(root)
        && decode_hash(&encode_hash(leaf)) == Some(leaf)
        && decode_hash(&encode_hash(parent)) == Some(parent)
        && mmr::verify_proof(
            parent,
            leaf,
            &mmr::Proof {
                leaf_index: 0,
                tree_size: 2,
                path: vec![mmr::Sibling {
                    hash: root,
                    side: mmr::Side::Right,
                }],
            },
            2,
        );
    let payload = if ok {
        serde_json::json!({ "self_test": "pass", "abi_version": ABI_VERSION, "module_kind": "dve_verifier" })
    } else {
        serde_json::json!({ "self_test": "fail", "error": "hash/proof round trip mismatch" })
    };
    store_result(payload.to_string().into_bytes())
}

#[derive(Deserialize)]
struct ParentHashInput {
    left: String,
    right: String,
}

fn handle_parent_hash(input: &[u8]) -> Vec<u8> {
    let parsed: ParentHashInput = match serde_json::from_slice(input) {
        Ok(v) => v,
        Err(_) => return error_json("invalid parent_hash input"),
    };
    match (decode_hash(&parsed.left), decode_hash(&parsed.right)) {
        (Some(left), Some(right)) => {
            serde_json::json!({ "hash": encode_hash(mmr::parent_hash(left, right)) })
                .to_string()
                .into_bytes()
        }
        _ => error_json("invalid hash hex"),
    }
}

#[derive(Deserialize)]
struct MmrSibling {
    hash: String,
    side: String,
}

#[derive(Deserialize)]
struct VerifyMmrProofInput {
    expected_root: String,
    leaf: String,
    leaf_index: u64,
    tree_size: u64,
    path: Vec<MmrSibling>,
}

fn handle_verify_mmr_proof(input: &[u8]) -> Vec<u8> {
    let parsed: VerifyMmrProofInput = match serde_json::from_slice(input) {
        Ok(v) => v,
        Err(_) => return error_json("invalid verify_mmr_proof input"),
    };
    let (Some(root), Some(leaf)) = (
        decode_hash(&parsed.expected_root),
        decode_hash(&parsed.leaf),
    ) else {
        return error_json("invalid hash hex");
    };
    let mut path = Vec::with_capacity(parsed.path.len());
    for sibling in &parsed.path {
        let Some(hash) = decode_hash(&sibling.hash) else {
            return error_json("invalid sibling hash hex");
        };
        let side = match sibling.side.as_str() {
            "left" => mmr::Side::Left,
            "right" => mmr::Side::Right,
            _ => return error_json("sibling side must be \"left\" or \"right\""),
        };
        path.push(mmr::Sibling { hash, side });
    }
    let valid = mmr::verify_proof(
        root,
        leaf,
        &mmr::Proof {
            leaf_index: parsed.leaf_index,
            tree_size: parsed.tree_size,
            path,
        },
        parsed.tree_size,
    );
    serde_json::json!({ "valid": valid })
        .to_string()
        .into_bytes()
}

#[derive(Deserialize)]
struct VerifyEventLogInput {
    events: serde_json::Value,
    expected_root: String,
}

fn handle_verify_event_log(input: &[u8]) -> Vec<u8> {
    let parsed: VerifyEventLogInput = match serde_json::from_slice(input) {
        Ok(v) => v,
        Err(_) => return error_json("invalid verify_event_log input"),
    };
    let events_json = match serde_json::to_string(&parsed.events) {
        Ok(s) => s,
        Err(_) => return error_json("invalid events payload"),
    };
    // eventlog::verify_json already returns a complete JSON result object
    // (valid/tip/failing_index/error) — pass it through as-is rather than
    // re-wrapping it.
    eventlog::verify_json(&events_json, &parsed.expected_root).into_bytes()
}

#[derive(Deserialize)]
struct VerifyArtifactMerkleInput {
    artifact_hashes: Vec<String>,
    expected_root: String,
}

fn handle_verify_artifact_merkle(input: &[u8]) -> Vec<u8> {
    let parsed: VerifyArtifactMerkleInput = match serde_json::from_slice(input) {
        Ok(v) => v,
        Err(_) => return error_json("invalid verify_artifact_merkle input"),
    };
    let valid = eventlog::artifact_merkle_root(&parsed.artifact_hashes) == parsed.expected_root;
    serde_json::json!({ "valid": valid })
        .to_string()
        .into_bytes()
}

#[no_mangle]
pub extern "C" fn invoke(operation: i32, input_ptr: i32, input_len: i32) -> i32 {
    // SAFETY: input_ptr/input_len come from the host and describe a region
    // previously returned by allocate() that the host has written into.
    let input = unsafe { input_bytes(input_ptr, input_len) };
    let output = match operation {
        OP_EMPTY_ROOT => serde_json::json!({ "root": encode_hash(mmr::empty_root()) })
            .to_string()
            .into_bytes(),
        OP_LEAF_HASH => serde_json::json!({ "hash": encode_hash(mmr::leaf_hash(input)) })
            .to_string()
            .into_bytes(),
        OP_PARENT_HASH => handle_parent_hash(input),
        OP_VERIFY_MMR_PROOF => handle_verify_mmr_proof(input),
        OP_VERIFY_EVENT_LOG => handle_verify_event_log(input),
        OP_VERIFY_ARTIFACT_MERKLE => handle_verify_artifact_merkle(input),
        _ => error_json("unsupported operation"),
    };
    store_result(output)
}
