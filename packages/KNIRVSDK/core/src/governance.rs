use crate::{client::HttpClient, error::Result};
use serde_json::{json, Value};

#[derive(Clone, Debug)]
pub struct GovernanceClient {
    oracle_http: HttpClient,
    server_http: HttpClient,
}
impl GovernanceClient {
    pub fn new(oracle_url: impl AsRef<str>, server_url: impl AsRef<str>) -> Result<Self> {
        Ok(Self {
            oracle_http: HttpClient::new(oracle_url, crate::client::ClientConfig::default())?,
            server_http: HttpClient::new(server_url, crate::client::ClientConfig::default())?,
        })
    }
    /// Builds a governance client using the unified client's configured HTTP
    /// transports, preserving auth, retries, timeouts, and gateway failover.
    pub fn new_with_http(oracle_http: HttpClient, server_http: HttpClient) -> Self {
        Self {
            oracle_http,
            server_http,
        }
    }
    pub async fn register_did(&self, doc_json: &str) -> Result<Value> {
        self.oracle_http
            .post(
                "/oracle/v3/did/register",
                serde_json::from_str(doc_json).unwrap_or(Value::Null),
            )
            .await
    }
    pub async fn resolve_did(&self, did: &str) -> Result<Value> {
        self.oracle_http
            .get(&format!("/oracle/v3/did/{did}"), &[])
            .await
    }
    pub async fn create_envelope(
        &self,
        node_id: &str,
        agent_id: &str,
        source: &str,
    ) -> Result<Value> {
        self.server_http
            .post(
                "/api/v1/governance/identity/envelopes",
                json!({"node_id": node_id, "agent_id": agent_id, "source": source}),
            )
            .await
    }
    pub async fn revoke_identity(
        &self,
        identity_id: &str,
        node_id: &str,
        reason: &str,
        revoked_by: &str,
    ) -> Result<Value> {
        self.server_http
            .post(
                "/api/v1/governance/identity/revoke",
                json!({"identity_id": identity_id, "node_id": node_id, "reason": reason, "revoked_by": revoked_by}),
            )
            .await
    }
    pub async fn check_revoked(&self, node_id: &str) -> Result<bool> {
        let result: Value = self
            .server_http
            .get(
                &format!("/api/v1/governance/identity/revoked/{node_id}"),
                &[],
            )
            .await?;
        Ok(result
            .get("revoked")
            .and_then(|v| v.as_bool())
            .unwrap_or(false))
    }
    pub async fn normalize_input(
        &self,
        node_id: &str,
        action: &str,
        action_type: &str,
    ) -> Result<Value> {
        self.server_http
            .post(
                "/api/v1/governance/policy/inputs",
                json!({"node_id": node_id, "action": action, "action_type": action_type}),
            )
            .await
    }
    pub async fn get_contract(&self) -> Result<Value> {
        self.server_http
            .get("/api/v1/governance/policy/contract", &[])
            .await
    }
    pub async fn record_breaker_success(&self, breaker_id: &str) -> Result<Value> {
        self.server_http
            .post(
                &format!("/api/v1/governance/reliability/breakers/{breaker_id}/success"),
                json!({}),
            )
            .await
    }
    pub async fn record_breaker_failure(&self, breaker_id: &str) -> Result<Value> {
        self.server_http
            .post(
                &format!("/api/v1/governance/reliability/breakers/{breaker_id}/failure"),
                json!({}),
            )
            .await
    }
    pub async fn is_breaker_allowed(&self, breaker_id: &str) -> Result<bool> {
        let result: Value = self
            .server_http
            .get(
                &format!("/api/v1/governance/reliability/breakers/{breaker_id}/allow"),
                &[],
            )
            .await?;
        Ok(result
            .get("allowed")
            .and_then(|v| v.as_bool())
            .unwrap_or(false))
    }
    pub async fn define_slo(&self, name: &str, target: f64) -> Result<Value> {
        self.server_http
            .post(
                "/api/v1/governance/reliability/slos/define",
                json!({"name": name, "target": target}),
            )
            .await
    }
    pub async fn record_compliance_event(
        &self,
        node_id: &str,
        agent_id: &str,
        event_type: &str,
        severity: &str,
    ) -> Result<Value> {
        self.server_http
            .post(
                "/api/v1/governance/compliance/events",
                json!({"node_id": node_id, "agent_id": agent_id, "event_type": event_type, "severity": severity}),
            )
            .await
    }
    pub async fn get_compliance_status(&self, node_id: &str) -> Result<Value> {
        self.server_http
            .get(
                &format!("/api/v1/governance/compliance/status?node_id={node_id}"),
                &[],
            )
            .await
    }
    pub async fn verify_compliance_chain(&self) -> Result<bool> {
        let result: Value = self
            .server_http
            .get("/api/v1/governance/compliance/chain/verify", &[])
            .await?;
        Ok(result
            .get("valid")
            .and_then(|v| v.as_bool())
            .unwrap_or(false))
    }
}
