package harness

import (
	"archive/tar"
	"bufio"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

// DefaultColibriVersion is the release the harness installs. Colibri publishes no
// checksum for its source archive, so a moving branch is never used.
const DefaultColibriVersion = "v2.0.0"

// DefaultColibriPort is where the search for a free Colibri port starts.
const DefaultColibriPort = 8099

// freePort returns preferred when nothing listens on it, else the next free port.
func freePort(host string, preferred int) int {
	for p := preferred; p < preferred+200; p++ {
		ln, err := net.Listen("tcp", net.JoinHostPort(host, strconv.Itoa(p)))
		if err == nil {
			ln.Close()
			return p
		}
	}
	return preferred
}

// stickyPort keeps Colibri on the port it was set up with: setup writes the port into
// Colibri's configuration, so a port that moves between runs would redo the setup. It
// falls back to the first free port from DefaultColibriPort when the old one is taken.
func stickyPort(home, host string) int {
	if home != "" {
		if b, err := os.ReadFile(filepath.Join(home, "state", "knirv-install.json")); err == nil {
			var rec installRecord
			if json.Unmarshal(b, &rec) == nil && rec.Port > 0 && freePort(host, rec.Port) == rec.Port {
				return rec.Port
			}
		}
	}
	return freePort(host, DefaultColibriPort)
}

// ErrNoModelFits means Colibri's own catalog has no model this machine can run, so
// installing is pointless until the machine changes.
var ErrNoModelFits = errors.New("colibri: no model fits this machine")

// permanentError marks failures a retry cannot fix (no Python, no model fits).
type permanentError struct{ err error }

func (p permanentError) Error() string { return p.err.Error() }
func (p permanentError) Unwrap() error { return p.err }

// IsPermanent reports whether retrying the failed Colibri step is pointless.
func IsPermanent(err error) bool {
	var p permanentError
	return errors.As(err, &p)
}

// ColibriConfig describes where Colibri lives and how it is run. Everything it
// writes stays under Home.
type ColibriConfig struct {
	// Home holds the source, the downloaded models and Colibri's own state.
	Home string
	// Version is the release tag to install (default DefaultColibriVersion).
	Version string
	// SourceURL overrides the release archive URL (tests, mirrors).
	SourceURL string
	// Model is Colibri's one setup parameter, `--model`. Empty takes the model Colibri
	// itself recommends for this machine, which is what its one-step setup does when
	// the user presses Enter. KNIRV_COLIBRI_MODEL sets it from the environment.
	Model string
	// Host and Port are where Colibri's server listens. The default is 127.0.0.1 and
	// the first free port from 8099 on (8000, 8080, 8082, 8090 and 9090 belong to
	// llama-server and the other KNIRV services).
	Host string
	Port int
	// Python is the interpreter that runs `c/coli` (default python3).
	Python string
	// ReadyTimeout bounds how long Start waits for the model to load (default 30m).
	ReadyTimeout time.Duration
	// Force skips the host requirement and the speed check (KNIRV_COLIBRI=force).
	Force bool
	// MinTokensPerSec is the slowest measured generation speed Colibri is kept at
	// (default DefaultMinTokensPerSec, KNIRV_COLIBRI_MIN_TPS; negative: never measure).
	MinTokensPerSec float64
	// WarmupTimeout bounds the speed check's first request, which loads the expert
	// cache from disk (default 10m).
	WarmupTimeout time.Duration
	// CPUInfoPath is where the CPU's features are read from (default /proc/cpuinfo).
	CPUInfoPath string
	// Env is added to Colibri's environment (HF_TOKEN and the like pass through anyway).
	Env  []string
	HTTP *http.Client
	// Logf receives progress and Colibri's own output. Nil is silent.
	Logf func(format string, args ...any)
}

// Colibri installs and runs https://github.com/JustVugg/colibri, the primary model
// harness. Install and Start are idempotent: both resume where they stopped.
type Colibri struct {
	cfg ColibriConfig

	mu      sync.Mutex
	baseURL string // Colibri's OpenAI base URL, e.g. http://127.0.0.1:8090/v1
	modelID string // the API model id requests must send
	chosen  string // the model Install resolved

	feat featCache
}

// featCache remembers what the served engine accepts, so routing does not ask Colibri
// on every request. Only a positive answer is cached.
type featCache struct {
	base           string
	at             time.Time
	tools, grammar bool
}

// NewColibri fills defaults.
func NewColibri(cfg ColibriConfig) *Colibri {
	if cfg.Version == "" {
		cfg.Version = firstNonEmpty(strings.TrimSpace(os.Getenv("KNIRV_COLIBRI_VERSION")), DefaultColibriVersion)
	}
	if cfg.SourceURL == "" {
		cfg.SourceURL = strings.TrimSpace(os.Getenv("KNIRV_COLIBRI_SOURCE_URL"))
	}
	if cfg.SourceURL == "" {
		cfg.SourceURL = "https://github.com/JustVugg/colibri/archive/refs/tags/" + cfg.Version + ".tar.gz"
	}
	if cfg.Host == "" {
		cfg.Host = "127.0.0.1"
	}
	if cfg.Port == 0 {
		cfg.Port = stickyPort(cfg.Home, cfg.Host)
	}
	if cfg.Python == "" {
		cfg.Python = "python3"
	}
	if cfg.ReadyTimeout <= 0 {
		cfg.ReadyTimeout = 30 * time.Minute
	}
	if cfg.Model == "" {
		cfg.Model = strings.TrimSpace(os.Getenv("KNIRV_COLIBRI_MODEL"))
	}
	if cfg.HTTP == nil {
		cfg.HTTP = &http.Client{Timeout: 5 * time.Second}
	}
	if cfg.CPUInfoPath == "" {
		cfg.CPUInfoPath = "/proc/cpuinfo"
	}
	if strings.EqualFold(strings.TrimSpace(os.Getenv("KNIRV_COLIBRI")), "force") {
		cfg.Force = true
	}
	if cfg.MinTokensPerSec == 0 {
		if v, ok := parseMinTPS(os.Getenv("KNIRV_COLIBRI_MIN_TPS")); ok {
			cfg.MinTokensPerSec = v
		}
	}
	return &Colibri{cfg: cfg}
}

func (c *Colibri) logf(format string, args ...any) {
	if c.cfg.Logf != nil {
		c.cfg.Logf(format, args...)
	}
}

func (c *Colibri) srcDir() string   { return filepath.Join(c.cfg.Home, "src", "colibri-"+c.cfg.Version) }
func (c *Colibri) stateDir() string { return filepath.Join(c.cfg.Home, "state") }
func (c *Colibri) modelsDir() string {
	return filepath.Join(c.cfg.Home, "models")
}
func (c *Colibri) coli() string { return filepath.Join(c.srcDir(), "c", "coli") }

// BaseURL is Colibri's OpenAI-compatible base URL once it has started, else "".
func (c *Colibri) BaseURL() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.baseURL
}

