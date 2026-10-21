//! `SupervisionEpisode v1` (supervisor_model.md) -> `head-train/v1`.
//!
//! Only episodes whose decision was *verified* become action/phase labels:
//! `outcome.verified && !outcome.human_override`, with a non-abstaining decision.
//! A shadow advisor's own advice is never a label. Unverified and overridden
//! episodes are kept as inputs for the heads that describe the task rather than
//! the decision (H10); their recorded decision is masked, never trained on, and
//! a human's `override_action` (when recorded) is the label instead.
//!
//! Heads whose answer is a field of the prompt (`intent`, `risk`) are not
//! emitted: the model would learn to copy them (H4). Teacher distributions go
//! in a separate `soft` object, never over the ground-truth `labels` (H2). The prompt text must equal the
//! server's `buildAdvisePrompt` (backend_server/internal/web/head_advise.go).

use regex::Regex;
use serde::Deserialize;
use serde_json::{json, Map, Value};
use std::collections::{BTreeMap, BTreeSet};

/// The canonical prompt vector, shared with the Go and TypeScript SDK packages
/// (KNIRVSDK/testvectors/models/prompt_vector.json). If one side changes, all must.
const PROMPT_VECTOR_JSON: &str = include_str!("../../../../KNIRVSDK/testvectors/models/prompt_vector.json");

/// The prompt string from the shared vector.
pub fn prompt_vector() -> String {
    let v: Value = serde_json::from_str(PROMPT_VECTOR_JSON).expect("shared prompt vector is valid JSON");
    v["prompt"].as_str().expect("vector has a prompt").to_string()
}

#[derive(Deserialize, Default)]
struct Task {
    #[serde(default)]
    request: String,
    #[serde(default)]
    intent: String,
    #[serde(default)]
    risk: String,
}
#[derive(Deserialize, Default)]
struct State {
    #[serde(default)]
    phase: String,
    #[serde(default)]
    budget: BTreeMap<String, i64>,
    #[serde(default)]
    agent_chain: Vec<String>,
}
#[derive(Deserialize, Default)]
struct Evidence {
    #[serde(default)]
    class: String,
}
#[derive(Deserialize, Default)]
struct Trace {
    #[serde(default)]
    agent: String,
}
#[derive(Deserialize, Default)]
struct Decision {
    #[serde(default)]
    recommended_action: String,
    #[serde(default)]
    next_phase: String,
    #[serde(default)]
    abstain: bool,
}
#[derive(Deserialize, Default)]
struct Outcome {
    #[serde(default)]
    verified: bool,
    #[serde(default)]
    human_override: bool,
    /// What the human did instead, when the collector recorded it.
    #[serde(default)]
    override_action: String,
}
#[derive(Deserialize, Default)]
struct Provenance {
    #[serde(default)]
    created_at: String,
}
#[derive(Deserialize, Default)]
struct Workspace {
    #[serde(default)]
    repo_family: String,
}
#[derive(Deserialize)]
pub struct Episode {
    #[serde(default, alias = "event_id")]
    episode_id: String,
    #[serde(default)]
    workspace: Workspace,
    #[serde(default)]
    task: Task,
    #[serde(default, alias = "state_before")]
    state: State,
    #[serde(default)]
    evidence: Vec<Evidence>,
    #[serde(default)]
    trace: Trace,
    decision: Option<Decision>,
    outcome: Option<Outcome>,
    #[serde(default)]
    provenance: Provenance,
}

/// Canonical model input; identical to the server's prompt builder.
pub fn prompt(e: &Episode) -> String {
    let patterns = [
        r"AKIA[0-9A-Z]{16}",
        r"gh[pousr]_[A-Za-z0-9]{36}",
        r"(?i)(?:password|passwd|token|secret|api[_-]?key)\s*[=:]\s*[^\s]{8,}",
    ];
    let mut req = e.task.request.clone();
    for p in patterns {
        req = Regex::new(p).expect("static pattern").replace_all(&req, "[REDACTED]").into_owned();
    }
    let budget = e.state.budget.iter().map(|(k, v)| format!("{k}={v}")).collect::<Vec<_>>().join(","); // BTreeMap: sorted
    let mut classes: Vec<&str> = e.evidence.iter().map(|x| x.class.as_str()).collect();
    classes.sort_unstable();
    format!(
        "phase:{} intent:{} risk:{} budget:{} agents:{} evidence:{}\nrequest:{}",
        e.state.phase, e.task.intent, e.task.risk, budget, e.state.agent_chain.join(">"), classes.join(","), req
    )
}

