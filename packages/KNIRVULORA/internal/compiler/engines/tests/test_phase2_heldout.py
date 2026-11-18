"""Unit tests for phase2_heldout (I_52 Q35: held-out passages per skill). CPU only, no torch.

Run from this directory:  PYTHONDONTWRITEBYTECODE=1 python3 -m unittest test_phase2_heldout
The split test on the real labels is skipped if the labels file is absent.
"""
import json
import os
import unittest

import grounded_spike as gs
import phase2_heldout as ho
import phase2_rows as pr

ABSTAIN = gs.ABSTAIN
LABELS = os.path.expanduser("~/ori_knirv/scratch_I/run_A/labels.json")
RETAIN = [{"prompt": f"retain {i}", "completion": f"answer {i}"} for i in range(26)]


def fake_labels():
    """Small labelled set: 4 skills, passages p1..p4 per skill, questions on every passage."""
    L = {"train": [], "negs": [], "held": [], "probes": []}
    for skill, npass in [("debug", 4), ("check", 5), ("document", 10), ("scope", 10)]:
        for j in range(npass):
            pid = f"{skill}#{j}"
            L["train"].append({"passage": pid, "skill": skill, "prompt": f"{pid} train q", "completion": "c"})
            L["held"].append({"passage": pid, "skill": skill, "prompt": f"{pid} held q"})
            L["probes"].append({"passage": pid, "skill": skill, "prompt": f"{pid} probe q"})
    return L


class SplitPassages(unittest.TestCase):
    def setUp(self):
        self.ids = sorted(f"{s}#{j}" for s, n in [("debug", 13), ("check", 15), ("document", 29), ("scope", 30)]
                          for j in range(n))

    def test_counts_per_skill(self):
        h, counts = ho.split_passages(self.ids, frac=1 / 3, seed=0)
        self.assertEqual({s: counts[s]["held_out"] for s in counts}, {"check": 5, "debug": 4, "document": 10, "scope": 10})
        self.assertEqual({s: counts[s]["train"] for s in counts}, {"check": 10, "debug": 9, "document": 19, "scope": 20})
        self.assertEqual(sum(len(v) for v in h.values()), 29)

    def test_reproducible_and_seed_sensitive(self):
        a, _ = ho.split_passages(self.ids, seed=0)
        b, _ = ho.split_passages(list(reversed(self.ids)), seed=0)  # input order must not matter
        c, _ = ho.split_passages(self.ids, seed=1)
        self.assertEqual(a, b)
        self.assertNotEqual(a, c)

    def test_held_out_ids_belong_to_their_skill(self):
        h, _ = ho.split_passages(self.ids, seed=0)
        for skill, ids in h.items():
            for pid in ids:
                self.assertEqual(ho.skill_of(pid), skill)

    def test_at_least_one_passage_per_skill_and_never_all(self):
        # a skill with one passage keeps it for training (never all passages held out)
        h, counts = ho.split_passages(["debug#0", "debug#1", "check#0"], frac=0.99, seed=0)
        self.assertEqual(counts["check"]["held_out"], 0)
        self.assertEqual(counts["debug"]["held_out"], 1)
        self.assertEqual(h["check"], [])


class SplitLabels(unittest.TestCase):
    def test_zero_overlap_and_eval_only_on_heldout(self):
        L = fake_labels()
        h, _ = ho.split_passages(ho.all_passage_ids(L), seed=0)
        H = ho.heldout_set(h)
        parts = ho.split_labels(L, H)
        self.assertEqual(ho.check_split(h, parts["train_rows"], parts["train_heldout_rows"], [],
                                        parts["eval_held"], parts["eval_probes"]), [])
        train_p = {r["passage"] for r in parts["train_rows"]}
        self.assertFalse(train_p & H)
        for skill in h:
            train_skill = {r["passage"] for r in parts["train_rows"] if r["skill"] == skill}
            self.assertFalse(train_skill & set(h[skill]), skill)
        self.assertTrue(all(r["passage"] in H for r in parts["eval_held"] + parts["eval_probes"]))
        self.assertEqual(len(parts["eval_held"]) + len(parts["other_held"]), len(L["held"]))

    def test_detects_training_row_on_heldout_passage(self):
        L = fake_labels()
        h, _ = ho.split_passages(ho.all_passage_ids(L), seed=0)
        H = ho.heldout_set(h)
        leaked = list(L["train"]) + [{"passage": sorted(H)[0], "skill": "x", "prompt": "leak", "completion": "c"}]
        problems = ho.check_split(h, leaked, [], [], [], [])
        self.assertTrue(problems)

    def test_detects_eval_question_on_training_passage(self):
        L = fake_labels()
        h, _ = ho.split_passages(ho.all_passage_ids(L), seed=0)
        H = ho.heldout_set(h)
        train_pid = next(p for p in ho.all_passage_ids(L) if p not in H)
        problems = ho.check_split(h, [], [], [], [{"passage": train_pid, "prompt": "q"}], [])
        self.assertTrue(problems)


