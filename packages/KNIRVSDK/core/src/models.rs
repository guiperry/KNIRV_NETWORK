//! The KNIRVMODELS wire contract's logic: the byte-v1 tokenizer, the supervisor
//! advice prompt (with secret redaction) and mapping, and the platform "who does
//! what" role lines. Go and TypeScript used to keep twin copies held together by
//! `testvectors/models/`; those vectors now gate this one implementation.
//!
//! Deterministic: the clock is an input (`now`, RFC 3339), and the redaction
//! patterns are matched by hand (no `regex` dependency) with the semantics of the
//! Go `regexp` patterns they replace.

use serde::{Deserialize, Serialize};
use std::collections::BTreeMap;

/// Go marshals nil slices and maps as `null`; treat that as the empty default.
fn null_default<'de, D, T>(d: D) -> Result<T, D::Error>
where
    D: serde::Deserializer<'de>,
    T: Default + Deserialize<'de>,
{
    Ok(Option::<T>::deserialize(d)?.unwrap_or_default())
}

pub const BYTE_V1_EOS: u32 = 259;
pub const BYTE_V1_PAD: u32 = 256;
pub const BYTE_V1_BOS: u32 = 257;
pub const BYTE_V1_SEP: u32 = 258;
pub const BYTE_V1_OPT_OPEN: u32 = 260;
pub const BYTE_V1_OPT_END: u32 = 261;
pub const BYTE_V1_VOCAB: usize = 262;

/// BOS + the text's bytes, truncated or PAD-filled to `width`.
pub fn encode_byte_v1(text: &str, width: usize) -> Vec<u32> {
    let mut tokens = Vec::with_capacity(width);
    tokens.push(BYTE_V1_BOS);
    for byte in text.bytes() {
        if tokens.len() == width {
            break;
        }
        tokens.push(byte as u32);
    }
    while tokens.len() < width {
        tokens.push(BYTE_V1_PAD);
    }
    tokens
}

/// Choice-set HEAD input after the model-owned summary slot:
/// `BOS question SEP state SEP OPT_OPEN choice OPT_END ...`.
/// It intentionally accepts only neutral question/state strings; label fields
/// such as `intent` and `risk` are never serialized into this prompt.
pub fn encode_choice_set_v1(
    question: &str,
    state: &str,
    choices: &[String],
    width: usize,
) -> Vec<u32> {
    let mut tokens = Vec::with_capacity(width);
    tokens.push(BYTE_V1_BOS);
    tokens.extend(question.bytes().map(u32::from));
    tokens.push(BYTE_V1_SEP);
    tokens.extend(state.bytes().map(u32::from));
    tokens.push(BYTE_V1_SEP);
    for choice in choices {
        tokens.push(BYTE_V1_OPT_OPEN);
        tokens.extend(choice.bytes().map(u32::from));
        tokens.push(BYTE_V1_OPT_END);
    }
    tokens.truncate(width);
    tokens.resize(width, BYTE_V1_PAD);
    tokens
}

/// Plan text as raw bytes: tokens up to the first EOS, specials dropped.
pub fn plan_bytes(plan_tokens: &[u32]) -> Vec<u8> {
    let mut out = Vec::new();
    for &t in plan_tokens {
        if t == BYTE_V1_EOS {
            break;
        }
        if t < 256 {
            out.push(t as u8);
        }
    }
    out
}

#[derive(Debug, Default, Deserialize)]
struct ManifestConfig {
    #[serde(default)]
    max_seq_len: i64,
    #[serde(default)]
    summary_slots: i64,
}

/// Tokens the model takes: its sequence length minus the summary slots.
pub fn input_width(config: &serde_json::Value) -> Result<usize, String> {
    let c: ManifestConfig = serde_json::from_value(config.clone())
        .map_err(|e| format!("model config is not usable: {e}"))?;
    if c.max_seq_len <= c.summary_slots {
        return Err("model config has no input room".into());
    }
    Ok((c.max_seq_len - c.summary_slots) as usize)
}

// ---- advice ----------------------------------------------------------------

#[derive(Debug, Clone, PartialEq, Serialize, Deserialize, Default)]
pub struct Decision {
    #[serde(default, deserialize_with = "null_default")]
    pub name: String,
    #[serde(default, deserialize_with = "null_default")]
    pub kind: String,
    #[serde(default)]
    pub selected: Option<String>,
    #[serde(default)]
    pub confidence: Option<f32>,
    #[serde(default, deserialize_with = "null_default")]
    pub abstain: bool,
}

