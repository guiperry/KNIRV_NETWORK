package tokenizer_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"knirvmodels/pkg/hands/tokenizer"
)

// fixtureVocab is a fixed word->id vocab. <UNK> is id 0 so unknown words have a
// known id to assert against.
var fixtureVocab = map[string]int{
	"<UNK>":  0,
	"hello":  1,
	"world":  2,
	"héllo":  3,
	"日本語":    4,
	"🙂":      5,
	"knirv":  6,
	"tokens": 7,
}

// writeVocab writes vocab as the JSON {word: id} file NewBPETokenizer expects.
func writeVocab(t *testing.T, vocab map[string]int) string {
	t.Helper()
	data, err := json.Marshal(vocab)
	if err != nil {
		t.Fatalf("marshal vocab: %v", err)
	}
	path := filepath.Join(t.TempDir(), "vocab.json")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write vocab: %v", err)
	}
	return path
}

func newFixtureTokenizer(t *testing.T) *tokenizer.BPETokenizer {
	t.Helper()
	return tokenizer.NewBPETokenizer(writeVocab(t, fixtureVocab))
}

// The exported BPETokenizer must satisfy the Tokenizer contract.
var _ tokenizer.Tokenizer = (*tokenizer.BPETokenizer)(nil)

func TestBPEVocabSize(t *testing.T) {
	tok := newFixtureTokenizer(t)
	if got, want := tok.VocabSize(), len(fixtureVocab); got != want {
		t.Fatalf("VocabSize() = %d, want %d", got, want)
	}
}

func TestBPEVocabSizeMissingFileIsEmpty(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "does-not-exist.json")
	tok := tokenizer.NewBPETokenizer(missing)
	if got := tok.VocabSize(); got != 0 {
		t.Fatalf("VocabSize() for missing file = %d, want 0", got)
	}
	if got := tok.Encode(""); len(got) != 0 {
		t.Fatalf("Encode(\"\") on empty vocab = %v, want empty", got)
	}
}

func TestBPEEncodeDecodeRoundTrip(t *testing.T) {
	tok := newFixtureTokenizer(t)
	cases := []struct {
		name string
		text string
		ids  []int
	}{
		{"ascii words", "hello world", []int{1, 2}},
		{"single word", "knirv", []int{6}},
		{"accented latin", "héllo world", []int{3, 2}},
		{"cjk", "日本語", []int{4}},
		{"emoji", "hello 🙂", []int{1, 5}},
		{"mixed unicode", "knirv 日本語 🙂 tokens", []int{6, 4, 5, 7}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ids := tok.Encode(c.text)
			if !reflect.DeepEqual(ids, c.ids) {
				t.Fatalf("Encode(%q) = %v, want %v", c.text, ids, c.ids)
			}
			if got := tok.Decode(ids); got != c.text {
				t.Fatalf("Decode(Encode(%q)) = %q, want exact round trip", c.text, got)
			}
		})
	}
}

// Whitespace is not preserved: Encode splits on strings.Fields, so runs of
// spaces/tabs/newlines collapse to one space on the way back. Pin that behaviour.
func TestBPEWhitespaceCollapsesOnRoundTrip(t *testing.T) {
	tok := newFixtureTokenizer(t)
	ids := tok.Encode("  hello\t\n world  ")
	if want := []int{1, 2}; !reflect.DeepEqual(ids, want) {
		t.Fatalf("Encode = %v, want %v", ids, want)
	}
	if got, want := tok.Decode(ids), "hello world"; got != want {
		t.Fatalf("Decode = %q, want %q", got, want)
	}
}

func TestBPEEmptyInput(t *testing.T) {
	tok := newFixtureTokenizer(t)
	if ids := tok.Encode(""); len(ids) != 0 {
		t.Fatalf("Encode(\"\") = %v, want no tokens", ids)
	}
	if ids := tok.Encode("   \t\n"); len(ids) != 0 {
		t.Fatalf("Encode(whitespace only) = %v, want no tokens", ids)
	}
	if got := tok.Decode(nil); got != "" {
		t.Fatalf("Decode(nil) = %q, want empty string", got)
	}
	if got := tok.Decode([]int{}); got != "" {
		t.Fatalf("Decode([]) = %q, want empty string", got)
	}
}

func TestBPEUnknownWordMapsToUNK(t *testing.T) {
	tok := newFixtureTokenizer(t)
	ids := tok.Encode("hello zzz")
	if want := []int{1, fixtureVocab["<UNK>"]}; !reflect.DeepEqual(ids, want) {
		t.Fatalf("Encode(\"hello zzz\") = %v, want %v", ids, want)
	}
	if got, want := tok.Decode(ids), "hello <UNK>"; got != want {
		t.Fatalf("Decode(unknown word ids) = %q, want %q", got, want)
	}
}

func TestBPEDecodeUnknownIDIsUNK(t *testing.T) {
	tok := newFixtureTokenizer(t)
	if got, want := tok.Decode([]int{1, 999, -1}), "hello <UNK> <UNK>"; got != want {
		t.Fatalf("Decode(unknown ids) = %q, want %q", got, want)
	}
}

func TestBPEEncodeIsDeterministic(t *testing.T) {
	path := writeVocab(t, fixtureVocab)
	a := tokenizer.NewBPETokenizer(path)
	b := tokenizer.NewBPETokenizer(path)
	const text = "knirv tokens hello world 日本語 zzz"
	first := a.Encode(text)
	for i := 0; i < 20; i++ {
		if got := a.Encode(text); !reflect.DeepEqual(got, first) {
			t.Fatalf("repeat %d: Encode = %v, first = %v", i, got, first)
		}
	}
	if got := b.Encode(text); !reflect.DeepEqual(got, first) {
		t.Fatalf("second tokenizer from same vocab: Encode = %v, want %v", got, first)
	}
	if a.Decode(first) != b.Decode(first) {
		t.Fatalf("Decode differs between identical tokenizers")
	}
}

func TestTiktokenUnknownEncodingErrors(t *testing.T) {
	// An unknown encoding name is rejected before any BPE file is fetched, so this
	// runs fully offline.
	if _, err := tokenizer.NewTiktokenTokenizer("no_such_encoding"); err == nil {
		t.Fatalf("NewTiktokenTokenizer(unknown) returned nil error")
	}
}
