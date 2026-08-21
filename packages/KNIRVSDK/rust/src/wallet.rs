//! Controller-custodied wallet API. User private keys never enter this process.
use crate::{
    client::HttpClient,
    error::{Error, Result},
    signing::{Action, DirectSignRequest, Fee},
    types::SkillInvocationParams,
};
use serde::{Deserialize, Serialize};
use serde_json::{json, Value};
use std::time::{Duration, Instant};
use tokio::time::sleep;

#[derive(Clone, Debug, Deserialize, Serialize)]
pub struct WalletResponse<T = Value> {
    pub code: u16,
    pub status: WalletStatus,
    #[serde(rename = "type")]
    pub response_type: WalletResponseType,
    pub message: Option<String>,
    pub data: Option<T>,
}
#[derive(Clone, Debug, Deserialize, Serialize)]
#[serde(rename_all = "lowercase")]
pub enum WalletStatus {
    Success,
    Failure,
    Reject,
}
#[derive(Clone, Debug, Deserialize, Serialize)]
#[serde(rename_all = "lowercase")]
pub enum WalletResponseType {
    Account,
    Network,
    Sign,
    Transaction,
}
#[derive(Deserialize)]
struct ApprovalRequest {
    request_id: String,
}
#[derive(Deserialize)]
struct ApprovalStatus {
    status: String,
    result: Option<Value>,
    reason: Option<String>,
}
#[derive(Clone, Debug)]
pub struct KnirvWallet {
    http: HttpClient,
    approval_timeout: Duration,
    poll_interval: Duration,
}
impl KnirvWallet {
    pub fn new(http: HttpClient) -> Self {
        Self {
            http,
            approval_timeout: Duration::from_secs(300),
            poll_interval: Duration::from_millis(1500),
        }
    }
    pub fn with_approval_timing(mut self, timeout: Duration, interval: Duration) -> Self {
        self.approval_timeout = timeout;
        self.poll_interval = interval;
        self
    }
    pub async fn account(&self) -> Result<WalletResponse> {
        let data = self.http.get("/api/controller/account", &[]).await?;
        Ok(success(WalletResponseType::Account, data))
    }
    pub async fn add_network(&self, network: Value) -> Result<WalletResponse> {
        self.approved(
            "/api/controller/networks",
            network,
            WalletResponseType::Network,
        )
        .await
    }
    pub async fn switch_network(&self, chain_id: &str) -> Result<WalletResponse> {
        self.approved(
            "/api/controller/networks/switch",
            json!({"chain_id":chain_id}),
            WalletResponseType::Network,
        )
        .await
    }
    pub async fn sign_transaction(
        &self,
        transaction: &DirectSignRequest,
    ) -> Result<WalletResponse> {
        self.approved(
            "/api/controller/signing/requests",
            json!({"kind":"transaction","chain_id":transaction.chain_id,"transaction":transaction}),
            WalletResponseType::Sign,
        )
        .await
    }
    pub async fn sign_message(&self, envelope: Value) -> Result<WalletResponse> {
        self.approved(
            "/api/controller/signing/requests",
            json!({"kind":"message","envelope":envelope}),
            WalletResponseType::Sign,
        )
        .await
    }
    pub async fn do_contract(&self, transaction: &DirectSignRequest) -> Result<WalletResponse> {
        self.approved(
            "/api/controller/signing/requests",
            json!({"kind":"transaction","chain_id":transaction.chain_id,"transaction":transaction}),
            WalletResponseType::Transaction,
        )
        .await
    }
    pub async fn invoke_skill(&self, params: &SkillInvocationParams) -> Result<WalletResponse> {
        let action = Action {
            schema_version: String::new(),
            action: "knirv.skill.invoke".into(),
            sender: params.sender.clone(),
            recipient: Some(params.skill_id.clone()),
            amount: params.amount.parse().ok(),
            payload: params
                .metadata
                .as_ref()
                .map(|m| serde_json::to_vec(m).unwrap_or_default()),
            timestamp_unix: std::time::SystemTime::now()
                .duration_since(std::time::UNIX_EPOCH)
                .map(|d| d.as_secs())
                .unwrap_or(0),
        };
        let fee = match &params.fee {
            Some(f) => Fee {
                denom: f.denom.clone(),
                amount: f.amount.clone(),
                gas_limit: f.gas_limit,
                payer: f.payer.clone(),
                granter: f.granter.clone(),
            },
            None => Fee::default(),
        };
        let request = DirectSignRequest {
            action,
            chain_id: "knirv-1".into(),
            account_number: params.account_number,
            sequence: params.sequence,
            fee,
        };
        self.approved(
            "/api/controller/signing/requests",
            json!({"kind":"transaction","chain_id":"knirv-1","transaction":request}),
            WalletResponseType::Transaction,
        )
        .await
    }
    pub async fn resolve_uri(&self, uri: &str) -> Result<WalletResponse> {
        let data = self
            .http
            .get("/api/transmission/resolve", &[("uri", uri.into())])
            .await?;
        Ok(success(WalletResponseType::Transaction, data))
    }
    pub async fn broadcast(&self, data: Value) -> Result<WalletResponse> {
        let data = self.http.post("/api/transmission/broadcast", data).await?;
        Ok(success(WalletResponseType::Transaction, data))
    }
    async fn approved(
        &self,
        path: &str,
        body: Value,
        kind: WalletResponseType,
    ) -> Result<WalletResponse> {
        let approval: ApprovalRequest = self.http.post(path, body).await?;
        let deadline = Instant::now() + self.approval_timeout;
        while Instant::now() < deadline {
            let state: ApprovalStatus = self
                .http
                .get(
                    &format!("/api/controller/signing/requests/{}", approval.request_id),
                    &[],
                )
                .await?;
            match state.status.as_str() {
                "approved" => return Ok(success(kind, state.result.unwrap_or(Value::Null))),
                "rejected" => {
                    return Ok(WalletResponse {
                        code: 1,
                        status: WalletStatus::Reject,
                        response_type: kind,
                        message: state.reason,
                        data: None,
                    })
                }
                "expired" => {
                    return Ok(WalletResponse {
                        code: 1,
                        status: WalletStatus::Failure,
                        response_type: kind,
                        message: Some("KNIRVCONTROLLER approval expired".into()),
                        data: None,
                    })
                }
                _ => sleep(self.poll_interval).await,
            }
        }
        Err(Error::Timeout)
    }
}
fn success(kind: WalletResponseType, data: Value) -> WalletResponse {
    WalletResponse {
        code: 0,
        status: WalletStatus::Success,
        response_type: kind,
        message: None,
        data: Some(data),
    }
}
