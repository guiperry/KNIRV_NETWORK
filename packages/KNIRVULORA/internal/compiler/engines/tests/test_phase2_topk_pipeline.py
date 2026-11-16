"""Pipeline-layer tests for the top-k gate (I_52 19a, validator mutants C, D, E; I_52 seeds 1-2 stronger path).

CPU only, no torch, no llama-server. The generation cache is pre-filled, so stage_eval_topk and
stage_eval_stronger score cached answers and never call serve() or generate_grouped().
Run from this directory:  PYTHONDONTWRITEBYTECODE=1 python3 -m unittest test_phase2_topk_pipeline
"""
import json
import os
import sys
import tempfile
import types
import unittest
from argparse import Namespace
from unittest import mock

import derive_facts as df
import phase2_multilora as pm
import span_verifier as sv

# Passage 1 has no span the answers use. Passage 2 holds the NRV_LOG_LEVEL span, passage 3 the cache path.
P1 = "docker ps containers list running containers"
P2 = "Set NRV_LOG_LEVEL=debug in .env before restarting the service."
P3 = "The cache directory is ~/.cache/knirv and is cleared weekly."
TEXTS = [P1, P2, P3]
ANS_P2 = "Set NRV_LOG_LEVEL=debug in .env."   # every span is in passage 2 only
ANS_P3 = "The cache is at ~/.cache/knirv."     # every span is in passage 3 only


def make_passages():
    out = []
    for i, text in enumerate(TEXTS):
        out.append({"id": f"p{i + 1}", "skill": "debug", "text": text,
                    "spans": sv_spans(text)})
    return out


def sv_spans(text):
    return df.spans(text)


class Fixture(unittest.TestCase):
    def setUp(self):
        self.passages = make_passages()
        self.by_id = {p["id"]: p for p in self.passages}
        self.corpus_lc = "\n".join(TEXTS).lower()
        self.tfidf = df.TfIdf(TEXTS)
        # the held question's gold passage is p2; its prompt must rank p1 top and p2 second by TF-IDF,
        # so top-1 refuses and top-3 is needed to reach p2. Checked here, not assumed.
        order = sv.best_passage_indices(self.tfidf, "docker containers", 3)
        self.assertEqual(order, [0, 1, 2])


class ArmScoresTopK(Fixture):
    def tops(self, texts_order):
        return [self.passages[i] for i in texts_order]

    def test_k_selects_first_k_passages(self):
        # held q1: gold p2 at rank 2; held q2: gold p3 at rank 3. Both answers are in the gold passage only.
        held = [{"prompt": "q1", "skill": "debug", "passage": "p2"},
                {"prompt": "q2", "skill": "debug", "passage": "p3"}]
        probe = [{"prompt": "q3", "skill": "debug", "passage": "p2"}]
        tops = {
            "held": [self.tops([0, 1, 2]), self.tops([0, 1, 2])],
            "probe": [self.tops([0, 1, 2])],
        }
        outs = {"held": [ANS_P2, ANS_P3], "probe": [ANS_P2]}
        r1 = pm.arm_scores_topk(outs, held, probe, tops, [0, 1], [0], 1, self.by_id, self.corpus_lc)
        r3 = pm.arm_scores_topk(outs, held, probe, tops, [0, 1], [0], 3, self.by_id, self.corpus_lc)
        # k=1: top-1 is p1 only, so both held answers are refused: gated good 0.
        self.assertEqual(r1["gated"]["good"], 0.0)
        self.assertEqual(r1["gated"]["gate_false_refusal"], 1.0)
        # k=3 admits both (passages 2 and 3 are inside the top-3): gated good 1.0.
        self.assertEqual(r3["gated"]["good"], 1.0)
        self.assertEqual(r3["gated"]["gate_false_refusal"], 0.0)
        # probe: k=1 refuses the probe answer, k=3 keeps it (it is corpus-grounded)
        self.assertEqual(r1["probe"]["refused_gated"], 1.0)
        self.assertEqual(r3["probe"]["refused_gated"], 0.0)

    def test_k2_is_not_k3(self):
        # gold passage p3 is third: k=2 must still refuse, k=3 must admit (catches [:k-1] and [:2] slices)
        held = [{"prompt": "q2", "skill": "debug", "passage": "p3"}]
        probe = [{"prompt": "q3", "skill": "debug", "passage": "p2"}]
        tops = {"held": [self.tops([0, 1, 2])], "probe": [self.tops([0, 1, 2])]}
        outs = {"held": [ANS_P3], "probe": [ANS_P2]}
        r2 = pm.arm_scores_topk(outs, held, probe, tops, [0], [0], 2, self.by_id, self.corpus_lc)
        r3 = pm.arm_scores_topk(outs, held, probe, tops, [0], [0], 3, self.by_id, self.corpus_lc)
        self.assertEqual(r2["gated"]["good"], 0.0)
        self.assertEqual(r3["gated"]["good"], 1.0)

    def test_idx_subset_selects_rows(self):
        held = [{"prompt": "q1", "skill": "debug", "passage": "p2"},
                {"prompt": "q2", "skill": "debug", "passage": "p3"}]
        probe = [{"prompt": "q3", "skill": "debug", "passage": "p2"}]
        tops = {"held": [self.tops([0, 1, 2]), self.tops([0, 1, 2])], "probe": [self.tops([0, 1, 2])]}
        outs = {"held": [ANS_P2, ANS_P3], "probe": [ANS_P2]}
        r = pm.arm_scores_topk(outs, held, probe, tops, [1], [0], 3, self.by_id, self.corpus_lc)
        self.assertEqual(r["n_held"], 1)
        self.assertEqual(r["gated"]["good"], 1.0)


