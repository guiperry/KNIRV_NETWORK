package dveevidence

import "testing"

func TestCanonicalMerkleNormalizationAndStrictEventFields(t *testing.T) {
	if MerkleRoot([]string{"aaa", "bbb"}) != MerkleRoot([]string{"sha256:aaa", "sha256:bbb"}) {
		t.Fatal("raw and prefixed artifact hashes normalized differently")
	}
	events, root, err := BuildEventLog([]Event{{Timestamp: "t0", Type: "start"}, {Timestamp: "t1", Type: "end"}})
	if err != nil {
		t.Fatal(err)
	}
	missingPrev := append([]Event(nil), events...)
	missingPrev[1].PrevHash = ""
	missingHash := append([]Event(nil), events...)
	missingHash[0].Hash = ""
	if err := VerifyEventLog(missingHash, root); err == nil {
		t.Fatal("missing event hash was accepted")
	}
	if err := VerifyEventLog(missingPrev, root); err == nil {
		t.Fatal("missing prev_hash was accepted")
	}
}
