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

/// One choice-set candidate with its probability (knirv-heart/v2). The `id` is
/// the option's stable identifier (e.g. an advisory action name); `text` is the
/// human-readable label; `probability` is the set-head's softmax mass for that
/// candidate. Padded/oversized candidates carry zero mass and are simply absent
/// from this list, so variable-N sets need no mask on the consumer side: each
/// question carries exactly its own valid candidates, index-aligned to the
/// request order (the producer's `-inf` masked logits never reach the wire).
#[derive(Debug, Clone, PartialEq, Serialize, Deserialize, Default)]
pub struct Candidate {
    #[serde(default, deserialize_with = "null_default")]
    pub id: String,
    #[serde(default, deserialize_with = "null_default")]
    pub text: String,
    #[serde(default)]
    pub probability: f32,
}

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
    /// Per-candidate probabilities of a choice-set (knirv-heart/v2) answer, in the
    /// request's own order. Absent for v1 decision-head output.
    #[serde(default, skip_serializing_if = "Vec::is_empty")]
    pub candidates: Vec<Candidate>,
    /// For a score-kind head (knirv-heart/v2): the calibrated level value
    /// `Σ p_i·i/(N-1)`. Absent for choice/null heads and v1 output.
    #[serde(default)]
    pub value: Option<f32>,
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

// ---- Clef translation rules (Clef_KNIRV_Design.md §4.2) ---------------------
//
// Carried over verbatim from the KNIRVMODELS supervision package so the SDK's
// model-output → `SupervisorAdvice` converter applies exactly the same rules the
// training/pipeline side uses. They are pure, tested primitives; the converter
// below wires them into the v2 choice-set (knirv-heart/v2) output. The local
// Clef harness under MODELS/pipeline/5_HEAD_TRAINER/clef-harness is the
// calibration baseline (reported ECE 0.63); HEAD must beat it. No hosted Clef
// calls are made from the SDK.

/// `DEFAULT_ABSTAIN_TAU` is the abstention threshold for the Clef rule-3 abstain
/// decision (abstain when the peak probability is below tau). PROVISIONAL: its
/// value must be fitted on a held-out split (Task 4.6) and is not a trained
/// label. Until then every caller uses this one constant so the threshold is
/// consistent and easy to replace.
pub const DEFAULT_ABSTAIN_TAU: f32 = 0.5;

/// `clamp01` clamps a probability to `[0,1]` so a malformed value cannot yield a
/// negative or `>1` confidence.
fn clamp01(x: f32) -> f32 {
    if x < 0.0 {
        0.0
    } else if x > 1.0 {
        1.0
    } else {
        x
    }
}

/// `noul_confidence` derives confidence for a binary (noul) question from
/// P(yes)=`p_yes`. Per Clef rule 2 the confidence is `max(p, 1-p)` — never the
/// raw `p` — and the derivation is explicit. Inputs are clamped to `[0,1]`.
pub fn noul_confidence(p_yes: f32) -> f32 {
    let p = clamp01(p_yes);
    p.max(1.0 - p)
}

/// `noul_probabilities` synthesises the calibrated distribution for a noul answer
/// as `{true: p, false: 1-p}`. A single abstain rule then works across all three
/// Clef question types (noul/choice/score) because each reduces to a peak
/// probability.
pub fn noul_probabilities(p_yes: f32) -> (f32, f32) {
    let p = clamp01(p_yes);
    (p, 1.0 - p)
}

/// `peak_probability` returns the maximum value in `probs` (0 when empty): the
/// confidence the model places on its most likely outcome. It is the quantity the
/// abstain rule thresholds on.
pub fn peak_probability(probs: &[f32]) -> f32 {
    probs.iter().fold(0.0, |m, &p| if p > m { p } else { m })
}

/// `should_abstain` implements Clef rule 3: abstain when the peak probability is
/// below `tau`. Abstention is a deterministic calibration threshold, never a
/// model-chosen label.
pub fn should_abstain(peak: f32, tau: f32) -> bool {
    peak < tau
}

/// `round_severity_for_display` rounds a continuous severity to `decimals`
/// places. Per Clef rule 1 severity is carried as a float internally and rounded
/// only at the display boundary; callers must not round before storing or
/// comparing.
pub fn round_severity_for_display(severity: f32, decimals: i32) -> f32 {
    let d = if decimals < 0 { 0 } else { decimals };
    let scale = 10f64.powi(d as i32);
    (((severity as f64) * scale).round() / scale) as f32
}

