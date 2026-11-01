"""KNIRV <-> Clef schema translation.

Mirrors two contracts that must stay in lockstep:

  KNIRV   KNIRV_CORP/packages/cli/internal/supervisor/advisor/advisor.go
          (SupervisorContext, SupervisorAdvice, allowedActions, MinConfidence)
          and supervisor_model.md's SupervisionEpisode v1 decision block.

  Clef    clef_contract.py, transcribed from Cloudflare's published schemas.

This module is pure: no network, no model. It is the piece that would ship as
the KNIRV-side adapter, so it is deliberately the only place that knows about
both vocabularies.

Three translation rules from Clef_KNIRV_Design.md section 4.2 are enforced here
and are the reason this is not a one-liner:

  1. score is CONTINUOUS ("can land between levels") -> Severity keeps the
     float; rounding happens only at display time.
  2. noul returns no confidence -> derived explicitly in clef_contract.parse_answer.
  3. abstention is a threshold on peak probability, never a model-chosen label.
"""

from __future__ import annotations

from dataclasses import dataclass
from typing import Any, Dict, List, Optional

from clef_contract import Decision, Question, choice, noul, score

# Verbatim from advisor.go:246. Order matters for display only; the Go map is
# the authority and this must stay identical to it.
ALLOWED_ACTIONS: List[str] = [
    "allow",
    "request_review",
    "request_evidence",
    "ask_operator",
    "recover",
    "abstain",
]

# advisor.go:37
MIN_CONFIDENCE = 0.3

# supervisor_model.md:183 -- Required label family 1.
INTENTS: List[str] = [
    "plan",
    "investigate",
    "develop",
    "review",
    "validate",
    "recover",
    "ask_operator",
]

# Ordered, so it maps onto a Clef `score` rubric rather than a `choice`.
# Index is meaningful: 0 = least severe.
SEVERITY_LEVELS: List[str] = ["none", "low", "medium", "high", "critical"]

# Descriptions make the rubric legible to the model. A bare list of words gives
# the routing head nothing to condition on.
SEVERITY_DESCRIPTIONS: Dict[str, str] = {
    "none": "No risk: read-only, reversible, or trivially local change",
    "low": "Low risk: reversible local change with no external or destructive effect",
    "medium": "Medium risk: touches shared state or requires a full test run to reverse",
    "high": "High risk: destructive, irreversible, or affects other agents or operators",
    "critical": "Critical risk: credential, provenance, or supply-chain integrity is implicated",
}

ACTION_DESCRIPTIONS: Dict[str, str] = {
    "allow": "The action is safe to proceed without further review",
    "request_review": "Pause and request human review before proceeding",
    "request_evidence": "Pause and require the named missing evidence artifact",
    "ask_operator": "Escalate to the operator for a decision the policy does not cover",
    "recover": "The action failed; enter the recovery path",
    "abstain": "Policy context is unavailable or contradictory; make no recommendation",
}

INTENT_DESCRIPTIONS: Dict[str, str] = {
    "plan": "Decompose the request before touching code",
    "investigate": "Gather information about a problem without changing state",
    "develop": "Write or modify code",
    "review": "Evaluate existing changes for correctness and policy compliance",
    "validate": "Run tests or verification and report results",
    "recover": "Restore a previous good state after failure",
    "ask_operator": "The request cannot proceed without an operator decision",
}


class TranslationError(ValueError):
    pass


@dataclass
class SupervisorContext:
    """The redacted, immutable snapshot. Field names match the Go struct tags."""

    current_phase: str
    task_request: str
    task_intent: str
    task_risk: str
    policy_bundle_hash: str = ""
    policy_refs: List[str] = None  # type: ignore[assignment]
    evidence_classes: List[str] = None  # type: ignore[assignment]
    previous_phase: str = ""
    agent_chain: List[str] = None  # type: ignore[assignment]

    def __post_init__(self) -> None:
        self.policy_refs = list(self.policy_refs or [])
        self.evidence_classes = list(self.evidence_classes or [])
        self.agent_chain = list(self.agent_chain or [])


@dataclass
class SupervisorAdvice:
    """Target shape for advisor.go's SupervisorAdvice."""

    recommended_action: str
    next_phase: str
    confidence: float
    abstain: bool
    abstain_reason: str = ""
    severity: Optional[float] = None  # rule 1: continuous, may sit between levels
    severity_band: str = ""
    intent: str = ""
    missing_evidence: List[str] = None  # type: ignore[assignment]
    policy_references: List[str] = None  # type: ignore[assignment]
    model_version: str = ""
    dataset_manifest_hash: str = ""

    def __post_init__(self) -> None:
        self.missing_evidence = list(self.missing_evidence or [])
        self.policy_references = list(self.policy_references or [])


def severity_index(band: str) -> int:
    """Map a severity band to its rubric index, for building Gold labels.

    Returns int, not float, so callers can index SEVERITY_LEVELS directly.
    """
    try:
        return SEVERITY_LEVELS.index(band)
    except ValueError as exc:
        raise TranslationError(f"unknown severity band {band!r}") from exc


