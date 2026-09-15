// Package connector owns the per-family projection bridges that map between a
// base model's native frame and the uLoRA canonical space.
//
// A connector is a property of the MODEL, not of a skill. P_in comes from the
// target model's own principal subspace and P_out from its output basis; nothing
// about the corpus or a canonical core enters either. Shipping connectors inside
// bundles therefore put them in the wrong place: the same skill targeting three
// models would carry three connectors in every bundle, and a model that appeared
// later could not bind a perfectly good core.
//
// So they live here instead: derived once per model family, cached on disk, and
// reused across every skill that binds to that model.
package connector

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"ulora/internal/api"
	"ulora/internal/safetensors"
)

// Canonical core tensor names inside a cached connector file.
const (
	TensorPIn  = "P_in"
	TensorPOut = "P_out"
)

// Connector is the projection pair for one base model.
//
// PIn  ∈ R^(K × d_in)   so that A_target = A_core · PIn
// POut ∈ R^(d_out × K)  so that B_target = POut · B_core
type Connector struct {
	Family     string
	ParamCount string
	// CanonicalDim is the K the pair was built against. A connector is only
	// valid for a core of the same K, so it is stored with it rather than
	// assumed — otherwise a K change would silently misproject every tensor.
	CanonicalDim int
	PIn          [][]float32
	POut         [][]float32
}

// Key is the cache identity of a connector: the model family and size.
func Key(family, paramCount string) string {
	norm := func(s string) string {
		return strings.ToLower(strings.TrimSpace(s))
	}
	if norm(paramCount) == "" {
		return norm(family)
	}
	return norm(family) + "-" + norm(paramCount)
}

// Cache stores connectors as one deterministic safetensors file per model.
type Cache struct {
	dir string
}

// NewCache returns a cache rooted at dir. The directory is created lazily on
// first write, so a read-only deployment can still resolve existing connectors.
func NewCache(dir string) *Cache {
	return &Cache{dir: dir}
}

// pathFor returns the file backing a key, and whether the key is usable at all.
func (c *Cache) pathFor(family, paramCount string) (string, error) {
	key := Key(family, paramCount)
	if key == "" {
		return "", fmt.Errorf("connector cache: family is required to identify a connector")
	}
	if key != filepath.Base(key) {
		return "", fmt.Errorf("connector cache: refusing key %q: it is not a single path element", key)
	}
	return filepath.Join(c.dir, key+".safetensors"), nil
}

// Get returns the cached connector for a model, or (nil, nil) when none has been
// derived yet.
//
// A connector built against a different K is reported as absent rather than
// returned: it would multiply against a core of the wrong width, and the
// resulting matrices would have plausible shapes and meaningless values.
func (c *Cache) Get(family, paramCount string, canonicalDim int) (*Connector, error) {
	if c == nil || c.dir == "" {
		return nil, nil
	}
	path, err := c.pathFor(family, paramCount)
	if err != nil {
		return nil, err
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("connector cache: read %s: %w", path, err)
	}

	file, err := safetensors.Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("connector cache: parse %s: %w", path, err)
	}

	pIn, err := floatMatrix(file, TensorPIn)
	if err != nil {
		return nil, fmt.Errorf("connector cache %s: %w", path, err)
	}
	pOut, err := floatMatrix(file, TensorPOut)
	if err != nil {
		return nil, fmt.Errorf("connector cache %s: %w", path, err)
	}

	// K is recoverable from the shapes: P_in is K × d_in and P_out is d_out × K.
	storedK := len(pIn)
	if canonicalDim > 0 && storedK != canonicalDim {
		return nil, nil
	}

	return &Connector{
		Family:       family,
		ParamCount:   paramCount,
		CanonicalDim: storedK,
		PIn:          pIn,
		POut:         pOut,
	}, nil
}

// Put writes a connector to the cache. The file is written via a temp file and
// renamed, so a crash mid-write cannot leave a half-parsed connector that a later
// bind would read as valid.
func (c *Cache) Put(conn *Connector) error {
	if c == nil || c.dir == "" {
		return fmt.Errorf("connector cache: no directory configured")
	}
	if conn == nil {
		return fmt.Errorf("connector cache: nil connector")
	}
	if len(conn.PIn) == 0 || len(conn.POut) == 0 {
		return fmt.Errorf("connector cache: refusing to store an empty connector for %s", Key(conn.Family, conn.ParamCount))
	}

	path, err := c.pathFor(conn.Family, conn.ParamCount)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(c.dir, 0o755); err != nil {
		return fmt.Errorf("connector cache: create %s: %w", c.dir, err)
	}

	raw, err := safetensors.Write(map[string][][]float32{
		TensorPIn:  conn.PIn,
		TensorPOut: conn.POut,
	}, map[string]string{"canonical_dim": fmt.Sprint(conn.CanonicalDim)})
	if err != nil {
		return fmt.Errorf("connector cache: encode %s: %w", path, err)
	}

	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o644); err != nil {
		return fmt.Errorf("connector cache: write %s: %w", tmp, err)
	}
	if err := os.Rename(tmp, path); err != nil {
		return fmt.Errorf("connector cache: commit %s: %w", path, err)
	}
	return nil
}

// ConnectorFor implements runtime.ConnectorProvider, resolving the projection for
// a model from the cache.
//
// Returns (nil, nil, nil) when this model has no connector yet — the runtime then
// knows it must derive one — as distinct from an error, which means the lookup
// itself failed.
func (c *Cache) ConnectorFor(spec api.BaseModelSpec, canonicalDim int) ([][]float32, [][]float32, error) {
	conn, err := c.Get(spec.Family, spec.ParamCount, canonicalDim)
	if err != nil {
		return nil, nil, err
	}
	if conn == nil {
		return nil, nil, nil
	}
	return conn.PIn, conn.POut, nil
}

// Digest is a content hash of the connector pair, for logging which projection a
// bind actually used.
func (c *Connector) Digest() string {
	h := sha256.New()
	for _, m := range [][][]float32{c.PIn, c.POut} {
		for _, row := range m {
			for _, v := range row {
				fmt.Fprintf(h, "%v,", v)
			}
		}
	}
	return hex.EncodeToString(h.Sum(nil))
}

func floatMatrix(file *safetensors.File, name string) ([][]float32, error) {
	tensor, ok := file.Tensor(name)
	if !ok {
		return nil, fmt.Errorf("cached connector has no tensor %q", name)
	}
	return tensor.Float32Matrix()
}