#[derive(Debug, Default, Deserialize)]
pub struct Reasoning {
    #[serde(default, deserialize_with = "null_default")]
    pub decisions: Vec<Decision>,
    #[serde(default, deserialize_with = "null_default")]
    pub model_version: String,
    #[serde(default, deserialize_with = "null_default")]
    pub dataset_manifest_hash: String,
    #[serde(default, deserialize_with = "null_default")]
    pub policy_bundle_hash: String,
}

/// Decision envelope: the output of the decision core (or a caller-supplied
/// decision). Mirrors the Go `DecisionEnvelope` and the TS `DecisionEnvelope`.
#[derive(Debug, Clone, Serialize, Deserialize, Default, PartialEq)]
pub struct DecisionEnvelope {
    #[serde(default)]
    pub decisions: Vec<Decision>,
    #[serde(default)]
    pub abstained: bool,
    #[serde(default)]
    pub reason: Option<String>,
}

/// HEART generate request. Plain generation (no `decision` and `with_decision`
/// false) never consults the decision core, so legacy clients keep working when
/// the core is unavailable. Set `with_decision` to have HEART resolve the core.
#[derive(Debug, Clone, Serialize, Deserialize, Default)]
pub struct HeartGenerateRequest {
    #[serde(default)]
    pub prompt: String,
    #[serde(default)]
    pub decision: Option<DecisionEnvelope>,
    #[serde(default)]
    pub with_decision: bool,
}

/// HEART generate response: always carries the (possibly absent or abstaining)
/// decision metadata alongside the text.
#[derive(Debug, Clone, Serialize, Deserialize, Default)]
pub struct HeartGenerateResponse {
    #[serde(default)]
    pub text: String,
    #[serde(default)]
    pub ready: bool,
    #[serde(default)]
    pub progress: f64,
    #[serde(default)]
    pub decision: Option<DecisionEnvelope>,
    #[serde(default)]
    pub abstained: bool,
    #[serde(default)]
    pub error: Option<String>,
}

/// Route the decision-aware HEART generate contract is served at.
pub const HEART_GENERATE_PATH: &str = "/models/heart/generate";

#[derive(Debug, Default, Deserialize)]
pub struct EvidenceItem {
    #[serde(default, deserialize_with = "null_default")]
    pub class: String,
}

#[derive(Debug, Default, Deserialize)]
pub struct TaskSummary {
    #[serde(default, deserialize_with = "null_default")]
    pub request: String,
    #[serde(default, deserialize_with = "null_default")]
    pub intent: String,
    #[serde(default, deserialize_with = "null_default")]
    pub risk: String,
}

/// The CLI's request body for `/api/supervision/advise`.
#[derive(Debug, Default, Deserialize)]
pub struct AdviseContext {
    #[serde(default, deserialize_with = "null_default")]
    pub current_phase: String,
    #[serde(default, deserialize_with = "null_default")]
    pub remaining_budget: BTreeMap<String, i64>,
    #[serde(default, deserialize_with = "null_default")]
    pub agent_chain: Vec<String>,
    #[serde(default, deserialize_with = "null_default")]
    pub policy_bundle_hash: String,
    #[serde(default, deserialize_with = "null_default")]
    pub evidence_summary: Vec<EvidenceItem>,
    #[serde(default, deserialize_with = "null_default")]
    pub task_summary: TaskSummary,
}

#[derive(Debug, Default, Serialize, PartialEq)]
pub struct SupervisorAdvice {
    pub recommended_action: String,
    pub next_phase: String,
    pub confidence: f32,
    pub abstain: bool,
    #[serde(skip_serializing_if = "String::is_empty")]
    pub abstain_reason: String,
    pub model_version: String,
    pub dataset_manifest_hash: String,
    pub policy_bundle_hash: String,
    pub received_at: String,
}

/// The advisory vocabulary. "abstain" is deliberately absent: the model never outputs it.
const ADVICE_ACTIONS: [&str; 5] = [
    "allow",
    "request_review",
    "request_evidence",
    "ask_operator",
    "recover",
];

/// ASCII whitespace as RE2's `\s`.
fn is_space(c: char) -> bool {
    matches!(c, ' ' | '\t' | '\n' | '\x0c' | '\r')
}

/// `(?i)` letter match with Go's simple case folding (including the two non-ASCII
/// characters that fold onto ASCII letters: U+212A -> k, U+017F -> s).
fn eq_fold(c: char, lower_ascii: char) -> bool {
    let c = match c {
        '\u{212A}' => 'k',
        '\u{017F}' => 's',
        other => other,
    };
    c.to_lowercase().next() == Some(lower_ascii)
}

