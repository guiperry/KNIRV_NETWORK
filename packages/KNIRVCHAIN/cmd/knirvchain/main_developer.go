//go:build developer
// +build developer

package main

import (
	"KNIRVCHAIN/config"
	"log"
)

func init() {
	// This file is only compiled when the 'developer' build tag is specified
	log.Println("Initializing KNIRVCHAIN Developer Node")

	// Set the node role to Peer (Developer)
	nodeRole = config.RolePeer
}
