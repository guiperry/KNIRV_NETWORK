package launcher

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// mockRegistry emulates the REGISTRY.KNIRV.NETWORK worker's /register and
// /lookup endpoints for the txChainAddress field.
func mockRegistry(t *testing.T) (*httptest.Server, map[string]map[string]any) {
	t.Helper()
	store := map[string]map[string]any{}
	mux := http.NewServeMux()
	mux.HandleFunc("/register", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		var payload NodeRegistration
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		node := map[string]any{
			"ip":   payload.IP,
			"port": payload.Port,
			"type": payload.Type,
		}
		if payload.TxChainAddress != "" {
			node["txChainAddress"] = payload.TxChainAddress
		}
		store[payload.ChainID] = node
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{"status": 201})
	})
	mux.HandleFunc("/lookup/", func(w http.ResponseWriter, r *http.Request) {
		chainID := r.URL.Path[len("/lookup/"):]
		node, ok := store[chainID]
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]any{"error": "Node not found"})
			return
		}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(node)
	})
	return httptest.NewServer(mux), store
}

func TestRegisterNodePublishesTxChainAddress(t *testing.T) {
	srv, store := mockRegistry(t)
	defer srv.Close()

	reg := NodeRegistration{
		ChainID:        "node-1",
		IP:             "203.0.113.5",
		Port:           8084,
		Type:           "bootnode",
		TxChainAddress: "0xrootwallet",
	}
	if err := RegisterNode(context.Background(), srv.URL, reg); err != nil {
		t.Fatalf("RegisterNode: %v", err)
	}
	stored, ok := store["node-1"]
	if !ok {
		t.Fatal("node not stored")
	}
	if stored["txChainAddress"] != "0xrootwallet" {
		t.Fatalf("txChainAddress not stored: %v", stored)
	}
}

func TestLookupTxChainAddress(t *testing.T) {
	srv, _ := mockRegistry(t)
	defer srv.Close()

	reg := NodeRegistration{ChainID: "node-2", Port: 8084, Type: "bootnode", TxChainAddress: "0xbootwallet"}
	if err := RegisterNode(context.Background(), srv.URL, reg); err != nil {
		t.Fatalf("RegisterNode: %v", err)
	}

	addr, err := LookupTxChainAddress(context.Background(), srv.URL, "node-2")
	if err != nil {
		t.Fatalf("LookupTxChainAddress: %v", err)
	}
	if addr != "0xbootwallet" {
		t.Fatalf("got %q, want 0xbootwallet", addr)
	}

	// Unknown node returns empty, no error.
	missing, err := LookupTxChainAddress(context.Background(), srv.URL, "ghost")
	if err != nil {
		t.Fatalf("LookupTxChainAddress(missing): %v", err)
	}
	if missing != "" {
		t.Fatalf("expected empty for missing node, got %q", missing)
	}
}
