package inferencer

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	models "github.com/guiperry/knirv-sdk-go/models"
)

type fakeExternal struct{ calls atomic.Int32 }

func (f *fakeExternal) GenerateText(string) (string, error) {
	f.calls.Add(1)
	return "external", nil
}

// daemon fakes the models daemon's HEART routes.
func daemon(t *testing.T, ready bool, healthHits, generateHits *atomic.Int32) *models.Client {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc(models.HeartPrefix+"/health", func(w http.ResponseWriter, _ *http.Request) {
		healthHits.Add(1)
		if ready {
			_, _ = w.Write([]byte(`{"available":true,"status":"online","loaded":true,"ready":true,"progress":1}`))
			return
		}
		_, _ = w.Write([]byte(`{"available":true,"status":"idle","loaded":false,"ready":false,"progress":0}`))
	})
	mux.HandleFunc(models.HeartPrefix+"/generate", func(w http.ResponseWriter, _ *http.Request) {
		generateHits.Add(1)
		if !ready {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte(`{"error":"model not trained","ready":false}`))
			return
		}
		_, _ = w.Write([]byte(`{"text":"internal","ready":true}`))
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return models.NewClient(srv.URL, nil)
}

func TestHeartGeneratorReadyRoutesInternal(t *testing.T) {
	var h, g atomic.Int32
	gen := NewHeartTextGenerator(daemon(t, true, &h, &g))
	ext := &fakeExternal{}
	sw := NewInferenceSwitcher(gen, ext)
	got, err := sw.GenerateText(context.Background(), "hi")
	if err != nil || got != "internal" || ext.calls.Load() != 0 {
		t.Fatalf("got %q %v (external calls %d)", got, err, ext.calls.Load())
	}
	if gen.Progress() != 1 {
		t.Fatalf("progress %v", gen.Progress())
	}
}

func TestHeartGeneratorIdleFallsThroughToExternalWithoutGenerating(t *testing.T) {
	var h, g atomic.Int32
	sw := NewInferenceSwitcher(NewHeartTextGenerator(daemon(t, false, &h, &g)), &fakeExternal{})
	got, err := sw.GenerateText(context.Background(), "hi")
	if err != nil || got != "external" {
		t.Fatalf("got %q %v", got, err)
	}
	if g.Load() != 0 {
		t.Fatal("an idle HEART must not be asked to generate")
	}
}

func TestHeartGeneratorDaemonDownIsNotReady(t *testing.T) {
	srv := httptest.NewServer(http.NotFoundHandler())
	url := srv.URL
	srv.Close() // nothing listens any more
	gen := NewHeartTextGenerator(models.NewClient(url, nil))
	if gen.IsReady() {
		t.Fatal("an unreachable daemon is not ready")
	}
	sw := NewInferenceSwitcher(gen, &fakeExternal{})
	if got, err := sw.GenerateText(context.Background(), "x"); err != nil || got != "external" {
		t.Fatalf("got %q %v", got, err)
	}
}

func TestInternalFailureFallsBackAndForceStillErrors(t *testing.T) {
	var h, g atomic.Int32
	// Health says ready, generate answers 503: HEART went away between the two.
	mux := http.NewServeMux()
	mux.HandleFunc(models.HeartPrefix+"/health", func(w http.ResponseWriter, _ *http.Request) {
		h.Add(1)
		_, _ = w.Write([]byte(`{"available":true,"status":"online","loaded":true,"ready":true,"progress":1}`))
	})
	mux.HandleFunc(models.HeartPrefix+"/generate", func(w http.ResponseWriter, _ *http.Request) {
		g.Add(1)
		w.WriteHeader(http.StatusServiceUnavailable)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()
	gen := NewHeartTextGenerator(models.NewClient(srv.URL, nil))
	ext := &fakeExternal{}
	sw := NewInferenceSwitcher(gen, ext)
	if got, err := sw.GenerateText(context.Background(), "x"); err != nil || got != "external" {
		t.Fatalf("auto mode must fall back: %q %v", got, err)
	}
	if sw.IsReady() {
		t.Fatal("a failed internal call must clear readiness")
	}
	sw.SetForce(1)
	if _, err := sw.GenerateText(context.Background(), "x"); err == nil || !strings.Contains(err.Error(), "heart") {
		t.Fatalf("forced internal must surface the error, got %v", err)
	}
	if !errors.Is(func() error { _, e := gen.GenerateText(context.Background(), "x"); return e }(), models.ErrNotReady) {
		t.Fatal("503 must map to ErrNotReady")
	}
}
