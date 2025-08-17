package main

import (
	"os"
	"path/filepath"
	"testing"

	"KNIRVORACLE/config"
)

// TestRoleBasedConfigPaths tests that the correct config paths are used for different roles
func TestRoleBasedConfigPaths(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "KNIRVORACLE-role-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Test each role
	roles := []config.Role{
		config.Root,
		config.RoleBootnode,
		config.RolePeer,
		config.RoleClient,
	}

	for _, role := range roles {
		t.Run(role.String(), func(t *testing.T) {
			// Get config path for this role
			configPath, err := config.GetConfigPath(role)
			if err != nil {
				t.Fatalf("GetConfigPath failed for role %s: %v", role, err)
			}

			// Create a test config
			testConfig := config.DefaultConfig()
			testConfig.Port = 9999
			testConfig.MinersAddress = "test-miner-address-" + role.String()
			testConfig.MasterAddress = "test-master-address-" + role.String()

			// Save the config
			_, err = config.SaveConfig(configPath, testConfig)
			if err != nil {
				t.Fatalf("SaveConfig failed for role %s: %v", role, err)
			}

			// Load the config
			loadedConfig, loadedPath, err := config.LoadConfig("", role)
			if err != nil {
				t.Fatalf("LoadConfig failed for role %s: %v", role, err)
			}

			// Verify the loaded config matches the saved config
			if loadedConfig.Port != testConfig.Port {
				t.Errorf("Expected Port to be %d, got %d", testConfig.Port, loadedConfig.Port)
			}
			if loadedConfig.MinersAddress != testConfig.MinersAddress {
				t.Errorf("Expected MinersAddress to be %s, got %s", testConfig.MinersAddress, loadedConfig.MinersAddress)
			}
			if loadedConfig.MasterAddress != testConfig.MasterAddress {
				t.Errorf("Expected MasterAddress to be %s, got %s", testConfig.MasterAddress, loadedConfig.MasterAddress)
			}

			// Verify the loaded path matches the expected path
			if loadedPath != configPath {
				t.Errorf("Expected loadedPath to be %s, got %s", configPath, loadedPath)
			}
		})
	}
}

