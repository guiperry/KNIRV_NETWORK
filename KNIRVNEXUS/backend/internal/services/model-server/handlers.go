package objectserver

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
func (as *ModelServer) HandleServerInfo(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(as.serverInfo)
}

// HandleListModels handles the /list endpoint
func (as *ModelServer) HandleListModels(w http.ResponseWriter, r *http.Request) {
	// Get all files in the model directory
	files, err := os.ReadDir(as.modelDir)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error reading model directory: %v", err), http.StatusInternalServerError)
		return
	}

	// Build the response
	var objects []ModelInfo
	for _, file := range files {
		if !file.IsDir() && (strings.HasSuffix(file.Name(), ".so") || strings.HasSuffix(file.Name(), ".wasm")) {
			info, err := file.Info()
			if err != nil {
				log.Printf("Error getting file info for %s: %v", file.Name(), err)
				continue
			}

			objects = append(objects, ModelInfo{
				Name:         file.Name(),
				Size:         info.Size(),
				LastModified: info.ModTime(),
			})
		}
	}

	// Return the list of objects
	response := ListResponse{
		Models: objects,
		Count:  len(objects),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// HandleDownloadModel handles the /objects/{name} endpoint
func (as *ModelServer) HandleDownloadModel(w http.ResponseWriter, r *http.Request) {
	// Extract the model name from the URL
	modelName := strings.TrimPrefix(r.URL.Path, "/objects/")
	if modelName == "" {
		http.Error(w, "Model name is required", http.StatusBadRequest)
		return
	}

	// Sanitize the model name to prevent directory traversal
	modelName = filepath.Base(modelName)

	// Construct the full path to the model file
	modelPath := filepath.Join(as.modelDir, modelName)

	// Check if the file exists
	fileInfo, err := os.Stat(modelPath)
	if os.IsNotExist(err) {
		http.Error(w, "Model not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, fmt.Sprintf("Error accessing model file: %v", err), http.StatusInternalServerError)
		return
	}

	// Ensure it's a regular file
	if !fileInfo.Mode().IsRegular() {
		http.Error(w, "Invalid model file", http.StatusBadRequest)
		return
	}

	// Open the model file
	file, err := os.Open(modelPath)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error opening model file: %v", err), http.StatusInternalServerError)
		return
	}
	defer file.Close()

	// Set headers
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", modelName))
	w.Header().Set("Content-Length", fmt.Sprintf("%d", fileInfo.Size()))

	// Copy the file to the response
	if _, err := io.Copy(w, file); err != nil {
		log.Printf("Error serving model file %s: %v", modelName, err)
		return
	}

	// Log the download
	log.Printf("Served model %s (%d bytes) to %s", modelName, fileInfo.Size(), r.RemoteAddr)
}

// HandleUploadModel handles the /upload endpoint
func (as *ModelServer) HandleUploadModel(w http.ResponseWriter, r *http.Request) {
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
	file, header, err := r.FormFile("model")
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
	destPath := filepath.Join(as.modelDir, filename)
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
		Message:  "Model file uploaded successfully",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)

	// Log the upload
	log.Printf("Uploaded model file %s (%d bytes) from %s", filename, size, r.RemoteAddr)
}

// HandleDeleteModel handles the /delete/{name} endpoint
func (as *ModelServer) HandleDeleteModel(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract the model name from the URL
	modelName := strings.TrimPrefix(r.URL.Path, "/delete/")
	if modelName == "" {
		http.Error(w, "Model name is required", http.StatusBadRequest)
		return
	}

	// Sanitize the model name
	modelName = filepath.Base(modelName)

	// Construct the full path to the model file
	modelPath := filepath.Join(as.modelDir, modelName)

	// Check if the file exists
	if _, err := os.Stat(modelPath); os.IsNotExist(err) {
		http.Error(w, "Model file not found", http.StatusNotFound)
		return
	}

	// Delete the file
	if err := os.Remove(modelPath); err != nil {
		http.Error(w, fmt.Sprintf("Error deleting model: %v", err), http.StatusInternalServerError)
		return
	}

	// Return success response
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"success": true, "message": "Model deleted successfully"}`)

	// Log the deletion
	log.Printf("Deleted model %s from %s", modelName, r.RemoteAddr)
}
