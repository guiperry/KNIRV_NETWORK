# ULoRA Third-Party Dependencies

This directory contains vendored third-party dependencies for the CPU-only MLC Engine build.

## Build

Run the build script from the package root:

```bash
cd packages/KNIRVULORA
./scripts/build-mlc-cpu.sh
```

## Contents (built by build-mlc-cpu.sh)

- `tvm-unity/` — Apache TVM Unity (relax) with LLVM backend only. No CUDA/Vulkan.
- `mlc-llm/` — MLC LLM with CPU-only configuration.

## Quantization

For CPU targets, use `q4f32_1` or `q0f32` quantization formats (32-bit accumulation),
not FP16-dependent formats like `q4f16_1`.
