package oracle

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/knirvcorp/knirvoracle/internal/oracle/consensus"
	"github.com/knirvcorp/knirvoracle/internal/oracle/crosschain"
	"github.com/knirvcorp/knirvoracle/internal/oracle/did"
	"github.com/knirvcorp/knirvoracle/internal/oracle/economics"
	"github.com/knirvcorp/knirvoracle/internal/oracle/governance"
	"github.com/knirvcorp/knirvoracle/internal/oracle/ibc"
	"github.com/knirvcorp/knirvoracle/internal/oracle/ledger"
	"github.com/knirvcorp/knirvoracle/internal/oracle/mmr"
	"github.com/knirvcorp/knirvoracle/internal/oracle/p2p"
	"github.com/knirvcorp/knirvoracle/internal/oracle/payment"
	"github.com/knirvcorp/knirvoracle/internal/oracle/registry"
	"github.com/knirvcorp/knirvoracle/internal/oracle/replica"
	"github.com/knirvcorp/knirvoracle/internal/oracle/token"
	"github.com/knirvcorp/knirvoracle/internal/oracle/types"

	knirvsigning "github.com/guiperry/knirv-sdk-go/signing"
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

	// Durable state ledger: the token read model is hydrated from
	// it and every token mutation appends before the in-memory change.
	ledgerMu       sync.Mutex
	ledger         *ledger.Log
	state          *ledger.State
	stateRoot      string
	ledgerDegraded bool

	// Replication source: head production and the ranges standbys pull.
	replication *ReplicationSource
	follower    *replica.Follower
	followersMu sync.RWMutex
	followers   []replica.Source

	// Verifier registry: validation-chain nodes that attest
	// checkpoints into finality. Quorum = ≥2/3 of registered verifiers.
	verifiersMu   sync.RWMutex
	verifiers     map[string][]byte
	verifiersPath string

	// Skill ownership registry: durable perpetual invocation-fee
	// entitlements registered via POST /oracle/v3/skills/ownership (see
	// types.SkillOwnershipRecord), keyed by skill ID.
	skillOwnershipMu   sync.RWMutex
	skillOwnership     map[string]*types.SkillOwnershipRecord
	skillOwnershipPath string

	// Checkpoint pipeline: per-foreign-chain registry, audit MMR,
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
	// supervisorDone is closed when the bond supervisor goroutine exits, so a
	// clean shutdown (and tests) can wait for it to actually terminate after
	// Stop cancels the lifecycle context.
	supervisorDone chan struct{}
}

// Oracle activation modes. They mirror noderole.OracleMode; the role resolver
// is the only place that chooses one, and it arrives here as ORACLE_MODE.
const (
	ModeActive  = "active"
	ModeStandby = "standby"
	ModeOff     = "off"
)

// NormalizeMode returns a valid mode, defaulting to active when unset.
func NormalizeMode(mode string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "", ModeActive:
		return ModeActive, nil
	case ModeStandby:
		return ModeStandby, nil
	case ModeOff:
		return ModeOff, nil
	}
	return "", fmt.Errorf("invalid oracle mode %q (want active, standby or off)", mode)
}

