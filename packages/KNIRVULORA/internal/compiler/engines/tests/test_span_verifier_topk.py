"""Unit tests for the top-k span gate (I_52 19a). Run from this directory:
PYTHONDONTWRITEBYTECODE=1 python3 -m unittest test_span_verifier_topk
"""
import unittest

import derive_facts as df
import span_verifier as sv

P1 = "Run `docker ps -a` to list containers. Logs go to /var/log/app."
P2 = "Set NRV_LOG_LEVEL=debug in .env before restarting."
P3 = "The cache directory is ~/.cache/knirv and is cleared weekly."
PASSAGES = [P1, P2, P3]


class TopKGate(unittest.TestCase):
    def test_k1_single_passage_matches_old_supported(self):
        # with one passage the top-k gate must agree with the original top-1 gate, answer by answer
        answers = [
            "Run `docker ps -a` to list containers.",
            "Set NRV_LOG_LEVEL=debug in .env.",
            "Use `docker ps -a` and set NRV_LOG_LEVEL=debug.",
            "Plain prose with no spans at all.",
            "",
            "   ",
            "Look in /var/log/app for logs.",
        ]
        for ans in answers:
            for p in PASSAGES:
                self.assertEqual(sv.supported_topk(ans, [p]), sv.supported(ans, p), (ans, p))
                self.assertEqual(sv.unsupported_spans_topk(ans, [p]), sv.unsupported_spans(ans, p), (ans, p))

    def test_k3_accepts_span_found_only_in_passage_2(self):
        ans = "Set NRV_LOG_LEVEL=debug in .env."
        self.assertFalse(sv.supported(ans, P1))           # top-1 alone refuses
        self.assertTrue(sv.supported_topk(ans, [P1, P2]))  # found in passage 2
        self.assertTrue(sv.supported_topk(ans, [P1, P2, P3]))

    def test_k3_accepts_span_found_only_in_passage_3(self):
        ans = "The cache is at ~/.cache/knirv."
        self.assertFalse(sv.supported(ans, P1))
        self.assertFalse(sv.supported(ans, P2))
        self.assertTrue(sv.supported_topk(ans, [P1, P2, P3]))

    def test_k3_rejects_span_found_in_none(self):
        ans = "Run `kubectl apply` now."
        self.assertTrue(sv.unsupported_spans_topk(ans, PASSAGES))
        self.assertFalse(sv.supported_topk(ans, PASSAGES))

    def test_every_span_must_be_grounded_somewhere(self):
        # one span in passage 2, one in none: the answer is refused
        ans = "Set NRV_LOG_LEVEL=debug and run `kubectl apply`."
        self.assertFalse(sv.supported_topk(ans, PASSAGES))
        bad = sv.unsupported_spans_topk(ans, PASSAGES)
        self.assertTrue(any("kubectl" in b for b in bad))
        self.assertFalse(any("NRV_LOG_LEVEL" in b for b in bad))

    def test_boundary_rule_still_applies_across_passages(self):
        # `.env` must not match inside `.envrc` in any passage
        ans = "Edit `.env` now."
        self.assertEqual(sv.unsupported_spans_topk(ans, ["Use .envrc here.", "nothing else"]), [".env"])
        self.assertTrue(sv.supported_topk(ans, ["Use .envrc here.", "Edit .env now."]))

    def test_no_span_answer_vacuously_supported_and_empty_refused(self):
        self.assertTrue(sv.supported_topk("Plain prose.", PASSAGES))
        self.assertFalse(sv.supported_topk("", PASSAGES))
        self.assertFalse(sv.supported_topk("  ", PASSAGES))

    def test_duplicate_span_counted_once(self):
        self.assertEqual(len(sv.unsupported_spans_topk("Use `kubectl` and `kubectl`.", PASSAGES)), 1)


class BestPassageIndices(unittest.TestCase):
    def setUp(self):
        self.tf = df.TfIdf(PASSAGES)

    def test_k1_equals_best_passage_index(self):
        for q in ["how do I list containers with docker", "where are the logs", "log level setting",
                  "cache directory", "zzzz unrelated words", ""]:
            self.assertEqual(sv.best_passage_indices(self.tf, q, 1), [sv.best_passage_index(self.tf, q)], q)

    def test_k3_returns_all_three_best_first(self):
        idx = sv.best_passage_indices(self.tf, "docker containers", 3)
        self.assertEqual(sorted(idx), [0, 1, 2])
        self.assertEqual(idx[0], sv.best_passage_index(self.tf, "docker containers"))


if __name__ == "__main__":
    unittest.main()
