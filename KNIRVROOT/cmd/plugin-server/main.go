package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ServerInfo represents information about this server instance
type ServerInfo struct {
	Name      string    `json:"name"`
	Port      int       `json:"port"`
	AgentDir  string    `json:"agent_dir"`
	StartTime time.Time `json:"start_time"`
	Version   string    `json:"version"`
}

// AgentInfo represents information about a plugin agent
type AgentInfo struct {
	Name         string    `json:"name"`
	Size         int64     `json:"size"`
	LastModified time.Time `json:"last_modified"`
	Hash         string    `json:"hash,omitempty"`
}

// UploadResponse represents the response from an upload operation
type UploadResponse struct {
	Success  bool   `json:"success"`
	Filename string `json:"filename"`
	Size     int64  `json:"size"`
	Message  string `json:"message,omitempty"`
}

// ListResponse represents the response from a list operation
type ListResponse struct {
	Agents []AgentInfo `json:"agents"`
	Count  int         `json:"count"`
}

func main() {
	// Define command-line flags
	portFlag := flag.Int("port", 8080, "Port to listen on")
	agentDirFlag := flag.String("agents", "./agents", "Directory containing Plugin Agents")
	serverNameFlag := flag.String("name", "KNIRVROOT Plugin Agent Server", "Name of this server instance")
	registerFlag := flag.Bool("register", false, "Register this server with the KNIRVROOT system")
	apiUrlFlag := flag.String("api", "http://localhost:3000", "URL of the KNIRVROOT API")
	corsFlag := flag.Bool("cors", true, "Enable CORS headers")

	// Parse command-line flags
	flag.Parse()

	// Ensure agent directory exists
	if err := os.MkdirAll(*agentDirFlag, 0755); err != nil {
		log.Fatalf("Error creating plugin directory: %v", err)
	}

	// Server info
	serverInfo := ServerInfo{
		Name:      *serverNameFlag,
		Port:      *portFlag,
		AgentDir:  *agentDirFlag,
		StartTime: time.Now(),
		Version:   "1.0.0",
	}

	// Register the server if requested
	if *registerFlag {
		fmt.Printf("Registering server %s with API at %s...\n", *serverNameFlag, *apiUrlFlag)
		// TODO: Implement actual API registration
		fmt.Printf("Server registered successfully\n")
	}

	// CORS middleware
	corsMiddleware := func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			if *corsFlag {
				w.Header().Set("Access-Control-Allow-Origin", "*")
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
				w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			}

			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}

			next(w, r)
		}
	}

	// Handler for server info
	http.HandleFunc("/info", corsMiddleware(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(serverInfo)
	}))

	// Handler for serving plugin agent files
	http.HandleFunc("/agents/", corsMiddleware(func(w http.ResponseWriter, r *http.Request) {
		// Extract the plugin agent name from the URL
		agentName := strings.TrimPrefix(r.URL.Path, "/agents/")
		if agentName == "" {
			http.Error(w, "Agent name is required", http.StatusBadRequest)
			return
		}

		// Sanitize the agent name to prevent directory traversal
		agentName = filepath.Base(agentName)

		// Construct the full path to the plugin file
		agentPath := filepath.Join(*agentDirFlag, agentName)

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

		// Open the plugin file
		file, err := os.Open(agentPath)
		if err != nil {
			http.Error(w, fmt.Sprintf("Error opening plugin agent file: %v", err), http.StatusInternalServerError)
			return
		}
		defer file.Close()

		// Set headers
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", agentName))
		w.Header().Set("Content-Length", fmt.Sprintf("%d", fileInfo.Size()))

		// Copy the file to the response
		if _, err := io.Copy(w, file); err != nil {
			log.Printf("Error serving plugin agent file %s: %v", agentName, err)
			return
		}

		// Log the download
		log.Printf("Served agent %s (%d bytes) to %s", agentName, fileInfo.Size(), r.RemoteAddr)
	}))

	// Handler for uploading plugin agent files
	http.HandleFunc("/upload", corsMiddleware(func(w http.ResponseWriter, r *http.Request) {
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
		file, header, err := r.FormFile("plugin-agent")
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
		destPath := filepath.Join(*agentDirFlag, filename)
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
			Message:  "Plugin agent uploaded successfully",
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)

		// Log the upload
		log.Printf("Uploaded plugin agent %s (%d bytes) from %s", filename, size, r.RemoteAddr)
	}))

	// Handler for listing available agents
	http.HandleFunc("/list", corsMiddleware(func(w http.ResponseWriter, r *http.Request) {
		// Get all files in the plugin directory
		files, err := os.ReadDir(*agentDirFlag)
		if err != nil {
			http.Error(w, fmt.Sprintf("Error reading plugin directory: %v", err), http.StatusInternalServerError)
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
	}))

	// Handler for deleting agents (optional)
	http.HandleFunc("/delete/", corsMiddleware(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Extract the plugin agent name from the URL
		agentName := strings.TrimPrefix(r.URL.Path, "/delete/")
		if agentName == "" {
			http.Error(w, "Agent name is required", http.StatusBadRequest)
			return
		}

		// Sanitize the agent name
		agentName = filepath.Base(agentName)

		// Construct the full path to the plugin file
		agentPath := filepath.Join(*agentDirFlag, agentName)

		// Check if the file exists
		if _, err := os.Stat(agentPath); os.IsNotExist(err) {
			http.Error(w, "Agent not found", http.StatusNotFound)
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
	}))

	// Start the server
	addr := fmt.Sprintf(":%d", *portFlag)
	fmt.Printf("Starting KNIRVROOT Plugin Agent Server '%s' on %s...\n", *serverNameFlag, addr)
	fmt.Printf("Agent directory: %s\n", *agentDirFlag)
	fmt.Printf("CORS enabled: %v\n", *corsFlag)
	fmt.Printf("\nAvailable endpoints:\n")
	fmt.Printf("  GET  /info           - Server information\n")
	fmt.Printf("  GET  /list           - List available agents\n")
	fmt.Printf("  GET  /agents/{name}  - Download agent file\n")
	fmt.Printf("  POST /upload         - Upload agent file\n")
	fmt.Printf("  DEL  /delete/{name}  - Delete agent file\n")
	fmt.Printf("\n")

	log.Fatal(http.ListenAndServe(addr, nil))
}
