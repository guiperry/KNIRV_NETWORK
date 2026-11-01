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
use crate::{
    client::{HttpClient, Method},
    error::{Error, Result},
    types::*,
};
use serde::Serialize;
use serde_json::Value;

#[derive(Clone, Debug)]
pub struct ActuarialService {
    http: HttpClient,
}

impl ActuarialService {
    pub fn new(http: HttpClient) -> Self {
        Self { http }
    }
    async fn post<T: serde::de::DeserializeOwned, R: Serialize>(
        &self,
        path: &str,
        request: &R,
    ) -> Result<T> {
        self.http.post(path, serde_json::to_value(request)?).await
    }
    pub async fn risk_classes(&self) -> Result<Vec<ActuarialRiskClass>> {
        self.http.get("/api/v1/actuarial/risk-classes", &[]).await
    }
    pub async fn pools(&self) -> Result<Vec<SyndicatePool>> {
        self.http.get("/api/v1/actuarial/pools", &[]).await
    }
    pub async fn pool(&self, id: &str) -> Result<SyndicatePool> {
        self.http
            .get(&format!("/api/v1/actuarial/pools/{id}"), &[])
            .await
    }
    pub async fn create_pool(&self, request: &CreateSyndicatePoolRequest) -> Result<SyndicatePool> {
        self.post("/api/v1/actuarial/pools", request).await
    }
    pub async fn create_stake(
        &self,
        pool_id: &str,
        request: &CreateStakeRequest,
    ) -> Result<StakePosition> {
        self.post(
            &format!("/api/v1/actuarial/pools/{pool_id}/stakes"),
            request,
        )
        .await
    }
    pub async fn request_stake_exit(
        &self,
        stake_id: &str,
        request: &RequestStakeExitRequest,
    ) -> Result<()> {
        self.post(
            &format!("/api/v1/actuarial/stakes/{stake_id}/exit"),
            request,
        )
        .await
    }
    pub async fn submission(&self, id: &str) -> Result<VulnerabilitySubmission> {
        self.http
            .get(&format!("/api/v1/actuarial/submissions/{id}"), &[])
            .await
    }
    pub async fn submissions(
        &self,
        resolver_wallet: Option<&str>,
        status: Option<&str>,
    ) -> Result<Vec<VulnerabilitySubmission>> {
        let mut query = Vec::new();
        if let Some(wallet) = resolver_wallet {
            query.push(("resolver_wallet", wallet.to_owned()));
        }
        if let Some(status) = status {
            query.push(("status", status.to_owned()));
        }
        self.http.get("/api/v1/actuarial/submissions", &query).await
    }
    pub async fn create_submission(
        &self,
        request: &CreateSubmissionRequest,
    ) -> Result<VulnerabilitySubmission> {
        self.post("/api/v1/actuarial/submissions", request).await
    }
    pub async fn claim_submission(
        &self,
        id: &str,
        request: &ClaimSubmissionRequest,
    ) -> Result<VulnerabilitySubmission> {
        self.post(
            &format!("/api/v1/actuarial/submissions/{id}/claim"),
            request,
        )
        .await
    }
    pub async fn create_submission_artifacts(
        &self,
        id: &str,
        request: &CreateSubmissionArtifactsRequest,
    ) -> Result<ValidationResult> {
        self.post(
            &format!("/api/v1/actuarial/submissions/{id}/artifacts"),
            request,
        )
        .await
    }
    pub async fn request_quote(
        &self,
        id: &str,
        request: &RequestQuoteRequest,
    ) -> Result<QuoteResult> {
        self.post(
            &format!("/api/v1/actuarial/submissions/{id}/quote"),
            request,
        )
        .await
    }
    pub async fn decision(&self, id: &str) -> Result<PricingDecision> {
        self.http
            .get(&format!("/api/v1/actuarial/decisions/{id}"), &[])
            .await
    }
    pub async fn settlement(&self, id: &str) -> Result<Settlement> {
        self.http
            .get(&format!("/api/v1/actuarial/settlements/{id}"), &[])
            .await
    }
    pub async fn create_risk_class(
        &self,
        request: &CreateActuarialRiskClassRequest,
    ) -> Result<ActuarialRiskClass> {
        self.post("/api/v1/actuarial/risk-classes", request).await
    }
    pub async fn enterprise_credential(
        &self,
        request: &CreateEnterpriseCredentialRequest,
    ) -> Result<EnterpriseCredential> {
        self.post("/api/v1/actuarial/credentials/enterprise", request)
            .await
    }
    pub async fn credential_challenge(
        &self,
        request: &CredentialChallengeRequest,
    ) -> Result<CredentialChallenge> {
        self.post("/api/v1/actuarial/credentials/challenge", request)
            .await
    }
    pub async fn exposure(&self, wallet: &str) -> Result<ExposureSummary> {
        self.http
            .get(
                "/api/v1/actuarial/me/exposure",
                &[("wallet", wallet.into())],
            )
            .await
    }
    pub async fn pool_report(&self, id: &str) -> Result<PoolReport> {
        self.http
            .get(&format!("/api/v1/actuarial/reports/pools/{id}"), &[])
            .await
    }
    pub async fn create_payout_destination(
        &self,
        request: &CreatePayoutDestinationRequest,
    ) -> Result<PayoutDestination> {
        self.post("/api/v1/actuarial/payout-destinations", request)
            .await
    }

