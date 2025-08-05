package benchmark

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/guiperry/KNIRVCHAIN-CLI/core"
)

func BenchmarkWalletCreation(b *testing.B) {
	// Create temporary directory for test wallets
	tempDir, err := os.MkdirTemp("", "wallet-bench-test")
	if err != nil {
		b.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create wallet manager
	walletManager, err := core.NewWalletManager(tempDir)
	if err != nil {
		b.Fatalf("Failed to create wallet manager: %v", err)
	}

	// Reset timer before the loop
	b.ResetTimer()

	// Run the benchmark
	for i := 0; i < b.N; i++ {
		// Generate key pair
		privateKey, err := walletManager.GenerateKeyPair()
		if err != nil {
			b.Fatalf("Failed to generate key pair: %v", err)
		}

		// Get address
		address := core.GetAddressFromPrivateKey(privateKey)

		// Save wallet
		password := "test-password"
		filePath := filepath.Join(tempDir, address+".json")
		err = walletManager.SaveWallet(privateKey, password, filePath)
		if err != nil {
			b.Fatalf("Failed to save wallet: %v", err)
		}
	}
}

func BenchmarkWalletLoading(b *testing.B) {
	// Create temporary directory for test wallets
	tempDir, err := os.MkdirTemp("", "wallet-bench-test")
	if err != nil {
		b.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create wallet manager
	walletManager, err := core.NewWalletManager(tempDir)
	if err != nil {
		b.Fatalf("Failed to create wallet manager: %v", err)
	}

	// Generate key pair
	privateKey, err := walletManager.GenerateKeyPair()
	if err != nil {
		b.Fatalf("Failed to generate key pair: %v", err)
	}

	// Get address
	address := core.GetAddressFromPrivateKey(privateKey)

	// Save wallet
	password := "test-password"
	filePath := filepath.Join(tempDir, address+".json")
	err = walletManager.SaveWallet(privateKey, password, filePath)
	if err != nil {
		b.Fatalf("Failed to save wallet: %v", err)
	}

	// Reset timer before the loop
	b.ResetTimer()

	// Run the benchmark
	for i := 0; i < b.N; i++ {
		// Load wallet
		_, err := walletManager.LoadWallet(filePath, password)
		if err != nil {
			b.Fatalf("Failed to load wallet: %v", err)
		}
	}
}

func BenchmarkTransactionSigning(b *testing.B) {
	// Create wallet manager
	walletManager, err := core.NewWalletManager(".")
	if err != nil {
		b.Fatalf("Failed to create wallet manager: %v", err)
	}

	// Generate key pair
	privateKey, err := walletManager.GenerateKeyPair()
	if err != nil {
		b.Fatalf("Failed to generate key pair: %v", err)
	}

	// Create unsigned transaction details
	unsignedTxDetails := core.UnsignedTransactionDetails{
		From:      "0x1234567890abcdef1234567890abcdef12345678",
		To:        "0xabcdef1234567890abcdef1234567890abcdef12",
		Value:     100,
		Data:      map[string]interface{}{"test": "data"},
		Timestamp: 1625097600,
		Fee:       10,
		Type:      "TEST",
	}

	// Convert data to bytes
	dataBytes := []byte(`{"test":"data"}`)

	// Reset timer before the loop
	b.ResetTimer()

	// Run the benchmark
	for i := 0; i < b.N; i++ {
		// Sign the transaction
		_, _, err := core.SignTransactionData(privateKey, unsignedTxDetails, dataBytes)
		if err != nil {
			b.Fatalf("Failed to sign transaction: %v", err)
		}
	}
}