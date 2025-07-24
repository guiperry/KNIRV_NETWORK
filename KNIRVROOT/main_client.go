//go:build client
// +build client

package main

import (
	"KNIRVROOT/config"
	"log"
)

func init() {
	// This file is only compiled when the 'client' build tag is specified
	log.Println("Initializing KNIRVROOT Client Node")

	// Set the node role to Client
	nodeRole = config.RoleClient

	// Enable terminal UI by default for client nodes
	useTerminalUI = true

	log.Println("Client mode: Terminal UI enabled by default")
}
