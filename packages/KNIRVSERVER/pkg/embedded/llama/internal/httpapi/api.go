// Package httpapi exposes a small, OpenAI-compatible facade over llama-server.
package httpapi

import (
	"github.com/guiperry/knirv/llama/internal/adapters"
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

// New creates a facade for llama-server. apiKey, when configured, is the
// wrapper-to-server credential; it deliberately replaces any client-supplied
// authorization header before proxying to the private upstream.
func New(upstream, model, apiKey string) (http.Handler, error) {
	return NewWithAdapters(upstream, model, apiKey, nil)
}

// NewWithAdapters is New plus the adapter surface. With a nil manager it is New.
//
// Selection is by name in the request body: {"adapter":"skill-x", ...}. Unknown
// names are a 404, never a silent fall back to the base model. When adapters are
// loaded, every request is forwarded with an explicit `lora` list (the chosen one
// at scale 1, all others 0), so no request can pick up an adapter by default.
// Clients may not send `lora` themselves; llama adapter ids are not a stable API.
func NewWithAdapters(upstream, model, apiKey string, mgr *adapters.Manager) (http.Handler, error) {
	target, err := url.Parse("http://" + upstream)
	if err != nil {
		return nil, err
	}
	proxy := httputil.NewSingleHostReverseProxy(target)
	director := proxy.Director
	proxy.Director = func(req *http.Request) {
		director(req)
		if apiKey != "" {
			req.Header.Set("Authorization", "Bearer "+apiKey)
		}
	}
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
		if apiKey != "" {
			req.Header.Set("Authorization", "Bearer "+apiKey)
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
	chat, completions := http.Handler(proxy), http.Handler(proxy)
	if mgr != nil {
		chat, completions = withAdapters(proxy, mgr), withAdapters(proxy, mgr)
		registerAdapterRoutes(mux, mgr, model)
	}
	mux.Handle("POST /v1/chat/completions", chat)
	mux.Handle("POST /v1/completions", completions)
	return mux, nil
}
