// Package install discovers or provisions the llama.cpp server and a default model.
package install

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/guiperry/knirv/llama/internal/config"
)

const (
	DefaultServerURL = "https://releases.knirv.com/knirv/llama/linux-amd64/llama-server.gz"
	// LatestReleaseURL reports the newest upstream llama.cpp release; its tag is
	// the build number ("b6789") that `llama-server --version` prints.
	LatestReleaseURL = "https://api.github.com/repos/ggml-org/llama.cpp/releases/latest"
	upstreamRepoURL  = "https://github.com/ggml-org/llama.cpp.git"
)

type Options struct {
	DataDir, ServerPath, ServerURL, ModelPath, ModelURL, ModelName string
	NoInstall                                                      bool
	// CheckUpdate compares the resolved llama-server against the latest upstream
	// release and replaces it when older. A failed check or update keeps the
	// working server, so an offline host still starts.
	CheckUpdate bool
}

type Result struct{ ServerPath, ModelPath, ModelName string }

type Installer struct {
	Run func(context.Context, string, ...string) error
	// Output runs a command and returns its combined stdout and stderr.
	Output   func(context.Context, string, ...string) (string, error)
	Get      func(string) (*http.Response, error)
	LookPath func(string) (string, error)
	// Prog, when non-nil, receives provisioning progress (download bars and
	// stage lines). Leave nil for silent provisioning, as the tests do.
	Prog *progress
}

func (i *Installer) startProgress(title string, statuses ...string) {
	if i.Prog != nil {
		i.Prog.start(title, 0, statuses...)
	}
}

func (i *Installer) stopProgress(summary string) {
	if i.Prog != nil {
		i.Prog.stop(summary)
	}
}

func (i *Installer) logProgress(format string, args ...any) {
	if i.Prog != nil {
		i.Prog.logf(format, args...)
		return
	}
	fmt.Printf(format, args...)
	if !strings.HasSuffix(format, "\n") {
		fmt.Println()
	}
}

func New() *Installer {
	return &Installer{
		Run: func(ctx context.Context, name string, args ...string) error {
			// Provisioning is intentionally visible to the parent process. In
			// particular, CMake's diagnostic output is the only useful detail
			// when a platform cannot build llama.cpp.
			cmd := exec.CommandContext(ctx, name, args...)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			return cmd.Run()
		},
		Output: func(ctx context.Context, name string, args ...string) (string, error) {
			out, err := exec.CommandContext(ctx, name, args...).CombinedOutput()
			return string(out), err
		},
		Get:      http.Get,
		LookPath: exec.LookPath,
	}
}

var buildNumber = regexp.MustCompile(`(?m)^version:\s*(\d+)`)

// installedBuild returns the llama.cpp build number of a llama-server binary.
func (i *Installer) installedBuild(ctx context.Context, server string) (int, error) {
	out, err := i.Output(ctx, server, "--version")
	m := buildNumber.FindStringSubmatch(out)
	if m == nil {
		if err != nil {
			return 0, fmt.Errorf("%s --version: %w", server, err)
		}
		return 0, fmt.Errorf("no build number in %s --version output", server)
	}
	return strconv.Atoi(m[1])
}

// latestBuild returns the newest upstream release tag and its build number.
func (i *Installer) latestBuild() (tag string, build int, err error) {
	resp, err := i.Get(LatestReleaseURL)
	if err != nil {
		return "", 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", 0, fmt.Errorf("unexpected HTTP status %s", resp.Status)
	}
	var release struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&release); err != nil {
		return "", 0, err
	}
	build, err = strconv.Atoi(strings.TrimPrefix(release.TagName, "b"))
	if err != nil {
		return "", 0, fmt.Errorf("unrecognised release tag %q", release.TagName)
	}
	return release.TagName, build, nil
}

