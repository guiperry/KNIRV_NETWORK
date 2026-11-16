package install

import (
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
)

func TestDownloadServerExtractsGzipArtifact(t *testing.T) {
	var compressed bytes.Buffer
	writer := gzip.NewWriter(&compressed)
	if _, err := writer.Write([]byte("server-binary")); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}

	i := New()
	i.Get = func(string) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, Status: "200 OK", Body: io.NopCloser(bytes.NewReader(compressed.Bytes()))}, nil
	}
	i.Run = func(context.Context, string, ...string) error { return nil }
	path, err := i.downloadServer(context.Background(), t.TempDir(), "https://releases.knirv.com/knirv/llama/linux-amd64/llama-server.gz", "llama-server")
	if err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(path)
	if err != nil || string(got) != "server-binary" {
		t.Fatalf("server = %q, %v", got, err)
	}
	info, err := os.Stat(path)
	if err != nil || info.Mode()&0111 == 0 {
		t.Fatalf("server is not executable: %v, %v", info, err)
	}
}

func TestEnsureUsesProvidedFilesWithoutInstalling(t *testing.T) {
	dir := t.TempDir()
	server := dir + "/llama-server"
	model := dir + "/model.gguf"
	for _, path := range []string{server, model} {
		if err := os.WriteFile(path, []byte("x"), 0700); err != nil {
			t.Fatal(err)
		}
	}
	i := New()
	i.Run = func(context.Context, string, ...string) error { t.Fatal("installer should not run"); return nil }
	got, err := i.Ensure(context.Background(), Options{DataDir: dir, ServerPath: server, ModelPath: model, NoInstall: true})
	if err != nil {
		t.Fatal(err)
	}
	if got.ServerPath != server || got.ModelPath != model {
		t.Fatalf("unexpected result: %#v", got)
	}
}

func TestEnsureDownloadsMissingModel(t *testing.T) {
	dir := t.TempDir()
	server := dir + "/llama-server"
	if err := os.WriteFile(server, []byte("x"), 0700); err != nil {
		t.Fatal(err)
	}
	i := New()
	i.Get = func(string) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Status: "200 OK", Body: io.NopCloser(strings.NewReader("model"))}, nil
	}
	got, err := i.Ensure(context.Background(), Options{DataDir: dir, ServerPath: server, ModelName: "test.gguf", ModelURL: "https://example.test/model"})
	if err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(got.ModelPath)
	if err != nil || string(b) != "model" {
		t.Fatalf("model = %q, %v", b, err)
	}
}

func releaseResponder(tag string) func(string) (*http.Response, error) {
	return func(url string) (*http.Response, error) {
		if url == LatestReleaseURL {
			return &http.Response{StatusCode: 200, Status: "200 OK", Body: io.NopCloser(strings.NewReader(`[{"tag_name":"v0.8.0"},{"tag_name":"` + tag + `"}]`))}, nil
		}
		return nil, io.ErrUnexpectedEOF
	}
}

func TestUpdateServerKeepsCurrentBuild(t *testing.T) {
	dir := t.TempDir()
	server := dir + "/llama-server"
	if err := os.WriteFile(server, []byte("x"), 0700); err != nil {
		t.Fatal(err)
	}
	i := New()
	i.Get = releaseResponder("b6000")
	i.Output = func(context.Context, string, ...string) (string, error) { return "version: 6000 (abc)\n", nil }
	i.Run = func(context.Context, string, ...string) error { t.Fatal("up to date: nothing should run"); return nil }
	if got := i.updateServer(context.Background(), Options{DataDir: dir}, server); got != server {
		t.Fatalf("server = %s", got)
	}
}

func TestUpdateServerKeepsWorkingBinaryWhenUpdateFails(t *testing.T) {
	dir := t.TempDir()
	server := dir + "/llama-server"
	if err := os.WriteFile(server, []byte("x"), 0700); err != nil {
		t.Fatal(err)
	}
	i := New()
	i.Get = releaseResponder("b6100") // the prebuilt download then fails
	i.Output = func(context.Context, string, ...string) (string, error) { return "version: 6000 (abc)\n", nil }
	i.Run = func(context.Context, string, ...string) error { return io.ErrUnexpectedEOF }
	i.LookPath = func(string) (string, error) { return "/usr/bin/x", nil }
	if got := i.updateServer(context.Background(), Options{DataDir: dir}, server); got != server {
		t.Fatalf("server = %s", got)
	}
	if _, err := os.Stat(server); err != nil {
		t.Fatalf("working server was removed: %v", err)
	}
}

