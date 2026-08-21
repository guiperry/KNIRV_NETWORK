use thiserror::Error;

pub type Result<T> = std::result::Result<T, Error>;

#[derive(Debug, Error)]
pub enum Error {
    #[error("invalid configuration: {0}")]
    Configuration(String),
    #[error("validation error: {0}")]
    Validation(String),
    #[error("request timed out")]
    Timeout,
    #[error("transport error: {0}")]
    Transport(#[from] reqwest::Error),
    #[error("HTTP {status}: {message}")]
    Api { status: u16, message: String },
    #[error("response decoding error: {0}")]
    Decode(#[from] serde_json::Error),
    #[error("cryptographic error: {0}")]
    Crypto(String),
    #[error("idempotency key conflict")]
    IdempotencyConflict,
    #[error("unsupported network: {0}")]
    UnsupportedNetwork(String),
}