    /// Runs one named API call and returns the server's JSON untouched. This is
    /// the single table of endpoint verbs and paths: WebAssembly hosts reach it
    /// through the `actuarial.call` binding operation instead of keeping a
    /// private copy of the routes. `params` carries `id` (path segment), `body`
    /// and `query` as the operation needs them.
    pub async fn call(&self, op: &str, params: &Value) -> Result<Value> {
        const BASE: &str = "/api/v1/actuarial";
        let id = || -> Result<String> {
            params
                .get("id")
                .and_then(Value::as_str)
                .filter(|id| !id.is_empty())
                .map(path_segment)
                .ok_or_else(|| Error::Validation(format!("{op} requires an id")))
        };
        let body = || params.get("body").cloned().unwrap_or(Value::Null);
        let query = |names: &[&'static str]| -> Vec<(&'static str, String)> {
            names
                .iter()
                .filter_map(|name| {
                    let value = params.get("query")?.get(*name)?.as_str()?;
                    (!value.is_empty()).then(|| (*name, value.to_owned()))
                })
                .collect()
        };
        let (method, path, query) = match op {
            "risk_classes" => (Method::GET, format!("{BASE}/risk-classes"), vec![]),
            "pools" => (Method::GET, format!("{BASE}/pools"), vec![]),
            "pool" => (Method::GET, format!("{BASE}/pools/{}", id()?), vec![]),
            "submission" => (Method::GET, format!("{BASE}/submissions/{}", id()?), vec![]),
            "submissions" => (
                Method::GET,
                format!("{BASE}/submissions"),
                query(&["resolver_wallet", "status"]),
            ),
            "decision" => (Method::GET, format!("{BASE}/decisions/{}", id()?), vec![]),
            "settlement" => (Method::GET, format!("{BASE}/settlements/{}", id()?), vec![]),
            "exposure" => (Method::GET, format!("{BASE}/me/exposure"), query(&["wallet"])),
            "pool_report" => (Method::GET, format!("{BASE}/reports/pools/{}", id()?), vec![]),
            "create_submission" => (Method::POST, format!("{BASE}/submissions"), vec![]),
            "create_stake" => (Method::POST, format!("{BASE}/pools/{}/stakes", id()?), vec![]),
            "request_stake_exit" => (Method::POST, format!("{BASE}/stakes/{}/exit", id()?), vec![]),
            "create_submission_artifacts" => {
                (Method::POST, format!("{BASE}/submissions/{}/artifacts", id()?), vec![])
            }
            "request_quote" => (Method::POST, format!("{BASE}/submissions/{}/quote", id()?), vec![]),
            "claim_submission" => (Method::POST, format!("{BASE}/submissions/{}/claim", id()?), vec![]),
            "create_payout_destination" => (Method::POST, format!("{BASE}/payout-destinations"), vec![]),
            "create_risk_class" => (Method::POST, format!("{BASE}/risk-classes"), vec![]),
            "create_enterprise_credential" => (Method::POST, format!("{BASE}/credentials/enterprise"), vec![]),
            other => return Err(Error::Validation(format!("unknown actuarial operation: {other}"))),
        };
        let body = (method == Method::POST).then(body);
        self.http.request(method, &path, &query, body).await
    }
}

