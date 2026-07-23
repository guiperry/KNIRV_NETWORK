package oracle

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/knirvcorp/knirvoracle/internal/oracle/consensus"
	"github.com/knirvcorp/knirvoracle/internal/oracle/crosschain"
	"github.com/knirvcorp/knirvoracle/internal/oracle/crypto"
	"github.com/knirvcorp/knirvoracle/internal/oracle/did"
	"github.com/knirvcorp/knirvoracle/internal/oracle/economics"
	"github.com/knirvcorp/knirvoracle/internal/oracle/governance"
	"github.com/knirvcorp/knirvoracle/internal/oracle/ibc"
	"github.com/knirvcorp/knirvoracle/internal/oracle/p2p"
	"github.com/knirvcorp/knirvoracle/internal/oracle/payment"
	"github.com/knirvcorp/knirvoracle/internal/oracle/registry"
	"github.com/knirvcorp/knirvoracle/internal/oracle/token"
	"github.com/knirvcorp/knirvoracle/internal/oracle/types"

	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	"go.uber.org/zap"
)

// Oracle is the main coordinator for all oracle functionality
// It integrates token, governance, economics, consensus, IBC, cross-chain, and P2P
type Oracle struct {
	// Core components
	nrnToken         *token.NRN
	governanceSystem *governance.GovernanceSystem
	economicsEngine  *economics.EconomicsEngine
	consensusEngine  *consensus.ConsensusEngine
	ibcHandler       *ibc.Handler
	crossChainRouter *crosschain.Router
	p2pManager       *p2p.P2PManager
	bridgeManager    *crosschain.BridgeManager
	paymentProcessor *payment.Processor
	rollupsMu        sync.RWMutex
	rollups          map[string]*types.RollupRecord
	rollupsPath      string

	// Verifier registry (Phase 4): validation-chain nodes that attest
	// checkpoints into finality. Quorum = ≥2/3 of registered verifiers.
	verifiersMu   sync.RWMutex
	verifiers     map[string][]byte
	verifiersPath string

	// Checkpoint pipeline (Phase 2): per-foreign-chain registry, audit MMR,
	// and indexed checkpoint records.
	checkpoint *checkpointState

	// DID resolver
	didResolver *did.Resolver

	// Configuration
	config *OracleConfig
	logger *zap.Logger

	// Lifecycle
	ctx    context.Context
	cancel context.CancelFunc
}

// OracleConfig contains configuration for the oracle
type OracleConfig struct {
	// Chain configuration
	ChainID   string        `json:"chain_id"`
	NetworkID string        `json:"network_id"`
	BlockTime time.Duration `json:"block_time"`

	// Token configuration
	TokenName       string   `json:"token_name"`
	TokenSymbol     string   `json:"token_symbol"`
	InitialSupply   *big.Int `json:"initial_supply"`
	MaxSupply       *big.Int `json:"max_supply"`
	OwnerPrivateKey string   `json:"owner_private_key"`
	ContractAddress string   `json:"contract_address,omitempty"`
	XionRPC         string   `json:"xion_rpc,omitempty"`

	// P2P configuration
	P2PListenAddr    string   `json:"p2p_listen_addr"`
	BootstrapPeers   []string `json:"bootstrap_peers"`
	DHTEnabled       bool     `json:"dht_enabled"`
	GossipSubEnabled bool     `json:"gossipsub_enabled"`

	// Validator configuration
	ValidatorMode bool   `json:"validator_mode"`
	ValidatorKey  string `json:"validator_key,omitempty"`

	// IBC configuration
	IBCEnabled bool `json:"ibc_enabled"`

	// RPC & API
	RPCAddr string `json:"rpc_addr"`
	APIAddr string `json:"api_addr"`

	// Wallet server configuration
	WalletServerEnabled bool     `json:"wallet_server_enabled"`
	WalletInitialFund   *big.Int `json:"wallet_initial_fund"`

	// Payment processor configuration (fiat-triggered NRN disbursement)
	PaymentProcessorEnabled bool    `json:"payment_processor_enabled"`
	StripeSecretKey         string  `json:"stripe_secret_key,omitempty"`
	StripeWebhookSecret     string  `json:"stripe_webhook_secret,omitempty"`
	CoinbaseAPIKey          string  `json:"coinbase_api_key,omitempty"`
	CoinbaseWebhookSecret   string  `json:"coinbase_webhook_secret,omitempty"`
	PaymentTokenDecimals    int     `json:"payment_token_decimals"`
	PaymentUSDPerToken      float64 `json:"payment_usd_per_token"`
	PaymentETHPerToken      float64 `json:"payment_eth_per_token"`

	// Storage configuration
	DataDir   string `json:"data_dir"`
	DBBackend string `json:"db_backend"`
}

