//! Permanent GPU-backward sanity tests for Task 4.4.
//!
//! These pin the fixes from the 2026-11-06 investigation:
//!
//! Gated on BOTH `cuda` and `diag`: the tests spawn the trainer's `probe-gradparity` /
//! `probe-overfit` subcommands, which exist only when the `diag` feature is enabled. Without
//! `diag` the default binary omits those commands, so the tests are excluded (they would fail
//! trying to run a non-existent subcommand). Build with `--features cuda,diag` to run them.
//!  - candle 0.11's CUDA backward is broken for stride-0 (broadcast) weight operands in
//!    `broadcast_mul`/`matmul`/`broadcast_add` (it zeroed the set head's gradient at the
//!    broadcast dimension), and for rank>=4 / rank>=3 softmax tensors. Both are worked around in
//!    `knirv-heart-decision`: weight copies are `repeat`-materialized (or plain 2D matmul) and
//!    `softmax_last_dim_diff` flattens every softmax to 2D before a stable `exp(x - logsumexp(x))`.
//!    If a future candle/toolkit change regresses the CUDA backward, these tests fail loudly
//!    instead of silently producing an untrained model.
//!
//! Gated on CUDA availability: on CPU-only machines the tests pass trivially (the CPU path is the
//! reference), so CI without a GPU stays green.

#![cfg(all(feature = "cuda", feature = "diag"))]

use std::process::Command;

fn trainer() -> Command {
    Command::new(env!("CARGO_BIN_EXE_heart-decision-trainer"))
}

fn cuda_available() -> bool {
    candle_core::Device::cuda_if_available(0)
        .map(|d| d.is_cuda())
        .unwrap_or(false)
}

/// CPU-vs-GPU gradient parity: no zero-grad params, and every meaningful-gradient param has a
/// cosine above the fp32-noise floor. Catches a silently wrong CUDA backward.
#[test]
fn gpu_backward_parity_matches_cpu() {
    if !cuda_available() {
        eprintln!("no CUDA device; skipping GPU parity test");
        return;
    }
    let out = trainer()
        .args(["probe-gradparity", "--task", "a", "--preset", "tiny", "--seed", "1"])
        .output()
        .expect("run trainer");
    let stdout = String::from_utf8_lossy(&out.stdout);
    let stderr = String::from_utf8_lossy(&out.stderr);
    let combined = format!("{stdout}{stderr}");
    assert!(out.status.success(), "gradparity failed:\n{combined}");
    assert!(combined.contains("GRADPARITY GATE PASS"), "gradparity gate not passed:\n{combined}");
}

/// GPU overfit: the optimizer must drive the loss below 0.05 on a tiny memorizable set within the
/// step budget (proves gradients actually flow on GPU). Exit non-zero otherwise.
#[test]
fn gpu_overfit_converges() {
    if !cuda_available() {
        eprintln!("no CUDA device; skipping GPU overfit test");
        return;
    }
    let out = trainer()
        .args([
            "probe-overfit",
            "--task",
            "a",
            "--preset",
            "tiny",
            "--steps",
            "150",
            "--seed",
            "1",
            "--lr",
            "0.003",
            "--const-lr",
            "--no-feature",
        ])
        .output()
        .expect("run trainer");
    let stdout = String::from_utf8_lossy(&out.stdout);
    let stderr = String::from_utf8_lossy(&out.stderr);
    let combined = format!("{stdout}{stderr}");
    assert!(out.status.success(), "overfit failed:\n{combined}");
    assert!(combined.contains("OVERFIT GATE PASS"), "overfit gate not passed:\n{combined}");
}
