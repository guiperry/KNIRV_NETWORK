package install

import (
	"bytes"
	"compress/gzip"
	"context"
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
	path, err := i.downloadServer(context.Background(), t.TempDir(), "https://releases.knirv.com/knirv/llama/linux-amd64/llama-server.gz")
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