# ---- stage_eval_topk and stage_eval_stronger: the k flow through the stage, not only the scorer ----

HELD = [{"prompt": "docker containers", "skill": "debug", "passage": "p2"}]
PROBE = [{"prompt": "docker containers", "skill": "debug", "passage": "p2"}]


def fake_transformers():
    mod = types.ModuleType("transformers")
    mod.AutoTokenizer = types.SimpleNamespace(from_pretrained=lambda *a, **k: None)
    return mod


def arm_cache(answer_for_discriminating_arm):
    """All arms cached. Only the 'single' arm carries the k-discriminating answer; the rest are plain prose."""
    plain = "Plain prose with no spans."
    out = {}
    for name, n in [("held", len(HELD)), ("probe", len(PROBE))]:
        for a in ["base", "matched_routed", "matched_oracle"]:
            out[f"{name}/{a}"] = {"outs": [plain] * n, "wall_s": 1.0}
        out[f"{name}/single"] = {"outs": [answer_for_discriminating_arm] * n, "wall_s": 1.0}
    return out


class StageEvalTopK(Fixture):
    """stage_eval_topk: retrieval inside the stage must take max(TOPK_KS) passages (mutant E)."""

    def run_stage(self, cache):
        tmp = tempfile.mkdtemp()
        with open(os.path.join(tmp, "generations_topk.json"), "w") as f:
            json.dump(cache, f)
        a = Namespace(student="x", out=tmp)
        L = {"held": HELD, "probes": PROBE}
        gguf = {f"matched_{s}": "g" for s in pm.pr.MATCHED_SKILLS}
        gguf["single_plain_all"] = "g"
        with mock.patch.dict(sys.modules, {"transformers": fake_transformers()}), \
                mock.patch.object(pm.r2, "local_dir", return_value="x"):
            res = pm.stage_eval_topk(a, L, self.passages, self.by_id, self.corpus_lc, self.tfidf, gguf, "base")
        return res

    def test_single_arm_k1_refuses_k3_admits(self):
        res = self.run_stage(arm_cache(ANS_P2))
        self.assertEqual(res["arms"]["single"]["full"]["k1"]["gated"]["good"], 0.0)
        self.assertEqual(res["arms"]["single"]["full"]["k3"]["gated"]["good"], 1.0)
        self.assertEqual(res["arms"]["single"]["full"]["k1"]["probe"]["refused_gated"], 1.0)
        self.assertEqual(res["arms"]["single"]["full"]["k3"]["probe"]["refused_gated"], 0.0)

    def test_retrieval_inside_stage_returns_three_passages(self):
        # mutant E (top-1 retrieval only) would make k=3 identical to k=1 here
        res = self.run_stage(arm_cache(ANS_P2))
        self.assertNotEqual(res["arms"]["single"]["full"]["k3"]["gated"]["good"],
                            res["arms"]["single"]["full"]["k1"]["gated"]["good"])


