// Package httpapi exposes a small, OpenAI-compatible facade over llama-server.
package httpapi

import (
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
)

func New(upstream, model string) (http.Handler, error) {
	target, err := url.Parse("http://" + upstream)
	if err != nil {
		return nil, err
	}
	proxy := httputil.NewSingleHostReverseProxy(target)
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