// DefaultOracleConfig returns default configuration
func DefaultOracleConfig() *OracleConfig {
	initialSupply, _ := new(big.Int).SetString("1000000000", 10)
	maxSupply, _ := new(big.Int).SetString("10000000000", 10)

	return &OracleConfig{
		ChainID:             "knirvoracle-1",
		NetworkID:           "knirv-testnet",
		BlockTime:           5 * time.Second,
		TokenName:           "KNIRV Network Token",
		TokenSymbol:         "NRN",
		InitialSupply:       initialSupply,
		MaxSupply:           maxSupply,
		OwnerPrivateKey:     "",
		ContractAddress:     "",
		XionRPC:             "https://rpc.xion.testnet",
		P2PListenAddr:       "/ip4/0.0.0.0/tcp/26656",
		BootstrapPeers:      []string{},
		DHTEnabled:          true,
		GossipSubEnabled:    true,
		ValidatorMode:       false,
		IBCEnabled:          true,
		RPCAddr:             "127.0.0.1:26657",
		APIAddr:             "0.0.0.0:8080",
		WalletServerEnabled: true,
		WalletInitialFund:   big.NewInt(1000),

		PaymentProcessorEnabled: false,
		PaymentTokenDecimals:    18,
		PaymentUSDPerToken:      0,
		PaymentETHPerToken:      0,

		DataDir:   "./data/oracle",
		DBBackend: "badger",
	}
}