// OracleConfig contains configuration for the oracle.
type OracleConfig struct {
	// Chain configuration
	ChainID   string        `json:"chain_id"`
	NetworkID string        `json:"network_id"`
	BlockTime time.Duration `json:"block_time"`

	// Mode is "active", "standby" or "off", decided by noderole.Resolve in
	// KNIRVSERVER and passed as ORACLE_MODE. A standby answers read and
	// replication routes only. Empty means active.
	Mode string `json:"mode"`

	// Token configuration
	TokenName       string   `json:"token_name"`
	TokenSymbol     string   `json:"token_symbol"`
	InitialSupply   *big.Int `json:"initial_supply"`
	MaxSupply       *big.Int `json:"max_supply"`
	OwnerPrivateKey string   `json:"owner_private_key"`
	// ReplicationSigner is an already-unlocked root-owner handle supplied by
	// the host; key material is never serialized in this field.
	ReplicationSigner knirvsigning.RemoteSigner `json:"-"`
	ContractAddress   string                    `json:"contract_address,omitempty"`
	XionRPC           string                    `json:"xion_rpc,omitempty"`

	// P2P configuration
	P2PListenAddr    string   `json:"p2p_listen_addr"`
	BootstrapPeers   []string `json:"bootstrap_peers"`
	DHTEnabled       bool     `json:"dht_enabled"`
	GossipSubEnabled bool     `json:"gossipsub_enabled"`

	// Validator configuration
	ValidatorMode bool `json:"validator_mode"`
	// ValidatorSigner is an already-unlocked signing identity supplied by the
	// host (resolved from secrets.Provider per the role resolver, Task 8.8).
	// Key material is never serialized in this field or read from config/env;
	// ORACLE_VALIDATOR_KEY has no runtime reader.
	ValidatorSigner knirvsigning.RemoteSigner `json:"-"`

	// IBC configuration
	IBCEnabled bool `json:"ibc_enabled"`

	// RPC & API
	RPCAddr string `json:"rpc_addr"`
	APIAddr string `json:"api_addr"`

	// Wallet server configuration
	WalletServerEnabled bool     `json:"wallet_server_enabled"`
	WalletInitialFund   *big.Int `json:"wallet_initial_fund"`

	// Payment processor configuration (fiat-triggered NRN disbursement)
	PaymentProcessorEnabled bool   `json:"payment_processor_enabled"`
	StripeSecretKey         string `json:"stripe_secret_key,omitempty"`
	StripeWebhookSecret     string `json:"stripe_webhook_secret,omitempty"`
	// StripeThinWebhookSecret is the signing secret for a second Stripe Event
	// Destination delivering "thin" payloads — see payment.Config's doc
	// comment on the same field for why a second secret is needed at all.
	StripeThinWebhookSecret string  `json:"stripe_thin_webhook_secret,omitempty"`
	CoinbaseAPIKey          string  `json:"coinbase_api_key,omitempty"`
	CoinbaseWebhookSecret   string  `json:"coinbase_webhook_secret,omitempty"`
	PaymentTokenDecimals    int     `json:"payment_token_decimals"`
	PaymentUSDPerToken      float64 `json:"payment_usd_per_token"`
	PaymentETHPerToken      float64 `json:"payment_eth_per_token"`

	// Subscription-plan checkout (KNIRV.COM Pro plan). See payment.Config for
	// how these are used — this is a distinct flow from the fiat->NRN
	// disbursement fields above.
	StripeProfessionalPriceID    string `json:"stripe_professional_price_id,omitempty"`
	StripeEnterprisePriceID      string `json:"stripe_enterprise_price_id,omitempty"`
	StripeInvestorPriceID        string `json:"stripe_investor_price_id,omitempty"`
	PaymentSuccessURL            string `json:"payment_success_url,omitempty"`
	PaymentCancelURL             string `json:"payment_cancel_url,omitempty"`
	PaymentOnboardingCallbackURL string `json:"payment_onboarding_callback_url,omitempty"`

	// Storage configuration
	DataDir   string `json:"data_dir"`
	DBBackend string `json:"db_backend"`

	// ReplicationSources are ordered gateway/direct HTTP sources for a standby.
	// A source is consumed only with a verified root signing address. The
	// authenticated HTTP adapter is installed by the launcher once it has a
	// bootnode signer (Task 8.8); an empty adapter list therefore pauses safely.
	ReplicationSources     []string      `json:"replication_sources,omitempty"`
	ReplicationTrustAnchor string        `json:"replication_trust_anchor,omitempty"`
	ReplicationInterval    time.Duration `json:"replication_interval,omitempty"`

	// ChainVerifyURL is KNIRVCHAIN's HTTP base URL, used to verify a
	// SYNDICATE_* commitment's inclusion proof (GET /proof/tx/{hash}) before
	// authorizing an actuarial-syndicate settlement payout — see
	// internal/oracle/actuarial. Disbursement never proceeds without a
	// verified proof from this endpoint.
	ChainVerifyURL string `json:"chain_verify_url,omitempty"`
}

// DefaultOracleConfig returns default configuration
func DefaultOracleConfig() *OracleConfig {
	initialSupply, _ := new(big.Int).SetString("1000000000", 10)
	maxSupply, _ := new(big.Int).SetString("10000000000", 10)

	return &OracleConfig{
		ChainID:   "knirvoracle-1",
		NetworkID: "knirv-testnet",
		Mode:      ModeActive,
		// BlockTime is a heartbeat, not a security clock — the checkpoint
		// dispute window is wall-clock based (registry.DefaultProofWindow),
		// not tied to this cadence at all (see consensus/engine.go's
		// consensusLoop). 0 disables the periodic ticker entirely, relying
		// solely on real checkpoint/finality/rollup submissions to advance
		// the AppHash (see Oracle.commitAuditMMR). LoadConfigFromEnv derives
		// this from network mode when ORACLE_BLOCK_TIME is not set explicitly
		// (30 minutes on testnet, disabled on production); this default of 30
		// minutes only applies when OracleConfig is constructed directly
		// (e.g. in tests), bypassing that env-driven resolution.
		BlockTime:       30 * time.Minute,
		TokenName:       "KNIRV Network Token",
		TokenSymbol:     "NRN",
		InitialSupply:   initialSupply,
		MaxSupply:       maxSupply,
		OwnerPrivateKey: "",
		ContractAddress: "",
		// xiond is supervised by KNIRVSERVER and exposed through KNIRVGATEWAY.
		XionRPC:        "http://127.0.0.1:8080/xion/rpc",
		P2PListenAddr:  "/ip4/0.0.0.0/tcp/26656",
		BootstrapPeers: []string{},
		// KNIRVGATEWAY is the sole libp2p/DHT/GossipSub owner.
		DHTEnabled:          false,
		GossipSubEnabled:    false,
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

		DataDir:             "./data/oracle",
		DBBackend:           "badger",
		ReplicationInterval: 30 * time.Second,
	}
}

