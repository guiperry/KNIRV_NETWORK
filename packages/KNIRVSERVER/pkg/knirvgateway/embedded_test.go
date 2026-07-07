package knirvgateway

import (
	"bytes"
	"compress/gzip"
	"os"
	"path/filepath"
	"testing"
)

// TestDecompressEmbedded_PlainData verifies that non-gzip data passes through unchanged.
func TestDecompressEmbedded_PlainData(t *testing.T) {
	original := []byte("ELF\x02\x01\x01\x00\x00\x00\x00\x00\x00\x00\x00\x00hello world")
	result, err := decompressEmbedded(original)
	if err != nil {
		t.Fatalf("decompressEmbedded(plain) returned error: %v", err)
	}
	if !bytes.Equal(result, original) {
		t.Fatalf("plain data was modified: got %x, want %x", result, original)
	}
}

// TestDecompressEmbedded_GzipData verifies that gzip-compressed data is correctly decompressed.
func TestDecompressEmbedded_GzipData(t *testing.T) {
	original := []byte("#!/bin/bash\necho hello knirv\n")
	var buf bytes.Buffer
	w := gzip.NewWriter(&buf)
	if _, err := w.Write(original); err != nil {
		t.Fatalf("failed to gzip test data: %v", err)
	}
	w.Close()
	compressed := buf.Bytes()

	result, err := decompressEmbedded(compressed)
	if err != nil {
		t.Fatalf("decompressEmbedded(gzip) returned error: %v", err)
	}
	if !bytes.Equal(result, original) {
		t.Fatalf("decompressed data mismatch: got %q, want %q", string(result), string(original))
	}
}

// TestDecompressEmbedded_EmptyData verifies edge case of empty input.
func TestDecompressEmbedded_EmptyData(t *testing.T) {
	result, err := decompressEmbedded(nil)
	if err != nil {
		t.Fatalf("decompressEmbedded(nil) returned error: %v", err)
	}
	if result != nil {
		t.Fatalf("expected nil, got %x", result)
	}

	result, err = decompressEmbedded([]byte{})
	if err != nil {
		t.Fatalf("decompressEmbedded(empty) returned error: %v", err)
	}
	if len(result) != 0 {
		t.Fatalf("expected empty, got %x", result)
	}
}

// TestExtractEmbeddedBinary_WithGzip verifies ExtractEmbeddedBinary works
// correctly when the embedded data is gzip-compressed (simulated).
func TestExtractEmbeddedBinary_WithGzip(t *testing.T) {
	original := []byte("test-knirvgateway-binary-content")
	var buf bytes.Buffer
	w := gzip.NewWriter(&buf)
	if _, err := w.Write(original); err != nil {
		t.Fatalf("gzip write: %v", err)
	}
	w.Close()

	// Temporarily replace embeddedBinary with gzipped data
	saved := embeddedBinary
	embeddedBinary = buf.Bytes()
	defer func() { embeddedBinary = saved }()

	destDir := t.TempDir()
	destPath, err := ExtractEmbeddedBinary(destDir)
	if err != nil {
		t.Fatalf("ExtractEmbeddedBinary: %v", err)
	}

	// Verify the extracted file exists and has correct content
	got, err := os.ReadFile(destPath)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if !bytes.Equal(got, original) {
		t.Fatalf("extracted content mismatch: got %q, want %q", string(got), string(original))
	}

	// Verify it's executable
	info, err := os.Stat(destPath)
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	mode := info.Mode()
	if mode&0111 == 0 {
		t.Fatalf("extracted binary is not executable: %v", mode)
	}

	// Verify the expected name
	if filepath.Base(destPath) != "knirvgateway" {
		t.Fatalf("unexpected filename: %s", destPath)
	}
}
