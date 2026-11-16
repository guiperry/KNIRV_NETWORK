package vlm

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	sdk "github.com/guiperry/knirv-sdk-go/models"
)

// fakeLlamaServer stands in for llama-server: /health and an OpenAI chat endpoint that
// records the last request and answers with reply.
func fakeLlamaServer(t *testing.T, reply string, last *map[string]any) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/health":
			w.Write([]byte(`{"status":"ok"}`))
		case "/v1/chat/completions":
			raw, _ := io.ReadAll(r.Body)
			if last != nil {
				_ = json.Unmarshal(raw, last)
			}
			out, _ := json.Marshal(map[string]any{
				"choices": []map[string]any{{"message": map[string]string{"content": reply}}},
			})
			w.Write(out)
		default:
			http.NotFound(w, r)
		}
	}))
}

const tinyPNG = "data:image/png;base64,iVBORw0KGgo="

func TestMiniCPMNotConfiguredIsInactive(t *testing.T) {
	b := NewMiniCPMBackend("")
	if ok, detail := b.Ready(context.Background()); ok || detail == "" {
		t.Fatalf("empty endpoint must be not-ready with a reason, got ok=%v detail=%q", ok, detail)
	}
	if (b.Capabilities() != sdk.HeadCapabilities{}) {
		t.Fatalf("inactive backend must report no capability")
	}
	if _, err := b.Generate(context.Background(), sdk.HeadGenerateRequest{Prompt: "x"}); !errors.Is(err, ErrNoModel) {
		t.Fatalf("Generate on inactive backend: want ErrNoModel, got %v", err)
	}
}

func TestMiniCPMReadyFollowsHealth(t *testing.T) {
	srv := fakeLlamaServer(t, "", nil)
	defer srv.Close()
	if ok, detail := NewMiniCPMBackend(srv.URL).Ready(context.Background()); !ok {
		t.Fatalf("healthy server must be ready, detail=%q", detail)
	}
	down := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer down.Close()
	if ok, _ := NewMiniCPMBackend(down.URL).Ready(context.Background()); ok {
		t.Fatalf("503 from /health must be not-ready")
	}
}

func TestMiniCPMGenerateSendsInlineImagesAndReturnsText(t *testing.T) {
	var last map[string]any
	srv := fakeLlamaServer(t, "  a red square  ", &last)
	defer srv.Close()
	b := NewMiniCPMBackend(srv.URL)
	resp, err := b.Generate(context.Background(), sdk.HeadGenerateRequest{
		Prompt: "what is shown?",
		Images: []sdk.ImageRef{{URL: tinyPNG, Kind: "photo"}},
	})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if resp.Text != "a red square" || resp.Model != backendMiniCPM || resp.Version == "" {
		t.Fatalf("unexpected response: %+v", resp)
	}
	if last["model"] != minicpmDefaultModel {
		t.Fatalf("model not sent: %v", last["model"])
	}
	raw, _ := json.Marshal(last)
	if !strings.Contains(string(raw), `"type":"image_url"`) || !strings.Contains(string(raw), tinyPNG) {
		t.Fatalf("image not forwarded as image_url data URI: %s", raw)
	}
	if !strings.Contains(string(raw), `"temperature":0.7`) {
		t.Fatalf("model-card sampling defaults missing: %s", raw)
	}
}

func TestMiniCPMRejectsNonInlineImages(t *testing.T) {
	srv := fakeLlamaServer(t, "unused", nil)
	defer srv.Close()
	b := NewMiniCPMBackend(srv.URL)
	for _, u := range []string{
		"https://example.com/cat.png",
		"file:///etc/passwd",
		"/var/lib/knirvserver/secret.png",
		"data:text/plain;base64,aGk=",
	} {
		_, err := b.Generate(context.Background(), sdk.HeadGenerateRequest{
			Prompt: "x", Images: []sdk.ImageRef{{URL: u}},
		})
		if !errors.Is(err, ErrImageRejected) {
			t.Fatalf("image %q: want ErrImageRejected, got %v", u, err)
		}
	}
}

func TestMiniCPMVisionOperations(t *testing.T) {
	srv := fakeLlamaServer(t, "two circles", nil)
	defer srv.Close()
	b := NewMiniCPMBackend(srv.URL)

	resp, err := b.Vision(context.Background(), sdk.VisionRequest{
		Operation: "describe", Images: []sdk.ImageRef{{URL: tinyPNG}},
	})
	if err != nil {
		t.Fatalf("describe: %v", err)
	}
	if resp.Operation != "describe" || !strings.Contains(string(resp.Result), "two circles") {
		t.Fatalf("unexpected describe result: %+v", resp)
	}

	if _, err := b.Vision(context.Background(), sdk.VisionRequest{Operation: "embed"}); !errors.Is(err, ErrOperationUnsupported) {
		t.Fatalf("embed must be unsupported, got %v", err)
	}
	if _, err := b.Vision(context.Background(), sdk.VisionRequest{Operation: "ground", Images: []sdk.ImageRef{{URL: "https://x/y.png"}}}); !errors.Is(err, ErrImageRejected) {
		t.Fatalf("ground with URL image: want ErrImageRejected, got %v", err)
	}
}

func TestMiniCPMFactorySelectsBackend(t *testing.T) {
	srv := fakeLlamaServer(t, "", nil)
	defer srv.Close()
	t.Setenv("KNIRV_VLM_MINICPM_ENDPOINT", srv.URL)
	b := New("minicpm")
	if b.Name() != backendMiniCPM {
		t.Fatalf("factory returned %q", b.Name())
	}
	if ok, _ := b.Ready(context.Background()); !ok {
		t.Fatalf("factory backend should be ready against the fake server")
	}
}

// TestMiniCPMLiveVision runs only when KNIRV_VLM_MINICPM_LIVE_ENDPOINT names a running
// llama-server with MiniCPM-V and its mmproj loaded. It sends a real image through the
// backend and checks that the reply is non-empty text.
func TestMiniCPMLiveVision(t *testing.T) {
	ep := strings.TrimSpace(os.Getenv("KNIRV_VLM_MINICPM_LIVE_ENDPOINT"))
	if ep == "" {
		t.Skip("set KNIRV_VLM_MINICPM_LIVE_ENDPOINT to run against a live llama-server")
	}
	img, err := os.ReadFile(os.Getenv("KNIRV_VLM_MINICPM_LIVE_IMAGE"))
	if err != nil || len(img) == 0 {
		t.Skip("set KNIRV_VLM_MINICPM_LIVE_IMAGE to a PNG path")
	}
	b := NewMiniCPMBackend(ep)
	b.client.Timeout = 15 * time.Minute
	b.MaxTokens = 48
	if ok, detail := b.Ready(context.Background()); !ok {
		t.Fatalf("live endpoint not ready: %s", detail)
	}
	data := "data:image/png;base64," + base64.StdEncoding.EncodeToString(img)
	resp, err := b.Vision(context.Background(), sdk.VisionRequest{
		Operation: "describe",
		Images:    []sdk.ImageRef{{URL: data, Kind: "photo"}},
	})
	if err != nil {
		t.Fatalf("live describe: %v", err)
	}
	var out struct {
		Text string `json:"text"`
	}
	if err := json.Unmarshal(resp.Result, &out); err != nil || strings.TrimSpace(out.Text) == "" {
		t.Fatalf("live describe returned no text: %s", resp.Result)
	}
	t.Logf("live describe: %s", out.Text)
}
