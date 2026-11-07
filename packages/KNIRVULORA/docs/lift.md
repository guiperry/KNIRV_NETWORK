# Lift algebra (KNIRVULORA)

Source: `engines/connector_core.py`, `engines/derive_connector.py`.

- P_in shape: K x d_in (connector_core.py build_connector: `p_in = embed_rows(v_tgt, canonical_dim)`).
- P_out shape: d_out x K (`p_out = embed_cols(u_tgt @ rotation_out.T, canonical_dim)`).
- Orthonormality: P_in rows and P_out columns come from truncated SVD bases (truncated_basis) aligned via Procrustes when a source basis exists; standalone derivation uses identity rotation (align_bases returns identity when no source).
- A_core = A_native · P_inᵀ, B_core = P_outᵀ · B_native mapping per docstring.
- Well-defined only with real weight matrices (require_weight_matrix refuses missing weights); no RNG fallback.
EOF