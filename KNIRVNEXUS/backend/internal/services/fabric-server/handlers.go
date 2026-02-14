package fabricserver

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
func (as *FabricServer) HandleServerInfo(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(as.serverInfo)
}

// HandleListFabrics handles the /list endpoint
func (as *FabricServer) HandleListFabrics(w http.ResponseWriter, r *http.Request) {
	// Get all files in the fabric directory
	files, err := os.ReadDir(as.fabricDir)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error reading fabric directory: %v", err), http.StatusInternalServerError)
		return
	}

	// Build the response
	var fabricUnits []FabricInfo
	for _, file := range files {
		if !file.IsDir() && (strings.HasSuffix(file.Name(), ".so") || strings.HasSuffix(file.Name(), ".wasm")) {
			info, err := file.Info()
			if err != nil {
				log.Printf("Error getting file info for %s: %v", file.Name(), err)
				continue
			}

			fabricUnits = append(fabricUnits, FabricInfo{
				Name:         file.Name(),
				Size:         info.Size(),
				LastModified: info.ModTime(),
			})
		}
	}

	// Return the list of fabric units
	response := ListResponse{
		Fabrics: fabricUnits,
		Count:   len(fabricUnits),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// HandleDownloadFabric handles the /objects/{name} endpoint
func (as *FabricServer) HandleDownloadFabric(w http.ResponseWriter, r *http.Request) {
	// Extract the fabric name from the URL
	fabricName := strings.TrimPrefix(r.URL.Path, "/objects/")
	if fabricName == "" {
		http.Error(w, "Fabric name is required", http.StatusBadRequest)
		return
	}

	// Sanitize the fabric name to prevent directory traversal
	fabricName = filepath.Base(fabricName)

	// Construct the full path to the fabric file
	fabricPath := filepath.Join(as.fabricDir, fabricName)

	// Check if the file exists
	fileInfo, err := os.Stat(fabricPath)
	if os.IsNotExist(err) {
		http.Error(w, "Fabric not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, fmt.Sprintf("Error accessing fabric file: %v", err), http.StatusInternalServerError)
		return
	}

	// Ensure it's a regular file
	if !fileInfo.Mode().IsRegular() {
		http.Error(w, "Invalid fabric file", http.StatusBadRequest)
		return
	}

	// Open the fabric file
	file, err := os.Open(fabricPath)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error opening fabric file: %v", err), http.StatusInternalServerError)
		return
	}
	defer file.Close()

	// Set headers
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", fabricName))
	w.Header().Set("Content-Length", fmt.Sprintf("%d", fileInfo.Size()))

	// Copy the file to the response
	if _, err := io.Copy(w, file); err != nil {
		log.Printf("Error serving fabric file %s: %v", fabricName, err)
		return
	}

	// Log the download
	log.Printf("Served fabric %s (%d bytes) to %s", fabricName, fileInfo.Size(), r.RemoteAddr)
}

// HandleUploadFabric handles the /upload endpoint
func (as *FabricServer) HandleUploadFabric(w http.ResponseWriter, r *http.Request) {
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
	file, header, err := r.FormFile("fabric")
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
	destPath := filepath.Join(as.fabricDir, filename)
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
		Message:  "Fabric file uploaded successfully",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)

	// Log the upload
	log.Printf("Uploaded fabric file %s (%d bytes) from %s", filename, size, r.RemoteAddr)
}

// HandleDeleteFabric handles the /delete/{name} endpoint
func (as *FabricServer) HandleDeleteFabric(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract the fabric name from the URL
	fabricName := strings.TrimPrefix(r.URL.Path, "/delete/")
	if fabricName == "" {
		http.Error(w, "Fabric name is required", http.StatusBadRequest)
		return
	}

	// Sanitize the fabric name
	fabricName = filepath.Base(fabricName)

	// Construct the full path to the fabric file
	fabricPath := filepath.Join(as.fabricDir, fabricName)

	// Check if the file exists
	if _, err := os.Stat(fabricPath); os.IsNotExist(err) {
		http.Error(w, "Fabric file not found", http.StatusNotFound)
		return
	}

	// Delete the file
	if err := os.Remove(fabricPath); err != nil {
		http.Error(w, fmt.Sprintf("Error deleting fabric item: %v", err), http.StatusInternalServerError)
		return
	}

	// Return success response
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"success": true, "message": "Fabric unit deleted successfully"}`)

	// Log the deletion
	log.Printf("Deleted fabric item %s from %s", fabricName, r.RemoteAddr)
}
