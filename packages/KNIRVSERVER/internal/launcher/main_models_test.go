package launcher

import (
	"reflect"
	"testing"
)

func TestBackendCommandArgs(t *testing.T) {
	tests := []struct {
		name              string
		configFile        string
		autoStartModels   bool
		autoStartPipeline bool
		autoStartDirect   bool
		autoStartLlama    bool
		llamaSocket       string
		want              []string
	}{
		{name: "no options", want: []string{}},
		{name: "config only", configFile: "/tmp/testnet.yaml", want: []string{"--config", "/tmp/testnet.yaml"}},
		{name: "models only", autoStartModels: true, want: []string{"-models"}},
		{name: "pipeline only", autoStartPipeline: true, want: []string{"-pipeline"}},
		{name: "direct only", autoStartDirect: true, want: []string{"-direct"}},
		{name: "models and pipeline", autoStartModels: true, autoStartPipeline: true, want: []string{"-models", "-pipeline"}},
		{name: "models, pipeline, and direct", autoStartModels: true, autoStartPipeline: true, autoStartDirect: true, want: []string{"-models", "-pipeline", "-direct"}},
		{name: "config and models", configFile: "/tmp/testnet.yaml", autoStartModels: true, want: []string{"--config", "/tmp/testnet.yaml", "-models"}},
		{name: "config, models, pipeline, and direct", configFile: "/tmp/testnet.yaml", autoStartModels: true, autoStartPipeline: true, autoStartDirect: true, want: []string{"--config", "/tmp/testnet.yaml", "-models", "-pipeline", "-direct"}},
		{name: "llama only", autoStartLlama: true, llamaSocket: "/var/lib/knirvserver/sockets/llama.sock", want: []string{"-llama", "-llama-socket", "/var/lib/knirvserver/sockets/llama.sock"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := backendCommandArgs(tt.configFile, tt.autoStartModels, tt.autoStartPipeline, tt.autoStartDirect, tt.autoStartLlama, tt.llamaSocket); !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("backendCommandArgs() = %v, want %v", got, tt.want)
			}
		})
	}
}
