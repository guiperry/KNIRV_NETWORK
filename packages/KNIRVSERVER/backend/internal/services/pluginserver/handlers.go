package pluginserver

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// HandleServerInfo handles the /info endpoint
func (as *PluginServer) HandleServerInfo(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(as.serverInfo)
}

// HandleListPlugins handles the /list endpoint
func (as *PluginServer) HandleListPlugins(w http.ResponseWriter, r *http.Request) {
	// Get all files in the plugin directory
	files, err := os.ReadDir(as.pluginDir)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error reading plugin directory: %v", err), http.StatusInternalServerError)
		return
	}

	// Build the response
	var pluginUnits []PluginInfo
	for _, file := range files {
		if !file.IsDir() && (strings.HasSuffix(file.Name(), ".so") || strings.HasSuffix(file.Name(), ".wasm")) {
			info, err := file.Info()
			if err != nil {
				log.Printf("Error getting file info for %s: %v", file.Name(), err)
				continue
			}

			pluginUnits = append(pluginUnits, PluginInfo{
				Name:         file.Name(),
				Size:         info.Size(),
				LastModified: info.ModTime(),
			})
		}
	}

	// Return the list of plugin units
	response := ListResponse{
		Plugins: pluginUnits,
		Count:   len(pluginUnits),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// HandleDownloadPlugin handles the /objects/{name} endpoint
func (as *PluginServer) HandleDownloadPlugin(w http.ResponseWriter, r *http.Request) {
	// Extract the plugin name from the URL
	pluginName := strings.TrimPrefix(r.URL.Path, "/objects/")
	if pluginName == "" {
		http.Error(w, "Plugin name is required", http.StatusBadRequest)
		return
	}

	// Sanitize the plugin name to prevent directory traversal
	pluginName = filepath.Base(pluginName)

	// Construct the full path to the plugin file
	pluginPath := filepath.Join(as.pluginDir, pluginName)

	// Check if the file exists
	fileInfo, err := os.Stat(pluginPath)
	if os.IsNotExist(err) {
		http.Error(w, "Plugin not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, fmt.Sprintf("Error accessing plugin file: %v", err), http.StatusInternalServerError)
		return
	}

	// Ensure it's a regular file
	if !fileInfo.Mode().IsRegular() {
		http.Error(w, "Invalid plugin file", http.StatusBadRequest)
		return
	}

	// Open the plugin file
	file, err := os.Open(pluginPath)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error opening plugin file: %v", err), http.StatusInternalServerError)
		return
	}
	defer file.Close()

	// Set headers
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", pluginName))
	w.Header().Set("Content-Length", fmt.Sprintf("%d", fileInfo.Size()))

	// Copy the file to the response
	if _, err := io.Copy(w, file); err != nil {
		log.Printf("Error serving plugin file %s: %v", pluginName, err)
		return
	}

	// Log the download
	log.Printf("Served plugin %s (%d bytes) to %s", pluginName, fileInfo.Size(), r.RemoteAddr)
}

// HandleUploadPlugin handles the /upload endpoint
func (as *PluginServer) HandleUploadPlugin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse the multipart form (100 MB max)
	if err := r.ParseMultipartForm(100 << 20); err != nil {
		http.Error(w, fmt.Sprintf("Error parsing form: %v", err), http.StatusBadRequest)
		return
	}

	// Get the file from the form
	file, header, err := r.FormFile("plugin")
	if err != nil {
		http.Error(w, fmt.Sprintf("Error getting file: %v", err), http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Sanitize filename
	filename := filepath.Base(header.Filename)
	if filename == "" || filename == "." || filename == ".." {
		http.Error(w, "Invalid filename", http.StatusBadRequest)
		return
	}

	// Create the destination file
	destPath := filepath.Join(as.pluginDir, filename)
	dst, err := os.Create(destPath)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error creating file: %v", err), http.StatusInternalServerError)
		return
	}
	defer dst.Close()

	// Copy the uploaded file to the destination file
	size, err := io.Copy(dst, file)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error saving file: %v", err), http.StatusInternalServerError)
		return
	}

	// Return success response
	response := UploadResponse{
		Success:  true,
		Filename: filename,
		Size:     size,
		Message:  "Plugin file uploaded successfully",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)

	// Log the upload
	log.Printf("Uploaded plugin file %s (%d bytes) from %s", filename, size, r.RemoteAddr)
}

// HandleDeletePlugin handles the /delete/{name} endpoint
func (as *PluginServer) HandleDeletePlugin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract the plugin name from the URL
	pluginName := strings.TrimPrefix(r.URL.Path, "/delete/")
	if pluginName == "" {
		http.Error(w, "Plugin name is required", http.StatusBadRequest)
		return
	}

	// Sanitize the plugin name
	pluginName = filepath.Base(pluginName)

	// Construct the full path to the plugin file
	pluginPath := filepath.Join(as.pluginDir, pluginName)

	// Check if the file exists
	if _, err := os.Stat(pluginPath); os.IsNotExist(err) {
		http.Error(w, "Plugin file not found", http.StatusNotFound)
		return
	}

	// Delete the file
	if err := os.Remove(pluginPath); err != nil {
		http.Error(w, fmt.Sprintf("Error deleting plugin item: %v", err), http.StatusInternalServerError)
		return
	}

	// Return success response
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"success": true, "message": "Plugin unit deleted successfully"}`)

	// Log the deletion
	log.Printf("Deleted plugin item %s from %s", pluginName, r.RemoteAddr)
}
