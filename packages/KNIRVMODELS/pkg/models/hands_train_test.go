package models

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"knirvmodels/pkg/hands"
	"knirvmodels/pkg/hands/gpt"

	"github.com/guiperry/knirv-sdk-go/models/frame"
)

type trainFrame struct {
	id     string
	valid  bool
	tokens []int
}

// writeTrainAsset writes a minimal NRV3 asset whose token_window payloads are
// embedded in the registry JSON. Only "NRV3" + the registry length + JSON are
// needed for nrvtrainer's reader.
func writeTrainAsset(t *testing.T, path string, frames []trainFrame) {
	t.Helper()
	reg := map[string]any{"frames": func() []map[string]any {
		out := make([]map[string]any, 0, len(frames))
		for _, f := range frames {
			entry := map[string]any{
				"id":             f.id,
				"timestamp_unix": int64(1),
				"count":          0,
				"offset":         0,
				"z3": map[string]any{
					"constraint_count": 1,
					"satisfiable":      f.valid,
				},
			}
			if len(f.tokens) > 0 {
				entry["payload"] = map[string]any{
					"token_window": map[string]any{"tokens": f.tokens},
				}
			}
			out = append(out, entry)
		}
		return out
	}()}
	body, err := json.Marshal(reg)
	if err != nil {
		t.Fatalf("marshal registry: %v", err)
	}
	buf := bytes.NewBuffer(nil)
	buf.WriteString("NRV3")
	if err := binary.Write(buf, binary.LittleEndian, uint32(len(body))); err != nil {
		t.Fatalf("write length: %v", err)
	}
	buf.Write(body)
	if err := os.WriteFile(path, buf.Bytes(), 0o600); err != nil {
		t.Fatalf("write asset: %v", err)
	}
}