/// `resolve_choice` maps a decision head onto a `(selected_id, confidence)` pair
/// using the Clef rules. For a v2 choice-set head (knirv-heart/v2) with
/// `candidates` the confidence is the peak candidate probability and abstention
/// follows rule 3 (peak < tau); the recommended action is the argmax candidate.
/// For a v1 decision head the legacy `selected`/`confidence`/`abstain` fields are
/// used. Returns `None` when the head should abstain.
fn resolve_choice(d: &Decision) -> Option<(String, f32)> {
    // Honor a producer abstention first. The producer derives `abstain` from its
    // validated model configuration (`cfg.tau`, whose default is not the SDK's
    // provisional 0.5), so its signal is authoritative and must win even when the
    // SDK-side peak is above `DEFAULT_ABSTAIN_TAU`. Safe to keep the SDK threshold
    // as a secondary guard below.
    if d.abstain {
        return None;
    }
    if d.kind == "choice" && !d.candidates.is_empty() {
        let peak = d
            .candidates
            .iter()
            .map(|c| c.probability)
            .fold(0.0_f32, |m, p| if p > m { p } else { m });
        if should_abstain(peak, DEFAULT_ABSTAIN_TAU) {
            return None;
        }
        let best = d
            .candidates
            .iter()
            .max_by(|a, b| {
                a.probability
                    .partial_cmp(&b.probability)
                    .unwrap_or(std::cmp::Ordering::Equal)
            })
            .expect("candidates is non-empty");
        return Some((best.id.clone(), peak));
    }
    match (&d.selected, d.abstain) {
        (Some(s), false) => Some((s.clone(), d.confidence.unwrap_or(0.0))),
        _ => None,
    }
}

