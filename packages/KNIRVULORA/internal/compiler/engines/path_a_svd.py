#!/usr/bin/env python3
"""
Path A: Fast Subspace Transfer engine via SVD + Orthogonal Procrustes.

Given a source adapter's (A_src, B_src) LoRA matrices and a source/target
base model weight pair, projects the source LoRA into the target model's
coordinate frame:

1. SVD of source and target base weights to extract principal subspaces.
2. Orthogonal Procrustes alignment: R = U_hat @ V_hat^T from SVD of
   M = V_src^T @ V_tgt.
3. Project:
   A_tgt = A_src @ (V_src_k @ R_in @ V_tgt_k^T)
   B_tgt = (U_tgt_k @ R_out^T @ U_src_k^T) @ B_src

Input JSON (stdin):
{
  "source_weights_path": "/path/to/source.safetensors",
  "source_model": {"hidden_size": ...},
  "target_model": {"hidden_size": ...},
  "config": {"rank": ..., "alpha": ...}
}

Output: writes safetensors to "output_path" with projected lora_A, lora_B.
"""

import sys
import json
import numpy as np
from scipy.linalg import svd
from safetensors.numpy import save_file, load_file


def truncated_svd(W, k=None):
    """Compute truncated SVD, returning (U_k, S_k, Vt_k)."""
    if k is None:
        k = min(W.shape)
    U, S, Vt = svd(W, full_matrices=False)
    k = min(k, len(S))
    return U[:, :k], S[:k], Vt[:k, :]


def orthogonal_procrustes(A, B):
    """
    Find orthogonal R minimizing ||A - B @ R||_F.
    Returns R = U @ V^T from SVD of B^T @ A = U @ S @ V^T.
    """
    M = B.T @ A
    U, _, Vt = svd(M, full_matrices=False)
    R = U @ Vt
    return R


def path_a_transfer(source_weights_path, source_model, target_model, config):
    rank = config.get("rank", 16)
    alpha = config.get("alpha", 32.0)

    src_tensors = load_file(source_weights_path)
    
    def find_lora_key(tensors, suffix):
        for k in tensors.keys():
            if k.endswith("/" + suffix) or k == suffix:
                return k
        return suffix
    
    a_key = find_lora_key(src_tensors, "lora_A")
    b_key = find_lora_key(src_tensors, "lora_B")
    A_src = src_tensors[a_key]
    B_src = src_tensors[b_key]

    W_src_raw = source_model.get("weight_matrix")
    W_tgt_raw = target_model.get("weight_matrix")

    d_src_in = source_model.get("hidden_size", 4096)
    d_tgt_in = target_model.get("hidden_size", 4096)
    d_src_out = source_model.get("hidden_size", 4096)
    d_tgt_out = target_model.get("hidden_size", 4096)

    if W_src_raw is not None:
        W_src = np.array(W_src_raw, dtype=np.float32)
    else:
        rng = np.random.RandomState(42)
        W_src = rng.randn(d_src_out, d_src_in).astype(np.float32)

    if W_tgt_raw is not None:
        W_tgt = np.array(W_tgt_raw, dtype=np.float32)
    else:
        rng = np.random.RandomState(84)
        W_tgt = rng.randn(d_tgt_out, d_tgt_in).astype(np.float32)

    k = min(rank, min(W_src.shape), min(W_tgt.shape))

    U_src, S_src, Vt_src = truncated_svd(W_src, k)
    U_tgt, S_tgt, Vt_tgt = truncated_svd(W_tgt, k)

    V_src_k = Vt_src[:k, :].T
    V_tgt_k = Vt_tgt[:k, :].T
    U_src_k = U_src[:, :k]
    U_tgt_k = U_tgt[:, :k]

    R_in = orthogonal_procrustes(V_tgt_k, V_src_k)
    R_out = orthogonal_procrustes(U_tgt_k, U_src_k)

    P_in = V_src_k @ R_in @ V_tgt_k.T
    P_out = U_tgt_k @ R_out.T @ U_src_k.T

    if A_src.shape[1] != d_src_in:
        if A_src.shape[1] < d_src_in:
            A_src = np.pad(A_src, ((0, 0), (0, d_src_in - A_src.shape[1])))
        else:
            A_src = A_src[:, :d_src_in]

    if B_src.shape[0] != d_src_out:
        if B_src.shape[0] < d_src_out:
            B_src = np.pad(B_src, ((0, d_src_out - B_src.shape[0]), (0, 0)))
        else:
            B_src = B_src[:d_src_out, :]

    A_tgt = A_src @ P_in
    B_tgt = P_out @ B_src

    if d_tgt_in > A_tgt.shape[1]:
        A_tgt = np.pad(A_tgt, ((0, 0), (0, d_tgt_in - A_tgt.shape[1])))
    elif d_tgt_in < A_tgt.shape[1]:
        A_tgt = A_tgt[:, :d_tgt_in]

    if d_tgt_out > B_tgt.shape[0]:
        B_tgt = np.pad(B_tgt, ((0, d_tgt_out - B_tgt.shape[0]), (0, 0)))
    elif d_tgt_out < B_tgt.shape[0]:
        B_tgt = B_tgt[:d_tgt_out, :]

    return {
        "lora_A": A_tgt.astype(np.float32),
        "lora_B": B_tgt.astype(np.float32),
        "config_rank": np.array([rank], dtype=np.int64),
        "config_alpha": np.array([alpha], dtype=np.float32),
        "scaling": np.array([alpha / rank], dtype=np.float32),
    }


def main():
    raw = sys.stdin.read()
    payload = json.loads(raw)

    source_weights_path = payload["source_weights_path"]
    source_model = payload["source_model"]
    target_model = payload["target_model"]
    config = payload.get("config", {})
    output_path = payload["output_path"]

    weights = path_a_transfer(source_weights_path, source_model, target_model, config)

    metadata = {
        "framework": "numpy_scipy",
        "engine": "path_a_svd_procrustes",
        "rank": int(weights["config_rank"][0]),
        "alpha": float(weights["config_alpha"][0]),
    }

    save_file(weights, output_path, metadata=metadata)
    print(json.dumps({"status": "ok", "engine": "path_a_svd_procrustes", "tensors": list(weights.keys())}))


if __name__ == "__main__":
    main()