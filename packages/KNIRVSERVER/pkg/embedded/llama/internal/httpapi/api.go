// Package httpapi exposes a small, OpenAI-compatible facade over llama-server.
package httpapi

import (
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
)

const maxLoggedCompletionBytes = 1 << 20

// completionLogBody mirrors a proxied response while preserving the exact
// stream sent to the caller. Logging at EOF/Close captures responses from all
// clients, not only the backend's non-streaming provider path.
type completionLogBody struct {
	io.ReadCloser
	buf    strings.Builder
	logged bool
}

func (b *completionLogBody) Read(p []byte) (int, error) {
	n, err := b.ReadCloser.Read(p)
	if n > 0 && b.buf.Len() < maxLoggedCompletionBytes {
		remaining := maxLoggedCompletionBytes - b.buf.Len()
		captureN := n
		if captureN > remaining {
			captureN = remaining
		}
		_, _ = b.buf.Write(p[:captureN])
	}
	if err == io.EOF {
		b.logCompletion()
	}
	return n, err
}

func (b *completionLogBody) Close() error {
	b.logCompletion()
	return b.ReadCloser.Close()
}

func (b *completionLogBody) logCompletion() {
	if b.logged {
		return
	}
	b.logged = true
	if completion := strings.TrimSpace(b.buf.String()); completion != "" {
		log.Printf("[KNIRVLLAMA] completion response: %s", completion)
	}
}

func New(upstream, model string) (http.Handler, error) {
	target, err := url.Parse("http://" + upstream)
	if err != nil {
		return nil, err
	}
	proxy := httputil.NewSingleHostReverseProxy(target)
	proxy.ModifyResponse = func(resp *http.Response) error {
		if resp.Request != nil && (resp.Request.URL.Path == "/v1/chat/completions" || resp.Request.URL.Path == "/v1/completions") {
			resp.Body = &completionLogBody{ReadCloser: resp.Body}
		}
		return nil
	}
	proxy.ErrorHandler = func(w http.ResponseWriter, _ *http.Request, _ error) {
		http.Error(w, "llama-server is unavailable", http.StatusBadGateway)
	}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, target.String()+"/health", nil)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			http.Error(w, "llama-server is unavailable", http.StatusBadGateway)
			return
		}
		defer resp.Body.Close()
		w.WriteHeader(resp.StatusCode)
		_, _ = io.Copy(w, resp.Body)
	})
	mux.HandleFunc("GET /v1/models", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"object":"list","data":[{"id":"` + model + `","object":"model"}]}`))
	})
	mux.Handle("POST /v1/chat/completions", proxy)
	mux.Handle("POST /v1/completions", proxy)
	return mux, nil
}
