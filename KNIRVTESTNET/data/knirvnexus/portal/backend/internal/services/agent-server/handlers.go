package agentserver

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
func (as *AgentServer) HandleServerInfo(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(as.serverInfo)
}

// HandleListAgents handles the /list endpoint
func (as *AgentServer) HandleListAgents(w http.ResponseWriter, r *http.Request) {
	// Get all files in the agent directory
	files, err := os.ReadDir(as.agentDir)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error reading agent directory: %v", err), http.StatusInternalServerError)
		return
	}

	// Build the response
	var agents []AgentInfo
	for _, file := range files {
		if !file.IsDir() && (strings.HasSuffix(file.Name(), ".so") || strings.HasSuffix(file.Name(), ".wasm")) {
			info, err := file.Info()
			if err != nil {
				log.Printf("Error getting file info for %s: %v", file.Name(), err)
				continue
			}

			agents = append(agents, AgentInfo{
				Name:         file.Name(),
				Size:         info.Size(),
				LastModified: info.ModTime(),
			})
		}
	}

	// Return the list of agents
	response := ListResponse{
		Agents: agents,
		Count:  len(agents),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// HandleDownloadAgent handles the /agents/{name} endpoint
func (as *AgentServer) HandleDownloadAgent(w http.ResponseWriter, r *http.Request) {
	// Extract the agent name from the URL
	agentName := strings.TrimPrefix(r.URL.Path, "/agents/")
	if agentName == "" {
		http.Error(w, "Agent name is required", http.StatusBadRequest)
		return
	}

	// Sanitize the agent name to prevent directory traversal
	agentName = filepath.Base(agentName)

	// Construct the full path to the agent file
	agentPath := filepath.Join(as.agentDir, agentName)

	// Check if the file exists
	fileInfo, err := os.Stat(agentPath)
	if os.IsNotExist(err) {
		http.Error(w, "Agent not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, fmt.Sprintf("Error accessing agent file: %v", err), http.StatusInternalServerError)
		return
	}

	// Ensure it's a regular file
	if !fileInfo.Mode().IsRegular() {
		http.Error(w, "Invalid agent file", http.StatusBadRequest)
		return
	}

	// Open the agent file
	file, err := os.Open(agentPath)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error opening agent file: %v", err), http.StatusInternalServerError)
		return
	}
	defer file.Close()

	// Set headers
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", agentName))
	w.Header().Set("Content-Length", fmt.Sprintf("%d", fileInfo.Size()))

	// Copy the file to the response
	if _, err := io.Copy(w, file); err != nil {
		log.Printf("Error serving agent file %s: %v", agentName, err)
		return
	}

	// Log the download
	log.Printf("Served agent %s (%d bytes) to %s", agentName, fileInfo.Size(), r.RemoteAddr)
}

// HandleUploadAgent handles the /upload endpoint
func (as *AgentServer) HandleUploadAgent(w http.ResponseWriter, r *http.Request) {
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
	file, header, err := r.FormFile("agent")
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
	destPath := filepath.Join(as.agentDir, filename)
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
		Message:  "Agent file uploaded successfully",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)

	// Log the upload
	log.Printf("Uploaded agent file %s (%d bytes) from %s", filename, size, r.RemoteAddr)
}

// HandleDeleteAgent handles the /delete/{name} endpoint
func (as *AgentServer) HandleDeleteAgent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract the agent name from the URL
	agentName := strings.TrimPrefix(r.URL.Path, "/delete/")
	if agentName == "" {
		http.Error(w, "Agent name is required", http.StatusBadRequest)
		return
	}

	// Sanitize the agent name
	agentName = filepath.Base(agentName)

	// Construct the full path to the agent file
	agentPath := filepath.Join(as.agentDir, agentName)

	// Check if the file exists
	if _, err := os.Stat(agentPath); os.IsNotExist(err) {
		http.Error(w, "Agent file not found", http.StatusNotFound)
		return
	}

	// Delete the file
	if err := os.Remove(agentPath); err != nil {
		http.Error(w, fmt.Sprintf("Error deleting agent: %v", err), http.StatusInternalServerError)
		return
	}

	// Return success response
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"success": true, "message": "Agent deleted successfully"}`)

	// Log the deletion
	log.Printf("Deleted agent %s from %s", agentName, r.RemoteAddr)
}
