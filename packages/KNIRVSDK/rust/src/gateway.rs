use crate::{
    client::HttpClient,
    error::{Error, Result},
    types::{
        BurnEvent, EconomicOverview, EconomicRules, FeeCalculationRequest, FeeStructure,
        GatewayStatus, GatewayTransaction, HealthStatus, IntegrationStatus, LLMRegistrationRequest,
        LLMRegistrationResponse, ListResponse, NetworkAuthorsResponse, NetworkFeesResponse,
        PoAuDChallenge, PoAuDProof, PoAuDResponse, PoAuDStatus, PoAuDSubmissionResult,
        PoAuDUserReputation, Route, ServiceEconomics, Skill, SkillInput, SkillMetrics,
        ValidationRewardRequest, ValidationRewardResponse,
    },
};
use serde_json::{json, Value};

/// KNIRVGATEWAY client, covering Economics, PoAuD, routes, health, integrations,
/// LLM registration, fees, metrics, transactions, burn, and rules.
#[derive(Clone, Debug)]
pub struct GatewayClient {
    http: HttpClient,
}
impl GatewayClient {
    pub fn new(http: HttpClient) -> Self {
        Self { http }
    }
    pub async fn list_skills(&self, query: &[(&str, String)]) -> Result<ListResponse<Skill>> {
        self.http.get("/economics/skills", query).await
    }
    pub async fn skill(&self, id: &str) -> Result<Skill> {
        required(id, "skill ID")?;
        self.http.get(&format!("/economics/skills/{id}"), &[]).await
    }
    pub async fn create_skill(&self, input: &SkillInput) -> Result<Skill> {
        validate_skill(input)?;
        self.http
            .post("/economics/skills", serde_json::to_value(input)?)
            .await
    }
    pub async fn update_skill(&self, id: &str, input: &SkillInput) -> Result<Skill> {
        required(id, "skill ID")?;
        validate_skill(input)?;
        self.http
            .put(
                &format!("/economics/skills/{id}"),
                serde_json::to_value(input)?,
            )
            .await
    }
    pub async fn delete_skill(&self, id: &str) -> Result<Value> {
        required(id, "skill ID")?;
        self.http.delete(&format!("/economics/skills/{id}")).await
    }
    pub async fn search_skills(&self, query: &[(&str, String)]) -> Result<ListResponse<Skill>> {
        if query.is_empty() {
            return Err(Error::Validation("search query is required".into()));
        }
        self.http.get("/economics/skills/search", query).await
    }
    pub async fn llm_models(&self) -> Result<Value> {
        self.http.get("/economics/llm/models", &[]).await
    }
    pub async fn llm_usage(&self, period: Option<String>) -> Result<Value> {
        let q = period.map(|v| vec![("period", v)]).unwrap_or_default();
        self.http.get("/economics/llm/usage", &q).await
    }
    pub async fn estimate_llm_cost(&self, text: &str, model: &str) -> Result<Value> {
        required(text, "text")?;
        required(model, "model")?;
        self.http
            .post(
                "/economics/llm/estimate",
                json!({"text": text, "model": model}),
            )
            .await
    }
    pub async fn llm_register(
        &self,
        req: &LLMRegistrationRequest,
    ) -> Result<LLMRegistrationResponse> {
        self.http
            .post("/economics/llm/register", serde_json::to_value(req)?)
            .await
    }
    pub async fn validate(&self, skill_id: &str, data: Value) -> Result<Value> {
        required(skill_id, "skill ID")?;
        self.http
            .post(
                "/economics/validation/validate",
                json!({"skill_id":skill_id,"data":data}),
            )
            .await
    }
    pub async fn validation_rules(&self) -> Result<Value> {
        self.http.get("/economics/validation/rules", &[]).await
    }
    pub async fn validation_reward(
        &self,
        req: &ValidationRewardRequest,
    ) -> Result<ValidationRewardResponse> {
        self.http
            .post("/economics/validation/reward", serde_json::to_value(req)?)
            .await
    }
    pub async fn calculate_fees(&self, req: &FeeCalculationRequest) -> Result<NetworkFeesResponse> {
        self.http
            .post("/economics/fees/calculate", serde_json::to_value(req)?)
            .await
    }
    pub async fn get_fee_structure(&self) -> Result<FeeStructure> {
        self.http.get("/economics/fees/structure", &[]).await
    }
    pub async fn get_metrics_overview(&self) -> Result<EconomicOverview> {
        self.http.get("/economics/metrics", &[]).await
    }
    pub async fn get_skill_metrics(&self) -> Result<Vec<SkillMetrics>> {
        self.http.get("/economics/metrics/skills", &[]).await
    }
    pub async fn get_service_metrics(&self, service_name: &str) -> Result<ServiceEconomics> {
        required(service_name, "service name")?;
        self.http
            .get(&format!("/economics/service/{service_name}/metrics"), &[])
            .await
    }
    pub async fn get_transaction(&self, transaction_id: &str) -> Result<GatewayTransaction> {
        required(transaction_id, "transaction ID")?;
        self.http
            .get(&format!("/economics/transaction/{transaction_id}"), &[])
            .await
    }
    pub async fn list_transactions(
        &self,
        limit: Option<u32>,
        status: Option<&str>,
    ) -> Result<Vec<GatewayTransaction>> {
        let mut query = Vec::new();
        if let Some(limit) = limit {
            query.push(("limit", limit.to_string()));
        }
        if let Some(status) = status {
            query.push(("status", status.into()));
        }
        self.http.get("/economics/transactions", &query).await
    }
    pub async fn get_burn_history(&self, limit: Option<u32>) -> Result<Vec<BurnEvent>> {
        let q = limit
            .map(|v| vec![("limit", v.to_string())])
            .unwrap_or_default();
        self.http.get("/economics/burn/history", &q).await
    }
    pub async fn get_total_burned(&self) -> Result<String> {
        let result: Value = self.http.get("/economics/burn/total", &[]).await?;
        Ok(result
            .get("data")
            .and_then(|d| d.get("total_burned"))
            .and_then(|v| v.as_str())
            .unwrap_or_default()
            .into())
    }
    pub async fn get_rules(&self) -> Result<EconomicRules> {
        self.http.get("/economics/rules", &[]).await
    }
    pub async fn update_rules(&self, rules: &EconomicRules) -> Result<EconomicRules> {
        self.http
            .put("/economics/rules", serde_json::to_value(rules)?)
            .await
    }
    pub async fn proofs(&self, query: &[(&str, String)]) -> Result<Vec<PoAuDProof>> {
        self.http.get("/poaud/proofs", query).await
    }
    pub async fn create_proof(
        &self,
        skill_id: &str,
        user_id: &str,
        proof_data: Value,
    ) -> Result<PoAuDProof> {
        required(skill_id, "skill ID")?;
        required(user_id, "user ID")?;
        self.http
            .post(
                "/poaud/proofs",
                json!({"skill_id":skill_id,"user_id":user_id,"proof_data":proof_data}),
            )
            .await
    }
    pub async fn challenges(&self, query: &[(&str, String)]) -> Result<Vec<PoAuDChallenge>> {
        self.http.get("/poaud/challenges", query).await
    }
    pub async fn submit_challenge(
        &self,
        id: &str,
        user_id: &str,
        solution: Value,
    ) -> Result<PoAuDSubmissionResult> {
        required(id, "challenge ID")?;
        required(user_id, "user ID")?;
        self.http
            .post(
                &format!("/poaud/challenges/{id}/submit"),
                json!({"user_id":user_id,"solution":solution}),
            )
            .await
    }
    pub async fn reputation(&self, user_id: &str) -> Result<PoAuDUserReputation> {
        required(user_id, "user ID")?;
        self.http
            .get(&format!("/poaud/reputation/users/{user_id}"), &[])
            .await
    }
    pub async fn leaderboard(&self, query: &[(&str, String)]) -> Result<Value> {
        self.http.get("/poaud/reputation/leaderboard", query).await
    }
    pub async fn paud_enable(&self) -> Result<PoAuDResponse> {
        self.http.post("/poaud/enable", json!({})).await
    }
    pub async fn paud_disable(&self) -> Result<PoAuDResponse> {
        self.http.post("/poaud/disable", json!({})).await
    }
    pub async fn paud_get_status(&self) -> Result<PoAuDStatus> {
        self.http.get("/poaud/status", &[]).await
    }
    pub async fn paud_add_network_author(&self, address: &str) -> Result<PoAuDResponse> {
        required(address, "address")?;
        self.http
            .post("/poaud/network-authors/add", json!({"address": address}))
            .await
    }
    pub async fn paud_remove_network_author(&self, address: &str) -> Result<PoAuDResponse> {
        required(address, "address")?;
        self.http
            .post("/poaud/network-authors/remove", json!({"address": address}))
            .await
    }
    pub async fn paud_list_network_authors(&self) -> Result<NetworkAuthorsResponse> {
        self.http.get("/poaud/network-authors", &[]).await
    }
    pub async fn health(&self) -> Result<HealthStatus> {
        self.http.get("/health", &[]).await
    }
    pub async fn get_gateway_status(&self) -> Result<GatewayStatus> {
        self.http.get("/gateway/status", &[]).await
    }
    pub async fn get_integration_status(&self) -> Result<IntegrationStatus> {
        self.http.get("/economics/integration/status", &[]).await
    }
    pub async fn routes(&self) -> Result<Vec<Route>> {
        self.http.get("/gateway/routes", &[]).await
    }
    pub async fn integrations(&self) -> Result<Value> {
        self.http.get("/integrations", &[]).await
    }
}
fn required(value: &str, field: &str) -> Result<()> {
    if value.trim().is_empty() {
        Err(Error::Validation(format!("{field} is required")))
    } else {
        Ok(())
    }
}
fn validate_skill(input: &SkillInput) -> Result<()> {
    required(&input.name, "name")?;
    required(&input.description, "description")?;
    if input.cost < 0.0 {
        return Err(Error::Validation("cost must be positive".into()));
    }
    if input
        .capabilities
        .as_ref()
        .is_some_and(|items| items.iter().any(|v| v.trim().is_empty()))
    {
        return Err(Error::Validation(
            "capabilities must be non-empty strings".into(),
        ));
    }
    Ok(())
}
