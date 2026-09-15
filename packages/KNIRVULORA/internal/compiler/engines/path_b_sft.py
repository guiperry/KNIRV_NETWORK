#!/usr/bin/env python3
"""
Path B: direct fit of the canonical core from a corpus.

WHAT THIS IS, AND WHAT IT IS NOT
-------------------------------
This engine produces a uLoRA CANONICAL CORE: per semantic module, a low-rank
pair in the shared canonical space of dimension K.

    <module>/lora_A   (r, K)
    <module>/lora_B   (K, r)

so that a downstream binder can project it onto a concrete model with
A_target = A_core @ P_in, B_target = P_out @ B_core.

It is NOT transformer LoRA training. No base model is loaded, there is no
forward pass, and the loss is not cross-entropy over a vocabulary. What it does
is fit a low-rank linear map in canonical space from paired corpus embeddings:

    x      = embed(context)             in R^K
    y_true = embed(corrected_completion) - embed(context)   in R^K
    y_pred = (alpha/r) * B @ A @ x
    loss   = mean squared error over the corpus

The output is therefore an honest linear fit over a deterministic embedding of
the corpus text. It is usable as the canonical core that uLoRAd consumes for real
per-layer training against an actual base model; it is not a substitute for that
training, and the metadata it writes says so explicitly so nothing downstream can
mistake it for one.

Two things this replaces, both of which made the artifact unusable:

  * It emitted a single flat "lora_A"/"lora_B" pair fitted to one target's
    (d_in, d_out) rather than a per-module core in K-space, so the result could
    not be bound by anything.
  * It encoded text with Python's builtin hash(), which CPython salts per
    process. The same corpus produced different weights on every run, and
    therefore a different bundle content hash — which breaks the content
    addressing the whole protocol rests on.

Input JSON (stdin):
{
  "dataset": [{"context": "...", "corrected_completion": "...", "target_model": "..."}],
  "target_model": {"hidden_size": 4096, ...},
  "config": {"rank": 16, "alpha": 32.0, "learning_rate": 0.01, "epochs": 50,
             "canonical_dim": 1024, "target_modules": ["self_attn.q_proj", ...],
             "encode_dim": 512, "seed": 42},
  "output_path": "<safetensors>"
}

Output tensors:
  <module>/lora_A   (r, K)
  <module>/lora_B   (K, r)
"""

import hashlib
import json
import sys
import traceback

import numpy as np
from safetensors.numpy import save_file


class PathBError(Exception):
    """A refusal: the request cannot be honoured as stated."""


def embed_text(text, dim, salt="ulora-v1"):
    """
    Deterministic feature-hash embedding of text into R^dim.

    Deliberately NOT Python's builtin hash(): that is salted per interpreter
    process, so the same corpus encodes differently on each run. Here the token
    digest comes from blake2b, so the vector is stable across processes, machines
    and interpreter versions regardless of PYTHONHASHSEED. Determinism has to be
    a property of the code, not of the environment it happens to run in.

    A fixed digest also means the signature can be extended (change `salt`)
    without silently reinterpreting previously trained weights.
    """
    vec = np.zeros(dim, dtype=np.float32)
    tokens = text.lower().split()
    if not tokens:
        tokens = [text[:1] if text else " "]

    for token in tokens:
        digest = hashlib.blake2b(f"{salt}:{token}".encode("utf-8"), digest_size=16).digest()
        h = int.from_bytes(digest, "little")
        for i in range(dim):
            # Signed hashing: without the sign term every token contributes a
            # positive offset, so unrelated texts drift toward a similar
            # direction and the embedding loses discriminative power.
            sign = 1.0 if (h >> (i % 64)) & 1 else -1.0
            vec[i] += sign * (0.5 + 0.5 * np.sin((h % 100003) * (i + 1) * 0.01))

    norm = np.linalg.norm(vec)
    if norm > 0:
        vec = vec / norm
    return vec


