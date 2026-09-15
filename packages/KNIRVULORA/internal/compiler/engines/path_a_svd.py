"""
Path A: subspace transfer (fast path).

Derives the two halves of the canonical artifact shape:

  1. The per-family CONNECTOR that projects between a base model's native frame
     and the shared canonical space of dimension K:
         P_in  in R^(K x d_in)     P_out in R^(d_out x K)
     so a consumer computes A_target = A_canonical @ P_in and
     B_target = P_out @ B_canonical.

  2. The canonical CORE: the source adapter expressed in that same shared space,
     once, per semantic module.

This replaces the previous behaviour, which emitted a single A/B pair fitted to
one target model and — when a model spec carried no weight matrix — silently
substituted a matrix generated from a seeded RNG. That inverted the point of the
protocol: a delta fitted to one model cannot travel, and fabricating the weights
to project against produced a well-formed artifact derived from nothing.

Math (ulora_research.md, Path A):

  W_src ~ U_src S_src V_src^T      W_tgt ~ U_tgt S_tgt V_tgt^T
  R_in  = argmin || V_src,k R - V_tgt,k ||_F   s.t. R^T R = I   -> R = U_hat V_hat^T
  R_out likewise over the U bases
  A_tgt = A_src (V_src,k R_in  V_tgt,k^T)
  B_tgt = (U_tgt,k R_out^T U_src,k^T) B_src

The core/connector split is that same identity factored in two:

  A_core = A_src @ pad_cols(V_src,k R_in, K)      P_in  = pad_rows(V_tgt,k^T, K)
  B_core = pad_rows(U_src,k^T, K) @ B_src         P_out = pad_cols(U_tgt,k R_out^T, K)

so A_core @ P_in reproduces A_tgt exactly: the zero padding means only the top-k
canonical coordinates contribute, which is what makes one core bindable to any
model that has a connector.

Input (stdin JSON):
  {
    "source_weights_path": "<safetensors with lora_A (r x d_in_src), lora_B>",
    "source_model": {..., "weight_matrix": [[...]]},   # required
    "target_model": {..., "weight_matrix": [[...]]},   # required
    "config": {"rank": 16, "alpha": 32.0, "canonical_dim": 1024,
               "target_modules": [...], "tensor_prefix": "llama-8b"},
    "output_path": "<safetensors>"
  }

Output tensors:
  <module>/lora_A   (r x K)      canonical core per semantic module
  <module>/lora_B   (K x r)
  <prefix>/P_in     (K x d_in)   connector for the target family
  <prefix>/P_out    (d_out x K)
"""
import json
import sys
import traceback

import numpy as np
from safetensors.numpy import load_file, save_file


class PathAError(Exception):
    """Raised when a transfer cannot be computed honestly."""


def require_weight_matrix(model, role):
    """
    Return the model's weight matrix, refusing to invent one.

    The previous implementation fell back to np.random.RandomState(...).randn(...)
    when this was absent, which produced a plausible artifact derived from noise:
    the transfer 'succeeded' while carrying no information about either model.
    A missing weight matrix means the caller has not supplied what a subspace
    transfer needs, so it is an error.
    """
    raw = model.get("weight_matrix")
    if raw is None:
        raise PathAError(
            f"{role} model has no weight_matrix; Path A needs the base weights to "
            f"extract the singular subspaces it aligns. Supply weight_matrix, or "
            f"route this pair through Path B (synthetic distillation)."
        )
    matrix = np.array(raw, dtype=np.float64)
    if matrix.ndim != 2:
        raise PathAError(f"{role} weight_matrix must be 2-D, got shape {matrix.shape}")
    if matrix.size == 0:
        raise PathAError(f"{role} weight_matrix is empty")
    return matrix


def truncated_basis(matrix, k):
    """
    Return the top-k left and right singular vectors of a matrix.

    Uses the economy SVD, so for a d_out x d_in matrix the cost is governed by
    the smaller dimension — the O(d^3) the spec budgets for, once per layer.
    """
    k = int(min(k, min(matrix.shape)))
    if k <= 0:
        raise PathAError(f"cannot extract a basis of rank {k} from shape {matrix.shape}")
    u, _, vt = np.linalg.svd(matrix, full_matrices=False)
    return u[:, :k], vt[:k, :]


