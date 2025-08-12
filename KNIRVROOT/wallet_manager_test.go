package main

import (
	"KNIRVROOT/config"

	"os"
	"path/filepath"
	"testing"
)

// testPathProvider implements PathProvider for testing
type testPathProvider struct {
	walletPath       string
	masterWalletPath string
}

func (p *testPathProvider) GetWalletPath(role ...config.Role) (string, error) {
	return p.walletPath, nil
}

func (p *testPathProvider) GetMasterWalletPath(role ...config.Role) (string, error) {
	return p.masterWalletPath, nil
}

func TestWalletManager_SaveLoadMasterWallet(t *testing.T) {
	// Create temp dir for test isolation
	tempDir, err := os.MkdirTemp("", "wallet-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Setup test address
	testAddress := "KNIRVROOT-test-address"

	// Initialize wallet manager with test paths
	encryptionKey := make([]byte, 32) // AES-256 key
	wm, err := NewWalletManager(encryptionKey, &testPathProvider{
		walletPath:       filepath.Join(tempDir, "wallet.dat"),
		masterWalletPath: filepath.Join(tempDir, "master_wallet.dat"),
	})
	if err != nil {
		t.Fatalf("Failed to create wallet manager: %v", err)
	}

	// Create test wallet
	testWallet := &Wallet{
		Address: testAddress,
	}

	// Test SaveMasterWallet
	err = wm.SaveMasterWallet(testWallet, config.RoleBootnode)
	if err != nil {
		t.Fatalf("SaveMasterWallet failed: %v", err)
	}

	// Test LoadMasterWallet with correct address
	// For bootnode role, we need to save the regular wallet too since LoadMasterWallet checks for it
	err = wm.SaveWallet(testWallet, config.RoleBootnode)
	if err != nil {
		t.Fatalf("SaveWallet failed: %v", err)
	}

	loadedWallet, err := wm.LoadMasterWallet(testAddress, config.RoleBootnode)
	if err != nil {
		t.Fatalf("LoadMasterWallet failed with correct address: %v", err)
	}
	if loadedWallet.GetAddress() != testAddress {
		t.Errorf("Address mismatch: expected %s, got %s", testAddress, loadedWallet.GetAddress())
	}

	// Test LoadMasterWallet with empty address (should load any wallet)
	loadedWallet, err = wm.LoadMasterWallet("", config.RoleBootnode)
	if err != nil {
		t.Fatalf("LoadMasterWallet failed with empty address: %v", err)
	}
	if loadedWallet.GetAddress() != testAddress {
		t.Errorf("Address mismatch: expected %s, got %s", testAddress, loadedWallet.GetAddress())
	}
}

func TestWalletManager_WalletConsistencyChecks(t *testing.T) {
	// Create temp dir for test isolation
	tempDir, err := os.MkdirTemp("", "wallet-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Setup test address
	testAddress := "KNIRVROOT-test-address"

	// Initialize wallet manager with test paths
	encryptionKey := make([]byte, 32) // AES-256 key
	wm, err := NewWalletManager(encryptionKey, &testPathProvider{
		walletPath:       filepath.Join(tempDir, "wallet.dat"),
		masterWalletPath: filepath.Join(tempDir, "master_wallet.dat"),
	})
	if err != nil {
		t.Fatalf("Failed to create wallet manager: %v", err)
	}

	// Create test wallet
	testWallet := &Wallet{
		Address: testAddress,
	}

	// Save both wallets for bootnode role
	err = wm.SaveMasterWallet(testWallet, config.RoleBootnode)
	if err != nil {
		t.Fatalf("SaveMasterWallet failed: %v", err)
	}

	err = wm.SaveWallet(testWallet, config.RoleBootnode)
	if err != nil {
		t.Fatalf("SaveWallet failed: %v", err)
	}

	// Test consistency checks
	_, err = wm.LoadMasterWallet(testAddress, config.RoleBootnode)
	if err != nil {
		t.Fatalf("Consistency check failed: %v", err)
	}

	// Test with mismatched address (negative case)
	_, err = wm.LoadMasterWallet("wrong-address", config.RoleBootnode)
	if err == nil {
		t.Error("Expected error for mismatched address but got none")
	}
}
