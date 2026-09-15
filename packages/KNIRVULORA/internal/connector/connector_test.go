package connector

import (
	"os"
	"path/filepath"
	"testing"

	"ulora/internal/api"
)

func sample(family, paramCount string, k int) *Connector {
	pIn := make([][]float32, k)
	for i := range pIn {
		pIn[i] = make([]float32, 3)
		pIn[i][0] = float32(i) * 0.5
	}
	pOut := make([][]float32, 3)
	for i := range pOut {
		pOut[i] = make([]float32, k)
		pOut[i][0] = float32(i) * 0.25
	}
	return &Connector{Family: family, ParamCount: paramCount, CanonicalDim: k, PIn: pIn, POut: pOut}
}

func TestCacheRoundTrip(t *testing.T) {
	cache := NewCache(t.TempDir())
	want := sample("llama", "8b", 4)

	if err := cache.Put(want); err != nil {
		t.Fatalf("put: %v", err)
	}
	got, err := cache.Get("llama", "8b", 4)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got == nil {
		t.Fatal("connector should be present after a put")
	}
	if len(got.PIn) != 4 || len(got.POut) != 3 || got.CanonicalDim != 4 {
		t.Fatalf("round trip changed shape: PIn %dx%d POut %dx%d K=%d",
			len(got.PIn), len(got.PIn[0]), len(got.POut), len(got.POut[0]), got.CanonicalDim)
	}
	if got.PIn[2][0] != want.PIn[2][0] {
		t.Fatalf("values differ: %v vs %v", got.PIn[2][0], want.PIn[2][0])
	}
}

// Absence is (nil, nil) — a signal to derive one — and must not be confused with
// a lookup failure, which is an error.
func TestCacheMissIsNotAnError(t *testing.T) {
	cache := NewCache(t.TempDir())
	got, err := cache.Get("mistral", "7b", 4)
	if err != nil {
		t.Fatalf("a miss must not error: %v", err)
	}
	if got != nil {
		t.Fatalf("expected no connector, got %+v", got)
	}
}

// A connector built for a different K would multiply against a core of the wrong
// width and yield plausible shapes with meaningless values, so it is reported as
// absent rather than returned.
func TestCacheTreatsWrongCanonicalDimAsAbsent(t *testing.T) {
	cache := NewCache(t.TempDir())
	if err := cache.Put(sample("llama", "8b", 4)); err != nil {
		t.Fatalf("put: %v", err)
	}

	got, err := cache.Get("llama", "8b", 8)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got != nil {
		t.Fatal("a connector built against a different K must not be returned")
	}
}

func TestCacheIsKeyedPerModel(t *testing.T) {
	cache := NewCache(t.TempDir())
	if err := cache.Put(sample("llama", "8b", 4)); err != nil {
		t.Fatalf("put 8b: %v", err)
	}
	if err := cache.Put(sample("llama", "70b", 4)); err != nil {
		t.Fatalf("put 70b: %v", err)
	}

	// Same family, different size: both must survive independently, or one
	// model's projection would silently be used for another.
	for _, pc := range []string{"8b", "70b"} {
		if _, err := cache.Get("llama", pc, 4); err != nil {
			t.Fatalf("get %s: %v", pc, err)
		}
	}
}

// An empty connector would bind every module to nothing.
func TestCacheRefusesEmptyConnector(t *testing.T) {
	cache := NewCache(t.TempDir())
	if err := cache.Put(&Connector{Family: "llama", ParamCount: "8b"}); err == nil {
		t.Fatal("an empty connector must be refused")
	}
}

// A family that could escape the cache directory must be rejected: the key
// becomes a filename, so a crafted family could otherwise write anywhere.
func TestCacheRefusesPathTraversalKey(t *testing.T) {
	cache := NewCache(t.TempDir())
	if err := cache.Put(sample("../escape", "x", 2)); err == nil {
		t.Fatal("a key containing a path separator must be refused")
	}
}

func TestCacheMissingFamilyIsRefused(t *testing.T) {
	cache := NewCache(t.TempDir())
	if _, err := cache.Get("", "", 4); err == nil {
		t.Fatal("a connector cannot be identified without a family")
	}
}

// A partially written file must never be readable as a valid connector, which is
// why Put commits via rename.
func TestCacheIgnoresTemporaryFile(t *testing.T) {
	dir := t.TempDir()
	cache := NewCache(dir)

	// A leftover temp file from an interrupted write.
	if err := os.WriteFile(filepath.Join(dir, "llama-8b.safetensors.tmp"), []byte("partial"), 0o644); err != nil {
		t.Fatalf("write temp: %v", err)
	}

	got, err := cache.Get("llama", "8b", 4)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got != nil {
		t.Fatal("an interrupted write must not read as a connector")
	}
}

// The cache satisfies the binder's provider contract, including reporting
// absence as (nil, nil, nil) rather than an error.
func TestCacheImplementsConnectorProvider(t *testing.T) {
	cache := NewCache(t.TempDir())
	spec := api.BaseModelSpec{Family: "llama", ParamCount: "8b", HiddenSize: 3}

	pIn, pOut, err := cache.ConnectorFor(spec, 4)
	if err != nil || pIn != nil || pOut != nil {
		t.Fatalf("absent model should yield (nil, nil, nil), got (%v, %v, %v)", pIn, pOut, err)
	}

	if err := cache.Put(sample("llama", "8b", 4)); err != nil {
		t.Fatalf("put: %v", err)
	}
	pIn, pOut, err = cache.ConnectorFor(spec, 4)
	if err != nil {
		t.Fatalf("connectorFor: %v", err)
	}
	if pIn == nil || pOut == nil {
		t.Fatal("a present hub model must yield its projection")
	}
}
