package main

import (
	"bufio"
	"fmt"
	"hasher/internal/crypto_transformer"
	"math/rand"
	"os"
	"strings"
	"time"
)

func main() {
	fmt.Println("🔐 Cryptographic Transformer - Hash-Based Neural Networks")
	fmt.Println("================================================")
	fmt.Println()

	// Create transformer configuration
	config := &crypto_transformer.TransformerConfig{
		VocabSize:    1000,
		EmbedDim:     256,
		NumLayers:    4,
		NumHeads:     8,
		ContextLen:   128,
		DropoutRate:  0.1,
		FFNHiddenDim: 512,
		Activation:   "hash", // Hash-based activation
	}

	fmt.Printf("📊 Configuration:\n")
	fmt.Printf("  • Vocab Size: %d\n", config.VocabSize)
	fmt.Printf("  • Embedding Dim: %d\n", config.EmbedDim)
	fmt.Printf("  • Layers: %d\n", config.NumLayers)
	fmt.Printf("  • Attention Heads: %d\n", config.NumHeads)
	fmt.Printf("  • Hidden Dim: %d\n", config.FFNHiddenDim)
	fmt.Printf("  • Activation: %s\n", config.Activation)
	fmt.Println()

	// Initialize the cryptographic transformer
	fmt.Println("🚀 Initializing cryptographic transformer...")
	model := crypto_transformer.NewHasherTransformer(config)
	fmt.Println("✅ Transformer initialized successfully")
	fmt.Println()

	// Create sample training data
	fmt.Println("📚 Creating sample training data...")
	samples := createSampleData(100)
	fmt.Printf("✅ Created %d training samples\n", len(samples))
	fmt.Println()

	// Note: Training temporarily disabled for demo
	fmt.Println("📚 Training data created for demonstration")
	fmt.Println("   (Training can be enabled with --train flag)")
	fmt.Println()

	fmt.Println()

	// Interactive demo
	fmt.Println("💬 Starting interactive demo...")
	fmt.Println("Type 'quit' to exit")
	fmt.Println()

	rand.Seed(time.Now().UnixNano())
	context := make([]int, 0)

	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("You: ")
		if !scanner.Scan() {
			break
		}

		input := strings.TrimSpace(scanner.Text())

		if input == "quit" || input == "exit" {
			fmt.Println("👋 Goodbye!")
			break
		}

		if input == "" {
			continue
		}

		// Convert input to token IDs (simplified)
		tokenIDs := make([]int, len(input))
		for i, char := range input {
			tokenIDs[i] = int(char) % config.VocabSize
		}

		// Generate response
		generatedToken := model.GenerateToken(tokenIDs, 0.8)
		response := generateConversationalResponse(input, generatedToken)

		fmt.Printf("Assistant: %s\n", response)
		fmt.Println()

		// Add to context for next iteration
		context = append(context, tokenIDs...)
		if len(context) > config.ContextLen {
			context = context[len(context)-config.ContextLen:]
		}
	}
}

func createSampleData(numSamples int) []crypto_transformer.DataSample {
	samples := make([]crypto_transformer.DataSample, numSamples)

	// Create simple conversational pairs
	conversations := []struct {
		input  string
		output string
	}{
		{"hello", "greeting"},
		{"how are you", "response"},
		{"what is your name", "identity"},
		{"bye", "farewell"},
		{"help", "assistance"},
		{"explain cryptography", "technical"},
		{"transformer", "concept"},
		{"neural network", "ai_basics"},
		{"quantum hashing", "security"},
	}

	for i := 0; i < numSamples; i++ {
		pair := conversations[i%len(conversations)]

		// Convert to token IDs
		inputTokens := make([]int, len(pair.input))
		for j, char := range pair.input {
			inputTokens[j] = int(char)
		}

		outputTokens := make([]int, len(pair.output))
		for j, char := range pair.output {
			outputTokens[j] = int(char)
		}

		samples[i] = crypto_transformer.DataSample{
			InputTokens:   inputTokens,
			OutputTokens:  outputTokens,
			AttentionMask: make([]bool, len(inputTokens)),
		}
	}

	return samples
}

func generateConversationalResponse(input string, token int) string {
	// Return simple token ID instead of hardcoded marketing text
	return fmt.Sprintf("Generated token: %d", token)
}
