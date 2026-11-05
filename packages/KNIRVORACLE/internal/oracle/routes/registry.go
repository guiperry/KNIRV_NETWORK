package routes

import (
	"net/http"
	"strings"
)

// routeClass is how much a route can change Oracle state. It exists so that a
// new route cannot be added without deciding, and so the standby allowlist has
// a written justification next to each entry.
type routeClass int

const (
	// classRead changes nothing. Some reads are still not served by a standby
	// (balances are not public and are not replicated yet), which
	// is why standby is a separate flag rather than a synonym for classRead.
	classRead routeClass = iota + 1
	// classReplication serves the replication protocol.
	classReplication
	// classMutating signs, mints, burns, votes, registers or otherwise writes.
	classMutating
)

func (c routeClass) String() string {
	switch c {
	case classRead:
		return "read"
	case classReplication:
		return "replication"
	case classMutating:
		return "mutating"
	}
	return "unclassified"
}

// routeDef is one registered Oracle route plus its classification.
type routeDef struct {
	pattern string
	class   routeClass
	// standby is true for the routes a standby Oracle serves. A
	// standby is a failover ledger: it mirrors state and answers proofs, and
	// refuses everything that could sign or mutate.
	standby bool
	handler http.HandlerFunc
}

// routeTable is the single list RegisterRoutes walks. A route that is not in
// this table cannot be registered, and every entry carries a class, so
// "unclassified route" is unrepresentable rather than merely tested for.
func (r *OracleRoutes) routeTable() []routeDef {
	return []routeDef{
		// Token endpoints.
		{"/oracle/v3/token/info", classRead, false, r.handleTokenInfo},
		{"/oracle/v3/token/balance/", classRead, false, r.handleTokenBalance},
		{"/oracle/v3/token/mint", classMutating, false, r.handleTokenMint},
		{"/oracle/v3/token/transfer", classMutating, false, r.handleTokenTransfer},
		{"/oracle/v3/token/transfer/signed", classMutating, false, r.handleTokenTransferSigned},
		{"/oracle/v3/token/burn", classMutating, false, r.handleTokenBurn},
		{"/oracle/v3/token/burn/signed", classMutating, false, r.handleTokenBurnSigned},
		{"/oracle/v3/account/nonce/", classRead, false, r.handleAccountNonce},

		// Governance endpoints.
		{"/oracle/v3/governance/proposals", classRead, false, r.handleProposals},
		{"/oracle/v3/governance/proposals/", classMutating, false, r.handleProposal},
		{"/oracle/v3/governance/vote", classMutating, false, r.handleVote},
		{"/oracle/v3/governance/validators", classRead, false, r.handleValidators},
		{"/oracle/v3/governance/validators/", classRead, false, r.handleValidator},
		{"/oracle/v3/governance/params", classRead, false, r.handleNetworkParams},

		// Economics endpoints.
		{"/oracle/v3/economics/metrics", classRead, false, r.handleEconomicMetrics},
		{"/oracle/v3/economics/fees", classRead, false, r.handleFees},
		{"/oracle/v3/economics/rewards", classRead, false, r.handleRewards},
		{"/oracle/v3/economics/staking", classRead, false, r.handleStaking},
		{"/oracle/v3/economics/burns", classRead, false, r.handleBurns},

		// Metered consumption settlement.
		{"/oracle/v3/token/fee/internal", classMutating, false, r.handleInternalFeeCollect},

		// Skill economics (internal-service-token gated).
		{"/oracle/v3/skills/burn", classMutating, false, r.handleSkillBurn},
		{"/oracle/v3/skills/bounty", classMutating, false, r.handleSkillBounty},
		{"/oracle/v3/skills/ownership", classMutating, false, r.handleSkillOwnership},
		{"/oracle/v3/skills/ownership/", classMutating, false, r.handleSkillOwnership},
		{"/oracle/v3/skills/verify", classMutating, false, r.handleSkillVerify},

		// Cross-chain endpoints.
		{"/oracle/v3/crosschain/transfer", classMutating, false, r.handleCrossChainTransfer},
		{"/oracle/v3/crosschain/transfer/", classRead, false, r.handleGetTransfer},
		{"/oracle/v3/crosschain/bridges", classRead, false, r.handleBridges},

		// IBC endpoints.
		{"/oracle/v3/ibc/channels", classRead, false, r.handleIBCChannels},
		{"/oracle/v3/ibc/connections", classRead, false, r.handleIBCConnections},
		{"/oracle/v3/ibc/clients", classRead, false, r.handleIBCClients},

		// Consensus endpoints. Status is what a standby reports about its head.
		{"/oracle/v3/consensus/status", classRead, true, r.handleConsensusStatus},
		{"/oracle/v3/consensus/validators", classRead, false, r.handleConsensusValidators},

		// P2P endpoints.
		{"/oracle/v3/p2p/status", classRead, false, r.handleP2PStatus},
		{"/oracle/v3/p2p/peers", classRead, false, r.handleP2PPeers},
		{"/oracle/v3/p2p/pause", classMutating, false, r.handleP2PPause},
		{"/oracle/v3/p2p/resume", classMutating, false, r.handleP2PResume},

		// Rollup settlement endpoints.
		{"/oracle/v3/rollups/submit", classMutating, false, r.handleSubmitRollup},
		{"/oracle/v3/rollups/", classRead, false, r.handleRollup},

		// Checkpoint pipeline: registry, admission, MMR proofs.
		{"/oracle/v3/registry/register", classMutating, false, r.handleRegistryRegister},
		{"/oracle/v3/registry/rotate", classMutating, false, r.handleRegistryRotate},
		{"/oracle/v3/registry/verify-key", classMutating, false, r.handleSetVerificationKey},
		{"/oracle/v3/registry/", classRead, true, r.handleGetRegistration},
		{"/oracle/v3/checkpoints", classMutating, false, r.handleSubmitCheckpoint},
		{"/oracle/v3/checkpoints/", classRead, true, r.handleCheckpointsByChain},
		{"/oracle/v3/mmr/root", classRead, true, r.handleMMRRoot},
		{"/oracle/v3/mmr/proof/", classRead, true, r.handleMMRProof},

		// Finality pipeline.
		{"/oracle/v3/finality", classMutating, false, r.handleSubmitFinality},
		{"/oracle/v3/finality/attest", classMutating, false, r.handleSubmitAttestation},
		{"/oracle/v3/verifiers/register", classMutating, false, r.handleRegisterVerifier},
		{"/oracle/v3/authority/report", classMutating, false, r.handleAuthorityReport},

		// Installation and wallet endpoints.
		{"/oracle/v3/install/wallet", classMutating, false, r.handleInstallWallet},
		{"/oracle/v3/install/dve_uri", classMutating, false, r.handleInstallDVEURI},
		{"/oracle/v3/wallet/register", classMutating, false, r.handleRegisterWallet},
		{"/oracle/v3/wallet/status", classRead, false, r.handleWalletStatus},

		// Legacy wallet-server compatibility endpoints.
		{"/generate_wallet", classMutating, false, r.handleGenerateWallet},
		{"/balance/", classRead, false, r.handleLegacyWalletBalance},
		{"/transactions", classMutating, false, r.handleCreateTransaction},
		{"/send_signed_txn", classMutating, false, r.handleSendSignedTransaction},
		{"/test/faucet", classMutating, false, r.handleTestFaucet},

		// Payment processor: fiat-triggered NRN disbursement and plan checkout.
		{"/oracle/v3/payment/webhooks/stripe", classMutating, false, r.oracle.GetPaymentProcessor().HandleStripeWebhook},
		{"/oracle/v3/payment/checkout/create", classMutating, false, r.oracle.GetPaymentProcessor().HandleCreateCheckoutSession},

		// Replication. Reads and head serving are public; the
		// registry, state and aux snapshots require pull authentication.
		{"/oracle/v3/replication/head", classReplication, true, r.handleReplicationHead},
		{"/oracle/v3/replication/heads", classReplication, true, r.handleReplicationHeads},
		{"/oracle/v3/replication/leaves", classReplication, true, r.handleReplicationLeaves},
		{"/oracle/v3/replication/checkpoints", classReplication, true, r.handleReplicationCheckpoints},
		{"/oracle/v3/replication/registry", classReplication, true, r.handleReplicationRegistry},
		{"/oracle/v3/replication/state", classReplication, true, r.handleReplicationState},
		{"/oracle/v3/replication/aux", classReplication, true, r.handleReplicationAux},
		// This is the sole standby mutation: it is an operator-token-gated
		// acknowledgement of an already retained signed head, not a state
		// replacement. See docs/runbooks/diverged-standby.md.
		{"/oracle/v3/replication/clear-divergence", classReplication, true, r.handleClearReplicaDivergence},

		// Health and status.
		{"/oracle/v3/health", classRead, true, r.handleHealth},
		{"/oracle/v3/status", classRead, true, r.handleStatus},

		// DID resolution: registration writes, resolution reads.
		{"/oracle/v3/did/register", classMutating, false, r.handleDIDRegister},
		{"/oracle/v3/did/", classRead, false, r.handleDID},
	}
}

