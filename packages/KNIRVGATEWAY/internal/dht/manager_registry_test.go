package dht

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRegisterWithRegistryDoesNotFollowMethodChangingRedirect(t *testing.T) {
	var methods []string
	registry := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		methods = append(methods, r.Method)
		if r.URL.Path == "/register" {
			http.Redirect(w, r, "/canonical-register", http.StatusFound)
			return
		}
		w.WriteHeader(http.StatusMethodNotAllowed)
	}))
	defer registry.Close()

	err := registerWithRegistry(registry.URL, "test-chain", 4001, "203.0.113.1", "peer-id", "server")
	if err == nil || !strings.Contains(err.Error(), "redirected registration request") {
		t.Fatalf("expected redirect configuration error, got %v", err)
	}
	if got, want := strings.Join(methods, ","), http.MethodPost; got != want {
		t.Fatalf("registry saw methods %q; want only %q", got, want)
	}
}
