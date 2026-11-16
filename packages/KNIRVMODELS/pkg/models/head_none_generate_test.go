package models

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"knirvmodels/pkg/vlm"
)

// TestHeadGenerateNoneReturnsStructured503 pins the vlm=none contract for the
// mirrored text generator: POST /models/head/generate with no backend must return a
// structured 503 whose body is a JSON error object, and must never carry generated
// text. It drives the real mounted Head through an in-process mux (no sockets).
func TestHeadGenerateNoneReturnsStructured503(t *testing.T) {
	h := NewHead(vlm.NoneBackend{})
	mux := http.NewServeMux()
	h.Mount(mux, "/models/head")

	req := httptest.NewRequest(http.MethodPost, "/models/head/generate",
		strings.NewReader(`{"prompt":"hello"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d (body %q)", rec.Code, http.StatusServiceUnavailable, rec.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("503 body is not a JSON object: %v (%q)", err, rec.Body.String())
	}
	if msg, _ := body["error"].(string); msg == "" {
		t.Fatalf("503 body lacks a non-empty error field: %v", body)
	}
	if _, ok := body["text"]; ok {
		t.Fatalf("503 body must not carry generated text: %v", body)
	}
}