def build_questions(
    ctx: SupervisorContext,
    valid_phases: Optional[List[str]] = None,
    required_evidence: Optional[List[str]] = None,
) -> Dict[str, Question]:
    """SupervisorContext -> Clef questions.

    All of these fit in ONE request (limit is 64 questions), which is where the
    latency claim actually cashes out -- see design doc section 4.3.
    """
    questions: Dict[str, Question] = {}

    questions["action"] = choice(
        "Which supervisory action should the KNIRV supervisor recommend for this task? "
        "The model advises only; deterministic policy and evidence gates still decide.",
        dict(ACTION_DESCRIPTIONS),
    )

    questions["intent"] = choice(
        "What is the operator's intent for this task?",
        dict(INTENT_DESCRIPTIONS),
    )

    questions["severity"] = score(
        "How severe is the risk if this task proceeds unsupervised?",
        [SEVERITY_DESCRIPTIONS[lvl] for lvl in SEVERITY_LEVELS],
    )

    if valid_phases:
        questions["next_phase"] = choice(
            "Which supervisor phase should follow the current phase?",
            {p: f"Enter the {p} phase" for p in valid_phases},
        )

    for cls in required_evidence or []:
        key = f"evidence_{cls}"
        questions[key] = noul(
            f"Is the required evidence artifact of class '{cls}' already present and sufficient?",
            true_means=f"A verified {cls} artifact is present",
            false_means=f"No verified {cls} artifact is present",
        )

    return questions


def render_state(ctx: SupervisorContext, retrieved_precedents: Optional[List[Dict[str, Any]]] = None) -> Any:
    """SupervisorContext -> Clef `state`.

    Structured (object) rather than a prose blob: docs accept text, JSON,
    images, or video, and a structured state makes the contrast-permutation
    experiments in the scorecard tractable.
    """
    state: Dict[str, Any] = {
        "current_phase": ctx.current_phase,
        "task": {
            "request": ctx.task_request,
            "declared_intent": ctx.task_intent,
            "declared_risk": ctx.task_risk,
        },
        "policy_bundle_hash": ctx.policy_bundle_hash,
        "policy_refs": ctx.policy_refs,
        "evidence_present": ctx.evidence_classes,
    }
    if ctx.previous_phase:
        state["previous_phase"] = ctx.previous_phase
    if ctx.agent_chain:
        state["agent_chain"] = ctx.agent_chain
    if retrieved_precedents:
        state["retrieved_precedents"] = retrieved_precedents
    return state


def to_advice(
    decisions: Dict[str, Decision],
    tau: float = MIN_CONFIDENCE,
    model_version: str = "",
    manifest_hash: str = "",
    next_phase_default: str = "",
) -> SupervisorAdvice:
    """Clef decisions -> SupervisorAdvice, with the three rules applied.

    `tau` is deliberately a parameter and NOT the Go MinConfidence constant:
    rule 3 says tau must be fitted against the Gold manifest, not inherited.
    """
    advice = SupervisorAdvice(
        recommended_action="abstain",
        next_phase=next_phase_default,
        confidence=0.0,
        abstain=True,
        model_version=model_version,
        dataset_manifest_hash=manifest_hash,
    )

    if "action" not in decisions:
        advice.abstain_reason = "no action answer returned"
        return advice

    action = decisions["action"]

    # Rule 3: confidence-based abstention. We deliberately do NOT let the model
    # pick the "abstain" action label -- that conflates low confidence with a
    # policy-mandated refusal.
    if action.abstains(tau):
        advice.abstain_reason = f"action peak probability {action.top_probability:.3f} < tau {tau:.3f}"
        return advice

    if action.chosen not in ALLOWED_ACTIONS:
        advice.abstain_reason = f"action {action.chosen!r} is not in the KNIRV advisory vocabulary"
        return advice

    advice.recommended_action = str(action.chosen)
    advice.confidence = action.confidence
    advice.abstain = False

    if "intent" in decisions and decisions["intent"].chosen:
        advice.intent = str(decisions["intent"].chosen)

    if "next_phase" in decisions:
        np = decisions["next_phase"]
        if np.abstains(tau):
            advice.abstain = True
            advice.abstain_reason = f"next_phase peak probability {np.top_probability:.3f} < tau {tau:.3f}"
            return advice
        if np.chosen:
            advice.next_phase = str(np.chosen)

    # Rule 1: keep the continuous score. Rounding is a display concern.
    sev = decisions.get("severity")
    if sev is not None:
        s = sev.raw.get("score")
        if isinstance(s, (int, float)):
            advice.severity = float(s)
            idx = int(round(float(s)))
            advice.severity_band = (
                SEVERITY_LEVELS[idx] if 0 <= idx < len(SEVERITY_LEVELS) else "unbanded"
            )
            if not advice.abstain and advice.severity >= 3.0:
                # Critical severity may warrant a confirmation floor regardless of
                # which action was chosen. Deterministic code still has the last word.
                advice.policy_references.append(f"severity-floor:{advice.severity_band}")

    advice.missing_evidence = sorted(
        qid.removeprefix("evidence_")
        for qid, d in decisions.items()
        if qid.startswith("evidence_") and d.raw.get("noul", 1.0) < 0.5
    )

    return advice


def to_episode_labels(advice: SupervisorAdvice, valid_phases: List[str]) -> Dict[str, Any]:
    """SupervisorAdvice -> the SupervisionEpisode v1 `decision` block.

    Used to write Gold labels; keeps the harness and the trainer on one shape.
    """
    return {
        "recommended_action": advice.recommended_action,
        "next_phase": advice.next_phase,
        "required_evidence": list(advice.missing_evidence),
        "approval": "not_required" if advice.recommended_action == "allow" else "required",
        "abstain": advice.abstain,
        "confidence": round(float(advice.confidence), 4),
        "severity": advice.severity,
        "severity_band": advice.severity_band,
    }