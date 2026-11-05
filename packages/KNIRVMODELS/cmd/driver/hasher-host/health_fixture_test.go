package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestHealthResponseMatchesSharedFixture(t *testing.T) {
	path := filepath.Join("..", "..", "..", "..", "KNIRVSDK", "testvectors", "models", "hasher_host_health.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var want map[string]any
	if err := json.Unmarshal(raw, &want); err != nil {
		t.Fatal(err)
	}
	actual, err := json.Marshal(HealthResponse{Status: "healthy", Uptime: "0s", ConnectionHealthy: true, DirectMode: true})
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal(actual, &got); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("health = %v, want %v", got, want)
	}
}
