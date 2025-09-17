//go:build bootnode
// +build bootnode

package main

import (
	"KNIRVORACLE/config" // Keep this for role setting
	"log"               // Keep for logging
)

func init() {
	// This file is only compiled when the 'bootnode' build tag is specified
	log.Println("Initializing KNIRVORACLE Bootnode")

	// Set the node role to Bootnode
	nodeRole = config.RoleBootnode

	// The direct OS-level signal handler setup is removed from here.
	// Signal handling will be managed by the main.go's waitForShutdownSignal.
	log.Println("Bootnode role set. Signal handling will be managed by main application logic.")
}
