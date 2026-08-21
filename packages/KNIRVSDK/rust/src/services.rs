use crate::{
    client::HttpClient,
    error::{Error, Result},
    types::*,
};
use serde_json::{json, Value};

fn required(value: &str, field: &str) -> Result<()> {
    if value.trim().is_empty() {
        Err(Error::Validation(format!("{field} is required")))
    } else {
        Ok(())
    }
}

#[derive(Clone, Debug)]
pub struct BadgeService {
    http: HttpClient,
}
impl BadgeService {
    pub fn new(http: HttpClient) -> Self {
        Self { http }
    }
    pub async fn get_agent_badges(&self, agent_id: &str) -> Result<Vec<Badge>> {
        required(agent_id, "agent ID")?;
        self.http
            .get(&format!("/api/agents/{agent_id}/badges"), &[])
            .await
    }
    pub async fn get_skill_badges(&self, agent_id: &str) -> Result<Vec<SkillBadge>> {
        required(agent_id, "agent ID")?;
        self.http
            .get(&format!("/api/agents/{agent_id}/badges/skills"), &[])
            .await
    }
    pub async fn get_capability_badges(&self, agent_id: &str) -> Result<Vec<CapabilityBadge>> {
        required(agent_id, "agent ID")?;
        self.http
            .get(&format!("/api/agents/{agent_id}/badges/capabilities"), &[])
            .await
    }
    pub async fn get_property_badges(&self, agent_id: &str) -> Result<Vec<PropertyBadge>> {
        required(agent_id, "agent ID")?;
        self.http
            .get(&format!("/api/agents/{agent_id}/badges/properties"), &[])
            .await
    }
    pub async fn validate_badge(&self, badge_id: &str) -> Result<Value> {
        required(badge_id, "badge ID")?;
        self.http
            .post(&format!("/api/badges/{badge_id}/validate"), json!({}))
            .await
    }
    pub async fn issue_badge(&self, badge: &Badge) -> Result<Badge> {
        self.http
            .post("/api/badges", serde_json::to_value(badge)?)
            .await
    }
}

#[derive(Clone, Debug)]
pub struct DVEService {
    http: HttpClient,
}
impl DVEService {
    pub fn new(http: HttpClient) -> Self {
        Self { http }
    }
    pub async fn list_environments(&self, user_id: Option<&str>) -> Result<Vec<DVEEnvironment>> {
        let query: Vec<(&str, String)> =
            user_id.into_iter().map(|v| ("userId", v.into())).collect();
        self.http.get("/api/dve/environments", &query).await
    }
    pub async fn create_environment(&self, config: &DVEEnvironment) -> Result<DVEEnvironment> {
        self.http
            .post("/api/dve/environments", serde_json::to_value(config)?)
            .await
    }
    pub async fn get_environment(&self, environment_id: &str) -> Result<DVEEnvironment> {
        required(environment_id, "environment ID")?;
        self.http
            .get(&format!("/api/dve/environments/{environment_id}"), &[])
            .await
    }
    pub async fn delete_environment(&self, environment_id: &str) -> Result<Value> {
        required(environment_id, "environment ID")?;
        self.http
            .delete(&format!("/api/dve/environments/{environment_id}"))
            .await
    }
    pub async fn start_session(&self, environment_id: &str) -> Result<DVESession> {
        required(environment_id, "environment ID")?;
        self.http
            .post(
                &format!("/api/dve/environments/{environment_id}/sessions"),
                json!({}),
            )
            .await
    }
    pub async fn get_session(&self, session_id: &str) -> Result<DVESession> {
        required(session_id, "session ID")?;
        self.http
            .get(&format!("/api/dve/sessions/{session_id}"), &[])
            .await
    }
    pub async fn terminate_session(&self, session_id: &str) -> Result<Value> {
        required(session_id, "session ID")?;
        self.http
            .delete(&format!("/api/dve/sessions/{session_id}"))
            .await
    }
}