func TestUpdateServerInstallsNewerPrebuilt(t *testing.T) {
	dir := t.TempDir()
	server := dir + "/llama-server"
	if err := os.WriteFile(server, []byte("old"), 0700); err != nil {
		t.Fatal(err)
	}
	i := New()
	i.Get = func(url string) (*http.Response, error) {
		if url == LatestReleaseURL {
			return releaseResponder("b6100")(url)
		}
		return &http.Response{StatusCode: 200, Status: "200 OK", Body: io.NopCloser(strings.NewReader("new"))}, nil
	}
	i.Output = func(_ context.Context, path string, _ ...string) (string, error) {
		if strings.HasSuffix(path, ".update") {
			return "version: 6100 (def)\n", nil
		}
		return "version: 6000 (abc)\n", nil
	}
	i.Run = func(context.Context, string, ...string) error { return nil }
	got := i.updateServer(context.Background(), Options{DataDir: dir, ServerURL: "https://example.test/llama-server"}, server)
	if want := dir + "/bin/llama-server"; got != want {
		t.Fatalf("server = %s, want %s", got, want)
	}
	if b, _ := os.ReadFile(got); string(b) != "new" {
		t.Fatalf("binary = %q", b)
	}
}

// Real `llama-server --version` output: a git checkout build (7780, verified on
// this host from ~/models/llama.cpp) and a tarball build (b12388 release archive,
// no git metadata, so build 0).
const (
	gitBuildVersion     = "version: 7780 (7dee9ff59)\nbuilt with GNU 11.4.0 for Linux x86_64\n"
	tarballBuildVersion = "version: 1.0.0-dev (build 0, commit unknown)\nbuilt with GNU 11.4.0 for Linux x86_64\n"
)

func TestParseBuild(t *testing.T) {
	tests := []struct {
		name    string
		out     string
		want    int
		unknown bool
		wantErr bool
	}{
		{name: "git checkout build", out: gitBuildVersion, want: 7780},
		{name: "git build, 5-digit build", out: "version: 12388 (abcdef012)\n", want: 12388},
		{name: "tarball build is not 1", out: tarballBuildVersion, unknown: true},
		{name: "tarball build, build 0 without commit", out: "version: 1.0.0-dev (build 0)\n", unknown: true},
		{name: "build-tagged nonzero", out: "version: 1.0.0-dev (build 12388, commit abc)\n", want: 12388},
		{name: "no version line", out: "usage: llama-server [options]\n", wantErr: true},
		{name: "empty output", out: "", wantErr: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseBuild(tc.out)
			if tc.unknown {
				if !errors.Is(err, errUnknownBuild) {
					t.Fatalf("err = %v, want errUnknownBuild", err)
				}
				return
			}
			if tc.wantErr {
				if err == nil || errors.Is(err, errUnknownBuild) {
					t.Fatalf("err = %v, want a plain parse error", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Fatalf("build = %d, want %d", got, tc.want)
			}
		})
	}
}

func TestInstalledBuildRealOutputs(t *testing.T) {
	i := New()
	i.Output = func(context.Context, string, ...string) (string, error) { return tarballBuildVersion, nil }
	if _, err := i.installedBuild(context.Background(), "/x/llama-server"); !errors.Is(err, errUnknownBuild) {
		t.Fatalf("tarball: err = %v, want errUnknownBuild", err)
	}
	i.Output = func(context.Context, string, ...string) (string, error) { return gitBuildVersion, nil }
	got, err := i.installedBuild(context.Background(), "/x/llama-server")
	if err != nil || got != 7780 {
		t.Fatalf("git: got %d, %v; want 7780", got, err)
	}
}

// A tarball build must not trigger a download or source build on every start.
func TestUpdateServerKeepsUnknownBuildWithoutRebuilding(t *testing.T) {
	dir := t.TempDir()
	server := dir + "/llama-server"
	if err := os.WriteFile(server, []byte("x"), 0700); err != nil {
		t.Fatal(err)
	}
	i := New()
	i.Get = func(url string) (*http.Response, error) {
		if url != LatestReleaseURL {
			t.Fatalf("unexpected fetch of %s for an unknown build", url)
		}
		return releaseResponder("b12388")(url)
	}
	i.Output = func(context.Context, string, ...string) (string, error) { return tarballBuildVersion, nil }
	i.Run = func(context.Context, string, ...string) error {
		t.Fatal("unknown build: nothing should run")
		return nil
	}
	if got := i.updateServer(context.Background(), Options{DataDir: dir}, server); got != server {
		t.Fatalf("server = %s, want %s", got, server)
	}
}
