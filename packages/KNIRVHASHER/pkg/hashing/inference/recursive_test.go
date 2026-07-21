package inference

import (
	"testing"

	"knirvhasher/pkg/hashing"
	"knirvhasher/pkg/hashing/core"
	"knirvhasher/pkg/hashing/neural"
	"knirvhasher/pkg/hashing/transformer"
)

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

type availableHash struct{ fakeHashMethod }

func (f *availableHash) IsAvailable() bool { return true }
func (f *availableHash) ComputeBatch(data [][]byte) ([][32]byte, error) {
	result := make([][32]byte, len(data))
	for i := range result {
		for j := range result[i] {
			result[i][j] = byte(i * j)
		}
	}
	return result, nil
}

func TestRecursiveEngine_Creation(t *testing.T) {
	net, err := neural.NewHashNetwork(8, 4, 4, 4)
	if err != nil {
		t.Fatalf("failed to create network: %v", err)
	}
	engine, err := NewRecursiveEngineWithHashMethod(net, nil, 3, 0.01, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if engine.Passes != 3 {
		t.Fatalf("expected 3 passes, got %d", engine.Passes)
	}
}

func TestRecursiveEngine_SetMode(t *testing.T) {
	net, err := neural.NewHashNetwork(8, 4, 4, 4)
	if err != nil {
		t.Fatalf("failed to create network: %v", err)
	}
	engine, err := NewRecursiveEngineWithHashMethod(net, nil, 3, 0.01, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	engine.SetMode(hashing.ModeTransformer)
	if engine.Mode != hashing.ModeTransformer {
		t.Fatalf("expected transformer mode, got %s", engine.Mode)
	}
}

func TestRecursiveEngine_RecursiveMode_Infer(t *testing.T) {
	net, err := neural.NewHashNetwork(8, 4, 4, 4)
	if err != nil {
		t.Fatalf("failed to create network: %v", err)
	}
	engine, err := NewRecursiveEngineWithHashMethod(net, nil, 3, 0.0, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	result, err := engine.Infer([]byte("hello world"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Consensus == nil {
		t.Fatal("expected non-nil consensus")
	}
	if result.ValidPasses == 0 {
		t.Fatal("expected at least one valid pass")
	}
}

func TestRecursiveEngine_FeedforwardMode_Infer(t *testing.T) {
	net, err := neural.NewHashNetwork(8, 4, 4, 4)
	if err != nil {
		t.Fatalf("failed to create network: %v", err)
	}
	engine, err := NewRecursiveEngineWithHashMethod(net, nil, 3, 0.0, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	engine.SetMode(hashing.ModeFeedforward)
	result, err := engine.Infer([]byte("hello world"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Consensus == nil {
		t.Fatal("expected non-nil consensus")
	}
}

func TestRecursiveEngine_TransformerMode_Infer(t *testing.T) {
	net, err := neural.NewHashNetwork(8, 4, 4, 4)
	if err != nil {
		t.Fatalf("failed to create network: %v", err)
	}
	engine, err := NewRecursiveEngineWithHashMethod(net, nil, 3, 0.0, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	tok, err := transformer.NewTiktokenTokenizer("cl100k_base")
	if err == nil {
		engine.Tokenizer = tok
	}
	engine.SetMode(hashing.ModeTransformer)
	result, err := engine.Infer([]byte("hello"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Consensus == nil {
		t.Fatal("expected non-nil consensus")
	}
}

func TestRecursiveEngine_TransformerMode_NoTokenizer(t *testing.T) {
	net, err := neural.NewHashNetwork(8, 4, 4, 4)
	if err != nil {
		t.Fatalf("failed to create network: %v", err)
	}
	engine, err := NewRecursiveEngineWithHashMethod(net, nil, 3, 0.0, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	engine.SetMode(hashing.ModeTransformer)
	_, err = engine.Infer([]byte("hello"))
	if err == nil {
		t.Fatal("expected error without tokenizer")
	}
}

func TestRecursiveEngine_NilNetwork(t *testing.T) {
	_, err := NewRecursiveEngineWithHashMethod(nil, nil, 3, 0.01, false)
	if err == nil {
		t.Fatal("expected error for nil network")
	}
}

func TestRecursiveEngine_DefaultPasses(t *testing.T) {
	net, _ := neural.NewHashNetwork(8, 4, 4, 4)
	engine, _ := NewRecursiveEngineWithHashMethod(net, nil, 0, 0.01, false)
	if engine.Passes != 21 {
		t.Fatalf("expected default 21 passes, got %d", engine.Passes)
	}
}

func TestRecursiveEngine_DefaultJitter(t *testing.T) {
	net, _ := neural.NewHashNetwork(8, 4, 4, 4)
	engine, _ := NewRecursiveEngineWithHashMethod(net, nil, 3, 1.5, false)
	if engine.Jitter != 0.01 {
		t.Fatalf("expected default 0.01 jitter, got %f", engine.Jitter)
	}
}

func TestRecursiveEngine_HardwareAccelerated(t *testing.T) {
	net, _ := neural.NewHashNetwork(8, 4, 4, 4)
	method := &availableHash{}
	engine, _ := NewRecursiveEngineWithHashMethod(net, method, 3, 0.0, false)
	if !engine.IsUsingHardware() {
		t.Fatal("expected hardware to be available")
	}
	result, err := engine.Infer([]byte("test"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ValidPasses == 0 {
		t.Fatal("expected at least one valid pass")
	}
}
