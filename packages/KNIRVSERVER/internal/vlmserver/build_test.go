package vlmserver

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestChooseBuildRTX3060IsCUDAForItsComputeCapability(t *testing.T) {
	h := Host{GPUName: "NVIDIA GeForce RTX 3060", GPUComputeCap: "8.6", VRAMTotal: 12288 * mib, HasAVX2: false}
	spec := ChooseBuild(h, "/usr/local/cuda-12.6/bin/nvcc")
	if !spec.CUDA || spec.Variant != "cuda-sm86" {
		t.Fatalf("variant=%s cuda=%v", spec.Variant, spec.CUDA)
	}
	args := strings.Join(spec.CMakeArgs, " ")
	if !strings.Contains(args, "-DGGML_CUDA=ON") || !strings.Contains(args, "-DCMAKE_CUDA_ARCHITECTURES=86") {
		t.Fatalf("cmake args %q", args)
	}
	if spec.CUDAHome != "/usr/local/cuda-12.6" {
		t.Fatalf("cuda home %q", spec.CUDAHome)
	}
}

// Without a toolkit the CUDA variant is still selected, so a prebuilt CUDA binary is found.
func TestChooseBuildWithoutNVCCKeepsCUDAVariantForPrebuilt(t *testing.T) {
	h := Host{GPUName: "NVIDIA GeForce RTX 3060", GPUComputeCap: "8.6", VRAMTotal: 12288 * mib, HasAVX2: true}
	spec := ChooseBuild(h, "")
	if !spec.CUDA || spec.Variant != "cuda-sm86" || spec.CUDAHome != "" {
		t.Fatalf("no nvcc must keep the CUDA variant (for a prebuilt): %+v", spec)
	}
}

// A prebuilt CUDA build that matches is used without any toolkit present.
func TestEnsureBuildReusesPrebuiltCUDAWithoutNVCC(t *testing.T) {
	dir := t.TempDir()
	h := Host{GPUName: "NVIDIA GeForce RTX 3060", GPUComputeCap: "8.6", VRAMTotal: 12288 * mib}
	spec := ChooseBuild(h, "")
	out := BinDirFor(dir, spec)
	os.MkdirAll(out, 0o755)
	os.WriteFile(filepath.Join(out, "llama-server"), []byte("x"), 0o755)
	rec := `{"variant":"cuda-sm86","commit":"` + LlamaCppCommit + `","cmake_args":["-DGGML_CUDA=ON","-DCMAKE_CUDA_ARCHITECTURES=86"]}`
	os.WriteFile(filepath.Join(out, "knirv-build.json"), []byte(rec), 0o644)
	got, err := EnsureBuild(context.Background(), dir, spec, nil)
	if err != nil || got != out {
		t.Fatalf("prebuilt must be used without nvcc: dir=%q err=%v", got, err)
	}
}

// Without a matching prebuilt and without nvcc, the build fails before any download.
func TestEnsureBuildWithoutPrebuiltOrNVCCFailsFast(t *testing.T) {
	h := Host{GPUName: "NVIDIA GeForce RTX 3060", GPUComputeCap: "8.6", VRAMTotal: 12288 * mib}
	spec := ChooseBuild(h, "")
	if _, err := EnsureBuild(context.Background(), t.TempDir(), spec, nil); err == nil {
		t.Fatal("expected an error: no prebuilt and no nvcc")
	}
}

func TestChooseBuildWithoutComputeCapIsCPU(t *testing.T) {
	h := Host{GPUName: "old driver", VRAMTotal: 4096 * mib}
	if spec := ChooseBuild(h, "/usr/local/cuda/bin/nvcc"); spec.CUDA {
		t.Fatal("a GPU whose compute capability is unknown must not get a CUDA build")
	}
}

func TestChooseBuildGenericCPUDisablesSIMD(t *testing.T) {
	spec := ChooseBuild(Host{CPUCores: 4}, "")
	if spec.Variant != "cpu-generic" {
		t.Fatalf("variant %s", spec.Variant)
	}
	args := strings.Join(spec.CMakeArgs, " ")
	if !strings.Contains(args, "-DGGML_NATIVE=OFF") || !strings.Contains(args, "-DGGML_AVX2=OFF") {
		t.Fatalf("generic build must not use host SIMD: %q", args)
	}
}