// buildConsensusValidators derives the consensus validator set from the
// host-supplied RemoteSigner (Task 8.8). It returns nil (no validators) when the
// Oracle is not in validator mode, and an error when validator mode is on but no
// signer is available. The validator identity (address + public key) comes
// entirely from the signer; no raw key is ever read from config or env.
func buildConsensusValidators(config *OracleConfig) ([]*consensus.ConsensusValidator, error) {
	if !config.ValidatorMode {
		return nil, nil
	}
	if config.ValidatorSigner == nil {
		return nil, fmt.Errorf("validator mode enabled but no validator signer supplied")
	}
	address, err := types.AddressFromString(config.ValidatorSigner.Address())
	if err != nil {
		return nil, fmt.Errorf("validator signer address %q is invalid: %w", config.ValidatorSigner.Address(), err)
	}
	validator := &consensus.ConsensusValidator{
		Address:          address,
		PubKey:           config.ValidatorSigner.PublicKey(),
		VotingPower:      big.NewInt(100),
		ProposerPriority: big.NewInt(0),
	}
	return []*consensus.ConsensusValidator{validator}, nil
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
	if err := governanceSystem.SetChainID(config.ChainID); err != nil {
		cancel()
		return nil, err
	}
	// Wire the §6.6 validator-set-sync hook. KNIRVORACLE only ever runs with
	// root.key's own key already loaded as OwnerPrivateKey (see
	// cmd/oracle/rootkey.go), which is exactly the identity §6.6's genesis
	// trust model expects to publish the first (and, until a takeover
	// confirms, every subsequent) validator set.
	if config.OwnerPrivateKey != "" {
		registrySync := governance.NewRegistrySync(config.ChainID, logger)
		if err := registrySync.SetSigningIdentity(config.OwnerPrivateKey); err != nil {
			logger.Warn("registry validator-set sync disabled: invalid signing identity", zap.Error(err))
		} else {
			governanceSystem.SetRegistrySync(registrySync)
		}
	}

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

	// Initialize consensus validators from the host-supplied RemoteSigner
	// (Task 8.8): the validator identity comes from OpenBao / role resolution,
	// never from a hex key in config or env. A sealed or missing signer yields a
	// loud error (Degraded) so the Oracle fails to start rather than silently
	// running without a validator.
	validators, err := buildConsensusValidators(config)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to build consensus validators: %w", err)
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
	crossChainRouter := crosschain.NewRouter(ibcHandler, bridgeManager, governanceSystem, logger)

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
		nrnToken:           nrnToken,
		governanceSystem:   governanceSystem,
		didResolver:        didResolver,
		economicsEngine:    economicsEngine,
		consensusEngine:    consensusEngine,
		ibcHandler:         ibcHandler,
		crossChainRouter:   crossChainRouter,
		p2pManager:         p2pManager,
		bridgeManager:      bridgeManager,
		rollups:            make(map[string]*types.RollupRecord),
		rollupsPath:        filepath.Join(config.DataDir, "rollups.json"),
		verifiers:          make(map[string][]byte),
		verifiersPath:      filepath.Join(config.DataDir, "verifiers.json"),
		skillOwnership:     make(map[string]*types.SkillOwnershipRecord),
		skillOwnershipPath: filepath.Join(config.DataDir, "skill_ownership.json"),
		config:             config,
		logger:             logger,
		ctx:                ctx,
		cancel:             cancel,
	}

	oracle.paymentProcessor = payment.NewProcessor(payment.Config{
		Enabled:                   config.PaymentProcessorEnabled,
		StripeSecretKey:           config.StripeSecretKey,
		StripeWebhookSecret:       config.StripeWebhookSecret,
		StripeThinWebhookSecret:   config.StripeThinWebhookSecret,
		CoinbaseAPIKey:            config.CoinbaseAPIKey,
		CoinbaseWebhookSecret:     config.CoinbaseWebhookSecret,
		TokenDecimals:             config.PaymentTokenDecimals,
		USDPerToken:               config.PaymentUSDPerToken,
		ETHPerToken:               config.PaymentETHPerToken,
		StripeProfessionalPriceID: config.StripeProfessionalPriceID,
		StripeEnterprisePriceID:   config.StripeEnterprisePriceID,
		StripeInvestorPriceID:     config.StripeInvestorPriceID,
		DefaultSuccessURL:         config.PaymentSuccessURL,
		DefaultCancelURL:          config.PaymentCancelURL,
		OnboardingCallbackURL:     config.PaymentOnboardingCallbackURL,
	}, oracle, logger)

	if err := os.MkdirAll(config.DataDir, 0755); err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create oracle data directory: %w", err)
	}
	if err := oracle.loadRollups(); err != nil {
		cancel()
		return nil, fmt.Errorf("failed to load persisted oracle rollups: %w", err)
	}
	if err := oracle.loadSkillOwnership(); err != nil {
		cancel()
		return nil, fmt.Errorf("failed to load persisted skill ownership records: %w", err)
	}

	cs, err := LoadCheckpointState(config.DataDir)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to load checkpoint state: %w", err)
	}
	oracle.checkpoint = cs
	// checkpoint_store.go's mmr_leaf_log.json (loaded into cs.mmr above) is the
	// single persisted copy of the audit MMR — the consensus app's AppHash is
	// just a view over this same object from here on (see SetAuditMMR and
	// appendAuditLeafLocked, the only place new leaves are ever added). There
	// used to be a second, independently-persisted copy inside ABCIApplication
	// (audit_mmr_leaf_log.json) that this constructor cross-checked cs.mmr
	// against, refusing to start on any mismatch. That "safety" check was
	// itself the bug: the two files were written by different call sites
	// (Commit vs. the checkpoint admission path) and routinely fell out of
	// sync on an unclean shutdown, permanently bricking startup with
	// "checkpoint MMR and persisted AppHash MMR diverged; refusing unsafe
	// recovery" even when cs.mmr itself was perfectly intact. With only one
	// persisted copy left, there is nothing left for it to diverge from.
	logger.Info("Installing persisted checkpoint MMR as the audit MMR",
		zap.Uint64("leaves", cs.mmr.Size()),
		zap.String("root", fmt.Sprintf("%x", cs.mmr.BagRoot())),
	)
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
			// Admit the finality leaf through the same path as HTTP.
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

	// Bring up the durable state ledger last: it needs the token's configured
	// owner and the resolved data directory, and it replaces the token's fresh
	// genesis with replayed state when a log already exists.
	if err := oracle.initLedger(); err != nil {
		cancel()
		return nil, fmt.Errorf("state ledger: %w", err)
	}
	if err := oracle.initReplicationSource(); err != nil {
		cancel()
		return nil, fmt.Errorf("replication source: %w", err)
	}
	if err := oracle.initFollower(); err != nil {
		// A bootnode must still start its read-only Oracle when its mirror cannot
		// be opened. It reports the fault and never falls back to active mode.
		oracle.ledgerDegraded = true
		logger.Warn("standby replication mirror unavailable", zap.Error(err))
	}
	if oracle.Mode() == ModeActive && oracle.replication.signer != nil {
		// A fresh node has no head yet; produce one so a standby has an anchor.
		if _, err := oracle.replication.ProduceHead(); err != nil {
			oracle.logger.Warn("initial replication head not produced", zap.Error(err))
		}
		go oracle.replication.run()
	}

	logger.Info("Oracle initialized",
		zap.String("chain_id", config.ChainID),
		zap.String("network_id", config.NetworkID),
		zap.Int("state_log_size", oracle.ledger.Len()),
		zap.String("state_root", oracle.stateRoot),
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
	// Legacy rollup bridge: project the rollup into the checkpoint MMR so it is
	// covered by the same AppHash that anchors signed checkpoints.
	if _, err := o.ProjectRollup(record); err != nil {
		o.logger.Warn("failed to project rollup into checkpoint MMR",
			zap.String("rollup_id", record.ID), zap.Error(err))
	}
	return nil
}

