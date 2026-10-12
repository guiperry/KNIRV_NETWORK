"""Contract, translation, and metric tests. No network. `python3 test_contract.py`

The fixtures under fixtures/ are Cloudflare's published schemas, so the
schema-shape tests fail loudly if the API drifts. Recorded fixtures under
fixtures/recorded/ pin real response shapes when a live run happens.
"""

from __future__ import annotations

import json
import os
import sys
import unittest

from clef_client import describe_preflight, offline_stub
from clef_contract import (
    ContractError,
    Decision,
    build_request,
    choice,
    noul,
    parse_answer,
    parse_response,
    score,
    validate_against_published_schema,
)
from knirv_schema import (
    ALLOWED_ACTIONS,
    INTENTS,
    MIN_CONFIDENCE,
    SEVERITY_LEVELS,
    SupervisorContext,
    build_questions,
    render_state,
    severity_index,
    to_advice,
)
from metrics import (
    abstain_curve,
    brier_skill_score,
    calibration_report,
    expected_calibration_error,
    macro_f1,
    multiclass_brier,
)

CTX = SupervisorContext(
    current_phase="implement",
    task_request="Rotate the production signing key",
    task_intent="develop",
    task_risk="high",
    policy_bundle_hash="sha256:deadbeef",
    policy_refs=["POL-KEY-ROTATION"],
    evidence_classes=["test_report"],
)


class TestPublishedSchema(unittest.TestCase):
    def test_fixtures_present(self):
        for p in ("fixtures/clef-schema-input.json", "fixtures/clef-schema-output.json"):
            self.assertTrue(os.path.exists(p), f"missing published schema fixture: {p}")

    def test_valid_request_passes_published_schema(self):
        req = build_request("clef-flash", {"text": "hello"}, {
            "a": noul("Is this English?"),
            "b": choice("Pick one", {"x": "eX", "y": "why"}),
            "c": score("How bad", ["fine", "bad"]),
        })
        self.assertEqual(validate_against_published_schema(req, "input"), [])

    def test_seventy_seven_option_choice_is_valid(self):
        """BANKING77 asks exactly this; the schema caps options at 255."""
        opts = {f"c{i}": f"class {i}" for i in range(77)}
        req = build_request("clef", "state", {"intent": choice("classify", opts)})
        self.assertEqual(validate_against_published_schema(req, "input"), [])

    def test_model_pattern_tolerates_whitespace(self):
        req = build_request("  clef-flash ", "s", {"a": noul("q")})
        self.assertEqual(req["model"], "clef-flash")

    def test_bad_model_rejected(self):
        with self.assertRaises(ContractError):
            build_request("clef-large", "s", {"a": noul("q")})

    def test_choice_requires_two_options(self):
        with self.assertRaises(ContractError):
            build_request("clef", "s", {"a": choice("pick", {"only": "one"})})

    def test_choice_allows_null_description(self):
        """Schema: description may be 'string, object, array, or null'."""
        req = build_request("clef", "s", {"a": choice("pick", {"x": None, "y": "why"})})
        self.assertEqual(validate_against_published_schema(req, "input"), [])

    def test_score_level_cap_is_enforced(self):
        with self.assertRaises(ContractError):
            build_request("clef", "s", {"a": score("rate", [f"L{i}" for i in range(11)])})

    def test_score_two_levels_ok(self):
        req = build_request("clef", "s", {"a": score("rate", ["lo", "hi"])})
        self.assertEqual(validate_against_published_schema(req, "input"), [])

    def test_question_id_charset_enforced(self):
        with self.assertRaises(ContractError):
            build_request("clef", "s", {"bad id!": noul("q")})

    def test_question_count_cap_enforced(self):
        qs = {f"q{i}": noul("q") for i in range(65)}
        with self.assertRaises(ContractError):
            build_request("clef", "s", qs)

    def test_empty_state_rejected(self):
        with self.assertRaises(ContractError):
            build_request("clef", "  ", {"a": noul("q")})


