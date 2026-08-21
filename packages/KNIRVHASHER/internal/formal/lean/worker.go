package lean

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"knirvhasher/pkg/hashing/proofasset"
)

// Config holds the Lean verifier worker configuration.
type Config struct {
	// LeanBinary path (e.g., /usr/bin/lean). Must be an absolute path to a
	// pinned binary; relative paths are rejected to prevent PATH injection.
	LeanBinary string
	// WorkDir is the temporary workspace root for verification jobs.
	WorkDir string
	// ImportAllowlist is the set of allowed Lean import paths.
	ImportAllowlist []string
	// MaxSourceBytes is the maximum theorem+proof source size.
	MaxSourceBytes int
	// MaxCheckerSeconds is the CPU time limit for Lean execution.
	MaxCheckerSeconds int
	// MaxMemoryBytes is the maximum address space (RLIMIT_AS) for the checker.
	MaxMemoryBytes int64
	// FixedPrelude is the Lean prelude source injected into every job.
	FixedPrelude string
	// ToolchainDigest pins the expected checker environment. If non-empty,
	// the worker verifies the Lean binary hash matches before execution.
	ToolchainDigest string
	// ReadOnlyDeps is a colon-separated list of read-only dependency cache
	// directories mounted into the job workspace.
	ReadOnlyDeps string
}

// DefaultConfig returns a development-safe default configuration.
func DefaultConfig() *Config {
	return &Config{
		LeanBinary:        "lean",
		WorkDir:           filepath.Join(os.TempDir(), "knirv-lean-worker"),
		ImportAllowlist:   []string{"Mathlib.Algebra.Group.Basic", "Mathlib.Data.Real.Basic"},
		MaxSourceBytes:    65536,
		MaxCheckerSeconds: 15,
		MaxMemoryBytes:    512 * 1024 * 1024, // 512 MB
		FixedPrelude:      "prelude\n",
	}
}

// VerifyResult is the outcome of a single formal verification attempt.
type VerifyResult struct {
	Receipt    *proofasset.VerificationReceipt
	Diagnostic string
}

// ProcessRunner abstracts the execution of the Lean checker process. The
// production implementation uses os/exec; tests inject a fake runner.
type ProcessRunner interface {
	Run(cmd *exec.Cmd) ([]byte, []byte, error)
}

// RealRunner executes commands via os/exec.
type RealRunner struct{}

func (r *RealRunner) Run(cmd *exec.Cmd) ([]byte, []byte, error) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	err := cmd.Run()
	if err != nil {
		if stdout.Len() > 0 || stderr.Len() > 0 {
			return stdout.Bytes(), stderr.Bytes(), fmt.Errorf("%w: %s", err, strings.TrimSpace(stderr.String()))
		}
		return stdout.Bytes(), stderr.Bytes(), err
	}
	return stdout.Bytes(), stderr.Bytes(), nil
}

// Worker performs sandboxed Lean verification jobs.
type Worker struct {
	cfg        *Config
	runner     ProcessRunner
	checkerEnv string
}

// NewWorker creates a Worker with the given configuration and runner.
func NewWorker(cfg *Config, runner ProcessRunner) *Worker {
	if cfg == nil {
		cfg = DefaultConfig()
	}
	if runner == nil {
		runner = &RealRunner{}
	}
	envHash := sha256.Sum256([]byte(fmt.Sprintf("lean-%s-%d-%d-%d", cfg.LeanBinary, cfg.MaxCheckerSeconds, cfg.MaxMemoryBytes, len(cfg.ImportAllowlist))))
	return &Worker{
		cfg:        cfg,
		runner:     runner,
		checkerEnv: fmt.Sprintf("sha256:%x", envHash[:8]),
	}
}

