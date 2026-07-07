package oracle

import (
	"fmt"
	"math/big"
	"os"
	"strconv"
	"strings"
	"time"
)

// LoadConfigFromEnv loads Oracle configuration from environment variables
// Falls back to defaults if variables are not set
func LoadConfigFromEnv() (*OracleConfig, error) {
	config := DefaultOracleConfig()

	// Core Oracle Settings
	if chainID := os.Getenv("ORACLE_CHAIN_ID"); chainID != "" {
		config.ChainID = chainID
	}

	if networkID := os.Getenv("ORACLE_NETWORK_ID"); networkID != "" {
		config.NetworkID = networkID
	}

	if blockTimeStr := os.Getenv("ORACLE_BLOCK_TIME"); blockTimeStr != "" {
		blockTime, err := strconv.Atoi(blockTimeStr)
		if err != nil {
			return nil, fmt.Errorf("invalid ORACLE_BLOCK_TIME: %w", err)
		}
		config.BlockTime = time.Duration(blockTime) * time.Second
	}

	// Token Configuration
	if initialSupplyStr := os.Getenv("ORACLE_INITIAL_SUPPLY"); initialSupplyStr != "" {
		initialSupply, ok := new(big.Int).SetString(initialSupplyStr, 10)
		if !ok {
			return nil, fmt.Errorf("invalid ORACLE_INITIAL_SUPPLY: %s", initialSupplyStr)
		}
		config.InitialSupply = initialSupply
	}

	if maxSupplyStr := os.Getenv("ORACLE_MAX_SUPPLY"); maxSupplyStr != "" {
		maxSupply, ok := new(big.Int).SetString(maxSupplyStr, 10)
		if !ok {
			return nil, fmt.Errorf("invalid ORACLE_MAX_SUPPLY: %s", maxSupplyStr)
		}
		config.MaxSupply = maxSupply
	}

	if ownerKey := os.Getenv("ORACLE_OWNER_KEY"); ownerKey != "" {
		config.OwnerPrivateKey = ownerKey
	}

	if contractAddr := os.Getenv("ORACLE_NRN_CONTRACT"); contractAddr != "" {
		config.ContractAddress = contractAddr
	}

	if xionRPC := os.Getenv("ORACLE_XION_RPC"); xionRPC != "" {
		config.XionRPC = xionRPC
	}

	// P2P Configuration
	if p2pAddr := os.Getenv("ORACLE_P2P_ADDR"); p2pAddr != "" {
		config.P2PListenAddr = p2pAddr
	}

	if bootstrapPeers := os.Getenv("ORACLE_BOOTSTRAP_PEERS"); bootstrapPeers != "" {
		config.BootstrapPeers = strings.Split(bootstrapPeers, ",")
	}

	if dhtEnabledStr := os.Getenv("ORACLE_DHT_ENABLED"); dhtEnabledStr != "" {
		dhtEnabled, err := strconv.ParseBool(dhtEnabledStr)
		if err != nil {
			return nil, fmt.Errorf("invalid ORACLE_DHT_ENABLED: %w", err)
		}
		config.DHTEnabled = dhtEnabled
	}

	if gossipEnabledStr := os.Getenv("ORACLE_GOSSIPSUB_ENABLED"); gossipEnabledStr != "" {
		gossipEnabled, err := strconv.ParseBool(gossipEnabledStr)
		if err != nil {
			return nil, fmt.Errorf("invalid ORACLE_GOSSIPSUB_ENABLED: %w", err)
		}
		config.GossipSubEnabled = gossipEnabled
	}

	// RPC & API
	if rpcAddr := os.Getenv("ORACLE_RPC_ADDR"); rpcAddr != "" {
		config.RPCAddr = rpcAddr
	}

	if apiAddr := os.Getenv("ORACLE_API_ADDR"); apiAddr != "" {
		config.APIAddr = apiAddr
	}

	if walletServerEnabledStr := os.Getenv("ORACLE_WALLET_SERVER_ENABLED"); walletServerEnabledStr != "" {
		walletServerEnabled, err := strconv.ParseBool(walletServerEnabledStr)
		if err != nil {
			return nil, fmt.Errorf("invalid ORACLE_WALLET_SERVER_ENABLED: %w", err)
		}
		config.WalletServerEnabled = walletServerEnabled
	}

	if walletInitialFundStr := os.Getenv("ORACLE_WALLET_INITIAL_FUNDING"); walletInitialFundStr != "" {
		walletInitialFund, ok := new(big.Int).SetString(walletInitialFundStr, 10)
		if !ok {
			return nil, fmt.Errorf("invalid ORACLE_WALLET_INITIAL_FUNDING: %s", walletInitialFundStr)
		}
		config.WalletInitialFund = walletInitialFund
	}

	// IBC Configuration
	if ibcEnabledStr := os.Getenv("ORACLE_IBC_ENABLED"); ibcEnabledStr != "" {
		ibcEnabled, err := strconv.ParseBool(ibcEnabledStr)
		if err != nil {
			return nil, fmt.Errorf("invalid ORACLE_IBC_ENABLED: %w", err)
		}
		config.IBCEnabled = ibcEnabled
	}

	// Storage
	if dataDir := os.Getenv("ORACLE_DATA_DIR"); dataDir != "" {
		config.DataDir = dataDir
	}

	if dbBackend := os.Getenv("ORACLE_DB_BACKEND"); dbBackend != "" {
		config.DBBackend = dbBackend
	}

	// Validator Mode
	if validatorModeStr := os.Getenv("ORACLE_VALIDATOR_MODE"); validatorModeStr != "" {
		validatorMode, err := strconv.ParseBool(validatorModeStr)
		if err != nil {
			return nil, fmt.Errorf("invalid ORACLE_VALIDATOR_MODE: %w", err)
		}
		config.ValidatorMode = validatorMode
	}

	// Payment processor (fiat-triggered NRN disbursement)
	if v := os.Getenv("ORACLE_PAYMENT_PROCESSOR_ENABLED"); v != "" {
		enabled, err := strconv.ParseBool(v)
		if err != nil {
			return nil, fmt.Errorf("invalid ORACLE_PAYMENT_PROCESSOR_ENABLED: %w", err)
		}
		config.PaymentProcessorEnabled = enabled
	}
	if v := os.Getenv("STRIPE_SECRET_KEY"); v != "" {
		config.StripeSecretKey = v
	}
	if v := os.Getenv("STRIPE_WEBHOOK_SECRET"); v != "" {
		config.StripeWebhookSecret = v
	}
	if v := os.Getenv("COINBASE_API_KEY"); v != "" {
		config.CoinbaseAPIKey = v
	}
	if v := os.Getenv("COINBASE_WEBHOOK_SECRET"); v != "" {
		config.CoinbaseWebhookSecret = v
	}
	if v := os.Getenv("ORACLE_PAYMENT_TOKEN_DECIMALS"); v != "" {
		decimals, err := strconv.Atoi(v)
		if err != nil {
			return nil, fmt.Errorf("invalid ORACLE_PAYMENT_TOKEN_DECIMALS: %w", err)
		}
		config.PaymentTokenDecimals = decimals
	}
	if v := os.Getenv("ORACLE_PAYMENT_USD_PER_TOKEN"); v != "" {
		f, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid ORACLE_PAYMENT_USD_PER_TOKEN: %w", err)
		}
		config.PaymentUSDPerToken = f
	}
	if v := os.Getenv("ORACLE_PAYMENT_ETH_PER_TOKEN"); v != "" {
		f, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid ORACLE_PAYMENT_ETH_PER_TOKEN: %w", err)
		}
		config.PaymentETHPerToken = f
	}

	return config, nil
}

