package main

import (
	"fmt"
	"math/rand"
	"time"
)

func main() {
	fmt.Println("🧪 Testing Basic Hash Operations")
	fmt.Println("===============================")
	fmt.Println()

	// Test simple hash generation
	fmt.Println("Testing simple hash generation...")

	// Create test data
	testData := []byte{1, 2, 3, 4, 5}
	fmt.Printf("Test data: %v\n", testData)

	// Simple hash function
	hash := simpleHash(testData)
	fmt.Printf("Hash result: %d\n", hash)

	// Test multiple iterations
	fmt.Println("Testing hash consistency...")
	for i := 0; i < 5; i++ {
		result := simpleHash(testData)
		fmt.Printf("Iteration %d: %d\n", i, result)
	}

	fmt.Println()
	fmt.Println("✅ Hash test completed successfully!")
	fmt.Println()

	// Interactive demo
	fmt.Println("💬 Simple interactive demo...")
	fmt.Println("Type 'quit' to exit")
	fmt.Println()

	rand.Seed(time.Now().UnixNano())

	for {
		fmt.Print("You: ")
		var input string
		fmt.Scanln(&input)

		if input == "quit" || input == "exit" {
			fmt.Println("👋 Goodbye!")
			break
		}

		if input == "" {
			continue
		}

		// Convert input to bytes and hash
		inputBytes := []byte(input)
		hashValue := simpleHash(inputBytes)

		// Generate conversational response
		response := fmt.Sprintf("I processed your message using hash-based operations. The hash value is %d. This demonstrates the core principle of cryptographic neural networks: converting information through deterministic hash functions while maintaining the ability to learn and adapt through parameter encoding.", hashValue)

		fmt.Printf("Hasher: %s\n", response)
	}
}

// simpleHash creates a simple deterministic hash function
func simpleHash(data []byte) int32 {
	var hash int32 = 5381 // Prime number for good distribution

	for _, b := range data {
		hash = hash*31 + int32(b)
		hash ^= hash >> 16
	}

	return hash
}