/// Risk levels are ordinal; the score head regresses their bin centres.
fn risk_score(level: &str) -> Option<f64> {
    match level {
        "low" => Some(0.125),
        "medium" => Some(0.375),
        "high" => Some(0.625),
        "critical" => Some(0.875),
        _ => None,
    }
}

#[derive(Default, Debug, PartialEq)]
pub struct Report {
    pub read: usize,
    pub written: usize,
    pub skipped_malformed: usize,
    /// No decision/outcome at all: nothing to learn from.
    pub skipped_unverified: usize,
    pub skipped_abstain: usize,
    /// Written, but the recorded decision is masked (unverified or overridden).
    pub kept_masked: usize,
    /// Overridden episodes labelled with the human's action.
    pub kept_corrected: usize,
    pub soft_attached: usize,
    pub soft_masked: usize,
}

/// Heads that are fields of the prompt. Emitting them as labels lets the model
/// read the answer off its input.
pub const INPUT_HEADS: &[&str] = &["intent", "risk"];

/// Heads the Clef teacher may label by default. `action` is excluded because it
/// carries the abstain decision: the M7-0 gate measured clef-flash calling a tool
/// on 25% of BFCL irrelevance items (should be 0), so abstention is learned only
/// from verified outcomes.
pub const DEFAULT_SOFT_HEADS: &[&str] = &["next_phase", "agent"];

/// `soft`: episode_id -> {head -> {option -> probability}} from the Clef teacher,
/// attached under `soft` (train split only). Only [`DEFAULT_SOFT_HEADS`] are taken.
pub fn convert(input: &str, soft: &BTreeMap<String, Map<String, Value>>) -> (Vec<Value>, Report) {
    convert_masked(input, soft, DEFAULT_SOFT_HEADS)
}

/// As [`convert`], with an explicit set of heads the teacher may label.
/// Teacher distributions for other heads are dropped and counted in `soft_masked`.
pub fn convert_masked(input: &str, soft: &BTreeMap<String, Map<String, Value>>, allowed: &[&str]) -> (Vec<Value>, Report) {
    convert_opts(input, soft, allowed, false)
}

