"""Workers AI transport for Clef, plus a credential-free stub.

Two entry points on purpose:

  ClefClient.ask(...)     -- live call, requires CLOUDFLARE_API_TOKEN
  offline_stub(...)       -- deterministic synthetic answers

The stub exists so the whole pipeline -- translation, validation, metrics,
reporting -- is executable and testable today with no Cloudflare account. It
returns *schema-valid* responses with deliberately poor accuracy, so any
pipeline bug surfaces immediately instead of hiding behind a green run that
never happened.
"""

from __future__ import annotations

import hashlib
import json
import os
import random
import time
import urllib.error
import urllib.request
from dataclasses import dataclass
from typing import Any, Dict, List, Optional

from clef_contract import (
    API_BASE,
    MODEL_SLUGS,
    PRICE_PER_MTOK,
    ContractError,
    Question,
    build_request,
    parse_response,
    validate_against_published_schema,
)

TOKEN_ENV = "CLOUDFLARE_API_TOKEN"
ACCOUNT_ENV = "CLOUDFLARE_ACCOUNT_ID"

DOTENV = os.path.join(os.path.dirname(os.path.abspath(__file__)), ".env")


def load_dotenv(path: Optional[str] = None) -> List[str]:
    """Populate the environment from a .env file.

    Existing environment variables are NEVER overwritten, so a real CI secret
    always wins over a local file. Values are not logged or echoed anywhere.

    .env is gitignored. It holds a bearer token for a billable endpoint.
    """
    path = path or DOTENV
    if not os.path.exists(path):
        return []
    loaded: List[str] = []
    with open(path) as fh:
        for raw in fh:
            line = raw.strip()
            if not line or line.startswith("#") or "=" not in line:
                continue
            key, _, value = line.partition("=")
            key = key.strip()
            value = value.strip().strip('"').strip("'")
            if key and key not in os.environ:
                os.environ[key] = value
                loaded.append(key)
    return loaded


def credentials_available(path: Optional[str] = None) -> bool:
    load_dotenv(path)
    return bool(os.environ.get(TOKEN_ENV) and os.environ.get(ACCOUNT_ENV))


@dataclass
class Call:
    latency_s: float
    input_tokens: int
    output_tokens: int
    cost_usd: float
    ok: bool
    error: str = ""


class ClefClient:
    def __init__(
        self,
        model: str = "clef-flash",
        token: Optional[str] = None,
        account_id: Optional[str] = None,
        timeout: float = 120.0,
    ) -> None:
        if model not in MODEL_SLUGS:
            raise ContractError(f"unknown model {model!r}")
        self.model = model
        load_dotenv()
        self.token = token or os.environ.get(TOKEN_ENV)
        self.account_id = account_id or os.environ.get(ACCOUNT_ENV)
        self.timeout = timeout

    def _require_credentials(self) -> None:
        if not self.token or not self.account_id:
            raise ContractError(
                f"live Clef calls need {TOKEN_ENV} and {ACCOUNT_ENV} in the environment. "
                "Use offline_stub() to exercise the pipeline without an account, or see "
                "Clef_KNIRV_Design.md section 6.1 before putting KNIRV data through this path."
            )

    def ask(
        self,
        state: Any,
        questions: Dict[str, Question],
        validate: bool = True,
    ) -> tuple[Any, Call]:
        self._require_credentials()
        body = build_request(self.model, state, questions)
        if validate:
            problems = validate_against_published_schema(body, "input")
            if problems:
                raise ContractError("request fails Cloudflare's published input schema:\n  " + "\n  ".join(problems))

        url = API_BASE.format(account_id=self.account_id, model_slug=MODEL_SLUGS[self.model])
        payload = json.dumps(body).encode()
        req = urllib.request.Request(
            url,
            data=payload,
            headers={
                "Authorization": f"Bearer {self.token}",
                "Content-Type": "application/json",
            },
            method="POST",
        )

        t0 = time.perf_counter()
        try:
            with urllib.request.urlopen(req, timeout=self.timeout) as resp:
                raw = json.loads(resp.read().decode())
            latency = time.perf_counter() - t0
        except urllib.error.HTTPError as exc:
            detail = exc.read().decode()[:500]
            return None, Call(
                latency_s=time.perf_counter() - t0,
                input_tokens=0,
                output_tokens=0,
                cost_usd=0.0,
                ok=False,
                error=f"HTTP {exc.code}: {detail}",
            )
        except (urllib.error.URLError, TimeoutError, json.JSONDecodeError) as exc:
            return None, Call(
                latency_s=time.perf_counter() - t0,
                input_tokens=0,
                output_tokens=0,
                cost_usd=0.0,
                ok=False,
                error=f"{type(exc).__name__}: {exc}",
            )

        # Workers AI wraps the model result in an envelope.
        result = raw.get("result", raw)
        if not raw.get("success", True):
            return None, Call(latency, 0, 0, 0.0, False, error=str(raw.get("errors", raw))[:500])

        if validate:
            problems = validate_against_published_schema(result, "output")
            if problems:
                return None, Call(latency, 0, 0, 0.0, False,
                                  error="response fails Cloudflare's published output schema:\n  " + "\n  ".join(problems))

        parsed = parse_response(result, expected_ids=list(questions))
        cost = parsed.input_tokens / 1_000_000.0 * PRICE_PER_MTOK[self.model]
        return parsed, Call(latency, parsed.input_tokens, parsed.output_tokens, cost, True)


