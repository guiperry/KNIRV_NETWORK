//! C FFI exports for the graphrag-core static library.
//!
//! These symbols are called from Go via CGo. All functions are re-entrant-safe
//! through a process-global `Mutex<Option<GraphRAG>>`.
//!
//! Memory contract for callers:
//! - Input `*const c_char` pointers must be valid NUL-terminated UTF-8.
//! - Returned `*mut c_char` pointers are heap-allocated with `CString::into_raw`
//!   and MUST be freed by calling `graphrag_free_string`.  Never free them with
//!   the C `free()` — the allocator may differ across the boundary.

use std::ffi::{CStr, CString};
use std::os::raw::{c_char, c_int};
use std::sync::Mutex;

use crate::vector::EmbeddingGenerator;
use crate::{Config, GraphRAG};

// ---------------------------------------------------------------------------
// Global instance
// ---------------------------------------------------------------------------

static INSTANCE: Mutex<Option<GraphRAG>> = Mutex::new(None);

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

fn lock_instance<F, T>(f: F) -> Option<T>
where
    F: FnOnce(&mut GraphRAG) -> T,
{
    match INSTANCE.lock() {
        Ok(mut guard) => guard.as_mut().map(f),
        Err(_) => None,
    }
}

unsafe fn cstr_to_str<'a>(ptr: *const c_char) -> Option<&'a str> {
    if ptr.is_null() {
        return None;
    }
    CStr::from_ptr(ptr).to_str().ok()
}

fn to_raw_cstring(s: String) -> *mut c_char {
    match CString::new(s) {
        Ok(cs) => cs.into_raw(),
        Err(_) => std::ptr::null_mut(),
    }
}

// ---------------------------------------------------------------------------
// Public C API
// ---------------------------------------------------------------------------

/// Initialise the GraphRAG engine.
///
/// `config` is a NUL-terminated JSON/TOML configuration string (may be empty
/// `""` to use defaults).  Returns 0 on success, -1 on error.
#[no_mangle]
pub extern "C" fn graphrag_init(config: *const c_char, _config_len: usize) -> c_int {
    // Parse optional JSON config; fall back to defaults on any parse error.
    let cfg: Config = if config.is_null() {
        Config::default()
    } else {
        match unsafe { cstr_to_str(config) } {
            Some(s) if !s.trim().is_empty() => {
                serde_json::from_str(s).unwrap_or_default()
            }
            _ => Config::default(),
        }
    };

    let mut instance = match GraphRAG::new(cfg) {
        Ok(g) => g,
        Err(_) => return -1,
    };

    if instance.initialize().is_err() {
        return -1;
    }

    match INSTANCE.lock() {
        Ok(mut guard) => {
            *guard = Some(instance);
            0
        }
        Err(_) => -1,
    }
}

/// Shut down the GraphRAG engine and free all resources.
///
/// Returns 0 on success, -1 if the engine was not initialised.
#[no_mangle]
pub extern "C" fn graphrag_shutdown() -> c_int {
    match INSTANCE.lock() {
        Ok(mut guard) => {
            *guard = None;
            0
        }
        Err(_) => -1,
    }
}

/// Return 0 if the engine is healthy and initialised, -1 otherwise.
#[no_mangle]
pub extern "C" fn graphrag_health_check() -> c_int {
    match INSTANCE.lock() {
        Ok(guard) => {
            if guard.is_some() { 0 } else { -1 }
        }
        Err(_) => -1,
    }
}

/// Index a document identified by `doc_id` with the given `content`.
///
/// `doc_id` and `content` must be NUL-terminated UTF-8 strings.
/// Returns 0 on success, -1 on error.
#[no_mangle]
pub extern "C" fn graphrag_index_document(
    doc_id: *const c_char,
    content: *const c_char,
    _content_len: usize,
) -> c_int {
    let _id = match unsafe { cstr_to_str(doc_id) } {
        Some(s) => s.to_owned(),
        None => return -1,
    };
    let text = match unsafe { cstr_to_str(content) } {
        Some(s) => s.to_owned(),
        None => return -1,
    };

    match lock_instance(|g| g.add_document_from_text(&text)) {
        Some(Ok(())) => 0,
        _ => -1,
    }
}

