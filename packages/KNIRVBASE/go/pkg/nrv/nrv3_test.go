package nrv

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"testing"
	"time"
)

func fixtureBracket(i int) Bracket {
	var b Bracket
	for j := range b.Projections {
		b.Projections[j] = byte(i*31 + j)
	}
	b.SubSecondUS = uint32(1000 + i)
	b.Syntactic = PackSyntacticByte(uint8(i+1), uint8(i%4), uint8((i+1)%4))
	b.DepHead = int8(-3 * (i + 1))
	b.IntentFlags = uint8(i + 7)
	b.DomainSig = uint16(0xA000 + i)
	b.GoldenSeed = uint32(0xDEADBE00 + i)
	for j := range b.Memory {
		b.Memory[j] = byte(100 + i + j)
	}
	b.LSHSalt = uint32(i<<16 | 0x1234)
	return b
}

// slowBracket drifts slowly (stays P) and jumps hard every 90 brackets (forces I).
func slowBracket(i int) Bracket {
	b := fixtureBracket(i)
	for j := range b.Projections {
		b.Projections[j] = byte(10+j) + byte(i%3)
		if i%90 == 89 {
			b.Projections[j] = byte(255 - 7*j)
		}
	}
	return b
}

// makeNRV3 builds a file like the models's nrvio.Writer: full brackets in the
// bracket section; bracket_index "xor" = bracket XOR previous bracket (index 0:
// the bracket itself), type I for index 0 and P afterwards.
func makeNRV3(frames ...int) (*NRV3Writer, [][]Bracket) {
	fw := NewNRV3Writer()
	var all [][]Bracket
	off, n := int64(0), 0
	for fi, count := range frames {
		f := NRV3FrameEntry{
			ID: fmt.Sprintf("frame-%d", fi), TimestampUnix: 1700000000 + int64(fi), Count: count, Offset: off,
			Linguistic: NRV3LinguisticMetadata{Tokens: 12 + fi, Domain: "prose"},
			Thermo:     NRV3ThermoMetadata{Temperature: 65.25},
		}
		var buf []byte
		var bs []Bracket
		var prev [BracketSize]byte
		for k := 0; k < count; k++ {
			b := slowBracket(n)
			n++
			enc := EncodeBracket(&b)
			xor := make([]byte, BracketSize)
			typ := "P"
			if k == 0 {
				typ = "I"
				copy(xor, enc[:])
			} else {
				for j := range xor {
					xor[j] = enc[j] ^ prev[j]
				}
			}
			prev = enc
			f.BracketIndex = append(f.BracketIndex, NRV3BracketIndexEntry{Index: k, Type: typ, XOR: xor})
			buf = append(buf, enc[:]...)
			bs = append(bs, b)
		}
		fw.AddFrame(f, buf)
		all = append(all, bs)
		off += int64(count) * BracketSize
	}
	return fw, all
}

func TestNRV3RoundTrip(t *testing.T) {
	fw, _ := makeNRV3(3, 2)
	data, err := fw.Bytes()
	if err != nil {
		t.Fatal(err)
	}
	r, err := ParseNRV3(data)
	if err != nil {
		t.Fatal(err)
	}
	if r.Registry.Version != "3.0" || len(r.Registry.Frames) != 2 {
		t.Fatalf("registry = %+v", r.Registry)
	}
	f0, f1 := r.Registry.Frames[0], r.Registry.Frames[1]
	if f1.Offset != int64(f0.Count*BracketSize) {
		t.Errorf("frame 1 offset = %d", f1.Offset)
	}
	if got := len(r.BracketsForFrame(f0)); got != 3 {
		t.Errorf("frame 0 brackets = %d", got)
	}
	if got := r.RawBrackets(f1); len(got) != 2*BracketSize {
		t.Errorf("RawBrackets(frame 1) = %d bytes", len(got))
	}
	if r.RawBrackets(NRV3FrameEntry{Offset: 1 << 40, Count: 1}) != nil {
		t.Error("out-of-range frame must yield nil")
	}
}