fn starts_fold(chars: &[char], at: usize, word: &str) -> Option<usize> {
    let mut i = at;
    for w in word.chars() {
        if i >= chars.len() || !eq_fold(chars[i], w) {
            return None;
        }
        i += 1;
    }
    Some(i)
}

/// End of a case-insensitive `password|passwd|token|secret|api[_-]?key` at `at`.
fn keyword_end(chars: &[char], at: usize) -> Option<usize> {
    for word in ["password", "passwd", "token", "secret"] {
        if let Some(end) = starts_fold(chars, at, word) {
            return Some(end);
        }
    }
    let after_api = starts_fold(chars, at, "api")?;
    let mut i = after_api;
    if i < chars.len() && (chars[i] == '_' || chars[i] == '-') {
        i += 1;
    }
    starts_fold(chars, i, "key")
}

/// `(?i)(?:password|...)\s*[=:]\s*[^\s]{8,}` starting exactly at `at`; returns the match end.
fn match_credential(chars: &[char], at: usize) -> Option<usize> {
    let mut i = keyword_end(chars, at)?;
    while i < chars.len() && is_space(chars[i]) {
        i += 1;
    }
    if i >= chars.len() || (chars[i] != '=' && chars[i] != ':') {
        return None;
    }
    i += 1;
    while i < chars.len() && is_space(chars[i]) {
        i += 1;
    }
    let start = i;
    while i < chars.len() && !is_space(chars[i]) {
        i += 1;
    }
    (i - start >= 8).then_some(i)
}

/// `AKIA[0-9A-Z]{16}` at `at`.
fn match_aws_key(chars: &[char], at: usize) -> Option<usize> {
    if chars.get(at..at + 4)? != ['A', 'K', 'I', 'A'] {
        return None;
    }
    let tail = chars.get(at + 4..at + 20)?;
    tail.iter()
        .all(|c| matches!(c, '0'..='9' | 'A'..='Z'))
        .then_some(at + 20)
}

/// `gh[pousr]_[A-Za-z0-9]{36}` at `at`.
fn match_github_token(chars: &[char], at: usize) -> Option<usize> {
    if *chars.get(at)? != 'g'
        || *chars.get(at + 1)? != 'h'
        || !matches!(chars.get(at + 2)?, 'p' | 'o' | 'u' | 's' | 'r')
        || *chars.get(at + 3)? != '_'
    {
        return None;
    }
    let tail = chars.get(at + 4..at + 40)?;
    tail.iter()
        .all(|c| c.is_ascii_alphanumeric())
        .then_some(at + 40)
}

/// Replaces every leftmost non-overlapping match of one pattern.
fn redact_with(input: &str, matcher: fn(&[char], usize) -> Option<usize>) -> String {
    let chars: Vec<char> = input.chars().collect();
    let mut out = String::with_capacity(input.len());
    let mut i = 0;
    while i < chars.len() {
        match matcher(&chars, i) {
            Some(end) => {
                out.push_str("[REDACTED]");
                i = end;
            }
            None => {
                out.push(chars[i]);
                i += 1;
            }
        }
    }
    out
}

fn redact(request: &str) -> String {
    let r = redact_with(request, match_aws_key);
    let r = redact_with(&r, match_github_token);
    redact_with(&r, match_credential)
}

/// The canonical model input. It uses only fields a recorded `SupervisionEpisode`
/// also carries, so training and serving see the same text. Deterministic (sorted
/// maps); the request is redacted again here because this is the last boundary
/// before the model.
pub fn build_advise_prompt(c: &AdviseContext) -> String {
    let budget = c
        .remaining_budget
        .iter()
        .map(|(k, v)| format!("{k}={v}"))
        .collect::<Vec<_>>()
        .join(",");
    let mut classes: Vec<&str> = c
        .evidence_summary
        .iter()
        .map(|e| e.class.as_str())
        .collect();
    classes.sort_unstable();
    format!(
        "phase:{} intent:{} risk:{} budget:{} agents:{} evidence:{}\nrequest:{}",
        c.current_phase,
        c.task_summary.intent,
        c.task_summary.risk,
        budget,
        c.agent_chain.join(">"),
        classes.join(","),
        redact(&c.task_summary.request)
    )
}

pub fn abstain_advice(
    reason: &str,
    model_version: &str,
    manifest: &str,
    now: &str,
) -> SupervisorAdvice {
    SupervisorAdvice {
        abstain: true,
        abstain_reason: reason.into(),
        model_version: model_version.into(),
        dataset_manifest_hash: manifest.into(),
        received_at: now.into(),
        ..Default::default()
    }
}