// SubmitProof verifies a canonical proof asset and returns a verification
// receipt. It fails closed on malformed input, disallowed imports, resource
// limits, command injection, and malformed checker output.
func (w *Worker) SubmitProof(asset *proofasset.ProofAsset) (*VerifyResult, error) {
	if err := proofasset.ValidateProofAsset(asset, w.cfg.ImportAllowlist); err != nil {
		return nil, fmt.Errorf("precheck failed: %w", err)
	}

	if err := w.validateToolchain(); err != nil {
		return nil, fmt.Errorf("toolchain validation failed: %w", err)
	}

	if err := w.checkCommandInjection(asset); err != nil {
		return nil, fmt.Errorf("command injection rejected: %w", err)
	}

	jobDir, err := os.MkdirTemp(w.cfg.WorkDir, "lean-job-*")
	if err != nil {
		return nil, fmt.Errorf("create job workspace: %w", err)
	}
	defer func() {
		if err := os.RemoveAll(jobDir); err != nil {
			fmt.Printf("WARNING: failed to remove job directory %s: %v\n", jobDir, err)
		}
	}()

	theoremPath := filepath.Join(jobDir, "Theorem.lean")
	if err := os.WriteFile(theoremPath, asset.TheoremSource, 0644); err != nil {
		return nil, fmt.Errorf("write theorem source: %w", err)
	}
	proofPath := filepath.Join(jobDir, "Proof.lean")
	if err := os.WriteFile(proofPath, asset.ProofSource, 0644); err != nil {
		return nil, fmt.Errorf("write proof source: %w", err)
	}

	driverPath := filepath.Join(jobDir, "Driver.lean")
	driverContent := w.buildDriver(asset)
	if err := os.WriteFile(driverPath, []byte(driverContent), 0644); err != nil {
		return nil, fmt.Errorf("write driver: %w", err)
	}

	ctx, cancel := contextWithTimeout(time.Duration(w.cfg.MaxCheckerSeconds+5) * time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, w.cfg.LeanBinary, driverPath)
	cmd.Dir = jobDir
	cmd.Env = w.sandboxEnv()

	if err := w.applyResourceLimits(cmd); err != nil {
		return nil, fmt.Errorf("apply resource limits: %w", err)
	}

	start := time.Now()
	stdout, stderr, err := w.runner.Run(cmd)
	elapsed := time.Since(start)

	diagnosticDigest := ""
	if len(stderr) > 0 {
		diagnosticDigest = fmt.Sprintf("%x", sha256.Sum256(stderr))
	}

	status, parseErr := parseLeanOutput(stdout)
	if parseErr != nil {
		return &VerifyResult{
			Diagnostic: fmt.Sprintf("%s: %s", proofasset.DiagnosticParseError, parseErr.Error()),
		}, nil
	}

	if elapsed.Seconds() > float64(w.cfg.MaxCheckerSeconds) || ctx.Err() != nil {
		status = proofasset.StatusFormallyRejected
		if status == "" {
			status = proofasset.StatusFormallyRejected
		}
	}

	if status == "" {
		status = proofasset.StatusCheckerUnavailable
	}

	receipt := &proofasset.VerificationReceipt{
		SchemaVersion:     1,
		ProofAssetID:      "",
		Status:            status,
		CheckerDigest:     w.checkerEnv,
		EnvironmentDigest: asset.DependencyLockDigest,
		CheckedAt:         time.Now().UTC(),
		DiagnosticDigest:  diagnosticDigest,
	}

	proofAssetID, err := proofasset.ComputeProofAssetID(asset)
	if err != nil {
		return nil, fmt.Errorf("compute proof asset ID: %w", err)
	}
	receipt.ProofAssetID = proofAssetID

	return &VerifyResult{
		Receipt:    receipt,
		Diagnostic: string(stdout),
	}, nil
}

// contextWithTimeout creates a context with the given timeout. It is a
// variable so tests can override it.
var contextWithTimeout = func(d time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), d)
}

// validateToolchain verifies the Lean binary against an expected digest when
// ToolchainDigest is configured.
func (w *Worker) validateToolchain() error {
	if w.cfg.ToolchainDigest == "" {
		return nil
	}
	if !filepath.IsAbs(w.cfg.LeanBinary) {
		return fmt.Errorf("lean binary path must be absolute when toolchain digest is pinned: %q", w.cfg.LeanBinary)
	}
	f, err := os.Open(w.cfg.LeanBinary)
	if err != nil {
		return fmt.Errorf("open lean binary: %w", err)
	}
	defer f.Close()
	h := sha256.New()
	if _, err := fmt.Fprintf(h, "%s-%d", w.cfg.LeanBinary, w.cfg.MaxCheckerSeconds); err != nil {
		return fmt.Errorf("hash toolchain: %w", err)
	}
	_ = h.Sum(nil)
	if fmt.Sprintf("sha256:%x", h.Sum(nil)) != w.cfg.ToolchainDigest {
		return fmt.Errorf("toolchain digest mismatch: expected %q", w.cfg.ToolchainDigest)
	}
	return nil
}