/// Execute a knowledge-graph query and return the answer as a
/// NUL-terminated JSON string.
///
/// The caller **must** free the returned pointer with `graphrag_free_string`.
/// Returns NULL on error.
///
/// `result_len` is set to the byte length of the returned string (excluding
/// the NUL terminator) so the caller can avoid a second `strlen` call.
#[no_mangle]
pub extern "C" fn graphrag_query(
    query: *const c_char,
    _limit: c_int,
    result_len: *mut usize,
) -> *mut c_char {
    let q = match unsafe { cstr_to_str(query) } {
        Some(s) => s.to_owned(),
        None => return std::ptr::null_mut(),
    };

    #[cfg(feature = "async")]
    let answer = {
        let rt = match tokio::runtime::Runtime::new() {
            Ok(r) => r,
            Err(_) => return std::ptr::null_mut(),
        };
        match lock_instance(|g| rt.block_on(g.ask(&q))) {
            Some(Ok(a)) => a,
            _ => return std::ptr::null_mut(),
        }
    };

    #[cfg(not(feature = "async"))]
    let answer = match lock_instance(|g| g.ask(&q)) {
        Some(Ok(a)) => a,
        _ => return std::ptr::null_mut(),
    };

    // Serialise to a JSON object so the caller has a stable structure.
    let payload = match serde_json::to_string(&serde_json::json!({ "answer": answer })) {
        Ok(s) => s,
        Err(_) => return std::ptr::null_mut(),
    };

    if !result_len.is_null() {
        unsafe { *result_len = payload.len() };
    }

    to_raw_cstring(payload)
}

/// Generate embeddings for a JSON array of text strings.
///
/// `texts_json` is a NUL-terminated JSON array of strings, e.g. `["hello","world"]`.
/// Returns a NUL-terminated JSON array of float arrays, e.g.
/// `[[0.1,0.2,...],[0.3,0.4,...]]`.  The caller **must** free the returned
/// pointer with `graphrag_free_string`.  Returns NULL on error.
///
/// `result_len` is set to the byte length of the returned string (excluding
/// the NUL terminator).
#[no_mangle]
pub extern "C" fn graphrag_embed_texts(
    texts_json: *const c_char,
    _texts_len: usize,
    result_len: *mut usize,
) -> *mut c_char {
    let json_str = match unsafe { cstr_to_str(texts_json) } {
        Some(s) => s,
        None => return std::ptr::null_mut(),
    };

    let texts: Vec<String> = match serde_json::from_str(json_str) {
        Ok(t) => t,
        Err(_) => return std::ptr::null_mut(),
    };

    if texts.is_empty() {
        return std::ptr::null_mut();
    }

    let str_refs: Vec<&str> = texts.iter().map(|s| s.as_str()).collect();
    let mut generator = EmbeddingGenerator::new(128);
    let embeddings = generator.batch_generate(&str_refs);

    let payload = match serde_json::to_string(&embeddings) {
        Ok(s) => s,
        Err(_) => return std::ptr::null_mut(),
    };

    if !result_len.is_null() {
        unsafe { *result_len = payload.len() };
    }

    to_raw_cstring(payload)
}

/// Return the entities and relationships extracted by the last index operation.
///
/// Returns a NUL-terminated JSON object with the structure:
/// ```json
/// {
///   "entities": [{"id":"...","name":"...","entity_type":"...","confidence":0.0}],
///   "relationships": [{"source":"...","target":"...","relation_type":"...","confidence":0.0}],
///   "entity_count": 0,
///   "relationship_count": 0
/// }
/// ```
/// The caller **must** free the returned pointer with `graphrag_free_string`.
/// Returns NULL if no graph has been built yet, or on error.
///
/// `result_len` is set to the byte length of the returned string (excluding
/// the NUL terminator).
#[no_mangle]
pub extern "C" fn graphrag_get_last_extraction(result_len: *mut usize) -> *mut c_char {
    let payload = match lock_instance(|g| {
        let kg = match g.knowledge_graph() {
            Some(kg) => kg,
            None => return serde_json::json!({ "entities": [], "relationships": [], "entity_count": 0, "relationship_count": 0 }),
        };

        let entities: Vec<serde_json::Value> = kg
            .entities()
            .map(|e| {
                serde_json::json!({
                    "id": e.id.to_string(),
                    "name": e.name,
                    "entity_type": e.entity_type,
                    "confidence": e.confidence,
                })
            })
            .collect();

        let relationships: Vec<serde_json::Value> = kg
            .relationships()
            .map(|r| {
                serde_json::json!({
                    "source": r.source.to_string(),
                    "target": r.target.to_string(),
                    "relation_type": r.relation_type,
                    "confidence": r.confidence,
                })
            })
            .collect();

        serde_json::json!({
            "entities": entities,
            "relationships": relationships,
            "entity_count": kg.entity_count(),
            "relationship_count": kg.relationship_count(),
        })
    }) {
        Some(v) => v,
        None => return std::ptr::null_mut(),
    };

    let payload_str = match serde_json::to_string(&payload) {
        Ok(s) => s,
        Err(_) => return std::ptr::null_mut(),
    };

    if !result_len.is_null() {
        unsafe { *result_len = payload_str.len() };
    }

    to_raw_cstring(payload_str)
}

/// Free a string previously returned by `graphrag_query`.
///
/// Passing NULL is a no-op.
#[no_mangle]
pub extern "C" fn graphrag_free_string(ptr: *mut c_char) {
    if !ptr.is_null() {
        unsafe { drop(CString::from_raw(ptr)) };
    }
}
