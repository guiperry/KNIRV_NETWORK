package semantic

import "testing"

func TestLSHMapperReproducibility(t *testing.T) {
	e := make([]float32, 768)
	e[0] = 1
	a := NewSemanticMapper(1337).MapToBracket(e, RecordMetadata{Text: "solve x = 1", DatasetID: "d"})
	b := NewSemanticMapper(1337).MapToBracket(e, RecordMetadata{Text: "solve x = 1", DatasetID: "d"})
	if a != b {
		t.Fatal("same seed and input must be deterministic")
	}
	if a.DomainSig != 0x2000 {
		t.Fatalf("domain=%x", a.DomainSig)
	}
}
