#!/usr/bin/env python3
"""
Shared connector mathematics for the uLoRA engines.

A CONNECTOR is the projection pair that maps between a base model's native frame
and the shared canonical space of dimension K:

    P_in  in R^(K x d_in)     P_out in R^(d_out x K)

so a consumer computes A_target = A_core @ P_in and B_target = P_out @ B_core.

It is derived purely from the model's own principal subspaces, which is what
makes it a property of the MODEL rather than of a skill. Two callers need this:

  * path_a_svd.py     — as the connector half of a subspace transfer
  * derive_connector.py — standalone, to populate the runtime's per-model cache

They must produce identical output for the same model, or a bundle's expectation
and the cache's contents would disagree, so the mathematics lives here and both
import it. That was the reason to extract it: a second copy that merely looked
equivalent would be a divergence waiting to happen.
"""

import hashlib
import json
import sys

import numpy as np


class ConnectorError(Exception):
    """A refusal: the request cannot be honoured as stated."""


def require_weight_matrix(spec, role):
    """
    Return a model spec's weight matrix, or refuse.

    There is deliberately no fallback. A connector is built from the model's
    principal subspaces, so without real weights there is nothing to project
    against; generating one from a seeded RNG (as this code once did) yields a
    mathematically valid basis that aligns nothing, and the resulting adapter
    binds, loads and generates text while carrying the wrong meaning.
    """
    if not isinstance(spec, dict):
        raise ConnectorError(f"{role}_model must be an object")
    raw = spec.get("weight_matrix")
    if raw is None:
        raise ConnectorError(
            f"{role} model carries no weight_matrix, so there is no basis to derive a "
            "connector from. A connector is a projection built from the model's own "
            "principal subspaces; it cannot be invented without silently misaligning "
            "the adapter."
        )
    matrix = np.array(raw, dtype=np.float64)
    if matrix.ndim != 2:
        raise ConnectorError(f"{role} weight_matrix must be 2-D, got shape {matrix.shape}")
    if matrix.size == 0:
        raise ConnectorError(f"{role} weight_matrix is empty")
    return matrix


def truncated_basis(matrix, k):
    """
    Top-k left and right singular vectors of a matrix.

    Uses the economy SVD, so for a d_out x d_in matrix the cost is governed by the
    smaller dimension — the O(d^3) the spec budgets for, once per layer.
    """
    k = int(min(k, min(matrix.shape)))
    if k <= 0:
        raise ConnectorError(f"cannot extract a basis of rank {k} from shape {matrix.shape}")
    u, _, vt = np.linalg.svd(matrix, full_matrices=False)
    return u[:, :k], vt[:k, :]


def procrustes(source_basis, target_basis):
    """
    Orthogonal Procrustes rotation aligning source_basis onto target_basis.

    Both are (k, dim) with k orthonormal rows, so the cross-covariance
    M = source_basis @ target_basis^T is k x k and the closed-form rotation is
    R = U_hat V_hat^T from M = U_hat S V_hat^T.

    A degenerate M still yields an orthogonal R — the SVD always exists — but it
    carries little information, so the residual is reported rather than silently
    accepted.
    """
    if source_basis.shape[0] != target_basis.shape[0]:
        raise ConnectorError(
            f"bases must share a rank to align: {source_basis.shape[0]} vs {target_basis.shape[0]}"
        )
    cross = source_basis @ target_basis.T
    u_hat, _, v_hat_t = np.linalg.svd(cross, full_matrices=True)
    rotation = u_hat @ v_hat_t
    # The rotation was solved in the (n, k) layout: cross = source^T target has
    # shape (k, k) and the solution satisfies source_basis.T @ R ~ target_basis.T.
    # The residual must be formed the same way — mixing a (k, dim) basis with a
    # (k, k) rotation cannot multiply at all.
    residual = float(np.linalg.norm(source_basis.T @ rotation - target_basis.T))
    return rotation, residual


def align_bases(source_basis, target_basis):
    """
    Align two k-row orthonormal bases, returning (rotation, residual).

    A Procrustes rotation is only defined when both bases live in the SAME ambient
    space, which is what lets their cross-covariance be formed at all. Bases from
    models of different widths do not: their columns index different coordinate
    axes, so M is not a covariance of anything and the rotation is meaningless.

    That case needs no error, because the canonical k-space IS the shared frame:
    P_in maps the target's input space into it and the core is already expressed
    there, so identity is the correct and only meaningful alignment. Rotating
    would be inventing a correspondence between axes that do not correspond.

    Returns residual None when alignment was skipped, so callers can record that
    the transfer was a pure subspace re-expression.
    """
    if source_basis.shape[1] != target_basis.shape[1]:
        return np.eye(source_basis.shape[0], dtype=np.float64), None
    return procrustes(source_basis, target_basis)


