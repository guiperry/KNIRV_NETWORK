package ledger

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The leaf-hash list must mirror the log exactly after every commit kind, and
// any disagreement must be reported rather than repaired.
func TestLeafListMirrorsLogAcrossAppendAndBatch(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state_log.json")
	lg := openWithGenesis(t, path)
	if _, err := lg.Append("a", map[string]int{"v": 1}); err != nil {
		t.Fatalf("append: %v", err)
	}
	if _, err := lg.AppendBatch([]BatchEntry{{Kind: "b", Payload: 2}, {Kind: "c", Payload: 3}}); err != nil {
		t.Fatalf("batch: %v", err)
	}
	if err := lg.VerifyLeafList(); err != nil {
		t.Fatalf("live list should verify: %v", err)
	}
	lg.Close()

	reopened, err := Open(path)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer reopened.Close()
	if err := reopened.VerifyLeafList(); err != nil {
		t.Fatalf("list after restart should verify: %v", err)
	}
	if reopened.Broken() {
		t.Fatal("clean log must not report broken")
	}
}

func TestLeafListMismatchesAreReportedNotRepaired(t *testing.T) {
	cases := map[string]func(string) string{
		"truncated": func(s string) string { return s[:strings.LastIndex(strings.TrimSuffix(s, "\n"), "\n")+1] },
		"extra":     func(s string) string { return s + strings.Repeat("0", 64) + "\n" },
		"altered": func(s string) string {
			return "f" + s[1:]
		},
		"partial": func(s string) string { return s[:len(s)-10] },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "state_log.json")
			lg := openWithGenesis(t, path)
			if _, err := lg.Append("a", 1); err != nil {
				t.Fatalf("append: %v", err)
			}
			lg.Close()

			leafPath := LeafListPath(path)
			raw, err := os.ReadFile(leafPath)
			if err != nil {
				t.Fatalf("read leaf list: %v", err)
			}
			if err := os.WriteFile(leafPath, []byte(mutate(string(raw))), 0600); err != nil {
				t.Fatalf("write leaf list: %v", err)
			}
			before, _ := os.ReadFile(leafPath)

			reopened, err := Open(path)
			if err != nil {
				t.Fatalf("open must still succeed on a mismatched list: %v", err)
			}
			defer reopened.Close()
			if err := reopened.VerifyLeafList(); err == nil {
				t.Fatal("mismatched leaf list was not reported")
			}
			after, _ := os.ReadFile(leafPath)
			if string(before) != string(after) {
				t.Fatal("verification must not rewrite the leaf list")
			}
		})
	}
}

func TestMissingLeafListWithRecordsIsReported(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state_log.json")
	lg := openWithGenesis(t, path)
	lg.Close()
	if err := os.Remove(LeafListPath(path)); err != nil {
		t.Fatalf("remove leaf list: %v", err)
	}
	reopened, err := Open(path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer reopened.Close()
	if err := reopened.VerifyLeafList(); err == nil {
		t.Fatal("a missing leaf list with records must be reported")
	}
}
