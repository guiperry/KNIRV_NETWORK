package launcher

import (
	"reflect"
	"testing"
)

func TestLlamaHarnessEnvPassesOnlyExplicitSettings(t *testing.T) {
	if got := llamaHarnessEnv(LlamaConfig{}); len(got) != 0 {
		t.Fatalf("defaults must stay the harness's own: %v", got)
	}
	got := llamaHarnessEnv(LlamaConfig{Parallel: 2, CtxSize: 4096, Threads: 6, ModelURL: "https://x/m.gguf", ModelName: "m.gguf", ModelPath: "/m/m.gguf"})
	want := []string{
		"KNIRV_LLAMA_PARALLEL=2", "KNIRV_LLAMA_CTX_SIZE=4096", "KNIRV_LLAMA_THREADS=6",
		"KNIRV_LLAMA_MODEL_URL=https://x/m.gguf", "KNIRV_LLAMA_MODEL_NAME=m.gguf", "KNIRV_LLAMA_MODEL_PATH=/m/m.gguf",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("%v", got)
	}
}
