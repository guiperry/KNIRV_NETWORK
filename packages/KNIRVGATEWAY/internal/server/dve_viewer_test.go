package server

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/KNIRV/KNIRV_NETWORK/KNIRVGATEWAY/internal/config"
)

func TestGatewayServesDVEVerifierWithoutBackendPageProxy(t *testing.T) {
	gateway := testServer(&config.Config{Port: 8888})
	server := httptest.NewServer(gateway.router)
	defer server.Close()

	response, err := http.Get(server.URL + "/dve/sha256:test/?session=session-1")
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	body, _ := io.ReadAll(response.Body)
	if response.StatusCode != http.StatusOK || !strings.Contains(string(body), "Independent browser verification") {
		t.Fatalf("viewer response = %d %s", response.StatusCode, body)
	}
	if response.Header.Get("Content-Security-Policy") == "" {
		t.Fatal("viewer is missing a content security policy")
	}

	asset, err := http.Get(server.URL + "/dve/_assets/verifier_bg.wasm")
	if err != nil {
		t.Fatal(err)
	}
	defer asset.Body.Close()
	if asset.StatusCode != http.StatusOK {
		t.Fatalf("wasm status = %d", asset.StatusCode)
	}
	if asset.Header.Get("Content-Type") != "application/wasm" {
		t.Fatalf("wasm content type = %q", asset.Header.Get("Content-Type"))
	}
}