func mountTrainHands(t *testing.T, gen generator) *httptest.Server {
	t.Helper()
	h := NewHandsWith(func() (generator, error) { return gen, nil },
		nil, nil, nil, frame.Default())
	mux := http.NewServeMux()
	h.Mount(mux, "/models/hands")
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func postTrain(t *testing.T, srv *httptest.Server, body string) *http.Response {
	t.Helper()
	resp, err := http.Post(srv.URL+"/models/hands/train", "application/json", bytes.NewBufferString(body))
	if err != nil {
		t.Fatalf("post train: %v", err)
	}
	t.Cleanup(func() { resp.Body.Close() })
	return resp
}

func waitForJob(t *testing.T, srv *httptest.Server) map[string]any {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		resp, err := http.Get(srv.URL + "/models/hands/status")
		if err != nil {
			t.Fatalf("status: %v", err)
		}
		var payload struct {
			Training map[string]any `json:"training"`
		}
		_ = json.NewDecoder(resp.Body).Decode(&payload)
		resp.Body.Close()
		if payload.Training != nil && payload.Training["state"] != "running" {
			return payload.Training
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("training job did not finish")
	return nil
}

func TestHandsTrainTrainsAndSaves(t *testing.T) {
	t.Setenv("KNIRV_HANDS_ENABLED", "1")
	dir := t.TempDir()
	t.Setenv("KNIRV_HANDS_DIR", dir)
	asset := filepath.Join(t.TempDir(), "corpus.nrv")
	writeTrainAsset(t, asset, []trainFrame{
		{id: "f1", valid: true, tokens: []int{1, 2, 3, 4, 5, 6}},
		{id: "f2", valid: true, tokens: []int{6, 5, 4, 3, 2, 1}},
		{id: "f3", valid: false, tokens: []int{1, 2, 3, 4, 5, 6}},
	})

	srv := mountTrainHands(t, newTrainEngine(t))
	body := `{"input":` + strconv.Quote(asset) + `,"maxSteps":2,"contextLength":4}`
	resp := postTrain(t, srv, body)
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("train status = %d, want 202", resp.StatusCode)
	}
	var job TrainJob
	if err := json.NewDecoder(resp.Body).Decode(&job); err != nil {
		t.Fatalf("decode job: %v", err)
	}
	if job.ID == "" || job.State != "running" {
		t.Fatalf("job = %+v, want running with id", job)
	}

	final := waitForJob(t, srv)
	if final["state"] != "succeeded" {
		t.Fatalf("final state = %v (%v), want succeeded", final["state"], final["error"])
	}
	if _, err := os.Stat(filepath.Join(dir, "model-latest")); err != nil {
		t.Fatalf("checkpoint not saved: %v", err)
	}
}

func newTrainEngine(t *testing.T) generator {
	t.Helper()
	return hands.NewEngine(&gpt.Config{
		VocabSize: 16, EmbedDim: 8, NumHeads: 2, NumLayers: 1,
		ContextLen: 16, FFNHiddenDim: 16, DecayAlpha: 1.0,
	}, "")
}

func TestHandsTrainRejectsBadRequests(t *testing.T) {
	t.Setenv("KNIRV_HANDS_ENABLED", "1")
	dir := t.TempDir()
	t.Setenv("KNIRV_HANDS_DIR", dir)
	asset := filepath.Join(dir, "corpus.nrv")
	writeTrainAsset(t, asset, []trainFrame{{id: "f1", valid: true, tokens: []int{1, 2, 3, 4}}})
	srv := mountTrainHands(t, newTrainEngine(t))

	if resp, err := http.Get(srv.URL + "/models/hands/train"); err != nil {
		t.Fatal(err)
	} else if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("GET status = %d, want 405", resp.StatusCode)
	}

	cases := []struct {
		name string
		body string
		want int
	}{
		{"malformed", `{`, http.StatusBadRequest},
		{"missing input", `{}`, http.StatusBadRequest},
		{"relative input", `{"input":"corpus.nrv"}`, http.StatusBadRequest},
		{"wrong extension", `{"input":"/tmp/corpus.json"}`, http.StatusBadRequest},
		{"missing file", `{"input":"/tmp/does-not-exist.nrv"}`, http.StatusNotFound},
		{"epochs ceiling", `{"input":` + strconv.Quote(asset) + `,"epochs":101}`, http.StatusBadRequest},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resp := postTrain(t, srv, tc.body)
			if resp.StatusCode != tc.want {
				t.Fatalf("status = %d, want %d", resp.StatusCode, tc.want)
			}
		})
	}
}

func TestHandsTrainRejectsAssetWithoutValidFrames(t *testing.T) {
	t.Setenv("KNIRV_HANDS_ENABLED", "1")
	t.Setenv("KNIRV_HANDS_DIR", t.TempDir())
	asset := filepath.Join(t.TempDir(), "invalid.nrv")
	writeTrainAsset(t, asset, []trainFrame{
		{id: "f1", valid: false, tokens: []int{1, 2, 3, 4}},
	})
	srv := mountTrainHands(t, newTrainEngine(t))
	resp := postTrain(t, srv, `{"input":`+strconv.Quote(asset)+`,"contextLength":4}`)
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422", resp.StatusCode)
	}
}

type blockingTrainer struct {
	scriptedGen
	started chan struct{}
	release chan struct{}
}

func (b *blockingTrainer) TrainWindows(windows [][]int, opts hands.TrainOptions) (hands.TrainResult, error) {
	close(b.started)
	<-b.release
	return hands.TrainResult{Windows: len(windows), Steps: 1}, nil
}

