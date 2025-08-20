package main

import (
	"embed"
	"io"
	"io/fs"
	"net/http"
	"path"
	"strings"
	"time"
)

// Embed the Next.js build output
//go:embed all:out
var embeddedFiles embed.FS

// EmbeddedFS provides access to the embedded frontend files
type EmbeddedFS struct {
	files fs.FS
}

// NewEmbeddedFS creates a new embedded filesystem
func NewEmbeddedFS() (*EmbeddedFS, error) {
	// Get the subdirectory containing the actual files
	files, err := fs.Sub(embeddedFiles, "out")
	if err != nil {
		return nil, err
	}
	
	return &EmbeddedFS{
		files: files,
	}, nil
}

// ServeHTTP serves the embedded frontend files
func (efs *EmbeddedFS) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Clean the path
	cleanPath := path.Clean(r.URL.Path)
	
	// Remove leading slash
	if strings.HasPrefix(cleanPath, "/") {
		cleanPath = cleanPath[1:]
	}
	
	// If path is empty, serve index.html
	if cleanPath == "" {
		cleanPath = "index.html"
	}
	
	// Try to open the file
	file, err := efs.files.Open(cleanPath)
	if err != nil {
		// If file not found, try with .html extension
		if !strings.HasSuffix(cleanPath, ".html") {
			htmlPath := cleanPath + ".html"
			if file, err = efs.files.Open(htmlPath); err == nil {
				cleanPath = htmlPath
			}
		}
		
		// If still not found, try index.html in the directory
		if err != nil {
			indexPath := path.Join(cleanPath, "index.html")
			if file, err = efs.files.Open(indexPath); err == nil {
				cleanPath = indexPath
			}
		}
		
		// If still not found, serve 404
		if err != nil {
			efs.serve404(w, r)
			return
		}
	}
	defer file.Close()
	
	// Set content type based on file extension
	efs.setContentType(w, cleanPath)
	
	// Read the file content and serve it
	content, err := io.ReadAll(file)
	if err != nil {
		http.Error(w, "Error reading file", http.StatusInternalServerError)
		return
	}
	
	// Set cache headers for static assets
	if isStaticAsset(cleanPath) {
		w.Header().Set("Cache-Control", "public, max-age=31536000") // 1 year
	} else {
		w.Header().Set("Cache-Control", "no-cache")
	}
	
	// Serve the content
	http.ServeContent(w, r, cleanPath, getModTime(), strings.NewReader(string(content)))
}

// serve404 serves the 404 page
func (efs *EmbeddedFS) serve404(w http.ResponseWriter, r *http.Request) {
	// Try to serve the custom 404 page
	file, err := efs.files.Open("404.html")
	if err != nil {
		// Fallback to basic 404
		http.Error(w, "404 - Page Not Found", http.StatusNotFound)
		return
	}
	defer file.Close()
	
	content, err := io.ReadAll(file)
	if err != nil {
		http.Error(w, "404 - Page Not Found", http.StatusNotFound)
		return
	}
	
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusNotFound)
	w.Write(content)
}

// setContentType sets the appropriate content type based on file extension
func (efs *EmbeddedFS) setContentType(w http.ResponseWriter, filename string) {
	ext := path.Ext(filename)
	
	switch ext {
	case ".html":
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
	case ".css":
		w.Header().Set("Content-Type", "text/css; charset=utf-8")
	case ".js":
		w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
	case ".json":
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
	case ".png":
		w.Header().Set("Content-Type", "image/png")
	case ".jpg", ".jpeg":
		w.Header().Set("Content-Type", "image/jpeg")
	case ".gif":
		w.Header().Set("Content-Type", "image/gif")
	case ".svg":
		w.Header().Set("Content-Type", "image/svg+xml")
	case ".ico":
		w.Header().Set("Content-Type", "image/x-icon")
	case ".woff":
		w.Header().Set("Content-Type", "font/woff")
	case ".woff2":
		w.Header().Set("Content-Type", "font/woff2")
	case ".ttf":
		w.Header().Set("Content-Type", "font/ttf")
	case ".eot":
		w.Header().Set("Content-Type", "application/vnd.ms-fontobject")
	case ".txt":
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	case ".xml":
		w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	default:
		// Let Go determine the content type
		w.Header().Set("Content-Type", "application/octet-stream")
	}
}

// isStaticAsset checks if the file is a static asset that can be cached
func isStaticAsset(filename string) bool {
	ext := path.Ext(filename)
	staticExts := []string{".css", ".js", ".png", ".jpg", ".jpeg", ".gif", ".svg", ".ico", ".woff", ".woff2", ".ttf", ".eot"}
	
	for _, staticExt := range staticExts {
		if ext == staticExt {
			return true
		}
	}
	
	// Also check for Next.js static assets
	if strings.Contains(filename, "/_next/static/") {
		return true
	}
	
	return false
}

// getModTime returns a static modification time for embedded files
func getModTime() time.Time {
	// Use a static time for embedded files to enable caching
	return time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
}

// GetFileSystem returns the embedded file system
func (efs *EmbeddedFS) GetFileSystem() fs.FS {
	return efs.files
}

// FileExists checks if a file exists in the embedded filesystem
func (efs *EmbeddedFS) FileExists(path string) bool {
	_, err := efs.files.Open(path)
	return err == nil
}

// ReadFile reads a file from the embedded filesystem
func (efs *EmbeddedFS) ReadFile(path string) ([]byte, error) {
	return fs.ReadFile(efs.files, path)
}

// ListFiles lists all files in a directory
func (efs *EmbeddedFS) ListFiles(dir string) ([]string, error) {
	entries, err := fs.ReadDir(efs.files, dir)
	if err != nil {
		return nil, err
	}
	
	var files []string
	for _, entry := range entries {
		files = append(files, entry.Name())
	}
	
	return files, nil
}
