package api

import (
	"os"
	"path/filepath"
)

// sandboxToolsDir returns the directory that may contain bundled sandbox
// helper binaries (bwrap, Xvfb, x11vnc) shipped with the engine dist. The
// engine prefers these over whatever is on PATH so the end user never has to
// install the packages separately.
//
// Resolution order (mirrors how the Electron host is located in main.go):
//  1. $KNIRVENGINE_SANDBOX_TOOLS_DIR (explicit override, e.g. for tests/packaging)
//  2. <exeDir>/tools
//  3. <exeDir>/../tools
//  4. <cwd>/tools
//
// If none of those exist, the engine-side default (<exeDir>/tools) is returned
// so callers can still build candidate paths; resolveSandboxTool only treats a
// tool as bundled when the file actually exists there.
func sandboxToolsDir() string {
	if dir := os.Getenv("KNIRVENGINE_SANDBOX_TOOLS_DIR"); dir != "" {
		return dir
	}

	candidates := []string{}
	if exe, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exe)
		candidates = append(candidates,
			filepath.Join(exeDir, "tools"),
			filepath.Join(exeDir, "..", "tools"),
		)
	}
	if wd, err := os.Getwd(); err == nil {
		candidates = append(candidates, filepath.Join(wd, "tools"))
	}

	for _, dir := range candidates {
		if info, err := os.Stat(dir); err == nil && info.IsDir() {
			return dir
		}
	}

	if exe, err := os.Executable(); err == nil {
		return filepath.Join(filepath.Dir(exe), "tools")
	}
	return "tools"
}

// resolveSandboxTool returns the absolute path to a bundled copy of name if one
// exists in the sandbox tools dir (and is executable), otherwise the bare name
// so that exec.Command falls back to PATH resolution.
func resolveSandboxTool(name string) string {
	candidate := filepath.Join(sandboxToolsDir(), name)
	if info, err := os.Stat(candidate); err == nil && !info.IsDir() && info.Mode()&0111 != 0 {
		return candidate
	}
	return name
}

// isBundledTool reports whether name refers to a binary physically located in
// the bundled tools directory (as opposed to a bare name resolved via PATH).
func isBundledTool(name string) bool {
	if !filepath.IsAbs(name) {
		return false
	}
	dir := sandboxToolsDir()
	if _, err := os.Stat(dir); err != nil {
		return false
	}
	cleanDir, err := filepath.Abs(dir)
	if err != nil {
		return false
	}
	return filepath.Dir(name) == cleanDir
}

// bundledToolsLibDir returns the lib/ subdirectory of the bundled tools dir,
// where the transitive shared-library dependencies copied by
// scripts/bundle-sandbox-tools.sh live. It is empty when no such directory
// exists.
func bundledToolsLibDir() string {
	dir := filepath.Join(sandboxToolsDir(), "lib")
	if _, err := os.Stat(dir); err != nil {
		return ""
	}
	return dir
}
