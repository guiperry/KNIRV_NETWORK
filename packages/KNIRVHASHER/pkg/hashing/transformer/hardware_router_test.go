package transformer

import (
	"fmt"
	"testing"

	"knirvhasher/pkg/hashing/core"
)

func TestHardwareRouter_Project_SoftwareFallback(t *testing.T) {
	router := NewHardwareRouter(nil, FallbackSoftware)
	out, err := router.Project([]float32{0.5, 0.5}, [][32]byte{{0xAB}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out) != 1 {
		t.Fatalf("expected 1 output, got %d", len(out))
	}
}

func TestHardwareRouter_Project_HardwareSuccess(t *testing.T) {
	method := &fakeHashMethod{}
	router := NewHardwareRouter(method, FallbackSoftware)
	out, err := router.Project([]float32{0.5, 0.5}, [][32]byte{{0xAB}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out) != 1 {
		t.Fatalf("expected 1 output, got %d", len(out))
	}
}

func TestHardwareRouter_Project_HardwareFailureFallsBack(t *testing.T) {
	method := &failingHashMethod{}
	router := NewHardwareRouter(method, FallbackMixed)
	out, err := router.Project([]float32{0.5, 0.5}, [][32]byte{{0xAB}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out) != 1 {
		t.Fatalf("expected 1 output, got %d", len(out))
	}
}

func TestHardwareRouter_Project_HardwareFailureErrors(t *testing.T) {
	method := &failingHashMethod{}
	router := NewHardwareRouter(method, FallbackError)
	_, err := router.Project([]float32{0.5, 0.5}, [][32]byte{{0xAB}})
	if err == nil {
		t.Fatal("expected error with FallbackError strategy")
	}
}

func TestHardwareRouter_HashToVocab(t *testing.T) {
	method := &fakeHashMethod{}
	router := NewHardwareRouter(method, FallbackSoftware)
	scores := router.HashToVocab([]float32{0.1, 0.2}, [32]byte{}, 10)
	if len(scores) != 10 {
		t.Fatalf("expected 10 scores, got %d", len(scores))
	}
}

func TestHardwareRouter_ProjectBatch(t *testing.T) {
	method := &fakeHashMethod{}
	router := NewHardwareRouter(method, FallbackSoftware)
	inputs := [][]float32{{0.5}, {0.6}}
	seeds := [][][32]byte{{{0xAB}}, {{0xCD}}}
	out, err := router.ProjectBatch(inputs, seeds)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out) != 2 {
		t.Fatalf("expected 2 outputs, got %d", len(out))
	}
}

func TestHardwareRouter_ProjectBatch_Mismatch(t *testing.T) {
	method := &fakeHashMethod{}
	router := NewHardwareRouter(method, FallbackSoftware)
	inputs := [][]float32{{0.5}}
	seeds := [][][32]byte{{{0xAB}}, {{0xCD}}}
	_, err := router.ProjectBatch(inputs, seeds)
	if err == nil {
		t.Fatal("expected mismatch error")
	}
}

type failingHashMethod struct{}

func (f *failingHashMethod) Name() string                                    { return "failing" }
func (f *failingHashMethod) IsAvailable() bool                               { return true }
func (f *failingHashMethod) Initialize() error                              { return nil }
func (f *failingHashMethod) Shutdown() error                                { return nil }
func (f *failingHashMethod) ComputeHash(data []byte) ([32]byte, error)      { return [32]byte{}, nil }
func (f *failingHashMethod) ComputeBatch(data [][]byte) ([][32]byte, error) {
	return nil, fmt.Errorf("simulated batch failure")
}
func (f *failingHashMethod) MineHeader(header []byte, nonceStart, nonceEnd uint32) (uint32, error) {
	return 0, nil
}
func (f *failingHashMethod) MineHeaderBatch(headers [][]byte, nonceStart, nonceEnd uint32) ([]uint32, error) {
	return nil, nil
}
func (f *failingHashMethod) GetCapabilities() *core.Capabilities { return nil }
func (f *failingHashMethod) Execute21PassLoop(header []byte, targetTokenID uint32) (*core.JitterResult, error) {
	return nil, nil
}
func (f *failingHashMethod) Execute21PassLoopBatch(headers [][]byte, targetTokenID uint32) ([]*core.JitterResult, error) {
	return nil, nil
}
func (f *failingHashMethod) ExecuteRecursiveMine(header []byte, passes int) ([]byte, error) {
	return nil, nil
}
func (f *failingHashMethod) LoadJitterTable(table map[uint32]uint32) error { return nil }
func (f *failingHashMethod) GetJitterStats() map[string]interface{}       { return nil }
