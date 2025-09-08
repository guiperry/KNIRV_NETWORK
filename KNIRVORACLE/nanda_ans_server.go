package main

import (
	"embed"
	"io/fs"
	"log"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/gorilla/mux"
)

//go:embed nanda_ans/out
var nandaANSFiles embed.FS

// setupNANDAANSHandler sets up the HTTP handler for serving the embedded NANDA-ANS static files
func setupNANDAANSHandler(router *mux.Router) {
	// Create a sub-filesystem for the nanda_ans/out directory
	nandaFS, err := fs.Sub(nandaANSFiles, "nanda_ans/out")
	if err != nil {
		log.Printf("Error setting up NANDA-ANS filesystem: %v", err)
		return
	}

	// Create a custom handler that serves the files with proper routing for SPA
	nandaHandler := &NANDAANSHandler{
		fileSystem: http.FS(nandaFS),
	}

	// Register the handler for the /nanda-ans/ path using Gorilla Mux
	router.PathPrefix("/nanda-ans/").Handler(http.StripPrefix("/nanda-ans/", nandaHandler))

	log.Printf("NANDA-ANS service available at http://localhost:<port>/nanda-ans/")
}

// NANDAANSHandler implements http.Handler for serving NANDA-ANS static files
type NANDAANSHandler struct {
	fileSystem http.FileSystem
}

// ServeHTTP implements http.Handler for serving embedded NANDA-ANS files
func (h *NANDAANSHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Clean the path
	path := strings.TrimPrefix(r.URL.Path, "/")
	if path == "" {
		path = "index.html"
	}

	// Try to open the file
	file, err := h.fileSystem.Open(path)
	if err != nil {
		// If file not found, try with .html extension
		// Check if path has no file extension (not just no dots)
		if filepath.Ext(path) == "" {
			htmlPath := path + ".html"
			if file, err = h.fileSystem.Open(htmlPath); err != nil {
				// If still not found, serve index.html for SPA routing
				if file, err = h.fileSystem.Open("index.html"); err != nil {
					http.NotFound(w, r)
					return
				}
				path = "index.html"
			} else {
				path = htmlPath
			}
		} else {
			// If still not found, serve index.html for SPA routing
			if file, err = h.fileSystem.Open("index.html"); err != nil {
				http.NotFound(w, r)
				return
			}
			path = "index.html"
		}
	}
	defer file.Close()

	// Get file info for content type and caching
	stat, err := file.Stat()
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Set appropriate content type based on file extension
	contentType := getContentType(path)
	w.Header().Set("Content-Type", contentType)

	// Set caching headers for static assets
	if strings.HasPrefix(path, "_next/static/") {
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	} else if strings.HasSuffix(path, ".html") {
		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	} else {
		w.Header().Set("Cache-Control", "public, max-age=3600")
	}

	// Serve the file
	http.ServeContent(w, r, stat.Name(), stat.ModTime(), file)
}

// getContentType returns the appropriate content type for a file based on its extension
func getContentType(path string) string {
	if strings.HasSuffix(path, ".html") {
		return "text/html; charset=utf-8"
	}
	if strings.HasSuffix(path, ".css") {
		return "text/css"
	}
	if strings.HasSuffix(path, ".js") {
		return "application/javascript"
	}
	if strings.HasSuffix(path, ".json") {
		return "application/json"
	}
	if strings.HasSuffix(path, ".png") {
		return "image/png"
	}
	if strings.HasSuffix(path, ".jpg") || strings.HasSuffix(path, ".jpeg") {
		return "image/jpeg"
	}
	if strings.HasSuffix(path, ".gif") {
		return "image/gif"
	}
	if strings.HasSuffix(path, ".svg") {
		return "image/svg+xml"
	}
	if strings.HasSuffix(path, ".ico") {
		return "image/x-icon"
	}
	if strings.HasSuffix(path, ".woff") {
		return "font/woff"
	}
	if strings.HasSuffix(path, ".woff2") {
		return "font/woff2"
	}
	if strings.HasSuffix(path, ".ttf") {
		return "font/ttf"
	}
	if strings.HasSuffix(path, ".txt") {
		return "text/plain"
	}
	return "application/octet-stream"
}
