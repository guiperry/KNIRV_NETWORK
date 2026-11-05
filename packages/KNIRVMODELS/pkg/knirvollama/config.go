// Package knirvollama provisions and supervises Ollama and its embedding model the
// way pkg/knirvllama provisions llama.cpp and Qwen: pinned download, verified
// checksum, a supervised child bound to loopback, visible phases, and a status
// block. KNIRVSERVER reaches it only through KNIRVMODELS (Task 3.2, owner
// correction C1/C4); nothing in SERVER manages the process.
package knirvollama

import (
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	// DefaultBaseURL is where Ollama listens unless configured otherwise.
	DefaultBaseURL = "http://127.0.0.1:11434"
	// DefaultEmbeddingModel is the model the embedder (Task 3.1) uses.
	DefaultEmbeddingModel = "nomic-embed-text"
	// DefaultStartTimeout covers a first run (download plus model pull), the same
	// class as llama's 35 minutes.
	DefaultStartTimeout = 35 * time.Minute
)

// Config is the Ollama manager's configuration. Every field has an environment
// variable (FromEnv) so KNIRVSERVER can pass its yaml keys to the inference child.
type Config struct {
	// Enabled lets the manager install and start Ollama. Off, it still adopts an
	// instance that is already running and correct, and otherwise stays idle.
	Enabled bool
	// BaseURL is the endpoint to adopt or to start on (ollama_base_url / OLLAMA_HOST).
	BaseURL string

	// ServerURL and ServerSHA256 pin the Linux release artifact (ollama_server_url,
	// ollama_server_sha256). Version is informational (ollama_version).
	ServerURL    string
	ServerSHA256 string
	Version      string

	// EmbeddingModel and EmbeddingDigest pin the embedding model
	// (ollama_embedding_model, ollama_embedding_digest). An empty digest means the
	// model is not pinned: it is reported, not enforced.
	EmbeddingModel  string
	EmbeddingDigest string

	// DataDir is <app data>/knirvollama.
	DataDir string

	// Tuning passed to the child. Zero or empty leaves Ollama's default. The values
	// are the open D-H2 sizing items.
	NumParallel int
	Threads     int
	KeepAlive   string

	StartTimeout time.Duration
	PullTimeout  time.Duration
	StopTimeout  time.Duration
	// MaxStartAttempts is how many times Start tries when the child exits during
	// startup (default 3).
	MaxStartAttempts int
	RetryBackoff     []time.Duration

	// Logf receives operator-facing progress lines. Nil discards them.
	Logf func(format string, args ...any)
}

// DefaultConfig returns the defaults rooted at the server's app data directory.
func DefaultConfig() Config {
	return Config{
		BaseURL:        DefaultBaseURL,
		EmbeddingModel: DefaultEmbeddingModel,
		DataDir:        filepath.Join(AppDataDir(), "knirvollama"),
		KeepAlive:      "30m",
		StartTimeout:   DefaultStartTimeout,
		PullTimeout:    DefaultStartTimeout,
		StopTimeout:    10 * time.Second,
	}
}

// AppDataDir is the server data root: KNIRV_APP_DATA_DIR, else /var/lib/knirvserver,
// else under the home directory (the same rule pkg/knirvllama uses).
func AppDataDir() string {
	if d := strings.TrimSpace(os.Getenv("KNIRV_APP_DATA_DIR")); d != "" {
		return d
	}
	if err := os.MkdirAll("/var/lib/knirvserver", 0o755); err == nil {
		return "/var/lib/knirvserver"
	}
	if home, err := os.UserHomeDir(); err == nil {
		return filepath.Join(home, ".local", "share", "knirvserver")
	}
	return filepath.Join(os.TempDir(), "knirvserver")
}