def fit_core(dataset, config, canonical_dim, seed):
    """
    Fit one canonical core (A_core, B_core) in R^K over an explicit record set.

    Takes the records rather than the whole corpus: a core is per semantic module,
    so the caller decides which records belong to it.
    """
    r = int(config.get("rank", 16))
    alpha = float(config.get("alpha", 32.0))
    lr = float(config.get("learning_rate", 0.01))
    epochs = int(config.get("epochs", 50))

    if not dataset:
        raise PathBError("no records were attributed to this core: there is nothing to fit")

    # encode_dim is fixed to K: the fit happens in canonical space, so the
    # embedding must live there too.
    encode_dim = int(config.get("encode_dim") or canonical_dim)
    if encode_dim != canonical_dim:
        raise PathBError(
            f"config.encode_dim {encode_dim} must equal canonical_dim {canonical_dim}: "
            "the core acts in the canonical space, so fitting it against an embedding "
            "of a different width would mix two spaces"
        )

    np.random.seed(seed)

    inputs, targets = [], []
    for i, rec in enumerate(dataset):
        if "context" not in rec or "corrected_completion" not in rec:
            raise PathBError(f"dataset record {i} is missing context or corrected_completion")
        ctx = embed_text(rec["context"], encode_dim)
        comp = embed_text(rec["corrected_completion"], encode_dim)
        inputs.append(ctx)
        # The residual the adapter must supply: how the desired completion
        # differs from what the context alone implies.
        targets.append(comp - ctx)

    X = np.stack(inputs).astype(np.float64)
    Y = np.stack(targets).astype(np.float64)
    n = X.shape[0]

    scale = 0.01
    a_core = np.random.randn(r, canonical_dim).astype(np.float64) * scale
    b_core = np.random.randn(canonical_dim, r).astype(np.float64) * scale
    scaling = alpha / r

    losses = []
    for _ in range(epochs):
        epoch_loss = 0.0
        for idx in np.random.permutation(n):
            x = X[idx]
            y_true = Y[idx]

            ax = a_core @ x
            delta = scaling * (b_core @ ax)
            error = delta - y_true
            epoch_loss += float(np.mean(error ** 2))

            d_delta = (2.0 / canonical_dim) * error
            grad_b = np.outer(d_delta, ax) * scaling
            grad_a = scaling * (b_core.T @ d_delta).reshape(r, 1) * x.reshape(1, -1)

            a_core -= lr * grad_a
            b_core -= lr * grad_b
        losses.append(epoch_loss / n)

    return a_core.astype(np.float32), b_core.astype(np.float32), losses