def embed_cols(matrix, canonical_dim):
    """Place a (rows, k) matrix in the first k columns of a (rows, K) matrix."""
    if matrix.shape[1] > canonical_dim:
        raise ConnectorError(
            f"cannot embed {matrix.shape[1]} directions into canonical_dim {canonical_dim}"
        )
    out = np.zeros((matrix.shape[0], canonical_dim), dtype=np.float32)
    out[:, : matrix.shape[1]] = matrix
    return out


def embed_rows(matrix, canonical_dim):
    """Place a (k, cols) matrix in the first k rows of a (K, cols) matrix."""
    if matrix.shape[0] > canonical_dim:
        raise ConnectorError(
            f"cannot embed {matrix.shape[0]} directions into canonical_dim {canonical_dim}"
        )
    out = np.zeros((canonical_dim, matrix.shape[1]), dtype=np.float32)
    out[: matrix.shape[0], :] = matrix
    return out


def subspace_rank(w_src, w_tgt, rank, canonical_dim):
    """
    Rank of the shared subspace: bounded by both models' widths, the adapter rank,
    and K.
    """
    return int(min(rank, w_src.shape[0], w_src.shape[1], w_tgt.shape[0], w_tgt.shape[1], canonical_dim))


def build_connector(w_tgt, canonical_dim, rank=None, w_src=None, rotation_out=None):
    """
    Build P_in and P_out for a target model.

    The projection comes from the target's own principal subspaces:
        P_in  = embed_rows(V_tgt, K)          K x d_in
        P_out = embed_cols(U_tgt @ R_out^T, K) d_out x K

    R_out is the optional output-side rotation. It is only meaningful when a
    source model of the same width was supplied, in which case the connector
    carries a relative alignment as well as the target's basis. With no source
    (standalone derivation) the rotation is identity, which is the pure
    model-intrinsic projection the cache stores.

    Returns (p_in, p_out, info) where info records the rank actually used and
    whether an alignment was applied, for the caller's metadata.
    """
    if canonical_dim <= 0:
        raise ConnectorError(
            "canonical_dim (K) must be positive: it defines the shared space and "
            "therefore every connector's shape"
        )

    if rank is None:
        rank = canonical_dim
    rank = int(min(rank, min(w_tgt.shape), canonical_dim))
    if rank <= 0:
        raise ConnectorError(f"cannot build a connector of rank {rank} from shape {w_tgt.shape}")

    u_tgt, v_tgt = truncated_basis(w_tgt, rank)

    if rotation_out is None:
        if w_src is not None:
            u_src, _ = truncated_basis(w_src, rank)
            rotation_out, residual_out = align_bases(u_src.T, u_tgt.T)
        else:
            rotation_out, residual_out = np.eye(rank, dtype=np.float64), None
    else:
        residual_out = None

    p_in = embed_rows(v_tgt, canonical_dim)                    # K x d_in
    p_out = embed_cols(u_tgt @ rotation_out.T, canonical_dim)  # d_out x K

    info = {
        "subspace_rank": rank,
        "canonical_dim": canonical_dim,
        "alignment": "identity" if residual_out is None else "procrustes",
        "procrustes_residual_out": None if residual_out is None else f"{residual_out:.6g}",
    }
    return p_in, p_out, info


def connector_metadata(engine, info, extra=None):
    """
    Single metadata key holding sorted JSON.

    savetensors serialises the metadata mapping from a Rust HashMap, whose
    iteration order is randomly seeded per process: with several keys the same
    tensors and the same metadata produce different bytes on every run, so a
    content hash changes for unchanged input. One key removes the ordering
    freedom; sort_keys makes the inner JSON canonical.
    """
    payload = {"framework": "numpy_scipy", "engine": engine}
    payload.update(info)
    if extra:
        payload.update(extra)
    return {"ulora": json.dumps(payload, sort_keys=True)}


def fail(message, code=2):
    """Report a refusal on stderr and exit non-zero."""
    print(f"{message}", file=sys.stderr)
    sys.exit(code)


def fail_unexpected(exc):
    """Report an unexpected failure with its location, then exit non-zero."""
    import traceback

    print(f"{type(exc).__name__}: {exc}", file=sys.stderr)
    traceback.print_exc(file=sys.stderr)
    sys.exit(1)
