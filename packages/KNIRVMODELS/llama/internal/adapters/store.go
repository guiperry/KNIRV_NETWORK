package adapters

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sync"
)

// Entry is one registered adapter.
type Entry struct {
	Name       string  `json:"name"`
	Path       string  `json:"path"`
	BaseSHA256 string  `json:"base_sha256"`
	BundleHash string  `json:"bundle_hash,omitempty"`
	Skill      string  `json:"skill,omitempty"`
	Alpha      float64 `json:"alpha,omitempty"`
}

var nameRE = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{0,63}$`)

// ValidName reports whether name is usable as an adapter name in a request.
func ValidName(name string) bool { return nameRE.MatchString(name) }

// Store persists the registered adapters. Order is significant: it is the order
// they are passed to llama-server, and therefore their ids.
type Store struct {
	mu      sync.Mutex
	path    string
	entries []Entry
}

// OpenStore loads (or starts) the registry stored in dir/adapters.json.
func OpenStore(dir string) (*Store, error) {
	s := &Store{path: filepath.Join(dir, "adapters.json")}
	raw, err := os.ReadFile(s.path)
	if os.IsNotExist(err) {
		return s, nil
	}
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(raw, &s.entries); err != nil {
		return nil, fmt.Errorf("adapter registry %s is corrupt: %w", s.path, err)
	}
	return s, nil
}

// List returns a copy of the entries in load order.
func (s *Store) List() []Entry {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]Entry(nil), s.entries...)
}

// Set replaces the whole list and persists it (temp file + rename).
func (s *Store) Set(entries []Entry) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	raw, err := json.MarshalIndent(entries, "", " ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o644); err != nil {
		return err
	}
	if err := os.Rename(tmp, s.path); err != nil {
		return err
	}
	s.entries = append([]Entry(nil), entries...)
	return nil
}