/// Maps decision heads onto `SupervisorAdvice`; every doubtful case abstains.
pub fn advice_from_reasoning(r: &Reasoning, c: &AdviseContext, now: &str) -> SupervisorAdvice {
    let abstain =
        |reason: &str| abstain_advice(reason, &r.model_version, &r.dataset_manifest_hash, now);
    let find = |name: &str| {
        r.decisions
            .iter()
            .find(|d| d.name == name && d.kind == "choice")
    };

    let Some(action) = find("action") else {
        return abstain("model has no action head");
    };
    let selected = match (&action.selected, action.abstain) {
        (Some(s), false) => s,
        _ => return abstain("action confidence below the fitted abstention threshold"),
    };
    if !ADVICE_ACTIONS.contains(&selected.as_str()) {
        return abstain("action is not in the advisory vocabulary");
    }
    let mut confidence = action.confidence.unwrap_or(0.0);

    // No next_phase head means the model offers no transition advice: stay put.
    let mut next = c.current_phase.clone();
    if let Some(np) = find("next_phase") {
        match (&np.selected, np.abstain) {
            (Some(s), false) => next = s.clone(),
            _ => return abstain("next_phase confidence below the fitted abstention threshold"),
        }
        if let Some(nc) = np.confidence {
            if nc < confidence {
                confidence = nc;
            }
        }
    }
    // The bundle the model was trained under; a mismatch makes the CLI discard the advice.
    let bundle = if r.policy_bundle_hash.is_empty() {
        c.policy_bundle_hash.clone()
    } else {
        r.policy_bundle_hash.clone()
    };
    SupervisorAdvice {
        recommended_action: selected.clone(),
        next_phase: next,
        confidence,
        abstain: false,
        abstain_reason: String::new(),
        model_version: r.model_version.clone(),
        dataset_manifest_hash: r.dataset_manifest_hash.clone(),
        policy_bundle_hash: bundle,
        received_at: now.into(),
    }
}

// ---- platform status -------------------------------------------------------

#[derive(Debug, Default, Deserialize)]
pub struct ModelStatus {
    #[serde(default, deserialize_with = "null_default")]
    pub name: String,
    #[serde(default, deserialize_with = "null_default")]
    pub version: String,
    #[serde(default, deserialize_with = "null_default")]
    pub ready: bool,
    #[serde(default, deserialize_with = "null_default")]
    pub loaded: bool,
    #[serde(default, deserialize_with = "null_default")]
    pub error: String,
}

#[derive(Debug, Default, Deserialize)]
pub struct LlamaAdapter {
    #[serde(default, deserialize_with = "null_default")]
    pub name: String,
    #[serde(default, deserialize_with = "null_default")]
    pub loaded: bool,
}

#[derive(Debug, Default, Deserialize)]
pub struct LlamaStatus {
    #[serde(default, deserialize_with = "null_default")]
    pub ready: bool,
    #[serde(default, deserialize_with = "null_default")]
    pub base_sha256: String,
    #[serde(default, deserialize_with = "null_default")]
    pub adapters: Vec<LlamaAdapter>,
    #[serde(default, deserialize_with = "null_default")]
    pub error: String,
}

#[derive(Debug, Default, Deserialize)]
pub struct RouteStats {
    #[serde(default, deserialize_with = "null_default")]
    pub circuit: String,
    #[serde(default, deserialize_with = "null_default")]
    pub last_error: String,
}

#[derive(Debug, Default, Deserialize)]
pub struct PlatformStatus {
    #[serde(default, deserialize_with = "null_default")]
    pub daemon: Vec<ModelStatus>,
    #[serde(default, deserialize_with = "null_default")]
    pub daemon_error: String,
    #[serde(default)]
    pub llama: Option<LlamaStatus>,
    #[serde(default, deserialize_with = "null_default")]
    pub stats: BTreeMap<String, RouteStats>,
}

/// One row of "which model does what": role -> model -> version.
#[derive(Debug, Default, Serialize, PartialEq)]
pub struct RoleLine {
    pub role: String,
    pub label: String,
    pub model: String,
    pub version: String,
    pub state: String,
    #[serde(skip_serializing_if = "String::is_empty")]
    pub detail: String,
}

fn sha12(s: &str) -> String {
    s.chars().take(12).collect()
}

// ===== verify (Task 3.4) =====
//
// The HASHER verify contract's invariants live here once. The Go and TypeScript
// bindings call `models.verify_check` / `models.verify_request_check`; they keep
// only the wire types and the HTTP.