#[derive(Clone, Debug)]
pub struct TreasuryService {
    http: HttpClient,
}
impl TreasuryService {
    pub fn new(http: HttpClient) -> Self {
        Self { http }
    }
    pub async fn get_nrn_token_info(&self) -> Result<NRNToken> {
        self.http.get("/api/treasury/nrn", &[]).await
    }
    pub async fn get_treasury_balance(&self) -> Result<Value> {
        self.http.get("/api/treasury/balance", &[]).await
    }
    pub async fn get_treasury_operations(
        &self,
        limit: Option<u32>,
    ) -> Result<Vec<TreasuryOperation>> {
        let query: Vec<(&str, String)> = limit
            .into_iter()
            .map(|v| ("limit", v.to_string()))
            .collect();
        self.http.get("/api/treasury/operations", &query).await
    }
    pub async fn request_faucet(&self, user_address: &str, amount: &str) -> Result<FaucetRequest> {
        required(user_address, "user address")?;
        required(amount, "amount")?;
        self.http
            .post(
                "/api/treasury/faucet",
                json!({"userAddress": user_address, "amount": amount}),
            )
            .await
    }
    pub async fn get_faucet_request(&self, request_id: &str) -> Result<FaucetRequest> {
        required(request_id, "faucet request ID")?;
        self.http
            .get(&format!("/api/treasury/faucet/{request_id}"), &[])
            .await
    }
    pub async fn mint_nrn(&self, amount: &str, reason: &str) -> Result<TreasuryOperation> {
        required(amount, "amount")?;
        required(reason, "reason")?;
        self.http
            .post(
                "/api/treasury/mint",
                json!({"amount": amount, "reason": reason}),
            )
            .await
    }
}

#[derive(Clone, Debug)]
pub struct AgentService {
    http: HttpClient,
}
impl AgentService {
    pub fn new(http: HttpClient) -> Self {
        Self { http }
    }
    pub async fn list_agents(&self, user_id: Option<&str>) -> Result<Vec<Agent>> {
        let query: Vec<(&str, String)> =
            user_id.into_iter().map(|v| ("userId", v.into())).collect();
        self.http.get("/api/agents", &query).await
    }
    pub async fn get_agent(&self, agent_id: &str) -> Result<Agent> {
        required(agent_id, "agent ID")?;
        self.http.get(&format!("/api/agents/{agent_id}"), &[]).await
    }
    pub async fn create_agent(&self, agent: &Agent) -> Result<Agent> {
        self.http
            .post("/api/agents", serde_json::to_value(agent)?)
            .await
    }
    pub async fn update_agent(&self, agent_id: &str, updates: &Agent) -> Result<Agent> {
        required(agent_id, "agent ID")?;
        self.http
            .put(
                &format!("/api/agents/{agent_id}"),
                serde_json::to_value(updates)?,
            )
            .await
    }
    pub async fn delete_agent(&self, agent_id: &str) -> Result<Value> {
        required(agent_id, "agent ID")?;
        self.http.delete(&format!("/api/agents/{agent_id}")).await
    }
    pub async fn get_agent_workflows(&self, agent_id: &str) -> Result<Vec<AgentWorkflow>> {
        required(agent_id, "agent ID")?;
        self.http
            .get(&format!("/api/agents/{agent_id}/workflows"), &[])
            .await
    }
    pub async fn create_workflow(
        &self,
        agent_id: &str,
        workflow: &AgentWorkflow,
    ) -> Result<AgentWorkflow> {
        required(agent_id, "agent ID")?;
        self.http
            .post(
                &format!("/api/agents/{agent_id}/workflows"),
                serde_json::to_value(workflow)?,
            )
            .await
    }
    pub async fn invoke_skill(&self, request: &SkillInvokeRequest) -> Result<SkillInvocation> {
        self.http
            .post("/api/skills/invoke", serde_json::to_value(request)?)
            .await
    }
}

