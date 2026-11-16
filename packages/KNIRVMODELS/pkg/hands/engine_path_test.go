package hands

import (
	"path/filepath"
	"sync"
	"testing"

	"knirvmodels/pkg/hands/gpt"
)

// TestEnginePathReadDuringTrainWindows exercises the race between the training
// goroutine (which records the checkpoint path) and concurrent Path() readers
// such as the HANDS weights handler. Run with -race.
func TestEnginePathReadDuringTrainWindows(t *testing.T) {
	e := NewEngine(&gpt.Config{
		VocabSize: 16, EmbedDim: 8, NumHeads: 2, NumLayers: 1,
		ContextLen: 16, FFNHiddenDim: 16, DecayAlpha: 1.0,
	}, "")
	savePath := filepath.Join(t.TempDir(), "model-latest")
	windows := [][]int{{1, 2, 3, 4}, {4, 3, 2, 1}}

	stop := make(chan struct{})
	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-stop:
					return
				default:
					_ = e.Path()
				}
			}
		}()
	}

	res, err := e.TrainWindows(windows, TrainOptions{Epochs: 1, LearningRate: 1e-3, MaxSteps: 2, SavePath: savePath})
	close(stop)
	wg.Wait()
	if err != nil {
		t.Fatalf("TrainWindows: %v", err)
	}
	if res.Checkpoint != savePath {
		t.Fatalf("checkpoint = %q, want %q", res.Checkpoint, savePath)
	}
	if got := e.Path(); got != savePath {
		t.Fatalf("Path() = %q, want %q", got, savePath)
	}
}