// updateServer replaces server with the latest upstream build when it is older.
// It returns the path to use, which is server itself whenever nothing changed or
// the update failed; the working binary is never removed before a replacement
// has been verified.
func (i *Installer) updateServer(ctx context.Context, o Options, server string) string {
	tag, latest, err := i.latestBuild()
	if err != nil {
		i.logProgress("[KNIRVLLAMA] Could not check the latest llama.cpp release (%v); keeping %s", err, server)
		return server
	}
	installed, err := i.installedBuild(ctx, server)
	if err != nil {
		i.logProgress("[KNIRVLLAMA] Could not read the installed llama.cpp version (%v); updating to %s", err, tag)
	} else if installed >= latest {
		i.logProgress("[KNIRVLLAMA] llama.cpp b%d is up to date (latest %s)", installed, tag)
		return server
	} else {
		i.logProgress("[KNIRVLLAMA] llama.cpp b%d is older than %s; updating", installed, tag)
	}

	serverURL := o.ServerURL
	if serverURL == "" {
		serverURL = DefaultServerURL
	}
	// Download beside the live binary under another name, so a stale or broken
	// artifact can be discarded without touching the server that works.
	if staged, err := i.downloadServer(ctx, o.DataDir, serverURL, "llama-server.update"); err == nil {
		if got, verr := i.installedBuild(ctx, staged); verr == nil && got >= latest {
			dest := filepath.Join(o.DataDir, "bin", "llama-server")
			if err := os.Rename(staged, dest); err == nil {
				i.logProgress("[KNIRVLLAMA] Updated llama-server to b%d", got)
				return dest
			}
		}
		_ = os.Remove(staged)
		i.logProgress("[KNIRVLLAMA] Prebuilt llama-server is not %s yet; building it from source", tag)
	} else {
		i.stopProgress("")
		fmt.Printf("[KNIRVLLAMA] Prebuilt llama-server unavailable (%v); building %s from source\n", err, tag)
	}
	built, err := i.installServer(ctx, o.DataDir, tag)
	if err != nil {
		fmt.Printf("[KNIRVLLAMA] Update to %s failed (%v); keeping %s\n", tag, err, server)
		return server
	}
	i.logProgress("[KNIRVLLAMA] Built llama-server %s", tag)
	return built
}

func (i *Installer) Ensure(ctx context.Context, o Options) (Result, error) {
	if o.DataDir == "" {
		return Result{}, fmt.Errorf("data directory is required")
	}
	if o.ModelName == "" {
		o.ModelName = config.DefaultModelName
	}
	if o.ModelURL == "" {
		o.ModelURL = config.DefaultModelURL
	}

	serverPaths := append([]string{o.ServerPath, filepath.Join(o.DataDir, "llama.cpp", "build", "bin", "llama-server")}, legacyServerPaths()...)
	server := firstFile(serverPaths...)
	if server == "" {
		if o.NoInstall {
			return Result{}, fmt.Errorf("llama-server was not found and installation is disabled")
		}
		serverURL := o.ServerURL
		if serverURL == "" {
			serverURL = DefaultServerURL
		}
		i.logProgress("[KNIRVLLAMA] Downloading prebuilt llama-server from %s\n", serverURL)
		var downloadErr error
		server, downloadErr = i.downloadServer(ctx, o.DataDir, serverURL, "llama-server")
		if downloadErr != nil {
			// The native build streams large amounts of CMake output straight to
			// the terminal, so retire any in-place bar before it starts.
			i.stopProgress("")
			// A prebuilt CPU binary is portable across the supported Debian/Kali
			// image. Keep a native build as a fallback for incompatible libc/CPU
			// combinations and for deployments that require host-native tuning.
			fmt.Printf("[KNIRVLLAMA] Prebuilt llama-server unavailable (%v); falling back to a native build\n", downloadErr)
			var err error
			server, err = i.installServer(ctx, o.DataDir, "")
			if err != nil {
				return Result{}, fmt.Errorf("provision llama-server: prebuilt download failed: %v; native build failed: %w", downloadErr, err)
			}
		}
	} else if o.CheckUpdate && !o.NoInstall {
		server = i.updateServer(ctx, o, server)
	}
	modelPaths := []string{o.ModelPath, filepath.Join(o.DataDir, "models", o.ModelName)}
	// Legacy installations only contain the bundled TinyLlama model. Do not let
	// one override an explicitly selected model name or URL.
	if o.ModelName == config.DefaultModelName && o.ModelURL == config.DefaultModelURL {
		modelPaths = append(modelPaths, legacyModelPaths()...)
	}
	model := firstFile(modelPaths...)
	if model == "" {
		if o.NoInstall {
			return Result{}, fmt.Errorf("model was not found and installation is disabled")
		}
		var err error
		model, err = i.downloadModel(o.DataDir, o.ModelName, o.ModelURL)
		if err != nil {
			return Result{}, err
		}
	}
	return Result{ServerPath: server, ModelPath: model, ModelName: strings.TrimSuffix(filepath.Base(model), filepath.Ext(model))}, nil
}