// ModelID is the model id Colibri's API expects in requests, else "".
func (c *Colibri) ModelID() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.modelID
}

// ChosenModel is the `--model` value Install resolved, else "".
func (c *Colibri) ChosenModel() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.chosen
}

// Install makes Colibri ready to start: it fetches the source, sets the one
// parameter (the model), and lets Colibri's own setup build or fetch its engine and
// download the model with resume. A rerun after an interruption continues.
func (c *Colibri) Install(ctx context.Context) error {
	if c.cfg.Home == "" {
		return permanentError{errors.New("colibri: no home directory configured")}
	}
	if err := c.hostRequirement(); err != nil {
		return err
	}
	if _, err := exec.LookPath(c.cfg.Python); err != nil {
		return permanentError{fmt.Errorf("colibri needs Python 3.10 or newer (%q not found): %w", c.cfg.Python, err)}
	}
	if err := c.fetchSource(ctx); err != nil {
		return fmt.Errorf("fetch colibri %s: %w", c.cfg.Version, err)
	}
	model, gb, err := c.chooseModel(ctx)
	if err != nil {
		return err
	}
	c.mu.Lock()
	c.chosen = model
	c.mu.Unlock()
	if err := c.knownTooSlow(model); err != nil {
		return err // measured earlier on this machine: do not download or start it again
	}

	if err := os.MkdirAll(c.modelsDir(), 0o755); err != nil {
		return err
	}
	if have, ok := c.alreadyInstalled(model, gb); ok {
		c.logf("colibri already set up with --model %s (%.1f GB on disk); not downloading again", model, float64(have)/1e9)
		return nil
	}
	c.logf("setting up colibri with --model %s (download and build can take a long while; a rerun resumes)", model)
	_, err = c.run(ctx, true, "setup", "--yes", "--no-start", "--no-browser",
		"--model", model,
		"--dir", c.modelsDir(),
		"--host", c.cfg.Host,
		"--port", strconv.Itoa(c.cfg.Port))
	if err != nil {
		return fmt.Errorf("colibri setup: %w", err)
	}
	if err := c.writeInstallRecord(model); err != nil {
		c.logf("could not record the finished colibri setup (it will be checked again next start): %v", err)
	}
	return nil
}