const GATE_NAMES: [&str; 7] = [
    "refused",
    "discouraged",
    "uncommitted",
    "seen",
    "attested",
    "accepted",
    "committed",
];

#[derive(Debug, Default, Deserialize)]
struct VerifyEvidenceView {
    #[serde(default)]
    span_id: String,
}

#[derive(Debug, Deserialize)]
struct VerifyClaimView {
    #[serde(default)]
    verdict: String,
    #[serde(default)]
    gate: String,
    gate_level: i64,
    #[serde(default, deserialize_with = "null_default")]
    evidence: Vec<VerifyEvidenceView>,
    #[serde(default, deserialize_with = "null_default")]
    cited: Vec<String>,
    #[serde(default)]
    degraded: bool,
}

/// The fields of a verify response the invariants read.
#[derive(Debug, Deserialize)]
pub struct VerifyResponseView {
    #[serde(default)]
    status: String,
    #[serde(default)]
    job_id: String,
    #[serde(default)]
    gate: String,
    gate_level: i64,
    gate_score: f64,
    confidence: f64,
    #[serde(default, deserialize_with = "null_default")]
    claims: Vec<VerifyClaimView>,
    #[serde(default)]
    degraded: bool,
    #[serde(default)]
    chargeable: bool,
    #[serde(default)]
    unverified: bool,
    #[serde(default)]
    mode: String,
    #[serde(default)]
    flash_used: bool,
    #[serde(default, deserialize_with = "null_default")]
    outbound_queries: Vec<String>,
    refused: bool,
    is_accurate: bool,
    #[serde(default, deserialize_with = "null_default")]
    citations: Vec<i64>,
}

fn gate_name(level: i64) -> Option<&'static str> {
    usize::try_from(level).ok().and_then(|i| GATE_NAMES.get(i).copied())
}

/// The first way a verify response breaks the contract's invariants, or `None`.
pub fn check_verify(r: &VerifyResponseView) -> Option<String> {
    if gate_name(r.gate_level) != Some(r.gate.as_str()) {
        return Some(format!("gate {:?} does not match gate_level {}", r.gate, r.gate_level));
    }
    if r.refused != (r.gate_level == 0) {
        return Some(format!("refused={} but gate_level is {}", r.refused, r.gate_level));
    }
    if r.is_accurate != (r.gate_level >= 5) {
        return Some(format!("is_accurate={} but gate_level is {}", r.is_accurate, r.gate_level));
    }
    if r.confidence != r.gate_score {
        return Some(format!("confidence {} must equal gate_score {}", r.confidence, r.gate_score));
    }
    match r.status.as_str() {
        "pending" => {
            if r.gate_level != 2 || !r.unverified || r.chargeable || r.job_id.is_empty() {
                return Some("a pending answer must be uncommitted, unverified, not chargeable and carry a job id".into());
            }
        }
        "done" => {}
        other => return Some(format!("status {other:?} is neither done nor pending")),
    }
    if r.mode == "index" && (r.flash_used || !r.outbound_queries.is_empty()) {
        return Some("an index-mode answer cannot report live lookup".into());
    }
    if r.mode != "index" && r.flash_used && r.outbound_queries.is_empty() {
        return Some("flash_used without the outbound queries that prove it".into());
    }
    if r.degraded && r.chargeable {
        return Some("a degraded answer must not be chargeable".into());
    }
    let mut spans: Vec<&str> = Vec::new();
    for (i, c) in r.claims.iter().enumerate() {
        if gate_name(c.gate_level) != Some(c.gate.as_str()) {
            return Some(format!("claim {i}: gate {:?} does not match gate_level {}", c.gate, c.gate_level));
        }
        if !matches!(c.verdict.as_str(), "SUPPORTED" | "REFUTED" | "NEI") {
            return Some(format!("claim {i}: verdict {:?}", c.verdict));
        }
        if c.gate_level >= 3 && c.verdict != "SUPPORTED" {
            return Some(format!("claim {i}: gate {} without a SUPPORTED verdict", c.gate));
        }
        if c.degraded && c.gate_level > 2 {
            return Some(format!("claim {i}: a degraded claim reached {}", c.gate));
        }
        for id in &c.cited {
            if !c.evidence.iter().any(|e| &e.span_id == id) {
                return Some(format!("claim {i} cites {id}, which is not in its evidence"));
            }
        }
        if c.gate_level == 0 && !c.evidence.is_empty() && c.verdict == "SUPPORTED" {
            return Some(format!("claim {i}: refused but SUPPORTED"));
        }
        spans.extend(c.evidence.iter().map(|e| e.span_id.as_str()));
    }
    for &p in &r.citations {
        let span = match usize::try_from(p).ok().and_then(|i| spans.get(i)) {
            Some(span) => *span,
            None => return Some(format!("citation {p} is out of range")),
        };
        if !r.claims.iter().any(|c| c.cited.iter().any(|id| id == span)) {
            return Some(format!("citation {p} points at {span}, which no claim cites"));
        }
    }
    None
}

