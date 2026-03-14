// Copyright 2026 KNIRV-SERVER
// SPDX-License-Identifier: GPL-3.0-or-later

package runtime

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"
	"time"
)

// ViewportProxyImpl implements the ViewportProxy interface with real HTTP/WebSocket proxying
type ViewportProxyImpl struct {
	container *UnifiedContainer
	renderers []string
	running   bool
	targetURL string
	proxy     *httputil.ReverseProxy
	mu        sync.RWMutex
	startedAt time.Time
}

// NewViewportProxyImpl creates a new viewport proxy implementation
func NewViewportProxyImpl(container *UnifiedContainer, renderers []string) ViewportProxy {
	return &ViewportProxyImpl{
		container: container,
		renderers: renderers,
		running:   false,
	}
}

// Start starts the viewport proxy
func (vp *ViewportProxyImpl) Start() error {
	vp.mu.Lock()
	defer vp.mu.Unlock()

	if vp.running {
		return fmt.Errorf("viewport proxy already running")
	}

	log.Printf("Starting viewport proxy for container %s with renderers: %v",
		vp.container.ID, vp.renderers)

	// Determine target port from container spec
	targetPort := 8080
	if vp.container.Spec != nil {
		for _, pm := range vp.container.Spec.Ports {
			if pm.HostPort > 0 {
				targetPort = pm.HostPort
				break
			}
		}
	}

	vp.targetURL = fmt.Sprintf("http://localhost:%d", targetPort)

	// Build reverse proxy to agent container
	target, err := url.Parse(vp.targetURL)
	if err != nil {
		return fmt.Errorf("invalid target URL %s: %w", vp.targetURL, err)
	}

	vp.proxy = httputil.NewSingleHostReverseProxy(target)
	// Upgrade transport for WebSocket support
	vp.proxy.ModifyResponse = func(r *http.Response) error {
		r.Header.Set("X-KNIRV-Container-ID", vp.container.ID)
		r.Header.Set("X-KNIRV-Agent-Viewport", "true")
		return nil
	}
	vp.proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		log.Printf("ViewportProxy error for container %s: %v", vp.container.ID, err)
		http.Error(w, fmt.Sprintf("viewport proxy error: %v", err), http.StatusBadGateway)
	}

	vp.running = true
	vp.startedAt = time.Now()

	return nil
}

// Stop stops the viewport proxy
func (vp *ViewportProxyImpl) Stop() error {
	vp.mu.Lock()
	defer vp.mu.Unlock()

	if !vp.running {
		return nil
	}

	log.Printf("Stopping viewport proxy for container %s (uptime: %s)",
		vp.container.ID, time.Since(vp.startedAt).Round(time.Second))

	vp.running = false
	vp.proxy = nil

	return nil
}

// HandleHTTP proxies HTTP/WebSocket requests to the container viewport
func (vp *ViewportProxyImpl) HandleHTTP(path string) ([]byte, error) {
	vp.mu.RLock()
	defer vp.mu.RUnlock()

	if !vp.running {
		return nil, fmt.Errorf("viewport proxy not running")
	}

	// Return proxy target info - actual proxying happens via ServeHTTP
	return []byte(fmt.Sprintf(`{"container_id":"%s","target":"%s","path":"%s","renderers":%q}`,
		vp.container.ID, vp.targetURL, path, vp.renderers)), nil
}

// ServeHTTP implements http.Handler so the proxy can be used directly in HTTP handlers
func (vp *ViewportProxyImpl) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	vp.mu.RLock()
	proxy := vp.proxy
	running := vp.running
	vp.mu.RUnlock()

	if !running || proxy == nil {
		http.Error(w, "viewport proxy not running", http.StatusServiceUnavailable)
		return
	}

	proxy.ServeHTTP(w, r)
}

// GetStatus returns the viewport proxy status
func (vp *ViewportProxyImpl) GetStatus() string {
	vp.mu.RLock()
	defer vp.mu.RUnlock()

	if vp.running {
		return fmt.Sprintf("running (target: %s, uptime: %s)",
			vp.targetURL, time.Since(vp.startedAt).Round(time.Second))
	}
	return "stopped"
}