// commitAuditMMR bags the consensus audit MMR into the Oracle AppHash and
// persists the leaf log so the AppHash survives an Oracle restart
// (see merkle-math.md). The Oracle runs in non-validator mode, so the
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

func (o *Oracle) loadSkillOwnership() error {
	o.skillOwnershipMu.Lock()
	defer o.skillOwnershipMu.Unlock()

	data, err := os.ReadFile(o.skillOwnershipPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to read skill ownership state: %w", err)
	}

	var records []*types.SkillOwnershipRecord
	if err := json.Unmarshal(data, &records); err != nil {
		return fmt.Errorf("failed to decode skill ownership state: %w", err)
	}

	o.skillOwnership = make(map[string]*types.SkillOwnershipRecord, len(records))
	for _, record := range records {
		if record == nil {
			continue
		}
		o.skillOwnership[record.SkillID] = record
	}

	return nil
}

func (o *Oracle) persistSkillOwnershipLocked() error {
	if o.skillOwnershipPath == "" {
		return nil
	}

	records := make([]*types.SkillOwnershipRecord, 0, len(o.skillOwnership))
	for _, record := range o.skillOwnership {
		records = append(records, record)
	}

	payload, err := json.MarshalIndent(records, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to encode skill ownership state: %w", err)
	}

	tempPath := o.skillOwnershipPath + ".tmp"
	if err := os.WriteFile(tempPath, payload, 0644); err != nil {
		return fmt.Errorf("failed to write skill ownership temp file: %w", err)
	}

	if err := os.Rename(tempPath, o.skillOwnershipPath); err != nil {
		return fmt.Errorf("failed to move skill ownership state into place: %w", err)
	}

	return nil
}

