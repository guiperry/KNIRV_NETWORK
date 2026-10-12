package storage

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/knirvcorp/knirvbase/internal/crypto/pqc"
	"github.com/knirvcorp/knirvbase/pkg/nrv"
)

// writeNRV3 builds a hasher-shaped file: full brackets, chain-XOR index.
func writeNRV3(t *testing.T, path string, counts ...int) [][]nrv.Bracket {
	t.Helper()
	fw := nrv.NewNRV3Writer()
	var all [][]nrv.Bracket
	n, off := 0, int64(0)
	for fi, count := range counts {
		f := nrv.NRV3FrameEntry{
			ID: fmt.Sprintf("frame-%d", fi), TimestampUnix: 1700000000, Count: count, Offset: off,
			Linguistic: nrv.NRV3LinguisticMetadata{Tokens: 3, Domain: "prose"},
		}
		var buf []byte
		var bs []nrv.Bracket
		var prev [nrv.BracketSize]byte
		for k := 0; k < count; k++ {
			var b nrv.Bracket
			for j := range b.Projections {
				b.Projections[j] = byte(10+j) + byte(n%3)
				if n%90 == 89 {
					b.Projections[j] = byte(255 - 7*j)
				}
			}
			b.GoldenSeed, b.SubSecondUS, b.DepHead = uint32(n*2654435761), uint32(n), int8(n%100-50)
			b.Memory[0], b.LSHSalt = byte(n), uint32(n<<8)
			n++
			enc := nrv.EncodeBracket(&b)
			xor, typ := make([]byte, nrv.BracketSize), "P"
			for j := range xor {
				xor[j] = enc[j] ^ prev[j]
			}
			if k == 0 {
				typ = "I"
				copy(xor, enc[:])
			}
			prev = enc
			f.BracketIndex = append(f.BracketIndex, nrv.NRV3BracketIndexEntry{Index: k, Type: typ, XOR: xor})
			buf = append(buf, enc[:]...)
			bs = append(bs, b)
		}
		fw.AddFrame(f, buf)
		all = append(all, bs)
		off += int64(count) * nrv.BracketSize
	}
	if err := nrv.WriteNRV3(path, fw); err != nil {
		t.Fatal(err)
	}
	return all
}

// readAll reads every converted frame back through the real KNIRVBASE reader
// (which resolves P-brackets against their anchors).
func readAll(t *testing.T, paths []string) []*nrv.Bracket {
	t.Helper()
	var out []*nrv.Bracket
	for _, p := range paths {
		r, err := NewNRVReader(p)
		if err != nil {
			t.Fatal(err)
		}
		for _, e := range r.GetRegistry().Frames {
			_, bs, err := r.GetFrame(e.ID)
			if err != nil {
				t.Fatal(err)
			}
			out = append(out, bs...)
		}
		_ = r.Close()
	}
	return out
}