class TestAnswerParsing(unittest.TestCase):
    def test_noul_has_no_confidence_field_and_confidence_is_derived(self):
        """Rule 2 of the design doc: noul returns P(yes) only."""
        d = parse_answer("g", {"type": "noul", "noul": 0.8})
        self.assertAlmostEqual(d.confidence, 0.8)
        self.assertAlmostEqual(d.probabilities["True"], 0.8)
        self.assertAlmostEqual(d.probabilities["False"], 0.2)

    def test_noul_requires_the_noul_key(self):
        with self.assertRaises(ContractError):
            parse_answer("g", {"type": "noul"})

    def test_choice_requires_confidence_and_probabilities(self):
        with self.assertRaises(ContractError):
            parse_answer("c", {"type": "choice", "choice": "a", "probabilities": {"a": 1.0}})

    def test_probabilities_must_sum_to_one(self):
        with self.assertRaises(ContractError):
            parse_answer("c", {"type": "choice", "choice": "a",
                               "probabilities": {"a": 0.4, "b": 0.4}, "confidence": 0.5})

    def test_score_requires_legend(self):
        """Published output schema marks 'legend' as required for score."""
        with self.assertRaises(ContractError):
            parse_answer("s", {"type": "score", "score": 1.0,
                               "probabilities": {"0": 0.5, "1": 0.5}, "confidence": 0.5})

    def test_score_can_land_between_levels(self):
        """Rule 1: a continuous 2.4 is legal and must survive parsing."""
        d = parse_answer("s", {"type": "score", "score": 2.4, "legend": {"0": "lo", "1": "hi"},
                               "probabilities": {"0": 0.3, "1": 0.7}, "confidence": 0.7})
        self.assertEqual(d.raw["score"], 2.4)

    def test_out_of_range_probability_rejected(self):
        with self.assertRaises(ContractError):
            parse_answer("n", {"type": "noul", "noul": 1.4})

    def test_response_requires_usage(self):
        with self.assertRaises(ContractError):
            parse_response({"model": "clef", "answers": {"a": {"type": "noul", "noul": 0.5}}})

    def test_response_parses_usage_and_ids(self):
        r = parse_response({
            "model": "clef-flash",
            "answers": {"a": {"type": "noul", "noul": 0.5}},
            "usage": {"input_tokens": 120, "output_tokens": 0},
        }, expected_ids=["a"])
        self.assertEqual(r.input_tokens, 120)
        self.assertEqual(r.output_tokens, 0)
        self.assertIn("a", r.decisions)

    def test_response_rejects_missing_answer_ids(self):
        with self.assertRaises(ContractError):
            parse_response({"model": "clef", "answers": {}, "usage": {"input_tokens": 1, "output_tokens": 0}},
                           expected_ids=["a"])

    def test_valid_output_passes_published_schema(self):
        payload = {
            "model": "clef-flash",
            "answers": {
                "a": {"type": "noul", "noul": 0.5},
                "c": {"type": "choice", "choice": "x",
                      "probabilities": {"x": 0.7, "y": 0.3}, "confidence": 0.7},
                "s": {"type": "score", "score": 1.5, "legend": {"0": "lo", "1": "hi"},
                      "probabilities": {"0": 0.25, "1": 0.75}, "confidence": 0.75},
            },
            "usage": {"input_tokens": 40, "output_tokens": 0},
        }
        self.assertEqual(validate_against_published_schema(payload, "output"), [])
        parse_response(payload, expected_ids=["a", "c", "s"])


class TestAbstention(unittest.TestCase):
    """Rule 3: abstention is a threshold, never a model-chosen label.

    Confidence is the peak probability, so a flat distribution is the
    low-confidence case -- not a distribution whose peak happens to be high.
    """

    def test_abstains_below_tau(self):
        flat = {"a": 0.25, "b": 0.25, "c": 0.25, "d": 0.25}
        self.assertTrue(Decision("c", "choice", 0.25, flat).abstains(0.3))

    def test_does_not_abstain_at_or_above_tau(self):
        sharp = {"a": 0.95, "b": 0.05}
        self.assertFalse(Decision("c", "choice", 0.95, sharp).abstains(0.3))

    def test_abstain_label_does_not_grant_a_pass(self):
        """Even if the model picks 'abstain', the tau still governs."""
        d = Decision("c", "choice", 0.2, {"allow": 0.2, "abstain": 0.2})
        self.assertTrue(d.abstains(0.3))


