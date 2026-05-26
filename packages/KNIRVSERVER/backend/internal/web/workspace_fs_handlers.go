package web

import (
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	dve "backend_server/internal/services/dve_workspace"

	"github.com/gorilla/mux"
)

// WorkspaceFSHandlers provides REST endpoints for browsing the DVE OverlayFS workspace.
type WorkspaceFSHandlers struct {
	dveService interface {
		GetWorkspace(dveID string) (*dve.OverlayWorkspace, error)
	}
	explorerCfg ExplorerConfig
}

// ExplorerConfig controls file explorer behavior.
type ExplorerConfig struct {
	MaxFileSizeBytes  int64    // cap for inline cat/preview
	AllowUpload       bool
	AllowDelete       bool
	HiddenPathPrefixes []string // paths to omit from listings (e.g. "work/")
	ShowLayerOrigin   bool     // annotate files with "upper" or "lower"
	HideSystemDirs    bool
}

// NewWorkspaceFSHandlers creates workspace FS handlers.
func NewWorkspaceFSHandlers(dveSvc interface {
	GetWorkspace(dveID string) (*dve.OverlayWorkspace, error)
}, cfg ExplorerConfig) *WorkspaceFSHandlers {
	return &WorkspaceFSHandlers{
		dveService:  dveSvc,
		explorerCfg: cfg,
	}
}

// workspaceRoot validates dveID, resolves ws.MergedDir, and checks path safety.
func (h *WorkspaceFSHandlers) workspaceRoot(dveID, reqPath string) (*dve.OverlayWorkspace, string, error) {
	ws, err := h.dveService.GetWorkspace(dveID)
	if err != nil {
		return nil, "", fmt.Errorf("workspace not found: %w", err)
	}
	clean := filepath.Join(ws.MergedDir, filepath.Clean("/"+reqPath))
	resolved, err := filepath.EvalSymlinks(clean)
	if err != nil {
		resolved = clean
	}
	rel, err := filepath.Rel(ws.MergedDir, resolved)
	if err != nil || strings.HasPrefix(rel, "..") {
		return nil, "", fmt.Errorf("path outside workspace")
	}
	for _, hidden := range h.explorerCfg.HiddenPathPrefixes {
		if strings.HasPrefix(rel, hidden) {
			return nil, "", fmt.Errorf("path not accessible")
		}
	}
	return ws, resolved, nil
}

// layerOf detects whether a resolved path belongs to the upper (writable) or lower (read-only) layer.
func (h *WorkspaceFSHandlers) layerOf(ws *dve.OverlayWorkspace, resolvedPath string) string {
	rel, _ := filepath.Rel(ws.MergedDir, resolvedPath)
	upperPath := filepath.Join(ws.UpperDir, rel)
	if _, err := os.Lstat(upperPath); err == nil {
		return "upper"
	}
	return "lower"
}

// DirEntry represents a single entry in a directory listing.
type DirEntry struct {
	Name    string `json:"name"`
	IsDir   bool   `json:"is_dir"`
	Size    int64  `json:"size,omitempty"`
	ModTime string `json:"mod_time,omitempty"`
	Layer   string `json:"layer,omitempty"`
	MIME    string `json:"mime,omitempty"`
}

