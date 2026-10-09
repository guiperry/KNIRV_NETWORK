package textencode

import (
	"errors"
	"testing"

	"data-encoder/pkg/nrvio"
)

func TestEncodeIsDeterministicAndShaped(t *testing.T) {
	e := New()
	rec := Record{Instruction: "Fix the memory leak", Input: "def load(p): return open(p).read()", Content: "def load(p):\n    with open(p) as fh:\n        return fh.read()"}
	a, err := e.Encode(rec)
	if err != nil {
		t.Skipf("tokenizer unavailable offline: %v", err)
	}
	b, err := e.Encode(rec)
	if err != nil {
		t.Fatal(err)
	}
	if len(a) == 0 || len(a) != len(b) {
		t.Fatalf("got %d and %d brackets", len(a), len(b))
	}
	for i := range a {
		if nrvio.EncodeBracket(a[i]) != nrvio.EncodeBracket(b[i]) {
			t.Fatalf("bracket %d differs between runs", i)
		}
	}
	if a[0].IntentFlags&0x2 == 0 {
		t.Fatal("code input should set the code intent flag")
	}
	if a[0].DomainSig != 0x3000 {
		t.Fatalf("domain = %#x, want DOMAIN_CODE", a[0].DomainSig)
	}
}

func TestEncodeRejectsOversizedAndEmptyRecords(t *testing.T) {
	e := New()
	e.MaxTokens = 4
	_, err := e.Encode(Record{Content: "one two three four five six seven"})
	if e.tkErr != nil {
		t.Skipf("tokenizer unavailable offline: %v", e.tkErr)
	}
	if !errors.Is(err, ErrTooLong) {
		t.Fatalf("err = %v, want ErrTooLong", err)
	}
	e.MaxTokens = 0
	if _, err := e.Encode(Record{Content: "x"}); err == nil {
		t.Fatal("a one-token record was accepted")
	}
}
