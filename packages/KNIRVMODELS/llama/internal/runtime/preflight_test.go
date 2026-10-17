package runtime

import (
	"context"
	"net"
	"os"
	"strings"
	"testing"
	"time"
)

func fakeBinary(t *testing.T) string {
	t.Helper()
	p := t.TempDir() + "/llama-server"
	if err := os.WriteFile(p, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestPreflightHealthyConfigIsOK(t *testing.T) {
	memAvailable = func() (int64, error) { return 8 << 30, nil }
	defer func() { memAvailable = readMemAvailable }()
	r := Preflight(fakeBinary(t), writeGGUF(t, 4096), freeAddr(t))
	if !r.OK() || len(r.Warnings) != 0 {
		t.Fatalf("%+v", r)
	}
	if r.ModelBytes != 4096 {
		t.Fatalf("model bytes %d", r.ModelBytes)
	}
}

func TestPreflightCatchesBrokenInputs(t *testing.T) {
	bad := t.TempDir() + "/bad.gguf"
	os.WriteFile(bad, []byte("not a model"), 0o644)
	for name, c := range map[string]struct{ bin, model, want string }{
		"missing model": {fakeBinary(t), "/nonexistent.gguf", "not readable"},
		"truncated":     {fakeBinary(t), bad, "not a GGUF"},
		"missing bin":   {"/nonexistent/llama-server", writeGGUF(t, 64), "llama-server binary"},
	} {
		r := Preflight(c.bin, c.model, freeAddr(t))
		if r.OK() || !strings.Contains(strings.Join(r.Errors, "|"), c.want) {
			t.Fatalf("%s: %+v", name, r)
		}
	}
}

func TestPreflightWarnsWhenMemoryIsShort(t *testing.T) {
	memAvailable = func() (int64, error) { return 1 << 10, nil }
	defer func() { memAvailable = readMemAvailable }()
	r := Preflight(fakeBinary(t), writeGGUF(t, 1<<20), freeAddr(t))
	if !r.OK() || len(r.Warnings) != 1 || !strings.Contains(r.Warnings[0], "slow") {
		t.Fatalf("%+v", r)
	}
}

func TestPreflightRejectsPortHeldByAnotherProcess(t *testing.T) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Skip(err)
	}
	defer l.Close()
	go func() { // accepts and never speaks HTTP, like a non-llama service
		for {
			c, err := l.Accept()
			if err != nil {
				return
			}
			defer c.Close()
		}
	}()
	r := Preflight(fakeBinary(t), writeGGUF(t, 64), l.Addr().String())
	if r.OK() || !strings.Contains(strings.Join(r.Errors, "|"), "already in use") {
		t.Fatalf("%+v", r)
	}
}

func TestWarmFileReadsWholeFileAndStopsOnCancel(t *testing.T) {
	memAvailable = func() (int64, error) { return 64 << 30, nil }
	defer func() { memAvailable = readMemAvailable }()
	p := writeGGUF(t, 9<<20)
	n, err := WarmFile(context.Background(), p)
	if err != nil || n != 9<<20 {
		t.Fatalf("n=%d err=%v", n, err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if n, err := WarmFile(ctx, p); err == nil || n == 9<<20 {
		t.Fatalf("cancelled warm must stop early: n=%d err=%v", n, err)
	}
}

func TestWarmFileSkipsWhatWouldNotFit(t *testing.T) {
	memAvailable = func() (int64, error) { return 1 << 20, nil }
	defer func() { memAvailable = readMemAvailable }()
	n, err := WarmFile(context.Background(), writeGGUF(t, 9<<20))
	if err != nil || n != 0 {
		t.Fatalf("n=%d err=%v", n, err)
	}
	_ = time.Second
}
