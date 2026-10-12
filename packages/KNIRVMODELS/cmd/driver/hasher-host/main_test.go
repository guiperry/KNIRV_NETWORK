package main

import (
	"path/filepath"
	"testing"
)

func TestResolveGRPCSocket(t *testing.T) {
	tests := []struct {
		name              string
		explicitSocket    string
		knirvserverMode   bool
		wantSocket        string
	}{
		{
			name:           "explicit socket overrides default",
			explicitSocket: "/custom/path.sock",
			wantSocket:     "/custom/path.sock",
		},
		{
			name:            "knirvserver mode uses default socket",
			knirvserverMode: true,
			wantSocket:      filepath.Join(socketDir, "formal-verifier.sock"),
		},
		{
			name:          "no knirvserver mode returns empty",
			wantSocket:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolveGRPCSocket(tt.explicitSocket, tt.knirvserverMode)
			if got != tt.wantSocket {
				t.Errorf("resolveGRPCSocket() = %v, want %v", got, tt.wantSocket)
			}
		})
	}
}
