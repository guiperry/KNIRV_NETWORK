package vlmserver

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// ProvisionFile is one weight file the MiniCPM server needs at runtime.
type ProvisionFile struct {
	Name   string // file name inside the model directory
	URL    string // HTTPS source
	Size   int64  // expected byte length
	SHA256 string // expected lowercase hex SHA-256 (the Hugging Face LFS oid)
}

// MiniCPMFiles is the pinned set installed on first run: the IQ4_XS language model
// (openbmb/MiniCPM-V-4.7-35B-A3B-gguf, architecture qwen35moe, Apache-2.0 tag) and its
// vision projector.
var MiniCPMFiles = []ProvisionFile{
	{
		Name:   "MiniCPM-V-4_7-35B-A3B-IQ4_XS.gguf",
		URL:    "https://huggingface.co/openbmb/MiniCPM-V-4.7-35B-A3B-gguf/resolve/main/MiniCPM-V-4_7-35B-A3B-IQ4_XS.gguf",
		Size:   18728286400,
		SHA256: "fce60c709a7f22098980bda5b9b59e0969f4847b2657a2d91a6fe90ff14cefe9",
	},
	{
		Name:   "mmproj-model-f16.gguf",
		URL:    "https://huggingface.co/openbmb/MiniCPM-V-4.7-35B-A3B-gguf/resolve/main/mmproj-model-f16.gguf",
		Size:   1119575840,
		SHA256: "bff3b688e14c300eca273092c37c50a0ddfcc2054e079af1a0480edeed5d20f1",
	},
}

// requireHTTPS limits downloads to https. Tests point at a local server and clear it.
var requireHTTPS = true

// bodyIdleTimeout is how long a download body may make no progress before the
// transfer is aborted. It is an idle bound, not an overall deadline: a healthy
// 18.7 GB transfer can run for hours. It is a var only so tests can shorten it;
// production keeps the 2 minute default.
var bodyIdleTimeout = 2 * time.Minute

// watchIdleBody closes body when no Read makes progress for timeout. The HTTP
// client's ResponseHeaderTimeout covers the wait before the body starts; this
// covers a server that accepts the request and then stalls mid-body.
func watchIdleBody(body io.ReadCloser, timeout time.Duration) io.ReadCloser {
	ib := &idleBody{r: body, activity: make(chan struct{}, 1), stop: make(chan struct{})}
	go func() {
		timer := time.NewTimer(timeout)
		defer timer.Stop()
		for {
			select {
			case <-ib.activity:
				if !timer.Stop() {
					select {
					case <-timer.C:
					default:
					}
				}
				timer.Reset(timeout)
			case <-timer.C:
				_ = body.Close()
				return
			case <-ib.stop:
				return
			}
		}
	}()
	return ib
}

type idleBody struct {
	r        io.ReadCloser
	activity chan struct{}
	stop     chan struct{}
	once     sync.Once
}

func (b *idleBody) Read(p []byte) (int, error) {
	n, err := b.r.Read(p)
	if n > 0 {
		select {
		case b.activity <- struct{}{}:
		default:
		}
	}
	return n, err
}

func (b *idleBody) Close() error {
	b.once.Do(func() { close(b.stop) })
	return b.r.Close()
}

// verifiedRecord is written next to a verified file so a restart does not re-hash
// 18 GB. It is trusted only while size and modification time still match.
type verifiedRecord struct {
	Name    string `json:"name"`
	Size    int64  `json:"size"`
	ModUnix int64  `json:"mod_unix"`
	SHA256  string `json:"sha256"`
}

