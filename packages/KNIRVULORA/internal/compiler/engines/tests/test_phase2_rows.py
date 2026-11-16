"""Unit tests for phase2_rows (matched per-skill row building). CPU only, no torch.

Run from this directory:  PYTHONDONTWRITEBYTECODE=1 python -m unittest test_phase2_rows
"""
import os
import unittest

import json
import tempfile

import grounded_spike as gs
import phase2_rows as pr

HERE = os.path.dirname(os.path.abspath(__file__))
ABSTAIN = gs.ABSTAIN  # imported, not copied: this test follows any change to the refusal string
RETAIN = [{"prompt": f"retain {i}", "completion": f"answer {i}"} for i in range(26)]


def fake_train():
    rows = []
    for skill, n in [("debug", 31), ("check", 62), ("document", 112), ("scope", 132)]:
        rows += [{"skill": skill, "prompt": f"{skill} q{i}", "completion": f"{skill} a{i}"} for i in range(n)]
    return rows  # 337 rows, as in run_A/labels.json


SINGLE_SHARE = pr.single_share(337 + 26 + 6, 6)  # phase-1 single adapter: 6 abstentions of 369 rows


class AbstentionCount(unittest.TestCase):
    def test_single_share_is_phase1(self):
        self.assertAlmostEqual(SINGLE_SHARE, 6 / 369)
        self.assertLess(abs(SINGLE_SHARE - 0.016), 0.001)

    def test_zero_share_gives_zero(self):
        self.assertEqual(pr.abstention_count(100, 0.0), 0)

    def test_counts_for_each_skill(self):
        # n_pos = labelled rows of the skill + 26 retention rows
        expected = {"debug": 1, "check": 1, "document": 2, "scope": 3}
        counts = {"debug": 31, "check": 62, "document": 112, "scope": 132}
        for s, n in counts.items():
            self.assertEqual(pr.abstention_count(n + 26, SINGLE_SHARE), expected[s], s)

    def test_count_is_closest_to_share(self):
        for n_pos in range(10, 400, 7):
            k = pr.abstention_count(n_pos, SINGLE_SHARE)
            err = abs(k / (n_pos + k) - SINGLE_SHARE)
            for other in (k - 1, k + 1):
                if other >= 0:
                    self.assertLessEqual(err, abs(other / (n_pos + other) - SINGLE_SHARE) + 1e-12)


class BuildMatchedRows(unittest.TestCase):
    def build(self, skill, seed=0):
        return pr.build_matched_rows(skill, fake_train(), RETAIN, pr.GENERIC_PROMPTS, SINGLE_SHARE, seed, ABSTAIN)

    def test_exact_counts_per_skill(self):
        expect = {"debug": (31, 26, 1, 58), "check": (62, 26, 1, 89),
                  "document": (112, 26, 2, 140), "scope": (132, 26, 3, 161)}
        for skill, (own, ret, ab, total) in expect.items():
            rows, info = self.build(skill)
            self.assertEqual(info["skill_rows"], own, skill)
            self.assertEqual(info["retain_rows"], ret, skill)
            self.assertEqual(info["abstain_rows"], ab, skill)
            self.assertEqual(info["total_rows"], total, skill)
            self.assertEqual(len(rows), total, skill)
            self.assertEqual(sum(r["completion"] == ABSTAIN for r in rows), ab, skill)

    def test_scope_included_and_no_other_skills(self):
        rows, _ = self.build("scope")
        own = [r for r in rows if r["prompt"].startswith("scope q")]
        self.assertEqual(len(own), 132)
        self.assertFalse(any(r["prompt"].startswith(("debug q", "check q", "document q")) for r in rows))

    def test_retention_rows_are_the_same_for_every_adapter(self):
        for skill in pr.MATCHED_SKILLS:
            rows, _ = self.build(skill)
            self.assertEqual([r for r in rows if r["prompt"].startswith("retain ")], RETAIN)

    def test_abstention_rows_use_generic_prompts_only(self):
        for skill in pr.MATCHED_SKILLS:
            rows, _ = self.build(skill)
            ab = [r["prompt"] for r in rows if r["completion"] == ABSTAIN]
            self.assertTrue(set(ab) <= set(pr.GENERIC_PROMPTS))
            self.assertEqual(len(ab), len(set(ab)), "no duplicate refusal in one adapter")

    def test_deterministic_for_seed(self):
        self.assertEqual(self.build("scope", 0)[0], self.build("scope", 0)[0])

    def test_zero_share_has_no_abstention(self):
        rows, info = pr.build_matched_rows("debug", fake_train(), RETAIN, pr.GENERIC_PROMPTS, 0.0, 0, ABSTAIN)
        self.assertEqual(info["abstain_rows"], 0)
        self.assertFalse(any(r["completion"] == ABSTAIN for r in rows))

    def test_share_reported_matches_rows(self):
        rows, info = self.build("document")
        self.assertAlmostEqual(info["abstain_share"], round(2 / 140, 4))

    def test_generic_prompts_match_phase1_source(self):
        with open(os.path.join(HERE, "grounded_spike_r2.py"), encoding="utf-8") as f:
            src = f.read()
        for p in pr.GENERIC_PROMPTS:
            self.assertIn(f'"{p}"', src)
        self.assertEqual(len(pr.GENERIC_PROMPTS), 6)


def fake_negs():
    """Labelled hard negatives per skill (same shape as labels.json "negs"), 30 per skill."""
    return [{"skill": s, "passage": f"{s}#0", "prompt": f"{s} neg {i}", "completion": ABSTAIN}
            for s in ["debug", "check", "document", "scope"] for i in range(30)]


