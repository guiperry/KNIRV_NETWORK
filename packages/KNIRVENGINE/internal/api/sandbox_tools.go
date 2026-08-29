package api

import (
	"KNIRVENGINE/desktop-client/internal/utils"
	"os"
	"path/filepath"
)

// sandboxToolsDir returns the managed sandbox helper directory. The Linux
// installer initializes it under the engine App Data root so tools are never
// persisted beside an executable or in the caller's working directory.
//
// $KNIRVENGINE_SANDBOX_TOOLS_DIR remains an explicit test/packaging override.
func sandboxToolsDir() string {
	if dir := os.Getenv("KNIRVENGINE_SANDBOX_TOOLS_DIR"); dir != "" {
		return dir
	}
	dir, err := utils.GetToolsDir()
	if err != nil {
		return ""
	}
	return dir
}

// resolveSandboxTool returns the absolute path to a bundled copy of name if one
// exists in the sandbox tools dir (and is executable), otherwise the bare name
// so that exec.Command falls back to PATH resolution.
func resolveSandboxTool(name string) string {
	for _, candidate := range []string{
		filepath.Join(sandboxToolsDir(), name),
		filepath.Join(sandboxToolsDir(), "pyenv", "bin", name),
		filepath.Join(sandboxToolsDir(), "dotnettools", name),
	} {
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() && info.Mode()&0111 != 0 {
			// The dotnet launcher is not self-contained: it resolves host/fxr,
			// shared frameworks, and SDKs relative to its own directory. Copying
			// only /usr/bin/dotnet into tools/ creates a launcher that always
			// fails with "host/fxr does not exist". Use the host installation
			// unless a complete bundle is explicitly present.
			if name == "dotnet" && !bundledDotnetRuntimeExists(filepath.Dir(candidate)) {
				continue
			}
			return candidate
		}
	}
	return name
}

func bundledDotnetRuntimeExists(bundleDir string) bool {
	info, err := os.Stat(filepath.Join(bundleDir, "host", "fxr"))
	return err == nil && info.IsDir()
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
	return filepath.Dir(name) == cleanDir ||
		filepath.Dir(name) == filepath.Join(cleanDir, "pyenv", "bin") ||
		filepath.Dir(name) == filepath.Join(cleanDir, "dotnettools")
}

// isNativeBundledTool reports whether name is a native executable copied to
// the root of the tools bundle. Managed Python and .NET entry points are also
// bundled tools, but they own their runtime-library resolution and must not
// inherit the bundle-wide LD_LIBRARY_PATH.
func isNativeBundledTool(name string) bool {
	if !isBundledTool(name) {
		return false
	}
	cleanDir, err := filepath.Abs(sandboxToolsDir())
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