#[derive(Debug, Default, Deserialize)]
pub struct VerifyRequestView {
    #[serde(default)]
    text: String,
    #[serde(default, deserialize_with = "null_default")]
    claims: Vec<serde_json::Value>,
}

/// A verify request needs a (non-blank) text or at least one claim.
pub fn check_verify_request(r: &VerifyRequestView) -> Option<String> {
    if r.text.trim().is_empty() && r.claims.is_empty() {
        return Some("verify requires text or claims".into());
    }
    None
}

/// The fixed list of role lines, in a fixed order. Anything the document does not
/// mention is reported as unavailable, never omitted.
pub fn roles(s: &PlatformStatus) -> Vec<RoleLine> {
    let daemon = |role: &str, label: &str, name: &str| {
        let mut l = RoleLine {
            role: role.into(),
            label: label.into(),
            model: name.into(),
            ..Default::default()
        };
        match s.daemon.iter().find(|m| m.name == name) {
            None if !s.daemon_error.is_empty() => {
                l.state = "unavailable".into();
                l.detail = s.daemon_error.clone();
            }
            None => {
                l.state = "unavailable".into();
                l.detail = "not served by the models daemon".into();
            }
            Some(m) if m.ready => {
                l.state = "ready".into();
                l.version = m.version.clone();
            }
            Some(m) if !m.loaded && m.error.is_empty() => {
                l.state = "idle".into();
                l.version = m.version.clone();
                l.detail = "not built yet; built on first use".into();
            }
            Some(m) => {
                l.state = "not ready".into();
                l.version = m.version.clone();
                l.detail = if m.error.is_empty() {
                    "not trained yet".into()
                } else {
                    m.error.clone()
                };
            }
        }
        l
    };

    // Phase 4: the server_tasks role is served by the HANDS daemon (the
    // operator-facing GPT that proposes and runs server tasks), not llama. llama
    // is only an explicit fallback when HANDS is not registered with the models
    // daemon. The contract lives here, in the core, never in the bindings.
    let mut server_tasks = daemon("server_tasks", "server tasks and skills", "hands");
    if !s.daemon.iter().any(|m| m.name == "hands") {
        // Fall back to the embedded llama wrapper for the server_tasks line.
        server_tasks.model = "llama".into();
        server_tasks.state = "unavailable".into();
        server_tasks.detail = "llama is not running".into();
        if let Some(status) = &s.llama {
            server_tasks.state = "not ready".into();
            server_tasks.detail = status.error.clone();
            if status.ready {
                server_tasks.state = "ready".into();
                if !status.base_sha256.is_empty() {
                    server_tasks.version = format!("gguf-{}", sha12(&status.base_sha256));
                }
                let loaded: Vec<&str> = status
                    .adapters
                    .iter()
                    .filter(|a| a.loaded)
                    .map(|a| a.name.as_str())
                    .collect();
                server_tasks.detail = if loaded.is_empty() {
                    "no skill adapter loaded".into()
                } else {
                    format!("adapters: {}", loaded.join(", "))
                };
            } else if server_tasks.detail.is_empty() {
                server_tasks.detail = "starting".into();
            }
        }
    }

    let mut external = RoleLine {
        role: "external".into(),
        label: "fallback and planning".into(),
        model: "external".into(),
        version: "provider-chain".into(),
        state: "ready".into(),
        detail: "chosen per request; the only model that leaves this machine".into(),
    };
    if let Some(st) = s.stats.get("external") {
        if !st.circuit.is_empty() && st.circuit != "closed" {
            external.state = "not ready".into();
            external.detail = format!("circuit {}: {}", st.circuit, st.last_error);
        }
    }
    vec![
        // The decision core moved from HEAD to HEART (Phase 2): HEART owns
        // /models/heart/decide and HEAD keeps only a compatibility proxy.
        daemon("decide", "supervision decisions", "heart"),
        daemon("generate", "text generation", "heart"),
        daemon("validate", "validation", "hasher"),
        // Phase 3: HEAD's post-migration role is vision / multimodal and the
        // mirrored text generator.
        daemon("vision", "vision and mirrored text", "head"),
        // Phase 4: HANDS owns server_tasks; llama is the fallback.
        server_tasks,
        external,
    ]
}