/// `emit_input_heads` re-enables `intent`/`risk` labels (they leak, see [`INPUT_HEADS`]).
pub fn convert_opts(input: &str, soft: &BTreeMap<String, Map<String, Value>>, allowed: &[&str], emit_input_heads: bool) -> (Vec<Value>, Report) {
    let mut rep = Report::default();
    let mut out = Vec::new();
    let mut seen = BTreeSet::new();
    for line in input.lines().filter(|l| !l.trim().is_empty()) {
        rep.read += 1;
        let Ok(e) = serde_json::from_str::<Episode>(line) else {
            rep.skipped_malformed += 1;
            continue;
        };
        let (Some(d), Some(o)) = (&e.decision, &e.outcome) else {
            rep.skipped_unverified += 1;
            continue;
        };
        if e.episode_id.is_empty() || !seen.insert(e.episode_id.clone()) {
            rep.skipped_malformed += 1;
            continue;
        }
        let trusted = o.verified && !o.human_override;
        let abstained = d.abstain || d.recommended_action.is_empty() || d.recommended_action == "abstain";
        if trusted && abstained {
            rep.skipped_abstain += 1;
            continue;
        }

        let mut labels = Map::new();
        if trusted {
            labels.insert("action".into(), json!(d.recommended_action));
            if !d.next_phase.is_empty() {
                labels.insert("next_phase".into(), json!(d.next_phase));
            }
            if matches!(e.trace.agent.as_str(), "claude" | "codex" | "opencode" | "hermes") {
                labels.insert("agent".into(), json!(e.trace.agent));
            }
        } else {
            // The recorded decision was not confirmed: do not learn it. A human's
            // correction is the right answer for `action`.
            rep.kept_masked += 1;
            if o.human_override && !o.override_action.is_empty() {
                labels.insert("action".into(), json!(o.override_action));
                rep.kept_corrected += 1;
            }
        }
        if emit_input_heads {
            if !e.task.intent.is_empty() {
                labels.insert("intent".into(), json!(e.task.intent));
            }
            if let Some(r) = risk_score(&e.task.risk) {
                labels.insert("risk".into(), json!(r));
            }
        }
        if labels.is_empty() {
            rep.kept_masked -= 1;
            rep.skipped_unverified += 1;
            continue;
        }

        let mut soft_obj = Map::new();
        if let Some(s) = soft.get(&e.episode_id) {
            for (head, dist) in s {
                if !allowed.contains(&head.as_str()) || (!emit_input_heads && INPUT_HEADS.contains(&head.as_str())) {
                    rep.soft_masked += 1;
                    continue;
                }
                soft_obj.insert(head.clone(), dist.clone());
                rep.soft_attached += 1;
            }
        }
        // An empty repo family must not pool every such episode into one group (H3).
        let group = if e.workspace.repo_family.is_empty() { format!("ep:{}", e.episode_id) } else { e.workspace.repo_family.clone() };
        let mut rec = json!({"id": e.episode_id, "group": group, "input": prompt(&e), "labels": labels});
        if !soft_obj.is_empty() {
            rec["soft"] = Value::Object(soft_obj);
        }
        if !e.provenance.created_at.is_empty() {
            rec["ts"] = json!(e.provenance.created_at);
        }
        out.push(rec);
        rep.written += 1;
    }
    (out, rep)
}

#[cfg(test)]
mod tests {
    use super::*;

    fn ep(id: &str, extra: Value) -> String {
        let mut v = json!({
            "episode_id": id, "workspace": {"repo_family": "fam"},
            "task": {"request": "fix the build", "intent": "review", "risk": "medium"},
            "state_before": {"phase": "develop", "budget": {"b": 2, "a": 1}, "agent_chain": ["claude", "codex"]},
            "evidence": [{"class": "report"}, {"class": "diff"}],
            "trace": {"agent": "codex"},
            "decision": {"recommended_action": "request_review", "next_phase": "develop_review", "abstain": false},
            "outcome": {"verified": true, "human_override": false}
        });
        v.as_object_mut().unwrap().extend(extra.as_object().unwrap().clone());
        v.to_string()
    }

    #[test]
    fn prompt_matches_the_servers_vector() {
        let e: Episode = serde_json::from_str(&ep("x", json!({}))).unwrap();
        assert_eq!(prompt(&e), prompt_vector());
    }

    #[test]
    fn redacts_secrets_like_the_server() {
        let e: Episode = serde_json::from_str(&ep("x", json!({"task": {"request": "use AKIAABCDEFGHIJKLMNOP and password=hunter2hunter2", "intent": "review", "risk": "low"}}))).unwrap();
        let p = prompt(&e);
        assert!(!p.contains("AKIA") && !p.contains("hunter2"), "{p}");
        assert!(p.contains("[REDACTED]"));
    }

    #[test]
    fn only_verified_non_overridden_decisions_become_decision_labels() {
        let lines = [
            ep("ok", json!({})),
            ep("unverified", json!({"outcome": {"verified": false}})),
            ep("overridden", json!({"outcome": {"verified": true, "human_override": true}})),
            ep("abstained", json!({"decision": {"recommended_action": "allow", "abstain": true}})),
            ep("no-outcome", json!({"outcome": null})),
            ep("ok", json!({})), // duplicate id
            "not json".into(),
        ]
        .join("\n");
        let (out, rep) = convert(&lines, &BTreeMap::new());
        // "ok" trains; "unverified" and "overridden" carry no usable label once the decision is masked
        // and the input heads are not emitted, so they are skipped rather than written empty.
        assert_eq!(out.len(), 1);
        assert_eq!(rep, Report { read: 7, written: 1, skipped_malformed: 2, skipped_unverified: 3, skipped_abstain: 1, ..Default::default() });
        let l = &out[0]["labels"];
        assert_eq!((l["action"].as_str(), l["next_phase"].as_str(), l["agent"].as_str()), (Some("request_review"), Some("develop_review"), Some("codex")));
        assert_eq!(out[0]["input"], prompt_vector());
    }