// NewOracle creates a new Oracle instance
func NewOracle(config *OracleConfig, logger *zap.Logger) (*Oracle, error) {
	ctx, cancel := context.WithCancel(context.Background())

	// Initialize NRN token
	nrnToken, err := initializeNRNToken(config, logger)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to initialize NRN token: %w", err)
	}

	// Initialize governance
	governanceSystem := governance.NewGovernanceSystem(logger)

	// Initialize economics engine
	economicsEngine := economics.NewEconomicsEngine(nrnToken, logger)

	// Initialize consensus engine
	consensusEngine := consensus.NewConsensusEngine(
		config.ChainID,
		config.BlockTime,
		config.ValidatorMode,
		config.DataDir,
		logger,
	)

	// Initialize consensus validators
	var validators []*consensus.ConsensusValidator
	if config.ValidatorMode && config.ValidatorKey != "" {
		kp, err := crypto.PrivateKeyFromHex(config.ValidatorKey)
		if err != nil {
			cancel()
			return nil, fmt.Errorf("failed to parse validator key: %w", err)
		}

		validator := &consensus.ConsensusValidator{
			Address:          kp.Address,
			PubKey:           ethcrypto.FromECDSAPub(kp.PublicKey),
			VotingPower:      big.NewInt(100), // Default voting power
			ProposerPriority: big.NewInt(0),
		}
		validators = append(validators, validator)

		logger.Info("Validator initialized",
			zap.String("address", kp.Address.String()),
			zap.Int64("voting_power", 100),
		)
	}

	// Initialize chain with validators
	if err := consensusEngine.InitChain(validators); err != nil {
		cancel()
		return nil, fmt.Errorf("failed to initialize consensus chain: %w", err)
	}

	// Initialize bridge manager
	bridgeManager := crosschain.NewBridgeManager()
	if err := bridgeManager.RegisterBridge(crosschain.DefaultBridgeConfigs()[0]); err != nil {
		cancel()
		return nil, fmt.Errorf("failed to register default bridges: %w", err)
	}

	// Initialize IBC handler
	ibcHandler := ibc.NewHandler(logger)

	// Initialize cross-chain router
	crossChainRouter := crosschain.NewRouter(ibcHandler, bridgeManager, logger)

	// Initialize P2P manager
	p2pConfig := &p2p.P2PConfig{
		ListenAddr:     config.P2PListenAddr,
		BootstrapPeers: config.BootstrapPeers,
		EnableDHT:      config.DHTEnabled,
		EnableGossip:   config.GossipSubEnabled,
	}
	p2pManager := p2p.NewP2PManager(p2pConfig, logger)

	didResolver := did.NewResolver(did.NewMemoryStore())

	oracle := &Oracle{
		nrnToken:         nrnToken,
		governanceSystem: governanceSystem,
		didResolver:      didResolver,
		economicsEngine:  economicsEngine,
		consensusEngine:  consensusEngine,
		ibcHandler:       ibcHandler,
		crossChainRouter: crossChainRouter,
		p2pManager:       p2pManager,
		bridgeManager:    bridgeManager,
		rollups:          make(map[string]*types.RollupRecord),
		rollupsPath:      filepath.Join(config.DataDir, "rollups.json"),
		verifiers:        make(map[string][]byte),
		verifiersPath:    filepath.Join(config.DataDir, "verifiers.json"),
		config:           config,
		logger:           logger,
		ctx:              ctx,
		cancel:           cancel,
	}

	oracle.paymentProcessor = payment.NewProcessor(payment.Config{
		Enabled:               config.PaymentProcessorEnabled,
		StripeSecretKey:       config.StripeSecretKey,
		StripeWebhookSecret:   config.StripeWebhookSecret,
		CoinbaseAPIKey:        config.CoinbaseAPIKey,
		CoinbaseWebhookSecret: config.CoinbaseWebhookSecret,
		TokenDecimals:         config.PaymentTokenDecimals,
		USDPerToken:           config.PaymentUSDPerToken,
		ETHPerToken:           config.PaymentETHPerToken,
	}, oracle, logger)

	if err := os.MkdirAll(config.DataDir, 0755); err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create oracle data directory: %w", err)
	}
	if err := oracle.loadRollups(); err != nil {
		cancel()
		return nil, fmt.Errorf("failed to load persisted oracle rollups: %w", err)
	}

	cs, err := LoadCheckpointState(config.DataDir)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to load checkpoint state: %w", err)
	}
	oracle.checkpoint = cs
	// Inclusion proofs and consensus AppHash must be views over the exact same
	// append-only object, not independently maintained replicas.
	if appRoot, appSize := oracle.consensusEngine.GetApp().AuditSnapshot(); appSize > 0 && (appSize != cs.mmr.Size() || appRoot != cs.mmr.BagRoot()) {
		cancel()
		return nil, fmt.Errorf("checkpoint MMR and persisted AppHash MMR diverged; refusing unsafe recovery")
	}
	oracle.consensusEngine.SetAuditMMR(cs.mmr)

	if err := oracle.loadVerifiers(); err != nil {
		cancel()
		return nil, fmt.Errorf("failed to load verifier registry: %w", err)
	}

	// Wire consensus-delivered checkpoint/finality transactions to the same
	// admission path as the HTTP endpoints (merkle-math.md §3.3). The ABCI app
	// routes the tx and the Oracle decodes + admits it.
	oracle.consensusEngine.GetApp().SetCheckpointTxHandler(func(txType string, payload []byte) error {
		switch txType {
		case "checkpoint":
			var cp types.Checkpoint
			if err := json.Unmarshal(payload, &cp); err != nil {
				return fmt.Errorf("decode checkpoint tx: %w", err)
			}
			_, err := oracle.SubmitCheckpoint(&cp)
			return err
		case "finality":
			// Phase 4: admit the finality leaf through the same path as HTTP.
			var fin types.FinalityRecord
			if err := json.Unmarshal(payload, &fin); err != nil {
				return fmt.Errorf("decode finality tx: %w", err)
			}
			_, err := oracle.SubmitFinality(&fin)
			return err
		default:
			return fmt.Errorf("unknown checkpoint tx type: %s", txType)
		}
	})

	// Drive the finality window-miss sweeper (the Oracle runs non-validator, so
	// there is no block loop to trigger it).
	go oracle.sweepLoop()

	logger.Info("Oracle initialized",
		zap.String("chain_id", config.ChainID),
		zap.String("network_id", config.NetworkID),
	)

	return oracle, nil
}