// standbyAlwaysAllowed are the exact paths a standby serves regardless of the
// table; the table's standby flags are asserted against this list in tests so
// the two cannot drift.
var standbyExactPaths = map[string]bool{
	"/health":                     true,
	"/oracle/v3/health":           true,
	"/oracle/v3/status":           true,
	"/oracle/v3/consensus/status": true,
	"/oracle/v3/mmr/root":         true,
}

// standbyPrefixes are read-only path prefixes a standby serves for GET/HEAD.
// The trailing slash is deliberate: it does not match /oracle/v3/checkpoints
// (the mutating submit route).
var standbyPrefixes = []string{
	"/oracle/v3/mmr/proof/",
	"/oracle/v3/checkpoints/",
	"/oracle/v3/registry/",
	"/oracle/v3/replication/",
}

// StandbyServes reports whether a request may be served by a standby Oracle.
// It is deny-by-default: an unlisted path is refused even if it would be a
// harmless read, so a route added without a decision is blocked.
func StandbyServes(method, path string) bool {
	if method == http.MethodPost && path == "/oracle/v3/replication/clear-divergence" {
		return true
	}
	if method != http.MethodGet && method != http.MethodHead {
		return false
	}
	if standbyExactPaths[path] {
		return true
	}
	for _, prefix := range standbyPrefixes {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}
	return false
}

// StandbyRefusal is the body every refused route returns. It names the reason
// so an operator can tell "this node is a standby" from "this node is broken".
func standbyRefusal(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusServiceUnavailable)
	_, _ = w.Write([]byte(`{"error":"oracle standby"}`))
}