    #[test]
    fn input_heads_are_not_emitted_unless_asked() {
        let (out, _) = convert(&ep("ok", json!({})), &BTreeMap::new());
        let l = out[0]["labels"].as_object().unwrap();
        assert!(!l.contains_key("intent") && !l.contains_key("risk"), "H4: these are fields of the prompt");
        let (out, _) = convert_opts(&ep("ok", json!({})), &BTreeMap::new(), DEFAULT_SOFT_HEADS, true);
        assert_eq!(out[0]["labels"]["intent"], "review");
        assert_eq!(out[0]["labels"]["risk"], 0.375);
    }

    #[test]
    fn unverified_episodes_are_kept_masked_and_corrections_become_labels() {
        let lines = [
            ep("unverified", json!({"outcome": {"verified": false}})),
            ep("corrected", json!({"outcome": {"verified": true, "human_override": true, "override_action": "ask_operator"}})),
        ]
        .join("\n");
        let (out, rep) = convert_opts(&lines, &BTreeMap::new(), DEFAULT_SOFT_HEADS, true);
        assert_eq!((rep.written, rep.kept_masked, rep.kept_corrected), (2, 2, 1));
        let un = &out[0]["labels"];
        assert!(un.get("action").is_none() && un.get("next_phase").is_none() && un.get("agent").is_none(), "recorded decision must be masked");
        assert_eq!(un["intent"], "review");
        assert_eq!(out[1]["labels"]["action"], "ask_operator");
    }

    #[test]
    fn empty_repo_family_gets_its_own_group_and_provenance_time_is_kept() {
        let (out, _) = convert(&ep("solo", json!({"workspace": {"repo_family": ""}, "provenance": {"created_at": "2026-01-02T03:04:05Z"}})), &BTreeMap::new());
        assert_eq!(out[0]["group"], "ep:solo");
        assert_eq!(out[0]["ts"], "2026-01-02T03:04:05Z");
    }

    #[test]
    fn teacher_distributions_go_in_soft_and_never_over_labels() {
        let mut soft = BTreeMap::new();
        let mut m = Map::new();
        m.insert("next_phase".into(), json!({"develop_review": 0.7, "develop": 0.3}));
        soft.insert("ok".to_string(), m);
        let (out, rep) = convert(&ep("ok", json!({})), &soft);
        assert_eq!((rep.soft_attached, rep.soft_masked), (1, 0));
        assert_eq!(out[0]["soft"]["next_phase"]["develop"], 0.3);
        assert_eq!(out[0]["labels"]["next_phase"], "develop_review", "H2: the hard label survives");
    }

    #[test]
    fn teacher_cannot_label_action_or_input_heads_by_default() {
        let mut soft = BTreeMap::new();
        let mut m = Map::new();
        m.insert("action".into(), json!({"abstain": 0.1, "ask_operator": 0.9}));
        m.insert("intent".into(), json!({"review": 0.5, "validate": 0.5}));
        soft.insert("ok".to_string(), m.clone());
        let (out, rep) = convert(&ep("ok", json!({})), &soft);
        assert_eq!((rep.soft_attached, rep.soft_masked), (0, 2));
        assert!(out[0].get("soft").is_none());
        assert_eq!(out[0]["labels"]["action"], "request_review");
        // Explicitly widening the mask is possible, never implicit.
        let (out, rep) = convert_masked(&ep("ok", json!({})), &soft, &["action"]);
        assert_eq!(rep.soft_attached, 1);
        assert_eq!(out[0]["soft"]["action"]["ask_operator"], 0.9);
    }

    #[test]
    fn unknown_agents_are_not_invented() {
        let (out, _) = convert(&ep("x", json!({"trace": {"agent": "mystery"}})), &BTreeMap::new());
        assert!(!out[0]["labels"].as_object().unwrap().contains_key("agent"));
    }
}