func TestNRV3Errors(t *testing.T) {
	if _, err := ParseNRV3([]byte("NRV2\x00\x00\x00\x00")); err == nil {
		t.Fatal("NRV2 must be refused")
	}
	if _, err := ParseNRV3([]byte("AB")); err == nil {
		t.Fatal("short file must be refused")
	}
	if _, err := ParseNRV3([]byte("XXXX\x00\x00\x00\x00")); err == nil {
		t.Fatal("unknown magic must be refused")
	}
	if _, err := ParseNRV3([]byte("NRV3\xff\xff\xff\x7f")); err == nil {
		t.Fatal("oversized registry length must be refused")
	}
	fw, _ := makeNRV3(1)
	data, _ := fw.Bytes()
	if _, err := ParseNRV3(data[:len(data)-1]); err == nil {
		t.Fatal("truncated bracket section must be refused")
	}
}

func TestNRV3WriteAndReadFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "x.nrv3")
	fw, _ := makeNRV3(3, 2)
	if err := WriteNRV3(path, fw); err != nil {
		t.Fatal(err)
	}
	r, err := OpenNRV3(path)
	if err != nil {
		t.Fatal(err)
	}
	got, _ := json.Marshal(r.Registry)
	want, _ := json.Marshal(fw.registry)
	if !bytes.Equal(got, want) {
		t.Errorf("registry mismatch")
	}
}

func TestReadNRV3AtTerminatesAndHonoursOffset(t *testing.T) {
	fw, _ := makeNRV3(4)
	data, _ := fw.Bytes()
	prefix := []byte("junk-before-the-container")
	src := bytes.NewReader(append(append([]byte{}, prefix...), data...))

	done := make(chan error, 1)
	var r *NRV3Reader
	go func() {
		var err error
		r, err = ReadNRV3At(src, int64(len(prefix)))
		done <- err
	}()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("ReadNRV3At did not terminate")
	}
	if len(r.Brackets) != 4*BracketSize {
		t.Errorf("brackets = %d bytes, want %d", len(r.Brackets), 4*BracketSize)
	}
	_ = io.EOF
}

// decodeConverted undoes the KNIRVBASE delta encoding the way the reader does:
// a P-bracket's projections are XORed with its anchor's decoded projections.
func decodeConverted(t *testing.T, cf ConvertedFrame) []Bracket {
	t.Helper()
	byID := map[string]Bracket{}
	var out []Bracket
	for _, m := range cf.Index {
		var raw [BracketSize]byte
		copy(raw[:], cf.Brackets[m.Offset:m.Offset+BracketSize])
		b := DecodeBracket(raw)
		if m.Type == DeltaTypeP {
			if m.AnchorID == nil {
				t.Fatalf("P-bracket %s has no anchor", m.ID)
			}
			a, ok := byID[*m.AnchorID]
			if !ok {
				t.Fatalf("P-bracket %s anchors to %s, which is not an earlier bracket of this frame", m.ID, *m.AnchorID)
			}
			b.Projections = ApplyProjectionDelta(b.Projections, a.Projections)
		}
		byID[m.ID] = b
		out = append(out, b)
	}
	return out
}

