package vlmserver

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"time"
)

// LlamaCppRepo and LlamaCppCommit pin the llama.cpp source the launcher builds. The commit
// is the one the MiniCPM model and the CUDA build were verified against.
const (
	LlamaCppRepo   = "https://github.com/ggml-org/llama.cpp.git"
	LlamaCppCommit = "785322d17f2d51e5185cf996a1f4e46d87640841"
)

// BuildSpec is one llama-server build, chosen from the host.
type BuildSpec struct {
	Variant   string   // directory name under <data>/vlm/bin: cuda-sm86, cpu-native, cpu-generic
	CMakeArgs []string // flags beyond the common set
	CUDA      bool
	CUDAHome  string // nvcc's installation directory when CUDA is set
	Reason    string
}

// cudaGlob locates CUDA toolkit installs (a variable so tests can point it elsewhere).
var cudaGlob = "/usr/local/cuda-*"

// preferredCUDAMajor is the CUDA toolkit series the pinned build is verified with.
const preferredCUDAMajor = 12

// nvccPattern matches the version directory of a CUDA toolkit install.
var nvccPattern = regexp.MustCompile(`^cuda-(\d+)\.(\d+)$`)

// FindNVCC returns the nvcc to build CUDA with. Order: CUDACXX; then the newest CUDA 12
// toolkit install (see preferredCUDAMajor); then nvcc on PATH; then any other toolkit
// install. It returns "" when none exists.
func FindNVCC() string {
	if p := strings.TrimSpace(os.Getenv("CUDACXX")); p != "" {
		if st, err := os.Stat(p); err == nil && !st.IsDir() {
			return p
		}
	}
	type cand struct {
		path         string
		major, minor int
	}
	var cands []cand
	dirs, _ := filepath.Glob(cudaGlob)
	for _, d := range dirs {
		m := nvccPattern.FindStringSubmatch(filepath.Base(d))
		if m == nil {
			continue
		}
		var maj, min int
		fmt.Sscanf(m[1]+" "+m[2], "%d %d", &maj, &min)
		cands = append(cands, cand{filepath.Join(d, "bin", "nvcc"), maj, min})
	}
	sort.Slice(cands, func(i, j int) bool {
		pi, pj := cands[i].major == preferredCUDAMajor, cands[j].major == preferredCUDAMajor
		if pi != pj {
			return pi
		}
		if cands[i].major != cands[j].major {
			return cands[i].major > cands[j].major
		}
		return cands[i].minor > cands[j].minor
	})
	if len(cands) > 0 && cands[0].major == preferredCUDAMajor {
		if st, err := os.Stat(cands[0].path); err == nil && !st.IsDir() {
			return cands[0].path
		}
	}
	if p, err := exec.LookPath("nvcc"); err == nil {
		return p
	}
	for _, c := range cands {
		if st, err := os.Stat(c.path); err == nil && !st.IsDir() {
			return c.path
		}
	}
	if st, err := os.Stat("/usr/local/cuda/bin/nvcc"); err == nil && !st.IsDir() {
		return "/usr/local/cuda/bin/nvcc"
	}
	return ""
}

// ChooseBuild picks the build for this host. CUDA needs a GPU that reports its compute
// capability and an nvcc to compile with. Without them the CPU build is used, and the
// reason is recorded so the operator can see what was missing.
func ChooseBuild(h Host, nvcc string) BuildSpec {
	if h.VRAMTotal > 0 && h.GPUComputeCap != "" {
		sm := strings.ReplaceAll(h.GPUComputeCap, ".", "")
		spec := BuildSpec{
			Variant: "cuda-sm" + sm,
			CUDA:    true,
			CMakeArgs: []string{
				"-DGGML_CUDA=ON",
				"-DCMAKE_CUDA_ARCHITECTURES=" + sm,
			},
		}
		if nvcc != "" {
			spec.CUDAHome = filepath.Dir(filepath.Dir(nvcc))
			spec.Reason = fmt.Sprintf("%s (sm_%s) with nvcc %s", h.GPUName, sm, nvcc)
		} else {
			// No toolkit here (a container with only the CUDA runtime, for example). The
			// spec is kept so a prebuilt CUDA build for this variant is still used; without
			// one, EnsureBuild fails and the caller falls back to the CPU build.
			spec.Reason = fmt.Sprintf("%s (sm_%s) without nvcc: needs a prebuilt %s build", h.GPUName, sm, spec.Variant)
		}
		return spec
	}
	if h.HasAVX2 {
		return BuildSpec{
			Variant:   "cpu-native",
			CMakeArgs: []string{"-DGGML_NATIVE=ON"},
			Reason:    "CPU with AVX2",
		}
	}
	// Generic CPU: no SIMD flags beyond the x86-64 baseline, so the binary runs on any
	// host of this architecture.
	return BuildSpec{
		Variant:   "cpu-generic",
		CMakeArgs: []string{"-DGGML_NATIVE=OFF", "-DGGML_AVX=OFF", "-DGGML_AVX2=OFF", "-DGGML_FMA=OFF", "-DGGML_F16C=OFF", "-DGGML_BMI2=OFF"},
		Reason:    "CPU without AVX2 (or not detected)",
	}
}