// Provision makes sure every file is in dir with the pinned size and SHA-256. A verified
// file is kept; a partial download resumes with a Range request; a file whose hash does
// not match is deleted and reported, so a substituted weight is never loaded.
func Provision(ctx context.Context, dir string, files []ProvisionFile, client *http.Client, logf func(string, ...any)) (map[string]string, error) {
	if logf == nil {
		logf = func(string, ...any) {}
	}
	if client == nil {
		client = http.DefaultClient
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create vlm model dir: %w", err)
	}
	paths := make(map[string]string, len(files))
	for _, f := range files {
		path := filepath.Join(dir, f.Name)
		paths[f.Name] = path
		if verifiedCached(path, f) {
			logf("[vlm] %s verified (cached)", f.Name)
			continue
		}
		if ok, _ := verifyFile(path, f); ok {
			recordVerified(path, f)
			logf("[vlm] %s verified", f.Name)
			continue
		}
		if err := download(ctx, client, f, path, logf); err != nil {
			return nil, fmt.Errorf("provision %s: %w", f.Name, err)
		}
		ok, got := verifyFile(path, f)
		if !ok {
			_ = os.Remove(path)
			return nil, fmt.Errorf("provision %s: sha256 mismatch (got %s, want %s); file removed", f.Name, got, f.SHA256)
		}
		recordVerified(path, f)
		logf("[vlm] %s downloaded and verified", f.Name)
	}
	return paths, nil
}

func verifiedCached(path string, f ProvisionFile) bool {
	st, err := os.Stat(path)
	if err != nil || st.Size() != f.Size {
		return false
	}
	b, err := os.ReadFile(path + ".verified")
	if err != nil {
		return false
	}
	var rec verifiedRecord
	if json.Unmarshal(b, &rec) != nil {
		return false
	}
	return rec.Name == f.Name && rec.Size == st.Size() && rec.ModUnix == st.ModTime().Unix() && rec.SHA256 == f.SHA256
}

func recordVerified(path string, f ProvisionFile) {
	st, err := os.Stat(path)
	if err != nil {
		return
	}
	rec := verifiedRecord{Name: f.Name, Size: st.Size(), ModUnix: st.ModTime().Unix(), SHA256: f.SHA256}
	b, _ := json.Marshal(rec)
	_ = os.WriteFile(path+".verified", b, 0o600)
}

// verifyFile reports whether path has the pinned size and hash, and the hash it saw.
func verifyFile(path string, f ProvisionFile) (bool, string) {
	st, err := os.Stat(path)
	if err != nil || st.Size() != f.Size {
		return false, ""
	}
	h, err := hashFile(path)
	if err != nil {
		return false, ""
	}
	return h == f.SHA256, h
}

func hashFile(path string) (string, error) {
	fh, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer fh.Close()
	h := sha256.New()
	if _, err := io.Copy(h, fh); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// download fetches f into path through path+".part", resuming an existing partial file.
func download(ctx context.Context, client *http.Client, f ProvisionFile, path string, logf func(string, ...any)) error {
	if requireHTTPS && !strings.HasPrefix(f.URL, "https://") {
		return fmt.Errorf("refusing non-HTTPS source %q", f.URL)
	}
	part := path + ".part"
	var have int64
	if st, err := os.Stat(part); err == nil && st.Size() < f.Size {
		have = st.Size()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, f.URL, nil)
	if err != nil {
		return err
	}
	if have > 0 {
		req.Header.Set("Range", fmt.Sprintf("bytes=%d-", have))
		logf("[vlm] resuming %s at %d bytes", f.Name, have)
	} else {
		logf("[vlm] downloading %s (%d bytes)", f.Name, f.Size)
	}
	res, err := client.Do(req)
	if err != nil {
		return err
	}
	body := watchIdleBody(res.Body, bodyIdleTimeout)
	defer body.Close()
	switch {
	case res.StatusCode == http.StatusPartialContent && have > 0:
	case res.StatusCode == http.StatusOK:
		have = 0 // the server ignored Range: start over
	default:
		return fmt.Errorf("unexpected HTTP status %s", res.Status)
	}
	flags := os.O_CREATE | os.O_WRONLY
	if have > 0 {
		flags |= os.O_APPEND
	} else {
		flags |= os.O_TRUNC
	}
	out, err := os.OpenFile(part, flags, 0o600)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, body); err != nil {
		out.Close()
		return fmt.Errorf("write %s: %w", f.Name, err)
	}
	if err := out.Close(); err != nil {
		return err
	}
	return os.Rename(part, path)
}
