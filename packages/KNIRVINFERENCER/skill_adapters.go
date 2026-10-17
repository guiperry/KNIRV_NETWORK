package inferencer

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	models "github.com/guiperry/knirv-sdk-go/models"
)

// SkillAdapters turns a compiled ULoRA bundle into an adapter the embedded llama
// is running, and says exactly what happened.
//
//	llama   GET  /v1/adapters                -> which base model is running (path)
//	ulorad  POST /ulora/v1/adapters/gguf     -> bind the bundle to THAT model, write a GGUF LoRA
//	llama   POST /v1/adapters                -> verify base hash, restart llama with it
//
// Every step can fail, and a failure is returned as an error: the caller (the
// harness) must then run the base model AND record that the skill did not run. It
// must never present base output as the skill's.
type SkillAdapters struct {
	Llama      *http.Client // dials llama.sock
	LlamaBase  string       // "http://knirv-llama"
	Ulora      *http.Client // dials ulora.sock
	UloraBase  string       // "http://ulora"
	UloraToken string

	mu sync.Mutex // one bind/register at a time: each may restart llama
}

// SkillAdapter is the result of Ensure.
type SkillAdapter struct {
	Name        string `json:"name"`        // pass as options["adapter"]
	BundleHash  string `json:"bundle_hash"` // what was bound
	BaseSHA256  string `json:"base_sha256"` // what it was bound to
	AdapterPath string `json:"adapter_path"`
	// Reused is true when nothing had to change: the adapter was already loaded.
	Reused bool `json:"reused"`
	// BoundNow is false when ULoRA served the adapter file from its cache.
	BoundNow bool `json:"bound_now"`
}

// AdapterName is the stable name an adapter gets in llama: derived from the bundle
// content hash, so the same bytes are always the same adapter.
func AdapterName(bundleHash string) string {
	h := strings.ToLower(bundleHash)
	if len(h) > 16 {
		h = h[:16]
	}
	return "skill-" + h
}

type llamaAdapters struct {
	Base struct {
		Path   string `json:"path"`
		SHA256 string `json:"sha256"`
	} `json:"base"`
	Adapters []struct {
		Name       string `json:"name"`
		Loaded     bool   `json:"loaded"`
		BaseSHA256 string `json:"base_sha256"`
		BundleHash string `json:"bundle_hash"`
	} `json:"adapters"`
}

func (s *SkillAdapters) do(ctx context.Context, c *http.Client, method, url string, body any, token string, out any) (int, string, error) {
	var rd io.Reader
	if body != nil {
		raw, _ := json.Marshal(body)
		rd = bytes.NewReader(raw)
	}
	req, err := http.NewRequestWithContext(ctx, method, url, rd)
	if err != nil {
		return 0, "", err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if c == s.Llama {
		// Adapters only exist in llama.cpp. The inference endpoint may be serving from
		// another engine (Colibri); this makes it start and use llama.cpp for the call.
		req.Header.Set("X-Knirv-Require-Backend", "llama.cpp")
	}
	res, err := c.Do(req)
	if err != nil {
		return 0, "", err
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(res.Body, 4<<20))
	if res.StatusCode/100 == 2 && out != nil {
		if err := json.Unmarshal(raw, out); err != nil {
			return res.StatusCode, string(raw), fmt.Errorf("malformed response: %w", err)
		}
	}
	return res.StatusCode, strings.TrimSpace(string(raw)), nil
}

// Ensure makes the bundle's adapter available in llama. contentHash, when given,
// must match the bundle (ULoRA refuses a mismatch).
func (s *SkillAdapters) Ensure(ctx context.Context, bundleID, contentHash string) (*SkillAdapter, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	ctx, cancel := context.WithTimeout(ctx, 10*time.Minute) // a reload loads the whole model
	defer cancel()

	var cur llamaAdapters
	code, body, err := s.do(ctx, s.Llama, http.MethodGet, s.LlamaBase+"/v1/adapters", nil, "", &cur)
	if err != nil || code != 200 {
		return nil, fmt.Errorf("llama adapters unavailable (HTTP %d): %v %s", code, err, body)
	}
	if cur.Base.Path == "" {
		return nil, fmt.Errorf("llama did not report its base model path")
	}

	var bound struct {
		AdapterPath string `json:"adapter_path"`
		BaseSHA256  string `json:"base_sha256"`
		ContentHash string `json:"content_hash"`
		Cached      bool   `json:"cached"`
	}
	code, body, err = s.do(ctx, s.Ulora, http.MethodPost, s.UloraBase+"/ulora/v1/adapters/gguf",
		map[string]string{"bundle_id": bundleID, "content_hash": contentHash, "base_model_path": cur.Base.Path}, s.UloraToken, &bound)
	if err != nil {
		return nil, fmt.Errorf("ulora unavailable: %w", err)
	}
	if code != 200 {
		return nil, fmt.Errorf("ulora refused to bind %q to the running model (HTTP %d): %s", bundleID, code, body)
	}

	name := AdapterName(bound.ContentHash)
	out := &SkillAdapter{Name: name, BundleHash: bound.ContentHash, BaseSHA256: bound.BaseSHA256, AdapterPath: bound.AdapterPath, BoundNow: !bound.Cached}
	for _, a := range cur.Adapters {
		if a.Name == name && a.Loaded && a.BundleHash == bound.ContentHash && a.BaseSHA256 == bound.BaseSHA256 {
			out.Reused = true
			return out, nil
		}
	}
	code, body, err = s.do(ctx, s.Llama, http.MethodPost, s.LlamaBase+"/v1/adapters",
		map[string]string{"name": name, "path": bound.AdapterPath}, "", nil)
	if err != nil {
		return nil, fmt.Errorf("llama unavailable while loading %s: %w", name, err)
	}
	if code != 201 {
		return nil, fmt.Errorf("llama refused adapter %s (HTTP %d): %s", name, code, body)
	}
	return out, nil
}

// NewSkillAdapters builds the orchestrator over the two unix sockets.
func NewSkillAdapters(llamaSocket, uloraSocket, uloraToken string) *SkillAdapters {
	return &SkillAdapters{
		Llama: models.NewUnixHTTPClient(llamaSocket, 11*time.Minute), LlamaBase: "http://knirv-llama",
		Ulora: models.NewUnixHTTPClient(uloraSocket, 11*time.Minute), UloraBase: "http://ulora", UloraToken: uloraToken,
	}
}
