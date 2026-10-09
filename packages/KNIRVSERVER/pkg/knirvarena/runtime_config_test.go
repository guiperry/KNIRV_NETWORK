package knirvarena

import (
	"encoding/json"
	"net/http/httptest"
	"testing"
)

func TestRuntimeConfigDefaultsToTestnet(t *testing.T) {
	for _, tc := range []struct {
		mode, override, network, url string
	}{
		{"", "", "testnet", "https://testnet-gateway.knirv.com"},
		{"testnet", "", "testnet", "https://testnet-gateway.knirv.com"},
		{"production", "", "mainnet", "https://gateway.knirv.com"},
		{"prod", "", "mainnet", "https://gateway.knirv.com"},
		{"testnet", "http://localhost:8082/", "testnet", "http://localhost:8082"},
	} {
		t.Setenv("KNIRV_NETWORK_MODE", tc.mode)
		t.Setenv("KNIRV_ARENA_SERVER_URL", tc.override)
		rec := httptest.NewRecorder()
		handleRuntimeConfig(rec, httptest.NewRequest("GET", "/runtime-config.json", nil))
		var got RuntimeConfig
		if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
			t.Fatal(err)
		}
		if got.Network != tc.network || got.ServerURL != tc.url {
			t.Fatalf("mode %q override %q: got %+v", tc.mode, tc.override, got)
		}
	}
}
