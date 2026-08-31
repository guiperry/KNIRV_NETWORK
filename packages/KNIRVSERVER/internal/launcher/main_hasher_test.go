package launcher

import (
	"reflect"
	"testing"
)

func TestBackendCommandArgs(t *testing.T) {
	tests := []struct {
		name              string
		configFile        string
		autoStartHasher   bool
		autoStartPipeline bool
		autoStartDirect   bool
		want              []string
	}{
		{name: "no options", want: []string{}},
		{name: "config only", configFile: "/tmp/testnet.yaml", want: []string{"--config", "/tmp/testnet.yaml"}},
		{name: "hasher only", autoStartHasher: true, want: []string{"-hasher"}},
		{name: "pipeline only", autoStartPipeline: true, want: []string{"-pipeline"}},
		{name: "direct only", autoStartDirect: true, want: []string{"-direct"}},
		{name: "hasher and pipeline", autoStartHasher: true, autoStartPipeline: true, want: []string{"-hasher", "-pipeline"}},
		{name: "hasher, pipeline, and direct", autoStartHasher: true, autoStartPipeline: true, autoStartDirect: true, want: []string{"-hasher", "-pipeline", "-direct"}},
		{name: "config and hasher", configFile: "/tmp/testnet.yaml", autoStartHasher: true, want: []string{"--config", "/tmp/testnet.yaml", "-hasher"}},
		{name: "config, hasher, pipeline, and direct", configFile: "/tmp/testnet.yaml", autoStartHasher: true, autoStartPipeline: true, autoStartDirect: true, want: []string{"--config", "/tmp/testnet.yaml", "-hasher", "-pipeline", "-direct"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := backendCommandArgs(tt.configFile, tt.autoStartHasher, tt.autoStartPipeline, tt.autoStartDirect); !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("backendCommandArgs() = %v, want %v", got, tt.want)
			}
		})
	}
}
