package knirvmodels

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMigrateLegacyDataDir(t *testing.T) {
	app := t.TempDir()
	old := filepath.Join(app, "knirvhasher", "data")
	if err := os.MkdirAll(old, 0o755); err != nil {
		t.Fatal(err)
	}
	os.WriteFile(filepath.Join(old, "x.json"), []byte("1"), 0o644)

	moved, err := MigrateLegacyDataDir(app)
	if err != nil || !moved {
		t.Fatalf("moved=%v err=%v", moved, err)
	}
	if _, err := os.Stat(filepath.Join(app, "knirvmodels", "data", "x.json")); err != nil {
		t.Fatal("data not at new path")
	}
	if _, err := os.Stat(filepath.Join(app, "knirvhasher", "data", "x.json")); err != nil {
		t.Fatal("legacy symlink does not resolve")
	}
	// Second run is a no-op (old path is now a symlink).
	if moved, err := MigrateLegacyDataDir(app); moved || err != nil {
		t.Fatalf("second run moved=%v err=%v", moved, err)
	}
}

func TestMigrateLegacyDataDirNeverOverwrites(t *testing.T) {
	app := t.TempDir()
	os.MkdirAll(filepath.Join(app, "knirvhasher"), 0o755)
	os.MkdirAll(filepath.Join(app, "knirvmodels"), 0o755)
	if moved, err := MigrateLegacyDataDir(app); moved || err != nil {
		t.Fatalf("moved=%v err=%v", moved, err)
	}
	if _, err := os.Stat(filepath.Join(app, "knirvhasher")); err != nil {
		t.Fatal("old dir was touched")
	}
}