/// Percent-encodes one path segment (what `encodeURIComponent` produced in the
/// TypeScript client).
fn path_segment(raw: &str) -> String {
    let mut out = String::with_capacity(raw.len());
    for byte in raw.bytes() {
        match byte {
            b'A'..=b'Z' | b'a'..=b'z' | b'0'..=b'9' | b'-' | b'_' | b'.' | b'~' => out.push(byte as char),
            _ => out.push_str(&format!("%{byte:02X}")),
        }
    }
    out
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::http_transport::{block_on, BoxFuture, HttpRequest, HttpResponse, HttpTransport, TransportError};
    use crate::ClientConfig;
    use serde_json::json;
    use std::sync::{Arc, Mutex};
    use std::time::Duration;

    #[derive(Debug, Default)]
    struct Recorder(Mutex<Vec<HttpRequest>>);
    impl HttpTransport for Recorder {
        fn send(&self, request: HttpRequest) -> BoxFuture<'_, std::result::Result<HttpResponse, TransportError>> {
            self.0.lock().unwrap().push(request);
            Box::pin(std::future::ready(Ok(HttpResponse { status: 200, body: b"{\"ok\":true}".to_vec(), ..Default::default() })))
        }
        fn sleep(&self, _: Duration) -> BoxFuture<'_, ()> {
            Box::pin(std::future::ready(()))
        }
    }

    fn service() -> (ActuarialService, Arc<Recorder>) {
        let recorder = Arc::new(Recorder::default());
        let config = ClientConfig { transport: Some(recorder.clone()), ..Default::default() };
        (ActuarialService::new(HttpClient::new("https://backend.test", config).unwrap()), recorder)
    }

    #[test]
    fn every_operation_maps_to_its_verb_and_path() {
        let cases = [
            ("risk_classes", json!({}), "GET", "/api/v1/actuarial/risk-classes"),
            ("pool", json!({"id": "a b/c"}), "GET", "/api/v1/actuarial/pools/a%20b%2Fc"),
            ("submissions", json!({"query": {"status": "open", "resolver_wallet": ""}}), "GET", "/api/v1/actuarial/submissions?status=open"),
            ("exposure", json!({"query": {"wallet": "w1"}}), "GET", "/api/v1/actuarial/me/exposure?wallet=w1"),
            ("exposure", json!({}), "GET", "/api/v1/actuarial/me/exposure"),
            ("pool_report", json!({"id": "p"}), "GET", "/api/v1/actuarial/reports/pools/p"),
            ("create_stake", json!({"id": "p", "body": {"x": 1}}), "POST", "/api/v1/actuarial/pools/p/stakes"),
            ("request_stake_exit", json!({"id": "s", "body": {}}), "POST", "/api/v1/actuarial/stakes/s/exit"),
            ("claim_submission", json!({"id": "s", "body": {}}), "POST", "/api/v1/actuarial/submissions/s/claim"),
            ("create_enterprise_credential", json!({"body": {}}), "POST", "/api/v1/actuarial/credentials/enterprise"),
        ];
        for (op, params, method, path) in cases {
            let (svc, rec) = service();
            block_on(svc.call(op, &params)).unwrap_or_else(|e| panic!("{op}: {e}"));
            let seen = rec.0.lock().unwrap();
            assert_eq!(seen[0].method, method, "{op}");
            assert_eq!(seen[0].url, format!("https://backend.test{path}"), "{op}");
        }
    }

    #[test]
    fn a_post_carries_its_body_and_unknown_ops_and_missing_ids_fail() {
        let (svc, rec) = service();
        block_on(svc.call("create_submission", &json!({"body": {"a": 1}}))).unwrap();
        assert_eq!(rec.0.lock().unwrap()[0].body, br#"{"a":1}"#);
        assert!(matches!(block_on(svc.call("nope", &json!({}))), Err(Error::Validation(_))));
        assert!(matches!(block_on(svc.call("pool", &json!({}))), Err(Error::Validation(_))));
        assert_eq!(rec.0.lock().unwrap().len(), 1, "invalid calls must not reach the network");
    }
}
