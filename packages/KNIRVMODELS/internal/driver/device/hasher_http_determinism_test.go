package device

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync"
	"testing"

	"knirvmodels/pkg/hashing/methods/software"
	"knirvmodels/pkg/hashing/transformer"
	"knirvmodels/pkg/models"
)

// hashDevice exposes the real device guard to the model and counts any use.
// The current transformer path must not depend on the device at all.
type hashDevice struct {
	*software.SoftwareMethod
	device *Device
	calls  int
}

func (h *hashDevice) ComputeBatch(inputs [][]byte) ([][32]byte, error) {
	h.calls++
	out := make([][32]byte, len(inputs))
	for i, input := range inputs {
		var err error
		out[i], err = h.device.ComputeHash(input)
		if err != nil {
			return nil, err
		}
	}
	return out, nil
}

func TestHasherHTTPDirectModeMatchesSoftware(t *testing.T) {
	cfg := &transformer.UnifiedConfig{VocabSize: 64, EmbedDim: 16, NumHeads: 2, NumLayers: 1, ContextLen: 8, Hidden1: 8, Hidden2: 4, OutputSize: 4, FFNHiddenDim: 32, Activation: "hash", Passes: 3}
	seeds := transformer.BuildDefaultSeedStore(cfg)
	var results [2][]map[string]any
	for mode := 0; mode < 2; mode++ {
		method := &hashDevice{SoftwareMethod: software.NewSoftwareMethod(), device: &Device{useDirectASIC: mode == 1, stats: &DeviceStats{}, mu: sync.RWMutex{}}}
		if err := method.Initialize(); err != nil {
			t.Fatal(err)
		}
		mux := http.NewServeMux()
		models.NewHasher(func() (*transformer.UnifiedHasherEngine, error) {
			return transformer.NewUnifiedHasherEngineWithConfig(cfg, seeds, method, transformer.ModeTransformer), nil
		}).Mount(mux, "/models/hasher")
		for _, request := range []struct{ path, body string }{
			{"/models/hasher/validate", `{"prediction":1,"domain":"nope"}`},
			{"/models/hasher/infer", `{"text":"determinism"}`},
		} {
			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, request.path, strings.NewReader(request.body)))
			if rec.Code != http.StatusOK {
				t.Fatalf("direct=%t %s: %d %s", mode == 1, request.path, rec.Code, rec.Body)
			}
			var body map[string]any
			if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
				t.Fatal(err)
			}
			// Wall-clock timing is intentionally not a deterministic response field.
			delete(body, "latency_ms")
			results[mode] = append(results[mode], body)
		}
		if method.calls != 0 {
			t.Fatalf("transformer inference unexpectedly used the device %d times", method.calls)
		}
	}
	if !reflect.DeepEqual(results[0], results[1]) {
		t.Fatalf("direct off: %#v\ndirect on: %#v", results[0], results[1])
	}
}