// RegisterSkillOwnership persists rights, overwriting any existing
// registration for the same SkillID (later registrations win — the caller,
// KNIRVGRAPH's DRQ pipeline, is expected to register once per canonical
// mint, but this keeps the operation idempotent under retries).
func (o *Oracle) RegisterSkillOwnership(rights *types.SkillOwnershipRecord) error {
	o.skillOwnershipMu.Lock()
	defer o.skillOwnershipMu.Unlock()

	o.skillOwnership[rights.SkillID] = rights
	if err := o.persistSkillOwnershipLocked(); err != nil {
		return err
	}
	return nil
}

// GetSkillOwnership retrieves a previously registered ownership record.
func (o *Oracle) GetSkillOwnership(skillID string) (*types.SkillOwnershipRecord, bool) {
	o.skillOwnershipMu.RLock()
	defer o.skillOwnershipMu.RUnlock()
	record, ok := o.skillOwnership[skillID]
	return record, ok
}

// Start starts all oracle services
func (o *Oracle) Start() error {
	o.logger.Info("Starting Oracle services", zap.String("mode", o.Mode()))

	if o.Mode() == ModeStandby {
		// A standby is a failover ledger: it holds the verified mirror and
		// answers read/proof routes. It must not start consensus (no votes), the
		// economics engine (no fee settlement), IBC, P2P publishing or the
		// payment processor (no disbursement). Those are gated by mode here,
		// and the HTTP guard refuses their routes besides.
		o.logger.Info("Standby mode: consensus, economics, IBC, P2P and payment are not started")
		o.startFollower()
		return nil
	}

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

	// Server-side D14 bond completion + refund sweep. Root nodes only: a root
	// completes reserved authority bonds (root-owner-signed Release) and refunds
	// abandoned holds. Restart-safe (initial pass) and idempotent.
	if o.Mode() == ModeActive {
		o.startBondSupervisor(o.ctx)
	}

	return nil
}

// Stop stops all oracle services
func (o *Oracle) Stop() error {
	o.logger.Info("Stopping Oracle services", zap.String("mode", o.Mode()))

	if o.Mode() == ModeStandby {
		if o.ledger != nil {
			o.ledger.Close()
		}
		o.cancel()
		return nil
	}

	// Stop in reverse order
	o.p2pManager.Stop()

	if o.config.IBCEnabled {
		o.ibcHandler.Stop()
	}

	o.consensusEngine.Stop()
	o.economicsEngine.Stop()

	if o.ledger != nil {
		o.ledger.Close()
	}

	o.cancel()

	o.logger.Info("Oracle services stopped")

	return nil
}

// SetReplicaSources injects authenticated transports for standby replication.
// The launcher owns their credentials; Oracle deliberately never accepts a
// signing key or password through this API.
func (o *Oracle) SetReplicaSources(sources []replica.Source) {
	o.followersMu.Lock()
	defer o.followersMu.Unlock()
	o.followers = append([]replica.Source(nil), sources...)
}

// ReplicaStatus returns an honest zero-value status when this node is not a
// standby or has not received a configured trust anchor yet.
func (o *Oracle) ReplicaStatus() replica.Status {
	if o.follower == nil {
		return replica.Status{Paused: o.Mode() == ModeStandby, UnverifiedClasses: []string{"R1", "R2", "R3", "R4", "R5", "R6"}}
	}
	return o.follower.Status()
}

// ClearReplicaDivergence acknowledges a retained, verified standby head after
// an operator has followed the fork-recovery runbook. It is intentionally not
// a generic replica reset: the follower preserves its durable mirror and
// remains paused until a subsequent authenticated sync proves an extension.
func (o *Oracle) ClearReplicaDivergence(expectedHeadHash string) error {
	if o.Mode() != ModeStandby || o.follower == nil {
		return fmt.Errorf("replica divergence recovery is only available on a configured standby")
	}
	return o.follower.ClearDivergence(expectedHeadHash)
}

func (o *Oracle) initFollower() error {
	if o.Mode() != ModeStandby {
		return nil
	}
	if o.config.ReplicationTrustAnchor == "" {
		return fmt.Errorf("replication trust anchor is not configured")
	}
	follower, err := replica.Open(filepath.Join(o.config.DataDir, "replica_state.json"), o.config.ReplicationTrustAnchor)
	if err != nil {
		return err
	}
	o.follower = follower
	follower.SetOnApplied(o.applyReplicaSnapshot)
	if snapshot := follower.Snapshot(); snapshot.Head.Head.Seq != 0 {
		if err := o.applyReplicaSnapshot(snapshot); err != nil {
			return fmt.Errorf("restore persisted replica: %w", err)
		}
	}
	if len(o.config.ReplicationSources) == 0 {
		follower.SetPaused(true)
		return nil
	}
	// Until Task 8.8 supplies the bootnode-master through secrets.Provider,
	// existing file-backed deployments pass their already-loaded boot identity
	// through the RemoteSigner interface. No HTTP source ever receives the raw
	// key, and Bao can replace this construction without changing the follower.
	if o.config.OwnerPrivateKey == "" {
		follower.SetPaused(true)
		return fmt.Errorf("replication sources configured but standby signer is unavailable")
	}
	key, err := hex.DecodeString(strip0x(o.config.OwnerPrivateKey))
	if err != nil {
		return fmt.Errorf("replication signer key: %w", err)
	}
	signer, err := knirvsigning.NewLocalSigner(key)
	if err != nil {
		return fmt.Errorf("replication signer: %w", err)
	}
	sources := make([]replica.Source, 0, len(o.config.ReplicationSources))
	for _, baseURL := range o.config.ReplicationSources {
		source, err := replica.NewHTTPSource(baseURL, o.config.ChainID, signer, nil)
		if err != nil {
			return err
		}
		sources = append(sources, source)
	}
	o.SetReplicaSources(sources)
	return nil
}

