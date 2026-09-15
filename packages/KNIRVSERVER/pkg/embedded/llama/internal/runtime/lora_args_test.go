package runtime

import (
	"strings"
	"testing"
)

// A DVE can only execute a minted uLoRA if llama-server is started with the
// adapter applied. These pin the exact flag set, because a wrong flag here
// silently produces a model running WITHOUT the adapter — an evaluation that
// looks fine and measures the base model.
func TestArgsLoadsLoRAAdapter(t *testing.T) {
	args := Args("/bin/llama-server", "/models/base.gguf", "127.0.0.1:8080", Options{
		LoRA: []LoRAAdapter{{Path: "/bundles/ulora_cluster-1.gguf"}},
	})

	if got := flagValue(t, args, "--lora"); got != "/bundles/ulora_cluster-1.gguf" {
		t.Fatalf("--lora = %q", got)
	}
	if contains(args, "--lora-scaled") {
		t.Fatal("a zero scale must not produce --lora-scaled")
	}
}

func TestArgsLoadsScaledLoRAAdapter(t *testing.T) {
	args := Args("/bin/llama-server", "/models/base.gguf", "127.0.0.1:8080", Options{
		LoRA: []LoRAAdapter{{Path: "/bundles/a.gguf", Scale: 0.75}},
	})

	idx := indexOf(args, "--lora-scaled")
	if idx < 0 {
		t.Fatalf("expected --lora-scaled, got %v", args)
	}
	if idx+2 >= len(args) || args[idx+1] != "/bundles/a.gguf" || args[idx+2] != "0.75" {
		t.Fatalf("--lora-scaled takes path then scale, got %v", args[idx:])
	}
}

// An empty path must never be emitted: llama-server would consume the next flag
// as the adapter filename and fail at startup.
func TestArgsSkipsEmptyLoRAPath(t *testing.T) {
	args := Args("/bin/llama-server", "/models/base.gguf", "127.0.0.1:8080", Options{
		LoRA: []LoRAAdapter{{Path: "   "}, {Path: "/bundles/b.gguf"}},
	})

	if contains(args, "--lora-scaled") {
		t.Fatalf("empty path with zero scale produced --lora-scaled: %v", args)
	}
	// Only one adapter should have been emitted.
	if n := strings.Count(strings.Join(args, " "), "--lora"); n != 1 {
		t.Fatalf("expected exactly one adapter flag, got %d in %v", n, args)
	}
	if got := flagValue(t, args, "--lora"); got != "/bundles/b.gguf" {
		t.Fatalf("--lora = %q, want the non-empty adapter", got)
	}
}

func TestArgsPreservesAdapterOrderAndBaseFlags(t *testing.T) {
	args := Args("/bin/llama-server", "/models/base.gguf", "127.0.0.1:8080", Options{
		Parallel: 2,
		CtxSize:  4096,
		Threads:  4,
		APIKey:   "secret",
		LoRA: []LoRAAdapter{
			{Path: "/bundles/one.gguf"},
			{Path: "/bundles/two.gguf", Scale: 1.5},
		},
	})

	joined := strings.Join(args, " ")
	for _, want := range []string{
		"-m /models/base.gguf", "--host 127.0.0.1", "--port 8080",
		"--parallel 2", "--ctx-size 4096", "--threads 4", "--api-key secret",
	} {
		if !strings.Contains(joined, want) {
			t.Errorf("missing %q in %v", want, args)
		}
	}
	if indexOf(args, "/bundles/one.gguf") > indexOf(args, "/bundles/two.gguf") {
		t.Fatalf("adapters must keep their order: %v", args)
	}
	// Adapters go after the base-model flags so that argv stays stable.
	if indexOf(args, "--lora") < indexOf(args, "--api-key") {
		t.Fatalf("adapter flags must come after base flags: %v", args)
	}
}

func flagValue(t *testing.T, args []string, flag string) string {
	t.Helper()
	idx := indexOf(args, flag)
	if idx < 0 || idx+1 >= len(args) {
		t.Fatalf("flag %s not found in %v", flag, args)
	}
	return args[idx+1]
}

func indexOf(args []string, want string) int {
	for i, arg := range args {
		if arg == want {
			return i
		}
	}
	return -1
}

func contains(args []string, want string) bool { return indexOf(args, want) >= 0 }
