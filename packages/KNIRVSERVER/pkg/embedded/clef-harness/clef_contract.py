"""Clef API contract, transcribed from Cloudflare's published JSON schemas.

Source of truth (fetched 2026-10-05):
  https://developers.cloudflare.com/workers-ai/models/clef-flash/schema-input.json
  https://developers.cloudflare.com/workers-ai/models/clef-flash/schema-output.json

Everything here is verified against those files by test_contract.py. Do not
hand-edit from memory -- if the upstream schema changes, the tests fail and
that is the intended signal.
"""

from __future__ import annotations

import json
import os
import re
from dataclasses import dataclass, field
from typing import Any, Dict, List, Optional

API_BASE = "https://api.cloudflare.com/client/v4/accounts/{account_id}/ai/run/@cf/cloudflare/{model_slug}"

MODEL_SLUGS = {"clef": "clef", "clef-flash": "clef-flash"}
# The published input schema permits surrounding whitespace: ^\s*(clef|clef-flash)\s*$
MODEL_RE = re.compile(r"^\s*(clef|clef-flash)\s*$")

# Published unit pricing, per MILLION input tokens (docs, 2026-10-01).
PRICE_PER_MTOK = {"clef": 0.24, "clef-flash": 0.09}

# Documented limits.
MAX_QUESTIONS = 64
MAX_QUESTION_ID = 100
QUESTION_ID_RE = re.compile(r"^[A-Za-z0-9_.\-]+$")
MAX_IMAGES = 4
MAX_BODY_BYTES = 13 * 1024 * 1024
CONTEXT_TOKENS = 65536

QUESTION_TYPES = ("noul", "choice", "score")

# Limits stated only in the published schema descriptions, not in the prose docs.
MIN_CHOICE_OPTIONS, MAX_CHOICE_OPTIONS = 2, 255
MIN_SCORE_LEVELS, MAX_SCORE_LEVELS = 2, 10


class ContractError(ValueError):
    """Raised when a request or response violates the published schema."""


def _is_instruction(v: Any) -> bool:
    """instructions may be a non-empty string, or an object/array holding the
    question plus referenced data (schema descriptions for all three types)."""
    if isinstance(v, str):
        return bool(v.strip())
    return isinstance(v, (dict, list)) and len(v) > 0


@dataclass
class Question:
    type: str
    instructions: str
    criteria: Optional[Any] = None

    def validate(self, qid: str) -> None:
        if self.type not in QUESTION_TYPES:
            raise ContractError(f"question {qid!r}: type {self.type!r} not in {QUESTION_TYPES}")
        if not _is_instruction(self.instructions):
            raise ContractError(f"question {qid!r}: instructions must be a non-empty string, object, or array")
        if self.type == "choice":
            if not isinstance(self.criteria, dict) or not self.criteria:
                raise ContractError(f"question {qid!r}: choice requires a non-empty criteria object")
            n = len(self.criteria)
            if not (MIN_CHOICE_OPTIONS <= n <= MAX_CHOICE_OPTIONS):
                raise ContractError(
                    f"question {qid!r}: choice requires {MIN_CHOICE_OPTIONS}..{MAX_CHOICE_OPTIONS} options, got {n}"
                )
            for opt, desc in self.criteria.items():
                if not isinstance(opt, str) or not opt:
                    raise ContractError(f"question {qid!r}: choice criteria keys must be non-empty strings")
                # Schema: description may be "string, object, array, or null when
                # no detail is needed" -- null is legal, an empty string is not
                # explicitly forbidden but is certainly not a description.
                if desc is not None and not isinstance(desc, (str, dict, list)):
                    raise ContractError(
                        f"question {qid!r}: choice criteria {opt!r} must be string/object/array/null"
                    )
        elif self.type == "score":
            if not isinstance(self.criteria, list):
                raise ContractError(f"question {qid!r}: score requires an ordered criteria list")
            n = len(self.criteria)
            if not (MIN_SCORE_LEVELS <= n <= MAX_SCORE_LEVELS):
                raise ContractError(
                    f"question {qid!r}: score requires {MIN_SCORE_LEVELS}..{MAX_SCORE_LEVELS} levels, got {n}"
                )
            # Schema: "Ordered level descriptions (string, object, or array), lowest first"
            if not all(c is None or isinstance(c, (str, dict, list)) for c in self.criteria):
                raise ContractError(f"question {qid!r}: score criteria must be string/object/array")
        elif self.type == "noul" and self.criteria is not None:
            if not isinstance(self.criteria, dict):
                raise ContractError(f"question {qid!r}: noul criteria must be an object")
            for side in ("true", "false"):
                if side in self.criteria and not isinstance(self.criteria[side], (str, dict, list)):
                    raise ContractError(f"question {qid!r}: noul criteria.{side} must be a description")

    def to_wire(self) -> Dict[str, Any]:
        out: Dict[str, Any] = {"type": self.type, "instructions": self.instructions}
        if self.criteria is not None:
            out["criteria"] = self.criteria
        return out


