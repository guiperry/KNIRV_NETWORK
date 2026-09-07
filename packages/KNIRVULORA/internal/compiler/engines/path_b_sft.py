#!/usr/bin/env python3
"""
Path B: Direct SFT (Supervised Fine-Tuning) engine.

Given a corpus of (context, corrected_completion) pairs and a target base
model's dimension spec, learns a low-rank adapter (A, B matrices) via
gradient descent minimizing cross-entropy loss on the corpus.

The adapter is learned in a toy embedding space: each context/completion
pair is vector-encoded (hash-based bag-of-features → fixed dim), and the
adapter learns a residual correction map:

    h_corrected = h_context + (alpha/r) * B @ A @ h_context

where A is (r, d_in) and B is (d_out, r).

Input JSON (stdin):
{
  "dataset": [{"context": "...", "corrected_completion": "...", "target_model": "..."}],
  "target_model": {"hidden_size": 4096, "intermediate_size": 14336, ...},
  "config": {"rank": 16, "alpha": 32.0, "learning_rate": 0.01, "epochs": 50}
}

Output: writes a safetensors file to the path given by "output_path" containing
tensors named "lora_A" and "lora_B".
"""

import sys
import json
import numpy as np
from safetensors.numpy import save_file, load_file


def encode_text(text: str, dim: int) -> np.ndarray:
    """Hash-based bag-of-features encoding of text into a fixed-dim vector."""
    vec = np.zeros(dim, dtype=np.float32)
    tokens = text.lower().split()
    if not tokens:
        tokens = [text[:1] if text else " "]
    for tok in tokens:
        h = hash(tok)
        for i in range(dim):
            vec[i] += ((h >> i) & 1) * 0.5 + 0.5 * np.sin(h * (i + 1) * 0.01)
    norm = np.linalg.norm(vec)
    if norm > 0:
        vec = vec / norm
    return vec


def compute_target_delta(context_enc: np.ndarray, completion_enc: np.ndarray) -> np.ndarray:
    """
    The 'correct' correction is the difference between the desired completion
    embedding and what the context alone produces. In a real model this delta
    would be the residual the LoRA must inject; here we compute it directly
    from the paired embeddings.
    """
    return completion_enc - context_enc


def train_sft(dataset, target_model, config):
    """Train a low-rank adapter via SGD on the corpus."""
    d_in = target_model.get("hidden_size", target_model.get("intermediate_size", 1024))
    d_out = d_in
    r = config.get("rank", 16)
    alpha = config.get("alpha", 32.0)
    lr = config.get("learning_rate", 0.01)
    epochs = config.get("epochs", 50)
    seed = config.get("seed", 42)
    np.random.seed(seed)

    encode_dim = config.get("encode_dim", min(d_in, 512))

    # Encode all pairs
    inputs = []
    targets = []
    for rec in dataset:
        ctx = encode_text(rec["context"], encode_dim)
        comp = encode_text(rec["corrected_completion"], encode_dim)
        delta = compute_target_delta(ctx, comp)
        inputs.append(ctx)
        targets.append(delta)

    X = np.stack(inputs)
    Y = np.stack(targets)

    # Pad or truncate embeddings to model dimension
    if encode_dim < d_in:
        X_padded = np.zeros((X.shape[0], d_in), dtype=np.float32)
        Y_padded = np.zeros((Y.shape[0], d_out), dtype=np.float32)
        X_padded[:, :encode_dim] = X
        Y_padded[:, :encode_dim] = Y
        X = X_padded
        Y = Y_padded
    elif encode_dim > d_in:
        X = X[:, :d_in]
        Y = Y[:, :d_out]

    n = X.shape[0]

    # Initialize A (r, d_in) and B (d_out, r) with small random values
    scale = 0.01
    A = np.random.randn(r, d_in).astype(np.float32) * scale
    B = np.random.randn(d_out, r).astype(np.float32) * scale

    scaling = alpha / r

    # SGD training loop
    for epoch in range(epochs):
        indices = np.random.permutation(n)
        for idx in indices:
            x = X[idx]
            y_true = Y[idx]

            # Forward: delta = scaling * B @ A @ x
            Ax = A @ x
            delta = scaling * (B @ Ax)

            # Loss: MSE between predicted delta and target delta
            # (proxy for cross-entropy in the embedding space)
            error = delta - y_true
            loss = np.mean(error ** 2)

            # Gradients
            # dL/delta = 2 * error / d_out
            d_delta = (2.0 / d_out) * error

            # dL/dB = d_delta @ Ax^T * scaling
            grad_B = np.outer(d_delta, Ax) * scaling
            # dL/dA = scaling * B^T @ d_delta @ x^T
            grad_A = scaling * (B.T @ d_delta).reshape(r, 1) * x.reshape(1, -1)

            A -= lr * grad_A
            B -= lr * grad_B

    return {
        "lora_A": A.astype(np.float32),
        "lora_B": B.astype(np.float32),
        "config_rank": np.array([r], dtype=np.int64),
        "config_alpha": np.array([alpha], dtype=np.float32),
        "scaling": np.array([scaling], dtype=np.float32),
    }


def main():
    raw = sys.stdin.read()
    payload = json.loads(raw)

    dataset = payload["dataset"]
    target_model = payload["target_model"]
    config = payload.get("config", {})
    output_path = payload["output_path"]

    weights = train_sft(dataset, target_model, config)

    metadata = {
        "framework": "numpy",
        "engine": "path_b_sft",
        "rank": weights["config_rank"][0],
        "alpha": float(weights["config_alpha"][0]),
    }

    save_file(weights, output_path, metadata=metadata)
    print(json.dumps({"status": "ok", "engine": "path_b_sft", "tensors": list(weights.keys())}))


if __name__ == "__main__":
    main()
