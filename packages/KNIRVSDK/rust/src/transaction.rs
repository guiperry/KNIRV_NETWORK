use crate::{
    client::HttpClient,
    error::Result,
    types::{
        Block, BlockSubmitParams, CapabilityDescriptorUnion, Chain, HealthStatus, ResourceUri,
        SubmitResponse, Transaction, TransactionSubmitParams, UriGeneratorCreateParams,
        UriResponse,
    },
};
use serde_json::Value;

#[derive(Clone, Debug)]
pub struct TransactionClient {
    http: HttpClient,
}
impl TransactionClient {
    pub fn new(http: HttpClient) -> Self {
        Self { http }
    }
    pub fn http(&self) -> &HttpClient {
        &self.http
    }
    pub async fn chain(&self) -> Result<Chain> {
        self.http.get("/chain", &[]).await
    }
    pub async fn submit_block(&self, block: &Block) -> Result<SubmitResponse> {
        self.http.post("/block", serde_json::to_value(block)?).await
    }
    pub async fn submit_block_typed(&self, params: &BlockSubmitParams) -> Result<SubmitResponse> {
        self.http
            .post("/block", serde_json::to_value(params)?)
            .await
    }
    pub async fn submit_transaction(&self, transaction: &Transaction) -> Result<SubmitResponse> {
        self.http
            .post("/transaction", serde_json::to_value(transaction)?)
            .await
    }
    pub async fn submit_transaction_typed(
        &self,
        params: &TransactionSubmitParams,
    ) -> Result<SubmitResponse> {
        self.http
            .post("/transaction", serde_json::to_value(params)?)
            .await
    }
    pub async fn transaction_pool(&self) -> Result<Vec<Transaction>> {
        self.http.get("/txn_pool", &[]).await
    }
    pub async fn create_uri(&self, resource: &ResourceUri) -> Result<UriResponse> {
        self.http
            .post("/uriGenerator", serde_json::to_value(resource)?)
            .await
    }
    pub async fn create_uri_typed(&self, params: &UriGeneratorCreateParams) -> Result<UriResponse> {
        self.http
            .post("/uriGenerator", serde_json::to_value(params)?)
            .await
    }
    pub async fn health(&self) -> Result<HealthStatus> {
        self.http.get("/health", &[]).await
    }
    pub async fn ping(&self) -> Result<Value> {
        self.http.get("/ping", &[]).await
    }
    pub async fn info(&self) -> Result<Value> {
        self.http.get("/info", &[]).await
    }
    pub async fn peers(&self) -> Result<Value> {
        self.http.get("/peers", &[]).await
    }
    pub async fn mcp_contexts(&self, params: Value) -> Result<Value> {
        self.http
            .get("/mcp/contexts", &[("params", params.to_string())])
            .await
    }
    pub async fn mcp_context_get(&self, ctx_id: &str) -> Result<Value> {
        self.http.get(&format!("/mcp/context/{ctx_id}"), &[]).await
    }
    pub async fn mcp_capabilities(&self, params: Value) -> Result<Value> {
        self.http
            .get("/mcp/capabilities", &[("params", params.to_string())])
            .await
    }
    pub async fn mcp_capability_get(
        &self,
        capability_id: &str,
    ) -> Result<CapabilityDescriptorUnion> {
        self.http
            .get(&format!("/mcp/capability/{capability_id}"), &[])
            .await
    }
    pub async fn mcp_prepare_registration(&self, body: Value) -> Result<Value> {
        self.http
            .post("/mcp/capability/prepare-registration", body)
            .await
    }
    pub async fn mcp_invoke(&self, body: Value) -> Result<Value> {
        self.http.post("/mcp/capability/invoke", body).await
    }
    pub async fn mcp_update(&self, body: Value) -> Result<Value> {
        self.http.put("/mcp/capability", body).await
    }
    pub async fn mcp_invocations(&self, capability_id: &str) -> Result<Value> {
        self.http
            .get(
                "/mcp/capability/invocations",
                &[("capability_id", capability_id.into())],
            )
            .await
    }
}
