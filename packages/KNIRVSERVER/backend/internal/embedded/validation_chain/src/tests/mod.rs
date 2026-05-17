use crate::{ValidationEngine, validation_chain_init, validation_chain_shutdown,
            validation_chain_health_check, validation_chain_validate, validation_chain_free_string};

#[test]
fn test_validation_engine_init_defaults() {
    let engine = ValidationEngine::new(serde_json::Value::Null);
    assert!(engine.is_ok());
}

#[test]
fn test_validation_engine_init_with_config() {
    let config = serde_json::json!({ "chain_id": "testnet", "difficulty": 1 });
    let engine = ValidationEngine::new(config);
    assert!(engine.is_ok());
}

#[test]
fn test_validate_valid_transaction() {
    let engine = ValidationEngine::new(serde_json::Value::Null).unwrap();
    let tx_json =
        r#"{"data": "test data", "signature": "test-sig", "transaction_hash": "0xabc"}"#;
    let (code, response) = engine.validate(tx_json);
    assert_eq!(code, 0, "expected valid transaction, got error: {}", response);
    let parsed: serde_json::Value = serde_json::from_str(&response).unwrap();
    assert_eq!(parsed["valid"], true);
    assert_eq!(parsed["transaction_hash"], "0xabc");
}

#[test]
fn test_validate_empty_data() {
    let engine = ValidationEngine::new(serde_json::Value::Null).unwrap();
    let tx_json = r#"{"data": "", "signature": "", "transaction_hash": null}"#;
    let (code, response) = engine.validate(tx_json);
    assert_ne!(code, 0, "expected invalid transaction");
    let parsed: serde_json::Value = serde_json::from_str(&response).unwrap();
    assert_eq!(parsed["valid"], false);
}

#[test]
fn test_validate_malformed_json() {
    let engine = ValidationEngine::new(serde_json::Value::Null).unwrap();
    let (code, response) = engine.validate("not valid json");
    assert_ne!(code, 0, "expected invalid for malformed JSON");
    let parsed: serde_json::Value = serde_json::from_str(&response).unwrap();
    assert_eq!(parsed["valid"], false);
    assert!(parsed["error"]
        .as_str()
        .unwrap_or("")
        .contains("parse error"));
}

#[test]
fn test_cffi_validation_chain_init() {
    let result = unsafe { validation_chain_init(std::ptr::null(), 0) };
    assert_eq!(result, 0, "init with null config should succeed");

    let health = unsafe { validation_chain_health_check() };
    assert_eq!(health, 0, "health check should pass after init");

    let shutdown = unsafe { validation_chain_shutdown() };
    assert_eq!(shutdown, 0, "shutdown should succeed");
}

#[test]
fn test_cffi_health_before_init() {
    let _ = unsafe { validation_chain_shutdown() };
    let health = unsafe { validation_chain_health_check() };
    assert_eq!(health, -1, "health should fail before init");
}

#[test]
fn test_cffi_validation_init_with_config() {
    let config = r#"{"chain_id": "test"}"#;
    let c_config = std::ffi::CString::new(config).unwrap();
    let result = unsafe { validation_chain_init(c_config.as_ptr(), config.len()) };
    assert_eq!(result, 0);

    let tx = r#"{"data": "test", "signature": "sig1", "transaction_hash": "0x123"}"#;
    let c_tx = std::ffi::CString::new(tx).unwrap();
    let mut result_code: std::os::raw::c_int = -1;
    let mut result_len: usize = 0;

    let ptr = unsafe {
        validation_chain_validate(
            c_tx.as_ptr(),
            tx.len(),
            &mut result_code as *mut std::os::raw::c_int,
            &mut result_len as *mut usize,
        )
    };
    assert!(!ptr.is_null(), "validate should return non-null");
    assert_eq!(result_code, 0, "valid transaction should return code 0");

    unsafe { validation_chain_free_string(ptr) };
    unsafe { validation_chain_shutdown() };
}

#[test]
fn test_cffi_validate_before_init() {
    let _ = unsafe { validation_chain_shutdown() };

    let tx = r#"{"data": "test", "signature": "sig1"}"#;
    let c_tx = std::ffi::CString::new(tx).unwrap();
    let mut result_code: std::os::raw::c_int = 0;
    let mut result_len: usize = 0;

    let ptr = unsafe {
        validation_chain_validate(
            c_tx.as_ptr(),
            tx.len(),
            &mut result_code as *mut std::os::raw::c_int,
            &mut result_len as *mut usize,
        )
    };
    assert!(
        !ptr.is_null(),
        "should still return a response even without init"
    );
    assert_eq!(result_code, -1, "uninitialized engine should return -1");

    let response = unsafe { std::ffi::CStr::from_ptr(ptr) }
        .to_str()
        .unwrap_or("");
    assert!(response.contains("engine not initialised"));

    unsafe { validation_chain_free_string(ptr) };
}