/// Maps decision heads onto `SupervisorAdvice`; every doubtful case abstains.
/// The v2 choice-set (knirv-heart/v2) action/next_phase heads are read through
/// the Clef primitives; v1 decision-head output is still supported.
pub fn advice_from_reasoning(r: &Reasoning, c: &AdviseContext, now: &str) -> SupervisorAdvice {
    let abstain =
        |reason: &str| abstain_advice(reason, &r.model_version, &r.dataset_manifest_hash, now);
    let find = |name: &str| r.decisions.iter().find(|d| d.name == name && d.kind == "choice");

    let Some(action) = find("action") else {
        return abstain("model has no action head");
    };
    let (selected, conf) = match resolve_choice(action) {
        Some(x) => x,
        None => return abstain("action confidence below the fitted abstention threshold"),
    };
    if !ADVICE_ACTIONS.contains(&selected.as_str()) {
        return abstain("action is not in the advisory vocabulary");
    }
    let mut confidence = conf;

    // No next_phase head means the model offers no transition advice: stay put.
    let mut next = c.current_phase.clone();
    if let Some(np) = find("next_phase") {
        match resolve_choice(np) {
            Some((s, nc)) => {
                next = s;
                if nc < confidence {
                    confidence = nc;
                }
            }
            None => return abstain("next_phase confidence below the fitted abstention threshold"),
        }
    }
    // The bundle the model was trained under; a mismatch makes the CLI discard the advice.
    let bundle = if r.policy_bundle_hash.is_empty() {
        c.policy_bundle_hash.clone()
    } else {
        r.policy_bundle_hash.clone()
    };
    SupervisorAdvice {
        recommended_action: selected,
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
            // Set-request fixtures (knirv-heart/v2) carry no advice; skip them
            // here, matching the Go/TS consumers.
            if case["reasoning"].is_null() {
                continue;
            }
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

#[cfg(test)]
mod clef_tests {
    use super::*;

    fn ctx() -> AdviseContext {
        AdviseContext {
            current_phase: "develop".into(),
            ..Default::default()
        }
    }

    fn reasoning_with(decisions: Vec<Decision>) -> Reasoning {
        Reasoning {
            decisions,
            model_version: "v2-test".into(),
            dataset_manifest_hash: "dm".into(),
            policy_bundle_hash: "sha256:trained".into(),
            ..Default::default()
        }
    }

    #[test]
    fn clef_rules() {
        assert!((noul_confidence(0.2) - 0.8).abs() < 1e-6);
        assert!((noul_confidence(-1.0) - 1.0).abs() < 1e-6);
        assert!((noul_confidence(2.0) - 1.0).abs() < 1e-6);
        let (t, f) = noul_probabilities(0.3);
        assert!((t - 0.3).abs() < 1e-6 && (f - 0.7).abs() < 1e-6);
        assert!((peak_probability(&[0.1, 0.7, 0.2]) - 0.7).abs() < 1e-6);
        assert_eq!(peak_probability(&[]), 0.0);
        assert!(should_abstain(0.4, 0.5));
        assert!(!should_abstain(0.6, 0.5));
        assert!((round_severity_for_display(0.12345, 2) - 0.12).abs() < 1e-6);
        assert!((round_severity_for_display(0.12345, 1) - 0.1).abs() < 1e-6);
        assert_eq!(DEFAULT_ABSTAIN_TAU, 0.5);
    }

    #[test]
    fn v2_action_choice_set_picks_argmax_and_min_confidence() {
        let d_action = Decision {
            name: "action".into(),
            kind: "choice".into(),
            candidates: vec![
                Candidate { id: "allow".into(), text: "allow".into(), probability: 0.7 },
                Candidate { id: "request_review".into(), text: "request review".into(), probability: 0.2 },
                Candidate { id: "ask_operator".into(), text: "ask operator".into(), probability: 0.1 },
            ],
            ..Default::default()
        };
        let d_next = Decision {
            name: "next_phase".into(),
            kind: "choice".into(),
            candidates: vec![
                Candidate { id: "develop_review".into(), text: "develop review".into(), probability: 0.6 },
                Candidate { id: "validate".into(), text: "validate".into(), probability: 0.4 },
            ],
            ..Default::default()
        };
        let a = advice_from_reasoning(&reasoning_with(vec![d_action, d_next]), &ctx(), "now");
        assert!(!a.abstain);
        assert_eq!(a.recommended_action, "allow");
        assert_eq!(a.next_phase, "develop_review");
        assert!((a.confidence - 0.6).abs() < 1e-6);
        assert_eq!(a.policy_bundle_hash, "sha256:trained");
    }

    #[test]
    fn v2_action_below_tau_abstains() {
        let d = Decision {
            name: "action".into(),
            kind: "choice".into(),
            candidates: vec![
                Candidate { id: "allow".into(), text: "allow".into(), probability: 0.3 },
                Candidate { id: "request_review".into(), text: "rr".into(), probability: 0.4 },
            ],
            ..Default::default()
        };
        let a = advice_from_reasoning(&reasoning_with(vec![d]), &ctx(), "now");
        assert!(a.abstain);
        assert!(a.abstain_reason.contains("below the fitted"));
    }

    #[test]
    fn v2_out_of_vocabulary_action_abstains() {
        let d = Decision {
            name: "action".into(),
            kind: "choice".into(),
            candidates: vec![
                Candidate { id: "rm -rf /".into(), text: "x".into(), probability: 0.9 },
            ],
            ..Default::default()
        };
        let a = advice_from_reasoning(&reasoning_with(vec![d]), &ctx(), "now");
        assert!(a.abstain);
        assert!(a.abstain_reason.contains("vocabulary"));
    }

    #[test]
    fn v1_decision_head_fallback_still_works() {
        let d = Decision {
            name: "action".into(),
            kind: "choice".into(),
            selected: Some("allow".into()),
            confidence: Some(0.95),
            ..Default::default()
        };
        let a = advice_from_reasoning(&reasoning_with(vec![d]), &ctx(), "now");
        assert!(!a.abstain);
        assert_eq!(a.recommended_action, "allow");
        assert!((a.confidence - 0.95).abs() < 1e-6);
        assert_eq!(a.next_phase, "develop");
    }

    #[test]
    fn v2_candidates_round_trip_through_json() {
        let json = serde_json::json!({
            "decisions": [{
                "name": "action", "kind": "choice",
                "candidates": [
                    {"id": "allow", "text": "allow", "probability": 0.7},
                    {"id": "request_review", "text": "rr", "probability": 0.3}
                ]
            }]
        });
        let r: Reasoning = serde_json::from_value(json).unwrap();
        assert_eq!(r.decisions[0].candidates.len(), 2);
        let a = advice_from_reasoning(&r, &ctx(), "now");
        assert_eq!(a.recommended_action, "allow");
        assert!((a.confidence - 0.7).abs() < 1e-6);
    }
}
