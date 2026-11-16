package embedded

import (
	"crypto/sha256"
	"debug/elf"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// The CUDA hasher library must be part of the embedded set, exported next to the
// data-seeder binary that links it, executable, and byte-identical to the embedded
// copy. KNIRV_APP_DATA_DIR keeps the test out of the real /var/lib/knirvserver.
func TestEnsureExtractedExportsLibcudaHash(t *testing.T) {
	if !IsEmbedded("libcuda_hash.so") {
		t.Skip("pipeline binaries are not embedded in this build; run make build-pipeline-all first")
	}
	appData := t.TempDir()
	t.Setenv("KNIRV_APP_DATA_DIR", appData)

	if err := EnsureExtracted(); err != nil {
		t.Fatalf("EnsureExtracted: %v", err)
	}

	lib := filepath.Join(appData, "bin", "libcuda_hash.so")
	st, err := os.Stat(lib)
	if err != nil {
		t.Fatalf("libcuda_hash.so was not exported: %v", err)
	}
	if st.Mode().Perm()&0111 == 0 {
		t.Fatalf("libcuda_hash.so is not executable: mode %s", st.Mode())
	}
	embedded, err := GetEmbedded("libcuda_hash.so")
	if err != nil {
		t.Fatal(err)
	}
	if got, err := fileSHA256(lib); err != nil || got != sha256.Sum256(embedded) {
		t.Fatalf("exported libcuda_hash.so does not match the embedded copy: %v", err)
	}
	if _, err := os.Stat(filepath.Join(appData, "bin", "data-seeder")); err != nil {
		t.Fatalf("data-seeder was not exported next to the library: %v", err)
	}

	// A stale/corrupt copy is detected by hash and replaced, not trusted.
	if err := os.WriteFile(lib, []byte("stale"), 0755); err != nil {
		t.Fatal(err)
	}
	if _, err := extractVerified("libcuda_hash.so"); err != nil {
		t.Fatalf("extractVerified: %v", err)
	}
	if got, err := fileSHA256(lib); err != nil || got != sha256.Sum256(embedded) {
		t.Fatalf("corrupt libcuda_hash.so was not replaced: %v", err)
	}
}

// Discriminates the hash check from the old always-overwrite export: an identical
// copy that is already in place must be left alone (its mtime is not touched).
// The old EnsureExtracted rewrote every file on each start, so this fails there.
func TestEnsureExtractedLeavesIdenticalCopyAlone(t *testing.T) {
	if !IsEmbedded("libcuda_hash.so") {
		t.Skip("pipeline binaries are not embedded in this build; run make build-pipeline-all first")
	}
	appData := t.TempDir()
	t.Setenv("KNIRV_APP_DATA_DIR", appData)
	if err := EnsureExtracted(); err != nil {
		t.Fatalf("first EnsureExtracted: %v", err)
	}
	lib := filepath.Join(appData, "bin", "libcuda_hash.so")
	past := time.Now().Add(-48 * time.Hour).Truncate(time.Second)
	if err := os.Chtimes(lib, past, past); err != nil {
		t.Fatal(err)
	}
	if err := EnsureExtracted(); err != nil {
		t.Fatalf("second EnsureExtracted: %v", err)
	}
	st, err := os.Stat(lib)
	if err != nil {
		t.Fatal(err)
	}
	if !st.ModTime().Equal(past) {
		t.Fatalf("identical libcuda_hash.so was rewritten (mtime %s, want %s)", st.ModTime(), past)
	}
}

// The exported data-seeder must resolve libcuda_hash.so from its own directory,
// which is /var/lib/knirvserver/bin in production (both files are extracted there).
func TestDataSeederRunpathResolvesFromOrigin(t *testing.T) {
	if !IsEmbedded("data-seeder") {
		t.Skip("pipeline binaries are not embedded in this build; run make build-pipeline-all first")
	}
	appData := t.TempDir()
	t.Setenv("KNIRV_APP_DATA_DIR", appData)
	if err := EnsureExtracted(); err != nil {
		t.Fatalf("EnsureExtracted: %v", err)
	}
	f, err := elf.Open(filepath.Join(appData, "bin", "data-seeder"))
	if err != nil {
		t.Fatalf("open data-seeder: %v", err)
	}
	defer f.Close()
	runpaths, err := f.DynString(elf.DT_RUNPATH)
	if err != nil {
		t.Fatalf("read data-seeder RUNPATH: %v", err)
	}
	for _, runpath := range runpaths {
		if strings.Contains(runpath, "$ORIGIN") {
			return
		}
	}
	t.Fatal("data-seeder RUNPATH does not contain $ORIGIN; libcuda_hash.so next to it would not be found")
}

func TestAvailableBinariesListsLibcudaHash(t *testing.T) {
	for _, b := range AvailableBinaries {
		if b.Name == "libcuda_hash.so" {
			return
		}
	}
	t.Fatal("libcuda_hash.so is missing from AvailableBinaries")
}