def noul(instructions: str, true_means: Optional[str] = None, false_means: Optional[str] = None) -> Question:
    crit = None
    if true_means or false_means:
        crit = {}
        if true_means:
            crit["true"] = true_means
        if false_means:
            crit["false"] = false_means
    return Question("noul", instructions, crit)


def choice(instructions: str, criteria: Dict[str, str]) -> Question:
    return Question("choice", instructions, criteria)


def score(instructions: str, levels: List[str]) -> Question:
    return Question("score", instructions, list(levels))


@dataclass
class Decision:
    """A parsed answer for one question.

    `probabilities` is the calibrated distribution. For noul it is synthesised
    as {True: p, False: 1-p} so that a single abstention rule works across all
    three question types.
    """

    qid: str
    type: str
    confidence: float
    probabilities: Dict[str, float]
    chosen: Optional[str] = None
    raw: Dict[str, Any] = field(default_factory=dict)

    @property
    def top_probability(self) -> float:
        return max(self.probabilities.values()) if self.probabilities else 0.0

    def abstains(self, tau: float) -> bool:
        """Abstention is a threshold on peak probability, NOT a model-chosen label.

        See Clef_KNIRV_Design.md section 4.2 rule 3.
        """
        return self.top_probability < tau


def build_request(model: str, state: Any, questions: Dict[str, Question]) -> Dict[str, Any]:
    m = MODEL_RE.match(model) if isinstance(model, str) else None
    if m is None:
        raise ContractError(f"model must match the published pattern ^\\s*(clef|clef-flash)\\s*$, got {model!r}")
    model = m.group(1)
    if not isinstance(questions, dict) or not questions:
        raise ContractError("questions is required and must be a non-empty object")
    if len(questions) > MAX_QUESTIONS:
        raise ContractError(f"too many questions: {len(questions)} > {MAX_QUESTIONS}")
    for qid, q in questions.items():
        if len(qid) > MAX_QUESTION_ID or not QUESTION_ID_RE.match(qid):
            raise ContractError(f"invalid question id {qid!r}")
        q.validate(qid)
    if state is None or (isinstance(state, str) and not state.strip()):
        raise ContractError("state is required")
    return {"model": model, "state": state, "questions": {k: v.to_wire() for k, v in questions.items()}}


def parse_answer(qid: str, answer: Dict[str, Any]) -> Decision:
    if not isinstance(answer, dict) or "type" not in answer:
        raise ContractError(f"answer {qid!r}: missing type")
    atype = answer["type"]
    if atype not in QUESTION_TYPES:
        raise ContractError(f"answer {qid!r}: unexpected type {atype!r}")

    if atype == "noul":
        # Docs: noul returns ONLY P(yes) -- there is no confidence field.
        if "noul" not in answer:
            raise ContractError(f"answer {qid!r}: noul missing 'noul'")
        p = _prob(answer["noul"], qid)
        # Rule 2 (design doc 4.2): derive confidence explicitly and record that
        # we did, rather than silently measuring the wrong quantity.
        return Decision(qid, atype, max(p, 1.0 - p), {str(True): p, str(False): 1.0 - p}, raw=answer)

    for field_name in ("confidence", "probabilities"):
        if field_name not in answer:
            raise ContractError(f"answer {qid!r}: {atype} missing {field_name!r}")
    probs = answer["probabilities"]
    if not isinstance(probs, dict) or not probs:
        raise ContractError(f"answer {qid!r}: probabilities must be a non-empty object")
    probs = {str(k): _prob(v, qid) for k, v in probs.items()}
    total = sum(probs.values())
    if abs(total - 1.0) > 0.02:
        raise ContractError(f"answer {qid!r}: probabilities sum to {total:.4f}, expected 1.0")
    conf = _prob(answer["confidence"], qid)

    if atype == "choice":
        chosen = answer.get("choice")
        if chosen is None:
            raise ContractError(f"answer {qid!r}: choice missing 'choice'")
        return Decision(qid, atype, conf, probs, chosen=str(chosen), raw=answer)

    if "score" not in answer:
        raise ContractError(f"answer {qid!r}: score missing 'score'")
    if "legend" not in answer or not isinstance(answer["legend"], dict):
        # Schema requires 'legend' for score; the loop above already enforced
        # confidence+probabilities, but legend was missed there.
        raise ContractError(f"answer {qid!r}: score requires a 'legend' object")
    s = answer["score"]
    if isinstance(s, bool) or not isinstance(s, (int, float)):
        raise ContractError(f"answer {qid!r}: score must be a number, got {s!r}")
    return Decision(qid, atype, conf, probs, chosen=None, raw=answer)


