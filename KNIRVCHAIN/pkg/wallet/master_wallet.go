package wallet

import (
	"encoding/json"
	"errors" // Import errors package
	"fmt"
	"log"
	"os"
	"strings" // Import strings package

	"KNIRVCHAIN/config"
	"KNIRVCHAIN/pkg/utils"

	"github.com/syndtr/goleveldb/leveldb"
)

// Using MASTER_WALLET_KEY from constants.go

// getMasterWallet retrieves the master wallet from the database or creates a new one if it doesn't exist
// Now role-aware to handle different behaviors based on node role
func getMasterWallet(db *leveldb.DB, role config.Role) (*WalletImpl, error) {
	// For Root and Bootnode roles, we can load/create master wallet
	// For other roles, we should not be creating master wallets
	canCreateMasterWallet := role == config.Root || role == config.RoleBootnode

	// First, try to load from file system for non-Root roles
	if role != config.Root {
		wm, err := NewWalletManager(utils.DeriveEncryptionKey())
		if err != nil {
			return nil, fmt.Errorf("failed to create wallet manager: %w", err)
		}
		// Try to load master wallet using the public LoadMasterWallet method
		wallet, loadErr := wm.LoadMasterWallet("", role)
		if loadErr == nil {
			log.Printf("Loaded master wallet from file with address: %s", wallet.GetAddress())
			return wallet, nil
		}
		// Only log if it's not a simple "file not found" error
		if !os.IsNotExist(loadErr) {
			log.Printf("Failed to load master wallet from file: %v", loadErr)
		}
	}

	// Try to get the master wallet from the database
	masterWalletData, err := db.Get([]byte(utils.MASTER_WALLET_KEY), nil)
	if err == nil && len(masterWalletData) > 0 {
		var wallet WalletImpl
		if errUnmarshal := json.Unmarshal(masterWalletData, &wallet); errUnmarshal != nil {
			return nil, fmt.Errorf("failed to deserialize master wallet: %w", errUnmarshal)
		}

		// After unmarshaling, check if the wallet is usable (has private key)
		if wallet.PrivateKey == nil {
			log.Printf("Warning: Master wallet loaded from DB ('%s') is incomplete (missing private key). Address found: '%s'. Will create a new one.", utils.MASTER_WALLET_KEY, wallet.GetAddress())
			// Treat as if not found, so it falls through to creation logic.
			err = errors.New("master wallet incomplete") // This will make the next 'if' fail.
		} else {
			// Wallet is complete and loaded
			log.Printf("Loaded existing master wallet with address: %s from key '%s'", wallet.GetAddress(), utils.MASTER_WALLET_KEY)

			// If we're not a Root role, save the wallet to file system for future use
			if role != config.Root {
				wm, err := NewWalletManager(utils.DeriveEncryptionKey())
				if err != nil {
					log.Printf("Warning: Failed to create wallet manager: %v", err)
				} else if saveErr := wm.SaveMasterWallet(&wallet, role); saveErr != nil {
					log.Printf("Warning: Failed to save master wallet to file: %v", saveErr)
				}
			}

			return &wallet, nil
		}
	}

	// If we can't create a master wallet for this role, return an error
	if !canCreateMasterWallet {
		return nil, fmt.Errorf("master wallet not found and cannot be created for role %s", role.String())
	}

	// If err is not nil (e.g., GetBytes failed, or we set it to "master wallet incomplete")
	// or if masterWalletData is empty, create a new one.
	if err != nil && !strings.Contains(err.Error(), "key not found") && !strings.Contains(err.Error(), "master wallet incomplete") {
		log.Printf("Warning: Error trying to get master wallet from DB ('%s'), proceeding to create new: %v", utils.MASTER_WALLET_KEY, err)
	}

	log.Println("Creating new master wallet for token disbursement...")
	newWallet, createErr := NewWallet()
	if createErr != nil {
		return nil, fmt.Errorf("failed to create new master wallet: %w", createErr)
	}

	walletData, marshalErr := json.Marshal(newWallet)
	if marshalErr != nil {
		return nil, fmt.Errorf("failed to serialize new master wallet: %w", marshalErr)
	}

	if err := db.Put([]byte(utils.MASTER_WALLET_KEY), walletData, nil); err != nil {
		return nil, fmt.Errorf("failed to store master wallet: %w", err)
	}

	log.Printf("Created and stored new master wallet with address: %s under key '%s'", newWallet.GetAddress(), utils.MASTER_WALLET_KEY)

	// Save to file system as well for non-Root roles
	if role != config.Root {
		wm, err := NewWalletManager(utils.DeriveEncryptionKey())
		if err != nil {
			log.Printf("Warning: Failed to create wallet manager: %v", err)
		} else if saveErr := wm.SaveMasterWallet(newWallet, role); saveErr != nil {
			log.Printf("Warning: Failed to save new master wallet to file: %v", saveErr)
		}
	}

	return newWallet, nil
}
