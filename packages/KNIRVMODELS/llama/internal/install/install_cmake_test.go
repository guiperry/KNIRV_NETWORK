package install

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCMakeConfigureArgsCPUDefault(t *testing.T) {
	got := cmakeConfigureArgs("/src", "/src/build", false)
	want := []string{"-S", "/src", "-B", "/src/build", "-DLLAMA_BUILD_SERVER=ON", "-DLLAMA_USE_PREBUILT_UI=OFF", "-DLLAMA_BUILD_UI=OFF", "-DGGML_CUDA=OFF"}
	if strings.Join(got, " ") != strings.Join(want, " ") {
		t.Fatalf("args = %v\nwant %v", got, want)
	}
}

func TestCMakeConfigureArgsCUDA(t *testing.T) {
	got := cmakeConfigureArgs("/src", "/src/build", true)
	if !contains(got, "-DGGML_CUDA=ON") || contains(got, "-DGGML_CUDA=OFF") {
		t.Fatalf("args = %v; want -DGGML_CUDA=ON only", got)
	}
	if !contains(got, "-DLLAMA_USE_PREBUILT_UI=OFF") {
		t.Fatalf("args = %v; want -DLLAMA_USE_PREBUILT_UI=OFF", got)
	}
}

func TestCUDARequested(t *testing.T) {
	cudaDir := t.TempDir()
	noNvcc := func(string) (string, error) { return "", os.ErrNotExist }
	yesNvcc := func(string) (string, error) { return "/usr/bin/nvcc", nil }
	env := func(m map[string]string) func(string) string {
		return func(k string) string { return m[k] }
	}
	cases := []struct {
		name string
		env  map[string]string
		look func(string) (string, error)
		want bool
	}{
		{"nothing detected", nil, noNvcc, false},
		{"nvcc on PATH", nil, yesNvcc, true},
		{"CUDA_HOME directory", map[string]string{"CUDA_HOME": cudaDir}, noNvcc, true},
		{"CUDA_HOME missing directory", map[string]string{"CUDA_HOME": filepath.Join(cudaDir, "nope")}, noNvcc, false},
		{"override 0 beats nvcc", map[string]string{"KNIRV_LLAMA_CUDA": "0"}, yesNvcc, false},
		{"override 1 beats no toolkit", map[string]string{"KNIRV_LLAMA_CUDA": "1"}, noNvcc, true},
		{"unknown override value detects", map[string]string{"KNIRV_LLAMA_CUDA": "maybe"}, yesNvcc, true},
	}
	for _, c := range cases {
		if got := cudaRequested(env(c.env), c.look); got != c.want {
			t.Errorf("%s: cudaRequested = %v, want %v", c.name, got, c.want)
		}
	}
}

// installServer passes the CMake arguments from the environment and detection to
// the configure step. The build itself is faked.
func TestInstallServerConfigureUsesCPUWhenOverridden(t *testing.T) {
	t.Setenv("KNIRV_LLAMA_CUDA", "0")
	dir := t.TempDir()
	built := filepath.Join(dir, "llama.cpp", "build", "bin", "llama-server")
	var configure []string
	i := New()
	i.LookPath = func(string) (string, error) { return "/usr/bin/x", nil }
	i.Output = func(context.Context, string, ...string) (string, error) { return "version: 6200 (head)\n", nil }
	i.Run = func(_ context.Context, name string, args ...string) error {
		if name == "cmake" && contains(args, "-B") {
			configure = args
		}
		if name == "cmake" && contains(args, "--build") {
			writeScript(t, built, "version: 6200 (head)")
		}
		return nil
	}
	if _, err := i.installServer(context.Background(), dir, ""); err != nil {
		t.Fatal(err)
	}
	if !contains(configure, "-DGGML_CUDA=OFF") || !contains(configure, "-DLLAMA_USE_PREBUILT_UI=OFF") {
		t.Fatalf("configure args = %v; want CPU build with prebuilt UI off", configure)
	}
}