def main():
    payload = json.loads(sys.stdin.read())
    config = payload.get("config", {})
    dataset = payload.get("dataset", [])
    output_path = payload["output_path"]

    r = int(config.get("rank", 16))
    canonical_dim = int(config.get("canonical_dim", 0))
    modules = config.get("target_modules") or []
    allow_shared = bool(config.get("allow_shared_core", False))
    seed = int(config.get("seed", 42))

    if canonical_dim <= 0:
        raise PathBError(
            "config.canonical_dim (K) must be positive: it defines the shared space "
            "the core lives in, and without it the tensors have no shape"
        )
    if not modules:
        raise PathBError("config.target_modules cannot be empty: a core is produced per semantic module")
    if r <= 0:
        raise PathBError(f"config.rank must be positive, got {r}")
    if r > canonical_dim:
        raise PathBError(
            f"config.rank {r} exceeds canonical_dim {canonical_dim}: the low-rank "
            "factor cannot be wider than the space it acts on"
        )
    if not dataset:
        raise PathBError("dataset is empty: there is no corpus to fit")

    # Group records by the module their correction applies to.
    #
    # Without attribution there is exactly one thing the corpus can support: a
    # single core shared by every module. That is a real limitation, not a
    # neutral default — a core trained on unrelated fixes is not that module's
    # tuning — so it is only produced when the caller says it knows
    # (allow_shared_core), and the fact is recorded in the metadata.
    grouped = {}
    unattributed = []
    for index, rec in enumerate(dataset):
        if "context" not in rec or "corrected_completion" not in rec:
            raise PathBError(f"dataset record {index} is missing context or corrected_completion")
        module = str(rec.get("target_module") or "").strip()
        if module:
            grouped.setdefault(module, []).append(rec)
        else:
            unattributed.append(rec)

    if unattributed and not allow_shared:
        raise PathBError(
            f"{len(unattributed)} of {len(dataset)} dataset records carry no target_module, so there "
            "is nothing to say which layer their correction belongs to. Refusing to silently fit one "
            "core and present it as every module's tuning. Either attribute the records, or set "
            "config.allow_shared_core to accept a single shared core and have that recorded."
        )

    shared = None
    if allow_shared:
        # The shared core is fitted from everything attributed to no module, or
        # from the whole corpus if nothing was attributed at all.
        shared_records = unattributed or dataset
        shared = fit_core(shared_records, config, canonical_dim, seed)

    weights = {}
    counts = {}
    sources = {}
    losses_by_module = {}
    for module in modules:
        records = grouped.get(module)
        if records:
            # Fitted once per module, from that module's own records.
            a_core, b_core, losses = fit_core(records, config, canonical_dim, seed)
            sources[module] = "attributed"
            counts[module] = len(records)
        elif shared is not None:
            # Reused, not refitted: one shared core is fitted once and written to
            # every module that has no attribution of its own.
            a_core, b_core, losses = shared
            sources[module] = "shared_core"
            counts[module] = len(unattributed or dataset)
        else:
            raise PathBError(
                f"no dataset records are attributed to module {module!r} and no shared core was "
                "allowed, so this module would have no core at all. Attribute records to it, or set "
                "config.allow_shared_core."
            )
        weights[f"{module}/lora_A"] = a_core
        weights[f"{module}/lora_B"] = b_core
        losses_by_module[module] = losses[-1] if losses else None

    total_loss = losses_by_module.get(modules[0]) if modules else None

    metadata = {
        "ulora": json.dumps({
            "framework": "numpy",
            "engine": "path_b_canonical_fit",
            "supervision": "per_module_canonical_fit" if grouped else "shared_canonical_fit",
            # Stated plainly so nothing downstream reads this as trained LoRA.
            "is_transformer_lora_training": False,
            "embedding": "blake2b_feature_hash",
            "rank": r,
            "alpha": float(config.get("alpha", 32.0)),
            "canonical_dim": canonical_dim,
            "records": len(dataset),
            "records_per_module": counts,
            "core_sources": sources,
            "unattributed_records": len(unattributed),
            # True if ANY module was served by the shared core, not merely if the
            # corpus was wholly unattributed. A partial corpus with an explicit
            # opt-in still writes a shared core to the unattributed modules, and
            # reporting that as "not shared" would understate what the artifact is.
            "core_shared_across_modules": any(v == "shared_core" for v in sources.values()),
            "modules_with_attributed_core": sorted(m for m, v in sources.items() if v == "attributed"),
            "modules_with_shared_core": sorted(m for m, v in sources.items() if v == "shared_core"),
            "epochs": int(config.get("epochs", 50)),
            "loss_last": total_loss,
        }, sort_keys=True),
    }

    save_file(weights, output_path, metadata=metadata)
    print(json.dumps({
        "status": "ok",
        "engine": "path_b_canonical_fit",
        "tensors": sorted(weights.keys()),
        "records": len(dataset),
        "core_sources": sources,
        "loss_last": total_loss,
    }))


if __name__ == "__main__":
    try:
        main()
    except PathBError as exc:
        # A refusal is a result, not a crash.
        print(f"path_b_sft: {exc}", file=sys.stderr)
        sys.exit(2)
    except (KeyError, ValueError, OSError) as exc:
        # Unexpected: the message alone does not say where, so include the
        # traceback. It goes to stderr; stdout carries the JSON result.
        print(f"path_b_sft: {type(exc).__name__}: {exc}", file=sys.stderr)
        traceback.print_exc(file=sys.stderr)
        sys.exit(1)
