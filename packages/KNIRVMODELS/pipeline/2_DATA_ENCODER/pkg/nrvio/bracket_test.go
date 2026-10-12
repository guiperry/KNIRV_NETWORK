package nrvio

import "testing"

func TestBracketEncodeDecodeRoundTrip(t *testing.T) {
	var want Bracket
	for i := range want.Projections {
		want.Projections[i] = byte(i)
	}
	want.SubSecondUS, want.Syntactic, want.DepHead, want.DomainSig, want.GoldenSeed, want.LSHSalt = 42, PackSyntactic(3, 2, 1), -7, 0x2000, 99, 123
	for i := range want.Memory {
		want.Memory[i] = byte(i + 10)
	}
	got := DecodeBracket(EncodeBracket(want))
	if got != want {
		t.Fatalf("round trip mismatch: %#v != %#v", got, want)
	}
}

func TestBracketUsesCanonicalKNIRVBASEOffsets(t *testing.T) {
	b := Bracket{SubSecondUS: 0x01020304, Syntactic: 0xa5, DepHead: -2, IntentFlags: 0x7f, DomainSig: 0x2001, GoldenSeed: 0x11223344, LSHSalt: 0x55667788}
	raw := EncodeBracket(b)
	if raw[36] != 0xa5 || raw[37] != 0xfe || raw[38] != 0x7f {
		t.Fatalf("canonical scalar offsets were not used: %x", raw[36:39])
	}
	if got := raw[39:41]; got[0] != 0x01 || got[1] != 0x20 {
		t.Fatalf("domain offset = %x, want 0120", got)
	}
	if got := raw[41:45]; got[0] != 0x44 || got[3] != 0x11 {
		t.Fatalf("golden-seed offset = %x", got)
	}
	if got := raw[59:63]; got[0] != 0x88 || got[3] != 0x55 {
		t.Fatalf("lsh-salt offset = %x", got)
	}
}