class StageEvalStronger(Fixture):
    """stage_eval_stronger: same k flow for the stronger per-skill arms and the seed-s single arm."""

    def run_stage(self, reuse_cache, own_cache, reuse_arms="base,single,matched_routed,matched_oracle"):
        tmp = tempfile.mkdtemp()
        own_path = os.path.join(tmp, "generations_stronger.json")
        with open(own_path, "w") as f:
            json.dump(own_cache, f)
        reuse_path = ""
        if reuse_cache is not None:
            reuse_path = os.path.join(tmp, "generations_topk.json")
            with open(reuse_path, "w") as f:
                json.dump(reuse_cache, f)
        a = Namespace(student="x", out=tmp, reuse_generations=reuse_path, reuse_arms=reuse_arms)
        L = {"held": HELD, "probes": PROBE}
        gguf = {f"stronger_{s}": "g" for s in pm.pr.MATCHED_SKILLS}
        gguf["single_plain_all"] = "g"
        with mock.patch.dict(sys.modules, {"transformers": fake_transformers()}), \
                mock.patch.object(pm.r2, "local_dir", return_value="x"):
            return pm.stage_eval_stronger(a, L, self.passages, self.by_id, self.corpus_lc, self.tfidf, gguf, "base")

    def test_stronger_arms_k_flow(self):
        full = arm_cache(ANS_P2)
        own = {k: v for k, v in full.items() if k.split("/")[1] == "single"}
        for name in ["held", "probe"]:
            own[f"{name}/stronger_routed"] = {"outs": [ANS_P2], "wall_s": 1.0}
            own[f"{name}/stronger_oracle"] = {"outs": [ANS_P2], "wall_s": 1.0}
        # every key is in own cache, so nothing is generated and the scorer sees the cached answers
        res = self.run_stage(None, own)
        self.assertEqual(res["arms"]["stronger_routed"]["full"]["k1"]["gated"]["good"], 0.0)
        self.assertEqual(res["arms"]["stronger_routed"]["full"]["k3"]["gated"]["good"], 1.0)
        self.assertEqual(res["arms"]["stronger_oracle"]["full"]["k3"]["gated"]["good"], 1.0)
        self.assertEqual(res["arms"]["single"]["full"]["k3"]["gated"]["good"], 1.0)
        # seed-1/2 style run: base reused, no matched arms present in the scored arms
        self.assertNotIn("matched_routed", res["arms"])

    def test_seed_run_reuses_base_only(self):
        reuse = {}
        for name in ["held", "probe"]:
            reuse[f"{name}/base"] = {"outs": ["Plain prose."], "wall_s": 1.0}
            reuse[f"{name}/single"] = {"outs": ["SHOULD NOT BE USED"], "wall_s": 1.0}
        own = {}
        for name in ["held", "probe"]:
            own[f"{name}/single"] = {"outs": [ANS_P2], "wall_s": 1.0}
            own[f"{name}/stronger_routed"] = {"outs": [ANS_P2], "wall_s": 1.0}
            own[f"{name}/stronger_oracle"] = {"outs": [ANS_P2], "wall_s": 1.0}
        res = self.run_stage(reuse, own, reuse_arms="base")
        # the single arm must be the seed's own generation, not the reused one
        self.assertEqual(res["generation_source"]["held/single"], "generated")
        self.assertEqual(res["arms"]["single"]["full"]["k3"]["gated"]["good"], 1.0)