// installRecord is written after Colibri's setup finishes, so later starts can tell a
// finished install from an interrupted download without fetching anything.
type installRecord struct {
	Version string `json:"version"`
	Model   string `json:"model"`
	Host    string `json:"host"`
	Port    int    `json:"port"`
}

func (c *Colibri) recordPath() string { return filepath.Join(c.stateDir(), "knirv-install.json") }

func (c *Colibri) writeInstallRecord(model string) error {
	if err := os.MkdirAll(c.stateDir(), 0o755); err != nil {
		return err
	}
	b, err := json.Marshal(installRecord{Version: c.cfg.Version, Model: model, Host: c.cfg.Host, Port: c.cfg.Port})
	if err != nil {
		return err
	}
	tmp := c.recordPath() + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, c.recordPath())
}

// alreadyInstalled reports whether setup finished earlier for this exact version,
// model, host and port and the model files are still on disk. Only then is Colibri's
// setup (and its download) skipped. expectedGB is the size Colibri lists for the
// model; 0 means unknown, and any model files at all are accepted. The second result
// is false when the files are missing or short, so setup runs and resumes them.
func (c *Colibri) alreadyInstalled(model string, expectedGB float64) (int64, bool) {
	b, err := os.ReadFile(c.recordPath())
	if err != nil {
		return 0, false
	}
	var rec installRecord
	if json.Unmarshal(b, &rec) != nil ||
		rec.Version != c.cfg.Version || rec.Model != model || rec.Host != c.cfg.Host || rec.Port != c.cfg.Port {
		return 0, false
	}
	var have int64
	filepath.WalkDir(c.modelsDir(), func(_ string, d fs.DirEntry, err error) error {
		if err == nil && d.Type().IsRegular() {
			if fi, e := d.Info(); e == nil {
				have += fi.Size()
			}
		}
		return nil
	})
	// 90% leaves room for the gap between the listed size and the bytes on disk.
	if have == 0 || (expectedGB > 0 && float64(have) < expectedGB*1e9*0.9) {
		return have, false
	}
	return have, true
}

// Start launches Colibri's server in the background and waits until it is ready.
func (c *Colibri) Start(ctx context.Context) error {
	if _, err := c.run(ctx, true, "start", "--background", "--no-browser"); err != nil {
		return fmt.Errorf("colibri start: %w", err)
	}
	deadline := time.Now().Add(c.cfg.ReadyTimeout)
	tick := time.NewTicker(2 * time.Second)
	defer tick.Stop()
	last := time.Now()
	for {
		st, err := c.status(ctx)
		if err == nil && st.Server.State == "ready" {
			base := st.Server.URLs.OpenAIBase
			if base == "" {
				base = fmt.Sprintf("http://%s:%d/v1", c.cfg.Host, c.cfg.Port)
			}
			c.mu.Lock()
			c.baseURL, c.modelID = strings.TrimRight(base, "/"), st.Server.ModelID
			c.mu.Unlock()
			c.logf("colibri ready at %s (model id %q)", base, st.Server.ModelID)
			return nil
		}
		if err == nil && (st.Server.State == "error" || st.Install.Phase == "error") {
			return fmt.Errorf("colibri failed to start: %s", firstNonEmpty(st.Install.Message, st.Server.State))
		}
		if time.Since(last) >= 30*time.Second {
			state := "unknown"
			if err == nil {
				state = st.Server.State
			}
			c.logf("waiting for colibri to load its model (state %s)", state)
			last = time.Now()
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("colibri not ready after %s", c.cfg.ReadyTimeout)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-tick.C:
		}
	}
}

