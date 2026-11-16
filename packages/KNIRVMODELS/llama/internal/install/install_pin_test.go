package install

import (
	"context"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// The llama.cpp build is pinned: install and update target PinnedLlamaTag and never
// ask upstream which release is newest. These tests use only the version and URL
// construction; every network function is a stub that fails the test if reached.

func TestPinnedLlamaTagIsB12388(t *testing.T) {
	if PinnedLlamaTag != "b12388" {
		t.Fatalf("PinnedLlamaTag = %q, want b12388", PinnedLlamaTag)
	}
	if pinnedTag != PinnedLlamaTag {
		t.Fatalf("pinnedTag = %q, want the shipped pin %q", pinnedTag, PinnedLlamaTag)
	}
	if n, ok := parseTagBuild(PinnedLlamaTag); !ok || n != 12388 {
		t.Fatalf("parseTagBuild(%q) = %d, %v; want 12388, true", PinnedLlamaTag, n, ok)
	}
}

func TestPinnedBuildMakesNoUpstreamRequest(t *testing.T) {
	i := New()
	i.Get = func(url string) (*http.Response, error) {
		t.Fatalf("pinned build must not consult upstream, got GET %s", url)
		return nil, nil
	}
	tag, build, err := i.pinnedBuild()
	if err != nil {
		t.Fatalf("pinnedBuild: %v", err)
	}
	if tag != "b12388" || build != 12388 {
		t.Fatalf("pinnedBuild = %q, %d; want b12388, 12388", tag, build)
	}
}

func TestPinnedBuildRejectsNonBuildTag(t *testing.T) {
	pinUpstream(t, "v0.8.0")
	if _, _, err := New().pinnedBuild(); err == nil {
		t.Fatal("a non-build tag must not be accepted as the pin")
	}
}

func TestFreshInstallFallbackClonesPinnedTag(t *testing.T) {
	// Legacy server locations are under $HOME: point it at an empty dir so a host
	// llama-server cannot satisfy the install.
	t.Setenv("HOME", t.TempDir())
	dataDir := t.TempDir()
	model := filepath.Join(dataDir, "model.gguf")
	if err := os.WriteFile(model, []byte("m"), 0644); err != nil {
		t.Fatal(err)
	}
	var calls [][]string
	i := New()
	// Prebuilt download fails (no network): the install falls back to a source build.
	i.Get = func(url string) (*http.Response, error) {
		return nil, errors.New("network disabled in tests")
	}
	i.LookPath = func(string) (string, error) { return "/usr/bin/x", nil }
	i.Run = func(_ context.Context, name string, args ...string) error {
		calls = append(calls, append([]string{name}, args...))
		// Stop after the clone: configure/build are not run in tests.
		return errors.New("stop after clone")
	}
	_, err := i.Ensure(context.Background(), Options{
		DataDir: dataDir, ServerURL: "http://127.0.0.1:9/llama-server.gz",
		ModelPath: model, ModelName: "model.gguf",
	})
	if err == nil {
		t.Fatal("Ensure should fail at the stubbed clone")
	}
	var clone []string
	for _, c := range calls {
		if len(c) > 1 && c[0] == "git" && c[1] == "clone" {
			clone = c
		}
	}
	if clone == nil {
		t.Fatalf("no git clone was attempted; calls = %v", calls)
	}
	want := []string{"git", "clone", "--depth", "1", "--branch", "b12388", upstreamRepoURL, filepath.Join(dataDir, "llama.cpp")}
	if !reflect.DeepEqual(clone, want) {
		t.Fatalf("clone args = %v\nwant          %v", clone, want)
	}
	if !strings.Contains(strings.Join(clone, " "), "--branch b12388") {
		t.Fatal("fresh source build is not pinned to b12388")
	}
}

func TestUpdateServerTargetsPinnedBuildAndNeverDowngrades(t *testing.T) {
	// Installed build is older than the pin: the update targets exactly b12388 and
	// never asks upstream for a newer tag.
	dir := t.TempDir()
	server := filepath.Join(dir, "llama-server")
	if err := os.WriteFile(server, []byte("x"), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", t.TempDir())
	i := New()
	i.Get = func(url string) (*http.Response, error) {
		if strings.Contains(url, "api.github.com") {
			t.Fatalf("no upstream release lookup expected, got GET %s", url)
		}
		return nil, errors.New("network disabled in tests")
	}
	i.Output = func(context.Context, string, ...string) (string, error) { return "version: 12000 (abc)\n", nil }
	var cloned []string
	i.Run = func(_ context.Context, name string, args ...string) error {
		if name == "git" && len(args) > 0 && args[0] == "clone" {
			cloned = append([]string{name}, args...)
		}
		return errors.New("stop after clone")
	}
	i.LookPath = func(string) (string, error) { return "/usr/bin/x", nil }
	i.updateServer(context.Background(), Options{DataDir: dir, ServerURL: "http://127.0.0.1:9/llama-server.gz"}, server)
	if cloned == nil {
		t.Fatal("older build should be replaced by a build of the pinned tag")
	}
	if !strings.Contains(strings.Join(cloned, " "), "--branch b12388") {
		t.Fatalf("update clone args = %v; want --branch b12388", cloned)
	}

	// Installed build is newer than the pin: kept, nothing runs.
	i2 := New()
	i2.Get = func(url string) (*http.Response, error) {
		if strings.Contains(url, "api.github.com") {
			t.Fatalf("no upstream release lookup expected, got GET %s", url)
		}
		return nil, errors.New("network disabled in tests")
	}
	i2.Output = func(context.Context, string, ...string) (string, error) { return "version: 13000 (abc)\n", nil }
	i2.Run = func(context.Context, string, ...string) error {
		t.Fatal("a newer installed build must not be downgraded or rebuilt")
		return nil
	}
	if got := i2.updateServer(context.Background(), Options{DataDir: dir}, server); got != server {
		t.Fatalf("server = %s, want %s kept", got, server)
	}
}
