use crate::{client::HttpClient, error::Result, types::OracleStatus};
use serde_json::Value;

#[derive(Clone, Debug)]
pub struct OracleClient {
    http: HttpClient,
}
impl OracleClient {
    pub fn new(base_url: impl AsRef<str>) -> Result<Self> {
        Ok(Self {
            http: HttpClient::new(base_url, crate::client::ClientConfig::default())?,
        })
    }
    pub fn new_with_http(http: HttpClient) -> Self {
        Self { http }
    }
    pub async fn get_status(&self) -> Result<OracleStatus> {
        self.http.get("/status", &[]).await
    }
    pub async fn get_status_raw(&self) -> Result<Value> {
        self.http.get("/status", &[]).await
    }
}