class ReuseArmsAndGeneration(Fixture):
    """I_52 fixes issue 4 (validator M3): with --reuse-arms base and no own-cache file, the seed's own
    single arm must be generated, never taken from the reused (seed-0, leaky) cache."""

    def run_gen_stage(self, reuse_cache, reuse_arms):
        tmp = tempfile.mkdtemp()
        reuse_path = os.path.join(tmp, "generations_topk.json")
        with open(reuse_path, "w") as f:
            json.dump(reuse_cache, f)
        a = Namespace(student="x", out=tmp, reuse_generations=reuse_path, reuse_arms=reuse_arms)
        L = {"held": HELD, "probes": PROBE}
        gguf = {f"stronger_{s}": "g" for s in pm.pr.MATCHED_SKILLS}
        gguf["single_plain_all"] = "g"
        fake_proc = mock.Mock()
        gen_calls = []

        def fake_generate(tok, prompts, scales, n_ids):
            gen_calls.append(scales)
            return [ANS_P2] * len(prompts)

        with mock.patch.dict(sys.modules, {"transformers": fake_transformers()}), \
                mock.patch.object(pm.r2, "local_dir", return_value="x"), \
                mock.patch.object(pm, "serve", return_value=fake_proc), \
                mock.patch.object(pm, "generate_grouped", side_effect=fake_generate):
            res = pm.stage_eval_stronger(a, L, self.passages, self.by_id, self.corpus_lc, self.tfidf, gguf, "base")
        return res, gen_calls

    def test_reuse_base_only_without_own_cache_generates_single(self):
        reuse = {}
        for name in ["held", "probe"]:
            reuse[f"{name}/base"] = {"outs": ["Plain prose."], "wall_s": 1.0}
            reuse[f"{name}/single"] = {"outs": ["SHOULD NOT BE USED"], "wall_s": 1.0}
            # complete reuse file (all arms present) so a mutant cannot fail on a missing key first
            for arm in ["matched_routed", "matched_oracle"]:
                reuse[f"{name}/{arm}"] = {"outs": ["Plain prose."], "wall_s": 1.0}
        res, gen_calls = self.run_gen_stage(reuse, reuse_arms="base")
        self.assertEqual(res["generation_source"]["held/single"], "generated")
        self.assertEqual(res["generation_source"]["probe/single"], "generated")
        self.assertEqual(res["generation_source"]["held/base"], "reused")
        # the seed's own single adapter (server id STRONG_SINGLE_ID) was actually generated
        self.assertTrue(any(sc and sc[0] == {pm.STRONG_SINGLE_ID: 1.0} for sc in gen_calls), gen_calls)
        # and the scored single arm is the generated answer, not the reused placeholder
        self.assertEqual(res["arms"]["single"]["full"]["k3"]["gated"]["good"], 1.0)
        self.assertNotIn("matched_routed", res["arms"])


class SeedPlumbing(unittest.TestCase):
    """I_52 fixes issue 5: the seed reaches LoRA training config and the stronger row sampling
    (validator mutants M4a single_lf training seed, M4b stronger training + rows, M4c row sampling)."""

    def setUp(self):
        import grounded_spike as gs
        self.abstain = gs.ABSTAIN
        self.tmp = tempfile.TemporaryDirectory()
        self.addCleanup(self.tmp.cleanup)
        self.ret = os.path.join(self.tmp.name, "retention_in.json")
        with open(self.ret, "w") as f:
            json.dump([{"prompt": f"retain {i}", "completion": f"answer {i}"} for i in range(5)], f)
        self.L = make_seed_L(self.abstain)

    def engine_recorder(self):
        calls = []

        def fake_engine(base, dataset, cfg, out_path):
            calls.append({"cfg": dict(cfg), "out": out_path, "rows": len(dataset)})
            return {"final_loss": 0.5, "steps": 1}
        return calls, fake_engine

    def args(self, out, seed):
        return Namespace(student="x", out=out, retention_rows=self.ret, seed=seed)

    def test_single_lf_training_uses_seed(self):
        out = os.path.join(self.tmp.name, "s2")
        os.makedirs(out)
        calls, fake = self.engine_recorder()
        a = self.args(out, 2)
        with mock.patch.object(pm.r2, "local_dir", return_value="x"), \
                mock.patch.object(pm.sp, "run_engine", side_effect=fake):
            info = pm.stage_train_single_lf(a, self.L)
        self.assertEqual(len(calls), 1)
        self.assertEqual(calls[0]["cfg"]["seed"], 2)
        self.assertEqual(info["seed"], 2)
        self.assertTrue(a.single_adapter.endswith("single_lf_seed2.safetensors"))

    def test_stronger_training_and_row_sampling_use_seed(self):
        import phase2_rows as pr
        out = os.path.join(self.tmp.name, "st2")
        os.makedirs(out)
        calls, fake = self.engine_recorder()
        a = self.args(out, 2)
        with mock.patch.object(pm.r2, "local_dir", return_value="x"), \
                mock.patch.object(pm.sp, "run_engine", side_effect=fake), \
                mock.patch.object(pm.pr, "build_stronger_rows", wraps=pr.build_stronger_rows) as br:
            pm.stage_train_stronger(a, self.L)
        self.assertEqual(len(calls), len(pm.pr.MATCHED_SKILLS))
        for c in calls:
            self.assertEqual(c["cfg"]["seed"], 2)
            self.assertEqual(c["cfg"]["epochs"], pm.pr.STRONGER_EPOCHS)
        for c in br.call_args_list:
            self.assertEqual(c.args[6], 2)   # build_stronger_rows(skill, train, retain, negs, generic, share, seed, ...)

    def test_seed_changes_stronger_rows_through_stage(self):
        rows = {}
        for seed in (1, 2):
            out = os.path.join(self.tmp.name, f"r{seed}")
            os.makedirs(out)
            _, fake = self.engine_recorder()
            with mock.patch.object(pm.r2, "local_dir", return_value="x"), \
                    mock.patch.object(pm.sp, "run_engine", side_effect=fake):
                pm.stage_train_stronger(self.args(out, seed), self.L)
            with open(os.path.join(out, "stronger_rows_debug.json")) as f:
                rows[seed] = json.load(f)
        self.assertNotEqual(rows[1], rows[2])