class BuildRows(unittest.TestCase):
    def test_plain_recipe_rows_and_leak_guard(self):
        train = [{"prompt": f"t{i}", "completion": f"a{i}"} for i in range(10)] + [{"prompt": "SAME Q", "completion": "x"}]
        rows, info = ho.build_heldout_rows(train, RETAIN, pr.GENERIC_PROMPTS, ["same q  "], ABSTAIN)
        # 10 train + 26 retention + 6 generic; the eval-matching train prompt is dropped (case/space normalised)
        self.assertEqual(len(rows), 10 + 26 + 6)
        self.assertEqual(info["dropped_eval_overlap"], ["SAME Q"])
        self.assertEqual(info["abstain_rows"], 6)
        self.assertEqual(sum(r["completion"] == ABSTAIN for r in rows), 6)
        self.assertEqual(info["eval_prompt_overlap_after_build"], [])

    def test_retention_and_generic_are_not_dropped_by_guard(self):
        rows, info = ho.build_heldout_rows([], RETAIN, pr.GENERIC_PROMPTS, [RETAIN[0]["prompt"]], ABSTAIN)
        # the retention row that equals an eval prompt is still a retention row, so the leak check reports it
        self.assertEqual(info["eval_prompt_overlap_after_build"], [RETAIN[0]["prompt"].lower()])


@unittest.skipUnless(os.path.exists(LABELS), "real labels.json not present")
class RealLabels(unittest.TestCase):
    def setUp(self):
        with open(LABELS, encoding="utf-8") as f:
            self.L = json.load(f)

    def test_real_split_counts_and_zero_overlap(self):
        pids = ho.all_passage_ids(self.L)
        self.assertEqual(len(pids), 87)
        h, counts = ho.split_passages(pids, frac=1 / 3, seed=0)
        self.assertEqual({s: counts[s]["passages"] for s in counts},
                         {"debug": 13, "check": 15, "document": 29, "scope": 30})
        H = ho.heldout_set(h)
        parts = ho.split_labels(self.L, H)
        self.assertEqual(ho.check_split(h, parts["train_rows"], parts["train_heldout_rows"], [],
                                        parts["eval_held"], parts["eval_probes"]), [])
        # every held-out passage carries at least one held or probe question
        q_pass = {r["passage"] for r in parts["eval_held"] + parts["eval_probes"]}
        self.assertEqual(sorted(H - q_pass), [])
        # labelled training rows: 337 total, the rest of the split is exact
        self.assertEqual(len(parts["train_rows"]) + len(parts["train_heldout_rows"]), len(self.L["train"]))
        for s in h:
            tr = {r["passage"] for r in parts["train_rows"] if r["skill"] == s}
            self.assertFalse(tr & set(h[s]), s)


class HeldoutGenerationsFile(unittest.TestCase):
    """stage_eval_heldout must write generations_heldout.json (CPU only: model, tokenizer and generation are mocked)."""

    def test_stage_writes_generations_heldout_json(self):
        import json as _json
        import sys
        import tempfile
        import types
        from unittest import mock

        import phase2_multilora as pml

        held = [{"prompt": "h0", "skill": "debug"}, {"prompt": "h1", "skill": "check"}]
        probe = [{"prompt": "p0", "skill": "document"}]
        parts = {"eval_held": held, "eval_probes": probe}
        passages = [{"text": "passage A", "skill": "debug"}, {"text": "passage B", "skill": "check"},
                    {"text": "passage C", "skill": "document"}]
        args = types.SimpleNamespace(student="fake-student")
        gguf = {"heldout": "/nonexistent/h.gguf", "old_single": "/nonexistent/o.gguf"}

        calls = []

        def fake_generate(tok, prompts, scales, n_ids=None):
            calls.append((list(prompts), n_ids))
            return [f"gen:{p}:{len(scales[0])}" for p in prompts]

        fake_tf = types.SimpleNamespace(AutoTokenizer=types.SimpleNamespace(
            from_pretrained=lambda path: object()))
        fake_proc = types.SimpleNamespace(terminate=lambda: None, wait=lambda timeout=None: 0)

        with tempfile.TemporaryDirectory() as out_dir:
            args.out = out_dir
            with mock.patch.dict(sys.modules, {"transformers": fake_tf}), \
                 mock.patch.object(pml, "serve", return_value=fake_proc) as serve_m, \
                 mock.patch.object(pml, "generate_grouped", side_effect=fake_generate), \
                 mock.patch.object(pml.r2, "local_dir", return_value="/nonexistent/student"), \
                 mock.patch.object(pml.sv, "best_passage_index", side_effect=lambda tfidf, q: 0 if q != "p0" else 2), \
                 mock.patch.object(pml, "arm_metrics", return_value={"gated": {"good": 0.0}}):
                results = pml.stage_eval_heldout(args, None, parts, passages, {}, "", None, gguf, "/nonexistent/base.gguf")

            path = os.path.join(out_dir, "generations_heldout.json")
            self.assertTrue(os.path.exists(path))
            with open(path) as f:
                data = _json.load(f)

        self.assertEqual(sorted(data), sorted(f"{n}/{arm}" for n in ["held", "probe"]
                                              for arm in ["base", "heldout_single", "old_single_control"]))
        for key, val in data.items():
            self.assertEqual(set(val), {"outs"}, key)
            n = len(held) if key.startswith("held/") else len(probe)
            self.assertEqual(len(val["outs"]), n, key)
        self.assertEqual(data["held/base"]["outs"], ["gen:h0:0", "gen:h1:0"])
        self.assertEqual(data["probe/heldout_single"]["outs"], ["gen:p0:1"])
        self.assertEqual(data["probe/old_single_control"]["outs"], ["gen:p0:1"])
        self.assertEqual(serve_m.call_count, 1)
        self.assertEqual({n_ids for _, n_ids in calls}, {pml.HELDOUT_N_IDS})
        self.assertEqual(sorted(results["arms"]), ["base", "heldout_single", "old_single_control"])


if __name__ == "__main__":
    unittest.main()
