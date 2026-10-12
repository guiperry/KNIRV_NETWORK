package head

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
)

// ErrNoModel means the trainer has not promoted a model yet.
var ErrNoModel = errors.New("no HEAD model has been promoted")

var versionRE = regexp.MustCompile(`^v[0-9]+-[0-9a-f]{8}$`)

// Manifest is what clients fetch before downloading weights.
type Manifest struct {
	Version   string          `json:"version"`
	Config    json.RawMessage `json:"config"`
	SHA256    string          `json:"sha256"`
	Size      int64           `json:"size"`
	Signature string          `json:"signature,omitempty"`
}

// Service serves the model the trainer last promoted into Dir
// (layout written by 5_HEAD_TRAINER: head_config.json, model_latest -> model-<v>.safetensors).
type Service struct {
	Dir string

	mu       sync.Mutex
	runner   *Runner
	loaded   string // version currently in the runner
	info     Info
	hashes   map[string]string
	hashSize map[string]int64
}

func NewService(dir string) *Service {
	return &Service{Dir: dir, hashes: map[string]string{}, hashSize: map[string]int64{}}
}

func (s *Service) latestVersion() (string, error) {
	target, err := os.Readlink(filepath.Join(s.Dir, "model_latest"))
	if err != nil {
		return "", ErrNoModel
	}
	name := filepath.Base(target)
	v := strings.TrimSuffix(strings.TrimPrefix(name, "model-"), ".safetensors")
	if !strings.HasPrefix(name, "model-") || !versionRE.MatchString(v) {
		return "", fmt.Errorf("model_latest points at unexpected file %q", name)
	}
	return v, nil
}

func (s *Service) hashOf(version string) (string, int64, error) {
	if h, ok := s.hashes[version]; ok { // weight files are immutable per version
		return h, s.hashSize[version], nil
	}
	f, err := os.Open(filepath.Join(s.Dir, "model-"+version+".safetensors"))
	if err != nil {
		return "", 0, err
	}
	defer f.Close()
	h := sha256.New()
	n, err := io.Copy(h, f)
	if err != nil {
		return "", 0, err
	}
	sum := hex.EncodeToString(h.Sum(nil))
	s.hashes[version], s.hashSize[version] = sum, n
	return sum, n, nil
}

// Manifest describes the promoted model.
func (s *Service) Manifest() (*Manifest, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	v, err := s.latestVersion()
	if err != nil {
		return nil, err
	}
	cfg, err := os.ReadFile(filepath.Join(s.Dir, "head_config.json"))
	if err != nil {
		return nil, err
	}
	var probe struct {
		ModelVersion string `json:"model_version"`
	}
	if json.Unmarshal(cfg, &probe) != nil || probe.ModelVersion != v {
		return nil, fmt.Errorf("head_config.json (%q) does not match model_latest (%q)", probe.ModelVersion, v)
	}
	sum, size, err := s.hashOf(v)
	if err != nil {
		return nil, err
	}
	return &Manifest{Version: v, Config: cfg, SHA256: sum, Size: size}, nil
}

// WeightsPath returns the file for a promoted or kept version. The version is
// validated against a strict pattern: it is user-supplied.
func (s *Service) WeightsPath(version string) (string, error) {
	if !versionRE.MatchString(version) {
		return "", errors.New("invalid version")
	}
	p := filepath.Join(s.Dir, "model-"+version+".safetensors")
	if _, err := os.Stat(p); err != nil {
		return "", err
	}
	return p, nil
}

// ETag is the SHA-256 of a version's weights (immutable per version).
func (s *Service) ETag(version string) (string, error) {
	if !versionRE.MatchString(version) {
		return "", errors.New("invalid version")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	h, _, err := s.hashOf(version)
	return h, err
}

// Reason runs the promoted model, (re)loading it when the trainer promotes a new version.
func (s *Service) Reason(ctx context.Context, tokens []uint32) (json.RawMessage, error) {
	s.mu.Lock()
	v, err := s.latestVersion()
	if err != nil {
		s.mu.Unlock()
		return nil, err
	}
	if s.runner == nil {
		if s.runner, err = NewRunner(ctx, s.Dir); err != nil {
			s.mu.Unlock()
			return nil, err
		}
	}
	if s.loaded != v {
		info, err := s.runner.Load("head_config.json", "model-"+v+".safetensors")
		if err != nil {
			s.mu.Unlock()
			return nil, fmt.Errorf("load %s: %w", v, err)
		}
		s.loaded, s.info = v, info
	}
	runner, info := s.runner, s.info
	s.mu.Unlock()

	if len(tokens) == 0 || len(tokens) > info.MaxInputLen {
		return nil, fmt.Errorf("input must be 1..%d tokens", info.MaxInputLen)
	}
	return runner.Reason(tokens)
}

// Status is reported by /heart/health.
func (s *Service) Status() map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()
	v, err := s.latestVersion()
	if err != nil {
		return map[string]any{"ready": false, "reason": err.Error()}
	}
	return map[string]any{"ready": true, "version": v, "loaded": s.loaded == v}
}

func (s *Service) Close(ctx context.Context) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.runner != nil {
		s.runner.Close(ctx)
		s.runner = nil
		s.loaded = ""
	}
}