def make_seed_L(abstain):
    """Small labelled set shaped like labels.json (train, negs, held, probes)."""
    skills = ["debug", "check", "document", "scope"]
    train = [{"skill": s, "prompt": f"{s} q{i}", "completion": f"{s} a{i}", "passage": f"{s}#{i % 3}"}
             for s in skills for i in range(20)]
    negs = [{"skill": s, "passage": f"{s}#0", "prompt": f"{s} neg {i}", "completion": abstain}
            for s in skills for i in range(30)]
    held = [{"prompt": "held one", "skill": "debug", "passage": "debug#0"}]
    probes = [{"prompt": "probe one", "skill": "debug", "passage": "debug#0"}]
    return {"train": train, "negs": negs, "held": held, "probes": probes}


class LeakFilterStage(unittest.TestCase):
    """I_52 fixes issue 5 (validator mutant M2b): stage_train_single_lf must drop eval-overlap rows
    itself; a bypass would train on them."""

    def test_single_lf_stage_drops_held_probe_rows(self):
        import grounded_spike as gs
        L = make_seed_L(gs.ABSTAIN)
        L["train"].append({"skill": "debug", "prompt": "HELD ONE  ", "completion": "leaked", "passage": "debug#0"})
        with tempfile.TemporaryDirectory() as tmp:
            ret = os.path.join(tmp, "ret.json")
            with open(ret, "w") as f:
                json.dump([{"prompt": "retain 0", "completion": "answer 0"}], f)
            out = os.path.join(tmp, "o")
            os.makedirs(out)
            a = Namespace(student="x", out=out, retention_rows=ret, seed=1)
            calls = []

            def fake(base, dataset, cfg, out_path):
                calls.append(dataset)
                return {"final_loss": 0.5, "steps": 1}
            with mock.patch.object(pm.r2, "local_dir", return_value="x"), \
                    mock.patch.object(pm.sp, "run_engine", side_effect=fake):
                info = pm.stage_train_single_lf(a, L)   # must not raise the overlap RuntimeError
            self.assertEqual(info["dropped_eval_overlap"], ["HELD ONE  "])
            trained_prompts = [r["prompt"] for r in calls[0]]
            self.assertNotIn("HELD ONE  ", trained_prompts)
            self.assertNotIn("held one", [p.lower().strip() for p in trained_prompts])


class LeakFilter(unittest.TestCase):
    def test_drop_eval_overlap_removes_only_matching_prompts(self):
        import phase2_rows as pr
        rows = [{"prompt": "How many modes are there for `/check`?", "completion": "x"},
                {"prompt": "something else", "completion": "y"}]
        kept, dropped = pr.drop_eval_overlap(rows, ["how many modes are there for `/check`?  "])
        self.assertEqual([r["prompt"] for r in kept], ["something else"])
        self.assertEqual(dropped, ["How many modes are there for `/check`?"])


if __name__ == "__main__":
    unittest.main()