// ValidateConfig validates the Oracle configuration
func ValidateConfig(config *OracleConfig) error {
	if config.ChainID == "" {
		return fmt.Errorf("chain_id is required")
	}

	if config.NetworkID == "" {
		return fmt.Errorf("network_id is required")
	}

	if config.BlockTime <= 0 {
		return fmt.Errorf("block_time must be positive")
	}

	if config.InitialSupply == nil || config.InitialSupply.Cmp(big.NewInt(0)) <= 0 {
		return fmt.Errorf("initial_supply must be positive")
	}

	if config.MaxSupply == nil || config.MaxSupply.Cmp(big.NewInt(0)) <= 0 {
		return fmt.Errorf("max_supply must be positive")
	}

	if config.InitialSupply.Cmp(config.MaxSupply) > 0 {
		return fmt.Errorf("initial_supply cannot exceed max_supply")
	}

	if config.P2PListenAddr == "" {
		return fmt.Errorf("p2p_listen_addr is required")
	}

	if config.DataDir == "" {
		return fmt.Errorf("data_dir is required")
	}

	if config.DBBackend != "badger" && config.DBBackend != "memory" {
		return fmt.Errorf("db_backend must be 'badger' or 'memory'")
	}

	return nil
}

// ConfigSummary returns a human-readable summary of the configuration
// (without sensitive data like private keys)
func ConfigSummary(config *OracleConfig) string {
	walletInitialFund := "0"
	if config.WalletInitialFund != nil {
		walletInitialFund = config.WalletInitialFund.String()
	}

	return fmt.Sprintf(`Oracle Configuration:
  Chain ID: %s
  Network ID: %s
  Block Time: %v
  Initial Supply: %s NRN
  Max Supply: %s NRN
  P2P Listen: %s
  Bootstrap Peers: %d
  DHT Enabled: %t
  GossipSub Enabled: %t
  IBC Enabled: %t
  Data Directory: %s
  DB Backend: %s
  Validator Mode: %t
  RPC Address: %s
  API Address: %s
  Wallet Server Enabled: %t
  Wallet Initial Funding: %s`,
		config.ChainID,
		config.NetworkID,
		config.BlockTime,
		config.InitialSupply.String(),
		config.MaxSupply.String(),
		config.P2PListenAddr,
		len(config.BootstrapPeers),
		config.DHTEnabled,
		config.GossipSubEnabled,
		config.IBCEnabled,
		config.DataDir,
		config.DBBackend,
		config.ValidatorMode,
		config.RPCAddr,
		config.APIAddr,
		config.WalletServerEnabled,
		walletInitialFund,
	)
}

// HasRootKey reports whether the oracle owner private key is present.
// The oracle only runs when a root key is configured.
func HasRootKey(config *OracleConfig) bool {
	return config.OwnerPrivateKey != ""
}
