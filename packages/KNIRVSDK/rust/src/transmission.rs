use crate::{
    client::HttpClient,
    error::{Error, Result},
};
use serde_json::Value;

#[derive(Clone, Debug)]
pub struct TransmissionClient {
    http: HttpClient,
}

impl TransmissionClient {
    pub fn new(http: HttpClient) -> Self {
        Self { http }
    }

    pub async fn resolve_uri(&self, uri: &str) -> Result<Value> {
        if uri.trim().is_empty() {
            return Err(Error::Validation("URI is required".into()));
        }
        self.http
            .get("/api/transmission/resolve", &[("uri", uri.into())])
            .await
    }

    pub async fn broadcast(&self, data: Value) -> Result<Value> {
        self.http.post("/api/transmission/broadcast", data).await
    }
}
