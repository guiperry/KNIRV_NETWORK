#!/usr/bin/env python3
"""
Hybrid teacher corpus builder for the Task 5.2 spike (context distillation).

Policy (matches the inference plan: one teacher backend per run, never two at once):

  pass 1  LOCAL   llama-server (Qwen2.5-7B Q3_K_M, GPU). The cluster's skills are in
                  the system prompt; the teacher answers each task as an agent holding
                  all of them. This server is shut down before pass 2 starts.
  gate    CHECK   deterministic acceptance, no model involved: the answer must contain
                  the target skill's exact command, value and status code, and none of
                  another member's command/code (cross-contamination).
  pass 2  REMOTE  (optional, --escalate) only the records that failed the gate are
                  re-labelled by Workers AI llama-3.3-70b, then gated the same way.
                  Skills sent are the cluster's fixture skills only.

Every record carries teacher_id. Records that fail the gate after all passes are
dropped and counted; they are never patched up. The Cloudflare token and account id
are read from the environment and are never written to disk or logs.

Output: <out>/corpus_<tag>_c<ci>.json
"""
import argparse
import hashlib
import json
import os
import random
import subprocess
import sys
import time
import urllib.error
import urllib.request

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
import spike_path_c as sp  # noqa: E402

LLAMA_BIN = os.environ.get("LLAMA_SERVER", "/home/gperry/llama-cuda-build/llama.cpp/build/bin/llama-server")
PORT = 18096
SYSTEM = ("You are a support engineer holding these skills. Answer the user's question using the matching "
          "skill, in one or two sentences, stating the exact command, flag value and status code.\n")


def gate(target, members, text):
    if not text or not all(k in text for k in sp.key_tokens(target)):
        return "missing-target-tokens"
    for m in members:
        if m["id"] != target["id"] and (m["cmd"] in text or m["code"] in text):
            return "cross-contamination"
    return None


class QuotaExceeded(Exception):
    """The remote provider refused for quota (HTTP 429). Not a verdict on the record."""


def local_call(system, q, temp, max_tokens=120):
    body = json.dumps({"messages": [{"role": "system", "content": system}, {"role": "user", "content": q}],
                       "max_tokens": max_tokens, "temperature": temp}).encode()
    r = urllib.request.Request(f"http://127.0.0.1:{PORT}/v1/chat/completions", body, {"Content-Type": "application/json"})
    return json.load(urllib.request.urlopen(r, timeout=120))["choices"][0]["message"]["content"]


def remote_call(system, q, model):
    a, t = os.environ["CLOUDFLARE_ACCOUNT_ID"], os.environ["CLOUDFLARE_API_TOKEN"]
    body = json.dumps({"messages": [{"role": "system", "content": system}, {"role": "user", "content": q}],
                       "max_tokens": 200, "temperature": 0.3}).encode()
    r = urllib.request.Request(f"https://api.cloudflare.com/client/v4/accounts/{a}/ai/run/{model}", body,
                               {"Authorization": "Bearer " + t, "Content-Type": "application/json"})
    try:
        res = json.load(urllib.request.urlopen(r, timeout=90)).get("result", {})
    except urllib.error.HTTPError as e:
        if e.code == 429:
            raise QuotaExceeded(str(e.code)) from None
        raise
    return res.get("response") or (res.get("choices", [{}])[0].get("message", {}).get("content")) or ""


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--model", required=True, help="local GGUF teacher")
    ap.add_argument("--out", required=True)
    ap.add_argument("--seeds", default="0,1,2")
    ap.add_argument("--sizes", default="3,4,5")
    ap.add_argument("--per-skill", type=int, default=32)
    ap.add_argument("--temp", type=float, default=0.7)
    ap.add_argument("--escalate", action="store_true")
    ap.add_argument("--remote-model", default="@cf/meta/llama-3.3-70b-instruct-fp8-fast")
    a = ap.parse_args()
    os.makedirs(a.out, exist_ok=True)

    plan = []  # (tag, ci, skills, system, [(skill, question)])
    for seed in (int(x) for x in a.seeds.split(",")):
        for ci, size in enumerate(int(x) for x in a.sizes.split(",")):
            tag = f"s{seed}"
            skills = [sp.make_skill(f"{tag}c{ci}", i) for i in range(size)]
            system = SYSTEM + "\n".join(sp.skill_text(s) for s in skills)
            tasks = []
            for s in skills:
                for q in sp.tasks(s, sp.SYMPTOMS_TRAIN, a.per_skill, None):
                    tasks.append((s, q))
            plan.append((tag, ci, skills, system, tasks))

    env = dict(os.environ, LD_LIBRARY_PATH=os.path.dirname(LLAMA_BIN))
    srv = subprocess.Popen([LLAMA_BIN, "-m", a.model, "-ngl", "99", "-c", "4096", "-fa", "on", "--port", str(PORT)],
                           env=env, stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
    results = {}
    try:
        for _ in range(120):
            try:
                urllib.request.urlopen(f"http://127.0.0.1:{PORT}/health", timeout=2)
                break
            except Exception:
                time.sleep(2)
        for tag, ci, skills, system, tasks in plan:
            recs, rejected, reasons = [], [], {}
            for s, q in tasks:
                txt = local_call(system, q, a.temp)
                why = gate(s, skills, txt)
                if why:
                    rejected.append((s, q))
                    reasons[why] = reasons.get(why, 0) + 1
                else:
                    recs.append({"skill": s["id"], "prompt": q, "completion": txt.strip(), "teacher_id": "local:qwen2.5-7b-q3_k_m"})
            results[(tag, ci)] = (recs, rejected, reasons)
            print(f"[local] {tag} c{ci}: accepted {len(recs)}/{len(tasks)} rejected {reasons}", flush=True)
    finally:
        srv.terminate()
        srv.wait(timeout=30)  # local teacher is gone before any remote call

    for tag, ci, skills, system, tasks in plan:
        recs, rejected, reasons = results[(tag, ci)]
        remote_fixed = 0
        if a.escalate and rejected:
            still = []
            for s, q in rejected:
                try:
                    txt = remote_call(system, q, a.remote_model)
                except Exception as e:  # remote failure leaves the record dropped
                    still.append((s, q)); print("remote error:", type(e).__name__, flush=True); continue
                if gate(s, skills, txt) is None:
                    recs.append({"skill": s["id"], "prompt": q, "completion": txt.strip(), "teacher_id": "cf:" + a.remote_model})
                    remote_fixed += 1
                else:
                    still.append((s, q))
            rejected = still
            print(f"[remote] {tag} c{ci}: recovered {remote_fixed}, dropped {len(rejected)}", flush=True)
        recs.sort(key=lambda r: (r["skill"], r["prompt"], r["completion"]))
        body = {"tag": tag, "cluster": ci, "size": len(skills), "records": recs,
                "accepted": len(recs), "dropped": len(rejected), "local_reject_reasons": reasons,
                "remote_recovered": remote_fixed,
                "skill_set_root": hashlib.sha256("\n".join(sorted(sp.skill_text(s) for s in skills)).encode()).hexdigest()}
        body["corpus_hash"] = hashlib.sha256(json.dumps(recs, sort_keys=True).encode()).hexdigest()
        with open(os.path.join(a.out, f"corpus_{tag}_c{ci}.json"), "w") as f:
            json.dump(body, f, indent=1)


if __name__ == "__main__":
    main()