func TestConvertFrameRoundTripsAndUsesRealDeltas(t *testing.T) {
	fw, orig := makeNRV3(200)
	data, _ := fw.Bytes()
	r, _ := ParseNRV3(data)
	f := r.Registry.Frames[0]

	frames, err := ConvertFrame(f, r.RawBrackets(f))
	if err != nil {
		t.Fatal(err)
	}
	if len(frames) != 1 || frames[0].ID != "frame-0" {
		t.Fatalf("frames = %d, id %q", len(frames), frames[0].ID)
	}
	cf := frames[0]

	// The original brackets come back exactly (this is what a KNIRVBASE reader sees).
	got := decodeConverted(t, cf)
	if len(got) != 200 {
		t.Fatalf("decoded %d brackets", len(got))
	}
	for i := range got {
		want := orig[0][i]
		if got[i] != want {
			t.Fatalf("bracket %d does not round-trip:\n got %+v\nwant %+v", i, got[i], want)
		}
	}

	// And the stream really is delta encoded: P-brackets store XORed projections,
	// every other byte untouched; I-bracket positions follow the shared policy.
	var nI, nP int
	for i, m := range cf.Index {
		if m.Type == DeltaTypeI {
			nI++
			if m.AnchorID != nil {
				t.Errorf("I-bracket %s has an anchor", m.ID)
			}
			continue
		}
		nP++
		var raw [BracketSize]byte
		copy(raw[:], cf.Brackets[m.Offset:m.Offset+BracketSize])
		stored := DecodeBracket(raw)
		if stored.Projections == orig[0][i].Projections {
			t.Errorf("P-bracket %s stores full projections (not a delta)", m.ID)
		}
		stored.Projections = orig[0][i].Projections
		if stored != orig[0][i] {
			t.Errorf("P-bracket %s changed bytes outside the projections", m.ID)
		}
	}
	if nP == 0 || nI < 4 { // 200 brackets: index 0, every 50th, and the hard jump at 89/179
		t.Errorf("I=%d P=%d: expected both kinds", nI, nP)
	}
	for i := 0; i < len(cf.Index); i += IFrameInterval {
		if cf.Index[i].Type != DeltaTypeI {
			t.Errorf("bracket %d should be an I-bracket (interval)", i)
		}
	}
	if cf.Ling.Token != "prose" || cf.Ling.Unit != "12" || cf.Thermo.AvgTempC != 65.25 {
		t.Errorf("metadata: %+v %+v", cf.Ling, cf.Thermo)
	}
}

func TestConvertFrameSplitsLargeFrames(t *testing.T) {
	n := MaxBracketsPerFrame*2 + 10
	fw, orig := makeNRV3(n)
	data, _ := fw.Bytes()
	r, _ := ParseNRV3(data)
	f := r.Registry.Frames[0]
	frames, err := ConvertFrame(f, r.RawBrackets(f))
	if err != nil {
		t.Fatal(err)
	}
	if len(frames) != 3 {
		t.Fatalf("frames = %d, want 3", len(frames))
	}
	k := 0
	for fi, cf := range frames {
		if want := fmt.Sprintf("frame-0#%d", fi+1); cf.ID != want {
			t.Errorf("frame id %q, want %q", cf.ID, want)
		}
		if cf.Index[0].Type != DeltaTypeI || cf.Index[0].Offset != 0 {
			t.Errorf("frame %d must start with an I-bracket at offset 0", fi)
		}
		for _, b := range decodeConverted(t, cf) { // also proves anchors never cross frames
			if b != orig[0][k] {
				t.Fatalf("bracket %d does not round-trip", k)
			}
			k++
		}
	}
	if k != n {
		t.Fatalf("decoded %d of %d brackets", k, n)
	}
}

func TestConvertFrameRejectsBadInput(t *testing.T) {
	fw, _ := makeNRV3(3)
	data, _ := fw.Bytes()
	r, _ := ParseNRV3(data)
	f := r.Registry.Frames[0]
	raw := r.RawBrackets(f)

	bad := f
	bad.Count = 4
	if _, err := ConvertFrame(bad, raw); err == nil {
		t.Error("count/index mismatch must fail")
	}
	if _, err := ConvertFrame(f, raw[:len(raw)-1]); err == nil {
		t.Error("short bracket data must fail")
	}
	bad = f
	bad.BracketIndex = append([]NRV3BracketIndexEntry(nil), f.BracketIndex...)
	bad.BracketIndex[1].Type = "Q"
	if _, err := ConvertFrame(bad, raw); err == nil {
		t.Error("unknown delta type must fail")
	}
}
