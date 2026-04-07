package unit

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/KNIRV/KNIRV_NETWORK/KNIRVCLI/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWalletCreationAndLoading(t *testing.T) {
	// Create temporary directory for test wallets
	tempDir, err := os.MkdirTemp("", "wallet-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create wallet manager
	walletManager := core.NewWalletManager(tempDir, nil)

	// Generate key pair
	privateKey, err := walletManager.GenerateKeyPair()
	require.NoError(t, err)
	require.NotNil(t, privateKey)

	// Get address
	address := core.GetAddressFromPrivateKey(privateKey)
	require.NotEmpty(t, address)

	// Save wallet
	password := "test-password"
	filePath := filepath.Join(tempDir, "test-wallet.json")
	err = walletManager.SaveWallet(privateKey, password, filePath)
	require.NoError(t, err)

	// Verify file exists
	_, err = os.Stat(filePath)
	require.NoError(t, err)

	// Load wallet
	loadedKey, err := walletManager.LoadWallet(filePath, password)
	require.NoError(t, err)
	require.NotNil(t, loadedKey)

	// Test with wrong password
	_, err = walletManager.LoadWallet(filePath, "wrong-password")
	assert.Error(t, err)
}

func TestWalletListing(t *testing.T) {
	// Create temporary directory for test wallets
	tempDir, err := os.MkdirTemp("", "wallet-list-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create wallet manager
	walletManager := core.NewWalletManager(tempDir, nil)

	// List wallets (should be empty)
	wallets, err := walletManager.ListWallets()
	require.NoError(t, err)
	assert.Empty(t, wallets)

	// Create some wallets
	numWallets := 3
	for i := 0; i < numWallets; i++ {
		// Generate key pair
		privateKey, err := walletManager.GenerateKeyPair()
		require.NoError(t, err)

		// Get address
		address := core.GetAddressFromPrivateKey(privateKey)
		require.NotEmpty(t, address)

		// Save wallet
		password := "test-password"
		filePath := walletManager.GetWalletPath(address)
		err = walletManager.SaveWallet(privateKey, password, filePath)
		require.NoError(t, err)
	}

	// List wallets again
	wallets, err = walletManager.ListWallets()
	require.NoError(t, err)
	assert.Len(t, wallets, numWallets)
}

func TestWalletExport(t *testing.T) {
	// Create temporary directory for test wallets
	tempDir, err := os.MkdirTemp("", "wallet-export-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create wallet manager
	walletManager := core.NewWalletManager(tempDir, nil)

	// Generate key pair
	privateKey, err := walletManager.GenerateKeyPair()
	require.NoError(t, err)

	// Get address
	address := core.GetAddressFromPrivateKey(privateKey)
	require.NotEmpty(t, address)

	// Save wallet
	password := "test-password"
	filePath := filepath.Join(tempDir, "export-wallet.json")
	err = walletManager.SaveWallet(privateKey, password, filePath)
	require.NoError(t, err)

	// Load wallet
	loadedKey, err := walletManager.LoadWallet(filePath, password)
	require.NoError(t, err)
	require.NotNil(t, loadedKey)

	// Export to file
	exportPath := filepath.Join(tempDir, "exported-key.txt")
	exportFile, err := os.Create(exportPath)
	require.NoError(t, err)
	defer exportFile.Close()

	// Write private key to file
	privateKeyHex := core.GetPublicKeyHex(&privateKey.PublicKey)
	_, err = exportFile.WriteString(privateKeyHex)
	require.NoError(t, err)

	// Verify export file exists
	_, err = os.Stat(exportPath)
	require.NoError(t, err)
}
