//go:build client
// +build client

package main

import (
	"KNIRVCHAIN/config"
	"log"
)

func init() {
	// This file is only compiled when the 'client' build tag is specified
	log.Println("Initializing KNIRVCHAIN Client Node")

	// Set the node role to Client
	nodeRole = config.RoleClient
}