func TestConvertNRV3FileRoundTripsThroughTheKNIRVBASEReader(t *testing.T) {
	dir := t.TempDir()
	in := filepath.Join(dir, "in.nrv3")
	orig := writeNRV3(t, in, 150, 40)

	files, err := ConvertNRV3File(in, filepath.Join(dir, "out.nrv"), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 || files[0] != filepath.Join(dir, "out.nrv") {
		t.Fatalf("files = %v", files)
	}
	got := readAll(t, files)
	if len(got) != 190 {
		t.Fatalf("read %d brackets, want 190", len(got))
	}
	k := 0
	for _, frame := range orig {
		for _, want := range frame {
			g := *got[k]
			g.ID, g.FrameID, g.FrameUnix, g.Meta = "", "", 0, nil
			if g != want {
				t.Fatalf("bracket %d differs after conversion:\n got %+v\nwant %+v", k, g, want)
			}
			k++
		}
	}

	r, _ := NewNRVReader(files[0])
	defer r.Close()
	var nP int
	for _, e := range r.GetRegistry().Frames {
		for _, m := range e.BracketIndex {
			if m.Type == nrv.DeltaTypeP {
				nP++
			}
		}
	}
	if nP == 0 {
		t.Error("expected delta-encoded P-brackets")
	}
}

func TestConvertNRV3FileShardsLargeInputs(t *testing.T) {
	dir := t.TempDir()
	in := filepath.Join(dir, "big.nrv3")
	total := MaxBracketsPerFile + 5000 // one 21k-bracket frame: far over one file's registry budget
	orig := writeNRV3(t, in, total)

	files, err := ConvertNRV3File(in, filepath.Join(dir, "big.nrv"), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 2 || files[1] != filepath.Join(dir, "big.1.nrv") {
		t.Fatalf("files = %v", files)
	}
	got := readAll(t, files)
	if len(got) != total {
		t.Fatalf("read %d brackets, want %d", len(got), total)
	}
	for _, i := range []int{0, 1, 49, 50, 89, 90, 1023, 1024, 15999, 16000, total - 1} {
		g := *got[i]
		g.ID, g.FrameID, g.FrameUnix, g.Meta = "", "", 0, nil
		if g != orig[0][i] {
			t.Fatalf("bracket %d differs after sharded conversion", i)
		}
	}
}

func TestConvertNRV3FileSignsFramesWhenKeyGiven(t *testing.T) {
	dir := t.TempDir()
	in := filepath.Join(dir, "in.nrv3")
	writeNRV3(t, in, 30, 30)
	kp, err := pqc.GeneratePQCKeyPair("conv", "signature")
	if err != nil {
		t.Fatal(err)
	}

	files, err := ConvertNRV3File(in, filepath.Join(dir, "signed.nrv"), kp)
	if err != nil {
		t.Fatal(err)
	}
	r, _ := NewNRVReader(files[0])
	defer r.Close()
	reg := r.GetRegistry()
	if reg.PQCManifest.KeyID != kp.ID || len(reg.PQCManifest.FrameSignatures) != 2 {
		t.Fatalf("manifest = %+v", reg.PQCManifest)
	}
	for _, e := range reg.Frames {
		ok, err := r.VerifyFrame(e.ID, kp)
		if err != nil || !ok {
			t.Fatalf("frame %s does not verify: %v %v", e.ID, ok, err)
		}
	}
	other, _ := pqc.GeneratePQCKeyPair("other", "signature")
	if ok, _ := r.VerifyFrame(reg.Frames[0].ID, other); ok {
		t.Fatal("a different key must not verify")
	}

	unsigned, _ := ConvertNRV3File(in, filepath.Join(dir, "unsigned.nrv"), nil)
	ur, _ := NewNRVReader(unsigned[0])
	defer ur.Close()
	if len(ur.GetRegistry().PQCManifest.FrameSignatures) != 0 {
		t.Fatal("no key: no signatures")
	}
}

func TestConvertNRV3FileCleansUpOnError(t *testing.T) {
	dir := t.TempDir()
	in := filepath.Join(dir, "in.nrv3")
	writeNRV3(t, in, 10)
	data, _ := os.ReadFile(in)
	// corrupt the delta type of the second bracket in the registry
	bad := []byte(string(data))
	idx := indexOf(bad, []byte(`"type":"P"`))
	copy(bad[idx:], []byte(`"type":"Q"`))
	if err := os.WriteFile(in, bad, 0o644); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(dir, "out.nrv")
	if _, err := ConvertNRV3File(in, out, nil); err == nil {
		t.Fatal("expected an error")
	}
	if _, err := os.Stat(out); !os.IsNotExist(err) {
		t.Fatal("a failed conversion must not leave output behind")
	}
}

func indexOf(b, sub []byte) int {
	for i := 0; i+len(sub) <= len(b); i++ {
		if string(b[i:i+len(sub)]) == string(sub) {
			return i
		}
	}
	return -1
}

// TestConvertRealNRV3 converts a real models batch file and checks every bracket
// comes back identical through the KNIRVBASE reader:
//
//	KNIRV_REAL_NRV3=/var/lib/knirvserver/knirvmodels/data/frames/batches/<batch>/training_frames.nrv \
//	  go test ./internal/storage -run RealNRV3 -v -timeout 30m
func TestConvertRealNRV3(t *testing.T) {
	in := os.Getenv("KNIRV_REAL_NRV3")
	if in == "" {
		t.Skip("set KNIRV_REAL_NRV3 to a models NRV3 file to run")
	}
	src, err := nrv.OpenNRV3(in)
	if err != nil {
		t.Fatal(err)
	}
	var want []byte
	for _, f := range src.Registry.Frames {
		want = append(want, src.RawBrackets(f)...)
	}

	files, err := ConvertNRV3File(in, filepath.Join(t.TempDir(), "real.nrv"), nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%d input brackets -> %d files", len(want)/nrv.BracketSize, len(files))

	k, nP := 0, 0
	for _, p := range files {
		r, err := NewNRVReader(p)
		if err != nil {
			t.Fatal(err)
		}
		for _, e := range r.GetRegistry().Frames {
			_, bs, err := r.GetFrame(e.ID)
			if err != nil {
				t.Fatal(err)
			}
			for i, b := range bs {
				if e.BracketIndex[i].Type == nrv.DeltaTypeP {
					nP++
				}
				g := *b
				enc := nrv.EncodeBracket(&g)
				if !bytes.Equal(enc[:], want[k*nrv.BracketSize:(k+1)*nrv.BracketSize]) {
					t.Fatalf("bracket %d differs after conversion", k)
				}
				k++
			}
		}
		_ = r.Close()
	}
	if k*nrv.BracketSize != len(want) {
		t.Fatalf("converted %d brackets, want %d", k, len(want)/nrv.BracketSize)
	}
	t.Logf("all %d brackets round-trip; %d stored as P-deltas", k, nP)
}
