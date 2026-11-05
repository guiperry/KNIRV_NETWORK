package knirvollama

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
)

// maxArtifactBytes bounds a download and an extraction (the Linux release with CUDA
// libraries is about 1.5 GB).
const maxArtifactBytes = 8 << 30

// probe reports whether base answers like Ollama.
func probe(ctx context.Context, hc *http.Client, base string) (banner string, ok bool) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, base+"/", nil)
	if err != nil {
		return "", false
	}
	resp, err := hc.Do(req)
	if err != nil {
		return "", false
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(io.LimitReader(resp.Body, 256))
	return strings.TrimSpace(string(b)), strings.Contains(string(b), "Ollama is running")
}

// hostOf splits a base URL into host and port.
func hostOf(base string) (host string, port int, err error) {
	u, err := url.Parse(base)
	if err != nil {
		return "", 0, err
	}
	host = u.Hostname()
	port, _ = strconv.Atoi(u.Port())
	if port == 0 {
		port = 11434
	}
	return host, port, nil
}

// isLoopbackHost reports whether host can only be reached from this machine. An
// empty host and the unspecified addresses 0.0.0.0 and :: are not loopback: Ollama
// treats them as "all interfaces" (E4).
func isLoopbackHost(host string) bool {
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

// listenerWildcard reports whether a TCP listener on port is bound to all
// interfaces, by reading /proc/net/tcp and tcp6. found is false when no listener
// was seen or the table could not be read, in which case only the configured host
// can be judged. Replaced by tests.
var listenerWildcard = func(port int) (wildcard, found bool) {
	for _, file := range []string{"/proc/net/tcp", "/proc/net/tcp6"} {
		data, err := os.ReadFile(file)
		if err != nil {
			continue
		}
		for i, line := range strings.Split(string(data), "\n") {
			f := strings.Fields(line)
			if i == 0 || len(f) < 4 || f[3] != "0A" { // 0A = LISTEN
				continue
			}
			addr, p, ok := strings.Cut(f[1], ":")
			if !ok {
				continue
			}
			if n, err := strconv.ParseInt(p, 16, 32); err != nil || int(n) != port {
				continue
			}
			found = true
			if strings.Trim(addr, "0") == "" { // 00000000 or 32 zeros
				wildcard = true
			}
		}
	}
	return wildcard, found
}

// checkBind returns a description of the bind for the status block and an error
// when the endpoint is reachable from beyond this machine.
func checkBind(base string) (bind string, err error) {
	host, port, perr := hostOf(base)
	if perr != nil {
		return "", fmt.Errorf("bad endpoint %q: %w", base, perr)
	}
	if !isLoopbackHost(host) {
		return host, fmt.Errorf("endpoint %s is not loopback; refusing to use an Ollama that listens on all interfaces", base)
	}
	if wildcard, found := listenerWildcard(port); found && wildcard {
		return "0.0.0.0", fmt.Errorf("an Ollama listener on port %d is bound to all interfaces (0.0.0.0); rebind it to 127.0.0.1", port)
	}
	return host, nil
}

// installRecord is written after a verified install so a rerun does not download
// again (the same idea as Colibri's install record).
type installRecord struct {
	URL    string `json:"url"`
	SHA256 string `json:"sha256"`
	Binary string `json:"binary"`
	Size   int64  `json:"size"`
}

func (c Config) binDir() string       { return filepath.Join(c.DataDir, "bin") }
func (c Config) recordPath() string   { return filepath.Join(c.DataDir, "install.json") }
func (c Config) modelsDir() string    { return filepath.Join(c.DataDir, "models") }
func (c Config) installedBin() string { return filepath.Join(c.binDir(), "bin", "ollama") }

// ErrUnpinned is returned when an install is requested without a pinned artifact.
var ErrUnpinned = errors.New("ollama_server_url and ollama_server_sha256 must both be set to install Ollama")

// EnsureBinary returns the path of the pinned Ollama binary, downloading and
// verifying it when it is not already installed. It never runs an installer script.
// A checksum mismatch installs nothing.
func EnsureBinary(ctx context.Context, cfg Config, hc *http.Client) (string, error) {
	cfg = cfg.normalize()
	if cfg.ServerURL == "" || cfg.ServerSHA256 == "" {
		return "", ErrUnpinned
	}
	want := strings.ToLower(strings.TrimPrefix(cfg.ServerSHA256, "sha256:"))
	if rec, ok := readRecord(cfg.recordPath()); ok && rec.URL == cfg.ServerURL && rec.SHA256 == want {
		if st, err := os.Stat(rec.Binary); err == nil && st.Mode().IsRegular() && st.Size() == rec.Size {
			return rec.Binary, nil // already installed: resume-safe, idempotent
		}
	}
	if err := os.MkdirAll(cfg.DataDir, 0o755); err != nil {
		return "", err
	}
	tmp, err := os.CreateTemp(cfg.DataDir, "download-*")
	if err != nil {
		return "", err
	}
	defer os.Remove(tmp.Name())
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, cfg.ServerURL, nil)
	if err != nil {
		tmp.Close()
		return "", err
	}
	resp, err := hc.Do(req)
	if err != nil {
		tmp.Close()
		return "", fmt.Errorf("download %s: %w", cfg.ServerURL, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		tmp.Close()
		return "", fmt.Errorf("download %s: HTTP %d", cfg.ServerURL, resp.StatusCode)
	}
	h := sha256.New()
	n, err := io.Copy(io.MultiWriter(tmp, h), io.LimitReader(resp.Body, maxArtifactBytes+1))
	tmp.Close()
	if err != nil {
		return "", fmt.Errorf("download %s: %w", cfg.ServerURL, err)
	}
	if n > maxArtifactBytes {
		return "", fmt.Errorf("download %s: larger than %d bytes", cfg.ServerURL, int64(maxArtifactBytes))
	}
	if got := hex.EncodeToString(h.Sum(nil)); got != want {
		return "", fmt.Errorf("checksum mismatch for %s: got %s, pinned %s; nothing was installed", cfg.ServerURL, got, want)
	}

	stage, err := os.MkdirTemp(cfg.DataDir, "stage-*")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(stage)
	name := path.Base(strings.SplitN(cfg.ServerURL, "?", 2)[0])
	switch {
	case strings.HasSuffix(name, ".tgz") || strings.HasSuffix(name, ".tar.gz"):
		if err := extractTarGz(tmp.Name(), stage); err != nil {
			return "", err
		}
	case strings.HasSuffix(name, ".zst") || strings.HasSuffix(name, ".zip"):
		return "", fmt.Errorf("artifact %q: only .tgz/.tar.gz archives and bare binaries are supported (pin a .tgz release)", name)
	default:
		if err := os.MkdirAll(filepath.Join(stage, "bin"), 0o755); err != nil {
			return "", err
		}
		if err := copyFile(tmp.Name(), filepath.Join(stage, "bin", "ollama"), 0o755); err != nil {
			return "", err
		}
	}
	staged := filepath.Join(stage, "bin", "ollama")
	st, err := os.Stat(staged)
	if err != nil || !st.Mode().IsRegular() {
		return "", fmt.Errorf("artifact %s has no bin/ollama", cfg.ServerURL)
	}
	if err := os.Chmod(staged, 0o755); err != nil {
		return "", err
	}
	// Swap the verified tree into place.
	dest := cfg.binDir()
	if err := os.RemoveAll(dest); err != nil {
		return "", err
	}
	if err := os.Rename(stage, dest); err != nil {
		return "", err
	}
	bin := cfg.installedBin()
	fin, err := os.Stat(bin)
	if err != nil {
		return "", err
	}
	data, _ := json.MarshalIndent(installRecord{URL: cfg.ServerURL, SHA256: want, Binary: bin, Size: fin.Size()}, "", "  ")
	if err := os.WriteFile(cfg.recordPath(), data, 0o644); err != nil {
		return "", err
	}
	return bin, nil
}

func readRecord(p string) (installRecord, bool) {
	var rec installRecord
	data, err := os.ReadFile(p)
	if err != nil || json.Unmarshal(data, &rec) != nil {
		return rec, false
	}
	return rec, true
}

func copyFile(src, dst string, mode os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}

// extractTarGz unpacks a release archive into dir. Entries that would leave dir,
// device nodes and hard links are refused.
func extractTarGz(archive, dir string) error {
	f, err := os.Open(archive)
	if err != nil {
		return err
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return fmt.Errorf("artifact is not a gzip archive: %w", err)
	}
	defer gz.Close()
	root := filepath.Clean(dir) + string(os.PathSeparator)
	var total int64
	tr := tar.NewReader(gz)
	for {
		h, err := tr.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return fmt.Errorf("read archive: %w", err)
		}
		target := filepath.Join(dir, filepath.FromSlash(h.Name))
		if !strings.HasPrefix(target+string(os.PathSeparator), root) {
			return fmt.Errorf("archive entry %q escapes the install directory", h.Name)
		}
		switch h.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
		case tar.TypeReg:
			if total += h.Size; total > maxArtifactBytes {
				return fmt.Errorf("archive expands beyond %d bytes", int64(maxArtifactBytes))
			}
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(h.Mode)&0o755|0o600)
			if err != nil {
				return err
			}
			if _, err := io.Copy(out, io.LimitReader(tr, h.Size)); err != nil {
				out.Close()
				return err
			}
			if err := out.Close(); err != nil {
				return err
			}
		case tar.TypeSymlink:
			link := h.Linkname
			if filepath.IsAbs(link) || !strings.HasPrefix(filepath.Join(filepath.Dir(target), link)+string(os.PathSeparator), root) {
				return fmt.Errorf("archive symlink %q points outside the install directory", h.Name)
			}
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			if err := os.Symlink(link, target); err != nil {
				return err
			}
		default:
			return fmt.Errorf("archive entry %q has unsupported type %q", h.Name, h.Typeflag)
		}
	}
}

// digestMatches compares a model digest with a pin. Either may carry a "sha256:"
// prefix; a pin of at least 12 hex characters may be a prefix of the digest.
func digestMatches(got, want string) bool {
	got = strings.ToLower(strings.TrimPrefix(got, "sha256:"))
	want = strings.ToLower(strings.TrimPrefix(want, "sha256:"))
	if want == "" || got == "" {
		return false
	}
	return got == want || (len(want) >= 12 && strings.HasPrefix(got, want))
}