@dataclass
class ClefResponse:
    model: str
    decisions: Dict[str, Decision]
    input_tokens: int
    output_tokens: int

    @property
    def cost_usd(self) -> float:
        return estimate_cost_usd(self.model, self.input_tokens) if self.model in PRICE_PER_MTOK else float("nan")


def parse_response(payload: Dict[str, Any], expected_ids: Optional[List[str]] = None) -> ClefResponse:
    """Validate a full Clef result envelope.

    The published output schema requires top-level `model`, `answers`, and
    `usage` {input_tokens, output_tokens}. `usage` is what makes the cost model
    in the design doc an estimate rather than a guess, so it is required here
    rather than merely read when present.
    """
    if not isinstance(payload, dict):
        raise ContractError("response must be a JSON object")
    for req in ("model", "answers", "usage"):
        if req not in payload:
            raise ContractError(f"response missing required field {req!r}")

    answers = payload["answers"]
    if not isinstance(answers, dict) or not answers:
        raise ContractError("response.answers must be a non-empty object")
    if expected_ids is not None:
        missing = set(expected_ids) - set(answers)
        extra = set(answers) - set(expected_ids)
        if missing:
            raise ContractError(f"response missing answers for {sorted(missing)}")
        if extra:
            raise ContractError(f"response has unexpected answers {sorted(extra)}")

    usage = payload["usage"]
    for tok in ("input_tokens", "output_tokens"):
        v = usage.get(tok)
        if isinstance(v, bool) or not isinstance(v, int) or v < 0:
            raise ContractError(f"response.usage.{tok} must be a non-negative integer, got {v!r}")

    return ClefResponse(
        model=str(payload["model"]),
        decisions={qid: parse_answer(qid, ans) for qid, ans in answers.items()},
        input_tokens=usage["input_tokens"],
        output_tokens=usage["output_tokens"],
    )


def _prob(v: Any, qid: str) -> float:
    if isinstance(v, bool) or not isinstance(v, (int, float)):
        raise ContractError(f"answer {qid!r}: expected a number, got {v!r}")
    f = float(v)
    if f != f or f in (float("inf"), float("-inf")):
        raise ContractError(f"answer {qid!r}: non-finite probability {v!r}")
    if not (0.0 <= f <= 1.0):
        raise ContractError(f"answer {qid!r}: probability {f} out of [0,1]")
    return f


def estimate_cost_usd(model: str, input_tokens: int) -> float:
    return input_tokens / 1_000_000.0 * PRICE_PER_MTOK[model]


SCHEMA_PATHS = {
    "input": os.path.join(os.path.dirname(__file__), "fixtures", "clef-schema-input.json"),
    "output": os.path.join(os.path.dirname(__file__), "fixtures", "clef-schema-output.json"),
}


def validate_against_published_schema(payload: Any, which: str = "input") -> List[str]:
    """Validate against Cloudflare's published JSON schema.

    Returns a list of human-readable violations (empty == valid). Kept separate
    from parse_* so that a Clef API drift produces a precise jsonschema error
    rather than a generic ContractError.
    """
    try:
        import jsonschema
    except ImportError:
        return ["jsonschema not installed; skipping published-schema validation"]
    with open(SCHEMA_PATHS[which]) as fh:
        schema = json.load(fh)
    validator = jsonschema.Draft7Validator(schema)
    return [
        f"{'/'.join(str(p) for p in e.absolute_path) or '<root>'}: {e.message}"
        for e in sorted(validator.iter_errors(payload), key=lambda e: list(e.absolute_path))
    ]