// SubmitRollup admits a rollup batch. The submitting chain must be
// registered (POST /oracle/v3/registry/register, same registry checkpoints
// use) and record.Proposer must be one of its registered authors with a
// valid signature quorum over record.Digest() — otherwise this endpoint
// would accept economic data (NRN transaction batches) from anyone.
func (o *Oracle) SubmitRollup(record *types.RollupRecord) error {
	if record == nil {
		return fmt.Errorf("rollup record is required")
	}
	if err := o.verifyRollupSubmission(record); err != nil {
		return fmt.Errorf("unauthorized rollup submission: %w", err)
	}
	o.rollupsMu.Lock()
	defer o.rollupsMu.Unlock()
	o.rollups[record.ID] = record
	if err := o.persistRollupsLocked(); err != nil {
		return err
	}
	// Phase 3 bridge: project the legacy rollup into the checkpoint MMR so it is
	// covered by the same AppHash that anchors signed checkpoints.
	if _, err := o.ProjectRollup(record); err != nil {
		o.logger.Warn("failed to project rollup into checkpoint MMR",
			zap.String("rollup_id", record.ID), zap.Error(err))
	}
	return nil
}

// commitAuditMMR bags the consensus audit MMR into the Oracle AppHash and
// persists the leaf log so the AppHash survives an Oracle restart
// (merkle-math.md Phase 3). The Oracle runs in non-validator mode, so the
// consensus loop never produces blocks; admission must commit the audit leaf
// directly after appending it, otherwise the AppHash is never updated or
// persisted in production.
func (o *Oracle) commitAuditMMR() error {
	if o.consensusEngine == nil {
		return nil
	}
	if _, err := o.consensusEngine.GetApp().Commit(); err != nil {
		return fmt.Errorf("commit audit MMR to AppHash: %w", err)
	}
	return nil
}

// verifyRollupSubmission requires the submitting chain to be registered and
// record.Proposer's signature quorum to be valid over record.Digest() — the
// same registry.VerifySignatureQuorum checkpoint admission already uses.
func (o *Oracle) verifyRollupSubmission(record *types.RollupRecord) error {
	if record.ChainID == "" {
		return fmt.Errorf("chain_id is required")
	}
	reg, ok := o.checkpoint.registry.Get(record.ChainID)
	if !ok {
		return fmt.Errorf("chain %s not registered", record.ChainID)
	}
	if _, ok := o.checkpoint.registry.AuthorWeight(record.ChainID, record.Proposer); !ok {
		return fmt.Errorf("proposer %s is not a registered author", record.Proposer)
	}
	return registry.VerifySignatureQuorum(reg, record.Digest(), record.Signatures)
}

// verifyRollupAction requires a single valid signature from a registered
// author of the rollup's chain, authorizing a finalize/dispute transition.
// Lighter than full submission quorum since this only transitions the
// status of an already-admitted record, not re-authorizing its contents.
func (o *Oracle) verifyRollupAction(record *types.RollupRecord, action string, sig types.AuthorSig) error {
	if _, ok := o.checkpoint.registry.AuthorWeight(record.ChainID, sig.Address); !ok {
		return fmt.Errorf("%s is not a registered author for chain %s", sig.Address, record.ChainID)
	}
	digest := types.RollupActionDigest(record.ID, record.ChainID, action)
	if !types.VerifyAuthorSigDigest(digest, sig) {
		return fmt.Errorf("invalid signature for %s action on rollup %s", action, record.ID)
	}
	return nil
}

func (o *Oracle) GetRollup(id string) (*types.RollupRecord, bool) {
	o.rollupsMu.RLock()
	defer o.rollupsMu.RUnlock()
	record, ok := o.rollups[id]
	return record, ok
}

func (o *Oracle) FinalizeRollup(id string, finalizedAt time.Time, sig types.AuthorSig) (*types.RollupRecord, error) {
	o.rollupsMu.Lock()
	defer o.rollupsMu.Unlock()

	record, ok := o.rollups[id]
	if !ok {
		return nil, fmt.Errorf("rollup not found: %s", id)
	}
	if err := o.verifyRollupAction(record, "finalize", sig); err != nil {
		return nil, fmt.Errorf("unauthorized finalize: %w", err)
	}

	record.Status = types.RollupStatusFinalized
	timestamp := finalizedAt.UTC()
	record.FinalizedAt = &timestamp
	if err := o.persistRollupsLocked(); err != nil {
		return nil, err
	}
	return record, nil
}