// A build is reused only when the variant, the flags and the pinned commit match.
func TestHasBuildReusesOnlyAMatchingRecord(t *testing.T) {
	dir := t.TempDir()
	spec := ChooseBuild(Host{HasAVX2: true}, "")
	out := BinDirFor(dir, spec)
	if err := os.MkdirAll(out, 0o755); err != nil {
		t.Fatal(err)
	}
	if hasBuild(dir, spec) {
		t.Fatal("no binary yet: must not be reused")
	}
	os.WriteFile(filepath.Join(out, "llama-server"), []byte("x"), 0o755)
	rec := `{"variant":"cpu-native","commit":"` + LlamaCppCommit + `","cmake_args":["-DGGML_NATIVE=ON"]}`
	os.WriteFile(filepath.Join(out, "knirv-build.json"), []byte(rec), 0o644)
	if !hasBuild(dir, spec) {
		t.Fatal("matching build must be reused")
	}
	stale := strings.Replace(rec, LlamaCppCommit, "0000000000000000000000000000000000000000", 1)
	os.WriteFile(filepath.Join(out, "knirv-build.json"), []byte(stale), 0o644)
	if hasBuild(dir, spec) {
		t.Fatal("a build from another commit must not be reused")
	}
}

func TestFindNVCCPrefersCUDACXX(t *testing.T) {
	fake := filepath.Join(t.TempDir(), "nvcc")
	os.WriteFile(fake, []byte("#!/bin/sh\n"), 0o755)
	t.Setenv("CUDACXX", fake)
	if got := FindNVCC(); got != fake {
		t.Fatalf("FindNVCC = %q, want %q", got, fake)
	}
}

func TestFindNVCCPrefersCUDA12OverNewerToolkit(t *testing.T) {
	root := t.TempDir()
	for _, v := range []string{"cuda-13.4", "cuda-12.6", "cuda-11.4"} {
		os.MkdirAll(filepath.Join(root, v, "bin"), 0o755)
		os.WriteFile(filepath.Join(root, v, "bin", "nvcc"), []byte("x"), 0o755)
	}
	old := cudaGlob
	cudaGlob = filepath.Join(root, "cuda-*")
	t.Cleanup(func() { cudaGlob = old })
	t.Setenv("CUDACXX", "")
	t.Setenv("PATH", "")
	if got := FindNVCC(); got != filepath.Join(root, "cuda-12.6", "bin", "nvcc") {
		t.Fatalf("FindNVCC = %q, want the CUDA 12.6 toolkit", got)
	}
}

func TestFindNVCCPrefers12OverNVCCOnPATH(t *testing.T) {
	root := t.TempDir()
	for _, v := range []string{"cuda-13.4", "cuda-12.6"} {
		os.MkdirAll(filepath.Join(root, v, "bin"), 0o755)
		os.WriteFile(filepath.Join(root, v, "bin", "nvcc"), []byte("x"), 0o755)
	}
	pathDir := t.TempDir()
	os.WriteFile(filepath.Join(pathDir, "nvcc"), []byte("x"), 0o755)
	old := cudaGlob
	cudaGlob = filepath.Join(root, "cuda-*")
	t.Cleanup(func() { cudaGlob = old })
	t.Setenv("CUDACXX", "")
	t.Setenv("PATH", pathDir)
	if got := FindNVCC(); got != filepath.Join(root, "cuda-12.6", "bin", "nvcc") {
		t.Fatalf("FindNVCC = %q, want the CUDA 12.6 toolkit over PATH", got)
	}
}

func TestWithEnvReplacesEveryEntry(t *testing.T) {
	env := withEnv([]string{"CUDACXX=", "A=1", "CUDACXX=old"}, "CUDACXX", "/new/nvcc")
	count := 0
	for _, kv := range env {
		if strings.HasPrefix(kv, "CUDACXX=") {
			count++
			if kv != "CUDACXX=/new/nvcc" {
				t.Fatalf("stale entry %q", kv)
			}
		}
	}
	if count != 1 {
		t.Fatalf("CUDACXX appears %d times", count)
	}
}