// installServer builds llama-server from source. A non-empty tag builds that
// upstream release (fetching it into an existing checkout); empty builds the
// default branch head on a fresh clone.
func (i *Installer) installServer(ctx context.Context, dataDir, tag string) (string, error) {
	dir := filepath.Join(dataDir, "llama.cpp")
	if err := i.ensureCommands(ctx, map[string]string{"git": "git"}); err != nil {
		return "", err
	}
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		fmt.Println("[KNIRVLLAMA] Cloning llama.cpp for native build")
		args := []string{"clone", "--depth", "1"}
		if tag != "" {
			args = append(args, "--branch", tag)
		}
		if err := i.Run(ctx, "git", append(args, upstreamRepoURL, dir)...); err != nil {
			return "", fmt.Errorf("clone llama.cpp: %w", err)
		}
	} else if tag != "" {
		fmt.Printf("[KNIRVLLAMA] Fetching llama.cpp %s\n", tag)
		if err := i.Run(ctx, "git", "-C", dir, "fetch", "--depth", "1", "origin", "tag", tag, "--no-tags"); err != nil {
			return "", fmt.Errorf("fetch llama.cpp %s: %w", tag, err)
		}
		if err := i.Run(ctx, "git", "-C", dir, "checkout", "--force", "FETCH_HEAD"); err != nil {
			return "", fmt.Errorf("checkout llama.cpp %s: %w", tag, err)
		}
	}
	if err := i.ensureCommands(ctx, map[string]string{"cmake": "cmake", "cc": "build-essential", "c++": "build-essential"}); err != nil {
		return "", err
	}
	build := filepath.Join(dir, "build")
	fmt.Println("[KNIRVLLAMA] Configuring native llama.cpp fallback build")
	if err := i.Run(ctx, "cmake", "-S", dir, "-B", build, "-DLLAMA_BUILD_SERVER=ON"); err != nil {
		return "", fmt.Errorf("configure llama.cpp: %w", err)
	}
	if err := i.Run(ctx, "cmake", "--build", build, "--target", "llama-server", "--config", "Release"); err != nil {
		return "", fmt.Errorf("build llama.cpp server: %w", err)
	}
	path := firstFile(filepath.Join(build, "bin", "llama-server"), filepath.Join(build, "bin", "Release", "llama-server"))
	if path == "" {
		return "", fmt.Errorf("llama.cpp build completed but llama-server was not produced")
	}
	return path, nil
}

// ensureCommands installs only missing native-build dependencies. It is called
// solely after both a cached server and the release artifact are unavailable.
func (i *Installer) ensureCommands(ctx context.Context, packages map[string]string) error {
	missing := make(map[string]struct{})
	for command, pkg := range packages {
		if _, err := i.LookPath(command); err != nil {
			missing[pkg] = struct{}{}
		}
	}
	if len(missing) == 0 {
		return nil
	}
	if _, err := i.LookPath("apt-get"); err != nil {
		return fmt.Errorf("missing native-build dependencies %v and apt-get is unavailable", missing)
	}
	args := make([]string, 0, len(missing)+3)
	for pkg := range missing {
		args = append(args, pkg)
	}
	sort.Strings(args)
	fmt.Printf("[KNIRVLLAMA] Installing missing native-build packages: %s\n", strings.Join(args, ", "))
	if err := i.Run(ctx, "apt-get", "update"); err != nil {
		return fmt.Errorf("apt-get update: %w", err)
	}
	installArgs := append([]string{"install", "--yes", "--no-install-recommends"}, args...)
	if err := i.Run(ctx, "apt-get", installArgs...); err != nil {
		return fmt.Errorf("apt-get install %s: %w", strings.Join(args, ", "), err)
	}
	return nil
}