// applyReplicaSnapshot is the only bridge from the verified mirror to Oracle
// read models. It never appends a local ledger event or produces a new head:
// a standby must expose exactly the state signed by its source, not become a
// competing writer.
func (o *Oracle) applyReplicaSnapshot(snapshot replica.Snapshot) error {
	if snapshot.State == nil {
		return fmt.Errorf("replica state is nil")
	}
	leaves := make([]mmr.Hash, 0, len(snapshot.Leaves))
	for _, text := range snapshot.Leaves {
		bytes, err := hex.DecodeString(text)
		if err != nil || len(bytes) != len(mmr.Hash{}) {
			return fmt.Errorf("invalid replica MMR leaf %q", text)
		}
		var leaf mmr.Hash
		copy(leaf[:], bytes)
		leaves = append(leaves, leaf)
	}
	if root := mmr.FromLeaves(leaves).BagRoot(); hex.EncodeToString(root[:]) != snapshot.Head.Head.MMRRoot {
		return fmt.Errorf("replica MMR root mismatch")
	}

	// Match the live mutation lock order: ledger/read models first, then the
	// audit MMR. Standby routes are read-only, so no local mutation can race a
	// successful follower apply.
	o.ledgerMu.Lock()
	defer o.ledgerMu.Unlock()
	if err := o.hydrateLedgerState(snapshot.State); err != nil {
		return err
	}
	root, err := ledger.StateRoot(snapshot.State)
	if err != nil {
		return err
	}
	if root != snapshot.Head.Head.StateRoot {
		return fmt.Errorf("replica state root mismatch")
	}
	o.state = snapshot.State
	o.stateRoot = root

	o.checkpoint.mu.Lock()
	defer o.checkpoint.mu.Unlock()
	if err := o.checkpoint.registry.Replace(snapshot.Registry.Chains); err != nil {
		return fmt.Errorf("replace replicated registry: %w", err)
	}
	records := make(map[string]*types.CheckpointRecord, len(snapshot.Checkpoints))
	for _, record := range snapshot.Checkpoints {
		if record == nil {
			return fmt.Errorf("nil replicated checkpoint record")
		}
		records[recordKey(record.Checkpoint.ChainID, record.Checkpoint.StartHeight)] = record
	}
	o.checkpoint.records = records
	o.checkpoint.leafLog = leaves
	o.checkpoint.mmr = mmr.FromLeaves(leaves)
	if err := o.persistCheckpointLocked(); err != nil {
		return err
	}

	o.skillOwnershipMu.Lock()
	o.skillOwnership = snapshot.Aux.SkillOwnership
	if o.skillOwnership == nil {
		o.skillOwnership = make(map[string]*types.SkillOwnershipRecord)
	}
	if err := o.persistSkillOwnershipLocked(); err != nil {
		o.skillOwnershipMu.Unlock()
		return err
	}
	o.skillOwnershipMu.Unlock()

	o.rollupsMu.Lock()
	o.rollups = snapshot.Aux.Rollups
	if o.rollups == nil {
		o.rollups = make(map[string]*types.RollupRecord)
	}
	err = o.persistRollupsLocked()
	o.rollupsMu.Unlock()
	return err
}

func (o *Oracle) startFollower() {
	if o.follower == nil {
		return
	}
	interval := o.config.ReplicationInterval
	if interval <= 0 {
		interval = 30 * time.Second
	}
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			o.followersMu.RLock()
			sources := append([]replica.Source(nil), o.followers...)
			o.followersMu.RUnlock()
			if err := o.follower.SyncSources(o.ctx, sources); err != nil {
				o.logger.Warn("standby replication pull failed", zap.Error(err))
			}
			select {
			case <-o.ctx.Done():
				return
			case <-ticker.C:
			}
		}
	}()
}

// GetNRNToken returns the NRN token instance
func (o *Oracle) GetNRNToken() *token.NRN {
	return o.nrnToken
}

func (o *Oracle) ChainID() string { return o.config.ChainID }

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

