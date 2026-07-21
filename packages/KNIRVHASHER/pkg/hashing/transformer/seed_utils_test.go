package transformer

import (
	"testing"

	"knirvhasher/pkg/hashing/core"
)

func TestSeedToFloat(t *testing.T) {
	seed := [32]byte{0xFF, 0xFF, 0xFF, 0xFF}
	val := SeedToFloat(seed)
	if val != 1.0 {
		t.Fatalf("expected 1.0, got %f", val)
	}

	seed = [32]byte{0x00, 0x00, 0x00, 0x00}
	val = SeedToFloat(seed)
	if val != 0.0 {
		t.Fatalf("expected 0.0, got %f", val)
	}
}

func TestProjectSeeds(t *testing.T) {
	input := []float32{0.5, 0.5}
	seeds := [][32]byte{{0xAB}}
	out := ProjectSeeds(input, seeds, "hash")
	if len(out) != 1 {
		t.Fatalf("expected length 1, got %d", len(out))
	}
}

func TestProjectSeeds2D(t *testing.T) {
	input := []float32{0.5, 0.5}
	seeds := [][][32]byte{{{0xAB}}}
	out := ProjectSeeds2D(input, seeds, "hash")
	if len(out) != 1 {
		t.Fatalf("expected length 1, got %d", len(out))
	}
}

func TestProjectBack(t *testing.T) {
	input := []float32{1.0, 2.0, 3.0}
	out := ProjectBack(input, 2, "hash")
	if len(out) != 2 {
		t.Fatalf("expected length 2, got %d", len(out))
	}
}

func TestHashToVocab(t *testing.T) {
	hidden := []float32{0.1, 0.2, 0.3}
	var seed [32]byte
	scores := HashToVocab(hidden, seed, 10)
	if len(scores) != 10 {
		t.Fatalf("expected length 10, got %d", len(scores))
	}
}

func TestLayerNorm(t *testing.T) {
	if LayerNorm(5.0, -10.0, 10.0) != 5.0 {
		t.Error("expected 5.0")
	}
	if LayerNorm(-20.0, -10.0, 10.0) != -10.0 {
		t.Error("expected -10.0")
	}
	if LayerNorm(20.0, -10.0, 10.0) != 10.0 {
		t.Error("expected 10.0")
	}
}

func TestArgmax32(t *testing.T) {
	s := []float32{0.1, 0.9, 0.5}
	if Argmax32(s) != 1 {
		t.Fatalf("expected 1, got %d", Argmax32(s))
	}
}

func TestSampleTemp32(t *testing.T) {
	scores := []float32{0.1, 0.9}
	result := SampleTemp32(scores, 1.0)
	if result != 0 && result != 1 {
		t.Fatalf("expected 0 or 1, got %d", result)
	}
}

func TestSortFloat32(t *testing.T) {
	s := []float32{3.0, 1.0, 2.0}
	SortFloat32(s)
	if s[0] != 1.0 || s[1] != 2.0 || s[2] != 3.0 {
		t.Fatalf("unexpected sort result: %v", s)
	}
}

type fakeHashMethod struct{}

func (f *fakeHashMethod) Name() string                                    { return "fake" }
func (f *fakeHashMethod) IsAvailable() bool                               { return false }
func (f *fakeHashMethod) Initialize() error                              { return nil }
func (f *fakeHashMethod) Shutdown() error                                { return nil }
func (f *fakeHashMethod) ComputeHash(data []byte) ([32]byte, error)      { return [32]byte{}, nil }
func (f *fakeHashMethod) ComputeBatch(data [][]byte) ([][32]byte, error) { return nil, nil }
func (f *fakeHashMethod) MineHeader(header []byte, nonceStart, nonceEnd uint32) (uint32, error) {
	return 0, nil
}
func (f *fakeHashMethod) MineHeaderBatch(headers [][]byte, nonceStart, nonceEnd uint32) ([]uint32, error) {
	return nil, nil
}
func (f *fakeHashMethod) GetCapabilities() *core.Capabilities { return nil }
func (f *fakeHashMethod) Execute21PassLoop(header []byte, targetTokenID uint32) (*core.JitterResult, error) {
	return nil, nil
}
func (f *fakeHashMethod) Execute21PassLoopBatch(headers [][]byte, targetTokenID uint32) ([]*core.JitterResult, error) {
	return nil, nil
}
func (f *fakeHashMethod) ExecuteRecursiveMine(header []byte, passes int) ([]byte, error) {
	return nil, nil
}
func (f *fakeHashMethod) LoadJitterTable(table map[uint32]uint32) error { return nil }
func (f *fakeHashMethod) GetJitterStats() map[string]interface{}       { return nil }