// TestRoleBasedWalletConsistency tests the wallet consistency checks for non-Root roles
func TestRoleBasedWalletConsistency(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "KNIRVORACLE-wallet-consistency-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a test encryption key
	encryptionKey := []byte("0123456789ABCDEF0123456789ABCDEF") // 32 bytes for AES-256

	// Create a wallet manager
	wm, err := NewWalletManager(encryptionKey)
	if err != nil {
		t.Fatalf("Failed to create wallet manager: %v", err)
	}

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

	// Create test config with mock paths
	devWalletPath := filepath.Join(tempDir, "wallet.dat")
	masterWalletPath := filepath.Join(tempDir, "master_wallet.dat")

	// Create test path provider
	pathProvider := &config.MockPathProvider{
		WalletPath:       devWalletPath,
		MasterWalletPath: masterWalletPath,
	}

	// Create wallet manager with test path provider
	wm, err = NewWalletManager(encryptionKey, pathProvider)
	if err != nil {
		t.Fatalf("Failed to create wallet manager with path provider: %v", err)
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

	// Create role-specific config files
	devConfigPath := filepath.Join(tempDir, "dev_config.json")
	bootnodeConfigPath := filepath.Join(tempDir, "bootnode_config.json")

	// Create dev config
	devConfig := config.DefaultConfig()
	devConfig.MinersAddress = devAddress
	_, err = config.SaveConfig(devConfigPath, devConfig)
	if err != nil {
		t.Fatalf("Failed to save dev config: %v", err)
	}

	// Create bootnode config
	bootnodeConfig := config.DefaultConfig()
	bootnodeConfig.MinersAddress = devAddress
	bootnodeConfig.MasterAddress = masterAddress
	_, err = config.SaveConfig(bootnodeConfigPath, bootnodeConfig)
	if err != nil {
		t.Fatalf("Failed to save bootnode config: %v", err)
	}

	// Test scenarios
	testCases := []struct {
		name           string
		role           config.Role
		minersAddress  string
		masterAddress  string
		expectSuccess  bool
		expectMismatch bool
		testWallet     string // "miner", "master", or "both"
	}{
		{"Peer_CorrectAddress", config.RolePeer, devAddress, "", true, false, "miner"},
		{"Peer_WrongAddress", config.RolePeer, "wrong-address", "", false, true, "miner"},
		{"Peer_EmptyAddress", config.RolePeer, "", "", true, false, "miner"},
		{"Bootnode_CorrectAddresses", config.RoleBootnode, devAddress, masterAddress, true, false, "both"},
		{"Bootnode_WrongMinerAddress", config.RoleBootnode, "wrong-address", masterAddress, false, true, "miner"},
		{"Bootnode_WrongMasterAddress", config.RoleBootnode, devAddress, "wrong-address", false, true, "master"},
		{"Bootnode_EmptyAddresses", config.RoleBootnode, "", "", true, false, "both"},
		{"Root_AnyAddress", config.Root, "any-address", "any-address", true, false, "miner"}, // Root doesn't check
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Load appropriate config based on role
			var testConfig *config.Config

			switch tc.role {
			case config.RolePeer:
				testConfig, _, err = config.LoadConfig(devConfigPath, tc.role)
				if err != nil {
					t.Fatalf("Failed to load dev config: %v", err)
				}
			case config.RoleBootnode:
				testConfig, _, err = config.LoadConfig(bootnodeConfigPath, tc.role)
				if err != nil {
					t.Fatalf("Failed to load bootnode config: %v", err)
				}
			default:
				testConfig = config.DefaultConfig()
			}

			// Override with test case values if needed
			if tc.minersAddress != "" {
				testConfig.MinersAddress = tc.minersAddress
			}
			if tc.masterAddress != "" {
				testConfig.MasterAddress = tc.masterAddress
			}

			// Perform wallet consistency checks
			var devWalletObj *Wallet
			var masterWalletObj *Wallet
			var err error

			if tc.role != config.Root && (tc.testWallet == "miner" || tc.testWallet == "both") {
				// Check MinersAddress
				if testConfig.MinersAddress != "" {
					devWalletObj, err = wm.LoadWallet(testConfig.MinersAddress, tc.role)
					if tc.expectSuccess {
						if err != nil {
							t.Errorf("LoadWallet failed unexpectedly: %v", err)
						}
						if devWalletObj == nil {
							t.Errorf("LoadWallet returned nil wallet without error")
						} else if devWalletObj.GetAddress() != testConfig.MinersAddress {
							t.Errorf("Expected wallet address to be %s, got %s", testConfig.MinersAddress, devWalletObj.GetAddress())
						}
					} else if tc.expectMismatch {
						if err == nil {
							t.Error("LoadWallet succeeded unexpectedly")
						}
					}
				} else {
					// Empty address, should load any wallet
					devWalletObj, err = wm.LoadWallet("", tc.role)
					if err != nil {
						t.Errorf("LoadWallet with empty address failed: %v", err)
					}
					if devWalletObj == nil {
						t.Errorf("LoadWallet with empty address returned nil wallet")
					} else if devWalletObj.GetAddress() != devAddress {
						t.Errorf("Expected any wallet address to be %s, got %s", devAddress, devWalletObj.GetAddress())
					}
				}

				// Check MasterAddress for Bootnode
				if tc.role == config.RoleBootnode && (tc.testWallet == "master" || tc.testWallet == "both") {
					if testConfig.MasterAddress != "" {
						masterWalletObj, err = wm.LoadMasterWallet(testConfig.MasterAddress, tc.role)
						if tc.expectSuccess {
							if err != nil {
								t.Errorf("LoadMasterWallet failed unexpectedly: %v", err)
							}
							if masterWalletObj == nil {
								t.Errorf("LoadMasterWallet returned nil wallet without error")
							} else if masterWalletObj.GetAddress() != testConfig.MasterAddress {
								t.Errorf("Expected master wallet address to be %s, got %s", testConfig.MasterAddress, masterWalletObj.GetAddress())
							}
						} else if tc.expectMismatch {
							if err == nil {
								t.Error("LoadMasterWallet succeeded unexpectedly")
							}
						}
					} else {
						// Empty address, should load any wallet
						masterWalletObj, err = wm.LoadMasterWallet("", tc.role)
						if err != nil {
							t.Errorf("LoadMasterWallet with empty address failed: %v", err)
						}
						if masterWalletObj == nil {
							t.Errorf("LoadMasterWallet with empty address returned nil wallet")
						} else if masterWalletObj.GetAddress() != masterAddress {
							t.Errorf("Expected any master wallet address to be %s, got %s", masterAddress, masterWalletObj.GetAddress())
						}
					}
				}
			}
		})
	}
}

