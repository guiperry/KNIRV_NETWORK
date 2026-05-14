package tests

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Mock XION structures for testing
type XionConfig struct {
	ChainID             string `json:"chain_id"`
	RPCEndpoint         string `json:"rpc_endpoint"`
	GasPrice            string `json:"gas_price"`
	NRNTokenAddress     string `json:"nrn_token_address"`
	FaucetAddress       string `json:"faucet_address"`
	GaslessEnabled      bool   `json:"gasless_enabled"`
}

type XionMetaAccount struct {
	Address     string `json:"address"`
	ChainID     string `json:"chain_id"`
	Balance     string `json:"balance"`
	NRNBalance  string `json:"nrn_balance"`
	Gasless     bool   `json:"gasless_enabled"`
	CreatedAt   time.Time `json:"created_at"`
}

type XionTransaction struct {
	From            string                 `json:"from"`
	To              string                 `json:"to"`
	Amount          string                 `json:"amount"`
	Denom           string                 `json:"denom"`
	Memo            string                 `json:"memo"`
	GasLimit        string                 `json:"gas_limit"`
	GasPrice        string                 `json:"gas_price"`
	Gasless         bool                   `json:"gasless"`
	Type            string                 `json:"type"`
	ContractAddress string                 `json:"contract_address,omitempty"`
	SkillID         string                 `json:"skill_id,omitempty"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
}

type XionTransactionResult struct {
	TxHash      string `json:"tx_hash"`
	BlockHeight int64  `json:"block_height"`
	GasUsed     string `json:"gas_used"`
	Success     bool   `json:"success"`
	Error       string `json:"error,omitempty"`
}

// Mock XION Integration Service
type MockXionIntegrationService struct {
	config   XionConfig
	accounts map[string]*XionMetaAccount
	txs      []*XionTransactionResult
}

func NewMockXionIntegrationService() *MockXionIntegrationService {
	return &MockXionIntegrationService{
		config: XionConfig{
			ChainID:             "xion-testnet-1",
			RPCEndpoint:         "https://rpc.xion-testnet-1.burnt.com:443",
			GasPrice:            "0.025uxion",
			NRNTokenAddress:     "xion1nrn_contract_test_address",
			FaucetAddress:       "xion1faucet_contract_test_address",
			GaslessEnabled:      true,
		},
		accounts: make(map[string]*XionMetaAccount),
		txs:      make([]*XionTransactionResult, 0),
	}
}

func (s *MockXionIntegrationService) GetConfig() XionConfig {
	return s.config
}

func (s *MockXionIntegrationService) CreateMetaAccount(address string) (*XionMetaAccount, error) {
	if !strings.HasPrefix(address, "xion1") {
		return nil, assert.AnError
	}

	account := &XionMetaAccount{
		Address:    address,
		ChainID:    s.config.ChainID,
		Balance:    "1000000",
		NRNBalance: "500000",
		Gasless:    s.config.GaslessEnabled,
		CreatedAt:  time.Now(),
	}

	s.accounts[address] = account
	return account, nil
}

func (s *MockXionIntegrationService) GetMetaAccount(address string) (*XionMetaAccount, error) {
	account, exists := s.accounts[address]
	if !exists {
		return nil, assert.AnError
	}
	return account, nil
}

func (s *MockXionIntegrationService) GetBalance(address string, denom string) (string, error) {
	account, exists := s.accounts[address]
	if !exists {
		return "0", nil
	}

	switch denom {
	case "uxion":
		return account.Balance, nil
	case "nrn":
		return account.NRNBalance, nil
	default:
		return "0", nil
	}
}

func (s *MockXionIntegrationService) TransferNRN(from, to, amount string) (*XionTransactionResult, error) {
	if !strings.HasPrefix(from, "xion1") || !strings.HasPrefix(to, "xion1") {
		return nil, assert.AnError
	}

	result := &XionTransactionResult{
		TxHash:      "0x" + strings.Repeat("a", 64),
		BlockHeight: time.Now().Unix(),
		GasUsed:     "0", // Gasless
		Success:     true,
	}

	s.txs = append(s.txs, result)
	return result, nil
}

func (s *MockXionIntegrationService) BurnNRNForSkill(address, skillID, amount string, metadata map[string]interface{}) (*XionTransactionResult, error) {
	if !strings.HasPrefix(address, "xion1") || skillID == "" || amount == "" {
		return nil, assert.AnError
	}

	result := &XionTransactionResult{
		TxHash:      "0x" + strings.Repeat("b", 64),
		BlockHeight: time.Now().Unix(),
		GasUsed:     "0", // Gasless
		Success:     true,
	}

	s.txs = append(s.txs, result)
	return result, nil
}

func (s *MockXionIntegrationService) RequestFromFaucet(address, amount string) (*XionTransactionResult, error) {
	if !strings.HasPrefix(address, "xion1") {
		return nil, assert.AnError
	}

	// Update account balance
	if account, exists := s.accounts[address]; exists {
		// Simple addition for testing
		account.NRNBalance = "1500000" // Mock increased balance
	}

	result := &XionTransactionResult{
		TxHash:      "0x" + strings.Repeat("f", 64),
		BlockHeight: time.Now().Unix(),
		GasUsed:     "0", // Gasless
		Success:     true,
	}

	s.txs = append(s.txs, result)
	return result, nil
}

func (s *MockXionIntegrationService) SendTransaction(tx *XionTransaction) (*XionTransactionResult, error) {
	if tx.From == "" || tx.To == "" || tx.Amount == "" {
		return &XionTransactionResult{
			Success: false,
			Error:   "Invalid transaction parameters",
		}, assert.AnError
	}

	result := &XionTransactionResult{
		TxHash:      "0x" + strings.Repeat("c", 64),
		BlockHeight: time.Now().Unix(),
		GasUsed:     "0", // Gasless
		Success:     true,
	}

	s.txs = append(s.txs, result)
	return result, nil
}

func (s *MockXionIntegrationService) GetTransactionHistory(address string) ([]*XionTransactionResult, error) {
	return s.txs, nil
}

func TestXionIntegrationService(t *testing.T) {
	service := NewMockXionIntegrationService()

	t.Run("Configuration", func(t *testing.T) {
		config := service.GetConfig()

		assert.Equal(t, "xion-testnet-1", config.ChainID)
		assert.Equal(t, "https://rpc.xion-testnet-1.burnt.com:443", config.RPCEndpoint)
		assert.Equal(t, "0.025uxion", config.GasPrice)
		assert.NotEmpty(t, config.NRNTokenAddress)
		assert.NotEmpty(t, config.FaucetAddress)
		assert.True(t, config.GaslessEnabled)
	})

	t.Run("MetaAccountManagement", func(t *testing.T) {
		testAddress := "xion1jg8mtutu9khhfwc4nxmuhcpftf0pajdhfvsqf5"

		t.Run("CreateMetaAccount", func(t *testing.T) {
			account, err := service.CreateMetaAccount(testAddress)

			require.NoError(t, err)
			assert.Equal(t, testAddress, account.Address)
			assert.Equal(t, "xion-testnet-1", account.ChainID)
			assert.Equal(t, "1000000", account.Balance)
			assert.Equal(t, "500000", account.NRNBalance)
			assert.True(t, account.Gasless)
			assert.False(t, account.CreatedAt.IsZero())
		})

		t.Run("GetMetaAccount", func(t *testing.T) {
			// First create an account
			_, err := service.CreateMetaAccount(testAddress)
			require.NoError(t, err)

			// Then retrieve it
			account, err := service.GetMetaAccount(testAddress)

			require.NoError(t, err)
			assert.Equal(t, testAddress, account.Address)
			assert.Equal(t, "1000000", account.Balance)
		})

		t.Run("InvalidAddress", func(t *testing.T) {
			_, err := service.CreateMetaAccount("invalid-address")
			assert.Error(t, err)

			_, err = service.GetMetaAccount("non-existent-address")
			assert.Error(t, err)
		})
	})

	t.Run("BalanceOperations", func(t *testing.T) {
		testAddress := "xion1test1234567890abcdef1234567890abcdef12"

		// Create account first
		_, err := service.CreateMetaAccount(testAddress)
		require.NoError(t, err)

		t.Run("GetXionBalance", func(t *testing.T) {
			balance, err := service.GetBalance(testAddress, "uxion")

			require.NoError(t, err)
			assert.Equal(t, "1000000", balance)
		})

		t.Run("GetNRNBalance", func(t *testing.T) {
			balance, err := service.GetBalance(testAddress, "nrn")

			require.NoError(t, err)
			assert.Equal(t, "500000", balance)
		})

		t.Run("GetUnknownTokenBalance", func(t *testing.T) {
			balance, err := service.GetBalance(testAddress, "unknown")

			require.NoError(t, err)
			assert.Equal(t, "0", balance)
		})

		t.Run("NonExistentAccount", func(t *testing.T) {
			balance, err := service.GetBalance("xion1nonexistent", "uxion")

			require.NoError(t, err)
			assert.Equal(t, "0", balance)
		})
	})

	t.Run("NRNTransferOperations", func(t *testing.T) {
		fromAddress := "xion1from1234567890abcdef1234567890abcdef12"
		toAddress := "xion1to1234567890abcdef1234567890abcdef123"
		amount := "1000000"

		t.Run("SuccessfulTransfer", func(t *testing.T) {
			result, err := service.TransferNRN(fromAddress, toAddress, amount)

			require.NoError(t, err)
			assert.True(t, result.Success)
			assert.NotEmpty(t, result.TxHash)
			assert.Greater(t, result.BlockHeight, int64(0))
			assert.Equal(t, "0", result.GasUsed) // Gasless
		})

		t.Run("InvalidAddresses", func(t *testing.T) {
			_, err := service.TransferNRN("invalid-from", toAddress, amount)
			assert.Error(t, err)

			_, err = service.TransferNRN(fromAddress, "invalid-to", amount)
			assert.Error(t, err)
		})
	})

	t.Run("SkillInvocationOperations", func(t *testing.T) {
		testAddress := "xion1skill1234567890abcdef1234567890abcdef12"
		skillID := "skill-test-001"
		amount := "1000000"
		metadata := map[string]interface{}{
			"input":     "test input",
			"model":     "CodeT5",
			"maxTokens": 100,
		}

		t.Run("SuccessfulSkillInvocation", func(t *testing.T) {
			result, err := service.BurnNRNForSkill(testAddress, skillID, amount, metadata)

			require.NoError(t, err)
			assert.True(t, result.Success)
			assert.NotEmpty(t, result.TxHash)
			assert.Greater(t, result.BlockHeight, int64(0))
			assert.Equal(t, "0", result.GasUsed) // Gasless
		})

		t.Run("InvalidParameters", func(t *testing.T) {
			_, err := service.BurnNRNForSkill("invalid-address", skillID, amount, metadata)
			assert.Error(t, err)

			_, err = service.BurnNRNForSkill(testAddress, "", amount, metadata)
			assert.Error(t, err)

			_, err = service.BurnNRNForSkill(testAddress, skillID, "", metadata)
			assert.Error(t, err)
		})
	})

	t.Run("FaucetOperations", func(t *testing.T) {
		testAddress := "xion1faucet1234567890abcdef1234567890abcdef"
		amount := "1000000"

		// Create account first
		_, err := service.CreateMetaAccount(testAddress)
		require.NoError(t, err)

		t.Run("SuccessfulFaucetRequest", func(t *testing.T) {
			result, err := service.RequestFromFaucet(testAddress, amount)

			require.NoError(t, err)
			assert.True(t, result.Success)
			assert.NotEmpty(t, result.TxHash)
			assert.Greater(t, result.BlockHeight, int64(0))
			assert.Equal(t, "0", result.GasUsed) // Gasless

			// Check that balance was updated
			balance, err := service.GetBalance(testAddress, "nrn")
			require.NoError(t, err)
			assert.Equal(t, "1500000", balance) // Should be increased
		})

		t.Run("InvalidAddress", func(t *testing.T) {
			_, err := service.RequestFromFaucet("invalid-address", amount)
			assert.Error(t, err)
		})
	})

	t.Run("TransactionOperations", func(t *testing.T) {
		t.Run("SendValidTransaction", func(t *testing.T) {
			tx := &XionTransaction{
				From:     "xion1from1234567890abcdef1234567890abcdef12",
				To:       "xion1to1234567890abcdef1234567890abcdef123",
				Amount:   "1000000",
				Denom:    "uxion",
				Memo:     "Test transaction",
				GasLimit: "200000",
				GasPrice: "0.025uxion",
				Gasless:  true,
				Type:     "transfer",
			}

			result, err := service.SendTransaction(tx)

			require.NoError(t, err)
			assert.True(t, result.Success)
			assert.NotEmpty(t, result.TxHash)
		})

		t.Run("SendInvalidTransaction", func(t *testing.T) {
			tx := &XionTransaction{
				From: "xion1from1234567890abcdef1234567890abcdef12",
				// Missing To and Amount
			}

			result, err := service.SendTransaction(tx)

			assert.Error(t, err)
			assert.False(t, result.Success)
			assert.NotEmpty(t, result.Error)
		})
	})

	t.Run("TransactionHistory", func(t *testing.T) {
		testAddress := "xion1history1234567890abcdef1234567890abcde"

		// Perform some transactions first
		_, _ = service.TransferNRN(testAddress, "xion1to123", "1000")
		_, _ = service.BurnNRNForSkill(testAddress, "skill-001", "500", nil)
		_, _ = service.RequestFromFaucet(testAddress, "2000")

		history, err := service.GetTransactionHistory(testAddress)

		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(history), 3) // At least the 3 transactions we made

		// Verify transaction structure
		for _, tx := range history {
			assert.NotEmpty(t, tx.TxHash)
			assert.Greater(t, tx.BlockHeight, int64(0))
			assert.True(t, tx.Success)
		}
	})

	t.Run("AddressValidation", func(t *testing.T) {
		validAddresses := []string{
			"xion1jg8mtutu9khhfwc4nxmuhcpftf0pajdhfvsqf5",
			"xion1234567890abcdef1234567890abcdef12345678",
			"xion1test1234567890abcdef1234567890abcdef12",
		}

		invalidAddresses := []string{
			"invalid-address",
			"cosmos1jg8mtutu9khhfwc4nxmuhcpftf0pajdhfvsqf5",
			"xion",
			"",
		}

		for _, addr := range validAddresses {
			t.Run("Valid_"+addr, func(t *testing.T) {
				account, err := service.CreateMetaAccount(addr)
				assert.NoError(t, err)
				assert.NotNil(t, account)
			})
		}

		for _, addr := range invalidAddresses {
			t.Run("Invalid_"+addr, func(t *testing.T) {
				_, err := service.CreateMetaAccount(addr)
				assert.Error(t, err)
			})
		}
	})
}
