package dve_workspace

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
	"github.com/tetratelabs/wazero/sys"
)

// SkillExecConfig configures a single Wazero skill execution.
type SkillExecConfig struct {
	WASMBytes     []byte        // compiled skill WASM module
	WorkspaceRoot string        // DVE merged/ directory (WASI FS root)
	Args          []string      // argv[1:]
	Env           []string      // key=value pairs
	Timeout       time.Duration // max wall-clock time
	MaxMemoryMB   uint32        // WASM linear memory limit in MB
}

// SkillExecResult contains the output of a Wazero skill execution.
type SkillExecResult struct {
	ExitCode uint32
	Stdout   string
	Stderr   string
	Elapsed  time.Duration
}

// ExecuteSkillWASM runs a skill WASM module in a Wazero sandbox.
// The module's filesystem is confined to workspaceRoot.
// Host network and clock are available only if cfg allows them.
func ExecuteSkillWASM(ctx context.Context, cfg SkillExecConfig) (SkillExecResult, error) {
	var stdout, stderr strings.Builder
	start := time.Now()

	if cfg.WorkspaceRoot == "" {
		cfg.WorkspaceRoot = "/tmp"
	}

	ctx, cancel := context.WithTimeout(ctx, cfg.Timeout)
	defer cancel()

	// Calculate memory pages (64KB per page)
	memoryPages := uint32(0)
	if cfg.MaxMemoryMB > 0 {
		memoryPages = (cfg.MaxMemoryMB * 1024 * 1024) / 65536
	}

	rtCfg := wazero.NewRuntimeConfig().
		WithCloseOnContextDone(true)

	if memoryPages > 0 {
		rtCfg = rtCfg.WithMemoryLimitPages(memoryPages)
	}

	rt := wazero.NewRuntimeWithConfig(ctx, rtCfg)
	defer rt.Close(ctx)

	// Instantiate WASI
	wasi_snapshot_preview1.MustInstantiate(ctx, rt)

	// Configure filesystem: mount the workspace root as /
	// with /bin mounted read-only to protect BusyBox applets
	fsCfg := wazero.NewFSConfig().
		WithDirMount(cfg.WorkspaceRoot, "/").
		WithReadOnlyDirMount(cfg.WorkspaceRoot+"/bin", "/bin")

	// Module configuration
	modCfg := wazero.NewModuleConfig().
		WithStdout(&stdout).
		WithStderr(&stderr).
		WithFSConfig(fsCfg).
		WithArgs(append([]string{"skill"}, cfg.Args...)...).
		WithEnv("KNIRV_DVE", "1").
		WithSysNanosleep().
		WithSysWalltime()

	for _, e := range cfg.Env {
		kv := strings.SplitN(e, "=", 2)
		if len(kv) == 2 {
			modCfg = modCfg.WithEnv(kv[0], kv[1])
		}
	}

	// Compile the WASM module
	compiled, err := rt.CompileModule(ctx, cfg.WASMBytes)
	if err != nil {
		return SkillExecResult{}, fmt.Errorf("compile wasm: %w", err)
	}

	// Instantiate and run
	_, err = rt.InstantiateModule(ctx, compiled, modCfg)
	exitCode := uint32(0)
	if err != nil {
		if exitErr, ok := err.(*sys.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			return SkillExecResult{}, fmt.Errorf("run wasm: %w", err)
		}
	}

	return SkillExecResult{
		ExitCode: exitCode,
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
		Elapsed:  time.Since(start),
	}, nil
}