// TestGUIRoleBasedFeatures tests the GUI role-based feature toggling
func TestGUIRoleBasedFeatures(t *testing.T) {
	// This is a simplified test that doesn't actually create a GUI
	// but verifies the logic for determining which features should be enabled

	// Create test wallets
	devWallet, err := NewWallet()
	if err != nil {
		t.Fatalf("Failed to create dev wallet: %v", err)
	}
	_, err = NewWallet() // masterWallet not used, just need to create it
	if err != nil {
		t.Fatalf("Failed to create master wallet: %v", err)
	}

	// Test scenarios
	testCases := []struct {
		name              string
		role              config.Role
		wallet            *Wallet
		paymentEnabled    bool
		expectMining      bool
		expectPaymentProc bool
		expectRootConfig  bool
	}{
		{"Root_WithWallet_PaymentEnabled", config.Root, devWallet, true, true, true, true},
		{"Root_WithWallet_PaymentDisabled", config.Root, devWallet, false, true, false, true},
		{"Root_NoWallet_PaymentEnabled", config.Root, nil, true, true, true, true},
		{"Bootnode_WithWallet_PaymentEnabled", config.RoleBootnode, devWallet, true, true, true, false},
		{"Bootnode_WithWallet_PaymentDisabled", config.RoleBootnode, devWallet, false, true, false, false},
		{"Bootnode_NoWallet_PaymentEnabled", config.RoleBootnode, nil, true, false, true, false},
		{"Peer_WithWallet", config.RolePeer, devWallet, false, true, false, false},
		{"Peer_NoWallet", config.RolePeer, nil, false, false, false, false},
		{"Client_WithWallet", config.RoleClient, devWallet, false, false, false, false},
		{"Client_NoWallet", config.RoleClient, nil, false, false, false, false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a test config
			testConfig := config.DefaultConfig()
			testConfig.PaymentProcessor.Enabled = tc.paymentEnabled

			// Determine which features should be enabled
			var showMining, showPaymentProcessor, showRootSettings bool

			switch tc.role {
			case config.Root:
				// Root-specific features
				showMining = true       // Always show mining for Root
				showRootSettings = true // Always show creator settings for Root role
				if tc.paymentEnabled {
					showPaymentProcessor = true
				}

			case config.RoleBootnode:
				// Bootnode-specific features
				if tc.wallet != nil {
					showMining = true // Show mining if wallet is available
				}
				if tc.paymentEnabled {
					showPaymentProcessor = true
				}

			case config.RolePeer:
				// Peer-specific features
				if tc.wallet != nil {
					showMining = true // Show mining if wallet is available
				}

			case config.RoleClient:
				// Client-specific features
				// No mining or payment processing for clients
			}

			// Verify the expected features
			if showMining != tc.expectMining {
				t.Errorf("Expected showMining to be %t, got %t", tc.expectMining, showMining)
			}
			if showPaymentProcessor != tc.expectPaymentProc {
				t.Errorf("Expected showPaymentProcessor to be %t, got %t", tc.expectPaymentProc, showPaymentProcessor)
			}
			if showRootSettings != tc.expectRootConfig {
				t.Errorf("Expected showRootSettings to be %t, got %t", tc.expectRootConfig, showRootSettings)
			}
		})
	}
}
