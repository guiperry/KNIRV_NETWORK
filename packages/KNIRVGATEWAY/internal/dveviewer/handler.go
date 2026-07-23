package dveviewer

import (
	"embed"
	"io/fs"
	"net/http"
	"path"
)

//go:embed static/* assets/viewer-app.js assets/verifier.js assets/verifier_bg.wasm
var content embed.FS

type Handler struct {
	assets http.Handler
}

func New() *Handler {
	assets, err := fs.Sub(content, "assets")
	if err != nil {
		panic(err)
	}
	return &Handler{assets: http.FileServer(http.FS(assets))}
}

func (h *Handler) Asset(w http.ResponseWriter, r *http.Request) {
	name := path.Base(r.URL.Path)
	if name != "viewer-app.js" && name != "verifier.js" && name != "verifier_bg.wasm" {
		http.NotFound(w, r)
		return
	}
	r.URL.Path = "/" + name
	w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	h.assets.ServeHTTP(w, r)
}

func (h *Handler) Page(w http.ResponseWriter, r *http.Request) {
	page, err := content.ReadFile("static/viewer.html")
	if err != nil {
		http.Error(w, "DVE verifier unavailable", http.StatusServiceUnavailable)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Security-Policy", "default-src 'none'; script-src 'self' 'wasm-unsafe-eval'; style-src 'self' 'unsafe-inline'; connect-src 'self'; img-src 'self' data:; base-uri 'none'; frame-ancestors 'self'")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	_, _ = w.Write(page)
}
