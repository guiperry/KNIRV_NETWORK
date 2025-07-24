//go:build developer
// +build developer

package main

import (
	"KNIRVROOT/config"
	"log"
)

func init() {
	// This file is only compiled when the 'developer' build tag is specified
	log.Println("Initializing KNIRVROOT Developer Node")

	// Set the node role to Peer (Developer)
	nodeRole = config.RolePeer

	// Enable terminal UI by default for developer nodes
	useTerminalUI = true

	log.Println("Developer mode: Terminal UI enabled by default")
}
