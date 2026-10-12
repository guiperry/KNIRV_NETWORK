package headless

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// The backend reaches the HEAD over the daemon's socket, so the daemon's mux must carry /heart/head/*.
func TestDaemonServesHEADRoutes(t *testing.T) {
	t.Setenv("KNIRV_APP_DATA_DIR", t.TempDir())
	s := NewServer(t.TempDir()+"/h.sock", 0o660)
	for _, p := range []string{"/heart/head/models", "/heart/head/matrix"} {
		rec := httptest.NewRecorder()
		s.mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, p, nil))
		if rec.Code == http.StatusNotFound && rec.Body.String() == "404 page not found\n" {
			t.Fatalf("%s is not routed", p)
		}
	}
	rec := httptest.NewRecorder()
	s.mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/heart/head/models", nil))
	if rec.Code != http.StatusServiceUnavailable { // routed, but no model published yet
		t.Fatalf("got %d: %s", rec.Code, rec.Body)
	}
}