class StrongerRows(unittest.TestCase):
    def build(self, skill, negs=None, excluded=(), share=pr.STRONGER_SHARE, train=None, seed=0):
        return pr.build_stronger_rows(skill, fake_train() if train is None else train, RETAIN,
                                      fake_negs() if negs is None else negs, pr.GENERIC_PROMPTS, share, seed,
                                      ABSTAIN, excluded_prompts=excluded)

    def test_abstention_rows_are_skill_specific(self):
        for skill in pr.MATCHED_SKILLS:
            rows, info = self.build(skill)
            ab = [r["prompt"] for r in rows if r["completion"] == ABSTAIN]
            self.assertEqual(len(ab), info["abstain_rows"])
            self.assertTrue(all(p.startswith(f"{skill} neg ") for p in ab), skill)
            self.assertEqual(info["abstain_generic"], 0)

    def test_share_is_at_target_within_rounding(self):
        for skill in pr.MATCHED_SKILLS:
            rows, info = self.build(skill)
            self.assertLessEqual(info["abstain_share"], pr.STRONGER_SHARE + 0.01, skill)
            self.assertGreaterEqual(info["abstain_share"], pr.STRONGER_SHARE - 0.01, skill)

    def test_generic_fill_only_when_negatives_run_out(self):
        few = [n for n in fake_negs() if n["skill"] == "debug"][:2]
        rows, info = self.build("debug", negs=few, share=0.2)
        self.assertEqual(info["abstain_skill_negs"], 2)
        self.assertGreater(info["abstain_generic"], 0)
        generic_used = [r["prompt"] for r in rows if r["completion"] == ABSTAIN and not r["prompt"].startswith("debug neg")]
        self.assertTrue(set(generic_used) <= set(pr.GENERIC_PROMPTS))

    def test_excluded_eval_prompt_is_never_an_abstention_row(self):
        negs = fake_negs() + [{"skill": "debug", "passage": "debug#1", "prompt": "debug neg 0 ", "completion": ABSTAIN}]
        rows, info = self.build("debug", negs=negs, excluded=["DEBUG NEG 0"])
        self.assertEqual(info["neg_dropped_overlap"], 1)
        self.assertEqual(pr.overlap_prompts(rows, [{"prompt": "debug neg 0"}], []), [])

    def test_eval_overlapping_training_row_is_dropped(self):
        train = fake_train() + [{"skill": "check", "prompt": "held check q", "completion": "x"}]
        rows, info = self.build("check", train=train, excluded=["held check q"])
        self.assertEqual(info["own_dropped_overlap"], 1)
        self.assertEqual(info["skill_rows"], 62)

    def test_seed_is_deterministic(self):
        self.assertEqual(self.build("scope", seed=3)[0], self.build("scope", seed=3)[0])

    def test_seed_changes_abstention_sample(self):
        # pool (30 negatives per skill) is larger than the abstention count, so the seed picks the rows.
        # Validator M4c (rng ignores seed) made seeds 1 and 2 identical; this must differ.
        for skill in pr.MATCHED_SKILLS:
            a = [r["prompt"] for r in self.build(skill, seed=1)[0] if r["completion"] == ABSTAIN]
            b = [r["prompt"] for r in self.build(skill, seed=2)[0] if r["completion"] == ABSTAIN]
            self.assertEqual(len(a), len(b))
            self.assertNotEqual(sorted(a), sorted(b), skill)

    def test_overlap_guard_detects_shared_prompt(self):
        rows = [{"prompt": "Same?", "completion": "a"}, {"prompt": "other", "completion": "b"}]
        self.assertEqual(pr.overlap_prompts(rows, [{"prompt": "same?  "}], []), ["same?"])
        self.assertEqual(pr.overlap_prompts(rows, [{"prompt": "unrelated"}], [{"prompt": "nope"}]), [])

    def test_real_labels_have_no_overlap_after_exclusion(self):
        labels = os.path.expanduser("~/ori_knirv/scratch_I/run_A/labels.json")
        if not os.path.exists(labels):
            self.skipTest("labels.json not present")
        with open(labels, encoding="utf-8") as f:
            L = json.load(f)
        ev = [x["prompt"] for x in L["held"] + L["probes"]]
        for skill in pr.MATCHED_SKILLS:
            rows, _ = pr.build_stronger_rows(skill, L["train"], RETAIN, L["negs"], pr.GENERIC_PROMPTS,
                                             pr.STRONGER_SHARE, 0, ABSTAIN, excluded_prompts=ev)
            self.assertEqual(pr.overlap_prompts(rows, L["held"], L["probes"]), [], skill)


class RetentionPinning(unittest.TestCase):
    def test_generated_rows_are_written_and_reused(self):
        made = []

        def make():
            made.append(1)
            return [{"prompt": "p0", "completion": "c0"}, {"prompt": "p1", "completion": "c1"}]

        with tempfile.TemporaryDirectory() as d:
            out = os.path.join(d, "retention_rows.json")
            rows, src = pr.retention_rows("", out, make)
            self.assertEqual(src, "generated")
            self.assertEqual(len(made), 1)
            again, src2 = pr.retention_rows(out, os.path.join(d, "copy.json"), lambda: self.fail("must not regenerate"))
            self.assertEqual(src2, "reused")
            self.assertEqual(again, rows)
            with open(os.path.join(d, "copy.json"), encoding="utf-8") as f:
                self.assertEqual(json.load(f), rows)

    def test_bad_file_is_refused(self):
        with tempfile.TemporaryDirectory() as d:
            bad = os.path.join(d, "bad.json")
            with open(bad, "w", encoding="utf-8") as f:
                json.dump([{"prompt": "only prompt"}], f)
            with self.assertRaises(ValueError):
                pr.retention_rows(bad, "", lambda: [])


if __name__ == "__main__":
    unittest.main()
