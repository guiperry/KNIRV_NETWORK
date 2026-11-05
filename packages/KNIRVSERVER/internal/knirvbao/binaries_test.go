package knirvbao

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInstallBinaryDoesNotReplaceUnexpectedFiles(t *testing.T) {
	root := filepath.Join(t.TempDir(), "openbao")
	path, digest, err := installBinary(root, "bao", []byte("test executable"))
	if err != nil {
		t.Fatal(err)
	}
	second, secondDigest, err := installBinary(root, "bao", []byte("test executable"))
	if err != nil || second != path || digest != secondDigest {
		t.Fatal("reusing verified binary failed")
	}
	if err := os.WriteFile(path, []byte("unexpected contents"), 0700); err != nil {
		t.Fatal(err)
	}
	if _, _, err := installBinary(root, "bao", []byte("test executable")); err == nil {
		t.Fatal("mismatched existing file accepted")
	}
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(contents) != "unexpected contents" {
		t.Fatal("existing file was overwritten")
	}
}

func TestInstallBinaryRejectsInsecureDirectory(t *testing.T) {
	root := filepath.Join(t.TempDir(), "openbao")
	if err := os.Mkdir(root, 0755); err != nil {
		t.Fatal(err)
	}
	if _, _, err := installBinary(root, "bao", []byte("binary")); err == nil {
		t.Fatal("public directory accepted")
	}
}