func (i *Installer) downloadServer(ctx context.Context, dataDir, url, name string) (string, error) {
	i.startProgress("Provisioning KNIRVLLAMA", "downloading prebuilt llama-server...", "decompressing release archive...", "verifying executable...")
	dir := filepath.Join(dataDir, "bin")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("create server directory: %w", err)
	}
	resp, err := i.Get(url)
	if err != nil {
		i.stopProgress("")
		return "", fmt.Errorf("download prebuilt llama-server: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		i.stopProgress("")
		return "", fmt.Errorf("download prebuilt llama-server: unexpected HTTP status %s", resp.Status)
	}
	cr := &countingReader{r: resp.Body, prog: i.Prog}
	reader := io.Reader(cr)
	if strings.HasSuffix(strings.ToLower(url), ".gz") {
		gz, err := gzip.NewReader(resp.Body)
		if err != nil {
			i.stopProgress("")
			return "", fmt.Errorf("read prebuilt llama-server archive: %w", err)
		}
		defer gz.Close()
		reader = gz
	}
	path := filepath.Join(dir, name)
	tmp := path + ".part"
	f, err := os.OpenFile(tmp, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0755)
	if err != nil {
		i.stopProgress("")
		return "", err
	}
	_, copyErr := io.Copy(f, reader)
	closeErr := f.Close()
	if copyErr != nil {
		i.stopProgress("")
		return "", fmt.Errorf("write prebuilt llama-server: %w", copyErr)
	}
	if closeErr != nil {
		i.stopProgress("")
		return "", closeErr
	}
	if err := os.Rename(tmp, path); err != nil {
		i.stopProgress("")
		return "", fmt.Errorf("finalize prebuilt llama-server: %w", err)
	}
	// Verify the artifact before caching it permanently. This catches a release
	// built against an incompatible glibc or CPU feature set and lets Ensure
	// continue into the native-build fallback on the same first run.
	i.stopProgress(fmt.Sprintf("[KNIRVLLAMA] Installed prebuilt llama-server (%s)", fmtBytes(cr.total)))
	if err := i.Run(ctx, path, "--version"); err != nil {
		_ = os.Remove(path)
		return "", fmt.Errorf("verify prebuilt llama-server: %w", err)
	}
	return path, nil
}

func (i *Installer) downloadModel(dataDir, name, url string) (string, error) {
	i.startProgress("Provisioning model weights", "downloading model weights...", "verifying download...")
	dir := filepath.Join(dataDir, "models")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("create models directory: %w", err)
	}
	path := filepath.Join(dir, name)
	tmp := path + ".part"
	resp, err := i.Get(url)
	if err != nil {
		i.stopProgress("")
		return "", fmt.Errorf("download model: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		i.stopProgress("")
		return "", fmt.Errorf("download model: unexpected HTTP status %s", resp.Status)
	}
	f, err := os.OpenFile(tmp, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0600)
	if err != nil {
		i.stopProgress("")
		return "", err
	}
	cr := &countingReader{r: resp.Body, prog: i.Prog}
	_, copyErr := io.Copy(f, cr)
	closeErr := f.Close()
	if copyErr != nil {
		i.stopProgress("")
		return "", fmt.Errorf("write model: %w", copyErr)
	}
	if closeErr != nil {
		i.stopProgress("")
		return "", closeErr
	}
	if err := os.Rename(tmp, path); err != nil {
		i.stopProgress("")
		return "", fmt.Errorf("finalize model download: %w", err)
	}
	i.stopProgress(fmt.Sprintf("[KNIRVLLAMA] Model ready (%s)", fmtBytes(cr.total)))
	return path, nil
}

func firstFile(paths ...string) string {
	for _, path := range paths {
		if path == "" {
			continue
		}
		if info, err := os.Stat(path); err == nil && !info.IsDir() {
			return path
		}
	}
	return ""
}
func legacyServerPaths() []string {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}
	return []string{filepath.Join(home, "llama.cpp", "build", "bin", "llama-server"), filepath.Join(home, "models", "llama.cpp", "build", "bin", "llama-server")}
}
func legacyModelPaths() []string {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}
	return []string{filepath.Join(home, "models", "tinyllama-1.1b-chat-v1.0.Q4_0.gguf"), filepath.Join(home, "tinyllama-1.1b-chat-v1.0.Q4_0.gguf")}
}
