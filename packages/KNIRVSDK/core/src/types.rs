use serde::{Deserialize, Serialize};
use serde_json::Value;
use std::collections::BTreeMap;

/// Typed wire models for KNIRVSERVER's actuarial syndicate API.  These are
/// intentionally separate from the chain transaction types below: an
/// actuarial mutation is an authenticated API operation, not a Cosmos direct
/// transaction.
#[derive(Clone, Debug, Default, Deserialize, Serialize)]
pub struct ActuarialRiskClass {
    pub id: String,
    pub domain: String,
    pub taxonomy_version: String,
    pub display_name: String,
    pub description: String,
    #[serde(default)] pub required_features: Vec<String>,
    #[serde(default)] pub supported_runtimes: Vec<String>,
    #[serde(default)] pub eligibility_policy: BTreeMap<String, String>,
    pub curated_challenge: Option<CuratedCodeErrorChallenge>,
    pub limits: ActuarialRiskClassLimits,
    pub status: String,
    pub min_sample_size: i64,
    pub created_at: String,
    pub updated_at: String,
    pub retired_at: Option<String>,
}

#[derive(Clone, Debug, Default, Deserialize, Serialize)]
pub struct CuratedCodeErrorChallenge {
    pub legacy_id: String,
    pub r#type: String,
    pub buggy_code: String,
    pub context: String,
    #[serde(default)] pub hints: Vec<String>,
}

#[derive(Clone, Debug, Default, Deserialize, Serialize)]
pub struct ActuarialRiskClassLimits {
    pub max_payout_per_claim: i64,
    pub max_aggregate_liability: i64,
    pub max_blast_radius: i64,
}

#[derive(Clone, Debug, Default, Deserialize, Serialize)]
pub struct CreateActuarialRiskClassRequest {
    pub id: String,
    pub display_name: String,
    pub description: String,
    pub taxonomy_version: String,
    pub domain: String,
    pub difficulty_tier: Option<i64>,
    pub limits: ActuarialRiskClassLimits,
    pub curated_challenge: Option<CuratedCodeErrorChallenge>,
}

#[derive(Clone, Debug, Default, Deserialize, Serialize)]
pub struct SyndicatePool {
    pub id: String,
    pub risk_class_id: String,
    pub currency: String,
    pub rail: String,
    pub status: String,
    pub total_stake: i64,
    pub liquid_balance: i64,
    pub reserved_balance: i64,
    pub created_at: String,
    pub updated_at: String,
}

#[derive(Clone, Debug, Default, Deserialize, Serialize)]
pub struct CreateSyndicatePoolRequest { pub risk_class_id: String, pub currency: String, pub rail: String }

#[derive(Clone, Debug, Default, Deserialize, Serialize)]
pub struct SignedActuarialIntent {
    pub nonce: String,
    pub canonical_payload: String,
    /// JSON-encoded `knirv.message.v1` SignedMessage. See `actuarial.rs`.
    pub signature: String,
}

#[derive(Clone, Debug, Default, Deserialize, Serialize)]
pub struct CreateStakeRequest {
    pub amount: i64, pub operator_wallet: String, pub node_id: String,
    pub credential_commitment: String, pub signed_intent: SignedActuarialIntent,
}
#[derive(Clone, Debug, Default, Deserialize, Serialize)]
pub struct StakePosition {
    pub id: String, pub pool_id: String, pub operator_wallet: String,
    pub deposited_amount: i64, pub locked_amount: i64, pub withdrawable_amount: i64,
    pub status: String,
}
#[derive(Clone, Debug, Default, Deserialize, Serialize)]
pub struct RequestStakeExitRequest { pub operator_wallet: String, pub signed_intent: SignedActuarialIntent }