// StateRoot is the hash of the replayed ledger state; it is what a replication
// head commits to.
func (o *Oracle) StateRoot() string { return o.stateRoot }

// Mode is the activation mode this Oracle is running in. Empty configuration
// means active, which is the historical behaviour.
func (o *Oracle) Mode() string {
	if o == nil || o.config == nil {
		return ModeActive
	}
	if o.config.Mode == "" {
		return ModeActive
	}
	return o.config.Mode
}

// initLedger opens the durable log, writes genesis once, replays it and
// hydrates the token read model. The ledger wins over configuration (D-R9): a
// disagreement is reported and the node runs Degraded rather than rewriting
// balances from config.
func (o *Oracle) initLedger() error {
	logPath := filepath.Join(o.config.DataDir, "state_log.json")
	lg, err := ledger.Open(logPath)
	if err != nil {
		return err
	}

	if lg.Len() == 0 {
		genesis := ledger.Genesis{
			ChainID:       o.config.ChainID,
			Name:          o.config.TokenName,
			Symbol:        o.config.TokenSymbol,
			Owner:         o.nrnToken.Owner().String(),
			InitialSupply: o.config.InitialSupply.String(),
			MaxSupply:     o.config.MaxSupply.String(),
		}
		if _, err := lg.Append(ledger.KindGenesis, genesis); err != nil {
			lg.Close()
			return err
		}
	}

	state, err := ledger.Replay(lg.Records())
	if err != nil {
		lg.Close()
		return err
	}

	owner := o.nrnToken.Owner().String()
	mismatch := state.ChainID != o.config.ChainID ||
		state.Owner != owner ||
		state.Name != o.config.TokenName ||
		state.Symbol != o.config.TokenSymbol ||
		state.InitialSupply != o.config.InitialSupply.String() ||
		state.MaxSupply != o.config.MaxSupply.String()
	if mismatch {
		o.ledgerDegraded = true
		o.logger.Warn("Ledger genesis disagrees with configuration; using the ledger",
			zap.String("ledger_chain_id", state.ChainID), zap.String("config_chain_id", o.config.ChainID),
			zap.String("ledger_owner", state.Owner), zap.String("config_owner", owner))
	}

	if err := o.hydrateLedgerState(state); err != nil {
		lg.Close()
		return err
	}

	root, err := ledger.StateRoot(state)
	if err != nil {
		lg.Close()
		return err
	}

	o.ledger = lg
	o.state = state
	o.stateRoot = root
	if missing := o.uncommittedLedgerRecords(lg.Records()); missing > 0 {
		o.ledgerDegraded = true
		o.logger.Warn("Ledger records have no audit leaf; the audit MMR is behind the state log",
			zap.Int("missing_leaves", missing))
	}

	o.nrnToken.SetRecorder(o.recordMutation)
	o.nrnToken.SetRecorderBatch(o.recordMutationBatch)
	o.governanceSystem.SetRecorder(o.recordMutation)
	// Genesis is deliberately not committed to the audit MMR here: a fresh
	// Oracle must recover to an empty MMR until a checkpoint or a live mutation
	// arrives (oracle_startup_test.go asserts both properties).
	return nil
}

// hydrateLedgerState refreshes the token and governance read models from the
// one deterministic ledger state used by both local replay and a verified
// standby mirror.
func (o *Oracle) hydrateLedgerState(state *ledger.State) error {
	if state == nil {
		return fmt.Errorf("ledger state is nil")
	}
	total, ok := new(big.Int).SetString(state.TotalSupply, 10)
	if !ok {
		return fmt.Errorf("total supply %q is not a decimal integer", state.TotalSupply)
	}
	balances := make(map[string]*big.Int, len(state.Balances))
	for address, raw := range state.Balances {
		amount, ok := new(big.Int).SetString(raw, 10)
		if !ok {
			return fmt.Errorf("balance for %s is not a decimal integer", address)
		}
		balances[address] = amount
	}
	if err := o.nrnToken.Hydrate(state.ChainID, state.Owner, total, balances, state.Nonces, state.Registered, state.Bonds); err != nil {
		return err
	}
	return o.governanceSystem.Hydrate(state.Governance)
}

// uncommittedLedgerRecords counts log records whose audit leaf is absent from
// the persisted MMR. It detects a crash between the log append and the leaf
// append. Genesis is excluded: it is written without a leaf (see initLedger).
//
// Repair is the follower's job (8.5): rebuilding the MMR needs the checkpoint
// leaves, which this check deliberately does not attempt to reconstruct.
func (o *Oracle) uncommittedLedgerRecords(records []ledger.Record) int {
	if o.checkpoint.mmr == nil || len(records) <= 1 {
		return 0
	}
	o.checkpoint.mu.RLock()
	present := make(map[mmr.Hash]bool, len(o.checkpoint.leafLog))
	for _, leaf := range o.checkpoint.leafLog {
		present[leaf] = true
	}
	o.checkpoint.mu.RUnlock()

	missing := 0
	for _, record := range records[1:] { // skip genesis
		leaf := mmr.LeafHash([]byte("state:" + record.Hash))
		if !present[leaf] {
			missing++
		}
	}
	return missing
}

