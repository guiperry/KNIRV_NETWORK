package harness

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// Config.Extra blocks (the Ollama manager's status) appear in /harness/status.
func TestConfigExtraIsMergedIntoStatus(t *testing.T) {
	h := New(Config{FrontSocket: "unused.sock", Extra: func() map[string]any {
		return map[string]any{"ollama": map[string]any{"ready": true, "endpoint": "http://127.0.0.1:11434"}}
	}})
	rec := httptest.NewRecorder()
	h.Router().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/harness/status", nil))
	var doc map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &doc); err != nil {
		t.Fatalf("%v: %s", err, rec.Body.String())
	}
	o, _ := doc["ollama"].(map[string]any)
	if o == nil || o["ready"] != true || doc["primary_note"] == nil {
		t.Fatalf("status %v", doc)
	}
}
