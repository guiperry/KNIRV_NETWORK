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

	err := registerWithRegistry(registry.URL, "test-chain", 4001, "203.0.113.1", "peer-id", "server", "", 0)
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

	if err := registerWithRegistry(registry.URL, "chain-1", 4001, "203.0.113.1", "peer-1", "bootnode", "gw-1.knirv.com", 0); err != nil {
		t.Fatalf("register: %v", err)
	}
	if got.FQDN != "gw-1.knirv.com" || got.Type != "bootnode" {
		t.Fatalf("payload = %+v, want fqdn gw-1.knirv.com and type bootnode", got)
	}
}

func TestRegisterWithRegistrySendsGatewayPort(t *testing.T) {
	cases := []struct {
		name        string
		gatewayPort int
		wantInBody  bool
		wantPort    int
	}{
		{"advertised", 8443, true, 8443},
		{"max port", 65535, true, 65535},
		{"unset omits field", 0, false, 0},
		{"out of range is not sent", 70000, false, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var raw map[string]interface{}
			registry := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				raw = map[string]interface{}{}
				_ = json.NewDecoder(r.Body).Decode(&raw)
				w.WriteHeader(http.StatusCreated)
			}))
			defer registry.Close()

			if err := registerWithRegistry(registry.URL, "chain-1", 4001, "203.0.113.1", "peer-1", "bootnode", "gw-1.knirv.com", tc.gatewayPort); err != nil {
				t.Fatalf("register: %v", err)
			}
			v, present := raw["gatewayPort"]
			if present != tc.wantInBody {
				t.Fatalf("gatewayPort present=%v, want %v (body=%v)", present, tc.wantInBody, raw)
			}
			if present && int(v.(float64)) != tc.wantPort {
				t.Fatalf("gatewayPort=%v, want %d", v, tc.wantPort)
			}
			if got := raw["port"]; int(got.(float64)) != 4001 {
				t.Fatalf("p2p port changed: %v, want 4001", got)
			}
		})
	}
}

func TestDHTManagerRegistryGatewayPort(t *testing.T) {
	cases := map[int]int{0: 0, -1: 0, 1: 1, 8443: 8443, 65535: 65535, 65536: 0}
	for in, want := range cases {
		dm := &DHTManager{gatewayPort: in}
		if got := dm.registryGatewayPort(); got != want {
			t.Errorf("registryGatewayPort(%d) = %d, want %d", in, got, want)
		}
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
