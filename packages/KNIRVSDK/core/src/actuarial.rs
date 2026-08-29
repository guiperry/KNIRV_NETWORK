//! Typed access to KNIRVSERVER's unified actuarial syndicate API.
//! Endpoint paths live here so applications do not each grow a private BFF.
//!
//! ## Signing decision
//! `knirv.action.v1` is reserved for Cosmos `SIGN_MODE_DIRECT` transactions.
//! Actuarial requests are authenticated API operations, so their
//! `signed_intent` compatibility field carries the SDK's canonical
//! `knirv.message.v1` `SignedMessage` over its canonical JSON payload. This
//! deliberately avoids treating an API mutation as a chain transaction while
//! retaining one audited SDK signing implementation. New consumers must use
//! `SignedActuarialIntent`, not introduce a second signature format.
use crate::{client::HttpClient, error::Result, types::*};
use serde::Serialize;

#[derive(Clone, Debug)]
pub struct ActuarialService { http: HttpClient }

impl ActuarialService {
    pub fn new(http: HttpClient) -> Self { Self { http } }
    async fn post<T: serde::de::DeserializeOwned, R: Serialize>(&self, path: &str, request: &R) -> Result<T> {
        self.http.post(path, serde_json::to_value(request)?).await
    }
    pub async fn risk_classes(&self) -> Result<Vec<ActuarialRiskClass>> { self.http.get("/api/v1/actuarial/risk-classes", &[]).await }
    pub async fn pools(&self) -> Result<Vec<SyndicatePool>> { self.http.get("/api/v1/actuarial/pools", &[]).await }
    pub async fn pool(&self, id: &str) -> Result<SyndicatePool> { self.http.get(&format!("/api/v1/actuarial/pools/{id}"), &[]).await }
    pub async fn create_pool(&self, request: &CreateSyndicatePoolRequest) -> Result<SyndicatePool> { self.post("/api/v1/actuarial/pools", request).await }
    pub async fn create_stake(&self, pool_id: &str, request: &CreateStakeRequest) -> Result<StakePosition> { self.post(&format!("/api/v1/actuarial/pools/{pool_id}/stakes"), request).await }
    pub async fn request_stake_exit(&self, stake_id: &str, request: &RequestStakeExitRequest) -> Result<()> { self.post(&format!("/api/v1/actuarial/stakes/{stake_id}/exit"), request).await }
    pub async fn submission(&self, id: &str) -> Result<VulnerabilitySubmission> { self.http.get(&format!("/api/v1/actuarial/submissions/{id}"), &[]).await }
    pub async fn submissions(&self, resolver_wallet: Option<&str>, status: Option<&str>) -> Result<Vec<VulnerabilitySubmission>> {
        let mut query = Vec::new();
        if let Some(wallet) = resolver_wallet { query.push(("resolver_wallet", wallet.to_owned())); }
        if let Some(status) = status { query.push(("status", status.to_owned())); }
        self.http.get("/api/v1/actuarial/submissions", &query).await
    }
    pub async fn create_submission(&self, request: &CreateSubmissionRequest) -> Result<VulnerabilitySubmission> { self.post("/api/v1/actuarial/submissions", request).await }
    pub async fn claim_submission(&self, id: &str, request: &ClaimSubmissionRequest) -> Result<VulnerabilitySubmission> { self.post(&format!("/api/v1/actuarial/submissions/{id}/claim"), request).await }
    pub async fn create_submission_artifacts(&self, id: &str, request: &CreateSubmissionArtifactsRequest) -> Result<ValidationResult> { self.post(&format!("/api/v1/actuarial/submissions/{id}/artifacts"), request).await }
    pub async fn request_quote(&self, id: &str, request: &RequestQuoteRequest) -> Result<QuoteResult> { self.post(&format!("/api/v1/actuarial/submissions/{id}/quote"), request).await }
    pub async fn decision(&self, id: &str) -> Result<PricingDecision> { self.http.get(&format!("/api/v1/actuarial/decisions/{id}"), &[]).await }
    pub async fn settlement(&self, id: &str) -> Result<Settlement> { self.http.get(&format!("/api/v1/actuarial/settlements/{id}"), &[]).await }
    pub async fn create_risk_class(&self, request: &CreateActuarialRiskClassRequest) -> Result<ActuarialRiskClass> { self.post("/api/v1/actuarial/risk-classes", request).await }
    pub async fn enterprise_credential(&self, request: &CreateEnterpriseCredentialRequest) -> Result<EnterpriseCredential> { self.post("/api/v1/actuarial/credentials/enterprise", request).await }
    pub async fn credential_challenge(&self, request: &CredentialChallengeRequest) -> Result<CredentialChallenge> { self.post("/api/v1/actuarial/credentials/challenge", request).await }
    pub async fn exposure(&self, wallet: &str) -> Result<ExposureSummary> { self.http.get("/api/v1/actuarial/me/exposure", &[("wallet", wallet.into())]).await }
    pub async fn pool_report(&self, id: &str) -> Result<PoolReport> { self.http.get(&format!("/api/v1/actuarial/reports/pools/{id}"), &[]).await }
    pub async fn create_payout_destination(&self, request: &CreatePayoutDestinationRequest) -> Result<PayoutDestination> { self.post("/api/v1/actuarial/payout-destinations", request).await }
}