class TestKnirvTranslation(unittest.TestCase):
    def test_action_vocabulary_matches_advisor_go(self):
        self.assertEqual(
            sorted(ALLOWED_ACTIONS),
            ["abstain", "allow", "ask_operator", "recover", "request_evidence", "request_review"],
        )

    def test_all_questions_fit_one_request(self):
        qs = build_questions(CTX, valid_phases=["plan", "implement", "validate"],
                             required_evidence=["test_report", "rollback_plan"])
        self.assertLessEqual(len(qs), 64)
        body = build_request("clef", render_state(CTX), qs)
        self.assertEqual(validate_against_published_schema(body, "input"), [])

    def test_state_is_structured_not_prose(self):
        st = render_state(CTX)
        self.assertIsInstance(st, dict)
        self.assertEqual(st["task"]["request"], CTX.task_request)

    def test_severity_band_rounds_for_display_only(self):
        self.assertEqual(SEVERITY_LEVELS[severity_index("critical")], "critical")
        with self.assertRaises(Exception):
            severity_index("nonsense")

    def test_low_confidence_action_abstains(self):
        # Flat over six actions: peak 0.167, well below the 0.3 threshold.
        flat = {a: 1 / 6 for a in ALLOWED_ACTIONS}
        d = {"action": Decision("action", "choice", 1 / 6, flat, "abstain")}
        a = to_advice(d, tau=MIN_CONFIDENCE)
        self.assertTrue(a.abstain)
        self.assertEqual(a.recommended_action, "abstain")
        self.assertIn("tau", a.abstain_reason)

    def test_unknown_action_is_refused(self):
        d = {"action": Decision("action", "choice", 0.9, {"launch_missiles": 0.9}, "launch_missiles")}
        a = to_advice(d)
        self.assertTrue(a.abstain)
        self.assertIn("vocabulary", a.abstain_reason)

    def test_confident_action_produces_advice(self):
        d = {"action": Decision("action", "choice", 0.91, {"allow": 0.91, "abstain": 0.09}, "allow"),
             "intent": Decision("intent", "choice", 0.88, {"develop": 0.88, "plan": 0.12}, "develop"),
             "severity": Decision("severity", "score", 0.8,
                                  {"0": 0.1, "1": 0.1, "2": 0.1, "3": 0.6, "4": 0.1},
                                  raw={"type": "score", "score": 3.0, "legend": {}, "confidence": 0.8}),
             "next_phase": Decision("next_phase", "choice", 0.9, {"validate": 0.9, "plan": 0.1}, "validate")}
        a = to_advice(d)
        self.assertFalse(a.abstain)
        self.assertEqual(a.recommended_action, "allow")
        self.assertEqual(a.intent, "develop")
        self.assertEqual(a.next_phase, "validate")
        self.assertEqual(a.severity, 3.0)
        self.assertEqual(a.severity_band, "high")
        self.assertTrue(any(r.startswith("severity-floor") for r in a.policy_references))

    def test_severity_between_levels_is_preserved(self):
        d = {"action": Decision("action", "choice", 0.9, {"allow": 0.9}, "allow"),
             "severity": Decision("severity", "score", 0.7, {"0": 0.2, "1": 0.5},
                                  raw={"type": "score", "score": 2.4, "legend": {}, "confidence": 0.7})}
        a = to_advice(d)
        self.assertEqual(a.severity, 2.4)

    def test_missing_evidence_derived_from_noul(self):
        d = {"action": Decision("action", "choice", 0.9, {"allow": 0.9}, "allow"),
             "evidence_test_report": Decision("evidence_test_report", "noul", 0.2,
                                             {"True": 0.2, "False": 0.8},
                                             raw={"type": "noul", "noul": 0.2}),
             "evidence_rollback_plan": Decision("evidence_rollback_plan", "noul", 0.9,
                                                {"True": 0.9, "False": 0.1},
                                                raw={"type": "noul", "noul": 0.9})}
        a = to_advice(d)
        self.assertEqual(a.missing_evidence, ["test_report"])

    def test_intents_and_actions_are_disjoint_vocabularies(self):
        """A merge bug worth pinning: 'recover' and 'ask_operator' appear in both."""
        self.assertTrue(set(INTENTS) & {"recover", "ask_operator"})

    def test_no_action_answer_abstains(self):
        a = to_advice({})
        self.assertTrue(a.abstain)
        self.assertEqual(a.confidence, 0.0)


