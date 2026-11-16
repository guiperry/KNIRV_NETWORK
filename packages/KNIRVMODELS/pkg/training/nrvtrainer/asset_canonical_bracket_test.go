package nrvtrainer

import (
	"encoding/binary"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/knirvcorp/knirvbase/pkg/nrv"
)

// writeCanonicalWithBrackets writes a canonical NRV container whose single
// frame declares the given bracket count and length, and whose bracket bytes
// are a valid 80-byte-aligned block of `actual` bytes.
func writeCanonicalWithBrackets(t *testing.T, count, length int, actual []byte) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "canonical.nrv")
	bracketOffset := int64(nrv.HeaderSize + nrv.RegistryPadding)

	registryJSON, err := json.Marshal(map[string]any{
		"version":    1,
		"dataset_id": "dataset-bracket",
		"frames": []any{map[string]any{
			"id": "f1",
			"z3": map[string]any{"status": "VALID"},
			"brackets": map[string]any{
				"count": count, "offset": bracketOffset, "length": length,
			},
		}},
		"pqc_manifest": map[string]any{"key_id": "k", "algorithm": "ML-DSA-65"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(registryJSON) > nrv.RegistryPadding {
		t.Fatal("registry too large for test")
	}

	out := make([]byte, nrv.HeaderSize+nrv.RegistryPadding, int(bracketOffset)+len(actual))
	binary.LittleEndian.PutUint32(out[0:4], nrv.Magic)
	binary.LittleEndian.PutUint32(out[4:8], 1)
	copy(out[nrv.HeaderSize:], registryJSON)
	out = append(out, actual...)
	binary.LittleEndian.PutUint32(out[8:12], uint32(len(out)))
	if err := os.WriteFile(path, out, 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestCanonicalBracketGeometryEnforced(t *testing.T) {
	valid := testBracket(t, 3)

	t.Run("valid count*80 passes", func(t *testing.T) {
		path := writeCanonicalWithBrackets(t, 1, 80, valid)
		asset, err := LoadAsset(path)
		if err != nil {
			t.Fatalf("LoadAsset valid bracket: %v", err)
		}
		if len(asset.Frames) != 1 || len(asset.Frames[0].RawBrackets) != bracketSize {
			t.Fatalf("frames = %#v", asset.Frames)
		}
	})

	malformed := map[string]struct {
		count, length int
		actual        []byte
	}{
		"length not multiple of 80": {1, 79, valid[:79]},
		"length larger than count":  {1, 160, append(append([]byte{}, valid...), valid...)},
		"length smaller than count": {2, 80, valid},
		"negative count":            {-1, 0, nil},
		"negative length":           {0, -80, nil},
		"count without length":      {1, 0, nil},
	}
	for name, tc := range malformed {
		t.Run(name, func(t *testing.T) {
			path := writeCanonicalWithBrackets(t, tc.count, tc.length, tc.actual)
			_, err := LoadAsset(path)
			if err == nil {
				t.Fatalf("LoadAsset accepted malformed bracket count=%d length=%d", tc.count, tc.length)
			}
			if !strings.Contains(err.Error(), "bracket") {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}