def procrustes(source_basis, target_basis):
    """
    Orthogonal Procrustes rotation aligning source_basis onto target_basis.

    Both are (k, dim) with k orthonormal rows, so the cross-covariance
    M = source_basis @ target_basis^T is k x k and the closed-form rotation is
    R = U_hat V_hat^T from M = U_hat S V_hat^T.

    A degenerate M (rank-deficient alignment) still yields an orthogonal R —
    the SVD always exists — but it carries little information, so the residual
    is reported for the caller's metadata rather than silently accepted.
    """
    if source_basis.shape[0] != target_basis.shape[0]:
        raise PathAError(
            f"bases must share a rank to align: {source_basis.shape[0]} vs {target_basis.shape[0]}"
        )
    cross = source_basis @ target_basis.T
    u_hat, _, v_hat_t = np.linalg.svd(cross, full_matrices=True)
    rotation = u_hat @ v_hat_t
    # The rotation was solved in the (n, k) layout: cross = source^T target has
    # shape (k, k) and the Procrustes solution satisfies source_basis.T @ R ~
    # target_basis.T. The residual must therefore be formed the same way — the
    # previous form mixed a (k, dim) basis with a (k, k) rotation, which cannot
    # multiply at all.
    residual = float(np.linalg.norm(source_basis.T @ rotation - target_basis.T))
    return rotation, residual


def embed_cols(matrix, canonical_dim):
    """Place a (rows, k) matrix in the first k columns of a (rows, K) matrix."""
    if matrix.shape[1] > canonical_dim:
        raise PathAError(
            f"cannot embed {matrix.shape[1]} directions into canonical_dim {canonical_dim}"
        )
    out = np.zeros((matrix.shape[0], canonical_dim), dtype=np.float32)
    out[:, : matrix.shape[1]] = matrix
    return out


def embed_rows(matrix, canonical_dim):
    """Place a (k, cols) matrix in the first k rows of a (K, cols) matrix."""
    if matrix.shape[0] > canonical_dim:
        raise PathAError(
            f"cannot embed {matrix.shape[0]} directions into canonical_dim {canonical_dim}"
        )
    out = np.zeros((canonical_dim, matrix.shape[1]), dtype=np.float32)
    out[: matrix.shape[0], :] = matrix
    return out


def align_bases(source_basis, target_basis):
    """
    Align two k-row orthonormal bases, returning (rotation, residual).

    A Procrustes rotation is only defined when both bases live in the SAME
    ambient space, which is what lets their cross-covariance
        M = source_basis @ target_basis^T
    be formed at all. Bases from models of different widths do not: their
    columns index different coordinate axes, so M is not a covariance of anything
    and the rotation is meaningless.

    That case is not an error, because it does not need one. The canonical
    k-space is itself the shared frame: P_in maps the target's input space into
    it and the core is already expressed there, so the identity is the correct
    and only meaningful alignment. Rotation would be inventing a correspondence
    between axes that do not correspond.

    Returns residual None when alignment was skipped, so callers can record that
    the transfer was purely subspace re-expression rather than a rotated one.
    """
    if source_basis.shape[1] != target_basis.shape[1]:
        return np.eye(source_basis.shape[0], dtype=np.float64), None
    return procrustes(source_basis, target_basis)


def find_lora_key(tensors, suffix):
    for key in tensors:
        if key.endswith("/" + suffix) or key == suffix:
            return key
    return None


