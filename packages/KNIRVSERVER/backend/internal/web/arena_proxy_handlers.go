// Copyright 2026 KNIRV-SERVER
// SPDX-License-Identifier: GPL-3.0-or-later

package web

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"go.uber.org/zap"
)

type ArenaProxyHandler struct {
	socketPath string
	logger     *zap.Logger
	client     *http.Client
}

func NewArenaProxyHandler(socketPath string, logger *zap.Logger) *ArenaProxyHandler {
	if logger == nil {
		logger = zap.NewNop()
	}

	// Create HTTP client that connects via Unix socket
	client := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
				return net.Dial("unix", socketPath)
			},
		},
	}

	return &ArenaProxyHandler{
		socketPath: socketPath,
		logger:     logger,
		client:     client,
	}
}

// RegisterRoutes registers the arena proxy routes
func (h *ArenaProxyHandler) RegisterRoutes(router interface {
	HandleFunc(string, func(http.ResponseWriter, *http.Request)) *mux.Route
	Handle(string, http.Handler) *mux.Route
}) {
	// Register all /arena/* paths to be proxied to the KNIRVARENA service
	// Use a route that catches all arena paths
	muxRouter, ok := router.(*mux.Router)
	if !ok {
		h.logger.Warn("Arena proxy handler: router is not a *mux.Router, falling back to HandleFunc")
		if handleFunc, ok := router.(interface {
			HandleFunc(string, func(http.ResponseWriter, *http.Request)) *mux.Route
		}); ok {
			handleFunc.HandleFunc("/arena", h.handleArenaRoot)
			handleFunc.HandleFunc("/arena/", h.handleArenaRoot)
		}
		return
	}

	// Mount all arena routes with a path prefix
	arenaRouter := muxRouter.PathPrefix("/arena").Subrouter()
	arenaRouter.HandleFunc("", h.handleArenaRoot).Methods("GET", "HEAD")
	arenaRouter.HandleFunc("/", h.handleArenaRoot).Methods("GET", "HEAD")
	arenaRouter.PathPrefix("/").HandlerFunc(h.handleArenaProxy)

	h.logger.Info("Arena proxy routes registered", zap.String("socket_path", h.socketPath))
}

// handleArenaRoot handles the root /arena path
func (h *ArenaProxyHandler) handleArenaRoot(w http.ResponseWriter, r *http.Request) {
	h.proxyRequest(w, r, "/")
}

// handleArenaProxy proxies all /arena/* requests to the KNIRVARENA service
func (h *ArenaProxyHandler) handleArenaProxy(w http.ResponseWriter, r *http.Request) {
	// Extract the path after /arena and proxy it
	path := r.URL.Path
	if strings.HasPrefix(path, "/arena") {
		path = strings.TrimPrefix(path, "/arena")
		if path == "" {
			path = "/"
		}
	}
	h.proxyRequest(w, r, path)
}

// proxyRequest proxies the HTTP request to the KNIRVARENA service via Unix socket
func (h *ArenaProxyHandler) proxyRequest(w http.ResponseWriter, r *http.Request, path string) {
	// Build the target URL (localhost is used because we connect via Unix socket)
	targetURL := fmt.Sprintf("http://localhost%s", path)
	if r.URL.RawQuery != "" {
		targetURL = fmt.Sprintf("%s?%s", targetURL, r.URL.RawQuery)
	}

	// Create a new request to the backend service
	proxyReq, err := http.NewRequestWithContext(r.Context(), r.Method, targetURL, r.Body)
	if err != nil {
		h.logger.Error("Failed to create proxy request", zap.Error(err), zap.String("path", path))
		http.Error(w, "Failed to proxy request", http.StatusInternalServerError)
		return
	}

	// Copy headers from the original request
	proxyReq.Header = make(http.Header)
	for key, values := range r.Header {
		// Skip certain headers that should not be proxied
		if shouldSkipHeader(key) {
			continue
		}
		for _, value := range values {
			proxyReq.Header.Add(key, value)
		}
	}

	// Set the Host header to the target service
	proxyReq.Host = "localhost"

	// Execute the proxy request
	resp, err := h.client.Do(proxyReq)
	if err != nil {
		h.logger.Error("Arena proxy request failed",
			zap.Error(err),
			zap.String("path", path),
			zap.String("method", r.Method),
		)
		http.Error(w, "Failed to reach arena service", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	// Copy response headers
	for key, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}

	// Write the status code
	w.WriteHeader(resp.StatusCode)

	// Copy the response body
	if _, err := io.Copy(w, resp.Body); err != nil {
		h.logger.Error("Failed to copy response body", zap.Error(err), zap.String("path", path))
		// Response header already written, so we can't send error
		return
	}

	h.logger.Debug("Arena proxy request completed",
		zap.String("path", path),
		zap.String("method", r.Method),
		zap.Int("status", resp.StatusCode),
	)
}

// shouldSkipHeader returns true if the header should not be proxied
func shouldSkipHeader(key string) bool {
	skipHeaders := []string{
		"Connection",
		"Keep-Alive",
		"Proxy-Authenticate",
		"Proxy-Connection",
		"TE",
		"Trailers",
		"Transfer-Encoding",
		"Upgrade",
	}
	for _, skip := range skipHeaders {
		if strings.EqualFold(key, skip) {
			return true
		}
	}
	return false
}
