package gui

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/knirv/nexus-backend/internal/config"
)

// Server represents the GUI server for serving the Next.js frontend
type Server struct {
	config     *config.Config
	httpServer *http.Server
}

// NewServer creates a new GUI server instance
func NewServer(cfg *config.Config) *Server {
	return &Server{
		config: cfg,
	}
}

// Start starts the GUI server
func (s *Server) Start(ctx context.Context) error {
	if !s.config.GUI.Enabled {
		return fmt.Errorf("GUI mode is not enabled")
	}

	// Create HTTP server
	mux := http.NewServeMux()

	// Serve Next.js static files
	staticPath := "./.next/static/"
	staticHandler := http.StripPrefix("/_next/static/", http.FileServer(http.Dir(staticPath)))
	mux.Handle("/_next/static/", staticHandler)

	// Serve public files
	publicHandler := http.FileServer(http.Dir("./public/"))
	mux.Handle("/favicon.ico", publicHandler)
	mux.Handle("/robots.txt", publicHandler)
	mux.Handle("/logo.svg", publicHandler)

	// API proxy endpoints for local development
	mux.HandleFunc("/api/", s.handleAPIProxy)

	// Health check endpoint
	mux.HandleFunc("/health", s.handleHealth)

	// Serve the main application (SPA fallback)
	mux.HandleFunc("/", s.handleSPA)

	// Create HTTP server
	addr := fmt.Sprintf("%s:%d", s.config.GUI.BindAddress, s.config.GUI.Port)
	s.httpServer = &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in a goroutine
	go func() {
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Printf("GUI server error: %v\n", err)
		}
	}()

	fmt.Printf("GUI server started on http://%s\n", addr)
	return nil
}

// Stop stops the GUI server
func (s *Server) Stop(ctx context.Context) error {
	if s.httpServer == nil {
		return nil
	}

	return s.httpServer.Shutdown(ctx)
}

// handleAPIProxy handles API proxy requests to the backend services
func (s *Server) handleAPIProxy(w http.ResponseWriter, r *http.Request) {
	// Enable CORS for local development
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	// For now, return a simple response
	// TODO: Implement proper API proxy to backend services
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status": "GUI mode - API proxy not yet implemented"}`))
}

// handleSPA handles SPA routing by serving the index.html for all non-API routes
func (s *Server) handleSPA(w http.ResponseWriter, r *http.Request) {
	// For API routes, let them be handled by the API proxy
	if r.URL.Path == "/health" {
		s.handleHealth(w, r)
		return
	}

	// For all other routes, serve the Next.js index page
	indexPath := "./.next/server/app/page.html"

	// Check if the file exists, if not serve a simple HTML page
	if _, err := os.Stat(indexPath); os.IsNotExist(err) {
		// Serve a simple HTML page for GUI mode
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		html := `<!DOCTYPE html>
<html>
<head>
    <title>KNIRV NEXUS - GUI Mode</title>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <style>
        body { font-family: Arial, sans-serif; margin: 40px; background: #f5f5f5; }
        .container { max-width: 800px; margin: 0 auto; background: white; padding: 40px; border-radius: 8px; box-shadow: 0 2px 10px rgba(0,0,0,0.1); }
        h1 { color: #333; }
        .status { background: #e8f5e8; padding: 15px; border-radius: 4px; margin: 20px 0; }
        .api-links { margin: 20px 0; }
        .api-links a { display: block; margin: 5px 0; color: #0066cc; }
    </style>
</head>
<body>
    <div class="container">
        <h1>KNIRV NEXUS - GUI Mode</h1>
        <div class="status">
            <strong>Status:</strong> GUI Mode Active - No authentication required
        </div>
        <p>The KNIRV NEXUS system is running in GUI mode for local administration.</p>
        <div class="api-links">
            <h3>Available API Endpoints:</h3>
            <a href="/api/health">Health Check</a>
            <a href="/api/dve-nodes">DVE Nodes</a>
            <a href="/api/validation-tasks">Validation Tasks</a>
            <a href="/api/system-health">System Health</a>
        </div>
    </div>
</body>
</html>`
		w.Write([]byte(html))
		return
	}

	// Serve the actual Next.js page
	file, err := os.Open(indexPath)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	defer file.Close()

	w.Header().Set("Content-Type", "text/html")
	io.Copy(w, file)
}

// handleHealth handles health check requests
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status": "healthy", "mode": "gui"}`))
}
