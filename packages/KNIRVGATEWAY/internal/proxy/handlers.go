package proxy

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"

	"go.uber.org/zap"
)

// Handler represents a reverse proxy handler
type Handler struct {
	logger *zap.Logger
}

// NewHandler creates a new proxy handler
func NewHandler(logger *zap.Logger) *Handler {
	return &Handler{
		logger: logger,
	}
}

// ProxyTo creates a reverse proxy handler for a target URL
func (h *Handler) ProxyTo(target string, pathRewrite map[string]string) http.Handler {
	targetURL, err := url.Parse(target)
	if err != nil {
		h.logger.Error("Invalid proxy target", zap.String("target", target), zap.Error(err))
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "Invalid proxy configuration", http.StatusInternalServerError)
		})
	}

	proxy := httputil.NewSingleHostReverseProxy(targetURL)

	// Customize director to handle path rewrites
	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)

		// Apply path rewrites
		for old, new := range pathRewrite {
			if len(req.URL.Path) >= len(old) && req.URL.Path[:len(old)] == old {
				req.URL.Path = new + req.URL.Path[len(old):]
				break
			}
		}

		// Set forwarded headers
		req.Header.Set("X-Forwarded-Host", req.Host)
		if req.TLS != nil {
			req.Header.Set("X-Forwarded-Proto", "https")
		} else {
			req.Header.Set("X-Forwarded-Proto", "http")
		}
	}

	// Error handler
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		h.logger.Error("Proxy error",
			zap.String("target", target),
			zap.String("path", r.URL.Path),
			zap.Error(err),
		)
		http.Error(w, "Service unavailable", http.StatusBadGateway)
	}

	return proxy
}

// DynamicProxy creates a handler that proxies to a dynamically resolved target
func (h *Handler) DynamicProxy(getTarget func(*http.Request) (string, error)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		target, err := getTarget(r)
		if err != nil {
			h.logger.Warn("Failed to resolve proxy target",
				zap.String("path", r.URL.Path),
				zap.Error(err),
			)
			http.Error(w, err.Error(), http.StatusServiceUnavailable)
			return
		}

		h.ProxyTo(target, nil).ServeHTTP(w, r)
	}
}

// ServiceProxy creates a proxy that checks if service is available before proxying
func (h *Handler) ServiceProxy(serviceName string, getPort func(string) (int, bool), pathRewrite map[string]string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		port, running := getPort(serviceName)
		if !running {
			h.logger.Warn("Service not available",
				zap.String("service", serviceName),
				zap.String("path", r.URL.Path),
			)
			http.Error(w, fmt.Sprintf("%s service unavailable", serviceName), http.StatusServiceUnavailable)
			return
		}

		target := fmt.Sprintf("http://localhost:%d", port)
		h.ProxyTo(target, pathRewrite).ServeHTTP(w, r)
	}
}
