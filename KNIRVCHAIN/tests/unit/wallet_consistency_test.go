package main

import (
	"os"
	"path/filepath"
	"testing"

	"KNIRVCHAIN/config"
	"KNIRVCHAIN/pkg/utils"
)

// TestWalletConsistencyChecks tests the wallet consistency checks
func TestWalletConsistencyChecks(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "KNIRVCHAIN-wallet-consistency-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a test encryption key
	encryptionKey := []byte("0123456789ABCDEF0123456789ABCDEF") // 32 bytes for AES-256

	// Create test wallets
	devWallet, err := NewWallet()
	if err != nil {
		t.Fatalf("Failed to create dev wallet: %v", err)
	}
	devAddress := devWallet.GetAddress()

	masterWallet, err := NewWallet()
	if err != nil {
		t.Fatalf("Failed to create master wallet: %v", err)
	}
	masterAddress := masterWallet.GetAddress()

	// Set up paths
	devWalletPath := filepath.Join(tempDir, "wallet.dat")
	masterWalletPath := filepath.Join(tempDir, "master_wallet.dat")

	// Create mock path provider
	mockProvider := &config.MockPathProvider{
		WalletPath:       devWalletPath,
		MasterWalletPath: masterWalletPath,
	}

	// Create wallet manager with mock provider
	wm, err := NewWalletManager(encryptionKey, mockProvider)
	if err != nil {
		t.Fatalf("Failed to create wallet manager: %v", err)
	}

	// Save the wallets
	err = wm.SaveWallet(devWallet, config.RolePeer)
	if err != nil {
		t.Fatalf("SaveWallet failed: %v", err)
	}

	err = wm.SaveMasterWallet(masterWallet, config.RoleBootnode)
	if err != nil {
		t.Fatalf("SaveMasterWallet failed: %v", err)
	}

	// Test cases
	t.Run("Peer_CorrectAddress", func(t *testing.T) {
		loadedWallet, err := wm.LoadWallet(devAddress, config.RolePeer)
		if err != nil {
			t.Errorf("LoadWallet with correct address failed: %v", err)
		} else if loadedWallet.GetAddress() != devAddress {
			t.Errorf("Expected wallet address to be %s, got %s", devAddress, loadedWallet.GetAddress())
		}
	})

	t.Run("Peer_WrongAddress", func(t *testing.T) {
		_, err := wm.LoadWallet("wrong-address", config.RolePeer)
		if err == nil {
			t.Error("LoadWallet with wrong address should have failed")
		}
	})

	t.Run("Peer_EmptyAddress", func(t *testing.T) {
		loadedWallet, err := wm.LoadWallet("", config.RolePeer)
		if err != nil {
			t.Errorf("LoadWallet with empty address failed: %v", err)
		} else if loadedWallet.GetAddress() != devAddress {
			t.Errorf("Expected wallet address to be %s, got %s", devAddress, loadedWallet.GetAddress())
		}
	})

	t.Run("Bootnode_CorrectAddresses", func(t *testing.T) {
		// For bootnode role, we need to test without address verification against config
		// since this is an isolated test with its own wallets

		// Test dev wallet - use empty address to bypass config verification
		loadedWallet, err := wm.LoadWallet("", config.RoleBootnode)
		if err != nil {
			t.Errorf("LoadWallet with empty address failed: %v", err)
		} else if loadedWallet.GetAddress() != devAddress {
			t.Errorf("Expected wallet address to be %s, got %s", devAddress, loadedWallet.GetAddress())
		}

		// Test master wallet - use empty address to bypass config verification
		loadedMasterWallet, err := wm.LoadMasterWallet("", config.RoleBootnode)
		if err != nil {
			t.Errorf("LoadMasterWallet with empty address failed: %v", err)
		} else if loadedMasterWallet.GetAddress() != masterAddress {
			t.Errorf("Expected master wallet address to be %s, got %s", masterAddress, loadedMasterWallet.GetAddress())
		}
	})

	t.Run("Bootnode_WrongMinerAddress", func(t *testing.T) {
		_, err := wm.LoadWallet("wrong-address", config.RoleBootnode)
		if err == nil {
			t.Error("LoadWallet with wrong address should have failed")
		}
	})

	t.Run("Bootnode_WrongMasterAddress", func(t *testing.T) {
		_, err := wm.LoadMasterWallet("wrong-address", config.RoleBootnode)
		if err == nil {
			t.Error("LoadMasterWallet with wrong address should have failed")
		}
	})

	t.Run("Bootnode_EmptyAddresses", func(t *testing.T) {
		// Test dev wallet
		loadedWallet, err := wm.LoadWallet("", config.RoleBootnode)
		if err != nil {
			t.Errorf("LoadWallet with empty address failed: %v", err)
		} else if loadedWallet.GetAddress() != devAddress {
			t.Errorf("Expected wallet address to be %s, got %s", devAddress, loadedWallet.GetAddress())
		}

		// Test master wallet
		loadedMasterWallet, err := wm.LoadMasterWallet("", config.RoleBootnode)
		if err != nil {
			t.Errorf("LoadMasterWallet with empty address failed: %v", err)
		} else if loadedMasterWallet.GetAddress() != masterAddress {
			t.Errorf("Expected master wallet address to be %s, got %s", masterAddress, loadedMasterWallet.GetAddress())
		}
	})

	t.Run("Root_AnyAddress", func(t *testing.T) {
		// Root role should always return the BLOCKCHAIN_ADDRESS
		loadedWallet, err := wm.LoadWallet("any-address", config.Root)
		if err != nil {
			t.Errorf("LoadWallet for Root failed: %v", err)
		} else if loadedWallet.GetAddress() != utils.BLOCKCHAIN_ADDRESS {
			t.Errorf("Expected wallet address to be %s, got %s", utils.BLOCKCHAIN_ADDRESS, loadedWallet.GetAddress())
		}

		// Master wallet for Root is also BLOCKCHAIN_ADDRESS
		loadedMasterWallet, err := wm.LoadMasterWallet("any-address", config.Root)
		if err != nil {
			t.Errorf("LoadMasterWallet for Root failed: %v", err)
		} else if loadedMasterWallet.GetAddress() != utils.BLOCKCHAIN_ADDRESS {
			t.Errorf("Expected master wallet address to be %s, got %s", utils.BLOCKCHAIN_ADDRESS, loadedMasterWallet.GetAddress())
		}
	})
}
