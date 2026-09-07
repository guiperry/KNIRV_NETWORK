package bundling

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
)

func TestHashReader(t *testing.T) {
	data := []byte("hello world")
	h := sha256.Sum256(data)
	expected := hex.EncodeToString(h[:])

	result, err := HashReader(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != expected {
		t.Fatalf("expected %s, got %s", expected, result)
	}
}

func TestContentHashOfFile(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test.txt")
	content := []byte("test content for hashing")
	if err := os.WriteFile(filePath, content, 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	h := sha256.Sum256(content)
	expected := hex.EncodeToString(h[:])

	result, err := ContentHashOfFile(filePath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != expected {
		t.Fatalf("expected %s, got %s", expected, result)
	}
}

func TestContentHashOfFileNotFound(t *testing.T) {
	_, err := ContentHashOfFile("/nonexistent/file/path")
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
}

func TestWriteBundle(t *testing.T) {
	tmpDir := t.TempDir()
	bundleDir := filepath.Join(tmpDir, "bundle_contents")
	bundlePath := filepath.Join(tmpDir, "test.ulora")

	if err := EnsureDir(bundleDir); err != nil {
		t.Fatalf("ensure dir: %v", err)
	}

	if err := os.WriteFile(filepath.Join(bundleDir, ManifestFile), []byte(`{"test":true}`), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	if err := os.WriteFile(filepath.Join(bundleDir, WeightsFile), []byte("fake weights"), 0o644); err != nil {
		t.Fatalf("write weights: %v", err)
	}
	if err := os.WriteFile(filepath.Join(bundleDir, AnchorsFile), []byte("fake anchors"), 0o644); err != nil {
		t.Fatalf("write anchors: %v", err)
	}

	if err := WriteBundle(bundleDir, bundlePath); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, err := os.Stat(bundlePath); os.IsNotExist(err) {
		t.Fatal("bundle file was not created")
	}
}

func TestWriteBundleMissingFiles(t *testing.T) {
	tmpDir := t.TempDir()
	bundleDir := filepath.Join(tmpDir, "partial_bundle")
	bundlePath := filepath.Join(tmpDir, "partial.ulora")

	if err := EnsureDir(bundleDir); err != nil {
		t.Fatalf("ensure dir: %v", err)
	}

	if err := os.WriteFile(filepath.Join(bundleDir, ManifestFile), []byte(`{"test":true}`), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	if err := WriteBundle(bundleDir, bundlePath); err != nil {
		t.Fatalf("expected success for partial bundle, got: %v", err)
	}
}

func TestExtractBundle(t *testing.T) {
	tmpDir := t.TempDir()
	srcDir := filepath.Join(tmpDir, "source")
	extractDir := filepath.Join(tmpDir, "extracted")

	if err := EnsureDir(srcDir); err != nil {
		t.Fatalf("ensure dir: %v", err)
	}

	manifestContent := []byte(`{"name":"test"}`)
	weightsContent := []byte("model weights data")
	anchorsContent := []byte("anchors data")

	if err := os.WriteFile(filepath.Join(srcDir, ManifestFile), manifestContent, 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	if err := os.WriteFile(filepath.Join(srcDir, WeightsFile), weightsContent, 0o644); err != nil {
		t.Fatalf("write weights: %v", err)
	}
	if err := os.WriteFile(filepath.Join(srcDir, AnchorsFile), anchorsContent, 0o644); err != nil {
		t.Fatalf("write anchors: %v", err)
	}

	bundlePath := filepath.Join(tmpDir, "test.ulora")
	if err := WriteBundle(srcDir, bundlePath); err != nil {
		t.Fatalf("write bundle: %v", err)
	}

	if err := ExtractBundle(bundlePath, extractDir); err != nil {
		t.Fatalf("extract bundle: %v", err)
	}

	extractedManifest, err := os.ReadFile(filepath.Join(extractDir, ManifestFile))
	if err != nil {
		t.Fatalf("read extracted manifest: %v", err)
	}
	if !bytes.Equal(extractedManifest, manifestContent) {
		t.Fatalf("manifest mismatch: expected %s, got %s", manifestContent, extractedManifest)
	}

	extractedWeights, err := os.ReadFile(filepath.Join(extractDir, WeightsFile))
	if err != nil {
		t.Fatalf("read extracted weights: %v", err)
	}
	if !bytes.Equal(extractedWeights, weightsContent) {
		t.Fatalf("weights mismatch")
	}

	extractedAnchors, err := os.ReadFile(filepath.Join(extractDir, AnchorsFile))
	if err != nil {
		t.Fatalf("read extracted anchors: %v", err)
	}
	if !bytes.Equal(extractedAnchors, anchorsContent) {
		t.Fatalf("anchors mismatch")
	}
}

func TestReadFileFromDir(t *testing.T) {
	tmpDir := t.TempDir()
	content := []byte("test content")
	filePath := filepath.Join(tmpDir, ManifestFile)
	if err := os.WriteFile(filePath, content, 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	result, err := ReadFile(tmpDir, ManifestFile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !bytes.Equal(result, content) {
		t.Fatalf("content mismatch")
	}
}

func TestReadFileFromArchive(t *testing.T) {
	tmpDir := t.TempDir()
	srcDir := filepath.Join(tmpDir, "source")
	if err := EnsureDir(srcDir); err != nil {
		t.Fatalf("ensure dir: %v", err)
	}

	manifestContent := []byte(`{"name":"test-bundle"}`)
	if err := os.WriteFile(filepath.Join(srcDir, ManifestFile), manifestContent, 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	if err := os.WriteFile(filepath.Join(srcDir, WeightsFile), []byte("weights"), 0o644); err != nil {
		t.Fatalf("write weights: %v", err)
	}

	bundlePath := filepath.Join(tmpDir, "test.ulora")
	if err := WriteBundle(srcDir, bundlePath); err != nil {
		t.Fatalf("write bundle: %v", err)
	}

	result, err := ReadFile(bundlePath, ManifestFile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !bytes.Equal(result, manifestContent) {
		t.Fatalf("content mismatch: expected %s, got %s", manifestContent, result)
	}
}

func TestReadFileNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	_, err := ReadFile(tmpDir, "nonexistent.json")
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
}

func TestReadFileFromArchiveNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	srcDir := filepath.Join(tmpDir, "source")
	if err := EnsureDir(srcDir); err != nil {
		t.Fatalf("ensure dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(srcDir, ManifestFile), []byte("test"), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	bundlePath := filepath.Join(tmpDir, "test.ulora")
	if err := WriteBundle(srcDir, bundlePath); err != nil {
		t.Fatalf("write bundle: %v", err)
	}

	_, err := ReadFile(bundlePath, "nonexistent.json")
	if err == nil {
		t.Fatal("expected error for nonexistent file in archive")
	}
}

func TestEnsureDir(t *testing.T) {
	tmpDir := t.TempDir()
	newDir := filepath.Join(tmpDir, "subdir", "nested")
	if err := EnsureDir(newDir); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info, err := os.Stat(newDir); err != nil || !info.IsDir() {
		t.Fatalf("directory was not created")
	}
}

func TestBundleRoundTrip(t *testing.T) {
	tmpDir := t.TempDir()

	originalDir := filepath.Join(tmpDir, "original")
	if err := EnsureDir(originalDir); err != nil {
		t.Fatalf("ensure dir: %v", err)
	}
	manifestData := []byte(`{"$schema":"https://ulora.org/schema/v1/manifest.json","name":"roundtrip"}`)
	weightsData := []byte("binary weights data here")
	anchorsData := []byte("anchors binary data")

	if err := os.WriteFile(filepath.Join(originalDir, ManifestFile), manifestData, 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	if err := os.WriteFile(filepath.Join(originalDir, WeightsFile), weightsData, 0o644); err != nil {
		t.Fatalf("write weights: %v", err)
	}
	if err := os.WriteFile(filepath.Join(originalDir, AnchorsFile), anchorsData, 0o644); err != nil {
		t.Fatalf("write anchors: %v", err)
	}

	bundlePath := filepath.Join(tmpDir, "roundtrip.ulora")
	if err := WriteBundle(originalDir, bundlePath); err != nil {
		t.Fatalf("write bundle: %v", err)
	}

	hash1, err := ContentHashOfFile(bundlePath)
	if err != nil {
		t.Fatalf("hash bundle: %v", err)
	}

	extractDir := filepath.Join(tmpDir, "extracted")
	if err := ExtractBundle(bundlePath, extractDir); err != nil {
		t.Fatalf("extract bundle: %v", err)
	}

	manifestResult, err := ReadFile(bundlePath, ManifestFile)
	if err != nil {
		t.Fatalf("read manifest from archive: %v", err)
	}
	if !bytes.Equal(manifestResult, manifestData) {
		t.Fatalf("manifest mismatch")
	}

	weightsResult, err := ReadFile(extractDir, WeightsFile)
	if err != nil {
		t.Fatalf("read weights from dir: %v", err)
	}
	if !bytes.Equal(weightsResult, weightsData) {
		t.Fatalf("weights mismatch")
	}

	hash2, err := ContentHashOfFile(bundlePath)
	if err != nil {
		t.Fatalf("hash bundle again: %v", err)
	}
	if hash1 != hash2 {
		t.Fatalf("content hash changed: %s vs %s", hash1, hash2)
	}
}

func TestFileConstants(t *testing.T) {
	if ManifestFile != "manifest.json" {
		t.Fatalf("expected manifest.json, got %s", ManifestFile)
	}
	if WeightsFile != "core_weights.safetensors" {
		t.Fatalf("expected core_weights.safetensors, got %s", WeightsFile)
	}
	if AnchorsFile != "calibration_anchors.parquet" {
		t.Fatalf("expected calibration_anchors.parquet, got %s", AnchorsFile)
	}
}

func TestReadFileFromArchiveNonTar(t *testing.T) {
	tmpDir := t.TempDir()
	bundlePath := filepath.Join(tmpDir, "plain.ulora")
	if err := os.WriteFile(bundlePath, []byte("not a tarball"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	_, err := ReadFile(bundlePath, ManifestFile)
	if err == nil {
		t.Fatal("expected error for non-tar file")
	}
}

func createTarBytes(tmpDir string, t *testing.T) []byte {
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)

	content := []byte("test content")
	hdr := &tar.Header{
		Name: ManifestFile,
		Mode: 0o644,
		Size: int64(len(content)),
	}
	if err := tw.WriteHeader(hdr); err != nil {
		t.Fatalf("write header: %v", err)
	}
	if _, err := tw.Write(content); err != nil {
		t.Fatalf("write content: %v", err)
	}
	tw.Close()
	gw.Close()
	return buf.Bytes()
}
