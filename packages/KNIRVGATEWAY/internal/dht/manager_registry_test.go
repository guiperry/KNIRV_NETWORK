package dht

import (
	"encoding/json"
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

	err := registerWithRegistry(registry.URL, "test-chain", 4001, "203.0.113.1", "peer-id", "server", "")
	if err == nil || !strings.Contains(err.Error(), "redirected registration request") {
		t.Fatalf("expected redirect configuration error, got %v", err)
	}
	if got, want := strings.Join(methods, ","), http.MethodPost; got != want {
		t.Fatalf("registry saw methods %q; want only %q", got, want)
	}
}

func TestRegisterWithRegistrySendsFQDN(t *testing.T) {
	var got registrationPayload
	registry := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&got)
		w.WriteHeader(http.StatusCreated)
	}))
	defer registry.Close()

	if err := registerWithRegistry(registry.URL, "chain-1", 4001, "203.0.113.1", "peer-1", "bootnode", "gw-1.knirv.com"); err != nil {
		t.Fatalf("register: %v", err)
	}
	if got.FQDN != "gw-1.knirv.com" || got.Type != "bootnode" {
		t.Fatalf("payload = %+v, want fqdn gw-1.knirv.com and type bootnode", got)
	}
}

func TestRegistryFQDNSkipsIPsAndPlaceholders(t *testing.T) {
	cases := map[string]string{
		"gw-1.knirv.com": "gw-1.knirv.com",
		"GW-1.KNIRV.com": "gw-1.knirv.com",
		"203.0.113.1":    "",
		"localhost":      "",
		"":               "",
		"nodots":         "",
	}
	for host, want := range cases {
		dm := &DHTManager{publicHost: host}
		if got := dm.registryFQDN(); got != want {
			t.Errorf("registryFQDN(%q) = %q, want %q", host, got, want)
		}
	}
}