def offline_stub(
    state: Any,
    questions: Dict[str, Question],
    seed: int = 0,
    accuracy: float = 0.35,
    seed_state: Optional[str] = None,
) -> tuple[Any, Call]:
    """Schema-valid synthetic responses. No network.

    `seed_state` may carry the gold label (e.g. the BANKING77 category) so the
    stub can be exercised at a known accuracy -- still noisy, so calibration
    metrics have something non-degenerate to measure.

    Deliberately does NOT look like a real model: confidence is independent of
    correctness, which is what a miscalibrated model looks like. That makes ECE
    a live assertion in the smoke test rather than a coincidence.
    """
    rng = random.Random(f"{seed}:{seed_state}:{json.dumps(state, sort_keys=True, default=str)}")

    state_chars = len(json.dumps(state, default=str))
    est_in = max(1, state_chars // 4)

    answers: Dict[str, Any] = {}
    gold = seed_state
    for qid, q in questions.items():
        if q.type == "choice":
            opts = list(q.criteria)
            raw = [rng.random() for _ in opts]
            if gold in opts and rng.random() < accuracy:
                weights = [0.02] * len(opts)
                weights[opts.index(gold)] = 1.0
                raw = weights
            total = sum(raw)
            probs = {o: r / total for o, r in zip(opts, raw)}
            conf = rng.random()  # independent of correctness, on purpose
            answers[qid] = {
                "type": "choice",
                "choice": max(probs, key=probs.get),
                "probabilities": probs,
                "confidence": conf,
            }
        elif q.type == "score":
            levels = list(q.criteria)
            idx = rng.randrange(len(levels))
            raw = [rng.random() * 0.1 for _ in levels]
            raw[idx] = 1.0
            total = sum(raw)
            probs = {str(i): r / total for i, r in enumerate(raw)}
            answers[qid] = {
                "type": "score",
                "score": float(idx),
                "legend": {str(i): (lv if isinstance(lv, str) else json.dumps(lv)) for i, lv in enumerate(levels)},
                "probabilities": probs,
                "confidence": rng.random(),
            }
        else:  # noul
            p = rng.random()
            answers[qid] = {"type": "noul", "noul": p}

    payload = {
        "model": "clef-flash",
        "answers": answers,
        "usage": {"input_tokens": est_in, "output_tokens": 0},
    }
    parsed = parse_response(payload, expected_ids=list(questions))
    cost = est_in / 1_000_000.0 * PRICE_PER_MTOK["clef-flash"]
    return parsed, Call(latency_s=0.0, input_tokens=est_in, output_tokens=0, cost_usd=cost, ok=True)


class LiveUnavailable(RuntimeError):
    """Live Clef inference cannot proceed. Carries the real reason."""


def preflight(model: str = "clef-flash", dotenv: Optional[str] = None) -> Dict[str, Any]:
    """Diagnose live-call readiness before spending anything.

    Each Workers AI failure looks identical -- {"errors":[{"code":10000,
    "message":"Authentication error"}]} -- whether the cause is a bad token, a
    wrong account, or a token without AI permission. Only the separate probes
    below tell them apart, so a caller that just sees "Authentication error"
    cannot act on it.

    Stages:
      1. tokens/verify      -> is the token real and active?
      2. accounts           -> is the account id correct for this token?
      3. ai/models/search   -> does the token carry Workers AI permission?
      4. ai/run             -> can it actually run a model?

    Returns a dict with per-stage status. Never raises for a failed probe.
    """
    load_dotenv(dotenv)
    token = os.environ.get(TOKEN_ENV)
    account = os.environ.get(ACCOUNT_ENV)
    out: Dict[str, Any] = {
        "model": model, "token_present": bool(token), "account_present": bool(account),
        "ready": False, "stages": {}, "blocker": None,
    }

    if not token or not account:
        out["blocker"] = f"missing {TOKEN_ENV} and/or {ACCOUNT_ENV}"
        return out

    base = "https://api.cloudflare.com/client/v4"

    def probe(name: str, url: str, body: Optional[Dict[str, Any]] = None) -> Any:
        data = json.dumps(body).encode() if body is not None else None
        headers = {"Authorization": f"Bearer {token}"}
        if data:
            headers["Content-Type"] = "application/json"
        r = urllib.request.Request(url, data=data, headers=headers,
                                   method="POST" if body is not None else "GET")
        try:
            with urllib.request.urlopen(r, timeout=60) as resp:
                return resp.status, json.loads(resp.read().decode())
        except urllib.error.HTTPError as exc:
            raw = exc.read().decode()
            try:
                parsed = json.loads(raw)
                msg = parsed.get("errors", [{}])[0].get("message", raw)[:200]
            except Exception:
                msg = raw[:200]
            return exc.code, {"error": msg}
        except Exception as exc:
            return None, {"error": f"{type(exc).__name__}: {exc}"[:200]}

    s, b = probe("token", f"{base}/user/tokens/verify")
    ok = s == 200
    out["stages"]["token_verify"] = {"status": s, "ok": ok}
    if not ok:
        out["blocker"] = f"token rejected by Cloudflare (HTTP {s}): {b.get('error', b)}"
        return out

    s, b = probe("account", f"{base}/accounts")
    ok = s == 200 and any(a.get("id") == account for a in b.get("result", []))
    out["stages"]["account_visible"] = {"status": s, "ok": ok}
    if s == 200 and not ok:
        names = [a.get("name") for a in b.get("result", [])]
        out["blocker"] = (f"account {account} is not visible to this token; "
                          f"token can see: {names}")
        return out
    if not ok:
        out["blocker"] = f"cannot list accounts (HTTP {s}): {b.get('error', b)}"
        return out

    s, b = probe("ai_perm", f"{base}/accounts/{account}/ai/models/search")
    ok = s == 200
    out["stages"]["workers_ai_permission"] = {"status": s, "ok": ok}
    if not ok:
        out["blocker"] = (
            f"token is valid but lacks Workers AI permission "
            f"(HTTP {s}: {b.get('error', b)}). "
            "Add it at Cloudflare dashboard -> My Profile -> API Tokens -> edit -> "
            "Permissions -> Account -> 'Workers AI' -> Read. The same error also "
            "appears for a wrong region or an AI-disabled account."
        )
        return out

    clef_visible = [m for m in (b.get("result") or [])
                    if "clef" in str(m.get("name", "")).lower()]
    out["clef_models_visible"] = [m.get("name") for m in clef_visible]

    s, b = probe("run", f"{base}/accounts/{account}/ai/run/@cf/cloudflare/{MODEL_SLUGS[model]}",
                 {"state": {"probe": True},
                  "questions": {"q": {"type": "noul", "instructions": "Is 1 equal to 1?"}}})
    ok = s == 200 and bool(b.get("success"))
    out["stages"]["run"] = {"status": s, "ok": ok}
    if not ok:
        out["blocker"] = f"Workers AI reachable but run failed (HTTP {s}): {b.get('errors', b)}"
        return out

    out["ready"] = True
    return out


def describe_preflight(pf: Dict[str, Any]) -> str:
    lines = ["Workers AI preflight:", f"  model={pf['model']} ready={pf['ready']}"]
    for stage, res in pf.get("stages", {}).items():
        lines.append(f"  {'PASS' if res['ok'] else 'FAIL'}  {stage:24s} HTTP {res['status']}")
    if pf.get("clef_models_visible"):
        lines.append(f"  clef models visible: {pf['clef_models_visible']}")
    if pf.get("blocker"):
        lines.append(f"  BLOCKER: {pf['blocker']}")
    return "\n".join(lines)


def fingerprint(payload: Any) -> str:
    """Stable short hash of a request body, for correlating results to inputs."""
    return hashlib.sha256(json.dumps(payload, sort_keys=True, default=str).encode()).hexdigest()[:16]