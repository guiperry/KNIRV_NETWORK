use std::{cell::RefCell, collections::HashMap};

const MAX_PAYLOAD_BYTES: usize = 4 * 1024 * 1024;

#[derive(Default)]
struct SequenceState {
    map: HashMap<String, i32>,
}

impl SequenceState {
    fn check_and_update(&mut self, key: &str, seq: i32) -> bool {
        let last = self.map.get(key).copied().unwrap_or(0);
        if seq <= last {
            return false;
        }
        self.map.insert(key.to_string(), seq);
        true
    }
}

thread_local! { static SEQUENCE_STATE: RefCell<SequenceState> = RefCell::new(SequenceState::default()); }
fn with_sequence_state<T>(callback: impl FnOnce(&mut SequenceState) -> T) -> T {
    SEQUENCE_STATE.with(|state| callback(&mut state.borrow_mut()))
}

struct ResultPool {
    pool: Vec<Vec<u8>>,
}

impl ResultPool {
    fn new() -> Self {
        Self { pool: Vec::new() }
    }

    fn push(&mut self, bytes: Vec<u8>) -> usize {
        let handle = self.pool.len();
        self.pool.push(bytes);
        handle
    }

    fn get(&self, handle: usize) -> Option<&Vec<u8>> {
        self.pool.get(handle)
    }

    fn clear(&mut self) {
        self.pool.clear();
    }
}

thread_local! { static RESULT_POOL: RefCell<ResultPool> = RefCell::new(ResultPool::new()); }
fn with_result_pool<T>(callback: impl FnOnce(&mut ResultPool) -> T) -> T {
    RESULT_POOL.with(|pool| callback(&mut pool.borrow_mut()))
}

#[no_mangle]
pub extern "C" fn abi_version() -> i32 {
    1
}

#[no_mangle]
pub extern "C" fn module_kind() -> i32 {
    3
}

#[no_mangle]
pub extern "C" fn allocate(length: usize) -> usize {
    if length == 0 || length > MAX_PAYLOAD_BYTES {
        return 0;
    }
    let mut buf = vec![0u8; length];
    let ptr = buf.as_mut_ptr() as usize;
    std::mem::forget(buf);
    ptr
}

#[no_mangle]
pub extern "C" fn deallocate(_ptr: usize, _length: usize) {
    // Adapter-managed lifetime; no-op here.
}

#[no_mangle]
pub extern "C" fn zeroize() {
    // No persistent sensitive state beyond request scope.
}

#[no_mangle]
pub extern "C" fn self_test(_input_pointer: *const u8, _input_length: usize) -> i32 {
    if abi_version() != 1 || module_kind() != 3 {
        return -1;
    }
    let msg = r#"{"status":"ok"}"#;
    put_result(msg)
}

#[no_mangle]
pub extern "C" fn result_pointer(handle: i32) -> usize {
    let handle = handle as usize;
    match RESULT_POOL.with(|pool| {
        pool.borrow()
            .get(handle)
            .map(|bytes| bytes.as_ptr() as usize)
    }) {
        Some(pointer) => pointer,
        None => 0,
    }
}

#[no_mangle]
pub extern "C" fn result_length(handle: i32) -> usize {
    let handle = handle as usize;
    match RESULT_POOL.with(|pool| pool.borrow().get(handle).map(|bytes| bytes.len())) {
        Some(length) => length,
        None => 0,
    }
}

#[no_mangle]
pub extern "C" fn free_result(handle: i32) {
    let handle = handle as usize;
    with_result_pool(|pool| {
        if handle < pool.pool.len() {
            pool.pool[handle].clear();
        }
    });
}

#[no_mangle]
pub extern "C" fn invoke(
    operation_pointer: *const u8,
    operation_length: usize,
    input_pointer: *const u8,
    input_length: usize,
) -> i32 {
    let op = unsafe {
        let slice = std::slice::from_raw_parts(operation_pointer, operation_length);
        String::from_utf8_lossy(slice).into_owned()
    };

    match op.as_str() {
        "validate_envelope" => __validate_envelope(input_pointer, input_length),
        "canonicalize" => __canonicalize(input_pointer, input_length),
        "compute_digest" => __compute_digest(input_pointer, input_length),
        "check_sequence" => __check_sequence(input_pointer, input_length),
        _ => {
            let err = r#"{"error":"unknown_operation"}"#;
            put_result(err)
        }
    }
}

