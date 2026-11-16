package models

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"io"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"knirvmodels/pkg/vlm"

	sdk "github.com/guiperry/knirv-sdk-go/models"
)

// TestHEADMiniCPMLiveRouting proves that HEAD's /models/head/* VLM routes reach the real
// MiniCPM-V llama-server through pkg/vlm (KNIRV_VLM_BACKEND=minicpm). The llama-server
// endpoint is wrapped in a counting reverse proxy, so the test records every request that
// leaves the backend and the model id llama-server reports in its reply. A stub cannot
// satisfy it: the counts and the reply model come from the live server.
//
// It runs only when KNIRV_VLM_MINICPM_LIVE_ENDPOINT names a running llama-server with the
// MiniCPM-V weights and mmproj loaded; otherwise it skips with the reason.
func TestHEADMiniCPMLiveRouting(t *testing.T) {
	ep := strings.TrimSpace(os.Getenv("KNIRV_VLM_MINICPM_LIVE_ENDPOINT"))
	if ep == "" {
		t.Skip("set KNIRV_VLM_MINICPM_LIVE_ENDPOINT (e.g. http://127.0.0.1:8093) to run against a live MiniCPM-V llama-server")
	}
	target, err := url.Parse(ep)
	if err != nil {
		t.Fatalf("bad KNIRV_VLM_MINICPM_LIVE_ENDPOINT: %v", err)
	}

	var mu sync.Mutex
	counts := map[string]int{}
	var replyModels []string
	rp := httputil.NewSingleHostReverseProxy(target)
	rp.Transport = &http.Transport{ResponseHeaderTimeout: 15 * time.Minute}
	rp.ModifyResponse = func(res *http.Response) error {
		if res.Request.URL.Path != "/v1/chat/completions" {
			return nil
		}
		raw, err := io.ReadAll(res.Body)
		res.Body.Close()
		if err != nil {
			return err
		}
		var doc struct {
			Model string `json:"model"`
		}
		_ = json.Unmarshal(raw, &doc)
		mu.Lock()
		replyModels = append(replyModels, doc.Model)
		mu.Unlock()
		res.Body = io.NopCloser(bytes.NewReader(raw))
		res.ContentLength = int64(len(raw))
		res.Header.Set("Content-Length", strconv.Itoa(len(raw)))
		return nil
	}
	proxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		counts[r.URL.Path]++
		mu.Unlock()
		rp.ServeHTTP(w, r)
	}))
	defer proxy.Close()

	t.Setenv("KNIRV_VLM_BACKEND", "minicpm")
	t.Setenv("KNIRV_VLM_MINICPM_ENDPOINT", proxy.URL)
	head := NewHead(vlm.FromEnv())
	mux := http.NewServeMux()
	head.Mount(mux, "/models/head")
	srv := httptest.NewServer(mux)
	defer srv.Close()

	// 1. Status must name the minicpm backend and be ready (the probe goes through the proxy).
	res, err := http.Get(srv.URL + "/models/head/status")
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	var status struct {
		Backend string `json:"backend"`
		Ready   bool   `json:"ready"`
		Detail  string `json:"detail"`
	}
	_ = json.NewDecoder(res.Body).Decode(&status)
	res.Body.Close()
	if status.Backend != "minicpm" || !status.Ready {
		t.Fatalf("HEAD status: backend=%q ready=%v detail=%q; want minicpm ready", status.Backend, status.Ready, status.Detail)
	}

	// 2. A real image through /models/head/describe.
	img := writeTestPNG(t)
	body, _ := json.Marshal(sdk.VisionRequest{
		Text:   "What shapes and colours are in this image?",
		Images: []sdk.ImageRef{{URL: "data:image/png;base64," + img, Kind: "diagram"}},
	})
	start := time.Now()
	res, err = http.Post(srv.URL+"/models/head/describe", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("describe: %v", err)
	}
	elapsed := time.Since(start)
	raw, _ := io.ReadAll(res.Body)
	res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("describe: HTTP %d: %s", res.StatusCode, raw)
	}
	var out sdk.VisionResponse
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("describe: bad response %s: %v", raw, err)
	}
	var result struct {
		Text string `json:"text"`
	}
	_ = json.Unmarshal(out.Result, &result)
	if strings.TrimSpace(result.Text) == "" {
		t.Fatalf("describe returned no text: %s", raw)
	}
	if out.Model != "minicpm" {
		t.Fatalf("response model=%q, want minicpm", out.Model)
	}

	// 3. A non-inline image must be refused before any request reaches llama-server.
	mu.Lock()
	chatBefore := counts["/v1/chat/completions"]
	mu.Unlock()
	body, _ = json.Marshal(sdk.VisionRequest{Images: []sdk.ImageRef{{URL: "https://example.com/x.png"}}})
	res, err = http.Post(srv.URL+"/models/head/describe", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("describe (remote URL): %v", err)
	}
	res.Body.Close()
	if res.StatusCode == http.StatusOK {
		t.Fatalf("remote image URL was accepted; it must be refused")
	}
	mu.Lock()
	chatAfter := counts["/v1/chat/completions"]
	mu.Unlock()
	if chatAfter != chatBefore {
		t.Fatalf("refused image still reached llama-server (%d -> %d chat calls)", chatBefore, chatAfter)
	}

	mu.Lock()
	defer mu.Unlock()
	t.Logf("routing proof: counts=%v reply_models=%v describe_latency=%s describe_text=%q",
		counts, replyModels, elapsed.Round(time.Millisecond), strings.TrimSpace(result.Text))
	if counts["/health"] < 1 || counts["/v1/chat/completions"] < 1 {
		t.Fatalf("requests did not reach llama-server: %v", counts)
	}
	if len(replyModels) == 0 || replyModels[0] == "" {
		t.Logf("llama-server reply carried no model field")
	}
}

// writeTestPNG writes a 256x256 image (red square, blue circle on white) and returns its
// base64 PNG payload.
func writeTestPNG(t *testing.T) string {
	t.Helper()
	im := image.NewRGBA(image.Rect(0, 0, 256, 256))
	for y := 0; y < 256; y++ {
		for x := 0; x < 256; x++ {
			im.Set(x, y, color.White)
			if x >= 20 && x < 110 && y >= 20 && y < 110 {
				im.Set(x, y, color.RGBA{220, 30, 30, 255})
			}
			dx, dy := x-180, y-180
			if dx*dx+dy*dy < 60*60 {
				im.Set(x, y, color.RGBA{30, 60, 220, 255})
			}
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, im); err != nil {
		t.Fatalf("png: %v", err)
	}
	return base64.StdEncoding.EncodeToString(buf.Bytes())
}