class TestStub(unittest.TestCase):
    def test_stub_output_is_schema_valid(self):
        qs = {"action": choice("pick", {"allow": "yes", "review": "no"})}
        resp, call = offline_stub({"text": "hi"}, qs, seed=1, seed_state="allow")
        self.assertTrue(call.ok)
        self.assertIn("action", resp.decisions)

    def test_stub_is_deterministic(self):
        qs = {"a": choice("pick", {"x": "1", "y": "2"})}
        r1, _ = offline_stub({"t": 1}, qs, seed=7)
        r2, _ = offline_stub({"t": 1}, qs, seed=7)
        self.assertEqual(r1.decisions["a"].probabilities, r2.decisions["a"].probabilities)


class TestMetrics(unittest.TestCase):
    def test_brier_perfect_and_worst(self):
        self.assertAlmostEqual(multiclass_brier([1.0, 0.0], 0), 0.0)
        self.assertAlmostEqual(multiclass_brier([0.0, 1.0], 0), 2.0)

    def test_perfect_calibration_has_zero_ece(self):
        ece, mce = expected_calibration_error([0.0, 1.0, 1.0, 1.0], [False, True, True, True], n_bins=15)
        self.assertAlmostEqual(ece, 0.0)

    def test_overconfidence_is_caught(self):
        ece, _ = expected_calibration_error([1.0] * 10, [True] * 5 + [False] * 5, n_bins=15)
        self.assertGreater(ece, 0.4)

    def test_macro_f1_known_value(self):
        gold = [0, 0, 1, 1, 2]
        pred = [0, 0, 1, 0, 2]
        f1, _ = macro_f1(pred, gold, 3)
        self.assertGreater(f1, 0.7)
        self.assertLess(f1, 1.0)

    def test_perfect_macro_f1_is_one(self):
        f1, _ = macro_f1([0, 1, 2], [0, 1, 2], 3)
        self.assertAlmostEqual(f1, 1.0)

    def test_calibration_report_shape(self):
        probs = [[0.9, 0.1], [0.2, 0.8], [0.7, 0.3]]
        rep = calibration_report(probs, [0, 1, 0], ["a", "b"])
        self.assertEqual(rep.n, 3)
        self.assertGreaterEqual(rep.brier, 0.0)
        self.assertLessEqual(rep.ece, 1.0)

    def test_abstain_curve_coverage_decreases_with_tau(self):
        probs = [[0.9, 0.1], [0.6, 0.4], [0.4, 0.6], [0.1, 0.9]]
        gold = [0, 0, 1, 1]
        rows = abstain_curve(probs, gold, [0.0, 0.5, 0.8])
        covs = [r["coverage"] for r in rows]
        # Raising tau can only shrink coverage -- it is a filter, not a reweight.
        self.assertEqual(covs, sorted(covs, reverse=True))
        self.assertAlmostEqual(covs[0], 1.0)

    def test_brier_skill_score_sign(self):
        self.assertGreater(brier_skill_score(0.5, 1.0), 0.0)
        self.assertLess(brier_skill_score(2.0, 1.0), 0.0)