// FromEnv overlays the KNIRV_OLLAMA_* variables (and OLLAMA_HOST for the endpoint)
// on the defaults.
//
//	KNIRV_OLLAMA_ENABLED=1        install and start Ollama if none is running
//	KNIRV_OLLAMA_BASE_URL         endpoint (else OLLAMA_HOST, else 127.0.0.1:11434)
//	KNIRV_OLLAMA_SERVER_URL       pinned release artifact (.tgz/.tar.gz, or a bare binary)
//	KNIRV_OLLAMA_SERVER_SHA256    its sha256
//	KNIRV_OLLAMA_VERSION          informational
//	KNIRV_OLLAMA_EMBEDDING_MODEL  default nomic-embed-text
//	KNIRV_OLLAMA_EMBEDDING_DIGEST pinned model digest
//	KNIRV_OLLAMA_NUM_PARALLEL, KNIRV_OLLAMA_THREADS, KNIRV_OLLAMA_KEEP_ALIVE
func FromEnv(logf func(string, ...any)) Config {
	c := DefaultConfig()
	c.Logf = logf
	get := func(k string) string { return strings.TrimSpace(os.Getenv(k)) }
	switch strings.ToLower(get("KNIRV_OLLAMA_ENABLED")) {
	case "1", "true", "yes", "on":
		c.Enabled = true
	}
	if v := get("KNIRV_OLLAMA_BASE_URL"); v != "" {
		c.BaseURL = v
	} else if v := get("OLLAMA_HOST"); v != "" {
		c.BaseURL = v
	}
	c.ServerURL = get("KNIRV_OLLAMA_SERVER_URL")
	c.ServerSHA256 = strings.ToLower(get("KNIRV_OLLAMA_SERVER_SHA256"))
	c.Version = get("KNIRV_OLLAMA_VERSION")
	if v := get("KNIRV_OLLAMA_EMBEDDING_MODEL"); v != "" {
		c.EmbeddingModel = v
	}
	c.EmbeddingDigest = strings.ToLower(get("KNIRV_OLLAMA_EMBEDDING_DIGEST"))
	if n, err := strconv.Atoi(get("KNIRV_OLLAMA_NUM_PARALLEL")); err == nil && n > 0 {
		c.NumParallel = n
	}
	if n, err := strconv.Atoi(get("KNIRV_OLLAMA_THREADS")); err == nil && n > 0 {
		c.Threads = n
	}
	if v := get("KNIRV_OLLAMA_KEEP_ALIVE"); v != "" {
		c.KeepAlive = v
	}
	return c
}

// normalize fills the zero values and returns the cleaned config.
func (c Config) normalize() Config {
	d := DefaultConfig()
	if c.BaseURL == "" {
		c.BaseURL = d.BaseURL
	}
	c.BaseURL = normalizeBaseURL(c.BaseURL)
	if c.EmbeddingModel == "" {
		c.EmbeddingModel = d.EmbeddingModel
	}
	if c.DataDir == "" {
		c.DataDir = d.DataDir
	}
	if c.StartTimeout <= 0 {
		c.StartTimeout = d.StartTimeout
	}
	if c.PullTimeout <= 0 {
		c.PullTimeout = d.PullTimeout
	}
	if c.StopTimeout <= 0 {
		c.StopTimeout = d.StopTimeout
	}
	if c.MaxStartAttempts <= 0 {
		c.MaxStartAttempts = 3
	}
	if c.RetryBackoff == nil {
		c.RetryBackoff = []time.Duration{5 * time.Second, 30 * time.Second}
	}
	if c.Logf == nil {
		c.Logf = func(string, ...any) {}
	}
	return c
}

// normalizeBaseURL accepts "host:port", "http://host:port" and a bare host
// (OLLAMA_HOST allows all three) and returns "http://host:port".
func normalizeBaseURL(s string) string {
	s = strings.TrimRight(strings.TrimSpace(s), "/")
	if !strings.Contains(s, "://") {
		s = "http://" + s
	}
	u, err := url.Parse(s)
	if err != nil || u.Host == "" {
		return s
	}
	if u.Port() == "" {
		u.Host = u.Hostname() + ":11434"
		if strings.Contains(u.Hostname(), ":") {
			u.Host = "[" + u.Hostname() + "]:11434"
		}
	}
	return u.Scheme + "://" + u.Host
}