#[cfg(test)]
mod tests {
    use super::*;
    use serde_json::json;
    use serde_json::Value;

    fn vector(name: &str) -> Value {
        let path = std::path::Path::new(env!("CARGO_MANIFEST_DIR"))
            .join("../testvectors/models")
            .join(name);
        serde_json::from_slice(&std::fs::read(path).unwrap()).unwrap()
    }

    #[test]
    fn byte_v1_vectors() {
        for case in vector("byte_v1.json").as_array().unwrap() {
            let got = encode_byte_v1(
                case["text"].as_str().unwrap(),
                case["width"].as_u64().unwrap() as usize,
            );
            let want: Vec<u32> = case["tokens"]
                .as_array()
                .unwrap()
                .iter()
                .map(|t| t.as_u64().unwrap() as u32)
                .collect();
            assert_eq!(got, want, "{}", case["text"]);
        }
    }

    #[test]
    fn prompt_vector_and_redaction() {
        let v = vector("prompt_vector.json");
        let ctx: AdviseContext = serde_json::from_value(v["context"].clone()).unwrap();
        assert_eq!(build_advise_prompt(&ctx), v["prompt"].as_str().unwrap());

        let r = vector("prompt_redaction.json");
        let ctx: AdviseContext = serde_json::from_value(r["redaction"]["context"].clone()).unwrap();
        let prompt = build_advise_prompt(&ctx);
        for secret in r["redaction"]["must_not_contain"].as_array().unwrap() {
            assert!(!prompt.contains(secret.as_str().unwrap()), "{prompt}");
        }
        assert_eq!(prompt, build_advise_prompt(&ctx));

        let set = &v["set_prompt"];
        let choices = set["choices"]
            .as_array()
            .unwrap()
            .iter()
            .map(|x| x.as_str().unwrap().to_owned())
            .collect::<Vec<_>>();
        let got = encode_choice_set_v1(
            set["question"].as_str().unwrap(),
            set["state"].as_str().unwrap(),
            &choices,
            set["width"].as_u64().unwrap() as usize,
        );
        let want = set["tokens"]
            .as_array()
            .unwrap()
            .iter()
            .map(|x| x.as_u64().unwrap() as u32)
            .collect::<Vec<_>>();
        assert_eq!(got, want);
    }

    #[test]
    fn redaction_edge_cases_match_the_go_patterns() {
        let cases = [
            ("AKIAABCDEFGHIJKLMNOP", "[REDACTED]"),
            ("AKIAABCDEFGHIJKLMNO", "AKIAABCDEFGHIJKLMNO"), // one short
            ("akiaABCDEFGHIJKLMNOP", "akiaABCDEFGHIJKLMNOP"), // case-sensitive
            ("ghp_0123456789abcdefghijklmnopqrstuvwxyz", "[REDACTED]"),
            (
                "ghx_0123456789abcdefghijklmnopqrstuvwxyz",
                "ghx_0123456789abcdefghijklmnopqrstuvwxyz",
            ),
            ("PASSWORD = hunter2hunter2 end", "[REDACTED] end"),
            ("Api-Key:abcdefgh", "[REDACTED]"),
            ("apikey: short", "apikey: short"), // value under 8 characters
            (
                "token=\u{e9}\u{e9}\u{e9}\u{e9}\u{e9}\u{e9}\u{e9}\u{e9}",
                "[REDACTED]",
            ), // counts characters, not bytes
            (
                "secret : 12345678 and token=abcdefghij",
                "[REDACTED] and [REDACTED]",
            ),
            ("mytoken=12345678", "my[REDACTED]"), // matches inside a word, as the regex does
            ("mytokenx=12345678", "mytokenx=12345678"),
            ("to\u{212A}en=12345678", "[REDACTED]"), // Go folds U+212A onto k
            ("pa\u{017F}\u{017F}word=12345678", "[REDACTED]"), // and U+017F onto s
        ];
        for (input, want) in cases {
            assert_eq!(redact(input), want, "{input:?}");
        }
    }