func TestHandsTrainConflictAndGenerationBlocked(t *testing.T) {
	t.Setenv("KNIRV_HANDS_ENABLED", "1")
	t.Setenv("KNIRV_HANDS_DIR", t.TempDir())
	asset := filepath.Join(t.TempDir(), "corpus.nrv")
	writeTrainAsset(t, asset, []trainFrame{{id: "f1", valid: true, tokens: []int{1, 2, 3, 4}}})

	gen := &blockingTrainer{
		scriptedGen: scriptedGen{ready: true},
		started:     make(chan struct{}),
		release:     make(chan struct{}),
	}
	srv := mountTrainHands(t, gen)

	body := `{"input":` + strconv.Quote(asset) + `,"contextLength":4,"maxSteps":1}`
	if resp := postTrain(t, srv, body); resp.StatusCode != http.StatusAccepted {
		t.Fatalf("first train status = %d, want 202", resp.StatusCode)
	}
	<-gen.started

	if resp := postTrain(t, srv, body); resp.StatusCode != http.StatusConflict {
		t.Fatalf("second train status = %d, want 409", resp.StatusCode)
	}

	gresp, err := http.Post(srv.URL+"/models/hands/generate", "application/json",
		bytes.NewBufferString(`{"prompt":"hi","maxTokens":4}`))
	if err != nil {
		t.Fatal(err)
	}
	gresp.Body.Close()
	if gresp.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("generate during training = %d, want 503", gresp.StatusCode)
	}

	close(gen.release)
	final := waitForJob(t, srv)
	if final["state"] != "succeeded" {
		t.Fatalf("final state = %v, want succeeded", final["state"])
	}
}

func TestHandsTrainRejectsAssetOverFrameLimit(t *testing.T) {
	t.Setenv("KNIRV_HANDS_ENABLED", "1")
	t.Setenv("KNIRV_HANDS_DIR", t.TempDir())
	frames := make([]trainFrame, maxTrainFrames+1)
	for i := range frames {
		frames[i] = trainFrame{id: "f", valid: true}
	}
	asset := filepath.Join(t.TempDir(), "too-many-frames.nrv")
	writeTrainAsset(t, asset, frames)
	srv := mountTrainHands(t, newTrainEngine(t))
	resp := postTrain(t, srv, `{"input":`+strconv.Quote(asset)+`,"contextLength":4}`)
	if resp.StatusCode != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want 413 for %d frames", resp.StatusCode, len(frames))
	}
}

func TestHandsTrainRejectsAssetOverWindowLimit(t *testing.T) {
	t.Setenv("KNIRV_HANDS_ENABLED", "1")
	t.Setenv("KNIRV_HANDS_DIR", t.TempDir())
	// contextLength 2 / stride 2 over 2*(maxTrainWindows+1) tokens yields maxTrainWindows+1 windows.
	tokens := make([]int, 2*(maxTrainWindows+1))
	for i := range tokens {
		tokens[i] = i % 16
	}
	asset := filepath.Join(t.TempDir(), "too-many-windows.nrv")
	writeTrainAsset(t, asset, []trainFrame{{id: "big", valid: true, tokens: tokens}})
	srv := mountTrainHands(t, newTrainEngine(t))
	resp := postTrain(t, srv, `{"input":`+strconv.Quote(asset)+`,"contextLength":2,"stride":2}`)
	if resp.StatusCode != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want 413 for %d windows", resp.StatusCode, maxTrainWindows+1)
	}
}

func TestHandsTrainRejectsOutOfVocabularyTokens(t *testing.T) {
	t.Setenv("KNIRV_HANDS_ENABLED", "1")
	t.Setenv("KNIRV_HANDS_DIR", t.TempDir())
	asset := filepath.Join(t.TempDir(), "oov.nrv")
	// newTrainEngine has VocabSize 16; 99 must be rejected, not wrapped to 3.
	writeTrainAsset(t, asset, []trainFrame{{id: "bad", valid: true, tokens: []int{1, 2, 99, 4}}})
	srv := mountTrainHands(t, newTrainEngine(t))
	resp := postTrain(t, srv, `{"input":`+strconv.Quote(asset)+`,"contextLength":4}`)
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422 for out-of-vocabulary token", resp.StatusCode)
	}
}