// Healthy reports whether Colibri's API answers right now. It must answer with an
// OpenAI model list: something else listening on the port (an unrelated web server)
// does not count.
func (c *Colibri) Healthy() bool {
	base := c.BaseURL()
	if base == "" {
		return false
	}
	req, err := http.NewRequest(http.MethodGet, base+"/models", nil)
	if err != nil {
		return false
	}
	resp, err := c.cfg.HTTP.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		io.Copy(io.Discard, io.LimitReader(resp.Body, 1<<16))
		return false
	}
	var doc struct {
		Data []json.RawMessage `json:"data"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&doc); err != nil {
		return false
	}
	return doc.Data != nil
}

// HostPort is the address of the running server, for dialling it directly.
func (c *Colibri) HostPort() string {
	base := c.BaseURL()
	if base == "" {
		return ""
	}
	u, err := url.Parse(base)
	if err != nil {
		return ""
	}
	return u.Host
}

// Stop shuts the Colibri server and its engine down.
func (c *Colibri) Stop(ctx context.Context) error {
	c.mu.Lock()
	c.baseURL, c.modelID = "", ""
	c.mu.Unlock()
	if _, err := os.Stat(c.coli()); err != nil {
		return nil
	}
	_, err := c.run(ctx, false, "stop", "--port", strconv.Itoa(c.cfg.Port))
	return err
}

type colibriStatus struct {
	Install struct {
		Phase   string `json:"phase"`
		Running bool   `json:"running"`
		Message string `json:"message"`
	} `json:"install"`
	Server struct {
		State   string `json:"state"`
		ModelID string `json:"model_id"`
		URLs    struct {
			OpenAIBase string `json:"openai_base_url"`
		} `json:"urls"`
	} `json:"server"`
}

func (c *Colibri) status(ctx context.Context) (*colibriStatus, error) {
	out, err := c.run(ctx, false, "status", "--json")
	if err != nil {
		return nil, err
	}
	var st colibriStatus
	if err := json.Unmarshal(extractJSON(out), &st); err != nil {
		return nil, fmt.Errorf("parse colibri status: %w", err)
	}
	return &st, nil
}

// listedModel is one entry of `coli setup --list --json`.
type listedModel struct {
	ID          string  `json:"id"`
	Fits        bool    `json:"fits"`
	Recommended bool    `json:"recommended"`
	Status      string  `json:"status"`
	Reason      string  `json:"reason"`
	DownloadGB  float64 `json:"download_gb"`
}

// chooseModel resolves Colibri's one parameter. An explicit model wins; otherwise it
// is the model Colibri marks recommended for this machine. The second result is the
// model's listed download size in GB, 0 when Colibri did not list it.
func (c *Colibri) chooseModel(ctx context.Context) (string, float64, error) {
	out, err := c.run(ctx, false, "setup", "--list", "--json", "--dir", c.modelsDir())
	if c.cfg.Model != "" {
		if err != nil {
			return c.cfg.Model, 0, nil // the size check is optional for an explicit model
		}
		return c.cfg.Model, listedSize(out, c.cfg.Model), nil
	}
	if err != nil {
		return "", 0, fmt.Errorf("colibri model list: %w", err)
	}
	id, err := pickRecommended(out)
	if err != nil {
		return "", 0, err
	}
	return id, listedSize(out, id), nil
}

func parseListed(out []byte) []listedModel {
	raw := extractJSON(out)
	var models []listedModel
	if json.Unmarshal(raw, &models) == nil {
		return models
	}
	var wrapped struct {
		Models []listedModel `json:"models"`
	}
	if json.Unmarshal(raw, &wrapped) == nil {
		return wrapped.Models
	}
	return nil
}

func listedSize(out []byte, id string) float64 {
	for _, m := range parseListed(out) {
		if m.ID == id {
			return m.DownloadGB
		}
	}
	return 0
}

func pickRecommended(out []byte) (string, error) {
	raw := extractJSON(out)
	var models []listedModel
	if err := json.Unmarshal(raw, &models); err != nil {
		var wrapped struct {
			Models []listedModel `json:"models"`
		}
		if err2 := json.Unmarshal(raw, &wrapped); err2 != nil || wrapped.Models == nil {
			return "", fmt.Errorf("parse colibri model list: %w", err)
		}
		models = wrapped.Models
	}
	for _, m := range models {
		if m.Recommended && m.Fits {
			return m.ID, nil
		}
	}
	var why []string
	for _, m := range models {
		if m.Recommended {
			why = append(why, m.ID+": "+firstNonEmpty(m.Reason, m.Status))
		}
	}
	return "", permanentError{fmt.Errorf("%w (%s)", ErrNoModelFits, strings.Join(why, "; "))}
}

// run executes `<python> c/coli <args>` with Colibri's state kept under Home. When
// stream is set, the output is also forwarded line by line to Logf.
func (c *Colibri) run(ctx context.Context, stream bool, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, c.cfg.Python, append([]string{c.coli()}, args...)...)
	cmd.Dir = c.srcDir()
	cmd.Env = append(append(os.Environ(),
		"COLI_SETUP_HOME="+c.stateDir(),
		"PYTHONUNBUFFERED=1"),
		c.cfg.Env...)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error { return syscall.Kill(-cmd.Process.Pid, syscall.SIGTERM) }
	cmd.WaitDelay = 10 * time.Second

	var stdout, stderr bytes.Buffer
	if stream {
		cmd.Stdout = io.MultiWriter(&stdout, &lineLogger{prefix: "[COLIBRI] ", logf: c.logf})
		cmd.Stderr = io.MultiWriter(&stderr, &lineLogger{prefix: "[COLIBRI] ", logf: c.logf})
	} else {
		cmd.Stdout, cmd.Stderr = &stdout, &stderr
	}
	if err := cmd.Run(); err != nil {
		tail := strings.TrimSpace(lastLines(stderr.String()+"\n"+stdout.String(), 6))
		if ctx.Err() != nil {
			return stdout.Bytes(), ctx.Err()
		}
		return stdout.Bytes(), fmt.Errorf("%s %s: %w: %s", filepath.Base(c.cfg.Python), strings.Join(args[:min(2, len(args))], " "), err, tail)
	}
	return stdout.Bytes(), nil
}

// fetchSource downloads and unpacks the release once. The unpack goes to a
// temporary directory that is renamed into place, so an interrupted run leaves
// nothing that looks installed.
func (c *Colibri) fetchSource(ctx context.Context) error {
	if _, err := os.Stat(c.coli()); err == nil {
		return nil
	}
	parent := filepath.Dir(c.srcDir())
	if err := os.MkdirAll(parent, 0o755); err != nil {
		return err
	}
	c.logf("downloading colibri %s from %s", c.cfg.Version, c.cfg.SourceURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.cfg.SourceURL, nil)
	if err != nil {
		return err
	}
	client := &http.Client{Timeout: 10 * time.Minute}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download %s: HTTP %d", c.cfg.SourceURL, resp.StatusCode)
	}
	tmp, err := os.MkdirTemp(parent, ".colibri-unpack-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)
	if err := untarGz(io.LimitReader(resp.Body, 512<<20), tmp); err != nil {
		return err
	}
	if _, err := os.Stat(filepath.Join(tmp, "c", "coli")); err != nil {
		return errors.New("archive does not contain c/coli")
	}
	os.RemoveAll(c.srcDir())
	if err := os.Rename(tmp, c.srcDir()); err != nil {
		return err
	}
	return nil
}

// untarGz extracts a gzip tar into dest, dropping the archive's single top-level
// directory. Links, devices and any path that escapes dest are skipped.
func untarGz(r io.Reader, dest string) error {
	gz, err := gzip.NewReader(r)
	if err != nil {
		return err
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	root := filepath.Clean(dest) + string(os.PathSeparator)
	for {
		h, err := tr.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		name := h.Name
		if i := strings.IndexByte(name, '/'); i >= 0 {
			name = name[i+1:]
		} else {
			continue // the top-level directory entry itself
		}
		if name == "" {
			continue
		}
		target := filepath.Join(dest, filepath.FromSlash(name))
		if !strings.HasPrefix(target+string(os.PathSeparator), root) {
			continue
		}
		switch h.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			mode := os.FileMode(h.Mode).Perm() | 0o600
			f, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
			if err != nil {
				return err
			}
			if _, err := io.Copy(f, tr); err != nil {
				f.Close()
				return err
			}
			if err := f.Close(); err != nil {
				return err
			}
		}
	}
}

// extractJSON returns the JSON document in output that may carry log lines around it.
func extractJSON(out []byte) []byte {
	out = bytes.TrimSpace(out)
	start := bytes.IndexAny(out, "{[")
	if start < 0 {
		return out
	}
	end := bytes.LastIndexAny(out, "}]")
	if end < start {
		return out[start:]
	}
	return out[start : end+1]
}

func lastLines(s string, n int) string {
	lines := strings.Split(strings.TrimSpace(s), "\n")
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	return strings.Join(lines, " | ")
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

// lineLogger forwards complete lines (split on \n or \r, as progress bars use \r)
// to logf with a prefix.
type lineLogger struct {
	prefix string
	logf   func(string, ...any)
	mu     sync.Mutex
	buf    []byte
}

func (l *lineLogger) Write(p []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.buf = append(l.buf, p...)
	sc := bufio.NewScanner(bytes.NewReader(l.buf))
	sc.Split(func(data []byte, atEOF bool) (int, []byte, error) {
		if i := bytes.IndexAny(data, "\r\n"); i >= 0 {
			return i + 1, data[:i], nil
		}
		return 0, nil, nil
	})
	consumed := 0
	for sc.Scan() {
		consumed += len(sc.Bytes()) + 1
		if line := strings.TrimSpace(sc.Text()); line != "" && l.logf != nil {
			l.logf("%s%s", l.prefix, line)
		}
	}
	l.buf = l.buf[min(consumed, len(l.buf)):]
	return len(p), nil
}

// colibriFamilyFeatures lists what each Colibri v2.0.0 engine family accepts beyond
// plain chat: tool calls and grammar-forced output (response_format json_object,
// json_schema or grammar). It mirrors c/family_registry.py and is keyed by the `arch`
// Colibri's /health reports. An engine not listed is assumed to support neither, so
// requests that need them go to llama.cpp instead of failing.
var colibriFamilyFeatures = map[string]struct{ tools, grammar bool }{
	"glm53": {true, false}, "glm": {true, true}, "inkling": {false, false}, "kimi": {true, false},
	"olmoe": {false, false}, "qwen36": {false, false}, "qwen38": {true, false},
	"deepseek_v4": {true, false}, "deepseek_v41": {true, false}, "mimo": {true, false},
}

// Prepared reports whether a finished install is recorded for this version, so the
// next start only has to load the model and the llama.cpp fallback need not be started
// in case the install fails.
func (c *Colibri) Prepared() bool {
	b, err := os.ReadFile(c.recordPath())
	if err != nil {
		return false
	}
	var rec installRecord
	if json.Unmarshal(b, &rec) != nil || rec.Version != c.cfg.Version {
		return false
	}
	entries, err := os.ReadDir(c.modelsDir())
	return err == nil && len(entries) > 0
}

// features asks the running server which engine family answers and returns what that
// family accepts. ok is false when it cannot tell (the server is down, or answers
// something unexpected).
func (c *Colibri) features() (tools, grammar, ok bool) {
	base := c.BaseURL()
	if base == "" {
		return false, false, false
	}
	c.mu.Lock()
	fc := c.feat
	c.mu.Unlock()
	if fc.base == base && time.Since(fc.at) < time.Minute {
		return fc.tools, fc.grammar, true
	}
	u, err := url.Parse(base)
	if err != nil {
		return false, false, false
	}
	resp, err := c.cfg.HTTP.Get(u.Scheme + "://" + u.Host + "/health")
	if err != nil {
		return false, false, false
	}
	defer resp.Body.Close()
	var doc struct {
		Arch string `json:"arch"`
	}
	if resp.StatusCode != http.StatusOK || json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&doc) != nil {
		return false, false, false
	}
	f, known := colibriFamilyFeatures[doc.Arch]
	if known {
		c.mu.Lock()
		c.feat = featCache{base: base, at: time.Now(), tools: f.tools, grammar: f.grammar}
		c.mu.Unlock()
	}
	return f.tools, f.grammar, known
}

// SupportsGrammar reports whether the served engine accepts grammar-forced output.
func (c *Colibri) SupportsGrammar() bool { _, g, _ := c.features(); return g }

// SupportsTools reports whether the served engine accepts tool calls.
func (c *Colibri) SupportsTools() bool { t, _, _ := c.features(); return t }