    #[test]
    fn advice_cases() {
        for case in vector("advice_cases.json").as_array().unwrap() {
            let name = case["name"].as_str().unwrap();
            let r: Reasoning = serde_json::from_value(case["reasoning"].clone()).unwrap();
            let c: AdviseContext = serde_json::from_value(case["context"].clone()).unwrap();
            let a = advice_from_reasoning(&r, &c, "2026-10-31T00:00:00Z");
            let want = &case["expect"];
            assert_eq!(a.abstain, want["abstain"].as_bool().unwrap(), "{name}");
            assert_eq!(a.model_version, r.model_version, "{name}");
            assert_eq!(a.dataset_manifest_hash, r.dataset_manifest_hash, "{name}");
            if a.abstain {
                assert!(
                    !a.abstain_reason.is_empty() && a.recommended_action.is_empty(),
                    "{name}"
                );
                continue;
            }
            assert_eq!(
                a.recommended_action,
                want["recommended_action"].as_str().unwrap(),
                "{name}"
            );
            assert_eq!(a.next_phase, want["next_phase"].as_str().unwrap(), "{name}");
            assert_eq!(
                a.confidence,
                want["confidence"].as_f64().unwrap() as f32,
                "{name}"
            );
            assert_eq!(
                a.policy_bundle_hash,
                want["policy_bundle_hash"].as_str().unwrap(),
                "{name}"
            );
        }
    }

    #[test]
    fn platform_roles_vectors() {
        for (name, case) in vector("platform_status.json").as_object().unwrap() {
            let status: PlatformStatus = serde_json::from_value(case["status"].clone()).unwrap();
            let got = serde_json::to_value(roles(&status)).unwrap();
            assert_eq!(got, case["roles"], "{name}");
        }
    }

    #[test]
    fn decision_route_and_vision_role_are_authoritative() {
        // The decision core lives in HEART (Phase 2) and the post-migration HEAD
        // role is vision (Phase 3). These are the SDK's source-of-truth contract;
        // the Go/TS bindings must mirror them, never own them.
        let status: PlatformStatus = serde_json::from_value(json!({"daemon": [{"name": "heart", "ready": true, "version": "gpt-0.2"}]})).unwrap();
        let roles = roles(&status);
        let decide = roles.iter().find(|r| r.role == "decide").expect("decide role present");
        assert_eq!(decide.model, "heart", "decision core moved to HEART");
        assert!(
            roles.iter().any(|r| r.role == "vision" && r.model == "head"),
            "vision role present and served by HEAD"
        );
    }

    #[test]
    fn input_width_and_plan() {
        let m = vector("head_manifest.json");
        assert_eq!(input_width(&m["config"]).unwrap(), 39);
        assert!(input_width(&serde_json::json!({"max_seq_len": 4, "summary_slots": 4})).is_err());
        assert_eq!(plan_bytes(&[104, 105, 259, 120]), b"hi");
    }

    #[test]
    fn heart_generate_contract_roundtrip() {
        let v: Value = vector("heart_generate.json");
        let req: HeartGenerateRequest =
            serde_json::from_value(v["request"].clone()).unwrap();
        assert_eq!(req.prompt, "restart the failing service");
        assert!(req.with_decision);
        assert!(req.decision.is_none());

        let resp: HeartGenerateResponse =
            serde_json::from_value(v["response"].clone()).unwrap();
        assert_eq!(resp.text, "restarting service");
        assert!(!resp.abstained);
        let env = resp.decision.unwrap();
        assert_eq!(env.decisions[0].name, "action");
        assert_eq!(env.decisions[0].selected.as_deref(), Some("restart-service"));

        // Plain request: no decision, no with_decision -> default fields.
        let plain: HeartGenerateRequest =
            serde_json::from_value(v["plain_request"].clone()).unwrap();
        assert!(!plain.with_decision);
        assert!(plain.decision.is_none());
        assert_eq!(plain.prompt, "hello");

        // The path constant must match the server route.
        assert_eq!(HEART_GENERATE_PATH, "/models/heart/generate");
    }
}

#[cfg(test)]
mod null_tests {
    use super::*;
    use serde_json::json;

    #[test]
    fn go_nulls_read_as_empty() {
        let c: AdviseContext = serde_json::from_value(json!({
            "current_phase": "p", "remaining_budget": null, "agent_chain": null,
            "evidence_summary": null, "task_summary": {"request": "r", "intent": "", "risk": ""}
        }))
        .unwrap();
        assert!(build_advise_prompt(&c)
            .starts_with("phase:p intent: risk: budget: agents: evidence:\nrequest:r"));
        let r: Reasoning =
            serde_json::from_value(json!({"decisions": null, "model_version": "v"})).unwrap();
        assert!(advice_from_reasoning(&r, &c, "t").abstain);
        let s: PlatformStatus =
            serde_json::from_value(json!({"daemon": null, "stats": null, "llama": null})).unwrap();
        assert_eq!(roles(&s).len(), 6);
    }
}