func (o *Oracle) DisputeRollup(id string, reason string, disputedAt time.Time, sig types.AuthorSig) (*types.RollupRecord, error) {
	o.rollupsMu.Lock()
	defer o.rollupsMu.Unlock()

	record, ok := o.rollups[id]
	if !ok {
		return nil, fmt.Errorf("rollup not found: %s", id)
	}
	if err := o.verifyRollupAction(record, "dispute", sig); err != nil {
		return nil, fmt.Errorf("unauthorized dispute: %w", err)
	}

	record.Status = types.RollupStatusDisputed
	record.Dispute = reason
	timestamp := disputedAt.UTC()
	record.DisputedAt = &timestamp
	if err := o.persistRollupsLocked(); err != nil {
		return nil, err
	}
	return record, nil
}

func (o *Oracle) loadRollups() error {
	o.rollupsMu.Lock()
	defer o.rollupsMu.Unlock()

	data, err := os.ReadFile(o.rollupsPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to read oracle rollup state: %w", err)
	}

	var records []*types.RollupRecord
	if err := json.Unmarshal(data, &records); err != nil {
		return fmt.Errorf("failed to decode oracle rollup state: %w", err)
	}

	o.rollups = make(map[string]*types.RollupRecord, len(records))
	for _, record := range records {
		if record == nil {
			continue
		}
		o.rollups[record.ID] = record
	}

	return nil
}

func (o *Oracle) persistRollupsLocked() error {
	if o.rollupsPath == "" {
		return nil
	}

	records := make([]*types.RollupRecord, 0, len(o.rollups))
	for _, record := range o.rollups {
		records = append(records, record)
	}

	payload, err := json.MarshalIndent(records, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to encode oracle rollup state: %w", err)
	}

	tempPath := o.rollupsPath + ".tmp"
	if err := os.WriteFile(tempPath, payload, 0644); err != nil {
		return fmt.Errorf("failed to write oracle rollup temp file: %w", err)
	}

	if err := os.Rename(tempPath, o.rollupsPath); err != nil {
		return fmt.Errorf("failed to move oracle rollup state into place: %w", err)
	}

	return nil
}

// Start starts all oracle services
func (o *Oracle) Start() error {
	o.logger.Info("Starting Oracle services")

	// Start economics engine
	if err := o.economicsEngine.Start(); err != nil {
		return fmt.Errorf("failed to start economics engine: %w", err)
	}

	// Start consensus engine
	if err := o.consensusEngine.Start(); err != nil {
		return fmt.Errorf("failed to start consensus engine: %w", err)
	}

	// Start IBC handler
	if o.config.IBCEnabled {
		if err := o.ibcHandler.Start(); err != nil {
			return fmt.Errorf("failed to start IBC handler: %w", err)
		}
	}

	// Start P2P manager
	if err := o.p2pManager.Start(); err != nil {
		return fmt.Errorf("failed to start P2P manager: %w", err)
	}

	o.logger.Info("Oracle services started successfully")

	return nil
}

// Stop stops all oracle services
func (o *Oracle) Stop() error {
	o.logger.Info("Stopping Oracle services")

	// Stop in reverse order
	o.p2pManager.Stop()

	if o.config.IBCEnabled {
		o.ibcHandler.Stop()
	}

	o.consensusEngine.Stop()
	o.economicsEngine.Stop()

	o.cancel()

	o.logger.Info("Oracle services stopped")

	return nil
}

// GetNRNToken returns the NRN token instance
func (o *Oracle) GetNRNToken() *token.NRN {
	return o.nrnToken
}

// GetGovernanceSystem returns the governance system
func (o *Oracle) GetGovernanceSystem() *governance.GovernanceSystem {
	return o.governanceSystem
}

// GetEconomicsEngine returns the economics engine
func (o *Oracle) GetEconomicsEngine() *economics.EconomicsEngine {
	return o.economicsEngine
}