// recordMutation appends one semantic event, applies it to the state mirror and
// commits its hash to the audit MMR. It runs while the token or governance lock
// is held; it never calls back into either, so it cannot deadlock.
func (o *Oracle) recordMutation(kind string, payload any) error {
	o.ledgerMu.Lock()
	defer o.ledgerMu.Unlock()
	if o.ledger == nil {
		return nil
	}
	record, err := o.ledger.Append(kind, payload)
	if err != nil {
		return err
	}
	if err := ledger.Apply(o.state, record); err != nil {
		return fmt.Errorf("ledger apply seq %d: %w", record.Seq, err)
	}
	if root, err := ledger.StateRoot(o.state); err == nil {
		o.stateRoot = root
	}
	return o.commitLedgerLeaf(record)
}

// recordMutationBatch commits several records as one validated ledger batch: they
// are appended and fsync'd together by the log, applied to the read model, and
// each committed to the audit MMR. The whole batch succeeds or fails together, so
// a recorder failure or crash between the records cannot leave a half-applied
// escrow (e.g. a token transfer with no corresponding hold status).
func (o *Oracle) recordMutationBatch(entries []ledger.BatchEntry) error {
	o.ledgerMu.Lock()
	defer o.ledgerMu.Unlock()
	if o.ledger == nil {
		return nil
	}
	batch := make([]ledger.BatchEntry, len(entries))
	for i, e := range entries {
		batch[i] = ledger.BatchEntry{Kind: e.Kind, Payload: e.Payload}
	}
	records, err := o.ledger.AppendBatch(batch)
	if err != nil {
		return err
	}
	for _, record := range records {
		if err := ledger.Apply(o.state, record); err != nil {
			return fmt.Errorf("ledger apply seq %d: %w", record.Seq, err)
		}
		if root, err := ledger.StateRoot(o.state); err == nil {
			o.stateRoot = root
		}
		if err := o.commitLedgerLeaf(record); err != nil {
			return err
		}
	}
	return nil
}

// commitLedgerLeaf makes the record part of the Oracle chain's audit MMR, so a
// checkpoint or block commits to the state that produced it. A standby mirror
// is built from replication (8.5) and is not a source of finality, so it does
// not contribute leaves.
func (o *Oracle) commitLedgerLeaf(record ledger.Record) error {
	if o.Mode() != ModeActive || o.checkpoint.mmr == nil {
		return nil
	}
	o.checkpoint.mu.Lock()
	defer o.checkpoint.mu.Unlock()
	o.appendAuditLeafLocked([]byte("state:" + record.Hash))
	return o.persistCheckpointLocked()
}

// GetStatus returns the current status of the oracle.
func (o *Oracle) GetStatus() map[string]interface{} {
	logSize := 0
	if o.ledger != nil {
		logSize = o.ledger.Len()
	}
	replicaStatus := o.ReplicaStatus()
	if o.Mode() == ModeActive && o.replication != nil {
		if head, ok := o.replication.CurrentHead(); ok {
			replicaStatus.HeadSeq = head.Head.Seq
		}
	}
	return map[string]interface{}{
		"state_log_size":  logSize,
		"state_root":      o.stateRoot,
		"ledger_degraded": o.ledgerDegraded,
		"chain_id":        o.config.ChainID,
		"network_id":      o.config.NetworkID,
		"mode":            o.Mode(),
		// Replication position. A standby follower fills these in;
		// until then an active Oracle is the source of truth and has no lag,
		// and a standby reports zero rather than claiming to be current.
		"head_seq":           replicaStatus.HeadSeq,
		"lag_seq":            replicaStatus.LagSeq,
		"lag_seconds":        replicaStatus.LagSeconds,
		"diverged":           replicaStatus.Diverged,
		"replication_paused": replicaStatus.Paused,
		"unverified_classes": replicaStatus.UnverifiedClasses,
		"token":              o.nrnToken.Info(),
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
	if err := nrnToken.SetChainID(config.ChainID); err != nil {
		return nil, err
	}
	// Initialize the trusted root tx-chain registry id from the owner's public key
	// before the token serves requests. This is an immutable genesis constant
	// (identical to the launcher's derived id); setting it here means production
	// holds bind to it instead of failing closed on an empty value.
	if ownerPub := nrnToken.OwnerPublicKeyBytes(); len(ownerPub) > 0 {
		nrnToken.SetRootRegistryID(token.DeriveRootRegistryID(ownerPub))
	}

	logger.Info("NRN token initialized",
		zap.String("name", config.TokenName),
		zap.String("symbol", config.TokenSymbol),
		zap.String("initial_supply", config.InitialSupply.String()),
		zap.String("max_supply", config.MaxSupply.String()),
	)

	return nrnToken, nil
}
