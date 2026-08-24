//! Typed access to KNIRVSERVER's unified actuarial syndicate API.
//! Endpoint paths live here so applications do not each grow a private BFF.
use crate::{client::HttpClient, error::Result};
use serde_json::{json, Value};

#[derive(Clone, Debug)]
pub struct ActuarialService { http: HttpClient }

impl ActuarialService {
    pub fn new(http: HttpClient) -> Self { Self { http } }
    pub async fn risk_classes(&self) -> Result<Value> { self.http.get("/api/v1/actuarial/risk-classes", &[]).await }
    pub async fn pools(&self) -> Result<Value> { self.http.get("/api/v1/actuarial/pools", &[]).await }
    pub async fn pool(&self, id: &str) -> Result<Value> { self.http.get(&format!("/api/v1/actuarial/pools/{id}"), &[]).await }
    pub async fn submission(&self, id: &str) -> Result<Value> { self.http.get(&format!("/api/v1/actuarial/submissions/{id}"), &[]).await }
    pub async fn create_submission(&self, request: Value) -> Result<Value> { self.http.post("/api/v1/actuarial/submissions", request).await }
    pub async fn claim_submission(&self, id: &str, resolver_wallet: &str) -> Result<Value> { self.http.post(&format!("/api/v1/actuarial/submissions/{id}/claim"), json!({"resolver_wallet": resolver_wallet})).await }
    pub async fn create_risk_class(&self, request: Value) -> Result<Value> { self.http.post("/api/v1/actuarial/risk-classes", request).await }
    pub async fn enterprise_credential(&self, request: Value) -> Result<Value> { self.http.post("/api/v1/actuarial/credentials/enterprise", request).await }
    pub async fn exposure(&self) -> Result<Value> { self.http.get("/api/v1/actuarial/me/exposure", &[]).await }
    pub async fn pool_report(&self, id: &str) -> Result<Value> { self.http.get(&format!("/api/v1/actuarial/reports/pools/{id}"), &[]).await }
}