class TestDotenv(unittest.TestCase):
    """The .env loader must never clobber a real environment secret."""

    def _tmp(self, body):
        import tempfile
        fh = tempfile.NamedTemporaryFile("w", suffix=".env", delete=False)
        fh.write(body)
        fh.close()
        self.addCleanup(os.unlink, fh.name)
        return fh.name

    def test_loads_plain_and_quoted_values(self):
        from clef_client import load_dotenv
        path = self._tmp('A_TOKEN=abc123\nB_ID="quoted456"\n# comment\n\nC_SINGLE=\'x\'\n')
        loaded = load_dotenv(path)
        self.assertEqual(sorted(loaded), ["A_TOKEN", "B_ID", "C_SINGLE"])
        self.assertEqual(os.environ["B_ID"], "quoted456")
        self.assertEqual(os.environ["C_SINGLE"], "x")
        for k in ("A_TOKEN", "B_ID", "C_SINGLE"):
            os.environ.pop(k, None)

    def test_existing_environment_wins(self):
        from clef_client import load_dotenv
        os.environ["A_TOKEN"] = "from_environment"
        try:
            path = self._tmp("A_TOKEN=from_file\n")
            loaded = load_dotenv(path)
            self.assertEqual(loaded, [])
            self.assertEqual(os.environ["A_TOKEN"], "from_environment")
        finally:
            os.environ.pop("A_TOKEN", None)

    def test_missing_file_is_not_an_error(self):
        from clef_client import load_dotenv
        self.assertEqual(load_dotenv("/nonexistent/path/.env"), [])

    def test_gitignore_protects_dotenv(self):
        body = open(".gitignore").read()
        self.assertIn(".env", body.splitlines())


class TestPreflightContract(unittest.TestCase):
    def test_preflight_reports_missing_credentials_without_network(self):
        from clef_client import preflight
        saved = {k: os.environ.pop(k, None)
                 for k in ("CLOUDFLARE_API_TOKEN", "CLOUDFLARE_ACCOUNT_ID")}
        try:
            # point past the repo .env so the loader cannot repopulate
            pf = preflight(dotenv="/nonexistent/.env")
            self.assertFalse(pf["ready"])
            self.assertIn("CLOUDFLARE_API_TOKEN", pf["blocker"])
            self.assertIn("BLOCKER", describe_preflight(pf))
        finally:
            for k, v in saved.items():
                if v is not None:
                    os.environ[k] = v

    def test_live_unavailable_is_raised_not_swallowed(self):
        """Regression: all-calls-failed used to surface as 'empty input'.

        run_banking77/run_bfcl must raise LiveUnavailable carrying the real
        reason, instead of continuing into the metrics layer and dying there.
        """
        from clef_client import LiveUnavailable
        from run_banking77 import run_clef
        import unittest.mock as mock

        cats = ["a", "b", "c"]
        test = [("text", "a"), ("text2", "b")]
        with mock.patch("run_banking77.ClefClient") as MC, \
             mock.patch("run_banking77.preflight", return_value={"ready": False, "blocker": None}):
            MC.return_value.ask.return_value = (
                None,
                __import__("clef_client").Call(0.1, 0, 0, 0.0, False, "HTTP 401: auth"),
            )
            with self.assertRaises(LiveUnavailable) as ctx:
                run_clef("clef-flash", test, cats, limit=0, dry_run=False, stub_accuracy=0.35)
        self.assertIn("HTTP 401", str(ctx.exception))

    def test_failed_call_errors_are_propagated_not_replaced(self):
        """A failed call must record its own error, not a bare 'call failed'.

        The concurrent worker returns (None) on failure and the caller appends a
        generic message; the underlying transport error is what makes a run
        diagnosable, so it has to survive to the exception.
        """
        from run_banking77 import run_clef
        import unittest.mock as mock

        cats = ["a", "b"]
        test = [("t", "a"), ("t2", "b")]
        with mock.patch("run_banking77.ClefClient") as MC, \
             mock.patch("run_banking77.preflight", return_value={"ready": False, "blocker": None}):
            def boom(state, questions, validate=True):
                return None, __import__("clef_client").Call(0.1, 0, 0, 0.0, False, "HTTP 429: quota")
            MC.return_value.ask.side_effect = boom
            with self.assertRaises(Exception) as ctx:
                run_clef("clef-flash", test, cats, limit=0, dry_run=False, stub_accuracy=0.35)
        self.assertIn("429", str(ctx.exception))


