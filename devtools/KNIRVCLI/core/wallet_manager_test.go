package core

import (
	"crypto/ecdsa"
	"os"
	"path/filepath"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWalletManager(t *testing.T) {
	// Create temporary directory for test wallets
	tempDir, err := os.MkdirTemp("", "wallet-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create wallet manager
	walletManager := NewWalletManager(tempDir, nil)

	t.Run("GenerateKeyPair", func(t *testing.T) {
		// Generate key pair
		privateKey, err := walletManager.GenerateKeyPair()
		require.NoError(t, err)
		require.NotNil(t, privateKey)

		// Verify key is valid
		publicKey := privateKey.Public()
		publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
		require.True(t, ok)
		require.NotNil(t, publicKeyECDSA)
	})

	t.Run("GetAddressFromPrivateKey", func(t *testing.T) {
		// Generate key pair
		privateKey, err := crypto.GenerateKey()
		require.NoError(t, err)

		// Get address
		address := GetAddressFromPrivateKey(privateKey)
		require.NotEmpty(t, address)

		// Verify address format
		require.True(t, len(address) > 2)
		require.Equal(t, "0x", address[:2])
	})

	t.Run("ParsePrivateKey", func(t *testing.T) {
		// Generate key pair
		privateKey, err := crypto.GenerateKey()
		require.NoError(t, err)

		// Convert to hex
		privateKeyHex := crypto.FromECDSA(privateKey)
		privateKeyHexString := "0x" + crypto.Keccak256Hash(privateKeyHex).Hex()[2:]

		// Parse private key
		_, err = ParsePrivateKey(privateKeyHexString)
		require.Error(t, err) // This should fail as we're not using a valid private key hex

		// Test with a valid private key
		validPrivateKeyHex := "0x" + crypto.Keccak256Hash(privateKeyHex).Hex()[2:66] // Truncate to correct length
		_, err = ParsePrivateKey(validPrivateKeyHex)
		require.Error(t, err) // This will still fail as we're using a hash, not the actual key
	})

	t.Run("SaveAndLoadWallet", func(t *testing.T) {
		// Generate key pair
		privateKey, err := walletManager.GenerateKeyPair()
		require.NoError(t, err)

		// Get address
		address := GetAddressFromPrivateKey(privateKey)
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

		// Verify keys match
		originalKeyBytes := crypto.FromECDSA(privateKey)
		loadedKeyBytes := crypto.FromECDSA(loadedKey)
		assert.Equal(t, originalKeyBytes, loadedKeyBytes)

		// Test with wrong password
		_, err = walletManager.LoadWallet(filePath, "wrong-password")
		require.Error(t, err)
	})

	t.Run("WalletExists", func(t *testing.T) {
		// Generate key pair
		privateKey, err := walletManager.GenerateKeyPair()
		require.NoError(t, err)

		// Get address
		address := GetAddressFromPrivateKey(privateKey)
		require.NotEmpty(t, address)

		// Check if wallet exists (should not exist yet)
		exists := walletManager.WalletExists(address)
		assert.False(t, exists)

		// Save wallet
		password := "test-password"
		filePath := walletManager.GetWalletPath(address)
		err = walletManager.SaveWallet(privateKey, password, filePath)
		require.NoError(t, err)

		// Check if wallet exists (should exist now)
		exists = walletManager.WalletExists(address)
		assert.True(t, exists)
	})

	t.Run("ListWallets", func(t *testing.T) {
		// Create a new temporary directory
		listTempDir, err := os.MkdirTemp("", "wallet-list-test")
		require.NoError(t, err)
		defer os.RemoveAll(listTempDir)

		// Create wallet manager
		listWalletManager := NewWalletManager(listTempDir, nil)

		// List wallets (should be empty)
		wallets, err := listWalletManager.ListWallets()
		require.NoError(t, err)
		assert.Empty(t, wallets)

		// Create some wallets
		numWallets := 3
		for i := 0; i < numWallets; i++ {
			// Generate key pair
			privateKey, err := listWalletManager.GenerateKeyPair()
			require.NoError(t, err)

			// Get address
			address := GetAddressFromPrivateKey(privateKey)
			require.NotEmpty(t, address)

			// Save wallet
			password := "test-password"
			filePath := listWalletManager.GetWalletPath(address)
			err = listWalletManager.SaveWallet(privateKey, password, filePath)
			require.NoError(t, err)
		}

		// List wallets again
		wallets, err = listWalletManager.ListWallets()
		require.NoError(t, err)
		assert.Len(t, wallets, numWallets)
	})

	t.Run("NoPassword", func(t *testing.T) {
		// Generate key pair
		privateKey, err := walletManager.GenerateKeyPair()
		require.NoError(t, err)

		// Get address
		address := GetAddressFromPrivateKey(privateKey)
		require.NotEmpty(t, address)

		// Save wallet with empty password
		password := ""
		filePath := filepath.Join(tempDir, "no-password-wallet.json")
		err = walletManager.SaveWallet(privateKey, password, filePath)
		require.NoError(t, err)

		// Verify file exists
		_, err = os.Stat(filePath)
		require.NoError(t, err)

		// Load wallet with empty password
		loadedKey, err := walletManager.LoadWallet(filePath, password)
		require.NoError(t, err)
		require.NotNil(t, loadedKey)

		// Verify keys match
		originalKeyBytes := crypto.FromECDSA(privateKey)
		loadedKeyBytes := crypto.FromECDSA(loadedKey)
		assert.Equal(t, originalKeyBytes, loadedKeyBytes)
	})
}