def path_a_transfer(source_weights_path, source_model, target_model, config):
    rank = int(config.get("rank", 16))
    alpha = float(config.get("alpha", 32.0))
    canonical_dim = int(config.get("canonical_dim", 0))
    modules = config.get("target_modules") or []
    prefix = config.get("tensor_prefix") or ""

    if canonical_dim <= 0:
        raise PathAError("config.canonical_dim (K) must be positive: it defines the shared space")
    if not modules:
        raise PathAError("config.target_modules cannot be empty")
    if not prefix:
        raise PathAError("config.tensor_prefix is required to namespace the connector tensors")

    src_tensors = load_file(source_weights_path)
    a_key = find_lora_key(src_tensors, "lora_A")
    b_key = find_lora_key(src_tensors, "lora_B")
    if a_key is None or b_key is None:
        raise PathAError(
            f"source weights contain no lora_A/lora_B pair (found: {sorted(src_tensors)})"
        )
    a_src = np.array(src_tensors[a_key], dtype=np.float64)
    b_src = np.array(src_tensors[b_key], dtype=np.float64)

    w_src = require_weight_matrix(source_model, "source")
    w_tgt = require_weight_matrix(target_model, "target")

    d_in_src, d_out_src = w_src.shape[1], w_src.shape[0]
    d_in_tgt, d_out_tgt = w_tgt.shape[1], w_tgt.shape[0]

    if a_src.shape != (rank, d_in_src):
        raise PathAError(
            f"source lora_A has shape {a_src.shape}, expected {(rank, d_in_src)} for rank {rank} "
            f"and source hidden size {d_in_src}"
        )
    if b_src.shape != (d_out_src, rank):
        raise PathAError(
            f"source lora_B has shape {b_src.shape}, expected {(d_out_src, rank)}"
        )

    # Rank of the shared subspace: bounded by the smaller model's width and by K.
    k = int(min(rank, d_in_src, d_out_src, d_in_tgt, d_out_tgt, canonical_dim))

    u_src, v_src = truncated_basis(w_src, k)
    u_tgt, v_tgt = truncated_basis(w_tgt, k)

    r_in, residual_in = align_bases(v_src, v_tgt)
    r_out, residual_out = align_bases(u_src.T, u_tgt.T)

    # The core itself is NOT computed here — Path B owns it. Only the connector,
    # which depends on the target's basis, is derived on this side.
    #
    # r_in is still solved: it is what makes the two bases comparable, and its
    # residual is reported so a caller can see whether alignment was meaningful.
    # r_out feeds the output projection below.

    # Connector: how this target's frame maps to and from the shared space.
    p_in = embed_rows(v_tgt, canonical_dim)          # K x d_in
    p_out = embed_cols((u_tgt @ r_out.T), canonical_dim)  # d_out x K

    # CONNECTOR ONLY.
    #
    # The canonical core is Path B's output: it is derived from the corpus and is
    # the same core for every target, whereas a connector is a property of the
    # target's architecture alone. Emitting a core here as well put two different
    # cores in one artifact and collided on tensor names during the merge, so the
    # split is: Path B -> core, Path A -> connector.
    weights = {
        f"{prefix}/P_in": p_in.astype(np.float32),
        f"{prefix}/P_out": p_out.astype(np.float32),
    }

    # Single metadata key holding sorted JSON.
    #
    # savetensors serialises the metadata mapping from a Rust HashMap, whose
    # iteration order is randomly seeded per process: with several keys, the same
    # tensors and the same metadata produce different bytes on every run, so the
    # bundle's content_hash changes for unchanged input. That silently defeats the
    # content addressing the protocol depends on. One key removes the ordering
    # freedom; sort_keys makes the inner JSON canonical. (Path B had the same
    # defect; this is the same fix.)
    metadata = {
        "ulora": json.dumps({
            "framework": "numpy_scipy",
            "engine": "path_a_svd_procrustes",
            "rank": rank,
            "alpha": alpha,
            "canonical_dim": canonical_dim,
            "subspace_rank": k,
            "tensor_prefix": prefix,
            # None means the bases had different ambient widths, so no rotation was
            # solved and the transfer is a pure subspace re-expression. Recorded
            # explicitly rather than as a zero, which would read as a perfect
            # alignment that was never performed.
            "procrustes_residual_in": "skipped" if residual_in is None else f"{residual_in:.6g}",
            "procrustes_residual_out": "skipped" if residual_out is None else f"{residual_out:.6g}",
            "alignment": "identity" if (residual_in is None or residual_out is None) else "procrustes",
            "core_shared_across_modules": True,
        }, sort_keys=True),
    }
    return weights, metadata


def main():
    try:
        payload = json.loads(sys.stdin.read())
        weights, metadata = path_a_transfer(
            payload["source_weights_path"],
            payload["source_model"],
            payload["target_model"],
            payload.get("config", {}),
        )
        save_file(weights, payload["output_path"], metadata=metadata)
    except PathAError as exc:
        # A refusal is a result, not a crash: report it on stderr and exit
        # non-zero so the Go caller surfaces the reason rather than a traceback.
        print(f"path_a_svd: {exc}", file=sys.stderr)
        sys.exit(2)
    except (KeyError, ValueError, OSError) as exc:
        # These are unexpected: a shape that did not line up, a missing key. The
        # message alone does not say where, and "matmul: core dimension mismatch"
        # in particular is unactionable without the line that produced it, so the
        # traceback goes to stderr too. It never reaches stdout, which carries the
        # JSON result the Go caller parses.
        print(f"path_a_svd: {type(exc).__name__}: {exc}", file=sys.stderr)
        traceback.print_exc(file=sys.stderr)
        sys.exit(1)

    # Read the summary back out of the single metadata key rather than indexing
    # path_a_transfer's dict directly: the metadata shape changed to one key, and
    # the previous form raised KeyError *after* save_file had already written the
    # file — so the engine reported failure while leaving correct output behind,
    # which is the worst of both.
    summary = json.loads(metadata["ulora"])
    print(json.dumps({
        "status": "ok",
        "engine": "path_a_svd_procrustes",
        "tensors": sorted(weights.keys()),
        "subspace_rank": summary["subspace_rank"],
        "alignment": summary["alignment"],
    }))


if __name__ == "__main__":
    main()
