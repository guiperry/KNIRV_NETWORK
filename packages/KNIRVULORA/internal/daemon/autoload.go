package daemon

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"ulora/internal/gguf"
)

// Adapter auto-load.
//
// At start, ulorad reads one descriptor file and loads every adapter it names
// into an in-memory registry. The descriptor lives at:
//
//	<DataDir>/adapters/autoload.json
//
// (DataDir is the same root as ULORA_DATA_DIR / KNIRV_APP_DATA_DIR/ulora, see
// internal/config). A missing file means nothing is auto-loaded; it is not an
// error. Format, version 1:
//
//	{
//	  "version": 1,
//	  "adapters": [
//	    {
//	      "adapter_path": "adapters/<sha12>/<bundle12>.gguf",
//	      "base_model_path": "/models/llama-3.1-8b-q4.gguf"
//	    }
//	  ]
//	}
//
// Relative adapter_path values resolve against <DataDir>/adapters; base_model_path
// must be absolute or resolved by the operator (it is read, never written).
//
// Each entry is verified before it is loaded:
//   - the adapter must be a readable GGUF LoRA file whose recorded base_sha256
//     is present, and
//   - that recorded digest must equal the SHA-256 of base_model_path, computed
//     with the same cached function the GGUF and bind routes use.
//
// Any failure refuses that one entry, logs the reason (paths and digests only,
// never key material), and leaves the others loaded. Refusals never abort start.

const autoloadVersion = 1

// AutoloadFile is the on-disk descriptor for auto-loaded adapters.
type AutoloadFile struct {
	Version  int             `json:"version"`
	Adapters []AutoloadEntry `json:"adapters"`
}

// AutoloadEntry names one adapter and the base model it must be merged into.
type AutoloadEntry struct {
	AdapterPath   string `json:"adapter_path"`
	BaseModelPath string `json:"base_model_path"`
}

// LoadedAdapter is one adapter that passed auto-load verification.
type LoadedAdapter struct {
	AdapterPath  string `json:"adapter_path"`
	Architecture string `json:"architecture"`
	Skill        string `json:"skill,omitempty"`
	BundleHash   string `json:"bundle_hash"`
	BaseSHA256   string `json:"base_sha256"`
}

// adapterRegistry holds what auto-load accepted. It is populated once at start
// and read by the authenticated listing route.
type adapterRegistry struct {
	mu     sync.RWMutex
	loaded []LoadedAdapter
}

func (r *adapterRegistry) list() []LoadedAdapter {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]LoadedAdapter, len(r.loaded))
	copy(out, r.loaded)
	return out
}

// AutoloadPath is where the descriptor is read from for a given data dir.
func AutoloadPath(dataDir string) string {
	return filepath.Join(dataDir, "adapters", "autoload.json")
}

// autoloadAdapters reads the descriptor and verifies each entry. It never fails
// start: a bad descriptor is logged and nothing is loaded.
func (s *Server) autoloadAdapters() {
	path := AutoloadPath(s.dataDir())
	raw, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return
	}
	if err != nil {
		log.Printf("ulorad autoload: cannot read %s: %v", path, err)
		return
	}
	var file AutoloadFile
	if err := json.Unmarshal(raw, &file); err != nil {
		log.Printf("ulorad autoload: %s is not valid JSON: %v", path, err)
		return
	}
	if file.Version != autoloadVersion {
		log.Printf("ulorad autoload: %s version %d unsupported (want %d); nothing loaded", path, file.Version, autoloadVersion)
		return
	}
	for _, e := range file.Adapters {
		la, err := s.verifyAutoloadEntry(e)
		if err != nil {
			log.Printf("ulorad autoload: refused adapter %s: %v", e.AdapterPath, err)
			continue
		}
		s.adapters.mu.Lock()
		s.adapters.loaded = append(s.adapters.loaded, la)
		s.adapters.mu.Unlock()
		log.Printf("ulorad autoload: loaded adapter %s (skill=%s base_sha256=%s)", la.AdapterPath, la.Skill, la.BaseSHA256)
	}
}

// verifyAutoloadEntry checks one entry and returns the loaded record, or an
// error naming the refusal reason.
func (s *Server) verifyAutoloadEntry(e AutoloadEntry) (LoadedAdapter, error) {
	adapterPath := strings.TrimSpace(e.AdapterPath)
	basePath := strings.TrimSpace(e.BaseModelPath)
	if adapterPath == "" {
		return LoadedAdapter{}, fmt.Errorf("adapter_path is required")
	}
	if basePath == "" {
		return LoadedAdapter{}, fmt.Errorf("base_model_path is required")
	}
	if !filepath.IsAbs(adapterPath) {
		adapterPath = filepath.Join(s.dataDir(), "adapters", adapterPath)
	}

	info, _, err := gguf.ReadAdapterInfo(adapterPath)
	if err != nil {
		return LoadedAdapter{}, fmt.Errorf("read adapter: %w", err)
	}
	recorded := strings.ToLower(strings.TrimSpace(info.BaseSHA256))
	if recorded == "" {
		return LoadedAdapter{}, fmt.Errorf("adapter records no base_sha256; its base cannot be verified")
	}

	actual, err := s.baseSHA256(basePath)
	if err != nil {
		return LoadedAdapter{}, fmt.Errorf("hash base model: %w", err)
	}
	if !strings.EqualFold(recorded, actual) {
		return LoadedAdapter{}, fmt.Errorf("base model mismatch: adapter targets base_sha256=%s, base_model_path %s is %s",
			recorded, basePath, actual)
	}

	return LoadedAdapter{
		AdapterPath:  adapterPath,
		Architecture: info.Architecture,
		Skill:        info.Skill,
		BundleHash:   info.BundleHash,
		BaseSHA256:   recorded,
	}, nil
}

// handleAutoloaded lists the adapters auto-load accepted at start.
func (s *Server) handleAutoloaded(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"adapters": s.adapters.list()})
}
