package main

import (
	"embed"
	"io/fs"
	"log"
	"net/http"
	"os"
)

//go:embed documentation/docsify documentation/docsify/* documentation/docsify/**/*
var docsifyFiles embed.FS

// setupDocsHandler sets up the HTTP handler for serving the embedded documentation
func setupDocsHandler(mux *http.ServeMux) {
	// Create a sub-filesystem for the docsify directory
	docsFS, err := fs.Sub(docsifyFiles, "documentation/docsify")
	if err != nil {
		log.Printf("Error setting up documentation filesystem: %v", err)
		return
	}

	// Create a file server for the documentation
	fileServer := http.FileServer(http.FS(docsFS))

	// Register the handler for the /docs/ path
	mux.Handle("/docs/", http.StripPrefix("/docs/", fileServer))

	log.Printf("Documentation available at http://localhost:<port>/docs/")
}

// Example of how to use this in your main.go:
/*
func main() {
	// Create a new HTTP server mux
	mux := http.NewServeMux()

	// Set up your regular API handlers
	// ...

	// Set up the documentation handler
	setupDocsHandler(mux)

	// Start the server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server starting on port %s", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatalf("Error starting server: %v", err)
	}
}
*/

// For testing this file directly
func docs() {
	if len(os.Args) > 1 && os.Args[1] == "serve-docs" {
		mux := http.NewServeMux()
		setupDocsHandler(mux)

		port := "8080"
		log.Printf("Documentation server starting on http://localhost:%s/docs/", port)
		if err := http.ListenAndServe(":"+port, mux); err != nil {
			log.Fatalf("Error starting server: %v", err)
		}
	} else {
		log.Println("This file provides documentation embedding functionality.")
		log.Println("Run with 'serve-docs' argument to start a standalone documentation server.")
	}
}

func init() {
	docs()
}