// buildRecord is written beside a finished build. A build is reused only while the
// commit, the variant and the flags match it.
type buildRecord struct {
	Variant   string   `json:"variant"`
	Commit    string   `json:"commit"`
	CMakeArgs []string `json:"cmake_args"`
	BuiltAt   string   `json:"built_at"`
}

// BinDirFor is where a variant's build lives.
func BinDirFor(dataDir string, spec BuildSpec) string {
	return filepath.Join(dataDir, "bin", spec.Variant)
}

// hasBuild reports whether bin for spec already exists and matches the pinned source.
func hasBuild(dataDir string, spec BuildSpec) bool {
	dir := BinDirFor(dataDir, spec)
	if _, err := os.Stat(filepath.Join(dir, "llama-server")); err != nil {
		return false
	}
	b, err := os.ReadFile(filepath.Join(dir, "knirv-build.json"))
	if err != nil {
		return false
	}
	var rec buildRecord
	if json.Unmarshal(b, &rec) != nil {
		return false
	}
	return rec.Commit == LlamaCppCommit && rec.Variant == spec.Variant && equalArgs(rec.CMakeArgs, spec.CMakeArgs)
}

func equalArgs(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// EnsureBuild returns the directory holding llama-server for spec, building it first when
// no matching build exists. The build runs in the background-capable context; it is not
// fatal to the caller, which may fall back to another variant.
func EnsureBuild(ctx context.Context, dataDir string, spec BuildSpec, logf func(string, ...any)) (string, error) {
	if logf == nil {
		logf = func(string, ...any) {}
	}
	if hasBuild(dataDir, spec) {
		return BinDirFor(dataDir, spec), nil
	}
	if err := requireTools(spec); err != nil {
		return "", err
	}
	srcDir := filepath.Join(dataDir, "src", "llama.cpp")
	if err := ensureSource(ctx, srcDir, logf); err != nil {
		return "", err
	}
	buildDir := filepath.Join(srcDir, "build-"+spec.Variant)
	// Configure from scratch: a cache from an earlier attempt can pin the wrong compiler.
	if err := os.RemoveAll(buildDir); err != nil {
		return "", err
	}
	jobs := runtime.NumCPU()
	if jobs < 1 {
		jobs = 1
	}

	configure := []string{"-S", srcDir, "-B", buildDir,
		"-DCMAKE_BUILD_TYPE=Release", "-DLLAMA_CURL=OFF", "-DLLAMA_BUILD_TESTS=OFF"}
	configure = append(configure, spec.CMakeArgs...)
	if spec.CUDA {
		// Name the compiler and toolkit explicitly, so CMake cannot pick up another nvcc.
		configure = append(configure,
			"-DCMAKE_CUDA_COMPILER="+filepath.Join(spec.CUDAHome, "bin", "nvcc"),
			"-DCUDAToolkit_ROOT="+spec.CUDAHome)
	}
	env := os.Environ()
	if spec.CUDA {
		nvcc := filepath.Join(spec.CUDAHome, "bin", "nvcc")
		env = withEnv(env, "CUDACXX", nvcc)
		env = withEnv(env, "CUDA_PATH", spec.CUDAHome)
		env = withEnv(env, "PATH", filepath.Join(spec.CUDAHome, "bin")+string(os.PathListSeparator)+os.Getenv("PATH"))
	}
	logf("[vlm] building llama-server (%s): %s", spec.Variant, spec.Reason)
	if err := runCmd(ctx, env, logf, "cmake", configure...); err != nil {
		return "", fmt.Errorf("cmake configure: %w", err)
	}
	if err := runCmd(ctx, env, logf, "cmake", "--build", buildDir, "--config", "Release", "-j", fmt.Sprint(jobs), "--target", "llama-server"); err != nil {
		return "", fmt.Errorf("cmake build: %w", err)
	}

	// Copy the executable and every shared library beside it, so its rpath still resolves.
	out := BinDirFor(dataDir, spec)
	if err := os.RemoveAll(out); err != nil {
		return "", err
	}
	if err := os.MkdirAll(out, 0o755); err != nil {
		return "", err
	}
	bins, _ := filepath.Glob(filepath.Join(buildDir, "bin", "*"))
	for _, src := range bins {
		st, err := os.Stat(src)
		if err != nil || st.IsDir() {
			continue
		}
		if err := copyFile(src, filepath.Join(out, filepath.Base(src))); err != nil {
			return "", err
		}
	}
	rec := buildRecord{Variant: spec.Variant, Commit: LlamaCppCommit, CMakeArgs: spec.CMakeArgs, BuiltAt: time.Now().UTC().Format(time.RFC3339)}
	b, _ := json.Marshal(rec)
	if err := os.WriteFile(filepath.Join(out, "knirv-build.json"), b, 0o644); err != nil {
		return "", err
	}
	logf("[vlm] llama-server built into %s", out)
	return out, nil
}

// withEnv sets key=value in env, replacing every existing entry for key. Some runtimes
// keep the first duplicate rather than the last, so duplicates are removed explicitly.
func withEnv(env []string, key, value string) []string {
	out := make([]string, 0, len(env)+1)
	for _, kv := range env {
		if k, _, ok := strings.Cut(kv, "="); ok && k == key {
			continue
		}
		out = append(out, kv)
	}
	return append(out, key+"="+value)
}

// requireTools checks what a build needs before anything is downloaded.
func requireTools(spec BuildSpec) error {
	for _, t := range []string{"git", "cmake", "c++"} {
		if _, err := exec.LookPath(t); err != nil {
			return fmt.Errorf("building llama-server needs %s on PATH", t)
		}
	}
	if spec.CUDA {
		if _, err := os.Stat(filepath.Join(spec.CUDAHome, "bin", "nvcc")); err != nil {
			return fmt.Errorf("CUDA build needs nvcc under %s", spec.CUDAHome)
		}
	}
	return nil
}

// ensureSource clones the pinned llama.cpp commit into srcDir, or does nothing when it is
// already there at that commit.
func ensureSource(ctx context.Context, srcDir string, logf func(string, ...any)) error {
	if out, err := exec.Command("git", "-C", srcDir, "rev-parse", "HEAD").Output(); err == nil &&
		strings.TrimSpace(string(out)) == LlamaCppCommit {
		return nil
	}
	if err := os.MkdirAll(srcDir, 0o755); err != nil {
		return err
	}
	logf("[vlm] fetching llama.cpp %s", LlamaCppCommit[:12])
	steps := [][]string{
		{"-C", srcDir, "init", "-q"},
		{"-C", srcDir, "remote", "add", "origin", LlamaCppRepo},
		{"-C", srcDir, "fetch", "--depth", "1", "origin", LlamaCppCommit},
		{"-C", srcDir, "checkout", "-q", "FETCH_HEAD"},
	}
	for _, args := range steps {
		if err := runCmd(ctx, os.Environ(), nil, "git", args...); err != nil {
			// A remote already added on an earlier attempt is not an error.
			if args[2] == "remote" {
				continue
			}
			return fmt.Errorf("git %s: %w", strings.Join(args[2:], " "), err)
		}
	}
	return nil
}

// runCmd runs a command and streams its output to logf (when set) and to the log file.
func runCmd(ctx context.Context, env []string, logf func(string, ...any), name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Env = env
	var w io.Writer = io.Discard
	if logf != nil {
		w = lineWriter{logf}
	}
	cmd.Stdout, cmd.Stderr = w, w
	return cmd.Run()
}

type lineWriter struct{ logf func(string, ...any) }

func (l lineWriter) Write(p []byte) (int, error) {
	for _, line := range strings.Split(strings.TrimRight(string(p), "\n"), "\n") {
		if strings.TrimSpace(line) != "" {
			l.logf("%s", line)
		}
	}
	return len(p), nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	st, err := in.Stat()
	if err != nil {
		return err
	}
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, st.Mode().Perm()|0o600)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}