#[derive(Clone, Debug, Default, Deserialize, Serialize)]
pub struct CreateSubmissionRequest {
    pub researcher_credential_id: String, pub researcher_wallet: String,
    pub claimed_risk_class: String, pub idempotency_key: String,
    pub enterprise_credential_id: Option<String>, pub signed_intent: Option<SignedActuarialIntent>,
}
#[derive(Clone, Debug, Default, Deserialize, Serialize)]
pub struct ClaimSubmissionRequest { pub resolver_wallet: String, pub signed_intent: SignedActuarialIntent }
#[derive(Clone, Debug, Default, Deserialize, Serialize)]
pub struct VulnerabilitySubmission {
    pub id: String, pub researcher_credential_id: String, pub researcher_wallet: String,
    pub claimed_risk_class: String, pub domain: String, pub credential_type: String,
    pub enterprise_credential_id: Option<String>, pub resolver_wallet: Option<String>,
    pub validation_state: String, pub validation_result_hash: Option<String>,
    pub idempotency_key: String, pub status: String, pub decision_id: Option<String>,
    pub created_at: String, pub updated_at: String,
}
#[derive(Clone, Debug, Default, Deserialize, Serialize)]
pub struct CreateSubmissionArtifactsRequest {
    pub researcher_wallet: String, pub poc_reference: String, pub report_reference: String,
    pub poc_hash: String, pub report_hash: String, pub scope_hash: String,
    pub dve_id: String, pub dve_session_id: String, pub signed_intent: Option<SignedActuarialIntent>,
}
#[derive(Clone, Debug, Default, Deserialize, Serialize)]
pub struct ValidationResult { pub state: String, pub result_hash: Option<String>, pub reason: Option<String> }
#[derive(Clone, Debug, Default, Deserialize, Serialize)]
pub struct RequestQuoteRequest {
    pub model_id: String, pub snapshot_id: String, pub pool_id: String,
    #[serde(default)] pub feature_values: BTreeMap<String, i64>, pub confidence: i64,
    pub requester_wallet: String, pub signed_intent: Option<SignedActuarialIntent>,
}
#[derive(Clone, Debug, Default, Deserialize, Serialize)]
pub struct PricingDecision {
    pub id: String, pub snapshot_id: String, pub model_id: String, pub pool_id: String,
    pub quote_amount: i64, pub reserve_amount: i64, pub decision_hash: String,
    #[serde(default)] pub reason_codes: Vec<String>, pub created_at: String,
}
#[derive(Clone, Debug, Default, Deserialize, Serialize)]
pub struct QuoteResult { pub decision: PricingDecision, pub reservation_id: Option<String> }
#[derive(Clone, Debug, Default, Deserialize, Serialize)]
pub struct Settlement {
    pub id: String, pub decision_id: String, pub pool_id: String, pub submission_id: String,
    pub amount: i64, pub provider_id: String, pub provider_status: String, pub status: String,
    pub created_at: String, pub updated_at: String,
}
#[derive(Clone, Debug, Default, Deserialize, Serialize)]
pub struct ReserveLedgerEntry { pub entity_type: String, pub amount: i64, pub direction: String, pub reference_id: String }
#[derive(Clone, Debug, Default, Deserialize, Serialize)]
pub struct PoolReport {
    pub pool: SyndicatePool, pub active_stake_count: usize,
    #[serde(default)] pub reserve_entries: Vec<ReserveLedgerEntry>, pub available_liquidity: i64,
}
#[derive(Clone, Debug, Default, Deserialize, Serialize)]
pub struct StakeExposure { pub stake: StakePosition, pub gross_exposure: i64, pub realized_loss: i64 }
#[derive(Clone, Debug, Default, Deserialize, Serialize)]
pub struct ExposureSummary { pub wallet: String, #[serde(default)] pub positions: Vec<StakeExposure>, pub gross_exposure: i64, pub realized_loss: i64 }
#[derive(Clone, Debug, Default, Deserialize, Serialize)]
pub struct CreatePayoutDestinationRequest { pub wallet: String, pub country: String, pub email: String }
#[derive(Clone, Debug, Default, Deserialize, Serialize)]
pub struct PayoutDestination { pub wallet: String, pub provider: String, pub provider_account_id: String }
#[derive(Clone, Debug, Default, Deserialize, Serialize)]
pub struct CredentialChallengeRequest { pub wallet: String }
#[derive(Clone, Debug, Default, Deserialize, Serialize)]
pub struct CredentialChallenge { pub id: String, pub wallet: String, pub nonce: String, pub expires_at: String }
#[derive(Clone, Debug, Default, Deserialize, Serialize)]
pub struct CreateEnterpriseCredentialRequest { pub organization_id: String, pub organization_name: String, pub expires_at: Option<String> }
#[derive(Clone, Debug, Default, Deserialize, Serialize)]
pub struct EnterpriseCredential { pub id: String, pub organization_id: String, pub organization_name: String, pub session_subject: String, pub expires_at: String }

#[derive(Clone, Debug, Default, Deserialize, Serialize)]
pub struct Transaction {
    pub id: String,
    pub fee: String,
    pub from: String,
    pub public_key: String,
    pub signature: String,
    pub timestamp: u64,
    #[serde(rename = "type")]
    pub transaction_type: String,
    pub version: u32,
    pub chain_id: Option<String>,
    pub account_number: Option<String>,
    pub transaction_hash: Option<String>,
    pub body_bytes: Option<String>,
    pub auth_info_bytes: Option<String>,
    pub signatures: Option<Vec<String>>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub data: Option<Value>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub to: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub value: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub status: Option<String>,
}
#[derive(Clone, Debug, Default, Deserialize, Serialize)]
pub struct Block {
    pub block_number: Option<u64>,
    pub hash: Option<String>,
    pub nonce: Option<u64>,
    #[serde(rename = "prevHash")]
    pub prev_hash: Option<String>,
    pub proposer_address: Option<String>,
    pub timestamp: Option<u64>,
    pub transactions: Option<Vec<Transaction>>,
}
#[derive(Clone, Debug, Default, Deserialize, Serialize)]
pub struct Chain {
    pub blocks: Option<Vec<Block>>,
    pub chain_address: Option<String>,
    pub chain_id: Option<String>,
    pub reflections: Option<Value>,
    pub transaction_pool: Option<Vec<Transaction>>,
}
#[derive(Clone, Debug, Default, Deserialize, Serialize)]
pub struct SubmitResponse {
    pub message: Option<String>,
    #[serde(alias = "transactionHash")]
    pub transaction_hash: Option<String>,
    pub success: Option<bool>,
}
#[derive(Clone, Debug, Default, Deserialize, Serialize)]
pub struct ResourceUri {
    pub content_hash: Option<String>,
    pub metadata: Option<Value>,
    pub owner: Option<String>,
    pub resource_type: Option<String>,
}
#[derive(Clone, Debug, Default, Deserialize, Serialize)]
pub struct UriResponse {
    pub message: Option<String>,
    pub resource_id: Option<String>,
    pub success: Option<bool>,
    pub uri: Option<String>,
}
#[derive(Clone, Debug, Default, Deserialize, Serialize)]
pub struct Skill {
    pub id: String,
    pub name: String,
    pub description: String,
    pub cost: f64,
    pub capabilities: Option<Vec<String>>,
    pub success_rate: Option<f64>,
    pub usage_count: Option<u64>,
    pub total_earned: Option<f64>,
    pub created_at: Option<String>,
    pub updated_at: Option<String>,
    #[serde(flatten)]
    pub extra: std::collections::BTreeMap<String, Value>,
}
#[derive(Clone, Debug, Default, Deserialize, Serialize)]
pub struct SkillInput {
    pub name: String,
    pub description: String,
    pub cost: f64,
    pub capabilities: Option<Vec<String>>,
}
#[derive(Clone, Debug, Default, Deserialize, Serialize)]
pub struct ListResponse<T> {
    pub items: Option<Vec<T>>,
    pub data: Option<Vec<T>>,
    pub total: Option<u64>,
    pub page: Option<u64>,
    pub per_page: Option<u64>,
}
#[derive(Clone, Debug, Default, Deserialize, Serialize)]
pub struct HealthStatus {
    pub status: Option<String>,
    pub timestamp: Option<String>,
    pub services: Option<Value>,
    #[serde(flatten)]
    pub extra: std::collections::BTreeMap<String, Value>,
}

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
#[derive(Clone, Debug, Deserialize, Serialize)]
#[serde(rename_all = "SCREAMING_SNAKE_CASE")]
pub enum WalletResponseExecuteType {
    AddEstablish,
    GetAccount,
    AddNetwork,
    SwitchNetwork,
    DoContract,
    SignTx,
}
#[derive(Clone, Debug, Deserialize, Serialize)]
#[serde(rename_all = "SCREAMING_SNAKE_CASE")]
pub enum WalletResponseFailureType {
    NetworkTimeout,
    UnapprovedChain,
    UnapprovedHost,
    LockedAccount,
    InvalidFormat,
    InvalidTransaction,
    UnexpectedError,
}
#[derive(Clone, Debug, Deserialize, Serialize)]
#[serde(rename_all = "SCREAMING_SNAKE_CASE")]
pub enum WalletResponseRejectType {
    EstablishRejected,
    SignRejected,
    TransactionRejected,
}
#[derive(Clone, Debug, Deserialize, Serialize)]
#[serde(rename_all = "SCREAMING_SNAKE_CASE")]
pub enum WalletResponseSuccessType {
    EstablishSuccess,
    SignSuccess,
    TransactionSuccess,
}

#[derive(Clone, Debug, Deserialize, Serialize)]
pub struct SkillInvocationParams {
    pub user_id: String,
    pub skill_id: String,
    pub sender: String,
    pub amount: String,
    pub account_number: u64,
    pub sequence: u64,
    pub fee: Option<KnirvFee>,
    pub metadata: Option<Value>,
}

#[derive(Clone, Debug, Default, Deserialize, Serialize)]
pub struct KnirvFee {
    pub denom: Option<String>,
    pub amount: Option<String>,
    pub gas_limit: Option<u64>,
    pub payer: Option<String>,
    pub granter: Option<String>,
}

#[derive(Clone, Debug, Deserialize, Serialize)]
pub struct MessageEnvelope {
    pub schema_version: Option<String>,
    pub domain: String,
    pub purpose: String,
    pub chain_id: String,
    pub nonce: String,
    pub issued_at_unix: u64,
    pub expires_at_unix: u64,
    pub payload: Option<Vec<u8>>,
}
#[derive(Clone, Debug, Deserialize, Serialize)]
pub struct SignedMessageEnvelope {
    pub envelope: String,
    pub signature: String,
    pub public_key: String,
    pub address: String,
}
#[derive(Clone, Debug, Deserialize, Serialize)]
pub struct ParsedMessageEnvelope {
    pub schema_version: String,
    pub domain: String,
    pub purpose: String,
    pub chain_id: String,
    pub nonce: String,
    pub issued_at_unix: u64,
    pub expires_at_unix: u64,
    pub payload: Vec<u8>,
}

#[derive(Clone, Debug, Default, Deserialize, Serialize)]
pub struct OracleStatus {
    pub chain_id: Option<String>,
    pub network_id: Option<String>,
    pub token: Option<OracleTokenInfo>,
    pub consensus: Option<OracleConsensusInfo>,
    pub governance: Option<Value>,
    pub economics: Option<Value>,
    pub p2p: Option<Value>,
    pub ibc: Option<OracleIBCInfo>,
}
#[derive(Clone, Debug, Default, Deserialize, Serialize)]
pub struct OracleTokenInfo {
    pub name: Option<String>,
    pub symbol: Option<String>,
    pub total_supply: Option<String>,
    pub max_supply: Option<String>,
    pub contract_address: Option<String>,
}
#[derive(Clone, Debug, Default, Deserialize, Serialize)]
pub struct OracleConsensusInfo {
    pub latest_block_height: Option<i64>,
    pub validator_count: Option<u32>,
}
#[derive(Clone, Debug, Default, Deserialize, Serialize)]
pub struct OracleIBCInfo {
    pub enabled: Option<bool>,
}

#[derive(Clone, Debug, Default, Deserialize, Serialize)]
pub struct GovernanceDIDDocument {
    pub id: Option<String>,
    pub document: Option<Value>,
}
#[derive(Clone, Debug, Deserialize, Serialize)]
pub struct IdentityEnvelope {
    pub node_id: String,
    pub agent_id: String,
    pub source: String,
}
#[derive(Clone, Debug, Deserialize, Serialize)]
pub struct PolicyInput {
    pub node_id: String,
    pub action: String,
    pub action_type: String,
}
#[derive(Clone, Debug, Deserialize, Serialize)]
pub struct ComplianceEvent {
    pub node_id: String,
    pub agent_id: String,
    pub event_type: String,
    pub severity: String,
}
#[derive(Clone, Debug, Deserialize, Serialize)]
pub struct BreakerSuccessRequest {
    pub breaker_id: String,
}
#[derive(Clone, Debug, Deserialize, Serialize)]
pub struct SLODefinition {
    pub name: String,
    pub target: f64,
}

#[derive(Clone, Debug, Default, Deserialize, Serialize)]
pub struct GatewayStatus {
    pub status: Option<String>,
    pub version: Option<String>,
    pub uptime: Option<u64>,
    pub services: Option<Value>,
    pub last_updated: Option<String>,
}
#[derive(Clone, Debug, Default, Deserialize, Serialize)]
pub struct IntegrationStatus {
    pub knirvchain_url: Option<String>,
    pub knirvnexus_url: Option<String>,
    pub knirvoracle_url: Option<String>,
    pub knirvgraph_url: Option<String>,
    pub last_sync: Option<String>,
    pub status: Option<String>,
}
#[derive(Clone, Debug, Default, Deserialize, Serialize)]
pub struct Route {
    pub path: Option<String>,
    pub methods: Option<Vec<String>>,
    pub target: Option<String>,
    pub auth_required: Option<bool>,
    pub rate_limit: Option<u32>,
}
#[derive(Clone, Debug, Default, Deserialize, Serialize)]
pub struct LLMRegistrationRequest {
    pub user_id: String,
    pub llm_id: String,
    pub registration_fee: String,
}
#[derive(Clone, Debug, Default, Deserialize, Serialize)]
pub struct LLMRegistrationResponse {
    pub transaction_id: Option<String>,
    pub status: Option<String>,
    pub fee: Option<String>,
    pub timestamp: Option<String>,
}
#[derive(Clone, Debug, Default, Deserialize, Serialize)]
pub struct ValidationRewardRequest {
    pub validator_id: String,
    pub target_id: String,
    pub validation_result: bool,
}
#[derive(Clone, Debug, Default, Deserialize, Serialize)]
pub struct ValidationRewardResponse {
    pub transaction_id: Option<String>,
    pub status: Option<String>,
    pub reward: Option<String>,
    pub timestamp: Option<String>,
}
#[derive(Clone, Debug, Default, Deserialize, Serialize)]
pub struct FeeCalculationRequest {
    pub skill_id: Option<String>,
    pub transaction_type: String,
    pub amount: f64,
    pub priority: String,
}
#[derive(Clone, Debug, Default, Deserialize, Serialize)]
pub struct FeeStructure {
    pub base_fee_rate: Option<f64>,
    pub base_fee_percentage: Option<f64>,
    pub priority_rates: Option<std::collections::BTreeMap<String, f64>>,
    pub minimum_fee: Option<f64>,
    pub maximum_fee: Option<f64>,
    pub currency: Option<String>,
}
#[derive(Clone, Debug, Default, Deserialize, Serialize)]
pub struct NetworkFeesResponse {
    pub gas_used: Option<u64>,
    pub priority: Option<String>,
    pub total_fee: Option<f64>,
    pub base_fee: Option<f64>,
    pub priority_fee: Option<f64>,
    pub gas_price: Option<String>,
    pub currency: Option<String>,
}
#[derive(Clone, Debug, Default, Deserialize, Serialize)]
pub struct EconomicOverview {
    pub total_transactions: Option<u64>,
    pub total_volume: Option<f64>,
    pub total_revenue: Option<u64>,
    pub active_users: Option<u32>,
    pub network_health: Option<String>,
}
#[derive(Clone, Debug, Default, Deserialize, Serialize)]
pub struct SkillMetrics {
    pub skill_id: Option<String>,
    pub total_invocations: Option<u32>,
    pub success_rate: Option<f64>,
    pub average_reward: Option<f64>,
    pub total_earnings: Option<f64>,
}
#[derive(Clone, Debug, Default, Deserialize, Serialize)]
pub struct ServiceEconomics {
    pub revenue: Option<String>,
    pub costs: Option<String>,
    pub profit: Option<String>,
    pub tokens_earned: Option<String>,
    pub tokens_spent: Option<String>,
    pub user_count: Option<u32>,
    pub transaction_count: Option<u32>,
    pub last_updated: Option<String>,
}
#[derive(Clone, Debug, Default, Deserialize, Serialize)]
pub struct EconomicMetrics {
    pub total_supply: Option<String>,
    pub circulating_supply: Option<String>,
    pub total_burned: Option<String>,
    pub total_staked: Option<String>,
    pub active_validators: Option<u32>,
    pub transaction_volume: Option<String>,
    pub average_gas_price: Option<String>,
    pub network_utilization: Option<f64>,
    pub token_velocity: Option<f64>,
    pub last_updated: Option<String>,
    pub service_metrics: Option<std::collections::BTreeMap<String, ServiceEconomics>>,
}
#[derive(Clone, Debug, Default, Deserialize, Serialize)]
pub struct GatewayTransaction {
    pub id: Option<String>,
    pub transaction_type: Option<String>,
    pub from: Option<String>,
    pub to: Option<String>,
    pub amount: Option<String>,
    pub purpose: Option<String>,
    pub metadata: Option<Value>,
    pub status: Option<String>,
    pub timestamp: Option<String>,
    pub confirmed_at: Option<String>,
    pub block_height: Option<u64>,
    pub gas_used: Option<u64>,
}
#[derive(Clone, Debug, Default, Deserialize, Serialize)]
pub struct BurnEvent {
    pub tx_id: Option<String>,
    pub user: Option<String>,
    pub amount: Option<String>,
    pub purpose: Option<String>,
    pub skill_id: Option<String>,
    pub timestamp: Option<String>,
    pub validated: Option<bool>,
}
#[derive(Clone, Debug, Default, Deserialize, Serialize)]
pub struct EconomicRules {
    pub skill_invocation_cost: Option<String>,
    pub llm_registration_fee: Option<String>,
    pub validation_reward: Option<String>,
    pub burn_rates: Option<std::collections::BTreeMap<String, String>>,
    pub minting_rules: Option<MintingRules>,
    pub staking_requirements: Option<StakingRequirements>,
    pub governance_thresholds: Option<GovernanceThresholds>,
}
#[derive(Clone, Debug, Default, Deserialize, Serialize)]
pub struct MintingRules {
    pub max_supply: Option<String>,
    pub inflation_rate: Option<f64>,
    pub validator_rewards: Option<String>,
    pub developer_rewards: Option<String>,
    pub community_rewards: Option<String>,
}
#[derive(Clone, Debug, Default, Deserialize, Serialize)]
pub struct StakingRequirements {
    pub min_validator_stake: Option<String>,
    pub min_developer_stake: Option<String>,
    pub slashing_penalty: Option<f64>,
    pub unbonding_period: Option<u64>,
}
#[derive(Clone, Debug, Default, Deserialize, Serialize)]
pub struct GovernanceThresholds {
    pub proposal_deposit: Option<String>,
    pub voting_threshold: Option<f64>,
    pub quorum_threshold: Option<f64>,
    pub voting_period: Option<u64>,
}

#[derive(Clone, Debug, Default, Deserialize, Serialize)]
pub struct PoAuDStatus {
    pub enabled: Option<bool>,
    pub network_authors_count: Option<u32>,
    pub main_pool_size: Option<u32>,
    pub pas_pool_size: Option<u32>,
    pub delegated_transactions: Option<u64>,
    pub delegation_stats: Option<Value>,
}
#[derive(Clone, Debug, Default, Deserialize, Serialize)]
pub struct NetworkAuthor {
    pub address: Option<String>,
}
#[derive(Clone, Debug, Default, Deserialize, Serialize)]
pub struct NetworkAuthorsResponse {
    pub network_authors: Option<Vec<String>>,
    pub count: Option<u32>,
}
#[derive(Clone, Debug, Default, Deserialize, Serialize)]
pub struct PoAuDResponse {
    pub success: Option<bool>,
    pub enabled: Option<bool>,
    pub message: Option<String>,
    pub address: Option<String>,
}
#[derive(Clone, Debug, Default, Deserialize, Serialize)]
pub struct PoAuDProof {
    pub id: Option<String>,
    pub user_id: Option<String>,
    pub skill_id: Option<String>,
    pub verified: Option<bool>,
    pub proof_hash: Option<String>,
    pub proof_data: Option<Value>,
    pub confidence: Option<f64>,
    pub evidence: Option<Value>,
    pub status: Option<String>,
    pub created_at: Option<String>,
    pub verified_at: Option<String>,
}
#[derive(Clone, Debug, Default, Deserialize, Serialize)]
pub struct PoAuDChallenge {
    pub id: Option<String>,
    pub title: Option<String>,
    pub description: Option<String>,
    pub difficulty: Option<String>,
    pub status: Option<String>,
    pub reward: Option<f64>,
    pub created_at: Option<String>,
}
#[derive(Clone, Debug, Default, Deserialize, Serialize)]
pub struct PoAuDUserReputation {
    pub user_id: Option<String>,
    pub reputation_score: Option<i32>,
    pub rank: Option<u32>,
    pub total_proofs: Option<u32>,
    pub verified_proofs: Option<u32>,
    pub success_rate: Option<f64>,
    pub badges: Option<Vec<String>>,
    pub skill_ratings: Option<Value>,
    pub last_updated: Option<String>,
}
#[derive(Clone, Debug, Default, Deserialize, Serialize)]
pub struct PoAuDVerificationResult {
    pub verification_id: Option<String>,
    pub status: Option<String>,
    pub confidence: Option<f64>,
    pub timestamp: Option<i64>,
    pub proof_id: Option<String>,
    pub evidence: Option<Value>,
}
#[derive(Clone, Debug, Default, Deserialize, Serialize)]
pub struct PoAuDSubmissionResult {
    pub submission_id: Option<String>,
    pub challenge_id: Option<String>,
    pub user_id: Option<String>,
    pub solution: Option<Value>,
    pub status: Option<String>,
    pub score: Option<Value>,
    pub estimated_review_time: Option<String>,
    pub submitted_at: Option<String>,
}

#[derive(Clone, Debug, Default, Deserialize, Serialize)]
pub struct BlockSubmitParams {
    pub block_number: Option<u64>,
    pub hash: Option<String>,
    pub nonce: Option<u64>,
    pub prev_hash: Option<String>,
    pub proposer_address: Option<String>,
    pub timestamp: Option<u64>,
    pub transactions: Option<Vec<Transaction>>,
}
#[derive(Clone, Debug, Default, Deserialize, Serialize)]
pub struct TransactionSubmitParams {
    pub id: Option<String>,
    pub fee: Option<String>,
    pub from: Option<String>,
    pub public_key: Option<String>,
    pub signature: Option<String>,
    pub timestamp: Option<u64>,
    #[serde(rename = "type")]
    pub transaction_type: Option<String>,
    pub version: Option<u32>,
    pub chain_id: Option<String>,
    pub account_number: Option<String>,
    pub data: Option<Value>,
    pub to: Option<String>,
    pub value: Option<String>,
}
#[derive(Clone, Debug, Default, Deserialize, Serialize)]
pub struct UriGeneratorCreateParams {
    pub content_hash: Option<String>,
    pub metadata: Option<Value>,
    pub owner: Option<String>,
    pub resource_type: Option<String>,
}
#[derive(Clone, Debug, Default, Deserialize, Serialize)]
pub struct ContextRecord {
    pub id: Option<String>,
    pub capability_id: Option<String>,
    pub initiator: Option<String>,
    pub interaction_type: Option<String>,
    pub status: Option<String>,
    pub signature: Option<String>,
    pub created_at: Option<String>,
    pub updated_at: Option<String>,
    pub metadata: Option<Value>,
}
#[derive(Clone, Debug, Default, Deserialize, Serialize)]
pub struct CapabilityDescriptorUnion {
    pub id: Option<String>,
    pub capability_type: Option<String>,
    pub custom_metadata: Option<Value>,
    pub description: Option<String>,
    pub gas_fee_nrn: Option<i64>,
    pub name: Option<String>,
    pub owner: Option<String>,
    pub timestamp: Option<i64>,
    pub version: Option<String>,
    pub content_hash: Option<String>,
    pub resource_type: Option<String>,
    pub schema: Option<Value>,
    pub execution_pointer: Option<String>,
    pub input_schema_json: Option<String>,
    pub output_schema_json: Option<String>,
    pub parameters_schema_json: Option<String>,
    pub template: Option<String>,
    pub graph_schema: Option<String>,
}

#[derive(Clone, Debug, Default, Deserialize, Serialize)]
pub struct Badge {
    pub id: Option<String>,
    pub badge_type: Option<String>,
    pub name: Option<String>,
    pub description: Option<String>,
    pub criteria: Option<String>,
    pub issuer: Option<String>,
    pub issued_at: Option<String>,
    pub expires_at: Option<String>,
    pub metadata: Option<Value>,
    pub verified: Option<bool>,
}
#[derive(Clone, Debug, Default, Deserialize, Serialize)]
pub struct SkillBadge {
    pub id: Option<String>,
    pub badge_type: Option<String>,
    pub name: Option<String>,
    pub description: Option<String>,
    pub criteria: Option<String>,
    pub issuer: Option<String>,
    pub issued_at: Option<String>,
    pub expires_at: Option<String>,
    pub metadata: Option<Value>,
    pub verified: Option<bool>,
    pub skill_id: Option<String>,
    pub proficiency_level: Option<String>,
    pub validation_proofs: Option<Vec<String>>,
}
#[derive(Clone, Debug, Default, Deserialize, Serialize)]
pub struct CapabilityBadge {
    pub id: Option<String>,
    pub badge_type: Option<String>,
    pub name: Option<String>,
    pub description: Option<String>,
    pub criteria: Option<String>,
    pub issuer: Option<String>,
    pub issued_at: Option<String>,
    pub expires_at: Option<String>,
    pub metadata: Option<Value>,
    pub verified: Option<bool>,
    pub capability_type: Option<String>,
    pub permissions: Option<Vec<String>>,
    pub scope: Option<Vec<String>>,
}
#[derive(Clone, Debug, Default, Deserialize, Serialize)]
pub struct PropertyBadge {
    pub id: Option<String>,
    pub badge_type: Option<String>,
    pub name: Option<String>,
    pub description: Option<String>,
    pub criteria: Option<String>,
    pub issuer: Option<String>,
    pub issued_at: Option<String>,
    pub expires_at: Option<String>,
    pub metadata: Option<Value>,
    pub verified: Option<bool>,
    pub property_type: Option<String>,
    pub value: Option<String>,
    pub attestations: Option<Vec<String>>,
}

#[derive(Clone, Debug, Default, Deserialize, Serialize)]
pub struct DVEEnvironment {
    pub id: Option<String>,
    pub name: Option<String>,
    pub environment_type: Option<String>,
    pub status: Option<String>,
    pub resources: Option<Value>,
    pub endpoints: Option<Value>,
    pub created_at: Option<String>,
    pub expires_at: Option<String>,
}
#[derive(Clone, Debug, Default, Deserialize, Serialize)]
pub struct DVESession {
    pub id: Option<String>,
    pub environment_id: Option<String>,
    pub user_id: Option<String>,
    pub status: Option<String>,
    pub started_at: Option<String>,
    pub last_activity: Option<String>,
    pub resources: Option<Value>,
}

#[derive(Clone, Debug, Default, Deserialize, Serialize)]
pub struct TreasuryOperation {
    pub id: Option<String>,
    pub operation_type: Option<String>,
    pub amount: Option<String>,
    pub from_address: Option<String>,
    pub to_address: Option<String>,
    pub reason: Option<String>,
    pub timestamp: Option<String>,
    pub tx_hash: Option<String>,
    pub status: Option<String>,
}
#[derive(Clone, Debug, Default, Deserialize, Serialize)]
pub struct FaucetRequest {
    pub id: Option<String>,
    pub user_address: Option<String>,
    pub amount: Option<String>,
    pub status: Option<String>,
    pub requested_at: Option<String>,
    pub processed_at: Option<String>,
    pub tx_hash: Option<String>,
}
#[derive(Clone, Debug, Default, Deserialize, Serialize)]
pub struct NRNToken {
    pub symbol: Option<String>,
    pub decimals: Option<u32>,
    pub total_supply: Option<String>,
    pub circulating_supply: Option<String>,
    pub minting_rate: Option<String>,
    pub treasury_balance: Option<String>,
}

#[derive(Clone, Debug, Default, Deserialize, Serialize)]
pub struct Agent {
    pub id: Option<String>,
    pub name: Option<String>,
    pub agent_type: Option<String>,
    pub status: Option<String>,
    pub capabilities: Option<Vec<String>>,
    pub badges: Option<Vec<Badge>>,
    pub owner: Option<String>,
    pub created_at: Option<String>,
    pub last_activity: Option<String>,
    pub configuration: Option<Value>,
}
#[derive(Clone, Debug, Default, Deserialize, Serialize)]
pub struct AgentWorkflow {
    pub id: Option<String>,
    pub agent_id: Option<String>,
    pub name: Option<String>,
    pub description: Option<String>,
    pub steps: Option<Vec<WorkflowStep>>,
    pub status: Option<String>,
    pub created_at: Option<String>,
    pub updated_at: Option<String>,
}
#[derive(Clone, Debug, Default, Deserialize, Serialize)]
pub struct WorkflowStep {
    pub id: Option<String>,
    pub step_type: Option<String>,
    pub name: Option<String>,
    pub configuration: Option<Value>,
    pub dependencies: Option<Vec<String>>,
    pub timeout: Option<u64>,
}
#[derive(Clone, Debug, Default, Deserialize, Serialize)]
pub struct SkillDefinition {
    pub id: Option<String>,
    pub name: Option<String>,
    pub description: Option<String>,
    pub category: Option<String>,
    pub cost: Option<String>,
    pub provider: Option<String>,
    pub metadata: Option<Value>,
    pub badges: Option<Vec<SkillBadge>>,
    pub capabilities: Option<Vec<String>>,
}
#[derive(Clone, Debug, Default, Deserialize, Serialize)]
pub struct SkillInvocation {
    pub id: Option<String>,
    pub skill_id: Option<String>,
    pub user_id: Option<String>,
    pub agent_id: Option<String>,
    pub amount: Option<String>,
    pub status: Option<String>,
    pub result: Option<Value>,
    pub timestamp: Option<String>,
    pub nrn_cost: Option<String>,
    pub gasless: Option<bool>,
}

#[derive(Clone, Debug, Default, Deserialize, Serialize)]
pub struct ConnectivityProof {
    pub id: Option<String>,
    pub agent_id: Option<String>,
    pub proof_type: Option<String>,
    pub proof_data: Option<Value>,
    pub timestamp: Option<String>,
    pub verified: Option<bool>,
    pub nrn_reward: Option<String>,
}
#[derive(Clone, Debug, Default, Deserialize, Serialize)]
pub struct NetworkRoute {
    pub id: Option<String>,
    pub source: Option<String>,
    pub destination: Option<String>,
    pub latency: Option<u64>,
    pub bandwidth: Option<u64>,
    pub reliability: Option<f64>,
    pub cost: Option<String>,
    pub last_updated: Option<String>,
}

#[derive(Clone, Debug, Default, Deserialize, Serialize)]
pub struct FactualitySlice {
    pub id: Option<String>,
    pub slice_type: Option<String>,
    pub status: Option<String>,
    pub configuration: Option<Value>,
    pub network_integration: Option<Value>,
    pub initialized_at: Option<String>,
    pub last_health_check: Option<String>,
}
#[derive(Clone, Debug, Default, Deserialize, Serialize)]
pub struct FactualityVerification {
    pub id: Option<String>,
    pub content: Option<String>,
    pub domain: Option<String>,
    pub confidence: Option<String>,
    pub score: Option<f64>,
    pub sources: Option<Vec<String>>,
    pub timestamp: Option<String>,
    pub cached: Option<bool>,
}

#[derive(Clone, Debug, Default, Deserialize, Serialize)]
pub struct ServiceHealth {
    pub service: Option<String>,
    pub status: Option<String>,
    pub uptime: Option<u64>,
    pub last_check: Option<String>,
    pub response_time: Option<String>,
    pub details: Option<Value>,
}
#[derive(Clone, Debug, Default, Deserialize, Serialize)]
pub struct NetworkHealth {
    pub overall: Option<String>,
    pub services: Option<Value>,
    pub timestamp: Option<String>,
    pub summary: Option<Value>,
}

#[derive(Clone, Debug, Default, Deserialize, Serialize)]
pub struct SkillInvokeRequest {
    pub agent_id: String,
    pub skill_id: String,
    pub user_id: String,
    pub amount: String,
    pub parameters: Option<Value>,
}

#[derive(Clone, Debug, Deserialize, Serialize)]
pub struct KNIRVNetworkInfo {
    pub chain_id: Option<String>,
    pub network_name: Option<String>,
    pub rpc_url: Option<String>,
    pub currency: Option<Value>,
    pub environment: Option<String>,
    pub services: Option<Value>,
}