// GetConsensusEngine returns the consensus engine
func (o *Oracle) GetConsensusEngine() *consensus.ConsensusEngine {
	return o.consensusEngine
}

// GetIBCHandler returns the IBC handler
func (o *Oracle) GetIBCHandler() *ibc.Handler {
	return o.ibcHandler
}

// GetCrossChainRouter returns the cross-chain router
func (o *Oracle) GetCrossChainRouter() *crosschain.Router {
	return o.crossChainRouter
}

// GetP2PManager returns the P2P manager
func (o *Oracle) GetP2PManager() *p2p.P2PManager {
	return o.p2pManager
}

// GetBridgeManager returns the bridge manager
func (o *Oracle) GetBridgeManager() *crosschain.BridgeManager {
	return o.bridgeManager
}

// GetDIDResolver returns the DID resolver
func (o *Oracle) GetDIDResolver() *did.Resolver {
	return o.didResolver
}

// GetPaymentProcessor returns the payment processor, which disburses NRN
// through the oracle's own mint path in response to fiat webhook events.
func (o *Oracle) GetPaymentProcessor() *payment.Processor {
	return o.paymentProcessor
}

// GetStatus returns the current status of the oracle
func (o *Oracle) GetStatus() map[string]interface{} {
	return map[string]interface{}{
		"chain_id":   o.config.ChainID,
		"network_id": o.config.NetworkID,
		"token":      o.nrnToken.Info(),
		"wallet_server": map[string]interface{}{
			"enabled":          o.config.WalletServerEnabled,
			"initial_funding":  o.GetWalletInitialFunding().String(),
			"treasury_address": o.nrnToken.Owner().String(),
		},
		"consensus":  o.consensusEngine.GetInfo(),
		"governance": o.governanceSystem.GetValidatorStats(),
		"economics":  o.economicsEngine.GetEconomicSnapshot(),
		"p2p":        o.p2pManager.GetStats(),
		"ibc": map[string]interface{}{
			"enabled": o.config.IBCEnabled,
		},
	}
}

func (o *Oracle) WalletServerEnabled() bool {
	return o.config.WalletServerEnabled
}

func (o *Oracle) GetWalletInitialFunding() *big.Int {
	if o == nil || o.config == nil || o.config.WalletInitialFund == nil {
		return big.NewInt(0)
	}
	return new(big.Int).Set(o.config.WalletInitialFund)
}

func (o *Oracle) FundAddress(addr types.Address, amount *big.Int, reason string) (*token.MintReceipt, error) {
	if err := validateWalletFundingAmount(amount); err != nil {
		return nil, err
	}

	receipt, err := o.nrnToken.Mint(addr, amount)
	if err != nil {
		return nil, err
	}

	o.logger.Info("Funded wallet from oracle treasury",
		zap.String("address", addr.String()),
		zap.String("amount", amount.String()),
		zap.String("reason", reason),
		zap.String("transaction_hash", receipt.TransactionHash),
	)

	return receipt, nil
}

func validateWalletFundingAmount(amount *big.Int) error {
	if amount == nil {
		return fmt.Errorf("funding amount is required")
	}
	if amount.Sign() <= 0 {
		return fmt.Errorf("funding amount must be positive")
	}
	return nil
}

// Helper: initialize NRN token
func initializeNRNToken(config *OracleConfig, logger *zap.Logger) (*token.NRN, error) {
	// Validate supply values
	if config.InitialSupply == nil {
		return nil, fmt.Errorf("initial supply is required")
	}
	if config.MaxSupply == nil {
		return nil, fmt.Errorf("max supply is required")
	}

	// Create NRN token
	nrnToken, err := token.NewNRN(
		config.TokenName,
		config.TokenSymbol,
		config.InitialSupply,
		config.MaxSupply,
		config.OwnerPrivateKey,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create NRN token: %w", err)
	}

	logger.Info("NRN token initialized",
		zap.String("name", config.TokenName),
		zap.String("symbol", config.TokenSymbol),
		zap.String("initial_supply", config.InitialSupply.String()),
		zap.String("max_supply", config.MaxSupply.String()),
	)

	return nrnToken, nil
}