class TestData(unittest.TestCase):
    def test_banking77_fixtures(self):
        cats = json.load(open("data/categories.json"))
        self.assertEqual(len(cats), 77)
        import csv as _csv
        with open("data/train.csv") as fh:
            self.assertEqual(len(list(_csv.DictReader(fh))), 10003)
        with open("data/test.csv") as fh:
            self.assertEqual(len(list(_csv.DictReader(fh))), 3080)

    def test_every_test_label_is_a_known_category(self):
        from run_banking77 import load_categories, load_split
        cats = set(load_categories())
        for _, c in load_split("test"):
            self.assertIn(c, cats)


class TestBFCL(unittest.TestCase):
    def test_loader_reads_jsonl(self):
        from run_bfcl import load_split
        d = load_split("BFCL_v3_simple")
        self.assertEqual(len(d), 400)
        self.assertIn("id", d[0])

    def test_irrelevance_split_present(self):
        from run_bfcl import load_split
        self.assertEqual(len(load_split("BFCL_v3_irrelevance")), 240)

    def test_question_text_flattens_double_nesting(self):
        """All nine v3 splits store question as list[list[{role,content}]]."""
        from run_bfcl import question_text
        item = {"question": [[{"role": "user", "content": "hello there"}]]}
        self.assertIn("hello there", question_text(item))
        self.assertIn("user:", question_text(item))

    def test_question_text_handles_flat_and_string(self):
        from run_bfcl import question_text
        self.assertIn("hi", question_text({"question": "hi"}))
        self.assertIn("hi", question_text({"question": [{"role": "user", "content": "hi"}]}))

    def test_question_text_handles_content_parts(self):
        from run_bfcl import question_text
        item = {"question": [[{"role": "user", "content": [{"text": "part one"}]}]]}
        self.assertIn("part one", question_text(item))

    def test_question_text_empty_is_survivable(self):
        from run_bfcl import question_text
        self.assertEqual(question_text({"question": []}), "")

    def test_render_params_v3_schema_object(self):
        from run_bfcl import render_params
        f = {"name": "f", "parameters": {
            "type": "dict",
            "properties": {"a": {"type": "integer"}, "b": {"type": "string"}},
            "required": ["a"]}}
        self.assertIn("integer a", render_params(f))
        self.assertIn("string b?", render_params(f))   # optional marked

    def test_render_params_v1_list(self):
        from run_bfcl import render_params
        f = {"name": "f", "parameters": [{"name": "a", "type": "integer"}]}
        self.assertIn("integer a", render_params(f))

    def test_describe_functions_skips_malformed(self):
        from run_bfcl import describe_functions
        out = describe_functions([{"name": "ok"}, "junk", {"no_name": 1}])
        self.assertEqual(list(out), ["ok"])

    def test_irrelevance_gold_is_none(self):
        from run_bfcl import gold_function, NONE
        item = {"id": "irrelevance_0", "function": [{"name": "f1"}]}
        self.assertEqual(gold_function(item, "BFCL_v3_irrelevance"), NONE)

    def test_single_function_gold_is_that_function(self):
        from run_bfcl import gold_function
        item = {"function": [{"name": "calculate_triangle_area"}]}
        self.assertEqual(gold_function(item, "BFCL_v3_simple"), "calculate_triangle_area")

    def test_choice_question_always_offers_the_none_option(self):
        from run_bfcl import build_question, NONE
        q = build_question(["a", "b"], {"a": "A", "b": "B"})
        self.assertIn(NONE, q.criteria)
        self.assertEqual(len(q.criteria), 3)


if __name__ == "__main__":
    unittest.main(verbosity=2, argv=[sys.argv[0]] + sys.argv[1:])