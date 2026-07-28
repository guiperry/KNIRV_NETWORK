package nrvio

import "testing"

func TestBracketEncodeDecodeRoundTrip(t *testing.T) {
	var want Bracket
	for i := range want.Projections {
		want.Projections[i] = byte(i)
	}
	want.SubSecondUS, want.POSTag, want.Tense, want.DepHead, want.DomainSig, want.GoldenSeed, want.LSHSalt = 42, 3, 2, 7, 0x2000, 99, 123
	for i := range want.Memory {
		want.Memory[i] = byte(i + 10)
	}
	got := DecodeBracket(EncodeBracket(want))
	if got != want {
		t.Fatalf("round trip mismatch: %#v != %#v", got, want)
	}
}