// ListDir handles GET /api/dve/{id}/workspace/ls?path=...
func (h *WorkspaceFSHandlers) ListDir(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	dveID := vars["id"]
	reqPath := r.URL.Query().Get("path")
	if reqPath == "" {
		reqPath = "/"
	}

	ws, resolved, err := h.workspaceRoot(dveID, reqPath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	info, err := os.Stat(resolved)
	if err != nil {
		http.Error(w, "path not found", http.StatusNotFound)
		return
	}
	if !info.IsDir() {
		http.Error(w, "not a directory", http.StatusBadRequest)
		return
	}

	f, err := os.Open(resolved)
	if err != nil {
		http.Error(w, "cannot read directory", http.StatusInternalServerError)
		return
	}
	defer f.Close()

	names, err := f.Readdirnames(-1)
	if err != nil {
		http.Error(w, "cannot read directory", http.StatusInternalServerError)
		return
	}

	var entries []DirEntry
	for _, name := range names {
		fullPath := filepath.Join(resolved, name)
		fi, err := os.Stat(fullPath)
		if err != nil {
			continue
		}

		// Hide system directories if configured
		if h.explorerCfg.HideSystemDirs && fi.IsDir() {
			sysDirs := map[string]bool{"proc": true, "sys": true, "dev": true, "run": true, "mnt": true}
			if sysDirs[name] {
				continue
			}
		}

		entry := DirEntry{
			Name:    name,
			IsDir:   fi.IsDir(),
			Size:    fi.Size(),
			ModTime: fi.ModTime().UTC().Format("2006-01-02T15:04:05Z"),
		}
		if h.explorerCfg.ShowLayerOrigin {
			entry.Layer = h.layerOf(ws, fullPath)
		}
		if !fi.IsDir() {
			entry.MIME = mime.TypeByExtension(filepath.Ext(name))
		}
		entries = append(entries, entry)
	}

	writeFSJSON(w, http.StatusOK, map[string]interface{}{
		"path":     reqPath,
		"entries":  entries,
		"writable": h.explorerCfg.AllowUpload && h.layerOf(ws, resolved) == "upper",
	})
}

// ReadFile handles GET /api/dve/{id}/workspace/cat?path=...
func (h *WorkspaceFSHandlers) ReadFile(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	dveID := vars["id"]
	reqPath := r.URL.Query().Get("path")
	if reqPath == "" {
		http.Error(w, "path required", http.StatusBadRequest)
		return
	}

	_, resolved, err := h.workspaceRoot(dveID, reqPath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	info, err := os.Stat(resolved)
	if err != nil {
		http.Error(w, "file not found", http.StatusNotFound)
		return
	}
	if info.IsDir() {
		http.Error(w, "cannot read directory as file", http.StatusBadRequest)
		return
	}

	// Cap file size
	if info.Size() > h.explorerCfg.MaxFileSizeBytes && h.explorerCfg.MaxFileSizeBytes > 0 {
		http.Error(w, fmt.Sprintf("file too large (%d bytes, max %d bytes)", info.Size(), h.explorerCfg.MaxFileSizeBytes), http.StatusRequestEntityTooLarge)
		return
	}

	data, err := os.ReadFile(resolved)
	if err != nil {
		http.Error(w, "cannot read file", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

// DownloadFile handles GET /api/dve/{id}/workspace/download?path=...
func (h *WorkspaceFSHandlers) DownloadFile(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	dveID := vars["id"]
	reqPath := r.URL.Query().Get("path")
	if reqPath == "" {
		http.Error(w, "path required", http.StatusBadRequest)
		return
	}

	_, resolved, err := h.workspaceRoot(dveID, reqPath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	info, err := os.Stat(resolved)
	if err != nil {
		http.Error(w, "file not found", http.StatusNotFound)
		return
	}
	if info.IsDir() {
		http.Error(w, "cannot download directory", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filepath.Base(resolved)))
	w.Header().Set("Content-Type", "application/octet-stream")
	http.ServeFile(w, r, resolved)
}

// UploadFile handles POST /api/dve/{id}/workspace/upload (multipart, auth required)
func (h *WorkspaceFSHandlers) UploadFile(w http.ResponseWriter, r *http.Request) {
	if !h.explorerCfg.AllowUpload {
		http.Error(w, "uploads not allowed", http.StatusForbidden)
		return
	}

	vars := mux.Vars(r)
	dveID := vars["id"]
	destPath := r.URL.Query().Get("path")
	if destPath == "" {
		destPath = "/workspace"
	}

	ws, resolved, err := h.workspaceRoot(dveID, destPath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	// Only allow uploads to the upper layer
	if h.layerOf(ws, resolved) != "upper" {
		http.Error(w, "cannot write to read-only layer", http.StatusForbidden)
		return
	}

	r.ParseMultipartForm(32 << 20) // 32MB max
	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "missing file field", http.StatusBadRequest)
		return
	}
	defer file.Close()

	destFile := filepath.Join(resolved, header.Filename)
	out, err := os.Create(destFile)
	if err != nil {
		http.Error(w, "cannot create file", http.StatusInternalServerError)
		return
	}
	defer out.Close()

	written, err := io.Copy(out, file)
	if err != nil {
		http.Error(w, "write error", http.StatusInternalServerError)
		return
	}

	writeFSJSON(w, http.StatusCreated, map[string]interface{}{
		"path":   filepath.Join(destPath, header.Filename),
		"size":   written,
		"layer":  h.layerOf(ws, destFile),
	})
}

// DeleteFile handles DELETE /api/dve/{id}/workspace/rm?path=... (auth required)
func (h *WorkspaceFSHandlers) DeleteFile(w http.ResponseWriter, r *http.Request) {
	if !h.explorerCfg.AllowDelete {
		http.Error(w, "deletes not allowed", http.StatusForbidden)
		return
	}

	vars := mux.Vars(r)
	dveID := vars["id"]
	reqPath := r.URL.Query().Get("path")
	if reqPath == "" {
		http.Error(w, "path required", http.StatusBadRequest)
		return
	}

	ws, resolved, err := h.workspaceRoot(dveID, reqPath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	// Only allow deletes from the upper layer
	if h.layerOf(ws, resolved) != "upper" {
		http.Error(w, "cannot delete from read-only layer", http.StatusForbidden)
		return
	}

	info, err := os.Stat(resolved)
	if err != nil {
		http.Error(w, "file not found", http.StatusNotFound)
		return
	}

	if info.IsDir() {
		if err := os.RemoveAll(resolved); err != nil {
			http.Error(w, "cannot remove directory", http.StatusInternalServerError)
			return
		}
	} else {
		if err := os.Remove(resolved); err != nil {
			http.Error(w, "cannot remove file", http.StatusInternalServerError)
			return
		}
	}

	writeFSJSON(w, http.StatusOK, map[string]string{"message": "deleted"})
}

// SetWorkspaceFSHandlers wires workspace FS handlers into DVEHandlers.
// Called after NewDVEHandlers to inject the workspace explorer API.
func (h *DVEHandlers) SetWorkspaceFSHandlers(fs *WorkspaceFSHandlers) {
	h.workspaceFSHandlers = fs
}

// writeFSJSON writes a JSON response.
func writeFSJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(data)
}