#[derive(Clone, Debug)]
pub struct NetworkService {
    http: HttpClient,
}
impl NetworkService {
    pub fn new(http: HttpClient) -> Self {
        Self { http }
    }
    pub async fn submit_connectivity_proof(
        &self,
        proof: &ConnectivityProof,
    ) -> Result<ConnectivityProof> {
        self.http
            .post("/api/network/proofs", serde_json::to_value(proof)?)
            .await
    }
    pub async fn get_connectivity_proofs(
        &self,
        agent_id: Option<&str>,
    ) -> Result<Vec<ConnectivityProof>> {
        let query: Vec<(&str, String)> = agent_id
            .into_iter()
            .map(|v| ("agentId", v.into()))
            .collect();
        self.http.get("/api/network/proofs", &query).await
    }
    pub async fn get_network_routes(&self) -> Result<Vec<NetworkRoute>> {
        self.http.get("/api/network/routes", &[]).await
    }
    pub async fn find_optimal_route(
        &self,
        source: &str,
        destination: &str,
    ) -> Result<NetworkRoute> {
        required(source, "source")?;
        required(destination, "destination")?;
        let query = vec![
            ("source", source.into()),
            ("destination", destination.into()),
        ];
        self.http.get("/api/network/routes/optimal", &query).await
    }
    pub async fn get_network_stats(&self) -> Result<Value> {
        self.http.get("/api/network/stats", &[]).await
    }
}

#[derive(Clone, Debug)]
pub struct FactualityService {
    http: HttpClient,
}
impl FactualityService {
    pub fn new(http: HttpClient) -> Self {
        Self { http }
    }
    pub async fn get_slice_info(&self) -> Result<FactualitySlice> {
        self.http.get("/api/factuality/slice", &[]).await
    }
    pub async fn verify_content(
        &self,
        content: &str,
        domain: &str,
    ) -> Result<FactualityVerification> {
        required(content, "content")?;
        required(domain, "domain")?;
        self.http
            .post(
                "/api/factuality/verify",
                json!({"content": content, "domain": domain}),
            )
            .await
    }
    pub async fn get_verification_history(
        &self,
        limit: Option<u32>,
    ) -> Result<Vec<FactualityVerification>> {
        let query: Vec<(&str, String)> = limit
            .into_iter()
            .map(|v| ("limit", v.to_string()))
            .collect();
        self.http.get("/api/factuality/history", &query).await
    }
    pub async fn get_verification(&self, verification_id: &str) -> Result<FactualityVerification> {
        required(verification_id, "verification ID")?;
        self.http
            .get(
                &format!("/api/factuality/verifications/{verification_id}"),
                &[],
            )
            .await
    }
}

#[derive(Clone, Debug)]
pub struct HealthService {
    http: HttpClient,
}
impl HealthService {
    pub fn new(http: HttpClient) -> Self {
        Self { http }
    }
    pub async fn get_network_health(&self) -> Result<NetworkHealth> {
        self.http.get("/api/health/network", &[]).await
    }
    pub async fn get_service_health(&self, service_name: &str) -> Result<ServiceHealth> {
        required(service_name, "service name")?;
        self.http
            .get(&format!("/api/health/services/{service_name}"), &[])
            .await
    }
    pub async fn get_all_services_health(&self) -> Result<Value> {
        self.http.get("/api/health/services", &[]).await
    }
    pub async fn ping_service(&self, service_name: &str) -> Result<Value> {
        required(service_name, "service name")?;
        self.http
            .post(&format!("/api/health/ping/{service_name}"), json!({}))
            .await
    }
}

#[derive(Clone, Debug)]
pub struct ConfigService {
    http: HttpClient,
}
impl ConfigService {
    pub fn new(http: HttpClient) -> Self {
        Self { http }
    }
    pub async fn get_network_info(&self) -> Result<KNIRVNetworkInfo> {
        self.http.get("/api/config/network", &[]).await
    }
    pub async fn switch_network(&self, environment: &str) -> Result<KNIRVNetworkInfo> {
        required(environment, "environment")?;
        self.http
            .post(
                "/api/config/network/switch",
                json!({"environment": environment}),
            )
            .await
    }
    pub async fn list_networks(&self) -> Result<Vec<KNIRVNetworkInfo>> {
        self.http.get("/api/config/networks", &[]).await
    }
}
