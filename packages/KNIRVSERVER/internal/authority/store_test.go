package authority

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func fakeStored(op string, self, prev byte) StoredRecord {
	h := func(b byte) string { return hex.EncodeToString(append(make([]byte, 31), b)) }
	return StoredRecord{
		Record: RecordJSON{
			Tier: 1, Domain: "evidence.commit", Scope: op, Operator: op, Epoch: 1,
			CanonicalActionHash: h(0xaa), PrevRecordHash: h(prev), Header: strings.Repeat("00", 80), Hash: h(self),
			WitnessKind: "sw", SessionCert: CertJSON{SessionPub: "02" + strings.Repeat("11", 32), Operator: op, Scope: op, Entropy: "csprng"},
			Signature: strings.Repeat("22", 64),
		},
		IssuedAt: 1, Verified: true, Mode: ModeObserve,
	}
}

func zeroPrev(op string, self byte) StoredRecord {
	sr := fakeStored(op, self, 0)
	sr.Record.PrevRecordHash = strings.Repeat("00", 32)
	return sr
}

func TestStoreAppendReopenKeepsChain(t *testing.T) {
	dir := t.TempDir()
	st, repaired, err := OpenStore(dir)
	if err != nil || repaired {
		t.Fatalf("open: %v repaired=%v", err, repaired)
	}
	if err := st.Append(zeroPrev("opA", 1)); err != nil {
		t.Fatal(err)
	}
	if err := st.Append(fakeStored("opA", 2, 1)); err != nil {
		t.Fatal(err)
	}
	if err := st.Append(zeroPrev("opB", 9)); err != nil { // another operator has its own chain
		t.Fatal(err)
	}
	st.Close()

	st, repaired, err = OpenStore(dir)
	if err != nil || repaired {
		t.Fatalf("reopen: %v repaired=%v", err, repaired)
	}
	defer st.Close()
	if st.Total() != 3 || st.Count("opA") != 2 || st.Count("opB") != 1 {
		t.Fatalf("counts after reopen: total %d A %d B %d", st.Total(), st.Count("opA"), st.Count("opB"))
	}
	if head := st.Head("opA"); head[31] != 2 {
		t.Fatalf("head of opA = %x", head)
	}
	if _, ok := st.Get(fakeStored("opA", 1, 0).Record.Hash); !ok {
		t.Fatal("record 1 not found after reopen")
	}
}

func TestStoreRefusesForkGapAndDuplicate(t *testing.T) {
	st, _, err := OpenStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	if err := st.Append(zeroPrev("opA", 1)); err != nil {
		t.Fatal(err)
	}
	if err := st.Append(fakeStored("opA", 2, 1)); err != nil {
		t.Fatal(err)
	}
	for name, sr := range map[string]StoredRecord{
		"fork (second child of record 1)": fakeStored("opA", 3, 1),
		"gap (skips a link)":              fakeStored("opA", 4, 7),
		"genesis again":                   zeroPrev("opA", 5),
		"duplicate hash":                  fakeStored("opA", 2, 2),
	} {
		if err := st.Append(sr); err == nil {
			t.Errorf("%s was accepted", name)
		}
	}
	if st.Count("opA") != 2 || st.Head("opA")[31] != 2 {
		t.Fatalf("a refused append changed the store: count %d head %x", st.Count("opA"), st.Head("opA"))
	}
}

func TestStoreRepairsTornFinalLine(t *testing.T) {
	dir := t.TempDir()
	st, _, _ := OpenStore(dir)
	if err := st.Append(zeroPrev("opA", 1)); err != nil {
		t.Fatal(err)
	}
	st.Close()
	f, err := os.OpenFile(filepath.Join(dir, recordsFile), os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	f.WriteString(`{"record":{"tier":1,"domain":"half a rec`) // crash mid-append: no newline
	f.Close()

	st, repaired, err := OpenStore(dir)
	if err != nil || !repaired {
		t.Fatalf("expected a repair, got repaired=%v err=%v", repaired, err)
	}
	if st.Total() != 1 {
		t.Fatalf("total = %d after repair", st.Total())
	}
	if err := st.Append(fakeStored("opA", 2, 1)); err != nil {
		t.Fatalf("append after repair: %v", err)
	}
	st.Close()
	st, repaired, err = OpenStore(dir)
	if err != nil || repaired || st.Total() != 2 {
		t.Fatalf("second reopen: err=%v repaired=%v total=%d", err, repaired, st.Total())
	}
	st.Close()
}

func TestStoreFailsClosedOnMiddleCorruptionAndBrokenLink(t *testing.T) {
	good1, _ := jsonLine(zeroPrev("opA", 1))
	good2, _ := jsonLine(fakeStored("opA", 2, 1))
	badLink, _ := jsonLine(fakeStored("opA", 3, 9)) // does not extend record 2
	for name, content := range map[string]string{
		"garbage in the middle": good1 + "this is not json\n" + good2,
		"broken chain link":     good1 + good2 + badLink,
		"first record not genesis": func() string {
			s, _ := jsonLine(fakeStored("opA", 2, 1))
			return s
		}(),
	} {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			if err := os.WriteFile(filepath.Join(dir, recordsFile), []byte(content), 0o600); err != nil {
				t.Fatal(err)
			}
			if _, _, err := OpenStore(dir); !errors.Is(err, ErrCorrupt) {
				t.Fatalf("want ErrCorrupt, got %v", err)
			}
		})
	}
}

func TestStoreFilePermissionsAndAnchorState(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "authority")
	st, _, err := OpenStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	if info, _ := os.Stat(dir); info.Mode().Perm() != 0o700 {
		t.Errorf("dir mode %v", info.Mode().Perm())
	}
	if info, _ := os.Stat(filepath.Join(dir, recordsFile)); info.Mode().Perm() != 0o600 {
		t.Errorf("records mode %v", info.Mode().Perm())
	}
	if got, err := st.LoadAnchor(); err != nil || got != (AnchorState{}) {
		t.Fatalf("empty anchor state: %+v %v", got, err)
	}
	want := AnchorState{LastEnd: 7, PrevCheckpoint: strings.Repeat("ab", 32), Registered: true}
	if err := st.SaveAnchor(want); err != nil {
		t.Fatal(err)
	}
	if got, err := st.LoadAnchor(); err != nil || got != want {
		t.Fatalf("anchor state = %+v, %v", got, err)
	}
	if info, _ := os.Stat(filepath.Join(dir, anchorFile)); info.Mode().Perm() != 0o600 {
		t.Errorf("anchor mode %v", info.Mode().Perm())
	}
}

func jsonLine(sr StoredRecord) (string, error) {
	b, err := json.Marshal(sr)
	return string(b) + "\n", err
}
