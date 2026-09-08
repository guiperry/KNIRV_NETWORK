package runtime

import (
	"strconv"
	"testing"
)

func TestArgsBuildsExpectedFlagSet(t *testing.T) {
	got := Args("./llama-server", "/models/tiny.gguf", "127.0.0.1:8000", Options{
		Parallel: 1,
		CtxSize:  4096,
		Threads:  4,
		APIKey:   "s3cr3t",
	})

	want := []string{
		"./llama-server",
		"-m", "/models/tiny.gguf",
		"--host", "127.0.0.1",
		"--port", "8000",
		"--parallel", "1",
		"--ctx-size", "4096",
		"--threads", "4",
		"--api-key", "s3cr3t",
	}

	if len(got) != len(want) {
		t.Fatalf("Args() = %#v, want %#v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("Args()[%d] = %q, want %q (full = %#v)", i, got[i], want[i], got)
		}
	}
}

func TestArgsZeroOptionsOmitsOptionalFlags(t *testing.T) {
	got := Args("./llama-server", "m.gguf", "127.0.0.1:8000", Options{})

	// Default behaviour (kept compatible with pre-Phase-A callers): no
	// --parallel / --ctx-size / --threads / --api-key flags at all. The
	// manager layer is responsible for setting Parallel to 1 explicitly.
	for _, arg := range got {
		switch arg {
		case "--parallel", "--ctx-size", "--threads", "--api-key":
			t.Fatalf("Args() must not emit %q when Options{} is supplied: %#v", arg, got)
		}
	}
}

func TestArgsHandlesHostPortExtraction(t *testing.T) {
	got := Args("./llama-server", "m.gguf", "127.0.0.1:12345", Options{Parallel: 1})
	found := false
	for i, a := range got {
		if a == "--port" && i+1 < len(got) && got[i+1] == "12345" {
			found = true
		}
	}
	if !found {
		t.Fatalf("Args() did not derive port from address: %#v", got)
	}
}

func TestOptionsArgsStringConversion(t *testing.T) {
	o := Options{Parallel: 2, CtxSize: 8192, Threads: 8}
	got := o.args("m", strconv.Itoa(8000))
	mustHave := []string{"--parallel", "2", "--ctx-size", "8192", "--threads", "8"}
	for _, want := range mustHave {
		ok := false
		for _, a := range got {
			if a == want {
				ok = true
				break
			}
		}
		if !ok {
			t.Fatalf("args %#v missing %q", got, want)
		}
	}
}
