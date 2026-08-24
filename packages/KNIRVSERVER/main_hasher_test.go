package main

import (
	"reflect"
	"testing"
)

func TestBackendCommandArgs(t *testing.T) {
	tests := []struct {
		name            string
		configFile      string
		autoStartHasher bool
		want            []string
	}{
		{name: "no options", want: []string{}},
		{name: "config only", configFile: "/tmp/testnet.yaml", want: []string{"--config", "/tmp/testnet.yaml"}},
		{name: "hasher only", autoStartHasher: true, want: []string{"-hasher"}},
		{name: "config and hasher", configFile: "/tmp/testnet.yaml", autoStartHasher: true, want: []string{"--config", "/tmp/testnet.yaml", "-hasher"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := backendCommandArgs(tt.configFile, tt.autoStartHasher); !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("backendCommandArgs() = %v, want %v", got, tt.want)
			}
		})
	}
}
