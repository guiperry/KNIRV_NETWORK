// Generated ProtoBuf types for KNIRV-CORTEX
pub mod cortex {
    include!(concat!(env!("OUT_DIR"), "/knirv.cortex.v1.rs"));
}

// Re-export commonly used types
pub use cortex::*;

// Utility functions for ABI boundary
pub mod abi {
    /// Pack pointer and length into a single u64
    /// High 32 bits = pointer, Low 32 bits = length
    pub fn pack_ptr_len(ptr: *const u8, len: usize) -> u64 {
        let ptr_val = ptr as u64;
        let len_val = len as u64;
        (ptr_val << 32) | len_val
    }
    
    /// Unpack pointer and length from a u64
    /// Returns (pointer, length)
    pub fn unpack_ptr_len(packed: u64) -> (*const u8, usize) {
        let ptr = (packed >> 32) as *const u8;
        let len = (packed & 0xFFFFFFFF) as usize;
        (ptr, len)
    }
    
    /// Pack mutable pointer and length into a single u64
    pub fn pack_mut_ptr_len(ptr: *mut u8, len: usize) -> u64 {
        let ptr_val = ptr as u64;
        let len_val = len as u64;
        (ptr_val << 32) | len_val
    }
    
    /// Unpack mutable pointer and length from a u64
    pub fn unpack_mut_ptr_len(packed: u64) -> (*mut u8, usize) {
        let ptr = (packed >> 32) as *mut u8;
        let len = (packed & 0xFFFFFFFF) as usize;
        (ptr, len)
    }
}

// Envelope helpers
impl Envelope {
    pub fn ok(payload: Vec<u8>) -> Self {
        Self {
            kind: EnvelopeKind::Ok as i32,
            payload,
            trace_id: None,
            timestamp: std::time::SystemTime::now()
                .duration_since(std::time::UNIX_EPOCH)
                .unwrap_or_default()
                .as_millis() as u64,
        }
    }

    pub fn error(error: CortexError) -> Self {
        let payload = error.encode_to_vec();
        Self {
            kind: EnvelopeKind::Error as i32,
            payload,
            trace_id: None,
            timestamp: std::time::SystemTime::now()
                .duration_since(std::time::UNIX_EPOCH)
                .unwrap_or_default()
                .as_millis() as u64,
        }
    }
    
    pub fn with_trace_id(mut self, trace_id: String) -> Self {
        self.trace_id = Some(trace_id);
        self
    }
}

// Error helpers
impl CortexError {
    pub fn new(code: u32, message: impl Into<String>) -> Self {
        Self {
            code,
            message: message.into(),
            details: None,
        }
    }
    
    pub fn with_details(mut self, details: impl Into<String>) -> Self {
        self.details = Some(details.into());
        self
    }
}

// Common error codes
pub mod error_codes {
    pub const INVALID_INPUT: u32 = 1000;
    pub const PROCESSING_FAILED: u32 = 1001;
    pub const MEMORY_LIMIT_EXCEEDED: u32 = 1002;
    pub const TIMEOUT: u32 = 1003;
    pub const MODEL_NOT_LOADED: u32 = 1004;
    pub const UNSUPPORTED_OPERATION: u32 = 1005;
    pub const RUNTIME_ERROR: u32 = 1006;
}

// Prost trait implementations
use prost::Message;

impl InferenceInput {
    pub fn encode_to_vec(&self) -> Vec<u8> {
        let mut buf = Vec::new();
        self.encode(&mut buf).unwrap();
        buf
    }
}

impl InferenceOutput {
    pub fn encode_to_vec(&self) -> Vec<u8> {
        let mut buf = Vec::new();
        self.encode(&mut buf).unwrap();
        buf
    }
}

impl CortexError {
    pub fn encode_to_vec(&self) -> Vec<u8> {
        let mut buf = Vec::new();
        self.encode(&mut buf).unwrap();
        buf
    }
}

impl Envelope {
    pub fn encode_to_vec(&self) -> Vec<u8> {
        let mut buf = Vec::new();
        self.encode(&mut buf).unwrap();
        buf
    }
}