fn __validate_envelope(ptr: *const u8, len: usize) -> i32 {
    let json = unsafe {
        let slice = std::slice::from_raw_parts(ptr, len);
        String::from_utf8_lossy(slice).into_owned()
    };

    let mut request_id = String::new();
    let mut user_subject = String::new();
    let mut device_id = String::new();
    let mut dve_id = String::new();
    let mut target_type = String::new();
    let mut target_id = String::new();
    let mut capability = String::new();
    let mut sequence: i32 = 0;
    let mut issued_at_unix: i32 = 0;
    let mut expires_at_unix: i32 = 0;
    let mut payload_digest = String::new();

    for (key, value) in json_key_value_pairs(&json) {
        match key.as_str() {
            "request_id" => request_id = value,
            "user_subject" => user_subject = value,
            "device_id" => device_id = value,
            "dve_id" => dve_id = value,
            "target_type" => target_type = value,
            "target_id" => target_id = value,
            "capability" => capability = value,
            "sequence" => sequence = value.parse().unwrap_or(0),
            "issued_at_unix" => issued_at_unix = value.parse().unwrap_or(0),
            "expires_at_unix" => expires_at_unix = value.parse().unwrap_or(0),
            "payload_digest" => payload_digest = value,
            _ => {}
        }
    }

    if request_id.is_empty() || user_subject.is_empty() {
        return put_result(r#"{"valid":false,"reason":"missing_identity"}"#);
    }
    if target_type != "dve_expert_advisor" && target_type != "cli_supervisor" {
        return put_result(r#"{"valid":false,"reason":"invalid_target_type"}"#);
    }
    if target_id.is_empty() {
        return put_result(r#"{"valid":false,"reason":"missing_target_id"}"#);
    }
    if capability.is_empty() {
        return put_result(r#"{"valid":false,"reason":"missing_capability"}"#);
    }

    let seq_key = format!("{}:{}:{}", user_subject, device_id, dve_id);
    if !with_sequence_state(|state| state.check_and_update(&seq_key, sequence)) {
        return put_result(r#"{"valid":false,"reason":"sequence_replay"}"#);
    }

    if payload_digest.len() != 71 || !payload_digest.starts_with("sha256:") {
        return put_result(r#"{"valid":false,"reason":"invalid_digest_format"}"#);
    }

    put_result(r#"{"valid":true}"#)
}

fn __canonicalize(ptr: *const u8, len: usize) -> i32 {
    let json = unsafe {
        let slice = std::slice::from_raw_parts(ptr, len);
        String::from_utf8_lossy(slice).into_owned()
    };
    let canonical = canonical_json(&json);
    put_result(&canonical)
}

fn canonical_json(input: &str) -> String {
    let pairs = json_key_value_pairs(input);
    let mut entries = Vec::new();
    for (k, v) in pairs {
        entries.push(format!("\"{}\":{}", k, v));
    }
    format!("{{{}}}", entries.join(","))
}

fn __compute_digest(_ptr: *const u8, _len: usize) -> i32 {
    put_result(r#"{"delegated":true}"#)
}

fn __check_sequence(ptr: *const u8, len: usize) -> i32 {
    let json = unsafe {
        let slice = std::slice::from_raw_parts(ptr, len);
        String::from_utf8_lossy(slice).into_owned()
    };

    let mut user = String::new();
    let mut device = String::new();
    let mut dve = String::new();
    let mut seq: i32 = 0;

    for (key, value) in json_key_value_pairs(&json) {
        match key.as_str() {
            "user_subject" => user = value,
            "device_id" => device = value,
            "dve_id" => dve = value,
            "sequence" => seq = value.parse().unwrap_or(0),
            _ => {}
        }
    }

    let key = format!("{}:{}:{}", user, device, dve);
    let accepted = with_sequence_state(|state| state.check_and_update(&key, seq));
    put_result(&format!("{{\"accepted\":{}}}", accepted as i32))
}

#[no_mangle]
pub extern "C" fn wasm_init() {
    SEQUENCE_STATE.with(|state| *state.borrow_mut() = SequenceState::default());
    RESULT_POOL.with(|pool| *pool.borrow_mut() = ResultPool::new());
}

fn json_key_value_pairs(json: &str) -> Vec<(String, String)> {
    let mut pairs = Vec::new();
    let mut i = 0;
    while i < json.len() {
        while i < json.len() && json.as_bytes()[i] <= 32 {
            i += 1;
        }
        if i >= json.len() {
            break;
        }
        if json.as_bytes()[i] == b'"' {
            if let Some(key) = json_string(json, &mut i) {
                while i < json.len() && json.as_bytes()[i] != b':' {
                    i += 1;
                }
                i += 1;
                if let Some(value) = json_value(json, &mut i) {
                    pairs.push((key, value));
                }
            } else {
                break;
            }
        } else {
            break;
        }
    }
    pairs
}

fn json_string(s: &str, i: &mut usize) -> Option<String> {
    if *i >= s.len() || s.as_bytes()[*i] != b'"' {
        return None;
    }
    *i += 1;
    let mut out = String::new();
    while *i < s.len() {
        let ch = s.as_bytes()[*i];
        if ch == b'"' {
            *i += 1;
            return Some(out);
        }
        if ch == b'\\' && *i + 1 < s.len() {
            *i += 1;
            out.push(s.chars().nth(*i).unwrap_or('\\'));
        } else {
            out.push(s.chars().nth(*i).unwrap_or('\0'));
        }
        *i += 1;
    }
    Some(out)
}

fn json_value(s: &str, i: &mut usize) -> Option<String> {
    while *i < s.len() && s.as_bytes()[*i] <= 32 {
        *i += 1;
    }
    if *i >= s.len() {
        return Some(String::new());
    }
    let ch = s.as_bytes()[*i];
    if ch == b'"' {
        return json_string(s, i);
    }
    if ch == b'{' || ch == b'[' {
        let close = if ch == b'{' { b'}' } else { b']' };
        let mut depth = 1;
        *i += 1;
        let start = *i;
        while *i < s.len() && depth > 0 {
            let c = s.as_bytes()[*i];
            if c == b'"' {
                if let Some(_) = json_string(s, i) {
                    continue;
                }
            }
            if c == close {
                depth -= 1;
            } else if c == close {
                depth += 1;
            }
            *i += 1;
        }
        return Some(s[start..*i].to_string());
    }
    let start = *i;
    while *i < s.len() {
        let c = s.as_bytes()[*i];
        if c <= 32 || c == b',' || c == b'}' || c == b']' {
            break;
        }
        *i += 1;
    }
    Some(s[start..*i].to_string())
}

fn put_result(bytes: &str) -> i32 {
    with_result_pool(|pool| pool.push(bytes.as_bytes().to_vec()) as i32)
}
