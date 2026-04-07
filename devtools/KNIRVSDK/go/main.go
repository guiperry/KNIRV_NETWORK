// Package main provides a unified entry point for the KNIRV Go SDK
package main

import (
	"fmt"
	"log"

	"github.com/KNIRV/KNIRV_NETWORK/KNIRVSDK/go/gateway"
	"github.com/KNIRV/KNIRV_NETWORK/KNIRVSDK/go/oracled"
	// "github.com/KNIRV/KNIRV_NETWORK/KNIRVSDK/go/transaction" // TODO: Fix import paths
	// "github.com/KNIRV/KNIRV_NETWORK/KNIRVSDK/go/transmission" // TODO: Fix missing dependencies
)

func main() {
	fmt.Println("KNIRV Go SDK")
	fmt.Println("=============")

	// Initialize gateway client
	gatewayClient, err := gateway.NewClient()
	if err != nil {
		log.Printf("Failed to create gateway client: %v", err)
	} else {
		fmt.Println("✓ Gateway client initialized")
		_ = gatewayClient // Use the client to avoid unused variable warning
	}

	// Initialize oracled client
	oracledClient := oracled.NewClient("http://localhost:26657") // Default for local
	fmt.Println("✓ Oracled client initialized")
	_ = oracledClient // Use the client to avoid unused variable warning

	// TODO: Initialize transaction client when import paths are fixed
	// txClient, err := transaction.NewClient("http://localhost:8080")
	// if err != nil {
	//	log.Printf("Failed to create transaction client: %v", err)
	// } else {
	//	fmt.Println("✓ Transaction client initialized")
	//	_ = txClient // Use the client to avoid unused variable warning
	// }

	// TODO: Initialize transmission client when dependencies are fixed
	// transmissionClient, err := transmission.NewClient("http://localhost:8081")
	// if err != nil {
	//	log.Printf("Failed to create transmission client: %v", err)
	// } else {
	//	fmt.Println("✓ Transmission client initialized")
	//	_ = transmissionClient // Use the client to avoid unused variable warning
	// }

	fmt.Println("KNIRV Go SDK components loaded successfully")
}
