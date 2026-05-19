package knirvarena

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createTestBundle(t *testing.T) ([]byte, string) {
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)

	files := map[string]string{
		"index.html": "<!DOCTYPE html><html><head><title>KNIRVARENA</title></head><body></body></html>",
		"assets/app.js": "console.log('KNIRVARENA');",
		"assets/style.css": "body { margin: 0; }",
	}

	for name, content := range files {
		hdr := &tar.Header{
			Name: name,
			Size: int64(len(content)),
			Mode: 0644,
		}
		require.NoError(t, tw.WriteHeader(hdr))
		_, err := tw.Write([]byte(content))
		require.NoError(t, err)
	}

	require.NoError(t, tw.Close())
	require.NoError(t, gz.Close())

	// Create subdirectory assets
	var buf2 bytes.Buffer
	gz2 := gzip.NewWriter(&buf2)
	tw2 := tar.NewWriter(gz2)

	require.NoError(t, tw2.WriteHeader(&tar.Header{
		Name: "assets/",
		Typeflag: tar.TypeDir,
		Mode: 0755,
	}))
	require.NoError(t, tw2.WriteHeader(&tar.Header{
		Name: "assets/sub.js",
		Size: int64(len("sub")),
		Mode: 0644,
	}))
	_, err := tw2.Write([]byte("sub"))
	require.NoError(t, err)

	require.NoError(t, tw2.Close())
	require.NoError(t, gz2.Close())

	// Return the first bundle (simpler) for most tests
	return buf.Bytes(), ""
}

func TestExtractEmbeddedApp_EmptyDestDir(t *testing.T) {
	// Should work with default dir when embedded bundle is non-empty
	// but our embedded bundle is the real tar.gz (placeholder)
	// For unit testing we can't test with the real embedded bundle easily
	// as it's //go:embed'd at compile time
	t.Skip("requires real embedded bundle at compile time")
}

func TestWriteFileAtomically(t *testing.T) {
	dir := t.TempDir()
	dest := filepath.Join(dir, "test.txt")

	err := writeFileAtomically(dest, []byte("hello"), 0644)
	require.NoError(t, err)

	data, err := os.ReadFile(dest)
	require.NoError(t, err)
	assert.Equal(t, "hello", string(data))
}

func TestWriteFileAtomically_CreatesDirs(t *testing.T) {
	dir := t.TempDir()
	dest := filepath.Join(dir, "sub", "nested", "test.txt")

	err := writeFileAtomically(dest, []byte("nested"), 0644)
	require.NoError(t, err)

	data, err := os.ReadFile(dest)
	require.NoError(t, err)
	assert.Equal(t, "nested", string(data))
}

func TestEnsureDir(t *testing.T) {
	dir := t.TempDir()
	nested := filepath.Join(dir, "a", "b", "c")

	err := ensureDir(nested)
	require.NoError(t, err)

	info, err := os.Stat(nested)
	require.NoError(t, err)
	assert.True(t, info.IsDir())
}

func TestDefaultExtractDir_HonorsEnv(t *testing.T) {
	t.Setenv("KNIRV_ARENA_EXTRACT_DIR", "/tmp/arena-test-override")
	dir, err := defaultExtractDir()
	require.NoError(t, err)
	assert.Equal(t, "/tmp/arena-test-override", dir)
}
