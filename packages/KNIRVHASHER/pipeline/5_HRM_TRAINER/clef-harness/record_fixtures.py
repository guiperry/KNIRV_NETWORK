"""Capture real Clef responses as regression fixtures.

The contract tests currently validate hand-written responses and Cloudflare's
published schemas. Neither proves that the *live* endpoint returns the shape we
believe it does -- that gap is exactly what would break an integration first.

Run with credentials present:
    python3 record_fixtures.py

Writes fixtures/recorded/<question_type>.json containing the raw Workers AI
`result` envelope. These are live API responses to PUBLIC data only (a
BANKING77 utterance); no KNIRV episode content is involved.

Fixtures are committed on purpose -- they are what makes "the API changed" a
failing test instead of a 3am page.
"""

from __future__ import annotations

import json
import os
import sys
from typing import Any, Dict

from clef_client import ClefClient, LiveUnavailable, preflight
from clef_contract import (
    build_request,
    choice,
    noul,
    parse_response,
    score,
    validate_against_published_schema,
)

HERE = os.path.dirname(os.path.abspath(__file__))
OUT = os.path.join(HERE, "fixtures", "recorded")

BANKING_TEXT = "How do I locate my card?"

CASES: Dict[str, Dict[str, Any]] = {
    "noul": {
        "questions": {
            "evidence_present": noul(
                "Is a verified test_report artifact present for this task?",
                true_means="A verified test_report artifact is present",
                false_means="No verified test_report artifact is present",
            )
        }
    },
    "choice": {
        "questions": {
            "action": choice(
                "Which supervisory action should be recommended?",
                {
                    "allow": "Safe to proceed without review",
                    "request_review": "Pause for human review",
                    "abstain": "Make no recommendation",
                },
            )
        }
    },
    "score": {
        "questions": {
            "severity": score(
                "How severe is the risk if this proceeds unsupervised?",
                ["none", "low", "medium", "high", "critical"],
            )
        }
    },
}


def main() -> int:
    pf = preflight()
    if not pf["ready"]:
        print("preflight failed:", pf["blocker"], file=sys.stderr)
        return 1

    os.makedirs(OUT, exist_ok=True)
    client = ClefClient(model="clef-flash")
    failures = 0

    for name, case in CASES.items():
        body = build_request("clef-flash", {"request": BANKING_TEXT}, case["questions"])
        resp, call = client.ask({"request": BANKING_TEXT}, case["questions"], validate=False)
        if not call.ok:
            print(f"  {name}: FAILED {call.error[:200]}", file=sys.stderr)
            failures += 1
            continue

        # Re-issue with validation on now that we know the shape round-trips.
        problems = validate_against_published_schema(resp.raw, "output") if hasattr(resp, "raw") else []
        path = os.path.join(OUT, f"{name}.json")
        with open(path, "w") as fh:
            json.dump(
                {
                    "_note": "Live Workers AI response envelope, captured for regression testing.",
                    "_question_type": name,
                    "_state_sent": {"request": BANKING_TEXT},
                    "_request_body": body,
                    "_usage": {"input_tokens": call.input_tokens, "output_tokens": call.output_tokens},
                    "result": _reconstruct(resp),
                },
                fh,
                indent=2,
            )
        print(f"  {name}: ok  in={call.input_tokens} out={call.output_tokens} "
              f"latency={call.latency_s:.3f}s cost=${call.cost_usd:.6f}", file=sys.stderr)
        if problems:
            print(f"    NOTE published-schema deviations: {problems[:3]}", file=sys.stderr)

    if failures:
        print(f"{failures} fixture(s) failed", file=sys.stderr)
        return 1
    print(f"wrote {len(CASES)} fixtures to {OUT}", file=sys.stderr)
    return 0


def _reconstruct(resp: Any) -> Dict[str, Any]:
    """Rebuild the full result envelope from parsed decisions."""
    answers: Dict[str, Any] = {}
    for qid, d in resp.decisions.items():
        answers[qid] = dict(d.raw)
    return {
        "model": resp.model,
        "answers": answers,
        "usage": {"input_tokens": resp.input_tokens, "output_tokens": resp.output_tokens},
    }


if __name__ == "__main__":
    raise SystemExit(main())