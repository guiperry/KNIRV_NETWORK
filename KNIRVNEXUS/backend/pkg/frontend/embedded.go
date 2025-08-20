package frontend

import (
	"embed"
	"io/fs"
	"net/http"
	"path"
	"strings"
	"time"
)

// Embed the Next.js build output
// Note: This will be set up properly when the package is moved to the correct location
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

	// Serve the file
	http.ServeContent(w, r, cleanPath, getModTime(), file)
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

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusNotFound)
	http.ServeContent(w, r, "404.html", getModTime(), file)
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