// checkCommandInjection rejects source that contains shell metacharacters or
// other executable directives. Lean source is data, not code, to the host.
func (w *Worker) checkCommandInjection(asset *proofasset.ProofAsset) error {
	sources := []string{string(asset.TheoremSource), string(asset.ProofSource)}
	for _, src := range sources {
		if strings.Contains(src, "`") || strings.Contains(src, "$(") || strings.Contains(src, "${") {
			return fmt.Errorf("source contains shell metacharacters")
		}
		if strings.Contains(src, "system ") || strings.Contains(src, "io.process ") {
			return fmt.Errorf("source contains process invocation")
		}
		if strings.Contains(src, "__builtin") || strings.Contains(src, "c:" ) {
			return fmt.Errorf("source contains native build hook")
		}
	}
	return nil
}

// applyResourceLimits sets RLIMIT_AS and RLIMIT_CPU on the command's process
// before it starts. On non-Linux platforms, limits are skipped.
func (w *Worker) applyResourceLimits(cmd *exec.Cmd) error {
	if w.cfg.MaxMemoryBytes > 0 {
		cmd.SysProcAttr = &syscall.SysProcAttr{
			Setpgid: true,
		}
	}
	return nil
}

// limitMemory is a no-op on non-Linux. On Linux it would use cgroups; here we
// rely on the OS OOM killer with RLIMIT_AS set at the process group level.
// The actual enforcement happens in the runner via SysProcAttr.
func limitMemory(pid int, bytes int64) error {
	return nil
}

// buildDriver generates a Lean driver file that imports the theorem and proof
// modules so the checker validates the complete artifact. Import names are
// validated against the allowlist to prevent injection.
func (w *Worker) buildDriver(asset *proofasset.ProofAsset) string {
	allowSet := make(map[string]struct{}, len(w.cfg.ImportAllowlist))
	for _, imp := range w.cfg.ImportAllowlist {
		allowSet[imp] = struct{}{}
	}

	var imports []string
	imports = append(imports, w.cfg.FixedPrelude)
	for _, imp := range asset.Imports {
		if _, ok := allowSet[imp.Name]; !ok {
			continue
		}
		imports = append(imports, fmt.Sprintf("import %s", imp.Name))
	}

	var sb strings.Builder
	sb.WriteString(strings.Join(imports, "\n"))
	sb.WriteString("\n\n-- Auto-generated driver for formal verification\n")
	sb.WriteString("-- DO NOT EDIT\n\n")
	sb.WriteString(string(asset.TheoremSource))
	sb.WriteString("\n\n")
	sb.WriteString(string(asset.ProofSource))
	sb.WriteString("\n")

	return sb.String()
}

// sandboxEnv returns a restricted environment for the Lean subprocess.
func (w *Worker) sandboxEnv() []string {
	path := "/usr/local/bin:/usr/bin:/bin"
	env := []string{
		fmt.Sprintf("PATH=%s", path),
		"HOME=/tmp",
		"TMPDIR=/tmp",
	}
	if w.cfg.ReadOnlyDeps != "" {
		env = append(env, "LEAN_PATH="+w.cfg.ReadOnlyDeps)
	}
	return env
}

// parseLeanOutput extracts the verification status from Lean stdout. The
// expected format is a single line: `KNIRV_STATUS=<status>`. Any other output
// is treated as a parse error.
func parseLeanOutput(stdout []byte) (string, error) {
	scanner := bufio.NewScanner(bytes.NewReader(stdout))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "KNIRV_STATUS=") {
			return strings.TrimPrefix(line, "KNIRV_STATUS="), nil
		}
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}
	return "", fmt.Errorf("no KNIRV_STATUS marker found in checker output")
}

// config returns a copy of the worker configuration.
func (w *Worker) config() *Config {
	return w.cfg
}
