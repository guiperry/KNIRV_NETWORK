package main

import (
	"log"

	"github.com/knirvchain/internal/mcp"
)

func main() {
	server, err := mcp.NewMCPServer("localhost:8080", "test_key")
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Starting KNIRVCHAIN MCP server on :8080")
	if err := server.Start(":8080"); err != nil {
		log.Fatal(err)
	}
}